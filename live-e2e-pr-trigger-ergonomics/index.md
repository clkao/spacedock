---
id: q12h3wny9k4ceq5xq4rnzn1t
title: live-e2e PR trigger ergonomics — drop the label gate, rely on the environment-approval gate (or auto-label)
status: ideation
source: CL (2026-05-31) — "labeling pr requiring e2e live on creation, otherwise they don't trigger… or mimic old main's ci-setting where they're always pending and we explicitly approve env"
started: 2026-06-01T05:04:22Z
completed:
verdict:
score: "0.30"
worktree:
issue:
---

Make the live-runtime e2e run on PRs without a manual per-PR step. Today `runtime-live-e2e.yml`
(on `next`) gates the live job with `if: workflow_dispatch OR PR-carries-'run-live-e2e'-label`, **on
top of** the CI-E2E* environment required-reviewer approval. So a PR that isn't labeled at creation
never queues the live job at all — the operator has to remember to add the label, or manually
dispatch. Two stacked gates (label + environment) do one job's worth of work.

## The two paths CL named

1. **Mimic old main: always-pending + explicit env approval (recommended).** Drop the `run-live-e2e`
   label condition from the live job's `if`. Every PR to `next` then queues the live job straight
   into `waiting` on its CI-E2E* environment; the maintainer approves the deployment for the PRs
   that should burn budget and ignores/rejects the rest. The environment required-reviewer gate
   already provides the budget safety the label was standing in for, so nothing runs a paid model
   unapproved. The secret-free `offline` job stays the always-run gate. This is exactly the
   "always pending, we explicitly approve env" model.
2. **Auto-label on PR creation (lower blast radius).** Keep the label gate but have the `pr-merge`
   mod apply the `run-live-e2e` label when it opens the PR (`gh pr create --label run-live-e2e`),
   removing the manual step while preserving the current architecture. Falls back to documenting the
   labeling requirement for hand-opened PRs.

## Acceptance criteria

**AC-1 — A PR to `next` queues the live job without any manual labeling step.**
Verified by: open a PR (or simulate the `if` evaluation) and confirm the live job reaches `waiting`
on its environment gate with no label applied — under path 1, by asserting the workflow `if` no
longer references the label; under path 2, by asserting the mod applied the label at PR-open.

**AC-2 — No paid model runs without an explicit approval.**
Verified by: the live job still declares `environment: CI-E2E*` and stays in `waiting` until a
required reviewer approves; assert the environment gate is intact after the change.

**AC-3 — The secret-free offline gate still runs on every PR.**
Verified by: the `offline` job (no environment, no secret) remains unconditional and is the PR's
real pass/fail gate.

## Test plan

Primarily a workflow-YAML change validated by reading the rendered `if`/`environment` conditions and
a dry PR (or `act`/`gh workflow view`); plus, for path 2, a `pr-merge` mod behavioral assertion that
`--label run-live-e2e` is passed. Ideation should pick path 1 vs 2 (FO lean: path 1 — fewer moving
parts, matches the model CL described, env gate already enforces budget safety), and confirm the
fork-PR security model (`pull_request` not `pull_request_target`; same-repo-or-no-secrets) is
unchanged by dropping the label condition.
