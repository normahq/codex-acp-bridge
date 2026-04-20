# codex-acp-bridge

[![test](https://github.com/normahq/codex-acp-bridge/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/normahq/codex-acp-bridge/actions/workflows/test.yml)
[![lint](https://github.com/normahq/codex-acp-bridge/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/normahq/codex-acp-bridge/actions/workflows/lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/normahq/codex-acp-bridge)](https://goreportcard.com/report/github.com/normahq/codex-acp-bridge)
[![coverage](https://codecov.io/gh/normahq/codex-acp-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/normahq/codex-acp-bridge)
[![npm version](https://img.shields.io/npm/v/%40normahq%2Fcodex-acp-bridge)](https://www.npmjs.com/package/@normahq/codex-acp-bridge)

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

Try with ACP tools:

```bash
# Inspect initialize/session payloads
npx @normahq/acp-dump -- npx -y @normahq/codex-acp-bridge@latest
npx @normahq/acp-dump --json -- npx -y @normahq/codex-acp-bridge@latest

# Interactive ACP session
npx @normahq/acp-repl -- npx -y @normahq/codex-acp-bridge@latest

# If installed globally:
acp-dump -- codex-acp-bridge
acp-repl -- codex-acp-bridge
```

Documentation:

- [Usage](docs/usage.md)
- [JSON API](docs/json-api.md)
