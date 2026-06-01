---
id: jxg4re5argq10x5y87sjbzkz
title: pr-merge mod audit-link must reference the orphan state-branch, not the code SHA
status: ideation
source: captain (2026-06-01) — pr-mod audit link 404s; entity lives on the orphan branch, not the code commit
started: 2026-06-01T04:46:05Z
completed:
verdict:
score: 0.34
worktree:
issue:
---

The `pr-merge` mod builds the PR body's audit link pointing at the **code worktree's commit** (`git -C {worktree} rev-parse --short HEAD`), but the entity file it links to lives in the `.spacedock-state` checkout — a separate orphan-branch checkout (`spacedock-state/dev`) that the code commit never contains. So `/blob/{code-sha}/{path}` resolves to nothing → 404. Fix the link to reference the **state** checkout, using an immutable state-commit SHA so it survives archival.

## The bug (in `.spacedock-state/_mods/pr-merge.md`)
The audit-link spec (PR-body template row ~line 80 and the Audit-link extraction-rule row ~line 92) is:
```
[{short-id}](/{owner}/{repo}/blob/{short-sha}/{path-to-entity-file})
```
where `{short-sha}` = `git -C {worktree} rev-parse --short HEAD` — the **CODE** worktree's SHA. But the entity file (`{slug}/index.md`) lives in the `.spacedock-state` checkout, whose content is the orphan branch `spacedock-state/dev`. The code commit has zero `.spacedock-state` files, so the link 404s.

## The fix — IMMUTABLE state-commit SHA + path-at-that-SHA (proven live this session)
Build the link against the **state checkout's commit SHA**, not the code SHA, and use the entity's path *as it exists at that SHA*:
```
[{short-id}](/{owner}/{repo}/blob/{state-sha}/{state-relative-path})
```
- `{state-sha}` = `git -C docs/dev/.spacedock-state rev-parse HEAD` (full SHA) — the state checkout's HEAD **at PR-open time**. By the time the merge hook computes audit inputs, the FO has already committed `mod-block=merge:pr-merge` to the state checkout (that commit is state HEAD), so HEAD points at a commit that contains the active entity file. This SHA is on `spacedock-state/dev` and is pushed to `origin` (the FO pushes the state branch on `pr:`), so it resolves on GitHub.
- `{state-relative-path}` = the entity file's path **relative to the `.spacedock-state` root**, at that SHA = the **active** path `{slug}/index.md` (a PR-pending entity has not archived yet). Because the SHA is immutable, this stays valid through archival — the later `_archive/` move creates *new* commits but never rewrites the snapshot at `{state-sha}`.

**Why SHA, not branch-ref.** The original hypothesis (`/blob/spacedock-state/dev/{path}`) was tested live and **404s after the entity archives**: a branch ref always resolves to HEAD's *current* tree, and archival moves `{slug}/index.md` → `_archive/{slug}/index.md`, so the active-path link breaks the moment the entity terminalizes. The immutable SHA captures a snapshot that outlives the move. Proven this session: FO hand-fixed #244 to `/blob/13c5ed75…/host-neutrality-seam/index.md` → **HTTP 200** even though that entity has since archived; the branch-ref form on #241 (`/blob/spacedock-state/dev/cli-cobra-redesign/index.md`) → **HTTP 404** because cli-cobra-redesign archived.

## Decision: SHA-at-PR-open + active path (not archive-SHA)
Use **state-HEAD-at-PR-open** + the **active** `{slug}/index.md` path. Rationale: at the merge hook the entity is necessarily PR-pending (not yet archived), so the active path is the only path that exists; the archive SHA does not exist yet and cannot be referenced. The active-path-at-an-immutable-SHA is exactly the FO's proven form (#244). No need to predict the future archive location.

## Acceptance criteria
**AC-1 — The mod's audit link targets an immutable state-commit SHA and resolves.** End state: `_mods/pr-merge.md`'s audit-link template and the Audit-link extraction-rule row emit `/blob/{state-sha}/{state-relative-path}`, where `{state-sha}` is the full SHA from `git -C docs/dev/.spacedock-state rev-parse HEAD` (the state checkout, NOT the code worktree) and `{state-relative-path}` is the entity's active `.spacedock-state`-relative path. The code-worktree `git rev-parse` for the audit SHA is gone. Verified by: a PR body generated via the mod has an audit link that loads the entity file on GitHub (HTTP 200).
**AC-2 — The link survives archival.** End state: because the link pins an immutable state commit (not a branch ref), it still resolves (HTTP 200) after the entity moves to `_archive/`. Verified by: re-following the audit link for an entity that has archived since PR-open returns HTTP 200 (the live #244 SHA-form 200 / #241 branch-form 404 contrast already demonstrates this).

## Notes
- Tiny `_mods/pr-merge.md` prose edit → dispatched worker (`_mods/` off-limits to direct FO edits).
- The state-checkout path `docs/dev/.spacedock-state` is the workflow-fixed location the mod already references (mod line 40, 64); the SHA-computation step in the mod's merge hook gains a `git -C docs/dev/.spacedock-state rev-parse HEAD` alongside (or replacing, for the audit slot) the existing code-worktree `rev-parse`.
- **Coupling with `mods-definition-dir-location` (f2, in ideation concurrently):** f2 moves the mod to `docs/dev/_mods/pr-merge.md` atomically with a `scanMods` code change. Both this fix and f2 edit `_mods/pr-merge.md`. **Agreed land order: this fix lands FIRST** (it is a self-contained prose edit on the mod at its current location `.spacedock-state/_mods/pr-merge.md`); f2 then carries the already-fixed file when it moves it. If f2 somehow lands first, the f2 mover carries this audit-link fix into the moved file. The FO sequences the two dispatches so the two `_mods/pr-merge.md` edits do not collide.
- **Staff review: not warranted.** Small, single-file prose edit; the SHA approach is already proven live this session (HTTP 200 on #244). No design uncertainty remains.
- `pr-merge-mod` is the parent (this is a follow-up defect on the mod it shipped). Off the 0.19.2 critical path — the mod works (opens PRs, tracks `pr:`); only the audit *link* was broken.

## Test plan
- **AC-1 / AC-2 (live HTTP):** After the mod edit lands and a PR is opened via the mod, `curl -s -o /dev/null -w "%{http_code}" {audit-link-url}` returns `200`. Already demonstrated this session on the FO's hand-fixed #244 (SHA form → 200) vs #241 (branch-ref form → 404 post-archive). No new test harness needed — a single live curl on the next mod-generated PR is the proof; cost: seconds.
- **Static prose check:** `_mods/pr-merge.md` no longer computes the audit SHA from `{worktree}` and the template/extraction rows read `/blob/{state-sha}/...` with `{state-sha}` sourced from `git -C docs/dev/.spacedock-state rev-parse HEAD`. Cost: a grep + read.
- No Go/fixture tests — the mod is plain-text workflow prose, not tool code; behavior is verified at the live-PR layer where the claim lives.

## Stage Report: ideation

- DONE: Lock the durable audit-link form: an IMMUTABLE state-commit SHA + the path-at-that-SHA.
  Decided state-HEAD-at-PR-open (full SHA via `git -C docs/dev/.spacedock-state rev-parse HEAD`) + active `{slug}/index.md`; verified live #244 SHA-form=200 (post-archive) vs #241 branch-form=404; confirmed 13c5ed75 is a `spacedock-state/dev` commit, pushed to origin.
- DONE: Specify the exact _mods/pr-merge.md edit: audit-link template line + extraction-rule row emit the SHA-based form.
  Body §The fix + AC-1 pin both rows (~line 80 template, ~line 92 extraction) to `/blob/{state-sha}/{state-relative-path}` with state-checkout SHA replacing the code-worktree `rev-parse`; behavioral AC = generated PR audit link HTTP 200.
- DONE: Coordinate with f2 (mods-definition-dir-location); agree a land order; state whether staff review is warranted.
  Agreed: this fix lands FIRST (self-contained prose edit at current location), f2 carries the fixed file when it moves it; FO sequences the two dispatches. Staff review: NOT warranted (small, SHA proven live). Messaged team-lead.

### Summary
Hardened the provisional branch-ref hypothesis into the proven IMMUTABLE-SHA form: the audit link must use the state checkout's HEAD SHA at PR-open time (`git -C docs/dev/.spacedock-state rev-parse HEAD`) + the active entity path, not the code-worktree SHA and not a branch ref. Confirmed live this session that the branch-ref+active-path form 404s after archival while the SHA form stays 200 (#241 vs #244). Decided SHA-at-PR-open over archive-SHA because the entity is necessarily PR-pending at the merge hook. Land order: this fix before f2's mod move; no staff review needed.
