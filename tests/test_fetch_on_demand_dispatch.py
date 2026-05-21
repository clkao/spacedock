# ABOUTME: Live E2E test for the fetch-on-demand dispatch shape (AC-8).
# ABOUTME: Verifies the FO-dispatched ensign reads the stage def via fetch commands and writes a Stage Report.

from __future__ import annotations

import re
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    LogParser,
    git_add_commit,
    install_agents,
    plugin_location_hint,
    run_first_officer,
    setup_fixture,
)


def _last_stage_report(entity_text: str) -> str:
    parts = re.split(r"(?m)^##\s+Stage Report", entity_text)
    if len(parts) <= 1:
        return ""
    return parts[-1]


_HAIKU_XFAIL_REASON = (
    "pending #200 — haiku-low FO can bypass `claude-team build` and fall through "
    "to the break-glass template, which inlines stage def + omits standing-teammates "
    "rather than emitting `### Fetch commands`. Helper-path coverage is provided "
    "by the parser-level golden-diff tests in tests/test_claude_team.py."
)


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_fetch_on_demand_dispatch_runs_fetch_commands_and_writes_stage_report(
    request, test_project, runtime, model, effort
):
    """AC-8: a real ensign dispatched via the restructured helper successfully reads
    the stage definition by running the fetch commands the dispatch prompt references,
    and writes a well-formed Stage Report.

    Reuses the existing checklist-pipeline fixture. The helper restructure is in the
    code path the FO invokes via `claude-team build`; we assert the dispatch prompt
    references `claude-team show-stage-def` (fetch-on-demand shape) AND the ensign's
    Stage Report shows the checklist accounting (proxy: the ensign successfully read
    the stage definition and completed the work; if the fetch failed the ensign would
    either error out or skip checklist accounting)."""
    if runtime != "claude":
        pytest.skip("AC-8 exercises the claude `-p` helper-path dispatch")
    if model == "haiku":
        request.applymarker(pytest.mark.xfail(strict=False, reason=_HAIKU_XFAIL_REASON))

    t = test_project
    workflow_dir = setup_fixture(t, "checklist-pipeline", "checklist-pipeline")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: fetch-on-demand dispatch test fixture")

    entity_main = workflow_dir / "checklist-task.md"
    entity_archive = workflow_dir / "_archive" / "checklist-task.md"
    assert entity_main.is_file()

    prompt = (
        f"Process only the entity `checklist-task` through the workflow at {workflow_dir}/. "
        "Process one entity through one stage, then stop."
    )
    hint = plugin_location_hint(t.repo_root)
    fo_exit = run_first_officer(
        t,
        prompt,
        agent_id="spacedock:first-officer",
        extra_args=[
            "--max-budget-usd", "2.00",
            "--model", model,
            "--effort", effort,
            "--append-system-prompt", hint,
        ],
    )
    if fo_exit != 0:
        print(f"(first officer exit code {fo_exit} — may be expected under budget caps)")

    log = LogParser(t.log_dir / "fo-log.jsonl")
    log.write_agent_prompt(t.log_dir / "agent-prompt.txt")
    agent_prompt = log.agent_prompt()

    # Structural: the dispatch carries the fetch-on-demand shape.
    assert "### Fetch commands" in agent_prompt, (
        "dispatch prompt must carry the ### Fetch commands block when the helper-path "
        "build runs"
    )
    assert "claude-team show-stage-def" in agent_prompt, (
        "dispatch prompt must reference claude-team show-stage-def as a fetch command"
    )

    # Functional: the ensign successfully consumed the stage def + checklist and
    # wrote a Stage Report with DONE/SKIPPED/FAILED markers.
    entity_path = entity_archive if entity_archive.is_file() else entity_main
    assert entity_path.is_file()
    entity_text = entity_path.read_text()
    stage_report = _last_stage_report(entity_text)
    assert stage_report, "ensign must write a ## Stage Report section"
    assert re.search(r"(?m)^- (DONE|SKIPPED|FAILED):", stage_report), (
        "stage report must use DONE/SKIPPED/FAILED markers"
    )

    # Concrete deliverable: the ensign produced the file the checklist asked for.
    out_path = t.test_project_dir / "checklist-pipeline" / "output.txt"
    assert out_path.is_file(), "ensign must produce the checklist deliverable"


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
