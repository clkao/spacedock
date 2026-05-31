---
id: bey37wn2zh5mf2gj01d2a05h
title: spacedock codex — safehouse launcher (codex via skill, no --agent)
status: ideation
source: sprint — Ship the Launcher slice A' (captain, 2026-05-30); codex reachable via safehouse --enable all-agents
started: 2026-05-31T01:32:22Z
completed:
verdict:
score: "0.30"
worktree:
issue:
---

Make `spacedock codex` launch a Codex session through safehouse, with the first officer reached via the `spacedock:first-officer` SKILL (Codex has no `--agent` analog).

## Target behavior (captain, 2026-05-30 — ideation hardens these)

- Codex runs UNDER safehouse with codex's own sandbox bypassed (captain decision: safehouse is the sandbox, not codex's built-in one):
  `safehouse --trust-workdir-config [extra] -- codex --dangerously-bypass-approvals-and-sandbox [prompt that invokes the spacedock:first-officer skill]`
- No `--agent` flag exists in codex (probed: `codex --help` has `plugin`/`exec`, no agent-select). The FO is selected by invoking the skill (initial prompt or auto-load) — exact mechanism is an ideation decision.
- Codex plugin-presence check uses codex's plugin surface (`codex plugin list`, NO `--json` — real codex 0.132.0 rejects `--json`).

## Dependencies
- Reuses the safehouse-detection + interposition helper built by `claude-safehouse-launcher` (e72ambzmkkt3hp1whpz2tczr) → sequence after A.
- Shares the codex-resolution fix landing in tq's hole #2 (`codex plugin list`, no `--json`).

## Acceptance criteria (provisional — ideation hardens each)

**AC-1 — codex safehouse argv.** stub harness observes `safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox <fo-skill-prompt>`.

**AC-2 — FO skill invocation.** the emitted codex prompt/mechanism selects `spacedock:first-officer` (observe the prompt token or config the launcher emits).

**AC-3 — missing codex plugin → clear error, rc≠0, no launch.**

**AC-4 (captain-run) — live codex smoke** through safehouse yields a working FO session outside the sandbox.

## Open question for ideation
- Whether `spacedock codex` requires `.safehouse` (codex self-sandboxes; captain chose safehouse-as-sandbox with codex bypass) or supports a no-safehouse fallback like the claude path.
