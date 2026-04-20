# ABOUTME: E2E test for the feedback-to keepalive rule in the first-officer template.
# ABOUTME: Pinned to teams_mode; asserts TeamCreate + impl dispatch + validation dispatch + SendMessage feedback reuse.

from __future__ import annotations

import re
import subprocess
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
    run_first_officer_streaming,
    setup_fixture,
)


REPO_ROOT = Path(__file__).resolve().parent.parent


PER_STAGE_OVERALL_S = 45
PER_DISPATCH_BUDGET_S = 30

SUBPROCESS_EXIT_BUDGET_S = 60


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


def _is_team_create(entry: dict) -> bool:
    return _is_tool_use(entry, "TeamCreate") is not None


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_feedback_keepalive(test_project, model, effort):
    """FO drives teams-mode keepalive: TeamCreate -> impl ensign -> validation ensign -> SendMessage reuse (not fresh Agent)."""
    t = test_project

    print("--- Phase 1: Set up test project from fixture ---")
    setup_fixture(t, "keepalive-pipeline", "keepalive-pipeline")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: keepalive test fixture")

    status_cmd = ["python3", str(t.repo_root / "skills" / "commission" / "bin" / "status"),
                  "--workflow-dir", "keepalive-pipeline"]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    status_result = subprocess.run(
        status_cmd + ["--next"], capture_output=True, text=True, cwd=t.test_project_dir,
    )
    t.check("status --next detects dispatchable entity",
            "keepalive-test-task" in status_result.stdout)
    print()

    print("--- Phase 2: Run first officer (claude) ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the keepalive path."
        )

    abs_workflow = t.test_project_dir / "keepalive-pipeline"
    prompt = f"Process all tasks through the workflow at {abs_workflow}/ to terminal completion."

    # Keep-alive sentinel: under `claude -p`, a turn ending with text-only
    # emits stop_reason=end_turn and closes the prompt cycle; the next cycle
    # starts with fresh context and opus-4-7-low hallucinates reasons to tear
    # down the team. This sentinel file forces every idle turn to end with a
    # tool_use (stop_reason=tool_use) instead of text-only, preserving the
    # cycle until the test harness signals workflow completion.
    keepalive_done = t.test_project_dir / ".fo-keepalive-done"
    headless_hint = (
        f"The spacedock plugin directory is at `{t.repo_root}`. Use it "
        f"directly; do NOT run `find / -name claude-team` — the binaries you "
        f"need are `{t.repo_root}/skills/commission/bin/status` and "
        f"`{t.repo_root}/skills/commission/bin/claude-team`.\n\n"
        f"HEADLESS KEEP-ALIVE RULE. You are running in `claude -p` headless "
        f"mode. The file `{keepalive_done}` signals end-of-workflow. Until "
        f"that file exists, every turn MUST end with a tool_use, not with "
        f"text. When your only work is to wait for an ensign's completion "
        f"signal, end the turn by running the Bash tool with command "
        f"`test -f {keepalive_done}` as a no-op keep-alive probe. Do NOT "
        f"emit `SendMessage(shutdown_request)`, `TeamDelete`, or any other "
        f"teardown action while awaiting an ensign. The ensign completion "
        f"signal is a user-role message starting with `Done:` delivered via "
        f"the team inbox. Once all ensigns have completed and the workflow "
        f"is done, you may end with text."
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
        dispatch_budget=DispatchBudget(soft_s=15.0, hard_s=60.0, shutdown_grace_s=10.0),
    ) as w:
        w.expect(_is_team_create, timeout_s=PER_STAGE_OVERALL_S, label="TeamCreate emitted")
        print("[OK] TeamCreate emitted (teams mode engaged)")

        impl_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="implementation",
            label="implementation dispatch close",
        )
        print(f"[OK] implementation dispatch closed in {impl_record.elapsed:.1f}s")

        validation_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="validation",
            label="validation dispatch close",
        )
        print(f"[OK] validation dispatch closed in {validation_record.elapsed:.1f}s")

        # Keepalive contract: cycle-2 feedback routing MUST be SendMessage to the
        # still-alive implementation ensign, NOT a fresh Agent() dispatch.
        w.expect(
            lambda e: _is_send_message_to(e, "implementation"),
            timeout_s=PER_STAGE_OVERALL_S,
            label="SendMessage to implementation ensign (feedback reuse)",
        )
        print("[OK] feedback routed via SendMessage to kept-alive implementation ensign")

        # Workflow contract satisfied — release the FO's keep-alive sentinel
        # so it can emit its final end_turn and exit cleanly.
        keepalive_done.touch()
        print(f"[OK] keep-alive sentinel {keepalive_done.name} touched")

        w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)

    print("--- Phase 3: Validation ---")
    records = w.dispatch_records
    print(f"  dispatch records: {[(r.ensign_name, round(r.elapsed, 1)) for r in records]}")
    t.check(
        "FO emitted exactly two ensign Agent() dispatches (impl + validation; feedback via SendMessage)",
        len(records) == 2,
    )
    t.check(
        "all dispatches closed under the per-dispatch budget",
        all(r.elapsed <= PER_DISPATCH_BUDGET_S for r in records),
    )

    print()
    print("[Static Template Checks]")
    core = (REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md").read_text()
    t.check(
        "shared-core contains feedback-to keepalive rule for fresh dispatch",
        bool(re.search(r"If fresh dispatch.*feedback-to.*keep.*alive", core, re.DOTALL | re.IGNORECASE)),
    )
    t.check(
        "shared-core contains auto-bounce rule for REJECTED feedback gates",
        bool(re.search(r"feedback gate.*REJECTED.*auto-bounce", core, re.DOTALL | re.IGNORECASE)),
    )
    t.check(
        "shared-core documents feedback rejection flow with feedback-to routing",
        bool(re.search(r"Feedback Rejection Flow", core)) and bool(re.search(r"feedback-to.*target", core, re.IGNORECASE)),
    )

    t.finish()
