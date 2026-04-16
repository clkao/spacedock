#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Unit tests for the claude-team script's context-budget and build subcommands.
# ABOUTME: Tests token extraction, model mapping, threshold logic, dispatch assembly, and validation rules.

from __future__ import annotations

import json
import os
import stat
import subprocess
import sys
import tempfile
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
SCRIPT = REPO_ROOT / "skills" / "commission" / "bin" / "claude-team"


def make_jsonl_fixture(
    tmp_path: Path,
    agent_name: str,
    usage: dict | None = None,
    model: str | None = None,
) -> tuple[Path, Path]:
    """Create a fake subagent jsonl + meta.json pair.

    Returns (meta_path, jsonl_path).
    """
    projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
    projects_dir.mkdir(parents=True, exist_ok=True)

    meta_path = projects_dir / "agent-abc123.meta.json"
    jsonl_path = projects_dir / "agent-abc123.jsonl"

    meta_path.write_text(json.dumps({"agentType": agent_name}))

    lines = []
    # A user entry
    lines.append(json.dumps({"type": "human", "message": {"role": "user", "content": "hello"}}))
    # An assistant entry with usage
    if usage is not None:
        msg = {"role": "assistant", "usage": usage}
        if model is not None:
            msg["model"] = model
        lines.append(json.dumps({
            "type": "assistant",
            "message": msg,
        }))
    jsonl_path.write_text("\n".join(lines) + "\n")

    return meta_path, jsonl_path


def make_team_config(
    tmp_path: Path,
    team_name: str,
    member_name: str,
    model: str,
) -> Path:
    """Create a fake team config with one member."""
    teams_dir = tmp_path / ".claude" / "teams" / team_name
    teams_dir.mkdir(parents=True, exist_ok=True)

    config_path = teams_dir / "config.json"
    config_path.write_text(json.dumps({
        "name": team_name,
        "members": [
            {
                "name": "team-lead",
                "agentType": "team-lead",
                "model": "claude-opus-4-6[1m]",
            },
            {
                "name": member_name,
                "agentType": "spacedock:ensign",
                "model": model,
            },
        ],
    }))
    return config_path


def run_context_budget(tmp_path: Path, name: str) -> subprocess.CompletedProcess:
    """Run the claude-team context-budget subcommand with HOME overridden."""
    env = {**os.environ, "HOME": str(tmp_path)}
    return subprocess.run(
        [sys.executable, str(SCRIPT), "context-budget", "--name", name],
        capture_output=True,
        text=True,
        env=env,
    )


class TestContextBudgetBasic:
    """Basic context-budget functionality."""

    def test_outputs_valid_json(self, tmp_path):
        usage = {
            "input_tokens": 5000,
            "cache_creation_input_tokens": 3000,
            "cache_read_input_tokens": 2000,
        }
        make_jsonl_fixture(tmp_path, "spacedock-ensign-foo-impl", usage)
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        assert data["name"] == "spacedock-ensign-foo-impl"
        assert data["resident_tokens"] == 10000
        assert data["model"] == "claude-opus-4-6"
        assert data["context_limit"] == 200000
        assert data["usage_pct"] == pytest.approx(5.0)
        assert data["threshold_pct"] == 60
        assert data["reuse_ok"] is True

    def test_uses_last_assistant_entry(self, tmp_path):
        """When multiple assistant entries exist, use the last one's usage."""
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        meta_path = projects_dir / "agent-abc123.meta.json"
        jsonl_path = projects_dir / "agent-abc123.jsonl"

        meta_path.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))

        lines = [
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 1000, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0,
            }}}),
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 50000, "cache_creation_input_tokens": 30000, "cache_read_input_tokens": 40000,
            }}}),
        ]
        jsonl_path.write_text("\n".join(lines) + "\n")

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert data["resident_tokens"] == 120000


class TestThresholdBoundary:
    """60% threshold boundary tests."""

    def _make_at_percentage(self, tmp_path, pct: float, model: str = "claude-opus-4-6"):
        """Create fixtures that produce a specific usage percentage."""
        context_limit = 200000
        resident_tokens = int(context_limit * pct / 100)
        usage = {
            "input_tokens": resident_tokens,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 0,
        }
        make_jsonl_fixture(tmp_path, "spacedock-ensign-foo-impl", usage)
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", model)

    def test_59_percent_reuse_ok(self, tmp_path):
        self._make_at_percentage(tmp_path, 59.0)
        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert data["reuse_ok"] is True

    def test_60_percent_reuse_ok(self, tmp_path):
        self._make_at_percentage(tmp_path, 60.0)
        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert data["reuse_ok"] is True

    def test_61_percent_reuse_not_ok(self, tmp_path):
        self._make_at_percentage(tmp_path, 61.0)
        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert data["reuse_ok"] is False


class TestModelMapping:
    """Model-to-context-limit mapping tests."""

    @pytest.mark.parametrize("model,expected_limit", [
        ("claude-opus-4-6", 200000),
        ("claude-opus-4-6[1m]", 1000000),
        ("claude-sonnet-4-6", 200000),
        ("claude-sonnet-4-6[1m]", 1000000),  # heuristic works on any base model
        ("claude-haiku-4-5-20251001", 200000),
        ("unknown-model-xyz", 200000),
        ("unknown-model-xyz[1m]", 1000000),  # heuristic works on unknown models too
    ])
    def test_model_context_limits(self, tmp_path, model, expected_limit):
        usage = {
            "input_tokens": 100000,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 0,
        }
        make_jsonl_fixture(tmp_path, "spacedock-ensign-foo-impl", usage)
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", model)

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert data["context_limit"] == expected_limit
        assert data["model"] == model


class TestErrorCases:
    """Error handling tests."""

    def test_missing_jsonl(self, tmp_path):
        """No matching subagent jsonl found."""
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")
        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode != 0

    def test_no_assistant_turns(self, tmp_path):
        """Jsonl exists but has no assistant entries with usage."""
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        meta_path = projects_dir / "agent-abc123.meta.json"
        jsonl_path = projects_dir / "agent-abc123.jsonl"

        meta_path.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))
        jsonl_path.write_text(json.dumps({"type": "human", "message": {"role": "user"}}) + "\n")

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode != 0

    def test_no_subcommand(self, tmp_path):
        """Running without a subcommand should fail."""
        env = {**os.environ, "HOME": str(tmp_path)}
        result = subprocess.run(
            [sys.executable, str(SCRIPT)],
            capture_output=True,
            text=True,
            env=env,
        )
        assert result.returncode != 0

    def test_missing_name_flag(self, tmp_path):
        """Running context-budget without --name should fail."""
        env = {**os.environ, "HOME": str(tmp_path)}
        result = subprocess.run(
            [sys.executable, str(SCRIPT), "context-budget"],
            capture_output=True,
            text=True,
            env=env,
        )
        assert result.returncode != 0


class TestPeakTokens:
    """Peak-token extraction: scan backward past zero-usage final turns."""

    def test_dead_ensign_zero_final_turn_returns_peak(self, tmp_path):
        """An ensign that died mid-turn has a zero-usage final assistant entry.

        The script must report the last non-zero peak, not zero.
        """
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        meta_path = projects_dir / "agent-abc123.meta.json"
        jsonl_path = projects_dir / "agent-abc123.jsonl"

        meta_path.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))

        lines = [
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 10000, "cache_creation_input_tokens": 20000, "cache_read_input_tokens": 30000,
            }}}),
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 50000, "cache_creation_input_tokens": 60000, "cache_read_input_tokens": 65000,
            }}}),
            # Dead turn: all zeros
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0,
            }}}),
        ]
        jsonl_path.write_text("\n".join(lines) + "\n")

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        # Second turn: 50000 + 60000 + 65000 = 175000 on a 200000-limit -> 87.5%
        assert data["resident_tokens"] == 175000
        assert data["usage_pct"] == pytest.approx(87.5)
        assert data["reuse_ok"] is False

    def test_live_ensign_last_turn_is_peak(self, tmp_path):
        """A healthy ensign's last turn has non-zero usage; returns that value."""
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        meta_path = projects_dir / "agent-abc123.meta.json"
        jsonl_path = projects_dir / "agent-abc123.jsonl"

        meta_path.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))

        lines = [
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 1000, "cache_creation_input_tokens": 500, "cache_read_input_tokens": 500,
            }}}),
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 5000, "cache_creation_input_tokens": 3000, "cache_read_input_tokens": 2000,
            }}}),
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 20000, "cache_creation_input_tokens": 15000, "cache_read_input_tokens": 15000,
            }}}),
        ]
        jsonl_path.write_text("\n".join(lines) + "\n")

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        # Last turn: 20000 + 15000 + 15000 = 50000 on 200000 -> 25%
        assert data["resident_tokens"] == 50000
        assert data["usage_pct"] == pytest.approx(25.0)
        assert data["reuse_ok"] is True

    def test_multiple_trailing_zero_turns(self, tmp_path):
        """Multiple trailing zero-usage assistant turns still return the real peak."""
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        meta_path = projects_dir / "agent-abc123.meta.json"
        jsonl_path = projects_dir / "agent-abc123.jsonl"

        meta_path.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))

        zero_turn = json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
            "input_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0,
        }}})
        lines = [
            json.dumps({"type": "assistant", "message": {"role": "assistant", "usage": {
                "input_tokens": 40000, "cache_creation_input_tokens": 40000, "cache_read_input_tokens": 40000,
            }}}),
            zero_turn,
            zero_turn,
            zero_turn,
        ]
        jsonl_path.write_text("\n".join(lines) + "\n")

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        assert data["resident_tokens"] == 120000


class TestMostRecentMatch:
    """When multiple meta.json files match, use most recently modified."""

    def test_picks_most_recent(self, tmp_path):
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        # Older file
        old_meta = projects_dir / "agent-old111.meta.json"
        old_jsonl = projects_dir / "agent-old111.jsonl"
        old_meta.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))
        old_jsonl.write_text(json.dumps({
            "type": "assistant",
            "message": {"role": "assistant", "usage": {
                "input_tokens": 1000,
                "cache_creation_input_tokens": 0,
                "cache_read_input_tokens": 0,
            }},
        }) + "\n")
        # Set older mtime
        os.utime(old_meta, (1000000, 1000000))

        # Newer file
        new_meta = projects_dir / "agent-new222.meta.json"
        new_jsonl = projects_dir / "agent-new222.jsonl"
        new_meta.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))
        new_jsonl.write_text(json.dumps({
            "type": "assistant",
            "message": {"role": "assistant", "usage": {
                "input_tokens": 99000,
                "cache_creation_input_tokens": 0,
                "cache_read_input_tokens": 0,
            }},
        }) + "\n")
        # Set newer mtime
        os.utime(new_meta, (2000000, 2000000))

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "claude-opus-4-6")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert data["resident_tokens"] == 99000


class TestRuntimeModelDetection:
    """Runtime model from jsonl overrides config-declared model."""

    def test_config_1m_runtime_bare_uses_200k(self, tmp_path):
        """Config says opus[1m] but runtime jsonl shows bare opus → 200k limit, drift warning."""
        usage = {
            "input_tokens": 85179,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 0,
        }
        make_jsonl_fixture(tmp_path, "spacedock-ensign-foo-impl", usage, model="claude-opus-4-6")
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "opus[1m]")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        assert data["model"] == "claude-opus-4-6"
        assert data["context_limit"] == 200000
        assert data["usage_pct"] == pytest.approx(42.6)
        assert "config_drift_warning" in data
        assert data["config_declared_model"] == "opus[1m]"

    def test_config_1m_runtime_1m_uses_1m(self, tmp_path):
        """Config says opus[1m] and runtime jsonl also shows [1m] → 1M limit, no drift."""
        usage = {
            "input_tokens": 85179,
            "cache_creation_input_tokens": 0,
            "cache_read_input_tokens": 0,
        }
        make_jsonl_fixture(tmp_path, "spacedock-ensign-foo-impl", usage, model="claude-opus-4-6[1m]")
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "opus[1m]")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        assert data["model"] == "claude-opus-4-6[1m]"
        assert data["context_limit"] == 1000000
        assert data["usage_pct"] == pytest.approx(8.5)
        assert "config_drift_warning" not in data

    def test_mixed_models_uses_smallest_context(self, tmp_path):
        """Jsonl has mixed models → use smallest context window, emit warning."""
        projects_dir = tmp_path / ".claude" / "projects" / "test-project" / "session-1" / "subagents"
        projects_dir.mkdir(parents=True, exist_ok=True)

        meta_path = projects_dir / "agent-abc123.meta.json"
        jsonl_path = projects_dir / "agent-abc123.jsonl"

        meta_path.write_text(json.dumps({"agentType": "spacedock-ensign-foo-impl"}))

        lines = [
            json.dumps({"type": "assistant", "message": {
                "role": "assistant",
                "model": "claude-opus-4-6[1m]",
                "usage": {"input_tokens": 10000, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
            }}),
            json.dumps({"type": "assistant", "message": {
                "role": "assistant",
                "model": "claude-opus-4-6",
                "usage": {"input_tokens": 50000, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
            }}),
        ]
        jsonl_path.write_text("\n".join(lines) + "\n")

        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "opus[1m]")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        assert data["context_limit"] == 200000
        assert "mixed_models_warning" in data

    def test_no_model_in_jsonl_falls_back_to_config(self, tmp_path):
        """Jsonl assistant entries have no model field → fall back to config with warning."""
        usage = {
            "input_tokens": 5000,
            "cache_creation_input_tokens": 3000,
            "cache_read_input_tokens": 2000,
        }
        make_jsonl_fixture(tmp_path, "spacedock-ensign-foo-impl", usage)
        make_team_config(tmp_path, "test-team", "spacedock-ensign-foo-impl", "opus[1m]")

        result = run_context_budget(tmp_path, "spacedock-ensign-foo-impl")
        assert result.returncode == 0, f"stderr: {result.stderr}"
        data = json.loads(result.stdout)
        assert data["model"] == "opus[1m]"
        assert data["context_limit"] == 1000000
        assert "config_fallback_warning" in data


# --- Build subcommand tests ---

STATUS_PATH = REPO_ROOT / "skills" / "commission" / "bin" / "status"
WORKFLOW_DIR = REPO_ROOT / "docs" / "plans"
WORKFLOW_README = WORKFLOW_DIR / "README.md"


def _make_workflow_fixture(tmp_path: Path) -> tuple[Path, Path]:
    """Create a minimal workflow directory with README and entity file for build tests."""
    wf_dir = tmp_path / "workflow"
    wf_dir.mkdir()
    readme = wf_dir / "README.md"
    readme.write_text(
        "---\n"
        "commissioned-by: spacedock@test\n"
        "entity-label: task\n"
        "stages:\n"
        "  defaults:\n"
        "    worktree: false\n"
        "    concurrency: 2\n"
        "  states:\n"
        "    - name: backlog\n"
        "      initial: true\n"
        "    - name: ideation\n"
        "    - name: implementation\n"
        "      worktree: true\n"
        "    - name: validation\n"
        "      worktree: true\n"
        "      fresh: true\n"
        "      feedback-to: implementation\n"
        "      gate: true\n"
        "    - name: done\n"
        "      terminal: true\n"
        "---\n"
        "\n"
        "# Test Workflow\n"
        "\n"
        "## Stages\n"
        "\n"
        "### `backlog`\n"
        "\n"
        "The initial holding stage.\n"
        "\n"
        "- **Inputs:** None\n"
        "- **Outputs:** A seed task\n"
        "\n"
        "### `ideation`\n"
        "\n"
        "Flesh out the idea.\n"
        "\n"
        "- **Inputs:** Seed description\n"
        "- **Outputs:** Fleshed-out task\n"
        "\n"
        "### `implementation`\n"
        "\n"
        "Produce the deliverable.\n"
        "\n"
        "- **Inputs:** Approved design\n"
        "- **Outputs:** Code committed to worktree\n"
        "\n"
        "### `validation`\n"
        "\n"
        "Verify the deliverable.\n"
        "\n"
        "- **Inputs:** Implementation summary\n"
        "- **Outputs:** PASSED/REJECTED recommendation\n"
        "\n"
        "### `done`\n"
        "\n"
        "Terminal.\n"
    )
    entity = wf_dir / "my-task.md"
    entity.write_text(
        "---\n"
        "id: 001\n"
        "title: My test task\n"
        "status: ideation\n"
        "score: 0.50\n"
        "worktree:\n"
        "---\n"
        "\n"
        "Description of the task.\n"
    )
    return wf_dir, entity


def _make_worktree_entity(tmp_path: Path, wf_dir: Path) -> Path:
    """Create an entity with a worktree field and an actual worktree directory.

    Also creates a .git marker in tmp_path so find_git_root resolves correctly.
    """
    # Ensure find_git_root can find the project root at tmp_path
    (tmp_path / ".git").mkdir(exist_ok=True)
    entity = wf_dir / "wt-task.md"
    wt_rel = ".worktrees/spacedock-ensign-wt-task"
    wt_abs = tmp_path / wt_rel
    wt_abs.mkdir(parents=True)
    # Put a copy of the entity in the worktree
    entity.write_text(
        "---\n"
        "id: 002\n"
        "title: Worktree task\n"
        "status: implementation\n"
        f"worktree: {wt_rel}\n"
        "---\n"
        "\n"
        "A worktree-backed task.\n"
    )
    (wt_abs / "wt-task.md").write_text(entity.read_text())
    return entity


def run_build(wf_dir: Path, stdin_data: dict, cwd: Path | None = None) -> subprocess.CompletedProcess:
    """Run claude-team build with the given stdin JSON."""
    return subprocess.run(
        [sys.executable, str(SCRIPT), "build", "--workflow-dir", str(wf_dir)],
        input=json.dumps(stdin_data),
        capture_output=True,
        text=True,
        cwd=cwd or wf_dir.parent,
    )


class TestBuildHelp:
    """AC-1: Subcommand exists."""

    def test_claude_team_build_help(self):
        result = subprocess.run(
            [sys.executable, str(SCRIPT), "build", "--help"],
            capture_output=True, text=True,
        )
        assert result.returncode == 0
        assert "--workflow-dir" in result.stdout


class TestBuildNormalDispatch:
    """AC-2: Normal dispatch emits valid JSON."""

    def test_build_normal_dispatch(self, tmp_path):
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Do the first thing", "2. Do the second thing"],
            "team_name": "test-team",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert out["schema_version"] == 1
        assert out["subagent_type"] == "spacedock:ensign"
        assert "prompt" in out
        assert "My test task" in out["prompt"]
        assert "ideation" in out["description"]


class TestBuildTeamMode:
    """AC-3: Team-mode includes name, team_name, and completion signal."""

    def test_build_team_mode_dispatch(self, tmp_path):
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Test item"],
            "team_name": "happy-team",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert out["name"] == "spacedock-ensign-my-task-ideation"
        assert out["team_name"] == "happy-team"
        assert 'SendMessage(to="team-lead"' in out["prompt"]
        assert "Completion Signal" in out["prompt"]


class TestBuildBareMode:
    """AC-4: Bare-mode omits team fields and completion signal."""

    def test_build_bare_mode_dispatch(self, tmp_path):
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "bare_mode": True,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert "name" not in out
        assert "team_name" not in out
        assert "SendMessage" not in out["prompt"]
        assert "Completion Signal" not in out["prompt"]


class TestBuildWorktreeStage:
    """AC-5: Worktree-stage dispatch includes worktree instructions."""

    def test_build_worktree_stage_dispatch(self, tmp_path):
        wf_dir, _ = _make_workflow_fixture(tmp_path)
        entity = _make_worktree_entity(tmp_path, wf_dir)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "implementation",
            "checklist": ["1. Write code"],
            "team_name": "impl-team",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp, cwd=tmp_path)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert "Your working directory is" in out["prompt"]
        assert "Your git branch is" in out["prompt"]
        assert "spacedock-ensign/wt-task" in out["prompt"]
        assert "Do NOT switch branches" in out["prompt"]
        # AC-2: assert exact worktree-local entity path preserves workflow-dir subpath.
        # Fixture: git_root=tmp_path, workflow under tmp_path/workflow/, so subpath is "workflow/".
        expected_entity = str(
            tmp_path / ".worktrees" / "spacedock-ensign-wt-task" / "workflow" / "wt-task.md"
        )
        assert f"Read the entity file at {expected_entity}" in out["prompt"]


class TestBuildEntityPathTranslation:
    """AC-1/AC-2/AC-3: Entity path translation preserves workflow_dir subpath."""

    def test_build_entity_path_nested_workflow_dir(self, tmp_path):
        """AC-1: Nested workflow dir (docs/plans/) preserves subpath in worktree entity path."""
        (tmp_path / ".git").mkdir()
        wf_dir = tmp_path / "docs" / "plans"
        wf_dir.mkdir(parents=True)
        (wf_dir / "README.md").write_text(
            "---\n"
            "commissioned-by: spacedock@test\n"
            "entity-label: task\n"
            "stages:\n"
            "  defaults:\n"
            "    worktree: false\n"
            "  states:\n"
            "    - name: ideation\n"
            "      initial: true\n"
            "    - name: implementation\n"
            "      worktree: true\n"
            "    - name: done\n"
            "      terminal: true\n"
            "---\n"
            "\n"
            "## Stages\n\n"
            "### `ideation`\n\nThink.\n\n"
            "### `implementation`\n\nBuild.\n\n"
            "### `done`\n\nTerminal.\n"
        )
        wt_rel = ".worktrees/spacedock-ensign-task"
        wt_abs = tmp_path / wt_rel
        (wt_abs / "docs" / "plans").mkdir(parents=True)
        entity = wf_dir / "task.md"
        entity.write_text(
            "---\n"
            "id: 100\n"
            "title: Nested task\n"
            "status: implementation\n"
            f"worktree: {wt_rel}\n"
            "---\n"
        )
        (wt_abs / "docs" / "plans" / "task.md").write_text(entity.read_text())
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "implementation",
            "checklist": ["1. Build"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp, cwd=tmp_path)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        expected_entity = str(wt_abs / "docs" / "plans" / "task.md")
        assert f"Read the entity file at {expected_entity}" in out["prompt"]

    def test_build_entity_path_non_worktree(self, tmp_path):
        """AC-3: Non-worktree stage leaves entity path untranslated (main-branch path)."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Think"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert f"Read the entity file at {entity}" in out["prompt"]
        assert ".worktrees" not in out["prompt"]


class TestBuildFeedbackDispatch:
    """AC-6: Feedback dispatch includes feedback context."""

    def test_build_feedback_dispatch(self, tmp_path):
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Fix the issue"],
            "team_name": "fb-team",
            "feedback_context": "The validator found bug X in line 42.",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert "Feedback from prior review" in out["prompt"]
        assert "bug X in line 42" in out["prompt"]


class TestBuildValidationRules:
    """AC-7: Each validation rule enforced."""

    def test_build_validation_rule_1_missing_required_field(self, tmp_path):
        """Rule 1: Missing required field."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            # "stage" is missing
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "missing required field 'stage'" in result.stderr

    def test_build_validation_rule_2_schema_version(self, tmp_path):
        """Rule 2: Unsupported schema version."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 99,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 2
        assert "unsupported input schema_version 99, expected 1" in result.stderr

    def test_build_validation_rule_3_stage_not_found(self, tmp_path):
        """Rule 3: Stage not in workflow."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "nonexistent",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "stage 'nonexistent' not found" in result.stderr

    def test_build_validation_rule_4_worktree_missing_path(self, tmp_path):
        """Rule 4: Worktree stage but entity has no worktree path."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        # entity has empty worktree field, but implementation requires worktree
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "implementation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "worktree stage 'implementation' but entity has no worktree path" in result.stderr

    def test_build_validation_rule_4_worktree_dir_not_exist(self, tmp_path):
        """Rule 4: Worktree path does not exist on disk."""
        wf_dir, _ = _make_workflow_fixture(tmp_path)
        entity = wf_dir / "ghost-wt.md"
        entity.write_text(
            "---\n"
            "id: 003\n"
            "title: Ghost worktree\n"
            "status: implementation\n"
            "worktree: .worktrees/does-not-exist\n"
            "---\n"
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "implementation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp, cwd=tmp_path)
        assert result.returncode == 1
        assert "worktree path" in result.stderr
        assert "does not exist" in result.stderr

    def test_build_validation_rule_5_feedback_context_missing(self, tmp_path):
        """Rule 5: Feedback reflow without feedback_context."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
            "is_feedback_reflow": True,
            # feedback_context intentionally missing
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "feedback_context is missing" in result.stderr

    def test_build_validation_rule_7_name_too_long(self, tmp_path):
        """Rule 7: Derived name exceeds NAME_MAX_LEN (200) characters."""
        wf_dir, _ = _make_workflow_fixture(tmp_path)
        long_slug = "a" * 220
        entity = wf_dir / f"{long_slug}.md"
        entity.write_text(
            "---\n"
            f"id: 004\n"
            f"title: Long slug task\n"
            "status: ideation\n"
            "worktree:\n"
            "---\n"
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "exceeds 200 characters" in result.stderr

    def test_build_long_derived_name(self, tmp_path):
        """AC-5: Derived name longer than the old 63-char limit succeeds."""
        wf_dir, _ = _make_workflow_fixture(tmp_path)
        # "spacedock-ensign-" (17) + slug (45) + "-ideation" (9) = 71 chars total.
        slug = "fo-enforce-mod-blocking-at-runtime-cycle-three"[:45]
        assert len(slug) == 45
        entity = wf_dir / f"{slug}.md"
        entity.write_text(
            "---\n"
            "id: 005\n"
            "title: Long slug in real life\n"
            "status: ideation\n"
            "worktree:\n"
            "---\n"
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert len(out["name"]) == 71
        assert out["name"] == f"spacedock-ensign-{slug}-ideation"

    def test_build_very_long_name_still_rejected(self, tmp_path):
        """Sanity bound: a name above 200 chars still rejects."""
        wf_dir, _ = _make_workflow_fixture(tmp_path)
        long_slug = "z" * 205
        entity = wf_dir / f"{long_slug}.md"
        entity.write_text(
            "---\n"
            "id: 006\n"
            "title: Pathological slug\n"
            "status: ideation\n"
            "worktree:\n"
            "---\n"
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "exceeds 200 characters" in result.stderr

    def test_build_validation_rule_8_team_name_missing(self, tmp_path):
        """Rule 8: Team mode without team_name."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "bare_mode": False,
            # team_name intentionally missing
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "team mode requires team_name" in result.stderr

    def test_build_validation_rule_9_checklist_empty(self, tmp_path):
        """Rule 9: Empty checklist."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": [],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "checklist must not be empty" in result.stderr

    def test_build_validation_rule_10_entity_not_readable(self, tmp_path):
        """Rule 10: Entity file does not exist."""
        wf_dir, _ = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 1,
            "entity_path": str(tmp_path / "nonexistent.md"),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 1
        assert "entity file not readable" in result.stderr

    def test_build_validation_rule_11_workflow_readme_missing(self, tmp_path):
        """Rule 11: Workflow README not found."""
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        entity = tmp_path / "dummy.md"
        entity.write_text("---\ntitle: Dummy\n---\n")
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(empty_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(empty_dir, inp)
        assert result.returncode == 1
        assert "workflow README not found" in result.stderr


class TestBuildSchemaVersion:
    """AC-8: Schema version check."""

    def test_build_schema_version_rejection(self, tmp_path):
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        inp = {
            "schema_version": 2,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "t",
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 2
        assert "unsupported input schema_version 2, expected 1" in result.stderr


class TestStatusSiblingImport:
    """AC-9: Sibling import works and is guarded against signature drift."""

    def _import_status(self):
        import importlib.machinery
        loader = importlib.machinery.SourceFileLoader("_status_lib", str(STATUS_PATH))
        spec = importlib.util.spec_from_file_location("_status_lib", str(STATUS_PATH), loader=loader)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod

    def test_status_sibling_import_parse_frontmatter(self):
        mod = self._import_status()
        assert callable(mod.parse_frontmatter)
        # Smoke test against a stable fixture entity (decoupled from live workflow)
        fixture = REPO_ROOT / "tests" / "fixtures" / "workflow-entity" / "sample-entity.md"
        result = mod.parse_frontmatter(str(fixture))
        assert isinstance(result, dict)
        assert "title" in result

    def test_status_sibling_import_parse_stages_block(self):
        mod = self._import_status()
        assert callable(mod.parse_stages_block)
        result = mod.parse_stages_block(str(WORKFLOW_README))
        assert isinstance(result, list)
        assert len(result) > 0
        assert "name" in result[0]

    def test_status_sibling_import_load_active_entity_fields(self):
        mod = self._import_status()
        assert callable(mod.load_active_entity_fields)
        # Smoke test against a stable fixture entity (decoupled from live workflow)
        fixture = REPO_ROOT / "tests" / "fixtures" / "workflow-entity" / "sample-entity.md"
        result = mod.load_active_entity_fields(
            str(fixture),
            str(REPO_ROOT),
        )
        assert isinstance(result, dict)

    def test_status_sibling_import_find_git_root(self):
        mod = self._import_status()
        assert callable(mod.find_git_root)
        result = mod.find_git_root(str(WORKFLOW_DIR))
        assert os.path.isdir(result)


class TestParseStagesBlockExtraFields:
    """AC-10: parse_stages_block includes feedback-to, agent, fresh fields."""

    def test_parse_stages_block_extra_fields(self):
        import importlib.machinery
        loader = importlib.machinery.SourceFileLoader("_status_lib", str(STATUS_PATH))
        spec = importlib.util.spec_from_file_location("_status_lib", str(STATUS_PATH), loader=loader)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)

        stages = mod.parse_stages_block(str(WORKFLOW_README))
        assert stages is not None

        stage_by_name = {s["name"]: s for s in stages}

        # validation stage has feedback-to: implementation and fresh: true
        assert "validation" in stage_by_name
        val_stage = stage_by_name["validation"]
        assert val_stage.get("feedback-to") == "implementation"
        assert val_stage.get("fresh") == "true"

        # stages without these fields should not have them
        assert "feedback-to" not in stage_by_name["ideation"]
        assert "agent" not in stage_by_name["ideation"]


class TestBuildBreakGlassFallback:
    """Implementation Note 4: Break-glass template is syntactically valid."""

    def test_break_glass_template_is_parseable(self):
        """The break-glass Agent() template in the runtime adapter is valid Python syntax."""
        import ast
        recovery_path = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-recovery.md"
        legacy_path = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime.md"
        runtime_path = recovery_path if recovery_path.exists() else legacy_path
        text = runtime_path.read_text()

        # Extract the break-glass code block
        in_break_glass = False
        in_code_block = False
        code_lines = []
        for line in text.splitlines():
            if "Break-Glass Manual Dispatch" in line:
                in_break_glass = True
                continue
            if in_break_glass and line.strip() == "```":
                if not in_code_block:
                    in_code_block = True
                    continue
                else:
                    break
            if in_code_block:
                code_lines.append(line)

        code_text = "\n".join(code_lines)
        assert len(code_lines) > 0, "Could not extract break-glass code block"
        # It should parse as a valid Python expression (function call)
        tree = ast.parse(code_text, mode="eval")
        assert isinstance(tree.body, ast.Call)
        assert tree.body.func.id == "Agent"


def _load_status_lib():
    import importlib.machinery
    loader = importlib.machinery.SourceFileLoader("_status_lib", str(STATUS_PATH))
    spec = importlib.util.spec_from_file_location("_status_lib", str(STATUS_PATH), loader=loader)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class TestParseStagesWithDefaultsModel:
    """AC-parser: parse_stages_with_defaults surfaces model on stages + defaults."""

    def test_parse_stages_with_defaults_surfaces_model(self, tmp_path):
        readme = tmp_path / "README.md"
        readme.write_text(
            "---\n"
            "commissioned-by: spacedock@test\n"
            "stages:\n"
            "  defaults:\n"
            "    worktree: false\n"
            "    model: haiku\n"
            "  states:\n"
            "    - name: backlog\n"
            "      initial: true\n"
            "    - name: work\n"
            "      model: opus\n"
            "    - name: done\n"
            "      terminal: true\n"
            "---\n"
        )
        mod = _load_status_lib()
        stages, defaults = mod.parse_stages_with_defaults(str(readme))
        assert defaults.get("model") == "haiku"
        stage_by_name = {s["name"]: s for s in stages}
        assert stage_by_name["work"].get("model") == "opus"
        # A stage without a declared model must not carry the key.
        assert "model" not in stage_by_name["backlog"]
        assert "model" not in stage_by_name["done"]


def _make_model_fixture(tmp_path: Path, defaults_model: str | None, stage_model: str | None, stage_name: str = "work") -> tuple[Path, Path]:
    """Create a workflow fixture with optional defaults.model and stage.model."""
    wf_dir = tmp_path / "workflow"
    wf_dir.mkdir(exist_ok=True)
    defaults_block = "    worktree: false\n    concurrency: 2\n"
    if defaults_model is not None:
        defaults_block += f"    model: {defaults_model}\n"
    stage_block = (
        "    - name: backlog\n"
        "      initial: true\n"
        f"    - name: {stage_name}\n"
    )
    if stage_model is not None:
        stage_block += f"      model: {stage_model}\n"
    stage_block += "    - name: done\n      terminal: true\n"
    (wf_dir / "README.md").write_text(
        "---\n"
        "commissioned-by: spacedock@test\n"
        "entity-label: task\n"
        "stages:\n"
        "  defaults:\n"
        f"{defaults_block}"
        "  states:\n"
        f"{stage_block}"
        "---\n"
        "\n"
        "## Stages\n\n"
        "### `backlog`\n\nHold.\n\n"
        f"### `{stage_name}`\n\nWork.\n\n"
        "### `done`\n\nTerminal.\n"
    )
    entity = wf_dir / "task.md"
    entity.write_text(
        "---\n"
        "id: 001\n"
        "title: Model task\n"
        "status: backlog\n"
        "worktree:\n"
        "---\n"
    )
    return wf_dir, entity


class TestBuildEmitsModel:
    """AC-build-emits, AC-precedence, AC-null: top-level model field in output JSON."""

    def test_build_emits_model_from_stage(self, tmp_path):
        wf_dir, entity = _make_model_fixture(tmp_path, defaults_model="opus", stage_model="haiku")
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert "model" in out
        assert out["model"] == "haiku"

    def test_build_precedence_stage_wins(self, tmp_path):
        wf_dir, entity = _make_model_fixture(tmp_path, defaults_model="opus", stage_model="sonnet")
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert out["model"] == "sonnet"

    def test_build_precedence_defaults(self, tmp_path):
        wf_dir, entity = _make_model_fixture(tmp_path, defaults_model="haiku", stage_model=None)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert out["model"] == "haiku"

    def test_build_precedence_null(self, tmp_path):
        wf_dir, entity = _make_model_fixture(tmp_path, defaults_model=None, stage_model=None)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert "model" in out
        assert out["model"] is None


class TestBuildEnumValidation:
    """AC-enum-validation: reject non-enum model values with field + enum list in stderr."""

    def test_build_rejects_non_enum_stage_model(self, tmp_path):
        wf_dir, entity = _make_model_fixture(
            tmp_path, defaults_model=None, stage_model="claude-haiku-4-5-20251001"
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode != 0
        assert "stages.states[" in result.stderr
        assert ".model" in result.stderr
        assert "must be one of: sonnet, opus, haiku" in result.stderr

    def test_build_rejects_non_enum_defaults_model(self, tmp_path):
        wf_dir, entity = _make_model_fixture(
            tmp_path, defaults_model="bogus", stage_model=None
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode != 0
        assert "stages.defaults.model" in result.stderr
        assert "must be one of: sonnet, opus, haiku" in result.stderr


class TestBuildVisibilityStderr:
    """AC-visibility: helper prints a one-line stderr notice when effective_model non-null."""

    def test_build_stderr_notice_on_haiku_defaults(self, tmp_path):
        wf_dir, entity = _make_model_fixture(tmp_path, defaults_model="haiku", stage_model=None)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        assert "[build] effective_model=haiku" in result.stderr
        assert "from defaults" in result.stderr

    def test_build_no_stderr_notice_when_null(self, tmp_path):
        wf_dir, entity = _make_model_fixture(tmp_path, defaults_model=None, stage_model=None)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "work",
            "checklist": ["1. Item"],
            "team_name": "t",
            "bare_mode": False,
        }
        result = run_build(wf_dir, inp)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        assert "effective_model=" not in result.stderr


class TestRuntimeAdapterModelProse:
    """AC-adapter-prose + AC-break-glass: Claude runtime adapter forwards model."""

    def test_claude_runtime_adapter_forwards_model(self):
        core_path = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md"
        legacy_path = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime.md"
        runtime_path = core_path if core_path.exists() else legacy_path
        text = runtime_path.read_text()
        # Emitted-fields enumeration includes `model`.
        assert "`model`" in text
        # Forwarding clause instructs FO to forward output.model as Agent model=.
        assert "output.model" in text
        assert "model=" in text

    def test_break_glass_template_has_conditional_model_slot(self):
        recovery_path = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-recovery.md"
        legacy_path = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime.md"
        runtime_path = recovery_path if recovery_path.exists() else legacy_path
        text = runtime_path.read_text()
        # Locate the break-glass code block and assert the model= slot appears inside.
        marker = "Break-Glass Manual Dispatch"
        idx = text.index(marker)
        block = text[idx:idx + 4000]
        assert "model=" in block
        # Conditional wording anchor — the slot must be documented as conditional.
        assert "conditional" in block.lower() or "omit" in block.lower()


class TestSharedCoreReuseModelMatch:
    """AC-reuse-match + AC-reuse-visibility: shared-core carries the reuse bullet and diagnostic."""

    def test_shared_core_has_reuse_model_match_bullet(self):
        shared_path = REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md"
        text = shared_path.read_text()
        assert "lookup_model(worker_name) == next_stage.effective_model" in text

    def test_shared_core_has_reuse_mismatch_diagnostic_anchor(self):
        shared_path = REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md"
        text = shared_path.read_text()
        assert "does not match next stage effective_model" in text


class TestSharedCoreProbeDiscipline:
    """AC-probe-discipline: shared-core carries the schema-first probe rule."""

    def test_shared_core_has_probe_discipline_anchor(self):
        shared_path = REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md"
        text = shared_path.read_text()
        assert "usage presence is not existence evidence" in text


_STANDING_MOD_BODY = """---
name: comm-officer
description: Standing prose-polishing teammate for this workflow
version: 0.1.0
standing: true
---

# Comm Officer

## Hook: startup

- `subagent_type: general-purpose`
- `name: comm-officer`
- `team_name: {current team}`
- `model: sonnet`

## Agent Prompt

You are the comm officer. Reply with polished prose.
"""


_NON_STANDING_MOD_BODY = """---
name: pr-merge
description: PR merge helper (not a standing teammate)
version: 0.1.0
---

# PR merge

## Hook: merge

- runs on validation PASSED

## Agent Prompt

Trivial.
"""


def _write_team_config(home: Path, team: str, members: list) -> Path:
    teams_dir = home / ".claude" / "teams" / team
    teams_dir.mkdir(parents=True, exist_ok=True)
    config_path = teams_dir / "config.json"
    config_path.write_text(json.dumps({"name": team, "members": members}))
    return config_path


def _run_build_with_home(wf_dir: Path, stdin_data: dict, home: Path, cwd: Path | None = None) -> subprocess.CompletedProcess:
    env = {**os.environ, "HOME": str(home)}
    return subprocess.run(
        [sys.executable, str(SCRIPT), "build", "--workflow-dir", str(wf_dir)],
        input=json.dumps(stdin_data),
        capture_output=True,
        text=True,
        cwd=cwd or wf_dir.parent,
        env=env,
    )


class TestBuildStandingTeammateEnumeration:
    """AC-13: `claude-team build` auto-enumerates alive standing teammates into prompts."""

    _SECTION_HEADING = "### Standing teammates available in your team"

    def test_build_emits_standing_section_when_alive(self, tmp_path):
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        mods_dir = wf_dir / "_mods"
        mods_dir.mkdir()
        (mods_dir / "comm-officer.md").write_text(_STANDING_MOD_BODY)
        _write_team_config(
            tmp_path,
            "active-team",
            [{"name": "team-lead"}, {"name": "comm-officer"}],
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "active-team",
            "bare_mode": False,
        }
        result = _run_build_with_home(wf_dir, inp, home=tmp_path)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert self._SECTION_HEADING in out["prompt"]
        assert "comm-officer" in out["prompt"]
        assert "Standing prose-polishing teammate" in out["prompt"]
        # Must be injected BEFORE the Completion Signal block so the literal
        # SendMessage(to="team-lead", ...) line stays at end-of-prompt.
        assert out["prompt"].index(self._SECTION_HEADING) < out["prompt"].index("### Completion Signal")

    def test_build_omits_standing_section_when_absent(self, tmp_path):
        """Standing mod declared but matching member NOT alive in team config."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        mods_dir = wf_dir / "_mods"
        mods_dir.mkdir()
        (mods_dir / "comm-officer.md").write_text(_STANDING_MOD_BODY)
        _write_team_config(
            tmp_path,
            "solo-team",
            [{"name": "team-lead"}],
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "solo-team",
            "bare_mode": False,
        }
        result = _run_build_with_home(wf_dir, inp, home=tmp_path)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert self._SECTION_HEADING not in out["prompt"]

    def test_build_omits_standing_section_in_bare_mode(self, tmp_path):
        """Bare mode has no team_name → no section even if mods exist."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        mods_dir = wf_dir / "_mods"
        mods_dir.mkdir()
        (mods_dir / "comm-officer.md").write_text(_STANDING_MOD_BODY)
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "bare_mode": True,
        }
        result = _run_build_with_home(wf_dir, inp, home=tmp_path)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert self._SECTION_HEADING not in out["prompt"]

    def test_build_omits_standing_section_when_no_standing_mods(self, tmp_path):
        """`_mods/` exists but contains only non-standing mods → no section."""
        wf_dir, entity = _make_workflow_fixture(tmp_path)
        mods_dir = wf_dir / "_mods"
        mods_dir.mkdir()
        (mods_dir / "pr-merge.md").write_text(_NON_STANDING_MOD_BODY)
        _write_team_config(
            tmp_path,
            "some-team",
            [{"name": "team-lead"}, {"name": "pr-merge"}],
        )
        inp = {
            "schema_version": 1,
            "entity_path": str(entity),
            "workflow_dir": str(wf_dir),
            "stage": "ideation",
            "checklist": ["1. Item"],
            "team_name": "some-team",
            "bare_mode": False,
        }
        result = _run_build_with_home(wf_dir, inp, home=tmp_path)
        assert result.returncode == 0, f"stderr: {result.stderr}"
        out = json.loads(result.stdout)
        assert self._SECTION_HEADING not in out["prompt"]


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
