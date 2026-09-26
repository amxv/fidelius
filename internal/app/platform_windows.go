//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	advapi32        = syscall.NewLazyDLL("advapi32.dll")
	convertSDDL     = advapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	setFileSecurity = advapi32.NewProc("SetFileSecurityW")
)

const daclSecurityInformation = 0x00000004
const protectedDACL = 0x80000000
const detachedProcess = 0x00000008

func currentUserSID() (string, error) {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	return user.User.Sid.String()
}

func defaultSessionRoot() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "fidelius", "sessions"), nil
}

func securePrivateDirectory(path string) error { return setUserOnlyACL(path, true) }
func securePrivateFile(path string) error      { return setUserOnlyACL(path, false) }

func setUserOnlyACL(path string, directory bool) error {
	sid, err := currentUserSID()
	if err != nil {
		return err
	}
	flags := ""
	if directory {
		flags = "OICI"
	}
	sddl := fmt.Sprintf("D:P(A;%s;FA;;;%s)", flags, sid)
	sddlPtr, err := syscall.UTF16PtrFromString(sddl)
	if err != nil {
		return err
	}
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	var descriptor uintptr
	r1, _, callErr := convertSDDL.Call(uintptr(unsafe.Pointer(sddlPtr)), 1, uintptr(unsafe.Pointer(&descriptor)), 0)
	if r1 == 0 {
		return fmt.Errorf("create private Windows ACL: %w", callErr)
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	r1, _, callErr = setFileSecurity.Call(uintptr(unsafe.Pointer(pathPtr)), daclSecurityInformation|protectedDACL, descriptor)
	if r1 == 0 {
		return fmt.Errorf("apply private Windows ACL: %w", callErr)
	}
	return nil
}

func configureCleanupProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess}
}

func validatePlatformSecretName(name string) error {
	if strings.ContainsAny(name, `:<>"|?*`) || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("invalid Windows secret name %q", name)
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("invalid Windows secret name %q", name)
	}
	return nil
}

func platformSecretNameKey(name string) string { return strings.ToLower(name) }
