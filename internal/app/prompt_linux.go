//go:build linux

package app

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func platformPrompt(req promptRequest) (promptResult, error) {
	if path, err := exec.LookPath("zenity"); err == nil {
		return promptWithZenity(path, req)
	}
	if path, err := exec.LookPath("kdialog"); err == nil {
		return promptWithKDialog(path, req)
	}
	return promptResult{}, fmt.Errorf("Fidelius needs a desktop dialog helper: install `zenity` (GTK) or `kdialog` (KDE)")
}

func promptWithZenity(path string, req promptRequest) (promptResult, error) {
	text := linuxPromptText(req)
	args := []string{
		"--forms",
		"--title=Fidelius",
		"--text=" + text,
		"--separator=\n",
		"--ok-label=Save",
		"--cancel-label=Cancel",
		"--width=480",
	}
	for _, name := range req.Names {
		args = append(args, "--add-password="+name)
	}

	for {
		cmd := exec.Command(path, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
				return promptResult{Cancelled: true}, nil
			}
			detail := strings.TrimSpace(stderr.String())
			if detail == "" {
				detail = err.Error()
			}
			return promptResult{}, fmt.Errorf("zenity could not open the Fidelius prompt: %s", detail)
		}

		raw := strings.TrimSuffix(stdout.String(), "\n")
		parts := strings.Split(raw, "\n")
		if len(parts) != len(req.Names) {
			return promptResult{}, fmt.Errorf("zenity returned the wrong number of secrets")
		}
		values := make(map[string]string, len(parts))
		missing := false
		for i, name := range req.Names {
			if parts[i] == "" {
				missing = true
			}
			values[name] = parts[i]
		}
		if !missing {
			return promptResult{Values: values}, nil
		}
		_ = exec.Command(path, "--error", "--title=Fidelius", "--text=Every secret is required.").Run()
	}
}

func promptWithKDialog(path string, req promptRequest) (promptResult, error) {
	values := make(map[string]string, len(req.Names))
	for i, name := range req.Names {
		for {
			text := linuxPromptText(req)
			if len(req.Names) > 1 {
				text += fmt.Sprintf("\n\n%s (%d of %d)", name, i+1, len(req.Names))
			} else {
				text += "\n\n" + name
			}
			cmd := exec.Command(path, "--title", "Fidelius", "--password", text)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err != nil {
				if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
					return promptResult{Cancelled: true}, nil
				}
				detail := strings.TrimSpace(stderr.String())
				if detail == "" {
					detail = err.Error()
				}
				return promptResult{}, fmt.Errorf("kdialog could not open the Fidelius prompt: %s", detail)
			}
			value := strings.TrimSuffix(stdout.String(), "\n")
			if value != "" {
				values[name] = value
				break
			}
			_ = exec.Command(path, "--error", "Secret cannot be empty.", "--title", "Fidelius").Run()
		}
	}
	return promptResult{Values: values}, nil
}

func linuxPromptText(req promptRequest) string {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "Paste the requested secrets."
	}
	return fmt.Sprintf("%s\n\nAuto-delete in %s.", message, req.AutoDeleteLabel)
}
