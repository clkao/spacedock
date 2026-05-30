---
id: tq66yjc7sqbhyc52eg8h2ecx
title: Spacedock packaging and distribution
status: backlog
source: handoff self-hosting gap
score: "0.35"
worktree:
---

Define and build the v1 distribution model so a fresh Claude Code or Codex session runs the Spacedock first officer from the repo's own native binary + a version-gated plugin — no Python in the dispatch path, and no contract files copied into per-agent skill folders.

Model (decided with the captain):
- The `spacedock` binary is the canonical artifact: `status` + `dispatch` (native, via `native-dispatch-helper`) + host front-ends `spacedock claude` / `spacedock codex` + `init` / `doctor`. It exposes a CONTRACT-VERSION in `--version`.
- Per-host PLUGINS (Claude Code AND Codex) register the stable agents `spacedock:first-officer` / `spacedock:ensign` (named `--agent`/subagent_type entry points) whose contracts call `spacedock status` / `spacedock dispatch`. The plugin IS the amended skill surface (already vendored under `skills/`, zero plugin-private-path refs), published per host. `spacedock init` USES the host plugin mechanism (installs the plugin) — it does NOT copy contract files into `~/.claude/skills`.
- `spacedock claude` wraps `claude --agent spacedock:first-officer …`; `spacedock codex` the Codex analog. One front door, host-native underneath.
- VERSION GATE on a CONTRACT-VERSION axis (not raw semver — bump only when the binary's flag surface / observable behavior the skill depends on changes). The plugin manifest declares `requires-contract: ">=N,<N+1"`; checked at three points: `spacedock claude` (front door, fail fast), `spacedock init`/`doctor` (install/upgrade), and FO Startup step 0 (per-session safety net). Mismatch -> actionable upgrade message. (Mirrors roborev's daemon/cli version-mismatch detection.)

## Acceptance criteria

**AC-1 - `spacedock --version` reports a contract-version and `spacedock doctor` reports skill<->binary compatibility.**
Verified by: `spacedock --version` includes a contract-version token; `spacedock doctor` (or `init --check`) compares the installed plugin's declared contract range to the binary and prints OK or an actionable mismatch; tests for compatible / too-old-binary / too-old-plugin.

**AC-2 - The FO Startup procedure version-gates as step 0.**
Verified by: the vendored FO contract's Startup begins by checking `spacedock --version` against its declared contract range and aborting with an actionable message on mismatch; a fixture exercises the abort path.

**AC-3 - `spacedock claude` / `spacedock codex` launch the stable plugin-registered agent, version-gated.**
Verified by: `spacedock claude` resolves to launching `claude --agent spacedock:first-officer` (the plugin agent) after a passing version check; the Codex path documented; the front-end fails fast on version mismatch before launching.

**AC-4 - `spacedock init` installs the per-host plugin via the host's plugin mechanism, with no per-agent skill-file copies.**
Verified by: `spacedock init` for Claude Code installs the plugin so `Skill()` / `--agent spacedock:first-officer` work; no files written into `~/.claude/skills` outside the plugin's own install; the Codex path documented (and implemented if the Codex plugin mechanism supports it).

**AC-5 - The published plugin is the amended skill surface (calls `spacedock`, zero plugin-private-path refs) with a declared contract-version.**
Verified by: the plugin's FO/ensign agent contracts call `spacedock status` / `spacedock dispatch`; static check for zero `skills/commission/bin/status` refs; the plugin manifest declares its contract range.

## Test gates

- `go test ./...`
- `spacedock --version` contract-version + `spacedock doctor` compat (compatible / too-old-binary / too-old-plugin).
- FO Startup version-gate abort-path test.
- Static: plugin contracts call `spacedock`; zero plugin-private-path refs.

## Notes

Depends on `native-dispatch-helper` (one binary for status + dispatch). Develop on the spacedock repo's `next` branch (copy/adapt the skill surface from `~/git/spacedock`), flip to main when ready. OPEN for ideation: (1) how Claude Code AND Codex plugins install from a specific git BRANCH (e.g. `next`) for pre-release testing — research marketplace/branch-ref support and document the exact install command; (2) who owns the contract-version bump discipline (the rule for when a binary change forces a contract bump + a matching plugin release). Beyond the bootstrap roadmap — this closes the self-hosting + distribution gap.
