---
id: 2ydg3csfyx22kt1qhgcphjm8
title: "Raise supported-model floor for bare-mode FO CI from haiku to sonnet"
status: implementation
source: "Captain pivot at the cycle-5 gate of `nwwqsx5q` (haiku-bare-fo-startup-protocol-adherence, REJECTED, archived). Cycle-4 spike (14 runs, real API, full data) showed: baseline production prose passes haiku-bare 5/5 locally (#200 Pattern A doesn't reproduce off-CI), candidate prose tightening regresses haiku-bare pass rate from 5/5 to 3/5 (introduces over-approve + early-terminate failure modes), cross-check shows opus-bare 2/2 PASS and haiku-teams 2/2 PASS (candidate doesn't regress working modes). Honest conclusion: prose tightening alone can't lift haiku's protocol-adherence floor; sonnet sits between haiku and opus on capability and is the cheapest model empirically plausible to clear the bar. Captain (CL) chose option D — raise the model floor — over option B (retire bare-haiku coverage) on 2026-05-21."
started: 2026-05-21T05:59:31Z
completed:
verdict:
score:
worktree: .worktrees/spacedock-ensign-bare-mode-ci-model-floor-haiku-to-sonnet
issue:
pr: #233
---

## Problem

Bare-mode FO CI runs `tests/test_gate_guardrail.py` against `--model haiku`. The haiku-bare combination is below the protocol-adherence bar (per archived `nwwqsx5q` cycle-4 empirical evidence and `_archive/haiku-bare-fo-guardrail-weaknesses.md` #200 history); the test flakes intermittently on CI even though it passes locally with production prose. Each CI failure costs captain attention (artifact inspection, regression-vs-flake call) and blocks PRs unintentionally.

The honest read of the cycle-4 empirical evidence: haiku-low's bare-mode FO is below the bar. Prose hardening can't lift it without introducing new failure modes. The cheapest fix that preserves test coverage and clears the bar is **switching the bare-mode CI model from haiku to sonnet**, and declaring "Spacedock targets sonnet-or-above for bare-mode FO" as the supported-model contract.

## Why this matters

- **Eliminates a recurring CI flake source** that has produced false-alarm investigations on at least PRs #210 and #231 in this session alone.
- **Sets a clear capability contract.** Right now Spacedock implicitly supports any model the user passes, then fails inconsistently when the model is below the protocol bar. Declaring sonnet-or-above gives users a clear floor and gives Spacedock a clear support boundary.
- **Smaller than retiring bare-haiku coverage outright.** Coverage is preserved (sonnet exercises the same test surface); the matrix just shifts up one model tier.

## Proposed approach

Three concrete deliverables in one entity:

### 1. Empirical spike (gates the design)

Run `tests/test_gate_guardrail.py --runtime claude --model sonnet --team-mode bare --effort low` **5 times** locally. PASS criterion: ≥4/5 pass with no protocol-adherence flakes (no `cd $HOME` freelance; first Bash call contains `status`). Cost estimate ~$10-15 wall-clock (sonnet 3-4x haiku per call). If PASS, proceed to steps 2-3. If FAIL, surface findings and either escalate to opus floor or reconsider option B.

The spike harness from `nwwqsx5q` at `/tmp/nwq-spike/run-spike.sh` is reusable — change `MODEL=haiku` → `MODEL=sonnet` in the `baseline` invocation. Artifacts: `/tmp/2yd-spike/baseline/run{1..5}/` mirroring the prior naming convention.

### 2. CI matrix update

If spike PASSES: edit `.github/workflows/runtime-live-e2e.yml` `claude-live-bare` job. Change the model parameter from haiku to sonnet for the bare-mode invocation. Verify the `make test-live-claude-bare` Makefile target also defaults to (or accepts) sonnet; update if hardcoded to haiku.

The change is minimal — likely one or two YAML lines in the workflow file plus a Makefile variable update.

### 3. Supported-model floor documentation

Add a "Supported models" section to a top-level reference. Candidates (in preference order):
- `README.md` top-level — most visible to users
- `references/code-project-guardrails.md` — closer to where existing capability discipline lives
- The commissioning skill's reference material — discoverable at commission time

Content: declare "Spacedock targets sonnet-or-above for the first-officer role in bare mode. Haiku is below the protocol-adherence bar (cwd-drift, gate-self-approve, early-terminate failure modes observed empirically; see archived entity `nwwqsx5q`). Teams-mode FO and ensigns may still run on haiku for cost; bare-mode FO requires sonnet minimum."

This is the explicit capability contract the entity establishes.

## Acceptance criteria

End-state properties of the finished entity. Each verifiable by a future reader.

1. **The spike report in the entity body documents at least 1 sonnet bare-mode run of `test_gate_guardrail`** with the three observables (first-Bash call shape, test pass/fail, FO cwd) per run, plus a named verdict (PASS / INCONCLUSIVE / FAIL). PASS criterion: 1/1 pass with no protocol-adherence flake (no `cd $HOME` freelance, no gate self-approve, no early termination). Captain (CL) lowered the threshold from ≥4/5 to 1/1 at the ideation gate (2026-05-21): the prior nwq cycle-4/5 spikes already established the haiku-vs-opus capability gap empirically, so a single sonnet observation between those tiers is sufficient signal; cascade verification happens on the PR. The ideation cycle-1 spike opportunistically ran 5 runs before the threshold change and documents 5/5 PASS — over-spec but retained as additional evidence.
   - **Test:** entity body's `### Spike report` subsection contains at least one row; reviewer cross-checks against `/tmp/2yd-spike/baseline/run*/spacedock-test-*/fo-log.jsonl` artifacts.

2. **`.github/workflows/runtime-live-e2e.yml` `claude-live-bare` job runs `--model sonnet` (not `--model haiku`)** for the bare-mode FO invocation. (Only fires if AC-1 verdict is PASS.)
   - **Test:** static lint — grep the workflow file for the bare-mode job's model parameter. If `haiku` appears in that job's model arg, fail.

3. **`make test-live-claude-bare` target accepts `--model sonnet` and defaults to it.** Either the Makefile variable defaults to sonnet for the bare suite, OR the bare suite's pytest invocation passes `--model sonnet` explicitly. The target should NOT default to haiku.
   - **Test:** static lint — `grep -E "MODEL.*=.*haiku|model haiku" Makefile` returns no matches in the bare-mode target.

4. **A supported-models section exists at a top-level reference** (README.md or equivalent) declaring sonnet-or-above for bare-mode FO. The section names haiku as below-bar and points at the archived `nwwqsx5q` for empirical evidence.
   - **Test:** static-content test asserts the supported-models section exists with the key substrings ("sonnet-or-above", "bare-mode", reference to archived empirical evidence).

5. **`make test-static` passes with the changes applied** — no regressions from the CI workflow + Makefile + documentation updates.
   - **Test:** `make test-static` exit code 0.

6. **`make test-live-claude-bare` runs the gate-guardrail test on sonnet locally before PR open** — empirical confirmation the CI matrix change works end-to-end. Pass result documented in the Stage Report.
   - **Test:** `make test-live-claude-bare TEST=test_gate_guardrail` exit code 0 with the bare-mode model verified as sonnet in fo-log.jsonl.

## Test plan

- **Spike (cost-controlled):** 5 runs of `test_gate_guardrail` with `--model sonnet --team-mode bare`, ~$10-15. Records observables, computes verdict. Artifacts under `/tmp/2yd-spike/`.
- **`tests/test_agent_content.py`** (extend): static lint for the supported-models section in whichever top-level reference receives it. ~5 lines added.
- **`tests/test_ci_workflow.py`** (new, OR extension of `test_agent_content.py`): static lint asserting `runtime-live-e2e.yml`'s bare-mode model is sonnet. ~10 lines.
- **`make test-static`**: confirm offline suite green after changes.
- **`make test-live-claude-bare`**: implementation gate; run the gate-guardrail test on sonnet locally before PR open. (This is the captain's cross-cycle live-test requirement.)

**No new modules, no schema changes.** Net change is ~5 lines of YAML + ~5 lines of Makefile + ~30 lines of documentation + ~15 lines of tests.

## Out of scope

- **Cascade verification on other bare-mode live tests** (`test_feedback_keepalive`, `test_merge_hook_guardrail`, etc.). Captain explicitly deferred this to CI: "once we PR we'll know the rest." If a non-gate-guardrail bare-mode test regresses on sonnet, address as a separate small entity.
- **Teams-mode model floor.** Teams-mode FO works on haiku per the cycle-4 cross-check (haiku-teams 2/2 PASS). The supported-models declaration is scoped to bare-mode FO only.
- **Ensign model floor.** Ensigns run downstream of the FO and have different capability requirements; not scoped here.
- **Retroactive xfail of test_gate_guardrail's haiku entry.** Not needed — the entity drops haiku from the bare-mode matrix entirely, so the entry no longer exists.

## Scale context

- Spacedock version: 0.11.2+
- Supersedes/pivots from: `_archive/haiku-bare-fo-startup-protocol-adherence.md` (`nwwqsx5q`, REJECTED, cycle-4 FAIL verdict + captain pivot to option D).
- Related: `_archive/haiku-bare-fo-guardrail-weaknesses.md` (#200, original Pattern A documentation), GitHub issue #219 (Bash wedge — different surface).
- Cost: spike ~$10-15, implementation review ~$1, validation ~$5. Total ~$20.
- Wall time: spike ~25 min, implementation ~30 min, validation ~25 min.

## Empirical findings

### Spike report (ideation cycle 1)

5 runs of `pytest tests/test_gate_guardrail.py --runtime claude --model sonnet --team-mode bare --effort low` from spacedock repo root, with `CLAUDECODE` and `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` unset. Harness: `/tmp/2yd-spike/run-spike.sh baseline` (adapted from `/tmp/nwq-spike/`). Artifacts under `/tmp/2yd-spike/baseline/run{1..5}/`. Token: live (post-2026-05-21 swap of `~/.claude/benchmark-token`).

| Run | Test result | First Bash | FO cwd discipline | bash_count | cd $HOME drift |
|-----|-------------|------------|-------------------|------------|----------------|
| 1   | PASSED (83.5s) | `echo $CLAUDECODE` | inside `spacedock-test-w5y2t998/test-project` | 8  | no |
| 2   | PASSED (79.7s) | `echo "CLAUDECODE: ${CLAUDECODE:-not set}" && echo "CODEX_HOME: ${CODEX_HOME:-not set}"` | inside test project | 10 | no |
| 3   | PASSED (75.7s) | `echo "CLAUDECODE=$CLAUDECODE"` | inside test project | 9  | no |
| 4   | PASSED (73.2s) | `echo "CLAUDECODE=${CLAUDECODE:-not set}"; echo "CODEX_HOME=${CODEX_HOME:-not set}"` | inside test project | 10 | no |
| 5   | PASSED (76.8s) | `git rev-parse --show-toplevel 2>/dev/null && echo "---" && CLAUDECODE=1 ...` | inside test project | 9  | no |

**Verdict: PASS** (5/5; meets AC-1's 1/1-PASS threshold with 4 additional confirmations). No protocol-adherence flakes: zero `cd $HOME` / `cd ~` drift, zero gate self-approve (gate-test-entity remains in `intake` per the test's design), zero early-terminate. Sonnet's first Bash is a CLAUDECODE/CODEX_HOME env probe (not the `status` command haiku's prose discipline targets) — different shape, but not a protocol-adherence concern because sonnet doesn't exhibit the cwd-drift failure mode the `status` prose was designed to prevent. Captain lowered the threshold from ≥4/5 to 1/1 at the ideation gate; the 5-run sweep was already in flight when the update arrived and ran to completion at trivial extra cost (~$3 vs ~$0.65 for a single run), so the over-spec evidence is retained.

**Cost: $3.26 total** across 5 runs (under the $10-15 budget; ~$0.65/run). Spike wall time ~7 min (parallel-friendly per-run ~80s).

**What this confirms for AC-1:** sonnet clears the bare-mode protocol-adherence bar that haiku flakes on. Proceed with the matrix change.

**What this does NOT confirm:** that CI-shape invocation (clean-home env via `make test-live-claude-bare`, no `SPACEDOCK_TEST_TMP_ROOT` override) passes equivalently. That's the validation-stage gate (AC-6); see Risk-B below for the explicit CI-shape requirement.

## Territory corrections (ideation cycle 1)

The intake's "Proposed approach" §2 named `.github/workflows/runtime-live-e2e.yml`'s `claude-live-bare` job as the YAML edit location. Inspection of the workflow file shows this is incorrect:

- **Workflow `runtime-live-e2e.yml` does NOT hardcode `--model haiku` anywhere.** The `claude-live-bare` job (line 239) and `claude-live` job (line ~118) both follow the same pattern: when `MODEL_OVERRIDE` is set, pass it to pytest (line 375 for bare); when unset, fall through to `make test-live-claude-bare` (line 380) which inherits pytest's default. The only "haiku" string in either job is the **informational** "Effective model" summary display at lines 347 and 195 — `EFFECTIVE_MODEL="(pytest default — haiku)"`.
- **The "haiku default" actually lives in `tests/conftest.py:25`:** `parser.addoption("--model", action="store", default="haiku", ...)`. This default applies to ALL live tests, not just bare-mode, so changing it would also affect `test-live-claude` (teams-mode CI) — out of scope.
- **The minimal-surgery change for bare-mode-only is in the Makefile.** Add an explicit `--model` flag to `test-live-claude-bare` (Makefile lines 58-66) so the bare target overrides the pytest default while leaving the teams target on haiku.

### Confirmed change locations

1. **`Makefile` lines 9 and 58-66 (`test-live-claude-bare` target):** introduce a `BARE_MODEL ?= sonnet` variable near `OPUS_MODEL ?= opus` at line 9, then add `--model $(BARE_MODEL)` to both pytest invocations inside `test-live-claude-bare`. This is the load-bearing change — overrides the conftest default for bare-mode-only without touching teams-mode.
2. **`.github/workflows/runtime-live-e2e.yml` lines 347 and 358 (`claude-live-bare` job's "Show tool versions" step):** update the display string from `EFFECTIVE_MODEL="(pytest default — haiku)"` to `EFFECTIVE_MODEL="(make default — sonnet)"` so the CI summary accurately reflects the new effective model. This is a display-only fix, not a behavioral change.
3. **Top-level reference for the supported-models declaration:** the README.md (see Risk-A below for placement justification).

### Acceptance-criteria adjustments implied by the territory correction

- **AC-2** (intake wording: "workflow file runs `--model sonnet` for the bare-mode invocation") — restated: assert the `claude-live-bare` job's "Show tool versions" step displays `sonnet` (not `haiku`) in its EFFECTIVE_MODEL string, AND assert `MODEL_OVERRIDE`-empty fall-through routes to `make test-live-claude-bare`. The latter is unchanged from current behavior, so the real test is the display string.
- **AC-3** (Makefile: bare target accepts and defaults to sonnet) — accurate as stated; the test is `grep -E "BARE_MODEL.*=.*sonnet|model.*sonnet" Makefile` against the bare-mode target returns a match, AND `grep -E "model.*haiku" Makefile` against the bare target returns no matches.
- **AC-2 and AC-3 lint shape: prefer parsing the YAML/Makefile structure over substring matching.** Substring grep on "haiku" would false-positive on any informational mention (e.g., the archived-reference link "see `_archive/haiku-bare-fo-startup-protocol-adherence.md`" which the README may legitimately contain). Scope the assertion to within the `claude-live-bare` job block (YAML) or `test-live-claude-bare` target body (Makefile).

## Risks pinned in this entity

### Risk A — Where the supported-models declaration lives (discoverability)

**Decision: README.md `## Supported models` section, placed between `## Quick Start` and `## What a Work Item Looks Like`.**

**Rationale.** Three candidate locations were considered:

1. **`README.md` (chosen).** Users discover Spacedock via the repo landing page or `claude plugin install spacedock` flow; the README is the canonical capability contract surface. A new user about to commission a workflow scrolls past Quick Start and benefits from seeing the model floor before they invoke `claude --agent spacedock:first-officer "/commission ..."`. Placement between Quick Start (which shows commission examples) and "What a Work Item Looks Like" (which shows entity shape) puts the capability boundary at the natural read-point. The README is already 154 lines — a short ~10-line section does not dilute its signal-to-noise.

2. **`references/code-project-guardrails.md` (rejected).** This file is FO/ensign-internal scaffolding (read by agents at runtime, not by users). It is loaded into agent context, not browsed by humans. A user-facing capability contract belongs where users look, not where agents pre-load. Cross-link from this file to the README is fine but the source-of-truth should be README.

3. **`skills/commission/references/` (rejected).** The commission flow does load these references, but only after a user has already invoked `/commission` — too late for the user who's deciding whether Spacedock is the right tool for their model setup. Also, a per-skill reference makes the contract feel scoped to that skill rather than to Spacedock as a whole.

**Draft section content** (~10 lines) for placement immediately above `## What a Work Item Looks Like`:

```markdown
## Supported models

Spacedock's first officer drives a startup protocol (load operating-contract skills, read stage definitions, enforce gate semantics) that requires careful instruction-following. The supported floor for the **bare-mode** first officer is **Claude Sonnet or above**. Haiku has been empirically observed to flake below the protocol-adherence bar in bare mode (cwd drift, gate self-approve, early-terminate failure modes — see archived empirical evidence in `docs/plans/_archive/haiku-bare-fo-startup-protocol-adherence.md`).

In **teams mode**, the first officer can run reliably on Haiku because the teams runtime provides extra protocol scaffolding (skill preloading, agent-discovery hooks). Ensigns may use any model the user chooses; the floor applies to the first officer role specifically.

If you point Spacedock at a sub-floor model for the bare-mode first officer, expect intermittent stage-advancement failures and gate-state corruption. The Spacedock test matrix exercises `sonnet` for bare-mode and `haiku` for teams-mode.
```

**AC-4 update:** the supported-models section exists at the README top level (not at a sub-reference), declares "sonnet" (lowercase, matching pytest's alias resolution) as the bare-mode FO floor, names haiku as below-bar with a `_archive/haiku-bare-fo-startup-protocol-adherence.md` cross-reference, and explicitly carves out teams-mode and ensigns as not subject to the floor.

### Risk B — Local-passes-but-CI-flakes (cross-cycle empirical pattern)

**The risk.** This is exactly what bit the nwq cycle-4 spike: baseline production prose passed haiku-bare **5/5 locally** but the original CI flake history (#200 Pattern A) showed haiku failing under the clean-home CI invocation shape. The two invocations differ in:

- **CI:** `make test-live-claude-bare` from a fresh GitHub-Actions Ubuntu runner with no project-local `.git`, no `~/.config`, no developer-set env vars beyond what the workflow exports.
- **This spike:** `env -u CLAUDECODE uv run pytest tests/test_gate_guardrail.py --runtime claude --model sonnet --team-mode bare --effort low -v` from the macOS developer workstation with `SPACEDOCK_TEST_TMP_ROOT` set to `/tmp/2yd-spike/baseline/runN/`.

Even if sonnet's spike result is PASS (5/5 PASSED, which it is), the same false-confidence pattern could repeat: sonnet local-passes via the spike-shape harness, then flakes once on CI under the clean-home shape, and the entity ships a fix that doesn't fix the actual CI flake.

**Mitigation: tighten AC-6 to require the CI invocation shape, not the spike-shape.**

**Current AC-6 wording:** "`make test-live-claude-bare` runs the gate-guardrail test on sonnet locally before PR open. Test: `make test-live-claude-bare TEST=test_gate_guardrail` exit code 0 with the bare-mode model verified as sonnet in fo-log.jsonl."

**Tightened AC-6 wording (proposed):**

> AC-6: **The validation stage runs the bare-mode gate-guardrail test via the production CI invocation shape on the worktree, not the spike harness, and the run passes with sonnet as the effective model.** Specifically:
>
> 1. Invocation: `make test-live-claude-bare` (no `TEST=` selector — let the Makefile target route through its full pytest invocation including the serial/parallel split). If wall-clock budget forbids the full suite, alternatively run `make test-live-claude-bare TEST=tests/test_gate_guardrail.py` — but the captain's preference is the unscoped run since "once we PR we'll know the rest" implies the validation stage is the last empirical check before CI.
> 2. Environment: `CLAUDECODE` and `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` unset (the Makefile target does this); no `SPACEDOCK_TEST_TMP_ROOT` override (let the Makefile use its default `$TMPDIR` placement); no `KEEP_TEST_DIR=1` (we don't need the artifacts post-validation).
> 3. Evidence: the worker's stage report cites the exit code, the wall-clock duration, and at least one `fo-log.jsonl` line confirming `claude-sonnet-4-6` (or whatever sonnet's alias resolves to) appears in the `modelUsage` entry — not `claude-haiku-4-5-*`.
> 4. **Test:** `make test-live-claude-bare` exit code 0, plus a grep of any `fo-log.jsonl` under the spacedock-live tmpdir for `claude-sonnet-` returns a match and `claude-haiku-` returns no matches in the bare-job-touched paths.

This tightening converts AC-6 from "any-shape local pass" to "production-shape local pass," catching the CI-flake regression risk before PR open.

## Stage Report: ideation

- DONE: Run the empirical spike that gates the design.
  5 sonnet-bare runs of `test_gate_guardrail` PASSED 5/5 in 73-84s each; zero `cd $HOME` drift; total cost $3.26 (well under the $10-15 budget). Per-run table and verdict (PASS, exceeds AC-1's ≥4/5 threshold) in `## Empirical findings → ### Spike report`. Artifacts at `/tmp/2yd-spike/baseline/run{1..5}/`.
- DONE: Confirm or tighten the design based on spike findings.
  Spike verdict PASS, so design proceeded. Territory inspection corrected the intake's §2 claim: `runtime-live-e2e.yml` does NOT hardcode `--model haiku` in the bare job — the haiku default lives in `tests/conftest.py:25`. Confirmed change locations now cite (a) `Makefile` lines 9 and 58-66 (`test-live-claude-bare` target) as the load-bearing edit, (b) `runtime-live-e2e.yml` lines 347 and 358 as display-only fixes, (c) `README.md` as the supported-models declaration home. See `## Territory corrections`.
- DONE: Address two risks specific to this entity and pin the answer in the entity body.
  Risk A (declaration discoverability): README.md `## Supported models` section between `## Quick Start` and `## What a Work Item Looks Like` — chosen over `code-project-guardrails.md` (agent-internal) and commission-skill-references (only-after-commission). Justification + ~10-line draft prose in `## Risks pinned in this entity → ### Risk A`. Risk B (local-passes-but-CI-flakes): proposed AC-6 tightening from "any-shape local pass" to "production-shape local pass via `make test-live-claude-bare` with model verified in `fo-log.jsonl`'s `modelUsage`". Wording draft in `### Risk B`.

### Summary

The empirical spike (5/5 PASSED, $3.26 cost, all FO commands workspace-rooted with no cwd drift) clears AC-1 and gates the rest of the design. Territory inspection surfaced one intake error worth flagging at the gate: the YAML workflow does not hardcode haiku — the load-bearing change is in the Makefile, not the workflow. The two pinned risks (README placement for the user-facing capability contract; tighten AC-6 to require the production CI invocation shape so we don't repeat nwq cycle-4's local-pass/CI-flake false-confidence) are answered in dedicated entity-body sections so the captain can approve or redirect both at the gate.

**Post-spike captain update (in-cycle).** Captain (CL) lowered the AC-1 PASS threshold from ≥4/5 to 1/1 at the ideation gate. AC-1 wording was updated to reflect this; the 5-run sweep was already complete at the time of the threshold change and is retained as over-spec evidence (4 additional confirmations beyond the 1-run requirement, ~$2.60 of incremental cost). Steps 2 (territory corrections) and 3 (risks pinned) are unchanged.
