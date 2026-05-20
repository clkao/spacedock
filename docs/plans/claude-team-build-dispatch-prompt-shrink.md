---
id: 4q934ttakvtv5ngt662pez34
title: "claude-team build dispatch prompts are ~7000 chars; could be ~500"
status: done
source: "GitHub issue #229 — CL observed dispatch prompts inline boilerplate that duplicates `spacedock:ensign` skill body"
started: 2026-05-20T20:34:07Z
completed: 2026-05-20T22:26:57Z
verdict: REJECTED
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

### Section-by-section: kept vs. dropped vs. shrunk

The current dispatch prompt assembled by `cmd_build` in `skills/commission/bin/claude-team` (lines 334–456) has 11 components (the source code calls these `prompt_parts` and numbers its comments 0/1/2/3/4/6/7/8/9/10 — comment "5" is reserved and unused; the table below uses the source comment numbers so each row's `claude-team:LINE` citation matches the comment number in the source). Each row is classified as **per-dispatch content** (varies by entity/stage/team — keep) or **operating-contract content** (boilerplate identical across dispatches at the same stage — drop, because `spacedock:ensign` already covers it via skill).

| `prompt_parts` # | Section (source line) | Disposition | Rationale |
|---|---|---|---|
| 0 | `## First action` — `Skill(skill="spacedock:ensign")` directive (`claude-team:338-350`) | **KEEP** | Belt-and-suspenders for the skill-preload failure mode (see Failsafe below); tests at `tests/test_claude_team.py:736-771` already assert it appears before the header. |
| 1 | `You are working on: {title}` / `Stage: {stage}` header (`claude-team:353`) | **KEEP** | Per-dispatch identifiers; ensign needs them on first read. |
| 2 | `### Stage definition:` + verbatim README subsection (`claude-team:356`) | **KEEP** | Per-dispatch: stage definitions vary across `ideation`/`validation`/`implementation`/etc. CL's correction at the backlog gate: this is the actually-informative content that justifies the dispatch. |
| 3 | Worktree path + branch declarations (`claude-team:359-366`, conditional) | **KEEP** | Per-dispatch: worktree path and branch name derive from the entity. Cannot move to the skill body (skill is static). |
| 4 | `Read the entity file at {path} for the current spec.` (`claude-team:369-379`) | **SHRINK** | Keep the path (per-dispatch). Drop the "for the current spec" / "for the full spec. It contains:" trailer — `ensign-shared-core.md:16` already says "Read the entity file before making changes." New form (one line): `Read the entity file at {path}.` Net savings ~220 chars. |
| 6 | `### Feedback from prior review` block (`claude-team:382-383`, conditional) | **KEEP** | Per-dispatch content; this IS the feedback. |
| 7 | Scope notes (`claude-team:386-387`, conditional) | **KEEP** | Per-dispatch content; FO supplies them when present. |
| 8a | `### Completion checklist` header + checklist items (`claude-team:391-393`) | **KEEP** | Per-dispatch: the checklist IS the stage's specific work. |
| 8b | `### Summary\n{brief description of what was accomplished}` template (`claude-team:394-395`) | **KEEP** (revised from cycle 1) | Prior art at `_archive/claude-team-inject-skill-invoke.md` rows P8c (line 76) and design notes (lines 101, 115) classifies this line as KEEP: "the per-dispatch checklist items in P8e and the `### Summary` slot is legitimately per-dispatch and belongs in the spawner." The Summary slot is the scaffolding that tells the ensign where in the stage report to write the 2-3-sentence narrative; that's per-dispatch context (it sits adjacent to the per-dispatch checklist), not a duplicate of the Stage Report Protocol. Cycle 1 was wrong to drop it; reclassify KEEP. |
| 8c | `### Stage report` appending instructions (`claude-team:397-403`) | **DROP** | The entire Stage Report Protocol — append-at-end, DONE/SKIPPED/FAILED markers, verbatim-checklist-item-text rule, `(cycle N)` re-dispatch naming — already lives in `ensign-shared-core.md:48-77`. Prior art (`claude-team-inject-skill-invoke.md` rows P8a/P8b/P8d, design lines 101) reaches the same conclusion: spawner removes the duplicative imperatives, keeps Summary scaffolding. Drop ~400 chars. |
| 9 | `### Standing teammates available in your team` (`claude-team:406-437`, conditional) | **KEEP** | Per-dispatch: the set of standing teammates and their routing-usage bodies vary by team. Cannot move to the static skill body. |
| 10a | Per-dispatch `SendMessage(to="team-lead", message="Done: {title} completed {stage}. Report written to {path}.")` literal (`claude-team:446-447`) | **KEEP** | The entity_title/stage/entity_file_ref are per-dispatch and the literal parenthesis-equals syntax is what the ensign must emit. Prior art P10a (line 80) flags this as "duplicative with runtime adapter, aligned with shared-core" — but the runtime-adapter template (`claude-ensign-runtime.md:21-23`) uses placeholders (`{entity title}`, `{stage}`, `{entity_file_path}`) and cannot substitute for the spawner's fully-resolved literal. KEEP. |
| 10b | Surrounding "Plain text only. No JSON. Until you send this message…" prose (`claude-team:448-450`) | **DROP** | Plain-text/no-JSON rule already lives in `claude-ensign-runtime.md:25` ("Plain text only. Never send JSON.") which the skill loads. The "FO keeps waiting / idle notifications are normal" elaboration is operational-context narrative the ensign doesn't need at every dispatch. Net drop ~370 chars. |
| 10c | `**If you are the first officer forwarding this prompt to Agent():** copy the entire block above into `Agent(prompt=...)` character-for-character…` (`claude-team:451-456`) | **KEEP** (revised from cycle 1) | Prior art P10c (line 82) explicitly classifies this as "missing-in-core — its audience is the FO that is about to forward the prompt, not the ensign. Keep it in the spawner." Cycle 1's proposal to move it into `claude-ensign-runtime.md` was an inversion: that file loads in the ensign's context, not the FO's, so a safeguard against FO paraphrase belongs in the FO-readable spawner output, not in the ensign-loaded runtime adapter. KEEP. |

**Two findings cascade from this revision (cycle-1 → cycle-2):**

- Row 8b (Summary slot) is reclassified KEEP. The breakglass template (below) is no longer required to drop the `### Summary` line.
- Row 10c (FO-forwarding warning) is KEEP. No move into `claude-ensign-runtime.md` is needed; that runtime adapter requires no edits as part of this entity. (Cycle 1 proposed moving both 10b and 10c into the runtime adapter; cycle 2 keeps 10c in the spawner and treats 10b's plain-text rule as already-covered by the existing `claude-ensign-runtime.md:25` content, no move needed.)

**Net before/after (recomputed for cycle 2).** Taking the dispatch this ensign received at the top of session as the worked example (6900 chars baseline, team-mode + worktree-backed + feedback-context + 1 standing teammate). Section-by-section char counts after shrink (rounded):

| `prompt_parts` # | Before | After | Delta |
|---|---|---|---|
| 0 (First action) | 600 | 600 | 0 |
| 1 (header) | 80 | 80 | 0 |
| 2 (stage definition) | 1500 | 1500 | 0 |
| 3 (worktree+branch) | 300 | 300 | 0 |
| 4 (entity-read line) | 300 | 80 | -220 |
| 6 (feedback) | 200 | 200 | 0 |
| 7 (scope notes) | 600 | 600 | 0 |
| 8a (checklist) | 800 | 800 | 0 |
| 8b (Summary slot) | 70 | 70 | 0 (KEEP, revised) |
| 8c (Stage Report block) | 400 | 0 | -400 |
| 9 (standing teammates) | 1200 | 1200 | 0 |
| 10a (literal SendMessage) | 250 | 250 | 0 |
| 10b (plain-text prose) | 370 | 0 | -370 |
| 10c (FO-forwarding warning) | 350 | 350 | 0 (KEEP, revised) |
| **Total** | **~7020** | **~6030** | **~-990** |

So cycle-2 expected savings are roughly 1000 chars per worked-example dispatch (~14% of baseline), three drops totalling about 990 chars: row 4 trailer (-220), row 8c (-400), row 10b (-370). The "7000 → 500" target in the original GitHub issue is unreachable; the realistic target is "drop the three pure-boilerplate blocks identified above." This is consistent with CL's backlog-gate framing that the actual char count falls out of which sections are kept, not the other way around.

**No skill-body edits required.** All dropped content already lives in `references/ensign-shared-core.md` (Working step 1 at line 16, Stage Report Protocol at lines 48-77) or `references/claude-ensign-runtime.md` (`## Completion Signal` at lines 19-25), both loaded by `spacedock:ensign` on first action. No new phrases are moved into either file as part of this entity.

### Failsafe: skip-skill-load failure mode

`docs/plans/_archive/agent-boot-skill-preload.md` documents the case where haiku-class models boot without loading the `spacedock:ensign` skill — the `skills:` frontmatter preload is blocked for plugin agents by upstream `claude-code#25834`, and the body-fallback `Skill(skill="spacedock:ensign")` invocation can be skipped if the model dives straight into the checklist. A shrunk dispatch prompt that assumes the skill is loaded is fragile against this failure mode.

**Verification (positive signal that the skill loaded):** the dispatch prompt's `## First action` directive (row 0, kept) explicitly instructs `Skill(skill="spacedock:ensign")` as the first action. Existing tests at `tests/test_claude_team.py:736-771` assert this section appears before the header in helper output. Acceptance criterion 5 (below) adds a structural check that the directive remains the first prompt_part in cycle-2-shrunk output, so the positive-signal test surface does not regress.

**Failsafe (degraded-mode behavior if the skill is NOT loaded).** The high-stakes invariants must survive a skip-skill-load + cycle-2-shrunk-prompt combination without data corruption. Walk-through:

- Row 1 header tells the ensign what entity and stage it's working.
- Row 2 stage-definition block tells it what the stage means (per-dispatch content; the skill doesn't carry stage definitions even when loaded).
- Row 3 worktree+branch declarations keep it inside scope.
- Row 4 entity-path line (shrunk to `Read the entity file at {path}.`) still tells it where the spec is.
- Row 8a checklist tells it the work items.
- Row 10a literal `SendMessage(to="team-lead", message="...")` is the bare minimum completion signal.

What's lost vs. the current prompt under skip-skill-load:

- Stage Report Protocol scaffolding (row 8c drop). Without the skill, the ensign may omit the stage report or write it in the wrong format. This is **structural** (missing/malformed `## Stage Report: {stage}` section), **detectable** (FO grep on the entity file post-completion), and **recoverable** (re-dispatch feedback). No destructive failure mode.
- Plain-text-only completion-signal reminder (row 10b drop). Without the skill, the ensign might emit a structured-JSON completion message that the FO's message-handling code doesn't parse. This is **functional** (FO times out waiting for plain-text completion), **detectable** (FO inbox shows a JSON-formatted Done message), and **recoverable** (FO can interpret the JSON manually or re-dispatch). No destructive failure mode either.

**Acknowledged paraphrase-risk weakening of the failsafe (Finding 2):** the reviewer is correct that the FO-forwarding warning (row 10c) and the plain-text rule (row 10b) exist because of observed paraphrase risk in haiku, and cycle 1's proposed move into `claude-ensign-runtime.md` was exactly into the place that doesn't load when the safeguard is needed. Cycle 2 fixes this by **keeping** row 10c in the spawner (its audience is the FO, not the ensign) and **dropping** row 10b without moving it — the equivalent plain-text rule already lives in `claude-ensign-runtime.md:25`. That rule still doesn't load in skip-skill-load mode, but row 10b's specific "FO keeps waiting / idle notifications are normal" elaboration is operational-context narrative, not a behavioral guard; losing it under skip-skill-load is a smaller cost than losing row 10c's character-for-character paraphrase guard would be. The failsafe **is** marginally weaker than the current prompt under skip-skill-load: the specific weakness is "ensign might emit a structured-JSON completion message instead of plain text," which is functional and recoverable rather than destructive.

**No new mechanism needed.** The existing two-path boot (frontmatter preload + body fallback) from `_archive/agent-boot-skill-preload.md` is sufficient; this entity doesn't change boot mechanics.

### Breakglass FO template consistency

The break-glass manual dispatch template in `skills/first-officer/references/claude-first-officer-runtime.md:114-123` is what the FO uses when `claude-team build` exits non-zero. Its current prompt body (line 120) is:

```
"You are working on: {entity title}\n\nStage: {stage}\n\n### Stage definition:\n\n{copy stage subsection from README verbatim}\n\nRead the entity file at {entity_file_path}.\n\n### Completion checklist\n\n{numbered checklist}\n\n### Summary\n{brief description of what was accomplished}\n\n### Stage report\n\nAppend a Stage Report section at the end of the entity file (per the shared-core Stage Report Protocol). Use the title `Stage Report: {stage}`. Account for every checklist item above with a `- DONE:` / `- SKIPPED:` / `- FAILED:` entry. Use the checklist item text verbatim when possible.\n\n### Completion Signal\n\nSendMessage(to=\"team-lead\", message=\"Done: {entity title} completed {stage}. Report written to {entity_file_path}.\")"
```

The breakglass template currently inlines the `### Summary` template line, the `### Stage report` boilerplate, and the bare Completion Signal SendMessage line. It does NOT carry the `## First action` Skill-invoke directive (notable — breakglass is the only path where the helper's directive is absent).

**Decision: the breakglass template is in-scope for this entity.** Edits needed to match the cycle-2 shrunk helper-output shape:

- **Keep** the `### Summary` line (cycle 1 said drop; cycle 2 keeps it to align with row 8b reclassification).
- **Drop** the `### Stage report` appending-instructions block (matches row 8c).
- **Keep** the bare `SendMessage(to=...)` line (matches row 10a; the breakglass template never carried row 10b or row 10c prose, so no changes there).
- **Add** a `Skill(skill="spacedock:ensign")` first-action directive at the head of the prompt body (matches row 0). This closes the existing inconsistency where breakglass dispatches an ensign without the boot directive — a gap that becomes more load-bearing once the surrounding prose in helper-emitted prompts is dropped.

**Stale-prose audit on `claude-first-officer-runtime.md` (Finding 7).** Walking the file to name any sentence rendered stale or contradicted by adding `Skill(skill="spacedock:ensign")` to the breakglass template and dropping the `### Stage report` block:

- Line 73 (MANDATORY assembly): "The key fields that MUST come from helper output are `subagent_type`, `name`, `team_name`, `model`, and `prompt` (which contains the completion signal). Manual assembly is a protocol violation except in the documented break-glass fallback below." **Not stale** — still true; the breakglass change doesn't relax this.
- Line 113 (breakglass intro): "Do NOT use this template while the helper is working. Report the helper failure to the captain before proceeding. Use this minimal template as a degraded fallback." **Not stale** — still applies; the cycle-2 breakglass is still a degraded fallback.
- Line 123 (breakglass explanatory): "The break-glass template omits worktree instructions, feedback context, and scope notes. The `model=` slot is conditional — include it only when the stage (or `stages.defaults`) declares a model from `sonnet | opus | haiku`; omit the entire `model=` argument otherwise. Use only when the helper is unavailable." **Mostly not stale**, but the cycle-2 breakglass adds further intentional omissions relative to helper output: the standing-teammates block (row 9), the FO-forwarding warning (row 10c), and per-stage operational prose. Cycle 2 will edit this paragraph to enumerate the full omission list and to note that the breakglass relies on the newly-added `Skill(skill="spacedock:ensign")` directive to load the operating contract. The exact wording is left to implementation; AC-3's test anchors on structural requirements (each omission term appears; the Skill-directive reliance is mentioned), not a literal draft sentence.
- Lines 16-46 (`## Team Lifecycle and Naming`): no mention of breakglass content. **Not stale.**
- Lines 125-156 (`## Degraded Mode`): describes session-wide bare-mode after team-infrastructure failure; orthogonal to breakglass per-dispatch fallback. **Not stale.**

No other downstream prose is contradicted by the cycle-2 breakglass change. The audit confirms the change is contained: breakglass body + one explanatory sentence at line 123. No upstream rule about helper-output shape needs to change (the helper still emits everything in rows 0-10c that the breakglass omits; that asymmetry was already documented and is preserved).

## Acceptance criteria

1. The helper-emitted prompt from `cmd_build` for a representative team-mode worktree-backed feedback dispatch (the fixture defined in AC-5) lacks the three dropped blocks named in the section table above: the `Read the entity file at {path} for the current spec.` long-form trailer (replaced by the one-line shrunk form), the `### Stage report\n\nAppend a Stage Report section…` block, and the `Plain text only. No JSON. Until you send this message, the first officer keeps waiting…` paragraph.
   - Test: `tests/test_claude_team.py` extension. Substring-absence assertion on each of the three dropped phrases. **Plus a structural marker** (Finding 6): assert that for the cycle-2 fixture, no line in the emitted prompt starts with `### Stage report` (any whitespace), and that the emitted prompt contains exactly one occurrence of `### Summary` (row 8b, KEEP) and exactly one occurrence of `### Completion Signal` (row 10's header, where applicable in team mode).
2. The helper-emitted prompt for the same fixture contains the kept rows: `Skill(skill="spacedock:ensign")` directive (row 0), `You are working on:` header (row 1), `### Stage definition:` (row 2), worktree+branch lines (row 3, conditional), the shrunk one-line entity-read instruction (row 4), `### Feedback from prior review` block (row 6, conditional), scope notes block (row 7, conditional), `### Completion checklist` header (row 8a), `### Summary` line (row 8b), `### Standing teammates available in your team` (row 9, conditional), the literal `SendMessage(to="team-lead", message="Done: ...")` line (row 10a), and the `**If you are the first officer forwarding this prompt to Agent():**` paragraph (row 10c).
   - Test: `tests/test_claude_team.py` extension. Substring-presence assertion on each of the 12 kept items. Existing assertions that match `Read the entity file at {expected_entity} for the current spec` (file currently at `tests/test_claude_team.py:856`, `:914`, `:931`) are updated to match the cycle-2 one-line shrunk form.
3. The break-glass manual dispatch template at `skills/first-officer/references/claude-first-officer-runtime.md:114-123` contains `Skill(skill="spacedock:ensign")` as the first prompt-body element, retains the `### Summary` slot, omits any `Append a Stage Report section at the end of the entity file` text, and retains the bare `SendMessage(to=\"team-lead\", ...)` completion-signal line.
   - Test: a new static lint test (new file `tests/test_breakglass_dispatch_prompt.py` or extension of `tests/test_agent_content.py`) reads the runtime-adapter file, extracts the breakglass template body (the string literal in the `prompt=` slot), and asserts the four properties above as substring presence/absence checks. Also asserts the explanatory paragraph adjacent to the breakglass template enumerates **all** breakglass omissions relative to helper output — at minimum: worktree instructions, feedback context, scope notes, the standing-teammates block, the FO-forwarding warning, and per-stage operational prose — and mentions that the breakglass relies on the `Skill(skill="spacedock:ensign")` directive to load the operating contract. The test asserts these as structural requirements (each omission term appears; the Skill-directive reliance is mentioned) rather than matching a literal draft sentence, so the explanatory paragraph can be reworded during PR review without breaking the test.
4. The dispatch prompt golden file at `tests/fixtures/dispatch_prompt_post_shrink.txt` exists, was checked in as part of the cycle-2 implementation commit (not generated at test-run time), and is byte-for-byte equal to the helper output for the canonical fixture inputs.
   - Test: a single golden-file diff test in `tests/test_claude_team.py` that pipes the canonical fixture inputs through `cmd_build` and asserts the emitted prompt equals the file contents at `tests/fixtures/dispatch_prompt_post_shrink.txt`. The fixture inputs (canonical entity, stage, checklist, etc.) are also checked into the tests/fixtures directory. **Baseline-pinning rationale (Finding 4):** the golden file is the regression boundary; if a future commit re-inlines any dropped boilerplate, the golden-diff fails immediately with the offending bytes shown. The previous cycle-1 AC-5 ("at least 1000 chars shorter than a baseline snapshot captured at the start of implementation") is removed because the baseline was self-fulfilling.
5. The helper-emitted prompt's first non-blank content line (after the schema header, if any) is the `## First action` heading from row 0, immediately followed by the `Skill(skill="spacedock:ensign")` literal.
   - Test: `tests/test_claude_team.py` extension building on the existing `tests/test_claude_team.py:736-771` assertions. Adds a positive-position check: the emitted prompt's first heading line is `## First action` and the `Skill(skill="spacedock:ensign")` literal appears within the next 200 characters. This is the load-path positive signal named in the Failsafe section.

**Finding-5 note (load-path coverage for moved content):** cycle 2 moves NO new phrases into `claude-ensign-runtime.md`. The previously-proposed AC-4 (static grep on the runtime adapter for the moved "FO keeps waiting" and FO-forwarding phrases) is removed because the move is removed. No new load-path E2E coverage is needed; the existing merge hook E2E (`tests/test_merge_hook_guardrail.py`) and rejection flow E2E (`tests/test_rejection_flow.py`) already exercise the skill-load path for the unchanged `claude-ensign-runtime.md` content. If implementation later finds that skip-skill-load mode is more common than this design assumes, that's a separate entity (extend boot mechanics) — out of scope here.

## Test plan

All tests are parser/fixture-level — no live agent runs needed for the cycle-2 shrink itself. The shrink is a static text-transformation of helper output; live behavior is exercised by the existing E2E suite, which is sufficient because the changed runtime contract (drop three boilerplate blocks) does not interact with stage-execution behavior.

- **`tests/test_claude_team.py`** (existing file, extend): 1 absence-assertion test for each of the three dropped phrases (3 tests); 1 structural-marker test for the `### Stage report` line-prefix and `### Summary` / `### Completion Signal` occurrence counts (1 test, AC-1's tail); 12 presence-assertion tests for the kept rows (typically batched into one parameterized test, AC-2); 1 first-heading positional test (AC-5); 1 golden-file diff test against `tests/fixtures/dispatch_prompt_post_shrink.txt` (AC-4). Update existing tests at `tests/test_claude_team.py:856`, `:914`, `:931` to the cycle-2 one-line entity-read form (3 line edits).
- **`tests/fixtures/dispatch_prompt_post_shrink.txt`** (new file, checked in as part of cycle-2 implementation): canonical golden output. Fixture inputs (entity file, stage name, checklist) are also new files under `tests/fixtures/`. The golden file is committed first, in the same commit that lands the `cmd_build` change, so the golden-diff test fails atomically on any subsequent re-inlining regression.
- **`tests/test_breakglass_dispatch_prompt.py`** (new file, or extension of `tests/test_agent_content.py`): 1 test asserting breakglass template at `claude-first-officer-runtime.md:114-123` carries the `Skill(skill="spacedock:ensign")` directive, retains `### Summary`, omits the `### Stage report` block, retains the bare `SendMessage(...)` line, and the explanatory paragraph at line 123 enumerates the cycle-2 omissions (AC-3).
- **No new E2E.** Existing merge hook E2E (`tests/test_merge_hook_guardrail.py`) and rejection flow E2E (`tests/test_rejection_flow.py`) cover the skill-load path that this entity assumes works. If they pass with the shrunk prompt, the skill-load contract is intact; if implementation surfaces a regression there, it's evidence of a real failure mode worth investigating, not a test-only failure.

Estimated complexity: small. ~30 lines of Python touched in `skills/commission/bin/claude-team` (drop three `prompt_parts.append(...)` segments and shrink the entity-read line); ~50 lines of test additions; ~10 lines of edits to `claude-first-officer-runtime.md` (breakglass body + line-123 explanatory sentence); no edits to `claude-ensign-runtime.md`. No new modules, no schema changes, no test framework changes.

## Stage Report: ideation

- DONE: Specify exactly which sections of the current `claude-team build` dispatch prompt template are kept, dropped, or moved into the `spacedock:ensign` skill body.
  See `## Design` → "Section-by-section: kept vs. dropped vs. moved" — 11-row table classifies every component emitted by `cmd_build` at `skills/commission/bin/claude-team:334-456` with kept/dropped/moved disposition and one-sentence rationale per row; CL's backlog-gate correction applied (stage definition is KEEP; the operating-contract material — Read-instruction prose, Summary template, Stage Report instructions, Completion-Signal surrounding prose — is DROP).
- DONE: Address the skip-skill-load failure mode named at the backlog gate.
  See `## Design` → "Failsafe: skip-skill-load failure mode" — positive signal (existing `## First action` directive + test at `tests/test_claude_team.py:736-771` asserts it appears before the header) plus an explicit acceptable-degradation argument (worktree scope, completion-signal syntax, and entity path survive a skip-skill-load + shrunk-prompt combination; only the Stage Report Protocol degrades, and the FO can detect and re-dispatch).
- DONE: Confirm the FO's break-glass manual dispatch template stays consistent with the new helper-emitted shape.
  See `## Design` → "Breakglass FO template consistency" — decided in-scope for this entity; specifies the four breakglass edits (drop Summary, drop Stage Report block, keep bare SendMessage, add Skill-invoke directive); acceptance criterion 3 covers it; static lint test in test plan covers it.

### Summary

> **Superseded by `## Stage Report: ideation (cycle 2)` below.** The paragraph that originally appeared here described cycle 1's four-drop / 1100-1300-chars-saved framing; cycle 2 revised that to a three-drop / ~990-chars-saved disposition (rows 8b and 10c reclassified KEEP). The cycle-1 framing is preserved in the git history (commit `ccd84011`) but is not the current design. Read the cycle-2 stage report for the authoritative cycle-summary narrative.

### Feedback Cycles

#### Cycle 1 — rejected at ideation gate (2026-05-20)

Staff reviewer surfaced 7 material findings undermining the soundness of cycle-1 ideation. Rerouted to the same ensign via SendMessage (reuse OK: 5.9% of 1M context budget). Findings to address in cycle 2:

1. **Prior-art collision on `### Summary` line (row 7b).** `_archive/claude-team-inject-skill-invoke.md` lines 76/101/115 classified the Summary line as KEEP ("legitimately per-dispatch scaffolding"). Cycle 1 drops it without engaging the prior reasoning. Either cite evidence the prior keep-decision was wrong, or reclassify 7b as KEEP.
2. **Failsafe argument inverts the safety contract.** The prose warning proposed for move into `claude-ensign-runtime.md` exists because paraphrase risk was observed in haiku — but the skill, by definition, doesn't load in the failure mode the safeguard protects against. Putting the safeguard into the unloaded place is backwards. Either weaken the failsafe claim or scope-out the Completion-Signal-prose drop.
3. **3 of 5 AC are stage-imperative ("emit", "continues to emit", "shrinks by ... vs baseline") rather than entity-level end-state.** Per the README's explicit AC guidance, rephrase as end-state properties of the finished entity ("the helper output for a representative dispatch lacks the four enumerated blocks," "the helper output is at least 1000 chars shorter than the captured baseline golden file," etc.).
4. **AC-5 baseline is movable.** "Calibrated in implementation" + "captured at implementation start" makes the threshold self-fulfilling. Pin the baseline as a golden file checked in before implementation, or drop AC-5 and let AC-1's structural assertions carry the load.
5. **AC-4 test (static grep on `claude-ensign-runtime.md`) proves content moved but not that the load-path delivers it.** The failsafe depends on skill load. Either name an existing E2E assertion that catches a missing skill load of the moved phrases, or add one.
6. **AC-1 test is a regression-fingerprint, not a structural check.** Four exact substrings would pass if a future author re-inlines different boilerplate covering the same semantics. Add a structural marker (e.g., assert no line starts with `### Stage report`).
7. **Breakglass scope-in introduces a coordination dependency not enumerated.** Adding `Skill(skill="spacedock:ensign")` to the FO breakglass template is a behavioral change, not just a consistency edit. The "verify it's preserved" prose at entity line 121 hand-waves it — actually verify and name any downstream prose the change makes stale.

Polish-tier findings (non-blocking but worth fixing): arithmetic reconciliation (~6900 baseline − ~4660 kept-content claim ≠ ~1200 saved claim); two off-by-small line citations (`ensign-shared-core.md:15-17` should be 16; `claude-team:368-379` should be 369-379); entity table row labels don't align with source `prompt_parts` comment numbers; "~50 lines" + "9 prompt_parts blocks" inconsistency (table has 11).

#### Cycle 2 — re-reviewed; approved with one hygiene fix (2026-05-20)

Re-review verified all 7 cycle-1 material findings as genuinely addressed (not papered over), prior-art citations on the two reversals (rows 8b, 10c) verified verbatim, all 4 polish findings fixed. Reviewer's closing line: "cycle-2 ideation is materially solid; the single remaining material finding is a documentation-hygiene issue, not a design soundness issue."

Captain approved cycle 2 and chose to route a quick hygiene pass back as cycle 3 (option a) rather than absorb into implementation. Reuse OK (10.2% of 1M context budget). Cycle 3 brief:

1. **Drop or mark-superseded the stale cycle-1 Summary paragraph at entity lines 179-190.** That paragraph still claims "four pure-boilerplate blocks / 1100-1300 chars saved" — cycle 2's actual disposition is three drops / ~990 chars. The cycle-2 stage report supersedes it correctly, but the contradiction is a documentation hazard for later readers (especially PR reviewers reading top-to-bottom). Either edit in place to match cycle-2 reality, or strike with a one-line "superseded by Stage Report: ideation (cycle 2) below" note.
2. **Sanity-check `tests/test_claude_team.py:914` and `:931` line citations in AC-2.** Reviewer verified `:856` exists; the other two are unverified. Open the test file, confirm the lines match the cited assertions on the `Read the entity file at {expected_entity} for the current spec` phrase, and either confirm in place or correct.

Polish-tier (also do these while you're in there if cheap; absorb into implementation if not): rephrase AC-3 to specify structural requirements ("the explanatory paragraph enumerates the four omissions; mentions the Skill-directive reliance") instead of asserting a literal draft sentence at line 123 of the runtime adapter — coupling the test to draft wording is brittle. Split the 7-line Failsafe acknowledgment paragraph at entity line 122 into two sentences for readability.

#### Cycle 3 — captain rejected at implementation gate; restructuring scope (2026-05-20)

Cycle-3 ideation hygiene pass landed clean, FO advanced to implementation gate. At that gate the captain pushed back: the cycle-2 design's ~990 char savings is too small a target. Real-prompt measurement (saved at `/tmp/dispatch-4q9-prompt.txt`) showed the bulk of the prompt is per-dispatch content the FO already authors plus three large inlined-but-fetchable blocks — stage definition (2639 chars, 30.8%), standing teammates section (1699 chars, 19.9%), and boot directive (460 chars, 5.4%). Per-dispatch FO context burn is ~17,500 chars total of which ~12,700 chars carries content the FO never uses for its own reasoning. The cycle-1/2/3 design was the wrong target by an order of magnitude.

Captain's reframe: have `claude-team build` emit fetch commands (sed-style line-range invocations) the ensign runs on first action instead of inlining content. Saves prompt size for the ensign AND saves FO context (helper output + Agent prompt args = 3x cost). The cycle-2 design's correctness work (breakglass-template fix + structural-marker test idea) is sound but orthogonal to the bigger restructure.

Entity 4q9 is rejected and replaced by two successor entities:

- **`0x93enxe1hpmk95a25476zyn`** (claude-team-build-fetch-on-demand-dispatch-spec) — the bigger restructure. Ideation will include an actual spike to prove fetch-on-demand works before locking the design.
- **`2x6ra77h668vwr3gxc7zh80f`** (fo-breakglass-template-skill-invoke-directive) — small standalone carve-out of the one cycle-2 finding orthogonal to the restructure. If the bigger restructure absorbs it, this entity gets rejected without code change.

Cycle 1-3 work is preserved in git history (`ccd84011`, `ccb442d5`, `cb4bef12`) as the scoping that surfaced the wrong target.

## Stage Report: ideation (cycle 2)

- DONE: Engage with `_archive/claude-team-inject-skill-invoke.md` on the `### Summary` line (row 7b → row 8b). Reclassify or argue with evidence.
  Reclassified to KEEP in cycle-2 design table at row 8b. Cycle 1 was wrong: prior-art P8c (`_archive/claude-team-inject-skill-invoke.md:76`) and design notes (lines 101, 115) explicitly classify the Summary slot as "legitimately per-dispatch scaffolding" that belongs in the spawner. Cycle 2 adopts that classification. Breakglass template revision (cycle 1: drop `### Summary`) is reversed accordingly.
- DONE: Fix the failsafe-argument inversion (Completion-Signal prose move was backwards).
  Cycle 2 scopes out the move entirely. Row 10c (FO-forwarding warning, audience = FO not ensign) is reclassified KEEP-in-spawner per prior-art P10c (line 82: "missing-in-core — its audience is the FO that is about to forward it. Keep it in the spawner"). Row 10b (plain-text/no-JSON prose) is dropped without moving: the equivalent rule already lives in `claude-ensign-runtime.md:25`. The Failsafe section explicitly acknowledges the marginal weakening under skip-skill-load (functional, recoverable failure mode — not destructive).
- DONE: Rephrase AC-1, AC-2, AC-5 as entity-level end-state properties.
  AC-1 now reads "The helper-emitted prompt … lacks the three dropped blocks" (end-state property of output). AC-2 now reads "The helper-emitted prompt … contains the kept rows" (end-state property). AC-5 now reads "The helper-emitted prompt's first non-blank content line … is the `## First action` heading" (end-state property). The cycle-1 verb phrases ("emits", "continues to emit", "shrinks by ... vs baseline") are gone.
- DONE: Pin AC-5's baseline (cycle 1) — golden file checked in before implementation.
  Cycle-1 AC-5 (char-count floor with self-calibrated baseline) is removed. Replaced by cycle-2 AC-4: a golden file at `tests/fixtures/dispatch_prompt_post_shrink.txt`, checked in as part of the cycle-2 implementation commit (not generated at test time). Single golden-diff test in `tests/test_claude_team.py` carries the regression-resistance load. AC-1's three substring-absence checks + the new structural marker carry the semantic-regression load on top of that.
- DONE: Strengthen AC-4 (cycle 1 — runtime-adapter content move) beyond a file-grep.
  Cycle-1 AC-4 is removed entirely because the move is scoped out (Finding 2). The Finding-5 note in the AC section explicitly states "cycle 2 moves NO new phrases into `claude-ensign-runtime.md`; no new load-path E2E coverage is needed" and names the existing merge-hook E2E and rejection-flow E2E as the load-path coverage for the unchanged skill content.
- DONE: Harden AC-1's test with a structural marker beyond substring fingerprints.
  AC-1's test now includes a structural marker assertion: "no line in the emitted prompt starts with `### Stage report`" (line-prefix structural check, catches re-inlining via different wording), plus occurrence counts on `### Summary` (exactly 1) and `### Completion Signal` (exactly 1 in team mode). The substring-absence assertions remain as the specific regression fingerprints; the structural marker catches semantically-equivalent re-inlines.
- DONE: Verify the breakglass coordination claim — walk `claude-first-officer-runtime.md` and name stale prose.
  Cycle-2 design `### Breakglass FO template consistency` section now contains a "Stale-prose audit" subsection that walks every section of `claude-first-officer-runtime.md` touching the breakglass template (lines 65-67, 73, 113, 123, 125-156). Finding: line 123's enumeration of breakglass omissions becomes incomplete and must be edited to enumerate standing-teammates block, FO-forwarding warning, and per-stage operational prose. One concrete sentence-level edit, added to the breakglass scope. No other downstream prose contradicts the cycle-2 change.

### Summary

Cycle 2 reverses two cycle-1 misclassifications (Summary slot row 8b, FO-forwarding warning row 10c) where prior art at `_archive/claude-team-inject-skill-invoke.md` had already correctly classified them KEEP. The Completion-Signal-prose-move into `claude-ensign-runtime.md` is scoped out entirely — the move was backwards relative to the skip-skill-load failure mode it claimed to protect against. The resulting cycle-2 shrink is three drops (entity-read trailer, Stage Report block, plain-text-only prose) totalling about 1000 chars per worked-example dispatch, down from cycle 1's claimed 1200. Acceptance criteria are rewritten as end-state properties, anchored by a golden-file regression test checked in before implementation. The breakglass coordination audit names one downstream sentence (line 123 enumeration) that must be updated alongside the template body.

## Stage Report: ideation (cycle 3)

- DONE: Drop or mark-superseded the stale cycle-1 Summary paragraph at entity lines 179-190.
  Replaced the cycle-1 Summary paragraph (under `## Stage Report: ideation` → `### Summary`) with a blockquote supersede-note pointing readers to `## Stage Report: ideation (cycle 2)` below. Preserves the audit trail (cycle-1 framing remains in commit `ccd84011`) without leaving the contradictory four-drop / 1100-1300-chars claim above the cycle-2 stage report.
- DONE: Sanity-check `tests/test_claude_team.py:914` and `:931` line citations in AC-2.
  Ran `grep -n "Read the entity file at" tests/test_claude_team.py`. Confirmed all three lines exist with the cited assertions: `:856` and `:914` both match `assert f"Read the entity file at {expected_entity}" in out["prompt"]`; `:931` matches `assert f"Read the entity file at {entity}" in out["prompt"]` — note `:931` uses `{entity}` rather than `{expected_entity}` as the format variable, but the assertion shape is the same. AC-2's citation to all three line numbers is accurate; no entity edit needed. (Implementation will need to update all three; this is already in AC-2's "Update existing tests… (3 line edits)".)
- DONE: Polish — rephrase AC-3's "the explanatory paragraph enumerates the cycle-2 omissions" from a literal draft sentence to structural requirements.
  AC-3's test description now reads "asserts these as structural requirements (each omission term appears; the Skill-directive reliance is mentioned) rather than matching a literal draft sentence, so the explanatory paragraph can be reworded during PR review without breaking the test." The corresponding text in the design `### Breakglass FO template consistency` section (line 147 area) was also updated: "The exact wording is left to implementation; AC-3's test anchors on structural requirements." Both no longer pre-commit to a draft sentence.
- DONE: Polish — split the 7-line Failsafe acknowledgment paragraph at entity line 122 into two sentences.
  Split into three sentences (one was already manageable). First sentence: reviewer correctness + cycle-1 inversion description. Second: cycle-2 fix (keep row 10c, drop row 10b without moving) + rationale. Third: the marginal-weakness acknowledgment. Reads more cleanly than the original 7-line single sentence; no semantic change.

### Summary

Cycle 3 is a documentation-hygiene pass on top of cycle-2 ideation, which was approved as materially solid. The two material items (stale cycle-1 Summary paragraph; test-file line-citation sanity check) are addressed: the cycle-1 Summary is replaced with a supersede-note that preserves audit trail, and the three cited line numbers in `tests/test_claude_team.py` were verified by grep (all three exist with the cited assertion shapes; `:931` uses `{entity}` instead of `{expected_entity}` as the format variable, noted in this report). The two polish items (decouple AC-3 from a draft sentence; split the long Failsafe paragraph) were also applied in place. No design substance changed; cycle-2 design and acceptance criteria stand.
