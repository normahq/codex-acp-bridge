# Legacy usage

The legacy command and Go module forward to the canonical codex-acp
implementation. The legacy Go adapter is frozen at `v1.10.1`; the deprecated npm alias
continues to follow canonical releases. Current runtime options and examples
are maintained in
[canonical usage](https://github.com/baldaworks/codex-acp/blob/main/docs/usage.md).

```bash
npx -y @normahq/codex-acp-bridge@latest
go install github.com/normahq/codex-acp-bridge/cmd/codex-acp-bridge@latest
codex-acp-bridge
codex-acp-bridge --name team-codex
```

The npm package exports `codex-acp` and `codex-acp-bridge`, sharing one launcher.
The Go executable retains its old name and uses the pinned canonical command.
Both use canonical help, flags and default ACP identity `codex-acp`.

Codex authentication, session state, models, MCP transport and stdio semantics
remain native to the canonical implementation. Historical wire metadata names
remain compatible. See [README](../README.md), [JSON API](json-api.md),
[archive policy](releasing.md) and
[migration](https://github.com/baldaworks/codex-acp/blob/main/docs/migration.md).
