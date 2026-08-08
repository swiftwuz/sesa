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

const usage = `Usage:
  sesa login <context>
  sesa run <context> [-- <codex arguments...>]`

var contextNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

type invocation struct {
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

	configDir, err := userConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sesa: locate user config directory: %v\n", err)
		return 1
	}

	codexHome := contextHome(configDir, inv.context)
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "sesa: create context %q: %v\n", inv.context, err)
		return 1
	}

	codexPath, err := lookPath("codex")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sesa: codex executable not found in PATH")
		return 127
	}

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
	if len(args) < 2 {
		return invocation{}, errors.New("expected a command and context")
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
		return invocation{context: context, codexArgs: []string{"login"}}, nil
	case "run":
		if len(args) == 2 {
			return invocation{context: context}, nil
		}
		if args[2] != "--" {
			return invocation{}, errors.New("put Codex arguments after --")
		}
		return invocation{context: context, codexArgs: args[3:]}, nil
	default:
		return invocation{}, fmt.Errorf("unknown command %q", command)
	}
}

func contextHome(userConfigDir, context string) string {
	return filepath.Join(userConfigDir, "sesa", "contexts", context, "codex")
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
