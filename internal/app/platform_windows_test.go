//go:build windows

package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	for _, path := range []string{dir, filepath.Join(dir, "TOKEN")} {
		cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `[Console]::Out.Write((Get-Acl -LiteralPath $env:FIDELIUS_TEST_PATH).Sddl)`)
		cmd.Env = append(os.Environ(), "FIDELIUS_TEST_PATH="+path)
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("read ACL for %s: %v", path, err)
		}
		sddl := string(output)
		dacl := strings.SplitN(sddl, "S:", 2)[0]
		sid, err := currentUserSID()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(dacl, "D:P") || !strings.Contains(dacl, sid) || strings.Count(dacl, "(") != 1 {
			t.Fatalf("ACL does not protect %s for current user: %s", path, sddl)
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
