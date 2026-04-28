---
id: 200
title: "Haiku-bare FO behavioral weaknesses on guardrail suite (test_gate_guardrail + test_feedback_keepalive)"
status: validation
source: "session 2026-04-18 investigation of PR #132 (#190) bare-mode CI failures — two distinct haiku-bare-FO weakness patterns surfaced after #186 cycle-6 fixed test_gate_guardrail's model-fixture plumbing and #190's two-path observer sharpened test_feedback_keepalive's diagnostic. Neither failure is a regression from #186 or #190; both are pre-existing haiku-bare-FO weaknesses newly made visible by better fixtures."
started: 2026-04-28T15:02:39Z
completed:
verdict:
score: 0.5
worktree: .worktrees/spacedock-ensign-haiku-bare-fo-guardrail-weaknesses
issue:
pr: #160
mod-block: merge:pr-merge
---

## Why this matters

PR #132 (#190) CI exposed two distinct haiku-bare-FO behavioral failures on tests the `--effort low` + claude-haiku-4-5 + bare mode combination cannot currently reach the protocol-adherence bar the tests validate. Pre-session these failures were masked — test_gate_guardrail was vacuously passing (fixture ignored `--model`); test_feedback_keepalive was carrying a narrow `xfail(team_mode=="bare" AND model=="claude-haiku-4-5")` which #190 removed in favor of a diagnostic that would catch ALL failure modes. After #186 cycle-6 + #190 landed, the bare-mode failures are honest signal the test suite is designed to surface.

The two patterns are independent in their shape:

**Pattern A — `test_gate_guardrail` bootstrap failure.** The haiku-bare FO never reaches the workflow. Evidence (from `fo-log.jsonl` in the CI artifact `spacedock-test-2izdmkjp`):
- Early `Bash` tool_use with literal `{PWD}` brace-bug in the shell command (haiku failed to interpolate the variable, emitted it as a literal).
- `cd /home/runner/work/spacedock/spacedock && ... status --discover` — FO cd'd to the repo root, not the test-project cwd. `status --discover` found no workflows there.
- Final assistant text: "To proceed, I need one of the following: ... What workflow would you like me to process?" — FO gives up asking for clarification.
- Resulting test verdict: `FAIL: first officer presented gate review` (regex `gate review|recommend approve|recommend reject` never matches because FO never reached the gate). 6/7 other checks trivially pass (negative assertions: "did NOT advance", "did NOT self-approve").

2026-04-28 recurrence on PR #159 (`claude-live-bare`, CI run `25058370658`, job `73405230791`) confirmed the same Pattern A class with a more dangerous discovery outcome: the runtime `cwd` started inside the temp test project, but FO then executed commands from `/home/runner/work/spacedock/spacedock` and `status --discover` selected the repository's real `docs/plans` workflow instead of the fixture workflow. The fixture entity stayed at `status: work`, so the gate-hold negative checks passed incidentally, but the FO never presented a gate review for `gate-test-entity`. This is still a #200 bootstrap/wrong-cwd failure, not an SD-B32 regression.

That PR #159 failure also exposed a marker/harness gap: the known xfail was guarded on `model == "claude-haiku-4-5"`, while the CI make target passed the default pytest alias `model == "haiku"` and the runtime log reported the concrete model as `claude-haiku-4-5-20251001`. The behavior belongs to Pattern A; the alias miss is a separate test-marker normalization issue that should be fixed so the known weakness is consistently classified as XFAIL.

**Pattern B — `test_feedback_keepalive` tool-shape discipline failure.** The haiku-bare FO completes the workflow end-to-end but with wrong tool shapes. Evidence (from `spacedock-test-h42ehkks` CI artifact):
- Impl dispatch at line 54 correctly uses `subagent_type="spacedock:ensign"`.
- Validation dispatch at line 98 uses `subagent_type=None` (generic Agent, not spacedock:ensign). Path-A predicate requires `spacedock:ensign`; doesn't match.
- Line 151 attempts a `spacedock:ensign` dispatch but the prompt body is literally `SendMessage(to="afc4...", message="...")` — haiku confused Agent() with SendMessage(), nesting the SendMessage call string inside an Agent prompt.
- Entity archive has `## Stage Report` headers but not the literal `### Feedback Cycles` header Path-B requires (`_inline_process_complete` reads for exactly that anchor).
- `greeting.txt` was written in the worktree but never merged back to the test-project root. Path-B's filesystem check reads the test-project root, misses it.
- FO exits code=0 after archiving (cleanly) → new assertion fires: "FO subprocess exited (code=0) before either Path-A or Path-B signal was observed."

## Proposed approach

**Near-term (ship in #190's PR before #132 merges):** permanent `@pytest.mark.xfail(strict=False)` guarding bare-mode haiku on BOTH test_gate_guardrail and test_feedback_keepalive, with reason strings citing this task (#200). The guard must treat `haiku`, `claude-haiku-4-5`, and concrete runtime variants such as `claude-haiku-4-5-20251001` as the same class. This silences the currently-visible failures without weakening the test assertions themselves — stronger models (haiku-teams, opus-*) still exercise the full protocol.

**Medium-term (this task's scope):** investigate whether FO prose changes (shared-core dispatch contract, runtime adapter, skill prompt) could improve haiku-bare reliability on these two shapes. Concrete candidates:
- For Pattern A: sharpen startup-procedure prose so haiku less readily cd's away from the test project; reinforce that `status --discover` must run in the test-project cwd. Possibly add a `startup-sanity-check` helper that fails loudly if the cwd doesn't contain the workflow dir.
- For Pattern B: add explicit prose "use `subagent_type=spacedock:ensign` when dispatching stage workers"; warn against nesting `SendMessage` calls inside `Agent` prompts; and reinforce the `### Feedback Cycles` heading as the load-bearing audit anchor (not `## Stage Report`).

**Long-term (out of scope for this task):** if FO prose changes don't close the gap, the pragmatic answer is "haiku-bare is not a supported combination for these tests" — retire the bare-haiku variants and let other combos (bare-opus, teams-haiku, teams-opus) carry the coverage.

## Acceptance criteria

**AC-1 — Near-term xfail lands with #190 PR (#132).**
The xfail guard in both tests uses a single classifier shape — `"haiku" in model.lower()` — so it classifies the pytest alias `haiku`, the canonical id `claude-haiku-4-5`, and concrete runtime variants like `claude-haiku-4-5-20251001` as the same xfail class. No alias-by-alias `or` chains, no separate guards per variant. The reason string cites this task (`#200`).

Verified by:
- `grep -nE '"haiku" in model\.lower\(\)' tests/test_gate_guardrail.py tests/test_feedback_keepalive.py` returns at least one match per file.
- `grep -nE '#200' tests/test_gate_guardrail.py tests/test_feedback_keepalive.py` returns at least one match per file (reason-string citation).
- Local invocation classifies all three variants as XFAIL (not FAILED) on bare mode:
  - `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model haiku --effort low -v`
  - `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model claude-haiku-4-5 --effort low -v`
  - `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model claude-haiku-4-5-20251001 --effort low -v`
  - Same three invocations against `tests/test_feedback_keepalive.py`.
- PR #132 CI goes green on claude-live-bare.

**AC-2 — Pattern A root cause documented with reproducible evidence.**
Verified by: the entity body's Pattern A section cites specific fo-log line numbers from a named artifact + reproduces the `{PWD}` brace-bug under a minimal haiku-bare invocation.

**AC-3 — Pattern B root cause documented with reproducible evidence.**
Verified by: the entity body's Pattern B section cites specific fo-log lines showing (a) `subagent_type=None` validation dispatch, (b) `SendMessage` string nested inside `Agent` prompt, (c) missing `### Feedback Cycles` header in archived entity, (d) `greeting.txt` in worktree but not in test-project root.

**AC-4 — Medium-term FO prose proposal or explicit defer.**
Verified by: either a concrete before/after diff proposed for `skills/first-officer/SKILL.md` or `references/claude-first-officer-runtime.md` addressing Pattern A + Pattern B, OR a documented decision to defer (e.g., "haiku-bare weaknesses not worth FO prose surgery; retire bare-haiku coverage instead — filed as follow-up").

**AC-5 — Static suite green post-merge.**
Verified by: `make test-static` passes on main after implementation.

**AC-6 — Opus extended-thinking text does NOT bypass `test_gate_guardrail`'s self-approval scrub regex.**
Verified by: the self-approval-scrub logic in `tests/test_gate_guardrail.py` (around lines 100-108, the regex matching past-tense "approved.*advancing") handles opus extended-thinking content that may contain strings like `<thinking>approved ... advancing</thinking>`. Either the regex is updated to strip `<thinking>` blocks before matching, OR the `fo_text_output` capture strips thinking content upstream, OR an opus-specific xfail is added with a reason citing this AC.

**Evidence motivating AC-6:** PR #132 re-run (CI run `24612094887`) claude-live-opus showed `test_gate_guardrail` failing 2/6 checks. Entity state was correct (held at `status: work`, no archive, gate review text present), but two checks failed — most plausibly the self-approval-scrub false-positive on opus extended-thinking text. Different check count from bare (2/6 vs 1/7) indicates distinct failure paths per model/context. Artifact: `spacedock-test-zxuaa3uo/fo-log.jsonl` (opus); compare with `spacedock-test-2izdmkjp/fo-log.jsonl` (haiku-bare Pattern A).

## Test plan

All verification commands run **locally** — captain has explicitly requested local-only verification for this task. No CI round-trip required for any AC except the green-CI claim in AC-1.

- **AC-1 (static portion):**
  - `grep -nE '"haiku" in model\.lower\(\)' tests/test_gate_guardrail.py tests/test_feedback_keepalive.py`
  - `grep -nE '#200' tests/test_gate_guardrail.py tests/test_feedback_keepalive.py`
  Both must return at least one match per file.
- **AC-1 (behavioral, optional ~$0.50):** three local invocations per test confirm the alias-normalization works end-to-end (each must report XFAIL, not FAILED):
  - `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model haiku --effort low -v`
  - `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model claude-haiku-4-5 --effort low -v`
  - `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model claude-haiku-4-5-20251001 --effort low -v`
  - Same three against `tests/test_feedback_keepalive.py`.
- **AC-2 / AC-3 (root-cause documentation):** static — verified by reading the entity body's Pattern A / Pattern B sections. No command needed beyond `grep -n 'Pattern A\|Pattern B' docs/plans/haiku-bare-fo-guardrail-weaknesses.md`.
- **AC-4 (medium-term proposal or defer):** static — entity body contains either a concrete before/after diff for the named files or an explicit defer decision. No live runs required.
- **AC-5 (static suite green):** `make test-static` from repo root.
- **AC-6 (opus extended-thinking scrub):** static portion — `grep -nE 'thinking|<thinking>' tests/test_gate_guardrail.py` to confirm the chosen mitigation (regex update, upstream strip, or opus xfail) is present and cites this AC. Optional behavioral confirmation: `unset CLAUDECODE && uv run pytest tests/test_gate_guardrail.py --runtime claude --model opus --effort high -v` reports the in-scope case as PASS or XFAIL (not FAILED).

## Out of scope

- Fixing haiku-bare FO reliability on other tests. This task scope is strictly test_gate_guardrail + test_feedback_keepalive.
- Retiring bare-haiku coverage across the test suite (long-term option; separate task if the medium-term approach fails).
- FO prose changes that affect runtime behavior across ALL models — this task's prose changes (if any) must target haiku-bare without regressing opus/haiku-teams.

## Cross-references

- **#185** — cycle 3 added the original narrow bare+haiku xfail on test_feedback_keepalive that #190 removed. This task effectively restores it (but framed differently).
- **#186** — fixed test_gate_guardrail model-fixture plumbing, making #200's Pattern A visible.
- **#190** — sharpened test_feedback_keepalive diagnostic, making #200's Pattern B visible.
- **#194** — multi-model FO-side standing-teammate-spawn flake. Adjacent but distinct; do NOT absorb.
- **#160** — haiku FO multi-stage dispatch compression. Related FO-tool-shape weakness on haiku; may share root cause with Pattern B.
- Artifacts: CI run `24610475442`, job `71963876313` (claude-live-bare). Test dirs `spacedock-test-2izdmkjp` (gate_guardrail) + `spacedock-test-h42ehkks` (keepalive).

## Stage Report: ideation

- DONE: Test plan names a concrete LOCAL test command per AC (especially AC-1's grep and AC-5's `make test-static`) so an implementer can verify each acceptance criterion locally without round-tripping through CI.
  Test plan rewritten with explicit `unset CLAUDECODE && uv run pytest ...` invocations per AC, plus `grep` commands for AC-1/AC-2/AC-3/AC-6 static portions and `make test-static` for AC-5.
- DONE: AC-1's xfail-classifier shape is concretely specified: a single decorator/marker pattern that classifies `haiku`, `claude-haiku-4-5`, and concrete runtime variants as the same xfail class — no duplicated alias-by-alias guards. The pytest-marker normalization gap noted in the entity body is addressed.
  AC-1 rewritten to pin the classifier as `"haiku" in model.lower()` (single expression matching all three variants); `grep -nE` verification command included. This closes the `test_gate_guardrail.py:41` brittle equality gap (`model == "claude-haiku-4-5"` currently misses `haiku` alias and `claude-haiku-4-5-20251001`).
- DONE: AC items remain end-state properties, not stage actions.
  Audited all six AC headlines — AC-1 "xfail lands", AC-2/AC-3 "root cause documented", AC-4 "proposal or defer", AC-5 "suite green", AC-6 "text does NOT bypass" — all read as end-state properties. No imperative verb-phrase rewrites needed.

### Summary

Refined the existing fleshed-out task body without rewriting it. Two surgical edits: (1) AC-1 now pins the xfail classifier to a single `"haiku" in model.lower()` expression that handles all three model-name variants the captain enumerated, addressing the alias-normalization gap that PR #159 surfaced; (2) Test plan now lists concrete local `uv run pytest` and `grep` commands per AC so implementer verification stays local per captain instruction. AC items already read as end-state properties; no rewrite needed.
