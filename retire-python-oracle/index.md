---
id: 02a92vbcn4x7453bhszprpje
title: Retire the Python oracle — remove the embedded vendor/status + VendorRunner, graduate the differential-parity suite
status: backlog
source: FO investigation (2026-06-01) — the sole open precondition gating zj (yaml-parser-migration), currently orphaned (0x scoped it out; sprint-notes deferred it). Post-bootstrap.
started:
completed:
verdict:
score: "0.25"
worktree:
issue:
---

Retire the vendored Python status oracle so the codebase no longer carries a ~94KB embedded
interpreter-dependent reference. RUNTIME python-freedom is already done (verified on `next`: no
non-test `python3` exec; the steady-state FO loop is native after zs #246, with the `~/.claude`
reads quarantined in `internal/claudeteam`). What remains is **test-time**: the oracle
(`internal/status/vendor/status`, 96568 bytes) is still `//go:embed`-ed and run by `VendorRunner`
(`internal/status/vendor_runner.go`) as the differential-parity reference — native Go output is
byte-asserted against it across ~20 test files / ~50+ cases in `internal/status` + `internal/dispatch`.

This is the **sole open precondition** gating `zj` (yaml-parser-migration): byte-parity with the
Python oracle is the only thing forcing the hand-rolled frontmatter parser; once the oracle is
retired, `zj` can adopt a YAML library with documented divergences.

## Preconditions (per zj's gating clause)
1. **Native parity certified** — the differential-parity suite is green; native status + dispatch are trusted. **Effectively MET.**
2. **`claude-runtime-segregation` (zs) landed** — removed the last python RUNTIME shell-out. **MET (#246, archived).**
3. **VendorRunner + the embedded vendor/status retired** — **this entity.** The long pole.

## Scope
- **Graduate the differential-parity suite to standalone assertions.** Replace the oracle-comparison
  calls (`runOracle`/`indRunOracle`/`vendoredOracle` in the `native_*` / `zz_independent` / dispatch
  `*_parity` files + the two shared harnesses) with frozen goldens or embedded expected literals, so
  no test needs `python3` or either oracle script. The in-tree templates already exist: the
  `zz_independent --new` desired-behavior tests and `golden_read`'s frozen oracle-capture goldens.
- **Delete the oracle + its seam:** `internal/status/vendor/status` (96568 bytes), the
  `//go:embed vendor/status` (`vendor_runner.go:14`), `VendorRunner` (`vendor_runner.go`) and its
  test instantiations; drop `golden_read`'s `-update`/`runOracle` regeneration path.
- **Retire the dispatch-side vendored test dep** `skills/commission/bin/claude-team` (a hard test dep —
  `parity_harness_test.go` `t.Fatalf` if absent) once dispatch parity is frozen.
- Leave the already-standalone intentional-divergence tests untouched (build-statecommit, json_boot, fields-dedupe).

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — No embedded Python oracle remains.** End state: `internal/status/vendor/status`,
the `//go:embed`, and `VendorRunner` are gone; `go test ./...` requires no `python3` on PATH.
Verified by: a repo grep finds no python-exec path; the build has no embed of a Python script.

**AC-2 — The differential-parity suite is graduated, not deleted.** End state: the ~50+ former
oracle-comparison cases assert against frozen goldens/literals and stay green, preserving the
coverage the oracle-comparison gave. Verified by: the suite passes with the oracle scripts removed.

**AC-3 — Behavior preserved.** End state: the graduated tests still catch the divergences the
oracle-comparison caught (goldens are the captured oracle outputs at retirement time). Verified by:
a flip-test (mutate a native code path → a graduated test fails).

## Sequencing
oracle-retirement → (bootstrap graduation) → `zj` (YAML-library frontmatter migration). Do NOT fold
into `zj`: this freezes parity to goldens (parity-freeze risk); `zj` swaps the parser (library-
divergence risk) — keep the two risks isolated. Post-bootstrap; not on the current dev-workflow-
ergonomics sprint's critical path.

## Notes
The native code lives in `internal/status`, `internal/dispatch`, and `internal/claudeteam` (the
Claude seam holding the `~/.claude` reads) — all on `next`. (An FO investigation initially mis-read a
stale `main` checkout as lacking `internal/claudeteam`; it exists on `next` via seam #244 + zs #246.)
