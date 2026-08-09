package repository

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotRepository = errors.New("current directory is not inside a Git repository")

func GitRoot(directory string) (string, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return "", errors.New("git executable not found in PATH")
	}
	output, err := exec.Command(git, "-C", directory, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", ErrNotRepository
	}
	root := strings.TrimSpace(string(output))
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	return filepath.Clean(resolved), nil
}
