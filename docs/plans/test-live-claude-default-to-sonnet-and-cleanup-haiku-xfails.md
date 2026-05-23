---
id: qfaypkqwqfd8fzjb1anvy834
title: "Bump test-live-claude default to sonnet + clean up haiku-attributed xfails"
status: backlog
source: "Captain (CL) question 2026-05-22 after a CI-failure audit on PR #234 (rdt) surfaced eight live tests carrying haiku-attributed xfail markers. PR #233 already raised the bare-mode floor to sonnet (`BARE_MODEL ?= sonnet`); the team-mode CI gate (`test-live-claude`) still defaults to haiku via `tests/conftest.py:25`. The result is a main CI gate that runs a known-unreliable model and manages the noise through xfail debt. This entity aligns team-mode with bare-mode and drops the now-unnecessary xfail markers."
started:
completed:
verdict:
score:
worktree:
---

# Bump test-live-claude default to sonnet + clean up haiku-attributed xfails

## Problem

The main team-mode CI gate (`make test-live-claude`, run by the `claude-live` job) does not pass a `--model` flag to pytest, so it inherits the default from `tests/conftest.py:25` — **haiku**. Eight live tests carry xfail markers explicitly attributed to haiku-specific failures (multi-stage compression, model propagation, pr-merge rebase skip, etc.).

When PR #233 raised the bare-mode floor to sonnet (`BARE_MODEL ?= sonnet` in the Makefile), the team-mode default was not updated in lockstep. The current state is: the main CI gate runs the model we know is unreliable for FO work and papers over the resulting drift with xfail markers, while the bare-mode and opus matrices run stronger models that mostly pass the same tests.

A cross-matrix audit of PR #234's CI run confirms:

| Test | bare (sonnet) | team-default (haiku) | opus |
|---|---|---|---|
| test_commission | XPASS | XPASS | XPASS |
| test_rebase_branch_before_push | (serial-skipped here) | XFAIL | XPASS |
| test_repo_edit_guardrail | XFAIL | XFAIL | XPASS |
| test_dispatch_names | XFAIL | XFAIL | XFAIL |
| test_agent_captain_interaction | XFAIL | XFAIL | XFAIL |
| test_per_stage_model_haiku_propagates | XFAIL | XFAIL | XFAIL |

Sonnet+opus already deliver most of the value haiku is supposed to deliver. Haiku-only failures shouldn't sit on the main CI gate as xfail debt.

## Proposed approach

1. **Bump the team-mode CI default to sonnet.** Update `tests/conftest.py:25` to `default="sonnet"` (and review the `BARE_MODEL ?= sonnet` Makefile pattern to mirror it for the `test-live-claude` target if a per-target override is more idiomatic). Keep the Makefile target name `test-live-claude` unchanged; just the model floor changes.
2. **Drop the now-unnecessary `@pytest.mark.xfail(... reason="pending #N ... haiku ...")` markers** where the test PASSes on sonnet. Audit confirms `test_commission.py:22` XPASSes on all three model matrices today.
3. **Convert single-model-passing markers to model-conditional.** Where a test XPASSes on a stronger model but still XFAILs on others (e.g., `test_rebase_branch_before_push.py:76` — haiku-only failure), convert the unconditional `@pytest.mark.xfail(...)` decorator to the `request.applymarker(pytest.mark.xfail(...))` pattern guarded by `if model == "haiku":`, matching how `test_checklist_e2e.py:255` and `test_fetch_on_demand_dispatch.py:56` already handle it.
4. **Optional follow-up: spawn a dedicated `test-live-claude-haiku` matrix** for those who want to track the platform-bug surface explicitly. Out of scope for v1 unless the captain requests it.

## Acceptance criteria

End-state properties of the finished entity. Each AC is testable inside this entity's own deliverables.

1. **`test-live-claude` uses sonnet by default.** `tests/conftest.py:25`'s `--model` default is `"sonnet"`; the Makefile target `test-live-claude` invokes pytest without a `--model` override; the resulting pytest run reports `--model sonnet` in its config dump.
   - **Test:** static check on `tests/conftest.py:25` for the literal `default="sonnet"`. Plus a parser-level test that runs `pytest --collect-only -m live_claude -p no:randomly --runtime claude` and asserts the displayed config option matches.

2. **`test_commission` xfail marker is removed.** `tests/test_commission.py:22` no longer carries `@pytest.mark.xfail(strict=False, reason="pending #197 ...")`. The test is expected to run and pass under sonnet.
   - **Test:** static check that the file does not contain `pytest.mark.xfail` AT the test function (other decorators are fine). The test passes when run under the new default model.

3. **Haiku-conditional xfail markers use the `request.applymarker(...)` pattern.** Tests that currently carry an unconditional `@pytest.mark.xfail(... haiku ...)` decorator but pass on sonnet/opus are converted to inject the xfail only when `model == "haiku"`. At minimum, `test_rebase_branch_before_push.py:76` and `test_dispatch_names.py:27` are converted (audit may identify others — list them in the Stage Report at implementation).
   - **Test:** static check — each converted test's body contains `if model == "haiku":` followed by `request.applymarker(pytest.mark.xfail(...))`; the file no longer carries the haiku reason in a top-level decorator.

4. **`make test-live-claude` passes on sonnet without the dropped markers.** All `live_claude` tests that were previously skipped/xfailed under haiku now run and either pass or are conditionally xfailed only when `--model haiku` is passed explicitly.
   - **Test:** CI green on the `claude-live` job after the change lands. Specifically: zero XPASS reports (an XPASS under sonnet means we missed a conversion). XFAIL count drops to the model-conditional cases.

5. **Audit summary committed in the Stage Report.** Implementation lists every xfail marker touched, its before/after disposition (removed / converted-to-conditional / left-alone), and the rationale per item. No silent changes.
   - **Test:** Stage Report contains a "Marker disposition table" section enumerating every test file inspected and its outcome.

## Test plan

- **Static checks:** AC-1 (conftest default), AC-2 (test_commission marker absent), AC-3 (conditional pattern present in converted files).
- **Live-claude CI:** AC-4 (the `claude-live` job goes green on sonnet without new XPASS).
- **No new live test required.** This entity reshapes existing test infrastructure; behavior is verified by re-running the existing suite under the new defaults.

## Out of scope

- **A dedicated `test-live-claude-haiku` matrix.** Out of scope for v1; can be added later if captain wants explicit haiku-platform-bug tracking.
- **Investigating and fixing the underlying haiku-FO behaviors** (multi-stage compression, model propagation, etc.). Those remain tracked under their respective `pending #N` issues. This entity only adjusts test discipline.
- **Touching `OPUS_MODEL` or `BARE_MODEL` Makefile vars.** Those are correct as-is; only the team-mode default changes.

## Risks

### Risk A — sonnet cost increase

Switching from haiku to sonnet for the main CI gate increases per-run cost. Mitigation: the bare-mode matrix already pays this cost; the team-mode bump is a known incremental, not a surprise. If the cost becomes painful, the captain can either reduce CI frequency or carve out a periodic-only haiku check.

### Risk B — sonnet uncovers new test failures that haiku-noise was hiding

The current xfail debt may be papering over failures that ALSO occur on sonnet but were never observed because the test xfailed first. Mitigation: AC-4's "zero XPASS" check catches the inverse case; for genuine new failures, file individual fix entities rather than re-adding xfail debt.

### Risk C — conditional-marker drift

Converting unconditional markers to conditional `request.applymarker` requires the test function to accept a `model` fixture. Some tests may not currently take that fixture parameter; the implementer needs to add it. Cost: ~5-10 minutes per file. Mitigation: AC-3's static check verifies the pattern landed correctly.

## Scale context

- Spacedock version: 0.12.0+
- Builds on: PR #233 (sonnet floor for bare-mode FO)
- Composes with: ongoing haiku-platform-bug tracking under #158, #160, #171, #196, #197, #198 — none of these issues close as a result of this entity, but their xfail markers move out of the team-mode CI noise.
- Estimated complexity: small. ~15-20 line conftest change, ~3-5 tests touched for marker conversion, +1 to drop test_commission marker. ~$5-10 in agent budget.
- No live-claude E2E required beyond the existing CI green check.
