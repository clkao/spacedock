---
id: yv8kbe048ad4y1mb0dtqe5dj
title: "pr-merge audit link should render the short SD-B32 prefix, not the full 24-char stored ID"
status: implementation
source: "Captain observation 2026-04-30 during PR #179 — `[jaafh7xzz0va63rj6bgdgh3p](...)` is unreadable; operators and transcripts refer to the entity as `ja`. Captain direction: 'edit pr mod to use short handles for id=sd-b32'"
started: 2026-04-30T23:35:00Z
completed:
verdict:
score: 0.6
worktree: .worktrees/spacedock-ensign-pr-merge-audit-link-short-sd-b32
issue:
pr:
mod-block:
---

## Problem

The pr-merge mod's PR-body audit link template is `[{entity-id}](/{owner}/{repo}/blob/{short-sha}/{path-to-entity-file})`. For sd-b32 workflows the rendered link is `[jaafh7xzz0va63rj6bgdgh3p](...)` — the 24-character stored ID, which:

1. Is unreadable in PR list / mobile / commit-message contexts
2. Doesn't match how operators or transcripts ever refer to the entity (we say `ja`, not the full hash)
3. Is the only sd-b32-specific render that doesn't already use the shortest-unique-prefix that `status` displays in its ID column

For sequential workflows (`[001]`) and slug-only workflows (`[restore-initial-prompt]`), the existing render is already short and readable. Only sd-b32 is broken.

## Captain direction (2026-04-30)

> edit pr mod to use short handles for id=sd-b32

This task captures that direction. Treat it as the ideation gate spec — captain has approved direction; implementation is the next step.

## Proposed approach

1. **Add a `status --short-id REF` flag** to `skills/commission/bin/status`. Returns the shortest-unique-prefix from active+archived for sd-b32 workflows; returns the literal stored ID for sequential and slug workflows. This keeps prefix-computation centralized in `status` (which already does it for the ID column) instead of forcing each consumer to re-derive it.
2. **Update both pr-merge mod copies** (`mods/pr-merge.md` and `docs/plans/_mods/pr-merge.md`) so the audit-link rendering step calls `status --short-id` and uses the result in the `[{entity-id}]` slot. Keep both files byte-identical (the `[Mod Install Freshness]` test from #197 enforces this).
3. **Add a `tests/test_status_short_id.py` host** with three parametric cases (sd-b32, sequential, slug) and one ambiguity case (sd-b32 with archived collision forcing prefix lengthening).

## Acceptance criteria

**AC-1 — `status --short-id REF` returns the shortest-unique-prefix for sd-b32, the stored ID for sequential and slug.**
Verified by: `tests/test_status_short_id.py` parametric cases. For sd-b32, asserts the returned prefix equals what `status` displays in its ID column. For sequential (`001`) and slug (`my-task`), asserts the returned value equals the literal stored ID.

**AC-2 — `status --short-id` lengthens the prefix when active+archived contain a colliding sibling.**
Verified by: `tests/test_status_short_id.py` ambiguity case. Two entities share a 2-char prefix; `--short-id` for each returns 3+ chars (whatever disambiguates).

**AC-3 — Both pr-merge mod copies are byte-identical after the update.**
Verified by: existing `[Mod Install Freshness]` check in `tests/test_commission.py` continues to pass after the edit.

**AC-4 — Both pr-merge mod copies' audit-link template references `status --short-id` (or equivalent) for the rendered ID slot.**
Verified by: grep both mod files; the prose in the audit-link instruction step mentions `--short-id` or the equivalent computation step that produces the short prefix.

**AC-5 — `make test-static` is green after the changes.**
Verified by: full static suite run.

## Test plan

1. Add `tests/test_status_short_id.py`:
   - sd-b32 fixture with one entity → `--short-id` returns 2-char prefix
   - sd-b32 fixture with two entities sharing a 2-char prefix → `--short-id` returns 3+ char prefix for each
   - sd-b32 fixture with active + archived collision → `--short-id` returns disambiguating prefix
   - sequential fixture → `--short-id` returns the numeric ID literal
   - slug fixture → `--short-id` returns the slug literal
2. The `[Mod Install Freshness]` byte-compare in `tests/test_commission.py` is the regression guard for AC-3.
3. `make test-static` after both changes.

## Out of scope

- Audit-link rendering for non-PR contexts (e.g., debrief output, status display) — those already use short prefixes in user-facing surfaces.
- Backfilling existing PRs (#155, #36, #179, etc.) with the new short rendering — those are historical and can stay as-is.
- Any change to `status --resolve` semantics — `--resolve` continues to return full stored IDs.

## Cross-references

- Filed alongside `06z0dycs40qr0a9b35waxaxf` (refit-docs-plans-readme) which the captain ordered in the same turn.
- The existing `[Mod Install Freshness]` byte-compare from #197 (PR #176) is the regression guard for AC-3.

## Stage Report: implementation

- DONE: Implement `status --short-id REF` in `skills/commission/bin/status`. For sd-b32 workflows return the shortest-unique-prefix across active+archived. For sequential and slug return the literal stored ID/slug. Reuse the existing prefix computation that powers the ID column.
  Added `parse_short_id_arg` and `print_short_id_or_exit` (reuses `compute_sd_b32_display_ids` and `resolve_reference_candidates` with `include_archived=True`); wired into `main()` with incompat checks parallel to `--resolve`.
- DONE: Update BOTH pr-merge mod copies identically (`mods/pr-merge.md` and `docs/plans/_mods/pr-merge.md`) so the audit-link rendering step calls `status --short-id` and inserts the result into the `[{entity-id}]` slot. After the edit, confirm `diff mods/pr-merge.md docs/plans/_mods/pr-merge.md` is empty.
  Added a short-id computation step in `## Hook: merge` and updated the extraction-rule row to `[{short-id}](...)`; `diff` reports identical and `[Mod Install Freshness]` test is green.
- DONE: Add `tests/test_status_short_id.py` covering AC-1 (3 id-styles) and AC-2 (sd-b32 active+archived collision forces prefix lengthening); run it and `make test-static` from the worktree and cite both pass counts in the report.
  4/4 new short-id tests pass. `make test-static` 561 passed, 26 deselected. Updated `tests/test_pr_merge_template.py::test_template_describes_audit_link_format` to pin `[{short-id}]` and added a sibling `test_audit_link_uses_short_id_command`.

### Summary

Added `status --short-id REF` that returns the shortest-unique-prefix used by the ID column for sd-b32, and the literal stored ID/slug for sequential/slug. Updated both pr-merge mod copies to call `status --short-id` when rendering the audit-link `[{entity-id}]` slot, fixing the unreadable 24-char render seen on PR #179. AC-3 byte-parity is preserved (mod copies identical, `[Mod Install Freshness]` green); AC-5 full static suite is green at 561/561.

## Stage Report: validation

- DONE: Run `make test-static` and `unset CLAUDECODE && uv run pytest tests/test_status_short_id.py tests/test_pr_merge_template.py -v` from the worktree; cite exit codes and pass/fail counts.
  `make test-static` exit 0, 561 passed / 26 deselected / 15 subtests passed. Targeted pytest exit 0, 32 passed (4 short-id + 28 pr-merge-template).
- DONE: For each AC-1..AC-5 in the entity body, name the cited test from its `Verified by:` clause and confirm it appears green in the run.
  - AC-1 (sd-b32 prefix, sequential/slug literals): `tests/test_status_short_id.py::TestShortId::test_short_id_sd_b32_returns_shortest_unique_prefix`, `::test_short_id_sequential_returns_literal_id`, `::test_short_id_slug_returns_literal_slug` — all PASSED.
  - AC-2 (active+archived collision lengthens prefix): `tests/test_status_short_id.py::TestShortId::test_short_id_sd_b32_lengthens_prefix_when_active_archived_collide` — PASSED.
  - AC-3 (mod byte-parity): direct `diff mods/pr-merge.md docs/plans/_mods/pr-merge.md` returns no output (exit 0). `[Mod Install Freshness]` continues green inside `make test-static`.
  - AC-4 (audit-link template references `status --short-id`): grep on both copies shows the `## Hook: merge` instruction prose ("compute the short entity-id slot for the audit link by running `status --short-id {entity ref}`") and the extraction-rule row (`[{short-id}](...)`) at lines 40 and 68 of each copy. `tests/test_pr_merge_template.py::TestAuditMetadata::test_audit_link_uses_short_id_command` and `::test_template_describes_audit_link_format` — both PASSED.
  - AC-5 (`make test-static` green): full static run exit 0 at 561 passed.
- DONE: Issue PASSED or REJECTED.
  PASSED.

### Summary

All five acceptance criteria verified with cited evidence. `make test-static` is green at 561/561; the targeted suite covering the new short-id helper and the pr-merge template is green at 32/32. Mod byte-parity confirmed by direct `diff` (exit 0) in addition to the `[Mod Install Freshness]` regression guard. Recommendation: PASSED.

## Stage Report: implementation (cycle 2)

Captain expanded scope at the push-approval gate after cycle 1 landed: the pre-approval draft must show the FULL templated PR body so the captain reviews the prose that will land on GitHub before push, not just Title/Branch/Changes/Files metadata. AC-6 covers the pre-approval body rendering; AC-7 (implied) is that the post-approval step becomes a literal `gh pr create` invocation against the already-constructed body — no separate construction pass.

- DONE: Edit BOTH pr-merge mod copies (`mods/pr-merge.md` and `docs/plans/_mods/pr-merge.md`) so the `## Hook: merge` pre-approval draft includes the fully constructed PR body (motivation lead + `## What changed` + `## Evidence` + audit link + `Closes` line if applicable), NOT just the Title/Branch/Changes/Files metadata. The post-approval step becomes a literal `gh pr create` invocation against the already-constructed body — no separate post-approval construction step. Confirm `diff mods/pr-merge.md docs/plans/_mods/pr-merge.md` returns empty.
  Reordered the merge hook: audit-link inputs (short SHA, owner/repo, short entity id) and the full templated PR body are now constructed BEFORE the captain draft. The draft adds a `**Body:**` line with the full body in a fenced block. On approval, the post-approval block runs push/rebase/push-branch and then `gh pr create ... --body "{constructed body}"` against the already-constructed body — explicit "do not rebuild it" wording preserved. Pre-approval prose retains the existing AC-5 strings (`git rev-parse --short HEAD`, `status --short-id`, fallback-to-`main` reported to captain). `diff mods/pr-merge.md docs/plans/_mods/pr-merge.md` returns no output (exit 0).
- DONE: Add a `tests/test_pr_merge_template.py` test for AC-6 asserting the `## Hook: merge` pre-approval prose names the constructed PR body (template-rendered) and does NOT contain a separate post-approval body-construction step.
  Added `TestPreApprovalDraftIncludesFullBody` with four cases: pre-approval references the body template and the `{constructed body}` placeholder; pre-approval prose names every required body section (`motivation lead`, `## What changed`, `## Evidence`, `audit link`, `Closes`); post-approval block contains no body-construction phrasing (`Build the PR body using the template`, `construct the PR body`, etc.); post-approval `gh pr create` invocation references `{constructed body}` as already-built. Hook section is bounded by `## Hook: merge` start and the `### PR body template` subsection start.
- DONE: Run `make test-static` and `unset CLAUDECODE && uv run pytest tests/test_status_short_id.py tests/test_pr_merge_template.py -v` from the worktree; cite exit codes and pass counts.
  `make test-static` exit 0, 565 passed / 26 deselected / 15 subtests passed (up from 561 in cycle 1: 4 new AC-6 tests). Targeted pytest exit 0, 36 passed (4 short-id + 32 pr-merge-template, of which 4 are the new `TestPreApprovalDraftIncludesFullBody` cases).

### Summary

Cycle 2 layered the pre-approval body-rendering scope-expansion onto the cycle-1 commit. Both mod copies now construct the full PR body (motivation lead, `## What changed`, `## Evidence`, audit link, `Closes` line) before the captain draft, present that body inside the draft so the captain reviews the actual prose, and then on approval pipe the already-constructed body verbatim into `gh pr create`. AC-6 is covered by four new tests in `TestPreApprovalDraftIncludesFullBody`; full static suite is green at 565/565; mod byte-parity preserved.
