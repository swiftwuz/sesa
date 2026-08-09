# Sesa for Codex

Sesa for Codex shows which isolated Sesa context is active in a VS Code window
and warns when that context conflicts with the repository's mapping.

The extension is a thin client over the `sesa` CLI. It does not authenticate
users, inspect Codex credentials, or make account-selection decisions.

Sesa is unofficial and is not affiliated with OpenAI.

Click the Sesa status-bar item to choose an available context. For an unmapped
repository, the extension creates the mapping automatically. Replacing an
existing mapping requires confirmation, and changing identities always opens a
new isolated VS Code window.

## Development install

Build a local VSIX:

```sh
npm install
npm run package
```

In a VS Code window launched by `sesa code`, run **Extensions: Install from
VSIX...** and select `dist/sesa-for-codex.vsix`. Reopen that window through
Sesa so the extension receives its isolated context.
