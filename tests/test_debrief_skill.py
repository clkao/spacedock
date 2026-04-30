# ABOUTME: Static prose-guard tests for skills/debrief/SKILL.md Phase 2e fallback chain.
# ABOUTME: Covers AC2 (primary/legacy/degraded prose) and AC3 (degraded-mode self-annotation).

from __future__ import annotations

import re
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
DEBRIEF_SKILL = REPO_ROOT / "skills" / "debrief" / "SKILL.md"
NO_LOCAL_STATUS_FIXTURE = REPO_ROOT / "tests" / "fixtures" / "debrief-no-local-status"


def read(path: Path) -> str:
    return path.read_text()


def phase_2e_body() -> str:
    text = read(DEBRIEF_SKILL)
    match = re.search(
        r"^### 2e\. What's next$(.*?)^### ",
        text, re.MULTILINE | re.DOTALL,
    )
    assert match, "Phase 2e section not found in debrief SKILL.md"
    return match.group(1)


class TestPhase2ePrimaryInvocation:
    """AC2: plugin-shipped status named as the primary invocation."""

    def test_plugin_shipped_path_present(self):
        body = phase_2e_body()
        assert "{spacedock_plugin_dir}/skills/commission/bin/status" in body

    def test_plugin_shipped_invoked_with_workflow_dir_flag(self):
        body = phase_2e_body()
        assert "--workflow-dir {dir}" in body

    def test_local_status_not_unconditional_primary(self):
        """The first invocation block must reference the plugin-shipped path,
        not the bare {dir}/status. AC2 explicitly forbids presenting
        {dir}/status as the unconditional primary."""
        body = phase_2e_body()
        plugin_idx = body.find("{spacedock_plugin_dir}/skills/commission/bin/status")
        local_idx = body.find("{dir}/status")
        assert plugin_idx != -1, "plugin-shipped path missing"
        assert plugin_idx < local_idx or local_idx == -1, (
            "{dir}/status appears before the plugin-shipped path — "
            "would imply local status is the primary invocation"
        )


class TestPhase2eLegacyFallback:
    """AC2: legacy {dir}/status fallback documented for spacedock<=0.10.x."""

    def test_legacy_fallback_mentions_local_status(self):
        body = phase_2e_body()
        assert "{dir}/status" in body

    def test_legacy_fallback_marked_as_back_compat(self):
        body = phase_2e_body()
        legacy_terms = ("back-compat", "legacy")
        assert any(term in body.lower() for term in legacy_terms), (
            "legacy fallback should be marked as back-compat / legacy"
        )


class TestPhase2eDegradedFallback:
    """AC2 + AC3: frontmatter-scan degraded fallback with self-annotation."""

    def test_degraded_fallback_licenses_frontmatter_scan(self):
        body = phase_2e_body()
        assert "frontmatter" in body.lower()
        assert "scan" in body.lower() or "read" in body.lower()

    def test_degraded_fallback_covers_three_buckets(self):
        body = phase_2e_body()
        lowered = body.lower()
        assert "dispatchable" in lowered
        assert "gate" in lowered
        assert "worktree" in lowered

    def test_degraded_mode_self_annotation_present(self):
        """AC3: rendered output gets a one-line annotation when degraded
        path is taken so the captain knows data was reconstructed."""
        body = phase_2e_body()
        assert "reconstructed from entity frontmatter" in body
        assert "no status helper available" in body


class TestNoLocalStatusFixture:
    """Fixture sanity: the regression fixture matches the failure shape."""

    def test_fixture_directory_exists(self):
        assert NO_LOCAL_STATUS_FIXTURE.is_dir()

    def test_fixture_has_no_local_status_executable(self):
        """Crux of the fixture — no {dir}/status file. If this passes
        accidentally, the fixture no longer exercises the failure shape."""
        assert not (NO_LOCAL_STATUS_FIXTURE / "status").exists()

    def test_fixture_has_readme_with_commissioned_by(self):
        readme = NO_LOCAL_STATUS_FIXTURE / "README.md"
        assert readme.is_file()
        assert "commissioned-by: spacedock@" in readme.read_text()

    def test_fixture_has_entities_spanning_three_buckets(self):
        entity_files = sorted(
            p for p in NO_LOCAL_STATUS_FIXTURE.glob("*.md") if p.name != "README.md"
        )
        assert len(entity_files) >= 3
        bodies = [p.read_text() for p in entity_files]
        joined = "\n".join(bodies)
        assert "status: backlog" in joined
        assert "status: review" in joined
        assert "status: build" in joined
        assert "worktree: .worktrees/" in joined
