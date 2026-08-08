package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"sesa/internal/codex"
	"sesa/internal/contexts"
)

const helpText = `Sesa switches safely between isolated Codex accounts.

Usage:
  sesa help
  sesa doctor
  sesa list
  sesa login <context>
  sesa status <context>
  sesa run <context> [-- <codex arguments...>]

Commands:
  help              Show this help
  doctor            Diagnose the Codex installation and isolated contexts
  list              List available contexts
  login <context>   Log in through the official Codex CLI
  status <context>  Show the official Codex login status
  run <context>     Launch Codex in an isolated context

Context names use 1-32 lowercase letters, digits, or hyphens and must start
with a letter. Pass Codex arguments after --.`

const usage = `Usage:
  sesa help
  sesa doctor
  sesa list
  sesa login <context>
  sesa status <context>
  sesa run <context> [-- <codex arguments...>]`

type runner interface {
	Check() error
	Version() (string, error)
	LoginStatus(home string) error
	Run(home string, args []string) error
}

type App struct {
	userConfigDir func() (string, error)
	codex         runner
	stdout        io.Writer
	stderr        io.Writer
}

func New(stdin io.Reader, stdout, stderr io.Writer) App {
	return App{
		userConfigDir: os.UserConfigDir,
		codex:         codex.Runner{Stdin: stdin, Stdout: stdout, Stderr: stderr},
		stdout:        stdout,
		stderr:        stderr,
	}
}

func (a App) Run(args []string) int {
	inv, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: %v\n\n%s\n", err, usage)
		return 2
	}
	if inv.action == actionHelp {
		fmt.Fprintln(a.stdout, helpText)
		return 0
	}

	configDir, err := a.userConfigDir()
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: locate user config directory: %v\n", err)
		return 1
	}
	store := contexts.New(configDir)

	if inv.action == actionDoctor {
		return a.doctor(store)
	}
	if inv.action == actionList {
		return a.list(store)
	}
	if inv.action == actionStatus {
		exists, err := store.Exists(inv.context)
		if err != nil {
			fmt.Fprintf(a.stderr, "sesa: inspect context %q: %v\n", inv.context, err)
			return 1
		}
		if !exists {
			fmt.Fprintf(a.stderr, "sesa: context %q does not exist\n", inv.context)
			return 1
		}
	} else if err := store.Ensure(inv.context); err != nil {
		fmt.Fprintf(a.stderr, "sesa: create context %q: %v\n", inv.context, err)
		return 1
	}

	if err := a.codex.Check(); err != nil {
		return a.codexError(err)
	}
	fmt.Fprintf(a.stderr, "Sesa context: %s\n", strings.ToUpper(inv.context))
	if err := a.codex.Run(store.Home(inv.context), inv.codexArgs); err != nil {
		return a.codexError(err)
	}

	return 0
}

func (a App) doctor(store contexts.Store) int {
	fmt.Fprintln(a.stdout, "Sesa doctor")
	healthy := true

	version, err := a.codex.Version()
	if err != nil {
		fmt.Fprintf(a.stdout, "✗ Codex CLI: %v\n", err)
		healthy = false
	} else {
		fmt.Fprintf(a.stdout, "✓ Codex CLI found (%s)\n", version)
	}

	if err := store.CheckStorage(); err != nil {
		fmt.Fprintf(a.stdout, "✗ Context storage: %v\n", err)
		return 1
	}
	fmt.Fprintln(a.stdout, "✓ Context storage accessible")

	names, err := store.List()
	if err != nil {
		fmt.Fprintf(a.stdout, "✗ Context discovery: %v\n", err)
		return 1
	}
	if len(names) == 0 {
		fmt.Fprintln(a.stdout, "✓ No contexts configured")
	}
	for _, name := range names {
		if err := store.Inspect(name); err != nil {
			fmt.Fprintf(a.stdout, "✗ %s: unsafe context home: %v\n", name, err)
			healthy = false
			continue
		}
		fmt.Fprintf(a.stdout, "✓ %s: isolated home\n", name)
		if err := a.codex.LoginStatus(store.Home(name)); err != nil {
			fmt.Fprintf(a.stdout, "✗ %s: not authenticated\n", name)
			healthy = false
		} else {
			fmt.Fprintf(a.stdout, "✓ %s: authenticated\n", name)
		}
	}

	if !healthy {
		return 1
	}
	return 0
}

func (a App) codexError(err error) int {
	if errors.Is(err, codex.ErrNotFound) {
		fmt.Fprintln(a.stderr, "sesa: codex executable not found in PATH")
		fmt.Fprintln(a.stderr, "Install the Codex CLI: https://developers.openai.com/codex/cli")
		return 127
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() >= 0 {
		return exitError.ExitCode()
	}
	fmt.Fprintf(a.stderr, "sesa: launch codex: %v\n", err)
	return 1
}

func (a App) list(store contexts.Store) int {
	names, err := store.List()
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: list contexts: %v\n", err)
		return 1
	}
	if len(names) == 0 {
		fmt.Fprintln(a.stdout, "No contexts found.")
		return 0
	}
	for _, name := range names {
		fmt.Fprintln(a.stdout, name)
	}
	return 0
}
