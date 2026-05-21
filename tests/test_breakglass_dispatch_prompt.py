#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Static-content lint for the FO break-glass dispatch prompt template.
# ABOUTME: AC-7 pins Skill-invoke directive presence and inlined-not-fetched shape.

from __future__ import annotations

from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
RUNTIME_ADAPTER = (
    REPO_ROOT
    / "skills"
    / "first-officer"
    / "references"
    / "claude-first-officer-runtime.md"
)


def _extract_breakglass_template_body() -> str:
    """Return the contents of the ``` fenced code block immediately under the
    `**Break-Glass Manual Dispatch` heading."""
    text = RUNTIME_ADAPTER.read_text()
    anchor = "**Break-Glass Manual Dispatch"
    idx = text.find(anchor)
    assert idx >= 0, "break-glass anchor not found in runtime adapter"
    rest = text[idx:]
    # First triple-backtick fence after the anchor opens the template body
    open_fence = rest.find("\n```\n")
    assert open_fence >= 0, "could not locate opening fence of break-glass template"
    after_open = rest[open_fence + len("\n```\n"):]
    close_fence = after_open.find("\n```\n")
    assert close_fence >= 0, "could not locate closing fence of break-glass template"
    return after_open[:close_fence]


def test_breakglass_template_prepends_skill_invoke_directive():
    """AC-7: the break-glass prompt body opens with `Skill(skill="spacedock:ensign")`."""
    body = _extract_breakglass_template_body()
    assert 'Skill(skill=\\"spacedock:ensign\\")' in body, (
        "break-glass template must invoke the Skill operating contract as its first action"
    )
    # The directive precedes the per-dispatch header
    skill_idx = body.find('Skill(skill=\\"spacedock:ensign\\")')
    header_idx = body.find("You are working on:")
    assert skill_idx >= 0 and header_idx >= 0
    assert skill_idx < header_idx, (
        "Skill-invoke directive must appear BEFORE the per-dispatch header"
    )


def test_breakglass_template_inlines_stage_definition_no_fetch_commands_block():
    """AC-7: the break-glass template carries no `### Fetch commands` block — it
    inlines the stage definition verbatim because the helper has failed and the
    ensign cannot rely on `claude-team show-stage-def`."""
    body = _extract_breakglass_template_body()
    assert "### Fetch commands" not in body, (
        "break-glass template must NOT reference `claude-team show-stage-def` — the "
        "helper is precisely what just failed"
    )
    assert "### Stage definition:" in body, (
        "break-glass template must inline the stage definition verbatim"
    )


def test_breakglass_explanatory_paragraph_enumerates_omissions_structurally():
    """AC-3 anchor: the explanatory paragraph adjacent to the break-glass template
    structurally lists the four omissions (worktree, feedback, scope, standing-teammates)
    plus the FO-forwarding warning and per-stage operational prose."""
    text = RUNTIME_ADAPTER.read_text()
    # Find the paragraph right after the template's closing fence
    body_anchor = "The break-glass template"
    para_idx = text.find(body_anchor)
    assert para_idx >= 0
    # Take ~600 chars after the anchor — paragraph length
    snippet = text[para_idx:para_idx + 1000]
    for needle in (
        "worktree",
        "feedback",
        "scope",
        "standing-teammates",
        "FO-forwarding",
        "per-stage operational",
    ):
        assert needle in snippet, (
            f"break-glass explanatory paragraph must enumerate `{needle}` "
            f"omission/warning structurally"
        )


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
