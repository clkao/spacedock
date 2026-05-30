---
id: tq66yjc7sqbhyc52eg8h2ecx
title: Spacedock packaging and distribution
status: implementation
source: handoff self-hosting gap
score: "0.35"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-spacedock-packaging
started: 2026-05-30T19:18:28Z
---

Define and build the v1 distribution model so a fresh Claude Code or Codex session runs the Spacedock first officer from the repo's own native binary + a version-gated plugin — no Python in the dispatch path, and no contract files copied into per-agent skill folders.

Model (decided with the captain):
- The `spacedock` binary is the canonical artifact: `status` + `dispatch` (native, via `native-dispatch-helper`) + host front-ends `spacedock claude` / `spacedock codex` + `init` / `doctor`. It exposes a CONTRACT-VERSION in `--version`.
- Per-host PLUGINS (Claude Code AND Codex) register the stable agents `spacedock:first-officer` / `spacedock:ensign` (named `--agent`/subagent_type entry points) whose contracts call `spacedock status` / `spacedock dispatch`. The plugin IS the amended skill surface (already vendored under `skills/`, zero plugin-private-path refs), published per host. `spacedock init` USES the host plugin mechanism (installs the plugin) — it does NOT copy contract files into `~/.claude/skills`.
- `spacedock claude` wraps `claude --agent spacedock:first-officer …`; `spacedock codex` the Codex analog. One front door, host-native underneath.
- VERSION GATE on a CONTRACT-VERSION axis (not raw semver — bump only when the binary's flag surface / observable behavior the skill depends on changes). The plugin manifest declares `requires-contract: ">=N,<N+1"`; checked at three points: `spacedock claude` (front door, fail fast), `spacedock init`/`doctor` (install/upgrade), and FO Startup step 0 (per-session safety net). Mismatch -> actionable upgrade message. (Mirrors roborev's daemon/cli version-mismatch detection.)

## Design

### Spike findings (cycle 2, empirically grounded)

A behavioral spike (2026-05-30) ran the OPEN-1 host commands and the cross-repo manifest read on this machine. Observed reality, not docs:

- **Two repos, one Go module.** `spacedock-v1` (`/Users/clkao/git/spacedock-research/spacedock-v1`) has `go.mod`; `~/git/spacedock` has NO `go.mod` and no `next` branch (local branches: `main` + feature/backup; the real upstream is `clkao/spacedock` per `~/.claude/plugins/known_marketplaces.json`, but `next` is not pushed there yet). No `plugin.json`/`marketplace.json` is vendored into `spacedock-v1` today — only the skill tree (`commission/ ensign/ first-officer/ integration/`) is. So a Go test in `spacedock-v1` CANNOT read `requires-contract` from `~/git/spacedock` without a vendored fixture or a brittle absolute cross-repo path. This is the structural defect; the allocation below resolves it.
- **`requires-contract` is a tolerated unknown field.** `claude plugin validate` on a manifest carrying `requires-contract` emits a WARNING ("Unknown field 'requires-contract'. Claude Code ignores it at load time.") and PASSES (exit 0). The existing `interface` field draws the identical warning and has shipped in production for releases. So the host ignores the field, never rejects it, and the binary reads it itself — the design is viable. The warning is expected, not a defect; `spacedock doctor` output should note it so operators are not alarmed.
- **The installed manifest is resolvable at a deterministic path.** `claude plugin marketplace add <path>` + `claude plugin install spacedock@spacedock` (run with isolated `CLAUDE_CONFIG_DIR` + `CLAUDE_CODE_PLUGIN_CACHE_DIR`) install end-to-end with exit 0, and `claude plugin list --json` then reports `"installPath": ".../cache/spacedock/spacedock/<version>"`. The installed `requires-contract` is intact at `<installPath>/.claude-plugin/plugin.json`. So `spacedock doctor`'s production resolver shells `claude plugin list --json`, reads `installPath` for `spacedock@spacedock`, and parses that manifest — no cache-path globbing.
- **`.codex-plugin/plugin.json` is authoritative; `.claude-plugin` is generated.** `scripts/release.sh` in `~/git/spacedock` declares `AUTHORITATIVE_PLUGIN_JSON=".codex-plugin/plugin.json"` and `sync_legacy_plugin_manifest()` does a full `json.loads → json.dumps` copy into the legacy `.claude-plugin/plugin.json`. Adding `requires-contract` to the authoritative Codex manifest propagates it verbatim to the legacy Claude manifest — the two stay content-identical by construction. The field MUST be added to `.codex-plugin/plugin.json`; editing the legacy copy desyncs until the next release run.
- **Codex has no `--agent` analog (confirms the captain's forced fork).** `codex --help` exposes no agent-selection flag; `codex exec` takes free-text "initial instructions for the agent" but cannot launch a named registered subagent. So `spacedock codex` reduces to version-gate + documented prose. Codex commands also differ from cycle-1's text: install is `codex plugin add spacedock@spacedock` (NOT `codex plugin install`); marketplace add is `codex plugin marketplace add <owner/repo[@ref]> [--ref REF] [--sparse PATH]`; refresh is `codex plugin marketplace upgrade`.

### Cross-repo test allocation (resolves the structural defect)

Each AC's test lands where its subject lives, and the four sources of truth for the contract are kept mechanically co-located:

| Surface under test | Lives in | Test home |
|---|---|---|
| `CONTRACT_VERSION` constant, `--version` token, `doctor` verdicts, front-door/init seams | `spacedock-v1` (Go) | `internal/cli` + `internal/contract` Go unit tests |
| FO Startup step-0 embedded range, vendored FO/ensign calling `spacedock status`/`dispatch` | `spacedock-v1/skills/*` (vendored) | `skills/integration/skill_text_test.go` (static) |
| Plugin manifest `requires-contract` (authoritative `.codex-plugin/plugin.json`) | `~/git/spacedock` (no Go) | a manifest fixture VENDORED into `spacedock-v1/internal/contract/testdata/`, kept in sync by a checked-in copy + a `~/git/spacedock`-side packaging test |

**Four sources of truth for the contract integer, and how each stays in sync:**
1. `CONTRACT_VERSION` in the binary (`spacedock-v1`) — the source of truth.
2. FO Startup step-0 embedded range (`spacedock-v1/skills/first-officer/...`) — a Go static test asserts this embedded range brackets `CONTRACT_VERSION` (both live in `spacedock-v1`, so a single `go test` enforces it).
3. Authoritative manifest `requires-contract` (`~/git/spacedock/.codex-plugin/plugin.json`) — bracketing checked against a VENDORED fixture copy in `spacedock-v1`, plus a `~/git/spacedock`-side packaging test (mirroring the existing `test_codex_plugin_manifest_*` pattern that `release.sh` already stamps) asserting the authoritative manifest matches the vendored fixture's range.
4. Legacy `.claude-plugin/plugin.json` — generated from #3 by `release.sh`, so it carries `requires-contract` automatically; the `~/git/spacedock`-side packaging test asserts legacy == authoritative.

The vendored fixture is the seam that lets a `spacedock-v1` Go test bracket the manifest range against `CONTRACT_VERSION` without reading a non-Go sibling repo at test time. The cross-repo drift risk (fixture vs real manifest) is closed by the `~/git/spacedock`-side packaging test, not by the Go test reaching across repos.

### Contract-version axis

The compatibility axis is a single monotonic integer, the CONTRACT-VERSION, distinct from the plugin's display `version` (semver) and from the binary's build/release version. It names the binary's *observable contract that the skill surface depends on*: the set of `spacedock` subcommands, flags, and output sections (`--discover`, `--boot`, `--set`, `--archive`, `--resolve`, and the future `dispatch` surface) plus the parse shapes those produce. A release that only changes internals, fixes a bug without altering the flag surface, or adds a new flag the skill does not yet require does NOT bump the contract. A release that removes/renames a flag the skill calls, changes a `--boot` output section the FO parses, or changes dispatch-spec emission DOES bump it.

- The binary holds two constants: `CONTRACT_VERSION` (integer, e.g. `1`) and the existing semver build version. `--version` prints both, e.g. `spacedock 0.2.0 (contract 1)`.
- The plugin manifest declares the contract range it was authored against as `requires-contract`. It is expressed as a half-open integer range string `">=N,<M"` so a plugin can declare forward-tolerance (`">=1,<3"` accepts binaries at contract 1 or 2). The common case is `">=N,<N+1"` (exact-contract pin).
- A binary at contract `C` is COMPATIBLE with a plugin declaring `">=lo,<hi"` iff `lo <= C < hi`. Outside that interval there are two distinguishable failures:
  - **too-old-binary** — `C < lo`. The installed `spacedock` predates the contract this plugin needs. Remedy: upgrade/rebuild the binary.
  - **too-old-plugin** — `C >= hi`. The installed plugin predates the binary's contract. Remedy: update/reinstall the plugin (`spacedock init` or the host's plugin update command).

Storing the axis as an integer (not semver) keeps the comparison a total order with no pre-release/build-metadata ambiguity, and makes the bump decision a deliberate human act rather than a side effect of routine version bumps.

### Where the plugin's declared range is read

The gate needs the *installed plugin's* `requires-contract` at runtime. The binary discovers it from the installed plugin manifest (resolution paths confirmed by spike):
- Claude Code: shell `claude plugin list --json`, find the `spacedock@spacedock` entry, read its `installPath`, and parse `<installPath>/.claude-plugin/plugin.json` for `requires-contract`. (Spike confirmed `installPath` is reported and the field is intact there.) The `claude plugin list --json` call is an injectable seam for tests.
- Codex: the analogous `codex plugin list` resolution against the installed `.codex-plugin/plugin.json`.
- `spacedock doctor [--plugin-manifest PATH]` accepts an explicit manifest path that bypasses host-cache resolution — used by fixtures and by operators debugging a specific manifest.
- `spacedock doctor` reads the manifest, parses `requires-contract`, compares against the binary's `CONTRACT_VERSION`, and prints one of five verdicts (see AC-1). When no installed plugin is found (host reports none, or `installPath` missing), `doctor` reports **no-plugin-found** ("no installed Spacedock plugin found") rather than asserting compatibility — a distinct, non-fatal-by-default state.

**Front-door behavior when the installed manifest is unresolvable.** At the `spacedock claude`/`codex` front door, an unresolvable manifest (host CLI errors, no `spacedock@spacedock` entry, malformed `installPath`) is NOT treated as "compatible." The front door prints a one-line warning naming the unresolvable cause and the remedy (`run spacedock doctor` / `spacedock init`) and — because the front door's job is fail-fast safety — exits non-zero WITHOUT launching the host. The exception is an explicit operator override (`spacedock claude --skip-contract-check`) for the bootstrap case where the plugin is being installed for the first time. This makes "I have a binary but no installed plugin" a loud, actionable state at the front door rather than a silent launch into a session that the FO step-0 gate would then abort anyway.

`requires-contract` carrying the contract range in the manifest (rather than the binary carrying a min-plugin-version) keeps both directions checkable from data both sides already have: the binary always knows its own `C`; the manifest always travels with the plugin. The host ignores the field (spike: warned-but-tolerated), so adding it costs nothing on the host side.

### Three-point version gate

The same comparison runs at three points; each point differs only in *what it does on mismatch*.

1. **`spacedock claude` / `spacedock codex` front door (fail fast).** Before exec-ing the host (`claude --agent spacedock:first-officer …` / Codex analog), run the comparison. On mismatch, print the actionable message to stderr and exit non-zero WITHOUT launching the host — the operator never reaches a session that would fail at startup. On compatible, exec the host. (AC-3)
2. **`spacedock init` / `spacedock doctor` (install/upgrade time).** `init` installs/updates the host plugin via the host mechanism, then runs `doctor`. `doctor` is the standalone compatibility report (AC-1). This is where a user lands after upgrading either side; the message names which side to move.
3. **FO Startup step 0 (per-session safety net).** The vendored FO contract's Startup begins with a contract check: run `spacedock --version`, parse the `contract N` token, and confirm it satisfies the contract range the FO contract was authored against (the FO contract embeds its own expected range as a literal, so the safety net does not depend on the plugin manifest being readable from inside the agent). On mismatch, abort startup with the actionable message before any discovery/dispatch. This catches the case where the binary on PATH at session time differs from the one present at install time. (AC-2)

The actionable message is identical in shape across all points (only the leading context line differs). The per-class remedy strings are pinned literals, parameterized only by the host (`claude`/`codex`) and the dev branch when pre-release:

```
Spacedock contract mismatch: binary is contract <C>, plugin requires <range>.
  <remedy line per class below>
Run `spacedock doctor` for details.
```

Per-class remedy lines (the `<host>` and `@<branch>` parameters are filled from the detected host and, for pre-release, the configured dev branch; default release path omits `@<branch>`):

- **too-old-binary** (`C < lo`): `too-old-binary: your spacedock binary (contract <C>) predates this plugin (needs <range>). Rebuild/upgrade spacedock: go install github.com/clkao/spacedock-v1/cmd/spacedock@latest (or pull and 'go build').`
- **too-old-plugin** (`C >= hi`): `too-old-plugin: your installed plugin (needs <range>) predates this binary (contract <C>). Update it: spacedock init --host <host> (or '<host> plugin update spacedock').`
- **malformed-range** (manifest `requires-contract` does not parse as `">=N,<M"`): `malformed contract range <raw> in <manifest-path>: expected ">=N,<M". This is a packaging bug — the plugin manifest is wrong, not your install.` — no install/upgrade remedy (neither side is "too old"); exits non-zero so a broken manifest fails loudly rather than silently passing as compatible.
- **no-plugin-found**: `no installed Spacedock plugin found for host <host>. Install it: spacedock init --host <host>.`

These literals are the test oracle for AC-1/AC-2/AC-3: each verdict's stderr is asserted to contain its pinned remedy substring. The derivation rule (host + optional branch substitution) is the only dynamic part and is itself unit-tested.

### Front door and init/doctor surface

New `spacedock` subcommands (built once `native-dispatch-helper` lands the `dispatch` surface they sit beside):
- `spacedock claude [args…]` — version-gate, then `exec claude --agent spacedock:first-officer` with passthrough args.
- `spacedock codex [args…]` — version-gate, then the Codex front-officer launch analog (documented; implemented where the Codex agent-launch surface allows it — see OPEN-resolution below).
- `spacedock init [--host claude|codex] [--check]` — install/update the per-host plugin via the host's plugin mechanism, then run `doctor`. `--check` runs the compatibility report without installing. No writes into `~/.claude/skills` outside the host's own plugin install. (AC-4)
- `spacedock doctor [--plugin-manifest PATH]` — print the compatibility report (OK / too-old-binary / too-old-plugin / no-plugin-found). (AC-1)

The plugin published per host IS the amended skill surface already vendored under `skills/` (FO/ensign references calling `spacedock status` / `spacedock dispatch`, zero `skills/commission/bin/*` refs once `native-dispatch-helper` lands). `spacedock init` installing the plugin — not copying skill files — is what makes `Skill()` / `--agent spacedock:first-officer` resolve. (AC-5)

## Acceptance criteria

Every AC names a behavioral exercise-and-observe oracle (run a command / drive a seam, observe output and side effects), not a grep over prose. The one unavoidably-static surface (AC-2's "step 0 exists in the FO text") is paired with a behavioral abort-path exercise so the prose claim never stands alone.

**AC-1 - `spacedock --version` reports a contract-version and `spacedock doctor` distinguishes all five compatibility verdicts.**
Verified by: a Go test runs `spacedock --version` and asserts the output contains a `contract <N>` token where `<N>` equals the `CONTRACT_VERSION` constant and parses as an integer; the existing exact-match `TestVersion` is updated in the same change (the contract token breaks its `== "spacedock "+Version` assertion — see Dependencies). A table-driven Go test runs `spacedock doctor --plugin-manifest <fixture>` against fixtures under `internal/contract/testdata/` and observes exit code + stderr for each of five verdicts: **compatible** (range brackets `CONTRACT_VERSION` -> exit 0, "OK"), **too-old-binary** (`lo > C` -> non-zero, stderr contains the pinned too-old-binary remedy), **too-old-plugin** (`hi <= C` -> non-zero, pinned too-old-plugin remedy), **malformed-range** (`requires-contract` not parseable as `">=N,<M"` -> non-zero, pinned malformed-range message, NO upgrade remedy), and **no-plugin-found** (manifest absent / host reports no install -> reports "no installed Spacedock plugin found", does not assert compatibility). The five pinned remedy/verdict strings are the oracle.

**AC-2 - The FO Startup procedure version-gates as step 0, the embedded range stays bracketed to `CONTRACT_VERSION`, and the gate aborts on mismatch.**
Verified by THREE oracles: (1) a static skill-text test (extending `skills/integration/skill_text_test.go`) asserts Startup's first numbered step runs `spacedock --version`, parses the `contract` token, and aborts before the `--discover`/`--boot` steps; (2) a Go test parses the range literal embedded in the vendored FO contract text and asserts it brackets the `CONTRACT_VERSION` constant — closing the 4th-source-of-truth drift with a single `go test` (both surfaces live in `spacedock-v1`); (3) a behavior fixture: a `spacedock` stub whose `--version` prints a contract OUTSIDE the FO's embedded range drives a harness that observes the gate emits the pinned abort message and makes NO discovery/dispatch call (assert the stub's `status --discover`/`--boot` subcommands were never invoked).

**AC-3 - `spacedock claude` launches the plugin-registered agent version-gated; `spacedock codex` is a version-gate + documented prose launch (Codex has no `--agent` analog).**
Verified by: a Go test drives `spacedock claude` through an injectable launch seam (the `status.Runner` pattern already in `internal/cli`): on a compatible contract it observes the seam invoked with argv beginning `claude --agent spacedock:first-officer`; on each mismatch verdict it observes the launch seam is NOT invoked and the process exits non-zero with the pinned remedy. For Codex, the spike confirmed NO `--agent`/named-subagent launch flag exists, so `spacedock codex` is SCOPED to: run the version gate, then print the documented "use the spacedock:first-officer skill in Codex" prose and exit (no agent-launch exec). The Go test observes the version gate runs and the documented prose is emitted on compatible; observes non-zero + pinned remedy on mismatch. "Done" for AC-3 explicitly does NOT claim Codex agent-launch.

**AC-4 - `spacedock init --host claude` installs the per-host plugin via the host plugin mechanism with no per-agent skill-file copies; the Codex path is documented prose.**
Verified by: a Go test drives `spacedock init --host claude` through an injectable install seam and observes it issues the host plugin commands (`claude plugin marketplace add … && claude plugin install spacedock@spacedock`) rather than a filesystem copy; a companion behavioral check runs the real command pair against a fixture `CLAUDE_CONFIG_DIR`/`CLAUDE_CODE_PLUGIN_CACHE_DIR` (spike-proven to work in isolation, exit 0) and observes (a) `claude plugin list --json` reports the installed `spacedock@spacedock` with an `installPath`, and (b) NO path under the fixture `~/.claude/skills` outside the plugin install root was written (assert that tree is untouched). For Codex, `spacedock init --host codex` emits the documented `codex plugin marketplace add <src> [--ref <branch>]` + `codex plugin add spacedock@spacedock` command pair as prose (Codex install verb is `add`, not `install` — spike-confirmed); whether it shells those commands or only prints them is an implementation choice the test pins to the chosen behavior. "Done" for AC-4 does NOT claim programmatic Codex install beyond emitting the correct commands.

**AC-5a - The vendored skill surface calls `spacedock status` and the published manifest declares a well-formed `requires-contract` bracketing `CONTRACT_VERSION` (testable now).**
Verified by: a static skill-text test (extending `TestNoPluginStatusPathInVendoredSkills`) asserts the vendored FO/ensign contracts call `spacedock status` and contain zero `skills/commission/bin/status` refs; a Go test reads the VENDORED manifest fixture (`internal/contract/testdata/plugin.json`, a checked-in copy of the authoritative `.codex-plugin/plugin.json`) and observes its `requires-contract` parses and brackets `CONTRACT_VERSION`; and a `~/git/spacedock`-side packaging test (mirroring the existing `test_codex_plugin_manifest_*` pattern) observes the authoritative `.codex-plugin/plugin.json` matches the vendored fixture's range AND the generated legacy `.claude-plugin/plugin.json` equals the authoritative one. This closes fixture-vs-real drift without a cross-repo read at Go-test time.

**AC-5b - The dispatch path carries zero `claude-team` refs (closed by native-dispatch-helper).**
Verified by: a static skill-text test asserts zero `skills/commission/bin/claude-team` refs in the dispatch path of the vendored FO/ensign surface. NOTE: this is delivered by `native-dispatch-helper` (`7w8w5nsa5mbc807b3jb88psv`, now in implementation), which replaces the Python `claude-team build`/`show-stage-def` dispatch refs in `claude-first-officer-runtime.md` with native `spacedock dispatch`. AC-5b is satisfied only after that entity lands; this packaging entity sequences AFTER it (dispatch-first, per captain). Until then AC-5b is BLOCKED-by-dependency, not failed.

## Test gates

- `go test ./...` (all Go-side tests live in `spacedock-v1`).
- `spacedock --version` contract-token test + updated `TestVersion`; `spacedock doctor` across all FIVE verdict fixtures (compatible / too-old-binary / too-old-plugin / malformed-range / no-plugin-found), each asserting exit code + pinned stderr substring.
- FO Startup version-gate: static step-0-ordering assertion + Go embedded-range-brackets-`CONTRACT_VERSION` test + behavior fixture driving the abort with a contract-out-of-range `--version` stub (assert no `status --discover`/`--boot` call).
- `spacedock claude` launch-seam test: compatible -> seam invoked with `claude --agent spacedock:first-officer`; each mismatch verdict -> seam NOT invoked, exit non-zero, pinned remedy.
- `spacedock init --host claude` install-seam test + isolated-`CLAUDE_CONFIG_DIR` behavioral install observing `plugin list --json` reports the install and `~/.claude/skills` is untouched outside the plugin root.
- Static + fixture: vendored FO/ensign call `spacedock status`, zero `commission/bin/status` refs (AC-5a); vendored manifest fixture's `requires-contract` brackets `CONTRACT_VERSION`; `~/git/spacedock`-side packaging test: authoritative `.codex-plugin/plugin.json` matches the fixture range and the generated legacy `.claude-plugin/plugin.json` equals authoritative.

## Test plan

Behavior-first, fixtures where a live host-plugin install is not the claim. Spike-grounded: the riskiest paths (the gate logic, the cross-repo manifest read, the host install) were exercised on this machine before this plan was written.

- **Contract comparison + doctor verdicts (Go unit, `internal/contract`).** The range parse-and-compare is pure logic; table-driven over all five verdicts, fixtures are plain `plugin.json` files under `internal/contract/testdata/`. No host, no network. Riskiest path -> validated first. malformed-range is a first-class verdict (loud parse error, no upgrade remedy), not an afterthought.
- **FO Startup abort path (behavior fixture + Go bracketing).** A `spacedock` stub whose `--version` prints a chosen `contract N` drives a harness over the step-0 gate; observe the pinned abort message and that no discovery/dispatch subcommand is invoked. The embedded-range-brackets-`CONTRACT_VERSION` check is a Go test over the vendored FO text (both in `spacedock-v1`). The static step-0-ordering assertion is the prose half; the behavior fixture + bracketing test are the load-bearing claims.
- **Front door / init (Go unit seams + one isolated behavioral install).** Launch and install run through injectable seams (the `status.Runner` pattern already in `internal/cli`) so unit tests need no real host. ONE behavioral install runs the real `claude plugin marketplace add`/`install` against an isolated `CLAUDE_CONFIG_DIR`/`CLAUDE_CODE_PLUGIN_CACHE_DIR` (spike-proven safe and exit-0) to observe `installPath` and the untouched-`skills` invariant — this is the real AC-4 claim, kept hermetic by env isolation, not mocked. Live `claude --agent` interactive launch is NOT a unit gate.
- **Cross-repo manifest co-location (vendored fixture + sibling-repo packaging test).** `spacedock-v1` Go tests read the vendored `internal/contract/testdata/plugin.json` (never `~/git/spacedock` directly). The `~/git/spacedock`-side packaging test (Python, alongside the existing `test_codex_plugin_*` suite) is what asserts the vendored fixture range matches the authoritative manifest and that the legacy copy is generated from it. Drift is caught in the repo that owns the manifest.

Estimated cost/complexity: moderate, lower-risk than cycle 1 because the three structural unknowns are now spike-resolved (host ignores `requires-contract`; `installPath` is resolvable; `.codex-plugin` is authoritative and auto-syncs). Remaining risk is sequencing (AC-5b waits on `native-dispatch-helper`) and the modest discipline of keeping the vendored fixture synced — both have explicit enforcing tests. No live workflow run is needed for any AC.

## Dependencies and staff review

**Depends on `native-dispatch-helper` (sequencing settled: dispatch-first).** The `spacedock dispatch` subcommand the front door sits beside, and AC-5b (zero `skills/commission/bin/claude-team` refs in the dispatch path), are delivered by `native-dispatch-helper` (`7w8w5nsa5mbc807b3jb88psv`, now in implementation). The captain settled sequencing as dispatch-first: this packaging entity implements AFTER `native-dispatch-helper` lands, so the `spacedock dispatch` references resolve and AC-5b becomes satisfiable. Everything else (contract axis, 3-point gate, `claude`/`codex`/`init`/`doctor`, `requires-contract` declaration — AC-1/2/3/4/5a) is independent and could land first; AC-5b stays BLOCKED-by-dependency until the native dispatch surface exists. The contract-version axis is itself independent of dispatch.

**Other breaking-change to coordinate:** adding the `contract <N>` token to `--version` breaks the existing exact-match `TestVersion` (`internal/cli/cli_test.go:35`, asserts `== "spacedock "+Version`). The implementing change MUST update that test in the same commit — flagged so validation does not read it as an accidental regression.

**Staff review: recommended.** This ideation touches a new on-disk contract (the `requires-contract` manifest field + `CONTRACT_VERSION` axis), a new cross-component compatibility protocol, the FO Startup contract, AND a cross-repo test-co-location scheme (vendored fixture + sibling-repo packaging test) — exactly the "skill integration / new on-disk format" class the workflow README flags for independent ideation review. Recommend a staff review of (a) the cross-repo test allocation and four-sources-of-truth sync scheme, (b) the front-door unresolvable-manifest behavior + `--skip-contract-check` override, and (c) the bump-discipline rule, before presenting the ideation gate. (Cycle 1's audit already validated the integer axis + injectable-seam mechanism — those are settled and need not be re-reviewed.)

## Notes

Develop on the spacedock repo's `next` branch (copy/adapt the skill surface from `~/git/spacedock`), flip to main when ready.

### OPEN-1 resolved — installing each host plugin from a specific git branch for pre-release

Both host CLIs (verified present on this machine: `claude` and `codex 0.132.0`) support pinning a marketplace add to a git ref, so a pre-release branch is installable without merging to the default branch. Spike status of each command is marked: [RUN] = executed and observed; [SYNTAX] = host help/docs confirm the flag, but the exact `next`-branch target was not run because no `next` branch is pushed to `clkao/spacedock` yet (only the LOCAL-PATH equivalent was run end-to-end).

**Prerequisite:** create and push the `next` branch to the upstream `clkao/spacedock` (the marketplace's declared repo per `~/.claude/plugins/known_marketplaces.json`). The local clone's git remotes are forks (`ijac`, `kc`); `next` exists nowhere yet. Until pushed, pre-release testing uses the LOCAL-PATH install (spike-verified) against a checkout of the branch.

**Claude Code** — pin the marketplace SOURCE to the branch with `@ref`, then install:
```bash
claude plugin marketplace add clkao/spacedock@next     # [SYNTAX] (@ref confirmed; needs next pushed)
claude plugin install spacedock@spacedock              # [RUN] verified against a LOCAL-PATH marketplace
```
- `@ref` after `owner/repo` pins the marketplace source to the branch; without it the add defaults to the repo's default branch. The marketplace source accepts `ref` (branch/tag) but NOT `sha` (per host docs).
- `spacedock@spacedock` is `plugin-name@marketplace-name` (both named `spacedock` per the existing `marketplace.json`). [RUN] `claude plugin install spacedock@spacedock` against a local-path marketplace carrying `requires-contract` installed at exit 0 and `claude plugin list --json` reported a resolvable `installPath`.
- LOCAL-PATH equivalent for pre-merge testing (no branch push needed): `claude plugin marketplace add /path/to/spacedock-checkout` then the same install. [RUN].
- Refresh after pushing new commits: `claude plugin marketplace update spacedock` (keeps the installed plugin), then restart as the host requires. [SYNTAX]
- `spacedock init --host claude` SHOULD shell out to exactly this command pair (branch parameterized) so the binary and the documented manual path stay identical.

**Codex** (CLI verb differs from Claude — install is `add`, not `install`):
```bash
codex plugin marketplace add clkao/spacedock --ref next   # [SYNTAX] --ref confirmed in `codex plugin marketplace add --help`
# equivalently: codex plugin marketplace add clkao/spacedock@next   (owner/repo[@ref] is the documented SOURCE form)
codex plugin add spacedock@spacedock                      # [SYNTAX] verb is `add`; `codex plugin install` does NOT exist
```
- `--ref next` (or the `@next` suffix) pins the Git marketplace source to the branch; `--sparse <PATH>` is available for a monorepo subpath (not needed — the plugin package is the repo root via `.agents/plugins/marketplace.json`).
- Refresh after new commits: `codex plugin marketplace upgrade` (all) or `codex plugin marketplace upgrade <name>`. [SYNTAX]
- Codex front-door/init reality (spike-confirmed): `codex --help` exposes NO `--agent`/named-subagent launch flag. So `spacedock codex` is version-gate + documented prose, NOT an agent-launch wrapper — AC-3/AC-4 state this plainly so "done" does not overclaim. The manual invocation Codex users run is "use the spacedock:first-officer skill in this directory."

(Sources: live `claude plugin …` / `codex plugin …` CLI help on this machine (claude present; codex 0.132.0) plus the cycle-2 spike runs above for the [RUN]-marked claims; Claude Code plugin-marketplaces docs and Codex plugin docs for the [SYNTAX]-marked `@ref`/`--ref`/refresh claims not exercised against a pushed `next` branch.)

### OPEN-2 resolved — contract-version bump discipline and owner

**Rule.** Bump `CONTRACT_VERSION` (integer +1) when, and only when, a change to the `spacedock` binary alters the observable contract the vendored skill surface depends on: removing/renaming a subcommand or flag the FO/ensign contracts call (`status --discover|--boot|--set|--archive|--resolve`, future `dispatch …`), changing a `--boot` / status output section the FO parses, or changing dispatch-spec emission shape. Do NOT bump for internal refactors, bug fixes that preserve the flag/output surface, or additive flags the skill does not yet call. The bump is a deliberate decision, never an automatic side effect of a semver/release bump.

**Coupled release rule.** A `CONTRACT_VERSION` bump MUST land together with: (1) the authoritative `.codex-plugin/plugin.json` `requires-contract` range moved to match; (2) the FO Startup step-0 embedded range moved to match; (3) any FO/ensign contract text that calls the changed surface; (4) the vendored `internal/contract/testdata/plugin.json` fixture copy. The legacy `.claude-plugin/plugin.json` does NOT need a manual edit — spike-confirmed `scripts/release.sh::sync_legacy_plugin_manifest()` does a full `json.loads → json.dumps` copy of the authoritative Codex manifest, so `requires-contract` propagates verbatim. A contract bump without these is the exact mismatch the gate catches; the discipline is to never ship them apart.

**Mechanical backstops (the four sources of truth stay co-located):**
- `spacedock-v1` Go test: FO step-0 embedded range brackets `CONTRACT_VERSION` (both in-repo — one `go test`).
- `spacedock-v1` Go test: vendored fixture `requires-contract` brackets `CONTRACT_VERSION`.
- `~/git/spacedock` packaging test (alongside the existing `test_codex_plugin_manifest_*` suite that `release.sh` already stamps): authoritative `.codex-plugin/plugin.json` matches the vendored fixture range; legacy `.claude-plugin/plugin.json` equals the authoritative copy.

These turn "forgot to move a range" into a red test in whichever repo owns the surface, rather than a runtime mismatch a user discovers.

**Owner.** The owner is the change author who modifies the binary's observable surface, enforced by the bracketing tests above as the mechanical backstop and by this workflow's ideation/validation gates as the human backstop. The rule and the `spacedock-v1`-side enforcing tests live with `CONTRACT_VERSION` (the `internal/contract` package), so a surface change cannot land green without moving the ranges. The `~/git/spacedock`-side half lives with the manifest it guards, mirroring how `release.sh` already stamps a manifest-version assertion at bump time.

This closes the self-hosting + distribution gap: a fresh host session installs the version-gated plugin, the plugin's agents call the native `spacedock` binary for status + dispatch, and three independent checkpoints (front door, init/doctor, FO step-0) refuse to run a skill surface against an incompatible binary with an actionable, per-class remedy.

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

## Feedback Cycles

### Cycle 1 — ideation gate REJECT (3-lens staff audit, 2026-05-30)

Audit rejected — the contract-axis model + the injectable-seam mechanism are sound; this is reject-to-fix. Material findings + resolved forks the revision MUST address:

- **Cross-repo test surface (the structural defect).** The seam + `CONTRACT_VERSION` + `skill_text_test.go` live in spacedock-v1 (a Go module); the manifests AC-4/AC-5 assert (`.claude-plugin`/`.codex-plugin/plugin.json`, `marketplace.json`) live ONLY in `~/git/spacedock` (no go.mod, no `next` branch); no `requires-contract` exists yet. A Go bracketing test can't read a manifest in a non-Go sibling repo without a vendored fixture. Decide where each AC's test lands + how `CONTRACT_VERSION` (spacedock-v1) and `requires-contract` (spacedock) stay co-located. SPIKE this: actually run the OPEN-1 host install commands + attempt the cross-repo manifest read, ground the allocation in observed reality.
- **Split AC-5** → AC-5a (vendored FO/ensign call `spacedock status` + manifest `requires-contract` — testable now) / AC-5b (zero `claude-team` dispatch-path refs — closed by native-dispatch-helper, NOW in implementation).
- **Version-gate protocol holes:** enumerate `malformed-range` as a 5th doctor verdict (non-zero exit, parse-error, NO remedy); add a static test that the FO Startup step-0 EMBEDDED range brackets `CONTRACT_VERSION` (a 4th source of truth with no sync check today); define the front-door behavior when the installed plugin manifest is unresolvable; pin the literal per-class remedy strings or the derivation rule (host/branch-dependent); flag that the `--version` `contract N` token breaks the existing exact-match `TestVersion`; name `.codex-plugin/plugin.json` as the AUTHORITATIVE manifest edit point (`release.sh` regenerates the legacy `.claude-plugin` copy, so editing the legacy one desyncs).

**Captain-resolved forks:** Codex = **version-gate + documented prose** (FORCED — the repo's tool-mapping confirms Codex has no `--agent` analog; state plainly in AC-3/AC-4 so "done" does not overclaim Codex agent-launch). Sequencing = **dispatch-first** (implement after native-dispatch-helper). Contract-axis = integer `CONTRACT_VERSION` + half-open `requires-contract` range (SETTLED — do not reopen). Develop on the spacedock `next` branch. Every revised AC must name a behavioral exercise-and-observe oracle.

## Stage Report: ideation (cycle 2)

Spike-first revision. All cycle-1 findings + captain-resolved forks addressed; the contract-axis model and injectable-seam mechanism were NOT reopened (cycle-1 audit settled them).

- DONE: SPIKE the cross-repo test surface — run OPEN-1 host commands + attempt the cross-repo manifest read; ground allocation in observed reality.
  `### Spike findings`: confirmed spacedock-v1 has go.mod / `~/git/spacedock` does not + no `next` branch; `claude plugin validate` WARNS-but-passes on `requires-contract`; `claude plugin marketplace add <path>` + `install` run at exit 0 and `plugin list --json` reports a resolvable `installPath`; `.codex-plugin/plugin.json` is authoritative and `release.sh` full-copies it to the legacy `.claude-plugin`. Isolated `CLAUDE_CONFIG_DIR`; real `~/.claude` untouched; temp dirs removed.
- DONE: Resolve cross-repo test allocation + four-sources-of-truth co-location.
  `### Cross-repo test allocation`: per-surface test-home table; vendored `internal/contract/testdata/plugin.json` fixture for the Go bracketing test; `~/git/spacedock`-side packaging test guards fixture-vs-authoritative drift; legacy manifest auto-generated.
- DONE: Split AC-5 → AC-5a (testable now) / AC-5b (BLOCKED-by-`native-dispatch-helper`, dispatch-first sequencing).
  AC-5a/AC-5b are now separate ACs with distinct oracles; Dependencies section states the dispatch-first sequence as settled.
- DONE: Close version-gate holes — malformed-range 5th verdict, FO step-0 bracketing test, unresolvable-manifest front-door behavior, pinned per-class remedy strings, `TestVersion` break flagged, `.codex-plugin` named authoritative.
  AC-1 now enumerates five verdicts; the message block pins all four remedy literals + derivation rule; AC-2 adds the embedded-range bracketing test; "Where the range is read" defines the unresolvable-manifest front-door behavior + `--skip-contract-check`; Dependencies flags the `cli_test.go:35` exact-match break; OPEN-2 names `.codex-plugin/plugin.json` authoritative + the `sync_legacy_plugin_manifest` propagation.
- DONE: Codex scope = version-gate + documented prose (forced); behavioral oracle per AC.
  Spike confirmed no `codex --agent` analog; AC-3/AC-4 state the reduced Codex scope plainly and each AC names an exercise-and-observe oracle (run/drive-seam + observe), not a prose grep.

### Summary

The cycle-1 reject was structural, not conceptual: the test surface spans two repos and the doc had treated them as co-located. A behavioral spike on this machine resolved the three unknowns that blocked a sound allocation — Claude Code tolerates (warns, ignores) an unknown `requires-contract` field; the installed manifest is resolvable via `claude plugin list --json` → `installPath`; and `.codex-plugin/plugin.json` is the authoritative manifest that `release.sh` auto-copies to the legacy Claude path. The allocation now lands each AC's test where its subject lives, vendors a manifest fixture into spacedock-v1 for the Go bracketing test, and pushes fixture-vs-real drift detection into a `~/git/spacedock`-side packaging test. AC-5 is split (5a testable now, 5b blocked on `native-dispatch-helper`, dispatch-first), all five doctor verdicts (incl. malformed-range) and the four-source-of-truth sync are enumerated with pinned oracle strings, the `TestVersion` break is flagged for the implementer, and Codex's no-`--agent` reality is stated plainly so "done" cannot overclaim. No production code written (ideation); frontmatter and the FO-owned Feedback Cycles section untouched.

## Stage Report: implementation

- DONE: Contract-version core (new internal/contract pkg): CONTRACT_VERSION constant; `--version` emits the `contract N` token (and UPDATE the exact-match TestVersion in the same commit); range parse + numeric bracket comparison; `spacedock doctor [--plugin-manifest PATH]` with ALL FIVE verdicts table-driven over fixtures in internal/contract/testdata/ (incl. a VENDORED manifest fixture plugin.json), each asserting exit code + the pinned remedy substring. Riskiest path — validate first.
  `internal/contract/{contract,doctor}.go`: CONTRACT_VERSION=1, ParseRange (half-open `>=N,<M`), five-verdict Compare with pinned remedy literals, RunDoctor. 22 contract unit tests + `TestDoctorVerdicts` table over compatible/too-old-binary/too-old-plugin/malformed-range/no-plugin-found fixtures (each asserts exit + pinned stderr/stdout substring). `--version` token + TestVersion update in commit 5be7e3b. Validated first; behaviorally exercised the binary across all five verdicts with real exit codes (0/1/1/1/0). Commits 92b5b89, 5be7e3b.
- DONE: The 3-point gate via injectable seams (status.Runner pattern): `spacedock claude` front-door (version-gate, fail-fast, then launch seam `claude --agent spacedock:first-officer`; `--skip-contract-check` bootstrap override); `spacedock codex` = version-gate + documented-prose only; `spacedock init --host claude` via an install seam + ONE isolated-CLAUDE_CONFIG_DIR behavioral install observing `claude plugin list --json` installPath + ~/.claude/skills untouched; FO Startup step-0 — edit the VENDORED FO Startup to gate `spacedock --version` vs the embedded range and abort, + a Go test that the embedded range brackets CONTRACT_VERSION + an abort-path behavior fixture.
  `internal/cli/{frontdoor,init,host_exec}.go`: `hostOps` seam (ResolveManifest/Launch/Install), runClaude/runCodex/runInit/runDoctor wired into the router. Front-door fail-fast on mismatch AND unresolvable-manifest (no launch); `--skip-contract-check` bypass. `TestClaudePluginInstallIsHostNative` runs a REAL `claude plugin marketplace add`/`install` against an isolated CLAUDE_CONFIG_DIR+cache (1.13s, not skipped), observes installPath + untouched skills/. FO Startup step-1 contract gate added (embedded range `>=1,<2`, aborts before --discover/--boot); `TestStartupStepZeroIsContractGate` (ordering), `TestStartupEmbeddedRangeBracketsContractVersion`, and `TestStartupGateAbortsBeforeDiscover` (real spacedock stub whose --version prints an out-of-range contract → pinned abort + zero discover calls). The contract comparison + doctor verdicts were the riskiest path and were validated first. Commits 5be7e3b, 18dc164, 83915b0.
- DONE: AC-5a (spacedock-v1 half): static test the vendored FO/ensign contracts call `spacedock status` (zero `skills/commission/bin/status`); the vendored manifest fixture's requires-contract parses + brackets CONTRACT_VERSION. Confirm AC-5b is ALREADY green via the merged native-dispatch-helper's TestDispatchBlockUsesNativeBuild. Then go test ./... + -race, gofmt -l, go vet with REAL captured exit codes.
  `skills/integration/contract_status_path_test.go` (FO calls `spacedock status`, zero plugin-private status path in FO+ensign); `TestVendoredFixtureBracketsContractVersion` in internal/contract closes the fixture-vs-binary half. `TestDispatchBlockUsesNativeBuild` (AC-5b) confirmed green, NOT re-implemented. Gates with captured exit codes: `go test ./...` exit 0 (389 passed), `go test -race ./...` exit 0 (389 passed), `gofmt -l .` exit 0 (0 files), `go vet ./...` exit 0. Commit 1ddd468.

### Summary

Implemented the v1 distribution model's compatibility spine on the spacedock-v1 side. New `internal/contract` package holds CONTRACT_VERSION=1, the half-open `requires-contract` range grammar, the five-verdict compare with pinned per-class remedy strings, and `spacedock doctor`; the riskiest path (compare + doctor verdicts) was built and validated first. `--version` now carries the `contract N` token (TestVersion updated in the same commit). The 3-point gate runs through an injectable `hostOps` seam mirroring `status.Runner`: `spacedock claude` version-gates then launches `claude --agent spacedock:first-officer` (fail-fast on mismatch/unresolvable, `--skip-contract-check` bootstrap), `spacedock codex` is version-gate + documented prose (no `--agent` analog), `spacedock init` installs the per-host plugin via the host mechanism (no skill-file copies), and the vendored FO Startup gains a step-0 contract gate with embedded range `>=1,<2`. Every AC names a behavioral exercise-and-observe oracle: doctor verdicts over real fixtures with real exit codes, a real isolated `claude plugin install` (AC-4), and a real `spacedock` stub driving the abort path (AC-2). AC-5b is confirmed already green via the merged native-dispatch-helper. CAPTAIN-SCOPED `~/git/spacedock` edits (authoritative manifest `requires-contract`, cross-repo drift test, @next live-install) are DEFERRED to a release-time follow-up per dispatch and untouched. All gates green with captured exit codes; code committed to worktree branch spacedock-ensign/spacedock-packaging.
