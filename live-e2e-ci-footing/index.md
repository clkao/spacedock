---
id: b6tef0q53k5v9d3vsga49sz4
title: Wire an approval-gated live-runtime e2e CI job (CI-E2E env) — footing for the deferred live net
status: implementation
source: FO/captain (2026-05-31) — behavior-coverage sprint follow-on; the CI-E2E* approval-gated environments exist but no v1 workflow references them
score: "0.30"
started: 2026-05-31T18:40:08Z
completed:
verdict:
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-live-e2e-ci-footing
issue:
---

The repo `spacedock-dev/spacedock` already has three GitHub **Environments with required-reviewer
approval gates** (reviewer = clkao): `CI-E2E`, `CI-E2E-CODEX`, `CI-E2E-OPUS`, plus repo secrets
`ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `HOMEBREW_TAP_TOKEN`. **No v1 workflow references them**
(`grep 'environment:' .github/` is empty) — the gates are provisioned but the door was never hung.
The two existing v1 workflows (`release.yml`, `next-publish.yml`) are auto-triggered with repo-level
secrets and no approval gate.

These environments are the landing pad for the **deferred live-runtime net** the coverage matrix
(archived `behavior-test-skeleton-and-matrix`, id `8033qbqdrh4zba10w0d34m4j`) parked as "CI when we
get there" — the live half of row 15 (does a *real* FO honor reject→reflow→keep-alive), and rows
16/17 (team fail-early live, codex packaged-agent). The behavior-coverage pair just shipped covers
the *deterministic* halves; this entity starts closing the *live* half.

## Reference

`~/git/spacedock/.github/workflows/runtime-live-e2e.yml` (the Python net, 25KB) is the proven
template: `workflow_dispatch` trigger with `model_override`/effort inputs; a `static-offline` job
plus live jobs `claude-live` / `claude-live-bare` / `claude-live-opus` (each `environment: CI-E2E`
or `CI-E2E-OPUS`, secret `ANTHROPIC_API_KEY`) and `codex-live` (`environment: CI-E2E-CODEX`, secret
`OPENAI_API_KEY`); each runs `uv run pytest --runtime claude|codex …`. v1 is Go, so the open design
question ideation must resolve is **what the v1 live test actually runs** (a `//go:build live`-tagged
Go test that shells a real `spacedock claude`/`codex` dispatch and asserts mechanical outputs? a
shell smoke driving the binary end-to-end? reuse of the Python harness against the v1 binary?).

## Scope — mechanism-first (smallest end-to-end proof FIRST)

Per "validate the smallest end-to-end exercise of the riskiest path first": this entity wires **ONE**
approval-gated live job (`claude`, `environment: CI-E2E`) that runs the **smallest meaningful live
dispatch→ensign→stage cycle** and asserts a real mechanical output — proving the whole mechanism
(env gate + API-key env secret + live runtime + a non-mock behavioral assertion + artifact). The full
multi-tier matrix (codex/opus/bare, porting all Python live tests) is the **extension roadmap**, not
this entity.

## Acceptance criteria (hardened)

Each AC names a property of the finished entity, not a stage action, and how it is verified.

**AC-1 — The repo carries a `workflow_dispatch` workflow `.github/workflows/runtime-live-e2e.yml`
whose single live job declares `environment: CI-E2E`, reads `ANTHROPIC_API_KEY` from
`secrets.ANTHROPIC_API_KEY`, and cannot reach its live step until a required reviewer approves the
deployment.** Verified by: (offline/static) the YAML declares `environment:\n  name: CI-E2E` on the
`claude-live` job and `ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}` in that job's `env:`, and
a `Check required secret` step fails fast when the key is empty; (captain-run) a real
`workflow_dispatch` run shows the `claude-live` job parked in `waiting` at the CI-E2E gate while the
offline job runs to completion, and the live step starts only after the captain approves the
deployment.

**AC-2 — The live job runs a Go live test (`//go:build live`-tagged) that shells the v1 binary's real
front door `spacedock claude … --plugin-dir <repo>` headless to drive an actual FO→ensign→stage
dispatch cycle against a fixture workflow, then asserts the same anchored mechanical outputs as the
deterministic `internal/ensigncycle` skeleton (an appended `## Stage Report:` section with a `- DONE:`
marker and `### Summary`, no checkbox-bullet form, and a path-scoped git commit that names only the
entity) — not a mock, not a prose-grep.** Verified by: (captain-run) the gated `go test -tags live`
step exits 0 with the anchored assertions green over a real model run; the test fails red if the
real cycle produces a malformed stage report or a non-path-scoped commit (the same negative controls
the skeleton pins deterministically — `internal/ensigncycle/cycle_test.go:186`).

**AC-3 — The default offline test surface runs with no API key and no `environment:`, and the
`live`-tagged test is the only API-spending, gate-bearing surface.** Verified by: (offline/static)
the offline job runs `go build ./...` + `go test ./...` (default build tags) with no `environment:`
and no secret in its `env:`, and the live test compiles out of the default suite under the
`//go:build live` tag (spike-proven: a `live`-tagged `TestLiveOnly` is excluded from `go test ./...`
and included only under `go test -tags live`); only the `claude-live` job carries `environment:
CI-E2E` + `ANTHROPIC_API_KEY`.

## Out of scope (extension roadmap, not this entity)
- The full 4-tier matrix: `codex-live` (CI-E2E-CODEX), `claude-live-opus` (CI-E2E-OPUS), bare-team tier.
- Porting all Python live tests (test_gate_guardrail, test_rejection_flow, test_feedback_keepalive, …).
- Notarization / release-lane concerns (separate from the live-e2e net).

## Notes — higher-stakes; FO will not silently auto-approve
This entity writes a real CI workflow that spends API budget and uses approval-gated production
secrets. The implementation deliverable is `.github/workflows/*.yml` (+ a live-test entrypoint), and
validation requires a **captain-triggered** gated run (the sandbox cannot trigger Actions, approve a
gate, or spend API budget). The FO will bring the ideation design to the captain rather than
auto-approving, and route the live verification to the captain.

## Design — resolved open question: what the v1 live test RUNS

The Python net (`runtime-live-e2e.yml`) launches `claude -p <prompt> --plugin-dir <repo> --agent
spacedock:first-officer --output-format stream-json` headless, captures the JSONL transcript, and
asserts mechanical **tool-call shapes** parsed out of the log (`scripts/test_lib.py:run_first_officer`
+ `LogParser`). v1 is Go, so the natural — and smaller — analog is to assert the **on-disk
mechanical outputs** the existing deterministic skeleton already pins, sourced from a real run instead
of a scripted Go ensign.

**Chosen entrypoint: a `//go:build live`-tagged Go test (`internal/ensigncycle/live_test.go`) that
shells the v1 binary's real front door `spacedock claude` headless.** Concretely the test:

1. Stages a fixture workflow + a flat entity in the initial stage (the exact `stageFixture` shape the
   skeleton already builds — `internal/ensigncycle/cycle_test.go:64`), git-init'd.
2. Shells the **real front door**: `spacedock claude --plugin-dir <repo-root> --skip-contract-check
   -p "<drive-the-entity-to-done task>" --permission-mode bypassPermissions --output-format
   stream-json --verbose --model haiku` (cwd = fixture root, `ANTHROPIC_API_KEY` inherited from the
   job env, `CLAUDECODE` unset). The front door execs exactly `claude --agent
   spacedock:first-officer -p … --plugin-dir … --output-format stream-json …` — proven
   deterministically by `internal/cli/frontdoor_test.go:62` (passthrough forwards `-p`, `--plugin-dir`
   relaxes the contract gate per `cli/frontdoor.go:109`). This is the v1 binary genuinely driving a
   real model through the dispatch→ensign→stage protocol.
3. Reads back the fixture entity + git log and runs the **same anchored assertions** the skeleton
   uses (`stageReportHeading`, `doneMarker`, `### Summary`, `!checkboxBullet`, `commitNameOnly == [the
   entity]`). The assertion vocabulary is reused verbatim; only the producer changes (real runtime vs
   scripted Go ensign).

**Why this is the SMALLEST meaningful live mechanism-proof.** It exercises the full live stack the
deferred net cares about — real binary front door + real plugin load + real model + real
dispatch→ensign→stage cycle + a real path-scoped state commit — yet asserts the *minimal* mechanical
contract (stage-report shape + path-scoped commit) that the deterministic skeleton already proved is
the right go-red anchor (not the prose-grep trap). It reuses the skeleton's fixture and assertion
helpers rather than inventing a parallel harness, and it ports ZERO of the multi-behavior Python suite
(rejection/keepalive/gate-guardrail) — those are the explicit extension roadmap. Driving the FO
(rather than dispatching a bare ensign) is deliberate: it proves the *whole* mechanism end-to-end,
which is the footing the deferred row-15/16/17 net builds on.

### Alternatives considered and rejected
- **Reuse the Python pytest harness against the v1 binary.** Rejected: drags the entire `uv`/pytest
  stack + `test_lib.py` into a Go repo for one test; the on-disk assertions are simpler and already
  exist in Go.
- **A standalone shell smoke driving the binary.** Rejected: no anchored-regex go-red discipline, no
  reuse of the proven skeleton assertions, harder to keep honest (prose-grep trap risk).
- **Dispatch a bare ensign directly (skip the FO).** Rejected as the footing: it would prove a
  thinner slice than the deferred net (which is about real FO behavior). Kept as a possible
  cost-reduction knob if the full-FO run proves too budget-heavy at validation — flagged for the
  captain, not pre-decided.

## Spike — riskiest path proven LOCALLY (no API budget spent)

The riskiest unknown is **build-tag isolation + headless front-door shape**: does a `live`-tagged Go
test stay out of the secret-free offline suite yet run under `-tags live`, and does `spacedock claude`
actually forward the headless flags? Both proven locally without spending the gated budget:

- **Build-tag isolation (the AC-3 mechanism).** A throwaway module with a `//go:build live`
  `TestLiveOnly` + a plain `TestRegular`: `go test ./...` ran ONLY `TestRegular` (live excluded);
  `go test -tags live ./...` ran BOTH. So the live test compiles out of the offline job's default
  suite and is reachable only on the gated job's `-tags live` invocation. PROVEN.
- **Headless front-door shape (the AC-2 launch mechanism).** `internal/cli/frontdoor_test.go:62`
  already proves deterministically that `runClaude(["-p","do the thing"])` emits argv `claude --agent
  spacedock:first-officer -p "do the thing" <bootstrap>`, and `--plugin-dir` relaxes the contract gate
  (`cli/frontdoor.go:109`). So the planned `spacedock claude --plugin-dir … -p … --output-format
  stream-json` shells exactly the headless `claude` invocation the Python net uses. PROVEN (existing
  green test).
- **Assertion logic (the AC-2 go-red discipline).** The on-disk anchored assertions the live test
  reuses are already proven to go red on a broken cycle by the deterministic skeleton's two negative
  controls (`internal/ensigncycle/cycle_test.go:186` — checkbox-bullet report + renamed emit line).
  PROVEN (existing green test, 485/486 of the offline suite green locally; the lone failure is the
  pre-existing `TestCodexResolveManifestAgainstInstalledHost` env artifact — a broken local `codex`
  CLI — which `Skip`s on a clean runner).

**Single gated, captain-run step (FLAG-NOT-FAKE).** The one step the sandbox CANNOT run is the actual
live model dispatch inside `go test -tags live` (it spends `ANTHROPIC_API_KEY` budget behind the
CI-E2E approval gate). Everything around it — the fixture build, the front-door argv shape, the build
-tag exclusion, the anchored assertions' go-red behavior — is proven locally above. The live step is
designed so the captain triggers the `workflow_dispatch`, approves the CI-E2E gate, and observes the
single `go test -tags live` step go green/red.

## Workflow design (implementation target — not committed in ideation)

A new `.github/workflows/runtime-live-e2e.yml`, `workflow_dispatch`-only (NO `pull_request_target`
for this footing — the Python net's fork-PR security model is out of scope; the captain triggers
manually). `permissions: contents: read`.

- **`offline` job** (no `environment:`, no secret): `actions/checkout@v4`, `actions/setup-go@v5`
  (go 1.22), then `go build ./...` and `go test ./...` (default tags — the live test compiles out).
  This is the AC-3 secret-free gate and the `needs:` predecessor of the live job.
- **`claude-live` job** — `needs: offline`, `environment:\n  name: CI-E2E`, `env: ANTHROPIC_API_KEY:
  ${{ secrets.ANTHROPIC_API_KEY }}` (+ `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS`, `DISABLE_AUTOUPDATER`).
  Steps: `Check required secret` (fail fast on empty key — mirrors the Python template), checkout,
  setup-go, setup-node (Claude Code needs node), `Install Claude Code` (`curl … claude.ai/install.sh`,
  optional `claude_version` input pin), `go build -o ./spacedock ./cmd/spacedock` (the binary the test
  shells), git identity config, then `go test -tags live -run TestLiveEnsignCycle ./internal/ensigncycle/ -v`
  with an optional `model`/`effort`/`test_selector` `workflow_dispatch` input surface mirroring the
  Python net. Upload the fixture-run artifacts on `always()`.

`workflow_dispatch` inputs (mirroring the Python net, scoped to one job): `claude_version` (pin
installer), `model` (default `haiku` — cheapest live proof), `effort` (default `low`). No
codex/opus/bare inputs — those arrive with the extension matrix.

## Test plan

| Surface | Proof | Where | Cost |
|---|---|---|---|
| AC-1 wiring (env gate + secret ref + fail-fast) | static YAML inspection + a Go/`actionlint`-style assertion is overkill; a small parse-the-YAML check or manual review confirms `environment: CI-E2E` + `secrets.ANTHROPIC_API_KEY` + the `Check required secret` step | implement stage | $0 |
| AC-1 gate behavior (job parks in `waiting` until approved) | **captain-run** `workflow_dispatch`: observe `claude-live` in `waiting` while `offline` completes; live step starts only post-approval | captain | API budget (1 live run) |
| AC-2 live mechanism (real cycle → anchored on-disk outputs) | **captain-run** `go test -tags live` green over a real `haiku` run; reuses the skeleton's anchored assertions | captain (the gated step) | folded into the same 1 live run |
| AC-2 go-red discipline | already proven deterministically by the skeleton's negative controls — no extra live cost | `internal/ensigncycle/cycle_test.go:186` (existing) | $0 |
| AC-3 offline secret-free + tag exclusion | `go test ./...` green with no key; `live` test excluded under default tags (spike-proven), included under `-tags live` | implement stage + local | $0 |

Estimated validation cost: a SINGLE captain-triggered live run on `haiku`/`low` (the cheapest model)
proves AC-1 gate behavior + AC-2 live mechanism together. Everything else is $0 offline/local. The
implement stage writes the workflow YAML + `internal/ensigncycle/live_test.go`; the FO routes the one
gated run to the captain (no silent auto-approve).

## Stage Report: ideation

- DONE: Study the Python reference runtime-live-e2e.yml AND resolve the OPEN design question: what does the v1 (Go) live test actually RUN end-to-end? Name the concrete entrypoint and justify why it is the SMALLEST meaningful live mechanism-proof.
  Resolved: a `//go:build live`-tagged Go test (`internal/ensigncycle/live_test.go`) that shells the real front door `spacedock claude --plugin-dir <repo> -p … --output-format stream-json` to drive a live FO→ensign→stage cycle, then reuses the deterministic skeleton's anchored on-disk assertions (stage-report shape + path-scoped commit). See "Design — resolved open question".
- DONE: Spike the riskiest unknown FIRST: prove the chosen live-test entrypoint can drive a real dispatch→completion cycle and assert a mechanical output. Prove as much LOCALLY as possible WITHOUT spending the gated API budget; clearly mark the single gated live step as captain-run. Observe the scaffolding work before committing.
  Build-tag isolation PROVEN locally (throwaway module: `live`-tagged test excluded from `go test ./...`, included under `-tags live`); headless front-door argv shape + `--plugin-dir` gate-relax PROVEN by existing `frontdoor_test.go:62`; go-red assertion discipline PROVEN by existing `cycle_test.go:186`; offline suite 485/486 green locally (lone fail = pre-existing codex-env artifact that Skips on a clean runner). Single gated live `go test -tags live` step flagged captain-run.
- DONE: Harden AC-1/AC-2/AC-3 with the concrete workflow design (job names, environment: CI-E2E, secret wiring, the exact live entrypoint command, the offline/no-secret job) and a test plan naming exactly what the captain-triggered gated run will observe.
  ACs rewritten as end-state properties; "Workflow design" names `offline` + `claude-live` jobs, `environment: CI-E2E`, `ANTHROPIC_API_KEY` wiring, the exact `go test -tags live` entrypoint and `spacedock claude` shell command; "Test plan" table maps each AC to its proof + cost (single haiku/low captain run proves AC-1 gate + AC-2 live together; rest $0).

### Summary

Resolved the central open question: the v1 live test is a `//go:build live`-tagged Go test that shells the real `spacedock claude` front door headless to drive a live FO→ensign→stage cycle, asserting the SAME anchored on-disk mechanical outputs (stage-report shape + path-scoped commit) the deterministic `internal/ensigncycle` skeleton already proves — reusing its fixture + assertion helpers rather than a parallel harness. Proved the two riskiest mechanisms locally without spending API budget: `//go:build live` tag isolation (offline suite excludes it, `-tags live` includes it) and the headless front-door argv shape (existing green tests). The one gated, API-spending step (`go test -tags live` behind the CI-E2E approval gate) is flagged captain-run; a single haiku/low `workflow_dispatch` run validates AC-1 gate behavior + AC-2 live mechanism together, with everything else $0 offline/local.

## Stage Report: implementation

- DONE: Write internal/ensigncycle/live_test.go (`//go:build live`) per the ideation design: stage a fixture workflow + entity (reuse the stageFixture shape), shell the REAL front door `spacedock claude --plugin-dir <repo-root> --skip-contract-check -p "<drive-the-entity-to-done>" --permission-mode bypassPermissions --output-format stream-json --model haiku` (ANTHROPIC_API_KEY from env), then read back the entity + git log and assert the SAME anchored on-disk outputs the skeleton uses (stageReportHeading, doneMarker, `### Summary`, NOT checkboxBullet, commitNameOnly==[the entity]). Prove BUILD-TAG ISOLATION: `go test ./...` excludes it (default tags), `go test -tags live` includes it.
  Written (commit a28bb9d). Reuses the skeleton's package-level regexes/helpers (stageReportHeading/doneMarker/checkboxBullet/commitNameOnly, readmeNonWorktree/entityFixture/writeFile/gitInit/readFile) verbatim — only the producer changes (real model via `spacedock claude` vs scripted Go ensign). Tag isolation PROVEN: default `go test -list` shows 4 tests (no TestLiveEnsignCycle); `-tags live -list` shows all 5.
- DONE: Write `.github/workflows/runtime-live-e2e.yml`: `workflow_dispatch`-only (inputs: claude_version pin, model default `haiku`, effort default `low`); `permissions: contents: read`. `offline` job (NO environment, NO secret: checkout + setup-go + `go build ./...` + `go test ./...`). `claude-live` job: `needs: offline`, `environment: name: CI-E2E`, `env: ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}`, a fail-fast 'check required secret' step, checkout/setup-go/setup-node, install Claude Code, `go build -o ./spacedock ./cmd/spacedock`, git identity, `go test -tags live -run TestLiveEnsignCycle ./internal/ensigncycle/ -v`, upload artifacts on `always()`.
  Written (commit a28bb9d). yq-verified: `.jobs.claude-live.environment.name`=CI-E2E, secret ref=`${{ secrets.ANTHROPIC_API_KEY }}`, `Check required secret` step present; `.jobs.offline.environment`=null and `.jobs.offline.env`=null; trigger=workflow_dispatch only; `needs: offline`; top `permissions: contents: read`. Build step also exports SPACEDOCK_BIN so the test shells the freshly built binary.
- DONE: Local gates (the LIVE model dispatch is CAPTAIN-RUN — flag-not-fake): `go build ./...`, `go test ./...` (offline; live test excluded) green with REAL captured exit codes; confirm the live test COMPILES under the tag without running it; validate the workflow YAML (actionlint if available, else a parse check). Flag the single gated live run for the captain.
  `go build ./...` exit 0. Offline `go test ./...`: 488 passed, 1 failed — the lone fail is the PRE-EXISTING `TestCodexResolveManifestAgainstInstalledHost` env artifact (sandbox cannot read ~/.codex/config.toml: "Operation not permitted") that ideation flagged Skips on a clean runner; the `internal/ensigncycle/` package (my new code) is fully green (exit 0). Live test compile-only under `-tags live` (`-run '^$'`) exit 0; `go vet -tags live` clean. actionlint (go-installed) on the workflow: exit 0, no findings.

### Summary

Wired the approval-gated live-runtime e2e footing: `.github/workflows/runtime-live-e2e.yml` (workflow_dispatch-only; an offline secret-free gate job + a `claude-live` job behind `environment: CI-E2E` reading `ANTHROPIC_API_KEY`, fail-fast on empty key) and `internal/ensigncycle/live_test.go` (`//go:build live`) that shells the real `spacedock claude --plugin-dir` front door to drive a live FO→ensign→stage cycle, reusing the deterministic skeleton's anchored on-disk assertions (stage-report shape + path-scoped commit) verbatim. Everything provable offline is green with real captured exit codes: build-tag isolation (live test excluded from `go test ./...`, included under `-tags live`), the live test compiles+vets clean under the tag without running, actionlint passes the workflow, and yq confirms the AC-1 env-gate/secret wiring + AC-3 secret-free offline job. CAPTAIN-RUN (FLAG-NOT-FAKE): the single `go test -tags live` step spends ANTHROPIC_API_KEY budget behind the CI-E2E required-reviewer gate — the captain must trigger the workflow_dispatch (model default haiku/effort low), approve the CI-E2E deployment, and observe `claude-live` park in `waiting` until approved then run the live step green; the sandbox cannot trigger Actions, approve the gate, or spend API budget. CODE on branch spacedock-ensign/live-e2e-ci-footing (commit a28bb9d).

## Stage Report: validation

- DONE: Reproduce the offline/build-tag/compile proofs independently: `go test ./...` (default tags) green with REAL captured exit codes and the live test EXCLUDED; `go test -tags live -list ./internal/ensigncycle/` shows TestLiveEnsignCycle while default `-list` does NOT; the live test COMPILES under the tag without running it; actionlint (or a YAML parse) on .github/workflows/runtime-live-e2e.yml passes. DO NOT run the live API step.
  Reproduced independently: `go build ./...` exit 0; `go test ./...` exit 1 = 488 passed / 1 failed where the SOLE failure is the pre-existing `TestCodexResolveManifestAgainstInstalledHost` env artifact (`~/.codex/config.toml: Operation not permitted`, Skips on a clean runner — not my regression), `ensigncycle` package green and `TestLiveEnsignCycle` EXCLUDED; default `go test -list` → 4 tests (no TestLiveEnsignCycle), `-tags live -list` → all 5 including TestLiveEnsignCycle; `go test -tags live -run '^$' ./internal/ensigncycle/` exit 0 (compiles, "no tests to run") and `go vet -tags live` exit 0; actionlint (real, fetched via `go run github.com/rhysd/actionlint/...`) exit 0 NO findings + `yq '.'` parse exit 0. Live API step NOT run (captain-run, sandbox-blocked).
- DONE: Verify the workflow AC-wiring by inspection (yq/grep): the `claude-live` job has `environment: name: CI-E2E` + references `secrets.ANTHROPIC_API_KEY` + a fail-fast empty-secret check; the `offline` job has NO `environment` and NO secret; trigger is `workflow_dispatch` only (no pull_request/push); `permissions: contents: read`; `claude-live` has `needs: offline`; the live step runs `go test -tags live -run TestLiveEnsignCycle`.
  yq-confirmed: `.on | keys` == ["workflow_dispatch"] (no pull_request/push); top `permissions: contents: read`; `.jobs.claude-live.environment.name` == CI-E2E; `.jobs.claude-live.env.ANTHROPIC_API_KEY` == `${{ secrets.ANTHROPIC_API_KEY }}`; `.jobs.claude-live.needs` == offline; `Check required secret` step body `if [ -z "${ANTHROPIC_API_KEY}" ]; then ... exit 1`; `.jobs.offline.environment` == null AND `.jobs.offline.env` == null AND zero `secrets.` refs in the offline job (its steps are only checkout/setup-go/`go build ./...`/`go test ./...`); live step run == `go test -tags live -run TestLiveEnsignCycle ./internal/ensigncycle/ -v`.
- DONE: Verify live_test.go is a REAL behavioral assertion (not a mock/tautology): it shells the real `spacedock claude --plugin-dir ... --output-format stream-json --model <input>` front door and asserts the SAME anchored on-disk outputs as the skeleton (stage-report shape + path-scoped commit), so it would go RED on a broken cycle; confirm the single live step is the ONLY cost-bearing/gated surface. Full gates green (only ./... failure the pre-existing env-only codex test).
  `TestLiveEnsignCycle` shells the REAL front door via `exec.CommandContext(binary,"claude","--plugin-dir",repoRoot,...,"--output-format","stream-json","--model",model,"--",task)` (no model stub) and asserts the SAME strict package-level regexes/helpers as the skeleton — `stageReportHeading`/`doneMarker`/`### Summary`/`!checkboxBullet`/`commitNameOnly==[make-it-work.md]` — which it INHERITS verbatim from cycle_test.go (no local redefinition; compile under tag would error on redeclare, it didn't). Go-red discipline proven $0: re-ran the skeleton's `TestEnsignCycleGoesRedOnBrokenOutput` (same regexes) green — they reject checkbox-bullets and renamed emit lines. Only cost-bearing/gated surface confirmed: diff is exactly the 2 deliverable files (333 insertions), `live_test.go` is the ONLY `//go:build live` file, only the `claude-live` job carries CI-E2E + ANTHROPIC_API_KEY (release.yml unchanged, uses GITHUB_TOKEN/HOMEBREW_TAP_TOKEN only). Spot-check: `go build -o ./spacedock ./cmd/spacedock` exit 0 and `spacedock claude --help` passes through to real `claude` — the captain's gated step reaches a working binary.

### Summary

Independently reproduced every offline/static proof around the single CAPTAIN-RUN live step and confirms PASSED. Build-tag isolation, compile-under-tag, vet, full offline suite (488 passed; lone fail is the pre-existing codex-env artifact, not a regression; live test excluded), actionlint (real go-run, no findings), and yq AC-wiring inspection all green with real captured exit codes. live_test.go is a genuine behavioral assertion — it shells the real `spacedock claude --plugin-dir` front door and reuses the skeleton's strict anchored regexes verbatim (go-red discipline re-proven via the skeleton's negative controls), and the `claude-live` `go test -tags live` step is the sole cost-bearing/CI-E2E-gated surface. FLAG-NOT-FAKE: the actual gated live model dispatch was NOT run (sandbox cannot trigger Actions, approve the CI-E2E gate, or spend ANTHROPIC_API_KEY) — the captain must trigger the workflow_dispatch (haiku/low), approve the gate, and observe `claude-live` park in `waiting` then go green. Recommendation: PASSED.

## Feedback Cycles

### Cycle 1 — REJECTED (post-validation: wrong auth mechanism; not locally runnable) — captain-surfaced

Validation PASSED the offline surface, but the captain surfaced that the live test's AUTH mechanism diverges from the proven original and makes the test impossible to run locally. The original spacedock harness (`~/git/spacedock/scripts/test_lib.py:325-372`, `_isolated_claude_env`) uses a TWO-path decision tree with HOME isolation:

- **(a) operator-local:** `~/.claude/benchmark-token` non-empty → fresh temp HOME, inject `CLAUDE_CODE_OAUTH_TOKEN=<token>`, DROP `ANTHROPIC_API_KEY` (token authoritative; `claude setup-token`).
- **(b) CI:** no token file but `ANTHROPIC_API_KEY` present → fresh temp HOME, pass `ANTHROPIC_API_KEY` through (GitHub-runner path).
- **(c) neither → None** (caller skips, does NOT fatal).

v1's `internal/ensigncycle/live_test.go` hard-requires `ANTHROPIC_API_KEY` (`t.Fatal` :43-44) and uses the real (non-isolated) HOME. So it CANNOT run on an operator machine (which authenticates via OAuth/`benchmark-token`, not an API key) — `~/.claude/benchmark-token` is present here (108 bytes) yet the test refuses it — and it skips HOME isolation. The Python CI uses `ANTHROPIC_API_KEY` (path b), so v1's WORKFLOW key-wiring is correct and STAYS; the fix is the TEST's auth + HOME isolation.

**Fix (route to implementation):**
1. Port the decision tree into a small testable helper (e.g. `isolatedClaudeEnv(t)`): (a) benchmark-token → fresh `t.TempDir()` HOME + `CLAUDE_CODE_OAUTH_TOKEN` + drop `ANTHROPIC_API_KEY`; (b) `ANTHROPIC_API_KEY` → fresh HOME + passthrough; (c) neither → `t.Skip(...)`, NOT `t.Fatal`.
2. Run `spacedock claude` with that isolated env (clean HOME + chosen credential).
3. Add an OFFLINE unit test for the helper mirroring `~/git/spacedock/tests/test_test_lib_helpers.py` (fixture token file → OAuth path + key dropped; key-only → passthrough; neither → skip) — fast, no live model.
4. Confirm plugin loading works under the clean HOME (test uses `--plugin-dir`; the original stages the plugin into HOME — verify `--plugin-dir` suffices with a fresh HOME, else stage it).
5. Do NOT attempt the nested-claude live run inside the dispatched ensign (deep nesting/sandbox); the FO runs the gated live test locally with the real `benchmark-token` after the fix.

**Cleared (do NOT change):** the workflow's `ANTHROPIC_API_KEY` wiring (correct CI path b), the security-audited gate/trigger/permissions, and the anchored on-disk assertions (genuine, not tautology).

## Stage Report: implementation (cycle 1)

- DONE: Port the original's `_isolated_claude_env` decision tree into a small testable helper in internal/ensigncycle, run `spacedock claude` with that isolated env, replace the hard t.Fatal at :43-44.
  `isolatedClaudeEnv(t, realHome)` (liveenv_test.go) + pure `decideClaudeEnv` port `scripts/test_lib.py:319-375`: (a) benchmark-token → fresh t.TempDir HOME + CLAUDE_CODE_OAUTH_TOKEN + drop ANTHROPIC_API_KEY; (b) ANTHROPIC_API_KEY → fresh HOME passthrough; (c) neither → t.Skip (NOT t.Fatal). live_test.go now sets `cmd.Env = isolatedClaudeEnv(t, os.Getenv("HOME"))`; the old t.Fatal key check is gone. Code commit 46ea628.
- DONE: Add an OFFLINE unit test (plain, NOT //go:build live) for the env-selection helper mirroring tests/test_test_lib_helpers.py: token→OAuth+key-dropped+fresh HOME; key-only→passthrough+fresh HOME; neither→none/skip; fake HOME so it never reads real ~/.claude.
  liveenv_decision_test.go: TestDecideClaudeEnv (4 subtests: token-wins-drops-key, api-key-passthrough, neither→authNone, empty-token→api-key) + TestIsolatedClaudeEnvOAuthPath (CLAUDE_CODE_OAUTH_TOKEN set, ANTHROPIC_API_KEY+CLAUDECODE dropped, HOME=fresh≠fakeHome) + TestIsolatedClaudeEnvAPIKeyPath. All under default tags; 7/7 PASS, no live model. raw `-list` confirms they run in BOTH default and `-tags live`.
- DONE: Confirm plugin loading under the clean HOME; gates green with real captured exit codes (go test ./..., go vet, gofmt); live test COMPILES under -tags live and SKIPS cleanly with neither credential. Do NOT run the nested-claude live cycle.
  --plugin-dir is an ABSOLUTE repoRoot path reachable from any HOME — the original's stage-plugin-into-HOME is a weak-model discovery-pollution mitigation (#200), not a --plugin-dir-needs-HOME requirement; kept --plugin-dir as-is (YAGNI — stage into HOME only if the FO's live run shows it insufficient). Captured exit codes: `go build ./...` 0; ensigncycle `go test` 0 (15 pass incl. new helper tests); `go vet -tags live` 0; `gofmt -l` clean (0 files); compile-under-tag `-tags live -run '^$'` 0; live SKIP proven — `env -u ANTHROPIC_API_KEY HOME=<empty> go test -tags live -run TestLiveEnsignCycle` → `--- SKIP` (the message fires before any claude exec, no nested cycle). Build-tag isolation: TestLiveEnsignCycle present ONLY under `-tags live -list`, absent from default `-list`. Full `go test ./...` = ensigncycle green; the sole repo failure is the PRE-EXISTING `TestCodexResolveManifestAgainstInstalledHost` env artifact (`~/.codex/config.toml: Operation not permitted`) — proven pre-existing by re-running it on the clean stashed base (same FAIL), not a regression.

### Summary

Fixed the cycle-1 REJECT: ported the proven Python `_isolated_claude_env` two-path auth + HOME-isolation decision tree into `internal/ensigncycle` (pure `decideClaudeEnv` + `isolatedClaudeEnv(t, realHome)` wrapper) and wired `TestLiveEnsignCycle` to use it, deleting the hard `t.Fatal` ANTHROPIC_API_KEY check so an operator machine authenticates via `~/.claude/benchmark-token` (OAuth, key dropped), CI via `ANTHROPIC_API_KEY` passthrough, and a machine with neither SKIPS cleanly against a fresh empty HOME. Added a default-tag offline unit test (liveenv_decision_test.go, 7/7 green, fake HOME, no live model) covering all three branches plus the concrete OAuth/API-key env output, mirroring `tests/test_test_lib_helpers.py`. The fix is localized to live_test.go + two new test files (323 insertions) — the workflow YAML auth wiring, gate/trigger/permissions, and anchored on-disk assertions are untouched per the feedback's CLEARED list. All offline gates green with real captured exit codes; the live model dispatch stays CAPTAIN/FO-RUN (the FO runs it locally with the real benchmark-token, which now selects the OAuth path). Code commit 46ea628 on spacedock-ensign/live-e2e-ci-footing.

### Cycle 2 — FO live-run findings (auth fixed; two new issues surfaced by REAL model runs)

With the refreshed `~/.claude/benchmark-token` (extracted from the live session keychain), the FO ran the gated live cycle locally against two real models. Auth now works (OAuth path); the cycle EXECUTES end-to-end (plugin loads via `--plugin-dir` as `spacedock@inline`, `spacedock:first-officer`/`ensign` agents register). Two findings:

- **haiku — too weak (incomplete cycle).** 115s run: the FO did NOT produce the protocol outputs — no `## Stage Report`/`- DONE:`/`### Summary` appended, and the commit named `[README.md make-it-work.md]` (not path-scoped). Matches the captain's Python-land conclusion that **sonnet is the floor**.
- **claude-opus-4-8 — completed the cycle CORRECTLY, but the test asserts the wrong end-state.** 348s / $2.88: the FO booted, dispatched `make-it-work` `backlog→done` in real **team mode**, the ensign wrote its stage report + committed, the FO finalized (`verdict=passed`) and **ARCHIVED the entity to `_archive/make-it-work.md`**, then tore down the team — a textbook full cycle. The test FAILED only at `open …/make-it-work.md: no such file` because driving to the TERMINAL `done` stage **moves the entity to `_archive/`**. The live test inherited the scripted-skeleton's assertions (one in-place stage-report append + one path-scoped HEAD commit), which do NOT match a real full-FO-to-done cycle (team dispatch, multiple commits, archival).

**Fix (route to implementation):**
1. **Assert the real completed-cycle end-state.** After the run, locate the entity at the original path OR `_archive/` (flat `_archive/<slug>.md` or folder `_archive/<slug>/index.md`); assert it carries a `## Stage Report` with `- DONE:` + `### Summary` and `status: done` / `verdict: passed`. The test must still go RED on an INCOMPLETE cycle (haiku: no stage report anywhere). Replace the brittle `commitNameOnly(HEAD)==[entity]` assertion (a full cycle's HEAD is the archive/merge commit) with an invariant that fits — e.g. assert SOME commit is path-scoped to the entity, or drop the strict single-file HEAD check (the path-scoped-commit invariant is already pinned deterministically by the skeleton's `TestEnsignCycleMechanicalOutputs`; the live test's job is to prove a real model COMPLETES the cycle).
2. **Models: drop haiku; CI runs SONNET + OPUS (captain directive).** Default `SPACEDOCK_LIVE_MODEL=sonnet` (the floor). The workflow runs two gated variants: a `sonnet` job on `environment: CI-E2E` and an `opus` job on `environment: CI-E2E-OPUS` (a matrix or two jobs), each carrying `ANTHROPIC_API_KEY`. Drop the haiku default everywhere (test + workflow inputs).
3. After the fix, the FO re-runs **sonnet** locally to confirm it passes (the captain's "if it passes" gate) before pinning.

**Cleared (do NOT change):** the auth/HOME-isolation fix (cycle 1, correct), the offline helper unit tests, the security-audited gate/trigger/permissions, and the `ANTHROPIC_API_KEY` secret wiring (still the CI credential).

## Stage Report: implementation (cycle 2)

- DONE: Fix the live test to assert the REAL completed-and-archived cycle end-state (locate entity in place OR `_archive/<slug>.md` OR `_archive/<slug>/index.md`; assert `## Stage Report` + `- DONE:` + `### Summary` AND `status: done`/`verdict: passed`; must still go RED on an incomplete cycle; replace the brittle `commitNameOnly(HEAD)==[entity]` check).
  live_test.go now calls `locateEntity(root,"make-it-work")` (searches the 3 end-state paths; hard-fails if found nowhere), asserts the reused stage-report regexes PLUS `frontmatterField`(`status: done`)/`verdictPassed`(`verdict: passed`, case-insensitive — fixture does not pin casing), and `someCommitNamesOnly(...)` scans the WHOLE log for a path-scoped entity commit (HEAD is the FO archive commit on a full cycle). Go-red on the haiku incomplete shape is pinned offline: `liveassert_unit_test.go` — `TestSomeCommitNamesOnly/sibling_sweep_only_is_rejected` (the `[README.md make-it-work.md]` sweep) + `TestTerminalFrontmatterAnchors` (backlog/empty-verdict rejected). Code commit 96f766a.
- DONE: Drop haiku; default SPACEDOCK_LIVE_MODEL=sonnet; run TWO gated variants — sonnet on `environment: CI-E2E`, opus (claude-opus-4-8) on `environment: CI-E2E-OPUS` — each `needs: offline`, each with ANTHROPIC_API_KEY, each `go test -tags live`. Remove the haiku default from the workflow_dispatch `model` input and the test default.
  `claude-live` is now a `strategy.matrix.include` (`fail-fast: false`) over `{sonnet → CI-E2E}` and `{claude-opus-4-8 → CI-E2E-OPUS}`; `environment.name: ${{ matrix.environment }}`, `SPACEDOCK_LIVE_MODEL: ${{ matrix.model }}`, per-leg artifact name. yq-confirmed: matrix include = the two model→env pairs, `needs: offline`, secret ref intact, offline job env/environment still null. The `model` workflow_dispatch input is removed; the test default is now `sonnet` (live_test.go:55). Code commit 96f766a.
- DONE: Offline gates green with REAL captured exit codes; live test still COMPILES under -tags live and SKIPS cleanly with no creds; keep new entity-locate/assertion logic offline-unit-testable; actionlint the workflow. Do NOT run the nested-claude live cycle.
  `go build ./...` exit 0; `gofmt -l internal/ensigncycle/` clean (0 files); `go vet ./...` exit 0 (and `go vet -tags live ./internal/ensigncycle/` exit 0). Full `go test ./...` = `internal/ensigncycle` green (24 default tests incl. the 3 new helper tests); SOLE repo failure is the PRE-EXISTING `TestCodexResolveManifestAgainstInstalledHost` env artifact in `internal/cli` (`~/.codex/config.toml: Operation not permitted`, Skips on a clean runner — untouched by this diff). Build-tag isolation holds: `TestLiveEnsignCycle` present ONLY under `-tags live -list`, absent from default. Compile-only under `-tags live -run '^$'` exit 0; clean SKIP proven (`env -u ANTHROPIC_API_KEY HOME=<fresh>` → `--- SKIP`, fires before any claude exec). actionlint (go-run) exit 0, no findings. Nested-claude live cycle NOT run (FO re-runs sonnet locally).

### Summary

Fixed the cycle-2 findings. (A) The live assertions now match the REAL full-FO-to-done end-state instead of the scripted skeleton's single in-place append: `locateEntity` finds the entity whether it stayed in place or was ARCHIVED (flat `_archive/<slug>.md` or folder `_archive/<slug>/index.md`), the test asserts the reused stage-report shape PLUS the FO's terminal frontmatter (`status: done`/`verdict: passed`, case-insensitive), and `someCommitNamesOnly` scans the whole log for a path-scoped entity commit rather than pinning HEAD (which is the FO's archive commit on a full cycle). The new locate/scan/frontmatter logic is default-tag offline-unit-tested and proven to go RED on the haiku incomplete-cycle shape. (B) haiku is dropped: the `claude-live` job is a `fail-fast:false` matrix over sonnet (CI-E2E) and claude-opus-4-8 (CI-E2E-OPUS), each its own approval-gated deployment carrying ANTHROPIC_API_KEY; the test default model is now sonnet and the haiku `model` workflow input is removed. All offline gates green with real captured exit codes (build/gofmt/vet/actionlint/build-tag isolation/clean-skip); the lone `go test ./...` failure is the pre-existing codex-env artifact, not a regression. CLEARED items untouched: the cycle-1 auth/HOME-isolation fix, the offline helper unit tests, the security-audited gate/trigger/permissions, and the ANTHROPIC_API_KEY secret wiring. Code commit 96f766a on spacedock-ensign/live-e2e-ci-footing. The gated live model dispatch stays FO/captain-run — the FO re-runs sonnet locally to confirm the new assertions pass.

## Stage Report: implementation (cycle 3)

- DONE: Relax ONLY the verdict assertion in the live-test end-state check (live_test.go + the liveassert helper/tests): require the entity's `verdict:` field is SET (non-empty) rather than exactly `passed`. KEEP strict: `status: done`, archived (located in place or under `_archive/`), and the `## Stage Report` shape (`- DONE:` + `### Summary`, no checkbox bullet). The test MUST still go RED on an INCOMPLETE cycle (haiku produced NO stage report at all).
  One-spot semantic change: `verdictPassed` (`^verdict:\s*passed\s*$`) → `verdictSet` (`^verdict:[^\S\n]*\S.*$`) in liveassert_test.go — matches any non-empty verdict, rejects empty/whitespace-only. `[^\S\n]*` (horizontal whitespace) instead of `\s*` so it cannot consume the line break and let `\S` reach the next frontmatter line (caught by the offline unit test). live_test.go (b)-check now calls `verdictSet`; status:done/archive-locate/stage-report shape (heading+DONE+Summary+!checkbox)/someCommitNamesOnly all UNCHANGED. Go-red on the incomplete cycle stays pinned offline. Code commit ce0db36.
- DONE: Update the offline unit tests (liveassert_unit_test.go / liveassert_test.go) so they match the relaxed rule: a completed end-state with verdict set (any non-empty value) PASSES; a missing stage report / missing done-status / empty verdict FAILS. Keep them offline (default tags), fast, no live model.
  `TestTerminalFrontmatterAnchors` now asserts `verdict: PASSED`/`verdict: passed`/`verdict: done` ALL match (the sonnet-vs-opus word divergence) and `verdict:` (empty value, status:done) does NOT match — plus backlog status still rejected. Default tags, no live model; 24/24 ensigncycle tests pass.
- DONE: Offline gates green with REAL captured exit codes (go build ./..., go test ./... [live excluded], go vet, gofmt -l); live test still COMPILES under -tags live and SKIPS cleanly with no creds. Do NOT run the nested-claude live cycle.
  `go build ./...` exit 0; `go vet ./...` + `go vet -tags live ./internal/ensigncycle/` exit 0; `gofmt -l .` clean (0 files); `go test ./...` = 504 passed / 1 failed where the SOLE failure is the PRE-EXISTING `TestCodexResolveManifestAgainstInstalledHost` cli env artifact (diff touches only the 3 ensigncycle files, cli untouched — not a regression), ensigncycle package fully green. Build-tag isolation: default `-list` = 10 tests NO TestLiveEnsignCycle, `-tags live -list` = 11 WITH it. Compile-only `-tags live -run '^$'` exit 0. Clean SKIP proven: `env -u ANTHROPIC_API_KEY -u CLAUDE_CODE_OAUTH_TOKEN HOME=<fresh>` → `--- SKIP` at live_test.go:62 (fires before any claude exec), PASS. Nested-claude live cycle NOT run.

### Summary

Applied the captain's one-spot relaxation: the live-cycle end-state check now requires the entity's `verdict:` be SET (non-empty) rather than exactly `passed`, so the sonnet run (which writes `verdict: done`) passes the same as opus (`verdict: passed`) — both complete the full FO→ensign→done→archive cycle, and the exact verdict word is FO judgment that varies by model. `verdictPassed` → `verdictSet` (`^verdict:[^\S\n]*\S.*$`, horizontal-whitespace-only before the value so an empty `verdict:` line cannot satisfy it across the line break); the live test's (b)-check calls it. Everything else from cycle 2 is untouched: `status: done`, the archive-aware `locateEntity`, the stage-report shape (heading + `- DONE:` + `### Summary`, no checkbox bullet), `someCommitNamesOnly`, the sonnet+opus matrix, the auth/HOME-isolation fix, and the gate/trigger/permissions. The offline unit test `TestTerminalFrontmatterAnchors` proves the relaxed rule: `passed`/`PASSED`/`done` all match, an empty `verdict:` (with status:done) is rejected, and backlog status is rejected — so the suite still goes RED on an incomplete cycle (haiku: no stage report). All offline gates green with real captured exit codes (build/vet/gofmt/test); ensigncycle 24/24, full suite's lone failure is the pre-existing codex-env cli artifact; build-tag isolation and clean-skip re-proven; the nested live cycle was NOT run (CI runs the gated variants). Code commit ce0db36 on spacedock-ensign/live-e2e-ci-footing.
