#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live proof that the local Pi ensign skill can run in a git worktree and reopen the same session for follow-up.

from __future__ import annotations

from pathlib import Path
import subprocess
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    PiLogParser,
    build_pi_ensign_invocation_prompt,
    run_pi_ensign,
)


@pytest.mark.live_pi
@pytest.mark.serial
def test_pi_local_ensign_skill_supports_worktree_reopened_follow_up(test_project):
    t = test_project
    entity_path = t.test_project_dir / "task-218.md"
    entity_path.write_text(
        "---\n"
        "id: 218\n"
        "title: Pi ensign local skill reuse\n"
        "status: implementation\n"
        "worktree: .worktrees/218-implementation-ensign\n"
        "---\n\n"
        "Initial body.\n"
    )
    subprocess.run(["git", "add", "task-218.md"], cwd=t.test_project_dir, check=True)
    subprocess.run(["git", "commit", "-m", "setup: add task entity"], cwd=t.test_project_dir, check=True)
    setup_head = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=t.test_project_dir,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()

    worktree_path = t.test_project_dir / ".worktrees" / "218-implementation-ensign"
    subprocess.run(
        ["git", "worktree", "add", "-b", "spacedock-ensign-218-implementation", str(worktree_path)],
        cwd=t.test_project_dir,
        check=True,
    )
    worktree_entity_path = worktree_path / "task-218.md"
    session_dir = t.test_dir / "pi-ensign-sessions"
    ensign_skill_path = t.repo_root / "skills" / "ensign"

    first_prompt = build_pi_ensign_invocation_prompt(
        worktree_path,
        worktree_entity_path,
        "implementation",
        "Append the line `Implementation note: local pi ensign ran inside the assigned worktree.` to the entity body. Then append the implementation stage report using the checklist below.",
        worktree_path,
        [
            "Append the implementation note to the entity body",
            "Append the implementation stage report",
            "Commit the work",
        ],
        local_skill_path=ensign_skill_path,
        local_plugin_root=t.repo_root,
        worker_label="218-implementation/Ensign",
    )

    first_exit = run_pi_ensign(
        t,
        first_prompt,
        session_dir=session_dir,
        cwd=worktree_path,
        log_name="pi-ensign-implementation.jsonl",
        timeout_s=180,
    )
    t.check("initial ensign run exited cleanly", first_exit == 0)

    first_log = PiLogParser(t.log_dir / "pi-ensign-implementation.jsonl")
    session_id = first_log.session_id()
    t.check("initial ensign run exposed a session id", bool(session_id))

    worktree_entity_text = worktree_entity_path.read_text()
    t.check(
        "implementation run updated the worktree entity body",
        "Implementation note: local pi ensign ran inside the assigned worktree." in worktree_entity_text,
    )
    t.check(
        "implementation run appended a worktree stage report",
        "## Stage Report: implementation" in worktree_entity_text,
    )
    t.check(
        "main checkout entity stayed unchanged during worktree implementation",
        "Implementation note: local pi ensign ran inside the assigned worktree." not in entity_path.read_text(),
    )

    worktree_head_after_first = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=worktree_path,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    main_head_after_first = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=t.test_project_dir,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    t.check("implementation run committed on the worktree branch", worktree_head_after_first != setup_head)
    t.check("implementation run left the main checkout head unchanged", main_head_after_first == setup_head)

    second_prompt = build_pi_ensign_invocation_prompt(
        worktree_path,
        worktree_entity_path,
        "validation",
        "Append the line `Validation note: reopened session reuse confirmed from the same worktree.` to the entity body. Then append the validation stage report using the checklist below.",
        worktree_path,
        [
            "Append the validation note to the entity body",
            "Append the validation stage report",
            "Commit the work",
        ],
        local_skill_path=ensign_skill_path,
        local_plugin_root=t.repo_root,
        worker_label="218-validation/Ensign",
    )

    second_exit = run_pi_ensign(
        t,
        second_prompt,
        session_dir=session_dir,
        session=session_id,
        cwd=worktree_path,
        log_name="pi-ensign-validation.jsonl",
        timeout_s=180,
    )
    t.check("reopened ensign run exited cleanly", second_exit == 0)

    second_log = PiLogParser(t.log_dir / "pi-ensign-validation.jsonl")
    t.check("reopened ensign run stayed on the same session id", second_log.session_id() == session_id)

    updated_worktree_entity_text = worktree_entity_path.read_text()
    t.check(
        "reopened ensign run applied the follow-up change in the worktree entity",
        "Validation note: reopened session reuse confirmed from the same worktree." in updated_worktree_entity_text,
    )
    t.check(
        "reopened ensign run appended the second worktree stage report",
        "## Stage Report: validation" in updated_worktree_entity_text,
    )
    t.check(
        "main checkout entity still does not contain the follow-up worktree changes",
        "Validation note: reopened session reuse confirmed from the same worktree." not in entity_path.read_text(),
    )

    worktree_head_after_second = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=worktree_path,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    main_head_after_second = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=t.test_project_dir,
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    t.check("validation follow-up committed more work on the worktree branch", worktree_head_after_second != worktree_head_after_first)
    t.check("validation follow-up still left the main checkout head unchanged", main_head_after_second == setup_head)

    t.finish()
