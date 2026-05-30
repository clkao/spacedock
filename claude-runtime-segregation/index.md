---
id: zse4a3ds0x19gpdcjh7anhgs
title: Segregate Claude-specific runtime coupling (context-budget, team/standing) from the generic contract
status: ideation
source: sprint — FO/captain (native-dispatch-helper scope split)
score: "0.32"
worktree:
started: 2026-05-30T22:10:50Z
---

The `claude-team` helper carries a runtime-coupled surface that `native-dispatch-helper` deliberately scopes OUT of the native dispatch path: `context-budget` (reads Claude Code's `~/.claude/.../agent-*.jsonl` transcripts + team `config.json`), and the standing-teammate / team subcommands `spawn-standing`, `list-standing`, `show-standing` (emit Agent() specs, probe team membership, enumerate `_mods/`). These are Claude-Code-specific and not on the initial-dispatch critical path — but the FO still shells to `python3 + claude-team` for **every ensign-reuse decision** (`context-budget`), the feedback-rejection flow, and standing-teammate spawn. So a "Python-free self-hosted" handoff is, today, only Python-free for *initial dispatch*; the steady-state FO loop is not.

This entity owns that surface and the contract reorganization it implies. Two coupled questions:

1. **Subcommand disposition.** For each runtime-coupled `claude-team` subcommand (`context-budget`, `spawn-standing`, `list-standing`, `show-standing`, and `build`'s `_mods`-enumeration branch that decides whether to emit a `show-standing` fetch line), decide: reimplement native (in a Claude-Code-specific package, NOT the generic dispatch/status core), keep a thin shim, or keep Python — and justify against the cost (transcript-format coupling, per-host layout) vs the benefit (a truly Python-free FO loop).
2. **Claude-vs-generic contract split.** The operating contract is split today into a generic core (`first-officer-shared-core.md`) and host runtime adapters (`claude-first-officer-runtime.md`, `codex-first-officer-runtime.md`). Determine whether Claude-Code-specific assumptions (transcript-budget reads, team/standing semantics, on-disk team layout) have leaked into the generic core, and if so, relocate them into the Claude runtime adapter (or a Claude-specific module) so the generic core is host-neutral. The native binary's package boundaries should mirror this: generic dispatch/status logic in host-neutral packages, Claude-Code coupling behind a clear seam.

## Acceptance criteria

Sharpened by ideation 2026-05-30 (see `## Ideation design`). Every AC is an end-state property of the finished entity, falsifiable by an oracle external to this entity body. No doc-only ACs.

**AC-1 — The five runtime-coupled subcommands resolve through the native binary's `spacedock dispatch` surface (in `internal/claudeteam`, not the generic dispatch/status core), each at parity with the vendored Python oracle.**
End state: `spacedock dispatch context-budget`, `list-standing`, `show-standing`, `spawn-standing` are no longer rejected as deferred (today `internal/dispatch/dispatch.go` returns exit 2 for them), and `spacedock dispatch build` emits the `_mods`/`show-standing` fetch line.
Oracle: the three-channel parity harness (`internal/dispatch/parity_harness_test.go` idiom — native in-process vs vendored `claude-team` exec'd with pinned hermetic `HOME`, byte-compare stdout/stderr/exit after the documented `claude-team`→`spacedock dispatch` fetch-prefix rewrite) extended with a fixture `~/.claude` tree (team `config.json` + `agent-*.meta.json` + `agent-*.jsonl`) for `context-budget` and a `_mods`-bearing workflow fixture for the standing subcommands and the `build` branch. Each subcommand's success path AND its loud-failure paths (missing jsonl, missing mod, missing `standing: true`, bad model enum) are at parity. SPIKE-A pre-validated the context-budget extraction parity (85183 == 85183 on a frozen snapshot).

**AC-2 — The FO ensign-reuse path has a defined, tested behavior when no context budget is available, on both hosts.**
End state: reuse-condition-0 is a *capability* in the generic contract, not a hardcoded Claude command. When the budget probe reports the worker over budget → fresh-dispatch; when the budget source is unavailable (Claude: jsonl absent/unreadable) → fresh-dispatch (fail-safe, never silent-reuse); when the runtime declares no budget probe at all (Codex, established by SPIKE-C) → condition-0 is satisfied and the remaining reuse conditions decide.
Oracle: a behavior fixture drives `spacedock dispatch context-budget` against a fixture `~/.claude` tree with the named agent's jsonl absent and asserts the documented contract (non-zero exit / no `reuse_ok: true` → FO fresh-dispatches), plus the contract prose for the no-probe-runtime case is host-qualified and covered by the AC-3 prose oracle. This closes the steady-state-Python question with an observable contract.

**AC-3 — The generic operating-contract core (`first-officer-shared-core.md`) and the generic Go packages (`internal/dispatch`, `internal/status`) carry no UNQUALIFIED Claude-Code-specific runtime coupling; the coupling lives in the Claude adapter / `internal/claudeteam`.**
End state: every named Claude-runtime helper command/function in the generic core sits inside an explicitly host-qualified span (the `X on Codex, Y on Claude` shape already at `first-officer-shared-core.md:206`); the concrete commands move to `claude-first-officer-runtime.md`; `internal/dispatch` + `internal/status` contain no `~/.claude`/home-rooted team/transcript reads.
Oracle (two complementary structural invariants over real files, NOT a substring count):
- prose-side: a test parses the markdown structure of `first-officer-shared-core.md` and fails if a Claude-only helper token (`claude-team`, `context-budget`, `spawn-standing`, `list-standing`, `show-standing`, `member_exists`, `lookup_model`) appears in a generic-algorithm span that lacks a host qualifier — falsifiable by re-introducing the line-141 unqualified `claude-team context-budget` step and watching it fail.
- code-side: a test over `internal/dispatch` + `internal/status` source (via `go/parser`) asserts no `.claude` path-join / home-rooted transcript-read literals — falsifiable by adding such a read and watching it fail.

**AC-4 — `context_limit_for_model` resolves the opus context window by a forward family rule (opus minor ≥ 7 → 1M), not a per-release exact-name list, in both the native implementation and the vendored Python oracle.**
End state: `claude-opus-4-8` (no `[1m]` suffix) → 1_000_000; `claude-opus-4-6` → 200_000; `[1m]` suffix → 1_000_000; a hypothetical `claude-opus-4-10` → 1_000_000 without a code change. The live false-negative (every opus-4-8 ensign read against the 200k denominator) is gone.
Oracle: a fixture `~/.claude` tree stamping the agent `claude-opus-4-8` (no suffix, as the real team `config.json` does for ensigns) at a sub-600k resident count asserts `context_limit == 1_000_000` and `reuse_ok == true`; a boundary table asserts the per-model expectations above. SPIKE-B pre-validated the rule against the current buggy Python.

## Test gates

- `go test ./...`
- The extended three-channel parity harness for `context-budget`, `list-standing`, `show-standing`, `spawn-standing`, and `build`'s `_mods` branch (native vs vendored Python over `~/.claude` + `_mods` fixtures).
- The AC-2 budget-unavailable fallback behavior fixture.
- The AC-3 host-neutrality invariants (prose-side markdown-structure parse + code-side `go/parser` over the generic packages).
- The AC-4 1M family-rule fixture + boundary table.

## Notes

Sequences after `native-dispatch-helper` (`7w8w5nsa5mbc807b3jb88psv`), which defines the native dispatch surface and the package seam this entity extends. native-dispatch-helper's `build` is scoped to NON-`_mods` workflows (the self-hosted `docs/dev` path); this entity owns `build`'s `_mods`/`show-standing` fetch-line branch + the standing subcommands, closing the `_mods` parity gap.

Per the captain's note: if a design choice here rests on an unverified mechanism (e.g. exact transcript-budget parity, or whether a host-neutral seam is achievable without per-host branching), ideation must SPIKE it — run the smallest end-to-end exercise first and record the behavioral evidence — rather than assert it.

Relation to the deliverable-principles proposal (`docs/dev/_proposals/encoding-deliverable-principles.md`): that proposal's contract edits and this entity's contract reorganization both touch the operating-contract reference files; coordinate so they don't collide, and prefer the vendored-copy-first authoring direction documented there.

**Context-budget 1M-detection bug (found 2026-05-30, fold into this entity's scope).** `context_limit_for_model` returns the 200k DEFAULT for `claude-opus-4-8` ensigns: the `EXTENDED` list has `claude-opus-4-7` but NOT `4-8`, and spawned ensigns drop the `[1m]` suffix in their team-config/jsonl `model` field (only the lead keeps `claude-opus-4-8[1m]`). But since opus **4-7**, ensigns run the 1M window REGARDLESS of the dropped suffix (the suffix was only load-bearing back on 4-6). Result: `reuse_ok` false-negatives every opus-4-8 ensign at the wrong 200k denominator (a 159k-resident ensign reads as 80% when it is ~16% of 1M). Fix: a forward FAMILY RULE — parse the opus minor version and return 1M for minor ≥ 7 — so the mapping never goes stale on the next release; do NOT depend on the `[1m]` suffix or a per-release exact-name list. Applies whether context-budget stays Python or is reimplemented native. Behavioral test: a fixture stamping `claude-opus-4-8` (no suffix) asserts `context_limit` == 1_000_000 and `reuse_ok` true at a sub-600k resident count.

## Ideation design (2026-05-30)

### SPIKE evidence (forensic, run before designing)

Two unverified mechanisms gated this design; both spiked against real artifacts before any decision was locked.

**SPIKE-A — context-budget transcript parity (native Go vs Python).** Drove a throwaway stdlib-Go reimplementation of the three context-budget primitives (`find_subagent_jsonl` → `extract_resident_tokens` → `context_limit_for_model`) against the same real `~/.claude/.../subagents/agent-*.jsonl` this very ensign is writing, then against a *frozen snapshot* of it (the live file grows between reads — peak-resident is a moving target by design, not a parity defect).
- Frozen-snapshot resident extraction: Python `extract_resident_tokens` = **85183**, Go backward-scan = **85183** (exact match). `extract_runtime_models` = `['claude-opus-4-8']` both sides.
- Real usage shape confirmed: assistant entries carry `usage.{input_tokens, cache_creation_input_tokens, cache_read_input_tokens}`; resident = their sum; backward scan skips the trailing all-zero entry an overflow-dead ensign leaves. Plain JSONL, no Claude-proprietary encoding — stdlib `encoding/json` line-decode reproduces it.
- Verdict: native context-budget parity is **achievable and cheap**. No format surprise. The riskiest path is de-risked.

**SPIKE-B — the 1M-detection family rule, across the version boundary.** Implemented the forward family rule in Go (`^claude-opus-4-(\d+)`, minor ≥ 7 → 1M; `[1m]` suffix → 1M; else 200k) and ran the boundary cases against the CURRENT (buggy) Python:

| model | CURRENT Python | family rule (correct) |
|---|---|---|
| `claude-opus-4-8` | 200000 **(BUG)** | 1000000 |
| `claude-opus-4-8[1m]` | 1000000 | 1000000 |
| `claude-opus-4-7` | 1000000 | 1000000 |
| `claude-opus-4-6` | 200000 | 200000 (correct — 4-6 needs the suffix) |
| `claude-opus-4-10` (hypothetical) | 200000 (would go stale) | 1000000 (forward-safe) |
| sonnet / haiku | 200000 | 200000 |

- Reproduced the live bug end-to-end: `claude-team context-budget --name {this-ensign}` returns `context_limit: 200000` while the ensign genuinely runs the 1M window. Confirmed the root cause empirically: the real team `config.json` stamps the ensign `claude-opus-4-8` (no suffix) while the lead keeps `claude-opus-4-8[1m]`, and the real jsonl `model` field is `claude-opus-4-8` (no suffix).
- Verdict: the family rule fixes the false-negative and never goes stale; `4-6` correctly stays 200k (minor < 7), so the rule is not over-broad.

**SPIKE-C — is a host-neutral seam achievable without per-host branching?** Read the Codex FO runtime adapter end to end. Finding: the **Codex adapter has no context-budget mechanism and no standing-teammate section at all**. Its reuse flow decides reuse on "addressable + shared reuse conditions pass" via `send_input`/`wait_agent`; there is no transcript-reading budget probe, because Codex has no `~/.claude/.../agent-*.jsonl` to read. context-budget and standing-teammates are **structurally Claude-only capabilities**, not generic capabilities that Claude happens to implement first.
- Verdict: a host-neutral seam IS achievable, and the seam is a *capability boundary*, not a per-host `if claude:` branch. The generic core expresses reuse-condition-0 as "if the runtime adapter provides a context-budget probe and it reports `reuse_ok: false`, dispatch fresh"; the Claude adapter fills the probe, the Codex adapter declares it absent (condition-0 vacuously satisfied). This is the same pattern the contract already uses at `first-officer-shared-core.md` line 206 (`send_input on Codex, SendMessage on Claude teams`).

### Decision 1 — Subcommand disposition

Drives the native binary's `spacedock dispatch` surface; mirrors the package seam `native-dispatch-helper` established (generic dispatch/status logic in host-neutral packages; Claude coupling behind a clear seam). The disposition is **native reimplementation behind a Claude-specific package**, NOT in the generic `internal/dispatch` core.

| subcommand | disposition | rationale | oracle |
|---|---|---|---|
| `context-budget` | **native**, in a new Claude-specific package (`internal/claudeteam`), folding the 1M family-rule fix | SPIKE-A proved cheap parity; this is the per-ensign-reuse hot path the entity exists to de-Python; transcript format is stable JSONL. Coupling (reads `~/.claude`) is real but quarantined to the Claude package — it does NOT enter `internal/dispatch`/`internal/status`. | parity test: native vs vendored Python over a fixture `~/.claude` tree (team `config.json` + `agent-*.meta.json` + `agent-*.jsonl`), byte/struct-compare the JSON envelope and exit code |
| `list-standing` | **native**, same Claude package | pure `_mods/*.md` frontmatter enumeration (`standing: true`), no transcript coupling; needed at FO boot before members exist; trivially native | parity test over a `_mods` fixture: native vs Python newline-delimited sorted paths |
| `show-standing` | **native scaffolding, Claude-rendered body** | the composition (`enumerate_declared_standing_teammates` + render) is runtime-neutral but the rendered body is Claude-specific SendMessage routing prose; native keeps the body string verbatim from the Python oracle so parity holds byte-for-byte | parity test over a `_mods` fixture including the `## Routing Usage` body-extraction path |
| `spawn-standing` | **native**, same Claude package | emits an Agent() spec JSON + `member_exists` team-config probe; closes the last steady-state Python shell-out (feedback/standing spawn); model-enum + heading-order validation port directly | parity test: already-alive path (fixture team config has the member) and spec-emit path (member absent), plus the loud-failure cases (missing mod, missing `standing: true`, bad model enum) |
| `build`'s `_mods`/`show-standing` fetch-line branch | **native**, extending `internal/dispatch/build.go` | `native-dispatch-helper` deferred exactly this branch to this entity (see `build.go:60-61`). The branch is `enumerate_declared_standing_teammates(workflow_dir, team_name)` → if non-empty, append a `spacedock dispatch show-standing` fetch line. The enumeration helper is runtime-neutral and can live in `internal/dispatch` (it is pure `_mods` parsing); the Claude-specific `show-standing` *body* it points at lives in the Claude package. | extend the existing three-channel parity harness with a `_mods`-bearing fixture: assert native build emits the standing fetch line iff Python does, byte-identical after the documented `claude-team`→`spacedock dispatch` fetch-prefix rewrite |

**Why native for all five, not keep-Python or shim:** the entity's whole reason to exist is that the steady-state FO loop is not Python-free even though initial dispatch is. Keep-Python on `context-budget` would leave the per-reuse hot path shelling to `python3 + claude-team` forever; that's the status quo this entity removes. A shim buys nothing over native here — the logic is small and the parity test is the same cost either way. The transcript-format coupling cost (the cited downside of native) is contained by the package boundary: it lives in `internal/claudeteam`, never in the generic dispatch/status core, so the host-neutrality invariant (Decision 2) still holds.

**Package placement:** `enumerate_declared_standing_teammates` (pure `_mods` frontmatter parsing, the `build` branch needs it) is runtime-neutral → `internal/dispatch`. Everything that reads `~/.claude` (context-budget, `member_exists`, `lookup_model`) or renders Claude SendMessage prose (`show-standing` body) → `internal/claudeteam`. This keeps `internal/dispatch` and `internal/status` free of `~/.claude` reads.

### Decision 2 — Claude-vs-generic contract split (AC-3)

The generic `first-officer-shared-core.md` carries Claude-Code-specific runtime coupling today. Concrete leaks identified (real-file evidence, vendored copy in this checkout):

| location | leak | relocation |
|---|---|---|
| line 141 (reuse condition 0) | `run \`claude-team context-budget --name {ensign-name}\`. If \`reuse_ok\` is false, skip to fresh dispatch` — names a Claude-only binary as an unconditional algorithm step | generic core states the *capability*: "if the runtime adapter provides a context-budget probe and it reports the worker over budget, dispatch fresh; if the adapter declares no budget probe, this condition is satisfied." Concrete `spacedock dispatch context-budget` invocation lives in `claude-first-officer-runtime.md` (already partly there, line 160). |
| line 205 (feedback rejection flow) | same `claude-team context-budget` named invocation | same relocation; generic core references the capability, Claude adapter owns the command |
| line 116 (prose-polish routing) | `Check team-config membership via \`member_exists\`` + `Workers dispatched via \`claude-team build\`` | generic core describes "check the runtime's team-membership predicate"; `member_exists` (a Claude helper name) and the `claude-team build`/`spacedock dispatch build` command move to the Claude adapter |
| line 324 (`## Standing Teammates`) | `The FO discovers each at boot (via \`list-standing\`)` — names a Claude subcommand | generic core: "discovers each at boot via the runtime adapter's standing-discovery step"; `list-standing` command moves to the Claude adapter (already there, line 34) |
| lines 327-329 (`## Standing Teammates`) | `## Hook: startup` / `_mods/{name}.md` / SendMessage / `TeamDelete` — Claude-only declaration + routing mechanics presented as generic | the *concept* (long-lived specialist, team-scoped lifecycle, best-effort non-blocking routing) is genuinely cross-runtime and STAYS in the core, host-qualified the way line 326 already is; the Claude-specific *mechanics* (`## Hook: startup` parsing, `_mods` layout, SendMessage, `TeamDelete`) move to the Claude adapter's standing section |

NOT leaks (already correctly host-qualified — do not touch): line 206 (`send_input on Codex, SendMessage on Claude teams`), line 326 (`Because Claude teams are per-captain-session ... other runtimes re-derive scope`), line 348 (`on Claude Code: ... file-staleness safety net`). The fix follows the pattern these lines already establish.

**Native package boundary mirrors the prose split:** generic dispatch/status logic stays in `internal/dispatch` + `internal/status` (no `~/.claude` reads); Claude coupling (context-budget transcript reads, `member_exists`, standing-body rendering) lives in `internal/claudeteam`.

**The host-neutrality invariant (AC-3 oracle — a structural check over real artifacts, NOT a substring grep).** Two complementary oracles:

1. **Prose-side (over the vendored contract files):** a test parses `first-officer-shared-core.md` and asserts that every reference to a *named Claude helper command or helper function* (`claude-team`, `spacedock dispatch context-budget`, `context-budget`, `spawn-standing`, `list-standing`, `show-standing`, `member_exists`, `lookup_model`) appears ONLY inside a span that is explicitly host-qualified — i.e., either the same sentence/bullet names a non-Claude runtime alternative (the `X on Codex, Y on Claude` shape at line 206) OR the reference lives under the Claude adapter file, not the generic core. The invariant is "no *unqualified* Claude-runtime command in the generic core," resolved by parsing the markdown structure (which bullet/section a token sits in and whether that span carries a host qualifier), not by `grep -c claude`. A bare mention inside an explicitly Claude-qualified clause is allowed; an unqualified algorithm step that names a Claude-only command fails.
2. **Code-side (over the real Go packages):** a test asserts `internal/dispatch` and `internal/status` import-graph and source contain no `~/.claude`/`os.UserHomeDir`-rooted team/transcript reads — those reside only in `internal/claudeteam`. Mechanizable via `go/parser` over the package files (no transcript-path string literals; no `.claude` path joins) — a structural property of the source, falsifiable by adding such a read and watching it fail.

### Decision 3 — Context-budget 1M fix (folded in)

Replace `NATIVE_1M_MODELS` exact-name set + `[1m]`-substring with the forward family rule in BOTH the (native) implementation and the vendored Python oracle (so parity holds): `[1m]` suffix → 1M; else parse `claude-opus-4-(\d+)`, minor ≥ 7 → 1M; else 200k. SPIKE-B validated the boundary. Behavioral oracle: a fixture `~/.claude` tree stamping the ensign `claude-opus-4-8` (no suffix) at a sub-600k resident count asserts `context_limit == 1_000_000` and `reuse_ok == true` — the exact false-negative the bug produces today.

### Coordination (operating-contract collision avoidance)

Three entities touch the operating-contract reference files. Declared touch-points so the FO can sequence them without collision, vendored-copy-first per `encoding-deliverable-principles.md`:
- **this entity:** `first-officer-shared-core.md` `## Standing Teammates` (full section) + the **reuse-condition-0 paragraph** and the **feedback-rejection step 4** inside `## Completion and Gates` + line 116 in the prose-polish routing note. Plus `claude-first-officer-runtime.md` (gains the relocated commands).
- **deliverable-contract-hardening / `encoding-deliverable-principles.md`:** edits a DIFFERENT paragraph of `## Completion and Gates` (the AC-coverage cross-check) and `## Probe and Ideation Discipline`. Same section as my reuse-condition edit, different paragraphs — sequence, do not parallelize, on `## Completion and Gates`.
- **spacedock-packaging:** adds an FO-Startup step-0; if that lands in `## Startup`/boot prose it does not overlap my `## Standing Teammates`/`## Completion and Gates` edits.

### Independent staff review

This ideation warrants an **independent staff review** before the gate. Per the stage definition's staff-review trigger, the design spans (a) a native/Python parity contract over real `~/.claude` transcripts, (b) a host-neutrality invariant that must be a structural check rather than a grep, and (c) a multi-entity operating-contract edit collision — each individually the kind of "native parity / contract reorganization" complexity the FO is told to route for review. The review should check: the parity-fixture realism (does the fixture `~/.claude` tree reproduce the suffix-drop the live bug needs?), the host-neutrality invariant's resistance to gaming (can it pass while a leak survives?), and the package-boundary code-side oracle's soundness.

## Stage Report: ideation

- DONE: Decide the runtime-coupled-subcommand disposition (context-budget, spawn-standing, list-standing, show-standing, and build's _mods/show-standing fetch-line branch): per subcommand, native vs thin-shim vs keep-Python, justified against cost vs benefit. SPIKE the riskiest unverified mechanism and record the behavioral evidence.
  `## Ideation design` → Decision 1 table: all five native, in a Claude-specific `internal/claudeteam` package (the runtime-neutral `_mods` enumeration stays in `internal/dispatch`). SPIKE-A proved native-vs-Python context-budget parity on a frozen snapshot of this ensign's own real jsonl (resident 85183 == 85183, model `claude-opus-4-8` both sides).
- DONE: Design the Claude-vs-generic contract split (AC-3): identify Claude-Code-specific leaks in the generic core, relocate them into the Claude runtime adapter, and define the host-neutrality invariant as a check over real artifacts. FOLD IN the context-budget 1M-detection fix.
  Decision 2 enumerates five concrete leaks in `first-officer-shared-core.md` (lines 141, 205, 116, 324, 327-329) with relocations and three not-leaks (already host-qualified); the invariant is a two-part structural oracle (markdown-structure parse + `go/parser` over the generic packages), not a grep. 1M fix folded in as Decision 3 + AC-4, SPIKE-B-validated.
- DONE: Rewrite the seed ACs as behavioral end-state properties, each naming an exercise-and-observe oracle — NO doc-only ACs. State whether ideation warrants an independent staff review.
  AC-1..AC-4 rewritten as end-state properties, each naming a concrete oracle (extended three-channel parity harness over `~/.claude`+`_mods` fixtures; budget-unavailable fallback fixture; two-part host-neutrality invariant; 1M family-rule fixture+boundary table). `### Independent staff review` recommends YES with three review focuses.

### Summary

Spiked all three risks before locking any decision. SPIKE-A: native Go reproduces Python's context-budget resident-extraction exactly (85183==85183 on a frozen snapshot of the live jsonl). SPIKE-B: the forward family rule fixes the live `claude-opus-4-8` false-negative and stays correct across the version boundary (4-6 stays 200k, hypothetical 4-10 → 1M). SPIKE-C (the decisive one): reading the Codex FO adapter showed context-budget and standing-teammates are *structurally* Claude-only — Codex has neither — so the host-neutral seam is a capability boundary, not a per-host `if`. Disposition: all five subcommands go native behind `internal/claudeteam`, removing the steady-state Python shell-out the entity exists to kill, while the generic dispatch/status core stays free of `~/.claude` reads. Recommending an independent staff review given the parity-contract + host-neutrality-invariant + multi-entity-contract-collision span. No worktree (state-checkout ideation); no test gates run this stage (no code shipped — implementation is the next stage).
