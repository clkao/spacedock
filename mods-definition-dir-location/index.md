---
id: f2yr32fgw3pfxp7ekq4wy1np
title: Mod files live with the workflow definition (docs/dev/_mods/), not the state checkout
status: ideation
source: captain (2026-06-01) — "mods live with the workflow definition"
started: 2026-06-01T04:46:06Z
completed:
verdict:
score: 0.34
worktree:
issue:
---

The status tool resolves mods from the **entity** root, so in a split-root workflow mods must live in the state checkout (`docs/dev/.spacedock-state/_mods/`) — not next to the workflow definition (`docs/dev/README.md`). Captain decision: **mods belong with the workflow definition.** Change `scanMods` to read the definition dir, and migrate the `pr-merge` mod there.

## Current behavior (verified in code)
- `scanMods(entityDir)` joins `entityDir/_mods` (`internal/status/mutate.go:282-283`).
- Three call sites pass the **entity** dir today:
  - `boot.go:190` — `gatherBoot` already receives both `definitionDir` and `entityDir`; it currently calls `scanMods(entityDir)`. The fix is a one-token swap to `scanMods(definitionDir)`.
  - `handlers.go:129` — `runSet`'s terminal merge-hook guard calls `scanMods(roots.entityDir)`; `roots.definitionDir` is in scope (same `roots` struct), so swap to `scanMods(roots.definitionDir)`.
  - `mutate.go:197` — `runArchive`'s merge-hook guard calls `scanMods(entityDir)`, but `runArchive(entityDir, spellingDir, slug, ...)` does **not** currently receive `definitionDir`. This call site requires a **signature change**: thread `definitionDir` into `runArchive` and have its caller (`native_runner.go:302`, which has `roots.definitionDir` in scope) pass it. This is the only non-trivial mechanical edit.
- In split-root (`README state: .spacedock-state`, `boot.go` `entityDir != definitionDir`), `entityDir` = `.spacedock-state`, so only `.spacedock-state/_mods/` registers; a mod in `docs/dev/_mods/` is never scanned (proven by the `pr-merge-mod` entity's probe).
- In a NON-split-root workflow `entityDir == definitionDir`, so today's behavior already puts `_mods/` in the workflow dir — only split-root diverges.

## Oracle parity — this is a native-only change (verified in `bin/status`)
The Go launcher is a parity reimplementation of the Python oracle (`/Users/clkao/git/spacedock/skills/commission/bin/status`); the merge-guard tests (`archive_guard_test.go`, `native_guard_test.go`) compare native output byte-for-byte against `runOracle`. **The oracle has no split-root concept at all**: it carries a single `pipeline_dir` and resolves the README, entities, AND `scan_mods(pipeline_dir)` all from it (verified: `scan_mods` at line 935, guards at lines 1915/2433, no `state:`/`definition_dir`/`entity_dir` split anywhere). Split-root is **already an intentional native-only divergence** (like the `STATE_BACKEND` boot banner stripped in `harness_test.go:121-133`), and every existing split-root test uses `runNative` only, never `runOracle`. Consequences:
- The existing oracle-parity guard tests run single-root fixtures (`guard-workflow`, `seq-workflow`) where `definitionDir == entityDir`, so `scanMods(definitionDir)` resolves identically — **they stay byte-identical to the oracle**, AC-2 holds with no oracle change.
- The new split-root behavior is asserted with **native-only** tests (no `runOracle`), consistent with the established split-root test convention. The oracle does not change and is not consulted for this behavior.

## Why definition-dir is right
Mods are **workflow definition** — lifecycle hooks declared for the workflow, kin to the README's `stages`, not mutable per-entity state. They should sit next to the README (`docs/dev/_mods/`), travel with the definition (the main repo / the binary's view of the workflow), and not be scattered into the state checkout. The current entity-dir coupling optimizes for "mods travel with state on resume," but the definition (README) already lives in the main repo, so a 2nd host that has the repo + the state branch gets mods (definition) + entities (state) correctly either way.

## The change
1. Resolve `_mods/` from `definitionDir`, not `entityDir`, at all three call sites (`boot.go:190`, `handlers.go:129`, `mutate.go:197` + the `runArchive` signature/caller change at `native_runner.go:302`). Non-split-root is unaffected (`definitionDir == entityDir`).
2. **Clean cutover, no back-compat dual scan.** There is exactly one mod and one split-root workflow today; scanning both dirs would re-introduce the state-checkout `_mods/` path the captain is removing and add a precedence rule to reason about. Per the global YAGNI/no-backward-compat rule, do the clean swap. (Back-compat dual scan would need explicit captain approval.)
3. **Atomic migration of `pr-merge.md`.** The migration (file move) and the `scanMods` swap MUST land in the **same commit/PR** — the moment `scanMods` switches to `definitionDir`, a mod still sitting only at `.spacedock-state/_mods/` goes dark, and the merge guard stops firing. Use `git mv .spacedock-state/_mods/pr-merge.md docs/dev/_mods/pr-merge.md` in the same change as the code edit. Note: `pr-merge.md` lives in the **orphan state-branch** checkout (`.spacedock-state` tracks `spacedock-state/dev`), while the code edits and `docs/dev/_mods/` live in the **main repo** — so the "move" is really *delete from state branch* + *add to main repo*, two index operations that the implementation must coordinate, not a single in-tree rename. The registration gap is closed as long as both land before the change is exercised on a live workflow.

## Carried fix — jx's audit-link orphan-branch correction (entity `jxg4re5argq10x5y87sjbzkz`)
`pr-mod-audit-link-orphan-branch` (jx, in ideation concurrently) fixes a defect in the SAME `pr-merge.md` body: its PR audit link points at `/blob/{code-short-sha}/{path}`, but the entity file lives on the orphan state-branch, so the link 404s; the fix is `/blob/spacedock-state/dev/{state-relative-path}`. **This entity is the MOVER**, and both entities' notes agree the mover carries the fix. To avoid serializing two edits to the same off-limits `_mods/` file, the implementation applies jx's audit-link edit to `pr-merge.md` **in the same relocation commit** (move + audit-link edit + scanMods swap = one PR). jx's entity remains the source of truth for the audit-link AC and its HTTP-200 verification; this entity does not re-specify it. Land order: fold jx's fix into the relocation PR (a prior in-place edit would just be re-moved). *(Confirmation requested from jx; both entity notes already direct the mover to carry it.)*

## Acceptance criteria

**AC-1 — `scanMods` reads the workflow definition dir under split-root.** End state: in a split-root workflow (`README state: .spacedock-state`), a mod placed at `<definitionDir>/_mods/` registers — `status --boot` lists it under MODS and the terminal merge guard fires on a `--set status=<terminal>` / `--archive` while `pr` and `mod-block` are empty — and a mod placed only at `<entityDir>/_mods/` (the state checkout) does NOT register and does NOT arm the guard. Verified by: a native-only Go test (no `runOracle`) building a split-root fixture via the existing `buildSplitRoot` helper, asserting (a) a definition-dir merge mod makes a terminal transition exit 1 with the merge-hook error and (b) the same mod placed only in the state checkout lets the terminal transition succeed (exit 0). This is the inverse of the current `pr-merge-mod` probe.

**AC-2 — Non-split-root workflows and oracle parity are unaffected.** End state: in a single-root workflow (`definitionDir == entityDir`) mod scanning resolves the same dir as before. Verified by: existing oracle-parity guard tests (`archive_guard_test.go`, `native_guard_test.go`) stay green byte-for-byte against `runOracle` (single-root `guard-workflow`/`seq-workflow` fixtures where the dirs coincide).

**AC-3 — The `pr-merge` mod is migrated with no registration gap.** End state: `docs/dev/_mods/pr-merge.md` exists and registers in this repo's split-root `docs/dev` workflow; `.spacedock-state/_mods/pr-merge.md` is removed; the merge guard still fires from the new location. Verified by: `spacedock status --boot --workflow-dir docs/dev` MODS shows `merge: pr-merge`; a terminal `--set`/`--archive` on a `pr`-empty / `mod-block`-empty entity is refused with the merge-hook error. (Mechanism-first: run this smallest live boot+guard check on the real `docs/dev` workflow before declaring done.)

**AC-4 — The carried audit-link fix is present in the relocated mod.** End state: the relocated `docs/dev/_mods/pr-merge.md` contains jx's orphan-branch audit-link form, not the code-SHA form. Verified by: jx's entity (`jxg4re5argq10x5y87sjbzkz`) owns the substantive audit-link AC + HTTP-200 check; this entity only asserts the relocated file carries the edit (grep: the body references `/blob/spacedock-state/dev/` and no longer derives the link from `rev-parse --short HEAD`).

## Test plan
- **AC-1 (split-root scan, native-only Go test):** new test in `internal/status` reusing `buildSplitRoot` + `splitRootReadme` (extend the README with a terminal stage already present). Place a merge mod (`## Hook: merge`) under `<def>/_mods/`, assert a terminal `--set`/`--archive` exits 1 with the merge-hook text; move the same mod to `<state>/_mods/` only, assert the terminal transition exits 0. ~1 fixture, two `runNative` calls. Cost: low (unit-level, no live network). This is the riskiest path (the contract being changed) → write it first.
- **AC-2 (parity regression):** no new test; the existing guard tests are the proof. Run `go test ./internal/status` and confirm the oracle-parity guard tests stay green (oracle present, else they `t.Skip`). Cost: zero new code.
- **AC-3 (live migration smoke):** after the `git mv` + code change, build and run `spacedock status --boot --workflow-dir docs/dev` on this repo, confirm `merge: pr-merge` under MODS; attempt a terminal transition on a synthetic `pr`-empty entity (or `--archive` a disposable one with `--force` withheld) and confirm the guard refuses. CLI-level, on the real split-root workflow. Cost: low; pay this small bill first to validate the on-disk migration before the comprehensive suite.
- **AC-4 (carried-edit presence):** grep `docs/dev/_mods/pr-merge.md` for the orphan-branch link form. Trivial. The substantive HTTP-200 verification belongs to jx's entity, not re-run here.

## Notes
- Touches `internal/status` (3 `scanMods` call sites + `runArchive` signature + `native_runner.go:302` caller) and a cross-checkout file move of `_mods/pr-merge.md` (delete on the orphan state branch, add in the main repo) → ships via a normal PR onto `next`.
- Depends on / coordinates with `pr-merge-mod` (which installed the mod at the current state-checkout location). Land this after the `pr-merge-mod` pilot so the live 38 PR isn't disrupted mid-flight. Update the `pr-merge-mod` entity's "where it lives" note to point at `docs/dev/_mods/` once this lands.
- Coupled with `pr-mod-audit-link-orphan-branch` (jx) — this entity is the mover and carries jx's audit-link fix in the same relocation; see the "Carried fix" section.
- Small, focused; off the immediate 0.19.2 critical path (the mod works at its current location until this lands).

## Staff review
**Warranted.** This is a split-root behavior change with three call sites (one needing a signature change), an oracle-parity argument (native-only divergence), and an atomic cross-checkout migration with a registration-gap hazard plus a carried fix from a sibling entity. Per the ideation stage's staff-review guidance ("native status parity / split-root behavior" are named complexity triggers), an independent review of the design soundness, the oracle-parity claim, and the migration/land-order sequencing is appropriate before the ideation gate.

## Stage Report: ideation

- DONE: Design the scanMods change so the mod scanner reads the workflow DEFINITION dir (docs/dev/_mods/) instead of the state checkout (.spacedock-state/_mods/) under split-root, and the ATOMIC migration of the existing pr-merge mod from .spacedock-state/_mods/ to docs/dev/_mods/ with no registration gap.
  "The change" section: 3 call sites (boot.go:190, handlers.go:129 swap to definitionDir; mutate.go:197 needs runArchive signature change threading definitionDir from native_runner.go:302), clean cutover (no dual-scan back-compat), atomic git-mv-in-same-commit; verified the cross-checkout reality (state branch delete + main-repo add) in code.
- DONE: Behavioral AC: under a split-root fixture, the mod scanner finds mods in docs/dev/_mods/ (not the state checkout); the pr-merge merge hook stays registered through the migration (a terminal-transition guard test still sees the hook).
  AC-1 (native-only Go test via buildSplitRoot: def-dir mod arms guard / state-dir mod does not) + AC-3 (live boot+guard smoke on docs/dev) in the Test plan. Confirmed oracle is single-pipeline_dir (no split-root) so this is a native-only divergence and parity guard tests stay byte-identical (AC-2).
- DONE: Coordinate with jx (pr-mod-audit-link, in ideation concurrently): this entity is the MOVER, so it should carry jx's SHA-based audit-link fix in the same _mods/pr-merge.md relocation — agree the land order with jx. State whether staff review is warranted.
  Sent SendMessage to jx proposing fold-into-relocation land order; documented in "Carried fix" section + AC-4 citing jx's entity jxg4re5argq10x5y87sjbzkz as source of truth. Staff review assessed WARRANTED (split-root + parity + migration complexity). jx reply pending but both entity notes already direct the mover to carry the fix.

### Summary

Hardened the ideation: pinned the exact three scanMods call sites (the runArchive one at mutate.go:197 needs a signature change — the only non-trivial edit), and established the key design fact that the Python oracle has no split-root concept, so this is a native-only divergence and the existing oracle-parity guard tests stay byte-identical. Rewrote ACs to be end-state/behavior-first with a mechanism-first live smoke (AC-3), folded jx's orphan-branch audit-link fix into the relocation as AC-4 (jx owns the substantive AC), and chose clean cutover over dual-scan per YAGNI. Flagged staff review as warranted.
