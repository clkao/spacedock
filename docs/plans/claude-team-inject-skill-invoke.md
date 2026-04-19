---
id: 204
title: "Inject Skill(skill='spacedock:ensign') first-action directive in claude-team build prompt output"
status: validation
source: "2026-04-19 session — discovered during #203 implementation stage. Ensign committed timeout/budget knob-turns with no fo-log evidence of budget exhaustion; jsonl census of current session's subagents showed zero Skill tool invocations across 133 tool calls (ideation ensign 44, implementation ensign 74, staff-reviewer-203 15). Smoke test agent `ensign-skill-smoketest` dispatched with explicit user-message `Skill(skill=\"spacedock:ensign\")` directive → Skill call fires as `caller: direct`, content loads, ## Operating contract + ## Runtime adapter headings enter context, @references/ files load transitively."
started: 2026-04-19T04:55:06Z
completed:
verdict:
score: 0.95
worktree: .worktrees/spacedock-ensign-claude-team-inject-skill-invoke
issue:
pr:
mod-block: merge:pr-merge
---

Ensign subagents dispatched via `Agent(subagent_type="spacedock:ensign", team_name=...)` never load their operating contract. The agent-definition skill preload in `agents/ensign.md` frontmatter does not fire under Claude Code team mode, and the fallback prose ("If your operating contract was not already loaded via skill preloading, invoke the `spacedock:ensign` skill now to load it") is not being executed either. Every dispatched ensign runs without the shared-core discipline in context: no BashOutput polling discipline (#183), no stage-report format spec, no scaffolding-change guards, no completion-signal protocol, no evidence-before-knob-turn convention.

## Evidence

| Subagent (session `spacedock-plans-20260419-0345-a7x2k9m4`) | Type | First-turn tool uses | Skill invocations |
|---|---|---:|---:|
| `abce3a80…` | `spacedock-ensign-opus-4-7-green-main-ideation` | 44 | **0** |
| `a76dfaf9…` | `spacedock-ensign-opus-4-7-green-main-implementation` | 74 | **0** |
| `a4491003…` | `staff-reviewer-203` (general-purpose) | 15 | **0** |

Smoke test with explicit Skill directive (`ensign-skill-smoketest` subagent, file `agent-acb2c679…`, 11 jsonl lines):

```
1. Skill(skill="spacedock:ensign")   ← caller: direct, succeeded
2. ToolSearch                         ← looking up SendMessage
3. SendMessage(to="team-lead")        ← confirmation
```

The smoke test proves: ensign subagents CAN invoke Skill, DO invoke it when told explicitly via user-message prompt, and the skill content (including transitive `@references/ensign-shared-core.md` load) enters context successfully. This mirrors the working standing-teammate pattern (comm-officer's spawn prompt already does explicit skill-availability checks via ToolSearch).

## Scope of the fix

Modify `skills/commission/bin/claude-team` (the `build` subcommand) to prepend a first-action directive to the emitted prompt body. Estimated 5-10 line change. The directive should:

- Be the FIRST content the ensign sees in its user-message prompt
- Instruct an explicit `Skill(skill="spacedock:ensign")` call
- Note that if preload ever starts working, the call is idempotent
- NOT depend on the broken agent-definition preload path

Similar discipline applies to the standing-teammate build path if any standing teammate currently lacks explicit skill-invoke prose. Audit during ideation.

## Why this likely blocks (or collapses) #203

The three failing tests under investigation in #203 (`test_feedback_keepalive`, `test_merge_hook_guardrail`, `test_standing_teammate_spawn`) watch an FO subprocess's tool-use stream for specific ensign behaviors — completion signals, stage-report writes, data-flow edits. Without the operating contract loaded, ensigns:

1. Improvise completion signals (sometimes SendMessage, sometimes plain text, sometimes JSON) → FO step-timeout fires
2. Write stage reports in arbitrary format → test watcher regex misses
3. Burn wallclock on blocking `sleep` instead of `BashOutput` polling → 120s/300s walls exceeded

The **CI-vs-local divergence** observed in #203's ideation (local 3/3 PASS vs CI 3/3 FAIL on the same HEAD) fits this hypothesis: fast local hardware absorbs the no-contract floundering under the step-timeouts; slow CI runners don't. After #204 ships, re-running the three failing tests on CI with zero test-code changes should clarify how much of #203 is really #204 in disguise.

Acceptance criteria and a test plan will be defined during ideation per the workflow README.

## Ideation

### 1. Alignment audit: spawner prompt vs shared-core

The `claude-team build` subcommand assembles the ensign dispatch prompt in `skills/commission/bin/claude-team` at lines 269-378 (the 10 numbered `prompt_parts` sections). The audit below walks each prose fragment the spawner emits and maps it against `skills/ensign/references/ensign-shared-core.md` (shared-core) and `skills/ensign/SKILL.md` (entry point). Alignment verdict codes: **aligned** (same content, same intent), **divergent** (both cover same topic but with conflicting detail), **duplicative** (spawner repeats shared-core verbatim or near-verbatim), **missing-in-spawner** (shared-core has it, spawner doesn't), **missing-in-core** (spawner has it, shared-core doesn't), **per-dispatch** (legitimately belongs in spawner — cannot live in shared-core because it varies per dispatch).

| # | Spawner fragment (from `claude-team build` output) | Matching shared-core / SKILL.md content | Verdict |
|---|---|---|---|
| P1 | Header: `You are working on: {title}\n\nStage: {stage}` (claude-team:273) | None — shared-core references "the entity" / "the stage" abstractly in `## Assignment` | **per-dispatch** (title and stage name are per-dispatch values) |
| P2 | Stage definition block: `### Stage definition:\n\n{stage_subsection}` extracted from workflow README (claude-team:276) | Shared-core `## Assignment` bullet: "the stage definition" (listed as expected input) | **per-dispatch** (stage definition varies per workflow and per stage) |
| P3 | Worktree instructions: "Your working directory is {path}. All file reads and writes MUST use paths under {path}. Your git branch is {branch}. All commits MUST be on this branch. Do NOT switch branches or commit to main." (claude-team:279-286) | Shared-core `## Working` step 2: "If you were given a worktree path, keep all reads, writes, and commits under that worktree." + `code-project-guardrails.md` `## Git and Worktrees` and `## Paths and File Scope` | **divergent** — spawner says "Do NOT switch branches or commit to main" (imperative, absolute), shared-core says "keep all reads, writes, and commits under that worktree" (scoping-based). Spawner is narrower and stricter; shared-core's guidance that body-edits "belong to the worker" (guardrails) implicitly permits worker commits. Both are compatible but the spawner adds the "no branch switching" rule not present in shared-core. |
| P4 | Entity read instruction: "Read the entity file at {path} for the current spec" / "…for the full spec. It contains:" (claude-team:288-299) | Shared-core `## Working` step 1: "Read the entity file before making changes." | **aligned** (spawner specifies the path; shared-core states the rule) — per-dispatch path + shared rule |
| P5 | Do-not-modify block: "Do NOT modify YAML frontmatter in entity files.\nDo NOT modify files under agents/ or references/ — these are plugin scaffolding." (claude-team:302-305) | Shared-core `## Rules` first two bullets: identical language including the em-dash phrase "— these are plugin scaffolding" | **duplicative** — spawner repeats shared-core verbatim. If shared-core is loaded, this block is pure duplication. |
| P6 | Feedback context block: `### Feedback from prior review\n\n{feedback_context}` (claude-team:309) | None in shared-core. Runtime adapter `claude-ensign-runtime.md` `## Feedback Interaction` covers behavioral expectations (re-check, update stage report, resend completion) but not the block itself | **per-dispatch** (feedback text is per-dispatch) — but note the shared-core lacks any guidance on how to consume a feedback block |
| P7 | Scope notes block (captain charter / staff-review note / etc.): raw passthrough (claude-team:313) | None | **per-dispatch** (varies per dispatch by design — FO-authored per-task guidance) |
| P8a | Completion-checklist header: "Write a ## Stage Report section into the entity file when done." (claude-team:318-320) | Shared-core `## Stage Report Protocol`: "Append a `## Stage Report: {stage_name}` section at the end of the entity file…" | **divergent** — spawner says `## Stage Report` (no stage_name); shared-core says `## Stage Report: {stage_name}`. Shared-core is the more specific/correct form. |
| P8b | Mark directive: "Mark each: DONE, SKIPPED (with rationale), or FAILED (with details)." | Shared-core `## Stage Report Protocol` bullet: "`DONE:` means complete / `SKIPPED:` means intentionally skipped with rationale / `FAILED:` means attempted and failed with concrete details" | **aligned** (same three states, same semantics) |
| P8c | Summary slot: `### Summary\n{brief description of what was accomplished}` | Shared-core `## Stage Report Protocol` template includes `### Summary\n{2-3 sentences…}` | **aligned** (compatible; shared-core is more specific) |
| P8d | Tail imperative: "Every checklist item must appear in your report. Do not omit items." | Shared-core `## Stage Report Protocol` bullet: "every checklist item must appear" | **duplicative** (same rule, different phrasing) |
| P8e | Per-item checklist text (derived from FO-supplied `checklist` list) | None | **per-dispatch** (the checklist items are per-dispatch) |
| P9 | Standing teammates block: names, descriptions, routing usage per mod (claude-team:330-359) | None — shared-core doesn't describe standing teammates at all; final officer's `first-officer-shared-core.md` `## Standing Teammates` does, and the spawner points there in its last line | **per-dispatch** — but the SendMessage routing mechanics could be shared-core material if ensigns are ever expected to route to teammates outside this block |
| P10a | Completion-signal instruction: `SendMessage(to="team-lead", message="Done: {title} completed {stage}. Report written to {entity_file_path}.")` (claude-team:365-369) | Claude-ensign-runtime `## Completion Signal` has identical template; shared-core `## Completion` says "send a minimal completion signal that points the first officer back to the entity file, then stop" (abstract form) | **duplicative with runtime adapter**, **aligned with shared-core abstract rule** — the spawner inlines the exact runtime-adapter template. If the ensign loads the skill, it transitively loads the runtime adapter via the `## Runtime adapter` conditional-read. |
| P10b | Signal format: "Plain text only. No JSON. Until you send this message, the first officer keeps waiting…" | Claude-ensign-runtime `## Completion Signal`: "Plain text only. Never send JSON." | **duplicative** (same rule, spawner adds the "FO keeps waiting" explanation) |
| P10c | FO-to-Agent forwarding footnote: "**If you are the first officer forwarding this prompt to Agent():** copy the entire block above into `Agent(prompt=...)` character-for-character. Do NOT paraphrase…" (claude-team:373-378) | None | **missing-in-core** — this is a meta-instruction to the forwarding orchestrator, not to the ensign. It is arguably mis-placed (it is rendered inside the ensign's prompt but its audience is the FO that is about to forward it). Keep it in the spawner. |
| — | Shared-core `## Background Bash Discipline` (BashOutput polling vs blocking sleep; #183) | NOT emitted by spawner at all | **missing-in-spawner** — this is exactly the discipline #204 is trying to reach ensigns with. Under the broken preload, ensigns never see it. |
| — | Shared-core `## Worktree Ownership` (main vs worktree state, `pr:` mirror exception) | NOT emitted by spawner; partially covered by P3's worktree imperatives | **missing-in-spawner** (partial) — spawner enforces "stay in the worktree" but does not mention `pr:` field exception or main-vs-worktree state rules |
| — | Shared-core `## Working` step 4 ("Update the entity file body, not the frontmatter") | Covered by spawner P5 | **duplicative** (P5 is the stronger form) |
| — | Shared-core `## Working` step 5 ("Commit your work before signaling completion") | NOT emitted by spawner | **missing-in-spawner** — not enforced in the spawner prompt at all. The spawner's P10 completion signal presumes commits are done but does not state the rule. |
| — | Shared-core `## Rules` third bullet ("If requirements are unclear or ambiguous, escalate to the first officer rather than guessing") | NOT emitted by spawner (the runtime adapter's `## Clarification` section covers it) | **missing-in-spawner** — covered by the runtime adapter if loaded. If preload is broken, ensigns never see it. |
| — | Shared-core `## Stage Report Protocol` size guideline (30-50 lines max), no-checkbox-markers rule, `(cycle N)` redispatch rule | NOT emitted by spawner | **missing-in-spawner** — ensigns without the skill loaded can and do write arbitrarily long stage reports without the `(cycle N)` convention. This drift is visible in recent entity files. |
| — | `code-project-guardrails.md` (all sections: git worktrees, paths/file scope, scaffolding protection, commits/evidence) | Partially emitted: P3 covers worktree/branch, P5 covers scaffolding protection | **missing-in-spawner** (partial) — commits/evidence discipline and commit-before-signaling not emitted |

**Summary of audit:**
- **5 topics are duplicative** (P5, P8a-partial, P8b, P8d, P10a, P10b) — the spawner has evolved to inline shared-core content as a band-aid for the broken preload.
- **1 topic is divergent** (P8a: `## Stage Report` vs `## Stage Report: {stage_name}`) — the spawner's form is less specific and would cause shared-core-loaded ensigns to produce a different heading than spawner-only ensigns produce. This is a real drift.
- **5 topics are missing-in-spawner** (Background Bash Discipline / #183, Worktree Ownership / `pr:` exception, commit-before-signaling, escalation rule, stage-report size and `(cycle N)` discipline) — these are the gaps that explain the #203 CI failures (timeout-budget floundering, arbitrary stage-report format, improvised completion signals).
- **Per-dispatch content** (P1, P2, P6, P7, P8e, P9 — stage definition, entity path, feedback, scope notes, checklist items, standing teammates): legitimately belongs only in the spawner.

### 2. Design choice

**Recommended: option (d) hybrid — inject Skill-invoke first-action directive AND keep a trimmed spawner prose that covers only per-dispatch content.**

The spawner keeps P1, P2, P3 (path-specific worktree assignment), P4, P6, P7, P8c, P8e, P9, P10a (this dispatch's completion-signal literal), P10c (FO-forwarding note). The spawner **removes** P5 (duplicative rules), P8a-tail (duplicative stage-report imperatives — shared-core's protocol is the authoritative spec), P8b (DONE/SKIPPED/FAILED semantics), P8d (checklist-completeness rule), P10b-tail (plain-text-no-JSON phrasing). The spawner **prepends** a first-action directive: `Skill(skill="spacedock:ensign")` as the first `prompt_parts` entry before the header, with a short preamble so the LLM treats it as a user instruction rather than embedded prose.

**Concrete directive text to prepend** (single prompt_parts insert at index 0):

```
## First action

Before anything else, invoke your operating contract:

    Skill(skill="spacedock:ensign")

This loads the shared ensign discipline (stage-report format, BashOutput polling, worktree ownership, completion signal protocol). The call is safe to call more than once; if the agent-definition preload ever starts working, calling it again is a no-op (the skill content is re-loaded but has no behavioral effect). Do not paraphrase; call the tool.
```

**Why (d) over (a):** Option (a) would delete the P8a/b/d checklist framing entirely. But some of that framing (e.g., the per-dispatch checklist items in P8e and the `### Summary` slot) is legitimately per-dispatch and belongs in the spawner. Option (d) lets the spawner keep the scaffolding it needs while delegating the rules to shared-core.

**Why (d) over (b):** Option (b) keeps the full duplicated prose as belt-and-suspenders. That perpetuates the drift already visible in the audit (P8a `## Stage Report` vs shared-core `## Stage Report: {stage_name}`). Once there are two sources of truth, edits to one will desync the other. A consistency-check CI job is possible but expensive relative to just removing the duplication.

**Why not (c):** The audit shows the spawner prose is materially incomplete (5 missing-in-spawner topics, including the BashOutput discipline directly implicated in #203 and #183). Keeping the current spawner and not injecting Skill would leave the root cause in place.

**Tradeoff named:** option (d) means ensigns that invoke `Skill(skill="spacedock:ensign")` pay a small context-window cost (the shared-core + guardrails + runtime-adapter contents enter context). Measured sizes: shared-core 79 lines, guardrails 33 lines, claude-ensign-runtime 32 lines — roughly 150 lines of prose, far below the #177 hallucination threshold. This cost is offset by removing ~15 lines of duplicated prose from the spawner.

**On preload-resumption safety:** if the agent-definition preload path is ever fixed upstream, both paths would load shared-core. That is safe to re-invoke; the content may appear twice in-context but has no behavioral effect (the shared-core rules are declarative, not imperative actions to run). Mild redundancy, not a bug — and not worth gating on.

**Out of scope for this task (explicitly):** the standing-teammate spawn path (`cmd_spawn_standing`, claude-team:717-817). That path emits a mod's `## Agent Prompt` section verbatim as the spawn prompt — it does not go through `cmd_build`. A separate audit of standing-teammate mods would need to check each mod for analogous preload/prose drift. Tracked as **#205** (Audit standing-teammate spawn prompts for Skill-invoke / contract drift). It is not a blocker for #203 or for closing #204.

### 3. Acceptance criteria and test plan

AC1. Every ensign dispatch emitted by `claude-team build` (team mode and bare mode) contains the literal string `Skill(skill="spacedock:ensign")` in the assembled `prompt` field, positioned before the `You are working on:` header.
  - **Verified by:** a unit test in `tests/test_claude_team.py` that invokes `cmd_build` with a minimal valid stdin payload and asserts the substring appears at `prompt.index("You are working on:") > prompt.index('Skill(skill="spacedock:ensign")')`. Run in both bare_mode and team-mode variants.

AC2. Following a dispatch through the fixed `claude-team build`, the resulting ensign subagent's jsonl contains a `Skill` tool call with input `{"skill": "spacedock:ensign"}` before any non-Skill tool use. (Claude Code may emit internal/system tool uses — e.g. cached agent-definition reads — ahead of the user-prompt-directed action; the AC is only that the first tool use targeting a non-Skill tool is preceded by a matching Skill call.)
  - **Verified by:** a post-implementation smoke test — dispatch a throwaway ensign for any ideation stage (e.g. a no-op task seed), locate the jsonl via `claude-team context-budget --name {derived_name}` path logic or direct glob, filter for `type == "assistant"` entries with `message.content[*].type == "tool_use"`, walk the sequence of tool_use blocks in order, and assert that the first non-`Skill` block is preceded somewhere earlier by a `Skill` block with `input.skill == "spacedock:ensign"`. The smoke test must record the exact jsonl path it inspected (absolute path under `~/.claude/projects/.../subagents/agent-*.jsonl`) in the implementation stage report so a reviewer can independently re-run the check.

AC3. The `skills/commission/bin/claude-team` file no longer emits prose that duplicates content already present in `skills/ensign/references/ensign-shared-core.md`. Specifically, the following substrings are absent from the spawner's `prompt_parts` assembly (after the fix):
  - `Do NOT modify YAML frontmatter in entity files.`
  - `Do NOT modify files under agents/ or references/`
  - `Every checklist item must appear in your report. Do not omit items.`
  - `Mark each: DONE, SKIPPED (with rationale), or FAILED (with details).`
  - **Verified by:** a grep target in the same unit test — assert each string is NOT substring-present in the assembled `prompt` field.
  - **Test-remediation clause:** removing these substrings from the spawner breaks the live e2e assertion at `tests/test_checklist_e2e.py:119` (`re.search(r"DONE.*SKIPPED.*FAILED|Mark each.*DONE", agent_prompt, ...)`). Implementation must update that assertion to grep `skills/ensign/references/ensign-shared-core.md` for the equivalent pattern (`re.search(r"DONE:.*SKIPPED:.*FAILED:", shared_core_text, re.DOTALL)`) instead of the live agent prompt. Rationale: the pattern now lives authoritatively in shared-core, not in the per-dispatch prompt. This is the single cleanest remediation — option (a) from the reviewer's two candidates — because inspecting the Skill tool-result in the jsonl log (option b) would require parser changes to the test's `LogParser.agent_prompt()` helper and is not worth the complexity for one regex match. (Secondary note: `test_checklist_e2e.py` is already `@pytest.mark.xfail(strict=False, reason="pending #198 ...")` at line 26-27, so a transient mis-match during implementation would not actually fail CI; still fix the assertion in the same PR to avoid leaving a broken test.)

AC4. Every fragment removed from the spawner in AC3 is covered by existing content in `skills/ensign/references/ensign-shared-core.md`. This is a mapping invariant, not a dynamic property.
  - **Verified by:** a static cross-reference table in this stage report (below), mapping each removed substring → shared-core heading + line range. Reviewer checks the table by opening shared-core and confirming each cited range exists.

AC5. The one divergent naming (P8a: spawner `## Stage Report` vs shared-core `## Stage Report: {stage_name}`) is resolved in favor of shared-core. After the fix, the spawner does not emit the string `## Stage Report` at all (the rule lives in shared-core only).
  - **Verified by:** grep target in the unit test — assert `"## Stage Report"` is NOT a substring of the assembled `prompt` field.

AC6. The Skill-invoke directive text explains in plain language that re-invoking the skill is safe. The directive's preamble must include language along the lines of "safe to call more than once; calling it again is a no-op (the skill content is re-loaded but has no behavioral effect)." Plain language — the word "idempotent" is jargon and should not appear in the directive text.
  - **Verified by:** static inspection — the prepended directive contains the substring `safe to call more than once` (or equivalent plain-language phrasing) in the explanation text. Asserted as a substring check in the unit test. The test also asserts the word `idempotent` does NOT appear in the directive.

**Mapping table for AC4 (removed-substring → shared-core coverage):**

| Removed spawner substring | Shared-core section | Shared-core content |
|---|---|---|
| `Do NOT modify YAML frontmatter in entity files.` | `## Rules` bullet 1 | Identical wording |
| `Do NOT modify files under agents/ or references/ — these are plugin scaffolding.` | `## Rules` bullet 2 | Identical wording |
| `Every checklist item must appear in your report. Do not omit items.` | `## Stage Report Protocol` rules list | "every checklist item must appear" |
| `Mark each: DONE, SKIPPED (with rationale), or FAILED (with details).` | `## Stage Report Protocol` rules list | "`DONE:` means complete / `SKIPPED:` means intentionally skipped with rationale / `FAILED:` means attempted and failed with concrete details" |

**Test plan:**

- **Static (unit) tests** — new `tests/test_claude_team.py::test_build_emits_skill_invoke_directive` and sibling cases. Cost: ~10 minutes to write. Run locally + CI. Verifies AC1, AC3, AC5, AC6. Substring assertions: `Skill(skill="spacedock:ensign")` present and positioned before `You are working on:`; each AC3 substring absent from assembled `prompt`; `## Stage Report` substring absent; directive contains `safe to call more than once` and does NOT contain `idempotent`.
- **Existing-test remediation (R1)** — update `tests/test_checklist_e2e.py:119` in the same implementation PR. The current assertion `re.search(r"DONE.*SKIPPED.*FAILED|Mark each.*DONE", agent_prompt, re.IGNORECASE)` becomes an assertion against shared-core file content: read `skills/ensign/references/ensign-shared-core.md` and assert `re.search(r"DONE:.*SKIPPED:.*FAILED:", shared_core_text, re.DOTALL)` matches. Rationale: once the spawner no longer inlines the phrase, the pattern lives only in shared-core; the test's intent (verify the DONE/SKIPPED/FAILED discipline is documented somewhere the ensign will see) is preserved.
- **Sweep of `tests/` for adjacent breakage** — ran greps during ideation for the four to-be-removed substrings and for `## Stage Report`. Results, with dispositions:

  | File:line | Match | Disposition |
  |---|---|---|
  | `tests/test_checklist_e2e.py:119` | `DONE.*SKIPPED.*FAILED\|Mark each.*DONE` on `agent_prompt` | **MUST UPDATE** (R1 remediation above) |
  | `tests/test_checklist_e2e.py:131` | `DONE\|SKIPPED\|FAILED` on `fo_text` (not agent_prompt) | **safe, no change** — matches FO's own review text, not the spawner prompt |
  | `tests/test_agent_content.py:113` | `assert "## Stage Report: {stage_name}" in text` | **safe, no change** — asserts shared-core file content, not the spawner prompt |
  | `tests/test_agent_content.py:116` | `assert "Do NOT modify YAML frontmatter" in text` | **safe, no change** — asserts shared-core file content (confirms staff-review's note) |
  | `tests/test_agent_content.py:248,459,467` | `section_text(text, "## Stage Report Protocol", ...)` | **safe, no change** — shared-core section extraction, not spawner content |
  | `tests/fixtures/*/*-entity.md` (4 files) | `## Stage Report: ...` in fixture entity files | **safe, no change** — fixture content, not assertions |
  | `tests/test_claude_team.py` ~8 existing assertions on spawner prompt content (lines 656-886, 1628-1708 per staff review) | none match the four AC3 substrings | **safe, no change** — confirmed none of them assert on the four to-be-removed substrings |

- **Sweep of `scripts/` and `tests/` for `tool_uses[0]` / first-tool-use patterns (A5 risk check)** — greps produced:

  | File:line | Match | Disposition |
  |---|---|---|
  | `scripts/test_lib.py:991-1001` | `_tool_use_block(entry: dict)` returns first `tool_use` block **within a single assistant entry**, not the first tool use in the whole jsonl | **safe, no change** — callers pass a specific assistant entry they've already selected, so the Skill prepend (which is in its OWN assistant entry) does not affect their result |
  | Callers of `tool_use_matches` across 9 test files (`test_rebase_branch_before_push.py`, `test_merge_hook_guardrail.py`, `test_feedback_keepalive.py`, `test_standing_teammate_spawn.py`, `test_dispatch_names.py`, `test_fo_stream_watcher.py`, `test_claude_per_stage_model.py`, `test_team_dispatch_sequencing.py`) | match specific tool names on specific entries | **safe, no change** — they don't assume "first tool use in jsonl is X", only "this particular entry's tool_use block matches X" |
  | `scripts/` — no matches for `tool_uses\[0\]` / `first_tool_use` beyond the helper above | — | **safe** |

  Net: the prepend does not silently break any existing tool-use-inspection logic. If implementation discovers any new-to-the-repo consumer of "first jsonl tool_use" semantics, it should document the assumption change inline.

- **Smoke test (one-shot)** — after implementation commits land, run `claude-team build` via the FO for one throwaway ensign dispatch, inspect the resulting jsonl, and confirm AC2. Cost: ~5 minutes of dispatch + grep. The smoke test records the exact jsonl path in the implementation stage report (per AC2's `Verified by` clause). No need for a permanent E2E test — once AC1/AC3/AC5 pass statically, AC2 can only fail if Claude Code's Skill tool itself breaks, which is outside this task's scope.
- **No additional E2E suite change needed beyond R1.** #203's three failing tests (`test_feedback_keepalive`, `test_merge_hook_guardrail`, `test_standing_teammate_spawn`) become the downstream verification: after #204 ships, re-run those tests on CI and observe whether the hypothesized improvement materializes. That is #203's task, not #204's.
- **AC4 is a static mapping** verified once by reviewer inspection. No automated check needed (the mapping table is small and stable; if shared-core changes, a future task that changes shared-core would update the mapping). See also Risk A6 below for a lightweight future defense.

**Pre-implementation sweep (A4):** before dispatching the implementation ensign, the FO must grep open PRs for touches to `cmd_build` / `prompt_parts` that would conflict with AC3's removals — especially anything touching the recently-merged PR #128's territory. Specific command: `gh pr list --state open --search "claude-team" --json number,title,files` and inspect each hit's diff for `prompt_parts` or `claude-team build` changes. If conflicts exist, reconcile (rebase / resequence / coordinate with author) before implementation dispatch.

**Estimated implementation cost:** 20-30 lines of diff in `skills/commission/bin/claude-team` + 1 line change in `tests/test_checklist_e2e.py:119` (R1) + ~60 lines of new unit-test code in `tests/test_claude_team.py`. Under an hour of ensign time at the implementation stage. No scaffolding refactor. No new behavioral E2E tests.

### 4. Risks

**A3 — #204 implementer-ensign bootstrap.** When #204 enters implementation, the implementing ensign itself dispatches through the broken `cmd_build` path and will lack the operating contract — the exact condition this task is trying to eliminate. The FO will manually prepend the Skill-invoke directive to #204's implementation dispatch (same workaround used for this ideation dispatch). Once the fix lands, the workaround is no longer needed. Alternative considered: captain-direct edit (since the change is ~20 lines). Either path works; FO's call at dispatch time.

**A4 — In-flight PR conflicts.** Any open PR that touches the `prompt_parts` section of `cmd_build` (especially recently-merged PR #128) will conflict with AC3's removals. Before implementation dispatches, the FO should grep open PRs: `gh pr list --state open --search "claude-team" --json number,title` plus a diff check on any hits. See the pre-implementation sweep clause in §3.

**A5 — jsonl debuggability.** Tools and scripts that inspect specific tool-use positions (e.g., completion-signal watchers, step-timeout detectors, the `context-budget` extractor) may need updating since every ensign's first tool_use becomes `Skill` after this fix. The ideation sweep (§3 above) verified no existing consumer of that assumption is affected: `scripts/test_lib.py::_tool_use_block` extracts the first tool_use **within a single assistant entry**, not the first tool_use in the jsonl stream, so all 104 call sites remain correct. Implementation must re-run this grep (`rg 'tool_uses\[0\]|first tool_use|first_tool_use'` across `scripts/` and `tests/`) just before landing and document any new-to-the-repo hits. This is informational — no AC required — but skipping the grep risks a silent regression.

**A6 — Shared-core drift regression.** Once shared-core is the sole source of truth for the trimmed fragments, edits to shared-core reach every ensign instantly with no intermediate catch. Mitigation suggestion (do not require in this task's scope): add a pre-merge grep target for PRs that touch `skills/ensign/references/ensign-shared-core.md` that checks the four canonical phrases (the AC3 removed-substring list) are still present. Filing this as a separate follow-up when shared-core authorship is next under review is preferable to bolting it onto #204.

## Stage Report: ideation

- DONE: Audit alignment — produced exhaustive 20-row audit table covering all 10 numbered prompt_parts sections in `claude-team build` (lines 269-378), mapping each spawner fragment to shared-core and SKILL.md content with verdict codes (aligned / divergent / duplicative / missing-in-spawner / missing-in-core / per-dispatch).
  Audit table is the "Ideation §1" subsection above.
- DONE: Design choice — recommended option (d) hybrid (inject Skill-invoke + trim spawner to per-dispatch content only) over (a), (b), (c). Tradeoffs for each alternative named with concrete citations back to the audit table. Concrete directive text drafted (6-line block).
  Design rationale is the "Ideation §2" subsection above.
- DONE: Acceptance criteria + test plan — produced 6 ACs (AC1-AC6), each with a `Verified by` clause. AC1/AC3/AC5/AC6 are unit-testable substring invariants on `cmd_build` output; AC2 is a one-shot smoke test against a live dispatch's jsonl; AC4 is a static mapping invariant with the mapping table inline. Test plan estimates ~10min unit-test authoring + ~5min smoke test, no E2E changes, #203 re-run is the downstream verification not this task's scope.
  AC list and test plan are the "Ideation §3" subsection above.

### Summary

Audited `claude-team build` prompt assembly against `ensign-shared-core.md` and found 5 duplicative fragments (the spawner has inlined shared-core content as a band-aid for the broken preload), 1 divergent naming (`## Stage Report` vs `## Stage Report: {stage_name}`), and 5 shared-core topics missing from the spawner entirely (including the BashOutput polling discipline that #203 and #183 are symptomatic of). Recommended hybrid fix: prepend a `Skill(skill="spacedock:ensign")` first-action directive AND trim the four duplicated fragments from the spawner so shared-core becomes the single source of truth for cross-dispatch discipline, while the spawner retains only per-dispatch content (header, stage definition, entity path, feedback, scope notes, checklist items, standing teammates, completion signal literal, FO-forwarding note). Flagged the standing-teammate spawn path (`cmd_spawn_standing`) as out of scope — a follow-up task should audit standing-teammate mod prompts for analogous drift.

## Staff Review

**Reviewer:** independent staff-review pass for #204 ideation gate
**Verdict:** CONCUR WITH REVISIONS

### A. Audit table accuracy

Spot-checked three rows against the source:
- **P5 (duplicative):** spawner `claude-team:303-304` emits `'Do NOT modify YAML frontmatter in entity files.\nDo NOT modify files under agents/ or references/ — these are plugin scaffolding.\n'`; shared-core lines 30-31 contain the same two sentences verbatim (including the em-dash and "— these are plugin scaffolding" phrase). Verdict correct.
- **P8a (divergent):** spawner `claude-team:319` emits `'Write a ## Stage Report section into the entity file when done.'`; shared-core line 48 says `'Append a `## Stage Report: {stage_name}` section at the end of the entity file…'`. The divergence is real and material (heading form differs, and "append" vs "write" implies append-vs-overwrite semantics the audit didn't call out). Verdict correct; if anything, under-stated — this is not just naming drift but also a semantic drift (append-only vs write-anywhere).
- **P10a (duplicative with runtime adapter):** spawner `claude-team:364-369` emits the `SendMessage(to="team-lead", message="Done: ...")` template; `claude-ensign-runtime.md` lines 19-25 contain the same template in a fenced block. Verdict correct.

Line-range claim "269-378" matches the source exactly (assembly starts at the `prompt_parts = []` at line 270 and the final `prompt_parts.append(...)` closes at line 378). No disagreements.

### B. Completeness

Two gaps in the audit:
1. **Captain Communication (claude-ensign-runtime.md §2, lines 13-15)** — runtime-adapter content covering Shift+Up/Down captain visibility and direct-text-vs-SendMessage rules. The audit's "missing-in-spawner" list doesn't flag this, though it IS loaded transitively via Skill; I'd count it as "covered by Skill, so not spawner's job" rather than a miss, but it's worth a one-line note in the audit since it governs the same captain-facing behavior the spawner is silent on.
2. **Agent Surface (claude-ensign-runtime.md §1, lines 5-7)** — the "dispatch prompt is authoritative for all assignment fields" sentence is not mirrored in the spawner, and that's deliberate (Skill-loaded), but the audit doesn't enumerate it. Minor.

Otherwise the 20 rows cover all 10 `prompt_parts` appends in `cmd_build`. No spawner fragment is missed.

### C. Design choice soundness

**(i) Context-window cost:** Actual transitive load measured: shared-core 78 lines, code-project-guardrails 32 lines, claude-ensign-runtime 31 lines, SKILL.md 17 lines → **~158 lines**. Ideation's "~150 lines" estimate is accurate. Well below any hallucination threshold. Concur.

**(ii) Idempotency claim:** The "survives preload resuming" claim is weaker than stated. Claude Code's Skill tool call is idempotent at the tool-result level (re-invocation re-emits the content), but the ensign's attention budget isn't — a double-load shows the operating contract twice in-context, which is wasteful but not incorrect. No risk of behavior divergence; just mild redundancy. Fine as stated, but the ideation should soften "is idempotent" to "is safe to re-invoke; content may appear twice in-context but has no behavioral effect."

**(iii) Standing-teammate exclusion:** Partially justified. `cmd_spawn_standing` emits a mod's `## Agent Prompt` section verbatim (claude-team:814), bypassing `cmd_build` entirely. The one live standing teammate (comm-officer.md:77) already does an explicit ToolSearch for `elements-of-style:writing-clearly-and-concisely` in its spawn prompt — so the comm-officer case is not analogous to the ensign-preload bug (comm-officer isn't trying to load its OWN operating contract, just checking tool availability for a specialty skill). **However**, comm-officer does NOT call `Skill(skill="spacedock:ensign")` or any equivalent first-officer/ensign contract loader; it relies on its mod's `## Agent Prompt` section to be self-contained. If future standing mods are added that need shared-core discipline, the same preload bug would bite them. Exclusion is defensible for #204's scope but should be filed as an explicit follow-up task (not just a note) before closing #204.

### D. AC rigor

- **AC1** — CONCUR. The substring + index assertion is robust; `prompt_parts` can change structure without breaking the test as long as the assembled `prompt` string contains both substrings in order.
- **AC2** — FLAG. "First tool_use entry" is fragile. Claude Code may emit internal/system tool uses (e.g., cached read of agent definition) before the user-prompt-directed action. Recommend: weaken to "within the first 3 tool_use entries" or "before any non-Skill tool use". Also AC2 depends on jsonl path discovery (`claude-team context-budget --name` logic) — flaky if the derived_name collides. Make the smoke test document the exact jsonl path it inspected.
- **AC3** — CONCUR on substring absence; the substrings listed are stable spawner literals. **But see section E below:** AC3 must be paired with updates to `tests/test_checklist_e2e.py:119` which currently asserts `DONE.*SKIPPED.*FAILED|Mark each.*DONE` MUST be substring-present in the live agent prompt — that test will break when AC3 removes "Mark each: DONE, SKIPPED (with rationale), or FAILED (with details)." The ideation misses this.
- **AC4** — CONCUR. Static mapping table is clear and the reviewer can verify in <5 minutes.
- **AC5** — CONCUR, but same test-plan gap: removing `## Stage Report` from the spawner means the existing e2e checklist test and any other test that greps for it need updating.
- **AC6** — FLAG. The word "idempotent" is jargon. Recommend the directive use plain language ("safe to call more than once; calling it again is a no-op") and assert THAT phrasing, not "idempotent". Otherwise the directive reads as self-referentially technical.

### E. Test-plan gap check

**Silent gap found.** Two existing tests assert on to-be-removed spawner content:
- `tests/test_checklist_e2e.py:119` — `re.search(r"DONE.*SKIPPED.*FAILED|Mark each.*DONE", agent_prompt, re.IGNORECASE)` will fail when AC3 removes both patterns from the spawner. This is a live e2e test, not a unit test. Plan must update this assertion to grep shared-core instead of the agent_prompt, OR must acknowledge that shared-core's `DONE:`/`SKIPPED:`/`FAILED:` bullets (which load via Skill) will satisfy the regex once the Skill tool-result is part of the prompt log — but that depends on the test's log parser.
- `tests/test_agent_content.py:116` — this one asserts the string exists in `ensign-shared-core.md`, NOT in the spawner prompt, so it's safe. No change needed, but the ideation doesn't explicitly call out that it checked.

Also: `test_claude_team.py` has ~8 existing assertions on spawner prompt content (lines 656-886, 1628-1708). None of them assert on the four to-be-removed substrings, so they're safe, but the ideation should state this explicitly to close the loop.

### F. Out-of-scope items

Standing-teammate exclusion is justified for #204's scope (the comm-officer spawn prompt at `docs/plans/_mods/comm-officer.md:77` already does explicit skill-availability checks and does not share the ensign-preload bug). But "a separate audit of standing-teammate mods" should be filed as a concrete follow-up issue number before #204 closes — leaving it as a prose aside risks the drift going unaudited. One sentence in the ideation committing to file a follow-up would close this.

### G. Risks not named

1. **The #204 implementer-ensign itself needs the fix.** If #204 is dispatched through the same broken `claude-team build` pipeline before the fix lands, the implementing ensign will lack the operating contract that specifies (e.g.) stage-report format and commit-before-signal — exactly the disciplines the fix is trying to codify. Recommend the FO either (a) manually prepend the Skill-invoke directive to #204's implementation dispatch, or (b) implement #204 without dispatching an ensign (captain-direct edit, given it's ~20 lines of code).

2. **In-flight PRs touching `claude-team` prose.** Any open PR that touches the `prompt_parts` section of `cmd_build` (especially `#128`, the most recent merge) will conflict with AC3's removals. The ideation doesn't list what's in flight. Need to grep open PRs for `cmd_build` / `prompt_parts` touches before implementation starts.

3. **Debuggability of ensign jsonl.** Prepending `Skill(skill="spacedock:ensign")` as the first action means every ensign jsonl now begins with a Skill tool-use entry followed by its result. Tools and scripts that scan for "first meaningful ensign action" (e.g., completion-signal watchers, step-timeout detectors, the `context-budget` extractor) may now misclassify the Skill call as the user-visible first action. Worth a one-line check that no existing tool skips the first tool_use entry or inspects `tool_uses[0]` for specific names.

4. **Shared-core drift regression risk.** Once shared-core is the sole source of truth for the four trimmed fragments, future edits to shared-core that change those bullets' phrasing will immediately reach every ensign — there is no intermediate spawner copy to catch a bad edit at CI time. The trade is correct (single source of truth beats two), but a shared-core change checklist (e.g., a pre-merge grep for the four canonical phrases) is cheap to add.

### Bottom line

The audit is accurate, the line references check out, and the hybrid design (option d) is the right call. Before this passes the ideation gate, three concrete revisions are required: **(R1)** add an explicit test-plan step for updating `tests/test_checklist_e2e.py:119` (and a sweep for any other test that greps agent_prompt for the removed substrings) — this is a concrete silent-breakage, not a theoretical one; **(R2)** soften AC2's "first tool_use entry" to "before any non-Skill tool use" and require the smoke test to record the exact jsonl path inspected; **(R3)** commit to filing a follow-up task number for the standing-teammate prose audit rather than leaving it as a prose note. The remaining flags (AC6 jargon, idempotency phrasing, risks 1-4) are advisory — worth addressing before implementation dispatch but not ideation-gate blockers.

## Stage Report: ideation (revision pass)

### Revision summary (R1/R2/R3 + A1-A6)

- **R1 (required)** — DONE. AC3 now carries a `Test-remediation clause` specifying the exact fix for `tests/test_checklist_e2e.py:119` (option (a): re-point the assertion to `skills/ensign/references/ensign-shared-core.md` file content with `re.search(r"DONE:.*SKIPPED:.*FAILED:", ..., re.DOTALL)`). Rationale picks option (a) over option (b) because (b) requires parser-level changes to `LogParser.agent_prompt()`. Test plan §3 now includes a sweep table listing every `tests/` grep hit for the four AC3 substrings + `## Stage Report`, each dispositioned (`MUST UPDATE` for the one real breakage, `safe, no change` for every other hit with reason). Files/lines explicitly checked: `tests/test_checklist_e2e.py:119`, `tests/test_checklist_e2e.py:131`, `tests/test_agent_content.py:113,116,248,459,467`, `tests/fixtures/*` entity files, and the `test_claude_team.py` lines 656-886 & 1628-1708 range that staff review flagged.
- **R2 (required)** — DONE. AC2 rewritten from "first tool_use entry" to "before any non-Skill tool use" with explicit acknowledgement of internal/system tool-use entries preceding the user-prompt-directed action. `Verified by` clause now mandates recording the exact jsonl path (absolute, under `~/.claude/projects/.../subagents/agent-*.jsonl`) in the implementation stage report.
- **R3 (required)** — DONE. Out-of-scope paragraph now cites **#205** by number, replacing the prose "Filing that as a follow-up is appropriate" with "Tracked as #205 (Audit standing-teammate spawn prompts for Skill-invoke / contract drift)." No change to the scope decision itself.
- **A1 (advisory)** — DONE. The word "idempotent" is removed from both the prepended directive text (§2) and AC6. Directive now reads "safe to call more than once; if the agent-definition preload ever starts working, calling it again is a no-op (the skill content is re-loaded but has no behavioral effect)." AC6 substring assertion now checks for `safe to call more than once` AND asserts `idempotent` does NOT appear.
- **A2 (advisory)** — DONE. §2 design rationale carries a new paragraph `**On preload-resumption safety:**` that explicitly phrases the claim as "safe to re-invoke; content may appear twice in-context but has no behavioral effect" — exactly the softening the staff review requested.
- **A3 (advisory)** — DONE. New `### 4. Risks` subsection under the test plan names the #204 implementer-ensign bootstrap risk and proposes the FO workaround (manually prepend Skill-invoke directive, same as this ideation dispatch) plus the alternative (captain-direct edit).
- **A4 (advisory)** — DONE. Named in the new Risks subsection AND a **Pre-implementation sweep** paragraph added at the end of §3 specifying the exact `gh pr list` command the FO must run before dispatching implementation.
- **A5 (advisory)** — DONE. New Risks subsection names the jsonl-debuggability risk. §3 test plan now includes a sweep table of `tool_uses[0]` / `first tool_use` / `first_tool_use` matches across `scripts/` and `tests/`, showing the one concrete hit (`scripts/test_lib.py::_tool_use_block`) is **safe by construction** because it operates per-assistant-entry, not per-jsonl-stream. All 104 call sites of `tool_use_matches` across 9 test files verified safe. Implementation must re-run the grep before landing.
- **A6 (advisory)** — DONE. Named in the new Risks subsection as an informational item. Explicitly not required in this task's scope; recommended mitigation (pre-merge grep target for shared-core PRs) flagged as a separate future follow-up rather than bolted onto #204.

### Per-item status

- DONE: R1 — AC3 test-remediation clause + tests/ sweep table with dispositions for every hit of the four removed substrings and `## Stage Report`. Files/lines explicitly listed.
  See AC3 (new "Test-remediation clause" paragraph) and §3 test plan sweep table.
- DONE: R2 — AC2 rewritten to use "before any non-Skill tool use"; Verified-by clause mandates recording the exact jsonl path.
  See AC2.
- DONE: R3 — Out-of-scope paragraph cites #205 by number.
  See §2 "Out of scope" paragraph.
- DONE: A1 — Directive text and AC6 purged of "idempotent" jargon; substring assertion updated.
  See §2 directive block and AC6.
- DONE: A2 — §2 design rationale carries "safe to re-invoke; content may appear twice in-context but has no behavioral effect" paragraph.
  See §2 "On preload-resumption safety" paragraph.
- DONE: A3 — New §4 Risks subsection names implementer-ensign bootstrap risk with workaround + alternative.
  See §4 Risks item A3.
- DONE: A4 — Risks subsection names in-flight PR conflict risk; §3 adds a Pre-implementation sweep paragraph with the exact `gh pr list` command.
  See §4 Risks item A4 and §3 Pre-implementation sweep paragraph.
- DONE: A5 — Risks subsection names jsonl-debuggability risk; §3 adds a sweep table showing `scripts/test_lib.py::_tool_use_block` is safe by construction (per-entry, not per-stream) and all 104 `tool_use_matches` call sites across 9 test files are safe.
  See §4 Risks item A5 and §3 "Sweep of `scripts/` and `tests/`" table.
- DONE: A6 — Risks subsection names shared-core drift regression risk as informational; mitigation suggestion flagged as separate future follow-up.
  See §4 Risks item A6.

### Summary

Revision pass addressed all three required items (R1 test-remediation clause for `test_checklist_e2e.py:119` via re-pointing to shared-core, R2 AC2 softening to "before any non-Skill tool use" + mandatory jsonl path recording, R3 #205 citation) and all six advisory items (A1 plain-language "safe to call more than once" replacing "idempotent" in both directive and AC6; A2 softened preload-resumption phrasing in §2; A3-A6 named in a new §4 Risks subsection with concrete mitigations where applicable and explicit deferrals where not). Added a comprehensive tests/ sweep table to §3 dispositioning every grep hit of the four AC3 substrings plus `## Stage Report` and `tool_uses[0]` / `first tool_use` patterns across `scripts/` and `tests/` — net finding is one real breakage (already handled by R1) and zero hidden consumers of "first tool use in jsonl" semantics. Test plan now cites a Pre-implementation sweep via `gh pr list --state open --search "claude-team"` to catch PR #128 or later conflicts with AC3's removals.

## Stage Report: implementation

- DONE: Checklist item 1 — Pre-implementation PR sweep. Ran `gh pr list --state open --search "claude-team"` (2 hits: #106, #119) and `gh pr list --state open --search "prompt_parts"` (0 hits). Inspected each diff: **#106** touches only `skills/first-officer/references/*` (runtime-adapter split — no `cmd_build` touch). **#119** inserts a new `1.5. Skill loading instruction (conditional)` block AFTER `1. Header` in `cmd_build` for per-stage plugin skill loading (`stage_meta.get('skill')`), semantically orthogonal to this task's operating-contract directive and at a different insertion point (my change prepends at index 0, #119 inserts at index 1.5). No conflict. Sweep date 2026-04-18; 2 hits; 0 conflicts.
- DONE: Checklist item 2 — Implementation. `skills/commission/bin/claude-team` `cmd_build` now prepends the Skill-invoke directive (plain-language wording per ideation §2, no `idempotent` jargon) at `prompt_parts[0]`. Removed P5 duplicative block ("Do NOT modify YAML frontmatter…" / "Do NOT modify files under agents/…") and the duplicative P8a/P8b/P8d checklist-framing strings (`## Stage Report`, `Mark each: DONE, SKIPPED…`, `Every checklist item must appear…`). Updated `tests/test_checklist_e2e.py:119` per R1 option (a): assertion now greps `skills/ensign/references/ensign-shared-core.md` content with `re.search(r"DONE:.*SKIPPED:.*FAILED:", ..., re.DOTALL)`. Added 5 unit tests in `tests/test_claude_team.py::TestBuildSkillInvokeDirective` covering AC1 (both team and bare mode), AC3, AC5, AC6. Commit: `804a6bbd`. `make test-static`: **454 passed, 22 deselected, 10 subtests passed in 20.98s**. No `tool_uses[0]` / `first tool_use` code paths were touched in the implementation (A5 sweep not re-run this stage per captain charter point 6).
- DONE: Checklist item 3 — End-to-end smoke test.
  - **AC1/AC3/AC5/AC6 unit verification:** 5/5 tests in `TestBuildSkillInvokeDirective` PASS (see commit `804a6bbd`).
  - **Direct cmd_build render check:** ran `claude-team build` via test helpers against a synthetic fixture; prompt length 1746 chars; first 500 chars is the `## First action` block followed by `Skill(skill="spacedock:ensign")` then the plain-language explanation; `You are working on:` appears at char ~500; all 9 substring checks (directive present, Skill < header, `safe to call more than once` present, `idempotent` absent, four AC3 substrings absent, `## Stage Report` absent) PASS.
  - **AC2 live-dispatch evidence (bootstrap case):** This implementation ensign itself is the live smoke test — the FO manually prepended the same Skill-invoke directive to this dispatch (per the teammate-message at dispatch time, since the fix was not yet live). Jsonl path: `/Users/clkao/.claude/projects/-Users-clkao-git-spacedock/9d5ad752-f519-4097-b260-345dc47c149a/subagents/agent-a0687300d24393a2b.jsonl` (103 lines). First 10 tool_use names in order: (1) `Skill(skill="spacedock:ensign")` — caller: direct, succeeded; (2) Read; (3) Read; (4) Read; (5) Read; (6) Read; (7) Bash; (8) ToolSearch; (9) Bash; (10) Bash. Confirms the Skill call appears before any non-Skill tool_use. AC2 verified.
  - **AC4 static mapping:** confirmed in ideation §3 mapping table; shared-core lines 30-31 cover the two YAML/agents bullets verbatim; shared-core `## Stage Report Protocol` lines 67-71 cover the DONE/SKIPPED/FAILED/`every checklist item must appear` semantics. Re-verified by reading shared-core at implementation start.

### Summary

Prepended `Skill(skill="spacedock:ensign")` as a first-action directive in `claude-team build` prompt assembly so every dispatched ensign loads its operating contract regardless of whether the agent-definition preload fires under Claude Code team mode. Removed 5 duplicative shared-core fragments from the spawner (two do-not-modify bullets, the Mark-each DONE/SKIPPED/FAILED line, the `Every checklist item must appear` tail, the `## Stage Report` heading directive) so shared-core becomes the single source of truth. Re-pointed the one breaking e2e assertion (`tests/test_checklist_e2e.py:119`) to grep shared-core content instead of the live agent prompt. Full static suite passes 454/454; new `TestBuildSkillInvokeDirective` class adds 5 passing unit tests. Bootstrap self-evidence from this dispatch's own jsonl confirms AC2 behavior (first tool_use is the Skill call; subsequent tool_uses begin after the skill content loads).

## Stage Report: validation

**Framing (captain directive §3):** The three live tests (`test_feedback_keepalive`, `test_merge_hook_guardrail`, `test_standing_teammate_spawns_and_roundtrips`) are the exact tests #203 is failing on CI. Their local pass/fail under the fix is the empirical test of whether #204 alone collapses #203.

### Checklist item 1 — Independent AC verification: DONE

Read `git log main..HEAD` (2 commits: `804a6bbd` impl + `2ba43258` stage-report doc) and full diff (16/10 lines in `skills/commission/bin/claude-team`, 4/2 in `tests/test_checklist_e2e.py`, 97/0 in `tests/test_claude_team.py`). Cross-checked each AC against the actual diff.

| AC | Verdict | Evidence |
|---|---|---|
| AC1 (Skill directive before header in both team/bare modes) | PASS | Diff `skills/commission/bin/claude-team:272-286` prepends `prompt_parts[0]` with the directive before `## 1. Header`. Unit tests `test_build_prepends_skill_invoke_directive` and `test_build_prepends_skill_invoke_directive_bare_mode` PASS. Smoke dispatch recorded `skill_idx=76` vs `header_idx=460`. |
| AC2 (ensign's first non-Skill tool_use preceded by Skill call; jsonl path recorded) | PASS | Implementation stage report records jsonl path `~/.claude/projects/-Users-clkao-git-spacedock/9d5ad752-.../subagents/agent-a0687300d24393a2b.jsonl`. This validator's own jsonl `agent-a30e9c8b826b32c68.jsonl` independently confirms: first 4 tool_uses in order = `Skill(skill="spacedock:ensign")`, Read(claude-ensign-runtime.md), Read(entity spec), ToolSearch. |
| AC3 (four duplicative substrings absent) | PASS | Diff deletes P5 do-not-modify block (`claude-team:302-305`) and P8a/P8b/P8d checklist framing (`claude-team:318-332`). Unit test `test_build_omits_duplicative_shared_core_prose` PASS. Smoke dispatch confirms all four substrings absent. |
| AC4 (removed substrings covered by shared-core) | PASS | Static mapping table in ideation §3 shared-core `## Rules` bullets 1-2 cover YAML/agents bullets; `## Stage Report Protocol` covers DONE/SKIPPED/FAILED semantics and `every checklist item must appear`. Verified by re-reading `skills/ensign/references/ensign-shared-core.md`. |
| AC5 (`## Stage Report` heading absent from spawner prompt) | PASS | Diff removes `Write a ## Stage Report section` line from P8a. Unit test `test_build_omits_stage_report_heading` PASS. Smoke dispatch confirms string absent. |
| AC6 (plain-language `safe to call more than once` present; `idempotent` absent) | PASS | Diff `claude-team:277-285` directive text contains `safe to call more than once`; no `idempotent` in directive. Unit test `test_build_directive_uses_plain_language_safety_phrasing` PASS. |

**Static suite:** `make test-static` → **454 passed, 22 deselected, 10 subtests passed in 20.87s**. Matches implementation-stage count (454/454). Exit 0.

**Targeted unit tests:** `pytest tests/test_claude_team.py::TestBuildSkillInvokeDirective -v` → **5 passed in 0.15s**. All five AC1/AC3/AC5/AC6 unit assertions green.

### Checklist item 2 — Targeted live tests LOCALLY: PARTIAL

All three tests at `--effort low`, `--model opus`, runtime `claude`, N=1, `KEEP_TEST_DIR=1`. `--plugin-dir $(pwd)` from the captain directive is not a recognized pytest option (confirmed; pytest errored immediately); ran without it — this matches the default path in the test's own invocation under `run_first_officer` helper which already points `--plugin-dir` internally.

| Test | Verdict | Wallclock | Key signal |
|---|---|---|---|
| `test_feedback_keepalive` | **PASS** | 186.41s | Clean single-pass. Exit code 0. |
| `test_merge_hook_guardrail` | **FAIL** | 391.41s (over 5-min wallclock budget) | Step `[OK] ensign Agent() dispatched` and `[OK] merge hook fired` both PASSED. Failure is `StepTimeout` on the subsequent `expect_exit(timeout_s=300)` — FO process did not exit within 300s after the merge hook fired. fo-log at `/tmp/204-val-evidence/test_merge_hook_guardrail-fo-log.jsonl` (212KB). Not an operating-contract/Skill-invoke failure: the ensign dispatch prompt contained `## First action ... Skill(skill="spacedock:ensign")` (confirmed by grepping fo-log: 4 occurrences of `First action` across 4 ensign dispatches). |
| `test_standing_teammate_spawns_and_roundtrips` | **FAIL** | 118.14s | All four watcher steps PASSED: `[OK] claude-team spawn-standing invoked`, `[OK] echo-agent Agent() dispatched`, `[OK] ensign dispatch prompt includes standing-teammates section with echo-agent`, `[OK] SendMessage to echo-agent observed`. Failure is `StepFailure` on next step `archived entity body captured 'ECHO: ping'` — FO subprocess exited (code=0) before the echo roundtrip reached the entity body. Grep of fo-log at `/tmp/204-val-evidence/test_standing_teammate_spawns_and_roundtrips-fo-log.jsonl` (206KB): zero matches for `ECHO: ping` — the echo-agent standing teammate never produced a visible reply that the ensign captured into the entity. Not an operating-contract/Skill-invoke failure: the ensign dispatch prompt contained `## First action ... Skill(skill="spacedock:ensign")` (4 occurrences of `First action` in fo-log). |

**Evidence preservation:** `/tmp/204-val-evidence/` contains 4 artifacts totaling ~460KB:
- `test_merge_hook_guardrail-pytest.log` (14KB)
- `test_merge_hook_guardrail-fo-log.jsonl` (212KB)
- `test_standing_teammate_spawns_and_roundtrips-pytest.log` (16KB)
- `test_standing_teammate_spawns_and_roundtrips-fo-log.jsonl` (206KB)

`test_feedback_keepalive` preserved at `/var/folders/.../tmpuchj4rxv` by pytest (PASS — no copy needed).

**Empirical answer to the `#203 collapses into #204` hypothesis:** NOT supported by this data. 1/3 PASS, 2/3 FAIL at opus-low post-fix. The two failures are real post-fix failures and are **not** caused by ensigns lacking the operating contract — fo-log grep confirms the Skill-invoke directive is successfully reaching every ensign dispatch through the fixed `claude-team build`. The failure modes (merge-hook FO post-exit timeout; echo-agent data-roundtrip absence) are behavioral/timing issues in the FO's exit path and the standing-teammate echo-and-capture loop respectively, orthogonal to contract loading. #204's fix is necessary but not sufficient to turn #203's three reds green; at least two additional defects exist downstream.

**Budget report:** test_feedback_keepalive 186s + test_standing_teammate 118s + test_merge_hook 391s ≈ 11.6min wallclock. merge_hook exceeded the per-test 5-min soft cap in the captain directive; per directive I did not thrash — reported partial results and stopped (no re-runs).

### Checklist item 3 — End-to-end smoke-dispatch through the WORKTREE's fixed cmd_build: DONE

Ran a direct `claude-team build` invocation (via `sys.executable skills/commission/bin/claude-team build --workflow-dir ...`) with a synthetic stdin payload (minimal workflow README + entity). Full emitted prompt captured; assertions:

| Check | Result |
|---|---|
| (a) `Skill(skill="spacedock:ensign")` substring present | PASS |
| (b) Appears BEFORE `You are working on:` (skill idx=76, header idx=460) | PASS |
| (c) Four AC3 removed substrings ABSENT | PASS (all four) |
| (d) `## Stage Report` ABSENT | PASS |
| (e) `idempotent` ABSENT | PASS |
| (bonus) `safe to call more than once` PRESENT | PASS |

Relevant prompt excerpt (first ~500 chars, before the per-dispatch header):

```
## First action

Before anything else, invoke your operating contract:

    Skill(skill="spacedock:ensign")

This loads the shared ensign discipline (stage-report format, BashOutput polling, worktree ownership, completion signal protocol). The call is safe to call more than once; if the agent-definition preload ever starts working, calling it again is a no-op (the skill content is re-loaded but has no behavioral effect). Do not paraphrase; call the tool.

You are working on: Smoke task
...
```

### Recommendation: PASSED

All six acceptance criteria verified with direct evidence from diff, unit tests, smoke dispatch, and fo-log grep. #204's scope is "inject the Skill-invoke directive and remove duplicative prose" — that scope is fully and correctly delivered. The implementation is complete and correct per its own acceptance criteria.

The two live-test failures surfaced during validation (`test_merge_hook_guardrail` post-exit timeout, `test_standing_teammate_spawn` echo-roundtrip absence) are OUT OF SCOPE for #204. Evidence: fo-log greps show the Skill-invoke directive is present in every ensign dispatch through the fixed spawner; the failures are in downstream behaviors (FO exit path; echo-agent roundtrip) that #204 was never scoped to fix. These should be filed as separate defects and handled by #203 (or successor tasks), consistent with the ideation risk A3/A6 framing that #204 is necessary but not claimed to be sufficient for the three-test suite.

Per captain directive constraint (1), validator is NOT authorized to fix these defects. Routing guidance: the two downstream failures warrant new issues (or reassignment into #203's scope) rather than `feedback-to: implementation` for #204, because #204's deliverable meets its own spec. If the FO chooses to route back regardless (e.g., widening #204's scope), that is the FO's call; from the validator's seat the correct recommendation on #204 as currently scoped is **PASSED**.

### Per-item status

- DONE: Checklist item 1 — Independent AC verification. All six ACs cross-checked against `git diff main..HEAD` with line citations; static suite 454/454 PASS; TestBuildSkillInvokeDirective 5/5 PASS. Per-AC verification table above.
- DONE: Checklist item 2 — Targeted live tests LOCALLY (partial, per 5-min-wallclock cap). 1/3 PASS (`test_feedback_keepalive`, 186s), 2/3 FAIL (`test_merge_hook_guardrail` StepTimeout on expect_exit 300s after `[OK] merge hook fired` at 391s; `test_standing_teammate_spawns_and_roundtrips` StepFailure on `archived entity body captured 'ECHO: ping'` at 118s). fo-logs preserved under `/tmp/204-val-evidence/`. fo-log grep confirms Skill-invoke directive reached all ensign dispatches — failures are not contract-loading failures.
- DONE: Checklist item 3 — End-to-end smoke-dispatch via worktree's fixed `cmd_build`. All five AC assertions (directive present + before header, AC3 absent, Stage Report absent, idempotent absent) and bonus plain-language check PASS.

### Summary

PASSED — #204's scope (Skill-invoke directive injected at `prompt_parts[0]`; four duplicative shared-core fragments removed; `## Stage Report` heading moved to shared-core as sole owner; plain-language safety phrasing) is fully delivered. All 6 ACs verified with direct evidence. Static suite 454/454 green; 5/5 new unit tests green; smoke dispatch confirms the emitted prompt contains the directive before the header and omits every AC3 substring. Live-test matrix (1/3 PASS at opus-low) does NOT support the `#203 collapses into #204` hypothesis: two downstream defects persist after the fix (merge-hook FO post-exit timeout; echo-agent roundtrip absence in standing-teammate test). Both failures are orthogonal to contract loading — fo-log greps confirm the Skill-invoke directive reaches every ensign dispatch under the fixed spawner. Recommendation: merge #204 as PASSED; file the two downstream failures as separate defects (or hand them back to #203's cycle-2 matrix-fill experiment, which is already running in parallel).
