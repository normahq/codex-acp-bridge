# codex-acp-bridge

`codex-acp-bridge` runs the Codex bridge backend and exposes it as an ACP agent over stdio.

## Features

- Exposes Codex app-server as ACP over stdio.
- Supports ACP `session/new.models` from `model/list`.
- Supports ACP `session/set_model` and `session/set_mode`.
- Supports prompt text and image blocks.
- Supports per-session MCP servers from ACP `mcpServers` (`stdio`, `http`; rejects `sse`).
- Supports strict session metadata mapping via `session/new._meta.codex` (documented in usage docs).

## Installation

Global install (npm):

```bash
npm install -g @normahq/codex-acp-bridge@latest
```

One-off run with npx:

```bash
npx -y @normahq/codex-acp-bridge@latest
```

## Usage

Run the bridge:

```bash
codex-acp-bridge
codex-acp-bridge --name team-codex
codex-acp-bridge --debug
```

Flags:

- `--name`: ACP agent name exposed via `initialize.agentInfo.name`.
  Default: `norma-codex-acp-bridge`.
- `--debug`: Enable debug logging.

Documentation:

- [Usage](docs/usage.md)
- [JSON API](docs/json-api.md)
