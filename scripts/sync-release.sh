#!/usr/bin/env bash
set -euo pipefail

canonical_repo=baldaworks/codex-acp
canonical_module=github.com/baldaworks/codex-acp
requested="${1:-}"
if [[ -n "$requested" && ! "$requested" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Expected a canonical vX.Y.Z release" >&2
  exit 1
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Synchronization requires a clean checkout" >&2
  exit 1
fi
release_endpoint="repos/$canonical_repo/releases/${requested:+tags/}"
release_endpoint+="${requested:-latest}"
version=$(gh api "$release_endpoint" --jq 'select(.draft == false and .prerelease == false) | .tag_name')
if [[ ! "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ || ( -n "$requested" && "$version" != "$requested" ) ]]; then
  echo "Canonical release is not an accepted vX.Y.Z release" >&2
  exit 1
fi
current=$(awk -v module="$canonical_module" '$1 == module {print $2}' go.mod)
if [[ ! "$current" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Missing exact canonical Go dependency" >&2
  exit 1
fi
if [[ "$(printf '%s\n' "$current" "$version" | sort -V | head -n 1)" != "$current" ]]; then
  echo "Refusing to downgrade canonical dependency" >&2
  exit 1
fi
# Fetch only the requested tag; never move a locally accepted historical tag.
remote_tag=$(git ls-remote --tags origin "refs/tags/$version")
if [[ -n "$remote_tag" ]]; then
  git fetch --quiet origin "refs/tags/$version:refs/tags/$version"
  pinned=$(git show "$version:go.mod" | awk -v module="$canonical_module" '$1 == module {print $2}')
  if [[ "$pinned" != "$version" ]]; then
    echo "Existing immutable legacy tag has a different canonical dependency" >&2
    exit 1
  fi
  echo "Legacy $version already synchronized"
else
  go mod edit "-require=$canonical_module@$version"
  go mod tidy
  go test -race ./...
  go tool golangci-lint run
  git add go.mod go.sum
  if ! git diff --cached --quiet; then
    git commit -m "chore(deps): synchronize codex-acp $version"
  fi
  git tag -a "$version" -m "Legacy compatibility for codex-acp $version"
  git push --atomic origin HEAD:main "refs/tags/$version"
fi
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "version=$version" >> "$GITHUB_OUTPUT"
fi
printf '%s\n' "$version"
