---
id: "214"
title: "Stage worktree stickiness — once in a worktree, stay until terminal"
status: ideation
source: GitHub issue #104 (filed by CL). Originally surfaced via the spacedock-prompt/experiments workflow (template-simplification variants for FO, variant 015 tighten-hedging). Today's session (2026-04-30) extended scope: PR #176, #177, #180 each hit entity-body merge conflicts because the FO writes `### Feedback Cycles` to main while the worktree branch writes stage reports to the same trailing region — folded entity `k9s` (`feedback-cycles-on-worktree-not-main`, archived) into this task as a sub-scope on captain direction.
started: 2026-05-01T04:47:21Z
completed:
verdict:
score: 0.65
worktree:
issue: "#104"
pr:
mod-block:
---

# Clarify stage worktree semantics: post-worktree stickiness

## Problem statement

The workflow stage model lets each stage declare `worktree: true|false` (with a per-workflow default in `stages.defaults`). When a stage with `worktree: true` is followed by a stage that inherits `worktree: false`, the downstream stage strands its prior artifacts:

- Run-stage outputs (files written under the worktree, branches on a submodule's `experiment/{slug}` branch, uncommitted WIP) are only visible from within the worktree.
- A downstream `worktree: false` stage would nominally dispatch against main — where those artifacts do not exist until the terminal merge.

**Concrete example** (from the experiments workflow that surfaced this):

```yaml
stages:
  defaults:
    worktree: false
  states:
    - name: hypothesis
    - name: run
      worktree: true
    - name: analyze        # inherits worktree: false
      feedback-to: run
    - name: done
      terminal: true
```

- `run` writes `_results/{slug}.json` and commits template diffs on branch `spacedock-ensign/{slug}` with a matching `experiment/{slug}` submodule branch.
- `analyze` per README takes `_results/{slug}.json` as input and produces an adopt/reject/combine recommendation.
- But `analyze` inherits `worktree: false`, so the FO would dispatch it against main — where the results JSON and the submodule branch do not exist yet.
- The FO has to override the config to keep `analyze` inside the `run` worktree for it to work at all.
- If `analyze` rejects and `feedback-to: run` routes back, the feedback fix obviously needs to land in the same worktree — reinforcing that once an entity enters a worktree stage, it should not leave it until the terminal merge.

## FO contract intersection

The shared-core `## Completion and Gates` reuse conditions currently include:

> **Reuse conditions** (all must hold — if any fails, dispatch fresh):
> 3. Next stage has the same `worktree` mode as the completed stage

(`skills/first-officer/references/first-officer-shared-core.md:144` — verified against the file as of this ideation)

With stickiness, this condition is meaningless and must be reworded — once an entity is in a worktree, the downstream stage's declared `worktree:` value should not gate reuse or dispatch location. The replacement gates reuse on the *entity's* live worktree, not on the *stage's* declared mode.

The per-stage `worktree:` field also needs reinterpretation: it becomes "if the entity does not yet have a worktree, create one at this stage" rather than "run this stage in a worktree or on main."

The folded sub-scope (Feedback Cycles routing, see `## Folded sub-scope: Feedback Cycles routing`) is the second contract surface this entity touches: the FO Write Scope `### Feedback Cycles` bullet, the Feedback Rejection Flow ownership sentence, the Worktree Ownership body-content list, the FO Write Scope carve-out clarification, and the claude-runtime cooperative-shutdown-sweep parenthetical.

## Alternatives (from issue body)

### A — Stickiness by default

Once an entity has been dispatched to any stage with `worktree: true`, all subsequent non-terminal stages for that entity operate in the same worktree, regardless of their own `worktree` field. The `worktree:` field per stage then means "create a worktree if none exists yet" rather than "run in a worktree or on main." Terminal (`done`) stage retains existing merge-and-cleanup semantics.

Simplest. Matches current de-facto operational need. FO keeps `worktree:` on entity frontmatter across run → analyze, clears on terminal merge.

### B — Explicit per-stage opt-out

Introduce `worktree: inherit-from-previous` or similar. More verbose, lets a workflow opt-out of stickiness if needed. Probably YAGNI — can't think of a real reason to un-stick.

### C — Require downstream stages to redeclare `worktree: true`

Less surprising syntactically but fragile — any author forgetting to redeclare trips the same trap.

**CL's lean:** A. Captured here for ideation to confirm or push back with evidence.

## Proposed approach

### The invariant

Once an entity has been dispatched to any stage declared `worktree: true`, the FO stamps `worktree: .worktrees/{worker_key}-{slug}` on entity frontmatter and that value remains stable across every subsequent non-terminal advancement. The terminal stage's merge-and-cleanup flow is the only place `worktree:` clears. The per-stage `worktree:` field's meaning narrows to: *"if the entity does not yet have a worktree, create one when dispatching to this stage."* Stages with `worktree: false` (or inheriting it) no longer mean "run on main" — they mean "do not create a worktree on first dispatch." After the first worktree-creating stage, the entity's stamped `worktree:` field is the source of truth for dispatch routing, not the stage's declared mode.

### Locked contract wording

Each surface below names the current line in `skills/first-officer/references/first-officer-shared-core.md` (verified against the file at this ideation; line numbers may drift before implementation lands), the existing prose, and the exact replacement.

**1. Shared-core reuse condition #3 reword (line 144).**

Existing:

> 3. Next stage has the same `worktree` mode as the completed stage

Replacement:

> 3. Reuse-routing matches the entity's worktree state — if the entity has `worktree:` set, the next stage routes into that same worktree; if `worktree:` is empty and the next stage declares `worktree: true`, dispatch fresh so the new worktree's first agent is born inside it

Anchor phrase for tests: `Reuse-routing matches the entity's worktree state`.

**2. `## FO Write Scope` `### Feedback Cycles` bullet (line 218).**

Existing:

> - **`### Feedback Cycles` section** — in entity bodies, tracking rejection rounds

Replacement:

> - **`### Feedback Cycles` section** — in entity bodies, tracking rejection rounds. **When `worktree:` is set on the entity, the FO writes the cycle entry to the worktree copy of the entity file and commits on the worktree branch (the cycle entry then rides the next stage-report commit into the merge). When `worktree:` is empty, the FO writes to main.** Under stage-worktree stickiness, `worktree:` is empty only before the first worktree-creating dispatch.

Anchor phrase for tests: `When \`worktree:\` is set` (must appear inside the `### Feedback Cycles` bullet).

**3. `## Feedback Rejection Flow` ownership sentence (line 178).**

Existing:

> The first officer owns the `### Feedback Cycles` section and keeps it on the main branch.

Replacement:

> The first officer owns the `### Feedback Cycles` section. Routing follows FO Write Scope: worktree-side when `worktree:` is set, main-side otherwise.

Anchor phrase for tests: `worktree-side when \`worktree:\` is set, main-side otherwise`.

**4. `## Worktree Ownership` body-content list (line 206-210, list body).**

Existing list body (line 208):

> - For worktree-backed entities, active stage/status/report/body state lives in the worktree copy.

Replacement (replace the bullet, do not append a new section):

> - For worktree-backed entities, active stage/status/report/body state — including `### Feedback Cycles` entries — lives in the worktree copy.

Anchor phrase for tests: `including \`### Feedback Cycles\` entries`.

**5. `## FO Write Scope` "Entity body content beyond `### Feedback Cycles`" carve-out clarification (line 228).**

Existing:

> - **Entity body content** beyond `### Feedback Cycles` — stage reports, design content, implementation notes belong to dispatched workers

Replacement:

> - **Entity body content** beyond `### Feedback Cycles` — stage reports, design content, implementation notes belong to dispatched workers. The FO's `### Feedback Cycles` carve-out applies in the appropriate view (worktree copy when `worktree:` is set, main otherwise); other body content remains worker-only in either view.

Anchor phrase for tests: `applies in the appropriate view`.

**6. `claude-first-officer-runtime.md:155` view parenthetical.**

Existing (the cooperative-shutdown sweep exemption that reads `### Feedback Cycles` to detect active feedback state):

> Exempt any agent whose entity is in an active feedback-cycle state (tracked via a `### Feedback Cycles` subsection in the entity body). Those reviewers may hold load-bearing context from the prior cycle that re-dispatch cannot reconstruct. Sweep feedback-cycle reviewers only on explicit captain confirmation.

Replacement (insert one parenthetical):

> Exempt any agent whose entity is in an active feedback-cycle state (tracked via a `### Feedback Cycles` subsection in the entity body; read from the worktree copy when `worktree:` is set on the entity, otherwise from main). Those reviewers may hold load-bearing context from the prior cycle that re-dispatch cannot reconstruct. Sweep feedback-cycle reviewers only on explicit captain confirmation.

Anchor phrase for tests: `read from the worktree copy when \`worktree:\` is set`.

**7. Commission development README template (`skills/commission/references/templates/development.md`).**

Add one sentence to the per-stage-`worktree`-field explanation. The current template documents `worktree` in the entity-frontmatter table at line 59 and uses `worktree: true` on `implementation` / `validation` at lines 186, 188 without naming the stickiness rule. Add this sentence to the frontmatter table's `worktree` row description (line 59):

Existing:

> | `worktree` | string | Worktree path while a dispatched agent is active, empty otherwise |

Replacement:

> | `worktree` | string | Worktree path while a dispatched agent is active, empty otherwise. Once set on first dispatch into a `worktree: true` stage, it stays set across all non-terminal advancements (stickiness) and clears at terminal merge. |

Anchor phrase for tests: `Once set on first dispatch`.

The existing `experiment.md` and `refinement.md` templates have no `worktree: true` stages, so no parallel edit is required there; the stickiness rule applies trivially (no worktree ever exists, no behavior change).

### Resolved design questions

**(1) Interaction with `fresh: true`.** Stickiness is worktree-lifecycle; `fresh` is agent-lifecycle — independent axes. When stage B declares `fresh: true` and the entity is in a worktree from stage A, the correct behavior is: dispatch a fresh agent *into the same worktree*. The reuse-condition reword (surface 1 above) handles this by routing on `worktree:` rather than on stage mode; the `fresh: true` path is unchanged from today (it always shuts down the prior agent and dispatches a new one). No additional contract surface is needed beyond the reword.

**(2) Interaction with `feedback-to`.** The existing Feedback Rejection Flow already keeps the fix agent in the same worktree — stickiness merely makes this rule uniform across all advancement paths, not just rejection-routing. The folded sub-scope (Feedback Cycles routing, surfaces 2–6 above) is the only feedback-related contract change; the rejection-flow control flow itself does not change.

**(3) Per-stage `worktree:` field meaning post-change.** A workflow with `worktree: true` only on a middle stage means "entity runs on main until reaching this stage; from here on, in a worktree until terminal." A workflow with `worktree: true` on the initial stage means "always in a worktree from first dispatch." A workflow with `worktree: true` nowhere means "always on main." This is the sole behavioral semantics — the field gates *worktree creation*, not per-stage routing.

**(4) Documentation surfaces.** Covered by the locked contract wording above (surfaces 1–7). No additional doc surfaces require edits — the FO shared-core and runtime adapters carry the canonical rule, and the commission development README template carries the captain-facing summary.

**(5) Migration for existing workflows.** All existing in-repo workflows use `worktree: true` either on a contiguous run (`implementation` + `validation`) or everywhere or nowhere. Stickiness does not change their behavior because the run of `worktree: true` stages is already contiguous and bounded by the terminal stage. The folded Feedback Cycles routing rule does change the *write target* for any cycle entries appended during a worktree-backed stage, but mid-flight entities with cycle entries already on main keep those entries on main — the new rule applies to subsequent appends only, no rewrite. This matches the no-migration policy already documented in the archived `k9s` entity. The only affected workflows are external ones with non-contiguous patterns (the spacedock-prompt/experiments workflow that motivated this entity), and those are affected positively.

### Folded-sub-scope reconciliation

The Feedback Cycles routing rule (folded from entity `k9s`) is a corollary of stickiness, not an independent change. Surfaces 2–6 of the locked contract wording are the folded sub-scope; surface 1 is the stickiness invariant proper. The `## Folded sub-scope: Feedback Cycles routing` section below is unchanged — it is the load-bearing input from the cycle-2 design rationale and this `## Proposed approach` builds on it rather than re-stating it.

## Acceptance criteria

Each AC is an entity-level end-state property with a `Verified by:` clause naming a specific test file and test shape.

**AC-1 — Shared-core reuse condition #3 names stickiness routing with a structural anchor.**
The assembled FO agent content contains the literal substring `Reuse-routing matches the entity's worktree state` inside the `**Reuse conditions**` block, and does NOT contain the prior phrase `Next stage has the same \`worktree\` mode as the completed stage`. Verified by: extending `tests/test_repo_edit_guardrail.py`'s static-content phase (which already greps assembled FO content for FO Write Scope items) with two literal-substring assertions — one positive on the new anchor inside the Reuse conditions block (extracted by anchored regex around `**Reuse conditions**`), one negative on the removed phrase anywhere in the assembled text.

**AC-2 — Entity `worktree:` frontmatter persists across non-terminal advancements once stamped.**
For an entity with `worktree: .worktrees/{path}` set, calling `status --set {slug} status={next_non_terminal_stage}` without passing `worktree=` leaves the `worktree:` field unchanged; calling `status --archive {slug}` (or terminal `status --set` with `worktree=`) clears it. Verified by: a new offline test `tests/test_worktree_stickiness.py` that builds a workflow fixture with `initial → middle (worktree: true) → terminal` stages, drives `status --set` through the stages, and asserts the `worktree:` field remains stable across the inherited-default downstream stage and clears at terminal.

**AC-3 — Commission development README template documents stickiness in the `worktree` frontmatter row.**
The file `skills/commission/references/templates/development.md` contains the literal substring `Once set on first dispatch` in the `worktree` row of the frontmatter table. Verified by: extending `tests/test_commission_template.py` with one literal-substring assertion against the rendered template.

**AC-4 — `claude-team build` prompt assembly routes to the entity's stamped worktree, not the next stage's declared mode.**
For an entity with `worktree: .worktrees/{path}` set, when the next stage declares `worktree: false` (or inherits the workflow default of false), the `claude-team build` prompt names the worktree path as the dispatch target — matching what the FO would emit when re-using the worktree under stickiness. Verified by: extending `tests/test_dispatch_names.py` (or, if its scope does not naturally cover prompt-target paths, a new `tests/test_worktree_stickiness_dispatch.py`) with a prompt-assembly assertion: build a fixture entity in a worktree, invoke `claude-team build` for a downstream `worktree: false`-declared stage, assert the assembled prompt contains the worktree path string.

**AC-5 — When `worktree:` is set on an entity, the FO writes new `### Feedback Cycles` entries inside the worktree copy and commits on the worktree branch; the main copy of the entity file is untouched until PR merge.**
After the FO routes feedback back to implementation on a worktree-backed entity: `git -C {worktree_path} show HEAD:{workflow_dir}/{slug}.md` contains the new cycle entry; `git show main:{workflow_dir}/{slug}.md` does not. Verified by: extending `tests/test_rejection_flow.py` with the two split assertions (live-claude tier already wired for that test).

**AC-6 — Two consecutive feedback cycles on a worktree-backed entity merge into main with zero conflicts in the entity file, and a control case proves the test exercises the actual PR #176/#177 conflict shape.**
A treatment fixture (worktree-side cycle-2 entry placed line-adjacent to the cycle-1 stage-report region) merges via `git merge --no-ff` with exit code 0 and zero `<<<<<<<` markers in the entity file. A control fixture (cycle-2 entry written on main, old rule) reproduces the conflict — `git merge` either fails or leaves conflict markers in the entity file. Verified by: a new offline test `tests/test_feedback_cycles_merge_clean.py` plus a fixture under `tests/fixtures/feedback-cycles-merge/` carrying both cases.

**AC-7 — Shared-core wording for the Feedback Cycles routing rule has structural anchors readers and tests can rely on.**
The assembled FO agent content contains the literal substring `When \`worktree:\` is set` inside the FO Write Scope `### Feedback Cycles` bullet's surrounding paragraph (extracted by anchored regex around the bullet, not free-grepped over the whole file), and contains the literal substring `worktree-side when \`worktree:\` is set, main-side otherwise` in the Feedback Rejection Flow section. Verified by: extending `tests/test_repo_edit_guardrail.py`'s static-content phase with two literal-substring assertions on the extracted blocks.

## Out of scope

- Runtime support for git submodules inside worktrees (the experiment workflow's pattern). This entity is about FO contract semantics, not submodule plumbing.
- Changing the archive flow's worktree-cleanup behavior.
- Introducing new `worktree:` field values beyond `true|false`. Alternative B in the issue body is rejected as YAGNI.
- Migrating existing in-repo workflows (none have the non-contiguous pattern).

## Test plan

| AC | Test surface | Tier | Approx cost |
|----|--------------|------|-------------|
| AC-1 | Extend `tests/test_repo_edit_guardrail.py` static-content phase: positive substring on new reuse-condition anchor, negative substring on removed phrase | offline (assembled-agent content check) | seconds; no live model |
| AC-2 | New offline test `tests/test_worktree_stickiness.py` driving `status --set` through `initial → middle (worktree: true) → terminal` | offline (`make test-static`) | seconds; no live model |
| AC-3 | Extend `tests/test_commission_template.py` with one literal-substring assertion on the rendered development template | offline | seconds |
| AC-4 | Extend `tests/test_dispatch_names.py` (or new `tests/test_worktree_stickiness_dispatch.py`) with one prompt-target assertion | offline | seconds |
| AC-5 | Extend `tests/test_rejection_flow.py` with worktree-vs-main split assertions on the cycle entry | live-claude (existing tier) | adds ~0 budget — same FO run, two extra assertions |
| AC-6 | New offline test `tests/test_feedback_cycles_merge_clean.py` + fixture under `tests/fixtures/feedback-cycles-merge/` (treatment + control) | offline | seconds |
| AC-7 | Extend `tests/test_repo_edit_guardrail.py` static-content phase with two literal-substring assertions on extracted blocks | offline | seconds |

Six of seven ACs are offline; only AC-5 burns live-claude budget, and it folds into the existing rejection-flow live test. The offline merge-clean fixture (AC-6) is the load-bearing proof for the Feedback Cycles half — it directly exercises the merge property the change targets, with a control case that fails the old way. Static checks alone would not have caught PR #176/#177 (the pain is mechanical line-overlap), so the fixture-level `git merge` + conflict-marker check is the exact-abstraction proof.

Estimated total cost: low-to-medium. One shared-core edit + one runtime-adapter parenthetical + one commission-template edit + four test edits/additions + one fixture directory. 60–90 min implementation. No live CI matrix beyond the existing `tests/test_rejection_flow.py` tier.

## Cross-references

- GitHub issue #104
- `skills/first-officer/references/first-officer-shared-core.md:112` — reuse condition #3 (the rule that needs rewording)
- `skills/commission/bin/claude-team::cmd_build` — dispatch prompt assembly; check whether it reads entity `worktree:` field or re-derives from stage config
- `skills/commission/bin/status::_set` + `_archive` — frontmatter field preservation logic
- External workflow `spacedock-prompt/experiments` — originating use case (not in this repo)
- **Folded sub-scope (entity `k9s` archived 2026-04-30):** `### Feedback Cycles` routing under stickiness — see `## Folded sub-scope: Feedback Cycles routing` below.

## Folded sub-scope: Feedback Cycles routing

Folded from entity `k9s` (`feedback-cycles-on-worktree-not-main`) on 2026-04-30 per captain direction after the cycle-2 ideation gate. PR #176 / #177 / #180 in this session each hit a merge conflict in the entity body because the FO writes `### Feedback Cycles` to main while the worktree branch writes stage reports to the same trailing region.

Under the stickiness invariant from this entity, the cycle-routing rule becomes a one-line corollary: cycle entries are entity-body content, entity-body content during in-flight stages lives where stage reports live, stage reports live in the worktree under stickiness — therefore cycle entries live in the worktree while `worktree:` is set, and on main only before the first worktree-creating dispatch. The terminal merge brings the worktree-side entries onto main.

Concrete shared-core changes folded in (in addition to the reuse-condition #3 reword named in `## FO contract intersection`):

- **`## FO Write Scope`** (`first-officer-shared-core.md:218`) — replace the unconditional `### Feedback Cycles` bullet with a sub-clause naming both states: `When \`worktree:\` is set, the FO writes the cycle entry to the worktree copy and commits on the worktree branch; when \`worktree:\` is empty, the FO writes to main.` Anchor phrase: `When \`worktree:\` is set`.
- **`## Feedback Rejection Flow`** (`first-officer-shared-core.md:178`) — replace `keeps it on the main branch` with `Routing follows FO Write Scope: worktree-side when \`worktree:\` is set, main-side otherwise.`
- **`## Worktree Ownership`** (`first-officer-shared-core.md:206`) — add `### Feedback Cycles` to the list of body content that stays in the worktree during in-flight stages.
- **`## FO Write Scope` "Entity body content beyond `### Feedback Cycles`"** (`first-officer-shared-core.md:228`) — clarify the carve-out applies in the appropriate view.
- **`claude-first-officer-runtime.md:155`** (cooperative-shutdown sweep / cycle-detection reader) — add a one-line view-clarification parenthetical so the reader operates on the worktree copy when `worktree:` is set.

Folded ACs (renumber as part of `21`'s final AC list during `21`'s ideation):

- **AC (Feedback Cycles, worktree-side write):** When `worktree:` is set on an entity, the FO writes new `### Feedback Cycles` entries inside the worktree copy of the entity file and commits on the worktree branch; the main copy of the entity file is untouched until PR merge. Verified by extending `tests/test_rejection_flow.py` with two assertions after the FO routes feedback back to implementation: `git -C {worktree_path} show HEAD:{workflow_dir}/{slug}.md` contains the new cycle entry; `git show main:{workflow_dir}/{slug}.md` does not contain it.
- **AC (Feedback Cycles, merge-clean with control case):** Two consecutive feedback cycles on a worktree-backed entity merge into main with zero conflicts in the entity file, and a control case proves the test exercises the actual PR #176/#177 conflict shape. Verified by a new offline test `tests/test_feedback_cycles_merge_clean.py` plus a fixture under `tests/fixtures/feedback-cycles-merge/`. Treatment case (worktree-side cycle entry placed line-adjacent to stage-report region) merges clean (`git merge` exit 0, zero `<<<<<<<` markers). Control case (cycle entry on main, old rule) reproduces the conflict — proving the test is not passing because of benign whitespace separation.
- **AC (Feedback Cycles, shared-core wording with structural anchors):** `first-officer-shared-core.md` describes the conditional routing rule with anchors readers and tests can rely on. Verified by extending `tests/test_repo_edit_guardrail.py` static-content phase with two literal-substring assertions: `When \`worktree:\` is set` inside the FO Write Scope `### Feedback Cycles` bullet's surrounding paragraph; `worktree-side when \`worktree:\` is set, main-side otherwise` in the Feedback Rejection Flow section.

The standalone-shipping form of `k9s` (with additional ACs for empty-worktree, bare-mode equivalence, no-migration policy) is preserved in the archived entity body for reference; under stickiness those three ACs collapse and are not folded in.

The full cycle-2 design rationale (architectural answers, findings-under-stickiness, test plan) lives in `docs/plans/_archive/feedback-cycles-on-worktree-not-main.md`. The ideation worker on `21` should use it as load-bearing input for the Feedback Cycles portion of `21`'s eventual design.

## Summary

Today the FO has to work around a gap: once an entity runs a `worktree: true` stage, the next inherited-default stage lies to the operator by claiming it should run on main, even though its inputs are in the worktree. The proposed fix (Option A, stickiness by default) makes the implicit contract explicit, simplifies the reuse-condition logic, and matches CL's operational workaround. Ideation has locked exact contract wording for seven surfaces (one shared-core reuse-condition reword, four shared-core Feedback Cycles edits, one runtime-adapter parenthetical, one commission-template row), resolved the five design questions, and consolidated seven entity-level ACs (six offline-verifiable, one folded into the existing live rejection-flow test).

## Stage Report: ideation

- DONE: Lock the contract wording. For each shared-core / runtime / commission-skill surface that the entity body or its folded sub-scope names, propose the exact replacement prose: shared-core reuse condition #3 reword (line 112), `## FO Write Scope` `### Feedback Cycles` sub-clause (line 218), `## Feedback Rejection Flow` ownership sentence (line 178), `## Worktree Ownership` body-content list addition (line 206), `## FO Write Scope` carve-out clarification (line 228), `claude-first-officer-runtime.md:155` view parenthetical, plus any new sentence needed in the workflow README template. Cite line numbers to the current shared-core file (verify before quoting).
  Seven surfaces locked in `## Proposed approach` → `### Locked contract wording`. Verified line numbers against `skills/first-officer/references/first-officer-shared-core.md` as of this ideation: reuse condition #3 is at line 144 (entity body cited 112; corrected in `## FO contract intersection`), other line numbers match the entity body's citations. Commission-skill surface lands in `skills/commission/references/templates/development.md` line 59 (the `worktree` row of the frontmatter table).
- DONE: Resolve the four design questions in the entity body — (1) `fresh: true` interaction, (2) `feedback-to` interaction, (3) per-stage `worktree:` field meaning post-change, (5) migration — and reconcile with the folded sub-scope's design (Feedback Cycles routing as a corollary of stickiness). Question (4) is documentation, addressed by step 1 above. Each answer should land as a paragraph or sub-section in `## Proposed approach` (which the entity does not yet have — add it).
  `## Proposed approach` added with `### The invariant`, `### Locked contract wording`, `### Resolved design questions` (1-5), and `### Folded-sub-scope reconciliation`. (1) `fresh: true` is agent-lifecycle and orthogonal — fresh agent into same worktree. (2) `feedback-to` already keeps fix in worktree; stickiness uniformizes. (3) per-stage `worktree:` field gates worktree creation, not routing. (5) no migration needed — existing in-repo workflows have contiguous worktree-true runs; mid-flight cycle entries on main stay there.
- DONE: Refine the AC list. Merge the existing draft AC-1..AC-5 with the 3 folded ACs (worktree-side write, merge-clean with control case, structural-anchor wording) into a single canonical list, renumbered. Each AC must be an entity-level end-state property with a `Verified by:` clause naming a specific test file + test shape. Tighten any AC that reads as a stage action (`Run X` / `Produce Y`) into the property it produces. Replace AC-1's grep-only check with a structural anchor like the cycle-2 design used (`When \`worktree:\` is set` substring, etc.).
  `## Acceptance criteria` consolidated into 7 ACs: AC-1 (reuse-condition anchor + negative-grep on removed phrase), AC-2 (worktree frontmatter persistence), AC-3 (commission template stickiness sentence), AC-4 (claude-team build prompt routes to worktree), AC-5 (Feedback Cycles worktree-side write — folded), AC-6 (merge-clean with control case — folded), AC-7 (shared-core anchor wording — folded). Test plan table rewritten to map each AC to its test surface, tier, and cost. AC-1 now uses the literal-substring anchor `Reuse-routing matches the entity's worktree state` plus a negative assertion on the removed phrase.

### Summary

Locked the FO contract surfaces for stickiness as a single canonical design: one shared-core reuse-condition reword (the stickiness invariant proper) plus four shared-core edits and one runtime-adapter parenthetical for the folded Feedback Cycles routing sub-scope, plus one commission-template row update for captain-facing documentation. Resolved all five design questions with concrete contract decisions — `fresh: true` and `feedback-to` are orthogonal axes that need no new wording, the per-stage `worktree:` field gates worktree creation only, and existing in-repo workflows need no migration. The AC list consolidates seven entity-level end-state properties with structural anchors (replacing the cycle-1 grep-only check); six are offline-verifiable, one folds into the existing live rejection-flow test.
