---
id: 44cpt70xq06sfzjy46dtp8ey
title: Bundle the plugin into the binary and inject --plugin-dir (zero-config Claude install)
status: backlog
source: captain (2026-05-31) — "bundle the plugin distribution so we just inject --plugin-dir, no marketplace config"
started:
completed:
verdict:
score: "0.30"
worktree:
issue:
---

Idea: the `spacedock` binary **embeds the plugin** (`skills/`, `agents/`, `.claude-plugin/`) via Go `//go:embed`, extracts it to a version-keyed cache dir on `spacedock claude`, and launches `claude --plugin-dir <that>`. Then a Claude user needs **no marketplace, no `claude plugin install`, no `install` step** — `brew install spacedock` → `spacedock claude` just works, the binary and plugin are always version-coherent (same artifact), and the stale-plugin / 0.12.1-collision problem disappears (you can't have a stale plugin when it's embedded).

This is a **distribution-model** decision that partly reframes n1's marketplace lane: bundling becomes the *primary Claude* lane; the marketplace lane stays for **codex** (+ an opt-in "let Claude's registry manage it" lane).

## Spike findings (this session)

- **Claude supports it:** `--plugin-dir <path>` "Load a plugin from a directory or .zip" (Claude Code 2.1.154). Go 1.26 has `embed`.
- **Precedence — PROVEN (no-LLM):** with the real `spacedock@spacedock` 0.19.1 installed AND `--plugin-dir <inline 9.9.9>`, `claude --plugin-dir … plugin details spacedock` resolved to `spacedock 9.9.9 · Source: spacedock@inline`. **The inline `--plugin-dir` plugin overrides the installed one** — so injecting `--plugin-dir <bundled>` overrides a stale 0.12.1. (This also explains why this session's FO loaded from 0.12.1: that session had no *effective* `--plugin-dir`, not that inline lost.)
- **Behavioral agent-resolution confirmation — PENDING (auth-blocked):** the `-p` sentinel test could not run in a scratch `CLAUDE_CONFIG_DIR` (`Not logged in`). Runnable artifact handed to the captain (a sentinel `--plugin-dir` plugin + `claude --agent spacedock:first-officer -p` one-liner) to confirm the agent — not just the plugin listing — resolves inline.

## Open research — the codex gap (BLOCKS making this the universal lane)

**Codex has no `--plugin-dir` equivalent** (only a `plugin` subcommand for managed install; no dir/zip load flag in top-level or `exec` help). So bundle-and-inject is **Claude-only** as-is. Before this becomes the primary lane we need a codex path: e.g., does codex support loading a plugin/skill from a local path by any mechanism, or must codex always use the marketplace/`plugin add` lane? The captain set this as the gating question ("once we figure out if there's a good path for codex too").

## Acceptance criteria (provisional — ideation hardens; gated on the codex research)

**AC-1 — `spacedock claude` runs the embedded plugin with zero host config.** End state: on a machine with NO spacedock marketplace/plugin (and even with a stale 0.12.1 installed), `spacedock claude` extracts the embedded plugin and launches `claude --plugin-dir <cache>` such that `--agent spacedock:first-officer` resolves to the embedded plugin. Verified by: a live launch (sentinel agent) confirming the embedded plugin answers over any installed one.
**AC-2 — Embed + extract is correct and cache-keyed.** End state: the binary embeds the full plugin surface and extracts idempotently to a per-version cache dir (no re-extract when unchanged). Verified by: unit test over the extract step + a checksum/version check.
**AC-3 — Codex path decided.** End state: either a working codex bundle/inject path, or an explicit decision that codex stays on the marketplace lane (documented). Verified by: the codex research lands a decision + (if a path exists) a smoke.

## Notes
- Reduces the contract-gate / task-38 pain surface for Claude (an embedded plugin can't go stale) — but 38 still matters for the codex/marketplace lane.
- Sequence: captain wants `38` (+ 0.19.2) and the cobra `cli-cobra-redesign` first; this is "next," pending the codex-path research.
- Coupling to `cli-cobra-redesign`: this decides whether `install`/marketplace is the default Claude path — coordinate so the cobra command surface reflects the chosen distribution model.
