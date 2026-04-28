# Usage

This command runs the Codex bridge backend and exposes it as an ACP agent over stdio.

Command:

```bash
npx -y @normahq/codex-acp-bridge@latest
# or when installed globally:
codex-acp-bridge
```

## Why this exists

- ACP runners need a stable ACP endpoint.
- `codex-acp-bridge` provides a stable command name for Codex ACP integration.
- The bridge uses Codex app-server backend runtime semantics.

## Usage

```bash
# Start bridge with defaults
codex-acp-bridge

# Set ACP agent name seen by ACP clients in initialize.agentInfo.name
codex-acp-bridge --name team-codex
```

## ACP Tooling Examples

Use `acp-dump` to inspect ACP initialize/session behavior:

```bash
npx -y @normahq/acp-dump -- npx -y @normahq/codex-acp-bridge@latest
npx -y @normahq/acp-dump --json -- npx -y @normahq/codex-acp-bridge@latest
```

Use `acp-repl` for an interactive ACP prompt session:

```bash
npx -y @normahq/acp-repl -- npx -y @normahq/codex-acp-bridge@latest
```

If tools are installed globally:

```bash
acp-dump -- codex-acp-bridge
acp-repl -- codex-acp-bridge
```

## Flags

- `--name`:
  ACP agent name reported in `initialize.agentInfo.name`.
  Default: `norma-codex-acp-bridge`.
- `--debug`:
  Enable debug logging for the bridge process.

## Behavior

- Starts the Codex backend with per-session working directory selection:
  - If ACP `session/new.params.cwd` is set, that value is used for the backend process.
  - Otherwise, the bridge process working directory is used.
- Opens ACP agent-side stdio connection for clients.
- Creates one backend session per ACP session.
- Reads per-session Codex defaults from `session/new.params._meta.codex` (strictly validated).
- Supports ACP cancellation via `session/cancel`.
- Supports per-session MCP servers via ACP `session/new` `mcpServers` parameter.
  - Supported transports: `stdio`, `http`.
  - `sse` is not supported.
  - Each `mcpServers[]` entry must define exactly one transport.
  - Bridge maps these values to `config.mcp_servers.<id>.*` in backend thread start params.
  - Merge contract: ACP `mcpServers` entries override same-name servers in `config.mcp_servers`; other configured MCP servers remain active.
  - MCP startup visibility:
    - `session/new._meta.codex.mcp` includes `contract` and requested server descriptors.
    - `session/prompt._meta.codex.mcp.startupStatus` includes latest startup status/error for requested servers.
- Supports `session/set_model` and `session/set_mode` for ACP session state.
  - `session/set_model` updates model selection used by subsequent `turn/start` calls.
  - `session/set_mode` is stored in ACP session state only; current bridge implementation does not forward mode into backend `thread/start` or `turn/start` payload fields.
- Supports ACP session configuration options for reasoning effort.
  - `session/new.configOptions` includes a `reasoning_effort` select option when app-server `model/list` advertises reasoning efforts for the current model.
  - `session/set_config_option` with `configId=reasoning_effort` updates the effort used by subsequent `turn/start.effort` payloads.
  - Supported values are model-advertised and may include values such as `minimal`, `low`, `medium`, `high`, or `xhigh`.
- Populates ACP `session/new.models` from app-server `model/list` when available.
- Model selection is ACP-native; prefer `session/set_model`.
- Prompt content support:
  - Text and image prompt blocks are supported (`PromptCapabilities.image=true`).
  - Audio prompt blocks are not supported in `session/prompt` (`PromptCapabilities.audio=false`).

## `session/new._meta.codex` Mapping

Supported keys and mappings:

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

`config.model_reasoning_effort` remains available as a Codex-specific startup default. Prefer ACP `session/set_config_option` for interactive reasoning-effort changes after session creation.

Validation and precedence:

- Unknown `codex` keys are rejected with ACP `invalid_params`.
- `profile` overrides `config.profile`.
- `compactPrompt` overrides `config.compact_prompt`.
- ACP `mcpServers` mapping overrides same-name entries in `config.mcp_servers` (merge semantics; non-overlapping entries are retained).

Example `session/new` request:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "session/new",
  "params": {
    "cwd": "/workspace",
    "_meta": {
      "sessionId": "session-123",
      "codex": {
        "sandbox": "workspace-write",
        "approvalPolicy": "on-request",
        "approvalsReviewer": "user",
        "profile": "dev",
        "compactPrompt": "compact"
      }
    },
    "mcpServers": []
  }
}
```

## Exit behavior

- Returns non-zero if backend setup fails.
- Returns zero when ACP client disconnects normally.
