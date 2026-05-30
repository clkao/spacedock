#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: E2E test for the first officer booting into an empty repo with no workflow.
# ABOUTME: Verifies the FO handles the empty-repo case gracefully and suggests commissioning.

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (
    TestRunner, LogParser, create_test_project,
    install_agents, run_first_officer,
)


def parse_args() -> tuple[argparse.Namespace, list[str]]:
    parser = argparse.ArgumentParser(description="Empty-repo boot E2E test")
    parser.add_argument("--model", default="haiku", help="Model to use (default: haiku)")
    parser.add_argument("--effort", default="low", help="Effort level (default: low)")
    return parser.parse_known_args()


def main():
    args, extra_args = parse_args()
    t = TestRunner("Empty Repo Boot E2E Test")

    # --- Phase 1: Set up empty project (no workflow commissioned) ---

    print("--- Phase 1: Set up empty test project (no workflow) ---")

    create_test_project(t)
    install_agents(t)

    print()
    print("[Fixture Setup]")
    t.pass_("created empty git repo with no workflow")
    t.pass_("installed first-officer agent")

    print()

    # --- Phase 2: Run the first officer ---

    print("--- Phase 2: Run first officer in empty repo (this takes ~30-60s) ---")

    fo_exit = run_first_officer(
        t,
        "Report workflow status.",
        extra_args=[
            "--model", args.model,
            "--effort", args.effort,
            "--max-budget-usd", "0.50",
            *extra_args,
        ],
    )

    # --- Phase 3: Validate ---

    print("--- Phase 3: Validation ---")

    log = LogParser(t.log_dir / "fo-log.jsonl")
    log.write_fo_texts(t.log_dir / "fo-texts.txt")

    fo_text_output = "\n".join(log.fo_texts())

    print()
    print("[Empty Repo Handling]")

    # Check 1: FO produced output (did not crash silently)
    t.check("first officer produced output",
            bool(fo_text_output.strip()))

    # Check 2: FO recognized no workflow exists
    t.check("first officer recognized no workflow",
            bool(re.search(
                r"no workflow|no.*commissioned|not found|no.*README|does not exist|empty|no.*pipeline|no.*entities",
                fo_text_output, re.IGNORECASE)))

    # Check 3: FO suggested commissioning or provided guidance
    t.check("first officer suggested commissioning or provided guidance",
            bool(re.search(
                r"commission|create.*workflow|set up|initialize|spacedock.*commission|get started",
                fo_text_output, re.IGNORECASE)))

    # Check 4: FO did not dispatch any workers (nothing to dispatch in empty repo)
    agent_calls = log.agent_calls()
    t.check(f"first officer did not dispatch workers (got {len(agent_calls)} dispatches)",
            len(agent_calls) == 0)

    # --- Results ---
    t.results()


if __name__ == "__main__":
    main()
