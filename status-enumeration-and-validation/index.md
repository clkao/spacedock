---
id: s0cbyrbh9b2yb456exnsm72b
title: status correctness — --next undercount, archive enumeration consistency, --set stage-name validation
status: implementation
source: issue sweep (2026-05-31) — CL dev-workflow-ergonomics triage; consolidates #230, #207, #189, #163
started: 2026-06-01T05:04:22Z
completed:
verdict:
score: "0.28"
worktree: .worktrees/spacedock-ensign-status-enumeration-and-validation
issue:
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
