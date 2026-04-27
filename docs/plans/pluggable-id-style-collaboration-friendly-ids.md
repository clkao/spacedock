---
id: 217
title: "Pluggable id-style with collaboration-friendly generated IDs"
status: ideation
source: "GitHub issue #150 (filed by CL, 2026-04-27)"
started: 2026-04-27T21:32:17Z
completed:
verdict:
score: 0.66
worktree:
issue: "#150"
pr:
mod-block:
---

Spacedock should support pluggable `id-style` strategies and add a generated ID style that is better for collaborative workflows than centrally allocated sequential numbers.

## Problem statement

`id-style: sequential` is readable and works well for single-writer workflows, but it makes entity creation a coordinated write. Today `status --next-id` computes the next numeric ID from active plus archived entities. If two people or agents create different task files from stale branches, worktrees, or offline clones, both can legitimately choose the same next number. Git may merge the files cleanly because their slugs differ, but Spacedock now has a semantic duplicate in the `id` field.

Spacedock already treats the filename slug as the operational path key, while the `id` field is the human-facing compact identity shown in status tables, stage reports, dispatch output, PR bodies, and issue references. This task should make that identity policy explicit and pluggable, preserve existing sequential workflows, cover the `id-style: slug` request from GitHub issue #98, and provide a generated style that is safe for concurrent and offline creation without a central reservation step.

## Prior-art findings

- Shortest-unique-prefix identifiers use a prefix index over the set of known keys. Trie/PATRICIA-style structures are established prefix-retrieval structures: a trie stores strings by common prefixes, and PATRICIA compresses sparse trie branches. For Spacedock this maps to "store a full ID, display the shortest prefix that is unique across this workflow." It is compact, but the displayed prefix is contextual and can grow after another branch imports a colliding prefix.
- Git accepts full object IDs or a unique leading substring within the repository. `git rev-parse --short` shortens to a unique prefix, and `git describe --abbrev` can emit a longer suffix as the repository grows. This is strong prior art for accepting unambiguous prefixes and rejecting ambiguous ones, but it also shows the tradeoff: a prefix that was sufficient yesterday may not be sufficient after new objects arrive.
- RFC 9562 UUIDs are 128-bit identifiers that require no central registration. The RFC explicitly frames UUIDs as an alternative when coordinating auto-increment values across distributed systems is burdensome. UUIDv7 adds time-ordering, but any short-prefix display of UUIDv7 must account for its timestamp-heavy left side.
- ULID stores a 48-bit millisecond timestamp plus 80 bits of randomness and uses a 26-character Crockford Base32 string. KSUID stores a 32-bit timestamp plus 128 random bits and uses a 27-character Base62 string. These are useful full-ID formats, but their leading timestamp component means many IDs created near each other share the same early characters, so shortest-prefix display must often include enough random suffix to become useful.
- Nano ID is closer to the stored-short-ID option: choose an alphabet and length, then budget collision risk with birthday-paradox math. Its default 21-character URL-safe ID targets UUID-v4-like collision probability, but Spacedock can choose a shorter human-friendly generated ID because workflows are expected to have far fewer entities than large databases.

Sources checked: [Git rev-parse](https://git-scm.com/docs/git-rev-parse), [Git describe](https://git-scm.com/docs/git-describe), [RFC 9562 UUIDs](https://www.rfc-editor.org/rfc/rfc9562.html), [ULID spec](https://github.com/ulid/spec), [KSUID README](https://github.com/segmentio/ksuid), [Nano ID README](https://github.com/ai/nanoid), [NIST trie definition](https://xlinux.nist.gov/dads/HTML/trie.html), [PATRICIA notes](https://www.allisons.org/ll/AlgDS/Tree/PATRICIA/), [GitHub issue #98](https://github.com/clkao/spacedock/issues/98), and [GitHub issue #150](https://github.com/clkao/spacedock/issues/150).

## Proposed approach

Introduce an `id-style` strategy layer used by the status viewer, commission scaffolding, and first-officer task creation guidance. The initial supported styles should be:

- `sequential`: the current behavior. `id` is required, numeric-looking values continue to sort/display as today, and `status --next-id` returns the next zero-padded number across active plus archived entities.
- `slug`: the issue #98 behavior. `id` is optional; the effective identity is the entity slug. Status output should keep the existing columns for compatibility and render the effective ID as the slug. `status --next-id` should fail with a clear "not applicable for id-style: slug" error because a future slug depends on the entity title, not a central counter.
- `generated`: a new collaboration-friendly style. `id` is required and stores a short generated ID, not a dynamically shortened view of a longer hidden value. `status --next-id` returns a fresh candidate generated from a CSPRNG, checks it against active plus archived entities, and retries on known collisions. Validation is explicit through `status --validate` and fail-fast in status display paths, so duplicate generated IDs cannot be silently presented as valid workflow state.

The recommended generated format is a 12-character Crockford Base32 string normalized to lowercase, using the ULID-style alphabet without visually confusing letters. That gives 60 bits of entropy. Birthday-bound collision probability remains negligible for Spacedock-scale workflows: about 4.3e-9 at 100,000 generated IDs and about 4.3e-7 at 1,000,000 generated IDs, before the local collision check and retry path. A deterministic test hook can force collisions so the implementation proves retry and failure behavior without relying on chance.

At commission time, the captain should choose the style explicitly unless batch mode provides a strong default. The prompt should offer three choices:

- `generated` (recommended for collaborative workflows): use when multiple people or agents may create entities across branches, worktrees, offline edits, or long-running projects. This should be the recommended choice when the workflow has worktree stages, PR/merge mods, team-mode agents, or the captain mentions collaboration/concurrency.
- `sequential` (compatibility/default): use when the workflow is single-writer, small, or needs continuity with existing numeric IDs. This remains the backwards-compatible default when no collaboration signal is present or when the command runs in non-interactive batch mode without an explicit `--id-style`.
- `slug` (canonical filename): use when the slug is already the durable identity, such as named projects, semantically numbered episodes, or workflows with single-digit or low double-digit entity counts. This covers GitHub issue #98.

Refit should not silently change an existing workflow's style. It should report the current `id-style`, recommend `generated` only when collaboration pressure exists, and require explicit captain approval before changing README frontmatter. If entities already exist, refit should require `status --validate` to pass under the target style and should point to manual migration guidance; it should not rewrite entity IDs as part of this task.

Generated README frontmatter examples should be concrete and copyable. Only the `id-style` line changes; the surrounding workflow metadata and stages remain the same shape:

```yaml
---
commissioned-by: spacedock@0.11.0
entity-type: entity
entity-label: task
entity-label-plural: tasks
id-style: sequential
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: done
      terminal: true
---
```

```yaml
---
commissioned-by: spacedock@0.11.0
entity-type: entity
entity-label: task
entity-label-plural: tasks
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: done
      terminal: true
---
```

```yaml
---
commissioned-by: spacedock@0.11.0
entity-type: entity
entity-label: task
entity-label-plural: tasks
id-style: generated
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: done
      terminal: true
---
```

With the recommended first implementation, generated IDs are fixed-length. Display length does not grow when a workflow moves from tens to hundreds to thousands of entities:

| Workflow size | Example generated IDs | Display behavior |
|---------------|-----------------------|------------------|
| 10s of entities | `4k9q2m7x8c3v`, `8t5n0p2w6j9r`, `c3v7k1m9q2x8` | show all 12 characters |
| 100s of entities | `9m2c7v4xq8j3`, `h6t3k9p2w5r8`, `x8q4m2v7c9n5` | show all 12 characters |
| 1000s of entities | `v7k3q9x2m5c8`, `2w8r5t9p3j6n`, `n4c8x2q7m9v5` | show all 12 characters |

A dynamic shortest-prefix display remains a future alternative, not the first generated style. In that variant, Spacedock would store a longer full ID and display only the shortest unique prefix, so a stored value like `4k9q2m7x8c3v9r5t6w2p` might display as `4k9q` in a small workflow, `4k9q2m` after more similar IDs appear, and `4k9q2m7x` if thousands of entities make shorter prefixes ambiguous. That dynamic display would be contextual and could lengthen after branch merges; the recommended fixed 12-character generated ID avoids that moving display target.

## Design comparison

**Full stable ID plus shortest unique display prefix.** This is the Git-like design: store a full UUID/ULID/KSUID/NanoID value and display/accept the shortest unique prefix in the workflow. It minimizes real collision risk and lets the display length grow with the corpus. The downside is collaboration ergonomics: a prefix copied into chat, a PR body, or a stage report can become ambiguous after another branch adds an ID with the same prefix. ULID/KSUID/UUIDv7 also have timestamp-leading strings, so short prefixes are less compact for bursts. This is a good future display mode, but it is not the best first collaboration-friendly storage mode.

**Stored short generated IDs with collision handling.** This is the recommended generated style. The visible ID is the canonical stored ID, so references remain stable after merges. Creation needs no central reservation and no shared state beyond a best-effort local collision check. The risk is probabilistic rather than impossible, so validation must include explicit duplicate detection across active, archived, flat, and folder-form entities. The fixed length is a product choice; 12 Crockford Base32 characters are a good default for readability and collision budget.

**Pluggable `id-style` strategy.** This should wrap both existing sequential behavior and new styles behind the same strategy contract. It avoids special-casing `slug` separately from generated IDs and makes the README frontmatter field meaningful. The status script can remain stdlib-only by using a small in-file registry first; if the behavior grows, it can later move to a helper module without changing the workflow schema.

## Behavior details

The strategy contract should answer these questions for a workflow:

- whether `id` is required in entity frontmatter
- how to compute the effective display ID for a scanned entity
- how `status --next-id` behaves
- how `status --boot` emits `NEXT_ID`
- how references are resolved from exact IDs, ID prefixes, and slugs
- whether active and archived entities participate in uniqueness checks
- how `status --validate` checks duplicate or invalid effective IDs
- what migration checks are required before changing styles

Collision and schema validation should have a named CLI surface. Add `status --validate`, scoped by `--workflow-dir`, which scans active entities plus `_archive/` using the same flat/folder discovery rules as display and mutation paths. It exits 0 and prints `VALID` when the workflow is internally consistent. It exits 1 and prints one stderr line per problem when it finds a duplicate effective ID, a missing required `id`, an invalid generated ID alphabet/length, a non-numeric sequential ID that would affect allocation, or any flat/folder conflict that changes the canonical entity path. Each error should name the workflow path, active or archived scope, slug, effective ID, and canonical file path. Default status output, `--archived`, `--next`, `--next-id`, and `--boot` should run the same validation before printing normal output; on validation failure they should exit 1 without printing a partial status table or boot payload. `--set` and `--archive` should continue to use active-only mutation resolution, but after a successful mutation they should leave any subsequent validation failure visible to the caller rather than masking it.

`--boot` should keep the `NEXT_ID:` line for first-officer compatibility, but its meaning becomes strategy-dependent and should be accompanied by an `ID_STYLE: {style}` line. For `sequential`, `NEXT_ID: 005` keeps the current highest-active-or-archived-plus-one behavior. For `generated`, `NEXT_ID: {candidate}` emits the same kind of collision-checked generated candidate as `status --next-id`; it is informational and not a reservation, so the first officer must call `status --next-id` again immediately before creating a new entity. For `slug`, `--boot` succeeds and emits `NEXT_ID: n/a (id-style: slug)`, while direct `status --next-id` exits non-zero with the same not-applicable message because the eventual ID is the slug derived from the title.

First-officer task creation should use the strategy explicitly. Under `sequential`, the FO calls `status --next-id` immediately before writing the entity and stores that value in `id`. Under `generated`, the FO also calls `status --next-id` immediately before writing the entity and stores that generated value in `id`; if `status --validate` or a subsequent display command reports a collision after a branch/worktree merge, the FO refreshes workflow state, calls `--next-id` again, and rewrites only the new unmerged entity before committing. Under `slug`, the FO derives the slug from the title, omits `id` or leaves it blank according to the workflow template, and never calls `--next-id` for creation.

Reference resolution should be implemented as a status CLI helper, not runtime-only prose. Add `status --resolve <ref>` as the canonical resolver used by FO logic and available to operators. With `--workflow-dir`, resolution is limited to one workflow. With `--root`, it discovers workflows and requires either an unambiguous match or a workflow qualifier of the form `{workflow-basename}::{ref}`; if two workflows share a basename, the qualifier must be an absolute workflow path followed by `::{ref}`. The resolver prints a single machine-readable line containing `workflow=`, `scope=active|archived`, `slug=`, `id=`, and `path=`, or exits 1 with all candidates listed on stderr.

Within one workflow, resolver precedence should preserve current slug-based operations: exact active slug wins over ID/prefix matches unless the caller uses `id:{value}` or `slug:{value}`. Exact effective ID resolves next. For `generated`, `prefix:{value}` or a bare value of at least four generated-ID characters may resolve as a unique ID prefix; ambiguous prefixes fail and list all matching slugs/IDs. `--set` and `--archive` should route their entity argument through the same resolver in active-only scope, so they can accept slug, exact ID, or unique generated prefix, but they must refuse archived matches with an "archived entity is read-only" error. `status --resolve --archived <ref>` is read-only and includes archived entities; if the same reference matches both active and archived entities, it fails unless the caller qualifies with `active:{ref}` or `archive:{ref}`.

Folder-form entities should use the existing discovery rule: `{slug}/index.md` is the canonical entity file, and folder form wins over a same-slug flat file with a warning. Archived folder entities live under `_archive/{slug}/index.md`. Every validation check, `--next-id` check, `--boot` candidate, status display, and resolver should use the same discovery path so flat, folder, active, and archived entities cannot disagree.

Cross-workflow references are not globally unique. A resolver operating above one workflow must require `--workflow-dir`, an unambiguous discovered workflow, or the `{workflow}::{ref}` qualifier described above. If two workflows contain the same slug, exact ID, or generated prefix, the unqualified `--root ... --resolve <ref>` form must fail and list candidate workflow directories. This avoids treating generated IDs as globally meaningful when Spacedock's workflow directory is the real namespace.

Migration is intentionally not an executable rewrite feature in this task. Existing workflows with `id-style: sequential` require no data rewrite. New workflows may choose `slug` or `generated` at commission time. The deliverable should document manual migration guidance and provide validation that makes manual migration safe: duplicate effective IDs, invalid generated IDs, missing required IDs, flat/folder conflicts, and active/archive conflicts are caught before normal status output. Sequential-to-generated rewrite automation, `legacy-id` population, and bulk README/entity frontmatter changes should be a separate tracked task or refit enhancement, not part of this implementation.

Backward compatibility matters more than making the new abstraction elegant. Existing status output, `--next`, `--boot`, `--where`, `--set`, `--archive`, and first-officer task creation should continue to work for sequential workflows. The phrase "next ID" can remain the CLI name for compatibility, but the docs and runtime prompts should stop saying it is always sequential once strategies exist.

## Acceptance criteria

**AC-1 - Workflows have a documented `id-style` strategy contract covering `sequential`, `slug`, and `generated`.**
Verified by: static tests over `skills/commission/SKILL.md`, `docs/plans/README.md` template text, and first-officer runtime references confirm the three styles, required/optional `id` semantics, and `--next-id` behavior are documented without calling all IDs sequential.

**AC-2 - Existing sequential workflows keep their current behavior.**
Verified by: existing and updated `tests/test_status_script.py::TestNextIdOption` fixtures show `status --next-id` returns the next zero-padded numeric ID across active and archived entities, and `make test-static` passes without changing current sequential output expectations except where wording is intentionally generalized.

**AC-3 - `id-style: slug` treats the slug as the effective ID and makes `status --next-id` non-applicable.**
Verified by: status-script fixtures with omitted `id` fields show default and `--next` tables render a usable effective ID from the slug, `--where` still works, and `status --next-id` exits non-zero with a clear slug-style message.

**AC-4 - `id-style: generated` stores stable short generated IDs and generates new candidates without central allocation.**
Verified by: status-script tests run `status --next-id` under generated style with a deterministic RNG hook, assert the output matches the configured generated alphabet/length, assert active plus archived collisions are retried, and assert forced retry exhaustion fails loudly.

**AC-5 - Invalid or duplicate effective IDs are caught before status output is trusted.**
Verified by: `tests/test_status_script.py` fixtures for active flat files, active `{slug}/index.md`, archived `.md` files, and archived `{slug}/index.md` show `status --validate` exits 1 with slug/id/path evidence for duplicate generated IDs, invalid generated IDs, missing required IDs, and sequential allocation conflicts; default status output, `--archived`, `--next`, `--next-id`, and `--boot` fail before printing partial normal output on the same invalid fixtures.

**AC-6 - `--boot` and first-officer task creation are strategy-aware.**
Verified by: boot tests show `ID_STYLE:` plus `NEXT_ID:` for sequential, generated, and slug workflows; generated boot output contains a valid generated candidate, slug boot output contains `NEXT_ID: n/a (id-style: slug)`, direct slug `--next-id` exits non-zero, and static first-officer reference tests show creation calls `--next-id` for sequential/generated but not for slug.

**AC-7 - Entity reference resolution is deterministic through `status --resolve`.**
Verified by: resolver tests cover exact slug precedence over ID/prefix, `id:` and `slug:` forced qualifiers, exact ID, unique generated prefix, ambiguous generated prefix, unknown reference, active-only `--set`/`--archive` resolution, archived-only mutation refusal, read-only `--resolve --archived`, and `active:`/`archive:` disambiguation, with assertions on the printed path or candidate-list error.

**AC-8 - Cross-workflow ambiguity is rejected rather than guessed.**
Verified by: temp-root fixtures with two discovered workflows containing the same slug, exact ID, or generated prefix show `--root ... --resolve <ref>` fails with both workflow paths, while explicit `--workflow-dir`, `{workflow-basename}::{ref}`, or absolute-path workflow qualifiers resolve the intended entity.

**AC-9 - Migration scope is documented and validation-backed without shipping a rewrite helper.**
Verified by: static docs tests confirm this task documents manual migration guidance and explicitly defers rewrite automation; validation fixtures prove duplicate effective IDs, invalid generated IDs, missing IDs, archived/folder conflicts, and flat/folder conflicts are reported without modifying README or entity frontmatter.

**AC-10 - Concurrent or offline task creation is safer under generated IDs than under sequential IDs.**
Verified by: a filesystem-level test simulates two isolated creators adding different entity files from the same starting workflow; generated style yields distinct IDs without a shared counter, while a forced generated-ID duplicate is detected by `status --validate` and by fail-fast default/`--next`/`--boot` validation before status output can silently present duplicate effective IDs.

**AC-11 - Commission/refit documentation preserves concrete style-selection and generated-ID examples.**
Verified by: static docs/template tests assert the commission flow offers `generated`, `sequential`, and `slug` with the documented recommendations; README frontmatter examples for all three styles include the exact `id-style:` values; generated-ID examples for 10s, 100s, and 1000s of entities are 12-character lowercase Crockford Base32 strings; and docs distinguish fixed generated display from the future dynamic-prefix alternative.

## Test plan

Most proof should be offline unit and fixture tests because the behavior is local file parsing, ID generation, and deterministic reference resolution. Live E2E tests are not required unless implementation also changes first-officer runtime dialogue around filing new tasks.

- Add focused status-script tests in `tests/test_status_script.py` for README `id-style` parsing, `--next-id` delegation, generated collision retry, slug-style non-applicability, `--boot` `ID_STYLE`/`NEXT_ID` output, `--validate`, `--resolve`, archived entities, and folder-form entities. Estimated cost: medium to high; the file already has helpers for active, archived, boot, and folder-form workflows.
- Add fail-fast validation tests proving default status output, `--archived`, `--next`, `--next-id`, and `--boot` refuse invalid generated/sequential/slug effective-ID state before printing normal output. Estimated cost: medium.
- Add static content tests in `tests/test_agent_content.py` and commission-template tests so docs and runtime references no longer hard-code "sequential" where they mean strategy-dependent "new ID", and so FO creation behavior is documented for sequential, generated, and slug workflows. Estimated cost: low.
- Add migration-scope tests as static docs checks plus status validation fixtures only. This task should not test README/entity rewrite automation because it does not ship an executable migration helper. Estimated cost: low.
- Add one concurrency simulation using temporary directories or branches, not live agents. It should copy the same starting workflow into two isolated locations, create entities under generated style, merge/copy the files into one workflow, and run `status --validate`, default status, `--next`, and `--boot` checks. Estimated cost: medium.
- Add static docs/template tests for the style-selection prompt, the three README frontmatter examples, the 10s/100s/1000s generated-ID examples, and the note that fixed generated IDs do not shorten or grow by workflow size. Estimated cost: low.
- Run `uv run pytest tests/test_status_script.py -q` during implementation for the core behavior, then `make test-static` before validation. If first-officer live creation prompts change materially, add a small runtime fixture or transcript-level test before considering expensive live E2E.

## Stage Report: ideation

- DONE: Prior-art research covers shortest-unique-prefix identifiers, Git abbreviated hashes, UUID/ULID/KSUID prefix variants, NanoID-style collision budgeting, and maps them to Spacedock collaboration constraints.
  Evidence: Prior-art section cites trie/PATRICIA, Git `rev-parse`/`describe`, RFC 9562, ULID, KSUID, and Nano ID, then maps each to Spacedock tradeoffs.
- DONE: Proposed design compares full stable ID plus shortest unique display prefix, stored short generated IDs with collision handling, and a pluggable `id-style` strategy that can also cover the `slug` style from GitHub issue #98.
  Evidence: Design comparison and proposed approach recommend stored short generated IDs while defining `sequential`, `slug`, and `generated` strategies.
- DONE: Acceptance criteria and test plan specify concrete behavior for reference resolution, `status --next-id`, archived and folder-form entities, cross-workflow ambiguity, migrations, and safer concurrent/offline task creation.
  Evidence: AC-2 through AC-9 and the test plan cover the requested behaviors with fixture-level verification paths.

### Summary

The ideation output recommends a strategy layer over the existing `id-style` field and a 12-character Crockford Base32 generated style for distributed task creation. It keeps sequential workflows compatible, models issue #98 as the `slug` strategy, and defines resolver, archive, folder-form, cross-workflow, and migration behavior at a testable level.

## Stage Report: ideation (repair)

- DONE: Collision validation surface and behavior are explicit and testable across active/archived flat/folder entities.
  Evidence: Behavior details now define `status --validate`, fail-fast display paths, error evidence, and active/archived flat/folder coverage; AC-5 and AC-10 test it.
- DONE: `--boot` / `NEXT_ID` and FO task creation behavior are defined for sequential, slug, and generated styles.
  Evidence: Behavior details now define `ID_STYLE`, strategy-specific `NEXT_ID`, direct `--next-id`, and FO creation rules; AC-6 tests it.
- DONE: Resolver and migration scopes are concrete, internally consistent, and reflected in acceptance criteria/tests.
  Evidence: Behavior details now define `status --resolve`, qualifiers, precedence, scope, and migration as docs plus validation only; AC-7 through AC-9 test those scopes.

### Summary

The repair makes the implementation surfaces concrete instead of relying on runtime prose: validation is `status --validate`, resolution is `status --resolve`, and boot output remains parseable through strategy-aware `NEXT_ID`. Migration scope is narrowed to documentation and validation support, leaving rewrite automation for a separate task.

## Stage Report: ideation (repair 2)

- DONE: README frontmatter selection and recommendation examples are concrete enough for a user and implementation worker to follow.
  Evidence: Proposed approach now includes commission/refit selection rules plus copyable README frontmatter examples for `sequential`, `slug`, and `generated`.
- DONE: Sample IDs for 10s, 100s, and 1000s of entities are shown and tied to the chosen fixed/generated or dynamic-prefix design.
  Evidence: Proposed approach now shows fixed 12-character generated IDs for each workflow size and distinguishes dynamic shortest-prefix display as a future alternative.
- DONE: Acceptance criteria or test plan are updated if needed so these examples are preserved by implementation/docs tests.
  Evidence: Added AC-11 and a static docs/template test-plan bullet for style-selection guidance, frontmatter examples, generated-ID examples, and fixed-vs-dynamic display behavior.

### Summary

This repair adds concrete operator-facing examples for choosing `id-style` during commission/refit and for how generated IDs look at different workflow sizes. The chosen first design remains fixed 12-character generated IDs; dynamic prefix growth is documented only as a future display alternative.
