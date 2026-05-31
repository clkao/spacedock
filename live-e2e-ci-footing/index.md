---
id: b6tef0q53k5v9d3vsga49sz4
title: Wire an approval-gated live-runtime e2e CI job (CI-E2E env) — footing for the deferred live net
status: backlog
source: FO/captain (2026-05-31) — behavior-coverage sprint follow-on; the CI-E2E* approval-gated environments exist but no v1 workflow references them
score: "0.30"
started:
completed:
verdict:
worktree:
issue:
---

The repo `spacedock-dev/spacedock` already has three GitHub **Environments with required-reviewer
approval gates** (reviewer = clkao): `CI-E2E`, `CI-E2E-CODEX`, `CI-E2E-OPUS`, plus repo secrets
`ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `HOMEBREW_TAP_TOKEN`. **No v1 workflow references them**
(`grep 'environment:' .github/` is empty) — the gates are provisioned but the door was never hung.
The two existing v1 workflows (`release.yml`, `next-publish.yml`) are auto-triggered with repo-level
secrets and no approval gate.

These environments are the landing pad for the **deferred live-runtime net** the coverage matrix
(archived `behavior-test-skeleton-and-matrix`, id `8033qbqdrh4zba10w0d34m4j`) parked as "CI when we
get there" — the live half of row 15 (does a *real* FO honor reject→reflow→keep-alive), and rows
16/17 (team fail-early live, codex packaged-agent). The behavior-coverage pair just shipped covers
the *deterministic* halves; this entity starts closing the *live* half.

## Reference

`~/git/spacedock/.github/workflows/runtime-live-e2e.yml` (the Python net, 25KB) is the proven
template: `workflow_dispatch` trigger with `model_override`/effort inputs; a `static-offline` job
plus live jobs `claude-live` / `claude-live-bare` / `claude-live-opus` (each `environment: CI-E2E`
or `CI-E2E-OPUS`, secret `ANTHROPIC_API_KEY`) and `codex-live` (`environment: CI-E2E-CODEX`, secret
`OPENAI_API_KEY`); each runs `uv run pytest --runtime claude|codex …`. v1 is Go, so the open design
question ideation must resolve is **what the v1 live test actually runs** (a `//go:build live`-tagged
Go test that shells a real `spacedock claude`/`codex` dispatch and asserts mechanical outputs? a
shell smoke driving the binary end-to-end? reuse of the Python harness against the v1 binary?).

## Scope — mechanism-first (smallest end-to-end proof FIRST)

Per "validate the smallest end-to-end exercise of the riskiest path first": this entity wires **ONE**
approval-gated live job (`claude`, `environment: CI-E2E`) that runs the **smallest meaningful live
dispatch→ensign→stage cycle** and asserts a real mechanical output — proving the whole mechanism
(env gate + API-key env secret + live runtime + a non-mock behavioral assertion + artifact). The full
multi-tier matrix (codex/opus/bare, porting all Python live tests) is the **extension roadmap**, not
this entity.

## Acceptance criteria (provisional — ideation hardens)

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 — A `workflow_dispatch` v1 workflow defines a live job gated on `environment: CI-E2E` that
draws `ANTHROPIC_API_KEY` and pauses for required-reviewer approval before the live step runs.**
Verified by: the workflow YAML declares `environment: CI-E2E` on the live job and references
`secrets.ANTHROPIC_API_KEY`; a real captain-triggered run shows the job parked at the approval gate
until approved (captain-run — sandbox cannot trigger Actions or spend API budget).

**AC-2 — The live job exercises a real `spacedock` dispatch→completion cycle and asserts a mechanical
output (NOT a mock / NOT prose-grep).** Verified by: the job's test entrypoint drives a real runtime
(`spacedock claude`/`codex` or equivalent) and asserts an observable mechanical output (e.g. a stage
report shape / completion-signal / state commit), failing if that output is broken — the live
counterpart of the `internal/ensigncycle` skeleton.

**AC-3 — The offline/static portion stays runnable without secrets and the live job is the only
gated/cost-bearing surface.** Verified by: an offline job (lint/build/static) runs with no environment
and no API key; only the live job carries `environment:` + the API-key secret.

## Out of scope (extension roadmap, not this entity)
- The full 4-tier matrix: `codex-live` (CI-E2E-CODEX), `claude-live-opus` (CI-E2E-OPUS), bare-team tier.
- Porting all Python live tests (test_gate_guardrail, test_rejection_flow, test_feedback_keepalive, …).
- Notarization / release-lane concerns (separate from the live-e2e net).

## Notes — higher-stakes; FO will not silently auto-approve
This entity writes a real CI workflow that spends API budget and uses approval-gated production
secrets. The implementation deliverable is `.github/workflows/*.yml` (+ a live-test entrypoint), and
validation requires a **captain-triggered** gated run (the sandbox cannot trigger Actions, approve a
gate, or spend API budget). The FO will bring the ideation design to the captain rather than
auto-approving, and route the live verification to the captain.
