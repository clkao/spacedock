# ABOUTME: E2E test for the feedback-to keepalive rule in the first-officer template.
# ABOUTME: Asserts strict Path-A shape (impl dispatch -> validation dispatch -> second impl dispatch) with per-dispatch budgets.

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

IMPL_OVERALL_S = 180
VALIDATION_OVERALL_S = 180
FEEDBACK_OVERALL_S = 240

PER_DISPATCH_BUDGET_S = 45

SUBPROCESS_EXIT_BUDGET_S = 120


@pytest.mark.live_claude
def test_feedback_keepalive(test_project, model, effort, request):
    """FO drives Path-A (impl -> validation rejects -> re-dispatch impl) with per-dispatch budgets."""
    t = test_project

    team_mode_opt = request.config.getoption("--team-mode")
    if team_mode_opt in ("teams", "bare"):
        resolved_team_mode = team_mode_opt
    else:
        import os as _os
        _env = _os.environ.get("CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS", "").strip().lower()
        resolved_team_mode = "teams" if _env in ("1", "true") else "bare"
    if resolved_team_mode == "bare" and model == "claude-haiku-4-5":
        pytest.xfail(
            reason=(
                "pending #200 — haiku-bare FO tool-shape discipline "
                "(subagent_type=None validation, SendMessage nested in Agent prompt)"
            )
        )

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
    prompt = f"Process all tasks through the workflow at {abs_workflow}/ to terminal completion. Drive every dispatchable task through its stages until the entity reaches the done stage."

    with run_first_officer_streaming(
        t,
        prompt,
        agent_id="spacedock:first-officer",
        extra_args=["--model", model, "--effort", effort, "--max-budget-usd", "5.00"],
        dispatch_budget=DispatchBudget(soft_s=60.0, hard_s=300.0, shutdown_grace_s=30.0),
    ) as w:
        impl_record = w.expect_dispatch_close(
            overall_timeout_s=IMPL_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="implementation",
            label="implementation dispatch close",
        )
        print(f"[OK] implementation dispatch closed in {impl_record.elapsed:.1f}s")

        validation_record = w.expect_dispatch_close(
            overall_timeout_s=VALIDATION_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="validation",
            label="validation dispatch close",
        )
        print(f"[OK] validation dispatch closed in {validation_record.elapsed:.1f}s")

        feedback_record = w.expect_dispatch_close(
            overall_timeout_s=FEEDBACK_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="implementation",
            label="feedback-cycle implementation dispatch close",
        )
        print(f"[OK] feedback-cycle implementation dispatch closed in {feedback_record.elapsed:.1f}s")

        w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)

    print("--- Phase 3: Validation ---")
    records = w.dispatch_records
    print(f"  dispatch records: {[(r.ensign_name, round(r.elapsed, 1)) for r in records]}")
    t.check("FO emitted exactly three ensign dispatches", len(records) == 3)
    t.check("all dispatches closed under the per-dispatch budget",
            all(r.elapsed <= PER_DISPATCH_BUDGET_S for r in records))

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
