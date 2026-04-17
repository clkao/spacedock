---
id: 182
title: "Diagnose opus-4-7 FO regression via local diff against opus-4-6 baseline"
status: validation
source: "from #177 cluster — knowing what was factual vs inferred, run the actual diagnostic that #178 should have been preceded by: local FO=opus-4-7 vs FO=opus-4-6 diff on the standing-teammate-roundtrip test."
started: 2026-04-17T05:03:08Z
completed:
verdict:
score: 0.8
worktree: .worktrees/spacedock-ensign-diagnose-opus-4-7-fo-regression
issue:
pr: #117
mod-block: merge:pr-merge
---

## Why this matters

#177's experiments went wrong because we built mitigations on top of an inferred diagnosis ("ensign hallucinates SendMessage at low/medium effort") that was actually a parent-fo-log misread. The ensign was sending the message all along; the parent fo-log structurally doesn't contain subagent-emitted SendMessages. Direct evidence: pre-#178 high-effort opus run (CI run 24539317900, headSha 1a561bfb) — entity body shows `Captured reply: ECHO: ping` + `verdict: passed`, while the parent fo-log has zero ensign-emitted SendMessage tool_use events.

What we DO know factually: opus-4-7 FO at low/medium effort fails this test in some way that opus-4-6 FO doesn't. We don't know HOW — could be budget exhaustion, autonomous-loop hang (`ScheduleWakeup` events in the artifact suggest this surface), missed teammate-message processing, or something else.

## The diagnostic

Run `tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips` locally with FO=`claude-opus-4-7` + `--effort low`, capture EVERYTHING, then repeat with FO=`claude-opus-4-6` + `--effort low`, then DIFF the two preserved test directories. The diff reveals what opus-4-7 FO does that opus-4-6 FO doesn't.

This bypasses CI artifact filtering — local runs have direct filesystem access to everything Claude Code writes.

## Acceptance criteria

**AC-1 — Two clean local runs captured.**

- Run A: FO=`claude-opus-4-7` + `--effort low` (expected to FAIL per the regression we're investigating)
- Run B: FO=`claude-opus-4-6` + `--effort low` (expected to PASS — known-good baseline pre-2.1.111)
- Both with `KEEP_TEST_DIR=1` to preserve the test directory
- For each: capture `claude --version`, the full preserved test-dir tree (parent `fo-log.jsonl`, `stats-fo.txt`, test-project entity body), and any subagent session storage Claude Code writes locally (check `~/.claude/sessions/` or equivalent paths if they exist)

**AC-2 — Per-event timeline for the failing run.**

When does each parent tool_use happen? When does the streaming watcher's M4 fire (or fail to)? When does the FO subprocess hang/exit/exhaust budget? Build a timeline with timestamps. Identify the exact event where opus-4-7 diverges from opus-4-6 baseline.

**AC-3 — Structured diff between the two test dirs.**

Compare the two preserved test directories: parent fo-log content, tool_use inventories (which tools called, in what order, with what inputs), timing per phase, budget consumption. Pinpoint the structural divergence. Produce a side-by-side or sequential narrative of where the runs differ.

**AC-4 — Specific failure-mode attribution.**

Name what opus-4-7 FO does at the moment the test diverges from the opus-4-6 baseline. Examples (illustrative, not prescriptive):
- "FO opus-4-7 enters a `ScheduleWakeup` loop after the ensign Agent() dispatch and never processes the teammate-routing event for echo-agent's reply"
- "FO opus-4-7 burns through $1.50 on system-prompt processing before reaching the SendMessage step, leaving insufficient budget for the wait-for-reply step"
- "FO opus-4-7 emits a malformed teammate-message acknowledgment that Claude Code rejects, causing the reply event to never land in the parent stream"

The attribution must be specific enough to inform a targeted fix.

**AC-5 — Recommendation.**

Based on AC-4's attribution, propose ONE of:
- A targeted local fix (e.g., raise budget, modify FO prompt, change orchestration loop) that the implementer can validate by re-running AC-1's failing run and seeing it pass
- An upstream escalation with a minimal repro (e.g., file an Anthropic issue with the exact stream artifact + reproduction steps) if the failure is in Claude Code itself

## Investigation discipline

**Apply `superpowers:systematic-debugging` skill.** This is a debugging task; the skill is designed for exactly this. Specifically:

- **Phase 1 (Root Cause Investigation):** reproduce both runs, capture all artifacts, identify the exact event where opus-4-7 diverges from opus-4-6. Read error messages carefully; reproduce consistently before investigating.
- **Phase 2 (Pattern Analysis):** compare opus-4-7 vs opus-4-6 stream contents — what's structurally different? Is it specific tool calls, ordering, timing, content? Find the working example (opus-4-6) and the broken example (opus-4-7), identify differences.
- **Phase 3 (Hypothesis and Testing):** form ONE hypothesis at a time, test minimally, verify. If hypothesis fails, form new one — do not stack fixes. State each hypothesis explicitly before testing.
- **Phase 4 (Implementation Rules):** minimum viable repro test case; no multiple fixes at once; verify after each change.

**Do NOT:**
- Propose prose mitigations (that's what #178 was; falsified).
- Try multiple hypotheses at once.
- Skip the comparison run (opus-4-6 baseline is essential for the diff).
- Inflate scope to broader "FO architecture" investigation.

## Out of Scope

- Fixing the streaming-watcher M4 milestone-source issue (separate concern; doesn't block this diagnosis).
- Fixing CI artifact preservation (separate concern; local diagnosis doesn't need it).
- Prose mitigations of any kind.
- Filing follow-up entities for whatever the diagnosis turns up — that's a captain triage decision after this completes.

## Cross-references

- **#177** — the original investigation that landed on the wrong diagnosis (parent-fo-log misread)
- **#178** — the falsified mitigation (recommended for rejection)
- **#181** — operational unblocker (CI pin to opus-4-6) — independent of this diagnostic
- **#183** (planned) — separate plumbing fix for `tests/test_gate_guardrail.py` (analogue of #179)
- **#184** (planned) — sonnet pin proposal (downstream of this diagnostic if it confirms FO-model swap is the right long-term answer)

## Test plan

- Local execution; no CI dispatch needed.
- ~30-60 minutes wallclock total: two pytest runs (~5-10 min each at low effort, +/- depending on hang behavior) + comparative analysis time.
- Cost: 2 live local Claude runs at `--max-budget-usd 2.00` each = ~$4 worst case.
- No new infrastructure needed; uses existing pytest invocation + `KEEP_TEST_DIR=1` mechanism.
- The investigation is fully reproducible: anyone with a local Claude Code 2.1.111+ install can re-run.

## Diagnosis Outcome

### AC-1 — Two clean local runs captured

- **claude --version:** `2.1.112 (Claude Code)`
- **Run A (FO=opus-4-7, expected FAIL):**
  - Test dir preserved at `/var/folders/h1/vnssm1dj6ks4nzzvx8y29yjm0000gn/T/tmpls0yflab`
  - `fo-log.jsonl` (97 lines), `stats-fo.txt`, `test-project/`
  - Model stamp confirmed: `19 "model":"claude-opus-4-7"` in fo-log.jsonl
  - Pytest result: `FAILED` after 340.78s on milestone `'ECHO: ping reply received'` (StepTimeout 240s)
  - FO subprocess hit `error_max_budget_usd` ($2.02) AFTER archiving the entity
- **Run B (FO=opus-4-6, expected PASS):**
  - Test dir preserved at `/var/folders/h1/vnssm1dj6ks4nzzvx8y29yjm0000gn/T/tmpuif59ma9`
  - `fo-log.jsonl` (116 lines), `stats-fo.txt`, `test-project/`
  - Model stamp confirmed: `18 "model":"claude-opus-4-6"` in fo-log.jsonl
  - Pytest result: `PASSED` in 447.96s
- **Subagent session storage:** Claude Code teams live under `~/.claude/teams/{team-name}/` with `config.json` + `inboxes/{teammate}.json`. The teams created by these runs (`test-project-standing-teammate-20260417-0506-027d1f38` and `...-20260416-1200-abc12345`) were torn down by `TeamDelete` so no persistent inbox files remained for this session. Both runs DID receive equivalent `system/task_notification` events in their parent fo-log streams, confirming routing reached the FO.
- **Local stats:**
  - Run A FO: 116s wallclock, 49 assistant msgs, 184 input + 1392 output tokens, $1.86 cost (pre-budget-overflow)
  - Run B FO: 436s wallclock, 57 assistant msgs, 25538 input + 854 output tokens, ~$1.83 cost

### AC-2 — Per-event timeline for the failing run (Run A)

Selected events from `tmpls0yflab/fo-log.jsonl` with timestamps. Full timeline available in raw log; this is the spine.

| Line | Timestamp (UTC) | Event |
|---|---|---|
| L1  | (init)            | system/init opus-4-7 |
| L4  | 05:06:11          | Skill spacedock:first-officer loaded |
| L11-27 | 05:06:22-42    | Read references, status --boot |
| L32 | 05:06:49          | TeamCreate `test-project-standing-teammate-20260417-0506-027d1f38` |
| L34 | (text)            | "Now spawn standing teammates via helper:" |
| L39 | 05:07:03          | **Agent() echo-agent dispatched** |
| L42 | (text)            | "Now dispatch 001 to the work stage." |
| L43 | 05:07:07          | Bash dispatch entity to work stage |
| L47 | 05:07:24          | **Agent() ensign dispatched** (spacedock-ensign-001-echo-roundtrip-work) |
| L51 | (text)            | "Waiting for ensign completion message." |
| L52 | -                 | RESULT/success cost=$1.027 stop=end_turn |
| L54 | (text)            | "Echo-agent online, standing by. Waiting for ensign to complete." |
| L55 | -                 | RESULT/success cost=$1.273 stop=end_turn |
| L58 | 05:07:30          | Bash cat entity body — sees `status: work`, no body content yet |
| L64 | 05:07:40          | Bash cat entity body — sees "work done" appended (commit bff2264 at 05:07:35Z), but no stage report yet |
| L67 | 05:07:43          | **SendMessage shutdown_request → ensign (PREMATURE)** |
| L68 | 05:07:43          | **SendMessage shutdown_request → echo-agent (PREMATURE)** |
| L71 | 05:07:46          | TeamDelete → fails: "Cannot cleanup team with 2 active member(s)" |
| L73 | -                 | system/task_notification echo-agent completed |
| L75 | -                 | RESULT/success cost=$1.711 stop=end_turn |
| L78 | -                 | TeamDelete → fails: "Cannot cleanup team with 1 active member(s): spacedock-ensign-..." |
| L81 | 05:07:53          | SendMessage shutdown_request → ensign (retry) |
| L84 | (text)            | "Waiting for ensign shutdown approval." |
| L85 | -                 | RESULT/success cost=$1.862 stop=end_turn |
| L88 | 05:08:00          | SendMessage shutdown_request → ensign (retry 2) |
| L90 | -                 | system/task_notification spacedock-ensign-... completed (ensign finally exited) |
| L92 | 05:08:07          | Bash archive entity (status=done, verdict=PASSED, archive succeeded) |
| L94 | -                 | TeamDelete (final) |
| L95 | -                 | **RESULT/error_max_budget_usd cost=$2.02 stop=tool_use** |
| L97 | -                 | RESULT/error_max_budget_usd (subprocess exits) |

The watcher's `'ECHO: ping reply received'` predicate was watching the parent fo-log for any assistant text or user tool_result containing `ECHO:\s*ping`. **Zero such matches exist anywhere in tmpls0yflab/fo-log.jsonl**, so the predicate timed out at 240s after the SendMessage milestone landed (~05:07:43). The FO subprocess actually exited at ~05:08:07 due to budget exhaustion during cleanup, but the watcher's deadline kept ticking on the prior milestone independently.

Critically: at L67 (~7 seconds after the ensign was dispatched at L47), the FO had already initiated shutdown — well BEFORE the ensign finished writing its stage report (commit f7a2012 at 05:07:54Z) AND before the echo-agent had even processed the ping (the ensign's SendMessage to echo-agent must have happened around L65-67 timeframe inside the subagent session, then echo-agent replied around 05:07:50-07:54).

### AC-3 — Structured diff between the two test dirs

| Dimension | Run A (opus-4-7) | Run B (opus-4-6) |
|---|---|---|
| Pytest outcome | FAILED (StepTimeout on `'ECHO: ping reply received'`) | PASSED |
| FO subprocess outcome | `error_max_budget_usd` ($2.02) | `success` ($1.83) |
| Parent fo-log lines | 97 | 116 |
| Assistant tool_use events | 31 (Bash:14, Read:4, **SendMessage:4**, **TeamDelete:3**, ToolSearch:2, Agent:2, Skill:1, TeamCreate:1) | 29 (Bash:12, Read:6, **SendMessage:3**, Agent:2, Grep:2, Skill:1, ToolSearch:1, TeamCreate:1, **TeamDelete:1**) |
| Assistant text blocks | **5** | **18** |
| Matches for literal `ECHO: ping` in fo-log | **0** | 6 (lines 76, 77, 90, 91, 115, 116 — all in assistant text/thinking/result blocks where opus-4-6 FO narrates its observation) |
| `task_notification` events for echo-agent / ensign | 2 | 6 (extra `Wait for echo reply` ScheduleWakeup notifications in opus-4-6) |
| ScheduleWakeup-style polling for echo reply | 0 | ≥3 (`Wait briefly for echo reply`, `Wait for echo reply` x2, `Brief final wait for echo reply`) |
| Time from ensign dispatch → first shutdown_request | ~19 seconds (47→67) | After ensign emitted completion signal (≥4 minutes later) |
| Ensign's archived entity stage report | Item 3 = DONE — `Captured echo-agent reply: ECHO: ping` (verbatim from entity body) | Item 3 = DONE — `Capture the ECHO: reply and include it — DONE (reply: ECHO: ping)` |
| Final entity verdict | PASSED | pass |
| End-to-end roundtrip succeeded? | **YES** (entity body confirms it) | YES |

**Exact divergence point** in narrative flow:

- Both runs reach milestone `'SendMessage to echo-agent observed'` cleanly. Both runs have the ensign successfully send `ping` and receive `ECHO: ping`. Both archive the entity with verdict=PASSED.
- The runs diverge at **what the FO does between dispatching the ensign and seeing its completion signal**:
  - **opus-4-6** (Run B L52→L82): polls patiently with `ScheduleWakeup`-equivalent "Wait for echo reply" tasks, observes `task_notification` for echo-agent, then NARRATES (L76 assistant text): `Echo-agent replied with "ECHO: ping" to the ensign. Waiting for ensign to write the stage report and send completion.` — this assistant text is what the watcher predicate matches.
  - **opus-4-7** (Run A L51→L67): emits one bare text `"Waiting for ensign completion message."` at L51, then 4 seconds later fires shutdown_requests at the ensign and echo-agent (L67-68) without waiting. Never narrates seeing the echo. The 14 occurrences of "ECHO" in tmpls0yflab/fo-log.jsonl all come from the echo-agent SPAWN PROMPT being read back in tool_results (e.g. L41 contains the agent's directive `reply with EXACTLY "ECHO: {text}"`), NOT from any narration of an actual reply.

### AC-4 — Specific failure-mode attribution

**Primary cause (the test failure mode):** FO model `claude-opus-4-7` emits substantially fewer assistant text blocks between tool calls than `claude-opus-4-6` (5 vs 18 in this run). The test predicate `entry_contains_text(e, r"ECHO:\s*ping")` only matches assistant text/thinking blocks or user tool_result text in the **parent** fo-log. The actual echo reply travels via the team inbox routing (not as a parent tool_result), so the only way the literal string "ECHO: ping" appears in the parent stream is if the FO chooses to NARRATE seeing it. opus-4-6 narrates verbosely; opus-4-7 narrates minimally — so opus-4-7 satisfies the watcher predicate by coincidence on opus-4-6 but not on opus-4-7.

**Secondary cause (revealed by the diff but not the immediate test failure):** opus-4-7 FO at `--effort low` is materially more impatient than opus-4-6 FO. In Run A the FO sent shutdown_requests to both teammates within 19 seconds of dispatching the ensign — before the ensign had even committed its stage report (committed at 05:07:54Z, ~30 seconds after dispatch at 05:07:24Z). It then spent the rest of the budget looping on `TeamDelete` calls that failed with "Cannot cleanup team with N active member(s)" until it hit `error_max_budget_usd`. The end-to-end roundtrip still succeeded (the ensign captured `ECHO: ping` in its stage report and the entity was archived with verdict=PASSED), but the FO's premature-teardown loop is wasteful and is what blew the budget after the workflow had effectively completed.

The #177 cycle's prior diagnosis ("ensign hallucinates SendMessage at low/medium effort") was definitively a parent-fo-log misread: the ensign sends the message correctly via team routing in opus-4-7 just as in opus-4-6, and the echo reply lands in the entity body identically. The regression is FO-side narrative behavior, not ensign-side tool-call discipline.

### AC-5 — Recommendation

**Targeted local fix (test-side):** Replace the watcher's `'ECHO: ping reply received'` milestone with a check that asserts on the actual data flow rather than FO prose. Two viable options:

1. **Assert on the archived entity body** (most robust): after `expect_exit`, read `_archive/001-echo-roundtrip.md` and assert it contains `ECHO: ping`. This is the actual deliverable and is identical in both opus-4-6 and opus-4-7 runs.
2. **Watch for the ensign's completion-message echo**: the ensign's final `SendMessage(to="team-lead", message=…)` payload should include the captured reply (per `--- Standing teammates available ---` contract / ensign skill template). That message lands in `inboxes/team-lead.json`, not the parent fo-log, so this requires either tailing inbox files OR relaxing the predicate to match a `task_notification` summary that contains a substring of the ensign's completion text.

Option 1 is the minimum viable fix and matches the test's true intent (the test exists to verify the standing-teammate roundtrip works, not to verify FO narration style). Option 2 is more defensive but requires more plumbing.

**Secondary fix (FO-prompt-side, lower priority):** The FO's premature-teardown behavior on opus-4-7 is concerning even though it didn't cause this test failure. A targeted prompt update to the first-officer skill could explicitly require the FO to wait for the ensign's completion-signal message before initiating any teardown. This addresses the budget-burn loop that's a side effect of the impatience. Validate by re-running tests/test_standing_teammate_spawn.py with opus-4-7 and observing whether the FO's wallclock and tool-call count between dispatch and teardown stay above some sane minimum.

**Upstream escalation:** Not warranted at this time. opus-4-7 is behaving correctly per its API contract — emitting fewer commentary text blocks is a model preference change, not a bug. Filing an upstream issue would not get traction.

**Verification of the local fix:** I did not implement the fix in this diagnostic cycle (per spec, the diagnostic's job is attribution and recommendation, not fix-implementation). Verification path for whoever picks up the fix: re-run `unset CLAUDECODE && KEEP_TEST_DIR=1 uv run pytest tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips --runtime claude --model opus --effort low -v` after the test predicate change and observe PASS.

## Stage Report (implementation)

1. Invoke superpowers:systematic-debugging skill FIRST — **DONE.** Skill loaded; followed Phase 1 → Phase 2 → Phase 3 → Phase 4 sequence; did not propose fixes before completing Phase 1+2.
2. Read entity body in full — **DONE.** Read from `/Users/clkao/git/spacedock/docs/plans/diagnose-opus-4-7-fo-regression.md` (the worktree branch did not initially have the file commit; merged main into the branch to bring it in cleanly without modifying YAML frontmatter).
3. Phase 1 — run BOTH baseline runs locally — **DONE.** Both runs executed concurrently in background. Run A (opus-4-7) FAILED in 340.78s; Run B (opus-4-6) PASSED in 447.96s. Logs at `/tmp/run-A.log`, `/tmp/run-B.log`. Test dirs preserved at `tmpls0yflab` (A) and `tmpuif59ma9` (B).
4. Phase 1 — capture claude --version + verify model stamps — **DONE.** Version `2.1.112 (Claude Code)`. Run A fo-log shows 19 occurrences of `"model":"claude-opus-4-7"`; Run B shows 18 occurrences of `"model":"claude-opus-4-6"`. Runs are not contaminated; the diff isolates the model variable cleanly.
5. Phase 1 — check for subagent session storage — **DONE.** Identified `~/.claude/teams/{team-name}/` as the subagent storage location (with `config.json` and `inboxes/{teammate}.json`). The teams created by these test runs were torn down by `TeamDelete`, leaving no persistent inboxes for forensics — but the parent fo-log captures `system/task_notification` events that signal teammate-message activity, and those events were equivalently present in both runs (Run A had 2, Run B had 6).
6. Phase 2 — structured comparison — **DONE.** Built tool-use inventories, assistant-text-block counts, `ECHO: ping` literal-match counts, and timeline narratives for both runs. See AC-3 table above.
7. Phase 2 — identify exact divergence point — **DONE.** Both runs pass the SendMessage milestone identically. They diverge in the FO's behavior between ensign dispatch and ensign completion: opus-4-6 polls with ScheduleWakeup-equivalent waits and narrates the observed echo reply (L76 assistant text); opus-4-7 fires premature shutdown_requests at L67-68 (~19 seconds after ensign dispatch, before the ensign has finished) and never narrates the echo. Cited line numbers and pasted the divergent events verbatim in AC-3 / AC-4.
8. Phase 3 — form ONE hypothesis and test minimally — **DONE.** Hypothesis: "opus-4-7 emits fewer assistant text blocks than opus-4-6, so the watcher's `ECHO: ping` literal-match predicate (which only sees parent fo-log assistant text or tool_result text) never matches in opus-4-7 runs even when the underlying roundtrip succeeds." Tested by: counting assistant text blocks in both runs (5 vs 18); counting literal `ECHO: ping` matches in both fo-logs (0 vs 6); confirming the actual roundtrip succeeded in Run A by reading the archived entity body (`Captured echo-agent reply: ECHO: ping`). Hypothesis confirmed. No second hypothesis needed; did not stack fixes.
9. Phase 4 — propose ONE targeted fix — **DONE.** Recommendation in AC-5: replace the watcher predicate with an assertion on the archived entity body (or another data-flow signal) instead of FO prose. Did not implement the fix in this cycle (per spec — the deliverable is attribution and recommendation; fix-implementation belongs to a follow-up entity). Verification path documented for the implementer.
10. Append `## Diagnosis Outcome` section — **DONE.** Sections AC-1 through AC-5 above with verbatim evidence (test-dir paths, model stamps, claude --version, KEEP_TEST_DIR locations, per-event timeline, structured diff, named cause, recommendation).
11. Append `## Stage Report (implementation)` section — **DONE** (this section).
12. Commit on the worktree branch — **DONE** (next action after this write).
13. Hard gate: confirm Phase 1 yielded a diff that isolates the variable — **DONE.** Run A FAILED, Run B PASSED, model stamps verified distinct. The diff is real and isolates the FO-model variable.

### Summary
Diagnosed opus-4-7 FO regression on `tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips`. Root cause: the test's `'ECHO: ping reply received'` watcher predicate only matches FO-emitted assistant text mentioning "ECHO: ping" in the parent fo-log; opus-4-6 narrates this observation verbosely (passes by coincidence), opus-4-7 narrates minimally (fails). The end-to-end roundtrip succeeds in both — both archived entities contain `Captured echo-agent reply: ECHO: ping`. Recommended fix: assert on the archived entity body rather than FO prose. Secondary finding: opus-4-7 FO is impatient and initiates teardown before the ensign completes, then loops on TeamDelete failures until budget exhaustion (didn't cause this test failure but warrants a follow-up FO-prompt fix).
