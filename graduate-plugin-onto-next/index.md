---
id: n15ga31s6dbs3xxbhbz4at8s
title: Graduate the full plugin surface onto next (released-lane completion, next-as-source)
status: implementation
source: sprint — captain 2026-05-31 ("next-as-source is fine"); released-lane follow-up to fresh-install-journey (dev-lane)
started: 2026-05-31T16:07:11Z
completed:
verdict:
score: "0.30"
worktree: .worktrees/spacedock-ensign-graduate-plugin-onto-next
issue:
---

The dev-lane self-hosting loop shipped (D: v1 vendors a MINIMAL `.claude-plugin/plugin.json` + first-officer/ensign so `--plugin-dir <repo>` loads v1's own skills). The captain confirmed **next-as-source**: the released lane (`spacedock init` / marketplace) publishes the plugin from `spacedock-dev/spacedock@next` — NO cross-repo edit to `main`. This entity makes `next` a COMPLETE publishable plugin AND closes the marketplace-resolution gap so the binary's own install path works.

## Why
Publishing from `next` today would ship an INCOMPLETE plugin AND the install path the binary issues does not even resolve. Two distinct gaps, both verified against the working tree (HEAD `de5bb44` = `next`):

1. **Incomplete skill surface.** `next` carries only `first-officer`+`ensign` SKILL.md (D's dev-lane minimal scope). It DROPS `debrief`+`refit` and ships an empty-shell `commission/` (only `bin/`, no SKILL.md). The canonical full user surface lives on `origin/main` (old 0.12.1): `git ls-tree origin/main` → `skills/{commission,debrief,ensign,first-officer,refit}/SKILL.md` + `agents/{ensign,first-officer}.md`. `next` also ships `skills/integration/` (Go test files only — `*_test.go`), which is test-only and must NOT publish.
2. **No marketplace manifest at the next root.** `next` root has `.claude-plugin/plugin.json` but NO `.claude-plugin/marketplace.json` and NO `.codex-plugin/plugin.json` (confirmed: `git ls-tree origin/next` lists only `.claude-plugin/plugin.json`). The binary's `execHost.Install` (`internal/cli/host_exec.go:200`) shells `claude plugin marketplace add spacedock-dev/spacedock@next` then `claude plugin install spacedock@spacedock`. The shorthand `owner/repo@ref` resolves the marketplace manifest at the repo ROOT for that ref — which does not exist on `next`. So `spacedock init --host claude` against `next` fails at `marketplace add` today.

## Ground truth (empirically established this ideation, working tree + installed marketplaces)

**Binary hardcodes the install id `spacedock@spacedock`.** `resolveClaudeManifest` (`host_exec.go:55`) matches `e.ID == "spacedock@spacedock"`; `Install` (`host_exec.go:211`) installs `spacedock@spacedock`. The id is `{plugin-entry-name}@{marketplace-name}`, so BOTH the marketplace `name` and its plugin-entry `name` must be `spacedock` for doctor/version-gate to resolve. Any other name → doctor resolves nothing and renders the non-fatal no-plugin-found report (exit 0, per `runDoctor`'s empty-path handling at `init.go:89-91`), NOT a hard failure. This is the constraint that drives the recommendation below.

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
- **The old-lane collision is resolved by retirement, not coexistence.** The old `spacedock` marketplace lives on `origin/main` (`.claude-plugin/marketplace.json` name `spacedock`, entry `spacedock`, `source:"./plugins/spacedock"`, version `0.12.1`). Once `next` carries a valid root marketplace.json, `claude plugin marketplace add spacedock-dev/spacedock@next` REPOINTS the existing `spacedock` marketplace cache to the `next` ref — proven live by the FO's `spike-next-mp` spike: with a candidate root marketplace.json on the target ref, both fresh-add (isolated config) and repoint-over-existing-old-marketplace succeed and the marketplace cache checks out the new ref. CORRECTION to the prior draft: task 38's earlier `@next` add FAILED precisely BECAUSE `next` had no root marketplace.json — the add left the marketplace pinned to `main`. That failure is the bug THIS entity fixes by adding the manifest; it is NOT evidence that repoint works against a manifest-less ref. One marketplace name, one plugin, swapped from old→next. This is the intended migration outcome; the legacy lane is not kept alive.

**Three version axes, two of them release-driven.** The system has three distinct version fields, and the prior draft was wrong to treat plugin.json's version as a frozen placeholder:
1. **Binary `Version`** — `internal/cli/cli.go:23`, goreleaser-stamped via `-ldflags -X …cli.Version={{.Version}}` (`.goreleaser.yaml:30`), currently `0.19.0`.
2. **plugin.json `version`** — currently frozen at `0.1.0-dev` because nothing stamps it. THIS IS USER-VISIBLE: the spike confirmed `claude plugin list --json` and the `--plugin-dir` plugin panel display plugin.json's `version`, NOT the marketplace entry's calendar version. So binary `0.19.0` vs displayed `0.1.0-dev` is a visible divergence — addressed by AC-4 (release pipeline must stamp plugin.json `version`, and the new `.codex-plugin/plugin.json` `version`, to the release version so binary ↔ display align).
3. **Marketplace-entry `version`** — `0.0.YYYYMMDDNN` (calendar/monotonic), the `claude plugin update` re-pull key. Bumping it is the ONLY thing that makes `update` re-pull the moving `next` branch (spike: `update` is version-keyed and reported "already at latest 0.1.0-dev" without a bump; `install` no-ops when already installed). Kept DISTINCT from plugin.json's version: plugin.json tracks the release; the entry version is the moving-branch cache-buster.

**Staleness discipline (adopt superpowers-dev's calendar version).** The marketplace-entry calendar bump must be automated so it cannot be forgotten — tie it to the CI that publishes `next` (the same workflow that builds/releases), writing today's date+seq into the root marketplace.json entry on each `next` publish. The mechanism, not just the value, is part of AC-2.

## Acceptance criteria (provisional — ideation hardens)

**Host scope.** This entity targets BOTH hosts: the `.codex-plugin/plugin.json` manifest and a real Codex smoke are in scope (AC-2c). Codex uses a different verb shape — `codex plugin marketplace add owner/repo --ref next` then `codex plugin add spacedock@spacedock` — and `resolveCodexManifest` (`host_exec.go:72`) resolves the manifest from the Codex cache, which requires `.codex-plugin/plugin.json` to exist. Codex is NOT scoped out.

**AC-1 — `next` carries the full user plugin surface, nothing test-only.** End state: the `next` tree contains SKILL.md + agents + complete reference closure for the five USER skills `first-officer`, `ensign`, `commission`, `debrief`, `refit` (sourced/reconciled from the canonical `origin/main` surface), and does NOT publish `skills/integration/` (test-only Go files). The exclusion mechanism is explicit: `skills/integration/` holds only `*_test.go` (no `SKILL.md`), so the host's skill discovery (which keys on `SKILL.md`) already omits it from the published skill set — no manifest allow/deny-list needed; the audit VERIFIES this rather than presuming it. Verified by: `claude --plugin-dir <next-checkout> plugin details spacedock` in an isolated `CLAUDE_CONFIG_DIR` lists EXACTLY those five skills and both agents AND asserts `integration` is absent from that listing; a dependency-closure audit (fresh-install-journey style) shows no dangling `@reference`/`bin/` path in any shipped skill.

**AC-2 — Root marketplace + both host manifests resolve and install the complete plugin, with a self-bumping staleness key.** End state: `next` root carries `.claude-plugin/marketplace.json` (name `spacedock`, entry name `spacedock`, `source:{url,ref:next}`, calendar `version`) and `.codex-plugin/plugin.json` (authoritative, `requires-contract:">=1,<2"`); `.claude-plugin/plugin.json` already carries `requires-contract:">=1,<2"` (confirmed) and is expanded to the full surface; and CI bumps the marketplace entry's calendar version on each `next` publish. Verified by:
- **(a) Claude fresh-add** — in a fresh isolated `CLAUDE_CONFIG_DIR` with NO prior spacedock marketplace, `claude plugin marketplace add spacedock-dev/spacedock@next && claude plugin install spacedock@spacedock` installs the complete plugin and `spacedock doctor --host claude` → `OK: binary contract 1 satisfies plugin range >=1,<2` (exit 0).
- **(b) Claude repoint-over-old** — in an isolated config seeded with the OLD `spacedock` marketplace (origin/main, entry pinned to `0.12.1`), `claude plugin marketplace add spacedock-dev/spacedock@next` repoints it: assert the marketplace cache's checked-out ref moved to `next` AND (after the upgrade reinstall, see AC-3b) the installed plugin is off `0.12.1`. This is the riskier path and runs AFTER (a) — see Test plan ordering.
- **(c) Codex fresh-add** — in a fresh isolated `CODEX_HOME`, `codex plugin marketplace add spacedock-dev/spacedock --ref next && codex plugin add spacedock@spacedock` installs the plugin and `spacedock doctor --host codex` → OK (exit 0); the resolved manifest is the cached `.codex-plugin/plugin.json`.
- **(d) Staleness key moves** — a unit/fixture test invokes the CI bump FUNCTION twice over the marketplace.json and asserts the entry `version` strictly increases (the re-pull key actually moves) — not two hand-written literals.

**AC-3a — `spacedock init --host claude` issues the correct install argv against `@next` and a fresh install gates green (verifiable in THIS entity).** End state: `spacedock init --host claude` (devBranch pinned to `next` via `SPACEDOCK_DEV_BRANCH`/ldflags until `next` is default) issues `marketplace add spacedock-dev/spacedock@next` + the install of `spacedock@spacedock`, and on a machine with NO prior spacedock install a subsequent `spacedock claude` gates green WITHOUT `--skip-contract-check`/`--plugin-dir`. Verified by: a host-ops seam unit test asserts the issued argv matches the spec (the exact argv shape is owned by task 38 — see AC-3b/Coupling — so this test asserts argv-from-current-spec, and is updated in lockstep when 38 lands); plus a live fresh-install isolated-config smoke showing post-init `spacedock claude -- "<noop task>"` resolves `--agent spacedock:first-officer` and gates green.

**AC-3b — `spacedock init` upgrades an already-installed STALE plugin to green (verifiable only AFTER task 38 lands).** End state: on a machine with the stale `0.12.1` plugin installed, `spacedock init --host claude` leaves `doctor` reporting compatible (the install moves off `0.12.1`). The proven mechanism is uninstall+reinstall: `claude plugin uninstall spacedock@spacedock && claude plugin install spacedock@spacedock` (plain `install` no-ops on an existing install — Defect 2). This AC is GATED on task 38 shipping the 3-command `Install` argv (`marketplace add … && uninstall … && install …`); it is NOT verifiable by this entity alone. Verified by: after 38 lands, a live upgrade-from-stale smoke (seed `0.12.1`, run init, assert `doctor` OK exit 0) plus the host-ops argv unit test updated to the 3-command shape.

**AC-4 — The displayed plugin version tracks the release, not a placeholder.** End state: the release pipeline stamps `.claude-plugin/plugin.json` `version` (and the new `.codex-plugin/plugin.json` `version`) to the release version (the same value goreleaser stamps into binary `Version`), so `claude plugin list --json` / the `--plugin-dir` plugin panel display a version that matches the binary instead of the frozen `0.1.0-dev`. This is DISTINCT from the marketplace-entry calendar version (AC-2d), which remains the moving-branch re-pull key. Verified by: a fixture/CI test asserting that after the stamp step both plugin.json files carry the release version (not `0.1.0-dev`), and that the marketplace-entry calendar version is left untouched by that same step.

## Sequencing honesty (what THIS entity does and does not fix)

Shipping THIS entity alone fixes the FRESH-install and FRESH-add lanes (AC-2a/2c, AC-3a) — a new user gets the complete plugin and a green gate. It does NOT fix any ALREADY-INSTALLED user: the upgrade-from-stale path (AC-2b's "off 0.12.1", AC-3b) only works once task 38 ships the uninstall+reinstall `Install` argv, because plain `install` no-ops on an existing install. Dependency direction: task 38 is the unblock for the upgrade lane and is DOWNSTREAM of this entity's packaging decision (38 needed the install path decided before it could name the remedy command and write the argv). So: this entity lands the manifests/marketplace; 38 lands the argv that makes those manifests reach stale machines.

## Coupling to task 38 (init-upgrade-and-contract-remedy)

Task 38 is DOWNSTREAM of and gated on this entity's install-path decision:
- **38's remedy wording can now name a command.** With direction (A) decided, 38's empty-`requires-contract` remedy (its AC-1) names `spacedock init --host claude` (which issues `marketplace add spacedock-dev/spacedock@next` + reinstall) as the upgrade path. 38 could not name a concrete command until this decision landed.
- **38's init-fix MUST use uninstall+reinstall, not the no-op install.** Spike + 38's own repro confirm `claude plugin install spacedock@spacedock` NO-OPS when already installed, so init does not upgrade a stale 0.12.1 plugin. The proven robust re-pull is `uninstall && install`. This entity's upgrade ACs (AC-2b "off 0.12.1", AC-3b) and 38's Defect-2 fix (its AC-2) must converge on the SAME mechanism: `execHost.Install` issues `marketplace add … && plugin uninstall spacedock@spacedock && plugin install spacedock@spacedock` (uninstall is a no-op when not yet installed, so it is safe on first run). 38 OWNS the `Install` argv change (3-command shape); this entity owns the manifest/marketplace surface the argv installs FROM, and its AC-3a argv unit test is updated in lockstep when 38 lands.

## Test plan

- **Skill-surface closure (AC-1):** static + fixture, minutes. Extend the existing `skills/integration/plugin_manifest_test.go` / `skill_text_test.go` audit to enumerate the five user skills and assert each has a SKILL.md with valid frontmatter and no dangling reference; assert `integration` is excluded from the published manifest's skill resolution. Plus one isolated-config `claude --plugin-dir <checkout> plugin details spacedock` smoke (live, ~1 min).
- **Manifest + install resolution (AC-2):** live isolated-host smoke (~3-5 min total). ORDER MATTERS: run (a) Claude fresh-add (clean config, lower risk) FIRST, then (b) repoint-over-old (seed old `0.12.1` marketplace — the riskier path), then (c) Codex fresh-add (isolated `CODEX_HOME`). The FO's `spike-next-mp` already proved fresh-add + repoint mechanically; these are the regression-locking runs against the real committed manifests. (d) Calendar-version monotonicity is a cheap unit/fixture test that invokes the CI bump FUNCTION twice and asserts strict increase (no network, no literals).
- **init/gate green (AC-3a):** host-ops seam unit test for the issued argv (no network, sub-second) is the primary proof; one live fresh-install post-init `spacedock claude` smoke confirms the agent resolves and the gate is green without escape hatches.
- **Upgrade-from-stale (AC-3b):** GATED on task 38 — live upgrade smoke (seed `0.12.1`, run init, assert `doctor` OK) + argv unit test at the 3-command shape. Not run in this entity's window; tracked as the cross-entity verification once 38 lands.
- **Version-stamp (AC-4):** fixture/CI test (no network) asserting the stamp step writes the release version into both plugin.json files and leaves the marketplace-entry calendar version untouched.
- **Cost/complexity:** the risky, must-validate-first path is AC-2's marketplace-resolution against the REAL committed root manifest (the self-referential url+ref entry must resolve via the `owner/repo@ref` shorthand, and Codex's `--ref` shape must resolve the cached `.codex-plugin/plugin.json`) — do the smallest live fresh-add FIRST, before porting all skill text, since a resolution failure invalidates the packaging direction. The spike de-risked the mechanism; this confirms the committed artifacts. Skill-surface porting is mechanical once resolution is proven. Estimated: a few hours, dominated by reconciling/porting the three missing skills' reference closure from `origin/main` and wiring the two CI version steps (release-stamp + calendar-bump).

## Folded cleanups
- The stale `release.yml` comment (`.github/workflows/release.yml:38` — "so the brews block can push the formula bump") → it is now a `homebrew_casks` block pushing a cask. jf-audit Polish, fix here.

## Notes
- Released-lane companion to fresh-install-journey (dev-lane). No cross-repo `requires-contract` edit (next-as-source). Eventual graduation: `next` → default branch lets the marketplace entry drop the `ref:next` pin (or move to a stable semver entry tracking tags, superpowers-style) and retires the legacy Python `main` lane.
- Three version axes: binary `Version` (goreleaser tag) and plugin.json/.codex-plugin.json `version` are ALIGNED by the release-stamp step (AC-4 — plugin.json tracks the release, it is NOT a frozen placeholder); the marketplace-entry `version` (calendar `0.0.YYYYMMDDNN`) stays SEPARATE as the `plugin update` re-pull key.

## Stage Report: ideation

- DONE: Approach picks the marketplace packaging mechanism (url+ref:next vs subdirectory layout) AND resolves the spacedock@spacedock install-id collision + old(0.12.1)/next coexistence, grounded in the VALIDATED spike facts and the superpowers reference pattern below — not hand-waving.
  Recommended direction (A): single self-referential `spacedock` marketplace at next root, plugin entry `source:{url,ref:next}`. Grounded in `host_exec.go:55,211` (hardcoded `spacedock@spacedock` id = `{entry}@{marketplace}`), origin/main's old `spacedock`/`./plugins/spacedock` marketplace, and the installed superpowers-marketplace two-entry pattern (`superpowers-dev` `ref:dev` + calendar `0.0.2026021001`). Rejected (B) with reasons: needs binary id-relaxation + still can't coexist (namespace follows plugin.json name).
- DONE: Acceptance criteria are entity-level end-state properties, each with a concrete, reproducible test method.
  AC-1/2/3 rewritten as end-states: full five-skill surface + integration excluded (closure audit + `plugin details` smoke); root marketplace.json + .codex-plugin/plugin.json resolve + calendar-version monotonicity (isolated-config install→`doctor` exit 0 + CI bump test); init issues `marketplace add …@next` + green gate w/o escape hatch (host-ops argv unit test + live smoke).
- DONE: Test plan addresses the version-staleness train AND names the coupling to task 38.
  Staleness: calendar/monotonic marketplace-entry version is the `plugin update` re-pull key, CI-bumped per next publish, kept separate from binary/plugin.json versions; monotonicity is a fixture test. Coupling section ties 38's remedy wording (now names `spacedock init`) and mandates 38's `Install` use uninstall+reinstall (not no-op install), converging both on the same argv.

### Summary

Established full ground truth from the working tree: next currently lacks a root marketplace.json (so the binary's own `marketplace add spacedock-dev/spacedock@next` fails today), drops debrief/refit, ships an empty commission shell, and carries test-only integration. Recommended direction (A) — a single self-referential `spacedock` marketplace using url+ref:next — because it satisfies the hardcoded `spacedock@spacedock` id with zero binary change, needs no repo restructuring, and matches next-as-source by retiring (not coexisting with) the old lane; (B) was rejected as it requires binary id-relaxation yet still cannot deliver namespace coexistence. Adopted superpowers-dev's calendar-version discipline as the CI-automated `plugin update` re-pull key and documented the bidirectional coupling to task 38 (uninstall+reinstall mechanism + remedy wording).

## Stage Report: ideation (cycle 2 — revision round 1)

- DONE: M2 — corrected the over-stated repoint claim
  Rewrote the "repoints existing spacedock marketplace (observed in task 38)" line: repoint works ONLY against a ref carrying a valid root marketplace.json (proven by FO's spike-next-mp); task 38's earlier @next add FAILED because next had none. AC-2 now verifies (a) fresh-add AND (b) repoint-over-old (cache ref moved + plugin off 0.12.1).
- DONE: M3 — Codex decision stated
  Added a "Host scope" line: BOTH hosts in scope. New AC-2c is a real Codex smoke (`codex plugin marketplace add … --ref next` + `plugin add`, resolving cached .codex-plugin/plugin.json). Codex NOT scoped out.
- DONE: M4 — split AC-3
  AC-3a (verifiable HERE: argv-from-spec + fresh-install green gate) / AC-3b (verifiable only AFTER task 38: upgrade-from-stale via uninstall+reinstall). Argv unit test noted as updated in lockstep when 38 lands.
- DONE: M5 — sequencing honesty
  Added "Sequencing honesty" section: this entity fixes fresh-install/fresh-add only; no already-installed user is fixed until 38's uninstall+reinstall argv ships. Dependency direction stated (38 downstream of this packaging decision, unblock for the upgrade lane).
- DONE: VERSION-ALIGNMENT — added AC-4
  Release pipeline stamps plugin.json + .codex-plugin/plugin.json `version` to the release version (binary↔display align), kept DISTINCT from the marketplace-entry calendar key. Documented the displayed-version fact (spike: plugin list shows plugin.json version, not calendar). Three-axes Note corrected (plugin.json tracks release, not placeholder).
- DONE: P2/P3/P4/P5 polish
  P2: monotonicity test invokes the bump FUNCTION twice (AC-2d). P3: AC-2 live order fresh-add→repoint→codex (test plan). P4: "doctor resolves nothing" reframed as non-fatal no-plugin-found report exit 0 (init.go:89-91). P5: integration exclusion mechanism pinned (no SKILL.md → host discovery omits it) and the smoke VERIFIES absence.

### Summary

Hardening + one fact-correction per the independent staff review and captain, direction (A) unchanged. The key correction: repoint is NOT unconditional — task 38's prior @next failure was the manifest-less-ref bug this entity fixes, now backed by the FO's spike-next-mp live proof. ACs expanded to cover both hosts, split into here-verifiable (3a) vs 38-gated (3b), and a new AC-4 aligns the user-visible plugin.json version with the release while keeping the marketplace calendar key separate as the update re-pull mechanism.

## Stage Report: implementation

- DONE: Fresh-add lanes are actually installable against the REAL committed artifacts (AC-2a; AC-2c codex analog).
  Committed root `.claude-plugin/marketplace.json` (name spacedock, entry spacedock, source {url,ref:next}, calendar `0.0.2026053101`) + `.codex-plugin/plugin.json` (requires-contract >=1,<2) on branch spacedock-ensign/graduate-plugin-onto-next (51ffb1e). LIVE proof: fresh isolated CLAUDE_CONFIG_DIR → marketplace add → `claude plugin install spacedock@spacedock` (installed from a local bare repo carrying my committed tree as `next`, via the exact url+ref:next git source the manifest uses) → `claude plugin list --json` shows spacedock@spacedock → `spacedock doctor --host claude` OK exit 0. Both committed manifests doctor-resolve compatible via `--plugin-manifest` (claude + codex, exit 0). Codex live install of the same manifest succeeded (`codex plugin add`, cache carries the new `.codex-plugin/plugin.json`).
- DONE: Full user skill surface present and clean: commission/debrief/refit ported and RECONCILED.
  All `{spacedock_plugin_dir}`, `skills/commission/bin/status` command refs, `.agents/plugins/marketplace.json`, and `claude --agent spacedock:first-officer` reconciled to `spacedock status` / `spacedock claude` / `{project_root}/mods|agents`. Carried `mods/pr-merge.md` (commission/refit closure). LIVE: `claude --plugin-dir <checkout> plugin details spacedock` → `Skills (5) commission, debrief, ensign, first-officer, refit` + 2 agents, integration ABSENT. New audit tests (skill_surface_test.go) enumerate the five, assert reference-closure resolves, assert integration is SKILL.md-free, and ban plugin-private paths across all shipped .md. `go test ./...` + `-race` green; gofmt clean.
- DONE: Version alignment wired (AC-4 + M-new).
  internal/release (StampVersion + BumpCalendarVersion) + cmd/spacedock-release; release.yml stamps both plugin.json `version` to the release and commits to `next`; next-publish.yml bumps the marketplace calendar key; marketplace-entry calendar key untouched by the stamp (asserted). M-new LIVE: stamped plugin.json to 0.19.0, published on a bare `next`, reinstalled → `claude plugin list --json` DISPLAYS `0.19.0` (was 0.1.0-dev) — binary↔display divergence proven closed end-to-end, not just file-write.
- SKIPPED: AC-2b repoint-over-old + AC-3b upgrade-from-stale.
  38-gated per dispatch SCOPE: did NOT touch execHost.Install's 2-command argv (task 38 owns the uninstall+reinstall 3-command shape). AC-3a host-ops argv test asserts the CURRENT 2-command shape; init @next targeting wired (devBranch default next + goreleaser ldflag + SPACEDOCK_DEV_BRANCH override) with a seam test asserting init issues @next.

### Summary

Made `next` a complete, self-installable plugin: root marketplace.json (self-referential url+ref:next) + .codex-plugin/plugin.json, three ported+reconciled user skills (commission/debrief/refit) on the Go-binary command surface, carried mods/pr-merge.md, and release-pipeline version steps (AC-4 plugin.json stamp committed to next + AC-2d calendar bump). Wired devBranch=next so `spacedock init` targets `spacedock-dev/spacedock@next`. Fresh-add (AC-2a/2c) and the M-new displayed-version fix proven LIVE against the committed artifacts; AC-2b/3b left to task 38 per scope. Two findings for FO/captain: (1) `bin/status` STAYS — it is a python LIBRARY imported by `claude-team` (removing it broke dispatch tests; restored), not a dead command; (2) `execHost.codexEntryInstalled` greps `<id> (installed` (codex 0.132.0 paren form) but codex 0.135.0 renders a table without the paren, so `doctor --host codex` resolves nothing live — pre-existing resolver/codex-version drift, same root as the pre-existing TestCodexResolveManifestAgainstInstalledHost env failure, out of this entity's scope.
