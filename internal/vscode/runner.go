package vscode

import (
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/swiftwuz/sesa/internal/codex"
)

var ErrNotFound = errors.New("code executable not found in PATH")

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

type Launch struct {
	Context     string
	CodexHome   string
	UserDataDir string
	Target      string
}

func (Runner) Check() error {
	if _, err := exec.LookPath("code"); err != nil {
		return ErrNotFound
	}
	return nil
}

func (r Runner) Run(launch Launch) error {
	path, err := exec.LookPath("code")
	if err != nil {
		return ErrNotFound
	}

	cmd := exec.Command(path, "--new-window", "--user-data-dir", launch.UserDataDir, launch.Target)
	environment := codex.WithEnvironment(os.Environ(), "CODEX_HOME", launch.CodexHome)
	cmd.Env = codex.WithEnvironment(environment, "SESA_CONTEXT", launch.Context)
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}
