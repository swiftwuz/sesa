package vscode

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckReportsMissingCode(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := (Runner{}).Check(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Check() error = %v, want ErrNotFound", err)
	}
}

func TestRunPassesIsolationArgumentsAndEnvironment(t *testing.T) {
	binDir := t.TempDir()
	output := filepath.Join(t.TempDir(), "invocation")
	stub := filepath.Join(binDir, "code")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$CODEX_HOME\" \"$@\" > \"$SESA_TEST_OUTPUT\"\n" +
		"printf 'code output'\n" +
		"printf 'code error' >&2\n"
	if err := os.WriteFile(stub, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("SESA_TEST_OUTPUT", output)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := (Runner{Stdout: stdout, Stderr: stderr}).Run("/contexts/personal/codex", "/contexts/personal/vscode", "/repos/project"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"/contexts/personal/codex",
		"--new-window",
		"--user-data-dir",
		"/contexts/personal/vscode",
		"/repos/project",
		"",
	}, "\n")
	if string(data) != want {
		t.Fatalf("invocation = %q, want %q", data, want)
	}
	if stdout.String() != "code output" || stderr.String() != "code error" {
		t.Fatalf("child output not forwarded: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
