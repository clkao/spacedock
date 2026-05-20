---
id: 4q934ttakvtv5ngt662pez34
title: "claude-team build dispatch prompts are ~7000 chars; could be ~500"
status: ideation
source: "GitHub issue #229 — CL observed dispatch prompts inline boilerplate that duplicates `spacedock:ensign` skill body"
started: 2026-05-20T20:34:07Z
completed:
verdict:
score:
worktree:
issue: "#229"
pr:
---

## Problem

`claude-team build`'s emitted dispatch prompt inlines, on every dispatch:

- Full stage definition (the same block on every entity at the same stage)
- Read-entity instructions
- Completion signal protocol (full breakglass note)
- Worktree + branch declarations
- Stage report appending instructions

Ensigns load the `spacedock:ensign` skill on their first action anyway, so most of this content is redundant duplication of the skill body.

## Proposal

Shrink the dispatch prompt to roughly:

- Entity path
- Stage name
- Checklist items (the stage-specific work)
- Scope notes (worktree, branch)
- Completion signal target

Let the ensign skill carry the rest. Reduces ~7000 chars per dispatch to ~500.

## Why this matters

- Faster reads for the human watching the team
- Less context burned at dispatch time
- Easier to spot differences between dispatches (most of the content is currently identical boilerplate)

## Scale context

- Spacedock version: 0.11.2
- GitHub issue: https://github.com/clkao/spacedock/issues/229

## Design

### Section-by-section: kept vs. dropped vs. moved

The current dispatch prompt assembled by `cmd_build` in `skills/commission/bin/claude-team` (lines 334–456) has 11 components. Each is classified below as **per-dispatch content** (varies by entity/stage/team — keep) or **operating-contract content** (boilerplate identical across dispatches at the same stage — drop, because `spacedock:ensign` already preloads/loads it via skill).

| # | Section (current source line) | Disposition | Rationale (one sentence) |
|---|---|---|---|
| 0 | `## First action` — `Skill(skill="spacedock:ensign")` directive (`claude-team:338-350`) | **KEEP** | Belt-and-suspenders for the skill-preload failure mode (see Failsafe below); also tests at `tests/test_claude_team.py:736-771` assert it appears before the header. |
| 1 | `You are working on: {title}` / `Stage: {stage}` header (`claude-team:353`) | **KEEP** | Per-dispatch identifiers; ensign needs them on first read. |
| 2 | `### Stage definition:` + verbatim README subsection (`claude-team:356`) | **KEEP** | Per-dispatch: stage definitions differ between `ideation`, `validation`, `implementation`, etc. CL's correction at the backlog gate: this is the actually-informative content that justifies the dispatch. |
| 3 | Worktree path + branch declarations (`claude-team:359-366`, conditional) | **KEEP** | Per-dispatch: worktree path and branch name are derived from the entity. Cannot be moved to the skill body (skill is static). |
| 4 | `Read the entity file at {path} for the current spec.` (`claude-team:368-379`) | **SHRINK** | Keep the path (per-dispatch), drop the "for the current spec" + ":\n  It contains:" boilerplate. The "always read the entity file before making changes" instruction already lives in `ensign-shared-core.md:15-17`. New form: `Entity: {path}`. |
| 5 | `### Feedback from prior review` block (`claude-team:382-383`, conditional) | **KEEP** | Per-dispatch content; this IS the feedback. |
| 6 | Scope notes (`claude-team:386-387`, conditional) | **KEEP** | Per-dispatch content; FO supplies them when present. |
| 7 | `### Completion checklist` + checklist items (`claude-team:390-396`) | **KEEP** | Per-dispatch: the checklist IS the stage's specific work. |
| 7b | `### Summary\n{brief description of what was accomplished}` template line (`claude-team:393-396`) | **DROP** | Stage Report Protocol in `ensign-shared-core.md:62-65` already specifies the Summary subsection structure. Templating it inline in every dispatch is duplicate. |
| 8 | `### Stage report` appending instructions (`claude-team:398-403`) | **DROP** | The entire Stage Report Protocol — including append-at-end discipline, DONE/SKIPPED/FAILED markers, verbatim-checklist-item-text rule, and re-do-cycle naming — already lives in `ensign-shared-core.md:48-77`. The skill is invoked on first action; the ensign has it before processing the checklist. |
| 9 | `### Standing teammates available in your team` (`claude-team:406-437`, conditional) | **KEEP** | Per-dispatch: the set of standing teammates and their routing-usage bodies vary by team. Cannot move to the static skill body. |
| 10 | `### Completion Signal` block (`claude-team:440-456`, conditional) | **SHRINK** | Keep the literal `SendMessage(to="team-lead", message="Done: {title} completed {stage}. Report written to {path}.")` line — the entity_title/stage/entity_file_ref are per-dispatch and the literal parenthesis-equals syntax is what the ensign must emit. **Drop** the two surrounding prose paragraphs ("This is a team-mode dispatch / Plain text only / Idle notifications are normal" + the meta "If you are the first officer forwarding this prompt"). Move both into `claude-ensign-runtime.md` (the runtime adapter already loaded by the skill) under the existing `## Completion Signal` section (lines 17-25), which already says "Plain text only. Never send JSON." but doesn't yet carry the "FO keeps waiting / idle notifications are normal" clarification or the FO-forwarding warning. |

**Net before/after.** Current prompt for a typical worktree-backed team-mode dispatch with a feedback context and standing teammates: ~7000 chars (the dispatch we received at the top of this session is ~6900). After shrink: the kept per-dispatch content plus shrunk Read/Completion-Signal blocks sums to roughly:

- First action directive: ~600 (unchanged)
- Header + Stage definition: ~per-stage; ~1500 typical (unchanged — this is the bulk)
- Worktree+branch: ~300 (unchanged when present)
- Entity path: ~80 (was ~300)
- Checklist: ~per-stage; ~800 typical (unchanged)
- Standing teammates: ~1200 (unchanged when present)
- Completion signal: ~180 (was ~800)
- Stage report block: 0 (was ~400)

So the actual saved bytes are roughly 1200 chars per dispatch — much less than the "7000 → 500" framing in the original proposal, because the per-dispatch content (stage definition, checklist, standing teammates) is the *majority* of the prompt and CL's correction explicitly says to keep it. The proposal's "~500 chars" target is unreachable without dropping the stage definition; CL has now ruled that out. The realistic target is "drop the four pure-boilerplate blocks: Read-instruction prose, Summary template line, Stage Report instructions, Completion Signal surrounding prose."

**No skill-body edits required.** All dropped content already lives in `references/ensign-shared-core.md` (Working, Stage Report Protocol) or `references/claude-ensign-runtime.md` (Completion Signal), both loaded by `spacedock:ensign` on first action. The one addition needed: append the "FO keeps waiting / idle notifications are normal" + FO-forwarding warning to `claude-ensign-runtime.md` `## Completion Signal`.

### Failsafe: skip-skill-load failure mode

`docs/plans/_archive/agent-boot-skill-preload.md` documents the case where haiku-class models boot without loading the `spacedock:ensign` skill — the `skills:` frontmatter preload is blocked for plugin agents by upstream `claude-code#25834`, and the body-fallback `Skill(skill="spacedock:ensign")` invocation can be skipped if the model dives straight into the checklist. A shrunk dispatch prompt that assumes the skill is loaded is fragile against this failure mode.

**Verification (positive signal that the skill loaded):** the dispatch prompt's `## First action` section (section 0 above, kept) explicitly instructs `Skill(skill="spacedock:ensign")` as the first action. Existing tests at `tests/test_claude_team.py:736-771` assert this section appears before the header. We will add one transcript-level test (cheap parser fixture, no live agent run) that confirms the assembled prompt still emits the directive as the first content after the schema header.

**Failsafe (degraded-mode behavior if the skill is NOT loaded):** the kept content alone is enough for an ensign to complete a stage without catastrophic data loss:

- The kept `Read the entity file at {path}` line tells it where the spec is.
- The kept `### Stage definition` block tells it what the stage means.
- The kept `### Completion checklist` tells it the work items.
- The kept worktree+branch declarations keep it inside scope.
- The kept literal `SendMessage(to="team-lead", message="Done: ...")` line is the bare minimum completion signal — even without the surrounding prose, a model that pattern-matches on the parenthesis-equals syntax (which is what the prose currently warns against paraphrasing) will emit it.

What is lost in skip-skill-load + shrunk-prompt mode, compared to skip-skill-load + current-prompt mode: the Stage Report Protocol (DONE/SKIPPED/FAILED markers, append-at-end discipline). Without the skill, the ensign may omit the stage report or write it in the wrong format. This is detectable by the FO (entity file lacks `## Stage Report: {stage}`) and is recoverable by re-dispatching feedback or accepting the entity without a stage report. The data loss is structural (missing report section) not destructive (no overwrites, no wrong-branch commits). **This is acceptable degradation** — the high-stakes invariants (worktree scope, completion-signal syntax, entity path) are preserved in the shrunk prompt.

**No new mechanism needed.** The existing two-path boot (frontmatter preload + body fallback) from `_archive/agent-boot-skill-preload.md` is sufficient; this entity doesn't change boot mechanics. The shrunk prompt simply makes the load-failure case marginally more degraded, in exchange for ~1200 chars saved per dispatch.

### Breakglass FO template consistency

The break-glass manual dispatch template in `skills/first-officer/references/claude-first-officer-runtime.md:114-123` is what the FO uses when `claude-team build` exits non-zero. Its current prompt body (line 120) is:

```
"You are working on: {entity title}\n\nStage: {stage}\n\n### Stage definition:\n\n{copy stage subsection from README verbatim}\n\nRead the entity file at {entity_file_path}.\n\n### Completion checklist\n\n{numbered checklist}\n\n### Summary\n{brief description of what was accomplished}\n\n### Stage report\n\nAppend a Stage Report section at the end of the entity file (per the shared-core Stage Report Protocol). Use the title `Stage Report: {stage}`. Account for every checklist item above with a `- DONE:` / `- SKIPPED:` / `- FAILED:` entry. Use the checklist item text verbatim when possible.\n\n### Completion Signal\n\nSendMessage(to=\"team-lead\", message=\"Done: {entity title} completed {stage}. Report written to {entity_file_path}.\")"
```

The breakglass template currently inlines the `### Summary` template line, the `### Stage report` boilerplate, and the bare Completion Signal SendMessage line. It does NOT carry the `## First action` Skill-invoke directive (notable — breakglass is the only path where the helper's directive is absent).

**Decision:** the breakglass template is **in-scope for this entity** and must be updated to match the shrunk shape. Specifically:

- Drop the `### Summary` line (matches shrink section 7b).
- Drop the `### Stage report` appending-instructions block (matches shrink section 8).
- Keep the bare `SendMessage(to=...)` line (matches shrink section 10 shrunk form).
- **Add** a `Skill(skill="spacedock:ensign")` first-action directive at the head of the prompt body, matching helper output section 0. This closes the existing inconsistency where breakglass dispatches an ensign without the boot directive — a gap that becomes more load-bearing once the surrounding prose in the helper-emitted prompt is dropped.

Update the explanatory paragraph at `claude-first-officer-runtime.md:123` to reflect that the shrunk break-glass also relies on the skill being loaded, and to note that the FO must report helper failures to the captain before using it (this prose already exists at line 113; verify it's preserved).

## Acceptance criteria

1. `cmd_build` in `skills/commission/bin/claude-team` emits a dispatch prompt that contains the `Skill(skill="spacedock:ensign")` first-action directive, the per-dispatch header, stage definition, worktree+branch (when applicable), entity path, feedback context (when applicable), scope notes (when applicable), checklist, standing-teammates block (when applicable), and the bare `SendMessage(to="team-lead", ...)` completion-signal line.
   - Test: extend `tests/test_claude_team.py` with assertions that the four dropped blocks are absent from the emitted prompt: no `### Summary\n{brief description` template, no `### Stage report\n\nAppend a Stage Report section` boilerplate, no `Plain text only. No JSON.` prose, no `**If you are the first officer forwarding this prompt to Agent():**` paragraph.
2. `cmd_build` continues to emit the kept content for the relevant code paths: worktree dispatch keeps the worktree+branch lines; feedback dispatch keeps the `### Feedback from prior review` block; team-mode dispatch keeps the bare `SendMessage(...)` line.
   - Test: existing tests at `tests/test_claude_team.py` covering worktree dispatch, feedback context, and team mode continue to pass; adjust the specific-string assertions where they assert dropped boilerplate (e.g. `assert f"Read the entity file at {expected_entity}" in prompt` becomes `assert f"Entity: {expected_entity}"` or whatever the new shrunk form emits).
3. `skills/first-officer/references/claude-first-officer-runtime.md` break-glass manual dispatch template (currently lines 114-123) matches the new shrunk shape: includes `Skill(skill="spacedock:ensign")` directive, omits `### Summary` + `### Stage report` blocks, retains the bare `SendMessage(to=...)` completion-signal line.
   - Test: a static lint test (similar to existing scaffolding tests) asserts the break-glass prompt body in `claude-first-officer-runtime.md` does NOT contain `Append a Stage Report section at the end of the entity file` and DOES contain `Skill(skill="spacedock:ensign")`.
4. `skills/ensign/references/claude-ensign-runtime.md` `## Completion Signal` section carries the "FO keeps waiting for that explicit completion message; idle notifications are normal between-turn state" clarification, and the FO-forwarding warning ("If you are the first officer forwarding this prompt to Agent(): copy the entire block character-for-character; do not paraphrase the parenthesis-equals syntax").
   - Test: static grep test asserts both phrases appear in `claude-ensign-runtime.md`. (No new mechanism — content-move only.)
5. The dispatch prompt assembled in the test fixture for a typical worktree-backed team-mode dispatch shrinks by roughly 1100-1300 chars vs. baseline (the four dropped blocks).
   - Test: parametrized test that builds a representative dispatch via the helper, asserts emitted prompt char count is at least 1000 chars shorter than a baseline snapshot captured at the start of implementation. (The exact threshold is calibrated in implementation; ~1000 is a floor to catch regressions where someone re-inlines the dropped boilerplate.)

## Test plan

All tests are parser/fixture-level — no live agent runs needed. The shrink is a static text-transformation of the helper output; live behavior is covered by the existing E2E suite (merge hook E2E with haiku/low from `_archive/agent-boot-skill-preload.md`), which exercises the skill-load path that this entity assumes works.

- **`tests/test_claude_team.py`** (existing file, extend): one new test per dropped block (4 tests asserting absence); one test asserting shrunk char-count floor (1 test); one test asserting the bare `SendMessage(...)` line is still emitted in team mode (probably already exists — verify). Update existing tests that assert the literal `Read the entity file at {path}` string to match the new shorter form.
- **New static test** (could live in `tests/test_agent_content.py` or a new `tests/test_breakglass_dispatch_prompt.py`): assert breakglass template in `claude-first-officer-runtime.md` carries the `Skill(skill="spacedock:ensign")` directive and lacks the dropped boilerplate.
- **No new E2E.** The merge hook E2E and rejection flow E2E from `agent-boot-skill-preload.md` already exercise the skill-load path; if they pass with the shrunk prompt, the runtime contract holds. If implementation surfaces a regression there, it's evidence of a real failure mode (skill not loading) — not a test-only failure.

Estimated complexity: small. ~50 lines of Python touched in `claude-team` (string slicing of the 9 `prompt_parts` blocks), ~30 lines of test additions, ~15 lines of edits to `claude-first-officer-runtime.md` and `claude-ensign-runtime.md`. No new modules, no schema changes, no test framework changes.

## Stage Report: ideation

- DONE: Specify exactly which sections of the current `claude-team build` dispatch prompt template are kept, dropped, or moved into the `spacedock:ensign` skill body.
  See `## Design` → "Section-by-section: kept vs. dropped vs. moved" — 11-row table classifies every component emitted by `cmd_build` at `skills/commission/bin/claude-team:334-456` with kept/dropped/moved disposition and one-sentence rationale per row; CL's backlog-gate correction applied (stage definition is KEEP; the operating-contract material — Read-instruction prose, Summary template, Stage Report instructions, Completion-Signal surrounding prose — is DROP).
- DONE: Address the skip-skill-load failure mode named at the backlog gate.
  See `## Design` → "Failsafe: skip-skill-load failure mode" — positive signal (existing `## First action` directive + test at `tests/test_claude_team.py:736-771` asserts it appears before the header) plus an explicit acceptable-degradation argument (worktree scope, completion-signal syntax, and entity path survive a skip-skill-load + shrunk-prompt combination; only the Stage Report Protocol degrades, and the FO can detect and re-dispatch).
- DONE: Confirm the FO's break-glass manual dispatch template stays consistent with the new helper-emitted shape.
  See `## Design` → "Breakglass FO template consistency" — decided in-scope for this entity; specifies the four breakglass edits (drop Summary, drop Stage Report block, keep bare SendMessage, add Skill-invoke directive); acceptance criterion 3 covers it; static lint test in test plan covers it.

### Summary

Ideation reframed the proposal in light of CL's backlog-gate correction: the ~7000→~500 target is unreachable because the per-dispatch content (stage definition, checklist, standing teammates) is the bulk of the prompt and must stay. The realistic shrink drops four pure-boilerplate blocks (Read-instruction prose, Summary template line, Stage Report appending instructions, Completion-Signal surrounding prose) for roughly 1100-1300 chars saved per dispatch. Breakglass FO template is pulled into scope and gains a `Skill(skill="spacedock:ensign")` first-action directive it was previously missing. Skill-load-failure failsafe is argued as acceptable structural-not-destructive degradation — no new boot mechanism needed.
