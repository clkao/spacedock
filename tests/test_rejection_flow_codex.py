# ABOUTME: E2E test for the validation rejection flow (codex runtime).
# ABOUTME: Codex-only sibling of test_rejection_flow.py; kept on LogParser/send_input semantics; skipped pending #141 on the codex side.

from __future__ import annotations

import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    CodexLogParser,
    git_add_commit,
    rejection_follow_up_observed,
    rejection_signal_present,
    run_codex_first_officer,
    setup_fixture,
)


def _codex_rejection_flow_milestones(log: CodexLogParser) -> dict[str, bool]:
    milestones = {
        "boot_status": False, "implementation_dispatch": False, "implementation_wait": False,
        "implementation_completed": False, "validation_dispatch": False, "validation_wait": False,
        "validation_completed": False, "rejection_seen": False, "follow_up_seen": False,
        "final_response": False,
    }
    for raw_line in log.raw_lines:
        try:
            entry = json.loads(raw_line)
        except json.JSONDecodeError:
            continue
        item = entry.get("item", {})
        if not isinstance(item, dict):
            continue
        item_type = item.get("type")
        if item_type == "command_execution":
            command = item.get("command") or ""
            output = item.get("aggregated_output") or ""
            if "--boot" in command:
                milestones["boot_status"] = True
            if "status=implementation" in command or "entering implementation" in command:
                milestones["implementation_dispatch"] = True
                if milestones["rejection_seen"]:
                    milestones["follow_up_seen"] = True
            if "status=validation" in command or "entering validation" in command:
                milestones["validation_dispatch"] = True
            if re.search(r"REJECTED|recommend reject|failing test|Expected 5, got -1", output, re.IGNORECASE):
                milestones["rejection_seen"] = True
        if item_type == "collab_tool_call":
            tool = item.get("tool")
            prompt = item.get("prompt") or ""
            agent_states = item.get("agents_states") or {}
            state_text = json.dumps(agent_states)
            if tool == "spawn_agent":
                if "stage_name: `implementation`" in prompt or "stage_name: implementation" in prompt:
                    milestones["implementation_dispatch"] = True
                    if milestones["rejection_seen"]:
                        milestones["follow_up_seen"] = True
                if "stage_name: `validation`" in prompt or "stage_name: validation" in prompt:
                    milestones["validation_dispatch"] = True
            elif tool == "wait":
                if re.search(r"implementation", prompt, re.IGNORECASE) or re.search(r"implementation", state_text, re.IGNORECASE):
                    milestones["implementation_wait"] = True
                if re.search(r"validation", prompt, re.IGNORECASE) or re.search(r"validation", state_text, re.IGNORECASE):
                    milestones["validation_wait"] = True
                for state in agent_states.values():
                    if not isinstance(state, dict) or state.get("status") != "completed":
                        continue
                    message = str(state.get("message") or "")
                    if re.search(r"implementation", message, re.IGNORECASE):
                        milestones["implementation_completed"] = True
                    if re.search(r"validation", message, re.IGNORECASE):
                        milestones["validation_completed"] = True
                if re.search(r"REJECTED|recommend reject|failing test|Expected 5, got -1", state_text, re.IGNORECASE):
                    milestones["rejection_seen"] = True
            elif tool == "send_input":
                milestones["follow_up_seen"] = True
        if item_type == "agent_message":
            text = item.get("text") or ""
            if re.search(r"REJECTED|recommend reject|failing test|Expected 5, got -1", text, re.IGNORECASE):
                milestones["rejection_seen"] = True
            if re.search(r"follow-up|feedback-to|feedback route|route findings|fix", text, re.IGNORECASE):
                milestones["follow_up_seen"] = True

    if log.completed_agent_messages():
        milestones["final_response"] = True
    return milestones


def _codex_rejection_follow_up_order(log: CodexLogParser):
    rejection_index = None
    follow_up_index = None
    rejection_pattern = r"REJECTED|recommend reject|failing test|Expected 5, got -1"
    for idx, raw_line in enumerate(log.raw_lines):
        try:
            entry = json.loads(raw_line)
        except json.JSONDecodeError:
            continue
        item = entry.get("item", {})
        if not isinstance(item, dict):
            continue
        item_type = item.get("type")
        item_text = json.dumps(item)
        if rejection_index is None and re.search(rejection_pattern, item_text, re.IGNORECASE):
            rejection_index = idx
        if follow_up_index is not None:
            continue
        if item_type == "collab_tool_call":
            tool = item.get("tool")
            prompt = item.get("prompt") or ""
            if tool == "send_input":
                follow_up_index = idx
            elif tool in {"spawn", "spawn_agent"} and re.search(r"stage_name:\s*implementation", prompt, re.IGNORECASE):
                follow_up_index = idx
        elif item_type == "agent_message":
            text = item.get("text") or ""
            if re.search(r"follow-up|feedback-to|route findings|fix", text, re.IGNORECASE):
                follow_up_index = idx
    return rejection_index, follow_up_index


def _codex_rejection_flow_stop_ready(log_path: Path) -> bool:
    log = CodexLogParser(log_path)
    milestones = _codex_rejection_flow_milestones(log)
    return (
        milestones["final_response"]
        and milestones["follow_up_seen"]
        and milestones["implementation_dispatch"]
    )


@pytest.mark.skip(reason="pending #141 — reviewer keepalive across feedback cycles — codex reuse via send_input has separate semantics; #210 split only, un-skip tracked separately")
@pytest.mark.live_codex
def test_rejection_flow_codex(test_project, model, effort):
    """Rejected validation triggers a fix dispatch via relay protocol (codex)."""
    t = test_project
    agent_id = "spacedock:first-officer"

    print("--- Phase 1: Set up test project from fixture ---")
    fixture_dir = t.repo_root / "tests" / "fixtures" / "rejection-flow"
    setup_fixture(t, "rejection-flow", "rejection-pipeline")

    shutil.copy2(fixture_dir / "math_ops.py", t.test_project_dir)
    tests_dir = t.test_project_dir / "tests"
    tests_dir.mkdir(exist_ok=True)
    shutil.copy2(fixture_dir / "tests" / "test_add.py", tests_dir)
    git_add_commit(t.test_project_dir, "setup: rejection flow fixture with buggy implementation")

    status_cmd = ["python3", str(t.repo_root / "skills" / "commission" / "bin" / "status"),
                  "--workflow-dir", "rejection-pipeline"]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    status_result = subprocess.run(
        status_cmd + ["--next"], capture_output=True, text=True, cwd=t.test_project_dir,
    )
    t.check("status --next detects dispatchable entity",
            "buggy-add-task" in status_result.stdout)
    print()

    print("--- Phase 2: Run first officer (codex) ---")
    fo_exit = run_codex_first_officer(
        t, "rejection-pipeline",
        agent_id=agent_id,
        run_goal="Process only the entity `buggy-add-task`.",
        timeout_s=420,
        stop_checker=_codex_rejection_flow_stop_ready,
    )

    print("--- Phase 3: Validation ---")
    log = CodexLogParser(t.log_dir / "codex-fo-log.txt")
    log.write_text(t.log_dir / "codex-fo-text.txt")
    fo_text = log.full_text()
    worker_messages = "\n".join(log.completed_agent_messages())
    milestones = _codex_rejection_flow_milestones(log)

    print()
    print("[Rejection Flow Behavior]")
    entity_main = t.test_project_dir / "rejection-pipeline" / "buggy-add-task.md"
    worktrees_dir = t.test_project_dir / ".worktrees"

    t.check("at least one worker completed", bool(worker_messages.strip()))

    t.check(
        "reviewer stage report contains REJECTED recommendation",
        rejection_signal_present("rejection-pipeline", "buggy-add-task", entity_main, worktrees_dir, worker_messages, fo_text),
    )

    spawn_count = log.spawn_count()
    print("  Codex milestones:")
    for key, reached in milestones.items():
        print(f"    {key}: {reached}")
    t.check(
        "Codex launcher reached the bounded rejection-flow stop condition",
        fo_exit == 0 or milestones["final_response"],
    )
    t.check(
        "multiple worker dispatches occurred",
        spawn_count >= 2 or bool(re.search(r"validation|implementation", worker_messages, re.IGNORECASE)),
    )
    t.check(
        "bounded Codex run waits for the validation result before gate handling",
        milestones["validation_dispatch"] and milestones["validation_wait"] and milestones["validation_completed"],
    )
    t.check(
        "follow-up work after rejection was observable",
        rejection_follow_up_observed("rejection-pipeline", "buggy-add-task", worktrees_dir, worker_messages, fo_text),
    )
    rejection_index, follow_up_index = _codex_rejection_follow_up_order(log)
    t.check(
        "rejection follow-up happens after rejection is observed",
        rejection_index is not None and follow_up_index is not None and follow_up_index > rejection_index,
    )
    worktree_candidates = [
        worktrees_dir / "ensign-buggy-add-task",
        worktrees_dir / "spacedock-ensign-buggy-add-task",
    ]
    safe_worktree_mentions = [str(path) for path in worktree_candidates]
    t.check(
        "worker uses safe worktree key",
        any(path.is_dir() for path in worktree_candidates)
        or any(path_text in fo_text for path_text in safe_worktree_mentions)
        or any(path_text in worker_messages for path_text in safe_worktree_mentions),
    )
    t.check(
        "logical packaged id does not leak into worktree path",
        all("spacedock:ensign" not in path.as_posix() for path in worktree_candidates if path.is_dir()),
    )
    branches = subprocess.run(
        ["git", "branch", "--list"],
        capture_output=True, text=True, cwd=t.test_project_dir, check=True,
    ).stdout
    t.check(
        "validation dispatch uses the shared safe branch name",
        "ensign/buggy-add-task" in branches or "spacedock-ensign/buggy-add-task" in branches,
    )
    t.check("boot status ran", milestones["boot_status"])
    t.check("implementation dispatch was observed", milestones["implementation_dispatch"])
    t.check("validation dispatch was observed", milestones["validation_dispatch"])
    t.check(
        "Codex run reached a concrete rejection-flow milestone before stop",
        milestones["validation_wait"] or milestones["validation_completed"] or milestones["rejection_seen"] or milestones["follow_up_seen"],
    )

    t.finish()
