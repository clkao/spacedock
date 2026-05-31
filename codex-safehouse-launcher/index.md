---
id: bey37wn2zh5mf2gj01d2a05h
title: spacedock codex — safehouse launcher (codex via skill, no --agent)
status: implementation
source: sprint — Ship the Launcher slice A' (captain, 2026-05-30); codex reachable via safehouse --enable all-agents
started: 2026-05-31T01:32:22Z
completed:
verdict:
score: "0.30"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-codex-safehouse-launcher
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

## Ideation decisions (hardened — codex 0.132.0 probed live)

### Launch contract (AC-1, AC-2) — pinned argv oracles

`spacedock codex` becomes a LAUNCH path mirroring `runClaude`, NOT prose. This SUPERSEDES the
shipped scoping in `_archive/spacedock-packaging` (codex = version-gate + prose only). The current
`runCodex` and `TestCodexFrontDoorVersionGateThenProse` (asserts NO launch) must be REPLACED.

Inner codex argv (host-specific assembly — the only part that differs from runClaude):

    codex --dangerously-bypass-approvals-and-sandbox <fo-prompt>

- `--dangerously-bypass-approvals-and-sandbox` exists verbatim in codex 0.132.0 (`codex --help`):
  sets `sandbox: danger-full-access`, `approval: never`. This is codex's OWN sandbox bypassed —
  safehouse is the sandbox (captain decision).
- Flag-then-positional parse order CONFIRMED live: `codex exec --dangerously-bypass-... -m X "prompt"`
  delivered the positional verbatim as the session `user` message (model-400 forced early exit, no
  hang). The interactive launch form `codex [PROMPT]` shares the same positional mechanism.

**AC-2 mechanism — RESOLVED.** Codex has NO `--agent`/`--skill` select flag (re-probed: top-level,
`exec`, `plugin` — none). The ONLY FO-selection injection point is the positional `[PROMPT]` token.
So the launcher appends a fixed bootstrap prompt that NAMES the skill — exactly parallel to claude's
`bootstrapPrompt`, but the codex form must invoke the skill by name rather than rely on `--agent`.
The user-invocable entrypoint is `spacedock:first-officer` (codex plugin manifest ships
`"skills": "./skills/"`; the skill is `user-invocable: true`; codex-first-officer-runtime.md confirms
"The user-invocable entrypoint is `spacedock:first-officer`"). The emitted prompt must instruct codex
to invoke that skill, e.g. a token containing the literal `spacedock:first-officer` plus the
launch-and-go intent. AC-2 observes the emitted argv contains the `spacedock:first-officer` skill
token. Exact wording is an implementation choice; the load-bearing invariant is the skill-name token.

Outer wrap REUSES the merged `internal/safehouse` seam unchanged:

    safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox <fo-prompt>

When `.safehouse` present → `safehouse.Present(dir)` true → gate on `safehouse.Available(lookPath)` →
`safehouse.Wrap(inner, nil)`. Same dir+lookPath threading as runClaude.

**No-`.safehouse` behavior — DECISION (closes the open question).** Codex self-sandboxes by default.
The bypass flag is ONLY safe when safehouse provides the sandbox. So when `.safehouse` is ABSENT,
`spacedock codex` MUST NOT emit `--dangerously-bypass-approvals-and-sandbox` into an unsandboxed
launch (that would run codex with no sandbox AND no safehouse — the worst case). Two coherent options
for the no-safehouse path; recommend (a):
  (a) RECOMMENDED — require `.safehouse`: with no profile, refuse to launch with an actionable error
      ("`spacedock codex` runs codex under safehouse; this directory has no .safehouse profile").
      Asymmetric with claude (which has a plain fallback) BECAUSE codex's built-in sandbox is the
      thing being bypassed — there is no safe unsandboxed codex launch that also bypasses approvals.
  (b) plain `codex <fo-prompt>` with NO bypass flag (codex keeps its own sandbox). Rejected lean:
      mixes two sandbox models and the FO/dispatch flow assumes the safehouse-as-sandbox contract.
Captain to ratify (a) vs (b) at AC-4 smoke; implementation defaults to (a) unless told otherwise.

### Reuse boundary (AC-1) — confirmed against MERGED main

`internal/safehouse` (commit 02ec334, on main) exposes `Present(dir)`, `Available(lookPath)`,
`Wrap(inner, extra)` — all inner-command-agnostic (Wrap doc: "the claude and codex launchers share
it"; tests already cover a non-claude inner). runCodex reuses all three with identical dir+lookPath
threading as runClaude (frontdoor.go:99-115). Only the codex INNER argv assembly is host-specific:
`{"codex", "--dangerously-bypass-approvals-and-sandbox", <fo-prompt>}` vs claude's
`{"claude", ("--dangerously-skip-permissions"), "--agent", "spacedock:first-officer", ...}`.
runCodex must also gain the `dir`, `lookPath`, and `stdout`/`stderr` params runClaude has (it
currently takes none of the safehouse inputs). The version-gate (`gateHost(ops, "codex", ...)`) and
`splitFrontDoorArgs` reuse is already in place. Note `containsResume` is claude-flag-specific
(`--resume`/`-r`/`--continue`/`-c`); codex resume is a SUBCOMMAND (`codex resume`), so codex
bootstrap-prompt suppression needs its own resume detection or is deferred (FO lean: defer — codex
resume is out of this slice's scope; the bootstrap prompt is appended unconditionally for the
interactive launch and the captain can `codex resume` separately).

### Carried polish items (folded into scope)

1. **host_exec.go `latestVersionDir` lexical→semver (latent bug).** Lines 121-139 pick the
   lexically-greatest dir name; with `0.9.0`+`0.10.0` it picks `0.9.0` (older), contradicting the
   comment. CONFIRMED unreachable today: live cache holds a single dir (`0.12.1`). Bites only after
   `requires-contract` ships AND a 9→10 rollover leaves a stale dir. Fix in scope: semver-aware
   compare (the dir names are plugin versions, e.g. `0.12.1`) when this file is touched for runCodex.
2. **Resolver-branch unit-test gap.** No unit tests for `latestVersionDir` / `codexEntryInstalled` /
   `codexHome` / the cache-path + degradation branches of `resolveCodexManifest`; the only codex
   resolver coverage is one single-version happy-path integration test. In scope: add unit tests for
   these branches (incl. a multi-version dir that pins the semver fix above, and the
   not-installed/no-cache/missing-manifest → "" degradation paths). `codexEntryInstalled` matching on
   `spacedock@spacedock (installed` is confirmed against live `codex plugin list` output.

### Riskiest unknown (AC-4) — captain-run live smoke

We run INSIDE safehouse, so the ensign cannot run the live interactive launch (safehouse binary not
on PATH here; an interactive codex TUI would hang the agent). The mechanism check the captain runs,
in a directory WITH a `.safehouse` profile and codex installed:

    spacedock codex            # observe the launched argv before it execs

Expected exec argv (the implementation-locking oracle):

    safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox \
      "<prompt naming spacedock:first-officer>"

Evidence that confirms before lock: (1) codex's own startup banner shows `sandbox: danger-full-access`
+ `approval: never` (proving the bypass flag landed and safehouse, not codex, is the sandbox);
(2) codex invokes the `spacedock:first-officer` skill from the positional prompt and the FO startup
sequence begins; (3) the FO can run `spacedock status` / dispatch from inside the safehouse sandbox
(outside codex's native sandbox). A non-`.safehouse` run should hit the decision-(a) refusal (or the
ratified alternative). The flag-then-positional parse and prompt delivery are ALREADY confirmed via
the `codex exec` probe above; AC-4 confirms the full safehouse+interactive+skill-pickup chain.

### Implementation note (no production code written this stage)

Implementation shares `frontdoor.go (runCodex)` + `host_exec.go`. The merged claude-safehouse work
(A) is on main, so no current conflict. TDD: write the failing codex-launch front-door test
(observe safehouse-wrapped argv with the skill token) FIRST, replacing the obsolete prose-only test;
then the resolver-branch unit tests + the semver fix.

## Stage Report: ideation

- DONE: The codex launch contract is pinned as exercise-and-observe oracles: `.safehouse`-present → `safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox <fo-skill-invocation>`; RESOLVE how the FO is selected with no `--agent` and the no-`.safehouse` codex behavior.
  AC-1/AC-2 argv oracles pinned; AC-2 mechanism RESOLVED = positional `[PROMPT]` token naming `spacedock:first-officer` (no select flag exists, re-probed); no-`.safehouse` DECISION = refuse-launch (a) recommended (bypass is unsafe without safehouse-as-sandbox). Flag-then-positional parse confirmed live (codex 0.132.0).
- DONE: Reuse boundary confirmed against MERGED main: runCodex reuses internal/safehouse Present+Available+Wrap (same dir+lookPath threading as runClaude); only codex inner-argv assembly is host-specific. Fold the two carried codex-resolver polish items into scope.
  Confirmed against commit 02ec334 (Wrap is inner-agnostic, shared). runCodex must gain dir/lookPath/stdout params; inner = `{codex, --dangerously-bypass-..., <fo-prompt>}`. Both polish items (latestVersionDir lexical→semver; resolver-branch unit-test gap) scoped, with the lexical bug confirmed unreachable today (single `0.12.1` cache dir).
- DONE: Riskiest unknown scoped: a captain-run live codex smoke (we run inside safehouse) — exact command + observed evidence that confirms the codex argv before implementation locks it.
  AC-4 command `spacedock codex` in a `.safehouse` dir; evidence = codex banner `sandbox: danger-full-access`/`approval: never` + FO skill pickup + `spacedock status` reachable. Partial mechanism (flag+positional parse, prompt delivery) already confirmed via a non-hanging `codex exec` probe; safehouse not installable here so the interactive chain is captain-run by design.

### Summary

Hardened the codex launcher from prose-only into a safehouse-wrapped LAUNCH path that mirrors runClaude, superseding the shipped `spacedock-packaging` scoping. The load-bearing resolution: codex has no agent/skill select flag, so the FO is reached via the positional prompt token naming `spacedock:first-officer` (parallel to claude's bootstrapPrompt). Closed the open question with a decision — no `.safehouse` should refuse to launch, since `--dangerously-bypass-approvals-and-sandbox` is only safe when safehouse provides the sandbox (captain to ratify at the AC-4 smoke). No production code written; the two carried resolver-polish items are folded into implementation scope with a TDD-first note.
