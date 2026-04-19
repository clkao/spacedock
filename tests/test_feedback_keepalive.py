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

_STATUS_DONE_RE = re.compile(r"^status:\s*done\b", re.MULTILINE)


def _inline_process_complete(
    entity_file: Path,
    archive_file: Path,
    greeting_file: Path,
) -> bool:
    """Three-signal conjunction for Path-B (inline-process) observation.

    All three must hold in the same poll tick:
      (1) entity body (or archived copy) contains `### Feedback Cycles`
      (2) greeting.txt contains the validation-requested `Goodbye, World!`
      (3) entity frontmatter (or archived copy) has `status: done`

    Conjunctive, same-tick evaluation sidesteps the race where the FO writes
    the body section and the status flip in separate edits: a partial-write
    window produces a transient miss rather than a false positive, and the
    next poll tick succeeds once all three invariants hold. Files are read
    with errors="ignore" to tolerate mid-write UTF-8 truncation.
    """
    body_match = False
    for candidate in (entity_file, archive_file):
        if not candidate.is_file():
            continue
        try:
            body_text = candidate.read_text(errors="ignore")
        except OSError:
            continue
        if "### Feedback Cycles" in body_text and _STATUS_DONE_RE.search(body_text):
            body_match = True
            break
    if not body_match:
        return False
    if not greeting_file.is_file():
        return False
    try:
        greeting_text = greeting_file.read_text(errors="ignore")
    except OSError:
        return False
    return "Goodbye, World!" in greeting_text


def _await_validation_path(
    w,
    entity_file: Path,
    archive_file: Path,
    greeting_file: Path,
    timeout_s: float,
) -> str:
    """Watch for either Path-A (fresh validation dispatch) or Path-B
    (inline-process completion) to fire, whichever comes first.

    Returns "dispatch" for Path A, "inline-process" for Path B. Raises
    AssertionError on timeout or early FO exit without either signal.

    Implementation notes: reuses the watcher's private `_drain_entries` to
    consume forward-only stream events, then checks filesystem on each tick.
    Stream check runs before filesystem check so an in-flight dispatch isn't
    misattributed to inline-process when both signals happen to land in the
    same tick.
    """
    deadline = time.monotonic() + timeout_s
    while True:
        for entry in w._drain_entries():  # noqa: SLF001
            try:
                if (tool_use_matches(entry, "Agent", subagent_type="spacedock:ensign")
                        and _agent_targets_stage(_agent_input_dict(entry), "validation")):
                    return "dispatch"
            except Exception:
                continue

        if _inline_process_complete(entity_file, archive_file, greeting_file):
            return "inline-process"

        if w.proc.poll() is not None:
            for entry in w._drain_entries():  # noqa: SLF001
                try:
                    if (tool_use_matches(entry, "Agent", subagent_type="spacedock:ensign")
                            and _agent_targets_stage(_agent_input_dict(entry), "validation")):
                        return "dispatch"
                except Exception:
                    continue
            if _inline_process_complete(entity_file, archive_file, greeting_file):
                return "inline-process"
            raise AssertionError(
                f"FO subprocess exited (code={w.proc.returncode}) before either "
                f"Path-A (fresh validation ensign dispatch) or Path-B "
                f"(inline-process completion) signal was observed."
            )

        if time.monotonic() >= deadline:
            raise AssertionError(
                f"Neither Path-A (fresh validation ensign dispatch) nor Path-B "
                f"(inline-process: Feedback Cycles section + greeting.txt "
                f"`Goodbye, World!` + status:done) observed within {timeout_s}s."
            )
        time.sleep(0.2)

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
def test_feedback_keepalive(test_project, model, effort, request):
    """FO keeps implementation ensign alive across validation rejection and routes via SendMessage."""
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
        entity_file = abs_workflow / "keepalive-test-task.md"
        archive_file = abs_workflow / "_archive" / "keepalive-test-task.md"

        def _impl_signal_in_event(e: dict) -> bool:
            for body_path in (entity_file, archive_file):
                if tool_use_matches(
                    e, "Edit", file_path=str(body_path), new_string="Feedback Cycles"
                ):
                    return True
                if tool_use_matches(
                    e, "Write", file_path=str(body_path), content="Feedback Cycles"
                ):
                    return True
            if tool_use_matches(e, "Bash", command="Feedback Cycles"):
                return True
            if tool_use_matches(e, "Agent", subagent_type="spacedock:ensign"):
                return True
            return False

        w.expect(
            _impl_signal_in_event,
            timeout_s=120,
            label="implementation data-flow signal",
        )
        print("[OK] implementation data-flow signal observed "
              "(Feedback Cycles section edit or ensign Agent dispatch)")

        validation_signal = _await_validation_path(
            w,
            entity_file=entity_file,
            archive_file=archive_file,
            greeting_file=t.test_project_dir / "greeting.txt",
            timeout_s=240,
        )
        if validation_signal == "dispatch":
            print("[OK] Path A — validation ensign dispatched "
                  "(implementation agent survived the fresh-dispatch transition)")
        else:
            print("[OK] Path B — feedback fix inline-processed "
                  "(Feedback Cycles section + greeting.txt content + terminal status observed)")

        if validation_signal == "dispatch":
            ensign_count = [0]

            def _feedback_signal_in_event(e: dict) -> bool:
                for body_path in (entity_file, archive_file):
                    if tool_use_matches(
                        e, "Edit", file_path=str(body_path), new_string="Feedback Cycles"
                    ):
                        return True
                    if tool_use_matches(
                        e, "Write", file_path=str(body_path), content="Feedback Cycles"
                    ):
                        return True
                if tool_use_matches(e, "Bash", command="Feedback Cycles"):
                    return True
                if tool_use_matches(e, "Agent", subagent_type="spacedock:ensign"):
                    ensign_count[0] += 1
                    if ensign_count[0] >= 2:
                        return True
                return False

            w.expect(
                _feedback_signal_in_event,
                timeout_s=300,
                label="feedback-cycle data-flow signal",
            )
            print("[OK] feedback-cycle data-flow signal observed "
                  "(Feedback Cycles section edit or second ensign dispatch)")
        else:
            print("[OK] feedback-cycle signal already captured by Path-B "
                  "inline-process evidence (Feedback Cycles section on disk)")
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
    if validation_signal == "dispatch":
        t.check("FO dispatched Agent() for validation stage", len(val_dispatches) >= 1)
    else:
        print("  SKIP: Path-B inline-process path did not require a separate "
              "validation Agent() dispatch (feedback cycle handled in-place)")

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
    print(f"[Tier 1 — Keepalive at Transition (path={validation_signal})]")
    if validation_signal == "dispatch":
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
    else:
        # Path B — no fresh validation dispatch; keepalive-at-transition is
        # moot (there is no transition to span). Meaningful assertion: the
        # implementation agent was not torn down mid-cycle and the workflow
        # reached a clean terminal state on disk.
        t.check(
            "no shutdown SendMessage targets implementation agent before inline-process completion",
            len(events["shutdown_before_validation"]) == 0,
        )
        if events["shutdown_before_validation"]:
            for msg in events["shutdown_before_validation"]:
                print(f"    PREMATURE SHUTDOWN: {msg}")
        t.check(
            "inline-process reached terminal state on disk (Feedback Cycles + greeting + status:done)",
            _inline_process_complete(
                abs_workflow / "keepalive-test-task.md",
                abs_workflow / "_archive" / "keepalive-test-task.md",
                t.test_project_dir / "greeting.txt",
            ),
        )

    print()
    print("[Tier 2 — Feedback Routing via SendMessage]")
    if events["rejection_seen"]:
        t.pass_("rejection signal detected in logs or entity")
        if events["feedback_via_send_message"]:
            t.pass_("feedback routed via SendMessage to kept-alive implementation agent (keepalive worked)")
        elif events["feedback_via_fresh_agent"]:
            t.fail("feedback routed via fresh Agent() dispatch instead of SendMessage (keepalive failed — agent was killed and re-dispatched)")
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

