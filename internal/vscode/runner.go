package vscode

import (
	"errors"
	"io"
	"os"
	"os/exec"

	"sesa/internal/codex"
)

var ErrNotFound = errors.New("code executable not found in PATH")

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (Runner) Check() error {
	if _, err := exec.LookPath("code"); err != nil {
		return ErrNotFound
	}
	return nil
}

func (r Runner) Run(home, userDataDir, target string) error {
	path, err := exec.LookPath("code")
	if err != nil {
		return ErrNotFound
	}

	cmd := exec.Command(path, "--new-window", "--user-data-dir", userDataDir, target)
	cmd.Env = codex.WithEnvironment(os.Environ(), "CODEX_HOME", home)
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}
