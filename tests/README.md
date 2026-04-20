# Test Authoring Guidelines

This document covers the Spacedock test infrastructure, conventions, and how to write tests that integrate with the existing harness.

## Test Infrastructure Overview

Tests live in `tests/` and share two library modules under `scripts/`:

| Module | Purpose | Session type |
|--------|---------|-------------|
| `scripts/test_lib.py` | Non-interactive tests via `claude -p` or `codex exec` | Pipe mode (structured JSONL logs) |
| `scripts/test_lib_interactive.py` | Interactive multi-turn tests via PTY | Interactive TUI (raw terminal output) |

### `test_lib.py` — Non-Interactive Harness

Provides `TestRunner`, `LogParser`, `CodexLogParser`, and project setup helpers. Use this when:

- Testing FO behavior that runs to completion without human interaction (single-entity mode, guardrails, dispatch patterns)
- You need structured log parsing (tool calls, agent dispatches, text output)
- The test should support both `--runtime claude` and `--runtime codex`

Key classes and functions:

- **`TestRunner`** — Test framework with pass/fail counters, temp directory management, and results summary. Automatically cleans up temp dirs on success and preserves them on failure.
- **`LogParser`** — Parses `--output-format stream-json` JSONL logs from `claude -p`. Provides `tool_calls()`, `agent_calls()`, `fo_texts()`, `assistant_messages()`.
- **`CodexLogParser`** — Parses mixed JSON/text output from `codex exec --json`. Provides `full_text()`, `spawn_count()`, `completed_agent_messages()`.
- **`create_test_project(runner)`** — Creates a temp git repo with an initial empty commit.
- **`setup_fixture(runner, fixture_name, pipeline_dir)`** — Copies a fixture from `tests/fixtures/` into the test project.
- **`install_agents(runner, include_ensign=False)`** — Copies agent wrappers (`.claude/agents/`) into the test project.
- **`run_first_officer(runner, prompt, ...)`** — Runs `claude -p` with `--plugin-dir`, `--agent`, `--permission-mode bypassPermissions`, `--verbose`, `--output-format stream-json`. Returns exit code.
- **`run_codex_first_officer(runner, workflow_dir, ...)`** — Runs `codex exec` with skill home setup. Returns exit code.
- **`assembled_agent_content(runner, agent_name)`** — Reads the agent entry point and all referenced files, returning the full behavioral contract as a string. Useful for static content checks.
- **`git_add_commit(project_dir, message)`** — Stage all and commit.
- **`extract_stats(log_path, phase_name, output_dir)`** — Extract token usage, wallclock time, and model delegation from a JSONL log.

### `test_lib_interactive.py` — PTY Interactive Harness

Provides `InteractiveSession` for driving live Claude Code TUI sessions. Use this when:

- Testing multi-turn interactive behavior (skill invocations, team interactions, captain-agent communication)
- The behavior under test requires an interactive session (not reproducible via `-p`)
- You need to send multiple messages and check responses sequentially

Key class:

- **`InteractiveSession`** — Spawns `claude` in a PTY, waits for the prompt, sends messages, and checks output for patterns.
  - `start(ready_timeout)` — Start session and wait for the `❯` prompt.
  - `send(message)` — Type a message and press Enter.
  - `send_key(key_name)` — Send a special key sequence (e.g., `shift-down`, `ctrl-c`).
  - `wait_for(pattern, timeout, min_matches)` — Wait for a regex pattern in output after the last `send()`. Default `min_matches=2` to skip the input echo.
  - `stop()` — Send `/exit` and kill the process.
  - `get_clean_output()` — Return output with ANSI escapes stripped.
  - `get_subagent_logs(project_dir)` — Find subagent JSONL logs for the session.

**Trust dialog handling:** When `cwd` points to an untrusted directory, Claude Code shows a trust dialog that blocks the PTY. Use `start_with_trust_handling()` (see `test_single_entity_mode.py`) to detect and dismiss the dialog before waiting for the prompt.

## Standard CLI Flags

All E2E tests read their runtime configuration from pytest CLI options registered in `tests/conftest.py`:

| Flag | Default | Description |
|------|---------|-------------|
| `--runtime` | `claude` | Which runtime to test (`claude` or `codex`). |
| `--model` | `haiku` | Model for the test run. |
| `--effort` | `low` | Effort level (non-interactive tests). |
| `--budget` | `None` | Max budget in USD (optional). |
| `--team-mode` | `auto` | `auto` / `teams` / `bare`. Filters tests pinned to `teams_mode` or `bare_mode`. `auto` reads `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` (`"1"` or `"true"` → teams, else bare). |

Tests receive these values via the matching `runtime`, `model`, `effort`, `budget` fixtures. A `test_project` fixture yields a `TestRunner` with a tmpdir + git init; a `fo_run` factory fixture exposes a runtime-aware wrapper around `run_first_officer` / `run_codex_first_officer`.

Tier membership is declared with pytest markers:

- `@pytest.mark.live_claude` — spawns a live Claude runtime (pipe or PTY).
- `@pytest.mark.live_codex` — spawns a live Codex runtime.
- `@pytest.mark.serial` — must run serially (PTY, stubbed git/gh, or explicit sequencing). Parallel tests carry no `serial` marker.
- `@pytest.mark.teams_mode` — test requires `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`. Auto-skipped when running with `--team-mode=bare`.
- `@pytest.mark.bare_mode` — test requires `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` unset (or `"0"`). Auto-skipped when running with `--team-mode=teams`. Mutually exclusive with `teams_mode`; a test carrying both fails collection loudly.
- No marker = implicit static/unit — collected by `make test-static`.

Most live tests are mode-agnostic (no team-mode marker) and run under whichever mode the CI job or local invocation selects. Pin a test to a mode only when its invariant is mode-specific — e.g. `test_single_entity_team_skip` asserts the absence of a team, so it must run under `bare_mode`; `test_team_dispatch_sequencing` inspects `TeamCreate`/`TeamDelete` ordering, so it must run under `teams_mode`.

## When to Use Which Harness

### Static content checks (`assembled_agent_content`, `file_contains`)

Use for verifying that reference files, agent contracts, or assembled prompts contain required text patterns. These are fast, free, and should be the first line of defense.

Examples: `test_agent_content.py`, `test_codex_packaged_agent_ids.py`, static portions of `test_reuse_dispatch.py`.

### Non-interactive E2E (`test_lib.py` + `run_first_officer`)

Use for testing FO/ensign behavior that runs to completion in pipe mode. The test sets up a fixture, runs `claude -p` or `codex exec`, and parses the structured JSONL logs for tool calls, text output, and behavioral patterns.

Examples: `test_gate_guardrail.py`, `test_rejection_flow.py`, `test_single_entity_team_skip.py`.

### FO Prompt Discipline (Claude and Codex)

When a test exercises first-officer behavior on either runtime:

- Invoke only `spacedock:first-officer` (Claude) or `$first-officer` (Codex).
- Keep the invocation prompt minimal: identify the workflow target and entity scope only. The clean pattern is one line, e.g. `f"Process all tasks through the workflow at {abs_workflow}/ to completion."` (see `test_merge_hook_guardrail.py`) or `f"Process the entity `X` through the workflow at {abs_workflow}/."` (see `test_feedback_keepalive.py`).
- Do not add behavioral coaching in the test prompt for reuse, wait semantics, shutdown, rejection routing, keepalive, feedback-to, gate approvals, hook dialogue, or any other FO operating rule. If an FO behavior needs to change, encode that in the scaffolding under test: shared core, runtime adapter, skill references, or fixture/workflow structure.
- Prompt coaching masks FO-contract-loading bugs. If the clean prompt surfaces a new red that the hint-laden prompt hid, that is the signal — file it as a contract-loading finding, do not revert the cleanup.
- Exception: files that name the variable `tempt_prompt` are adversarial-input tests (e.g. `test_repo_edit_guardrail.py`, `test_scaffolding_guardrail.py`) — the tempting phrasing is the experimental variable under test. Keep as-is.
- Prefer shared runtime-switchable tests such as `test_rejection_flow.py --runtime codex` for generic workflow behavior. Use runtime-only E2E tests only for truly runtime-specific deltas that cannot be covered by the shared path.

### Strict Per-Stage Assertions (Watcher API)

Live E2E tests that drive the FO through multiple ensign dispatches should assert one behavioral shape, not a soup of heuristic signals. The `FOStreamWatcher` API supports this directly:

- `w.expect_dispatch_close(timeout_s, ensign_name=..., label=...) -> DispatchRecord` — block until the next ensign dispatch with a matching name-substring closes (by `SendMessage(to="team-lead", message="Done: ...")`). Raises `StepTimeout` on budget miss, `StepFailure` on early FO exit.
- `w.dispatch_records` — post-hoc list of `DispatchRecord(ensign_name, elapsed)` for every closed ensign dispatch.
- `DispatchBudget(soft_s=15, hard_s=60, shutdown_grace_s=10)` — per-dispatch soft (warn to fo-log + stdout) / hard (cooperative shutdown + kill on grace) budgets, threaded through `run_first_officer_streaming(dispatch_budget=...)`.

Anti-patterns to avoid when writing new live E2E:

- Path-A / Path-B disjunctions that accept either of two FO trajectories. If the test is about one contract, assert that contract; if both paths are worth covering, split into two tests.
- "SKIP: pipeline did not reach stage X within budget" softeners that convert a missing signal into a pass. If the stage is required by the contract, it's a hard assertion; if it isn't, it doesn't belong in the test.
- `milestones[...]` dicts where Phase-3 accepts any subset of keys via `or`-chains. Replace with per-dispatch `expect_dispatch_close` calls or an assertion on `w.dispatch_records`.
- OR-chain fallbacks like `entry_contains_text(e, r"SOMETHING") or tool_use_matches(e, "Bash", command="SOMETHING")` that accept narration in place of behavior — the tool_use branch is the contract, the text branch is a tautology bait.

### Teams-Mode Under `claude -p` (headless runtime quirks)

Agent Teams dispatches (`Agent(subagent_type="spacedock:ensign")` with `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`) behave differently under `claude -p` headless mode vs. an interactive TTY session. Both quirks are upstream ([anthropics/claude-code#26426](https://github.com/anthropics/claude-code/issues/26426) — "Agent Teams inbox polling doesn't work in non-interactive/SDK streaming mode"; closed as `not_planned`). Tests that exercise teammate keepalive must account for them.

**Quirk 1 — Inbox polling is a React UI hook and doesn't fire under `-p`.** In interactive mode, the `InboxPoller` React hook runs every 1000ms, reads `$HOME/.claude/teams/{team_name}/inboxes/team-lead.json`, and surfaces teammate messages as user-role content. Under `-p` the TUI never renders, so the hook never fires. Inbox JSON files accumulate on disk but are never delivered to the lead (FO). Symptoms: the FO has no way to observe ensign `Done:` messages, its `task_notification(completed)` system events never fire for `task_type=in_process_teammate`, and the FO hangs waiting for signals that are physically present on disk but never read.

**Quirk 2 — Text-only turn endings close the prompt cycle and reset context.** When an FO turn ends with `stop_reason=end_turn` and text-only content (no tool_use), `claude -p` emits `type=result subtype=success` and starts a new prompt cycle. The new cycle re-enters with fresh context, and models at lower effort (notably opus-low) may hallucinate reasons to act — e.g. emit `SendMessage(shutdown_request, reason="session ending")` to a still-working ensign based on a fabricated `Human:` prefix user message they generate to themselves. Symptoms: FO tears down ensigns mid-work ~15s after spawn; ensigns never see the shutdown_request (they aren't polling inbox mid-tool-loop); teams get stuck in `TeamDelete → "Cannot cleanup with 1 active member"` loops.

**Combined fix — inbox-poll keep-alive pattern.**

Tests that need the FO to actually observe teammate Done messages (keepalive contract tests, feedback-routing tests, anything with multi-stage teammate handoff) should inject an `--append-system-prompt` hint that:

1. Names a keep-alive sentinel file the harness controls (e.g. `{test_project_dir}/.fo-keepalive-done`).
2. Tells the FO: until that sentinel exists, every turn MUST end with a Bash tool_use rather than text. The Bash tool_use runs `scripts/fo_inbox_poll.py` to block-poll the team-lead inbox file for teammate `Done:` messages and print them to stdout.
3. The FO's Bash tool_result now contains the real inbox content, surfacing teammate messages into the FO's stream (filling the gap the missing `InboxPoller` left).
4. The turn ends with `stop_reason=tool_use` instead of `stop_reason=end_turn`, keeping `-p`'s prompt cycle open and preventing the fresh-context hallucination.

Canonical implementation in `tests/test_feedback_keepalive.py` (see `headless_hint`) and `scripts/fo_inbox_poll.py`. The `FOStreamWatcher` recognizes inbox-poll Bash tool_result content as a close anchor — specifically, entries shaped as `from: spacedock-ensign-{slug}-{stage}` + `text: Done: ...` will close any open dispatch whose `ensign_name` contains the matching stage substring.

**When to touch the sentinel.** Touch `{keepalive_done}` AFTER the test's contract assertions have fired. Do NOT treat the sentinel as a "workflow is done" signal the FO should obey strictly — the FO will correctly continue to process any in-flight terminal-stage work (cycle-2 impl fix, merge, archive) even after the sentinel appears. Wrap `expect_exit(...)` in `try/except` so post-contract FO activity is non-blocking; the `run_first_officer_streaming` context manager's `finally:` block kills the subprocess cleanly on test completion.

**Event-driven vs timeout-driven progression.** The stage-to-stage advance in a test like `test_feedback_keepalive` is driven by runtime-emitted signals (TeamCreate tool_use, dispatch-close via inbox-poll content or task_notification, SendMessage tool_use), not by timeouts. Timeouts are backstops for failure, not progression gates. The only timeout in the happy path is `fo_inbox_poll.py --timeout` (default 10s), which is a bounded-poll window inside each FO Bash invocation; a shorter/longer value affects the number of FO turns but not correctness.

**Investigating a failing live run.** The FO's `fo-log.jsonl` is the primary artifact. Look for:

- `system init` events — each marks a new `-p` prompt cycle. Multiple inits mean the session cycled; opus-low is most likely to misbehave at cycle-2+.
- `result subtype=success` with `stop_reason=end_turn` — cycle close. If this appears while the workflow isn't terminal, the keep-alive discipline failed.
- Assistant text entries with `"Human: ..."` prefix — smoking-gun for the fabricated-user-message hallucination.
- Bash tool_use with `command` containing `fo_inbox_poll.py` — inbox-poll attempts. The matching `tool_result` contains any delivered messages.
- `SendMessage` tool_use with `type: shutdown_request` addressed at an ensign — confirm this follows an actual completion signal, not a self-generated one.

The ensign's own side lives in `$HOME/.claude/projects/{cwd-slug}/{session_id}/subagents/agent-{hash}.jsonl` (inside the test's isolated HOME under `/var/folders/.../spacedock-clean-home-*/`). Useful for proving an ensign completed even when the FO never observed it.

There is no dedicated timeline-dump tool today; investigations typically use inline `python3 -c "import json; ..."` one-liners against the jsonl. A reusable `scripts/fo_log_timeline.py` would be a reasonable follow-up when investigation overhead becomes a bottleneck.

### Interactive PTY (`test_lib_interactive.py` + `InteractiveSession`)

Use only when the behavior under test requires an interactive session — multi-turn conversation, skill invocations, team member switching, or behavior that differs between interactive and pipe mode.

Examples: `test_interactive_poc.py`, `test_single_entity_mode.py`.

**Headless-CI behavior.** Both PTY tests carry `@pytest.mark.skipif(not sys.stdin.isatty(), reason="requires real TTY; CI runners are headless — see #155")`. They SKIP on GitHub Actions ubuntu-latest runners (no attached TTY) and RUN locally when invoked from a real terminal. See task #155 for the long-term plan (CI-detection vs test-split vs PTY-harness fix).

### Purely offline (no claude/codex)

Some tests validate infrastructure without needing a live session: `test_stats_extraction.py` (log parsing), `test_status_script.py` (status script behavior), `test_codex_packaged_agent_ids.py` (worker id resolution).

## Fixture Conventions

Fixtures live under `tests/fixtures/{fixture-name}/` and represent minimal self-contained workflows:

```
tests/fixtures/spike-no-gate/
  README.md        # Workflow README with frontmatter (stages, entity labels)
  status           # Executable status script (copied from the real one)
  test-entity.md   # Pre-seeded entity file with frontmatter
```

Each fixture directory contains:

- **`README.md`** — Workflow definition with `commissioned-by` frontmatter, stages, schema, and stage definitions.
- **`status`** — The status script (executable). Copied from `skills/commission/bin/status` or a test-specific variant.
- **Entity files** — One or more `.md` files with YAML frontmatter at appropriate starting stages.
- **Optional extras** — Source files, test directories, or `_mods/` for hook-specific tests.

Existing fixtures and their purposes:

| Fixture | Stages | Purpose |
|---------|--------|---------|
| `spike-no-gate` | backlog → work → done | Minimal no-gate pipeline for basic dispatch tests |
| `spike-gated` | backlog → work (gated) → done | Gate approval behavior |
| `gated-pipeline` | Similar to spike-gated | Gate guardrail testing |
| `multi-stage-pipeline` | Multiple stages | Multi-stage dispatch and reuse |
| `reuse-pipeline` | Stages with reuse conditions | Agent reuse vs fresh dispatch |
| `rejection-flow` | Includes validation with `feedback-to` | Rejection and feedback cycle testing |
| `rejection-flow-packaged-agent` | Same with `agent:` property | Packaged agent rejection flow |
| `merge-hook-pipeline` | Terminal stage with `_mods/` | Merge hook execution |
| `push-main-pipeline` | Terminal with PR mod | Push ordering and PR creation |
| `output-format-custom` / `output-format-default` | With/without `## Output Format` | Output format behavior |

## Running Tests

The whole suite runs under pytest. When invoking from inside a Claude Code session (including from dispatched team agents / ensigns), unset `CLAUDECODE` first — Claude Code refuses to launch as a subprocess when that variable is set. The `unset CLAUDECODE &&` prefix is the escape hatch and is already baked into the Makefile targets.

Stable repo-level entrypoints:

```bash
make test-static                                      # offline suite, no live marker
make test-live-claude                                 # all live_claude tests (serial first, parallel second)
make test-live-codex                                  # all live_codex tests
make test-e2e TEST=tests/test_gate_guardrail.py RUNTIME=codex   # single-file override
```

- `make test-static` runs `pytest tests/ --ignore=tests/fixtures -m "not live_claude and not live_codex"`. `tests/fixtures/` contains runnable fixture payloads and is excluded from collection.
- `make test-live-claude` runs the serial tier first (`-m "live_claude and serial" -x`), then the parallel tier (`-m "live_claude and not serial" -n $LIVE_CLAUDE_WORKERS`) regardless of the serial tier's outcome, and fails overall if either tier failed. `make test-live-codex` uses the same split with `LIVE_CODEX_WORKERS`; its cheap shared-runtime preflight is `test_gate_guardrail.py`, and the wrapper still tolerates any intentionally empty marker tier without masking real test failures.
- `make test-live-claude-opus` is the same shape with `--model opus --effort low` overrides.
- `make test-e2e` is the single-file override — pass `TEST=...` for the target file and `RUNTIME=claude|codex` for the runtime. This replaces the old `test-e2e-commission` target (use `TEST=tests/test_commission.py`).
- Do not invoke bare `pytest tests/` as the suite entrypoint unless you intentionally want pytest to recurse into `tests/fixtures/`.

Parallel worker count defaults are conservative because `claude -p` / `codex exec` share host runtime state:

```bash
LIVE_CLAUDE_WORKERS=4 make test-live-claude           # raise when stable
LIVE_CODEX_WORKERS=2 make test-live-codex
```

Direct pytest invocation for ad-hoc runs:

```bash
unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model haiku -v
unset CLAUDECODE && uv run pytest tests/ -m "live_claude and not serial" -n 2 --runtime claude
```

### Quick local smoke before pushing

Before pushing a PR — especially one that touches dispatch, commission, scaffolding, or the runtime adapters — run three cheap checks locally:

```bash
# 1) static discipline: ~6s, free
make test-static

# 2) cheapest live signal: ~60s, ~$0.02 haiku
unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model haiku -v

# 3) Codex shared-runtime pilot: ~60s, low-cost fail-fast preflight
unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime codex -v
```

`test_gate_guardrail.py` is the shared runtime pilot live test — smallest fixture, single gate transition, fails loudly on any FO-level regression (self-approval, wrong status, early archive). The Claude invocation is the cheapest Anthropic smoke; the Codex invocation is the cheap Codex preflight before burning the expensive parallel tier. If both pass locally, the CI live jobs usually pass too.

For a serial-tier fail-fast sweep before burning the expensive parallel tier:

```bash
# Claude bare-mode serial tier — matches claude-live-bare's first phase
unset CLAUDECODE CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS && \
  uv run pytest tests/ --ignore=tests/fixtures \
    -m "live_claude and serial" --runtime claude --team-mode=bare -x -v

# Codex serial tier — matches codex-live's first phase
unset CLAUDECODE && \
  uv run pytest tests/ --ignore=tests/fixtures \
    -m "live_codex and serial" --runtime codex -x -v
```

Stops on the first serial-tier failure in ~90s. The `-x` flag is the fail-fast lever; removing it runs all serial tests regardless. `test_gate_guardrail` carries both `live_claude` and `live_codex` + `serial`, so it runs first in both tiers — if it fails you see the regression in the cheapest possible window before the expensive parallel tests start.

### Known xfail / skip state

Some live tests are currently marked `xfail` or `skipif` on the `#148` branch — normal, not a regression of your change:

- **`@pytest.mark.xfail(reason="pending #154 ...")`** — applied to eight tests whose assertions target `agents/first-officer.md` for tokens that moved into `skills/first-officer/SKILL.md` and the reference files during the #085 skill-preload refactor. Surfaces as `XFAIL` in the pytest summary. Affected tests: `test_commission`, `test_agent_captain_interaction`, `test_output_format`, `test_reuse_dispatch`, `test_team_health_check`, `test_repo_edit_guardrail`, `test_dispatch_completion_signal`, `test_checklist_e2e`. Some show `XPASS` under bare mode because the drift only bites teams-mode paths — `strict=False` makes xpass silently OK. When #154 lands, these markers come off.
- **`@pytest.mark.skipif(not sys.stdin.isatty(), reason="requires real TTY; ... see #155")`** — applied to the two PTY-using tests (`test_interactive_poc_live`, `test_single_entity_mode`). Skips under headless CI, runs locally from a real terminal.
- **`@pytest.mark.skip(reason="pending #141 ...")`** — applied to `test_rejection_flow` (same-stage reviewer reuse during feedback cycles is correct behavior #141 will formalize; the test's current `ensign_count >= 3` assertion is bare-mode-biased).

## PR Runtime Live E2E

The expensive runtime-backed PR suite lives in `.github/workflows/runtime-live-e2e.yml`. It triggers on `pull_request_target` (types `opened`, `synchronize`, `reopened`) and still supports `workflow_dispatch` for targeted reruns. The workflow's first job is `static-offline` (running `make test-static`); all live-e2e jobs declare `needs: static-offline` so env-approval prompts only fire once the offline suite is green — this is the single source of the offline signal. The live jobs check out the PR merge ref (`refs/pull/<N>/merge` — the PR head merged into the target branch) with `persist-credentials: false`, so tests validate the PR's code as it would look after merging — catching base-drift when main has moved — and the `GITHUB_TOKEN` is not persisted into `.git/config` for later steps to abuse.

The security model relies on a single gate: the environment's required-reviewer approval. Because `pull_request_target` runs with base-branch context, repo secrets are available even for fork PRs — so the env-approval gate is the ONLY protection against malicious PR code running with secrets. Maintainers: review the PR head SHA's diff BEFORE approving the env deployment. The merge ref composes that diff with the current target; if the PR head diff looks malicious, refuse approval regardless of how benign the merge commit looks — secrets never reach the bad code. The workflow-level `permissions: { contents: read, pull-requests: read }` narrows the default `GITHUB_TOKEN` surface as an extra containment layer.

The default PR-triggered path is intentionally fixed. When a PR opens, GitHub creates four live jobs:

- `claude-live` (teams mode, haiku)
- `claude-live-bare` (bare mode, haiku)
- `claude-live-opus` (teams mode, opus)
- `codex-live`

Those jobs are not parameterized at approval time. The environment review UI only releases already-defined jobs; it does not collect `workflow_dispatch` inputs such as model selection or matrix expansion.

GitHub still presents the approval flow through the deployment review UI, even though the jobs set `deployment: false`. The current environment split is:

- `CI-E2E` for `claude-live` and `claude-live-bare` (same cost tier — both haiku — share the approval gate)
- `CI-E2E-OPUS` for `claude-live-opus`
- `CI-E2E-CODEX` for `codex-live`

Each environment has its own approval gate, so `claude-live-opus` can be released independently from `claude-live` — useful when a haiku run flakes in a way that may be model-specific, or when extra signal is wanted before merging a dispatch-prose change. `claude-live-bare` shares the `CI-E2E` gate with `claude-live` since both are haiku; releasing `CI-E2E` releases both jobs. Until an approved reviewer releases the relevant environment, that job stays pending and cannot access the environment-scoped API key.

### Operator flow

1. **Push a PR** — The workflow triggers automatically. The `claude-live`, `claude-live-bare`, `claude-live-opus`, and `codex-live` jobs appear on the PR status as pending review, with a "waiting for environment approval" banner in Actions. `make test-static` runs immediately without approval and reports back like any normal CI job.
2. **Captain decides the PR is ready for live validation.** Typical triggers: static CI is green, the PR description matches the diff, and the cost of burning live-runtime budget is justified.
3. **Approve the environment deployment.** Either through the GitHub UI (the PR's "pending review" banner → "Review deployments" → approve `CI-E2E`, `CI-E2E-OPUS`, and/or `CI-E2E-CODEX`), or via the CLI:
   ```bash
   gh api repos/{owner}/{repo}/actions/runs/{run_id}/pending_deployments \
     -f 'environment_ids[]={env_id}' -F state=approved -f comment='approved'
   ```
   The first officer can perform this step on the captain's explicit instruction, but does not self-approve. Approving one environment releases only that environment's job; approve both to release both.
4. **Jobs run.** Each job's summary shows run provenance (trigger source, PR number, workflow SHA, head SHA, fork/same-repo, approvers). Failures come back as a normal red CI signal on the PR.
5. **Re-test after a fix.** Push a new commit to the branch. The workflow re-triggers and the live jobs go back to pending review — approve again. For targeted reruns without a new commit, use `workflow_dispatch` (see below).

### Makefile targets

| Target | Model | When to use |
|--------|-------|-------------|
| `make test-live-claude` | haiku (default) | The primary CI signal. Runs on the `CI-E2E` environment via `runtime-live-e2e.yml`. Cheap, fast, catches most regressions. Teams-mode (`CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`). |
| `make test-live-claude-bare` | haiku (default) | Bare-mode variant of the haiku suite. Also runs on `CI-E2E`. Unsets `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` and passes `--team-mode=bare`, so tests pinned to `teams_mode` are auto-skipped and tests pinned to `bare_mode` run; mode-agnostic tests exercise the bare dispatch code path. |
| `make test-live-claude-opus` | opus with `--effort low` | Stronger-model variant of the haiku suite. Runs on the `CI-E2E-OPUS` environment via `runtime-live-e2e.yml`. Separately approvable from the haiku job, so it can be released when a haiku run flakes in a way that may be model-specific, or when a dispatch-prose change needs a second signal. Teams-mode. |
| `make test-live-codex` | codex default | Codex-runtime equivalent. Runs on the `CI-E2E-CODEX` environment. |
| `make test-live-codex-bare` | codex default | Bare-mode Codex variant. Local-only today; no dedicated CI job. File a follow-up if bare-codex signal becomes necessary. |

Open a PR and then approve the pending environment review to release the live runtime checks. For targeted reruns, API-driven launches, or future release-branch matrix runs, invoke the workflow manually from Actions or with:

```bash
gh workflow run runtime-live-e2e.yml --ref <branch>
```

Manual dispatch runs whatever branch the `--ref` points at (no PR association, no merge-with-target) — use it for on-main bisection or parameterized investigations. The `pr_number` input is gone; when you want to validate a specific PR's code as it would look after merging, push to its branch and let the `pull_request_target` auto-trigger fire — the merge ref resolves automatically, and both the PR head SHA and the resolved merge SHA are rendered in the job summary.

`workflow_dispatch` inputs are supplied when the run is created, not when the environment approval is granted. That makes manual dispatch the right entrypoint for any parameterized or matrix live runs.

### Bisection inputs (`runtime-live-e2e.yml`)

Four optional `workflow_dispatch` inputs narrow a run to a single variable. All default to empty; when unset, the job runs its normal `make test-live-*` target.

| Input | Purpose | Example |
|-------|---------|---------|
| `claude_version` | Pin Claude Code to a specific version (`stable`, `latest`, or `X.Y.Z`). When set, the install step runs `curl -fsSL https://claude.ai/install.sh \| bash -s -- "$CLAUDE_VERSION"`. | `2.1.110` |
| `test_selector` | Pytest nodeid or path to scope the run. When set, each job runs `uv run pytest "$TEST_SELECTOR"` instead of `make test-live-*`. Jobs whose markers don't match the selector collect zero tests and exit cleanly. | `tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips` |
| `effort_override` | Override the `--effort` flag. Applied to `claude-live`, `claude-live-bare`, `claude-live-opus`. | `high` |
| `model_override` | Override the `--model` flag for Claude jobs. Lets you bypass default-alias resolution (e.g., pin `claude-opus-4-6` even when the installed Claude Code resolves `opus` to `claude-opus-4-7`). No-op on `codex-live` — Codex uses its own model-selection path. | `claude-opus-4-6` |

Each Claude job's `GITHUB_STEP_SUMMARY` records the installed `claude --version` and the effective model used, so bisection runs are self-documenting.

### Bisection recipe

To narrow a Claude Code regression to a single version, dispatch the same test against pinned versions one at a time and compare results:

```bash
gh workflow run runtime-live-e2e.yml --ref main \
  -f claude_version=2.1.107 \
  -f test_selector=tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips \
  -f effort_override=low
```

Then rerun with `claude_version=2.1.110`, `2.1.111`, etc. The narrow window between a passing and failing version isolates the regression's introduction.

### Mitigation recipe

When a Claude Code version flips a model alias in a way that breaks a test (e.g., 2.1.111 flipped `--model opus` from `claude-opus-4-6` to `claude-opus-4-7`), pin the dated model explicitly to bypass the alias:

```bash
gh workflow run runtime-live-e2e.yml --ref main \
  -f claude_version=2.1.111 \
  -f test_selector=tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips \
  -f effort_override=low \
  -f model_override=claude-opus-4-6
```

The dated-model pin is future-proof against further default-alias flips. Prefer it over pinning `claude_version` when the Claude Code version itself is otherwise healthy.

Required environment secrets:

- `CI-E2E`: `ANTHROPIC_API_KEY` for `claude-live` and `claude-live-bare`
- `CI-E2E-OPUS`: `ANTHROPIC_API_KEY` for `claude-live-opus`
- `CI-E2E-CODEX`: `OPENAI_API_KEY` for `codex-live`

Each job fails immediately with a clear message if its required secret is missing after the environment is approved.

This workflow is expected to report current live-suite status honestly. If a runtime test fails, the corresponding job stays red; shipping the CI wiring does not imply the Claude and Codex suites are already fully green.

Operators should expect each job summary to show the run provenance explicitly:

- Trigger source
- PR number
- Base SHA (the workflow definition commit from base branch)
- PR head SHA (the diff under review — the commit maintainers inspect before approving the env deployment)
- Checkout ref (`refs/pull/<N>/merge` — the PR head merged into the target)
- Resolved merge SHA (the post-merge HEAD that actually ran, recorded by a post-checkout step)
- same-repo vs fork status
- approval/reviewer context

The live workflow sets `KEEP_TEST_DIR=1` automatically and uploads each job's preserved temp dirs as GitHub Actions artifacts:

- `runtime-live-e2e-claude-live`
- `runtime-live-e2e-claude-live-bare`
- `runtime-live-e2e-claude-live-opus`
- `runtime-live-e2e-codex-live`

For local debugging, set `KEEP_TEST_DIR=1` to preserve temp directories after test runs. Set `SPACEDOCK_TEST_TMP_ROOT=/path/to/root` to force `TestRunner` to create preserved dirs under a predictable parent directory.

## Test Hygiene Follow-ups

These tests carry patterns that violate the FO Prompt Discipline or Strict Per-Stage Assertions sections above. Filed during #203 cycle-7 but deferred so that cycle could stay focused on the opus-4-7 green-main failure inventory. Each row is a candidate for a focused cleanup follow-up (likely a new task entity per row, or a single umbrella entity with three items).

### Tier A — structurally similar to the pre-cycle-7 `test_feedback_keepalive`

| Test | Shape | Suggested rewrite |
|---|---|---|
| `test_rejection_flow.py` | Runtime-branched (claude vs codex two different assertion blocks) with a `milestones[...]` dict of ~8 keys consumed via OR-chains (`milestones["validation_wait"] or milestones["validation_completed"] or milestones["rejection_seen"] or milestones["follow_up_seen"]`) and disjunctions like `fo_exit == 0 or milestones["final_response"]`. | Strict per-dispatch: impl dispatch close → validation dispatch close → second impl dispatch close (rejection → fix), with `expect_dispatch_close(ensign_name="implementation" / "validation")`. Drop the milestones dict and OR-chains. Split claude and codex into two tests if the shared invariant doesn't fit both. |
| `test_reuse_dispatch.py` | Three-way branches on dispatch counts (`len(implementation_dispatches) == 0` → pass via reuse, `>= 1` → info-not-fail, etc.) — ~27 possible pass paths for what should be two assertions: "reuse conditions match → SendMessage", "`fresh: true` → Agent()". | Assert exactly one SendMessage reuse event on analysis→implementation transition, assert exactly one fresh Agent() dispatch at validation. Drop the count-threshold branches. |
| `test_rebase_branch_before_push.py` | 242-line body with nanny prompt (`"say 'yes, go ahead' when asked"`), pre-streaming watcher shape. Contains inherited soft fallbacks. | Strip the nanny prompt hints (the hook dialogue belongs in the merge-hook contract, not the test prompt). Migrate to `expect_dispatch_close` for any ensign dispatches + `expect` for the `git push origin main` Bash signal. |

### Tier B — moderate softness

| Test | Issue |
|---|---|
| `test_merge_hook_guardrail.py` | Uses hardcoded `timeout_s=300` on `expect_exit` and `expect`. Would benefit from per-dispatch budgets once the #203 instrumentation lands on main. Fixture variants (`hook_expected=True`/`False`) are legitimate and should stay. |
| `test_gate_guardrail.py` | Has `"SKIP: first officer gate report not found (ensign may not have completed before budget cap)"` — classic "didn't see what I wanted → pass" softener. Either the gate report is contractually required (then it's an assertion) or it isn't (then drop the check). |
| `test_team_fail_early_live.py` | Mixes probe-gated skips with test content. Consider splitting probe/skip logic into a fixture. |

### Background and motivation

The pattern being removed: live E2E tests that accumulate heuristic "milestones" (dispatch counts, text matches, file-state flags) and accept any subset via OR-chains, so the test passes as long as the FO does *something vaguely close to* the intended behavior. The failure mode is that the test flakes on any adjacent regression — and when it does flake, the signal is `StepTimeout` after 300s with no indication of which dispatch was slow or which signal was missing.

The fix: pick one FO trajectory per test, assert each dispatch closes under its per-stage budget via `expect_dispatch_close`, let `w.dispatch_records` carry the post-hoc evidence. When a test needs to cover two trajectories (e.g. Path-A fresh dispatch vs Path-B in-place processing), split into two tests — one contract per test.

## File Requirements

Every test file must:

1. Have two `# ABOUTME:` comment lines at the top explaining what the test does.
2. Expose one or more `test_*` functions collected by pytest.
3. Carry the appropriate tier marker(s) — `@pytest.mark.live_claude`, `@pytest.mark.live_codex`, and/or `@pytest.mark.serial` — or no marker for static/unit tests. Optionally pin to a team mode with `@pytest.mark.teams_mode` or `@pytest.mark.bare_mode` when the test's invariant is mode-specific; leave both off to stay mode-agnostic.
4. Use pytest assertions for pass/fail. Tests that drive a `TestRunner` for log parsing should call `t.finish()` at the end; `finish()` raises `AssertionError` if any `t.check` calls failed.

Example skeleton:

```python
# ABOUTME: E2E test for feature X in the first-officer template.
# ABOUTME: Verifies behavior Y using fixture Z.

import pytest


@pytest.mark.live_claude
def test_feature_x(test_project, model, effort):
    t = test_project
    # ... setup_fixture, install_agents, run_first_officer, etc.
    t.finish()
```
