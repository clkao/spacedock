# Sprint notes — pending sprint-end actions (FO)

## AT SPRINT END (all entities done): parallel antipattern reviews
Captain directive (2026-05-30): when the sprint completes, dispatch PARALLEL reviews
for antipatterns — over-abstraction, over-engineering, tautological/grep tests — with
TWO senior personas: a senior STAFF SOFTWARE ENGINEER and a senior AI ENGINEER.
Read-only adversarial audits over the merged result; report findings before declaring
the sprint done.

## Deliverable-principles encoding (proposal pending disposition)
docs/dev/_proposals/encoding-deliverable-principles.md — senior-eng proposal for
encoding the four principles (no doc-only; behavioral-not-grep; enforce-in-code;
spike-in-ideation) into the workflow README + operating contract, with a code guard
(`status --validate` self-oracle lint + terminal-PASSED `--set` guard). Captain to
decide: file as a tracked contract-hardening entity (recommended — the code guard is
the behavioral oracle) vs bank for the next refit.

## TDD correction + PORTABILITY principle (captain, 2026-05-30)
The template-tdd-comparison workflow concluded "rely on global CLAUDE.md" for TDD.
SUPERSEDED — that is WRONG: CLAUDE.md is per-user/private; a clean-room self-hosted
session (the sprint goal) has no such file. Corrected position:
- Encode TDD authoring discipline in the PORTABLE shipped contract — the
  `spacedock:ensign` operating contract that governs every dispatched worker — NOT in
  global CLAUDE.md. (The gate still enforces behavioral-proof P1/P2; TDD is the
  worker-side authoring rule in the ensign skill.)
- Name the generalizing meta-principle PORTABILITY / SELF-SUFFICIENCY: the shipped
  contract must assume nothing from the user's environment (no global CLAUDE.md, no
  Python-on-PATH, no plugin-private paths). Same family as the Python-free and
  zero-plugin-private-path goals. The comparison workflow's lenses never questioned
  CLAUDE.md universality — a portability blind spot (argues for the end-of-sprint
  senior-eng antipattern reviews).
- Refit scope now: P1/P2/P4 wording + P1 code guard + ergonomics snippets (## Out of
  scope slot, promoted ## Problem/## Proposed approach/## Test plan headings, native
  --next doc) + a TDD authoring line in the ensign contract + a PORTABILITY principle
  in the operating contract + a scan for other global/environment-reliance leaks. ALL
  in the shipped contract. One refit.

## Debrief notes
- external-tracker-checkpoint shipped PASSED but AC-6 was a prose self-oracle (the
  doc-only antipattern) — kept as the live example that motivated the principles.
