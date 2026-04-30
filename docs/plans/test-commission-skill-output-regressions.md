---
id: 197
title: "test_commission: commission skill produces leaked templates, absolute paths, unwanted _mods/pr-merge.md"
status: validation
source: "PR #131 CI (#154 cycle-1 pre-merge) — after #154 lifted the content-drift xfail and swapped test_commission's static content reads to `assembled_agent_content`, 60/63 inner checks pass; 3/63 remain FAIL on all three claude variants (claude-live, claude-live-bare, claude-live-opus)"
started: 2026-04-30T22:10:38Z
completed:
verdict:
score: 0.7
worktree: .worktrees/spacedock-ensign-commission-readme-portable-status-path
issue:
pr: #176
mod-block: merge:pr-merge
---

## Problem

`test_commission` is decorated `@pytest.mark.xfail(strict=False, reason="pending #197 …")`. A live re-run against the current 5a worktree (`spacedock-ensign-commission-readme-portable-status-path` HEAD `8a19c7e6`) shows **2 inner failures out of 68 checks** — a different shape than the PR #131-era body (3/63). The two current failures are:

### Failure 1: `[File Existence] workflow-local pr-merge mod is not generated`
Test code: `t.check("workflow-local pr-merge mod is not generated", not (workflow_dir / "_mods" / "pr-merge.md").exists())`
The test asserts the file MUST NOT exist. Commission produces it: the generated `_mods/pr-merge.md` is byte-identical to the plugin-shipped `mods/pr-merge.md`. Source: `skills/commission/SKILL.md:482-483` instructs the LLM to `cp "{spacedock_plugin_dir}/mods/pr-merge.md" {dir}/_mods/pr-merge.md`.

### Failure 2: `[No Leaked Template Variables] no leaked template variables`
Test code (lines 315-330): scans every `*.md` under `workflow_dir` for `\{[a-z_]+\}` excluding lines containing `${` or `slug`. Five leaks reported; **four of the five live inside `_mods/pr-merge.md`** (a verbatim copy of the plugin file, which legitimately contains doc/template variables `{number}`, `{branch}`, `{constructed body}`):

- `_mods/pr-merge.md`: `gh pr view {number} --json state --jq '.state'`
- `_mods/pr-merge.md`: `**Branch:** {branch} -> main`
- `_mods/pr-merge.md`: `git push origin {branch}`
- `_mods/pr-merge.md`: `gh pr create --base main --head {branch} --title "{entity title}" --body "{constructed body}"`

The fifth lives in `refit-command.md` (a generated entity body, not a template):
- `refit-command.md`: `Verified by: the README frontmatter \`commissioned-by\` field reads \`spacedock@{current_version}\` after refit completes.` — the LLM wrote `{current_version}` as English prose describing a refit AC. It is not a leaked template slot; it is prose that *mentions* a template variable.

### Failure 3 (historical, currently passing): `no absolute paths in generated files`
The `[No Absolute Paths]` section now passes (5a cycle 2 added the `[README Portability]` section and 5a's portability fix landed in commit `2906988e commission: replace per-machine status snippets with FO-invocation prose`). This failure class is already resolved upstream and out of scope for #197.

## Captain decision (cycle 2): Option B-ii + A2

Captain reviewed cycle-1 ideation and picked **Option B-ii + A2** (not Option A). Reasoning:

The 4 leaks inside `_mods/pr-merge.md` (`{number}`, `{branch}` x2, `{entity title}`, `{constructed body}`) are not commission-time slots that should be resolved at generation. They are documentation prose telling the FO what to substitute at merge-hook execution time. The mod is correctly written. The test's leak-scan was overly broad for `_mods/*.md`.

Option A's architectural shift (plugin-mods auto-discovery + opt-out mechanism) is unwarranted for this task. Keep the install model — commission continues to `cp` plugin pr-merge.md into the workflow's `_mods/`. Refine the test instead so it stops flagging legitimate runtime placeholders inside mod files, and add a freshness signal that catches accidental edits or stale-after-plugin-upgrade drift. Fix the 5th leak (refit-command.md `{current_version}`) at content source.

## Proposed approach (Option B-ii + A2)

1. **`tests/test_commission.py` — exclude `_mods/*.md` from the leak scan** (around lines 315-330). The `rglob("*.md")` walk should skip any file whose path contains `_mods/`. These files legitimately contain runtime placeholders documented for FO substitution.
2. **`tests/test_commission.py` — add a `[Mod Install Freshness]` check** (or fold into `[File Existence]` / `[PR Merge Mod]`). For each `{workflow_dir}/_mods/{name}.md`, byte-compare against `{spacedock_plugin_dir}/mods/{name}.md`. Pass when bytes match; fail and report a one-line diff summary on drift. This catches accidental edits to the local copy and stale copies after plugin upgrades.
3. **`tests/test_commission.py` — delete or replace the `workflow-local pr-merge mod is not generated` check** at line 79. Under B-ii the file IS expected to exist (it's the install). Replace with the byte-compare freshness check from step 2 if not folded in there. The `[File Existence]` section keeps its other checks unchanged.
4. **`skills/commission/SKILL.md` — adjust the refit-command seed-entity scaffolding prose (A2)** so the LLM doesn't model `{current_version}`-style brace-syntax in entity body text. Replace bracey example phrasing with backticks-around-words or natural-language descriptions (e.g., "reads the current spacedock version" instead of `spacedock@{current_version}`). Locate the seed-entity block in SKILL.md that demonstrates the refit-command AC and rewrite the example AC line so it parses cleanly under the leak-scan regex.

## Acceptance criteria

End-state properties; each AC has a `Verified by:` clause naming the existing extended `test_commission`.

- **AC1**: Workflow-local mods match plugin source byte-for-byte. For each `{workflow_dir}/_mods/{name}.md`, bytes equal `{spacedock_plugin_dir}/mods/{name}.md`.
  Verified by: new `[Mod Install Freshness]` (or equivalent) check in `tests/test_commission.py` reads both files and asserts byte-equality. Drift is a real bug — accidental edit to the local copy or stale-after-plugin-upgrade.
- **AC2**: No `{lower_word}` patterns in entity-body `*.md` files (excluding `_mods/*.md` which legitimately use runtime placeholders).
  Verified by: `tests/test_commission.py:315-330` leak scan, modified to skip files whose path contains `_mods/`. Returns an empty `leaked` list against fresh commission output.
- **AC3**: The `refit-command.md` seed-entity prose generated by commission contains no `\{[a-z_]+\}` patterns in entity-body text.
  Verified by: AC2's scan covers it — `refit-command.md` is in `workflow_dir`, NOT in `_mods/`, so it's still scanned.
- **AC4**: The `xfail(strict=False, reason="pending #197 …")` decorator on `test_commission` is left in place by this task. Removal is a follow-up after CI confirms xpass on a few cycles; `strict=False` tolerates xpass without breaking CI.
  Verified by: `tests/test_commission.py` diff shows the decorator unchanged after this task's PR merges; a follow-up note (or separate issue) tracks the eventual removal.

## Test plan

The existing `test_commission` (extended to 68 checks via 5a cycle 2 commit `12f1fa87`) is the authoritative proof. After this task:

1. **Pre-fix baseline (already captured)**: `66 passed, 2 failed (out of 68 checks)` against 5a worktree HEAD `8a19c7e6`.
2. **Post-fix expected**: leak scan now excludes `_mods/`, byte-compare freshness check added, refit-command seed prose tweaked. Total inner checks go from 68 to 68 (delete one, add one) or 69 (keep+add). Either way, expected outcome is `N passed, 0 failed` → pytest reports `XPASS` (tolerated by `strict=False`).
3. **Run command**: `unset CLAUDECODE && uv run pytest tests/test_commission.py -v -s` from the worktree root. Wallclock ~100s for the live commission phase.
4. **No regression coverage needed** beyond `test_commission`. Mechanism is unchanged — `scan_mods`, mod-block enforcement, FO runtime, and existing fixtures (`merge-hook-pipeline`, `push-main-pipeline`) are not touched.
5. **No new test files needed** — all changes land inside the existing `tests/test_commission.py`.

## Out of scope for this task

- **Decorator removal**. The xfail decorator stays. Removal happens after CI confirms a few clean xpass cycles.
- **Plugin-mods auto-discovery**. Option A from cycle 1 is shelved — install model stays as-is.
- **Mechanism changes**. `scan_mods`, mod-block enforcement, and FO runtime references are not touched.
- **5a's portability section**. 5a (`commission-readme-portable-status-path`) edits the same `SKILL.md` but in different sections. No coordination needed; merge ordering is captain's call (5a is already PR #176; #197 bundles into the same PR per dispatch).

## Bundling

Per the dispatch, this task bundles into 5a's PR #176 once ideation gate passes. Implementation will use 5a's existing worktree on branch `spacedock-ensign/commission-readme-portable-status-path`. No new worktree for ideation.

## Stage Report: ideation

- DONE: Pin the actual current failure shape: re-run `unset CLAUDECODE && uv run pytest tests/test_commission.py -v -s` against the current 5a worktree and enumerate every failing inner check.
  Live re-run against 5a worktree HEAD `8a19c7e6` returned `66 passed, 2 failed (out of 68 checks)`. Two failures named verbatim: `[File Existence] workflow-local pr-merge mod is not generated` and `[No Leaked Template Variables] no leaked template variables`. Stale 3/63 framing reconciled — historical Failure 3 (absolute paths) now passes upstream via 5a commit `2906988e`.
- DONE: Concrete-fix plan: for each currently-failing check, name the file and line where commission is leaking the template variable / generating the wrong path / failing to produce the workflow-local pr-merge mod.
  Failure 1 root-caused to `skills/commission/SKILL.md:482-483` (the `cp "{spacedock_plugin_dir}/mods/pr-merge.md"` instruction). Failure 2 root-caused to two sources: 4 of 5 leaks live inside the byte-identical `_mods/pr-merge.md` copy (resolved by fixing Failure 1); the 5th leak (`refit-command.md` `{current_version}`) is LLM-generated entity prose seeded by SKILL.md's refit-command scaffolding. Architectural conflict named: `scan_mods` only sees `_mods/*.md`, so naive deletion breaks pr-merge runtime — captured as Option A vs Option B with recommendation A.
- DONE: AC items as end-state properties with concrete `Verified by:` clauses. Verification is the existing `test_commission`.
  Six ACs filed (AC1–AC6). AC1, AC2, AC5 verified by existing inner checks in `tests/test_commission.py:79` and `:315-330`. AC3 verified by existing pr-merge fixture tests. AC4 requires one targeted unit test on `scan_mods` opt-out. AC6 captures the decorator-removal-as-follow-up requirement called out in the dispatch — `strict=False` tolerates xpass so leaving it in place is safe.

### Summary

Pinned the live 2/68 failure shape, root-caused both failures to `skills/commission/SKILL.md:482-483` plus LLM seed-entity prose, and surfaced the architectural conflict that simply deleting the `cp` would break pr-merge runtime. Recommended Option A (extend `scan_mods` to discover plugin-shipped mods, with workflow-local shadowing and a captain opt-out path) so the fix lands as a one-shot mechanism upgrade rather than mechanism breakage. Captain ratifies direction at the gate; implementation will share 5a's worktree per dispatch.


### Feedback Cycles

**Cycle 1 — captain reframed at ideation gate (2026-04-30 ~22:25 UTC).**

Captain picked **Option B-ii + A2** after reviewing what was actually leaked in `_mods/pr-merge.md`. The 4 leaks in pr-merge.md (`{number}`, `{branch}` x2, `{entity title}`, `{constructed body}`) are not commission-time slots — they're documentation prose telling the FO what to substitute at merge-hook execution time. The mod is correctly written; the test's leak-scan was overly broad for `_mods/*.md`.

So the architectural shift in Option A (plugin-mods auto-discovery + opt-out mechanism) is unwarranted for this task. Keep the install model (`cp` from plugin source). Refine the test instead.

Captain's chosen direction: **B-ii** — the one-line scope change replaces leak-scan on `_mods/*.md` with a byte-compare against plugin source. Same coverage for "no leaks", plus a freshness signal that catches accidental edits to the local copy or stale-after-plugin-upgrade drift. **A2** for the 5th leak (refit-command.md `{current_version}`) — fix at content source by adjusting commission's seed-entity scaffolding.

Re-routing ideation to rewrite Approach + ACs + Test plan around B-ii + A2.

## Stage Report: ideation (cycle 2 — Option B-ii adopted)

- DONE: Replace cycle-1 architectural-conflict section with a short Captain-decision note recording the B-ii + A2 direction.
  `## Architectural conflict` removed; `## Captain decision (cycle 2): Option B-ii + A2` added with reasoning condensed from captain's framing — pr-merge.md placeholders are correctly authored runtime substitutions, not commission-time slots.
- DONE: Replace cycle-1 approach with a B-ii + A2 approach (test refinements + content-source fix for refit-command prose).
  Four-step approach: (1) leak scan skips `_mods/*.md`, (2) add `[Mod Install Freshness]` byte-compare check, (3) delete/replace the `workflow-local pr-merge mod is not generated` check, (4) tweak commission's refit-command seed scaffolding so the LLM stops writing `{var}` syntax in entity prose.
- DONE: Replace cycle-1 ACs with the new shorter list (drop AC1/AC3/AC4 mechanism-shift items; keep adjusted AC2/AC5/AC6; add byte-compare freshness AC).
  New AC1 (byte-compare freshness), AC2 (leak scan with `_mods/` exclusion), AC3 (refit-command seed prose has no `{var}` patterns — covered by AC2's scan), AC4 (xfail decorator stays, removal as follow-up). All four ACs verified by `tests/test_commission.py` only.
- DONE: Replace cycle-1 test plan with the simpler post-fix expectations (existing test, leak scan now excludes `_mods/`, byte-compare added, expect XPASS).
  Test plan reduced to 5 numbered points; no new test files; no regression coverage outside `test_commission` because mechanism is unchanged.
- DONE: Append `## Stage Report: ideation (cycle 2 — Option B-ii adopted)` at the end of the entity file.
  This section.

### Summary

Captain rejected Option A (architectural shift) at the ideation gate and chose Option B-ii + A2. Reworked the entity body in place: shortened the architectural section into a Captain-decision note, replaced the proposed approach with four concrete test-and-prose refinements scoped to `tests/test_commission.py` and `skills/commission/SKILL.md`, replaced the AC list with four end-state properties verified entirely by the existing extended `test_commission`, and tightened the test plan to match. Implementation surface is now small: leak-scan exclusion, byte-compare freshness check, delete/replace one existence check, and a content-source tweak to the refit-command seed scaffolding. No mechanism changes, no new test files, no regression risk outside `test_commission`.
