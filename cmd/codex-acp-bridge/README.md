# codex-acp-bridge

`codex-acp-bridge` runs the Codex bridge backend and exposes it as an ACP agent over stdio.

## Installation

Global install (npm):

```bash
npm install -g @normahq/codex-acp-bridge@latest
```

One-off run with npx:

```bash
npx -y @normahq/codex-acp-bridge@latest
```

## Run

```bash
codex-acp-bridge
codex-acp-bridge --name team-codex
codex-acp-bridge --debug
```

## Flags

- `--name`: ACP agent name exposed via `initialize.agentInfo.name`.
  Default: `norma-codex-acp-bridge`.
- `--debug`: Enable debug logging.

## Behavior

- Creates separate backend sessions per ACP session.
- Working directory precedence for each backend session:
  - Uses ACP `session/new.params.cwd` when provided.
  - Falls back to bridge process working directory.
- Session-level Codex defaults are configured through ACP `session/new.params._meta.codex` (not CLI flags).
- Supports ACP `session/set_model` and `session/set_mode`.
  - `session/set_model` is applied to subsequent `turn/start` payloads.
  - `session/set_mode` is stored as ACP-side session state and is not forwarded into backend request payloads.
- Exposes ACP `session/new.models` from app-server `model/list` when available.
- Model selection is ACP-native: use `session/set_model` (or ACP client `--model`) rather than bridge-specific model flags.
- Prompt content support:
  - Text and image blocks are supported (`PromptCapabilities.image=true`).
  - Audio blocks are rejected in `session/prompt` (`PromptCapabilities.audio=false`).
- Supports per-session MCP servers via ACP `session/new` `mcpServers` parameter (`stdio` and `http` only; `sse` is rejected).

## Session Metadata Mapping

Bridge session defaults are read from `session/new._meta.codex` with strict validation (unknown keys are rejected with `invalid_params`).

Supported keys:

- `sandbox` -> `thread/start.sandbox`
- `approvalPolicy` -> `thread/start.approvalPolicy`
- `approvalsReviewer` -> `thread/start.approvalsReviewer`
- `baseInstructions` -> `thread/start.baseInstructions`
- `developerInstructions` -> `thread/start.developerInstructions`
- `modelProvider` -> `thread/start.modelProvider`
- `personality` -> `thread/start.personality`
- `serviceTier` -> `thread/start.serviceTier`
- `ephemeral` -> `thread/start.ephemeral`
- `profile` -> `thread/start.config.profile`
- `compactPrompt` -> `thread/start.config.compact_prompt`
- `config` -> merged into `thread/start.config`

Precedence for `thread/start.config` keys:

- `profile` overrides `config.profile`.
- `compactPrompt` overrides `config.compact_prompt`.
- ACP `mcpServers` mapping overrides `config.mcp_servers`.

## MCP Servers

The bridge supports passing MCP servers via ACP `session/new`. On thread start, each server in `mcpServers` is translated to `config.mcp_servers` values.

Example ACP `session/new` request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "session/new",
  "params": {
    "cwd": "/workspace",
    "_meta": {
      "codex": {
        "sandbox": "workspace-write",
        "approvalPolicy": "on-request",
        "profile": "dev",
        "compactPrompt": "compact"
      }
    },
    "mcpServers": [
      {
        "stdio": {
          "name": "filesystem",
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      },
      {
        "http": {
          "name": "github",
          "url": "https://api.github.com"
        }
      }
    ]
  }
}
```

Supported transports: `stdio`, `http`.
