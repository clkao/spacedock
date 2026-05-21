---
id: 001
title: "Sample entity in no-standing-teammates workflow"
status: implementation
started:
completed:
verdict:
worktree: .worktrees/sample-001
issue:
pr:
---

## Problem

Small fixture entity used to assert `cmd_build` omits the `show-standing`
fetch line for workflows with no `_mods/` directory.
