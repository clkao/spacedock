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
    if model == "claude-haiku-4-5" or model == "haiku" or "haiku" in model.lower():
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
