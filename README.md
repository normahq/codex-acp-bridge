# codex-acp-bridge

[![test](https://github.com/normahq/codex-acp-bridge/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/normahq/codex-acp-bridge/actions/workflows/test.yml)
[![lint](https://github.com/normahq/codex-acp-bridge/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/normahq/codex-acp-bridge/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/normahq/codex-acp-bridge)](https://goreportcard.com/report/github.com/normahq/codex-acp-bridge)
[![coverage](https://codecov.io/gh/normahq/codex-acp-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/normahq/codex-acp-bridge)
[![npm version](https://img.shields.io/npm/v/%40normahq%2Fcodex-acp-bridge)](https://www.npmjs.com/package/@normahq/codex-acp-bridge)

**Turn Codex into a full-scale ACP server.**

Zero deps. No OpenAI API keys. Uses your existing Codex subscription.

`codex-acp-bridge` exposes `codex app-server` as ACP over stdio, so ACP clients can run Codex with native session and model flows.

## Requirements

- `codex` CLI in `PATH`.
- Authenticated Codex session on the machine running the bridge.
- Active Codex subscription.

## 60-Second Quickstart

Run the bridge:

```bash
npx -y @normahq/codex-acp-bridge@latest
```

Inspect ACP initialize/session payloads:

```bash
npx -y @normahq/acp-dump -- npx -y @normahq/codex-acp-bridge@latest
npx -y @normahq/acp-dump --json -- npx -y @normahq/codex-acp-bridge@latest
```

Start an interactive ACP session:

```bash
npx -y @normahq/acp-repl -- npx -y @normahq/codex-acp-bridge@latest
```

## Installation

Global install (npm):

```bash
npm install -g @normahq/codex-acp-bridge@latest
```

One-off run with npx:

```bash
npx -y @normahq/codex-acp-bridge@latest
```

## GitHub

- Repository: https://github.com/normahq/codex-acp-bridge
- Issues: https://github.com/normahq/codex-acp-bridge/issues
- Releases: https://github.com/normahq/codex-acp-bridge/releases
- Test workflow: https://github.com/normahq/codex-acp-bridge/actions/workflows/test.yml
- Lint workflow: https://github.com/normahq/codex-acp-bridge/actions/workflows/lint.yml

## What "full-scale ACP server" means

- Exposes Codex app-server as ACP over stdio.
- Supports ACP `session/resume` via app-server thread resume.
- Uses app-server `thread.id` as the ACP `sessionId` for direct resume mapping.
- Populates ACP `session/new.models` from `model/list`.
- Supports ACP `session/set_model` and `session/set_mode`.
- Supports ACP `session/new.configOptions` and `session/set_config_option` for model-advertised reasoning effort values.
- Supports text and image prompt blocks.
- Supports per-session MCP servers from ACP `mcpServers` (`stdio`, `http`; rejects `sse`) using merge contract (same-name overrides, other configured servers remain active).
- Supports strict session metadata mapping via `session/new._meta.codex`.
- Returns backend-native durable ACP session IDs from `session/new`.

## Usage

Run the bridge:

```bash
codex-acp-bridge
codex-acp-bridge --name team-codex
codex-acp-bridge --message-streaming
codex-acp-bridge --reasoning-thoughts=both
codex-acp-bridge --reasoning-streaming=false
codex-acp-bridge --debug
```

Flags:

- `--name`: ACP agent name exposed via `initialize.agentInfo.name`.
  Default: `norma-codex-acp-bridge`.
- `--message-streaming`: Stream app-server `agentMessage` deltas as ACP `agent_message_chunk` updates.
  Default: `false`.
- `--reasoning-streaming`: Stream app-server reasoning deltas live when enabled; when disabled, emit final thought chunks from completed reasoning items only.
  Default: `true`.
- `--reasoning-thoughts`: Reasoning lane to project as ACP thoughts: `off`, `summary`, `content`, or `both`.
  Default: `summary`.
- `--debug`: Enable debug logging.

If tools are installed globally:

```bash
acp-dump -- codex-acp-bridge
acp-repl -- codex-acp-bridge
```

Documentation:

- [Usage](https://github.com/normahq/codex-acp-bridge/blob/main/docs/usage.md)
- [JSON API](https://github.com/normahq/codex-acp-bridge/blob/main/docs/json-api.md)
