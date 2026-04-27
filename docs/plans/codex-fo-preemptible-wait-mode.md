---
id: 216
title: "Codex FO runtime: add preemptible wait mode for subagent completions"
status: validation
source: "GitHub issue #148, 2026-04-27 - local mitigation for openai/codex#15723 completion wakeup limitation"
started: 2026-04-27T18:27:05Z
completed:
verdict:
score: 0.72
worktree: .worktrees/spacedock-ensign-codex-fo-preemptible-wait-mode
issue: "#148"
pr:
mod-block: merge:pr-merge
---

## Problem Statement

Codex currently does not wake the calling agent when background subprocesses or subagents complete (`openai/codex#15723`). That blocks the task-153 direction of making completion notifications preempt unrelated side discussion: if no new Codex turn is scheduled when a worker finishes, Spacedock cannot rely on a notification to make the first officer take autonomous follow-up action.

Spacedock still has a separate, local problem it can solve now. In interactive Codex sessions, task 138 established that workers should stay in the background by default unless the next orchestration step is blocked on their result. Task 140 established that completed gated or critical-path work becomes the next required action. Task 131 established that critical-path reused workers must be awaited on the same handle after `send_input`. The missing piece is what happens when the first officer intentionally decides that the next step is blocked and starts waiting, then the captain interrupts before the worker result is collected.

If the captain interrupts the session with a question or instruction while the FO is waiting for ensigns, the FO can lose the fact that the workflow is still blocked on those same ensigns. The result is either drift back into ordinary conversation or an accidental reliance on unsupported completion wakeups. The desired behavior is simple: when the FO is waiting for ensigns and receives non-stopping captain input, it processes that input and then resumes `wait_agent` for the same unresolved ensigns unless the captain explicitly pauses/stops or the interruption creates a clarification blocker.

Completion notifications remain useful as opportunistic evidence when Codex surfaces them, but they are not the authority for this task. Under the current upstream limitation, resumed `wait_agent` collection on the live handle is the authoritative path.

## Scope Boundary

In scope:

- Minimal Codex first-officer runtime guidance for interrupted waits.
- Behavioral transcript/parser coverage proving interruption handling resumes `wait_agent` on the same unresolved ensign handles.
- The boundary that completion notifications are useful context only and do not replace resumed `wait_agent` collection.

Out of scope:

- Solving `openai/codex#15723` or depending on completion notifications to schedule a new turn.
- Reopening task 153's completion-notification preemption path while that upstream wakeup limitation remains open.
- Reversing task 138 by foregrounding every dispatched worker in interactive mode. Interrupted-wait handling applies only after the FO is already waiting because the next step is blocked on worker output or the captain explicitly asked it to wait.
- Changing shared first-officer scheduling semantics unless a tiny cross-reference is required. The behavioral guidance is Codex-specific.
- Adding polling sleeps, background watcher scripts, or prompt-level behavioral coaching in the Codex invocation helper as the proof mechanism.

## Proposed Approach

1. Amend Codex runtime guidance with the intended FO behavior.

   Add a short Codex runtime instruction: if the FO is waiting for ensigns and the captain sends a non-stopping message, handle that input and then resume `wait_agent` for the same unresolved ensigns unless the captain says pause or stop. If the input creates a clarification blocker, ask for clarification instead of continuing to wait. Completion notifications during the interruption are useful context only; they do not replace resumed `wait_agent` collection on the same worker handles.

2. Add focused transcript/parser coverage.

   Parser or transcript tests should use `scripts/test_lib.py` / `CodexLogParser` or a small sibling helper if that keeps the assertions clean. The useful fixture shape is a synthetic Codex JSONL/transcript sequence with:

   - an initial `wait_agent`/`wait` call for handles `item_23` and `item_42`
   - a user interruption while the FO is waiting
   - an FO response showing it processed the interruption
   - a resumed `wait_agent`/`wait` call with the same `receiver_thread_ids` values
   - eventual completion evidence for those same handles, including the case where one worker completed during the interruption but still had to be collected by the resumed wait

   Existing live surfaces to verify or extend include `tests/test_codex_packaged_agent_e2e.py` for same-handle reuse expectations and `tests/test_test_lib_helpers.py` for parser-level fixtures. A new narrow fixture file is acceptable if the existing helpers do not naturally represent user interruption events.

3. Keep live E2E claims proportional.

   A true interactive Codex PTY test would be valuable if the harness can deterministically inject user input while a `wait_agent` call is pending. That should be treated as medium/high cost and optional for the first implementation slice unless the runtime already exposes a stable boundary. The required proof for this task can be a transcript fixture because the task is intentionally avoiding unsupported completion-wakeup behavior.

## Acceptance criteria

**AC-1 - The Codex first-officer runtime contains only a minimal interrupted-wait behavior amendment.**

Verified by review of `skills/first-officer/references/codex-first-officer-runtime.md`: it should instruct the FO to process non-stopping captain input during a wait and then resume waiting for the same unresolved ensigns, without defining a detailed Codex wait-state schema.

**AC-2 - Interruption during waiting resumes `wait_agent` on the same unresolved ensign handles.**

Tested by a transcript/parser fixture asserting the initial wait handles, user interruption, FO handling response, and resumed wait handles appear in order and that the resumed wait uses the same `receiver_thread_ids` values.

**AC-3 - Completion notifications do not replace resumed wait collection.**

Tested by the same transcript/parser fixture including a completion notification during the interruption before the resumed `wait_agent`; completion is considered collected only after the resumed wait returns completion evidence.

**AC-4 - Pause/stop and clarification blockers remain exceptions to resumed waiting.**

Verified by runtime guidance review: if the captain says pause or stop, the FO should not resume waiting; if the interruption creates a clarification blocker, it should ask rather than continue waiting.

**AC-5 - Prose-doc tests are not used to prove the new wait behavior.**

Verified by test review: `tests/test_agent_content.py` must not include a prose-matching test for this interrupted-wait behavior; behavioral proof lives in parser/transcript tests.

**AC-6 - The finished behavior does not depend on unsupported Codex completion wakeups.**

Tested by review of the focused fixture/test to confirm the authoritative collection path is resumed `wait_agent` on live handles, not a simulated autonomous completion notification that schedules a new FO turn or replaces FO reconciliation.

## Test Plan

Static regression tests remain useful for unrelated runtime structure, but they are not the proof for this behavior:

- `uv run --with pytest python tests/test_agent_content.py -q`
- Do not add or keep prose-doc assertions for the interrupted-wait behavior. In particular, do not check for a detailed wait-state schema, exact user-facing wait wording, or runtime-doc prose as the behavioral proof.

Parser and transcript fixture tests are low/medium cost and should be the primary behavioral proof:

- Add or extend a focused parser test, likely in `tests/test_test_lib_helpers.py`, using synthetic Codex JSONL/transcript events that show wait handles `{H1, H2}`, user interruption, FO processing the interruption, resumed wait on `{H1, H2}`, and eventual reconciliation of completion evidence.
- If needed, add a small helper in `scripts/test_lib.py` or `CodexLogParser` to extract wait/preemption/resume sequences by handle without scattering ad hoc JSON traversal across tests.
- The fixture must include the case where one worker completes during the interruption but remains unresolved until the resumed wait. It must not model completion notifications as the thing that wakes the FO; the proof is the explicit resumed `wait_agent`.
- Parser/transcript assertions should prove same-handle resumed waiting and final resolution after the resumed wait. They may include user-facing wait text as fixture context, but should not assert exact phrasing.

Live Codex E2E is optional for the first implementation slice and should only be claimed if the harness can deterministically inject input during a pending `wait_agent` call:

- Candidate surface: a new narrow Codex interactive/PTY harness test if `scripts/test_lib_interactive.py` or a sibling helper can expose an input-ready boundary during wait.
- Expected cost: medium/high, because task 153 already showed the completion-notification surface is upstream-blocked and current Codex interactive observability is fragile.
- If no stable live boundary exists, the implementation should state that the live E2E is deferred and rely on static plus transcript coverage rather than overclaiming.

Regression checks for related same-handle behavior should remain in scope when touched:

- `uv run --with pytest python tests/test_codex_packaged_agent_e2e.py -q` when live Codex credentials/runtime budget are available, because it already checks `send_input` followed by `wait` on the same worker handle.
- `uv run --with pytest python tests/test_rejection_flow.py --runtime codex` only if implementation changes feedback/reuse routing behavior. This task should not otherwise broaden into task 153 or shared rejection-flow scheduling.

E2E need: not mandatory for acceptance unless implementation claims real interactive interruption handling beyond the transcript fixture. The minimum acceptable proof is a deterministic fixture showing interruption-resume on the same unresolved ensign handles without unsupported wakeups.

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
- DONE: Acceptance criteria and test plan prove interruption-resume behavior on the same FO-uncollected wait set without requiring unsupported Codex wakeups.
  Evidence: AC-2, AC-4, AC-6, and the parser/transcript fixture plan require wait set `{H1, H2}`, interruption, resumed wait for FO-uncollected entries, and eventual reconciliation without notification-driven wakeup.

### Summary

Fleshed out the entity into a Codex-specific preemptible wait design. The plan keeps task 153's notification-preemption path out of scope, makes `wait_agent` on live handles authoritative, and defines static plus fixture coverage for user interruption followed by same-wait-set resume and reconciliation.

## Stage Report: ideation (cycle 2)

- DONE: Make the wait-mode communication explicit that interruption/resume state should be clear.
  Evidence: This earlier cycle added detailed wait communication; cycle 2 repair supersedes any exact wording contract and keeps only behavioral guidance.
- DONE: Multi-handle wait-set coverage is implied but not strongly tested.
  Evidence: `## Scope Boundary`, AC-2, AC-4, and the parser/transcript fixture plan now require every label and internal handle in a plural wait set to be preserved and resumed.
- DONE: Define `unresolved` as FO-uncollected, not necessarily still running.
  Evidence: `## Proposed Approach` defines unresolved as FO-uncollected and requires resumed `wait_agent` collection even when a worker completes during the interruption.
- DONE: Clarify that runtime handles belong in internal wait intent/logs/tests for same-handle verification.
  Evidence: Cycle 2 repair keeps internal-handle preservation and removes user-facing prose matching as an acceptance target.

### Summary

Revised the ideation spec for the captain and staff review feedback. Cycle 2 repair supersedes the earlier wording-specific communication target; the testable behavior remains plural wait-set preservation with FO-uncollected completion reconciliation.

## Stage Report: implementation

- DONE: Codex runtime guidance implements the approved preemptible-wait contract, including plural wait sets and FO-uncollected unresolved semantics.
  Evidence: `skills/first-officer/references/codex-first-officer-runtime.md` adds `## Codex Preemptible Wait Mode`; cycle 2 repair removes the rejected prose-specific contract from current requirements.
- DONE: Static and/or parser fixture coverage proves interruption-resume on the same FO-uncollected wait set, including a multi-handle case, without depending on unsupported Codex completion wakeups.
  Evidence: `tests/test_agent_content.py`, `tests/test_test_lib_helpers.py`, and `scripts/test_lib.py` cover the contract plus a two-handle resumed wait after `preempted_by_user_input`; completion notification is only side-channel evidence.
- DONE: Verification evidence is sufficient for independent validation: changed files, commands run, and any live-E2E deferral rationale are recorded in the stage report.
  Evidence: Changed files are `docs/plans/codex-fo-preemptible-wait-mode.md`, `skills/first-officer/references/codex-first-officer-runtime.md`, `scripts/test_lib.py`, `tests/test_agent_content.py`, and `tests/test_test_lib_helpers.py`; commands run: `uv run --with pytest python tests/test_agent_content.py -q` (49 passed), `uv run --with pytest python -m pytest tests/test_test_lib_helpers.py -q` (19 passed), and `git diff --check` (passed). Live Codex E2E was not run because this task's accepted proof is static plus deterministic transcript coverage and unsupported completion wakeups must not be simulated as the authority.

### Summary

Implemented the Codex preemptible wait contract locally in the FO runtime docs and added deterministic test coverage for interrupted same-wait-set resume. The parser fixture proves a plural wait set resumes on the same FO-uncollected handles even when a completion notification appears during the interruption.

## Stage Report: validation

- DONE: Applicable validation commands are selected from `tests/README.md`, rerun in the worktree, and reported with exact pass/fail results.
  Evidence: `uv run --with pytest python tests/test_agent_content.py -q` -> 49 passed; `uv run --with pytest python -m pytest tests/test_test_lib_helpers.py -q` -> 19 passed; `git diff --check` -> passed; `make test-static` -> 514 passed, 25 deselected, 10 subtests passed.
- DONE: Every AC-1 through AC-6 is verified with concrete evidence or explicitly failed; the review checks for unsupported Codex wakeup reliance and wait-set reconciliation.
  Evidence: AC-1 through AC-6 are accounted for below; no failing criteria found. The entity has `Tested by` clauses rather than literal `Verified by` clauses, so validation reproduced the cited test evidence from those clauses.
- DONE: The validation report gives a clear PASSED or REJECTED recommendation and preserves implementation ownership by not making unrequested fixes.
  Evidence: No implementation files were changed during validation; this stage report is the only validation edit.

### Acceptance Criteria

- AC-1: PASSED. `skills/first-officer/references/codex-first-officer-runtime.md` defines `background worker`, `preemptible wait`, and `post-wait completion handling`; `tests/test_agent_content.py` asserts those states and the blocked-step/explicit-wait boundary.
- AC-2: PASSED. The runtime wait-intent contract lists worker label, `dispatch_agent_id`, runtime handle, entity path/id, stage name, blocked reason, collection state, and source; `tests/test_test_lib_helpers.py` proves resumed waits reuse `["item_23", "item_42"]`.
- AC-3: SUPERSEDED. This validation was based on the rejected prose-specific AC-3; current AC-3 focuses on blocked wait/resume state and avoids exact user-facing wait-text assertions.
- AC-4: PASSED. The parser fixture orders initial wait, user interruption, `preempted_by_user_input`, completion notification side-channel evidence, resumed wait on the same handles, and final collection evidence with no drops or replacements.
- AC-5: PASSED. Static checks cover `completed`, `timed_out`, `failed`, `preempted_by_user_input`, `paused_by_user`, and `clarification_required`.
- AC-6: PASSED. Runtime text says completion notifications are opportunistic only and do not schedule an autonomous FO turn; parser helper only treats completion as collected after a resumed wait on live handles.

### Summary

Validation selected the static/parser commands called out by the entity and the stable offline suite from `tests/README.md`. Live Codex E2E was not run because the entity states it is optional for this slice unless real interactive interruption handling is claimed; the implementation claims static plus deterministic transcript coverage.

Recommendation: PASSED

### Feedback Cycles

- Cycle 1 (2026-04-27): Captain rejected the validation gate despite a PASSED recommendation. Required changes:
  - Do not make detailed wait-instruction prose a first-class implementation or test target; Codex should already provide the basic interrupt affordance.
  - Focus the implementation and tests on continuous waiting behavior after interruption: preserve the wait set, process the interruption, and resume waiting for unresolved ensigns unless the captain says pause or stop.
  - Avoid static tests that match user-facing prose. Prefer behavioral/parser fixtures that prove same-wait-set resume and FO-collected/uncollected state.
- Cycle 2 (2026-04-27): Captain rejected the validation gate again. Required changes:
  - Do not re-describe Codex's internal wait representation in runtime docs or ACs.
  - Amend the intended first-officer behavior directly: after user interruption during a blocked wait, process the input and resume waiting for the same unresolved ensigns unless the captain says pause or stop.
  - Remove prose-doc tests such as `test_codex_runtime_docs_define_preemptible_wait_mode_contract`; keep tests on behavior/parser evidence rather than runtime-doc wording.

## Stage Report: implementation (cycle 2)

- DONE: Detailed wait-instruction prose is no longer a static/prose-matching contract; runtime docs focus on behavior and wait-set continuity.
  Evidence: `skills/first-officer/references/codex-first-officer-runtime.md` removes wording-specific status rules, replacing them with behavioral wait/resume guidance.
- DONE: Tests prove continuous waiting after interruption through parser/transcript behavior, not exact user-facing prose matching.
  Evidence: `tests/test_agent_content.py` no longer checks exact wait-status wording; `tests/test_test_lib_helpers.py` and `scripts/test_lib.py` assert same-handle resume, initial FO-uncollected handles, FO-collected handles after resumed wait, and no dropped/replacement handles.
- DONE: Verification evidence is recorded with commands and results.
  Evidence: `uv run --with pytest python tests/test_agent_content.py -q` (49 passed), `uv run --with pytest python -m pytest tests/test_test_lib_helpers.py -q` (19 passed), and `git diff --check` (passed).

### Summary

Reworked the cycle 1 implementation to keep Codex preemptible wait behavioral rather than prose-driven. The runtime contract now centers on preserving and resuming the FO-uncollected wait set, and the fixture proof covers continuous waiting through interruption without relying on exact user-facing wait text.

## Stage Report: implementation (cycle 2 repair)

- DONE: Stale prose-contract language in the entity body was removed or reframed.
  Evidence: Current `## Scope Boundary`, `## Proposed Approach`, AC-3, and `## Test Plan` now state that detailed wait-instruction prose is not contractual and that behavioral/parser proof owns wait continuity.
- DONE: Tests/runtime docs were checked for exact user-facing wait-prose matching and remain behavior-focused.
  Evidence: `skills/first-officer/references/codex-first-officer-runtime.md`, `tests/test_agent_content.py`, `tests/test_test_lib_helpers.py`, and `scripts/test_lib.py` have no remaining exact wait-hint or wait-status prose assertions; static tests remain structural and parser tests assert same-wait-set resume plus FO-collected/uncollected state.
- DONE: Verification commands were rerun or a clear rationale is given for targeted verification.
  Evidence: Targeted verification rerun for this docs-alignment repair: wait-prose grep over runtime/tests/scripts found no matches, `uv run --with pytest python tests/test_agent_content.py -q` passed, `uv run --with pytest python -m pytest tests/test_test_lib_helpers.py -q` passed, and `git diff --check` passed.

### Summary

Repaired the entity body so the accepted requirements match the captain feedback: exact wait-instruction wording is no longer a contract, while wait-set continuity remains the tested behavior. Runtime docs and tests were checked again and remain focused on preserving/resuming FO-uncollected waits rather than matching user-facing prose.

## Stage Report: validation (cycle 2)

- DONE: Applicable validation commands are selected from `tests/README.md`, rerun in the worktree, and reported with exact pass/fail results.
  Evidence: `uv run --with pytest python tests/test_agent_content.py -q` -> 49 passed; `uv run --with pytest python -m pytest tests/test_test_lib_helpers.py -q` -> 19 passed; `git diff --check` -> passed; `make test-static` -> 514 passed, 25 deselected, 10 subtests passed.
- DONE: Every AC-1 through AC-6 is verified with concrete evidence or explicitly failed, including the feedback-cycle requirement to avoid static user-facing prose matching.
  Evidence: AC-1 through AC-6 are accounted for below; `rg` over `tests/test_agent_content.py`, `tests/test_test_lib_helpers.py`, and `scripts/test_lib.py` found no exact wait-instruction prose patterns.
- DONE: The validation report gives a clear PASSED or REJECTED recommendation and preserves implementation ownership by not making unrequested fixes.
  Evidence: No implementation files were changed; this validation edit appends only this entity report.

### Acceptance Criteria

- AC-1: PASSED. `skills/first-officer/references/codex-first-officer-runtime.md` defines `background worker`, `preemptible wait`, and `post-wait completion handling`; `tests/test_agent_content.py` asserts those states and the blocked-step/explicit-wait boundary.
- AC-2: PASSED. The runtime wait-intent contract lists worker label, `dispatch_agent_id`, runtime handle, entity path/id, stage name, blocked reason, collection state, and source; `tests/test_test_lib_helpers.py` proves resumed waits reuse `["item_23", "item_42"]`.
- AC-3: PASSED. Runtime docs communicate blocked wait/resume state while saying not to make a precise instruction sentence contractual; tests avoid exact user-facing wait wording and verify blocked wait-set behavior instead.
- AC-4: PASSED. The parser fixture orders initial wait on `item_23`/`item_42`, user interruption, `preempted_by_user_input`, side-channel completion notification, resumed wait on the same FO-uncollected handles, and final collection with no dropped or replacement handles.
- AC-5: PASSED. Static checks cover `completed`, `timed_out`, `failed`, `preempted_by_user_input`, `paused_by_user`, and `clarification_required`.
- AC-6: PASSED. Runtime docs state completion notifications are opportunistic only and do not schedule an autonomous FO turn; parser helper only marks completion collected after resumed wait on live handles.

### Feedback-Cycle Verdict

PASSED. Detailed wait-instruction prose is not a contract or static prose-matching target; the surviving proof is behavioral/parser coverage for preserving the wait set, handling interruption, and resuming `wait_agent` for FO-uncollected entries unless paused/stopped or clarification-blocked.

### Summary

Validation reproduced the cited static and parser evidence, then ran the stable offline suite from `tests/README.md`. Live Codex E2E was not run because the entity treats it as optional unless deterministic interactive interruption handling is claimed, and this implementation claims static plus deterministic transcript coverage.

Recommendation: PASSED

## Stage Report: implementation (cycle 3)

- DONE: Runtime guidance was simplified to an intended-behavior amendment rather than a detailed Codex wait-state contract.
  Evidence: `skills/first-officer/references/codex-first-officer-runtime.md` now has a short `## Interrupted Waits` rule and no detailed wait-intent schema.
- DONE: Prose-doc tests for the new wait behavior were removed; behavioral/parser tests prove resumed waiting after interruption on the same unresolved ensigns.
  Evidence: `tests/test_agent_content.py` no longer contains `test_codex_runtime_docs_define_preemptible_wait_mode_contract` or the continuity/outcome prose test; `tests/test_test_lib_helpers.py` uses `CodexLogParser.interrupted_wait_sequences()` to prove interruption, same-handle resumed wait, notification-as-context, and final resolution.
- DONE: Verification evidence is recorded with commands and results.
  Evidence: `uv run --with pytest python tests/test_agent_content.py -q` (47 passed), `uv run --with pytest python -m pytest tests/test_test_lib_helpers.py -q` (19 passed), targeted schema/prose-test grep over runtime/tests/scripts found no matches, and `git diff --check` passed.

### Summary

Reworked the implementation around the captain's narrower behavior: interrupted FO waits resume on the same unresolved ensign handles unless the captain pauses/stops or creates a clarification blocker. Removed the runtime-doc prose tests and stripped the parser fixture/helper of the detailed wait-intent schema.
