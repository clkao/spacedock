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

## Stage Report: implementation

- DONE: Rewrite the module path `github.com/clkao/spacedock-v1` → `github.com/spacedock-dev/spacedock` in go.mod AND every `*.go` import (13 files). Confirm ZERO remaining refs.
  `grep -rl clkao/spacedock-v1 --include='*.go' .` == 0; go.mod declares `module github.com/spacedock-dev/spacedock`. 14 files rewritten via `grep -rl | while read; sed`.
- DONE: Gates green with REAL captured exit codes, then a binary smoke.
  build/vet exit 0; gofmt -l clean; `go test ./...` 470 passed (exit 0); `go test -race ./...` 470 passed (exit 0); binary rebuilt, `./spacedock --version` → `spacedock 0.1.0-dev (contract 1)`, `./spacedock status --workflow-dir docs/dev --json` → `{"command":"status","entities":[]}` (both exit 0).
- DONE: Diff is import-path-only — NO logic changes. Commit on the worktree branch.
  Commit 6364a8b on `spacedock-ensign/module-path-migration`: 14 files, 17 insertions / 17 deletions, every hunk a path-string substitution.

### Summary

Pure mechanical module-path rename `github.com/clkao/spacedock-v1` → `github.com/spacedock-dev/spacedock` across go.mod + 13 `*.go` files. No logic touched; the diff is 17/17 line-for-line path swaps. All gates green (build, vet, gofmt, 470 tests, 470 race tests) and the rebuilt binary runs `--version` and `status --json` cleanly. Per scope: no plugin.json, no marketplace, no remote/push — those belong to fresh-install-journey (D) and the FO's post-merge push.
