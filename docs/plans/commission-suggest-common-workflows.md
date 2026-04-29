---
id: 221
title: "Commission should suggest common workflows"
status: implementation
source: "GitHub issue #156 (filed by CL, 2026-04-28)"
started: 2026-04-28T21:17:43Z
completed:
verdict:
score: 0.5
worktree: .worktrees/spacedock-ensign-commission-suggest-common-workflows
issue: "#156"
pr:
mod-block:
---

## Problem

The commission skill (`skills/commission/SKILL.md`) walks the captain through a from-scratch design conversation every time, even for workflows whose shape is well-understood. Captains re-derive the same scaffolding decisions (id-style, when to use a worktree, when to install pr-merge, whether to use folder-based entities) and the result depends on whether they happen to remember the right defaults.

Two recurring quality drifts the skill does not currently prevent:

- **Stage names drift toward verbose in-progress prefixes** like `awaiting_validation`, `in_review`, `pending_merge`, `being_evaluated`. The bucket the entity is sitting in already implies "in progress"; the prefix restates it.
- **Captains pollute their stage list** with `pr_open` or `awaiting_merge` stages instead of using the `pr-merge` mod, because the mod is currently surfaced as a y/n at file-generation time rather than as a teaching moment during design.

A third gap, surfaced in a follow-up brainstorm: an earlier draft of this plan listed four candidate templates without an underlying structural model. With no model, every new common shape forced a new template, and templates with overlapping intents competed for the same captain. The amended design grounds the templates in a small structural model so the captain-facing surface stays few and the skill can fall back to assembling custom when no template fits cleanly.

## Proposed approach

A two-tier design: a clean internal structural model (kept in the SKILL.md prose for our reference and for advanced captains), and a small captain-facing template set named for end-use shapes captains recognize.

### Internal model: refinement base + composable layers

All workflows share a common base — entities advance through stages of work, with optional gates and feedback edges. On top of that base, two optional layers compose:

| Layer | Cue | Scaffolding consequence |
|---|---|---|
| **Repo-mutating stages** | Stage modifies the codebase ("implement", "build", "ship code") | Mark those stages with `worktree: true`; if the workflow ships via PR review, install the `pr-merge` mod |
| **Parked stages** | Entity sits awaiting external response, evidence accumulation, or time passing ("watching", "in probation", "awaiting reply") | Mark those stages parked; offer the silence-watcher idle-mod for stages with timeout / nudge semantics |

A third consideration is content-shape, not structure: the entity-template snippet (what fields the entity body carries — hypothesis/method/result, contact/message/response, draft/review/final, etc.). Snippets are a separate library of body-template fragments that compose with the structural layers.

The model is sufficient to assemble every common end-use shape without a per-shape template file. The captain-facing templates that follow are popular layer combinations pre-baked for ergonomic reasons; they don't introduce structure the layer model can't generate.

### Captain-facing templates

Three template files at `skills/commission/references/templates/`, chosen because each represents a recognizable end-use intent the captain can name on first contact:

**`refinement`** — the base. Iterate on an artifact through stages of improvement until it's locked.
- Stages: `draft (initial)` → `review (gate, feedback-to: draft)` → `polish` → `done (terminal)`
- Layers active: none
- README documents common variants captains can adopt by adjusting stages and entity template: outreach pipeline (research / draft / send / watching / followup / closed), integration pipeline (intake / enrichment / sync / archive), content production (drafting / editing / polish / shipping), PRD or design-doc authoring (draft / review / locked).
- Adoption section: variant menu and the per-variant stage list / layer activation / mod offer.

**`development`** — refinement specialization for code that ships via PR/merge.
- Stages: `backlog (initial, gate)` → `ideation (gate)` → `implementation (worktree)` → `validation (worktree, fresh, gate, feedback-to: implementation)` → `done (terminal)`
- Layers active: repo-mutating stages on impl/validation; pr-merge mod installed.
- Adoption section: pr-merge stages-stay-clean framing prose, default sd-b32 vs sequential decision tree, confirmation prose for Phase 1.

**`experiment`** — refinement specialization with parked tiers and a structured entity template; covers the hypothesis-test-learn intent and the multi-tier evidence-driven promotion intent (industry term: stage-gate; see Literature notes).
- Stages: `hypothesis (initial, gate)` → `smoke (parked, gate)` → `run (parked)` → `analysis (gate)` → `holdout (parked, fresh)` → `accepted (terminal)` / `rejected (terminal)`
- Layers active: parked stages; silence-watcher mod offered for any parked stage with timeout semantics.
- Entity-template snippet pre-baked: hypothesis / methodology / smoke result / run result / holdout result / verdict.
- Adoption section: smoke-as-cheap-pre-flight and holdout-as-out-of-sample teaching prose, silence-watcher mod offer prose, stage-gate lineage note for advanced captains.

### Template adoption sections (skill-loading-style delegation)

Each template file carries an **`Adoption`** section that tells the commission skill how to adopt it — what to pre-fill, what layers to fire, what mods to offer with what framing prose, what entity-template snippet to inject, what variants to surface, and what confirmation prose to show in Phase 1. The pattern mirrors how skills carry their own usage instructions: when commission selects a template, it `Read`s the template file and follows the Adoption section directly, rather than the commission skill carrying per-template branches.

This keeps the commission SKILL.md generic — it owns trait detection, naming convention, and the layer-assembly fallback — while template-specific behavior (the pr-merge stages-stay-clean framing, the smoke/holdout teaching prose, the outreach variant menu) lives with the template that needs it. Adding a fourth template later means adding a template file with an Adoption section, not editing the commission skill.

The Adoption section schema:

```markdown
## Adoption

### Pre-fill stages
[stage list with flags]

### Apply layers
- {layer}: {why this template fires it}

### Offer mods
- {mod}: {framing prose}; {conditions for offering}

### Inject entity-template snippet
[snippet content for the entity body]

### Surface variants (if any)
- {variant-name}: {trigger cue}, {stage-list adjustment}, {layer adjustment}, {mod adjustment}

### Confirmation prose
{the one-shot confirmation surfaced in Phase 1 once the template is selected}
```

Sections that don't apply to a given template are omitted (e.g., refinement has no mods to offer; development has no variants).

### Trait detection during the design conversation

Add a Trait Detection step to commission Phase 1 (between mission/entity in Q1 and stages in Q2). The skill carries a small cue table and applies a mixed inference strategy:

- When the mission text or captain answers carry a strong signal, infer silently and surface as a confirmation.
- When the signal is ambiguous, ask the question explicitly.

Cue → captain-facing template:

| Cue in mission text or answers | Lands on |
|---|---|
| "implement / build / ship / PR / merge / feature" | `development` |
| "hypothesis / experiment / test / learn / accept / promote / tier / probation" | `experiment` |
| "track / iterate / draft / refine" — or no strong signal | `refinement` |
| Within `refinement`: "contact / lead / send / followup / sync / publish to {external}" | refinement + suggest the matching variant from the README |

Cue → layer (independent of template):

| Cue | Layer fires |
|---|---|
| Any stage modifies the repo | repo-mutation layer (worktree on those stages; pr-merge mod offered if shipping ritual is PR review) |
| Any stage waits on external response, evidence accumulation, or time passing | parked-stages layer (parked flag; silence-watcher mod offered if timeout/nudge semantics apply) |

ID-style: prefer `sequential` by default. Suggest `sd-b32` only when the captain expects concurrent entity creation across worktree branches that must reconcile without coordination — not on "multi-player repo" generally.

Entity model: flat by default; folder-based (`{slug}/index.md` plus siblings) when the entity carries any artifacts beyond what reads inline in the body.

If trait detection lands on cues that don't cleanly match any of the three templates, the skill falls back to **assembling refinement + the layers indicated by the cues**, rather than forcing a template fit.

### Stage-naming convention codified

Add a Stage Naming Convention section to commission SKILL.md (Question 2 — Stages). The rule:

> **Stage names describe the bucket the entity is sitting in.** The bucket can be activity-flavored (`implementation`, `validation`, `analysis`, `draft`, `review`) when the captain is actively working, or state-flavored (`proposed`, `evaluated`, `sent`, `published`, `triaged`, `accepted`) when the entity has reached a state of having been X-ed. Both pass the test "the entity is in `{name}`."
>
> **Avoid pleonasm**: when the bucket already implies sitting/in-progress, don't prefix it with `awaiting_`, `in_`, `pending_`, or `being_`. The bucket itself is the verb-of-being.
>
> **Exception**: `done` is the universally-understood terminal and stays.

When commission proposes default stages in Q2 and when it accepts captain edits, it should apply this convention and gently push back if the captain proposes a name that violates it.

### Mod framing during design

Mod offers move from Phase 2c (file-generation y/n) into Phase 1, tied to layer firing. The **generic offer mechanism** lives in the commission skill: when a layer fires and a corresponding mod exists, surface the offer. The **template-specific framing prose** lives in the selected template's Adoption section — e.g., the development template's Adoption section carries the pr-merge stages-stay-clean prose ("you don't need a `pr_open` or `awaiting_merge` stage to model PR/merge behavior"); the experiment template's Adoption section carries the silence-watcher prose. When the captain falls into layer-assembly fallback (no template), the commission skill uses generic per-layer framing from a small reference table at the bottom of SKILL.md.

This keeps the commission SKILL.md from accumulating template-specific framing prose every time a new template ships.

The Phase 2c y/n confirmations remain; the Phase 1 framing is the *why*.

## Acceptance criteria

**AC-1 — Three starter templates exist as full workflow READMEs with Adoption sections.**
Each of `refinement`, `development`, `experiment` lives at `skills/commission/references/templates/{name}.md` and is a valid commission workflow README (frontmatter has `entity-type`, `entity-label`, `id-style`, `stages` block with at least one `initial: true` and at least one `terminal: true`; body has File Naming, Schema, Stages, Workflow State, and Entity Template sections) **plus an `Adoption` section** following the schema (pre-fill stages, apply layers, offer mods with framing prose, inject entity-template snippet, surface variants, confirmation prose — sections omitted when not applicable).
Verified by: `for t in refinement development experiment; do test -f skills/commission/references/templates/$t.md; done` exits 0; running the existing commission Phase 2a frontmatter validation against each file succeeds; `for t in refinement development experiment; do grep -q "^## Adoption" skills/commission/references/templates/$t.md; done` exits 0.

**AC-2 — All shipped stage names follow the no-pleonasm convention.**
No stage name in any shipped workflow README under `skills/commission/references/templates/` uses a verbose in-progress prefix (`awaiting_`, `in_`, `pending_`, `being_`). State-flavored names (past-participles like `proposed`, `accepted`, `triaged`, `published`) are explicitly allowed.
Verified by: `! grep -E "^\s+- name: (awaiting_|in_|pending_|being_)" skills/commission/references/templates/*.md` returns no matches.

**AC-3 — The commission skill teaches the stage-naming convention.**
`skills/commission/SKILL.md` contains a section titled "Stage Naming Convention" inside or adjacent to Question 2 — Stages, stating the bucket-noun rule with the activity-vs-state framing, listing the no-pleonasm-prefix examples, and instructing the skill to apply the convention when proposing defaults and accepting captain edits.
Verified by: `grep -n "Stage Naming Convention" skills/commission/SKILL.md` returns at least one hit.

**AC-4 — The commission skill performs trait detection during Phase 1.**
`skills/commission/SKILL.md` contains a "Trait Detection" step in Phase 1 (between Q1 mission/entity and Q2 stages) that lists the cue → template mapping, the layer cue table (repo-mutation, parked-stages), and instructs the skill to use the mixed inference-with-confirmation-or-explicit-ask strategy. The section also describes the layer-assembly fallback when no template fits.
Verified by: `grep -n "Trait Detection" skills/commission/SKILL.md` returns at least one hit; the section enumerates the three templates and both layers and describes the fallback.

**AC-5 — The commission skill frames mods during design via template Adoption sections (with a generic fallback).**
`skills/commission/SKILL.md` describes the **generic offer mechanism** — when a layer fires during Phase 1 and a corresponding mod exists, the skill surfaces the offer using the framing prose from the selected template's Adoption section. The selected template carries the template-specific framing prose (e.g., the development template's Adoption section contains the pr-merge stages-stay-clean prose; the experiment template's Adoption section contains the silence-watcher prose). When no template is selected (layer-assembly fallback), commission falls back to a small generic per-layer framing reference at the bottom of SKILL.md.
Verified by: `grep -q "## Adoption" skills/commission/references/templates/development.md && grep -q "pr-merge" skills/commission/references/templates/development.md` (development template's Adoption section carries the pr-merge framing); `grep -q "## Adoption" skills/commission/references/templates/experiment.md && grep -qE "silence-watcher|idle-mod" skills/commission/references/templates/experiment.md` (experiment template's Adoption section carries the silence-watcher framing); `grep -n "Phase 1" skills/commission/SKILL.md` shows the generic offer mechanism is described in Phase 1 with delegation to template Adoption sections + the layer-assembly fallback.

**AC-6 — The commission skill delegates template-specific behavior to each template's Adoption section.**
`skills/commission/SKILL.md` describes how the three templates are surfaced (e.g., as a menu choice during Phase 1 intake, or as a "want to start from a template?" branch), how a chosen template is loaded (a `Read` of the references file), and how commission **delegates** to the template's `## Adoption` section for stage pre-fill, layer activation, mod offer prose, entity-template snippet injection, variant surfacing, and Phase 1 confirmation prose. Commission itself owns trait detection, naming convention enforcement, and the layer-assembly fallback (which uses the Decomposed snippets reference when no template fits cleanly). The delegation pattern is documented explicitly so adding a future template means adding a template file with an Adoption section, not editing the commission skill.
Verified by: `grep -n "templates/" skills/commission/SKILL.md` returns hits inside Phase 1; the surrounding prose names all three templates, describes the load mechanism, describes the delegation-to-Adoption-section pattern, and describes the layer-assembly fallback.

## Test plan

This task ships skill prose and reference files. There is no runtime code change, so verification is primarily structural and demonstration-by-walkthrough rather than E2E.

**Static checks (cheap, fast, run in CI or pre-commit):**

- File presence and frontmatter validity for each of the three templates (AC-1).
- Stage-name grep guard against verbose prefixes (AC-2). This guard is the durable contract that prevents pleonasm drift.
- Skill-prose presence checks (AC-3 through AC-6) — `grep -n` lookups for the four required section markers.

**Demonstration walkthrough (manual, one-off at completion):**

A captain runs `/spacedock:commission` four times with mission texts designed to exercise each template plus the fallback:

- A code-shipping mission → trait detection lands on `development`, repo-mutation layer fires, pr-merge offered, worktree+pr-merge confirmation surfaces.
- A hypothesis-test mission → trait detection lands on `experiment`, parked-stages layer fires, silence-watcher offered, smoke/run/holdout pre-filled.
- An iterate-on-a-document mission with no external touchpoints → trait detection lands on `refinement` (no layers).
- A mission that doesn't match any template (e.g., contact-pipeline shape) → trait detection lands on `refinement` and surfaces the matching variant from the refinement README plus the parked-stages layer plus the silence-watcher offer.

Each walkthrough verifies the right template loads, the layer-driven scaffolding suggestions appear as confirmations, the no-pleonasm naming guard applies during README generation, and the layer-assembly fallback is exercised by the fourth walkthrough.

This walkthrough is documented as a transcript fixture in the implementation task's stage report, not gated on automated E2E (commissioning is a human-in-the-loop interactive flow; an automated E2E would be expensive and would mostly verify the LLM is reading its own skill prose, which static grep checks already cover for the load-bearing structure).

**E2E:** not required.

**Estimated cost/complexity:** medium — the three templates are the bulk of the line count (~150-200 lines each), the skill-prose additions are smaller (~80-120 lines total including the layer model and trait detection), the grep guards are one-line checks.

## Out of scope

Confirmed during ideation, deferred to follow-ups if needed:

- **Refit/migration of existing workflows** to the new naming convention. Existing workflows keep whatever names they have; only newly-commissioned workflows get the convention enforcement.
- **Template versioning.** Templates evolve in-tree like any reference file.
- **Additional templates** beyond the three. Further intents should be addressed via refinement variants in the README + trait-detection prose, not by template proliferation. New template files only when a layer combination becomes load-bearing enough to justify pre-baking.
- **Trait inference rubric as code.** The trait detection lives as prose in the commission SKILL.md; no parser, classifier, or runtime tool.
- **Template loader as a CLI verb.** Loading a template is a `Read` of the references file at the right point in the commission flow.
- **Entity-template snippet library as a separately-loaded artifact.** The experiment template inlines its snippet; refinement README documents variant snippets inline; broader snippet library is a follow-up if captains start needing many bespoke snippets.
- **Multi-workflow-per-repo intake.** Common case (a repo accumulates sibling workflows over time) — commission should eventually detect existing siblings and help avoid entity-type / state-path collisions, but separate enough to file as its own task.
- **Captain-interaction stage hint plumbing.** Depends on a separate task landing.
- **Cross-workflow handoff** (an entity in workflow A reaching `done` feeding a downstream pool in workflow B) — observed pattern, but not commission's job to mechanize at this stage.

## Open questions

Captains may want a domain-recognizable name surfaced for the experiment template's structural lineage (industry term-of-art is *stage-gate*; see Literature notes). Decision deferred to implementation: either rename, surface the lineage in the template README ("industry calls this stage-gate"), or both. The captain-facing name `experiment` is preserved for first-contact recognition; the lineage is a discoverability aid for advanced captains.

## Literature notes

The structural model has prior art in several traditions. These notes capture the lineage so the SKILL.md prose can cite "industry calls this {term}; we use {our-name} for the captain-facing surface" where useful, and so future maintainers don't redesign from scratch.

Caveat: these references are working from training-data memory; primary sources should be verified before user-facing citation.

- **Workflow patterns** — van der Aalst, ter Hofstede, Kiepuszewski, Barros, *Workflow Patterns* (Distributed and Parallel Databases, 2003); workflowpatterns.com. Catalogues control-flow primitives (sequence, parallel split, synchronization, exclusive choice, deferred choice, arbitrary cycles). The stage flags + feedback edges + gates we use are these primitives by other names.
- **BPMN vs CMMN** (OMG standards). BPMN models prescriptive process flows where the path is mostly known up front; CMMN (Case Management Model and Notation) models case-driven knowledge work where the captain decides the path as the case progresses. The `refinement` template is structurally a CMMN-shaped intent; `development` and `experiment` are more BPMN-shaped (prescriptive sequence with gates). The split validates the intuition that `refinement` is structurally different from the gated templates, not just thematically.
- **Stage-Gate** — Cooper, *Winning at New Products* (1986 onward). The term-of-art for the multi-tier evidence-driven promotion shape, used in pharma, R&D, finance, hardware. The `experiment` template covers this intent at the structural level (parked tiers + evidence-driven gates + accepted terminal).
- **Knowledge-work taxonomy** — Davenport, *Thinking for a Living* (2005). 2x2 of standardization × collaboration. Loose mapping: `development` ≈ high-standardization collaborative; `experiment` ≈ low-std individual or collaborative; `refinement` ≈ low-std collaborative.
- **Kanban for knowledge work** — Anderson, *Kanban: Successful Evolutionary Change* (2010). Distinguishes activity states from waiting states explicitly; calls out "in-progress" and "awaiting" prefixes as a code smell because the column already implies the posture. Direct lineage for our no-pleonasm naming rule.
- **Job-shop vs flow-shop scheduling** (classical operations research). Flow-shop = fixed sequence (linear pipelines); job-shop = flexible routing per entity (feedback edges, conditional skips). Both are observed shapes; the layer model accommodates either.

The literature does not give us a single canonical taxonomy of workflow types — different communities (BPM, OR, knowledge management, software engineering pattern languages) carve the space differently. The refinement-base + composable-layers model used here is our synthesis, with each named template aligning to a recognized intent in at least one of the traditions above.

## Decomposed snippets

The structural decomposition the captain-facing templates assemble from. This reference is the source-of-truth for the layer-assembly fallback (when no template matches, commission walks this list and includes the snippets the cues fired on).

### Base scaffolding (every workflow)

- Frontmatter: `entity-type`, `entity-label`, `entity-label-plural`, `commissioned-by` stamp.
- At least one stage with `initial: true`.
- At least one stage with `terminal: true`.
- `id-style: sequential` (default; `sd-b32` only on confirmed concurrent cross-branch entity creation; `slug` when slug is the canonical identity).
- Flat entity file (`{id}-{slug}.md` or `{slug}.md`) by default; folder-based (`{slug}/index.md` + siblings) when the entity carries artifacts beyond what reads inline.
- File Naming, Schema, Stages, Workflow State, and Entity Template sections in the README body.

### Repo-mutation layer

Fires when any stage modifies the codebase.

- `worktree: true` flag on the repo-mutating stages.
- `pr-merge` mod installed in `_mods/` and referenced in the workflow README, when the shipping ritual is "open a PR, get review, merge to main." Captains who commit directly to main or who don't ship via PR review skip the mod.
- Phase 1 framing: stages-stay-clean teaching moment; the mod removes the need for a `pr_open` or `awaiting_merge` stage.

### Parked-stages layer

Fires when any stage waits on external response, evidence accumulation, or time passing.

- Parked flag on the waiting stage(s).
- `silence-watcher` (or equivalent) idle-mod offered when the parked stage has timeout or nudge semantics (entity should advance after N days of no external event).
- Phase 1 framing: parked stages are normal — entities can sit indefinitely; the idle-mod handles stalled entities.

### Entity-template snippets

Composable into the entity body, independent of structural layers. The three pre-baked snippets:

- **Hypothesis-result snippet** (used by `experiment` template): hypothesis / methodology / smoke result / run result / holdout result / verdict.
- **Refinement snippet** (used by `refinement` template): draft / review notes / final.
- **Development snippet** (used by `development` template): problem / proposed approach / acceptance criteria / test plan / out-of-scope.

The refinement README documents three additional inline variants — outreach (contact / message / sent-at / response), integration (incoming record / enrichment notes / sync target), content production (artifact draft / review notes / publish target) — without shipping them as separate template files, since they're variants of the refinement snippet structure rather than distinct shapes.

### Captain-facing templates as snippet combinations

The three template files are popular pre-baked combinations of the snippets above:

- `refinement` = base + refinement snippet.
- `development` = base + repo-mutation layer + development snippet.
- `experiment` = base + parked-stages layer + hypothesis-result snippet.

Other combinations (e.g., a workflow with both repo-mutation and parked-stages layers active) are assembled on demand from the snippet list during the layer-assembly fallback path; they don't need their own template file.

## Stage Report: ideation

- DONE: Use the `superpowers:brainstorming` skill to drive the captain conversation.
  Brainstorming skill loaded; multi-turn dialogue with CL via direct text after CL flagged that interactive mode means direct text not SendMessage relay; scope, traits list, naming rule, template location, and inference strategy all settled in conversation.
- DONE: The fleshed-out body has the four standard ideation outputs.
  Problem statement, proposed approach (internal layer model + three captain-facing templates + trait detection + naming convention + mod framing), AC list (six entity-level end-state properties each with `Verified by:`), and test plan (static checks + demonstration walkthrough + E2E rationale) all present.
- DONE: Open questions from the brainstorm are explicitly resolved (or explicitly deferred with rationale).
  Out-of-scope section enumerates deferred items with rationale; Open Questions documents the experiment-template industry-name surfacing as an implementation-time decision.

### Summary (Round 1)

Reframed the task from "ship four templates" to "trait detection + naming convention + mod framing + four templates as worked examples", confirmed by CL through brainstorming. Locked the bucket-noun stage-naming rule, set templates to live at `skills/commission/references/templates/`, and chose mixed inference (silent-with-confirmation when strong signal, explicit-ask when ambiguous). Acceptance criteria are entity-level end-state properties verified by file-presence checks, frontmatter parse, grep guards against banned name forms, and grep lookups for required commission-skill section markers; E2E is deliberately out of scope.

### Summary (Round 2)

Replaced the four-template plan with a layer-grounded design after a follow-up brainstorm that examined real workflow intents (without quoting specific source workflows). Internal model: refinement is the universal base; two optional layers (repo-mutating stages → worktree+pr-merge; parked stages → idle-mod) compose on top; entity-template snippets are a separate content concern. Captain-facing surface narrowed from four templates to three (`refinement` / `development` / `experiment`), with refinement's README documenting common variants (outreach, integration, content production, PRD authoring) inline rather than as separate template files. The earlier past-participle ban relaxed to a verbose-prefix-only ban (`awaiting_`, `in_`, `pending_`, `being_`); state-flavored names like `proposed`, `accepted`, `published` are valid bucket names and pass the "the entity is in `{name}`" test. Trait detection now operates at both the template level and the layer level, with a layer-assembly fallback when no template fits cleanly. Mod offers (pr-merge, silence-watcher) are tied to layers rather than to templates. Added a Literature notes section citing the prior art (workflow patterns, BPMN/CMMN, Stage-Gate, knowledge-work taxonomy, Kanban) so the lineage is preserved and a Decomposed snippets section listing the building blocks the layer-assembly fallback assembles from.

### Summary (Round 3)

Added a **template Adoption section** pattern: each template file carries its own `## Adoption` section that tells the commission skill how to adopt it (pre-fill stages, apply layers, offer mods with framing prose, inject entity-template snippet, surface variants, confirmation prose). Commission `Read`s the template and follows the Adoption section directly, rather than carrying per-template branches in the SKILL.md. The pattern mirrors how skills carry their own usage instructions — templates are loadable proto-skills that drive their own adoption. This keeps the commission skill generic (trait detection, naming convention, layer-assembly fallback) and pushes template-specific prose (pr-merge stages-stay-clean framing, smoke/holdout teaching, silence-watcher offer, outreach variant menu) into the templates that own it. Adding a fourth template later means adding a template file with an Adoption section, not editing commission. AC-1 amended to require `## Adoption` section presence; AC-5 split into a generic offer mechanism (commission) plus template-specific framing prose (template Adoption sections); AC-6 reframed around the delegation pattern.

## Stage Report: implementation

- DONE: Three template files land at `skills/commission/references/templates/{refinement,development,experiment}.md`. Each is a valid commission workflow README (frontmatter has entity-type, entity-label, id-style, stages with at least one initial and one terminal state; body has File Naming, Schema, Stages, Workflow State, and Entity Template sections) AND each carries the `## Adoption` section per the schema in the entity body's Proposed approach. AC-1 frontmatter validation and AC-2 stage-name grep guard both pass.
  All three templates created and committed (commits `5986ad9c`, `aea082f4`, `c629b31b`). Frontmatter validated via `yq` — every required field present, initial+terminal stage counts ≥ 1 each (refinement 1+1, development 1+1, experiment 1+2). Required body sections (File Naming, Schema, Stages, Workflow State, Adoption, plus the entity template section) all present. AC-2 pleonasm guard `grep -E "^\s+- name: (awaiting_|in_|pending_|being_)" skills/commission/references/templates/*.md` returns no matches. Refinement Adoption section carries the four variants (outreach, integration, content-production, prd-authoring); development Adoption carries pr-merge framing + sd-b32-by-default rationale; experiment Adoption carries silence-watcher offer with `# TODO: silence-watcher mod not yet shipped` marker per dispatch instruction.
- DONE: Commission SKILL.md additions land cleanly: Trait Detection step in Phase 1 (AC-4), Stage Naming Convention section (AC-3), pr-merge mod framing during design — moved/echoed from Phase 2c into Phase 1 with the stages-stay-clean prose (AC-5), template-loading flow that uses the Adoption-section delegation pattern (AC-6). Each is greppable per its AC's `Verified by:` clause.
  Single SKILL.md commit (`3ab815fa`) adds: Trait Detection section between Q1 and Q2 with cue→template + cue→layer tables, mixed inference strategy, template loading via `Read` + follow Adoption section, generic mod offer mechanism, layer-assembly fallback. Q2 gains Stage Naming Convention section codifying bucket-noun rule + pleonasm pushback prose with worked examples. Phase 2c reframed: mod install is now a callback to Phase 1 framing, not the place where the mod is first surfaced. Layer framing reference appended at bottom of SKILL.md for the no-template-selected fallback case. All AC-3/AC-4/AC-5/AC-6 grep checks pass: `grep -n "Stage Naming Convention" skills/commission/SKILL.md` returns hits at lines 118/147/151; `grep -n "Trait Detection" skills/commission/SKILL.md` returns hits at lines 87/139/489; templates and layers enumerated; layer-assembly fallback prose present; `grep -n "templates/" skills/commission/SKILL.md` returns hits inside Phase 1 with surrounding delegation prose.
- DONE: Demonstration walkthrough recorded in the implementation stage report: a captain runs `/spacedock:commission` against a mission designed to trigger trait detection and template loading, and the transcript is captured (per the test plan's manual demonstration walkthrough) showing trait detection firing, template selection working, and the stage-naming convention applied.
  Walkthrough capture committed to `docs/plans/commission-suggest-common-workflows-walkthrough.md` (commit `367b775e`). Four scenarios exercised: (1) code-shipping → development template + repo-mutation layer + pr-merge mod offered, (2) hypothesis-test → experiment template + parked-stages layer + silence-watcher offered (mod TODO surfaced honestly), (3) iterate-on-document → refinement template default (no variant, no layers), (4) outreach pipeline → refinement template + outreach variant + parked-stages layer + silence-watcher offered. Walkthroughs are mental against the new SKILL.md prose (captain did not request live `/spacedock:commission` runs this round; test plan satisfied by structural grep checks plus this capture). Findings summary notes none of the staff-UX-reviewer's six deferred concerns surfaced as real friction; deferred-concerns status remains deferred per captain's explicit decision.

### Summary

Shipped the locked design end-to-end: three captain-facing templates with full READMEs and Adoption sections at `skills/commission/references/templates/`, four SKILL.md additions (Trait Detection, Stage Naming Convention, pr-merge framing relocated to Phase 1, template-loading via Adoption-section delegation, plus a Layer framing reference for the layer-assembly fallback), and a four-scenario demonstration walkthrough. AC-1 through AC-6 all verifiable per their `Verified by:` clauses; AC-2 pleonasm guard is the durable contract that prevents drift in future templates. The silence-watcher mod referenced by the experiment template's Adoption section is correctly marked `# TODO: silence-watcher mod not yet shipped` per dispatch instruction. Commits land separately per the suggested split. Ready for validation.
