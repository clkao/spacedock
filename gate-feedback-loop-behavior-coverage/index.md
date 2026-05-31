---
id: m6h1tyyen9z9g7cm830wcpzq
title: Behavioral coverage for the gate/feedback approval loop (matrix row 15)
status: done
source: FO/captain (2026-05-31) — comprehensive-behavior-coverage sprint; coverage matrix row 15 (largest pure GAP)
started: 2026-05-31T18:06:58Z
completed: 2026-05-31T18:27:27Z
verdict: PASSED
score: "0.34"
worktree: 
issue:
---

Row 15 of the coverage matrix (archived entity `behavior-test-skeleton-and-matrix`,
id `8033qbqdrh4zba10w0d34m4j`) is the **largest pure GAP**: v1 has **zero behavioral
coverage** of the gate → reject → feedback-reflow → keep-alive loop. The Python side
covers it live (`tests/test_gate_guardrail.py`, `test_rejection_flow{,_codex}.py`,
`test_feedback_keepalive.py`, `test_merge_hook_guardrail.py`). The matrix's port roadmap
names this as priority #2 (after the skeleton, which is shipped): "needs a scripted-FO
harness extension of the skeleton."

The ensign is a live LLM, so — exactly as entity 80 framed it — we cannot test the
*whole* loop with a live agent in `go test`. The footing must target the **deterministic
seams** the gate/feedback flow depends on, stubbing the LLM with a scripted stand-in, and
assert mechanical outputs with **anchored emit-form** regexes (never bare `strings.Contains`,
which the spike proved is fooled by the body's own warning prose).

## Candidate deterministic seams (ideation to confirm/narrow)

The gate/feedback flow's mechanical contract lives in:
- the **feedback-reflow dispatch body** — `dispatch.Run` with `is_feedback_reflow: true`
  must carry the routed fix-request assignment + `feedback-to` target stage (vs. a plain
  acknowledgment). This is the in-process seam analogous to the skeleton's build seam.
- the **`### Feedback Cycles` entity-body section** — the FO-owned rejection-round record;
  its shape and the 3-cycle escalation boundary are byte-observable.
- the **validation stage's PASSED/REJECTED recommendation** parsing into the gate decision.

## Acceptance criteria (hardened)

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 — A behavioral test exercises the gate/feedback rejection-reflow deterministic seam:
a real `dispatch.Run` feedback-reflow build that routes the fix request to the `feedback-to`
target stage.** The finished entity contains a test that drives the in-process
`dispatch.Run(["build", ...])` with `is_feedback_reflow: true` + a `feedback_context` payload,
dispatched to the gate stage's `feedback-to` target, and asserts the dispatch body's mechanical
outputs: (a) an anchored `### Feedback from prior review` routing section, (b) the routed
rejection context carried inside it (the FO-mode requirement that the routed message carry the
concrete fix work, not a bare acknowledgment), and (c) a contrast assertion that a plain
(non-reflow) dispatch to the *same* target stage does NOT emit the routing section.
Verified by: `go test ./internal/ensigncycle/` runs the test green; it calls the real build seam
in-process (no live LLM), reusing the `internal/ensigncycle` scripted-cycle fixture pattern.

**AC-2 — The test FAILS when the gate/feedback mechanical output breaks, via anchored emit-form
assertions (never a bare `strings.Contains` a prose warning could satisfy).** The finished
entity's test is a real guard: a named negative control that regresses the production reflow
seam turns it RED. Concretely, the entity ships TWO negatives: **NEG-A** — strip the routed
`feedback_context` payload from the emitted body → the context-presence assertion goes RED
(proves the test catches a dropped/empty fix request, the FO-mode "not just an acknowledgment"
failure); **NEG-B** — a missing-`feedback_context` reflow build → `dispatch.Run` Rule 5 exits
non-zero with `dispatching to feedback target stage '{stage}' but feedback_context is missing`,
asserted by an anchored stderr-message match, proving the build-side guard fires. Both are
encoded as in-test guards (the negative paths are exercised, not described). The routing-section
assertion is the anchored `(?m)^### Feedback from prior review$` form, not a bare substring; the
spike confirmed the heading string is emitted exactly once in `build.go` (no warning-prose
duplicate), so the trap here is a *dropped routed payload*, which the context-presence assertion
catches.

## Out of scope
- Live-LLM gate compliance (does a *real* FO/ensign honor the loop, self-approve, or count
  cycles) — stays live-pytest / deferred to a live-runtime harness (matrix rows 16/17/CI).
- **The `### Feedback Cycles` 3-cycle escalation boundary** — confirmed during ideation to have
  NO in-process Go seam: it is FO-owned entity-body prose, written and counted by a live LLM,
  with no `internal/status` parser that reads or acts on it (`grep` of `internal/status/*.go`
  for `Feedback Cycles` is empty). Asserting its shape would be either prose-grep (the antipattern
  this entity exists to replace) or would require inventing a production parser (scope creep,
  YAGNI). The cycle-counting behavior is the same live-LLM class as gate compliance above.
- The merge-hook guardrail (this workflow has no merge hooks).

## Notes
Test-infra surface (extends `internal/ensigncycle` or a sibling). Likely disjoint from the
prose-grep retirement entity (`harden-integration-prose-grep-tests`), which lives in
`skills/integration/` — the two can run in parallel worktrees.

---

# Ideation design (hardened)

## The pinned deterministic seam

The gate/feedback loop's mechanical contract reduces to ONE byte-observable, no-live-LLM seam:
the **feedback-reflow dispatch body** built by `internal/dispatch.Run(["build", ...])` when
`is_feedback_reflow: true`. Reading `build.go`:

- Rule 5 (`build.go:225`): `is_feedback_reflow && feedback_context == ""` → exit non-zero with
  `dispatching to feedback target stage '{stage}' but feedback_context is missing`. This is the
  build-side guard that a reflow MUST carry a routed payload.
- Section 6 (`build.go:343-345`): when `feedback_context != ""`, the body emits
  `### Feedback from prior review\n\n{feedback_context}\n`. This is the routed fix request the
  FO's Feedback Rejection Flow (`first-officer-shared-core.md:199-214`) requires: step 5 says
  "the routed message must carry the concrete next-stage assignment and requested fix work, not
  just an acknowledgment." The reflow is dispatched to the rejected gate stage's `feedback-to`
  target (e.g. `validation → feedback-to: implementation`), not to the reviewer.

The other two candidate seams from the seed do NOT survive as in-process targets:
- **`### Feedback Cycles` / 3-cycle escalation** — no Go seam (see Out of scope). Live-LLM only.
- **validation PASSED/REJECTED → gate decision** — the recommendation parse + the gate hold are
  FO-LLM behaviors (the FO reads the stage report and decides). No deterministic Go function
  "is" the gate. Live-pytest covers it (`test_gate_guardrail.py`, `live`). Out of scope here.

So the test targets the reflow-build seam: the deterministic half of the loop, on the dispatch
side of the LLM, exactly mirroring how the shipped skeleton targets `dispatch.Run` rather than a
live ensign.

## Spike result (riskiest unknown — done FIRST, per entity-80 discipline)

Built a throwaway `TestSpikeFeedbackReflowSeam` in `internal/dispatch/` (reusing the parity
harness's `runNative` + `readDispatchBody` + `readmeWorktree`/`entityFM` fixtures, whose README
already declares `validation → feedback-to: implementation`) and ran it:

- **Positive:** two real `dispatch.Run` builds from the same fixture — one
  `is_feedback_reflow:true` + `feedback_context:"REJECTED: ..."` dispatched to `implementation`
  (the feedback-to target), one plain to the same stage. Asserted (a) the reflow body matches the
  anchored `(?m)^### Feedback from prior review$`, (b) it contains the routed `REJECTED: ...`
  context, (c) the plain body does NOT match the routing heading. **PASS in ~1s, no live agent.**
- **Negative control (REAL production mutation):** guarded `build.go:343` behind `if false &&`
  (dropping the section-6 emission). The spike went **RED** at both the anchored-heading and the
  context-presence assertions (`reflow body missing anchored feedback-routing section`). Restored
  `build.go`; spike removed; `go test ./internal/dispatch/ ./internal/ensigncycle/` green.

**Lesson confirmed:** the seam is scriptable in `go test` against the real build, and the
anchored routing-section assertion + context-presence assertion together catch a regressed/dropped
fix request. The heading string is emitted exactly once in `build.go` (verified by grep), so unlike
the completion-signal case there is no warning-prose duplicate; the live trap here is a *dropped
routed payload*, caught by the context-presence assertion — which is why AC-2's NEG-A strips the
payload rather than renaming a call.

## Test plan

**Home:** extend `internal/ensigncycle` (the skeleton's home — test-infra, disjoint from
`frontdoor.go`). Add `feedback_test.go` reusing the existing `stageFixture` helpers, OR add a
worktree-stage fixture (the package's current `readmeNonWorktree` has no gate stage, so the new
test needs a README with a `feedback-to` stage like the parity harness's `readmeWorktree`). No new
production code — reuses the in-process `dispatch.Run` build seam.

**`TestFeedbackReflowRoutesFixRequest`** (runnable `go test ./internal/ensigncycle/`):
1. Stage a git-init'd fixture whose README declares a gate stage with `feedback-to:` (e.g.
   `validation → feedback-to: implementation`); a `{slug}.md` entity stamped at the gate stage
   with a `worktree:` value (reflow rides the existing worktree).
2. Build the reflow dispatch: `dispatch.Run(["build", "--workflow-dir", root], stdinJSON, ...)`
   with `is_feedback_reflow:true`, a concrete `feedback_context` (the routed REJECTED findings),
   `stage` = the `feedback-to` target. Read back the dispatch body.
3. Build a plain dispatch to the same target stage (no reflow) for the contrast assertion.
4. **Assert (anchored, behavior-first):**
   - (a) reflow body matches `(?m)^### Feedback from prior review$` (the routing section).
   - (b) reflow body `strings.Contains` the routed `feedback_context` payload verbatim (the
     concrete fix request — NOT a bare acknowledgment).
   - (c) plain body does NOT match the routing-section regex (the seam is reflow-specific).

**`TestFeedbackReflowGoesRedOnBrokenOutput`** (the AC-2 verification — named negative controls):
- **NEG-A (dropped payload):** take the real reflow body and strip the `feedback_context`
  substring; assert the context-presence check fails on it — the in-test encoding of the spike's
  production-mutation negative (proves a dropped fix request is caught).
- **NEG-B (build-side guard):** call `dispatch.Run` with `is_feedback_reflow:true` and EMPTY
  `feedback_context`; assert exit code != 0 AND an anchored stderr match of
  `dispatching to feedback target stage '{stage}' but feedback_context is missing` (Rule 5 fires;
  the build refuses an unrouted reflow). Asserts the message form, not a bare substring.

**Cost/complexity:** Go unit test, fixture-only, ~1s, no live runtime — same tier and harness as
the shipped skeleton. No CLI or live-workflow test needed (those are the out-of-scope live rows).

**Why proof is at the right altitude:** the claim is "the dispatch build routes a fix request to
the feedback-to target on reflow." That is a `dispatch.Run` command behavior → a Go unit test
asserting the real body is the matching altitude (not a live-workflow smoke test, which would be
testing the LLM's compliance, the explicitly out-of-scope row).

## Stage Report: ideation

- DONE: Pin the deterministic seam(s): read internal/dispatch dispatch.Run's feedback-reflow path (is_feedback_reflow=true) and the FO `### Feedback Cycles` contract; name the exact in-process / byte-observable seam the test will target (not a live LLM).
  Pinned the ONE real seam: the `dispatch.Run` reflow build (`build.go:225` Rule 5 guard + `build.go:343` section-6 routing emission), dispatched to the gate stage's `feedback-to` target. Confirmed `### Feedback Cycles` (FO prose, no `internal/status` parser — grep empty) and the PASSED/REJECTED gate decision (FO-LLM) are NOT in-process seams; scoped them out with rationale. See "The pinned deterministic seam".
- DONE: Spike the riskiest unknown FIRST: build the smallest scripted test against that seam and observe it PASS plus a negative control go RED, before committing the design (per the entity-80 spike discipline).
  Built+ran throwaway `TestSpikeFeedbackReflowSeam` in internal/dispatch: positive PASS (~1s, no live agent, real `dispatch.Run`); negative control = REAL production mutation (`if false &&` at build.go:343) → spike RED at the anchored routing + context assertions. Restored build.go, removed spike, baseline `go test ./internal/dispatch/ ./internal/ensigncycle/` green. See "Spike result".
- DONE: Harden AC-1/AC-2 to behavior-first with anchored emit-form assertions (never bare strings.Contains) and a concrete test plan that names the negative control.
  AC-1 now pins the reflow-build property (routing section + routed payload + plain-dispatch contrast) verified by `go test`; AC-2 names TWO negatives — NEG-A (strip the routed payload → context assertion RED) and NEG-B (missing feedback_context → Rule 5 anchored-stderr non-zero exit). Anchored `(?m)^### Feedback from prior review$` form; the heading is emitted once (grep-verified) so the live trap is a dropped payload. Concrete test plan with home, fixture, assertions, cost, and altitude rationale. See "Test plan".

### Summary

Ideation pinned the gate/feedback loop's single byte-observable seam — the `dispatch.Run`
feedback-reflow build that routes the fix request to the `feedback-to` target stage — and
grounded it in a run spike rather than on paper: a real build + anchored routing assertion
PASSED, and a real production mutation (dropping the section-6 emission) turned it RED. The
load-bearing scoping decision: the `### Feedback Cycles` 3-cycle escalation and the PASSED/REJECTED
gate decision are live-LLM behaviors with NO in-process Go seam (grep of `internal/status` for the
cycle section is empty), so asserting them would be either prose-grep or production scope creep;
they stay live-pytest/out-of-scope, consistent with row-15's framing. AC-1/AC-2 are hardened
behavior-first with the anchored `### Feedback from prior review` form and two named negatives
(dropped routed payload; Rule 5 missing-context guard), and the test plan extends the shipped
`internal/ensigncycle` skeleton at the same fixture-only, no-live-LLM altitude.

## Stage Report: implementation

- DONE: Ship `TestFeedbackReflowRoutesFixRequest` in internal/ensigncycle per the Test plan: a real in-process `dispatch.Run` reflow build (is_feedback_reflow=true, concrete feedback_context) dispatched to the gate stage's feedback-to target, asserting the anchored `(?m)^### Feedback from prior review$` routing section + the routed payload present verbatim + a contrast that a plain (non-reflow) dispatch to the SAME target does NOT emit the routing section.
  internal/ensigncycle/feedback_test.go: real dispatch.Run build to the `validation → feedback-to: implementation` target; asserts anchored routing section, verbatim routed payload, and non-reflow contrast. Commit 1840a01.
- DONE: Ship `TestFeedbackReflowGoesRedOnBrokenOutput` with BOTH named negatives as in-test guards: NEG-A (strip the routed feedback_context from the real body → context-presence assertion goes RED) and NEG-B (reflow build with EMPTY feedback_context → dispatch.Run Rule 5 exits non-zero with an anchored stderr match of `dispatching to feedback target stage '{stage}' but feedback_context is missing`).
  NEG-A strips routedFeedback from the real reflow body and proves the context-presence check fails (heading survives, so the context assertion is the guard); NEG-B drives an empty-feedback_context reflow and asserts non-zero exit + anchored Rule 5 stderr. Both verified RED on real build.go mutations (section-6 emission dropped; Rule 5 disabled), then build.go restored pristine (git diff empty).
- DONE: Gates green with REAL captured exit codes ($?, never | tail): `go test ./...`, `go test -race ./internal/ensigncycle/`, `gofmt -l` (empty), `go vet` — all from inside the worktree.
  -race ./internal/ensigncycle/ → 8 passed exit 0; gofmt -l . → empty exit 0; go vet ./... → no issues exit 0. `go test ./...` exit 1 only on `TestCodexResolveManifestAgainstInstalledHost` — confirmed PRE-EXISTING on the stashed-clean baseline (host `codex` CLI config-load failure, package internal/cli, untouched by this test-only change).

### Summary

Shipped the gate/feedback rejection-reflow behavioral coverage (matrix row 15) as two tests in internal/ensigncycle, no production code changes. The seam is the real in-process dispatch.Run reflow build (build.go:225 Rule 5 guard + build.go:343 section-6 routing emission): added a README fixture with a worktree gate stage declaring `feedback-to: implementation` and an entity stamped at the gate stage with a worktree value, mirroring internal/dispatch's readmeWorktree. Both negatives are real guards — verified each goes RED on the matching production mutation and confirmed build.go restored pristine. Extracted readDispatchBodyFromStdout to de-duplicate the build-stdout→body parse shared with stageFixture. The lone `go test ./...` failure is a pre-existing environmental codex-host test, not regressed by this change.

## Stage Report: validation

- DONE: INDEPENDENTLY reproduce BOTH negative controls by re-applying the REAL build.go mutations in the worktree (NOT trusting the implementation report): (a) drop/disable the build.go section-6 routing emission -> confirm TestFeedbackReflowRoutesFixRequest goes RED; (b) disable Rule 5 -> confirm NEG-B no longer gets its non-zero exit. Then restore build.go pristine and confirm `git diff` is empty.
  NEG-A: guarded build.go:343 behind `if false &&` → TestFeedbackReflowRoutesFixRequest went RED at feedback_test.go:117 (anchored routing section) AND :122 (context payload). NEG-B: guarded build.go:225 Rule 5 behind `if false &&` → reflow_with_empty_feedback_context went RED at feedback_test.go:187 (build now exits 0, stdout JSON emitted, instead of non-zero). Restored via `git checkout -- internal/dispatch/build.go` after each; final `git diff` empty, tree clean.
- DONE: Verify each `**AC-N**`: reproduce the 'Verified by' clause for AC-1 (anchored routing section + verbatim routed payload + non-reflow contrast) and AC-2 (NEG-A + NEG-B real in-test guards, anchored emit-form not bare strings.Contains).
  AC-1: read feedback_test.go — TestFeedbackReflowRoutesFixRequest drives real dispatch.Run (build "--workflow-dir") to the `feedback-to: implementation` target; asserts anchored `(?m)^### Feedback from prior review$`, verbatim routedFeedback payload, and a plain-dispatch contrast that does NOT match the heading. AC-2: NEG-A/NEG-B are in-test guards (negatives exercised, not described); routing assertion is the anchored regex (not bare Contains), NEG-B uses anchored `reflowMissingContext` stderr regex. `go test ./internal/ensigncycle/` → 8 passed, exit 0.
- DONE: Gates green with REAL captured exit codes ($?, never | tail) from inside the worktree: `go test ./...`, `go test -race ./internal/ensigncycle/`, `gofmt -l .`, `go vet ./...` — and confirm the ONLY ./... failure is the pre-existing env-only TestCodexResolveManifestAgainstInstalledHost.
  ensigncycle: 8 passed exit 0; -race ./internal/ensigncycle/: 8 passed exit 0; gofmt -l . empty exit 0; go vet ./... no issues exit 0; go test ./... 479 passed / 1 failed exit 1 — sole failure TestCodexResolveManifestAgainstInstalledHost (internal/cli, "failed to load configuration" host codex config-load). Confirmed not a regression: `git diff --stat de5bb44 HEAD` touches ONLY internal/ensigncycle (test-only, zero production files), so the untouched internal/cli test cannot be regressed by this commit.

### Summary

PASSED. Independently reproduced both negative controls by re-applying the real build.go mutations myself (not re-reading the report): dropping the section-6 emission turned TestFeedbackReflowRoutesFixRequest RED at both the anchored routing-section and context-presence assertions; disabling Rule 5 made NEG-B's empty-feedback_context build exit 0 instead of non-zero, turning it RED. Restored build.go pristine after each (git diff empty). Both ACs verified: AC-1's reflow body carries the anchored routing section + verbatim routed payload with a working non-reflow contrast; AC-2's two negatives are real in-test guards using anchored emit-form regexes, not bare substring matches. All gates green; the change is test-only (diff confined to internal/ensigncycle), so the lone `go test ./...` failure (codex-host config-load in internal/cli) is pre-existing and out of scope, not a regression.
