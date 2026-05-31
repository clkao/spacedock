---
id: scz5x5sbr1gy06z36qhh2py5
title: Split-root collaboration (Phase B) — same-repo orphan-branch state, $inline sentinel, workflow-slug state-branch, state init + FO sanity-check + push/pull sync
status: backlog
source: FO/captain (2026-05-31) — Phase-B follow-on to split-root-ergonomics (Phase A); captain-directed orphan-branch model + three journeys
score: "0.30"
started:
completed:
verdict:
worktree:
issue:
---

Phase B of the split-root model (Phase A = boot surfacing + absolute path + commission rule +
single-root test, entity `split-root-ergonomics-and-remote-model`). The captain chose the
**same-repo orphan-branch model first** (separate-repo + `state-remote:` is a deferred possibility,
NOT this entity). This entity makes split-root **multi-host collaborative** with explicit semantics.

Depends on Phase A: the boot `state_backend` + `entity_dir_present` fields are the diagnostic seam the
FO sanity-check (journey 3) hangs off. The Phase-A ideation body (B4/B5 sections, the resume spike)
is the starting design; this entity supersedes its remote-shape recommendation toward the
captain-directed orphan-branch model.

## The model (captain-directed): same-repo orphan branch

State lives on an **orphan branch in the SAME repo** (no second repo, no second remote). State
commits land on the orphan branch (pushed to the main `origin`); the code branch never sees them
(zero churn). The state checkout is a linked worktree of the main repo at the gitignored `state:` path.

## Three journeys to make crisp (captain)

**Journey 1 — Commission a split-root workflow (orphan-branch, same repo).**
Commission writes `state: <path>` (+ a workflow-slug-derived `state-branch`), creates the orphan
branch, and checks it out at the state path as a linked worktree (gitignored on the code branch), so a
freshly-commissioned split workflow is immediately usable.

**Journey 2 — Commission an inline workflow.**
Commission writes the explicit inline sentinel (`state: $inline`) OR omits `state:` (empty → inline,
backward-compat). Entities live beside the README on the same branch.

**Journey 3 — Clone a shape-(1) repo and resume.**
A fresh `git clone` brings the code branch; the state worktree is absent (orphan branch not checked
out). The FO sanity-checks at boot (split-root declared + `entity_dir_present: false` from Phase A) →
HALTS dispatch, reports "state not initialized," and runs/prompts `spacedock state init`, which
fetches the orphan branch and checks it out at the state path. Then work proceeds.

## Scope / design points (ideation hardens — design-first, captain-gated)

**B1 — Explicit `state:` semantics + backward-compat (captain point 2).**
- empty/absent `state:` → **inline** (single-root), backward-compatible default (unchanged).
- `state: $inline` (or a finalized non-colliding sentinel) → **explicit inline**.
- `state: <relative-path>` → **split-root**.
- `resolveRoots`/`splitRootStateCheckout` must treat the inline sentinel as inline (not a path to
  join). Rationale: disambiguate intentional-inline from split-root-with-missing-state, so a missing
  dir on a path-valued `state:` is unambiguously init-needed (journey 3), not maybe-inline.

**B2 — Workflow-specific `state-branch` (captain point 1).**
- Default branch name derives from the workflow slug: `spacedock-state/<workflow-slug>` (matches the
  existing `spacedock-state/dev`). Optional README `state-branch:` override. Multiple workflows in one
  repo → distinct, non-colliding state branches. Remote = the main repo's `origin`.

**B3 — Commission scaffolding (journeys 1 & 2).**
- Split: write `state:` + derived `state-branch`, create the orphan branch, check it out at the state
  path as a linked worktree (gitignored). Inline: write `$inline` (or omit). Freshly-commissioned
  split workflow immediately usable.

**B4 — `spacedock state init` (journey 3 resume).**
- Reads `state:`/`state-branch` from the README. If split-root + state path absent:
  `git fetch origin <state-branch>` + `git worktree add <state-path> <state-branch>`. Idempotent
  (fetch/no-op if present). Documented manual one-liner fallback.

**B5 — FO sanity-check gate (journey 3, captain point 3).**
- FO startup: if split-root declared AND `entity_dir_present: false` → HALT dispatch, report, run/
  prompt `spacedock state init` before proceeding. A FO-contract addition keyed on the Phase-A boot
  field.

**B6 — Multi-writer sync discipline: push / pull --rebase (captain point 3).**
- The state branch is shared via `origin`. FO/ensigns MUST push state commits after committing, and
  `git pull --rebase` when relying on possibly-stale state / integrating concurrent writers. A new
  FO + ensign contract addition extending the path-scoped state-commit rule. Ideation designs the
  minimal sync points (boot? pre-dispatch? on push-rejection?) and how it composes with the existing
  path-scoped-commit concurrency rule.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — `state:` modes are explicit.** `$inline` parses as inline; empty `state:` defaults to inline
(backward-compat); a relative path → split-root. Verified by a status-parser test over all three.

**AC-2 — `state-branch` defaults to a workflow-slug-specific name** (`spacedock-state/<slug>`),
overridable. Verified by a test; reproducible on this workflow.

**AC-3 — Commission scaffolds both journeys.** A split commission yields a usable orphan-branch state
worktree; an inline commission yields beside-the-README entities. Verified by commission e2e/tests.

**AC-4 — `spacedock state init` resumes a cloned shape-(1) repo.** Fresh main clone → `state init` →
`spacedock status` renders the entities. Verified by an e2e mirroring the Phase-A resume spike.

**AC-5 — FO sanity-check blocks on uninitialized state and resumes after init.** Verified by the FO
contract + a behavioral check on the boot-field gate (split-root + entity_dir_present:false → halt).

**AC-6 — Push/pull--rebase sync discipline is contractual and exercised.** A 2-writer sync e2e: host A
commits+pushes a state change, host B `pull --rebase` sees it; the FO/ensign contracts carry the rule.

## Out of scope
- Separate-repo + `state-remote:` shape (deferred possibility; not this entity).
- Mods / PR-merge behavior; a full lock-based concurrency model beyond path-scoped-commit + pull--rebase.

## Notes — design-first, captain-gated
Genuine architecture (orphan-branch worktree mechanics, the `$inline` sentinel token, the sync-point
design, the FO halt-gate). The FO will bring the ideation design to the captain rather than
auto-approving, and ideation should propose its own internal sequencing (e.g. B1/B2 parser semantics
first, then B3/B4 commission+init, then B5/B6 FO/ensign contract). Reference: the Phase-A entity's
ideation design + resume spike, docs/specs/state-behavior-extension.md, docs/roadmap/bootstrap-roadmap.md.
