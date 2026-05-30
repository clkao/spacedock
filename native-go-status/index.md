---
id: 60kbtwqrp8z8szp96e0mfm5a
title: Implement native Go status parity
status: implementation
score: "0.65"
source: bootstrap roadmap
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-native-go-status
started: 2026-05-30T04:30:15Z
---

# Implement Native Go Status Parity

Replace the vendored Python status runner with a native Go implementation that reproduces the current tool's behavior byte-for-byte, then fold in the confirmed `--new` atomic-create decision so seed entities never exist id-less.

## Problem Statement

After Stage 2 (vendor-status-compatibility), `spacedock status` forwards every argument to a vendored copy of the 2547-line Python script (`skills/commission/bin/status`) behind a narrow `Runner` interface:

```
type Runner interface {
    Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (exitCode int, err error)
}
```

The vendored exec path works but ties the launcher to a `python3` interpreter at runtime and to a frozen blob of Python the Go repo cannot maintain. This stage supplies the **native-Go implementation that backs the same `Runner` interface** so callers (`internal/cli/`) and the Stage 2 golden + mutation tests stay green with no caller change. The risk this stage protects against is the native rewrite silently diverging from the oracle in output bytes, sort order, mutation semantics, or validation messages that the first officer has already encoded its event loop around.

Three load-bearing facts about how the FO consumes the current script (so parity is not optional):

- The FO **parses table rows and `--next` rows** to choose dispatchable entities, **parses `--boot` output by section** (MODS / ID_STYLE / NEXT_ID / ORPHANS / PR_STATE / DISPATCHABLE / TEAM_STATE), **parses `--resolve` output** (`workflow= scope= slug= id= path=`), and **trusts `--set` stdout** (`field: old -> new`, one line per field) to narrate mutations without re-reading the entity file.
- It relies on the status tool's **exit codes** to gate the loop, and on the **mutation guards** in `--set`/`--archive` (mod-block, merge-hook, terminal-transition invariants) plus the **validation messages** that gate state trust. **The status tool uses exactly two exit codes — `0` success and `1` error** (verified against the oracle: 7×`sys.exit(0)`, 55×`sys.exit(1)`, zero `sys.exit(2)`, and no `argparse`). **Usage/parse errors (bad `--where`, missing flag arguments, incompatible-flag combinations) print `Error: ...` to stderr and exit `1`, NOT `2`.** The native runner must reproduce `{0, 1}` and must never return `2` for a usage error. (The `2` exit reserved for unknown-command/usage belongs to the outer `internal/cli/` launcher layer — `cli.go` — not to the status runner; that boundary is unchanged here.)
- It depends on the **id lifecycle**: read `ID_STYLE`, call `--next-id --id-seed "{slug-or-title}"` immediately before writing a new entity, then write the entity with that `id` baked into frontmatter (`first-officer-shared-core.md:63-67`).

That last fact exposes a **chicken-and-egg in entity creation** and an **id-strictness inconsistency** that decision B resolves (folded into this stage):

- For `sequential` and `sd-b32`, validation requires every active+archived entity to carry a valid `id` (`validate_workflow`, the Python script). But the FO's creation path is *not atomic*: it computes an id, then separately writes the file. If a seed file ever lands id-less before its `id` is set, the next *enumeration* op rejects the whole workflow. There is no single operation that mints an id and writes the entity together.
- Worse, **id-presence is enforced inconsistently**. Read/enumerate ops — default table, `--next`, `--boot`, `--next-id`, `--resolve`, `--short-id`, `--validate` — call `fail_on_validation_errors` and therefore enforce id-presence *globally* across all entities. But **targeted mutation** ops — `--set` and `--archive` — resolve their single target via `resolve_reference_candidates` and do **not** run validation, so they happily mutate even when another entity is id-less. So "is an id required?" answers differently depending on which subcommand you run. This is the `--set`-vs-enumerate inconsistency: `--set` can run against a workflow that `--validate` rejects.

The design problem: **reimplement the full observable contract of the current script in native Go (parser, stage parser, discovery, formatter, mutation engine, validation), and add a `--new` atomic create that mints an id and writes the entity in one filesystem operation so a seed never exists id-less — then state a single, consistent id-strictness rule across read and mutation ops.**

## Proposed Approach

### Behavioral-parity scope (independent of vendor's exec-vs-copy choice)

**ASSUMPTION (stated for FO reconciliation at the gate):** This is a *behavioral* parity spec. It targets the observable contract `(argv, env, cwd) -> (stdout bytes, stderr bytes, exit code, filesystem mutations under workflow_dir)` that Stage 2 froze as goldens. It is **independent of whether vendor-status-compatibility chose exec-the-Python-copy or some other backing** — the native implementation must reproduce the same four channels for the same inputs, verified by reusing Stage 2's golden + mutation fixtures as the oracle. If Stage 2's `Runner` signature differs from the one quoted above at gate time, the FO reconciles; the native package satisfies whatever single narrow `Runner` interface `internal/status/` exports.

**ASSUMPTION:** `state:` split-root resolution is explicitly **out of scope** here (that is Stage 6, native-state-dir). This stage assumes the Stage 3 symlink (`.spacedock-state/README.md -> ../README.md`) is present so stages and entities resolve from one `--workflow-dir`, exactly as the current script sees them. Native split-root is layered on in Stage 6 without changing the parser/formatter/mutation/validation cores designed here.

**ASSUMPTION:** PR fields are treated as **ordinary frontmatter fields** for now (roadmap §5: "PR field preservation as a normal field for now"). No PR merge flow, no `gh` integration changes, no mod behavior is added. `--boot`'s `PR_STATE`/`ORPHANS`/`TEAM_STATE` sections are reproduced as the current script emits them (including the deterministic `none` forms for fixtures with no PRs/orphans), but no new PR semantics are introduced.

### Environment- and cwd-derived inputs (parity tests must pin these)

The oracle's output is not a pure function of `(argv, files)`; several outputs depend on ambient process state. The native runner must read the **same** inputs from the **same** sources, and the parity tests must pin them identically on both sides:

- **`SPACEDOCK_TEST_SD_B32_TIMESTAMP`** — overrides the sd-b32 timestamp when set (the script's test hook); otherwise `datetime.now(timezone.utc)` ISO-microseconds. Affects `--next-id`, `--boot` NEXT_ID, `--new`.
- **`SPACEDOCK_ID_ACTOR` → `USER` → `USERNAME`** (first non-empty) and **`SPACEDOCK_ID_CONTEXT`** — enter the sd-b32 digest material. Affect every sd-b32 id derivation.
- **`HOME` / `~`** — the `--boot` `TEAM_STATE` probe reads `~/.claude/teams/*/config.json` mtimes within a 30-minute window. Machine- and time-dependent.
- **`PATH`** — `--boot` `PR_STATE` searches `PATH` for `gh`; `--boot` `ORPHANS` runs `git worktree list`. Tool-availability-dependent.
- **cwd** — `--discover` without `--root` runs `git rev-parse --show-toplevel`, falling back to `os.getcwd()`; the result is cwd-relative. (`--discover` is out of the read-golden parity set but the native runner must reproduce this resolution; tests pass an explicit `--root` to make it deterministic.)
- **`os.path.realpath`** — `--resolve` `workflow=` and the sd-b32 digest `workflow=` line resolve symlinks (macOS `/var`→`/private/var`); `path=` does not. The native runner must apply realpath at exactly the same call sites.

The parity harness controls all of the above: it sets/clears the `SPACEDOCK_*` vars and a fixed actor on both oracle and native, points `HOME` at a controlled dir (or normalizes `TEAM_STATE` away), constructs fixtures with no PRs/orphans so `gh`/worktree state does not leak, and passes explicit `--root` for `--discover`. Anything left ambient is normalized in the test, never in the product.

### Native architecture (small Go packages, stdlib only, under `internal/status/`)

Per AGENTS.md ("small Go packages with clear boundaries", "standard library unless a dependency removes real complexity", "internal/status/ home"), the native runner decomposes into focused units the way the Python script already does, each independently testable against the oracle:

**0. Root-resolution seam** (`roots.go`) — `resolveRoots(workflowDir) -> (definitionDir, entityDir)`. Introduced **now**, even though in Stage 5 `definitionDir == entityDir == workflowDir` (single-root, matching the current script's `os.path.join(workflow_dir, 'README.md')` and same-directory entity scan). The seam is threaded through every unit by role:
- **`definitionDir`** feeds the **stage parser** (reads `definitionDir/README.md` stages block) and the **identity engine** (sd-b32 digest uses the workflow realpath / id-style from the README).
- **`entityDir`** feeds **discovery**, **mutation**, **archive** (`entityDir/_archive`), and the entity-side of **validation**.

This is the explicit reconciliation native-state-dir (Stage 6) flagged as its hard dependency: Stage 6 makes `resolveRoots` return `definitionDir` = README directory and `entityDir` = the `state:` path, with no other call-site changes. Threading the two roles now — while they are equal — turns Stage 6 from a call-site-wide retrofit into a one-function extension. **ASSUMPTION (FO reconcile at gate):** Stage 6 owns the actual split logic and the `state:` field read; Stage 5 only establishes the seam with both roles pointing at `workflowDir`, so Stage 5's golden/mutation parity is unaffected.

1. **Frontmatter parser** (`frontmatter.go`) — line-oriented, no YAML dependency, matching `_has_opening_fence` + `parse_frontmatter`:
   - A file has frontmatter **iff** its first non-empty (`\n`-only lines skipped), non-BOM line is exactly `---`; a whitespace-only first content line (e.g. `"   \n"`) disqualifies it. A leading UTF-8 BOM on the first line is stripped before the check.
   - Between the first and second `---`, each line is split on the **first** `:`; key and value are stripped. A value that is `>= 2` chars and identically quoted at both ends with `"` or `'` has the quotes stripped. **Nested/indented lines (`line[0].isspace()`) are ignored** — only top-level `key: value` pairs become fields. Empty values (`score:`) yield empty string, displayed as blank (never `-`/`0`/field-name). Duplicate keys: last top-level occurrence wins (matches dict assignment).

2. **Stage parser** (`stages.go`) — matching `parse_stages_block` + `parse_stages_with_defaults`:
   - Reads the `stages:` block from README frontmatter by indentation, producing ordered stages with resolved `worktree` (default `false`), `concurrency` (default `2`), `gate`/`terminal`/`initial` (default `false`), and optional carried fields `feedback-to`, `agent`, `fresh`, `model`. Returns nil when no `stages:` block or no `states:` entries. Also exposes the raw `stages.defaults` map.
   - Stage-name validation regex `^[a-z0-9][a-z0-9-]*[a-z0-9]$` with the same kebab suggestion text.

3. **Entity discovery** (`discover.go`) — matching `discover_entity_files` / `resolve_entity_path`:
   - An entity is a flat `{slug}.md` **or** a folder `{slug}/index.md`. `README.md`, non-`.md` files, dotfiles, and files failing `_has_opening_fence` are skipped. Reserved subdirs are **exactly `{_archive, _mods}`** and dot-prefixed dirs are skipped (so `_debriefs/` is ignored only because it has no `index.md` — NOT because it is reserved; the native code must match this, leaving `_debriefs` un-reserved). When both forms hold a slug, folder form wins and the **same stderr warning** is emitted. Results sorted by slug.

4. **Output formatter** (`format.go`) — matching `print_status_table` / `print_next_table` / `print_boot`:
   - Default table columns `ID SLUG STATUS TITLE SCORE SOURCE` with the **exact `%-6s %-30s %-20s %-30s %-8s %s` widths**; `--next` table `ID SLUG CURRENT NEXT WORKTREE` with `%-6s %-30s %-20s %-20s %s`. Header row, dash separator row, then data rows. `--fields`/`--all-fields` append extra columns with the same padded base format and `%-20s`/`%s` extra widths, header upper-cased, separator dashes `min(len,20)`, cells truncated to 20 with `…` (U+2026), blank for missing/empty.
   - Sort: default by `(stage_order asc, -score)` with empty score sorting **last** (sort key `1`) and unknown status → order `99`; `--next` by `-score` (empty last).
   - `--boot` sections in the exact order MODS, ID_STYLE/NEXT_ID(/MIN_PREFIX for sd-b32), ORPHANS, PR_STATE, DISPATCHABLE, TEAM_STATE, each with its `none` form and column layout.

5. **Identity engine** (`identity.go`) — matching the three id-styles:
   - `sequential`: zero-padded `%03d` of max numeric id across active+archived, +1.
   - `sd-b32`: 24-char lowercase from alphabet `0123456789abcdefghjkmnpqrstvwxyz`, SHA-256 over the exact digest material lines (`spacedock-sd-b32-v1`, `workflow=<realpath>`, `context=`, `seed=`, `actor=`, `timestamp=`, `nonce=`), 5-bit big-endian extraction, nonce loop 0..1023, honoring `SPACEDOCK_TEST_SD_B32_TIMESTAMP`, `SPACEDOCK_ID_ACTOR`/`USER`/`USERNAME`, `SPACEDOCK_ID_CONTEXT`. Display ids are shortest unique prefixes (MIN_PREFIX 2) across active+archived.
   - `slug`: identity is the slug; `--next-id` not applicable (same error).

6. **Mutation engine** (`mutate.go`) — matching `update_frontmatter` / `run_archive`:
   - `--set`: rewrite matching top-level frontmatter lines in place as `key: value`; insert missing fields before the closing `---`; bare timestamp fields (`started`/`completed`) auto-fill `now()` as `YYYY-MM-DDTHH:MM:SSZ` **only if currently empty**; emit `field: old -> new` per resolved field (clear → `field: old -> `; bare-fill → `field:  -> {ts}`). Preserve the full mutation-guard set (mod-block, merge-hook, terminal-transition) with identical error text and exit 1, and PR mirror-back to main copy for worktree-backed entities.
   - `--archive`: stamp `archived: {ts}`, move flat `{slug}.md` or folder `{slug}/` under `_archive/`, print `archived: {dest}`, with the same source-missing / already-archived / mod-block / merge-hook guards.

7. **Validation** (`validate.go`) — matching `validate_workflow`:
   - Flat/folder conflicts (active + archived), stage-name regex, and per-style id rules: `sequential` (missing id, non-numeric, duplicate among groups with ≥1 active), `sd-b32` (missing, invalid stored, duplicate), `slug` (duplicate effective id). Same `Error: ... workflow= scope= slug= id= [display=] path=` evidence lines and exit 1.

### Decision B: `--new <STDIN` atomic create + a single id-strictness rule

**Confirmed decision B, folded in:** Add `status --new --workflow-dir <dir> <slug> < entity-body` (entity frontmatter+body on STDIN). In **one filesystem operation** it: (1) mints the next id for the workflow's `id-style` (reusing the identity engine, with `--id-seed`/`--id-actor` accepted for sd-b32), (2) writes the entity to `{dir}/{slug}.md` (or `{dir}/{slug}/index.md` if STDIN declares folder form — default flat, matching current creation), with the minted `id` already present in frontmatter, and (3) never leaves a window where the seed exists id-less. The id is stamped into the STDIN frontmatter (inserting `id:` if absent, refusing if STDIN already carries a non-empty conflicting `id`), so the seed is born valid. This resolves the chicken-and-egg: there is now a single create primitive instead of "compute id, then write file, then hope no read op runs in between." Errors if the target slug already exists (flat or folder), if STDIN has no opening `---` fence, or if `id-style: slug` is combined with `--id-seed`/`--id-actor` (not applicable, same as `--next-id`).

**Id-strictness rule made consistent (resolves the `--set`-vs-enumerate inconsistency).** The native implementation states one rule explicitly: **id-presence is a per-entity property enforced by `--validate` and by every enumeration/read op (default, `--next`, `--boot`, `--next-id`, `--resolve`, `--short-id`) for `sequential`/`sd-b32`** — these scan all entities and fail the whole workflow on any id-less entity, exactly as today. **Targeted single-entity mutation (`--set`, `--archive`) does not run global validation** and only requires that *its own resolved target* exists — also exactly as today. `--new` closes the only gap that made this rule observably inconsistent: previously a seed could be born id-less and a subsequent read would reject the workflow; now `--new` guarantees seeds carry an id at birth, so the "global enforcement on read, local-only on mutate" split is no longer reachable through normal creation. The rule is therefore: **reads enforce id-presence globally; mutations enforce only target existence; creation (`--new`) guarantees id-presence at birth.** No existing op's strictness changes — `--new` removes the inconsistency by construction rather than by loosening or tightening any enumerate/mutate op.

**`--new` is what justifies the `Runner` interface's `stdin` parameter.** The Stage-2 vendored oracle never reads stdin (it is a pure `argv`→output tool), so the `stdin io.Reader` in `Runner.Run(ctx, args, stdin, stdout, stderr)` had no consumer until now. `--new` is the first and (in this stage) only consumer: the entity body arrives on stdin. This resolves the staff-review concern that the `stdin` parameter looked speculative — it is load-bearing for `--new` and was correctly provisioned in the Stage-2 seam.

**Atomic writes and tool-owned state serialization (design principle).** The oracle's `--set` rewrites in place (`open(filepath, 'w')`) and `--archive` does `os.rename`, with **no locking** — concurrent writers can interleave a read-modify-write and lose an update, and the tool delegates all commit/serialization to free-form caller `git` commands. We hit exactly this class of race this session (a shared-state-repo commit captured a sibling's pre-staged file). The native implementation adopts a single principle for all state writes — `--set`, `--archive`, and `--new`:
- **Write atomically**: assemble the full new file contents in memory, write to a temp file in the same directory, then `os.Rename` into place (rename is atomic within a filesystem). This is already required for `--new` (so no id-less window is ever observable); apply the same temp-file+rename to `--set` and the frontmatter stamp inside `--archive`. A reader can never observe a half-written entity.
- **The tool owns state-commit serialization** rather than relying on free-form `git commit` by the caller. The atomic-write principle makes each individual entity write safe; serializing the *commit* of state changes (so two mutations do not race the shared index) is the tool's responsibility, not the caller's. Concretely: state-mutating subcommands should commit their own narrow, path-scoped change set (the single entity file they touched) rather than leaving a dirty index for an out-of-band `git commit` to sweep. **ASSUMPTION (FO reconcile at gate):** the precise commit/locking mechanism (e.g. an entity-file-scoped commit emitted by `--set`/`--archive`/`--new`, or an advisory lock on the state dir) is a small design choice to settle in implementation; this stage commits to the *principle* — atomic per-entity writes plus tool-owned, path-scoped state commits — not yet to a specific lock primitive. It does not change any read-path output.

**Scope guard.** `--new` is the only *new* surface; every other behavior is parity with the current script. `--new` does not run global validation before minting (it would otherwise inherit the same chicken-and-egg if an unrelated entity were temporarily id-less); it mints against the existing-id set the same way `--next-id` computes its candidate.

## Acceptance criteria

Each AC names a property of the finished native runner, not a stage action, and how it is verified.

**AC-1 - The native runner reproduces the current script's stdout, stderr, and exit code for every read subcommand on pinned fixtures.**
For a frozen workflow fixture (sd-b32 and sequential variants, flat + folder entities, one empty-score entity to lock "empty sorts last", one unknown-status entity to lock order 99), native stdout/stderr/exit equals the oracle for default table, `--archived`, `--next`, `--where`, `--fields`, `--all-fields`, `--next-id`, `--resolve`, and `--short-id`, after the shared timestamp/abspath/sd-b32 normalization (Test Plan).
Verified by: golden parity tests in `internal/status` comparing native output to goldens captured from the oracle (the current Python script / Stage 2 fixtures) for each listed flag; sd-b32 `--next-id`/`--resolve`/`--short-id` pinned via `SPACEDOCK_TEST_SD_B32_TIMESTAMP` + explicit `--id-seed`/`--id-actor`.

**AC-2 - The frontmatter and stage parsers match the current line-oriented parsers on the edge cases the FO depends on.**
The frontmatter parser matches `_has_opening_fence` + `parse_frontmatter` for: empty fields (`score:` → blank), quoted fields (matched-quote stripping), nested/indented lines ignored, missing/whitespace-first-line frontmatter rejected, leading-BOM handling, and last-top-level-key-wins. The stage parser matches `parse_stages_block` for defaults, gates, terminal stages, worktree flags, feedback targets, and the no-`stages:` → nil case.
Verified by: Go table tests in `internal/status` (`frontmatter_test.go`, `stages_test.go`) asserting parsed structs against expected values for each case, with at least the empty/quoted/nested/missing-frontmatter and defaults/gates/terminal/worktree/feedback-target cases enumerated in the roadmap test gates.

**AC-3 - Entity discovery handles flat and folder forms identically to the current script, including the form-conflict warning and reserved-dir set.**
Discovery returns the same `(slug, path)` set sorted by slug; folder form wins on conflict with the same stderr warning; reserved subdirs are exactly `{_archive, _mods}`; dot-prefixed dirs and `index.md`-less folders (e.g. `_debriefs/`) are ignored without being treated as entities or as reserved.
Verified by: discovery tests in `internal/status` over a fixture containing a flat entity, a folder entity, a flat/folder conflict slug, `_archive/`, `_mods/`, `_debriefs/`, and a dot-dir — asserting the resolved set, sort order, and captured conflict warning.

**AC-4 - Mutation commands produce the same frontmatter edits, stdout narration, timestamp format, and guard behavior as the current script.**
`--set` (field set, clear, bare-timestamp auto-fill that skips already-set fields) and `--archive` produce the same resulting frontmatter and the same stdout (`field: old -> new` / `field: old -> ` / `field:  -> {ts}`; `archived: {dest}`), stamp `YYYY-MM-DDTHH:MM:SSZ`, preserve PR as a normal field including mirror-back on worktree-backed entities, and enforce mod-block / merge-hook / terminal-transition guards with identical error text and exit 1.
Verified by: temp-workflow mutation tests in `internal/status` that run each mutation natively and through the oracle into separate temp dirs and diff resulting files + stdout (timestamps/abspaths normalized); guard tests asserting a terminal `--set` under an active mod-block exits 1 with the current message.

**AC-5 - Unknown and external-tracker frontmatter fields survive mutation verbatim.**
`issue`, `source`, and any unknown top-level frontmatter field an entity carries are preserved byte-for-byte through `--set` (of an unrelated field) and `--archive` — neither dropped, reordered relative to surrounding lines, nor reformatted — matching the oracle's in-place line-oriented rewrite (which only touches lines whose key is in the update set and appends genuinely-missing fields before the closing `---`). This is the explicit reconciliation of external-tracker-checkpoint's load-bearing assumption A1 (the external reference must survive status, mutation, and archive), so that entity's preservation assertions can fold into this mutation suite.
Verified by: a mutation fixture entity carrying `issue: ENG-123`, `source: linear`, and an arbitrary unknown field (e.g. `tracker-url: ...`); tests assert those lines are byte-identical in the file after a `--set` on a different field and after `--archive`, run natively and diffed against the oracle's result.

**AC-6 - Validation reproduces the current failure set and messages for the documented error classes.**
For `sequential`/`sd-b32`/`slug` styles, validation flags duplicate ids, invalid/non-numeric ids, missing required ids, flat/folder conflicts, unknown/invalid stage names, and surfaces the same `Error: ... workflow= scope= slug= id= [display=] path=` evidence lines with exit 1; a valid workflow prints `VALID` exit 0. (Terminal/archive guards are exercised under AC-4's mutation guards.)
Verified by: validation tests in `internal/status` over fixtures seeded with each defect class, asserting stderr error lines and exit code equal the oracle's; a positive fixture asserting `VALID`.

**AC-7 - `--new` atomically mints an id and writes a valid entity, so a seed never exists id-less, and the id-strictness rule is consistent across read, mutate, and create.**
`status --new <slug>` reads an entity body from STDIN, mints the `id-style`-appropriate id (sd-b32 honoring `--id-seed`/`--id-actor`), writes `{slug}.md` (or folder form when STDIN declares it) with the id present in frontmatter in one operation, and refuses when the slug already exists, STDIN lacks an opening fence, or `--id-seed`/`--id-actor` is used with `id-style: slug`. After `--new`, the workflow passes `--validate` with no intermediate id-less state observable to any read op.
Verified by: `internal/status` tests that pipe a fixture body to `--new`, assert the written file's frontmatter (minted id matches `--next-id` under the same pinned env; id-style branches covered), assert `--validate` is `VALID` immediately after with no id-less window, and assert the slug-exists / no-fence / slug-style-`--id-seed` error+exit-code paths.

## Test Plan

All proof is **Go unit/fixture tests** in `internal/status` (and a routing assertion in `internal/cli`) at the level of each claim, per ideation guidance. Baseline gates: `go test ./...` and `go test ./... -race` (AGENTS.md); `gofmt -w ./cmd ./internal`. Estimated complexity: **high** — this is a faithful reimplementation; the work is in matching parser edge cases, sort/format byte-exactness, and the mutation/validation message set. The oracle is the current Python script (and Stage 2's checked-in fixtures/goldens), available locally; no live workflow tests are required.

**Oracle and fixtures.** Reuse Stage 2's frozen workflow fixtures and goldens as the parity oracle so Stage 5 proves "native == vendored == Python" with one fixture set. Add fixtures only for cases Stage 2 did not need (unknown-status row, `_debriefs` presence, `--new` inputs). Provide a documented regenerate path (e.g. `go test -run TestGolden -update`) so oracle drift is visible in review, never silently absorbed.

**Parser/stage tests (AC-2).** Pure table tests — no temp dirs — over hand-written frontmatter/README strings covering every edge case enumerated in the roadmap gates (empty, quoted, nested-ignored, missing frontmatter; defaults, gates, terminal, worktree, feedback-to). Cheap, fast, exhaustive.

**Discovery tests (AC-3).** One fixture tree with flat + folder + conflict + `_archive` + `_mods` + `_debriefs` + dot-dir; assert resolved set, slug sort, and captured warning.

**Read-golden tests (AC-1).** For each read flag, run native + capture stdout, compare to the oracle golden after the **shared normalization applied identically to both sides, in the test only (never in product)**:
- **Wall-clock timestamps** — `YYYY-MM-DDTHH:MM:SSZ` (and ISO-microsecond for sd-b32 default timestamp) → `<TS>` placeholder; not pinnable except via the sd-b32 test hook.
- **Absolute paths** — `--resolve`/`--archive` emit absolute paths; `workflow=` is `realpath`'d (macOS `/var`→`/private/var`), `path=` is not. Run both sides against the same temp root, normalize the root prefix, and apply `realpath` to the expected root for the `workflow=` field specifically. Never bake an absolute path into a golden.
- **sd-b32 `--next-id` / `--boot` NEXT_ID** — pin all material (`SPACEDOCK_TEST_SD_B32_TIMESTAMP`, explicit `--id-seed`/`--id-actor`, controlled `SPACEDOCK_ID_CONTEXT`) so the candidate is reproducible and equals the oracle; additionally assert format (24 chars, SD-B32 alphabet) so a format regression is caught even if the env drifts.
- **`--boot` env sections** — verify structurally: section headers present and in order; ID_STYLE/NEXT_ID/DISPATCHABLE bodies parity-checked; `TEAM_STATE` hint line and live `PR_STATE`/`ORPHANS` rows normalized away; fixtures constructed with no orphans/PRs so those sections render their deterministic `none` forms.
- **Entity ordering** — fixture pins distinct scores/stages, one empty score (locks "empty last"), one unknown status (locks order 99); captured directly by the golden, no normalization.

**Mutation tests (AC-4).** Each copies a fixture into `t.TempDir()`, `git init`s it (absolute-dest archive paths need a real dir), runs the mutation natively and through the oracle into separate temp dirs, then diffs resulting files + normalized stdout. Guard tests assert a terminal `--set` under an active mod-block, a merge-hook-unsatisfied terminal `--set`, and an `--archive` of a mod-blocked entity each exit 1 with the current error text.

**Unknown-field preservation tests (AC-5).** The mutation fixture includes an entity carrying `issue: ENG-123`, `source: linear`, and an arbitrary unknown field; tests assert those exact lines survive a `--set` on a different field and an `--archive` byte-for-byte, diffed against the oracle. Folds external-tracker-checkpoint's A1 preservation assertions into this suite.

**Validation tests (AC-6).** Per-defect fixtures (dup id, bad/non-numeric id, missing id, flat/folder conflict, unknown/invalid stage name) assert stderr lines + exit 1 against the oracle; a clean fixture asserts `VALID`.

**`--new` tests (AC-7).** Pipe a fixture body to `--new` in a temp workflow; assert the written file's minted id equals `--next-id` under identical pinned env for each id-style branch, assert `--validate` is `VALID` immediately after (no id-less window), and assert the slug-exists, missing-fence, and slug-style-with-`--id-seed` error+exit-code paths. A focused test asserts that between the `--new` call's start and finish there is never an on-disk entity with an empty `id` for `sequential`/`sd-b32` (e.g. the write is to a temp file then `os.Rename`d into place, or the body is fully assembled in memory with the id before the single write).

**Byte-for-byte normalization risks flagged** (each must be handled in the test, not the product): trailing-newline behavior of the frontmatter writer — Python's `'\n'.join(content.split('\n'))` is identity-preserving on EOF (verified: a file ending in `\n` round-trips ending in `\n`, one without does not gain one), so the native writer must **preserve the file's existing terminal-newline state exactly**, neither unconditionally appending nor stripping, or mutation goldens diff on EOF; fixed-width column padding and the U+2026 ellipsis truncation, the `%-8s` SOURCE padding that only appears when extra columns follow, sd-b32 5-bit extraction endianness, and the realpath asymmetry between `workflow=` and `path=` in resolver output.

## Notes

Behavioral parity, not internal-shape parity: the native packages may be organized differently from the Python functions as long as the four observable channels match. `state:` split-root (Stage 6) and any PR/mod behavior remain out of scope. The current `skills/commission/bin/status` script (and Stage 2's checked-in fixtures/goldens) is the compatibility oracle for all golden, mutation, and validation parity tests. Decision B (`--new` atomic create + the explicit single id-strictness rule) is the only new surface; it is justified by the entity-creation chicken-and-egg and the `--set`-vs-enumerate strictness inconsistency described in the Problem Statement, and it is designed to remove that inconsistency by construction without altering any existing op's strictness.

## Stage Report: ideation

- DONE: Design the native Go status architecture behind vendor-status-compatibility's narrow interface (frontmatter parser, stage parser, output formatter, mutation engine, validation) matching the current Python tool; behavioral parity spec independent of vendor's exec-vs-copy choice, stated as an assumption.
  Proposed Approach §"Native architecture" decomposes into 7 stdlib-only `internal/status/` units (frontmatter/stages/discover/format/identity/mutate/validate) each pinned to specific Python functions; the exec-vs-copy independence and split-root/PR-as-normal-field assumptions are stated explicitly for FO gate reconciliation.
- DONE: Fold in confirmed decision B — `--new <STDIN` ATOMIC create (mint id + write entity in one op so seeds never exist id-less) AND define id strictness resolving the chicken-and-egg and the --set-vs-enumerate inconsistency.
  Proposed Approach §"Decision B" specifies `--new` (mint via identity engine, stamp id into STDIN frontmatter, single fs write/rename, slug-exists/no-fence/slug-style guards) and states one consistent rule: reads enforce id-presence globally, mutations enforce only target existence, `--new` guarantees id at birth — removing the inconsistency by construction. AC-6 verifies the no-id-less-window property.
- DONE: AC (**AC-N** + Verified by) + golden parity (default/--archived/--next/--where/--fields/--all-fields/--next-id/--resolve/--short-id), mutation (--set, timestamp fill, PR-field preservation, --archive), and validation (dup/bad IDs, flat-folder conflict, unknown stages, terminal/archive guards) tests; flag byte-for-byte normalization risks.
  AC-1..AC-6 in `**AC-N - property.**` + `Verified by:` form; Test Plan covers all nine read flags as oracle goldens, --set/timestamp-fill/PR-mirror/--archive as oracle-diff temp-workflow tests, validation per-defect fixtures, and flags six byte-for-byte risk areas with per-area normalization (timestamps, realpath asymmetry, sd-b32 pinning, --boot structural-only, ordering, EOF-newline — the last verified live against the oracle writer).

### Summary

Designed Stage 5 as a faithful native-Go reimplementation of the current Python status tool behind the Stage 2 `Runner` interface, decomposed into small stdlib `internal/status/` packages (parser, stage parser, discovery, formatter, identity, mutation, validation) each pinned to the oracle's observable contract `(argv,env,cwd) -> (stdout,stderr,exit,fs mutations)`. Folded in confirmed decision B: a `--new <STDIN` atomic create that mints an id and writes a valid seed in one operation, plus an explicitly-stated single id-strictness rule (reads enforce globally, mutations enforce target-only, create guarantees id-at-birth) that removes the entity-creation chicken-and-egg and the `--set`-vs-enumerate inconsistency by construction without changing any existing op's strictness. Notable: verified live against the oracle that the frontmatter writer is EOF-newline-identity-preserving (a real byte-parity trap), and stated exec-vs-copy independence, split-root-out-of-scope, and PR-as-normal-field as explicit assumptions for the FO to reconcile at the gate.

## Stage Report: ideation (cycle 2)

Revision folding in six reconciliations from the staff review and sibling designs. Architecture and decision B unchanged; six additive fixes:

- DONE: EXIT-CODE fact-fix — corrected "{0,1,2}" to "{0,1}".
  Verified against the oracle (`grep`): 7×`sys.exit(0)`, 55×`sys.exit(1)`, zero `sys.exit(2)`, no `argparse`. Problem Statement now states usage/parse errors exit 1 (not 2) and the native runner must reproduce `{0,1}`; clarified the launcher's own exit-2 (`internal/cli/`) is a separate, unchanged boundary.
- DONE: UNKNOWN-FIELD preservation — added AC-5 reconciling external-tracker-checkpoint's assumption A1.
  New AC-5: `issue`/`source`/arbitrary unknown top-level fields survive `--set`/`--archive` byte-for-byte (the in-place line-oriented mutator already does this; now an explicit guarantee with a fixture entity). AC-6/AC-7 renumbered from former AC-5/AC-6; test plan adds an unknown-field-preservation suite.
- DONE: RESOLUTION seam — added `resolveRoots(workflowDir) -> (definitionDir, entityDir)` (unit 0).
  `definitionDir==entityDir==workflowDir` in Stage 5; `definitionDir` threads to stage+identity parsing, `entityDir` to discovery/mutation/archive/validation. Turns native-state-dir (Stage 6) into a one-function extension; stated as the seam Stage 6 flagged, with an FO-reconcile assumption that Stage 6 owns the split + `state:` read.
- DONE: ENV/CWD dependence — added a dedicated subsection enumerating env/cwd-derived inputs.
  Lists `SPACEDOCK_TEST_SD_B32_TIMESTAMP`, `SPACEDOCK_ID_ACTOR`/`USER`/`USERNAME`, `SPACEDOCK_ID_CONTEXT`, `HOME` (TEAM_STATE), `PATH` (gh/worktree), cwd (`--discover` git rev-parse/getcwd), and the `realpath` asymmetry — and how the parity harness pins each.
- DONE: STDIN justification — stated `--new` is what makes the Runner's `stdin` parameter load-bearing.
  Added to Decision B: the Stage-2 oracle never reads stdin; `--new` is the first consumer (entity body on stdin), resolving the staff-review "speculative stdin param" concern.
- DONE: CONCURRENCY/locking — folded atomic-write + tool-owned commit serialization into decision B as a design principle.
  Oracle does in-place rewrite + `os.rename` with no locking (this session hit the shared-index race). Native principle: assemble-in-memory + temp-file + `os.Rename` for `--set`/`--archive`/`--new`; tool owns path-scoped state commits rather than free-form `git commit`. Stated as principle with an FO-reconcile assumption on the exact lock/commit primitive; no read-path change.

### Summary

Applied all six requested reconciliations as additive edits — the native architecture and decision B are intact. The load-bearing exit-code fact was independently re-verified against the oracle (only `{0,1}`, never `2`) and corrected. The `resolveRoots` seam now pre-threads the Stage 6 split-root distinction while both roles equal `workflowDir`, so native-state-dir becomes a clean extension. Unknown-field preservation is promoted to an explicit AC (AC-5) so external-tracker-checkpoint's A1 folds into this mutation suite. Env/cwd-derived inputs are enumerated for the parity harness, the `--new` stdin justification is recorded, and the atomic-write + tool-owned-commit-serialization principle is folded into decision B (directly motivated by this session's shared-index race). Three new FO-reconcile assumptions are stated explicitly: Stage 6 owns the split logic, the exact lock/commit primitive is an implementation choice, and the Runner signature is whatever `internal/status/` exports at gate time.
