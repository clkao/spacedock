---
id: segh9j67xb7hv1qxgqzxe90g
title: Ship the team's proven working habits in the tool's own instructions
status: implementation
source: hx decomposition (B of 3) — captain 2026-06-01; staff review
score: "0.30"
started: 2026-06-01T06:04:55Z
completed:
verdict:
worktree: .worktrees/spacedock-ensign-ship-working-principles-in-contract
issue:
---

**What this is for (plain).** The ways of working we've learned the hard way currently live in one
person's personal settings or their head — so a teammate on a fresh setup, or a published copy of the
tool, doesn't get them. This writes those habits into the tool's own shipped instructions, so anyone
who installs it follows the same discipline with no personal configuration required.

**The habits to encode (plain).** Every task must produce a real, checkable change — not just a
document. Prove things by exercising them, not by re-reading notes. Try the riskiest unknown early,
before committing to a plan. And how the FO operates: name the end value before starting, lead with a
clear recommendation the captain can say yes to, and just do the obvious reversible work without
ceremony.

**Value to the user / FO.** The discipline travels with the tool, not in one machine's config. A new
contributor — or a clean-room install with no personal settings — is governed by the same way of
working, so quality doesn't depend on who happens to be running it.

This is part B of three from the now-superseded parent `deliverable-contract-hardening`
(id `hxs93wd0bjwhc3vsjwx1seew`). This child is the PROSE half — instruction-text edits only.

## Scope (ideation hardens; coordinated edits)

- Edits to the shipped instruction files: the workflow guide (`docs/dev/README.md`), the FO operating
  contract (`first-officer-shared-core.md`), and the worker contract (`ensign-shared-core.md`) — the
  four principles, the "no hidden dependencies" principle, the FO posture, the "write the failing test
  first" rule, and the small task-template ergonomics. Plain language throughout (no insider jargon in
  the shipped text — the word "oracle" appears nowhere in the shipped files).
- **Coordination:** the FO-contract edits touch the same file other tasks edit — keep to your own
  paragraphs (the gate-check paragraph + the spike-first bullet) and **pin the FO-posture section's
  exact location with `zs` BEFORE implementation**, so the one coordinated file doesn't collide.

## Acceptance criteria (provisional)

**AC-1 — The shipped instructions carry the principles + FO posture + the test-first rule, in plain
language.** Verified by: the named edits are present in the three shipped files; a simple text-presence
test confirms the shipped instruction files contain zero insider-jargon tokens (the plain-language
guarantee). Honest ceiling: this is wording-present, not behavior — the behavioral teeth are part A's
guard and the FO's gate check; this task ensures the discipline is written down where a clean-room
worker reads it.

**AC-2 — Nothing project-specific is lost.** Verified by: an FO structural pass confirms the workflow
guide keeps its existing project content (the split-root model, the testing table, the native command
surface, the project-specific good/bad examples) and only adds the new slots.

## Out of scope
- The code guard (part A) and the portability test (part C).
- The README's keep-or-remove decision on the during-migration compatibility commands is a **captain
  sub-decision** to surface at the ideation gate, not a silent cleanup.
