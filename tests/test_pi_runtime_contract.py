#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Static contract checks for the first Pi runtime slice.

from __future__ import annotations

from pathlib import Path
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import TestRunner, assembled_agent_content  # noqa: E402


REPO_ROOT = Path(__file__).resolve().parent.parent


def read_text(path: str) -> str:
    return (REPO_ROOT / path).read_text()


def test_first_officer_and_ensign_skills_advertise_pi_runtime_selection():
    fo_text = read_text("skills/first-officer/SKILL.md")
    ensign_text = read_text("skills/ensign/SKILL.md")

    assert "pi-first-officer-runtime.md" in fo_text
    assert "PI_CODING_AGENT_DIR" in fo_text or "pi runtime" in fo_text.lower()

    assert "pi-ensign-runtime.md" in ensign_text
    assert "PI_CODING_AGENT_DIR" in ensign_text or "pi runtime" in ensign_text.lower()


def test_assembled_agent_content_supports_pi_runtime_contracts():
    t = TestRunner("pi runtime contract", keep_test_dir=False)

    fo_text = assembled_agent_content(t, "first-officer", runtime="pi")
    ensign_text = assembled_agent_content(t, "ensign", runtime="pi")

    assert "# Pi First Officer Runtime" in fo_text
    assert "first-class runtime target" in fo_text
    assert "scripts/pi_worker_runtime.py" in fo_text
    assert "scripts/pi_session_registry.py" in fo_text
    assert "# Pi Ensign Runtime" in ensign_text
    assert "session-backed worker" in ensign_text
    assert "stop immediately" in ensign_text
    assert "reopening this same session later" in ensign_text


def test_pytest_runtime_option_accepts_pi():
    conftest_text = read_text("tests/conftest.py")

    assert 'choices=["claude", "codex", "pi"]' in conftest_text


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
