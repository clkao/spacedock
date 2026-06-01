---
name: pr-merge
description: Open a code-branch PR to next at the merge boundary and track it to merge, state-root-aware
version: 0.12.1
---

# PR Merge

Manages the PR lifecycle for workflow entities processed in worktree stages of this **split-root** workflow. The CODE for an entity lives on its worktree branch in the main repo (`origin`, base branch `next`); the entity STATE (frontmatter, `pr:`, `mod-block:`, stage reports) lives in the separate `.spacedock-state` checkout (`origin`, branch `spacedock-state/dev`). This hook opens a PR for the code branch at the terminal merge boundary — **before** cleanup deletes the branch — records `pr:` on the entity state, blocks until the PR merges, then lets the FO terminalize and archive.

The two origins stay clean by construction: the code PR carries only the code-branch range (the worktree clone has no `.spacedock-state` paths), and the `pr:`/`mod-block:` writes are `spacedock status --set` against the state checkout, committed path-scoped there. **This hook MUST NOT touch `.spacedock-state` from the code worktree** — all state writes go through `spacedock status --set --workflow-dir docs/dev`, which targets the resolved state checkout.

## Hook: startup

Scan all entity files (in the workflow directory only, not `_archive/`) for entities with a non-empty `pr` field and a non-terminal status. For each, extract the PR number (strip any `#`, `owner/repo#` prefix) and check: `gh pr view {number} --json state --jq '.state'`.

If `MERGED`, advance the entity to its terminal stage. Because a `mod-block` is set while the PR is pending, the clear and the terminalization are two separate `--set` calls (the mechanism refuses combining `mod-block=` with terminal fields):
1. `spacedock status --workflow-dir docs/dev --set {slug} mod-block=` (commit: `mod-block: {slug} cleared (pr-merge completed)`),
2. `spacedock status --workflow-dir docs/dev --set {slug} completed verdict=PASSED worktree=`, then `spacedock status --workflow-dir docs/dev --archive {slug}`.
Both `--set`s and the archive are committed path-scoped to the state checkout. Remove the worktree (`git worktree remove {path}`) and delete the **local** branch (`git branch -d {branch}`) — the remote branch was already cleaned by the PR merge. Report each auto-advanced entity to the captain.

If `CLOSED` (closed without merge), report to the captain: "{entity title} has PR {pr number} which was closed without merging. How to proceed? Options: reopen the PR, create a new PR from the same branch, or clear `pr` and fall back to the local `--no-ff` merge." Wait for the captain's direction before taking action.

If `OPEN`, no action needed — the PR is still in review.

If `gh` is not available, warn the captain and skip PR state checks.

## Hook: idle

Check PR-pending entities using the same logic as the startup hook: scan entity files for non-empty `pr` and non-terminal status, run `gh pr view` for each, and advance merged PRs (two-step `mod-block=` clear then terminalize, path-scoped). This provides a periodic re-check in case the event loop's built-in PR scan missed a state change (defense in depth). Report any advanced entities to the captain.

## Hook: merge

Runs at the terminal merge boundary, before any local merge or cleanup, for the entity's CODE worktree branch `{branch}` (the branch named in the entity's `worktree:` field, located at `{worktree}`).

**PR APPROVAL GUARDRAIL — Do NOT push or create a PR without explicit captain approval.** Opening a PR and pushing the branch are outward-facing. Before presenting the draft, construct the full PR body so the captain reviews the actual prose that will land on GitHub.

Compute the audit-link inputs first: state SHA via `git -C docs/dev/.spacedock-state rev-parse HEAD` (the full SHA of the **state** checkout's HEAD — the entity's `mod-block=merge:pr-merge` commit, which the FO committed before invoking this hook, so HEAD already contains the active entity file); owner/repo via `gh repo view --json nameWithOwner --jq '.nameWithOwner'`; short entity-id slot via `spacedock status --workflow-dir docs/dev --short-id {slug}` (shortest-unique-prefix for sd-b32 workflows, matching the status table's ID column).

Build the full PR body using the template below — motivation lead, `## What changed`, `## Evidence`, `---` separator, `[{short-id}](...)` audit link, and `Closes {issue}` line if the entity frontmatter `issue` is set. This is the body that will be passed to `gh pr create` verbatim; do not reconstruct it after approval. The entity body and stage reports are read from the **state checkout** (`docs/dev/.spacedock-state/{slug}/index.md`), not the worktree.

Then present the draft to the captain:

- **Title:** {entity title}
- **Branch:** {branch} -> next
- **Changes:** {N} file(s) changed across {N} commit(s) (`git -C {worktree} diff --stat origin/next...{branch}`)
- **Files:** {list of changed files}
- **Body:**

  ```
  {constructed body}
  ```

Wait for the captain's explicit approval before pushing. Do NOT infer approval from silence, acknowledgment of the summary, or the gate approval that preceded this step — only an explicit "push it", "go ahead", "yes", or equivalent counts.

**On approval:** This is a split-root workflow — the FO rebases the code branch onto `origin/next` BEFORE invoking this hook, and the entity state lives in the separate `.spacedock-state` checkout, so this hook does NOT rebase or push any state branch. Push only the code branch:

1. `git -C {worktree} push -u origin {branch}` — push the entity's code branch to the code remote (`origin` = the main repo, base branch `next`). Do NOT push `next` or any `.spacedock-state` branch from here; the FO coordinates state-remote pushes separately.

If the push fails (no remote, auth error), report to the captain and fall back to the local `--no-ff` merge (see fallback below).

2. Create the PR: `gh pr create --base next --head {branch} --title "{entity title}" --body "{constructed body}"` against the body already constructed above — do not rebuild it. Capture the PR number `{N}`.

3. Record it on the entity STATE: `spacedock status --workflow-dir docs/dev --set {slug} pr=#{N}`. This writes `pr:` into the state-checkout entity frontmatter; the FO commits it path-scoped to `.spacedock-state` (`git -C docs/dev/.spacedock-state add -- {slug}/index.md && git -C docs/dev/.spacedock-state commit -m "pr: {slug} #{N} pending" -- {slug}/index.md`) and pushes `spacedock-state/dev` so the PR-pending state survives session resume and is visible on a 2nd host.

Setting `pr:` is the **blocking** signal (FO Merge-and-Cleanup step 3a). The FO set `mod-block=merge:pr-merge` before invoking this hook; with `pr:` now set the hook has blocked, so the FO leaves `mod-block` set, reports PR-pending to the captain, and does NOT local-merge or archive. The entity stays at its current stage with `pr` set until the PR merges. The FO advances to the terminal stage and archives when it detects the merge (via the event-loop PR check, the idle hook, or the startup hook) — see those hooks for the two-step `mod-block=` clear then terminalize.

### PR body template

Lead with motivation + end-user value; audit metadata goes at the bottom. The goal is that a reviewer or future debugger sees the "why" first and the audit link last.

**Template structure (top to bottom):**

| Section | Required | Content |
|---|---|---|
| Motivation lead | **yes** | 1 sentence, ≤ 25 words, blending motivation and end-user value. No parentheticals. |
| `## What changed` | **yes** | Action-verb bullets, 3–5 total, each ≤ 15 words. One change per bullet. No rationale inside the bullet — if a change needs justification, it belongs in the task body, not the PR. |
| `## Evidence` | **yes when validation ran** | Test suites with `N/N passed` format, 1–2 bullets. Do not include per-test-class breakdowns or enumerated suite lists — one pass ratio per suite, plus at most one line confirming live-probe verification. |
| `## Review guidance` | optional | 1 line pointing reviewer at the critical file or risky change — include only when a stage report explicitly flagged it |
| `---` separator + `[{entity-id}](/{owner}/{repo}/blob/{state-sha}/{state-relative-path})` | **yes** | Audit link, at the bottom. `{state-sha}` is the full SHA of the state checkout's HEAD (see merge-hook step); `{state-relative-path}` is the entity file's active `.spacedock-state`-relative location (`{slug}/index.md`). An immutable state-commit SHA — NOT a branch ref — so the link still resolves after the entity archives. |
| `Closes {issue}` | **yes when issue set** | Under the audit link, using the value exactly as it appears in frontmatter, e.g., `#48` or `owner/repo#48` |
| `Related: {siblings}` | optional | Under Closes, only when stage reports flagged follow-ups |

**Extraction rules (apply deterministically from the entity file):**

| PR body section | Source in entity file | Transformation |
|---|---|---|
| Motivation lead | Entity body paragraph(s) between closing `---` and the first `##` heading | Condense first paragraph to 1-2 sentences. Lead with impact or action verb — not "This PR" or "This task". Blend motivation + value. |
| What changed | Implementation stage report's DONE items | One action-verb bullet per meaningful unit. Collapse sibling bullets that describe the same thing. Do NOT include "what we deliberately did NOT change" bullets — scope boundaries belong in the task body, not the PR, unless a validation stage report flagged them as risk. |
| Evidence | Validation stage report items that assert AC verification (typically rerun-test items) | One bullet per suite with `N/N passed` format. Include any quantitative result the stage report explicitly called out (wallclock delta, size %, perf). Fallback to implementation report's self-test items if no validation stage exists. |
| Review guidance | Explicit "focus on X" / "risk here" notes in either stage report | 1 line. **Omit if no such note exists.** |
| Audit link | Short entity id from `spacedock status --workflow-dir docs/dev --short-id {slug}`, active path from the entity file's `.spacedock-state`-relative location (`{slug}/index.md`), state SHA from `git -C docs/dev/.spacedock-state rev-parse HEAD` (the STATE checkout's HEAD — NOT the code worktree, NOT a branch ref) | Format as `[{short-id}](/{owner}/{repo}/blob/{state-sha}/{state-relative-path})` |
| Closes | Entity frontmatter `issue` field (exactly as written) | Prefix `Closes ` |
| Related | Explicit "related task" / "follow-up" mentions in stage reports | 1 line. **Omit if none.** |

Target total length: **60-120 words**.

### Fallback: no PR host available

If `gh` is not on PATH, or `gh pr create` fails, or the branch push fails, do NOT set `pr:`. Report to the captain that no PR could be opened and fall back to the FO's default local merge (Merge-and-Cleanup step 6): a local `--no-ff` merge of the code branch `{branch}` from the worktree onto `next`, recording the local-merge landing in the entity report. Because no `pr:` was set, the hook completed **without blocking** — the FO clears the `mod-block` it set before invoking (its own standalone `spacedock status --workflow-dir docs/dev --set {slug} mod-block=`, committed path-scoped), then terminalizes and archives at the same merge boundary, just without a PR. (The mechanism-level guard refuses terminalizing while `pr` and `mod-block` are both empty and a merge hook is registered; clearing `mod-block` only after the local merge lands keeps that guard satisfied through the no-PR path.)

**On captain decline (PR host present, captain says no):** Do NOT automatically fall back to local merge. Ask the captain how to proceed — options include the local `--no-ff` merge or leaving the branch unmerged. Only act on the captain's explicit choice.
