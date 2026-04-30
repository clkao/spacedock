# ABOUTME: Static checks that the commission README heredoc is portable.
# ABOUTME: Asserts no machine-specific paths or status invocations and that the FO entrypoint is documented.

from __future__ import annotations

import difflib
import re
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parent.parent
SKILL_PATH = REPO_ROOT / "skills" / "commission" / "SKILL.md"

HEREDOC_FENCE_RE = re.compile(r"^````", re.MULTILINE)


def read_skill() -> str:
    return SKILL_PATH.read_text()


def heredoc_bounds(text: str) -> tuple[int, int]:
    """Return (open_line, close_line) 1-indexed of the README heredoc fence."""
    fences = [m.start() for m in HEREDOC_FENCE_RE.finditer(text)]
    assert len(fences) >= 2, "Expected at least two ```` fences in SKILL.md"
    open_idx, close_idx = fences[0], fences[1]
    open_line = text.count("\n", 0, open_idx) + 1
    close_line = text.count("\n", 0, close_idx) + 1
    return open_line, close_line


def heredoc_body(text: str) -> str:
    """Return the body between the first pair of ```` fences."""
    fences = list(HEREDOC_FENCE_RE.finditer(text))
    assert len(fences) >= 2
    start = text.find("\n", fences[0].start()) + 1
    end = fences[1].start()
    return text[start:end]


def test_heredoc_has_no_machine_specific_paths():
    """AC-1: Generated README contains no machine-specific path interpolations."""
    body = heredoc_body(read_skill())
    forbidden = ["{spacedock_plugin_dir}", ".claude/plugins/cache"]
    found = [pat for pat in forbidden if pat in body]
    assert not found, (
        f"README heredoc must not contain machine-specific path patterns; "
        f"found: {found}"
    )


def test_heredoc_has_no_status_invocation_prose():
    """AC-2: Generated README contains no status invocation prose."""
    body = heredoc_body(read_skill())
    assert "bin/status" not in body, (
        "README heredoc must not document `bin/status` invocations — "
        "status usage is encapsulated in the first-officer skill"
    )


def test_heredoc_workflow_state_section_points_to_first_officer():
    """AC-3: Generated README's runtime-entrypoint section is the canonical FO-invocation prose."""
    body = heredoc_body(read_skill())
    match = re.search(
        r"^## Workflow State\n(.*?)(?=^## )",
        body,
        re.MULTILINE | re.DOTALL,
    )
    assert match, "README heredoc must contain a `## Workflow State` section"
    section = match.group(1)
    assert "claude --agent spacedock:first-officer" in section, (
        "## Workflow State section must mention `claude --agent spacedock:first-officer`"
    )
    # Sanity: no other invocation examples in this section.
    assert "bin/status" not in section
    assert "{spacedock_plugin_dir}" not in section


def test_setup_prose_interpolations_remain_outside_heredoc():
    """AC-5: Setup-time interpolations are preserved verbatim and live outside the README heredoc."""
    text = read_skill()
    _, close_line = heredoc_bounds(text)

    expected_setup_lines = [
        'cp "{spacedock_plugin_dir}/mods/pr-merge.md" {dir}/_mods/pr-merge.md',
        "Read the first-officer agent file at `{spacedock_plugin_dir}/agents/first-officer.md`",
        "Show the current state of the workflow with `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}`",
    ]

    lines = text.splitlines()
    for needle in expected_setup_lines:
        matches = [i + 1 for i, line in enumerate(lines) if needle in line]
        assert matches, f"Expected setup-time line not found: {needle!r}"
        for line_no in matches:
            assert line_no > close_line, (
                f"Setup-time line {line_no} ({needle!r}) must live outside the "
                f"README heredoc (closes at line {close_line})"
            )


def test_refit_show_diff_against_old_readme_surfaces_drift():
    """AC-4: Refit's existing Show-Diff against an old README surfaces the constraint drift.

    Builds an old-style README fixture containing the three status-invocation snippets
    plus the grep-l line, then runs unified diff against the current commission heredoc
    template (the same content refit's Phase 3b would generate). Asserts deletions
    contain {spacedock_plugin_dir} and bin/status lines and additions contain the
    canonical FO-invocation paragraph.
    """
    old_readme = (
        "## Workflow State\n"
        "\n"
        "View the workflow overview:\n"
        "\n"
        "```bash\n"
        "{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir}\n"
        "```\n"
        "\n"
        "Include archived ideas with `--archived`:\n"
        "\n"
        "```bash\n"
        "{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir} --archived\n"
        "```\n"
        "\n"
        "Find dispatchable ideas ready for their next stage:\n"
        "\n"
        "```bash\n"
        "{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir} --next\n"
        "```\n"
        "\n"
        "Find ideas in a specific stage:\n"
        "\n"
        "```bash\n"
        'grep -l "status: ideation" {dir}/*.md\n'
        "```\n"
        "\n"
        "## Next Section\n"
    )

    body = heredoc_body(read_skill())
    match = re.search(
        r"(^## Workflow State\n.*?)(?=^## )",
        body,
        re.MULTILINE | re.DOTALL,
    )
    assert match, "README heredoc must contain a `## Workflow State` section"
    new_section = match.group(1) + "## Next Section\n"

    diff = list(
        difflib.unified_diff(
            old_readme.splitlines(keepends=True),
            new_section.splitlines(keepends=True),
            fromfile="README.md (existing)",
            tofile="README.md (current template)",
            n=3,
        )
    )

    deletions = [line for line in diff if line.startswith("-") and not line.startswith("---")]
    additions = [line for line in diff if line.startswith("+") and not line.startswith("+++")]

    assert any("{spacedock_plugin_dir}" in line for line in deletions), (
        "Diff deletions must include at least one `{spacedock_plugin_dir}` line"
    )
    assert any("bin/status" in line for line in deletions), (
        "Diff deletions must include at least one `bin/status` line"
    )
    assert any("claude --agent spacedock:first-officer" in line for line in additions), (
        "Diff additions must include the canonical FO-invocation paragraph"
    )
