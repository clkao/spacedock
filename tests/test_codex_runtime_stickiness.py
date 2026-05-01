# ABOUTME: Offline static-content tests for first-officer runtime adapter stickiness anchors.
# ABOUTME: Asserts the codex and claude runtime files carry the post-stickiness routing prose.

from __future__ import annotations

from pathlib import Path


_REPO_ROOT = Path(__file__).resolve().parent.parent
_CODEX_RUNTIME = (
    _REPO_ROOT / "skills" / "first-officer" / "references" / "codex-first-officer-runtime.md"
)
_CLAUDE_RUNTIME = (
    _REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime.md"
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


def test_claude_runtime_stickiness_anchor():
    """Surface 6: claude runtime feedback-cycle exemption names worktree-copy read source."""
    text = _CLAUDE_RUNTIME.read_text()

    assert "read from the worktree copy when `worktree:` is set" in text, (
        "surface-6 positive: claude runtime must carry the worktree-copy read anchor"
    )
