---
id: "208"
title: "Port test_reuse_dispatch to cycle-7 pattern — streaming watcher + inbox-poll keep-alive + teams-mode-pinned contract"
status: implementation
source: "cycle-7 deferred-scope list (opus-4-7-green-main.md:1079 — Tier-A hygiene port); existing xfail on #154 + #160; cycle-4/cycle-6 full-suite runs show XFAIL on opus (not XPASS)"
started: 2026-04-20T06:47:23Z
completed:
verdict:
score: 0.70
worktree: .worktrees/spacedock-ensign-test-reuse-dispatch-cycle7-port
issue:
pr:
mod-block: merge:pr-merge
---

# Port test_reuse_dispatch to Cycle-7 Pattern Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Un-xfail and green `tests/test_reuse_dispatch.py` at opus-4-7 teams mode by rewriting it on top of the cycle-7 streaming watcher + inbox-poll keep-alive pattern. Haiku-teams stays xfailed with a rationale matching the cycle-7 haiku xfail pattern.

**Architecture:** Rewrite the test to mirror `tests/test_feedback_keepalive.py` (post-cycle-7 shape): pin to `@pytest.mark.teams_mode`, use `run_first_officer_streaming` with `DispatchBudget`, inject the keep-alive sentinel + inbox-poll append-system-prompt, assert the three-signal reuse contract via `expect_dispatch_close` (analysis close + validation close) + one `expect(SendMessage-to-analysis-ensign)` between them (the reuse contract). Handle the validation gate by adding a captain-auto-approval shim to the append-system-prompt, or by relaxing the fixture to drop `gate: true` on validation if the gate is incidental to the reuse contract.

**Tech Stack:** Python, pytest, `scripts/test_lib.py` (`run_first_officer_streaming`, `FOStreamWatcher`, `DispatchBudget`, `expect_dispatch_close`, `expect`), `scripts/fo_inbox_poll.py`, `tests/fixtures/reuse-pipeline/`.

---

## Background

`tests/test_reuse_dispatch.py` verifies that the FO reuses an analysis ensign to handle implementation via `SendMessage` (no second `Agent()` call), then fresh-dispatches `validation` because the fixture sets `fresh: true`. The test is currently xfailed on two overlapping classes:

1. **#154 legacy** — the test's static Phase-1 checks were initially flagged as pre-#085 prose drift; #154 cleared the prose but the runtime shape remains fragile.
2. **#160 haiku compression** — haiku collapses the two-stage "analysis + implementation" handoff into a single dispatch. Sibling to #158 (haiku skips `git rebase main`).

Evidence from `docs/plans/_evidence/green-opus-4-7-full-suite/cycle{4,6}-fullsuite-run1.log`: **opus runs XFAIL too** (not XPASS) — so this is broader than haiku compression. The test's own shape is what's fragile:

- Uses the old `run_first_officer` (not `run_first_officer_streaming`), so no `DispatchBudget` instrumentation, no close-anchor signals.
- Prompt carries five nanny hints (reuse conditions verbatim, gate auto-approval directive) that violate the FO Prompt Discipline added in `tests/README.md` during cycle-7.
- Phase-3 uses `milestones` count-threshold branches with `if len(...) == 0: t.pass_()` / `if len(...) >= 1: t.pass_()` — the exact Tier-A anti-pattern called out in `tests/README.md` (`test_reuse_dispatch` is explicitly named in the Tier-A table).
- No keep-alive sentinel, no inbox-poll Bash probe, so it hits `anthropics/claude-code#26426` under `-p` the same way pre-cycle-7 `test_feedback_keepalive` did.

## Fixture shape (unchanged unless noted)

`tests/fixtures/reuse-pipeline/README.md`:

```yaml
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: analysis
    - name: implementation
    - name: validation
      fresh: true
      feedback-to: implementation
      gate: true
    - name: done
      terminal: true
```

`tests/fixtures/reuse-pipeline/reuse-test-task.md`: single entity, status `backlog`, trivial description "A simple test task for verifying ensign reuse behavior."

**Decision point on validation gate.** Under `claude -p` there is no captain to approve a gated stage; the FO correctly holds at the gate and the test hangs. Two options:

- **(A) Drop `gate: true` from validation in the fixture.** The reuse contract is the analysis→implementation reuse and the implementation→validation fresh-dispatch. The gate at validation is incidental to what the test asserts. Dropping the gate keeps the two relevant contract signals intact.
- **(B) Keep the gate and add a captain-approval shim.** The test harness could write a `gate-approval` sentinel file that the FO polls (similar to the keep-alive sentinel), or the append-system-prompt could authorize the FO to auto-approve when a specific condition holds. More complex and couples the test to a gate-approval contract it's not meant to verify.

**This plan chooses (A).** Dropping `gate: true` on validation in the fixture is a one-character fixture edit with no callers other than this test (verified via `grep -rn "reuse-pipeline" tests/`). Gate behavior has its own dedicated test (`test_gate_guardrail.py`). If we later need to assert gate behavior on the reuse pipeline, we split into a sibling test.

## Expected FO trajectory (teams mode, opus-4-7)

1. `TeamCreate(test-project-reuse-pipeline-...)`
2. `Agent(subagent_type="spacedock:ensign", description="... analysis")` — initial dispatch
3. (ensign runs analysis work → writes Stage Report → emits `SendMessage(to="team-lead", "Done: ...completed analysis...")`)
4. FO polls inbox via Bash → reads the `Done:` message
5. FO decides reuse per shared-core rules (same worktree mode, no `fresh: true` on implementation) → emits `SendMessage(to="spacedock-ensign-reuse-test-task-analysis", message="Advancing to next stage: implementation\n\n### Stage definition:\n\n...")`
6. (ensign receives reuse message at its next inbox check → does implementation work → emits second `SendMessage(to="team-lead", "Done: ...completed implementation...")`)
7. FO polls inbox → sees second Done → sees `fresh: true` on validation → fresh-dispatches `Agent(... description="... validation")`
8. (validation ensign runs → emits Done)
9. FO observes validation Done → marks entity done → terminal
10. Test harness touches keep-alive sentinel → FO emits `end_turn` text and exits

## Contract assertions

Post-rewrite `Phase 3` checks (replacing the current OR-chain / milestone soup):

1. `TeamCreate` emitted (mode-engagement guard, same as keepalive test).
2. `expect_dispatch_close(ensign_name="analysis", overall_timeout_s=120, dispatch_budget_s=90)` — analysis closes.
3. `expect(_is_send_message_to_analysis_ensign, timeout_s=45)` — reuse contract. The SendMessage target must match the ensign name `spacedock-ensign-reuse-test-task-analysis` (not "team-lead", not "implementation" — `analysis` is the reused agent's name; it's being advanced TO the implementation stage). Message body must contain "Advancing to next stage: implementation" AND "Stage definition:" (per `first-officer-shared-core.md:119`).
4. `expect_dispatch_close(ensign_name="validation", overall_timeout_s=120, dispatch_budget_s=90)` — validation closes (fresh: true honored).
5. Sentinel touched; exit.
6. Post-hoc: `w.dispatch_records` has exactly 2 entries (analysis + validation); the SendMessage reuse is NOT a third Agent dispatch.
7. Static template checks (unchanged): reuse conditions in shared-core, SendMessage format, fresh: true disqualification, worktree mode, bare mode guard, feedback-to keep-alive, gate approval references reuse conditions.

## File Structure

- Create: (none — fixture and helper script already exist)
- Modify: `tests/fixtures/reuse-pipeline/README.md` (drop `gate: true` on validation, keep `fresh: true` + `feedback-to: implementation`)
- Modify: `tests/test_reuse_dispatch.py` (complete rewrite — 178 lines → ~140 lines on the cycle-7 template)
- Test: `tests/test_reuse_dispatch.py` itself

No new scripts; `scripts/fo_inbox_poll.py` from cycle-7 is reused verbatim.

## Task breakdown

### Task 1: Un-gate the fixture's validation stage

**Files:**
- Modify: `tests/fixtures/reuse-pipeline/README.md`

- [ ] **Step 1: Verify no other test uses this fixture's gate behavior**

Run: `grep -rn "reuse-pipeline" tests/ | grep -v reuse_dispatch`
Expected: empty (only `test_reuse_dispatch.py` uses this fixture).

- [ ] **Step 2: Edit the fixture README**

```yaml
    - name: validation
      fresh: true
      feedback-to: implementation
      # gate: true  <-- remove this line
```

Exact edit: delete the line `      gate: true` under the validation stage in `tests/fixtures/reuse-pipeline/README.md`. Keep `fresh: true` and `feedback-to: implementation`.

- [ ] **Step 3: Run static tests**

Run: `cd /Users/clkao/git/spacedock/.worktrees/spacedock-ensign-opus-4-7-green-main && make test-static`
Expected: 475 passed (unchanged); the fixture README is not content-asserted by any static test.

- [ ] **Step 4: Commit**

```bash
git add tests/fixtures/reuse-pipeline/README.md
git commit -m "fixture: #208 drop gate on reuse-pipeline validation

Reuse contract tests the analysis->implementation SendMessage reuse
and the fresh: true validation redispatch. The gate at validation is
incidental and has no captain to approve it under claude -p. Drop the
gate so the test can assert the reuse contract cleanly. Gate behavior
is tested in test_gate_guardrail.py with a dedicated fixture."
```

---

### Task 2: Rewrite test_reuse_dispatch.py body on the cycle-7 template

**Files:**
- Modify: `tests/test_reuse_dispatch.py` (complete rewrite)

- [ ] **Step 1: Read the cycle-7 reference implementation**

Open `tests/test_feedback_keepalive.py` and note the sections:
- Imports from `test_lib` (`DispatchBudget`, `emit_skip_result`, `git_add_commit`, `install_agents`, `probe_claude_runtime`, `run_first_officer_streaming`, `setup_fixture`).
- Module constants: `PER_STAGE_OVERALL_S = 120`, `PER_DISPATCH_BUDGET_S = 90`, `SUBPROCESS_EXIT_BUDGET_S = 180`.
- Helpers: `_is_tool_use`, `_is_send_message_to`, `_is_team_create`.
- Haiku xfail block (cycle-7 commit `d2491301`).
- `headless_hint` with plugin-dir path + inbox-poll Bash directive at timeout=5, sentinel file, poll script, seen-file.
- Context-managed `run_first_officer_streaming` with `dispatch_budget=DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0)`.
- `w.expect(_is_team_create, ...)` → `w.expect_dispatch_close(...)` x2 → `w.expect(SendMessage ...)` → sentinel touch → try/except `w.expect_exit(...)`.
- Phase-3 post-hoc: `len(records) == 2`, dispatch budget assertion.
- Static template checks preserved from original.

- [ ] **Step 2: Replace the full file body**

Overwrite `tests/test_reuse_dispatch.py` with:

```python
# ABOUTME: E2E test for ensign reuse dispatch behavior in the FO template.
# ABOUTME: Pinned to teams_mode; asserts analysis dispatch + SendMessage reuse + fresh validation dispatch.

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    DispatchBudget,
    assembled_agent_content,
    emit_skip_result,
    git_add_commit,
    install_agents,
    probe_claude_runtime,
    run_first_officer_streaming,
    setup_fixture,
)


REPO_ROOT = Path(__file__).resolve().parent.parent


PER_STAGE_OVERALL_S = 120
PER_DISPATCH_BUDGET_S = 90

SUBPROCESS_EXIT_BUDGET_S = 180


def _is_tool_use(entry: dict, name: str) -> dict | None:
    if entry.get("type") != "assistant":
        return None
    msg = entry.get("message") or {}
    for block in (msg.get("content") or []):
        if (
            isinstance(block, dict)
            and block.get("type") == "tool_use"
            and block.get("name") == name
        ):
            return block
    return None


def _is_send_message_to(entry: dict, recipient_substr: str) -> bool:
    block = _is_tool_use(entry, "SendMessage")
    if not block:
        return False
    inp = block.get("input") or {}
    return recipient_substr in str(inp.get("to", ""))


def _is_reuse_send_message(entry: dict) -> bool:
    """The reuse contract: SendMessage to the analysis ensign whose body
    carries the implementation stage assignment. The addressee is the
    ensign's full name (e.g. spacedock-ensign-reuse-test-task-analysis);
    the body contains 'Advancing to next stage: implementation' and
    'Stage definition:' per shared-core line 119.
    """
    block = _is_tool_use(entry, "SendMessage")
    if not block:
        return False
    inp = block.get("input") or {}
    to = str(inp.get("to", ""))
    message = str(inp.get("message", ""))
    return (
        "reuse-test-task-analysis" in to
        and "implementation" in message.lower()
        and "Stage definition" in message
    )


def _is_team_create(entry: dict) -> bool:
    return _is_tool_use(entry, "TeamCreate") is not None


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_reuse_dispatch(test_project, model, effort):
    """FO drives teams-mode reuse: TeamCreate -> analysis dispatch -> SendMessage advancing to implementation -> fresh validation dispatch."""
    t = test_project

    # haiku-4-5 drops keep-alive discipline under claude -p (#26426 class);
    # matches the cycle-7 haiku xfail pattern on test_feedback_keepalive.
    # Haiku has no reasoning-effort tiers, so --effort does not affect this.
    if model == "claude-haiku-4-5":
        pytest.xfail(
            reason=(
                "pending haiku-teams reuse — haiku-4-5 drops the "
                "keep-alive Bash-probe discipline at `system init` cycle "
                "boundaries and hallucinates teardown "
                "(anthropics/claude-code#26426 class; opus-4-7 green)"
            )
        )

    print("--- Phase 1: Set up test project from fixture ---")
    setup_fixture(t, "reuse-pipeline", "reuse-pipeline")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: reuse dispatch fixture")

    status_cmd = ["python3", str(t.repo_root / "skills" / "commission" / "bin" / "status"),
                  "--workflow-dir", "reuse-pipeline"]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    status_result = subprocess.run(
        status_cmd + ["--next"], capture_output=True, text=True, cwd=t.test_project_dir,
    )
    t.check("status --next detects dispatchable entity",
            "reuse-test-task" in status_result.stdout)
    print()

    print("--- Phase 2: Run first officer (claude) ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the reuse path."
        )

    abs_workflow = t.test_project_dir / "reuse-pipeline"
    prompt = f"Process all tasks through the workflow at {abs_workflow}/ to terminal completion."

    keepalive_done = t.test_project_dir / ".fo-keepalive-done"
    poll_script = t.repo_root / "scripts" / "fo_inbox_poll.py"
    seen_file = t.test_project_dir / ".fo-inbox-seen"
    headless_hint = (
        f"The spacedock plugin directory is at `{t.repo_root}`. Use it "
        f"directly; do NOT run `find / -name claude-team` — the binaries you "
        f"need are `{t.repo_root}/skills/commission/bin/status` and "
        f"`{t.repo_root}/skills/commission/bin/claude-team`.\n\n"
        f"HEADLESS INBOX-POLLING RULE. You are running in `claude -p` headless "
        f"mode. Per anthropics/claude-code#26426, inbox-delivered teammate "
        f"messages accumulate on disk at `$HOME/.claude/teams/{{team_name}}/"
        f"inboxes/team-lead.json` but are NOT surfaced to your stream. The "
        f"workaround is to surface them yourself via an external polling "
        f"script.\n\n"
        f"Until the sentinel file `{keepalive_done}` exists, every turn "
        f"MUST end with a Bash tool_use (not text) that runs the poll "
        f"script:\n\n"
        f"    python3 {poll_script} --home \"$HOME\" --pattern 'Done:' "
        f"--timeout 5 --seen-file {seen_file}\n\n"
        f"The script blocks up to 5 seconds waiting for a new inbox "
        f"message whose text contains 'Done:'. Its stdout contains the "
        f"teammate message (or is empty on timeout, in which case repeat). "
        f"Treat any 'from: spacedock-ensign-...' block with 'text: Done: "
        f"... completed {{stage}}' as the teammate's completion signal for "
        f"that stage — proceed to the next workflow step per shared-core "
        f"discipline. Never emit `SendMessage(shutdown_request)`, "
        f"`TeamDelete`, or other teardown while awaiting an ensign. Once "
        f"the workflow reaches terminal completion, you may end with text."
    )

    with run_first_officer_streaming(
        t,
        prompt,
        agent_id="spacedock:first-officer",
        extra_args=[
            "--model", model,
            "--effort", effort,
            "--max-budget-usd", "5.00",
            "--append-system-prompt", headless_hint,
        ],
        dispatch_budget=DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0),
    ) as w:
        w.expect(_is_team_create, timeout_s=PER_STAGE_OVERALL_S, label="TeamCreate emitted")
        print("[OK] TeamCreate emitted (teams mode engaged)")

        analysis_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="analysis",
            label="analysis dispatch close",
        )
        print(f"[OK] analysis dispatch closed in {analysis_record.elapsed:.1f}s")

        # Reuse contract: after analysis completes, the FO MUST advance the
        # analysis ensign to implementation via SendMessage, NOT a fresh Agent.
        # The SendMessage target is the analysis ensign's full name; the body
        # carries the implementation stage assignment.
        w.expect(
            _is_reuse_send_message,
            timeout_s=PER_STAGE_OVERALL_S,
            label="SendMessage advancing analysis ensign to implementation (reuse)",
        )
        print("[OK] reuse dispatch via SendMessage to analysis ensign")

        validation_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="validation",
            label="validation dispatch close",
        )
        print(f"[OK] validation dispatch closed in {validation_record.elapsed:.1f}s (fresh: true honored)")

        # Workflow contract satisfied — release the keep-alive sentinel.
        keepalive_done.touch()
        print(f"[OK] keep-alive sentinel {keepalive_done.name} touched")

        try:
            w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)
            print("[OK] FO exited cleanly after sentinel")
        except Exception as exc:
            print(f"  NOTE: FO did not exit within {SUBPROCESS_EXIT_BUDGET_S}s post-sentinel ({type(exc).__name__}); contract assertions already passed")

    print("--- Phase 3: Validation ---")
    records = w.dispatch_records
    print(f"  dispatch records: {[(r.ensign_name, round(r.elapsed, 1)) for r in records]}")
    t.check(
        "FO emitted exactly two ensign Agent() dispatches (analysis + validation; implementation reused via SendMessage)",
        len(records) == 2,
    )
    t.check(
        "all dispatches closed under the per-dispatch budget",
        all(r.elapsed <= PER_DISPATCH_BUDGET_S for r in records),
    )

    print()
    print("[Static Template Checks]")
    core = (REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md").read_text()
    runtime_ref = (REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime.md").read_text()
    assembled = assembled_agent_content(t, "first-officer")

    t.check("reuse conditions documented in shared-core",
            "Reuse conditions" in core and "bare mode" in core.lower())
    t.check("SendMessage format in reuse path",
            "SendMessage(" in core and "Stage definition:" in core)
    t.check("fresh: true disqualifies reuse",
            bool(re.search(r"NOT have.*fresh: true", core)))
    t.check("worktree mode match required",
            bool(re.search(r"same.*worktree.*mode", core, re.IGNORECASE)))
    t.check("bare mode guard present",
            bool(re.search(r"Not in bare mode", core)))
    t.check("feedback-to keep-alive in fresh dispatch path",
            bool(re.search(r"If fresh dispatch.*feedback-to.*keep.*alive", core, re.DOTALL | re.IGNORECASE)))
    t.check("gate approval references reuse conditions",
            bool(re.search(r"captain approves.*reuse conditions", core, re.DOTALL | re.IGNORECASE)))
    t.check("no 'Always dispatch fresh' in assembled FO",
            "Always dispatch fresh" not in assembled)
    t.check("dispatch step uses neutral language",
            "Dispatch a worker via" in core and "Dispatch a fresh worker" not in core)
    t.check("runtime clarifies SendMessage for reuse only",
            "NEVER use SendMessage to dispatch" not in runtime_ref
            and bool(re.search(r"SendMessage.*completion path|completion path.*SendMessage", runtime_ref, re.IGNORECASE)))

    t.finish()
```

- [ ] **Step 3: Run static tests**

Run: `cd /Users/clkao/git/spacedock/.worktrees/spacedock-ensign-opus-4-7-green-main && make test-static`
Expected: 475 passed (no static test content asserts on this file; the rewrite preserves static-check shape).

- [ ] **Step 4: Run offline dispatch-budget unit tests**

Run: `uv run pytest tests/test_dispatch_budget.py -x -q`
Expected: 21 passed (the watcher close anchors used by this test are the same ones that cycle-7 locked in).

- [ ] **Step 5: Commit**

```bash
git add tests/test_reuse_dispatch.py
git commit -m "impl: #208 rewrite test_reuse_dispatch on cycle-7 streaming watcher + inbox-poll pattern

Replace the pre-cycle-7 milestones + OR-chain shape with strict per-stage
contract assertions on run_first_officer_streaming + DispatchBudget +
expect_dispatch_close. Un-xfail on opus-4-7; retain haiku xfail per the
cycle-7 pattern (anthropics/claude-code#26426 class).

Drops the prompt nanny hints (reuse conditions coaching, gate auto-
approval directive). Test exercises the real reuse contract:
TeamCreate -> analysis Agent -> SendMessage(to=analysis-ensign,
\"Advancing to next stage: implementation\" + Stage definition) ->
validation Agent (fresh: true honored). Post-hoc: exactly 2 Agent
dispatches, implementation is reused via SendMessage not a 3rd Agent."
```

---

### Task 3: Live verification at opus-4-7 teams mode

**Files:**
- (none — test-only)

- [ ] **Step 1: Prepare isolated temp dir**

Run: `mkdir -p /tmp/reuse-r1`

- [ ] **Step 2: Single live run at opus-4-7 teams**

Run:

```bash
cd /Users/clkao/git/spacedock/.worktrees/spacedock-ensign-opus-4-7-green-main && \
  unset CLAUDECODE && \
  KEEP_TEST_DIR=1 SPACEDOCK_TEST_TMP_ROOT=/tmp/reuse-r1 \
  CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1 \
  uv run pytest tests/test_reuse_dispatch.py --runtime claude \
    --model opus --effort low --team-mode=teams -v
```

Expected: PASSED in 3-5 minutes. The test should fire every `[OK]` marker in order: TeamCreate → analysis dispatch close → reuse SendMessage → validation dispatch close → sentinel touched.

- [ ] **Step 3: Triage on failure**

If the run fails, inspect the fo-log:
- `find /tmp/reuse-r1 -name "fo-log.jsonl" | head -1` → the FO log path.
- Look for `SendMessage` tool_uses — confirm one targets the analysis ensign name and carries the implementation stage definition.
- If `analysis` dispatch never closes: check the inbox-poll Bash tool_results (`grep "spacedock-ensign-reuse-test-task-analysis" {fo-log}`) to confirm the Done: message was surfaced.
- If reuse SendMessage never fires: check the FO text for "Advancing to next stage" or "reuse" — likely an opus rule-following issue, worth filing as a separate entity.

- [ ] **Step 4: Commit evidence**

```bash
# Assuming green
echo "green at opus-4-7 teams, $(date)" >> docs/plans/test-reuse-dispatch-cycle7-port.md
git add docs/plans/test-reuse-dispatch-cycle7-port.md
git commit -m "report: #208 N=1 green at opus-4-7 teams"
```

If red after three rounds of triage, file a follow-up entity with the specific failure mode and pause. Do NOT add OR-chain softeners to the test to make it green.

---

### Task 4: Un-xfail + stage report

**Files:**
- Modify: `docs/plans/test-reuse-dispatch-cycle7-port.md` (set `status: validation` or `status: done` per outcome)

- [ ] **Step 1: Update the entity file status**

If green at opus-4-7 teams:

```yaml
status: done
completed: "{ISO-8601 timestamp}"
verdict: PASSED
```

If red and filed follow-up entity:

```yaml
status: validation
# leave completed/verdict empty; describe the follow-up in the body
```

- [ ] **Step 2: Write a brief `## Stage Report: implementation` section**

Body should include: final-commit SHAs for the fixture edit + test rewrite, live-run wallclock, fo-log evidence path, any follow-up entity IDs filed for residual failures.

- [ ] **Step 3: Commit**

```bash
git add docs/plans/test-reuse-dispatch-cycle7-port.md
git commit -m "report: #208 stage report — done|validation

{one-sentence summary of outcome}"
```

- [ ] **Step 4: Push**

Run: `git push origin spacedock-ensign/opus-4-7-green-main`

---

## Acceptance criteria

1. `tests/fixtures/reuse-pipeline/README.md` no longer has `gate: true` on the validation stage; `fresh: true` and `feedback-to: implementation` are retained.
2. `tests/test_reuse_dispatch.py` no longer uses `run_first_officer` / `LogParser`; it uses `run_first_officer_streaming` + `DispatchBudget` + `FOStreamWatcher.expect_dispatch_close`.
3. Test carries `@pytest.mark.teams_mode`; the `@pytest.mark.xfail(strict=False, reason="pending #160 ...")` marker is gone; an inline `pytest.xfail(reason="pending haiku-teams reuse ...")` guard exists only for `model == "claude-haiku-4-5"`.
4. Prompt is a single line: `f"Process all tasks through the workflow at {abs_workflow}/ to terminal completion."` — no FO-discipline coaching.
5. All nine static template checks from the original test survive in the rewrite (they do not depend on the runtime shape and remain valid).
6. `make test-static` passes at 475+ tests.
7. `uv run pytest tests/test_dispatch_budget.py` stays at 21 passed.
8. Single live run at `--model opus --effort low --team-mode=teams` under `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` passes cleanly in 3-5 minutes with all `[OK]` markers fired.
9. `docs/plans/test-reuse-dispatch-cycle7-port.md` (this file) has its status updated per outcome with a `## Stage Report` section.

## Coordination notes

- Cycle-8 teammates are concurrently porting the cycle-7 pattern to `test_standing_teammate` (tasks #5-9) and `test_merge_hook_guardrail` (tasks #10-14). The files this plan touches do not overlap with theirs:
  - This plan: `tests/test_reuse_dispatch.py`, `tests/fixtures/reuse-pipeline/README.md`.
  - Cycle-8 standing_teammate: `tests/test_standing_teammate.py`, `tests/fixtures/standing-teammate/`.
  - Cycle-8 merge_hook: `tests/test_merge_hook_guardrail.py`, `tests/fixtures/merge-hook-pipeline/`.
- No edits to `scripts/test_lib.py`, `scripts/fo_inbox_poll.py`, or `tests/README.md` are required — all pieces from cycle-7 are reusable.
- If `scripts/test_lib.py` is modified by another cycle-8 teammate during this plan's execution, verify the cycle-7 close-anchor helpers (`_tool_result_text`, `_parse_inbox_done_sender`, `_find_open_dispatch_for_sender`, `_looks_like_bare_done`) and the `expect_dispatch_close` / `DispatchBudget` APIs are still present and behave per the offline tests in `tests/test_dispatch_budget.py`.

## Out of scope

- Bare-mode reuse test. Bare mode does not use `SendMessage` for reuse — the analogous contract would be "second Agent() call with the same worker label" or "synchronous `Agent(reuse=true)` if the platform supports it." Out of scope here; bare mode gets its own sibling test if needed.
- Fixing #160 (haiku multi-dispatch compression) at the FO shared-core level. This plan retains the haiku xfail; a separate entity addresses the compression root cause via prose or structural fix.
- `test_rejection_flow` and `test_rebase_branch_before_push` ports. These are Tier-A hygiene siblings; each gets its own entity using the same pattern.

## Summary

This plan ports `test_reuse_dispatch` onto the cycle-7 streaming watcher + inbox-poll keep-alive pattern that made `test_feedback_keepalive` green at opus-4-7 teams. Scope is narrowly the test + its fixture (one-character fixture edit to drop the validation gate). No shared-core prose changes; no harness modifications. Execution is four small tasks (fixture edit, test rewrite, live verification, stage report). Expected outcome: opus-4-7 teams GREEN, haiku xfailed with rationale matching cycle-7.
