package app

import (
	"bytes"
	"strings"
	"testing"
)

func runForTest(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestHelpExplainsPurpose(t *testing.T) {
	code, out, _ := runForTest(t, "--help")
	if code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if !strings.Contains(out, "allows agents to ask humans to enter secrets") {
		t.Fatalf("unexpected help output: %q", out)
	}
	if !strings.Contains(out, "--message") {
		t.Fatalf("help should explain the optional message: %q", out)
	}
	if !strings.Contains(out, "security find-generic-password") {
		t.Fatalf("help should explain Keychain retrieval: %q", out)
	}
}

func TestMissingService(t *testing.T) {
	code, _, errOut := runForTest(t, "OPENAI_API_KEY")
	if code != 2 || !strings.Contains(errOut, "missing Keychain service") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestMissingAccount(t *testing.T) {
	code, _, errOut := runForTest(t, "-s", "my-app")
	if code != 2 || !strings.Contains(errOut, "at least one secret") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestDuplicateAccount(t *testing.T) {
	code, _, errOut := runForTest(t, "-s", "my-app", "TOKEN", "TOKEN")
	if code != 2 || !strings.Contains(errOut, "duplicate") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestMessageIsPassedToPrompt(t *testing.T) {
	old := launchPrompt
	launchPrompt = func(req request) (promptResult, error) {
		if req.message != "I need this to finish the scrape." {
			t.Fatalf("unexpected message: %q", req.message)
		}
		return promptResult{Saved: []savedKey{{Account: "MAPS_KEY", Length: 8}}}, nil
	}
	t.Cleanup(func() { launchPrompt = old })

	code, _, errOut := runForTest(t, "-s", "scraper", "-m", "I need this to finish the scrape.", "MAPS_KEY")
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestSuccessfulPromptReportsOnlyMetadata(t *testing.T) {
	old := launchPrompt
	launchPrompt = func(req request) (promptResult, error) {
		if req.service != "my-app" || len(req.accounts) != 2 {
			t.Fatalf("unexpected request: %#v", req)
		}
		return promptResult{Saved: []savedKey{{Account: "FIRST_KEY", Length: 12}, {Account: "SECOND_KEY", Length: 24}}}, nil
	}
	t.Cleanup(func() { launchPrompt = old })

	code, out, errOut := runForTest(t, "-s", "my-app", "FIRST_KEY", "SECOND_KEY")
	if code != 0 || errOut != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	for _, expected := range []string{"Saved 2 secrets", "FIRST_KEY: 12 chars", "SECOND_KEY: 24 chars"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("missing %q in %q", expected, out)
		}
	}
}

func TestCancelledPromptUsesDistinctExitCode(t *testing.T) {
	old := launchPrompt
	launchPrompt = func(request) (promptResult, error) {
		return promptResult{Cancelled: true}, nil
	}
	t.Cleanup(func() { launchPrompt = old })

	code, out, _ := runForTest(t, "-s", "my-app", "TOKEN")
	if code != 2 || !strings.Contains(out, "Cancelled") {
		t.Fatalf("code=%d stdout=%q", code, out)
	}
}
