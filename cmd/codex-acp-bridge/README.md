# Legacy codex-acp-bridge command

```bash
go install github.com/normahq/codex-acp-bridge/cmd/codex-acp-bridge@latest
codex-acp-bridge version
codex-acp-bridge
```

This archived Go command is frozen at `v1.10.1`, including `@latest`.

The command preserves the original installation path and delegates to the
canonical public Cobra command through this module's pinned dependency.
Signal handling, stdio and failure exit status remain supported. The command
help and default agent identity use the canonical `codex-acp` name.

For new installations use
`go install github.com/baldaworks/codex-acp/cmd/codex-acp@latest`.
See [compatibility and installation](../../README.md) and
[canonical usage](https://github.com/baldaworks/codex-acp/blob/main/docs/usage.md).
