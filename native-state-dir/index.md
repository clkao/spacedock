---
id: nc85apg7333k7c594qam5m2n
title: Implement native state-dir support
status: validation
score: "0.55"
source: bootstrap roadmap
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-native-state-dir
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

## Stage Report: implementation

- DONE: Native `state:` split-root: extend the resolveRoots seam so entityDir = definitionDir/<state> (reject absolute or `..`-escaping); stages + identity from definitionDir/README.md, entity scan/_archive/--set/--archive only under entityDir; single-root unchanged; discovery finds the main README once and prunes/ignores .spacedock-state with or without a stray symlinked README.
  `resolveRoots` (roots.go) reads `state:` and returns (roots, error); Stage 5 had already threaded definitionDir/entityDir through every call site, so no other unit changed. Discovery prunes by basename + resolved `state:` target (handlers.go). Code commit 2183dd2 on spacedock-ensign/native-state-dir.
- DONE: PRODUCTION FLIP + carried boot fix: cli.Run defaults to NativeRunner; scanOrphans uses pyJoin so an absolute worktree: is probed as-is (os.path.join semantics) matching the oracle's DIR_EXISTS.
  cli.go flip + boot.go:50 pyJoin. All existing tests stay green incl. zz parity, vendor golden/mutation, and TestSymlinkStateProfile (pins VendorRunner). `go run ./cmd/spacedock status --workflow-dir <main>/docs/dev` lists state-checkout entities, renders README stages, `--boot` shows the absolute worktree DIR_EXISTS=yes, and `--discover` returns docs/dev exactly once with the compat symlink present.
- DONE: New split-root parity tests: no-symlink status, README-stages+dup-ID validate, --set/--archive mutate only the state checkout, discovery single-count (both sub-cases), and a boot absolute-worktree ORPHANS parity test vs the oracle.
  native_state_dir_test.go (AC-1..AC-6, 14 cases) + boot_orphan_abs_test.go (1 case). AC-7 single-root regression covered by the unchanged Stage 5 goldens. `go test ./...` 240 passed, `-race` 240 passed, gofmt + vet clean.

### Summary

Implemented native split-root entirely in the `resolveRoots` seam (Stage 5 had already threaded definitionDir/entityDir through every call site): a README `state:` field diverges entityDir to definitionDir/<state>, with absolute/`..`-escaping values rejected. Discovery prunes the state checkout by basename and by each workflow's resolved `state:` target, resolving the double-count. Flipped the production default to NativeRunner and fixed the carried boot.go absolute-worktree ORPHANS parity (pyJoin vs filepath.Join). NOTABLE: the AC-5 launcher smoke test pointed `--workflow-dir` at the state checkout (the legacy symlink-compat operator model); under a native default that path's symlinked README re-applies `state:` and resolves a nested non-existent dir, so I repointed it at the definition dir — the native operator model the flip exists to enable and what Stage 7 will use. TestSymlinkStateProfile stays green because it pins VendorRunner.

## Stage Report: validation

- DONE: Reproduce native `state:` split-root INDEPENDENTLY: build the worktree binary and (a) against the REAL workflow run `status --workflow-dir docs/dev` — list .spacedock-state entities + render stages from docs/dev/README.md byte-identical to the old vendored path.
  Built `/tmp/spacedock` (commit 2183dd2). `/tmp/spacedock status --workflow-dir docs/dev` and oracle `--workflow-dir docs/dev/.spacedock-state` produced identical stdout+stderr, same sha256 (87b0af37…). Three .spacedock-state entities listed, stages from docs/dev/README.md.
- DONE: (b) fresh no-symlink split-root fixture: --set mutates ONLY the entity under state dir (definition dir incl README untouched), --archive moves ONLY under <state>/_archive, --validate reads stages from main README, --discover returns the definition dir EXACTLY once (with and without a stray symlinked state README).
  Independent /tmp fixtures (no symlink): --set changed only state entity, README sha unchanged before/after; --archive moved flat+folder (reports subtree) to state/_archive, no _archive under def dir, README untouched; --validate flagged a main-README bad stage-name AND a state-dir dup-id (oracle-parity on symlink layout); --discover returned def dir exactly once in BOTH no-symlink and stray-symlink cases (oracle double-counts both).
- DONE: Confirm the PRODUCTION FLIP: cli.go default is NativeRunner so the real `spacedock status` runs native; the carried boot.go ORPHANS fix makes an ABSOLUTE `worktree:` field report DIR_EXISTS matching the oracle; and NO regression — full suite green.
  cli.go:19 `Run` constructs `&status.NativeRunner{}`; flip also proven by binary behavior (split-root render + single-count discover are native-only — VendorRunner default would fail both). Real workflow `--boot` and an independent abs-worktree-outside-git-root fixture both report DIR_EXISTS=yes matching the oracle (pyJoin). go test ./... 240 run / 0 fail / 0 skip; -race 0 fail no data races; `gofmt -l .` empty; `go vet ./...` exit 0. All dispatch-named tests PASS: native_state_dir (AC-1..6, incl. resolver set/absent/empty + abs/escape reject), boot_orphan_abs, 14 zz Ind* parity, vendor golden/mutation, skills/integration, TestSymlinkStateProfile (pins VendorRunner).
- DONE: Give a PASSED or REJECTED recommendation backed by reproduced evidence.
  PASSED — see recommendation below.

### Summary

PASSED. Independently reproduced every claim against the worktree binary, not the prior agent's run. Native `--workflow-dir docs/dev` is byte-identical (same sha256) to the old vendored oracle path against the real workflow; native `--discover` returns docs/dev exactly once while the oracle still double-counts (the headline fix), holding with and without the stray symlinked state README. On fresh no-symlink fixtures, --set/--archive touch only the state checkout (README sha unchanged, _archive only under state dir), --validate sources stage-name checks from the main README and dup-id from the state dir at oracle parity, and absolute/`..`-escaping `state:` values are rejected end-to-end. The production flip is real (cli.go:19 NativeRunner) and the carried boot.go absolute-worktree ORPHANS fix matches the oracle (DIR_EXISTS=yes) both on the real workflow and an independent outside-git-root fixture. Full gates green: 240 tests / 0 fail / 0 skip, -race clean (no data races), gofmt -l empty, go vet exit 0. One non-blocking observation: the cli flip's default is verified by inspection + binary behavior but no unit test exercises the public `cli.Run` entrypoint's default runner (the existing test injects NativeRunner into the unexported `run` core); behavior is nonetheless conclusively proven. NOTE: `claude-team show-stage-def` (the dispatch Fetch command) is not on PATH in this environment (exit 127); the validation checklist was fully specified in the dispatch + entity spec, so I proceeded — flagging the missing helper for FO awareness.

## Feedback Cycles

### Cycle 1 — validation REJECTED (2026-05-30)

Validation ensign PASSED (native@docs/dev byte-identical, discovery dedup, --set/--archive state-only, gates green) — but only exercised the def-dir operator model. A parallel adversarial staff audit (own read-only checkout) found two flip-blast-radius issues; the split-root mechanism, pyJoin (26 os.path.join cases), discovery-prune, and threading were all VERIFIED-correct.

- **M1 (material) — flip + live symlink nesting.** With the production default flipped to NativeRunner, `spacedock status --workflow-dir docs/dev/.spacedock-state` returns a SILENT EMPTY TABLE (exit 0): `.spacedock-state/README.md` is a symlink whose `state: .spacedock-state` is re-applied (roots.go:55), nesting entityDir to a nonexistent `.spacedock-state/.spacedock-state`. Contradicts bootstrap-roadmap.md:48 + docs/dev/README.md compat-phase docs + the live symlink. symlink_profile_test.go gave false confidence (its fixture README omits `state:`). CAPTAIN DECISION: **defer the flip to Stage 7** — Stage 6 ships native split-root as SELECTABLE (production default stays VendorRunner); Stage 7 removes the symlink AND flips together (no nesting hazard once the symlink is gone).
- **M2 (scoped) — no worktree-copy overlay on active reads.** Native scanEntities reads only the pipeline-dir copy; the oracle (scan_entities_active / load_active_entity_fields) overlays the worktree-copy frontmatter for entities with a `worktree:` field. Correct for split-root (state lives in the checkout, never a worktree copy), but the flip would make native the global default and mis-read NON-split-root worktree workflows. CAPTAIN DECISION: **implement the worktree overlay** for full parity.

Fix scope (cycle 2): (1) revert the cli.go flip — default back to VendorRunner, native stays selectable; (2) implement the worktree-copy active-read overlay matching the oracle for non-split-root worktree-bearing entities (+ parity test: pipeline copy != worktree copy -> native shows worktree copy, matching oracle); (3) keep all native split-root code (resolveRoots, boot.go pyJoin fix, discovery dedup) + tests; (4) repoint/repin the launcher smoke test so it stays green + meaningful under the reverted default.

## Stage Report: implementation (cycle 2)

- DONE: M1 — REVERT THE FLIP. cli.Run default back to `&status.VendorRunner{}`; native stays selectable via the injectable `run()` core (all native split-root code kept). Launcher smoke test repointed to the .spacedock-state state dir with a `README.md -> ../README.md` compat symlink (VendorRunner compat path) so it stays green + meaningful under the reverted default.
  cli.go:18-20 VendorRunner. launcher_smoke_test.go: `--workflow-dir <stateDir>` + compat symlink. Confirmed the OLD smoke test failed under the revert (empty table — oracle doesn't read `state:`) before repointing; repointed test PASS. Production default verified live: `spacedock status --workflow-dir docs/dev/.spacedock-state` lists the three real entities (exit 0). Code commit ac2d39b on spacedock-ensign/native-state-dir.
- DONE: M2 — IMPLEMENT the worktree-copy active-read overlay matching the oracle (scan_entities_active / load_active_entity_fields). Active reads overlay the worktree-COPY frontmatter for entities with a non-empty `worktree:` field when that copy exists at `<git_root>/<worktree>/<rel-under-pipeline-dir>`; fall back to the pipeline-dir copy when it does not. Split-root entities (no worktree copy in main repo) still fall back to the state-checkout copy; non-split-root worktree-bearing entities now read the worktree copy.
  discover.go: added `worktreeMirrorPath` (pyJoin, matches `_worktree_mirror_path`), `loadActiveEntityFields` (matches `load_active_entity_fields`), `scanEntitiesActive` (derives gitRoot from the entity dir the way the oracle derives it from pipeline_dir). `activeAndArchivedEntities` now uses `scanEntitiesActive` for the active portion; handlers.go:227 uses `scanEntitiesActive` for the displayed active table (matching oracle main 2526-2527). Parity test `TestWorktreeOverlayActiveReads` (worktree_overlay_test.go, 4 subtests) builds a non-split-root worktree-backed entity with pipeline status=implementation != worktree status=review and asserts native table/--where/--fields/--resolve byte-match VendorRunner (the oracle) AND show the worktree-copy value. Confirmed the test FAILED before the overlay (native showed implementation, oracle review) then PASSED after.
- DONE: KEEP all cycle-1 native split-root code (resolveRoots state resolution + abs/escape rejection, boot.go pyJoin, discovery dedup, threading) and their tests intact. Gates all green with REAL exit codes (not piped through tail).
  No cycle-1 mechanism code touched (git diff is only cli.go, discover.go, handlers.go, the new test, and the smoke test). Gates: `go test ./...` exit 0; `go test ./... -race` exit 0 (no data races); `gofmt -l ./cmd ./internal ./skills` empty; `go vet ./...` exit 0. 86 top-level tests, 0 fail, 0 skip.

### Summary

Cycle-2 made exactly the two captain-directed changes plus the smoke-test repoint, touching no cycle-1 mechanism code. M1 reverted the production default to VendorRunner (native stays selectable through the unexported `run()` core) and repointed the AC-5 launcher smoke test to the `.spacedock-state` checkout with a `README -> ../README.md` compat symlink so it exercises the real binary on the VendorRunner-supported symlink-compat layout. M2 added the worktree-copy active-read overlay in `discover.go` mirroring the oracle's `scan_entities_active`/`load_active_entity_fields`: `activeAndArchivedEntities` and the displayed active table now route through `scanEntitiesActive`, which overlays the worktree-copy frontmatter for `worktree:`-bearing entities and falls back to the pipeline copy otherwise — so split-root entities are unaffected while non-split-root worktree workflows now match the oracle. The new `TestWorktreeOverlayActiveReads` byte-compares native against VendorRunner across table/--where/--fields/--resolve and was confirmed red-before/green-after. All gates green. NOTE: the dispatch Fetch command `claude-team show-stage-def` is not on this PATH (the real binary is `claude-team build/spawn-standing/...` with no `show-stage-def` subcommand; flagged for the prior validation ensign too); the assignment was fully specified in the dispatch + entity spec, so I proceeded. The cli default is verified by inspection + live binary behavior; no unit test pins `cli.Run`'s production runner (the cli test injects runners into the unexported core), unchanged from cycle 1.

## Stage Report: validation (cycle 2)

- DONE: M1 GONE — production default is VendorRunner and the state-checkout path renders (no silent empty table), byte-identical to the live oracle; native split-root @ docs/dev stays selectable and byte-identical to vendored @ .spacedock-state.
  Inspection: `cli.go` `Run()` constructs `&status.VendorRunner{}` (flip reverted). Built the binary at HEAD ac2d39b. Production-default `spacedock status --workflow-dir docs/dev/.spacedock-state` returns exit 0 and LISTS the three real entities (5 stdout lines, NOT empty), stdout+stderr byte-identical to the live oracle on that path (same sha256 87b0af37dc9aba…). Drove the exported NativeRunner directly against the REAL `docs/dev` (split-root) and byte-compared to the live oracle @ `docs/dev/.spacedock-state` across default table, `--validate`, `--next`, `--archived` — stdout, stderr, AND exit code identical in all four; the three entities (native-state-dir, remove-symlink-retest, external-tracker-checkpoint) appear, sourced from the state checkout with stages from the main README. The cycle-1 nesting hazard (state README symlink re-applies `state:` under a flipped NativeRunner) cannot occur because the default no longer routes the state path through NativeRunner.
- DONE: M2 worktree-copy overlay reproduced INDEPENDENTLY vs the oracle; split-root entities correctly FALL BACK; the overlay is load-bearing (mutation-checked).
  Built my own non-split-root git fixture (own frontmatter: pipeline status=build, worktree-copy status=done at `<git_root>/wt/feat.md`). Native ACTIVE reads — default table, `--where status=done`, `--fields status`, `--resolve` — show the WORKTREE-copy value (done) and are stdout/stderr/exit byte-identical to the live oracle's scan_entities_active; the pipeline-copy value (build) is NOT displayed in the table. MUTATION CHECK: temporarily disabled the overlay (scanEntitiesActive → plain parseFrontmatter) and confirmed BOTH my independent repro AND the implementation's `TestWorktreeOverlayActiveReads` go RED (native shows build/implementation, oracle shows done/review) — proving the overlay is load-bearing, not a no-op; restored discover.go byte-identical to HEAD afterward. SPLIT-ROOT FALLBACK: an independent split-root fixture whose state-checkout entity sets `worktree: wt` but whose worktree dir lacks the entity file shows the state-checkout status (build), not done — overlay does not misfire. Confirmed live too: the real `native-state-dir` entity carries an ABSOLUTE `worktree:`; its mirror path `<worktree>/native-state-dir/index.md` does not exist in the code worktree, so native (gitRoot derived from `.spacedock-state`, matching the oracle's find_git_root(pipeline_dir)) falls back to the state copy — which is exactly why the M1 byte-identity held. `TestWorktreeOverlayActiveReads` runs and passes (4 subtests).
- DONE: No regression — full gates green with REAL exit codes (not piped through tail).
  `go test ./...` exit 0 (245 PASS / 0 FAIL / 0 SKIP, verbose-counted); `go test ./... -race` exit 0 (no data races); `gofmt -l ./cmd ./internal ./skills` empty; `go vet ./...` exit 0. All dispatch-named tests PASS: native_state_dir (TestResolveRootsStateField set/absent + TestResolveRootsStateRejected absolute/escape; TestSplitRootStatusNoSymlink, StagesAndIdentity, SetMutatesOnlyState, ArchiveMovesOnlyState, DiscoverySingleCount both sub-cases), TestBootAbsoluteWorktreeDirExists, 14 zz Ind* parity (read flags seq/sdb32/slug, mutation, validation, usage-errors, new, eofnewline, archive-dest, realpath, crlf, score-sort, slug-no-id-stamp, folder-worktree-dispatch, contract-clause), vendor golden (TestGoldenRead) + mutation (TestMutationParity, TestNativeMutationParity), skills/integration, TestSymlinkStateProfile (pins VendorRunner), TestWorktreeOverlayActiveReads. Working tree clean after validation (temp repro test removed).
- DONE: PASSED — see recommendation below.

### Summary

PASSED. Independently reproduced both captain-directed fixes against the live oracle, not the prior agent's run. M1: the cli.Run default is VendorRunner again (inspection + binary behavior); the production-default binary on `--workflow-dir docs/dev/.spacedock-state` lists the three real entities and is byte-identical (sha256 87b0af37…) to the live oracle on that path — no silent empty table — while native split-root @ docs/dev remains selectable via the exported NativeRunner and is stdout/stderr/exit byte-identical to the vendored state-checkout path across default/--validate/--next/--archived. M2: built an independent non-split-root git fixture (pipeline status=build vs worktree-copy status=done) and confirmed native table/--where/--fields/--resolve show the worktree-copy value byte-matching the live oracle's scan_entities_active; a mutation check (overlay disabled) drove both my repro and TestWorktreeOverlayActiveReads RED, proving the overlay is load-bearing; split-root entities (worktree set, no worktree copy) correctly fall back to the state-checkout copy, verified on an independent fixture AND on the live native-state-dir entity (absolute worktree, mirror path absent → fallback, which is why M1 byte-identity held). No regression: go test ./... exit 0 (245 PASS / 0 FAIL / 0 SKIP), -race clean (no data races), gofmt -l empty, go vet exit 0. One non-blocking carryover (unchanged from cycle 1): no unit test pins `cli.Run`'s production runner — the cli test injects runners into the unexported `run()` core — but the VendorRunner default is conclusively proven by inspection plus live binary behavior (the state-checkout path renders via VendorRunner, and a flipped default would silently nest). NOTE: the dispatch Fetch command `claude-team show-stage-def` is not on this PATH (exit 127); the assignment was fully specified in the dispatch + entity spec, so I proceeded — flagging the missing helper for FO awareness.
