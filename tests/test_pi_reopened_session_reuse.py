#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live proof that Pi reopened sessions support first-slice same-worker reuse semantics.

from __future__ import annotations

from pathlib import Path
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import PiLogParser, run_pi_prompt  # noqa: E402


@pytest.mark.live_pi
@pytest.mark.serial
def test_pi_reopened_session_reuse_preserves_same_worker_handle(test_project):
    t = test_project
    session_dir = t.test_dir / "pi-reuse-sessions"
    sentinel = "SPACEDOCK_PI_REUSE_218_SENTINEL"
    ack = "ACK_SPACEDOCK_PI_REUSE_218"

    first_exit = run_pi_prompt(
        t,
        f"Remember the sentinel token `{sentinel}` for this session. Reply with exactly `{ack}` and nothing else.",
        session_dir=session_dir,
        no_context_files=True,
        log_name="pi-reuse-pass-1.jsonl",
        timeout_s=120,
    )
    t.check("initial Pi session exited cleanly", first_exit == 0)

    first_log = PiLogParser(t.log_dir / "pi-reuse-pass-1.jsonl")
    first_text = first_log.full_text().strip()
    session_id = first_log.session_id()
    session_file = first_log.session_file(session_dir)
    t.check("first turn returned the expected acknowledgment", first_text == ack)
    t.check("first turn exposed a Pi session id", bool(session_id))
    t.check("first turn produced a concrete session file", bool(session_file and session_file.is_file()))
    first_line_count = len(session_file.read_text().splitlines()) if session_file else 0

    second_exit = run_pi_prompt(
        t,
        "Reply with only the exact sentinel token you were asked to remember in the previous turn.",
        session_dir=session_dir,
        session=session_id,
        no_context_files=True,
        log_name="pi-reuse-pass-2.jsonl",
        timeout_s=120,
    )
    t.check("reopened Pi session by session id exited cleanly", second_exit == 0)

    second_log = PiLogParser(t.log_dir / "pi-reuse-pass-2.jsonl")
    second_text = second_log.full_text().strip()
    second_usage = second_log.last_assistant_usage()
    t.check("reopened session id stayed stable", second_log.session_id() == session_id)
    t.check("reopened session recalled the previous-turn sentinel", second_text == sentinel)
    t.check(
        "reopened session consumed cached prior context",
        int(second_usage.get("cacheRead", 0) or 0) > 0,
    )
    second_line_count = len(session_file.read_text().splitlines()) if session_file else 0
    t.check("same session file grew after reopened follow-up", second_line_count > first_line_count)

    third_exit = run_pi_prompt(
        t,
        "Reply with only the sentinel token again.",
        session_dir=session_dir,
        session=session_file,
        no_context_files=True,
        log_name="pi-reuse-pass-3.jsonl",
        timeout_s=120,
    )
    t.check("reopened Pi session by session file exited cleanly", third_exit == 0)

    third_log = PiLogParser(t.log_dir / "pi-reuse-pass-3.jsonl")
    third_usage = third_log.last_assistant_usage()
    third_line_count = len(session_file.read_text().splitlines()) if session_file else 0
    t.check("session-file reopen kept the same session id", third_log.session_id() == session_id)
    t.check("session-file reopen also recalled the sentinel", third_log.full_text().strip() == sentinel)
    t.check(
        "session-file reopen also read cached prior context",
        int(third_usage.get("cacheRead", 0) or 0) > 0,
    )
    t.check("same session file kept accumulating turns", third_line_count > second_line_count)

    t.finish()
