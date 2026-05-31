---
id: 2y7n58x5yx7yn64sy6q3jzmk
title: Sandbox-flag passthrough — spacedock --safehouse-enable / --safehouse-add-dirs
status: implementation
source: sprint — captain (2026-05-31); closes the safehouse --enable gap (.safehouse config can't carry capabilities)
started: 2026-05-31T01:51:45Z
completed:
verdict:
score: "0.32"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-sandbox-flag-passthrough
issue:
---

Let operators pass sandbox capability/path knobs through `spacedock claude` (and `spacedock codex`) to the underlying sandbox, namespaced by sandbox so future sandboxes get their own namespace.

## Why (gap found 2026-05-31)
safehouse's docs confirm `--enable=KEY` (docker, ssh, kubectl, …) is **command-line only** — the workdir `.safehouse` file carries only path grants (`add-dirs`/`add-dirs-ro`), NOT capabilities. The shipped `spacedock claude` (A) passes `--trust-workdir-config` + `extra=nil`, so there is currently NO way to enable docker/ssh through the launcher. The F11 decision ("`--trust-workdir-config` covers it") was based on a wrong assumption about `.safehouse`'s schema.

## Target design (captain, 2026-05-31)
Sandbox-namespaced front-door flags, consumed by the launcher and translated into the existing `safehouse.Wrap(inner, extra)` `extra` slot (which lands before the `--`):

- `spacedock claude --safehouse-enable=ssh,docker --safehouse-add-dirs=<paths> [-- claude-args]`
  → `safehouse --enable=ssh --enable=docker --add-dirs=<paths> --trust-workdir-config -- claude --dangerously-skip-permissions --agent spacedock:first-officer …`
- `--safehouse-enable=` is comma-separated → one repeated `--enable=KEY` per value.
- `--safehouse-add-dirs=` / `--safehouse-add-dirs-ro=` → `--add-dirs=` / `--add-dirs-ro=`.
- The `--safehouse-` prefix is the per-sandbox NAMESPACE: a future sandbox `X` would add `--X-*` flags that map to X's own flag surface. Design the parse/translate seam so adding a namespace is a clean extension — but do NOT build any other sandbox now (YAGNI).
- Applies to both `spacedock claude` and `spacedock codex` (both go through the shared safehouse Wrap path).

## Scope expansion (captain, 2026-05-31): explicit `--safehouse` + OR wrap-trigger

- **`--safehouse` (explicit force-on)** is a front-door flag, consumed like `--skip-contract-check` (never forwarded to the host). It forces the safehouse wrap even when the dir has no `.safehouse` profile.
- **The wrap-decision becomes an OR of three.** The launcher wraps in safehouse if ANY of — a `.safehouse` profile is present (shipped auto-detect), OR `--safehouse` is passed, OR any `--safehouse-enable=`/`--safehouse-add-dirs=`/`--safehouse-add-dirs-ro=` knob is passed. This REVISES the shipped wrap-decision in BOTH launchers (today they wrap only on `.safehouse` presence): `runClaude` (merged) and `runCodex` (pending the `codex-safehouse-launcher` merge).
- **Knobs imply sandbox-on.** Passing any `--safehouse-*` knob auto-turns the sandbox on (implies `--safehouse`), so `spacedock claude --safehouse-enable=docker` in a no-profile dir WORKS (it wraps). This REVERSES the earlier fail-fast AC-4. The only remaining "plain launch" (no wrap) case is when NONE of {`.safehouse` profile, `--safehouse`, a `--safehouse-*` knob} is present.
- **Codex safety rule unchanged.** Codex's plain path is `codex` with NO bypass; only the wrapped path carries `--dangerously-bypass-approvals-and-sandbox` (safe only because safehouse is then the sandbox). When `--safehouse`/a knob now forces the wrap in a no-profile dir, codex correctly gets the bypass flag inside the wrap.
- `--no-safehouse` (force-OFF) is OUT of scope (YAGNI) — not needed for this gap.

## Design (hardened, ideation 2026-05-31)

### Seam: where parse/translate lives, how codex reuses it
Two functions, split by what each one *knows*:

1. **Namespace dispatch — `internal/cli`, alongside `splitFrontDoorArgs`.** A
   front-door splitter pulls the `--safehouse-*` tokens out of `args` (the same
   pass that already consumes `--skip-contract-check`/`--`), strips the
   `--safehouse-` prefix, and hands the de-prefixed tokens to the safehouse
   translator. This is the ONLY place that knows the `--safehouse-` ↔ safehouse
   mapping. A future sandbox `X` adds one more prefix→translator arm here; it does
   not touch the translator or `Wrap`.

2. **Flag translation — `internal/safehouse`, next to `Wrap`.** A pure function
   `safehouse.TranslateFlags(deprefixed []string) (extra []string, err error)`
   that knows safehouse's flag vocabulary: `enable=ssh,docker` → `--enable=ssh
   --enable=docker`; `add-dirs=P` → `--add-dirs=P`; `add-dirs-ro=P` →
   `--add-dirs-ro=P`. It returns the `extra` slice fed verbatim into the existing
   `safehouse.Wrap(inner, extra)` slot. It owns NO namespace knowledge — it is
   the safehouse-specific half a future sandbox's translator would sit beside.

The de-prefix boundary is the extension seam: `internal/cli` maps prefixes to
translators; each `internal/<sandbox>` package owns its own `TranslateFlags`.
`Wrap`'s signature is unchanged — `extra` is the slot that already exists.

**Codex reuse:** both `runClaude` and `runCodex` call the same front-door
splitter and the same `safehouse.TranslateFlags`, then pass `extra` into the
shared `safehouse.Wrap`. No codex-specific translation. (Per the in-flight
`codex-safehouse-launcher`, `runCodex` already takes `dir`/`lookPath` and routes
through `Wrap`; this change adds the `extra` argument to both call sites.)

### Parse pass interaction with `splitFrontDoorArgs`
The bare `--safehouse` flag and the `--safehouse-*` knob tokens must all be
CONSUMED at the front door, never forwarded to the host (claude/codex must never
see them). The cleanest shape folds them into the existing single pass:
`splitFrontDoorArgs` grows two returns — a `forceSafehouse bool` (set by the bare
`--safehouse` token, handled like `--skip-contract-check`) and a `safehouseFlags
[]string` (the de-prefixed `--safehouse-*` knobs) — leaving `passthrough`
host-only and `skipCheck` unchanged. `--` still terminates nothing mid-stream
(current code drops `--` and keeps scanning); a `--safehouse`/`--safehouse-*`
token appearing AFTER `--` is still consumed as a front-door flag, matching the
current treatment of `--skip-contract-check` after `--`. Only the
`enable`/`add-dirs`/`add-dirs-ro` knob keys are recognized; an unknown
`--safehouse-<key>` is a hard error (AC-8), so a typo can never silently fall
through to the host as passthrough. Disambiguation: the bare token is exactly
`--safehouse`; a knob is `--safehouse-<key>=…` (prefix `--safehouse-`). The two
forms do not collide.

### Wrap-trigger: OR of three (revises the shipped `.safehouse`-only trigger)
Both launchers compute `wrap := safehouse.Present(dir) || forceSafehouse ||
len(safehouseFlags) > 0`. This replaces today's `safehouse.Present(dir)`-only
test in `runClaude` (merged) and `runCodex` (pending `codex-safehouse-launcher`).
Everything gated on the wrap decision moves behind `wrap`: claude's
`--dangerously-skip-permissions`, codex's `--dangerously-bypass-approvals-and-sandbox`,
the `safehouse.Available(lookPath)` binary check, and the `safehouse.Wrap` call.
The plain (unwrapped) launch happens only when all three are false.

Passing a `--safehouse-*` knob therefore IMPLIES sandbox-on (the
`len(safehouseFlags) > 0` arm) — `spacedock claude --safehouse-enable=docker` in
a no-profile dir wraps and works. This REVERSES the earlier fail-fast AC-4: there
is no longer a "knob given but plain path" error case, because a knob can never
land on the plain path.

### Resolve: keep `--trust-workdir-config` unconditional when `--safehouse` forces a no-profile wrap
`safehouse.Wrap` keeps emitting `--trust-workdir-config` unconditionally (its
signature is unchanged; only the `extra` slot grows). Rationale: with no
`.safehouse` profile, `--trust-workdir-config` has nothing to load and is a
harmless no-op; omitting it would force `Wrap` to branch on profile-presence (a
new parameter and a second argv shape) for zero behavioral gain. Stable
`Wrap` + unconditional `--trust-workdir-config` is the simpler, YAGNI choice.

## Acceptance criteria

**AC-1 — `--safehouse-enable=ssh,docker` forwards repeated `--enable` flags.**
The finished launcher, given `--safehouse-enable=ssh,docker` with a `.safehouse`
profile present, produces a safehouse argv carrying `--enable=ssh --enable=docker`
(comma-split into repeated flags) positioned before the `--` separator and after
`--trust-workdir-config`.
Verified by: recorded-`Launch` oracle (`fakeHost.launchedArg`, the existing
pattern in `safehouse_frontdoor_test.go`) asserts the exact wrapped argv:
`safehouse --trust-workdir-config --enable=ssh --enable=docker -- claude …`.

**AC-2 — `--safehouse-add-dirs=` / `--safehouse-add-dirs-ro=` forward path grants.**
The finished launcher forwards `--safehouse-add-dirs=<p>` → `--add-dirs=<p>` and
`--safehouse-add-dirs-ro=<p>` → `--add-dirs-ro=<p>` into the same pre-`--` extra
slot, in operator-given order.
Verified by: recorded-`Launch` oracle asserts `--add-dirs=<p>` / `--add-dirs-ro=<p>`
appear in the wrapped argv before `--`.

**AC-3 — the namespace seam is a clean extension point.**
The `--safehouse-` prefix is stripped by the `internal/cli` dispatcher and mapped
to `safehouse.TranslateFlags`; the translator holds no namespace knowledge.
Verified by: a unit-level test of `safehouse.TranslateFlags` (de-prefixed input →
`extra` output, no `--safehouse-` strings reach it) PLUS a front-door test showing
the dispatcher strips the prefix. The two-function split is the structural proof a
second namespace would not require rewriting the dispatcher or `Wrap`.

**AC-4 — explicit `--safehouse` forces the wrap in a no-profile dir (claude).**
`spacedock claude --safehouse` in a dir with NO `.safehouse` profile produces a
safehouse-wrapped argv (`safehouse --trust-workdir-config -- claude
--dangerously-skip-permissions --agent spacedock:first-officer …`); the bare
`--safehouse` token is consumed and never reaches claude.
Verified by: recorded-`Launch` oracle (no `.safehouse` fixture, `--safehouse`
arg) asserts the argv begins `safehouse` and carries `--dangerously-skip-permissions`,
and that `--safehouse` is absent from the inner argv.

**AC-5 — explicit `--safehouse` forces the wrap in a no-profile dir (codex).**
`spacedock codex --safehouse` in a no-profile dir produces
`safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox …`;
the bypass flag appears ONLY because safehouse now wraps. The bare `--safehouse`
token is consumed and never reaches codex.
Verified by: recorded-`Launch` oracle asserts the argv begins `safehouse` and the
inner codex argv carries `--dangerously-bypass-approvals-and-sandbox`, with
`--safehouse` absent from it.

**AC-6 — a `--safehouse-*` knob implies sandbox-on in a no-profile dir (claude & codex).**
With NO `.safehouse` profile and NO bare `--safehouse`, passing a single knob
(e.g. `--safehouse-enable=docker`) still wraps: claude →
`safehouse --trust-workdir-config --enable=docker -- claude --dangerously-skip-permissions …`;
codex → `safehouse --trust-workdir-config --enable=docker -- codex --dangerously-bypass-approvals-and-sandbox …`.
Verified by: recorded-`Launch` oracles (one per host, no `.safehouse` fixture)
assert the argv begins `safehouse` and carries `--enable=docker` before `--`.
(This is the reversal of the old fail-fast AC-4: a knob can never land on the
plain path.)

**AC-7 — the plain launch happens only when none of {profile, --safehouse, knob} is present.**
In a no-profile dir with no `--safehouse` and no `--safehouse-*` knob, the launch
is plain (no `safehouse` token, no `--dangerously-*` flag) — the shipped behavior
is preserved for the unsandboxed case.
Verified by: the existing `TestClaudeNoSafehouseLaunchesPlain` oracle continues to
pass unchanged, plus a codex analog asserting plain `codex` with no bypass.

**AC-8 — unknown `--safehouse-<key>` errors; known flags never leak to the host.**
An unrecognized `--safehouse-<key>` is a hard error (rc≠0, no `Launch`); the bare
`--safehouse` and the three known knobs are consumed at the front door and never
appear in the host passthrough argv.
Verified by: a front-door oracle with `--safehouse-bogus=x` asserts rc≠0; the
AC-1/AC-2/AC-4/AC-5/AC-6 oracles already assert no `--safehouse`/`--safehouse-*`
token survives into the inner `claude`/`codex` argv.

## Test plan
- **Level:** Go unit tests in `internal/cli/safehouse_frontdoor_test.go`
  (recorded-`Launch` oracles via `fakeHost`) + `internal/safehouse/safehouse_test.go`
  (pure `TranslateFlags` table test). No CLI/live-workflow tests needed — the claim
  is argv-shape and parse behavior, both fully observable at unit level.
- **Cost:** low; reuses the existing `fakeHost`/`equalArgv`/`safehouseFixtureDir`
  harness. Estimated a handful of new test funcs.
- **Codex parity:** mirror AC-1/AC-2 and add AC-5/AC-6/AC-7 codex analogs for
  `runCodex` (the wrapped codex argv is
  `safehouse … -- codex --dangerously-bypass-approvals-and-sandbox …`; the plain
  codex argv carries no bypass). The forced-wrap cases (AC-4/AC-5/AC-6) exercise
  the revised OR-trigger in a no-`.safehouse` dir.

## Notes / sequencing
- Lands on `internal/cli/frontdoor.go` (front-door flag parsing, like `--skip-contract-check`; the bare `--safehouse` flag + `--safehouse-*` knob parsing + the OR wrap-trigger) + `internal/safehouse` (`TranslateFlags` beside `Wrap`; `Wrap`'s signature is unchanged — `extra` slot already exists, `--trust-workdir-config` stays unconditional). Shares those files with `codex-safehouse-launcher` (in implementation) → serialize after it. That worktree already reshapes `runCodex` to take `dir`/`lookPath` and route through `Wrap`; this change threads the `extra` argument into both call sites AND revises the wrap-decision in BOTH `runClaude` and `runCodex` from `safehouse.Present(dir)`-only to the OR of {profile, `--safehouse`, knob}.
- Module path is currently `github.com/clkao/spacedock-v1`; the `spacedock-dev/spacedock` migration has NOT landed. Build on the final module path if that migration lands before implementation.

## Stage Report: ideation

- DONE: The `--safehouse-enable=ssh,docker` / `--safehouse-add-dirs=` / `--safehouse-add-dirs-ro=` flag → safehouse-argv translation is pinned as a recorded-Launch exercise-and-observe oracle (comma-split → repeated `--enable=KEY`; paths → `--add-dirs=`), positioned before the `--`, reusing safehouse.Wrap's extra slot.
  AC-1/AC-2 specify recorded-`Launch` oracles over `fakeHost.launchedArg` asserting the exact wrapped argv `safehouse --trust-workdir-config --enable=ssh --enable=docker -- claude …`, before `--`, via the existing slot.
- DONE: The per-sandbox NAMESPACE seam is designed as a clean extension point (the `--safehouse-` prefix maps to safehouse's flag surface; a hypothetical second sandbox namespace would not require rewriting the dispatcher) — WITHOUT building any other sandbox (YAGNI). Name where the parse/translate function lives and how codex reuses it.
  Two-function split: `internal/cli` dispatcher strips the `--safehouse-` prefix (knows the namespace mapping); `safehouse.TranslateFlags` in `internal/safehouse` does pure flag translation (knows safehouse's vocabulary). Both `runClaude` and `runCodex` call the same pair, then feed `extra` to the unchanged `safehouse.Wrap`. AC-3.
- DONE: Resolve AC-4: behavior when `--safehouse-*` flags are passed but the launch takes the no-`.safehouse` (plain) path — pick error vs documented-ignore with a behavioral oracle, and confirm interaction with the existing splitFrontDoorArgs front-door-flag parsing (like `--skip-contract-check`).
  Decision: ERROR (fail-fast) — silent ignore would leave the operator believing docker/ssh is on while running unsandboxed. `--safehouse-*` tokens are consumed in the same `splitFrontDoorArgs` pass (never forwarded to the host); unknown keys hard-error (AC-5), so no token leaks to passthrough. AC-4 oracle: no-`.safehouse` fixture → rc≠0, `launchedArg == nil`, stderr names `.safehouse`.

### Summary
Hardened all three provisional ACs into recorded-`Launch` unit oracles and split the new AC-4/AC-5. Key design decision: a two-function seam — a `--safehouse-` prefix dispatcher in `internal/cli` (the only place that knows the namespace↔safehouse mapping, the clean extension point for a future `--X-*` sandbox) and a pure `safehouse.TranslateFlags` beside `Wrap` (knows only safehouse's flag vocabulary), feeding the existing unchanged `extra` slot. AC-4 resolved as fail-fast error (not documented-ignore) because silently dropping an explicit capability grant is a security-relevant surprise; the `--safehouse-*` tokens are consumed in the existing `splitFrontDoorArgs` pass so they never leak to the host. No production code written. The `spacedock-dev/spacedock` module migration has not landed (module is still `github.com/clkao/spacedock-v1`) — noted as a sequencing condition.

## Stage Report: ideation (cycle 2)

Captain expanded scope: explicit `--safehouse` force-on flag + OR wrap-trigger + AC-4 reversal (knobs imply sandbox-on).

- DONE: The `--safehouse-enable=ssh,docker` / `--safehouse-add-dirs=` / `--safehouse-add-dirs-ro=` flag → safehouse-argv translation is pinned as a recorded-Launch exercise-and-observe oracle (comma-split → repeated `--enable=KEY`; paths → `--add-dirs=`), positioned before the `--`, reusing safehouse.Wrap's extra slot.
  Unchanged from cycle 1 — AC-1/AC-2 (recorded-`Launch` over `fakeHost.launchedArg`).
- DONE: The per-sandbox NAMESPACE seam is designed as a clean extension point (the `--safehouse-` prefix maps to safehouse's flag surface; a hypothetical second sandbox namespace would not require rewriting the dispatcher) — WITHOUT building any other sandbox (YAGNI). Name where the parse/translate function lives and how codex reuses it.
  Unchanged — AC-3 two-function split (`internal/cli` prefix dispatcher + pure `safehouse.TranslateFlags`).
- DONE: Resolve AC-4: behavior when `--safehouse-*` flags are passed but the launch takes the no-`.safehouse` (plain) path.
  REVERSED per captain: knobs now IMPLY sandbox-on, so a knob can never land on the plain path. Old fail-fast AC-4 replaced by AC-6 (knob implies wrap, claude & codex). The only plain-launch case is when none of {profile, `--safehouse`, knob} is present (AC-7).
- DONE: Add explicit `--safehouse` force-on flag (consumed like `--skip-contract-check`).
  AC-4 (claude) / AC-5 (codex): `--safehouse` in a no-profile dir forces the wrap; bare token never leaks to host.
- DONE: Revise the wrap-trigger to OR of three in BOTH launchers.
  `wrap := safehouse.Present(dir) || forceSafehouse || len(safehouseFlags) > 0`; replaces the shipped `.safehouse`-only test in `runClaude` (merged) and `runCodex` (pending merge). AC-7 pins plain-launch only when all three false; codex bypass flag only inside the forced wrap.
- DONE: Resolve `--trust-workdir-config` when `--safehouse` forces a no-profile wrap.
  Keep it unconditional — harmless no-op with no profile; omitting it would force a `Wrap` signature/argv branch for zero gain. `Wrap` stays unchanged.
- DONE: Keep AC-5 (unknown `--safehouse-<key>` hard-errors; known flags never leak to the host).
  Renumbered to AC-8; now also covers the bare `--safehouse` token never reaching the host.

### Summary
Folded the captain's scope expansion into the spec: added the explicit `--safehouse` force-on front-door flag, revised the wrap-decision in both launchers from `.safehouse`-presence-only to an OR of {profile, `--safehouse`, any `--safehouse-*` knob}, and REVERSED the old fail-fast AC-4 into imply-on (a knob auto-turns the sandbox on, so `spacedock claude --safehouse-enable=docker` works in a no-profile dir). Resolved the small open question: `safehouse.Wrap` keeps `--trust-workdir-config` unconditional (harmless no-op without a profile; signature stays stable). ACs renumbered to AC-1..AC-8 with claude+codex forced-wrap oracles; codex bypass flag is emitted only inside the wrap (safety rule preserved). `--no-safehouse` force-off left out of scope (YAGNI). Implementation still serializes after `codex-safehouse-launcher` (shared files) and now also revises that worktree's wrap-trigger. No production code written.

## Scope expansion (captain, 2026-05-31): COMPREHENSIVE launch parity

This entity becomes the comprehensive **launch-parity** vehicle: `spacedock claude`
/ `spacedock codex` must REPLACE every hand-typed safehouse launch in the captain's
real `ps` (full source in `_sprint-notes.md` → "Launch-parity spec from captain's
real ps"). The cycle-1/2 sandbox knobs (AC-1..AC-8) are retained verbatim; this
cycle adds the launch-arg-assembly gaps the `ps` revealed. Both launchers ship today
with a FIXED bootstrap prompt + `.safehouse` auto-detect; this cycle revises the
launch-arg assembly in `runClaude` + `runCodex` as ONE coherent pass.

Captain decisions feeding this cycle:
- **(a) prompt UX = trailing positional.** The launch prompt is `base + " " + task`
  (bare → base only; resume → no prompt). `base` is the existing `bootstrapPrompt`
  / `codexBootstrapPrompt` const.
- **(b) both launchers** get the task-prompt override.
- Multiple `--add-dirs` and multiple `--plugin-dir` must be supported.
- The `--plugin-dir`/gate interaction and the codex-bare decision are
  brainstorm-friendly (captain may steer).

Implementation sequences on `frontdoor.go` (no concurrent migration). All cycle-3
gaps land in the same `runClaude` + `runCodex` revision that threads the cycle-2
`extra` slot through.

### Design: trailing-positional task prompt (LP-1)

**The grammar problem.** Today `splitFrontDoorArgs` routes EVERYTHING that is not
`--skip-contract-check`/`--` into host passthrough. A trailing task must be told
apart from (i) the front-door flags (`--safehouse*`, `--skip-contract-check`),
(ii) the host passthrough (`--model x`, `--plugin-dir P`, `resume <id>`). The
captain's real lines put `--plugin-dir P` BEFORE the task with no `--` separator
(`claude --plugin-dir /…/spacedock "<base + task>"`), so a naive "first bare
positional" rule would mis-assign a host flag's value as the task.

**Decision — the task is the SINGLE trailing bare (non-`-`) positional, scanned
right-to-left, and only the LAST token qualifies.** The front-door parser, after
pulling its own flags, treats the args as: zero-or-more host tokens, then an
OPTIONAL final bare positional = the task. Concretely:

- Scan the post-front-door-flag args. If the LAST token does not start with `-`
  AND is not the value-half of a known host flag that takes a value, it is the
  task; everything before it is host passthrough.
- A `--` separator forces ALL following tokens to host passthrough (escape hatch:
  `spacedock claude -- "a literal claude positional"` sends the positional to
  claude, no task). This preserves the existing `--` contract.
- Bare `spacedock claude` (no trailing positional) → no task → base prompt only.

**Why right-to-left / last-token-only, not "first bare positional":** the captain's
lines interleave host flags (`--plugin-dir P`, `--model M`) whose VALUES are bare
tokens. Claiming the first bare token would steal `P` from `--plugin-dir`. The task
is always the human sentence at the very end of the line, so anchoring on the LAST
token (and requiring no `--` shadow) is the rule that matches every real `ps` line.

**The value-half ambiguity (`--model gpt-x` with no task).** `spacedock claude
--model gpt-x` — is `gpt-x` the task or the value of `--model`? Resolution: the
front-door parser does NOT know claude's/codex's full flag vocabulary (we forward
host flags verbatim by design — see cycle-2's "unknown safehouse key errors but
host flags pass through"). So we cannot reliably know `--model` consumes a value.
Two sub-options:

- **LP-1a (recommended): the task is ONLY recognized when it is unambiguous —
  i.e. the args contain NO host flags, OR the operator used `--` to fence the
  passthrough and put the task before it.** Forms that work:
  `spacedock claude "task"` (pure task) → base+task;
  `spacedock claude "task" -- --model gpt-x` (task fenced before passthrough) →
  base+task with `--model gpt-x` forwarded;
  `spacedock claude -- --model gpt-x` (all passthrough, no task) → base only.
  A trailing bare token that FOLLOWS host flags with no `--` fence
  (`spacedock claude --plugin-dir P "task"`) is the captain's primary line, so it
  MUST work — see LP-1b.
- **LP-1b (captain's real-line shape): allow `--plugin-dir`/`--model`-style host
  flags BEFORE the trailing task without a `--`, by making the trailing-task rule
  "last token is bare AND the token before it is not a known value-taking host
  flag we recognize."** Since we don't model the full host vocabulary, the safe
  set is small: recognize the handful of value-taking host flags the captain
  actually uses (`--model`/`-m`, `--plugin-dir`, `--resume`/`-r` non-`=` form,
  `--add-dir`) only for the purpose of NOT mis-claiming their value as the task.
  This is a heuristic and is brittle.

**RECOMMENDATION (needs captain confirm): LP-1a + a leading-task convention.**
Make the trailing task the LAST token only when there is no `--` fence; require the
operator to put host flags AFTER a `--` fence when combining with a task. BUT the
captain's `--plugin-dir P "task"` line has the flag BEFORE the task. To honor it
WITHOUT a host-flag vocabulary, flip the convention: **the task is the FIRST bare
positional, and host flags follow.** No — `--plugin-dir`'s value is bare and would
be stolen. There is no vocabulary-free rule that handles `--plugin-dir P "task"`
unambiguously. **Therefore the cleanest contract is: the task is the trailing
positional and host flags that take values must be fenced after `--`** — i.e.
`spacedock claude --safehouse-enable=docker "task" -- --model gpt-x --plugin-dir P`.
This is a small ask of the operator and is the ONLY vocabulary-free, unambiguous
rule. The captain's muscle-memory line (`--plugin-dir P "task"`) would need the
fence: `spacedock claude "task" -- --plugin-dir P`. **OPEN-LP1: captain confirms
the fence convention vs. paying for a small recognized-host-flag table (LP-1b).**

**Codex symmetry:** identical rule. `spacedock codex "task"` → `codexBase + " " +
task`; bare → base only; resume → none (LP-2).

### Design: codex resume subcommand + bare codex (LP-2)

Codex's resume is a SUBCOMMAND (`codex resume <uuid>`), not a flag — unlike claude's
`--resume`. The real `ps` line is `safehouse … codex --dangerously-bypass… resume
<uuid>` with NO prompt appended.

- **Detect resume as the first host-passthrough token being `resume`.** When the
  post-front-door passthrough begins with `resume`, forward `resume <rest>` verbatim
  and append NO prompt (mirrors claude's `containsResume` suppression, but codex's
  trigger is the leading subcommand token, not a flag anywhere in the args).
- **Bare codex (no task, no resume): pick base-prompt, NOT truly-bare.** The `ps`
  shows BOTH (line 9 truly-bare `codex --dangerously-bypass…`; line 10 base prompt).
  **DECISION: bare `spacedock codex` → base prompt** (the FO-skill bootstrap), for
  symmetry with `spacedock claude` (which always appends base when not resuming) and
  because the launcher's PURPOSE is to start the first officer — a truly-bare codex
  opens an idle agent with no FO selection, defeating the launcher. The truly-bare
  interactive form is still reachable via `spacedock codex resume` semantics or by
  running `codex` directly outside the launcher; we do not add a flag for it (YAGNI).
  Rationale pinned in LP-AC-2.

### Design: `--plugin-dir` dev-mode + contract-gate interaction (LP-3)

Almost every real claude line carries `--plugin-dir /Users/clkao/git/spacedock`
(and sometimes a 2nd `--plugin-dir …/noteplan-plugin`) — the captain's PRIMARY dev
workflow, loading the LOCAL plugin checkout. Two requirements:

1. **Passthrough verbatim, multiplicity preserved.** One or more `--plugin-dir
   <path>` forward unchanged into the inner `claude` argv in operator order. This is
   already true (they ride the host-passthrough channel); LP-3 only PINS it with an
   oracle and confirms the trailing-task rule (LP-1) does not steal a `--plugin-dir`
   value.
2. **The contract gate must NOT block a `--plugin-dir` dev launch.** The gate
   resolves the INSTALLED plugin's manifest and compares `requires-contract`. But a
   `--plugin-dir` launch loads the LOCAL checkout, whose contract is whatever the
   dev tree is — the installed-plugin verdict is irrelevant (and may fail-fast on a
   stale or absent install, blocking the captain's core loop).

   **DECISION (recommended): detecting any `--plugin-dir` in passthrough relaxes the
   gate — equivalent to an implicit `--skip-contract-check`.** Rationale: a
   `--plugin-dir` launch BYPASSES the installed plugin entirely, so gating on the
   installed plugin's contract is meaningless. The local checkout is the source of
   truth; if it is contract-incompatible the dev will see it at runtime. This is the
   narrowest rule (detect-and-relax), needs no new flag, and matches the existing
   `--skip-contract-check` escape exactly. The operator can still pass
   `--skip-contract-check` explicitly; `--plugin-dir` simply implies it.

   **Alternative considered (rejected): auto-bootstrap** (have the gate point the
   local `--plugin-dir` at a contract check of the dev tree). Rejected as
   over-engineering — there is no installed manifest at the `--plugin-dir` path to
   verify in the gate's vocabulary, and YAGNI: the dev who passes `--plugin-dir`
   already owns the local tree's correctness.

   **Codex parity:** codex has a `--plugin-dir` analog only if the captain uses it;
   the `ps` shows `--plugin-dir` on claude lines only. **DECISION: apply the same
   relax rule symmetrically in `runCodex`** (cheap, consistent, no extra cost) so
   `spacedock codex --plugin-dir P` also relaxes the gate. Pinned in LP-AC-3.

   Interaction with cycle-2: the `--safehouse*` design is UNCHANGED. `--plugin-dir`
   rides the host-passthrough channel (it IS a host flag); only its presence is
   inspected to relax the gate. It does not affect the wrap-trigger OR.

## Acceptance criteria (cycle 3 — launch parity; additive to AC-1..AC-8)

**LP-AC-1 — trailing-positional task → base + " " + task (claude & codex).**
`spacedock claude "do the thing"` (no `.safehouse`, compatible plugin) produces an
inner claude argv whose LAST token is `bootstrapPrompt + " " + "do the thing"`;
bare `spacedock claude` produces an inner argv whose last token is `bootstrapPrompt`
exactly (no trailing space). Same shape for codex with `codexBootstrapPrompt`. A
`--`-fenced host flag (`spacedock claude "task" -- --model gpt-x`) forwards
`--model gpt-x` to claude AND appends `base + " " + task` as the last token.
Verified by: recorded-`Launch` oracles asserting the exact last token equals the
concatenation, plus a fenced-passthrough oracle asserting both the host flag and
the concatenated prompt appear.
(Pending OPEN-LP1: if the captain picks the recognized-host-flag table LP-1b, add
an oracle for `spacedock claude --plugin-dir P "task"` with no fence → `P` forwarded
and `base + " " + task` last.)

**LP-AC-2 — codex resume subcommand suppresses the prompt; bare codex gets base.**
`spacedock codex resume <uuid>` (passthrough begins with `resume`) forwards
`resume <uuid>` into the inner codex argv and appends NO prompt (no `codexBootstrapPrompt`
token anywhere). Bare `spacedock codex` (no task, no resume) appends
`codexBootstrapPrompt` exactly (the base-prompt decision, NOT truly-bare).
Verified by: a recorded-`Launch` oracle for `resume <uuid>` asserting `resume`,
`<uuid>` present and NO prompt token; a bare-codex oracle asserting the last token
is `codexBootstrapPrompt`.

**LP-AC-3 — `--plugin-dir` passes through (multiplicity) AND relaxes the gate.**
`spacedock claude --plugin-dir /a --plugin-dir /b` forwards both `--plugin-dir /a`
and `--plugin-dir /b` verbatim into the inner claude argv in order, AND launches
even when the installed-plugin gate would FAIL (e.g. a too-old-binary or
no-plugin-found manifest) — `--plugin-dir` presence relaxes the gate like an
implicit `--skip-contract-check`. Same relax rule for `spacedock codex --plugin-dir
/a`. Without `--plugin-dir` (and without `--skip-contract-check`) the gate still
fails fast (the cycle-1/2 gate oracles continue to pass unchanged).
Verified by: a recorded-`Launch` oracle with a failing manifest + `--plugin-dir`
asserting rc=0 and both `--plugin-dir` tokens in order; a regression assertion that
a failing manifest WITHOUT `--plugin-dir` still returns rc≠0 with no Launch.

## Test plan (cycle 3 additions)
- **Level:** same Go unit level — recorded-`Launch` oracles in
  `internal/cli/safehouse_frontdoor_test.go` (or a sibling `frontdoor` test file)
  via the existing `fakeHost`/`equalArgv`/`safehouseFixtureDir` harness. The task
  concatenation, codex-resume suppression, and gate-relax are all argv-shape +
  rc-shape claims fully observable at unit level.
- **Cost:** low; reuses the existing harness. A handful of new test funcs per LP-AC.
- **Parser tests:** the trailing-task split logic (LP-1) gets a focused table test
  over `splitFrontDoorArgs` (or its successor) — inputs → (passthrough, task,
  forceSafehouse, safehouseFlags, skipCheck) — covering: pure task, fenced task,
  bare, `--`-only-passthrough, resume.

## Notes / sequencing (cycle 3)
- All cycle-3 work lands in the SAME `runClaude` + `runCodex` revision as the
  cycle-2 `extra`-slot threading and OR wrap-trigger. `splitFrontDoorArgs` grows the
  task return (and the cycle-2 `forceSafehouse`/`safehouseFlags` returns) in one
  coherent pass. No concurrent migration; the `codex-safehouse-launcher` worktree
  has merged (per assignment: claude merged, codex merged), so this is now a single
  serialized revision of `frontdoor.go`, not a post-merge rebase.
- OPEN-LP1 (trailing-task grammar: fence convention vs. recognized-host-flag table)
  is the one design decision left for the captain. Recommendation: ship the
  vocabulary-free fence convention (LP-1a) first; add the small recognized-flag
  table (LP-1b) only if the captain's muscle-memory `--plugin-dir P "task"` line is
  worth the brittleness. Both are observable at unit level; the AC text flags the
  conditional oracle.
- The `--plugin-dir`-relaxes-gate rule (LP-3) reuses the existing `skipCheck`
  plumbing — no new gate path, just an additional trigger for the existing bypass.

## Stage Report: ideation (cycle 3)

Captain expanded scope to COMPREHENSIVE launch parity: the cycle-2 sandbox knobs PLUS
the three launch-arg gaps from the captain's real `ps` (custom task prompt, codex
resume/bare, `--plugin-dir` dev-mode + gate), as one coherent `runClaude`+`runCodex`
revision.

- DONE: Custom-task-prompt as recorded-Launch oracles (captain UX = trailing positional): `spacedock claude "task"` → claude prompt = base + ' ' + task; bare `spacedock claude` → base only; resume (family) → NO prompt. Same shape for codex. Pin how the trailing task is distinguished from the existing front-door flags + claude/codex passthrough.
  LP-1 + LP-AC-1. Decision: task = SINGLE TRAILING bare (non-`-`) positional, not shadowed by a `--` fence; host value-taking flags fence after `--`. The vocabulary-free rule (we forward host flags verbatim, so cannot model their value-arity). OPEN-LP1 left for captain: fence convention (recommended) vs. a small recognized-host-flag table to honor the muscle-memory `--plugin-dir P "task"` line.
- DONE: codex resume + bare: detect codex's `resume` SUBCOMMAND (forward `resume <id>`, NO prompt); resolve bare `spacedock codex` (base vs truly-bare) with rationale + oracle.
  LP-2 + LP-AC-2. Resume = leading passthrough token `resume` (codex's subcommand, distinct from claude's `--resume` flag) → forward verbatim, suppress prompt. Bare codex DECISION: base prompt (not truly-bare) — symmetric with claude, and the launcher's purpose is to start the FO; a truly-bare codex opens an idle no-FO agent. Truly-bare not given a flag (YAGNI).
- DONE: `--plugin-dir` dev-mode + contract-gate: forward one or MORE `--plugin-dir <path>` verbatim (passthrough), AND the contract gate must NOT block a `--plugin-dir` dev launch. Design the gate interaction with a behavioral oracle. Keep cycle-2 --safehouse design intact.
  LP-3 + LP-AC-3. Decision: presence of any `--plugin-dir` in passthrough RELAXES the gate = implicit `--skip-contract-check` (a `--plugin-dir` launch loads the LOCAL checkout, so the installed-plugin verdict is irrelevant). Narrowest rule, reuses existing `skipCheck` plumbing, no new flag. Auto-bootstrap alternative rejected (over-engineering/YAGNI). Applied symmetrically in `runCodex`. Multiplicity preserved; cycle-2 `--safehouse*` wrap-trigger unchanged.

### Summary
Expanded the entity from sandbox-knob passthrough to comprehensive launch parity, folding the three `ps`-revealed gaps into one `runClaude`+`runCodex` revision additive to AC-1..AC-8. Key decisions: (1) trailing task = single trailing bare positional, vocabulary-free, with host value-flags fenced after `--`; the one captain-facing open question (OPEN-LP1) is the fence convention vs. a small recognized-host-flag table for the muscle-memory `--plugin-dir P "task"` line. (2) codex `resume` is detected as the leading passthrough subcommand token (distinct from claude's `--resume` flag) and suppresses the prompt; bare codex gets the base prompt, not truly-bare, for symmetry and because the launcher exists to start the FO. (3) any `--plugin-dir` in passthrough relaxes the contract gate (implicit `--skip-contract-check`) since the local checkout supersedes the installed plugin — reusing existing `skipCheck` plumbing, applied symmetrically to codex, rejecting an auto-bootstrap alternative as over-engineering. Three new ACs (LP-AC-1..3) pinned as recorded-`Launch`/rc oracles on the existing `fakeHost` harness plus a parser table test. No production code written. Implementation is a single serialized `frontdoor.go` revision (codex-safehouse-launcher merged); no concurrent migration.

## Stage Report: implementation

- DONE: Task-prompt override via FENCE convention (recorded-Launch oracles)
  Captain decision applied: host flags BEFORE `--`, bare task AFTER `--`. `splitFrontDoorArgs` returns a `frontDoorArgs{passthrough, task, hasTask, forceSafehouse, safehouseFlags, skipCheck}`; `launchPrompt` = base + " " + task when fenced, base when bare. claude resume-family (`containsResume`) and codex leading `resume` subcommand (`codexResume`) suppress the prompt. Oracles: `TestFenceTaskPromptOverride` (claude/codex fenced + bare + host-flag-before-fence), `TestCodexResumeSubcommandSuppressesPrompt`. SHA 31ad7c8.
- DONE: `--plugin-dir` forwarded verbatim AND relaxes the gate (both hosts); cycle-2 sandbox knobs intact
  `hasPluginDir` relaxes the gate like an implicit `--skip-contract-check`; multiplicity/order preserved on the host-passthrough channel. Cycle-2: `safehouse.TranslateFlags` (enable comma-split, add-dirs/add-dirs-ro, unknown-key hard-error) feeds the unchanged `Wrap` extra slot; OR wrap-trigger `Present(dir) || forceSafehouse || len(safehouseFlags)>0`; knobs imply-on. Oracles: `TestPluginDirRelaxesGate` (claude/codex relax + no-plugin-dir-still-fails-fast), AC-1/2/3/4/5/6/7/8 in `launch_parity_test.go`, `TestTranslateFlags` in safehouse. SHA 31ad7c8.
- DONE: ONE coherent runClaude+runCodex revision in frontdoor.go; existing tests updated; gates green with real exit codes
  Both launchers thread `extra` + the OR-trigger + the prompt logic in one pass. Existing oracles moved host flags off the old `["--", flag]` shape (the fence now means task). `go test ./...` 466 passed (rc=0), `go test -race ./...` rc=0, `gofmt -l .` clean, `go vet ./...` rc=0. SHA 31ad7c8.

### Summary
Implemented the full cycle-3 launch-parity design as one `runClaude`+`runCodex` revision on `frontdoor.go`, with the cycle-2 sandbox knobs (AC-1..AC-8) intact. Applied the captain's OPEN-LP1 FENCE decision: host value-taking flags go before the `--` fence and forward verbatim; the bare task is the text after the fence (vocabulary-free, no host-flag table). `splitFrontDoorArgs` became a single pass returning a `frontDoorArgs` struct (passthrough, fenced task, forceSafehouse, de-prefixed safehouseFlags, skipCheck); `safehouse.TranslateFlags` holds the safehouse vocabulary beside the unchanged `Wrap` (the AC-3 two-function seam). The wrap-trigger is the OR of {`.safehouse` profile, `--safehouse`, any knob}; any `--plugin-dir` relaxes the contract gate symmetrically for both hosts; claude resume-family and codex's leading `resume` subcommand suppress the prompt. All 11 ACs pinned as recorded-`Launch`/rc oracles plus a parser table test; the 12 pre-existing launcher oracles were updated to the fence arg model (host flags before `--`). Did NOT run real safehouse/codex (we are inside safehouse) — all evidence is unit-level argv/rc, stub-recorded via `fakeHost`. A live smoke is captain-run if warranted. Code committed on `spacedock-ensign/sandbox-flag-passthrough` at 31ad7c8.
