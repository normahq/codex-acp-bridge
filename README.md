# codex-acp-bridge — archived

**Archived and read-only.** The final legacy Go/GitHub release is `v1.10.1`.
This repository receives no further synchronization or releases.

The project is maintained at [baldaworks/codex-acp](https://github.com/baldaworks/codex-acp).
This repository preserves the original Go module identity, command/import
paths, Git history and historical GitHub releases. From `v1.9.3` onward it
contains thin adapters to a pinned canonical release. The MIT license remains.

## Installation

Prefer the canonical executable for new installations:

```bash
go install github.com/baldaworks/codex-acp/cmd/codex-acp@latest
npx -y codex-acp@latest
```

Existing installations can retain their paths:

```bash
go install github.com/normahq/codex-acp-bridge/cmd/codex-acp-bridge@latest
npx -y @normahq/codex-acp-bridge@latest
npm install -g @normahq/codex-acp-bridge@latest
codex-acp-bridge version
codex-acp-bridge
```

The old Go path stays at `v1.10.1`, including `@latest`. Use `@v1.10.1`
for pinned Go installation. The deprecated npm alias is published from the
canonical repository and continues to follow its releases; use `@1.10.1`
for pinned npm execution.
Go executables install into `GOBIN` or `$(go env GOPATH)/bin`; include it in `PATH`.
Codex CLI and host authentication remain required for sessions. Run the command's
`login` subcommand or `codex login` to authenticate.

The legacy npm package provides both command names through one launcher and
uses the five shared `@baldaworks/codex-acp-*` native packages. Historical npm
versions and their old scoped binary dependencies remain available. npm
publication occurs only in the canonical repository.

## Go API compatibility

Existing code may continue importing
`github.com/normahq/codex-acp-bridge/pkg/cobracmd`. Its exported `New()` and
`Command()` constructors return the canonical `*cobra.Command`. The original
`cmd/codex-acp-bridge/cmd` import path also retains `Command()`. No consumer
`replace` directive is required. This module keeps its original `module` path
and pins `github.com/baldaworks/codex-acp` at the final legacy release `v1.10.1`.

New code should import `github.com/baldaworks/codex-acp/pkg/cobracmd` directly.
Both entrypoints use the canonical flags, command help and default agent name
`codex-acp`; use `--name` for a custom identity. ACP wire metadata keys retain
the historical `codex-acp-bridge/*` prefix and existing protocol contracts.

## Releases and contributions

All ongoing development, Go releases, native builds and npm publication belong
to [baldaworks/codex-acp](https://github.com/baldaworks/codex-acp).
The synchronization workflow has been retired. Legacy Go tags and GitHub
releases end at `v1.10.1`; use the canonical paths for future Go updates.

Existing Git history, tags and archive URLs remain available. Legacy GitHub
archives retain their `codex-acp-bridge-*` names and contain the matching
canonical native binary bytes. The deprecated npm alias remains available
through canonical publishing, independently of this archived repository.

File issues and implementation PRs at https://github.com/baldaworks/codex-acp.
See [usage](docs/usage.md), [JSON API](docs/json-api.md),
[archive policy](docs/releasing.md) and the
[full migration policy](https://github.com/baldaworks/codex-acp/blob/main/docs/migration.md).
