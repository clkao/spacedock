---
id: na9ydw9h5cvey0qmqeb96te8
title: test_feedback_keepalive dispatch-count assertion is stale under fresh validation
status: validation
source: captain (CL) — observed on PR #208 cycle-3 live-CI (claude-live-opus job)
started: 2026-05-12T18:47:19Z
completed:
verdict:
score:
worktree: .worktrees/spacedock-ensign-test-feedback-keepalive-count-assertion-stale
issue: "#209"
mod-block: 
pr: #210
---

`tests/test_feedback_keepalive.py::test_feedback_keepalive` asserts the FO emits *exactly two* ensign `Agent()` dispatches under the feedback-keepalive flow. The current FO contract emits three because validation has `fresh: true`, forcing a fresh validation `Agent()` after the kept-alive implementer's fix lands.

Observed on PR #208 `claude-live-opus` run: 6 of 7 checks pass; the failing one is:

```
FAIL: FO emitted exactly two ensign Agent() dispatches (impl + validation; feedback via SendMessage)
  dispatch records: [('Implementation: greeting.txt', 30.2), ('Validation', 22.6), ('Validation cycle 2', 0.0)]
```

The other six checks — including "feedback routed via SendMessage to kept-alive implementation ensign" and the three static-template greps for shared-core feedback-flow rules — all pass. Only the dispatch-count assertion is stale.

Full GitHub issue: #209.

## What the test was designed to catch

The original bug: instead of `SendMessage`-routing the fixup to the kept-alive implementation ensign, the FO fresh-dispatches a *new* implementation `Agent()`. Under that bug the dispatch count would be four (impl, validation, *new impl*, re-validation) — distinct from both the legitimate-today count (three) and the obsolete-expected count (two).

## What the contract emits today

1. `Agent()` #1 — implementation (kept alive for `feedback-to: implementation`)
2. `Agent()` #2 — validation (rejects, recommends feedback)
3. SendMessage → kept-alive #1 with fixup; #1 commits the fix
4. `Agent()` #3 — re-validation, fresh because `fresh: true`

## Resolution directions (decide in ideation)

- **Option A: per-stage breakdown.** Replace the single dispatch-count assertion with three sub-assertions: exactly 1 `Agent()` whose name suffix is `implementation`, exactly 2 `Agent()`s whose name suffix is `validation`, and the SendMessage-to-kept-alive assertion (already separately checked). Preserves discrimination against the original bug (under which impl-count would be 2 and validation-count would still be 2 — fails the impl-count check).
- **Option B: rebrand to a fresh-aware total.** Assert `total Agent() dispatches == 3` and `count(name-ending-in-implementation) == 1`. Tighter but still catches the buggy scenario (impl-count = 2). Simpler than A but loses the separate validation-count signal if a future contract change adds a third validation cycle for a legitimate reason.
- **Option C: assert behavior, not counts.** Drop the dispatch-count check entirely; rely on the SendMessage-to-kept-alive positive assertion plus a NO-Agent()-with-implementation-name-after-validation negative assertion. Most robust to future contract drift but harder to read.

## Out of scope

- The other failing job on PR #208 (`claude-live` — `test_standing_teammate_spawn` step timeout: TeamCreate not emitted within 60s) is a separate concern, likely a CI flake. Track separately if it persists.
- Changing the FO contract itself (e.g., trying to make re-validation reuse instead of fresh-dispatch). The `fresh: true` on validation is intentional and not in scope here; this task only updates the test to reflect the contract as written.

## Chosen approach: single-stage impl-count check

Replace the single `len(records) == 2` assertion in `tests/test_feedback_keepalive.py` with one count assertion keyed on the dispatch's `ensign_name` (the Agent `description` field; `test_lib.DispatchRecord.ensign_name` at `scripts/test_lib.py:1155`):

- exactly one record whose `ensign_name` lowercased contains `"implementation"`

The kept-alive SendMessage assertion already lives at lines 164-168 of the test and stays unchanged. The per-dispatch budget assertion also stays unchanged.

### Why impl-count alone (degenerate Option A from cycle 1)

Cycle 1 implemented the full Option-A pair (`impl-count == 1` AND `validation-count == 2`). Local live-E2E on `claude+opus` rejected the validation-count half — see the cycle-1 entry under `### Feedback Cycles` for the captain's narrative. The local FO emitted only one validation-suffixed Agent() dispatch; CI's third dispatch had elapsed=0.0s which is a harness extraction artifact, not a real spawn. Either reading kills `validation-count == 2`.

The actually-load-bearing discriminator is `impl-count == 1`:

| Scenario | impl-count |
|---|---|
| Legitimate (current FO contract) | 1 (PASS) |
| Original bug (FO fresh-dispatches new impl Agent instead of SendMessage routing) | 2 (FAIL) |

The SendMessage-to-kept-alive assertion (separately checked, lines 164-168) catches the *positive* side of the same contract. Together they give crisp regression coverage of the original bug without depending on the brittle validation-count signal.

## Acceptance criteria

End-state properties of `tests/test_feedback_keepalive.py` after the edit:

1. **AC-1: impl-count assertion exists at value 1.** The test file contains a `t.check(...)` whose boolean is `sum(1 for r in records if "implementation" in r.ensign_name.lower()) == 1` (or equivalent comprehension). *Verified by:* `grep -n 'implementation' tests/test_feedback_keepalive.py` shows the count expression on a `t.check` line; comprehension evaluation against the two fixture lists in AC-3 returns `True, False` (legit, bug) respectively.

2. **AC-2: stale total-count assertion removed; validation-count assertion not present.** The exact string `len(records) == 2` no longer appears in the file, and no `t.check` line asserts a count on `"validation"`-named records. *Verified by:* `grep -c 'len(records) == 2' tests/test_feedback_keepalive.py` returns 0; `grep -c 'validation-suffixed' tests/test_feedback_keepalive.py` returns 0.

3. **AC-3: rewritten check passes against current FO contract, fails against original bug.** Two synthetic `DispatchRecord` fixtures are exercised as unit tests in `tests/test_feedback_keepalive_helpers.py`:
   - **Fixture L (legitimate today):** `[("Implementation: greeting.txt", 30.2), ("Validation", 22.6)]` — impl-count check PASSES.
   - **Fixture B (original bug):** `[("Implementation: greeting.txt", 30.2), ("Validation", 22.6), ("Implementation cycle 2: greeting.txt", 12.0), ("Validation cycle 2", 0.0)]` — impl-count check FAILS (counts 2).
   *Verified by:* `pytest tests/test_feedback_keepalive_helpers.py -v` shows the new test cases green.

4. **AC-4: live `test_feedback_keepalive` is no longer blocked by the count assertion.** The Phase 3 block produces no `FAIL: FO emitted exactly two ensign Agent() dispatches…` line on a successful live run; the new impl-count check reports PASS. *Verified by:* PR re-run of `claude-live-opus` CI shows the feedback-keepalive job green; the prior failing assertion text no longer appears in the test output.

## Test plan

Two layers, sized to the claim. The claim is **test-assertion logic**, not FO runtime behavior — so unit-level fixture proof carries most of the weight; live E2E is a downstream confirmation, not the proof bearer.

### Layer 1 — unit (fixture-based, the load-bearing proof)

Two test cases in `tests/test_feedback_keepalive_helpers.py` (the existing helper-unit-test file). Each constructs a synthetic `list[DispatchRecord]` and exercises the impl-count expression directly. Pattern:

```python
from test_lib import DispatchRecord

def _impl_count(records): return sum(1 for r in records if "implementation" in r.ensign_name.lower())

def test_passes_against_current_contract():
    records = [DispatchRecord("Implementation: greeting.txt", 30.2),
               DispatchRecord("Validation", 22.6)]
    assert _impl_count(records) == 1

def test_catches_original_bug_on_impl_count():
    records = [DispatchRecord("Implementation: greeting.txt", 30.2),
               DispatchRecord("Validation", 22.6),
               DispatchRecord("Implementation cycle 2: greeting.txt", 12.0),
               DispatchRecord("Validation cycle 2", 0.0)]
    assert _impl_count(records) != 1   # fails the ==1 check → catches the bug
```

Cost: seconds. Real Python, real `DispatchRecord` import, no subprocess, no Claude runtime.

### Layer 2 — live E2E (downstream confirmation, not proof)

Re-run `tests/test_feedback_keepalive.py::test_feedback_keepalive` on the PR with the assertion edit:

- Local invocation: `uv run pytest tests/test_feedback_keepalive.py -v -m live_claude` against the current FO + ensign stack. Expected: Phase 3 shows the impl-count PASS line instead of the prior FAIL line. Skip is acceptable if the Claude runtime probe fails locally.
- Live CI confirmation: `claude-live-opus` job on the PR — feedback-keepalive case green.

Cost: ~3-5 minutes wallclock per run. Run once on the PR; no need to re-run if Layer 1 stays green.

E2E is intentionally *not* the proof for the assertion logic — it depends on real FO output, which the unit fixtures already abstract away. If Layer 1 is green and Layer 2 is red, the divergence is FO contract drift (a separate task), not a defect in this fix.

### Why not other proof shapes

- **Static prose grep of the assertion strings.** Tempting (AC-1 already does this for structure), but a grep can't prove the count expression returns the right verdict on a buggy-FO input. Fixture exercise does.
- **Transcript-fixture replay through the live test code path.** Would require synthesising a fake JSONL stream and a fake FO subprocess. Massive over-instrumentation for a one-line count change — the assertion is pure Python over a list of dataclasses; exercising it directly is the right granularity.

## Risk and discrimination notes

- The `ensign_name` substring match is case-insensitive (`.lower()`) and matches `"Implementation: greeting.txt"`, `"implementation"`, `"Implementation cycle 2: ..."` uniformly — no special-casing for prefix vs suffix needed. Confirmed against the failure-output sample `('Implementation: greeting.txt', 30.2)` and `scripts/test_lib.py:1462-1466` where `ensign_name` is sourced from Agent `description`.
- If the FO later renames `description` strings to drop the stage word (e.g., `"Fix greeting"` instead of `"Implementation: greeting.txt"`), both checks would fail simultaneously — that's a contract change the test *should* surface, not silently absorb.
- The fixture-based proof is robust to changes in `test_lib._update_dispatch_budgets` because it imports `DispatchRecord` directly and bypasses the JSONL parser entirely.

## Stage Report: ideation

- DONE: Acceptance criteria are entity-level end-state properties about the post-edit test
  Five ACs above (AC-1 through AC-5) each name a verifiable property of the finished file/test, each with `Verified by:` citations (grep commands, comprehension-evaluation outputs, pytest run, CI re-run signal).
- DONE: Proposed approach picks among the three seed directions with explicit reasoning about discrimination power
  Option A chosen; reasoning table compares all three options across legit / bug / drift scenarios; Option A wins on signal specificity for legitimate drift while still catching the original bug via impl-count==1.
- DONE: Test plan picks proof at the right abstraction level
  Layer 1 (fixture-based unit tests in `tests/test_feedback_keepalive_helpers.py`) is the load-bearing proof — three synthetic `DispatchRecord` lists exercise the count expressions directly. Layer 2 (live E2E re-run) is downstream confirmation only. Static prose grep and transcript-fixture replay explicitly rejected with rationale.

### Summary

Chose Option A: replace the stale `len(records) == 2` with two per-stage count assertions (`impl-suffix==1`, `validation-suffix==2`) keyed on `DispatchRecord.ensign_name`. The rewritten assertion still catches the original bug (impl-count would be 2) and surfaces future legitimate drift on the validation axis specifically rather than as a generic total mismatch. Proof is fixture-based unit tests in the existing `test_feedback_keepalive_helpers.py` against three synthetic dispatch-record lists (legitimate, original-bug, future-drift); live E2E is confirmation, not proof. No new helpers introduced; the count expressions inline cleanly.

## Stage Report: implementation

- DONE: Mechanism-check first — fixture tests in `tests/test_feedback_keepalive_helpers.py` covering legitimate / original-bug / future-drift scenarios, committed as a discrete commit before editing the live test.
  Commit `ccbb7eee`; `uv run pytest tests/test_feedback_keepalive_helpers.py -v` shows 3 new cases green (12 total in the file).
- DONE: Replaced stale `t.check("FO emitted exactly two ensign Agent() dispatches...")` in `tests/test_feedback_keepalive.py` with two per-stage suffix-count assertions on `DispatchRecord.ensign_name` (case-insensitive substring); SendMessage-to-kept-alive and per-dispatch-budget assertions unchanged.
  Commit `22ae874b`; `grep -c 'len(records) == 2' tests/test_feedback_keepalive.py` → 0; new count expressions at lines 191 (`"implementation"` == 1) and 195 (`"validation"` == 2).
- DONE: Fixture test and rewritten live-CI assertion both pass under the current 3-dispatch contract; original-bug 4-dispatch fixture still fails the impl-count check.
  `uv run pytest tests/test_feedback_keepalive_helpers.py -v` → 12 passed; `make test-static` → 603 passed, 26 deselected. Live `claude-live-opus` confirmation is deferred to Layer 2 of the test plan (PR re-run).

### Summary

Wrote the fixture-based mechanism check first (3 new test cases against `DispatchRecord(ensign_name, elapsed)` from `scripts/test_lib.py`), then replaced the stale `len(records) == 2` with two suffix-count assertions keyed on `ensign_name.lower()`. Field name confirmed as `ensign_name` per `test_lib.DispatchRecord`; the failure-output sample (`('Implementation: greeting.txt', 30.2)` etc.) is the stage-prefixed Agent description used by that field, so the case-insensitive substring match works as the ideation specified without label-string special-casing. `make test-static` is green; live E2E confirmation will follow on the PR.

## Stage Report: validation

- DONE: AC-1 impl-count assertion exists at value 1
  `tests/test_feedback_keepalive.py:191` carries `sum(1 for r in records if "implementation" in r.ensign_name.lower()) == 1` on a `t.check` line; independent comprehension evaluation against L/B/D fixtures returns `True, False, True` as the AC specifies.
- DONE: AC-2 validation-count assertion exists at value 2
  `tests/test_feedback_keepalive.py:195` carries `sum(1 for r in records if "validation" in r.ensign_name.lower()) == 2` on a `t.check` line; comprehension evaluation against L/B/D fixtures returns `True, True, False` as the AC specifies.
- DONE: AC-3 stale `len(records) == 2` removed
  `grep -c 'len(records) == 2' tests/test_feedback_keepalive.py` → 0.
- DONE: AC-4 fixture pytest green and discrimination-power confirmed
  `uv run pytest tests/test_feedback_keepalive_helpers.py -v` → 12 passed, 0 failed; `_FIXTURE_ORIGINAL_BUG` has two impl-suffixed names so `_impl_count` returns 2 and `test_catches_original_bug_on_impl_count` asserts `!= 1` (PASS) — the new assertion still rejects the original-bug shape.
- DONE: AC-5 stale failing-assertion text removed locally
  `grep -c "exactly two ensign Agent" tests/test_feedback_keepalive.py` → 0; replaced by `"exactly 1 implementation-suffixed"` (line 190) and `"exactly 2 validation-suffixed"` (line 194). Live `claude-live-opus` re-run is Layer-2 downstream confirmation per the test plan; not the proof bearer for this stage.
- DONE: Scope drift check
  `git diff --stat 4e601cb9..057b73d4` touches only `tests/test_feedback_keepalive.py`, `tests/test_feedback_keepalive_helpers.py`, and `docs/plans/test-feedback-keepalive-count-assertion-stale.md` — exactly the allowed scope.
- DONE: `make test-static` clean
  `make test-static` → 603 passed, 26 deselected, 15 subtests passed in 27.51s.
- DONE: Mechanism-check ordering held
  `git log --oneline` shows `ccbb7eee` (fixture tests) precedes `22ae874b` (assertion swap) precedes `057b73d4` (stage report) — failing fixtures landed before the assertion edit they validate.

### Summary

PASSED. All five ACs independently reproduced: AC-1/AC-2 expression text + verdict matches; AC-3/AC-5 stale strings absent (0 hits each); AC-4 fixture pytest is 12 green and the original-bug fixture (two impl-suffixed records, `_impl_count == 2`) correctly fails the new `impl == 1` shape — discrimination power preserved. Scope clean (3 files, all expected), `make test-static` green (603 passed), and the three implementation commits ordered fixtures-before-swap-before-report. Layer-2 live E2E on `claude-live-opus` remains downstream confirmation per the test plan.

### Feedback Cycles

- **Cycle 1 (captain-rejected post-validation, routed to implementation):** Local Layer-2 live-E2E run on `claude+opus` (the same target combination as the CI failure that motivated this task) **failed** the new `validation-count == 2` assertion. The local FO emitted only one validation-suffixed Agent() dispatch — `[('Create a greeting file: implementation', 30.7), ('Create a greeting file: validation', 26.3)]` — no third Agent() for re-validation. SendMessage feedback routing fired correctly, sentinel was touched, FO exited cleanly. The CI's third dispatch `('Validation cycle 2', 0.0)` had elapsed = 0.0s, which is physically impossible for real ensign work (local validation took 26.3s) — strongly suggests a test-harness extraction artifact, not a real Agent() spawn. Either reading kills the cycle-1 assertion: locally it's 1 validation dispatch, CI it was 2 only via a phantom 0.0s record. The actually-load-bearing discriminator is `impl-count == 1` — that's what catches the original bug (where a buggy FO would fresh-dispatch a second impl Agent()). The validation-count assertion adds no signal beyond what `impl-count == 1` plus the existing SendMessage-routing PASS already provide. Captain's direction: drop the `validation-count == 2` assertion; keep only `impl-count == 1`; update fixture tests accordingly. Separate follow-up task to be filed to investigate the harness's phantom dispatch.

## Stage Report: implementation (cycle 2)

- DONE: Mechanism-check first — fixture tests in `tests/test_feedback_keepalive_helpers.py` updated to drop `_val_count` helper and the validation-axis cases; legitimate fixture trimmed to `[("Implementation: greeting.txt", 30.2), ("Validation", 22.6)]`; the original-bug case keeps the 4-record shape and asserts `_impl_count != 1`. Committed as a discrete commit before editing the live test.
  Commit `08b8c15d`; `uv run pytest tests/test_feedback_keepalive_helpers.py -v` → 11 passed (was 12; the future-drift validation case is gone).
- DONE: Removed the validation-count assertion from `tests/test_feedback_keepalive.py`. The impl-count `t.check` and per-dispatch-budget `t.check` remain; SendMessage-routing and static-template greps remain unchanged.
  Commit `60e3a0fe`; `grep -c 'validation-suffixed' tests/test_feedback_keepalive.py` → 0.
- DONE: Updated entity body (AC-1..AC-5 → AC-1..AC-4, proposed-approach prose, test-plan Layer-1 pattern) to reflect the single-stage check. Cited cycle-1 Feedback Cycles entry for the audit trail.
  This commit; AC-2 now covers both "stale `len(records) == 2` absent" and "validation-count assertion absent"; AC-3 fixture list trimmed to L + B (no D).
- DONE: `make test-static` clean.
  602 passed, 26 deselected, 15 subtests passed in ~30s (one fewer pass than cycle 1's 603, expected: the future-drift validation case was removed).

### Summary

Cycle-2 narrowing per captain: dropped `validation-count == 2` from both the live test and the helper fixtures; kept `impl-count == 1` as the sole dispatch-count discriminator. The original-bug fixture still fails the new shape (`_impl_count == 2`, not 1), preserving regression coverage of the load-bearing path: FO must route fixup via SendMessage to the kept-alive implementation ensign, not fresh-dispatch a second Agent(). Live `claude-live-opus` re-run is the validator's job in cycle-2 validation; local mechanism check is green.

## Stage Report: validation (cycle 2)

- DONE: AC-1 impl-count assertion exists at value 1
  `tests/test_feedback_keepalive.py:191` carries `sum(1 for r in records if "implementation" in r.ensign_name.lower()) == 1` on a `t.check` line (label on `:190`). Independent comprehension evaluation: legit fixture L → `impl_count == 1` (True), original-bug fixture B → `impl_count == 2` (False) — matches AC-3's expected `True, False` verdicts.
- DONE: AC-2 stale total-count assertion removed; validation-count assertion not present
  `grep -c 'len(records) == 2' tests/test_feedback_keepalive.py` → 0; `grep -c 'validation-suffixed' tests/test_feedback_keepalive.py` → 0. Survey of all `t.check` lines (94, 98, 189, 193, 201, 205, 209) confirms no count assertion keyed on `"validation"`-suffixed records remains; the remaining `validation` references are `w.expect_dispatch_close` (`:154-158`), prints, and comments — not count assertions.
- DONE: AC-3 rewritten check passes against current FO contract, fails against original bug
  `uv run pytest tests/test_feedback_keepalive_helpers.py -v` → 11 passed in 0.01s (was 12 in cycle 1; drift case correctly removed). `TestDispatchCountAssertions::test_passes_against_current_contract` PASS, `test_catches_original_bug_on_impl_count` PASS — the latter asserts `_impl_count(_FIXTURE_ORIGINAL_BUG) != 1` against a 4-record list with two impl-suffixed names (`Implementation: greeting.txt` + `Implementation cycle 2: greeting.txt`), so the surviving `impl-count == 1` assertion still rejects the original-bug shape.
- DONE: AC-4 live-CI claim — explicitly downstream, not run by validator
  Per dispatch instructions, AC-4's live `claude-live-opus` re-run is the captain's responsibility before pushing. Validator does not run the live E2E (cost + time). Local Layer-1 fixture proof is green and is the load-bearing proof per the entity's test-plan section.
- DONE: Discrimination-power check
  Fixture-L and fixture-B independently evaluated outside pytest (`_impl_count(L) == 1` returns True; `_impl_count(B) == 1` returns False). The two fixtures diverge under the surviving assertion — discrimination preserved. Critically, the simplification did NOT collapse legit and bug into the same verdict.
- DONE: Scope drift check
  `git diff --stat ab598382..HEAD` shows exactly 3 files touched since the cycle-1 validation report: `docs/plans/test-feedback-keepalive-count-assertion-stale.md` (88 lines), `tests/test_feedback_keepalive.py` (4 lines), `tests/test_feedback_keepalive_helpers.py` (34 lines). All within the allowed scope. Four commits cited: `b8603b55` (cycle-1 feedback), `08b8c15d` (fixture trim), `60e3a0fe` (assertion drop), `edeb792c` (entity body + plan).
- DONE: `make test-static` clean
  `make test-static` → 602 passed, 26 deselected, 15 subtests passed in 27.93s. Matches the ~602 expected count cited in the dispatch (one fewer than cycle-1's 603 because the future-drift validation-axis fixture case was dropped).

### Summary

PASSED. Cycle-2 narrowing verified end-to-end: AC-1 impl-count expression present and verdict-correct against L/B fixtures; AC-2's two stale strings (`len(records) == 2`, `validation-suffixed`) both at 0 hits; AC-3 fixture pytest 11 green with the original-bug case still rejected by the surviving `== 1` check; AC-4 deferred to captain's local live run per dispatch. Discrimination power preserved — the simplification did not collapse legit and bug into the same verdict. Scope clean (3 files), `make test-static` green at 602 passed.
