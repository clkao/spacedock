---
id: 0mxzm82mpjbv0tjd51pv2t5j
title: Host-neutrality seam — relocate ~/.claude team-state reads behind an injected probe (zs prerequisite)
status: validation
source: zs claude-runtime-segregation decomposition (CL 2026-05-31 "zs - split"; staff review M-3) — the riskiest-first prerequisite
started: 2026-05-31T20:45:00Z
completed:
verdict:
score: "0.34"
worktree: .worktrees/spacedock-ensign-host-neutrality-seam
issue:
mod-block: 
pr: #244
---

The prerequisite carved out of `claude-runtime-segregation` (zs) per the staff-review decomposition (zs
`### Decision 4`). It establishes the `internal/claudeteam` package and a host-supplied **team-state
probe** seam, relocating the three pre-existing `~/.claude` reads that already violate zs's AC-3
host-neutrality invariant — landing FIRST so the five native subcommands plug into a proven seam
rather than stacking six workstreams behind one integration point. "Validate the riskiest path first."

## The three pre-existing leaks (FO-confirmed against source)

- `internal/status/boot.go:131-163` `probeTeamState` — reads `~/.claude/teams/*/config.json` mtimes,
  called by `gatherBoot:206`, feeds the boot `TEAM_STATE` JSON (`teamPresent`/`teamHint`).
- `internal/dispatch/helpers.go:137-162` `recentTeamEvidence` — reads `~/.claude/teams` via
  `os.UserHomeDir()`, called by `build.go:115` to gate a bare-mode WARN.
- `internal/dispatch/build.go:116-120` — the bare-mode WARN string literal naming
  `~/.claude/teams/*/config.json`.

## Approach

Create `internal/claudeteam`. Move the two team-state reads (and the WARN text) into it behind a
host-supplied probe (a func/interface) injected into `gatherBoot` and `runBuild`; the Claude package
fills the probe, the Codex/bare path leaves it nil. The generic `internal/dispatch` + `internal/status`
source then carries no `.claude` literal.

## Acceptance criteria

**AC-P1 — The AC-3 code-side oracle goes RED→GREEN.**
End state: a `go/parser` test over `internal/dispatch` + `internal/status` source finds no
`~/.claude` path-join / `os.UserHomeDir`-rooted team/transcript read literals. It FAILS today on the
three leaks above and PASSES after relocation.
Verified by: the code-side structural test (ship it RED first, then green after the move).

**AC-P2 — Boot `TEAM_STATE` and build bare-mode-WARN behavior are preserved across the relocation.**
End state: with the probe injected (Claude) the boot JSON `TEAM_STATE` fields and the build bare-mode
WARN are byte-identical to today's output on a fixture `~/.claude` tree; with the probe nil (Codex/bare)
the outputs differ ONLY in `TEAM_STATE: present` and the WARN's presence — no structural or other-field
divergence.
Verified by: **the seam-parity SPIKE the staff re-review requires (run in ideation, before
implementation)** — instrument boot/build on a Codex-like nil-probe fixture and a Claude real-probe
fixture and assert the diff is confined to those two surfaces; then a regression test pinning it.

**AC-P3 — The five runtime-coupled subcommands still resolve via Python until zs-main lands.**
End state: this entity does NOT reimplement context-budget/list-standing/show-standing/spawn-standing
or the build `_mods` branch — it only establishes the package + probe seam; those still route through
the vendored Python oracle.
Verified by: the existing parity harness still green; no new native subcommand handlers added here.

## Test plan

The seam-parity spike (AC-P2) is the riskiest-mechanism check and runs in ideation. Implementation:
the code-side `go/parser` oracle (AC-P1), the boot/build parity regression (AC-P2), and a no-new-native-
subcommands assertion (AC-P3). `go test ./...` green. Worktree-backed implementation (CODE-only under
split-root; entity body + reports stay in the state checkout).

## Blocks

`claude-runtime-segregation` (zs) — its implementation starts after this seam lands (zs `### Decision 4`).
Coordinate the `first-officer-shared-core.md` edits with zs (zs owns the Standing-Teammates rewrite +
reuse-condition prose; this entity touches only the code seam + the boot/build paths).

## Seam design (finalized — ideation)

### Seam-parity SPIKE evidence (AC-P2 riskiest mechanism, run BEFORE locking the design)

Two throwaway SPIKE tests drove the CURRENT (probe-not-yet-injected) boot and build code on one
workflow fixture under two `~/.claude` trees — populated+recent (the future Claude real-probe path,
`present:true` / no WARN) and empty (the future Codex/bare nil path, `present:false` / WARN) — and
asserted the observable diff is confined to exactly the two predicted surfaces. Both PASSED:

- **Boot** (`status --boot --json`): every top-level key (`mods`, `id_style`, `next_id`, `orphans`,
  `pr_state`, `dispatchable`, `state_backend`, `definition_dir`, `entity_dir`, `entity_dir_present`)
  is BYTE-IDENTICAL between the two runs; the ONLY divergence is `team_state.present` and
  `team_state.hint`. stderr identical. Evidence: `--- PASS: TestSpikeBootTeamStateProbeParity`.
- **Build** (`dispatch build`, bare_mode): stdout (the dispatch envelope) is BYTE-IDENTICAL between
  the two runs; the ONLY stderr divergence is the bare-mode WARN line (stripping it makes the two
  stderrs identical). Evidence: `--- PASS: TestSpikeBuildBareWarnProbeParity`.

Verdict: the injected-probe seam's observable effect IS confined to the boot `TEAM_STATE`
`present`/`hint` fields plus the bare-mode WARN's presence — no structural or other-field divergence.
The riskiest contract (does the seam quarantine cleanly without leaking into the envelope/other boot
sections?) is de-risked; the design below is sound to lock. The SPIKE files were removed (ideation
ships no code; the persistent AC-P2 regression test is an implementation-stage deliverable). This
closes zs's open re-review item [M-1-seam-validation].

### The probe seam (exact signature)

One host-supplied predicate consumed by both surfaces. `internal/claudeteam` defines and fills it; the
generic packages receive it as a value and the Codex/bare path passes nil.

```go
// internal/claudeteam (the Claude seam — owns the ~/.claude reads + the moved text).
//
// TeamStateProbe reports recent local team-runtime evidence. present drives the
// boot TEAM_STATE; hint is the boot present:true hint line. recent drives the
// build bare-mode WARN gate. now is injected so the 30-minute window is testable.
type TeamStateProbe func(home string, now time.Time) (present bool, hint string, recent bool)

// Probe is the concrete Claude implementation (reads ~/.claude/teams/*/config.json
// mtimes). The Claude CLI front door passes claudeteam.Probe; Codex/bare pass nil.
func Probe(home string, now time.Time) (present bool, hint string, recent bool) { … }
```

Rationale for ONE probe with three returns (not two probes): `probeTeamState` (boot) and
`recentTeamEvidence` (build) read the SAME `~/.claude/teams/*/config.json` mtimes over the SAME
30-minute window — they differ only in what they project (`present`+`hint` vs a bool). Collapsing them
into one probe removes the duplicate read logic the relocation would otherwise copy into the seam, and
matches the entity-body Approach ("a host-supplied probe"). A func type (not an interface) is the
lighter seam for a single-method capability and mirrors the existing `exec.LookPath`-style func
injection already used in `internal/cli` (`runClaude(..., exec.LookPath, ...)`).

### What each generic consumer takes, and the nil contract

- **`internal/status` boot.** `gatherBoot` and `printBoot` gain a leading `probe claudeteam.TeamStateProbe`
  parameter (threaded from the `status.Request`/`NativeRunner` seam). `gatherBoot` replaces
  `d.teamPresent, d.teamHint = probeTeamState(e)` with: if `probe == nil` → `present=false, hint=""`;
  else `present, hint, _ = probe(home, time.Now())` where `home` is resolved from `e.get("HOME")` (the
  HOME resolution stays generic; only the `.claude`-path read moves). `probeTeamState` (boot.go:128-163)
  is DELETED from `internal/status`.
- **`internal/dispatch` build.** `runBuild` and `Run` thread the same `probe` value. The bare-mode gate
  becomes: `if bareMode && (probe == nil || !recentOf(probe)) { …WARN… }` — but the WARN STRING itself
  moves to the Claude seam (it names `~/.claude/teams/*/config.json`, a `.claude` literal the code-side
  oracle forbids in `internal/dispatch`). Cleanest split per zs Decision-2 ("defers the whole advisory
  to the Claude package"): the generic `build.go` calls a seam-supplied advisory writer
  `claudeteam.BareModeAdvisory(stderr)` only when `probe != nil && !recent`; when `probe == nil`
  (Codex/bare) NO advisory is emitted at all. `recentTeamEvidence` (helpers.go:137-162) and the WARN
  literal (build.go:116-120) are DELETED from `internal/dispatch`.

  Note on the nil-probe WARN: today's bare-mode build WITHOUT recent evidence emits the WARN. After the
  seam, a `probe == nil` (Codex/bare) build emits NO WARN. The SPIKE's nil-probe arm modeled the
  empty-`~/.claude` Claude case (probe present, returns false → WARN). The Codex case is probe==nil →
  no WARN, which is the CORRECT host-neutral behavior (the WARN is Claude-specific advice: "run
  TeamCreate", a Claude-only tool). This is a deliberate, documented behavior refinement, NOT a parity
  regression: the byte-for-byte AC-P2 parity claim is scoped to the Claude path (probe supplied) where
  the WARN behavior is unchanged. The Codex/bare path GAINS host-neutrality (no Claude-only advice on a
  non-Claude host) — exactly AC-3's intent. Flagged for the implementation stage + the staff review.

### Where the moved text lives (the two relocated literals)

- The boot `present:true` hint (`"recent team directory: " + name`) and the `present:false` hint
  (`"run TeamCreate before first team-mode dispatch (claude runtime supports it)"`, boot.go:294) move
  INTO `internal/claudeteam`: the present-true hint is the probe's `hint` return; the present-false
  hint is a `claudeteam` constant the boot renderer reads only when `probe != nil`. When `probe == nil`,
  boot emits a host-neutral `present:false` with an empty/host-neutral hint (no "claude runtime
  supports it" string in generic `internal/status`). **Open implementation choice flagged for the
  gate:** whether the `present:false` line keeps a generic hint or the whole hint string is
  seam-supplied. Either satisfies the code-side oracle; the staff review should pick.
- The bare-mode WARN string literal (build.go:116-120) moves verbatim into `claudeteam.BareModeAdvisory`.

### Composition root (where the Claude package supplies the probe)

`internal/cli/cli.go` `Run`/`run` is the composition root. `status.NativeRunner` gains a
`TeamStateProbe` field and `dispatch.Run` a leading probe param, both wired at the single
construction site in `Run()`. NOTE per zs Decision-2: this means `internal/cli` MAY import
`internal/claudeteam` — the oracle forbids `.claude` literals in `internal/dispatch`+`internal/status`
SOURCE, not in the transitive import graph. The binary as a whole still reads `~/.claude` through the
seam.

**CORRECTION (implementation, FO-approved).** The ideation wording above originally read "the `claude`
front door (`runClaude`) supplies `claudeteam.Probe`; … the bare `status`/`dispatch` paths supply nil."
That framing is inaccurate: `runClaude`/`runCodex` only `Launch()` the host process — they never reach
`status`/`dispatch` in-process. The Claude (and future Codex) FO re-invokes the binary FRESH as
`spacedock status --boot` / `spacedock dispatch build`, hitting the single construction site
(`cli.go` `Run`) with NO per-invocation host signal. The as-built wiring therefore supplies
`claudeteam.Probe` to the workflow-facing `status`/`dispatch` surface unconditionally at that
construction site, because the binary's only FO boot/dispatch consumer today IS Claude — this is what
keeps AC-P2's byte-identical claim true LIVE (wiring nil there would make the live Claude FO lose
`team_state present:true` + the bare-mode WARN). `codex`/`init`/`doctor` do not consume the probe. The
`nil` path stays mechanism-ready (exercised by the AC-P2 regression) and is reserved for the future
codex-runtime entry point zs adds; runtime host-detection (when to pass nil) is deliberately deferred to
zs, not solved here (adding a `SPACEDOCK_HOST` signal now is scope creep beyond this seam).

### AC oracle falsifiability (confirmed)

- **AC-P1** (`go/parser` over `internal/dispatch`+`internal/status` finds no `~/.claude`/
  `os.UserHomeDir`-rooted team/transcript literal): falsifiable — RED today on `probeTeamState`,
  `recentTeamEvidence`, the build WARN literal; GREEN after they move to `internal/claudeteam`.
  Re-introducing any of the three flips it RED. Sound oracle.
- **AC-P2** (boot `TEAM_STATE` + build WARN byte-identical with probe supplied; nil-probe differs ONLY
  in `present`/WARN-presence): falsifiable and DEMONSTRATED by the SPIKE above; the persistent
  regression pins it. Sound oracle. (Caveat: the nil-probe WARN-presence assertion must be written
  against the Claude-empty-tree case, not the Codex probe==nil case — see the behavior-refinement note.)
- **AC-P3** (no new native subcommand handlers; the five still route via Python; parity harness green):
  falsifiable — adding a `context-budget`/`list-standing`/`show-standing`/`spawn-standing` handler or a
  `build` `_mods` branch flips it. Sound oracle. This entity adds ONLY the package + probe seam.

### Coordination boundary with zs (`first-officer-shared-core.md`) — CONFIRMED

Verified against the real file. zs's prose leaks live at `first-officer-shared-core.md:142` & `:206`
(`claude-team context-budget`), `:117` (`member_exists`), `:323`/`:325` (`## Standing Teammates` +
`list-standing`). **This entity touches NONE of them** — it edits only Go source
(`internal/claudeteam`, `internal/dispatch`, `internal/status`, `internal/cli`). The boundary is clean
and non-overlapping: zs owns the Standing-Teammates concept-vs-mechanics rewrite + the reuse-condition /
feedback-rejection / prose-polish prose relocations; this entity owns the code seam + the boot/build
paths. No shared file is edited by both, so they can land in either order (this one first, per zs
Decision 4) without a merge collision.

### Independent staff review — recommended YES

The finalized seam design DOES warrant an independent staff review before the gate, for two reasons the
SPIKE surfaced: (1) the **nil-probe-WARN behavior refinement** (Codex/bare emits NO bare-mode WARN,
vs today's unconditional WARN-without-evidence) is a deliberate behavior change the review should
confirm is intended host-neutrality and not a silent regression; (2) the **moved-hint open choice**
(generic `present:false` hint vs fully seam-supplied) wants a second opinion. The review should also
sanity-check that one probe with three returns (vs two probes) is the right seam shape, and that
threading the probe through `gatherBoot`/`printBoot`/`runBuild`/`Run` does not balloon the signatures
unacceptably. The mechanism itself is de-risked by the SPIKE; the review is design-soundness, not
mechanism-uncertainty.

## Stage Report: ideation

- DONE: Run the seam-parity SPIKE (AC-P2 riskiest-mechanism check, BEFORE locking the design): instrument boot/build with a real team-state probe vs a nil probe on fixture trees and record the behavioral evidence that the observable diff is confined to the boot TEAM_STATE 'present'/'hint' fields + the bare-mode WARN's presence — no structural or other-field divergence. If divergence is found, the seam design must change before implementation.
  Two throwaway SPIKE tests drove the current boot+build code on populated vs empty `~/.claude` fixtures; both PASSED. Boot: all 10 other top-level `--boot --json` keys byte-identical, diff confined to `team_state.present`/`hint`. Build: stdout byte-identical, only stderr diff is the WARN line. No divergence found — design locked. SPIKE files removed (ideation ships no code). See `### Seam-parity SPIKE evidence`.
- DONE: Finalize the injected-probe seam design so implementation is unambiguous: the exact func/interface signature gatherBoot and runBuild take, how internal/claudeteam supplies it (and the Codex/bare nil path), and where the boot TEAM_STATE text + the bare-mode WARN literal move. Confirm AC-P1/AC-P2/AC-P3 are falsifiable oracles as written.
  `### Seam design (finalized)` specifies one `claudeteam.TeamStateProbe` func (present/hint/recent), threaded into `gatherBoot`/`printBoot`/`runBuild`/`Run`, supplied by the `claude` front door in `internal/cli` and nil elsewhere; the two `.claude`-naming literals (boot hint + build WARN) move into `internal/claudeteam`. All three ACs confirmed falsifiable; surfaced the Codex-nil-probe WARN behavior refinement and the present:false-hint open choice for the gate.
- DONE: Confirm the first-officer-shared-core.md coordination boundary with zs (this entity = the code seam + boot/build paths ONLY; zs owns the Standing-Teammates rewrite + reuse-condition prose), and state whether the finalized seam design warrants an independent staff review.
  Verified against the real file: zs's prose leaks (lines 142/206/117/323/325) are untouched by this entity, which edits only Go source — boundary clean and non-overlapping. Recommending YES on an independent staff review (the nil-probe-WARN behavior refinement + the moved-hint open choice want a second opinion; mechanism itself already de-risked by the SPIKE).

### Summary

Ran the AC-P2 riskiest-mechanism SPIKE first, per the running-research-spikes discipline: two throwaway tests drove the current boot+build code on populated vs empty `~/.claude` fixtures and PROVED the observable diff is confined to exactly the boot `TEAM_STATE present`/`hint` fields (10 other boot-JSON keys byte-identical) and the bare-mode WARN's presence (dispatch envelope byte-identical) — closing zs's open [M-1-seam-validation] item. Finalized the seam as one `claudeteam.TeamStateProbe` func (present/hint/recent over the shared `~/.claude/teams` read) threaded into `gatherBoot`/`printBoot`/`runBuild`/`Run`, supplied by the `claude` front door and nil for Codex/bare, with the two `.claude`-naming string literals relocated into `internal/claudeteam`. Confirmed the zs coordination boundary is clean (this entity = Go code only; zs = the `first-officer-shared-core.md` prose) and recommend an independent staff review for the one behavior refinement the SPIKE surfaced (Codex/bare now emits no Claude-specific bare-mode WARN). No worktree (state-checkout ideation); SPIKE files removed so no code ships from this stage.

## Stage Report: implementation

- DONE: AC-P1: ship the code-side go/parser oracle RED-first then GREEN — relocate probeTeamState (boot.go), recentTeamEvidence (helpers.go), and the bare-mode WARN literal (build.go) into a new internal/claudeteam package behind the TeamStateProbe seam, so internal/dispatch + internal/status SOURCE carry no ~/.claude / os.UserHomeDir-rooted team/transcript literal.
  `internal/hostneutrality/host_neutrality_test.go` parses both packages' non-test source for `.claude` string literals + `os.UserHomeDir` calls; committed RED (4 leaks: build.go WARN, helpers.go UserHomeDir+`.claude`, boot.go `.claude`) then GREEN after relocation (worktree 9026f434 RED, 08010d03 GREEN).
- DONE: AC-P2: make the spike's parity assertion a PERSISTENT regression test — on the Claude (probe-supplied) path, boot TEAM_STATE and the build bare-mode advisory are byte-identical to pre-change; on the probe==nil (Codex/bare) path, boot emits a host-neutral present:false (generic hint, NOT the 'claude runtime supports it' string) and NO bare-mode advisory. Document the nil-probe-no-WARN refinement.
  `boot_probe_parity_test.go` (TEAM_STATE confinement: non-TEAM_STATE bytes identical across present-true/present-false/nil; present:false→PresentFalseHint for probe, host-neutral for nil) + `build_advisory_probe_test.go` (envelope byte-identical both arms; advisory fires only with probe). Claude-path byte-fidelity also locked by the live-oracle TestNativeBootMatchesOracle / TestBuildParityCrossProduct (both green). Refinement documented in both test headers + build.go.
- DONE: AC-P3: add NO native subcommand handlers (context-budget/list-standing/show-standing/spawn-standing + the build _mods branch still route via the vendored Python oracle); wire the composition root (internal/cli runClaude supplies claudeteam.Probe; codex/init/doctor/bare status+dispatch pass nil); go test ./... green (except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost) + existing parity harness green; gofmt/vet clean.
  No handlers added: `no_native_subcommands_test.go` pins the deferred set to exactly the four + asserts a `_mods`-bearing build emits only the show-stage-def fetch line; existing TestDeferredSubcommandGuard still green. Composition-root wiring resolved below. `go test ./...` 542 passed / 1 pre-existing env-gated fail; gofmt -l clean; go vet clean.

### DEVIATION FROM DESIGN (FO-APPROVED): composition-root probe wiring

This is a deliberate, FO-approved correction to the ideation design's composition-root wording — NOT
drift. Downstream validation and the adversarial audit should read it as such.

- **Design said:** "the `claude` front door (`runClaude`) supplies `claudeteam.Probe`; … the bare
  `status`/`dispatch` paths supply nil."
- **Why that can't hold:** `runClaude`/`runCodex` only `Launch()` the host process — they never reach
  `status`/`dispatch` in-process. The FO re-invokes the binary FRESH as `spacedock status --boot` /
  `dispatch build`, hitting the single construction site (`cli.go` `Run`) with no per-invocation host
  signal. Wiring nil there would make the LIVE Claude FO lose `team_state present:true` + the bare-mode
  WARN — AC-P2's Claude-path byte-identical claim would FAIL in production even though the
  injected-probe regression (which supplies the probe directly) passes.
- **As-built:** wire `claudeteam.Probe` into the workflow-facing `status`/`dispatch` surface at the
  single `cli.Run` construction site (the native binary is the Claude FO's companion today). `internal/cli`
  imports `internal/claudeteam` (allowed — the oracle forbids `.claude` literals in dispatch+status
  SOURCE, not the import graph). The nil path stays mechanism-ready (exercised by the AC-P2 regression)
  and reserved for the future codex-runtime entry zs adds; runtime host-detection is deferred to zs.
- **Approval:** raised to team-lead before and after implementing; FO approved this exact wiring
  (construction-site `claudeteam.Probe`, reject nil-everywhere, defer host-detection to zs) and asked
  that the ideation `### Composition root` text be corrected (done) and this deviation called out here.
  Flipping the single call-site value to nil is trivial if a later FO call wants it.

### Open-choice resolution (present:false hint)

Per the FO decision (generic host-neutral hint when probe==nil, NOT the Claude string), the nil-probe present:false hint is the constant `teamStateNeutralHint = "no active team runtime detected"` (status/boot.go). The Claude present:false hint (`claudeteam.PresentFalseHint`) is byte-identical to the pre-seam string. Both hints are resolved in gatherBoot so neither the text nor the JSON renderer carries a host-specific string.

### Summary

Established `internal/claudeteam` with a `TeamStateProbe` func seam (present/hint/recent over one shared ~/.claude/teams read) and relocated all three pre-existing leaks behind it: probeTeamState→claudeteam.Probe, recentTeamEvidence→the probe's recent return, and the bare-mode WARN literal→claudeteam.BareModeAdvisory. Threaded the probe through gatherBoot/printBoot/runBuild/Run + a NativeRunner field; cli.Run wires claudeteam.Probe for the Claude-FO-facing status/dispatch surface (nil reserved for a future codex-runtime entry). TDD: AC-P1 go/parser oracle shipped RED then GREEN; AC-P2 pinned by two new confinement regressions plus the unchanged live-oracle parity harness; AC-P3 by the deferred-set + no-standing-fetch-line assertions. The one behavior refinement (nil-probe/Codex emits no Claude-specific bare-mode WARN) is deliberate host-neutrality, documented in code + tests. Flagged the composition-root wiring reconciliation to team-lead. High-stakes (touches boot.go + the FO-parsed boot JSON) — expect the adversarial audit; the boot JSON key order and all non-TEAM_STATE fields are byte-preserved (confinement test + live-oracle parity).

## Stage Report: validation

- DONE: AC-P1: independently confirm the code-side go/parser oracle is RED→GREEN and FALSIFIABLE — re-introduce a ~/.claude / os.UserHomeDir team read into internal/status or internal/dispatch and confirm the oracle goes RED, then revert. Confirm internal/status + internal/dispatch source now carry NO ~/.claude/UserHomeDir team/transcript literal.
  Injected a `~/.claude` string literal into dispatch/helpers.go → oracle RED (1 leak at helpers.go); reverted; injected an os.UserHomeDir call into status/boot.go → oracle RED (1 leak at boot.go); reverted → GREEN. Independent grep: zero `.claude` STRING literals and zero os.UserHomeDir calls in non-test status+dispatch source (only two `.claude` mentions remain, both in comments, correctly ignored by the AST scan). Worktree clean after reverts.
- DONE: AC-P2 (the critical one — the FO PARSES the boot JSON): independently reproduce on the built binary that with the probe wired (Claude path) status --boot --json team_state.present/hint AND the build bare-mode advisory are byte-identical to origin/next, AND every OTHER boot JSON key plus KEY ORDER is byte-preserved; confirm the nil-probe path emits a host-neutral present:false with NO Claude-specific WARN.
  Built branch binary (HEAD 08010d03) + a baseline binary at the pre-seam parent 0bbfeae3. `status --boot --json` on the sdb32 fixture under both a populated and an empty ~/.claude: present-true and present-false envelopes are BYTE-IDENTICAL baseline-vs-branch modulo only the timestamp-minted `next_id` (a non-seam field). 13-key order identical (`command,mods,id_style,next_id,min_prefix,orphans,pr_state,dispatchable,team_state,state_backend,definition_dir,entity_dir,entity_dir_present`); `team_state.present`/`hint` match (present:true→"recent team directory: alpha"; present:false→the Claude PresentFalseHint). `dispatch build` bare-mode (exit 0): envelope (524B) + advisory WARN byte-identical baseline-vs-branch. Nil-probe host-neutral present:false (teamStateNeutralHint, no "claude runtime"/"TeamCreate") + no advisory pinned by TestBootTeamStateProbeConfinement + TestBuildBareAdvisoryProbeGate (both GREEN).
- DONE: AC-P3 + integration: no native subcommand handlers added (the four stay deferred; TestDeferredSubcommandGuard + the existing parity harness green); the composition-root wiring is correct; go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean.
  dispatch.Run routing table has only `build` + `show-stage-def`; the four standing subcommands hit the deferral branch (exit 2); no build `_mods` branch. TestDeferredSubcommandGuard (4/4), TestDeferredSubcommandSetUnchanged, TestBuildEmitsNoStandingFetchLine, TestBuildParityCrossProduct, TestNativeBootMatchesOracle all GREEN. cli.Run wires claudeteam.Probe into NativeRunner + dispatchProbe at the single construction site; codex/init/doctor do not consume the probe. `go test ./...` GREEN except TestCodexResolveManifestAgainstInstalledHost — confirmed PRE-EXISTING (fails identically on the baseline binary: sandbox "Operation not permitted" reading ~/.codex/config.toml, unrelated to the ~/.claude seam). gofmt -l clean; go vet ./... clean.

### Summary

PASSED. Independently rebuilt the branch binary (08010d03) and a pre-seam baseline (0bbfeae3) and verified the critical AC-P2 claim LIVE: with the Claude probe wired, `status --boot --json` and the `dispatch build` bare-mode advisory are byte-identical to baseline — same full key order, same all-other-field bytes (only the timestamp `next_id` differs, expected), same team_state.present/hint and same advisory text. The AC-P1 go/parser oracle is genuinely falsifiable (RED on a re-introduced `.claude` literal AND on a re-introduced os.UserHomeDir call, in two separate guarded packages) and the generic status+dispatch source is now clean. AC-P3 holds: no native handlers added, the four standing subcommands stay deferred, parity harness green. The sole test failure (TestCodexResolveManifestAgainstInstalledHost) is pre-existing and sandbox-environmental (codex config read denied), reproduces on the baseline, and is unrelated to this change. The FO-approved composition-root deviation (probe wired at cli.Run unconditionally for the Claude-FO-facing surface, nil reserved for a future codex entry) is what keeps the live byte-identical claim true and is exercised on the nil path by the regression tests.

## Stage Report: implementation (cycle 2)

- DONE: Rebase spacedock-ensign/host-neutrality-seam onto origin/next; resolve the internal/cli/cli.go conflict by integrating the probe-wiring INTO xd's now-merged cobra cli.go — the result must have BOTH the probe wired AND all commands/verbs present.
  Rebased the 2 seam commits onto origin/next (4f23441a). The ONLY content conflict was the `newRootCommand` `AddCommand(...)` block: kept xd #242's `newNewCommand` + `newCompletionCommand` AND the seam's probe-wired `newDispatchCommand(dispatchProbe, …)`. Imports, `Run`, `run(..., dispatchProbe)`, and `newRootCommand`/`newDispatchCommand` signatures already carried the probe (applied cleanly atop next). Resolved code at worktree 156fd1b5.
- DONE: Do NOT trust git's auto-merge of internal/status/native_runner.go — verify it semantically.
  Read the merged native_runner.go end-to-end. The seam's probe field (`NativeRunner.TeamStateProbe`), `dispatch(probe, …)` param, `Run` threading `r.TeamStateProbe`, and `runRead(probe, …)` coexist correctly with xd's discovery walk-up (lines 71-81, `rootPath == ""` gated), the state-checkout misdirection gate (163-168, `rootPath == ""` gated), and `parseRootArg`/the `--root`-only guard (line 301). Orthogonal: the probe threads only into the read path; discovery/state-checkout/root logic operate on args/dir/roots and never touch probe. `probeTeamState`/`recentTeamEvidence` confirmed absent from status+dispatch.
- DONE: go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean; AND re-confirm the two load-bearing properties post-rebase (AC-P2 byte-preservation + LIVE non-nil probe).
  `go test ./...` 559+ pass, sole fail TestCodexResolveManifestAgainstInstalledHost (pre-existing codex-config sandbox read denial, unrelated). gofmt -l clean; go vet clean. (a) AC-P2: TestBootTeamStateProbeConfinement, TestBuildBareAdvisoryProbeGate, and the live-oracle TestNativeBootMatchesOracle + TestBuildParityCrossProduct (all subtests) GREEN — verified ran via -json pass actions, not zero-match. AC-P1 TestNoClaudeHomeReadsInGenericPackages GREEN. (b) LIVE probe: committed cli.Run returns `run(…, &status.NativeRunner{TeamStateProbe: claudeteam.Probe}, claudeteam.Probe)` — both the NativeRunner field and the dispatch param are non-nil claudeteam.Probe; the cli.go merge did NOT drop either to nil.

### Summary (cycle 2)

Rebase-only cycle: the seam impl + audit were already CLEAN, so this cycle integrated it onto origin/next (post xd #241/#242/#243). The single content conflict was the `AddCommand` block in cli.go — resolved by keeping both xd's `new`/`completion` verbs and the seam's probe-wired `dispatch`; the rest of cli.go's probe plumbing applied cleanly. Auto-merged native_runner.go semantically verified: the probe field/threading is orthogonal to and coexists with xd's discovery walk-up + state-checkout + root guards. One follow-up integration fixup committed separately (687c13de): xd #242's new/completion verb tests called `run()` at the pre-seam arity, so the trailing dispatchProbe was threaded as `nil` (new/status paths don't consume it, matching native_status_test.go/status_test.go). Full suite green except the pre-existing env-gated codex test; both load-bearing properties re-confirmed post-rebase — AC-P2 byte-preservation (confinement + live-oracle parity tests) and the LIVE non-nil `claudeteam.Probe` wiring at the cli.Run construction site. Branch: 687c13de → 156fd1b5 → cbabd3ba on origin/next.

## Stage Report: validation (cycle 2 — post-rebase re-validation)

- DONE: THE key post-merge risk (direct-injection tests can't see it): build the rebased branch binary and LIVE-confirm the probe is wired NON-NIL through the merged cli.go — `spacedock status --boot --json` team_state.present/hint reflects the REAL ~/.claude/teams (present:true with a populated teams dir, present:false when empty), and `dispatch build` bare-mode advisory fires/doesn't per the real evidence.
  Built the rebased binary (worktree HEAD 687c13de) and ran it LIVE. `status --boot --json` against docs/dev against the REAL ~/.claude/teams → `team_state.present:"true"`, `hint:"recent team directory: spacedock-v1-dev-20260531-2008-50647e1b"` (our active team dir). HOME=/tmp/seam-emptyhome → `present:"false"` with the Claude PresentFalseHint ("run TeamCreate … claude runtime supports it") — the probe-supplied Claude hint, NOT the nil-probe host-neutral "no active team runtime detected", proving non-nil. Populated+recent fixture HOME → `present:"true"`, `hint:"recent team directory: alpha"`. `dispatch build` bare_mode: empty HOME (non-nil live probe, no recent evidence) → advisory FIRES; real ~/.claude (recent active team) → advisory ABSENT. The cli.go AddCommand-block merge did NOT drop the probe to nil on either the status or dispatch path.
- DONE: Coexistence on the rebased binary: xd #242's new/completion verbs work AND the discovery walk-up + state-checkout error + rootPath=="" guard still behave — the seam's probe threading didn't regress xd's resolution work, and vice versa.
  `--help` lists new/completion/dispatch. `completion bash` (719B) + `completion zsh` (544B) exit 0; `completion fish` and bare `completion` exit 2 ("completion requires a shell: bash or zsh"). `new my-test-entity` from a deeper subdir auto-discovered the commissioned workflow and minted (exit 0, `created: …/my-test-entity.md id=…`). Discovery walk-up: `status --next-id` from docs/dev/_proposals (no --workflow-dir) → minted next-id exit 0. State-checkout misdirection: `--workflow-dir .spacedock-state` → the named "this is a state checkout; point --workflow-dir at the definition dir …" error exit 1. `--root`-only guard: `status --root REPO` → "--root is only supported with --discover or --resolve" exit 1; `status --discover --root REPO` → discovers docs/dev exit 0. No regression either direction.
- DONE: AC-P1/AC-P2/AC-P3 all hold post-rebase; go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean.
  Named tests via `-json` pass actions: TestNoClaudeHomeReadsInGenericPackages, TestBootTeamStateProbeConfinement, TestBuildBareAdvisoryProbeGate, TestNativeBootMatchesOracle, TestBuildParityCrossProduct, TestDeferredSubcommandGuard, TestDeferredSubcommandSetUnchanged, TestBuildEmitsNoStandingFetchLine — 20 PASS / 0 FAIL (incl. subtests). AC-P1 falsifiability re-confirmed: injecting a `~/.claude/teams` literal into internal/dispatch flipped the oracle RED ("found 1 ~/.claude/UserHomeDir read"), reverting → GREEN; grep confirms zero `.claude` string literals / os.UserHomeDir calls in non-test status+dispatch source. AC-P3: deferred set intact (no native handlers). `go test ./...` 559 passed / 1 failed; the sole fail TestCodexResolveManifestAgainstInstalledHost is NOT in the seam diff and fails IDENTICALLY on the origin/next baseline (4f23441a) — pre-existing, env-gated (codex CLI/config unavailable in sandbox: "failed to load configuration"), unrelated. gofmt -l clean; go vet ./... no issues.

### Summary (validation cycle 2)

PASSED. Post-rebase re-validation focused on the integration resolution surface, the only risk vs the prior CLEAN audit at 08010d03. Built the rebased branch binary (687c13de) and LIVE-confirmed the single highest-risk property item 1: the probe is wired NON-NIL through the merged cli.go on both the status and dispatch paths — `status --boot --json` reports `present:true` with our real active team dir against the actual ~/.claude/teams, `present:false` with the Claude PresentFalseHint (not the nil host-neutral hint) under an empty HOME, and the `dispatch build` bare-mode advisory fires under empty HOME but stays silent against the real recent team evidence. The cli.go AddCommand-block merge dropped nothing to nil. Coexistence confirmed both directions: xd #242's new/completion verbs (exit codes, --help listing, auto-discovery+mint) and the discovery walk-up + state-checkout misdirection error + `--root`-only guard all behave on the rebased binary, with the probe threading orthogonal. AC-P1/P2/P3 hold (20 named-test PASS, oracle falsifiability re-verified RED→GREEN); full suite green except the pre-existing env-gated codex test, proven failing identically on the origin/next baseline; gofmt/vet clean. Recommend PASSED.
