#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live Codex regression for immediate preemptible waits after fresh dispatch.

from __future__ import annotations

import re
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    CodexLogParser,
    git_add_commit,
    run_codex_first_officer,
    setup_fixture,
)


def _item_entries(log: CodexLogParser) -> list[tuple[int, dict]]:
    entries: list[tuple[int, dict]] = []
    for idx, entry in enumerate(log.json_entries):
        if entry.get("type") != "item.completed":
            continue
        item = entry.get("item", {})
        if isinstance(item, dict):
            entries.append((idx, item))
    return entries


def _first_spawn_followed_by_wait(log: CodexLogParser) -> tuple[dict, dict | None, list[str]]:
    entries = _item_entries(log)
    for pos, (_, item) in enumerate(entries):
        if item.get("type") != "collab_tool_call":
            continue
        if item.get("tool") not in {"spawn", "spawn_agent"}:
            continue

        handles = [str(handle) for handle in item.get("receiver_thread_ids") or []]
        intervening_messages: list[str] = []
        for _, later in entries[pos + 1:]:
            if later.get("type") == "agent_message" and later.get("text"):
                intervening_messages.append(str(later["text"]))
                continue
            if later.get("type") != "collab_tool_call":
                return item, None, intervening_messages
            if later.get("tool") not in {"wait", "wait_agent"}:
                return item, None, intervening_messages
            wait_handles = [str(handle) for handle in later.get("receiver_thread_ids") or []]
            if handles and wait_handles == handles:
                return item, later, intervening_messages
            return item, None, intervening_messages

        return item, None, intervening_messages

    return {}, None, []


def _has_preemptible_wait_status(messages: list[str]) -> bool:
    text = "\n".join(messages).lower()
    return (
        bool(re.search(r"\besc\b|message|interrupt", text))
        and "safe" in text
        and "resume" in text
        and "pause" in text
        and "stop" in text
    )


@pytest.mark.live_codex
@pytest.mark.serial
def test_codex_fresh_dispatch_immediately_enters_preemptible_wait(test_project):
    """A fresh Codex spawn immediately waits on the returned handle."""
    t = test_project

    print("--- Phase 1: Set up no-gate workflow fixture ---")
    setup_fixture(t, "spike-no-gate", "dispatch-wait-pipeline")
    git_add_commit(t.test_project_dir, "setup: dispatch immediate wait fixture")
    t.check_cmd(
        "status script runs without errors",
        ["bash", "dispatch-wait-pipeline/status"],
        cwd=t.test_project_dir,
    )
    print()

    print("--- Phase 2: Run Codex first officer ---")
    fo_exit = run_codex_first_officer(
        t,
        "dispatch-wait-pipeline",
        run_goal=(
            "Start the workflow in normal first-officer mode. "
            "Dispatch the next ready task and report the resulting status."
        ),
        timeout_s=360,
    )
    t.check("Codex launcher exited cleanly", fo_exit == 0)
    print()

    print("--- Phase 3: Validate immediate preemptible wait after dispatch ---")
    log = CodexLogParser(t.log_dir / "codex-fo-log.txt")
    spawn_call, wait_call, status_messages = _first_spawn_followed_by_wait(log)

    t.check("Codex FO made a fresh worker dispatch", bool(spawn_call))
    spawn_handles = [str(handle) for handle in spawn_call.get("receiver_thread_ids") or []]
    t.check("fresh dispatch returned runtime handle(s)", bool(spawn_handles))
    t.check(
        "first fresh dispatch is immediately followed by wait_agent on the same handle(s)",
        bool(wait_call)
        and [str(handle) for handle in wait_call.get("receiver_thread_ids") or []] == spawn_handles,
    )
    t.check(
        "operator-facing wait status says Esc/message interruption is safe and will resume unless paused/stopped",
        _has_preemptible_wait_status(status_messages),
    )

    t.finish()
