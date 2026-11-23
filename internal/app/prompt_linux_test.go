package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxPrivateResponse(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ desktop, preferred string }{{"ubuntu:GNOME", "gtk"}, {"KDE", "qt"}} {
		for _, name := range []string{"gtk", "qt"} {
			script := "#!/bin/sh\ncat >/dev/null\nprintf 'R" + name + "\\000' >&3\nprintf discarded-output\n"
			if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("XDG_CURRENT_DESKTOP", tc.desktop)
		result, err := linuxPrompt(dir, promptRequest{Names: []string{"TOKEN"}})
		if err != nil || result.Values["TOKEN"] != tc.preferred {
			t.Fatal("wrong desktop selection or result channel", err)
		}
	}
}
func TestLinuxPromptFallbackAndCancel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	if err := os.WriteFile(filepath.Join(dir, "qt"), []byte("#!/bin/sh\ncat >/dev/null\nprintf R >&3\nexit 2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	result, err := linuxPrompt(dir, promptRequest{Names: []string{"TOKEN"}})
	if err != nil || !result.Cancelled {
		t.Fatal("expected cancellation after fallback", err)
	}
}
