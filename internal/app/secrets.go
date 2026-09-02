package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const internalCleanupCommand = "__cleanup"

func validateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("secret names cannot be empty")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("invalid secret name %q", name)
	}
	return nil
}

func writeSecretSession(values map[string]string, autoDelete time.Duration) (string, time.Time, error) {
	root, err := ensureSessionRoot()
	if err != nil {
		return "", time.Time{}, err
	}
	_ = cleanupExpiredSessions()

	deleteAt := time.Now().Add(autoDelete)
	dir, err := os.MkdirTemp(root, fmt.Sprintf("%d-", deleteAt.Unix()))
	if err != nil {
		return "", time.Time{}, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return "", time.Time{}, err
	}

	for name, value := range values {
		if err := validateSecretName(name); err != nil {
			_ = os.RemoveAll(dir)
			return "", time.Time{}, err
		}
		path := filepath.Join(dir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			_ = os.RemoveAll(dir)
			return "", time.Time{}, err
		}
		_, writeErr := io.WriteString(file, value)
		closeErr := file.Close()
		if writeErr != nil {
			_ = os.RemoveAll(dir)
			return "", time.Time{}, writeErr
		}
		if closeErr != nil {
			_ = os.RemoveAll(dir)
			return "", time.Time{}, closeErr
		}
	}
	return dir, deleteAt, nil
}

func scheduleAutoDelete(dir string, deleteAt time.Time) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, internalCleanupCommand, dir, strconv.FormatInt(deleteAt.UnixNano(), 10))
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func runCleanup(args []string) int {
	if len(args) != 2 {
		return 2
	}
	dir := args[0]
	deleteAtNanos, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return 2
	}
	if err := validateSessionDir(dir); err != nil {
		return 2
	}
	deleteAt := time.Unix(0, deleteAtNanos)
	if wait := time.Until(deleteAt); wait > 0 {
		time.Sleep(wait)
	}
	if err := removeSecretSession(dir); err != nil {
		return 1
	}
	return 0
}

func cleanupExpiredSessions() error {
	root, err := sessionRoot()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "-")
		if !ok {
			continue
		}
		deleteAt, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil || deleteAt > now {
			continue
		}
		_ = os.RemoveAll(filepath.Join(root, entry.Name()))
	}
	return nil
}

func removeSecretSession(dir string) error {
	if err := validateSessionDir(dir); err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func ensureSessionRoot() (string, error) {
	root, err := sessionRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return "", err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("temporary root is not a private directory")
	}
	return root, nil
}

func sessionRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("FIDELIUS_TEMP_ROOT")); override != "" {
		return filepath.Clean(override), nil
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("fidelius-%d", os.Getuid())), nil
}

func validateSessionDir(dir string) error {
	root, err := sessionRoot()
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, dirAbs)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return fmt.Errorf("refusing to remove path outside Fidelius temporary root")
	}
	return nil
}
