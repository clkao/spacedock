---
id: 3qpv8qv6pbvejajejs2dn3v8
title: Merge-ceremony ergonomics — ship-local combo, honest no-remote fallback, workflow merge-policy
status: backlog
source: issue sweep (2026-05-31) — CL dev-workflow-ergonomics triage; consolidates #223, #217, #225
started:
completed:
verdict:
score: "0.30"
worktree:
issue:
---

Collapse the repetitive local-merge ceremony an FO performs at every terminal boundary when a
workflow has no PR host, and stop the merge-hook guard from forcing `--force` on the documented
no-remote fallback. Today an FO running a local (non-PR) merge repeats a multi-step sequence per
entity, and the terminal-transition guard treats the legitimate no-remote fallback as a skipped
hook — so the operator must pass `--force`, which defeats the guard's purpose.

Consolidates three open issues:
- **#223** — add a `ship-local` combo that collapses the ~7-step merge→terminalize→archive→cleanup
  sequence into one guided action for the no-PR path.
- **#217** — the pr-merge mod's no-remote fallback plus the terminal-transition guard force
  `--force` on every local merge; the fallback should satisfy the guard honestly (clear `mod-block`
  after the local merge lands) without `--force`.
- **#225** — a workflow-level merge-policy declaration (e.g. `merge: local | pr`) so the FO and the
  guard stop re-deriving "is there a PR host?" per entity and stop demanding `--force`.

## Acceptance criteria

**AC-1 — A no-PR terminal merge completes without `--force`.**
Verified by: a test exercising the no-remote fallback path that asserts terminalize+archive succeed
with `mod-block` cleared and no `--force` flag.

**AC-2 — One guided action performs the local-merge → terminalize → archive → cleanup sequence.**
Verified by: a CLI/behavioral test that the combo lands the branch and reaches `done`+archived in a
single invocation.

**AC-3 — A workflow can declare its merge policy; the guard and FO honor it.**
Verified by: a fixture workflow declaring local-merge policy where the terminal guard does not
demand a PR or `--force`.

## Test plan

Go unit/behavioral tests over the terminal-transition guard and the new combo command; a fixture
workflow with no merge hook / local merge-policy to prove the guard path. Ideation should confirm
this composes with the existing `pr-merge` mod (the PR path stays unchanged) and with split-root
(state writes stay path-scoped to the state checkout).
