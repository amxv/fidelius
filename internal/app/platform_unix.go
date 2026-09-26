//go:build darwin || linux

package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func defaultSessionRoot() (string, error) {
	return filepath.Join(os.TempDir(), fmt.Sprintf("fidelius-%d", os.Getuid())), nil
}

func securePrivateDirectory(path string) error { return os.Chmod(path, 0o700) }
func securePrivateFile(path string) error      { return os.Chmod(path, 0o600) }
func configureCleanupProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
func validatePlatformSecretName(string) error  { return nil }
func platformSecretNameKey(name string) string { return name }
