---
id: rzfw9erffjatwa40q0jrs5c6
title: Migrate Go module path to github.com/spacedock-dev/spacedock
status: implementation
source: sprint — captain (2026-05-31); repo moved to spacedock-dev/spacedock, install off `next`
started:
completed:
verdict:
score: "0.34"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-module-path-migration-spacedock-dev
issue:
---

Mechanical module-path rename so the binary + vendored skills land under the migrated home. The repo is now `spacedock-dev/spacedock` (captain). After this merges, the FO adds the origin and pushes `next` (captain-authorized).

## Acceptance criteria

**AC-1 — Module path is `github.com/spacedock-dev/spacedock`; tree builds + tests green; binary smoke works.**
End state: `go.mod` declares `module github.com/spacedock-dev/spacedock`; ZERO remaining `clkao/spacedock-v1` references in `*.go` / `go.mod`; `go build ./...`, `go test ./...`, `go test -race ./...`, `gofmt -l .`, `go vet ./...` all exit 0; the built binary still runs (`./spacedock --version`, `./spacedock status --workflow-dir docs/dev --json`).
Verified by: `grep -rl clkao/spacedock-v1 --include='*.go' . | wc -l` == 0 (+ go.mod check); the four gates with REAL captured exit codes; a binary smoke run observed.

**AC-2 — Rename only, no behavior change.**
Verified by: `git diff` is import-path-only (no logic edits); the full suite green (same test count as pre-rename, modulo nothing).

## Out of scope (this entity)
- Adding origin + pushing `next` — the FO does that post-merge (captain-authorized).
- The repo `.claude-plugin/plugin.json` + install docs + `requires-contract` — that is `fresh-install-journey` (D), which must also ensure `--plugin-dir <repo>` loads v1's OWN skills (see _sprint-notes friction #7).
- jf release pipeline ldflags target — jf picks up the new path (`github.com/spacedock-dev/spacedock/internal/cli.Version`).

## Notes
- 13 `*.go` files + `go.mod` reference the path. Sequence: all launcher/test work is merged on main (clean tree) — safe to rewrite whole-tree now. Coordinates with the queued dispatch-fix (k69e…) which also touches build.go — sequence them, not concurrent.
