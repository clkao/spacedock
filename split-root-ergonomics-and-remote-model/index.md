---
id: 5baenph5sbcrx7mhv3qbbjbn
title: Split-root ergonomics + remote/resume model — surface the state backend, recommend it at commission, define remote determination and 2nd-host resume
status: ideation
source: FO/captain (2026-05-31) — split-root cognitive-load + collaboration nuances surfaced during the behavior-coverage sprint (k6 fallout)
score: "0.32"
started: 2026-05-31T20:54:03Z
completed:
verdict:
worktree:
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
