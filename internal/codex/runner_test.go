package codex

import (
	"errors"
	"reflect"
	"testing"
)

func TestWithEnvironmentReplacesCodexHome(t *testing.T) {
	got := WithEnvironment([]string{"PATH=/bin", "CODEX_HOME=/old", "TERM=xterm"}, "CODEX_HOME", "/new")
	want := []string{"PATH=/bin", "TERM=xterm", "CODEX_HOME=/new"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WithEnvironment() = %q, want %q", got, want)
	}
}

func TestCheckReportsMissingCodex(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := (Runner{}).Check(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Check() error = %v, want ErrNotFound", err)
	}
}
