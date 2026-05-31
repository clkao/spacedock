---
id: 2y7n58x5yx7yn64sy6q3jzmk
title: Sandbox-flag passthrough — spacedock --safehouse-enable / --safehouse-add-dirs
status: ideation
source: sprint — captain (2026-05-31); closes the safehouse --enable gap (.safehouse config can't carry capabilities)
started: 2026-05-31T01:51:45Z
completed:
verdict:
score: "0.32"
worktree:
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
