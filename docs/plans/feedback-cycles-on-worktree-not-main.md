---
id: k9s1zdwzfmdzdayjdvjjpj6f
title: "FO-owned `### Feedback Cycles` section should live on the worktree branch, not main"
status: backlog
source: "Captain observation 2026-04-30 during PR #176 + PR #177 rebase: feedback cycles entries on main collide with worktree-branch stage reports, producing painful merge conflicts in the entity body when the PR lands"
started:
completed:
verdict:
score: 0.65
worktree:
issue:
pr:
mod-block:
---

The FO Write Scope (shared-core) currently gives the first officer write rights to the `### Feedback Cycles` section in entity bodies on main:

> **`### Feedback Cycles` section** — in entity bodies, tracking rejection rounds

In practice, during worktree-backed stages (implementation / validation), the worktree branch appends stage reports to the same entity file. The FO appends Feedback Cycles entries to the same file on main when a gate is rejected. Both regions live in the bottom of the entity body, sequentially. When the PR lands, `git merge` or `git rebase` has to hand-resolve the overlap because each side advanced different lines in the same trailing region.

PR #176 and PR #177 (2026-04-30) both went DIRTY/CONFLICTING for this exact reason. The captain's `/clear` session had to manually resolve the merge in the worktree because the cycle entries on main and the cycle-2 stage reports on the branch were adjacent and the merge driver couldn't tell which order they should interleave. The work is mechanical but slow and error-prone — it costs a captain turn every PR merge, and it scales linearly with the number of feedback cycles a PR went through.

## Sketch of the fix

Move FO writes of `### Feedback Cycles` out of main and into the worktree branch during worktree-backed stages. Concrete rules to land:

1. When a gate is rejected on a stage with `feedback-to`, the FO writes the cycle entry to the entity body **in the worktree** (not on main). The FO already has worktree access for git operations — entity-body writes via `Write`/`Edit` against the worktree path are mechanically the same as `status --set` against frontmatter, just with a different file scope.
2. The cycle entry rides along with the worker's stage reports through the next merge — no separate main-side commit.
3. Update the FO Write Scope clause in `skills/first-officer/references/first-officer-shared-core.md` to reflect the new location: Feedback Cycles is **branch-owned during worktree-backed stages**, **main-owned only when no worktree is active** (e.g., feedback during ideation).
4. Pre-existing entities mid-flight need no migration — the rule applies to feedback cycles written after the change ships.

## Relationship to #165

Task 165 (entity-state-sidecar-file) targets the same broader pain — state churn on main during worktree-backed stages — but at the frontmatter layer (`status`, `worktree`, `pr`, `mod-block`, `verdict`). Option (b) of #165 ("State transitions live in the worktree during worktree-backed stages") would incidentally pull Feedback Cycles into the worktree if implemented broadly, but #165's current scope is explicit about frontmatter. This task is the body-section sibling: narrower, cheaper to ship, doesn't require the discovery-side rework #165 would force on `status --boot`.

If the captain prefers, this task can be folded into #165 as a sub-scope; otherwise it stands alone as a smaller, independently-mergeable surface.

## Acceptance criteria

**AC-1 — FO writes Feedback Cycles entries to the worktree entity file during worktree-backed stages.**
Verified by: a live test that triggers a gate rejection during validation, then asserts the cycle entry appears in the worktree-side entity body and is absent from the main-side entity body until the PR merges.

**AC-2 — Two consecutive feedback cycles on the same entity merge cleanly into main without rebase conflicts in the entity body.**
Verified by: repro PR #176-style scenario in a fixture (entity has cycle-1 + cycle-2 reports, FO appends a cycle marker between them); `git merge origin/main` from the branch produces no conflicts in the entity file.

**AC-3 — FO Write Scope documentation reflects the new ownership rule.**
Verified by: grep `skills/first-officer/references/first-officer-shared-core.md` for "Feedback Cycles" returns wording that names the worktree branch as the owner during worktree-backed stages.

**AC-4 — Feedback during non-worktree-backed stages (e.g., ideation) still writes on main.**
Verified by: a parallel live test that triggers a rejection at an ideation gate and asserts the cycle entry lands on main directly (no worktree to write to).
