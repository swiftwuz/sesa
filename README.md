# Sesa

Switch safely between isolated Codex accounts.

Sesa is an open-source launcher for developers who use separate personal and
work Codex accounts. Each named context gets its own `CODEX_HOME`, while the
official Codex client continues to own login and credential management.

Sesa is unofficial and is not affiliated with OpenAI. It is not intended for
account pooling, credential sharing, quota circumvention, or bypassing rate
limits. See [SECURITY.md](SECURITY.md) for the project's security invariants.

## Authentication-isolation spike

The current implementation deliberately uses only the Go standard library. It
supports:

```console
sesa login personal
sesa login work
sesa run personal
sesa run work
sesa run work -- -C /path/to/project
```

On macOS, context homes are stored beneath:

```text
~/Library/Application Support/sesa/contexts/<context>/codex
```

Equivalent user configuration directories are used on Linux and Windows.

## Build and test

```console
go test ./...
go build -o sesa .
```

## Manual isolation proof

1. Build Sesa and log `personal` and `work` into two different official Codex
   accounts.
2. Run each context and confirm the expected identity using the official Codex
   interface.
3. Exit both processes, restart them through Sesa, and confirm both sessions
   persist independently.
4. Log out of one isolated context using `sesa run <context> -- logout`.
5. Confirm the other context remains authenticated.

Do not inspect or copy either context's `auth.json` during this test. The proof
is based on externally observable Codex behavior, not credential contents.
