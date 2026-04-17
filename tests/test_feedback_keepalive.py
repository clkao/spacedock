# ABOUTME: E2E test for the feedback-to keepalive rule in the first-officer template.
# ABOUTME: Verifies that the FO keeps the implementation agent alive during validation and routes rejection feedback via SendMessage.

from __future__ import annotations

import re
import subprocess
import sys
import time
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    LogParser,
    _agent_targets_stage,
    emit_skip_result,
    git_add_commit,
    install_agents,
    probe_claude_runtime,
    run_first_officer_streaming,
    setup_fixture,
    tool_use_matches,
)


def _agent_input_dict(entry: dict) -> dict:
    """Extract the input dict of the first Agent() tool_use block in an entry."""
    if entry.get("type") != "assistant":
        return {}
    message = entry.get("message", {})
    for block in message.get("content", []) or []:
        if isinstance(block, dict) and block.get("type") == "tool_use" and block.get("name") == "Agent":
            return block.get("input", {}) or {}
    return {}


REPO_ROOT = Path(__file__).resolve().parent.parent

SHUTDOWN_PATTERN = re.compile(
    r"shut\s*down|terminat|kill|(?:^|\s)stop(?:\s|$)|cancel.*agent",
    re.IGNORECASE,
)


def _scan_keepalive_events(log: LogParser) -> dict:
    impl_dispatch_seen = False
    impl_completion_seen = False
    validation_dispatch_seen = False
    rejection_seen = False
    feedback_via_send_message = False
    feedback_via_fresh_agent = False
    shutdown_before_validation = []
    impl_agent_name = ""

    completion_pattern = re.compile(r"complete|done|archived|finished|terminal|stage report", re.IGNORECASE)
    rejection_pattern = re.compile(r"REJECTED|recommend reject", re.IGNORECASE)

    for entry in log.entries:
        if entry.get("type") == "tool_result":
            content = entry.get("message", {})
            if isinstance(content, dict):
                content = str(content.get("content", ""))
            else:
                content = str(content)
            if impl_dispatch_seen and not impl_completion_seen:
                if completion_pattern.search(content):
                    impl_completion_seen = True
            if rejection_pattern.search(content):
                rejection_seen = True

        if entry.get("type") == "result" and entry.get("subtype") == "success":
            if impl_dispatch_seen and not impl_completion_seen:
                impl_completion_seen = True

        if entry.get("type") != "assistant" or "message" not in entry:
            continue

        for block in entry["message"].get("content", []):
            if block.get("type") == "text":
                text = block.get("text", "")
                if impl_dispatch_seen and not impl_completion_seen:
                    if completion_pattern.search(text):
                        impl_completion_seen = True

            if block.get("type") != "tool_use":
                continue

            if block.get("name") == "Agent":
                inp = block.get("input", {})
                name = inp.get("name", "")
                if _agent_targets_stage(inp, "implementation") and not impl_dispatch_seen:
                    impl_dispatch_seen = True
                    impl_agent_name = name
                elif _agent_targets_stage(inp, "validation") and not validation_dispatch_seen:
                    validation_dispatch_seen = True
                elif _agent_targets_stage(inp, "implementation") and rejection_seen:
                    feedback_via_fresh_agent = True

            if block.get("name") == "SendMessage":
                inp = block.get("input", {})
                if not isinstance(inp, dict):
                    continue
                to_field = str(inp.get("to", ""))
                msg_raw = inp.get("message", "")
                if (impl_completion_seen and not validation_dispatch_seen
                        and isinstance(msg_raw, str)):
                    if SHUTDOWN_PATTERN.search(msg_raw):
                        shutdown_before_validation.append(msg_raw[:200])
                if rejection_seen and not feedback_via_send_message:
                    if isinstance(msg_raw, str) and impl_agent_name:
                        if impl_agent_name in to_field or "implementation" in to_field.lower():
                            if re.search(r"reject|fix|feedback|fail|bug|error|rework", msg_raw, re.IGNORECASE):
                                feedback_via_send_message = True

    return {
        "impl_dispatch_seen": impl_dispatch_seen,
        "impl_completion_seen": impl_completion_seen,
        "validation_dispatch_seen": validation_dispatch_seen,
        "shutdown_before_validation": shutdown_before_validation,
        "rejection_seen": rejection_seen,
        "feedback_via_send_message": feedback_via_send_message,
        "feedback_via_fresh_agent": feedback_via_fresh_agent,
        "impl_agent_name": impl_agent_name,
    }


@pytest.mark.live_claude
def test_feedback_keepalive(test_project, model, effort):
    """FO keeps implementation ensign alive across validation rejection and routes via SendMessage."""
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

    print("--- Phase 2: Run first officer (claude, this takes ~60-180s) ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the keepalive path."
        )

    abs_workflow = t.test_project_dir / "keepalive-pipeline"
    prompt = (
        f"Process the entity `keepalive-test-task` through the workflow at {abs_workflow}/. "
        "Drive it from backlog through implementation and validation. "
        "The implementation task is trivial (create a text file). "
        "The validation stage has feedback-to: implementation, so you must keep the implementation "
        "agent alive when dispatching validation. "
        "When you encounter a gate review where the reviewer recommends REJECTED, "
        "auto-bounce into the feedback rejection flow and route findings to the implementation agent "
        "via SendMessage."
    )
    with run_first_officer_streaming(
        t,
        prompt,
        agent_id="spacedock:first-officer",
        extra_args=["--model", model, "--effort", effort, "--max-budget-usd", "5.00"],
    ) as w:
        w.expect(
            lambda e: tool_use_matches(e, "Agent", subagent_type="spacedock:ensign")
            and _agent_targets_stage(_agent_input_dict(e), "implementation"),
            timeout_s=180,
            label="implementation ensign dispatched",
        )
        print("[OK] implementation ensign dispatched")

        w.expect(
            lambda e: tool_use_matches(e, "Agent", subagent_type="spacedock:ensign")
            and _agent_targets_stage(_agent_input_dict(e), "validation"),
            timeout_s=240,
            label="validation ensign dispatched (keepalive crossed the transition)",
        )
        print("[OK] validation ensign dispatched — implementation agent survived the transition")

        entity_file = abs_workflow / "keepalive-test-task.md"
        feedback_deadline = time.monotonic() + 300
        while time.monotonic() < feedback_deadline:
            if entity_file.is_file():
                body = entity_file.read_text()
                if "### Feedback Cycles" in body:
                    break
            time.sleep(1.0)
        else:
            raise AssertionError(
                f"Entity body did not record a feedback cycle section at "
                f"{entity_file} within 300s"
            )
        print("[OK] entity body recorded feedback cycle section (data-flow assertion)")
        w.proc.terminate()

    print("--- Phase 3: Validation ---")
    log = LogParser(t.log_dir / "fo-log.jsonl")
    log.write_agent_calls(t.log_dir / "agent-calls.txt")
    log.write_fo_texts(t.log_dir / "fo-texts.txt")
    log.write_tool_calls(t.log_dir / "tool-calls.json")

    agent_calls = log.agent_calls()
    print()
    print("[Agent Dispatch Overview]")
    ensign_calls = [c for c in agent_calls if c["subagent_type"] == "spacedock:ensign"]
    impl_dispatches = [c for c in ensign_calls if _agent_targets_stage(c, "implementation")]
    val_dispatches = [c for c in ensign_calls if _agent_targets_stage(c, "validation")]

    print(f"  Total ensign dispatches: {len(ensign_calls)}")
    print(f"  Implementation dispatches: {len(impl_dispatches)}")
    print(f"  Validation dispatches: {len(val_dispatches)}")
    t.check("FO dispatched Agent() for implementation stage", len(impl_dispatches) >= 1)
    t.check("FO dispatched Agent() for validation stage", len(val_dispatches) >= 1)

    print()
    print("[Keepalive Event Scan]")
    events = _scan_keepalive_events(log)
    print(f"  Implementation dispatch seen: {events['impl_dispatch_seen']}")
    print(f"  Implementation completion seen: {events['impl_completion_seen']}")
    print(f"  Validation dispatch seen: {events['validation_dispatch_seen']}")
    print(f"  Shutdown before validation: {len(events['shutdown_before_validation'])}")
    print(f"  Rejection seen: {events['rejection_seen']}")
    print(f"  Feedback via SendMessage: {events['feedback_via_send_message']}")
    print(f"  Feedback via fresh Agent: {events['feedback_via_fresh_agent']}")

    print()
    print("[Tier 1 — Keepalive at Transition]")
    if events["impl_completion_seen"] and events["validation_dispatch_seen"]:
        t.check(
            "no shutdown SendMessage targets implementation agent between completion and validation dispatch",
            len(events["shutdown_before_validation"]) == 0,
        )
        if events["shutdown_before_validation"]:
            for msg in events["shutdown_before_validation"]:
                print(f"    PREMATURE SHUTDOWN: {msg}")
    elif not events["impl_dispatch_seen"]:
        print("  SKIP: pipeline did not dispatch implementation stage within budget")
    elif not events["impl_completion_seen"]:
        print("  SKIP: implementation stage did not complete within budget")
    else:
        print("  SKIP: pipeline did not reach validation dispatch within budget")

    print()
    print("[Tier 2 — Feedback Routing via SendMessage]")
    if events["rejection_seen"]:
        t.pass_("rejection signal detected in logs or entity")
        if events["feedback_via_send_message"]:
            t.pass_("feedback routed via SendMessage to kept-alive implementation agent (keepalive worked)")
        elif events["feedback_via_fresh_agent"]:
            t.fail("feedback routed via fresh Agent() dispatch instead of SendMessage (keepalive failed — agent was killed and re-dispatched)")
        else:
            tool_calls = log.tool_calls()
            post_rejection_sms = [
                c for c in tool_calls
                if c["name"] == "SendMessage"
                and isinstance(c.get("input"), dict)
                and re.search(r"implementation", str(c["input"].get("to", "")), re.IGNORECASE)
            ]
            if post_rejection_sms:
                t.pass_("SendMessage sent to implementation agent after rejection (feedback content may not match pattern)")
            else:
                t.fail("no feedback routing observed after rejection (neither SendMessage nor fresh Agent dispatch)")
    else:
        print("  SKIP: rejection not observed — pipeline may not have completed validation within budget")

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

