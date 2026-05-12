---
id: 0f1smz7rrgm3rdrw5ssqtrhr
title: FO gate presentation buries the lede and flattens reviewer priority
status: validation
source: captain (CL) — self-critique during ideation gate on `bw / name-pattern-rejects-stage-names-with-underscores`
started: 2026-05-12T03:38:42Z
completed:
verdict:
score:
worktree: .worktrees/spacedock-ensign-fo-gate-presentation-buries-lede
mod-block: merge:pr-merge
pr: #208
---

The FO's gate-review message — the one assembled per `skills/first-officer/references/claude-first-officer-runtime.md` `## Gate Presentation` and the shared-core gate flow — is captain-hostile in its current form. A fresh-eye captain reading the gate message I produced for `bw` ideation could not, without scrolling, answer "what direction was chosen" or "what's the one thing I actually need to decide". Concrete defects, in roughly the order they cost the captain attention:

1. **Lede is buried.** The actual decision prompt sits at the bottom of a ~60-line message. Chosen direction, FO verdict, and the single decision question should be the first three lines.

2. **Chosen ideation direction never surfaces in the FO's own prose.** The DONE checklist accounting cites "picks among directions with reasoning" but never names the picked direction. The captain has to infer it from reviewer findings or open the entity file.

3. **Stage Report attribution is ambiguous.** The verbatim ensign report is pasted under an `## Stage Report` header without quoting / fencing / preface — looks like FO-generated content. The runtime adapter's template says "paste verbatim" but doesn't specify rendering. The result blurs whose words those are.

4. **Reviewer findings are flat-bulleted with no priority tier.** A material fact-correction (e.g. "your one-line claim is wrong because file X doesn't currently do Y") sits next to wording polish (e.g. "AC2 substring assertion is over-fitted"). The captain weighs them as equals because the rendering treats them as equals.

5. **Recommendation prose duplicates the decision prompt.** A paragraph that says "I recommend #2" is followed by an enumerated "three paths" list that re-explains #2 with slightly different wording. Pick one form.

6. **Recommended bounce-back doesn't show the fixup.** "Address the reviewer's five concrete notes" forces the captain to scroll up and re-derive them. Show the actual fixup ask in one or two lines, or quote the concrete asks; don't reference "the five notes".

7. **Pedantic format drift is presented as load-bearing.** A paragraph called out "ideation used `1./2./3./4.` instead of `**AC-N**`" — either it blocks the gate or it doesn't. If not blocking, cut it.

8. **No worktree heads-up.** Approval of ideation moves the entity into a worktree-backed stage. One sentence noting "approval will open `.worktrees/...`" gives situational awareness for what `approve` actually does.

9. **Signal-to-length ratio is bad.** ~60 lines for a decision that needs maybe 15. The captain just typed `dispatch` 90 seconds prior; they did not opt into a six-section essay.

## Resolution directions (decide in ideation)

- **Option A: tighten the existing template.** The runtime adapter's current gate template is already short:
  ```
  Gate review: {entity title} — {stage}

  {paste the ## Stage Report section from the entity file verbatim}

  Assessment: {N} done, {N} skipped, {N} failed. [Recommend approve / Recommend reject: {reason}]
  ```
  The defects above are mostly FO-improvisation *beyond* this template. Possible fix: add a "Captain-facing assembly rules" subsection — lede-first chosen-direction line, reviewer-finding priority tiers, no parenthetical pedantry, max-15-line target — and call it the discipline rather than expand the template itself.

- **Option B: expand the template.** Add named slots: `Chosen direction`, `Material findings`, `Polish findings`, `Decision prompt`. Force the assembler to fill them. Risk: rigidity for entities where the structure doesn't fit (e.g. simple non-staff-review gates).

- **Option C: hybrid.** Keep the existing terse template as canonical for simple gates; add an "Expanded gate template (when staff review or material reviewer findings are present)" alternate with explicit slots.

Whichever path, the test for "fixed" is: a captain coming in cold on a gate message can name the chosen direction, the FO's verdict, and the decision question without scrolling.

## Out of scope (call out)

- The Stage Report *format itself* (the `- DONE/SKIPPED/FAILED:` accounting block written by the ensign into the entity body) is not the target here. The ensign-side template is fine. This task is about the FO's *gate-presentation* assembly of that report plus reviewer findings plus FO recommendation.
- Changing the gate decision contract (still captain-only, no FO self-approval, etc.).

## Proposed approach

Take **Option A** (tighten the existing template with an explicit "Captain-facing assembly rules" subsection), with one small structural change to the template itself: hoist a lede line above the pasted Stage Report. Reasoning:

- The current template (`claude-first-officer-runtime.md` `## Gate Presentation`, ~line 188-198) is already terse — only the title line, the verbatim Stage Report paste, and the Assessment line. The nine defects in the seed are FO-improvisation *outside* this template (chosen-direction narration, reviewer-finding rendering, recommendation prose, format-pedantry asides, signal-to-length). So the fix is to constrain what the FO is allowed to add around the template, not to rigidify slots inside it.
- **Option B** (named slots: `Chosen direction`, `Material findings`, `Polish findings`, `Decision prompt`) creates rigidity for simple gate stages where most slots are empty or contrived — validation gates with no reviewer subagent, work gates with no chosen direction, terminal gates with no decision question. The slot scaffolding would itself become noise on simple cases.
- **Option C** (hybrid: terse canonical + alt expanded template) doubles the surface area for what is fundamentally a discipline problem. Branching templates would themselves become FO improvisation about which template applies when.

One change to the canonical template *is* needed: defect #1 (lede buried) and defect #2 (chosen direction never surfaces in FO prose) cannot be fixed by discipline rules alone if the template itself routes the captain past the Stage Report paste before reaching the lede. So the template gains a one-line lede slot above the pasted Stage Report, and the assembly-rules subsection governs everything else (priority tiers, recommendation form, length budget, worktree heads-up, no pedantry).

### Edit point

`skills/first-officer/references/claude-first-officer-runtime.md` `## Gate Presentation` (currently ~line 188-198). Replace the entire section.

**Before** (current text, verbatim):

````
## Gate Presentation

Present gate reviews in this format:

```
Gate review: {entity title} — {stage}

{paste the ## Stage Report section from the entity file verbatim}

Assessment: {N} done, {N} skipped, {N} failed. [Recommend approve / Recommend reject: {reason}]
```
````

**After** (proposed text):

````
## Gate Presentation

Present gate reviews in this format:

```
Gate review: {entity title} — {stage}
Chosen direction: {one-line summary of the ensign's chosen approach, or `n/a` for stages without a chosen-direction concept (e.g., simple work stages, merge)}
Recommend {approve | reject: {one-line reason}}.

{paste the ## Stage Report section from the entity file verbatim, fenced in a ```markdown code block so authorship is unambiguous}

{If reviewer findings exist, render them under a `Reviewer findings` heading in two tiers — `Material:` (fact-corrections, contract violations, missing AC evidence, broken claims) and `Polish:` (wording, format drift, non-blocking suggestions). Drop the tier entirely if it has no items. If no reviewer ran, omit this whole block.}

Assessment: {N} done, {N} skipped, {N} failed.

Decision: {one-line decision prompt naming what approval/rejection does in concrete terms — e.g., "approve to enter implementation in worktree `.worktrees/...`" or "reject to bounce back to {feedback-to target} with the material findings above"}.
```

### Captain-facing assembly rules

The template above is the floor, not the ceiling — but the FO MUST hold to the following discipline when filling it:

1. **Lede first, decision last, nothing between them buried.** The first three lines (title, chosen direction, recommend) and the final line (decision prompt) are the message's spine. Everything else is supporting evidence that the captain may scroll for. If the captain stops reading after the first three lines, they can still vote.
2. **Chosen direction is required as FO prose.** When the stage involved selecting among options (ideation picks an approach, validation picks PASS/REJECTED, etc.), the FO names the chosen direction in its own one-line summary on the `Chosen direction:` line. Do not make the captain infer it from the Stage Report paste or the entity file. For stages without a chosen-direction concept (e.g., simple work stages), use `n/a`.
3. **Stage Report goes in a fenced `markdown` block.** The verbatim ensign-authored Stage Report is rendered inside a fenced code block (` ```markdown ` open / ` ``` ` close) so authorship is visually unambiguous — these are the ensign's words, not the FO's.
4. **Reviewer findings render in priority tiers.** When a staff-reviewer subagent ran, group its findings into `Material:` (fact-corrections, contract violations, missing AC evidence, claims contradicted by the codebase) and `Polish:` (wording, format drift, non-blocking suggestions). Drop the tier entirely if it has no items. Do not flat-bullet material findings next to polish findings.
5. **Recommendation appears exactly once.** The `Recommend {approve | reject: {reason}}` line is the only place the FO states its verdict. Do not duplicate it in a separate "I recommend #2" paragraph and then re-explain it in an enumerated list. Pick the one-line form.
6. **Bounce-back recommendations quote the concrete asks.** If recommending reject, the reason line names the specific concerns by content, not by reference. Bad: "address the reviewer's five concrete notes." Good: "tighten AC-2 substring assertion; correct the file X claim; cut the format-pedantry aside."
7. **No format-pedantry asides.** Format drift (`1./2./3./4.` instead of `**AC-N**`, missing trailing period, etc.) is not load-bearing for a gate decision. If it doesn't block the gate, do not surface it. If it does, it is a Material finding under reviewer findings — not a separate paragraph.
8. **One sentence of worktree heads-up when approval changes worktree state.** If approving this gate will open or close a worktree (entering a `worktree: true` stage, or merging out of one), the Decision line names it: "approve to enter implementation in worktree `.worktrees/{worker_key}-{slug}`". One sentence, not a section.
9. **Target length: 15-25 lines of FO-authored prose, plus the pasted Stage Report.** The Stage Report is whatever length the ensign wrote (typically 30-50 lines per the ensign protocol). The FO's surrounding prose — title, lede, recommendation, reviewer findings, assessment, decision — should fit in 15-25 lines. If it doesn't, the FO is over-narrating; cut.
````

(End of replacement.)

The `## Completion and Gates` section in `first-officer-shared-core.md` does not need a parallel edit — it already directs the FO to "present the stage report to the human operator" and delegates the rendering format to the runtime adapter. The discipline rules above are runtime-adapter-scoped because the message-assembly surface is Claude Code-specific (the captain reads them as direct text output, not as a structured artifact).

## Acceptance criteria

- **AC-1:** The `## Gate Presentation` section in `skills/first-officer/references/claude-first-officer-runtime.md` contains a `### Captain-facing assembly rules` subsection with at least nine numbered rules covering: lede-first ordering, chosen-direction as FO prose, Stage Report fencing, reviewer-finding priority tiers (`Material:` / `Polish:`), single-recommendation form, concrete bounce-back asks, no format-pedantry, worktree heads-up on stage transition, and a 15-25-line FO-prose length budget. Verified by `grep -c "^[0-9]\." skills/first-officer/references/claude-first-officer-runtime.md` returning at least 9 inside the new subsection (or equivalent static grep for each rule's anchor phrase).
- **AC-2:** The `## Gate Presentation` template in `skills/first-officer/references/claude-first-officer-runtime.md` contains a `Chosen direction:` line above the pasted Stage Report and a `Decision:` line below the assessment. Verified by `grep -n "Chosen direction:" skills/first-officer/references/claude-first-officer-runtime.md` and `grep -n "^Decision:" skills/first-officer/references/claude-first-officer-runtime.md` each returning at least one hit within the `## Gate Presentation` section.
- **AC-3:** The `## Gate Presentation` template in `skills/first-officer/references/claude-first-officer-runtime.md` instructs that the Stage Report is cited by file path and line range (no verbatim paste, no inline fenced ` ```markdown ` block). Verified by: `grep -n "Checklist:" skills/first-officer/references/claude-first-officer-runtime.md` returns at least one hit inside the `## Gate Presentation` section, AND `grep -nE "from .*Stage Report.*in.*lines" skills/first-officer/references/claude-first-officer-runtime.md` returns at least one hit inside the same section, AND `grep -c '```markdown' skills/first-officer/references/claude-first-officer-runtime.md` returns zero (no surviving fenced markdown directive in the file post-edit).
- **AC-4:** The `## Gate Presentation` template names `Reviewer findings` with the two-tier substructure `Material:` and `Polish:` and explicitly states the tier is omitted when no reviewer ran. Verified by grep for `Reviewer findings`, `Material:`, and `Polish:` all appearing in the `## Gate Presentation` section.
- **AC-5:** The `## Worked example` fixture in this entity body's first three non-blank, non-fence-delimiter lines, in order, match `Gate review: {...}`, `Chosen direction: {non-empty string, not the literal `n/a`}`, and `Recommend {approve|reject: ...}`. Verified by static reading of the committed fixture during validation.
- **AC-6:** The shared-core `## Completion and Gates` section in `skills/first-officer/references/first-officer-shared-core.md` is unchanged by this task. Verified by `git diff main -- skills/first-officer/references/first-officer-shared-core.md` showing no changes.

## Test plan

**Proof level: static prose review of the runtime-adapter edit, plus a hand-built transcript fixture for the worked example.** Justified per the workflow README's "choose proof at the same abstraction level" rule (line 78): the claim here is about FO orchestration prose — what message text the FO assembles for a gate review. The right proof level is the prose itself plus a representative rendering of what it produces. Live FO E2E runs are explicitly the wrong level: they are expensive (minutes per dispatch cycle), indirect (the FO's gate message depends on entity content, reviewer content, stage state, all of which would need to be set up), and prone to fixture drift (any unrelated change to FO prose-assembly logic would re-render the gate message). The bug being fixed is a *discipline* bug in the runtime-adapter text, not a runtime-behavior bug — fixing it means changing the text the FO reads, and proving the fix means reading that text and confirming the new text says what it should say.

Concrete validation steps:

1. **Static grep checks for AC-1 through AC-4** (durable doc structure): run the grep commands named in each AC against the post-edit `claude-first-officer-runtime.md` and confirm hits. Cost: seconds. Sufficient because the claims are about which strings appear in a markdown reference file — exactly the case the README cites for "static checks for durable doc/contract structure."
2. **Static reading of the worked-example fixture for AC-5** (representative rendering): read the committed fixture and confirm its first three non-blank, non-fence-delimiter lines match, in order, `Gate review: {...}`, `Chosen direction: {non-empty string, not the literal `n/a`}`, and `Recommend {approve|reject: ...}`. Cost: under a minute. Sufficient because the claim is about the lede's structural shape, and the fixture *is* the message the new template would produce.
3. **Git diff check for AC-6** (out-of-scope guard): `git diff main -- skills/first-officer/references/first-officer-shared-core.md` returns empty. Cost: one command. Sufficient because the claim is "this file did not change."

No live FO E2E is required. No transcript replay is required (the worked example is hand-built from the seed's actual `bw` scenario, which is in the seed body — that's the same evidentiary basis a transcript fixture would provide, at a fraction of the cost). The validator's job is to read the post-edit runtime-adapter section, confirm the rules say what they should say, and confirm the worked example renders the way the rules require.

**Risk and mitigation.** The main risk of static-prose proof is that the rules read well but fail to constrain FO behavior in practice — i.e., the discipline section is ignored when the FO actually assembles a gate message. Mitigation: the worked example (AC-5) is the discipline applied to a concrete, captain-witnessed scenario (the `bw` gate that motivated this task). If the rules can't be cleanly applied to that scenario in the fixture, the rules are wrong and ideation must re-cycle. This is cheaper than waiting for the next live gate to test the discipline.

**Estimated complexity.** Low. One file edit (the runtime adapter `## Gate Presentation` section), one fixture (the worked example in the entity body), four grep checks, one diff check. Total implementation surface: under 100 lines of net new text in two files.

## Worked example

Reconstructing the seed's motivating scenario (the `bw / name-pattern-rejects-stage-names-with-underscores` ideation gate) against the proposed new template. This is what the FO should have produced; it is the fixture for AC-5.

````
Gate review: name-pattern rejects stage names with underscores — ideation
Chosen direction: Option 2 — pre-validate stage names against Claude Code's NAME_PATTERN at workflow-load time and reject early with a captain-readable error, rather than letting the rejection surface from `Agent()` mid-dispatch.
Recommend reject: the AC-2 substring assertion is over-fitted to one error string, and the one-line claim that `commission.py` already validates stage names is contradicted by `grep NAME_PATTERN skills/commission/` returning zero hits.

Checklist (from ## Stage Report in {entity_file_path} lines {start}-{end}):
- DONE: AC end-state phrasing + verified-by citations
- DONE: Option 2 selected with tradeoff reasoning
- DONE: static grep + fixture-reading proof level

Reviewer findings:

Material:
- AC-2 substring assertion `"stage name 'foo_bar' rejected"` is over-fitted — the actual error message format isn't fixed and may legitimately drift; assert on the rejection *behavior* (non-zero exit, NAME_PATTERN named in stderr) instead.
- The seed's claim that `commission.py` already validates stage names against NAME_PATTERN is wrong — `grep NAME_PATTERN skills/commission/` returns zero hits. The validation does not exist yet; this task is adding it, not relocating it.

Polish:
- AC list uses `1./2./3./4.` instead of `**AC-N**`; convention is the latter.

Assessment: 3 done, 0 skipped, 0 failed.

Decision: reject to bounce back to ideation with the two material findings above. Approval would otherwise open implementation in worktree `.worktrees/spacedock-ensign-name-pattern-rejects-stage-names-with-underscores`.
````

Fresh-eye check (the seed's "fixed" criterion): a captain who reads only the first three lines can answer (a) chosen direction = early NAME_PATTERN validation, (b) FO verdict = reject, (c) decision question = bounce back to ideation. No scrolling required.

For comparison, the same scenario rendered against the *current* template would have buried the chosen direction (the ensign's Stage Report names it only inside the DONE bullet body), would have flat-bulleted the AC-2-over-fitting concern next to the `1./2./3./4.` format note, would have duplicated the recommendation in a separate paragraph, and would have referenced "the five concrete notes" rather than quoting the material ones.

## Stage Report: ideation

- DONE: Acceptance criteria are entity-level end-state properties about FO behavior — not imperatives like 'FO should X'. Each AC names a property of the finished change that a future reader can verify (e.g. 'the gate-presentation section in claude-first-officer-runtime.md contains a Captain-facing assembly rules subsection grep-matching ...'), with a concrete 'Verified by' citation. No imperative-verb AC items.
  AC-1 through AC-6 are end-state properties (grep returns N hits, git diff returns empty, fixture's first three lines satisfy criterion); each carries a 'Verified by' clause. No `FO should X` or imperative-verb phrasing.
- DONE: Proposed approach picks among the three seed directions (tighten existing template with assembly-discipline subsection / expand template with named slots / hybrid) — or proposes a different one — with explicit reasoning. The approach names the concrete edit point in skills/first-officer/references/claude-first-officer-runtime.md `## Gate Presentation` (current location ~line 188-199) AND specifies before/after wording for any template changes per the workflow README's template-changes rule ('specific before/after wording, not just "change X"').
  Picked Option A (tighten + assembly-rules subsection) with one structural change (lede line hoisted above Stage Report paste). Reasoning rejects Option B (rigid slots noisy on simple gates) and Option C (branching templates re-introduce improvisation). Edit point named: `claude-first-officer-runtime.md` `## Gate Presentation` ~line 188-198. Before/after wording supplied verbatim in the Proposed approach section.
- DONE: Test plan picks proof at the right abstraction level. This is FO orchestration behavior — proof should be static-prose review of the runtime adapter text and/or transcript fixtures of FO gate messages, NOT live FO E2E runs (which are expensive, indirect, and prone to fixture drift). Plan must explicitly justify the chosen proof level against the workflow README's 'choose proof at the same abstraction level' rule.
  Test plan picks static grep checks for AC-1 through AC-4 (durable doc structure), static fixture reading for AC-5 (worked-example rendering), and git diff for AC-6 (out-of-scope guard). README line 78 cited verbatim; live FO E2E explicitly rejected as wrong level (expensive, indirect, fixture-drift-prone) and the bug framed as a discipline bug in adapter text, not a runtime-behavior bug.

### Summary

Selected Option A from the seed's three directions: tighten the existing terse template with an explicit "Captain-facing assembly rules" subsection, plus one structural change (hoist a `Chosen direction:` lede line above the pasted Stage Report and add a `Decision:` line below the assessment). Wrote six AC items as end-state properties with concrete grep / git-diff / fixture-reading verification clauses, plus a worked-example fixture reconstructing the seed's motivating `bw` scenario against the new template. Test plan is entirely static-prose review at the runtime-adapter level — the bug is in adapter text, so proof is in adapter text; no live FO E2E required.

## Stage Report: ideation (cycle 2)

- DONE: AC-5 is subjective dressed as a check. Rewrite to the objective structural form supplied by the reviewer.
  AC-5 rewritten verbatim per the suggested form: asserts the worked-example fixture's first three non-blank, non-fence-delimiter lines match `Gate review: {...}`, `Chosen direction: {non-empty string, not the literal `n/a`}`, and `Recommend {approve|reject: ...}`. Fresh-eye framing removed. Spot-checked the fixture: first three qualifying lines are `Gate review: name-pattern rejects...`, `Chosen direction: Option 2 — pre-validate...` (non-empty, not `n/a`), `Recommend reject: ...`.
- DONE: Internal inconsistency between rule #2 and the template's `n/a` enumeration. Drop `validation` from the `n/a` list so rule #2 owns the chosen-direction-required claim end-to-end.
  Template comment (in the "After" block) edited: `... or `n/a` for stages without a chosen-direction concept (validation, work, merge)` → `... or `n/a` for stages without a chosen-direction concept (e.g., simple work stages, merge)`. Rule #2 now uncontested as the authority on which stages produce a chosen direction; validation is no longer enumerated as `n/a`-eligible.

### Summary

Two narrow fixups applied from the staff-reviewer's approve-with-notes verdict. AC-5 swapped from fresh-eye framing to the objective structural assertion the reviewer supplied. Template comment in the proposed `## Gate Presentation` "After" block updated to drop `validation` from the `n/a` enumeration, aligning the template with rule #2 of the assembly rules (validation's PASSED-vs-REJECTED counts as a chosen direction). Worked example untouched — it already uses a real chosen direction for ideation, so reconciliation didn't require fixture changes.

## Stage Report: implementation

- DONE: The runtime adapter's `## Gate Presentation` section in skills/first-officer/references/claude-first-officer-runtime.md is REPLACED with the entity body's proposed 'After' block (between the ````` fences at entity lines 89-122). Replacement is verbatim — including the template inside the fence and the nine-rule `### Captain-facing assembly rules` subsection.
  Replaced lines 188-198 of the runtime adapter (the prior 11-line `## Gate Presentation` section, including its terse three-line template) with the entity's "After" block verbatim. The post-edit section spans lines 188-218: title heading, "Present gate reviews in this format:" sentence, the fenced template (now with `Chosen direction:` lede, `Recommend ...` line, fenced `markdown` Stage Report paste directive, conditional `Reviewer findings` block, `Assessment:` line, and `Decision:` line), and the `### Captain-facing assembly rules` subsection with all nine numbered rules in order (lede-first / chosen-direction-required / fenced-markdown / priority-tiers / single-recommendation / quote-concrete-asks / no-format-pedantry / worktree-heads-up / 15-25-line budget). No paraphrasing applied — wording matches the entity body's canonical text.
- DONE: AC-1 through AC-4 grep checks all pass against the post-edit runtime adapter: at least nine numbered rules with anchor phrases, `Chosen direction:` and `Decision:` template lines, ` ```markdown ` fenced Stage Report directive, and `Reviewer findings` + `Material:` + `Polish:` substructure with omission language. Run the grep commands from the AC bodies and confirm hits before signaling complete.
  AC-1: `awk '/^### Captain-facing assembly rules/,/^## /' ... | grep -cE '^[0-9]+\. \*\*'` returned `9`. AC-2: `Chosen direction:` hit on line 194 (template) and line 211 (rule #2); `^Decision:` hit on line 203 (template). AC-3: ` ```markdown ` substring hit on line 197 (template paste directive) and line 212 (rule #3). AC-4: `Reviewer findings` + `Material:` + `Polish:` all hit on line 199 (template) and line 213 (rule #4); omission language present (`Drop the tier entirely if it has no items. If no reviewer ran, omit this whole block.`).
- DONE: AC-6 holds: `git diff main -- skills/first-officer/references/first-officer-shared-core.md` returns empty. The implementation must NOT touch shared-core; only the runtime adapter and the entity file change.
  `git diff main -- skills/first-officer/references/first-officer-shared-core.md` returns empty output. `git diff main --stat` reports one file changed (`claude-first-officer-runtime.md`, +22 / -2). Shared-core untouched.

Additional changes applied per the dispatch's polish-tier note:

- Test plan validation step 2 (entity line 142) was rewritten from the stale "fresh-eye captain can answer chosen direction / verdict / decision question without scrolling" framing to echo AC-5's structural form: "confirm its first three non-blank, non-fence-delimiter lines match, in order, `Gate review: {...}`, `Chosen direction: {non-empty string, not the literal `n/a`}`, and `Recommend {approve|reject: ...}`." The earlier fresh-eye phrasing was leftover from the pre-cycle-2 AC-5; the plan now matches the post-cycle-2 AC.
- Markdown sanity check: triple-backtick fence delimiters in the post-edit runtime adapter count to 4 (two balanced fenced blocks — pre-existing `Agent(...)` JSON block plus the new `## Gate Presentation` template block). No mkdocs setup is available in the worktree, so the sanity check is the fence count plus a visual read of the inserted section.

### Summary

Replaced the `## Gate Presentation` section of `skills/first-officer/references/claude-first-officer-runtime.md` (lines 188-198) verbatim with the entity body's proposed "After" block, expanding it from 11 lines to 31 lines. The new section adds a `Chosen direction:` lede line and a `Decision:` line to the template, directs the verbatim Stage Report paste into a fenced `markdown` code block, introduces a conditional `Reviewer findings` block with `Material:` / `Polish:` tiers, and adds a `### Captain-facing assembly rules` subsection with nine numbered discipline rules. AC-1 through AC-4 grep checks pass; AC-6 diff is empty (shared-core untouched). One supplementary edit applied per dispatch: the test plan's AC-5 validation step now echoes AC-5's structural form rather than the stale fresh-eye phrasing.

## Stage Report: validation

- DONE: Each **AC-N** in the entity body has its 'Verified by' clause reproduced — run each grep against the post-edit skills/first-officer/references/claude-first-officer-runtime.md and report concrete hit counts and line numbers. Run AC-6's `git diff main -- skills/first-officer/references/first-officer-shared-core.md` and report whether output is empty.
  - AC-1: `awk '/^### Captain-facing assembly rules/,/^## /' skills/first-officer/references/claude-first-officer-runtime.md | grep -nE '^[0-9]+\. \*\*'` returned 9 hits (rules 1-9 at relative awk-output lines 5-13, mapping to file lines 210-218). Each numbered rule's anchor phrase matches the required nine topics: lede-first (rule 1), chosen-direction-as-FO-prose (rule 2), Stage-Report-fenced-markdown (rule 3), reviewer-finding-priority-tiers (rule 4), single-recommendation (rule 5), concrete-bounce-back-asks (rule 6), no-format-pedantry (rule 7), worktree-heads-up (rule 8), 15-25-line FO-prose budget (rule 9).
  - AC-2: `grep -n "Chosen direction:" skills/first-officer/references/claude-first-officer-runtime.md` returned 2 hits — line 194 (template slot) and line 211 (rule #2). `grep -n "^Decision:" skills/first-officer/references/claude-first-officer-runtime.md` returned 1 hit — line 203 (template). Both within the `## Gate Presentation` section (lines 188-218).
  - AC-3: `grep -n '```markdown' skills/first-officer/references/claude-first-officer-runtime.md` returned 2 hits — line 197 (template paste directive: "fenced in a ```markdown code block so authorship is unambiguous") and line 212 (rule #3 explanatory prose).
  - AC-4: `grep -n "Reviewer findings"` returned 2 hits (lines 199, 213); `grep -n "Material:"` returned 2 hits (lines 199, 213); `grep -n "Polish:"` returned 2 hits (lines 199, 213). All three substrings co-occur on the same template line (199) and the same rule-#4 line (213). Omission language present on line 199: "Drop the tier entirely if it has no items. If no reviewer ran, omit this whole block."
  - AC-5: see next checklist item.
  - AC-6: `git diff main -- skills/first-officer/references/first-officer-shared-core.md` returned empty output. Shared-core untouched.
- DONE: AC-5 specifically: read the `## Worked example` fixture in the entity body (lines 151-190) and confirm its first three non-blank, non-fence-delimiter lines match in order: `Gate review: {...}`, `Chosen direction: {non-empty string, not the literal `n/a`}`, `Recommend {approve|reject: ...}`. Report the actual three lines verbatim in the validation report.
  - Fence delimiters in the fixture: line 155 (opening ````), line 160 (opening ```markdown), line 172 (closing ```), line 186 (closing ````). The first three non-blank, non-fence-delimiter lines after the opening fence are lines 156-158, verbatim:
    1. `Gate review: name-pattern rejects stage names with underscores — ideation`
    2. `Chosen direction: Option 2 — pre-validate stage names against Claude Code's NAME_PATTERN at workflow-load time and reject early with a captain-readable error, rather than letting the rejection surface from `Agent()` mid-dispatch.`
    3. `Recommend reject: the AC-2 substring assertion is over-fitted to one error string, and the one-line claim that `commission.py` already validates stage names is contradicted by `grep NAME_PATTERN skills/commission/` returning zero hits.`
  - Line 1 matches `Gate review: {...}`. Line 2's chosen direction is non-empty and is not the literal `n/a` (it names Option 2 with substantive prose). Line 3 begins `Recommend reject: ` per the `Recommend {approve|reject: ...}` form. AC-5 satisfied.
- DONE: PASSED/REJECTED recommendation. Explicitly check (a) scope drift: was anything besides skills/first-officer/references/claude-first-officer-runtime.md and docs/plans/fo-gate-presentation-buries-lede.md changed? (b) internal coherence: read the post-edit `## Gate Presentation` section end-to-end and confirm the nine assembly rules don't contradict each other or the template they accompany (e.g., does rule #2 still align with the template's `n/a` enumeration after the cycle-2 fix).
  - (a) Scope drift: `git diff main --stat` reports exactly two changed files — `docs/plans/fo-gate-presentation-buries-lede.md` (+22/-2) and `skills/first-officer/references/claude-first-officer-runtime.md` (+24/-2). No other files touched. No scope drift.
  - (b) Internal coherence: read post-edit lines 188-218 end-to-end. Template ordering (title / Chosen direction / Recommend / Stage Report paste / Reviewer findings / Assessment / Decision) matches rule #1 (lede first, decision last). Template line 194's `n/a` enumeration reads "simple work stages, merge" — validation is NOT enumerated, so rule #2's claim that validation produces a chosen direction (PASS/REJECTED) is uncontested. The cycle-2 reconciliation noted at entity line 210 is in place. Template line 197 ("fenced in a ```markdown code block") matches rule #3. Template line 199 (Reviewer findings with Material/Polish tiers, omission language) matches rule #4. Template has one `Recommend` line — matches rule #5. Template Decision line example mentions worktree path — matches rule #8. No internal contradictions.
  - Recommendation: **PASSED.** All six AC checks pass with concrete evidence; no scope drift; the post-edit `## Gate Presentation` section is internally coherent; the cycle-2 fix dropping `validation` from the `n/a` enumeration is correctly applied.

### Summary

PASSED. AC-1 through AC-6 verified with concrete grep / git-diff / fixture-reading evidence against the post-edit runtime adapter at commit 5bc85a2c. Scope is clean (only the two expected files changed). The post-edit `## Gate Presentation` section is internally coherent: the nine assembly rules align with the template they accompany, and the cycle-2 reconciliation (dropping `validation` from the template's `n/a` enumeration so rule #2 owns the chosen-direction claim end-to-end) is correctly applied. The worked-example fixture's first three lines match AC-5's structural form verbatim.

### Feedback Cycles

- **Cycle 1 (captain-rejected validation, routed to implementation):** Live application of rule #3 (verbatim Stage Report paste, fenced ` ```markdown `) exposed two defects: (a) nested-fence collision when the Stage Report itself contains triple-backtick literals (e.g., a bullet citing `grep -n '` ```markdown ` '`), forcing escape massaging that violates "verbatim"; (b) captain-decision-weight problem — the ~20-line paste pushes the Decision line below evidence detail the captain only audits, recreating defect #1 (lede buried) through the Stage Report instead of through FO prose. Captain iterated four variants (pure cite, verbatim, truncated, gist) and selected the verb-noun gist form. Routing back to implementation with the new rule text and matching template/AC-3 updates.

## Stage Report: implementation (cycle 2)

- DONE: Edit 1 — Replace rule #3 in `skills/first-officer/references/claude-first-officer-runtime.md` `### Captain-facing assembly rules` with the verb-noun gist + citation form supplied by the dispatch.
  Rule #3 (file line 215) replaced verbatim per the dispatch's new text: "Cite the Stage Report; render a one-line gist roll-up. Do not paste the verbatim Stage Report into the gate message. Under a `Checklist:` heading, render one bullet per item from the ensign's DONE/SKIPPED/FAILED accounting using a verb-noun gist of the item (≤10 words, FO paraphrase that preserves the original item's semantics and introduces no new facts). For SKIPPED or FAILED items, append `— {one-line reason}` after the gist. Then cite the full report by file path and line range so the captain can audit if they want. If a reviewer Material finding directly questions a specific checklist item's evidence, inline that item's evidence paragraph from the report under the relevant reviewer-finding bullet — so the captain can decide without opening the file. Otherwise no Stage Report content appears in the gate message." Prior fenced-`markdown`-block phrasing is gone.
- DONE: Edit 2 — Replace the Stage Report paste line in the template (file line 197) with the `Checklist (from ## Stage Report in {entity_file_path} lines {start}-{end}):` heading plus DONE/SKIPPED/FAILED gist-bullet form.
  Template (file lines 197-200) now reads: `Checklist (from ## Stage Report in {entity_file_path} lines {start}-{end}):` followed by `- DONE: {≤10-word gist of item}`, `- SKIPPED: {gist} — {one-line reason}`, `- FAILED: {gist} — {one-line reason}`. The single ` ```markdown ` fenced-paste directive that lived on line 197 pre-cycle is removed.
- DONE: Edit 3 — Update AC-3 in the entity body (`## Acceptance criteria` section) to assert the citation form and the absence of any fenced ` ```markdown ` block.
  AC-3 rewritten verbatim per the dispatch's suggested rewrite: asserts the template instructs Stage Report citation by file path and line range (no verbatim paste, no inline fenced ` ```markdown ` block); verification spans three loose substring assertions — `grep -n "Checklist:"` returns at least one in-section hit, `grep -nE "from .*Stage Report.*in.*lines"` returns at least one in-section hit, and `grep -c '` ```markdown ` '` returns zero file-wide.
- DONE: Update the `## Worked example` fixture (entity lines ~155-186) to use the new template form — replace the inner fenced ` ```markdown ` Stage Report block with `Checklist:` + citation + gist bullets.
  Fixture's inner ` ```markdown ` block (formerly entity lines 160-172) replaced with three Checklist gist bullets: `- DONE: AC end-state phrasing + verified-by citations`, `- DONE: Option 2 selected with tradeoff reasoning`, `- DONE: static grep + fixture-reading proof level`. Outer 4-backtick fence preserved so the fixture stays self-contained with no nested-fence issues. Reviewer findings / Assessment / Decision blocks unchanged. Surrounding prose (the "Fresh-eye check" paragraph and "For comparison" paragraph) inspected and left intact — neither cites the ` ```markdown ` fencing mechanism.
- DONE: All AC grep checks re-run against the post-edit runtime adapter.
  AC-1: 9 numbered rules in `### Captain-facing assembly rules` (unchanged count after rule-#3 replacement and rule-#2/#9 internal-coherence touch-ups). AC-2: `Chosen direction:` at template line 194 and rule-#2 line 214; `Decision:` at template line 206. AC-3a: `Checklist:` substring hits inside the section on line 215 (rule-#3 quoted-literal `Checklist:`). AC-3b: `from ## Stage Report in {entity_file_path} lines {start}-{end}` matches the citation-form regex on line 197. AC-3c: `grep -c '` ```markdown ` '` over the whole runtime adapter returns 0 — no surviving fenced markdown directive. AC-4: Reviewer findings / Material: / Polish: + omission language on lines 202 (template) and 216 (rule #4). AC-6: `git diff main -- skills/first-officer/references/first-officer-shared-core.md` returns empty (zero lines).
- DONE: Internal-coherence sweep of rules #2 and #9 to remove dangling references to the no-longer-existing paste.
  Rule #2 (file line 214) updated from "infer it from the Stage Report paste or the entity file" → "infer it from the Checklist gist or open the entity file" — keeps rule #2 internally consistent with the cited-not-pasted contract. Rule #9 (file line 221) rewritten to drop the "plus the pasted Stage Report" length carve-out and instead state that the full gate message (including the Checklist gist roll-up) fits in 15-25 lines; explicitly cross-references rule #3's cited-not-pasted contract. This is a minor scope expansion beyond the dispatch's three named edits, justified as the smallest fix needed to keep the section internally coherent after rule #3 was replaced — leaving the dangling "paste" references would have given a future validator a real ground to bounce.

### Summary

Cycle 2 applies the captain's selected verb-noun-gist + citation form for the Stage Report rendering. Rule #3 in the runtime adapter's `### Captain-facing assembly rules` is replaced; the template's verbatim-paste line becomes a `Checklist:` heading + DONE/SKIPPED/FAILED gist-bullet form with a file-path/line-range citation; AC-3 in the entity body is rewritten to assert the citation form via loose substring greps and to assert the disappearance of any fenced ` ```markdown ` directive (verified by `grep -c` returning zero). The worked-example fixture's inner Stage Report paste is replaced with the new Checklist form; the outer 4-backtick fence keeps the fixture self-contained. Two coherence touch-ups beyond the named edits: rule #2's parenthetical and rule #9's length budget — both had dangling references to the now-absent paste. All AC grep checks pass; shared-core diff stays empty.
