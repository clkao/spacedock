---
id: n15ga31s6dbs3xxbhbz4at8s
title: Graduate the full plugin surface onto next (released-lane completion, next-as-source)
status: ideation
source: sprint — captain 2026-05-31 ("next-as-source is fine"); released-lane follow-up to fresh-install-journey (dev-lane)
started: 2026-05-31T16:07:11Z
completed:
verdict:
score: "0.30"
worktree:
issue:
---

The dev-lane self-hosting loop shipped (D: v1 vendors a MINIMAL `.claude-plugin/plugin.json` + first-officer/ensign so `--plugin-dir <repo>` loads v1's own skills). The captain confirmed **next-as-source**: the released lane (`spacedock init` / marketplace) publishes the plugin from `spacedock-dev/spacedock@next` — NO cross-repo edit to `main`. This entity makes `next` a COMPLETE publishable plugin.

## Why
Publishing from `next` today would ship an INCOMPLETE plugin: `next` carries only first-officer+ensign SKILL.md (D's dev-lane minimal scope) — it would DROP `debrief`+`refit` (present on `main`) and ship `integration` (v1 test-only). And `next` has no `.codex-plugin/plugin.json` / `marketplace.json`. So the released lane isn't fully publishable yet.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — `next` carries the full user plugin surface.** All USER skills get SKILL.md + agents + reference closure: `first-officer`, `ensign` (done), plus `commission`, `debrief`, `refit`. EXCLUDE `integration` (test-only). Verified by: `claude --plugin-dir <repo> plugin details spacedock` (isolated config) lists the full skill set; dependency-closure check (no dangling references) like fresh-install-journey's audit.

**AC-2 — Both host manifests + marketplace.** Expand `.claude-plugin/plugin.json` to the full surface; add the authoritative `.codex-plugin/plugin.json`; add `marketplace.json` so `claude/codex plugin marketplace add spacedock-dev/spacedock@next` resolves. requires-contract `">=1,<2"` in both. Verified by: marketplace add + install from `@next` in an isolated host config installs the complete plugin; `spacedock doctor` → Compatible.

**AC-3 — `spacedock init` pins `@next`** (until `next` becomes the default branch) and the released-lane gate is green. Verified by: `spacedock init --host claude` issues `marketplace add spacedock-dev/spacedock@next`; post-install `spacedock claude` gates green with NO `--skip-contract-check`.

## Folded cleanups
- The stale `release.yml` comment ("the brews block can push the formula bump" → it's now a casks block pushing a cask) — jf-audit Polish, fix here.

## Notes
- Released-lane companion to fresh-install-journey (dev-lane). No cross-repo `requires-contract` edit (next-as-source). Eventual graduation: `next` → default branch retires the legacy Python `main`.
