---
id: 132
title: Codex first officer: derive reusable context/agent visibility from ~/.codex runtime state
status: ideation
source: FO observation during tasks 130/131/117 on 2026-04-11
started: 2026-04-15T05:18:01Z
completed:
verdict:
score: 0.62
worktree:
issue:
pr:
---

## Problem Statement

Claude-team gives the first officer a supported `context-budget` helper that emits a `reuse_ok` decision before worker reuse. The Codex path has no equivalent supported budget command today. It does have runtime surfaces that expose partial visibility:

- supported in-session handles and events from Codex multi-agent tools: `spawn_agent`, `send_input`, `wait_agent`, `close_agent`, agent messages, and `codex exec --json` event records
- Spacedock's existing Codex runtime contract in `skills/first-officer/references/codex-first-officer-runtime.md`, which already treats addressability, active-again state, critical-path waits, and explicit shutdown as first-class orchestration facts
- Spacedock's test harness in `scripts/test_lib.py`, especially `run_codex_first_officer()` and `CodexLogParser`, which already parses collab tool calls, agent messages, spawned-worker counts, and completed worker messages
- undocumented local Codex persistence under `$CODEX_HOME` / `~/.codex`; in this checkout, a bounded schema-only probe found `state_5.sqlite` tables such as `threads`, `thread_spawn_edges`, `agent_jobs`, and `agent_job_items`, plus `logs_2.sqlite.logs`

The missing design decision is how much Spacedock should rely on each category. Runtime tool handles and `--json` event logs are the only supportable source for immediate reuse decisions. Files such as `state_*.sqlite`, `logs_*.sqlite`, and any future `sessions/*.jsonl` may be useful for diagnostics or operator reporting, but they are not a stable contract and should not become mandatory for correctness.

## Scope Boundary

In scope:

- Define a Codex visibility layer that produces a small, explicit reuse observation for a worker handle: addressable, active or completed, active-again after routed reuse, last observed completion source, and whether the worker has been explicitly shut down.
- Identify whether Codex exposes enough supported signal to replace or approximate the Claude-only `claude-team context-budget --name ...` pre-reuse check on the Codex path.
- Add contract wording that distinguishes supported signals from best-effort diagnostics and explains fail-closed behavior when a signal is missing.
- Extend tests around `CodexLogParser`, runtime contract wording, and bounded Codex E2E logs so future changes preserve the visibility semantics.

Out of scope:

- Exact parity with Claude Code `context-budget`, including a precise token-window percentage or model-specific context limit, unless Codex exposes a supported source for it.
- A required dependency on private Codex database schemas, sqlite file names, log targets, or session storage layout.
- Scraping sensitive prompt or response text from `$CODEX_HOME` to make orchestration decisions.
- Building a generic Codex runtime inspector for unrelated workflows.

## Proposed Approach

1. Treat live Codex tool state as authoritative for reuse bookkeeping.

   The first officer should continue to preserve the live worker handle returned by `spawn_agent`, mark the handle active again after `send_input`, and require `wait_agent` on that same handle when the reused result is on the entity's critical path. The visibility layer should make these facts explicit in a small internal observation, rather than burying them only in prose. The implementation surface is `skills/first-officer/references/codex-first-officer-runtime.md`, plus any helper/test wording that assembles Codex dispatch prompts.

2. Define a Codex-specific reuse decision that does not pretend to know context budget.

   A Codex worker is reusable only when the shared non-budget reuse conditions pass, the live handle remains addressable, the worker is not already active on another critical path, and no explicit shutdown has been recorded. If the visibility layer cannot establish addressability or active/completed state from supported runtime data, reuse must fail closed to fresh dispatch. Any `reuse_ok`-style field on Codex should be named to avoid implying token-budget parity, for example `reuse_allowed` with a `basis` such as `live_handle`, `json_log`, or `diagnostic_state`.

3. Keep undocumented `$CODEX_HOME` inspection diagnostic and optional.

   A future helper may inspect local Codex persistence only behind an explicit diagnostic command or test-only probe. It should read schema/metadata defensively, tolerate absent files, and never require specific table names for normal orchestration. Current local observations are useful starting points: `threads` contains `tokens_used`, `agent_role`, `model`, `cli_version`, and archive metadata; `thread_spawn_edges` contains parent/child relationships and edge status. Those names are empirical, not contractual.

4. Extend parser and harness tests before relying on the signal.

   The existing `CodexLogParser` already exposes `collab_tool_calls()`, `agent_message_texts()`, `spawn_count()`, and `completed_agent_messages()`. Implementation should add focused helpers only if they reduce repeated test logic, such as extracting current state by thread id or checking that a `send_input` target was later awaited and closed. Existing surfaces to update or verify include:

   - `scripts/test_lib.py`
   - `tests/test_test_lib_helpers.py`
   - `tests/test_agent_content.py`
   - `tests/test_codex_packaged_agent_e2e.py`
   - `tests/test_rejection_flow_codex.py`, currently skipped for Codex feedback reuse coverage
   - `skills/first-officer/references/first-officer-shared-core.md` only where shared wording must avoid assuming the Claude-only `context-budget` helper exists on Codex

5. Surface operator-facing diagnostics without overclaiming.

   Codex FO messages should say what was observed: the stable worker label, logical id, handle/thread id, current active/completed/shutdown interpretation, and whether reuse is allowed or fresh dispatch is safer. If the only available evidence is diagnostic `$CODEX_HOME` state, the message should label it as diagnostic and not use it as the sole reason to reuse a worker.

## Acceptance criteria

**AC-1: Codex runtime contract separates supported visibility from diagnostic runtime-state inspection.**

Tested by a static content test in `tests/test_agent_content.py` that asserts the Codex FO runtime names live tool handles / `codex exec --json` as supported signals and labels `$CODEX_HOME` sqlite/jsonl inspection as optional diagnostic, not required orchestration state.

**AC-2: Codex reuse decisions fail closed when live handle addressability or active/completed state cannot be established.**

Tested by static contract assertions and parser unit tests with synthetic Codex JSONL events that omit receiver thread ids, omit completion state, or show a closed/shutdown handle; each fixture should produce a non-reusable or fresh-dispatch decision.

**AC-3: Routed Codex reuse is represented as an active follow-up on the same handle until `wait_agent` observes the new completion.**

Tested by extending `CodexLogParser` unit coverage and the existing packaged-agent E2E assertions so a `send_input` target must be followed by `wait` on the same `receiver_thread_ids` value before the FO treats the routed work as complete.

**AC-4: Codex visibility reporting includes stable Spacedock worker identity and observed basis.**

Tested by existing or extended E2E log assertions that operator-facing messages include the human-readable worker label, `spacedock:ensign`, the handle/thread id, and a reason such as `live_handle` or `json_log` when reuse is allowed or rejected.

**AC-5: Undocumented Codex persistence can change without breaking normal workflow tests.**

Tested by unit tests for any diagnostic inspector using temporary `$CODEX_HOME` directories with missing sqlite files, unknown table sets, malformed JSONL, and unreadable files. The expected result is a warning/unknown diagnostic signal, not an exception or reuse approval.

**AC-6: The shared reuse prose no longer implies the Claude-only `context-budget` command is available on every runtime.**

Tested by static assertions that the shared core either delegates the pre-reuse budget/visibility check to runtime-specific wording or explicitly scopes `claude-team context-budget --name ...` to the Claude runtime.

## Test Plan

Static tests are the cheapest and should be required for contract wording:

- `uv run --with pytest python tests/test_agent_content.py`
- focused assertions that Codex runtime wording covers supported signals, diagnostic-only `$CODEX_HOME` inspection, fail-closed reuse, and stable worker labels

Parser/unit tests should cover the risky logic without live Codex cost:

- `uv run --with pytest python tests/test_test_lib_helpers.py`
- synthetic `codex exec --json` fixtures for `spawn_agent`, `send_input`, `wait`, missing `receiver_thread_ids`, stale `send_input` completion echoes, and `close_agent`
- temporary `$CODEX_HOME` fixtures for diagnostic inspector behavior if an inspector is added

Runtime E2E is needed only for behavior that depends on real Codex multi-agent events:

- `uv run tests/test_codex_packaged_agent_e2e.py` or the equivalent runtime-live CI job when Codex credentials and runtime budget are available
- the E2E should verify that a feedback routed reuse uses `send_input`, waits on the same handle, reports active-again state, and explicitly shuts down no-longer-needed workers
- unskipping or replacing the Codex rejection-flow test should be treated as a follow-up only if the implementation changes runtime behavior beyond parser/contract wording

Estimated complexity is medium. Most work is contract and parser-level, but any claim that depends on live Codex multi-agent state needs at least one live E2E or preserved runtime-live artifact because `codex exec --json` and multi-agent event shape are external runtime surfaces.

## Stage Report: ideation

- DONE: A scoped problem statement distinguishes supported Codex visibility signals from brittle runtime-state spelunking.
  Evidence: `## Problem Statement` and `## Scope Boundary` separate live Codex handles / `--json` events from optional `$CODEX_HOME` sqlite/jsonl diagnostics.
- DONE: The proposed approach names concrete runtime/contract/test surfaces an implementation worker can edit or verify.
  Evidence: `## Proposed Approach` names `codex-first-officer-runtime.md`, `first-officer-shared-core.md`, `scripts/test_lib.py`, and relevant Codex tests.
- DONE: Acceptance criteria and test plan are entity-level, reproducible, and proportional to the risk of relying on Codex state.
  Evidence: `## Acceptance criteria` pairs each end-state property with a test, and `## Test Plan` separates static/unit checks from live Codex E2E.

### Summary

Fleshed out the task into a concrete design target for Codex worker visibility. The plan keeps supported live runtime handles authoritative, treats `$CODEX_HOME` persistence as diagnostic only, and defines tests that fail closed when Codex state is absent or ambiguous.
