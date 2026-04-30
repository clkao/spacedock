---
id: jaafh7xzz0va63rj6bgdgh3p
title: "claude-team build stage-heading regex doesn't tolerate italic-wrapped parentheticals (residual #138 case)"
status: backlog
source: "GitHub issue #178 (filed by FO from captain report, 2026-04-30)"
started:
completed:
verdict:
score: 0.55
worktree:
issue: "#178"
pr:
mod-block:
---

The `extract_stage_subsection` regex in `skills/commission/bin/status` was hardened in #138 / PR #145 to tolerate trailing parentheticals like `### \`triaged\` (terminal)`. It does NOT tolerate italic-wrapped parentheticals like `### \`brainstorm\` *(captain-interactive — no ensign)*`.

Real-world repro from a captain who hit it: four stage headings in `docs/feature-exploration/README.md` with italic-wrapped annotations. Workaround was manual italic-strip (commit `6fd804c` in another repo). Before the strip, every dispatch into one of those stages had `claude-team build` returning empty / FO falling into break-glass manual template.

Current regex:
```python
heading_re = re.compile(
    rf'^###\s+`?{re.escape(stage_name)}`?(?:\s+\([^)]*\))?\s*$'
)
```

Suggested fix:
```python
heading_re = re.compile(
    rf'^###\s+`?{re.escape(stage_name)}`?(?:\s+[*_]?\([^)]*\)[*_]?)?\s*$'
)
```

This adds optional `*` or `_` flanking the parenthetical (covers italic via `*` or `_`, common in Markdown style). Bold (`**(...)**`) is out of scope — file separately if it shows up.

Test home: `tests/test_claude_team.py::TestExtractStageSubsection` already has 4 regression tests for plain-parenthetical case. Add 1-2 more covering italic-wrapped form.

## Cross-references

- Builds on #138 / PR #145 (original parenthetical-tolerance fix)
- GH #178 issue body has the four real-world heading examples
