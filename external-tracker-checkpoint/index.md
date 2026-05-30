---
id: p31ep68fer46hhms1pvp3b6f
title: Check external tracker integration point
status: backlog
score: "0.25"
source: bootstrap roadmap
worktree:
---

# Check External Tracker Integration Point

Evaluate whether the split-root design gives clean integration points for kata, Linear-style tickets, and other external trackers.

## Acceptance Criteria

- A fixture entity can carry a simple external reference using top-level fields.
- Status, mutation, reports, and archive preserve the external reference.
- The design identifies which system owns execution status.
- No tracker-specific stage semantics are added to Spacedock.

## Test Gates

- `go test ./...`
- Fixture entity with `issue: ENG-123` and `source: linear` survives status and `--set`.
- Fixture entity with `issue: kata:task-abc123` and `source: kata` survives archive.
- Documentation states that richer tracker sync belongs in an adapter until a bridge contract exists.

## Notes

This checkpoint should happen after native split-root status works. It is not a blocker for the symlink compatibility phase.
