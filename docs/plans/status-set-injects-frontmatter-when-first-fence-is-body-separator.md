---
id: m4ywfn5tbtsaxebf6btxrhww
title: "status --set silently injects frontmatter into files whose first --- is a body separator"
status: validation
source: GitHub issue #186 (clkao/spacedock)
started: 2026-05-04T16:22:12Z
completed:
verdict:
score: 0.6
worktree: .worktrees/spacedock-ensign-status-set-injects-frontmatter-when-first-fence-is-body-separator
issue: "#186"
pr:
mod-block: merge:pr-merge
---

## Problem

`skills/commission/bin/status` treats the first `---` it encounters anywhere in a markdown file as the opening of YAML frontmatter, rather than requiring `---` on line 1. Files whose first `---` is a body horizontal rule (a common markdown idiom) are silently misidentified as entities; `status --set` then mutates them, splicing keys into the middle of a user-authored document with exit 0 and no warning.

The entity-discovery glob compounds this: every non-`README.md` `*.md` in the workflow directory is a candidate, so research artifacts, drafts, or other docs colocated with entities are eligible for corruption. Issue #186 contains the canonical reproduction (no need to re-derive it).

## Failure surfaces

Two writeable surfaces and two read-side surfaces participate:

| # | Location | Role | Severity |
|---|----------|------|----------|
| F1 | `parse_frontmatter` (line 96) | read | sets `in_fm=True` on first `---` regardless of position; corrupts every downstream caller's view of "what fields does this file have" |
| F2 | `update_frontmatter` (line 1438) | write | same scan, then inserts/rewrites fields between the two `---` lines; this is the splice that mutates user prose |
| F3 | `discover_entity_files` (line 365) | read | returns every `*.md` (minus README/dotfiles/reserved dirs) as an entity candidate without checking the file actually has frontmatter |
| F4 | `parse_stages_block` (line 188) | read | same shape as F1 but only ever called against `README.md` whose frontmatter is fixed by the workflow contract |

`--set notes` reaches `update_frontmatter` because `discover_entity_files` (F3) lists `notes.md`, `resolve_reference_candidates` finds `slug == 'notes'` (an empty-field entity object built by `scan_entities`), and `resolve_mutation_entity` returns it. So slug pre-resolution (issue suggestion #4) is already happening — it just trusts whatever F3 returns.

## Approach

Fix F1 + F2 + F3. Skip F4 — `parse_stages_block` is invoked only against `README.md`, whose frontmatter is required by the workflow contract and never starts with body prose; broadening it adds churn for no reachable bug.

Why not a smaller subset:

- **F1 alone:** would correct the read-side contract, but `discover_entity_files` would still surface no-frontmatter `*.md` files as zero-field entities, leaking into `--validate` output, `--next` ranking, and any future writer.
- **F3 alone:** narrows discovery, but anyone calling `parse_frontmatter` directly on a file path (e.g., the FO reading a referenced research doc) keeps the silent-misparse footgun.
- **F1 + F3:** correct under current callers, but `update_frontmatter` is the one function that loses a user document on misuse. A defensive write check (F2) costs ~3 lines and forecloses any future regression in F1 or any caller that bypasses discovery.

So the cut is F1 + F2 + F3. Suggestion #4 ("`--set` should resolve through entity discovery before opening the file") is already realised by `resolve_mutation_entity`; once F3 narrows discovery, #4 is satisfied transitively.

### F1: strict opening fence in `parse_frontmatter`

Require the first non-empty, non-BOM line to be exactly `---`. Otherwise return `{}` (file has no frontmatter — the parser's existing "no fields" return value).

- BOM: strip a leading `﻿` from the first line read before comparison.
- "Non-empty" means: after `rstrip('\n')`, the line equals the empty string. A line containing only spaces or tabs is **not** blank — it is content, and so an opening fence cannot follow it. (This matches Jekyll/Hugo strictness; documented as T-PARSE-4.)
- Leading blank lines (per the rule above) are allowed to precede the fence; nothing else is.
- Behaviour after the opening fence is unchanged.

### F2: defensive write check in `update_frontmatter`

Before scanning for `fm_start`/`fm_end`, verify the file's first non-empty, non-BOM line is `---`. If not, raise `ValueError(f'No frontmatter found in {filepath}')` (matches the existing error string so callers' error reporting is unchanged).

This duplicates F1's check by design — the write path must not depend on the read path's correctness.

### F3: entity discovery requires opening fence

In `discover_entity_files`, after the existing flat/folder filtering, drop any file whose first non-empty, non-BOM line is not `---`. Implementation: a small predicate `_has_opening_fence(path)` reused by F1 and F2 keeps the three surfaces in lockstep. Apply the predicate only to candidate flat files and `index.md` files; do not change reserved-dir / dotfile / README handling.

Behavioural consequence: a stray `notes.md` without frontmatter no longer appears in `--next`, `--validate`, the default table, or as a `--set` target. Slug resolution for `--set notes` then returns `unknown reference: notes` and exits 1.

Out of scope:
- Warning when a `*.md` is silently skipped. The workaround in the issue ("move non-entity markdown files out of the workflow directory") is a workflow concern; surfacing skipped files would require a separate `--lint` mode and isn't load-bearing for fixing the corruption bug.
- Retrofitting `parse_stages_block` (F4). README contract guarantees frontmatter starts at line 1; the same predicate could be applied for symmetry but adds no observable behaviour change and bloats the diff.
- Backwards compatibility shims. No durable user state depends on the buggy behaviour; an entity file that genuinely starts with frontmatter is still parsed identically.

## Acceptance criteria

Each criterion describes an end-state of the implementation; the test that verifies it is named in the next section.

- **AC-1 (parser strict fence — `parse_frontmatter`):** A file whose first non-empty, non-BOM line is not `---` returns `{}` from `parse_frontmatter`. Files with leading blank lines and/or a BOM before a valid `---` opening fence parse identically to the same file without that prefix. A leading whitespace-only line counts as content, so a file beginning with `"   \n---\n…"` returns `{}`. *Verified by:* T-PARSE-1, T-PARSE-2, T-PARSE-3, T-PARSE-4.
- **AC-2 (write guard — `update_frontmatter`):** Calling `update_frontmatter` on a file whose first non-empty, non-BOM line is not `---` raises `ValueError` with message `'No frontmatter found in {filepath}'` and leaves the file byte-identical to the pre-call state. *Verified by:* T-WRITE-1, T-WRITE-2.
- **AC-3 (active-scope discovery skip):** In a workflow directory containing `valid.md` (proper frontmatter) and `notes.md` (first `---` is a body horizontal rule), `status --validate` exits 0 with no errors *and* the default-table run (`status` with no flags) lists `valid` only. Folder-form entities `slug/index.md` whose `index.md` lacks an opening fence are likewise absent from both. *Verified by:* T-DISCOVER-1, T-DISCOVER-2.
- **AC-4 (archived-scope discovery skip):** A `_archive/orphan.md` whose first `---` is a body horizontal rule does not appear in `status --archived` output; an `_archive/valid.md` with proper frontmatter still does. *Verified by:* T-DISCOVER-3.
- **AC-5 (end-to-end — issue #186 reproduction):** Running `status --set notes id=001` against a workflow containing the reproduction's `README.md` and `notes.md` exits non-zero with `Error: unknown reference: notes` on stderr, and `notes.md` is byte-identical to its pre-call state (verified via SHA256 before and after). *Verified by:* T-E2E-186.
- **AC-6 (regression — valid entities still mutate):** `status --set <valid-slug> status=ideation` against a properly-fenced entity file rewrites the `status` field in frontmatter and exits 0, with stdout matching the existing `field: old -> new` shape. *Verified by:* T-E2E-VALID.

## Test plan

Unit tests live in `tests/test_status_*.py` and execute via the existing `build_status_script` / `run_status` harness (subprocess against a real `status` template-substituted into a tmpdir). Both unit and end-to-end tests are pure-Python with stdlib only; estimated cost is one new test file plus additions to `test_status_set_missing_field.py`. No network, no live LLM, no E2E beyond the subprocess harness — `status` is a self-contained Python script and the issue is fully reproducible at the script boundary.

| ID | Type | Harness | Expected result |
|----|------|---------|-----------------|
| T-PARSE-1 | unit / parser fixture | direct import or `python3 -c` against the templated script; feed a file with prose first then `---` | `parse_frontmatter` returns `{}` |
| T-PARSE-2 | unit / parser fixture | leading blank lines then `---\nid: 001\n---` | returns `{'id': '001'}` |
| T-PARSE-3 | unit / parser fixture | leading `﻿` BOM then `---\nid: 001\n---` | returns `{'id': '001'}` |
| T-PARSE-4 | unit / parser fixture | leading whitespace-only line `"   \n"` then `---\nid: 001\n---` | returns `{}` (whitespace-only line is content, not blank) |
| T-WRITE-1 | unit / parser fixture | call `update_frontmatter` against a file whose first `---` is a body separator | raises `ValueError`; file bytes unchanged (assert via `read_bytes()` before/after) |
| T-WRITE-2 | unit / parser fixture | call `update_frontmatter` against a fully-prose file with no `---` at all | raises `ValueError`; file bytes unchanged |
| T-DISCOVER-1 | unit / harness | `make_pipeline` with `valid.md` (proper FM) + `notes.md` (body-rule shape from issue); run `status --validate` (assert exit 0, no errors) and `status` default table (assert `valid` row present, `notes` row absent) | as stated |
| T-DISCOVER-2 | unit / harness | folder-form `slug/index.md` whose first `---` is a body rule, alongside a valid sibling | `slug` absent from default table; valid sibling present |
| T-DISCOVER-3 | unit / harness | `_archive/orphan.md` (body-rule shape) plus `_archive/valid.md` (proper FM); run `status --archived` | `valid` listed; `orphan` absent; exit 0 |
| T-E2E-186 | end-to-end / harness | exact reproduction from issue #186 (README + notes.md verbatim), invoke `status --set notes id=001` via `run_status` | exit code != 0; stderr contains `unknown reference: notes`; `notes.md` SHA256 unchanged |
| T-E2E-VALID | end-to-end / harness | proper entity file `task.md`; `status --set task status=ideation` | exit 0; frontmatter `status` rewritten; body unchanged; stdout matches `status: ... -> ideation` |

Pristine-output rule: T-E2E-186 captures stderr and asserts the expected error string; no stray warnings or tracebacks tolerated. T-E2E-VALID asserts stdout matches the existing `field: old -> new` shape.

The unit tests for AC-1/AC-2 can either import the script as a module (the existing pattern in `test_status_parse_mod_metadata.py` shows how) or invoke a tiny Python one-liner via the templated script — pick whichever matches the surrounding test style during implementation.

## Scope decision

Adopt **F1 + F2 + F3** (three of the four issue suggestions; suggestion #4 is satisfied transitively). Validation can cross-check this by:

1. Reading this section and the AC table.
2. Confirming `parse_stages_block` (F4 / suggestion-area-4 read path) is unchanged.
3. Confirming all three surfaces share a single `_has_opening_fence` helper so the strictness contract is defined once.

## Independent reviewer pass

A fresh-eyes review of the spec above flagged three deltas, all of which are already folded into the AC list and Test plan:

- **R1 — archive scope:** `discover_entity_files` is called against `_archive/` too (`scan_entities(archive_dir)` at line 565). The fix must skip body-fence files in archived scope, and the AC list must say so. → folded as AC-4 + T-DISCOVER-3.
- **R2 — sharper AC-3 verification:** The original AC-3 said "returns exactly `[…]` from `discover_entity_files`," which is an internal-API claim. Restated in terms of public commands (`status --validate` exit 0; default table contents) so the entity-level property is observable to validation. → folded into AC-3 / T-DISCOVER-1.
- **R3 — whitespace-only first line:** The strict-fence rule is ambiguous about `"   \n"` as a leading line. Decision: whitespace-only is content, not blank — matches Jekyll/Hugo strictness and prevents a future "looks like blank, fence still applies" footgun. → folded into F1 spec, AC-1, T-PARSE-4.

Reviewer also confirmed:
- **`parse_stages_block` exclusion is sound** because it is invoked only against `README.md`, whose frontmatter is contractually on line 1.
- **Mod-file behaviour is unaffected in a strict-improvement way:** `parse_mod_metadata` calls `parse_frontmatter`; a malformed mod with body-only `---` previously could read body lines as `key: value` pairs (e.g., a body line `standing: true` would be misparsed). After F1, such mods read as `{}` — `standing` defaults to `False`, never accidentally `True`. No new test required because the prior behaviour was incidental, not contractual.
- **No backwards-compatibility shim is needed.** Files with proper line-1 frontmatter parse identically to today; only files that today corrupt parse to a defined "empty frontmatter" state.

## Stage Report: ideation

- DONE: Acceptance criteria enumerate every failure surface the fix must close (e.g., parse_frontmatter strict opening fence, update_frontmatter defensive write check, entity discovery skipping non-entity files, --set slug pre-resolution) with explicit verification for each.
  AC-1 covers F1 (parser strict fence), AC-2 covers F2 (write guard), AC-3 covers F3 active-scope discovery skip, AC-4 covers F3 archived-scope discovery skip; slug pre-resolution (suggestion #4) is satisfied transitively by AC-3+AC-4 and re-verified end-to-end by AC-5. AC-6 is the regression check that valid mutations still work. Each AC names the test IDs that prove it.
- DONE: Test plan names parser-level fixture coverage AND at least one end-to-end `--set` against the issue #186 reproduction (file whose first `---` is a body horizontal rule), specifying the harness and expected exit state.
  T-PARSE-1..4 and T-WRITE-1/2 are parser fixtures; T-E2E-186 is the issue #186 reproduction via `run_status`, asserting non-zero exit, `unknown reference: notes` on stderr, and byte-identical `notes.md`.
- DONE: Scope decision recorded: minimal subset vs. all four suggested fixes from the issue, with reasoning the validation stage can cross-check.
  "Approach", "Scope decision", and "Independent reviewer pass" sections record F1+F2+F3 with rationale for skipping F4 and how suggestion #4 is satisfied transitively; cross-check checklist and reviewer-fold-in deltas (R1 archive scope, R2 AC-3 sharpening, R3 whitespace-only line) included.

### Summary

Selected F1 (strict opening fence in `parse_frontmatter`) + F2 (defensive write check in `update_frontmatter`) + F3 (discovery skip in `discover_entity_files`); the four issue suggestions reduce to three implementation surfaces because slug pre-resolution already routes through discovery. Test plan combines four parser fixtures, two write-guard fixtures, three discovery-harness tests (active + folder-form + archived), and two end-to-end `--set` runs — all via the existing `tests/test_status_*.py` subprocess harness; no live LLM or worktree needed because the bug is fully reproducible at the `status` script boundary. Independent reviewer pass folded in three deltas (archive scope, AC-3 sharpening, whitespace-only first line) before gate.

## Stage Report: implementation

- DONE: F1, F2, and F3 share a single `_has_opening_fence(path)` predicate (defined once, called from `parse_frontmatter`, `update_frontmatter`, and `discover_entity_files`); the strictness contract is not duplicated three ways.
  Predicate added at top of `skills/commission/bin/status` (above `parse_frontmatter`); called as a guard from all three surfaces (commit c9c5f0d5).
- DONE: All 11 tests from the test plan (T-PARSE-1..4, T-WRITE-1/2, T-DISCOVER-1/2/3, T-E2E-186, T-E2E-VALID) added under `tests/test_status_*.py` and passing locally.
  `tests/test_status_strict_opening_fence.py`: 11/11 passed. Note: T-E2E-186 asserts the actual error string `entity not found: notes` produced by `resolve_mutation_entity`'s `unknown` branch — the spec's `unknown reference: notes` was inaccurate about which error path fires; the AC's intent (non-zero exit + slug-not-resolved diagnostic + byte-identical file) is fully verified, including SHA256 before/after.
- DONE: `make test-static` exits 0 — no pre-existing tests regressed by F1/F2/F3.
  `587 passed, 26 deselected, 15 subtests passed in 27.71s`.

### Summary

Implemented F1+F2+F3 as one logical change pivoting on the shared `_has_opening_fence` predicate; total diff is +39 lines in `status` plus a 230-line test file. The only deviation from the spec was the assertion text in T-E2E-186 — the actual error path in `resolve_mutation_entity` produces `entity not found: <ref>` rather than `unknown reference: <ref>`; both prove the AC property (file untouched, non-zero exit, clear stderr), and the SHA256 byte-identity check is the load-bearing assertion. Full static suite green; commit c9c5f0d5 on branch `spacedock-ensign/status-set-injects-frontmatter-when-first-fence-is-body-separator`.

## Stage Report: validation

- DONE: Every AC (AC-1..AC-6) has its named tests rerun in the validator's session; results captured per-suite with `N/N passed` counts and AC-by-AC evidence.
  Targeted file `tests/test_status_strict_opening_fence.py`: **11/11 passed** in 0.21s. Per-AC breakdown:
  - **AC-1 (parser strict fence):** T-PARSE-1 `test_prose_before_first_fence_returns_empty` PASSED, T-PARSE-2 `test_leading_blank_lines_then_fence_parses` PASSED, T-PARSE-3 `test_leading_bom_then_fence_parses` PASSED, T-PARSE-4 `test_whitespace_only_first_line_is_content` PASSED.
  - **AC-2 (write guard):** T-WRITE-1 `test_body_separator_only_raises_and_preserves_bytes` PASSED, T-WRITE-2 `test_pure_prose_no_fence_raises_and_preserves_bytes` PASSED.
  - **AC-3 (active-scope discovery skip):** T-DISCOVER-1 `test_active_scope_skips_body_rule_file` PASSED, T-DISCOVER-2 `test_folder_form_index_without_fence_skipped` PASSED.
  - **AC-4 (archived-scope discovery skip):** T-DISCOVER-3 `test_archived_scope_skips_body_rule_file` PASSED.
  - **AC-5 (end-to-end issue #186):** T-E2E-186 `test_set_against_body_rule_file_fails_safely` PASSED.
  - **AC-6 (regression):** T-E2E-VALID `test_set_against_valid_entity_still_works` PASSED.
  Full repo offline suite via `make test-static`: **587 passed, 26 deselected, 15 subtests passed in 27.77s** — no pre-existing tests regressed.
- DONE: The implementation's spec/reality drift on T-E2E-186 (asserted `entity not found: notes` instead of the spec's `unknown reference: notes`) is independently confirmed against the live `status` binary; if the AC's intent (non-zero exit, byte-identical file, slug-not-resolved diagnostic) is fully met the validator records that and treats the assertion text as a spec correction, not a defect.
  Independent reproduction in `/tmp/drift-check`: invoked `python3 skills/commission/bin/status --set notes id=001` against a workflow with body-rule `notes.md`; live binary emitted `Error: entity not found: notes` on stderr and exit 1; `sha256sum` of `notes.md` was `b71cb89e26a99f6efd31f14e3d58fbcedd6b831c8ce9943420622880942462e3` both before and after the invocation (byte-identical). AC-5's three required properties — non-zero exit, slug-not-resolved diagnostic on stderr, byte-identical file via SHA256 — are all met. Recording as a spec correction: the actual error path in `resolve_mutation_entity` emits `entity not found: <ref>` rather than `unknown reference: <ref>`; the test assertion matches reality and the AC intent is fully verified.
- DONE: PASSED or REJECTED recommendation with one-line rationale; if REJECTED, the rationale names the specific AC that failed.
  **PASSED** — all six ACs verified by named tests (11/11 targeted tests green, full static suite 587/587 green, AC-5 spec-vs-reality drift independently confirmed against the live binary as a benign spec correction with full intent preserved).

### Summary

Validation reproduced every AC's named test, ran the full offline suite without regression, and independently verified the T-E2E-186 spec/reality drift against the live `status` binary — confirming `Error: entity not found: notes`, exit 1, and byte-identical SHA256 (`b71cb89e…`). Recommendation: **PASSED**. The deviation from the spec's `unknown reference:` wording is a spec correction (the actual `resolve_mutation_entity` "unknown" branch emits `entity not found: <ref>`), and all three load-bearing AC-5 properties are satisfied.
