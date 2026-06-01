---
id: s0cbyrbh9b2yb456exnsm72b
title: status correctness — --next undercount, archive enumeration consistency, --set stage-name validation
status: ideation
source: issue sweep (2026-05-31) — CL dev-workflow-ergonomics triage; consolidates #230, #207, #189, #163
started: 2026-06-01T05:04:22Z
completed:
verdict:
score: "0.28"
worktree:
issue:
---

Fix three correctness/ergonomics gaps in `spacedock status` that mislead the FO's scheduling and
let bad state in silently. These are read/scheduling-path bugs the FO hits every session.

Consolidates open issues:
- **#230** — `status --next` undercounts: when an entity's stage report is committed but the entity
  has not been advanced, it is not surfaced as dispatchable, so ready work stays invisible.
- **#207** — entity enumeration is inconsistent depending on whether an entity lives at top-level
  vs in `_archive/`; the helper should enumerate consistently.
- **#189** — `status --set {slug} status=X` does not validate `X` against the workflow's declared
  `stages.states[].name` membership. **Scope note:** `internal/status/validate.go`
  (`validateWorkflowStageNames`) already validates the README stage-name *format* against a regex —
  this AC is the distinct **membership** check on a `--set` value, not the format check; do not
  duplicate the regex validation.
- **#163** — `status --boot` crashing on an inline-comment value line was a Python-`ValueError` and
  is gone in the Go rewrite; this entity only needs to **verify** the Go parser correctly strips an
  inline `key: value  # comment` rather than re-assert the crash fix.

## Acceptance criteria

**AC-1 — A committed-but-unadvanced stage report does not hide a ready entity from `--next`.**
Verified by: a fixture where an entity has a committed stage report at a non-terminal stage; assert
`status --next` surfaces it (or documents why it should not).

**AC-2 — Enumeration is consistent across top-level and `_archive/` placement.**
Verified by: a golden/behavioral test comparing enumeration of equivalent entities in both locations.

**AC-3 — `status --set status=X` rejects X when X is not in the workflow's declared stages.**
Verified by: a test asserting a non-member stage value exits non-zero with an actionable error,
while a valid member succeeds.

**AC-4 — The Go frontmatter parser strips an inline comment on a value line.**
Verified by: a parser test on `key: value  # comment` asserting the value is `value`.

## Test plan

Go unit/golden tests in `internal/status/`. Ideation should confirm #230's intended behavior (is a
committed-report-but-unadvanced entity *meant* to be dispatchable, or is the report itself the bug?)
before implementation, and reconcile against the Python oracle's enumeration for parity where it
still binds.
