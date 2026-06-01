# Encoding the four deliverable principles into the Spacedock contract

A design study. **Proposal only — no contract/scaffolding file is edited here.**

The goal: encode four engineering principles into the workflow contract so future
Spacedock dev work cannot drift into the antipatterns they forbid. For each
principle this study names the exact placement, gives before/after wording in the
voice of the surrounding text, states how the rule itself is falsifiable (dogfood),
and flags what needs a captain decision.

The four principles, in priority order:

1. **No doc-only deliverable.** Every entity's deliverable must be falsifiable by an
   oracle *external to itself*.
2. **Proof is behavioral, not grep.** Proof must exercise the behavior and observe
   the outcome — not assert spelling in code or prose.
3. **Enforce in code, not prose.** Prefer a contract the binary enforces over one the
   agent is merely instructed to follow.
4. **Spike the riskiest unknown in ideation.** When soundness rests on an unverified
   mechanism, ideation runs the smallest end-to-end exercise of that path first.

---

## 0. What the contract already says (so we sharpen, not duplicate)

The workflow README (`docs/dev/README.md`) already gestures at all four:

- ideation Outputs: *"Acceptance criteria must include how each criterion will be
  tested"*, *"If an AC item reads as an imperative verb phrase, rewrite it as the
  end-state property it produces"*, *"Choose proof at the same abstraction level as
  the claim … static skill tests for instruction text … live workflow smoke tests
  only when runtime behavior is the claim."*
- ideation Bad: *"static prose tests for behavioral requirements, or tests that pass
  while missing the intended behavior."*
- validation Bad: *"accepting passing tests that encode stale prose, obsolete
  assumptions, or the wrong abstraction level."*
- validation Spot-check principle: *"do a cheap fixture or single-command spot-check
  to verify the infrastructure works end-to-end."*

The operating contract (`first-officer-shared-core.md`) already has the **AC coverage
cross-check** at `## Completion and Gates`: *"at every gate, scan the entity body's
`## Acceptance criteria` section and confirm each `**AC-N**` item has at least one
evidence citation … REJECT if this stage was the natural place to address it."*

These are the right hooks. Today they say *test your claims* and *cite evidence*.
They do not yet say *the evidence must be able to fail*, *spelling is not behavior*,
or *a decision is not a queue deliverable*. That gap is what this study closes.

### The drift is already live in this workflow

Two entities in `docs/dev/.spacedock-state/` demonstrate exactly the antipatterns,
which makes them the dogfood targets:

- **`external-tracker-checkpoint/index.md` (currently `status: validation`).** It is
  a self-described *"design checkpoint: the deliverable is a decision."* AC-1..AC-4
  are genuine behavioral deliverables (unknown-field round-trip through the real
  parser/mutator). But **AC-6** reads:

  > **AC-6 - The v1 decision … is recorded with justification …**
  > Verified by: this entity's **v1 DECISION** section states the decision … Proof:
  > static prose review of the decision section.

  This AC's oracle *is the prose it asserts exists.* It can never fail. It is
  principle 1's textbook self-oracle, heading for `done` + PASSED. And **AC-5**'s
  *"Verified by: design review … shows no `if source == "kata"`-style branching …
  absence of tracker names in command/parser code paths"* is principle 2's
  grep-over-code false-pass.

- **Archived `skill-integration-vendor-branch/index.md`** produced
  `skills/integration/skill_text_test.go`, whose assertions are
  `strings.Contains(content, "spacedock status --discover")` over markdown — proof
  that the *contract document* contains a substring, not that the *agent* runs the
  command. Principle 2's grep-over-prose, the worst tier.

Both are cited concretely below.

---

## Principle 1 — No doc-only deliverable

> Every entity's deliverable must be falsifiable by an oracle EXTERNAL TO ITSELF. An
> entity whose only output is prose is the antipattern: it becomes its own oracle. A
> genuine pure decision belongs in an ADR / the roadmap, not the implementation→
> validation dev queue. Cleanup/overhaul DO qualify (oracle = changed code + green
> tests). Design is a STAGE, not a terminal entity.

### (a) Placement

- **Workflow README → `ideation` Outputs and `done` Good/Bad** (workflow-specific
  gate rule: what an entity in *this* queue must be).
- **Operating contract → `## Completion and Gates`, the AC coverage cross-check**
  (cross-workflow gate discipline: how the FO refuses a self-oracle AC at any gate).

Both, because the rule has two faces: *what an entity must be* (README) and *the FO
refusing to pass one that isn't* (contract).

### (b) Before / after

**README `ideation` Outputs — append one bullet after the existing
"If an AC item reads as an imperative verb phrase …" bullet:**

BEFORE (last two AC bullets):

```
  - If an AC item reads as an imperative verb phrase, rewrite it as the end-state property it produces.
  - Test plans should state what verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.
```

AFTER:

```
  - If an AC item reads as an imperative verb phrase, rewrite it as the end-state property it produces.
  - Each AC's "Verified by" must name an oracle EXTERNAL to the entity body: a test, a command's output/exit code, a file the change produces, or resulting on-disk state. An AC whose oracle is the entity's own prose ("verified by static review of this entity's decision section") cannot fail and is therefore not an acceptance criterion. If the entity's only output is prose — a decision, a "design checkpoint" with no shipped artifact — it does not belong in this queue; record the decision in the roadmap or an ADR instead. Cleanup and overhaul DO qualify: the oracle is the changed code plus green tests.
  - Test plans should state what verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.
```

**README `done` Bad — extend the existing line:**

BEFORE:

```
- **Bad:** Closing without reading the validation report or overriding a REJECTED recommendation without reason
```

AFTER:

```
- **Bad:** Closing without reading the validation report, overriding a REJECTED recommendation without reason, or reaching done with PASSED on an entity whose deliverable is prose with no external oracle (a design that concludes "do not build X" ships as a roadmap/ADR decision, not a PASSED dev-queue entity)
```

**Operating contract `## Completion and Gates`, AC coverage cross-check —
the change is a clause inside the existing paragraph:**

BEFORE:

```
**AC coverage cross-check.** Additionally, at every gate, scan the entity body's `## Acceptance criteria` section and confirm each `**AC-N**` item has at least one evidence citation from this stage's report or a prior stage report. Name any AC without evidence; REJECT if this stage was the natural place to address it. This cross-check is independent of checklist DONE/SKIPPED/FAILED accounting — checklist items are dispatch signals, AC items are entity properties.
```

AFTER:

```
**AC coverage cross-check.** Additionally, at every gate, scan the entity body's `## Acceptance criteria` section and confirm each `**AC-N**` item has at least one evidence citation from this stage's report or a prior stage report. Name any AC without evidence; REJECT if this stage was the natural place to address it. The evidence must come from an oracle EXTERNAL to the entity body — a test, a command's output or exit code, or resulting on-disk state. An AC whose only cited proof is review of the entity's own prose ("verified by static review of this entity's decision section") is a self-oracle: it can never fail, so it does not satisfy this cross-check. When an entity's deliverable is purely a decision with no shippable artifact, do not advance it to a terminal PASSED verdict; surface to the captain that it belongs in the roadmap or an ADR. This cross-check is independent of checklist DONE/SKIPPED/FAILED accounting — checklist items are dispatch signals, AC items are entity properties.
```

### (c) Dogfood — how the rule itself is falsifiable

This is the principle that admits the strongest mechanism, because the contract
already has a code-enforced terminal-transition guard to model on.

**Strongest — code-enforced (`status` lint + a terminal-set guard).** Extend the
native `status` binary with an AC-oracle check, modeled exactly on the existing
mod-block / merge-hook invariant in `internal/status/handlers.go` (the
`isTerminalUpdate()` block at the `runSet` guard) and `internal/status/mutate.go`
(`runArchive`):

- A new `validateWorkflow` check (in `internal/status/validate.go`, alongside
  `findEntityFormConflicts`) parses each entity body's `## Acceptance criteria`
  section, pulls every `**AC-N**` item and its `Verified by:` line, and flags any AC
  whose `Verified by` text matches a self-oracle pattern — the operative signal is a
  `Verified by` that names *this entity's own* sections as its proof (regex over
  `this entity`, `this entity's`, `decision section`, `static (prose )?review of
  (this|the) … (decision|section)`). It emits the standard
  `Error: … slug= … path=` evidence line. Surfaced by `spacedock status --validate`,
  which the validation stage already runs before trusting state.
- The hard gate: extend the terminal-transition guard in `runSet`. Today
  `isTerminalUpdate()` + the mod-block/merge-hook checks refuse `status=done`,
  `completed`, `verdict`, `worktree=` when a precondition fails. Add: refuse a
  terminal `--set` with `verdict=passed` when the entity's ACs contain a self-oracle
  match and no `--force`. Error text in the same idiom as the mod-block guard:
  `Error: entity {slug} cannot advance to terminal PASSED — AC-{N} cites no external
  oracle (verified by review of the entity's own prose). Move the decision to the
  roadmap/ADR, give the AC an external oracle, or use --force to bypass.`

  This is the analog of *"This catches the FO forgetting to set `mod-block`"* — it
  catches the FO passing a self-oracle entity. `--force` is the captain-confirmed
  bypass, mirroring the mod-block `--force` semantics already documented in
  `## Mod-Block Enforcement`.

  Test it the way `archive_guard_test.go::TestTerminalSetUnderModBlockRejected` tests
  the mod-block guard: drive the real binary against a fixture entity whose only AC
  is a self-oracle, assert exit 1, assert the entity was NOT mutated to `done`. And a
  passing case: an entity with an external-oracle AC reaches `done` cleanly. That is a
  behavioral test of the guard (principle 2-compliant), not a grep.

**FO-gate-overlay-enforced.** The AC-coverage cross-check wording above is the FO
step. The FO performs it at every gate. Its proof ceiling is the same as any FO-prose
rule (see principle 3) — so the code guard above is the real guarantee; the prose
points at it.

**Honest ceiling / where it mis-fires (adversarial).** A regex over `Verified by`
text is itself a grep over prose — which principle 2 warns against. Defensible
because it is a *lint that flags suspects for a human gate*, not a pass/fail oracle: a
false positive is surfaced to the captain who `--force`s past it; a false negative
(an AC that is a self-oracle but phrased to dodge the regex, e.g. *"verified by
inspection of the recorded rationale"*) is caught by the FO cross-check, not the lint.
The two layers are complementary, neither is the sole guarantee, and the lint never
silently blocks (it warns; the guard blocks only the narrow terminal-PASSED case with
`--force` escape). The deeper limit: no static check can decide whether prose-only
output is a *legitimate ADR misfiled into the queue* vs *a non-deliverable* — that is
a judgment, which is why the terminal case routes to the captain rather than
hard-failing.

### (d) Captain decisions

- **ADR path.** There is no `docs/decisions/` today; the roadmap
  (`docs/roadmap/bootstrap-roadmap.md`) already hosts pure decisions — its "External
  Tracker Checkpoint" section literally poses *"decide whether any bidirectional sync
  belongs in v1."* Decision: do we add a `docs/decisions/` ADR dir, or keep
  decisions in the roadmap? **Recommendation: keep them in the roadmap for now**
  (YAGNI; the roadmap is the established home), and have the rule point there.
- **A `decision` entity-type that terminates at ideation?** A workflow could declare
  a stage like `decision` with `terminal: true` reachable from ideation, so a pure
  decision has a legitimate terminal home *inside* the queue without an
  implementation→validation deliverable. Decision: introduce it, or keep decisions
  out of the queue entirely? **Recommendation: keep them out** — adding an entity
  type is more machinery than the rule needs, and the roadmap already absorbs
  decisions. Flag for the captain because it is an architecture choice.
- **New `status --validate` sub-check + terminal-PASSED guard.** Adding the
  self-oracle lint and the guard is new binary behavior (stdlib-only Go, modeled on
  existing guards). Needs captain sign-off as a tracked entity, and the dogfood
  target (`external-tracker-checkpoint` AC-5/AC-6) should be re-shaped first so the
  guard's first real run is green-by-construction.

---

## Principle 2 — Proof is behavioral, not grep

> Proof must EXERCISE the behavior and OBSERVE the outcome. grep-over-code asserts
> spelling not behavior; grep-over-PROSE is WORSE (the document is not the behavior).
> CRUCIAL: a grep (substring present/absent) is the antipattern, but an INVARIANT
> CHECK that parses real artifacts and tests a relationship is LEGITIMATE. Do not ban
> all static tests.

### (a) Placement

- **Workflow README → `ideation` "Choose proof at the same abstraction level"
  bullet** (workflow-specific: what counts as proof of a claim in *this* queue). This
  is the load-bearing placement — it already names the proof-selection rule.
- **Workflow README → `validation` Bad** (the validator must reject grep-proof).

Not the operating contract: proof-level selection is workflow-specific (a docs
workflow's claims differ from a code launcher's). The contract already defers proof
choice to the README via the AC cross-check.

### (b) Before / after

**README `ideation` Outputs — rewrite the "Choose proof" bullet:**

BEFORE:

```
  - Choose proof at the same abstraction level as the claim: Go unit tests for parser and command behavior, golden fixtures for status output, static skill tests for instruction text, and live workflow smoke tests only when runtime behavior is the claim.
```

AFTER:

```
  - Choose proof at the same abstraction level as the claim, and prefer proof that EXERCISES the behavior and OBSERVES the outcome (output bytes, exit code, resulting on-disk state, or a parametrized test feeding many inputs and asserting uniform handling): Go unit tests for parser and command behavior, golden fixtures for status output, behavior fixtures that drive the binary for command-level claims, and live workflow smoke tests only when runtime behavior is the claim.
  - A substring grep is not behavioral proof. grep-over-code asserts spelling (it false-passes on `switch source { case "kata" }` and false-fails on a rename); grep-over-prose (asserting a skill or contract document contains/lacks a substring) is weaker still — the document is not the behavior, so "the contract says run --version at step 0" never proves the agent version-gates. When a claim is about behavior, the proof must run it. A static check IS legitimate when it parses real artifacts and tests a RELATIONSHIP between real values — e.g. "the plugin manifest's `requires-contract` range brackets the binary's `CONTRACT_VERSION`" exercises real comparison logic over real values and fails on real drift. The line is: invariant over real values = legitimate; substring present/absent = grep.
```

**README `validation` Bad — extend the existing line:**

BEFORE:

```
- **Bad:** Rubber-stamping without testing, ignoring failing edge cases, validating against wrong criteria, or accepting passing tests that encode stale prose, obsolete assumptions, or the wrong abstraction level
```

AFTER:

```
- **Bad:** Rubber-stamping without testing, ignoring failing edge cases, validating against wrong criteria, accepting passing tests that encode stale prose, obsolete assumptions, or the wrong abstraction level, or accepting a substring grep (over code or over contract/skill prose) as proof of a behavioral claim — proof of behavior must run the behavior; a static test passes only as an invariant over real parsed values, not a spelling check
```

### (c) Dogfood — how the rule itself is falsifiable

**Strongest — FO-gate-overlay + the validator's own judgment, with the existing
behavioral tests as the standing example.** This principle resists a clean code
guard: a binary cannot reliably decide "is this test behavioral or a grep?" by static
analysis (the test is arbitrary Go). What CAN be enforced:

- The standing example is in-repo and behavioral, so the bar is demonstrated, not
  just asserted: `internal/status/native_mutation_test.go` and
  `archive_guard_test.go` drive the real binary, assert exit codes and resulting file
  bytes — and `spacedock-packaging` AC-1's *"manifest's `requires-contract` range
  brackets the binary's `CONTRACT_VERSION`"* is the legitimate-invariant exemplar.
  Contrast `skills/integration/skill_text_test.go`, the grep-over-prose foil. The
  rule names both so the validator has a concrete good/bad pair.
- FO/validator step: at the validation gate, for each AC whose claim is behavioral,
  confirm the cited test runs the behavior (drives the binary / asserts output, exit,
  or state) rather than `strings.Contains` over source or markdown. This is an FO
  cross-check addition, ceiling = "performed at gate."

**Weakest-but-honest — a `status --validate` heuristic flag (optional).** The lint
could flag AC `Verified by` lines whose *only* cited proof is the word "grep"/"greps"
over a skill or contract path for a claim phrased behaviorally. This is itself a grep
and would mis-fire (it cannot read the test); propose it only as a warning suspect for
the gate, never a hard fail. Recommend NOT building it initially — the foil/example
pair plus the validator step is the right altitude; a grep that polices greps is poor
ROI.

**Honest ceiling.** This is the principle with the weakest code-enforcement story.
Be explicit: principle 2's guarantee is mostly validator discipline backed by an
in-repo good/bad example pair. The code-enforceable part is *making the legitimate
invariant-check pattern easy to reach for* (the manifest-range test exists; cite it),
not *detecting* a bad grep automatically.

### (d) Captain decisions

- Whether to ship the optional `--validate` grep-suspect warning. **Recommendation:
  no** (low ROI, self-referential). Flagged so the option is on record.
- No new entity-type or path needed; this principle lives entirely in the README and
  the FO gate step.

---

## Principle 3 — Enforce in code, not prose

> Prefer a contract the binary/code ENFORCES over one the agent is merely INSTRUCTED
> to follow. Push guarantees into the binary; let the prose point at the code gate.
> Where a contract must live in agent instructions, its proof ceiling is "wording
> present", so it cannot count as AC satisfaction — the real guarantee must be a
> code-level gate underneath.

### (a) Placement

- **Operating contract → a short principle near the top of `## Dispatch` / referenced
  from `## Completion and Gates`** (cross-workflow: how the FO frames a guarantee and
  how ensigns frame ACs). This is *the* meta-principle about where guarantees live;
  it belongs in the operating contract, not a single workflow README.
- **Workflow README → `ideation` Outputs** (one bullet so ensigns frame ACs against a
  code gate where one is possible).

### (b) Before / after

**Operating contract — add a subsection. The natural home is right after the AC
coverage cross-check in `## Completion and Gates`, since that is where AC proof
ceilings are adjudicated. New text:**

```
**Code-gate preference.** When a guarantee CAN be enforced by the binary or code (a
`status` guard, a test that fails on violation), prefer that over a guarantee the
agent is only instructed to follow in prose. A prose-only contract — a line in a skill
or reference that the agent is told to obey — has a proof ceiling of "the wording is
present." Wording-present is not behavior (see the proof rule), so a prose-only
contract MUST NOT count as AC satisfaction on its own: if the guarantee matters, the
real assurance is a code-level gate underneath, and the prose points at that gate. An
AC of the form "the contract says X" is satisfied only by "the binary/test enforces X
and here is the run that proves it."
```

**README `ideation` Outputs — add one bullet (after the new principle-1 bullet):**

```
  - Prefer ACs that a code gate can enforce over ACs the agent is merely instructed to follow. Where a behavior can be guarded by the binary or a failing test, the AC's oracle is that gate, not a sentence in a skill file. An AC whose only proof is "the skill text says to do X" has a ceiling of wording-present and cannot stand alone.
```

### (c) Dogfood — how the rule itself is falsifiable

**Strongest — the principle is enforced by principle 1's code guard, transitively.**
Principle 3 says "a prose-only AC cannot satisfy a gate." The self-oracle lint and
terminal-PASSED guard from principle 1 ARE the enforcement: an AC whose only proof is
"the skill text contains X" is a prose-self-oracle and trips the same guard. So
principle 3 does not need its own separate code path — it shares principle 1's gate.
The in-repo proof is the existing **mod-block / merge-hook mechanism enforcement**
itself: that is the canonical "guarantee pushed into the binary," the model the whole
proposal cites — *"Enforced at the mechanism level … regardless of whether the FO set
`mod-block` first … This catches the FO forgetting."* The contract's own text already
practices principle 3; this rule names the practice.

**Honest ceiling.** The principle is itself partly prose (the "prefer code gates"
instruction in the contract). Its proof ceiling is wording-present — which the rule
openly admits is not a guarantee. That is acceptable and consistent: the rule does not
claim to BE a code gate; it claims to redirect guarantees TO code gates, and its teeth
are principle 1's guard. Pretending the prose itself enforces anything would violate
the very principle — so the proposal does not.

### (d) Captain decisions

- None beyond principle 1's (it shares that guard). Worth flagging only that the
  contract gains a stated meta-principle, which the captain should bless as policy.

---

## Principle 4 — Spike the riskiest unknown in ideation

> When a design's soundness rests on an UNVERIFIED MECHANISM, ideation must run the
> smallest end-to-end exercise of the riskiest path FIRST ("what would invalidate the
> rest of this work if it broke? that goes first; pay the small bill first"). A spike
> is throwaway; its output is behavioral evidence + learning, often seeding the
> implementation's mechanism-first test fixture.

### (a) Placement

- **Workflow README → `ideation` Outputs and `ideation` Staff review**
  (workflow-specific: this queue's ideation must spike risky mechanisms; the existing
  Staff-review trigger is the natural enforcement seam).
- **Operating contract → `## Probe and Ideation Discipline`** (cross-workflow: the FO
  already has an ideation-probe section; the spike rule extends it). There is also a
  `running-research-spikes` skill the rule can reference.

Both: the README states the requirement for the entity; the contract states the FO's
behavior when dispatching/reviewing ideation.

### (b) Before / after

**README `ideation` Outputs — add a bullet (after the test-plan bullet):**

BEFORE (the test-plan bullet, end of the Outputs list region):

```
  - Test plans should state what verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.
```

AFTER (insert a new bullet after it):

```
  - Test plans should state what verifies the implementation, estimated cost/complexity, and whether fixture, CLI, or live workflow tests are needed.
  - When the design's soundness rests on an unverified mechanism or assumption — a parser round-trip, a runtime handoff, an on-disk format, a tool actually supporting a flag — ideation MUST run the smallest end-to-end exercise of that riskiest path first and record the behavioral evidence in the entity body. Ask: "what would invalidate the rest of this work if it broke?" — that gets spiked first; pay the small bill first. The spike is throwaway, but its output (evidence + learning) seeds the implementation's mechanism-first test fixture. If no mechanism is unverified — the design composes only already-proven behavior — record "no spike needed: {the proven mechanisms it relies on}" so the determination is auditable rather than silent.
```

**README `ideation` Staff review — extend the existing trigger:**

BEFORE:

```
- **Staff review:** When the FO assesses ideation as complex, such as native status parity, split-root behavior, or skill integration, it should request an independent review before presenting the ideation gate. The review checks design soundness, test plan sufficiency, and gaps.
```

AFTER:

```
- **Staff review:** When the FO assesses ideation as complex, such as native status parity, split-root behavior, or skill integration, it should request an independent review before presenting the ideation gate. The review checks design soundness, test plan sufficiency, gaps, and that the riskiest unverified mechanism was spiked (or that the entity records an auditable "no spike needed" with the proven mechanisms it relies on). A design whose soundness rests on an unspiked, unverified mechanism is not ready for the gate.
```

**Operating contract `## Probe and Ideation Discipline` — add a bullet:**

BEFORE (the section's existing first bullet):

```
- when checking whether tool X supports Y, read X's schema directly via ToolSearch before greping for existing callers — usage presence is not existence evidence.
```

AFTER (insert a new bullet before it, as the section's lead discipline):

```
- when a dispatched-ideation design rests on an unverified mechanism (a format round-trip, a runtime handoff, a tool actually supporting a flag), the riskiest path is spiked end-to-end first — the smallest exercise that would invalidate the rest of the work if it broke. Behavioral evidence from the spike goes in the entity body; "no spike needed" is recorded with the proven mechanisms relied on. See the `running-research-spikes` skill. This is the integration-level analog of the AC oracle rule: arrive at the gate with the riskiest claim DEMONSTRATED, not asserted.
- when checking whether tool X supports Y, read X's schema directly via ToolSearch before greping for existing callers — usage presence is not existence evidence.
```

### (c) Dogfood — how the rule itself is falsifiable

**FO-gate-overlay-enforced (the realistic ceiling).** "A spike happened" is hard to
code-detect, but the *recorded determination* is checkable at the ideation gate:

- The ideation gate (gated stage, `gate: true`) requires the entity body to contain
  EITHER spike evidence (a behavioral result — output, exit, state) OR an explicit
  `no spike needed: {mechanisms}` line. The FO checks for one of these before
  presenting the gate; its absence is a gate-blocker, not a silent pass. This rides
  the existing Staff-review seam.
- The in-repo proof that this works: `external-tracker-checkpoint` already
  *practices* it — its body flags assumptions **A1/A2/A3** as "depend on sibling
  entities not yet finalized; the FO should reconcile them at the ideation gate," and
  its preservation property is spiked by the sibling `native-state-dir`'s
  unknown-field round-trip tests (`native_mutation_test.go::TestNativeUnknownField
  Preservation`). That is the spike-seeds-the-fixture pattern, already in the repo —
  the rule names an existing good practice.

**Optional code assist — a `status --validate` ideation-gate check.** When an entity
is at `status: ideation`, the lint could warn if the body has neither a spike-evidence
marker nor a `no spike needed:` line. This is a presence check (a grep), so per
principle 2 it is a warning suspect for the human gate, never a hard fail, and it only
checks the *determination is recorded*, not that the spike was sound. Recommend
building it only if the FO-overlay proves leaky.

**Honest ceiling.** Code cannot judge whether the spiked path was actually the
riskiest one — that is design judgment, owned by the Staff reviewer. The enforceable
floor is "a determination is recorded and auditable." That is genuinely better than
silence (today nothing forces the question to be asked), and it is honest about not
being a soundness guarantee.

### (d) Captain decisions

- **MUST vs SHOULD.** The wording above uses **MUST** for "run the smallest exercise
  first" with an explicit escape ("no spike needed: {mechanisms}" recorded). Decision
  for the captain: keep it MUST-with-escape, or soften to SHOULD? **Recommendation:
  MUST-with-recorded-escape** — it forces the question to be asked and answered on the
  record, which is the whole point ("pay the small bill first"), while the escape
  prevents it from taxing genuinely-mechanical work.
- **How "no spike needed" is recorded.** Proposed: a literal `no spike needed:
  {mechanisms}` line in the entity body, checkable at the gate. Captain to confirm the
  exact marker text if a `--validate` assist is later built (the marker must be
  greppable to be lint-checkable).
- Whether to build the optional ideation-gate `--validate` presence check now or defer.

---

## Cross-cutting: how the four guards compose

| Principle | README (workflow-specific) | Operating contract (cross-workflow) | Strongest enforcement | Ceiling |
|---|---|---|---|---|
| 1 No doc-only | ideation Outputs + done Bad | AC coverage cross-check | **Code**: `--validate` self-oracle lint + terminal-PASSED `--set` guard (models mod-block guard) | Lint is suspect-flagging; captain `--force` escape |
| 2 Behavioral not grep | ideation proof bullet + validation Bad | (defers to README) | **FO/validator step** + in-repo good/bad example pair | Cannot auto-detect a bad grep; validator judgment |
| 3 Code not prose | ideation Outputs bullet | Code-gate-preference subsection | **Shares principle 1's guard** (prose-only AC = self-oracle) | The rule itself is prose; teeth are P1's gate |
| 4 Spike risky unknown | ideation Outputs + Staff review | Probe and Ideation Discipline | **FO ideation-gate check** for spike-evidence-or-recorded-escape | Cannot judge if the right thing was spiked |

A single new `status --validate` sub-check + one extended terminal-`--set` guard
(both stdlib-only, both modeled on the existing mod-block / merge-hook invariant)
carry the code-enforceable share of principles 1 and 3. Principles 2 and 4 are
predominantly FO-gate discipline backed by in-repo exemplars, with optional warning
lints flagged but not recommended for v1.

---

## Canonical vs vendored placement

The operating-contract files exist twice:

- **Canonical** plugin: `~/git/spacedock/skills/first-officer/references/…` and
  `~/git/spacedock/skills/ensign/…`. These use the plugin-private path
  `{spacedock_plugin_dir}/skills/commission/bin/status`.
- **Vendored** in this project: `spacedock-v1/skills/first-officer/references/…`,
  ahead of canonical — it routes to the launcher (`spacedock status …`) and carries
  the **Split-Root Worktree Contract** and concurrency-safe-commit amendments that
  canonical lacks (confirmed by diff: vendored is 353 lines vs canonical 340, with the
  launcher-command and split-root deltas).

This project's self-hosting goal is for the vendored skill surface to BECOME the
published plugin. So:

- **Workflow-README edits (principles 1, 2, 4) → land in this project's
  `docs/dev/README.md` only.** The README is per-workflow scaffolding; there is no
  canonical copy. No fan-out.
- **Operating-contract edits (principles 1, 3, 4) → author in the VENDORED copy
  (`spacedock-v1/skills/first-officer/references/first-officer-shared-core.md`) first**,
  because the vendored copy is the project's source of truth and already the divergent
  leading edge. Then flow upstream to canonical (`~/git/spacedock`) as part of the
  self-hosting reconciliation — the same direction the launcher-command and split-root
  amendments already flowed. Authoring in canonical first would force a second
  re-vendor and risk re-introducing the plugin-private path the vendored copy
  deliberately removed.
- **The code guard (`internal/status/…`) lands only in this project** — it is the
  launcher, which is what the plugin will ship as.

**One caution.** All these reference files are listed under the **Scaffolding
guardrail** (`code-project-guardrails.md`: *"Treat `skills/`, `agents/`, `references/`,
`plugin.json`, and workflow `README.md` … as protected surfaces. Changes … should be
tied to a tracked task."*). So every edit proposed here MUST go through a tracked dev
entity dispatched to a worker — the FO cannot hand-edit them on main. The dogfooding
is exact: this proposal's own contract changes must themselves arrive at the gate with
behavioral proof (the guard's red/green tests), not prose review.
