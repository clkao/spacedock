---
id: tq66yjc7sqbhyc52eg8h2ecx
title: Spacedock packaging and distribution
status: ideation
source: handoff self-hosting gap
score: "0.35"
worktree:
started: 2026-05-30T19:18:28Z
---

Define and build the v1 distribution model so a fresh Claude Code or Codex session runs the Spacedock first officer from the repo's own native binary + a version-gated plugin — no Python in the dispatch path, and no contract files copied into per-agent skill folders.

Model (decided with the captain):
- The `spacedock` binary is the canonical artifact: `status` + `dispatch` (native, via `native-dispatch-helper`) + host front-ends `spacedock claude` / `spacedock codex` + `init` / `doctor`. It exposes a CONTRACT-VERSION in `--version`.
- Per-host PLUGINS (Claude Code AND Codex) register the stable agents `spacedock:first-officer` / `spacedock:ensign` (named `--agent`/subagent_type entry points) whose contracts call `spacedock status` / `spacedock dispatch`. The plugin IS the amended skill surface (already vendored under `skills/`, zero plugin-private-path refs), published per host. `spacedock init` USES the host plugin mechanism (installs the plugin) — it does NOT copy contract files into `~/.claude/skills`.
- `spacedock claude` wraps `claude --agent spacedock:first-officer …`; `spacedock codex` the Codex analog. One front door, host-native underneath.
- VERSION GATE on a CONTRACT-VERSION axis (not raw semver — bump only when the binary's flag surface / observable behavior the skill depends on changes). The plugin manifest declares `requires-contract: ">=N,<N+1"`; checked at three points: `spacedock claude` (front door, fail fast), `spacedock init`/`doctor` (install/upgrade), and FO Startup step 0 (per-session safety net). Mismatch -> actionable upgrade message. (Mirrors roborev's daemon/cli version-mismatch detection.)

## Design

### Contract-version axis

The compatibility axis is a single monotonic integer, the CONTRACT-VERSION, distinct from the plugin's display `version` (semver) and from the binary's build/release version. It names the binary's *observable contract that the skill surface depends on*: the set of `spacedock` subcommands, flags, and output sections (`--discover`, `--boot`, `--set`, `--archive`, `--resolve`, and the future `dispatch` surface) plus the parse shapes those produce. A release that only changes internals, fixes a bug without altering the flag surface, or adds a new flag the skill does not yet require does NOT bump the contract. A release that removes/renames a flag the skill calls, changes a `--boot` output section the FO parses, or changes dispatch-spec emission DOES bump it.

- The binary holds two constants: `CONTRACT_VERSION` (integer, e.g. `1`) and the existing semver build version. `--version` prints both, e.g. `spacedock 0.2.0 (contract 1)`.
- The plugin manifest declares the contract range it was authored against as `requires-contract`. It is expressed as a half-open integer range string `">=N,<M"` so a plugin can declare forward-tolerance (`">=1,<3"` accepts binaries at contract 1 or 2). The common case is `">=N,<N+1"` (exact-contract pin).
- A binary at contract `C` is COMPATIBLE with a plugin declaring `">=lo,<hi"` iff `lo <= C < hi`. Outside that interval there are two distinguishable failures:
  - **too-old-binary** — `C < lo`. The installed `spacedock` predates the contract this plugin needs. Remedy: upgrade/rebuild the binary.
  - **too-old-plugin** — `C >= hi`. The installed plugin predates the binary's contract. Remedy: update/reinstall the plugin (`spacedock init` or the host's plugin update command).

Storing the axis as an integer (not semver) keeps the comparison a total order with no pre-release/build-metadata ambiguity, and makes the bump decision a deliberate human act rather than a side effect of routine version bumps.

### Where the plugin's declared range is read

The gate needs the *installed plugin's* `requires-contract` at runtime. The binary discovers it from the installed plugin manifest:
- Claude Code: the installed plugin's `plugin.json` (resolved from the host plugin cache; `spacedock doctor` may also accept an explicit `--plugin-manifest PATH` for tests/fixtures).
- Codex: the installed `.codex-plugin/plugin.json` (analogous resolution).
- `spacedock doctor` reads the manifest, parses `requires-contract`, compares against the binary's `CONTRACT_VERSION`, and prints OK or the actionable mismatch class. When the manifest cannot be found, `doctor` reports "no installed Spacedock plugin found" rather than asserting compatibility (a distinct, non-fatal-by-default state — see test plan).

`requires-contract` carrying the contract range in the manifest (rather than the binary carrying a min-plugin-version) keeps both directions checkable from data both sides already have: the binary always knows its own `C`; the manifest always travels with the plugin.

### Three-point version gate

The same comparison runs at three points; each point differs only in *what it does on mismatch*.

1. **`spacedock claude` / `spacedock codex` front door (fail fast).** Before exec-ing the host (`claude --agent spacedock:first-officer …` / Codex analog), run the comparison. On mismatch, print the actionable message to stderr and exit non-zero WITHOUT launching the host — the operator never reaches a session that would fail at startup. On compatible, exec the host. (AC-3)
2. **`spacedock init` / `spacedock doctor` (install/upgrade time).** `init` installs/updates the host plugin via the host mechanism, then runs `doctor`. `doctor` is the standalone compatibility report (AC-1). This is where a user lands after upgrading either side; the message names which side to move.
3. **FO Startup step 0 (per-session safety net).** The vendored FO contract's Startup begins with a contract check: run `spacedock --version`, parse the `contract N` token, and confirm it satisfies the contract range the FO contract was authored against (the FO contract embeds its own expected range as a literal, so the safety net does not depend on the plugin manifest being readable from inside the agent). On mismatch, abort startup with the actionable message before any discovery/dispatch. This catches the case where the binary on PATH at session time differs from the one present at install time. (AC-2)

The actionable message is identical in shape across all three points (only the leading context line differs):

```
Spacedock contract mismatch: binary is contract <C>, plugin requires <range>.
  <too-old-binary | too-old-plugin> — <one-line remedy with the exact command>.
Run `spacedock doctor` for details.
```

### Front door and init/doctor surface

New `spacedock` subcommands (built once `native-dispatch-helper` lands the `dispatch` surface they sit beside):
- `spacedock claude [args…]` — version-gate, then `exec claude --agent spacedock:first-officer` with passthrough args.
- `spacedock codex [args…]` — version-gate, then the Codex front-officer launch analog (documented; implemented where the Codex agent-launch surface allows it — see OPEN-resolution below).
- `spacedock init [--host claude|codex] [--check]` — install/update the per-host plugin via the host's plugin mechanism, then run `doctor`. `--check` runs the compatibility report without installing. No writes into `~/.claude/skills` outside the host's own plugin install. (AC-4)
- `spacedock doctor [--plugin-manifest PATH]` — print the compatibility report (OK / too-old-binary / too-old-plugin / no-plugin-found). (AC-1)

The plugin published per host IS the amended skill surface already vendored under `skills/` (FO/ensign references calling `spacedock status` / `spacedock dispatch`, zero `skills/commission/bin/*` refs once `native-dispatch-helper` lands). `spacedock init` installing the plugin — not copying skill files — is what makes `Skill()` / `--agent spacedock:first-officer` resolve. (AC-5)

## Acceptance criteria

**AC-1 - `spacedock --version` reports a contract-version and `spacedock doctor` reports skill<->binary compatibility, distinguishing every mismatch class.**
Verified by: `spacedock --version` output contains a `contract <N>` token (Go unit test asserting the token and that `<N>` parses as an integer). `spacedock doctor --plugin-manifest <fixture>` is exercised against four fixture manifests and asserts the exact verdict for each: **compatible** (range brackets the binary's contract -> exit 0, "OK"), **too-old-binary** (`requires-contract` lower bound above the binary's contract -> non-zero, names binary-upgrade remedy), **too-old-plugin** (`requires-contract` upper bound at/below the binary's contract -> non-zero, names plugin-update remedy), and **no-plugin-found** (manifest path absent -> reports "no installed Spacedock plugin found", does not assert compatibility). Each non-OK verdict's stderr contains the exact remedy command for its class.

**AC-2 - The FO Startup procedure version-gates as step 0, and the gate aborts on mismatch.**
Verified by: a static skill-text test over the vendored FO contract (extending `skills/integration/skill_text_test.go`) asserts Startup's first numbered step runs `spacedock --version`, parses the `contract` token, compares against the FO contract's embedded expected range, and aborts with the standard actionable message on mismatch — present before the `--discover` / `--boot` steps. A behavior fixture exercises the abort path: a `spacedock --version` stub reporting a contract outside the FO's expected range drives a harness that asserts the gate emits the abort message and performs no discovery/dispatch call.

**AC-3 - `spacedock claude` / `spacedock codex` launch the stable plugin-registered agent, version-gated, failing fast before launch on mismatch.**
Verified by: a Go test drives `spacedock claude` through an injectable launch seam (the same seam pattern as the existing `status.Runner`): on a compatible contract it asserts the seam is invoked with argv beginning `claude --agent spacedock:first-officer`; on each mismatch class it asserts the launch seam is NOT invoked and the process exits non-zero with the actionable message. The Codex front-end resolves analogously and is asserted at whatever fidelity the Codex launch surface supports (full launch-seam test if a `--agent` analog exists; documented-command + version-gate-only test otherwise — see OPEN-resolution).

**AC-4 - `spacedock init` installs the per-host plugin via the host's plugin mechanism, with no per-agent skill-file copies.**
Verified by: a Go test drives `spacedock init --host claude` through an injectable install seam and asserts it issues the host plugin-install command (not a filesystem copy into `~/.claude/skills`); a companion check asserts no path under `~/.claude/skills` outside the plugin's own install root is written (fixture HOME, assert the directory tree is untouched). The Codex path is documented; implemented and tested via the same seam if the Codex plugin mechanism supports programmatic install, otherwise documented as a manual `codex plugin marketplace add` + `install` step (see OPEN-resolution).

**AC-5 - The published plugin is the amended skill surface (calls `spacedock`, zero plugin-private-path refs) with a declared contract-version.**
Verified by: a static skill-text test (extending the existing `TestNoPluginStatusPathInVendoredSkills`) asserts the vendored FO/ensign contracts call `spacedock status` / `spacedock dispatch` and contain zero `skills/commission/bin/status` and zero `skills/commission/bin/claude-team` references in the dispatch path; a manifest test asserts each published plugin manifest (`.claude-plugin/plugin.json`, `.codex-plugin/plugin.json`) declares a well-formed `requires-contract` range that brackets the current binary `CONTRACT_VERSION`. NOTE: the zero-`claude-team`-ref half is satisfied only once `native-dispatch-helper` lands the native `dispatch` surface; until then the dispatch-path refs in `claude-first-officer-runtime.md` remain and this AC half is BLOCKED (see Dependencies).

## Test gates

- `go test ./...`
- `spacedock --version` contract-version token + `spacedock doctor` compat across all four fixture classes (compatible / too-old-binary / too-old-plugin / no-plugin-found).
- FO Startup version-gate abort-path: static skill-text assertion + behavior fixture driving the abort with a contract-out-of-range `--version` stub.
- `spacedock claude` launch-seam test: compatible launches `claude --agent spacedock:first-officer`; each mismatch class blocks launch and exits non-zero.
- `spacedock init` install-seam test: issues host plugin-install, writes nothing into `~/.claude/skills` outside the plugin install root.
- Static: plugin contracts call `spacedock`; zero plugin-private-path refs (extends `skills/integration/skill_text_test.go`); each host manifest declares a well-formed `requires-contract` bracketing the binary contract.

## Test plan

Behavior-first, fixtures where a live host-plugin install is not the claim.

- **Contract comparison + doctor verdicts (Go unit, fixtures).** The contract-range parse-and-compare is pure logic; test it directly with table-driven fixtures over the four verdict classes plus malformed-range inputs (loud parse error, not a silent "compatible"). `doctor` reads a manifest path so fixtures are plain `plugin.json` files under a testdata dir. No host, no network. Cheap; this is the riskiest path (the gate itself) and is validated first per the mechanism-check-before-comprehensive-run discipline.
- **FO Startup abort path (behavior fixture).** A `spacedock` stub whose `--version` prints a chosen `contract N` drives a small harness exercising the step-0 gate logic; assert the abort message and that no discovery/dispatch call is made. Plus the static skill-text assertion that step 0 exists and is ordered first. The static test is prose-shape; the behavior fixture is the real claim.
- **Front door / init (Go unit, injectable seams).** Reuse the existing `status.Runner`-style injectable-seam pattern (`internal/cli` already injects the status runner) so launch and install are driven without exec-ing a real host or mutating a real `~/.claude`. A live `claude --agent` smoke is NOT the claim and is deferred to validation's optional spot-check, not the unit gate. Fixture HOME for the no-skill-file-copy assertion.
- **Manifest contract declaration (static).** Assert `requires-contract` is present and well-formed in both host manifests and brackets the current `CONTRACT_VERSION`.

Estimated cost/complexity: moderate. The comparison logic, doctor, and front-door/init seams are small stdlib-only Go (string range parse, manifest read, exec seam). The bulk of risk is in two coupling points, not raw volume: (1) the dependency on `native-dispatch-helper` for the native `dispatch` surface and the zero-`claude-team`-ref half of AC-5; (2) the Codex front-door/init fidelity, which depends on what the Codex agent-launch + programmatic-install surface actually supports. No live workflow run is needed for any AC; fixtures and injectable seams cover all five.

## Dependencies and staff review

**Depends on `native-dispatch-helper`.** The `spacedock dispatch` subcommand that the front door sits beside, and the AC-5 requirement of zero `skills/commission/bin/claude-team` refs in the dispatch path, are BOTH delivered by `native-dispatch-helper` (entity `7w8w5nsa5mbc807b3jb88psv`). This packaging work should sequence AFTER that entity reaches at least implementation, or explicitly scope AC-5's zero-`claude-team`-ref half as BLOCKED-until-native-dispatch and ship the rest (contract axis, 3-point gate, `claude`/`codex`/`init`/`doctor`, `requires-contract` declaration) first. The contract-version axis itself is independent and can land before native dispatch — the first contract bump simply happens when the native `dispatch` surface changes the flag surface the skill calls.

**Staff review: recommended.** This ideation touches a new on-disk contract (the `requires-contract` manifest field + `CONTRACT_VERSION` axis), a new cross-component compatibility protocol, and the FO Startup contract — exactly the "skill integration / new on-disk format" class the workflow README flags for independent ideation review. Recommend a staff review of (a) the integer-contract-axis vs alternatives, (b) the three-point gate's mismatch-class messaging, and (c) the bump-discipline rule below, before presenting the ideation gate.

## Notes

Develop on the spacedock repo's `next` branch (copy/adapt the skill surface from `~/git/spacedock`), flip to main when ready.

### OPEN-1 resolved — installing each host plugin from a specific git branch for pre-release

Both hosts support pinning a marketplace add to a git ref, so `next` (or any pre-release branch) is installable without merging to the default branch.

**Claude Code** — pin the marketplace to the branch with `@ref`, then install:
```bash
claude plugin marketplace add clkao/spacedock@next
claude plugin install spacedock@spacedock
```
- `@ref` after the `owner/repo` shorthand pins the marketplace SOURCE to the branch (here `next`); without it the add defaults to the repo's default branch.
- `spacedock@spacedock` is `plugin-name@marketplace-name` (the marketplace and the plugin are both named `spacedock` per the existing `marketplace.json`).
- The marketplace source accepts `ref` (branch/tag) but NOT `sha`; for an exact-commit pin you pin the per-plugin `source` inside `marketplace.json` (which supports both `ref` and `sha`). For branch-based pre-release testing, `@next` on the marketplace add is sufficient.
- Refresh after pushing new commits to `next`: `claude plugin marketplace update spacedock` (does not lose the installed plugin), then reinstall/restart as the host requires.
- `spacedock init` for Claude Code SHOULD shell out to exactly this command pair (with the branch parameterized) so the binary and the documented manual path stay identical.

**Codex** — the Codex marketplace add takes a ref via either suffix or flag:
```bash
codex plugin marketplace add clkao/spacedock --ref next
# or: codex plugin marketplace add clkao/spacedock@next
codex plugin install spacedock@spacedock
```
- `--ref next` (or the `@next` suffix) pins the Git marketplace source to the branch. `--sparse <path>` is available if a monorepo subpath is ever needed (not required here — the plugin package is the repo root via the existing `.agents/plugins/marketplace.json` direct-root source).
- Refresh after new commits: `codex plugin marketplace upgrade spacedock` (or all with no name), then reinstall/restart per Codex.
- Codex front-door/init fidelity caveat: `spacedock codex` launching a registered first-officer agent depends on Codex's agent-launch surface (`--enable multi_agent` + a `--agent`-equivalent). Where that surface exists, implement and test the launch seam; where it does not yet, `spacedock codex` documents the manual "use the spacedock:first-officer skill" invocation and the front-door reduces to a version-gate + documented-launch. AC-3/AC-4 are written to accept this reduced Codex fidelity explicitly rather than block on it.

(Sources: Claude Code plugin-marketplaces docs — `@ref`/`#ref` on marketplace add, marketplace source supports `ref` not `sha`, `marketplace update` refresh; Codex `plugin marketplace add owner/repo --ref` / `@ref` + `--sparse`, `plugin marketplace upgrade` refresh.)

### OPEN-2 resolved — contract-version bump discipline and owner

**Rule.** Bump `CONTRACT_VERSION` (integer +1) when, and only when, a change to the `spacedock` binary alters the observable contract the vendored skill surface depends on: removing/renaming a subcommand or flag the FO/ensign contracts call (`status --discover|--boot|--set|--archive|--resolve`, future `dispatch …`), changing a `--boot` / status output section the FO parses, or changing dispatch-spec emission shape. Do NOT bump for internal refactors, bug fixes that preserve the flag/output surface, or additive flags the skill does not yet call. The bump is a deliberate decision, never an automatic side effect of a semver/release bump.

**Coupled release rule.** A `CONTRACT_VERSION` bump MUST land together with a matching plugin release that updates every host manifest's `requires-contract` range AND any FO/ensign contract text that calls the changed surface, in the same change set. A contract bump without a matching plugin release (or vice versa) is the exact mismatch the gate is designed to catch — so the discipline is to never ship them apart. A static test (extending `skill_text_test.go`) asserts each manifest's `requires-contract` brackets the current `CONTRACT_VERSION`, turning "forgot to bump the manifest" into a red test rather than a runtime mismatch a user discovers.

**Owner.** The owner is the change author who modifies the binary's observable surface, enforced by the bracketing static test as the mechanical backstop and by the ideation/validation gates of THIS workflow as the human backstop. Mirrors the existing `scripts/release.sh`-stamps-the-manifest-assertion pattern in `~/git/spacedock` (where a manifest-version assertion is stamped at bump time): the contract-bump equivalent is the bracketing test failing until the manifest range is moved. The rule and its enforcing test live with the binary's contract definition (the package that holds `CONTRACT_VERSION`), so the owner of a surface change cannot land it without confronting the gate.

This closes the self-hosting + distribution gap: a fresh host session installs the version-gated plugin, the plugin's agents call the native `spacedock` binary for status + dispatch, and three independent checkpoints refuse to run a skill surface against an incompatible binary with an actionable remedy.

## Stage Report: ideation

- DONE: Settle the contract-version axis + 3-point version-gate design (spacedock claude front door, init/doctor, FO Startup step 0) with actionable mismatch behavior; make AC-1/AC-2 testable (compatible / too-old-binary / too-old-plugin / abort-path).
  `## Design` settles an integer CONTRACT-VERSION axis distinct from semver, the `requires-contract` half-open range, the two mismatch classes, the three gate points, and one shared message shape. AC-1 now names all four doctor verdict classes with fixtures; AC-2 names the static step-0 assertion + the abort-path behavior fixture.
- DONE: Resolve OPEN-1 (Claude Code AND Codex branch install) with exact commands.
  `### OPEN-1 resolved`: `claude plugin marketplace add clkao/spacedock@next` + `claude plugin install spacedock@spacedock` (+ `marketplace update` refresh); `codex plugin marketplace add clkao/spacedock --ref next` + install (+ `marketplace upgrade`). Sourced from Claude Code/Codex plugin-marketplace docs (web research).
- DONE: Resolve OPEN-2 (contract-version bump discipline + owner).
  `### OPEN-2 resolved`: bump rule (observable-surface change only), coupled-release rule (manifest + skill text land together), owner = surface-change author, enforced by a bracketing static test mirroring `~/git/spacedock`'s release.sh manifest-assertion stamp.
- DONE: Behavior-first test plan using fixtures; flag the native-dispatch-helper dependency and whether ideation warrants staff review.
  `## Test plan` is fixture/injectable-seam-first (no live host/network), riskiest path (the gate) validated first. `## Dependencies and staff review` flags AC-5's zero-`claude-team`-ref half as BLOCKED-until-`native-dispatch-helper` (`7w8w5nsa5mbc807b3jb88psv`) and recommends staff review (new on-disk contract + FO Startup contract).

### Summary

Settled the distribution model's compatibility spine: an integer CONTRACT-VERSION (decoupled from release semver) advertised in `--version`, a `requires-contract` half-open range in each host plugin manifest, and one comparison run at three points (front door fail-fast, init/doctor install-time, FO Startup step-0 safety net) with two distinguishable mismatch classes and a single actionable-message shape. Both OPEN questions are closed with exact, host-verified branch-install commands and a bump-discipline rule backed by a bracketing static test. The test plan is fixture/seam-first with no live-host claim; the one cross-entity coupling — AC-5's zero-`claude-team`-ref half — is explicitly scoped BLOCKED-until-`native-dispatch-helper`, and independent staff review is recommended given the new on-disk contract and FO Startup contract changes. No code written (ideation stage); frontmatter untouched.
