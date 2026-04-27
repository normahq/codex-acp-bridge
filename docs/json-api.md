# JSON API Specification

Status: draft  
Date: 2026-04-24  
Audience: `codex-acp-bridge` adapter maintainers

## Source of truth

This document is code-first for codex-acp-bridge behavior:
- backend schema shape comes from `codex app-server` JSON schema output.
- app-server protocol behavior is documented at https://developers.openai.com/codex/app-server.
- adapter projection semantics come from `internal/apps/codexacpbridge` implementation and tests.

```bash
codex app-server generate-json-schema --experimental --out /tmp/codex-app-schema-docspec
```

The current inventory in this document was refreshed from `codex-cli 0.124.0`.

Primary files:
- `/tmp/codex-app-schema-docspec/ServerNotification.json`
- `/tmp/codex-app-schema-docspec/ServerRequest.json`
- `/tmp/codex-app-schema-docspec/*Response.json`

This document does not require backend source internals; it documents observable backend schema plus current bridge adapter behavior.

## API surface summary

- Server notifications: 58 methods
- Server-initiated requests: 9 methods
- `ThreadItem` variants: 16 types
- `ThreadStatus.activeFlags`: `waitingOnApproval`, `waitingOnUserInput`

## Required adapter invariants

- Correlate every streaming/update event by `(threadId, turnId, itemId)` when present.
- Correlate every server request lifecycle by `(threadId, requestId)`.
- Preserve streaming text exactly as received.
- Preserve `item/*/outputDelta` text exactly as received when forwarding ACP tool call updates.
- Do not trim, collapse, or normalize whitespace in message deltas or completed text.
- Treat `item/completed` payload as authoritative final state for that item.
- Do not project `agentMessage.phase=commentary` into ACP message or thought output.

## ACP projection (normative)

The adapter should project Codex backend events into ACP session semantics as follows.

`session/update.agent_thought_chunk` is reserved for reasoning text deltas (`item/reasoning/textDelta`, `item/reasoning/summaryTextDelta`) only.

### Core notification mapping

| Codex method | Required params | ACP projection | Notes |
| --- | --- | --- | --- |
| `thread/started` | `thread` | session/thread started metadata | Use `thread.id` as backend thread handle. |
| `thread/status/changed` | `threadId,status` | no ACP streaming update | Recognized operational status. No ACP-native status stream is emitted. |
| `turn/started` | `threadId,turn` | turn started | Keep `turn.id` for correlation. |
| `item/started` | `threadId,turnId,item` | item lifecycle started | Emits ACP tool-call lifecycle only for tool-like item types. Non-tool items such as `reasoning`, `plan`, and `agentMessage` are ignored here. |
| `item/agentMessage/delta` | `threadId,turnId,itemId,delta` | buffer only | Append delta byte-for-byte by `itemId`; phase is not present on delta events, so projection waits for `item/completed`. |
| `item/plan/delta` | `threadId,turnId,itemId,delta` | `session/update.plan` | Aggregates deltas by `itemId`; plan deltas are not thoughts. |
| `item/reasoning/textDelta` | `threadId,turnId,itemId,contentIndex,delta` | `session/update.agent_thought_chunk` | Preserve index ordering. |
| `item/reasoning/summaryPartAdded` | `threadId,turnId,itemId,summaryIndex` | no ACP streaming update | Structural marker only; no text payload to project. |
| `item/reasoning/summaryTextDelta` | `threadId,turnId,itemId,summaryIndex,delta` | `session/update.agent_thought_chunk` | Append by summary index. |
| `item/commandExecution/outputDelta` | `threadId,turnId,itemId,delta` | `session/update.tool_call_update` | Text output chunk for command item. |
| `item/fileChange/outputDelta` | `threadId,turnId,itemId,delta` | `session/update.tool_call_update` | Streaming patch preview/update. |
| `item/fileChange/patchUpdated` | `threadId,turnId,itemId,changes` | `session/update.tool_call_update` | Joins `changes[].diff` as tool content and preserves raw output. |
| `item/mcpToolCall/progress` | `threadId,turnId,itemId,message` | `session/update.tool_call_update` | Progress text. |
| `item/completed` | `threadId,turnId,item` | item lifecycle completed | Finalizes ACP tool-call lifecycle for tool-like item types. For `agentMessage`, projects accumulated `item.text` by phase: missing/null/empty or `final_answer` -> `session/update.agent_message_chunk`; `commentary` -> no ACP output. |
| `turn/plan/updated` | `threadId,turnId,plan` | plan snapshot update | Snapshot; do not assume continuity with deltas. |
| `turn/diff/updated` | `threadId,turnId,diff` | no ACP streaming update | Diff is not emitted as thought/message/tool content. |
| `thread/tokenUsage/updated` | `threadId,turnId,tokenUsage` | usage update | Forwards `tokenUsage.last.{inputTokens,outputTokens,totalTokens,cachedInputTokens}` into ACP meta usage (`cachedInputTokens` -> `cachedReadTokens`). |
| `error` | `threadId,turnId,error,willRetry` | error event | If `willRetry=false`, finalize turn as failed/interrupted. |
| `turn/completed` | `threadId,turn` | turn completed | Authoritative prompt terminal signal. Latest schema does not require a top-level `turnId`; if present, it must match the active turn. |
| `serverRequest/resolved` | `threadId,requestId` | request lifecycle ack | Clear pending request state. |

### Turn completion

`turn/completed` is the normal authoritative terminal signal for an ACP prompt. Do not infer prompt completion from final text chunks, `item/completed`, tool-call updates, plan updates, or `turn/diff/updated`.

`agentMessage.phase=final_answer` marks terminal answer text for the item, not terminal state for the turn. The bridge still waits for `turn/completed`.

Latest app-server schema requires `threadId` and `turn` for `turn/completed`; it does not require a top-level `turnId`. The bridge still applies active-prompt correlation before completing:
- there must be an active ACP prompt for the session;
- `threadId` must match the active backend thread;
- if a top-level `turnId` is present, it must match the active turn.

ACP stop reason is derived from `turn.status`:
- `completed` -> `end_turn`
- `interrupted` -> `cancelled`
- `failed` -> `refusal`
- missing, unknown, or `inProgress` -> `end_turn`

`error` with `willRetry=false` is the exceptional terminal path and maps to `refusal`; retried errors do not complete the ACP prompt. Usage attached to `session/prompt` completion comes from token usage on the terminal event when present, otherwise from the latest `thread/tokenUsage/updated` notification observed for the session.

### Command output distinction

`command/exec/outputDelta` and `item/commandExecution/outputDelta` are different channels and must not be merged blindly.

- `command/exec/outputDelta`: connection-scoped raw process stream with `deltaBase64`, `processId`, `stream`, `capReached`.
  - Current adapter behavior: does not project this channel into ACP content/thought updates.
- `item/commandExecution/outputDelta`: turn/item-scoped text stream keyed by `itemId`.

### Agent message phase

Completed `agentMessage` items have the shape `{id,text,phase?}` plus optional metadata such as `memoryCitation`. `phase` uses Responses API wire values:

- `commentary`: interim/preamble/progress assistant text. The bridge hides this from ACP output.
- `final_answer`: terminal answer text for that assistant message item. The bridge forwards this as ACP agent message text.
- missing, null, or empty: phase unknown. The bridge preserves legacy compatibility by forwarding as ACP agent message text.

`item/agentMessage/delta` does not include `phase`; clients must not infer whether a delta is commentary or final answer until the completed item arrives.

### Server request mapping

Each server request must be handled and answered with the correct response shape, then tracked until `serverRequest/resolved`.

| Request method | Required params | Response contract |
| --- | --- | --- |
| `item/commandExecution/requestApproval` | `itemId,threadId,turnId` | `CommandExecutionRequestApprovalResponse` (`decision`) |
| `item/fileChange/requestApproval` | `itemId,threadId,turnId` | `FileChangeRequestApprovalResponse` (`decision`) |
| `item/permissions/requestApproval` | `itemId,permissions,threadId,turnId` | `PermissionsRequestApprovalResponse` (`permissions`, optional `scope`) |
| `item/tool/call` | `arguments,callId,threadId,tool,turnId` | `DynamicToolCallResponse` (`success`,`contentItems`) |
| `item/tool/requestUserInput` | `itemId,questions,threadId,turnId` | `ToolRequestUserInputResponse` (`answers`) |
| `mcpServer/elicitation/request` | `serverName,threadId` (+ mode-specific fields) | `McpServerElicitationRequestResponse` (`action`, optional `content`) |
| `account/chatgptAuthTokens/refresh` | `reason` | `ChatgptAuthTokensRefreshResponse` (`accessToken`,`chatgptAccountId`) |
| `applyPatchApproval` (deprecated) | `callId,conversationId,fileChanges` | `ApplyPatchApprovalResponse` (`decision`) |
| `execCommandApproval` (deprecated) | `callId,command,conversationId,cwd,parsedCmd` | `ExecCommandApprovalResponse` (`decision`) |

Decision values used by approval responses include:
- `accept`
- `acceptForSession`
- `decline`
- `cancel`

### Partial forwarding behavior (implemented)

For request types without a native ACP equivalent, the adapter uses ACP `session/request_permission` to collect a user decision, then returns schema-valid backend responses:

- `item/tool/call`: returns `DynamicToolCallResponse` with `success=false` and explanatory `contentItems`.
- `item/tool/requestUserInput`: maps chosen option labels into `answers[question_id].answers[]`.
- `mcpServer/elicitation/request`: maps decision to `action=accept|decline|cancel` and includes `_meta` passthrough when provided.
- `applyPatchApproval` / `execCommandApproval`: ACP approval outcomes are translated to legacy decisions:
  `accept -> approved`, `acceptForSession -> approved_for_session`, `decline -> denied`, `cancel -> abort`.
- `account/chatgptAuthTokens/refresh`: adapter responds from environment when available:
  `CODEX_CHATGPT_ACCESS_TOKEN`, `CODEX_CHATGPT_ACCOUNT_ID`, optional `CODEX_CHATGPT_PLAN_TYPE`;
  otherwise returns a structured request error.

## Projection groups

### ACP streaming updates

- Agent text: phase-visible completed `agentMessage` from `item/completed`.
- Agent thoughts: `item/reasoning/textDelta`, `item/reasoning/summaryTextDelta`.
- Plans: `item/plan/delta`, `turn/plan/updated`.
- Tool calls: `item/started` and `item/completed` for tool-like item types only,
  `item/commandExecution/outputDelta`,
  `item/commandExecution/terminalInteraction`, `item/fileChange/outputDelta`,
  `item/fileChange/patchUpdated`, `item/mcpToolCall/progress`,
  `item/autoApprovalReview/started`, `item/autoApprovalReview/completed`,
  `hook/started`, `hook/completed`.

### Prompt/session state

- Prompt completion: `turn/completed`; `error` with `willRetry=false`.
- Usage and metadata: `thread/tokenUsage/updated`, `account/rateLimits/updated`,
  `mcpServer/startupStatus/updated`.
- Request lifecycle: `serverRequest/resolved`.
- Thread correlation: `thread/started`, `turn/started`.

### Recognized no-op notifications

These notifications are accepted for schema compatibility but are not projected into ACP
message, thought, plan, or tool-call updates:

```text
account/login/completed
account/updated
app/list/updated
command/exec/outputDelta
configWarning
deprecationNotice
externalAgentConfig/import/completed
fs/changed
fuzzyFileSearch/sessionCompleted
fuzzyFileSearch/sessionUpdated
guardianWarning
item/reasoning/summaryPartAdded
mcpServer/oauthLogin/completed
model/rerouted
model/verification
skills/changed
thread/archived
thread/closed
thread/compacted
thread/name/updated
thread/realtime/closed
thread/realtime/error
thread/realtime/itemAdded
thread/realtime/outputAudio/delta
thread/realtime/sdp
thread/realtime/started
thread/realtime/transcript/delta
thread/realtime/transcript/done
thread/status/changed
thread/unarchived
turn/diff/updated
warning
windows/worldWritableWarning
windowsSandbox/setupCompleted
```

The adapter also recognizes legacy `thread/realtime/transcriptUpdated` as a no-op for backward compatibility.

## Full method inventory

### Notifications (58)

```text
account/login/completed
account/rateLimits/updated
account/updated
app/list/updated
command/exec/outputDelta
configWarning
deprecationNotice
error
externalAgentConfig/import/completed
fs/changed
fuzzyFileSearch/sessionCompleted
fuzzyFileSearch/sessionUpdated
guardianWarning
hook/completed
hook/started
item/agentMessage/delta
item/autoApprovalReview/completed
item/autoApprovalReview/started
item/commandExecution/outputDelta
item/commandExecution/terminalInteraction
item/completed
item/fileChange/outputDelta
item/fileChange/patchUpdated
item/mcpToolCall/progress
item/plan/delta
item/reasoning/summaryPartAdded
item/reasoning/summaryTextDelta
item/reasoning/textDelta
item/started
mcpServer/oauthLogin/completed
mcpServer/startupStatus/updated
model/rerouted
model/verification
serverRequest/resolved
skills/changed
thread/archived
thread/closed
thread/compacted
thread/name/updated
thread/realtime/closed
thread/realtime/error
thread/realtime/itemAdded
thread/realtime/outputAudio/delta
thread/realtime/sdp
thread/realtime/started
thread/realtime/transcript/delta
thread/realtime/transcript/done
thread/started
thread/status/changed
thread/tokenUsage/updated
thread/unarchived
turn/completed
turn/diff/updated
turn/plan/updated
turn/started
warning
windows/worldWritableWarning
windowsSandbox/setupCompleted
```

### Server requests (9)

```text
account/chatgptAuthTokens/refresh
applyPatchApproval
execCommandApproval
item/commandExecution/requestApproval
item/fileChange/requestApproval
item/permissions/requestApproval
item/tool/call
item/tool/requestUserInput
mcpServer/elicitation/request
```

## Data-model notes for adapter state

- `ThreadStatus.type` enum: `notLoaded | idle | systemError | active`.
- `ThreadStatus.active.activeFlags` enum: `waitingOnApproval | waitingOnUserInput`.
- `Turn.status` enum: `completed | interrupted | failed | inProgress`.
- `ThreadItem.type` enum:
  `userMessage | hookPrompt | agentMessage | plan | reasoning | commandExecution | fileChange | mcpToolCall | dynamicToolCall | collabAgentToolCall | webSearch | imageView | imageGeneration | enteredReviewMode | exitedReviewMode | contextCompaction`.
- `agentMessage.phase`: `commentary | final_answer | null` and may be missing on legacy/provider output.

## Known pitfalls to avoid

- Do not infer end-of-turn from last message chunk, `agentMessage.phase=final_answer`, item completion, tool completion, plan updates, or diff updates; wait for `turn/completed` unless terminal `error` with `willRetry=false` arrives.
- Do not treat plan deltas as authoritative final plan text.
- Do not emit `agentMessage.phase=commentary` as ACP message or thought output.
- Do not mix connection-scoped `command/exec/outputDelta` with turn-scoped command item output.
- Do not drop or rewrite whitespace in message or reasoning deltas.
- Do not consider a server request done before `serverRequest/resolved`.

## Implementation status (current)

- `internal/apps/codexacpbridge` currently recognizes all methods listed in this schema inventory (`58` notifications, `9` server requests).
- Recognized no-op notifications are intentionally not projected into ACP streaming updates.
