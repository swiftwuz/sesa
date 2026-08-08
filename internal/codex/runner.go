package codex

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

var ErrNotFound = errors.New("codex executable not found in PATH")

type Runner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Check() error {
	if _, err := exec.LookPath("codex"); err != nil {
		return ErrNotFound
	}
	return nil
}

func (r Runner) Version() (string, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return "", ErrNotFound
	}
	output, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (r Runner) LoginStatus(home string) error {
	path, err := exec.LookPath("codex")
	if err != nil {
		return ErrNotFound
	}
	cmd := exec.Command(path, "login", "status")
	cmd.Env = WithEnvironment(os.Environ(), "CODEX_HOME", home)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func (r Runner) Run(home string, args []string) error {
	path, err := exec.LookPath("codex")
	if err != nil {
		return ErrNotFound
	}

	cmd := exec.Command(path, args...)
	cmd.Env = WithEnvironment(os.Environ(), "CODEX_HOME", home)
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}

func WithEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}
