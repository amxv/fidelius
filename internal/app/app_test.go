package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func runForTest(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func isolateTestStorage(t *testing.T) {
	t.Helper()
	t.Setenv("FIDELIUS_CONFIG_DIR", filepath.Join(t.TempDir(), "config"))
	t.Setenv("FIDELIUS_TEMP_ROOT", filepath.Join(t.TempDir(), "sessions"))
}

func TestRootHelpExplainsOneConcept(t *testing.T) {
	code, out, _ := runForTest(t, "--help")
	if code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	for _, expected := range []string{
		"securely ask humans for secrets",
		"fidelius ask",
		"private temporary directory path",
		"auto-deletes",
		"fidelius timeout",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("help missing %q: %q", expected, out)
		}
	}
	if strings.Contains(strings.ToLower(out), "keychain") {
		t.Fatalf("root help should not teach a storage destination: %q", out)
	}
}

func TestAskRequiresSecretName(t *testing.T) {
	code, _, errOut := runForTest(t, "ask")
	if code != 2 || !strings.Contains(errOut, "at least one secret") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestAskRejectsUnsafeFilename(t *testing.T) {
	code, _, errOut := runForTest(t, "ask", "../TOKEN")
	if code != 2 || !strings.Contains(errOut, "invalid secret name") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestAskCreatesPrivateDirectoryWithoutPrintingValues(t *testing.T) {
	isolateTestStorage(t)
	oldPrompt := launchPrompt
	oldSchedule := scheduleDelete
	launchPrompt = func(req promptRequest) (promptResult, error) {
		if req.Message != "I need these to finish the scrape." {
			t.Fatalf("unexpected message: %q", req.Message)
		}
		if req.AutoDeleteLabel != "5 minutes" {
			t.Fatalf("unexpected auto-delete label: %q", req.AutoDeleteLabel)
		}
		return promptResult{Values: map[string]string{
			"MAPS_KEY":  "super-secret-maps",
			"OTHER_KEY": "another-secret",
		}}, nil
	}
	scheduleDelete = func(string, time.Time) error { return nil }
	t.Cleanup(func() {
		launchPrompt = oldPrompt
		scheduleDelete = oldSchedule
	})

	code, out, errOut := runForTest(t, "ask", "-m", "I need these to finish the scrape.", "MAPS_KEY", "OTHER_KEY")
	if code != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	dir := strings.TrimSpace(out)
	if dir == "" || strings.Contains(out, "super-secret") || strings.Contains(errOut, "super-secret") || strings.Contains(errOut, "another-secret") {
		t.Fatalf("secret leaked or directory missing: stdout=%q stderr=%q", out, errOut)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "MAPS_KEY")); err != nil || string(got) != "super-secret-maps" {
		t.Fatalf("MAPS_KEY=%q err=%v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "OTHER_KEY")); err != nil || string(got) != "another-secret" {
		t.Fatalf("OTHER_KEY=%q err=%v", got, err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode=%v err=%v", infoMode(info), err)
	}
	if info, err := os.Stat(filepath.Join(dir, "MAPS_KEY")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode=%v err=%v", infoMode(info), err)
	}
	if !strings.Contains(errOut, "Received MAPS_KEY") || !strings.Contains(errOut, "Auto-delete in 5m") {
		t.Fatalf("unexpected metadata: %q", errOut)
	}
}

func TestAskCancellationLeavesStdoutEmpty(t *testing.T) {
	isolateTestStorage(t)
	oldPrompt := launchPrompt
	launchPrompt = func(promptRequest) (promptResult, error) {
		return promptResult{Cancelled: true}, nil
	}
	t.Cleanup(func() { launchPrompt = oldPrompt })

	code, out, errOut := runForTest(t, "ask", "TOKEN")
	if code != 2 || out != "" || !strings.Contains(errOut, "Cancelled") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
	}
}

func TestTimeoutDefaultsToFiveMinutesAndCanBeChanged(t *testing.T) {
	isolateTestStorage(t)

	code, out, errOut := runForTest(t, "timeout")
	if code != 0 || errOut != "" || strings.TrimSpace(out) != "Auto-delete timeout: 5m" {
		t.Fatalf("default code=%d stdout=%q stderr=%q", code, out, errOut)
	}

	code, out, errOut = runForTest(t, "timeout", "45s")
	if code != 0 || errOut != "" || !strings.Contains(out, "45s") {
		t.Fatalf("set code=%d stdout=%q stderr=%q", code, out, errOut)
	}

	code, out, _ = runForTest(t, "timeout")
	if code != 0 || strings.TrimSpace(out) != "Auto-delete timeout: 45s" {
		t.Fatalf("persisted code=%d stdout=%q", code, out)
	}
}

func TestTimeoutRejectsNonPositiveDuration(t *testing.T) {
	isolateTestStorage(t)
	code, _, errOut := runForTest(t, "timeout", "0s")
	if code != 2 || !strings.Contains(errOut, "greater than zero") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestCleanupRemovesOnlyFideliusSession(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	t.Setenv("FIDELIUS_TEMP_ROOT", root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "123-test")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "TOKEN"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	code := runCleanup([]string{dir, strconv.FormatInt(time.Now().Add(-time.Second).UnixNano(), 10)})
	if code != 0 {
		t.Fatalf("cleanup exit code %d", code)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("session still exists: %v", err)
	}
}

func infoMode(info os.FileInfo) os.FileMode {
	if info == nil {
		return 0
	}
	return info.Mode().Perm()
}
