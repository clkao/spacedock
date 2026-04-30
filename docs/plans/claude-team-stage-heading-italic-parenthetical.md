---
id: jaafh7xzz0va63rj6bgdgh3p
title: "claude-team build stage-heading regex doesn't tolerate italic-wrapped parentheticals (residual #138 case)"
status: validation
source: "GitHub issue #178 (filed by FO from captain report, 2026-04-30)"
started: 2026-04-30T22:49:21Z
completed:
verdict:
score: 0.55
worktree: .worktrees/spacedock-ensign-claude-team-stage-heading-italic-parenthetical
issue: "#178"
pr:
mod-block:
---

The `extract_stage_subsection` regex in `skills/commission/bin/claude-team` was hardened in #138 / PR #145 to tolerate trailing parentheticals like `### \`triaged\` (terminal)`. It does NOT tolerate italic-wrapped parentheticals like `### \`brainstorm\` *(captain-interactive — no ensign)*`.

Real-world repro from a captain who hit it: four stage headings in `docs/feature-exploration/README.md` with italic-wrapped annotations. Workaround was manual italic-strip (commit `6fd804c` in another repo). Before the strip, every dispatch into one of those stages had `claude-team build` silently extracting nothing, and the FO fell through to a break-glass manual template that produced a worse prompt.

Current regex:
```python
heading_re = re.compile(
    rf'^###\s+`?{re.escape(stage_name)}`?(?:\s+\([^)]*\))?\s*$'
)
```

## Captain reframe (2026-04-30)

The original entity body proposed a narrow regex extension (`(?:\s+[*_]?\([^)]*\)[*_]?)?`). Captain rejected this:

> "the solution it proposes is REALLY brittle. It solves this problem; it doesn't solve the problem that the system stopped because it got confused by heading formatting"

The real failure is two-layered:

1. **The regex is too rigid.** Every time captains use a slightly-different Markdown decoration in stage headings (backticks, italics, bold, square-bracket annotations, trailing em-dashes, etc.), the regex breaks and we patch with a narrower regex. Whack-a-mole. The fix is a more permissive name-extraction that survives any reasonable Markdown decoration captains naturally try.
2. **The failure mode is silent.** When the regex doesn't match, `extract_stage_subsection` returns `None` and `cmd_build` returns an error — but the FO has historically fallen through to a break-glass manual template. Captains see "the agent did weird stuff" instead of "your heading didn't parse, here's the line." A robust system either parses captain-formatted headings OR fails loud with a diagnostic that points at the offending line.

The two directions below compose: A handles the common case generously; B handles the genuinely-unparseable case loudly.

## Proposed approach

### Direction A — permissive name extraction (replaces the regex)

Replace the strict `heading_re` match with a two-step extraction:

1. Locate any `###` line.
2. Strip Markdown inline decoration from the heading text: backticks, `*`, `_`, and `~`. Then split on whitespace and on opening `(` / `[`. The line matches when the **first content token** equals the stage name (case-sensitive, exact).

Pseudocode:

```python
DECORATION_CHARS = str.maketrans('', '', '`*_~')

def heading_first_token(line):
    if not line.startswith('### '):
        return None
    rest = line[4:].strip()
    rest = rest.translate(DECORATION_CHARS)
    # Treat opening '(' or '[' as token terminators so annotations don't merge.
    for ch in '([':
        rest = rest.replace(ch, ' ')
    parts = rest.split()
    return parts[0] if parts else None

def matches_stage(line, stage_name):
    return heading_first_token(line) == stage_name
```

This survives every plausible decoration the captain checklist enumerates: italic (`*` / `_`), bold (`**` / `__`), backticks, mixed combinations, square-bracket annotations, parentheticals, and trailing em-dash text. It does not require ad-hoc regex extensions per decoration style.

Why "first content token" rather than "appears anywhere"? It rejects false positives like `### finalize work` matching `work`, and `### see also: work` matching `work`. The stage name must be the heading's primary identifier.

### Direction B — fail-loud diagnostic (replaces silent `None`)

In `extract_stage_subsection`, if the permissive scan from Direction A finds no match BUT some `###` line in the README contains the stage name as a substring, raise `ValueError` with:

- the line number,
- the raw heading text,
- the parsing requirement: "stage name must be the first content token of the heading after stripping Markdown decoration".

If no `###` line mentions the stage name at all, return `None` as today (genuinely missing stage).

`cmd_build` (currently at `claude-team:269-271`) must catch this `ValueError` and surface it via `_build_error` so the captain sees a structured diagnostic on stderr instead of silent fallthrough. The error message must include the offending heading line verbatim.

## Acceptance criteria

- AC-1: Any `###` heading whose first content token (after stripping `` ` ``, `*`, `_`, `~` and treating `(` / `[` as token terminators) equals the stage name is parsed as a match by `extract_stage_subsection`.
  Verified by: `TestExtractStageSubsection` test cases covering bare, backtick, italic-wrapped, bold-wrapped, mixed-decoration, square-bracket-annotation, and trailing-text-after-name forms all return non-`None` and start at the correct heading.
- AC-2: A `###` heading that mentions the stage name as a substring but where the stage name is NOT the first content token causes `extract_stage_subsection` to raise `ValueError` whose message contains the line number, the raw heading text, and the parsing requirement.
  Verified by: `TestExtractStageSubsection` test for cases like `### see also: work` (when looking for `work`) raises `ValueError` and asserts the message content.
- AC-3: A README with no `###` line mentioning the stage name at all returns `None` (preserves current behavior for genuinely-missing stages).
  Verified by: existing `test_rejects_nonexistent_stage` continues to pass.
- AC-4: Partial-name matches (e.g. `wor` against `### work`) do not match.
  Verified by: existing `test_rejects_partial_match` continues to pass; first-token equality check enforces exact match.
- AC-5: `cmd_build` catches `ValueError` from `extract_stage_subsection` and surfaces it via `_build_error` (stderr + non-zero exit), with the offending heading line included in the diagnostic.
  Verified by: end-to-end subprocess test in `TestBuildStageHeadingParentheticalE2E` invokes `claude-team build` against a README with an unparseable heading and asserts non-zero exit + the heading text appearing in stderr.
- AC-6: All four real-world examples from GH #178 issue body parse successfully.
  Verified by: a parameterized test using the four heading strings from #178 issue body (italic-wrapped annotations) returns non-`None` from `extract_stage_subsection`.
- AC-7: All seven existing `TestExtractStageSubsection` tests continue to pass without modification (regression guard for the #138 / PR #145 plain-parenthetical fix).
  Verified by: pytest run of `tests/test_claude_team.py::TestExtractStageSubsection` is green with no test changes.

## Test plan

Add tests to `tests/test_claude_team.py::TestExtractStageSubsection`:

- Permissive parser (Direction A):
  - `test_matches_italic_wrapped_parenthetical`: `### \`brainstorm\` *(captain-interactive — no ensign)*` matches `brainstorm`.
  - `test_matches_underscore_italic_parenthetical`: `### work _(initial)_` matches `work`.
  - `test_matches_bold_wrapped_name`: `### **work**` matches `work`.
  - `test_matches_bold_annotation`: `### name **(annotation)**` matches `name`.
  - `test_matches_mixed_decoration`: `` ### **`name`** *(annotation)* `` matches `name`.
  - `test_matches_square_bracket_annotation`: `### name [terminal]` matches `name`.
  - `test_matches_trailing_text_after_name`: `### work — does the thing` matches `work`.
  - `test_matches_real_world_178_examples` (parameterized): the four headings from GH #178 issue body all match.
- Fail-loud (Direction B):
  - `test_raises_when_stage_mentioned_but_not_first_token`: `### see also: work` raises `ValueError` when looking for `work`; assertion checks message contains the line number, the raw heading text, and the phrase "first content token".
  - `test_returns_none_when_stage_truly_absent`: README with no mention of the stage name returns `None` (NOT raise).
- E2E (Direction B integration with `cmd_build`):
  - `TestBuildStageHeadingParentheticalE2E::test_build_surfaces_unparseable_heading_diagnostic`: subprocess invocation of `claude-team build` against a README with `### see also: work` (looking for `work`) exits non-zero and stderr includes the heading line and parsing requirement.

Regression guard: existing seven `TestExtractStageSubsection` tests must remain unchanged and green.

## Cross-references

- Builds on #138 / PR #145 (original parenthetical-tolerance fix — preserved by AC-7)
- GH #178 issue body has the four real-world heading examples (drives AC-6)

## Stage Report: ideation

- DONE: Pick a permissive-extraction approach (Direction A) and justify against the brittleness critique
  See "Direction A — permissive name extraction" section: strip-decoration + first-content-token equality, justified against the captain checklist of decoration variants.
- DONE: Pick a fail-loud diagnostic shape (Direction B) and integrate with cmd_build
  See "Direction B — fail-loud diagnostic" section: `extract_stage_subsection` raises `ValueError` with line number / raw heading / parsing requirement; `cmd_build` (claude-team:269-271) catches and surfaces via `_build_error`.
- DONE: Test plan covering both directions
  See "Test plan" section: 8 new permissive-parser tests + 2 fail-loud tests + 1 E2E test; 7 existing tests preserved as regression guard (AC-7).
- DONE: AC items as end-state properties with `Verified by:` clauses
  See "Acceptance criteria" AC-1..AC-7; each has a `Verified by:` clause naming the specific test or test class.
- DONE: Add captain-reframe section near top documenting brittleness critique
  See "Captain reframe (2026-04-30)" section quoting the captain's verbatim critique and the two-layer reframe.
- DONE: Replace "Suggested fix" sketch with "Proposed approach" reflecting both directions
  Original "Suggested fix" code block removed; replaced by "Proposed approach" with Direction A pseudocode and Direction B integration spec.

### Summary

Reframed the ideation away from the original brittle regex-extension toward a two-direction approach: a permissive first-content-token extractor (Direction A) that survives any reasonable Markdown decoration, plus a fail-loud `ValueError` diagnostic (Direction B) integrated with `cmd_build` so unparseable-but-mentioned headings surface a structured error instead of silent fallthrough to break-glass. AC items are end-state properties with explicit `Verified by:` clauses; test plan covers both directions plus regression preservation of the seven existing #138/PR-145 tests. No prior stage report existed, so no feedback cycle marker was needed.

## Stage Report: implementation

- DONE: Replace `extract_stage_subsection` regex match with the permissive Direction A approach (strip decoration → check first content token equals stage name)
  Implemented `_heading_tokens` + `_heading_first_token` helpers; `extract_stage_subsection` now matches when the heading's first content token equals `stage_name` after stripping `` ` *_~ `` and treating `(` / `[` as token terminators. Commit `c0e4b8f6`.
- DONE: Add Direction B fail-loud path: raise `ValueError` with line number + raw heading + parsing requirement when stage name appears as a heading token but not the first one; return `None` when genuinely absent
  Secondary scan uses `_heading_tokens` (token-level, not raw substring) so partial-name lookups like `wor` against `### \`work\`` still return `None`. Error message includes line number, `repr(heading)`, and the phrase "first content token". Commit `c0e4b8f6`.
- DONE: Update `cmd_build` to catch `ValueError` from `extract_stage_subsection` and surface via `_build_error`
  `try`/`except ValueError` wraps the call at `claude-team:269-273` and forwards `str(e)` to `_build_error`, giving the captain a structured stderr diagnostic with non-zero exit. Commit `c0e4b8f6`.
- DONE: Add regression tests to `tests/test_claude_team.py::TestExtractStageSubsection` per entity test plan (8 new permissive-parser tests + 2 fail-loud) + E2E test in `TestBuildStageHeadingParentheticalE2E`
  Added 11 unit tests (italic, underscore-italic, bold-name, bold-annotation, mixed-decoration, square-bracket, trailing-text, 4 parameterized real-world #178 cases, fail-loud-when-mentioned, return-None-when-absent) and 1 subprocess E2E test (`test_build_surfaces_unparseable_heading_diagnostic`). Existing 7 TestExtractStageSubsection tests + existing E2E test unchanged and green. Commit `2c5d7ca6`.
- DONE: Local verification — `unset CLAUDECODE && uv run pytest tests/test_claude_team.py -v` and `make test-static`
  `tests/test_claude_team.py`: 105 passed (22 in the two affected classes). `make test-static`: 553 passed, 26 deselected, 15 subtests passed in 28.87s.

### Summary

Implemented Direction A (permissive first-content-token extractor) and Direction B (fail-loud `ValueError` for headings that mention the stage name but don't parse) in `skills/commission/bin/claude-team`, plus `cmd_build` `try`/`except` to surface the diagnostic on stderr. Refined Direction B's substring check to operate on tokens rather than raw substring so the existing `test_rejects_partial_match` (`wor` against `### \`work\``) continued to return `None` instead of raising — only headings that contain the stage name as a real word now trigger fail-loud. All 7 prior `TestExtractStageSubsection` tests + the prior E2E test remain unchanged and green. Two commits per stage discipline: `c0e4b8f6` (source) and `2c5d7ca6` (tests).
