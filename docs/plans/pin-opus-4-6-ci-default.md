---
id: 181
title: "Pin --model claude-opus-4-6 in CI workflow defaults (until upstream opus-4-7 regression resolves)"
status: validation
source: "from #177 + #178 cluster — opus-4-7 regression at low/medium effort makes claude-live-opus CI job unreliable. #178's boilerplate mitigation falsified by #177. This is the temporary unblocker for #172 PR #107 and any other PR whose CI runs hit opus-4-7 default. Reversible — pinning is a workflow-default change, not a code change; explicit model_override still works."
started: 2026-04-17T03:56:01Z
completed:
verdict:
score: 0.6
worktree: .worktrees/spacedock-ensign-pin-opus-4-6-ci-default
issue:
pr:
mod-block: merge:pr-merge
---

## Why this matters

172's PR #107 has 4/5 jobs green; the 5th (claude-live-opus) fails on opus-4-7 (the regression #177 is investigating). #177's #178-boilerplate mitigation was falsified at low/medium effort. The cleanest unblocker for 172 — and any other PR sitting in the same cluster — is to pin `claude-live-opus`'s effective model to `claude-opus-4-6` until upstream resolves the opus-4-7 hallucination.

This is reversible: it's a workflow-default change, not a code or test change. Explicit `model_override` workflow input still works for anyone who wants to test opus-4-7 directly.

## The change

Edit `.github/workflows/runtime-live-e2e.yml`:
- Locate the `claude-live-opus` job's model resolution (the input default or the env var that resolves to `--model opus`).
- Change the default from `opus` (which now resolves to `claude-opus-4-7` under Claude Code 2.1.111+) to `claude-opus-4-6` explicitly.
- Preserve `model_override` workflow input handling so future testers can override back to `opus` or any other value.

The `model_override` plumbing into `claude -p` already works end-to-end (verified by #179's fix; confirmed by #180's audit).

## Acceptance criteria

1. **Workflow YAML edit is surgical.** Only the `claude-live-opus` job's default model resolution changes. Other jobs (`claude-live`, `claude-live-bare`, `codex-live`, `static-offline`) untouched.
2. **Static suite passes.** `make test-static` from the worktree root, ≥ 422 passed (current main baseline).
3. **The pin is documented in the workflow file.** Inline comment explains why pinning, links to #177 + this entity.
4. **Verification: dispatch one CI run on the worktree branch with no model_override input.** The `claude-live-opus` job should now run on `claude-opus-4-6` (verify via `gh run download` + grep `assistant.message.model` from the run's `fo-log.jsonl`).

## Test plan

- Static: `make test-static` from worktree root.
- Live: one CI dispatch on the worktree branch via `gh workflow run runtime-live-e2e.yml --ref spacedock-ensign/pin-opus-4-6-ci-default -f claude_version=2.1.111` (no model_override). Capture run URL, claude-live-opus job conclusion, and model stamps from fo-log.jsonl.
- Total cost: ~5 min CI minutes (one focused dispatch).

## Out of Scope

- Any change to the `--model` default in test files (test_standing_teammate_spawn.py and friends) — those already correctly read pytest's `--model` option per #179.
- Pinning `claude-live` or any non-opus job (only opus is broken).
- Reverting the pin once upstream fixes opus-4-7 — that's a future task triggered by upstream signal.
- Long-term decision on whether opus-4-6 should be the *permanent* default — this is a temporary pin pending upstream.

## Cross-references

- #177 — root cause investigation (now repurposed to Layer 2 mitigation experiments)
- #178 — falsified low/medium-effort boilerplate mitigation (likely won't ship in current form)
- #172 — direct beneficiary; PR #107 will reach 5/5 green CI once this lands and 172 rebases
- #179 — landed the model_override plumbing that makes this pin work end-to-end
- #180 — closed REJECTED-duplicate (audit confirming #179 covers all live tests)

## Stage Report (implementation)

### Checklist

1. **Read entity body in full** — DONE. Spec is complete: `## The change`, `## Acceptance criteria`, `## Test plan` all present and unambiguous.
2. **Inspect workflow file, locate claude-live-opus model resolution** — DONE. `claude-live-opus` job spans lines 367-534 of `.github/workflows/runtime-live-e2e.yml`. Two literal `"opus"` defaults found inside the job: (a) line 488 in the "Show tool versions" step (`EFFECTIVE_MODEL="opus"` for the step summary), (b) line 518 in the "Run Claude live suite (opus)" step (`MODEL_FLAG="opus"`, the value actually passed as `--model "$MODEL_FLAG"` on line 523). Resolution chain: `workflow_dispatch.inputs.model_override` → job step `env: MODEL_OVERRIDE: ${{ inputs.model_override }}` → shell `if [ -n "$MODEL_OVERRIDE" ]; then ...; else MODEL_FLAG="opus"; fi`. No other `opus` defaults in the job; no env-var-level default.
3. **Identify smallest change** — DONE. Shape (a) from the spec applied: change the in-shell default values from `"opus"` to `"claude-opus-4-6"`. Both the display string (line 488) and the actual `--model` flag (line 518) updated for consistency. Workflow structure, other jobs, and `model_override` plumbing untouched. Total diff: 4 insertions, 2 deletions, 1 file.
4. **Apply change with inline comment** — DONE. Two comment lines added (one above each changed default), both reading: `# Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.`

   Before/after at line 488 (display default in "Show tool versions" step):
   ```
   - EFFECTIVE_MODEL="opus"
   + # Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.
   + EFFECTIVE_MODEL="claude-opus-4-6"
   ```

   Before/after at line 518 (actual `--model` flag in "Run Claude live suite (opus)" step):
   ```
   - MODEL_FLAG="opus"
   + # Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.
   + MODEL_FLAG="claude-opus-4-6"
   ```

5. **Static suite** — DONE. Ran `unset CLAUDECODE && make test-static` from worktree root.

   Final line verbatim:
   ```
   426 passed, 22 deselected, 10 subtests passed in 20.35s
   ```
   Zero failures. Above the AC #2 baseline of ≥ 422 passed (matches/exceeds main).
6. **Commit on worktree branch** — DONE. Commit `ca21f49f` on `spacedock-ensign/pin-opus-4-6-ci-default` with message `fix: #181 pin claude-live-opus to claude-opus-4-6 default (opus-4-7 regression workaround)`. Not pushed (FO handles push at merge boundary).
7. **Stage report written** — DONE (this section).
8. **Validator AC-4 dispatch note** — DONE. Validator should dispatch one CI run on `spacedock-ensign/pin-opus-4-6-ci-default` (no `model_override` input) and confirm `claude-live-opus` runs on `claude-opus-4-6` via `gh run download` + grep `assistant.message.model` from `fo-log.jsonl`. **IMPORTANT FLAG FOR VALIDATOR:** the YAML's `--model "$MODEL_FLAG"` path is only exercised when `test_selector` is non-empty (lines 508-523). When `test_selector` is empty (the typical PR-trigger and unselected dispatch path), the workflow falls through to `make test-live-claude-opus` at line 525, and that Makefile target (Makefile lines 32-40) **still hardcodes `--model opus`** — out of scope per AC #1 ("only the workflow YAML edit"). To exercise the new pin via AC-4, the validator's dispatch should set `test_selector` to a representative live-claude opus test (e.g. an `@pytest.mark.live_claude` test path) so the YAML's MODEL_FLAG branch executes. If the validator wants the pin active on the default (no-selector) make path as well, that needs a follow-up entity touching the Makefile target.

### Files changed

- `.github/workflows/runtime-live-e2e.yml` (4 insertions, 2 deletions; both edits inside `claude-live-opus` job)

### Summary

Pinned the `claude-live-opus` job's default model from `opus` (which now resolves to `claude-opus-4-7` under Claude Code 2.1.111+) to explicit `claude-opus-4-6` by changing two literal defaults in the job's shell steps; `model_override` workflow input is preserved and remains the only escape hatch. Inline comments link to #177 / #181 with a reversibility note. Static suite at 426 passed (≥ 422 baseline). Validator should exercise AC-4 via a dispatch with `test_selector` set, since the no-selector path delegates to `make test-live-claude-opus` whose own hardcoded `--model opus` is intentionally out of scope per AC #1.
