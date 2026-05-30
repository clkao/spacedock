---
id: nc85apg7333k7c594qam5m2n
title: Implement native state-dir support
status: ideation
score: "0.55"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:30:15Z
---

# Implement Native State-Dir Support

Teach native Go status to read workflow definition from the main README and mutable entities from the `state:` path, with no README symlink inside the state checkout.

## Problem statement

The symlink-state-profile (Stage 3) lets current tooling run against a per-workflow `.spacedock-state` checkout by symlinking `.spacedock-state/README.md -> ../README.md`. That bridge has two costs that the native split-root mode must eliminate:

1. **`docs/dev` reads empty.** `docs/dev/README.md` carries `state: .spacedock-state` and the mutable entities live under `docs/dev/.spacedock-state/`, not beside the README. Tooling that does not understand `state:` finds no entities when pointed at `docs/dev` directly; only `--workflow-dir docs/dev/.spacedock-state` works, and only because the symlinked README backfills the stage definition there. The operator must aim at the state checkout, not the workflow.

2. **Discovery double-counts.** `discover_workflows` walks the tree, follows symlinks (`followlinks=True`), and treats any directory with a `README.md` whose frontmatter starts `commissioned-by: spacedock@` as a workflow. The symlinked `.spacedock-state/README.md` resolves to a commissioned README, so both `docs/dev` and `docs/dev/.spacedock-state` match. The realpath de-dup keys on the *directory* path, not the README target, so the two distinct directories are reported as two workflows for what is one logical workflow.

Native split-root makes the workflow README the single definition source and the state checkout the single mutable-entity source, addressing both behaviors without any symlink.

This task depends on native-go-status (Stage 5), which delivers the Go-native frontmatter parser, README stage parser, entity discovery (flat + folder form), and the `status` / `--set` / `--archive` / `--validate` / discovery commands at parity with the current script. This task extends that native implementation; it does not re-implement parsing or discovery.

### Assumption on the native-go-status interface (reconcile at gate)

This design assumes Stage 5 exposes, in `internal/status/`, a resolution seam that turns a single `--workflow-dir` into the directory that holds the stage-defining `README.md` (the *definition dir*) and the directory that holds active entities and `_archive/` (the *entity dir*). In current single-root mode those are the same directory. This task makes the entity dir diverge from the definition dir when `state:` is set. If Stage 5 instead hardcodes "entities live beside the README" at every call site, the FO should flag it at the gate: the split-root work then also threads the entity dir through `scan_entities`-equivalent, `resolve_entity_path`-equivalent, `run_archive`-equivalent, and validation, rather than just populating one resolver. Either way the externally observable behavior in the ACs below is unchanged.

## Proposed approach

### Root resolution

Define two roots, matching `docs/specs/state-behavior-extension.md`:

```text
definition_dir = directory passed via --workflow-dir (holds README.md)
state_value    = `state:` field in definition_dir/README.md frontmatter (e.g. ".spacedock-state")
entity_dir     = definition_dir / state_value   (when state_value is set and non-empty)
               = definition_dir                 (when state_value is absent or empty)
```

`state_value` is resolved relative to the definition dir, joined and cleaned with the stdlib path joiner. An absolute `state_value`, or one that escapes the definition dir via `..`, is rejected with a clear error rather than silently followed — the v0 contract is a child checkout named `.spacedock-state`. Reads of the README, stage block, and `id-style`/identity rules always come from `definition_dir/README.md`. Reads and writes of entities always come from `entity_dir`.

Concretely, this means `--workflow-dir docs/dev` resolves stages from `docs/dev/README.md` and entities from `docs/dev/.spacedock-state/`. No `.spacedock-state/README.md` is consulted and none is required.

### Threading the entity dir

Every code path that currently assumes "entities are beside the README" splits into definition-dir vs entity-dir:

- stage parsing, `id-style`, identity/duplicate-ID allocation across active+archived: read from `definition_dir/README.md`.
- active entity scan, folder-form `index.md` discovery, `_archive` scan: read from `entity_dir`.
- `--set` frontmatter update: write to the entity file under `entity_dir`.
- `--archive`: stamp `archived:` and move `{entity_dir}/{slug}` (or `.md`) to `{entity_dir}/_archive/`.
- `--validate`, `--next`, `--resolve`, `--short-id`, `--where`, `--fields`: stages from `definition_dir`, entities from `entity_dir`.

The `_archive` directory is `entity_dir/_archive`, never `definition_dir/_archive`. Identity allocation still spans active + archived entities, all under `entity_dir`.

### Discovery (`--discover`)

Discovery must find the main workflow README once and never surface `.spacedock-state` as a second workflow. The rule: a directory is a workflow when its own `README.md` frontmatter starts `commissioned-by: spacedock@`. A state checkout in native mode has no `README.md` (no symlink), so it cannot match. To make this robust even if a stray symlinked README lingers, discovery also prunes any directory equal to a `state:` target of an already-discovered workflow, and prunes `.spacedock-state` by basename the way `DISCOVER_IGNORE_DIRS` prunes `.worktrees`. This means: walk, on hitting a commissioned README record the workflow and compute its `entity_dir`, and exclude that `entity_dir` from being independently reported. The main README is found; the state checkout is ignored.

### What stays the same

Single-root workflows (no `state:` field) behave exactly as Stage 5 ships them — `entity_dir == definition_dir`. Mods, PR guards, and merge-hook behavior remain out of scope. Output format, table columns, and validation messages are unchanged; only the *source directories* for definition vs entities differ.

## Acceptance criteria

Each AC names a property of the finished entity and how it is verified.

**AC-1 - `state:` resolves to a child entity dir relative to the workflow README directory.**
For a workflow whose `README.md` frontmatter contains `state: .spacedock-state`, the native status resolver computes `entity_dir = definition_dir/.spacedock-state` while keeping `definition_dir` as the stage/identity source; an absent or empty `state:` leaves `entity_dir == definition_dir`.
Verified by: a Go unit test in `internal/status/` over the resolver seam with cases `{state set, state absent, state empty}`, asserting the resolved definition and entity dirs; plus a malformed case (`state: /abs` and `state: ../escape`) asserting a rejection error.

**AC-2 - Split-root status renders entities with no README symlink in the state checkout.**
Given a fixture workflow with `README.md` in the definition dir and entities under `.spacedock-state/` with NO `.spacedock-state/README.md`, `spacedock status --workflow-dir <definition_dir>` lists those entities and renders stage columns from the main README.
Verified by: an integration test that builds a temp split-root fixture (no symlink), runs the status command, and asserts the active entities and stage-derived columns appear; a sub-assertion confirms `<definition_dir>/.spacedock-state/README.md` does not exist during the run.

**AC-3 - Stages and identity come from the main README; entities come from the state checkout.**
Stage declarations, `id-style`, and gate/terminal/worktree flags are read from `definition_dir/README.md`; active and archived entities, and ID-uniqueness allocation, are read from `entity_dir`.
Verified by: a fixture where the main README defines a stage set and an `id-style`, and the state checkout holds entities whose IDs must be unique across active+archived; the status/`--validate` output reflects the README's stages and flags a duplicate ID sourced only from `entity_dir`.

**AC-4 - `--set` mutates only state-checkout files.**
`spacedock status --workflow-dir <definition_dir> --set <slug> status=implementation` rewrites the frontmatter of the entity under `entity_dir` and changes no file under `definition_dir` outside `entity_dir` (in particular not `README.md`).
Verified by: an integration test that snapshots the definition dir's non-state files before and after `--set`, asserting only `entity_dir/<slug>` changed; reinforced by a `git status`-style assertion (using a temp git layout mirroring the nested-repo split) that the main repo shows no churn while the state checkout shows exactly the entity change.

**AC-5 - `--archive` moves only state-checkout files.**
`spacedock status --workflow-dir <definition_dir> --archive <slug>` stamps `archived:` and moves the entity to `entity_dir/_archive/`, touching no file under `definition_dir` outside `entity_dir`.
Verified by: an integration test that archives both a flat and a folder-form entity, asserting the source no longer exists under `entity_dir`, the destination exists under `entity_dir/_archive/`, `_archive` is created under `entity_dir` (never `definition_dir`), and the definition dir's non-state files are untouched.

**AC-6 - Discovery finds the main workflow README and ignores the state checkout.**
`spacedock status --discover` (or the discovery entrypoint) over a tree containing a split-root workflow returns the definition dir exactly once and never returns `.spacedock-state` as a separate workflow, with or without a stray symlinked README inside the state checkout.
Verified by: a discovery integration test with two sub-cases — (a) native split-root, no `.spacedock-state/README.md`: assert exactly one workflow path returned (the definition dir); (b) a `.spacedock-state/README.md` symlink present: assert discovery still returns exactly one workflow (the definition dir) and the state checkout is pruned, proving the double-count is resolved.

**AC-7 - Single-root workflows are unchanged.**
A workflow with no `state:` field resolves `entity_dir == definition_dir` and behaves byte-for-byte as the Stage 5 native implementation for status, `--set`, `--archive`, and discovery.
Verified by: reusing the Stage 5 single-root golden fixtures and asserting unchanged output once the split-root code path is added (regression guard that split-root resolution did not perturb the default path).

## Test plan

All tests are Go (`internal/status/`), the same surface Stage 5 establishes. No live workflow run is required — split-root behavior is fully exercisable with temp fixtures, which is cheaper and deterministic.

| Test | Kind | Proves | Cost |
|------|------|--------|------|
| resolver unit (`state` set/absent/empty/malformed) | Go unit | AC-1 | low |
| split-root status, no symlink | Go integration (temp fixture) | AC-2 | low |
| README-stages + state-entities + dup-ID validate | Go integration | AC-3 | low |
| `--set` mutates only state checkout | Go integration (+ temp nested-git assertion) | AC-4 | medium |
| `--archive` moves only state checkout (flat + folder) | Go integration | AC-5 | low |
| discovery: split-root single-count, both sub-cases | Go integration | AC-6 | medium |
| single-root regression against Stage 5 goldens | Go integration | AC-7 | low |
| `go test ./...` (and `-race` per AGENTS.md) | suite gate | whole stage green | low |

Riskiest-path-first: AC-1 (resolver) and AC-2 (no-symlink render) are the smallest end-to-end exercises of the new contract and should land before the mutation/discovery work; they validate the split-root mechanism before the broader fixtures are built.

Out of scope (consistent with the roadmap and AGENTS.md): mods, PR/merge guards, external-tracker writeback, and any change to output formatting. The first native split-root version should be boring.

## Notes

Mods and PR guards remain out of scope. The first native split-root version should be boring. Stage 7 (retest without symlink) is the downstream consumer that removes `.spacedock-state/README.md` and reruns against this mode; this task must make that removal a no-op for behavior.

## Stage Report: ideation

- DONE: Design native `state:` split-root resolution: stages parsed from the MAIN workflow README, entities read from the state checkout, NO README symlink required; discovery finds the main workflow README and IGNORES .spacedock-state (resolving the discover-double-count and docs/dev-reads-empty behaviors).
  Root resolution (definition_dir vs entity_dir), threading, and discovery rule designed in "Proposed approach"; AC-2/AC-3/AC-6 verify no-symlink render, README-as-stage-source, and single-count discovery. Double-count root cause traced to `discover_workflows` following the symlinked README (status:2000-2065); docs/dev-reads-empty traced to entities living under state_dir while tooling reads beside README.
- DONE: Specify that --set mutates ONLY state-checkout files and --archive moves ONLY state-checkout files, so the main repo shows no runtime state churn.
  AC-4 (`--set` only state checkout, README untouched) and AC-5 (`--archive` moves to `entity_dir/_archive`, definition dir untouched), each with a nested-git no-churn assertion mirroring the real layout (.spacedock-state is its own nested git repo, confirmed via `git rev-parse --show-toplevel`).
- DONE: AC (**AC-N** + Verified by) + test plan per Stage 6 required tests; depends on native-go-status (Stage 5) — design to its parser/interface, stating the assumption.
  Seven ACs in `**AC-N - property.**` + `Verified by:` form covering all six Stage 6 required tests plus a single-root regression guard; test-plan table maps each test to its AC, kind, and cost. Stage 5 dependency stated explicitly under "Assumption on the native-go-status interface" for FO gate reconciliation.

### Summary

Designed native split-root as a definition_dir/entity_dir seam layered on Stage 5's native status: stages and identity read from the main `README.md`, entities read and mutated under `entity_dir = definition_dir/<state:>`, with `_archive` always under `entity_dir`. Root-caused both target behaviors by reading the current Python `status` script: the discover-double-count comes from `discover_workflows` following the symlinked state README, and docs/dev-reads-empty comes from entities living under the state checkout while non-`state:`-aware tooling reads beside the README. The Stage 5 interface dependency is stated as an explicit assumption (a resolution seam that diverges entity dir from definition dir) for FO reconciliation at the gate; all proof is Go fixture/unit tests, no live run.
