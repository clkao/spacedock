# Pi First Officer Runtime

This file defines how the shared first-officer core executes on Pi.

## Runtime Shape

Pi is a first-class runtime target. For the first slice, treat Pi worker identity as a Spacedock-owned handle layered over Pi session identity rather than assuming Claude-style teams or Codex-style native collaborator handles.

The Pi path must support:
- fresh worker dispatch
- wait for completion
- same-worker routed follow-up / reuse
- explicit shutdown
- interactive and non-interactive execution
- worktree-backed stages

Standing teammates are out of scope for the first Pi slice.

## Worker Model

Use one Pi session per worker assignment. Persist enough metadata to reopen the same worker for routed reuse:
- FO-owned worker label
- logical dispatch id
- worker key
- Pi session id or session path
- cwd / worktree path
- entity slug and stage
- active / completed / shutdown state
- a completion epoch or equivalent marker so reused follow-up completions are distinguishable from stale prior completions

## Dispatch Adapter

The first-officer dispatch prompt is authoritative for Pi workers.

Fresh dispatch should:
1. create or open a Pi worker session
2. send a fully self-contained assignment
3. wait for completion on that same worker session before advancing the entity

Routed reuse should:
1. reopen the same Pi worker session
2. send the concrete next-stage or feedback-fix assignment
3. mark the worker active again
4. wait for the new completion before using it as evidence

Do not treat the existence of an older completed Pi session turn as proof that the follow-up routed work has completed.

## Working Directory

The FO stays anchored at the repo root.

When a stage is worktree-backed, the Pi worker runs in the assigned worktree and uses the worktree copy of the entity file. When a stage is not worktree-backed, the Pi worker stays on the main branch copy.

## Output Discipline

For the first Pi slice, report operator-facing progress in the same high-level lifecycle as other runtimes:
- dispatching worker label
- waiting on worker label
- routed follow-up on existing worker label
- shutting down worker label when no later routing remains

The stage report in the entity file remains the source of truth for completion and gate review.
