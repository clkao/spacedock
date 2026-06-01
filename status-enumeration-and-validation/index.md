---
id: s0cbyrbh9b2yb456exnsm72b
title: status correctness — --next undercount, archive enumeration consistency, --set stage-name validation
status: validation
source: issue sweep (2026-05-31) — CL dev-workflow-ergonomics triage; consolidates #230, #207, #189, #163
started: 2026-06-01T05:04:22Z
completed:
verdict:
score: "0.28"
worktree: .worktrees/spacedock-ensign-status-enumeration-and-validation
issue:
mod-block: merge:pr-merge
---

Fix three correctness/ergonomics gaps in `spacedock status` that mislead the FO's scheduling and
let bad state in silently. These are read/scheduling-path bugs the FO hits every session.

Consolidates open issues:
- **#230** — `status --next` undercounts: when an entity's stage report is committed but the entity
  has not been advanced, it is not surfaced as dispatchable, so ready work stays invisible.
- **#207** — entity enumeration is inconsistent depending on whether an entity lives at top-level
  vs in `_archive/`; the helper should enumerate consistently.
- **#189** — `status --set {slug} status=X` does not validate `X` against the workflow's declared
  `stages.states[].name` membership. **Scope note:** `internal/status/validate.go`
  (`validateWorkflowStageNames`) already validates the README stage-name *format* against a regex —
  this AC is the distinct **membership** check on a `--set` value, not the format check; do not
  duplicate the regex validation.
- **#163** — `status --boot` crashing on an inline-comment value line was a Python-`ValueError` and
  is gone in the Go rewrite; this entity only needs to **verify** the Go parser correctly strips an
  inline `key: value  # comment` rather than re-assert the crash fix.

## Ground truth (read the code, 2026-05-31)

All four items were verified against `internal/status/` (the native Go path the FO runs) and the
vendored Python oracle (`skills/commission/bin/status`, embedded as `internal/status/vendor/status`).
Findings reshape three of the four ACs from what the seed assumed:

- **#230 is NOT a parity bug and NOT a hidden-entity bug.** `computeDispatchable`
  (`internal/status/format.go:151`) is a faithful, byte-parity port of the oracle's candidate loop
  (`skills/commission/bin/status:875-910`). The *only* state signal either reads is frontmatter
  (`status`, `worktree`) plus the README stages block. **Neither implementation has any notion of "a
  stage report is committed"** — there is no git-log inspection and no entity-body parse for
  `## Stage Report`. An entity sitting at a non-terminal, non-gate stage with an empty `worktree`
  field IS already surfaced by `--next`. So a "committed-report-but-unadvanced" entity is suppressed
  only by one of three deliberate, correct rules: (a) its `worktree` field is still set (FO has not
  torn down the worker's worktree), (b) the next stage is at `concurrency`, or (c) it sits at a
  `gate`. The issue author's own words: *"technically correct from the state machine's
  perspective."* The proposal asks for **visibility** (a `--ready-to-advance` view or a
  `report-present`/`advanced` column), not a change to `--next` semantics.
- **#163's crash is gone, but the Go parser SILENTLY MISPARSES instead of stripping.** Verified
  empirically with throwaway tests:
  - Entity frontmatter `status: plan  # note` → `ParseFrontmatter` yields the literal
    `"plan  # note"` (comment NOT stripped). The seed's AC-4 claim that the Go parser already strips
    is **false**; a verify-only AC-4 would fail today.
  - Stage frontmatter `concurrency: 5  # debate` → `ParseStagesWithDefaults` yields `2` (silent
    fallback to default via `atoiOr`, `internal/status/stages.go:201`), and `worktree: true  # iso`
    yields `false`. The Python `ValueError` is gone (Go does not crash), but the value is wrong and
    undetectable — arguably worse than the crash. #163's real surface is the **stage** numeric/bool
    fields, not just an entity value line.
- **#189 has no membership check at all.** `updateFrontmatter` (`internal/status/mutate.go:38`)
  writes any `status=X` verbatim; `validateWorkflowStageNames` (`validate.go:110`) only checks the
  README's stage-name *format regex*, never `--set` value membership. The seed's scope note is
  correct — these are orthogonal.
- **#207 is the enumeration-source asymmetry, exactly as reported.** Default reads scan only
  top-level (`scanEntitiesActive`); `_archive/` entities appear only under `--archived`
  (`handlers.go:251-255`). A top-level entity whose frontmatter says `status: archive` is still
  enumerated because placement, not the `status` value, decides visibility.

**Parity is a hard constraint on three of four items.** Enumeration and `--set` are pinned by the
differential parity suite (`internal/status/zz_independent_parity_test.go`, `archive_guard_test.go`)
which asserts the Go launcher matches the oracle byte-for-byte. Any behavior change to enumeration
(#207) or `--set` (#189) must EITHER be made in both the oracle and the Go port, OR be an
*additive* surface that does not alter an existing parity-pinned path. The recommendations below are
chosen to stay additive where possible.

## #230 RECOMMENDATION (gates implementation — FO surfaces at the gate)

**Recommend: the committed-report-but-unadvanced entity is NOT a `--next` bug; the FO/ensign
advance-after-report contract is the load-bearing fix, and the visibility gap is an ADDITIVE
read surface — not a change to `--next`.** Rationale:

1. `--next` correctly answers *"what may I dispatch right now without violating concurrency, gates,
   or in-flight worktrees?"* Surfacing a concurrency-saturated or still-worktree'd entity there
   would make `--next` lie — the FO would dispatch and overrun the limit.
2. The actual defect the issue describes (a worker commits its report, is shut down, and the entity
   lingers) is a **contract gap**: the ensign signals completion and the FO is supposed to advance
   `status` and tear down the worktree. When that hand-off is dropped, the entity goes invisible to
   `--next` because `worktree` is still set. The root cause is the missing advance, not the
   enumerator. The clean fix is to make "advance-after-report" an enforced step, surfaced as the
   queue-depth view below — NOT to teach `--next` to ignore the worktree field.
3. The issue's own preferred remedy is a **separate** surface (`--ready-to-advance` or a column),
   which is additive and parity-safe.

**Decision the captain must ratify at the gate:** ship the visibility surface as a non-`--next`
read (recommended: a `--where`-style filter or a boot-section annotation rather than a brand-new
flag, to minimize new CLI surface), and tighten the advance-after-report contract — OR judge the
contract tightening sufficient on its own and drop the read surface as YAGNI. The recommendation is
the former; the captain decides scope.

## Acceptance criteria

**AC-1 — A `--next`-suppressed entity's reason is observable; advance-after-report is the load-bearing
contract.** Behavioral oracle: given a fixture workflow with an entity at a non-terminal, non-gate
stage, (i) with `worktree` empty, `status --next` surfaces it as dispatchable to the next stage;
(ii) with `worktree` set, `--next` suppresses it AND the visibility surface (per the ratified #230
decision) reports it as "ready pending advance / worktree teardown"; (iii) with the next stage at
`concurrency`, `--next` suppresses it and the surface attributes the suppression to concurrency.
The three suppression reasons (worktree-set, concurrency-full, gate) are distinguishable in the
visibility surface. If the captain drops the read surface at the gate, AC-1 reduces to a contract
test that advance-after-report is enforced and a documented-behavior note that `--next` suppression
is by-design.

**AC-2 — Enumeration source is consistent; placement does not change visibility for a given scope.**
Behavioral oracle: a fixture with two equivalent entities, one at top-level `<slug>.md` and one at
`_archive/<slug>.md`, both with the same frontmatter. The default (active) read and the `--archived`
read each enumerate by SCOPE consistently: an `_archive/`-placed entity appears in the archived
scope, a top-level entity appears in the active scope, and the chosen consistency rule is asserted
identically for both placements. The fix must declare and test ONE rule (recommended: `_archive/`
is the authoritative archived-scope source and top-level placement reflects active scope regardless
of the `status` frontmatter value), and the rule must hold under the parity suite — i.e. the oracle
exhibits the same enumeration or the divergence is documented as an intentional, oracle-mirrored
change.

**AC-3 — `status --set status=X` rejects X not in the workflow's declared `stages.states[].name`.**
Behavioral oracle: on a fixture workflow declaring stages `[a, b, c]`, `--set {slug} status=zzz`
exits non-zero with an actionable error naming the unknown value and listing known stages (e.g.
`error: 'zzz' is not a defined stage in workflow <wd> — known stages: [a, b, c]`), and the entity
frontmatter is UNCHANGED; `--set {slug} status=b` (a member) exits zero and mutates. This is the
membership check, distinct from `validateWorkflowStageNames`' format regex. Must be applied in BOTH
the oracle and the Go `runSet` to preserve parity (the parity suite runs `--set` differentially).
A `--force` bypass is in scope only if the captain wants mid-flight stage renames supported;
recommend deferring `--force` as YAGNI unless asked.

**AC-4 — Inline comments are stripped from frontmatter values AND from stage numeric/bool fields.**
Behavioral oracle: (i) `ParseFrontmatter` on `key: value  # comment` yields `value` (not
`value  # comment`); (ii) `ParseStagesWithDefaults` on `concurrency: 5  # debate` yields `5` (not
the default 2) and `worktree: true  # iso` yields `true`. This is a real fix, not a verification —
the seed's "already gone" assumption is wrong (see Ground truth). The strip must match the oracle:
either patch both the Python oracle and the Go parser to strip `# …` (preserving parity), or scope
AC-4 to the Go side only with a documented, oracle-mirrored divergence. **Scope-overlap flag:** the
`yaml-parser-migration` entity (`zjmjzznydmqr58bd46qz6q07`) replaces the hand-rolled parser with
`yaml.v3`, which strips inline comments natively and would subsume AC-4 — but that entity is GATED
on the Python oracle being fully retired (parity certified + `claude-runtime-segregation` landed +
VendorRunner retired), a long pole. Recommend fixing AC-4 now in the hand-rolled parser (small,
parity-paired) rather than waiting on the migration; note in the migration entity that AC-4's tests
become coverage it must keep green.

## #230 shipped behavior (documented-behavior note)

Per the captain-ratified scope, the #230 code deliverable in this entity is the ADDITIVE visibility
surface ONLY — `--next` / `computeDispatchable` semantics are unchanged. A `--next` suppression is
now observable via a computed field `next-suppressed-by` surfaced through the existing
`--fields` / `--where` machinery (no new flag): `status --fields next-suppressed-by` shows, per
entity, why `--next` held it, and `status --where 'next-suppressed-by = concurrency-full'` filters
on it. Values mirror the dispatch loop exactly: `worktree-set` (worktree not torn down),
`concurrency-full` (next stage saturated), `gate` (sits at a gate), `terminal`, or `""`
(dispatchable / not attributable). The field is COMPUTED, not frontmatter: it is materialized only
when explicitly named and is excluded from `--all-fields` so that parity-pinned surface stays
byte-identical. Both the Go native path and the Python oracle compute it identically (parity-green).

`--next` suppression remains BY DESIGN: an entity whose report is committed but whose `worktree` is
still set is correctly held from `--next` (dispatching it would overrun the worktree/concurrency
contract). The advance-after-report ENFORCEMENT (the FO advancing `status` and tearing down the
worktree after an ensign signals completion) is a prose FO/ensign-contract matter tracked as a
SEPARATE follow-up — it is NOT a status-binary change and was intentionally NOT made in this entity.
The surface above makes a dropped advance-after-report hand-off observable (the lingering entity
shows `next-suppressed-by: worktree-set`) so the contract gap is diagnosable from `status` output.

## Test plan

Go unit + differential-parity tests in `internal/status/`, sized to the claim:

- **AC-4** (cheapest, do first — it is the riskiest contract: a silent-misparse): parser unit tests
  on `ParseFrontmatter` and `ParseStagesWithDefaults` for the inline-comment cases above. If the
  oracle is patched too, add the `# …` strip to `parse_frontmatter`/`parse_stages_block` and keep
  the existing differential tests green. Minutes of cost; validates the parser contract before any
  enumeration/`--set` work builds on it.
- **AC-3**: a differential `--set` test on a fixture workflow — non-member rejected (exit non-zero,
  unchanged frontmatter, actionable stderr), member accepted (exit zero, mutated). Mirror the
  `archive_guard_test.go` launcher-vs-oracle idiom so parity is asserted, not just Go behavior.
- **AC-2**: a behavioral enumeration test with the two-placement fixture, asserting the chosen
  scope-consistency rule under both the default and `--archived` reads, launcher-vs-oracle.
- **AC-1**: a fixture-driven `--next` + visibility test exercising the three suppression reasons,
  shaped by whichever #230 scope the captain ratifies at the gate. If the read surface is dropped,
  this becomes the advance-after-report contract test plus a documented-behavior assertion.

No live workflow tests are needed — every claim is a parser/command/enumeration behavior provable
with Go unit and golden/differential fixtures. Estimated cost: low-to-moderate (a day of focused
implementation across the four items, dominated by the parity-paired #207/#189 changes).

## Sequencing & staff review

**This entity is on the serialized `internal/status` lane.** Its implementation touches
`internal/status/{handlers.go, validate.go, discover.go}` plus `format.go`, `mutate.go`, and
`stages.go`. The `architecture-review-cleanups` entity (`0xcqyh24hr5xnek3kfp8makg`) touches the same
package and its own body states it "sequences with the other internal/status entities (after
packaging; can fold into the implementation drain)." Per the collision analysis this entity MUST
sequence **after** the `claude-runtime-segregation` (zs) merges and **after**
`architecture-review-cleanups` lands, to avoid concurrent edits to the same serialized lane.

**Parity coupling raises the bar:** because #207 and #189 (and optionally #163) require paired
oracle + Go edits to preserve the differential parity suite, the implementation is more delicate than
a single-sided Go change. Combined with the live #230 semantic decision (a genuine behavior choice,
not a mechanical fix) and the scope-overlap with `yaml-parser-migration`, **staff review IS
warranted** — this matches the stage definition's "native status parity" trigger. Recommend the FO
request an independent design review before the ideation gate, focused on: (1) the #230 scope call
(additive read surface vs contract-only), (2) the #207 single-rule choice and whether the oracle
must mirror it, and (3) confirming AC-4 is fixed here rather than deferred to the gated migration.

## Stage Report: ideation

- DONE: RESOLVE #230 with a recommendation
  Recommended (in body, "#230 RECOMMENDATION"): committed-report-but-unadvanced is NOT a `--next` bug; `computeDispatchable` (format.go:151) faithfully ports the oracle (status:875-910) and neither has any "report committed" signal. The load-bearing fix is the advance-after-report contract; the visibility gap is an ADDITIVE read surface, never a change to `--next`. Captain ratifies scope (additive surface vs contract-only) at the gate.
- DONE: Specify falsifiable behavioral oracles in internal/status for each cluster item
  Each of AC-1..AC-4 now carries a concrete behavioral oracle (#230 three-suppression-reason fixture; #207 two-placement scope-consistency; #189 member/non-member `--set` exit+unchanged-frontmatter; #163 entity-value AND stage numeric/bool inline-comment strip).
- DONE: Reconcile against the vendored Python oracle (enumeration/--next parity); FLAG serialized-lane sequencing; state whether staff review is warranted
  Body "Ground truth" + "Sequencing & staff review": #230/#207/#189 reconciled against `skills/commission/bin/status`; parity is a hard constraint (zz_independent_parity_test.go) forcing paired oracle+Go edits for #207/#189/#163. Flagged: must sequence AFTER zs merges and AFTER architecture-review-cleanups (same internal/status lane). Staff review IS warranted (native-status-parity trigger + live #230 semantic call + yaml-parser-migration scope overlap).

### Summary

Read the actual `internal/status` Go path and the vendored Python oracle and ran throwaway probe tests, which overturned the seed's assumptions on two items. #230 is not a parity/hidden-entity bug — both implementations are identical and correct by design; the real fix is the advance-after-report contract plus an additive visibility surface (captain ratifies scope). #163/AC-4 is a real bug, not a verification: the Go parser does NOT strip inline comments — it returns `"plan  # note"` verbatim for entity values and silently falls back to defaults for stage numeric/bool fields (worse than the gone Python crash). #189 has no membership check at all. #207 is the enumeration-source asymmetry as reported. Flagged the hard parity constraint (paired oracle+Go edits for #207/#189/#163), the serialized-internal/status-lane sequencing (after zs + architecture-review-cleanups), the scope overlap with the gated yaml-parser-migration entity, and recommended staff review before the gate.

## Stage Report: implementation

- DONE: AC-3 (#189) --set status= membership check, parity-paired Go + oracle
  runSet + oracle (source+vendored) reject a non-member status with "Error: 'X' is not a defined stage in workflow <wd> — known stages: [...]", exit 1, frontmatter unchanged; member accepted. Differential test set_stage_membership_test.go launcher-vs-oracle. Commit 91481106.
- DONE: AC-2 (#207) enumeration-scope-by-placement rule declared + parity-pinned
  Ground truth: the Go native path ALREADY mirrors the oracle — placement (not the status value) decides scope (top-level=>active, _archive/=>archived; --archived appends archived to active). No behavior change was needed; the gap was an undeclared/unlocked rule. Added characterization test enum_scope_test.go (native-vs-oracle, both placements) on new two-placement fixture enum-scope-workflow with identical status, README prose declaring the one rule. Commit 614d9156.
- DONE: AC-4 (#163) STAGE numeric/bool inline-comment strip, parity-paired Go + oracle
  Real silent-misparse fixed: concurrency:5 # debate fell back to default 2 via atoiOr; worktree:true # iso became false. Added stripInlineComment (whitespace-preceded # = comment; unspaced #163 token kept) applied to every stage field value in both Go parse paths and both oracle functions (parse_stages_block + parse_stages_with_defaults), source+vendored. Parity-pinned: stages_comment_test.go unit + stages_comment_parity_test.go --next dispatch differential (stripped concurrency=1 caps dispatch at 1, not 2). Commit 04c1e950.
- DONE: FOLD-IN — trim stale runArchive doc-comment at mutate.go
  Removed the reverted both-dirs-union phrase ("plus entityDir/_mods/ during a split-root mod migration"); scanMods reads definitionDir/_mods only. Commit 614d9156.
- FAILED: AC-4 part (ii) GENERIC ParseFrontmatter entity-value inline-comment strip — BLOCKED on scope ruling
  A blanket whitespace-preceded-# strip on ALL frontmatter values truncates real existing free-text fields: source/title containing ` #NNN` (e.g. "consolidates #223, #217" → "consolidates"). Verified empirically across the state checkout. Escalated A (narrow/typed-only, my recommendation) vs B/C (blanket) to team-lead; no ruling received. The stage-field half (the real #163 surface) is done; only the generic entity-value strip remains, pending the scope decision.
- FAILED: AC-1 (#230) additive visibility surface + advance-after-report contract — BLOCKED on scope ruling
  Proposed a derived `next-suppressed-by` field (worktree-set/concurrency-full/gate/terminal) surfaced via existing --where/--fields (satisfies "--where-style, not a new flag"); awaiting team-lead ruling on (a) field name + materialization (show under --all-fields vs gated to keep --all-fields byte-identical) and (b) whether the advance-after-report contract — which lives in skills/first-officer/references/first-officer-shared-core.md (FO prose, a different skill's reference), not the status binary — is in-scope for me to edit or reduces to a documented-by-design note. Not guessing on a parity-pinned read surface or a cross-skill behavioral contract.

### Summary

Landed three of the four ACs and the fold-in, all parity-paired Go+oracle where the surface is parity-pinned, internal/status fully green (279 tests; the lone full-run cli failure is the pre-existing env-only TestCodexResolveManifestAgainstInstalledHost the dispatch flagged as not-a-regression). #207 needed no behavior change — the Go path already matched the oracle; the deliverable was declaring + locking the single placement-based scope rule. #189 and the #163 stage-field strip are real fixes. Two items remain BLOCKED on genuine design rulings I escalated and did not get: AC-4's generic entity-value strip (blanket strip truncates real source/title `#NNN` tokens — A vs B/C) and AC-1's #230 surface (field shape/materialization + whether the FO advance-after-report prose contract is mine to edit). Both touch parity-pinned/cross-skill contract surfaces, so I held rather than guess.

## Stage Report: implementation (cycle 2)

- DONE: AC-1 (#230) additive next-suppressed-by visibility surface, parity-paired Go + oracle
  Per team-lead ruling: field `next-suppressed-by` ∈ {""|worktree-set|concurrency-full|gate|terminal}, surfaced via the EXISTING --fields/--where machinery (no new flag), GATED out of --all-fields (materialization option ii). Extracted dispatchAnalysis (shared with computeDispatchable) so the reason mirrors --next exactly; --next/computeDispatchable output unchanged. Mirrored in the oracle (compute_suppressed_by/materialize_suppressed_by, source+vendored). Behavioral test next_suppressed_by_test.go distinguishes all three reasons + the "" dispatchable case, native-vs-oracle, and asserts --all-fields excludes the computed column. Fixture suppress-workflow. Commit 21838fa7.
- DONE: AC-1 documented-behavior note in the entity body
  Added "## #230 shipped behavior" note: the surface is the code deliverable; --next suppression is by-design; advance-after-report ENFORCEMENT is a separate FO/ensign prose-contract follow-up (NOT a status-binary change, intentionally not made here per the ruling); a dropped hand-off is now diagnosable as next-suppressed-by: worktree-set. State-checkout commit a231552.
- DONE: AC-2 (#207) — see cycle-1 report (no change)
- DONE: AC-3 (#189) — see cycle-1 report (no change)
- DONE: AC-4 (#163) stage-field strip — see cycle-1 report (no change)
- FAILED: AC-4 part (ii) GENERIC ParseFrontmatter entity-value strip — STILL ON HOLD per team-lead
  Team-lead relayed the captain chose option (C): blanket strip per the YAML whitespace-# rule + require quoting + the WRITER must auto-quote values containing ` #` so they round-trip (no silent re-truncate on --set). Team-lead is confirming one more disposition with the captain — whether to ALSO migrate-and-quote existing #-bearing entities in the state checkout (to avoid truncating live source/title on first read). Explicit instruction: do NOT implement AC-4 until the final instruction (including the migration disposition) is relayed. Holding as directed.

### Summary

AC-1 (#230) landed this cycle exactly to the ratified shape: an additive, parity-paired next-suppressed-by surface over the existing --fields/--where machinery, gated out of --all-fields, with --next semantics untouched (dispatchAnalysis is shared so the reason can't drift from dispatch). The advance-after-report enforcement was correctly left as a separate prose-contract follow-up per the ruling and recorded as a documented-behavior note. internal/status fully green (285). Three of four ACs + AC-4's stage-field half + the fold-in are now complete and committed. The ONLY remaining work is AC-4's generic entity-value strip (captain chose option C: blanket + auto-quote-on-write), which team-lead explicitly instructed me to hold until the existing-entity migration disposition is finalized. I am holding as directed, not blocked on my own analysis.

## Stage Report: implementation (cycle 3)

All four ACs are now complete. This cycle clears the previously-held AC-4(ii).

- DONE: AC-1 (#230) additive next-suppressed-by visibility surface
  Cycle 2 (commit 21838fa7). next-suppressed-by ∈ {""|worktree-set|concurrency-full|gate|terminal} via --fields/--where, gated out of --all-fields, --next unchanged (shared dispatchAnalysis), oracle-mirrored. Test next_suppressed_by_test.go distinguishes all three reasons + "" dispatchable, native-vs-oracle. Documented-by-design note in body. AC-1(b) advance-after-report left to FO/ensign prose follow-up per ruling (not touched).
- DONE: AC-2 (#207) enumeration-scope-by-placement rule declared + parity-pinned
  Cycle 1 (commit 614d9156). Go already mirrored the oracle; placement decides scope. Characterization test enum_scope_test.go (native-vs-oracle, both placements) + README prose.
- DONE: AC-3 (#189) --set status= membership check, parity-paired Go+oracle
  Cycle 1 (commit 91481106). Non-member exits 1 with actionable known-stages list, frontmatter unchanged; member accepted. Differential test set_stage_membership_test.go.
- DONE: AC-4 (#163) inline-comment strip — BOTH halves now landed, parity-paired Go+oracle
  Stage numeric/bool fields: cycle 1 (commit 04c1e950), stripInlineComment in both parse paths, --next dispatch differential. Generic entity values (part ii, option C, cycle 3 commit 89af53b9): parse_value strips an inline comment from ALL values (whitespace-preceded # = comment; unspaced token kept) and protects an interior # inside a quoted scalar; quote_for_write quotes a written value containing ` #` so it round-trips (no re-truncate on --set); values without ` #` byte-preserved. Mirrored in the oracle. Differential round-trip test comment_roundtrip_parity_test.go: a #-bearing --set value is written quoted and reads back whole, byte-identical launcher-vs-oracle. Existing UNQUOTED #-bearing values truncate on read — accepted option-C behavior; NO migration/back-compat in the binary (the one-off repo state migration is the FO's job per team-lead, NOT touched here).
- DONE: FOLD-IN — stale runArchive doc-comment trimmed
  Cycle 1 (commit 614d9156).

### Summary

All four consolidated issues are resolved and parity-paired Go+oracle where the surface is parity-pinned. internal/status fully green (295 tests); native-vs-oracle default and --all-fields reads byte-identical; the only full-run cli failure remains the pre-existing env-only TestCodexResolveManifestAgainstInstalledHost the dispatch flagged as not-a-regression. AC-4(ii) shipped per the captain's ratified option C: blanket reader-strip + writer-quote + accept-truncation of legacy unquoted #-bearing values, with the live state-checkout quote-migration explicitly left to the FO (not done here, state checkout untouched except this report). Five code commits on branch spacedock-ensign/status-enumeration-and-validation: 91481106, 614d9156, 04c1e950, 21838fa7, 89af53b9. Expecting the detached adversarial audit at validation (internal/status read/scheduling + parity + merge-guard-adjacent surface).

## Stage Report: validation

- DONE: INDEPENDENTLY reproduce all four ACs on the branch binary (build from the worktree HEAD; do NOT trust the report)
  Built branch binary from worktree HEAD 89af53b9 (`go build -o /tmp/spacedock-validate-bin`, go test ./internal/status/... = 295 passed). #189: `--set ent status=zzz` exits 1 with `Error: 'zzz' is not a defined stage in workflow ... — known stages: [a, b, c]`, frontmatter UNCHANGED; `status=b` (member) exits 0, mutates a->b. #207: enum-scope-workflow (identical `status: backlog` both placements) — active read shows only top-placed, `--archived` shows arch-placed; placement (not status value) decides scope. #163/AC-4: stage `concurrency: 5  # debate` strips to 5 (5 dispatched, not default-2) and `worktree: true  # iso` strips to true (WORKTREE: yes); generic value `realvalue  # comment`->`realvalue`, `v1.0#163` KEPT, quoted `"keep #interior hash"` interior # kept, legacy unquoted `free text #230` truncates to `free text`. #230: suppress-workflow three reasons distinguishable (waiting=concurrency-full, building=worktree-set, gated=gate); `--all-fields` excludes the column (native==oracle IDENTICAL); `--next`/computeDispatchable byte-identical native-vs-oracle.
- DONE: PARITY is the hard contract — differential suite byte-green native-vs-oracle; TAMPER-CHECK confirms tests genuinely guard parity
  Full named differential suite (zz_independent_parity, archive_guard, set_stage_membership, enum_scope, next_suppressed_by, stages_comment_parity, comment_roundtrip_parity, frontmatter_comment, mutate_quote) = 50 passed, all explicit per-test PASS, no SKIP. TAMPER 1: neutered the vendored oracle membership guard (`and False`) -> TestSetStatusNonMemberRejected FLIPS RED (`launcher exit=0, want 1`); restored, green. TAMPER 2: neutered vendored oracle `quote_for_write` (return val) -> TestCommentValueRoundTripParity FLIPS RED (`expected quoted #-bearing value on disk, got: id: "002"`); restored, 295 green, working tree clean.
- DONE: AC-4(ii) accept-truncation + writer round-trip; NO migration/back-compat; live state checkout NOT touched
  Round-trip: `--set source="consolidates #163 and #207"` writes QUOTED on disk (`source: "consolidates #163 and #207"`) and reads back WHOLE. Legacy unquoted `#`-bearing value truncates on read (accepted option-C). `git diff origin/next...HEAD` on changed Go files: no migrat/back-compat/legacy/fallback added. `docs/dev/.spacedock-state` untouched by the code branch (empty diff).
- DONE: Scope-note checks (a) by-design, (b) cli non-regression, (c) parity table-form
  (a) #230 contract tightening NOT a code/skill edit — `## #230 shipped behavior` by-design note present (body lines 147-166); only `skills/` change is the oracle binary `skills/commission/bin/status`, NO first-officer/ensign contract prose touched. (b) internal/cli byte-identical to origin/next (empty diff) — TestCodexResolveManifestAgainstInstalledHost is the documented env-only non-regression, NOT a finding. (c) parity asserted on TABLE form; `--json` native-only (oracle does not implement) as expected.
- DONE: REJECT-criteria checks — tests prove the CURRENT intended behavior, not an obsolete/over-specified target
  AC-4 does NOT over-strip: unspaced `#163` token and interior `#` in a quoted scalar are both KEPT (verified). The #230 surface does NOT drift from `--next`: `dispatchAnalysis` is the single shared function feeding both computeDispatchable and the surface; `--next` byte-identical native-vs-oracle. No stale/wrong-abstraction target found.

### Summary

VERDICT: PASSED. All four ACs independently reproduced on the freshly-built branch binary (HEAD 89af53b9), not trusted from the report. The differential parity suite is byte-green native-vs-oracle (50 named differential tests, 295 total in internal/status), and two independent tamper-checks (oracle membership guard; oracle quote_for_write) both FLIP the relevant differential RED then restore green — proving the tests genuinely guard parity rather than pass by construction. AC-4 strip is correctly scoped (spaced-# stripped; unspaced token and quoted-interior # kept; legacy unquoted truncation accepted; writer quotes #-bearing values for round-trip). No migration/back-compat code was added and the live state checkout was untouched. internal/cli is byte-identical to origin/next, so the lone env-only TestCodexResolveManifestAgainstInstalledHost is confirmed not a regression. The #230 advance-after-report contract tightening is correctly a documented-by-design note (no contract/skill prose edit), per the FO/captain ruling. One non-finding noted for the FO: a `--fields` invocation that names a field already in the default set (e.g. `--fields id,status`) shows a PRE-EXISTING native-vs-oracle divergence (oracle duplicates the column) independent of this branch; the new `next-suppressed-by` surface is byte-identical native-vs-oracle when named without such a collision (as the parity test does).
