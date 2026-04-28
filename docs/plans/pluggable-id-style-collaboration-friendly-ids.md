---
id: 217
title: "Pluggable id-style with collaboration-friendly generated IDs"
status: validation
source: "GitHub issue #150 (filed by CL, 2026-04-27)"
started: 2026-04-27T21:32:17Z
completed:
verdict:
score: 0.66
worktree: .worktrees/spacedock-ensign-pluggable-id-style-collaboration-friendly-ids
issue: "#150"
pr:
mod-block: merge:pr-merge
---

Spacedock should support pluggable `id-style` strategies and add a generated ID style that is better for collaborative workflows than centrally allocated sequential numbers.

## Problem statement

`id-style: sequential` is readable and works well for single-writer workflows, but it makes entity creation a coordinated write. Today `status --next-id` computes the next numeric ID from active plus archived entities. If two people or agents create different task files from stale branches, worktrees, or offline clones, both can legitimately choose the same next number. Git may merge the files cleanly because their slugs differ, but Spacedock now has a semantic duplicate in the `id` field.

Spacedock already treats the filename slug as the operational path key, while the configured ID strategy supplies the compact operator address shown in status tables, stage reports, dispatch output, PR bodies, and issue references. This task should make that identity policy explicit and pluggable, preserve existing sequential workflows, cover the `id-style: slug` request from GitHub issue #98, and provide a generated style that is safe for concurrent and offline creation without a central reservation step.

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
- `generated`: a new collaboration-friendly style. `id` is required and stores a full stable random ID, while status output and operator references use the shortest unique prefix at or above a configured minimum prefix length. `status --next-id` returns a fresh full generated ID from a CSPRNG, checks it against active plus archived entities, and retries on known full-ID collisions. Validation is explicit through `status --validate` and fail-fast in status display paths, so duplicate stored IDs or unresolved prefix ambiguity cannot be silently presented as valid workflow state.

The recommended generated storage format is a 24-character lowercase Crockford Base32 random string using the ULID-style alphabet without visually confusing letters. That gives 120 bits of entropy. `id-style: generated` should also have a minimum address prefix length of 2 characters. The stored `id` remains stable in frontmatter; the displayed/addressable ID is computed from the current active plus archived entity set. Most small workflows display 2-character IDs. Prefixes lengthen to 3, 4, 5, or more only when another entity shares the shorter prefix. A deterministic test hook can force full-ID collisions and prefix collisions so the implementation proves retry, prefix growth, and ambiguity handling without relying on chance.

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

With the recommended first implementation, generated storage IDs are long but displayed/addressable IDs are short prefixes. Display length grows only for the entities whose prefixes collide:

| Workflow size     | Stored `id` examples              | Display/address examples |
| ----------------- | --------------------------------- | ------------------------ |
| 10s of entities   | `4k9q2m7x8c3v9r5t6w2p0n1h`, `8t5n0p2w6j9r4c8x1m7q3v5k`, `c3v7k1m9q2x8t5n0p6w4j9r2` | `4k`, `8t`, `c3` |
| 100s of entities  | `9m2c7v4xq8j3h6t0p5w1r8n2`, `9m2cq8j3h6t0p5w1r8v7x4kn`, `h6t3k9p2w5r8c4x1m7q0n6v2` | `9m2c7`, `9m2cq`, `h6` |
| 1000s of entities | `v7k3q9x2m5c8h6t0p1w4r8n2`, `v7k3qrv5t9p3j6n2w8c4x1mk`, `v7k3qjm8c2x9n5r1t6p4w0hk` | `v7k3q9`, `v7k3qr`, `v7k3qj` |

The address prefix is contextual. If a later branch adds `4kz7m1...`, the existing entity that displayed as `4k` may lengthen to `4k9` while the new one displays as `4kz`. Full stored IDs never change; copied short prefixes remain accepted only while unique. When a copied prefix becomes ambiguous, `status --resolve` fails with all candidates and asks for a longer prefix or slug.

## Design comparison

**Full stable ID plus minimum shortest-unique display prefix.** This is the recommended generated design. Store a long random ID in frontmatter and display/accept the shortest prefix that is unique across active plus archived entities, subject to a minimum length of 2 characters. It directly satisfies the captain requirement: small workflows get compact addresses, while larger workflows or collision clusters lengthen only where needed. The tradeoff is that a copied prefix can become ambiguous after a merge; Spacedock handles that by failing resolution with candidate details instead of guessing.

**Stored short generated IDs with collision handling.** This remains an alternative, but it is no longer the recommendation. The visible ID is stable because it is the stored ID, but it cannot provide shorter addresses for small workflows unless the stored value is itself short, which raises collision pressure and still requires fixed display length. This misses the core requirement for IDs that grow with workflow size or collisions.

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

Collision and schema validation should have a named CLI surface. Add `status --validate`, scoped by `--workflow-dir`, which scans active entities plus `_archive/` using the same flat/folder discovery rules as display and mutation paths. It exits 0 and prints `VALID` when the workflow is internally consistent. It exits 1 and prints one stderr line per problem when it finds a duplicate effective ID for `sequential`/`slug`, a duplicate full stored ID for `generated`, a missing required `id`, an invalid generated ID alphabet/length, a non-numeric sequential ID that would affect allocation, or any flat/folder conflict that changes the canonical entity path. Prefix collisions in generated workflows are not validation failures by themselves; they cause the affected display/address prefixes to lengthen until they are unique. Each validation error should name the workflow path, active or archived scope, slug, stored ID, display/address ID when available, and canonical file path. Default status output, `--archived`, `--next`, `--next-id`, and `--boot` should run the same validation before printing normal output; on validation failure they should exit 1 without printing a partial status table or boot payload. `--set` and `--archive` should continue to use active-only mutation resolution, but after a successful mutation they should leave any subsequent validation failure visible to the caller rather than masking it.

`--boot` should keep the `NEXT_ID:` line for first-officer compatibility, but its meaning becomes strategy-dependent and should be accompanied by an `ID_STYLE: {style}` line. For `sequential`, `NEXT_ID: 005` keeps the current highest-active-or-archived-plus-one behavior. For `generated`, `NEXT_ID: {full-stored-id-candidate}` emits the same kind of collision-checked 24-character stored ID as `status --next-id`; it is informational and not a reservation, so the first officer must call `status --next-id` again immediately before creating a new entity. `--boot` should also emit `MIN_PREFIX: 2` for generated workflows so the FO knows the address/display policy. For `slug`, `--boot` succeeds and emits `NEXT_ID: n/a (id-style: slug)`, while direct `status --next-id` exits non-zero with the same not-applicable message because the eventual ID is the slug derived from the title.

First-officer task creation should use the strategy explicitly. Under `sequential`, the FO calls `status --next-id` immediately before writing the entity and stores that value in `id`. Under `generated`, the FO also calls `status --next-id` immediately before writing the entity and stores the returned full generated value in `id`; the short address appears only after status recomputes the shortest unique prefix against the workflow's active plus archived entities. If `status --validate` reports a duplicate full stored ID after a branch/worktree merge, the FO refreshes workflow state, calls `--next-id` again, and rewrites only the new unmerged entity before committing. If a merge merely creates a shared prefix, the FO does not rewrite frontmatter; status output lengthens the affected display prefixes. Under `slug`, the FO derives the slug from the title, omits `id` or leaves it blank according to the workflow template, and never calls `--next-id` for creation.

Reference resolution should be implemented as a status CLI helper, not runtime-only prose. Add `status --resolve <ref>` as the canonical resolver used by FO logic and available to operators. With `--workflow-dir`, resolution is limited to one workflow. With `--root`, it discovers workflows and requires either an unambiguous match or a workflow qualifier of the form `{workflow-basename}::{ref}`; if two workflows share a basename, the qualifier must be an absolute workflow path followed by `::{ref}`. The resolver prints a single machine-readable line containing `workflow=`, `scope=active|archived`, `slug=`, `id=`, and `path=`, or exits 1 with all candidates listed on stderr.

Within one workflow, resolver precedence should preserve current slug-based operations: exact active slug wins over ID/prefix matches unless the caller uses `id:{value}` or `slug:{value}`. Exact full stored ID resolves next. For `generated`, `prefix:{value}` or a bare value of at least two generated-ID characters resolves when it uniquely matches a stored ID prefix across active plus archived entities; ambiguous prefixes fail and list all matching slugs, stored IDs, and current display addresses. A prefix shorter than the minimum length fails with a minimum-prefix error. `--set` and `--archive` should route their entity argument through the same resolver in active-only scope, so they can accept slug, exact full ID, or unique generated prefix, but they must refuse archived matches with an "archived entity is read-only" error. `status --resolve --archived <ref>` is read-only and includes archived entities; if the same reference matches both active and archived entities, it fails unless the caller qualifies with `active:{ref}` or `archive:{ref}`.

For generated workflows, default status and `--next` should display the computed address prefix in the existing `ID` column rather than the full stored ID. The full stored ID remains in frontmatter and is available through `status --resolve`, validation errors, and any explicit raw-field/debug output added during implementation. This preserves compact operator-facing tables while keeping durable identity available for exact references and audits.

Folder-form entities should use the existing discovery rule: `{slug}/index.md` is the canonical entity file, and folder form wins over a same-slug flat file with a warning. Archived folder entities live under `_archive/{slug}/index.md`. Every validation check, `--next-id` check, `--boot` candidate, status display, display-prefix computation, and resolver should use the same discovery path so flat, folder, active, and archived entities cannot disagree.

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

**AC-4 - `id-style: generated` stores full stable generated IDs and displays shortest unique address prefixes.**
Verified by: status-script tests run `status --next-id` under generated style with a deterministic RNG hook, assert the stored ID candidate is a 24-character lowercase Crockford Base32 value, assert active plus archived full-ID collisions are retried, assert forced retry exhaustion fails loudly, and assert default/`--next` status output displays minimum 2-character prefixes that lengthen only for prefix-collision groups.

**AC-5 - Invalid or duplicate effective IDs are caught before status output is trusted.**
Verified by: `tests/test_status_script.py` fixtures for active flat files, active `{slug}/index.md`, archived `.md` files, and archived `{slug}/index.md` show `status --validate` exits 1 with slug/stored-id/display-id/path evidence for duplicate generated full IDs, invalid generated full IDs, missing required IDs, and sequential allocation conflicts; generated prefix collisions with distinct full IDs validate successfully and produce longer display prefixes; default status output, `--archived`, `--next`, `--next-id`, and `--boot` fail before printing partial normal output on invalid fixtures.

**AC-6 - `--boot` and first-officer task creation are strategy-aware.**
Verified by: boot tests show `ID_STYLE:` plus `NEXT_ID:` for sequential, generated, and slug workflows; generated boot output contains a valid 24-character stored-ID candidate and `MIN_PREFIX: 2`, slug boot output contains `NEXT_ID: n/a (id-style: slug)`, direct slug `--next-id` exits non-zero, and static first-officer reference tests show creation stores full generated IDs while status later computes short address prefixes.

**AC-7 - Entity reference resolution is deterministic through `status --resolve`.**
Verified by: resolver tests cover exact slug precedence over ID/prefix, `id:` and `slug:` forced qualifiers, exact full stored ID, unique generated prefix at or above the minimum length, too-short generated prefix, ambiguous generated prefix after branch-style collisions, unknown reference, active-only `--set`/`--archive` resolution, archived-only mutation refusal, read-only `--resolve --archived`, and `active:`/`archive:` disambiguation, with assertions on the printed path or candidate-list error.

**AC-8 - Cross-workflow ambiguity is rejected rather than guessed.**
Verified by: temp-root fixtures with two discovered workflows containing the same slug, exact ID, or generated prefix show `--root ... --resolve <ref>` fails with both workflow paths, while explicit `--workflow-dir`, `{workflow-basename}::{ref}`, or absolute-path workflow qualifiers resolve the intended entity.

**AC-9 - Migration scope is documented and validation-backed without shipping a rewrite helper.**
Verified by: static docs tests confirm this task documents manual migration guidance and explicitly defers rewrite automation; validation fixtures prove duplicate effective IDs, invalid generated IDs, missing IDs, archived/folder conflicts, and flat/folder conflicts are reported without modifying README or entity frontmatter.

**AC-10 - Concurrent or offline task creation is safer under generated IDs than under sequential IDs.**
Verified by: a filesystem-level test simulates two isolated creators adding different entity files from the same starting workflow; generated style yields distinct full IDs without a shared counter, prefix collisions after merging are handled by display-prefix growth, and a forced full-ID duplicate is detected by `status --validate` and by fail-fast default/`--next`/`--boot` validation before status output can silently present duplicate stored identity.

**AC-11 - Commission/refit documentation preserves concrete style-selection and generated-ID examples.**
Verified by: static docs/template tests assert the commission flow offers `generated`, `sequential`, and `slug` with the documented recommendations; README frontmatter examples for all three styles include the exact `id-style:` values; generated examples for 10s, 100s, and 1000s of entities show 24-character stored IDs with shorter display/address prefixes; and docs explain that display prefixes can lengthen after collisions while stored IDs remain stable.

## Test plan

Most proof should be offline unit and fixture tests because the behavior is local file parsing, ID generation, and deterministic reference resolution. Live E2E tests are not required unless implementation also changes first-officer runtime dialogue around filing new tasks.

- Add focused status-script tests in `tests/test_status_script.py` for README `id-style` parsing, `--next-id` delegation, generated full-ID collision retry, generated display-prefix computation/growth, slug-style non-applicability, `--boot` `ID_STYLE`/`NEXT_ID`/`MIN_PREFIX` output, `--validate`, `--resolve`, archived entities, and folder-form entities. Estimated cost: medium to high; the file already has helpers for active, archived, boot, and folder-form workflows.
- Add fail-fast validation tests proving default status output, `--archived`, `--next`, `--next-id`, and `--boot` refuse invalid generated/sequential/slug effective-ID state before printing normal output. Estimated cost: medium.
- Add static content tests in `tests/test_agent_content.py` and commission-template tests so docs and runtime references no longer hard-code "sequential" where they mean strategy-dependent "new ID", and so FO creation behavior is documented for sequential, generated, and slug workflows. Estimated cost: low.
- Add migration-scope tests as static docs checks plus status validation fixtures only. This task should not test README/entity rewrite automation because it does not ship an executable migration helper. Estimated cost: low.
- Add one concurrency simulation using temporary directories or branches, not live agents. It should copy the same starting workflow into two isolated locations, create entities under generated style, merge/copy the files into one workflow, and run `status --validate`, default status, `--next`, and `--boot` checks that prove distinct full IDs coexist and shared prefixes lengthen instead of forcing rewrites. Estimated cost: medium.
- Add static docs/template tests for the style-selection prompt, the three README frontmatter examples, the 10s/100s/1000s generated-ID examples, and the note that generated display/address prefixes are computed from stable stored IDs and may lengthen when collisions appear. Estimated cost: low.
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

## Stage Report: ideation (repair 3)

- DONE: Recommended generated style now has short addressable/display IDs that grow with entity count or collisions.
  Evidence: Proposed approach now recommends full stored generated IDs plus minimum shortest-unique display prefixes, with examples that lengthen from 4 to 5-6 characters.
- DONE: Storage, display, resolution, validation, and ambiguity behavior for dynamic prefixes are concrete and testable.
  Evidence: Behavior details now define 24-character stored IDs, `ID` column display prefixes, `MIN_PREFIX`, `status --validate`, `status --resolve`, too-short/ambiguous prefix errors, and merge-time prefix growth.
- DONE: Examples for 10s, 100s, and 1000s of entities demonstrate the intended shorter-then-growing behavior and are covered by AC/test-plan updates.
  Evidence: The generated-ID table now shows stored IDs plus short display addresses for each workflow size, and AC-4/5/6/7/10/11 plus the test plan require prefix-growth coverage.

### Summary

The recommended generated design now stores durable full random IDs and exposes short dynamic address prefixes as the operator-facing ID. Fixed 12-character display is no longer the recommendation; prefix growth and ambiguity handling are first-class validation and resolver behavior.

## Stage Report: ideation (approval amendment)

- DONE: The generated style consistently uses minimum prefix length 2.
  Evidence: Live design sections now specify minimum address prefix length 2, `MIN_PREFIX: 2`, and two-character resolver minimums.
- DONE: Examples still show short prefixes for small workflows and growth under prefix collisions.
  Evidence: The generated-ID table shows 2-character addresses for 10s of entities and longer 5-6 character prefixes where collisions are shown for 100s/1000s.
- DONE: AC/test-plan references to minimum prefix behavior are consistent with the amendment.
  Evidence: AC-4 and AC-6 now require minimum 2-character prefixes and `MIN_PREFIX: 2`; the test-plan references remain strategy-level and consistent with that value.

### Summary

The approval amendment changes the dynamic generated-ID design from `MIN_PREFIX: 4` to `MIN_PREFIX: 2` while preserving full stored IDs and shortest-unique display/address prefixes. No frontmatter or unrelated files were edited.

## Stage Report: implementation

- DONE: Pluggable `id-style` strategy implements and documents sequential, slug, and generated behavior with generated full stored IDs plus `MIN_PREFIX: 2` shortest-unique display/address prefixes.
  Evidence: `skills/commission/bin/status` now parses `id-style`, preserves sequential defaults, uses slug effective IDs, stores/validates 24-character generated IDs, and displays generated shortest unique prefixes with `MIN_PREFIX: 2`.
- DONE: Status CLI behavior covers `--next-id`, `--boot`/`ID_STYLE`/`MIN_PREFIX`, `--validate`, `--resolve`, active/archive flat/folder uniqueness, and ambiguity behavior.
  Evidence: `tests/test_status_script.py` covers strategy-specific `--next-id`, boot output, validation fail-fast, generated prefix growth, root/workflow resolution, active/archive ambiguity, and flat/folder validation conflicts; `uv run pytest tests/test_status_script.py -q` passed with 172 tests and 15 subtests.
- DONE: Tests/docs preserve commission/refit style selection, examples, migration scope, concurrent/offline creation safety, and backward compatibility for sequential workflows.
  Evidence: `skills/commission/SKILL.md`, `skills/refit/SKILL.md`, and first-officer runtime docs now document style selection, generated examples, migration boundaries, and task creation behavior; `make test-static` passed with 532 tests, 26 deselected, and 15 subtests.

### Summary

Implemented the status strategy layer for `sequential`, `slug`, and `generated`, including generated ID allocation, validation, shortest unique display prefixes, and deterministic reference resolution. Updated commission, refit, and first-officer docs so task creation and migration guidance are strategy-aware while preserving sequential workflow compatibility.

## Stage Report: validation

- FAILED: Validate AC-1 through AC-11 against the actual implementation, including the captain amendment `MIN_PREFIX: 2` and no fixed 12-character displayed-ID behavior.
  Evidence: AC-1 and AC-3 through AC-11 have targeted test evidence, but AC-2 fails because the existing sequential self-workflow cannot produce normal status output with historic archived duplicate IDs.
- DONE: Run targeted status/docs tests plus the stable offline suite as appropriate, and explicitly assess the observed self-workflow duplicate-archive failure against backward compatibility.
  Evidence: `uv run pytest tests/test_status_script.py -q` = 172 passed/15 subtests; `uv run pytest tests/test_agent_content.py tests/test_commission_template.py -q` = 55 passed; `make test-static` = 532 passed/26 deselected/15 subtests; duplicate-archive failure is a compatibility regression.
- DONE: Append a validation report with concrete evidence, test results, and a clear `PASSED` or `REJECTED` recommendation.
  Evidence: This validation report records a `REJECTED` recommendation; commit follows on the validation branch.
- DONE: AC-1 - Workflows have a documented `id-style` strategy contract covering `sequential`, `slug`, and `generated`.
  Evidence: Static docs tests passed; `tests/test_agent_content.py` and `tests/test_commission_template.py` assert all three styles, required/optional `id`, `--next-id`, and no generic "next sequential ID" wording.
- FAILED: AC-2 - Existing sequential workflows keep their current behavior.
  Evidence: Current `skills/commission/bin/status --workflow-dir docs/plans --where id=217`, `--boot`, `--next`, `--next-id`, and `--resolve 217` exit 1 on archived duplicate IDs `131` and `033`; the pre-implementation script printed the `217` row and `NEXT_ID: 220`.
- DONE: AC-3 - `id-style: slug` treats the slug as the effective ID and makes `status --next-id` non-applicable.
  Evidence: `TestIdStyleStrategies.test_slug_style_uses_slug_as_effective_id_and_next_id_is_not_applicable` passed in `tests/test_status_script.py`.
- DONE: AC-4 - `id-style: generated` stores full stable generated IDs and displays shortest unique address prefixes.
  Evidence: Generated `--next-id`, retry exhaustion, and active/archive prefix-growth tests passed; they assert 24-character stored IDs and 2-character minimum display prefixes rather than fixed 12-character display IDs.
- DONE: AC-5 - Invalid or duplicate effective IDs are caught before status output is trusted.
  Evidence: Validation/fail-fast tests for generated duplicates, invalid/missing IDs, sequential duplicate/non-numeric IDs, and flat/folder conflicts passed; the self-workflow failure shows this strictness currently overbreaks AC-2.
- DONE: AC-6 - `--boot` and first-officer task creation are strategy-aware.
  Evidence: Boot tests assert `ID_STYLE`, strategy-specific `NEXT_ID`, and `MIN_PREFIX: 2`; first-officer static tests assert sequential/generated/slug creation behavior.
- DONE: AC-7 - Entity reference resolution is deterministic through `status --resolve`.
  Evidence: Resolver tests cover slug precedence, forced qualifiers, generated prefix minimum/ambiguity, archived disambiguation, and active-scope `--set`/`--archive` resolution.
- DONE: AC-8 - Cross-workflow ambiguity is rejected rather than guessed.
  Evidence: Root resolver tests cover duplicate slugs, duplicate generated prefixes/full IDs, basename qualifiers, and absolute path qualifiers.
- DONE: AC-9 - Migration scope is documented and validation-backed without shipping a rewrite helper.
  Evidence: Static refit/commission docs tests assert manual migration, validation before style changes, and deferred rewrite automation; validation fixtures report bad state without rewriting frontmatter.
- DONE: AC-10 - Concurrent or offline task creation is safer under generated IDs than under sequential IDs.
  Evidence: `TestGeneratedIdConcurrency.test_generated_style_allows_isolated_creators_and_grows_prefix_after_merge` passed, including distinct full IDs and prefix growth after merge.
- DONE: AC-11 - Commission/refit documentation preserves concrete style-selection and generated-ID examples.
  Evidence: Static template tests assert the three style labels, 24-character Crockford examples for 10s/100s/1000s, `MIN_PREFIX: 2`, prefix lengthening, and stable stored IDs.

### Summary

Recommendation: `REJECTED`. The generated, slug, resolver, docs, and offline test coverage are strong, and the captain amendment is implemented as full stored 24-character generated IDs with shortest unique prefixes and `MIN_PREFIX: 2`. The blocking defect is backward compatibility: the existing `id-style: sequential` Spacedock workflow now fails normal status, boot, next-id, and resolution paths because archived historical duplicates are treated as fatal before output.

### Feedback Cycles

- Cycle 1 (2026-04-27T23:59:27Z): Validation rejected implementation for AC-2. Route back to implementation to preserve existing sequential self-workflow behavior despite historic archived duplicate IDs `131` and `033`, while keeping strict generated-ID duplicate validation and the approved `MIN_PREFIX: 2` generated display behavior.

## Stage Report: implementation (repair 1)

- DONE: AC-2 backward compatibility is repaired for existing sequential workflows with historic archived duplicate IDs, without weakening generated-ID strictness.
  Evidence: Sequential validation now ignores archived-only duplicate numeric IDs but still rejects duplicate groups involving active sequential entities; exact self-workflow commands for `--where id=217`, `--boot`, `--next`, `--next-id`, and `--resolve 217` returned output.
- DONE: Focused tests cover the repaired sequential duplicate-archive behavior and still cover generated duplicate validation plus `MIN_PREFIX: 2` prefix display behavior.
  Evidence: `tests/test_status_script.py::TestIdStyleStrategies` adds archived-only sequential duplicate coverage for `--where`, `--boot`, `--next`, `--next-id`, `--resolve`, and `--validate`; focused generated duplicate and prefix tests passed.
- DONE: Append a new `## Stage Report: implementation (repair 1)` section accounting for this checklist with DONE/SKIPPED/FAILED evidence, and commit all repair changes before reporting completion.
  Evidence: This report was appended to the entity body; repair commit follows after `uv run pytest tests/test_status_script.py -q` and `make test-static`.

### Summary

The repair narrows sequential duplicate validation so archived-only historical duplicates do not invalidate existing workflows or block normal status operations. Generated ID validation remains strict for duplicate full stored IDs and invalid generated IDs, and generated display prefixes still use shortest unique prefixes with `MIN_PREFIX: 2`.

## Stage Report: validation (recheck 1)

- DONE: AC-2 repair is independently verified with command evidence against the self-workflow or equivalent fixture.
  Evidence: Self-workflow `--where id=217`, `--boot`, `--next`, `--next-id`, `--resolve 217`, and `--validate` all exited 0 despite archived duplicate sequential IDs `033` and `131`; `--boot`/`--next-id` reported `220`.
- DONE: Generated duplicate validation and `MIN_PREFIX: 2` dynamic prefix behavior remain independently verified.
  Evidence: Focused generated tests for duplicate full IDs, fail-fast status paths, shortest active/archive prefixes, and boot `MIN_PREFIX: 2` passed: 4 tests plus 5 subtests.
- DONE: Validation report gives concrete test results and a clear `PASSED` or `REJECTED` recommendation.
  Evidence: This recheck report records a `PASSED` recommendation after focused checks and `make test-static`.
- DONE: AC-1 - Workflows have a documented `id-style` strategy contract covering `sequential`, `slug`, and `generated`.
  Evidence: `uv run pytest tests/test_agent_content.py tests/test_commission_template.py -q` passed 55 tests covering style docs, FO creation docs, and generated examples.
- DONE: AC-2 - Existing sequential workflows keep their current behavior.
  Evidence: Real `docs/plans` sequential workflow now permits normal status operations with historic archived duplicates; focused sequential fixture tests passed, including active duplicate rejection.
- DONE: AC-3 - `id-style: slug` treats the slug as the effective ID and makes `status --next-id` non-applicable.
  Evidence: `uv run pytest tests/test_status_script.py -q` passed 173 tests, including slug-style status, where-filter, boot, and direct `--next-id` assertions.
- DONE: AC-4 - `id-style: generated` stores full stable generated IDs and displays shortest unique address prefixes.
  Evidence: Generated tests assert 24-character stored IDs, collision retry/exhaustion, and shortest unique prefixes with minimum length 2.
- DONE: AC-5 - Invalid or duplicate effective IDs are caught before status output is trusted.
  Evidence: Generated duplicate full IDs still fail `--validate` and fail-fast output paths; sequential duplicate groups involving active entities and non-numeric IDs still fail validation.
- DONE: AC-6 - `--boot` and first-officer task creation are strategy-aware.
  Evidence: Boot/status tests plus static FO docs tests passed, including `ID_STYLE`, strategy-dependent `NEXT_ID`, and generated `MIN_PREFIX: 2`.
- DONE: AC-7 - Entity reference resolution is deterministic through `status --resolve`.
  Evidence: Resolver tests passed for slug precedence, forced qualifiers, generated prefix minimum/ambiguity, archived disambiguation, and mutation resolution.
- DONE: AC-8 - Cross-workflow ambiguity is rejected rather than guessed.
  Evidence: Root resolver tests passed for duplicate slugs, duplicate generated prefixes/full IDs, basename qualifiers, and absolute path qualifiers.
- DONE: AC-9 - Migration scope is documented and validation-backed without shipping a rewrite helper.
  Evidence: Static refit/commission docs tests passed for manual migration, validation-backed style changes, and deferred rewrite automation.
- DONE: AC-10 - Concurrent or offline task creation is safer under generated IDs than under sequential IDs.
  Evidence: Generated concurrency simulation remains covered in the full status-script suite and passed with prefix growth after merge.
- DONE: AC-11 - Commission/refit documentation preserves concrete style-selection and generated-ID examples.
  Evidence: Static template tests passed for all three style labels, 24-character Crockford examples, `MIN_PREFIX: 2`, prefix lengthening, and stable stored IDs; `rg` found no `fixed 12` or `12-character` wording in `skills/`, `tests/`, `references/`, or `docs/plans/README.md`.

### Summary

Recommendation: `PASSED`. Repair commit `815c0d1e` fixes the AC-2 regression by grandfathering archived-only sequential duplicate IDs while preserving active duplicate rejection and strict generated full-ID validation. Verification passed with `tests/test_status_script.py` = 173 passed/15 subtests, docs/template tests = 55 passed, and `make test-static` = 533 passed/26 deselected/15 subtests.
