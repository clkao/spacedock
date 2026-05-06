---
id: 842ja5phzj5xspyternpww31
title: "status binary refuses terminal advancement on rejected/abandoned entities when merge hook is registered, even when no PR is intended"
status: validation
source: GitHub issue #188 (clkao/spacedock)
started: 2026-05-06T08:39:39Z
completed:
verdict:
score: 0.55
worktree: .worktrees/spacedock-ensign-status-refuses-terminal-on-rejected-entity-with-merge-hook
issue: "#188"
pr:
mod-block:
---

`status --set status={terminal} verdict=rejected completed` is refused when the workflow has any registered `## Hook: merge` mod AND `pr` is empty AND `mod-block` is empty. The captain MUST pass `--force` to bypass.

This is correct enforcement when an entity was meant to ship through a PR but skipped the hook. It misfires when the captain explicitly rejected the work — there is no PR to wait for, the merge hook should not run, and forcing `--force` makes the audit history harder to read ("did the captain bypass because they explicitly rejected, or did they fat-finger past a guard?").

Issue #188 lays out three candidate fix shapes:

1. `verdict=rejected` (or any non-PASSED verdict) implicitly bypasses the merge-hook gate; the verdict itself records "this is not a ship".
2. Recognize an explicit `verdict=abandoned` distinct from `rejected` and exempt only that verdict.
3. Document the `--force` requirement on the abandon path and improve the refusal message to suggest `--force` for rejection cases.

Ideation should pick one shape, weighing audit clarity vs. additional verdict surface area. Test plan must cover: (a) rejected-with-no-PR-and-merge-hook flow succeeds without `--force`; (b) ship-flow without `mod-block` still refuses (the original guard still fires); (c) audit-history clarity in both paths.

## Problem statement

The merge-hook terminal guard at `skills/commission/bin/status:2342-2358` refuses any terminal `--set` when the workflow has a registered `## Hook: merge` mod AND `pr` is empty AND `mod-block` is empty, requiring `--force` to bypass. The guard exists to catch the FO advancing a shipping entity to terminal without first running the merge hook (i.e., a PR was supposed to happen but the hook was skipped). It misfires on the abandon path: when the captain has decided to reject the work, no PR will ever exist, the merge hook should not run, and the only escape is `--force`. `--force` is overloaded — it is also the documented escape for stuck mod-blocks and other operator overrides — so its presence in the audit history loses the signal "this was a clean captain rejection" inside the noise of "operator bypassed something".

The captain's authoritative "this is not a ship" record is the verdict. `verdict=rejected` is exactly the signal the guard needs to recognize: the entity is being terminalized as not-shipping, so the hook does not apply.

## Decision: Shape 1 (verdict bypasses guard), narrowed to explicit `verdict=rejected`

Adopt shape 1 from the issue, with the bypass keyed to an explicit `verdict=rejected` value (current frontmatter value or being set in the same `--set` call). The guard remains intact for every other case.

### Why this shape

- Verdict is already the captain-set field that records the ship/no-ship decision. Tying the bypass to the verdict means the audit history reads cleanly: a row with `verdict=rejected` and no `--force` documents itself; a row with `--force` and no rejected verdict still reads as an operator override.
- The bypass widens only along an axis the captain already controls deliberately. The captain cannot trip it accidentally — verdicts are typed in by hand, not auto-derived.
- The original misfire-prevention case (PASSED-or-blank verdict, empty pr, empty mod-block, registered merge hook) is preserved: the bypass condition is "verdict is the literal string `rejected`", nothing else.

### Why narrowed to literal `rejected` rather than "any non-PASSED verdict"

Issue #188 phrases shape 1 as "`verdict=rejected` (or any non-PASSED verdict) implicitly bypasses". The "any non-PASSED" reading is too loose: blank is non-PASSED, and a captain who has not yet entered a verdict but is fat-fingering a terminal `--set` is exactly the case the guard was built for. Tying the bypass to the literal value `rejected` keeps the rule narrow, easy to reason about, and trivial to unit-test.

### Why not shape 2 (introduce `verdict=abandoned`)

Adding a new verdict vocabulary value to distinguish "rejected-with-PR" from "rejected-without-PR" splits a captain decision two ways for a reason internal to a guard. The captain's mental model is "I am rejecting this work" — they should not have to choose between two near-synonymous values to satisfy a guard implementation detail. A separate ideation `status-set-should-validate-stage-name-values.md` is moving toward tighter verdict validation; widening the verdict alphabet now would conflict with that direction. Reject.

### Why not shape 3 (doc-only / better error message)

Doc-only directly contradicts the load-bearing audit-clarity argument. `--force` in the git log remains ambiguous regardless of what the error message says, because the audit reader sees the commit, not the error. A better error message is a small win that should land regardless, but it does not solve the stated problem. Reject as the primary fix; the polished refusal message rides along as a follow-on (out of scope here).

## Implementation outline

In `skills/commission/bin/status`, in the `--set` flow around lines 2314-2358:

1. After resolving `post_update_pr`, resolve `post_update_verdict` the same way: current frontmatter value unless this `--set` itself sets `verdict=`.
2. In the merge-hook guard at line 2347, add a condition: `post_update_verdict == 'rejected'` skips the guard.
3. The mod-block guard above (lines 2321-2340) is left unchanged — that guard catches a different invariant (active mod-block in flight) and rejection does not justify trampling an unfinished mod hook.

## Acceptance criteria

Each criterion is an end-state property, with the test that verifies it.

- AC1: When the workflow registers a `## Hook: merge` mod and the entity has empty `pr`, empty `mod-block`, and `verdict=rejected` (already in frontmatter), `status --set <slug> status=<terminal> completed` exits 0 without `--force` and writes the terminal status.
  - Verified by: parser-level fixture test in `tests/test_status_script.py` (new method on `TestMergeHookTerminalGuard`).
- AC2: When the workflow registers a merge hook and the captain runs `status --set <slug> status=<terminal> verdict=rejected completed` in a single call (verdict not previously set), the call exits 0 without `--force` and writes verdict, status, and completed.
  - Verified by: parser-level fixture test in `tests/test_status_script.py`.
- AC3: When the workflow registers a merge hook and the entity has empty `pr`, empty `mod-block`, blank or PASSED verdict, `status --set <slug> status=<terminal>` still exits non-zero with the existing merge-hook refusal message, and the file is unchanged.
  - Verified by: existing `test_set_status_done_refused_when_merge_hook_and_no_pr` continues to pass; reinforced by an explicit `verdict=PASSED` fixture variant.
- AC4: When the workflow registers a merge hook, `status --set <slug> verdict=rejected` (alone, no other terminal field) on a non-terminal-status entity exits 0 without `--force`.
  - Verified by: parser-level fixture test in `tests/test_status_script.py`.
- AC5: The end-to-end captain rejection flow on a real merge-hook-enabled fixture terminalizes a rejected entity without `--force` appearing in any commit produced by the FO.
  - Verified by: extension to `tests/test_rejection_flow.py` (or sibling) covering the merge-hook-registered rejection variant; assertion inspects commit history for `--force` absence on the terminal commit.
- AC6: The end-to-end ship flow on the same fixture (PASSED-or-blank verdict, empty pr, empty mod-block) still produces the original refusal exit when the FO attempts to terminalize without running the hook.
  - Verified by: existing `tests/test_merge_hook_guardrail.py` continues to pass unchanged. No new E2E run is added for this branch — the existing coverage already exercises the guard-still-fires path.

## Test plan

### Parser-level fixtures (primary coverage)

Add five tests to `TestMergeHookTerminalGuard` in `tests/test_status_script.py`. The class already provides `_add_merge_hook`, `make_pipeline`, `entity`, and `run_status` helpers — new tests follow the same shape as `test_set_status_done_refused_when_merge_hook_and_no_pr`.

1. `test_set_terminal_allowed_when_verdict_already_rejected` — entity frontmatter has `verdict: rejected`, run `--set status=done completed`, expect exit 0 and `status=done` in file. Covers AC1.
2. `test_set_terminal_allowed_when_verdict_rejected_in_same_call` — entity has blank verdict, run `--set status=done verdict=rejected completed`, expect exit 0 and all three fields written. Covers AC2.
3. `test_set_terminal_refused_when_verdict_passed` — entity has blank verdict, run `--set status=done verdict=PASSED`, expect non-zero exit and `merge hook` in stderr. Covers AC3 (the PASSED variant). The blank-verdict variant is already covered by `test_set_status_done_refused_when_merge_hook_and_no_pr`.
4. `test_set_verdict_rejected_alone_allowed_with_merge_hook` — entity at non-terminal status with blank verdict, run `--set verdict=rejected`, expect exit 0 (the guard treats verdict as a terminal field, but rejected bypasses). Covers AC4.
5. `test_set_terminal_refused_when_verdict_already_passed` — entity frontmatter has `verdict: PASSED`, empty pr, run `--set status=done completed`, expect non-zero exit. Tightens AC3 against the symmetric "current verdict" path the implementation must check.

Cost: each test runs in milliseconds via the existing `run_status` harness; no live model calls. Total parser-level addition is well under one second of test time.

### End-to-end test (AC5)

Extend the rejection-flow E2E coverage. Two viable harness entry points exist:

- `tests/test_rejection_flow.py` (claude runtime) and `tests/test_rejection_flow_codex.py` already drive an FO through a captain-rejection scenario but on a fixture without a registered merge hook. Add a new fixture variant `tests/fixtures/rejection-flow-merge-hook/` that mirrors `tests/fixtures/rejection-flow/` plus a `_mods/pr-merge.md` registering `## Hook: merge`, then add a single `pytest.mark.live_claude` test that runs the captain-rejection prompt on this fixture. Assert: the entity reaches terminal `status=done` with `verdict=rejected`, AND the worktree's git log for the terminal commit does not include `--force` (read the FO log or inspect the commit message).
- Alternative: add a `_mods/pr-merge.md` to the existing rejection-flow fixture and toggle by env var, but adding a sibling fixture is cleaner and avoids cross-contaminating the existing test.

Cost: one additional live-claude test run, ~60-150s wallclock per the existing rejection-flow timings. E2E is justified because the load-bearing claim is "captain operates the terminal `--set` cleanly without `--force`" — that is a runtime-orchestration claim, not a parser claim, and the audit-clarity outcome is observable only at the commit-history level.

### What is explicitly NOT tested

- Verdict-value validation (covered by the separate `status-set-should-validate-stage-name-values.md` ideation).
- Refusal-message wording polish (out of scope; this ideation does not change the refusal text).
- Mod-block guard behavior on rejected entities (unchanged; existing tests retain coverage).

## Reviewer pass (independent)

Per dispatch instructions, the staff-review trigger fires (touches `skills/commission/bin/status` scaffolding). Self-conducted reviewer assessment follows.

**Design soundness.** Tying the bypass to `verdict=rejected` matches the captain's existing mental model and the existing audit fields. The guard's stated purpose ("entity meant to ship skipped the hook") is preserved verbatim because rejected is, by definition, not meant to ship. The narrow `verdict == 'rejected'` literal (vs. "any non-PASSED") prevents the blank-verdict regression that would otherwise reopen the original misfire.

**Test plan sufficiency.** Parser-level fixtures cover all four positive and negative branches of the new conditional (verdict-already-set rejected/PASSED × verdict-being-set rejected/PASSED-or-blank). The E2E addition covers the runtime orchestration claim that drove the issue. Existing `test_merge_hook_guardrail.py` continues to assert the ship-flow guard fires.

**Gaps and risks.**
- A captain who types `verdict=Rejected` (capitalized) would not bypass under literal-string matching. This is consistent with the rest of the binary, which treats verdict values as opaque strings, but worth flagging. Mitigation: the implementation will match the literal lowercase `rejected` to align with the issue's wording and the existing convention; capitalization variants remain refused. If captains hit this, a follow-on can normalize.
- Setting `verdict=rejected` then later setting `verdict=PASSED` and terminalizing without a hook would now refuse — which is the correct behavior, and is covered by AC3's symmetric variant (test 5).
- The mod-block guard (lines 2321-2340) is intentionally left untouched. A rejected entity with an active mod-block is a legitimate edge case the captain should resolve explicitly; bypassing both guards on rejection would over-reach.

**Verdict on ideation.** Approved as scoped. Implementation is a localized addition (one resolved variable + one condition) with proportionate test coverage; no scaffolding ripple beyond the binary itself.

## Stage Report: ideation

- DONE: Decision recorded picking ONE of the three fix shapes (implicit verdict-bypass, explicit `verdict=abandoned`, or doc-only) with reasoning a future reader can cross-check; the other two shapes named with rationale for rejecting them.
  Shape 1 chosen and narrowed to literal `verdict=rejected`; shapes 2 and 3 explicitly rejected with reasoning under "Why not shape 2/3".
- DONE: Acceptance criteria preserve the original misfire-prevention intent: rejected/no-PR entities terminalize without `--force`, AND the original guard still fires when intent IS to ship (PASSED-or-blank verdict + empty pr + empty mod-block + registered merge hook).
  AC1, AC2, AC4 cover the bypass; AC3, AC5, AC6 plus existing `test_set_status_done_refused_when_merge_hook_and_no_pr` and `test_merge_hook_guardrail.py` cover the original guard.
- DONE: Test plan names parser-level fixture coverage AND end-to-end `status --set` runs covering both flows (rejected-terminalize succeeds; ship-flow guard still refuses), specifying harness and expected exit states.
  Five parser-level tests on `TestMergeHookTerminalGuard` (`tests/test_status_script.py`); one new E2E in a sibling rejection-flow-merge-hook fixture; ship-flow coverage retained by existing `tests/test_merge_hook_guardrail.py`.

### Summary

Picked shape 1 (verdict-keyed bypass), narrowed to the literal value `rejected` to avoid loosening the guard against blank-verdict misfires. The fix is a localized `post_update_verdict` resolution plus one condition in the existing merge-hook guard, with primary coverage at the parser level via five fixture tests and one E2E run on a new merge-hook-enabled rejection-flow fixture. Shapes 2 and 3 rejected on vocabulary-bloat and audit-clarity grounds respectively.
