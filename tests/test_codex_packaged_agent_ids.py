#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Unit-style coverage for Codex packaged logical worker ids in the Spacedock prototype.
#
# NOTE: Prompt checks in this file cover only Codex harness prompt discipline.
# Behavioral wait-policy proof belongs in live Codex E2E coverage.

from __future__ import annotations

import pytest

from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (
    build_codex_first_officer_invocation_prompt,
    build_codex_worker_bootstrap_prompt,
    resolve_codex_worker,
    resolve_skill_include,
)


def test_packaged_agent_id_resolves_to_skill_asset_with_safe_worker_key():
    resolved = resolve_codex_worker("spacedock:ensign")

    assert resolved["dispatch_agent_id"] == "spacedock:ensign"
    assert resolved["worker_key"] == "spacedock-ensign"
    assert resolved["asset_kind"] == "skill"
    assert resolved["asset_path"].name == "SKILL.md"
    assert resolved["asset_path"].parent.name == "ensign"


def test_unknown_packaged_agent_id_is_rejected():
    with pytest.raises(ValueError):
        resolve_codex_worker("spacedock:unknown")


def test_custom_non_spacedock_agent_id_falls_back_to_generic_worker():
    resolved = resolve_codex_worker("acme:reviewer")

    assert resolved["dispatch_agent_id"] == "acme:reviewer"
    assert resolved["worker_key"] == "acme-reviewer"
    assert resolved["asset_kind"] == "prompt"
    assert resolved["asset_name"] == "generic-worker"


def test_exec_harness_invokes_first_officer_skill_by_name_with_minimal_prompt():
    prompt = build_codex_first_officer_invocation_prompt("/tmp/example-workflow")

    assert "spacedock:first-officer" in prompt
    assert "/tmp/example-workflow" in prompt
    assert "workflow" in prompt
    assert "codex-first-officer-prompt.md" not in prompt
    assert "Treat that path as the explicit workflow target." not in prompt
    assert "wait_agent" not in prompt
    assert "spacedock-ensign" not in prompt
    assert "role_asset_name" not in prompt
    assert "fork_context=false" not in prompt
    assert "send_input" not in prompt


def test_exec_harness_can_pin_first_officer_to_local_skill_path():
    prompt = build_codex_first_officer_invocation_prompt(
        "/tmp/example-workflow",
        local_skill_path="/tmp/spacedock/skills/first-officer/SKILL.md",
        local_plugin_root="/tmp/spacedock",
    )

    assert "spacedock:first-officer" in prompt
    assert "/tmp/spacedock/skills/first-officer/SKILL.md" in prompt
    assert "/tmp/spacedock/skills/commission/bin/status" in prompt
    assert "wait_agent" not in prompt
    assert "send_input" not in prompt


def test_exec_harness_can_target_a_custom_logical_agent_id():
    prompt = build_codex_first_officer_invocation_prompt(
        "/tmp/example-workflow",
        agent_id="acme:first-officer",
    )

    assert "acme:first-officer" in prompt
    assert "spacedock:first-officer" not in prompt
    assert "wait_agent" not in prompt
    assert "spacedock-ensign" not in prompt


def test_packaged_worker_bootstrap_tells_worker_to_load_skill_contract():
    resolved = resolve_codex_worker("spacedock:ensign")

    prompt = build_codex_worker_bootstrap_prompt(
        resolved,
        workflow_dir=Path("/tmp/project/workflow"),
        entity_path=Path("/tmp/project/workflow/entity.md"),
        stage_name="implementation",
        stage_definition_text="Do the implementation work.",
        worktree_path=Path("/tmp/project/.spacedock/worktrees/spacedock-ensign-entity"),
        checklist=["Write the stage report", "Commit code changes"],
    )

    assert "packaged worker `spacedock:ensign`" in prompt
    assert "invoke the `spacedock:ensign` skill" in prompt
    assert "~/.agents/skills/{namespace}/agents/{name}.md" not in prompt
    assert "role_asset_kind: skill" in prompt
    assert "role_asset_name: ensign" in prompt
    assert "role_asset_path:" in prompt
    assert "spacedock:ensign" in prompt
    assert "worker_key: spacedock-ensign" in prompt


def test_skill_include_resolves_from_active_skill_directory_first(tmp_path):
    repo_root = tmp_path / "repo"
    skill_dir = repo_root / "skills" / "demo"
    refs_dir = skill_dir / "references"
    refs_dir.mkdir(parents=True)
    (skill_dir / "SKILL.md").write_text("@references/direct.md\n")
    (refs_dir / "direct.md").write_text("skill-local\n")
    repo_refs = repo_root / "references"
    repo_refs.mkdir(parents=True)
    (repo_refs / "direct.md").write_text("repo-root\n")

    resolved_path, source = resolve_skill_include(skill_dir / "SKILL.md", "references/direct.md", repo_root)

    assert resolved_path == refs_dir / "direct.md"
    assert source == "skill-relative"
    assert resolved_path.read_text() == "skill-local\n"


def test_skill_include_uses_bounded_repo_references_fallback(tmp_path):
    repo_root = tmp_path / "repo"
    skill_dir = repo_root / "skills" / "demo"
    skill_dir.mkdir(parents=True)
    (skill_dir / "SKILL.md").write_text("@references/fallback.md\n")
    repo_refs = repo_root / "references"
    repo_refs.mkdir(parents=True)
    (repo_refs / "fallback.md").write_text("repo-root\n")

    resolved_path, source = resolve_skill_include(skill_dir / "SKILL.md", "references/fallback.md", repo_root)

    assert resolved_path == repo_refs / "fallback.md"
    assert source == "bounded-fallback"
    assert resolved_path.read_text() == "repo-root\n"


def test_skill_include_errors_when_missing_everywhere(tmp_path):
    repo_root = tmp_path / "repo"
    skill_dir = repo_root / "skills" / "demo"
    skill_dir.mkdir(parents=True)
    (skill_dir / "SKILL.md").write_text("@references/missing.md\n")

    with pytest.raises(FileNotFoundError) as excinfo:
        resolve_skill_include(skill_dir / "SKILL.md", "references/missing.md", repo_root)

    message = str(excinfo.value)
    assert "references/missing.md" in message
    assert "SKILL.md" in message


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
