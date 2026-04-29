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

## Walkthrough 5 — Post-generation README-edit nudge + `review stages` flow (cycle 2 scope)

This walkthrough picks up after Phase 2 file generation has completed for any of the prior four scenarios — same captain, same mission. Phase 3 Step 1 fires.

**Commission says (announce):**

> Workflow generated! Here's what I created:
> - `docs/.../README.md` — workflow schema and stage definitions
> - {seed entity files}
> - `docs/.../_mods/pr-merge.md` — PR merge mod {if installed}

**Commission says (README-edit nudge — AC-7):**

> Quick heads-up before we start: the README I just generated is the **living spec** for this workflow. Each stage in `docs/.../README.md` has three per-stage bullets — `Outputs:` (what the worker produces), `Good:` (your quality bar), and `Bad:` (anti-patterns to avoid). I drafted those as best-guesses from the mission text, but they are not commitments — they are starting prose for you to tighten so they reflect your actual standards. Open `docs/.../README.md` and edit the bullets under each `### {stage_name}` heading before the first dispatch. Tightening costs minutes now; un-tightening after agents have been dispatched against vague bullets costs more.

**Commission says (review stages offer — AC-8):**

> Type `review stages` if you'd like me to walk you through each stage's expectations one at a time and offer amendments inline — otherwise we'll proceed to the pilot run with the README as-is.

### Branch A — captain proceeds without review

**Captain:** "looks good, proceed"

→ Commission falls through to the agents/launch announcement, then Phase 3 Step 2 (assume first-officer role). No README edits applied.

**What this branch demonstrates:**
- AC-7: nudge is surfaced and the captain can decide on their own.
- AC-8: `review stages` offer is opt-in, not blocking.

### Branch B — captain triggers `review stages`

**Captain:** `review stages`

→ Commission enters Step 1a. Pre-pass scan reads the generated README (in this walkthrough, the development template applied to a "ship feature work" mission from Walkthrough 1). The five stages are `backlog`, `ideation`, `implementation`, `validation`, `done`. Pre-pass flags candidates per the stretch-bullet heuristics:

- `backlog` — Outputs bullet "A seed task file with title, source, brief description" is workflow-specific (not flagged). Good bullet "Clear enough to recognize what the task is about" is moderately generic — flagged as candidate for tightening with a workflow-specific bar.
- `ideation` — Outputs bullets are detailed (problem statement, AC, test plan) — not flagged. Good bullet "Behavior-first, scoped, addresses a real need" is moderately generic — flagged.
- `implementation` — Outputs bullet "The deliverable committed to the worktree branch with a stage report" — workflow-specific because it names worktree branch and stage report — not flagged. Good bullet "Minimal changes that satisfy the AC, clean code" — moderately generic — flagged.
- `validation` — Outputs/Good/Bad are tightly tied to AC verification. Not flagged.
- `done` — terminal state, sparse bullets. Not flagged.

**Commission says (per-stage walk, stage 1 of 5 — backlog):**

> **Stage `backlog`** (1 of 5)
>
> What the task is sitting in: A task that has been proposed but has not yet had design work done.
>
> **Outputs** (what the worker produces):
> - A seed task file with title, source, brief description
>
> **Good** (your quality bar):
> - Clear enough to recognize what the task is about *— flagged: moderately generic; for this workflow, what does "clear enough" actually mean? (e.g., title + 1-line problem statement + how the captain would describe it in standup.)*
>
> **Bad** (anti-patterns to avoid):
> - Empty stub that even the captain cannot triage
>
> What would you like to do? Options:
> - `keep` — accept this stage as-is, move on
> - `tighten outputs` / `tighten good` / `tighten bad`
> - `drop {section} {n}` / `add {section}: {text}`
> - `next stage`
>
> Or describe what you want changed in your own words and I'll apply it.

**Captain:** `tighten good`

> What's the actual quality bar for a task to be considered "ready enough" to sit in backlog? (Replace the bullet "Clear enough to recognize what the task is about" with what?)

**Captain:** "title under 80 chars, one-line problem statement, captain or contributor name on `source`"

→ Commission `Edit`s `docs/.../README.md`, anchoring on the `### backlog` heading and the bullet text being replaced. New bullet:

> - Title under 80 chars, one-line problem statement, captain or contributor name on `source`

**Commission confirms:**

> Tightened backlog Good bullet: replaced with "Title under 80 chars, one-line problem statement, captain or contributor name on `source`". Anything else for this stage, or move on?

**Captain:** `next stage`

→ Commission proceeds through `ideation`, `implementation`, `validation`, `done` with the same pattern. Captain accepts most as-is, tightens one Good bullet on `implementation`.

**Final confirmation:**

> Stage review complete. Here's what got tightened:
> - `backlog`: Good bullet 1 tightened
> - `implementation`: Good bullet 1 tightened
>
> The README at `docs/.../README.md` reflects these edits. Ready to proceed to the pilot run? (yes / let me make more edits)

**Captain:** "yes"

→ Commission falls through to the agents/launch announcement, then Phase 3 Step 2.

**What this branch demonstrates:**
- AC-7: README is positioned as the living spec; the captain understands they are tightening it before the first dispatch.
- AC-8: `review stages` walks one stage at a time (progressive disclosure); pre-pass flags stretch bullets proactively (mixed-inference); per-bullet amendment options work; inline `Edit` applies changes; final confirmation summarizes.
- The proactive flag on `backlog`'s Good bullet correctly identifies "Clear enough to recognize what the task is about" as moderately generic — it would read identically in a research workflow's backlog or a content-production workflow's backlog.
- The non-flagged stages (`validation`, `done`) correctly skip the proactive prompt; the captain still has the per-stage options if they want to amend.

---

## Walkthrough findings summary

The five scenarios exercise the new prose end-to-end:

| Scenario | Template | Variant | Layers | Mods | Convention pushback? |
|---|---|---|---|---|---|
| 1: code shipping | development | — | repo-mutation | pr-merge | none needed |
| 2: hypothesis-test | experiment | — | parked-stages | silence-watcher (offered, mod TODO) | none needed |
| 3: iterate-on-doc | refinement | (default) | none | none | none needed |
| 4: outreach pipeline | refinement | outreach | parked-stages | silence-watcher (offered, mod TODO) | none needed |
| 5: post-gen UX (cycle 2) | (any) | — | — | — | n/a — exercises Phase 3 Step 1 README-edit nudge + `review stages` flow |

No real friction surfaced that maps to the staff-UX-reviewer's deferred concerns:

- **Jargon-heavy trait questions:** the trait-detection cue tables are internal-facing (commission applies them silently); captain-facing prose comes from template confirmation prose, which uses plain-language framing.
- **Mod-concept assumed-known prose:** development template's pr-merge framing explains *what* pr-merge does ("tracks PR state on the `pr` field, watches for merges, advances the entity"); experiment template's silence-watcher framing similarly explains. Captains who have never seen mods get a brief explanation in context.
- **Stage-name edge cases (`triage`, `qa`, `review`):** none of the three templates use these names. The Stage Naming Convention covers them by analogy (`triage` = activity-flavored, `qa` = activity-flavored, `review` = activity-flavored; all pass) but no walkthrough hit them directly.
- **Hybrid-intent layer-assembly:** none of the four scenarios hit the fallback path. A fifth walkthrough (e.g., "ship a series of A/B tests via PR"-shaped mission combining repo-mutation + parked-stages) would exercise the fallback explicitly. Not done because the test plan called for four walkthroughs and the fallback path is documented + grep-verified in SKILL.md.
- **Load-confirmation transparency:** the template `Read` is implicit in the SKILL.md instruction "Read the template file at references/templates/{template}.md"; the captain does not see a "loading template..." message because the prose-driven flow does not have one. Could be added as a small `"I'll use the {template} template — pulling it now..."` line if friction is observed in real captain runs.

No issues require captain attention or escalation. The deferred-concerns list from the staff UX reviewer remains deferred per the captain's explicit decision; no real friction was observed during this walkthrough that would justify reopening any of them.
