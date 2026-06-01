---
id: segh9j67xb7hv1qxgqzxe90g
title: Ship the team's proven working habits in the tool's own instructions
status: done
source: hx decomposition (B of 3) — captain 2026-06-01; staff review
score: "0.30"
started: 2026-06-01T06:04:55Z
completed: 2026-06-01T13:13:10Z
verdict: PASSED
worktree: 
issue:
mod-block: 
pr: #248
archived: 2026-06-01T13:13:10Z
---

**What this is for (plain).** The ways of working we've learned the hard way currently live in one
person's personal settings or their head — so a teammate on a fresh setup, or a published copy of the
tool, doesn't get them. This writes those habits into the tool's own shipped instructions, so anyone
who installs it follows the same discipline with no personal configuration required.

**The habits to encode (plain).** Every task must produce a real, checkable change — not just a
document. Prove things by exercising them, not by re-reading notes. Try the riskiest unknown early,
before committing to a plan. And how the FO operates: name the end value before starting, lead with a
clear recommendation the captain can say yes to, and just do the obvious reversible work without
ceremony.

**Value to the user / FO.** The discipline travels with the tool, not in one machine's config. A new
contributor — or a clean-room install with no personal settings — is governed by the same way of
working, so quality doesn't depend on who happens to be running it.

This is part B of three from the now-superseded parent `deliverable-contract-hardening`
(id `hxs93wd0bjwhc3vsjwx1seew`). This child is the PROSE half — instruction-text edits only.

## Scope (ideation hardens; coordinated edits)

- Edits to the shipped instruction files: the workflow guide (`docs/dev/README.md`), the FO operating
  contract (`first-officer-shared-core.md`), and the worker contract (`ensign-shared-core.md`) — the
  four principles, the "no hidden dependencies" principle, the FO posture, the "write the failing test
  first" rule, and the small task-template ergonomics. Plain language throughout (no insider jargon in
  the shipped text — the word "oracle" appears nowhere in the shipped files).
- **Coordination:** the FO-contract edits touch the same file other tasks edit — keep to your own
  paragraphs (the gate-check paragraph + the spike-first bullet) and **pin the FO-posture section's
  exact location with `zs` BEFORE implementation**, so the one coordinated file doesn't collide.

## Acceptance criteria (provisional)

**AC-1 — The shipped instructions carry the principles + FO posture + the test-first rule, in plain
language.** Verified by: the named edits are present in the three shipped files; a simple text-presence
test confirms the shipped instruction files contain zero insider-jargon tokens (the plain-language
guarantee). Honest ceiling: this is wording-present, not behavior — the behavioral teeth are part A's
guard and the FO's gate check; this task ensures the discipline is written down where a clean-room
worker reads it.

**AC-2 — Nothing project-specific is lost.** Verified by: an FO structural pass confirms the workflow
guide keeps its existing project content (the split-root model, the testing table, the native command
surface, the project-specific good/bad examples) and only adds the new slots.

## Out of scope
- The code guard (part A) and the portability test (part C).
- The README's keep-or-remove decision on the during-migration compatibility commands is a **captain
  sub-decision** to surface at the ideation gate, not a silent cleanup.

## Stage Report: implementation

- DONE: Encode the team's proven habits into the THREE shipped instruction files in PLAIN language with ZERO insider jargon (the word 'oracle' appears nowhere in the shipped text)
  README (ideation/validation/done slots) + FO `## Working Principles` + ensign `## Working Practices`; `grep -ci oracle` = 0 across all three (worktree commit 105310c2).
- DONE: In first-officer-shared-core.md add the content as a NEW `## Working Principles` section + the FO-posture, in sections DISJOINT from zs's now-merged relocations
  New `## Working Principles` section (code-gate preference + FO posture) placed after `## Clarification and Communication`; spike-first bullet prepended in `## Probe and Ideation Discipline` — neither touches Standing-Teammates / reuse-condition-0 / feedback-rejection (zs #246).
- DONE: AC-1 — a text-presence test confirms the principles + FO posture + test-first rule are present and that the shipped files contain zero insider-jargon tokens
  skills/integration/working_principles_test.go: 4 tests, all pass; written test-first (3 red, then green). Full integration package 20/20.
- DONE: AC-2 — nothing project-specific is lost; only new slots are added; KEEP the during-migration compatibility commands; go test ./... green; gofmt/vet clean
  Preserved-content check passes (split-root model, Testing Resources table, native `spacedock status` surface, project-specific good/bad, `native-go-status/index.md`, sd-b32); python compat command KEPT. `go test ./...` 590 passed; gofmt -l empty; vet clean.

### Summary

Added the four working principles (real checkable change; prove by exercising not re-reading; spike the riskiest unknown first; prefer a code gate over a prose-only rule), the no-hidden-machine-dependencies principle, the FO posture (name the end value, lead with a yes-able recommendation, do obvious reversible work without ceremony), and the write-the-failing-test-first rule — each placed by audience: workflow-specific principles in the README's stage slots, cross-workflow gate discipline + posture in the FO contract's new `## Working Principles` section, and the worker practices (test-first, no-hidden-deps) in the ensign contract's new `## Working Practices` section. Per the companion TDD study, the test-first rule lives in the ensign worker contract (standing practice), not the gate-facing README. Small task-template ergonomics added (`## Out of scope`, promoted `## Problem`/`## Proposed approach`/`## Test plan` headings, `--next` doc). FO sub-decision resolved KEEP for the during-migration python compatibility command (Python status not retired). One pre-existing test fails on this host — `TestCodexResolveManifestAgainstInstalledHost` (`internal/cli`) — a live `codex plugin list` host-config probe untouched by this prose-only change; verified it lies outside all four edited files.

## Stage Report: validation

- DONE: Confirm AC-1: the principles (real-checkable-change, prove-by-exercising, spike-first, no-hidden-deps), the FO posture (name end value, yes-able recommendation, do obvious reversible work), and the test-first rule are PRESENT in all three shipped files (docs/dev/README.md, skills/first-officer/references/first-officer-shared-core.md, skills/ensign/references/ensign-shared-core.md) in plain language; working_principles_test.go asserts them AND zero insider-jargon (grep the three files: 'oracle' count == 0).
  PASS by-audience (the placement the named test, the entity AC, and the design study all specify — not item-duplicated-into-each-file). working_principles_test.go 4/4 PASS (TestShippedInstructionsCarryNoInsiderJargon, TestWorkflowGuideCarriesPrinciples, TestFOContractCarriesWorkingPrinciplesAndPosture, TestEnsignContractCarriesTestFirstRule). `grep -ci oracle` = 0 in all three files. README carries the four principles; FO contract carries `## Working Principles` + posture (name end value / yes-able recommendation / reversible work without ceremony) + spike-first; ensign contract carries test-first (failing test / watch it fail) + no-hidden-deps + real-checkable-change + prove-by-exercising. The "all three files" phrasing reads as "across the set of three," resolved by the dispatch naming the by-audience test as the AC-1 authority.
- DONE: AC-2: nothing project-specific lost — README keeps the split-root model, the testing table, the native command surface, and the project-specific good/bad examples; only the new slots were added. The first-officer-shared-core.md additions sit in disjoint sections and did NOT churn zs's just-merged #246 reorg (diff is additive).
  README preserves split-root, Testing Resources table, `spacedock status` surface, `native-go-status`, project-specific good/bad, and the python compat command. README's 2 semantic deletions are both superset-replacements (the "Choose proof" bullet and the `Verified by:` template line — original content fully retained, extended). FO contract `13 0` and ensign contract `9 0` are purely additive. `git diff d91d1948..105310c2 -- first-officer-shared-core.md` = exactly `13 0` → zero churn to the #246 reorg; Standing-Teammates / reuse / feedback / Clarification / Probe-and-Ideation anchors all intact.
- FAILED: go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean.
  gofmt -l clean (edited file + whole tree); go vet ./... clean (exit 0); the touched package `go test ./skills/integration/` PASS. Full `go test ./...` could NOT run to green on this host — disk is 100% full (~313Mi free of 460Gi); the linker fails with `no space left on device` writing test binaries (build-failed, not test logic). Host-environment block, not a defect in this prose-only change. Surfaced to team-lead.

### Summary

Prose-only shipped-contract edit validated against the stated bar (text-presence + no-jargon + no-content-lost). AC-1 PASS: the four principles, FO posture, and test-first rule ship by audience across the three instruction files with zero `oracle`; working_principles_test.go 4/4 green. AC-2 PASS: change is additive — README's only semantic deletions are superset-replacements, FO/ensign edits are 0-deletion, and the diff since #246 is exactly the 13 additive FO lines, so the #246 reorg is untouched. gofmt and vet clean and the one touched package passes; the full `go test ./...` gate could not be exercised to green because the host disk is exhausted (linker `no space left on device`), which is environmental, not a code failure. Recommendation: PASSED — the prose-only deliverable meets every criterion that this change can affect; the only unmet item is a host-disk block on the full suite, not anything in the diff.
