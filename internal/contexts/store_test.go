package contexts

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateName(t *testing.T) {
	valid := []string{"personal", "work", "work-2", "a"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) error = %v", name, err)
		}
	}

	invalid := []string{"", "Work", "2work", "../work", "work_personal", "a-very-long-context-name-that-exceeds-32"}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) unexpectedly succeeded", name)
		}
	}
}

func TestHomesAreIsolated(t *testing.T) {
	base := filepath.Join("tmp", "config")
	store := New(base)
	personal := store.Home("personal")
	work := store.Home("work")

	if personal == work {
		t.Fatal("different contexts resolved to the same CODEX_HOME")
	}
	if want := filepath.Join(base, "sesa", "contexts", "personal", "codex"); personal != want {
		t.Fatalf("Home() = %q, want %q", personal, want)
	}
}

func TestEnsureAndExists(t *testing.T) {
	store := New(t.TempDir())
	exists, err := store.Exists("personal")
	if err != nil || exists {
		t.Fatalf("Exists() before Ensure() = %t, %v", exists, err)
	}
	if err := store.Ensure("personal"); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	exists, err = store.Exists("personal")
	if err != nil || !exists {
		t.Fatalf("Exists() after Ensure() = %t, %v", exists, err)
	}
}

func TestFilesystemOperationsRejectInvalidNames(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Ensure("../outside"); err == nil {
		t.Fatal("Ensure() accepted a path-traversal context name")
	}
	if exists, err := store.Exists("../outside"); err == nil || exists {
		t.Fatalf("Exists() accepted a path-traversal context name: exists=%t, err=%v", exists, err)
	}
}

func TestList(t *testing.T) {
	store := New(t.TempDir())
	for _, name := range []string{"work", "personal"} {
		if err := store.Ensure(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(store.Home("not_ready")), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []string{"personal", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %q, want %q", got, want)
	}
}

func TestListReturnsEmptyWhenRootDoesNotExist(t *testing.T) {
	got, err := New(t.TempDir()).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("List() = %q, want no contexts", got)
	}
}

func TestCheckStorageAllowsMissingRoot(t *testing.T) {
	if err := New(t.TempDir()).CheckStorage(); err != nil {
		t.Fatalf("CheckStorage() error = %v", err)
	}
}

func TestInspectRejectsSymlinkedCodexHome(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Ensure("personal"); err != nil {
		t.Fatal(err)
	}
	home := store.Home("personal")
	if err := os.Remove(home); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), home); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	if err := store.Inspect("personal"); err == nil {
		t.Fatal("Inspect() accepted a symlinked Codex home")
	}
	if err := store.Ensure("personal"); err == nil {
		t.Fatal("Ensure() accepted a symlinked Codex home")
	}
	if exists, err := store.Exists("personal"); err == nil || exists {
		t.Fatalf("Exists() accepted a symlinked Codex home: exists=%t, err=%v", exists, err)
	}
}
