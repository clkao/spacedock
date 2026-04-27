---
id: 216
title: "Codex FO runtime: add preemptible wait mode for subagent completions"
status: ideation
source: "GitHub issue #148, 2026-04-27 - local mitigation for openai/codex#15723 completion wakeup limitation"
started: 2026-04-27T18:27:05Z
completed:
verdict:
score: 0.72
worktree:
issue: "#148"
pr:
---

## Problem Statement

Codex currently does not wake the calling agent when background subprocesses or subagents complete (`openai/codex#15723`). That blocks the task-153 direction of making completion notifications preempt unrelated side discussion: if no new Codex turn is scheduled when a worker finishes, Spacedock cannot rely on a notification to make the first officer take autonomous follow-up action.

Spacedock still has a separate, local problem it can solve now. In interactive Codex sessions, task 138 established that workers should stay in the background by default unless the next orchestration step is blocked on their result. Task 140 established that completed gated or critical-path work becomes the next required action. Task 131 established that critical-path reused workers must be awaited on the same handle after `send_input`. The missing piece is what happens when the first officer intentionally decides that the next step is blocked and starts waiting, then the captain interrupts before the worker result is collected.

Today that wait intent is only implicit in the ongoing `wait_agent` call or in surrounding prose. If the captain interrupts the session with a question or instruction, the FO can lose the fact that it was still blocked on a specific unresolved worker handle. The result is either drift back into ordinary conversation or an accidental reliance on unsupported completion wakeups. The desired end state is an explicit, preemptible `wait_agent` mode: the FO records the live wait set, tells the captain what it is waiting on and that interruptions are allowed, treats non-stopping captain input as `preempted_by_user_input`, handles the input, and then resumes `wait_agent` on the same unresolved handle unless the captain explicitly pauses/stops or the interruption creates a clarification blocker.

Completion notifications remain useful as opportunistic evidence when Codex surfaces them, but they are not the authority for this task. Under the current upstream limitation, resumed `wait_agent` collection on the live handle is the authoritative path.

## Scope Boundary

In scope:

- Codex first-officer runtime guidance for an explicit preemptible wait mode when the entity's next orchestration step is blocked on worker results.
- The live wait-intent fields the FO must preserve: worker label, logical worker id, runtime handle, entity/stage, blocked reason, and whether the handle is still unresolved.
- Operator-facing wait status wording that leads with FO-owned worker labels, includes handles, names the blocked reason, and explicitly says interruptions are allowed.
- Outcome labels for wait attempts: `completed`, `timed_out`, `failed`, `preempted_by_user_input`, `paused_by_user`, and `clarification_required`.
- Static checks, parser helpers, and transcript fixtures that prove interrupted waits resume on the same unresolved handle without requiring Codex to wake the FO on completion.

Out of scope:

- Solving `openai/codex#15723` or depending on completion notifications to schedule a new turn.
- Reopening task 153's completion-notification preemption path while that upstream wakeup limitation remains open.
- Reversing task 138 by foregrounding every dispatched worker in interactive mode. Preemptible waiting applies only after the FO determines the next step is blocked on worker output or the captain explicitly asks it to wait.
- Changing shared first-officer scheduling semantics unless a tiny cross-reference is required. The behavioral contract is Codex-specific.
- Adding polling sleeps, background watcher scripts, or prompt-level behavioral coaching in the Codex invocation helper as the proof mechanism.

## Proposed Approach

1. Define `wait_agent` as an explicit Codex FO mode.

   Add a Codex runtime section in `skills/first-officer/references/codex-first-officer-runtime.md` that distinguishes three states:

   - background worker: spawned and tracked, but the current interactive turn is not blocked on it
   - preemptible wait: the FO is blocked on one or more unresolved handles and is intentionally calling `wait_agent`
   - post-wait completion handling: `wait_agent` returned completion evidence and task-140/task-131 rules decide the next workflow action

   The contract should say that entering preemptible wait requires a recorded wait intent before the `wait_agent` call. That intent includes the FO-owned worker label, `dispatch_agent_id`, runtime handle, entity path/id, stage name, blocked reason, and whether the wait came from a fresh dispatch or same-handle reuse after `send_input`.

2. Make operator-facing wait status concrete.

   Before waiting, the FO should emit a status line shaped like:

   ```text
   Waiting on `216-ideation/Ensign` (spacedock:ensign, handle: item_23) because the ideation gate cannot be prepared until the stage report returns. You can interrupt with questions or instructions; if you do not pause or stop the workflow, I will resume waiting on this same handle afterward.
   ```

   The exact wording can be adjusted by implementation, but the contract must preserve the content: label, logical id or handle, blocked reason, allowed interruption, and same-handle resume expectation.

3. Define interruption and resume semantics.

   If captain input arrives while a wait is unresolved, the FO treats the current wait attempt as `preempted_by_user_input`, not as `completed`, `failed`, or `timed_out`. It then handles the input according to ordinary FO rules:

   - If the captain says pause, stop, cancel, or changes the requested workflow target, the wait intent becomes `paused_by_user` or is superseded explicitly.
   - If the captain asks a question that can be answered without changing the blocked workflow, the FO answers and resumes `wait_agent` on the same unresolved worker handle.
   - If the captain's input dispatches additional worker work that becomes part of the same blocked next step, the FO updates the wait set and reports the new labels/handles before waiting again.
   - If the interruption reveals missing information required before waiting can continue, the mode becomes `clarification_required` and the FO must not pretend it is still waiting.

   This keeps user interruption support local and explicit without asking Codex to deliver unsupported autonomous wakeups.

4. Add focused tests and fixtures around the contract.

   Static runtime-content checks should live in `tests/test_agent_content.py` and assert that the Codex FO runtime names the preemptible wait mode, wait-intent fields, interruption outcome labels, same-handle resume rule, and unsupported-notification boundary.

   Parser or transcript tests should use `scripts/test_lib.py` / `CodexLogParser` or a small sibling helper if that keeps the assertions clean. The useful fixture shape is a synthetic Codex JSONL/transcript sequence with:

   - an initial `wait_agent`/`wait` call for handle `item_23`
   - a user interruption before that handle reports completion
   - an FO response that marks or narrates `preempted_by_user_input`
   - a resumed `wait_agent`/`wait` call with the same `receiver_thread_ids` value
   - eventual completion evidence for that same handle

   Existing live surfaces to verify or extend include `tests/test_codex_packaged_agent_e2e.py` for same-handle reuse expectations and `tests/test_test_lib_helpers.py` for parser-level fixtures. A new narrow fixture file is acceptable if the existing helpers do not naturally represent user interruption events.

5. Keep live E2E claims proportional.

   A true interactive Codex PTY test would be valuable if the harness can deterministically inject user input while a `wait_agent` call is pending. That should be treated as medium/high cost and optional for the first implementation slice unless the runtime already exposes a stable boundary. The required proof for this task can be a transcript fixture because the task is intentionally avoiding unsupported completion-wakeup behavior.

## Acceptance criteria

**AC-1 - The Codex first-officer runtime defines preemptible wait as a distinct mode from background workers and post-completion handling.**

Tested by static content checks in `tests/test_agent_content.py` against `skills/first-officer/references/codex-first-officer-runtime.md`, asserting the three states are present and that preemptible wait applies only when the next orchestration step is blocked or the captain explicitly asks to wait.

**AC-2 - The Codex wait-intent contract preserves enough identity to resume the same unresolved worker handle.**

Tested by static checks and a transcript/parser fixture asserting the wait intent includes worker label, logical id, runtime handle, entity/stage, blocked reason, and unresolved/completed state, and that resumed waits reuse the same `receiver_thread_ids` value after interruption.

**AC-3 - User-facing wait status includes FO-owned worker labels, runtime handles, the blocked reason, and an explicit interruption allowance.**

Tested by static wording checks or transcript fixture assertions that the wait status starts with or prominently includes the FO-owned worker label, includes the handle/logical id, says why the FO is blocked, and says the captain can interrupt while preserving the resume expectation.

**AC-4 - Non-stopping captain input during a pending wait is represented as `preempted_by_user_input` and resumes waiting on the same unresolved handle.**

Tested by a focused transcript or parser test where the initial wait on handle `H`, a user interruption, the FO answer, and the resumed wait on handle `H` appear in order before the handle's completion evidence. The test must fail if the resumed wait uses a replacement handle, treats the interruption as completion/failure, or drops the wait.

**AC-5 - Wait outcomes distinguish preemption from completion, timeout, failure, user pause/stop, and clarification blockers.**

Tested by static contract checks plus parser/fixture assertions for outcome labels: `completed`, `timed_out`, `failed`, `preempted_by_user_input`, `paused_by_user`, and `clarification_required`.

**AC-6 - The finished behavior does not depend on unsupported Codex completion wakeups.**

Tested by static checks that completion notifications are described as opportunistic evidence only, plus review of the focused fixture/test to confirm the authoritative collection path is `wait_agent` on live handles, not a simulated autonomous completion notification that schedules a new FO turn.

## Test Plan

Static contract tests are low cost and should be required:

- `uv run --with pytest python tests/test_agent_content.py -q`
- Assertions for preemptible wait mode, wait-intent fields, wait-status content, outcome labels, same-handle resume wording, and the explicit boundary around `openai/codex#15723`.

Parser and transcript fixture tests are low/medium cost and should be the primary behavioral proof:

- Add or extend a focused parser test, likely in `tests/test_test_lib_helpers.py`, using synthetic Codex JSONL/transcript events that show wait `H`, user interruption, `preempted_by_user_input`, resumed wait `H`, and eventual completion.
- If needed, add a small helper in `scripts/test_lib.py` or `CodexLogParser` to extract wait/preemption/resume sequences by handle without scattering ad hoc JSON traversal across tests.
- The fixture must not model completion notifications as the thing that wakes the FO; the proof is the preserved wait intent and explicit resumed `wait_agent`.

Live Codex E2E is optional for the first implementation slice and should only be claimed if the harness can deterministically inject input during a pending `wait_agent` call:

- Candidate surface: a new narrow Codex interactive/PTY harness test if `scripts/test_lib_interactive.py` or a sibling helper can expose an input-ready boundary during wait.
- Expected cost: medium/high, because task 153 already showed the completion-notification surface is upstream-blocked and current Codex interactive observability is fragile.
- If no stable live boundary exists, the implementation should state that the live E2E is deferred and rely on static plus transcript coverage rather than overclaiming.

Regression checks for related same-handle behavior should remain in scope when touched:

- `uv run --with pytest python tests/test_codex_packaged_agent_e2e.py -q` when live Codex credentials/runtime budget are available, because it already checks `send_input` followed by `wait` on the same worker handle.
- `uv run --with pytest python tests/test_rejection_flow.py --runtime codex` only if implementation changes feedback/reuse routing behavior. This task should not otherwise broaden into task 153 or shared rejection-flow scheduling.

E2E need: not mandatory for acceptance unless implementation claims real interactive interruption handling beyond the transcript fixture. The minimum acceptable proof is static contract coverage plus a deterministic fixture showing interruption-resume on the same unresolved handle without unsupported wakeups.

## Related

- GitHub issue: https://github.com/clkao/spacedock/issues/148
- Upstream Codex issue: https://github.com/openai/codex/issues/15723
- Related but not equivalent local task: `153` (`codex-completion-notifications-must-preempt-side-discussion`) depends on completion notification wakeup behavior that is currently upstream-blocked.
- Related local context: task `138` owns background-by-default interactive waits, task `140` owns completed gated/critical-path work becoming the next required action, task `131` owns same-handle waiting after critical-path reuse, and tasks `132`/`137` cover Codex visibility and FO-owned worker labels.

## Stage Report: ideation

- DONE: The design clearly distinguishes preemptible explicit waiting from upstream-blocked completion-notification preemption.
  Evidence: `## Problem Statement`, `## Scope Boundary`, and AC-6 separate `wait_agent` resume behavior from task 153 and `openai/codex#15723`.
- DONE: The proposed approach names concrete Codex runtime contract and test/fixture surfaces an implementation worker can edit or verify.
  Evidence: `## Proposed Approach` and `## Test Plan` name `codex-first-officer-runtime.md`, `tests/test_agent_content.py`, `scripts/test_lib.py`, `CodexLogParser`, `tests/test_test_lib_helpers.py`, and relevant live Codex tests.
- DONE: Acceptance criteria and test plan prove interruption-resume behavior on the same unresolved worker handle without requiring unsupported Codex wakeups.
  Evidence: AC-2, AC-4, AC-6, and the parser/transcript fixture plan require wait `H`, interruption, resumed wait `H`, and eventual completion without notification-driven wakeup.

### Summary

Fleshed out the entity into a Codex-specific preemptible wait design. The plan keeps task 153's notification-preemption path out of scope, makes `wait_agent` on live handles authoritative, and defines static plus fixture coverage for user interruption followed by same-handle wait resume.
