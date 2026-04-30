#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Guards pytest collection config against recursive plugin/fixture collection.
# ABOUTME: Ensures default suite collection is scoped to tests/ and excludes fixture payload trees.

from __future__ import annotations

from pathlib import Path
import tomllib


REPO_ROOT = Path(__file__).resolve().parent.parent
PYPROJECT_PATH = REPO_ROOT / "pyproject.toml"
MAKEFILE_PATH = REPO_ROOT / "Makefile"


def pytest_ini_options() -> dict:
    data = tomllib.loads(PYPROJECT_PATH.read_text())
    return data["tool"]["pytest"]["ini_options"]


def test_pytest_default_collection_is_scoped_to_tests_dir():
    options = pytest_ini_options()
    assert options.get("testpaths") == ["tests"]


def test_pytest_default_collection_excludes_fixture_and_plugin_recursion():
    options = pytest_ini_options()
    norecursedirs = set(options.get("norecursedirs", []))
    assert "tests/fixtures" in norecursedirs
    assert "plugins" in norecursedirs


def test_static_make_target_excludes_all_live_runtime_markers():
    makefile = MAKEFILE_PATH.read_text()
    assert '-m "not live_claude and not live_codex and not live_pi"' in makefile
