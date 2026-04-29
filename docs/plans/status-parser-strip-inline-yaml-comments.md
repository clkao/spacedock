---
id: 222
title: "status --boot crashes with ValueError when stage YAML has inline comment on value line"
status: implementation
source: "GitHub issue #163 (filed by Kent Chen / iamcxa, 2026-04-29)"
started: 2026-04-29T14:56:54Z
completed:
verdict:
score: 0.65
worktree: .worktrees/spacedock-ensign-status-parser-strip-inline-yaml-comments
issue: "#163"
pr:
mod-block:
---

The hand-rolled YAML parser in `skills/commission/bin/status` (around line 266 in `parse_stages_block`) crashes with `ValueError: invalid literal for int() with base 10: '2  # teammate-debate mode'` when a stage frontmatter has an inline `#` comment on a numeric value line.

CRITICAL when triggered — blocks all `status` operations on the affected workflow until the comment is moved or stripped. Rare in practice (most captains write README comments above value lines, not inline) but undetectable until first FO boot attempt.

External reporter (Kent Chen / `iamcxa`) provided concrete reproduction and a suggested fix: strip inline `#` from value strings before type-cast in `parse_stages_block`. Same fix applies to all type-cast fields: `worktree`, `gate`, `terminal`, `initial`, `concurrency`.

## Why this matters now

Standard YAML allows inline comments. The hand-rolled parser violates a documented YAML expectation. The class of bug is small (one helper function in `parse_stages_block`) but the failure mode is silent until first boot — a captain commissions a workflow with an inline-commented concurrency value, then the FO can never start.

## Approach

Add a small private helper at module scope in `skills/commission/bin/status`:

```python
def _strip_inline_comment(value):
    """Strip a YAML-style inline ` # comment` from a scalar string value."""
    return value.split('#', 1)[0].strip()
```

Apply it inside `parse_stages_block` before every type-cast or `.lower()` comparison on a value derived from the parsed YAML map. The helper takes a string and always returns a string, so the existing `int(...)` / `.lower() == 'true'` chain is unchanged.

### Verified field list

Confirmed by reading `parse_stages_block` (currently `skills/commission/bin/status:271-283`). Two defaults plus five per-state type-cast fields need the strip:

| Site | Line | Cast |
|---|---|---|
| `defaults['worktree']` | 271 | `.lower() == 'true'` |
| `defaults['concurrency']` | 272 | `int(...)` |
| `state['worktree']` | 278 | `.lower() == 'true'` |
| `state['concurrency']` | 279 | `int(...)` |
| `state['gate']` | 280 | `.lower() == 'true'` |
| `state['terminal']` | 281 | `.lower() == 'true'` |
| `state['initial']` | 282 | `.lower() == 'true'` |

The optional pass-through fields (`feedback-to`, `agent`, `fresh`, `model`, lines 284-286) are stored as-is. Reporter's issue did not flag these and the cast chain wouldn't crash on an inline comment, but applying the strip there too is the consistent behavior and avoids a future-confusion landmine when e.g. `model: opus  # team default` shows up. Decision: include `feedback-to`, `agent`, `fresh`, `model` in the strip pass for consistency. (`name`, line 259, is parsed differently via `partition('- name:')` and is out of scope — names with `#` are not a real concern.)

The companion function `parse_stages_with_defaults` (line 292) re-parses but only returns the raw `defaults` dict for the model field; its consumers are downstream of `parse_stages_block`, so applying the strip in `parse_stages_block` is sufficient to fix the boot crash. The raw `defaults` dict returned by `parse_stages_with_defaults` will still contain the unstripped value — this is acceptable because its only documented consumer reads `model`, which will get the strip via the consistency pass above. No call-site changes required.

### Out of scope

- This is **not** an upstream-library migration to PyYAML. The issue explicitly acknowledges "this isn't an upstream-library fix"; the hand-rolled parser stays. A full PyYAML migration is a separate larger task (#TBD if filed) with its own dependency-footprint and bootstrap implications.
- Block-level comments (full-line `# foo`) on stage value lines: already handled correctly today (those lines have no `:` so the `if ':' in dstripped:` guard at line 240/261 skips them). No change needed.
- Quoted-string handling (`worktree: "false  # not a comment"`): not currently supported by the parser at all (quotes are kept literally), and not requested by the issue. Out of scope.

## Test plan

**Host file:** Extend `tests/test_status_script.py` (existing pytest unittest TestCase host for the status script). No new test file — the existing host already covers parsing and dispatch, which is the same surface.

**Test entrypoint:** New test method `test_parse_stages_strips_inline_comments` (added to the existing `unittest.TestCase` class that exercises the script's parser in this file).

**Fixture shape:** An on-the-fly README fixture written into a `tempfile.TemporaryDirectory` (matches the existing pattern in this test module). The README's frontmatter `stages:` block contains a `defaults:` mapping with `concurrency: 2  # team-debate mode` and `worktree: false  # default off`, plus a `states:` list with one state setting `gate: true  # captain-approved`, `terminal: false  # not a sink`, `initial: true  # boot here`, and `concurrency: 4  # high-fanout`. The test invokes the parser (either by importing `parse_stages_block` from the built script, or by running `status --boot` on the fixture pipeline and asserting it does not crash and reports the expected stage configuration).

Two assertions:
1. The call returns successfully (no `ValueError`).
2. The parsed stage dict has the correct types and values: `concurrency == 4` (int), `worktree is False` (bool), `gate is True`, `terminal is False`, `initial is True`.

**Cost / complexity:** Low. Pure Python unit test, no subprocess fanout beyond what `tests/test_status_script.py` already does. Runs in `make test-static`.

**E2E needed?** No. The bug is in a pure parser function; a fixture + parser-call test is the same abstraction level as the claim. No live runtime dispatch is needed to prove inline-comment tolerance.

## Acceptance criteria

- AC1: `parse_stages_block` in `skills/commission/bin/status` does not raise `ValueError` when any of the type-cast frontmatter fields (`concurrency`, `worktree`, `gate`, `terminal`, `initial`, plus their `defaults:` counterparts where applicable) carry an inline ` # comment`.
  - Verified by: new pytest test `tests/test_status_script.py::<TestCase>::test_parse_stages_strips_inline_comments` exits 0.
- AC2: Parsed stage values for inline-commented fields have the correct post-cast types and values (e.g., `concurrency: 4  # ...` parses to integer `4`, `worktree: false  # ...` parses to boolean `False`).
  - Verified by: same test asserts equality on each of the five per-state fields and both defaults fields.
- AC3: Existing status-script behavior is unchanged for frontmatter without inline comments.
  - Verified by: full `make test-static` run is green (all pre-existing tests in `tests/test_status_script.py` and the rest of the static suite continue to pass).
- AC4: The fix is contained to `skills/commission/bin/status` (one new helper function plus call-site application inside `parse_stages_block`); no changes to other parser code paths or to the entity-frontmatter parser (which already strips comments via a different code path or is out of scope per the issue).
  - Verified by: implementation diff touches only `skills/commission/bin/status` and `tests/test_status_script.py`.

## Stage Report: ideation

- DONE: Approach is concrete: identify the helper-function shape and enumerate every type-cast field in `parse_stages_block` (in `skills/commission/bin/status`) that needs the inline-comment strip applied.
  Helper `_strip_inline_comment(value)` named; verified field list against `parse_stages_block` lines 271-283 — five per-state casts (`worktree`, `concurrency`, `gate`, `terminal`, `initial`) plus two defaults (`worktree`, `concurrency`); also documented decision to apply consistently to optional pass-through fields (`feedback-to`, `agent`, `fresh`, `model`).
- DONE: Test plan names a regression test fixture (a stage README with inline `#` comments on numeric and boolean fields) and the test entrypoint that exercises the parser.
  Host: extend existing `tests/test_status_script.py`; new method `test_parse_stages_strips_inline_comments`; fixture is an in-test README written to a `tempfile.TemporaryDirectory` with inline `#` on all type-cast fields. No new test file needed — existing host already covers parser surface.
- DONE: AC items are end-state properties with concrete `Verified by:` clauses.
  Four AC items added (AC1 no-crash, AC2 typed values, AC3 no regression, AC4 scope-contained), each with a `Verified by:` pointing to the new pytest test entrypoint or `make test-static`.

### Summary

Verified the field list against the actual `parse_stages_block` code: five per-state type-cast fields plus two defaults, all matching the issue reporter's list (with one consistency-pass addition for the optional pass-through fields). Pinned the test host to the existing `tests/test_status_script.py` rather than creating a new file (existing module already builds the script and exercises its parser, so it's the right shape). Marked PyYAML migration and quoted-string handling as explicitly out of scope per the issue's own framing.

## Stage Report: implementation

- DONE: `_strip_inline_comment` helper added at module scope in `skills/commission/bin/status` and applied at the verified call sites in `parse_stages_block` (the seven required type-cast sites + the four consistency-pass pass-through fields named in the entity body's Verified field list table). AC-1 / AC-2 / AC-4 satisfied.
  Helper added at `skills/commission/bin/status:188-190`. Applied to: `defaults['worktree']` (line 276), `defaults['concurrency']` (line 277), `state['worktree']` (line 283), `state['concurrency']` (line 284), `state['gate']` (line 285), `state['terminal']` (line 286), `state['initial']` (line 287), and the four pass-through fields `feedback-to`/`agent`/`fresh`/`model` (line 291). Diff scope: only `skills/commission/bin/status`. Commit `0516539b`.
- DONE: New pytest test method `test_parse_stages_strips_inline_comments` added to the existing `tests/test_status_script.py` TestCase host. Fixture is an in-test README written to `tempfile.TemporaryDirectory` per the entity body's Test plan; assertions cover both no-crash AND correct typed values for every field with an inline comment.
  Test added to `TestBootOption` (the most thematically aligned host — bug surfaces under `--boot`). Loads the built script via `SourceFileLoader` (script has no `.py` extension, so `spec_from_file_location` returns None — needed the explicit-loader form), toggles `sys.dont_write_bytecode` around the import to avoid leaking `__pycache__` into the per-class temp dir, and asserts `concurrency == 4`, `worktree is False`, `gate is True`, `terminal is False`, `initial is True` on the parsed `backlog` stage. Verified the assertion catches the original bug by neutering the helper in a one-off probe and confirming `ValueError: invalid literal for int() with base 10: '2  # team-debate mode'` reproduces. Commit `1984409d`.
- DONE: Local verification: the new test passes AND `make test-static` is green (AC-3). Captain prefers local-only verification — do not defer to CI.
  `unset CLAUDECODE && uv run pytest tests/test_status_script.py::TestBootOption::test_parse_stages_strips_inline_comments -v` → 1 passed. `make test-static` → 535 passed, 26 deselected, 15 subtests passed in 26.29s.

### Summary

Two commits on `spacedock-ensign/status-parser-strip-inline-yaml-comments`: `0516539b` adds the `_strip_inline_comment` helper at module scope and applies it to all seven type-cast sites plus the four pass-through fields per the verified table; `1984409d` adds `test_parse_stages_strips_inline_comments` to `TestBootOption` in `tests/test_status_script.py`. Test loads the script via explicit `SourceFileLoader` (script has no `.py` suffix so the default `spec_from_file_location` path returns None) and disables bytecode writes to keep the host class's temp dir clean. New test passes; full `make test-static` run green at 535/535.
