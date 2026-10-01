# Project Instructions for AI Agents

This file provides instructions and context for AI coding agents working on this project.

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


## Build & Test

```bash
go test -race ./...
go tool golangci-lint run
go build ./cmd/codex-acp-bridge
```

## Architecture and Conventions

This module preserves `github.com/normahq/codex-acp-bridge` installation and
public import paths. Its constructors delegate to the pinned canonical
`github.com/baldaworks/codex-acp/pkg/cobracmd`; there is no copied bridge runtime.
Use idiomatic Go, Conventional Commits and the canonical project's Beads tracker.

This repository is archived and read-only, frozen at Go/GitHub release v1.10.1.
Synchronization and release automation are retired; do all new work in the
canonical repository.
Never commit a Go replace directive or overwrite historical tags/assets.
The old repository does not publish npm. See `AGENTS.md`, `docs/releasing.md`
and the canonical repository for runtime changes, ACP contracts and migration docs.
