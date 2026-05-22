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

5. **The mdschema YAML artifacts exist and validate the corpus.** At minimum `schemas/workflow-readme.mdschema.yml` and `schemas/entity.mdschema.yml` are committed alongside the prose spec. A `validate-corpus` script (shell, Python, or Go) parses every entity in `docs/plans/` against the entity mdschema and the README against the README mdschema. The validator distinguishes `fail` (canonical type/pattern violation) from `warn` (runtime-coerced or conventional-set deviation).
   - **Test:** ≥95% of active entities pass at `fail` severity; warnings are enumerated but do not gate the floor. Archive failures are acceptable if they pre-date later schema additions and are documented.

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
- Builds on: commit `410a0731` v0 draft (recovered onto main as `148dfba5` at `docs/superpowers/specs/2026-05-12-spacedock-frontmatter-contract-spec.md`)
- Unblocks: sub-project #2 (launcher — partially), sub-projects #3 (status port) and #4 (claude-team port) — both require this spec to define their conformance target
- Estimated complexity: medium. Most of the work is mechanical (corpus walk, TBD resolution, mdschema authoring). The judgment calls are about scope cuts and the back-compat policy.
- Cost estimate: ~$10-20 in agent budget across ideation + implementation + validation. No live-claude E2E required.

## Design

The v0 draft on main (`docs/superpowers/specs/2026-05-12-spacedock-frontmatter-contract-spec.md`) is the working spec. This Design section records the design decisions reached during ideation: TBD resolutions, the mdschema artifact shapes, and the validation strategy. At implementation time these resolutions are folded into the spec body and the schema YAML files are committed under `schemas/`.

### TBD resolutions

The v0 draft enumerates six TBDs in its closing catalog. Dispositions:

1. **Stage subsection bullets — strict vs convention.** RESOLVED: spec mandates `Outputs:` only; `Inputs:`, `Good:`, `Bad:` are conventional. Rationale: `claude-team build` already loadbearingly extracts `Outputs:` into per-dispatch checklist items; the others have no machine consumer and exist for human authoring quality. Treating them as required would reject older READMEs without ROI. Validator emits a `warn` (not `fail`) when any of the conventional three are missing.

2. **Reserved namespaces for custom entity fields.** DEFERRED to follow-up. The spec explicitly states there are no reserved prefixes today and that the captain may add any non-canonical field. A future entity (filed at implementation time if not already) would introduce `x-*` reservation when a real conflict appears. Matches the README's "deferred" list.

3. **Frontmatter parser hardening — line-oriented vs full YAML.** RESOLVED: keep the line-oriented hand-rolled writer (preserves observed formatting, has bug-fix history around inline comments and fence detection); permit the Go read-path to use `gopkg.in/yaml.v3` provided it produces the same key set as the line-oriented parser on the active corpus. AC-5's corpus validation is the regression net.

4. **mdschema YAML files authored in implementation.** RESOLVED: AT IDEATION the schemas are drafted as worked examples (see `### mdschema artifact shapes` below). The committed `schemas/*.mdschema.yml` files are produced during implementation; the ideation gate validates field coverage against the worked example, not against committed schemas.

5. **Stage Report body — spec-validated vs prose convention.** DEFERRED, per the entity body's `## Out of scope`. Rationale: the ensign skill's prose contract is the canonical surface; spec-validating duplicates and creates a second source of truth. If a real malformed-report bug appears, file a follow-up.

6. **Per-version migration rules.** RESOLVED-AS-MONOTONIC for v1.0: the back-compat policy is "additions and optional fields allowed; removals and type changes forbidden until major-version bump." No migration logic is committed at v1.0. When a removal becomes necessary, a v2.0 bump is required and migration rules are designed then.

### mdschema artifact shapes

Two worked-example shapes follow. These are the design targets for the YAML files that land at implementation. Field coverage was validated against today's `docs/plans/` corpus (entities + README) — see `### Corpus field coverage` below.

#### `schemas/workflow-readme.mdschema.yml` (worked example)

```yaml
version: 1.0
target: workflow_readme
applies_to:
  filename: README.md
  required_at: workflow_root
frontmatter:
  strict_canonical: true
  required:
    - commissioned-by
    - entity-type
    - entity-label
    - entity-label-plural
    - id-style
    - stages
  optional:
    - mission
  fields:
    commissioned-by:
      type: string
      pattern: '^spacedock@([0-9]+\.[0-9]+\.[0-9]+|)$'
    entity-type:
      type: string
      pattern: '^[a-z][a-z0-9_]*$'
    entity-label:
      type: string
    entity-label-plural:
      type: string
    id-style:
      type: string
      enum: [sequential, sd-b32, slug]
    mission:
      type: string
    stages:
      type: mapping
      required: [states]
      optional: [defaults, transitions]
      shape:
        defaults:
          type: mapping
          fields:
            worktree: { type: boolean, default: false }
            concurrency: { type: integer, default: 2 }
            model: { type: string, enum: [sonnet, opus, haiku] }
        states:
          type: sequence
          min_items: 2
          item:
            type: mapping
            required: [name]
            fields:
              name: { type: string, pattern: '^[a-z0-9][a-z0-9-]*[a-z0-9]$' }
              initial: { type: boolean }
              terminal: { type: boolean }
              worktree: { type: boolean }
              concurrency: { type: integer, minimum: 1 }
              gate: { type: boolean }
              fresh: { type: boolean }
              feedback-to: { type: string }
              agent: { type: string }
              model: { type: string, enum: [sonnet, opus, haiku] }
        transitions:
          type: sequence
          item:
            type: mapping
            required: [from, to]
            fields:
              from: { type: string }
              to: { type: string }
              label: { type: string }
body:
  required_sections_per_stage:
    heading: '### `{stage_name}`'
    accept_bare: true
    accept_annotated: true
    required_bullets:
      - 'Outputs:'
    conventional_bullets:
      severity: warn
      list:
        - 'Inputs:'
        - 'Good:'
        - 'Bad:'
invariants:
  - id: initial_cardinality
    rule: 'exactly one stage with initial:true AND it is the first list entry'
  - id: terminal_cardinality
    rule: 'exactly one stage with terminal:true AND it is the last list entry'
  - id: stage_subsection_present
    rule: 'every stages.states[].name has a matching ### heading in body'
  - id: feedback_target_valid
    rule: 'feedback-to value (when present) names another stage in states[]'
```

#### `schemas/entity.mdschema.yml` (worked example)

```yaml
version: 1.0
target: entity
applies_to:
  filename_pattern: '<slug>.md or <slug>/index.md'
  required_at: workflow_root_or_archive
frontmatter:
  strict_canonical: true
  permissive_additions: true
  always_present:
    - id
    - title
    - status
    - score
    - source
    - worktree
  optional_canonical:
    - pr
    - started
    - completed
    - verdict
    - mod-block
    - archived
    - issue
  fields:
    id:
      type: string
      conditional:
        - when: { workflow.id-style: sequential }
          rule: 'non-negative integer rendered zero-padded'
        - when: { workflow.id-style: sd-b32 }
          rule: '24-char Spacedock-Base32 (lowercase a-z + 2-7)'
          pattern: '^[a-z2-7]{24}$'
        - when: { workflow.id-style: slug }
          rule: 'empty string'
    title: { type: string }
    status:
      type: string
      sentinel_on_unknown: 99
      should_match: workflow.stages.states[].name
    score: { type: numeric_string, coerce_empty: last, coerce_invalid: 0 }
    source: { type: string }
    worktree: { type: string }
    pr: { type: string }
    started: { type: iso8601 }
    completed: { type: iso8601 }
    verdict: { type: string, conventional: [PASSED, REJECTED] }
    mod-block:
      type: string
      pattern: '^[^:]+:[^:]+$'
      semantics: 'when non-empty, terminal-advancement is refused'
    archived: { type: iso8601 }
    issue: { type: string }
  custom_fields:
    policy: preserve_unknown
    no_reserved_prefix: true   # see TBD-2 deferral
body:
  required_opening: 'problem-statement paragraph before any heading'
  recognized_sections:
    - '## Acceptance criteria'
    - '## Test plan'
    - '## Stage Report: <stage_name>'    # not body-validated (TBD-5 deferred)
    - '### Feedback Cycles'              # FO-owned
invariants:
  - id: id_uniqueness
    rule: 'per id-style, ids unique across active + archived'
  - id: slug_uniqueness
    rule: 'slugs unique across active + archived'
  - id: mod_block_guard
    rule: 'mod-block non-empty AND status terminal → refuse without --force'
  - id: merge_hook_invariant
    rule: 'workflow has any _mods/*.md with `## Hook: merge` AND pr empty AND mod-block empty → refuse terminal without --force'
  - id: two_step_audit
    rule: 'single --set cannot both clear mod-block and advance to terminal'
  - id: pr_mirror
    rule: 'writing pr on worktree-backed entity also writes pr to canonical copy; no other field mirrors'
```

#### Validation pass / fail fixtures

A passing entity (matches `schemas/entity.mdschema.yml` v1.0):

```yaml
---
id: bxntyscd4sgxxdar9xty4nnt
title: "Spacedock frontmatter & state-machine contract spec"
status: ideation
score:
source: "Sub-project #1 of the Go port roadmap…"
worktree:
---
```

Validator output:

```
docs/plans/spacedock-frontmatter-contract-spec.md: PASS
```

A failing entity (canonical type violation + unknown stage):

```yaml
---
id: 042
title: "Bad example"
status: not-a-real-stage
score: not-a-number
source: ""
worktree:
verdict: MAYBE
---
```

Validator output:

```
docs/plans/bad-example.md: FAIL
  - id: violates id-style=sd-b32 pattern '^[a-z2-7]{24}$' (got '042')
  - status: 'not-a-real-stage' does not match any stages.states[].name (sentinel-99 at runtime; warned here)
  - score: 'not-a-number' is not a numeric_string (coerced to 0 at runtime; warned here)
  - verdict: 'MAYBE' is not in conventional set [PASSED, REJECTED] (warn, not fail — conventional only)
```

Severity rules: canonical-type pattern mismatches are `fail`; runtime-coerced or conventional-set deviations are `warn`. AC-5's "≥95% pass" floor counts `fail`-level failures only.

### Corpus field coverage check

Sampled `docs/plans/*.md` + `docs/plans/_archive/*.md` (commit `148dfba5` time) yields the union of top-level frontmatter keys: `archived, blocked-on, blocked-reason, commissioned-by, completed, depends, entity-label, entity-label-plural, entity-type, id, id-style, issue, mod-block, pr, score, source, stages, started, status, title, verdict, worktree`. Of these:

- `commissioned-by, entity-label, entity-label-plural, entity-type, id-style, stages` are README-only fields seen because the README is the same `*.md` shape (the validator dispatches on filename — README vs entity).
- All remaining keys are canonical or canonical-optional in the entity schema above.
- `blocked-on, blocked-reason, depends` are captain-added custom fields (`docs/plans/codex-completion-notifications-must-preempt-side-discussion.md`, `docs/plans/_archive/github-issue-pr-workflow.md`, `docs/plans/_archive/simplify-first-officer.md`). The schema's `custom_fields.policy: preserve_unknown` accepts them without warning.

No silent fields. AC-1's set-difference check is empty at v1.0.

### Validation strategy

- `mdschema-validate` (the implementation-step script) is dispatch-by-filename. It walks `docs/plans/**/*.md`, treats `README.md` as workflow-readme target, everything else as entity target. `_archive/**` is walked the same way.
- The Go port (sub-project #3, `status` port) embeds the entity schema as a build-time asset and validates on `--set` write. The CLI prints validation warnings to stderr but only refuses writes for `fail`-level violations on canonical type/pattern fields. This preserves backward compatibility with today's permissive Python writer.
- The Python implementation (`skills/commission/bin/status`) gains a `--validate` flag at implementation time that runs the same schema against an existing entity without mutating it. AC-6's pytest case is roundtrip: seed entity via `--set`, mutate, call `--validate`, assert PASS.
- CI runs `mdschema-validate` over the full corpus on every PR touching `docs/plans/`, `docs/superpowers/specs/`, `schemas/`, or `skills/commission/bin/status`.

### Stage Reports body — explicit non-validation

Per TBD-5 disposition, the schema body section recognizes `## Stage Report: <stage_name>` as a known heading but performs no structural validation on its contents. The ensign skill's `ensign-shared-core.md` `## Stage Report Protocol` section is the canonical contract. Future tightening, if needed, gets its own follow-up entity.
