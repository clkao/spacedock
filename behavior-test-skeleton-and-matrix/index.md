---
id: 8033qbqdrh4zba10w0d34m4j
title: Behavior-test skeleton + coverage matrix (replace the prose-grep antipattern)
status: backlog
source: sprint — captain (2026-05-31); prereq to cutting the ensign contract
started:
completed:
verdict:
score: "0.36"
worktree:
issue:
---

Establish a behavioral test footing for the ensign/FO/launcher contract so the contract can be cut/trimmed safely. Two deliverables: a **skeleton behavior test** and a **coverage matrix**.

## Why
v1's contract coverage today is `skills/integration/*_test.go` — **STATIC prose-grep** (asserts the contract TEXT contains clauses; passes even if a clause doesn't behaviorally work). That's the antipattern P1/P2 distrust. The Python side has the real behavioral net: a CI workflow `~/git/spacedock/.github/workflows/runtime-live-e2e.yml` plus live pytest (`test_team_fail_early_live.py`, `test_checklist_e2e.py`, `test_codex_packaged_agent_e2e.py`, `test_agent_captain_interaction.py`, …). Before cutting the ensign contract we need (a) a behavioral skeleton in v1 and (b) a map of what's covered where.

## Deliverables / Acceptance criteria (provisional — ideation hardens)

**AC-1 — Coverage MATRIX.** A matrix: rows = the ensign/FO/launcher contract behaviors that SHOULD be tested (dispatch→ensign→stage cycle, stage-report shape, completion signal, gate/feedback flow, checklist accounting, split-root commits, launcher argv, etc.); columns = **python-covers** (which pytest/CI, and whether behavioral or static) × **v1-implements** (which Go test — or "prose-grep antipattern" / "GAP"). Surfaces the prose-grep tests + the real gaps; becomes the port roadmap. Verified by: the matrix exists, every row cites concrete test names/paths on both sides (reproducible, not hand-waved).

**AC-2 — SKELETON behavior test.** One minimal BEHAVIORAL test in v1 that exercises a real dispatch→ensign→stage mechanical cycle (or ports the smallest meaningful Python live behavior) and asserts mechanical outputs (stage-report section shape, the state commit, the completion signal) of a scripted/fixture run — NOT a live LLM agent, NOT prose-grep. A scaffold others extend. Verified by: the test runs in `go test` (or a documented harness command) and FAILS if the asserted mechanical output is broken.

## Out of scope (this entity)
- Full port of all Python live tests (the matrix plans it; this ships the skeleton + map).
- CI setup (the `runtime-live-e2e.yml` analog) — deferred, "when we get there".
- Actually cutting the ensign contract (this is the prereq, not the cut).

## Notes / sequencing
Test-infra surface (new test package/harness) — disjoint from frontdoor.go, so parallel with launch-parity. The matrix is the high-value planning artifact; the skeleton proves the behavioral pattern.
