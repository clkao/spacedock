---
id: s68tqg0gqqyy8hpc2py48gq9
title: "debrief discovery should ignore .claude/worktrees workflow copies"
status: validation
source: "GitHub issue #174 (filed by Kent Chen / iamcxa, 2026-04-30)"
started: 2026-04-30T19:47:24Z
completed:
verdict:
score: 0.55
worktree: .worktrees/spacedock-ensign-debrief-tolerate-missing-workflow-status
issue: "#174"
pr: #177
mod-block: merge:pr-merge
---

`skills/debrief/SKILL.md:22` runs the workflow-discovery grep with `--exclude-dir=.worktrees` (among others) but does NOT exclude `.claude/worktrees`. In repos where agent-created git worktrees live under `.claude/worktrees/...`, every worktree carries a copy of the workflow README, so debrief discovery returns the primary workflow + N duplicate copies. The captain has to disambiguate even when there's a single intended workflow in the primary checkout.

## Suggested fix (intake — superseded by cycle 2 reframe)

> **Note:** The two shapes below were the original intake framing. Cycle-1 ideation chose Option B (post-filter the inline `grep`); cycle-2 ideation reframed to delegate Phase 1 to `status --discover` and fix the exclusion at the canonical surface. Kept here for audit. The "Chosen approach" section below is authoritative.

Add `.claude/worktrees` (or equivalent path filtering) to the discovery exclusion list at `skills/debrief/SKILL.md:22`. Two implementation shapes worth ideation:

- **Add another `--exclude-dir=` arg** — `grep`'s `--exclude-dir` matches directory NAME, so `--exclude-dir=worktrees` (without leading dot) would catch `.claude/worktrees/` along with any other dir literally named `worktrees`. Risk: may exclude legitimate user-level `worktrees/` dirs.
- **Post-filter the grep output** — pipe through `grep -v '/\.claude/worktrees/'` or equivalent. More surgical; preserves directory names captains might intentionally use.

Reporter (Kent Chen) flagged this for `spacedock@0.10.2`; the same code path is in current `0.11.0`. No regression test for discovery exclusion behavior currently exists.

## Chosen approach: delegate debrief Phase 1 to `status --discover`, fix the exclusion at the canonical surface

Two changes, in one PR:

1. **`skills/commission/bin/status` (canonical fix)**: extend `discover_workflows` so the `os.walk` prune drops `worktrees` whenever the parent directory is `.claude`. Concretely, after the existing `dirnames[:] = [d for d in dirnames if d not in DISCOVER_IGNORE_DIRS]` line at `skills/commission/bin/status:1885`, add a path-anchored guard: when `os.path.basename(dirpath) == '.claude'`, also remove `'worktrees'` from `dirnames`. This mirrors the reporter's expectation (skip the `.claude/worktrees/` segment specifically) without expanding `DISCOVER_IGNORE_DIRS` to a broad `worktrees` basename rule that would clobber user-committed `worktrees/` dirs.
2. **`skills/debrief/SKILL.md:22` (delegate)**: replace the inline `grep -rl ...` recipe with an invocation of the canonical discovery surface. Phase 1 Step 1 becomes "run `{spacedock_plugin_dir}/skills/commission/bin/status --discover` (it already defaults `--root` to `git rev-parse --show-toplevel`); each line of stdout is an absolute resolved workflow directory."

### Why this over the cycle-1 post-filter

- **Single source of truth for discovery exclusion.** `status --discover` is the canonical workflow-discovery surface (used by the FO boot path; `DISCOVER_IGNORE_DIRS` at `skills/commission/bin/status:1847` is documented as the canonical ignore set). Debrief currently re-implements that scan inline with a slightly-different `--exclude-dir` set. Adding a `grep -v` post-filter to debrief's private scan would leave two divergent exclusion lists in the codebase.
- **Future fixes propagate for free.** Once debrief delegates, every consumer of `status --discover` (FO, debrief, anything we add later) inherits exclusion changes from one edit.
- **The `.claude/worktrees/` fix lives at the right altitude.** It applies to all discovery consumers, not just debrief.
- **The cycle-1 surgical-precision concern (don't over-exclude user `worktrees/`) is preserved.** The path-anchored prune (`basename(dirpath) == '.claude'` then drop `worktrees`) is exactly as precise as the cycle-1 `grep -v '/\.claude/worktrees/'`, but lives in Python where `os.walk` can also short-circuit descent (no wasted traversal into the worktree copies).

### Why NOT broaden `DISCOVER_IGNORE_DIRS` to include `'worktrees'`

`DISCOVER_IGNORE_DIRS` is a basename ignore set; adding `'worktrees'` would prune any directory literally named `worktrees/` anywhere in the tree, not just `.claude/worktrees/`. This is the same trade-off as cycle-1's Option A and was rejected for the same reason — collateral damage to user-committed `worktrees/` dirs (e.g. a docs subdir).

### Why NOT broaden `DISCOVER_IGNORE_DIRS` to include `'.claude'`

`.claude/` may host other content the captain wants discoverable in the future. The path-anchored prune limits the change to exactly what the reporter described.

### Concrete edits

- `skills/commission/bin/status` near line 1885: after the existing `DISCOVER_IGNORE_DIRS` filter, add the path-anchored prune for `.claude/worktrees`. Update the docstring at line 1861 to add `.claude/worktrees` to the documented ignored-paths list.
- `skills/debrief/SKILL.md:21-23` (Phase 1 Step 1): replace the inline `grep -rl ...` recipe with a call to `{spacedock_plugin_dir}/skills/commission/bin/status --discover`. Output is one resolved workflow directory per line; existing "exactly one / multiple / none" branching at the end of Step 1 still applies.

### Why we are NOT merging with #8x

- **#8x is already approved and ready to implement.** Its ideation gate passed; merging would unwind that decision and force a fresh gate on a larger combined scope.
- **Different code paths.** #174 (this task) edits Phase 1 Step 1 + the status discover function. #8x edits Phase 2e (extraction). They touch different lines of `skills/debrief/SKILL.md` and #8x explicitly scopes itself away from #174 in its AC4 ("Adjacent #174 ... and #5a ... remain independent").
- **Different fix shapes.** This task delegates to an existing `status` subcommand and fixes one bug at the canonical site. #8x rewrites Phase 2e prose to add a primary/legacy/degraded fallback chain for the case where the local `{dir}/status` is missing. The narratives rhyme ("debrief should defer to status helpers") but the implementations don't share a line of code.
- **Sequencing is fine in either order.** Neither change depends on the other; landing them in two cycles is cheap.
- **Acknowledged narrative overlap.** Both cycles trend toward "debrief delegates to canonical status surfaces." That direction is recorded here and in #8x; we just don't bundle them into one PR.

## Acceptance criteria

- **AC1:** `status --discover` does not return any path under `.claude/worktrees/`.
  Verified by: new pytest in `tests/test_status_discover_ignores_claude_worktrees.py` builds a `tmp_path` containing a primary workflow README plus duplicate workflow READMEs under `.claude/worktrees/<branch>/.../README.md`, then invokes `skills/commission/bin/status --discover --root <tmp_path>` as a subprocess and asserts no output line contains `/.claude/worktrees/`.
- **AC2:** Existing `.worktrees/` exclusion (and the rest of `DISCOVER_IGNORE_DIRS`) continue to work — no regression.
  Verified by: same test seeds `.worktrees/<slug>/.../README.md` and `node_modules/foo/README.md` with the `commissioned-by: spacedock@` marker; asserts both are absent from `status --discover` output.
- **AC3:** A user-committed directory literally named `worktrees/` (no leading dot, not under `.claude/`) is still discoverable — the prune is path-anchored to `.claude/worktrees`, not basename-broad.
  Verified by: same test seeds `worktrees/docs/README.md` with the marker; asserts the resolved path is present in `status --discover` output.
- **AC4:** `skills/commission/bin/status` docstring at `discover_workflows` (lines ~1861-1862) lists `.claude/worktrees` alongside `.worktrees` in the documented ignored-paths set, so the canonical exclusion list is self-describing.
  Verified by: grep guard in the same test asserts the docstring substring `.claude/worktrees` appears in `skills/commission/bin/status`.
- **AC5:** Debrief Phase 1 Step 1 delegates to `status --discover` rather than running its own `grep -rl ...`.
  Verified by: grep guards on `skills/debrief/SKILL.md` — must contain `status --discover`; must NOT contain the legacy `grep -rl '^commissioned-by: spacedock@'` recipe nor the `--exclude-dir=` flag list. The branching prose ("exactly one / multiple / none") at the end of Step 1 is preserved.
- **AC6:** Other consumers of `discover_workflows` (notably `spacedock:first-officer` boot via `status --boot` adjacency) continue to work.
  Verified by: existing tests under `tests/` that exercise `status --discover` / FO boot continue to pass. No new regression suite needed for this AC; the existing suite is the verifier.

## Test plan

### Regression test entrypoint and fixture

- **New test file:** `tests/test_status_discover_ignores_claude_worktrees.py`
- **Fixture shape (built inside `tmp_path` per test invocation; no permanent fixture directory needed since the file content is trivial markdown):**
  ```
  tmp_path/
    workflows/planning/README.md                # commissioned-by: spacedock@<current>
    .claude/worktrees/
      ensign-foo/workflows/planning/README.md   # duplicate marker
      ensign-bar/workflows/planning/README.md   # duplicate marker
    .worktrees/
      legacy/workflows/planning/README.md       # duplicate marker (must stay excluded)
    worktrees/
      docs/README.md                            # marker (must NOT be excluded)
    node_modules/
      foo/README.md                             # marker (must stay excluded — sanity check on existing rules)
  ```
- **Test function** `test_discover_excludes_claude_worktrees`:
  1. Build the fixture tree, write the marker frontmatter into each README.
  2. Subprocess-invoke `python skills/commission/bin/status --discover --root <tmp_path>`, capture stdout.
  3. Assert `<tmp_path>/workflows/planning` is in output.
  4. Assert `<tmp_path>/worktrees/docs` is in output (AC3).
  5. Assert no output line contains `/.claude/worktrees/`, `/.worktrees/`, or `/node_modules/` (AC1, AC2).
- **Test function** `test_discover_workflows_docstring_documents_claude_worktrees`:
  1. Read `skills/commission/bin/status`, assert the substring `.claude/worktrees` appears (AC4).
- **Test function** `test_debrief_skill_delegates_to_status_discover`:
  1. Read `skills/debrief/SKILL.md`, assert the substring `status --discover` is present and the literal substring `grep -rl '^commissioned-by: spacedock@'` is absent (AC5).

### Manual smoke

- In a real spacedock repo with `.claude/worktrees/` entries, run `skills/commission/bin/status --discover` and confirm exactly the primary workflow path is returned.
- Run `/spacedock:debrief` with no argument and confirm Phase 1 Step 1 reports a single workflow without prompting for disambiguation.

### Non-regression scope

- The change in `skills/commission/bin/status` is additive (one line of prune logic, one docstring line). All existing `DISCOVER_IGNORE_DIRS` behavior is preserved.
- The change in `skills/debrief/SKILL.md` is a Phase 1 Step 1 rewrite; downstream debrief phases (2a-2f, 3, 4) are untouched.
- No other skills reference `exclude-dir` for `.claude/worktrees/`; verified via `grep -rl exclude-dir skills/`.

## Stage Report: ideation

- DONE: Pick one of the two suggested fix shapes (A: add `--exclude-dir=worktrees` (without leading dot, broader), B: post-filter grep output to drop `/.claude/worktrees/` paths) — name which and justify. Option B is more surgical but has more moving parts; option A is one-line but excludes any user-named `worktrees/` dir. Trade-off should be made explicit.
  Chose Option B. Path-anchored post-filter `grep -v '/\.claude/worktrees/'` matches the reporter's expectation exactly and preserves user-committed `worktrees/` dirs that Option A would over-exclude. Trade-off (one extra pipe stage vs. precision) recorded under "Why Option B over Option A".
- DONE: Test plan names a regression test fixture (a repo with `.claude/worktrees/` containing duplicate workflow READMEs) and the test entrypoint. Existing `.worktrees` exclusion must continue to work.
  Fixture spec written (primary workflow + two `.claude/worktrees/<branch>/.../README.md` duplicates + `.worktrees/legacy-slug/.../README.md` + sibling `worktrees/docs/README.md` to guard against over-exclusion). Entrypoint: `tests/test_debrief_discovery_excludes_claude_worktrees.py::test_discovery_filters_claude_worktrees`.
- DONE: AC items are end-state properties with concrete `Verified by:` clauses (discovery returns no `.claude/worktrees/` paths; existing `.worktrees` exclusion preserved; regression fixture passes).
  Four ACs written, each with a `Verified by:` clause pointing at concrete assertions in the regression test (presence of primary path, absence of `.claude/worktrees/` and `.worktrees/` paths, presence of sibling `worktrees/docs/`, preservation of every original `--exclude-dir=` token).

### Summary

Fix shape selected: post-filter the existing discovery `grep` with `grep -v '/\.claude/worktrees/'` rather than adding `--exclude-dir=worktrees`, so that user-committed sibling directories named `worktrees/` are not incidentally hidden. Acceptance criteria express end-state properties (discovery output contents and SKILL.md command shape) and each is verified by a single new pytest regression test with a fixture covering all four sensitivity classes (primary workflow, `.claude/worktrees/` duplicate, `.worktrees/` duplicate, user `worktrees/` sibling). The change is additive to one line in `skills/debrief/SKILL.md:22` and does not touch other skills.

### Feedback Cycles

**Cycle 1 — rejected at ideation gate.** Captain raised two reframes:
1. *Don't reinvent discovery.* `skills/commission/bin/status --discover` is the canonical workflow-discovery surface (used by FO boot via `status --boot`). Debrief's inline `grep -rl ...` duplicates it; adding a `grep -v` post-filter creates a second exclusion list that will drift from the canonical one in `DISCOVER_IGNORE_DIRS` at `skills/commission/bin/status:1847`. Re-ideate around delegating Phase 1 Step 1 to `status --discover` and fixing the exclusion at the canonical surface.
2. *Consider merging with #8x.* `#8x debrief-tolerate-missing-workflow-status` has a thematically related "debrief should defer to canonical status helpers" shape. Captain asked whether to merge.

Cycle-2 response (recorded in the new "Chosen approach" section above): adopted reframe 1 (delegate to `status --discover` + fix exclusion at canonical site). Declined merge with #8x: it is already approved at its ideation gate, touches a different code path (Phase 2e extraction, not Phase 1 discovery), and unwinding its gate to bundle would cost more than the narrative-coherence win.

## Stage Report: ideation (cycle 2)

- DONE: Reframe 1 — investigate `status --discover` and decide between (a) delegate or (b) keep post-filter with structural justification.
  Chose (a) delegate. `status --discover` exists at `skills/commission/bin/status:1850-1902` with the canonical `DISCOVER_IGNORE_DIRS` set at line 1847. It defaults `--root` to `git rev-parse --show-toplevel`, emits one absolute resolved workflow dir per line on stdout, and is incompatible with all other status flags so its surface is stable. Debrief Phase 1 Step 1 can consume it directly.
- DONE: Reframe 1 — locate and apply the exclusion fix at the canonical site.
  Fix moves to `skills/commission/bin/status` at the `discover_workflows` walk loop near line 1885. After the existing `DISCOVER_IGNORE_DIRS` basename prune, add a path-anchored guard: when `os.path.basename(dirpath) == '.claude'`, also drop `'worktrees'` from `dirnames`. This short-circuits descent into `.claude/worktrees/` without expanding the basename ignore set. Docstring at line 1861-1862 updated to list `.claude/worktrees` alongside `.worktrees`.
- DONE: Reframe 2 — decide whether to merge with #8x; flag trade-off explicitly.
  Stay separate. #8x is already past its ideation gate, scopes itself away from #174 in its AC4, and edits Phase 2e (extraction) while this task edits Phase 1 (discovery) plus `skills/commission/bin/status`. The two share narrative direction ("debrief delegates to canonical status surfaces") but no implementation lines. Merging would unwind an approved gate; sequential cycles are cheap.
- DONE: Rewrite Approach + ACs + Test plan around the reframed direction; preserve the original "Suggested fix" prose for audit.
  "Suggested fix (intake)" section flagged as superseded; "Chosen approach" rewritten to describe the canonical-site fix plus debrief delegation. Six ACs cover discovery exclusion (AC1), no-regression on existing prunes (AC2), no over-exclusion of user `worktrees/` (AC3), self-describing docstring (AC4), debrief delegation (AC5), and FO-boot non-regression via existing tests (AC6).
- DONE: Add `### Feedback Cycles` section after the prior Stage Report.
  Cycle 1 entry added with both reframes and the cycle-2 response summary.

### Summary

Reframed from "patch debrief's private discovery `grep`" to "delegate debrief Phase 1 to the canonical `status --discover` surface and fix the `.claude/worktrees/` exclusion inside `skills/commission/bin/status`." The fix is path-anchored (drop `worktrees` from dirnames only when the parent is `.claude`), preserving the cycle-1 precision concern (no over-exclusion of user `worktrees/`) while putting the rule at the right altitude. Six ACs cover the canonical fix, debrief delegation, and FO-boot non-regression; one new pytest file with three small functions verifies them. Declined merge with #8x — different code paths, #8x already approved, sequencing is cheap.

### Feedback Cycles

**Cycle 1 — captain rejected validation gate (2026-04-30 ~22:00 UTC) for three reframes (s6 portion).**

The captain rejected PR #177 (combined 8x+s6) at validation. For s6 specifically: the path-anchored prune in `discover_workflows` is too special-case. Captain's framing: `.claude/worktrees/` should already be in `.gitignore`; the discovery mechanism shouldn't reinvent its own exclusion list.

Captain's chosen direction (after we discussed A/B/C options): **just add the contents of `.gitignore` to `DISCOVER_IGNORE_DIRS` for now.** Pragmatic "ignore set augmented from gitignore." Trade-off accepted: basename matching for paths like `.claude/worktrees/` over-excludes any sibling `worktrees/` dir (same trade-off as cycle-1 Option A which we rejected then). Captain marks this as "for now" — willingness to revisit.

Cycle-2 implementation:
1. Drop the path-anchored prune at `skills/commission/bin/status:~1892` (`if os.path.basename(dirpath) == '.claude' ...`)
2. Update `discover_workflows` to read `{git_root}/.gitignore`, parse directory-pattern entries, merge their basenames into `DISCOVER_IGNORE_DIRS` at startup
3. Add `.claude/worktrees/` to the repo's `.gitignore` so the ignore set picks it up
4. Update `tests/test_status_discover_ignores_claude_worktrees.py` to seed `.gitignore` in the fixture and verify gitignore-derived exclusions work (the existing 3 sensitivity classes still apply)
5. Update `discover_workflows` docstring to describe the gitignore augmentation

Bundles with 8x cycle-2 work (drop fallbacks, drop static tests). Same worktree, same PR.
