//go:build !darwin && !linux && !windows

package app

import (
	"os"
	"os/exec"
	"path/filepath"
)

func defaultSessionRoot() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "fidelius", "sessions"), nil
}
func securePrivateDirectory(path string) error { return os.Chmod(path, 0o700) }
func securePrivateFile(path string) error      { return os.Chmod(path, 0o600) }
func configureCleanupProcess(*exec.Cmd)        {}
func validatePlatformSecretName(string) error  { return nil }
func platformSecretNameKey(name string) string { return name }
