---
id: 2y7n58x5yx7yn64sy6q3jzmk
title: Sandbox-flag passthrough — spacedock --safehouse-enable / --safehouse-add-dirs
status: ideation
source: sprint — captain (2026-05-31); closes the safehouse --enable gap (.safehouse config can't carry capabilities)
started: 2026-05-31T01:51:45Z
completed:
verdict:
score: "0.32"
worktree:
issue:
---

Let operators pass sandbox capability/path knobs through `spacedock claude` (and `spacedock codex`) to the underlying sandbox, namespaced by sandbox so future sandboxes get their own namespace.

## Why (gap found 2026-05-31)
safehouse's docs confirm `--enable=KEY` (docker, ssh, kubectl, …) is **command-line only** — the workdir `.safehouse` file carries only path grants (`add-dirs`/`add-dirs-ro`), NOT capabilities. The shipped `spacedock claude` (A) passes `--trust-workdir-config` + `extra=nil`, so there is currently NO way to enable docker/ssh through the launcher. The F11 decision ("`--trust-workdir-config` covers it") was based on a wrong assumption about `.safehouse`'s schema.

## Target design (captain, 2026-05-31)
Sandbox-namespaced front-door flags, consumed by the launcher and translated into the existing `safehouse.Wrap(inner, extra)` `extra` slot (which lands before the `--`):

- `spacedock claude --safehouse-enable=ssh,docker --safehouse-add-dirs=<paths> [-- claude-args]`
  → `safehouse --enable=ssh --enable=docker --add-dirs=<paths> --trust-workdir-config -- claude --dangerously-skip-permissions --agent spacedock:first-officer …`
- `--safehouse-enable=` is comma-separated → one repeated `--enable=KEY` per value.
- `--safehouse-add-dirs=` / `--safehouse-add-dirs-ro=` → `--add-dirs=` / `--add-dirs-ro=`.
- The `--safehouse-` prefix is the per-sandbox NAMESPACE: a future sandbox `X` would add `--X-*` flags that map to X's own flag surface. Design the parse/translate seam so adding a namespace is a clean extension — but do NOT build any other sandbox now (YAGNI).
- Applies to both `spacedock claude` and `spacedock codex` (both go through the shared safehouse Wrap path).

## Acceptance criteria (provisional — ideation hardens each into an exercise-and-observe oracle)

**AC-1 — `--safehouse-enable=ssh,docker` forwards repeated `--enable` flags.**
Verified by: recorded-Launch oracle observes the safehouse argv contains `--enable=ssh --enable=docker` (comma-split into repeated flags) positioned before the `--` separator.

**AC-2 — `--safehouse-add-dirs=` / `--safehouse-add-dirs-ro=` forward path grants.**
Verified by: recorded-Launch oracle observes `--add-dirs=<paths>` / `--add-dirs-ro=<paths>` in the safehouse extra slot.

**AC-3 — the namespace seam is a clean extension point.**
Verified by: a unit-level test of the parse/translate function showing the `--safehouse-` namespace maps to safehouse flags, with the seam shaped so a hypothetical second namespace would not require rewriting the dispatcher (design-level, exercised by the parser's structure — not prose).

**AC-4 (ideation to decide) — behavior when `--safehouse-*` flags are given but no `.safehouse` profile (plain launch).**
The launcher should not silently drop them — either error ("`--safehouse-enable` given but this directory has no `.safehouse` profile") or a documented ignore. Ideation picks one with a behavioral oracle.

## Notes / sequencing
- Lands on `internal/cli/frontdoor.go` (front-door flag parsing, like `--skip-contract-check`) + `internal/safehouse` (the `extra` builder). Shares those files with `codex-safehouse-launcher` (in implementation) → serialize after it.
- Build on the final module path after the `spacedock-dev/spacedock` migration if it has landed by implementation time.
