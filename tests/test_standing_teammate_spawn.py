# ABOUTME: Live E2E test for standing-teammate mod spawn + FO routing (#162).
# ABOUTME: Verifies `standing: true` mod auto-spawns a teammate and echo-agent roundtrip reaches entity body.

from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import (  # noqa: E402
    DispatchBudget,
    LogParser,
    emit_skip_result,
    git_add_commit,
    install_agents,
    probe_claude_runtime,
    run_first_officer_streaming,
    setup_fixture,
    tool_use_matches,
)


PER_STAGE_OVERALL_S = 60
ENSIGN_OVERALL_S = 300
ENSIGN_DISPATCH_BUDGET_S = 180
SUBPROCESS_EXIT_BUDGET_S = 180


def _agent_input(entry: dict) -> dict:
    """Extract the input dict of the first Agent() tool_use block in an entry."""
    message = entry.get("message", {})
    for block in message.get("content", []) or []:
        if isinstance(block, dict) and block.get("type") == "tool_use" and block.get("name") == "Agent":
            return block.get("input", {}) or {}
    return {}


@pytest.mark.live_claude
@pytest.mark.teams_mode
def test_standing_teammate_spawns_and_roundtrips(test_project, model, effort):
    """AC-12/AC-14: standing: true mod spawns a teammate the FO can route to.

    Fixture declares one standing teammate (`echo-agent`) via
    `_mods/echo-agent.md`. Progressive per-step assertions ensure fail-fast
    diagnostics: each milestone names the step that stalled.

    Observable contract (FO-visible in fo-log.jsonl):
      1. TeamCreate emitted (teams-mode engaged).
      2. `claude-team spawn-standing` Bash invoked by FO.
      3. echo-agent Agent() dispatched (standing teammate spawn).
      4. Ensign Agent() dispatched; its prompt contains the standing-teammates
         section listing echo-agent (AC-14).
      5. Ensign Done: close observed via inbox-poll anchor.

    Ground-truth deliverable (post-run, on-disk):
      6. Archived entity body contains `ECHO: ping` — proof the ensign
         completed the SendMessage roundtrip and captured the echo reply.

    The ensign's internal SendMessage(echo-agent) and Edit(entity body) live
    in the ensign's subagent jsonl, not the FO stream, so they are not
    asserted as FO-visible tool_uses (unreachable under `claude -p` teams-mode
    per anthropics/claude-code#26426).
    """
    t = test_project

    print("--- Phase 1: Set up fixture (workflow in subdir) ---")
    setup_fixture(t, "standing-teammate", "standing-teammate")
    install_agents(t, include_ensign=True)
    git_add_commit(t.test_project_dir, "setup: standing-teammate fixture")
    print()

    print("--- Phase 2: Run first officer with streaming watcher ---")
    ok, reason = probe_claude_runtime(model)
    if not ok:
        emit_skip_result(
            f"live Claude runtime unavailable before FO dispatch: {reason}. "
            "This environment cannot currently prove or disprove the standing-teammate path."
        )

    abs_workflow = t.test_project_dir / "standing-teammate"
    entity = abs_workflow / "001-echo-roundtrip.md"
    archived = abs_workflow / "_archive" / "001-echo-roundtrip.md"
    prompt = (
        f"Process the workflow at {abs_workflow}/ to terminal completion. "
        "During startup, spawn every standing teammate declared in "
        "_mods/*.md with standing: true. Then drive task 001 through its "
        "stages. The task instructs its ensign to SendMessage echo-agent "
        "with 'ping' and capture the reply — do not skip that step. Stop "
        "once the entity is archived."
    )

    # Headless runtime workaround (anthropics/claude-code#26426): see
    # test_feedback_keepalive.py for the full rationale. Summary: under
    # `claude -p`, the InboxPoller React hook never fires, so inbox-delivered
    # teammate messages accumulate on disk but never surface to the FO's
    # stream. The append-system-prompt installs an inbox-polling rule that
    # (a) keeps each idle turn ending with Bash tool_use (preserving the
    # prompt cycle past opus-4-7-low's text-only teardown hallucination) and
    # (b) surfaces real ensign `Done:` messages into the FO's stream as Bash
    # tool_result, which the watcher's inbox-poll close anchor recognizes.
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
            "--max-budget-usd", "2.00",
            "--append-system-prompt", headless_hint,
        ],
        dispatch_budget=DispatchBudget(soft_s=30.0, hard_s=180.0, shutdown_grace_s=10.0),
    ) as w:
        w.expect(
            lambda e: tool_use_matches(e, "TeamCreate"),
            timeout_s=PER_STAGE_OVERALL_S,
            label="TeamCreate emitted",
        )
        print("[OK] TeamCreate emitted (teams mode engaged)")

        w.expect(
            lambda e: tool_use_matches(e, "Bash", command="spawn-standing"),
            timeout_s=PER_STAGE_OVERALL_S,
            label="claude-team spawn-standing invoked",
        )
        print("[OK] claude-team spawn-standing invoked")

        w.expect(
            lambda e: tool_use_matches(e, "Agent", name="echo-agent"),
            timeout_s=PER_STAGE_OVERALL_S,
            label="echo-agent Agent() dispatched",
        )
        print("[OK] echo-agent Agent() dispatched")

        ensign_dispatch = w.expect(
            lambda e: tool_use_matches(e, "Agent")
            and _agent_input(e).get("subagent_type") == "spacedock:ensign",
            timeout_s=PER_STAGE_OVERALL_S,
            label="ensign Agent() dispatched",
        )
        ensign_prompt = _agent_input(ensign_dispatch).get("prompt", "")
        assert "### Standing teammates available in your team" in ensign_prompt, (
            "Ensign dispatch prompt missing the standing-teammates section. "
            f"Prompt preview: {ensign_prompt[:200]!r}"
        )
        assert "echo-agent" in ensign_prompt, (
            "Standing-teammates section did not list echo-agent by name. "
            f"Prompt preview: {ensign_prompt[:200]!r}"
        )
        print("[OK] ensign dispatch prompt includes standing-teammates section with echo-agent")

        ensign_record = w.expect_dispatch_close(
            overall_timeout_s=ENSIGN_OVERALL_S,
            dispatch_budget_s=ENSIGN_DISPATCH_BUDGET_S,
            ensign_name="work",
            label="ensign dispatch close (inbox-poll anchor)",
        )
        print(f"[OK] ensign dispatch closed in {ensign_record.elapsed:.1f}s")

        # Release the FO's keep-alive sentinel so it can emit its final
        # end_turn and exit cleanly. Post-contract FO activity (terminal
        # cleanup, archive, shutdown echo-agent) is not asserted; if it
        # exceeds the exit budget, the context manager kills it without
        # failing the test.
        keepalive_done.touch()
        print(f"[OK] keep-alive sentinel {keepalive_done.name} touched")

        try:
            w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)
            print("[OK] FO exited cleanly after sentinel")
        except Exception as exc:
            print(f"  NOTE: FO did not exit within {SUBPROCESS_EXIT_BUDGET_S}s post-sentinel ({type(exc).__name__}); contract assertions already passed")

    print()
    print("--- Phase 3: On-disk ground-truth assertion ---")
    body = None
    for path in (archived, entity):
        if path.is_file():
            body = path.read_text()
            print(f"  inspecting entity body at {path}")
            break
    assert body is not None, (
        f"Neither archived ({archived}) nor in-progress ({entity}) entity file exists. "
        "Ensign did not write the entity body at all."
    )
    assert "ECHO: ping" in body, (
        "Archived entity body does not contain 'ECHO: ping'. "
        "Ensign did not capture the echo-agent reply to the entity body.\n"
        f"Body tail (last 600 chars):\n{body[-600:]}"
    )
    print("[OK] entity body on-disk contains 'ECHO: ping' (data-flow ground truth)")

    print()
    print("--- Phase 4: Aggregate Agent() assertion ---")
    log = LogParser(t.log_dir / "fo-log.jsonl")
    agent_calls = log.agent_calls()
    echo_spawns = [c for c in agent_calls if c.get("name") == "echo-agent"]
    assert echo_spawns, (
        "Aggregate check: no echo-agent Agent() call in final log. "
        f"Agent() calls seen: {[(c.get('name'), c.get('subagent_type')) for c in agent_calls]}"
    )
    print(f"[OK] aggregate: echo-agent Agent() dispatched {len(echo_spawns)} time(s)")
