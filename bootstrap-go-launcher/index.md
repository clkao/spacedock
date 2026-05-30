---
id:
title: Bootstrap Go launcher
status: backlog
score: "0.95"
source: bootstrap roadmap
worktree:
---

# Bootstrap Go Launcher

Create the minimum Go project that can host the Spacedock v1 launcher.

## Acceptance Criteria

- `cmd/spacedock` exists and delegates to a small CLI package.
- The launcher supports help, version, and unknown-command behavior.
- The project has clear agent instructions for Go and skill development.
- No status behavior is implemented beyond a deliberate placeholder.

## Test Gates

- `go test ./...`
- `go run ./cmd/spacedock --help`
- `go run ./cmd/spacedock --version`
- CLI unit tests cover help, version, and unknown commands.

## Notes

This stage proves the repo is buildable before compatibility work begins.
