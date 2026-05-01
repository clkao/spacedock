---
id: 3gj9jz071ehf0xnstdxkh5ms
title: "test_reuse_dispatch and other live tests embed static prose greps against shared-core wording deleted by stickiness"
status: backlog
source: "claude-live-opus failure on PR #181 (stickiness merge) — 3 of 4 inner checks failed; one is a true positive: `test_reuse_dispatch.py:232-233` greps shared-core for `same.*worktree.*mode` (the literal phrase surface 1 of #181 deleted)."
started:
completed:
verdict:
score: 0.45
worktree:
issue:
pr:
mod-block:
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

## Acceptance criteria

**AC-1 — `test_reuse_dispatch.py` greps the post-stickiness anchor for reuse condition #3 and passes on opus tier.**
Verified by: `unset CLAUDECODE && make test-e2e TEST=tests/test_reuse_dispatch.py RUNTIME=claude` with `--model opus --effort low` exits 0 with the `[Static Template Checks]` section showing `PASS: worktree state-routing rule documented` (or equivalent renamed check).

**AC-2 — No live test embeds a static prose grep against any of PR #181's 8 deleted/changed phrases.**
Verified by: a one-shot audit script (or manual `grep -nE` across `tests/test_*.py` filtered by `live_*` markers) demonstrating zero matches against the deleted phrases listed above. Document the audit results in the entity body's stage report so a future change-watcher can re-run the same sweep.

**AC-3 — `make test-static` enforces structural anchors for the 8 surfaces, so future deletions of these phrases are caught offline.**
Verified by: confirming the existing `tests/test_repo_edit_guardrail.py::test_shared_core_stickiness_static_content` (PR #181, AC-1+AC-7) plus `tests/test_codex_runtime_stickiness.py` (PR #181, AC-1b) plus `tests/test_development_template.py` (PR #181, AC-3) cover the 8 surfaces. If any surface is uncovered, add a literal-substring assertion to the appropriate offline test file.

## Test plan

1. Direct fix: 1-line change in `tests/test_reuse_dispatch.py:232-233`. Run `make test-static` to confirm no regression. Run targeted opus live test to confirm fix.
2. Audit: `grep -nE "same.*worktree.*mode|keeps it on the main branch|If the stage is not marked for a worktree" tests/test_*.py` (the highest-impact deleted phrases). Inspect each match.
3. AC-3 verification: read existing static tests and confirm coverage against the 8-surface list. Add gap-fillers if needed.

Cost: low. ~30 min for direct fix + audit + verification. AC-1 needs one opus live run (~17 min wall, ~$1).

## Out of scope

- Re-running the full claude-live-opus suite for unrelated drifts (test_dispatch_completion_signal, test_feedback_keepalive). Those are tracked separately in #177 / #198 / #160.
- Restructuring live tests to remove all static prose checks (would be a broader cleanup task; this entity is scoped to the post-stickiness gap).

## Cross-references

- PR #181 (stage-worktree-stickiness, merged 2026-05-01) — root cause
- failed CI run: https://github.com/clkao/spacedock/actions/runs/25206509179/job/73908102321
- #177 opus-4-7-ensign-hallucination-scope — the broader opus-drift class (separate from this task)
