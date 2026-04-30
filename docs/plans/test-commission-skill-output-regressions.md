---
id: 197
title: "test_commission: commission skill produces leaked templates, absolute paths, unwanted _mods/pr-merge.md"
status: backlog
source: "PR #131 CI (#154 cycle-1 pre-merge) — after #154 lifted the content-drift xfail and swapped test_commission's static content reads to `assembled_agent_content`, 60/63 inner checks pass; 3/63 remain FAIL on all three claude variants (claude-live, claude-live-bare, claude-live-opus)"
started:
completed:
verdict:
score: 0.7
worktree:
issue:
pr:
mod-block:
---

## Problem

After #154 fixed the test-assertion content-home drift (19/65 → 3/63 on test_commission), three deterministic failures remain across all claude variants. These are commission-skill output-quality regressions, not #154 scope (test-assertion-refresh against skill-preload).

### Failure 1: `workflow-local pr-merge mod is not generated`
Commission generates `v0-test-1/_mods/pr-merge.md` alongside the plugin-shipped `mods/pr-merge.md`. The assertion expects ONLY the plugin-shipped file; commission should not emit a workflow-local duplicate.

### Failure 2: `no leaked template variables`
Generated `pr-merge.md` contains unescaped `{number}`, `{branch}` placeholders:
- `gh pr view {number} --json state --jq '.state'` (this is a gh CLI template variable, legit; test regex may be over-broad)
- `**Branch:** {branch} -> main` (this is a literal template slot that leaked)
- `git push origin {branch}` (leaked template)

Some of these are legit jq / gh-CLI template interpolation; others are unrendered workflow template slots.

### Failure 3: `no absolute paths in generated files`
Generated `README.md` contains the CI runner's absolute path: `/home/runner/work/spacedock/spacedock/skills/commission/bin/status --workflow-dir ./v0-test-1/`. Commission should produce the relative path or rely on `$PATH`-resolved invocation.

## Candidate root causes

1. **pr-merge workflow-local generation**: commission skill instruction or template may still prompt the LLM to scaffold a workflow-local mod copy even when the plugin ships one.
2. **Absolute paths**: commission prompt includes absolute paths in its own context (it's generating README while running from `/home/runner/...`), and the LLM transcribes them. Needs explicit guidance to prefer relative paths.
3. **Leaked templates**: commission template handling inconsistency — some `{var}` slots are expected to survive to the output (gh/jq templates), others are meant to be rendered.

## Out of scope for #154

This task tracks commission-skill output-quality regressions. #154 was strictly a test-assertion refresh and has already landed its scope improvements (84% of test_commission's drift fixed via `assembled_agent_content`).

## Acceptance criteria (provisional)

- `test_commission` passes ≥62/63 on `make test-live-claude` across all three claude variants
- Either commission skill stops emitting the 3 failure classes, OR tests' regexes are refined to distinguish legit `{var}` templates (gh/jq) from leaked slots
- Root cause documented per failure class

## Stage Report: implementation

- DONE: `tests/test_commission.py` leak-scan excludes `_mods/*.md`. AC2 satisfied — leak scan returns empty against fresh commission output for entity-body files.
  Commit `6c0c774a` adds `if "_mods" in md_file.parts: continue` to the rglob walk in `[No Leaked Template Variables]`. Live run: `[No Leaked Template Variables] PASS: no leaked template variables`.
- DONE: `tests/test_commission.py` adds `[Mod Install Freshness]` byte-compare check. AC1 satisfied — workflow-local mods byte-equal plugin source.
  Commit `6c0c774a` adds a new `[Mod Install Freshness]` section that iterates `workflow_dir/_mods/*.md` and `read_bytes()`-compares each against `repo_root/mods/{name}.md`. Live run: `PASS: workflow-local _mods/pr-merge.md matches plugin source byte-for-byte`.
- DONE: `tests/test_commission.py:79` `workflow-local pr-merge mod is not generated` check deleted (under B-ii the file IS expected to exist).
  Commit `6c0c774a` removes the line. Replaced semantically by the `[Mod Install Freshness]` byte-compare.
- DONE: `skills/commission/SKILL.md` refit-command seed-entity scaffolding adjusted so LLM doesn't write `{var}` syntax in entity body prose. AC3 satisfied via AC2's scan covering refit-command.md.
  Commit `fe99ba74` rewrites the body example from `{Description/thesis from {captain}'s seed input.}` to a parenthetical instruction telling the LLM to write plain prose and explicitly to rewrite `{var}` phrasing as natural language or backticked words. Live run: `PASS: no leaked template variables` (refit-command.md included in scan).
- DONE: Local verification — `pytest tests/test_commission.py -v -s` produces XPASS; `make test-static` green.
  `pytest`: `68 passed, 0 failed (out of 68 checks)` → `1 xpassed in 125.79s`. `make test-static`: `539 passed, 26 deselected, 15 subtests passed in 26.98s`.

### Summary

Two surgical commits land the Option B-ii + A2 fix on top of 5a's branch. Test changes cover three of four checklist items: leak scan now skips `_mods/*.md`, a new `[Mod Install Freshness]` block byte-compares each workflow-local mod against its plugin source, and the obsolete `workflow-local pr-merge mod is not generated` existence check is removed (replaced by the freshness check). The SKILL.md tweak rewrites the seed-entity body example so it no longer models brace-syntax in entity prose, which was the source of the `{current_version}`-style leaks in `refit-command.md`. Live `test_commission` run reports `68 passed, 0 failed` (XPASS, tolerated by `strict=False`); static suite stays at 539 passed.

## Stage Report: validation

- DONE: AC1 — workflow-local mods byte-equal plugin source; `[Mod Install Freshness]` check passes.
  Live output: `[Mod Install Freshness] PASS: workflow-local _mods/pr-merge.md matches plugin source byte-for-byte`. Code at `tests/test_commission.py:314-331` (read_bytes byte-compare loop).
- DONE: AC2 — leak scan returns empty for entity-body files (excluding `_mods/*.md`); leak scan skips `_mods/*.md`.
  Live output: `[No Leaked Template Variables] PASS: no leaked template variables`. Skip clause confirmed at `tests/test_commission.py:338-339` (`if "_mods" in md_file.parts: continue`).
- DONE: AC3 — refit-command.md seed prose has no `{var}` patterns; verified via the leak scan covering it.
  AC2 leak scan walks `workflow_dir.rglob("*.md")` (covers refit-command.md) and reports clean. Root-cause fix at `skills/commission/SKILL.md:464` rewrites the seed-entity body example from `{Description/thesis from {captain}'s seed input.}` to a parenthetical instruction explicitly forbidding brace-syntax in entity prose.
- DONE: AC4 — xfail decorator on test_commission stays unchanged.
  Verified by reading `tests/test_commission.py:22`: `@pytest.mark.xfail(strict=False, reason="pending #197 — commission-skill output regressions (template leaks, abs paths, workflow-local pr-merge); see docs/plans/test-commission-skill-output-regressions.md")` — identical to baseline.
- DONE: Live `unset CLAUDECODE && uv run pytest tests/test_commission.py -v -s` and `make test-static` re-runs.
  Live: `68 passed, 0 failed (out of 68 checks)` → `1 xpassed in 113.12s (0:01:53)`. Static: `539 passed, 26 deselected, 15 subtests passed in 27.08s`. Pre-fix baseline `66/68` → post-fix `68/68` confirmed.
- DONE: Spot-check SKILL.md seed-entity scaffolding edit.
  `skills/commission/SKILL.md:464` — body example replaced with "(Body: write the description or thesis from the captain's seed input as plain prose. Do NOT carry brace-syntax placeholders into the body — rewrite any `{var}` phrasing as natural language or backticked words.)" — LLM is no longer modeled to write `{var}` syntax in entity prose. Frontmatter `{var}` slots above are preserved (correct — they are template slots in the YAML scaffold, not entity body prose).

### Verdict

PASSED. All four ACs reproduced from this worktree with live test output. The two #197 commits (`6c0c774a` test refinements, `fe99ba74` SKILL.md seed prose tweak) are surgical and correctly scoped. Live test_commission lifts from `66/68` (xfailed) to `68/68` (xpassed under `strict=False`); static regression suite remains green.
