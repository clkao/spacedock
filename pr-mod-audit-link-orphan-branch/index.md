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

The `pr-merge` mod builds the PR body's audit link pointing at the **code worktree's commit**, but the entity file it links to lives on the **orphan state-branch** (`spacedock-state/dev`) — a separate checkout the code commit never contains. So the link 404s. Fix it to reference the orphan branch.

## The bug (in `.spacedock-state/_mods/pr-merge.md`)
The audit-link spec (the body table, ~lines 80/92) is:
```
[{short-id}](/{owner}/{repo}/blob/{short-sha}/{state-relative-path})
```
where `{short-sha}` = `git -C {worktree} rev-parse --short HEAD` — the **CODE** worktree's SHA. But:
- The entity file (`{slug}/index.md`) lives in the `.spacedock-state` checkout, whose content is the **orphan branch `spacedock-state/dev`** — NOT in the code repo at the code commit (separate checkout; the code commit has zero `.spacedock-state` files).
- So `/blob/{code-sha}/{path}` resolves to nothing → 404.

## The fix
Build the audit link against the **orphan branch ref**, not the code SHA:
```
[{short-id}](/{owner}/{repo}/blob/spacedock-state/dev/{state-relative-path})
```
- `{state-relative-path}` is the entity file's path **relative to the `.spacedock-state` root** (= the orphan-branch root) — e.g. `_archive/{slug}/index.md` once archived, or `{slug}/index.md` while active. (Pick the location the entity will be at when the link is followed — archived entities move to `_archive/`.)
- This requires the state to be **pushed to `spacedock-state/dev`** (now done) so the link resolves.
- Also correct PR #240's body, whose audit line is the placeholder `audit: 38 · spacedock-dev/spacedock` (no working link), if #240 is still open / for the record.

## Acceptance criteria (provisional — ideation hardens)
**AC-1 — The mod's audit link targets the orphan branch and resolves.** End state: `_mods/pr-merge.md`'s audit-link template uses `/blob/spacedock-state/dev/{state-relative-path}` (orphan-branch ref), not the code `{short-sha}`; the path is the entity file's `.spacedock-state`-relative location. Verified by: a PR opened via the mod has an audit link that loads the entity file on GitHub (HTTP 200), pointing at the orphan branch.
**AC-2 — The archived-vs-active path is correct.** End state: the link points at where the entity file actually lives when followed (active `{slug}/index.md`; archived `_archive/{slug}/index.md`). Verified by: follow the link for a PR-pending entity (active path) and re-check after archive (or document that the mod uses the active path at PR-open time, which is correct for a PR-pending entity).

## Notes
- Tiny `_mods/pr-merge.md` prose edit → dispatched worker (`_mods/` off-limits to direct FO edits).
- Coupling: `mods-definition-dir-location` moves the mod to `docs/dev/_mods/` (a different concern); land either order, but if both, the mover should carry this fix. `pr-merge-mod` is the parent (this is a follow-up defect on the mod it shipped).
- Off the 0.19.2 critical path; the mod works (opens PRs, tracks `pr:`) — only the audit *link* is broken.
