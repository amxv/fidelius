//go:build windows

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	getFileSecurity            = syscall.NewLazyDLL("advapi32.dll").NewProc("GetFileSecurityW")
	securityDescriptorToString = syscall.NewLazyDLL("advapi32.dll").NewProc("ConvertSecurityDescriptorToStringSecurityDescriptorW")
)

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == internalCleanupCommand {
		os.Exit(runCleanup(os.Args[2:]))
	}
	os.Exit(m.Run())
}

func TestWindowsSecretNames(t *testing.T) {
	for _, name := range []string{"CON", "nul.txt", "LPT1.key", "bad:name", "trailing.", "trailing "} {
		if err := validateSecretName(name); err == nil {
			t.Errorf("accepted unsafe Windows filename %q", name)
		}
	}
	if _, err := parseAsk([]string{"TOKEN", "token"}); err == nil {
		t.Fatal("accepted names that collide on Windows")
	}
}

func TestWindowsSessionACLAndScheduledCleanup(t *testing.T) {
	t.Setenv("FIDELIUS_TEMP_ROOT", filepath.Join(t.TempDir(), "sessions"))
	dir, deleteAt, err := writeSecretSession(map[string]string{"TOKEN": "private-value"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sid, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	for i, path := range []string{dir, filepath.Join(dir, "TOKEN")} {
		sddl, err := readFileDACL(path)
		if err != nil {
			t.Fatalf("read DACL for %s: %v", path, err)
		}
		expected, err := canonicalUserDACL(sid, i == 0)
		if err != nil {
			t.Fatal(err)
		}
		if sddl != expected {
			t.Fatalf("ACL does not protect %s for current user: got %s, want %s", path, sddl, expected)
		}
	}
	if err := scheduleAutoDelete(dir, deleteAt); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("scheduled Windows cleanup did not remove the session")
}

func readFileDACL(path string) (string, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	var needed uint32
	_, _, callErr := getFileSecurity.Call(uintptr(unsafe.Pointer(pathPtr)), daclSecurityInformation, 0, 0, uintptr(unsafe.Pointer(&needed)))
	if needed == 0 {
		return "", callErr
	}
	descriptor := make([]byte, needed)
	r1, _, callErr := getFileSecurity.Call(uintptr(unsafe.Pointer(pathPtr)), daclSecurityInformation, uintptr(unsafe.Pointer(&descriptor[0])), uintptr(needed), uintptr(unsafe.Pointer(&needed)))
	if r1 == 0 {
		return "", callErr
	}
	sddl, err := descriptorDACLString(uintptr(unsafe.Pointer(&descriptor[0])))
	runtime.KeepAlive(descriptor)
	return sddl, err
}

func canonicalUserDACL(sid string, directory bool) (string, error) {
	flags := ""
	if directory {
		flags = "OICI"
	}
	// Windows may serialize the invoking user's SID as an SDDL alias, such as LA.
	sddl := fmt.Sprintf("D:P(A;%s;FA;;;%s)", flags, sid)
	sddlPtr, err := syscall.UTF16PtrFromString(sddl)
	if err != nil {
		return "", err
	}
	var descriptor uintptr
	r1, _, callErr := convertSDDL.Call(uintptr(unsafe.Pointer(sddlPtr)), 1, uintptr(unsafe.Pointer(&descriptor)), 0)
	if r1 == 0 {
		return "", callErr
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))
	return descriptorDACLString(descriptor)
}

func descriptorDACLString(descriptor uintptr) (string, error) {
	var sddlPtr *uint16
	var sddlLen uint32
	r1, _, callErr := securityDescriptorToString.Call(descriptor, 1, daclSecurityInformation, uintptr(unsafe.Pointer(&sddlPtr)), uintptr(unsafe.Pointer(&sddlLen)))
	if r1 == 0 {
		return "", callErr
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(sddlPtr)))
	return syscall.UTF16ToString(unsafe.Slice(sddlPtr, int(sddlLen))), nil
}
