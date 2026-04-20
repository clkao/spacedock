# ABOUTME: Offline unit tests for per-ensign-dispatch soft/hard budget tracking (#203 cycle-7).
# ABOUTME: Covers DispatchBudget state machine: open, soft-warn, hard-trip, grace-kill, cooperative-close.

from __future__ import annotations

import json
import sys
import threading
import time
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    DispatchBudget,
    DispatchHardTimeout,
    DispatchRecord,
    FOStreamWatcher,
    StepFailure,
    StepTimeout,
)


class _FakeProc:
    """Minimal stand-in for subprocess.Popen used by FOStreamWatcher."""

    def __init__(self, returncode: int | None = None):
        self.returncode = returncode
        self.terminated = False
        self.killed = False

    def poll(self):
        return self.returncode

    def set_exited(self, returncode: int) -> None:
        self.returncode = returncode

    def terminate(self) -> None:
        self.terminated = True
        self.returncode = -15

    def kill(self) -> None:
        self.killed = True
        self.returncode = -9

    def wait(self, timeout=None):
        return self.returncode


def _write_line(path: Path, obj: dict) -> None:
    with path.open("a") as f:
        f.write(json.dumps(obj) + "\n")


def _agent_dispatch(tool_use_id: str, description: str = "impl-task") -> dict:
    return {
        "type": "assistant",
        "message": {
            "model": "claude-opus-4-7",
            "content": [
                {
                    "type": "tool_use",
                    "id": tool_use_id,
                    "name": "Agent",
                    "input": {
                        "subagent_type": "spacedock:ensign",
                        "description": description,
                        "prompt": "do the thing",
                    },
                }
            ],
        },
    }


def _agent_tool_result(tool_use_id: str, text: str = "Done: stage complete.") -> dict:
    """User tool_result entry for the given Agent tool_use_id (close anchor)."""
    return {
        "type": "user",
        "message": {
            "content": [
                {
                    "type": "tool_result",
                    "tool_use_id": tool_use_id,
                    "content": [{"type": "text", "text": text}],
                }
            ],
        },
    }


def _teams_spawn_ack(tool_use_id: str, agent_id: str = "spacedock-ensign-x") -> dict:
    """Teams-mode spawn-ack tool_result — fires immediately after Agent() spawn.

    This is NOT a completion signal and MUST NOT close a tracked dispatch.
    """
    return {
        "type": "user",
        "message": {
            "content": [
                {
                    "type": "tool_result",
                    "tool_use_id": tool_use_id,
                    "content": [
                        {
                            "type": "text",
                            "text": (
                                f"Spawned successfully.\n"
                                f"agent_id: {agent_id}\n"
                                f"name: {agent_id}"
                            ),
                        }
                    ],
                }
            ],
        },
    }


def _task_notification_completed(tool_use_id: str) -> dict:
    """Teams-mode task_notification with status=completed (real completion signal)."""
    return {
        "type": "system",
        "subtype": "task_notification",
        "tool_use_id": tool_use_id,
        "status": "completed",
    }


def _send_message(to: str = "team-lead", body: str = "Done: impl-task completed.") -> dict:
    """Teams-mode SendMessage — NOT a close anchor; only tool_result closes."""
    return {
        "type": "assistant",
        "message": {
            "model": "claude-opus-4-7",
            "content": [
                {
                    "type": "tool_use",
                    "id": "sm-1",
                    "name": "SendMessage",
                    "input": {"to": to, "message": body},
                }
            ],
        },
    }


def _make_watcher(tmp_path: Path, budget: DispatchBudget | None = None) -> tuple[FOStreamWatcher, _FakeProc, Path]:
    log = tmp_path / "fo.jsonl"
    log.touch()
    proc = _FakeProc()
    watcher = FOStreamWatcher(log, proc, dispatch_budget=budget or DispatchBudget())
    return watcher, proc, log


def test_dispatch_close_under_soft_budget_emits_no_warning(tmp_path, capsys):
    """Dispatch that closes under the soft budget MUST NOT emit a warning."""
    budget = DispatchBudget(soft_s=10.0, hard_s=30.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))
    _write_line(log, _agent_tool_result("du_1"))

    # Two polls: first registers dispatch, second sees the close.
    watcher._drain_entries()
    watcher._drain_entries()

    captured = capsys.readouterr()
    assert "soft budget" not in captured.out
    assert "dispatch_hard" not in captured.out
    assert watcher._open_dispatches == {}


def test_dispatch_exceeds_soft_emits_structured_warning(tmp_path, capsys):
    """Dispatch that exceeds soft-but-not-hard emits a structured warning, no kill."""
    budget = DispatchBudget(soft_s=0.05, hard_s=10.0, shutdown_grace_s=5.0)
    watcher, proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="slow-ensign"))
    watcher._drain_entries()

    time.sleep(0.15)
    watcher._drain_entries()

    captured = capsys.readouterr()
    assert "slow-ensign" in captured.out
    assert "soft budget" in captured.out
    assert "0.05s" in captured.out
    assert proc.terminated is False and proc.killed is False
    dispatch = next(iter(watcher._open_dispatches.values()))
    assert dispatch.warned is True


def test_dispatch_exceeds_hard_triggers_cooperative_shutdown_state(tmp_path, capsys):
    """Dispatch past hard budget transitions to SHUTDOWN_REQUESTING, no kill yet."""
    budget = DispatchBudget(soft_s=0.01, hard_s=0.05, shutdown_grace_s=10.0)
    watcher, proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))
    watcher._drain_entries()

    time.sleep(0.1)
    watcher._drain_entries()

    captured = capsys.readouterr()
    assert "soft budget" in captured.out
    assert "cooperative shutdown" in captured.out
    assert proc.terminated is False and proc.killed is False
    dispatch = next(iter(watcher._open_dispatches.values()))
    assert dispatch.shutdown_requested_at is not None


def test_dispatch_hard_plus_grace_expiry_kills_and_raises(tmp_path):
    """After hard + grace expires, watcher kills proc and raises DispatchHardTimeout."""
    budget = DispatchBudget(soft_s=0.01, hard_s=0.05, shutdown_grace_s=0.05)
    watcher, proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="hung-ensign"))
    watcher._drain_entries()

    time.sleep(0.1)
    # First re-poll: transitions to SHUTDOWN_REQUESTING.
    watcher._drain_entries()
    time.sleep(0.1)
    # Second re-poll after grace: kills proc + raises.
    with pytest.raises(DispatchHardTimeout) as excinfo:
        watcher._drain_entries()

    assert excinfo.value.ensign_name == "hung-ensign"
    assert excinfo.value.elapsed >= 0.05
    assert "0.05s hard budget" in str(excinfo.value)
    assert proc.terminated is True


def test_dispatch_closes_during_shutdown_grace_no_kill(tmp_path, capsys):
    """If the ensign's Done: arrives during the shutdown grace window, do NOT kill."""
    budget = DispatchBudget(soft_s=0.01, hard_s=0.05, shutdown_grace_s=5.0)
    watcher, proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))
    watcher._drain_entries()

    time.sleep(0.1)
    # Trigger shutdown-requesting transition.
    watcher._drain_entries()
    # Ensign finally closes cleanly inside the grace window.
    _write_line(log, _agent_tool_result("du_1"))
    watcher._drain_entries()

    assert proc.terminated is False and proc.killed is False
    assert watcher._open_dispatches == {}


def test_non_ensign_agent_dispatch_is_ignored(tmp_path, capsys):
    """Agent() calls that are NOT subagent_type=spacedock:ensign do not start tracking."""
    budget = DispatchBudget(soft_s=0.01, hard_s=0.05, shutdown_grace_s=5.0)
    watcher, proc, log = _make_watcher(tmp_path, budget)

    # Agent call with a different subagent_type — must be ignored.
    _write_line(
        log,
        {
            "type": "assistant",
            "message": {
                "model": "claude-opus-4-7",
                "content": [
                    {
                        "type": "tool_use",
                        "id": "du_x",
                        "name": "Agent",
                        "input": {
                            "subagent_type": "echo-agent",
                            "description": "echo",
                        },
                    }
                ],
            },
        },
    )
    watcher._drain_entries()
    time.sleep(0.1)
    watcher._drain_entries()

    captured = capsys.readouterr()
    assert "soft budget" not in captured.out
    assert watcher._open_dispatches == {}


def test_send_message_alone_does_not_close_dispatch(tmp_path):
    """SendMessage from the FO (even with 'Done:' body) is NOT a close anchor.

    Only the matching user tool_result for the Agent tool_use_id closes a
    dispatch. The close anchor is identical in bare and teams mode.
    """
    budget = DispatchBudget(soft_s=10.0, hard_s=30.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))
    _write_line(log, _send_message(body="Done: just a sendmessage, not a close."))
    watcher._drain_entries()

    assert len(watcher._open_dispatches) == 1


def test_tool_result_for_unrelated_id_does_not_close(tmp_path):
    """A tool_result whose tool_use_id doesn't match any tracked Agent is ignored."""
    budget = DispatchBudget(soft_s=10.0, hard_s=30.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))
    _write_line(log, _agent_tool_result("some-other-id"))
    watcher._drain_entries()

    assert len(watcher._open_dispatches) == 1


def test_expect_dispatch_close_returns_record_with_elapsed(tmp_path):
    """expect_dispatch_close returns a DispatchRecord for the closed dispatch."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="impl-dispatch"))

    def _close_soon():
        time.sleep(0.2)
        _write_line(log, _agent_tool_result("du_1"))

    threading.Thread(target=_close_soon, daemon=True).start()

    record = watcher.expect_dispatch_close(overall_timeout_s=2.0, label="impl close")
    assert isinstance(record, DispatchRecord)
    assert record.ensign_name == "impl-dispatch"
    assert record.elapsed >= 0.2
    assert watcher.dispatch_records[-1] is record


def test_expect_dispatch_close_raises_step_timeout_on_budget_miss(tmp_path):
    """expect_dispatch_close raises StepTimeout when the dispatch never closes."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="slow"))

    with pytest.raises(StepTimeout) as excinfo:
        watcher.expect_dispatch_close(overall_timeout_s=0.4, label="slow close")
    assert excinfo.value.label == "slow close"
    assert "slow" in str(excinfo.value)


def test_expect_dispatch_close_raises_step_failure_on_early_exit(tmp_path):
    """expect_dispatch_close raises StepFailure if proc exits before close."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))

    def _exit_soon():
        time.sleep(0.2)
        proc.set_exited(1)

    threading.Thread(target=_exit_soon, daemon=True).start()

    with pytest.raises(StepFailure) as excinfo:
        watcher.expect_dispatch_close(overall_timeout_s=5.0, label="proc-died close")
    assert excinfo.value.exit_code == 1
    assert excinfo.value.label == "proc-died close"


def test_expect_dispatch_close_name_match_skips_other_dispatches(tmp_path):
    """When ensign_name is given, expect_dispatch_close only closes on a name match."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="impl-first"))
    _write_line(log, _agent_dispatch("du_2", description="validation-second"))

    def _close_first_then_second():
        time.sleep(0.1)
        _write_line(log, _agent_tool_result("du_1"))
        time.sleep(0.2)
        _write_line(log, _agent_tool_result("du_2"))

    threading.Thread(target=_close_first_then_second, daemon=True).start()

    record = watcher.expect_dispatch_close(
        overall_timeout_s=2.0, ensign_name="validation", label="validation close"
    )
    assert record.ensign_name == "validation-second"
    assert len(watcher.dispatch_records) == 2


def test_expect_dispatch_close_name_match_is_case_insensitive(tmp_path):
    """Substring match in expect_dispatch_close ignores case on both sides."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="Implementation stage"))
    _write_line(log, _agent_dispatch("du_2", description="Dispatch implementation"))

    def _close_first():
        time.sleep(0.1)
        _write_line(log, _agent_tool_result("du_1"))

    threading.Thread(target=_close_first, daemon=True).start()

    record = watcher.expect_dispatch_close(
        overall_timeout_s=2.0, ensign_name="implementation", label="impl close a"
    )
    assert record.ensign_name == "Implementation stage"

    def _close_second():
        time.sleep(0.1)
        _write_line(log, _agent_tool_result("du_2"))

    threading.Thread(target=_close_second, daemon=True).start()

    record = watcher.expect_dispatch_close(
        overall_timeout_s=2.0, ensign_name="IMPLEMENTATION", label="impl close b"
    )
    assert record.ensign_name == "Dispatch implementation"


def test_expect_dispatch_close_oldest_when_no_name(tmp_path):
    """Without ensign_name, expect_dispatch_close returns the first-closed dispatch."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="first"))
    _write_line(log, _agent_dispatch("du_2", description="second"))

    def _close_first():
        time.sleep(0.1)
        _write_line(log, _agent_tool_result("du_1"))

    threading.Thread(target=_close_first, daemon=True).start()

    record = watcher.expect_dispatch_close(overall_timeout_s=2.0, label="first close")
    assert record.ensign_name == "first"


def test_expect_dispatch_close_raises_on_per_dispatch_budget_exceeded(tmp_path):
    """dispatch_budget_s asserts on the dispatch's own elapsed time at close."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="slow-dispatch"))

    def _close_after_delay():
        time.sleep(0.3)
        _write_line(log, _agent_tool_result("du_1"))

    threading.Thread(target=_close_after_delay, daemon=True).start()

    with pytest.raises(StepTimeout) as excinfo:
        watcher.expect_dispatch_close(
            overall_timeout_s=2.0,
            dispatch_budget_s=0.1,
            label="slow dispatch budget",
        )
    assert "per-dispatch budget" in str(excinfo.value)
    assert excinfo.value.label == "slow dispatch budget"


def test_expect_dispatch_close_passes_when_per_dispatch_budget_ok(tmp_path):
    """dispatch_budget_s allows closes whose elapsed is under the per-dispatch budget."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="fast"))

    def _close_fast():
        time.sleep(0.05)
        _write_line(log, _agent_tool_result("du_1"))

    threading.Thread(target=_close_fast, daemon=True).start()

    record = watcher.expect_dispatch_close(
        overall_timeout_s=2.0,
        dispatch_budget_s=1.0,
        label="fast dispatch budget",
    )
    assert record.elapsed < 1.0


def test_dispatch_records_accumulates_across_closes(tmp_path):
    """Every close appends to watcher.dispatch_records."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="a"))
    _write_line(log, _agent_tool_result("du_1"))
    _write_line(log, _agent_dispatch("du_2", description="b"))
    _write_line(log, _agent_tool_result("du_2"))
    watcher._drain_entries()

    assert [r.ensign_name for r in watcher.dispatch_records] == ["a", "b"]
    assert all(r.elapsed >= 0.0 for r in watcher.dispatch_records)


def test_teams_spawn_ack_does_not_close_dispatch(tmp_path):
    """Teams-mode spawn-ack tool_result ('Spawned successfully. agent_id: ...') MUST NOT close.

    In teams mode the Agent tool_result fires on spawn, not completion. Closing
    on spawn-ack would make every dispatch show elapsed~0 and mask real slowness.
    """
    budget = DispatchBudget(soft_s=10.0, hard_s=30.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="teams-impl"))
    _write_line(log, _teams_spawn_ack("du_1"))
    watcher._drain_entries()

    assert len(watcher._open_dispatches) == 1
    assert watcher.dispatch_records == []


def test_teams_task_notification_completed_closes_dispatch(tmp_path):
    """Teams-mode close anchor: system entry with subtype=task_notification, status=completed."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1", description="teams-dispatch"))
    _write_line(log, _teams_spawn_ack("du_1"))
    _write_line(log, _task_notification_completed("du_1"))
    watcher._drain_entries()

    assert watcher._open_dispatches == {}
    assert len(watcher.dispatch_records) == 1
    assert watcher.dispatch_records[0].ensign_name == "teams-dispatch"


def _inbox_poll_bash_result(tool_use_id: str, sender: str, stage: str) -> dict:
    """Bash tool_result carrying an fo_inbox_poll.py Done: entry."""
    body = (
        f"team: test-team-abc\n"
        f"from: {sender}\n"
        f"timestamp: 2026-04-20T02:21:50.779Z\n"
        f"summary: {stage} complete\n"
        f"text: Done: Create a greeting file completed {stage}. Report written."
    )
    return {
        "type": "user",
        "message": {
            "content": [
                {
                    "type": "tool_result",
                    "tool_use_id": tool_use_id,
                    "content": body,
                }
            ],
        },
    }


def test_inbox_poll_bash_result_closes_teams_dispatch(tmp_path):
    """Bash tool_result carrying fo_inbox_poll.py output closes the matching dispatch.

    Under `claude -p` (anthropics/claude-code#26426), task_notification does
    not fire for teammate dispatches. The inbox-poll Bash output is the
    close anchor.
    """
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(
        log,
        _agent_dispatch(
            "du_impl",
            description="Create a greeting file: implementation",
        ),
    )
    _write_line(log, _teams_spawn_ack("du_impl"))
    _write_line(
        log,
        _inbox_poll_bash_result(
            "du_poll_1",
            sender="spacedock-ensign-keepalive-test-task-implementation",
            stage="implementation",
        ),
    )
    watcher._drain_entries()

    assert watcher._open_dispatches == {}
    assert len(watcher.dispatch_records) == 1
    assert "implementation" in watcher.dispatch_records[0].ensign_name


def test_inbox_poll_without_done_does_not_close(tmp_path):
    """An inbox-poll output that doesn't contain `Done:` is ignored."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_impl", description="impl-dispatch"))
    _write_line(
        log,
        {
            "type": "user",
            "message": {
                "content": [
                    {
                        "type": "tool_result",
                        "tool_use_id": "du_poll_noop",
                        "content": "from: spacedock-ensign-impl\ntext: still working on it",
                    }
                ],
            },
        },
    )
    watcher._drain_entries()

    assert len(watcher._open_dispatches) == 1
    assert watcher.dispatch_records == []


def test_task_notification_in_progress_does_not_close(tmp_path):
    """Only status=completed closes; intermediate task_notifications must be ignored."""
    budget = DispatchBudget(soft_s=30.0, hard_s=60.0, shutdown_grace_s=5.0)
    watcher, _proc, log = _make_watcher(tmp_path, budget)

    _write_line(log, _agent_dispatch("du_1"))
    _write_line(
        log,
        {
            "type": "system",
            "subtype": "task_notification",
            "tool_use_id": "du_1",
            "status": "running",
        },
    )
    watcher._drain_entries()

    assert len(watcher._open_dispatches) == 1
    assert watcher.dispatch_records == []
