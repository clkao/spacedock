#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Unit-style coverage for the Pi first-officer harness prompt surface.

from __future__ import annotations

from pathlib import Path
import sys
import subprocess

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import test_lib  # noqa: E402
from test_lib import PiLogParser, TestRunner, build_pi_ensign_invocation_prompt, build_pi_first_officer_invocation_prompt, create_test_project  # noqa: E402


def test_pi_harness_invokes_first_officer_via_explicit_skill_command():
    prompt = build_pi_first_officer_invocation_prompt("/tmp/example-workflow")

    assert "/skill:first-officer" in prompt
    assert "/tmp/example-workflow" in prompt
    assert "codex" not in prompt.lower()
    assert "claude" not in prompt.lower()


def test_pi_harness_can_pin_local_skill_and_plugin_root():
    prompt = build_pi_first_officer_invocation_prompt(
        "/tmp/example-workflow",
        local_skill_path="/tmp/spacedock/skills/first-officer",
        local_plugin_root="/tmp/spacedock",
    )

    assert "/skill:first-officer" in prompt
    assert "/tmp/spacedock/skills/first-officer" in prompt
    assert "/tmp/spacedock/skills/commission/bin/status" in prompt



def test_pi_ensign_prompt_can_pin_local_skill_and_assignment_fields():
    prompt = build_pi_ensign_invocation_prompt(
        "/tmp/workflow",
        "/tmp/workflow/task.md",
        "implementation",
        "Append a note.",
        None,
        ["Append a note", "Commit the work"],
        local_skill_path="/tmp/spacedock/skills/ensign",
        local_plugin_root="/tmp/spacedock",
        worker_label="218-implementation/Ensign",
    )

    assert "/skill:ensign" in prompt
    assert "/tmp/spacedock/skills/ensign" in prompt
    assert "worker_label: 218-implementation/Ensign" in prompt
    assert "entity_path: /tmp/workflow/task.md" in prompt
    assert "stage_name: implementation" in prompt
    assert "- Append a note" in prompt


def test_run_pi_prompt_can_target_a_specific_reopened_session(monkeypatch):
    runner = TestRunner("pi prompt harness", keep_test_dir=False)
    create_test_project(runner)

    seen: dict[str, object] = {}

    def fake_run(cmd, **kwargs):
        seen["cmd"] = cmd
        seen["kwargs"] = kwargs
        return subprocess.CompletedProcess(cmd, 0)

    monkeypatch.setattr(subprocess, "run", fake_run)

    exit_code = test_lib.run_pi_prompt(
        runner,
        "Recall the previous sentinel.",
        session_dir=runner.test_dir / "pi-sessions",
        session="pi-session-123",
        no_context_files=True,
        log_name="pi-reuse-log.jsonl",
    )

    assert exit_code == 0
    assert seen["cmd"][:4] == ["pi", "--mode", "json", "--print"]
    assert "--session" in seen["cmd"]
    assert "pi-session-123" in seen["cmd"]
    assert "--no-context-files" in seen["cmd"]
    assert seen["kwargs"]["cwd"] == runner.test_project_dir
    assert seen["kwargs"]["text"] is True



def test_run_pi_ensign_assembles_pi_json_command(monkeypatch):
    runner = TestRunner("pi ensign harness", keep_test_dir=False)
    create_test_project(runner)

    seen: dict[str, object] = {}

    def fake_run(cmd, **kwargs):
        seen["cmd"] = cmd
        seen["kwargs"] = kwargs
        return subprocess.CompletedProcess(cmd, 0)

    monkeypatch.setattr(subprocess, "run", fake_run)

    exit_code = test_lib.run_pi_ensign(
        runner,
        "/skill:ensign\n\nAssignment:",
        session_dir=runner.test_dir / "pi-sessions",
        session="pi-session-123",
        log_name="pi-ensign-log.jsonl",
    )

    assert exit_code == 0
    assert seen["cmd"][:4] == ["pi", "--mode", "json", "--print"]
    assert "--skill" in seen["cmd"]
    assert str(runner.repo_root / "skills" / "ensign") in seen["cmd"]
    assert "--session" in seen["cmd"]
    assert "pi-session-123" in seen["cmd"]
    assert seen["kwargs"]["cwd"] == runner.test_project_dir
    assert seen["kwargs"]["text"] is True



def test_run_pi_first_officer_assembles_pi_json_command(monkeypatch, tmp_path):
    runner = TestRunner("pi harness", keep_test_dir=False)
    create_test_project(runner)

    seen: dict[str, object] = {}

    def fake_run(cmd, **kwargs):
        seen["cmd"] = cmd
        seen["kwargs"] = kwargs
        return subprocess.CompletedProcess(cmd, 0)

    monkeypatch.setattr(subprocess, "run", fake_run)

    exit_code = test_lib.run_pi_first_officer(
        runner,
        "docs/plans",
        run_goal="Process only task 218.",
    )

    assert exit_code == 0
    assert seen["cmd"][:4] == ["pi", "--mode", "json", "--print"]
    assert "--skill" in seen["cmd"]
    assert str(runner.repo_root / "skills" / "first-officer") in seen["cmd"]
    assert seen["kwargs"]["cwd"] == runner.test_project_dir
    assert seen["kwargs"]["text"] is True



def test_pi_log_parser_extracts_session_identity_and_usage(tmp_path):
    log_path = tmp_path / "pi-log.jsonl"
    log_path.write_text(
        '{"type":"session","id":"pi-session-123"}\n'
        '{"type":"message_end","message":{"role":"assistant","usage":{"cacheRead":1536}}}\n'
    )
    parser = PiLogParser(log_path)

    session_dir = tmp_path / "sessions"
    session_dir.mkdir()
    session_file = session_dir / "2026-04-27T23-41-25-454Z_pi-session-123.jsonl"
    session_file.write_text("{}\n")

    assert parser.session_id() == "pi-session-123"
    assert parser.session_file(session_dir) == session_file
    assert parser.last_assistant_usage()["cacheRead"] == 1536


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
