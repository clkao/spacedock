# ABOUTME: E2E and static tests for ensign reuse dispatch behavior in the FO template.
# ABOUTME: Runs the reuse-pipeline fixture through the FO and verifies Agent/SendMessage patterns.

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    LogParser,
    assembled_agent_content,
    git_add_commit,
    install_agents,
    read_entity_frontmatter,
    run_first_officer,
    setup_fixture,
)


REPO_ROOT = Path(__file__).resolve().parent.parent


@pytest.mark.xfail(reason="pending #154 — test assertions target `agents/first-officer.md` but post-#085 skill-preload the content lives in the skill/references layer", strict=False)
@pytest.mark.live_claude
def test_reuse_dispatch(test_project, model, effort):
    """Ensign reuse uses SendMessage; fresh: true forces new Agent dispatch."""
    t = test_project

    print("--- Phase 1: Set up test project from fixture ---")
    setup_fixture(t, "reuse-pipeline", "reuse-pipeline")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: reuse dispatch fixture")

    status_cmd = [
        "python3",
        str(t.repo_root / "skills" / "commission" / "bin" / "status"),
        "--workflow-dir", "reuse-pipeline",
    ]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    status_result = subprocess.run(
        status_cmd + ["--next"],
        capture_output=True, text=True, cwd=t.test_project_dir,
    )
    t.check("status --next detects dispatchable entity",
            "reuse-test-task" in status_result.stdout)
    print()

    print("--- Phase 2: Run first officer (claude) ---")
    abs_workflow = t.test_project_dir / "reuse-pipeline"
    fo_exit = run_first_officer(
        t,
        (
            f"Process the entity `reuse-test-task` through the workflow at {abs_workflow}/. "
            "Drive it from backlog through analysis, implementation, and validation to done. "
            "When analysis completes and the next stage (implementation) meets reuse conditions "
            "(same worktree mode, no fresh:true, teams available), reuse the agent via SendMessage "
            "instead of dispatching fresh. "
            "When you reach the validation gate, auto-approve if PASSED."
        ),
        agent_id="spacedock:first-officer",
        extra_args=[
            "--model", model,
            "--effort", effort,
            "--max-budget-usd", "5.00",
        ],
    )
    if fo_exit != 0:
        print("  (may be expected — budget cap or gate hold)")

    print("--- Phase 3: Validation ---")
    log = LogParser(t.log_dir / "fo-log.jsonl")
    log.write_agent_calls(t.log_dir / "agent-calls.txt")
    log.write_fo_texts(t.log_dir / "fo-texts.txt")
    log.write_tool_calls(t.log_dir / "tool-calls.json")

    agent_calls = log.agent_calls()
    tool_calls = log.tool_calls()

    print()
    print("[Agent Dispatch Pattern]")
    ensign_calls = [c for c in agent_calls if c["subagent_type"] == "spacedock:ensign"]
    analysis_dispatches = [c for c in ensign_calls if "analysis" in c["name"]]
    implementation_dispatches = [c for c in ensign_calls if "implementation" in c["name"]]
    validation_dispatches = [c for c in ensign_calls if "validation" in c["name"]]

    t.check("FO dispatched Agent() for analysis stage (initial dispatch)",
            len(analysis_dispatches) >= 1)

    if len(implementation_dispatches) == 0:
        t.pass_("FO skipped Agent() for implementation (reused via SendMessage)")
    else:
        print(f"  INFO: FO dispatched {len(implementation_dispatches)} Agent() call(s) for implementation")

    if len(validation_dispatches) >= 1:
        t.pass_("FO dispatched Agent() for validation stage (fresh: true forces fresh dispatch)")
    elif len(ensign_calls) >= 2:
        t.fail("FO reached validation but did not dispatch Agent() (fresh: true should force fresh)")
    else:
        print("  SKIP: pipeline did not progress to validation stage within budget")

    print()
    print("[SendMessage Reuse Pattern]")
    send_messages = [
        c for c in tool_calls
        if c["name"] == "SendMessage" and isinstance(c.get("input"), dict)
    ]
    reuse_messages = [
        m for m in send_messages
        if re.search(
            r"implementation|advancing to next stage",
            str(m.get("input", {}).get("message", "")),
            re.IGNORECASE,
        )
    ]
    t.check("FO sent SendMessage for reuse dispatch (analysis -> implementation transition)",
            len(reuse_messages) >= 1)
    if reuse_messages:
        msg_content = str(reuse_messages[0].get("input", {}).get("message", ""))
        t.check("reuse SendMessage contains stage definition",
                "Stage definition" in msg_content or "implementation" in msg_content.lower())

    print()
    print("[Entity Outcome]")
    entity_main = t.test_project_dir / "reuse-pipeline" / "reuse-test-task.md"
    archive_path = t.test_project_dir / "reuse-pipeline" / "_archive" / "reuse-test-task.md"
    if archive_path.is_file():
        t.pass_("entity archived (reached terminal stage)")
    elif entity_main.is_file():
        fm = read_entity_frontmatter(entity_main)
        status_val = fm.get("status", "?")
        if status_val == "done":
            t.pass_(f"entity reached terminal stage (status: {status_val})")
        elif status_val in ("validation", "implementation"):
            print(f"  SKIP: entity at {status_val} — FO may not have completed full cycle within budget")
        else:
            t.fail(f"entity did not reach expected stage (status: {status_val})")
    else:
        t.fail("entity file not found in active or archive location")

    print()
    print("[Static Template Checks]")
    core = (REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md").read_text()
    runtime_ref = (REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md").read_text()
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
            "Dispatch a worker for the stage" in core and "Dispatch a fresh worker" not in core)
    t.check("runtime clarifies SendMessage for reuse only",
            "NEVER use SendMessage to dispatch" not in runtime_ref
            and bool(re.search(r"SendMessage.*completion path|completion path.*SendMessage", runtime_ref, re.IGNORECASE)))

    t.finish()

