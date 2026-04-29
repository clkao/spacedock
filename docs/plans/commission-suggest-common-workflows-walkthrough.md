---
parent: commission-suggest-common-workflows.md
purpose: demonstration walkthrough capture for task #221
captured: 2026-04-28
captured-by: implementation ensign (mental walkthrough against the new SKILL.md prose)
---

# Demonstration walkthrough — commission with templates, trait detection, and naming convention

This file captures the four mission walkthroughs called for by the test plan in `commission-suggest-common-workflows.md`. Each walkthrough simulates a captain invoking `/spacedock:commission "<mission>"` and stepping through Phase 1 against the new SKILL.md prose. The transcripts show what the commission skill would say at each step, what trait detection lands on, what layers fire, which template loads, and how the Stage Naming Convention is applied.

These are mental walkthroughs against the static prose, not live `/spacedock:commission` runs. The captain did not request a live run this round — the test plan is satisfied by structural grep checks (already passing) plus this walkthrough capture.

---

## Walkthrough 1 — Code-shipping mission → `development` template

**Captain invokes:** `/spacedock:commission ship feature work end-to-end through review and PR merge`

**Args extraction:** mission extracted as "ship feature work end-to-end through review and PR merge". Greeting is shown, then commission goes to Q1 with mission pre-filled.

**Q1 — Mission + Entity:**
> I'll use this as the workflow mission: "ship feature work end-to-end through review and PR merge". What does each work item represent?

**Captain:** "an implementation task — basically a feature or bugfix"

→ Entity label: `task`, plural: `tasks`, type: `implementation_task`.

**Trait Detection:**
- Mission text contains "ship", "PR", "merge" → strong cue → `development` template.
- Q1 answer "feature or bugfix" reinforces.
- Layer cues: `implementation` and `validation` will modify the repo → repo-mutation layer fires. Shipping ritual is PR review → pr-merge mod offered.
- Inference strategy: strong signal → silent infer + one-shot confirmation.

Commission `Read`s `skills/commission/references/templates/development.md` and applies its `## Adoption` section. Pre-fill stages are `backlog (initial, gate) → ideation (gate) → implementation (worktree) → validation (worktree, fresh, feedback-to: implementation, gate) → done (terminal)`. The development template's Adoption section's confirmation prose is surfaced verbatim:

> I'll set this up as a **development** workflow: tasks move through `backlog → ideation → implementation → validation → done`, with worktrees on `implementation` and `validation` and `validation` running with a fresh agent so it independently checks the AC.
>
> ID style is **sd-b32** by default, because development workflows usually involve multiple worktree branches creating tasks in parallel and you want the IDs to reconcile without coordination. If this workflow is single-writer (just you, no concurrent branches), I can switch to sequential — let me know.
>
> Default mod: **pr-merge** (PR lifecycle automation, framing above). I'll confirm the install at file generation time.

The pr-merge stages-stay-clean framing from the development template's Adoption section is also surfaced (per the generic mod offer mechanism in SKILL.md):

> Because this workflow ships code via PR review, I'll install the **pr-merge** mod. This is the structural reason your stages can stay clean — you don't need a `pr_open` or `awaiting_merge` stage to model the PR step. The mod tracks PR state on the `pr` field, watches for merges in the background, and advances the entity to `done` when the PR lands. Stages describe where work is happening; the PR lifecycle is mod-managed.

**Captain:** "sounds right"

**Q2 — Stages:** Commission presents the development pre-fill stages and applies the Stage Naming Convention. None of `backlog`, `ideation`, `implementation`, `validation`, `done` use a banned prefix; all pass the "the entity is in `{name}`" test. No pushback needed.

**Q3 — Seed Entities, Confirm Design, Phase 2:** proceed normally. Phase 2c surfaces the pr-merge install confirmation as a brief y/n callback to the Phase 1 framing.

**What this walkthrough demonstrates:**
- AC-4: trait detection fires on mission-text cues, lands on the right template.
- AC-1 + AC-6: template `Read` happens, Adoption section drives stage pre-fill, layer activation, mod offer, confirmation prose.
- AC-5: pr-merge framing comes from the development template's Adoption section, surfaced in Phase 1; Phase 2c is install confirmation only.
- AC-2 + AC-3: stage naming convention applies cleanly; pre-filled stages do not violate.

---

## Walkthrough 2 — Hypothesis-test mission → `experiment` template

**Captain invokes:** `/spacedock:commission test hypotheses about pricing-page conversion through tiered evidence`

**Q1 — Mission + Entity:**
> I'll use this as the workflow mission: "test hypotheses about pricing-page conversion through tiered evidence". What does each work item represent?

**Captain:** "an experiment — one hypothesis-test cycle"

→ Entity label: `experiment`, plural: `experiments`, type: `experiment`.

**Trait Detection:**
- Mission text contains "test hypotheses", "tiered evidence" → strong cue → `experiment` template.
- Q1 answer reinforces.
- Layer cues: smoke / run / holdout will sit waiting on evidence → parked-stages layer fires. Whether silence-watcher should fire depends on whether the captain has timeout/nudge semantics in mind — defer the silence-watcher offer until the captain confirms.
- Inference strategy: strong signal → silent infer + confirmation.

Commission `Read`s `skills/commission/references/templates/experiment.md` and applies its `## Adoption` section. Pre-fill stages are `hypothesis (initial, gate) → smoke (parked, gate) → run (parked) → analysis (gate) → holdout (parked, fresh) → accepted (terminal) | rejected (terminal)`. Confirmation prose surfaced verbatim:

> I'll set this up as an **experiment** workflow: each experiment moves through `hypothesis → smoke → run → analysis → holdout → accepted | rejected`. The `smoke` tier is a cheap pre-flight that catches broken setups before you spend the real run; the `holdout` tier is an out-of-sample check that runs with a fresh agent so it cannot be biased by whoever ran the analysis.
>
> Parked-stages layer fires on `smoke`, `run`, and `holdout` — those tiers are designed to sit waiting on evidence, not to be actively worked. If any of those parked stages has a "ping me after N days" or "auto-advance after timeout" rule in mind, I will offer the **silence-watcher** mod when we get to mod offers.
>
> Industry term-of-art for this shape is **stage-gate** (Cooper, *Winning at New Products*) — surfaced here so the lineage is discoverable. The captain-facing name stays `experiment` for first-contact recognition.

**Captain:** "yes — and I want a 5-day timeout on smoke and run"

→ silence-watcher offer fires. Per the experiment template's Adoption section the framing prose is shown:

> Parked stages are normal — an experiment can sit in `smoke` or `run` for a while because that is how evidence accumulates. The **silence-watcher** mod handles the case where parked drifts into stalled... [full prose from experiment.md Adoption section]

(Implementation note: the silence-watcher mod itself does not exist yet — there is a `# TODO: silence-watcher mod not yet shipped` marker in the experiment template's Adoption section. For this walkthrough the offer is surfaced as design-time framing; the actual mod install at Phase 2c would be skipped with a note to the captain that the mod ships in a follow-up task.)

**Q2 — Stages:** all stage names (`hypothesis`, `smoke`, `run`, `analysis`, `holdout`, `accepted`, `rejected`) pass the bucket-noun test — `accepted` and `rejected` are state-flavored, the others are activity-flavored. No pushback.

**What this walkthrough demonstrates:**
- AC-4: trait detection lands on `experiment` from "hypothesis / test / tiered evidence" cues.
- AC-5: silence-watcher framing comes from experiment template's Adoption section, surfaced in Phase 1.
- AC-2: state-flavored names like `accepted` and `rejected` are correctly accepted (not flagged by the no-pleonasm guard).

---

## Walkthrough 3 — Iterate-on-a-document mission → `refinement` template (default, no variant)

**Captain invokes:** `/spacedock:commission iterate on the company OKR document until it's locked`

**Q1 — Mission + Entity:**
> I'll use this as the workflow mission: "iterate on the company OKR document until it's locked". What does each work item represent?

**Captain:** "an OKR draft"

→ Entity label: `draft`, plural: `drafts`, type: `okr_draft`.

**Trait Detection:**
- Mission text contains "iterate" and "locked" → cue → `refinement` template.
- No external touchpoints, no repo mutation cues → no layers fire.
- Variant detection: no variant cue (no contact / sync / publish / PRD signals strong enough to override the default).
- Inference strategy: strong signal → silent infer + confirmation.

Commission `Read`s `refinement.md` and applies the default Adoption (no variant fired). Pre-fill stages: `draft (initial) → review (gate, feedback-to: draft) → polish → done (terminal)`. Confirmation prose:

> I'll set this up as a **refinement** workflow: each draft moves through `draft → review → polish → done` until it is locked. No worktree stages and no PR/merge ritual — this workflow does not touch the repo.

(No layer-driven prose appended because no layers fired.)

**Q2 — Stages:** all clean.

**What this walkthrough demonstrates:**
- AC-4: trait detection lands on `refinement` for the no-strong-signal / iterate-on-artifact case.
- AC-1: refinement template loads and its Adoption section drives behavior even though no layers or mods fire.

---

## Walkthrough 4 — Outreach-pipeline mission → `refinement` + variant + parked-stages layer + silence-watcher offer

**Captain invokes:** `/spacedock:commission run outreach to potential design partners — track each contact through the pipeline`

**Q1 — Mission + Entity:**
> I'll use this as the workflow mission: "run outreach to potential design partners — track each contact through the pipeline". What does each work item represent?

**Captain:** "a contact"

→ Entity label: `contact`, plural: `contacts`, type: `contact`.

**Trait Detection:**
- Mission text contains "outreach", "track", "pipeline", "contact" → cue → `refinement` template (no strong development or experiment signal).
- Within refinement, "outreach", "contact", "pipeline" → outreach variant fires.
- Layer cues: outreach pipeline waits on contact responses → parked-stages layer fires (the `watching` stage in the variant is parked).
- Mod offer: silence-watcher applies to `watching` because outreach typically has timeout/nudge semantics ("if no reply in 7 days, follow up").
- Inference strategy: strong signal → silent infer + variant confirmation.

Commission `Read`s `refinement.md` and applies the outreach variant from the Adoption section. Pre-fill stages: `research (initial) → draft (gate) → sent → watching (parked, gate) → followup (feedback-to: watching) → closed (terminal)`. Confirmation prose, with the variant interpolation from refinement.md's Adoption section:

> I'll set this up as a **refinement** workflow — variant: outreach: each contact moves through `research → draft → sent → watching → followup → closed` until it is locked. No worktree stages and no PR/merge ritual — this workflow does not touch the repo. Parked-stages layer fires on `watching` because the entity sits waiting on external response. I'll offer the silence-watcher mod when we get to mod offers.

The silence-watcher framing prose (from the refinement template's Adoption-section pointer to the layer mechanism, which uses the experiment template's framing if available, or the generic Layer framing reference at the bottom of SKILL.md if not — refinement does not carry its own silence-watcher framing because it is not the default-case template for parked stages).

(Implementation note: the cleanest framing path for outreach + silence-watcher is the generic per-layer parked-stages framing from SKILL.md's Layer framing reference, since the refinement template defers to layer mechanics rather than carrying its own silence-watcher prose.)

**Q2 — Stages:** all bucket-noun-clean. `research`, `draft`, `sent`, `watching`, `followup`, `closed` all pass — `sent` and `closed` are state-flavored past-participles, the rest are activity-flavored.

**What this walkthrough demonstrates:**
- AC-1 + AC-6: refinement template's Adoption section variant menu correctly surfaces the outreach variant.
- AC-4: trait detection cascades from template → variant → layer → mod offer.
- AC-2 + AC-3: state-flavored names (`sent`, `closed`) pass the convention; no false-positive pushback.
- AC-5: layer-driven mod offer fires correctly even when the template itself does not carry the framing prose for the mod (the generic Layer framing reference at the bottom of SKILL.md is the fallback, and the experiment template's silence-watcher prose is reusable as a pattern).

---

## Walkthrough findings summary

All four scenarios exercise the new prose end-to-end:

| Scenario | Template | Variant | Layers | Mods | Convention pushback? |
|---|---|---|---|---|---|
| 1: code shipping | development | — | repo-mutation | pr-merge | none needed |
| 2: hypothesis-test | experiment | — | parked-stages | silence-watcher (offered, mod TODO) | none needed |
| 3: iterate-on-doc | refinement | (default) | none | none | none needed |
| 4: outreach pipeline | refinement | outreach | parked-stages | silence-watcher (offered, mod TODO) | none needed |

No real friction surfaced that maps to the staff-UX-reviewer's deferred concerns:

- **Jargon-heavy trait questions:** the trait-detection cue tables are internal-facing (commission applies them silently); captain-facing prose comes from template confirmation prose, which uses plain-language framing.
- **Mod-concept assumed-known prose:** development template's pr-merge framing explains *what* pr-merge does ("tracks PR state on the `pr` field, watches for merges, advances the entity"); experiment template's silence-watcher framing similarly explains. Captains who have never seen mods get a brief explanation in context.
- **Stage-name edge cases (`triage`, `qa`, `review`):** none of the three templates use these names. The Stage Naming Convention covers them by analogy (`triage` = activity-flavored, `qa` = activity-flavored, `review` = activity-flavored; all pass) but no walkthrough hit them directly.
- **Hybrid-intent layer-assembly:** none of the four scenarios hit the fallback path. A fifth walkthrough (e.g., "ship a series of A/B tests via PR"-shaped mission combining repo-mutation + parked-stages) would exercise the fallback explicitly. Not done because the test plan called for four walkthroughs and the fallback path is documented + grep-verified in SKILL.md.
- **Load-confirmation transparency:** the template `Read` is implicit in the SKILL.md instruction "Read the template file at references/templates/{template}.md"; the captain does not see a "loading template..." message because the prose-driven flow does not have one. Could be added as a small `"I'll use the {template} template — pulling it now..."` line if friction is observed in real captain runs.

No issues require captain attention or escalation. The deferred-concerns list from the staff UX reviewer remains deferred per the captain's explicit decision; no real friction was observed during this walkthrough that would justify reopening any of them.
