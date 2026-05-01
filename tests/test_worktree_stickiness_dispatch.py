#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Offline test for `claude-team build` stickiness routing (AC-4).
# ABOUTME: When entity has stamped worktree, the dispatch prompt names that worktree even for `worktree: false` stages.

from __future__ import annotations

import json
import os
import subprocess
import sys
import textwrap
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parent.parent
CLAUDE_TEAM = REPO_ROOT / "skills" / "commission" / "bin" / "claude-team"


_README_STICKINESS = textwrap.dedent("""\
    ---
    commissioned-by: spacedock@test
    entity-label: task
    stages:
      defaults:
        worktree: false
      states:
        - name: initial
          initial: true
        - name: implementation
          worktree: true
        - name: validation
        - name: done
          terminal: true
    ---

    # Stickiness Dispatch Workflow

    ## Stages

    ### `initial`
    - **Inputs:** seed
    - **Outputs:** ready

    ### `implementation`
    - **Inputs:** ready entity
    - **Outputs:** code in worktree

    ### `validation`
    - **Inputs:** code in worktree
    - **Outputs:** verdict

    ### `done`
    Terminal.
    """)


def _materialize_git_root_with_worktree(tmp_path: Path) -> tuple[Path, Path, Path]:
    """Build a git project with a worktree containing the workflow + entity."""
    git_root = tmp_path / "project"
    git_root.mkdir()
    subprocess.run(["git", "init", "-q", str(git_root)], check=True)
    # initial commit so a worktree branch can be derived
    (git_root / "README.md").write_text("# project\n")
    subprocess.run(
        ["git", "-C", str(git_root), "add", "."], check=True, capture_output=True,
    )
    subprocess.run(
        ["git", "-C", str(git_root), "-c", "user.email=t@t", "-c", "user.name=t",
         "commit", "-q", "-m", "init"],
        check=True,
    )

    # Workflow on main.
    workflow_dir = git_root / "workflow"
    workflow_dir.mkdir()
    (workflow_dir / "README.md").write_text(_README_STICKINESS)

    entity_path = workflow_dir / "task-a.md"
    entity_path.write_text(textwrap.dedent("""\
        ---
        id: 001
        title: Task A
        status: validation
        worktree: .worktrees/spacedock-ensign-task-a
        ---

        Seed body.
        """))

    # Make a real worktree on disk (claude-team build validates os.path.isdir).
    worktree_dir = git_root / ".worktrees" / "spacedock-ensign-task-a"
    worktree_dir.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        ["git", "-C", str(git_root), "worktree", "add", "-b",
         "spacedock-ensign/task-a", str(worktree_dir)],
        check=True, capture_output=True,
    )

    return git_root, workflow_dir, entity_path


def test_build_routes_to_worktree_on_stickiness(tmp_path):
    """AC-4: with entity worktree set and next stage `worktree: false`, prompt names the worktree path."""
    git_root, workflow_dir, entity_path = _materialize_git_root_with_worktree(tmp_path)

    inp = {
        "schema_version": 1,
        "entity_path": str(entity_path),
        "workflow_dir": str(workflow_dir),
        "stage": "validation",  # declared worktree: false (inherits default)
        "checklist": ["1. Re-run validation against the worktree."],
        "team_name": "test-team",
        "bare_mode": False,
    }

    env = {**os.environ, "HOME": str(tmp_path / "home")}
    Path(env["HOME"]).mkdir(exist_ok=True)
    result = subprocess.run(
        [sys.executable, str(CLAUDE_TEAM), "build",
         "--workflow-dir", str(workflow_dir)],
        input=json.dumps(inp),
        capture_output=True, text=True,
        cwd=tmp_path,
        env=env,
    )
    assert result.returncode == 0, (
        f"build must succeed under stickiness routing; "
        f"stderr={result.stderr!r}, stdout={result.stdout[:400]!r}"
    )

    # cmd_build emits a JSON object with the assembled prompt.
    out = json.loads(result.stdout)
    prompt = out.get("prompt", "")
    expected_worktree_path = str(git_root / ".worktrees" / "spacedock-ensign-task-a")
    assert expected_worktree_path in prompt, (
        f"AC-4: validation-stage prompt must name the entity's stamped worktree "
        f"({expected_worktree_path!r}) under stickiness, even though the stage "
        f"declares worktree: false. Prompt was:\n{prompt}"
    )
