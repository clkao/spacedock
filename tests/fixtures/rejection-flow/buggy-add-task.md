---
id: "001"
title: Wire up the add function in math_ops.py
status: backlog
score: 0.90
source: test
started:
completed:
verdict:
worktree:
---

`math_ops.py` already exists at the repo root with an `add(a, b)` implementation. Verify it is present in the worktree, run `tests/test_add.py` to record the current pass/fail state in your stage report, and commit the file unchanged.

DO NOT modify `math_ops.py` during the implementation stage. Validation will review the test output and determine whether the acceptance criteria are met. If validation REJECTS, a subsequent implementation cycle will address the failing cases.

## Acceptance Criteria

1. `add(2, 3)` returns `5`
2. `add(-1, 1)` returns `0`
3. `add(0, 0)` returns `0`
