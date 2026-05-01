# ABOUTME: Offline static-content test for stickiness wording in commission templates.
# ABOUTME: AC-3 — all three templates must document stickiness in their `worktree` frontmatter row.

from __future__ import annotations

import re
from pathlib import Path

import pytest


_REPO_ROOT = Path(__file__).resolve().parent.parent
_TEMPLATES = [
    _REPO_ROOT / "skills" / "commission" / "references" / "templates" / name
    for name in ("development.md", "experiment.md", "refinement.md")
]


_WORKTREE_ROW_RE = re.compile(
    r"\|\s*`worktree`\s*\|[^|]*\|([^\n|]*)\|",
)


@pytest.mark.parametrize("template_path", _TEMPLATES, ids=lambda p: p.name)
def test_template_worktree_row_documents_stickiness(template_path):
    """AC-3: each commission template's worktree row carries `Once set on first dispatch`."""
    text = template_path.read_text()
    match = _WORKTREE_ROW_RE.search(text)
    assert match, f"{template_path.name} must contain a `worktree` frontmatter row"
    description_cell = match.group(1)
    assert "Once set on first dispatch" in description_cell, (
        f"{template_path.name} worktree row must carry the stickiness anchor; "
        f"got cell: {description_cell!r}"
    )
