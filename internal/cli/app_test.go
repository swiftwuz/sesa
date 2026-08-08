package cli

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"sesa/internal/codex"
	"sesa/internal/contexts"
)

type fakeRunner struct {
	home           string
	args           []string
	checkErr       error
	version        string
	versionErr     error
	loginStatusErr map[string]error
	err            error
}

func (f *fakeRunner) Check() error {
	return f.checkErr
}

func (f *fakeRunner) Version() (string, error) {
	if f.version == "" {
		f.version = "codex-cli test"
	}
	return f.version, f.versionErr
}

func (f *fakeRunner) LoginStatus(home string) error {
	return f.loginStatusErr[home]
}

func (f *fakeRunner) Run(home string, args []string) error {
	f.home = home
	f.args = args
	return f.err
}

func testApp(configDir string, runner *fakeRunner) (App, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return App{
		userConfigDir: func() (string, error) { return configDir, nil },
		codex:         runner,
		stdout:        stdout,
		stderr:        stderr,
	}, stdout, stderr
}

func TestRunLaunchesCodexWithIsolatedHome(t *testing.T) {
	configDir := t.TempDir()
	runner := &fakeRunner{}
	app, _, stderr := testApp(configDir, runner)

	if got := app.Run([]string{"login", "personal"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	wantHome := contexts.New(configDir).Home("personal")
	if runner.home != wantHome || !reflect.DeepEqual(runner.args, []string{"login"}) {
		t.Fatalf("runner received home %q and args %q", runner.home, runner.args)
	}
	if !strings.Contains(stderr.String(), "Sesa context: PERSONAL") {
		t.Fatalf("stderr = %q, want context banner", stderr.String())
	}
}

func TestRunReportsMissingCodex(t *testing.T) {
	runner := &fakeRunner{checkErr: codex.ErrNotFound}
	app, _, stderr := testApp(t.TempDir(), runner)
	if got := app.Run([]string{"run", "work"}); got != 127 {
		t.Fatalf("Run() exit code = %d, want 127", got)
	}
	if !strings.Contains(stderr.String(), "codex executable not found in PATH") {
		t.Fatalf("stderr = %q, want missing Codex error", stderr.String())
	}
}

func TestStatusDoesNotCreateUnknownContext(t *testing.T) {
	configDir := t.TempDir()
	runner := &fakeRunner{}
	app, _, _ := testApp(configDir, runner)
	if got := app.Run([]string{"status", "missing"}); got != 1 {
		t.Fatalf("Run() exit code = %d, want 1", got)
	}
	exists, err := contexts.New(configDir).Exists("missing")
	if err != nil || exists {
		t.Fatalf("unknown context was created: exists=%t, err=%v", exists, err)
	}
	if runner.home != "" {
		t.Fatal("Codex ran for an unknown context")
	}
}

func TestList(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	for _, name := range []string{"work", "personal"} {
		if err := store.Ensure(name); err != nil {
			t.Fatal(err)
		}
	}
	app, stdout, _ := testApp(configDir, &fakeRunner{})
	if got := app.Run([]string{"list"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	if want := "personal\nwork\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestHelpDoesNotInspectConfiguration(t *testing.T) {
	stdout := &bytes.Buffer{}
	called := false
	app := App{
		userConfigDir: func() (string, error) {
			called = true
			return "", errors.New("unexpected")
		},
		codex:  &fakeRunner{},
		stdout: stdout,
		stderr: &bytes.Buffer{},
	}
	if got := app.Run([]string{"help"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	if called {
		t.Fatal("help inspected the user configuration directory")
	}
	if !strings.Contains(stdout.String(), "sesa login <context>") {
		t.Fatalf("stdout = %q, want help text", stdout.String())
	}
}

func TestListReturnsEmptyMessage(t *testing.T) {
	app, stdout, _ := testApp(t.TempDir(), &fakeRunner{})
	if got := app.Run([]string{"list"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	if stdout.String() != "No contexts found.\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDoctorReportsHealthyContexts(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	for _, name := range []string{"personal", "work"} {
		if err := store.Ensure(name); err != nil {
			t.Fatal(err)
		}
	}
	app, stdout, _ := testApp(configDir, &fakeRunner{version: "codex-cli 1.2.3"})
	if got := app.Run([]string{"doctor"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0; output:\n%s", got, stdout.String())
	}
	for _, wanted := range []string{
		"✓ Codex CLI found (codex-cli 1.2.3)",
		"✓ Context storage accessible",
		"✓ personal: isolated home",
		"✓ personal: authenticated",
		"✓ work: isolated home",
		"✓ work: authenticated",
	} {
		if !strings.Contains(stdout.String(), wanted) {
			t.Errorf("doctor output missing %q:\n%s", wanted, stdout.String())
		}
	}
}

func TestDoctorFailsWhenContextIsLoggedOut(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	if err := store.Ensure("personal"); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{loginStatusErr: map[string]error{store.Home("personal"): errors.New("not logged in")}}
	app, stdout, _ := testApp(configDir, runner)
	if got := app.Run([]string{"doctor"}); got != 1 {
		t.Fatalf("Run() exit code = %d, want 1", got)
	}
	if !strings.Contains(stdout.String(), "✗ personal: not authenticated") {
		t.Fatalf("doctor output = %q", stdout.String())
	}
}
