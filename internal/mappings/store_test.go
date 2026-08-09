package mappings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAddGetAndRemove(t *testing.T) {
	configDir := t.TempDir()
	store := New(configDir)
	repository := filepath.Join(t.TempDir(), "project")

	for _, context := range []string{"work", "personal", "work"} {
		if err := store.Add(repository, context); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}
	got, ok, err := store.Get(repository)
	if err != nil || !ok || !reflect.DeepEqual(got, []string{"personal", "work"}) {
		t.Fatalf("Get() = %q, %t, %v; want personal and work", got, ok, err)
	}
	removed, err := store.Remove(repository)
	if err != nil || !removed {
		t.Fatalf("Remove() = %t, %v; want true, nil", removed, err)
	}
	if _, ok, err := store.Get(repository); err != nil || ok {
		t.Fatalf("Get() after Remove() = ok %t, err %v", ok, err)
	}

	info, err := os.Stat(filepath.Join(configDir, "sesa", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestMissingConfigurationIsEmpty(t *testing.T) {
	if _, ok, err := New(t.TempDir()).Get("/project"); err != nil || ok {
		t.Fatalf("Get() = ok %t, err %v; want false, nil", ok, err)
	}
}

func TestAddRejectsInvalidEntries(t *testing.T) {
	store := New(t.TempDir())
	separator := string(filepath.Separator)
	for _, tt := range []struct {
		repository string
		context    string
	}{
		{repository: "relative/project", context: "work"},
		{repository: separator + "project" + separator + ".." + separator + "project", context: "work"},
		{repository: filepath.Join(string(filepath.Separator), "project"), context: "../work"},
	} {
		if err := store.Add(tt.repository, tt.context); err == nil {
			t.Errorf("Add(%q, %q) unexpectedly succeeded", tt.repository, tt.context)
		}
	}
}

func TestRejectsMalformedOrUnsupportedConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed", content: "{"},
		{name: "unknown field", content: `{"version":1,"repositories":{},"secret":"no"}`},
		{name: "unsupported version", content: `{"version":3,"repositories":{}}`},
		{name: "empty context set", content: `{"version":2,"repositories":{"/project":[]}}`},
		{name: "multiple values", content: `{"version":1,"repositories":{}} {}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			path := filepath.Join(configDir, "sesa", "config.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := New(configDir).Get("/project"); err == nil {
				t.Fatal("Get() unexpectedly accepted invalid configuration")
			}
		})
	}
}

func TestLoadsAndMigratesVersionOneConfiguration(t *testing.T) {
	configDir := t.TempDir()
	path := filepath.Join(configDir, "sesa", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := `{"version":1,"repositories":{"/project":"personal"}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New(configDir)
	allowed, ok, err := store.Get("/project")
	if err != nil || !ok || !reflect.DeepEqual(allowed, []string{"personal"}) {
		t.Fatalf("Get() = %q, %t, %v", allowed, ok, err)
	}
	if err := store.Add("/project", "work"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var migrated config
	if err := json.Unmarshal(data, &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.Version != 2 || !reflect.DeepEqual(migrated.Repositories["/project"], []string{"personal", "work"}) {
		t.Fatalf("migrated config = %#v", migrated)
	}
}
