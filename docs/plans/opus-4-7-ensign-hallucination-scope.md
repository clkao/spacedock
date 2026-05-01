---
id: 177
title: "opus-4-7 ensign hallucination at low/medium effort — scope of impact across spacedock dispatches"
status: ideation
source: "2026-04-16 session — PR #107/#105 CI failures bisected to Claude Code 2.1.110→2.1.111 default-alias flip from claude-opus-4-6 to claude-opus-4-7. Live-CI evidence + fo-log.jsonl artifacts confirm the ensign subagent on opus-4-7 fabricates tool-call outcomes rather than issuing the tool calls."
started: 2026-04-17T01:23:21Z
completed:
verdict:
score: 0.85
worktree: 
issue:
pr:
---

## Problem Statement

Claude Code 2.1.111 flipped the default `--model opus` resolution from `claude-opus-4-6` to `claude-opus-4-7`. Under `opus-4-7` at `--effort low` or `--effort medium`, dispatched ensigns exhibit a specific hallucination pattern: they execute easy tool-call steps (file writes, commits) but skip harder steps (`SendMessage` to teammates, tool-mediated verification) and fabricate the outcome in their stage reports. The FO accepts the stage report at face value because it reads DONE markers without verifying evidence against the session stream.

This concern is not limited to the one test that exposed it. The ensign dispatch shape — checklist + stage report + visible teammate descriptions — is the standard template for every spacedock ensign dispatch. The hallucination is contextual (simple isolation reproducers do not trigger it), prompt-shape-dependent, and effort-gated: `opus-4-7` at `--effort high` or `--effort xhigh` does not exhibit the low/medium fabrication pattern, but exposes a different failure at those effort levels (see Evidence at high/xhigh effort). `opus-4-6` at any effort does not exhibit either pattern.

## Evidence

- **Bisection**: `Claude Code 2.1.107` and `2.1.110` resolve `--model opus` to `claude-opus-4-6`; `2.1.111` resolves it to `claude-opus-4-7`. Verified via `fo-log.jsonl` `assistant.message.model` stamps across CI artifact downloads.
- **Repro in CI**: the 2026-04-16 spot-check at 2.1.111 + opus/medium failed with `StepTimeout: Step 'SendMessage to echo-agent observed' did not match within 240s`. The ensign's on-disk stage report claims `ECHO: ping` was captured; the parent `fo-log.jsonl` tool-use inventory has zero `SendMessage` entries.
- **No-repro in isolation**: a minimal 3-step task (`Write` + `Bash cat` + report stdout) executed correctly on both `opus-4-6` and `opus-4-7` at `--effort low`. The simple case does not expose the bug.
- **Scope gap**: the ensign's prompt always includes the "Standing teammates available in your team" section listing the reply format per teammate. `opus-4-7` uses that visible description to compose plausible tool-call outcomes without issuing the tool calls.

## Impact across spacedock dispatches

The pattern affects every ensign dispatched via `claude-team build` when the captain session is on `opus-4-7` at `--effort low` or `--effort medium`:

- **Validation-stage ensigns**: may rubber-stamp hallucinated implementation reports. A PR could ship broken code that "passed validation" because the validator skipped re-running tests and fabricated a PASSED recommendation. This is the highest-risk surface.
- **Implementation-stage ensigns**: may claim tests passed without running them, fabricate stdout from shell commands, or skip Write/Edit operations while reporting DONE. Observable only if CI reruns the tests or the captain audits evidence.
- **Standing-teammate routing**: any ensign told to route to `comm-officer` or a future standing teammate may skip the `SendMessage` and write its own "polished" content. Output looks like the teammate produced it.
- **FO itself** on `opus-4-7`: may skip `status --set` calls, fabricate PR numbers, or claim merges that did not happen. Observable if the captain checks git state, but silent otherwise.
- **Feedback cycles**: reviewers may claim re-verification without re-running anything. The rejection → fix → accept pattern still looks clean.

## Not affected (as far as we know)

- **Bare mode** (`claude-live-bare` passed on `2.1.111`). Smaller tool surface without `SendMessage`; needs more testing before claiming safe.
- **Codex runtime**: different model-selection path; not affected by the Claude Code default-alias flip.
- **Haiku jobs**: haiku passed on `2.1.111` CI; does not exhibit the pattern.
- **Explicit `--model claude-opus-4-6` pinning**: bypasses `opus-4-7` entirely. Confirmed locally (test passed in 2m5s) and should be confirmed in CI once the #176 plumbing bug is fixed.

## Current mitigations and gaps

- **Streaming watcher** (#173, #175): catches hallucinations when a test explicitly asserts on a tool-call shape via `w.expect(tool_use_matches(...))`. Eight tests currently have this coverage. Real-world spacedock usage (production ensigns, not tests) has no such guard.
- **Dated-model pin** (#176): bypasses `opus-4-7` via `--model claude-opus-4-6` workflow input. Currently broken by a test-level plumbing bug (tests hardcode `--model opus` in `extra_args`).
- **Effort bump to `--effort high` or `--effort xhigh`**: tested in CI (runs `24539317900` / `24539318567` on `2.1.111` + `--model opus` + effort high/xhigh). The low/medium hallucination pattern does not appear — the ensign correctly emits the `SendMessage` to the standing teammate (milestone 4 passes). However, a different failure mode surfaces at both high and xhigh: milestone 5 (`ECHO: ping reply received`) times out at 240s. The reply from `echo-agent` (on sonnet) never appears in the parent `fo-log.jsonl` within the window, even though the FO proceeds to archive the entity as completed. Effort bump removes one regression and exposes another — not a full mitigation.

## Evidence at high/xhigh effort (2026-04-16 runs)

- **`24539317900` (opus/high)**: `test_standing_teammate_spawns_and_roundtrips` failed on both `claude-live` and `claude-live-opus`. Specific error: `StepTimeout: Step 'ECHO: ping reply received' did not match within 240s`. Parent `fo-log.jsonl` shows the ensign DID emit a `SendMessage` to `echo-agent` (milestone 4 passed), but `ECHO: ping` never lands in the stream.
- **`24539318567` (opus/xhigh)**: identical failure pattern. `ECHO: ping reply received` timeout at 240s, milestone 4 clean.

Possible causes (open for investigation):

- `echo-agent` reply is routed through a subagent stream not folded into the parent `fo-log.jsonl`, so the test's parent-stream-only assertion cannot observe it.
- `echo-agent` (sonnet) takes longer than 240s to respond on the `2.1.111` runner under teammate-message scheduling.
- `echo-agent` hallucinates its own reply internally but never emits a `SendMessage` back to the ensign or the FO — an echo-agent-side variant of the `opus-4-7` hallucination pattern, tested on sonnet.
- Claude Code `2.1.111`'s teammate-message fold-in into the parent stream has a behavior change that predates or accompanies the default-alias flip.

The FO treating the entity as complete despite the missing reply suggests the test's stream-visibility expectation and the runtime's actual stream-delivery shape have diverged somewhere in the `2.1.110` → `2.1.111` window.

## Open questions for ideation

- Should production use of spacedock with Claude Code 2.1.111+ default to `--model claude-opus-4-6` or require explicit model pinning?
- Should the FO add a post-ensign-completion verification step that cross-checks the stage report's DONE claims against tool-call evidence in the stream?
- Should the ensign prompt template change — e.g., drop the "Standing teammates available" section from dispatch prompts where the ensign does not need to route — to reduce the visible context that primes hallucination?
- Is an upstream Anthropic issue warranted? The `fo-log.jsonl` artifacts are a reasonable starting reproducer even without a minimal single-agent case.
- Does the pattern hit other model families (sonnet-4-6) at low effort, or is it specific to `opus-4-7`'s effort calibration?
- Is the high/xhigh-only `ECHO: ping reply received` timeout the same underlying `opus-4-7` behavior in a different guise (echo-agent-equivalent fabrication on sonnet), a separate test-harness fold-in issue, or a Claude Code `2.1.111` runtime regression? Needs direct inspection of the high-effort `fo-log.jsonl` artifacts and comparison against the `2.1.107` baseline.

## Out of Scope

- Fixing the behavior in Claude Code or the model itself. This task covers spacedock-side mitigations and user guidance.
- Full rewrite of the ensign dispatch template. Any template changes follow after ideation resolves which changes are warranted.
- Building a minimal single-agent reproducer. The 2026-04-16 session established that isolation does not cheaply expose the pattern; the `fo-log.jsonl` CI artifacts serve as the working reproducer for now.

## Cross-references

- #171 — `Agent(model=...)` teams-mode propagation. Distinct bug (Agent-level), same surface (ensign model inheritance). Footnote in #171 explains the distinction.
- #173 — streaming watcher; the only guard currently catching this in CI.
- #174 / #176 — CI bisection and mitigation plumbing.
- #175 — test migration expanding stream-based coverage to 6 more live tests.
- #178 — tool-call-discipline boilerplate (PR #113, branch `spacedock-ensign/tool-call-discipline`). #177 is the live experiment that decides whether #178 ships or whether we fall back to pinning `--model claude-opus-4-6`.
- A separate small task (not yet filed) will fix the `extra_args` plumbing bug so #176's `model_override` actually reaches `claude -p`. That unblocks the CI mitigation proof.

## Decision

This task is a **focused live experiment**, not a hallucination-mitigation thesis. The mitigation under test is #178 (the tool-call-discipline boilerplate already shipped on branch `spacedock-ensign/tool-call-discipline`, PR #113). #177's deliverable is a yes/no on whether that boilerplate makes `opus-4-7` viable at `--effort low` and `--effort medium` for the standing-teammate roundtrip case that originally exposed the regression.

Mechanics:

- Create a worktree stacked on top of `spacedock-ensign/tool-call-discipline` (NOT `main`), so the experiment runs against the #178 mitigation as it would actually ship.
- Drive three CI runs of the smallest fail-fast test we have: `tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips`, on Claude Code `2.1.111`, against the stacked worktree. Two are the variables under test (`--model opus` + `--effort low`, `--effort medium`); one is a negative control (`--model claude-opus-4-6` to prove the test still passes when the 4-7 alias is bypassed).
- Outcome maps cleanly to two paths:
  - **PASSED at both low and medium**: recommend shipping #178 and leave the default `opus` alias alone. Capture this as a debrief note that unblocks #178's merge.
  - **FAILED at either**: recommend pinning `--model claude-opus-4-6` in workflow defaults and developer docs. Cite #176 as the plumbing prerequisite and file (or note the need for) the small follow-up that fixes the `extra_args` plumbing so the pin actually reaches `claude -p` in CI.

Explicitly **not** part of this task (per Out of Scope, restated for the staff reviewer):

- No FO-side post-completion verification design. That is a larger, separate mitigation surface.
- No further changes to the dispatch-prompt template beyond what #178 already ships. We are testing #178's prose, not iterating on it.
- No work on the high/xhigh `ECHO: ping reply received` timeout. That failure mode is a different surface (likely either a parent-stream fold-in regression or an echo-agent-side issue) and warrants its own task once this experiment lands.

## Acceptance Criteria

Each AC has a specific verify command, a clear pass/fail line, and the evidence to capture.

**AC-1 — Live CI: `--model opus` + `--effort low` on stacked branch passes.**

- Verify: dispatch `runtime-live-e2e.yml` against the stacked worktree branch with `claude_version=2.1.111`, `test_selector=tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips`, `effort_override=low`. Job: `claude-live-opus`.
- Pass: job result `success`. The streaming watcher milestones for `SendMessage to echo-agent observed` and `ECHO: ping reply received` both fire within their per-step timeouts. Wallclock ≈ 2-3 minutes is the *pass* expectation (matching the `claude-opus-4-6` baseline established in #176); a fail-fast `StepTimeout` at 60-180s is not directly comparable to a pass wallclock — per-milestone times from the streaming watcher are the right granularity for fail attribution.
- Fail: any milestone times out, or job result `failure`. The streaming watcher's labeled `StepTimeout` identifies which milestone the boilerplate failed to discipline.
- Capture: run URL, the `claude-live-opus` job's `assistant.message.model` stamps from `fo-log.jsonl` (proves we actually ran on `opus-4-7`), wallclock.

**AC-2 — Live CI: `--model opus` + `--effort medium` on stacked branch passes.**

- Verify: same dispatch as AC-1 with `effort_override=medium`.
- Pass / Fail / Capture: identical shape to AC-1.

**AC-3 — Negative control: `--model claude-opus-4-6` on stacked branch passes.**

- Verify: same dispatch as AC-1 with `effort_override=low` and `model_override=claude-opus-4-6` (depends on the #176-follow-up `extra_args` plumbing fix; if that plumbing is still broken at experiment time, fall back to a local run with `--model claude-opus-4-6 --effort low` against the stacked worktree and capture the local wallclock + `fo-log.jsonl` model stamps as evidence). The local fallback MUST invoke `claude --version` matching the CI dispatch's `claude_version=2.1.111` — without that pin, a Claude Code version regression could mask as a model regression and contaminate the negative-control signal.
- Pass: job (or local run) result `success`. Test passes in the expected ~2-3 minute window. Model stamps in `fo-log.jsonl` show `claude-opus-4-6`.
- Fail: if this fails, the test itself is broken on the stacked branch and AC-1 / AC-2 results cannot be trusted. Stop the experiment and surface to the captain — the test must be fixed before the experiment can run.
- Capture: run URL (or local wallclock + log path), model stamps, wallclock, and for the local-fallback path the `claude --version` output proving 2.1.111.

**AC-4 — Recommendation deliverable matches the outcome.**

- If AC-1 and AC-2 both pass: write a debrief note to `docs/plans/opus-4-7-ensign-hallucination-scope.md` (Stage Report or a dedicated `## Outcome` section) recommending #178 ships as-is, citing the three run URLs and the wallclock numbers. The note explicitly unblocks #178's merge mod-block. The recommendation covers the standing-teammate roundtrip surface only; broader confidence across the five impact surfaces enumerated in #177's Impact section requires follow-up scoping and is out of scope for this experiment.
- If AC-1 or AC-2 fails (both fail, or either fails): write the same note recommending we pin `--model claude-opus-4-6` in workflow defaults and update `tests/README.md` (or equivalent developer-facing doc) to document the pin until the upstream Claude Code regression is resolved. Cite #176 as the plumbing dependency. File a small follow-up task (or note its need) covering the workflow-default change itself, since that change is mechanically separate from this experiment.
- Mixed outcome (AC-1 PASS / AC-2 FAIL, or AC-2 PASS / AC-1 FAIL): treat as the FAIL path above. Any low/medium failure on the standard surface is shipping risk, so the recommendation is to pin `--model claude-opus-4-6` rather than ship #178 with a known effort-level gap. Note the mixed outcome explicitly in the recommendation so the next iteration of #178 can target the failing effort level.
- Verify: the recommendation note exists in the entity body, references the captured run URLs, and states one of the two paths above unambiguously.
- Pass: a future reader can determine from the entity alone which path was taken and why.

## Test Plan

- **Cost in CI minutes**: ~3 runs × ~5 min wallclock (the streaming watcher fails fast at 60-180s if the regression appears; 5 min is a generous upper bound that includes runner spin-up). Total ≈ 15 CI minutes for the experiment, plus any retries.
- **Risk level**: low. No new code beyond what #178 ships. The streaming watcher (#173, #175) provides the observability needed to attribute pass/fail to the specific milestone, so a flaky failure is distinguishable from a real regression.
- **No new code is written by this task.** All implementation lives in #178; #177 only consumes it via the stacked worktree.
- **Dependencies**:
  - Implementation stage must branch from `spacedock-ensign/tool-call-discipline` (the #178 branch), not `main`.
  - AC-3 ideally depends on the `extra_args` plumbing fix mentioned in Cross-references. If that fix is not merged at experiment time, AC-3 falls back to a local run as documented above — the experiment is not blocked on that follow-up.
- **E2E tests**: yes, this entire task IS an E2E test. The unit tests for #178 already exist on the stacked branch (`test_claude_team_spawn_standing.py` extension); #177 does not add more.
- **Static checks**: none new. Sufficiency is established by the streaming watcher's labeled milestones — pass/fail attribution is structural, not log-archaeology.

## Implementation Notes

For the implementation stage, the following mechanics matter:

- **Worktree stack**: create the worktree from the #178 branch tip, e.g.

  ```
  git worktree add .worktrees/opus-4-7-experiment -b spacedock-ensign/opus-4-7-low-medium-experiment spacedock-ensign/tool-call-discipline
  ```

  This branches the experiment off the mitigation branch so CI dispatches against the experiment branch include #178's prose.

- **If #178's branch advances during the experiment**: rebase the experiment branch onto the new tip (`git rebase spacedock-ensign/tool-call-discipline` from inside the worktree), force-push the experiment branch (`--force-with-lease`, never plain `--force`), and re-dispatch the affected CI runs. Document the rebase in the stage report so the captured run URLs are unambiguous about which #178 commit they tested.

- **CI dispatch shape** (canonical form for AC-1):

  ```
  gh workflow run runtime-live-e2e.yml \
    --ref spacedock-ensign/opus-4-7-low-medium-experiment \
    -f claude_version=2.1.111 \
    -f test_selector=tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips \
    -f effort_override=low
  ```

  AC-2 swaps `effort_override=medium`. AC-3 adds `-f model_override=claude-opus-4-6` and depends on the plumbing fix; otherwise run locally inside the worktree with `make test-live-claude-opus` after editing the Makefile target's model flag (or invoke pytest directly).

- **Evidence the implementation stage MUST capture**:
  - Run URL for each CI dispatch (AC-1, AC-2, optionally AC-3).
  - Experiment branch SHA at dispatch time (`gh run view` exposes it; record inline to survive any `--force-with-lease` rebases that advance the experiment branch mid-run).
  - Model stamp from each run's `fo-log.jsonl` `assistant.message.model` field — this proves the run actually executed on `claude-opus-4-7` (or `-6` for the control). Without this, a green AC-1/AC-2 could be a false positive caused by a silent alias resolution somewhere in the stack.
  - Wallclock per run, both the streaming watcher's reported milestone times and the overall job duration.
  - For any FAILED run: the labeled `StepTimeout` message identifying which milestone fired, plus a one-paragraph attribution against the same milestone in #178's stage report (does the boilerplate visibly fail to discipline this specific tool call shape?).

- **What the implementation stage should NOT do**:
  - Do not modify #178's prose. If the boilerplate needs iteration, that is a separate task (a fail outcome on #177 + a new mitigation attempt).
  - Do not try to reproduce the high/xhigh `ECHO: ping reply received` failure here. It is a different surface; out of scope per the Decision section.
  - Do not file the workflow-default-pin change as part of this task in the FAIL path — write the recommendation note and let the captain triage filing.

## Stage Report (ideation)

### Summary

Sharpened #177 from an open-ended scoping document into a focused live experiment spec. The experiment stacks on the #178 mitigation branch, runs three CI dispatches against `tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips` on Claude Code 2.1.111, and outputs a binary recommendation: PASS → ship #178; FAIL → pin `--model claude-opus-4-6`. No new code, ~15 CI minutes, observability already in place via the streaming watcher.

### Checklist

1. **Read entity body in full.** DONE. Problem Statement, Evidence, Impact, Not-affected, Current mitigations, Evidence at high/xhigh, Open questions, Out of Scope, and Cross-references all read. Substantive sections preserved unchanged per the dispatch instruction.
2. **Read #178's design.** DONE. `docs/plans/ensign-prompt-tool-call-discipline-boilerplate.md` reviewed in full. Boilerplate text and placement (between Completion checklist and Summary placeholder) confirmed. Branch `spacedock-ensign/tool-call-discipline` has three commits including stage report; PR #113 is the open mitigation.
3. **Skim #173, #175, #176 for context.** DONE. Found in `docs/plans/_archive/`. #173 shipped `FOStreamWatcher` + `run_first_officer_streaming` (PR #109). #175 migrated 6 more live tests to the streaming watcher pattern (PR #111). #176 added `model_override` workflow input (PR #110); the `extra_args` plumbing follow-up referenced in #177's Cross-references is noted as the AC-3 dependency. None of these were redesigned or duplicated.
4. **`## Decision` section.** DONE. States plainly: focused live experiment, stacked on #178 (not main), three runs, two-path outcome (PASS → ship #178; FAIL → pin `--model claude-opus-4-6` per #176 plumbing). Explicit non-goals reiterated for the staff reviewer.
5. **`## Acceptance Criteria` section.** DONE. Four ACs: AC-1 (live CI opus-4-7 + low), AC-2 (live CI opus-4-7 + medium), AC-3 (negative control on opus-4-6 with local-fallback if #176 plumbing not yet fixed), AC-4 (recommendation deliverable matching the outcome). Each AC has verify command, pass/fail line, and capture list.
6. **`## Test Plan` section.** DONE. ~3 runs × ~5 min = ~15 CI minutes. Risk: low (no new code, observability via streaming watcher already shipped). E2E: yes (this task IS the E2E test). Static checks: none new — milestone-labeled `StepTimeout` handles attribution.
7. **`## Implementation Notes` section.** DONE. Worktree command (`git worktree add ... -b spacedock-ensign/opus-4-7-low-medium-experiment spacedock-ensign/tool-call-discipline`), rebase mechanics if #178 advances mid-experiment (`--force-with-lease`, never plain force), canonical `gh workflow run` shape, evidence-capture list (run URLs, `assistant.message.model` stamps from `fo-log.jsonl`, wallclock, labeled `StepTimeout` for fails), explicit do-not list.
8. **Update existing sections only if needed for sharpening.** PARTIAL. Added one line to `## Cross-references` linking #178 explicitly as the mitigation-under-test (the existing list mentioned #173/#174/#175/#176 but not #178). Problem Statement, Evidence, Impact, Not affected, Current mitigations, Evidence at high/xhigh, Open questions, and Out of Scope sections all preserved verbatim — they are correct and load-bearing.
9. **Commit on main.** Pending — will execute after this report is written. On `main` (clean working tree at start), so the commit will be `ideation: #177 experiment spec — stack on #178, opus-4-7 low/medium live test`. Not in a worktree; the dispatch instruction's worktree-commit branch is N/A.
10. **Stage Report.** DONE (this section).

### Recommendation for the ideation gate

**PASS.** The spec is now a tight, falsifiable experiment with a clear two-path outcome and proportionate test plan. Total cost ≈ 15 CI minutes, no new code, observability already in place. The staff reviewer should focus on whether AC-3's local-run fallback is acceptable when the #176-follow-up plumbing fix is not yet merged, and whether AC-4's PASS/FAIL deliverable shape (an in-entity recommendation note) is sufficient versus requiring a separate doc edit.

## Staff Review

**Verdict: APPROVE WITH CHANGES.** The spec is structurally sound as a binary ship/pin signal for #178, and the scoping discipline (no new code, ~15 CI min, single test) is correct given the broader scoping evidence already captured in the Problem Statement / Impact / Evidence sections. Two structural gaps warrant surgical fixes before implementation; neither requires re-ideation.

### Design soundness

The experiment cleanly answers "does #178's boilerplate make opus-4-7 viable at low/medium effort for the standing-teammate roundtrip case?" — not "is opus-4-7 broadly safe." The Decision section (lines 89-105) is explicit about that narrowing, which is the right call. ACs are independent (each is one CI dispatch with one varied parameter), falsifiable (labeled `StepTimeout` attribution from the streaming watcher), and verifiable (run URL + model stamp + wallclock).

One silent assumption in the Decision (lines 96-99): treating `opus-4-7 + low` and `opus-4-7 + medium` as a unit ("PASSED at both" → ship; "FAILED at either" → pin) presupposes that hallucination behavior is monotonic across effort levels. The Evidence section (lines 21-26) only directly evidences the regression at one effort, and the high/xhigh evidence (lines 51-63) shows behavior is *non*-monotonic across effort (different failure mode at high). A mixed AC-1 PASS / AC-2 FAIL outcome is neither addressed in AC-4 (lines 130-135) nor in the Decision's two-path mapping. Recommend AC-4 explicitly cover the mixed-outcome case (likely: pin, since any low/medium failure on the standard surface is shipping risk).

### Test plan sufficiency

One test is the right scope here — not because the surface is small (it isn't; the Impact section enumerates five distinct dispatch shapes), but because the broader scoping work is already in #177's Problem/Evidence/Impact sections, and #178 is a binary mitigation question. Generalizing #178's efficacy across all five impact surfaces would require its own scoping task and is reasonably out of scope for a "should we merge PR #113" decision. The Test Plan (lines 137-146) should make this scoping logic explicit so a future reader does not over-claim from a green result — recommend a one-line note in AC-4's PASS-path deliverable that the recommendation covers the standing-teammate roundtrip surface only, and broader confidence requires follow-up.

### Ideation-flagged questions

**(a) AC-3 local-fallback acceptability (line 125).** The fallback is acceptable for the negative-control role *only if* the local run uses the same Claude Code version (2.1.111) as the CI runs. The CI-runner-vs-local-machine confound is real but secondary — what AC-3 actually controls for is "is the test broken on the stacked branch, independent of model," and that signal survives the environment change. Recommend AC-3 add an explicit note: local run must use `claude --version` matching the CI dispatch's `claude_version=2.1.111`, captured in the evidence list. Without that pin, the fallback could mask a Claude Code version-induced failure as a model-induced one.

**(b) AC-4 deliverable shape (lines 130-135).** The in-entity note is the right deliverable for the PASS path (it unblocks #178's mod-block, which is the operative action). For the FAIL path, the recommendation note alone does NOT change behavior — workflow defaults still resolve `opus` to `opus-4-7`. The current spec correctly defers the workflow-default edit ("File a small follow-up task," line 133), but the FAIL path should be explicit that until that follow-up lands, the ensign hallucination remains live in production. Recommend AC-4 FAIL path require both the in-entity note AND a filed follow-up task (or issue) with a specific title, so the captain can't mistake "recommendation written" for "behavior changed."

### Gaps

- **Mixed-outcome handling**: covered above; AC-4 needs the third path.
- **`--force-with-lease` + captured run URL** (line 160): rebasing the experiment branch loses the SHA history the run URL was tested against. The Implementation Notes evidence list (lines 174-178) requires `assistant.message.model` stamps but not the experiment-branch SHA at dispatch time. Recommend adding "experiment branch SHA at dispatch" to the capture list — `gh run view` exposes it, but recording it inline in the entity prevents post-rebase ambiguity.
- **Wallclock comparison fairness** (line 114): the "matching the `claude-opus-4-6` baseline" comparison is fine when both pass, but a fail-fast `StepTimeout` at 60-180s is not directly comparable to a 2-3 minute pass. This isn't a structural problem — the streaming watcher's per-milestone times are the right comparison granularity — but the AC-1 Pass line should clarify that "wallclock ≈ 2-3 min" is the *pass* expectation, not a fail comparison.
- **Open question #6 (line 72)** about high/xhigh `ECHO: ping reply received` is correctly out of scope per Decision line 105; flagging only that a future reader of this entity should not conflate the two failure surfaces. The Decision's explicit non-goal already handles this; no change needed.

## Stage Report (staff review)

### Summary

Independent second-opinion read on #177's ideation spec. Verdict: **APPROVE WITH CHANGES** — the experiment is structurally sound and proportionately scoped, with two surgical fixes recommended (mixed-outcome AC-4 path; AC-3 Claude Code version pin) and three smaller capture/clarification gaps. Ideation ensign's own PASS recommendation is roughly correct; the changes are additive, not blocking re-ideation.

### Checklist

1. **Staff reviewer role, append-only `## Staff Review` section.** DONE. Did not modify Decision (lines 89-105), Acceptance Criteria (107-135), Test Plan (137-146), Implementation Notes (148-183), or the ideation Stage Report. Only appended `## Staff Review` and this `## Stage Report (staff review)` section.
2. **Read entity body in full, paying attention to flagged sections.** DONE. Read all of #177; specifically scrutinized Decision, ACs, Test Plan, Implementation Notes, and the ideation ensign's two reviewer-flags (AC-3 fallback, AC-4 deliverable shape).
3. **Read #178's design.** DONE. `docs/plans/ensign-prompt-tool-call-discipline-boilerplate.md` reviewed in full — boilerplate prose, placement (between checklist and Summary placeholder), and acceptance criteria all parsed. Confirms #177's experiment tests the right artifact.
4. **Assess design soundness.** DONE. Captured in Staff Review § Design soundness. Key finding: silent assumption that low and medium hallucination behave monotonically; AC-4 lacks a mixed-outcome path.
5. **Assess test plan sufficiency.** DONE. Captured in Staff Review § Test plan sufficiency. One test is right scope for binary ship/pin signal; recommended a clarifying note that PASS recommendation covers standing-teammate roundtrip surface only.
6. **Address AC-3 fallback and AC-4 deliverable shape flags.** DONE. Captured in Staff Review § Ideation-flagged questions. AC-3 fallback acceptable with `claude_version=2.1.111` pin made explicit. AC-4 FAIL path needs filed follow-up, not just in-entity note, since the note alone does not change production behavior.
7. **Look for gaps.** DONE. Captured in Staff Review § Gaps: mixed-outcome handling (AC-4), experiment-branch SHA capture under `--force-with-lease` rebase (Implementation Notes evidence list), wallclock-comparison clarification (AC-1 Pass line), and confirmed high/xhigh failure-mode separation is already handled.
8. **Append `## Staff Review` section, 300-500 words.** DONE. Section is structured per the dispatch instruction (verdict summary; Design soundness; Test plan sufficiency; Ideation-flagged questions a/b; Gaps). Length within budget (≈540 words including verdict line, slightly over the upper bound because the mixed-outcome and AC-3 version-pin findings are concrete structural requests rather than minor notes).
9. **Commit on main.** Pending — will commit after writing this report. Working tree was clean on `main` at start; commit message will be `staff-review: #177 ideation — APPROVE WITH CHANGES`.
10. **`## Stage Report (staff review)` at very end.** DONE (this section).

### One-line summary for the captain

Ideation is structurally sound; APPROVE WITH CHANGES — two surgical fixes (AC-4 mixed-outcome path + AC-3 Claude Code version pin) and three minor capture/clarification gaps; no re-ideation needed.

## Ideation Revision (post-staff-review)

Folded the staff reviewer's APPROVE-WITH-CHANGES findings into the ideation in place:

- **Surgical Fix #1 (AC-4 mixed-outcome path)**: AC-4 now has an explicit third bullet for mixed outcomes (AC-1 PASS / AC-2 FAIL or vice versa), routed to the FAIL path (pin `--model claude-opus-4-6`) since any low/medium failure on the standard surface is shipping risk.
- **Surgical Fix #2 (AC-3 Claude Code version pin)**: AC-3's local-fallback now requires `claude --version` matching the CI dispatch's `claude_version=2.1.111`, with the version output added to the AC-3 capture list.
- **Gap #1 (experiment-branch SHA capture)**: Implementation Notes evidence list now includes the experiment-branch SHA at dispatch time, with a note that `gh run view` exposes it and recording it inline survives `--force-with-lease` rebases.
- **Gap #2 (AC-1 wallclock clarification)**: AC-1 Pass line now states that "wallclock ≈ 2-3 min" is the *pass* expectation and that fail-fast `StepTimeout` at 60-180s is not directly comparable; per-milestone times from the streaming watcher are the right granularity for fail attribution. AC-2 inherits via "identical shape to AC-1."
- **Gap #3 (AC-4 PASS-deliverable surface-scope note)**: AC-4 PASS path now states the recommendation covers the standing-teammate roundtrip surface only, and broader confidence across the five impact surfaces enumerated in the Impact section requires follow-up scoping.

The original Decision (lines 89-105), Test Plan, and Implementation Notes structure is unchanged — edits were surgical insertions/clarifications within already-flagged lines, not rewrites of unflagged sections. The `## Staff Review` and `## Stage Report (staff review)` sections are preserved verbatim as the audit record of what the reviewer found.

## Stage Report (ideation revision)

### Summary

Folded all five staff-reviewer findings into #177's ideation in place. Two surgical AC fixes (AC-4 mixed-outcome path; AC-3 version pin) and three smaller gap closures (SHA capture; wallclock clarification; PASS-path surface-scope note). Decision, Test Plan, and Implementation Notes structure unchanged; Staff Review and its Stage Report preserved as audit record. Ready to re-present at the ideation gate.

### Checklist

1. **Read entity body in full.** DONE. Read all 258 lines including Decision (89-105), ACs (107-135), Test Plan (137-146), Implementation Notes (148-183), and the Staff Review section (208-233) that drives this revision.
2. **Surgical Fix #1: AC-4 mixed-outcome path.** DONE. Added a third bullet to AC-4 covering mixed outcomes (AC-1 PASS / AC-2 FAIL or vice versa), routed to the FAIL path (pin `claude-opus-4-6`) per the staff reviewer's recommended treatment. Mixed outcome must be noted explicitly so the next iteration of #178 can target the failing effort level.
3. **Surgical Fix #2: AC-3 Claude Code version pin.** DONE. AC-3's local-fallback path now requires `claude --version` matching CI's `claude_version=2.1.111`, with the rationale (a Claude Code version regression could mask as a model regression) stated inline. The capture list now includes the version output for the local-fallback path.
4. **Gap #1: experiment-branch SHA capture.** DONE. Added a new bullet to the Implementation Notes evidence list (between run URL and model stamp) requiring the experiment branch SHA at dispatch time, with the staff reviewer's exact phrasing about `gh run view` exposing it and recording it inline to survive `--force-with-lease` rebases.
5. **Gap #2: AC-1 wallclock clarification.** DONE. AC-1 Pass line now distinguishes the 2-3 min wallclock as the *pass* expectation and notes that fail-fast `StepTimeout` at 60-180s is not directly comparable. AC-2's "identical shape to AC-1" inherits the clarification automatically.
6. **Gap #3: AC-4 PASS-deliverable surface-scope note.** DONE. AC-4 PASS path now ends with a sentence that the recommendation covers the standing-teammate roundtrip surface only, and broader confidence across the five impact surfaces enumerated in #177's Impact section requires follow-up scoping.
7. **Do not modify Staff Review or its Stage Report.** DONE. Both sections (lines 208-258 in the pre-revision file) are untouched. Verified by inspection — only AC-1, AC-3, AC-4, and the Implementation Notes evidence list were edited.
8. **Do not broaden scope.** DONE. No new ACs added; Decision, Test Plan, and unflagged sections of Implementation Notes preserved verbatim. All edits sit inside already-flagged lines.
9. **Append `## Ideation Revision (post-staff-review)` section.** DONE. Cross-references each fold-in by the staff reviewer's labels (Surgical Fix #1, #2, Gap #1, #2, #3) and confirms the original Decision/Test Plan/Implementation Notes structure is unchanged.
10. **Commit on main.** Pending — will run immediately after this report write completes. Working tree was clean on `main` at start; commit message per the dispatch: `ideation-revision: #177 fold staff-review findings — AC-4 mixed-outcome path, AC-3 version pin, SHA capture, wallclock clarification, surface-scope note`.
11. **`## Stage Report (ideation revision)` at very end.** DONE (this section).

### One-line summary for the captain

Staff-review fold-in complete: two surgical AC fixes and three gap closures applied verbatim; Decision/Test Plan/Implementation Notes structure unchanged; ready to re-present at the ideation gate.

## Repurpose: Layer 2 Mitigation Investigation

**Proposed new title (FO updates frontmatter):** "opus-4-7 ensign hallucination — root cause investigation and Layer 2 mitigation experiments"

### Preamble (captain-directed pivot)

The original experiment (Decision lines 89-105, ACs lines 107-138, Outcome, Stage Reports, Staff Review) ran cleanly on 2026-04-16 and produced a definitive negative result: AC-1 and AC-2 both FAILED on the stacked #178 mitigation branch (boilerplate prose did not discipline `opus-4-7` at low/medium effort), and AC-3 was BROKEN (the negative-control surfaced an independent failure that contaminated the signal). Those sections above are PRESERVED VERBATIM as the audit trail that motivates this pivot — do not edit them.

The captain has redirected this entity to investigate **Layer 2 prompt-shape mitigations**: hypotheses about *which part* of the dispatch prompt primes `opus-4-7`'s fabrication behavior, so that a future engineering task can target the actual priming surface rather than wrapping more boilerplate around it.

### Decision

This entity now investigates whether prompt-shape mitigations can address `opus-4-7` hallucination at low/medium effort. The primary hypothesis under test: the **rich teammate descriptions** in the dispatch prompt's `### Standing teammates available in your team` section (introduced in commit `0acd6501`, "claude-team build auto-enumerates alive standing teammates into dispatch prompts") prime `opus-4-7` to fabricate plausible tool-call outcomes. The full per-teammate routing usage body (Patterns 1-4 for `comm-officer`, four caller patterns with example syntax) is exactly the surface a model could use to compose a believable `SendMessage` outcome without emitting the call.

Three independent experiments, each isolating a different variable, each ~5 minutes of local execution. Each experiment falsifies a distinct sub-hypothesis. Outcomes feed into a single `## Repurpose Outcome` section the implementation will write, recommending which prompt-shape mitigation (if any) should become a future engineering task.

The history complication noted in the dispatch (test_standing_teammate_spawn.py was added in `8ac41339` *before* the standing-teammates section in `0acd6501`) means the naive "go back in time and re-run the test" approach does not isolate the section's contribution — the test inherently requires the section to pass. AC-R1 and AC-R2 work around this: AC-R1 picks a *different* test that does not exercise teammate routing at all (so the section's presence is irrelevant to the test's pass condition), and AC-R2 patches the section in place to keep it structurally present but strip its rich content.

Out-of-scope per original entity discipline (restated):
- Building new infrastructure as ACs (e.g., new prompt-assembly modes, FO-side post-completion verification).
- Forcing API tool choice (`tool_choice: any` or similar SDK-level mitigation).
- Iterating on #178's prose. The original experiment killed that path.

### Outcome Map

3 ACs × {pass, fail} = 8 combinations. The five most informative outcomes and their recommended follow-ups:

| Outcome | Interpretation | Recommended follow-up |
|---|---|---|
| **All-three PASS** | Section richness is the priming AND sonnet works AND team-mode shape matters. | File two follow-ups: (1) compress the section emission per AC-R2; (2) consider `--model sonnet` as a workflow default (belt-and-suspenders). |
| **All-three FAIL** | Regression is broader than prompt-shape and broader than `opus-4-7`. | Recommend Layer 3 FO-side post-completion verification path AND surface to upstream as a model-level regression. |
| **AC-R1 PASS + AC-R2 FAIL** (counterintuitive) | Team-mode shape (section + Completion Signal block) matters but per-teammate prose richness does not. | Investigate what specifically in the section header / Completion Signal block is the priming token, not the per-teammate prose body. |
| **AC-R3 PASS + others FAIL** | Clean `--model sonnet` workaround independent of any prompt fix. | Pin `--model sonnet` in workflow defaults; deprioritize prompt-shape mitigation. (See AC-R3 Capture re: 2.1.111 re-confirmation if 2.1.112 was the run version.) |
| **AC-R2 PASS + others FAIL** | Most actionable Layer 2 result — section richness is the specific priming surface. | Compress the section emission and ship. |

Note: AC-R3 is orthogonal evidence (it tests model-vs-prompt, not section-priming). Any AC-R1 FAIL means prompt-shape mitigation alone does not help, regardless of AC-R2/AC-R3 — recommend Layer 3 (FO verification) as the path forward in that case.

### Acceptance Criteria

Each AC has Verify command, Pass/Fail line, and Capture list. All ACs run locally — no CI dispatch needed.

**AC-R1 — Counterfactual on a non-routing test.**

- Hypothesis isolated: "the regression is *specific to* team-mode dispatch shape (which includes the standing-teammates section, the Completion Signal block, and team-mode framing)." Note: this AC varies three things at once vs. the standing-teammate test (section absence, team-mode absence, different test surface — FO-driven gate hold rather than ensign-driven roundtrip). It cannot isolate the section alone; it isolates the broader team-mode dispatch shape. If a test that does NOT involve any of these team-mode elements passes on `opus-4-7` + low effort, some aspect of team-mode dispatch shape is implicated as the priming surface. If it fails with the same hallucination class, the regression is independent of all team-mode prompt elements and prompt-shape mitigation will not help.
- Test selected: `tests/test_gate_guardrail.py::test_gate_guardrail`. Verified clean (no `standing`/`teammate`/`comm-officer`/`echo-agent` references in the file). It is the cheapest live test in the repo (~60s, ~$0.02 haiku per `tests/README.md` lines 192-199), runs on a minimal gated workflow fixture with no teammate routing.
- Verify (run from repo root):

  ```
  unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py \
    --runtime claude --model opus --effort low -v
  ```

  Confirm `claude --version` reports `2.1.111` (or 2.1.112 — note Claude Code drift is mild; original Evidence shows the regression observed on both). Capture `claude --version` output before the run.
- Pass: pytest exit code 0, all gate-guardrail assertions pass. Some aspect of team-mode dispatch shape (section + Completion Signal block + team-mode framing) is implicated as the priming surface — narrower than "all prompts" but broader than "the section alone." AC-R2 is the follow-up needed to narrow further within team-mode shape.
- Fail: pytest fails with the same FO-level hallucination class (FO claims status changes / archives without the corresponding tool calls). All prompt-shape mitigation is ruled out — the regression is independent of which specific dispatch element is present, so stripping or compressing any team-mode element will not help.
- Capture: `claude --version` output, pytest exit code, the test's `KEEP_TEST_DIR=1`-preserved `fo-log.jsonl`, and the `assistant.message.model` stamps from that log proving the run actually executed on `claude-opus-4-7`. For a fail, the specific assertion that fired and the tool-call inventory around it. Additionally for a fail: manually inspect `fo-log.jsonl` and identify whether the FO emitted the corresponding tool calls for any state-change claims it makes in text (the streaming watcher gives objective milestones for AC-R2/R3 but AC-R1's failure surface is FO-side, so this manual fo-log inspection is required to classify the failure as the same hallucination class as the original regression). Note: AC-R1 also runs from inside the rebased worktree for environmental consistency with AC-R2 and AC-R3 (a cheaper repo-root option exists but loses that consistency).

**AC-R2 — Section-stripped variant on the standing-teammate test.**

- Hypothesis isolated: "the *richness* of teammate descriptions (full Patterns 1-4 prose, examples, routing contracts) is what primes fabrication, not the section's mere presence." A minimal one-line-per-teammate variant lets us keep the structural section but remove the prose surface. If the test passes, rich descriptions are the priming and the fix is to compress them. If it fails, the section's presence itself is the priming and a deeper change (remove the section entirely, or use a different routing-discovery mechanism) would be needed.
- Patch target: `skills/commission/bin/claude-team` lines 276-308, the `enumerate_alive_standing_teammates` block. The MINIMAL variant emits only:

  ```
  ### Standing teammates available in your team

  - {name} is available; SendMessage to it.
  ```

  one line per teammate, no descriptions, no per-pattern usage bodies, no "Full routing contract" footer. The implementation patches the loop in place inside the worktree (do NOT push or merge — this is an experimental local patch).
- Verify: from inside the rebased worktree (see Implementation Notes), apply the minimal patch to `skills/commission/bin/claude-team`, then run:

  ```
  unset CLAUDECODE && uv run pytest \
    tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips \
    --runtime claude --model opus --effort low -v
  ```

- Pass: pytest exit code 0; both `SendMessage to echo-agent observed` and `ECHO: ping reply received` milestones fire. Rich descriptions are the priming surface; the engineering follow-up is to compress the section's emission.
- Fail: same milestone timeouts as the original AC-1 failure (most likely `SendMessage to echo-agent observed` at 240s). Section presence (not its richness) is the priming, OR the priming sits elsewhere in the prompt entirely (Completion Signal block, "via SendMessage" mention in the section header, etc.). Engineering follow-up needs to investigate further before settling on a fix.
- Capture: the patch diff (paste into the entity's stage report), pytest exit code, the preserved `fo-log.jsonl` model stamps, milestone timing breakdown from the streaming watcher's labeled output. For a fail, capture the dispatch prompt as it appeared with the patch applied (read the `Agent` tool_use input from `fo-log.jsonl`) so the next investigator can confirm the patch took effect.

**AC-R3 — Different ensign model.**

- Hypothesis isolated: "this is `opus-4-7`-specific calibration; `sonnet` does not exhibit the pattern." A working alternative ensign model is a viable workaround independent of any prompt-shape fix and is worth knowing even if AC-R1/AC-R2 produce a clean answer.
- Verify:

  **Pre-step (HARD REQUIREMENT, not a suggestion):** before running pytest, the implementer MUST run:

  ```
  git diff --quiet skills/commission/bin/claude-team
  ```

  If this returns non-zero, ABORT — that indicates AC-R2's patch was not reverted and AC-R3 would silently run against a contaminated prompt. Re-run AC-R2's revert sequence (`git checkout -- skills/commission/bin/claude-team`) and re-check before proceeding.

  Then run:

  ```
  unset CLAUDECODE && uv run pytest \
    tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips \
    --runtime claude --model sonnet --effort low -v
  ```

  No code patch — this exercises the standard prompt assembly with `--model sonnet` instead of `--model opus`.
- Pass: pytest exit code 0; both teammate-routing milestones fire. `sonnet` is a viable ensign-model workaround; the captain can pin `--model sonnet` in workflow defaults as a safer default than `--model opus` until upstream is fixed.
- Fail: pytest fails on the same milestones. `sonnet` ensign also has problems — not a clean workaround. The pattern is broader than `opus-4-7`-specific calibration; root cause is likely either the prompt shape (across models) or a Claude Code runtime issue.
- Capture: pytest exit code, `fo-log.jsonl` `assistant.message.model` stamps proving the ensign sub-call actually used `sonnet` (NOT `opus`), milestone timing, and `claude --version` output. Note: if the run uses 2.1.112, a PASS should be re-confirmed on 2.1.111 before pinning `--model sonnet` as a default-shift recommendation, since sonnet behavior on 2.1.112 was not pre-validated in this entity.

### Test Plan

- **Mechanics**: each AC is a single local `pytest` invocation against a real Claude runtime. The streaming watcher (#173) provides per-milestone fail attribution; preserved `fo-log.jsonl` artifacts (set `KEEP_TEST_DIR=1`) provide the audit evidence (model stamps, tool-call inventory).
- **Cost**: ~5 min per AC, ~15-20 min total. AC-R1 is the cheapest (gate-guardrail, no teammate roundtrip); AC-R2 and AC-R3 each run the standing-teammate test with its 4 milestone × 240s budget but typically fail-fast at ~60-180s when the regression appears.
- **Local execution only**: no CI dispatch. The original experiment burned three CI dispatches; the Layer 2 investigation does not need them — local runs with the streaming watcher provide the same evidence quality. Cost in CI minutes: zero.
- **Claude Code version**: ideally `2.1.111` to match the original experiment, but `2.1.112` is acceptable per the original Evidence (regression observed on both). Capture `claude --version` for each AC so the gate review can confirm version drift did not contaminate the result.
- **Static checks**: none new. The streaming watcher's labeled milestones and the `fo-log.jsonl` model stamps are the structural evidence.
- **E2E tests**: yes — these three ACs ARE E2E experiments. No new test code; AC-R2 patches existing infrastructure locally; AC-R1 and AC-R3 use existing tests with different CLI flags.

### Implementation Notes

**Worktree state**. The existing 177 worktree at `.worktrees/spacedock-ensign-opus-4-7-ensign-hallucination-scope` is checked out at #178's tip (`e1a087df`) and lacks #179's plumbing fix (#180, commit `addcbeee` on main). For the Layer 2 experiments, the implementer should:

- **Option A (preferred)**: rebase the existing worktree onto current `main`. From inside the worktree:

  ```
  git fetch origin
  git rebase origin/main
  ```

  This brings #179's plumbing fix and the latest `claude-team build` source code (which AC-R2 will patch).

- **Option B**: discard the existing worktree and recreate from main. From the repo root:

  ```
  git worktree remove .worktrees/spacedock-ensign-opus-4-7-ensign-hallucination-scope
  git worktree add .worktrees/spacedock-ensign-opus-4-7-ensign-hallucination-scope -b spacedock-ensign/opus-4-7-layer-2 main
  ```

  Cleaner but loses any in-progress local state from the original experiment.

Either option gives the implementer up-to-date plumbing AND the latest claude-team source. Pick Option A unless the existing worktree has uncommitted state that makes rebase messy.

**AC-R2 patch shape**. The patch replaces the loop body at `skills/commission/bin/claude-team:287-301` (the `for name, description, mod_path in standing_teammates:` block) with a single line per teammate:

```python
for name, description, mod_path in standing_teammates:
    lines.append(f'- {name} is available; SendMessage to it.')
```

And drops the "Full routing contract" footer at lines 303-307. The canonical revert sequence: apply the patch, run the test, then capture the diff with `git diff skills/commission/bin/claude-team > /tmp/ac-r2.patch` (this preserves the patch as a file and satisfies AC-R2's "paste patch diff in stage report" Capture requirement more reliably than reading from a stash), then `git checkout -- skills/commission/bin/claude-team` for a clean revert. Use this exact sequence rather than `git stash` — the captured `.patch` file is the authoritative artifact.

**Evidence to capture for each AC** (write into the `## Repurpose Outcome` section):

- `claude --version` output (proves the runtime version).
- pytest exit code and the specific assertion / milestone that fired (for fails).
- `assistant.message.model` stamps from the relevant `fo-log.jsonl` (proves the model actually executed; AC-R1 should show `claude-opus-4-7` for both FO and ensign, AC-R3 should show `sonnet` for the ensign sub-call).
- Per-milestone timing from the streaming watcher's labeled `StepTimeout` output (for AC-R2 and AC-R3 fails).
- For AC-R2: the patch diff and the dispatched prompt's standing-teammates section (extract from `fo-log.jsonl` `Agent` tool_use input) so a future reader can confirm the patch took effect.
- **Un-patched dispatch prompt baseline**: capture one un-patched dispatch prompt (extract from AC-R3 or AC-R1's `fo-log.jsonl` `Agent` tool_use input) AND one patched dispatch prompt (extract from AC-R2). Both go in the `## Repurpose Outcome` section as a side-by-side excerpt so a future reader can confirm exactly what AC-R2 changed.

**What the implementation MUST NOT do**:

- Do not commit the AC-R2 patch. It is a local experimental patch; revert before AC-R3 runs.
- Do not modify the original experiment's sections (Decision lines 89-105, ACs 107-138, Outcome, Stage Reports, Staff Review). Those are preserved audit trail.
- Do not add a fourth experiment. The three are scoped to be independent; any further hypothesis goes into a follow-up entity.
- Do not propose API-level mitigations (tool_choice forcing, SDK-level changes) — out of scope per the captain's directive.

The actionable output is a `## Repurpose Outcome` section recommending which prompt-shape mitigation (if any) actually moves the needle, evidence-backed, ready to feed into a future engineering task.

## Stage Report (ideation revision, repurpose)

### Summary

Repurposed #177 to investigate Layer 2 prompt-shape mitigations after the original experiment's AC-1/AC-2 FAIL + AC-3 BROKEN outcome falsified #178's boilerplate approach. Three independent local experiments (~15-20 min total): AC-R1 isolates the standing-teammates section's contribution via a non-routing test (`test_gate_guardrail`); AC-R2 isolates the section's *richness* via a minimal-content patch on `claude-team`; AC-R3 isolates `opus-4-7`-specific calibration via a `sonnet` ensign run. All original sections preserved verbatim as audit record.

### Checklist

1. **Read entity body in full and recognize the captain-directed pivot.** DONE. Read all 294 lines of the pre-repurpose file. Recognized the original experiment ran cleanly and is factual; AC-1/AC-2 failed and AC-3 broke as documented. The captain's pivot redirects to Layer 2 hypothesis testing while preserving the original sections as audit trail.
2. **Read FO-as-API use-cases spec and #178 body.** DONE. `docs/superpowers/specs/2026-04-17-spacedock-fo-as-api-use-cases.md` reviewed (full architecture context; Use Case 4 directly cites #177 as the failure pattern motivating hallucination-resistant API mutations). `docs/plans/ensign-prompt-tool-call-discipline-boilerplate.md` reviewed (#178's boilerplate prose, placement between Completion checklist and Summary placeholder, AC-3 acknowledges the experiment may produce a negative result — which it did).
3. **Investigate standing-teammate prompt section history.** DONE. `git log --oneline --all -S 'Standing teammates available in your team' -- skills/commission/bin/claude-team` returns one commit: `0acd6501 impl: #162 cycle 2 — claude-team build auto-enumerates alive standing teammates into dispatch prompts`. `git log --oneline --diff-filter=A -- tests/test_standing_teammate_spawn.py` returns `8ac41339 tests: #162 live E2E standing teammate spawn + roundtrip fixture`. Test was added BEFORE the section emission (the test inherently requires the section), so a naive "go back in time" experiment does not isolate the variable. Recorded in the Decision section's history-complication paragraph.
4. **Identify candidate non-routing live tests.** DONE. Surveyed `tests/`: skipped/xfail tests (`test_scaffolding_guardrail`, `test_repo_edit_guardrail`, `test_rejection_flow`, `test_push_main_before_pr`, `test_dispatch_completion_signal`, etc.) are not viable. `test_gate_guardrail.py` is clean (verified no `standing`/`teammate`/`comm-officer`/`echo-agent` references via grep), is the cheapest live test in the repo per `tests/README.md` lines 192-199, and exercises the FO+ensign loop on a gated workflow fixture without teammate routing. Selected as AC-R1's test.
5. **Append `## Repurpose: Layer 2 Mitigation Investigation` section after all existing sections.** DONE. Section appended after the existing `## Stage Report (ideation revision)` (line 294). Preamble explains the captain-directed pivot and links to the original AC-3 BROKEN finding as the trigger.
6. **Write `### Decision` subsection.** DONE. Names the primary hypothesis (rich teammate descriptions in the dispatch prompt prime `opus-4-7` fabrication), three independent experiments, ~5 min cost each, single `## Repurpose Outcome` section as the actionable output. Out-of-scope items restated per original entity discipline.
7. **Write `### Acceptance Criteria` for three experiments.** DONE. AC-R1 (counterfactual on `test_gate_guardrail`), AC-R2 (section-stripped variant on standing-teammate test, with explicit patch shape), AC-R3 (sonnet ensign on standing-teammate test). Each AC has Verify command, Pass/Fail line, and Capture list per the dispatch instruction.
8. **Write `### Test Plan`.** DONE. Local-execution mechanics, `KEEP_TEST_DIR=1` for evidence preservation, streaming watcher per-milestone attribution, ~15-20 min total cost, no CI dispatch, Claude Code version capture (2.1.111 ideally; 2.1.112 acceptable per original Evidence).
9. **Write `### Implementation Notes` with worktree state guidance.** DONE. Option A (rebase existing worktree onto main) preferred; Option B (recreate from main) as alternative. Both bring #179 plumbing + latest `claude-team` source. AC-R2 patch shape spelled out (replace loop body at `claude-team:287-301`; drop the routing-contract footer). Evidence-capture list and explicit must-not list included.
10. **Update title in prose.** DONE. Proposed title at top of Repurpose section: "opus-4-7 ensign hallucination — root cause investigation and Layer 2 mitigation experiments" — for the FO to pick up in frontmatter.
11. **Commit on main.** Pending — will run immediately after this report write completes. Working tree was clean on `main` at start.
12. **Stage Report (ideation revision, repurpose) at very end.** DONE (this section).
13. **Do NOT modify existing sections.** DONE. Original Decision (lines 89-105), Acceptance Criteria (107-138), Test Plan (137-146), Implementation Notes, Outcome, Stage Reports, Staff Review, and Stage Report (staff review) all preserved verbatim. Only appended new sections after line 294.

### One-line summary for the FO at the gate

Repurpose-ideation complete: three independent local experiments (AC-R1 non-routing test counterfactual, AC-R2 section-stripped patch, AC-R3 sonnet ensign) scoped at ~15-20 min total cost; each isolates a different priming hypothesis; original sections preserved as audit trail.

## Staff Review (repurpose)

**Verdict: APPROVE WITH CHANGES.** The Layer 2 pivot is well-motivated and the three-AC structure is correct in spirit. AC-R2 and AC-R3 are clean. AC-R1 has a real isolation problem that should be acknowledged in the spec rather than fixed (re-scoping AC-R1 would broaden the experiment beyond its 15-20 min budget). AC-R4-style mixed-outcome enumeration should be added to the Decision section before implementation, since 8 outcome combinations exist and the entity currently delegates them to implementer judgment. Three smaller capture/sequencing gaps; none blocks re-ideation.

### Independence claim

The three ACs each *vary* a different surface, but only AC-R2 and AC-R3 cleanly *isolate* a single variable.

- **AC-R2** (lines 339-361) keeps everything constant except the section's prose richness — section header preserved, one-line-per-teammate, same test, same model, same fixture. This is a clean isolation.
- **AC-R3** (lines 363-377) keeps everything constant except `--model`. Clean isolation of model-vs-prompt.
- **AC-R1** (lines 323-337) varies *three* things at once: (a) section absence (the named hypothesis), (b) team mode entirely (the gated-pipeline fixture has no `agents:` config — verified at `tests/fixtures/gated-pipeline/README.md:1-19`, no team configured, so `enumerate_alive_standing_teammates` returns empty and the Completion Signal block at `claude-team:310-319` is also skipped), and (c) different test surface (FO-driven gate hold vs ensign-driven roundtrip). A FAIL on AC-R1 cannot distinguish "section primes hallucination" from "team-mode dispatch shape primes hallucination" or "this test simply also exhibits the regression on a different surface." The Decision (line 312) acknowledges the test was added before the section, but does not acknowledge the multi-variable change at AC-R1.

Recommend: keep AC-R1 in the spec but re-frame its Pass/Fail interpretation. PASS still implicates *some* aspect of team-mode dispatch shape (section + completion-signal + standing-teammates header) as the priming surface — narrower than "the section" but still actionable. FAIL still rules out *all* prompt-shape mitigation. Update line 325's "specific to dispatch prompts that contain the standing-teammates section" to "specific to team-mode dispatch shape (which includes the standing-teammates section, the Completion Signal block, and team-mode framing)."

### AC-R1 test selection sanity

Verified `tests/test_gate_guardrail.py`: it does invoke the FO via `run_first_officer_streaming` (line 47-72), uses `install_agents` for the claude path (line 39), and the test exercises the FO+ensign loop on a gated workflow. The fixture (`gated-pipeline/README.md`) has no `agents:` block, so `claude-team build` does NOT emit the standing-teammates section for any dispatched ensign in this test. Premise checks out — the section is genuinely absent. The confound is *which other variables* are also absent (see Independence above), not whether the section is absent.

One additional caveat: the gate-guardrail test's failure surface is the FO itself (FO self-approving at the gate), not an ensign hallucinating in a stage report. The original regression class (Evidence lines 21-26) was *ensign* hallucination of `SendMessage` outcomes inside a stage report. AC-R1's pass condition (FO halts at gate) is a different observable. Recommend the Pass/Fail line at lines 335-336 explicitly note this asymmetry: a FAIL on AC-R1 would have to mean either "FO self-approved" or "FO claimed a state change without making the tool call" — the latter is the closer analog to the original regression class.

### AC-R2 patch shape sanity

Verified `claude-team` on main:
- Lines 287-301 contain the loop body (`for name, description, mod_path in standing_teammates:` plus the conditional `usage_body` branches) exactly as the spec describes.
- Lines 302-307 contain the `lines.append('')` + "Full routing contract:" footer that the spec says to drop.
- The patch as specified (lines 415-417 of the entity) leaves the section's heading + at least one bullet, so any "section structurally present" check still passes.

Patch target is correct. One sequencing gap (see Gaps).

### Outcome enumeration completeness

3 ACs × {pass, fail} = 8 combinations. The Decision section (line 310) says outcomes "feed into a single `## Repurpose Outcome` section the implementation will write." The implementer is left to judge what each outcome combination *recommends*. With Outcome being the single actionable deliverable, the spec should pre-enumerate the most informative combinations rather than delegating that interpretation to the implementer. Suggested minimum table:

- **all-three PASS**: section richness is the priming AND sonnet works AND the section is necessary. Recommend filing two follow-ups: (1) compress the section emission per AC-R2, (2) consider `--model sonnet` workflow default as belt-and-suspenders.
- **all-three FAIL**: regression is broader than prompt-shape and broader than `opus-4-7`. Recommend the FO-side post-completion verification path (Layer 3) and surface to upstream.
- **AC-R1 PASS + AC-R2 FAIL** (counterintuitive): section presence (or team-mode shape) matters but its richness does not. Recommend investigating *what specifically* in the section header / Completion Signal block is the priming token, not the per-teammate prose.
- **AC-R3 PASS + others FAIL**: clean `--model sonnet` workaround independent of any prompt fix. Recommend pinning `--model sonnet` in workflow defaults; deprioritize prompt-shape mitigation.
- **AC-R2 PASS + others FAIL**: most actionable Layer 2 result — compress the section, ship it.
- **Any AC-R1 FAIL**: prompt-shape mitigation alone does not help; recommend Layer 3 (FO verification) regardless of other ACs.

Recommend adding a `### Outcome Map` subsection to Decision (after line 318) with at least the five rows above. Without it, the implementer's `## Repurpose Outcome` will likely under-enumerate.

### Silent assumptions

- **(a) "Same hallucination class" objective definition** (lines 325, 336, 360). The streaming watcher surfaces `StepTimeout` with a milestone label — that's objective for AC-R2 and AC-R3 (same milestones expected). For AC-R1, the test does not have an ensign-side roundtrip milestone, so "same hallucination class" requires the implementer to inspect `fo-log.jsonl` for the FO-equivalent (FO claims a status change without the tool call). Recommend the Capture list at line 337 explicitly require: for a FAIL, identify whether the FO emitted the corresponding tool calls for any state-change claims it makes in text. Without that, AC-R1 FAIL evidence is judgment-call.
- **(b) AC-R3 isolates model, not section** (lines 363-377). Spec correctly notes this is "worth knowing even if AC-R1/AC-R2 produce a clean answer." No issue — flagging only that AC-R3 does not test the priming-via-section hypothesis at all, just the `opus-4-7`-specific calibration sub-question. The Decision section (line 308) frames the *primary* hypothesis as section richness, so AC-R3 is admitted as a secondary question. Acceptable, but the Outcome Map (above) should treat AC-R3 as orthogonal evidence, not as falsifying or confirming the primary hypothesis.
- **(c) Claude Code 2.1.111 vs 2.1.112 confound** (line 384). Spec says both acceptable per original Evidence. Reasonable for AC-R1 and AC-R2 (both should reproduce on either). For AC-R3 specifically, sonnet behavior under 2.1.112 has not been directly evidenced in this entity. Recommend AC-R3's Capture list (line 377) include the `claude --version` output explicitly, and if 2.1.112 is used, the implementer should note that sonnet's behavior on 2.1.112 was not pre-validated and a PASS should be re-confirmed on 2.1.111 before pinning `--model sonnet` as a recommendation.

### Gaps

- **AC-R2 patch revert mechanism** (line 419): "git stash or git checkout to revert" — the spec offers two equivalent options without picking one. `git stash` is reversible (the patch survives in the stash); `git checkout -- skills/commission/bin/claude-team` is destructive (patch lost unless captured to a separate file first). Recommend: capture the patch as a `.patch` file via `git diff > /tmp/ac-r2.patch` BEFORE applying, then `git checkout` to revert. The diff-to-file step also satisfies the Capture-list requirement at line 361 ("paste the patch diff into the stage report") more reliably than reading from the stash.
- **AC-R2 → AC-R3 patch leak risk** (line 419): if the implementer skips the revert step or does it incorrectly, AC-R3 silently runs against the patched `claude-team` and its result is contaminated. Recommend: AC-R3's Pre-step (added to its Verify command) be `git diff --quiet skills/commission/bin/claude-team` to confirm zero pending changes before running. A non-zero exit from that check should abort AC-R3.
- **Un-patched dispatch prompt baseline** (Implementation Notes, lines 421-427): AC-R2's Capture requires the *patched* prompt for confirmation. There is no requirement to capture the *un-patched* prompt as a baseline. Without it, a future reader cannot diff the two prompts to see exactly what AC-R2 changed. Recommend adding to the Implementation Notes evidence list: capture one un-patched dispatch prompt (from any AC-R3 run) and one patched dispatch prompt (from AC-R2), preferably as side-by-side excerpts in the Outcome section.
- **AC-R1 worktree applicability**: AC-R1 uses `tests/test_gate_guardrail.py` which lives at the repo root, not in the worktree. The Implementation Notes (line 388-410) describe rebasing the existing worktree onto main, but AC-R1 does not need any patch — it only needs current code. Spec should clarify whether AC-R1 runs from the rebased worktree (consistent with AC-R2/R3) or from the repo root (cheaper). Recommend the worktree for consistency, but the spec is currently silent.

Length: ≈940 words; over the upper bound because AC-R1's confounding and the missing Outcome Map are concrete structural requests, not minor notes.

## Stage Report (staff review, repurpose)

### Summary

Independent staff review of #177's repurpose ideation (Layer 2 prompt-shape investigation). Verdict: **APPROVE WITH CHANGES.** The three-AC structure is sound; AC-R2 and AC-R3 are clean isolations. AC-R1 confounds three variables (section absence + team-mode absence + different test surface) — recommend re-framing its interpretation rather than re-scoping it. Outcome enumeration is incomplete (8 combinations, no Outcome Map) — recommend adding a 5-row Outcome Map subsection to Decision before implementation. Three smaller gaps: AC-R2 patch revert mechanism, AC-R2→AC-R3 patch-leak guard, missing un-patched-prompt baseline.

### Checklist

1. **Append-only `## Staff Review (repurpose)` section; do not modify prior sections.** DONE. Added `## Staff Review (repurpose)` and this `## Stage Report (staff review, repurpose)` after line 462. Verified by inspection: lines 1-294 (original entity) and lines 296-462 (Repurpose section + revision report) are untouched.
2. **Read entity body in full, focus on Repurpose section (lines 296+).** DONE. Read all 462 lines. Original sections noted as audit trail per dispatch instruction. Repurpose Decision (306-318), AC-R1 (323-337), AC-R2 (339-361), AC-R3 (363-377), Test Plan (379-386), Implementation Notes (388-436) all scrutinized.
3. **Verify experimental design's independence claim.** DONE. Captured in Staff Review § Independence claim. AC-R2 and AC-R3 isolate cleanly; AC-R1 varies three things at once (section absence, team-mode absence, different test surface). Recommended re-framing AC-R1's interpretation rather than re-scoping the test.
4. **Sanity-check AC-R1's test selection.** DONE. Captured in Staff Review § AC-R1 test selection sanity. Verified `tests/test_gate_guardrail.py:30-156` invokes FO via `run_first_officer_streaming`, uses `install_agents`. Verified `tests/fixtures/gated-pipeline/README.md:1-19` has no `agents:` block. Premise (section is absent) is correct; confound (other variables are also absent) is the actual problem. Also flagged that the test's failure surface (FO self-approval) is not the same observable as the original regression (ensign hallucination in stage report).
5. **Sanity-check AC-R2's patch shape.** DONE. Captured in Staff Review § AC-R2 patch shape sanity. Verified `skills/commission/bin/claude-team:287-301` contains the loop, `:302-307` contains the routing-contract footer. Patch leaves section structurally present. Patch target is correct.
6. **Examine outcome enumeration.** DONE. Captured in Staff Review § Outcome enumeration completeness. 8 combinations exist; spec delegates interpretation to implementer. Recommended adding a 5-row Outcome Map subsection to Decision (after line 318) covering all-pass, all-fail, AC-R1 PASS + AC-R2 FAIL counterintuitive, AC-R3 PASS only, and AC-R2 PASS only.
7. **Check for silent assumptions.** DONE. Captured in Staff Review § Silent assumptions. (a) "Same hallucination class" requires objective definition for AC-R1 (recommended Capture-list addition); (b) AC-R3 admittedly does not test the section hypothesis (acceptable, flagged for the Outcome Map); (c) 2.1.112 sonnet behavior not pre-validated, recommended `claude --version` capture for AC-R3.
8. **Look for gaps.** DONE. Captured in Staff Review § Gaps. (a) AC-R2 patch revert mechanism ambiguous (`git stash` vs `git checkout`); recommended capturing as `.patch` file before apply. (b) AC-R2 → AC-R3 patch-leak risk; recommended `git diff --quiet` pre-check on AC-R3. (c) Missing un-patched dispatch prompt baseline; recommended capturing one for side-by-side. (d) AC-R1 worktree applicability not stated; recommended running from worktree for consistency.
9. **Append `## Staff Review (repurpose)` section, 300-500 words (longer if structural problem).** DONE. Section is ≈940 words — over the budget because AC-R1's confounding and the missing Outcome Map are concrete structural requests, not notes.
10. **Commit on main.** Pending — will run immediately after this report write completes. Working tree was clean on `main` at start; commit message will be `staff-review: #177 repurpose ideation — APPROVE WITH CHANGES`.
11. **`## Stage Report (staff review, repurpose)` at very end.** DONE (this section).

### One-line summary for the captain

Repurpose-ideation is structurally sound; APPROVE WITH CHANGES — AC-R1 confounds three variables (re-frame interpretation, don't re-scope), Decision should add an Outcome Map enumerating the 5 most informative outcome combinations, plus three smaller patch-handling gaps.

## Repurpose Revision (post-staff-review)

This is a focused fold-in pass of the staff reviewer's APPROVE-WITH-CHANGES findings (see `## Staff Review (repurpose)` for the source-of-truth findings). The Repurpose section's Decision / AC-R1 / AC-R2 / AC-R3 / Implementation Notes structure is unchanged except for the targeted edits below; no section was rewritten.

Folded-in findings:

- **Surgical Fix #1 (AC-R1 reframing)**: AC-R1's Hypothesis isolated line now acknowledges that the AC varies three things at once (section absence, team-mode absence, different test surface) and reframes the priming surface as "team-mode dispatch shape" (section + Completion Signal block + team-mode framing). Pass/Fail interpretation lines updated to match: PASS implicates "some aspect of team-mode dispatch shape" (broader than the section alone but still actionable; AC-R2 is the follow-up to narrow further); FAIL rules out *all* prompt-shape mitigation regardless of which dispatch element. Test selection (`test_gate_guardrail`) and Verify command unchanged.
- **Surgical Fix #2 (Outcome Map)**: new `### Outcome Map` subsection added to Decision (after the out-of-scope paragraph), enumerating the 5 most informative outcomes the staff reviewer specified — all-three PASS, all-three FAIL, AC-R1 PASS + AC-R2 FAIL (counterintuitive), AC-R3 PASS + others FAIL, and AC-R2 PASS + others FAIL — each with interpretation and recommended follow-up. Formatted as a markdown table. Closing note flags AC-R3 as orthogonal and any AC-R1 FAIL as triggering Layer 3.
- **Surgical Fix #3 (AC-R2 patch sequencing)**: Implementation Notes' revert mechanism replaced the ambiguous "git stash or git checkout" wording with the canonical sequence — `git diff skills/commission/bin/claude-team > /tmp/ac-r2.patch` to capture, then `git checkout -- skills/commission/bin/claude-team` to revert. Notes that the captured `.patch` file is the authoritative artifact for AC-R2's "paste patch diff in stage report" Capture requirement.
- **Surgical Fix #4 (AC-R3 patch-leak guard)**: AC-R3's Verify section now opens with a HARD-REQUIREMENT Pre-step — `git diff --quiet skills/commission/bin/claude-team` must run before pytest, and a non-zero exit ABORTS AC-R3 (would indicate AC-R2's patch was not reverted).
- **Surgical Fix #5 (un-patched dispatch prompt baseline)**: Implementation Notes' evidence-capture list adds a new requirement — capture one un-patched dispatch prompt (from AC-R1 or AC-R3 fo-log) and one patched dispatch prompt (from AC-R2), both placed in the `## Repurpose Outcome` section as side-by-side excerpts.
- **Small fix — AC-R1 Capture list addition**: AC-R1's Capture list now requires, for a FAIL, manual `fo-log.jsonl` inspection to identify whether the FO emitted the corresponding tool calls for any state-change claims it makes in text (the streaming watcher gives objective milestones for AC-R2/R3 but AC-R1's failure surface — FO self-approval — is different from the original ensign-hallucination class).
- **Small fix — AC-R1 worktree applicability**: AC-R1's Capture list now notes that AC-R1 also runs from inside the rebased worktree for environmental consistency with AC-R2 and AC-R3 (cheaper repo-root option exists but loses that consistency).
- **Small fix — AC-R3 sonnet 2.1.112 caveat**: AC-R3's Capture list now explicitly requires `claude --version` output, with a note that a 2.1.112 PASS should be re-confirmed on 2.1.111 before pinning `--model sonnet` as a default-shift recommendation (sonnet behavior on 2.1.112 was not pre-validated in this entity).

The original-experiment sections (lines 1-294), the `## Staff Review (repurpose)` section, and the `## Stage Report (staff review, repurpose)` section were not modified — those are audit records.

## Stage Report (ideation revision, repurpose post-staff-review)

### Summary

Surgical fold-in pass of the staff reviewer's APPROVE-WITH-CHANGES findings into the Repurpose section. Three surgical fixes (AC-R1 reframing to acknowledge three-variable confound, new Outcome Map subsection in Decision with 5 enumerated outcomes, AC-R2 canonical patch sequencing) plus AC-R3 patch-leak Pre-step, un-patched prompt baseline requirement, and three smaller capture additions (AC-R1 fo-log inspection on FAIL, AC-R1 worktree note, AC-R3 `claude --version` + 2.1.112 caveat). No section rewritten; structure preserved; original experiment + staff review sections untouched.

### Checklist

1. **Read entity body and staff-review findings (Repurpose section, Decision, AC-R1, AC-R2, AC-R3, Implementation Notes, Staff Review).** DONE. Read the full file via two passes (head 295-462 + 462-568); held the staff review section as source of truth.
2. **Surgical Fix #1 — AC-R1 reframing (Hypothesis isolated, Pass, Fail).** DONE. Replaced the Hypothesis isolated line to acknowledge the three-variable confound (section absence + team-mode absence + different test surface) and reframe the priming surface as "team-mode dispatch shape (which includes the standing-teammates section, the Completion Signal block, and team-mode framing)." Updated Pass to "some aspect of team-mode dispatch shape" (with a note that AC-R2 is the follow-up to narrow further) and Fail to "all prompt-shape mitigation is ruled out, regardless of which dispatch element." Test selection and Verify command unchanged per the instruction.
3. **Surgical Fix #2 — Add Outcome Map to Decision (after line 318).** DONE. Inserted new `### Outcome Map` subsection after the out-of-scope paragraph and before `### Acceptance Criteria`. Markdown table with 5 rows: all-three PASS, all-three FAIL, AC-R1 PASS + AC-R2 FAIL (counterintuitive), AC-R3 PASS + others FAIL, AC-R2 PASS + others FAIL — each with Interpretation and Recommended follow-up columns. Closing note: AC-R3 is orthogonal evidence; any AC-R1 FAIL triggers Layer 3 path regardless of other ACs.
4. **Surgical Fix #3 — AC-R2 patch sequencing (Implementation Notes).** DONE. Replaced "git stash or git checkout to revert" with the canonical sequence: `git diff skills/commission/bin/claude-team > /tmp/ac-r2.patch` (capture first), then `git checkout -- skills/commission/bin/claude-team` (clean revert). Noted the captured .patch file is authoritative for the AC-R2 Capture-list "paste patch diff" requirement.
5. **Surgical Fix #4 — AC-R3 patch-leak guard (Pre-step).** DONE. AC-R3's Verify section now opens with a HARD REQUIREMENT Pre-step: `git diff --quiet skills/commission/bin/claude-team` before pytest; non-zero ABORTS AC-R3. Stated as hard requirement, not suggestion, per the dispatch instruction.
6. **Surgical Fix #5 — Un-patched dispatch prompt baseline (Implementation Notes evidence list).** DONE. Added a bullet to the evidence-capture list: capture one un-patched dispatch prompt (from AC-R1 or AC-R3 fo-log) and one patched dispatch prompt (from AC-R2), both in the `## Repurpose Outcome` section as side-by-side excerpts.
7. **Small Fix — AC-R1 Capture list addition (manual fo-log inspection on FAIL).** DONE. AC-R1's Capture list now explicitly requires, for a FAIL, manual `fo-log.jsonl` inspection to identify whether the FO emitted the corresponding tool calls for any state-change claims it makes in text — explained why this is needed (streaming watcher objective milestones don't apply to AC-R1's FO-side failure surface).
8. **Small Fix — AC-R1 worktree applicability.** DONE. Added a note to AC-R1's Capture-list paragraph: AC-R1 also runs from inside the rebased worktree for consistency with AC-R2 and AC-R3 (cheaper repo-root option exists but loses environmental consistency).
9. **Small Fix — AC-R3 sonnet 2.1.112 caveat (Capture list).** DONE. AC-R3's Capture list now explicitly lists `claude --version` output, with the note that a 2.1.112 PASS must be re-confirmed on 2.1.111 before pinning `--model sonnet` as a default-shift recommendation (sonnet behavior on 2.1.112 was not pre-validated in this entity).
10. **Do NOT modify the existing `## Staff Review (repurpose)` section or `## Stage Report (staff review, repurpose)`.** DONE. Both sections (lines 464-568 in the pre-revision file) were not touched. Verified by inspection — only edits were inside AC-R1, the Decision section (Outcome Map insertion), and the Implementation Notes; plus the new appended Repurpose Revision and this Stage Report at the end of the file.
11. **Do NOT modify the original (pre-Repurpose) sections (lines 1-294).** DONE. No edits in lines 1-294. Verified by inspection.
12. **Append a brief `## Repurpose Revision (post-staff-review)` section.** DONE. Section appended after the `## Stage Report (staff review, repurpose)`'s one-line summary, before this Stage Report. Cross-references each fix #1-#5 + the small fixes; one-line confirmation that the Repurpose Decision/AC-R1/AC-R2/AC-R3/Implementation Notes structure is unchanged except for the targeted fixes.
13. **Commit on main with the specified message.** Pending — will run immediately after this report write completes. Working tree was clean at start (verified via `git status` before edits).
14. **Write `## Stage Report (ideation revision, repurpose post-staff-review)` at the very end.** DONE (this section).

### One-line summary for the FO at the gate

Repurpose-ideation revision complete: all 5 surgical fixes (AC-R1 reframing, Outcome Map, AC-R2 sequencing, AC-R3 patch-leak guard, un-patched prompt baseline) plus 3 small fixes folded in surgically; structure of Decision/AC-R1/AC-R2/AC-R3/Implementation Notes preserved; original-experiment and staff-review sections untouched.

## Repurpose Outcome

**Outcome combination: AC-R1 PASS, AC-R2 FAIL, AC-R3 PASS.**

Per the Decision Outcome Map (line 319) this combination is not one of the 5 enumerated rows but composes from two of them:
- AC-R3 PASS gives a clean alternative-model workaround (FO on `--model sonnet` while ensign continues on opus-4-7 — see Caveat below).
- AC-R2 FAIL on top of AC-R1 PASS rules out *section-richness compression* as the priming surface within team-mode dispatch shape — the section's mere presence (or some other team-mode element) is what primes opus-4-7 FO fabrication, not the per-teammate prose body.

**Recommended follow-up (composed):**
1. **Pin `--model sonnet` as the workflow default for FO** (per the Outcome Map's "AC-R3 PASS + others FAIL" row, with the 2.1.111 re-confirmation caveat at AC-R3 Capture line 401). This is the most actionable workaround and does not require changing the dispatch prompt template.
2. **Investigate the section-presence priming token** (per the Outcome Map's "AC-R1 PASS + AC-R2 FAIL" row): since stripping the section's prose did NOT fix opus-4-7 FO behavior but the section was preserved structurally, the priming sits in the section header, the Completion Signal block, or another team-mode element — NOT in the per-teammate Patterns 1-4 prose. A future engineering task should explore: (a) drop the standing-teammates section header/footer entirely when there is one teammate, (b) compress or relocate the Completion Signal block, (c) try a non-team-mode dispatch shape on the same fixture.
3. **Do NOT compress section emission as the engineering follow-up** (which "AC-R2 PASS + others FAIL" would have recommended): AC-R2's empirical result falsifies that path. The minimal one-line variant did not unblock the test; the failure mode shifted from milestone 4 (`SendMessage to echo-agent observed`) to milestone 5 (`ECHO: ping reply received`) — same shape as the high/xhigh effort runs documented in the original Evidence section (line 53), suggesting the failure is in the FO's reply-handling / shutdown-coordination loop, not in priming the ensign to skip the SendMessage.

### Evidence per AC

**AC-R1 — gate-guardrail counterfactual (PASS, after local plumbing patch).**

- Initial run (un-patched test): `pytest tests/test_gate_guardrail.py --runtime claude --model opus --effort low -v` PASSED in 43.08s, but `fo-log.jsonl` showed `"model":"claude-sonnet-4-6"` × 23, ZERO `claude-opus-4-7` stamps. The pytest CLI `--model opus` was silently dropped because `tests/test_gate_guardrail.py:48-53` calls `run_first_officer_streaming(... extra_args=["--max-budget-usd", "1.00"])` with no `--model` flag and the test does not consume the `model` fixture. Same plumbing-bug class as the original AC-3 BROKEN finding (cross-ref Outcome line 92, Follow-up line 122).
- Resolution (captain-authorized as a structural extension of AC-R2's local-patch discipline): apply a minimal local patch to `tests/test_gate_guardrail.py:52` adding `"--model", "opus"` to `extra_args`; capture the patch to `/tmp/ac-r1-test-patch.patch`; re-run; revert.
- Re-run (patched): pytest exit code 0, 52.42s wallclock; `fo-log.jsonl` model stamps: 20 `claude-opus-4-7` occurrences, ZERO sonnet (verbatim grep below).
- `claude --version`: `2.1.112 (Claude Code)` (drift from preferred 2.1.111 acceptable per Test Plan line 408).
- Verbatim grep on `/tmp/ac-r1-patched-fo-log.jsonl`:
  ```
  $ grep -oh 'claude-opus-4-[67]' /tmp/ac-r1-patched-fo-log.jsonl | sort | uniq -c
    20 claude-opus-4-7
  $ grep -oh '"model":"[^"]*"' /tmp/ac-r1-patched-fo-log.jsonl | sort | uniq -c
    19 "model":"claude-opus-4-7"
  $ head -1 /tmp/ac-r1-patched-fo-log.jsonl | grep -oE '"model":"[^"]+"|"claude_code_version":"[^"]+"'
  "model":"claude-opus-4-7"
  "claude_code_version":"2.1.112"
  ```
- Patch revert verified: `git checkout -- tests/test_gate_guardrail.py && git diff --quiet tests/test_gate_guardrail.py` exit 0.
- AC-R1 PATCH (paste of `/tmp/ac-r1-test-patch.patch`):
  ```diff
  diff --git a/tests/test_gate_guardrail.py b/tests/test_gate_guardrail.py
  index 9342e834..32cc0053 100755
  --- a/tests/test_gate_guardrail.py
  +++ b/tests/test_gate_guardrail.py
  @@ -49,7 +49,7 @@ def test_gate_guardrail(test_project, runtime):
               t,
               "Process all tasks through the workflow.",
               agent_id=agent_id,
  -            extra_args=["--max-budget-usd", "1.00"],
  +            extra_args=["--model", "opus", "--max-budget-usd", "1.00"],
           ) as w:
               w.expect(
                   lambda e: entry_contains_text(
  ```
- Interpretation per AC-R1 Pass line 349: PASS implicates "some aspect of team-mode dispatch shape" as the priming surface, narrower than "all prompts" but broader than "the section alone." The gated-pipeline fixture has no `agents:` block (no team mode, no standing-teammates section, no Completion Signal block) and FO+ensign on opus-4-7 + low effort completed cleanly. AC-R2 narrows further within team-mode shape.

**AC-R2 — section-stripped patch on standing-teammate test (FAIL).**

- Verify (un-patched at start): `git diff --quiet skills/commission/bin/claude-team` exit 0.
- Patch applied (replaces loop body at `claude-team:287-301`, drops footer 302-307); patch captured to `/tmp/ac-r2.patch` AFTER apply BEFORE run.
- Run: `pytest tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips --runtime claude --model opus --effort low -v` with `KEEP_TEST_DIR=1`. Exit code 1 (pytest FAILED) in 135.51s wallclock.
- Failure: `test_lib.StepFailure: FO subprocess exited (code=1) before step 'ECHO: ping reply received' matched.` FO subprocess exit code 1 (`error_max_budget_usd: Reached maximum budget ($2)`).
- Milestone timing (verbatim from captured stdout):
  - `[OK] claude-team spawn-standing invoked` (milestone 1, ≤60s budget)
  - `[OK] echo-agent Agent() dispatched` (milestone 2, ≤120s budget)
  - `[OK] ensign dispatch prompt includes standing-teammates section with echo-agent` (milestone 3, prompt assertion)
  - `[OK] SendMessage to echo-agent observed` (milestone 4, ≤240s budget — passed!)
  - **FAIL at milestone 5** `ECHO: ping reply received` (≤240s budget) — never landed in parent fo-log before FO subprocess exited from budget exhaustion.
- `fo-log.jsonl` model stamps: 48 `claude-opus-4-7`, 2 `sonnet`. FO + ensign both ran on opus-4-7 as required.
- Verbatim grep on `/tmp/ac-r2-fo-log.jsonl`:
  ```
  $ grep -oh 'claude-opus-4-[67]' /tmp/ac-r2-fo-log.jsonl | sort | uniq -c
    48 claude-opus-4-7
  $ grep -oh '"model":"[^"]*"' /tmp/ac-r2-fo-log.jsonl | sort | uniq -c
    44 "model":"claude-opus-4-7"
     2 "model":"sonnet"
  ```
- AC-R2 PATCH (paste of `/tmp/ac-r2.patch`):
  ```diff
  diff --git a/skills/commission/bin/claude-team b/skills/commission/bin/claude-team
  index faffbcc6..88e71d3c 100755
  --- a/skills/commission/bin/claude-team
  +++ b/skills/commission/bin/claude-team
  @@ -285,26 +285,7 @@ def cmd_build(args):
               '',
           ]
           for name, description, mod_path in standing_teammates:
  -            desc = description or 'standing teammate'
  -            usage_body = _parse_routing_usage_body(mod_path) if mod_path else None
  -            if usage_body:
  -                lines.append(f'- **{name}** ({desc})')
  -                for body_line in usage_body.split('\n'):
  -                    if body_line:
  -                        lines.append(f'  {body_line}')
  -                    else:
  -                        lines.append('')
  -            else:
  -                lines.append(
  -                    f'- **{name}** ({desc}): SendMessage with the relevant input '
  -                    f'shape; reply format per the mod.'
  -                )
  -        lines.append('')
  -        lines.append(
  -            'Full routing contract: see '
  -            '`skills/first-officer/references/first-officer-shared-core.md` '
  -            '`## Standing Teammates`.'
  -        )
  +            lines.append(f'- {name} is available; SendMessage to it.')
           prompt_parts.append('\n'.join(lines))
  
       # 10. Completion signal (conditional, team mode only)
  ```
- PATCHED dispatch prompt's standing-teammates section (extracted from AC-R2's `fo-log.jsonl` Agent tool_use input for `spacedock-ensign-001-echo-roundtrip-work`):
  ```
  ### Standing teammates available in your team

  The FO has spawned these standing teammates; you MAY route to them via SendMessage. Best-effort, non-blocking, 2-minute timeout; proceed with un-polished/un-reviewed content if no reply.

  - echo-agent is available; SendMessage to it.
  ```
- Patch revert verified after AC-R2 completion: `git checkout -- skills/commission/bin/claude-team && git diff --quiet -- tests/test_gate_guardrail.py skills/commission/bin/claude-team` exit 0 (BOTH files clean per the extended pre-step the team-lead authorized for AC-R2/R3).
- Interpretation per AC-R2 Fail line 374: section presence (not its richness) is the priming, OR the priming sits elsewhere in the prompt entirely (Completion Signal block, header, etc.). Engineering follow-up needs to investigate further before settling on a fix. Notably the failure mode SHIFTED — milestone 4 (the canonical original-AC-1 hallucination point) PASSED with the patched section, but milestone 5 (which fails on high/xhigh effort per Evidence line 53) FAILED. The reply-handling / shutdown-coordination loop is the surface that broke.

**AC-R3 — different ensign model: sonnet (PASS, with caveat that ensign sub-agent still ran on opus-4-7).**

- HARD pre-step (per AC-R3 Verify line 384): `git diff --quiet -- tests/test_gate_guardrail.py skills/commission/bin/claude-team` exit 0 — both files clean before AC-R3 ran. (Extended check per team-lead's instruction since AC-R1 also patched a file.)
- Run: `pytest tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips --runtime claude --model sonnet --effort low -v` with `KEEP_TEST_DIR=1`. Exit code 0 (pytest PASSED) in 158.30s wallclock. All 5 milestones fired.
- `fo-log.jsonl` model stamps: 74 `claude-sonnet-4-6` (FO process), 7 `claude-opus-4-7` (only inside teammate-result `modelUsage` rollups; the ensign Agent() dispatched WITHOUT explicit `model` field which defaulted to opus-4-7 in the spawning context).
- Verbatim grep on `/tmp/ac-r3-fo-log.jsonl`:
  ```
  $ grep -oh 'claude-opus-4-[67]' /tmp/ac-r3-fo-log.jsonl | sort | uniq -c
     7 claude-opus-4-7
  $ grep -oh '"model":"[^"]*"' /tmp/ac-r3-fo-log.jsonl | sort | uniq -c
     1 "model":"claude-opus-4-7"
    74 "model":"claude-sonnet-4-6"
     3 "model":"sonnet"
  $ head -1 /tmp/ac-r3-fo-log.jsonl | grep -oE '"model":"[^"]+"|"claude_code_version":"[^"]+"'
  "model":"claude-sonnet-4-6"
  "claude_code_version":"2.1.112"
  ```
- Per-session model breakdown (Python json walk over `fo-log.jsonl`): the parent FO session `9838534a-5c44-4c36-9254-fcbfcabcbb00` only emitted `claude-sonnet-4-6` assistant messages. The opus-4-7 stamps appear only in `tool_use_result.prompt.model` and `result.modelUsage` rollups for spawned teammates.
- Agent dispatch model field inspection (Python json walk for `tool_use.name == "Agent"`):
  - `echo-agent` Agent() dispatch: `name=echo-agent model=sonnet` (FO propagated `--model sonnet` to standing teammate)
  - Ensign Agent() dispatch: `name=spacedock-ensign-001-echo-roundtrip-work model=(none)` — no explicit model on Agent input, defaults to opus-4-7 per the spawning context (ensign agent type default).
- **CAVEAT — AC-R3 did NOT actually test sonnet on the ensign hallucination surface.** AC-R3's hypothesis (Decision line 379) was "this is opus-4-7-specific calibration; sonnet does not exhibit the pattern." The PASS shows that **when the FO is on sonnet** (and dispatches an opus-4-7 ensign that emits SendMessage and waits for ECHO reply), the test passes. The ensign itself was still opus-4-7. AC-R2 ran the same test with FO on opus-4-7 (and same default opus-4-7 ensign) and FAILED at the ECHO-reply milestone. This is strong evidence the regression is **FO-side, not ensign-side**: when the FO process is opus-4-7, the reply from the (opus-4-7) ensign-via-echo-agent fails to land in the parent fo-log within budget; when the FO is sonnet, the same reply (from the same opus-4-7 ensign) lands cleanly. This is consistent with the original Evidence line 35 ("FO itself on opus-4-7: may skip status --set calls...").
- 2.1.112 caveat per AC-R3 Capture line 401: this run was on Claude Code `2.1.112`. Sonnet behavior on 2.1.112 was not pre-validated in this entity. A PASS should be re-confirmed on 2.1.111 before pinning `--model sonnet` as a default-shift recommendation.
- Interpretation per AC-R3 Pass line 399, with the FO/ensign-model caveat: `--model sonnet` for the FO is a viable workaround independent of any prompt-shape fix. The captain can pin `--model sonnet` in workflow defaults (FO-stage default) as a safer default than opus until upstream is fixed; the regression is not "opus-4-7-specific calibration" in the model sense (since the ensign was still opus-4-7), it is "opus-4-7 FO process specifically can't keep teammate replies in its parent stream within budget."

### Side-by-side: un-patched vs patched dispatch prompt (standing-teammates section)

Both excerpts extracted from the actual `Agent` tool_use input the FO sent to the spawned ensign (`spacedock-ensign-001-echo-roundtrip-work`).

**Un-patched (AC-R3 baseline, current main `claude-team` source):**
```
### Standing teammates available in your team

The FO has spawned these standing teammates; you MAY route to them via SendMessage. Best-effort, non-blocking, 2-minute timeout; proceed with un-polished/un-reviewed content if no reply.

- **echo-agent** (Trivial standing teammate used by the standing-teammate live E2E): SendMessage with the relevant input shape; reply format per the mod.

Full routing contract: see `skills/first-officer/references/first-officer-shared-core.md` `## Standing Teammates`.
```

**Patched (AC-R2, after applying the minimal one-line variant to `claude-team`):**
```
### Standing teammates available in your team

The FO has spawned these standing teammates; you MAY route to them via SendMessage. Best-effort, non-blocking, 2-minute timeout; proceed with un-polished/un-reviewed content if no reply.

- echo-agent is available; SendMessage to it.
```

Diff: the patched variant drops the per-teammate parenthetical description, the `**bold**` markup, the per-teammate routing-usage body, and the trailing "Full routing contract" pointer line. Section header and intro paragraph preserved. With the standing-teammate count = 1, the visible bullet shrank from a markdown-formatted line with description and routing-usage hint to a single bare-prose line.

### Patches reverted before subsequent ACs ran (consolidated)

Per the team-lead's extended pre-step authorization, the AC-R2/R3 hard pre-step was extended to check BOTH files. Verified clean:

- AC-R1 → AC-R2 transition: `git diff --quiet tests/test_gate_guardrail.py` exit 0 (AC-R1 patch reverted).
- AC-R2 → AC-R3 transition: `git diff --quiet -- tests/test_gate_guardrail.py skills/commission/bin/claude-team` exit 0 (BOTH patches reverted).

Both patches were experimental-only local workarounds; neither is committed on the branch and neither will be pushed.

### Follow-up needed

**(a) Persistent plumbing fix for `tests/test_gate_guardrail.py` — discovered during AC-R1.**

- **Proposed task title**: "Wire pytest's `--model` fixture into `tests/test_gate_guardrail.py`'s `extra_args` (analogue of #179's fix for `test_standing_teammate_spawn.py`)."
- **Source**: 2026-04-17 #177 AC-R1 implementation — pytest's `--model opus` was silently dropped because `tests/test_gate_guardrail.py:48-53` does not consume the `model` fixture. Local patch (`/tmp/ac-r1-test-patch.patch`, captured above) is a temporary experimental workaround; the persistent fix is to wire the existing pytest `--model` fixture from `tests/conftest.py:106-107` into the test's `extra_args` list, mirroring #179's pattern for `test_standing_teammate_spawn.py:72`.
- **One-line spec**: At `tests/test_gate_guardrail.py:30`, add `model` to the function signature; at line 52 replace `extra_args=["--max-budget-usd", "1.00"]` with `extra_args=["--model", model, "--max-budget-usd", "1.00"]`. Add `model` fixture import at top if needed.
- **Why now**: leaving this unfixed means future AC-R1-style experiments on this test silently run on the wrong model (sonnet by default). The plumbing-bug class is exactly what motivated #180; #179's fix landed for the standing-teammate test but missed the gate-guardrail test.

**(b) Layer 3 FO-side post-completion verification — for the AC-R2 FAIL outcome.**

- **Proposed task title**: "Investigate FO reply-handling / shutdown-coordination loop on opus-4-7 — why teammate replies don't land in parent fo-log within budget."
- **Source**: 2026-04-17 #177 AC-R2 — patching the standing-teammates section to one-line-per-teammate did NOT fix opus-4-7 FO behavior. The failure mode SHIFTED from milestone 4 (`SendMessage to echo-agent observed`, the canonical original-AC-1 hallucination point) — which now PASSED — to milestone 5 (`ECHO: ping reply received`), which fails the same way it does on high/xhigh effort runs documented in the original Evidence section (line 53). Combined with AC-R3 PASS (FO on sonnet → reply lands), this strongly suggests the regression is **FO-side reply-folding**, not ensign-side hallucination. The Layer 2 prompt-shape mitigation hypothesis is falsified for section-richness; either drop a different team-mode element (Completion Signal block, section header) OR move to Layer 3 (FO post-completion verification, e.g., the FO grep-checks the stream for the reply pattern before declaring teammate-routing complete).
- **One-line spec**: scoping needs ideation; either: (i) downstream prompt-shape experiments (drop section header entirely when teammate count = 1, drop Completion Signal block, etc.), or (ii) FO-side stream-folding investigation (why opus-4-7 FO doesn't see teammate replies in its parent stream that sonnet FO sees on the same fixture).

**(c) Sonnet-default workflow pin — for the AC-R3 PASS outcome (with 2.1.112 caveat).**

- **Proposed task title**: "Pin `--model sonnet` as workflow FO default in spacedock workflow scaffolding (with 2.1.111 re-confirmation gate)."
- **Source**: 2026-04-17 #177 AC-R3 — FO on sonnet drove the standing-teammate test to terminal completion in 158s while FO on opus-4-7 failed at milestone 5 within 135s. The ensign sub-agent ran on opus-4-7 in both cases; only the FO model varied. This is a clean workaround independent of any prompt-shape fix.
- **Pre-condition**: re-run AC-R3 on Claude Code 2.1.111 to confirm the PASS does not depend on 2.1.112 behavior (per AC-R3 Capture caveat at line 401). Sonnet behavior on 2.1.112 was not pre-validated in this entity.
- **One-line spec**: change the `stages.defaults.model` for FO-driving stages in the spacedock workflow templates from default-opus to `sonnet`; document the regression context with a cross-reference to #177; add a TODO to re-evaluate when the upstream opus-4-7 FO regression is fixed.

FO will file these three follow-ups after captain triage; this implementation does not file them.

## Stage Report (repurpose implementation)

### Summary

Three Layer 2 mitigation experiments executed locally on Claude Code 2.1.112 + opus-4-7 / sonnet. Outcome: **AC-R1 PASS, AC-R2 FAIL, AC-R3 PASS** (with caveat that AC-R3 ensign sub-agent still ran on opus-4-7 — only the FO process varied). AC-R1 required a captain-authorized minimal local patch on `tests/test_gate_guardrail.py:52` to wire `--model opus` (same plumbing-bug class as the original AC-3 BROKEN finding). AC-R2 falsified the section-richness compression hypothesis; the failure shifted from milestone 4 (canonical hallucination point) to milestone 5 (reply-folding), matching the high/xhigh-effort failure mode in the original Evidence line 53. AC-R3 passed cleanly with FO on sonnet, providing a viable workaround. Three follow-ups described in `### Follow-up needed`: (a) persistent plumbing fix for gate-guardrail's `extra_args`, (b) Layer 3 / different-element prompt-shape investigation (since Layer 2 section compression is falsified), (c) sonnet workflow-default pin (with 2.1.111 re-confirmation gate).

### Checklist

1. **Read entity body, focus on Repurpose section (line 296+).** DONE. Read the full file (612 lines after the worktree was reset to main HEAD `59a0db41`); focused on Decision (306-318), Outcome Map (319-331), AC-R1/R2/R3 (333-401), Test Plan (403-410), Implementation Notes (412-461). Also confirmed the Repurpose Revision (570-608) was already folded in.

2. **Verify worktree HEAD vs main + capture HEAD SHA.** DONE — but with an upfront BLOCKER. The worktree at dispatch start was at the OLD validation tip `09b96f3d` (file 224 lines, no Repurpose section). The dispatch claimed the worktree was rebased onto current main, but `git log --oneline HEAD..main` showed 11 commits not in worktree HEAD including the entire Repurpose ideation chain (`bafd84c8`/`3659b07f`/`79dab33a`) and the advance commit `59a0db41`. Surfaced to team-lead before any other action; team-lead authorized `git reset --hard main`. Post-reset: `git rev-parse HEAD` = `59a0db413d91f9cb0a2121ee56f4dbd9ab2c8ef6` (matches main HEAD). `wc -l docs/plans/opus-4-7-ensign-hallucination-scope.md` = 612. `grep -n '^## Repurpose'` = lines 296 + 570.

3. **Capture local Claude Code version.** DONE. `claude --version` = `2.1.112 (Claude Code)`. Drift from preferred 2.1.111 acknowledged per Test Plan line 408 + AC-R3 caveat at line 401 (which surfaces in Follow-up (c) above as the 2.1.111 re-confirmation gate).

4. **AC-R1 — Counterfactual on a non-routing test.** DONE — but with a SECOND BLOCKER. Initial run with `--runtime claude --model opus --effort low` PASSED (43.08s) but `fo-log.jsonl` showed 23 sonnet stamps + zero opus stamps — pytest's `--model opus` was silently dropped because the test does not consume the model fixture. Surfaced to team-lead with three resolution options; team-lead authorized Option 1 (apply minimal local patch to `tests/test_gate_guardrail.py:52`, same discipline as AC-R2's patch on claude-team). Re-ran patched: pytest exit 0, 52.42s, 20 opus-4-7 stamps confirmed. Patch contents at /tmp/ac-r1-test-patch.patch and pasted above in Outcome. Patch reverted after AC-R1 with `git checkout --` and `git diff --quiet` exit 0 verified.

5. **AC-R2 — Section-stripped variant.** DONE. Pre-run captured `git diff` exit 0. Patch applied to `claude-team:287-308` (replace loop body with one-liner, drop footer); `git diff > /tmp/ac-r2.patch` AFTER apply BEFORE run. Run: pytest exit 1 in 135.51s. Failure: `StepFailure: FO subprocess exited (code=1) before step 'ECHO: ping reply received' matched.` Milestones 1-4 PASSED; milestone 5 failed (ECHO reply never landed in parent fo-log; FO hit `error_max_budget_usd: Reached maximum budget ($2)`). 48 opus-4-7 stamps in fo-log. Patch contents pasted above. Patched standing-teammates section extracted from `Agent` tool_use input pasted above.

6. **AC-R2 → AC-R3 patch revert.** DONE. `git checkout -- skills/commission/bin/claude-team` then `git diff --quiet -- tests/test_gate_guardrail.py skills/commission/bin/claude-team` exit 0 — BOTH files clean (per the team-lead's extended pre-step covering AC-R1's patch as well as AC-R2's).

7. **AC-R3 — Different ensign model.** DONE. Hard pre-step `git diff --quiet -- tests/test_gate_guardrail.py skills/commission/bin/claude-team` exit 0 confirmed before pytest invocation. Run: pytest exit 0 in 158.30s. All 5 milestones fired. FO on sonnet (74 sonnet stamps); ensign Agent() dispatched WITHOUT explicit model field, defaulted to opus-4-7 per spawning context. Caveat surfaced in Outcome: AC-R3 PASS shows that FO-on-sonnet drives the test to completion even when the ensign sub-agent is still opus-4-7 — strong evidence the regression is FO-side, not ensign-side. 2.1.112 caveat per AC-R3 Capture line 401 noted in Follow-up (c).

8. **Capture un-patched dispatch prompt baseline.** DONE. Extracted the un-patched `### Standing teammates available in your team` section from AC-R3's `fo-log.jsonl` (the spawned ensign teammate's dispatch prompt). Pasted side-by-side with AC-R2's patched section above in Outcome's "Side-by-side" subsection. Dispatched ensign name: `spacedock-ensign-001-echo-roundtrip-work`; model in spawn record: `claude-opus-4-7`.

9. **Determine outcome combination per Decision Outcome Map.** DONE. Outcome **AC-R1 PASS, AC-R2 FAIL, AC-R3 PASS** is not one of the 5 enumerated rows but composes from "AC-R3 PASS + others FAIL" + "AC-R1 PASS + AC-R2 FAIL". Recommended follow-up composed from both: pin `--model sonnet` (R3) AND investigate priming token outside per-teammate prose (R1+R2), AND do NOT pursue section-richness compression (R2 falsifies that). Detailed in Outcome's `### Recommended follow-up (composed)` subsection.

10. **Append `## Repurpose Outcome` section.** DONE. Section appended after the Repurpose Revision Stage Report's one-line summary (line 612). Contains: outcome combination, evidence per AC (verbatim grep, exit codes, milestone timing, claude --version), AC-R1 + AC-R2 patch diffs pasted, side-by-side patched-vs-unpatched prompt excerpts, `### Follow-up needed` subsection with 3 follow-ups (a/b/c).

11. **Follow-up needed subsection.** DONE. Three follow-ups described in `### Follow-up needed`: (a) persistent plumbing fix for `tests/test_gate_guardrail.py:52` extra_args (analogue of #179 for the gate-guardrail test); (b) Layer 3 FO-side reply-handling investigation (since AC-R2 falsified Layer 2 section-richness compression); (c) sonnet workflow-default pin (with 2.1.111 re-confirmation gate per AC-R3 caveat). FO files them after captain triage; this implementation does not file them.

12. **Commit on worktree branch.** Pending — will run immediately after this report write completes. Working tree was clean post-AC-R3 (verified `git status: nothing to commit`). Commit message will be `experiment: #177 ran AC-R1/R2/R3 — outcome AC-R1 PASS / AC-R2 FAIL / AC-R3 PASS, recommendation pin --model sonnet (R3) + investigate priming surface outside per-teammate prose (R1+R2 composed)`.

13. **Stage Report (repurpose implementation).** DONE (this section).

### One-line summary for the validator

Layer 2 experiment outcome AC-R1 PASS / AC-R2 FAIL / AC-R3 PASS — section-richness compression is NOT the priming surface (R2 falsifies); FO-on-sonnet is a viable workaround (R3, with 2.1.112 caveat); priming sits in some other team-mode element (R1+R2 composed). Two captain-authorized local patches applied and reverted (AC-R1 test plumbing, AC-R2 claude-team source); both files verified clean post-revert. Three follow-ups described for FO triage: (a) persistent plumbing fix for gate-guardrail's extra_args, (b) Layer 3 FO reply-handling investigation, (c) sonnet workflow-default pin.

## Stage Report (validation, repurpose-experiment)

### Summary

Validation of #177's repurpose-experiment results (AC-R1 PASS / AC-R2 FAIL / AC-R3 PASS, composed outside the pre-enumerated Outcome Map). Audited evidence integrity, patch capture-and-revert discipline, and the load-bearing "FO-side reply-folding" diagnosis. **Implementation work: PASSED** — three ACs honestly executed, both local patches captured before revert, both reverts verified clean, evidence chain fully traceable to /tmp artifacts and the worktree's current state. **Outcome composition: SOUND-BUT-OVERREACHING** — the FO-vs-ensign locus claim (AC-R3 pivot) is well-supported by the model-stamp evidence, but the more specific "FO reply-folding" framing narrows past what the evidence strictly proves; alternative FO-side mechanisms (budget exhaustion from per-token verbosity, general FO orchestration bug) remain consistent with the same data. Recommend the captain treat Follow-up (b)'s scope as "FO-side opus-4-7 regression — mechanism TBD" rather than committing to the reply-folding hypothesis at ideation time.

### Checklist

1. **Read entity body, focus on Repurpose Outcome (line 614+) and Stage Report (repurpose implementation) (line 815+).** DONE. Read all 851 lines via two-pass (lines 300-700, 700-851). Cross-referenced Decision Outcome Map at line 319 and original AC-R1/R2/R3 spec at lines 333-401. Confirmed the implementer's outcome combination "AC-R1 PASS, AC-R2 FAIL, AC-R3 PASS" is NOT one of the 5 enumerated rows — the implementer composed the "AC-R3 PASS + others FAIL" row (for follow-up #c) with the "AC-R1 PASS + AC-R2 FAIL counterintuitive" row (for follow-up #b), explicitly noted at entity line 618.

2. **Cross-check AC-R1's evidence chain (plumbing-bug claim + patch revert + model stamps).** DONE.
   - Source-side plumbing bug confirmed by reading `tests/test_gate_guardrail.py:30-60`. Function signature is `def test_gate_guardrail(test_project, runtime):` — no `model` fixture. Line 52 shows `extra_args=["--max-budget-usd", "1.00"]` — no `--model`. The pytest CLI `--model opus` is therefore silently dropped, exactly as the entity claims at line 631. Same plumbing-bug class as #179 (which fixed `test_standing_teammate_spawn.py`) — the gate-guardrail test was simply missed by that fix.
   - Patch captured at `/tmp/ac-r1-test-patch.patch` (560 bytes, mtime 21:09); contents match the entity's verbatim paste at lines 647-661. The patch adds exactly `"--model", "opus", ` to the `extra_args` list at line 52.
   - Patch revert verified: `git -C ... diff --quiet tests/test_gate_guardrail.py` exit 0.
   - Model-stamp re-grep on `/tmp/ac-r1-patched-fo-log.jsonl` (132569 bytes, mtime 21:10): `20 claude-opus-4-7` (matches entity line 638), `19 "model":"claude-opus-4-7"` (matches entity line 640), `claude_code_version: 2.1.112` (matches entity line 643). ZERO sonnet stamps in the patched run. Plumbing-fix succeeded.

3. **Cross-check AC-R2's evidence chain (patch correctness + revert + failure mode shift).** DONE.
   - Patch captured at `/tmp/ac-r2.patch` (1382 bytes, mtime 21:10); contents match the entity's verbatim paste at lines 686-718.
   - Source-side pre-image confirmed by reading `skills/commission/bin/claude-team:282-310`. The current source matches the patch's `-` lines exactly (loop body at 287-301, footer at 302-307). Patch reverted cleanly.
   - Patch revert verified: `git -C ... diff --quiet skills/commission/bin/claude-team` exit 0.
   - Model-stamp re-grep on `/tmp/ac-r2-fo-log.jsonl` (186561 bytes, mtime 21:13): `48 claude-opus-4-7`, `44 "model":"claude-opus-4-7"`, `2 "model":"sonnet"` — all match entity lines 680-684. FO + ensign both ran on opus-4-7.
   - Failure-mode shift claim (M4 PASSED, M5 FAILED): I cannot independently re-verify the streaming-watcher milestone breakdown from the fo-log alone (the watcher's milestone labels are stdout-only). However, the parent fo-log contains only 2 SendMessage tool_use entries, both shutdown_request — confirming the test's actual ECHO-ping SendMessage happens inside the ensign sub-agent context (not visible in the parent fo-log). I accept the entity's milestone breakdown at lines 670-675 as the authoritative artifact for milestone status. The `error_max_budget_usd: Reached maximum budget ($2)` failure cause at entity line 669 is consistent with M5-budget-exhaustion shape.

4. **Cross-check AC-R3's evidence chain (FO=sonnet, ensign=opus-4-7, test PASSED).** DONE — this is the load-bearing diagnostic claim.
   - Model-stamp re-grep on `/tmp/ac-r3-fo-log.jsonl` (244336 bytes, mtime 21:16): `7 claude-opus-4-7`, `1 "model":"claude-opus-4-7"`, `74 "model":"claude-sonnet-4-6"`, `3 "model":"sonnet"` — all match entity lines 738-743.
   - Re-walked the json for `tool_use.name == "Agent"` independently (Python script). Found 3 Agent dispatches:
     - `subagent_type='general-purpose' model='sonnet' desc=''` (likely a top-level FO call)
     - `subagent_type='general-purpose' model='sonnet' desc='Spawn echo-agent standing teammate'` (echo-agent dispatch — model explicitly `sonnet`, propagated from FO's `--model sonnet`)
     - `subagent_type='spacedock:ensign' model='(none)' desc='Echo-agent roundtrip live check: work'` (ensign dispatch — NO explicit model field on Agent input)
   - This independently confirms the entity's claim at lines 750-751 verbatim. The ensign Agent() call had no `model` field; per spawning context default it ran on opus-4-7 (corroborated by the 7 opus-4-7 stamps in the fo-log appearing in tool_use_result rollups).
   - Patch revert pre-step claim verified: both `tests/test_gate_guardrail.py` and `skills/commission/bin/claude-team` are clean in the current worktree (`git diff --quiet` exit 0 for both).

5. **Verify the outcome interpretation logic (composition soundness).** DONE — but with a critical caveat (see Recommendation b).
   - Composition logic: AC-R3 PASS shows that swapping FO-opus-4-7 for FO-sonnet (with ensign held constant on opus-4-7) drives the same test that AC-R2 failed to terminal completion. The implementer concludes the regression follows the FO process, not the ensign sub-agent. **This locus claim is well-supported.**
   - The implementer further narrows to "FO-side reply-folding" (entity line 803): the FO opus-4-7 process fails to fold the ensign's reply into its parent stream within budget. This is the load-bearing follow-up scoping claim.
   - **Alternative explanations consistent with the same evidence:**
     - **(i) FO-opus verbosity / budget-management bug.** AC-R2 hit `error_max_budget_usd: Reached maximum budget ($2)` at line 669. AC-R3 sonnet completed in 158s vs AC-R2 opus failing at 135s within the same $2 budget. Opus-4-7 may simply be more token-verbose at the FO orchestration role, exhausting budget before the reply window opens. This would manifest as M5 failure but isn't "reply-folding" — it's "FO can't get to M5 before running out of money." Mitigation would differ (raise budget? compress FO prompts? cap FO retries?) from a true reply-folding fix.
     - **(ii) AC-R2 patch confound.** AC-R2 changed both the section AND ran on opus-4-7. The M4→M5 failure shift could be the section change *helping with M4 priming* while opus-4-7 FO continues to fail at M5 for an unrelated reason. The implementer treats the M4 pass as evidence the section-richness hypothesis is partially-vindicated-but-not-sufficient (entity line 729). Reasonable, but means the M5 failure is on a *different mechanism* than the original AC-1 hallucination.
     - **(iii) FO-opus general orchestration regression.** The "reply-folding" specificity may be premature — opus-4-7 FO could be regressing on multiple coordination steps; M5 happens to be the visible failure on this fixture. Layer 3 ideation should test hypothesis-i and -iii alongside reply-folding.
   - The composition is **plausible and parsimonious** (Occam's-razor reading of the data), but the "reply-folding" specificity narrows past what the data strictly proves. The locus claim (FO-side, not ensign-side) is solid; the mechanism claim (reply-folding specifically) is speculative.

6. **Verify the three Follow-up entries.** DONE.
   - **(a) `tests/test_gate_guardrail.py` plumbing fix (entity line 793-798).** Well-scoped. Cites the specific source location (`tests/test_gate_guardrail.py:30, 52`), the analogue (#179 fix for the standing-teammate test), and a one-line spec (add `model` to function signature; replace `extra_args` line). Source attribution: "AC-R1 implementation" — precise. **Confirmed clean.**
   - **(b) Layer 3 FO investigation (entity line 800-804).** Explicitly scoped to ideation per the spec — this is correct because the mechanism is uncertain (see #5 above). The one-line spec acknowledges two possible directions (downstream prompt-shape experiments vs FO-side stream-folding investigation). Source attribution: "AC-R2 — patching the standing-teammates section to one-line-per-teammate did NOT fix opus-4-7 FO behavior." Precise. **Recommend the FO ideation broaden the framing from "reply-folding" to "FO-side opus-4-7 regression — mechanism TBD" to admit alternatives (i)/(iii) above. Otherwise scope-clean.**
   - **(c) Sonnet pin (entity line 806-811).** Has a sensible pre-condition (re-run AC-R3 on 2.1.111 before pinning). Source attribution: "AC-R3 — FO on sonnet drove the standing-teammate test to terminal completion in 158s while FO on opus-4-7 failed at milestone 5 within 135s." Precise. **Confirmed clean.**

7. **Verify the implementation work was honestly executed.** DONE.
   - **(a) All three ACs attempted.** YES — entity sections at lines 629-754 cover all three with full evidence.
   - **(b) AC-R1 retry-after-plumbing-discovery handled correctly.** YES — initial run had a contamination (sonnet stamps in fo-log despite `--model opus` on CLI). Implementer surfaced the issue to team-lead, got authorization for a captain-authorized local patch (extending AC-R2's local-patch discipline to AC-R1), captured the patch to `/tmp/ac-r1-test-patch.patch`, ran patched, captured 20 opus-4-7 stamps, reverted cleanly. Honest handling of the contamination — did not pretend the initial run was valid.
   - **(c) Patches captured before / around apply.** YES — both `/tmp/ac-r1-test-patch.patch` and `/tmp/ac-r2.patch` exist on disk and match the verbatim paste in the entity. The entity also pastes both diffs inline (lines 647-661, 686-718) so the audit trail survives even if /tmp is cleared.
   - **(d) Patches reverted between ACs.** YES — verified via `git diff --quiet` on both files in the current worktree (exit 0 for both). The extended pre-step (check BOTH files before AC-R3) was applied — entity line 733 documents this.
   - **(e) Outcome section + Stage Report present with verbatim evidence.** YES — `## Repurpose Outcome` at line 614 and `## Stage Report (repurpose implementation)` at line 815 both present, with verbatim grep output, exit codes, milestone counts, model stamps, patch diffs, and side-by-side prompt excerpts.

8. **Two recommendations.**
   - **(a) IMPLEMENTATION WORK: PASSED.** The experiment was correctly executed and honestly reported. The implementer surfaced two blockers (worktree-state mismatch at start, AC-R1 plumbing bug) before pressing on, captured both patches before revert, verified both reverts, and produced a complete evidence chain that re-validates from /tmp artifacts and the worktree's current source. The outcome combination (AC-R1 PASS / AC-R2 FAIL / AC-R3 PASS) was honestly composed from two Outcome Map rows rather than force-fit into a single row, with the composition rationale stated explicitly at entity line 618.
   - **(b) OUTCOME COMPOSITION: SOUND-BUT-OVERREACHING.** The locus claim ("regression is FO-side, not ensign-side") is solid — AC-R3's design with FO=sonnet+ensign=opus-4-7 PASSING vs AC-R2's FO=opus-4-7+ensign=opus-4-7 FAILING isolates the FO model as the variable that flips the test. The mechanism claim ("FO-side reply-folding") is plausible but narrows past the evidence; alternative FO-side mechanisms (budget-management/verbosity, general orchestration regression) remain consistent with the same data. The captain should act on the locus claim (Follow-up c sonnet pin is well-justified by AC-R3 alone) but treat Follow-up b's "reply-folding" framing as one hypothesis among several when scoping the Layer 3 ideation. The composition combining "AC-R3 PASS + others FAIL" with "AC-R1 PASS + AC-R2 FAIL counterintuitive" is internally consistent with both rows' interpretations and produces a defensible 3-follow-up recommendation set.

### One-line verdict for the captain

Implementation PASSED (three ACs cleanly executed, two patches captured-and-reverted, evidence re-verifiable from /tmp); outcome composition is SOUND on locus (regression is FO-side per AC-R3) but OVERREACHING on mechanism ("reply-folding" narrows past evidence — alternative FO-side mechanisms remain in play). Follow-up (a) plumbing-fix and (c) sonnet-pin are ready to file as-is; Follow-up (b)'s ideation should broaden framing from "reply-folding" to "FO-side opus-4-7 regression — mechanism TBD" before scoping.

## Cycle-3 sanity-check

**Outcome combination: AC-R1 PASS, AC-R2 PASS, AC-R3 PASS.** R1 and R3 match cycle-2; R2 **DIVERGED** (cycle-2 FAIL → cycle-3 PASS). Three live experiments re-run on Claude Code 2.1.121 + opus-4-7 / sonnet against current main HEAD `465d4ffc`, total ~9 min wallclock + opus tokens.

### Environmental drift since cycle-2

- **Claude Code: 2.1.112 → 2.1.121.** Nine patch versions of drift. Cycle-2's preferred 2.1.111 is now two minor versions behind; sonnet behavior on 2.1.121 was not pre-validated in cycle-2.
- **Test plumbing: `tests/test_gate_guardrail.py:51` now wires `--model` and `--effort` from pytest fixtures into `extra_args`.** Cycle-2's AC-R1 needed a captain-authorized local patch to add this; main now has it (likely #179's analogue or a sibling fix). The cycle-3 AC-R1 ran without any patch — direct invocation via pytest CLI flags works.
- **PR #181 (stage-worktree-stickiness) merged on 2026-04-30.** Tonight's opus-tier post-merge failures (`test_dispatch_completion_signal` + `test_feedback_keepalive` failing opus-only on https://github.com/clkao/spacedock/actions/runs/25206509179/job/73908102321) confirm the FO-on-opus failure mode is still active, but on different tests than the cycle-2/cycle-3 standing-teammate fixture.
- **`enumerate_alive_standing_teammates` → `enumerate_declared_standing_teammates`** (commit `0c60611b` post-cycle-2). Lazy spawn (#107) merged. The patch site for AC-R2 moved from cycle-2's `claude-team:287-301` to current `claude-team:410-430` but the loop body shape is structurally identical; the AC-R2 patch shape applied cleanly with no rework.

### Evidence per AC

**AC-R1 — gate-guardrail counterfactual (PASS, no patch needed).**

- Run: `unset CLAUDECODE && KEEP_TEST_DIR=1 uv run pytest tests/test_gate_guardrail.py --runtime claude --model opus --effort low -v`. Exit code 0, 44.33s wallclock.
- `claude --version`: `2.1.121 (Claude Code)`.
- Model stamps from `/tmp/cycle3/ac-r1-fo-log.jsonl`:
  ```
  $ grep -oh 'claude-opus-4-[67]' /tmp/cycle3/ac-r1-fo-log.jsonl | sort | uniq -c
    20 claude-opus-4-7
  $ grep -oh '"model":"[^"]*"' /tmp/cycle3/ac-r1-fo-log.jsonl | sort | uniq -c
    19 "model":"claude-opus-4-7"
  $ head -1 /tmp/cycle3/ac-r1-fo-log.jsonl | grep -oE '"claude_code_version":"[^"]+"'
  "claude_code_version":"2.1.121"
  ```
- Plumbing-patch verification: `tests/test_gate_guardrail.py:51` already reads `extra_args=["--model", model, "--effort", effort, "--max-budget-usd", "1.00"]` on current main. No local patch applied; no patch revert needed. The cycle-2 plumbing follow-up (Follow-up #a) has effectively landed.
- Comparison to cycle-2: identical PASS verdict (cycle-2: 52.42s, 20 opus stamps; cycle-3: 44.33s, 20 opus stamps). Wallclock 8s faster — within noise.
- Interpretation: confirms cycle-2's R1 PASS — gate-guardrail (no team mode, no standing-teammates section, no Completion Signal block) runs cleanly on opus-4-7 + low effort even on 2.1.121.

**AC-R2 — section-stripped patch on standing-teammate test (PASS — DIVERGED from cycle-2 FAIL).**

- Pre-step: `git diff --quiet skills/commission/bin/claude-team` exit 0.
- Patch applied: replace `claude-team:410-430` loop body with `lines.append(f'- {name} is available; SendMessage to it.')` and drop the "Full routing contract" footer. Patch captured to `/tmp/cycle3/ac-r2.patch` (797 bytes).
- Run: `unset CLAUDECODE && KEEP_TEST_DIR=1 uv run pytest tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips --runtime claude --model opus --effort low -v`. **Exit code 0, 318.53s wallclock.**
- `claude --version`: `2.1.121 (Claude Code)`.
- Model stamps from `/tmp/cycle3/ac-r2-fo-log.jsonl`:
  ```
  $ grep -oh 'claude-opus-4-[67]' /tmp/cycle3/ac-r2-fo-log.jsonl | sort | uniq -c
    34 claude-opus-4-7
  $ grep -oh '"model":"[^"]*"' /tmp/cycle3/ac-r2-fo-log.jsonl | sort | uniq -c
    31 "model":"claude-opus-4-7"
     2 "model":"sonnet"
  ```
- Patched dispatch prompt section (extracted from `Agent` tool_use in fo-log; verifies the patch took effect):
  ```
  ### Standing teammates available in your team

  These standing teammates are available in your team; you MAY route to them via SendMessage. Best-effort, non-blocking, 2-minute timeout; proceed with un-polished/un-reviewed content if no reply.

  - echo-agent is available; SendMessage to it.
  ```
- AC-R2 PATCH (paste of `/tmp/cycle3/ac-r2.patch`):
  ```diff
  diff --git a/skills/commission/bin/claude-team b/skills/commission/bin/claude-team
  index dd7a0825..7e48a484 100755
  --- a/skills/commission/bin/claude-team
  +++ b/skills/commission/bin/claude-team
  @@ -408,26 +408,7 @@ def cmd_build(args):
               '',
           ]
           for name, description, mod_path in standing_teammates:
  -            desc = description or 'standing teammate'
  -            usage_body = _parse_routing_usage_body(mod_path) if mod_path else None
  -            if usage_body:
  -                lines.append(f'- **{name}** ({desc})')
  -                for body_line in usage_body.split('\n'):
  -                    if body_line:
  -                        lines.append(f'  {body_line}')
  -                    else:
  -                        lines.append('')
  -            else:
  -                lines.append(
  -                    f'- **{name}** ({desc}): SendMessage with the relevant input '
  -                    f'shape; reply format per the mod.'
  -                )
  -        lines.append('')
  -        lines.append(
  -            'Full routing contract: see '
  -            '`skills/first-officer/references/first-officer-shared-core.md` '
  -            '`## Standing Teammates`.'
  -        )
  +            lines.append(f'- {name} is available; SendMessage to it.')
           prompt_parts.append('\n'.join(lines))
  ```
- Patch revert verified: `git checkout -- skills/commission/bin/claude-team && git diff --quiet skills/commission/bin/claude-team` exit 0.
- Comparison to cycle-2: cycle-2 R2 FAILED at milestone 5 (`ECHO: ping reply received`) in 135.51s with `error_max_budget_usd: Reached maximum budget ($2)`; cycle-3 R2 PASSED in 318.53s. **Same patch, same fixture, same effort tier — only Claude Code drifted (2.1.112 → 2.1.121) and the entire production main moved (PR #181, lazy-spawn, etc.).** The 318s wallclock is 2.4× cycle-2's failed run time; the test now reaches all 5 milestones within budget.
- Interpretation: cycle-2's claim that "section-richness compression is NOT the priming surface" is **falsified for 2.1.121** — the same patch that failed on 2.1.112 now PASSES. Either (a) the upstream Claude Code regression has been partially fixed in 2.1.113-121 (FO-opus-4-7 reply-folding / orchestration), making section richness load-bearing again, or (b) the section richness was always partially load-bearing but cycle-2's failure mode (M5 budget exhaustion) was a downstream symptom of FO-opus general bloat that is now compressed enough on the same $2 budget. Without re-running cycle-2's UNPATCHED test on 2.1.121 to establish the new baseline, the two hypotheses are observationally equivalent here.

**AC-R3 — sonnet FO on standing-teammate test (PASS — matches cycle-2).**

- Hard pre-step: `git diff --quiet skills/commission/bin/claude-team && git diff --quiet tests/test_gate_guardrail.py` exit 0 — both files clean before pytest.
- Run: `unset CLAUDECODE && KEEP_TEST_DIR=1 uv run pytest tests/test_standing_teammate_spawn.py::test_standing_teammate_spawns_and_roundtrips --runtime claude --model sonnet --effort low -v`. Exit code 0, 158.41s wallclock.
- `claude --version`: `2.1.121 (Claude Code)`.
- Model stamps from `/tmp/cycle3/ac-r3-fo-log.jsonl`:
  ```
  $ grep -oh 'claude-opus-4-[67]' /tmp/cycle3/ac-r3-fo-log.jsonl | sort | uniq -c
     5 claude-opus-4-7
  $ grep -oh '"model":"[^"]*"' /tmp/cycle3/ac-r3-fo-log.jsonl | sort | uniq -c
     1 "model":"claude-opus-4-7"
    53 "model":"claude-sonnet-4-6"
     3 "model":"sonnet"
  ```
- FO ran on sonnet (53 stamps), ensign sub-agent ran on opus-4-7 by default (5 stamps in tool_use_result rollups) — same FO-vs-ensign split as cycle-2's R3.
- Comparison to cycle-2: identical wallclock to cycle-2 (158.30s vs 158.41s — within 0.1s) and identical model-split shape. The 2.1.111-vs-2.1.112 caveat from cycle-2 (sonnet behavior on 2.1.112 not pre-validated) extends to 2.1.121 — but the cycle-3 PASS on 2.1.121 plus cycle-2's PASS on 2.1.112 plus the natural 2.1.111 expectation is now a 2-of-3 evidence chain that sonnet FO is a stable workaround across the 2.1.111–2.1.121 range.

### Did anything shift the failure shape?

The AC-R2 cycle-2 → cycle-3 flip is the load-bearing finding. Three candidate explanations:

1. **Claude Code 2.1.113-121 partially fixed FO-opus reply-folding / orchestration.** Cycle-2's milestone-5 budget exhaustion (FO failed to fold the ECHO reply into its parent stream within $2) does not occur on 2.1.121 with the same patch and same fixture. Plausible — Anthropic shipped fixes between 2.1.112 and 2.1.121 (model versions remained `claude-opus-4-7` throughout, but Claude Code internals changed). Tonight's PR #181 opus-tier failures show *some* FO-opus behavior is still broken on different tests, so any fix would be partial.
2. **PR #181 stage-worktree-stickiness OR lazy-spawn (#107) interact with the priming surface in ways that reduce token bloat in the standing-teammate fixture specifically.** Lazy-spawn means the FO does not pre-spawn echo-agent on test setup; it spawns on demand. That changes the timing/budget profile of the FO's orchestration loop in a way that could leave more headroom for the M5 reply-folding step. Plausible but mechanistically vague — would need fo-log diff against a cycle-2 fo-log to confirm.
3. **The section-richness hypothesis was always correct, and cycle-2's M5 FAIL was a downstream symptom of FO bloat that the section-strip patch alone could not compensate for at 2.1.112 budget headroom.** Cycle-2 was right to falsify "section compression alone unblocks opus-4-7 FO" but wrong to conclude "section richness is not the priming surface." On 2.1.121 with whatever combination of upstream fixes plus repo-internal changes, the section-strip patch is now sufficient — the priming was real, and other downstream costs simply now fit within the same $2 budget.

I cannot distinguish (1)/(2)/(3) without an UNPATCHED 2.1.121 run on the same fixture (would establish the new baseline) plus a 2.1.121 run with section-strip + lower budget (would test whether 318s/$1.x is the new pass margin). Both are out-of-scope for this sanity check. The dispatch authorized re-running the 3 ACs as documented, not adding a fourth.

### Read on cycle-2 recommendations

The cycle-2 composed recommendation set was:

- Follow-up (a): persistent plumbing fix for `tests/test_gate_guardrail.py` extra_args. **OBSOLETE** — already landed on main; AC-R1 ran cleanly without any patch.
- Follow-up (b): Layer 3 FO-side reply-handling investigation (since AC-R2 falsified Layer 2 section-richness). **PARTIALLY OBSOLETE** — AC-R2 now PASSES with section-strip on 2.1.121. The "reply-folding" framing is no longer the load-bearing hypothesis for THIS fixture; tonight's PR #181 opus-tier failures suggest the broader FO-opus regression sits elsewhere (different tests, different surfaces). Re-scope to "FO-on-opus-4-7 regression — mechanism TBD across multiple test fixtures."
- Follow-up (c): pin `--model sonnet` as workflow FO default (with 2.1.111 re-confirmation gate). **STILL LOAD-BEARING** — AC-R3 PASS on 2.1.121 strengthens the evidence (now a 2-version pass: 2.1.112 + 2.1.121). Tonight's PR #181 opus-tier failures show the regression is broader than any single-test patch (R2's section-strip) can cover; sonnet FO is still the only known mitigation that flips ALL FO-driven tests green.

The mitigation that stays load-bearing is **(c) pin `--model sonnet` as workflow FO default.** AC-R2's PASS gives an alternative path (compress section emission), but that path: (i) only addresses the standing-teammate routing surface, (ii) does not address tonight's PR #181 failures on `test_dispatch_completion_signal` and `test_feedback_keepalive` (which don't involve the standing-teammates section in the same way), (iii) requires modifying `claude-team` source for what may be a transient priming sensitivity. Sonnet pin covers all surfaces with a single workflow-default change.

## Final recommendation

**Advance #177 to implementation against the model-pin mitigation: change spacedock workflow FO defaults from `opus` to `sonnet` until upstream Claude Code resolves the FO-on-opus-4-7 regression class.**

### Implementation scope

The implementation stage of #177 should:

1. Change `stages.defaults.model` (or equivalent) for FO-driving stages in the spacedock workflow templates from `opus` (current default) to `sonnet`. Targets: `skills/commission/templates/`, any workflow scaffolding that ships the default FO model.
2. Document the regression context with a cross-reference to #177 in the workflow scaffolding's README or developer doc — explain why FO defaults to sonnet, link to this entity for the evidence chain.
3. Add a TODO / re-evaluation note to revisit the default once the upstream Claude Code regression on `claude-opus-4-7` FO orchestration is resolved (no specific upstream issue exists yet — note that filing one is a separate task).
4. Ensure the change is opt-out at the workflow level — captains who explicitly want `--model opus` for their FO should still be able to override.

### Acceptance Criteria

**AC-F1 — Workflow scaffolding default points to sonnet for FO-driving stages.**

- Verify: `grep -rE "model:\s*opus|model:\s*claude-opus" skills/commission/templates/ skills/first-officer/` returns no matches at the FO-default scope; the FO default in scaffolding files is `sonnet`.
- Pass: a captain commissioning a new spacedock workflow gets sonnet as the FO model by default. Verified by inspecting the generated workflow's stage-defaults block after running the commission flow on a fresh workspace.
- Verified by: `tests/test_commission_skill_output.py` (or equivalent commission-output regression test) including an assertion that the default FO model stamp is `sonnet`. If no such test exists, add one as part of AC-F1's implementation.

**AC-F2 — Regression context documented with cross-reference to #177.**

- Verify: the workflow scaffolding's README or developer doc (likely `skills/commission/README.md` or `skills/first-officer/README.md`) contains a note explaining the sonnet default and links to `docs/plans/opus-4-7-ensign-hallucination-scope.md`.
- Pass: `grep -lE "opus-4-7|177|FO-on-opus" skills/commission/README.md skills/first-officer/README.md` returns at least one match referring to this entity.
- Verified by: documentation lint / static check, OR a one-line static test asserting the cross-reference string is present in the relevant README.

**AC-F3 — Override mechanism preserved.**

- Verify: a captain can set `--model opus` on the FO via existing CLI flags (e.g., `--model opus` on `claude-team build` calls, or workflow-level model override) and the override is honored end-to-end.
- Pass: an integration test (or existing test like `test_standing_teammate_spawn` with explicit `--model opus`) demonstrates the override path still runs through the same FO assembly logic with the captain's chosen model.
- Verified by: an existing live test invocation with explicit `--model opus` continues to dispatch FO on opus (model stamps in fo-log.jsonl confirm). The cycle-3 AC-R2 / AC-R1 runs already establish this — no new test needed if the existing live tests cover the override.

**AC-F4 — Live re-confirmation on 2.1.111 before commit (or explicit waiver).**

- Verify: re-run the standing-teammate fixture with `--model sonnet --effort low` on Claude Code 2.1.111 (cycle-2's preferred version); confirm PASS. OR document a captain waiver if 2.1.111 is no longer accessible and the 2.1.121 + 2.1.112 evidence chain is accepted as sufficient.
- Pass: a fo-log.jsonl on 2.1.111 with sonnet stamps and a green test result, OR an entity-body waiver block citing the 2-of-3 version evidence chain (2.1.112 PASS, 2.1.121 PASS).
- Verified by: file existence check on the captured fo-log, OR presence of the waiver block in the entity body.

### Test plan

- **Cost**: ~2-3 min for the commission-output regression test (static), ~5 min if AC-F4 runs the live 2.1.111 re-confirmation. AC-F1 / AC-F2 are static doc/code edits with cheap verifications.
- **Risk**: low. Workflow defaults change is a single point of edit; override path verified by existing tests. The captain waiver path on AC-F4 is acceptable because the 2-version sonnet evidence chain (cycle-2 + cycle-3) is strong.
- **E2E**: not strictly required — the existing live tests (`test_standing_teammate_spawn`, `test_gate_guardrail`) already cover the FO+ensign roundtrip on sonnet. AC-F4's re-confirmation is the one optional E2E step.
- **Static**: yes — AC-F1 (default value), AC-F2 (cross-reference present), AC-F3 (override path test exists).

### Out of scope for #177 implementation

- Filing the upstream Anthropic issue about FO-on-opus-4-7 regression. Separate task; #177 implementation only changes our workflow defaults.
- Section-richness compression in `claude-team`. AC-R2 PASS makes this a viable alternative path, but it's narrower (standing-teammate surface only) than the sonnet pin and adds source-code complexity. If a future captain wants to ship section compression as a complementary mitigation, that's a follow-up entity.
- Investigating tonight's PR #181 opus-tier failures (`test_dispatch_completion_signal`, `test_feedback_keepalive`). The sonnet pin makes those tests green by default; the underlying mechanism is mechanism-TBD and out-of-scope per cycle-2 Follow-up (b)'s revised framing.

## Stage Report: ideation

- DONE: Re-run cycle-2's three experiments on current main as a sanity check that the AC-R1/R2/R3 outcomes still hold.
  AC-R1 PASS (44.33s, 20 opus-4-7 stamps, no plumbing patch needed); AC-R2 PASS (318.53s, 34 opus-4-7 stamps, with section-strip patch); AC-R3 PASS (158.41s, FO=sonnet, ensign default opus-4-7). All three /tmp/cycle3/ac-r{1,2,3}-fo-log.jsonl preserved.
- DONE: Compare current-main outcomes against cycle-2 outcomes.
  R1 and R3 match cycle-2; R2 DIVERGED (cycle-2 FAIL → cycle-3 PASS). Comparison and three candidate explanations written into `## Cycle-3 sanity-check` § "Did anything shift the failure shape?". Cycle-2 cycle Follow-up (a) plumbing fix has effectively landed on main; Follow-up (b) reply-folding framing is now partially obsolete; Follow-up (c) sonnet pin remains load-bearing.
- DONE: Update the entity body in place.
  Appended `## Cycle-3 sanity-check` and `## Final recommendation` sections after the existing `## Stage Report (validation, repurpose-experiment)` (the file's prior tail). All cycle-2 sections preserved verbatim as audit trail.

### Summary

Cycle-3 sanity check on current main (Claude Code 2.1.121, HEAD `465d4ffc`) re-ran cycle-2's three Layer 2 experiments. R1 PASS and R3 PASS match cycle-2; R2 PASS DIVERGED from cycle-2's FAIL — same patch, same fixture, same effort, only Claude Code drifted (2.1.112 → 2.1.121) and main advanced (PR #181 stickiness, lazy-spawn, etc.). Tonight's PR #181 opus-tier failures (`test_dispatch_completion_signal`, `test_feedback_keepalive`) confirm the FO-on-opus-4-7 regression class is still active on different test surfaces. Final recommendation: advance #177 to implementation against cycle-2's Follow-up (c) — pin `--model sonnet` as workflow FO default — because it covers ALL surfaces with a single workflow-default change, while the now-passing AC-R2 section-strip path covers only standing-teammate routing. AC-F1 through AC-F4 enumerated with verification commands and end-state tests.
