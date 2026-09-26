//go:build windows

package app

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
)

//go:embed prompt_windows.ps1
var windowsPromptScript embed.FS

func platformPrompt(req promptRequest) (promptResult, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return promptResult{}, fmt.Errorf("could not prepare Windows prompt: %w", err)
	}
	script, err := windowsPromptScript.ReadFile("prompt_windows.ps1")
	if err != nil {
		return promptResult{}, fmt.Errorf("could not load Windows prompt: %w", err)
	}
	utf16Script := utf16.Encode([]rune(string(script)))
	commandBytes := make([]byte, len(utf16Script)*2)
	for i, value := range utf16Script {
		commandBytes[i*2] = byte(value)
		commandBytes[i*2+1] = byte(value >> 8)
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-EncodedCommand", base64.StdEncoding.EncodeToString(commandBytes))
	cmd.SysProcAttr = windowsPromptProcessAttributes()
	cmd.Stdin = strings.NewReader(base64.StdEncoding.EncodeToString(payload))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = nil // A helper failure must never forward GUI or secret data to the terminal.
	if err := cmd.Run(); err != nil {
		return promptResult{}, fmt.Errorf("could not open the Windows Fidelius prompt: %w", err)
	}
	var response struct {
		Cancelled bool              `json:"cancelled"`
		Values    map[string]string `json:"values"`
	}
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		return promptResult{}, fmt.Errorf("Windows Fidelius prompt returned an invalid response")
	}
	return promptResult{Cancelled: response.Cancelled, Values: response.Values}, nil
}

func windowsPromptProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
