# ABOUTME: E2E regression test for team-mode dispatch completion-signal in the FO template.
# ABOUTME: Pinned to teams_mode; asserts TeamCreate + work dispatch close + entity advances to done.

from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    DispatchBudget,
    emit_skip_result,
    git_add_commit,
    install_agents,
    probe_claude_runtime,
    read_entity_frontmatter,
    run_first_officer_streaming,
    setup_fixture,
)


PER_STAGE_OVERALL_S = 120
PER_DISPATCH_BUDGET_S = 90

SUBPROCESS_EXIT_BUDGET_S = 180


def _is_tool_use(entry: dict, name: str) -> dict | None:
    if entry.get("type") != "assistant":
        return None
    msg = entry.get("message") or {}
    for block in (msg.get("content") or []):
        if (
            isinstance(block, dict)
            and block.get("type") == "tool_use"
            and block.get("name") == name
        ):
            return block
    return None


def _is_team_create(entry: dict) -> bool:
    return _is_tool_use(entry, "TeamCreate") is not None


def _first_agent_dispatch_prompt(fo_log_path: Path) -> str:
    """Return the prompt text from the first Agent(subagent_type='spacedock:ensign') dispatch in fo-log."""
    import json
    with open(fo_log_path) as fh:
        for raw in fh:
            try:
                entry = json.loads(raw)
            except json.JSONDecodeError:
                continue
            if entry.get("type") != "assistant":
                continue
            for block in (entry.get("message") or {}).get("content") or []:
                if (
                    isinstance(block, dict)
                    and block.get("type") == "tool_use"
                    and block.get("name") == "Agent"
                ):
                    inp = block.get("input") or {}
                    if inp.get("subagent_type") == "spacedock:ensign":
                        return str(inp.get("prompt") or "")
    return ""


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_dispatch_completion_signal(test_project, model, effort):
    """FO drives teams-mode single-stage completion: TeamCreate -> work dispatch close -> entity advances to done."""
    t = test_project

    # haiku-4-5 drops keep-alive discipline under claude -p (#26426 class);
    # matches the cycle-7 haiku xfail pattern on test_feedback_keepalive.
    if model == "claude-haiku-4-5" or model == "haiku" or "haiku" in model.lower():
        pytest.xfail(
            reason=(
                "pending haiku-teams completion-signal — haiku-4-5 drops the "
                "keep-alive Bash-probe discipline at `system init` cycle "
                "boundaries and hallucinates teardown "
                "(anthropics/claude-code#26426 class; opus-4-7 green)"
            )
        )

    print("--- Phase 1: Set up test project from fixture ---")
    setup_fixture(t, "completion-signal-pipeline", "completion-signal-pipeline")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: completion-signal regression fixture")
    status_cmd = [
        "python3",
        str(t.repo_root / "skills" / "commission" / "bin" / "status"),
        "--workflow-dir", "completion-signal-pipeline",
    ]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    print()

    print("--- Phase 2: Run first officer (claude) ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the completion-signal regression."
        )

    abs_workflow = t.test_project_dir / "completion-signal-pipeline"
    prompt = f"Process all tasks through the workflow at {abs_workflow}/ to terminal completion."

    keepalive_done = t.test_project_dir / ".fo-keepalive-done"
    poll_script = t.repo_root / "scripts" / "fo_inbox_poll.py"
    seen_file = t.test_project_dir / ".fo-inbox-seen"
    headless_hint = (
        f"The spacedock plugin directory is at `{t.repo_root}`. Use it "
        f"directly; do NOT run `find / -name claude-team` — the binaries you "
        f"need are `{t.repo_root}/skills/commission/bin/status` and "
        f"`{t.repo_root}/skills/commission/bin/claude-team`.\n\n"
        f"HEADLESS INBOX-POLLING RULE. You are running in `claude -p` headless "
        f"mode. Per anthropics/claude-code#26426, inbox-delivered teammate "
        f"messages accumulate on disk at `$HOME/.claude/teams/{{team_name}}/"
        f"inboxes/team-lead.json` but are NOT surfaced to your stream. The "
        f"workaround is to surface them yourself via an external polling "
        f"script.\n\n"
        f"Until the sentinel file `{keepalive_done}` exists, every turn "
        f"MUST end with a Bash tool_use (not text) that runs the poll "
        f"script:\n\n"
        f"    python3 {poll_script} --home \"$HOME\" --pattern 'Done:' "
        f"--timeout 5 --seen-file {seen_file}\n\n"
        f"The script blocks up to 5 seconds waiting for a new inbox "
        f"message whose text contains 'Done:'. Its stdout contains the "
        f"teammate message (or is empty on timeout, in which case repeat). "
        f"Treat any 'from: spacedock-ensign-...' block with 'text: Done: "
        f"... completed {{stage}}' as the teammate's completion signal for "
        f"that stage — proceed to the next workflow step per shared-core "
        f"discipline. Never emit `SendMessage(shutdown_request)`, "
        f"`TeamDelete`, or other teardown while awaiting an ensign. Once "
        f"the workflow reaches terminal completion, you may end with text."
    )

    with run_first_officer_streaming(
        t,
        prompt,
        agent_id="spacedock:first-officer",
        extra_args=[
            "--model", model,
            "--effort", effort,
            "--max-budget-usd", "3.00",
            "--append-system-prompt", headless_hint,
        ],
        dispatch_budget=DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0),
    ) as w:
        w.expect(_is_team_create, timeout_s=PER_STAGE_OVERALL_S, label="TeamCreate emitted")
        print("[OK] TeamCreate emitted (teams mode engaged)")

        work_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="work",
            label="work dispatch close",
        )
        print(f"[OK] work dispatch closed in {work_record.elapsed:.1f}s")

        # Workflow contract satisfied — release the keep-alive sentinel.
        keepalive_done.touch()
        print(f"[OK] keep-alive sentinel {keepalive_done.name} touched")

        try:
            w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)
            print("[OK] FO exited cleanly after sentinel")
        except Exception as exc:
            print(f"  NOTE: FO did not exit within {SUBPROCESS_EXIT_BUDGET_S}s post-sentinel ({type(exc).__name__}); contract assertions already passed")

    print("--- Phase 3: Validation ---")

    records = w.dispatch_records
    print(f"  dispatch records: {[(r.ensign_name, round(r.elapsed, 1)) for r in records]}")
    t.check(
        "FO emitted exactly one ensign Agent() dispatch (work stage)",
        len(records) == 1,
    )

    print()
    print("[Entity Advancement]")
    entity_main = t.test_project_dir / "completion-signal-pipeline" / "completion-signal-task.md"
    entity_archive = t.test_project_dir / "completion-signal-pipeline" / "_archive" / "completion-signal-task.md"

    if entity_archive.is_file():
        t.pass_("entity advanced and was archived without manual captain intervention")
    elif entity_main.is_file():
        fm = read_entity_frontmatter(entity_main)
        status_val = fm.get("status", "")
        if status_val == "done":
            t.pass_(f"entity advanced to terminal stage (status: {status_val})")
        else:
            t.fail(
                f"entity did NOT advance past dispatched stage (status: {status_val!r}). "
                "This reproduces the bug: team-dispatched ensign sent no completion signal, "
                "so the FO's DISPATCH IDLE GUARDRAIL waited forever."
            )
    else:
        t.fail("entity file missing from both main and _archive (unexpected state)")

    print()
    print("[Dispatch Template Sanity]")
    fo_log_path = t.log_dir / "fo-log.jsonl"
    dispatch_prompt = _first_agent_dispatch_prompt(fo_log_path)
    t.check(
        "team-mode ensign prompt carries SendMessage completion-signal instruction",
        'SendMessage(to="team-lead"' in dispatch_prompt,
    )

    t.finish()
