---
id: bxntyscd4sgxxdar9xty4nnt
title: "Spacedock frontmatter & state-machine contract spec (Go port sub-project #1)"
status: ideation
source: "Sub-project #1 of the Go port roadmap at docs/superpowers/specs/2026-05-12-spacedock-go-port-roadmap.md. A v0 draft was committed on 2026-05-12 (commit 410a0731 'spec(v0): spacedock frontmatter + state machine contract') but accidentally landed on the wip/fake-star-spike branch instead of main and never came back. The v0 file is at docs/superpowers/specs/2026-05-12-spacedock-frontmatter-contract-spec.md on that wip branch (374 lines, multiple inline TBDs). This entity drives v0 → v1: resolve TBDs, write the machine-checkable mdschema artifacts, prove conformance against today's corpus, and pull the doc onto main."
started: 2026-05-22T23:10:56Z
completed:
verdict:
score:
worktree:
---

# Spacedock frontmatter & state-machine contract spec — sub-project #1 of Go port

This entity finishes sub-project #1 of the Go port: a versioned, machine-checkable contract for the Spacedock workflow's frontmatter (README + entity) and stage state machine. The contract is the prerequisite for sub-projects #3 (status port) and #4 (claude-team port) — the Go implementations read/write against this spec, not against current Python behavior alone.

## Problem

The Spacedock workflow's frontmatter and state machine are documented only by reading the Python implementation. There is no contract a reimplementation can be reviewed against. The `commissioned-by: spacedock@0.11.1` stamps imply a versioned contract; nothing makes it explicit.

A v0 draft (~374 lines, commit `410a0731` on `wip/fake-star-spike`) exists with inline TBDs and a stubbed "Shape (machine-checkable artifacts — TBD)" section. The mdschema YAML files referenced from that section were never authored. The draft commit is orphaned on a parked WIP branch.

Until this entity ships, the Go port (sub-projects #3 and #4) cannot be reviewed against anything more durable than "matches the current Python."

## Proposed approach

1. **Recover the v0 draft onto main.** Cherry-pick or rewrite commit `410a0731` from `wip/fake-star-spike` so the v0 file lives on main as a working draft. Do not merge the rest of the WIP branch (it has unrelated fake-star spike work).
2. **Close every TBD in the v0 catalog.** Resolve each inline `**TBD:**` or move it to a named follow-up entity with one-line deferral rationale. The TBDs catalog at the tail of the draft enumerates them.
3. **Write the mdschema artifacts.** Author at minimum `mdschema-readme.yaml` (workflow README contract) and `mdschema-entity.yaml` (entity file contract). Commit alongside the prose spec.
4. **Build a `validate-corpus` script.** ~30 lines, validates every entity in `docs/plans/` against the entity mdschema and the README against the README mdschema. Output is per-file pass/fail with reasons.
5. **Round-trip the current Python.** Add a pytest case that round-trips an entity through `status --set` and re-validates against the mdschema.
6. **Promote to v1.** Once TBDs are closed, mdschemas validate the corpus, and the Python conforms, bump the version stamp on the spec from `v0` to `v1`.

## Acceptance criteria

End-state properties of the finished entity. Each AC is testable inside this entity's own deliverables.

1. **Field coverage is exhaustive against today's corpus.** Every frontmatter key that appears in `docs/plans/README.md` `stages:` block, in any entity under `docs/plans/`, `docs/plans/_archive/`, or in any other commissioned workflow under `docs/superpowers/specs/` is documented in the spec with type + required/optional + one-sentence semantics. No silent fields.
   - **Test:** discovery script walks the corpus, extracts the set of all unique frontmatter keys, intersects against the documented-fields list in the spec. Set difference is empty, or has an explicit allowlist with rationale.

2. **State machine matches today's `stages.states`.** Every stage declared in `docs/plans/README.md` `stages.states` appears in the spec's state-machine section with its declared properties (`gate`, `worktree`, `fresh`, `feedback-to`, `terminal`, `initial`, `concurrency`); every legal transition is named; the semantics of `gate`, `worktree`, and `feedback-to` are defined normatively in ≤2 sentences each.
   - **Test:** static cross-reference between the README's `stages:` block and the spec's state-table. Row-by-row equivalence; each declared property is defined.

3. **TBD catalog is closed.** Every inline `**TBD:**` from the v0 draft is either (a) resolved and folded into the relevant section, or (b) explicitly deferred to a named follow-up entity with a one-line rationale.
   - **Test:** `grep -c '^\*\*TBD:' spec.md` returns 0 inside the body; the closing "Resolved / Deferred" subsection lists each former TBD with its disposition.

4. **Versioning + back-compat clause is explicit.** The spec declares its version (`v1.0` or `version: 1.0` at the top), names a back-compat policy (allowed: additions, optional fields; forbidden: removals, type changes; until a major-version bump), and the policy is testable.
   - **Test:** static check for the version header + the back-compat clause naming allowed and forbidden changes.

5. **The mdschema YAML artifacts exist and validate the corpus.** At minimum `mdschema-readme.yaml` and `mdschema-entity.yaml` are committed alongside the prose spec. A `validate-corpus` script (shell, Python, or Go) parses every entity in `docs/plans/` against the entity mdschema and the README against the README mdschema.
   - **Test:** ≥95% of active entities pass; failures are enumerated with reasons. Archive failures are acceptable if they pre-date later schema additions and are documented.

6. **Current Python implementation conforms.** Today's `skills/commission/bin/status` and `skills/commission/bin/claude-team` read and write frontmatter that the mdschemas accept. The spec is descriptive, not aspirational.
   - **Test:** pytest case that creates a test entity via `status --set`, applies one mutation, and re-validates against the spec mdschema. Divergence is either a bug in the implementation (file it) or a spec gap (fix the spec). PR cannot merge with both diverging.

## Test plan

- **Static doc checks:** version header present, back-compat clause present, TBD count is 0, mdschema files referenced from the spec are committed.
- **Corpus validation:** the `validate-corpus` script is the load-bearing test. Runs in CI; output is reproducible.
- **Implementation roundtrip:** one pytest case under `tests/` exercises the `status --set` write path and re-validates output. Lives next to existing `tests/test_status.py` or equivalent.
- **No live E2E required.** This entity is documentation + machine-checkable artifacts; runtime behavior is not the claim.

## Out of scope

- **The Go port itself.** Sub-projects #2 (launcher binary), #3 (status port), #4 (claude-team port) are separate entities. This entity only ships the contract.
- **Stage Report body validation.** v0's TBD on whether to spec-validate `## Stage Report` body structure is deferred — keep it as ensign-side prose convention.
- **Reserved namespaces for custom frontmatter fields.** v0's TBD about reserving `x-*` prefixes is deferred; adopt only when there's a real conflict.

## Risks

### Risk A — corpus divergence

Active entities may already diverge from a strict spec interpretation (e.g., older entities omit fields added later). AC-5 sets a ≥95% pass floor and requires enumerated failures, so a small divergence is acceptable as long as it's visible and explained.

### Risk B — Python and spec drift

If the spec locks behavior the current Python doesn't actually deliver, AC-6's roundtrip pytest will fail. The discipline is: fix the spec or fix the implementation, never carry both diverging into merge.

### Risk C — over-coverage

Pulling in too much (e.g., Stage Report body validation, reserved namespaces) bloats the contract for thin value. The "Out of scope" section above is the scope cut. Ideation should not expand scope without captain approval.

## Scale context

- Spacedock version: 0.12.0+
- Builds on: commit `410a0731` v0 draft (on `wip/fake-star-spike`, to be recovered onto main)
- Unblocks: sub-project #2 (launcher — partially), sub-projects #3 (status port) and #4 (claude-team port) — both require this spec to define their conformance target
- Estimated complexity: medium. Most of the work is mechanical (corpus walk, TBD resolution, mdschema authoring). The judgment calls are about scope cuts and the back-compat policy.
- Cost estimate: ~$10-20 in agent budget across ideation + implementation + validation. No live-claude E2E required.
