# Sesa development guidance

- Keep every Go function at or below Sonar cognitive complexity 15.
- Split command dispatch from command-specific logic before adding nested
  branches to an existing function.
- Before handing off changes, run `gofmt`, `go test ./...`,
  `go test -race ./...`, and `go vet ./...`, then review newly changed functions
  for Sonar `go:S3776` risk.
- Preserve the credential and identity-separation invariants in `SECURITY.md`.
