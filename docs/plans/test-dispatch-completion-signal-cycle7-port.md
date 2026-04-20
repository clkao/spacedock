---
id: "209"
title: "Port test_dispatch_completion_signal to cycle-7 pattern — streaming watcher + inbox-poll keep-alive + teams-mode-pinned contract"
status: implementation
source: "entity #198 (fo-runtime-test-failures-post-154) — test currently xfailed on team-mode completion-signal exit hang; cycle-7 identified anthropics/claude-code#26426 as the upstream root cause and provided the keep-alive + inbox-poll workaround"
started: 2026-04-20T06:47:24Z
completed:
verdict:
score: 0.65
worktree: .worktrees/spacedock-ensign-test-dispatch-completion-signal-cycle7-port
issue:
pr:
mod-block: merge:pr-merge
---

# Port test_dispatch_completion_signal to Cycle-7 Pattern Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Un-xfail and green `tests/test_dispatch_completion_signal.py` at opus-4-7 teams mode by rewriting it on top of the cycle-7 streaming watcher + inbox-poll keep-alive pattern. Haiku-teams stays xfailed with a rationale matching the cycle-7 haiku xfail pattern.

**Architecture:** Replace `run_first_officer` + `LogParser` with `run_first_officer_streaming` + `FOStreamWatcher`. Pin to `@pytest.mark.teams_mode`. Assert the completion-signal contract via one `expect(_is_team_create)` + one `expect_dispatch_close(ensign_name="work")` + a post-hoc entity-archival check on disk. The `work` stage is worktree-enabled and has no downstream dispatches; the contract is literally "FO saw the ensign's Done and advanced the entity to done", which is exactly what the cycle-7 pattern surfaces.

**Tech Stack:** Python, pytest, `scripts/test_lib.py` (`run_first_officer_streaming`, `FOStreamWatcher`, `DispatchBudget`, `expect_dispatch_close`), `scripts/fo_inbox_poll.py`, `tests/fixtures/completion-signal-pipeline/`.

---

## Background

`tests/test_dispatch_completion_signal.py` verifies the core contract that #114 originally tracked: a team-dispatched ensign's `SendMessage(to="team-lead", "Done: ...")` must wake the FO, which then advances the entity past the dispatched stage. Pre-fix the FO's `DISPATCH IDLE GUARDRAIL` would wait forever because it never observed the completion signal.

Currently xfailed under entity #198 as "pending #198 — runtime FO team-mode completion-signal exit hang". Evidence from cycle-6 full-suite runs (`docs/plans/_evidence/green-opus-4-7-full-suite/cycle6/`): the failure is `fo_exit != 0 within budget`. Cycle-7 identified the upstream root cause ([anthropics/claude-code#26426](https://github.com/anthropics/claude-code/issues/26426)): `InboxPoller` is a React UI hook and doesn't fire under `claude -p`. The completion signal physically lands in `$HOME/.claude/teams/{team}/inboxes/team-lead.json` but is never delivered to the FO's stream. The cycle-7 workaround (external polling script surfaces inbox entries as Bash tool_result) is the exact fix this test needs.

This is the **cleanest cycle-7 port target** of the three Tier-A siblings: single dispatchable stage, no gates, no feedback-to, no reuse contract. The only contract is "FO observed Done, advanced status."

## Fixture shape (unchanged)

`tests/fixtures/completion-signal-pipeline/README.md`:

```yaml
stages:
  defaults:
    worktree: false
    fresh: false
    gate: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: work
      worktree: true
    - name: done
      terminal: true
```

Single dispatchable stage (`work`), no gates, no feedback-to. No fixture edits required.

## Expected FO trajectory (teams mode, opus-4-7)

1. `TeamCreate(test-project-completion-signal-pipeline-...)`
2. `Agent(subagent_type="spacedock:ensign", description="... work")` — initial and only dispatch
3. (ensign runs in its worktree → creates the deliverable → commits → emits `SendMessage(to="team-lead", "Done: ...completed work...")`)
4. FO polls inbox via Bash → reads the `Done:` message
5. FO advances status: `work → done` via `status --set`, commits on main, optionally archives
6. Test harness touches keep-alive sentinel → FO exits

## Contract assertions

Post-rewrite `Phase 3`:

1. `TeamCreate` emitted.
2. `expect_dispatch_close(ensign_name="work", overall_timeout_s=120, dispatch_budget_s=90)` — ensign closes.
3. Sentinel touched; `expect_exit` wrapped in try/except.
4. Post-hoc on disk: entity file `completion-signal-task.md` is either archived OR has `status: done` in frontmatter.
5. Post-hoc sanity: the ensign's dispatch prompt (recoverable from the Agent tool_use input in the FO log via an inline helper) contains `SendMessage(to="team-lead"` — verifies the FO's dispatch template still injects the completion-signal instruction.

Note: the original test has a fallback for bare mode ("FO dispatched in bare mode (no team_name); SendMessage unnecessary"). The rewrite pins to `teams_mode`, so the bare-fallback branch is removed. Bare mode is not exercised here.

## File Structure

- Create: (none)
- Modify: `tests/test_dispatch_completion_signal.py` (complete rewrite — 124 lines → ~170 lines on the cycle-7 template)
- Test: `tests/test_dispatch_completion_signal.py` itself

No new scripts. `scripts/fo_inbox_poll.py` from cycle-7 is reused verbatim.

## Task breakdown

### Task 1: Rewrite test_dispatch_completion_signal.py on the cycle-7 template

**Files:**
- Modify: `tests/test_dispatch_completion_signal.py` (complete rewrite)

- [ ] **Step 1: Read the cycle-7 reference implementation**

Open `tests/test_feedback_keepalive.py` and note: imports, module constants (`PER_STAGE_OVERALL_S = 120`, `PER_DISPATCH_BUDGET_S = 90`, `SUBPROCESS_EXIT_BUDGET_S = 180`), `_is_tool_use` / `_is_team_create` / `_is_send_message_to` helpers, haiku xfail block (commit `d2491301`), `headless_hint` with inbox-poll directive at timeout=5, context-managed `run_first_officer_streaming` with `DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0)`, the `expect` → `expect_dispatch_close` → sentinel touch → try/except `expect_exit` flow.

- [ ] **Step 2: Replace the full file body**

Overwrite `tests/test_dispatch_completion_signal.py` with:

```python
# ABOUTME: E2E regression test for team-mode dispatch completion-signal in the FO template.
# ABOUTME: Pinned to teams_mode; asserts TeamCreate + work dispatch close + entity advances to done.

from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    DispatchBudget,
    emit_skip_result,
    git_add_commit,
    install_agents,
    probe_claude_runtime,
    read_entity_frontmatter,
    run_first_officer_streaming,
    setup_fixture,
)


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


def _is_team_create(entry: dict) -> bool:
    return _is_tool_use(entry, "TeamCreate") is not None


def _first_agent_dispatch_prompt(fo_log_path: Path) -> str:
    """Return the prompt text from the first Agent(subagent_type='spacedock:ensign') dispatch in fo-log."""
    import json
    with open(fo_log_path) as fh:
        for raw in fh:
            try:
                entry = json.loads(raw)
            except json.JSONDecodeError:
                continue
            if entry.get("type") != "assistant":
                continue
            for block in (entry.get("message") or {}).get("content") or []:
                if (
                    isinstance(block, dict)
                    and block.get("type") == "tool_use"
                    and block.get("name") == "Agent"
                ):
                    inp = block.get("input") or {}
                    if inp.get("subagent_type") == "spacedock:ensign":
                        return str(inp.get("prompt") or "")
    return ""


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_dispatch_completion_signal(test_project, model, effort):
    """FO drives teams-mode single-stage completion: TeamCreate -> work dispatch close -> entity advances to done."""
    t = test_project

    # haiku-4-5 drops keep-alive discipline under claude -p (#26426 class);
    # matches the cycle-7 haiku xfail pattern on test_feedback_keepalive.
    if model == "claude-haiku-4-5":
        pytest.xfail(
            reason=(
                "pending haiku-teams completion-signal — haiku-4-5 drops the "
                "keep-alive Bash-probe discipline at `system init` cycle "
                "boundaries and hallucinates teardown "
                "(anthropics/claude-code#26426 class; opus-4-7 green)"
            )
        )

    print("--- Phase 1: Set up test project from fixture ---")
    setup_fixture(t, "completion-signal-pipeline", "completion-signal-pipeline")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: completion-signal regression fixture")
    status_cmd = [
        "python3",
        str(t.repo_root / "skills" / "commission" / "bin" / "status"),
        "--workflow-dir", "completion-signal-pipeline",
    ]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    print()

    print("--- Phase 2: Run first officer (claude) ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the completion-signal regression."
        )

    abs_workflow = t.test_project_dir / "completion-signal-pipeline"
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
            "--max-budget-usd", "3.00",
            "--append-system-prompt", headless_hint,
        ],
        dispatch_budget=DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0),
    ) as w:
        w.expect(_is_team_create, timeout_s=PER_STAGE_OVERALL_S, label="TeamCreate emitted")
        print("[OK] TeamCreate emitted (teams mode engaged)")

        work_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="work",
            label="work dispatch close",
        )
        print(f"[OK] work dispatch closed in {work_record.elapsed:.1f}s")

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
        "FO emitted exactly one ensign Agent() dispatch (work stage)",
        len(records) == 1,
    )

    print()
    print("[Entity Advancement]")
    entity_main = t.test_project_dir / "completion-signal-pipeline" / "completion-signal-task.md"
    entity_archive = t.test_project_dir / "completion-signal-pipeline" / "_archive" / "completion-signal-task.md"

    if entity_archive.is_file():
        t.pass_("entity advanced and was archived without manual captain intervention")
    elif entity_main.is_file():
        fm = read_entity_frontmatter(entity_main)
        status_val = fm.get("status", "")
        if status_val == "done":
            t.pass_(f"entity advanced to terminal stage (status: {status_val})")
        else:
            t.fail(
                f"entity did NOT advance past dispatched stage (status: {status_val!r}). "
                "This reproduces the bug: team-dispatched ensign sent no completion signal, "
                "so the FO's DISPATCH IDLE GUARDRAIL waited forever."
            )
    else:
        t.fail("entity file missing from both main and _archive (unexpected state)")

    print()
    print("[Dispatch Template Sanity]")
    fo_log_path = t.log_dir / "fo-log.jsonl"
    dispatch_prompt = _first_agent_dispatch_prompt(fo_log_path)
    t.check(
        "team-mode ensign prompt carries SendMessage completion-signal instruction",
        'SendMessage(to="team-lead"' in dispatch_prompt,
    )

    t.finish()
```

- [ ] **Step 3: Run static tests**

Run: `cd /Users/clkao/git/spacedock/.worktrees/spacedock-ensign-opus-4-7-green-main && make test-static`
Expected: 475 passed (unchanged; no content test inspects this file).

- [ ] **Step 4: Run offline dispatch-budget unit tests**

Run: `uv run pytest tests/test_dispatch_budget.py -x -q`
Expected: 21 passed (same cycle-7 watcher anchors).

- [ ] **Step 5: Commit**

```bash
git add tests/test_dispatch_completion_signal.py
git commit -m "impl: #211 rewrite test_dispatch_completion_signal on cycle-7 pattern

Replace run_first_officer + LogParser with run_first_officer_streaming +
FOStreamWatcher. Pin @pytest.mark.teams_mode, un-xfail on opus-4-7,
retain haiku xfail (anthropics/claude-code#26426 class).

Contract: TeamCreate -> work dispatch close -> entity advances to done.
Drop the bare-mode fallback branch (test is teams-only now). Retain the
dispatch-template sanity check (FO's Agent prompt must carry
SendMessage(to=\"team-lead\"...) instruction) via a small inline helper.

make test-static: 475 passed. offline dispatch-budget: 21 passed."
```

---

### Task 2: Live verification at opus-4-7 teams mode

**Files:**
- (none — test-only)

- [ ] **Step 1: Prepare isolated temp dir**

Run: `mkdir -p /tmp/completion-r1`

- [ ] **Step 2: Single live run**

Run:

```bash
cd /Users/clkao/git/spacedock/.worktrees/spacedock-ensign-opus-4-7-green-main && \
  unset CLAUDECODE && \
  KEEP_TEST_DIR=1 SPACEDOCK_TEST_TMP_ROOT=/tmp/completion-r1 \
  CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1 \
  uv run pytest tests/test_dispatch_completion_signal.py --runtime claude \
    --model opus --effort low --team-mode=teams -v
```

Expected: PASSED in 2-4 minutes. Single-stage pipeline is faster than keepalive.

- [ ] **Step 3: Triage on failure**

If red, inspect the fo-log:
- `find /tmp/completion-r1 -name "fo-log.jsonl" | head -1`
- Look for `fo_inbox_poll.py` Bash invocations and check that at least one tool_result contains `from: spacedock-ensign-completion-signal-task-work` + `text: Done: ...completed work...`.
- Confirm the entity's frontmatter was transitioned to `status: done` or archived.
- If dispatch never closed: check whether the FO sent `shutdown_request` (keep-alive rule violation) — file a follow-up entity if the discipline slipped.

---

### Task 3: Un-xfail entity #198 + stage report

**Files:**
- Modify: `docs/plans/fo-runtime-test-failures-post-154.md` (update the `test_dispatch_completion_signal` section)
- Modify: `docs/plans/test-dispatch-completion-signal-cycle7-port.md` (set `status: done`, add stage report)

- [ ] **Step 1: Update #198's section on this test**

In `docs/plans/fo-runtime-test-failures-post-154.md`, under the `test_dispatch_completion_signal` heading (near line 28), add a note:

```markdown
**Resolved by cycle-7 port (#211).** The root cause was anthropics/claude-code#26426 (InboxPoller is a React UI hook, doesn't fire under `-p`). The cycle-7 keep-alive + inbox-poll pattern ports cleanly here; the test was rewritten on that shape and green at opus-4-7 teams N=1 in {wallclock}. See `docs/plans/test-dispatch-completion-signal-cycle7-port.md`.
```

- [ ] **Step 2: Update this entity's status**

Set frontmatter to:

```yaml
status: done
completed: "{ISO-8601 timestamp}"
verdict: PASSED
```

Add a brief `## Stage Report: implementation` section with commit SHAs for the rewrite + live run wallclock + fo-log evidence path.

- [ ] **Step 3: Commit and push**

```bash
git add docs/plans/fo-runtime-test-failures-post-154.md docs/plans/test-dispatch-completion-signal-cycle7-port.md
git commit -m "report: #211 done — test_dispatch_completion_signal green at opus-4-7 teams

{wallclock} single-run; #198 section updated with cycle-7 resolution note."
git push origin spacedock-ensign/opus-4-7-green-main
```

---

## Acceptance criteria

1. `tests/test_dispatch_completion_signal.py` no longer uses `run_first_officer` / `LogParser`; uses `run_first_officer_streaming` + `DispatchBudget` + `FOStreamWatcher.expect_dispatch_close`.
2. Test carries `@pytest.mark.teams_mode`; `@pytest.mark.xfail(...reason="pending #198"...)` is gone; inline `pytest.xfail` guard exists only for `model == "claude-haiku-4-5"`.
3. Prompt is a single line: `f"Process all tasks through the workflow at {abs_workflow}/ to terminal completion."` — no coaching.
4. Bare-mode fallback branch removed; test is teams-mode-only.
5. `make test-static` passes at 475+ tests.
6. `uv run pytest tests/test_dispatch_budget.py` stays at 21 passed.
7. Single live run at `--model opus --effort low --team-mode=teams` with `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` passes cleanly in 2-4 minutes.
8. `docs/plans/fo-runtime-test-failures-post-154.md` carries a resolution note pointing at this entity.
9. This entity's status advances to `done` with a stage report.

## Coordination notes

- Concurrent cycle-8 work on `test_standing_teammate` and `test_merge_hook_guardrail` does not touch this file or its fixture.
- Sibling entities: #210 (`test-rejection-flow-cycle7-port`), #211 (`test-checklist-e2e-runtime-text-assertion-fix`) — different bug classes; independent.
- No shared-core prose edits required.

## Out of scope

- Bare-mode sibling. Bare `Agent()` is synchronous; "completion signal" in bare is the tool_result payload itself, not an inbox message. Different contract; needs its own test if required.
- Fixing the FO prose to emit the completion-signal instruction under alternate template paths. The current template does the right thing; we just test it.

## Summary

Smallest of the three Tier-A cycle-7 ports. One-stage fixture, one dispatch assertion, one post-hoc on-disk check, one dispatch-template sanity check. Expected outcome: opus-4-7 teams GREEN, haiku xfailed with rationale matching cycle-7.

## Stage Report: implementation

- DONE: Read entity body, cycle-7 infrastructure (scripts/test_lib.py streaming watcher + DispatchBudget + expect_dispatch_close + `_find_open_dispatch_for_sender`), scripts/fo_inbox_poll.py, and studied tests/test_feedback_keepalive.py as the port template.
  All five primitives exist at expected symbols in test_lib.py (checked lines 311, 322, 559, 616, 701, 743, 978, 1133, 1855, 1864).
- DONE: Port test_dispatch_completion_signal to cycle-7 pattern; remove outer `@pytest.mark.xfail`; retain inline haiku xfail.
  Commit fbba952c — 142 insertions / 51 deletions; teams_mode pinned; bare-mode fallback removed; SendMessage template sanity retained via inline `_first_agent_dispatch_prompt` helper reading fo-log.jsonl.
- DONE: `make test-static` green.
  476 passed, 22 deselected, 10 subtests passed in 24.88s.
- DONE: Target test N=3 at opus-low with `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`.
  run1 PASSED 1m40s; run2 PASSED 4m35s; run3 PASSED 4m24s. **3/3 PASS**. Evidence at `/tmp/209-dispatch-signal-evidence/run{1,2,3}.log` and KEEP_TEST_DIR artifacts at `/tmp/209-run{1,2,3}/`.
- DONE: Offline dispatch-budget unit tests stay green.
  22 passed in 3.15s.
- DONE: Push branch and open PR.
  Branch `spacedock-ensign/test-dispatch-completion-signal-cycle7-port` pushed; PR #140 opened (https://github.com/clkao/spacedock/pull/140) with Motivation + What changed + Evidence + audit-link + "Closes #209". CI approval NOT granted per instructions.
- SKIPPED: Un-xfail entity #198 + update `docs/plans/fo-runtime-test-failures-post-154.md`.
  Per assignment scope: my worktree owns the port only; the shared #198 tracking entity is edited by the FO after this PR merges.

### Summary

Cycle-7 port landed cleanly on the first cycle: 3/3 opus-low live PASS, static 476 green, offline budget 22 green. Contract satisfied via `TeamCreate` → `work` dispatch close → entity advances to `done`/archived, with the dispatch-template sanity check preserved as an inline fo-log.jsonl reader. PR #140 ready for team-lead review; haiku xfail retained with anthropics/claude-code#26426-class rationale matching cycle-7 sibling `test_feedback_keepalive`.
