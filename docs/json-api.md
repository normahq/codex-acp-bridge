# Legacy JSON API compatibility

The Go adapters and legacy npm command use the canonical ACP projection.
The current specification and code-first schema inventory are maintained at
[baldaworks/codex-acp/docs/json-api.md](https://github.com/baldaworks/codex-acp/blob/main/docs/json-api.md).
The bridge implementation and protocol tests are in the canonical repository.

Migration preserves `codex-acp-bridge/*` wire metadata keys, deterministic
reasoning identifiers, strict `session/new._meta.codex` validation, ACP-native
model configuration and existing stdio/http MCP transport constraints.
Public legacy `pkg/cobracmd.New` and `Command` return the canonical Cobra command.
There is no separate legacy protocol implementation to maintain.

See [usage](usage.md) and the
[full migration policy](https://github.com/baldaworks/codex-acp/blob/main/docs/migration.md).
