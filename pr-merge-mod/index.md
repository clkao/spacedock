---
id: 0mg8zaseaffrjcs8n8qtrfcj
title: pr-merge mod — open a code-branch PR at the merge boundary, composing with state-root
status: implementation
source: FO/captain (2026-06-01) — get pr-mod initialized + make it work well with state-root
score: "0.30"
started: 2026-06-01T02:30:24Z
completed:
verdict:
worktree: .worktrees/spacedock-ensign-pr-merge-mod
issue:
mod-block: merge:pr-merge
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

## Design

### The mod file and where it lives (verified, not assumed)

`_mods/pr-merge.md` lives in the **state checkout** at
`docs/dev/.spacedock-state/_mods/pr-merge.md` — **not** `docs/dev/_mods/`. This is the single
load-bearing correction the ideation spike surfaced. The seed text (and the dispatch) said "mods
live in `docs/dev/_mods/`"; that is wrong for a split-root workflow. The guard reads
`scanMods(entityDir)` (`internal/status/mutate.go:282`, `internal/status/handlers.go:129`), and for
this split-root workflow `entityDir` resolves to `.spacedock-state` (README `state: .spacedock-state`;
boot's `entityDir != definitionDir` split, `internal/status/boot.go:177-210`). A mod placed in
`docs/dev/_mods/` does **not** register; one in `.spacedock-state/_mods/` does. Proven by probe:

- mod at `.spacedock-state/_mods/zzz-probe.md` → `MODS\nmerge: zzz-probe`, and `--set status=done`
  and `--archive` both refuse with the named-hook error.
- same mod at `docs/dev/_mods/` → `MODS: none`, guard does not fire.

Filename `pr-merge.md` → mod name `pr-merge`; the FO runs registered merge mods in alphabetical
order by filename, so this is the first/only one.

### `## Hook: merge` body — what the FO does when it runs the hook

The mod is FO-executed prose (the FO runs hooks; it does not write them — FO Write Scope). At a
terminal entity the FO's Merge-and-Cleanup flow (steps 1-9) invokes registered merge hooks before any
local merge/archive. The `## Hook: merge` section instructs the FO to, for the entity's CODE worktree
branch `{branch}` (the branch named in `worktree:`):

1. Ensure the code branch is on the code remote: `git -C {worktree} push -u origin {branch}`
   (origin = `spacedock-dev/spacedock`, the code repo).
2. Open the PR: `gh pr create --base next --head {branch} --title "{entity title}" --body "..."`
   from the worktree. Base is `next` (`origin/next` exists). Capture the PR number.
3. Record it on the entity: `spacedock status --workflow-dir docs/dev --set {slug} pr=#{N}`
   (writes `pr:` into the state-checkout entity frontmatter).
4. The hook is now **blocking** (FO Merge-and-Cleanup step 3a: a `pr` field is set). The FO leaves
   `mod-block: merge:pr-merge` set, reports PR-pending, and does NOT local-merge or archive.

The set-then-invoke order from the FO contract still holds: the FO sets
`mod-block=merge:pr-merge` (step 1) *before* invoking, so resume knows which mod is pending. The
mechanism-level invariant (`handlers.go:129`, `mutate.go:196`) independently refuses terminal
`--set`/`--archive` while a merge hook is registered and both `pr` and `mod-block` are empty — it
catches a forgotten `mod-block`.

### Unblock at PR-merge time

When the PR lands on `next` (boot's `PR_STATE` scan shows `MERGED`, `boot.go:95-114`), the FO:
- clears the block in its own standalone `--set`: `status --set {slug} mod-block=` (the guard refuses
  combining `mod-block=` with terminal fields in one call, `handlers.go` — two commits required),
- then terminalizes: `--set {slug} completed verdict={v} worktree=`, `--archive {slug}`,
- removes the worktree and deletes the **local** branch; the **remote** branch was already cleaned by
  the PR merge (FO contract step 9: do not delete remote while PR pending).

This is exactly the friction the proven manual flow (#237/#238/#239) hit and fixes: PR the code
branch **at the merge boundary, before cleanup deletes it**, rather than cherry-picking from `main`'s
history afterward.

## Acceptance criteria

**AC-1 — A registered `## Hook: merge` mod opens a code-branch PR at terminalization and records it.**
End state: with `pr-merge.md` registered in `.spacedock-state/_mods/`, a terminal entity cannot be
`--set` to its terminal stage or `--archive`d until `pr:` (or `mod-block:`) is set; when the FO runs
the hook, the entity carries `pr: #{N}` pointing at a PR whose head is the entity's code worktree
branch and whose base is `next`.
Verified by: behavioral pilot on a real entity — register the mod, drive an entity to terminal, FO
runs the hook, observe `gh pr view` returns a PR with `baseRefName=next` and `headRefName={branch}`,
and the entity frontmatter shows `pr: #{N}`. Mechanism already proven this session: guard refusal
fires with named hook (`--set status=done` and `--archive` both blocked), unblocks once `pr:` set.

**AC-2 — The code PR and the entity state stay on their separate landings (state-root clean).**
End state: the PR's diff contains zero `.spacedock-state` paths (structural — code worktree and state
checkout are separate clones), and the PR-pending entity state lives in `.spacedock-state` pushed to
`spacedock-state/dev`, discoverable on a 2nd host.
Verified by: `gh pr diff {N} --name-only` lists only code-repo files; the entity `pr:`/`mod-block:`
write is committed path-scoped to the state checkout
(`git -C docs/dev/.spacedock-state add -- {slug}/index.md && git ... commit -- {slug}/index.md`) and
pushed `spacedock-state/dev`; a `status --boot` on a 2nd checkout that fetched `spacedock-state/dev`
shows the entity under `PR_STATE` with the pending PR. Note: both clones share the same GitHub repo
`spacedock-dev/spacedock` — "separate origins" means **separate branches** (code → `next`,
state → `spacedock-state/dev`), enforced by them being separate working clones, so a code commit
cannot carry a state file and vice versa.

**AC-3 — The mod degrades to a documented fallback when no PR host is available.**
End state: with no `gh` on PATH (or `gh pr create` fails), the hook's prose directs the FO to the
existing default local merge (FO Merge-and-Cleanup step 6: `--no-ff` merge of the code branch from
the worktree), recording the local-merge landing in the entity report; the entity still terminalizes
at the same merge boundary, just without a PR.
Verified by: with `gh` absent, the hook does not set `pr:`; the FO, finding no PR was opened, sets
`mod-block=` is unnecessary (no block was created — hook completed without blocking, step 5/6) and
performs the local `--no-ff` merge; entity reaches terminal. Tested via the fallback branch of the
pilot (PATH without `gh`).

## Test plan

- **Mechanism (proven, minutes):** mod registration + guard refusal — done in this ideation spike
  (probe above). A Go-level regression already exists in `internal/status/archive_guard_test.go` /
  `native_guard_test.go` covering the merge-hook guard; the mod just supplies the `## Hook: merge`
  file those guards key off. No new Go test needed for the guard itself.
- **Live workflow pilot (the real claim, AC-1/AC-2):** FO-run, on a real small entity — register
  `pr-merge.md`, drive to terminal, FO opens the PR, assert `gh pr view`/`gh pr diff --name-only` and
  the path-scoped state commit + push. This is FO-RUN: it creates a real PR against `next` and pushes
  to `spacedock-state/dev` (flag to captain before running — outward-facing).
- **Fallback (AC-3):** run the hook with `gh` removed from PATH; assert no `pr:` set and a local
  `--no-ff` merge lands. Fixture/CLI-level for the guard side; the merge side is FO behavior.
- The mod file is prose, not code, so its primary "test" is the live FO pilot that exercises the
  prose against the real guard machinery — a static prose assertion would not prove the hook composes
  with Merge-and-Cleanup.

### FO-RUN / outward-facing steps to flag at the implementation/validation gate
- `gh pr create --base next` opens a real PR against `spacedock-dev/spacedock` — publishes the branch.
- `git push` of the code branch to `origin` and the state to `spacedock-state/dev` — both outward.
The implementation stage writes the mod file (a dispatched worker, since `_mods/` is off-limits to
direct FO edits); the validation pilot that actually opens a PR is FO-run and should be captain-gated.

## Out of scope
- The full Phase-B `spacedock state init` resume subcommand + README `state-remote:` declaration
  (its own entity; this mod USES the already-applied state remote, does not build the resume command).
- roborev / review hooks (ng's domain).
- Any change to the code-PR review gates on `next` (separate).
- Any change to the guard mechanism in `internal/status/` — the `scanMods`/`mod-block`/`pr` machinery
  already exists and is sufficient; this mod only supplies the `## Hook: merge` prose file.

## Notes — keep it focused
This is ONE thing: a merge-hook mod that PRs the code branch, state-root-aware. Reference: the FO
`## Merge and Cleanup` (steps 1-9) + `## Mod-Block Enforcement` flow (the `mod-block`/`pr` machinery
already exists), the `## Mod Hook Convention` (`## Hook: merge`, alphabetical mod order), and the
proven manual #237/#238/#239 landings. The mod lives in **`docs/dev/.spacedock-state/_mods/`**
(verified — the guard's `scanMods(entityDir)` reads the state checkout in split-root, NOT
`docs/dev/_mods/`). Design-first for the captain at the ideation gate (a new lifecycle mod is a
workflow-shape change).

## Stage Report: ideation

- DONE: Design the `_mods/pr-merge.md` merge-lifecycle mod concretely (`## Hook: merge` section + terminalization behavior).
  `## Design` section specifies the FO-run hook: push code branch → `gh pr create --base next` → `status --set pr=#N` → block; reads FO `## Merge and Cleanup` (steps 1-9), `## Mod-Block Enforcement`, `## Mod Hook Convention`.
- DONE: STATE-ROOT composition — name exactly which git/gh commands run against which origin, friction confirmed.
  AC-2: code branch → `origin/next` (worktree clone); state `pr:`/`mod-block:` → path-scoped commit to `.spacedock-state` pushed `spacedock-state/dev`. Both clones share repo `spacedock-dev/spacedock`; "separate origins" = separate branches, enforced by separate working clones. Friction (branch deleted at cleanup before PRing → #237/#238/#239 cherry-pick) fixed by PRing at the merge boundary before cleanup.
- DONE: Spike the riskiest unknown + harden ACs; flag FO-RUN bits; define no-PR-host fallback.
  Probe proved the load-bearing unknown: `scanMods(entityDir)` reads `.spacedock-state/_mods/`, NOT `docs/dev/_mods/` — mod in state checkout registers (`merge: zzz-probe`) and the `--set status=done` + `--archive` guards refuse with the named hook; same mod in `docs/dev/_mods/` → `MODS: none`. ACs rewritten as end-states with verification. Fallback (AC-3): no `gh` → FO local `--no-ff` merge (Merge-and-Cleanup step 6). FO-RUN steps flagged (real PR against `next`, push to `spacedock-state/dev`).

### Summary
Hardened the pr-merge mod ideation into a concrete FO-run `## Hook: merge` design that composes with the already-applied state-root. The decisive spike correction: the seed/dispatch said mods live in `docs/dev/_mods/`, but for this split-root workflow the guard's `scanMods(entityDir)` reads the resolved state checkout, so the mod MUST live at `docs/dev/.spacedock-state/_mods/pr-merge.md` — proven by probe (registers + fires the terminal guard there, silent in `docs/dev/_mods/`). Mechanism is fully proven this session (guard refusal with named hook, unblock on `pr:`); the remaining real work is the live FO-run PR pilot, which is outward-facing and captain-gated. Scope held to one merge-hook mod; Phase-B state-init and roborev explicitly excluded.

## Stage Report: implementation

- DONE: `.spacedock-state/_mods/pr-merge.md` written per the entity's `## Design`: frontmatter `name: pr-merge`; `## Hook: startup` + `## Hook: idle` (scan non-empty-`pr` non-terminal entities, `gh pr view` each, advance MERGED via two-step `mod-block=` clear then terminalize, report CLOSED) and `## Hook: merge` (push code worktree branch to origin, `gh pr create --base next --head {branch}`, `status --set {slug} pr=#{N}`, leave mod-block, report PR-pending). Reconciled from `origin/next:mods/pr-merge.md`: base `main`->`next` everywhere; dropped `git rebase main`/`git push origin main` (FO rebases onto origin/next before the hook in this split-root flow); state-root-aware (all `pr:`/`mod-block:` writes go through `spacedock status --set --workflow-dir docs/dev`; hook MUST NOT touch `.spacedock-state` from the code worktree). Commit 0653a92.
- DONE: The mod REGISTERS and the terminal guard fires. `spacedock status --workflow-dir docs/dev --boot` shows `MODS\nmerge: pr-merge` (also `startup`/`idle`) — proves `scanMods` reads the `.spacedock-state` checkout. Behavioral probe on a throwaway `zzz-guard-probe` entity: terminal `--set status=done` refused ("merge hook(s) [pr-merge] that have not run") and `--archive` refused likewise (both exit 1); after `--set pr=#999` the terminal `--set status=done` succeeded (exit 0). Probe was never committed (no git footprint) and removed.
- DONE: AC-3 fallback prose present. The `## Hook: merge` `### Fallback: no PR host available` section directs the FO, when `gh` is absent / `gh pr create` fails / branch push fails, to NOT set `pr:` and fall back to the local `--no-ff` merge (Merge-and-Cleanup step 6); since no `pr:` was set the hook completed without blocking, FO clears its `mod-block` in a standalone `--set` after the local merge lands, entity still terminalizes. Separate captain-decline branch (PR host present, captain says no) routes to captain rather than auto-merging. Mod committed path-scoped to the state checkout (commit 0653a92); not pushed to spacedock-state/dev (FO coordinates state-remote pushes).

### Summary
Wrote `.spacedock-state/_mods/pr-merge.md`, the workflow's first merge-lifecycle mod, reconciled from the ported `origin/next:mods/pr-merge.md` reference for this split-root, base-`next` flow: base `main`->`next`, the `git rebase main`/`git push origin main` steps dropped (the FO rebases the code branch onto origin/next before invoking the hook), and every entity-state write (`pr:`, `mod-block:`) routed through `spacedock status --set --workflow-dir docs/dev` so the code worktree never touches `.spacedock-state`. Confirmed registration and the terminal guard behaviorally (MODS shows `merge: pr-merge`; both `--set` and `--archive` refuse while `pr`+`mod-block` empty, unblock once `pr:` set). Did NOT run the live PR pilot — that is the FO-run, captain-gated validation step (opening 38's PR). Mod committed path-scoped; not pushed to the state remote.
