---
id: 223
title: "id-style change should be forward-only — accept legacy sequential IDs when README declares sd-b32"
status: validation
source: "GitHub issue #169 (filed by CL, 2026-04-29)"
started: 2026-04-29T21:21:01Z
completed:
verdict:
score: 0.6
worktree: .worktrees/spacedock-ensign-forward-only-id-style-mixed-mode
issue: "#169"
pr:
mod-block:
---

## Problem

`skills/commission/bin/status` validator hard-rejects mixed id formats: when the workflow README declares `id-style: sd-b32`, every entity must hold a valid sd-b32 stored id. Legacy entities with numeric ids fail validation with `invalid sd-b32 stored id`. Captains who want sd-b32 going forward have no clean path short of bulk-migrating every existing entity (200+ for `docs/plans` — 50 active + 175 archived).

The display layer already tolerates the mixed case: `compute_sd_b32_display_ids` (status:631) builds the prefix dict only from valid sd-b32 ids, and `apply_effective_ids` (status:656) falls back to the stored id when an entity is not in that dict. Numeric stored ids therefore round-trip through display unchanged. Only the validator (status:701-714) enforces single-style and blocks the mixed state.

The driving use case is **this very workflow** (`docs/plans/`): once the validator change ships, the captain will flip `docs/plans/README.md` from `id-style: sequential` to `id-style: sd-b32` (next captain action). The implementation worker must remain aware that 200+ existing sequential entities must validate clean immediately after that flip — the test fixture covers this in miniature.

## Approach

Make the sd-b32 branch of `validate_workflow` (status:701-714) **forward-only**:

- When `id_style == 'sd-b32'`, accept a stored id as valid if **either** `is_valid_sd_b32_id(stored_id)` is true **or** `stored_id.isdigit()` is true (the same predicate the `sequential` branch uses at status:680).
- Track uniqueness across both id spaces with a single `seen` dict keyed on `stored_id` (not on shape). A numeric stored id and a sd-b32 stored id cannot collide because the alphabets are disjoint, so a single dict is sufficient and matches the existing structure.
- Continue to flag `missing required id` for entities with no stored id at all.
- Drop the `invalid sd-b32 stored id` error for the numeric case only; keep it for genuinely malformed values (e.g., `not-valid-sd-b32-id` from the existing test).

`compute_sd_b32_display_ids` and `apply_effective_ids` need **no changes** — re-read confirms (status:631-661): the prefix builder filters to `valid_ids` only, and the apply step falls back to `stored_id` for anything not in the display dict. Numeric stored ids therefore appear in the table column unmodified, alongside short sd-b32 prefixes for the new entities.

`compute_next_id` / `compute_next_sd_b32_id` (status:940-970) also need no changes: when `id_style: sd-b32` is declared, new entities continue to receive a freshly generated sd-b32 stored id. The `existing` set at status:941-945 already includes numeric stored ids from legacy entities, which is harmless (a numeric value will never collide with an sd-b32 candidate).

### Display ordering with mixed ids

Sort order in `print_status_table` (status:766) and `print_next_table` is governed by stage order and score, not by id. Mixed display ids therefore appear interleaved by stage/score with no special grouping — this matches today's behavior for any single style. **No change to sort keys.** The ID column is already left-aligned with a 6-char width, which fits both the 4-digit numeric ids in `docs/plans` and the 2-3 char sd-b32 prefixes.

### Prefix resolution for `status <id-prefix>`

The prefix-lookup branch at status:1593 (`stored_id.startswith(value)`) operates on stored ids and is style-agnostic. A numeric prefix matches numeric stored ids; an sd-b32 prefix matches sd-b32 stored ids. No change required.

## Acceptance criteria

- AC1: Validator accepts a workflow with `id-style: sd-b32` that contains a mix of one entity with a valid sd-b32 stored id and one entity with a numeric stored id; `status --validate` exits 0 and `status --boot` exits 0.
  Verified by: new pytest case in `tests/test_status_script.py` exits 0 (assert returncode == 0 and empty stderr for the validation lines).
- AC2: Validator still rejects a malformed stored id under `id-style: sd-b32` (e.g., `not-valid-sd-b32-id`) with the existing `invalid sd-b32 stored id` error.
  Verified by: existing `test_sd_b32_validate_rejects_duplicate_full_id_invalid_id_and_missing_id` continues to pass unchanged.
- AC3: Validator still rejects duplicate stored ids (sd-b32-sd-b32 collision and numeric-numeric collision) and missing stored ids under `id-style: sd-b32`.
  Verified by: existing duplicate/missing assertions in `test_sd_b32_validate_rejects_duplicate_full_id_invalid_id_and_missing_id` continue to pass; new pytest case adds a numeric-numeric duplicate under sd-b32 mode and asserts `duplicate sd-b32 stored id` (or new shared message) appears in stderr.
- AC4: New entity creation in a mixed-mode workflow produces an sd-b32 stored id, not a numeric id.
  Verified by: new pytest case runs `status --next-id` against the mixed fixture and asserts `is_valid_sd_b32_id(output)` (using the same `SD_B32_ALPHABET`/regex helpers already imported in the test module).
- AC5: Display table for the mixed fixture shows the numeric stored id verbatim and an sd-b32 short prefix for the sd-b32 entity, with both rows present.
  Verified by: new pytest case runs `status` (default table) against the mixed fixture and asserts both display strings appear in stdout.
- AC6: Existing static test suite (`pytest tests/test_status_script.py`) passes in full with no regressions.
  Verified by: full run of `pytest tests/test_status_script.py` exits 0.

## Test plan

**Regression-test fixture** — added to `tests/test_status_script.py` inside `TestPluggableIdStyle` (the class that already owns `test_sd_b32_validate_rejects_*` and uses `make_pipeline` + `readme_with_id_style('sd-b32')`):

- `test_sd_b32_accepts_legacy_numeric_ids_in_mixed_workflow` — pipeline with `id-style: sd-b32`, one entity holding a sd-b32 stored id (`sd_b32_id('ab')`), one entity holding a numeric stored id (`'1'`), one numeric (`'2'`). Asserts `--validate` exits 0, `--boot` exits 0, default table includes both display ids.
- `test_sd_b32_mixed_workflow_next_id_yields_sd_b32` — same fixture, asserts `status --next-id` output passes `SD_B32_ID_RE`.
- `test_sd_b32_mixed_workflow_rejects_numeric_duplicate` — pipeline with `id-style: sd-b32` and two numeric entities both holding id `'1'`. Asserts non-zero exit and `duplicate` substring in stderr.

**Test entrypoint:** `pytest tests/test_status_script.py -k sd_b32` (and the unrestricted `pytest tests/test_status_script.py` for AC6).

**Cost / complexity:** trivial. Pure-Python subprocess tests using existing `make_pipeline` / `run_status` / `entity` / `sd_b32_id` helpers. No new fixtures, no E2E, no real workflow execution. Estimated <1s additional runtime.

**E2E not needed:** the validator is a pure function over filesystem state and the existing test harness already exercises it via subprocess against synthesized pipelines — that is the right abstraction level for this behavioral claim.

## Out of scope

- **Symmetry for other id-style transitions** (sequential-mode accepting sd-b32 ids, slug-mode accepting either): narrow this change to the sd-b32 branch only. Sequential and slug have different semantic contracts (sequential assumes a monotonic counter; slug derives the id from the filename), so loosening them deserves separate ideation. Flagged as open question — captain may extend later if a concrete need appears.
- **Migration helper** (a `status --migrate-to-sd-b32` subcommand that bulk-rewrites existing entities): explicitly out of scope. The whole point of forward-only is that no migration is required. A helper can be added as a follow-up entity if a captain later wants to opportunistically convert legacy ids.
- **Display grouping** (showing numeric ids and sd-b32 ids in separate sections of the table): not implemented. Sort order remains stage/score-based; mixed ids interleave naturally. Revisit only if visual confusion is reported.

## Open questions

- Should the validator emit an info-level note when it observes a mixed id population (e.g., "12 numeric, 3 sd-b32") to make the transitional state visible to captains? Default answer: **no** — the table already shows both ids, and adding a note risks false positives in workflows that intentionally remain mixed long-term. Implementation worker may flag if user testing suggests otherwise.

## Stage Report: ideation

- DONE: Approach is concrete: pin the exact validator change (which branch in `validate_workflow` lines 701-714 to relax, what the per-entity acceptance check looks like, how uniqueness is tracked across the two id spaces) and confirm `compute_sd_b32_display_ids` + `apply_effective_ids` already handle the mixed-display path without changes (re-read the script to verify, do not just trust the issue body).
  Re-read status:631-661 confirms display path is style-tolerant; sd-b32 branch at status:701-714 to be relaxed by accepting `is_valid_sd_b32_id(stored_id) or stored_id.isdigit()`; uniqueness tracked in single `seen` dict keyed on stored_id (alphabets are disjoint, so no cross-collision).
- DONE: Test plan names a regression test fixture (a workflow README declaring `id-style: sd-b32` plus a mix of entities with legacy numeric ids and new sd-b32 ids) and the test entrypoint that exercises it. Include the next-id generation path (a new entity created in this mixed workflow gets sd-b32, not a numeric id).
  Three new pytest cases named in `tests/test_status_script.py::TestPluggableIdStyle`; entrypoint `pytest tests/test_status_script.py -k sd_b32`; next-id path covered by `test_sd_b32_mixed_workflow_next_id_yields_sd_b32`.
- DONE: AC items are end-state properties with concrete `Verified by:` clauses (e.g., `pytest <new test>` exits 0; static suite still green; new fixture validates clean under `status --boot`).
  Six AC items written as observable end-state properties, each with a `Verified by:` line citing a specific pytest case or existing assertion; `status --boot` covered in AC1; full-suite green covered in AC6.

### Summary

Pinned the change to a single branch (status:701-714): accept numeric stored ids alongside sd-b32 stored ids when `id-style: sd-b32` is declared, using `is_valid_sd_b32_id(x) or x.isdigit()` and a single shared `seen` dict. Confirmed via re-read of status:631-661 that the display layer needs no edits, and confirmed the next-id and prefix-lookup paths are already mixed-tolerant. Test plan adds three pytest cases under the existing `TestPluggableIdStyle` class, reusing the established `make_pipeline`/`entity`/`sd_b32_id` helpers; symmetry for other id-styles, migration helpers, and grouped display were ruled out of scope.

### Feedback Cycles

**Cycle 1 — validation REJECTED (2026-04-29 ~21:36 UTC), driving-use-case spot-check failed.**

Cycle-1 implementation passed all 6 ACs against synthetic fixtures, but the validator's bonus driving-use-case spot-check (flip `docs/plans/README.md` to `id-style: sd-b32`, run `--boot` against the real workflow) produced `duplicate sd-b32 stored id` errors against pre-existing archive-only duplicate ids (`131` × 2 — claude-team-context-limit-config-lie + codex-first-officer-reused-worker-wait-bookkeeping; `033` × 2 — graceful-degradation-without-teams + initial-prompt-frontmatter; plus other archive duplicates the validator's audit summary cited).

Root cause: the new sd-b32 branch lacks the active-vs-archive scope tolerance the sequential branch has at status:684-690 (which skips duplicate groups when no active entity participates). Synthetic fixtures all created active-vs-active duplicates, never archive-vs-archive — so the gap wasn't exercised.

Captain decision: drop archive-only duplicate-checking entirely (option 2 from FO triage). Both branches skip duplicate groups whose members are all in archive scope. Active-vs-archive collisions remain rejected (cross-references would silently break otherwise). Archive-only collisions are cosmetic data quirks in commit history, not runtime correctness issues.

This conceptual reframe explains the existing sequential-branch tolerance: archive entries are historical record, not workflow state. The sd-b32 branch should match.

Routing cycle 2 to the implementation ensign on standby (context 5.3%, reuse_ok). New AC + new test case for archive-only duplicate tolerance; small code mirror of the existing sequential-branch logic into the sd-b32 branch.

**Cycle 1 resolved (2026-04-29 ~21:54 UTC).** Cycle 2 implementation landed three commits (f88f722c validator restructure, 7b1a81ff archive-only dup tolerance test, 8fcdf8c7 cycle-2 stage report). Cycle-2 validation reproduced all 7 ACs cleanly AND the driving-use-case spot-check: flipping `docs/plans/README.md` to `id-style: sd-b32` against 50 active + 175 archived entities now yields `VALID` exit 0. The four pre-existing archive-only dup pairs are tolerated as designed. Captain approved the gate. Advancing to merge.
