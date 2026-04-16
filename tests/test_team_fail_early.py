#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Static section-anchored checks for the FO fail-early team-infrastructure defense prose.
# ABOUTME: Asserts Rules 1, 2, 4, and 6 of the team-fragility issue land in the claude-first-officer-runtime adapter.

# The old test_team_health_check.py asserted the presence of the pre-dispatch
# `test -f config.json` probe; this file inverts that contract. The test-refresh
# property (AC-T) is implicit: the old assertions fail against the new adapter,
# and the new assertions below fail against the old adapter.

from __future__ import annotations

import re
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent
ADAPTER_CORE_PATH = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md"
ADAPTER_RECOVERY_PATH = REPO_ROOT / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-recovery.md"


def parse_sections(text: str) -> dict[str, str]:
    """Parse a markdown document into a map of heading -> section body.

    Keys are the full heading lines (including leading ``#`` markers and title),
    e.g. ``## Team Creation`` or ``### Triggers``. Each value is the text that
    follows that heading up to (but not including) the next heading of the same
    or lower depth. Subsection headings remain part of the parent section body.
    """
    lines = text.splitlines()
    sections: dict[str, list[str]] = {}
    stack: list[tuple[int, str]] = []  # (depth, heading)
    for line in lines:
        m = re.match(r"^(#{1,6})\s+(.*)$", line)
        if m:
            depth = len(m.group(1))
            heading = line.strip()
            while stack and stack[-1][0] >= depth:
                stack.pop()
            stack.append((depth, heading))
            sections.setdefault(heading, [])
            continue
        for _depth, heading in stack:
            sections[heading].append(line)
    return {h: "\n".join(body) for h, body in sections.items()}


@pytest.fixture(scope="module")
def adapter_text() -> str:
    return ADAPTER_CORE_PATH.read_text() + "\n" + ADAPTER_RECOVERY_PATH.read_text()


@pytest.fixture(scope="module")
def sections(adapter_text: str) -> dict[str, str]:
    return parse_sections(adapter_text)


def test_ac1_dispatch_adapter_has_no_config_probe(sections: dict[str, str]) -> None:
    """AC-1: The `## Dispatch Adapter` section contains no pre-dispatch
    `test -f ... config.json` probe language."""
    dispatch = sections["## Dispatch Adapter"]
    assert "test -f" not in dispatch, (
        "Dispatch Adapter must not instruct a `test -f` probe; found one."
    )
    assert "config.json" not in dispatch, (
        "Dispatch Adapter must not reference config.json as a pre-dispatch probe."
    )
    assert "Team health check" not in dispatch, (
        "Dispatch Adapter must not carry the retired 'Team health check' imperative."
    )


def test_ac1b_team_creation_has_single_diagnostic_only_probe(
    adapter_text: str, sections: dict[str, str]
) -> None:
    """AC-1b: The `## Team Creation` section contains exactly one reference to
    the config.json probe, framed as diagnostic-only and explicitly
    non-short-circuiting for TeamCreate."""
    team_creation = sections["## Team Creation"]
    # Count config.json mentions in the Team Creation section body.
    assert team_creation.count("config.json") >= 1, (
        "Team Creation must carry the diagnostic-only probe reference."
    )
    # Only one reference anywhere in the file is expected.
    assert adapter_text.count("config.json") == 1, (
        f"Expected exactly one config.json reference in the adapter; "
        f"found {adapter_text.count('config.json')}."
    )
    # Diagnostic framing and non-short-circuit clause must be present.
    assert "DIAGNOSTIC-ONLY" in team_creation, (
        "Startup probe must be explicitly labelled DIAGNOSTIC-ONLY."
    )
    assert re.search(
        r"does NOT (gate|short-circuit|skip)", team_creation
    ), "Startup probe must include an unambiguous non-short-circuit clause."
    assert re.search(r"`?TeamCreate`?\s+always\s+runs", team_creation), (
        "Startup probe prose must assert that TeamCreate always runs."
    )


def test_ac2_retry_same_name_banned_and_no_same_name_teamdelete_teamcreate(
    adapter_text: str, sections: dict[str, str]
) -> None:
    """AC-2: The failure recovery prose bans retry-to-same-name and specifies
    a fresh-suffixed TeamCreate. The old `TeamDelete -> TeamCreate` same-name
    recovery sequence must not appear as a recovery path."""
    recovery = sections["## TeamCreate Failure Recovery (priority-ordered ladder)"]
    # Positive: retry-to-same-name ban exact phrase.
    assert "Retry to the same team name is banned" in recovery, (
        "Recovery prose must include the exact phrase "
        "'Retry to the same team name is banned'."
    )
    # Positive: fresh-suffixed phrasing.
    assert "fresh-suffixed" in recovery.lower() or "Fresh-suffixed" in recovery, (
        "Recovery prose must describe the ladder's first tier as fresh-suffixed."
    )
    # Negative: the old prescriptive "Call TeamDelete ... then call TeamCreate"
    # recovery-path instruction must be gone. The new prose is free to
    # mention the sequence in prohibitive context (e.g., "another
    # `TeamDelete → TeamCreate` cycle will re-contaminate"); the check is
    # scoped to the prescriptive instruction shape.
    assert not re.search(
        r"Call\s+TeamDelete[^\.]*(?:then|Then)\s+call\s+TeamCreate",
        recovery,
    ), "Old prescriptive 'Call TeamDelete ... then call TeamCreate' recovery must be gone."
    # Positive: the prose must forbid TeamDelete as a response to registry-desync.
    assert re.search(
        r"Do NOT\s+call\s+`?TeamDelete`?",
        recovery,
    ), "Recovery prose must explicitly forbid calling TeamDelete on registry-desync."


def test_ac2b_prior_agents_presumed_zombified_and_redispatch_from_frontmatter(
    sections: dict[str, str],
) -> None:
    """AC-2b: The recovery prose presumes all prior agent names zombified and
    prescribes re-dispatch from entity frontmatter."""
    recovery = sections["## TeamCreate Failure Recovery (priority-ordered ladder)"]
    assert "presumed zombified" in recovery, (
        "Recovery prose must state prior agent names are 'presumed zombified'."
    )
    assert "re-dispatch from entity frontmatter" in recovery, (
        "Recovery prose must instruct 're-dispatch from entity frontmatter'."
    )


def test_ac4_triggers_enumerated_as_list_in_degraded_mode(
    sections: dict[str, str],
) -> None:
    """AC-4-triggers: The Degraded Mode section contains a Triggers subsection
    with all three triggers as a markdown list."""
    assert "## Degraded Mode" in sections, (
        "Runtime adapter must declare a `## Degraded Mode` section."
    )
    triggers = sections.get("### Triggers")
    assert triggers is not None, (
        "Degraded Mode must include a `### Triggers` subsection."
    )
    # Three triggers must appear as list items in the Triggers subsection.
    list_items = [
        line.strip()
        for line in triggers.splitlines()
        if re.match(r"^\s*[-*]\s+", line) or re.match(r"^\s*\d+\.\s+", line)
    ]
    assert len(list_items) >= 3, (
        f"Triggers subsection must enumerate at least three list items; "
        f"found {len(list_items)}."
    )
    trigger_blob = "\n".join(list_items)
    assert 'Team does not exist' in trigger_blob, (
        "Triggers must include the first 'Team does not exist' error."
    )
    assert re.search(
        r"(SECOND dispatch failure|second dispatch failure)", trigger_blob
    ), "Triggers must include 'any second dispatch failure'."
    assert "/spacedock bare" in trigger_blob, (
        "Triggers must include the captain command `/spacedock bare`."
    )


def test_ac4_effects_listed_in_degraded_mode(sections: dict[str, str]) -> None:
    """AC-4-effects: The Degraded Mode Effects subsection lists the three
    invariants as bullets."""
    effects = sections.get("### Effects")
    assert effects is not None, (
        "Degraded Mode must include an `### Effects` subsection."
    )
    list_items = [
        line.strip()
        for line in effects.splitlines()
        if re.match(r"^\s*[-*]\s+", line) or re.match(r"^\s*\d+\.\s+", line)
    ]
    assert len(list_items) >= 3, (
        f"Effects subsection must enumerate at least three bullets; "
        f"found {len(list_items)}."
    )
    joined = "\n".join(list_items)
    # (a) no team_name on subsequent Agent dispatches
    assert re.search(r"No `?team_name`?", joined), (
        "Effects must state no `team_name` on subsequent Agent dispatches."
    )
    # (b) every stage fresh and blocks
    assert re.search(r"Every stage dispatches fresh", joined), (
        "Effects must state every stage dispatches fresh and blocks."
    )
    # (c) no SendMessage reuse
    assert re.search(r"No SendMessage reuse", joined), (
        "Effects must state no SendMessage reuse of prior agent names."
    )


def test_ac4_shutdown_sweep_with_feedback_cycle_exemption(
    sections: dict[str, str],
) -> None:
    """AC-4-shutdown: The Cooperative Shutdown Sweep subsection specifies a
    single-pass sweep, ignore-failures semantics, and the active-feedback-cycle
    exemption keyed to `### Feedback Cycles`."""
    sweep = sections.get("### Cooperative Shutdown Sweep")
    assert sweep is not None, (
        "Degraded Mode must include a `### Cooperative Shutdown Sweep` subsection."
    )
    assert "single-pass" in sweep, (
        "Sweep subsection must state it is a single-pass sweep."
    )
    assert "Ignore failures" in sweep or "ignore failures" in sweep, (
        "Sweep subsection must state 'ignore failures'."
    )
    assert "Do not retry" in sweep or "do not retry" in sweep, (
        "Sweep subsection must state the sweep is not retried."
    )
    assert "active feedback-cycle" in sweep or "active-feedback-cycle" in sweep, (
        "Sweep subsection must mention an active feedback-cycle exemption."
    )
    assert "### Feedback Cycles" in sweep, (
        "Sweep subsection must reference the `### Feedback Cycles` entity state key."
    )
    assert "explicit captain confirmation" in sweep, (
        "Sweep subsection must require explicit captain confirmation to sweep "
        "feedback-cycle reviewers."
    )


def test_ac4c_captain_report_template_verbatim(sections: dict[str, str]) -> None:
    """AC-4c: The Captain Report Template subsection contains the verbatim
    canonical sentence including the three concrete next-step options."""
    template = sections.get("### Captain Report Template")
    assert template is not None, (
        "Degraded Mode must include a `### Captain Report Template` subsection."
    )
    expected = (
        "Falling back to bare mode for the remainder of this session due to "
        "team-infrastructure failure. Prior team agents are presumed-zombified; "
        "I will not route work to them or through the team registry. "
        "If you want to escalate: restart the session to retry team mode with "
        "a fresh name, or let me continue — every stage will still complete, "
        "just without concurrent dispatch."
    )
    assert expected in template, (
        "Captain Report Template must contain the verbatim canonical sentence "
        "with its three next-step options."
    )


def test_ac6_teamcreate_name_uses_timestamp_and_shortuuid_suffix(
    sections: dict[str, str],
) -> None:
    """AC-6: The startup TeamCreate invocation example uses a
    `YYYYMMDD-HHMM` timestamp token and a `{shortuuid}` suffix, anchored to
    the `## Team Creation` section."""
    team_creation = sections["## Team Creation"]
    assert "YYYYMMDD-HHMM" in team_creation, (
        "Team Creation prose must use a YYYYMMDD-HHMM timestamp token in the "
        "TeamCreate name example."
    )
    assert "{shortuuid}" in team_creation, (
        "Team Creation prose must include a {shortuuid} suffix placeholder."
    )
    # The combined template should appear in the startup TeamCreate example.
    assert re.search(
        r"TeamCreate\(team_name=\"\{project_name\}-\{dir_basename\}-\{YYYYMMDD-HHMM\}-\{shortuuid\}\"\)",
        team_creation,
    ), "Startup TeamCreate example must use the full fresh-suffixed template."
