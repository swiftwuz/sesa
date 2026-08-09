# Sesa

Switch safely between isolated Codex accounts.

Sesa is an open-source launcher for developers who use separate personal and
work Codex accounts. Each named context gets its own `CODEX_HOME`, while the
official Codex client continues to own login and credential management.

Sesa can associate multiple repositories with a context, and a repository can
allow multiple contexts. Switching always launches a new isolated Codex or VS
Code process; Sesa never mutates the identity of an active process.

Sesa is unofficial and is not affiliated with OpenAI. It is not intended for
account pooling, credential sharing, quota circumvention, or bypassing rate
limits. See [SECURITY.md](SECURITY.md) for the project's security invariants.

## Requirements

- Go 1.25.6 or newer
- The official Codex CLI available as `codex` in `PATH`
- Visual Studio Code's `code` shell command in `PATH` for editor isolation

## Install

Install directly from GitHub:

```sh
go install github.com/swiftwuz/sesa@latest
```

Ensure Go's binary directory is in `PATH`. It defaults to `$(go env GOPATH)/bin`.

For local development:

```sh
git clone https://github.com/swiftwuz/sesa.git
cd sesa
go install .
```

## Usage

Authenticate each context through the official Codex login flow:

```sh
sesa login personal
sesa login work
```

Launch the isolated Codex CLI or VS Code:

```sh
sesa run personal
sesa code work .
```

Allow contexts for the current Git repository:

```sh
sesa link personal
sesa link work
sesa current
```

When exactly one context is allowed, `sesa run` and `sesa code` select it
automatically. Specify the context when several are allowed.

Run `sesa help` for the complete command reference and `sesa doctor` for local
diagnostics.

## VS Code extension

The extension is not yet published. Build a local VSIX with:

```sh
cd vscode-extension
npm ci
npm run package
```

In VS Code, run **Extensions: Install from VSIX...** and select
`vscode-extension/dist/sesa-for-codex.vsix`.

## License

Sesa is available under the [MIT License](LICENSE).
