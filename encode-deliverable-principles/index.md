---
id: v1awnfhs996ykymv409anywh
title: Encode deliverable principles + template ergonomics into README, FO/ensign contract, and status guards
status: backlog
source: FO triage (2026-06-01) — docs/dev/_proposals cleanup; consolidates the deliverable-principles study + the TDD/template-adoption ergonomics
started:
completed:
verdict:
score: "0.30"
worktree:
issue:
---

Encode four deliverable principles + the template-ergonomics snippets into the workflow's
own contract surfaces so future dev work cannot drift into the antipatterns they forbid.
Seeded from the parked design study in `proposal.md` (formerly
`docs/dev/_proposals/encoding-deliverable-principles.md`) — which carries the exact
before/after wording, the dogfood/falsifiability analysis, and the placement map.

The four principles: (1) **no doc-only deliverable** — every AC's oracle is EXTERNAL to the
entity body; (2) **proof is behavioral, not grep** — exercise the behavior and observe the
outcome (substring present/absent is not proof; invariant-over-real-values is); (3) **enforce
in code, not prose** — a prose-only contract has ceiling "wording present" and can't alone
satisfy an AC; (4) **spike the riskiest unknown in ideation** — smallest end-to-end exercise of
the riskiest path first, recording behavioral evidence or `no spike needed: {mechanisms}`.

## Placement (the FO-contract vs README split)

- **FO operating contract** (`skills/first-officer/references/first-officer-shared-core.md`,
  cross-workflow): P1 AC-cross-check clause (oracle must be external; pure-decision → roadmap/ADR,
  not terminal PASSED); P3 "code-gate preference" subsection; P4 spike rule in
  `## Probe and Ideation Discipline` (refs the `running-research-spikes` skill).
- **Ensign contract** (`skills/ensign/`): the TDD authoring-discipline line — per the PORTABILITY
  correction, TDD must live in the shipped ensign contract, NOT global CLAUDE.md (a clean-room
  self-hosted session has no CLAUDE.md).
- **Workflow README** (`docs/dev/README.md`, workflow-specific): P1 in ideation Outputs + done Bad;
  P2 in the ideation "choose proof" bullet + validation Bad; P4 in ideation Outputs + the Staff-review
  trigger; one P3 bullet in ideation Outputs. Plus template ergonomics: a `## Out of scope` template
  slot, promote `## Problem`/`## Proposed approach`/`## Test plan` to template headings, and a
  `spacedock status --workflow-dir docs/dev --next` doc example.
- **Code (this project's launcher only)**: P1's `status --validate` self-oracle lint + the
  terminal-PASSED `--set` guard (modeled on the mod-block/merge-hook invariant); P4's optional
  ideation-gate presence-check.

## Reconciliation required at ideation (do not double-file)

- **P1's code guard is already filed as `2a` (require-external-proof-guard).** This entity owns the
  WORDING (README + contract) and coordinates with 2a for the guard; it must not re-file the guard.
- **`se` (#248, archived PASSED — "ship the team's proven working habits in the tool's own
  instructions") already shipped working-principles prose into the contract.** Ideation MUST diff the
  shipped contract to land only the GAP — clearly P4-spike, P2 README wording, the template ergonomics,
  and the PORTABILITY/TDD-in-ensign line — without duplicating what `se` shipped.
- The TDD/template-adoption study's headline ("rely on global CLAUDE.md for TDD") is SUPERSEDED by the
  captain's PORTABILITY correction; only its template-ergonomics findings survive (captured above).

## Scaffolding-guardrail note

README, `skills/first-officer/references/`, and `skills/ensign/` are protected scaffolding — every
edit goes through this tracked entity dispatched to a worker, never an FO hand-edit. Contract edits
author in the VENDORED copy first (this project is the leading edge), then flow upstream to canonical
`~/git/spacedock` as part of self-hosting; README edits are project-only; the code guard lands only
here (it is the launcher the plugin ships). 0.19.3-class — off the 0.19.2 critical path.
