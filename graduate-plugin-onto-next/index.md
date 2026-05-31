---
id: n15ga31s6dbs3xxbhbz4at8s
title: Graduate the full plugin surface onto next (released-lane completion, next-as-source)
status: ideation
source: sprint — captain 2026-05-31 ("next-as-source is fine"); released-lane follow-up to fresh-install-journey (dev-lane)
started: 2026-05-31T16:07:11Z
completed:
verdict:
score: "0.30"
worktree:
issue:
---

The dev-lane self-hosting loop shipped (D: v1 vendors a MINIMAL `.claude-plugin/plugin.json` + first-officer/ensign so `--plugin-dir <repo>` loads v1's own skills). The captain confirmed **next-as-source**: the released lane (`spacedock init` / marketplace) publishes the plugin from `spacedock-dev/spacedock@next` — NO cross-repo edit to `main`. This entity makes `next` a COMPLETE publishable plugin AND closes the marketplace-resolution gap so the binary's own install path works.

## Why
Publishing from `next` today would ship an INCOMPLETE plugin AND the install path the binary issues does not even resolve. Two distinct gaps, both verified against the working tree (HEAD `de5bb44` = `next`):

1. **Incomplete skill surface.** `next` carries only `first-officer`+`ensign` SKILL.md (D's dev-lane minimal scope). It DROPS `debrief`+`refit` and ships an empty-shell `commission/` (only `bin/`, no SKILL.md). The canonical full user surface lives on `origin/main` (old 0.12.1): `git ls-tree origin/main` → `skills/{commission,debrief,ensign,first-officer,refit}/SKILL.md` + `agents/{ensign,first-officer}.md`. `next` also ships `skills/integration/` (Go test files only — `*_test.go`), which is test-only and must NOT publish.
2. **No marketplace manifest at the next root.** `next` root has `.claude-plugin/plugin.json` but NO `.claude-plugin/marketplace.json` and NO `.codex-plugin/plugin.json` (confirmed: `git ls-tree origin/next` lists only `.claude-plugin/plugin.json`). The binary's `execHost.Install` (`internal/cli/host_exec.go:200`) shells `claude plugin marketplace add spacedock-dev/spacedock@next` then `claude plugin install spacedock@spacedock`. The shorthand `owner/repo@ref` resolves the marketplace manifest at the repo ROOT for that ref — which does not exist on `next`. So `spacedock init --host claude` against `next` fails at `marketplace add` today.

## Ground truth (empirically established this ideation, working tree + installed marketplaces)

**Binary hardcodes the install id `spacedock@spacedock`.** `resolveClaudeManifest` (`host_exec.go:55`) matches `e.ID == "spacedock@spacedock"`; `Install` (`host_exec.go:211`) installs `spacedock@spacedock`. The id is `{plugin-entry-name}@{marketplace-name}`, so BOTH the marketplace `name` and its plugin-entry `name` must be `spacedock` for doctor/version-gate to resolve. Any other name → doctor finds nothing. This is the constraint that drives the recommendation below.

**`source` shapes (spike-validated, Claude Code 2.1.154):** a marketplace plugin-entry `source: {"source":"url","url":"…","ref":"next"}` installs `next` end-to-end with zero repo restructuring. `source: "."` is REJECTED. `source: "./subdir"` (relative subdirectory) is accepted but requires moving the plugin into a subdir. So url+ref is the no-restructure path.

**Staleness mechanism (the reference pattern, confirmed in the installed `superpowers-marketplace`):** ONE marketplace repo, TWO entries with DISTINCT entry names — stable `superpowers` (`source:{url}`, semver `5.1.0`, tracks tags) and `superpowers-dev` (`source:{url, ref:"dev"}`, version `0.0.2026021001` calendar/monotonic, loud description `"DEV BRANCH: YOU MUST UNINSTALL OTHER VERSIONS OF SUPERPOWERS BEFORE INSTALLING THIS"`). The dev entry's CALENDAR version is the `plugin update` comparison key — bumping it is what makes `update` re-pull a moving branch. The skill NAMESPACE follows the repo `plugin.json` name (`superpowers:`), NOT the entry name, so stable+dev both own `superpowers:` and cannot coexist installed — hence the loud warning.

## Recommended approach: (A) single self-referential `spacedock` marketplace on `next`, url+ref:next

**Decision: direction (A), not (B).** Add `.claude-plugin/marketplace.json` at the `next` repo root (self-referential — it lives in `next` and points back at `next`). Shape:

```json
{
  "name": "spacedock",
  "owner": { "name": "CL Kao" },
  "plugins": [
    {
      "name": "spacedock",
      "source": { "source": "url", "url": "https://github.com/spacedock-dev/spacedock.git", "ref": "next" },
      "description": "Turn directories of markdown files into structured workflows operated by AI agents",
      "version": "0.0.YYYYMMDDNN",
      "category": "workflow"
    }
  ]
}
```

Rationale grounded in the constraints:
- **Satisfies the hardcoded id with zero binary change.** marketplace `name:"spacedock"` + entry `name:"spacedock"` → install id `spacedock@spacedock`, which is exactly what `resolveClaudeManifest`/`Install` expect. Direction (B) (two entries `spacedock`+`spacedock-next` so old+next coexist) would REQUIRE relaxing the hardcoded id to match by plugin-name across marketplaces — added binary scope — AND still cannot deliver true coexistence, because skill namespace follows `plugin.json` name (`spacedock:` for both), exactly the superpowers limitation. (B) buys nothing the migration needs; (A) matches the captain's "next-as-source" framing and retires the old lane cleanly.
- **No repo restructuring.** url+ref:next keeps the plugin at the repo root; `source:"."` is rejected and a subdirectory layout is a needless move.
- **The old-lane collision is resolved by retirement, not coexistence.** The old `spacedock` marketplace lives on `origin/main` (`.claude-plugin/marketplace.json` name `spacedock`, entry `spacedock`, `source:"./plugins/spacedock"`, version `0.12.1`). On a user machine `claude plugin marketplace add spacedock-dev/spacedock@next` REPOINTS the existing `spacedock` marketplace to the `next` ref (this repoint was observed live in task 38's repro). One marketplace name, one plugin, swapped from old→next. This is the intended migration outcome; the legacy lane is not kept alive.

**Staleness discipline (adopt superpowers-dev's calendar version).** The marketplace entry `version` is `0.0.YYYYMMDDNN` (calendar/monotonic), NOT the binary's goreleaser-tag `Version` (`0.19.x`) and NOT the plugin.json `version` (`0.1.0-dev`) — these three versions are deliberately separate. Bumping the entry's calendar version is the ONLY thing that makes `claude plugin update` re-pull the moving `next` branch (spike: `update` is version-keyed and reported "already at latest 0.1.0-dev" without a bump; `install` no-ops when already installed). The bump must be automated so it cannot be forgotten — tie it to the CI that publishes `next` (the same workflow that builds/releases), writing today's date+seq into the root marketplace.json entry on each `next` publish. The mechanism, not just the value, is part of AC-2.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — `next` carries the full user plugin surface, nothing test-only.** End state: the `next` tree contains SKILL.md + agents + complete reference closure for the five USER skills `first-officer`, `ensign`, `commission`, `debrief`, `refit` (sourced/reconciled from the canonical `origin/main` surface), and does NOT publish `skills/integration/` (test-only Go files). Verified by: `claude --plugin-dir <next-checkout> plugin details spacedock` in an isolated `CLAUDE_CONFIG_DIR` lists exactly those five skills and both agents; a dependency-closure audit (fresh-install-journey style) shows no dangling `@reference`/`bin/` path in any shipped skill, and `integration` is absent from the published skill set.

**AC-2 — Root marketplace + both host manifests resolve and install the complete plugin, with a self-bumping staleness key.** End state: `next` root carries `.claude-plugin/marketplace.json` (name `spacedock`, entry name `spacedock`, `source:{url,ref:next}`, calendar `version`) and `.codex-plugin/plugin.json` (authoritative, `requires-contract:">=1,<2"`); `.claude-plugin/plugin.json` already carries `requires-contract:">=1,<2"` (confirmed) and is expanded to the full surface; and CI bumps the marketplace entry's calendar version on each `next` publish. Verified by: (a) in a fresh isolated `CLAUDE_CONFIG_DIR`, `claude plugin marketplace add spacedock-dev/spacedock@next && claude plugin install spacedock@spacedock` installs the complete plugin and `spacedock doctor --host claude` → `OK: binary contract 1 satisfies plugin range >=1,<2` (exit 0); (b) a CI/fixture test asserts the marketplace entry version strictly increases across two consecutive `next` publishes (the re-pull key actually moves).

**AC-3 — `spacedock init --host claude` installs/upgrades from `@next` and the released-lane gate is green with NO escape hatch.** End state: `spacedock init --host claude` (devBranch pinned to `next` via `SPACEDOCK_DEV_BRANCH`/ldflags until `next` is default) issues `marketplace add spacedock-dev/spacedock@next` then resolves the installed plugin, and a subsequent `spacedock claude` gates green WITHOUT `--skip-contract-check`/`--plugin-dir`. Verified by: a host-ops seam unit test asserts the issued argv pair (`marketplace add spacedock-dev/spacedock@next`, `install spacedock@spacedock`); and a live isolated-config smoke run shows post-init `spacedock claude -- "<noop task>"` resolves `--agent spacedock:first-officer` and gates green. NOTE: this AC assumes the install/upgrade no-op (Defect 2) is fixed by task 38 — see Coupling.

## Coupling to task 38 (init-upgrade-and-contract-remedy)

Task 38 is DOWNSTREAM of and gated on this entity's install-path decision:
- **38's remedy wording can now name a command.** With direction (A) decided, 38's empty-`requires-contract` remedy (its AC-1) names `spacedock init --host claude` (which issues `marketplace add spacedock-dev/spacedock@next` + reinstall) as the upgrade path. 38 could not name a concrete command until this decision landed.
- **38's init-fix MUST use uninstall+reinstall, not the no-op install.** Spike + 38's own repro confirm `claude plugin install spacedock@spacedock` NO-OPS when already installed, so init does not upgrade a stale 0.12.1 plugin. The proven robust re-pull is `uninstall && install`. This entity's `init` path (AC-3) and 38's Defect-2 fix (AC-2) must converge on the SAME mechanism: `execHost.Install` issues `marketplace add … && plugin uninstall spacedock@spacedock && plugin install spacedock@spacedock` (uninstall is a no-op when not yet installed, so it is safe on first run). Recommend 38 own the `Install` argv change; this entity owns the manifest/marketplace surface that the argv installs FROM.

## Test plan

- **Skill-surface closure (AC-1):** static + fixture, minutes. Extend the existing `skills/integration/plugin_manifest_test.go` / `skill_text_test.go` audit to enumerate the five user skills and assert each has a SKILL.md with valid frontmatter and no dangling reference; assert `integration` is excluded from the published manifest's skill resolution. Plus one isolated-config `claude --plugin-dir <checkout> plugin details spacedock` smoke (live, ~1 min).
- **Manifest + install resolution (AC-2):** live isolated-`CLAUDE_CONFIG_DIR` smoke (~2-3 min) — the marketplace-add + install + `spacedock doctor` exit-0 sequence (the spike already proved the mechanism; this is the regression-locking run). Calendar-version monotonicity is a cheap unit/fixture test over the CI bump step (no network).
- **init/gate green (AC-3):** host-ops seam unit test for the argv pair (no network, sub-second) is the primary proof; one live post-init `spacedock claude` smoke confirms the agent resolves and the gate is green without escape hatches. Live smoke is gated on task 38's `Install` argv fix (uninstall+reinstall) to actually upgrade a stale install.
- **Cost/complexity:** the risky, must-validate-first path is AC-2's marketplace-resolution (the self-referential url+ref root manifest must resolve via the `owner/repo@ref` shorthand) — do that smallest live install FIRST, before porting all skill text, since a resolution failure invalidates the packaging direction. Skill-surface porting is mechanical once resolution is proven. Estimated: a few hours of work, dominated by reconciling/porting the three missing skills' reference closure from `origin/main`.

## Folded cleanups
- The stale `release.yml` comment (`.github/workflows/release.yml:38` — "so the brews block can push the formula bump") → it is now a `homebrew_casks` block pushing a cask. jf-audit Polish, fix here.

## Notes
- Released-lane companion to fresh-install-journey (dev-lane). No cross-repo `requires-contract` edit (next-as-source). Eventual graduation: `next` → default branch lets the marketplace entry drop the `ref:next` pin (or move to a stable semver entry tracking tags, superpowers-style) and retires the legacy Python `main` lane.
- Three independent version axes, kept separate by design: binary `Version` (goreleaser tag, `0.19.x`), plugin.json `version` (`0.1.0-dev`), marketplace-entry `version` (calendar `0.0.YYYYMMDDNN`, the `plugin update` re-pull key).

## Stage Report: ideation

- DONE: Approach picks the marketplace packaging mechanism (url+ref:next vs subdirectory layout) AND resolves the spacedock@spacedock install-id collision + old(0.12.1)/next coexistence, grounded in the VALIDATED spike facts and the superpowers reference pattern below — not hand-waving.
  Recommended direction (A): single self-referential `spacedock` marketplace at next root, plugin entry `source:{url,ref:next}`. Grounded in `host_exec.go:55,211` (hardcoded `spacedock@spacedock` id = `{entry}@{marketplace}`), origin/main's old `spacedock`/`./plugins/spacedock` marketplace, and the installed superpowers-marketplace two-entry pattern (`superpowers-dev` `ref:dev` + calendar `0.0.2026021001`). Rejected (B) with reasons: needs binary id-relaxation + still can't coexist (namespace follows plugin.json name).
- DONE: Acceptance criteria are entity-level end-state properties, each with a concrete, reproducible test method.
  AC-1/2/3 rewritten as end-states: full five-skill surface + integration excluded (closure audit + `plugin details` smoke); root marketplace.json + .codex-plugin/plugin.json resolve + calendar-version monotonicity (isolated-config install→`doctor` exit 0 + CI bump test); init issues `marketplace add …@next` + green gate w/o escape hatch (host-ops argv unit test + live smoke).
- DONE: Test plan addresses the version-staleness train AND names the coupling to task 38.
  Staleness: calendar/monotonic marketplace-entry version is the `plugin update` re-pull key, CI-bumped per next publish, kept separate from binary/plugin.json versions; monotonicity is a fixture test. Coupling section ties 38's remedy wording (now names `spacedock init`) and mandates 38's `Install` use uninstall+reinstall (not no-op install), converging both on the same argv.

### Summary

Established full ground truth from the working tree: next currently lacks a root marketplace.json (so the binary's own `marketplace add spacedock-dev/spacedock@next` fails today), drops debrief/refit, ships an empty commission shell, and carries test-only integration. Recommended direction (A) — a single self-referential `spacedock` marketplace using url+ref:next — because it satisfies the hardcoded `spacedock@spacedock` id with zero binary change, needs no repo restructuring, and matches next-as-source by retiring (not coexisting with) the old lane; (B) was rejected as it requires binary id-relaxation yet still cannot deliver namespace coexistence. Adopted superpowers-dev's calendar-version discipline as the CI-automated `plugin update` re-pull key and documented the bidirectional coupling to task 38 (uninstall+reinstall mechanism + remedy wording).
