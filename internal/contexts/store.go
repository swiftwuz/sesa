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

func (s Store) CheckStorage() error {
	info, err := os.Lstat(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("context storage must not be a symbolic link")
	}
	if !info.IsDir() {
		return errors.New("context storage is not a directory")
	}
	return nil
}

func (s Store) Ensure(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	if err := s.CheckStorage(); err != nil {
		return err
	}
	if err := ensureDirectory(filepath.Dir(s.Home(name))); err != nil {
		return err
	}
	return ensureDirectory(s.Home(name))
}

func (s Store) Exists(name string) (bool, error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	info, err := os.Lstat(s.Home(name))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("%s must not be a symbolic link", s.Home(name))
	}
	return info.IsDir(), nil
}

func (s Store) Inspect(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	for _, path := range []string{filepath.Dir(s.Home(name)), s.Home(name)} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s must not be a symbolic link", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
	}
	return nil
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
		if ValidateName(entry.Name()) != nil {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("context %q must not be a symbolic link", entry.Name())
		}
		if !entry.IsDir() {
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

func ensureDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.Mkdir(path, 0o700)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s must not be a symbolic link", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}
