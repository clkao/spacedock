---
commissioned-by: spacedock@template
entity-type: artifact
entity-label: artifact
entity-label-plural: artifacts
id-style: sequential
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: draft
      initial: true
    - name: review
      gate: true
      feedback-to: draft
    - name: polish
    - name: done
      terminal: true
---

# Refinement Workflow Template

Iterate on an artifact through stages of improvement until it is locked. This is the universal base shape: an entity is drafted, reviewed for whether it is ready, polished if accepted, and marked done when finished. No layers active.

Use this template when the captain's mission is to track an artifact through rounds of improvement with a human-in-the-loop quality bar — design docs, PRDs, content pieces, outreach replies, integration records, anything where the work is "make this thing good enough." Variants in the Adoption section adapt the stage list and entity body to common end-use shapes (outreach, integration, content production, PRD authoring) without changing the underlying structure.

## File Naming

Each artifact lives as either:

- a flat markdown file `{slug}.md` (default — use this unless the artifact produces many side files), or
- a folder `{slug}/` containing `index.md` as the canonical entity file, when the artifact produces per-stage attachments (drafts, reviewer notes, transcripts, output files) that belong alongside the tracker.

Slugs are lowercase, hyphens, no spaces. Example: `q3-launch-narrative.md` or `q3-launch-narrative/index.md`.

## Schema

Every artifact file has YAML frontmatter. Fields are documented below; see **Artifact Template** for a copy-paste starter.

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier, format determined by id-style in README frontmatter |
| `title` | string | Human-readable artifact name |
| `status` | enum | One of: draft, review, polish, done |
| `source` | string | Where this artifact came from |
| `started` | ISO 8601 | When active work began |
| `completed` | ISO 8601 | When the artifact reached terminal status |
| `verdict` | enum | PASSED or REJECTED — set at final stage |
| `score` | number | Priority score, 0.0–1.0 (optional) |
| `worktree` | string | Worktree path while a dispatched agent is active, empty otherwise |
| `issue` | string | GitHub issue reference (optional cross-reference) |
| `pr` | string | GitHub PR reference (set when a PR is created) |

## Stages

### `draft`

The artifact is being produced or revised. Work happens here every time the artifact enters the loop, including after a review bounce.

- **Inputs:** The brief, prior reviewer notes (if this is a re-entry from review), source material
- **Outputs:** A complete artifact body ready for review
- **Good:** Addresses the brief, integrates prior feedback when re-entering, ready to be evaluated end-to-end
- **Bad:** Half-finished sections, ignores reviewer notes from the prior round, produces something that cannot be evaluated as a whole

### `review`

A reviewer reads the draft and decides: accept (advance to polish) or bounce back to draft with notes. This is an approval gate.

- **Inputs:** The draft artifact
- **Outputs:** Reviewer notes captured in the artifact body; either gate-approval to polish or rejection back to draft with concrete notes
- **Good:** Specific, actionable notes; clear accept/reject decision; the reviewer reads the whole draft before deciding
- **Bad:** Vague "this needs work" with no concrete asks, accepting things that have not been read carefully, deferring the decision indefinitely

### `polish`

Final cleanup before the artifact is locked: formatting, copy edits, last-pass consistency, nothing structural.

- **Inputs:** The accepted draft from review
- **Outputs:** A polished artifact ready to be marked done
- **Good:** Cosmetic improvements only, preserves the substance the reviewer accepted
- **Bad:** Reopening structural decisions, introducing new content that should have gone through review

### `done`

Terminal state. The artifact is locked and considered shipped (or filed, depending on the variant).

- **Inputs:** The polished artifact
- **Outputs:** None — this is a terminal state. Mark `completed`, set `verdict: PASSED`, and archive.
- **Good:** Clean handoff, terminal state set deliberately
- **Bad:** Reopening a done artifact instead of starting a new one

## Workflow State

View the workflow overview:

```bash
{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}
```

Output columns: ID, SLUG, STATUS, TITLE, SCORE, SOURCE.

Find dispatchable artifacts ready for their next stage:

```bash
{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir} --next
```

## Artifact Template

```yaml
---
id:
title: Artifact name here
status: draft
source:
started:
completed:
verdict:
score:
worktree:
issue:
pr:
---

Brief description of this artifact and what it aims to achieve.

## Draft

The current draft of the artifact lives here.

## Review notes

Reviewer notes accumulated across review rounds.

## Final

The locked, polished version (filled in at polish or done).
```

## Commit Discipline

- Commit status changes at dispatch and merge boundaries
- Commit artifact body updates when substantive

## Adoption

### Pre-fill stages

```yaml
- name: draft
  initial: true
- name: review
  gate: true
  feedback-to: draft
- name: polish
- name: done
  terminal: true
```

The default stage list above. Variants below adjust this list.

### Apply layers

None by default. The base refinement shape uses no structural layers; entities never touch the repo and never sit parked waiting on external events.

### Offer mods

None by default. Variants that activate the parked-stages layer (e.g., outreach with a `watching` stage) trigger the silence-watcher offer through the layer mechanism in the commission skill, not through this template.

### Inject entity-template snippet

Use the refinement snippet (draft / review notes / final) shown in the Artifact Template section above. Variants override this with their own snippet (see below).

### Surface variants

Refinement is the universal base — many common workflow shapes are refinement with adjusted stages and a different entity body. When trait detection lands on `refinement` and the cues match a variant below, surface it as a confirmation:

> Looks like an outreach pipeline. I'd suggest these stages instead of the default `draft / review / polish / done`. Want me to use this variant?

**outreach** — captain mentions contacts, leads, sending, followups, replies, drip, pipeline

- Stages: `research (initial)` → `draft (gate)` → `sent` → `watching (parked, gate)` → `followup (feedback-to: watching)` → `closed (terminal)`
- Layers: parked-stages (on `watching`)
- Mods: silence-watcher offered for `watching` (timeout/nudge semantics)
- Entity-template snippet: contact / message draft / sent-at / response / outcome

**integration** — captain mentions sync, ingest, enrichment, external system, record-by-record processing

- Stages: `intake (initial)` → `enrichment` → `sync (gate)` → `archived (terminal)`
- Layers: none (or repo-mutation if the sync target is a repo file)
- Entity-template snippet: incoming record / enrichment notes / sync target / sync result

**content-production** — captain mentions blog posts, articles, videos, publishing, editing, copy

- Stages: `drafting (initial)` → `editing (gate, feedback-to: drafting)` → `polish` → `shipping (gate)` → `published (terminal)`
- Layers: none (or parked-stages if `shipping` waits on an external publish window)
- Entity-template snippet: artifact draft / editor notes / final / publish target / published-at

**prd-authoring** — captain mentions design doc, PRD, RFC, spec, locked, approved

- Stages: `draft (initial)` → `review (gate, feedback-to: draft)` → `locked (terminal)`
- Layers: none
- Entity-template snippet: problem / proposal / open questions / decision (the final locked record)

If trait detection lands on `refinement` but the cues do not match any variant above, use the default stage list and the refinement snippet from the Artifact Template section.

### Confirmation prose

Surface this in Phase 1 once the template is selected (substituting the chosen variant if one fired):

> I'll set this up as a **refinement** workflow{ — variant: {variant}}: each {entity_label} moves through `{stage_list}` until it is locked. No worktree stages and no PR/merge ritual — this workflow does not touch the repo.{ Parked-stages layer fires on `{parked_stages}` because the entity sits waiting on external response.{ I'll offer the silence-watcher mod when we get to mod offers.}}
