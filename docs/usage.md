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

# Stream app-server agent messages live
codex-acp-bridge --message-streaming

# Disable app-server reasoning/thought streaming
codex-acp-bridge --reasoning-streaming=false
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
- `--message-streaming`:
  Stream app-server `item/agentMessage/delta` notifications as ACP `agent_message_chunk` updates.
  Default: `false`.
- `--reasoning-streaming`:
  Stream app-server reasoning deltas live when enabled. When disabled, suppress ACP thought output from reasoning entirely.
  Default: `true`.
- `--reasoning-thoughts`:
  Select which reasoning lane is projected as ACP thoughts: `off`, `summary`, `content`, or `both`.
  Default: `summary`.
- `--debug`:
  Enable debug logging for the bridge process.

## Behavior

- Starts the Codex backend with per-session working directory selection:
  - If ACP `session/new.params.cwd` is set, that value is used for the backend process.
  - Otherwise, the bridge process working directory is used.
- Negotiates app-server notification opt-outs during `initialize`:
  - `--message-streaming=false` opts out `item/agentMessage/delta`.
  - `--reasoning-streaming=false` opts out `item/reasoning/textDelta`, `item/reasoning/summaryTextDelta`, and `item/reasoning/summaryPartAdded`.
  - `--reasoning-thoughts=summary` opts out raw `item/reasoning/textDelta`.
  - `--reasoning-thoughts=content` opts out `item/reasoning/summaryTextDelta` and `item/reasoning/summaryPartAdded`.
  - `--reasoning-thoughts=off` opts out all reasoning delta notifications.
- Opens ACP agent-side stdio connection for clients.
- Creates one backend session per ACP session.
- `session/new` returns the app-server thread id (`thread.id`) as the ACP `sessionId`.
- Supports ACP `session/list` using backend `thread/list`.
  - `session/list` returns resumable Codex threads, not just sessions created by the current bridge process.
  - `session/list` maps ACP `cwd` filtering directly to backend `thread/list.cwd`.
- Supports ACP `session/close` using backend `thread/unsubscribe`.
  - `session/close` is a transient detach operation; it does not archive or delete the underlying Codex thread.
  - closed sessions remain resumable and listable until backend retention removes them.
- Supports ACP `session/resume` using direct app-server `thread/resume`.
  - `session/resume` restores session state only; it does not replay prior ACP message/thought/tool updates.
  - `thread.sessionId` remains a backend session-tree identifier; it is not the ACP resume handle.
  - ACP `session/load` is not implemented because the bridge does not replay prior conversation history as required by the protocol.
- Reads per-session Codex defaults from `session/new.params._meta.codex` (strictly validated).
- Supports ACP cancellation via `session/cancel`.
- Optional agent message streaming:
  - when `--message-streaming=true`, every app-server `agentMessage` item is streamed as ACP `agent_message_chunk`
  - streamed message chunks carry `_meta["codex-acp-bridge/itemId"]`, `_meta["codex-acp-bridge/completed"]`, and `_meta["codex-acp-bridge/phase"]`
  - `item/completed` closes the logical message; it does not complete the ACP turn
- Optional lane-aware reasoning thoughts:
  - `summary` projects `item/reasoning/summaryTextDelta`
  - `content` projects `item/reasoning/textDelta`
  - `both` projects both lanes and keeps them distinct via `_meta`
  - thought chunks carry `_meta["codex-acp-bridge/itemId"]`, `_meta["codex-acp-bridge/reasoningKind"]`, index metadata, and `_meta["codex-acp-bridge/completed"]`
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
  - `session/set_model` accepts only model IDs advertised by app-server `model/list`, updates model selection used by subsequent `turn/start` calls, and persists it to app-server thread settings when the thread already exists.
  - `session/set_mode` is stored in ACP session state only; current bridge implementation does not forward mode into backend `thread/start` or `turn/start` payload fields.
- Supports ACP session configuration options for reasoning effort.
  - `session/new.configOptions` includes a `reasoning_effort` select option when app-server `model/list` advertises reasoning efforts for the current model.
  - `session/set_config_option` with `configId=reasoning_effort` updates the effort used by subsequent `turn/start.effort` payloads and persists it to app-server thread settings when the thread already exists.
  - Supported values are model-advertised and may include values such as `minimal`, `low`, `medium`, `high`, or `xhigh`.
- Populates ACP `session/new.models` from app-server `model/list` when available.
- Model selection is ACP-native; prefer `session/set_model`.
- `session/prompt._meta.error` preserves raw provider/app-server terminal error details for `error(willRetry=false)` and `turn/completed(status=failed)` when the backend provides them.
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

- `session/new._meta.sessionId` is rejected; ACP session ids are backend-generated and durable.
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
