---
id: 5aqx95ck26bvj6dafmsa4rns
title: "Commissioned README should not reference machine-specific paths or status usage"
status: ideation
source: "GitHub issue #172 (filed by Jared Scott / gcko, 2026-04-30)"
started: 2026-04-30T19:47:24Z
completed:
verdict:
score: 0.7
worktree:
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

## Refit constraint check (mechanism + UX)

### Mechanism: substring grep guard on the commissioned README

Refit's Phase 3b ("README — Show Diff") gains a new pre-diff step that scans `{dir}/README.md` for two prohibited-content patterns:

1. **Machine-specific path leakage** — substring match for either:
   - literal unresolved placeholder `{spacedock_plugin_dir}`
   - absolute cache path fragment `.claude/plugins/cache` (matches both `/Users/...` and `~/...` variants)
2. **Status-usage prose leakage** — substring match for `bin/status`

Substring grep was chosen over markdown-section parsing because:
- both patterns are unique enough not to false-positive in normal mission/stage prose (no legitimate workflow README would contain `bin/status` or `.claude/plugins/cache` in its content)
- AST/section-presence checks add markdown parsing complexity for no marginal benefit
- a future captain-authored mention (e.g., a stage that genuinely involves running `bin/status`) would be unusual enough to warrant a one-line confirmation prompt rather than a silent allowlist

### Check signature (refit Phase 3b addition)

In `skills/refit/SKILL.md` Phase 3b, before the existing "generate template diff" step, insert a portability scan:

```
For each pattern in [`{spacedock_plugin_dir}`, `.claude/plugins/cache`, `bin/status`]:
  if pattern present in {dir}/README.md body text:
    record a drift finding with pattern + line numbers
```

If any drift findings recorded, surface them to the captain BEFORE the standard template diff:

> **Portability drift detected in `{dir}/README.md`:**
>
> Found {N} patterns that break multi-operator workflows:
> - `{pattern}` at line {N} — {one-line explanation: machine-specific path / status invocation in generated content}
>
> The current commission template no longer emits these patterns. The first officer skill encapsulates status access; captains run `claude --agent spacedock:first-officer` instead.
>
> Regenerate the affected README sections to the new shape? (y / n / show diff first)

On `y`, edit the README to replace the `## Workflow State` section with the canonical FO-invocation prose (same content as the new commission template emits) and remove any other lines containing the flagged patterns. On `n`, leave the README untouched and continue with the standard template diff. On `show diff first`, render the proposed edit as a diff and re-prompt.

### Cross-skill consistency

The canonical "## Workflow State" replacement prose lives in one place — `skills/commission/SKILL.md` heredoc — and refit reads it from there at runtime when offering regeneration. Refit does not duplicate the prose; it reads the current commission template and lifts the section. This keeps the two skills from drifting apart.

## Acceptance criteria

**AC-1 — Generated README contains no machine-specific path interpolations.**
Verified by: grep `{spacedock_plugin_dir}` and `.claude/plugins/cache` in a freshly commissioned `{dir}/README.md` returns zero matches.

**AC-2 — Generated README contains no status invocation prose.**
Verified by: grep `bin/status` in a freshly commissioned `{dir}/README.md` returns zero matches.

**AC-3 — Generated README's runtime-entrypoint section is the canonical FO-invocation prose.**
Verified by: `{dir}/README.md` contains a `## Workflow State` heading followed by prose mentioning `claude --agent spacedock:first-officer` and no other invocation examples in that section.

**AC-4 — Refit detects machine-specific paths in an existing commissioned README and surfaces the drift.**
Verified by: running refit against a fixture README containing `{spacedock_plugin_dir}/skills/commission/bin/status ...` produces a drift-detected prompt to the captain naming the offending pattern and line number, before the standard template diff.

**AC-5 — Refit detects status-usage prose in an existing commissioned README and surfaces the drift.**
Verified by: running refit against a fixture README containing `bin/status` produces a drift-detected prompt naming the offending pattern.

**AC-6 — Refit's offer-to-regenerate replaces the `## Workflow State` section with the canonical FO-invocation prose when accepted.**
Verified by: applying refit's regeneration to a drift fixture produces a README whose `## Workflow State` section matches the canonical prose emitted by the current commission template (same source of truth, lifted at runtime).

**AC-7 — Setup-time interpolations in `SKILL.md` (lines 503, 634, 662) remain unchanged.**
Verified by: diff of `skills/commission/SKILL.md` shows changes only inside the README heredoc bounds (roughly lines 279–455); `{spacedock_plugin_dir}` references at the three setup sites are preserved verbatim.

## Test plan

Static / parser-level (no live commission run needed):

1. **Unit-equivalent grep guard on commission output.** Generate a README via the modified commission heredoc against a synthetic design-input fixture (mission text + entity + stages). Run `grep -E '\{spacedock_plugin_dir\}|\.claude/plugins/cache|bin/status' {generated_README}` — must return zero matches and exit 1. Covers AC-1, AC-2.

2. **Section-presence check on commission output.** Same generated README. Assert the `## Workflow State` section exists and contains the substring `claude --agent spacedock:first-officer`. Covers AC-3.

3. **Refit drift-detection check against drift fixtures.** Two fixtures: (a) a README with `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir docs/foo` baked in, (b) a README with the canonical FO-invocation already present. Run refit's Phase 3b scan against each. Fixture (a) must produce drift findings naming both `{spacedock_plugin_dir}` and `bin/status` patterns; fixture (b) must produce zero drift findings. Covers AC-4, AC-5.

4. **Refit regeneration round-trip.** Apply refit's offer-to-regenerate to fixture (a). Re-run the grep guard from test 1 against the regenerated file — must return zero matches. Then assert the `## Workflow State` section content matches what the current commission template emits (string-equal after stripping leading/trailing whitespace). Covers AC-6.

5. **Setup-prose preservation diff.** After modifying `skills/commission/SKILL.md`, run `git diff skills/commission/SKILL.md` and assert all hunks fall within line range 279–455 (the README heredoc). Specifically verify the three `{spacedock_plugin_dir}` references at lines 503, 634, 662 are unchanged. Covers AC-7.

Live E2E (one smoke run, optional): commission a throwaway workflow into `/tmp/spacedock-portability-smoke/` with a minimal mission, then run the test-1 grep guard against the resulting README. This validates the full commission path end-to-end but is not required for AC verification — the static checks above cover the claim.

No live refit E2E is needed; the drift-detection check is pure prose scanning and the regeneration step is a deterministic Edit. Fixture-based testing is sufficient.

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
