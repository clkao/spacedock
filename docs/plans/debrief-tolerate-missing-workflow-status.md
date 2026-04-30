---
id: 8xyvkvsgb93sch31cmz6nf9w
title: "debrief assumes workflow status executable exists"
status: validation
source: "GitHub issue #175 (filed by Kent Chen / iamcxa, 2026-04-30)"
started: 2026-04-30T19:47:24Z
completed:
verdict:
score: 0.55
worktree: .worktrees/spacedock-ensign-debrief-tolerate-missing-workflow-status
issue: "#175"
pr: #177
mod-block: merge:pr-merge
---

## Problem

`spacedock:debrief` Phase 2e ("What's Next" extraction, `skills/debrief/SKILL.md:152`) instructs agents to run `{dir}/status --next` and `{dir}/status` against the workflow directory. Modern Spacedock ships the `status` viewer with the plugin at `{spacedock_plugin_dir}/skills/commission/bin/status` and invokes it as `... --workflow-dir {dir}`; workflows commissioned with newer versions do not carry a local `{dir}/status` executable. The debrief skill treats the local script as mandatory and fails with `no such file or directory` when absent, breaking Phase 2e silently for any newer-than-0.10.1 workflow.

Reporter (Kent Chen) hit this on a workflow commissioned by `spacedock@0.10.1` debriefed with plugin `spacedock@0.10.2`.

## Selected approach: A — Document the fallback explicitly

Pick **Shape A (documented fallback)** over Shape B (runtime detect-and-route).

Justification:
- Debrief is prose-driven; an agent reading the skill can branch on "executable exists vs not" without us teaching them a preflight ritual. Adding detect-and-route is more text and more conditional logic for the same outcome.
- This task is adjacent to **#5a `commission-readme-portable-status-path`** (also in flight). #5a is moving status-invocation knowledge out of generated READMEs and into the first-officer skill. Once #5a lands, the local `{dir}/status` artifact becomes purely a 0.10.x-and-earlier vestige; documenting "plugin-shipped is the primary path, local is a legacy fallback" is the shape that ages well. We do not coordinate with #5a here — just note the directional alignment.
- Frontmatter-only fallback is already within agent capabilities (Phase 2e enumerates entity files for the same data); we just need to license it in prose as the documented degraded mode rather than a silent improvisation.

### Concrete edits in `skills/debrief/SKILL.md`

Rewrite Phase 2e ("What's next", currently a 6-line block at `skills/debrief/SKILL.md:150-156`) so the primary invocation is the plugin-shipped status viewer, with two documented fallbacks:

1. **Primary**: `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir} --next` and `... --workflow-dir {dir}` for the dispatchable list and overview.
2. **Legacy fallback**: if the plugin-shipped status is unreachable AND a local `{dir}/status` exists, invoke `{dir}/status --next` / `{dir}/status` (back-compat with workflows commissioned by spacedock<=0.10.x that still ship a local script).
3. **Degraded fallback**: if neither is available, scan entity frontmatter directly to reconstruct the same three views (dispatchable, gate-blocked, in-progress with non-empty `worktree`). Note this in the rendered "What's Next" section so the captain knows the data was derived without the status helper.

Keep the section's downstream consumers (the "What's Next" section in the draft and final debrief at `skills/debrief/SKILL.md:206-207, 332-336`) unchanged — only the extraction mechanism shifts.

## Acceptance criteria

- AC1: Debrief Phase 2e runs to completion against a workflow that has no `{dir}/status` executable.
  - Verified by: behavioral fixture below (a workflow directory containing only README.md + entity files, no `status` file). Phase 2e produces a populated "What's Next" section.

- AC2: `skills/debrief/SKILL.md` Phase 2e prose names plugin-shipped status as the primary invocation and documents both the legacy `{dir}/status` fallback and the frontmatter-scan degraded fallback.
  - Verified by: grep guards on `skills/debrief/SKILL.md` — must contain both `{spacedock_plugin_dir}/skills/commission/bin/status` and a sentence licensing frontmatter-scan when no status helper is reachable; must NOT present `{dir}/status` as the unconditional primary.

- AC3: Frontmatter-scan degraded mode is surfaced to the captain in the rendered "What's Next" section when triggered (so a debrief produced without a status helper is self-describing).
  - Verified by: skill prose at the Phase 2e edit site instructs the agent to add a one-line "_(reconstructed from entity frontmatter — no status helper available)_" annotation in that section when the degraded path was taken.

- AC4: Adjacent #174 (debrief discovery ignoring `.claude/worktrees`) and #5a (commission README portability) remain independent — this task ships alone.
  - Verified by: implementation diff touches only `skills/debrief/SKILL.md` (Phase 2e block) and the regression fixture/test; no edits to `skills/commission/SKILL.md` or to debrief discovery (Phase 1) prose.

## Test plan

**Regression fixture** — add `tests/fixtures/debrief-no-local-status/` (new fixture directory):
- `README.md` with minimal Spacedock workflow frontmatter (`commissioned-by: spacedock@<current>`, one stage, slug id-style for simplicity).
- 2-3 entity `.md` files spanning at least one dispatchable status, one gated status, and one with a populated `worktree` field — enough to exercise all three "What's Next" buckets.
- **No `status` executable** — this is the crux of the fixture. Existing fixtures (e.g. `tests/fixtures/multi-stage-pipeline/status`) all ship a local `status` script; this one deliberately omits it.

**Test entrypoint** — no debrief test host exists today (`tests/` has 40+ files, none named `test_debrief_*`). Two options for the implementation stage to choose from:

- Add `tests/test_debrief_skill.py` as a new test host. Initial test asserts the prose-level guards from AC2/AC3 (grep `skills/debrief/SKILL.md` for the required tokens and the degraded-mode annotation prose). This is a static check — appropriate for a skill that is prose-executed by an agent rather than a binary we can subprocess.
- Optionally add a behavioral check that invokes the plugin-shipped status against the new fixture (`skills/commission/bin/status --workflow-dir tests/fixtures/debrief-no-local-status --next`) and asserts the dispatchable entity is reported. This proves the documented primary path actually works against a fixture matching the failure shape — but does not exercise the debrief skill itself (no agent in the loop). Keep this as a supplementary fixture sanity test, not the regression test for AC1.

The AC1 behavioral verification (debrief runs to completion against the no-local-status fixture) is best validated by a pilot manually invoking `spacedock:debrief` against the fixture during stage-report time and capturing the rendered output in the entity body. Document this expectation in the test plan; a fully automated agent-in-the-loop test is out of scope.

## Out of scope

- Coordinating the prose with #5a (`commission-readme-portable-status-path`). #5a removes status-usage prose from generated READMEs; this task fixes the debrief skill's status invocation. The two land independently and the directional alignment is intentional but not enforced.
- Fixing #174 (debrief discovery ignoring `.claude/worktrees`). Same skill, same reporter, but a different code path (Phase 1 Discovery vs Phase 2e Extraction).
- Removing the local-`{dir}/status` fallback entirely. Workflows commissioned by spacedock<=0.10.x in the wild still carry it; the legacy fallback is cheap to document and avoids a silent regression for those users.
- Adding a `spacedock` CLI wrapper on PATH (the systemic portability fix; cross-referenced from #5a, captain-future).

## Stage Report: ideation

PASS

- Picked Shape A (documented fallback) over Shape B (runtime detect-and-route). Justification: prose-driven skill, ages well alongside #5a, frontmatter-scan already within agent capabilities.
- Test plan names a regression fixture (`tests/fixtures/debrief-no-local-status/`, deliberately omits the local `status` executable) and a test entrypoint (`tests/test_debrief_skill.py` — does not exist today; would be a new test host). Distinguished prose-guard tests (automatable) from behavioral debrief-runs-to-completion verification (pilot-driven, not fully automatable).
- AC items are end-state properties with concrete `Verified by:` clauses: AC1 (fixture passes), AC2 (skill prose contains required tokens), AC3 (degraded-mode self-annotation), AC4 (diff scoped to this skill + fixture only).
- Adjacency to #5a noted in the entity body without coordination dependency.
- No worktree; edited the entity file directly.

## Stage Report: implementation

- DONE: Rewrite Phase 2e in `skills/debrief/SKILL.md` per the entity body's Approach
  Commit `4a0bf553`; primary = `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}` (with `--next`), legacy = `{dir}/status` gated on plugin-shipped unreachable AND local exists, degraded = frontmatter scan with self-annotation `_(reconstructed from entity frontmatter — no status helper available)_`.
- DONE: Add regression fixture `tests/fixtures/debrief-no-local-status/` + `tests/test_debrief_skill.py`
  Commit `d463331a`; fixture has README (commissioned-by spacedock@0.11.0) + 3 entities spanning backlog / review-gated / build-with-worktree, no local `status`. Test host has 12 prose-guard tests covering AC2/AC3 plus fixture sanity.
- DONE: Local verification — `make test-static` green; targeted `pytest tests/test_debrief_skill.py -v` green
  Targeted: `12 passed in 0.04s`. Static: `551 passed, 26 deselected, 15 subtests passed in 36.86s`.

### Summary

Phase 2e now documents a primary/legacy/degraded fallback chain instead of treating `{dir}/status` as mandatory; the legacy local script remains a back-compat path for spacedock<=0.10.x workflows, and a frontmatter scan is the self-describing degraded mode. Two commits on the worktree branch — one for the SKILL.md rewrite, one for the fixture + new test host. AC1's behavioral check (agent-in-the-loop debrief run) is intentionally pilot-driven per the test plan; AC2/AC3 are guarded statically; AC4 (scope) holds — diff touches only `skills/debrief/SKILL.md`, `tests/fixtures/debrief-no-local-status/`, and `tests/test_debrief_skill.py`.

## Stage Report: validation

PASSED

- DONE: AC1 reproduced — pilot-driven per test plan; the prose-guard substitute is sufficient. The fixture `tests/fixtures/debrief-no-local-status/` (README + 3 entities, no local `status`) matches the failure shape; the four `TestNoLocalStatusFixture` tests confirm the fixture remains failure-shaped (no `status` file present, README carries `commissioned-by: spacedock@`, entities span `backlog`/`review`/`build` with a populated `worktree`). Agent-in-the-loop debrief run remains a captain task per the entity body and is not automatable in CI.
- DONE: AC2 reproduced — `skills/debrief/SKILL.md:152-178` rewrites Phase 2e as a numbered fallback chain. Primary block names `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}` (with `--next` and bare); legacy block gates `{dir}/status` on plugin-shipped unreachable AND local existing, marked "back-compat path"; degraded block licenses frontmatter scan over `{dir}/*.md`. The 3 `TestPhase2ePrimaryInvocation` and 2 `TestPhase2eLegacyFallback` tests pass. `test_local_status_not_unconditional_primary` confirms `{dir}/status` does not appear before the plugin-shipped path in the section body.
- DONE: AC3 reproduced — degraded block ends with the literal annotation prose `_(reconstructed from entity frontmatter — no status helper available)_` (SKILL.md:172) and instructs the agent to prepend it to the rendered "What's Next" section. The 3 `TestPhase2eDegradedFallback` tests pass, including `test_degraded_mode_self_annotation_present` which asserts both "reconstructed from entity frontmatter" and "no status helper available" appear in the Phase 2e body.
- DONE: AC4 reproduced — `git diff $(git merge-base main HEAD)..HEAD --stat` shows changes confined to `skills/debrief/SKILL.md` (24 lines), `tests/fixtures/debrief-no-local-status/` (4 new files: README + 3 entities, 69 lines), `tests/test_debrief_skill.py` (119 lines), plus the entity file's own stage report. No edits to `skills/commission/SKILL.md` or to debrief Phase 1 prose. (The full `git diff main..HEAD --stat` also shows two adjacent entity files, but `git log HEAD --not main` confirms those are upstream changes on `main`, not contributions from this branch.)
- DONE: `make test-static` green — `551 passed, 26 deselected, 15 subtests passed in 29.78s`.
- DONE: Targeted `pytest tests/test_debrief_skill.py -v` green — `12 passed in 0.01s`. Per-test names matched: `TestPhase2ePrimaryInvocation` (3), `TestPhase2eLegacyFallback` (2), `TestPhase2eDegradedFallback` (3), `TestNoLocalStatusFixture` (4).
- DONE: Spot-checked Phase 2e prose at `skills/debrief/SKILL.md:150-178`. Three fallback paths are clearly documented and ordered (primary → legacy → degraded). The degraded-mode self-annotation is presented as a blockquote with explicit "prepend ... so the captain knows the data was reconstructed" instruction — instructive, not just a token. The diff preserves downstream consumers (the rendered "What's Next" section bullets at lines 174-177) unchanged; only the extraction mechanism shifted, as the entity body required.

Recommendation: PASSED.

## Stage Report: implementation (cycle 2 — drop fallbacks, drop static tests)

- DONE: Reframe 1 — drop primary/legacy/degraded fallback chain in Phase 2e
  Commit `21d070a9`. `skills/debrief/SKILL.md` Phase 2e is now a single primary path: invoke `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}` (with `--next` and bare); if unreachable, raise to the captain. No legacy `{dir}/status` fallback. No degraded frontmatter scan. No self-describing annotation prose. Rationale recorded in commit body: silent degradation hides real problems; FO already invoked `status --boot` so an unreachable helper is an environmental signal worth surfacing.
- DONE: Reframe 2 — delete static prose-guard test file + unreferenced fixture
  Commit `e7bce8ac`. Removed `tests/test_debrief_skill.py` (12 prose-guard tests added by cycle-1 commit `d463331a`) and `tests/fixtures/debrief-no-local-status/` (4 files: README + 3 entities). Same critique as 5a's static-tests rejection — prose-guards prove text not behavior; AC1 was always pilot-driven per the test plan.
- DONE: Local verification — `make test-static` green; targeted test count adjusted
  `make test-static`: `542 passed, 26 deselected, 15 subtests passed in 27.17s` (was 554; -12 corresponds exactly to the deleted `test_debrief_skill.py` suite). No regressions.

### Summary

Cycle-2 implementation collapses the cycle-1 three-tier fallback into a single primary path and drops the prose-guard test surface. Phase 2e now reads as a single helper invocation with an explicit "raise on unreachable" instruction. The static `test_debrief_skill.py` and its fixture are gone. Two commits land on the shared 8x+s6 branch; PR #177 picks them up. AC2/AC3 from the cycle-1 entity are now obsolete (no fallback chain, no degraded annotation); AC1's pilot-driven verification still applies but against the simpler shape.

## Stage Report: validation (cycle 2 — re-verifying drop-fallbacks + drop-static-tests)

PASSED

- DONE: Verified commit `21d070a9` exists with the claimed content. `git show --stat` confirms `skills/debrief/SKILL.md` -20/+7 (one file). Spot-read `skills/debrief/SKILL.md:149-163` — Phase 2e is now a single primary path: two `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}` invocations (with `--next` and bare), followed by the explicit "If the plugin-shipped status helper is unreachable, raise the error to the captain" instruction citing the FO's prior `status --boot`. No legacy `{dir}/status` fallback. No degraded frontmatter scan. No self-describing annotation prose.
- DONE: Verified commit `e7bce8ac` exists with the claimed content. `git show --stat` confirms 5 files deleted: `tests/test_debrief_skill.py` (-119) plus the four fixture files under `tests/fixtures/debrief-no-local-status/` (README + 3 entities, -69 total). Filesystem check confirms both paths are absent (`ls` returns "No such file or directory" for each).
- DONE: Phase 2e prose grep — `grep '{spacedock_plugin_dir}/skills/commission/bin/status' skills/debrief/SKILL.md` matches three times (lines 21, 154, 155 — line 21 is the `--discover` invocation in Phase 1, lines 154-155 are the Phase 2e primary path). `grep '{dir}/status' skills/debrief/SKILL.md` matches zero times — legacy invocation absent. `grep -in 'reconstructed from entity frontmatter\|no status helper available\|degraded' skills/debrief/SKILL.md` matches zero times — degraded-mode annotation prose absent.
- DONE: File-deletion check — `tests/test_debrief_skill.py` absent; `tests/fixtures/debrief-no-local-status/` absent. Both confirmed via `ls`.
- DONE: `make test-static` green — `542 passed, 26 deselected, 15 subtests passed in 26.97s`. Static count decreased exactly -12 from cycle-1's `554 passed` baseline, matching the 12 prose-guard tests removed by `e7bce8ac`. No regressions.
- DONE: Spot-checked Phase 2e prose at `skills/debrief/SKILL.md:149-163`. The simplified single-path semantics are clearly documented: code block with both `--next` and bare invocations, prose explanation of `{spacedock_plugin_dir}` resolution, explicit raise-on-unreachable instruction with rationale (FO's `status --boot` already proved reachability at session start). Downstream consumers (the bullet list at lines 161-163 mapping helper output to "What's Next" buckets) are preserved unchanged. Rationale in commit `21d070a9` body matches: silent degradation hides real problems; an unreachable helper post-boot is an environmental signal worth surfacing.

Recommendation: PASSED. Cycle-2 implementation faithfully executes the captain's reframe: drop the three-tier fallback chain and drop the 12 static prose-guard tests. The simpler single-primary-path shape is the merge candidate.
