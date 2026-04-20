#!/usr/bin/env python3
# ABOUTME: Workaround for anthropics/claude-code#26426 — polls team-lead inbox JSON files.
# ABOUTME: Under `claude -p` the InboxPoller React hook never fires; this script surfaces inbox messages via stdout.

"""Polls a Claude Code teams inbox JSON file for new messages matching a pattern.

Used by live E2E tests as a blocking keep-alive probe in the FO's Bash tool.
The FO runs `fo_inbox_poll.py --home $HOME --pattern Done: --timeout 10` each
idle turn; the script blocks up to timeout_s waiting for an unread inbox entry
whose text matches the pattern, then prints the entry and exits. The FO's Bash
tool_result contains the inbox content, surfacing it into the FO's stream-json
(which `InboxPoller` would normally do under an interactive TTY).

Exit 0 on match (with content on stdout) or timeout (no output).
Exit 2 on configuration error (missing HOME, malformed JSON after retries).

Does NOT modify `read` flags in the inbox file — that is the runtime's
responsibility; this script is a read-only surrogate surface.
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path


def find_inboxes(home: Path, inbox_name: str) -> list[Path]:
    """Return every {home}/.claude/teams/*/inboxes/{inbox_name}.json path."""
    base = home / ".claude" / "teams"
    if not base.is_dir():
        return []
    return sorted(base.glob(f"*/inboxes/{inbox_name}.json"))


def load_entries(path: Path) -> list[dict]:
    """Read and parse an inbox JSON file, tolerating partial writes."""
    try:
        return json.loads(path.read_text())
    except (json.JSONDecodeError, OSError):
        return []


def find_matches(
    inbox_paths: list[Path],
    pattern: str,
    seen_keys: set[tuple[str, str, str]],
) -> list[tuple[Path, dict]]:
    """Return (path, entry) for entries whose text contains pattern and which we haven't reported yet."""
    matches: list[tuple[Path, dict]] = []
    for path in inbox_paths:
        for entry in load_entries(path):
            if not isinstance(entry, dict):
                continue
            text = str(entry.get("text", ""))
            if pattern not in text:
                continue
            key = (
                str(path),
                str(entry.get("from", "")),
                str(entry.get("timestamp", "")),
            )
            if key in seen_keys:
                continue
            matches.append((path, entry))
    return matches


def format_match(path: Path, entry: dict) -> str:
    team_name = path.parent.parent.name
    summary = entry.get("summary") or ""
    return (
        f"team: {team_name}\n"
        f"from: {entry.get('from', '')}\n"
        f"timestamp: {entry.get('timestamp', '')}\n"
        f"summary: {summary}\n"
        f"text: {entry.get('text', '')}"
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--home", required=True, help="CLAUDE_CONFIG_DIR / HOME used for -p subprocess")
    parser.add_argument("--pattern", default="Done:", help="Substring the inbox entry text must contain")
    parser.add_argument("--inbox-name", default="team-lead", help="Inbox basename (default: team-lead)")
    parser.add_argument("--timeout", type=float, default=10.0, help="Max wait in seconds")
    parser.add_argument("--poll-interval", type=float, default=0.5, help="Poll interval in seconds")
    parser.add_argument("--seen-file", help="Optional path to a seen-keys sidecar so repeated calls only surface new messages")
    args = parser.parse_args()

    home = Path(args.home).expanduser()
    if not home.is_dir():
        print(f"error: home directory does not exist: {home}", file=sys.stderr)
        return 2

    seen_keys: set[tuple[str, str, str]] = set()
    if args.seen_file:
        sfp = Path(args.seen_file)
        if sfp.is_file():
            for line in sfp.read_text().splitlines():
                parts = line.split("\t")
                if len(parts) == 3:
                    seen_keys.add(tuple(parts))  # type: ignore[arg-type]

    deadline = time.monotonic() + args.timeout
    while True:
        inboxes = find_inboxes(home, args.inbox_name)
        matches = find_matches(inboxes, args.pattern, seen_keys)
        if matches:
            for path, entry in matches:
                print(format_match(path, entry))
                print("---")
                seen_keys.add((
                    str(path),
                    str(entry.get("from", "")),
                    str(entry.get("timestamp", "")),
                ))
            if args.seen_file:
                Path(args.seen_file).write_text(
                    "".join(f"{a}\t{b}\t{c}\n" for a, b, c in sorted(seen_keys))
                )
            return 0
        if time.monotonic() >= deadline:
            return 0
        time.sleep(args.poll_interval)


if __name__ == "__main__":
    sys.exit(main())
