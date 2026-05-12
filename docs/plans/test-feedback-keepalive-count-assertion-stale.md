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
mod-block: merge:pr-merge
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

## Chosen approach: Option A (per-stage breakdown)

Replace the single `len(records) == 2` assertion in `tests/test_feedback_keepalive.py:189-192` with two count assertions keyed on the dispatch's `ensign_name` (the Agent `description` field; `test_lib.DispatchRecord.ensign_name` at `scripts/test_lib.py:1155`):

- exactly one record whose `ensign_name` lowercased contains `"implementation"`
- exactly two records whose `ensign_name` lowercased contains `"validation"`

The kept-alive SendMessage assertion already lives at lines 164-168 of the test and stays unchanged. The per-dispatch budget assertion at lines 193-196 also stays unchanged.

### Why Option A over B and C

The three options were evaluated against the original bug's failure signature. Under the original bug (FO fresh-dispatches a new implementation `Agent()` instead of `SendMessage`-routing to the kept-alive #1), records would be `[impl, validation, impl-cycle-2, validation-cycle-2]` — total 4, impl-suffixed 2, validation-suffixed 2.

| Option | Current legit (3:1:2) | Original bug (4:2:2) | Future drift: 3rd legit validation cycle (4:1:3) |
|---|---|---|---|
| A: impl==1 AND val==2 | PASS | impl-check FAIL (=2) | val-check FAIL (=3) — surfaces drift as a real failure to revisit |
| B: total==3 AND impl==1 | PASS | total-check FAIL (=4) | total-check FAIL (=4) — same noise on drift, less specific signal |
| C: impl-after-validation == 0 | PASS | FAIL | PASS — silent on drift |

Option A wins on signal specificity: a future legitimate third validation cycle fails the *validation* count, not a generic total — the next reader sees immediately which stage drifted. Option A also keeps the impl-count check independent so the original-bug signature remains crisply attributable to a fresh impl dispatch.

Option C is rejected because it would silently accept any number of legitimate-looking validation cycles, weakening the regression net.

## Acceptance criteria

End-state properties of `tests/test_feedback_keepalive.py` after the edit:

1. **AC-1: impl-count assertion exists at value 1.** The test file contains a `t.check(...)` whose boolean is `sum(1 for r in records if "implementation" in r.ensign_name.lower()) == 1` (or equivalent comprehension). *Verified by:* `grep -n 'implementation' tests/test_feedback_keepalive.py` shows the count expression on a `t.check` line; `python3 -c` evaluation of the comprehension against the three fixture lists in AC-4 returns `True, False, True` (legit, bug, drift) respectively.

2. **AC-2: validation-count assertion exists at value 2.** The test file contains a `t.check(...)` whose boolean is `sum(1 for r in records if "validation" in r.ensign_name.lower()) == 2`. *Verified by:* `grep -n 'validation' tests/test_feedback_keepalive.py` shows the count expression on a `t.check` line; comprehension evaluation against the three fixtures returns `True, True, False`.

3. **AC-3: stale total-count assertion removed.** The exact string `len(records) == 2` no longer appears in the file. *Verified by:* `grep -c 'len(records) == 2' tests/test_feedback_keepalive.py` returns 0.

4. **AC-4: rewritten check passes against current FO contract, fails against original bug.** Three synthetic `DispatchRecord` fixtures are exercised in a new unit test in `tests/test_feedback_keepalive_helpers.py`:
   - **Fixture L (legitimate today):** `[("Implementation: greeting.txt", 30.2), ("Validation", 22.6), ("Validation cycle 2", 0.0)]` — both new checks PASS.
   - **Fixture B (original bug):** `[("Implementation: greeting.txt", 30.2), ("Validation", 22.6), ("Implementation cycle 2: greeting.txt", 12.0), ("Validation cycle 2", 0.0)]` — impl-count check FAILS (counts 2).
   - **Fixture D (future drift, third legit validation cycle):** `[("Implementation: greeting.txt", 30.2), ("Validation", 22.6), ("Validation cycle 2", 0.0), ("Validation cycle 3", 0.0)]` — validation-count check FAILS (counts 3).
   *Verified by:* `pytest tests/test_feedback_keepalive_helpers.py -v` shows three new test cases green.

5. **AC-5: live `test_feedback_keepalive` is no longer blocked by the count assertion.** The Phase 3 block produces no `FAIL: FO emitted exactly two ensign Agent() dispatches…` line on a successful live run; the new check lines report PASS for the three-dispatch contract output. *Verified by:* PR re-run of `claude-live-opus` CI shows the feedback-keepalive job green; the prior failing assertion text no longer appears in the test output.

## Test plan

Two layers, sized to the claim. The claim is **test-assertion logic**, not FO runtime behavior — so unit-level fixture proof carries most of the weight; live E2E is a downstream confirmation, not the proof bearer.

### Layer 1 — unit (fixture-based, the load-bearing proof)

Add three test cases to `tests/test_feedback_keepalive_helpers.py` (the existing helper-unit-test file). Each constructs a synthetic `list[DispatchRecord]` and exercises the rewritten count expressions directly. Pattern:

```python
from test_lib import DispatchRecord

def _impl_count(records): return sum(1 for r in records if "implementation" in r.ensign_name.lower())
def _val_count(records):  return sum(1 for r in records if "validation"     in r.ensign_name.lower())

def test_count_assertions_pass_against_current_contract():
    records = [DispatchRecord("Implementation: greeting.txt", 30.2),
               DispatchRecord("Validation", 22.6),
               DispatchRecord("Validation cycle 2", 0.0)]
    assert _impl_count(records) == 1
    assert _val_count(records) == 2

def test_count_assertions_fail_against_original_bug():
    records = [DispatchRecord("Implementation: greeting.txt", 30.2),
               DispatchRecord("Validation", 22.6),
               DispatchRecord("Implementation cycle 2: greeting.txt", 12.0),
               DispatchRecord("Validation cycle 2", 0.0)]
    assert _impl_count(records) == 2   # fails the ==1 check → catches the bug
    assert _val_count(records) == 2

def test_count_assertions_fail_against_future_validation_drift():
    records = [DispatchRecord("Implementation: greeting.txt", 30.2),
               DispatchRecord("Validation", 22.6),
               DispatchRecord("Validation cycle 2", 0.0),
               DispatchRecord("Validation cycle 3", 0.0)]
    assert _impl_count(records) == 1
    assert _val_count(records) == 3   # fails the ==2 check → surfaces drift
```

Implementation note: the helpers `_impl_count` / `_val_count` can live in the unit test file alongside the cases, or be the literal comprehension inlined into both the live test and the unit test. The simpler shape is inline comprehensions — no new helper to maintain.

Cost: seconds. Real Python, real `DispatchRecord` import, no subprocess, no Claude runtime.

### Layer 2 — live E2E (downstream confirmation, not proof)

Re-run `tests/test_feedback_keepalive.py::test_feedback_keepalive` on the PR with the assertion edit:

- Local invocation: `uv run pytest tests/test_feedback_keepalive.py -v -m live_claude` against the current FO + ensign stack. Expected: Phase 3 shows two PASS lines (impl-count==1, validation-count==2) instead of the prior FAIL line. Skip is acceptable if the Claude runtime probe fails locally.
- Live CI confirmation: `claude-live-opus` job on the PR — feedback-keepalive case green.

Cost: ~3-5 minutes wallclock per run (matches the prior PR #208 cycle-3 timing). Run once on the PR; no need to re-run if Layer 1 stays green.

E2E is intentionally *not* the proof for the assertion logic — it depends on real FO output, which the unit fixtures already abstract away. If Layer 1 is green and Layer 2 is red, the divergence is FO contract drift (a separate task), not a defect in this fix.

### Why not other proof shapes

- **Static prose grep of the assertion strings.** Tempting (the failing AC-1/AC-2 acceptance criteria already do this for structure), but a grep can't prove the count expression returns the right verdict on a buggy-FO input. Fixture exercise does.
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
