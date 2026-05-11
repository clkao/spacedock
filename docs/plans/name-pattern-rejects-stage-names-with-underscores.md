---
id: bwck187yrng4rgxeyar1kfzz
title: NAME_PATTERN rejects stage names with underscores
status: validation
source: captain (CL)
started: 2026-05-11T06:39:07Z
completed:
verdict:
score:
worktree: .worktrees/spacedock-ensign-name-pattern-rejects-stage-names-with-underscores
mod-block: merge:pr-merge
pr: #206
---

`claude-team build` derives `Agent()` names as `{worker_key}-{slug}-{stage}` and validates them against `NAME_PATTERN = ^[a-z0-9][a-z0-9-]*[a-z0-9]$` (`skills/commission/bin/claude-team:37`, used at line 317). Workflows whose `stages.states[].name` contains underscores cannot be dispatched — the derived name carries the underscore through and fails the regex with `derived name '...' contains invalid characters`.

Observed examples of stage names that trip this: `in_progress`, `live_full_approved`, `live_probation`.

The README's stage-name field is operator-authored prose. Spacedock's own README uses hyphens by convention, but nothing in commission or status enforces that convention, so workflows authored with snake_case stage names look valid at commission time and then break only at the first dispatch.

Possible resolution directions (decide in ideation):
- Allow underscores in `NAME_PATTERN` (and verify Claude Code's own agent-name validation accepts them).
- Reject underscore stage names at commission / `status --validate` time with a clear error pointing at the offending stage.
- Normalize underscores to hyphens when deriving the agent name (risks collisions across stages that differ only by `_` vs `-`).

Whichever path we pick, the failure must move from "first dispatch surprises the captain" to either "stage name is accepted everywhere" or "commission/validate refuses the stage name with an actionable message".

## Proposed approach — reject at `status --validate`

Take direction 2: surface the failure at workflow-authoring / validation time, not at first dispatch. Concretely:

1. In `skills/commission/bin/status`, add a new validation step inside `validate_workflow()` (around line 699). This is **not** a one-line addition to an existing loop — `validate_workflow()` today iterates entities only and never opens the README. The new step adds three concrete touch points:
   - resolve the README path (`os.path.join(workflow_dir, "README.md")`, mirroring how `status:2338` / `:2430` already do it elsewhere in the file);
   - call `parse_stages_block(readme_path)` (defined at `skills/commission/bin/status:221`) to get the parsed `states` list;
   - iterate that list and append one error per `state['name']` that fails `re.match(r'^[a-z0-9][a-z0-9-]*[a-z0-9]$', name)`, with an error string that names the workflow and stage and suggests the kebab-case form: e.g. `workflow '<dir>': stage name 'in_progress' must match ^[a-z0-9][a-z0-9-]*[a-z0-9]$; rename to 'in-progress' or similar`.

   Treat README-missing and `parse_stages_block` returning `None` as "no stages to check, skip silently" — `validate_workflow` is called on directories that may or may not have a README (e.g., archive scans), and the README's own shape is enforced elsewhere. The new code path is small (well under 20 lines) but it is genuinely new, with its own failure modes (README parse errors, missing file) that the test plan must cover.
2. In `skills/commission/bin/claude-team` (line 317), refine the existing `derived name '...' contains invalid characters` error to call out the most common cause — an underscored stage name — and point operators at `status --validate`. This is the safety net for workflows that skipped validation.
3. Optionally surface the convention once in `skills/commission/SKILL.md` near the workflow template (line ~291) so future commissions are authored correctly. One-line note, not a new section.

### Why this over the other two seed directions

- **Relax `NAME_PATTERN` to allow `_`** (direction 1) — rejected.
  - Spacedock's own naming convention is hyphen-only and pervasive: `FO-R-006` mandates hyphen-separated team names (`docs/plans/fo-behavior-spec-and-coverage-matrix.md:306`), entity slugs are documented as "lowercase, hyphens, no spaces" (`docs/plans/README.md:34`, `skills/commission/SKILL.md:439`), and every existing stage name in the codebase uses hyphens. Allowing `_` everywhere splits the convention and invites a workflow with both `in-progress` and `in_progress` to look fine to the regex while being a debugging hazard.
  - Claude Code's published agent-name rules are not part of the Spacedock contract; we should stay conservative on the conservative side of whatever Claude Code accepts rather than discover the limit at dispatch time. The current regex was deliberately chosen to be a safe subset.
  - No upside vs. direction 2 except sparing the captain a rename when a workflow already uses underscores — but renames in unstarted README stage names are cheap, and once entities exist with that `status:` value the rename touches every entity's frontmatter anyway. The captain pays that cost once, then is done.
- **Normalize `_` → `-` at derive time** (direction 3) — rejected.
  - Captain's seed already flags the collision risk: a workflow with sibling stages `live_full_approved` and `live-full-approved` (or, more realistically, an author who switches conventions mid-workflow) produces two distinct stages with one derived agent name. That is silent, downstream, and corrupts the shutdown-sweep guarantees that depend on agent name uniqueness.
  - Normalization breaks the grep-from-stage-name-to-agent-name correspondence. An operator who sees `derived name 'spacedock-ensign-foo-in-progress'` in a log cannot grep the README and find a stage named `in-progress` because the README says `in_progress`. The mapping is invisible and unguessable.
  - Silent rewriting violates the broader Spacedock principle that workflow text is authoritative — what's in the README is what runs.

### Why move enforcement out of dispatch and into validate

The current failure mode is "workflow looks fine at commission and at `--validate`; first `Agent()` call rejects with a low-context message". Direction 2 inverts that: `status --validate` becomes the single source of truth for "is this workflow shaped correctly?", which matches how the rest of the validate surface already works (entity id uniqueness, slug uniqueness, sd-b32 id well-formedness — all caught at `validate_workflow` rather than at first read).

### Why `--validate` is the *primary* home (not commission-prompt validation)

A reasonable alternative is "catch it in `skills/commission/SKILL.md`'s interactive workflow prompt at authoring time, since that's earlier than `--validate`". I'm deferring that to out-of-scope and treating `--validate` as the primary home for two reasons:

- `status --validate` is the existing single-source-of-truth shape-check that operators (and CI, and FO bootstraps) already invoke before trusting workflow state. Anything that wants to verify a workflow's shape today runs `--validate`; piggy-backing on it means no new entry point and no new contract surface.
- Commission-prompt validation is a separate ergonomics surface that catches only *fresh* commissions, not subsequent README edits — and stage-name additions/renames after commission are exactly the path that produces the "first dispatch surprises the captain" bug. A commission-time-only check would miss the bulk of the failure surface.

A future ergonomics improvement may add the same check to the commission prompt as a second layer; this task locks the primary, durable check in at `--validate` first.

### Boundaries

- The dispatch-time regex stays unchanged. It is the load-bearing safety check; we are adding a friendlier earlier failure point, not replacing the late one.
- The validation applies to the workflow README's `stages.states[].name`, not to entity slugs (slugs already obey hyphen convention via the slugify path).
- The validation is purely static: it reads the README and reports errors. No runtime behavior, no dispatch, no live Claude.
- **Scope of the new validator is the full dispatch-name character class, not just underscores.** The dispatch regex `^[a-z0-9][a-z0-9-]*[a-z0-9]$` rejects underscores, uppercase letters, leading/trailing hyphens, single-character names, names starting or ending with a hyphen, names containing spaces, dots, slashes, or any other non-`[a-z0-9-]` character. All of these break dispatch in the wild, not just `_`. The validator enforces the full regex and the test suite covers more than the single-underscore case, so we don't ship a half-fix that catches `in_progress` but silently accepts `InProgress` or `in progress` or `in.progress`.

## Acceptance Criteria

- **AC-1 — `status --validate` rejects any stage `name` that violates the dispatch-name character class `^[a-z0-9][a-z0-9-]*[a-z0-9]$`, naming the workflow and stage and including a kebab-case suggestion.**
  Verified by: a parametrized unit test that constructs a workflow README per case and asserts `validate_workflow()` returns an error string containing the offending stage name. Cases cover: `in_progress` (underscore), `InProgress` (uppercase), `in progress` (space), `in.progress` (dot), `-leading-hyphen`, `trailing-hyphen-`, single-character `x`. The underscore case additionally asserts the error string contains `in-progress` (the suggested kebab form). Test file: **`tests/test_status_validate.py`** — new file, consistent with the existing per-feature layout for status tests (e.g. `tests/test_status_set_missing_field.py`, `tests/test_status_strict_opening_fence.py`). New test function `test_validate_rejects_invalid_stage_names`.
- **AC-2 — When dispatch-time validation rejects a derived name (the existing `claude-team build` check at `skills/commission/bin/claude-team:317`), the error message identifies the offending stage name, not just the assembled derived name, and points operators at the upstream validator.**
  Verified by: a unit test that constructs a workflow whose stage `name` contains an underscore, calls `build` for that stage, and asserts stderr contains the substrings `stage name`, the offending derived name, and `validate` (case-insensitive). The exact wording around "kebab-case" is not asserted — only the stable substrings above — because the wording is prose that future copy-edits may touch without changing the contract. Extends `tests/test_claude_team.py` with `test_build_underscore_stage_error_message`.
- **AC-3 — A workflow with all stage names already matching the character class passes `status --validate` with exit 0 and prints `VALID`.**
  Verified by: a regression unit test that runs `status --validate` against the existing `_make_workflow_fixture` workflow from `tests/test_claude_team.py` (its stage names — e.g. `ideation`, `implementation` — already match), asserting `returncode == 0` and stdout `VALID`. Lives in the same new `tests/test_status_validate.py` file.
- **AC-4 — The commission SKILL template surfaces the stage-name convention in one line near the workflow `stages.states[]` template.**
  Verified by: `grep -nE "kebab|stage name.*\\^\\[a-z0-9\\]|stage.*lowercase.*hyphen" skills/commission/SKILL.md` returns at least one hit within the stages-template section (currently around lines 286-302).

## Test Plan

- **Mechanism check first** (per Rule: validate smallest end-to-end exercise of the riskiest path FIRST): write the failing parametrized unit test for AC-1 and confirm at least one case (e.g. `in_progress`) fails against the *current* code before changing anything. This proves the test wires up to the right entry point (`validate_workflow()`) and confirms today's behavior is "no error for `in_progress`". Validating the new code path's read-the-README contract here also catches the "validate_workflow doesn't read READMEs today" gotcha early — if the test setup expects an error and the implementation hasn't yet added the README-read step, the test will surface the gap concretely instead of leaving it implicit in the plan.
- **Parametrized unit test** for AC-1 in new file `tests/test_status_validate.py`. Seven `pytest.mark.parametrize` cases (the seven listed in AC-1), each constructing a `tmp_path` workflow with a single offending stage name and asserting one error per case. Cost: ~15 minutes wallclock to author, milliseconds to run.
- **Unit test** for AC-2 extending `tests/test_claude_team.py` (`test_build_underscore_stage_error_message`), using the established `_make_workflow_fixture` pattern. ~20 lines.
- **Regression unit test** for AC-3 in `tests/test_status_validate.py` — same fixture path, asserting clean exit on the standard hyphenated fixture.
- **Failure-mode coverage for the new code path**: include one parametrized case in AC-1's test for "README missing" (verify validator does not crash and emits no stage-name error) and one for "README present but no `stages:` block" (same — no crash, no error). These cover the "skip silently" branch noted in the Proposed approach above. Adds ~5 lines to the parametrize list.
- **Static check** for AC-4 — the `grep` command in the AC body. Manual one-shot run during implementation review; optionally codified as a test in `tests/test_status_validate.py` if cheap.
- **No E2E tests needed**. The claim is "static validation of workflow README config", not "runtime dispatch behavior". A live-Claude test would be paying a 10+ minute round-trip to verify a regex check that a unit test verifies in milliseconds. Per the ideation guide: "Choose proof at the same abstraction level as the claim: static checks for durable doc/contract structure".
- **No fixture changes expected**. Existing kebab-case fixtures already exercise the happy path; the new tests construct offending stage names inline in the test file.

## Out of scope

- Migrating existing in-the-wild workflows with snake_case stage names. The new validation will surface them; renaming is a per-workflow operator task, not part of this change.
- Changing `NAME_MAX_LEN`, `MODEL_ENUM`, or any other dispatch-name validation knob.
- Adding stage-name validation to commission's interactive prompts (a separate, larger ergonomics change).
- Touching `parse_stages_block` parsing — the raw string capture stays as-is; validation runs against the parsed result.

## Stage Report: ideation

- DONE: Acceptance criteria are entity-level end-state properties with concrete 'Verified by' citations (grep / test name / file path / command). No imperative-verb AC items.
  Four ACs, each stated as an end-state ("`status --validate` reports …", "error message … mentions …", "clean workflow … still passes", "template documents …") with named test functions, file paths, and a grep command as the verifier.
- DONE: Proposed approach picks among the three seed directions (relax NAME_PATTERN, reject underscores at commission/validate, or normalize underscores to hyphens) with explicit reasoning about tradeoffs — including the collision risk for normalization and whether Claude Code's own agent-name rules accept underscores.
  Selected direction 2 (reject at `status --validate`). Direction 1 rejected because it splits the pervasive hyphen-only convention (cited FO-R-006, slug docs) and treats Claude Code's external rules as load-bearing when they shouldn't be. Direction 3 rejected on the seed's own collision argument plus loss of grep-from-README-to-agent-name correspondence.
- DONE: Test plan picks proof at the right abstraction level for the chosen approach — unit test against NAME_PATTERN or commission validation for static checks, with E2E only if real-runtime dispatch behavior is the claim.
  Three unit tests against `validate_workflow()` and `claude-team build`, no E2E. Mechanism check (failing test before code change) included as the first step per the validate-the-riskiest-path-first rule.

### Summary

Fleshed the task body around direction 2 (reject underscored stage names at `status --validate`, with a friendlier dispatch-time error as a safety net). The collision risk and the codebase-wide hyphen convention rule out the other two seed directions. Test plan stays at unit-test level — this is static workflow-config validation, not runtime dispatch behavior, so an E2E would be over-instrumentation. Implementation surface is one added loop in `validate_workflow()` plus a more specific error string in `claude-team build`.

## Stage Report: ideation (cycle 2)

- DONE: Material — `validate_workflow` doesn't read the README today.
  Rewrote "Proposed approach" item 1 to name the three concrete touch points (resolve README path, call `parse_stages_block` at `status:221`, iterate and regex-check) and acknowledge this is a new code path, not a one-liner. Added explicit handling for README-missing / no-stages-block as "skip silently".
- DONE: Scope — narrow vs full character class.
  Broadened the validator scope to the full dispatch-name character class with explicit justification in a new "Scope of the new validator" boundary bullet. AC-1 now covers seven parametrized cases (underscore, uppercase, space, dot, leading/trailing hyphen, single-char), not just underscores.
- DONE: Argue `status --validate` as primary home.
  Added a new "Why `--validate` is the *primary* home (not commission-prompt validation)" subsection with two reasons: it's the existing single-source-of-truth shape-check operators/CI/FO already invoke, and commission-time-only validation misses the post-commission README-edit failure path.
- DONE: AC2 wording is over-fitted.
  Loosened AC-2 to assert stable substrings (`stage name`, the offending derived name, `validate` case-insensitive). Removed the literal `kebab-case` / `--validate` regex requirement and called out explicitly that prose wording is not contractually fixed.
- DONE: Flag `tests/test_status_validate.py` as new.
  AC-1 now annotates the file as **new** with a one-line rationale citing the existing per-feature status test layout (`tests/test_status_set_missing_field.py`, `tests/test_status_strict_opening_fence.py`).
- DONE: Optional polish — reformat ACs to `**AC-N — ...**` template form.
  Reformatted all four ACs from `1./2./3./4.` to `- **AC-N — {End-state property.}**` matching the workflow README's canonical template.

### Summary

Folded all five fixup notes plus the optional AC-format polish. The biggest correction was the dishonest "one-liner" framing in item 1 — `validate_workflow()` doesn't currently open the README at all, so the change is a small but genuinely new code path with its own failure modes (covered in the test plan). Scope is now the full character class, not just underscores, with seven parametrized test cases. AC-2 wording is loosened to stable substrings. Primary-home reasoning for `--validate` is explicit.

## Stage Report: implementation

- DONE: Mechanism-check-first: write the AC-1 parametrized failing test in tests/test_status_validate.py, run it against current code to confirm it fails (no error today for invalid stage names), commit the test BEFORE the validate_workflow change lands.
  Commit 06f71036 — 8/11 tests failed against current code (the 7 invalid-name parametrize cases plus the underscore-suggestion case); the 3 silent-skip / kebab-accept tests passed.
- DONE: validate_workflow's new stage-name check enforces the full dispatch-name character class (^[a-z0-9][a-z0-9-]*[a-z0-9]$), and all seven AC-1 parametrized cases (underscore, uppercase, space, dot, leading-hyphen, trailing-hyphen, single-char) plus the two failure-mode cases (README-missing, no-stages-block) pass.
  Commit 98e4e68e — adds STAGE_NAME_RE, validate_workflow_stage_names() (resolves README path, calls parse_stages_block, regex-checks each name, returns []  silently on README-missing/no-stages), and a kebab suggestion helper. 11/11 tests in tests/test_status_validate.py pass.
- DONE: claude-team build's error string at line 317-318 is refined to satisfy AC-2's stable-substring contract: stderr contains the offending derived name, the substring 'stage name', and 'validate' (case-insensitive). The AC-2 test passes against the refined string.
  Commit 9edec125 — claude-team:317-324 emits derived name + stage name + pointer at `status --validate`; new TestBuildUnderscoreStageError::test_build_underscore_stage_error_message in tests/test_claude_team.py asserts the three substrings.
- DONE: AC-4 SKILL.md convention note (covered by completion checklist via the grep verifier).
  Commit 9edec125 — added one-line comment in skills/commission/SKILL.md:287 above `defaults:`. Grep `grep -nE "kebab|stage name.*\^\[a-z0-9\]|stage.*lowercase.*hyphen" skills/commission/SKILL.md` returns the new line.
- DONE: Run `make test-static` from the worktree root as part of validation.
  600 passed, 26 deselected, 15 subtests passed in 27.49s. No regressions.

### Summary

Implemented direction 2 (reject at `status --validate`) per the ideation spec. Added STAGE_NAME_RE and a new validate_workflow_stage_names() helper that reads the README, parses the stages block, and emits one error per stage name violating the dispatch-name regex; README-missing and no-stages-block cases skip silently. Refined the claude-team dispatch-time error to name the offending stage and point at `status --validate`. Surfaced the convention as a one-line comment in the SKILL.md workflow template. Three commits, eleven new tests pass, full test-static green at 600/600.

## Stage Report: validation

- DONE: Each **AC-N** in the entity body has its 'Verified by' clause located and reproduced — run the named test, perform the named grep, and cite the concrete result. Report any AC whose evidence cannot be reproduced as FAILED, not as DONE.
  - **AC-1** (`tests/test_status_validate.py::test_validate_rejects_invalid_stage_names` + `test_validate_underscore_suggests_kebab`): reproduced — all 7 parametrize cases (`in_progress`, `InProgress`, `in progress`, `in.progress`, `-leading-hyphen`, `trailing-hyphen-`, `x`) PASS; the kebab-suggestion check for `in_progress` PASS. Stderr names the offending stage and includes the substring `stage name`.
  - **AC-2** (`tests/test_claude_team.py::TestBuildUnderscoreStageError::test_build_underscore_stage_error_message`): reproduced — 1 PASS. Independently confirmed by reading `skills/commission/bin/claude-team:317-324`: the error string interpolates the offending `derived_name`, the literal `stage name`, and `Run \`status --validate\`` (case-insensitive substring `validate` present). All three stable substrings verified.
  - **AC-3** (`tests/test_status_validate.py::test_validate_accepts_kebab_stage_names`): reproduced — PASS, exit 0, stdout contains `VALID` against the standard hyphenated `README_WITH_STAGES` fixture.
  - **AC-4** (grep verifier): reproduced — `grep -nE "kebab|stage name.*\^\[a-z0-9\]|stage.*lowercase.*hyphen" skills/commission/SKILL.md` returns `287:  # Stage names must match ^[a-z0-9][a-z0-9-]*[a-z0-9]$ ...`. Line 287 sits inside the workflow `stages:` template block (lines 286-302), satisfying the AC's "within the stages-template section" requirement.
- DONE: `make test-static` runs clean from the worktree (full count cited), AND the AC-specific tests (tests/test_status_validate.py and the AC-2 test in tests/test_claude_team.py) are run individually with explicit pass/fail counts in the report. A whole-suite pass with one AC-specific failure is still a REJECT.
  - `tests/test_status_validate.py` run individually: **11 passed, 0 failed** (0.30s).
  - `tests/test_claude_team.py::TestBuildUnderscoreStageError` run individually: **1 passed, 0 failed** (0.04s).
  - `make test-static`: **600 passed, 26 deselected, 15 subtests passed** in 27.33s. No failures, no regressions.
- DONE: PASSED/REJECTED recommendation with reasoning — explicitly check for scope drift (was NAME_PATTERN at claude-team:37 touched? Was anything outside the (status, claude-team, SKILL.md) surface changed?) and for tests passing but proving obsolete or wrong-abstraction behavior (per the workflow README's 'Bad' guidance for validation).
  - **PASSED.**
  - Scope-drift check: `git diff 25cfd2b3..HEAD --name-only` lists only `docs/plans/name-pattern-rejects-stage-names-with-underscores.md`, `skills/commission/SKILL.md`, `skills/commission/bin/claude-team`, `skills/commission/bin/status`, `tests/test_claude_team.py`, `tests/test_status_validate.py`. No other files touched. `grep -n "NAME_PATTERN" skills/commission/bin/claude-team` shows `NAME_PATTERN = re.compile(r'^[a-z0-9][a-z0-9-]*[a-z0-9]$')` at line 37 — unchanged from baseline. The dispatch regex stays load-bearing; only its error string at 317-324 was refined.
  - Abstraction-level check: the ideation explicitly rules out E2E ("static validation of workflow README config, not runtime dispatch behavior"). Unit tests at `validate_workflow()` and at `cmd_build`'s error path are the proof-at-the-same-abstraction-level the AC list demands. No test is rubber-stamping mocked behavior — `test_status_validate.py` shells out to a real built `status` script, and the AC-2 test invokes `cmd_build` against a real on-disk fixture.
  - Obsolete-behavior check: each AC's verified-by clause matches the current code path it targets (validator entry, dispatch-time check, SKILL template). No drift between AC prose and the tested target.

### Summary

Validated all four ACs against fresh reproduction of their named evidence. `make test-static`: 600 passed, 26 deselected, 15 subtests passed. AC-specific runs: 11/11 on `test_status_validate.py` and 1/1 on `TestBuildUnderscoreStageError`. Scope is tight (status, claude-team, SKILL.md, tests, entity file only); NAME_PATTERN at claude-team:37 is untouched. Recommendation: **PASSED**.
