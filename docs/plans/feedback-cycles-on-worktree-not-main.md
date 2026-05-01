---
id: k9s1zdwzfmdzdayjdvjjpj6f
title: "FO-owned `### Feedback Cycles` section should live on the worktree branch, not main"
status: ideation
source: "Captain observation 2026-04-30 during PR #176 + PR #177 rebase: feedback cycles entries on main collide with worktree-branch stage reports, producing painful merge conflicts in the entity body when the PR lands"
started: 2026-05-01T00:48:15Z
completed:
verdict:
score: 0.65
worktree:
issue:
pr:
mod-block:
---

The FO Write Scope (shared-core) currently gives the first officer write rights to the `### Feedback Cycles` section in entity bodies on main:

> **`### Feedback Cycles` section** — in entity bodies, tracking rejection rounds

In practice, during worktree-backed stages (implementation / validation), the worktree branch appends stage reports to the same entity file. The FO appends Feedback Cycles entries to the same file on main when a gate is rejected. Both regions live in the bottom of the entity body, sequentially. When the PR lands, `git merge` or `git rebase` has to hand-resolve the overlap because each side advanced different lines in the same trailing region.

PR #176 and PR #177 (2026-04-30) both went DIRTY/CONFLICTING for this exact reason. The captain's `/clear` session had to manually resolve the merge in the worktree because the cycle entries on main and the cycle-2 stage reports on the branch were adjacent and the merge driver couldn't tell which order they should interleave. The work is mechanical but slow and error-prone — it costs a captain turn every PR merge, and it scales linearly with the number of feedback cycles a PR went through.

## Proposed approach

### The conditional rule

Route `### Feedback Cycles` writes by the entity's `worktree:` frontmatter field at the moment the FO is about to write the cycle entry:

- **`worktree:` is non-empty** — the entity has a live worktree on disk. The FO writes the cycle entry to the entity file *inside that worktree* (`{worktree_path}/{workflow_dir}/{slug}.md`) and commits on the worktree branch. The cycle entry rides the worker's next stage-report commit into the merge, so main only sees Feedback Cycles entries when the PR lands.
- **`worktree:` is empty** — no worktree exists. The FO writes to main (the existing behavior), since there is no branch to defer to.

The decision is point-in-time: it depends only on whether `worktree:` is currently set, not on which stage is being rejected. This keeps the rule trivial to evaluate and audit-friendly.

### Architectural answers

**(a) When is the worktree first available?** In this workflow, `implementation` is the first worktree-backed stage. The worktree is created at dispatch (shared-core "Dispatch" step 7), at which point `worktree:` is populated in frontmatter. Therefore:
- Rejection at the **ideation gate** (and any feedback before the first worktree-creating dispatch) finds `worktree:` empty and writes to main. This is the historical behavior and remains correct — there is no branch to defer to.
- Rejection at the **validation gate** (which routes `feedback-to: implementation`) finds `worktree:` set and writes to the worktree.

The rule fires off the field, not off the stage name, so future workflows that change which stages are worktree-backed inherit the right behavior automatically.

**(b) Bare mode / single-entity mode.** Bare mode (no teams) is orthogonal to worktree presence. In `claude -p` and `codex exec`, the FO still has filesystem access to the worktree path whenever `git worktree add` has run for the entity. The rule applies uniformly: read `worktree:`, route accordingly. No bare-mode exception.

**(c) Backward compat & audit trail.** The FO Write Scope clause currently authorizes `### Feedback Cycles` writes on main without qualification. The new clause must:
- preserve main-write authorization for entities with no worktree, so historical entries on main remain valid and ideation-stage rejections continue to work
- add the worktree-write authorization for entities with a live worktree
- name the conditional explicitly so a future reader can tell, from the spec alone, why some Feedback Cycles entries are on main and others on a branch

The clause stays in the same `## FO Write Scope` allow-list block; it gains a sub-clause naming the routing rule.

### Source files that change

1. `skills/first-officer/references/first-officer-shared-core.md`
   - **Section `## FO Write Scope`** (line 218): replace the bare `**\`### Feedback Cycles\` section** — in entity bodies, tracking rejection rounds` bullet with a conditional-routing bullet that names both the worktree-write and main-write cases.
   - **Section `## Feedback Rejection Flow`** (line 178): replace `The first officer owns the \`### Feedback Cycles\` section and keeps it on the main branch.` with wording that names the routing rule and points back to FO Write Scope. Step 2 of the flow ("Track feedback cycles in a `### Feedback Cycles` section in the entity body.") gains a sub-bullet: "Write the entry inside the worktree (`{worktree_path}/{workflow_dir}/{slug}.md`) and commit on the worktree branch when `worktree:` is set; write to main otherwise."
   - **Section `## Worktree Ownership`** (line 206): the existing bullet "Ordinary active-state writes like `implementation -> validation` do not land on `main`" implicitly covers cycle entries once Write Scope is updated. Add a parenthetical naming Feedback Cycles explicitly so readers don't have to cross-reference.

2. `skills/first-officer/references/claude-first-officer-runtime.md` and `codex-first-officer-runtime.md`
   - The runtime adapters do not currently restate the Feedback Cycles location. Audit each adapter for any prose that says "on main" or "main branch" in proximity to Feedback Cycles; update only if a contradiction with the new rule exists.

### Where the conditional fires

The FO's gate-rejection path (shared-core "Feedback Rejection Flow", step 2) is the only place the FO writes `### Feedback Cycles`. The rule fires there: read `worktree:` from frontmatter, branch the write target, commit to the appropriate branch.

No new code helper is needed — the FO already does both kinds of writes elsewhere (worktree-side via `git -C {path}` or path-prefixed Write/Edit, main-side via direct Write/Edit). This is one conditional in one rejection-flow spec section, not a refactor.

### Migration story

- **Entities not yet rejected** when this lands: get the new behavior on first rejection.
- **Entities mid-cycle with prior Feedback Cycles entries on main**: those historical entries stay where they are; new entries follow the new rule. If an entity has cycle-1 on main (from before this change) and cycle-2 lands on the worktree (after this change), the merge still works — cycle-1 is already on main, cycle-2 arrives via the worktree branch, no overlap. The PR #176/#177 conflict was specifically about *both sides advancing in the same trailing region*, which the new rule prevents going forward.
- No frontmatter migration, no entity-file rewrite, no script. The rule changes future writes; past writes are left alone.

## Relationship to #165

Task 165 (entity-state-sidecar-file) targets the same broader pain — state churn on main during worktree-backed stages — but at the frontmatter layer (`status`, `worktree`, `pr`, `mod-block`, `verdict`). Option (b) of #165 ("State transitions live in the worktree during worktree-backed stages") would incidentally pull Feedback Cycles into the worktree if implemented broadly, but #165's current scope is explicit about frontmatter. This task is the body-section sibling: narrower, cheaper to ship, doesn't require the discovery-side rework #165 would force on `status --boot`.

If the captain prefers, this task can be folded into #165 as a sub-scope; otherwise it stands alone as a smaller, independently-mergeable surface.

## Acceptance criteria

**AC-1 — When `worktree:` is set on an entity, the FO writes new `### Feedback Cycles` entries inside the worktree copy of the entity file and commits on the worktree branch; the main copy of the entity file is untouched until PR merge.**
Verified by: extending `tests/test_rejection_flow.py` (the existing live validation-rejection test) with two assertions after the FO routes feedback back to implementation: (1) `git -C {worktree_path} show HEAD:{workflow_dir}/{slug}.md` contains a `### Feedback Cycles` block with the new cycle entry; (2) `git show main:{workflow_dir}/{slug}.md` does *not* contain the new cycle entry. Test shape mirrors the existing FO log + frontmatter assertions in that file. Use the live-claude tier already wired for that test.

**AC-2 — Two consecutive feedback cycles on the same worktree-backed entity merge into main with zero conflicts in the entity file.**
Verified by: a new offline test `tests/test_feedback_cycles_merge_clean.py` that builds a fixture under `tests/fixtures/feedback-cycles-merge/` containing:
- a workflow `README.md` with the standard four-stage spacedock layout
- one entity file `task.md` with frontmatter `status: validation`, `worktree: .worktrees/task`, and a body that includes one `## Stage Report: implementation` section followed by one `## Stage Report: validation` section (representing cycle-1)

The test scripts the conflict scenario directly (no live FO required): create a worktree branch off main, append a second `## Stage Report: implementation` (cycle-2) and a third `## Stage Report: validation (cycle 2)` to the worktree copy, append a `### Feedback Cycles` entry to the *worktree* copy (new behavior — between cycle-1 validation and cycle-2 implementation reports), and run `git merge --no-ff {branch}` from main. Assertion: `git merge` exits 0 *and* `git diff --check` reports zero conflict markers in `task.md` *and* the merged file contains exactly one `### Feedback Cycles` block with the cycle entry. A control case in the same test, with the cycle entry written on main instead of on the branch, must reproduce the conflict (non-zero exit or `<<<<<<<` markers) — proving the test exercises the actual failure mode.

**AC-3 — `skills/first-officer/references/first-officer-shared-core.md` describes the conditional routing rule in both the FO Write Scope allow-list and the Feedback Rejection Flow section, naming `worktree:` as the deciding field.**
Verified by: extending `tests/test_repo_edit_guardrail.py`'s static-content phase (it already greps the assembled FO content for FO Write Scope items) with checks that the assembled text contains the substring `Feedback Cycles` in proximity (within 200 characters) to both `worktree:` and `main`, and that the Feedback Rejection Flow section names the conditional. Greps run against the assembled agent content, not the raw shared-core file, so the cross-reference from the FO assembly chain is exercised.

**AC-4 — When `worktree:` is empty, the FO continues to write `### Feedback Cycles` entries directly to the main copy of the entity file.**
Verified by: a unit-style assertion in the offline test introduced for AC-2. Construct an entity with `worktree:` empty in frontmatter, invoke the FO's cycle-write helper (or, if the conditional remains inline in the FO spec rather than a helper, simulate the write the spec describes), assert the entry lands on the main copy. The Feedback Rejection Flow only fires when a stage has `feedback-to`; this workflow's only `feedback-to` happens to be between two worktree-backed stages, so the empty-`worktree:` case is exercised against a generic fixture workflow rather than the spacedock-plans workflow itself. Use `tests/fixtures/feedback-cycles-merge/` extended with a sibling `feedback-cycles-no-worktree/` fixture whose stage layout has a non-worktree `feedback-to` edge (e.g., a gated draft → review pair both with `worktree: false`).

**AC-5 — The conditional routing applies in bare mode (single-entity / `claude -p`) identically to teams mode: the rule fires off `worktree:`, not off teams availability.**
Verified by: the offline merge-clean test from AC-2 covers the routing logic directly without running an FO at all (it constructs the file states the FO would produce). For runtime confirmation, run the existing `tests/test_rejection_flow_codex.py` (codex exec is bare-mode by construction) with the same worktree-side / main-side split assertions added in AC-1 — a single shared assertion helper in `scripts/test_lib.py` keeps both AC-1 and AC-5 reading from the same checked behavior.

**AC-6 — Pre-existing entities whose Feedback Cycles entries already live on main are not rewritten or migrated by this change; the new rule applies only to writes that occur after the change ships.**
Verified by: a comment-level note in the shared-core change naming the no-migration policy, plus a fixture-style check in the offline merge-clean test (AC-2): start the entity with a `### Feedback Cycles` entry already on main (from before the change), apply the new rule for the next cycle on the worktree, run `git merge`, and assert the merged file contains both entries in correct order with zero conflict markers.

## Test plan

| AC | Test | Tier | Approx cost |
|----|------|------|-------------|
| AC-1 | Extend `tests/test_rejection_flow.py` with worktree-vs-main split assertions on the cycle entry | live-claude (existing tier) | adds ~0 budget — same FO run, two extra assertions |
| AC-2, AC-6 | New offline test `tests/test_feedback_cycles_merge_clean.py` + fixture under `tests/fixtures/feedback-cycles-merge/` | offline (`make test-static`) | seconds; no live model |
| AC-3 | Extend `tests/test_repo_edit_guardrail.py` static-content phase with conditional-routing greps | offline (assembled-agent content check) | seconds; no live model |
| AC-4 | Sibling fixture `tests/fixtures/feedback-cycles-no-worktree/` exercised by the same offline test | offline | seconds; no live model |
| AC-5 | Extend `tests/test_rejection_flow_codex.py` with the same split assertions as AC-1 | live-codex (existing tier) | adds ~0 budget — same FO run, two extra assertions |

Static / offline tests are the primary proof — they directly exercise the merge behavior the change is about. Live tests are the runtime confirmation that the FO actually follows the rule. No new live-only test is added; both live tests reuse existing fixtures and FO runs.

E2E coverage is not needed beyond the existing rejection-flow tests because the conditional is a one-line spec rule firing off a frontmatter field already read elsewhere — the offline fixture proves the merge property the change targets, and the live extensions prove the FO actually writes to the worktree path. Static checks alone would not have caught PR #176/#177's pain because the pain is mechanical (overlapping line ranges); the offline merge fixture is the exact-abstraction proof.

## Stage Report: ideation

- DONE: Resolve the architectural questions the captain flagged at intake: (a) when is the worktree FIRST available for FO writes — only after the first dispatch into a worktree-backed stage, so feedback rejection at *backlog* or *ideation* (which precede the first worktree-creating stage) must keep writing on main; (b) bare-mode: in single-entity / -p mode the FO still has filesystem access to the worktree path even without a live team — does the worktree-write rule still apply, or does bare mode keep main-writes for feedback?; (c) backward compat: the FO Write Scope clause currently authorizes ONLY `### Feedback Cycles` and frontmatter on main — the new rule has to express the conditional cleanly so future readers don't lose the audit trail of allowed main-writes.
  Answered in `## Proposed approach` → `### Architectural answers`. (a) routing fires off frontmatter `worktree:`, not stage name; pre-worktree rejections write main. (b) bare mode is orthogonal; rule fires off `worktree:` uniformly. (c) FO Write Scope sub-clause names both branches of the conditional plus rationale.
- DONE: Replace the `## Sketch of the fix` section with a concrete `## Proposed approach` covering: which FO source files change (shared-core + claude-runtime adapter wording), where in the FO's gate-rejection flow the conditional fires, and what the migration story is for entities mid-cycle when this lands.
  `## Proposed approach` now names `skills/first-officer/references/first-officer-shared-core.md` line 218 (FO Write Scope), line 178 (Feedback Rejection Flow ownership statement), step 2 of the rejection flow, and the worktree-ownership cross-reference; flags the runtime adapters for audit; and gives a no-migration policy for mid-flight entities.
- DONE: Refine the AC list: each AC must be an end-state property with a `Verified by:` clause that names a specific test file + test shape. Tighten AC-2's repro fixture spec — name the fixture path under `tests/fixtures/`, the entity body shape it constructs, and which assertion proves no conflict (e.g., `git merge` exit 0 + zero conflict markers). Add an AC if architectural question (a)/(b)/(c) reveals one — captain prefers ACs over hand-waving.
  AC-1 names extension to `tests/test_rejection_flow.py`. AC-2 names fixture `tests/fixtures/feedback-cycles-merge/`, entity-body shape (frontmatter + cycle-1 implementation/validation reports), and assertions (`git merge` exit 0 + `git diff --check` zero markers + control case proves the failure mode). AC-3 names `tests/test_repo_edit_guardrail.py` extension. AC-4 covers the empty-worktree case via a sibling fixture. AC-5 (new) covers bare-mode equivalence — answers (b). AC-6 (new) covers the no-migration policy — answers (c). All AC-N items are end-state properties with `Verified by:` clauses naming a specific test file + shape.

### Summary

Hardened the spec into a single-conditional design: the FO routes `### Feedback Cycles` writes by reading frontmatter `worktree:` at write time, with no bare-mode exception and no migration needed. Six entity-level ACs cover (a) the worktree-set case in live testing, (b) merge-clean proof in an offline fixture that includes a control case to prove the test exercises the actual failure mode, (c) shared-core wording verified via the existing FO write-scope guardrail test, plus the empty-worktree, bare-mode, and no-migration cases. Test plan is mostly offline: the merge property is the load-bearing claim, and a fixture-level `git merge` + conflict-marker check proves it directly without burning live-model budget.
