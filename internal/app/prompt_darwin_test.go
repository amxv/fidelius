//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinPromptUsesPrivateInheritedPipe(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper")
	script := `#!/bin/sh
printf '%s\n' '{"cancelled":false,"values":{"TOKEN":"private-value"}}' >&3
`
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FIDELIUS_APP_EXECUTABLE", helper)

	result, err := platformPrompt(promptRequest{
		Message:         "Need a token.",
		Names:           []string{"TOKEN"},
		AutoDeleteLabel: "5 minutes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Cancelled || result.Values["TOKEN"] != "private-value" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDarwinPromptCancellationUsesPrivatePipe(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "helper")
	script := `#!/bin/sh
printf '%s\n' '{"cancelled":true,"values":{}}' >&3
`
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FIDELIUS_APP_EXECUTABLE", helper)

	result, err := platformPrompt(promptRequest{Names: []string{"TOKEN"}, AutoDeleteLabel: "5 minutes"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Cancelled {
		t.Fatalf("expected cancellation: %#v", result)
	}
}
