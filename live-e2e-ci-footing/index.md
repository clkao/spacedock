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
