---
id: 5aqx95ck26bvj6dafmsa4rns
title: "Commissioned README should not reference machine-specific paths or status usage"
status: implementation
source: "GitHub issue #172 (filed by Jared Scott / gcko, 2026-04-30)"
started: 2026-04-30T19:47:24Z
completed:
verdict:
score: 0.7
worktree: .worktrees/spacedock-ensign-commission-readme-portable-status-path
issue: "#172"
pr:
mod-block:
---

## Problem (as filed by reporter)

`commission` generates READMEs containing absolute per-machine status invocations:

```
/Users/<user>/.claude/plugins/cache/spacedock/spacedock/0.11.0/skills/commission/bin/status --workflow-dir <dir>
```

Three sources of per-machine drift: username, plugin version directory, cache prefix. Single-operator workflows are unaffected. Team-shared workflows break silently for every operator other than the original commissioner — `command not found` with no in-README hint.

## Captain-directed scope (2026-04-30)

A standalone `spacedock` CLI on PATH is the systemic fix (gives plugin CLIs the portability property agents already have via `{plugin}:{agent}` identifiers), but it's a bigger change. For now, the scope is to fix the commissioned README directly along three constraints:

1. **No machine-specific paths.** The commissioned README must not embed `~/.claude/plugins/cache/...` (or any other per-machine absolute path). The `{spacedock_plugin_dir}` placeholder must not be resolved into the generated README.

2. **No status-usage prose.** The commissioned README must not document `status` invocation at all. Status usage is encapsulated in the first-officer skill — that's where the runtime knows how to find and use it. Captains who want to inspect workflow state run the FO; the FO knows how. Captains who want raw `status` access read the FO skill prose; the README doesn't need to teach them.

3. **Refer to the first-officer skill.** The commissioned README's runtime-entrypoint section becomes: "to operate this workflow, run `claude --agent spacedock:first-officer`." That's it. The first-officer agent identifier is portable because the plugin loader resolves it the same way on every machine.

4. **Refit checks the constraints.** When `spacedock:refit` runs against an existing workflow, it verifies the README does not contain machine-specific paths and does not document status usage. If it finds either, it flags the drift to the captain and offers to regenerate the relevant README sections to the new shape.

## Concrete edits in `skills/commission/SKILL.md`

Re-read of the six interpolation sites against current `SKILL.md` (line numbers verified, no drift):

| Line | Context | Bucket | Action |
|------|---------|--------|--------|
| 401 | Inside generated README heredoc (`## Workflow State` section, basic `status` example) | GENERATED | **Remove** — replace section per below |
| 409 | Inside generated README heredoc (`status --archived` example) | GENERATED | **Remove** — replace section per below |
| 415 | Inside generated README heredoc (`status --next` example) | GENERATED | **Remove** — replace section per below |
| 503 | Phase 2c setup-time `cp` to install pr-merge mod | SETUP | **Keep** — runs on captain's machine at commission time |
| 634 | Phase 3 Step 2 instruction to read first-officer agent file at pilot run | SETUP | **Keep** — runs on captain's machine during pilot run |
| 662 | Phase 3 Step 5 failure-handling instruction to commission skill itself | SETUP | **Keep** — runs on captain's machine during pilot run; not in generated README heredoc (heredoc ends at line 455) |

Reclassification note: the entity-body intake initially grouped 662 with the "remove" list. Verified by reading the file — line 662 sits in Phase 3 Step 5 of `SKILL.md`, well outside the README heredoc that spans roughly lines 279–455. It is setup-time prose addressed to the commission skill while the skill is acting as first officer for the pilot run. Per the captain's third constraint ("setup is captain-machine-local by definition"), it stays.

### Replacement for the generated `## Workflow State` section

Lines 396–422 of the heredoc currently contain three example status invocations plus a `grep -l` snippet. Replace the entire section with:

```markdown
## Workflow State

Workflow state is read by the first officer at boot. To view current state, dispatch the first officer or run it directly:

\`\`\`
claude --agent spacedock:first-officer
\`\`\`
```

The `grep -l "status: {stage_name}" {dir}/*.md` snippet at line 421 is also dropped — it teaches a status-discovery technique that belongs in the FO skill, not the generated README. Captains who want raw filesystem access already know how to grep.

## Refit constraint check

**Refit needs no new mechanism.** Phase 3b already implements the right pattern: generate what the current commission template would produce for this workflow, diff it against the user's existing README, present the diff, ask the captain to apply changes manually or call out specific edits.

Once the commission template no longer emits `{spacedock_plugin_dir}/.../bin/status` examples and instead emits the canonical FO-invocation prose, refit's existing Show-Diff naturally surfaces the drift: an old README that still has the three status-invocation snippets will diff against a new template that has the FO-invocation paragraph instead. The captain sees the drift in the diff output and decides whether to adopt the new shape — same UX they already use for any other template change.

The constraint compliance is therefore a property of the commission template, not a new check in refit. Update the template, and refit's existing prose-and-diff pattern carries the rest.

### Residual-case note

The pattern that won't surface cleanly via plain template-diff is one where the README has heavy captain customizations interleaved with the old status-invocation snippets, making the diff noisy enough that the captain might miss the deletion of the offending lines. This is plausible but not load-bearing: refit already documents that "differences may be template improvements or your intentional customizations" and asks the captain to review. The defense for the noisy-diff case is the existing review prose, not new machinery. If this case proves common in practice, a future minimal addendum to refit Phase 3b could call out machine-specific-path lines explicitly in the diff summary — but that's a follow-on, not part of this task's scope.

## Acceptance criteria

**AC-1 — Live commission produces a portable README.**
Verified by: extending `tests/test_commission.py::test_commission` (the `@pytest.mark.live_claude` end-to-end runner) with post-commission assertions on the generated `{workflow_dir}/README.md`:

- `{spacedock_plugin_dir}` substring count is zero
- `.claude/plugins/cache` substring count is zero
- `bin/status` substring count is zero
- `## Workflow State` section contains `claude --agent spacedock:first-officer`
- `## Workflow State` section contains no `bin/status` invocation example

The test is `xfail(strict=False)` for unrelated `#197` regressions; the new portability checks PASS individually whether the test xfails or xpasses. Local live run (`unset CLAUDECODE && uv run pytest tests/test_commission.py -v`) is the verification artifact.

**AC-2 — First officer documents the captain-facing state-display pattern.**
Verified by: `skills/first-officer/references/first-officer-shared-core.md` contains a Captain-Facing State Display subsection under `## Status Viewer` that names (a) the trigger rule (which captain questions invoke `status` for state display), (b) the canonical invocations (overview, dispatchables, archive view, single-entity lookup), and (c) output rendering guidance (forward stdout verbatim in a fence). The new commissioned README delegates state inspection to the FO; this AC ensures the FO knows what to invoke.

**AC-3 — Test fixture READMEs comply with the same constraints.**
Verified by: `grep -lE '\{spacedock_plugin_dir\}|\.claude/plugins/cache|bin/status' tests/fixtures/*/README.md` returns zero matches across the 16 fixture READMEs. Audit-only — no fixture edits required at this time.

**AC-5 — Setup-time interpolations in `SKILL.md` (lines 503, 634, 662 pre-edit; 483, 614, 642 post-edit) remain unchanged.**
Verified by: diff of `skills/commission/SKILL.md` shows changes only inside the README heredoc bounds (roughly lines 279–455); `{spacedock_plugin_dir}` references at the three setup sites are preserved verbatim. (AC numbering preserves AC-5 from cycle-1 for traceability; AC-4 from cycle-1 — refit Show-Diff against an old README — is dropped because the cycle-2 reframe makes the live commission output the single source of truth, not a static diff against a synthetic old README.)

## Test plan

1. **Live commission portability assertions.** Extend `tests/test_commission.py::test_commission` (live_claude) with the five portability checks listed in AC-1 against the generated `{workflow_dir}/README.md`. Run locally with `unset CLAUDECODE && uv run pytest tests/test_commission.py -v` and confirm each new check shows `PASS:` in stdout. Covers AC-1.

2. **FO reference inspection.** Read the modified `skills/first-officer/references/first-officer-shared-core.md` and confirm the new Captain-Facing State Display subsection contains the trigger rule, the four canonical invocations, and the rendering guidance. Run `make test-static` to confirm no static-test regression (the shared-core file is plain prose; the regression risk is purely in any test that greps it for specific substrings). Covers AC-2.

3. **Fixture README audit.** Run `grep -lE '\{spacedock_plugin_dir\}|\.claude/plugins/cache|bin/status' tests/fixtures/*/README.md` and confirm zero matches across all 16 fixture READMEs. Covers AC-3.

4. **Setup-prose preservation diff.** After modifying `skills/commission/SKILL.md`, run `git diff main..HEAD -- skills/commission/SKILL.md` and assert all hunks fall within line range 279–455 (the README heredoc); the three `{spacedock_plugin_dir}` references in setup-time prose are unchanged. Covers AC-5.

The live test in step 1 is the canonical end-to-end verification — it exercises the actual commission path and produces a real README to check, replacing the cycle-1 static heredoc grep that proved only the template literal.

## Out of scope

- Standalone `spacedock` CLI wrapper on PATH (the systemic fix; captain has it in mind for a separate larger task — flagged here for cross-reference).
- Migration helper for existing already-commissioned READMEs (refit's check + offer-to-regenerate covers the upgrade path).
- Other absolute paths in commissioned files (mod source paths in setup prose, etc.) unless they fall into the same anti-pattern at commission time.

## Cross-references

- **#221** (commission templates + Trait Detection, just shipped) — the proximate cause: trait detection now confidently produces multi-operator workflows for team-flavored missions, exposing the per-machine path as a plurality rather than an edge case.
- **GH #172** original framing offered three fix shapes (CLI wrapper, portable resolution snippet, doc note). The captain chose a fourth: encapsulate status in the FO and remove status from the README.
- Deferred follow-up: standalone `spacedock` CLI (captain-future).

## Stage Report: ideation

- DONE: Concrete edits pinned: re-read `skills/commission/SKILL.md` lines 401, 409, 415, 503, 634, 662; confirm GENERATED vs SETUP-TIME PROSE
  Verified all six lines via `grep -n 'spacedock_plugin_dir'` — no drift. Classification: 401/409/415 are inside the README heredoc (lines ~279–455) and must be removed; 503/634/662 are setup-time prose (Phase 2c install / Phase 3 Step 2 read agent file / Phase 3 Step 5 failure handling) and stay. Reclassified 662 from the entity-body's "remove" bucket to "keep" — it sits in Phase 3 Step 5, outside the heredoc.
- DONE: Refit constraint check shape pinned: name the actual mechanism + check signature + captain-facing surface
  Mechanism: substring grep guard on three patterns (`{spacedock_plugin_dir}`, `.claude/plugins/cache`, `bin/status`) inserted into refit Phase 3b before the standard template diff. Captain-facing surface: a drift-detected prompt naming pattern + line, with y/n/show-diff-first options. Regeneration lifts the canonical `## Workflow State` prose from the current commission template at runtime to avoid skill-to-skill drift.
- DONE: AC list with concrete `Verified by:` clauses (commissioned README grep guards + refit detection check + regression test)
  Seven ACs added covering: generated README grep guards (AC-1, AC-2), section-presence (AC-3), refit drift detection for both pattern families (AC-4, AC-5), refit regeneration round-trip (AC-6), and setup-prose preservation (AC-7). Five-step test plan added — four static/parser-level tests cover all ACs; one optional live E2E smoke run.

### Summary

Pinned the implementation path to two surgical edits: replace the `## Workflow State` heredoc section in `skills/commission/SKILL.md` with a single FO-invocation paragraph (drops the three `{spacedock_plugin_dir}/.../bin/status` examples plus the `grep -l "status:"` snippet), and add a substring grep guard to `skills/refit/SKILL.md` Phase 3b that surfaces drift before the standard template diff. Key clarification: the entity-body intake misclassified line 662 — it's setup-time prose in Phase 3 Step 5, not generated content, and stays. Acceptance is testable entirely via static grep + fixture-based refit checks; no live E2E is required.

### Feedback Cycles

**Cycle 1 (rejected by captain).** Captain rejected the gate on two grounds:

1. **Refit-mechanism reframe.** Cycle 1 proposed a new substring grep guard + custom drift-detection prompt in refit Phase 3b. Captain pushed back: refit already has a Show-Diff strategy for README.md, and once the commission template is updated, the diff naturally surfaces the drift. No new refit machinery needed. Constraint compliance is a property of the commission template, not new code in refit.
2. **Test plan reshape.** Tests 3 and 4 (drift-detection-fixture check and regeneration round-trip) were testing the new mechanism that no longer exists. They needed to be dropped and replaced with a single test that exercises refit's existing Show-Diff against a fixture old README, asserting the diff surfaces the right deletions and additions.

Cycle 2 reworked: dropped the new-mechanism prose from `## Refit constraint check`, replaced with "no new mechanism needed" prose plus a residual-case note. Dropped AC-4/5/6 (mechanism-specific), added a new AC-4 (existing diff surfaces drift), renumbered the setup-preservation AC to AC-5. Reshaped the test plan from five tests to four — dropped tests 3 and 4, added a new test 3 that exercises Phase 3b's existing diff against an old-README fixture against the new commission template.

## Stage Report: ideation (cycle 2)

- DONE: Replace `## Refit constraint check (mechanism + UX)` section with shorter "uses existing Show-Diff" prose
  Reframed section now states refit needs no new mechanism — constraint compliance is a property of the commission template, and refit's existing Phase 3b Show-Diff carries the rest. Residual-case note flags the noisy-customized-diff scenario as a possible future minimal addendum but explicitly out of scope here.
- DONE: Drop AC-4, AC-5, AC-6 (mechanism-specific); add AC about existing diff surfacing the new template against an old README
  AC-4 (drift detection prompt), AC-5 (status-prose detection prompt), AC-6 (regeneration round-trip) removed. New AC-4 verifies refit's existing Show-Diff against a fixture old README produces a diff with the expected deletions (lines containing `{spacedock_plugin_dir}` and `bin/status`) and additions (the canonical FO-invocation paragraph). Setup-preservation AC renumbered from AC-7 to AC-5.
- DONE: Reshape Test plan section per the reframe
  Dropped test 3 (drift-detection fixture scan) and test 4 (regeneration round-trip). New test 3 builds an old-README fixture, runs refit Phase 3b's existing diff against the modified commission template, and asserts the diff content. Test count went from 5 to 4. Optional live E2E and rationale for skipping live refit E2E preserved.
- DONE: Add `### Feedback Cycles` section after the prior Stage Report
  Cycle 1 entry added documenting both reframe grounds and the cycle-2 response shape.

### Summary

Reworked per captain's two reframes. Refit gains no new code — the commission template update alone, combined with refit's existing prose-and-diff Show-Diff pattern, satisfies the constraint. AC count went from 7 to 5, test plan from 5 tests to 4. The implementation path is now even more surgical: one heredoc edit in `skills/commission/SKILL.md`, zero edits in `skills/refit/SKILL.md`. Test 3 exercises existing refit behavior against a fixture, not new behavior.

## Stage Report: implementation

- DONE: Edit the README heredoc in `skills/commission/SKILL.md`: drop the three `{spacedock_plugin_dir}/.../bin/status` example blocks and the `grep -l "status:"` snippet; replace the `## Workflow State` section with the canonical FO-invocation paragraph; setup-time prose preserved verbatim
  Commit `2906988e`. Heredoc shrunk by 20 lines; remaining `{spacedock_plugin_dir}` references at lines 483 / 614 / 642 (was 503 / 634 / 662 pre-edit) all sit outside the heredoc which now closes at line 435.
- DONE: Add the regression tests per the entity body's Test plan (grep guard, section-presence, refit Show-Diff against old-README fixture, setup-prose preservation diff)
  Commit `d6b0387d`. Five tests in `tests/test_commission_readme_portability.py` covering AC-1 through AC-5. All pass locally.
- DONE: Local verification: targeted `pytest -k commission_readme_portability` green + full `make test-static` green
  Targeted: `5 passed in 0.02s`. Full static: `544 passed, 26 deselected, 15 subtests passed in 28.02s`.

### Summary

Two surgical commits as instructed: (a) heredoc edit in `skills/commission/SKILL.md` removing three status-invocation examples and the grep-l snippet, replacing the `## Workflow State` section with a single FO-invocation paragraph; (b) five static regression tests in a new `tests/test_commission_readme_portability.py` covering all five ACs. Zero changes to `skills/refit/SKILL.md` per the entity's "refit needs no new code" design — AC-4 is satisfied by the property that an old-README fixture diffed against the new heredoc surfaces `{spacedock_plugin_dir}` and `bin/status` deletions plus the FO-invocation addition. Full static suite (544 tests) green.

## Stage Report: validation

- DONE: AC-1 — Generated README contains no machine-specific path interpolations
  `test_heredoc_has_no_machine_specific_paths` PASSED; heredoc body (SKILL.md lines 279–435) contains zero `{spacedock_plugin_dir}` and zero `.claude/plugins/cache` substrings.
- DONE: AC-2 — Generated README contains no status invocation prose
  `test_heredoc_has_no_status_invocation_prose` PASSED; heredoc body contains zero `bin/status` substrings.
- DONE: AC-3 — Generated README's runtime-entrypoint section is the canonical FO-invocation prose
  `test_heredoc_workflow_state_section_points_to_first_officer` PASSED; spot-read of `skills/commission/SKILL.md` lines 396–402 confirms `## Workflow State` followed by FO-invocation paragraph and a single fenced `claude --agent spacedock:first-officer` example. No `bin/status` or `{spacedock_plugin_dir}` in that section.
- DONE: AC-4 — Refit's existing Show-Diff surfaces drift against an old README
  `test_refit_show_diff_against_old_readme_surfaces_drift` PASSED; fixture old-README diffed against current heredoc produces deletions including `{spacedock_plugin_dir}` and `bin/status` lines and additions including `claude --agent spacedock:first-officer`. No new refit code introduced (`git diff main..HEAD --stat` shows zero `skills/refit/` changes).
- DONE: AC-5 — Setup-time interpolations in `SKILL.md` (lines 503/634/662 pre-edit → 483/614/642 post-edit) remain unchanged
  `test_setup_prose_interpolations_remain_outside_heredoc` PASSED; `grep -n 'spacedock_plugin_dir' skills/commission/SKILL.md` returns 483 (`cp ... mods/pr-merge.md`), 614 (`Read the first-officer agent file ...`), 642 (`Show the current state of the workflow with ... bin/status ...`). All three sit outside the heredoc which closes at line 435. `git diff` confirms all hunks are within original lines 395–422 (inside the heredoc).
- DONE: `make test-static` regression run
  `544 passed, 26 deselected, 15 subtests passed in 39.58s`.
- DONE: Targeted `pytest tests/test_commission_readme_portability.py -v`
  `5 passed in 0.01s`.

### Verdict

PASSED. All five ACs reproduce against the dispatched evidence. Implementation is exactly two commits (`2906988e` heredoc edit + `d6b0387d` tests) plus the implementation stage report (`704a7099`). Diff scope confirmed: 1 file changed in `skills/commission/SKILL.md` (heredoc only, lines 395–422), 1 new file `tests/test_commission_readme_portability.py` (169 lines), zero changes in `skills/refit/`. The reframed "no new refit code" design from ideation cycle 2 holds — AC-4 is satisfied by the template-property carry, exercised by test 3.

## Stage Report: implementation (cycle 2 — captain-rejected validation gate fix-up)

Captain rejected the cycle-1 validation gate post-PR for three reframes documented in the entity body: (1) the synthetic static test file proved the template literal but not real commission output; replace with live-test extension. (2) the new commissioned README delegates state inspection to the FO, but the FO had no consolidated reference for the captain-state-display pattern; add one. (3) the same constraints must apply to test fixture READMEs.

- DONE: Reframe 1 — Replace synthetic static tests with extension of existing live commission test
  Commit `12f1fa87`. Deleted `tests/test_commission_readme_portability.py` (169 lines of heredoc-grep tests that proved only the template literal). Added a new `[README Portability]` section to `tests/test_commission.py::test_commission` (live_claude E2E) with five `t.check` calls against the actual generated `{workflow_dir}/README.md`: zero `{spacedock_plugin_dir}` substrings, zero `.claude/plugins/cache` substrings, zero `bin/status` substrings, `## Workflow State` section invokes `claude --agent spacedock:first-officer`, and that section contains no `bin/status` example. Local live verification: `unset CLAUDECODE && uv run pytest tests/test_commission.py -v -s` produced `66 passed, 2 failed (out of 68 checks)` with all 5 new portability checks shown as `PASS:` in stdout — the 2 failures are the pre-existing `#197` regressions (`workflow-local pr-merge mod is not generated` + `no leaked template variables`) that own the xfail. `xfail(strict=False)` tolerates either xfail or xpass; the test xfailed as expected without disturbing the new assertions. The xfail decorator stays — `#197` owns it.

- DONE: Reframe 2 — Add status-usage reference to FO runtime
  Commit `d24cea6c`. Added a `### Captain-Facing State Display` subsection under `## Status Viewer` in `skills/first-officer/references/first-officer-shared-core.md`. Chose shared-core over the Claude runtime adapter because the captain-state-display behavior is core FO semantics, not Claude-runtime-specific (Codex captains hit the same pattern). The subsection documents (a) the trigger rule listing the captain questions that invoke this pattern and explicitly distinguishes it from event-loop scheduling reads, (b) the four canonical invocations (overview / dispatchables / archive view / single-entity lookup), and (c) the output rendering guidance (forward stdout verbatim in a fence, add a one-line preface, do not paraphrase). Existing scattered status mentions (boot at shared-core L14-22, event loop at L240-243, mod-block enforcement at L257-264) stay in place — they cover specific FO-internal operations; the new subsection centralizes the captain-facing pattern.

- DONE: Reframe 3 — Apply constraints to test fixture READMEs
  Audit-only — no edits required. `grep -lE '\{spacedock_plugin_dir\}|\.claude/plugins/cache|bin/status' tests/fixtures/*/README.md` returns zero matches across all 16 fixture READMEs (`output-format-default`, `completion-signal-pipeline`, `checklist-pipeline`, `merge-hook-pipeline`, `rejection-flow`, `spike-no-gate`, `reuse-pipeline`, `rejection-flow-packaged-agent`, `standing-teammate`, `spike-gated`, `keepalive-pipeline`, `per-stage-model`, `multi-stage-pipeline`, `push-main-pipeline`, `gated-pipeline`, `output-format-custom`). The live test from reframe 1 produces a fresh README into a tmp_path and verifies it via the new assertions, so any future regression in commission output is caught at the live-test boundary even without a separate fixture.

- DONE: Local verification (live test + static suite)
  Live: `unset CLAUDECODE && uv run pytest tests/test_commission.py -v` → `1 xfailed in 104.18s`; the inner check counter shows `66 passed, 2 failed (out of 68 checks)` with all 5 new portability checks PASSING. Static: `make test-static` → `539 passed, 26 deselected, 15 subtests passed in 26.55s` (count dropped from 544 → 539 because the 5 deleted static heredoc tests are now folded into the live test).

### Summary

Three commits, one concern each: (a) `12f1fa87` swaps the static heredoc test for live-commission-output assertions; (b) `d24cea6c` adds the Captain-Facing State Display subsection to the FO shared-core; (c) this entity-body update. The cycle-1 validation gate verified the heredoc template literal; the cycle-2 verification chain now grounds the same five portability properties in the actual commission output and ensures the FO knows the pattern that the new README delegates to it. Fixture audit is clean — no fixture edits needed; the live-test path covers future regressions implicitly.
