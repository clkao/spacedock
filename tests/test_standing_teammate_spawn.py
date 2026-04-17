# ABOUTME: Live E2E test for standing-teammate mod spawn + FO routing (#162).
# ABOUTME: Verifies `standing: true` mod auto-spawns a teammate and SendMessage roundtrip works.

from __future__ import annotations

import sys
import time
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    LogParser,
    git_add_commit,
    install_agents,
    run_first_officer_streaming,
    setup_fixture,
    tool_use_matches,
)


def _agent_input(entry: dict) -> dict:
    """Extract the input dict of the first Agent() tool_use block in an entry."""
    message = entry.get("message", {})
    for block in message.get("content", []) or []:
        if isinstance(block, dict) and block.get("type") == "tool_use" and block.get("name") == "Agent":
            return block.get("input", {}) or {}
    return {}


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_standing_teammate_spawns_and_roundtrips(test_project, model, effort):
    """AC-12: standing: true mod spawns a teammate the FO can route to.

    Fixture declares one standing teammate (`echo-agent`) via
    `_mods/echo-agent.md`. Progressive per-step assertions ensure fail-fast
    diagnostics: each milestone names the step that stalled.

    1. Invoke `claude-team spawn-standing` (bash tool call).
    2. Spawn `echo-agent` via Agent() (tool_use with subagent_type +
       name=echo-agent in team config).
    3. Dispatch an ensign Agent() whose prompt contains the standing-teammates
       section listing echo-agent (AC-14).
    4. Route a SendMessage to echo-agent and receive a reply containing
       `ECHO: ping`.
    """
    t = test_project

    print("--- Phase 1: Set up fixture (workflow in subdir) ---")
    setup_fixture(t, "standing-teammate", "standing-teammate")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: standing-teammate fixture")
    print()

    print("--- Phase 2: Run first officer with streaming watcher ---")
    abs_workflow = t.test_project_dir / "standing-teammate"
    prompt = (
        f"Process the workflow at {abs_workflow}/ to terminal completion. "
        "During startup, spawn every standing teammate declared in "
        "_mods/*.md with standing: true. Then drive task 001 through its "
        "stages. The task instructs its ensign to SendMessage echo-agent "
        "with 'ping' and capture the reply — do not skip that step. Stop "
        "once the entity is archived."
    )

    with run_first_officer_streaming(
        t,
        prompt,
        agent_id="spacedock:first-officer",
        extra_args=["--model", model, "--effort", effort, "--max-budget-usd", "2.00"],
    ) as w:
        w.expect(
            lambda e: tool_use_matches(e, "Bash", command="spawn-standing"),
            timeout_s=120,
            label="claude-team spawn-standing invoked",
        )
        print("[OK] claude-team spawn-standing invoked")

        w.expect(
            lambda e: tool_use_matches(e, "Agent", name="echo-agent"),
            timeout_s=120,
            label="echo-agent Agent() dispatched",
        )
        print("[OK] echo-agent Agent() dispatched")

        ensign_dispatch = w.expect(
            lambda e: tool_use_matches(e, "Agent")
            and "echo-agent" not in _agent_input(e).get("name", ""),
            timeout_s=240,
            label="ensign Agent() dispatched",
        )
        ensign_prompt = _agent_input(ensign_dispatch).get("prompt", "")
        assert "### Standing teammates available in your team" in ensign_prompt, (
            "Ensign dispatch prompt missing the standing-teammates section. "
            f"Prompt preview: {ensign_prompt[:200]!r}"
        )
        assert "echo-agent" in ensign_prompt, (
            "Standing-teammates section did not list echo-agent by name. "
            f"Prompt preview: {ensign_prompt[:200]!r}"
        )
        print("[OK] ensign dispatch prompt includes standing-teammates section with echo-agent")

        w.expect(
            lambda e: tool_use_matches(e, "SendMessage", to="echo-agent"),
            timeout_s=240,
            label="SendMessage to echo-agent observed",
        )
        print("[OK] SendMessage to echo-agent observed")

        archived = abs_workflow / "_archive" / "001-echo-roundtrip.md"
        archive_deadline = time.monotonic() + 300
        while time.monotonic() < archive_deadline:
            if archived.is_file() and "ECHO: ping" in archived.read_text():
                break
            time.sleep(1.0)
        else:
            raise AssertionError(
                f"Archived entity with 'ECHO: ping' did not appear at {archived} within 300s"
            )
        print("[OK] archived entity body captured 'ECHO: ping' (data-flow assertion)")
        w.proc.terminate()

    print()
    print("--- Phase 3: Final aggregate assertions ---")
    log = LogParser(t.log_dir / "fo-log.jsonl")
    agent_calls = log.agent_calls()
    echo_spawns = [c for c in agent_calls if c.get("name") == "echo-agent"]
    assert echo_spawns, (
        "Aggregate check: no echo-agent Agent() call in final log. "
        f"Agent() calls seen: {[(c.get('name'), c.get('subagent_type')) for c in agent_calls]}"
    )
    print(f"[OK] aggregate: echo-agent Agent() dispatched {len(echo_spawns)} time(s)")
