# codex-acp-bridge

`codex-acp-bridge` exposes `codex app-server` as an ACP agent over stdio.

## Usage

Run directly with npx:

```bash
npx -y @normahq/codex-acp-bridge@latest
```

Or install globally:

```bash
npm install -g @normahq/codex-acp-bridge@latest
codex-acp-bridge
```

Common flags:

- `--name` ACP agent name exposed in `initialize.agentInfo.name` (default: `norma-codex-acp-bridge`)
- `--debug` enable bridge debug logging

## ACP behavior

- ACP `session/new.models` is populated from Codex `model/list` when available.
- ACP `session/set_model` updates model used by subsequent turns.
- ACP `session/set_mode` is stored in ACP session state only.
- Prompt content supports text + image blocks; audio is rejected as `invalid_params`.
- ACP `session/new.mcpServers` supports `stdio` and `http` transports (`sse` is rejected).

Session-level Codex defaults are configured through `session/new._meta.codex`.

Mapping details and examples:
- [cmd/codex-acp-bridge/README.md](cmd/codex-acp-bridge/README.md)
- [docs/codex-acp-bridge.md](docs/codex-acp-bridge.md)
- [docs/codex-acp-bridge-json-api.md](docs/codex-acp-bridge-json-api.md)

## Development

```bash
go test ./...
go test -tags='integration,codex' -count=1 ./internal/apps/codexacpbridge
```

## License

MIT
