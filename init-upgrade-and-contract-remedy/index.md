---
id: 38mavcnhs16tq7qhhvh2rj23
title: Contract-gate dead-end — actionable remedy for empty requires-contract + make init actually upgrade a stale plugin
status: backlog
source: captain (2026-05-31, hand-push/release session) — `spacedock claude --safehouse` on a stale 0.12.1 plugin
started:
completed:
verdict:
score: "0.36"
worktree:
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
