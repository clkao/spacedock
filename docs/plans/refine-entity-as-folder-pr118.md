---
id: 187
title: "Refine and land PR #118 — entity-as-folder support (Karen Hsieh)"
status: validation
source: "external contribution from Karen Hsieh (@ijac13) — PR #118 closes GitHub issue #99 with entity-as-folder scanner support. Spacedock-workflow entity filed to track refinement + review through our standards while preserving contributor attribution on merge."
started: 2026-04-18T00:54:30Z
completed:
verdict:
score: 0.6
worktree: .worktrees/spacedock-ensign-refine-entity-as-folder-pr118
issue: "#99"
pr: "#118"
mod-block:
---

## Contributor

Karen Hsieh (@ijac13 on GitHub) is the original author and owns the authorship on every commit in `feat/entity-as-folder`. Any refinement we apply must be **additive on top of her branch** (or squash-merge with `Co-authored-by: Karen Hsieh <...>` trailer preserved) — we do NOT rewrite her commits with our authorship. Contributor attribution is load-bearing.

## Why this matters

GitHub issue #99 (filed by CL in 0.9.4) proposed first-class entity-as-folder support so workflows producing many artifacts per entity can organize them under `{slug}/index.md` without breaking the scanner. Karen's PR #118 implements this:

- Scanner descends into `{slug}/index.md` when flat `{slug}.md` is absent.
- `status --set` resolves folder entities by the same rule.
- `status --archive` moves the whole folder to `_archive/{slug}/` and stamps `archived:` inside the inner index.
- `--archived`, `--boot`, `--next`, `--next-id`, `--where` inherit via shared discovery.
- Reserved subdirs (`_archive`, `_mods`) excluded from entity detection.
- Conflict case (both flat + folder present) warns on stderr and prefers folder.
- Flat-file workflows unchanged; 444 tests passing locally per PR body.

This entity exists because **we want to refine the submission through our workflow** before landing — reviewing the design against our standards, verifying coverage, potentially requesting scope adjustments.

## Scope for ideation

- Read the PR diff in full; inventory each change vs the issue-#99 spec.
- Identify refinements (wording, test coverage gaps, edge cases not yet exercised, API-surface questions) to request from Karen OR apply ourselves on a follow-up commit.
- Distinguish "must-fix before merge" refinements from "nice-to-have follow-up."
- Produce a concrete change request list for the PR review, or a clear PASSED-as-is acceptance.
- Confirm CI strategy: PR #118's `Runtime Live E2E` run (24555916111) is currently WAITING for env approval; decide whether to approve before or after ideation.

## Merge strategy

- Preserve authorship. Default path: merge-commit (no squash) so every commit keeps Karen as author.
- If squash is preferred for linear-history reasons, the squash commit MUST include `Co-authored-by: Karen Hsieh <email-from-commit>` in the trailer.
- The FO's `pr-merge` mod's PR-body template does not fit here (this is an external PR, not one we authored). The merge-hook's push step is skipped — the branch is already pushed by the contributor.

## Out of scope

- Rewriting Karen's commits to look like ours.
- Adding a spacedock-workflow-style entity body INSIDE `feat/entity-as-folder` — that's our internal convention, not hers.
- Forcing PR #118 through our ideation-gate-then-implementation pattern — implementation already exists; this workflow entity covers review + refinement only.

## Cross-references

- GitHub issue #99 — the original feature request (authored by CL, 0.9.4)
- PR #118 — the implementation (authored by Karen Hsieh / @ijac13, open)
- CI run 24555916111 — PR #118's Runtime Live E2E, currently WAITING for env approval

## Stage Report (ideation)

### 1. Read seed entity body — DONE
Read Contributor, Scope for ideation, Merge strategy, Out of scope, Cross-references sections. Contributor attribution (Karen Hsieh / @ijac13) and merge-commit-preferred path noted and respected.

### 2. Read PR #118 + issue #99 — DONE
- PR: 2 commits, +549/-40 across 3 files (`skills/commission/SKILL.md`, `skills/commission/bin/status`, `tests/test_status_script.py`). Both commits carry Karen as author with Claude co-author. `mergeable: MERGEABLE`, `mergeStateStatus: UNSTABLE` (live-CI failure; see #7).
- Issue #99: authored by CL in 0.9.4. Proposes `{slug}/index.md` as a directory-as-entity convention. Karen's comment on 2026-04-17 confirms she's hitting it on the release workflow and plans to implement the exact proposal.

### 3. Problem statement review — DONE
Issue #99 remains accurate. The proposal text matches what PR #118 ships. Edge cases not enumerated in the issue but correctly addressed by the PR: reserved-subdir exclusion (`_archive`, `_mods`), dotfile-subdir exclusion, conflict handling when both forms exist, worktree-path preservation for folder entities, and `--archived` listing of folder-archived entries. One edge case the issue didn't name that the PR handles via `_worktree_mirror_path`: worktree-backed folder entities map `{slug}/index.md` in the pipeline dir to `{worktree}/{slug}/index.md` inside the worktree, not `{worktree}/index.md`.

### 4. Solution approach review — DONE
Per-change classification (against issue #99):
- `discover_entity_files()` — (a) directly satisfies issue.
- `resolve_entity_path()` — (a) directly satisfies issue (`--set` folder-aware).
- `scan_entities` / `scan_entities_active` delegation to discovery — (b) necessary plumbing.
- `_worktree_mirror_path` + `resolve_active_entity_path(pipeline_dir=...)` + `load_active_entity_fields(pipeline_dir=...)` — (b) necessary plumbing; the issue didn't spell this out but folder-entity + worktree is a correctness requirement. Back-compat preserved via `pipeline_dir=None` default.
- `run_archive` folder branch — (a) directly satisfies issue.
- `RESERVED_SUBDIRS = frozenset({'_archive', '_mods'})` — (b) necessary plumbing; correct choice (keeps guard centralized).
- SKILL.md File Naming rewrite — (a) directly satisfies issue (operator discovery UX).
- Header-docstring expansion in `status` — (b) necessary plumbing (documents discovery rule).

No (c) scope-expansion or (d) missing-from-issue changes. The PR is scope-disciplined.

### 5. Load-bearing design question: how does the user choose between flat and folder? — DONE
**Options evaluated:**

**(a) Auto-detection** (current PR behavior): Scanner looks at filesystem; folder wins on conflict with stderr warning.
- Pros: zero operator burden; no new frontmatter field; no commission-skill template change; new operators discover the capability from SKILL.md alone; mixed workflows work for free.
- Cons: operator has no upfront signal about which form this workflow "prefers"; conflict case depends on a warning (silently tolerates duplication); parallel operators could each pick a different form for the same entity and produce confusion.
- Implication for existing workflows: none — flat-file scan is byte-for-byte preserved.
- Discovery UX: good — SKILL.md documents both forms as first-class.

**(b) Explicit workflow-level declaration** in the workflow's README frontmatter (`entity-style: flat | folder | mixed`):
- Pros: grep-able; enforceable at commission time; removes ambiguity for parallel operators; toolchain can warn when operator creates the wrong form.
- Cons: **scope expansion beyond what PR #118 implements** — requires README-frontmatter parsing in the status script, commission-skill template updates to emit the field, migration consideration for every existing workflow (default to `flat` or require explicit declaration?), and `refit`-skill changes to backfill. Likely +80-150 lines of plumbing and a new required-field choice.
- Implication for existing workflows: either silent default-to-`flat` (fine) or explicit-required (breaking). A breaking choice here is a bad trade for a minor UX gain.
- Discovery UX: slightly better — `entity-style: folder` in the README tells a new operator up front.

**(c) Per-entity declaration**: rejected — a file's frontmatter cannot describe the file's own name, and requiring a pointer file before the directory is self-defeating.

**Recommendation: option (a) as PR #118 ships, with two refinements** — details in the refinement list below. Option (b) is explicitly NOT recommended at this time; it's a scope expansion that doesn't pay its own way. If conflict-warning spam becomes a real operator pain point in practice, revisit (b) as a focused follow-up.

Rationale: issue #99 asks for both forms to be "first-class" and "backwards-compatible with file-per-entity workflows" — auto-detection delivers both. Option (b) would require commission-skill changes that the PR explicitly avoids, expanding the surface area beyond the issue.

### 6. Test coverage audit — DONE
PR #118 adds `TestEntityAsFolder` (12 tests) + `TestStatusDocstringEntityFolder` (1 test). Existing 138-case flat-file suite remains green. 444 tests reported passing locally.

**Gaps identified (nice-to-have, NOT merge-blocking):**

| # | Test name | Shape |
|---|---|---|
| T1 | `test_set_on_folder_entity_with_worktree_reads_worktree_index` | `--set` resolves to `{worktree}/{slug}/index.md` when entity has `worktree:` field. Exercises `_worktree_mirror_path`. |
| T2 | `test_archive_folder_entity_while_worktree_present` | Archival of a folder entity whose main copy has `worktree:` set — archive stamps and moves the main-side folder; worktree copy untouched. |
| T3 | `test_archive_folder_entity_refused_under_mod_block` | Mod-block guard in `run_archive` fires for folder form (symmetry with flat-form mod-block test). |
| T4 | `test_archive_warns_on_conflict_when_both_forms_present` | `--archive` emits the "preferring folder" warning on stderr when both `{slug}.md` and `{slug}/index.md` exist (parallel to the discovery/`--set` conflict tests). |
| T5 | `test_dotfile_subdir_not_treated_as_entity` | A `.hidden/index.md` directory is not discovered (pins the `startswith('.')` guard). |
| T6 | `test_flat_only_workflow_regression_after_folder_support` | Explicit regression: a flat-only workflow with no folder dirs produces byte-identical output to pre-PR behavior for default overview and `--next-id`. |

**Hygiene note (not blocking):** `_read_frontmatter` in the new test module duplicates `parse_frontmatter` from the script. Since tests `exec` the built script, sharing the helper isn't practical — leave as-is.

**Integration points covered:** `--next`, `--next-id`, `--boot`, `--where`, `--archived`, `--set`, `--archive` all exercised for folder form. Mod-block integration NOT exercised for folder form (→ T3).

### 7. Claude-live CI failure diagnosis — DONE
**Cause: structural — fork-PR secret gating, NOT a code regression.**

Evidence:
- Run 24555916111: 3 of 5 jobs (`claude-live`, `claude-live-bare`, `claude-live-opus`) failed at step `Check required secret` in 3-6 seconds, well before any test code executes. `codex-live` is pending; `static-offline` passed.
- `.github/workflows/runtime-live-e2e.yml` lines 51-56, 216-221, 381-386: each job's `Check required secret` step `exit 1`s when `ANTHROPIC_API_KEY` is empty.
- On PRs from fork branches (or any branch not yet approved into the live environment), GitHub Actions does not inject environment secrets until an approver authorizes the deployment. The secret is therefore empty and the guard trips.
- Artifact-upload warnings (`No files were found ... No artifacts will be uploaded`) are downstream of the early exit — the live suite never ran.

Resolution: NONE required on the PR branch. The failure is expected pre-approval state. Once CL (or another approver) approves the `runtime-live` environment for this PR, a re-run will have the secrets and the check will pass. The PR's code changes are orthogonal to this failure.

Recommendation to captain: approve the environment for PR #118 before merge so we see a real claude-live result. If claude-live passes post-approval, merge. If it fails post-approval, that's a real regression and we bounce.

### 8. Refinement request list — DONE

**Must-fix before merge:** NONE.

**Nice-to-have follow-up (any or all can be handled by an additive commit on Karen's branch with `Co-authored-by` attribution, or by a follow-up PR we author):**

1. (Design) Strengthen the conflict-resolution message — extend the current stderr warning to name the remediation command (e.g., "Remove `{flat_path}` or move its contents into `{folder_path}/` to silence this warning"). Tiny edit, pure UX. *Follow-up PR or additive commit.*
2. (Tests) Add T1-T6 from step 6. Highest value are T3 (mod-block × folder) and T6 (explicit flat-only regression pin). *Follow-up additive commit OR separate PR we author post-merge.*
3. (Docs) No change needed — SKILL.md wording already captures both forms clearly and distinguishes "default" (flat) from "when the entity produces per-stage artifacts" (folder).
4. (Design, NOT for this PR) Explicit workflow-level `entity-style` declaration — filed as a "revisit if conflict-warning spam becomes real pain" note; no action unless operators report it.

**Explicitly NOT requested:** any rewrite of Karen's commits, any reorganization of the `discover_entity_files` / `resolve_entity_path` API, any commission-skill README-frontmatter scope expansion.

### 9. Merge path — DONE
Merge strategy section in the entity body is still correct. Confirm:
- Default path: merge-commit (no squash), preserving authorship on both of Karen's commits.
- If CL prefers squash, the squash commit MUST include `Co-authored-by: Karen Hsieh <ijac.wei@gmail.com>` (email extracted from the PR commit metadata).
- No pre-merge additive commit is required. T1-T6 test additions and the conflict-message-strengthening refinement can land as a follow-up PR we author after Karen's PR merges, keeping her scope clean.

### 10. Acceptance criteria for the implementation stage — DONE

| AC | Verification |
|---|---|
| AC-1: PR #118 merges to main with Karen's authorship preserved on every commit (or squash-with-Co-authored-by trailer). | `git log --format='%an %ae' origin/main -5` shows `Karen Hsieh <ijac.wei@gmail.com>` on the merged commits or trailer. |
| AC-2: `runtime-live` environment approved; `claude-live`, `claude-live-bare`, `claude-live-opus` re-run and pass. | `gh pr checks 118` shows green on the post-approval re-run of run 24555916111 (or a newer run). |
| AC-3: Full offline test suite green on the merge commit. | `static-offline` CI job green; `uv run pytest` locally = 444+ passed, 0 failed. |
| AC-4: Entity body updated post-merge with merged-commit SHA in cross-references and status=validation → completed. | `grep` entity file for merge SHA after `gh pr merge 118`. |
| AC-5 (follow-up, non-blocking): Refinement list item 1 (conflict-message strengthening) and item 2 (T1-T6 tests) land as a separate PR within one session of the merge. | That PR's test count increases by ≥6 and the conflict warning contains "Remove" guidance. |

### 11. Test plan — DONE
- **Static (CI):** `static-offline` job runs the full offline suite on every push. Current state: green on run 24555916114. Cost: ~36s wallclock, negligible cost.
- **Live (CI, gated):** one re-run of `runtime-live-e2e` after environment approval. Cost: ~5-15 min wallclock, ~$1-3 of API spend across claude-live, claude-live-bare, claude-live-opus, codex-live. Runs only after CL approves the environment.
- **Local smoke (optional):** `uv run pytest tests/test_status_script.py -k TestEntityAsFolder -v` — ~3-5s wallclock, zero API cost. Useful if we want to re-verify on main post-merge.
- **E2E beyond CI:** not required. The new `TestEntityAsFolder` class is fully unit-style; no live-agent E2E is needed to validate folder discovery/set/archive semantics. Live-CI passing is sufficient to cover integration with the rest of the commission scaffolding.

### 12. Commit the updated body on main — DONE
Committed via the final commit of this ideation session.

### 13. Append Stage Report (ideation) section — DONE
This section.

### Summary
PR #118 is scope-disciplined, directly satisfies issue #99, and adds correct plumbing for the worktree + reserved-subdir + conflict edges that the issue didn't enumerate. No must-fix blockers. Recommended merge path: merge-commit preserving Karen's authorship, after CL approves the `runtime-live` environment so claude-live can actually run (the current red status is pre-approval secret-gating, not a regression). Design recommendation pinned: auto-detection (option a) as shipped, with a small conflict-message strengthening and six nice-to-have test additions to land as a follow-up PR we author. Explicit workflow-level `entity-style` declaration (option b) is NOT recommended — unnecessary scope expansion.
