---
id: bdxxg3yr1ys4nsmwk8x65j7b
title: Prove symlink state profile
status: implementation
score: "0.85"
source: bootstrap roadmap
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-symlink-state-profile
started: 2026-05-30T04:30:15Z
---

# Prove Symlink State Profile

Prove that a per-workflow `.spacedock-state` checkout runs current `spacedock status` behavior unchanged when its `README.md` symlinks back to the workflow README in the main repo.

## Problem Statement

The bootstrap separates workflow *definition* (the README, in the main code repo) from workflow *runtime state* (mutable entities, in a per-workflow `.spacedock-state` checkout) so that issue/state churn does not land on the code branch (`docs/specs/state-behavior-extension.md`). The native split-root reader that understands `state: .spacedock-state` and reads stages from one directory while reading entities from another does not exist yet — that is Stage 6 (native-state-dir, roadmap §6, `docs/dev/.spacedock-state/native-state-dir/index.md`).

This stage proves the **compatibility bridge** that lets the *unmodified* status runner operate on that split layout before native support exists. The mechanism is a single symlink: `.spacedock-state/README.md -> ../README.md`. With it, the runner can be pointed at the state checkout alone (`--workflow-dir .../.spacedock-state`) and find everything it needs in one directory — stages via the symlinked README, entities from the state checkout itself.

The status runner this stage exercises is supplied by Stage 2 (vendor-status-compatibility, `docs/dev/.spacedock-state/vendor-status-compatibility/index.md`): `spacedock status` forwards argv (including `--workflow-dir`) to a vendored copy of the current Python script behind a narrow `Runner` interface, streaming stdout/stderr/exit-code transparently. **Assumption (FO reconcile at gate):** Stage 2 lands the `spacedock status` → vendored-runner route with `--workflow-dir` pass-through. This stage's tests invoke `spacedock status --workflow-dir <state-dir>` and depend on that route existing; if Stage 2 is not yet merged when this stage is implemented, these tests target the vendored runner through whatever entry point Stage 2 defines. This stage adds **no** product code of its own to the runner — it is a layout-and-integration proof. The only product-side surface it may touch is a checked-in test fixture and the integration test itself.

The risks this stage retires are four observable properties of the symlink layout, each a way the bridge could silently misbehave:

1. **README resolution through the symlink** — the runner reads `os.path.join(workflow_dir, 'README.md')` (`skills/commission/bin/status` `get_id_style`/stages path); because `open()` follows symlinks, `.spacedock-state/README.md -> ../README.md` must transparently yield the real stages block. If the symlink were broken or pointed wrong, `--next` and stage ordering would silently degrade.
2. **Entity rendering from the state checkout** — active entities live directly under `.spacedock-state` (no `entities/` dir, per `AGENTS.md` State-Branch Bootstrap Rules), and `discover_entity_files` must list them from exactly that directory.
3. **Folder-form non-misdetection** — `discover_entity_files` treats a child directory as an entity only when it contains `index.md` at its own root and skips `_archive`/`_mods` (`RESERVED_SUBDIRS`). A folder-form entity's `reports/` and `artifacts/` subdirectories contain no root-level `index.md`, so they must NOT surface as separate entities. The scan is single-level (`os.listdir`), so `{slug}/reports/foo.md` is never reachable as an entity. This is the property most likely to regress if discovery ever became recursive.
4. **Archive move under the state checkout** — `run_archive` moves a folder entity `{workflow_dir}/{slug}/` → `{workflow_dir}/_archive/{slug}/` wholesale (carrying `reports/`/`artifacts/` with it) and stamps `archived:` in `index.md` first. With `--workflow-dir .../.spacedock-state` this lands the archive under `.spacedock-state/_archive`, where the active scan no longer sees it and `--archived` does.

## Proposed Approach

**One integration test that builds the real symlink layout in a temp repo and drives the actual runner.** No new product code; the proof is that the *existing* runner, pointed at a faithfully-constructed `.spacedock-state` profile, exhibits all four properties above. Per the ideation guidance, proof sits at the level of the claim — the claim is about on-disk layout + runner behavior, so the test constructs the layout on disk and asserts runner output and filesystem effects, not internal functions.

**Fixture construction (in `t.TempDir()`, not a static checked-in tree, because the layout contains a symlink and a git-init'd dir).** The test builds:

```text
<tmp>/docs/dev/
  README.md                       # real workflow README with a stages: block + id-style
  .spacedock-state/
    README.md -> ../README.md     # the compatibility symlink (os.Symlink / os.symlink)
    add-login.md                  # flat-form active entity
    refactor-dispatch/            # folder-form active entity
      index.md
      reports/
        ideation.md               # a stage report WITH frontmatter — the misdetection trap
      artifacts/
        notes.md
    seed-archive/                 # second folder-form entity, used to prove the archive move
      index.md
      reports/
        ideation.md
```

The fixture README and entity bodies are checked in under `internal/status/testdata/symlink-profile/` as plain files; the test copies them into the temp tree and creates the symlink at runtime (symlinks do not survive a `go test` checkout reliably across platforms, so the test materializes it). `git init` the temp `docs/dev` tree only if a driven command needs a git root (archive does not require git for the move itself; it runs no git). The `reports/ideation.md` file is given **real frontmatter** (an opening `---` fence and an `id:`) precisely so that a naively-recursive discovery would mistake it for an entity — making the non-misdetection assertion meaningful rather than vacuous.

**Driving the runner.** All assertions go through `spacedock status --workflow-dir <tmp>/docs/dev/.spacedock-state ...` (the Stage 2 route). The test asserts on stdout/exit-code and on the post-archive filesystem. Two flat fixtures of stage values let the default table ordering be deterministic.

**Scope guards (explicit).**
- **OUT: native split-root (`state:` resolution without a symlink).** This stage never points the runner at the main `docs/dev` directory and never exercises `state:`-field resolution. That is Stage 6 (native-state-dir). The boundary is concrete: *this* stage always passes `--workflow-dir .../.spacedock-state` and relies on the symlink; Stage 6 passes `--workflow-dir <main>` and relies on the `state:` field with no symlink.
- **OUT: mods and PR behavior.** No `_mods`, no merge-hook, no PR-state assertions (`AGENTS.md` priorities; roadmap §3 "no mod or PR behavior is required").
- **OUT: reimplementing or modifying the runner.** The runner is Stage 2's vendored oracle; this stage only constructs a layout and asserts behavior.
- This stage adds only an integration test plus its `testdata` fixture; it touches no `internal/cli` or `internal/status` product code.

## Acceptance Criteria

**AC-1 - A `.spacedock-state` checkout whose `README.md` symlinks to `../README.md` renders its active entities through `spacedock status --workflow-dir <state-dir>`.**
With the symlink layout in place, the default status table lists every active entity living directly under `.spacedock-state` (flat `add-login` and folder-form `refactor-dispatch`/`seed-archive`), with stage/score columns populated from each entity's frontmatter and rows ordered by the stages block read *through* the symlinked README. The runner is invoked with no flag beyond `--workflow-dir <state-dir>` and exits 0.
Verified by: integration test asserting the default-table stdout (timestamps/paths normalized) lists exactly the active entities and no others, and that the stage ordering matches the stages block; a companion assertion that `--next` reads dispatchable entities, proving the stages block resolved through the symlink rather than failing with "README.md has no stages block".

**AC-2 - A folder-form entity's `reports/` and `artifacts/` subdirectories are not discovered as separate entities, even when a file inside them carries frontmatter.**
`refactor-dispatch/reports/ideation.md` has a valid opening `---` fence and an `id:` field, yet the status table contains exactly one row for `refactor-dispatch` and no row for `reports`, `artifacts`, `ideation`, or any nested path. Discovery surfaces a child directory only when it holds a root-level `index.md`, and `_archive`/`_mods` are reserved.
Verified by: integration test asserting the active-entity slug set equals `{add-login, refactor-dispatch, seed-archive}` exactly — no `reports`/`artifacts`/`ideation` slug appears — and that the count of data rows in the default table equals 3.

**AC-3 - Archiving a folder-form entity moves the whole folder (with its `reports/` and `artifacts/`) under `.spacedock-state/_archive` and removes it from the active view.**
`spacedock status --workflow-dir <state-dir> --archive seed-archive` stamps `archived:` in `seed-archive/index.md`, then relocates `.spacedock-state/seed-archive/` to `.spacedock-state/_archive/seed-archive/` including its `reports/` subtree. Afterward the default table no longer lists `seed-archive`, and `--archived` does list it.
Verified by: integration test that runs `--archive seed-archive`, then asserts (a) `.spacedock-state/seed-archive/` no longer exists, (b) `.spacedock-state/_archive/seed-archive/index.md` exists and contains an `archived:` line, (c) `.spacedock-state/_archive/seed-archive/reports/ideation.md` exists (subtree carried), (d) the default table omits `seed-archive`, and (e) `--archived` includes it.

**AC-4 - The compatibility layout works end-to-end through the `spacedock status` command surface, not by calling discovery internals directly.**
Every assertion in AC-1..AC-3 is produced by invoking the `spacedock status` entry point with real argv against the constructed temp layout — the same path the first officer uses — so the proof covers the symlink + runner integration, not just an isolated function.
Verified by: the integration test invoking `spacedock status` (the Stage 2 command route) with `--workflow-dir` for all of the above; no assertion calls a status-internal Go function or the discovery routine directly.

## Test Plan

**Single integration test file**, `internal/status/symlink_profile_test.go` (or `internal/cli/` if Stage 2's command route is reached more naturally there — placement follows wherever the `spacedock status` entry point is invokable in-process; if the route only exists as a subprocess, the test execs the built binary). Baseline gate: `go test ./...` and `go test ./... -race` (`AGENTS.md`). Estimated complexity: **low-to-moderate** — there is no new product logic; the work is faithful fixture construction (symlink + folder-form + nested report) and three groups of assertions. No live workflow test, no `gh`, no network.

**Fixtures (named):**
- `internal/status/testdata/symlink-profile/README.md` — workflow README with a `stages:` block (at least two ordered stages so table ordering is testable) and an `id-style` (`sd-b32` or `sequential`; pick `slug` if entity IDs are awkward to pin, since slug-style needs no `--next-id`). Checked in as a normal file.
- `internal/status/testdata/symlink-profile/add-login.md` — flat-form entity, frontmatter only, distinct stage.
- `internal/status/testdata/symlink-profile/refactor-dispatch/index.md` — folder-form entity; `reports/ideation.md` (WITH `---` fence + `id:` — the misdetection trap) and `artifacts/notes.md` (no frontmatter).
- `internal/status/testdata/symlink-profile/seed-archive/index.md` — second folder-form entity with a `reports/ideation.md`, archived during the test.

**Test body (smallest proof surface, in order):**
1. **Build layout.** Copy the `testdata/symlink-profile/` tree into `t.TempDir()/docs/dev/.spacedock-state/`; place a sibling real README so the symlink target exists, then `os.Symlink("../README.md", <state>/README.md)`. (Constructing the symlink at runtime, not in `testdata`, avoids relying on the VCS preserving a symlink.)
2. **AC-1 + AC-2 (one `status` invocation).** Run `spacedock status --workflow-dir <state>`; normalize timestamps/abs-paths; assert the data-row slug set is exactly `{add-login, refactor-dispatch, seed-archive}`, row count is 3, ordering follows the stages block, and no `reports`/`artifacts`/`ideation` row appears. Run `spacedock status --workflow-dir <state> --next` and assert it does not error with "no stages block" (proves README resolved through symlink).
3. **AC-3 (archive).** Run `spacedock status --workflow-dir <state> --archive seed-archive`; assert the five filesystem + table conditions in AC-3 (folder gone from active root, present under `_archive/` with `archived:` stamp and carried `reports/` subtree, omitted from default table, present in `--archived`).
4. **AC-4** is satisfied structurally: every step above goes through the `spacedock status` argv surface.

**Normalization strategy** (mirrors Stage 2's, applied in-test to runner output, never in product): replace ISO-8601 UTC timestamps (`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z` → `<TS>`) and the temp-root prefix (→ `<ROOT>`) before any string assertion. The archive `archived:` line is asserted by presence + shape, not exact instant. No golden file is required for this stage — the assertions are set-membership, row-count, and filesystem existence, which are inherently env-independent; a golden table can be added later if Stage 5 wants byte-parity, but it is unnecessary to prove the symlink-profile properties.

**Why this is the smallest proof surface:** the four risks all live in *layout × runner discovery/archive*. A single temp-tree fixture exercising one flat entity, one folder entity with a frontmatter-bearing nested report (the only non-vacuous way to test misdetection), and one archive move covers every risk with one fixture and three runner invocations. Adding more entities or flags would not retire additional symlink-profile risk; the read-parity and mutation-parity of individual flags is Stage 2's job, and `state:`-field resolution is Stage 6's.

## Notes

This is the bridge that lets current tooling work before native split-root support exists. The runner is unmodified; the only moving part is the `.spacedock-state/README.md -> ../README.md` symlink plus the on-disk entity layout. The compatibility oracle is the current script at `skills/commission/bin/status`, reached through Stage 2's `spacedock status` route. Native `state:` split-root (no symlink) is deliberately out of scope here and is proved by Stage 6 (native-state-dir).

## Stage Report: ideation

- DONE: Rewrite AC into **AC-N** + Verified by for the SYMLINK compatibility phase: docs/dev/README.md plus .spacedock-state/README.md -> ../README.md; status renders entities from the state checkout; folder-form entities with reports/ and artifacts/ subdirs are NOT misdetected; archived entities move under .spacedock-state/_archive.
  AC-1 (render through symlinked README), AC-2 (folder-form non-misdetection), AC-3 (archive move under _archive carrying reports/ subtree), AC-4 (proven through the `spacedock status` command surface). Grounded in `skills/commission/bin/status`: `discover_entity_files` (folder = `{slug}/index.md`, `RESERVED_SUBDIRS={_archive,_mods}`, single-level `os.listdir`), README read via `os.path.join(workflow_dir,'README.md')` (follows symlink), `run_archive` (whole-folder move to `_archive/{slug}/`).
- DONE: Test plan — an integration test that builds the symlink layout in a temp workflow and asserts status rendering, folder-form non-misdetection, and the archive move; name the fixtures and the smallest proof surface.
  Single test `internal/status/symlink_profile_test.go`; fixtures under `internal/status/testdata/symlink-profile/` (README with stages block; flat `add-login.md`; folder `refactor-dispatch/index.md` + `reports/ideation.md` WITH frontmatter as the misdetection trap + `artifacts/notes.md`; folder `seed-archive/` archived during test). Symlink created at runtime via `os.Symlink`. Smallest surface = 1 temp fixture, 3 runner invocations (default table, `--next`, `--archive`). Assertions are set-membership / row-count / filesystem-existence (env-independent), no golden needed.
- DONE: Explicitly scope OUT native split-root (state: resolution without a symlink) — Stage 6 native-state-dir; this stage proves the SYMLINK phase only; no mod or PR behavior.
  Scope guards in Proposed Approach: this stage always passes `--workflow-dir .../.spacedock-state` and relies on the symlink; Stage 6 passes `--workflow-dir <main>` with the `state:` field and no symlink. No `_mods`, no PR/merge-hook assertions.

### Summary

Designed Stage 3 as a no-product-code integration proof: build the real `.spacedock-state/README.md -> ../README.md` symlink layout in a temp tree and drive the unmodified status runner (supplied by Stage 2's `spacedock status` route) to assert four properties — entities render through the symlinked README, a folder entity's frontmatter-bearing `reports/`/`artifacts/` subdirs are not misdetected as entities, archiving moves the whole folder under `.spacedock-state/_archive`, and all of it goes through the `spacedock status` argv surface. Every claim is grounded in the current `skills/commission/bin/status` mechanics (`discover_entity_files`, symlink-following README read, `run_archive`). Explicit assumption flagged for FO gate reconciliation: depends on Stage 2 landing the `spacedock status` → vendored-runner route with `--workflow-dir` pass-through. Native `state:` split-root is scoped OUT (Stage 6); no mods/PR behavior.
