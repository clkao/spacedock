---
id: "212"
title: "claude-team build rejects stage headings with trailing parentheticals"
status: validation
source: "GitHub issue #138 (filed by CL) — external workflow in CL's env with two-stage `new` → `triaged` (terminal) pipeline; helper rejects the terminal heading and forces break-glass dispatch for ~60 entities"
started: 2026-04-21T03:16:05Z
completed:
verdict:
score: 0.55
worktree: .worktrees/spacedock-ensign-claude-team-build-stage-heading-parenthetical
issue: "#138"
pr: #145
mod-block: merge:pr-merge
---

# Fix `claude-team build` stage-heading parser to tolerate trailing parentheticals

## Problem statement

`claude-team build` (at `skills/commission/bin/claude-team`) calls `extract_stage_subsection(readme_path, stage_name)` (line 72) to pull the `### {stage_name}` subsection out of a workflow README. The current heading match is exact-string against two forms:

```python
# skills/commission/bin/claude-team:82
accepted_headings = (f'### `{stage_name}`', f'### {stage_name}')
```

This rejects any heading with trailing content after the stage-name token. Real workflows commonly annotate the terminal stage as `### \`triaged\` (terminal)` for human readability. The YAML frontmatter's `stages.states[].terminal: true` field is the canonical truth source; the parenthetical in the heading is informational.

**Repro** (from issue body):
```bash
echo '{"schema_version":1,"entity_path":"/abs/path","workflow_dir":"/abs/path","stage":"triaged","checklist":["x"],"team_name":null,"feedback_context":null,"scope_notes":null,"bare_mode":true,"is_feedback_reflow":false}' | \
  claude-team build --workflow-dir /abs/path
```
Exits non-zero: `error: stage 'triaged' heading not found in README.md`.

**Scale:** 1 affected workflow in CL's environment (~60 entities through it during the session this surfaced). Other workflows happen to use bare `### \`stage\`` without trailing annotations and work fine.

**Current workaround:** fall back to break-glass manual `Agent()` dispatch per the Claude runtime adapter's documented escape hatch. Verbose but functional.

## Proposed fix

Replace the exact-match list with a regex that:
- Requires `###` + whitespace
- Optionally allows a backtick around the stage name (for the documented `### \`stage\`` convention)
- Matches the stage name literally
- Optionally allows a trailing whitespace + `(...)` parenthetical annotation
- Allows any trailing whitespace

Suggested shape (from the issue body):
```python
import re
heading_re = re.compile(rf'^###\s+`?{re.escape(stage_name)}`?(?:\s+\([^)]*\))?\s*$')
```

Apply when scanning `lines`; match `heading_re.match(line.strip())` (or compile once outside the loop).

Preserve the section-end sentinel logic unchanged (`### ` or `## ` starts a new section). Preserve the trailing-blank-line trim.

## Acceptance criteria

**AC-1 — Regex matches all documented heading forms without regression.**
Verified by: `tests/test_claude_team.py::TestExtractStageSubsection` (new class) asserts `extract_stage_subsection` returns the expected subsection for each of:
- `### \`work\`` (original backtick-quoted form)
- `### work` (original bare form)
- `### \`triaged\` (terminal)` (backtick + trailing annotation — the bug)
- `### triaged (terminal)` (bare + trailing annotation)
- `### \`work\` (middle)` (arbitrary annotation text)
Each assertion checks the returned subsection starts with the matching heading line and ends before the next `###`/`##` sibling section.

**AC-2 — Regex rejects unrelated stage names.**
Verified by: the same test class asserts `extract_stage_subsection(readme, 'nonexistent')` returns `None`, and `extract_stage_subsection(readme, 'wor')` (partial match) returns `None` when `### \`work\`` is present. This guards against accidental partial-match via missing anchors.

**AC-3 — Existing `claude-team build` end-to-end behavior unchanged for current workflows.**
Verified by: `make test-static` stays green at ≥ current passing count (current main: 485 passed post-#211 merge).

**AC-4 — The repro from the issue succeeds.**
Verified by: a temp-dir fixture constructing a minimal `README.md` with `### \`triaged\` (terminal)` and a temp entity file, piping the exact JSON payload from the issue body into `claude-team build`, asserts exit 0 and stdout contains the `triaged` subsection in the emitted prompt.

## Out of scope

- Broader prompt-assembly refactor of `cmd_build`.
- Tolerance for non-`###` heading levels (e.g., `## stage` — workflows commission `### stage` by convention; no need to widen).
- Backtick-inside-the-name forms like `### work-space` — current regex with `re.escape(stage_name)` handles this; no expansion needed.
- Changing the `stages.states` YAML schema.

## Test plan

Unit tests (fast, offline) in `tests/test_claude_team.py`:
- 1 test class `TestExtractStageSubsection` with ~6 assertions covering the 5 heading forms in AC-1 + the 2 rejection cases in AC-2.
- 1 end-to-end test invoking `claude-team build` as a subprocess with the repro fixture from AC-4.

Estimated cost: near-zero (offline static, no live-runtime needed). Fits in `make test-static`; no CI-matrix live run needed.

## Commit discipline

- `fix(#138): claude-team build tolerate trailing parenthetical in stage heading` — the regex swap in `skills/commission/bin/claude-team`
- `test(#138): cover stage-heading regex against backtick + parenthetical variants` — the new test class + e2e test in `tests/test_claude_team.py`
- `report: #212 implementation stage report — GitHub #138 closed` — stage report

## Cross-references

- GitHub issue #138 (this PR will close it)
- `skills/commission/bin/claude-team:72-97` — target function
- `tests/test_claude_team.py` — test home
