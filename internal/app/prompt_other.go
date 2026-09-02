//go:build !darwin && !linux

package app

import "fmt"

func platformPrompt(req promptRequest) (promptResult, error) {
	return promptResult{}, fmt.Errorf("Fidelius currently supports macOS and Linux")
}
