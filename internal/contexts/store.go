package contexts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

type Store struct {
	root string
}

func New(userConfigDir string) Store {
	return Store{root: filepath.Join(userConfigDir, "sesa", "contexts")}
}

func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid context %q (use 1-32 lowercase letters, digits, or hyphens; start with a letter)", name)
	}
	return nil
}

func (s Store) Home(name string) string {
	return filepath.Join(s.root, name, "codex")
}

func (s Store) Ensure(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	return os.MkdirAll(s.Home(name), 0o700)
}

func (s Store) Exists(name string) (bool, error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	info, err := os.Stat(s.Home(name))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (s Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || ValidateName(entry.Name()) != nil {
			continue
		}
		exists, err := s.Exists(entry.Name())
		if err != nil {
			return nil, err
		}
		if exists {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}
