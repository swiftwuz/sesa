package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sesa/internal/codex"
	"sesa/internal/contexts"
	"sesa/internal/mappings"
	"sesa/internal/repository"
	"sesa/internal/vscode"
)

const helpText = `Sesa switches safely between isolated Codex accounts.

Usage:
  sesa help
  sesa doctor
  sesa list
  sesa link <context>
  sesa current
  sesa unlink
  sesa login <context>
  sesa status <context>
  sesa run [<context>] [--allow-mismatch] [-- <codex arguments...>]
  sesa code <context> [path]

Commands:
  help              Show this help
  doctor            Diagnose the Codex installation and isolated contexts
  list              List available contexts
  link <context>    Map the current Git repository to a context
  current           Show the current repository's mapped context
  unlink            Remove the current repository's mapping
  login <context>   Log in through the official Codex CLI
  status <context>  Show the official Codex login status
  run [<context>]   Launch Codex in an isolated context
  code <context>    Open VS Code in an isolated context

Context names use 1-32 lowercase letters, digits, or hyphens and must start
with a letter. Pass Codex arguments after --.`

const usage = `Usage:
  sesa help
  sesa doctor
  sesa list
  sesa link <context>
  sesa current
  sesa unlink
  sesa login <context>
  sesa status <context>
  sesa run [<context>] [--allow-mismatch] [-- <codex arguments...>]
  sesa code <context> [path]`

type runner interface {
	Check() error
	Version() (string, error)
	LoginStatus(home string) error
	Run(home string, args []string) error
}

type codeRunner interface {
	Check() error
	Run(home, userDataDir, target string) error
}

type App struct {
	userConfigDir  func() (string, error)
	workingDir     func() (string, error)
	repositoryRoot func(string) (string, error)
	codex          runner
	code           codeRunner
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
}

func New(stdin io.Reader, stdout, stderr io.Writer) App {
	return App{
		userConfigDir:  os.UserConfigDir,
		workingDir:     os.Getwd,
		repositoryRoot: repository.GitRoot,
		codex:          codex.Runner{Stdin: stdin, Stdout: stdout, Stderr: stderr},
		code:           vscode.Runner{Stdout: stdout, Stderr: stderr},
		stdin:          stdin,
		stdout:         stdout,
		stderr:         stderr,
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
	mappingStore := mappings.New(configDir)

	switch inv.action {
	case actionDoctor:
		return a.doctor(store, mappingStore)
	case actionList:
		return a.list(store)
	case actionLink, actionCurrent, actionUnlink:
		return a.repositoryCommand(inv, store, mappingStore)
	case actionCode:
		return a.codeCommand(inv, store)
	default:
		return a.codexCommand(inv, store, mappingStore)
	}
}

func (a App) codeCommand(inv invocation, store contexts.Store) int {
	if err := a.prepareContext(store, inv.context, true); err != nil {
		fmt.Fprintf(a.stderr, "sesa: %v\n", err)
		return 1
	}
	if err := a.code.Check(); err != nil {
		return a.codeError(err)
	}
	target, err := a.absoluteTarget(inv.target)
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: resolve VS Code path: %v\n", err)
		return 1
	}
	if err := store.EnsureVSCodeUserData(inv.context); err != nil {
		fmt.Fprintf(a.stderr, "sesa: prepare VS Code isolation for %q: %v\n", inv.context, err)
		return 1
	}

	fmt.Fprintf(a.stderr, "Sesa VS Code context: %s\n", strings.ToUpper(inv.context))
	if err := a.code.Run(store.Home(inv.context), store.VSCodeUserData(inv.context), target); err != nil {
		return a.codeError(err)
	}
	return 0
}

func (a App) absoluteTarget(target string) (string, error) {
	if filepath.IsAbs(target) {
		return filepath.Clean(target), nil
	}
	workingDir, err := a.workingDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(workingDir, target)), nil
}

func (a App) codeError(err error) int {
	if errors.Is(err, vscode.ErrNotFound) {
		fmt.Fprintln(a.stderr, "sesa: code executable not found in PATH")
		fmt.Fprintln(a.stderr, "Install the VS Code shell command, then try again.")
		return 127
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() >= 0 {
		return exitError.ExitCode()
	}
	fmt.Fprintf(a.stderr, "sesa: launch VS Code: %v\n", err)
	return 1
}

func (a App) codexCommand(inv invocation, store contexts.Store, mappingStore mappings.Store) int {
	requireExistingContext := false
	if inv.action == actionRun {
		selected, mappedSelection, err := a.selectRunContext(inv, mappingStore)
		if err != nil {
			fmt.Fprintf(a.stderr, "sesa: %v\n", err)
			return 1
		}
		inv.context = selected
		requireExistingContext = mappedSelection
	}
	if err := a.prepareContext(store, inv.context, inv.action == actionStatus || requireExistingContext); err != nil {
		fmt.Fprintf(a.stderr, "sesa: %v\n", err)
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

func (a App) prepareContext(store contexts.Store, context string, requireExisting bool) error {
	if requireExisting {
		exists, err := store.Exists(context)
		if err != nil {
			return fmt.Errorf("inspect context %q: %w", context, err)
		}
		if !exists {
			return fmt.Errorf("context %q does not exist", context)
		}
		return nil
	}
	if err := store.Ensure(context); err != nil {
		return fmt.Errorf("create context %q: %w", context, err)
	}
	return nil
}

func (a App) doctor(store contexts.Store, mappingStore mappings.Store) int {
	fmt.Fprintln(a.stdout, "Sesa doctor")
	healthy := true

	version, err := a.codex.Version()
	if err != nil {
		fmt.Fprintf(a.stdout, "✗ Codex CLI: %v\n", err)
		healthy = false
	} else {
		fmt.Fprintf(a.stdout, "✓ Codex CLI found (%s)\n", version)
	}
	if err := a.code.Check(); err != nil {
		fmt.Fprintf(a.stdout, "✗ VS Code shell command: %v\n", err)
		healthy = false
	} else {
		fmt.Fprintln(a.stdout, "✓ VS Code shell command found")
	}

	if err := store.CheckStorage(); err != nil {
		fmt.Fprintf(a.stdout, "✗ Context storage: %v\n", err)
		return 1
	}
	fmt.Fprintln(a.stdout, "✓ Context storage accessible")
	entries, err := mappingStore.Entries()
	if err != nil {
		fmt.Fprintf(a.stdout, "✗ Repository mappings: %v\n", err)
		healthy = false
	} else {
		fmt.Fprintf(a.stdout, "✓ Repository mappings readable (%d)\n", len(entries))
	}

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
