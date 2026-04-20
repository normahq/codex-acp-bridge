# codex-acp-bridge — AGENTS.md

## Development Standards

- Follow idiomatic Go and Google Go best practices.
- Prefer project-local tooling via `go tool ...` when available.
- Use Conventional Commits for all commits.
- Sync shared branches with merge (`git pull --no-rebase`), not rebase.

## Quality Gates (Required)

Run before submitting changes:

```bash
go test -race ./...
go tool golangci-lint run
```

## Logging Policy

- Allowed: `github.com/rs/zerolog`, `log/slog`.
- Disallowed: `logrus`, `zap`, direct standard `log` usage.
- Initialize logging through `internal/logging.Init()`.
- Prefer structured logging fields over formatted strings.

## Bridge Guardrails

- Keep ACP contract compatibility stable.
- Keep strict validation for `session/new._meta.codex`.
- Keep model handling ACP-native (`session/set_model`), not bridge-specific model CLI flags.
- Keep MCP transport constraints aligned with implementation (`stdio` and `http`, reject `sse`).

## Documentation

- Product/usage docs are rooted in `README.md`.
- Protocol details are in:
  - `docs/codex-acp-bridge.md`
  - `docs/codex-acp-bridge-json-api.md`

## Release

- Omnidist profile is authoritative (`.omnidist/omnidist.yaml`).
- Version source is Git tags (`version.source: git-tag`).
- Publish flow is tag-driven via GitHub Actions release workflow.
