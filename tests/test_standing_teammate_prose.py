#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Static grep tests for standing-teammate prose in Claude runtime adapter + shared-core.
# ABOUTME: Covers AC-8 (adapter prose), AC-9 (shared-core concept), AC-10 (FO routing prose).

from __future__ import annotations

import re
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
CLAUDE_RUNTIME = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md"
SHARED_CORE = REPO_ROOT / "skills" / "first-officer" / "references" / "first-officer-shared-core.md"
COMM_OFFICER_MOD = REPO_ROOT / "docs" / "plans" / "_mods" / "comm-officer.md"


def read(path: Path) -> str:
    return path.read_text()


class TestClaudeAdapterProse:
    """AC-8: Claude runtime adapter documents the standing teammate spawn pass."""

    def test_heading_present(self):
        text = read(CLAUDE_RUNTIME)
        assert re.search(r"^### Standing teammate spawn pass$", text, re.MULTILINE)

    def test_helper_invocation_mentioned(self):
        text = read(CLAUDE_RUNTIME)
        assert "claude-team spawn-standing" in text

    def test_verbatim_discipline_phrasing(self):
        """Same 'forward verbatim' discipline used for claude-team build output."""
        text = read(CLAUDE_RUNTIME)
        assert re.search(r"[Ff]orward.*verbatim", text)

    def test_already_alive_status_handling(self):
        text = read(CLAUDE_RUNTIME)
        assert "already-alive" in text


class TestSharedCoreConcept:
    """AC-9: shared-core has a top-level `## Standing Teammates` concept section."""

    def test_section_heading_present(self):
        text = read(SHARED_CORE)
        assert re.search(r"^## Standing Teammates$", text, re.MULTILINE)

    def test_first_boot_wins_anchor(self):
        text = read(SHARED_CORE)
        assert "first-boot-wins" in text

    def test_team_scope_lifecycle_anchor(self):
        text = read(SHARED_CORE)
        assert "team-scope lifecycle" in text

    def test_routing_contract_anchor(self):
        text = read(SHARED_CORE)
        assert "routing contract" in text

    def test_declaration_format_anchor(self):
        text = read(SHARED_CORE)
        assert "declaration format" in text


class TestFORoutingProse:
    """AC-10: shared-core Dispatch section mentions comm-officer + lists out-of-scope cases."""

    def test_comm_officer_mentioned(self):
        text = read(SHARED_CORE)
        assert "comm-officer" in text

    def test_captain_chat_out_of_scope(self):
        text = read(SHARED_CORE)
        # The routing prose MUST explicitly exclude live captain-chat replies.
        assert re.search(r"out.?of.?scope.*captain-chat", text, re.IGNORECASE | re.DOTALL) or \
               re.search(r"[Oo]ut of scope[^\n]*captain[- ]chat", text)

    def test_operational_statuses_out_of_scope(self):
        text = read(SHARED_CORE)
        assert "operational statuses" in text or "short operational" in text

    def test_member_exists_check_mentioned(self):
        text = read(SHARED_CORE)
        assert "member_exists" in text

    def test_build_time_auto_enumeration_anchor(self):
        """AC-10 cycle-2 extension: prose acknowledges workers auto-discover
        standing teammates via their build-time prompt section, so the FO
        does not need to add per-dispatch routing opt-ins manually."""
        text = read(SHARED_CORE)
        assert "discover the same teammates automatically via their build-time prompt section" in text
        assert "### Standing teammates available in your team" in text


def _extract_section(text: str, heading: str) -> str:
    """Return the body of a `## {heading}` section, up to the next `## ` heading or EOF."""
    lines = text.splitlines()
    start = None
    for i, line in enumerate(lines):
        if line.strip() == heading:
            start = i
            break
    if start is None:
        return ""
    end = len(lines)
    for i in range(start + 1, len(lines)):
        if lines[i].startswith("## "):
            end = i
            break
    return "\n".join(lines[start + 1:end])


class TestCommOfficerRoutingUsage:
    """AC-6, AC-7: pilot comm-officer mod has a well-formed `## Routing Usage` section."""

    def test_routing_usage_heading_present(self):
        text = read(COMM_OFFICER_MOD)
        assert re.search(r"^## Routing Usage$", text, re.MULTILINE), (
            "comm-officer mod missing `## Routing Usage` heading"
        )

    def test_routing_usage_within_line_cap(self):
        """AC-6: body ≤ 25 lines (heading excluded) per the soft cap."""
        body = _extract_section(read(COMM_OFFICER_MOD), "## Routing Usage")
        assert body.strip(), "Routing Usage section body is empty"
        body_lines = body.splitlines()
        assert len(body_lines) <= 25, (
            f"Routing Usage body is {len(body_lines)} lines, exceeds 25-line soft cap"
        )

    def test_four_trigger_phrases_present(self):
        """AC-6: all four caller patterns named by their trigger phrases."""
        body = _extract_section(read(COMM_OFFICER_MOD), "## Routing Usage")
        assert "polish this file" in body, "missing file-in-place trigger phrase"
        assert "polish and write to" in body, "missing polish-and-write trigger phrase"
        assert "polish and edit" in body, "missing polish-and-edit trigger phrase"
        # Text-passthrough is the implicit default; must be named in the section.
        assert re.search(r"text.?passthrough", body, re.IGNORECASE), (
            "missing text-passthrough pattern name"
        )

    def test_scope_discipline_stays_in_routing_guidance(self):
        """AC-7: scope-discipline keywords live in `## Routing guidance`, not `## Routing Usage`."""
        text = read(COMM_OFFICER_MOD)
        guidance = _extract_section(text, "## Routing guidance (for FO and ensigns)")
        usage = _extract_section(text, "## Routing Usage")

        # Three scope-discipline keywords that the AC spec calls out as the
        # shape guidance owns. The pilot mod phrases "captain-chat" as
        # "Direct chat replies to the captain" — match the literal prose.
        scope_keywords = ["does NOT polish", "Direct chat replies", "operational statuses"]
        for kw in scope_keywords:
            assert kw in guidance, (
                f"scope-discipline keyword {kw!r} missing from `## Routing guidance`"
            )
            assert kw not in usage, (
                f"scope-discipline keyword {kw!r} leaked into `## Routing Usage` — "
                f"move it back to `## Routing guidance`"
            )


class TestClaudeAdapterListStandingPath:
    """AC-12: Claude runtime adapter documents `claude-team list-standing`, drops inline grep."""

    def test_list_standing_documented(self):
        text = read(CLAUDE_RUNTIME)
        assert "claude-team list-standing" in text, (
            "adapter must document the `claude-team list-standing` subcommand"
        )

    def test_grep_each_mod_file_invitation_removed(self):
        text = read(CLAUDE_RUNTIME)
        assert "grep each mod file" not in text, (
            "adapter still invites an inline `grep each mod file` — "
            "replace with `claude-team list-standing`"
        )

    def test_verbatim_discipline_preserved_for_spawn_step(self):
        """Step 4 'Forward that spec verbatim' discipline must remain intact."""
        text = read(CLAUDE_RUNTIME)
        assert re.search(r"[Ff]orward.*verbatim", text), (
            "adapter lost the 'forward verbatim' discipline phrasing"
        )
        # The spawn-standing forwarding instruction itself must still be there.
        assert "claude-team spawn-standing" in text


if __name__ == "__main__":
    import sys
    sys.exit(pytest.main([__file__, "-v"]))
