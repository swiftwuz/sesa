# Security policy

Sesa exists to keep a developer's personal and work Codex identities clearly
separated. It is not an account-pooling, credential-sharing, quota-circumvention,
or rate-limit-bypass tool.

Sesa is unofficial and is not affiliated with OpenAI.

## Security invariants

- Every named context uses a separate `CODEX_HOME`.
- The official Codex client owns browser login and credential lifecycle.
- Sesa must never read, parse, copy, snapshot, export, log, or rewrite
  `auth.json`, refresh tokens, or other Codex credentials.
- Sesa must not call private authentication endpoints or patch the official
  Codex CLI or extension.
- Switching contexts launches a new isolated process. Sesa must not mutate the
  identity of an active Codex or VS Code process.
- Human-readable CLI output and the future versioned JSON protocol remain
  separate interfaces.

Changes that weaken these invariants are out of scope, even when technically
possible.

## Local development

Do not place real Codex homes, credentials, tokens, or secret-bearing `.env`
files inside the repository. The `.gitignore` provides a backstop, but it is not
a substitute for reviewing staged changes before every commit.

Use synthetic fixtures for tests. Never use copied production authentication
files as fixtures.

## Reporting a vulnerability

Until a private reporting channel is published, do not open a public issue that
contains credentials, tokens, personal data, or an exploitable proof of concept.
Contact the repository owner privately and include only the minimum information
needed to reproduce the issue safely.
