package mappings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sesa/internal/contexts"
)

const currentVersion = 1

type config struct {
	Version      int               `json:"version"`
	Repositories map[string]string `json:"repositories"`
}

type Store struct {
	path string
}

func New(userConfigDir string) Store {
	return Store{path: filepath.Join(userConfigDir, "sesa", "config.json")}
}

func (s Store) Get(repository string) (string, bool, error) {
	cfg, err := s.load()
	if err != nil {
		return "", false, err
	}
	context, ok := cfg.Repositories[repository]
	return context, ok, nil
}

func (s Store) Set(repository, context string) error {
	if err := validateEntry(repository, context); err != nil {
		return err
	}
	cfg, err := s.load()
	if err != nil {
		return err
	}
	cfg.Repositories[repository] = context
	return s.save(cfg)
}

func (s Store) Entries() (map[string]string, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, err
	}
	entries := make(map[string]string, len(cfg.Repositories))
	for repository, context := range cfg.Repositories {
		entries[repository] = context
	}
	return entries, nil
}

func (s Store) Remove(repository string) (bool, error) {
	cfg, err := s.load()
	if err != nil {
		return false, err
	}
	if _, ok := cfg.Repositories[repository]; !ok {
		return false, nil
	}
	delete(cfg.Repositories, repository)
	return true, s.save(cfg)
}

func (s Store) load() (config, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return newConfig(), nil
	}
	if err != nil {
		return config{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var cfg config
	if err := decoder.Decode(&cfg); err != nil {
		return config{}, fmt.Errorf("decode %s: %w", s.path, err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return config{}, fmt.Errorf("decode %s: %w", s.path, err)
	}
	if cfg.Version != currentVersion {
		return config{}, fmt.Errorf("unsupported mapping configuration version %d", cfg.Version)
	}
	if cfg.Repositories == nil {
		cfg.Repositories = make(map[string]string)
	}
	for repository, context := range cfg.Repositories {
		if err := validateEntry(repository, context); err != nil {
			return config{}, fmt.Errorf("invalid repository mapping: %w", err)
		}
	}
	return cfg, nil
}

func (s Store) save(cfg config) (err error) {
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if temporary != nil {
			_ = temporary.Close()
		}
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err = temporary.Chmod(0o600); err != nil {
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(cfg); err != nil {
		return err
	}
	if err = temporary.Sync(); err != nil {
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	temporary = nil
	if err = os.Rename(temporaryPath, s.path); err != nil {
		return err
	}
	return nil
}

func newConfig() config {
	return config{Version: currentVersion, Repositories: make(map[string]string)}
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("multiple JSON values")
}

func validateEntry(repository, context string) error {
	if !filepath.IsAbs(repository) || filepath.Clean(repository) != repository {
		return fmt.Errorf("repository path %q must be absolute and clean", repository)
	}
	if err := contexts.ValidateName(context); err != nil {
		return err
	}
	return nil
}
