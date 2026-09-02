package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/amxv/fidelius/internal/buildinfo"
)

const commandName = "fidelius"

var launchPrompt = launchNativePrompt

type request struct {
	service  string
	message  string
	accounts []string
}

type savedKey struct {
	Account string `json:"account"`
	Length  int    `json:"length"`
}

type promptResult struct {
	Cancelled bool       `json:"cancelled"`
	Saved     []savedKey `json:"saved"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || hasHelp(args) {
		printHelp(stdout)
		return 0
	}
	if hasVersion(args) {
		fmt.Fprintf(stdout, "%s %s\n", commandName, buildinfo.CurrentVersion())
		return 0
	}

	req, err := parseRequest(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\nRun `%s --help` for usage.\n", err, commandName)
		return 2
	}

	result, err := launchPrompt(req)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if result.Cancelled {
		fmt.Fprintln(stdout, "Cancelled. No secrets were saved.")
		return 2
	}

	noun := "secrets"
	if len(result.Saved) == 1 {
		noun = "secret"
	}
	fmt.Fprintf(stdout, "Saved %d %s to macOS Keychain (service: %s).\n", len(result.Saved), noun, req.service)
	for _, item := range result.Saved {
		fmt.Fprintf(stdout, "%s: %d chars\n", item.Account, item.Length)
	}
	return 0
}

func parseRequest(args []string) (request, error) {
	var req request
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-s" || arg == "--service":
			if i+1 >= len(args) {
				return request{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			req.service = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--service="):
			req.service = strings.TrimSpace(strings.TrimPrefix(arg, "--service="))
		case arg == "-m" || arg == "--message":
			if i+1 >= len(args) {
				return request{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			req.message = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--message="):
			req.message = strings.TrimSpace(strings.TrimPrefix(arg, "--message="))
		case arg == "--":
			req.accounts = append(req.accounts, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(arg, "-"):
			return request{}, fmt.Errorf("unknown option %q", arg)
		default:
			req.accounts = append(req.accounts, arg)
		}
	}

	if req.service == "" {
		return request{}, fmt.Errorf("missing Keychain service (use -s SERVICE)")
	}
	if len(req.accounts) == 0 {
		return request{}, fmt.Errorf("provide at least one secret name")
	}

	seen := make(map[string]struct{}, len(req.accounts))
	for i, account := range req.accounts {
		account = strings.TrimSpace(account)
		if account == "" {
			return request{}, fmt.Errorf("secret names cannot be empty")
		}
		if _, ok := seen[account]; ok {
			return request{}, fmt.Errorf("duplicate secret name %q", account)
		}
		seen[account] = struct{}{}
		req.accounts[i] = account
	}
	return req, nil
}

func hasHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}

func hasVersion(args []string) bool {
	return len(args) == 1 && (args[0] == "-v" || args[0] == "--version")
}

func launchNativePrompt(req request) (promptResult, error) {
	if runtime.GOOS != "darwin" {
		return promptResult{}, fmt.Errorf("Fidelius currently requires macOS")
	}

	helper, err := helperExecutable()
	if err != nil {
		return promptResult{}, err
	}

	args := []string{"--service", req.service}
	if req.message != "" {
		args = append(args, "--message", req.message)
	}
	args = append(args, req.accounts...)
	cmd := exec.Command(helper, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return promptResult{}, fmt.Errorf("could not run Fidelius prompt: %s", detail)
	}

	var result promptResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return promptResult{}, fmt.Errorf("Fidelius prompt returned an invalid response")
	}
	return result, nil
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

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "fidelius allows agents to ask humans to enter secrets and saves them to macOS Keychain.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  fidelius -s SERVICE [-m MESSAGE] ACCOUNT [ACCOUNT...]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -s, --service SERVICE  Keychain service name")
	fmt.Fprintln(w, "  -m, --message MESSAGE  Short explanation shown to the human")
	fmt.Fprintln(w, "  -h, --help             Show help")
	fmt.Fprintln(w, "  -v, --version          Show version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  fidelius -s my-app OPENAI_API_KEY")
	fmt.Fprintln(w, `  fidelius -s scraper -m "I need the Maps key to finish the scrape." GOOGLE_MAPS_API_KEY`)
	fmt.Fprintln(w, "  fidelius -s my-app OPENAI_API_KEY STRIPE_SECRET_KEY")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The command waits until the human saves or cancels the prompt.")
	fmt.Fprintln(w, "Fidelius never prints secret values.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Retrieve a saved secret with macOS security:")
	fmt.Fprintln(w, "  security find-generic-password -s my-app -a OPENAI_API_KEY -w")
}
