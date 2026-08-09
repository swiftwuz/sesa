package mappings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"sesa/internal/contexts"
)

const (
	legacyVersion  = 1
	currentVersion = 2
)

type config struct {
	Version      int                 `json:"version"`
	Repositories map[string][]string `json:"repositories"`
}

type rawConfig struct {
	Version      int             `json:"version"`
	Repositories json.RawMessage `json:"repositories"`
}

type Store struct {
	path string
}

func New(userConfigDir string) Store {
	return Store{path: filepath.Join(userConfigDir, "sesa", "config.json")}
}

func (s Store) Get(repository string) ([]string, bool, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, false, err
	}
	allowed, ok := cfg.Repositories[repository]
	return append([]string(nil), allowed...), ok, nil
}

func (s Store) Add(repository, context string) error {
	if err := validateEntry(repository, context); err != nil {
		return err
	}
	cfg, err := s.load()
	if err != nil {
		return err
	}
	cfg.Repositories[repository] = addContext(cfg.Repositories[repository], context)
	return s.save(cfg)
}

func (s Store) Entries() (map[string][]string, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, err
	}
	entries := make(map[string][]string, len(cfg.Repositories))
	for repository, allowed := range cfg.Repositories {
		entries[repository] = append([]string(nil), allowed...)
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
	var raw rawConfig
	if err := decoder.Decode(&raw); err != nil {
		return config{}, fmt.Errorf("decode %s: %w", s.path, err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return config{}, fmt.Errorf("decode %s: %w", s.path, err)
	}
	cfg, err := decodeConfig(raw)
	if err != nil {
		return config{}, fmt.Errorf("decode %s: %w", s.path, err)
	}
	return cfg, nil
}

func decodeConfig(raw rawConfig) (config, error) {
	switch raw.Version {
	case legacyVersion:
		return decodeLegacyConfig(raw.Repositories)
	case currentVersion:
		return decodeCurrentConfig(raw.Repositories)
	default:
		return config{}, fmt.Errorf("unsupported mapping configuration version %d", raw.Version)
	}
}

func decodeLegacyConfig(data json.RawMessage) (config, error) {
	legacy := make(map[string]string)
	if err := decodeRepositories(data, &legacy); err != nil {
		return config{}, err
	}
	repositories := make(map[string][]string, len(legacy))
	for repository, context := range legacy {
		repositories[repository] = []string{context}
	}
	return validatedConfig(repositories)
}

func decodeCurrentConfig(data json.RawMessage) (config, error) {
	repositories := make(map[string][]string)
	if err := decodeRepositories(data, &repositories); err != nil {
		return config{}, err
	}
	return validatedConfig(repositories)
}

func decodeRepositories(data json.RawMessage, target any) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	return json.Unmarshal(data, target)
}

func validatedConfig(repositories map[string][]string) (config, error) {
	for repository, allowed := range repositories {
		if err := validateRepository(repository); err != nil {
			return config{}, fmt.Errorf("invalid repository mapping: %w", err)
		}
		if len(allowed) == 0 {
			return config{}, fmt.Errorf("invalid repository mapping: repository %q has no contexts", repository)
		}
		for _, context := range allowed {
			if err := contexts.ValidateName(context); err != nil {
				return config{}, fmt.Errorf("invalid repository mapping: %w", err)
			}
		}
		repositories[repository] = uniqueSorted(allowed)
	}
	return config{Version: currentVersion, Repositories: repositories}, nil
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
	return config{Version: currentVersion, Repositories: make(map[string][]string)}
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
	if err := validateRepository(repository); err != nil {
		return err
	}
	return contexts.ValidateName(context)
}

func validateRepository(repository string) error {
	if !filepath.IsAbs(repository) || filepath.Clean(repository) != repository {
		return fmt.Errorf("repository path %q must be absolute and clean", repository)
	}
	return nil
}

func addContext(allowed []string, context string) []string {
	return uniqueSorted(append(append([]string(nil), allowed...), context))
}

func uniqueSorted(contexts []string) []string {
	sort.Strings(contexts)
	result := contexts[:0]
	for _, context := range contexts {
		if len(result) == 0 || result[len(result)-1] != context {
			result = append(result, context)
		}
	}
	return result
}
