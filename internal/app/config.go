package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultAutoDeleteTimeout = 5 * time.Minute

type persistedConfig struct {
	AutoDeleteTimeout string `json:"auto_delete_timeout"`
}

func loadAutoDeleteTimeout() (time.Duration, error) {
	path, err := configPath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultAutoDeleteTimeout, nil
	}
	if err != nil {
		return 0, err
	}
	var cfg persistedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, err
	}
	if strings.TrimSpace(cfg.AutoDeleteTimeout) == "" {
		return defaultAutoDeleteTimeout, nil
	}
	return parseAutoDeleteDuration(cfg.AutoDeleteTimeout)
}

func saveAutoDeleteTimeout(d time.Duration) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(persistedConfig{AutoDeleteTimeout: formatDurationShort(d)}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func configPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("FIDELIUS_CONFIG_DIR")); override != "" {
		return filepath.Join(override, "config.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fidelius", "config.json"), nil
}

func parseAutoDeleteDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("auto-delete timeout cannot be empty")
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q (use values like 30s, 10m, or 2h)", value)
	}
	if d <= 0 {
		return 0, fmt.Errorf("auto-delete timeout must be greater than zero")
	}
	return d, nil
}

func formatDurationShort(d time.Duration) string {
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int64(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int64(d/time.Minute))
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%ds", int64(d/time.Second))
	}
	return d.String()
}

func formatDurationHuman(d time.Duration) string {
	if d%time.Hour == 0 {
		n := int64(d / time.Hour)
		if n == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", n)
	}
	if d%time.Minute == 0 {
		n := int64(d / time.Minute)
		if n == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", n)
	}
	if d%time.Second == 0 {
		n := int64(d / time.Second)
		if n == 1 {
			return "1 second"
		}
		return fmt.Sprintf("%d seconds", n)
	}
	return formatDurationShort(d)
}
