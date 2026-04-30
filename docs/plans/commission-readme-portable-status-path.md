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

**AC-1 — Generated README contains no machine-specific path interpolations.**
Verified by: grep `{spacedock_plugin_dir}` and `.claude/plugins/cache` in a freshly commissioned `{dir}/README.md` returns zero matches.

**AC-2 — Generated README contains no status invocation prose.**
Verified by: grep `bin/status` in a freshly commissioned `{dir}/README.md` returns zero matches.

**AC-3 — Generated README's runtime-entrypoint section is the canonical FO-invocation prose.**
Verified by: `{dir}/README.md` contains a `## Workflow State` heading followed by prose mentioning `claude --agent spacedock:first-officer` and no other invocation examples in that section.

**AC-4 — Refit's existing Show-Diff surfaces the drift when run against an old README.**
Verified by: running refit's Phase 3b template-diff against a fixture README containing the old status-invocation snippets produces a diff whose deletions include `{spacedock_plugin_dir}` and `bin/status` lines, and whose additions include the canonical FO-invocation paragraph. No new refit code is added; this verifies the existing prose-and-diff pattern carries the constraint once the commission template is updated.

**AC-5 — Setup-time interpolations in `SKILL.md` (lines 503, 634, 662) remain unchanged.**
Verified by: diff of `skills/commission/SKILL.md` shows changes only inside the README heredoc bounds (roughly lines 279–455); `{spacedock_plugin_dir}` references at the three setup sites are preserved verbatim.

## Test plan

Static / parser-level (no live commission run needed):

1. **Grep guard on commission output.** Generate a README via the modified commission heredoc against a synthetic design-input fixture (mission text + entity + stages). Run `grep -E '\{spacedock_plugin_dir\}|\.claude/plugins/cache|bin/status' {generated_README}` — must return zero matches and exit 1. Covers AC-1, AC-2.

2. **Section-presence check on commission output.** Same generated README. Assert the `## Workflow State` section exists and contains the substring `claude --agent spacedock:first-officer`. Covers AC-3.

3. **Refit Show-Diff against a pre-existing old README.** Build a fixture: an `{dir}/README.md` containing the old status-invocation snippets (the three `{spacedock_plugin_dir}/skills/commission/bin/status ...` blocks plus the `grep -l "status:"` line). Run refit's Phase 3b against this fixture using the modified commission template as the diff target. Capture the resulting diff and assert: deletions contain at least one line with `{spacedock_plugin_dir}` and at least one line with `bin/status`; additions contain the canonical FO-invocation paragraph (`claude --agent spacedock:first-officer`). This exercises existing refit behavior against the new template — no new refit code involved. Covers AC-4.

4. **Setup-prose preservation diff.** After modifying `skills/commission/SKILL.md`, run `git diff skills/commission/SKILL.md` and assert all hunks fall within line range 279–455 (the README heredoc). Specifically verify the three `{spacedock_plugin_dir}` references at lines 503, 634, 662 are unchanged. Covers AC-5.

Live E2E (one smoke run, optional): commission a throwaway workflow into `/tmp/spacedock-portability-smoke/` with a minimal mission, then run the test-1 grep guard against the resulting README. This validates the full commission path end-to-end but is not required for AC verification — the static checks above cover the claim.

No live refit E2E is needed; test 3 exercises Phase 3b's diff-generation step against a fixture, which is sufficient to verify that the existing prose-and-diff pattern surfaces the drift once the template is updated.

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
