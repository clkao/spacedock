---
id: scz5x5sbr1gy06z36qhh2py5
title: Split-root collaboration (Phase B) — same-repo orphan-branch state, $inline sentinel, workflow-slug state-branch, state init + FO sanity-check + push/pull sync
status: ideation
source: FO/captain (2026-05-31) — Phase-B follow-on to split-root-ergonomics (Phase A); captain-directed orphan-branch model + three journeys
score: "0.30"
started: 2026-05-31T21:15:12Z
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

---

# Ideation design (2026-05-31) — decision-ready, captain-gated

This design is decision-ready: each part names a recommendation, gives tradeoffs, and a concrete
reproducible proof at the right altitude. The model is FIXED by the captain (same-repo
**orphan-branch** state on the main `origin`; separate-repo + `state-remote:` is explicitly DEFERRED).
The three riskiest unknowns were SPIKED end-to-end before finalizing — results in §Spike. This entity
supersedes Phase A's "separate-repo + declared remote (shape a)" recommendation toward shape (c)
orphan-branch, which the captain selected.

## Grounded current state (verified 2026-05-31)

- **`state:` is read at THREE sites, all treating non-empty as a path to join.** (1) `resolveRoots`
  (status/roots.go:55-67) diverges `entityDir = definitionDir/<state>`; (2) `splitRootStateCheckout`
  (dispatch/helpers.go:127-135) returns `workflowDir/<state>` for the state-commit guidance; (3)
  `discoverWorkflows` (status/handlers.go:378) prunes `definitionDir/<state>` from the workflow scan.
  All three reject absolute / `..`-escaping values. **A `$inline` sentinel must be recognized at all
  three** or one site joins a literal `…/$inline` path — the strongest argument for a single shared
  helper (B1).
- **Phase A already ships the B5 seam.** The Phase-A worktree (`internal/status/boot.go`,
  `json_commands.go`) emits boot keys `state_backend` (`split-root`|`single-root`),
  `definition_dir`, `entity_dir`, `entity_dir_present` (string `"true"`/`"false"`), appended after
  `team_state` (`bootJSONKeys`, json_boot_test.go:20). `state_backend` is derived from
  `entityDir != definitionDir`; `entity_dir_present` is an `os.Stat(entityDir).IsDir()`. **B5's
  halt-gate keys directly off `state_backend == "split-root" && entity_dir_present == "false"`** with
  no new boot field. Design B against these planned fields; do not duplicate them.
- **The existing state checkout is structurally shape (a), NOT (c).** `docs/dev/.spacedock-state`
  TODAY is an *independent* repo (own `.git` with `hooks/`/`info/`, branch `spacedock-state/dev`, NO
  remote). The orphan-branch model wants it to be a *linked worktree of the main repo* on an orphan
  branch pushed to main `origin`. That is a structural shift; migrating THIS workflow's existing
  checkout is a one-time operational step (see §Migration), not part of the shipped feature surface.

## B1 — `$inline` sentinel + backward-compat, via ONE shared resolver

**Recommendation: a single shared classifier `classifyState(stateValue) → (mode, relPath, err)`** in
`internal/status` (the package `dispatch` already imports), consumed by all three read sites. Three
modes:
- `""` (empty/absent) → **inline** (single-root), unchanged backward-compat default.
- `"$inline"` (exact, after `TrimSpace`) → **explicit inline** — treated identically to empty for
  root resolution (NOT joined as a path; `entityDir == definitionDir`).
- any other value → **split-root**: validate (reject absolute, reject `..`-escape as today), return
  the cleaned relative path to join.

**Why a sentinel at all (not just empty).** Disambiguates *intentional inline* from
*split-root-with-missing-state*. Once shape-(1) workflows always carry a path-valued `state:`, a
missing dir on a path-valued `state:` is **unambiguously init-needed** (journey 3), never
maybe-inline. `$inline` lets commission stamp inline workflows explicitly so the absence of `state:`
keeps meaning legacy-default rather than overloading "needs init."

**Why `$inline` is non-colliding.** `state:` is a relative path; `$` is not a legal leading char for
the v0 child-checkout paths (which are `.spacedock-state`-style names), and the existing validator
already rejects absolute and `..` paths, so `$inline` cannot collide with a real checkout name. The
`$` prefix is a recognizable sentinel convention. (Open micro-decision for the captain: `$inline`
vs `@inline` vs a bareword `inline`. RECOMMEND `$inline` per the captain's suggestion — the `$`
sigil reads as "not a path" at a glance and is the lowest collision risk.)

**Parser behavior (concrete).** `classifyState` runs after `strings.TrimSpace`. `resolveRoots`,
`splitRootStateCheckout`, and `discoverWorkflows`'s prune all call it; on `inline`/`explicit-inline`
they keep `entityDir == definitionDir` / return `""` / skip the prune respectively. The existing
absolute/`..` rejection moves INTO the classifier so the three sites share one validation.

**Proof altitude (AC-1).** Go unit tests on the status parser over all three modes: `$inline` →
inline (`entityDir == definitionDir`, `splitRootStateCheckout == ""`), empty → inline (unchanged),
a relative path → split-root (`entityDir == definitionDir/<path>`). Plus a negative: `$inline` is NOT
joined as a literal path at any of the three sites (no `…/$inline` on disk-path output). Correct
altitude: parser/command behavior is a Go unit claim.

## B2 — Workflow-slug `state-branch` default

**Recommendation: default `state-branch = spacedock-state/<workflow-slug>`**, overridable by an
optional README `state-branch:` field. The slug is the workflow's identifying token (matches the
existing `spacedock-state/dev` for the `docs/dev` workflow). Multiple workflows in one repo → distinct
non-colliding orphan branches on the same main `origin`. The remote is ALWAYS the main repo's
`origin` (no `state-remote:` in this entity — deferred).

**Slug derivation.** RECOMMEND deriving the slug from the workflow dir's basename
(`filepath.Base(workflowDir)` → `dev` for `docs/dev`), which reproduces the existing
`spacedock-state/dev` exactly and needs no new frontmatter. (Alternative: a README `slug:`/`id`
field; rejected as heavier — basename already yields the shipped name.) An explicit README
`state-branch:` always wins when present.

**Proof altitude (AC-2).** A Go unit test on the branch-name derivation: basename `dev` →
`spacedock-state/dev`; an overriding `state-branch:` field is honored verbatim. Reproducible on THIS
workflow (its `docs/dev/README.md` already implies `spacedock-state/dev`).

## B3 — Commission scaffolding (journeys 1 & 2) — EXACT git mechanics

**Journey 1 (split, orphan-branch) — the spiked mechanic (§Spike (a), PASS).** Commission, for a
split workflow:
1. Writes README frontmatter `state: <path>` (e.g. `.spacedock-state`) + derived
   `state-branch: spacedock-state/<slug>` (omit if default; an explicit override may be written).
2. Adds `<path>/` to the code branch's `.gitignore` (so state commits never churn the code branch —
   R1).
3. Creates the orphan branch with a seed commit WITHOUT disturbing the code working tree. The spiked
   technique: birth the orphan in a temporary detached worktree, `git checkout --orphan <branch>`,
   then **clear the inherited index/tree** (`git rm -rf --cached .` + `rm -rf` the files — an orphan
   checkout inherits the source branch's tree, a real mechanic the spike surfaced), seed an initial
   state commit, remove the temp worktree.
4. Checks the orphan branch out at the gitignored `state:` path as a LINKED worktree of the main repo:
   `git worktree add <state-path> <state-branch>`.
5. (Optional, if `origin` exists) pushes the orphan branch: `git -C <state-path> push origin <branch>`.

Result: a freshly-commissioned split workflow is immediately usable — `spacedock status` renders, the
code branch is clean (spike confirmed `git status --porcelain` empty), state lives on its own branch.

**Journey 2 (inline) — trivial.** Commission writes `state: $inline` (explicit) OR omits `state:`
(empty → inline). Entities live beside the README on the same branch. No orphan branch, no worktree,
no gitignore entry. RECOMMEND commission write `$inline` explicitly for new inline workflows so the
backend choice is self-documenting and journey-3's "missing path-valued state" diagnostic stays
unambiguous.

**Proof altitude (AC-3).** A commission e2e: commissioning a split workflow yields an orphan-branch
state worktree that `spacedock status` renders + a clean code branch; commissioning inline yields
beside-the-README entities. The git-mechanics half (orphan-as-linked-worktree, gitignore, push) is
already PROVEN by §Spike (a); the e2e hardens the known-working path. Live workflow / integration
altitude (runtime claim). The skill-text changes to commission's SKILL.md (the journey-1 mechanics +
journey-2 prose) also carry a static skill-text test.

## B4 — `spacedock state init` (journey 3 resume) — the spiked mechanic

**Journey 3 (clone + resume) — the spiked mechanic (§Spike (b), PASS).** A fresh `git clone` of the
main repo brings the code branch; the state worktree is ABSENT (orphan branch not checked out, path
gitignored). `spacedock state init`:
1. Reads `state:` + `state-branch` from `{workflow-dir}/README.md` (default branch derived per B2).
2. If `state:` is `$inline`/empty → no-op (inline workflow has nothing to init; print a one-liner).
3. If split-root AND the `state:` path is **absent** on disk: `git fetch origin <state-branch>` then
   `git worktree add <state-path> <state-branch>`.
4. If the path is already present → idempotent no-op. **Guard explicitly**: the spike showed a second
   `git worktree add` FATALS with "already exists" (exit non-zero) — `state init` MUST check the path
   first and skip, not rely on git's no-op. RECOMMEND: present → `git fetch origin <branch>` +
   `git -C <state-path> pull --rebase` (refresh) then report, never re-`worktree add`.
5. Documented manual one-liner fallback (works today, IS the spike):
   `git fetch origin <branch> && git worktree add docs/dev/.spacedock-state <branch>`.

**Proof altitude (AC-4).** An e2e mirroring the Phase-A resume spike: build a main bare remote with a
commissioned split workflow + pushed orphan branch, fresh-clone main (state absent), run `state init`
(or the documented one-liner), assert `spacedock status` renders the seeded entity. Live
workflow / integration altitude. Mechanism PROVEN by §Spike (b).

## B5 — FO sanity-check halt-gate (journey 3, keyed on the Phase-A boot field)

**Recommendation: an FO-CONTRACT addition (not new code) keyed on the Phase-A boot fields.** At FO
startup the FO already reads `status --boot --json`. The gate:

> If `state_backend == "split-root"` AND `entity_dir_present == "false"`: the state checkout is not
> initialized. HALT dispatch. Report "state not initialized — run `spacedock state init` (or the
> manual `git fetch origin <branch> && git worktree add <state-path> <branch>`)." After init,
> re-read boot; proceed only when `entity_dir_present == "true"`.

**Why a contract, not a code gate.** The FO is a skill (first-officer SKILL.md), and the boot fields
Phase A ships make the condition a pure read — no new Go surface. This keeps B5 cheap and keeps the
halt-decision where the FO's other startup decisions live. (Alternative: a `spacedock doctor`/exit-code
gate in Go; heavier, and the FO must still interpret it — deferred unless the captain wants a
machine-checkable exit code.)

**Why this is safe.** Phase A's bonus diagnostic noted a 2nd host with absent state renders an EMPTY
table at exit 0 + `--validate` says VALID — a silent failure. The gate converts that silent-empty into
an explicit halt, so the FO never dispatches against a phantom-empty workflow.

**Proof altitude (AC-5).** The FO-contract text addition (first-officer SKILL.md) carries a static
skill-text test that the gate prose is present (the condition + the halt + the init pointer). The
behavioral half is the boot field itself, already pinned by Phase A's `json_boot_test.go`
(`entity_dir_present` for an absent vs present checkout). RECOMMEND a thin live-workflow smoke
(fresh clone → boot shows `entity_dir_present:false` → init → boot shows `true`) folded into B4's e2e
rather than a separate test — same fixture.

## B6 — Multi-writer sync: push / `pull --rebase` discipline (the spiked mechanic)

**Recommendation: extend the existing path-scoped-commit rule with MINIMAL sync points** (§Spike (c),
PASS). The state orphan branch is shared via `origin`; concurrent writers (FO + ensigns across hosts,
or even same-host parallel stages) must converge without clobbering.

**The minimal sync points (NOT every operation):**
- **After a state commit → push.** Whoever commits an entity/report pushes the orphan branch so peers
  can see it: `git -C <state-path> push origin <state-branch>`.
- **On push rejection (non-fast-forward) → `pull --rebase` then re-push.** The spike showed a
  concurrent writer's push is rejected; `git pull --rebase origin <state-branch>` replays the local
  path-scoped commit atop the peer's, yielding linear history (distinct entity files → no conflict),
  then the re-push succeeds. This is the load-bearing sync point.
- **At FO boot (before first dispatch) → `pull --rebase`** to integrate peers' state before computing
  dispatchability. One pull at boot, not per-read.
- **NOT pre-every-dispatch.** RECOMMEND against a pull before every dispatch (chatty, and the FO's
  boot pull + commit-time push already bound staleness to one round-trip). Pull-on-boot +
  push-after-commit + pull-on-rejection is the minimal sufficient set.

**Composition with the path-scoped-commit rule.** The existing rule (ensign core: stage + commit ONLY
your own entity path, never bare `git add -A`) is what makes `pull --rebase` clean: because each
writer commits exactly one entity file, concurrent writers touch disjoint paths → rebase replays with
no conflict (spike confirmed). B6 ADDS the push + pull-rebase steps; it does not change the commit
rule. Contract homes: the FO push/pull-on-boot in first-officer SKILL.md; the ensign
push-after-commit + pull-on-rejection in the ensign shared core (alongside the existing path-scoped
commit rule).

**Conflict handling (honest scope).** Two writers editing the SAME entity's frontmatter concurrently
CAN conflict on rebase. That is the path-scoped rule's known boundary; a full lock model is explicitly
out of scope (per the entity). The contract should say: on a rebase conflict the writer stops and
surfaces it (does not force-push), matching the existing "escalate rather than guess" discipline.

**Proof altitude (AC-6).** A 2-writer sync e2e: host A commits+pushes a state change, host B
`pull --rebase` sees it; the reverse direction with a push-rejection→pull-rebase→re-push round-trip.
The FO/ensign contract additions carry static skill-text tests for the push/pull-rebase prose. Live
git integration altitude (real clones, no mocks — the e2e rule). Mechanism PROVEN by §Spike (c).

## Spike — riskiest unknowns, all PASS (run 2026-05-31, throwaway sandbox)

Built a bare main `origin`, host A clone, host B clone — observed before designing:
- **(a) orphan-branch as linked worktree at gitignored path.** PASS. Birthed orphan in a temp detached
  worktree (clearing the inherited index/tree), checked it out at the gitignored `.spacedock-state`
  path via `git worktree add`, pushed to main `origin`. Code branch `git status --porcelain` was
  EMPTY (zero churn, R1 satisfied). Surfaced mechanic: orphan checkout inherits the source tree → must
  `git rm -rf --cached . && rm -rf` before seeding (folded into B3 step 3).
- **(b) `state init` from a fresh main clone.** PASS. Fresh clone → state path absent
  (`entity_dir_present:false`) → `git fetch origin <branch>` + `git worktree add <path> <branch>` →
  entity visible. Surfaced mechanic: a 2nd `worktree add` FATALS "already exists" → `state init` MUST
  guard with a path-exists check, not rely on git no-op (folded into B4 step 4).
- **(c) 2-writer `pull --rebase` sync.** PASS. A pushes; B's push rejected (non-fast-forward); B
  `pull --rebase` replays B's path-scoped commit atop A's → both entities present, linear history; B
  re-pushes; A `pull --rebase` sees B's. Distinct entity files → zero conflict (validates the
  path-scoped-commit + pull-rebase composition, B6).

Conclusion: every primitive the orphan-branch model needs is git-native and works end-to-end today;
the feature work is wiring (commission mechanics + `state init` + the two contract additions), not
inventing a sync mechanism.

## Migration note (this workflow's existing checkout — one-time, operational)

THIS workflow's `docs/dev/.spacedock-state` is currently shape-(a) (independent repo, no remote). To
adopt the orphan-branch model it must become a linked worktree of the main repo on an orphan branch
pushed to `origin`. That is a one-time operational migration (push the existing `spacedock-state/dev`
history to main `origin` as an orphan-style branch, re-add it as a linked worktree), NOT shipped
feature code. Flagged here so the captain can sequence it; it is not an AC of this entity.

## Internal sequencing proposal

1. **B1 + B2 (parser semantics) FIRST.** The `$inline` classifier + `state-branch` derivation are the
   foundation every other part reads; they are pure Go unit work with the cheapest proofs and zero
   architectural risk once the sentinel is chosen. Ship the shared `classifyState` + branch-derivation
   with their unit tests before anything depends on them. (This is the "what would invalidate the rest
   if it broke" item — the sentinel semantics — so it goes first, TDD-style.)
2. **B3 + B4 (commission + state init) SECOND.** Both are git-mechanics already spiked. B3 wires the
   commission scaffolding; B4 the `state init` subcommand. They depend on B1 (mode classification) and
   B2 (branch name). Proof: commission e2e + resume e2e.
3. **B5 + B6 (FO/ensign contracts) THIRD.** Both are skill-text contract additions keyed on the
   Phase-A boot field (B5) and the spiked sync points (B6). They depend on B4 (`state init` must exist
   for B5's halt to point at it) and on the path-scoped-commit rule (B6 extends it). Proof: static
   skill-text tests + the 2-writer e2e.

DEPENDS ON Phase A landing its boot `state_backend`/`entity_dir_present` fields (currently in
implementation) before B5 can key off them — sequence B5 after Phase A merges.

## Hardened acceptance criteria (behavior-first)

- **AC-1 — `state:` modes are explicit, via one shared classifier.** `$inline` → inline; empty →
  inline (backward-compat); a relative path → split-root; `$inline` is never joined as a literal path
  at any of the three read sites. Verified by status-parser Go unit tests over all three modes + the
  no-literal-join negative.
- **AC-2 — `state-branch` defaults to `spacedock-state/<slug>`, overridable.** Slug = workflow-dir
  basename (reproduces `spacedock-state/dev`); an explicit `state-branch:` wins. Verified by a Go unit
  derivation test; reproducible on this workflow.
- **AC-3 — Commission scaffolds both journeys.** Split commission → a usable orphan-branch state
  worktree + clean code branch; inline commission → beside-the-README entities. Verified by a
  commission e2e (git mechanics proven by §Spike (a)) + a static skill-text test on the new SKILL.md
  prose.
- **AC-4 — `spacedock state init` resumes a cloned shape-(1) repo, idempotently.** Fresh main clone →
  `state init` (fetch + worktree add, path-exists guarded) → `spacedock status` renders the entities;
  re-run is a no-op. Verified by an e2e mirroring §Spike (b).
- **AC-5 — FO sanity-check halts on uninitialized state and resumes after init.** `state_backend ==
  split-root && entity_dir_present == false` → FO halts + points at `state init`; after init proceeds.
  Verified by the FO-contract skill-text test + the Phase-A boot-field behavior (json_boot_test.go),
  with a thin live smoke folded into the B4 e2e.
- **AC-6 — Push/`pull --rebase` sync is contractual and exercised.** A 2-writer e2e: A commits+pushes,
  B `pull --rebase` sees it, and a push-rejection→pull-rebase→re-push round-trip; the FO push/pull-on-
  boot and ensign push-after-commit/pull-on-rejection rules carry the contract. Verified by the
  2-writer e2e (§Spike (c) proves the mechanism) + static skill-text tests.

## Test plan (cost / altitude)

- **B1** — Go unit tests on the shared `classifyState` over three modes + no-literal-join negative,
  exercised through all three call sites (resolveRoots, splitRootStateCheckout, discoverWorkflows
  prune). Cheap (~minutes), parser altitude. RED-first.
- **B2** — Go unit test on branch-name derivation (basename + override). Cheap, command altitude.
- **B3** — commission e2e (real git: orphan worktree, gitignore, status renders, clean code branch) +
  a static skill-text test on the SKILL.md mechanics prose. Moderate (real git, no mocks per the e2e
  rule); mechanism PROVEN by §Spike (a).
- **B4** — resume e2e (bare main remote + commissioned split workflow → fresh clone → `state init` →
  status renders; idempotent re-run). Moderate (real clone, no mocks); mechanism PROVEN by §Spike (b).
- **B5** — static skill-text test on first-officer SKILL.md (the halt-gate prose) + reuse Phase-A's
  `json_boot_test.go` for the boot field; a thin live smoke folded into B4's e2e. Cheap.
- **B6** — 2-writer sync e2e (real clones: push, rejection, pull --rebase, re-push) + static
  skill-text tests on the FO/ensign push/pull-rebase contract prose. Moderate (real git, no mocks);
  mechanism PROVEN by §Spike (c).

Total: 2 cheap Go-unit suites (B1/B2), 3 real-git e2es (B3/B4/B6, mechanisms all spiked → hardening
not discovery), and static skill-text assertions for the prose/contract changes (B3/B5/B6). No mocks
anywhere (live git per the e2e rule).

## Stage Report: ideation

- DONE: Design B1 + B2 (semantics): finalize a non-colliding explicit-inline sentinel for `state:` (captain suggested `$inline`) with empty→inline backward-compat and path→split-root, and the resolveRoots/splitRootStateCheckout handling (sentinel treated as inline, NOT joined as a path); plus the workflow-slug-derived `state-branch` default (`spacedock-state/<slug>`, overridable via README). Give concrete parser behavior + proof altitude (status-parser tests over all three modes; branch-name derivation test).
  B1 recommends ONE shared `classifyState` consumed by all THREE `state:` read sites (resolveRoots roots.go:55, splitRootStateCheckout helpers.go:127, discoverWorkflows prune handlers.go:378 — grounded), `$inline` treated as inline (never joined), with the no-literal-join negative test. B2 derives `spacedock-state/<basename>` (reproduces the shipped `spacedock-state/dev`), README override wins; Go-unit derivation proof.
- DONE: Design B3 + B4 + B5 (the journeys' mechanics): B3 commission scaffolding (EXACT orphan-branch-as-linked-worktree git mechanics) + inline; B4 `spacedock state init`; B5 the FO halt-gate keyed on the Phase-A `entity_dir_present` boot field. Name the proofs.
  B3 gives the spiked step-by-step (temp detached worktree → `--orphan` → clear inherited tree → seed → `git worktree add` at gitignored path → push); B4 the `state init` fetch+worktree-add with the path-exists idempotency guard the spike surfaced; B5 an FO-CONTRACT gate on `state_backend==split-root && entity_dir_present==false` (no new code — Phase A already ships the fields). Proofs: commission e2e, resume e2e, FO skill-text + Phase-A json_boot_test.
- DONE: Design B6 (multi-writer sync) + SPIKE the riskiest unknowns + propose internal sequencing + harden AC-1..AC-6.
  B6 = minimal sync points (FO pull-on-boot, push-after-commit, pull-rebase-on-rejection; NOT pre-every-dispatch) composing with the path-scoped-commit rule. SPIKED all three unknowns end-to-end (orphan-as-linked-worktree, state init from fresh clone, 2-writer pull --rebase) — ALL PASS, surfacing two real mechanics (orphan inherits source tree; 2nd worktree-add fatals) folded into B3/B4. Sequenced B1/B2 → B3/B4 → B5/B6; AC-1..AC-6 hardened behavior-first.

### Summary

Designed Phase B as the captain-fixed same-repo orphan-branch model, superseding Phase A's separate-repo recommendation. Headline findings: (1) `state:` is read at THREE sites that all treat non-empty as a path-to-join, so `$inline` needs ONE shared classifier or a site joins a literal `…/$inline` — the load-bearing B1 decision; (2) Phase A already ships the exact boot fields (`state_backend`, `entity_dir_present`) B5's halt-gate keys off, so B5 is a pure FO-contract addition with no new code; (3) all three riskiest unknowns were SPIKED end-to-end and PASS, surfacing two concrete mechanics (orphan checkout inherits the source tree → clear it before seeding; a 2nd `worktree add` fatals → `state init` must path-guard) now folded into the design. Sequencing: B1/B2 parser → B3/B4 commission+init → B5/B6 contracts, with B5 gated on Phase A merging. No code committed (ideation, design-only).
