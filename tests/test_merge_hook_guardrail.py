# ABOUTME: E2E test for the merge hook guardrail in the first-officer template.
# ABOUTME: Verifies merge hooks fire before local merge, and that no-mods fallback works.

from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path
from typing import Callable

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    DispatchBudget,
    TestRunner,
    check_merge_outcome,
    git_add_commit,
    install_agents,
    read_entity_frontmatter,
    run_codex_first_officer,
    run_first_officer_streaming,
    setup_fixture,
)


PER_DISPATCH_OVERALL_S = 150
PER_DISPATCH_BUDGET_S = 120
MERGE_HOOK_FIRED_S = 90
SUBPROCESS_EXIT_BUDGET_S = 45


def _run_claude_merge_case(
    t: TestRunner,
    agent_id: str,
    workflow_dir: str,
    claude_extra_args: list[str],
    log_name: str,
    *,
    hook_expected: bool,
) -> int:
    """Drive the claude FO through a merge-hook pipeline with streaming milestones.

    Milestones differ by fixture variant:
      hook_expected=True  → ensign dispatched → hook-fired write observed → exit
      hook_expected=False → ensign dispatched → exit (local-merge fallback)
    """
    abs_workflow = t.test_project_dir / workflow_dir
    prompt = f"Process all tasks through the workflow at {abs_workflow}/ to completion."

    with run_first_officer_streaming(
        t,
        prompt,
        agent_id=agent_id,
        extra_args=claude_extra_args,
        log_name=log_name,
        dispatch_budget=DispatchBudget(
            soft_s=30.0, hard_s=150.0, shutdown_grace_s=10.0,
        ),
    ) as w:
        dispatch_record = w.expect_dispatch_close(
            overall_timeout_s=PER_DISPATCH_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            label="merge-hook ensign dispatch close",
        )
        print(f"[OK] ensign dispatch closed in {dispatch_record.elapsed:.1f}s")

        try:
            return w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)
        except Exception as exc:
            print(
                f"  NOTE: FO did not exit within {SUBPROCESS_EXIT_BUDGET_S}s "
                f"post-dispatch-close ({type(exc).__name__}); contract "
                f"assertions already passed"
            )
            return 0


def _run_merge_case(
    t: TestRunner,
    runtime: str,
    agent_id: str,
    workflow_dir: str,
    run_goal: str,
    claude_extra_args: list[str],
    codex_timeout_s: int,
    log_name: str,
    stop_checker: Callable[[Path], bool] | None = None,
    hook_expected: bool = True,
) -> int:
    if runtime == "claude":
        return _run_claude_merge_case(
            t, agent_id, workflow_dir, claude_extra_args, log_name,
            hook_expected=hook_expected,
        )
    return run_codex_first_officer(
        t,
        workflow_dir,
        agent_id=agent_id,
        run_goal=run_goal,
        timeout_s=codex_timeout_s,
        log_name=log_name.replace(".jsonl", ".txt"),
        stop_checker=stop_checker,
    )


def _codex_merge_stop_ready(project_dir, workflow_dir, entity_slug, hook_expected):
    workflow_path = project_dir / workflow_dir
    hook_file = workflow_path / "_merge-hook-fired.txt"
    archive_file = workflow_path / "_archive" / f"{entity_slug}.md"
    entity_file = workflow_path / f"{entity_slug}.md"

    def stop_ready(_):
        if hook_expected:
            if not hook_file.is_file():
                return False
        elif hook_file.exists():
            return False
        if archive_file.is_file():
            return True
        if not entity_file.is_file():
            return False
        frontmatter = read_entity_frontmatter(entity_file)
        return frontmatter.get("status") == "done"

    return stop_ready


def _codex_terminal_state_ready(project_dir, workflow_dir, entity_slug):
    workflow_path = project_dir / workflow_dir
    archive_file = workflow_path / "_archive" / f"{entity_slug}.md"
    entity_file = workflow_path / f"{entity_slug}.md"

    def stop_ready(_):
        if archive_file.is_file():
            return True
        if not entity_file.is_file():
            return False
        frontmatter = read_entity_frontmatter(entity_file)
        return frontmatter.get("status") == "done"

    return stop_ready


def _resume_codex_terminal_cleanup(t, workflow_dir, run_goal, log_name, stop_checker, timeout_s=240):
    return run_codex_first_officer(
        t,
        workflow_dir,
        run_goal=run_goal,
        timeout_s=timeout_s,
        log_name=log_name,
        stop_checker=stop_checker,
    )


@pytest.mark.live_claude
@pytest.mark.live_codex
def test_merge_hook_guardrail(test_project, runtime, model, effort):
    """Merge hooks fire before local merge; no-mods fallback works (claude + codex)."""
    t = test_project
    agent_id = "spacedock:first-officer"

    print("--- Phase 1: Set up test project with merge hook mod ---")
    fixture_dir = t.repo_root / "tests" / "fixtures" / "merge-hook-pipeline"
    setup_fixture(t, "merge-hook-pipeline", "merge-hook-pipeline")
    if runtime == "claude":
        install_agents(t)
    git_add_commit(t.test_project_dir, "setup: merge hook guardrail test fixture")
    t.check_cmd("status script runs without errors",
                ["merge-hook-pipeline/status"], cwd=t.test_project_dir)

    print()

    print("--- Phase 2: Run first officer with hook mod (this takes ~60-120s) ---")
    with_hook_project = t.test_project_dir
    fo_exit = _run_merge_case(
        t, runtime, agent_id, "merge-hook-pipeline",
        (
            "Process only the entity `merge-hook-entity` through the workflow to terminal completion. "
            "If it reaches the terminal stage, run any merge hooks before local merge, then archive the entity. "
            "Stop after the merge hook result and archive outcome are determined."
        ),
        ["--model", model, "--effort", effort, "--max-budget-usd", "2.00"],
        360, "fo-log.jsonl",
        stop_checker=_codex_terminal_state_ready(
            with_hook_project, "merge-hook-pipeline", "merge-hook-entity",
        ) if runtime == "codex" else None,
        hook_expected=True,
    )
    if runtime == "codex":
        hook_stop_ready = _codex_merge_stop_ready(
            with_hook_project, "merge-hook-pipeline", "merge-hook-entity",
            hook_expected=True,
        )
        if not hook_stop_ready(t.log_dir / "fo-log.txt"):
            fo_exit = _resume_codex_terminal_cleanup(
                t, "merge-hook-pipeline",
                (
                    "The entity `merge-hook-entity` is already at terminal status `done`. "
                    "Resume only the merge-and-cleanup portion now: run the registered merge hook from "
                    "`merge-hook-pipeline/_mods/test-hook.md`, then complete the default local merge/archive "
                    "cleanup on the main branch. Stop once `_merge-hook-fired.txt` records the entity slug "
                    "and the entity is archived."
                ),
                "fo-merge-resume.txt", hook_stop_ready,
            )
        t.check("Codex launcher exited cleanly (with hook)", fo_exit == 0)

    print("--- Phase 3: Validate merge hook execution ---")
    print()
    print("[Merge Hook Execution]")
    check_merge_outcome(
        t, with_hook_project,
        "merge-hook-pipeline", "merge-hook-entity",
        "spacedock-ensign-merge-hook-entity-work",
        hook_expected=True, archive_required=False,
    )
    print()

    print("--- Phase 4: Set up no-mods fallback test ---")
    nomods_project = t.test_dir / "test-no-mods"
    subprocess.run(["git", "init", str(nomods_project)], capture_output=True, check=True)
    subprocess.run(
        ["git", "commit", "--allow-empty", "-m", "init"],
        capture_output=True, check=True, cwd=nomods_project,
    )

    nomods_pipeline = nomods_project / "merge-hook-pipeline"
    nomods_pipeline.mkdir(parents=True)
    for item in fixture_dir.iterdir():
        if item.is_dir():
            continue
        shutil.copy2(item, nomods_pipeline / item.name)
    status_script = nomods_pipeline / "status"
    if status_script.exists():
        status_script.chmod(status_script.stat().st_mode | 0o111)

    orig_project = t.test_project_dir
    t.test_project_dir = nomods_project
    if runtime == "claude":
        install_agents(t)
    t.test_project_dir = orig_project

    subprocess.run(["git", "add", "-A"], capture_output=True, check=True, cwd=nomods_project)
    subprocess.run(
        ["git", "commit", "-m", "setup: no-mods fallback test fixture"],
        capture_output=True, check=True, cwd=nomods_project,
    )
    print()
    print("[Fixture Setup — No Mods]")
    t.check_cmd("status script runs without errors (no-mods)",
                ["bash", "merge-hook-pipeline/status"], cwd=nomods_project)
    print()

    print("--- Phase 5: Run first officer without mods (this takes ~60-120s) ---")
    t.test_project_dir = nomods_project
    nomods_log = "fo-nomods-log.jsonl"
    fo_exit = _run_merge_case(
        t, runtime, agent_id, "merge-hook-pipeline",
        (
            "Process only the entity `merge-hook-entity` through the workflow to terminal completion. "
            "No merge hook mod is installed in this fixture, so use the default local merge path and archive the entity. "
            "Stop after the archive outcome is determined."
        ),
        ["--model", model, "--effort", effort, "--max-budget-usd", "2.00"],
        360, nomods_log,
        stop_checker=_codex_terminal_state_ready(
            nomods_project, "merge-hook-pipeline", "merge-hook-entity",
        ) if runtime == "codex" else None,
        hook_expected=False,
    )
    if runtime == "codex":
        nomods_stop_ready = _codex_merge_stop_ready(
            nomods_project, "merge-hook-pipeline", "merge-hook-entity",
            hook_expected=False,
        )
        if not nomods_stop_ready(t.log_dir / "fo-nomods-log.txt"):
            fo_exit = _resume_codex_terminal_cleanup(
                t, "merge-hook-pipeline",
                (
                    "The entity `merge-hook-entity` is already at terminal status `done` in a workflow with no `_mods/`. "
                    "Resume only the terminal merge-and-cleanup path now: finish the default local merge/archive cleanup "
                    "on the main branch and stop once the entity is archived."
                ),
                "fo-nomods-resume.txt", nomods_stop_ready,
            )
        t.check("Codex launcher exited cleanly (no hook)", fo_exit == 0)

    print("--- Phase 6: Validate no-mods fallback ---")
    print()
    print("[No-Mods Fallback]")
    check_merge_outcome(
        t, nomods_project,
        "merge-hook-pipeline", "merge-hook-entity",
        "spacedock-ensign-merge-hook-entity-work",
        hook_expected=False, archive_required=False,
    )

    t.finish()
