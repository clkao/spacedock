---
id: e72ambzmkkt3hp1whpz2tczr
title: spacedock claude — safehouse launcher (replace the manual invocation)
status: ideation
source: sprint — Ship the Launcher slice A (captain, 2026-05-30); harvested from superseded ~/git/spacedock launcher plan 9bt646cz0h4q79g98qz68k9d
started: 2026-05-30T23:45:55Z
completed:
verdict:
score: "0.40"
worktree:
issue:
---

Make `spacedock claude` a drop-in replacement for the captain's manual Claude Code invocation, launching the first officer through safehouse when a workdir safehouse profile is present. This builds the shared safehouse-detection + interposition helper that the codex launcher reuses.

## Target behavior (captain, 2026-05-30 — ideation hardens these into behavioral oracles)

- **When `.safehouse` exists in the working directory:**
  `safehouse --trust-workdir-config [extra-args] -- claude --dangerously-skip-permissions --agent spacedock:first-officer [initial-prompt]`
  — the initial prompt is appended UNLESS `--resume` is among the forwarded args.
- **When `.safehouse` does NOT exist:** launch plain `claude --agent spacedock:first-officer [forwarded args]` (no safehouse).
- The front door runs the plugin-presence / contract gate already built in `tq` (`internal/cli/frontdoor.go`) before any launch.

## Provenance / salvage

- Supersedes `~/git/spacedock` launcher plan `9bt646cz0h4q79g98qz68k9d` (status=implementation, dispatch-held). Harvest from its worktree (`~/git/spacedock/.worktrees/spacedock-ensign-spacedock-launcher-binary`):
  - `internal/claude/run.go` — `syscall.Exec` process-replace plumbing, `safehouse` LookPath → exit 127, plugin-detect gate.
  - `docs/plans/_evidence/spacedock-launcher-binary-ideation/safehouse-stub.sh` — argv-recording stub pattern for the canonical-argv test.
- NOT salvageable: its `buildSafehouseArgv` is pre-F11 / pre-this-model (no `.safehouse` detection, no `--trust-workdir-config`, no `--dangerously-skip-permissions`, no prompt/`--resume`).
- Lands on the same `internal/cli/frontdoor.go` as `tq`'s front-door fix → MUST sequence after tq merges (no parallel-merge).

## Acceptance criteria (provisional — ideation hardens each into an exercise-and-observe oracle)

**AC-1 — `.safehouse`-present path emits the canonical safehouse argv.**
Verified by: a Go test with a `safehouse` stub on PATH (recording argv) + a fixture `.safehouse` in the workdir observes `spacedock claude --foo` execs `safehouse --trust-workdir-config -- claude --dangerously-skip-permissions --agent spacedock:first-officer --foo`.

**AC-2 — No-`.safehouse` path launches plain claude (no safehouse).**
Verified by: same harness with no `.safehouse` present observes `claude --agent spacedock:first-officer --foo` and that the safehouse stub was never invoked.

**AC-3 — Missing plugin → clear error, rc≠0, no launch.**
Verified by: temp HOME with no plugin; assert install-hint on stderr and neither safehouse nor claude invoked. (Exercises tq's front-door gate through the launcher path.)

**AC-4 — Missing safehouse (when `.safehouse` present) → clear error + install hint, rc≠0.**
Verified by: a Go test with `.safehouse` present and `safehouse` absent from PATH observes a pinned install-hint on stderr, rc≠0, and no claude launch.

**AC-5 — `--resume` suppresses the injected initial prompt.**
Verified by: the stub harness observes that forwarding `--resume` produces argv WITHOUT the initial-prompt token.

**AC-6 (captain-run, closes F3) — live safehouse smoke.**
`safehouse --trust-workdir-config -- claude --agent spacedock:first-officer --help` yields claude's help (not a safehouse flag error) in a real unsandboxed shell. Run by the captain outside the sandbox; recorded as evidence. This is the riskiest unknown (Risk A) and gates implementation lock.

## Notes
- We run inside safehouse now; nested safehouse won't run here, so AC-6 is captain-run. We design, implement, and stub-test up to that line.
- The injected initial-prompt content (what FO-bootstrap prompt, if any) is an ideation decision.

## Ideation design (2026-05-30)

### Integration surface (resolved from tq's in-flight frontdoor.go)

`tq`'s `internal/cli/frontdoor.go` already builds the claude launch and execs it
through an injectable seam — this launcher interposes safehouse into that path,
it does not rebuild it:

- `runClaude(ctx, args, ops, stdout, stderr)` splits passthrough args
  (`splitFrontDoorArgs` consumes `--skip-contract-check` and a `--` separator),
  runs the contract gate (`gateHost`), then builds
  `argv := []string{"claude", "--agent", "spacedock:first-officer"} ++ passthrough`
  and calls `ops.Launch(argv)`.
- `hostOps.Launch(argv []string) error` is the exec seam: production
  (`execHost.Launch`) does `exec.LookPath(argv[0])` + `syscall.Exec`; the test
  fake (`fakeHost.Launch`) records argv in-process.

Consequence: the argv-shape oracles (AC-1/AC-2/AC-5) are pure Go unit tests on
the argv handed to the `Launch` seam — no `safehouse` binary, no PATH stub, no
real exec. The salvaged `safehouse-stub.sh` argv-record pattern is needed ONLY
for AC-6's live exec smoke (captain-run). This is a stronger oracle than the
provisional AC text ("safehouse stub on PATH") assumed.

`runClaude` has no workdir parameter today. Safehouse detection needs one, so the
launcher threads the already-resolved `dir` (the same value `run()` passes to the
status path) into `runClaude` and on into the shared helper. No new cwd lookup.

Blast radius of threading `dir` (NOT a body-only change — flagged for the
implementation TDD so it is not surprised): adding a `dir` parameter changes
`runClaude`'s SIGNATURE, so it cascades to:
- its call site `cli.go:48` — currently `runClaude(ctx, args[1:], execHost{}, stdout, stderr)`;
- all 5 existing `frontdoor_test.go` call sites (lines 66, 83, 103, 126, 146),
  which use the old arity and must be updated to pass a `dir`.
This is part of the same tq-shared-surface conflict: the signature edit touches
the exact lines tq authored, reinforcing the serialize-after-tq requirement.
`runCodex` (codex launcher) takes the same `dir`-threading change when it lands.

### Shared helper factoring (the codex launcher reuses this verbatim)

New package `internal/safehouse`:

```
// Present reports whether a .safehouse profile exists in workdir.
func Present(workdir string) bool

// Available reports whether the safehouse binary is resolvable via lookPath
// (default exec.LookPath; injected as a seam so tests pin not-found). When
// absent it returns ok=false and a pinned install-hint string for stderr.
func Available(lookPath func(string) (string, error)) (ok bool, hint string)

// Wrap returns the inner argv wrapped as
// `safehouse --trust-workdir-config [extra...] -- <inner>`. Callers gate on
// Present (and Available) first; Wrap itself only composes the prefix.
func Wrap(inner []string, extra []string) (argv []string)
```

`Wrap` is inner-command-agnostic: `runClaude` passes
`inner = {"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", passthrough+prompt...}`;
`codex-safehouse-launcher`'s `runCodex` passes
`inner = {"codex", "--dangerously-bypass-approvals-and-sandbox", <fo-skill-prompt>}`.

Complete reuse boundary (so the codex launcher reuses the WHOLE safehouse gate,
not just argv composition): `Present`, `Available`, and `Wrap` are ALL shared.
Only the host-specific inner-argv assembly and the prompt/`--resume` policy stay
in the respective `runX` functions. This closes the M3 concern that "codex reuses
Wrap verbatim" would otherwise leave AC-4's missing-binary gate unshared — the
missing-safehouse pre-check lives in `Available`, which codex's AC-3-analog
reuses directly.

`.safehouse` detection (`Present`) = `os.Stat(filepath.Join(workdir, ".safehouse"))`
is nil (a file OR directory both count as present; pinned by oracle, see AC-1
notes).

Two distinct `--` tokens, never conflated: the safehouse `--` is EMITTED by
`Wrap` to separate safehouse's own flags from the inner command. The operator
`--` is a DIFFERENT token, consumed and stripped upstream by tq's
`splitFrontDoorArgs` (`frontdoor.go:115`) before passthrough ever reaches `Wrap`.
So the `--` that lands in the final argv is always Wrap's safehouse separator,
not the operator's.

### Salvage call (explicit harvest vs rebuild)

Harvested from `~/git/spacedock/.worktrees/spacedock-ensign-spacedock-launcher-binary`:
- `internal/claude/run.go` — the `exec.LookPath` → distinct exit / `syscall.Exec`
  process-replace shape. REBUILT in place: that plumbing already exists in tq's
  `execHost.Launch`; we reuse tq's seam, not run.go's `Run`. The pattern (look up,
  exec-replace, return only on failure) is what we keep.
- `docs/plans/_evidence/.../safehouse-stub.sh` — the argv-recording stub
  (`printf '%s\n' "$@" > FILE`). HARVESTED VERBATIM for AC-6's captain-run live
  smoke evidence script; NOT used by the Go argv oracles.

NOT salvaged (confirmed pre-this-model): run.go's `buildSafehouseArgv` (no
`.safehouse` detection, no `--trust-workdir-config`, no
`--dangerously-skip-permissions`, no prompt/`--resume` handling) and its
`plugin.Detect` gate (superseded by tq's contract gate `gateHost`).

### Initial-prompt decision (captain-decided: FIXED bootstrap prompt)

Captain decision: `spacedock claude` launches AND GOES — a no-prompt launch opens
an idle agent session, which is not the goal. So a short FIXED bootstrap prompt is
appended as the final inner-argv token UNLESS `--resume` is among the forwarded
args (a resume already carries its own session intent, so the bootstrap prompt
would fight it).

Proposed exact wording (captain reviews/tweaks at the gate):

    Begin as the Spacedock first officer: run your startup sequence and work the event loop.

Rationale for this wording: it names the role (`first officer`) and explicitly
triggers the two things the agent must do on a fresh session — its Startup
sequence (discover root → `spacedock status --discover` → README → `--boot`) and
then the dispatch/event loop — rather than leaving the agent idle waiting for the
captain. It is short (one line, no embedded quotes that complicate argv) and
host-neutral enough that the codex launcher can adopt a parallel phrasing.

The prompt token is the LAST element of the inner argv (after the operator
passthrough), so it sits where `claude` treats a trailing positional as the
initial user message. When `--resume` is present, the token is omitted entirely.

### Sequencing

Implementation lands on `internal/cli/frontdoor.go` + `internal/cli/host_exec.go`
(`Launch` argv) — the SAME files tq (`spacedock-packaging`,
tq66yjc7sqbhyc52eg8h2ecx) is editing. This MUST serialize after tq merges to
main; no parallel-merge. New `internal/safehouse` is additive and conflict-free,
but the `runClaude` interposition edit conflicts with tq's lines.

## Hardened acceptance oracles

Each is an exercise-and-observe oracle: drive a code path, observe a recorded
artifact (argv handed to the `Launch` seam, or stderr+rc). No prose-only ACs; no
grep-the-source proofs. Test home: `internal/cli/safehouse_frontdoor_test.go`
(extends the existing `fakeHost` from `frontdoor_test.go`) + `internal/safehouse`
unit tests.

**AC-1 — `.safehouse` present emits the canonical safehouse-wrapped argv (with bootstrap prompt appended).**
Exercise: `runClaude` with a fixture dir containing a `.safehouse` file and
args `{"--", "--foo"}` (no `--resume`), backed by a `fakeHost` whose `Launch`
records argv. Observe: recorded argv equals
`["safehouse","--trust-workdir-config","--","claude","--dangerously-skip-permissions","--agent","spacedock:first-officer","--foo","<bootstrap-prompt>"]`
and rc==0, where `<bootstrap-prompt>` is the fixed token from the Initial-prompt
decision — appended LAST, after the operator passthrough. (Also pin: `.safehouse`
as a directory is detected identically — one extra case asserting `os.Stat`
truthiness covers file-vs-dir.)

**AC-2 — no `.safehouse` launches plain claude (no skip-permissions, no safehouse named).**
Exercise: same harness, fixture dir with NO `.safehouse`, args `{"--","--foo"}`
(no `--resume`). Observe: recorded argv equals
`["claude","--agent","spacedock:first-officer","--foo","<bootstrap-prompt>"]`
— argv[0] is `claude`, the token `safehouse` appears nowhere in argv, AND
`--dangerously-skip-permissions` appears NOWHERE in argv (captain decision:
skip-permissions is safehouse-path-only; never skip permissions in an
unsandboxed launch), rc==0. The bootstrap prompt is still appended on this path
(it is a launch-and-go concern, independent of safehouse).

**AC-3 — plugin-gate failure SHORT-CIRCUITS before any safehouse logic (ordering invariant).**
Exercise: `runClaude` with `.safehouse` PRESENT in the fixture dir AND the
safehouse binary ABSENT (lookPath seam returns not-found) AND a `fakeHost`
returning `manifest:""` (no plugin). Observe: rc≠0, stderr carries the
PLUGIN-gate remedy (`spacedock init`/`spacedock doctor`), stderr does NOT carry
the safehouse install hint, and `Launch` was never called. This proves the
NEW ordering the feature introduces: the contract gate runs first and
short-circuits, so neither the `.safehouse`/`Available` pre-check nor the launch
fires when the plugin is missing — even though a missing safehouse binary would
also be an error if reached. (This is the real ordering invariant, not a re-run
of tq's existing `manifest:""` gate test.)

**AC-4 — `.safehouse` present, plugin OK, but `safehouse` binary absent → install hint, rc≠0, no claude launch.**
Exercise: fixture dir WITH `.safehouse`, a COMPATIBLE `fakeHost` manifest (gate
passes), and `safehouse.Available` driven by an injected `lookPath` returning
not-found. Observe: rc≠0, stderr contains the pinned safehouse install hint
(the `hint` string `Available` returns), and `Launch` was never called.
Decision: the missing-safehouse pre-check lives in `internal/safehouse.Available`
and the launcher emits its pinned hint ITSELF rather than deferring to
`execHost.Launch`'s generic LookPath error — so the error is testable without a
real exec, the message is actionable, AND the codex launcher reuses the same
gate (M3). `lookPath` is injected (default `exec.LookPath`) so the test pins
not-found deterministically.

**AC-5 — `--resume` suppresses the bootstrap prompt; operator args still forward verbatim.**
Exercise: `runClaude` with `.safehouse` present and args `{"--","--resume"}`.
Observe: recorded argv forwards the operator's `--resume` and contains NO
bootstrap-prompt token; argv ==
`["safehouse","--trust-workdir-config","--","claude","--dangerously-skip-permissions","--agent","spacedock:first-officer","--resume"]`.
Companion case (the AC-1 positive already covers it): WITHOUT `--resume` the
argv ends with the bootstrap-prompt token. The two cases together pin the
"appended unless `--resume`" rule from both sides.

**AC-6 (captain-run, closes F3 / Risk A) — live safehouse smoke.**
Captain runs, OUTSIDE the sandbox, in a real unsandboxed shell (the command
matches the canonical AC-1 argv it gates, including `--dangerously-skip-permissions`):

    safehouse --trust-workdir-config -- claude --dangerously-skip-permissions --agent spacedock:first-officer --help

Observed evidence that closes F3 and unblocks the implementation argv-lock:
- claude's own `--help` text appears on stdout (NOT a `safehouse: unknown flag`
  or `safehouse: unrecognized argument` error, and NOT a claude
  "unknown flag --trust-workdir-config" / "unknown flag
  --dangerously-skip-permissions" — i.e. safehouse consumed
  `--trust-workdir-config` and the `--` correctly handed the remainder to
  claude, which accepts both `--dangerously-skip-permissions` and `--agent`).
- rc == 0.
Recorded as evidence (paste of the command + the observed stdout head + rc) under
the entity's evidence trail before implementation pins the canonical argv. If the
observed flag surface differs (e.g. safehouse spells the trust flag differently,
or rejects `--`), the AC-1 expected argv is corrected to match reality BEFORE
implementation — this is the gate F3 guards.

## Captain decisions (resolved at ideation)

1. **Initial prompt**: FIXED bootstrap prompt, appended unless `--resume` is
   forwarded (launch-and-go, no idle session). Proposed wording in the
   Initial-prompt decision section; captain reviews/tweaks at the gate.
2. **`--dangerously-skip-permissions` scope**: SAFEHOUSE-PATH ONLY. The
   no-`.safehouse` path stays plain `claude --agent spacedock:first-officer
   [args]` — never skip permissions in an unsandboxed launch. Reflected in AC-2.

## Complexity flag

This touches the front-door launch contract (shared with tq, reused by the codex
launcher) and pins a process-exec argv that gates a captain-run live smoke. A
staff review was run and its three material findings (M1 dir-threading blast
radius, M2 AC-3 ordering-invariant recharacterization, M3 `Available` gate for
codex reuse) are folded in above. The remaining load-bearing items for the gate:
the canonical argv shape (locked only after AC-6's live smoke) and the proposed
bootstrap-prompt wording (captain tweaks at the gate).

## Stage Report: ideation

- DONE: The `.safehouse`-present and `.safehouse`-absent launch argv contracts are pinned as behavioral exercise-and-observe oracles (safehouse-stub records argv), with the `--resume`-suppresses-injected-prompt case and the missing-safehouse error case each enumerated as their own oracle — no prose-only ACs.
  AC-1/AC-2/AC-5/AC-4 in "Hardened acceptance oracles"; argv recorded in-process by tq's `fakeHost.Launch` (stronger than a PATH stub) — see "Integration surface".
- DONE: Salvage call made explicit: state what is harvested from the superseded launcher vs rebuilt, and name where the safehouse-detect+interpose helper factors so the codex launcher reuses it.
  "Salvage call" section: run.go exec-shape reused via tq's `execHost.Launch`, `safehouse-stub.sh` harvested for AC-6 only; helper factors as `internal/safehouse.Wrap/Present`, inner-argv-agnostic for codex reuse.
- DONE: Riskiest-unknown spike scoped: AC-6 live safehouse smoke is captain-run — the ideation states the exact command the captain runs outside the sandbox and what observed evidence closes F3 before implementation locks the argv contract.
  AC-6: exact command `safehouse --trust-workdir-config -- claude --agent spacedock:first-officer --help`, observed evidence = claude help on stdout + rc==0 (not a safehouse flag error), recorded before argv-lock.

### Summary

Resolved the integration surface against tq's in-flight `frontdoor.go`: this launcher interposes a shared `internal/safehouse` helper (`Wrap`/`Present`, inner-argv-agnostic so codex reuses it) into the existing `runClaude`/`Launch` seam rather than rebuilding exec plumbing; the argv oracles are pure Go unit tests on the recorded `Launch` argv. Two captain decisions are open and flagged: (1) no-default-vs-fixed initial prompt, (2) whether the no-safehouse path also carries `--dangerously-skip-permissions`. Flagged complexity (shared front-door contract + captain-run argv-gating smoke) and recommended a staff review before the ideation gate; implementation MUST serialize after tq merges.

## Stage Report: ideation (cycle 2 — captain decisions + staff findings folded)

- DONE: The `.safehouse`-present and `.safehouse`-absent launch argv contracts are pinned as behavioral exercise-and-observe oracles, with the `--resume`-suppresses-injected-prompt case and the missing-safehouse error case each enumerated as their own oracle — no prose-only ACs.
  AC-1 (safehouse argv + bootstrap prompt appended), AC-2 (plain claude, NO skip-permissions per captain), AC-5 (`--resume` suppresses prompt), AC-4 (missing safehouse via `Available`).
- DONE: Salvage call made explicit; helper factors so the codex launcher reuses it.
  Reuse boundary drawn COMPLETE per M3: `Present`+`Available`+`Wrap` all shared in `internal/safehouse`; only inner-argv assembly + prompt/`--resume` policy stay per-host.
- DONE: Riskiest-unknown spike scoped: AC-6 live safehouse smoke is captain-run with the exact command + evidence that closes F3 before argv-lock.
  AC-6 command updated to match canonical argv (now includes `--dangerously-skip-permissions`).
- DONE: Captain decisions folded — fixed bootstrap prompt (proposed wording for gate review) appended unless `--resume`; skip-permissions is safehouse-path-only.
  "Captain decisions" + "Initial-prompt decision" sections; AC-1/AC-2/AC-5 reshaped accordingly.
- DONE: Staff findings folded — M1 (dir-threading blast radius: `runClaude` signature + `cli.go:48` + 5 `frontdoor_test.go` call sites), M2 (AC-3 recharacterized to the plugin-gate short-circuit ordering invariant), M3 (`Available` lookPath gate in `internal/safehouse` for codex reuse); polish (AC-6 argv parity, two-`--` distinction).
  Integration surface (M1), AC-3 (M2), helper factoring + AC-4 (M3), helper factoring two-`--` note (polish).

### Summary

Folded both captain decisions (fixed launch-and-go bootstrap prompt with proposed wording; skip-permissions safehouse-path-only) and the three material staff findings: corrected the `dir`-threading blast radius (signature + call site + 5 test sites, reinforcing serialize-after-tq), recharacterized AC-3 from a redundant gate re-run into the real plugin-gate short-circuit ordering invariant, and moved the missing-safehouse pre-check into `internal/safehouse.Available` so the codex launcher reuses the full gate (Present+Available+Wrap). The verified-correct core (integration surface, salvage, AC-6) is preserved; the proposed bootstrap-prompt wording is the one item still awaiting captain tweak at the gate.
