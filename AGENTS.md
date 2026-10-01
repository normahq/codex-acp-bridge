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

## Compatibility and Ownership

- Canonical implementation: `github.com/baldaworks/codex-acp`. This repository retains original module identity `github.com/normahq/codex-acp-bridge` and thin CLI/public API adapters.
- Preserve public `pkg/cobracmd.New`/`Command` signatures, command import/install paths, stdio, signal cancellation and failure exit status. Delegate to canonical code; do not copy its internal bridge implementation.
- Keep the exact canonical dependency pinned in go.mod. No committed replace directive. Canonical dependency versions and tags must exist before publishing this module.
- ACP transport/model/metadata behavior is maintained in the canonical repository. Preserve its wire compatibility through adapters.
- Update every affected doc: README, command README, usage/JSON API pointers, Go package doc.go, release sync instructions and agent guides.

## Release

- `sync-canonical-release.yml` follows published canonical `vX.Y.Z` releases hourly or by dispatch, using own-repository `GITHUB_TOKEN`. It tests, commits dependency changes and atomically creates an immutable matching tag.
- This repository does not publish npm or build an independent runtime. Canonical npm aliases share `@baldaworks/codex-acp-*` packages.
- Preserve all historical tags and release assets. Legacy archives reuse canonical native bytes under old names; retries verify rather than replace accepted assets.
- See `docs/releasing.md` and canonical migration/release docs. File implementation issues and PRs in baldaworks/codex-acp.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

Run Beads commands from the canonical baldaworks/codex-acp checkout; keep migration and compatibility tasks there rather than copying its database.

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --no-rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
