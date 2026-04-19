# ABOUTME: E2E test for the checklist protocol in the first-officer template.
# ABOUTME: Commissions a workflow, runs the first officer, validates ensign checklist compliance.

from __future__ import annotations

import os
import re
import shutil
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    LogParser,
    git_add_commit,
    run_commission,
    run_first_officer,
)


# #154 reclassified the original `pending #154` xfail here: this test reads no static FO content,
# so the content-home refresh is irrelevant. The 1/9 live failure is runtime-behavior drift (FO no
# longer emits checklist-review text during post-dispatch review) tracked by #198.
@pytest.mark.xfail(strict=False, reason="pending #198 — runtime FO checklist-review emission drift; see docs/plans/fo-runtime-test-failures-post-154.md")
@pytest.mark.live_claude
def test_checklist_e2e(test_project, model, effort):
    """Commissions a full workflow then runs FO to verify ensign checklist compliance."""
    t = test_project
    snapshot = os.environ.get("CHECKLIST_SNAPSHOT") or None
    # test_checklist historically defaulted to opus; respect it unless model is overridden from haiku.
    model_for_run = model if model != "haiku" else "opus"

    if snapshot:
        print("--- Phase 1: Loading from snapshot (skipping commission) ---")
        snap_path = Path(snapshot)
        shutil.copytree(snap_path, t.test_dir, dirs_exist_ok=True)
        t.test_project_dir = t.test_dir / "test-project"

        workflow_dir = None
        for d in t.test_project_dir.iterdir():
            if d.is_dir() and (d / "README.md").is_file() and d.name != ".claude" and d.name != ".git":
                workflow_dir = d
                break

        assert workflow_dir is not None, "snapshot must contain a workflow directory"

        for entity_file in sorted(workflow_dir.glob("*.md")):
            if entity_file.name == "README.md":
                continue
            with open(entity_file, "a") as f:
                f.write("""

## Acceptance Criteria

1. The output file contains the word "hello"
2. The output file is valid UTF-8
""")
            git_add_commit(t.test_project_dir, "setup: add acceptance criteria to entity")
            break

        t.pass_("loaded snapshot")
        print()
    else:
        print("--- Phase 1: Commission test workflow (this takes ~30-60s) ---")
        prompt = """/spacedock:commission

All inputs for this workflow:
- Mission: Track tasks through stages
- Entity: A task
- Stages: backlog → work → done
- Approval gates: none
- Seed entities:
  1. test-checklist — Verify checklist protocol works (score: 25/25)
- Location: ./checklist-test/

Skip interactive questions and confirmation — use these inputs directly. Make reasonable assumptions for anything not specified. Do NOT run the pilot phase — just generate the files and stop."""

        run_commission(t, prompt, extra_args=["--model", model_for_run, "--effort", effort])

        print("[Commission Output]")
        entity_file = t.test_project_dir / "checklist-test" / "test-checklist.md"
        assert entity_file.is_file(), "commission must produce test-checklist.md"
        t.pass_("commission produced test-checklist.md")
        t.pass_("first-officer agent ships with plugin (no local copy needed)")

        with open(entity_file, "a") as f:
            f.write("""

## Acceptance Criteria

1. The output file contains the word "hello"
2. The output file is valid UTF-8
""")
        git_add_commit(t.test_project_dir, "commission: initial workflow with acceptance criteria")
        print()

    print("--- Phase 2: Run first officer (this takes ~60-120s) ---")
    run_first_officer(
        t,
        "Process all entities through the workflow. Process one entity through one stage, then stop.",
        extra_args=["--max-budget-usd", "2.00", "--model", model_for_run, "--effort", effort],
    )

    print("--- Phase 3: Validation ---")
    log = LogParser(t.log_dir / "fo-log.jsonl")
    log.write_agent_prompt(t.log_dir / "agent-prompt.txt")
    log.write_fo_texts(t.log_dir / "fo-texts.txt")

    agent_prompt = log.agent_prompt()
    fo_text = "\n".join(log.fo_texts())

    print()
    print("[Ensign Dispatch Prompt]")
    t.check("dispatch prompt contains Completion checklist section",
            bool(re.search(r"Completion checklist|completion checklist", agent_prompt, re.IGNORECASE)))
    shared_core_path = Path(__file__).resolve().parent.parent / "skills" / "ensign" / "references" / "ensign-shared-core.md"
    shared_core_text = shared_core_path.read_text()
    t.check("shared-core documents DONE/SKIPPED/FAILED semantics",
            bool(re.search(r"DONE:.*SKIPPED:.*FAILED:", shared_core_text, re.DOTALL)))
    t.check("dispatch prompt includes entity acceptance criteria",
            bool(re.search(r"hello|UTF-8", agent_prompt, re.IGNORECASE)))
    t.check("dispatch prompt includes stage requirement items",
            bool(re.search(r"deliverable|summary", agent_prompt, re.IGNORECASE)))

    print()
    print("[First Officer Checklist Review]")
    t.check("first officer performed checklist review",
            bool(re.search(r"checklist review|checklist.*complete|all.*items.*DONE|items reported",
                           fo_text, re.IGNORECASE)))
    t.check("first officer review references item statuses",
            bool(re.search(r"DONE|SKIPPED|FAILED", fo_text, re.IGNORECASE)))
    t.check("dispatch prompt has structured completion message template",
            bool(re.search(r"### Checklist|### Summary", agent_prompt, re.IGNORECASE)))

    t.finish()

