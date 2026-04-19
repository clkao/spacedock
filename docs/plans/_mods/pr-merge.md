---
name: pr-merge
description: Push branches and create/track GitHub PRs for workflow entities
version: 0.9.7
---

# PR Merge

Manages the PR lifecycle for workflow entities processed in worktree stages. Pushes branches, creates PRs, detects merged PRs, and advances entities accordingly.

## Hook: startup

**VCS detection**: Before running any PR command, detect VCS provider: `git remote -v | grep -q "github\.com" && echo "vcs=github" || git remote -v | grep -q "gitlab\.com" && echo "vcs=gitlab" || echo "vcs=unknown"`. If github → use `gh` CLI. If gitlab → use `glab` CLI. If unknown → warn captain and skip PR state checks.

Scan all entity files (in the workflow directory only, not `_archive/`) for entities with a non-empty `pr` field and a non-terminal status. For each, extract the PR number (strip any `#`, `owner/repo#` prefix) and check:
- **GitHub**: `gh pr view {number} --json state --jq '.state'`
- **GitLab**: `glab mr view {number} --output json | jq -r '.state'`

If `MERGED` (GitHub) or `merged` (GitLab), advance the entity to its terminal stage: set `status` to the terminal stage, `completed` to ISO 8601 now, `verdict: PASSED`, clear `worktree`, archive the file, and clean up any worktree/branch. Report each auto-advanced entity to the captain.

If `CLOSED` (closed without merge), report to the captain: "{entity title} has PR {pr number} which was closed without merging. How to proceed? Options: reopen the PR, create a new PR from the same branch, or clear `pr` and fall back to local merge." Wait for the captain's direction before taking action.

If `OPEN`, no action needed — the PR is still in review.

If the VCS CLI tool (`gh` for GitHub, `glab` for GitLab) is not available, warn the captain and skip PR state checks.

## Hook: idle

**VCS detection**: Same as startup hook — detect VCS provider before running any PR command.

Check PR-pending entities using the same logic as the startup hook: scan entity files for non-empty `pr` and non-terminal status, run the VCS-detected PR view command for each, and advance merged PRs. This provides a periodic re-check in case the event loop's built-in PR scan missed a state change (defense in depth). Report any advanced entities to the captain.

## Hook: merge

**PR APPROVAL GUARDRAIL — Do NOT push or create a PR without explicit captain approval.** Before pushing, present a draft PR summary to the captain:

- **Title:** {entity title}
- **Branch:** {branch} -> main
- **Changes:** {N} file(s) changed across {N} commit(s)
- **Files:** {list of changed files}

Wait for the captain's explicit approval before pushing. Do NOT infer approval from silence, acknowledgment of the summary, or the gate approval that preceded this step — only an explicit "push it", "go ahead", "yes", or equivalent counts.

**On approval:** First, push main to ensure the remote is up to date with local state commits: `git push origin main`. Then rebase the worktree branch onto main: `git rebase main` (from the worktree directory). Then push the worktree branch: `git push origin {branch}`. If any step fails (no remote, auth error, rebase conflict), report to the captain and fall back to local merge.

Before constructing the PR body, compute the short SHA for the audit link by running `git rev-parse --short HEAD` in the worktree directory. If the command exits non-zero (no commits, detached HEAD), substitute the literal string `main` into the audit-link template instead and report the fallback to the captain. Resolve the owner/repo via:
- **GitHub**: `gh repo view --json nameWithOwner --jq '.nameWithOwner'`
- **GitLab**: `glab repo view --output json | jq -r '.path_with_namespace'`

Create a PR using the VCS-detected command:
- **GitHub**: `gh pr create --base main --head {branch} --title "{entity title}" --body "{constructed body}"`
- **GitLab**: `glab mr create --source-branch {branch} --target-branch main --title "{entity title}" --description "{constructed body}"`

If the VCS CLI tool (`gh` for GitHub, `glab` for GitLab) is not available, warn the captain and fall back to local merge. If `vcs=unknown` → warn the captain and fall back to local merge.

### PR body resolution

**Ship stage PR Draft takes priority.** Before applying the generic template below, check the entity file for a `## Ship Output` → `### PR Draft` section. If present, use its `Body:` content verbatim as the PR body — the ship stage already formatted it with workflow-specific UAT tables, DC verification commands, and reviewer reproduction steps. Only append the audit link (`[{entity-id}](...)`), `Closes`, and `Related` lines if they are not already in the draft.

**Fallback to generic template** only when no `### PR Draft` section exists in the entity file (e.g., workflows without a ship stage, or entities that skipped ship).

### PR body template (fallback)

Lead with motivation + end-user value; audit metadata goes at the bottom. The goal is that a reviewer or future debugger sees the "why" first and the audit link last.

**Template structure (top to bottom):**

| Section | Required | Content |
|---|---|---|
| Motivation lead | **yes** | 1 sentence, ≤ 25 words, blending motivation and end-user value. No parentheticals. |
| `## What changed` | **yes** | Action-verb bullets, 3–5 total, each ≤ 15 words. One change per bullet. No rationale inside the bullet — if a change needs justification, it belongs in the task body, not the PR. |
| `## Evidence` | **yes when validation ran** | Test suites with `N/N passed` format, 1–2 bullets. Do not include per-test-class breakdowns or enumerated suite lists — one pass ratio per suite, plus at most one line confirming live-probe verification. |
| `## Review guidance` | optional | 1 line pointing reviewer at the critical file or risky change — include only when a stage report explicitly flagged it |
| `---` separator + `[{entity-id}](/{owner}/{repo}/blob/{short-sha}/{path-to-entity-file})` | **yes** | Audit link, at the bottom |
| `Closes {issue}` | **yes when issue set** | Under the audit link, using the value exactly as it appears in frontmatter, e.g., `#48` or `owner/repo#48` |
| `Related: {siblings}` | optional | Under Closes, only when stage reports flagged follow-ups |

**Extraction rules (apply deterministically from the entity file):**

| PR body section | Source in entity file | Transformation |
|---|---|---|
| Motivation lead | Entity body paragraph(s) between closing `---` and the first `##` heading | Condense first paragraph to 1-2 sentences. Lead with impact or action verb — not "This PR" or "This task". Blend motivation + value. |
| What changed | Implementation stage report's `[x]` DONE items | One action-verb bullet per meaningful unit. Collapse sibling bullets that describe the same thing. Drop `[x]` markers. Do NOT include "what we deliberately did NOT change" bullets — scope boundaries belong in the task body, not the PR, unless a validation stage report flagged them as risk. |
| Evidence | Validation stage report items that assert AC verification (typically rerun-test items) | One bullet per suite with `N/N passed` format. Include any quantitative result the stage report explicitly called out (wallclock delta, size %, perf). Fallback to implementation report's self-test items if no validation stage exists. |
| Review guidance | Explicit "focus on X" / "risk here" notes in either stage report | 1 line. **Omit if no such note exists.** |
| Audit link | Entity id from frontmatter, path from the file's repo-relative location, short SHA from `git rev-parse --short HEAD` run in the worktree directory | Format as `[{id}](/{owner}/{repo}/blob/{short-sha}/{path})` |
| Closes | Entity frontmatter `issue` field (exactly as written) | Prefix `Closes ` |
| Related | Explicit "related task" / "follow-up" mentions in stage reports | 1 line. **Omit if none.** |

Target total length: **60-120 words**.

**Key design decisions:**

1. **Lead with motivation + end-user value.** First content is a 1-2 sentence user-facing impact statement. The audit link moves to the bottom as audit metadata.
2. **Prescribed sections + extraction rules** — not a strict verbatim template, not free-form. The mod specifies headings and source subsections; the FO paraphrases rather than pasting.
3. **Evidence section is conditional on validation stage.** Non-validated workflows fall back to implementation self-test evidence.
4. **Review guidance and Related are opt-in.** They appear only when stage reports explicitly flagged them, to prevent bloat.

Set the entity's `pr` field to the PR number (e.g., `#57`). Report the PR to the captain.

**On decline:** Do NOT automatically fall back to local merge. Ask the captain how to proceed — options include local merge or leaving the branch unmerged. Only act on the captain's explicit choice.

Do NOT archive yet. The entity stays at its current stage with `pr` set until the PR is merged. The FO handles advancement to the terminal stage and archival when it detects the merge (via the event loop PR check, idle hook, or startup hook).
