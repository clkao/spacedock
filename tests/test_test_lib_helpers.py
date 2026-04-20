#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Unit tests for shared live-harness helpers in scripts/test_lib.py.
# ABOUTME: Verifies Claude runtime preflight reporting and guardrail shell-write detection heuristics.

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (
    CodexLogParser,
    TestRunner,
    _isolated_claude_env,
    bash_command_targets_write,
    emit_skip_result,
    headless_inbox_polling_hint,
    prepare_codex_skill_home,
    probe_claude_runtime,
)


TARGETS = ("skills/", "agents/", "references/", "plugin.json")


def test_bash_command_targets_write_ignores_read_only_probes():
    for command in (
        "ls -la skills/",
        "cat agents/example.md",
        "head -n 5 references/core.md",
        "tail -n 5 references/core.md",
        "grep -n guardrail skills/example.md",
        "find references -name '*.md'",
        "file plugin.json",
        "stat plugin.json",
        "wc -l skills/example.md",
    ):
        assert not bash_command_targets_write(command, TARGETS)


def test_bash_command_targets_write_flags_shell_writes():
    assert bash_command_targets_write("echo '{}' > plugin.json", TARGETS)
    assert bash_command_targets_write("printf '# x' | tee skills/example.md", TARGETS)
    assert bash_command_targets_write("sed -i '' 's/old/new/' agents/example.md", TARGETS)


def test_bash_command_targets_write_requires_a_target_match():
    assert not bash_command_targets_write("echo hi > /tmp/elsewhere.txt", TARGETS)


def test_headless_inbox_polling_hint_contains_expected_prompt_text(tmp_path):
    repo_root = tmp_path / "repo"
    keepalive_done = tmp_path / ".fo-keepalive-done"
    seen_file = tmp_path / ".fo-inbox-seen"

    hint = headless_inbox_polling_hint(repo_root, keepalive_done, seen_file)

    assert "HEADLESS INBOX-POLLING RULE." in hint
    assert f"The spacedock plugin directory is at `{repo_root}`." in hint
    assert f"Until the sentinel file `{keepalive_done}` exists" in hint
    poll_script = repo_root / "scripts" / "fo_inbox_poll.py"
    assert (
        f"python3 {poll_script} --home \"$HOME\" --pattern 'Done:' "
        f"--timeout 5 --seen-file {seen_file}"
    ) in hint
    assert "Never emit `SendMessage(shutdown_request)`" in hint


def test_probe_claude_runtime_reports_timeout(monkeypatch):
    def fake_run(*args, **kwargs):
        raise subprocess.TimeoutExpired(cmd=args[0], timeout=17)

    monkeypatch.setattr(subprocess, "run", fake_run)

    ok, reason = probe_claude_runtime("haiku", timeout_s=17)

    assert not ok
    assert "within 17s" in reason


def test_probe_claude_runtime_reports_non_zero_exit(monkeypatch):
    class Result:
        returncode = 9
        stdout = ""

    monkeypatch.setattr(subprocess, "run", lambda *args, **kwargs: Result())

    ok, reason = probe_claude_runtime("haiku", timeout_s=30)

    assert not ok
    assert "exited 9" in reason


def test_probe_claude_runtime_reports_missing_result_record(monkeypatch):
    class Result:
        returncode = 0
        stdout = '{"type":"message"}'

    monkeypatch.setattr(subprocess, "run", lambda *args, **kwargs: Result())

    ok, reason = probe_claude_runtime("haiku", timeout_s=30)

    assert not ok
    assert "returned no stream-json result record" in reason


def test_probe_claude_runtime_succeeds_with_result_record_and_clean_env(monkeypatch):
    seen_env = {}

    class Result:
        returncode = 0
        stdout = '{"type":"result"}'

    def fake_run(*args, **kwargs):
        seen_env.update(kwargs["env"])
        return Result()

    monkeypatch.setenv("CLAUDECODE", "1")
    monkeypatch.setattr(subprocess, "run", fake_run)

    ok, reason = probe_claude_runtime("haiku", timeout_s=30)

    assert ok
    assert reason == ""
    assert "CLAUDECODE" not in seen_env


def test_emit_skip_result_prints_standardized_skip_output(capsys):
    with pytest.raises(SystemExit) as excinfo:
        emit_skip_result("runtime unavailable")

    captured = capsys.readouterr().out
    assert "SKIP: runtime unavailable" in captured
    assert "RESULT: SKIP" in captured
    assert excinfo.value.code == 0


def test_test_runner_uses_configured_temp_root(monkeypatch, tmp_path):
    configured_root = tmp_path / "live-artifacts"
    monkeypatch.setenv("SPACEDOCK_TEST_TMP_ROOT", str(configured_root))

    runner = TestRunner("helper temp root", keep_test_dir=True)

    assert runner.test_dir.parent == configured_root
    assert runner.test_dir.name.startswith("spacedock-test-")


def test_prepare_codex_skill_home_creates_writable_codex_home_when_real_home_missing(
    monkeypatch, tmp_path
):
    fake_home = tmp_path / "real-home"
    fake_home.mkdir()
    monkeypatch.setenv("HOME", str(fake_home))

    repo_root = Path(__file__).resolve().parent.parent
    prepared_home = prepare_codex_skill_home(tmp_path / "test-root", repo_root)

    codex_home = prepared_home / ".codex"
    assert codex_home.exists()
    assert codex_home.is_dir()
    assert not codex_home.is_symlink()


def test_codex_log_parser_returns_structured_collab_calls_in_order(tmp_path):
    log_path = tmp_path / "codex-log.jsonl"
    entries = [
        {
            "type": "item.completed",
            "item": {
                "id": "item_1",
                "type": "collab_tool_call",
                "tool": "spawn_agent",
                "receiver_thread_ids": ["thread-1"],
                "prompt": "stage_name: implementation",
            },
        },
        {
            "type": "item.completed",
            "item": {
                "id": "item_2",
                "type": "collab_tool_call",
                "tool": "send_input",
                "receiver_thread_ids": ["thread-1"],
                "prompt": "follow-up",
            },
        },
        {
            "type": "item.completed",
            "item": {
                "id": "item_3",
                "type": "collab_tool_call",
                "tool": "wait",
                "receiver_thread_ids": ["thread-1"],
                "agents_states": {
                    "thread-1": {
                        "status": "completed",
                        "message": "Commit hash: `abc1234`",
                    }
                },
            },
        },
    ]
    log_path.write_text("\n".join(json.dumps(entry) for entry in entries))

    parser = CodexLogParser(log_path)

    assert [call["tool"] for call in parser.collab_tool_calls()] == [
        "spawn_agent",
        "send_input",
        "wait",
    ]
    assert parser.collab_tool_calls("send_input")[0]["receiver_thread_ids"] == ["thread-1"]


def test_codex_log_parser_returns_only_agent_message_texts(tmp_path):
    log_path = tmp_path / "codex-log.jsonl"
    entries = [
        {
            "type": "item.completed",
            "item": {
                "id": "item_1",
                "type": "agent_message",
                "text": "Dispatching 001-implementation/Ensign (spacedock:ensign, handle: item_23).",
            },
        },
        {
            "type": "item.completed",
            "item": {
                "id": "item_2",
                "type": "collab_tool_call",
                "tool": "wait",
                "receiver_thread_ids": ["thread-1"],
                "agents_states": {},
            },
        },
        {
            "type": "item.completed",
            "item": {
                "id": "item_3",
                "type": "agent_message",
                "text": "Routing follow-up to 001-implementation/Ensign on handle item_23.",
            },
        },
    ]
    log_path.write_text("\n".join(json.dumps(entry) for entry in entries))

    parser = CodexLogParser(log_path)

    assert parser.agent_message_texts() == [
        "Dispatching 001-implementation/Ensign (spacedock:ensign, handle: item_23).",
        "Routing follow-up to 001-implementation/Ensign on handle item_23.",
    ]


def test_isolated_claude_env_injects_oauth_token_when_token_file_present(monkeypatch, tmp_path):
    fake_home = tmp_path / "real-home"
    claude_dir = fake_home / ".claude"
    claude_dir.mkdir(parents=True)
    (claude_dir / "benchmark-token").write_text("sk-oauth-test-token\n")
    monkeypatch.setenv("HOME", str(fake_home))
    monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-api-should-be-dropped")

    env = _isolated_claude_env()

    assert env is not None
    assert env["CLAUDE_CODE_OAUTH_TOKEN"] == "sk-oauth-test-token"
    assert "ANTHROPIC_API_KEY" not in env
    assert env["HOME"] != str(fake_home)
    assert Path(env["HOME"]).is_dir()


def test_isolated_claude_env_preserves_api_key_when_no_token_file(monkeypatch, tmp_path):
    fake_home = tmp_path / "real-home"
    fake_home.mkdir()
    monkeypatch.setenv("HOME", str(fake_home))
    monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-ci-api-key")

    env = _isolated_claude_env()

    assert env is not None
    assert env["ANTHROPIC_API_KEY"] == "sk-ci-api-key"
    assert "CLAUDE_CODE_OAUTH_TOKEN" not in env
    assert env["HOME"] != str(fake_home)
    assert Path(env["HOME"]).is_dir()


def test_isolated_claude_env_returns_none_when_no_auth_available(monkeypatch, tmp_path):
    fake_home = tmp_path / "real-home"
    fake_home.mkdir()
    monkeypatch.setenv("HOME", str(fake_home))
    monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
    monkeypatch.delenv("CLAUDE_CODE_OAUTH_TOKEN", raising=False)

    env = _isolated_claude_env()

    assert env is None
