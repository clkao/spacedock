# ABOUTME: Portable E2E test for the checklist protocol in the first-officer template.
# ABOUTME: Uses a deterministic fixture (no runtime commission) and validates:
# ABOUTME: (1) the ensign dispatch prompt contains a completion checklist
# ABOUTME: (2) the ensign accounts for that checklist in a Stage Report

from __future__ import annotations

import re
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    CodexLogParser,
    LogParser,
    git_add_commit,
    install_agents,
    plugin_location_hint,
    run_codex_first_officer,
    run_first_officer,
    setup_fixture,
)


def _extract_checklist_items(agent_prompt: str) -> list[str]:
    items: list[str] = []
    in_checklist = False
    for raw in agent_prompt.splitlines():
        line = raw.strip()
        # Seen variants:
        # - "### Completion checklist"
        # - "Completion checklist:"
        # - "Completion checklist (linchpins):"
        if re.match(r"(?i)^(###\s+)?completion checklist\b.*:\s*$", line) or re.match(
            r"(?i)^###\s+completion checklist\s*$", line
        ):
            in_checklist = True
            continue
        if in_checklist and re.match(r"(?i)^(instructions|requirements):\s*$", line):
            break
        if in_checklist and line.startswith("### "):
            break
        m = re.match(r"^\d+\.\s+(.*)$", line)
        if in_checklist and m:
            items.append(m.group(1).strip())
    return items


def _last_stage_report(entity_text: str) -> str:
    # Keep it simple: the ensign protocol is append-only; the last section is authoritative.
    parts = re.split(r"(?m)^##\s+Stage Report", entity_text)
    if len(parts) <= 1:
        return ""
    return parts[-1]


def _run_checklist_scenario(
    test_project,
    runtime,
    model,
    effort,
    *,
    claude_extra_args: list[str] | None = None,
):
    """Shared checklist-protocol harness for both claude and codex runtimes.

    `claude_extra_args` is forwarded to `run_first_officer` so callers can
    inject `--append-system-prompt` hints (see the helper-path entrypoint
    below, which pins the plugin directory).
    """
    t = test_project

    print("--- Phase 1: Set up test project from fixture ---")
    workflow_dir = setup_fixture(t, "checklist-pipeline", "checklist-pipeline")
    if runtime == "claude":
        install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: checklist protocol fixture")

    entity_main = workflow_dir / "checklist-task.md"
    entity_archive = workflow_dir / "_archive" / "checklist-task.md"
    t.check("fixture includes checklist-task entity", entity_main.is_file())
    print()

    print(f"--- Phase 2: Run first officer ({runtime}) ---")
    prompt = (
        f"Process only the entity `checklist-task` through the workflow at {workflow_dir}/. "
        "Process one entity through one stage, then stop."
    )
    if runtime == "claude":
        extra = ["--max-budget-usd", "2.00", "--model", model, "--effort", effort]
        if claude_extra_args:
            extra.extend(claude_extra_args)
        fo_exit = run_first_officer(
            t,
            prompt,
            agent_id="spacedock:first-officer",
            extra_args=extra,
        )
        if fo_exit != 0:
            print(f"  (first officer exit code {fo_exit} — may be expected under budget caps)")
    else:
        # Bounded stop: once the worker has written a stage report accounting for
        # checklist items, the test outcome is determined.
        def stop_ready(_log_path: Path) -> bool:
            path = entity_archive if entity_archive.is_file() else entity_main
            if not path.is_file():
                return False
            text = path.read_text()
            if "## Stage Report" not in text:
                return False
            return bool(re.search(r"(?m)^- (DONE|SKIPPED|FAILED):", text))

        fo_exit = run_codex_first_officer(
            t,
            "checklist-pipeline",
            agent_id="spacedock:first-officer",
            run_goal=prompt,
            timeout_s=900,
            stop_checker=stop_ready,
        )
        t.check("Codex launcher exited cleanly", fo_exit == 0)

    print("--- Phase 3: Validation ---")
    if runtime == "claude":
        log = LogParser(t.log_dir / "fo-log.jsonl")
        log.write_agent_prompt(t.log_dir / "agent-prompt.txt")
        log.write_fo_texts(t.log_dir / "fo-texts.txt")
        agent_prompt = log.agent_prompt()
    else:
        log = CodexLogParser(t.log_dir / "codex-fo-log.txt")
        agent_prompt = ""
        # Prefer the actual prompt passed to the worker; fall back to raw-text scan.
        for call in log.collab_tool_calls():
            if call.get("tool") not in {"spawn", "spawn_agent"}:
                continue
            prompt_text = call.get("prompt") or ""
            if re.search(r"(?i)completion checklist", prompt_text):
                agent_prompt = prompt_text
                break
        if not agent_prompt:
            agent_prompt = log.full_text()

    print()
    print("[Ensign Dispatch Prompt]")
    t.check(
        "dispatch prompt contains Completion checklist section",
        bool(re.search(r"Completion checklist|completion checklist", agent_prompt, re.IGNORECASE)),
    )

    shared_core_path = (
        Path(__file__).resolve().parent.parent
        / "skills"
        / "ensign"
        / "references"
        / "ensign-shared-core.md"
    )
    shared_core_text = shared_core_path.read_text()
    t.check(
        "shared-core documents DONE/SKIPPED/FAILED semantics",
        bool(re.search(r"DONE:.*SKIPPED:.*FAILED:", shared_core_text, re.DOTALL)),
    )

    checklist_items = _extract_checklist_items(agent_prompt)
    t.check("ensign prompt contains at least one checklist item", len(checklist_items) > 0)

    print()
    print("[Entity Stage Report]")
    entity_path = entity_archive if entity_archive.is_file() else entity_main
    t.check("entity exists (active or archived)", entity_path.is_file())
    entity_text = entity_path.read_text() if entity_path.is_file() else ""
    stage_report_text = _last_stage_report(entity_text)

    # The protocol requires a Stage Report that accounts for the dispatch checklist
    # items via DONE/SKIPPED/FAILED entries. Explicitly reject checkbox bullets.
    t.check("entity body contains a Stage Report section", bool(stage_report_text))
    t.check(
        "stage report uses DONE/SKIPPED/FAILED markers (no checkbox bullets)",
        bool(re.search(r"(?m)^- (DONE|SKIPPED|FAILED):", stage_report_text))
        and not bool(re.search(r"(?m)^- \[[xX ]\]", stage_report_text)),
    )
    t.check("stage report includes Summary subsection", "### Summary" in stage_report_text)

    for item in checklist_items:
        t.check(f"stage report accounts for checklist item: {item}", item in stage_report_text)

    # Sanity: the stage's deliverable should exist.
    out_path = t.test_project_dir / "checklist-pipeline" / "output.txt"
    t.check("output file created", out_path.is_file())

    t.finish()


@pytest.mark.live_claude
def test_checklist_e2e_helper_path(test_project, runtime, model, effort):
    """Helper-path: pin the plugin directory so the FO runs `claude-team build`.

    Without the plugin-path hint, a low-effort FO can skip `claude-team build`
    entirely and fall through to the break-glass template. Injecting
    `plugin_location_hint` via `--append-system-prompt` resolves the helper
    binaries for the FO so it takes the fully-featured dispatch path that
    `cmd_build` produces, including the `### Stage report` block.
    """
    if runtime != "claude":
        pytest.skip("helper-path test exercises the claude `-p` append-system-prompt path")
    t = test_project
    hint = plugin_location_hint(t.repo_root)
    _run_checklist_scenario(
        test_project,
        runtime,
        model,
        effort,
        claude_extra_args=["--append-system-prompt", hint],
    )


@pytest.mark.live_claude
def test_checklist_e2e_break_glass_path(test_project, runtime, model, effort):
    """Break-glass path: no plugin-path hint, exercises the fallback template.

    With no plugin-path hint, a low-effort FO may skip `claude-team build` and
    assemble the Agent() dispatch from the Break-Glass Manual Dispatch template
    in `claude-first-officer-runtime.md`. Post-#211 that template carries the
    `### Stage report` block verbatim from `cmd_build`, so the dispatched
    ensign still produces a compliant Stage Report.
    """
    if runtime != "claude":
        pytest.skip("break-glass-path test exercises the claude `-p` manual-dispatch path")
    _run_checklist_scenario(test_project, runtime, model, effort)


@pytest.mark.live_codex
def test_checklist_e2e_codex(test_project, runtime, model, effort):
    """Codex-runtime checklist protocol check (spawn_agent dispatch)."""
    if runtime != "codex":
        pytest.skip("codex-runtime test")
    _run_checklist_scenario(test_project, runtime, model, effort)
