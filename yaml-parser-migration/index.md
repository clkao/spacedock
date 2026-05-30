---
id: zjmjzznydmqr58bd46qz6q07
title: Migrate the frontmatter parser/mutator to a YAML library (post-oracle, deliberate divergences)
status: backlog
source: sprint — captain (parser-modernization, post-bootstrap)
score: "0.25"
worktree:
---

Replace the hand-rolled line-oriented frontmatter parser + line-rewriter mutator with a YAML library (`gopkg.in/yaml.v3`, `yaml.Node` for round-trip), being DELIBERATE about which Python-oracle quirks to drop (documented divergences) rather than enshrining them. The Go binary links deps at build time, so the Python-era zero-dep rationale does not apply (see sprint-notes "Go-binary dependency policy correction"); the only thing blocking the swap today is byte-PARITY with the Python oracle, which is a bootstrap migration scaffold, not a long-term goal.

## PREREQUISITE (documented — this entity is GATED on it)

The Python oracle must be RETIRED first. Concretely, all of:
1. **Parity certified** — the native status + dispatch implementations are trusted (the differential parity suite has done its job).
2. **`claude-runtime-segregation` (zs) landed** — it moves `context-budget` (and the standing-teammate surface) native, removing the last Python *runtime* shell-out. Until then the binary still needs `python3` at runtime, so the oracle cannot fully retire.
3. **VendorRunner + the embedded ~94 KB Python script retired** — the architecture review's own post-bootstrap item (`internal/status/vendor_runner.go`, `//go:embed vendor/status`, the migration-scaffold tests). Once retired, the byte-parity constraint dissolves and the reader can adopt a library.

Until the prerequisite holds, the hand-rolled parser is REQUIRED (a YAML-spec library diverges from Python's line parser on the edge cases the parity suite pins, breaking parity by construction). This entity does not start before bootstrap graduates.

## Test-coverage assessment (captain's question, answered)

We have a PARTIAL but real base of implementation-independent (desired-behavior) tests that would guard a swap:
- `internal/status/frontmatter_test.go` — 9 parser unit cases (basic/empty/matched-quotes/mismatched-quotes/nested-ignored/last-key-wins/no-fence/colon-first-split/BOM) asserting parsed RESULTS, not "matches Python".
- `TestNativeUnknownFieldPreservation` (`native_mutation_test.go`) — the unknown-field round-trip behavioral test (`issue`/`source`/`tracker-url` survive `--set`+`--archive`), implementation-independent.

But the BULK is oracle-parity (26 oracle-coupled test files) which retires with the oracle, AND several of the 9 unit cases encode Python QUIRKS ("mismatched quotes preserved", "last-key-wins", "colon-splits-first-only"). So part of this entity is DECIDING, per case, which is genuinely-desired (keep, assert it) vs a Python quirk (let the library win, update/retire the test) — and FILLING desired-behavior gaps so the swap is guarded independent of the oracle.

## Acceptance criteria (behavioral; sharpen at ideation)

**AC-1 — The frontmatter reader is a YAML library; a curated desired-behavior test suite (decoupled from the Python oracle) guards it, with each deliberate divergence from the retired Python documented.**
Verified by: the parser unit suite is re-pointed at the desired behavior (Python-quirk cases either kept-as-desired with rationale or replaced with the library's standard behavior + a documented divergence note); `go test ./...` green with the oracle retired (no parity-skip).

**AC-2 — Unknown-field round-trip is preserved (field-exact, not necessarily byte-exact).**
Verified by: a `TestNativeUnknownFieldPreservation`-style test — arbitrary unmodeled frontmatter fields (`issue`/`source`/`tracker-url`/etc.) survive `--set` and `--archive` with their values intact and order stable, via a `yaml.Node` parse→modify-target→marshal that leaves the rest of the node tree untouched. The contract relaxes from byte-identical to FIELD-identical (acceptable once there is no Python oracle to byte-match); this relaxation is documented.

**AC-3 — Net LOC reduction realized.**
Verified by: the hand-rolled parser/mutator code is removed and replaced by the library + a thin `yaml.Node` round-trip seam; the diff shows the net removal (estimate below) minus the small library-glue addition.

## Estimated LOC removal

Direct YAML-reader/mutator candidates (gross, replaceable by yaml.v3 + a yaml.Node round-trip):
- `internal/status/frontmatter.go` — 148 (the line parser)
- `internal/status/orderedmap.go` — 33 (ordered map; `yaml.Node` preserves order natively)
- `internal/status/mutate.go` `updateFrontmatter` — ~89 (the line-rewriter; becomes a yaml.Node parse→set→marshal)
- `internal/status/stages.go` — 213, of which the `stages:`-block YAML parsing is a partial candidate (not the README `### {stage}` markdown extraction)

Gross ≈ **270–400 LOC**; net ≈ **200–350 LOC removed** after adding the yaml.Node round-trip glue (~50–100). SEPARATE axis (NOT this entity — retires with dispatch-OUTPUT parity, not the yaml reader): the dispatch Python-mimicry helpers `internal/dispatch/{pyrepr.go (35), shellquote.go (38)}` + `pyJoin` + `splitTextLines` (~110 LOC). Ideation refines these counts.

## Notes

Post-bootstrap; off the critical path. Pairs with the sprint-notes "parsing modernization" follow-up (revise AGENTS.md line 10's dependency policy for the binary) and the VendorRunner-retirement architecture-review item. Be deliberate about divergences (captain): we choose what to break; we do not enshrine Python quirks where a standard is better. The unknown-field round-trip is the one hard requirement — `yaml.Node` is the way.
