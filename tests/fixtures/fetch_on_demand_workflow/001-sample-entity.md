---
id: 001
title: "claude-team build emits fetch-on-demand spec; ensign loads stage def + standing section on first action (fixture mirror)"
status: ideation
started:
completed:
verdict:
worktree: .worktrees/sample-001
issue:
pr:
---

## Problem

A canonical fixture entity used by `tests/test_claude_team.py` and
`tests/test_fetch_on_demand_dispatch.py` to exercise the fetch-on-demand
dispatch shape. The body deliberately mirrors a real ideation-stage dispatch
input so prompt-size measurements match production scale.

## Proposal

Run the canonical dispatch path against this entity and assert the emitted
prompt references fetch commands rather than inlining the stage definition.
