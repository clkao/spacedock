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
`--safehouse-*` tokens must be CONSUMED at the front door, never forwarded to the
host (claude/codex must never see `--safehouse-enable`). The cleanest shape is to
fold them into the existing single pass: `splitFrontDoorArgs` grows to also
collect `--safehouse-*` tokens into a separate return (`safehouseFlags`), leaving
`passthrough` host-only and `skipCheck` unchanged. `--` still terminates nothing
mid-stream (current code drops `--` and keeps scanning); a `--safehouse-*` token
appearing AFTER `--` is still consumed as a front-door flag, matching the current
treatment of `--skip-contract-check` after `--`. Only the `enable`/`add-dirs`/
`add-dirs-ro` keys are recognized; an unknown `--safehouse-<key>` is a hard error
(AC-5), so a typo can never silently fall through to the host as passthrough.

### AC-4 decision: ERROR (fail-fast), not documented-ignore
When `--safehouse-*` flags are present but the launch takes the plain (no
`.safehouse`) path, the launcher EXITS non-zero with an actionable message and
never launches. Rationale: silently ignoring an explicit capability/path grant
leaves the operator believing docker/ssh is enabled when the process is running
unsandboxed — a security-relevant surprise. Fail-fast matches the front door's
existing posture (`gateHost`, missing-binary hint). The flags are only meaningful
when safehouse interposes, so demanding `.safehouse` is honest.

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

**AC-4 — `--safehouse-*` given on the plain (no-`.safehouse`) path errors.**
With `--safehouse-enable=…` present but no `.safehouse` profile, the launcher
returns non-zero, prints a message naming the offending flag and the absent
`.safehouse` profile, and never calls `Launch`.
Verified by: front-door oracle (no `.safehouse` fixture) asserts rc≠0,
`fakeHost.launchedArg == nil`, and the stderr message mentions `.safehouse`.
Covers both `spacedock claude` and `spacedock codex`.

**AC-5 — unknown `--safehouse-<key>` errors; known keys never leak to the host.**
An unrecognized `--safehouse-<key>` is a hard error (rc≠0, no `Launch`); the three
known flags are consumed at the front door and never appear in the host
passthrough argv.
Verified by: a front-door oracle with `--safehouse-bogus=x` asserts rc≠0; the
AC-1/AC-2 oracles already assert no `--safehouse-*` token survives into the inner
`claude`/`codex` argv.

## Test plan
- **Level:** Go unit tests in `internal/cli/safehouse_frontdoor_test.go`
  (recorded-`Launch` oracles via `fakeHost`) + `internal/safehouse/safehouse_test.go`
  (pure `TranslateFlags` table test). No CLI/live-workflow tests needed — the claim
  is argv-shape and parse behavior, both fully observable at unit level.
- **Cost:** low; reuses the existing `fakeHost`/`equalArgv`/`safehouseFixtureDir`
  harness. Estimated a handful of new test funcs.
- **Codex parity:** mirror AC-1/AC-2/AC-4 for `runCodex` (the codex argv is
  `safehouse … -- codex --dangerously-bypass-approvals-and-sandbox …`).

## Notes / sequencing
- Lands on `internal/cli/frontdoor.go` (front-door flag parsing, like `--skip-contract-check`) + `internal/safehouse` (`TranslateFlags` beside `Wrap`; `Wrap`'s signature is unchanged — `extra` slot already exists). Shares those files with `codex-safehouse-launcher` (in implementation) → serialize after it. That worktree already reshapes `runCodex` to take `dir`/`lookPath` and route through `Wrap`; this change threads the `extra` argument into both `runClaude` and `runCodex` call sites.
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
