---
name: commission
description: "This skill should be used when the user asks to \"commission a workflow\", \"create a workflow\", \"design a workflow\", \"launch a workflow\", or wants to interactively design and generate a plain text workflow with stages, entities, and a first-officer agent."
user-invocable: true
---

# Commission a Plain Text Workflow

You are commissioning a plain text workflow. A plain text workflow is a directory of markdown files with YAML frontmatter, where each file is a work entity that moves through stages. The directory's README is the single source of truth for schema and stages, and the Spacedock plugin provides the plugin-shipped status viewer and plugin-shipped PR merge mod at runtime.

This is a v0 shuttle-mode workflow: an ensign agent handles all stages, with optional mods that inject behavior at lifecycle points (e.g., PR creation at merge time). You will walk {captain} through interactive design, generate all workflow files, then launch a pilot run.

Follow these three phases in order. Do not skip or combine phases.

## Batch Mode

If the user provides design inputs in their message (some or all of: mission, entity type, stages, approval gates, seed entities, location):

1. Extract all provided inputs
2. For any missing inputs, infer reasonable defaults based on the mission
3. Skip directly to **Confirm Design** with the assembled inputs
4. If the user says to skip confirmation or auto-approve gates, proceed without asking

This allows non-interactive use: all inputs in one message, straight to generation.

---

## Phase 1: Interactive Design

Before asking Question 1, greet {captain} with the following (skip this greeting entirely in batch mode):

> Welcome to Spacedock! We're going to design a plain text workflow together.
>
> I'll walk you through three phases:
> 1. **Design** — a few questions to shape the workflow
> 2. **Generate** — I'll create all the workflow files
> 3. **Pilot run** — I'll launch the workflow to process your seed entities
>
> Throughout this workflow, you'll be addressed as **{captain}** (the workflow operator).
>
> Let's start designing.

### Args Extraction

If the user's invocation message contains text beyond the command name (e.g.,
`/spacedock:commission product idea to simulated customer interview`), treat
that text as the mission statement.

- Extract `{mission}` from the args
- Proceed to Question 1 but present the extracted mission for confirmation
  rather than asking from scratch:

  > I'll use this as the workflow mission: "{extracted_mission}"
  >
  > What does each work item represent?

This skips the "what's this workflow for?" half of Q1 and goes straight to
the entity-type follow-up.

Ask {captain} the remaining questions **one at a time**. Wait for each answer before asking the next question. Do not batch questions.

### Question 1 — Mission + Entity

Ask:

> What's this workflow for, and what does each work item represent?
>
> Example: "Track design ideas through review stages" — the workflow is for tracking, each item is a design idea.

Extract `{mission}` and `{entity_description}` from the answer. If the answer clearly covers both mission and entity, proceed. If only the mission is clear, ask a brief follow-up:

> Got it. What does each work item in this workflow represent? (e.g., "a design idea", "a bug report", "a candidate feature")

**Derive the entity label** from `{entity_description}`:

1. Strip leading articles ("a", "an", "the")
2. Take the last word (the head noun in English) — this is `{entity_label}` (lowercase, singular)
3. Derive `{entity_label_plural}` by appending "s" to `{entity_label}`
4. Derive `{entity_type}` as the full description (after stripping articles) in snake_case (e.g., "a design idea" → `design_idea`)

Examples:
- "a design idea" → label: `idea`, plural: `ideas`, type: `design_idea`
- "a bug report" → label: `report`, plural: `reports`, type: `bug_report`
- "an implementation task" → label: `task`, plural: `tasks`, type: `implementation_task`
- "a PR" → label: `pr`, plural: `prs`, type: `pr`

### Trait Detection

Before Question 2 (Stages), apply trait detection to the mission text and the Q1 answers. The goal is to land the workflow on one of three captain-facing templates (or fall back to layer-assembly when no template fits cleanly), and to surface layer activations that drive scaffolding decisions and mod offers. Trait detection sits here because it needs Q1's mission and entity description as input, and because its output pre-fills the stage list Q2 then proposes.

#### Cue → captain-facing template

| Cue in mission text or Q1 answers | Lands on |
|---|---|
| "implement / build / ship / PR / merge / feature / refactor / land code" | `development` |
| "hypothesis / experiment / test / learn / accept / promote / tier / probation / A/B / eval" | `experiment` |
| "track / iterate / draft / refine / publish / send / outreach / lead / contact / sync / ingest" — or no strong signal | `refinement` |
| Within `refinement`: "contact / lead / send / followup", "ingest / sync / external", "publish / blog / article", "PRD / RFC / design doc" | refinement + suggest the matching variant from the refinement template's Adoption section |

#### Cue → layer (independent of template)

| Cue | Layer fires |
|---|---|
| Any stage modifies the repo (write code, edit files, generate artifacts in-tree) | repo-mutation layer (worktree on those stages; pr-merge mod offered if shipping ritual is PR review) |
| Any stage waits on external response, evidence accumulation, or time passing | parked-stages layer (parked flag on the waiting stages; silence-watcher idle-mod offered if timeout/nudge semantics apply) |

#### Inference strategy: silent-with-confirmation, or explicit-ask

- **Strong signal in mission text or Q1**: infer silently and surface as a one-shot confirmation. Example: *"This looks like a development workflow (you mentioned shipping code via PR). I'll use the development template, which sets up backlog → ideation → implementation → validation → done with worktrees on the build tiers and the pr-merge mod for the PR step. Sound right?"*
- **Ambiguous or no signal**: ask explicitly. Example: *"Is this workflow about (a) shipping code through review, (b) testing a hypothesis through tiers of evidence, or (c) iterating on an artifact until it is good enough? Or something else — describe and I will assemble the stages from the cues."*

Apply the same strategy to layers: if the cue is strong, fold the layer activation (and any tied mod offer, framed per the template's Adoption section) into the template confirmation prose; if ambiguous, ask explicitly during Q2.

#### Template loading

Once the template is selected, **`Read` the template file** at `references/templates/{template}.md` (resolve relative to the commission skill directory). Then **follow the template's `## Adoption` section directly** — that section tells you what stages to pre-fill, what layers to fire, what mods to offer with what framing prose, what entity-template snippet to inject, what variants to surface, and what confirmation prose to show in Phase 1.

The three templates available are `refinement`, `development`, and `experiment` — each at `references/templates/{name}.md`. The commission skill itself owns trait detection (this section), the Stage Naming Convention (Q2), the layer-assembly fallback (below), and the generic per-layer mod-framing reference (bottom of this file). Template-specific behavior — the development template's pr-merge stages-stay-clean framing, the experiment template's smoke/holdout teaching prose and silence-watcher offer, the refinement template's variant menu — lives in each template's Adoption section. Adding a fourth template later means dropping a new file in `references/templates/` with an Adoption section, not editing this skill.

#### Generic mod offer mechanism

When a layer fires during trait detection and a corresponding mod exists, surface the offer as part of the Phase 1 confirmation. The **framing prose** comes from the selected template's Adoption section — the development template carries the pr-merge stages-stay-clean prose, the experiment template carries the silence-watcher prose. When no template was selected (layer-assembly fallback), use the generic per-layer framing from the **Layer framing reference** at the bottom of this file.

The Phase 2c y/n confirmations remain — that is the file-generation install step, separate from the Phase 1 design-time framing.

#### Layer-assembly fallback

When trait detection fires layer cues but no template matches cleanly (e.g., a mission that mixes repo-mutation with parked stages and does not look like development or experiment), do not force-fit a template. Instead:

1. Skip the template `Read`.
2. Use the Decomposed snippets reference at `skills/commission/references/decomposed-snippets.md` as the source list of base scaffolding + layer snippets.
3. Assemble the suggested stage list from base scaffolding plus the layer snippets the cues fired.
4. Use the generic per-layer framing from the **Layer framing reference** at the bottom of this file when surfacing mod offers.

The fallback exists so commission gracefully handles novel intents instead of pushing every workflow into one of three boxes.

### Question 2 — Stages

If a template was selected during Trait Detection, use the template's Adoption-section pre-fill as the suggested stage list (with any layer-driven adjustments already applied per the template's framing). Otherwise (layer-assembly fallback), assemble the suggested list from the Decomposed snippets reference. Present the suggestion as an itemized list and ask {captain} to review:

> Based on your workflow mission, here are the stages I'd suggest:
>
> {for each stage: "1. **{stage_name}** — {one-line description}"}
>
> Would you like to modify, add, or remove any stages? (confirm or describe changes)

Apply the **Stage Naming Convention** (below) when proposing the default list and when accepting captain edits. Push back gently if the captain proposes a name that violates the convention; show the convention rule and offer the corrected name.

Store the confirmed stages as `{stages}`. The first stage is `{first_stage}` and the last is `{last_stage}`.

#### Stage Naming Convention

**Stage names describe the bucket the entity is sitting in.** The bucket can be activity-flavored (`implementation`, `validation`, `analysis`, `draft`, `review`) when the captain is actively working, or state-flavored (`proposed`, `evaluated`, `sent`, `published`, `triaged`, `accepted`) when the entity has reached a state of having been X-ed. Both pass the test "the entity is in `{name}`."

**Avoid pleonasm**: when the bucket already implies sitting/in-progress, do not prefix it with `awaiting_`, `in_`, `pending_`, or `being_`. The bucket itself is the verb-of-being. Examples to push back on:

- `awaiting_validation` → `validation` (the entity is in validation)
- `in_review` → `review` (the entity is in review)
- `pending_merge` → drop the stage; PR/merge is mod-managed via the `pr` field, not modeled as a stage (see the development template's pr-merge framing)
- `being_evaluated` → `evaluated` (state-flavored) or `evaluation` (activity-flavored)
- `awaiting_response` → `watching` (state-flavored, parked) or `outreach` (activity-flavored, parked)

**Exception**: `done` is the universally-understood terminal and stays as-is.

When the captain proposes a name that violates the convention, surface the rule and the corrected name as a suggestion — do not silently rewrite. Example pushback: *"`awaiting_validation` reads as 'the entity is in awaiting_validation', which double-states the in-progress posture. The bucket itself is the verb-of-being — would `validation` work? (You can keep `awaiting_validation` if there is a reason — let me know.)"*

### Question 3 — Seed Entities

Ask:

> Give me 2–3 starting items to seed the workflow. For each, provide:
> - **Title** — short name
> - **Description** — a sentence or two about what this entity is
> - **Score** (optional) — priority from 0.0 to 1.0

Store as `{seed_entities}` — a list of objects with title, description, and score. Default `source` to "commission seed" for all seed entities.

If {captain} references an external source for seed data (e.g., "find the info in ~/git/spacedock"
or "see the backlog in project X"), read the referenced files directly using Read/Glob.
Do NOT spawn an Agent for this — a direct file read is sufficient. Look for:
- README files in the referenced directory
- Markdown files with YAML frontmatter (existing entities)
- Any obvious manifest or index file

### Confirm Design

After collecting answers, derive all remaining values from the mission context:

- `{approval_gates}` — default: gate before the terminal stage (e.g., the last stage before terminal).
- `{rejection_flow}` — for each approval gate, determine which earlier stage gets bounced back to on rejection (default: the stage immediately before the gated stage).
- `{dir}` — `docs/{mission-slug}/` where `{mission-slug}` is the mission condensed to a short lowercase hyphenated directory name.
- `{captain}` — "Captain".
- `{id_style}` — choose explicitly with {captain}. Offer:
  - `sd-b32` (recommended for collaborative workflows): use when multiple people or agents may create entities across branches, worktrees, offline edits, or long-running projects.
  - `sequential` (compatibility/default): use when the workflow is single-writer, small, or needs continuity with existing numeric IDs. This is the non-interactive default when no collaboration signal is present and no `--id-style` was provided.
  - `slug` (canonical filename): use when the slug is already the durable identity, such as named projects, semantically numbered episodes, or workflows with single-digit or low double-digit entity counts.

Recommend `sd-b32` when the workflow has worktree stages, PR/merge mods, team-mode agents, or {captain} mentions collaboration, concurrency, branches, worktrees, offline editing, or multiple creators. SD-B32 means Spacedock Base32: stored IDs are 24-character lowercase values derived from SHA-256 digest material and formatted with Spacedock's human-safe alphabet `0123456789abcdefghjkmnpqrstvwxyz`; the status viewer displays and accepts the shortest unique prefix with `MIN_PREFIX: 2`.

Use these exact prompt labels: sd-b32 (recommended for collaborative workflows), sequential (compatibility/default), and slug (canonical filename).

Present the full summary with all derived values. Use plain language for stage behavior — do not expose implementation vocabulary like `worktree`, `gate`, `fresh`, or `feedback-to`:

> I'll call you {captain} — let me know if you prefer something else.
>
> For each run, we process {entity_description_as_item_label} going through the following stages:
>
> {for each stage: "{letter}. {stage_name} — {stage_description}"}
>
> {if any gates: "If you reject at {gated_stage}, it goes back to {target_stage} for revision."}
>
> {if domain_specific_fields: "With the following custom fields:"}
> {for each field: "- {field_name}: {field_description}"}
>
> Our pilot run will be with:
> {for each seed: "- {title}"}
>
> Entity identity will use `{id_style}`.
>
> All files will be created in `{dir}` for you to review.
>
> Accept this design, or tell me what to change.

Wait for {captain} to confirm before proceeding to Phase 2. If {captain} wants changes, apply them and re-present the summary.

---

## Phase 2: Generate Workflow Files

### Ensure Git Repository

Before generating files, ensure the project has a git repository:

1. Check if the current directory is inside a git repo (`git rev-parse --git-dir`).
2. If not, initialize one silently: `git init && git add -A && git commit --allow-empty -m "initial commit"`.
3. Do NOT ask {captain} for permission — a workflow requires git.

### Generation Discipline

Generate all workflow files without creating tasks or updating progress trackers.
Do NOT use TaskCreate, TaskUpdate, or TodoWrite during file generation — these
create visible noise in {captain}'s UI. The generation checklist at the end of
Phase 2 is sufficient for tracking completion.

### Read Spacedock Version

Before generating any files, read the Spacedock plugin manifest to get the current version:

1. Read `.codex-plugin/plugin.json` from the Spacedock plugin directory (the directory containing the `skills/` folder — resolve from your own plugin context). `.claude-plugin/plugin.json` is the synchronized legacy mirror during migration.
2. Extract the `version` field and store it as `{spacedock_version}`.

This version will be embedded in each generated scaffolding file.

### Generate Files

Create the workflow directory and generate the workflow files. Use the design answers to fill all templates — no placeholder text should remain in generated files.

```
mkdir -p {dir}
```

Also ensure `.worktrees/` is in the project's `.gitignore` (worktrees should never be committed):

```
# If .gitignore doesn't exist, create it. If it exists, append only if .worktrees/ isn't already listed.
grep -qxF '.worktrees/' {project_root}/.gitignore 2>/dev/null || echo '.worktrees/' >> {project_root}/.gitignore
```

### 2a. Generate `{dir}/README.md`

Write the README with ALL of the following sections. Every section is required — do not omit any.

Craft thoughtful, mission-specific content for each stage definition. The inputs, outputs, quality criteria, and anti-patterns should be specific to what this workflow actually does — not generic placeholders.

Do NOT include a Scoring Rubric section by default. Scoring uses a simple 0.0–1.0 float — no rubric needed. If {captain} explicitly asks for a multi-dimension rubric, include a Scoring Rubric section documenting their chosen dimensions.

Use this template structure, filling in all `{variables}` from the design phase:

````markdown
---
commissioned-by: spacedock@{spacedock_version}
entity-type: {entity_type}
entity-label: {entity_label}
entity-label-plural: {entity_label_plural}
id-style: {id_style}
stages:
  # Stage names must match ^[a-z0-9][a-z0-9-]*[a-z0-9]$ (kebab-case lowercase, no underscores or spaces); `status --validate` rejects others.
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: {first_stage}
      initial: true
    {For each middle stage, add an entry with per-stage overrides only when different from defaults:}
    - name: {stage_name}
      {worktree: true — only if the stage modifies code or produces artifacts beyond the entity file}
      {fresh: true — only if an independent perspective matters (e.g., a feedback stage that should assess without prior context)}
      {feedback-to: {target_stage} — if this stage has a rejection flow that bounces back to {target_stage}. Infer from the rejection_flow derived in Confirm Design.}
      {gate: true — if this stage is an approval gate}
      {agent: {agent-name} — only if {captain} specifies a non-default agent for this stage. Omit to use the default ensign. The value is the agent file basename without .md.}
      {skill: {plugin:skill-name} — only if the stage should load a specific skill from an installed plugin. The ensign will invoke Skill("{plugin:skill-name}") before starting stage work. Omit for stages without a plugin-provided skill.}
    - name: {last_stage}
      terminal: true
  transitions:
    {Omit this block entirely for linear workflows.}
    {For non-linear flows, add explicit edges:}
    - from: {source_stage}
      to: {target_stage}
      label: {human-readable label}
---

# {mission}

{One paragraph expanding on the mission, describing what this workflow processes and why.}

## File Naming

Each {entity_label} lives as either:

- a flat markdown file `{slug}.md` (default — use this unless the entity produces many artifacts), or
- a folder `{slug}/` containing `index.md` as the canonical entity file, when the {entity_label} produces per-stage artifacts (draft versions, transcripts, outputs) that belong alongside the tracker.

Slugs are lowercase, hyphens, no spaces. Example: `my-feature-idea.md` or `my-feature-idea/index.md`. The status scanner recognizes both forms; `--set` and `--archive` resolve the slug either way, and folder entities archive as a whole folder into the workflow's archive directory.

## Schema

Every {entity_label} file has YAML frontmatter. Fields are documented below; see **{Entity_label} Template** for a copy-paste starter.

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier, format determined by id-style in README frontmatter |
| `title` | string | Human-readable {entity_label} name |
| `status` | enum | One of: {stages as comma-separated list} |
| `source` | string | Where this {entity_label} came from |
| `started` | ISO 8601 | When active work began |
| `completed` | ISO 8601 | When the {entity_label} reached terminal status |
| `verdict` | enum | PASSED or REJECTED — set at final stage |
| `score` | number | Priority score, 0.0–1.0 (optional). Workflows can upgrade to a multi-dimension rubric in their README. |
| `worktree` | string | Worktree path while a dispatched agent is active, empty otherwise |
| `issue` | string | GitHub issue reference (e.g., `#42` or `owner/repo#42`). Optional cross-reference, set manually. |
| `pr` | string | GitHub PR reference (e.g., `#57` or `owner/repo#57`). Set when a PR is created for this entity's worktree branch. |

### ID Style

The `id-style` frontmatter setting controls the operator-facing ID strategy:

- `sequential`: `id` is required and stores the next zero-padded numeric value from `status --next-id`, counting active and archived entities.
- `sd-b32`: `id` is required and stores the full stable 24-character lowercase SD-B32 stored ID from `status --next-id --id-seed <slug-or-title>`. SD-B32 is Spacedock Base32: SHA-256 digest material formatted with Spacedock's human-safe alphabet `0123456789abcdefghjkmnpqrstvwxyz`. Status tables show shorter display/address prefixes computed from active plus archived entities. `status --boot` reports `ID_STYLE: sd-b32`, `NEXT_ID: {candidate}`, and `MIN_PREFIX: 2`.
- `slug`: `id` is optional; the effective ID is the entity slug. `status --next-id is not applicable for id-style: slug` because the slug comes from the title.

SD-B32 display/address prefixes can lengthen after another branch adds a colliding prefix, while stored IDs remain stable. Use `status --validate` before trusting workflow state and `status --resolve <ref>` to resolve slugs, stored IDs, or sd-b32 address prefixes.

Copyable README frontmatter examples:

```yaml
id-style: sequential
```

```yaml
id-style: sd-b32
```

```yaml
id-style: slug
```

SD-B32 examples:

| Workflow size | Stored `id` examples | Display/address examples |
| --- | --- | --- |
| 10s of entities | `4k9q2m7x8c3v9r5t6w2p0n1h`, `8t5n0p2w6j9r4c8x1m7q3v5k` | `4k`, `8t` |
| 100s of entities | `9m2c7v4xq8j3h6t0p5w1r8n2`, `9m2cq8j3h6t0p5w1r8v7x4kn` | `9m2c7`, `9m2cq` |
| 1000s of entities | `v7k3q9x2m5c8h6t0p1w4r8n2`, `v7k3qrv5t9p3j6n2w8c4x1mk` | `v7k3q9`, `v7k3qr` |

Generated IDs make concurrent and offline creation safer because creators do not share a central counter. Migration from existing sequential workflows is manual migration in this release: validate the target style, update README/entity frontmatter deliberately, and defer rewrite automation to a separate tracked task.

## Stages

{For EACH stage in the ordered list, generate a subsection:}

### `{stage_name}`

{If this stage has a `skill:` property: "**Load skill:** Invoke `Skill(\"{skill_value}\")` before starting stage work." Otherwise omit this line.}

{A sentence describing who sets this status and what it means for an {entity_label} to be in this stage.}

- **Inputs:** {What the worker reads to do this stage's work — be specific to the mission}
- **Outputs:** {What the worker produces — be specific to the mission. Keep bullets concise and verifiable — these become checklist items at dispatch time. Focus on non-obvious requirements that catch skipping, not obvious actions like "write code." Stage-output bullets become checklist items at dispatch; any entity-level end-state properties the stage produces belong under the entity body's `## Acceptance criteria` heading, not in the stage Outputs.}
- **Good:** {Quality criteria for work done in this stage}
- **Bad:** {Anti-patterns to avoid in this stage}

{End of per-stage sections.}

## Scoring

{ONLY include this section if {captain} explicitly requests a multi-dimension rubric. Otherwise omit entirely — the 0.0–1.0 float is self-explanatory from the schema.}

## Workflow State

Workflow state is read by the first officer at boot. To view current state, dispatch the first officer or run it directly:

```
claude --agent spacedock:first-officer
```

## {Entity_label} Template

```yaml
---
id:
title: {Entity_label} name here
status: {first_stage}
source:
started:
completed:
verdict:
score:
worktree:
issue:
pr:
---

Brief description of this {entity_label} and what it aims to achieve.

## Acceptance criteria

Each AC names a property of the finished entity (not a stage action) and how it is verified.

**AC-1 — {End-state property.}**
Verified by: {grep / test name / file path / command a future reader can reproduce.}
```

## Commit Discipline

- Commit status changes at dispatch and merge boundaries
- Commit {entity_label} body updates when substantive
````

### 2b. Generate Seed Entities

For each seed entity, create `{dir}/{slug}.md` where `{slug}` is the title converted to lowercase with spaces replaced by hyphens, non-alphanumeric characters (except hyphens) removed.

The `title` field is the human-readable name (e.g., "Full Cycle Test"). The filename `{slug}.md` is derived from it (lowercase, hyphens).

Set the `id` field according to `{id_style}`:

- `sequential`: assign IDs starting at `001`, zero-padded to 3 digits.
- `sd-b32`: call the plugin status viewer with `--next-id --id-seed "{slug-or-title}"` immediately before writing each entity and store the returned 24-character SD-B32 stored ID. This value is not a reservation; call again for the next entity.
- `slug`: omit `id` or leave it blank; the slug is the effective ID.

```markdown
---
id: {strategy-dependent id value, or blank for id-style: slug}
title: {entity title — human-readable, not the slug}
status: {first_stage}
source: {source if provided, otherwise "commission seed"}
started:
completed:
verdict:
score: {score, or leave empty}
worktree:
issue:
pr:
---

(Body: write the description or thesis from the captain's seed input as plain prose. Do NOT carry brace-syntax placeholders into the body — rewrite any `{var}` phrasing as natural language or backticked words.)
```

### 2c. Install Mods (conditional)

Mod **framing prose** was already surfaced during Phase 1's Trait Detection step (per the selected template's Adoption section, or the generic Layer framing reference for the layer-assembly fallback). This step is the file-generation install confirmation only — the captain already knows *why* the mod is being offered.

Check the README frontmatter for any stages with `worktree: true`. If at least one stage uses a worktree, confirm the pr-merge install:

> Install the **pr-merge** mod now? (We discussed this during design — it manages the PR lifecycle so the workflow's stage list stays clean.)
>
> (y/n, default: y)

In batch mode, install pr-merge by default for workflows with worktree stages. If the user explicitly says "no mods" or "no pr-merge", skip.

If installing, copy the mod:

```bash
mkdir -p {dir}/_mods
cp "{spacedock_plugin_dir}/mods/pr-merge.md" {dir}/_mods/pr-merge.md
```

If no stage uses a worktree, skip the pr-merge confirmation entirely. For other layer-tied mods (e.g., silence-watcher when the parked-stages layer fired with timeout/nudge semantics), confirm the install here using the same pattern: a brief y/n callback to the Phase 1 framing, default y when the layer fired during design.


### Generation Checklist

After generating all files, verify before proceeding:

- [ ] `{dir}/README.md` exists with mission, schema, all stage definitions, and {entity_label} template
- [ ] Each seed entity file exists at `{dir}/{slug}.md` with valid YAML frontmatter
- [ ] `{dir}/_mods/pr-merge.md` exists (only if a worktree stage exists and pr-merge was accepted)
- [ ] `.worktrees/` is in `{project_root}/.gitignore`

### Agent Warnings

After generation, check the README frontmatter for any stages with an `agent:` property. For each such referenced agent, check whether `{project_root}/.claude/agents/{agent}.md` exists. If a referenced agent file does not exist, warn {captain}:

> Stage '{stage_name}' references agent '{agent}' but `{project_root}/.claude/agents/{agent}.md` does not exist. You'll need to create this file before running the workflow.

This is a warning, not a blocker — proceed with the pilot run regardless. The first officer will fall back to dispatching `ensign` if the referenced agent file is not found at runtime.

---

## Phase 3: Pilot Run

After all files are generated and verified, launch the pilot run.

### Step 1 — Announce

Tell {captain} what was generated:

> Workflow generated! Here's what I created:
>
> - `{dir}/README.md` — workflow schema and stage definitions
> - {for each seed entity: "`{dir}/{slug}.md` — {title}"}
> - {if pr-merge mod was installed: "`{dir}/_mods/pr-merge.md` — PR merge mod"}

Then surface the **README-edit nudge** — a one-paragraph reminder that the per-stage prose is a starting point, not a commitment, and that the captain should tighten it before the first dispatch. The README is the living spec for what each stage means in this workflow; if the auto-generated bullets do not match the captain's actual quality bar, every dispatched agent will work from prose that is wrong-by-default. Edit time before first dispatch is cheap; edit time after agents have been dispatched against the wrong bar is not.

> Quick heads-up before we start: the README I just generated is the **living spec** for this workflow. Each stage in `{dir}/README.md` has three per-stage bullets — `Outputs:` (what the worker produces), `Good:` (your quality bar), and `Bad:` (anti-patterns to avoid). I drafted those as best-guesses from the mission text, but they are not commitments — they are starting prose for you to tighten so they reflect your actual standards. Open `{dir}/README.md` and edit the bullets under each `### {stage_name}` heading before the first dispatch. Tightening costs minutes now; un-tightening after agents have been dispatched against vague bullets costs more.

Then offer the **`review stages`** interactive flow as an opt-in alternative to opening the file in an editor:

> Type `review stages` if you'd like me to walk you through each stage's expectations one at a time and offer amendments inline — otherwise we'll proceed to the pilot run with the README as-is.

Wait for {captain} to either type `review stages` (the literal trigger phrase) or signal proceed (any other response, including "go", "proceed", "looks good", or just continuing the conversation). If the trigger fires, hand off to **Step 1a — Review Stages Handler** below; otherwise continue with the agents/launch announcement:

> Agents are shipped with the Spacedock plugin — no local agent files needed:
> - `spacedock:first-officer` — workflow orchestrator
> - `spacedock:ensign` — stage worker agent
>
> To run this workflow in future sessions, start Claude Code with:
>
> ```
> claude --agent spacedock:first-officer
> ```
>
> Starting the initial run now...

### Step 1a — Review Stages Handler (triggered by `review stages`)

When {captain} types the literal phrase `review stages`, walk them through each stage of the generated README one at a time, offering amendments per stage. **Progressive disclosure**: never dump the whole README at once. The goal is for a captain who has never edited a Spacedock README to feel guided rather than interrogated.

#### Pre-pass: scan for stretch bullets

Before starting the per-stage walk, read `{dir}/README.md` and scan each stage's `Outputs:` / `Good:` / `Bad:` bullets for **stretch bullets** — auto-generated content that is generic enough to be a candidate for tightening regardless of the workflow specifics. Flag candidates include:

- Outputs bullets that read as generic verbs (`produce deliverable`, `generate output`, `complete the work`)
- Good bullets that are platitudes (`high quality`, `well-written`, `correct`, `addresses the problem`)
- Bad bullets that are tautologies (`low quality`, `incomplete`, `wrong`, `does not address the problem`)
- Any bullet whose specifics could be lifted unchanged into a different workflow's README — that is a sign the bullet is not workflow-specific yet

Build a per-stage flag list. When you reach a stage during the walk, surface its flagged bullets proactively rather than waiting for {captain} to notice:

> Heads up — the `Outputs:` bullet "produce the deliverable" reads as generic. For *this* workflow, what does the deliverable actually look like at this stage? (e.g., a merged PR with passing CI, a transcript file at `{path}`, a row in `{table}` with status=done.)

This is the same mixed-inference / explicit-ask discipline as Trait Detection: when the auto-generated bullet is clearly a stretch, infer it needs tightening and proactively prompt; when it is plausibly workflow-specific already, present it without flagging and let {captain} decide.

#### Per-stage walk

For each stage in the README's `## Stages` section (in order), surface a focused view:

> **Stage `{stage_name}`** ({position} of {total})
>
> What the {entity_label} is sitting in: {one-line bucket-noun framing — pull from the stage's first sentence in the README}
>
> **Outputs** (what the worker produces):
> {for each Outputs bullet: "- {bullet}{if flagged: " *— flagged: {flag reason}*"}"}
>
> **Good** (your quality bar):
> {for each Good bullet: "- {bullet}{if flagged: " *— flagged: {flag reason}*"}"}
>
> **Bad** (anti-patterns to avoid):
> {for each Bad bullet: "- {bullet}{if flagged: " *— flagged: {flag reason}*"}"}
>
> What would you like to do? Options:
> - `keep` — accept this stage as-is, move on
> - `tighten outputs` — I'll ask what to replace each Outputs bullet with
> - `tighten good` — same for Good
> - `tighten bad` — same for Bad
> - `drop {section} {n}` — remove bullet {n} from {section} (e.g., `drop outputs 2`)
> - `add {section}: {text}` — append a new bullet to {section} (e.g., `add good: passes CI on first try`)
> - `next stage` — same as `keep`, move on
>
> Or describe what you want changed in your own words and I'll apply it.

Wait for {captain}'s response. Apply the requested changes by `Edit`-ing `{dir}/README.md` in place — anchor on the stage's `### {stage_name}` heading and the bullet text being modified. After each change, briefly confirm what was applied ("Tightened Outputs bullet 2: `produce deliverable` → `merged PR with passing CI`") and then re-prompt with the same options until {captain} chooses `keep` / `next stage`.

Track per-stage which bullets were `tighten`-ed vs kept as-is — this feeds the final confirmation.

#### Final confirmation

Once all stages have been reviewed (each one moved past via `keep` or `next stage`), summarize what got tightened:

> Stage review complete. Here's what got tightened:
>
> {for each stage that had any change:}
> - `{stage_name}`: {short description of what changed — e.g., "Outputs bullet 1 tightened, Bad bullet 3 dropped"}
>
> {if no stages were changed: "All stages kept as-is."}
>
> The README at `{dir}/README.md` reflects these edits. Ready to proceed to the pilot run? (yes / let me make more edits)

If {captain} says yes (or any proceed signal), continue with the agents/launch announcement from Step 1 (the part after the `review stages` offer). If {captain} wants more edits, ask what they want to change and apply directly — do not re-walk the per-stage flow unless they ask for it explicitly.

### Step 2 — Assume First-Officer Role

Do not spawn a subagent. Instead, the commission skill itself takes on the first-officer role for the initial run:

1. Read the first-officer agent file at `{spacedock_plugin_dir}/agents/first-officer.md`.
2. Follow its instructions: read the reference files (shared core, guardrails, Claude runtime), then read the workflow README, run the status script, and dispatch agents for entities ready to advance.

Execute the first-officer startup procedure directly. You are now the first officer for the remainder of this session.

### Step 3 — Team Probe

Before any dispatch:

1. Run `ToolSearch(query="select:TeamCreate", max_results=1)`.
2. If the result contains a TeamCreate definition, run `TeamCreate(...)` per the Claude Code runtime adapter's Team Creation section, and record the returned `team_name`. Forward that `team_name` into every subsequent dispatch input JSON.
3. If ToolSearch returns no match, enter bare mode explicitly (`team_name: null, bare_mode: true` on dispatch inputs) and report the mode to {captain}.

This step is mandatory. Skipping it and defaulting to bare is the failure mode #201 addresses — a commissioned FO that silently omits TeamCreate loses access to team-mode primitives (spawn-standing, concurrent dispatch, SendMessage coordination).

### Step 4 — Monitor and Report

Process entities following the first-officer event loop. When the workflow reaches an idle state or pauses at an approval gate, report the results to {captain}:

> **Pilot Run Results**
>
> {Summary of what happened: which entities were processed, what stages they moved through, any approval gates hit}

### Step 5 — Handle Failures

If the pilot run fails (agent errors, YAML gets mangled, dispatch issues):

- Report exactly what happened, including any error messages
- Show the current state of the workflow with `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}`
- Do not retry automatically — let {captain} decide next steps

This is v0. Either it works or we learn why it didn't.

### Step 6 — Post-Completion Guidance

After Step 4 or Step 5 (whether the pilot run succeeded or failed), always conclude with:

> **What's next?** To continue working this workflow in a future session, start Claude Code with:
>
> ```
> claude --agent spacedock:first-officer
> ```
>
> The first officer will read the workflow state, pick up where things left off, and dispatch agents for any entities ready for their next stage.

---

## Layer framing reference

Used only when trait detection landed in the layer-assembly fallback (no template selected). When a template was selected, framing prose comes from that template's `## Adoption` section, not from this reference.

### repo-mutation layer (no template)

> One or more of your stages modifies the repo. I'll mark those stages with `worktree: true` so each entity gets a dedicated worktree branch and the main checkout stays clean. If your shipping ritual is "open a PR, get review, merge to main," I'd suggest installing the **pr-merge** mod — it tracks PR state on the entity's `pr` field, watches for merges, and advances the entity to terminal when the PR lands. That removes the need for a `pr_open` or `awaiting_merge` stage in your stage list. Skip the mod if you commit directly to main or do not ship via PR review.

### parked-stages layer (no template)

> One or more of your stages waits on external response, evidence accumulation, or time passing. I'll mark those stages parked so the FO knows entities sitting there are normal, not stalled. If a parked stage has "ping me after N days" or "auto-advance after timeout" semantics, the **silence-watcher** idle-mod handles that — it watches for stall thresholds you set per parked stage and either pings the captain or advances the entity per your rule. Skip the mod if no parked stage has timeout semantics — silence-watcher only earns its keep when "stuck for too long" is a real concern.
