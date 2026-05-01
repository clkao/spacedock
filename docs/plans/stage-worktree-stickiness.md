---
id: "214"
title: "Stage worktree stickiness — once in a worktree, stay until terminal"
status: implementation
source: GitHub issue #104 (filed by CL). Originally surfaced via the spacedock-prompt/experiments workflow (template-simplification variants for FO, variant 015 tighten-hedging). Today's session (2026-04-30) extended scope: PR #176, #177, #180 each hit entity-body merge conflicts because the FO writes `### Feedback Cycles` to main while the worktree branch writes stage reports to the same trailing region — folded entity `k9s` (`feedback-cycles-on-worktree-not-main`, archived) into this task as a sub-scope on captain direction.
started: 2026-05-01T04:47:21Z
completed:
verdict:
score: 0.65
worktree: .worktrees/spacedock-ensign-stage-worktree-stickiness
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

The same one-sentence addition propagates to the parallel `worktree` rows in `skills/commission/references/templates/experiment.md` (line 84) and `skills/commission/references/templates/refinement.md` (line 53). Stickiness is a no-op in those templates today (neither has a `worktree: true` stage in its default scaffolding), but the contract documentation should describe the field uniformly across templates so a captain who later adds a `worktree: true` stage to either template inherits the correct mental model. The same `Once set on first dispatch` anchor phrase covers all three templates.

**8. Codex runtime adapter (`skills/first-officer/references/codex-first-officer-runtime.md`).**

Verified line 84 of the file says verbatim:

> 3. Create the worktree only when the stage definition says `worktree: true`. If the stage is not marked for a worktree, stay on the main branch.

This contradicts stickiness — under the new rule, dispatch routing keys off the entity's stamped `worktree:` field, not the next stage's declared mode. Replacement:

> 3. Create the worktree only when the entity does not yet have one stamped on its frontmatter and the stage definition says `worktree: true`. If the entity's frontmatter already has `worktree:` set, route the dispatch into that existing worktree regardless of the next stage's declared mode (stickiness — the entity remains in the same worktree until terminal merge). If the entity has no stamped worktree and the stage is not marked `worktree: true`, stay on the main branch.

Anchor phrase for tests: `route the dispatch into that existing worktree`.

This surface is symmetric with shared-core surface 1 — the codex adapter carries its own dispatch instructions because it spawns workers via `spawn_agent` rather than via Claude's `Agent()` tool, and its prose is read by the codex FO runtime independently of the shared-core. AC-1 covers the shared-core anchor only; AC-1b (added below) covers the codex-runtime anchor.

### Resolved design questions

**(1) Interaction with `fresh: true`.** Stickiness is worktree-lifecycle; `fresh` is agent-lifecycle — independent axes. When stage B declares `fresh: true` and the entity is in a worktree from stage A, the correct behavior is: dispatch a fresh agent *into the same worktree*. The reuse-condition reword (surface 1 above) handles this by routing on `worktree:` rather than on stage mode; the `fresh: true` path is unchanged from today (it always shuts down the prior agent and dispatches a new one). No additional contract surface is needed beyond the reword.

**(2) Interaction with `feedback-to`.** The existing Feedback Rejection Flow already keeps the fix agent in the same worktree — stickiness merely makes this rule uniform across all advancement paths, not just rejection-routing. The folded sub-scope (Feedback Cycles routing, surfaces 2–6 above) is the only feedback-related contract change; the rejection-flow control flow itself does not change.

**(3) Per-stage `worktree:` field meaning post-change.** A workflow with `worktree: true` only on a middle stage means "entity runs on main until reaching this stage; from here on, in a worktree until terminal." A workflow with `worktree: true` on the initial stage means "always in a worktree from first dispatch." A workflow with `worktree: true` nowhere means "always on main." This is the sole behavioral semantics — the field gates *worktree creation*, not per-stage routing. Corollary: once an entity has a worktree stamped, any *subsequent* stage's `worktree: true` declaration is a no-op for that entity — the worktree already exists and the dispatch routes into it. A captain debugging a workflow should expect this asymmetry: redeclaring `worktree: true` on stage N+1 has no observable effect when stage N already created the worktree, but it would create a worktree if stage N+1 were the first to demand one (e.g., a workflow with `backlog → ideation → analysis (worktree: true)`).

**(4) Documentation surfaces.** Covered by the locked contract wording above (surfaces 1–7). No additional doc surfaces require edits — the FO shared-core and runtime adapters carry the canonical rule, and the commission development README template carries the captain-facing summary.

**(5) Migration for existing workflows.** All existing in-repo workflows use `worktree: true` either on a contiguous run (`implementation` + `validation`) or everywhere or nowhere. Stickiness does not change their behavior because the run of `worktree: true` stages is already contiguous and bounded by the terminal stage. The folded Feedback Cycles routing rule does change the *write target* for any cycle entries appended during a worktree-backed stage, but mid-flight entities with cycle entries already on main keep those entries on main — the new rule applies to subsequent appends only, no rewrite. This matches the no-migration policy already documented in the archived `k9s` entity. The only affected workflows are external ones with non-contiguous patterns (the spacedock-prompt/experiments workflow that motivated this entity), and those are affected positively.

### Orphan worktree handling

The stickiness invariant is `worktree:` set ↔ live worktree on disk. A `worktree:` field that points at a deleted or missing path is an *orphan* state, and orphan handling is delegated to the existing boot-time orphan-detection in `status --boot` (which already cross-references entities with `worktree:` fields against filesystem and git state, surfacing anomalies for FO review — see `first-officer-shared-core.md` line 19, the `ORPHANS` section of the `--boot` output contract). This entity does not introduce a new orphan-recovery path: stickiness preserves the existing invariant, so the same boot-time detection catches the same failure modes. A defensive guard ("if a worktree-side write fails because the path is missing, fall back to main and log") is fine to add as a one-liner during implementation but is not load-bearing for the design — the fall-through behavior is the existing FO error-reporting path, not a new contract surface.

### Folded-sub-scope reconciliation

The Feedback Cycles routing rule (folded from entity `k9s`) is a corollary of stickiness, not an independent change. Surfaces 2–6 of the locked contract wording are the folded sub-scope; surface 1 is the stickiness invariant proper. The `## Folded sub-scope: Feedback Cycles routing` section below is unchanged — it is the load-bearing input from the cycle-2 design rationale and this `## Proposed approach` builds on it rather than re-stating it.

## Acceptance criteria

Each AC is an entity-level end-state property with a `Verified by:` clause naming a specific test file and test shape.

**AC-1 — Shared-core reuse condition #3 names stickiness routing with a structural anchor.**
The assembled FO agent content contains the literal substring `Reuse-routing matches the entity's worktree state` inside the `**Reuse conditions**` block, and does NOT contain the prior phrase `Next stage has the same \`worktree\` mode as the completed stage`. Verified by: extending `tests/test_repo_edit_guardrail.py`'s static-content phase (which already greps assembled FO content for FO Write Scope items) with two literal-substring assertions — one positive on the new anchor inside the Reuse conditions block (extracted by anchored regex around `**Reuse conditions**`), one negative on the removed phrase anywhere in the assembled text.

**AC-1b — Codex runtime adapter names stickiness routing with a structural anchor.**
The file `skills/first-officer/references/codex-first-officer-runtime.md` contains the literal substring `route the dispatch into that existing worktree` inside the per-dispatch numbered list (the section that today carries the contradicting "stay on the main branch" prose at line 84), and does NOT contain the prior phrase `If the stage is not marked for a worktree, stay on the main branch`. Verified by: a new offline test `tests/test_codex_runtime_stickiness.py` (or extending an existing codex-runtime static-content test if one exists in the same family as `test_repo_edit_guardrail.py`) that reads the file and asserts both the positive and negative substrings.

**AC-2 — Entity `worktree:` frontmatter persists across non-terminal advancements once stamped, including across `fresh: true` stages.**
For an entity with `worktree: .worktrees/{path}` set, calling `status --set {slug} status={next_non_terminal_stage}` without passing `worktree=` leaves the `worktree:` field unchanged; calling `status --archive {slug}` (or terminal `status --set` with `worktree=`) clears it. The fixture must include the canonical in-repo development-template path: `initial → implementation (worktree: true) → validation (worktree: true, fresh: true) → terminal`. The assertion must include that the `worktree:` field has the same value after the implementation→validation advancement as before — i.e., `fresh: true` does NOT cause a new worktree to be stamped or the existing one to be cleared. Verified by: a new offline test `tests/test_worktree_stickiness.py` that builds two workflow fixtures — (a) `initial → middle (worktree: true) → terminal` (the inherited-default-downstream case) and (b) `initial → implementation (worktree: true) → validation (worktree: true, fresh: true) → terminal` (the canonical development-template case) — drives `status --set` through each, and asserts the `worktree:` field remains stable through all non-terminal advancements and clears at terminal in both fixtures.

**AC-3 — Commission templates document stickiness in the `worktree` frontmatter row.**
All three commission templates (`skills/commission/references/templates/development.md`, `experiment.md`, `refinement.md`) contain the literal substring `Once set on first dispatch` in the `worktree` row of their frontmatter tables. Verified by: a new offline test `tests/test_development_template.py` (which loads each template file directly and asserts the substring is present in the `worktree` row, extracted by anchored regex around the `| \`worktree\` |` table cell). The existing `tests/test_commission_template.py` is not extended because it scopes to `skills/commission/SKILL.md`, not the templates directory; the new test file is the natural home for template-content assertions.

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
| AC-1b | New offline test `tests/test_codex_runtime_stickiness.py` reads `codex-first-officer-runtime.md`: positive substring on `route the dispatch into that existing worktree`, negative substring on the removed prose | offline | seconds |
| AC-2 | New offline test `tests/test_worktree_stickiness.py` driving `status --set` through two fixtures — `initial → middle (worktree: true) → terminal` and `initial → implementation (worktree: true) → validation (worktree: true, fresh: true) → terminal` | offline (`make test-static`) | seconds; no live model |
| AC-3 | New offline test `tests/test_development_template.py` reading all three commission templates and asserting the `Once set on first dispatch` substring inside each `worktree` row | offline | seconds |
| AC-4 | Extend `tests/test_dispatch_names.py` (or new `tests/test_worktree_stickiness_dispatch.py`) with one prompt-target assertion | offline | seconds |
| AC-5 | Extend `tests/test_rejection_flow.py` with worktree-vs-main split assertions on the cycle entry | live-claude (existing tier) | adds ~0 budget — same FO run, two extra assertions |
| AC-6 | New offline test `tests/test_feedback_cycles_merge_clean.py` + fixture under `tests/fixtures/feedback-cycles-merge/` (treatment + control) | offline | seconds |
| AC-7 | Extend `tests/test_repo_edit_guardrail.py` static-content phase with two literal-substring assertions on extracted blocks | offline | seconds |

Seven of eight ACs are offline; only AC-5 burns live-claude budget, and it folds into the existing rejection-flow live test. The offline merge-clean fixture (AC-6) is the load-bearing proof for the Feedback Cycles half — it directly exercises the merge property the change targets, with a control case that fails the old way. Static checks alone would not have caught PR #176/#177 (the pain is mechanical line-overlap), so the fixture-level `git merge` + conflict-marker check is the exact-abstraction proof.

Estimated total cost: low-to-medium. One shared-core edit (reuse condition + four Feedback Cycles edits) + one codex-runtime edit + one claude-runtime parenthetical + three commission-template edits (one per template) + five test edits/additions (`test_repo_edit_guardrail`, `test_codex_runtime_stickiness`, `test_worktree_stickiness`, `test_development_template`, `test_dispatch_names`) + one extension to `test_rejection_flow.py` + one fixture directory. 60–90 min implementation. No live CI matrix beyond the existing `tests/test_rejection_flow.py` tier.

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

Today the FO has to work around a gap: once an entity runs a `worktree: true` stage, the next inherited-default stage lies to the operator by claiming it should run on main, even though its inputs are in the worktree. The proposed fix (Option A, stickiness by default) makes the implicit contract explicit, simplifies the reuse-condition logic, and matches CL's operational workaround. Ideation has locked exact contract wording for eight surfaces (one shared-core reuse-condition reword, four shared-core Feedback Cycles edits, one claude-runtime parenthetical, one codex-runtime reword, three commission-template row updates — one each for development/experiment/refinement), resolved the five design questions plus orphan-handling delegation, and consolidated eight entity-level ACs (seven offline-verifiable, one folded into the existing live rejection-flow test).

## Stage Report: ideation

- DONE: Lock the contract wording. For each shared-core / runtime / commission-skill surface that the entity body or its folded sub-scope names, propose the exact replacement prose: shared-core reuse condition #3 reword (line 112), `## FO Write Scope` `### Feedback Cycles` sub-clause (line 218), `## Feedback Rejection Flow` ownership sentence (line 178), `## Worktree Ownership` body-content list addition (line 206), `## FO Write Scope` carve-out clarification (line 228), `claude-first-officer-runtime.md:155` view parenthetical, plus any new sentence needed in the workflow README template. Cite line numbers to the current shared-core file (verify before quoting).
  Seven surfaces locked in `## Proposed approach` → `### Locked contract wording`. Verified line numbers against `skills/first-officer/references/first-officer-shared-core.md` as of this ideation: reuse condition #3 is at line 144 (entity body cited 112; corrected in `## FO contract intersection`), other line numbers match the entity body's citations. Commission-skill surface lands in `skills/commission/references/templates/development.md` line 59 (the `worktree` row of the frontmatter table).
- DONE: Resolve the four design questions in the entity body — (1) `fresh: true` interaction, (2) `feedback-to` interaction, (3) per-stage `worktree:` field meaning post-change, (5) migration — and reconcile with the folded sub-scope's design (Feedback Cycles routing as a corollary of stickiness). Question (4) is documentation, addressed by step 1 above. Each answer should land as a paragraph or sub-section in `## Proposed approach` (which the entity does not yet have — add it).
  `## Proposed approach` added with `### The invariant`, `### Locked contract wording`, `### Resolved design questions` (1-5), and `### Folded-sub-scope reconciliation`. (1) `fresh: true` is agent-lifecycle and orthogonal — fresh agent into same worktree. (2) `feedback-to` already keeps fix in worktree; stickiness uniformizes. (3) per-stage `worktree:` field gates worktree creation, not routing. (5) no migration needed — existing in-repo workflows have contiguous worktree-true runs; mid-flight cycle entries on main stay there.
- DONE: Refine the AC list. Merge the existing draft AC-1..AC-5 with the 3 folded ACs (worktree-side write, merge-clean with control case, structural-anchor wording) into a single canonical list, renumbered. Each AC must be an entity-level end-state property with a `Verified by:` clause naming a specific test file + test shape. Tighten any AC that reads as a stage action (`Run X` / `Produce Y`) into the property it produces. Replace AC-1's grep-only check with a structural anchor like the cycle-2 design used (`When \`worktree:\` is set` substring, etc.).
  `## Acceptance criteria` consolidated into 7 ACs: AC-1 (reuse-condition anchor + negative-grep on removed phrase), AC-2 (worktree frontmatter persistence), AC-3 (commission template stickiness sentence), AC-4 (claude-team build prompt routes to worktree), AC-5 (Feedback Cycles worktree-side write — folded), AC-6 (merge-clean with control case — folded), AC-7 (shared-core anchor wording — folded). Test plan table rewritten to map each AC to its test surface, tier, and cost. AC-1 now uses the literal-substring anchor `Reuse-routing matches the entity's worktree state` plus a negative assertion on the removed phrase.

### Summary

Locked the FO contract surfaces for stickiness as a single canonical design: one shared-core reuse-condition reword (the stickiness invariant proper) plus four shared-core edits and one runtime-adapter parenthetical for the folded Feedback Cycles routing sub-scope, plus one commission-template row update for captain-facing documentation. Resolved all five design questions with concrete contract decisions — `fresh: true` and `feedback-to` are orthogonal axes that need no new wording, the per-stage `worktree:` field gates worktree creation only, and existing in-repo workflows need no migration. The AC list consolidates seven entity-level end-state properties with structural anchors (replacing the cycle-1 grep-only check); six are offline-verifiable, one folds into the existing live rejection-flow test.

### Feedback Cycles

- **Cycle 1 → 2 (2026-04-30):** Captain rejected the cycle-1 ideation gate. Staff review surfaced 3 blocking issues plus 3 minor findings: (1, blocking) AC-3 targeted `tests/test_commission_template.py` which scopes to `SKILL.md` not the templates directory, so the assertion would never run against the development template; (2, blocking) the codex runtime adapter at `codex-first-officer-runtime.md:84` carries a clause that contradicts stickiness ("If the stage is not marked for a worktree, stay on the main branch") and was missed by the cycle-1 surface inventory; (3, blocking) AC-2's fixture covered the inherited-default-downstream case but missed the canonical in-repo template path of `implementation (worktree: true) → validation (worktree: true, fresh: true)` — the most common real-world stickiness case; (4, minor) `experiment.md` and `refinement.md` carry the same `worktree` row wording and would go stale if not propagated; (5, minor) the redeclared-`worktree: true`-is-no-op consequence was implicit but not stated; (6, minor) orphan-`worktree:`-on-deleted-path delegation to `status --boot` was assumed but not restated. Cycle-2 response: retargeted AC-3 to a new `tests/test_development_template.py` covering all three templates (1); added surface 8 (codex runtime reword) with verified line 84 prose, the replacement, and AC-1b for assertion coverage (2); extended AC-2's fixture to include the `fresh: true` second worktree-backed stage, with an explicit assertion that `fresh: true` does not cause a new worktree to be stamped or the existing one to be cleared (3); propagated the `Once set on first dispatch` sentence to all three templates with AC-3 covering all three (4); added an explicit corollary sentence to Resolved Q (3) about redeclared-`worktree: true` being a no-op once sticky (5); added a new `### Orphan worktree handling` subsection delegating to existing `status --boot` orphan detection (6). This entity has `worktree:` empty (it has never been dispatched to a worktree-backed stage), so this cycle entry is being written on `main` — the architecturally correct location under both old and new rules.

## Stage Report: ideation (cycle 2)

- DONE: Address blocking finding (1) — AC-3 targets the wrong test file.
  AC-3 retargeted from extending `tests/test_commission_template.py` (which scopes to `SKILL.md`) to a new offline test `tests/test_development_template.py` that loads each commission template directly and asserts the `Once set on first dispatch` substring inside the `worktree` row, extracted by anchored regex around the `| \`worktree\` |` table cell. AC-3 now covers all three templates (development, experiment, refinement) — see also finding (4).
- DONE: Address blocking finding (2) — codex runtime contradicts stickiness, add 8th surface.
  Surface 8 added in `### Locked contract wording`. Verified `codex-first-officer-runtime.md:84` carries the existing prose verbatim (`If the stage is not marked for a worktree, stay on the main branch`). Replacement prose introduces stickiness routing into the per-dispatch numbered list with anchor `route the dispatch into that existing worktree`. AC-1b added for codex-runtime assertion coverage with positive substring on the new anchor and negative substring on the removed prose.
- DONE: Address blocking finding (3) — AC-2 misses the `fresh: true` after stickiness path.
  AC-2 extended to require two fixtures: (a) the original inherited-default-downstream case `initial → middle (worktree: true) → terminal`, and (b) the canonical development-template case `initial → implementation (worktree: true) → validation (worktree: true, fresh: true) → terminal`. AC-2 explicitly asserts that the `worktree:` field has the same value after the implementation→validation advancement as before — i.e., `fresh: true` does not cause a new worktree to be stamped or the existing one to be cleared.
- DONE: Address minor finding (4) — propagate `Once set on first dispatch` sentence to other templates.
  Surface 7 of `### Locked contract wording` extended: same one-sentence addition propagates to the parallel `worktree` rows in `experiment.md` (line 84) and `refinement.md` (line 53), with the rationale that contract documentation should describe the field uniformly across templates so a captain who later adds a `worktree: true` stage to either template inherits the correct mental model. AC-3 updated to cover all three templates.
- DONE: Address minor finding (5) — make redeclared-`worktree: true`-is-no-op explicit.
  Resolved Q (3) extended with a corollary sentence: once an entity has a worktree stamped, any subsequent stage's `worktree: true` declaration is a no-op for that entity. The asymmetry is named so a captain debugging a workflow expects it.
- DONE: Address minor finding (6) — restate orphan-`worktree:` delegation to `status --boot`.
  New `### Orphan worktree handling` subsection added inside `## Proposed approach` after Resolved Q (5). States the invariant (`worktree:` set ↔ live worktree on disk), names the existing boot-time orphan-detection in `status --boot` (cited at `first-officer-shared-core.md` line 19, the `ORPHANS` section), and clarifies that this entity does not introduce a new orphan-recovery path. Defensive guard noted as optional, not load-bearing.

### Summary

Cycle-2 hardens the cycle-1 design without changing direction: retargets AC-3 to a real test file, adds an 8th locked-wording surface for the contradicting codex-runtime clause (with AC-1b for coverage), extends AC-2 to cover the canonical `fresh: true` development-template path, propagates the captain-facing template sentence to all three templates, and tightens two implicit consequences (redeclared-`worktree: true` is a no-op once sticky; orphan-`worktree:` delegates to existing `status --boot` detection) into explicit prose. Eight surfaces are now locked, eight ACs cover them, seven of eight ACs run offline. The architectural soundness from cycle 1 is preserved — this is fix-up, not redesign.

## Stage Report: implementation

- DONE: Land the 8 locked-wording surfaces from the entity's `### Locked contract wording` section verbatim.
  All 8 surfaces applied. Anchors verified by grep: surface 1 line 144 (`Reuse-routing matches the entity's worktree state`), surface 2 line 218 (`When \`worktree:\` is set` inside FO Write Scope `### Feedback Cycles` bullet), surface 3 line 178 (`worktree-side when \`worktree:\` is set, main-side otherwise`), surface 4 line 208 (`including \`### Feedback Cycles\` entries`), surface 5 line 228 (`applies in the appropriate view`), surface 6 `claude-first-officer-runtime.md:155` (`read from the worktree copy when \`worktree:\` is set`), surface 7 — three templates each carry `Once set on first dispatch` (development.md / experiment.md / refinement.md), surface 8 `codex-first-officer-runtime.md:84` (`route the dispatch into that existing worktree`). Negative greps for the prior reuse phrase and the prior codex prose both return 0. `git diff --stat` for the surfaces shows 6 files touched (shared-core, two runtime adapters, three templates).
- DONE: Implement all 8 ACs' tests.
  AC-1 + AC-7: extended `tests/test_repo_edit_guardrail.py` with offline `test_shared_core_stickiness_static_content` (regex-extracted Reuse conditions block, FO Write Scope `### Feedback Cycles` bullet, Feedback Rejection Flow section); positive + negative substring asserts. AC-1b: new `tests/test_codex_runtime_stickiness.py`. AC-2: new `tests/test_worktree_stickiness.py` with three cases — inherited-default-downstream, development-template `fresh: true` path with explicit invariance assertion, and terminal-archive clears. AC-3: new `tests/test_development_template.py` parameterized across all three templates. AC-4: code change in `skills/commission/bin/claude-team::cmd_build` to route on entity stamped `worktree:` field instead of stage `worktree:` mode; new `tests/test_worktree_stickiness_dispatch.py` builds a real git worktree and asserts the assembled prompt names that worktree path for a `worktree: false`-declared next stage. AC-5: extended `tests/test_rejection_flow.py` with worktree-vs-main split assertions on `### Feedback Cycles` (live tier; not run here). AC-6: new `tests/test_feedback_cycles_merge_clean.py` plus four fixture files under `tests/fixtures/feedback-cycles-merge/` (base, treatment-worktree, control-main, worktree-stage-reports-only); treatment merges with exit 0 and 0 markers, control reproduces conflict (exit 1 / >0 markers).
- DONE: Run `make test-static` and the targeted suites from the worktree; cite exit codes and pass counts in the report.
  `make test-static` exit 0; 576 passed, 26 deselected (live), 15 subtests passed in 31.21s. Targeted re-run of the 7 affected suites: 61 passed, 1 xfailed (pre-existing `test_repo_edit_guardrail` live xfail tracked in #196). One pre-existing test (`test_first_officer_shared_core_documents_worktree_ownership_rule` in `tests/test_agent_content.py`) had a verbatim assert on the surface-4 phrase; updated to two split substring asserts that match the new wording (`active stage/status/report/body state` and `lives in the worktree copy`). AC-5 live-claude extension wired into `test_rejection_flow.py` after the existing `[Rejection Signal Present]` block; not run as out-of-scope per checklist.

### Summary

Eight surfaces from `### Locked contract wording` landed verbatim across `first-officer-shared-core.md` (5), `claude-first-officer-runtime.md` (1), `codex-first-officer-runtime.md` (1), and three commission templates. All 8 ACs have offline tests; AC-5 is plumbed into the existing live `tests/test_rejection_flow.py` for validation to run. AC-4 required a one-block code change in `claude-team::cmd_build` to route on the entity's stamped `worktree:` field instead of the next stage's declared mode — the surface-1 contract wording demanded it. AC-6 control case was sharpened until both branches modify the same trailing region of the entity file (the actual PR #176/#177 shape); without that the control passes vacuously and the treatment's clean-merge property is unproven. `make test-static` is green at 576/576 with one pre-existing-test surface-4 assertion updated to match the new wording.
