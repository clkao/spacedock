---
id: 3gj9jz071ehf0xnstdxkh5ms
title: "test_reuse_dispatch and other live tests embed static prose greps against shared-core wording deleted by stickiness"
status: validation
source: "claude-live-opus failure on PR #181 (stickiness merge) — 3 of 4 inner checks failed; one is a true positive: `test_reuse_dispatch.py:232-233` greps shared-core for `same.*worktree.*mode` (the literal phrase surface 1 of #181 deleted)."
started: 2026-05-01T08:37:48Z
completed:
verdict:
score: 0.45
worktree: .worktrees/spacedock-ensign-test-reuse-dispatch-static-prose-post-stickiness
issue:
pr:
mod-block: merge:pr-merge
---

## Problem

`tests/test_reuse_dispatch.py:232-233` checks for the literal phrase `same.*worktree.*mode` in `first-officer-shared-core.md`:

```python
t.check("worktree mode match required",
        bool(re.search(r"same.*worktree.*mode", core, re.IGNORECASE)))
```

PR #181 (`21` stage-worktree-stickiness) replaced exactly that phrase — surface 1 of the stickiness change reworded reuse condition #3 from `Next stage has the same \`worktree\` mode as the completed stage` to `Reuse-routing matches the entity's worktree state`. The grep now returns no match, so the live test fails on the now-obsolete assertion.

This was a true positive: the test encoded the old behavior, the contract changed, and the test caught the change after-the-fact via opus-tier live CI. Only opus tier reached it (haiku xfails `test_reuse_dispatch` for unrelated reasons), and `make test-static` never runs it (`@pytest.mark.live_claude` only).

PR #181's AC-1 negative grep correctly verified the old phrase is gone from the assembled FO content via `make test-static`, but it did not audit whether other tests' embedded static prose checks reference the same phrase from a different angle. That's the structural gap.

## Captain direction (2026-05-01)

Captain identified the failure during PR #181 post-merge CI review. Direction: **file a follow-up to update the obsolete grep AND audit other live tests for the same pattern.** PR #181 already merged; this task closes the loop.

## Proposed approach

### Direct fix

1. Update `tests/test_reuse_dispatch.py:232-233` to grep the new anchor instead of the deleted phrase. Replacement check:
   ```python
   t.check("worktree state-routing rule documented",
           bool(re.search(r"Reuse-routing matches the entity's worktree state", core)))
   ```
   The new phrase is the surface-1 anchor; matches the AC-1 grep PR #181 already enforces in `make test-static`.

2. Run `unset CLAUDECODE && make test-e2e TEST=tests/test_reuse_dispatch.py RUNTIME=claude --model opus --effort low` to confirm the fixed check passes on opus. Optionally re-run the full live-claude-opus suite to confirm no regression elsewhere.

### Audit (broader gap)

Sweep all `tests/test_*.py` files marked `@pytest.mark.live_claude` or `@pytest.mark.live_codex` for embedded static prose greps against the 8 surfaces PR #181 changed. Surfaces and anchor phrases:

- Surface 1 (shared-core line 144): `Reuse-routing matches the entity's worktree state` (was: `Next stage has the same \`worktree\` mode as the completed stage`)
- Surface 2 (shared-core line 218): `When \`worktree:\` is set` (was: bare `### Feedback Cycles` bullet)
- Surface 3 (shared-core line 178): `worktree-side when \`worktree:\` is set, main-side otherwise` (was: `keeps it on the main branch`)
- Surface 4 (shared-core line 208): `including \`### Feedback Cycles\` entries` (was: bare bullet)
- Surface 5 (shared-core line 228): `applies in the appropriate view` (carve-out clarification, was: bare bullet)
- Surface 6 (claude-runtime line 155): `read from the worktree copy when \`worktree:\` is set` (parenthetical added)
- Surface 7 (3 commission templates): `Once set on first dispatch` (added to `worktree` row)
- Surface 8 (codex-runtime line 84): `route the dispatch into that existing worktree` (was: `If the stage is not marked for a worktree, stay on the main branch`)

For each match found, decide: update to the new anchor, drop the check (if redundant with `make test-static`), or migrate to `make test-static` so future contract changes are caught at the offline tier rather than burning live-CI minutes.

## Audit (2026-04-30)

Ran `grep -nE "same.*worktree.*mode|keeps it on the main branch|If the stage is not marked for a worktree|Next stage has the same" tests/test_*.py`. Three matches:

| Match | Marker | Assertion polarity | Action |
|---|---|---|---|
| `tests/test_reuse_dispatch.py:233` `re.search(r"same.*worktree.*mode", core, re.IGNORECASE)` | `@pytest.mark.live_claude` | positive (asserts deleted phrase **is present**) | **Direct fix** — repoint to surface-1 anchor `Reuse-routing matches the entity's worktree state` |
| `tests/test_codex_runtime_stickiness.py:25` `"If the stage is not marked for a worktree, stay on the main branch" not in text` | none (offline) | negative (asserts deleted phrase **is absent**) | **Keep** — this is the AC-1b negative guard introduced by PR #181 itself |
| `tests/test_repo_edit_guardrail.py:56` `"Next stage has the same `worktree` mode as the completed stage" not in fo_text` | none (offline; the `live_claude` marker on line 85 is on `test_repo_edit_guardrail`, not on `test_shared_core_stickiness_static_content` at line 37) | negative (asserts deleted phrase **is absent**) | **Keep** — this is PR #181's AC-1 negative guard |

The audit's intent is to catch **positive greps that pin deleted prose**. Negative `not in` assertions on the deleted phrases are part of the change's offline guardrail and should remain. Only `test_reuse_dispatch.py:233` matches the failure pattern.

### 8-surface offline coverage

Mapping each PR-#181 surface to its offline assertion:

| # | Anchor (post-stickiness) | Source | Offline test |
|---|---|---|---|
| 1 | `Reuse-routing matches the entity's worktree state` | `first-officer-shared-core.md:144` | `tests/test_repo_edit_guardrail.py:50` ✓ |
| 2 | `When \`worktree:\` is set` (inside `### Feedback Cycles` FO Write Scope bullet) | `first-officer-shared-core.md:218` | `tests/test_repo_edit_guardrail.py:65` ✓ |
| 3 | `worktree-side when \`worktree:\` is set, main-side otherwise` | `first-officer-shared-core.md:178` | `tests/test_repo_edit_guardrail.py:74` ✓ |
| 4 | `including \`### Feedback Cycles\` entries` | `first-officer-shared-core.md:208` | **GAP** |
| 5 | `applies in the appropriate view` | `first-officer-shared-core.md:228` | **GAP** |
| 6 | `read from the worktree copy when \`worktree:\` is set` | `claude-first-officer-runtime.md:155` | **GAP** |
| 7 | `Once set on first dispatch` | 3 commission templates | `tests/test_development_template.py:31` ✓ |
| 8 | `route the dispatch into that existing worktree` | `codex-first-officer-runtime.md:84` | `tests/test_codex_runtime_stickiness.py:20` ✓ |

Surfaces 4, 5, and 6 are uncovered offline. Recommended additions (literal-substring asserts on the assembled FO content for 4/5; on the claude-runtime file for 6):

- Surface 4: in `tests/test_repo_edit_guardrail.py::test_shared_core_stickiness_static_content`, add `assert "including \`### Feedback Cycles\` entries" in fo_text`.
- Surface 5: same test, add `assert "applies in the appropriate view" in fo_text`.
- Surface 6: a new offline test (or extend `test_codex_runtime_stickiness.py` into a generic `test_runtime_stickiness_anchors.py`) asserting `"read from the worktree copy when \`worktree:\` is set" in claude_runtime_text`.

## Acceptance criteria

**AC-1 — `tests/test_reuse_dispatch.py` greps the post-stickiness anchor for reuse condition #3 and passes on opus tier.**
End-state: line 232-233 reads (or equivalent labeled check) `t.check("worktree state-routing rule documented", bool(re.search(r"Reuse-routing matches the entity's worktree state", core)))`. Verified by: `unset CLAUDECODE && make test-e2e TEST=tests/test_reuse_dispatch.py RUNTIME=claude` with `--model opus --effort low` exits 0 with the `[Static Template Checks]` section showing `PASS: worktree state-routing rule documented`. (Live verification is implementation-stage work; offline `make test-static` regression check is a precondition.)

**AC-2 — No live-marker test in `tests/test_*.py` contains a positive grep for any of PR #181's 8 deleted phrases.**
End-state: `grep -nE "same.*worktree.*mode|keeps it on the main branch|If the stage is not marked for a worktree|Next stage has the same \`worktree\` mode as the completed stage" tests/test_*.py` returns only matches that are (a) negative `not in` / `not bool(re.search(...))` assertions, or (b) inside offline (no `live_*` marker) tests. The audit table above captures the current state: the only positive-grep live-test match is `test_reuse_dispatch.py:233`, removed by AC-1.

**AC-3 — `make test-static` enforces literal-substring anchors for all 8 PR-#181 surfaces, so future deletions of these phrases are caught offline.**
End-state: each surface in the 8-surface table above has a corresponding `assert <anchor> in <source>` in an offline test (no `live_*` marker). Surfaces 1, 2, 3, 7, 8 are already covered as cited in the table. Surfaces 4, 5, 6 must be added per the recommended assertions above. Verified by: running `make test-static` after the additions exits 0, and the three new asserts can be located via `grep -nE "including \`### Feedback Cycles\` entries|applies in the appropriate view|read from the worktree copy when" tests/test_*.py`.

## Test plan

1. **AC-1 direct fix:** 1-line change in `tests/test_reuse_dispatch.py:232-233` repointing the grep to `Reuse-routing matches the entity's worktree state` and renaming the check label. Run `make test-static` to confirm no offline regression. Run `unset CLAUDECODE && make test-e2e TEST=tests/test_reuse_dispatch.py RUNTIME=claude --model opus --effort low` to confirm the fixed live check passes. Cost: ~17 min wall, ~$1.
2. **AC-2 audit:** rerun `grep -nE "same.*worktree.*mode|keeps it on the main branch|If the stage is not marked for a worktree|Next stage has the same \`worktree\` mode as the completed stage" tests/test_*.py` after the fix. Confirm only negative assertions inside offline tests remain (per audit table). Cost: <1 min.
3. **AC-3 gap-fill:** add three offline literal-substring asserts:
   - Extend `tests/test_repo_edit_guardrail.py::test_shared_core_stickiness_static_content` with two new `t.check` lines for surfaces 4 and 5.
   - Add a new offline test (or extend `tests/test_codex_runtime_stickiness.py` into a runtime-stickiness module) asserting surface 6's anchor in `claude-first-officer-runtime.md`.
   Verify with `make test-static`. Cost: ~10 min, no live spend.

Total estimated cost: ~30 min hands-on + one opus live run (~17 min wall, ~$1).

## Out of scope

- Re-running the full claude-live-opus suite for unrelated drifts (test_dispatch_completion_signal, test_feedback_keepalive). Those are tracked separately in #177 / #198 / #160.
- Restructuring live tests to remove all static prose checks (would be a broader cleanup task; this entity is scoped to the post-stickiness gap).

## Cross-references

- PR #181 (stage-worktree-stickiness, merged 2026-05-01) — root cause
- failed CI run: https://github.com/clkao/spacedock/actions/runs/25206509179/job/73908102321
- #177 opus-4-7-ensign-hallucination-scope — the broader opus-drift class (separate from this task)

## Stage Report: ideation

- DONE: Run the audit `grep -nE "same.*worktree.*mode|keeps it on the main branch|If the stage is not marked for a worktree|Next stage has the same" tests/test_*.py` and document each match with a recommended action.
  3 matches; documented in the `## Audit (2026-04-30)` table. Only `tests/test_reuse_dispatch.py:233` is a positive live-test grep needing the direct fix; the other two are offline negative guards introduced by PR #181 itself and are kept.
- DONE: Confirm the AC list as filed at intake is still correct after the audit.
  AC-1 unchanged; AC-2 now cites the audit table directly with concrete file:line evidence; AC-3 expanded with 8-surface coverage table and named gap-fillers for surfaces 4, 5, 6.
- DONE: Quick read of `tests/test_repo_edit_guardrail.py::test_shared_core_stickiness_static_content`, `tests/test_codex_runtime_stickiness.py`, and `tests/test_development_template.py` — confirm each surface is covered, name any gap with a recommended assertion.
  Confirmed coverage for surfaces 1, 2, 3, 7, 8. Surfaces 4 (`including \`### Feedback Cycles\` entries`), 5 (`applies in the appropriate view`), and 6 (`read from the worktree copy when \`worktree:\` is set`) are uncovered offline; recommended literal-substring asserts named in the entity body.

### Summary

The audit confirmed exactly one live-test positive grep (`test_reuse_dispatch.py:233`) against PR #181's deleted phrases — the true positive that triggered this task. The other two grep hits are negative `not in` guards inside offline tests, which are part of PR #181's offline guardrail and must remain. The 8-surface offline coverage check found 3 gaps (surfaces 4, 5, 6) that AC-3 now requires the implementation worker to close with three small literal-substring asserts; no live-CI spend needed for AC-3.

## Stage Report: implementation

- DONE: Apply the 1-line direct fix to `tests/test_reuse_dispatch.py:232-233`. Repoint the grep from `r"same.*worktree.*mode"` to `r"Reuse-routing matches the entity's worktree state"` and rename the check label from `worktree mode match required` to `worktree state-routing rule documented` (per AC-1 end-state).
  Changed at `tests/test_reuse_dispatch.py:232-233`. Verified: `grep -n "Reuse-routing matches" tests/test_reuse_dispatch.py` → `233:            bool(re.search(r"Reuse-routing matches the entity's worktree state", core)))`; `grep -n "same.*worktree.*mode" tests/test_reuse_dispatch.py` returns nothing.
- DONE: Close the 3 offline coverage gaps (surfaces 4, 5, 6) per the entity body's `### 8-surface offline coverage` recommendations.
  Surface 4 (`including `### Feedback Cycles` entries`) and surface 5 (`applies in the appropriate view`) added as two new `t.check` calls in `tests/test_repo_edit_guardrail.py::test_shared_core_stickiness_static_content`. Surface 6 (`read from the worktree copy when `worktree:` is set`) added as a new `test_claude_runtime_stickiness_anchor` function in `tests/test_codex_runtime_stickiness.py`, with the file's ABOUTME generalized to cover both runtime adapters (kept the existing filename to minimize blast radius). Verified: `grep -nE "including \`### Feedback Cycles\` entries|applies in the appropriate view|read from the worktree copy when" tests/test_*.py` returns the three new asserts at `tests/test_codex_runtime_stickiness.py:37`, `tests/test_repo_edit_guardrail.py:79-80`, `tests/test_repo_edit_guardrail.py:85-86`.
- DONE: Run `make test-static` AND the AC-1 live opus verification.
  `make test-static` → exit 0, `577 passed, 26 deselected, 15 subtests passed in 29.28s` (offline regression + the new AC-3 asserts all green).
  Live opus run via `unset CLAUDECODE && uv run pytest tests/test_reuse_dispatch.py --runtime claude --model opus --effort low -v -s` (the make-target syntax in the dispatch checklist did not pass through; pytest was invoked directly with the same flags) → exit 0, `1 passed in 189.86s`. Inner check totals: `14 passed, 0 failed (out of 14 checks)`. The renamed AC-1 check fired green: `PASS: worktree state-routing rule documented` (Static Template Checks block). Phase 2 ran on `claude-opus-4-7` (39 assistant messages, 1.7M cache-read tokens, 183s FO wallclock).

### Summary

AC-1 1-line repoint applied (`tests/test_reuse_dispatch.py:232-233`); AC-3 three offline literal-substring asserts added (`tests/test_repo_edit_guardrail.py:79-80,85-86` for surfaces 4 and 5 inside the assembled-FO test, `tests/test_codex_runtime_stickiness.py:33-39` for surface 6 against `claude-first-officer-runtime.md`). `make test-static` green (577 pass) and live opus `test_reuse_dispatch` green (14/14 inner checks, including the renamed `worktree state-routing rule documented`). AC-2 follows automatically: the only positive live-test grep against the 8 deleted phrases was the line repointed in AC-1. All three checklist items DONE; deliverable on branch `spacedock-ensign/test-reuse-dispatch-static-prose-post-stickiness` at commit `0ea21cf9`.

## Stage Report: validation

- DONE: Run `make test-static` from the worktree; cite exit code and pass count. Confirm the new AC-3 surface 4/5/6 asserts are in the run by grepping for them in the test files. AC-1 live verification was already executed by the implementation ensign (189.86s opus-4-7 PASS, inner checks 14/14); do NOT re-burn live opus minutes. Instead, verify the test_reuse_dispatch.py change holds via `grep -n "Reuse-routing matches" tests/test_reuse_dispatch.py` and `grep -n "same.*worktree.*mode" tests/test_reuse_dispatch.py` (should match the new anchor and return nothing for the old anchor respectively).
  `make test-static` exit 0, `577 passed, 26 deselected, 15 subtests passed in 27.75s`. Anchor grep returned `tests/test_reuse_dispatch.py:233: bool(re.search(r"Reuse-routing matches the entity's worktree state", core)))`; old-anchor grep returned no matches. AC-3 surface asserts located at `tests/test_repo_edit_guardrail.py:80,86` and `tests/test_codex_runtime_stickiness.py:37`. Live opus run not re-executed (impl already paid 189.86s, 14/14 inner checks).
- DONE: Cross-check AC-1, AC-2, AC-3. AC-1: confirm the renamed check label `worktree state-routing rule documented` is in `tests/test_reuse_dispatch.py:232-233`. AC-2: re-run the audit grep `grep -nE "same.*worktree.*mode|keeps it on the main branch|If the stage is not marked for a worktree|Next stage has the same" tests/test_*.py` and confirm only negative `not in` guards inside offline tests remain (the audit table in the entity body is the reference). AC-3: confirm the three new asserts at the cited lines (test_repo_edit_guardrail.py:79-80, :85-86, test_codex_runtime_stickiness.py:33-39 per the implementation report) and that all three actually execute under `make test-static`.
  AC-1: read of `tests/test_reuse_dispatch.py:232` shows label `"worktree state-routing rule documented"` with the new anchor at line 233. AC-2: audit grep returned exactly two matches, both negative `not in` assertions inside offline tests — `tests/test_codex_runtime_stickiness.py:28` (no live marker; whole file offline) and `tests/test_repo_edit_guardrail.py:56` (inside `test_shared_core_stickiness_static_content` at line 37, which carries no marker; the `@pytest.mark.live_claude` at line 97 is on the unrelated #196 xfail test). AC-3: all three asserts present at the cited lines and all executed under the green `make test-static` run (the surface-4/5 asserts are inside `test_shared_core_stickiness_static_content` and surface-6 is in `test_claude_runtime_stickiness_anchor`, both offline).
- DONE: Issue PASSED or REJECTED. If REJECTED, name the failing AC and missing/contradictory evidence.
  PASSED.

### Summary

All three ACs verified statically. AC-1: live opus run already proved green by implementation (189.86s, 14/14); the renamed check `worktree state-routing rule documented` and new anchor are in place at `tests/test_reuse_dispatch.py:232-233`, with the deleted phrase fully removed from the file. AC-2: the audit grep returns only two negative-polarity offline guards — both correct keepers. AC-3: the three offline literal-substring asserts for surfaces 4, 5, 6 are present at the cited lines and execute under `make test-static` (577 pass, exit 0). Recommendation: PASSED.
