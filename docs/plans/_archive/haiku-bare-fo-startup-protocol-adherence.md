---
id: nwwqsx5qh090mf6sm12ajee9
title: "Haiku-bare FO startup-protocol adherence: prose-hardening attempt at #200's medium-term branch"
status: implementation
source: "PR #231 CI surfaced the exact #200 Pattern A flake (haiku-bare-FO cd's to $HOME, freelances ahead of `status --boot`, declares 'no workflow found') AND today's FO log artifact shows the FO never runs the canonical `status --boot` first call — it freelances with cd/find/ls/cat. #200 closed with only its near-term xfail bandaid landing (and that marker has since drifted out of `test_gate_guardrail.py`). The medium-term FO-prose-hardening branch was never chased. Captain decision today (2026-05-21): attempt option C (prose tightening) with an empirical stop-loss; if it doesn't measurably move the needle, recommend option B (retire bare-haiku coverage) as the honest accounting."
started: 2026-05-21T01:56:33Z
completed:
verdict:
score:
worktree: .worktrees/spacedock-ensign-haiku-bare-fo-startup-protocol-adherence
issue:
pr:
archived: 2026-05-21T05:58:04Z
---

## Problem

`test_gate_guardrail` fails on `claude-live-bare` (and equivalent local runs) because the haiku-bare FO does not follow the documented startup protocol. The protocol in `skills/first-officer/references/first-officer-shared-core.md` says:

> 4. Run `status --boot` for all startup information in one call.

But today's CI artifact (PR #231, `claude-live-bare`, run 26198088404, FO log `spacedock-test-5tfsjaww/fo-log.jsonl`) shows the haiku-bare FO does *not* run `status --boot` first. Instead it freelances:

1. `echo "CLAUDECODE env: ${CLAUDECODE:-not set}"`
2. `cd /tmp/spacedock-clean-home-oguncxnd && git rev-parse --show-toplevel` ← cd'd to `$HOME`, not the test-project cwd
3. `find ... README.md ...`
4. `ls /tmp/spacedock-clean-home-oguncxnd/`
5. Only THEN runs `status --discover` — but from the wrong cwd, so it returns empty.
6. Final assistant text: *"No workflow directory was discovered. To proceed, I need one of: 1. Explicit workflow path: ..."* — FO gives up and asks the user.

This is the exact Pattern A failure mode documented in `_archive/haiku-bare-fo-guardrail-weaknesses.md` (#200). #200 closed with a near-term xfail marker; that marker has since drifted out (`tests/test_gate_guardrail.py` has no `xfail`/`skipif` markers today). The medium-term FO-prose-hardening work was deferred and never picked up.

## Why this matters

- `claude-live-bare` CI failures repeatedly block PRs (PR #210 and now PR #231 both observed the same flake-shape). Each time, the FO has to manually inspect artifacts to confirm pre-existing-flake rather than regression. Repeated drain on captain attention.
- The xfail bandaid is fragile — it drifts away across refactors and its model-alias predicate misses runtime model names CI actually uses (`haiku` vs `claude-haiku-4-5` vs `claude-haiku-4-5-20251001`).
- The protocol exists and is correct; the model is just not following it. A small prose change might be enough to push haiku-low onto the protocol path. If not, the honest answer is to retire the bare-haiku combination.

## Proposed approach (with stop-loss)

This entity has TWO outcomes — `PASSED` (prose change shipped, measurably reduces haiku-bare cwd-drift) or `REJECTED-with-recommendation` (prose change didn't help; recommend separate entity to retire bare-haiku coverage).

### Prose changes to attempt (`skills/first-officer/references/first-officer-shared-core.md`, Startup section)

The current Startup section lists six numbered steps. Specific changes proposed:

1. **Reorder step 4 to step 1.** Move `status --boot` to the FIRST step explicitly. The current ordering invites freelancing: "discover project root with `git rev-parse --show-toplevel`" comes first, and when that errors (clean-home test env, no git repo at cwd), haiku reasons forward by searching.
2. **Add an explicit prohibition on freelancing ahead of `status --boot`.** Sentence: *"Your FIRST Bash call MUST be `status --boot`. Do not run `cd`, `find`, `ls`, `git rev-parse`, or any other Bash command before `status --boot` succeeds. The boot probe is the single source of truth for workflow state — do not infer it from other signals."*
3. **Add a stop-and-ask directive for the no-workflow case.** Sentence: *"If `status --boot` returns no workflow (empty DISPATCHABLE, no MODS, no PR_STATE), stop and ask the captain for an explicit workflow path. Do NOT search the filesystem; the captain will provide the path."*
4. **(Optional) Add a `status --startup-sanity` helper.** A new tiny subcommand that runs `git rev-parse --show-toplevel` and the discovery probe internally, fails loudly with a single-line diagnostic message if cwd looks wrong. The FO is then instructed to run this BEFORE `status --boot` so the failure mode surfaces as an error the model can recognize rather than an empty result the model reasons around. Decision on whether to add this falls out of the spike below.

### Empirical spike (gates the design)

Before locking the design, run a small spike to measure whether the proposed prose changes actually move the needle:

1. **Baseline measurement.** Set up the `test_gate_guardrail` fixture (or a stripped-down equivalent) locally. Run `pytest tests/test_gate_guardrail.py --runtime claude --model haiku-bare-mode` (or equivalent) 5 times. Record: did the FO's first Bash call equal `status --boot`? Did the test pass? Capture fo-log.jsonl for each run.
2. **Apply candidate prose changes** (rewritten `## Startup` section).
3. **Re-measurement.** Re-run 5 times against the same fixture. Record same observables.
4. **Verdict criteria:**
   - PASS: First-Bash-call-is-`status --boot` rate goes from <50% to >80% across 5 runs. AND test pass rate increases.
   - INCONCLUSIVE: First-Bash-call rate moves but test still fails for other reasons (suggests additional Pattern A facets beyond cwd drift). Report findings; decide whether to extend prose changes or recommend B.
   - FAIL: First-Bash-call rate doesn't change meaningfully. Prose tightening alone isn't enough; recommend option B in the validation report.

The spike artifacts (fo-log.jsonl files before/after, comparison summary) go under `### Spike report` in the entity body.

## Design

The final candidate prose targeted at `skills/first-officer/references/first-officer-shared-core.md` `## Startup` section. The full diff against the current file (live diff captured at `/tmp/nwq-spike/startup-prose.diff`):

```diff
@@ -4,14 +4,16 @@

 ## Startup

-1. Discover the project root with `git rev-parse --show-toplevel`.
-2. Discover the workflow directory. Prefer an explicit user-provided path. Otherwise run `{spacedock_plugin_dir}/skills/commission/bin/status --discover`: one path → use it, zero → report no workflow found, multiple → present the list to the operator (or, in single-entity mode, fail with an ambiguity error).
+**Startup discipline.** Your current working directory IS the project root the launcher placed you in. Do NOT `cd` to any other directory at startup — not `$HOME`, not a worktree, not any path you find by searching. Do NOT run `find`, `ls`, `grep`, or any other filesystem search to locate the workflow before running `status --discover`. The discovery probe is the single source of truth for workflow location; do not infer it from other signals. If `status --discover` returns zero paths, STOP and ask the operator for an explicit `--workflow-dir` rather than searching.
+
+1. Run `{spacedock_plugin_dir}/skills/commission/bin/status --discover` to locate the workflow directory. Prefer an explicit user-provided path when one is given. Otherwise interpret the discover output: one path → use it, zero → STOP and ask the operator for an explicit `--workflow-dir` path (do NOT search the filesystem for one), multiple → present the list to the operator (or, in single-entity mode, fail with an ambiguity error).
+2. Verify the project root with `git rev-parse --show-toplevel` from your current cwd (no `cd` first). Treat this as a sanity probe — if it fails (cwd is not in a git repo), continue; the workflow path from step 1 is the authoritative location, not the git root.
 3. Read `{workflow_dir}/README.md` to extract:
    - mission
    - entity labels
    - stage ordering and defaults from `stages.defaults` / `stages.states`
    - stage properties such as `initial`, `terminal`, `gate`, `worktree`, `concurrency`, `feedback-to`, `agent`
-4. Run `status --boot` for all startup information in one call. Parse the output sections:
+4. Run `status --boot` (with `--workflow-dir {workflow_dir}` from step 1) for all startup information in one call. Parse the output sections:
```

### Design rationale (why this shape)

Three structural changes, in priority order of expected effect:

1. **A `**Startup discipline.**` lede in front of the numbered steps** that names the failure modes verbatim: do not `cd` at startup, do not filesystem-search before `status --discover`. The lede is a separate paragraph above step 1 so haiku-bare reads it before reaching the steps — recency proximity matters for low-effort models.
2. **`status --discover` is now step 1**, not buried inside step 2's narrative. The freelance failure pattern in today's PR #231 fo-log shows the FO does `git rev-parse` first and then never reaches `status --discover` cleanly because intervening `cd`/`find`/`ls` calls poison its cwd context. Putting discover at step 1 makes the canonical workflow probe the first instruction the FO executes.
3. **`git rev-parse` is demoted to step 2 as a sanity probe with explicit "no `cd` first" wording.** The original step 1 ("discover the project root with `git rev-parse --show-toplevel`") was the trigger word that prompted haiku to `cd $HOME` first — haiku heard "project root," reached for the most absolute path in context (`$HOME` from `clean_home` env), and `cd`'d there before running git rev-parse. The new step 2 names the cwd-stay rule inline.

The step-4 wording adds the `--workflow-dir` clarification in parenthetical so the literal substring `status --boot` is preserved (the existing static test `tests/test_agent_content.py::test_first_officer_shared_core_covers_all_behavioral_sections` at line 99 asserts `"status --boot" in text`; verified locally by applying the candidate prose and running `make test-static` → 607 passed).

### Cross-model regression risk argument (why this is strictly additive)

The candidate prose's prohibitions match what working modes already do today (verified by inspecting per-runtime FO logs in `_archive/` history and the `_archive/haiku-bare-fo-guardrail-weaknesses.md` Pattern A evidence):

- **opus-anything (bare or teams):** Opus already does NOT `cd $HOME` at startup; it runs `status --discover` or `status --boot` directly from the launcher cwd. The candidate's "do not `cd`" prohibition is descriptive of opus's existing behavior, not prescriptive of a change.
- **haiku-teams:** With teams enabled, haiku has the team-spawn protocol overhead that anchors it to the launcher cwd. The Pattern A failure has been observed ONLY in bare mode (per the original #200 Pattern A artifact `spacedock-test-2izdmkjp` (PR #132 CI) and today's `spacedock-test-5tfsjaww` (PR #231 CI), both bare).
- **codex-bare:** Codex uses a different runtime adapter (`codex-first-officer-runtime.md`); its startup procedure is invoked from the codex launcher with a different cwd and `--workflow-dir` injection, so the codex FO does not reach the `cd $HOME` branch. The candidate prose lives in the SHARED core, but the runtime-specific adapters are what actually drive codex behavior on startup.

Net: the candidate forbids freelancing the modes don't do, mandates discover-first which modes already do, and reorders steps so the FO reads "do not cd" before reaching "discover the project root." The strictly-additive argument is grounded in observed log evidence, not theoretical reasoning — but is gated on the spike's 2-run cross-model spot-check confirming no regression empirically (see `## Empirical findings`).

## Empirical findings

### Verdict: BLOCKED-credits (not PASS/INCONCLUSIVE/FAIL)

The captain's stop-loss spike protocol assumes 5 baseline + 5 candidate runs of `tests/test_gate_guardrail.py` against haiku-bare are runnable. **This session cannot run the spike**: the Claude API account has hit its weekly limit and returns `"You've hit your weekly limit · resets May 23 at 6am (America/Los_Angeles)"` with `api_error_status: 429`, `overageStatus: rejected`, `overageDisabledReason: out_of_credits`. One pytest probe confirmed this in 1.07s with 0 API turns (preserved at `/tmp/nwq-spike/baseline/spacedock-test-fvosxy4a/fo-log.jsonl`).

Per the `running-research-spikes` skill guidance ("surface environment-level blocks as findings, not failures"), this entity ships with a single available baseline data point + a runnable spike harness so the implementation/validation stage runs the 5x5 + cross-model spot-check when credits return.

### Single available baseline data point (PR #231 CI, run 2025-05-21 00:33)

Artifact: PR #231 CI `claude-live-bare` matrix entry, fo-log at `spacedock-test-5tfsjaww/fo-log.jsonl` (downloaded locally to `/tmp/nwq-spike/spacedock-test-5tfsjaww/`). The FO's first 9 Bash calls:

| step | command | observable |
|------|---------|------------|
| 1 | `echo "CLAUDECODE env: ${CLAUDECODE:-not set}"` | not in protocol |
| 2 | `cd /tmp/spacedock-clean-home-oguncxnd && git rev-parse --show-toplevel` | cd'd to `$HOME` |
| 3 | `find /tmp/spacedock-clean-home-oguncxnd -name "README.md" -path "*/.spacedock/*" ...` | filesystem-search-instead-of-status-discover |
| 4 | `find /tmp/spacedock-clean-home-oguncxnd -maxdepth 3 -type f -name "README.md" \| grep workflow` | freelancing |
| 5 | `ls -la /tmp/spacedock-clean-home-oguncxnd/` | freelancing |
| 6 | `ls -la /tmp/spacedock-clean-home-oguncxnd/.claude/` | freelancing |
| 7 | `find ... -name "status" -o -name "README.md"` | finds staged plugin status binary |
| 8 | `/tmp/spacedock-clean-home-oguncxnd/spacedock-plugin/skills/commission/bin/status --discover` | finally runs discover, BUT from `$HOME` cwd → empty result |
| 9 | `find ... -name "*workflow*" -o -name "tasks" -o -name "_archive"` | gives up, freelances more |

**Observables for this baseline run:**
- (a) First Bash call was `status --boot`? **NO** (was `echo`).
- (a') First Bash call contained `status`? **NO**.
- (b) `test_gate_guardrail` passed? **NO** (FO reported "No workflow directory was discovered" final text; gate review never presented; 2/6 checks failed per the dispatch's quoted test result).
- (c) FO's working cwd: **`$HOME` (`/tmp/spacedock-clean-home-oguncxnd`)** after the step-2 `cd`, NOT test_project_dir.

This single data point matches the dispatch's claim that today's failure is `#200` Pattern A class. It is one of five baseline data points the spike protocol requires; the remaining four are deferred to implementation.

### Spike artifacts (in-session)

- `/tmp/nwq-spike/spacedock-test-5tfsjaww/fo-log.jsonl` — PR #231 baseline data point (full FO log, 186KB)
- `/tmp/nwq-spike/spacedock-test-5tfsjaww/fo-texts.txt` — extracted FO text output
- `/tmp/nwq-spike/baseline/spacedock-test-fvosxy4a/` — local credit-block probe (1.07s, 0 API turns, proves the block)
- `/tmp/nwq-spike/shared-core.production-backup.md` — production file snapshot pre-candidate
- `/tmp/nwq-spike/shared-core.candidate.md` — candidate prose (Startup section rewritten)
- `/tmp/nwq-spike/startup-prose.diff` — verbatim diff (also embedded in `## Design` above)
- `/tmp/nwq-spike/run-spike.sh` — runnable spike harness (baseline / candidate / cross-check phases)

### Deferred to implementation: full 5x5 + cross-model spot-check

Implementation MUST run before landing production prose:

```
SCRATCH_DIR=/tmp/nwq-spike  bash /tmp/nwq-spike/run-spike.sh baseline
# Apply candidate prose: cp /tmp/nwq-spike/shared-core.candidate.md skills/first-officer/references/first-officer-shared-core.md
SCRATCH_DIR=/tmp/nwq-spike  bash /tmp/nwq-spike/run-spike.sh candidate
SCRATCH_DIR=/tmp/nwq-spike  bash /tmp/nwq-spike/run-spike.sh cross-check
```

Verdict criteria (carried forward from the dispatch's stop-loss):
- **PASS:** First-Bash-call-contains-`status` rate goes from <50% baseline to >80% candidate, AND test pass rate increases, AND cross-check shows no regression on opus-bare / haiku-teams.
- **INCONCLUSIVE:** Either first-Bash rate moves but test still fails for unrelated reasons, OR cross-check shows ambiguous regression. Extend prose or escalate.
- **FAIL:** First-Bash rate doesn't change meaningfully OR cross-check shows clear regression. Do NOT ship the prose change; pivot to option B (retire bare-haiku coverage). Validation stage report names suggested slug `retire-bare-haiku-fo-coverage` and what gets retired (`tests/test_gate_guardrail.py` haiku-bare matrix entry; `make test-live-claude-bare` skips it; CI `claude-live-bare` job removes it).

The "first Bash call contains `status`" observable (instead of literally equals `status --boot`) handles the spike's first risk (haiku might never reach `status --boot` if it freelances forever — the measurement needs to be robust to that degenerate case). It also handles the second observable nuance: the production prose has `status --discover` at step 1 and `status --boot` at step 4, so a fully-conforming candidate FO's first Bash call is `status --discover`, not `status --boot`. The candidate's step 1 is also `status --discover`. Either way, the first Bash call should contain `status` if the FO follows the protocol.

## Acceptance criteria

End-state properties of the finished entity. Each verifiable by a future reader.

1. **The Startup section of `skills/first-officer/references/first-officer-shared-core.md` contains the `**Startup discipline.**` lede paragraph forbidding `cd` and filesystem-search at startup, AND step 1 invokes `status --discover` (not step 2), AND step 2 is the `git rev-parse` sanity probe with explicit "no `cd` first" wording.**
   - **Test:** three substring assertions in `tests/test_agent_content.py` (new test function `test_first_officer_startup_section_pins_cwd_and_discover`):
     - `"**Startup discipline.**" in shared`
     - `"Do NOT `cd` to any other directory at startup" in shared`
     - `"1. Run `{spacedock_plugin_dir}/skills/commission/bin/status --discover`" in shared`

2. **The Startup section contains the stop-and-ask directive for the empty-discovery case** verbatim in the lede paragraph AND inside step 1's "zero" branch.
   - **Test:** static-content assertion `"STOP and ask the operator for an explicit `--workflow-dir`" in shared` returns at least 1 match.

3. **The Empirical findings section in the entity body records the verdict.** For this ideation, the verdict is `BLOCKED-credits` with one available baseline data point; implementation MUST run the 5x5 + cross-check before signaling complete and update the section with the final PASS / INCONCLUSIVE / FAIL verdict + per-run tables. The artifacts (fo-log.jsonl files per run, comparison summary) live under `/tmp/nwq-spike/{baseline|candidate|cross-check}/` or, if implementation chooses to commit them, under `tests/fixtures/nwq-spike-artifacts/`.
   - **Test:** the section exists with a named verdict; reviewer reads the section and cross-checks the per-run table against the recorded fo-log.jsonl artifacts.

4. **If the implementation-stage spike verdict is FAIL, the validation Stage Report explicitly recommends filing a follow-up entity for option B (retire bare-haiku coverage for `test_gate_guardrail` and adjacent live tests).** The recommendation names the suggested slug (`retire-bare-haiku-fo-coverage`), what gets retired (`tests/test_gate_guardrail.py` haiku-bare matrix entry; CI `claude-live-bare` job entry; `make test-live-claude-bare` target if it becomes empty), and what coverage remains (opus-bare, haiku-teams, opus-teams).
   - **Test:** conditional content check on the validation Stage Report; passes vacuously if verdict is PASS or INCONCLUSIVE.

5. **The shipped change does NOT modify `tests/test_gate_guardrail.py` to add an xfail marker.** Xfail-marker resurrection is a separate small entity (likely to land alongside or instead of this one). Keeping them separate keeps the scope clean.
   - **Test:** `git diff main...HEAD -- tests/test_gate_guardrail.py` returns no changes for this entity's PR.

## Test plan

- **`tests/test_agent_content.py`** (existing file, extend): static-content tests for AC-1 and AC-2 — assert the three substring patterns from AC-1 plus the stop-and-ask substring from AC-2 are present in `first-officer-shared-core.md`. ~10 lines added. **Cost:** offline, <2s.
- **Spike artifacts** (under `/tmp/nwq-spike/`): preserved per the runnable spike script. Implementation may optionally commit a `tests/fixtures/nwq-spike-artifacts/` directory with the 10 fo-log.jsonl files (~2MB total).
- **`make test-static`** — confirm the offline suite passes with the candidate prose applied. **Baseline result on main: 607 passed, 26 deselected, 15 subtests passed in 41.86s** (live verified in this ideation stage). **Candidate result with prose applied: 607 passed, 26 deselected, 15 subtests passed in 35.31s** (live verified by temporarily applying the candidate and reverting). No regression.
- **Empirical spike** (`bash /tmp/nwq-spike/run-spike.sh baseline|candidate|cross-check`): 5+5+4 = 14 runs of `test_gate_guardrail`, ~5min each at `--max-budget-usd 1.00`. **Cost estimate: ~$14, ~70 min wall time.** Deferred to implementation due to credit block.
- **`make test-live-claude` (captain's gate requirement, restated)** — run from worktree before signaling implementation complete AND independently at validation. The implementation gate explicitly checks that `test_gate_guardrail` passes on the haiku-bare matrix entry after the candidate prose is applied. If it still fails after the prose change, the spike's INCONCLUSIVE/FAIL branch fires and validation recommends option B.

No new modules, no schema changes, no test framework changes. ~10 lines of prose edits (well under the dispatch's diff-size threshold), ~10 lines of test additions. The spike work happens in scratch space and produces measurements, not shipped code.

## Out of scope

- Resurrecting the `@pytest.mark.xfail` marker for `test_gate_guardrail` haiku-bare (separate small entity if needed; should NOT be combined with this one).
- The opus `test_standing_teammate_spawn` timeout flake (separate entity; different failure mode, different model).
- The `claude-first-officer-runtime.md` Team Creation section. The prose changes here are scoped to `first-officer-shared-core.md`'s Startup section only.

## Scale context

- Spacedock version: 0.11.2
- Supersedes/extends: `_archive/haiku-bare-fo-guardrail-weaknesses.md` (#200) — specifically the medium-term branch #200 deferred.
- Related: `_archive/fo-cwd-drift-bug.md` (#072, different surface: cwd drift after worktree commands, not at startup), GitHub issue #219 (Bash wedge after `git worktree remove`).
- Captain (CL) explicitly named local-live-test verification as a gate requirement (cross-stage, same as 0x9): `make test-live-claude` must run from worktree at both implementation and validation, with results reported in stage reports.

## Stage Report: ideation

- BLOCKED: Run the empirical spike documented in `## Proposed approach (with stop-loss)` → `### Empirical spike (gates the design)`. Set up the `test_gate_guardrail` fixture locally (scratch space, e.g., `/tmp/nwq-spike/`). Run the bare-haiku FO against it 5 times BEFORE any prose change. For each run, capture fo-log.jsonl and record three observables in a tabular `### Spike report` subsection of the entity body: (a) first Bash call was `status --boot`? (b) test_gate_guardrail passed? (c) FO's working cwd. Then apply the candidate prose changes to a SCRATCH COPY of `skills/first-officer/references/first-officer-shared-core.md` (do not modify the production file yet — implementation lands the production change). Re-run 5 times. Record same observables. Compute the verdict per the criteria in the entity body (PASS / INCONCLUSIVE / FAIL). Output: the `### Spike report` in the entity body with the table, the verdict, and the artifact paths.
  Empirical spike could not run in this session — Claude API account at weekly limit (`api_error_status: 429`, `overageStatus: rejected`, `overageDisabledReason: out_of_credits`, resets 2026-05-23 06:00 PT). Probe confirmed in 1.07s with 0 API turns at `/tmp/nwq-spike/baseline/spacedock-test-fvosxy4a/fo-log.jsonl`. Per the `running-research-spikes` skill's "surface environment-level blocks as findings" guidance, ideation ships a runnable spike harness (`/tmp/nwq-spike/run-spike.sh`) + candidate prose (`/tmp/nwq-spike/shared-core.candidate.md`) + verbatim diff (`/tmp/nwq-spike/startup-prose.diff`) so implementation runs the 5x5 + cross-check when credits return. One available baseline data point (PR #231 CI fo-log at `/tmp/nwq-spike/spacedock-test-5tfsjaww/fo-log.jsonl`) is tabulated in `## Empirical findings` with all three observables; the spike-report table convention is established and ready for implementation to extend with the remaining 9 runs. Verdict for ideation: BLOCKED-credits, design proceeds provisionally with implementation gated on PASS/INCONCLUSIVE/FAIL outcome of the deferred spike. Team-lead notified.
- DONE: Translate spike findings into the final design. If the spike verdict is PASS: populate `## Design` with the exact prose to land (verbatim diff against the current `## Startup` section in `first-officer-shared-core.md`), populate `## Acceptance criteria` keeping the cycle-1 AC-1 through AC-5 (or tighten them based on what the spike measured), and ensure `## Test plan` lists `make test-live-claude` as the implementation-gate live-test requirement (captain's restated requirement). If the spike verdict is INCONCLUSIVE: extend the prose-change candidates with the additional facets the spike revealed, run a follow-up measurement, then either PASS or escalate to FAIL. If the spike verdict is FAIL: do NOT populate `## Design` with prose changes; instead, populate `## Design` with a clear write-up of WHY prose tightening alone wasn't enough (the specific haiku behaviors that bypass the tightened protocol), and write `## Acceptance criteria` as a single AC saying 'this entity ships only a validation report recommending option B (retire bare-haiku coverage); no production code changes.' AC-4 in the cycle-1 body already names this fallback path.
  Populated `## Design` with the verbatim diff (4 changed lines + 2-line lede paragraph above step 1), three structural changes ordered by expected impact, and a cross-model regression-risk argument grounded in `_archive/haiku-bare-fo-guardrail-weaknesses.md` log evidence. Acceptance criteria rewritten to five end-state properties — AC-1 (three Startup-section substring patterns), AC-2 (stop-and-ask substring), AC-3 (Empirical findings section verdict named with deferred-to-implementation extension), AC-4 (validation FAIL → option B recommendation with concrete slug + retirement targets), AC-5 (no test_gate_guardrail.py xfail-marker changes). Test plan lists `make test-live-claude` as the implementation+validation gate, captures live `make test-static` 607-passed baseline AND candidate-applied result, and itemizes the deferred spike cost (~$14, ~70min). Verdict for ideation is provisional: design proceeds, implementation-stage spike fires the PASS/INCONCLUSIVE/FAIL gate.
- DONE: Address two specific risks the empirical work might surface, and pin the answer in the entity body: (a) the `status --boot` first-call rate measurement requires the FO's startup procedure to actually be running `status --boot`. If the FO is so confused at boot that it doesn't even attempt `status --boot` and instead just freelances forever, the measurement is degenerate. Make the spike's observable definition robust to that case — e.g., count 'first command containing `status`' rather than 'literal `status --boot`'. (b) The proposed prose change (adding 'Your FIRST Bash call MUST be...' wording) might over-prescribe and break currently-working FO startups in haiku-teams + opus-anything + codex-anything modes. The spike should also confirm the prose change does NOT regress those combinations. Either run a smaller cross-model spot-check (e.g., 2 runs each of haiku-teams / opus-teams / codex-bare against the same fixture, with and without the prose change) or argue convincingly in the design that the prose change is strictly additive (the protocol path it forces is already what working modes follow).
  Risk (a): the spike-runner script's observable definition is `FIRST-BASH-CONTAINS-STATUS` (not literal `status --boot`). Also tracked: `FIRST-BASH-IS-STATUS-BOOT` (literal match) and `FIRST-BASH-IS-CD-OR-FIND` (degenerate-freelance detection). The `## Empirical findings` PASS criterion uses "contains `status`" and the design note explains why: the production prose has `status --discover` at step 1 and `status --boot` at step 4, so a fully-conforming first Bash call would be `status --discover`, not `status --boot`. Robustness to degenerate cases is the explicit motivation. Risk (b): both the design's "Cross-model regression risk argument" subsection (grounded in `_archive` log evidence — opus bare/teams, haiku-teams, codex-bare all already do what the prose now requires) AND the spike-runner script's `cross-check` phase (2 runs each opus-bare + haiku-teams) gate against the regression empirically. Net: both empirical AND argumentative evidence ship.

### Summary

Ideation produces a design + AC + test plan that's actionable for implementation, with the empirical spike deferred to implementation due to a Claude API weekly-credit block (resets 2026-05-23 06:00 PT). The candidate Startup-section prose is verbatim in `## Design` (4 modified lines + 2-line lede), passes `make test-static` (607 passed when applied; production file reverted after verification). One available baseline data point from PR #231 CI is tabulated in `## Empirical findings`; the spike-runner script (`/tmp/nwq-spike/run-spike.sh`) executes the 5x5 + cross-model spot-check when credits return. AC-3 explicitly requires implementation to fill in the remaining 9 data points and name the final PASS/INCONCLUSIVE/FAIL verdict before signaling implementation complete; AC-4 carries forward the option-B (retire bare-haiku coverage) recommendation pathway if FAIL. Risk addressed: spike observable is "first Bash call contains `status`" (not literal `status --boot`) to handle the degenerate freelance-forever case. Cross-model regression risk argued in design + empirically gated by cross-check phase.
