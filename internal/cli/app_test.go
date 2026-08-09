package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"sesa/internal/codex"
	"sesa/internal/contexts"
	"sesa/internal/mappings"
	"sesa/internal/protocol"
	"sesa/internal/repository"
	"sesa/internal/vscode"
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

type fakeCodeRunner struct {
	home        string
	userDataDir string
	target      string
	checkErr    error
	err         error
}

func (f *fakeCodeRunner) Check() error {
	return f.checkErr
}

func (f *fakeCodeRunner) Run(home, userDataDir, target string) error {
	f.home = home
	f.userDataDir = userDataDir
	f.target = target
	return f.err
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
		userConfigDir:  func() (string, error) { return configDir, nil },
		workingDir:     func() (string, error) { return "/project", nil },
		repositoryRoot: func(string) (string, error) { return "", repository.ErrNotRepository },
		codex:          runner,
		code:           &fakeCodeRunner{},
		stdin:          strings.NewReader(""),
		stdout:         stdout,
		stderr:         stderr,
	}, stdout, stderr
}

func TestCodeLaunchesIsolatedVSCodeInstance(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	if err := store.Ensure("personal"); err != nil {
		t.Fatal(err)
	}
	code := &fakeCodeRunner{}
	app, _, stderr := testApp(configDir, &fakeRunner{})
	app.code = code
	app.workingDir = func() (string, error) { return "/repos/project", nil }

	if got := app.Run([]string{"code", "personal", "."}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	if code.home != store.Home("personal") {
		t.Fatalf("CODEX_HOME = %q, want %q", code.home, store.Home("personal"))
	}
	if code.userDataDir != store.VSCodeUserData("personal") {
		t.Fatalf("user data dir = %q, want %q", code.userDataDir, store.VSCodeUserData("personal"))
	}
	if code.target != "/repos/project" {
		t.Fatalf("target = %q, want /repos/project", code.target)
	}
	if !strings.Contains(stderr.String(), "Sesa VS Code context: PERSONAL") {
		t.Fatalf("stderr = %q, want context banner", stderr.String())
	}
}

func TestCodeRequiresExistingContext(t *testing.T) {
	code := &fakeCodeRunner{}
	app, _, stderr := testApp(t.TempDir(), &fakeRunner{})
	app.code = code

	if got := app.Run([]string{"code", "missing"}); got != 1 {
		t.Fatalf("Run() exit code = %d, want 1", got)
	}
	if code.home != "" {
		t.Fatal("VS Code ran for an unknown context")
	}
	if !strings.Contains(stderr.String(), "does not exist") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCodeUsesRepositoryMapping(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	if err := store.Ensure("work"); err != nil {
		t.Fatal(err)
	}
	if err := mappings.New(configDir).Set("/repos/project", "work"); err != nil {
		t.Fatal(err)
	}
	code := &fakeCodeRunner{}
	app, _, _ := testApp(configDir, &fakeRunner{})
	app.code = code
	app.workingDir = func() (string, error) { return "/repos/project", nil }
	app.repositoryRoot = func(directory string) (string, error) {
		if directory != "/repos/project" {
			t.Fatalf("repositoryRoot() directory = %q", directory)
		}
		return "/repos/project", nil
	}

	if got := app.Run([]string{"code"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	if code.home != store.Home("work") {
		t.Fatalf("CODEX_HOME = %q, want %q", code.home, store.Home("work"))
	}
}

func TestCodeMismatchRequiresConfirmation(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	for _, name := range []string{"personal", "work"} {
		if err := store.Ensure(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := mappings.New(configDir).Set("/repos/project", "work"); err != nil {
		t.Fatal(err)
	}
	code := &fakeCodeRunner{}
	app, _, stderr := testApp(configDir, &fakeRunner{})
	app.code = code
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }
	app.stdin = strings.NewReader("no\n")

	if got := app.Run([]string{"code", "personal"}); got != 1 {
		t.Fatalf("Run() exit code = %d, want 1", got)
	}
	if code.home != "" {
		t.Fatal("VS Code ran after mismatch was declined")
	}
	if !strings.Contains(stderr.String(), "mapped to WORK, but PERSONAL was requested") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCodeMismatchCanBeExplicitlyAllowed(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	for _, name := range []string{"personal", "work"} {
		if err := store.Ensure(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := mappings.New(configDir).Set("/repos/project", "work"); err != nil {
		t.Fatal(err)
	}
	code := &fakeCodeRunner{}
	app, _, _ := testApp(configDir, &fakeRunner{})
	app.code = code
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }

	if got := app.Run([]string{"code", "personal", "--allow-mismatch"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	if code.home != store.Home("personal") {
		t.Fatalf("CODEX_HOME = %q, want %q", code.home, store.Home("personal"))
	}
}

func TestCodeReportsMissingExecutable(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	if err := store.Ensure("work"); err != nil {
		t.Fatal(err)
	}
	app, _, stderr := testApp(configDir, &fakeRunner{})
	app.code = &fakeCodeRunner{checkErr: vscode.ErrNotFound}

	if got := app.Run([]string{"code", "work"}); got != 127 {
		t.Fatalf("Run() exit code = %d, want 127", got)
	}
	if !strings.Contains(stderr.String(), "code executable not found in PATH") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(store.VSCodeUserData("work")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("VS Code user data was created despite missing executable: %v", err)
	}
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
		"✓ VS Code shell command found",
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

func TestDoctorReportsMissingVSCodeShellCommand(t *testing.T) {
	app, stdout, _ := testApp(t.TempDir(), &fakeRunner{})
	app.code = &fakeCodeRunner{checkErr: vscode.ErrNotFound}
	if got := app.Run([]string{"doctor"}); got != 1 {
		t.Fatalf("Run() exit code = %d, want 1", got)
	}
	if !strings.Contains(stdout.String(), "✗ VS Code shell command: code executable not found in PATH") {
		t.Fatalf("doctor output = %q", stdout.String())
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

func TestRepositoryMappingLifecycleAndMappedRun(t *testing.T) {
	configDir := t.TempDir()
	if err := contexts.New(configDir).Ensure("work"); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	app, stdout, _ := testApp(configDir, runner)
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }

	if got := app.Run([]string{"link", "work"}); got != 0 {
		t.Fatalf("link exit code = %d", got)
	}
	if got := app.Run([]string{"current"}); got != 0 {
		t.Fatalf("current exit code = %d", got)
	}
	if got := app.Run([]string{"run"}); got != 0 {
		t.Fatalf("mapped run exit code = %d", got)
	}
	if runner.home != contexts.New(configDir).Home("work") {
		t.Fatalf("mapped run home = %q", runner.home)
	}
	if !strings.Contains(stdout.String(), "WORK\n") {
		t.Fatalf("output = %q, want mapped context", stdout.String())
	}
	if got := app.Run([]string{"unlink"}); got != 0 {
		t.Fatalf("unlink exit code = %d", got)
	}
	if _, ok, err := mappings.New(configDir).Get("/repos/project"); err != nil || ok {
		t.Fatalf("mapping remains after unlink: ok=%t err=%v", ok, err)
	}
}

func TestCurrentJSONReportsMappedRepository(t *testing.T) {
	configDir := t.TempDir()
	if err := mappings.New(configDir).Set("/repos/project", "work"); err != nil {
		t.Fatal(err)
	}
	app, stdout, stderr := testApp(configDir, &fakeRunner{})
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }

	if got := app.Run([]string{"current", "--json"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	var response protocol.CurrentRepository
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output=%q", err, stdout.String())
	}
	if response.ProtocolVersion != 1 || response.Repository != "/repos/project" || !response.Mapped {
		t.Fatalf("response = %#v", response)
	}
	if response.Context == nil || *response.Context != "work" {
		t.Fatalf("context = %v, want work", response.Context)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "\n  \"protocolVersion\": 1,") {
		t.Fatalf("stdout is not indented JSON: %q", stdout.String())
	}
}

func TestCurrentJSONReportsUnmappedRepositoryAsState(t *testing.T) {
	app, stdout, stderr := testApp(t.TempDir(), &fakeRunner{})
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }

	if got := app.Run([]string{"current", "--json"}); got != 0 {
		t.Fatalf("Run() exit code = %d, want 0", got)
	}
	var response protocol.CurrentRepository
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; output=%q", err, stdout.String())
	}
	if response.ProtocolVersion != 1 || response.Repository != "/repos/project" || response.Mapped || response.Context != nil {
		t.Fatalf("response = %#v", response)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestLinkRequiresExistingContext(t *testing.T) {
	app, _, stderr := testApp(t.TempDir(), &fakeRunner{})
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }
	if got := app.Run([]string{"link", "missing"}); got != 1 {
		t.Fatalf("link exit code = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "does not exist") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestMappedRunRequiresMapping(t *testing.T) {
	app, _, stderr := testApp(t.TempDir(), &fakeRunner{})
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }
	if got := app.Run([]string{"run"}); got != 1 {
		t.Fatalf("run exit code = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "not mapped") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunMismatchRequiresConfirmation(t *testing.T) {
	configDir := t.TempDir()
	store := contexts.New(configDir)
	for _, name := range []string{"personal", "work"} {
		if err := store.Ensure(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := mappings.New(configDir).Set("/repos/project", "work"); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	app, _, stderr := testApp(configDir, runner)
	app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }
	app.stdin = strings.NewReader("no\n")

	if got := app.Run([]string{"run", "personal"}); got != 1 {
		t.Fatalf("run exit code = %d, want 1", got)
	}
	if runner.home != "" {
		t.Fatal("Codex ran after mismatch was declined")
	}
	if !strings.Contains(stderr.String(), "mapped to WORK, but PERSONAL was requested") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunMismatchCanBeConfirmedOrExplicitlyAllowed(t *testing.T) {
	for _, tt := range []struct {
		name  string
		args  []string
		input string
	}{
		{name: "interactive confirmation", args: []string{"run", "personal"}, input: "yes\n"},
		{name: "explicit flag", args: []string{"run", "personal", "--allow-mismatch"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			store := contexts.New(configDir)
			for _, name := range []string{"personal", "work"} {
				if err := store.Ensure(name); err != nil {
					t.Fatal(err)
				}
			}
			if err := mappings.New(configDir).Set("/repos/project", "work"); err != nil {
				t.Fatal(err)
			}
			runner := &fakeRunner{}
			app, _, _ := testApp(configDir, runner)
			app.repositoryRoot = func(string) (string, error) { return "/repos/project", nil }
			app.stdin = strings.NewReader(tt.input)
			if got := app.Run(tt.args); got != 0 {
				t.Fatalf("run exit code = %d, want 0", got)
			}
			if runner.home != store.Home("personal") {
				t.Fatalf("runner home = %q", runner.home)
			}
		})
	}
}
