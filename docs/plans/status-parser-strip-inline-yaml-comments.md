---
id: 222
title: "status --boot crashes with ValueError when stage YAML has inline comment on value line"
status: ideation
source: "GitHub issue #163 (filed by Kent Chen / iamcxa, 2026-04-29)"
started: 2026-04-29T14:56:54Z
completed:
verdict:
score: 0.65
worktree:
issue: "#163"
pr:
mod-block:
---

The hand-rolled YAML parser in `skills/commission/bin/status` (around line 266 in `parse_stages_block`) crashes with `ValueError: invalid literal for int() with base 10: '2  # teammate-debate mode'` when a stage frontmatter has an inline `#` comment on a numeric value line.

CRITICAL when triggered — blocks all `status` operations on the affected workflow until the comment is moved or stripped. Rare in practice (most captains write README comments above value lines, not inline) but undetectable until first FO boot attempt.

External reporter (Kent Chen / `iamcxa`) provided concrete reproduction and a suggested fix: strip inline `#` from value strings before type-cast in `parse_stages_block`. Same fix applies to all type-cast fields: `worktree`, `gate`, `terminal`, `initial`, `concurrency`.

## Why this matters now

Standard YAML allows inline comments. The hand-rolled parser violates a documented YAML expectation. The class of bug is small (one helper function in `parse_stages_block`) but the failure mode is silent until first boot — a captain commissions a workflow with an inline-commented concurrency value, then the FO can never start.

## Suggested approach

Per the issue's pseudo-fix:

```python
v = state.get('concurrency', str(default_concurrency))
v = v.split('#', 1)[0].strip()
'concurrency': int(v),
```

Apply to every type-cast field in `parse_stages_block`. Add a regression test fixture with a stage README containing inline `#` comments on numeric and boolean fields.
