---
id: 0mg8zaseaffrjcs8n8qtrfcj
title: pr-merge mod — open a code-branch PR at the merge boundary, composing with state-root
status: ideation
source: FO/captain (2026-06-01) — get pr-mod initialized + make it work well with state-root
score: "0.30"
started: 2026-06-01T02:30:24Z
completed:
verdict:
worktree:
issue:
---

Initialize the workflow's first **merge-lifecycle mod**: a `_mods/pr-merge.md` with a `## Hook: merge`
that, when an entity reaches its terminal stage, **opens a PR for the entity's CODE worktree branch
to `next`** (sets `pr:` on the entity, blocks via `mod-block` until merged) instead of the FO doing a
local merge. This graduates past the bootstrap README's "no PR merge flow / mods" statement — it is
the captain-requested pr-mod.

**Compose with state-root (split-root) — the load-bearing requirement.** This workflow is split-root:
CODE lives in the main repo (worktree branches → PR to the code origin `next`), the entity STATE
lives in the separate `docs/dev/.spacedock-state` checkout (now remoted: `origin` = the main repo,
branch `spacedock-state/dev`, applied 2026-06-01). The mod must keep the two origins clean:
- the **code PR** carries ONLY the code-branch range (`BASE..HEAD` of the entity's worktree branch),
  never any `.spacedock-state` file (structurally true — they are separate repos);
- the **entity state** (frontmatter, stage reports, the `pr:` field) is committed path-scoped to
  `.spacedock-state` and pushed to its remote (`spacedock-state/dev`), so PR-pending state survives
  session resume and is visible on a 2nd host.

**Grounded in the proven manual flow.** This session landed live-e2e (#237), k6 (#238), Phase A
(#239) onto `next` exactly this way by hand: fresh branch off `origin/next`, the entity's code
commits, `gh pr create --base next`, merge, while the entity state stayed in `.spacedock-state`. The
friction the manual flow hit (entity branches deleted at cleanup before PRing → had to cherry-pick
from `main`'s history) is the thing this mod fixes: **PR the code branch at the merge boundary, before
cleanup, then archive.** The mod automates that flow.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — A registered `## Hook: merge` mod opens a code-branch PR at terminalization and records it.**
Verified by: when an entity reaches its terminal stage with merge hooks registered, the FO's
Merge-and-Cleanup flow invokes the mod; the mod opens a PR for the entity's code worktree branch to
`next` (or the configured base) and sets `pr:` on the entity; the existing `mod-block`/`pr`-mirroring
machinery (status --set/--archive terminal guards) blocks terminalization until the PR merges, then
clears. Behavioral pilot on a real entity, not a doc assertion.

**AC-2 — The code PR and the entity state stay on their separate origins (state-root clean).**
Verified by: the opened PR's diff contains only code-repo files (no `.spacedock-state` path appears —
structurally guaranteed by the separate repos); the entity's frontmatter/`pr:`/reports are committed
path-scoped to `.spacedock-state` and pushed to `spacedock-state/dev`; a `status --boot` on a 2nd
checkout that fetched the state shows the PR-pending entity.

**AC-3 — The mod degrades to a documented fallback when no PR host is available.**
Verified by: with no `gh`/PR host, the mod falls back to the FO's existing local `--no-ff` merge of
the code branch (the current behavior), recording that path; same merge boundary, different landing.

## Out of scope
- The full Phase-B `spacedock state init` resume subcommand + README `state-remote:` declaration
  (its own entity; this mod USES the already-applied state remote, does not build the resume command).
- roborev / review hooks (ng's domain).
- Any change to the code-PR review gates on `next` (separate).

## Notes — keep it focused
This is ONE thing: a merge-hook mod that PRs the code branch, state-root-aware. Reference: the FO
`## Merge and Cleanup` + `## Mod-Block Enforcement` flow (the `mod-block`/`pr` machinery already
exists), the `## Mod Hook Convention` (`## Hook: merge`, alphabetical mod order), and the proven
manual #237/#238/#239 landings. Mods live in `docs/dev/_mods/` (none exist yet — this creates the
first). Design-first for the captain at the ideation gate (a new lifecycle mod is a workflow-shape
change).
