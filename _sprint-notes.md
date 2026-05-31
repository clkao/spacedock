# Sprint notes — pending sprint-end actions (FO)

## AT SPRINT END (all entities done): parallel antipattern reviews
Captain directive (2026-05-30): when the sprint completes, dispatch PARALLEL reviews
for antipatterns — over-abstraction, over-engineering, tautological/grep tests — with
TWO senior personas: a senior STAFF SOFTWARE ENGINEER and a senior AI ENGINEER.
Read-only adversarial audits over the merged result; report findings before declaring
the sprint done.

## Deliverable-principles encoding (proposal pending disposition)
docs/dev/_proposals/encoding-deliverable-principles.md — senior-eng proposal for
encoding the four principles (no doc-only; behavioral-not-grep; enforce-in-code;
spike-in-ideation) into the workflow README + operating contract, with a code guard
(`status --validate` self-oracle lint + terminal-PASSED `--set` guard). Captain to
decide: file as a tracked contract-hardening entity (recommended — the code guard is
the behavioral oracle) vs bank for the next refit.

## TDD correction + PORTABILITY principle (captain, 2026-05-30)
The template-tdd-comparison workflow concluded "rely on global CLAUDE.md" for TDD.
SUPERSEDED — that is WRONG: CLAUDE.md is per-user/private; a clean-room self-hosted
session (the sprint goal) has no such file. Corrected position:
- Encode TDD authoring discipline in the PORTABLE shipped contract — the
  `spacedock:ensign` operating contract that governs every dispatched worker — NOT in
  global CLAUDE.md. (The gate still enforces behavioral-proof P1/P2; TDD is the
  worker-side authoring rule in the ensign skill.)
- Name the generalizing meta-principle PORTABILITY / SELF-SUFFICIENCY: the shipped
  contract must assume nothing from the user's environment (no global CLAUDE.md, no
  Python-on-PATH, no plugin-private paths). Same family as the Python-free and
  zero-plugin-private-path goals. The comparison workflow's lenses never questioned
  CLAUDE.md universality — a portability blind spot (argues for the end-of-sprint
  senior-eng antipattern reviews).
- Refit scope now: P1/P2/P4 wording + P1 code guard + ergonomics snippets (## Out of
  scope slot, promoted ## Problem/## Proposed approach/## Test plan headings, native
  --next doc) + a TDD authoring line in the ensign contract + a PORTABILITY principle
  in the operating contract + a scan for other global/environment-reliance leaks. ALL
  in the shipped contract. One refit.

## Go-binary dependency policy correction (captain, 2026-05-30)
Zero-dep was a PYTHON artifact: the `claude-team`/`status` script shipped as source
inside the skill and ran in the user's interpreter, so it could assume no `pip install`.
The compiled Go binary links deps at build time (user installs nothing) — that rationale
does NOT carry over. Correct policy for the binary: PREFER well-tested libraries for
common functionality (frontmatter/YAML, etc.); do not reimplement common stuff. Hand-roll
ONLY where a contract demands it:
- (a) byte/value PARITY with the Python oracle — BOOTSTRAP-ONLY; dies when the oracle is
  retired (native-dispatch-helper is forced to hand-roll now for exactly this).
- (b) byte-PRESERVATION of unknown frontmatter fields through `--set`/`--archive` — DURABLE
  (yaml.v3 Marshal normalizes; the MUTATOR likely stays custom; the READER can move to a
  library/yaml.Node once parity no longer binds).
FOLLOW-UP: revise AGENTS.md line 10 ("use the standard library unless a dependency removes
real complexity") for the binary post-bootstrap, and re-evaluate the frontmatter READER for
a library approach once the Python oracle is gone. (Not AGENTS.md itself = scaffolding → a
tracked change; candidate to fold into a post-bootstrap cleanup, not mid-sprint.)

## Parity-with-Python is a migration scaffold, not a long-term goal (captain, 2026-05-30)
Byte/value parity with the Python oracle is the SAFE MIGRATION tactic (proves each native
command is a faithful drop-in so Python can be removed with no behavior change). It is NOT a
permanent design goal — enshrining Python's non-standard quirks (line-hack frontmatter parser,
idiosyncratic error strings) in Go forever is wrong where a standard is better. Arc:
- NOW (bootstrap): match Python byte/value-for-value (de-risks the swap). native-dispatch-helper
  stays parity-focused mid-flight; its REAL byte-contract is the DISPATCH-SPEC output (the ensign
  consumes exact bytes), not Python-quirk parsing.
- POST-PYTHON (oracle retired): revisit parsing for STANDARD COMPLIANCE (real YAML via a library,
  standard error idioms) with DELIBERATE, DOCUMENTED divergences from the retired Python where the
  standard wins. Migration check: confirm live entities still parse (simple key:value, likely fine).
- KEEP regardless: the byte-PRESERVATION contract (unknown fields + order survive --set/--archive)
  — format durability, not a Python quirk.
Same family as the dep-policy correction above → one post-bootstrap "parsing modernization"
follow-up (file as an entity at bootstrap graduation; sprint-note for now since it is off the
critical path).

## Two-origin / distribution-remotes follow-up (captain, 2026-05-30)
Surfaced when scoping spacedock-packaging to the spacedock-v1 side. Three coupled items,
post-bootstrap / release-time (none block the current sprint):
- **Push `next` to the marketplace repo.** Add a remote to `~/git/spacedock` and push a
  `next` branch (the marketplace's declared repo is `clkao/spacedock`; the local clone's
  remotes are forks). This is the prerequisite for spacedock-packaging's DEFERRED half:
  the `~/git/spacedock` authoritative `.codex-plugin/plugin.json` `requires-contract` edit
  + cross-repo drift test + the `@next` live pre-release install verification.
- **State repo gets its own origin as a separate ORPHAN branch.** `docs/dev/.spacedock-state`
  is a separate local git repo today with no remote. Give it a remote — an orphan branch
  (state history is independent of code history) — so the live workflow state is persisted
  /shareable distinctly from the code.
- **Iron out the two origins.** The code repo (spacedock-v1 origin) and the state repo
  (orphan-branch origin) are two separate remotes/histories. Coordinating them is a
  follow-up — ESPECIALLY the future pr-merge mod (which pushes branches + opens PRs and
  would need to know which origin/branch each artifact targets). pr/mod flow is out of
  scope for THIS bootstrap workflow (README says so), so this is a post-bootstrap concern.

## Roborev descoped from this sprint (captain, 2026-05-30)
roborev-validation-hook (ng) is OUT of this sprint — it is a larger DEV-WORKFLOW
improvement (incremental commit-review feeding the validation gate), not the core
product build, and is sandbox-blocked (its daemon needs HOME relocation; claude-code is
the only healthy review agent here). The BUILD stays deferred. UPDATE (captain,
2026-05-30): its OPERATING-MODEL ideation is dispatched now as a research SPIKE — decide
which integration model fits: (1) alongside docs/dev, incremental review as implementation
proceeds (per-commit or chunked-logical-commits) feeding the validation gate; vs (2) as
part of pr-mod (the PR workflow outside the local spacedock workflow), where the FO
re-dispatches to address review feedback and/or finalize merge. Model #2 ties into the
two-origin / pr-mod follow-up above. FO lean (spike decides): #2.

## Packaging validation — HELD (resume next session, captain wants a deeper dive)
spacedock-packaging is at status=validation, branch spacedock-ensign/spacedock-packaging
(worktree .worktrees/spacedock-ensign-spacedock-packaging). The staff audit (watdl8niq)
REJECTED with TWO material issues, both RUN-confirmed; the validation ensign was still
running at session end (dies at teardown — its findings only add). The core was confirmed
sound (compare math, 5 verdicts exit 0/1/1/1/0, safe front-door paths block, 389 green,
cross-repo deferral held). Resume by routing these two fixes (then re-validate → merge):
1. FRONT-DOOR GATE HOLE (serious): a host reporting an installPath to a dir LACKING
   .claude-plugin/plugin.json makes `spacedock claude` exit 0 and LAUNCH into a session with
   an unresolvable plugin. Root cause: host_exec.go ResolveManifest returns a non-empty path
   without checking the file exists; doctor.go RunDoctor maps missing-file → non-fatal
   no-plugin-found at exit 0; frontdoor.go gateHost only special-cases manifestPath=="" .
   FIX: gateHost must reject the NoPluginFound VERDICT regardless of how the path arrived
   (inspect the verdict, not the exit code), OR ResolveManifest verifies file existence.
   Uncovered by frontdoor_test.go (fakeHost only sets "" or an existing fixture path).
2. CODEX RESOLVER non-functional: ResolveManifest shells `codex plugin list --json`, which
   real codex 0.132.0 REJECTS (exit 2, "unexpected argument --json"). So `spacedock codex` +
   `spacedock doctor --host codex` always exit 1 (loud, not a silent hole). The cycle-2 spike
   marked codex commands [SYNTAX]-only; this shipped unverified. FIX: use codex's supported
   listing (no --json) + [RUN]-verify against installed codex, OR honestly scope codex
   resolution as not-yet-functional (Codex is already version-gate+prose-only for launch).
Polish (non-blocking): corrupt-JSON manifest → bare `error:` (6th outcome, loud, fine);
TestDispatchBlockUsesNativeBuild comment mislabels AC-2 (it is the AC-5b oracle).
Also: a detached audit checkout .worktrees/audit-spacedock-packaging is removable.

## Debrief notes
- external-tracker-checkpoint shipped PASSED but AC-6 was a prose self-oracle (the
  doc-only antipattern) — kept as the live example that motivated the principles.

## Polish carried from tq (spacedock-packaging) validation audit (2026-05-31) — fold into codex-safehouse-launcher (be)
The cycle-2 staff audit was CLEAN (no material); two Polish items, both in the codex resolver path (`internal/cli/host_exec.go`), to address when the codex launcher (A′) touches that file:
1. LATENT BUG — `latestVersionDir` (host_exec.go ~:121-139) picks the LEXICALLY greatest cache dir name, not semver-greatest, so with stale dirs `0.9.0` + `0.10.0` it picks `0.9.0` (older), contradicting its own comment. Not a launch hole (still routes through ManifestVerdict→gate) and currently UNREACHABLE (real codex keeps one version dir). Bites only once `requires-contract` ships AND a 9→10 rollover leaves a stale dir. Fix: semver-aware compare or a doc note.
2. TEST GAP — no unit tests for `latestVersionDir` / `codexEntryInstalled` / `codexHome` / the cache-path + degradation branches of `resolveCodexManifest`; the sole codex resolver test is a single-version happy-path integration test. The lexical bug above has no guarding test.

## Launcher-slice ideation gate decisions (captain, 2026-05-31)
Approved A′/B/C ideation gates with these decisions:
1. **Codex no-`.safehouse` → PLAIN codex, NO bypass** (option b, NOT the ensign-recommended refuse-to-launch). With no `.safehouse`, `spacedock codex` launches plain `codex <fo-prompt>` keeping codex's own sandbox; the `--dangerously-bypass-approvals-and-sandbox` flag is emitted ONLY on the `.safehouse`-present path. Symmetric with claude's fallback. (codex-safehouse-launcher AC must reflect b.)
2. **Repo migrates to `spacedock-dev/spacedock`** (NOT clkao/spacedock@next as earlier assumed). This sets: module path `github.com/spacedock-dev/spacedock` (rewrites every import + the jf ldflags target `…/internal/cli.Version`); release origin = spacedock-dev/spacedock; formula url = spacedock-dev/spacedock releases; same org as the homebrew-tap (spacedock-dev/homebrew-tap). The migration is a captain-coordinated repo move + a mechanical module-path rewrite. **jf/rg/D implementation is GATED on this migration** (they key off the final origin/module-path); dispatching against the old path = rework. codex (be) implementation is migration-independent (its imports get rewritten uniformly by the migration) → proceeds now.
3. **Formula license = Apache-2.0** (rg).
4. **safehouse install hint = point to safehouse's docs/site** (not a pinned command) — rg formula caveats + README link out.

## Launch-parity spec from captain's real `ps` (2026-05-31) — the ground truth for "replace ALL my custom launch"
Captain dumped real invocations. COVERED by shipped/in-flight: safehouse wrap + bypass/skip-perms; `--safehouse-enable=` (ssh/docker/keychain/all-agents) and `--safehouse-add-dirs=` (sandbox-flag-passthrough, 2y, ideation-done); base prompt; claude `--resume`-family suppression; claude passthrough of `--model`/`--resume <id>`.
GAPS the ps reveals (NOT yet covered — the remaining "replace all custom launch" work):
1. **Custom/task prompt (HIGH, claude+codex).** Captain frequently launches with the base prompt PLUS a task suffix: e.g. claude `… You totally got this… Triage both work and personal!`; codex `… Assume $spacedock:first-officer… Engage with literature workflow refresh, search arxiv…` / `… Engage ops workflow, perform daily check…`. The launcher must let the operator pass a task that is APPENDED to the base prompt (bare → base only; +task → base+task; resume → none). Today the prompt is a fixed const with no override.
2. **codex `resume <id>` SUBCOMMAND (MED-HIGH).** `safehouse … codex --dangerously-bypass-approvals-and-sandbox resume <uuid>` — codex resume is a subcommand (not a flag). When resuming, NO prompt is appended. Current codex appends prompt unconditionally (resume deferred at ideation). Needs codex-resume detection.
3. **codex bare / no prompt (MED).** `safehouse --enable ssh codex --dangerously-bypass-approvals-and-sandbox` (no prompt at all) — a bare interactive launch.
4. **`--plugin-dir` dev-mode (HIGH, claude).** Almost every claude line has `claude --plugin-dir /Users/clkao/git/spacedock` (and sometimes a 2nd `--plugin-dir /…/noteplan-plugin`) — loads the LOCAL plugin checkout (dev mode). Forwarded as passthrough today, BUT it interacts with the contract gate: using `--plugin-dir` means the local plugin, so the gate should accept it (or auto-`--skip-contract-check`). Multiple `--plugin-dir` allowed. This is the captain's PRIMARY dev workflow.
5. **add-dirs multiplicity.** Captain uses MANY `--add-dirs` per launch — `--safehouse-add-dirs=` must translate to repeated `--add-dirs` (or safehouse's multi-path form).
6. **No-`--agent` claude lines (2,4) are NON-spacedock** (plain claude for other repos) — OUT of scope for `spacedock claude`.
Disposition: fold these into a comprehensive launch-parity entity (expand 2y sandbox-flag-passthrough, or a sibling). The prompt-override UX is the one open design decision.

## Recalibrated sprint goal (captain, 2026-05-31, authoritative)
1. **Replace my custom launcher** — `spacedock claude/codex` replace the hand-typed safehouse invocations (launch-parity 2y, in implementation; fence convention for the task).
2. **Install path off `next`** — fresh install works from `spacedock-dev/spacedock@next` (migration + jf/rg/D + plugin-install-from-next + `requires-contract`).
3. **Skeleton behavior test** + **coverage matrix** (should-test × python-covers × v1-implements) — `behavior-test-skeleton-and-matrix` (8033…). NOT full port/CI; CI deferred ("when we get there").
FO has the conn for approval+merges (captain, 2026-05-31): drive autonomously, escalate only genuine forks.

## hx disposition (FO, 2026-05-31)
hx code-guards P1 (self-reference ACs) only; P2 (prose-grep tests) is the behavior-test-skeleton-and-matrix entity's job — they compose. hx adds skills/integration static invariants (AC-2 legitimate-structural; AC-3 zero-oracle = low-stakes prose-grep). hx is NOT on the launcher/install critical path; sequence LAST (it self-declares this; coordinates with zs/packaging on the FO contract) — likely next session.

## FRICTION / IMPROVEMENT log — FO dogfooding the new spacedock binary + skills (2026-05-31)
1. **State-commit confusion on NON-WORKTREE (ideation) dispatches — ROOT-CAUSED.** `.spacedock-state` is git-excluded from the main checkout but is its OWN repo. The native dispatch DOES inject the split-root `git -C {state_checkout}` state-commit guidance — but ONLY for worktree stages (`internal/dispatch/build.go:302` gates it on `if worktreePath != ""`). So IDEATION / non-worktree dispatches get NO state-commit instruction → those ensigns hit the "gitignored, couldn't commit" confusion (every confused report was ideation-stage; every worktree-stage ensign committed cleanly). Compounding: (a) the vendored ensign contract's split-root section is framed as a "Worktree Contract" (§Split-Root Worktree Contract in ensign-shared-core.md), so a non-worktree ensign doesn't see it as applying; (b) dispatched ensigns load the INSTALLED plugin contract (0.12.1), not this repo's vendored copy. NARROW FIX (queued): inject the split-root state-commit guidance for non-worktree stages too in build.go, + de-frame the contract section from worktree-only; behavioral test = a dispatch-build test asserting a non-worktree split-root dispatch carries the `git -C {state_checkout}` guidance.
6. **No workflow-side `requires-contract` + no status contract-hint (captain insight, 2026-05-31).** The state-dir/workflow doesn't declare which ensign/FO contract it needs, and `status` output emits no contract hint, so a dispatched ensign on a drifted INSTALLED contract has no signal. Broader improvement (next session): a `requires-contract`/`commissioned-by`-range on the workflow + a `status` line surfacing it (analogous to the plugin↔binary requires-contract from packaging, but workflow↔contract).
2. **Contract-gate bootstrap UX is rough.** `spacedock claude` fail-fasts (malformed-range) on the installed manifest lacking `requires-contract` — captain hit it on first real use, needed `--skip-contract-check`. The `--plugin-dir`-relaxes-gate (launch-parity) + shipping `requires-contract` (D) fix it, but first-install before requires-contract ships is a cliff. Consider a friendlier "no requires-contract yet → warn+continue in bootstrap" vs hard fail.
3. **Read-then-status--set staleness echo.** FO Read of a large entity file followed by `status --set` on it triggers the full-file cache-write echo (big context cost). The Probe discipline warns about it; Grep-only helps but entity files are large. Improvement: a `status --set` that doesn't trip staleness, or smaller entity files.
4. **dispatch build JSON via file.** `spacedock dispatch build` input must be written to a file (backticks/`<>` break heredocs) — minor but a per-dispatch step; a `--stdin-safe` or arg form could help.
5. **Worktree `spacedock` binary blocks removal.** Ensigns `go build -o ./spacedock` in the worktree → untracked binary → `git worktree remove` refuses (needs `--force` after audit). Improvement: ensigns build to a gitignored path, or the worktree gitignores `/spacedock`.
