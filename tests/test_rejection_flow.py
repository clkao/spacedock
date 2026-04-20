# ABOUTME: E2E test for the validation rejection flow in the first-officer template.
# ABOUTME: Pinned to teams_mode; asserts impl + validation dispatches + SendMessage feedback routing + reviewer-reuse for re-validation (#141).

from __future__ import annotations

import shutil
import subprocess
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
    rejection_signal_present,
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


def _is_send_message_to(entry: dict, recipient_substr: str) -> bool:
    block = _is_tool_use(entry, "SendMessage")
    if not block:
        return False
    inp = block.get("input") or {}
    return recipient_substr in str(inp.get("to", ""))


def _is_team_create(entry: dict) -> bool:
    return _is_tool_use(entry, "TeamCreate") is not None


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_rejection_flow(test_project, model, effort):
    """FO drives teams-mode rejection flow: impl -> validation REJECTED -> feedback routes back -> validation reviewer reused via SendMessage for re-review (#141)."""
    t = test_project

    if model == "claude-haiku-4-5" or "haiku" in model.lower():
        pytest.xfail(
            reason=(
                "pending haiku-teams rejection flow — haiku-4-5 drops the "
                "keep-alive Bash-probe discipline at `system init` cycle "
                "boundaries and hallucinates teardown "
                "(anthropics/claude-code#26426 class; opus-4-7 green)"
            )
        )

    print("--- Phase 1: Set up test project from fixture ---")
    fixture_dir = t.repo_root / "tests" / "fixtures" / "rejection-flow"
    setup_fixture(t, "rejection-flow", "rejection-pipeline")
    install_agents(t, include_ensign=True)

    shutil.copy2(fixture_dir / "math_ops.py", t.test_project_dir)
    tests_dir = t.test_project_dir / "tests"
    tests_dir.mkdir(exist_ok=True)
    shutil.copy2(fixture_dir / "tests" / "test_add.py", tests_dir)
    git_add_commit(t.test_project_dir, "setup: rejection flow fixture with buggy implementation")

    status_cmd = ["python3", str(t.repo_root / "skills" / "commission" / "bin" / "status"),
                  "--workflow-dir", "rejection-pipeline"]
    t.check_cmd("status script runs without errors", status_cmd, cwd=t.test_project_dir)
    status_result = subprocess.run(
        status_cmd + ["--next"], capture_output=True, text=True, cwd=t.test_project_dir,
    )
    t.check("status --next detects dispatchable entity",
            "buggy-add-task" in status_result.stdout)
    print()

    print("--- Phase 2: Run first officer (claude) ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the rejection-flow path."
        )

    abs_workflow = t.test_project_dir / "rejection-pipeline"
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
            "--max-budget-usd", "5.00",
            "--append-system-prompt", headless_hint,
        ],
        dispatch_budget=DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0),
    ) as w:
        w.expect(_is_team_create, timeout_s=PER_STAGE_OVERALL_S, label="TeamCreate emitted")
        print("[OK] TeamCreate emitted (teams mode engaged)")

        impl_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="implementation",
            label="cycle-1 implementation dispatch close",
        )
        print(f"[OK] cycle-1 implementation dispatch closed in {impl_record.elapsed:.1f}s")

        validation_record = w.expect_dispatch_close(
            overall_timeout_s=PER_STAGE_OVERALL_S,
            dispatch_budget_s=PER_DISPATCH_BUDGET_S,
            ensign_name="validation",
            label="cycle-1 validation dispatch close",
        )
        print(f"[OK] cycle-1 validation dispatch closed in {validation_record.elapsed:.1f}s")

        # Feedback routing: FO must route back to implementation. Accept
        # either SendMessage to the impl ensign (reuse) OR a fresh Agent
        # re-dispatch of implementation. Whichever fires first is the
        # observable signal that feedback routing happened.
        w.expect(
            lambda e: (
                _is_send_message_to(e, "implementation")
                or (
                    _is_tool_use(e, "Agent") is not None
                    and "implementation" in str(((_is_tool_use(e, "Agent") or {}).get("input") or {}).get("description", ""))
                )
            ),
            timeout_s=PER_STAGE_OVERALL_S,
            label="feedback routed back to implementation (SendMessage or fresh Agent)",
        )
        print("[OK] feedback routed back to implementation")

        # The #141 contract: after the fix, FO reuses the kept-alive
        # validation reviewer via SendMessage for re-review. This is
        # the key signal the original test's `ensign_count >= 3` wasn't
        # distinguishing from "fresh validation re-dispatch".
        w.expect(
            lambda e: _is_send_message_to(e, "validation"),
            timeout_s=PER_STAGE_OVERALL_S,
            label="SendMessage to validation reviewer for re-review (#141 keepalive)",
        )
        print("[OK] validation reviewer reused via SendMessage for re-review (#141)")

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

    # Dispatch count analysis. Accept either:
    #   3 Agents (impl, validation, impl-fix) — reuse path: validation re-review via SendMessage
    #   4 Agents (impl, validation, impl-fix, validation-recheck) — fresh path
    # Per #141 the reuse path is preferred; we don't enforce it strictly
    # because the context-budget check is a legitimate branch to fresh.
    t.check(
        "FO emitted 3 or 4 ensign Agent() dispatches (impl + validation + impl-fix +/- validation-recheck)",
        len(records) in (3, 4),
    )

    print()
    print("[Rejection Signal Present]")
    entity_main = t.test_project_dir / "rejection-pipeline" / "buggy-add-task.md"
    worktrees_dir = t.test_project_dir / ".worktrees"
    t.check(
        "reviewer stage report contains REJECTED recommendation",
        rejection_signal_present("rejection-pipeline", "buggy-add-task", entity_main, worktrees_dir, "", ""),
    )

    print()
    print("[Entity Advancement]")
    entity_archive = t.test_project_dir / "rejection-pipeline" / "_archive" / "buggy-add-task.md"
    if entity_archive.is_file():
        t.pass_("entity advanced to terminal stage and was archived")
    elif entity_main.is_file():
        fm = read_entity_frontmatter(entity_main)
        status_val = fm.get("status", "")
        if status_val == "done":
            t.pass_(f"entity advanced to terminal stage (status: {status_val})")
        else:
            t.fail(f"entity did not reach terminal stage (status: {status_val!r})")
    else:
        t.fail("entity file missing from both main and _archive")

    t.finish()
