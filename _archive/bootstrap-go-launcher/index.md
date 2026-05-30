---
id: 9869srv10sxp2ppkx9m0m321
title: Bootstrap Go launcher
status: done
score: "0.95"
source: bootstrap roadmap
worktree: 
started: 2026-05-30T04:04:36Z
completed: 2026-05-30T04:21:45Z
verdict: PASSED
archived: 2026-05-30T04:22:08Z
---

# Bootstrap Go Launcher

Create the minimum Go project that can host the Spacedock v1 launcher: a buildable
repo with a process entry point that delegates to a small CLI package, stable
help/version/unknown-command behavior, agent instructions, and a deliberate
`status` placeholder so no compatibility behavior leaks in before it is designed.

## Problem statement

Spacedock is migrating from plugin-shipped scripts to a stable `spacedock` command
surface. Every later roadmap stage (vendor status, symlink state profile, native
status parity, split-root) builds on a working Go repo with a clean CLI boundary.
This task proves the repo compiles, routes commands, and returns the right exit
codes, with nothing implemented past a `status` placeholder. It is foundation-only:
the deliverable largely predates this workflow and was landed by the
`chore: bootstrap spacedock v1` commit; ideation here is design/assessment, not a
rebuild.

## Acceptance criteria

Each AC names a property of the finished entity, not a stage action, and how it is
verified. All ACs below are already satisfied by the bootstrap commit; see
**Gap assessment** for the honest per-AC status.

**AC-1 - The process entry point delegates to a small CLI package and owns nothing but `os.Exit`.**
Verified by: `cmd/spacedock/main.go` is a single `main()` that calls
`cli.Run(os.Args[1:], os.Stdout, os.Stderr)` and exits with its return code;
routing/usage/exit-code logic lives in `internal/cli/cli.go`. Reproduce with
`go build ./...` (compiles) and inspect the two files.

**AC-2 - `--help`/`help`/`-h` and bare invocation print usage to stdout and exit 0.**
Verified by: Go test `TestHelp` in `internal/cli/cli_test.go` asserts exit 0,
`Usage:` on stdout, empty stderr. Runnable: `go run ./cmd/spacedock --help`
(prints usage, exit 0).

**AC-3 - `--version`/`version` prints `spacedock {Version}` to stdout and exits 0.**
Verified by: Go test `TestVersion` asserts stdout equals `spacedock <Version>`,
exit 0, empty stderr, where `Version = "0.1.0-dev"` in `cli.go`. Runnable:
`go run ./cmd/spacedock --version`.

**AC-4 - An unknown command writes `unknown command: {name}` plus usage to stderr and the CLI returns exit code 2.**
Verified by: Go test `TestUnknownCommand` asserts `Run([]string{"bogus"}, ...)`
returns 2 and stderr contains `unknown command: bogus`. (Under `go run` the
shell observes exit 1 because `go run` wraps the child's `exit status 2`; the
compiled binary exits 2 — assert against `Run`'s return value, not `go run`'s
shell code.)

**AC-5 - `status` is a deliberate placeholder only: it writes `spacedock status: not implemented yet` to stderr and returns exit code 2; no status logic exists.**
Verified by: the `case "status"` arm in `cli.go` prints the not-implemented
message to stderr and returns 2; there is no `internal/status/` package
(`find . -path ./internal/status -type d` returns nothing). Runnable:
`go run ./cmd/spacedock status` prints the message.

**AC-6 - The repo carries agent instructions covering Go conventions, project shape, and skill development.**
Verified by: `AGENTS.md` exists with `## Priorities`, `## Expected Commands`
(go test / `-race` / gofmt), `## Project Shape`, `## State-Branch Bootstrap
Rules`, and `## Skill Development` sections; `skills/README.md` documents the
skill-integration target. Reproduce: `grep -l "Expected Commands" AGENTS.md`.

**AC-7 - The Go suite passes clean, including under the race detector, and the formatted-code gate is green.**
Verified by: `go test ./...` and `go test ./... -race` both report 3 passed in
2 packages; `gofmt -l ./cmd ./internal` prints nothing (exit 0).

## Gap assessment

Honest status of the existing scaffold against each AC. The bootstrap commit
already lands a complete, passing deliverable for this stage; no residual product
work is required, only verification.

| AC | Already satisfied | Residual implementation work |
|----|-------------------|------------------------------|
| AC-1 entry point delegates | `cmd/spacedock/main.go` is a 12-line `main()` calling `cli.Run`; logic in `internal/cli/cli.go` | None |
| AC-2 help/usage exit 0 | `TestHelp` passes; `--help`/`help`/`-h`/bare all route to `printUsage` and return 0 | None |
| AC-3 version exit 0 | `TestVersion` passes; `Version = "0.1.0-dev"` | None |
| AC-4 unknown → exit 2 | `TestUnknownCommand` passes; default arm returns 2 with message+usage on stderr | None |
| AC-5 status placeholder only | `case "status"` returns 2 with not-implemented message; no `internal/status/` exists | None |
| AC-6 agent instructions | `AGENTS.md` + `skills/README.md` present with required sections | None |
| AC-7 clean suite + race + fmt | `go test ./...` and `-race` → 3 passed; `gofmt -l` clean | None |

Notes:
- The seed AC line "delegates to a small CLI package" maps to AC-1; "supports
  help, version, and unknown-command behavior" splits into AC-2/3/4 so each gets
  its own reproducible proof.
- The only sharp edge is the `go run` exit-code masking called out in AC-4. It is
  a test-harness artifact, not a defect; validators should assert on `Run`'s
  return value or the compiled binary's exit status, never on `go run`'s shell
  code.
- `internal/status/` is intentionally absent (AGENTS.md lists it as future). Its
  absence is part of AC-5's proof, not a gap.

## Test plan

Smallest proof surface; all gates are cheap (seconds) and require no fixtures,
network, or live workflow run. The deliverable predates this workflow, so the
plan is verification-only — there is no new code to test-drive.

| Check | Command | Proves |
|-------|---------|--------|
| Unit suite | `go test ./...` | AC-2/3/4 via `TestHelp`/`TestVersion`/`TestUnknownCommand`; package compiles (AC-1) |
| Race hygiene | `go test ./... -race` | No concurrency hazards (AC-7 gate; trivial here, kept as a standing gate) |
| Format gate | `gofmt -l ./cmd ./internal` | Code is gofmt-clean (empty output, AC-7) |
| Build | `go build ./...` | Entry point + package compile (AC-1) |
| Help smoke | `go run ./cmd/spacedock --help` | Usage to stdout, exit 0 (AC-2) |
| Version smoke | `go run ./cmd/spacedock --version` | `spacedock 0.1.0-dev`, exit 0 (AC-3) |
| Status placeholder | `go run ./cmd/spacedock status` | not-implemented message, placeholder-only (AC-5) |

Cost/complexity: trivial (whole suite runs in seconds; no fixtures). Fixture, CLI
golden, or live-workflow tests are **not** needed at this stage — golden status
fixtures and compatibility runs belong to the later vendor-status and native-status
stages, not bootstrap. The CLI's stdout/stderr split is asserted directly through
`cli.Run` with `bytes.Buffer`, which is the right abstraction level for this claim
and avoids depending on `go run`'s exit-code wrapping.

## Notes

This stage proves the repo is buildable before compatibility work begins. The
deliverable largely predates this workflow (landed by `chore: bootstrap spacedock
v1`); the ideation work here is to express the ACs as reproducible entity-level
properties and confirm, honestly, that the scaffold already satisfies them so
implementation is not redundant.

## Stage Report: ideation

- DONE: AC rewritten as entity-level properties in the README's `**AC-N** ... Verified by:` format, each with a concrete reproducible proof (go test name or runnable command).
  Body now has AC-1..AC-7 in `**AC-N - property.**` + `Verified by:` form; each cites a Go test name (`TestHelp`/`TestVersion`/`TestUnknownCommand`) or a runnable command.
- DONE: Honest gap-assessment of the existing scaffold (cmd/spacedock, internal/cli, AGENTS.md, tests) against each AC: state explicitly what already passes vs. what implementation must still do, so implementation is not redundant.
  Added **Gap assessment** table: every AC marked already-satisfied with "None" residual work, plus the `go run` exit-code-masking note and the intentional absence of `internal/status/`.
- DONE: Test plan names the smallest proof surface (Go unit tests for CLI routing/exit codes; gofmt and -race as hygiene gates) and notes that the deliverable largely predates the workflow.
  **Test plan** table lists `go test ./...`, `-race`, `gofmt -l`, `go build`, and the three smoke commands; states no fixtures/live runs are needed and that the deliverable predates the workflow.

### Summary

Verified the bootstrap scaffold empirically (`go test ./...` and `-race`: 3 passed
in 2 packages; `gofmt -l ./cmd ./internal` clean; all CLI smoke commands behave as
designed) and reshaped the entity body into the README's required AC format with
seven entity-level, reproducibly-proven acceptance criteria. The honest finding:
the `chore: bootstrap spacedock v1` commit already satisfies every AC, so the
implementation stage is verification-only — no residual product work. The single
notable sharp edge is that `go run` masks the CLI's exit-2 as shell exit 1, so
exit-code claims must be asserted against `cli.Run`'s return value or the compiled
binary, which the unit tests already do. No product code was touched; only the
entity file in the state checkout was edited.

## Stage Report: implementation

- DONE: Confirm all 7 ACs against the worktree checkout: run `go test ./...`, `go test ./... -race`, `gofmt -l ./cmd ./internal`, and the help/version/unknown/status smoke commands; record the ACTUAL output as evidence in the stage report.
  `go test ./...` and `go test ./... -race` → "3 passed in 2 packages"; `gofmt -l ./cmd ./internal` empty (exit 0); `go build ./...` Success. Compiled binary smokes: `--help`/bare → usage on stdout, empty stderr, exit 0 (AC-2); `--version` → `spacedock 0.1.0-dev` on stdout, exit 0 (AC-3); `bogus` → `unknown command: bogus`+usage on stderr, exit 2 (AC-4); `status` → `spacedock status: not implemented yet` on stderr, empty stdout, exit 2 (AC-5). Source inspection confirms AC-1 (`main.go` is a 11-line `main()` calling `cli.Run`; routing in `internal/cli/cli.go`). AC-6: AGENTS.md has all 5 required sections + skills/README.md present. AC-5 absence proof: `find . -path ./internal/status -type d` empty.
- DONE: If any AC gap surfaces, make the minimal fix and commit it to the worktree branch; otherwise state explicitly 'no code change required' with the evidence that proves each AC is already met.
  No code change required. Worktree working tree was clean before and after verification (`git status` empty); every AC proven satisfied by the bootstrap commit via the evidence above. No fix needed.
- DONE: Keep the two commit streams separate: any code change commits to the worktree branch `spacedock-ensign/bootstrap-go-launcher`; the stage report is written to the entity file in the state checkout and committed in THAT repo (the entity file is outside the worktree).
  No worktree code commit (none required). This stage report is the only edit, written to the entity file in the state checkout and committed in the `docs/dev/.spacedock-state` repo (branch `spacedock-state/dev`), separate from the worktree branch.

### Summary

Verification-only stage. Empirically confirmed all 7 ACs against the worktree
checkout (`spacedock-ensign/bootstrap-go-launcher`): unit suite and race detector
both report 3 passed in 2 packages, gofmt is clean, build succeeds, and every
smoke command behaves as designed. Exit-code claims for the unknown-command (AC-4)
and status-placeholder (AC-5) cases were proven against the compiled binary — both
return a true exit 2 — sidestepping the documented `go run` masking artifact. The
bootstrap commit already satisfies every AC; no code change was required and the
worktree tree stayed clean throughout. Only this stage report was written, to the
state-checkout entity file.

Process note for the FO: the dispatch's `### Fetch commands` block calls
`claude-team show-stage-def`, but the installed `claude-team` binary exposes only
`{context-budget,build,spawn-standing,list-standing}` and exits non-zero on
`show-stage-def`. The assignment was fully self-contained (inline checklist + this
entity file as the spec), so the fetch failure was non-blocking, but the dispatch
shape references a subcommand the current binary does not provide.

## Stage Report: validation

- DONE: Independently reproduce each AC-1..AC-7 'Verified by' clause from the worktree branch checkout (run `go test ./...`, `-race`, `gofmt -l ./cmd ./internal`, `go build ./...`, and smoke the COMPILED binary for the AC-4/AC-5 exit-2 codes). Confirm every AC has reproduced evidence; flag any that do not.
  Worktree on `spacedock-ensign/bootstrap-go-launcher` @ b10ac27, clean tree. `go test ./...` & `-race` → 3 passed in 2 packages (raw `-v`: TestHelp/TestVersion/TestUnknownCommand all PASS; `cmd/spacedock` = no test files). `gofmt -l ./cmd ./internal` empty (exit 0); `go build ./...` and `go vet ./...` exit 0. Compiled binary smokes (true exit codes, no `go run`): AC-2 bare/`--help`/`help`/`-h` → usage on stdout, empty stderr, exit 0; AC-3 `--version`/`version` → `spacedock 0.1.0-dev` on stdout, exit 0; AC-4 `bogus` → `unknown command: bogus`+usage on stderr, exit **2**; AC-5 `status` → `spacedock status: not implemented yet` on stderr, exit **2**. AC-1: `main.go` = 11-line `main()` calling `cli.Run(os.Args[1:], os.Stdout, os.Stderr)` + `os.Exit`; logic in `internal/cli/cli.go`. AC-5 absence: `internal/status` dir absent, no `package status` anywhere (AGENTS.md lists it as future). AC-6: all 5 AGENTS.md sections present, Expected Commands cites go test/-race/gofmt, `skills/README.md` present. Every AC reproduced.
- DONE: Judge that the tests prove the INTENDED behavior at the right level (exit codes asserted via cli.Run / compiled binary, not `go run`; status is a real placeholder, not a stub hiding logic) — reject if a test passes while encoding the wrong target.
  Exit codes are asserted at the right level: `TestUnknownCommand` checks `cli.Run` returns 2 directly, and I confirmed exit 2 against the compiled binary — neither path relies on `go run` (which masks exit 2 as shell exit 1). `status` is a genuine placeholder: the `case "status"` arm only prints the not-implemented message and returns 2; no `internal/status/` package and no `package status` source exists, so it cannot be a stub hiding logic. TestHelp/TestVersion assert real stdout/stderr/exit targets via `bytes.Buffer`, not mocks. One narrowness flag (non-blocking): AC-2's cited `TestHelp` only exercises `--help`; the `help`/`-h`/bare variants have no Go test, but I independently verified all four against the compiled binary and the AC-2 property holds.
- DONE: Give a PASSED or REJECTED recommendation backed by the reproduced evidence; if REJECTED, name the concrete gap to route back to implementation.
  Recommendation: **PASSED**. All 7 ACs independently reproduced from the worktree checkout; no gap warrants routing back to implementation.

### Summary

Independent validation: PASSED. Reproduced all 7 ACs from the worktree branch
checkout without trusting prior stage reports — `go test ./...` and `-race` give
3 passing tests across 2 packages, gofmt/build/vet are clean, and the AC-4/AC-5
exit-2 codes were confirmed against the COMPILED binary (true exit 2, sidestepping
the documented `go run` masking). Exit-code assertions are at the correct level
(`cli.Run` return value / compiled binary, never `go run`), and `status` is a real
placeholder with no hidden logic or `internal/status/` package. The only nuance is
that AC-2's cited `TestHelp` covers only `--help`; I verified the `help`/`-h`/bare
variants directly against the binary, so the AC-2 property still holds — narrower
test coverage than the stated property, not a defect. No code changes; worktree
stayed clean throughout. This stage report is the only edit, in the state checkout.
