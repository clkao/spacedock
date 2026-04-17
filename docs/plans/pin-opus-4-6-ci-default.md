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

## Stage Report (validation)

### Checklist

1. **Read entity body in full** — DONE. Reviewed `## The change`, `## Acceptance criteria`, and the implementer's `## Stage Report (implementation)` end-to-end.
2. **Verify diff is surgical and only touches the claude-live-opus job's MODEL_OVERRIDE-fallback expressions** — DONE.

   Verbatim diff from `git diff main..HEAD -- .github/workflows/runtime-live-e2e.yml`:
   ```diff
   diff --git a/.github/workflows/runtime-live-e2e.yml b/.github/workflows/runtime-live-e2e.yml
   index 3e82d81f..2791e0eb 100644
   --- a/.github/workflows/runtime-live-e2e.yml
   +++ b/.github/workflows/runtime-live-e2e.yml
   @@ -485,7 +485,8 @@ jobs:
              if [ -n "$MODEL_OVERRIDE" ]; then
                EFFECTIVE_MODEL="$MODEL_OVERRIDE"
              else
   -            EFFECTIVE_MODEL="opus"
   +            # Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.
   +            EFFECTIVE_MODEL="claude-opus-4-6"
              fi
              {
                echo "### Tool versions"
   @@ -515,7 +516,8 @@ jobs:
                if [ -n "$MODEL_OVERRIDE" ]; then
                  MODEL_FLAG="$MODEL_OVERRIDE"
                else
   -              MODEL_FLAG="opus"
   +              # Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.
   +              MODEL_FLAG="claude-opus-4-6"
                fi
                echo "Opus effort: $EFFORT_FLAG"
                echo "Opus model: $MODEL_FLAG"
   ```

   Confirmed:
   - (a) Only `.github/workflows/runtime-live-e2e.yml` is touched in our two commits (`ca21f49f` workflow change + `bd3762dd` entity body). `git log main..HEAD --stat` shows the workflow commit modifies only this file (4 insertions, 2 deletions).
   - (b) Both changed lines are inside the `claude-live-opus` job's MODEL_OVERRIDE-fallback `else` branches (lines 488 and 518 in the original).
   - (c) The `if [ -n "$MODEL_OVERRIDE" ]; then ...; else ...; fi` structure is preserved verbatim — `model_override` workflow input still takes precedence.
   - (d) Each inline comment cites `#177 / #181` and includes the reversibility note (`Reversible — restore default to opus once upstream resolves`).

3. **Re-run static suite** — DONE. Ran `unset CLAUDECODE && make test-static` from the worktree root.

   Final line verbatim:
   ```
   426 passed, 22 deselected, 10 subtests passed in 20.02s
   ```
   Zero failures. Matches the implementer's run (426/426). AC #2 (≥ 422) satisfied.

4. **Verify no other files/jobs touched** — DONE. `git diff --name-only main..HEAD` shows three paths:
   - `.github/workflows/runtime-live-e2e.yml` — our workflow change.
   - `docs/plans/pin-opus-4-6-ci-default.md` — the entity body (allowed).
   - `docs/plans/opus-4-7-ensign-hallucination-scope.md` — NOT from our commits; provenance is commit `59a0db41` (`advance: #177 entering implementation …`), present in the worktree base but absent from local `main` ref. Verified by `git log main..HEAD -- docs/plans/opus-4-7-ensign-hallucination-scope.md` returning empty for our two-commit range. Out of scope; not introduced by #181.

   Within `.github/workflows/runtime-live-e2e.yml`, only the `claude-live-opus` job is touched. Other jobs (`claude-live`, `claude-live-bare`, `codex-live`, `static-offline`) untouched.

5. **AC-1, AC-2, AC-3 confirmation** — DONE.
   - **AC-1 (workflow YAML edit is surgical):** PASS — see step 2 evidence; only the `claude-live-opus` job's two fallback defaults change.
   - **AC-2 (static suite passes, ≥ 422):** PASS — 426 passed (see step 3).
   - **AC-3 (inline comment present, links #177 + #181, reversibility note):** PASS — both inserted comments include `see #177 / #181` and `Reversible — restore default to opus once upstream resolves` (see step 2 diff).

6. **AC-4 (live CI dispatch verification)** — DEFERRED-TO-MERGE-HOOK. AC-4 requires pushing the branch and dispatching a live CI run, which needs captain approval per the pr-merge mod's PR APPROVAL GUARDRAIL. The push will happen at the merge hook after captain approves the validation gate; CI will dispatch as part of normal PR flow. Not exercised in this validation; not pushing the branch; not dispatching CI.

   Implementer's flag re-stated for the merge-hook step: the YAML's `--model "$MODEL_FLAG"` path is only exercised when `test_selector` is non-empty. The no-selector path delegates to `make test-live-claude-opus`, whose hardcoded `--model opus` is intentionally out of scope per AC #1. To exercise the new pin via AC-4 the dispatch should set `test_selector` to a representative live-claude opus test path.

7. **Stage report appended** — DONE (this section).

8. **Commit on worktree branch** — pending immediately after this write; not pushed.

Recommendation: PASSED — diff is surgical, static suite 426/426 green, AC-4 deferred to merge hook.

## Stage Report (implementation, scope expansion)

Captain-directed scope expansion: the original narrow-scope work (commits `ca21f49f` workflow + `bd3762dd` entity body) only pinned the YAML's `--model "$MODEL_FLAG"` selector path. The validator flagged that the no-selector PR-trigger path delegates to `make test-live-claude-opus`, which has its own hardcoded `--model opus` and bypasses the pin entirely. This expansion adds the make-target fix on top.

### Checklist

1. **Read entity body in full (incl. prior implementation + validation reports)** — DONE. Both prior reports reviewed end-to-end. Original narrow scope confirmed shipped on this branch; no-selector path gap confirmed flagged.

2. **Confirm captain-directed scope-expansion context** — DONE. Original work stays untouched; this dispatch ADDS the make-target fix on top. PR #116 stays open; new commits get added.

3. **Inspect Makefile `test-live-claude-opus` target + workflow no-selector invocation** — DONE.
   - `Makefile` lines 30-40 (pre-edit): `test-live-claude-opus` hardcoded `--model opus` in two pytest invocations (serial + parallel).
   - `.github/workflows/runtime-live-e2e.yml` line 527 (pre-edit): no-selector branch ran `make test-live-claude-opus` with no model arg, so the make target's hardcoded `opus` won — the YAML's MODEL_FLAG was computed but unused on this path.

4. **Identify smallest change** — DONE. Shape from spec applied:
   - Makefile: introduce `OPUS_MODEL ?= claude-opus-4-6` (defaults to opus-4-6 directly, NOT the broken `opus` alias) and substitute `--model opus` → `--model $(OPUS_MODEL)` in the two pytest invocations of `test-live-claude-opus`. Override capability preserved: `make test-live-claude-opus OPUS_MODEL=opus` re-tests on opus-4-7.
   - Workflow YAML: hoist the `MODEL_OVERRIDE`-fallback computation above the `if [ -n "$TEST_SELECTOR" ]` branch so MODEL_FLAG is available to both branches; pass `OPUS_MODEL="$MODEL_FLAG"` to `make test-live-claude-opus` on the no-selector branch. The selector branch's behavior is unchanged (still uses `--model "$MODEL_FLAG"`).

5. **Apply changes with inline comments** — DONE. Inline comments added in both files citing #177 / #181 + reversibility note + override syntax.

   **Makefile diff:**
   ```diff
   @@ -5,6 +5,8 @@ TEST ?= tests/
    RUNTIME ?= claude
    LIVE_CLAUDE_WORKERS ?= 4
    LIVE_CODEX_WORKERS ?= 4
   +# Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — override with `make test-live-claude-opus OPUS_MODEL=opus` to re-test on opus-4-7.
   +OPUS_MODEL ?= claude-opus-4-6

    test-static:
    	unset CLAUDECODE && uv run pytest tests/ --ignore=tests/fixtures \
   @@ -32,9 +34,9 @@ test-live-claude:
    test-live-claude-opus:
    	unset CLAUDECODE && { \
    	  uv run pytest tests/ --ignore=tests/fixtures \
   -	    -m "live_claude and serial" --runtime claude --model opus --effort low -x -v ; SEQ=$$? ; \
   +	    -m "live_claude and serial" --runtime claude --model $(OPUS_MODEL) --effort low -x -v ; SEQ=$$? ; \
    	  uv run pytest tests/ --ignore=tests/fixtures \
   -	    -m "live_claude and not serial" --runtime claude --model opus --effort low \
   +	    -m "live_claude and not serial" --runtime claude --model $(OPUS_MODEL) --effort low \
    	    -n $(LIVE_CLAUDE_WORKERS) -v ; PAR=$$? ; \
    	  test $$SEQ -eq 0 -a $$PAR -eq 0 ; \
    	}
   ```

   **Workflow YAML diff (`runtime-live-e2e.yml`, claude-live-opus job's "Run Claude live suite (opus)" step):**
   ```diff
   -          if [ -n "$TEST_SELECTOR" ]; then
   -            echo "Running selector: $TEST_SELECTOR"
   -            if [ -n "$EFFORT_OVERRIDE" ]; then
   -              EFFORT_FLAG="$EFFORT_OVERRIDE"
   -            else
   -              EFFORT_FLAG="low"
   -            fi
   -            if [ -n "$MODEL_OVERRIDE" ]; then
   -              MODEL_FLAG="$MODEL_OVERRIDE"
   -            else
   -              # Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.
   -              MODEL_FLAG="claude-opus-4-6"
   -            fi
   -            echo "Opus effort: $EFFORT_FLAG"
   -            echo "Opus model: $MODEL_FLAG"
   -            unset CLAUDECODE
   -            uv run pytest "$TEST_SELECTOR" --runtime claude --model "$MODEL_FLAG" --effort "$EFFORT_FLAG" -v
   -          else
   -            make test-live-claude-opus
   -          fi
   +          if [ -n "$MODEL_OVERRIDE" ]; then
   +            MODEL_FLAG="$MODEL_OVERRIDE"
   +          else
   +            # Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — restore default to opus once upstream resolves.
   +            MODEL_FLAG="claude-opus-4-6"
   +          fi
   +          if [ -n "$TEST_SELECTOR" ]; then
   +            echo "Running selector: $TEST_SELECTOR"
   +            if [ -n "$EFFORT_OVERRIDE" ]; then
   +              EFFORT_FLAG="$EFFORT_OVERRIDE"
   +            else
   +              EFFORT_FLAG="low"
   +            fi
   +            echo "Opus effort: $EFFORT_FLAG"
   +            echo "Opus model: $MODEL_FLAG"
   +            unset CLAUDECODE
   +            uv run pytest "$TEST_SELECTOR" --runtime claude --model "$MODEL_FLAG" --effort "$EFFORT_FLAG" -v
   +          else
   +            # Propagate the pinned/overridden model to the make target's OPUS_MODEL var so the no-selector path also picks up claude-opus-4-6 (or the override). See #177 / #181.
   +            echo "Opus model: $MODEL_FLAG"
   +            make test-live-claude-opus OPUS_MODEL="$MODEL_FLAG"
   +          fi
   ```

   Note: the YAML edit hoists the `MODEL_OVERRIDE`-fallback `if/else` above the `TEST_SELECTOR` branch so MODEL_FLAG is computed once and used by both paths. The validated MODEL_OVERRIDE-fallback expression is preserved verbatim (same comment, same default value); only the structural location changed. The selector branch's behavior is byte-equivalent post-hoist.

6. **Static suite** — DONE. Ran `unset CLAUDECODE && make test-static` from worktree root.

   Final line verbatim:
   ```
   426 passed, 22 deselected, 10 subtests passed in 19.70s
   ```
   Zero failures. Matches prior baseline (426/426). AC #2 (≥ 422) satisfied.

7. **Verify make-target effective model resolution** — DONE.

   Default (`make -n test-live-claude-opus 2>&1 | head -10`):
   ```
   unset CLAUDECODE && { \
   	  uv run pytest tests/ --ignore=tests/fixtures \
   	    -m "live_claude and serial" --runtime claude --model claude-opus-4-6 --effort low -x -v ; SEQ=$? ; \
   	  uv run pytest tests/ --ignore=tests/fixtures \
   	    -m "live_claude and not serial" --runtime claude --model claude-opus-4-6 --effort low \
   	    -n 4 -v ; PAR=$? ; \
   	  test $SEQ -eq 0 -a $PAR -eq 0 ; \
   	}
   ```
   Confirms `--model claude-opus-4-6` in both pytest invocations.

   Override (`make -n test-live-claude-opus OPUS_MODEL=foo 2>&1 | head -10`):
   ```
   unset CLAUDECODE && { \
   	  uv run pytest tests/ --ignore=tests/fixtures \
   	    -m "live_claude and serial" --runtime claude --model foo --effort low -x -v ; SEQ=$? ; \
   	  uv run pytest tests/ --ignore=tests/fixtures \
   	    -m "live_claude and not serial" --runtime claude --model foo --effort low \
   	    -n 4 -v ; PAR=$? ; \
   	  test $SEQ -eq 0 -a $PAR -eq 0 ; \
   	}
   ```
   Confirms `--model foo`. Override capability preserved.

8. **Stage report appended** — DONE (this section).

9. **Commit on worktree branch** — pending immediately after this write; not pushed (FO handles push at re-merge boundary).

10. **Validator re-verification note** — DONE. The validator now needs to re-verify BOTH paths:
    - **YAML-direct path** (`test_selector` non-empty): the previous validation's PASSED still holds for the structural pin; minor structural hoist of the `MODEL_OVERRIDE`-fallback `if/else` above the `TEST_SELECTOR` branch should be confirmed byte-equivalent in the selector branch's effective behavior.
    - **Make-target path** (no `test_selector`, the typical PR-trigger path): new in this expansion. Validator should confirm `make -n test-live-claude-opus` shows `--model claude-opus-4-6` and that the workflow's no-selector branch passes `OPUS_MODEL="$MODEL_FLAG"` to make. AC-4 live dispatch (if exercised) should now show opus-4-6 stamps in fo-log.jsonl on EITHER trigger path.

### Files changed (this expansion)

- `Makefile` (3 insertions, 2 deletions: `OPUS_MODEL` variable + 2 pytest substitutions in `test-live-claude-opus`)
- `.github/workflows/runtime-live-e2e.yml` (5 insertions, 2 deletions in claude-live-opus job's "Run Claude live suite (opus)" step: hoist MODEL_OVERRIDE-fallback above TEST_SELECTOR branch + propagate `OPUS_MODEL="$MODEL_FLAG"` on no-selector branch + 1 inline comment)

### Summary

Pinned the `make test-live-claude-opus` target's default model to `claude-opus-4-6` via a new `OPUS_MODEL ?= claude-opus-4-6` Makefile variable (overridable as `make test-live-claude-opus OPUS_MODEL=opus`); hoisted the workflow's `MODEL_OVERRIDE`-fallback above the `TEST_SELECTOR` branch and propagated the resolved MODEL_FLAG to make as `OPUS_MODEL="$MODEL_FLAG"`, so the no-selector PR-trigger path (which delegates to `make test-live-claude-opus`) also picks up claude-opus-4-6 by default. Static suite at 426 passed; `make -n` confirms default expands to `--model claude-opus-4-6` and `OPUS_MODEL=foo` override expands to `--model foo`. Validator should re-verify both YAML-direct and make-target paths now.
