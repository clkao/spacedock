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

### Shared helper factoring (the codex launcher reuses this verbatim)

New package `internal/safehouse`:

```
// Wrap detects a .safehouse profile in workdir. When present it returns the
// inner argv wrapped as `safehouse --trust-workdir-config [extra...] -- <inner>`
// and wrapped=true. When absent it returns inner unchanged and wrapped=false.
func Wrap(workdir string, inner []string, extra []string) (argv []string, wrapped bool)

// Present reports whether a .safehouse profile exists in workdir (for the
// pre-launch missing-binary gate).
func Present(workdir string) bool
```

`Wrap` is inner-command-agnostic: `runClaude` passes
`inner = {"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", passthrough...}`;
`codex-safehouse-launcher`'s `runCodex` passes
`inner = {"codex", "--dangerously-bypass-approvals-and-sandbox", <fo-skill-prompt>}`.
Host-specific inner-argv assembly and prompt/`--resume` policy stay in the
respective `runX` functions; only `.safehouse` detection + the safehouse prefix
is shared. This is the reuse boundary the codex launcher's "Dependencies" calls
for.

`.safehouse` detection = `os.Stat(filepath.Join(workdir, ".safehouse"))` is nil
(a file OR directory both count as present; pinned by oracle, see AC-1 notes).

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

### Initial-prompt decision (PENDING captain — see open question)

The `spacedock:first-officer` agent self-bootstraps on launch via its Startup
sequence (discover root → `spacedock status --discover` → read README → `--boot`),
so a synthetic default initial-prompt is redundant with the agent definition.
Recommended: NO default injected prompt — operator-supplied trailing args pass
through verbatim; `--resume` is just one such forwarded flag. Under this choice
AC-5 reframes to: "no synthetic prompt is ever injected; `--resume` and any
operator args forward verbatim." If the captain instead wants a fixed bootstrap
prompt (option B), AC-5 keeps its original "prompt appended unless `--resume`"
shape. Oracles below are written for the recommended no-default choice and noted
where option B diverges.

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

**AC-1 — `.safehouse` present emits the canonical safehouse-wrapped argv.**
Exercise: `runClaude` with a fixture dir containing a `.safehouse` file and
args `{"--", "--foo"}`, backed by a `fakeHost` whose `Launch` records argv.
Observe: recorded argv equals
`["safehouse","--trust-workdir-config","--","claude","--dangerously-skip-permissions","--agent","spacedock:first-officer","--foo"]`
and rc==0. (Also pin: `.safehouse` as a directory is detected identically — one
extra case asserting `os.Stat` truthiness covers file-vs-dir.)

**AC-2 — no `.safehouse` launches plain claude, safehouse never named.**
Exercise: same harness, fixture dir with NO `.safehouse`, args `{"--","--foo"}`.
Observe: recorded argv equals
`["claude","--dangerously-skip-permissions","--agent","spacedock:first-officer","--foo"]`
— argv[0] is `claude`, the token `safehouse` appears nowhere in argv, rc==0.
(NOTE: this pins that the no-safehouse path also carries
`--dangerously-skip-permissions`; confirm with captain — the provisional AC-2
text omitted it. If the captain wants the skip-permissions flag ONLY under
safehouse, AC-2's expected argv drops that token. Flagged as open question 2.)

**AC-3 — missing plugin → clear error, rc≠0, no launch.**
Exercise: `runClaude` with a `fakeHost` returning `manifest:""` (no plugin) —
the gate fails before any safehouse logic. Observe: rc≠0, stderr contains an
actionable remedy (`spacedock init`/`spacedock doctor`), and `fakeHost.Launch`
was never called (`launchedArg == nil`). This exercises tq's `gateHost` through
the launcher path; the safehouse interposition is downstream of the gate.

**AC-4 — `.safehouse` present but `safehouse` binary absent → install hint, rc≠0, no claude launch.**
Exercise: fixture dir WITH `.safehouse`; the launcher's pre-launch
`exec.LookPath("safehouse")` (or an injected `lookPath` seam returning
not-found) reports absent. Observe: rc≠0, stderr contains a pinned safehouse
install hint, and `Launch` was never called. Decision: the launcher pre-checks
safehouse presence and emits the pinned hint ITSELF rather than deferring to
`execHost.Launch`'s generic LookPath error — so the error is testable without a
real exec and the message is actionable. The LookPath check is injected as a
seam (default = `exec.LookPath`) so the test pins not-found deterministically.

**AC-5 — operator args (incl. `--resume`) forward verbatim; no synthetic prompt injected.**
(Recommended no-default-prompt choice.) Exercise: `runClaude` with `.safehouse`
present and args `{"--","--resume"}`. Observe: recorded argv ends with the
operator's `--resume` and contains NO synthetic initial-prompt token; argv ==
`["safehouse","--trust-workdir-config","--","claude","--dangerously-skip-permissions","--agent","spacedock:first-officer","--resume"]`.
DIVERGENCE under option B (default prompt): two cases — without `--resume` the
argv ends with the bootstrap-prompt token; with `--resume` the prompt token is
absent.

**AC-6 (captain-run, closes F3 / Risk A) — live safehouse smoke.**
Captain runs, OUTSIDE the sandbox, in a real unsandboxed shell:

    safehouse --trust-workdir-config -- claude --agent spacedock:first-officer --help

Observed evidence that closes F3 and unblocks the implementation argv-lock:
- claude's own `--help` text appears on stdout (NOT a `safehouse: unknown flag`
  or `safehouse: unrecognized argument` error, and NOT a claude
  "unknown flag --trust-workdir-config" — i.e. safehouse consumed
  `--trust-workdir-config` and the `--` correctly handed the remainder to
  claude).
- rc == 0.
Recorded as evidence (paste of the command + the observed stdout head + rc) under
the entity's evidence trail before implementation pins the canonical argv. If the
observed flag surface differs (e.g. safehouse spells the trust flag differently,
or rejects `--`), the AC-1 expected argv is corrected to match reality BEFORE
implementation — this is the gate F3 guards.

## Open questions for captain

1. **Initial prompt**: confirm NO default injected prompt (recommended, option A),
   vs a fixed bootstrap prompt (option B). Drives AC-5's exact shape.
2. **`--dangerously-skip-permissions` scope**: AC-1 (safehouse path) clearly
   carries it per the target behavior. Does the NO-safehouse path (AC-2) also
   carry `--dangerously-skip-permissions`? The provisional target behavior for
   the no-`.safehouse` case said only `claude --agent spacedock:first-officer
   [forwarded args]` (no skip-permissions). If that's intended, AC-2's expected
   argv must DROP `--dangerously-skip-permissions`. Current oracle draft assumes
   it is safehouse-path-only (matching the provisional text); confirm.

## Complexity flag

This touches the front-door launch contract (shared with tq, reused by the codex
launcher) and pins a process-exec argv that gates a captain-run live smoke. Per
the assignment's note, a staff review may precede the ideation gate. Recommend it:
the argv shape and the two open questions above are the load-bearing decisions and
benefit from a second read before implementation locks them.

## Stage Report: ideation

- DONE: The `.safehouse`-present and `.safehouse`-absent launch argv contracts are pinned as behavioral exercise-and-observe oracles (safehouse-stub records argv), with the `--resume`-suppresses-injected-prompt case and the missing-safehouse error case each enumerated as their own oracle — no prose-only ACs.
  AC-1/AC-2/AC-5/AC-4 in "Hardened acceptance oracles"; argv recorded in-process by tq's `fakeHost.Launch` (stronger than a PATH stub) — see "Integration surface".
- DONE: Salvage call made explicit: state what is harvested from the superseded launcher vs rebuilt, and name where the safehouse-detect+interpose helper factors so the codex launcher reuses it.
  "Salvage call" section: run.go exec-shape reused via tq's `execHost.Launch`, `safehouse-stub.sh` harvested for AC-6 only; helper factors as `internal/safehouse.Wrap/Present`, inner-argv-agnostic for codex reuse.
- DONE: Riskiest-unknown spike scoped: AC-6 live safehouse smoke is captain-run — the ideation states the exact command the captain runs outside the sandbox and what observed evidence closes F3 before implementation locks the argv contract.
  AC-6: exact command `safehouse --trust-workdir-config -- claude --agent spacedock:first-officer --help`, observed evidence = claude help on stdout + rc==0 (not a safehouse flag error), recorded before argv-lock.

### Summary

Resolved the integration surface against tq's in-flight `frontdoor.go`: this launcher interposes a shared `internal/safehouse` helper (`Wrap`/`Present`, inner-argv-agnostic so codex reuses it) into the existing `runClaude`/`Launch` seam rather than rebuilding exec plumbing; the argv oracles are pure Go unit tests on the recorded `Launch` argv. Two captain decisions are open and flagged: (1) no-default-vs-fixed initial prompt, (2) whether the no-safehouse path also carries `--dangerously-skip-permissions`. Flagged complexity (shared front-door contract + captain-run argv-gating smoke) and recommended a staff review before the ideation gate; implementation MUST serialize after tq merges.
