---
id: z0epv5b79qcs83w24acd1fnd
title: CLI redesign — cobra migration, install rename, Option-2 passthrough, jargon-free grouped help
status: ideation
source: captain brainstorm (2026-05-31) — `spacedock --help` readability; design APPROVED via superpowers:brainstorming
started: 2026-05-31T22:25:41Z
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

## Acceptance criteria (entity-level end-states)

Each AC is a property of the finished binary, with a behavioral oracle (run `cli.Run` / the in-process command, assert stdout/stderr/exit/argv). No greps-as-proof. All Go unit-level, stdlib + cobra; cost low.

**AC-1 — Top-level help is grouped, jargon-free, and META-free.** End state: `spacedock` with no args, `spacedock --help`, `-h`, and `help` all render the same grouped help on stdout (exit 0, empty stderr): the tagline line `spacedock — agentic workflow launcher`; three group headers `Launch`, `Setup`, `Workflow` in that order; the six commands grouped as `claude`/`codex` under Launch, `install`/`doctor` under Setup, `status`/`dispatch` under Workflow, each with a terse one-liner; and a single footer line pointing at `spacedock <command> --help` and noting `--version`. The output MUST NOT contain the strings `front door`, `contract-gated`, `META`, or a `--version`/`--help` line rendered as its own command row (they live in the footer, not the command list). Verified by: a substring/absence golden test over `cli.Run(["--help"])` stdout — assert each required token present (tagline, three group headers, six command names, footer pointer) AND each banned token (`front door`, `contract-gated`, `META`) absent; a second case asserts no-args and `help` produce byte-identical output to `--help`.

**AC-2 — `install` is the setup verb; `init` is gone.** End state: `spacedock install [--host claude|codex] [--check]` installs the per-host plugin via the host plugin mechanism then runs doctor (claude), or emits the documented codex add prose, exactly as `init` did — behavior unchanged, name changed. `spacedock init` is no longer a recognized command (routes to the unknown-command error, exit 2). `--check`, `--host`, and the `@next` dev-branch pinning are preserved. Verified by: a routing test asserting `cli.Run(["install","--host","claude", ...])` drives the same install→doctor seam the `init_test.go`/`init_devbranch_test.go` cases assert (those tests retargeted from `init` to `install`); plus a negative test asserting `cli.Run(["init"])` exits 2 with the unknown-command message. The real isolated-CLAUDE_CONFIG_DIR behavioral install test (`install_behavior_test.go`) is unaffected — it drives the host CLI directly, not the spacedock verb.

**AC-3 — Option-2 passthrough; the prompt is structurally unswallowable.** End state: `spacedock claude [spacedock-flags] [task] [-- host-flags]` parses the spacedock-owned flags (`--safehouse`, `--safehouse-enable=…`, `--safehouse-add-dirs[-ro]=…`, `--skip-contract-check`, `--plugin-dir`) wherever they appear before `--`, treats the tokens before `--` that are not spacedock flags as the task, and forwards every token after `--` verbatim to the host as passthrough. The built host argv is `claude [--dangerously-skip-permissions] --agent spacedock:first-officer <host-passthrough> <spacedock-owned-prompt>` — the prompt is ALWAYS the last token and ALWAYS spacedock-constructed (`bootstrapPrompt` or `bootstrapPrompt + " " + task`), never sourced from a host token, so no dangling host flag can consume it. Same shape for `codex`. Verified by: (a) a split test asserting `task` = the pre-`--` non-flag positional and `passthrough` = the post-`--ArgsLenAtDash` slice for representative argvs; (b) a regression test for the old swallow case — `spacedock claude --plugin-dir <dir> -- "task"` (and a trailing-dangling-host-flag variant) builds a host argv whose final token is the spacedock prompt and whose `--plugin-dir <dir>` rides in passthrough, proving the task is never adjacent to a value-taking host flag in a way the host could swallow.

**AC-4 — Per-command help carries the detail home.** End state: `spacedock claude --help` and `spacedock codex --help` render (exit 0) the sandbox knobs (`--safehouse`, `--safehouse-enable`, `--safehouse-add-dirs`, `--safehouse-add-dirs-ro`), `--skip-contract-check`, `--plugin-dir`, an explanation of the `--` host-flag forwarding, and an EXAMPLES block; `spacedock install --help` documents `--host`/`--check`. Verified by: substring tests over each command's `--help` stdout asserting the flag names, the `--` forwarding note, and the `Examples` heading are present.

**AC-5 — Skill-depended command set is preserved (no contract regression).** End state: every subcommand the FO/ensign skills invoke resolves with unchanged argv behavior — `spacedock --version` (with the `contract <N>` token intact), `spacedock status …` (all flags: `--discover`, `--boot`, `--next`, `--next-id`, `--archived`, `--where`, `--resolve`, `--validate`, `--workflow-dir`, `--set`, `--archive`, `--new`), `spacedock dispatch build`, `spacedock dispatch show-stage-def`, and `spacedock doctor`. `status` and `dispatch` forward their post-subcommand argv VERBATIM to the existing runners (reparented under cobra, not reparsed by cobra flag handling). Verified by: a forwarding test asserting `cli.Run(["status", <args>])` passes `<args>` unchanged to a fake `status.Runner` (cobra does not consume or reorder them — `DisableFlagParsing` or equivalent on the `status`/`dispatch` subcommands), and the existing `status_test.go`/`native_status_test.go`/dispatch tests pass unchanged.

## Cobra migration surface (internal/cli)

The hand-rolled `run()` switch in `internal/cli/cli.go:37` is replaced by a cobra command tree. The injectable seams (`status.Runner`, `hostOps`, `lookPath`) and the entry point (`cmd/spacedock/main.go` → `cli.Run(args, stdout, stderr) int`) are preserved — cobra is wired *inside* `Run`, so the package's public surface and exit-code contract are unchanged.

- **Root command** owns the tagline + grouped help template (AC-1). cobra command *Groups* (`cobra.Group{ID:"launch"...}`) back the Launch/Setup/Workflow grouping; a custom `UsageTemplate`/`HelpTemplate` (or a `SetHelpFunc`) renders the terse one-liners + footer and suppresses cobra's default `[flags]`/`Available Commands`/`Use "… --help"` boilerplate that would reintroduce META-style noise. `--version` is wired via the footer + the existing `(contract N)` token (NOT cobra's default `--version` line, which would render as its own row — AC-1 bans that).
- **Reparented, behavior-preserved** (AC-5): `status` and `dispatch` become subcommands with flag parsing DISABLED at the cobra layer (`DisableFlagParsing: true` or `Args: cobra.ArbitraryArgs` + raw `os.Args` slice) so their post-subcommand argv forwards verbatim to `runStatus`/`dispatch.Run` exactly as today. `doctor` reparents with its existing `--host`/`--plugin-manifest` parsing (can stay hand-parsed inside the RunE, or move to cobra flags — either is acceptable as long as behavior is identical).
- **Renamed** (AC-2): `init` → `install` subcommand wrapping `runInit` unchanged; the literal `init` is removed from routing (falls through to unknown-command, exit 2).
- **Re-grammared** (AC-3): `claude`/`codex` become subcommands. cobra owns the spacedock flags (`--safehouse`, `--safehouse-enable`, `--safehouse-add-dirs`, `--safehouse-add-dirs-ro`, `--skip-contract-check`, `--plugin-dir`); `ArgsLenAtDash()` splits the positional task (before `--`) from the host passthrough (after `--`). This REPLACES `splitFrontDoorArgs` (`frontdoor.go:282`), which currently parses the *inverted* grammar (host flags before `--`, task after). `runClaude`/`runCodex` keep their `frontDoorArgs` struct shape (passthrough/task/hasTask/forceSafehouse/safehouseFlags/skipCheck) — only the parser feeding it changes — so the launch-assembly, safehouse-wrap, gate-relax, and resume logic stay intact.

## Test plan

All proof is exercise-and-observe through `cli.Run` or the command's `RunE`, with the existing fakes (`fakeHost`, fake `status.Runner`). Stdlib + cobra; no live host, no network. Cost: low (in-process). The riskiest mechanism is the grammar inversion (AC-3) and the verbatim-forwarding preservation (AC-5) — validate those first per the mechanism-before-comprehensive rule.

| Test | Layer | Oracle | Cost |
|------|-------|--------|------|
| Grouped jargon-free help (AC-1) | `internal/cli` | `cli.Run(["--help"])`: assert tagline + 3 group headers + 6 command names + footer present; `front door`/`contract-gated`/`META` absent; no-args & `help` byte-identical to `--help` | low |
| `install` routes, `init` gone (AC-2) | `internal/cli` | `cli.Run(["install","--host","claude"])` drives install→doctor seam (retarget `init_test`/`init_devbranch_test`); `cli.Run(["init"])` → exit 2 unknown-command | low |
| Option-2 split (AC-3) | `internal/cli` | table over the cobra split: pre-`--` non-flag = task, post-`--` = passthrough; spacedock flags consumed wherever they appear before `--` | low |
| Prompt-unswallowable regression (AC-3) | `internal/cli` | `runClaude` with `--plugin-dir <d> -- "task"` (and a trailing dangling-host-flag variant): assert built host argv's LAST token is the spacedock prompt and `--plugin-dir <d>` rides in passthrough | low |
| Per-command help detail (AC-4) | `internal/cli` | `cli.Run(["claude","--help"])`/`["codex","--help"]`/`["install","--help"]`: assert sandbox knobs, `--plugin-dir`, `--` forwarding note, `Examples` heading present | low |
| status/dispatch verbatim forwarding (AC-5) | `internal/cli` | `cli.Run(["status",<args>])` → fake `status.Runner` receives `<args>` unchanged, unreordered; existing `status_test`/`native_status_test`/dispatch tests pass | low |
| `--version` contract token intact (AC-5) | `internal/cli` | existing `TestVersion`/`TestVersionContractToken` pass unchanged through the cobra root | low |
| Launch parity preserved (AC-3) | `internal/cli` | existing `launch_parity_test` cases retargeted to the Option-2 grammar (the `--` semantics flip: host flags now ride AFTER `--`); assert safehouse wrap, resume suppression, gate-relax unchanged | low |

Test gates: `go test ./...`. No `-race` needed (no new concurrency). The `launch_parity_test`, `frontdoor_parse_test`, and `frontdoor_test` cases MUST be rewritten to the Option-2 grammar — this is a breaking grammar change, not an additive one; the existing fence-convention assertions (`host-flag-before-fence`, post-fence-task) are inverted and must be replaced, not kept.

## Design flags (raised for the gate — not re-deciding the 6)

These are detail-level choices the captain-approved design implies but does not pin; surfacing them so the implementation worker does not guess. None is a flaw in the 6 decisions.

1. **Multi-word task = one positional or join?** Under Option-2 the task is "the optional positional." `spacedock claude do the thing` (unquoted) parses as three positionals before `--`. The old grammar joined post-fence tokens with a space. Recommendation: join the pre-`--` non-flag positionals with a single space (matches the old `strings.Join`), so an unquoted multi-word task still works; pin this in AC-3's split test. If instead the design wants "exactly one quoted positional," that is a stricter rule that breaks the old ergonomics — flag for the captain. Defaulting to JOIN preserves the most behavior.
2. **`--version`/`-v` collision.** cobra reserves `-v`? No — but cobra's built-in version support renders a command row. AC-1 keeps `--version` in the footer and the `(contract N)` token; the implementation must NOT use cobra's `rootCmd.Version` auto-flag (it would add an `Available Commands`-adjacent row and a bare version string without the contract token). Wire `--version` as an explicit pre-run check or a hidden flag whose handler emits the existing `spacedock %s (contract %d)` line.
3. **Unknown-command exit code stays 2.** The current hand-rolled router returns exit 2 for unknown commands (`cli.go:69`, `TestUnknownCommand`). cobra's default unknown-subcommand path returns exit 1. The implementation must preserve exit 2 (set `SilenceErrors`/`SilenceUsage` and map the unknown-command error, or override) so `TestUnknownCommand` and the `init`-is-gone negative test (AC-2) hold.

## Notes
- **Folds in** the launcher-arg-hardening (the `--plugin-dir $pwd`-swallow bug dies via Option 2). Drop that item from `cli-ergonomics` (xd); xd keeps workflow auto-discovery + remaining actionable-errors. (Confirmed: xd's scope/non-goals already list `spacedock claude`/`codex` as owned elsewhere — the swallow-hardening folds here cleanly.)
- **Coupling to `bundle-asset-distribution`:** if bundling becomes the primary Claude lane, `install`/marketplace becomes the *codex + opt-in* path; the command surface here should not hardcode marketplace-as-default. Concretely: `install` stays a real verb (claude marketplace+doctor, codex add-prose — the lane n1 shipped), but the `claude` launch path must NOT *require* `install` to have run (today the contract gate already permits `--skip-contract-check`/`--plugin-dir` bootstrap). Do not phrase the grouped help or `install --help` as "you must run install before claude" — keep `install` as setup, not a precondition, so a future embedded `--plugin-dir` bundle lane can become the default Claude path without a help/command rewrite. Coordinate ordering with that entity (it sequences AFTER this cobra redesign per its own notes).
- **Contract assessment — NO `CONTRACT_VERSION` bump needed.** Confirmed by grep: the FO/ensign skills depend ONLY on `spacedock --version`, `spacedock status …`, `spacedock dispatch build`/`show-stage-def`, `spacedock doctor`, and `spacedock claude`/`codex` (`first-officer-shared-core.md`, `claude-first-officer-runtime.md`, `debrief/SKILL.md`, `commission` templates). The `init`→`install` rename does NOT touch the skill-invoked set — `install`/`init` appears in NO skill or agent file (it is a human setup verb only). AC-5 pins that the skill-depended command set keeps unchanged argv behavior. So this is CLI ergonomics, not an observable-surface contract change: stay on 0.19.x, no CONTRACT_VERSION bump. (If a future revision of this design renamed/removed `status`/`dispatch`/`doctor`, THAT would be a contract concern — it does not.)
- Substantial + breaking → its own entity (not folded into xd).

## Stage Report: ideation

- DONE: The captain-APPROVED design is treated as DECIDED — formalized the 6 decisions into entity-level ACs + a test plan; no re-litigation.
  The 6 decisions (tagline, cobra, grouped jargon-free help, init→install, Option-2 passthrough, per-command help) map to AC-1..AC-5; flagged 3 detail-level choices the design implies but does not pin (task join, --version wiring, unknown-cmd exit code) under "Design flags" — none is a flaw in the 6.
- DONE: Acceptance criteria are entity-level end-states with concrete reproducible tests.
  AC-1 help golden/substring+absence (grouped, jargon-free, no META, banned `front door`/`contract-gated`); AC-2 `install` routing + retargeted install-behavior + `init`-gone negative; AC-3 Option-2 split (ArgsLenAtDash) + prompt-unswallowable regression; AC-4 per-command help substring; AC-5 skill-depended command set preserved (verbatim status/dispatch forwarding + contract token). Each names a `cli.Run`/RunE oracle, no greps-as-proof.
- DONE: Test plan + scoping names the cobra migration surface, confirms status/dispatch behavior preserved, states the bundle-asset coupling and the xd launcher-hardening drop.
  Cobra surface pinned in `internal/cli` (cli.go `run()` switch → cobra tree; root grouped help; status/dispatch reparented with flag-parsing disabled for verbatim forwarding; init→install; claude/codex re-grammared via ArgsLenAtDash replacing `splitFrontDoorArgs`); coupling note hardened (install must not be a precondition so a future embedded `--plugin-dir` bundle lane can become default); xd swallow-hardening folds here (confirmed xd already scopes claude/codex out); contract: NO bump (grep-confirmed skills never call `init`/`install`).

### Summary

Formalized the captain-approved cobra/CLI redesign into five entity-level ACs (AC-1 grouped jargon-free help, AC-2 install rename, AC-3 Option-2 unswallowable-prompt passthrough, AC-4 per-command help, AC-5 skill-command-set preservation) each with an in-process `cli.Run` behavioral oracle and a low-cost test-plan row. Grounded the cobra migration surface against the real `internal/cli` code: the hand-rolled `run()` switch becomes a cobra tree, `splitFrontDoorArgs`'s inverted `--` grammar is replaced by `ArgsLenAtDash` (the source of the unswallowable-prompt property), and `status`/`dispatch` reparent with flag-parsing disabled to forward verbatim. Confirmed by grep that NO skill/agent calls `init`/`install`, so the rename is not a contract change — no CONTRACT_VERSION bump, stay on 0.19.x; flagged three implied-but-unpinned detail choices (multi-word task join, `--version` footer wiring vs cobra's auto-flag, unknown-command exit 2) for the gate rather than guessing.
