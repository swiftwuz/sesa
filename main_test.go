package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want invocation
	}{
		{name: "help", args: []string{"help"}, want: invocation{action: "help"}},
		{name: "short help", args: []string{"-h"}, want: invocation{action: "help"}},
		{name: "long help", args: []string{"--help"}, want: invocation{action: "help"}},
		{name: "list", args: []string{"list"}, want: invocation{action: "list"}},
		{name: "login", args: []string{"login", "personal"}, want: invocation{action: "login", context: "personal", codexArgs: []string{"login"}}},
		{name: "status", args: []string{"status", "personal"}, want: invocation{action: "status", context: "personal", codexArgs: []string{"login", "status"}}},
		{name: "run", args: []string{"run", "work"}, want: invocation{action: "run", context: "work"}},
		{name: "run with Codex arguments", args: []string{"run", "work", "--", "-C", "/tmp/project"}, want: invocation{action: "run", context: "work", codexArgs: []string{"-C", "/tmp/project"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseArgsRejectsUnsafeOrAmbiguousInput(t *testing.T) {
	tests := [][]string{
		nil,
		{"help", "extra"},
		{"run"},
		{"list", "work"},
		{"run", "../work"},
		{"run", "Work"},
		{"run", "work", "-C", "/tmp/project"},
		{"login", "work", "extra"},
		{"status", "work", "extra"},
		{"unknown", "work"},
	}

	for _, args := range tests {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%q) unexpectedly succeeded", args)
		}
	}
}

func TestListContexts(t *testing.T) {
	configDir := t.TempDir()
	for _, context := range []string{"work", "personal"} {
		if err := os.MkdirAll(contextHome(configDir, context), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(configDir, "sesa", "contexts", "not_ready"), 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := listContexts(configDir)
	if err != nil {
		t.Fatalf("listContexts() error = %v", err)
	}
	want := []string{"personal", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listContexts() = %q, want %q", got, want)
	}
}

func TestListContextsReturnsEmptyWhenRootDoesNotExist(t *testing.T) {
	got, err := listContexts(t.TempDir())
	if err != nil {
		t.Fatalf("listContexts() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("listContexts() = %q, want no contexts", got)
	}
}

func TestContextHomesAreIsolated(t *testing.T) {
	base := filepath.Join("tmp", "config")
	personal := contextHome(base, "personal")
	work := contextHome(base, "work")

	if personal == work {
		t.Fatal("different contexts resolved to the same CODEX_HOME")
	}
	if want := filepath.Join(base, "sesa", "contexts", "personal", "codex"); personal != want {
		t.Fatalf("contextHome() = %q, want %q", personal, want)
	}
}

func TestWithEnvironmentReplacesCodexHome(t *testing.T) {
	got := withEnvironment([]string{"PATH=/bin", "CODEX_HOME=/old", "TERM=xterm"}, "CODEX_HOME", "/new")
	want := []string{"PATH=/bin", "TERM=xterm", "CODEX_HOME=/new"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("withEnvironment() = %q, want %q", got, want)
	}
}

func TestRunLaunchesCodexWithIsolatedHome(t *testing.T) {
	configDir := t.TempDir()
	var gotPath string
	var gotArgs, gotEnv []string

	exitCode := run(
		[]string{"login", "personal"},
		func() (string, error) { return configDir, nil },
		func(name string) (string, error) { return "/usr/local/bin/codex", nil },
		func(path string, args, env []string) error {
			gotPath, gotArgs, gotEnv = path, args, env
			return nil
		},
	)

	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if gotPath != "/usr/local/bin/codex" || !reflect.DeepEqual(gotArgs, []string{"login"}) {
		t.Fatalf("launcher received path %q and args %q", gotPath, gotArgs)
	}
	wantHome := "CODEX_HOME=" + contextHome(configDir, "personal")
	if !contains(gotEnv, wantHome) {
		t.Fatalf("launcher environment does not contain %q", wantHome)
	}
}

func TestRunReportsMissingCodex(t *testing.T) {
	exitCode := run(
		[]string{"run", "work"},
		func() (string, error) { return t.TempDir(), nil },
		func(name string) (string, error) { return "", errors.New("not found") },
		func(path string, args, env []string) error { return nil },
	)
	if exitCode != 127 {
		t.Fatalf("run() exit code = %d, want 127", exitCode)
	}
}

func TestStatusDoesNotCreateUnknownContext(t *testing.T) {
	configDir := t.TempDir()
	lookPathCalled := false
	exitCode := run(
		[]string{"status", "missing"},
		func() (string, error) { return configDir, nil },
		func(name string) (string, error) {
			lookPathCalled = true
			return "/usr/local/bin/codex", nil
		},
		func(path string, args, env []string) error { return nil },
	)
	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if lookPathCalled {
		t.Fatal("run() looked up Codex for an unknown context")
	}
	if exists, err := contextExists(contextHome(configDir, "missing")); err != nil || exists {
		t.Fatalf("unknown context was created: exists=%t, err=%v", exists, err)
	}
}

func contains(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}
