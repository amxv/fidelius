//go:build linux

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZenityPromptReturnsNamedSecrets(t *testing.T) {
	dir := t.TempDir()
	zenity := filepath.Join(dir, "zenity")
	if err := os.WriteFile(zenity, []byte("#!/bin/sh\nprintf 'first-secret\\nsecond-secret\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	result, err := platformPrompt(promptRequest{
		Message:         "Need two secrets.",
		Names:           []string{"FIRST", "SECOND"},
		AutoDeleteLabel: "5 minutes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Cancelled || result.Values["FIRST"] != "first-secret" || result.Values["SECOND"] != "second-secret" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestZenityCancellation(t *testing.T) {
	dir := t.TempDir()
	zenity := filepath.Join(dir, "zenity")
	if err := os.WriteFile(zenity, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	result, err := platformPrompt(promptRequest{Names: []string{"TOKEN"}, AutoDeleteLabel: "5 minutes"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Cancelled {
		t.Fatalf("expected cancellation: %#v", result)
	}
}
