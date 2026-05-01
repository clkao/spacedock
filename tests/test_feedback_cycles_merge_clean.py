# ABOUTME: AC-6 — feedback cycles merge cleanly under stickiness; control reproduces PR #176/#177 conflict.
# ABOUTME: Treatment writes both the cycle entry and stage reports on the worktree branch; control writes the cycle entry on main.

from __future__ import annotations

import shutil
import subprocess
from pathlib import Path


_REPO_ROOT = Path(__file__).resolve().parent.parent
_FIXTURE_DIR = _REPO_ROOT / "tests" / "fixtures" / "feedback-cycles-merge"
_ENTITY_RELPATH = "workflow/buggy-task.md"


def _git(repo: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess:
    cmd = ["git", "-C", str(repo), "-c", "user.email=t@t", "-c", "user.name=t",
           "-c", "init.defaultBranch=main", *args]
    return subprocess.run(cmd, capture_output=True, text=True, check=check)


def _init_repo_with_base(repo: Path, base_fixture: Path) -> None:
    """Initialize a repo whose main branch carries the base entity at workflow/buggy-task.md."""
    repo.mkdir()
    _git(repo, "init", "-q", "-b", "main", check=False)
    # Some git versions don't support -b on init; force a rename if needed.
    _git(repo, "checkout", "-q", "-B", "main", check=False)

    entity_path = repo / _ENTITY_RELPATH
    entity_path.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(base_fixture, entity_path)
    _git(repo, "add", _ENTITY_RELPATH)
    _git(repo, "commit", "-q", "-m", "base: cycle-1 already on main")


def _commit_file(repo: Path, fixture: Path, message: str) -> None:
    """Overwrite the entity file with `fixture` content and commit on the current branch."""
    target = repo / _ENTITY_RELPATH
    shutil.copy2(fixture, target)
    _git(repo, "add", _ENTITY_RELPATH)
    _git(repo, "commit", "-q", "-m", message)


def _entity_text(repo: Path) -> str:
    return (repo / _ENTITY_RELPATH).read_text()


def _conflict_marker_count(text: str) -> int:
    return text.count("<<<<<<<")


def test_treatment_two_cycles_merge_clean(tmp_path):
    """AC-6 treatment: cycle entry + stage reports both on worktree branch → merge clean."""
    repo = tmp_path / "treatment"
    _init_repo_with_base(repo, _FIXTURE_DIR / "base_entity.md")

    # Branch off main, write the full treatment file (cycle entry + stage
    # reports), commit, return to main untouched.
    _git(repo, "checkout", "-q", "-b", "spacedock-ensign/buggy-task")
    _commit_file(
        repo,
        _FIXTURE_DIR / "treatment_worktree_cycle2_entry.md",
        "cycle: append cycle-2 entry and stage reports on worktree branch",
    )
    _git(repo, "checkout", "-q", "main")

    # Merge worktree branch into main (no fast-forward to mimic PR-style merge).
    merge = _git(
        repo, "merge", "--no-ff", "--no-edit", "spacedock-ensign/buggy-task",
        check=False,
    )
    assert merge.returncode == 0, (
        f"AC-6 treatment: merge must succeed cleanly under stickiness; "
        f"stderr={merge.stderr!r} stdout={merge.stdout!r}"
    )
    final_text = _entity_text(repo)
    assert _conflict_marker_count(final_text) == 0, (
        f"AC-6 treatment: merged entity must have zero `<<<<<<<` markers; got "
        f"{_conflict_marker_count(final_text)} marker(s) in entity file"
    )
    assert "Cycle 2 (2026-04-30)" in final_text
    assert "Stage Report: validation (cycle 2)" in final_text


def test_control_main_cycle_entry_reproduces_conflict(tmp_path):
    """AC-6 control: cycle entry on main while stage reports on worktree → merge conflict.

    This is the PR #176/#177 shape: the FO writes `### Feedback Cycles` to main,
    while the worktree-branch ensign writes stage reports into the same trailing
    region of the entity body. The two diverging line-adjacent edits race, and
    `git merge` either fails or leaves conflict markers in the entity file.
    """
    repo = tmp_path / "control"
    _init_repo_with_base(repo, _FIXTURE_DIR / "base_entity.md")

    # Main writes the cycle-2 entry directly (old rule).
    _commit_file(
        repo,
        _FIXTURE_DIR / "control_main_cycle2_entry.md",
        "fo: append cycle-2 entry on main (old rule)",
    )

    # The worktree branch was created from the base (before the main-side cycle
    # entry) and adds cycle-2 stage reports. Reset a fresh branch to point at
    # the base commit (HEAD~1) and commit the worktree variant.
    _git(repo, "checkout", "-q", "-b", "spacedock-ensign/buggy-task", "HEAD~1")
    _commit_file(
        repo,
        _FIXTURE_DIR / "worktree_cycle2_stage_reports_only.md",
        "ensign: append cycle-2 stage reports on worktree branch",
    )
    _git(repo, "checkout", "-q", "main")

    merge = _git(
        repo, "merge", "--no-ff", "--no-edit", "spacedock-ensign/buggy-task",
        check=False,
    )
    final_text = _entity_text(repo)
    conflicted = merge.returncode != 0 or _conflict_marker_count(final_text) > 0
    assert conflicted, (
        f"AC-6 control: cycle entry on main + stage report on worktree must "
        f"reproduce the PR #176/#177 conflict shape; got merge exit "
        f"{merge.returncode} with {_conflict_marker_count(final_text)} marker(s) "
        f"in entity file. The treatment test would be vacuously passing if this "
        f"control case ALSO merged clean — that would prove benign whitespace "
        f"separation, not the stickiness fix."
    )
