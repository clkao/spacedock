---
id: 218
title: "Pi runtime compatibility baseline"
status: implementation
source: "github#147"
started: 2026-04-27T22:48:56Z
completed:
verdict:
score:
worktree: .worktrees/spacedock-ensign-pi-runtime-compatibility-baseline
issue: "#147"
pr:
mod-block: merge:pr-merge
---

## Problem Statement

GitHub issue #147 asks whether Spacedock can support the Pi agent runtime and, if not yet, what minimum runtime support would be needed to make that credible. Today the repo has explicit Claude Code and Codex runtime adapters, runtime-specific tests, and harness branches, but no Pi runtime path.

The main product question is not "can Pi do everything Claude/Codex do right now?" but "what is the smallest first-class Pi runtime surface that proves Spacedock can orchestrate a real workflow under Pi?" The first proving bar should be a shared live-workflow baseline rather than full parity. That means scoping the work to the minimum runtime contract needed to dispatch an ensign, wait for completion, handle a gate correctly, and preserve Spacedock's worktree and reuse semantics where they matter.

A second design constraint is that Pi should be treated as a genuine runtime target, not a disguised Claude or Codex code path. The implementation can still use Pi-specific harness glue, but the workflow behavior should match the shared first-officer contract closely enough that existing runtime tests can grow a `--runtime pi` branch instead of inventing a separate Pi-only workflow model.

## Scope Boundary

In scope for the first Pi slice:

- Add Pi as a first-class runtime alongside Claude Code and Codex.
- Support both interactive and non-interactive/batch execution.
- Support worktree-backed stages from day 1.
- Support fresh worker dispatch, blocking wait, same-worker routed follow-up/reuse, and explicit shutdown.
- Allow Pi-specific harness glue where necessary, as long as workflow behavior matches the shared contract.
- Use `tests/test_gate_guardrail.py --runtime pi` as the first proving target.

Explicitly out of scope for the first slice:

- Standing teammates such as `comm-officer`.
- Startup/shutdown team lifecycle.
- Broad live-suite parity across all existing runtime tests.
- Large runtime-abstraction refactors across Claude/Codex/Pi.

## Proposed Approach

Use a thin first-class Pi runtime adapter over the existing FO/ensign contract rather than a broad runtime-generalization refactor or a throwaway Pi-only special mode.

The runtime should expose a Spacedock-owned worker-handle model even if Pi itself is implemented in terms of sessions. The smallest viable contract is:

1. **Fresh dispatch**
   - The FO can launch a Pi worker for a self-contained assignment.
   - The worker runs either at repo root or in an assigned worktree.
   - Dispatch returns a stable handle.

2. **Wait for completion**
   - The FO can wait on a specific worker handle until the worker is idle/completed.
   - Completion evidence comes from the worker's final response and the entity file/stage report.

3. **Reuse / routed follow-up**
   - The FO can send a concrete next-stage or feedback assignment to the same worker handle.
   - The worker becomes active again on that same handle.
   - The FO can wait for the new completion and must not treat a stale prior completion as evidence.

4. **Explicit shutdown**
   - The FO can mark a worker unroutable once no further advancement, feedback, or gate handling is expected.

5. **Session continuity**
   - Interactive and batch runs both preserve enough identity to reopen a worker handle for reuse.
   - The runtime handle should be the Pi session id or session file itself, with only thin workflow metadata layered on top.

6. **Worktree isolation**
   - The FO remains anchored at repo root.
   - Worktree-backed worker stages run with the worktree as their cwd.

The likely Pi mapping is one Pi session per worker. Fresh dispatch creates or opens a worker session and sends a fully self-contained assignment. Wait drives that session until idle/completion. Reuse reopens the same session and sends the follow-up assignment. For the first slice, this reopened same-session follow-up counts as valid worker reuse even if Pi is not yet managed as a continuously live background worker. Shutdown is recorded explicitly in thin Spacedock workflow metadata even if the underlying Pi session file still exists.

Architecturally, this suggests:

- add `skills/first-officer/references/pi-first-officer-runtime.md`
- add the corresponding Pi runtime branch in ensign runtime selection
- introduce a small Pi worker adapter plus a thin worker-label -> session mapping that stores session id/path, cwd/worktree path, entity/stage, and active/completed/shutdown state
- add a `--runtime pi` branch in the live harness so tests can launch Pi and parse Pi-specific evidence while preserving shared behavioral assertions

The first live behavior target is the gate-preflight sequence:

1. FO discovers workflow and dispatchable entity.
2. FO transitions the entity into the next stage.
3. FO creates a worktree when the stage requires it.
4. FO launches a Pi worker with a fully self-contained assignment.
5. FO waits for completion.
6. FO reads the produced stage report.
7. FO holds at the gate without self-approving.
8. FO reports the gate state clearly.

This is the smallest meaningful contract because it proves the FO can orchestrate a real worker cycle under Pi rather than merely parse files.

## Acceptance criteria

**AC-1 — Pi is defined as a first-class runtime surface rather than an undocumented variant of Claude or Codex.**
Verified by: static contract tests that assert Pi-specific runtime selection and a Pi runtime reference document exist for the FO/ensign path.

**AC-2 — Pi can drive the shared gate-preflight workflow shape in both interactive and batch contexts.**
Verified by: `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime pi -v`.

**AC-3 — Pi worker orchestration supports fresh dispatch, same-worker routed follow-up/reuse, and explicit shutdown with stable worker identity.**
Verified by: focused Pi runtime unit/integration tests covering dispatch, wait, reuse, stale-completion rejection, and shutdown behavior.

**AC-4 — Worktree-backed stages run correctly under Pi while the FO remains anchored at repo root.**
Verified by: targeted Pi runtime tests asserting worktree cwd, worktree entity path routing, and repo-root FO operations.

**AC-5 — Pi-specific harness glue remains behaviorally compatible with the shared runtime test model.**
Verified by: harness/runtime tests that accept `--runtime pi` and preserve gate-guardrail behavioral assertions without creating a separate Pi-only workflow contract.

## Test Plan

Primary proof:

- `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime pi -v`

Expected supporting tests:

- static contract assertions in `tests/test_agent_content.py` or equivalent for Pi runtime selection and contract wording
- focused Pi runtime tests for worker registry/identity, dispatch, wait, reuse, shutdown, and worktree cwd/entity-path behavior
- harness tests for `--runtime pi` command selection and parseable output capture

Suggested validation order:

1. add static/runtime-selection assertions first
2. add Pi worker-registry and adapter unit tests
3. add worktree-aware Pi runtime tests
4. add `--runtime pi` harness coverage
5. turn on the Pi branch in `tests/test_gate_guardrail.py`

Estimated complexity is medium. The first slice is intentionally smaller than full runtime parity, but it still crosses runtime contract selection, worker/session orchestration, test harness plumbing, and at least one real live workflow path.

## Implementation Notes

The design should prefer a thin adapter over a broad runtime abstraction rewrite. A runtime-generalization refactor is attractive long-term, but it is not the fastest or safest way to answer #147.

The riskiest technical edges are:

- Pi is session-oriented, so Spacedock should use Pi sessions as canonical worker handles and add only the thinnest workflow mapping needed for labels and lifecycle bookkeeping.
- Reuse must track a new completion epoch/turn boundary so a reopened session does not accidentally reuse stale completion evidence.
- Shutdown may need to be defined at the Spacedock metadata layer even if Pi does not expose a native per-worker kill primitive identical to other runtimes.
- Interactive and batch invocation can use different transport surfaces, but they should share one runtime contract and one worker metadata model.
- Reopened-session reuse is acceptable for the first slice, but a later optimization path should prefer SDK-managed in-memory keep-alive sessions to reduce repeated process/session startup overhead.

A reasonable implementation order is:

- add Pi runtime contract files and runtime selection
- add the thin Pi worker/session mapping
- implement dispatch/wait/reuse/shutdown semantics
- wire worktree-aware prompt/context assembly
- add `--runtime pi` harness support
- make the gate-preflight test pass
- add a narrower follow-up reuse test before broadening runtime parity

## Stage Report: ideation

- DONE: The task now names a bounded first-slice goal for Pi support instead of asking for undefined full parity.
  Evidence: `## Scope Boundary` defines first-class Pi support, worktrees, dispatch/wait/reuse/shutdown, and `tests/test_gate_guardrail.py --runtime pi` as the first proving target.
- DONE: The proposed approach selects a thin first-class Pi adapter over a larger runtime rewrite or a throwaway Pi-only mode.
  Evidence: `## Proposed Approach` defines Pi sessions as the canonical worker handle, plus a thin label-to-session mapping, runtime files, harness branch, and first live behavior target.
- DONE: Acceptance criteria and test plan express end-state properties with reproducible checks.
  Evidence: `## Acceptance criteria` pairs each end-state property with concrete verification, and `## Test Plan` separates static, unit/integration, harness, and live gate-preflight coverage.

### Summary

This ideation pass reframes #147 as a minimum-runtime baseline problem rather than a full-parity runtime port. The recommended first slice is a first-class Pi runtime adapter that supports worktrees, fresh dispatch, wait, same-worker reuse, and explicit shutdown, with `tests/test_gate_guardrail.py --runtime pi` as the first meaningful proof.

### Feedback Cycles

- Cycle 1 — validation gate rejected after dual review. Findings routed back to `implementation`.
  - Reviewer consensus flagged three serious issues: Pi FO path is not actually wired to `PiWorkerRuntime` / `PiSessionRegistry`; `PiWorkerRuntime.shutdown()` is metadata-only and does not shut down live/background workers; Pi ensign runtime docs conflict with the non-interactive reopen-by-session model.
  - Additional findings included concurrent reuse of an already-active worker session, missing session-path fallback in reuse, and non-atomic registry persistence.

## Stage Report: implementation

- DONE: Added first-slice Pi runtime contract surfaces for the FO, ensign, and pytest runtime selection.
  Evidence: `skills/first-officer/SKILL.md`, `skills/ensign/SKILL.md`, `skills/first-officer/references/pi-first-officer-runtime.md`, `skills/ensign/references/pi-ensign-runtime.md`, and `tests/conftest.py` now advertise/accept Pi.
- DONE: Added Pi harness helpers that use Pi JSON mode, explicit local skill loading, and per-run session storage for both the FO and manually constructed ensign dispatches.
  Evidence: `scripts/test_lib.py` now provides `build_pi_first_officer_invocation_prompt()`, `build_pi_ensign_invocation_prompt()`, `run_pi_first_officer()`, and `run_pi_ensign()`, and `tests/test_pi_runtime_harness.py` verifies the assembled command shapes use `pi --mode json --print --session-dir ... --skill ...` against the repo-local skills.
- DONE: Added a minimal session-backed worker mapping scaffold for Pi reuse/shutdown bookkeeping.
  Evidence: `scripts/pi_session_registry.py` defines `WorkerSessionRecord` and `PiSessionRegistry`, and `tests/test_pi_session_registry.py` verifies metadata round-trip, active-again epoch bumps, and shutdown/unroutable behavior without introducing a second session system.
- DONE: Live Pi gate-preflight workflow execution is now proven.
  Evidence: `tests/test_gate_guardrail.py --runtime pi -v` passes, `scripts/test_lib.py` now provides `PiLogParser`, and the gate test validates gate hold behavior plus explicit `gate review` / `waiting-for-approval` output from the Pi run.
- DONE: Reopened-session Pi reuse is now proven for the first-slice same-worker semantics without adding new session-registry machinery.
  Evidence: `tests/test_pi_reopened_session_reuse.py` drives three real `pi --mode json --print` turns against the same Pi session, first by session id and then by session file path. The assertions prove stable session identity, successful previous-turn recall, and growth of the same on-disk session file across follow-up turns.
- DONE: Added a thin Pi worker runtime that turns the registry-backed session handle into dispatch/reuse/shutdown behavior while staying Pi-session-native.
  Evidence: `scripts/pi_worker_runtime.py` introduces `PiWorkerRuntime` and `PiWorkerCompletion`. Dispatch creates the initial session-backed worker record, reuse reopens the same Pi session and bumps `completion_epoch`, `completion_is_current()` rejects stale pre-reuse completions, and shutdown marks the worker unroutable via the existing registry.
- DONE: The repo-local Pi ensign skill is now usable for manual session-backed worker dispatch and reopened follow-up.
  Evidence: `tests/test_pi_ensign_skill_reuse_live.py` drives the local `skills/ensign` skill through two real Pi turns on the same session. The first turn updates an entity file, appends `## Stage Report: implementation`, and commits; the reopened follow-up updates the same entity again, appends `## Stage Report: validation`, commits again, and stays on the same Pi session id.
- DONE: Same-worker routed follow-up, stale-completion rejection, and post-shutdown reroute blocking are now validated.
  Evidence: `tests/test_pi_worker_runtime.py` unit-tests the runtime bookkeeping against a fake Pi invoker, and `tests/test_pi_worker_runtime_live.py` proves the real Pi path can dispatch, reopen the same worker session for follow-up, reject the old completion as stale after the epoch bump, and refuse reuse after shutdown.
- SKIPPED: Explicit shutdown on a real reused Pi worker session still does not prove a native Pi close/kill primitive.
  Evidence: the slice now proves the workflow-visible shutdown semantics (`mark unroutable`, block reroute) but still relies on Pi's natural non-interactive process exit rather than a distinct Pi-native session termination API.
- DONE: Validation commands were rerun after the runtime slice landed.
  Evidence: `unset CLAUDECODE && uv run pytest tests/test_pi_runtime_contract.py tests/test_pi_session_registry.py tests/test_pi_runtime_harness.py tests/test_pi_reopened_session_reuse.py tests/test_pi_worker_runtime.py tests/test_pi_worker_runtime_live.py tests/test_pi_ensign_skill_reuse_live.py tests/test_gate_guardrail.py --runtime pi -v` passed with `18 passed` in the assigned worktree.

### Implementation Summary

This pass established the initial Pi integration seam rather than full runtime behavior. The current constructs are:

- a Pi FO launcher based on `pi --mode json --print`
- explicit skill loading via `--skill {repo}/skills/first-officer`
- per-run Pi session storage via `--session-dir {runner.test_dir}/pi-sessions`
- a thin Spacedock-owned worker-label -> Pi-session mapping (`PiSessionRegistry`) that records workflow metadata and a `completion_epoch` on top of Pi's native session persistence

This slice now proves the intended reopened-session path for Pi reuse: the same Pi session can be reopened by stable handle, answer a follow-up using prior-turn context, and continue appending to the same session file. That is enough to support Spacedock's first-slice same-worker semantics.

Remaining gap: explicit shutdown is still only a Spacedock metadata concern for Pi. We now prove the workflow-visible effect — shutdown makes the worker unroutable and blocks later reuse — but we do not yet prove a Pi-native close/kill primitive for a reused worker session. Optimization gap: the current proof uses reopened non-interactive sessions; SDK-managed in-memory keep-alive reuse is still the follow-up path if we later want lower-latency worker continuation.

Changed files in this slice:

- `scripts/test_lib.py`
- `scripts/pi_worker_runtime.py`
- `tests/test_pi_runtime_harness.py`
- `tests/test_pi_reopened_session_reuse.py`
- `tests/test_pi_worker_runtime.py`
- `tests/test_pi_worker_runtime_live.py`
- `tests/test_pi_ensign_skill_reuse_live.py`

## Stage Report: implementation

- DONE: Prove worktree-backed Pi ensign behavior via a focused live test that uses the repo-local Pi ensign skill
  Evidence: `tests/test_pi_ensign_skill_reuse_live.py` now creates a real `git worktree`, runs the repo-local `skills/ensign` skill with `worktree_path` set, and asserts the worktree entity changes while the main checkout copy stays unchanged.
- DONE: Keep helper/runtime changes minimal and aligned with Pi-native reopened sessions
  Evidence: only `tests/test_pi_ensign_skill_reuse_live.py` changed; the proof still uses the existing `run_pi_ensign()`/`build_pi_ensign_invocation_prompt()` helpers and reopens the same Pi session id for follow-up.
- DONE: Append a new implementation stage report section with evidence, commands run, changed files, and remaining gaps
  Evidence: this section is appended at the end of `docs/plans/pi-runtime-compatibility-baseline.md` and includes commands, changed files, and remaining gaps below.
- DONE: Run the focused relevant tests for your changes
  Evidence: `unset CLAUDECODE && uv run pytest tests/test_pi_worker_runtime_live.py tests/test_pi_ensign_skill_reuse_live.py -v` passed with `2 passed`.
- DONE: Commit the work
  Evidence: committed as `test: cover pi ensign worktree reopen live path`.

### Summary

I tightened the live Pi proof by making the repo-local ensign skill run in a real git worktree and then reopening the same Pi session for a follow-up turn on that same worktree. The assertions now show both session reuse and worktree isolation without expanding the runtime surface or helper layer beyond the existing Pi-native session helpers.

### Commands Run

- `unset CLAUDECODE && uv run pytest tests/test_pi_ensign_skill_reuse_live.py -v`
- `unset CLAUDECODE && uv run pytest tests/test_pi_worker_runtime_live.py tests/test_pi_ensign_skill_reuse_live.py -v`

### Changed Files

- `tests/test_pi_ensign_skill_reuse_live.py`
- `docs/plans/pi-runtime-compatibility-baseline.md`

### Remaining Gaps

- Pi shutdown is still proven only at the Spacedock metadata/routability layer, not via a distinct Pi-native session close primitive.
- The live worktree proof is focused on the ensign path; broader FO-driven worktree reuse coverage can still be expanded later if needed.
