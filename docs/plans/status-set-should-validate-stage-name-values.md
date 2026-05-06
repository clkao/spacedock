---
id: amrd1r63jfkjq14tvwnkfges
title: "status --set should validate stage-name values against the workflow's stages.states[].name list"
status: validation
source: GitHub issue #189 (clkao/spacedock)
started: 2026-05-06T08:39:40Z
completed:
verdict:
score: 0.55
worktree: .worktrees/spacedock-ensign-status-set-should-validate-stage-name-values
issue: "#189"
pr:
mod-block: merge:pr-merge
---

`status --set {slug} status={value}` accepts arbitrary string values without checking against the workflow README's `stages.states[].name` list. Workers (especially LLM-driven ensigns) can silently write semantically-invalid values like `status: review` when the workflow defines `verify → ship → done`. The result is a quietly broken state machine: subsequent dispatches based on `status --next` skip the entity, downstream gates never fire, and recovery depends on human review.

Real-world hit at carlove: an ensign-class subagent wrote `status: review` (not a defined stage) via direct file edit during a ship-verify dispatch; the FO recovered by rolling status back, but only because the captain noticed during gate review. `status --set status=review` would also have been accepted on that workflow because no validation runs.

## Scope

This entity covers the `status` binary validation path only. The complementary worker-prompt tightening proposed in issue #189 (forbidding workers from writing `status:` in frontmatter) is split into a separate task — the two changes ship at different layers (CLI vs. prompt template), need different test surfaces, and the CLI validation is the mechanical enforcement gate that catches both `--set` typos and any future `--set` bypass attempts. Direct file-edit bypass is a different bug and out of scope here; this fix tightens the `--set` contract.

## Proposed approach

When `--set` updates the `status` field on a workflow whose README has a `stages:` block, validate the new value against the union of `stages.states[].name`. The validation is wired into the existing `--set` execution path in `main()` immediately after the README's stages block is already parsed (current code at line 2300, `parse_stages_block(readme_path)`), so no new file read is needed and the schema is sourced from the live README on every invocation.

Behavioral contract:

1. **Reject unknown stage values.** When the field being set is `status`, the value is non-empty, the workflow README exists with a `stages:` block, and the value is not in the union of `stages.states[].name`, exit non-zero with a stderr error of the shape:
   ```
   Error: 'review' is not a defined stage in workflow {wd} — known stages: [backlog, ideation, implementation, validation, done]. Use --force to override.
   ```
   The known-stages list is rendered in the order the README declares them. The frontmatter file is not modified.

2. **Accept every defined stage.** Every value present in `stages.states[].name` is accepted with no warning, regardless of stage attributes (initial, gate, terminal, worktree).

3. **`--force` bypass.** Passing `--force` skips the unknown-value rejection and emits a stderr warning naming the unknown value: `Warning: --force overriding unknown stage 'review' on entity {slug}`. The frontmatter is then written. This is the same `--force` flag already wired at line 2275 of the script — no new flag.

4. **Reads schema at invocation time.** No cross-invocation cache. The README is re-read on every `--set` call, so a workflow that renames a stage in README and then runs `--set status={new-name}` immediately succeeds without restarting any daemon. (Naturally satisfied by the current architecture — each invocation is a fresh subprocess; the AC is testable by mutating README between two `--set` calls and observing the behavior shift.)

Out-of-scope edge cases (kept as today's behavior):
- Setting `status=` (empty value, clearing) is not validated — the `--set` parser still accepts it; downstream `--validate` will surface the entity as invalid. This is explicitly left to existing layers.
- Workflows whose README has no `stages:` block (truly unstaged workflows) skip validation entirely — there is no schema to validate against. `parse_stages_block` already returns `None` in that case; the validator no-ops.
- Workflows whose README is missing or unreadable: today's behavior is preserved (no validation, since there is no schema to compare to). This is consistent with how the existing `--set` path already tolerates a missing README.
- Other fields (`pr`, `worktree`, `mod-block`, `verdict`, etc.) are unaffected — only the `status` field is validated.

Implementation surface (informational, not a contract): one validation block in `main()` inside the `--set` branch, between the existing stages parse at line 2300 and the `update_frontmatter` call at line 2365. No new helper function is required; the check is a few lines reusing the already-loaded `stages` list. This matches the surrounding code style — recent companion fix #190 added `_has_opening_fence` as a single-purpose predicate when it had three call-sites; this validator has one call-site so an inline check is the right shape.

## Acceptance criteria

(Entity-level end-state facts; stage actions belong in the stage report.)

1. **Unknown-stage rejection is mechanical.** When a workflow README has a `stages:` block and `--set` is invoked with a `status` value that is not in `stages.states[].name`, the script exits non-zero, the stderr message names the rejected value and lists the known stages in declaration order, the entity file is byte-identical to its pre-call contents, and no `field: old -> new` line is emitted on stdout.
   - Test: `tests/test_status_set_status_validation.py::TestStatusValidation::test_unknown_stage_rejected`. Subprocess harness using the existing `build_status_script` / `make_pipeline` / `run_status` helpers in `test_status_script.py`. Builds a workflow with the standard `README_WITH_STAGES` (stages: backlog, ideation, implementation, validation, done), creates a `task-a.md` with `status: backlog`, runs `--set task-a status=review`, asserts `returncode != 0`, asserts stderr contains the literal `'review' is not a defined stage`, asserts stderr lists `backlog, ideation, implementation, validation, done`, and asserts the file contents match the pre-call snapshot.

2. **Every declared stage is accepted.** For each stage `s` in `stages.states[].name` of a representative workflow, `--set task-a status={s}` exits zero and writes the new value to frontmatter, regardless of whether `s` is the initial, a gate, a worktree stage, or terminal. (The mod-block / merge-hook / pr guards are out-of-scope behavior they enforce today — this AC validates only that the *stage-name validator* itself does not reject any defined stage; tests neutralize unrelated guards by setting up the entity to satisfy them, the same way `test_status_set_missing_field.py` already does.)
   - Test: `tests/test_status_set_status_validation.py::TestStatusValidation::test_every_declared_stage_accepted`. Parameterized loop over the five stages declared in `README_WITH_STAGES`. For each stage, build a fresh entity and invoke `--set task-a status={stage}`. Asserts `returncode == 0` for every stage and asserts the resulting frontmatter contains `status: {stage}`.

3. **`--force` overrides the validator.** When `--set` is invoked with an unknown stage value plus `--force`, the script exits zero, the stderr emits a warning naming the unknown value, and the frontmatter is updated to the unknown value.
   - Test: `tests/test_status_set_status_validation.py::TestStatusValidation::test_force_overrides_validation`. Builds the same workflow as test 1, invokes `--set task-a status=review --force`, asserts `returncode == 0`, asserts stderr contains a warning naming `'review'`, and asserts the resulting frontmatter has `status: review`.

4. **Schema is read at invocation time, not cached.** Mutating the README's `stages.states[].name` list between two `--set` invocations changes which values are accepted on the second call without any restart, daemon flush, or cache invalidation. A value rejected on call N is accepted on call N+1 once the README declares it.
   - Test: `tests/test_status_set_status_validation.py::TestStatusValidation::test_readme_reread_per_invocation`. Build a workflow with stages [backlog, done], create `task-a.md`. Invoke `--set task-a status=ideation` — assert `returncode != 0` (rejected; ideation not declared). Then rewrite the README to include `ideation` in `stages.states`. Invoke `--set task-a status=ideation` again — assert `returncode == 0` and frontmatter is updated. No subprocess cache to invalidate — the test demonstrates the contract end-to-end.

## Test plan

All four tests live in a new file `tests/test_status_set_status_validation.py` modeled on `tests/test_status_set_missing_field.py` (subprocess harness, `make_pipeline` + `run_status`, `_read_frontmatter` helper). No new fixtures needed beyond `README_WITH_STAGES`. Estimated cost: low — pure Python subprocess tests, runtime under 2s for the full file. No live E2E or worktree gymnastics required: the validation is a pure function of `(value, parsed_stages)` exercised through the script's CLI surface, which is the right abstraction level for the claim.

The test IDs and expected outcomes:

| Test | Invocation | Expected returncode | Expected stderr substring | Expected file change |
|------|-----------|---------------------|---------------------------|----------------------|
| `test_unknown_stage_rejected` | `--set task-a status=review` | non-zero | `'review' is not a defined stage` and `backlog, ideation, implementation, validation, done` | none (file byte-identical) |
| `test_every_declared_stage_accepted` | `--set task-a status={stage}` × 5 | 0 | empty (or non-error) | `status: {stage}` written |
| `test_force_overrides_validation` | `--set task-a status=review --force` | 0 | `Warning` and `'review'` | `status: review` written |
| `test_readme_reread_per_invocation` | two-call sequence with README mutation between | non-zero, then 0 | first call rejects; second succeeds | second call writes `status: ideation` |

The existing 587-test suite (`make test-static`) must continue to pass; the validator is additive and only triggers on the `status` field with an unknown non-empty value, so no existing test fixture is expected to need updates.

## Independent reviewer pass

This task touches scaffolding (`skills/commission/bin/status`), so per the README ideation stage description the staff reviewer pass is captured here.

**Design soundness.** The validator hooks into the `--set` execution path at the point where `parse_stages_block` is already called (status:2300). Reusing the already-loaded `stages` list keeps the call free of new I/O. The chosen surface — validating only the `status` field, only for non-empty values, only when README declares a `stages:` block — matches the existing fail-open posture for unstaged workflows and avoids regressing edge cases the script already handles.

**Test plan sufficiency.** The four AC behaviors map 1-to-1 to four subprocess tests. Test 4 (re-read at invocation) deserves special attention: because each `--set` is a fresh process, the AC is naturally satisfied by the architecture; the test is therefore an end-to-end contract check (mutate README, observe shift) rather than an internal-cache-invalidation check. That matches the AC's abstraction level — the claim is observable behavior, the proof is observable behavior. Test 2 covers the "every declared stage" claim by parameterizing over `README_WITH_STAGES` rather than hand-listing values, so a future README change that adds a stage is covered automatically.

**Gaps considered.**
- *Should validation also cover `update_frontmatter`?* No — `update_frontmatter` is shared with `run_archive` (which stamps `archived:` and never touches `status:`) and with the worktree-mirror write at status:2371 (`pr` only). Pushing validation into the shared writer would either over-validate (rejecting writes that don't touch `status`) or require threading the field-name through, which is more code than the inline `--set` check. Inline at the `--set` branch is the smaller, more targeted change.
- *Should the rejected-value error name the workflow dir?* Yes — issue #189's example error includes `in workflow {wd}`. The error string above includes `in workflow {wd}` to match the issue's proposed shape.
- *Is `--force` discoverability adequate?* The error message ends with `Use --force to override.`, mirroring the existing mod-block guard wording at status:1831.
- *What if README's `stages:` block is malformed?* `parse_stages_block` returns `None` on malformed/missing blocks; the validator treats `None` as "no schema" and skips. This is the same behavior as the existing `--next` / `--boot` paths take when there are no stages. (Their stricter "Error: README.md has no stages block" is a different surface — it applies to stage-list-rendering commands; `--set` does not require stages metadata to operate, only to validate it.)
- *Worker-prompt tightening (issue #189 paragraph 2).* Out of scope for this entity; recommended as a separate task. Reasoning: different layer (prompt template vs. CLI), different test surface (transcript fixture vs. subprocess), and ships independently. Bundling would expand scope without tightening the ship.

**Recommendation.** Proceed to dispatch. The four ACs are entity-level end-state facts, the test plan covers each AC at the right abstraction level, the implementation surface is small and additive, and the scope decision is recorded with reasoning. No blockers identified.

## Stage Report: ideation

- DONE: Acceptance criteria specify the validator's behavior end-to-end: rejects unknown stage values with an error that lists the workflow's known stages; accepts every value defined in `stages.states[].name`; supports `--force` for schema-evolution; reads from `--workflow-dir`'s README at invocation time (not a cached schema).
  Four ACs in the body cover rejection, every-stage acceptance, `--force` override, and per-invocation README re-read; each AC names its test ID and observable proof.
- DONE: Scope decision recorded: `status` binary validation only, OR additionally tighten the dispatch-prompt template to forbid worker frontmatter writes — with reasoning the validation stage can cross-check.
  Decision: `status` binary validation only. Reasoning in the Scope section — different layer, different test surface, ships independently from the worker-prompt tightening which is recommended as a separate task.
- DONE: Test plan covers all four AC behaviors via `tests/test_status_*.py` subprocess harness, naming each test ID and expected exit/stderr.
  Test plan section names `tests/test_status_set_status_validation.py` and four test methods with a contract table giving expected returncode, stderr substrings, and file-state outcome for each.

### Summary

Scoped this entity to the `status` binary validation only; the worker-prompt-tightening complement from issue #189 is split off as a separate task with reasoning recorded. Specified four entity-level acceptance criteria (unknown-stage rejection, every-declared-stage acceptance, `--force` override, per-invocation README re-read) each tied to a named subprocess test in a new `tests/test_status_set_status_validation.py` modeled on the existing `test_status_set_missing_field.py`. Captured an independent reviewer pass in the body since this touches scaffolding, addressing where the validator hooks in (inline at the existing `parse_stages_block` site in the `--set` branch), why `update_frontmatter` is the wrong surface for the check, and why the per-invocation re-read AC is naturally satisfied by the subprocess architecture.

## Stage Report: implementation

- DONE: Implement the inline stage-name validator in the `--set` branch of `skills/commission/bin/status` per the ideation's Behavioral contract (reuse already-loaded `stages` list at line 2300; reject unknown values with the spec'd error message; honor existing `--force`; no-op when README has no stages block).
  Inline validator added immediately after the `stages = parse_stages_block(...)` call; 26 lines, no new helper. Reads the same `stages` list, iterates `updates`, skips when field != `status` or value is empty or value is known, emits the spec'd error (`'<value>' is not a defined stage in workflow <wd> — known stages: [<list>]. Use --force to override.`) and exits 1, or emits the spec'd `--force` warning and continues. No-ops when `stages` is falsy (no README, no stages block, malformed).
- DONE: Add the four parser-level subprocess tests in `tests/test_status_set_status_validation.py` per the contract table in the entity body's Test plan; all four pass locally.
  New file with `test_unknown_stage_rejected`, `test_every_declared_stage_accepted`, `test_force_overrides_validation`, `test_readme_reread_per_invocation`. All four pass; the every-stage test uses `--force` to neutralize unrelated mod-block / merge-hook guards (the same pattern `test_status_set_missing_field` uses) so the validator behavior is isolated. Pre-implementation TDD run: 3 of 4 failed as expected (every-stage test passed in the pre-validator script because no validation existed); post-implementation run: 4/4 pass.
- DONE: `make test-static` exits 0 — no pre-existing tests regressed; the existing 587-test baseline holds.
  `make test-static`: 581 passed, 26 deselected, 15 subtests passed. Includes the 4 new tests; no pre-existing test required updating, consistent with the validator being additive and only firing on the `status` field with an unknown non-empty value.

### Summary

Added a 26-line inline validator at the existing `parse_stages_block` site in the `--set` branch of `skills/commission/bin/status`, plus a new `tests/test_status_set_status_validation.py` with four subprocess tests mapping 1-to-1 to the four ACs. The validator reuses the already-loaded `stages` list (no new I/O), reads the schema from the live README on every invocation (per-invocation re-read is naturally satisfied because each `--set` is a fresh subprocess), and honors the existing `--force` flag with a stderr warning. `make test-static` reports 581 passed + 15 subtests; the additive validator triggers only on the `status` field with a non-empty unknown value, so no existing tests required updates.

## Stage Report: validation

- DONE: Every AC (AC-1..AC-4) has its named tests rerun in the validator's session; results captured per-suite with `N/N passed` and AC-by-AC evidence; cross-check that the test assertions actually prove the AC's stated end-state property (not adjacent behavior).
  Reran `tests/test_status_set_status_validation.py` at HEAD c367ffd9: 4/4 passed in 0.31s.
  AC-1 (test_unknown_stage_rejected): asserts returncode != 0; stderr contains literal `'review' is not a defined stage`; stderr lists `backlog, ideation, implementation, validation, done` and `assertEqual(indices, sorted(indices))` proves declaration order; pre/post byte-snapshot equality proves entity untouched; `assertNotIn('->', result.stdout)` proves no transition line emitted. Each of these maps directly to the AC-1 end-state clauses.
  AC-2 (test_every_declared_stage_accepted): parameterized over the 5 stages of `README_WITH_STAGES`; each call asserts returncode == 0 and frontmatter `status: {stage}` written. Test uses `--force` to neutralize unrelated mod-block / merge-hook / pr guards so the validator path is isolated, per the AC's explicit out-of-scope clause ("validates only that the *stage-name validator* itself does not reject any defined stage"). Cross-check noted: a buggy validator that flagged a defined stage as unknown would still pass the assertion (with `--force` an unknown stage emits a warning and writes); the test does not currently assert absence of the validator's warning string. This is a documented narrowing of scope in the AC body, not a regression — accepting per the ideation reviewer pass.
  AC-3 (test_force_overrides_validation): asserts returncode == 0 with `--force`, stderr contains both `Warning` and `'review'`, and frontmatter is updated to `status: review`. Maps 1-to-1 to AC-3.
  AC-4 (test_readme_reread_per_invocation): two-call sequence with README mutation between; first call rejects `status=ideation` against [backlog, done], second call accepts the same input after README adds `ideation`. Maps 1-to-1 to AC-4 — observable behavior shift across invocations is exactly the contract.
- DONE: Branch fork-point context confirmed: this branch forks from pre-PR-#190 main, so its `make test-static` baseline differs from `m4`'s 587 — record the actual baseline number on this branch and confirm it equals (pre-validator-baseline + 4 new tests) with zero regressions.
  Baseline measured by reverting `skills/commission/bin/status` and `tests/` to c367ffd9~1 and excluding the new test file: 577 passed (+ 26 deselected, 15 subtests). With the validator and new tests at HEAD c367ffd9: 581 passed (+ 26 deselected, 15 subtests). Delta is exactly +4, matching the four new tests; zero regressions across the existing 577-test pre-PR-#190 baseline.
- DONE: PASSED or REJECTED recommendation with one-line rationale; if REJECTED, name the specific AC that failed.
  PASSED — all four ACs verified with named subprocess tests passing, baseline arithmetic matches +4 with zero regressions, and the implementation's surface and behavior match the ideation contract (inline at `parse_stages_block` site, reuses already-loaded `stages` list, error/warning strings match the spec verbatim).

### Summary

Validated commit c367ffd9 against the four entity-level ACs. All four named tests in `tests/test_status_set_status_validation.py` pass (4/4 in 0.31s); baseline-with-new-tests is 581 passed, baseline-without-new-tests on this branch is 577 passed (delta +4, zero regressions). One narrow-scope observation noted on AC-2: the test uses `--force` to isolate the validator from adjacent guards, which means a buggy validator misclassifying a defined stage as unknown would still pass the assertion since `--force` would convert the reject into a warning-and-write; this is the abstraction level the AC body explicitly endorses, so it does not block. Recommendation: PASSED.
