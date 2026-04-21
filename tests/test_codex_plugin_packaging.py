#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Contract checks for the Codex-first Spacedock plugin packaging surface.

from __future__ import annotations

import json
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parent.parent


def read_json(path: str) -> dict:
    return json.loads((REPO_ROOT / path).read_text())


def read_text(path: str) -> str:
    return (REPO_ROOT / path).read_text()


def normalized_json(path: str) -> str:
    return json.dumps(read_json(path), sort_keys=True, indent=2)


def test_codex_plugin_manifest_matches_approved_contract():
    manifest_path = REPO_ROOT / ".codex-plugin" / "plugin.json"
    assert manifest_path.is_file(), "expected .codex-plugin/plugin.json"

    manifest = read_json(".codex-plugin/plugin.json")

    assert manifest["name"] == "spacedock"
    assert manifest["version"] == "0.10.1"
    assert (
        manifest["description"]
        == "Turn directories of markdown files into structured workflows operated by AI agents"
    )
    assert manifest["author"] == {"name": "CL Kao"}
    assert manifest["repository"] == "https://github.com/clkao/spacedock"
    assert manifest["license"] == "Apache-2.0"
    assert manifest["keywords"] == [
        "workflow",
        "pipeline",
        "agents",
        "markdown",
        "automation",
    ]
    assert manifest["skills"] == "./skills/"


def test_codex_marketplace_matches_approved_contract():
    marketplace_path = REPO_ROOT / ".agents" / "plugins" / "marketplace.json"
    assert marketplace_path.is_file(), "expected .agents/plugins/marketplace.json"

    marketplace = read_json(".agents/plugins/marketplace.json")
    assert marketplace["name"] == "spacedock"
    assert marketplace["owner"] == {"name": "CL Kao"}
    assert marketplace["plugins"] == [
        {
            "name": "spacedock",
            "source": "./plugins/spacedock",
            "description": "Turn directories of markdown files into structured workflows operated by AI agents",
            "category": "workflow",
        }
    ]


def test_plugins_spacedock_symlink_resolves_to_repo_root():
    link_path = REPO_ROOT / "plugins" / "spacedock"
    assert link_path.is_symlink(), "expected plugins/spacedock to be a symlink"
    assert link_path.resolve() == REPO_ROOT
    assert (link_path / ".codex-plugin" / "plugin.json").is_file()


def test_legacy_plugin_manifest_is_a_synchronized_mirror():
    assert normalized_json(".claude-plugin/plugin.json") == normalized_json(
        ".codex-plugin/plugin.json"
    )


def test_legacy_marketplace_is_a_synchronized_mirror():
    assert normalized_json(".claude-plugin/marketplace.json") == normalized_json(
        ".agents/plugins/marketplace.json"
    )


def test_release_script_uses_codex_files_as_authority_and_updates_legacy_mirrors():
    text = read_text("scripts/release.sh")

    assert '.codex-plugin/plugin.json' in text
    assert '.agents/plugins/marketplace.json' in text
    assert '.claude-plugin/plugin.json' in text
    assert '.claude-plugin/marketplace.json' in text
    assert "sync_legacy_plugin_manifest" in text
    assert "sync_legacy_marketplace" in text


def test_docs_and_skill_surfaces_describe_codex_authority_and_legacy_compatibility():
    readme = read_text("README.md")
    commission = read_text("skills/commission/SKILL.md")
    refit = read_text("skills/refit/SKILL.md")
    debrief = read_text("skills/debrief/SKILL.md")

    assert ".codex-plugin/plugin.json" in readme
    assert ".agents/plugins/marketplace.json" in readme
    assert "plugins/spacedock" in readme
    assert "~/.agents/skills/spacedock" in readme
    assert "legacy" in readme.lower()

    for text in (commission, refit, debrief):
        assert ".codex-plugin/plugin.json" in text

    for text in (refit, debrief):
        assert ".agents/plugins/marketplace.json" in text
