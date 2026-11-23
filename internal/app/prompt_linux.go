package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func platformPrompt(req promptRequest) (promptResult, error) {
	executable, err := os.Executable()
	if err != nil {
		return promptResult{}, err
	}
	return linuxPrompt(filepath.Join(filepath.Dir(executable), "fidelius-ui"), req)
}

func linuxPrompt(directory string, req promptRequest) (promptResult, error) {
	backends := []string{"gtk", "qt"}
	for _, desktop := range strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":") {
		if strings.EqualFold(desktop, "KDE") || strings.EqualFold(desktop, "Plasma") {
			backends = []string{"qt", "gtk"}
			break
		}
	}
	message := req.Message
	if strings.TrimSpace(message) == "" {
		message = "Paste the requested secrets."
	}
	request := strings.Join(append([]string{message, "Auto-delete in " + req.AutoDeleteLabel + "."}, req.Names...), "\x00") + "\x00"
	for _, backend := range backends {
		result, unavailable, err := runLinuxHelper(filepath.Join(directory, backend), request, req.Names)
		if !unavailable {
			return result, err
		}
	}
	return promptResult{}, errors.New("could not start a graphical secret prompt: use a desktop session with GTK 4 or Qt 6 Widgets and its display plugin installed")
}

func runLinuxHelper(path, request string, names []string) (promptResult, bool, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return promptResult{}, false, err
	}
	defer read.Close()
	cmd := exec.Command(path)
	cmd.ExtraFiles = []*os.File{write}
	cmd.Stdin = strings.NewReader(request)
	err = cmd.Start()
	write.Close()
	if err != nil {
		return promptResult{}, true, nil
	}

	// A ready byte distinguishes toolkit startup failures from a closed or failed form.
	data, readErr := io.ReadAll(read)
	waitErr := cmd.Wait()
	if readErr != nil {
		return promptResult{}, false, fmt.Errorf("could not read the desktop prompt")
	}
	if len(data) == 0 && waitErr != nil {
		return promptResult{}, true, nil
	}
	if len(data) == 0 || data[0] != 'R' {
		return promptResult{}, false, errors.New("invalid desktop prompt response")
	}
	if exit, ok := waitErr.(*exec.ExitError); ok && exit.ExitCode() == 2 && len(data) == 1 {
		return promptResult{Cancelled: true}, false, nil
	}
	if waitErr != nil {
		return promptResult{}, false, errors.New("desktop prompt closed unexpectedly")
	}
	parts := bytes.Split(data[1:], []byte{0})
	if len(parts) != len(names)+1 || len(parts[len(names)]) != 0 {
		return promptResult{}, false, errors.New("incomplete desktop prompt response")
	}
	values := make(map[string]string, len(names))
	for i, name := range names {
		values[name] = string(parts[i])
	}
	return promptResult{Values: values}, false, nil
}
