---
id: 5baenph5sbcrx7mhv3qbbjbn
title: Split-root ergonomics + remote/resume model — surface the state backend, recommend it at commission, define remote determination and 2nd-host resume
status: implementation
source: FO/captain (2026-05-31) — split-root cognitive-load + collaboration nuances surfaced during the behavior-coverage sprint (k6 fallout)
score: "0.32"
started: 2026-05-31T20:54:03Z
completed:
verdict:
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-split-root-ergonomics-and-remote-model
issue:
---

The split-root state backend (`state:` README field → entities in a separate `.spacedock-state`
checkout) is **invisible and under-specified**, which caused real friction this sprint: the FO
hand-injected absolute state paths into every worktree dispatch, ensigns were unsure where to commit,
and the k6 bug class (binding the wrong state-checkout path) went undetected until an adversarial
audit. Two deeper nuances the captain raised turn this into a collaboration-model question, not just
ergonomics.

## Grounded current state (verified 2026-05-31)

- `state:` spec: optional README frontmatter; absent → single-root (entities beside the README, same
  repo); present (relative path) → split-root (`entityDir = definitionDir/<state>`). `resolveRoots`
  (status/roots.go:40-69) rejects absolute / `..`-escaping values. Single-root is the fallback.
- **Backend is never surfaced.** `status --boot --json` has 9 keys (command, mods, id_style, next_id,
  min_prefix, orphans, pr_state, dispatchable, team_state) — none name the backend or the state
  checkout. `printBoot` (handlers.go:271) is *handed* `roots.definitionDir` AND `roots.entityDir` and
  throws them away at the output boundary. The human `status` header shows only the entity table.
- **Commission says nothing.** `grep state:`/split-root in skills/commission/SKILL.md → zero hits. Every
  newly commissioned workflow defaults to single-root with no decision guidance.
- **No single-root negative test.** build_statecommit_test.go covers only split-root positive cases;
  the parity test STRIPS the guidance line before comparing, so a single-root guidance-LEAK regression
  is masked.
- **State checkout has NO remote.** `git -C docs/dev/.spacedock-state remote -v` → empty; branch
  `spacedock-state/dev` is local-only; the dir is gitignored by main, carried by no submodule/clone/init.
  Main remote is `git@github.com:spacedock-dev/spacedock.git`.
- **2nd-host resume is impossible today.** A fresh clone of the main repo gets the README but not the
  entities; there is no documented or tooled path to obtain/bootstrap the state checkout on a 2nd host.

## Scope (ideation to confirm; may recommend a phase split)

**A. Ergonomics (reduce FO/ensign cognitive load — root-cause fix):**
1. Surface the state backend in `status --boot --json` (and optionally a one-line human banner):
   `state_backend` (split-root|single-root), `definition_dir`, `entity_dir`. printBoot already holds
   both roots — a near-trivial emit. The FO then reads the checkout from boot (single source of truth),
   no inference, no hand-injected paths.
2. Commission `state:` recommendation: a single-root-vs-split-root decision rule for new workflows
   ("workflow embedded in a code repo whose PRs you care about → split-root; standalone → single-root").
3. Single-root negative test: pin that a single-root (no `state:`) dispatch body emits NO
   `This workflow is split-root` line and NO `git -C` guidance, worktree + non-worktree.

**B. Remote / collaboration model (the harder design — the captain's two nuances):**
4. **Remote determination.** How does the `.spacedock-state` checkout get its remote? Today: none.
   Design the convention — e.g. a declarative README field (`state-remote:`?), derivation from the main
   remote + a separate repo / branch namespace, or an explicit init step. What is discoverable vs
   configured? How does the FO/tooling learn it?
5. **2nd-host resume.** When a 2nd person/host clones the main repo, what is the bootstrap path to
   obtain the state checkout and meaningfully resume? Design the command/convention (a `spacedock`
   subcommand to clone/init the state checkout from the declared remote? a documented manual path?).
   Consider: the gitignored separate-repo model vs submodule vs a tracked-but-separate-branch model,
   and which preserves the "no state churn on the code branch" purpose while enabling resume.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — The state backend is observable from `status --boot` without inference.**
Verified by: a boot read exposes the backend kind + the resolved definition/entity dirs; a behavioral
test asserts the boot output names the split-root state checkout for a split workflow and the
single-root case for a non-split one.

**AC-2 — Commission recommends a backend.** Verified by: the commission skill carries the
single-root-vs-split-root decision rule (+ how to set up the state checkout/remote if split).

**AC-3 — Single-root "no guidance" is pinned.** Verified by: a test that goes RED if single-root
emits split-root state-commit guidance.

**AC-4 — Remote determination is defined.** Verified by: a documented, tooled-or-conventional way the
state checkout's remote is declared/discovered; reproducible on this workflow.

**AC-5 — 2nd-host resume is possible and documented.** Verified by: a reproducible path by which a
fresh main-repo clone obtains the state checkout and resumes (smallest end-to-end: clone main → run
the bootstrap → `status` renders the entities).

## Out of scope (ideation may push more here)
- Mods / PR-merge behavior (explicitly out of the bootstrap per the state spec).
- A full multi-writer concurrency/locking model beyond the existing path-scoped-commit rule.

## Notes — design-first; FO brings the ideation design to the captain
Parts A4/A5 (remote + resume) are genuine architectural design with multiple viable shapes
(separate-repo+remote vs submodule vs tracked-state-branch), so the FO will present the ideation
design for captain approval rather than auto-approving. Ideation should weigh a PHASE SPLIT — the
A-ergonomics trio is independently shippable and low-risk; the B remote/resume model is the larger
design — and recommend whether to split into two entities. Reference: docs/specs/state-behavior-extension.md
(the `state:` v0 spec + external-tracker bridge principles), docs/roadmap/bootstrap-roadmap.md.

---

# Ideation design (2026-05-31)

This design is decision-ready: each part names a recommendation, gives tradeoffs, and a concrete
reproducible proof. The riskiest unknown (the 2nd-host resume bootstrap) was spiked end-to-end —
result in Part B5. **Headline recommendation: phase-split into two entities** (rationale at the end).

## Part A — Ergonomics (root-cause fix for the split-root cognitive load)

This is the trio that kills the k6 bug class at its source. All three are low-risk and independently
shippable. A fourth item (A0) is added below: it is the *concrete* root cause the FO flagged and the
cheapest highest-leverage fix of all.

### A0 — Make the emitted state-commit path ABSOLUTE (the FO's concrete root cause)

**Defect.** `runBuild` (build.go:62) receives `workflowDir` verbatim from the `--workflow-dir` flag
(dispatch.go:32-36) — NO absolutization. `splitRootStateCheckout(workflowDir)` (helpers.go:127-135)
returns `filepath.Join(workflowDir, state)`. So `--workflow-dir docs/dev` emits a **relative**
`git -C docs/dev/.spacedock-state add …`. From the repo root (non-worktree stages) this is correct.
From inside a worktree (worktree stages, where cwd is `.worktrees/…` and has no `docs/dev`) it is
WRONG — `git -C docs/dev/.spacedock-state` fails. This is exactly why the FO had to hand-inject
absolute state paths into every worktree dispatch this sprint.

**Why the existing test missed it.** `build_statecommit_test.go:37` sets `workflowDir = root` where
`root = t.TempDir()` — always absolute. The relative-`--workflow-dir` case is never exercised.

**Fix.** Absolutize the state checkout at its source so the emitted path is cwd-independent. The
cleanest seam: absolutize `workflowDir` once at the top of `runBuild` (before line 143), so every
downstream join — README path, `splitRootStateCheckout`, the fetch line's `--workflow-dir`, the
state-commit guidance — inherits an absolute base. `entityPath` is already project-root-absolute
(Rule 12, build.go:124-129), so absolutizing the checkout makes BOTH halves of the `git -C … add …`
command absolute. Resolve relative `workflowDir` against the process cwd with `filepath.Abs`.

> Open sub-decision for the captain: absolutize `workflowDir` globally in `runBuild` (simplest, makes
> the emitted `--workflow-dir` fetch line absolute too — arguably *more* robust for the worker) vs.
> absolutize ONLY `stateCheckout` inside `splitRootStateCheckout` (narrowest change, leaves the fetch
> line spelling unchanged). Recommendation: **absolutize `workflowDir` once in `runBuild`** — one
> seam, and an absolute fetch-line `--workflow-dir` is strictly safer for a worker whose cwd may be a
> worktree. The parity test strips the state-commit guidance from both sides already
> (parity_harness_test.go), so the change does not disturb golden parity of the rest of the body; the
> fetch-line spelling change is the one thing to confirm against the oracle parity expectation.

**Proof (behavior-first, RED today).** New case in `build_statecommit_test.go`: drive `runNative`
with a **relative** `--workflow-dir` (e.g. run from the repo root and pass `docs/dev`-style relative
dir, or `t.Chdir(root)` then pass a relative workflow subdir) for a worktree stage, and assert the
emitted `git -C` path `filepath.IsAbs(...)` — i.e. the body contains `git -C ` followed by an
absolute path, never a relative `git -C docs/`. This is RED before the fix and GREEN after.

### A1 — Surface the state backend in `status --boot --json`

**Today.** `bootJSON` (json_commands.go:140-190) emits 9 keys; none name the backend. `printBoot`
(handlers.go:271) is HANDED `roots.definitionDir` AND `roots.entityDir` (the resolved roots from
`resolveRoots`, roots.go:40-69) and discards them at the output boundary. `gatherBoot`/`bootData`
(boot.go:165-201) already receive `definitionDir` and `entityDir` as parameters.

**Design.** Add three fields to the boot envelope and to `bootData`:
- `state_backend`: `"split-root"` when `entityDir != definitionDir`, else `"single-root"`. (The
  divergence of the two roots is the canonical signal — `resolveRoots` only diverges them when a
  non-empty `state:` is present.)
- `definition_dir`: the absolute README root (`roots.definitionDir`).
- `entity_dir`: the absolute entity/state root (`roots.entityDir`).

Both dirs are emitted ABSOLUTE (the I/O spellings, not the as-passed spellings), so the FO reads the
state checkout from boot as a single source of truth with no inference and no hand-injected paths —
directly retiring the friction that motivated A0's symptom-side workaround.

**Placement.** Append the three keys after `team_state` (last position). The boot-JSON key-order
contract (json_boot_test.go:14-17) is load-bearing for the FO parser; appending at the end preserves
every existing key's relative order. Update `bootJSONKeys` to include the three new trailing keys.

**Wiring.** `gatherBoot`/`bootJSON` already thread `definitionDir`/`entityDir`; carry them onto
`bootData` (e.g. `stateBackend`, `definitionDir`, `entityDir` fields) and emit in `bootJSON`. The
text `printBoot` MAY add a one-line `STATE_BACKEND: split-root (entity_dir: …)` banner — RECOMMEND
adding it for human parity, since the FO's startup is JSON but a captain reading the text boot should
see it too. Low cost, same data.

**Proof (AC-1).** A behavioral test (extend json_boot_test.go) asserting: for a split-root fixture
`state_backend == "split-root"`, `entity_dir` ends in the state path and is absolute, and
`entity_dir != definition_dir`; for a single-root fixture `state_backend == "single-root"` and
`entity_dir == definition_dir`. Naming the *split checkout for a split workflow and single-root for a
non-split one* is the AC-1 wording, satisfied directly.

**Bonus diagnostic the spike surfaced (see B5).** A 2nd host that has not bootstrapped the state
checkout renders an EMPTY entity table with exit 0 and `--validate` says VALID — a silent failure.
Once boot emits `state_backend`/`entity_dir`, the FO (or a future `doctor` check) can distinguish
"split-root, entity_dir exists, 0 entities" from "split-root, entity_dir MISSING" and prompt the
resume bootstrap. Surfacing the backend is therefore not just ergonomics — it is the natural
diagnostic seam for Part B's resume. RECOMMEND boot ALSO emit whether `entity_dir` exists on disk
(e.g. `entity_dir_present: "true"|"false"`) so the absent-state-checkout case is observable rather
than masquerading as an empty workflow. (Captain decision: include this 4th field now, or defer to
Part B where resume lives. Given it is one `os.Stat`, RECOMMEND including it now — it makes A1's test
able to pin the silent-empty failure and gives Part B a ready signal.)

### A2 — Commission recommends a backend (decision rule)

**Today.** `grep state:`/split-root in skills/commission/SKILL.md → zero hits. Every commissioned
workflow defaults to single-root with no guidance. The README-generation section (SKILL.md §2a,
~line 269) writes frontmatter but never mentions `state:`.

**Design — decision-rule wording** (to add to commission's README-frontmatter guidance, §2a):

> **State backend (`state:` field).** Decide where the workflow's mutable entity state lives:
> - **Split-root** (`state: .spacedock-state` in the README frontmatter): use when the workflow is
>   embedded in a code repo whose PRs/branches you care about. The README (the living spec) stays on
>   your code branch; the mutable entity state lives in a separate `.spacedock-state` checkout, so
>   routine stage transitions never churn the code branch and never collide with a feature PR.
> - **Single-root** (omit `state:`): use for a standalone workflow that is not embedded in a code
>   repo you ship from — the entities live beside the README in the same directory.
>
> If you choose split-root, also see [the state-checkout setup] for how to create the
> `.spacedock-state` checkout and (for collaboration/resume) declare its remote.

The bracketed setup pointer resolves to Part B's convention (the `state-remote:` field + init
command). RECOMMEND commission, when it generates a split-root README, ALSO scaffold the empty
`.spacedock-state` checkout (a `git init` on the state branch) so a freshly commissioned split
workflow is immediately usable — but this scaffolding step depends on Part B's init design, so it is
a B-phase follow-on, not an A2 blocker. A2 ships the decision-rule prose alone.

**Proof (AC-2).** A static skill test (commission has skill-text tests) asserting SKILL.md carries
the split-root-vs-single-root decision rule: the `state: .spacedock-state` string, the "embedded in a
code repo whose PRs you care about" trigger phrase, and the single-root "omit `state:`" guidance.
This is the right proof altitude — A2 is an instruction-text change, so a static text assertion is
correct (not a runtime test).

### A3 — Single-root negative test (pin "no split-root guidance leaks")

**Today.** `build_statecommit_test.go` covers only split-root positive cases. The parity test STRIPS
the guidance line before comparing (parity_harness_test.go:97-100), so a single-root guidance-LEAK
regression is masked — a single-root dispatch could start emitting the split-root `This workflow is
split-root` block and no test would catch it.

**Design — the negative test (AC-3).** A new test that builds a SINGLE-root workflow (no `state:`
field) and asserts the dispatch body emits:
- NO `This workflow is split-root` sentence,
- NO `git -C` state-commit command,

for BOTH a worktree stage and a non-worktree stage (the two branches in build.go:310-332). RED if a
regression makes single-root emit split-root guidance. This is the dedicated guard the stripped
parity test cannot provide.

## Part B — Remote / collaboration model (the harder design)

### B4 — Remote determination: how the `.spacedock-state` checkout gets its remote

**Today.** The state checkout (`docs/dev/.spacedock-state`) is a STANDALONE git repo (its own `.git`
dir, branch `spacedock-state/dev`) gitignored by the main repo, with NO remote, no submodule wiring,
carried by no clone/init. Main remote: `git@github.com:spacedock-dev/spacedock.git`.

**Three shapes evaluated** (the captain's named alternatives), against the two hard requirements:
(R1) **no state churn on the code branch**, and (R2) **2nd-host resume**.

| Shape | R1 no code-branch churn | R2 resume | Notes |
|---|---|---|---|
| **(a) Separate repo + remote** (status quo structure + declare a remote) | YES — state is a separate repo, gitignored by main; the code branch never sees state commits | YES — `git clone <state-remote>` into the state path (spiked, B5) | Two repos to host; needs the remote DECLARED so it is discoverable |
| **(b) Git submodule** | PARTIAL — submodule *pointer* (gitlink + .gitmodules) lives ON the code branch; every state advance bumps the pinned SHA → a commit ON the code branch | YES — `git clone --recurse-submodules` | Violates R1: state transitions churn the code branch via gitlink bumps; also couples state to a code-branch commit, the exact thing split-root exists to avoid |
| **(c) Tracked-but-separate state branch** (one repo, an orphan `spacedock-state` branch in the MAIN repo) | YES at the branch level (state commits land on the orphan branch, not the code branch) but the state checkout becomes a second worktree/checkout of the SAME repo | YES — `git fetch origin spacedock-state && git worktree add … spacedock-state` | Reuses the main remote (no second repo to host, no second remote to declare) — derivable. But the state checkout is a worktree of the code repo, so it shares the code repo's hooks/config and a `git -C` into it is a worktree of main, not an independent repo — more coupling than (a) |

**Recommendation: (a) separate repo + remote, with a DECLARED remote.** It is the only shape that
fully satisfies R1 (zero code-branch churn — no gitlink bumps, no shared repo) AND matches the
already-shipped structure (the state checkout is *already* a standalone repo). (b) is rejected: a
submodule's whole point is to pin a SHA on the parent branch, which re-introduces code-branch churn
on every state transition — the precise problem `state:` exists to eliminate. (c) is viable and has a
real upside (no second remote to host/declare — derive `spacedock-state` branch from the main
remote), and is worth presenting as the **fallback** if the captain prefers a single-remote
operational model; its cost is that the state checkout becomes a worktree of the code repo (more
coupling, shared hooks/config) rather than an independent repo.

**Convention for declaring the remote (shape (a)).** Add a declarative README frontmatter field
alongside `state:`:

```yaml
state: .spacedock-state
state-remote: git@github.com:spacedock-dev/spacedock-state.git
state-branch: spacedock-state/dev    # optional; default "spacedock-state/<workflow>" or a fixed default
```

- **Discoverable, not configured.** `state-remote` (and optional `state-branch`) live in the README,
  which is on the code branch and present in every clone. So a fresh main clone ALREADY KNOWS where
  the state repo is — the resume bootstrap (B5) reads it from the README with no out-of-band config.
- **Derivation fallback.** If `state-remote` is absent, a convention can derive it from the main
  remote (e.g. main `…/spacedock.git` → `…/spacedock-state.git`), but an explicit field is clearer
  and avoids guessing the hosting layout. RECOMMEND explicit `state-remote`, with derivation as a
  documented convenience only if the captain wants zero-config defaults.
- **What is configured locally vs declared.** The README DECLARES the remote URL + branch. The local
  state checkout's `origin` is wired by the init/resume command (B5) from that declaration — not
  hand-set per host. So nothing host-specific lives in the README; the declaration is portable.

**Proof (AC-4).** Reproducible on THIS workflow: add `state-remote`/`state-branch` to
`docs/dev/README.md`, point the existing `docs/dev/.spacedock-state` checkout's `origin` at it,
push the `spacedock-state/dev` branch. Tooled discovery proof: `spacedock` reads `state-remote` from
the README (a parser/CLI test that the field is parsed and surfaced — naturally folds into A1's boot
surfacing if boot also emits `state_remote`). The convention is documented in
state-behavior-extension.md as the v0.1 remote profile.

### B5 — 2nd-host resume: SPIKED end-to-end ✅ (the riskiest unknown)

**Spike (smallest end-to-end path, run 2026-05-31):**
1. Built a "main repo" with a split-root workflow (`state: .spacedock-state`, state path gitignored),
   pushed to a bare main remote.
2. Built the state checkout as a standalone repo with one entity on branch `spacedock-state/dev`,
   pushed to a bare STATE remote.
3. **Host 2: fresh `git clone <main-remote>`** → state checkout ABSENT (expected — gitignored,
   separate repo). `spacedock status --workflow-dir docs/dev` → renders an EMPTY table, exit 0
   (the silent-empty failure noted in A1).
4. **BOOTSTRAP: `git clone -b spacedock-state/dev <state-remote> docs/dev/.spacedock-state`.**
5. **Host 2: `spacedock status` → renders the widget entity, identical to host 1.** ✅

**Conclusion — what is missing today is ONLY the declared remote + a one-command bootstrap.** No
status/render code change is needed for resume itself; status already reads the state checkout
correctly once it is present. The state checkout is already a standalone repo with full commit
history (`git -C docs/dev/.spacedock-state log` shows the real sprint history). The single missing
primitive is: **the state checkout's history is reachable from a declared remote, plus a command/
convention that clones it into the declared `state:` path.**

**Design — the resume bootstrap.** A `spacedock` subcommand (working name `spacedock state init`
/ `spacedock state resume`, exact name a captain call) that:
1. Reads `state:`, `state-remote`, `state-branch` from `{workflow_dir}/README.md`.
2. If the `state:` path is absent on disk: `git clone -b <state-branch> <state-remote> <state-path>`.
3. If present: a no-op (or `git -C <state-path> fetch && status`) — idempotent.

This is the tooled half of AC-5. The smallest end-to-end (clone main → run the bootstrap → `status`
renders the entities) is exactly the spike above, so AC-5 is PROVABLE today with only this command +
the B4 declaration. A documented MANUAL path (the literal `git clone -b … <state-remote> <state-path>`
one-liner) is the fallback if a subcommand is deferred — the manual path already works (it IS the
spike), so AC-5's "documented" half can ship even before the subcommand.

**Proof (AC-5).** An end-to-end test (CLI/integration): create main + state bare remotes from
fixtures, fresh-clone main, run the bootstrap (subcommand or the documented `git clone` line),
assert `spacedock status` renders the seeded entity. This mirrors the spike and is the right proof
altitude — resume is a runtime/integration claim, so a live-ish fixture e2e (not a static prose
test) is correct.

## Hardened acceptance criteria (behavior-first)

- **AC-0 (new) — The emitted state-commit `git -C` path is absolute (cwd-independent).** Verified by:
  a dispatch-build test driving a RELATIVE `--workflow-dir` for a worktree stage asserts the emitted
  `git -C` path is absolute (RED today, GREEN after A0). This is the root-cause fix; it directly
  retires the FO's hand-injection of absolute state paths.
- **AC-1 — The state backend is observable from `status --boot --json` without inference.** Verified
  by: a behavioral boot test asserting `state_backend` (split-root|single-root), absolute
  `definition_dir`/`entity_dir`, `entity_dir != definition_dir` for a split fixture and `==` for a
  single-root fixture, and (recommended) `entity_dir_present` distinguishing an absent state checkout
  from an empty one.
- **AC-2 — Commission recommends a backend.** Verified by: a static skill-text test asserting
  SKILL.md carries the split-root-vs-single-root decision rule (the trigger phrase + `state:` usage +
  the single-root "omit" guidance).
- **AC-3 — Single-root "no guidance" is pinned.** Verified by: a dispatch-build test that goes RED if
  a single-root (no `state:`) dispatch body emits the `This workflow is split-root` sentence or a
  `git -C` state-commit command, worktree AND non-worktree branch.
- **AC-4 — Remote determination is defined.** Verified by: `state-remote`/`state-branch` README
  fields are parsed and surfaced (a parser/CLI test), and the convention is reproducible on this
  workflow (declare + push the existing checkout); documented in state-behavior-extension.md.
- **AC-5 — 2nd-host resume is possible and documented.** Verified by: an e2e test — fresh main clone
  → run the bootstrap (subcommand or documented `git clone -b <branch> <state-remote> <state-path>`)
  → `spacedock status` renders the seeded entity. Proven smallest-path by the B5 spike.

## Test plan (cost / altitude)

- **A0** — one Go dispatch-build unit test (relative-`--workflow-dir` worktree case). Cheap (~minutes),
  Go unit altitude (command-emission behavior). RED-first.
- **A1** — extend json_boot_test.go with split + single-root fixtures; assert the new keys + ordering.
  Cheap, Go unit / golden altitude. Update `bootJSONKeys`.
- **A2** — static skill-text test on commission SKILL.md. Cheap, static-text altitude (correct for an
  instruction change).
- **A3** — one Go dispatch-build negative test (single-root, both branches). Cheap, Go unit altitude.
- **B4** — parser/CLI test for `state-remote`/`state-branch`; reproduce the declaration on this
  workflow (manual, one-time). Low cost.
- **B5** — one e2e/integration test with bare-remote fixtures (mirrors the spike). Moderate cost
  (real git clone, no mocks — live fixtures per the e2e rule). Live workflow / integration altitude,
  correct because resume is a runtime claim. The mechanism is already proven (spike), so the test
  hardens a known-working path rather than discovering one.

## Phase-split recommendation: SPLIT into two entities

**Recommend shipping A as its own entity NOW (A0–A3), and B as a separate follow-on entity (B4–B5).**

Rationale:
- **A is the root-cause fix for the k6 bug class and is independently shippable + low-risk.** A0
  (absolute path) alone retires the FO's hand-injection friction this sprint; A1–A3 are near-trivial
  emits/tests with no architectural choice to resolve. None of A depends on the remote model.
- **B carries the genuine architecture (the captain's two nuances) and a captain decision** — the
  shape choice (separate-repo+remote vs submodule vs tracked-state-branch) and the new
  `state-remote` convention. The spike DE-RISKED B (resume works with only a declared remote + a
  clone command), but the convention + subcommand are real design that warrants its own gate.
- **A unblocks B's diagnostic.** A1's boot surfacing (and the `entity_dir_present` signal) is the
  natural seam Part B's resume hangs off — shipping A first gives B a ready "state checkout absent"
  signal instead of B having to build it.

So: **Entity 1 (ship now): A0 boot-path-absolute + A1 boot surfacing + A2 commission rule + A3
single-root negative test.** **Entity 2 (follow-on, captain-gated design): B4 remote determination
(`state-remote` convention, separate-repo recommendation) + B5 resume bootstrap subcommand + e2e.**
AC-0..AC-3 ride Entity 1; AC-4..AC-5 ride Entity 2.

## Stage Report: ideation

- DONE: Design Part A (ergonomics): boot state-backend surfacing + commission decision-rule + single-root negative test, each with a concrete reproducible proof
  Part A0–A3 above: A0 names the FO's concrete root cause (relative `git -C` from a worktree; build.go:62/dispatch.go:32 pass workflowDir verbatim, helpers.go:127 joins it relative) with a RED-first relative-`--workflow-dir` proof; A1 wires `state_backend`/`definition_dir`/`entity_dir` (+ `entity_dir_present`) through bootJSON (json_commands.go:140) appended after team_state, proof via json_boot_test.go; A2 gives verbatim commission decision-rule prose + static-text proof; A3 gives the single-root no-leak negative test (both branches).
- DONE: Design Part B (remote/collaboration model): remote determination + 2nd-host resume, three shapes evaluated, ONE recommended with explicit tradeoffs
  B4 evaluates separate-repo+remote vs submodule vs tracked-state-branch against R1 (no code-branch churn) + R2 (resume) in a table; recommends (a) separate-repo + declared `state-remote`/`state-branch` README fields (submodule rejected: gitlink bumps churn the code branch; tracked-branch offered as single-remote fallback). B5 designs the resume bootstrap (`spacedock state init` reading the README declaration; documented `git clone -b` manual fallback).
- DONE: Spike the riskiest unknown (resume bootstrap) — smallest end-to-end, then phase-split + AC hardening
  Spiked end-to-end (B5): fresh main clone → state absent → `git clone -b spacedock-state/dev <state-remote> <state-path>` → `spacedock status` renders the entity identical to host 1. Proved the ONLY missing primitive is a declared remote + a clone command (no status code change). Hardened AC-0..AC-5 behavior-first; recommended SPLIT into two entities (A0–A3 ship now, B4–B5 captain-gated follow-on).

### Summary

The FO's concrete root cause is confirmed and folded in as A0: `runBuild` passes `--workflow-dir` verbatim, so `splitRootStateCheckout` emits a RELATIVE `git -C docs/dev/.spacedock-state` that works from the repo root but breaks inside a worktree — the source of the FO's hand-injected absolute paths. The existing state-commit test missed it because it always passes an absolute TempDir. The resume unknown was de-risked by an end-to-end spike: status already reads the state checkout fine; the only gap is a declared remote + a one-command clone. Recommendation: phase-split — ship the low-risk A-ergonomics trio (A0 absolute path + A1 boot surfacing + A2 commission rule + A3 negative test) as its own entity now, and take B (remote convention + resume bootstrap) to the captain as a separate gated design, since A1's boot surfacing is the natural diagnostic seam B's resume hangs off. NO code committed (ideation, design-only).
