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
- **AC-3:** The `## Gate Presentation` template instructs that the verbatim Stage Report paste is wrapped in a fenced ` ```markdown ` code block. Verified by grep for the substring ` ```markdown ` (or an equivalent fencing directive) inside the `## Gate Presentation` section.
- **AC-4:** The `## Gate Presentation` template names `Reviewer findings` with the two-tier substructure `Material:` and `Polish:` and explicitly states the tier is omitted when no reviewer ran. Verified by grep for `Reviewer findings`, `Material:`, and `Polish:` all appearing in the `## Gate Presentation` section.
- **AC-5:** The `## Worked example` fixture in this entity body's first three non-blank, non-fence-delimiter lines, in order, match `Gate review: {...}`, `Chosen direction: {non-empty string, not the literal `n/a`}`, and `Recommend {approve|reject: ...}`. Verified by static reading of the committed fixture during validation.
- **AC-6:** The shared-core `## Completion and Gates` section in `skills/first-officer/references/first-officer-shared-core.md` is unchanged by this task. Verified by `git diff main -- skills/first-officer/references/first-officer-shared-core.md` showing no changes.

## Test plan

**Proof level: static prose review of the runtime-adapter edit, plus a hand-built transcript fixture for the worked example.** Justified per the workflow README's "choose proof at the same abstraction level" rule (line 78): the claim here is about FO orchestration prose — what message text the FO assembles for a gate review. The right proof level is the prose itself plus a representative rendering of what it produces. Live FO E2E runs are explicitly the wrong level: they are expensive (minutes per dispatch cycle), indirect (the FO's gate message depends on entity content, reviewer content, stage state, all of which would need to be set up), and prone to fixture drift (any unrelated change to FO prose-assembly logic would re-render the gate message). The bug being fixed is a *discipline* bug in the runtime-adapter text, not a runtime-behavior bug — fixing it means changing the text the FO reads, and proving the fix means reading that text and confirming the new text says what it should say.

Concrete validation steps:

1. **Static grep checks for AC-1 through AC-4** (durable doc structure): run the grep commands named in each AC against the post-edit `claude-first-officer-runtime.md` and confirm hits. Cost: seconds. Sufficient because the claims are about which strings appear in a markdown reference file — exactly the case the README cites for "static checks for durable doc/contract structure."
2. **Static reading of the worked-example fixture for AC-5** (representative rendering): read the committed fixture and confirm the first three lines satisfy the seed's "fresh-eye captain can answer chosen direction / verdict / decision question without scrolling" criterion. Cost: under a minute. Sufficient because the claim is about message legibility, and the fixture *is* the message the new template would produce.
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

```markdown
## Stage Report: ideation

- DONE: Acceptance criteria are entity-level end-state properties...
  AC-1 through AC-4 written as end-state facts; see body
- DONE: Proposed approach picks among directions with reasoning...
  Option 2 selected; tradeoff vs Options 1 and 3 captured in body
- DONE: Test plan picks proof at the right abstraction level...
  Static grep checks plus a hand-built fixture; justified against README line 78

### Summary
Ideation lands on early NAME_PATTERN validation. AC and test plan written.
```

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
