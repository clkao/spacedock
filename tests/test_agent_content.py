#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Static content checks for shared Claude/Codex agent contracts and guardrails.
#
# NOTE: The Codex wait-policy checks in this file validate contract wording
# only. They do not execute a live interactive Codex session.

from __future__ import annotations

import re
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import TestRunner, assembled_agent_content


REPO_ROOT = Path(__file__).resolve().parent.parent


def read_text(path: str) -> str:
    return (REPO_ROOT / path).read_text()


def section_text(text: str, heading: str, stop_patterns: tuple[str, ...]) -> str:
    lines: list[str] = []
    in_section = False
    for line in text.splitlines():
        if re.match(rf"^{re.escape(heading)}$", line):
            in_section = True
            continue
        if in_section and any(re.match(pattern, line) for pattern in stop_patterns):
            break
        if in_section:
            lines.append(line)
    return "\n".join(lines)


def test_first_officer_skill_reads_references_directly():
    text = read_text("skills/first-officer/SKILL.md")
    assert "@references/first-officer-shared-core.md" in text
    assert "@references/code-project-guardrails.md" in text
    assert "claude-first-officer-runtime-core.md" in text
    assert "codex-first-officer-runtime.md" in text
    assert "CLAUDECODE" in text
    assert "CODEX_HOME" in text
    assert "${CLAUDE_SKILL_DIR}" not in text


def test_ensign_skill_reads_references_directly():
    text = read_text("skills/ensign/SKILL.md")
    assert "@references/ensign-shared-core.md" in text
    assert "code-project-guardrails.md" in text
    assert "claude-ensign-runtime.md" in text
    assert "codex-ensign-runtime.md" in text
    assert "CLAUDECODE" in text
    assert "CODEX_HOME" in text
    assert "${CLAUDE_SKILL_DIR}" not in text


def test_agent_entry_points_use_skill_preloading():
    fo_text = read_text("agents/first-officer.md")
    assert 'skills:' in fo_text
    assert 'spacedock:first-officer' in fo_text
    assert 'DISPATCHER' in fo_text

    ensign_text = read_text("agents/ensign.md")
    assert 'skills:' in ensign_text
    assert 'spacedock:ensign' in ensign_text


def test_first_officer_shared_core_covers_all_behavioral_sections():
    text = read_text("skills/first-officer/references/first-officer-shared-core.md")

    for heading in [
        "## Startup",
        "## Status Viewer",
        "## Single-Entity Mode",
        "## Working Directory",
        "## Dispatch",
        "## Completion and Gates",
        "## Feedback Rejection Flow",
        "## Merge and Cleanup",
        "## State Management",
        "## Worktree Ownership",
        "## Mod Hook Convention",
        "## Clarification and Communication",
        "## Issue Filing",
    ]:
        assert heading in text

    assert "Output Format" in text
    assert "feedback-to" in text
    assert "--next-id" in text
    assert "status --boot" in text


def test_first_officer_shared_core_documents_worktree_ownership_rule():
    text = read_text("skills/first-officer/references/first-officer-shared-core.md")

    assert "worktree-backed entities" in text
    assert "active stage/status/report/body state lives in the worktree copy" in text
    assert "`pr:` is mirrored on `main`" in text
    assert "Ordinary active-state writes like `implementation -> validation` do not land on `main`" in text


def test_ensign_shared_core_keeps_stage_report_protocol():
    text = read_text("skills/ensign/references/ensign-shared-core.md")
    assert "## Stage Report: {stage_name}" in text
    assert "append" in text.lower()
    assert "agents/" in text
    assert "Do NOT modify YAML frontmatter" in text


def test_ensign_shared_core_documents_worktree_owned_active_state_and_pr_mirror():
    text = read_text("skills/ensign/references/ensign-shared-core.md")
    assert "worktree-backed" in text.lower()
    assert "active stage/status/report/body state" in text
    assert "pr:" in text
    assert "mirrored" in text.lower()


def test_code_project_guardrails_cover_worktrees_and_scaffolding():
    text = read_text("skills/first-officer/references/code-project-guardrails.md")
    assert ".worktrees/" in text
    assert "agents/" in text
    assert "git worktree" in text
    assert "scaffolding" in text.lower()


def test_codex_runtime_docs_cover_merge_hook_finalize_path():
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")
    assert "codex_finalize_terminal_entity.py" not in text
    assert "merge hooks" in text.lower()
    assert "archive" in text.lower()
    assert "fork_context=false" in text
    assert "spacedock-ensign" in text
    assert "Never collapse a packaged logical id" in text
    assert "role_asset_name: ensign" in text
    assert "{worker_key}/{slug}" in text
    assert "active `SKILL.md`" in text
    assert "repository-wide search" in text


def test_codex_ensign_runtime_doc_mentions_skill_relative_bootstrap_resolution():
    text = read_text("skills/ensign/references/codex-ensign-runtime.md")
    assert "active `SKILL.md`" in text
    assert "bounded fallback" in text
    assert "searching the repository" in text


def test_assembled_codex_skill_contract_uses_skill_relative_bootstrap_language():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "ensign", runtime="codex")
    assert "Skill Bootstrap Resolution" in text
    assert "active `SKILL.md`" in text
    assert "bounded fallback" in text


def test_codex_runtime_docs_keep_interactive_workers_background_by_default():
    """Contract-level check: the runtime text describes the interactive wait policy."""
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer", runtime="codex")

    assert "interactive sessions" in text.lower()
    assert "do not foreground `wait_agent` immediately after `spawn_agent`" in text
    assert "explicitly asks to wait" in text
    assert "bounded single-entity runs" in text
    assert "wait immediately after dispatch" in text


def test_codex_runtime_docs_state_coverage_limits_in_plain_language():
    """Contract-level check: the docs stay honest about what the harness can prove."""
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert "interactive sessions" in text.lower()
    assert "background" in text.lower()
    assert "bounded single-entity runs" in text.lower()
    assert "wait_agent" in text


def test_reuse_and_shutdown_wording_stays_aligned_between_shared_core_and_codex_runtime():
    shared = read_text("skills/first-officer/references/first-officer-shared-core.md")
    runtime = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert re.search(r"completed worker.*addressable", shared, re.IGNORECASE | re.DOTALL)
    assert re.search(r"reuse conditions.*all must hold", shared, re.IGNORECASE | re.DOTALL)
    assert re.search(r"feedback-to.*send_input", shared, re.IGNORECASE | re.DOTALL)
    assert re.search(r"explicitly shut down|shut down explicitly", shared, re.IGNORECASE)

    assert re.search(r"completed worker.*send_input", runtime, re.IGNORECASE | re.DOTALL)
    assert re.search(r"feedback|advancement.*send_input", runtime, re.IGNORECASE | re.DOTALL)
    assert re.search(r"explicitly shut down|shutdown.*no longer needed", runtime, re.IGNORECASE)


def test_codex_runtime_docs_require_active_again_wait_and_shutdown_for_reused_workers():
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert re.search(r"send_input.*active again|active again.*send_input", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"critical path.*wait_agent|wait_agent.*critical path", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"reused worker.*wait_agent|wait_agent.*same worker handle", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"send_input.*prior completed state|stale completion echoed by send_input", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"concrete next-stage work|acknowledgment-only ping", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"actual follow-up fix|new commit|not just receipt of the rejection", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"route delivery alone is not enough|requested bounded outcome includes a routed reuse", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"reused cycle.*shut.*down|shut.*down.*reused cycle", text, re.IGNORECASE | re.DOTALL)


def test_codex_runtime_docs_define_human_readable_worker_labels():
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert re.search(r"human-readable", text, re.IGNORECASE)
    assert re.search(r"role key|worker label", text, re.IGNORECASE)
    assert re.search(r"130-impl/Herschel|entity[- ]stage[- ]display", text, re.IGNORECASE)


def test_codex_runtime_docs_require_fo_owned_label_examples_in_operator_updates():
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert "Dispatching `001-implementation/Ensign`" in text
    assert "Routing follow-up to `001-implementation/Ensign`" in text
    assert "Waiting on `001-implementation/Ensign`" in text
    assert "Shutting down `001-validation/Ensign`" in text
    assert "nickname returned by `spawn_agent`" in text


def test_codex_runtime_docs_define_bounded_single_entity_post_reuse_stop_sequence():
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert re.search(r"active again.*wait_agent.*same handle", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"shut.*down.*no longer needed.*before stopping", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"unless .*terminal completion.*explicitly requested", text, re.IGNORECASE | re.DOTALL)


def test_codex_runtime_docs_forbid_archive_history_search_in_bounded_single_entity_runs():
    text = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert re.search(r"do not browse .*docs/plans.*_archive", text, re.IGNORECASE | re.DOTALL)
    assert re.search(r"once .*workflow.*entity.*loaded.*stop searching", text, re.IGNORECASE | re.DOTALL)


def test_ensign_stage_report_uses_done_skipped_failed_markers():
    text = read_text("skills/ensign/references/ensign-shared-core.md")
    report_section = section_text(text, "## Stage Report Protocol", (r"^## Completion",))
    assert "DONE:" in report_section
    assert "SKIPPED:" in report_section
    assert "FAILED:" in report_section


def test_pr_merge_mod_copies_share_rich_body_template():
    installed = read_text("docs/plans/_mods/pr-merge.md")
    canonical = read_text("mods/pr-merge.md")

    assert installed == canonical, "pr-merge mod drift: docs/plans/_mods/ and mods/ must match"

    for text in (installed, canonical):
        assert "### PR body template" in text
        assert "Template structure" in text
        assert "Extraction rules" in text
        assert "Motivation lead" in text
        assert "## What changed" in text
        assert "## Evidence" in text
        assert "## Review guidance" in text
        assert "Workflow entity: {entity title}" not in text
        assert "Closes {issue}" in text
        assert "Related" in text
        assert "60-120 words" in text

    for section_name in (
        "Motivation lead",
        "What changed",
        "Evidence",
        "Review guidance",
        "Closes",
        "Related",
    ):
        assert installed.count(section_name) >= 2, (
            f"section name {section_name!r} must appear in both the template "
            f"structure and extraction rules tables"
        )


def test_assembled_claude_first_officer_has_gate_guardrails():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")
    assert "self-approve" in text.lower()
    assert re.search(r"only the captain can approve|never self-approve", text, re.IGNORECASE)
    assert "Gate review:" in text or "gate review" in text.lower()


def test_assembled_claude_first_officer_has_rejection_flow_guardrails():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")
    assert "Feedback Rejection Flow" in text
    assert "feedback-to" in text


def test_assembled_claude_first_officer_has_merge_hook_guardrails():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    merge_section = section_text(text, "## Merge and Cleanup", (r"^## ",))
    gate_section = section_text(text, "## Completion and Gates", (r"^## Feedback", r"^## Merge"))

    assert (
        "merge hooks before any local merge" in merge_section.lower()
        or "run registered merge hooks" in merge_section.lower()
    )
    assert "merge hook" in merge_section.lower()
    assert "registered" in merge_section.lower() or "hook" in merge_section.lower()
    assert re.search(r"before any local merge|before.*local merge", merge_section, re.IGNORECASE)
    assert re.search(r"do not.*local.merge|not local-merge", merge_section, re.IGNORECASE)
    assert re.search(r"terminal.*merge|merge handling", text, re.IGNORECASE)
    assert not re.search(r"Run merge hooks.*_mods", gate_section, re.IGNORECASE)
    assert re.search(r"no merge hook.*default local merge|If no merge", text, re.IGNORECASE)


def test_assembled_claude_first_officer_has_teamcreate_failure_recovery():
    t = TestRunner("agent content", keep_test_dir=False)
    assembled = assembled_agent_content(t, "first-officer")

    # "Already leading team" stays as a narrow startup-only recovery case.
    assert "Already leading team" in assembled
    # Core file references fresh-suffixed recovery and points to recovery file.
    assert re.search(r"fresh[-\s]?suffixed", assembled, re.IGNORECASE), (
        "Core must reference fresh-suffixed TeamCreate recovery."
    )

    # Recovery-specific content lives in the on-demand recovery file.
    recovery_path = Path(__file__).resolve().parent.parent / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-recovery.md"
    recovery_text = recovery_path.read_text()
    assert "Retry to the same team name is banned" in recovery_text

    # Block Agent dispatch while team state is uncertain.
    assert re.search(r"Block all Agent dispatch", assembled)
    assert re.search(r"never dispatch.*while team", assembled, re.IGNORECASE)

    # Sequencing rule in Dispatch Adapter is preserved.
    assert re.search(
        r"Sequencing rule.*Team lifecycle.*Agent.*NEVER.*same tool-call message",
        assembled, re.IGNORECASE | re.DOTALL,
    )


def test_assembled_claude_first_officer_has_no_predispatch_health_check():
    """Rule 1 of the team-fragility issue: the pre-dispatch
    `test -f config.json` probe must not appear in the runtime adapter's
    dispatch path. See tests/test_team_fail_early.py for section-anchored
    coverage of the full fail-early contract."""
    t = TestRunner("agent content", keep_test_dir=False)
    assembled = assembled_agent_content(t, "first-officer")

    # The retired pre-dispatch probe imperatives must be gone. Section-anchored
    # checks for the Dispatch Adapter section live in
    # tests/test_team_fail_early.py; this file guards a few whole-file markers
    # that the old REQUIRED probe paragraph carried.
    assert "Team health check" not in assembled, (
        "Retired 'Team health check' imperative must not appear in the assembled "
        "first-officer contract."
    )
    assert "verified the team is healthy" not in assembled, (
        "Retired 'verified the team is healthy' imperative must not appear."
    )
    assert not re.search(r"not in bare mode or single-entity mode", assembled), (
        "Retired 'skipped in bare mode or single-entity mode' health-check clause "
        "must be gone."
    )

    # Degraded Mode must be declared as a first-class section in the recovery file.
    recovery_path = Path(__file__).resolve().parent.parent / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-recovery.md"
    recovery_text = recovery_path.read_text()
    assert "## Degraded Mode" in recovery_text, (
        "Recovery file must declare the `## Degraded Mode` section."
    )


def test_assembled_claude_first_officer_has_dispatch_idle_guardrail():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    # The guardrail heading must be present
    assert "DISPATCH IDLE GUARDRAIL" in text

    # Idle is normal between-turn state
    assert "idle" in text.lower() and "between-turn state" in text.lower()

    # Three explicit shutdown conditions
    assert "completion message" in text.lower()
    assert "captain explicitly requests shutdown" in text.lower()
    assert "transitioning the entity to a new stage" in text.lower()

    # Never interpret idle as stuck
    assert re.search(r"never interpret idle.*stuck.*unresponsive", text, re.IGNORECASE)


def test_assembled_claude_first_officer_dispatch_template_has_team_mode_completion_signal():
    """The team-mode dispatch path MUST ensure the worker gets a SendMessage completion instruction.

    The structured helper deterministically includes the completion signal in
    team-mode prompts. The break-glass fallback template also includes it.
    Both paths ensure the FO's DISPATCH IDLE GUARDRAIL does not wait forever.
    """
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    core_path = Path(__file__).resolve().parent.parent / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md"
    core_text = core_path.read_text()

    dispatch_section = section_text(core_text, "## Dispatch Adapter", (r"^## ",))

    # The structured helper output template includes an explicit SendMessage completion
    # instruction for team-mode dispatch.
    assert re.search(
        r'SendMessage\(to=\\?"team-lead\\?"',
        dispatch_section,
    ), (
        "Dispatch Adapter section must instruct team-dispatched workers to "
        'SendMessage(to="team-lead", ...) on completion.'
    )

    # The structured helper controls team-mode gating via bare_mode input field.
    assert "bare_mode" in dispatch_section, (
        "Dispatch assembly must reference bare_mode for team-mode gating."
    )

    # The assembled FO contract must contain the same signal (sanity check that the
    # runtime file is actually the one loaded via assembled_agent_content).
    assert re.search(r'SendMessage\(to=\\?"team-lead\\?"', text)


def test_assembled_claude_ensign_has_captain_communication():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "ensign")

    # Captain Communication section exists
    assert "## Captain Communication" in text

    # Direct text output for captain interaction
    assert "direct text output" in text.lower()

    # SendMessage scoped to agent-to-agent use
    assert re.search(
        r"SendMessage.*only.*agent-to-agent", text, re.IGNORECASE | re.DOTALL
    )

    # Captain switches to ensign via Shift+Up/Down
    assert "Shift+Up/Down" in text


def test_assembled_codex_first_officer_has_dispatch_adapter():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer", runtime="codex")

    assert "fork_context=false" in text
    assert "spawn_agent" in text
    assert "TeamCreate" not in text


def test_ensign_stage_report_has_size_guideline():
    text = read_text("skills/ensign/references/ensign-shared-core.md")
    report_section = section_text(text, "## Stage Report Protocol", (r"^## Completion",))
    assert re.search(r"30.50 lines", report_section), (
        "Stage Report Protocol must include a 30-50 line size guideline"
    )


def test_ensign_stage_report_uses_append_mode():
    text = read_text("skills/ensign/references/ensign-shared-core.md")
    report_section = section_text(text, "## Stage Report Protocol", (r"^## Completion",))
    assert "append" in report_section.lower(), (
        "Stage Report Protocol must instruct appending the stage report"
    )


def test_dispatch_template_uses_targeted_read_instruction():
    text = read_text("skills/first-officer/references/claude-first-officer-runtime-core.md")
    dispatch_section = section_text(text, "## Dispatch Adapter", (r"^## ",))
    assert "for full context" not in dispatch_section, (
        "Dispatch template must not unconditionally instruct reading 'for full context'"
    )


def test_fo_completion_reads_last_stage_report():
    text = read_text("skills/first-officer/references/first-officer-shared-core.md")
    completion_section = section_text(text, "## Completion and Gates", (r"^## Feedback",))
    assert "last" in completion_section.lower(), (
        "Completion and Gates must reference reading the last stage report"
    )


def test_first_officer_runtime_docs_use_next_id_for_task_creation():
    shared = read_text("skills/first-officer/references/first-officer-shared-core.md")
    claude_runtime = read_text("skills/first-officer/references/claude-first-officer-runtime-core.md")
    codex_runtime = read_text("skills/first-officer/references/codex-first-officer-runtime.md")

    assert "status --next-id" in shared
    assert "status --boot" in shared
    assert "status --next-id" in claude_runtime
    assert "status --next-id" in codex_runtime


def test_assembled_codex_ensign_has_completion_summary_contract():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "ensign", runtime="codex")

    assert "completion summary" in text.lower()
    assert "logical worker id" in text.lower()
    assert "SendMessage" not in text


def test_assembled_claude_first_officer_has_context_budget_in_reuse_conditions():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    reuse_section = section_text(text, "## Completion and Gates", (r"^## Feedback", r"^## Merge"))
    assert "claude-team context-budget" in reuse_section, (
        "Reuse conditions must reference claude-team context-budget check"
    )


def test_assembled_claude_first_officer_has_context_budget_in_feedback_rejection():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    rejection_section = section_text(text, "## Feedback Rejection Flow", (r"^## Merge",))
    assert "claude-team context-budget" in rejection_section, (
        "Feedback rejection flow must reference claude-team context-budget check"
    )


def test_assembled_claude_first_officer_runtime_has_context_budget_section():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    assert "claude-team" in text
    assert "cooperative" in text.lower()
    assert "zombie" in text.lower()
    assert "uncommitted" in text.lower()


def test_assembled_skill_contract_exposes_resolved_include_paths():
    t = TestRunner("agent content", keep_test_dir=False)
    text = assembled_agent_content(t, "first-officer")

    assert "skill include resolution" in text
    assert "first-officer-shared-core.md" in text
    assert "code-project-guardrails.md" in text


def test_assembled_claude_first_officer_has_structured_dispatch():
    """AC-11: Runtime adapter instructs the FO to use claude-team build for dispatch assembly."""
    t = TestRunner("agent content", keep_test_dir=False)
    assembled = assembled_agent_content(t, "first-officer")

    core_path = Path(__file__).resolve().parent.parent / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md"
    runtime_text = core_path.read_text()

    dispatch_section = section_text(runtime_text, "## Dispatch Adapter", (r"^## ",))

    # The dispatch section references the helper
    assert "claude-team build" in dispatch_section, (
        "Dispatch Adapter must reference claude-team build"
    )

    # The dispatch section has the input JSON shape
    assert '"schema_version": 1' in dispatch_section
    assert '"entity_path"' in dispatch_section
    assert '"checklist"' in dispatch_section

    # The 4-step process is present
    assert "Pipe the JSON to the helper" in dispatch_section
    assert "On exit 0, parse the stdout JSON" in dispatch_section
    assert "On non-zero exit" in dispatch_section

    # The assembled FO contract must also contain the helper reference
    assert "claude-team build" in assembled


def test_assembled_claude_first_officer_has_break_glass_dispatch():
    """AC-12: Runtime adapter contains Break-Glass Manual Dispatch with minimal Agent() template."""
    recovery_path = Path(__file__).resolve().parent.parent / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-recovery.md"
    recovery_text = recovery_path.read_text()

    breakglass_section = section_text(recovery_text, "## Break-Glass Manual Dispatch", (r"^## ",))

    # Break-glass section exists
    assert breakglass_section, (
        "Recovery file must contain Break-Glass Manual Dispatch section"
    )

    # The break-glass template includes the essential Agent() fields
    assert 'subagent_type="{dispatch_agent_id}"' in breakglass_section
    assert 'name="{worker_key}-{slug}-{stage}"' in breakglass_section
    assert 'team_name="{team_name}"' in breakglass_section

    # It has the SendMessage completion signal
    assert re.search(
        r'SendMessage\(to=\\?"team-lead\\?"',
        breakglass_section,
    )


def test_assembled_claude_first_officer_has_bare_mode_guardrail():
    """Implementation Note 6: bare_mode guardrail sentence is present in dispatch prose."""
    core_path = Path(__file__).resolve().parent.parent / "skills" / "first-officer" / "references" / "claude-first-officer-runtime-core.md"
    runtime_text = core_path.read_text()

    dispatch_section = section_text(runtime_text, "## Dispatch Adapter", (r"^## ",))

    assert (
        "bare_mode" in dispatch_section
        and "never infer it from the stage" in dispatch_section
        and "live team state" in dispatch_section
    ), (
        "Dispatch prose must contain the bare_mode guardrail: "
        "'bare_mode field must match the current dispatch context — "
        "never infer it from the stage, always from the live team state'"
    )


def test_shared_core_grep_over_read_discipline_for_entity_body():
    """AC-2 (#159): shared-core `## Probe and Ideation Discipline` carries the Grep-over-Read
    rule for entity-body inspection and warns about the Claude-Code Read + `status --set`
    full-file staleness echo.
    """
    text = read_text("skills/first-officer/references/first-officer-shared-core.md")
    discipline = section_text(text, "## Probe and Ideation Discipline", (r"^## ",))

    # Grep-over-Read anchor for entity-body inspection
    assert re.search(r"prefer Grep over Read for targeted entity-body inspection", discipline, re.IGNORECASE), (
        "Probe and Ideation Discipline must prefer Grep over Read for entity-body inspection"
    )

    # Explicit warning about Read + status --set echoing the full file
    assert "status --set" in discipline
    assert re.search(r"file-staleness", discipline, re.IGNORECASE)
    assert re.search(r"system-reminder|echoes the entire current file", discipline, re.IGNORECASE)

    # `status --set` stdout narration affordance is called out
    assert "field: old -> new" in discipline


def test_claude_runtime_points_to_shared_core_entity_body_inspection_rule():
    """AC-2 (#159): the Claude runtime adapter carries a short pointer to the shared-core rule."""
    text = read_text("skills/first-officer/references/claude-first-officer-runtime-core.md")
    assert "## Probe and Ideation Discipline" in text, (
        "Claude runtime must point to the shared-core `## Probe and Ideation Discipline` section"
    )
    assert "status --set" in text


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
