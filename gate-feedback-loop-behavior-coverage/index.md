---
id: m6h1tyyen9z9g7cm830wcpzq
title: Behavioral coverage for the gate/feedback approval loop (matrix row 15)
status: ideation
source: FO/captain (2026-05-31) — comprehensive-behavior-coverage sprint; coverage matrix row 15 (largest pure GAP)
started: 2026-05-31T18:06:58Z
completed:
verdict:
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

## Acceptance criteria (provisional — ideation hardens)

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 — A behavioral test exercises the gate/feedback rejection-reflow deterministic seam.**
Verified by: a new test (extending the `internal/ensigncycle` scripted-ensign pattern, or a
sibling package) drives the real `dispatch.Run` feedback-reflow build (`is_feedback_reflow: true`)
and/or the `### Feedback Cycles` record, asserting the mechanical outputs of the routed fix
request — runnable in `go test`.

**AC-2 — The test FAILS when the gate/feedback mechanical output breaks, via anchored forms.**
Verified by: a negative control (regress the production seam — e.g. drop the routed assignment,
or break the reflow body's target-stage line) turns the test RED; the assertion is an anchored
emit-form regex, not a bare `strings.Contains` a prose warning could satisfy.

## Out of scope
- Live-LLM gate compliance (does a *real* FO/ensign honor the loop) — stays live-pytest /
  deferred to a live-runtime harness (matrix rows 16/17/CI).
- The merge-hook guardrail (this workflow has no merge hooks).

## Notes
Test-infra surface (extends `internal/ensigncycle` or a sibling). Likely disjoint from the
prose-grep retirement entity (`harden-integration-prose-grep-tests`), which lives in
`skills/integration/` — the two can run in parallel worktrees.
