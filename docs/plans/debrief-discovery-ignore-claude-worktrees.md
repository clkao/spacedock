---
id: s68tqg0gqqyy8hpc2py48gq9
title: "debrief discovery should ignore .claude/worktrees workflow copies"
status: ideation
source: "GitHub issue #174 (filed by Kent Chen / iamcxa, 2026-04-30)"
started: 2026-04-30T19:47:24Z
completed:
verdict:
score: 0.55
worktree:
issue: "#174"
pr:
mod-block:
---

`skills/debrief/SKILL.md:22` runs the workflow-discovery grep with `--exclude-dir=.worktrees` (among others) but does NOT exclude `.claude/worktrees`. In repos where agent-created git worktrees live under `.claude/worktrees/...`, every worktree carries a copy of the workflow README, so debrief discovery returns the primary workflow + N duplicate copies. The captain has to disambiguate even when there's a single intended workflow in the primary checkout.

## Suggested fix

Add `.claude/worktrees` (or equivalent path filtering) to the discovery exclusion list at `skills/debrief/SKILL.md:22`. Two implementation shapes worth ideation:

- **Add another `--exclude-dir=` arg** — `grep`'s `--exclude-dir` matches directory NAME, so `--exclude-dir=worktrees` (without leading dot) would catch `.claude/worktrees/` along with any other dir literally named `worktrees`. Risk: may exclude legitimate user-level `worktrees/` dirs.
- **Post-filter the grep output** — pipe through `grep -v '/\.claude/worktrees/'` or equivalent. More surgical; preserves directory names captains might intentionally use.

Reporter (Kent Chen) flagged this for `spacedock@0.10.2`; the same code path is in current `0.11.0`. No regression test for discovery exclusion behavior currently exists.

## Chosen approach: Option B (post-filter)

Pipe the discovery `grep` through `grep -v '/\.claude/worktrees/'` so any README path that lives under a `.claude/worktrees/` segment is dropped after the recursive scan.

Final command shape at `skills/debrief/SKILL.md:22`:

```bash
grep -rl '^commissioned-by: spacedock@' \
  --include='README.md' \
  --exclude-dir=node_modules \
  --exclude-dir=.worktrees \
  --exclude-dir=.git \
  --exclude-dir=vendor \
  --exclude-dir=dist \
  --exclude-dir=build \
  --exclude-dir=__pycache__ \
  "$project_root" \
  | grep -v '/\.claude/worktrees/'
```

### Why Option B over Option A

- Option A (`--exclude-dir=worktrees`, no leading dot) matches `grep`'s exclude-dir basename rule, so it would also hide any user-committed directory literally named `worktrees/` — e.g. a docs page `worktrees/README.md`. The reporter explicitly named `.claude/worktrees/` as the unwanted path; collateral damage to user-named dirs is not warranted.
- Option B mirrors the reporter's expectation exactly and is path-anchored (`/.claude/worktrees/`), so a sibling directory just named `worktrees/` is preserved.
- Cost of "more moving parts": one extra pipe stage and one regex. Trivial against the precision gained.
- The existing `--exclude-dir=.worktrees` rule remains untouched; it still short-circuits the recursive descent into `.worktrees/` (cheaper than post-filtering), so we keep that and only add the post-filter for the `.claude/worktrees/` case where the descent has already happened by the time we know we want to skip.

Behavioral note: post-filtering preserves filesystem traversal cost into `.claude/worktrees/` (grep still descends and reads matching READMEs). Acceptable: discovery runs once per debrief invocation, and the alternative (adding `--exclude-dir=.claude` blanket) would mask legitimate `.claude/` content the captain might later want to scan.

## Acceptance criteria

- **AC1:** When debrief discovery runs in a repo whose only `commissioned-by: spacedock@` README outside `.claude/worktrees/` is the primary workflow, and one or more identical READMEs exist under `.claude/worktrees/<branch>/...`, discovery returns exactly the primary workflow path and zero `.claude/worktrees/` paths.
  Verified by: regression test `tests/test_debrief_discovery_excludes_claude_worktrees.py` (see test plan) asserts the post-filtered command output contains the primary path and no path containing `/.claude/worktrees/`.
- **AC2:** The existing `.worktrees/` exclusion continues to suppress READMEs that live under a top-level `.worktrees/` directory.
  Verified by: same regression test seeds a `.worktrees/<slug>/README.md` with `commissioned-by: spacedock@` frontmatter and asserts that path is also absent from discovery output.
- **AC3:** A user-committed directory literally named `worktrees/` (no leading dot, not under `.claude/`) is NOT excluded by the discovery filter.
  Verified by: same regression test seeds `worktrees/docs/README.md` with the marker frontmatter and asserts it IS present in discovery output, confirming the post-filter is path-anchored to `/.claude/worktrees/` and does not over-exclude.
- **AC4:** The discovery command at `skills/debrief/SKILL.md:22` retains all current `--exclude-dir=` flags (`node_modules`, `.worktrees`, `.git`, `vendor`, `dist`, `build`, `__pycache__`) — the fix is additive, not a rewrite.
  Verified by: a content assertion in the regression test that the SKILL.md line still contains every original `--exclude-dir=` token, plus the new pipe-to-`grep -v '/\.claude/worktrees/'` suffix.

## Test plan

### Regression test fixture and entrypoint

- **New test file:** `tests/test_debrief_discovery_excludes_claude_worktrees.py`
- **Fixture shape (built inside `tmp_path`):**
  ```
  tmp_path/
    .git/                                     # `git init` so `git rev-parse --show-toplevel` works
    workflows/
      planning/
        README.md                             # frontmatter: commissioned-by: spacedock@0.11.0
    .claude/worktrees/
      ensign-foo/workflows/planning/README.md # duplicate (same frontmatter)
      ensign-bar/workflows/planning/README.md # duplicate (same frontmatter)
    .worktrees/
      legacy-slug/workflows/planning/README.md # duplicate (same frontmatter, must stay excluded)
    worktrees/
      docs/README.md                          # marker frontmatter, must NOT be excluded
  ```
- **Test entrypoint:** a single pytest function `test_discovery_filters_claude_worktrees` that:
  1. Reads the literal command from `skills/debrief/SKILL.md` line 22 (or extracts it via a small parse) and runs it against the fixture's `tmp_path` as `$project_root`.
  2. Asserts: result contains `workflows/planning/README.md` from the primary checkout AND `worktrees/docs/README.md`; result contains zero entries with `/.claude/worktrees/` or `/.worktrees/` in the path.
  3. A second assertion (AC4) parses the SKILL.md line and confirms the original `--exclude-dir=` token set is preserved.

### Manual smoke

- In a real working spacedock repo with active `.claude/worktrees/` entries, run the discovery `grep` command from SKILL.md by hand, confirm it returns only the primary workflow path.
- Run `/spacedock:debrief` (no argument) and confirm Phase 1 Step 1 reports a single workflow without prompting for disambiguation.

### Non-regression scope

- No other skills reference `.claude/worktrees` exclusion today (`grep -rl` confirmed only `skills/debrief/SKILL.md` matches `exclude-dir`). Scope of the change is limited to that one line.

## Stage Report: ideation

- DONE: Pick one of the two suggested fix shapes (A: add `--exclude-dir=worktrees` (without leading dot, broader), B: post-filter grep output to drop `/.claude/worktrees/` paths) — name which and justify. Option B is more surgical but has more moving parts; option A is one-line but excludes any user-named `worktrees/` dir. Trade-off should be made explicit.
  Chose Option B. Path-anchored post-filter `grep -v '/\.claude/worktrees/'` matches the reporter's expectation exactly and preserves user-committed `worktrees/` dirs that Option A would over-exclude. Trade-off (one extra pipe stage vs. precision) recorded under "Why Option B over Option A".
- DONE: Test plan names a regression test fixture (a repo with `.claude/worktrees/` containing duplicate workflow READMEs) and the test entrypoint. Existing `.worktrees` exclusion must continue to work.
  Fixture spec written (primary workflow + two `.claude/worktrees/<branch>/.../README.md` duplicates + `.worktrees/legacy-slug/.../README.md` + sibling `worktrees/docs/README.md` to guard against over-exclusion). Entrypoint: `tests/test_debrief_discovery_excludes_claude_worktrees.py::test_discovery_filters_claude_worktrees`.
- DONE: AC items are end-state properties with concrete `Verified by:` clauses (discovery returns no `.claude/worktrees/` paths; existing `.worktrees` exclusion preserved; regression fixture passes).
  Four ACs written, each with a `Verified by:` clause pointing at concrete assertions in the regression test (presence of primary path, absence of `.claude/worktrees/` and `.worktrees/` paths, presence of sibling `worktrees/docs/`, preservation of every original `--exclude-dir=` token).

### Summary

Fix shape selected: post-filter the existing discovery `grep` with `grep -v '/\.claude/worktrees/'` rather than adding `--exclude-dir=worktrees`, so that user-committed sibling directories named `worktrees/` are not incidentally hidden. Acceptance criteria express end-state properties (discovery output contents and SKILL.md command shape) and each is verified by a single new pytest regression test with a fixture covering all four sensitivity classes (primary workflow, `.claude/worktrees/` duplicate, `.worktrees/` duplicate, user `worktrees/` sibling). The change is additive to one line in `skills/debrief/SKILL.md:22` and does not touch other skills.
