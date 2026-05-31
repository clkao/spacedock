---
id: k69e2gcvykjrc5354ty7kt3g
title: Dispatch — inject split-root state-commit guidance for non-worktree (ideation) stages
status: backlog
source: FO dogfooding friction #1 (2026-05-31); root-caused build.go:302 worktree-gating
started:
completed:
verdict:
score: "0.24"
worktree:
issue:
---

The native dispatch build injects the split-root state-commit guidance (`git -C {state_checkout} add/commit -- {entity}`) ONLY for worktree stages — `internal/dispatch/build.go:302` gates the whole block on `if worktreePath != ""`. So NON-worktree dispatches (ideation, backlog) get NO state-commit instruction. Result: ideation ensigns edit the entity in `.spacedock-state` (git-excluded from the main checkout), try a bare `git add`, hit the exclusion, and report "couldn't commit — gitignored." This recurred on every ideation-stage dispatch this session (worktree-stage ensigns committed cleanly).

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — Non-worktree split-root dispatches carry the state-commit guidance.**
End state: for a split-root workflow, a NON-worktree-stage dispatch body includes the path-scoped `git -C {state_checkout} add {entity} && git -C {state_checkout} commit -- {entity}` guidance (and the "never bare `git add -A`/`git commit`" concurrency note), naming the real state-checkout path.
Verified by: a dispatch-build test (alongside `internal/dispatch/build_parity_test.go`) that builds a non-worktree split-root dispatch and asserts the body contains the path-scoped state-commit instruction with the resolved state-checkout path — exercise the build, observe the emitted body (behavioral, not prose-grep of the contract).

**AC-2 — Worktree-stage behavior unchanged.**
Verified by: existing worktree-stage dispatch parity tests stay green (the worktree path still emits its CODE-branch + state-commit guidance).

## Notes
- Small `internal/dispatch/build.go` change (lift the state-commit guidance out of the `worktreePath != ""` block, or add a non-worktree split-root branch). Companion: de-frame the vendored ensign contract's "Split-Root **Worktree** Contract" section so a non-worktree ensign sees it applies (skills/ensign/references/ensign-shared-core.md).
- Sequencing: touches `internal/dispatch/build.go` — coordinate with the module-path migration (which rewrites imports across the repo). Do this fix before OR after the migration, not concurrently.
- Not on the launcher/install critical path; queued.
