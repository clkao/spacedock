---
id: h6e4z6p3jksb4s609n2q3t3k
title: "Codex marketplace should point directly at Spacedock plugin root"
status: implementation
source: "2026-05-06 captain request: replace plugins/spacedock self-symlink with direct root marketplace source and verify through isolated Codex"
started: 2026-05-06T08:45:06Z
completed:
verdict:
score: 0.8
worktree: .worktrees/spacedock-ensign-codex-marketplace-root-source
---

Spacedock's repo-local Codex marketplace currently points at `./plugins/spacedock`, a checked-in symlink back to `..`. That keeps the repo root as the plugin package root while satisfying the common `./plugins/<name>` marketplace convention, but it creates a self-reference that copy/staging code has to special-case.

The captain selected the cleaner approach: make the marketplace source point directly at the repo/plugin root (`.` or `./`) and remove the `plugins/spacedock` symlink if Codex accepts that source path.

## Proposed approach

Update the authoritative Codex marketplace entry and legacy mirror so `spacedock` resolves directly to the repo root. Remove the symlink-specific packaging contract and replace it with a path-resolution contract: the marketplace source must resolve to the repository root containing `.codex-plugin/plugin.json`.

Update user-facing docs and contract tests that describe the old symlink invariant. Historical plans do not need rewriting unless an active test or generated contract treats them as current source of truth.

Verification must include a Codex-side smoke test against the implementation worktree as the plugin source. Use a temporary `CODEX_HOME`, run the Codex app-server/plugin APIs against the worktree marketplace, install `spacedock`, then run `codex debug prompt-input` with that same `CODEX_HOME` and confirm the `spacedock:*` skills are loaded from the temporary plugin install/cache.

## Acceptance criteria

**AC-1 -- Marketplace source resolves directly to the plugin root.**
Verified by: `unset CLAUDECODE && uv run pytest tests/test_codex_plugin_packaging.py -v` asserting `.agents/plugins/marketplace.json` and `.claude-plugin/marketplace.json` use the direct root source and that the source resolves to a directory containing `.codex-plugin/plugin.json`.

**AC-2 -- The self-symlink is no longer part of the packaging contract.**
Verified by: `test ! -e plugins/spacedock` and by the packaging test no longer requiring `plugins/spacedock` to be a symlink.

**AC-3 -- Codex accepts the implementation worktree as an isolated repo-local plugin source.**
Verified by: a temporary `CODEX_HOME` Codex app-server/plugin smoke test that runs `plugin/list` with `cwds` set to the worktree path, confirms the `spacedock` source path is the worktree root, runs `plugin/install`, then runs `codex debug prompt-input --enable multi_agent` from the worktree and confirms `spacedock:commission`, `spacedock:first-officer`, `spacedock:ensign`, `spacedock:refit`, and `spacedock:debrief` appear from the temporary Codex plugin installation.

**AC-4 -- Documentation and compatibility mirrors stay aligned.**
Verified by: packaging tests confirming `.claude-plugin/marketplace.json` remains synchronized with `.agents/plugins/marketplace.json`, and by README text describing the direct-root marketplace source rather than the symlink.

## Test plan

Run the focused static contract:

```bash
unset CLAUDECODE && uv run pytest tests/test_codex_plugin_packaging.py -v
```

Run the broader offline static suite if the focused change passes:

```bash
unset CLAUDECODE && make test-static
```

Run an isolated Codex smoke test from the implementation worktree. It should not rely on the user's existing plugin installation; only copy `auth.json` if the Codex command requires authentication to start. The observable evidence is app-server `plugin/list` and `plugin/install` success plus `codex debug prompt-input` showing the Spacedock skills from the temporary `CODEX_HOME`.
