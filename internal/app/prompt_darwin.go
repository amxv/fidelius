//go:build darwin

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type helperResponse struct {
	Cancelled bool              `json:"cancelled"`
	Values    map[string]string `json:"values"`
}

func platformPrompt(req promptRequest) (promptResult, error) {
	helper, err := helperExecutable()
	if err != nil {
		return promptResult{}, err
	}

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return promptResult{}, fmt.Errorf("could not create private prompt pipe: %w", err)
	}
	defer readPipe.Close()

	args := []string{"--secret-fd", "3", "--auto-delete", req.AutoDeleteLabel}
	if req.Message != "" {
		args = append(args, "--message", req.Message)
	}
	args = append(args, req.Names...)
	cmd := exec.Command(helper, args...)
	cmd.ExtraFiles = []*os.File{writePipe}
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		writePipe.Close()
		return promptResult{}, fmt.Errorf("could not run Fidelius prompt: %w", err)
	}
	_ = writePipe.Close()

	type readResult struct {
		data []byte
		err  error
	}
	readCh := make(chan readResult, 1)
	go func() {
		data, err := io.ReadAll(readPipe)
		readCh <- readResult{data: data, err: err}
	}()

	waitErr := cmd.Wait()
	result := <-readCh
	if waitErr != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = waitErr.Error()
		}
		return promptResult{}, fmt.Errorf("could not run Fidelius prompt: %s", detail)
	}
	if result.err != nil {
		return promptResult{}, fmt.Errorf("could not read Fidelius prompt response: %w", result.err)
	}

	var response helperResponse
	if err := json.Unmarshal(result.data, &response); err != nil {
		return promptResult{}, fmt.Errorf("Fidelius prompt returned an invalid response")
	}
	return promptResult{Cancelled: response.Cancelled, Values: response.Values}, nil
}

func helperExecutable() (string, error) {
	if override := strings.TrimSpace(os.Getenv("FIDELIUS_APP_EXECUTABLE")); override != "" {
		return override, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not locate Fidelius executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	helper := filepath.Join(filepath.Dir(executable), "Fidelius.app", "Contents", "MacOS", "fidelius-ui")
	if info, err := os.Stat(helper); err == nil && !info.IsDir() {
		return helper, nil
	}
	return "", fmt.Errorf("Fidelius.app is missing next to %s; reinstall Fidelius", executable)
}
