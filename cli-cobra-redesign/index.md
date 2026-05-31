---
id: z0epv5b79qcs83w24acd1fnd
title: CLI redesign — cobra migration, install rename, Option-2 passthrough, jargon-free grouped help
status: backlog
source: captain brainstorm (2026-05-31) — `spacedock --help` readability; design APPROVED via superpowers:brainstorming
started:
completed:
verdict:
score: "0.34"
worktree:
issue:
---

The top-level `spacedock --help` is a wall of mixed-altitude text: usage lines mix terse and verbose, some commands carry a wrapping inline sentence while others have none, the bottom is an unscannable prose paragraph, and flags (sandbox knobs, `--plugin-dir`) are buried in prose. This entity migrates the hand-rolled `internal/cli` dispatch to **cobra** and lands a grouped, jargon-free command surface. The design below was **brainstormed and captain-approved** (2026-05-31) — ideation formalizes it into entity-level ACs + a test plan; it does not re-decide.

## Approved design (captain-approved decisions)

1. **Tagline:** `spacedock — agentic workflow launcher`.
2. **cobra** (not hand-rolled): cobra owns flag parsing, command grouping, and per-command help. Overrides the AGENTS.md "stdlib unless a dependency removes real complexity" default — captain chose cobra explicitly for the auto-generated help/grouping/per-command structure.
3. **Grouped, jargon-free top-level help.** No "host front doors" / "contract-gated" in the headline (internal jargon a new user can't parse). Groups: **Launch** (claude, codex), **Setup** (install, doctor), **Workflow** (status, dispatch). **META dropped** — `--version` + the per-command-help pointer live in a one-line footer.
   ```
   spacedock — agentic workflow launcher

   Launch
     claude  [task] [-- claude-flags]   Start Claude Code as your Spacedock first officer
     codex   [task] [-- codex-flags]    Start Codex as your Spacedock first officer
   Setup
     install  [--host claude|codex]     Install the Spacedock plugin for a host, then check it
     doctor   [--host claude|codex]     Check the installed plugin and this binary are compatible
   Workflow
     status    [args]                   Show or update workflow state
     dispatch  build | show-stage-def   Build worker dispatch artifacts

   Run "spacedock <command> --help" for details.  ·  --version prints the version.
   ```
4. **`init` → `install`** (rename). "install" matches the user model (`brew install`, `gh extension install`); `init` misleads (implies operating on the local directory). Behavior unchanged: install per-host plugin, then auto-run doctor; `--check` stays.
5. **Passthrough — Option 2 (task is spacedock's; host flags after `--`).** `spacedock claude [spacedock-flags] [task] [-- host-flags]`. cobra owns spacedock flags (`--safehouse`, `--safehouse-enable=…`, `--skip-contract-check`); the optional positional is the task; everything after `--` forwards verbatim to claude/codex via cobra `ArgsLenAtDash`. spacedock builds `claude <host-flags> --agent spacedock:first-officer "<bootstrap + task>"`. **The prompt is always spacedock-owned, so the `--plugin-dir`-swallow bug is structurally impossible** (this folds in the launcher-arg-hardening). Same for `codex`. **Breaking** UX change (`--` now precedes host flags, not the task); **no back-compat shim** (pre-1.0).
6. **Per-command help** is the detail home: `spacedock claude --help` documents the sandbox knobs, `--plugin-dir`, the `--` forwarding, and an EXAMPLES block as aligned flag lists.

## Acceptance criteria (provisional — ideation hardens into testable end-states)

**AC-1 — Top-level help is grouped + jargon-free.** End state: `spacedock --help` renders the Launch/Setup/Workflow groups with terse one-liners, the tagline, no "front doors"/"contract-gated", no META block, and a one-line footer. Verified by: a golden/substring test over the help output.

**AC-2 — `install` replaces `init`.** End state: `spacedock install [--host]` installs the per-host plugin then doctors; `init` is gone. Verified by: CLI routing test; the install-behavior tests retargeted to `install`.

**AC-3 — Option-2 passthrough; prompt is unswallowable.** End state: `spacedock claude [flags] [task] -- [host-flags]` parses spacedock flags, treats the positional as the task, forwards post-`--` verbatim, and builds the host argv with a spacedock-owned prompt. Verified by: tests asserting the task-vs-`ArgsLenAtDash` split and that a trailing dangling host flag cannot consume the prompt (the old swallow case).

**AC-4 — Per-command help carries the detail.** End state: `spacedock claude --help` lists the sandbox/`--plugin-dir`/`--` forwarding flags + examples. Verified by: substring test over per-command help.

## Notes
- **Folds in** the launcher-arg-hardening (the `--plugin-dir $pwd`-swallow bug dies via Option 2). Drop that item from `cli-ergonomics` (xd); xd keeps workflow auto-discovery + remaining actionable-errors.
- **Coupling to `bundle-asset-distribution`:** if bundling becomes the primary Claude lane, `install`/marketplace becomes the *codex + opt-in* path; the command surface here should not hardcode marketplace-as-default. Coordinate ordering with that entity.
- Captain's release-cadence rule: stay on 0.19.x until the captain flips the minor. No `CONTRACT_VERSION` bump (CLI ergonomics, not an observable-surface contract change) unless the command set the skills depend on actually changes.
- Substantial + breaking → its own entity (not folded into xd).
