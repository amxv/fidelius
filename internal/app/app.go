package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/amxv/fidelius/internal/buildinfo"
)

const commandName = "fidelius"

var launchPrompt = platformPrompt
var scheduleDelete = scheduleAutoDelete

type askRequest struct {
	message string
	names   []string
}

type promptRequest struct {
	Message         string
	Names           []string
	AutoDeleteLabel string
}

type promptResult struct {
	Cancelled bool
	Values    map[string]string
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		printRootHelp(stdout)
		return 0
	}
	if len(args) == 1 && isVersion(args[0]) {
		fmt.Fprintf(stdout, "%s %s\n", commandName, buildinfo.CurrentVersion())
		return 0
	}

	switch args[0] {
	case "ask":
		return runAsk(args[1:], stdout, stderr)
	case "timeout":
		return runTimeout(args[1:], stdout, stderr)
	case internalCleanupCommand:
		return runCleanup(args[1:])
	default:
		fmt.Fprintf(stderr, "Error: unknown command %q\nRun `%s --help` for usage.\n", args[0], commandName)
		return 2
	}
}

func runAsk(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelp(args[0]) {
		printAskHelp(stdout)
		return 0
	}

	req, err := parseAsk(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\nRun `%s ask --help` for usage.\n", err, commandName)
		return 2
	}

	autoDelete, err := loadAutoDeleteTimeout()
	if err != nil {
		fmt.Fprintf(stderr, "Error: could not read Fidelius configuration: %v\n", err)
		return 1
	}
	_ = cleanupExpiredSessions()

	result, err := launchPrompt(promptRequest{
		Message:         req.message,
		Names:           req.names,
		AutoDeleteLabel: formatDurationHuman(autoDelete),
	})
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if result.Cancelled {
		fmt.Fprintln(stderr, "Cancelled. No secrets were saved.")
		return 2
	}
	if err := validatePromptResult(req.names, result.Values); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	dir, deleteAt, err := writeSecretSession(result.Values, autoDelete)
	if err != nil {
		fmt.Fprintf(stderr, "Error: could not create temporary secret files: %v\n", err)
		return 1
	}
	if err := scheduleDelete(dir, deleteAt); err != nil {
		_ = removeSecretSession(dir)
		fmt.Fprintf(stderr, "Error: could not schedule auto-delete: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, dir)
	for _, name := range req.names {
		fmt.Fprintf(stderr, "Received %s (%d chars).\n", name, len([]rune(result.Values[name])))
	}
	fmt.Fprintf(stderr, "Auto-delete in %s.\n", formatDurationShort(autoDelete))
	return 0
}

func runTimeout(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && isHelp(args[0]) {
		printTimeoutHelp(stdout)
		return 0
	}
	if len(args) > 1 {
		fmt.Fprintf(stderr, "Error: timeout accepts at most one duration\nRun `%s timeout --help` for usage.\n", commandName)
		return 2
	}

	if len(args) == 0 {
		d, err := loadAutoDeleteTimeout()
		if err != nil {
			fmt.Fprintf(stderr, "Error: could not read Fidelius configuration: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Auto-delete timeout: %s\n", formatDurationShort(d))
		return 0
	}

	d, err := parseAutoDeleteDuration(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if err := saveAutoDeleteTimeout(d); err != nil {
		fmt.Fprintf(stderr, "Error: could not save Fidelius configuration: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Auto-delete timeout set to %s.\n", formatDurationShort(d))
	return 0
}

func parseAsk(args []string) (askRequest, error) {
	var req askRequest
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-m" || arg == "--message":
			if i+1 >= len(args) {
				return askRequest{}, fmt.Errorf("%s requires a value", arg)
			}
			i++
			req.message = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--message="):
			req.message = strings.TrimSpace(strings.TrimPrefix(arg, "--message="))
		case arg == "--":
			req.names = append(req.names, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(arg, "-"):
			return askRequest{}, fmt.Errorf("unknown option %q", arg)
		default:
			req.names = append(req.names, arg)
		}
	}

	if len(req.names) == 0 {
		return askRequest{}, fmt.Errorf("provide at least one secret name")
	}
	seen := make(map[string]struct{}, len(req.names))
	for i, name := range req.names {
		name = strings.TrimSpace(name)
		if err := validateSecretName(name); err != nil {
			return askRequest{}, err
		}
		if _, ok := seen[name]; ok {
			return askRequest{}, fmt.Errorf("duplicate secret name %q", name)
		}
		seen[name] = struct{}{}
		req.names[i] = name
	}
	return req, nil
}

func validatePromptResult(names []string, values map[string]string) error {
	if len(values) != len(names) {
		return fmt.Errorf("Fidelius prompt returned the wrong number of secrets")
	}
	for _, name := range names {
		value, ok := values[name]
		if !ok {
			return fmt.Errorf("Fidelius prompt did not return %s", name)
		}
		if value == "" {
			return fmt.Errorf("Fidelius prompt returned an empty value for %s", name)
		}
	}
	return nil
}

func isHelp(v string) bool {
	return v == "-h" || v == "--help" || v == "help"
}

func isVersion(v string) bool {
	return v == "-v" || v == "--version"
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, "fidelius allows agents to securely ask humans for secrets.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  fidelius ask [options] NAME [NAME...]")
	fmt.Fprintln(w, "  fidelius timeout [DURATION]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  ask       Ask a human for secrets")
	fmt.Fprintln(w, "  timeout   Show or set the auto-delete timeout (default: 5m)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Example:")
	fmt.Fprintln(w, `  secrets=$(fidelius ask -m "I need the Maps key to finish the scraper." GOOGLE_MAPS_API_KEY)`)
	fmt.Fprintln(w, `  gh secret set GOOGLE_MAPS_API_KEY < "$secrets/GOOGLE_MAPS_API_KEY"`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Fidelius returns only a private temporary directory path.")
	fmt.Fprintln(w, "Secret values are never printed, and the directory auto-deletes.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -h, --help      Show help")
	fmt.Fprintln(w, "  -v, --version   Show version")
}

func printAskHelp(w io.Writer) {
	fmt.Fprintln(w, "Ask a human for one or more secrets.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  fidelius ask [-m MESSAGE] NAME [NAME...]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -m, --message TEXT   Explain why the secrets are needed")
	fmt.Fprintln(w, "  -h, --help           Show help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Fidelius opens a native prompt, then prints only the path to a private")
	fmt.Fprintln(w, "temporary directory containing one file per secret. The directory")
	fmt.Fprintln(w, "auto-deletes after the configured timeout.")
}

func printTimeoutHelp(w io.Writer) {
	fmt.Fprintln(w, "Show or set how long temporary secret files live before auto-delete.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  fidelius timeout")
	fmt.Fprintln(w, "  fidelius timeout DURATION")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  fidelius timeout 30s")
	fmt.Fprintln(w, "  fidelius timeout 10m")
	fmt.Fprintln(w, "  fidelius timeout 2h")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Default: 5m")
}
