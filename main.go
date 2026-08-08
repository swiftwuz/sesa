package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const helpText = `Sesa switches safely between isolated Codex accounts.

Usage:
  sesa help
  sesa list
  sesa login <context>
  sesa status <context>
  sesa run <context> [-- <codex arguments...>]

Commands:
  help              Show this help
  list              List available contexts
  login <context>   Log in through the official Codex CLI
  status <context>  Show the official Codex login status
  run <context>     Launch Codex in an isolated context

Context names use 1-32 lowercase letters, digits, or hyphens and must start
with a letter. Pass Codex arguments after --.`

const usage = `Usage:
  sesa help
  sesa list
  sesa login <context>
  sesa status <context>
  sesa run <context> [-- <codex arguments...>]`

var contextNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

type invocation struct {
	action    string
	context   string
	codexArgs []string
}

type launcher func(path string, args, env []string) error

func main() {
	os.Exit(run(os.Args[1:], os.UserConfigDir, exec.LookPath, launch))
}

func run(args []string, userConfigDir func() (string, error), lookPath func(string) (string, error), launchCodex launcher) int {
	inv, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sesa: %v\n\n%s\n", err, usage)
		return 2
	}
	if inv.action == "help" {
		fmt.Fprintln(os.Stdout, helpText)
		return 0
	}

	configDir, err := userConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sesa: locate user config directory: %v\n", err)
		return 1
	}
	if inv.action == "list" {
		contexts, err := listContexts(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sesa: list contexts: %v\n", err)
			return 1
		}
		if len(contexts) == 0 {
			fmt.Fprintln(os.Stdout, "No contexts found.")
			return 0
		}
		for _, context := range contexts {
			fmt.Fprintln(os.Stdout, context)
		}
		return 0
	}

	codexHome := contextHome(configDir, inv.context)
	if inv.action == "status" {
		exists, err := contextExists(codexHome)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sesa: inspect context %q: %v\n", inv.context, err)
			return 1
		}
		if !exists {
			fmt.Fprintf(os.Stderr, "sesa: context %q does not exist\n", inv.context)
			return 1
		}
	} else {
		if err := os.MkdirAll(codexHome, 0o700); err != nil {
			fmt.Fprintf(os.Stderr, "sesa: create context %q: %v\n", inv.context, err)
			return 1
		}
	}

	codexPath, err := lookPath("codex")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sesa: codex executable not found in PATH")
		fmt.Fprintln(os.Stderr, "Install the Codex CLI: https://developers.openai.com/codex/cli")
		return 127
	}

	fmt.Fprintf(os.Stderr, "Sesa context: %s\n", strings.ToUpper(inv.context))
	env := withEnvironment(os.Environ(), "CODEX_HOME", codexHome)
	if err := launchCodex(codexPath, inv.codexArgs, env); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() >= 0 {
			return exitError.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "sesa: launch codex: %v\n", err)
		return 1
	}

	return 0
}

func parseArgs(args []string) (invocation, error) {
	if len(args) == 0 {
		return invocation{}, errors.New("expected a command")
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		if len(args) != 1 {
			return invocation{}, errors.New("help does not accept arguments")
		}
		return invocation{action: "help"}, nil
	}
	if args[0] == "list" {
		if len(args) != 1 {
			return invocation{}, errors.New("list does not accept arguments")
		}
		return invocation{action: "list"}, nil
	}
	if len(args) < 2 {
		return invocation{}, errors.New("expected a context")
	}

	command, context := args[0], args[1]
	if !contextNamePattern.MatchString(context) {
		return invocation{}, fmt.Errorf("invalid context %q (use 1-32 lowercase letters, digits, or hyphens; start with a letter)", context)
	}

	switch command {
	case "login":
		if len(args) != 2 {
			return invocation{}, errors.New("login does not accept additional arguments")
		}
		return invocation{action: "login", context: context, codexArgs: []string{"login"}}, nil
	case "status":
		if len(args) != 2 {
			return invocation{}, errors.New("status does not accept additional arguments")
		}
		return invocation{action: "status", context: context, codexArgs: []string{"login", "status"}}, nil
	case "run":
		if len(args) == 2 {
			return invocation{action: "run", context: context}, nil
		}
		if args[2] != "--" {
			return invocation{}, errors.New("put Codex arguments after --")
		}
		return invocation{action: "run", context: context, codexArgs: args[3:]}, nil
	default:
		return invocation{}, fmt.Errorf("unknown command %q", command)
	}
}

func contextHome(userConfigDir, context string) string {
	return filepath.Join(userConfigDir, "sesa", "contexts", context, "codex")
}

func listContexts(userConfigDir string) ([]string, error) {
	root := filepath.Join(userConfigDir, "sesa", "contexts")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	contexts := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !contextNamePattern.MatchString(entry.Name()) {
			continue
		}
		exists, err := contextExists(contextHome(userConfigDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if exists {
			contexts = append(contexts, entry.Name())
		}
	}
	return contexts, nil
}

func contextExists(codexHome string) (bool, error) {
	info, err := os.Stat(codexHome)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func withEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func launch(path string, args, env []string) error {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
