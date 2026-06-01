#!/usr/bin/env -S uv run --with pytest python
# ///
# requires-python = ">=3.10"
# ///
# ABOUTME: Static checks for the spacedock:debrief skill's workflow status and schema contracts.

from __future__ import annotations

from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parent.parent
DEBRIEF_SKILL = REPO_ROOT / "skills" / "debrief" / "SKILL.md"


def read_debrief_skill() -> str:
    return DEBRIEF_SKILL.read_text()


def test_debrief_uses_packaged_status_fallback_when_workflow_status_missing():
    text = read_debrief_skill()

    assert "{dir}/status" in text
    assert "skills/commission/bin/status" in text
    assert "--workflow-dir {dir}" in text
    assert "if `{dir}/status` exists and is executable" in text
    assert "Do not require a workflow-local status helper" in text


def test_debrief_template_emits_schema_version_one():
    text = read_debrief_skill()
    template_start = text.index("Write the debrief to `{dir}/_debriefs/{date}-{sequence:02d}.md`")
    template = text[template_start:]

    assert "schema_version: 1" in template
    assert template.index("schema_version: 1") < template.index("session-date: {YYYY-MM-DD}")
