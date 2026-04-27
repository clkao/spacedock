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
from test_lib import TestRunner, build_pi_first_officer_invocation_prompt, create_test_project, run_pi_first_officer  # noqa: E402


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


def test_run_pi_first_officer_assembles_pi_json_command(monkeypatch, tmp_path):
    runner = TestRunner("pi harness", keep_test_dir=False)
    create_test_project(runner)

    seen: dict[str, object] = {}

    def fake_run(cmd, **kwargs):
        seen["cmd"] = cmd
        seen["kwargs"] = kwargs
        return subprocess.CompletedProcess(cmd, 0)

    monkeypatch.setattr(subprocess, "run", fake_run)

    exit_code = run_pi_first_officer(
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


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
