---
id: 0xcqyh24hr5xnek3kfp8makg
title: Architecture-review cleanups — share pyJoin, fix parity-test skip-vs-fail, drop dead code
status: backlog
source: sprint — architecture review (must-fix #2/#3 + now-safe nice-to-haves)
score: "0.28"
worktree:
---

The NOW-SAFE cleanups surfaced by the architecture review (`docs/dev/_reviews/architecture-review.md`). The review found the core architecturally sound with NO rework required; these are the small, contained, in-place fixes worth landing during the implementation drain. POST-BOOTSTRAP parity-debt is explicitly NOT here (it goes elsewhere — see Out of scope).

## Acceptance criteria (behavioral)

**AC-1 — `pyJoin` is shared, not byte-duplicated across the status↔dispatch seam.**
Verified by: `pyJoin` is exported as `status.PyJoin` and `internal/dispatch` consumes it (the byte-identical copy at `internal/dispatch/helpers.go:171-187` is deleted); `go build ./...` clean and the existing abs-worktree parity test still passes (both sides necessarily match `os.path.join`). This removes the silent-divergence hazard the review flagged (the absolute-component-wins quirk lived in two copies).

**AC-2 — Status parity tests FAIL, not skip, when the Python oracle is absent.**
Verified by: with the oracle pointed away (env unset / hardcoded laptop path absent), the strongest status parity assertions (`TestNativeReadMatchesOracle`, `TestNativeValidationParity`, the `TestInd*` matrix) FAIL rather than `t.Skip()` — observed by running with `SPACEDOCK_ORACLE` unset and watching them error (today ~66 subtests go green-by-skip on CI/a fresh clone). Mirror the dispatch package's tree-relative vendored-oracle + `t.Fatalf` pattern (`parity_harness_test.go:32`), or at minimum hard-fail in CI when the oracle var is unset. This is a test-integrity fix: the suite must not report PASS while the real parity assertions silently did not run.

**AC-3 — The verified dead code / test-hygiene items are removed; the suite stays green.**
Verified by: `go vet ./...` + `go test ./...` green after removing — dead struct field `roots.definitionDirSpelling` (written, never read; `roots.go:29,51`), unused `(*entity).get` (`discover.go:45-48`, zero callers), the one-line passthrough `runGit` (inline into its single caller, `native_runner.go:490-493`), the value-free `sortStrings` alias (`orderedmap.go:32-33`), the two `_ = strings.TrimSpace` import-keeping crutches (`native_usage_test.go:66`, `native_validate_test.go:150`), and the test-side SD-B32 alphabet re-anchored to the production `sdB32Chars` constant (`nextid_boot_test.go:12` vs `identity.go:18`). Each is a verified-zero-risk removal per the review.

## Test gates

- `go test ./...` + `-race`, `gofmt -l`, `go vet` — all green with captured exit codes.
- AC-2's oracle-absent-fails behavior demonstrated (run with the oracle var unset → the parity tests FAIL).

## Out of scope (goes elsewhere)

- **Scope the "Python-free" claim honestly** (review must-fix #4) → `claude-runtime-segregation` (zs) owns it (it moves context-budget native + reorganizes the contract).
- **`--quiet`** — captain elected to KEEP it; not pruned.
- **POST-BOOTSTRAP parity-debt** → the `yaml-parser-migration` / parsing-modernization follow-up: collapse the six mutual-exclusion flag-blocks (post-parity), retire VendorRunner + the embedded Python script (post-parity-certification), tighten Python schema_version quirks (post-oracle). And the bare-mode team-evidence `~/.claude/teams` mtime probe relocation to the FO adapter is a zs-adjacent post-bootstrap item.

## Notes

Touches `internal/status` + `internal/dispatch` (the `pyJoin` seam) — sequences with the other internal/status entities (after packaging; can fold into the implementation drain). Small, low-risk, behavior-preserving. The review's full finding list is at `docs/dev/_reviews/architecture-review.md`.
