---
id: mvmpr2vx79pj8w8j65g4vqyz
title: Retire the prose-grep integration assertions + repoint dispatch_test.go at native dispatch.Run
status: ideation
source: coverage matrix rows 9/10/12/13/14 (prose-grep antipattern) + handoff-confirmed test debt (dispatch_test.go drives retired Python claude-team build)
started: 2026-05-31T18:12:42Z
completed:
verdict:
score: "0.28"
worktree:
issue:
---

Two pieces of `skills/integration/` test debt the coverage matrix (archived entity
`behavior-test-skeleton-and-matrix`, id `8033qbqdrh4zba10w0d34m4j`) surfaced. Both are
"make an existing integration test genuinely behavioral instead of prose-grep / retired-helper."

**(a) Prose-grep antipattern (matrix rows 9/10/12/13/14).** Several assertions in
`skills/integration/skill_text_test.go` (and `contract_status_path_test.go`,
`contract_gate_test.go`) `os.ReadFile` a `.md` instruction file and `strings.Contains` the
contract TEXT for a clause. They pass even if the clause is **behaviorally dead** — the
contract says "use `--json`" but nothing proves a run consumes `--json`. Named in the matrix:
- row 9-prose: `contract_gate_test.go::TestStartupStepZeroIsContractGate` (asserts step-1 prose Contains, ordering via `strings.Index`)
- row 10-prose: `skill_text_test.go::TestLauncherStatusInvocations`, `contract_status_path_test.go::TestVendoredSkillsCallSpacedockStatus`
- row 11-prose: `skill_text_test.go::TestConcurrencySafeCommitClause`
- row 12: `skill_text_test.go::TestEventLoopReadsUseJSON`
- row 13-prose: `skill_text_test.go::TestDispatchBlockUsesNativeBuild`
- row 14: `skill_text_test.go::TestSplitRootContractClause`, `TestNoPRMergeOrModBehaviorIntroduced`, `contract_gate_test.go::TestStartupEmbeddedRangeBracketsContractVersion`

Most already have a behavioral counterpart (matrix `v1-implements` column). The matrix's
hx-reconciliation gives the decision rule: **port to behavioral where a seam exists; keep
genuine structural invariants (hx-AC-2 kind: an oracle-backed assertion over real on-disk
structure) with that label.** Do NOT blindly delete — distinguish prose-grep (no oracle, greps
prose) from legitimate-structural.

**(b) Retired-helper dependency (handoff-confirmed).** `skills/integration/dispatch_test.go`
still drives the **retired Python** `claude-team build` via `exec.Command("python3",
vendoredClaudeTeam(t), "build", ...)` (line ~45). The native seam is the in-process
`dispatch.Run`. Repoint the test at native `dispatch.Run` so it stops exercising the dead Python path.

## Acceptance criteria (provisional — ideation hardens)

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 — No `skills/integration` test asserts contract behavior via bare prose-grep where a
behavioral seam exists.** Verified by: each matrix-named prose-grep assertion is either
retargeted at its behavioral seam (binary/git/`dispatch.Run`) with the bare-Contains retired,
or explicitly retained+labeled as a genuine structural invariant with its oracle named; a
reproducible enumeration (grep/test-list) shows none of the ported ones remain as bare
prose-grep.

**AC-2 — `dispatch_test.go` exercises native `dispatch.Run`, not the retired Python helper.**
Verified by: `grep -n 'python3\|claude-team' skills/integration/dispatch_test.go` returns no
build-driving invocation; the test drives in-process `dispatch.Run` and asserts the same
observable dispatch outputs.

**AC-3 — The full suite stays green.** Verified by: `go test ./...` EXIT=0 (modulo the
known environment-only `TestCodexResolveManifestAgainstInstalledHost` failure), `gofmt -l`
clean, `go vet` clean — with real captured exit codes.

## Out of scope
- Rows 16/17 (team fail-early live name, codex packaged-agent) and the live-e2e CI net —
  deferred to a live-runtime harness.
- Row 15 (gate/feedback loop) — its own entity (`gate-feedback-loop-behavior-coverage`).

## Notes
Lives entirely in `skills/integration/`. Disjoint from the row-15 entity (which extends
`internal/ensigncycle`) — safe to run in a parallel worktree.
