---
id: f2yr32fgw3pfxp7ekq4wy1np
title: Mod files live with the workflow definition (docs/dev/_mods/), not the state checkout
status: backlog
source: captain (2026-06-01) — "mods live with the workflow definition"
started:
completed:
verdict:
score: 0.34
worktree:
issue:
---

The status tool resolves mods from the **entity** root, so in a split-root workflow mods must live in the state checkout (`docs/dev/.spacedock-state/_mods/`) — not next to the workflow definition (`docs/dev/README.md`). Captain decision: **mods belong with the workflow definition.** Change `scanMods` to read the definition dir, and migrate the `pr-merge` mod there.

## Current behavior (verified in code)
- `scanMods(entityDir)` joins `entityDir/_mods` (`internal/status/mutate.go:282-283`).
- Callers pass the **entity** dir: `boot.go:190` (`scanMods(entityDir)`), `handlers.go:129` (`scanMods(roots.entityDir)`), `mutate.go:197` (`scanMods(entityDir)`).
- In split-root (`README state: .spacedock-state`, `boot.go` `entityDir != definitionDir`), `entityDir` = `.spacedock-state`, so only `.spacedock-state/_mods/` registers; a mod in `docs/dev/_mods/` is never scanned (proven by the `pr-merge-mod` entity's probe).
- In a NON-split-root workflow `entityDir == definitionDir`, so today's behavior already puts `_mods/` in the workflow dir — only split-root diverges.

## Why definition-dir is right
Mods are **workflow definition** — lifecycle hooks declared for the workflow, kin to the README's `stages`, not mutable per-entity state. They should sit next to the README (`docs/dev/_mods/`), travel with the definition (the main repo / the binary's view of the workflow), and not be scattered into the state checkout. The current entity-dir coupling optimizes for "mods travel with state on resume," but the definition (README) already lives in the main repo, so a 2nd host that has the repo + the state branch gets mods (definition) + entities (state) correctly either way.

## The change
Resolve `_mods/` from `definitionDir`, not `entityDir`, at all three call sites (`boot.go:190`, `handlers.go:129`, `mutate.go:197`). Non-split-root is unaffected (`definitionDir == entityDir`). Decide whether to ALSO scan `entityDir` for one release of back-compat, or do a clean cutover (lean clean — there is exactly one mod and one split-root workflow today).

## Coupling — sequence carefully
The `pr-merge` mod currently lives at `.spacedock-state/_mods/pr-merge.md` (correct for *today's* tool — the `pr-merge-mod` entity targeted where the tool actually scans). This change MUST migrate it to `docs/dev/_mods/pr-merge.md` **atomically with the tool change** — otherwise the mod stops registering the moment `scanMods` switches to `definitionDir` and the merge guard silently goes dark. Update the `pr-merge-mod` entity's "where it lives" note to match.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — `scanMods` reads the workflow definition dir.** End state: in a split-root workflow, a mod in `docs/dev/_mods/` registers (`status --boot` shows it under MODS, the terminal guard fires) and a mod in `.spacedock-state/_mods/` does NOT. Verified by: a Go test over the split-root boot/guard path asserting the definition-dir mod registers and the state-dir one does not (inverse of the current `pr-merge-mod` probe).

**AC-2 — Non-split-root workflows are unaffected.** Verified by: existing single-root mod/guard tests (`archive_guard_test.go` / `native_guard_test.go`) stay green (where `definitionDir == entityDir`, `_mods/` resolves the same).

**AC-3 — The `pr-merge` mod is migrated, no registration gap.** End state: `docs/dev/_mods/pr-merge.md` exists and registers; `.spacedock-state/_mods/pr-merge.md` is removed; the merge guard still fires from the new location. Verified by: `status --boot` MODS shows `merge: pr-merge` after the move; a terminal `--set`/`--archive` still refuses while `pr`+`mod-block` empty.

## Notes
- Touches `internal/status` (scanMods call sites) + a file move in `_mods/` → ships via a normal PR onto `next`.
- Depends on / coordinates with `pr-merge-mod` (which installed the mod at the current state-checkout location). Land this after the `pr-merge-mod` pilot so the live 38 PR isn't disrupted mid-flight.
- Small, focused; off the immediate 0.19.2 critical path (the mod works at its current location until this lands).
