---
id: 221
title: "Commission should suggest common workflows"
status: ideation
source: "GitHub issue #156 (filed by CL, 2026-04-28)"
started: 2026-04-28T21:17:43Z
completed:
verdict:
score: 0.5
worktree:
issue: "#156"
pr:
mod-block:
---

## Problem

The commission skill (`skills/commission/SKILL.md`) walks the captain through a from-scratch design conversation every time. When a captain wants something common — track development tasks, run experiments, log a daily routine, sync with a CRM — they re-derive the same structure each time, and the result depends on whether they happen to remember the right defaults (sd-b32 for multi-player repos, worktree for code-mutating stages, folder entities for artifact-heavy work, the pr-merge mod as the standard finish-development hook).

There are also recurring quality drifts the skill does not currently prevent:

- **Stage names drift toward event-flavored adjectives** like `reviewed`, `approved`, `merged`, or verbose in-progress noise like `awaiting_validation` and `in_review`. Stage names should be activity-or-bucket nouns describing where the entity is sitting (the entity-level end state at that bucket is determined by the existence of the corresponding stage report, not by past-participle naming).
- **Captains pollute their stage list** with `pr_open` or `awaiting_merge` stages instead of using the pr-merge mod, because the mod is currently surfaced as a y/n at file-generation time rather than as a teaching moment during design.

## Proposed approach

Three concrete additions to the commission skill, plus four starter templates:

### 1. Trait detection during the design conversation

Add a **Trait Detection** step to commission Phase 1 (between mission/entity in Q1 and stages in Q2). The skill carries a small trait → scaffolding-consequence table and applies a mixed inference strategy:

- When the mission text or captain answers carry a strong signal, **infer silently and surface as a confirmation** (e.g., "This sounds like a multi-player repo, so suggesting `id-style: sd-b32` — keep or change?").
- When the signal is ambiguous, **ask the trait question explicitly**.

Traits and consequences:

| Trait | Strong-signal cues | Consequence |
|---|---|---|
| Multi-player repo | "team", "concurrent", "branches", "worktree", multiple human or agent creators | `id-style: sd-b32` |
| Named-entity (slug = identity) | dated entries, project-name entries, known canonical names | `id-style: slug` |
| Stage mutates the codebase | "implement", "build", "ship", "code", "PR" | `worktree: true` on that stage; offer pr-merge mod with the stages-stay-clean framing |
| Entity has many per-stage artifacts | "transcripts", "outputs", "data", "drafts", "exports" | folder-based entity (`{slug}/index.md`) |
| Stage requires human interaction | captain review, captain decision, "I'll review" | stage-level hint pointing at the captain-interaction guidance (per #215 if/when that lands; otherwise a noted open question deferred to that task) |

If no strong signal exists for sd-b32 vs slug vs sequential, the existing default selection in commission Phase 1 still applies (`sequential` for single-writer, `sd-b32` recommended for collaborative). Traits do not override the captain — they pre-fill the suggestion that the captain confirms.

### 2. Starter templates at `skills/commission/references/templates/`

Four templates ship as full workflow READMEs the commission skill can offer as starting points and the captain can clone-and-customize. Each template is a complete, valid workflow that demonstrates a different combination of traits.

Each template lives as `skills/commission/references/templates/{name}.md` and is a full README following the schema in commission Phase 2a — frontmatter with `entity-type`, `entity-label`, `id-style`, `stages` block; body with File Naming, Schema, Stages (Inputs/Outputs/Good/Bad per stage), Workflow State, and Entity Template sections.

The four templates:

**`development`** — build software (mirrors spacedock's own self-workflow).
- Stages: `backlog` (initial, gate) → `ideation` (gate) → `implementation` (worktree) → `validation` (worktree, fresh, feedback-to: implementation, gate) → `done` (terminal)
- Traits demonstrated: `id-style: sd-b32`, worktree on impl/validation, pr-merge mod installed, flat entity files

**`experiment`** — run something, learn from it.
- Stages: `hypothesis` (initial) → `setup` → `run` → `analysis` (gate) → `done` (terminal)
- Traits demonstrated: `id-style: sd-b32`, no worktree, folder-based entity (each experiment carries data, plots, transcripts, writeup), no pr-merge

**`daily-routine`** — recurring habit / journal-shaped.
- Stages: `intent` (initial) → `log` → `reflection` (terminal)
- Traits demonstrated: `id-style: slug` (slug-as-identity, e.g. `2026-04-28.md`), flat files, no worktree, no pr-merge
- Deliberately thin: three buckets is the point — captains see that workflows do not have to be elaborate. The terminal `reflection` (not `done`) demonstrates that the terminal stage is just whatever bucket the entity finally rests in.

**`working-with-crm`** — integration-shaped.
- Stages: `intake` (initial) → `enrichment` (gate) → `sync` → `archive` (terminal)
- Traits demonstrated: `id-style: sd-b32`, folder-based entity (each record carries pulled-data + enriched-data + sync-log), no worktree (CRM integrations push to an external system rather than mutating the repo), no pr-merge mod

### 3. Stage-naming convention codified in the commission skill

Add a **Stage Naming Convention** subsection to the commission SKILL.md (in Question 2 — Stages). The rule:

> **Stage names are nouns describing the activity-or-bucket the entity is sitting in.** The test: you can naturally say "the entity is in `{name}`." End states at each bucket are determined by the existence of the corresponding stage report, not encoded into the name.
>
> **Good** (activity / state nouns): `backlog`, `ideation`, `implementation`, `validation`, `analysis`, `archive`, `reflection`, `intake`, `sync`.
>
> **Avoid** (event-flavored past-participles, because the entity has not "just become reviewed" — it is sitting in a bucket): `reviewed`, `approved`, `merged`, `validated`, `published`.
>
> **Avoid** (verbose in-progress noise — the bucket already implies "in progress"): `being_reviewed`, `in_review`, `pending_validation`, `awaiting_merge`.
>
> **Exception**: `done` is the universally-understood terminal and stays.

When commission proposes default stages in Q2 and when it accepts captain edits, it should apply this convention and gently push back if the captain proposes a name that violates it.

### 4. Mod framing during design

Move (or echo) the pr-merge mod offer from Phase 2c (file-generation y/n) into Phase 1 (the design conversation), tied to the worktree trait firing. Frame it as the teaching moment:

> Your `{stage}` stage modifies the codebase, so I'm setting `worktree: true` on it. The standard finish-development hook is the **pr-merge mod** — it pushes the worktree branch and opens a GitHub PR when the entity reaches the merge boundary. This keeps your stages clean: you don't need a `pr_open` or `awaiting_merge` stage to model PR/merge behavior. Install the mod? (default: yes)

The Phase 2c y/n still functions as the install confirmation; the new Phase 1 framing is the *why*.

## Acceptance criteria

**AC-1 — Four starter templates exist as full workflow READMEs.**
Each of `development`, `experiment`, `daily-routine`, `working-with-crm` lives at `skills/commission/references/templates/{name}.md` and is a valid commission workflow README (frontmatter has `entity-type`, `entity-label`, `id-style`, `stages` with at least one initial and one terminal state; body has File Naming, Schema, Stages, Workflow State, and Entity Template sections).
Verified by: `for t in development experiment daily-routine working-with-crm; do test -f skills/commission/references/templates/$t.md; done` exits 0; running the existing commission Phase 2a frontmatter validation against each file succeeds (each file parses as YAML, has the required keys, and the stage list contains exactly one `initial: true` and at least one `terminal: true`).

**AC-2 — All shipped stage names follow the bucket-noun convention.**
No stage name in any shipped workflow README under `skills/commission/references/templates/` uses an `-ed` past-participle form (`reviewed`, `approved`, `merged`, `validated`, `published`) or a verbose in-progress prefix (`being_`, `in_`, `pending_`, `awaiting_`). The single allowed adjective is `done` as a terminal.
Verified by: a grep guard `! grep -E "^\s+- name: (.*ed|being_.*|in_.*|pending_.*|awaiting_.*)$" skills/commission/references/templates/*.md` returns no matches except where the matched name is exactly `done`.

**AC-3 — The commission skill teaches the stage-naming convention.**
`skills/commission/SKILL.md` contains a section titled "Stage Naming Convention" inside or adjacent to Question 2 — Stages, stating the bucket-noun rule, listing the good/avoid/exception examples from the proposed approach, and instructing the skill to apply the convention when proposing defaults and accepting captain edits.
Verified by: `grep -n "Stage Naming Convention" skills/commission/SKILL.md` returns at least one hit; visual confirmation that the section text matches the convention statement above.

**AC-4 — The commission skill performs trait detection during Phase 1.**
`skills/commission/SKILL.md` contains a "Trait Detection" step in Phase 1 (between Q1 mission/entity and Q2 stages) that lists the five traits, their strong-signal cues, and their scaffolding consequences, and instructs the skill to use the mixed inference-with-confirmation-or-explicit-ask strategy.
Verified by: `grep -n "Trait Detection" skills/commission/SKILL.md` returns at least one hit; the section enumerates all five traits from the trait table above.

**AC-5 — The commission skill frames the pr-merge mod during design (not only at generation time).**
`skills/commission/SKILL.md` describes offering the pr-merge mod in Phase 1 as the teaching moment when the worktree trait fires, with the "stages stay clean" framing. The Phase 2c y/n install confirmation can remain, but the design-phase framing is present.
Verified by: `grep -n "pr-merge" skills/commission/SKILL.md` shows a hit inside Phase 1 (not only inside Phase 2c); the surrounding prose mentions both the worktree-trait link and the stages-stay-clean rationale.

**AC-6 — The commission skill loads templates as starting points.**
`skills/commission/SKILL.md` describes how the four templates are surfaced (e.g., as a menu choice during Phase 1 intake, or as a "want to start from a template?" branch) and how a chosen template is loaded (a `Read` of the references file) and used to pre-fill stages, traits, and mod selection — which the captain can then accept or modify.
Verified by: `grep -n "templates/" skills/commission/SKILL.md` returns hits inside Phase 1; the surrounding prose names all four templates and describes the load mechanism.

## Test plan

This task ships skill prose and reference files. There is no runtime code change, so verification is primarily structural and demonstration-by-walkthrough rather than E2E.

**Static checks (cheap, fast, run in CI or pre-commit):**

- File presence and frontmatter validity for each of the four templates (AC-1).
- Stage-name grep guard against banned forms (AC-2). This guard is the durable contract that prevents convention drift.
- Skill-prose presence checks (AC-3 through AC-6) — `grep -n` lookups for the four required section markers.

**Demonstration walkthrough (manual, one-off at completion):**

A captain runs `/spacedock:commission` with a mission text designed to trigger each trait at least once, e.g., "track design ideas through ideation, implementation, and validation in a multi-contributor repo." The walkthrough verifies that:

- Trait Detection fires for the multi-player and code-mutating signals.
- The skill surfaces sd-b32 and worktree+pr-merge as confirmations rather than asking from scratch.
- The skill offers the four templates and loading the `development` template pre-fills the design correctly.
- The skill applies the stage-naming convention when generating the README.

This walkthrough is documented as a transcript fixture in the implementation task's stage report, not gated on automated E2E (commissioning is a human-in-the-loop interactive flow; an automated E2E would be expensive and would mostly verify the LLM is reading its own skill prose, which static grep checks already cover for the load-bearing structure).

**E2E:** not required for this task. Trait inference quality and template-load behavior are emergent from the skill prose; the static checks pin down the contract surface, the walkthrough catches obvious skill-prose bugs, and any future trait that needs harder testing can ship its own focused fixture.

**Estimated cost/complexity:** medium — the four templates are the bulk of the line count (~150-200 lines each), the skill-prose additions are smaller (~50-80 lines total), the grep guard is a one-line check.

## Out of scope

Confirmed during ideation, deferred to follow-ups if needed:

- **Refit/migration of existing workflows** to the new naming convention. Existing workflows keep whatever names they have; only newly-commissioned workflows get the convention enforcement.
- **Template versioning.** Templates evolve in-tree like any reference file; no per-template version pins.
- **Additional templates** beyond the four (e.g., marketing, incident response). File as separate tasks if needed.
- **Trait inference rubric as code.** The trait detection lives as prose in the commission SKILL.md; no new parser, classifier, or runtime tool.
- **Template loader as a CLI verb.** Loading a template is a `Read` of the references file at the right point in the commission flow, not a new `status --use-template` verb.
- **Captain-interaction stage hint plumbing.** AC includes mentioning the hint when the interaction trait fires, but the hint itself depends on #215 landing. If #215 is not landed at implementation time, this reduces to a noted open question rather than a wired-up reference.

## Open questions

CL noted that example workflows may be provided after this ideation gate and that we may revisit the templates then. The four template stage lists in this spec are best drafts to start from; when CL provides examples, the implementation task can adjust template content (within the same naming convention and trait demonstrations) without re-ideation.

## Stage Report: ideation

- DONE: Use the `superpowers:brainstorming` skill to drive the captain conversation.
  Brainstorming skill loaded; multi-turn dialogue with CL via direct text after CL flagged that interactive mode means direct text not SendMessage relay; scope, traits list, naming rule, template location, and inference strategy all settled in conversation.
- DONE: The fleshed-out body has the four standard ideation outputs.
  Problem statement, proposed approach (4 components), AC list (6 entity-level end-state properties each with `Verified by:`), and test plan (static checks + demonstration walkthrough + E2E rationale) all present.
- DONE: Open questions from the brainstorm are explicitly resolved (or explicitly deferred with rationale).
  Out-of-scope section enumerates 6 deferred items with rationale; one Open Questions item documents that CL may revisit template content after providing example workflows (deferred to implementation, not blocking ideation gate); no `TBD` or `??` markers remain.

### Summary

Reframed the task from "ship four templates" to "trait detection + naming convention + mod framing + four templates as worked examples", confirmed by CL through brainstorming. Locked the bucket-noun stage-naming rule (CL's reasoning: end-state is determined by existence of stage report, not by past-participle naming), set templates to live at `skills/commission/references/templates/`, and chose mixed inference (silent-with-confirmation when strong signal, explicit-ask when ambiguous). Acceptance criteria are entity-level end-state properties verified by file-presence checks, frontmatter parse, grep guards against banned name forms, and grep lookups for required commission-skill section markers; E2E is deliberately out of scope because trait quality is emergent skill-prose behavior that static contract checks already pin down.
