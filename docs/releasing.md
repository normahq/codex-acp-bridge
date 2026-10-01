# Legacy release synchronization

Canonical versions and npm/native publication are owned by
[baldaworks/codex-acp](https://github.com/baldaworks/codex-acp).
This repository maintains only the original Go module identity and thin CLI/API
adapters. Preserve all existing tags, npm versions and historical GitHub assets.

`sync-canonical-release.yml` runs hourly at minute 23 or by manual dispatch:

```bash
gh workflow run sync-canonical-release.yml --repo normahq/codex-acp-bridge -f version=v1.10.1
```

Omit `version` to select the latest published canonical release. The workflow
uses its own `GITHUB_TOKEN` with `contents: write`; it needs no cross-repository
write credential. It validates an exact non-prerelease `vX.Y.Z`, rejects
missing releases and downgrades, updates the canonical Go dependency, runs
race tests/lint and atomically pushes the commit and matching immutable tag.
An existing tag must pin the same canonical version and is never moved.

The canonical tag must be publicly available before dependency tidy/checksum
population. Local migration preparation can use a temporary Go workspace;
never commit a consumer replace directive. Initial adapter main is pushed only
after its canonical dependency exists, then the workflow can publish its tag.

`scripts/release-assets.py` verifies canonical checksums and reuses the same
native binaries under legacy archive/executable names. Archives include this
repository's README/license at the tagged revision and are reproducible. A
retry verifies accepted assets and uploads only missing ones; differing existing
assets fail instead of being overwritten.

Verify both pinned/latest legacy Go installs and public constructors after sync,
plus archive checksums and preserved historical tag/asset inventories. Registry
publication and trusted publishers belong to the canonical workflow. Refer to
[canonical releasing](https://github.com/baldaworks/codex-acp/blob/main/docs/releasing.md)
and [migration policy](https://github.com/baldaworks/codex-acp/blob/main/docs/migration.md).

The synchronization step uses `GOTOOLCHAIN=auto` so Go can select the
toolchain required by the canonical dependency before updating the legacy
module minimum. Ordinary CI then uses the updated `go.mod` baseline.
