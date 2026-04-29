---
id: 223
title: "id-style change should be forward-only — accept legacy sequential IDs when README declares sd-b32"
status: ideation
source: "GitHub issue #169 (filed by CL, 2026-04-29)"
started: 2026-04-29T21:21:01Z
completed:
verdict:
score: 0.6
worktree:
issue: "#169"
pr:
mod-block:
---

The `skills/commission/bin/status` validator hard-rejects mixed id formats: when README declares `id-style: sd-b32`, every entity must hold a valid sd-b32 stored id; legacy entities with numeric ids fail validation. Captains who want sd-b32 going forward have no clean path short of migrating every existing entity (200+ for this workflow).

The display layer already handles mixed gracefully (line 656: `apply_effective_ids` falls back to `stored_id` when not in the sd-b32 display dict). Only the validator (~lines 701-714) enforces single-style.

Make the style change **forward-only**: when `id-style: sd-b32` is declared, the validator accepts sd-b32 ids AND legacy numeric ids per-entity. New entities get sd-b32 (existing `compute_next_id` behavior, unchanged). Cross-references in old entity bodies stay correct because old ids don't change.

The driving use case is this very workflow: 50 active + 175 archived sequential entities, considering a forward switch to sd-b32 without disturbing history.

Issue body: https://github.com/clkao/spacedock/issues/169.
