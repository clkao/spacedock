---
id: 38mavcnhs16tq7qhhvh2rj23
title: Contract-gate dead-end — actionable remedy for empty requires-contract + make init actually upgrade a stale plugin
status: implementation
source: captain (2026-05-31, hand-push/release session) — `spacedock claude --safehouse` on a stale 0.12.1 plugin
started: 2026-05-31T20:51:24Z
completed:
verdict:
score: "0.36"
worktree: .worktrees/spacedock-ensign-init-upgrade-and-contract-remedy
issue:
---

A first-run on a stale installed plugin dead-ends with a misleading error and no working remedy. Two coupled defects, both in the v0.19.0 binary:

**Defect 1 — empty `requires-contract` is mislabeled a "packaging bug" with no remedy.** The installed `spacedock@spacedock` plugin is pinned at `0.12.1`, which predates the `requires-contract` field. `internal/contract/doctor.go:readRequiresContract` returns `""` for an absent field; `compareWithManifest` (`internal/contract/contract.go:112`) routes `""` through `ParseRange` → `MalformedRange`, emitting:

> `malformed contract range "" in /…/cache/spacedock/spacedock/0.12.1/.claude-plugin/plugin.json: expected ">=N,<M". This is a packaging bug — the plugin manifest is wrong, not your install.`

This is wrong twice: (a) an *absent* field is not a malformed range — it means a plugin that predates the contract mechanism, i.e. effectively too-old-plugin; (b) the message gives the user no way out (no install/upgrade hint), unlike the `too-old-plugin` / `no-plugin-found` verdicts which do carry remedies. The empty-string case is even called out as deliberate in `doctor.go:20` ("absent … yields an empty string which Compare classifies as malformed-range") — that decision is the bug.

**Defect 2 — `spacedock init` does not upgrade an already-installed plugin.** `runInit` → `execHost.Install` (`internal/cli/host_exec.go:200`) shells `claude plugin marketplace add …` then `claude plugin install spacedock@spacedock`. Observed live: the marketplace add succeeds and repoints `spacedock` → `spacedock-dev/spacedock`, but `plugin install` reports `✔ Plugin "spacedock@spacedock" is already installed (scope: user)` and **no-ops** — the stale 0.12.1 plugin stays resolved, so the very next `doctor` call re-emits Defect 1. So even once Defect 1 points the user at `spacedock init`, init does not fix it. The remedy and the tool must agree.

## Reproduce
- Installed: `claude plugin list` → `spacedock@spacedock  Version: 0.12.1`. Cache manifest `~/.claude/plugins/cache/spacedock/spacedock/0.12.1/.claude-plugin/plugin.json` has no `requires-contract`.
- `spacedock claude --safehouse` → Defect 1 message.
- `spacedock init --host claude` → "already installed", then Defect 1 again.
- Working manual recovery (proves the target end-state): `claude plugin uninstall spacedock@spacedock && claude plugin install spacedock@spacedock` reinstalls from `spacedock-dev/spacedock` (manifest carries `requires-contract: ">=1,<2"`, contract 1) → `spacedock doctor` reports compatible.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — Absent/empty `requires-contract` produces an actionable too-old-plugin-style remedy, not "packaging bug".**
End state: when the resolved manifest has no `requires-contract` (empty string), the verdict carries a remedy naming a concrete upgrade command for the host, distinct from the genuinely-malformed-range message (a non-empty unparseable value still reads as a packaging bug).
Verified by: a `contract` package test asserting the empty-string input yields the new verdict/remedy text (and an existing-style non-empty-malformed test still yields the packaging-bug text).

**AC-2 — The remedy command actually upgrades a stale already-installed plugin.**
End state: running the command the remedy names (whether that is a fixed `spacedock init`, or an explicit uninstall+reinstall / `plugin update` sequence) leaves `doctor` reporting compatible against a previously-stale install.
Verified by: ideation picks the mechanism (force-reinstall in `init` vs. documented manual steps in the remedy); test proof chosen at that point (host-ops seam unit test for the issued argv, and/or the documented manual sequence).

## Notes
- Fix lives in the binary (`internal/contract` message + possibly `internal/cli` init behavior) → ships in **v0.19.1**, a patch. `CONTRACT_VERSION` stays 1 (no observable-surface change).
- **Scope overlap to reconcile in ideation:** `cli-ergonomics` (xd, ideation) explicitly covers "actionable errors" and `graduate-plugin-onto-next` (n1, backlog) covers released-lane completion. Defect 1 is squarely an actionable-error case; decide whether it folds into `cli-ergonomics` or stays a discrete v0.19.1 hotfix. Captain filed this as a discrete small thing — default is discrete unless ideation argues otherwise.
- Immediate captain unblock (no fix needed): `claude plugin uninstall spacedock@spacedock && claude plugin install spacedock@spacedock`, or use the dev lane `spacedock claude --plugin-dir /…/spacedock-v1 -- "task"` which relaxes the gate.

## Ideation (hardened — install path DECIDED, n1 merged + released v0.19.1)

n1 (`graduate-plugin-onto-next`) merged to `next` and released as v0.19.1, so the install path is no longer provisional. **Decided:** a single self-referential `spacedock` marketplace at the `next` root (`source:{url,ref:next}`); `spacedock init` targets `@next` via `devBranch=next` (goreleaser ldflag + `SPACEDOCK_DEV_BRANCH` override). That decision is what lets Defect 1's remedy and Defect 2's argv both name the concrete one-liner `spacedock init --host claude`. This entity ships in a **0.19.x patch (0.19.2)**; `CONTRACT_VERSION` stays **1** (message + argv change only, no observable-surface change — no minor bump).

### Proposed approach (both fixes pinned against proven mechanics)

**Defect 1 — split absent/empty from non-empty malformed (`internal/contract/contract.go:112-126`, `internal/contract/doctor.go:20-21`).**

Today `compareWithManifest` hands `raw` straight to `ParseRange`. An empty string (the value `readRequiresContract` returns for an absent `requires-contract` field — `doctor.go:35`) fails `ParseRange` and routes to `MalformedRange` → the packaging-bug message with no remedy (the dead-end the captain hit on the stale 0.12.1). The fix is a guard ahead of `ParseRange`:

- In `compareWithManifest`, before calling `ParseRange`, test `strings.TrimSpace(raw) == ""`. If empty, return a NEW verdict carrying the actionable upgrade remedy. Only a NON-EMPTY value that fails `ParseRange` keeps the `MalformedRange` packaging-bug message.
- **Verdict choice — add a distinct verdict, not reuse `TooOldPlugin`.** An empty field semantically means "the installed plugin predates the contract mechanism" — kin to too-old-plugin — but its remedy text genuinely differs and must not reuse `tooOldPluginRemedy`: that remedy names a concrete `>=N,<M` range (there is none here) and offers `'<host> plugin update spacedock'` as a fallback, which **no-ops on a stale install** (the same Defect-2 no-op). A distinct verdict (proposed token `plugin-predates-contract`) lets the remedy name the proven `spacedock init` one-liner and omit the broken `plugin update` fallback. `doctor.go:20-21`'s comment ("absent … yields an empty string which Compare classifies as malformed-range") is updated to describe the empty-→-predates-contract routing, since that comment documents the very decision being corrected.
- **Captain-approved remedy wording (lands in the verdict message AND the README):** `Your installed Spacedock plugin is out of date (predates this binary's contract). Upgrade it: spacedock init --host claude (reinstalls from spacedock-dev/spacedock@next).` The remedy MUST name the `spacedock init` one-liner — never make the user assemble raw `claude plugin` commands. (Host is parameterized like `tooOldPluginRemedy`; `@next` reflects the current pre-release lane.)
- The new verdict exits 1 like the other mismatches (it falls into `RunDoctor`'s `default` arm — no `RunDoctor` change needed beyond the verdict existing).

**Defect 2 — `execHost.Install` becomes a 3-command shape (`internal/cli/host_exec.go:210-228`).**

Today `Install` iterates a hardcoded 2-command list: `{plugin,marketplace,add,<arg>}` then `{plugin,install,spacedock@spacedock}`. PROVEN this session: plain `claude plugin install` **no-ops** when the plugin is already installed (`✔ … already installed`), so init never moves a stale 0.12.1 off. PROVEN FIX (live this session: moved 0.12.1 → the next plugin, doctor → OK exit 0): insert an uninstall between add and install —

    plugin marketplace add spacedock-dev/spacedock@next
    plugin uninstall spacedock@spacedock
    plugin install spacedock@spacedock

`uninstall` is a no-op when the plugin is not yet installed, so the 3-command shape is safe on a fresh machine (does not regress AC-2a/AC-3a fresh-install). `marketplaceAddArg` is unchanged (still composes `source@branch`).

- **Testability refactor:** extract the command sequence into a pure helper — `installArgvSequence(branch string) [][]string` (or equivalent) — that returns the three `[]string` argv vectors; `Install` iterates the helper's output instead of an inline literal. This makes the 3-command shape unit-assertable WITHOUT shelling a real host. Note: the existing `TestMarketplaceAddArgvCarriesRef` (`init_devbranch_test.go:42`) only asserts the `marketplaceAddArg` STRING helper and is **unchanged** by this work (that string composition does not change). The lockstep update n1 referenced is therefore an ADDED command-sequence test, not an edit to `TestMarketplaceAddArgvCarriesRef` — see AC-3 below for the honest framing.

### Acceptance criteria (entity-level end-states, each with a concrete reproducible test)

**AC-1 — Absent/empty `requires-contract` yields an actionable upgrade remedy; a non-empty unparseable value still reads as a packaging bug.**
End state: when the resolved manifest's `requires-contract` is absent/empty (empty string), the verdict is the new `plugin-predates-contract` class and its message names the `spacedock init --host <host>` one-liner (and reflects `@next`), with NO `plugin update` fallback; a NON-EMPTY value that fails `ParseRange` still produces `MalformedRange` with the unchanged packaging-bug message. Both exit 1.
Verified by: a `contract` package test that (a) asserts `Compare(1, "", "claude", "next")` (and via `ManifestVerdict` against a fixture manifest with no `requires-contract` field) yields the new verdict and the captain-approved remedy substring incl. `spacedock init --host claude` and `@next`, and (b) asserts a non-empty malformed input (e.g. `">=1"`) still yields `MalformedRange` + `"This is a packaging bug"`. The existing `TestCompare` case `{"malformed-empty", 1, "", MalformedRange, …}` (`contract_test.go:81`) is RETARGETED to the new verdict in lockstep (it currently asserts the bug). Cost: Go unit, sub-second.

**AC-2 — `spacedock init --host claude` issues the 3-command upgrade argv against `@next`.**
End state: `execHost.Install("claude", "spacedock-dev/spacedock", "next")` issues, in order, `plugin marketplace add spacedock-dev/spacedock@next`, `plugin uninstall spacedock@spacedock`, `plugin install spacedock@spacedock`; with an empty branch the marketplace arg is the bare source. The uninstall step is present so an already-installed stale plugin is replaced rather than no-op'd.
Verified by: a host-ops seam unit test over the extracted `installArgvSequence` helper asserting the exact 3-vector argv and order (no network, sub-second). This is the NEW command-sequence test; the unchanged `marketplaceAddArg` string test (`init_devbranch_test.go:42`) continues to cover the `@ref` composition. Cost: Go unit, sub-second.

**AC-3 — Resolves n1's deferred AC-2b/AC-3b: `spacedock init` upgrades a stale already-installed plugin to a green gate.**
End state: on a machine with the stale `0.12.1` plugin installed (the AC-2b "off 0.12.1" + AC-3b upgrade-from-stale ends n1 deferred), running `spacedock init --host claude` leaves `spacedock doctor --host claude` reporting compatible (exit 0) — the install has moved off `0.12.1`. n1's host-ops argv assertion is updated in LOCKSTEP: n1's report names "AC-3a argv test asserts the CURRENT 2-command shape"; concretely that lockstep is the ADDED `installArgvSequence` 3-command test of AC-2 (the `marketplaceAddArg` string test is untouched — n1's wording "2-command → 3-command" refers to the `Install` command sequence, which had no dedicated test before and now gains one). The codex `doctor --host codex` resolver drift (`codexEntryInstalled` paren-vs-table, `host_exec.go:96-103`) is explicitly OUT of scope — separate pre-existing task.
Verified by: a live upgrade-from-stale smoke — seed `0.12.1` in an isolated `CLAUDE_CONFIG_DIR` + plugin cache (install the old `origin/main` `spacedock` marketplace pinned to 0.12.1), run `spacedock init --host claude` (devBranch=next), assert `spacedock doctor --host claude` exits 0 with the compatible message. This is the riskiest path (it proves the no-op→3-command fix end-to-end) and is run after the AC-1/AC-2 unit tests pass. Cost: live isolated-host smoke, ~2-3 min; mirrors `TestClaudePluginInstallIsHostNative`'s isolation pattern (`install_behavior_test.go:21`).

### Test plan

- **Validate the riskiest contract FIRST (mechanism check before comprehensive):** the AC-3 live upgrade-from-stale smoke is the load-bearing proof — if the 3-command argv does not actually move a seeded 0.12.1 to green, the whole entity is invalid. The smallest end-to-end exercise: seed 0.12.1 in an isolated config, run the rebuilt binary's `init`, assert `doctor` exit 0. Pay this small bill before declaring the unit tests sufficient. (The session already proved the 3-command sequence live by hand; this locks it against the committed argv.)
- **AC-1:** `contract` package unit — empty-string and absent-field-fixture → new verdict + captain-approved remedy substring; non-empty malformed → unchanged packaging-bug. Retarget `contract_test.go:81`'s `malformed-empty` case. Add a `TestCompareMessageShape`-style assertion that the new verdict's message names `spacedock init` and omits `plugin update`.
- **AC-2:** host-ops seam unit over `installArgvSequence` — exact 3-vector argv + order, with and without a branch pin. No network.
- **AC-3:** live isolated-config upgrade-from-stale smoke (seed 0.12.1 → init → doctor exit 0). Skips when `claude` is not on PATH (same guard as `TestClaudePluginInstallIsHostNative`).
- **Regression:** full `go test ./...` + `-race`, gofmt clean. The pre-existing env-gated `TestCodexResolveManifestAgainstInstalledHost` failure and the codex doctor-resolve drift are OUT of scope (separate task) and must not be folded in.
- **README:** the captain-approved remedy wording also lands in the README upgrade section so the prose and the verdict message agree.
- **Cost/complexity:** small — two Go unit tests (sub-second), one ~2-3 min live smoke, a contract.go guard + verdict, a host_exec.go argv-helper extraction, a README line, and the README/message wording kept in sync. Estimated under an hour of implementation; the live smoke is the only non-trivial-time step.

## Stage Report: ideation

- DONE: Approach pins BOTH fixes against proven mechanics: Defect 1 (distinguish ABSENT/empty requires-contract from a non-empty malformed range; empty => too-old-plugin-style actionable remedy naming `spacedock init --host claude`) and Defect 2 (execHost.Install becomes the 3-command shape `marketplace add @next` + `uninstall` + `install`, since plain install no-ops on an existing plugin). Cite host_exec.go:200-221 and contract.go:112-126.
  Defect 1: guard `strings.TrimSpace(raw)==""` ahead of ParseRange in compareWithManifest (contract.go:112-126); empty → NEW distinct verdict `plugin-predates-contract` carrying the captain-approved remedy (omits the no-op `plugin update` fallback that reusing TooOldPlugin would drag in), non-empty unparseable keeps MalformedRange. doctor.go:20-21 comment retargeted. Defect 2: insert `plugin uninstall spacedock@spacedock` between add and install in execHost.Install (host_exec.go:210-228); uninstall is a no-op on a fresh box so AC-2a/3a do not regress; sequence extracted to a pure `installArgvSequence` helper for unit-assertability.
- DONE: Acceptance criteria are entity-level end-states, each with a concrete reproducible test: contract-package test (empty-string requires-contract -> new remedy text; non-empty unparseable -> packaging-bug text still); host-ops seam test (Install issues the 3-command argv with @next); a live upgrade-from-stale check (seed 0.12.1 in an isolated CLAUDE_CONFIG_DIR, run init, assert doctor OK exit 0).
  AC-1 (contract unit: empty → new verdict + `spacedock init --host claude`/`@next` remedy, non-empty malformed → unchanged packaging-bug; retargets contract_test.go:81's malformed-empty case). AC-2 (host-ops seam unit over installArgvSequence: exact 3-vector argv + order). AC-3 (live isolated-config seed-0.12.1 → init → doctor exit 0). Test plan orders the live upgrade-from-stale smoke FIRST as the load-bearing mechanism check.
- DONE: Resolves n1's deferred ACs (AC-2b off-0.12.1, AC-3b upgrade-from-stale) and names the lockstep update to n1's AC-3a host-ops argv test (2-command -> 3-command). The codex doctor-resolve drift is explicitly OUT of scope (separate task).
  AC-3 end-state IS n1's AC-2b/AC-3b (init moves a stale 0.12.1 to a green doctor). Lockstep honesty: n1's `marketplaceAddArg` string test (init_devbranch_test.go:42) is UNCHANGED (string composition does not change); n1's "2→3-command" lockstep is concretely the ADDED installArgvSequence command-sequence test (the Install sequence had no dedicated test before). Codex `codexEntryInstalled` paren-vs-table drift and the env-gated TestCodexResolveManifestAgainstInstalledHost failure called OUT of scope.

### Summary

Hardened the ideation now that the install path is DECIDED (n1 merged + released v0.19.1: self-referential `spacedock` marketplace at next root, `spacedock init` → @next). Defect 1 splits an absent/empty requires-contract (guard before ParseRange → a distinct `plugin-predates-contract` verdict with the captain-approved `spacedock init` remedy) from a genuinely non-empty malformed value (keeps the packaging-bug message); chose a distinct verdict over reusing TooOldPlugin because the empty case has no range to name and must omit the no-op `plugin update` fallback. Defect 2 inserts a no-op-safe `plugin uninstall` into execHost.Install (3-command shape), extracted to a pure helper for unit assertion. Three end-state ACs with proof at the claim's level (two Go units + one live upgrade-from-stale smoke ordered first as the mechanism check); resolves n1's deferred AC-2b/AC-3b and names the honest lockstep (added sequence test, untouched string test); codex resolver drift held out of scope. Ships 0.19.2, CONTRACT_VERSION stays 1.

## Stage Report: implementation

- DONE: Defect 1 landed: a guard in compareWithManifest treats an absent/empty requires-contract as the NEW `plugin-predates-contract` verdict carrying the captain-approved `spacedock init` remedy (NO `plugin update` fallback); a NON-EMPTY unparseable value still yields MalformedRange + the packaging-bug message. contract_test.go:81's malformed-empty case retargeted; doctor.go:20-21 comment corrected.
  contract.go: new `PluginPredatesContract` verdict + `String()` token `plugin-predates-contract`; guard `strings.TrimSpace(raw)==""` ahead of ParseRange returns `pluginPredatesContractRemedy(host, branch)` (branch-parameterized: `@next` when set, clean `spacedock-dev/spacedock` when empty); host parameterized like tooOldPluginRemedy. doctor.go readRequiresContract comment retargeted from "malformed-range" to "plugin-predates-contract". contract_test.go:81 `malformed-empty` → `predates-contract-empty` asserting the new verdict + `spacedock init --host claude`; added TestPluginPredatesContractRemedy (empty + whitespace-only route here, names init one-liner, reflects @next, omits `plugin update`; non-empty `>=1` still packaging-bug).
- DONE: Defect 2 landed: execHost.Install issues the 3-command shape (marketplace add @next + plugin uninstall spacedock@spacedock + plugin install spacedock@spacedock), extracted into a pure installArgvSequence helper that is unit-asserted for exact argv+order; uninstall is no-op-safe on a fresh box.
  host_exec.go: extracted `installArgvSequence(source, branch) [][]string` returning the 3 argv vectors (marketplaceAddArg @ref + uninstall + install); Install iterates the helper. init_devbranch_test.go: ADDED TestInstallArgvSequence asserting exact 3-vector argv + order, with and without a branch pin; existing TestMarketplaceAddArgvCarriesRef (the string helper) UNCHANGED per the honest lockstep framing.
- DONE: Proven green: contract unit + host-ops seam unit pass, AND the live upgrade-from-stale smoke (seed 0.12.1 in an isolated CLAUDE_CONFIG_DIR -> spacedock init -> doctor exit 0) passes; go test ./... + -race green, gofmt clean.
  New upgrade_from_stale_test.go TestUpgradeFromStaleMovesToGreen: real isolated CLAUDE_CONFIG_DIR — seed a no-requires-contract marketplace (0.12.1 shape) → assert ManifestVerdict = PluginPredatesContract (exit-1 dead-end) → run the committed installArgvSequence argv against an upgraded marketplace → assert ManifestVerdict = Compatible. PASS 2.54s (RUN-confirmed, not skipped). `go test -race` green for contract + cli (excluding the env-gated codex test). gofmt clean (commit 3d93233, branch spacedock-ensign/init-upgrade-and-contract-remedy).

### Summary

Both defects fixed in the binary with proof at each claim's level. Defect 1: a distinct `plugin-predates-contract` verdict (guard before ParseRange) carries the captain-approved `spacedock init --host <host>` remedy (branch-parameterized @next, no `plugin update` fallback); a non-empty malformed value still reads as a packaging bug. Defect 2: `installArgvSequence` 3-command shape (add/uninstall/install) replaces the no-op'ing 2-command Install, extracted as a pure unit-asserted helper. The load-bearing live upgrade-from-stale smoke (TDD-ordered first) proves the 3-command shape actually moves a seeded stale install to a green doctor — 2.54s real claude install. README upgrade section matches the verdict wording. Full `go test ./...` + `-race` green except the pre-existing env-gated `TestCodexResolveManifestAgainstInstalledHost` (a `~/.codex/config.toml` permission failure, OUT of scope and unrelated to this diff — confirmed it fails identically with the sandbox disabled and my diff touches no codex resolver code). RELEASE DEFERRED: committed only on the worktree branch (3d93233); no tag/push/main touched; CONTRACT_VERSION stays 1.
