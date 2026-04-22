---
id: "215"
title: Interactive stage type for captain-performed work
status: ideation
source: captain session 2026-04-21
started: 2026-04-22T05:26:49Z
completed:
verdict:
score:
worktree:
---

Allow a workflow stage to be declared `interactive: true` so the FO keeps that stage foreground and hands the work to the captain instead of dispatching a worker. Intended for stages that need heavy back-and-forth (clarification-heavy ideation, design critique, hand-authored content) where a subagent would ping-pong more than it would progress.

This task itself is a motivating example: ideation on a scaffolding change is exactly the kind of heavy-clarification work an `interactive: true` stage would absorb.

## Problem

Some stages are naturally captain-performed: the captain has the context, the judgement, and the patience that a subagent does not, and the hand-off overhead (dispatch, idle poll, stage-report writing, gate presentation) costs more than it saves. The FO currently has exactly one dispatch mode — spawn an ensign via `claude-team build` → `Agent()` — and applies it to every non-terminal stage. Stages that are better done by the captain have to either pretend to be worker stages (and produce a thin, dishonest ensign report) or be skipped out-of-band (undermining the workflow record). There is no schema-level way to say "this stage is foreground work the captain drives directly."

The concrete gap the captain runs into: when an idea lands in `backlog` and moves to `ideation`, the captain often wants to think out loud *with* the FO in the main conversation, not to wait on an ensign dispatch that will re-ask questions the captain can already answer. Today that forces either an unnatural hand-off or an off-the-record conversation that never makes it into the entity body.

## Proposed approach

### Design choice: whole-stage captain-performed

Three alternatives were considered:

1. **Captain-performed whole stage (chosen).** `interactive: true` on a stage tells the FO to skip `claude-team build` / `Agent()` entirely, conduct the stage inline with the captain (direct text output on Claude, `send_input` on Codex), and require the captain to produce the stage body + stage report through direct authoring. The FO transitions frontmatter, presents the gate, and advances as usual — only the "who writes the body" step diverges.
2. **Captain-invited checkpoint mid-stage.** An ensign dispatches normally but pauses for captain input at a declared checkpoint. Rejected: the ping-pong cost is the thing the feature is trying to avoid, and the checkpoint shape duplicates what `gate: true` already provides.
3. **Dispatched worker with captain co-chat.** Spawn an ensign, route captain messages into it. Rejected: adds routing complexity to solve a problem the captain does not have (the captain *is* already at the keyboard); the worker becomes a scribe whose value over direct authoring is unclear.

Chosen: whole-stage captain-performed. Smallest schema surface, cleanest FO branch, honest about who did the work. Other alternatives can be layered on later without this design blocking them.

### Schema change

Add one optional boolean field to the stages.states entries in workflow `README.md` frontmatter: `interactive: true`. Default is `false`. Consumed by `parse_stages_block` in `skills/commission/bin/status` (same place `gate`, `worktree`, `terminal`, `fresh` are parsed).

Concrete location in `skills/commission/bin/status` around line 265–271:

Before:
```python
stage = {
    'name': state['name'],
    'worktree': state.get('worktree', str(default_worktree)).lower() == 'true',
    'concurrency': int(state.get('concurrency', str(default_concurrency))),
    'gate': state.get('gate', 'false').lower() == 'true',
    'terminal': state.get('terminal', 'false').lower() == 'true',
    'initial': state.get('initial', 'false').lower() == 'true',
}
for optional_field in ('feedback-to', 'agent', 'fresh', 'model'):
```

After:
```python
stage = {
    'name': state['name'],
    'worktree': state.get('worktree', str(default_worktree)).lower() == 'true',
    'concurrency': int(state.get('concurrency', str(default_concurrency))),
    'gate': state.get('gate', 'false').lower() == 'true',
    'terminal': state.get('terminal', 'false').lower() == 'true',
    'initial': state.get('initial', 'false').lower() == 'true',
    'interactive': state.get('interactive', 'false').lower() == 'true',
}
for optional_field in ('feedback-to', 'agent', 'fresh', 'model'):
```

The field flows to the FO via the same `parse_stages_with_defaults` / `parse_stages_block` path that already surfaces every other stage flag. No new helper is needed.

### FO dispatch-path divergence

In `skills/first-officer/references/first-officer-shared-core.md` `## Dispatch`, a new branch fires **before** step 3 (worktree conflict check) and **before** step 4 (dispatch_agent_id resolution):

> **Interactive stage short-circuit.** If the target stage has `interactive: true`, the FO does NOT invoke `claude-team build` and does NOT call `Agent()`. Instead:
>
> 1. Update main-branch frontmatter exactly as for a worker dispatch (`status --set {slug} status={next_stage} started`). An `interactive: true` stage MUST NOT also declare `worktree: true` — the schema validator in `claude-team build` rejects the combination (see Interactions below); the `worktree=...` field is therefore never set on this transition.
> 2. Commit the state transition on main with `dispatch: {slug} entering {next_stage} (interactive)`.
> 3. Extract the stage subsection from the README via the existing `extract_stage_subsection` helper.
> 4. Present the stage subsection, the generated dispatch checklist, and the entity file path to the captain via direct text output (Claude) or `send_input` equivalent (Codex). Use the exact header wording `Interactive stage: {stage} — captain-performed`. The captain authors the body and stage report directly in the entity file.
> 5. Wait for the captain to indicate completion. The completion signal for an interactive stage is the captain saying "done" / "advance" / invoking `/spacedock advance {slug}` (follow-up task), NOT an agent `SendMessage`. No `Agent()` handle to shut down; no keep-alive bookkeeping.
> 6. Re-read the entity file; verify the last `## Stage Report: {stage}` section exists and accounts for every checklist item (same invariant as worker dispatches). If missing or incomplete, ask the captain to repair it before advancing.
> 7. Run the normal gate / completion flow from step 9 onward of `## Dispatch`.

On the runtime adapters, the divergence touches two files but adds no new tool calls — both runtimes already have the captain-communication channel this relies on:

- `skills/first-officer/references/claude-first-officer-runtime.md` gains a short `## Interactive Stage Dispatch` subsection under `## Dispatch Adapter` pointing to the shared-core rule and noting that "captain interaction is direct text output" — the same rule that already applies to gate presentation.
- `skills/first-officer/references/codex-first-officer-runtime.md` gains the matching subsection noting that Codex uses the regular conversation stream (no `send_input` to a subagent because there is no subagent).

### Scaffolding wording changes

Workflow README stage docs (the `## Stages` sections in `docs/plans/README.md` and any freshly commissioned workflow) do NOT need default copy about `interactive: true` — it is an opt-in flag a workflow author adds per stage, not a default behavior. The schema field reference in the commission skill MUST document it, though.

Concrete before/after in `skills/commission/SKILL.md` (the stages-field documentation block — location to be verified by the implementation ensign, which must grep `gate: Mark this stage as a human approval checkpoint` or equivalent and add an adjacent entry):

Add, adjacent to the existing `gate`/`worktree`/`terminal` descriptions:
> - `interactive: true` — the stage is captain-performed. The FO does not dispatch a worker; the captain drives the work directly in the main conversation and authors the stage report. Mutually exclusive with `worktree: true`.

`claude-team build` MUST refuse to assemble an `Agent()` prompt for an `interactive: true` stage — it is a logic error to reach the build helper for such a stage. Add a Rule (sibling to Rule 4 "Worktree stage has worktree path") in `skills/commission/bin/claude-team` around line 236:

```python
if stage_meta.get('interactive', False):
    return _build_error(
        f"stage '{stage}' is interactive (captain-performed); "
        f"claude-team build must not be invoked for it. "
        f"The FO runs the stage inline."
    )
```

And sibling validation: reject an `interactive: true` stage that also declares `worktree: true` (at parse time in `parse_stages_block`, or at build time in `claude-team build` — prefer the earliest failure that still ships with a test):

```python
if stage_meta.get('interactive', False) and stage_meta.get('worktree', False):
    return _build_error(
        f"stage '{stage}' declares both interactive: true and worktree: true; "
        f"captain-performed stages run inline and cannot own a worktree."
    )
```

No change needed to ensign scaffolding (`skills/ensign/`) — ensigns are never dispatched for interactive stages, so their runtime is untouched.

### Interactions

- **`worktree: true`:** forbidden. Captain-performed work happens in the main conversation, and the FO working directory stays at project root per `## Working Directory`. Declaring both is a README bug; both parsers reject it.
- **`feedback-to`:** allowed. An interactive stage can receive a rejection bounce (the captain is already the author, so routing back to them is trivial — the FO re-presents the stage checklist with the reviewer findings appended, same as a worker dispatch with `feedback_context`). The `feedback-to` *target* being interactive is the primary useful combination. The reverse (an interactive stage pointing `feedback-to` at another stage) is also allowed but less likely; no special handling.
- **`gate: true`:** allowed and orthogonal. Interactive-authored reports go through the same gate presentation as worker-authored reports.
- **`fresh: true`:** no effect. There is no worker to reuse.
- **`agent: {name}`:** MUST be empty / unset. Schema validator in `claude-team build` rejects the combination since no agent is dispatched.
- **`terminal: true`:** allowed in principle (captain-authored terminal resolution), but orthogonal — terminal stages already do not dispatch.
- **Reuse / SendMessage advancement:** not applicable. No worker, no handle.
- **Standing teammates:** unchanged. They live at team scope; an interactive stage simply does not spawn a fresh ensign that would message them. The captain MAY route to them directly if the workflow needs it, but that is outside the scope of this task.
- **`status --next` dispatchability:** no change needed. `interactive: true` stages remain dispatchable in the same sense as worker stages — `--next` reports them, the FO then takes the interactive branch instead of the Agent branch. No new filter.

### What "done" looks like

- Schema parser accepts `interactive: true` on a stage and surfaces it on the parsed stage dict.
- `claude-team build` refuses to assemble a dispatch prompt for an `interactive: true` stage and refuses the `interactive + worktree` combination.
- The FO shared-core spec documents the interactive short-circuit with step-by-step flow.
- The runtime adapters each have a short subsection pointing at the shared-core rule.
- The commission skill's stages-field reference documents the flag.
- A live FO run against a workflow containing an `interactive: true` stage produces: captain sees the stage subsection + checklist in the main conversation, captain writes the stage report to the entity file, FO detects the report, FO presents at gate (or advances directly).

## Acceptance criteria

Each AC names a property of the finished entity (not a stage action) and how it is verified.

**AC-1 — The stages schema parser surfaces `interactive` as a boolean key on every parsed stage dict, defaulting to False.**
Verified by: `python3 -c "from skills.commission.bin._status import parse_stages_block; stages = parse_stages_block('tests/fixtures/interactive-stage/README.md'); assert stages[1]['interactive'] is True and stages[0]['interactive'] is False"` — a new fixture README under `tests/fixtures/interactive-stage/` declares one interactive stage and one non-interactive stage; the assertion passes.

**AC-2 — `claude-team build` refuses to assemble a dispatch for an `interactive: true` stage with a non-zero exit and a stderr message containing the anchor phrase `captain-performed`.**
Verified by: `tests/test_claude_team.py::test_build_rejects_interactive_stage` (new) — pipes a build-input JSON targeting an interactive stage, asserts exit code ≠ 0 and stderr contains `captain-performed`.

**AC-3 — `claude-team build` refuses to assemble a dispatch for a stage declaring both `interactive: true` and `worktree: true` with a non-zero exit and a stderr message naming both flags.**
Verified by: `tests/test_claude_team.py::test_build_rejects_interactive_worktree_combination` (new) — fixture README with the forbidden combination; assertion checks exit code and stderr anchor phrase `interactive: true and worktree: true`.

**AC-4 — `skills/first-officer/references/first-officer-shared-core.md` contains a section titled `Interactive stage short-circuit` (or a `## Dispatch` subsection with that exact phrase) describing the seven-step inline flow.**
Verified by: `grep -q "Interactive stage short-circuit" skills/first-officer/references/first-officer-shared-core.md` AND `grep -c "^[0-9]\." skills/first-officer/references/first-officer-shared-core.md` increases by 7 relative to the pre-change version (the numbered steps).

**AC-5 — Both runtime adapters (`claude-first-officer-runtime.md`, `codex-first-officer-runtime.md`) contain a subsection referencing the shared-core interactive short-circuit.**
Verified by: `grep -l "interactive" skills/first-officer/references/claude-first-officer-runtime.md skills/first-officer/references/codex-first-officer-runtime.md` returns both files, and each file contains the phrase `captain-performed`.

**AC-6 — `skills/commission/SKILL.md` documents the `interactive` field in its stages-field reference, naming mutual exclusion with `worktree: true`.**
Verified by: `grep -A2 "interactive:" skills/commission/SKILL.md` shows the description including the phrase `Mutually exclusive with` (or equivalent) referencing `worktree`.

**AC-7 — An end-to-end FO run against a fixture workflow with an `interactive: true` stage demonstrates the captain-performed flow: FO presents stage subsection + checklist in the conversation, does not spawn an ensign, accepts a captain-authored stage report, and advances to the next stage.**
Verified by: `tests/test_interactive_stage_e2e.py` (new, under `pytest -m live_claude`) — runs a one-entity workflow scripted via the existing PTY harness (`scripts/test_lib_interactive.py`, already used by `test_interactive_poc.py`); asserts (a) no `Agent()` call is emitted for the interactive stage (inspectable via the subagent-logs helper already in the harness), (b) the entity file gains a `## Stage Report: {interactive_stage}` section authored during the run, (c) the FO advances frontmatter to the next stage after the captain signals done.

**AC-8 — The checklist-coverage invariant (shared-core `## Completion and Gates` step 2) holds for captain-authored stage reports — an interactive stage whose report is missing checklist items is sent back for repair, same as a worker dispatch.**
Verified by: `tests/test_interactive_stage_e2e.py::test_incomplete_report_rejected` (new) — captain-authored report intentionally omits one checklist item; asserts FO does NOT advance and issues a repair request naming the missing item.

## Test plan

Risk profile: touches schema parsing (cheap to test offline), touches FO prose in reference files (cheap to test via grep), touches live FO behavior (needs one live E2E run because "the FO takes the interactive branch instead of dispatching" is a behavioral guarantee not observable from static files alone). The expensive path is AC-7 and AC-8.

**Static checks (cheap, deterministic, zero API cost):**

- Schema parser unit test — extend `tests/test_claude_team.py` with a new `test_parse_stages_block_interactive` that asserts the boolean surfaces and defaults correctly. New test, ~15 lines.
- `claude-team build` rejection tests (AC-2, AC-3) — two new cases in `tests/test_claude_team.py`, reusing its fixture-README patterns. New tests, ~30 lines each.
- Documentation greps (AC-4, AC-5, AC-6) — one new `tests/test_interactive_stage_docs.py` with three grep-based assertions. New test file, ~20 lines.

**Behavioral checks (one live E2E run):**

- AC-7 and AC-8 go together in `tests/test_interactive_stage_e2e.py`, one new file under `pytest -m live_claude and serial` (serial because the PTY harness already runs single-session). Reuses `InteractiveSession` from `scripts/test_lib_interactive.py` and the subagent-log inspection helper. Estimated cost: ~2–3 minutes of live-Claude time per run, one entity, two scripted user turns (ideation author + advance command). Proportional to risk — the behavioral guarantee is "no ensign is spawned for this stage," and only a live run can verify the absence.

**Codex coverage:** out of scope for this task. The Codex runtime adapter gets its documentation subsection (AC-5), but the live Codex E2E is deferred to a follow-up once the Claude path is proven. Filed-as-follow-up is acceptable because the divergence surface on Codex is the same shared-core rule; runtime-specific risk is limited to the conversation-channel difference, which the adapter text already flags.

**Not covered by this task:**

- A `/spacedock advance` slash command for the captain's "done" signal — this task treats the signal as free-text detection (captain says "done" or "advance"). A dedicated slash command can layer on later.
- Routing captain-chat to standing teammates during an interactive stage — already works today through normal conversation; no new mechanism needed.
- Retrofitting existing workflows' `ideation` stages to `interactive: true` — the flag is opt-in per workflow; this task ships the capability, not the migration.

## Stage Report: ideation

- DONE: Proposed approach names the concrete schema change (where `interactive: true` lives, which schema/helper consumes it) AND the FO dispatch-path divergence (what the FO does instead of Agent(): prompts captain, collects output, writes stage report, handles gating and reuse). Include specific before/after wording for any scaffolding change (README stage docs, first-officer reference adapters, `claude-team build`).
  Schema change is in `parse_stages_block` at `skills/commission/bin/status:265-271` with before/after diff; FO divergence is a seven-step short-circuit added to `first-officer-shared-core.md` `## Dispatch`; `claude-team build` gains two guard rules around line 236; `skills/commission/SKILL.md` stages-field reference documents the flag.
- DONE: Acceptance criteria are end-state properties (not imperatives), each with a verification command/grep/test/file path a future reader can reproduce. Cover at minimum: schema field accepted, FO honors it in a live dispatch, captain-authored stage report passes the checklist-coverage invariant, interaction with `worktree: true` and `feedback-to` is defined.
  Eight ACs written; AC-1 covers schema field, AC-7 covers live FO behavior, AC-8 covers checklist-coverage invariant on captain-authored reports, AC-3 covers `worktree: true` mutex; `feedback-to` interaction is in the Interactions subsection of the proposed approach. Every AC names a concrete grep/pytest path.
- DONE: Test plan separates cheap static checks (schema parse, `status` tooling, doc grep) from behavioral checks (FO routing under `interactive: true`), calls out which tests are new vs extensions, and names whether an E2E harness run is proportional to risk or overkill.
  Static checks extend `tests/test_claude_team.py` and add `tests/test_interactive_stage_docs.py` (~85 lines total, zero API cost); one live E2E in `tests/test_interactive_stage_e2e.py` under the existing PTY harness, ~2–3 min/run, scoped to Claude only. Codex E2E deferred with rationale.

### Summary

Fleshed out a captain-performed interactive stage design: one boolean schema flag (`interactive: true`), a seven-step FO short-circuit that bypasses `claude-team build` and `Agent()`, two guard rules in `claude-team build` (reject dispatch for interactive stages; reject `interactive + worktree` combination), and documentation updates across shared-core + both runtime adapters + commission skill. Picked whole-stage captain-performed over mid-stage checkpoint or dispatched-scribe alternatives because it has the smallest schema surface and the cleanest FO branch. Test plan is mostly cheap static checks plus one live E2E to verify the behavioral guarantee that no ensign spawns.
