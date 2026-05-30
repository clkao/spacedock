---
id: p31ep68fer46hhms1pvp3b6f
title: Check external tracker integration point
status: implementation
score: "0.25"
source: bootstrap roadmap
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-external-tracker-checkpoint
started: 2026-05-30T04:30:16Z
---

# Check External Tracker Integration Point

Evaluate whether the split-root design gives clean integration points for kata,
Linear-style tickets, and other external trackers, and decide whether any
bidirectional sync belongs in v1. This is a **design checkpoint**: the deliverable
is a decision plus a preserve-through-lifecycle contract, not a new feature. The
mechanism it depends on — frontmatter round-trip through the native parser and
mutator — is owned by sibling entities `native-go-status` and `native-state-dir`.

## Problem statement

Teams that run Spacedock alongside an external ticket ledger (kata, Linear, GitHub
Issues) need to carry the external reference into the Spacedock entity and keep it
intact for the whole execution lifecycle. The state-behavior extension
(`docs/specs/state-behavior-extension.md` -> **External Tracker Integration**)
already names the bridge shape: two flat top-level frontmatter fields,
`issue` (the human-facing external reference) and `source` (where the entity came
from). What is *not* yet settled, and what this checkpoint must settle, is two
things:

1. **Preservation.** The v1 frontmatter contract is line-oriented and the native
   parser must match it. The native mutator (`--set`, timestamp fill, `--archive`)
   re-serializes an entity's frontmatter on every write. The sharp risk is a
   mutator that re-emits only the fields it *recognizes* and silently drops
   unknown fields. `issue` and `source` are unknown to status semantics, so unless
   the mutator preserves unknown fields verbatim, the external reference is lost on
   the first `--set` or `--archive`. The preserve-through-lifecycle property is
   therefore an **unknown-field round-trip property of the mutator**, exercised
   end-to-end through status read, `--set`, report append, and `--archive`.

2. **Sync ownership.** Whether v1 ships any bidirectional sync (Spacedock writing
   status back into the external tracker, or the tracker driving Spacedock stage
   transitions) or whether all sync stays in an out-of-process adapter.

### Assumptions (flagged for FO reconciliation at the gate)

These depend on sibling entities not yet finalized; the FO should reconcile them at
the ideation gate.

- **A1 (native-go-status).** The native frontmatter parser preserves field order and
  carries fields it does not interpret. If `native-go-status` instead models a
  fixed struct of known fields and re-emits only those, `issue`/`source` are dropped
  and this checkpoint's AC fails — this checkpoint then becomes a *requirement on*
  `native-go-status`, not an independent design. Stated so the FO can fold the
  unknown-field-preservation requirement into that entity's AC if needed.
- **A2 (native-state-dir).** `--set` and `--archive` mutate only files under
  `.spacedock-state` (and `_archive` within it). The preservation AC is verified in
  split-root mode, so it inherits state-dir write scoping from `native-state-dir`.
- **A3 (folder-form entities).** Folder-form entities with `reports/` and
  `artifacts/` are not misdetected as separate entities (roadmap stage 3 / 5),
  so an entity carrying `issue`/`source` plus a `reports/` dir is one entity, and
  its reference survives the report-append step.

## Proposed approach

Adopt the two-field flat bridge already drafted in the state-behavior extension and
add a single binding contract on top of it: **the mutator preserves unknown
frontmatter fields verbatim through every write.** No new fields, no parser changes
beyond what `native-go-status` already owns, no tracker-specific code.

Lifecycle path the reference must survive (the "checkpoint"):

```text
import  ->  status read  ->  --set status=<stage>  ->  stage report append  ->  --archive
(issue/source set)   (rendered, not stripped)   (frontmatter rewrite)   (body grows)   (file moves under _archive)
```

`issue` and `source` are treated exactly like any other flat custom field: the
parser carries them, status rendering may surface them (e.g. under `--fields` /
`--all-fields`) but does not require them, and every mutating write copies them
through unchanged. Archive is a file *move* plus an `archived:` timestamp write, so
the same unknown-field-preservation rule covers it.

### v1 DECISION: no bidirectional sync in v1; sync stays an external adapter

**Decision: v1 ships the inbound link fields (`issue`, `source`) and the
preserve-through-lifecycle guarantee only. No bidirectional sync, no writeback, no
tracker-driven stage transitions ship in v1. All richer sync lives in an
out-of-process adapter until a bridge contract is written.**

Justification against the roadmap out-of-scope note ("external tracker writeback
beyond simple entity link fields is out of scope"):

- The roadmap explicitly scopes *out* writeback beyond simple link fields. Shipping
  any bidirectional sync in v1 would contradict that line directly.
- `AGENTS.md` is compatibility-first and stdlib-only and forbids new PR/mod
  behavior in bootstrap. A tracker writeback path implies network I/O, per-tracker
  auth, and a sync state machine — none of which belong in a status launcher.
- The state-behavior extension already states the rule: "Spacedock status remains
  the execution status unless a future bridge explicitly declares bidirectional
  ownership," and "Bridge tools should sync through entity creation, state changes,
  and stage reports rather than by adding tracker-specific stage rules." An adapter
  reading Spacedock entities + status output and pushing to the tracker satisfies
  every named integration need without a writeback contract.
- **Execution-status ownership: Spacedock owns execution status.** The external
  tracker may own backlog intake, discussion, assignment, and reporting. The
  `status` frontmatter field (stage) is owned by Spacedock; the adapter mirrors it
  outward read-only. This is the line a future bridge would have to renegotiate to
  claim bidirectional ownership — and v1 deliberately does not.

The v1 surface is thus *enabling, not integrating*: by guaranteeing the two fields
survive the lifecycle, v1 makes an adapter possible (it can always recover the
external reference from any entity, active or archived) without making any tracker a
required backend.

## Acceptance criteria

Each AC names a property of the finished checkpoint, not a stage action, with how it
is verified. The deliverable is a decision/spec plus fixture-level proof of the
preservation property; there is no tracker-specific product code.

**AC-1 - An entity carrying flat `issue` and `source` frontmatter fields round-trips through status read unchanged: the fields are parsed, not stripped, and not required for rendering.**
Verified by: a Go parser/status test on a fixture entity with `issue: ENG-123` /
`source: linear` asserts both fields are present in the parsed entity and that
default status output renders the entity normally (no error, no requirement that
the fields exist on other entities). Same-level proof: parser unit test +
golden/CLI status read.

**AC-2 - `--set` preserves `issue` and `source` verbatim while mutating an unrelated field.**
Verified by: a mutation test runs `--set <slug> status=implementation` on the
`ENG-123`/`linear` fixture and asserts the post-write frontmatter still contains
`issue: ENG-123` and `source: linear` byte-for-byte, alongside the changed
`status`. This is the unknown-field round-trip property of the mutator.

**AC-3 - Appending a stage report to a folder-form entity does not disturb its `issue`/`source` frontmatter, and the entity is not misdetected as multiple entities.**
Verified by: a fixture folder-form entity (`index.md` + `reports/`) carrying
`issue`/`source` is read after a report append; status lists it as exactly one
entity and its frontmatter reference is intact. Proof: discovery + status read test
over the folder-form fixture.

**AC-4 - `--archive` moves the entity under `.spacedock-state/_archive` with `issue`/`source` preserved through the move and the `archived:` timestamp write.**
Verified by: a mutation test runs `--archive <slug>` on a `kata:task-abc123`/`kata`
fixture and asserts the archived file under `_archive` still contains
`issue: kata:task-abc123` and `source: kata`, plus the new `archived:` field. The
move writes only under the state checkout (inherits A2 scoping).

**AC-5 - No tracker-specific stage semantics, fields, or code paths are introduced; `issue`/`source` are handled exactly as flat custom fields.**
Verified by: design review of this entity + the state-behavior extension shows no
`if source == "kata"`-style branching, no tracker-named stages, and no parser
special-case for `issue`/`source`; the only requirement added is generic
unknown-field preservation. Proof: static review against the spec; absence of
tracker names in command/parser code paths.

**AC-6 - The v1 decision — no bidirectional sync in v1; sync remains an external adapter; Spacedock owns execution status — is recorded with justification against the roadmap out-of-scope note.**
Verified by: this entity's **v1 DECISION** section states the decision and cites the
roadmap out-of-scope line and the state-behavior extension's ownership principle;
documentation states richer tracker sync belongs in an adapter until a bridge
contract exists. Proof: static prose review of the decision section.

## Test plan

The claim is a preservation *property*, so the proof level is Go parser/mutation
unit tests plus status golden/CLI reads over small fixtures — the same level and
fixtures the sibling `native-go-status` and `native-state-dir` entities already
stand up. No live workflow run, no network, no tracker is needed.

| Check | Verifies | Level / cost |
|-------|----------|--------------|
| Parser test: fixture with `issue`/`source` parsed and carried | AC-1 | Go unit, trivial |
| Status read over `issue`/`source` fixture renders normally | AC-1 | Go/CLI golden, trivial |
| `--set status=implementation` keeps `issue: ENG-123` / `source: linear` | AC-2 | Go mutation test, trivial |
| Report append over folder-form `issue`/`source` entity; discovery = 1 entity, reference intact | AC-3 | Go discovery+read test, trivial |
| `--archive` moves `kata:task-abc123` fixture under `_archive` with `issue`/`source` + `archived:` intact | AC-4 | Go mutation test, trivial |
| Static review: no tracker-specific branching/fields/stages | AC-5 | Static review, trivial |
| Decision section cites roadmap out-of-scope + adapter rule | AC-6 | Static prose review, trivial |
| `go test ./...` | baseline gate (AGENTS.md) | suite, seconds |

Cost/complexity: trivial. These tests are **most cheaply realized as fixtures /
assertions folded into the `native-go-status` and `native-state-dir` mutation
suites** (one shared "unknown fields survive every write" fixture exercised by their
existing `--set`/`--archive` tests), not as a standalone tracker test package —
keeping with YAGNI and avoiding a tracker-shaped surface that the decision says we
should not build. If the FO prefers, this checkpoint can ship as the design/decision
record alone and delegate the assertion lines to those sibling entities; that
dependency is flagged as A1/A2 above.

## Notes

This checkpoint should happen after native split-root status works. It is not a
blocker for the symlink compatibility phase.

The single load-bearing technical risk is A1: if `native-go-status` models
frontmatter as a fixed known-field struct rather than an order-preserving carrier,
`issue`/`source` are dropped on first write and AC-2/AC-4 cannot pass. The FO should
reconcile this at the gate by either (a) confirming `native-go-status` preserves
unknown fields, or (b) folding "preserve unknown frontmatter fields verbatim" into
`native-go-status`'s acceptance criteria. This checkpoint's value is mostly in
surfacing that requirement before the mutator is built.

## Stage Report: ideation

- DONE: Design the external-tracker integration checkpoint: how an external ticket (kata / Linear-style) imports into a Spacedock entity via the `issue` and `source` fields, and how that reference is PRESERVED through status, mutation, reports, and archive.
  Body **Proposed approach** adopts the two flat fields from `docs/specs/state-behavior-extension.md` and adds one binding contract — the mutator preserves unknown frontmatter fields verbatim — with the explicit lifecycle path (import -> status read -> `--set` -> report append -> `--archive`); preservation is AC-1..AC-4.
- DONE: Produce the v1 DECISION: does any bidirectional sync belong in v1, or should it remain an external adapter? Justify against the roadmap out-of-scope note (external tracker writeback beyond simple link fields is out of scope).
  **v1 DECISION** section: no bidirectional sync in v1, sync stays an out-of-process adapter, Spacedock owns execution status; justified against the roadmap out-of-scope line, AGENTS.md compatibility-first/stdlib rule, and the spec's bidirectional-ownership principle. Recorded as AC-6.
- DONE: AC (**AC-N** + Verified by) for the preserve-through-lifecycle property; note this is a DESIGN CHECKPOINT — the deliverable may be a decision/spec rather than code.
  Six ACs in `**AC-N - property.**` + `Verified by:` form (AC-1 status read, AC-2 `--set`, AC-3 report append + folder-form discovery, AC-4 `--archive`, AC-5 no tracker-specific semantics, AC-6 the v1 decision). Test plan states the deliverable is a decision/spec plus fixture-level assertions most cheaply folded into sibling mutation suites.

### Summary

Designed the external-tracker checkpoint as a decision-plus-contract, not a feature.
The core finding: "preserve through lifecycle" is really an unknown-field round-trip
property of the native mutator — `issue`/`source` are unknown to status semantics,
so they survive only if `--set`/`--archive` copy unknown fields through verbatim.
The v1 DECISION is no bidirectional sync in v1; the two link fields plus the
preservation guarantee ship, and all richer sync stays an external adapter with
Spacedock owning execution status — justified directly against the roadmap
out-of-scope note. Three assumptions (A1 parser carries unknown fields, A2 state-dir
write scoping, A3 folder-form detection) depend on sibling entities and are flagged
explicitly for the FO to reconcile at the gate; A1 is load-bearing (if
`native-go-status` uses a fixed known-field struct the preservation AC fails, so the
requirement may need folding into that entity). No product code touched; only this
entity file in the state checkout.

Process note for the FO: the dispatch's `### Fetch commands` block calls
`claude-team show-stage-def`, but `claude-team` is not on PATH in this environment
(exit 127); I ran it via its full path under `skills/commission/bin/` and it does
expose `show-stage-def`. Non-blocking — the inline checklist + entity file were a
complete spec — but the dispatch assumes `claude-team` is on PATH.

## Stage Report: implementation

- DONE: Add a direct AC-3 test: a folder-form fixture entity (index.md) carrying issue/source plus a reports/ subdir (or appended body) is discovered as exactly ONE entity and its issue/source frontmatter survives intact after a report append. Fold into the existing internal/status test suite — do NOT create a tracker-shaped package.
  `TestNativeFolderEntityReportAppendPreservesTracker` in `internal/status/native_mutation_test.go` (worktree commit ef566ca): builds a `070-tracker/index.md` + `070-tracker/reports/ideation.md` folder entity carrying `issue: kata:task-xyz789` / `source: kata`, appends a `## Stage Report` body section, then asserts the default status read matches the oracle byte-for-byte, the slug appears in exactly one data row (`strings.Count == 1`, not misdetected via the reports/ subdir), and `source: kata` / `issue: kata:task-xyz789` survive the append. Folded into the existing suite via `stageFixtureWith`; no tracker-shaped package added.
- DONE: go test ./... and go test ./... -race, gofmt -l, go vet all clean with REAL captured exit codes.
  `go test ./...` exit 0 (internal/status ok 11.9s, internal/cli ok, skills/integration ok); `go test ./... -race` exit 0 (internal/status ok 13.0s); `gofmt -l .` exit 0 (no files listed); `go vet ./...` exit 0 (no output). Test is load-bearing — flipping the asserted `issue:` value to a wrong string fails with the actual preserved frontmatter shown.

### Summary

AC-1/AC-2/AC-4 already had direct reproduced evidence in merged sibling suites
(`TestNativeUnknownFieldPreservation` for `--set`/`--archive` byte-for-byte;
`set-unrelated-keeps-unknown` in `zz_independent_parity_test.go` as the independent
oracle); AC-5/AC-6 are static and satisfied in this body. The only gap was AC-3's
specific combination — a folder-form entity carrying `issue`/`source` with an
internal `reports/` subdir, after a stage-report body append, discovered as exactly
one entity with frontmatter intact. Added one test for exactly that, folded into the
existing `internal/status` suite (no tracker package), proved against the live
oracle. All four gates clean with captured exit codes.
