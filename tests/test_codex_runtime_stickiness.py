# ABOUTME: Offline static-content test for the codex first-officer runtime stickiness reword.
# ABOUTME: AC-1b — asserts the per-dispatch numbered list carries the stickiness routing anchor.

from __future__ import annotations

from pathlib import Path


_REPO_ROOT = Path(__file__).resolve().parent.parent
_CODEX_RUNTIME = (
    _REPO_ROOT / "skills" / "first-officer" / "references" / "codex-first-officer-runtime.md"
)


def test_codex_runtime_stickiness_anchor():
    """AC-1b: codex runtime adapter names stickiness routing with a structural anchor."""
    text = _CODEX_RUNTIME.read_text()

    # Positive: the new anchor must appear in the file.
    assert "route the dispatch into that existing worktree" in text, (
        "AC-1b positive: codex runtime must carry the stickiness routing anchor"
    )

    # Negative: the prior contradicting prose must not appear.
    assert "If the stage is not marked for a worktree, stay on the main branch" not in text, (
        "AC-1b negative: prior worktree-mode-only prose must be removed"
    )
