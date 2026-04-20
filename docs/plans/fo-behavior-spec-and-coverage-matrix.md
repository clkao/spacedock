---
id: 202
title: "FO behavior spec + coverage matrix as source of truth"
status: ideation
source: "session 2026-04-18 — reflecting on the flake-task backlog (13 open tasks, mostly class-A/B LLM-output-brittleness). Captain observation: without a structured inventory of desired FO behaviors mapped to tests and model coverage, every flake triage is ad-hoc and every fix risks deflecting ambient flakes to new tasks without addressing the root class. Spec-as-source-of-truth: prose (FO SKILL + runtime adapters) and code (tests) should conform to the spec, not vice versa."
started: 2026-04-19T16:58:51Z
completed:
verdict:
score: 0.85
worktree: .worktrees/spacedock-ensign-fo-behavior-spec-and-coverage-matrix
issue:
pr:
mod-block:
---

## Why this matters

The flake-tracking backlog grew to 13 tasks this session, most in class A (haiku protocol adherence), class B (FO bootstrap discipline), and class C (LLM-output brittleness in behavioral tests). Workers repeatedly deflected adjacent flakes into new tasks rather than addressing root causes. The FO (me) approved each deflection because I had no way to evaluate "is this flake in a critical behavior or a redundant assertion?"

The root gap: **we have no structured inventory of desired FO behaviors.** FO contract prose is scattered across `SKILL.md`, `shared-core.md`, `claude-first-officer-runtime.md`, `codex-first-officer-runtime.md`. Tests are scattered across `tests/test_*.py`. Nothing ties them together. Every flake question begins with "go grep the prose."

**Without an inventory we can't answer:**

- Which desired FO behaviors have zero test coverage?
- Which behaviors are tested redundantly (and therefore safe to xfail in one place)?
- What is the flake rate of behavior X on model Y, mode Z?
- Which prose imperatives are load-bearing vs decorative?
- When we add a behavior to prose, does an assertion exist for it?

The captain's framing: **the spec is the source of truth. Prose and code are derivatives that must conform.** If prose drifts from the spec, the spec wins (or the spec is updated with a documented change). If code drifts, same. This inverts the current situation where prose+code drift independently and we debug by grep.

## Proposed approach

Adopt **RFC 2119 (MUST/SHOULD/MAY) requirements + requirements traceability matrix (RTM)** as the format. Each requirement gets:

- A stable ID (`FO-R-NNN`)
- Level (MUST / SHOULD / MAY / MUST NOT)
- Area tag (bootstrap, dispatch, gate-review, merge, team-mgmt, standing-teammate, error-recovery, …)
- Prose anchor (file:line range of the current contract prose)
- Test coverage (list of tests + assertion kind: static / live / behavioral)
- Model×mode coverage (which CI jobs exercise this)
- Flake status per cell (green / flaky-tracked / xfailed / uncovered)
- Status (covered / partial / uncovered / deprecated)

Rendered shape: **one markdown file per requirement** with YAML frontmatter (queryable) + prose body (readable). OR one aggregate markdown with a big table + anchors. Captain to pick during ideation; both shapes have worked in other projects.

Source-of-truth semantics:

1. FO prose (`SKILL.md`, `references/*.md`) links TO the spec by requirement ID.
2. Tests have a docstring or marker naming the requirement ID they verify.
3. New behaviors are added to the spec FIRST, then prose + tests are added to conform.
4. Flake tasks are resolved by updating the spec's status (fix / retire / mark model-specific xfail) — not by filing new tasks and hoping someone notices.

## Initial deliverables

1. **Format choice.** Picking between (a) one-file-per-requirement YAML+MD, (b) single aggregate markdown, or (c) hybrid (aggregate for browsing, individual files for deep linking).
2. **Enumerate requirements from current prose.** Grep `MUST|MUST NOT|SHOULD|NEVER|always|never` across `skills/first-officer/**/*.md` and the workflow README's FO-relevant bits. Assign IDs. First pass: ~30-50 requirements likely.
3. **Map tests to requirements.** For each test in `tests/test_*.py`, identify which requirement(s) it asserts. Flag tests that verify nothing specific (surface-level smoke).
4. **Stamp current coverage + flake rates.** Per requirement × model × mode cell, stamp from recent CI (last 2 weeks would capture post-#186 unpin + all this session's PRs).
5. **Identify gaps.** Uncovered requirements (no test) + redundantly-covered requirements + load-bearing flaky requirements.
6. **Refactor prose to reference spec IDs.** `SKILL.md` and runtime adapters point at `FO-R-NNN` instead of restating; spec body carries the authoritative prose.
7. **Refactor tests to declare which requirements they assert.** Docstring convention or pytest marker (`@pytest.mark.requirement("FO-R-013")`).
8. **Queryability tooling.** Shell/Python helpers: `fo-spec list --uncovered`, `fo-spec list --flaky --model opus-4-7`, `fo-spec coverage --test test_gate_guardrail`.

## Acceptance criteria

**AC-1 — Spec format chosen and documented.**
Verified by: one top-level file (`skills/first-officer/references/fo-behavior-spec/README.md` or similar) documents the chosen format — RFC 2119 imperatives + RTM, file layout (individual vs aggregate), ID scheme, area tags, status enum.

**AC-2 — Initial spec enumerates ≥30 FO requirements.**
Verified by: the spec directory (or aggregate file) contains ≥30 entries with stable IDs, each with at minimum: level, area, prose anchor, one-line description. Count may rise through iteration; 30 is the minimum floor.

**AC-3 — Every test in `tests/test_*.py` maps to at least one requirement (or is flagged unmapped).**
Verified by: a tooling run (or static audit) produces a mapping table; flagged unmapped tests have a documented rationale or are deleted.

**AC-4 — Coverage matrix stamps flake status for every covered requirement.**
Verified by: for each covered requirement, the spec records its status on at minimum claude-live, claude-live-bare, claude-live-opus, codex-live (green / flaky-tracked / xfailed / untested). "Flaky-tracked" cells cite the tracking task ID.

**AC-5 — Uncovered requirements are explicitly listed.**
Verified by: the spec has an "uncovered requirements" view or query that returns the set of `MUST / SHOULD` imperatives without test coverage.

**AC-6 — FO prose (SKILL.md + runtime adapters) references spec IDs.**
Verified by: `grep -c 'FO-R-' skills/first-officer/references/*.md skills/first-officer/SKILL.md` returns a substantial count (≥10 references as a floor for "the conforming refactor started").

**AC-7 — At least one tooling helper lands.**
Verified by: `fo-spec list --uncovered` or equivalent shell/Python invocation returns a parseable list. Single helper is sufficient for MVP; more can follow.

**AC-8 — Static suite green post-merge.**
Verified by: `make test-static` passes on main.

## Test plan

- **Static, primary:** grep-style verifications of AC-1 through AC-7. Each assertion is O(cheap) — file existence, frontmatter parsing, cross-reference count.
- **No live tests required.** The inventory is a structural artifact, not runtime behavior.
- **Cost:** low (~$3-5 for ideation; ~$10-15 for implementation depending on how much prose refactoring AC-6 entails).

## Out of scope

- **Fixing the open flake tasks.** The inventory INFORMS how to fix them; it does not itself fix them. After #202 lands, subsequent tasks (including potentially a consolidated "FO haiku protocol hardening program") pick from the coverage matrix.
- **Formal specification languages** (TLA+, Alloy, Z). Overkill for prose-driven FO; mentioned in format alternatives and rejected.
- **Gherkin / BDD harness.** Partial fit; may be adopted later for specific scenario-heavy areas but not required for v1 spec.
- **Cross-workflow spec generalization.** #202 scope is the Spacedock FO contract only. If the spec format proves useful, other workflows may adopt it separately.

## Cross-references

- **All open flake tasks** (#141, #155, #160, #161, #171, #194, #195, #196, #197, #198, #200, plus pending test_merge_hook_guardrail filing): these should be re-evaluated against the coverage matrix once #202 lands. Some may consolidate into a single root-cause task; some may be recategorized as uncovered-requirements; some may close as redundant.
- **#199** — FO agent-shutdown discipline + team-health command. Adjacent: both #199 and #202 are about making the FO's operational state observable and structured. #199 covers the runtime (team members vs entity state); #202 covers the behavioral contract (requirements vs tests vs coverage). Compatible; different layers.
- **#112** — multi-player claim semantics. Also adjacent via the agent-stamp concept. Not a dependency.

---

# Ideation revision — disentangled spec (captain-directed)

> The sections above are the v0 seed. Captain critique: v0 bundles four layers into one artifact (stable requirements + test map + prose map + per-run flake status), producing an artifact that is perpetually one commit stale on every axis. This revision replaces the **Proposed approach**, **Initial deliverables**, and **Acceptance criteria** sections above with a star-schema design. The rest of the seed body (Why this matters, Out of scope, Cross-references) stands.

## Layering (star schema)

Four layers, each owned by exactly one source. The spec owns L1 only.

| Layer | Content | Source of truth | Update cadence |
|---|---|---|---|
| **L1** | Stable requirements: ID, level (RFC 2119), area tag, one-line description, rationale | `skills/first-officer/references/fo-behavior-spec/requirements.md` | Slow — changes when a behavior is added / deprecated / re-leveled |
| **L2** | Test→requirement mapping | Declared **in test files** via `@pytest.mark.requirement("FO-R-NNN")` | Changes with tests; discovered by tooling, never stamped |
| **L3** | Prose→requirement mapping | Declared **in prose files** via `<!-- FO-R-NNN -->` HTML-comment anchors | Changes with prose; discovered by tooling, never stamped |
| **L4** | Per-run coverage + flake rate | Derived at query time from (a) current test source, (b) `gh run list` + `gh run view` CI history | Volatile; never committed |

L1 is the dimension table. L2/L3/L4 are fact sources; views join them.

### The one coupling retained: `known_flaky` on L1

Some behaviors are inherently flaky on a specific model family (e.g. Haiku's tendency to drop formatting imperatives). That is a **slow-changing semantic fact about the requirement**, not a per-run rate. Keeping it in L1 as an optional list of `{model, mode, reason, tracking_task}` means the spec can answer "should I expect FO-R-013 to be reliable on Haiku today" without going to CI history. It is distinct from L4's per-run rate: L4 says "last 50 runs had 3% failure on opus-4-7"; `known_flaky` says "Haiku is expected to trip this; tracked in #197". This field updates only when the *situation* changes, not every CI run.

All other v0 couplings (prose anchors as file:line ranges in the spec, per-cell flake stamps, explicit uncovered-requirements list) are removed.

## Format decision

### L1: hybrid aggregate + queryable frontmatter

`skills/first-officer/references/fo-behavior-spec/requirements.md` — single aggregate markdown file. Each requirement is a second-level section (`## FO-R-NNN`) with a fenced `requirement` YAML block containing structured fields, followed by a markdown prose body for rationale.

Rejected alternatives:
- **One file per requirement.** Strong for deep-linking but imposes high directory churn on a repo this size (~30-50 files landing in one PR, future churn on every re-level). Review cost compounds.
- **Pure aggregate table.** Simple but loses rationale; frontmatter-queryability requires a per-row YAML island anyway.

### L2: pytest marker only

`@pytest.mark.requirement("FO-R-013")` — the decorator is structured (toolable via `pytest --collect-only -q --strict-markers`), grep-able (`rg 'requirement\\("FO-R-'`), survives test renames, and stacks cleanly when one test covers multiple requirements. Docstring headers are rejected: parsing is fragile and they don't survive refactor tools.

Register the marker in `pyproject.toml` / `pytest.ini` with `--strict-markers` so typos fail loudly at collection.

### L3: HTML-comment anchors

`<!-- FO-R-013 -->` immediately above the governing prose block. Invisible in rendered markdown, grep-able, stable under prose edits, composable (multiple anchors can precede a block that asserts several requirements). Inline `[FO-R-013]` citations are rejected because they leak into reader-facing text. A separate xref file is rejected because it creates a third source of truth to keep synchronized.

### L4: tooling-only, never committed

A `fo-spec` helper (single script, `skills/first-officer/tools/fo_spec.py` or similar) implements query subcommands. Output is structured (JSON with human-table renderer); no derived state is written back into the repo.

## Blast radius for an ID rename

If `FO-R-013` is renamed to `FO-R-014`:
- L1: one edit in `requirements.md`
- L2: `rg -l 'FO-R-013' tests/` then sed across matches
- L3: `rg -l 'FO-R-013' skills/` then sed across matches
- L4: re-runs free

The tooling SHOULD ship a `fo-spec rename FO-R-013 FO-R-014` subcommand to automate all three edits atomically. Treat ID renames as the hardest maintenance operation; the tooling earning its keep depends on handling this well.

## Requirement record schema (L1)

```yaml
# inside a fenced ```requirement block under `## FO-R-NNN`
id: FO-R-013
level: MUST  # MUST | MUST_NOT | SHOULD | SHOULD_NOT | MAY
area: dispatch  # bootstrap | dispatch | gate-review | merge | team-mgmt | standing-teammate | error-recovery | ...
summary: "FO emits completion-signal literal when forwarding a dispatch prompt."
known_flaky:  # optional; omit entirely when empty
  - model: haiku-4-5
    mode: claude-live
    reason: "Haiku paraphrases parenthesis-equals syntax to English."
    tracking_task: 197
```

The prose body under the YAML block carries rationale, references to prior incidents, and any reader-facing explanation. No file:line prose anchors — tooling discovers those from L3.

Deliberately absent from L1: test list, prose-location list, coverage cells, flake rates. All derived.

## Tooling (L4 queries) — load-bearing

Now that coverage is never stamped, **the tooling is the only way to see it**. AC-7's "at least one tooling helper" is elevated from MVP nice-to-have to load-bearing.

Minimum viable subcommands for this task to count as done:

- `fo-spec list` — print all requirements with level/area/summary.
- `fo-spec list --uncovered` — requirements with zero pytest markers referencing them.
- `fo-spec list --unanchored` — requirements with zero prose anchors referencing them.
- `fo-spec tests FO-R-013` — tests that mark this requirement.
- `fo-spec prose FO-R-013` — prose locations that anchor this requirement.
- `fo-spec orphans` — pytest markers and prose anchors whose IDs are not in `requirements.md`.
- `fo-spec validate` — static check: every ID in tests/prose exists in spec; every spec ID is well-formed. Exits non-zero on drift. Wired into `make test-static`.

Out of scope for v1, left for a follow-up:
- `fo-spec coverage --model opus-4-7` (requires `gh` API calls; punt to a separate task to keep v1 offline-only).
- Flake-rate queries over CI history.

Rationale for offline-only v1: the `gh` API layer adds auth, rate-limit, and network-flake surface that would swamp the structural work. Ship L1/L2/L3 plumbing first; CI-history views layer on cleanly once the schema is stable.

## Revised acceptance criteria

AC-1 through AC-8 below supersede the v0 set. AC-4 (per-cell flake stamps) is removed. AC-5 (uncovered list) becomes a tooling query. AC-7 (tooling) is load-bearing.

**AC-1 — Spec layout and schema documented.**
`skills/first-officer/references/fo-behavior-spec/README.md` documents: the four-layer model, L1 file location, requirement record schema, L2 marker convention, L3 anchor convention, and `fo-spec` subcommand list.
*Test:* static — file exists, contains sections named `Layering`, `Requirement schema`, `Marker convention`, `Anchor convention`, `Tooling`.

**AC-2 — `requirements.md` enumerates ≥30 FO requirements.**
Each entry has an `## FO-R-NNN` heading, a fenced `requirement` YAML block with `id`, `level`, `area`, `summary`, and a prose body.
*Test:* static — `fo-spec validate` parses the file cleanly; count of `## FO-R-` headings ≥30; every YAML block is valid and its `id` matches the heading.

**AC-3 — `fo-spec` tool ships with the seven subcommands listed above.**
*Test:* unit tests (`tests/test_fo_spec.py`) — one test per subcommand, using a small fixture `requirements.md` + fixture test dir + fixture prose dir. `validate` has a positive and a negative case (drift detected → non-zero exit).

**AC-4 — Pytest marker registered and at least one real test adopts it.**
`pyproject.toml` / `pytest.ini` registers `requirement` as a known marker under `--strict-markers`. At least one existing test in `tests/test_*.py` carries `@pytest.mark.requirement("FO-R-NNN")` mapping to a real spec ID.
*Test:* static — `pytest --collect-only -q` does not warn about unknown `requirement` marker; `fo-spec tests FO-R-NNN` returns the adopted test.

**AC-5 — At least one prose anchor lands in FO references.**
At least one `<!-- FO-R-NNN -->` anchor in `skills/first-officer/**/*.md` mapping to a real spec ID.
*Test:* static — `fo-spec prose FO-R-NNN` returns the anchored location.

**AC-6 — `fo-spec validate` is wired into `make test-static`.**
Drift (unknown ID in tests or prose, malformed YAML, duplicate IDs) fails the static suite.
*Test:* integration — `make test-static` runs `fo-spec validate`; a deliberate drift in a scratch branch causes non-zero exit.

**AC-7 — `fo-spec list --uncovered` and `--unanchored` run against the real repo and produce a parseable list.**
The lists are the v1 answer to "what's the gap"; they replace the v0 static "uncovered requirements" section.
*Test:* integration — run against live repo; output parses as the documented format (JSON or table).

**AC-8 — Static suite green post-merge.**
*Test:* `make test-static` passes on main.

**Removed from v0:** AC-4 (per-cell flake stamps), AC-5 (static uncovered-requirements list), AC-6's "≥10 references floor" (replaced by AC-5's "at least one anchor" proof-of-life; large-scale prose refactor is a separate follow-up task, not an MVP gate).

## Revised test plan

- **Static verification of spec format:** `fo-spec validate` parsing the committed `requirements.md`.
- **Tooling unit tests (`tests/test_fo_spec.py`):** fixture-based per-subcommand coverage; parse a known-good and known-bad fixture tree.
- **One integration test:** `fo-spec validate` exits non-zero on drift (add a bogus `FO-R-999` marker in a temp test file, confirm validate fails).
- **Wire-in check:** `make test-static` invokes `fo-spec validate` and fails the suite on drift.
- **No live LLM tests.** The spec and tooling are structural.

Estimated cost: low. Implementation stage ~$10-20 including the 30+ requirement enumeration, tooling, marker adoption on one test, and anchor adoption on one prose location. Large-scale prose/test refactor is explicitly deferred.

## What's deferred to follow-up tasks

- Marking every existing test with its requirement ID (bulk L2 adoption).
- Anchoring every load-bearing prose block (bulk L3 adoption).
- CI-history coverage views (`fo-spec coverage --model X`).
- Flake-rate queries over `gh run` history.
- Re-evaluating the 13 open flake tasks against the coverage matrix.

These are all unblocked by #202 shipping but are not part of its acceptance. Filing them as separate tasks keeps #202's blast radius contained.

## L1 draft — requirement enumeration

This is a first-pass enumeration extracted from the current FO contract (`SKILL.md`, `first-officer-shared-core.md`, `claude-first-officer-runtime.md`, `codex-first-officer-runtime.md`, `code-project-guardrails.md`). **It is a draft for captain review, not a finished spec.** The goal is to validate the L1 schema against real content: does the area taxonomy hold? Does the level enum cover what the prose expresses? Do any requirements resist the format?

### Area taxonomy (draft)

- `startup` — workflow discovery, boot-time reads, status `--boot` contract
- `team-lifecycle` — TeamCreate, TeamDelete, Degraded Mode, recovery ladder (Claude-specific)
- `dispatch` — worker assembly, `claude-team build`, break-glass rules, Codex `spawn_agent`
- `worker-reuse` — reuse conditions, model-match, context budget, SendMessage advancement
- `gate-review` — gate presentation, AC cross-check, no self-approve
- `feedback-flow` — rejection routing, feedback cycles, 3-cycle escalation
- `merge-cleanup` — merge hooks, mod-block, terminal transitions, archive/worktree cleanup
- `state-writes` — FO Write Scope, frontmatter ownership, main vs worktree
- `standing-teammate` — discovery, lazy-spawn, routing contract, team-scope lifecycle
- `captain-interaction` — gate approval, clarification, idle-hallucination guardrail
- `probe-discipline` — Grep-over-Read, no full-file Read as probe
- `single-entity-mode` — bounded runs, auto-resolve gates, stop conditions

### Level enum (draft)

RFC 2119: `MUST`, `MUST_NOT`, `SHOULD`, `SHOULD_NOT`, `MAY`. No extensions — keep the vocabulary small.

### Requirements

**Format note:** each entry below is the **human-readable rendering** of the planned `requirement` YAML block. In the final spec, the YAML carries the structured fields and the prose below it carries rationale. Here, for review density, the two are collapsed into a single bullet per requirement. Anchor lines cite the current governing prose — they are **discovery hints**, not the L3 anchor itself (L3 is declared in the prose file via HTML comment).

#### `startup`

- **FO-R-001** (MUST) — FO discovers the workflow directory via explicit operator path, else `status --discover` with ambiguity-stops-on-multiple.
  *Prose:* `first-officer-shared-core.md` §Startup step 2.
- **FO-R-002** (MUST) — FO runs `status --boot` at startup to obtain MODS, NEXT_ID, ORPHANS, PR_STATE, DISPATCHABLE in one call.
  *Prose:* `first-officer-shared-core.md` §Startup step 4.
- **FO-R-003** (MUST) — FO runs startup mod hooks before any normal dispatch.
  *Prose:* `first-officer-shared-core.md` §Startup step 4 MODS bullet.
- **FO-R-004** (MUST) — Orphan worktree detection reports anomalies without auto-redispatch.
  *Prose:* `first-officer-shared-core.md` §Startup ORPHANS bullet.

#### `team-lifecycle` (Claude-only)

- **FO-R-005** (MUST) — `TeamCreate` is the first team-mode tool call in every Claude session; no `spawn-standing`, `Agent`, or `SendMessage` precedes it.
  *Prose:* `claude-first-officer-runtime.md` §Team Creation step 1.
- **FO-R-006** (MUST) — Team name format: `{project}-{dir_basename}-{YYYYMMDD-HHMM}-{shortuuid}`, lowercase, hyphen-separated.
  *Prose:* `claude-first-officer-runtime.md` §Team Creation step 1.
- **FO-R-007** (MUST) — FO stores the `team_name` returned by `TeamCreate` (which may differ from requested) and uses it for all subsequent calls.
  *Prose:* `claude-first-officer-runtime.md` §Team Creation step 1 note.
- **FO-R-008** (MUST_NOT) — FO MUST NOT delete existing team directories (`rm -rf ~/.claude/teams/...`).
  *Prose:* `claude-first-officer-runtime.md` §Team Creation step 1 note.
- **FO-R-009** (MUST_NOT) — FO MUST NOT combine `TeamDelete`, `TeamCreate`, or `Agent` dispatch in the same message — they race under parallel tool execution.
  *Prose:* `claude-first-officer-runtime.md` §Team Creation recovery procedure.
- **FO-R-010** (MUST) — On "Team does not exist" mid-session: one fresh-suffixed `TeamCreate` attempt (tier 1), else Degraded Mode (tier 2), else surface to captain (tier 3). No retries within a tier.
  *Prose:* `claude-first-officer-runtime.md` §Team Creation failure recovery ladder.
- **FO-R-011** (MUST) — FO blocks all `Agent` dispatch while team setup is unresolved.
  *Prose:* `claude-first-officer-runtime.md` §Team Creation, "Block all Agent dispatch".
- **FO-R-012** (MUST) — On Degraded Mode entry, FO emits the captain-report paragraph verbatim to direct text output (not SendMessage).
  *Prose:* `claude-first-officer-runtime.md` §Degraded Mode → Captain Report Template.
- **FO-R-013** (MUST) — Once Degraded Mode is active, no `team_name` parameter appears on any subsequent `Agent` dispatch for the remainder of the session.
  *Prose:* `claude-first-officer-runtime.md` §Degraded Mode → Effects.
- **FO-R-014** (MUST_NOT) — SendMessage to pre-degrade agent names is forbidden once Degraded Mode is active.
  *Prose:* `claude-first-officer-runtime.md` §Degraded Mode → Effects.
- **FO-R-015** (MUST) — On Degraded Mode entry, FO performs a single-pass cooperative shutdown sweep (one SendMessage `shutdown_request` per known agent name, best-effort, non-blocking), exempting agents in active feedback cycles.
  *Prose:* `claude-first-officer-runtime.md` §Degraded Mode → Cooperative Shutdown Sweep.

#### `dispatch`

- **FO-R-016** (MUST) — FO assembles `Agent()` prompts via `claude-team build`, forwarding `subagent_type`, `name`, `team_name`, `model`, `prompt` verbatim on zero-exit.
  *Prose:* `claude-first-officer-runtime.md` §Dispatch Adapter → MANDATORY.
- **FO-R-017** (MUST_NOT) — FO MUST NOT manually assemble `Agent()` prompts or invent `name` values when `claude-team build` is available.
  *Prose:* `claude-first-officer-runtime.md` §Dispatch Adapter → MANDATORY.
- **FO-R-018** (MUST_NOT) — FO MUST NOT use `subagent_type="first-officer"` for worker dispatch — that clones the FO.
  *Prose:* `claude-first-officer-runtime.md` §Dispatch Adapter.
- **FO-R-019** (MUST) — Break-glass manual dispatch template is used ONLY on `claude-team build` non-zero exit or unavailability. Zero-exit is never a break-glass trigger.
  *Prose:* `claude-first-officer-runtime.md` §Dispatch Adapter step 4, Break-Glass block.
- **FO-R-020** (MUST_NOT) — FO MUST NOT probe `~/.claude/teams/{team_name}/` filesystem state before `Agent()` in the normal dispatch path (guaranteed false positive under registry-desync).
  *Prose:* `claude-first-officer-runtime.md` §Dispatch Adapter → No pre-dispatch filesystem probe.
- **FO-R-021** (MUST) — Dispatch checklist is per-dispatch, stage-level, at most 3 items, naming what separates a good outcome from a ceremonial one. It is NOT the AC list and NOT a work-breakdown.
  *Prose:* `first-officer-shared-core.md` §Dispatch step 2.
- **FO-R-022** (MUST) — Dispatch commits the state transition on main with message `dispatch: {slug} entering {next_stage}` before spawning the worker.
  *Prose:* `first-officer-shared-core.md` §Dispatch step 6.
- **FO-R-023** (MUST) — On Codex, worker spawn uses `spawn_agent(..., fork_context=false)`; `fork_context=false` is never omitted.
  *Prose:* `codex-first-officer-runtime.md` §Dispatch Adapter.
- **FO-R-024** (MUST) — Codex worker prompts are fully self-contained (no inherited thread context).
  *Prose:* `codex-first-officer-runtime.md` §Dispatch Adapter.

#### `worker-reuse`

- **FO-R-025** (MUST) — Before reuse, FO runs `claude-team context-budget --name {ensign-name}`; `reuse_ok == false` forces fresh dispatch.
  *Prose:* `first-officer-shared-core.md` §Reuse conditions 0; `claude-first-officer-runtime.md` §Context Budget.
- **FO-R-026** (MUST) — Reuse requires ALL conditions: not bare mode, next stage not `fresh: true`, same worktree mode, `lookup_model(worker) == next_stage.effective_model`.
  *Prose:* `first-officer-shared-core.md` §Reuse conditions 1-4.
- **FO-R-027** (MUST) — When model-mismatch forces fresh dispatch, FO emits the diagnostic anchor phrase `does not match next stage effective_model` verbatim.
  *Prose:* `first-officer-shared-core.md` §Completion and Gates, model-mismatch paragraph.
- **FO-R-028** (MUST) — Reuse advancement uses `SendMessage(to="{ensign_name}")` on Claude, `send_input(handle)` on Codex — NEVER `claude-team build` (helper serves only initial dispatch).
  *Prose:* `claude-first-officer-runtime.md` §Dispatch Adapter → Reuse dispatch; `codex-first-officer-runtime.md` §Dispatch Adapter reuse flow.
- **FO-R-029** (MUST_NOT) — FO MUST NOT send `SendMessage(shutdown_request)` to dead/unresponsive ensigns; dead ensigns are tracked in session memory.
  *Prose:* `claude-first-officer-runtime.md` §Dead ensign handling.
- **FO-R-030** (MUST) — On Codex reuse, `send_input` is NOT completion evidence; FO calls `wait_agent` on the same handle before advancing when the result is on the entity's critical path.
  *Prose:* `codex-first-officer-runtime.md` §Dispatch Adapter reuse flow.

#### `gate-review`

- **FO-R-031** (MUST_NOT) — FO MUST NOT self-approve gates, infer approval from silence, or accept agent messages as gate approval.
  *Prose:* `first-officer-shared-core.md` §Completion and Gates "never self-approve"; `claude-first-officer-runtime.md` §Captain Interaction.
- **FO-R-032** (MUST) — FO keeps the dispatched agent alive while waiting at a gate.
  *Prose:* `first-officer-shared-core.md` §Completion and Gates.
- **FO-R-033** (MUST) — At every gate, FO cross-checks the entity's `## Acceptance criteria` section — every `**AC-N**` must have at least one evidence citation from this or a prior stage report.
  *Prose:* `first-officer-shared-core.md` §Completion and Gates, AC coverage cross-check paragraph.
- **FO-R-034** (MUST) — Gate presentation uses the exact template: `Gate review: {title} — {stage}` + verbatim Stage Report + `Assessment: {N} done, {N} skipped, {N} failed. [Recommend approve / Recommend reject: {reason}]`.
  *Prose:* `claude-first-officer-runtime.md` §Gate Presentation.
- **FO-R-035** (MUST) — Checklist review emits the explicit count summary `{N} done, {N} skipped, {N} failed`.
  *Prose:* `first-officer-shared-core.md` §Completion and Gates.

#### `feedback-flow`

- **FO-R-036** (MUST) — Feedback-gate REJECTED recommendations auto-bounce into the feedback rejection flow instead of waiting for manual review.
  *Prose:* `first-officer-shared-core.md` §Completion and Gates, gated-stage bullets.
- **FO-R-037** (MUST) — Feedback cycles are tracked in a `### Feedback Cycles` section of the entity body; 3 cycles escalates to the human.
  *Prose:* `first-officer-shared-core.md` §Feedback Rejection Flow step 2-3.
- **FO-R-038** (MUST) — Routed rejection messages carry concrete next-stage assignment and fix work — NOT an acknowledgment-only ping.
  *Prose:* `first-officer-shared-core.md` §Feedback Rejection Flow step 5; `codex-first-officer-runtime.md` §Dispatch Adapter reuse flow.

#### `merge-cleanup`

- **FO-R-039** (MUST) — When merge hooks exist, FO sets `mod-block=merge:{mod_name}` BEFORE invoking the first hook, in its own `--set` call.
  *Prose:* `first-officer-shared-core.md` §Merge and Cleanup step 1.
- **FO-R-040** (MUST) — Clearing `mod-block` runs in a standalone `--set` call separate from terminal-field updates; `status --set` refuses the combined form unless `--force` is passed.
  *Prose:* `first-officer-shared-core.md` §Merge and Cleanup step 5.
- **FO-R-041** (MUST) — `status --set` and `status --archive` mechanism-level enforcement refuses terminal transitions when merge hooks are registered AND `pr` is empty AND `mod-block` is empty (unless `--force`). The mechanism catches FO amnesia.
  *Prose:* `first-officer-shared-core.md` §Mod Hook Convention → Mod-Block Enforcement; `claude-first-officer-runtime.md` §Mod-Block Enforcement.
- **FO-R-042** (MUST) — Worktree removal uses `git worktree remove`, not `rm -rf` (filesystem deletion leaves stale tracking entries).
  *Prose:* `code-project-guardrails.md` §Git and Worktrees.
- **FO-R-043** (MUST_NOT) — FO MUST NOT delete the remote branch while a PR is still pending — remote-branch cleanup belongs to PR merge.
  *Prose:* `first-officer-shared-core.md` §Merge and Cleanup step 9.

#### `state-writes`

- **FO-R-044** (MUST) — FO's writable scope on main is exactly: entity frontmatter (via `status --set`), new entity files, `### Feedback Cycles` section, archive moves, state-transition commits. Nothing else.
  *Prose:* `first-officer-shared-core.md` §FO Write Scope.
- **FO-R-045** (MUST_NOT) — FO MUST NOT directly edit code files, test files, mod files, scaffolding, or entity body beyond `### Feedback Cycles` on main.
  *Prose:* `first-officer-shared-core.md` §FO Write Scope off-limits list.
- **FO-R-046** (MUST) — For worktree-backed entities, active stage/status/report/body state lives in the worktree copy; `pr:` is the mirrored exception on main.
  *Prose:* `first-officer-shared-core.md` §Worktree Ownership.

#### `standing-teammate`

- **FO-R-047** (MUST) — Standing-teammate discovery runs via `claude-team list-standing --workflow-dir {wd}` after TeamCreate resolves and BEFORE entering the normal dispatch event loop.
  *Prose:* `claude-first-officer-runtime.md` §Standing teammate discovery pass.
- **FO-R-048** (MUST) — Standing-teammate spawn is deferred to first team-mode dispatch (lazy-spawn), not at boot.
  *Prose:* `claude-first-officer-runtime.md` §Standing teammate lazy-spawn; `first-officer-shared-core.md` §Standing Teammates first-boot-wins.
- **FO-R-049** (MUST) — `claude-team spawn-standing` emits an Agent() call spec; FO forwards `subagent_type`, `name`, `team_name`, `model`, `prompt` verbatim to Agent.
  *Prose:* `claude-first-officer-runtime.md` §Standing teammate lazy-spawn step 1c.
- **FO-R-050** (MUST) — Standing-teammate routing uses SendMessage by the declared `name`, best-effort, non-blocking, 2-minute timeout; senders never wait synchronously.
  *Prose:* `first-officer-shared-core.md` §Standing Teammates routing contract.
- **FO-R-051** (MUST) — Discovery runs in single-entity/bare/Degraded Mode (it is cheap), but lazy-spawn is skipped in those modes.
  *Prose:* `claude-first-officer-runtime.md` §Standing teammate discovery pass, bare/Degraded note.

#### `captain-interaction`

- **FO-R-052** (MUST) — Captain communication uses direct text output, not SendMessage. SendMessage is for agent-to-agent only.
  *Prose:* `claude-first-officer-runtime.md` §Captain Interaction.
- **FO-R-053** (MUST) — After acknowledging idle notifications once, FO produces ZERO output for subsequent idle notifications until a real human message arrives (idle-hallucination guardrail).
  *Prose:* `claude-first-officer-runtime.md` §Agent Back-off, IDLE HALLUCINATION GUARDRAIL.
- **FO-R-054** (MUST_NOT) — FO MUST NOT interpret idle notifications as "stuck" or "unresponsive"; idle is normal between-turn state.
  *Prose:* `claude-first-officer-runtime.md` §Agent Back-off, DISPATCH IDLE GUARDRAIL.
- **FO-R-055** (MUST) — FO reports workflow state once when reaching idle or a gate; no status-spam while waiting.
  *Prose:* `first-officer-shared-core.md` §Clarification and Communication.
- **FO-R-056** (MUST_NOT) — FO MUST NOT file GitHub issues without explicit human approval.
  *Prose:* `first-officer-shared-core.md` §Issue Filing.

#### `probe-discipline`

- **FO-R-057** (SHOULD) — FO prefers Grep over Read for targeted entity-body inspection; full-file Read should not be used as a probe.
  *Prose:* `first-officer-shared-core.md` §Probe and Ideation Discipline.
- **FO-R-058** (MUST) — On Claude Code, FO trusts `status --set` stdout (`field: old -> new`) for mutation narration instead of re-Reading the file (avoids staleness-echo cache-write penalty).
  *Prose:* `first-officer-shared-core.md` §Probe and Ideation Discipline; `claude-first-officer-runtime.md` §Entity-Body Inspection.
- **FO-R-059** (MUST) — When checking tool-X supports-Y, FO reads X's schema via ToolSearch before greping for callers — usage presence is not existence evidence.
  *Prose:* `first-officer-shared-core.md` §Probe and Ideation Discipline first bullet.

#### `single-entity-mode`

- **FO-R-060** (MUST) — Single-entity mode activates on non-interactive sessions (`claude -p`, `codex exec`) when the prompt names a specific entity; does NOT activate in interactive sessions.
  *Prose:* `first-officer-shared-core.md` §Single-Entity Mode.
- **FO-R-061** (MUST) — In single-entity mode, FO skips team creation and uses bare-mode dispatch (the Agent tool without `team_name` blocks until completion, preventing premature `-p` session termination).
  *Prose:* `claude-first-officer-runtime.md` §Team Creation.
- **FO-R-062** (MUST) — In single-entity mode, gates auto-resolve from the stage report recommendation (PASSED → approve; REJECTED with `feedback-to` → auto-bounce; REJECTED without `feedback-to` → report failure and exit).
  *Prose:* `claude-first-officer-runtime.md` §Captain Interaction single-entity exception.
- **FO-R-063** (MUST) — The single-entity gate auto-resolve exception applies ONLY in single-entity mode; in interactive sessions the no-self-approve guardrail is absolute (see FO-R-031).
  *Prose:* `claude-first-officer-runtime.md` §Captain Interaction.

### Enumeration notes (observations that shaped the schema)

- **Count:** 63 requirements drafted, above the ≥30 floor. Captain may want to consolidate or split during review.
- **Coverage:** the four FO files plus guardrails. Workflow-level mechanics (status binary semantics, mod convention details) are covered at the FO-facing surface (FO-R-039/040/041); the status binary's own contract is arguably a separate `STATUS-R-NNN` spec — out of scope for v1.
- **Level distribution:** MUST ≈ 50, MUST_NOT ≈ 12, SHOULD ≈ 1, MAY ≈ 0. Heavy MUST skew reflects that most FO prose is imperative; SHOULD/MAY are rare. Expected for a safety-critical orchestrator.
- **Schema pressure points identified during enumeration:**
  - **Runtime scoping.** Some requirements are Claude-only (FO-R-005 through FO-R-015), some Codex-only (FO-R-023, FO-R-024, FO-R-030), most cross-runtime. Schema adds optional `runtime: [claude|codex|both]` — `both` is the default when absent. This replaces the ad-hoc "(Claude-only)" header prefix.
  - **Level extensions.** None needed. RFC 2119 core (5 levels) covered every observed imperative.
  - **Cross-reference within L1.** FO-R-063 references FO-R-031. Schema adds optional `refines: [ID, ...]` so exception-style requirements that sit atop a base requirement are machine-discoverable, not just prose.
  - **`known_flaky` usage so far:** none of the 63 populate it. That field activates when #202's follow-up re-evaluates open flake tasks against the coverage matrix. Keeping it optional (omit-when-empty) keeps v1 clean.
  - **Anchor-phrase requirements.** FO-R-012 (captain-report verbatim), FO-R-027 (diagnostic anchor phrase), FO-R-034 (gate presentation template) assert LITERAL string presence. Schema adds optional `anchor_phrase: "..."` so tooling/tests can grep for the exact phrase.

### Revised `requirement` YAML schema (with observations folded in)

```yaml
id: FO-R-013
level: MUST  # MUST | MUST_NOT | SHOULD | SHOULD_NOT | MAY
area: dispatch  # from the 12-entry taxonomy above
runtime: [claude]  # optional; [both] when omitted; values: claude | codex | both
summary: "FO emits completion-signal literal when forwarding a dispatch prompt."
anchor_phrase: "does not match next stage effective_model"  # optional; verbatim string that tests can grep
refines: [FO-R-031]  # optional; for exception-style reqs that sit atop a base req
known_flaky:  # optional; omit entirely when empty
  - model: haiku-4-5
    mode: claude-live
    reason: "Haiku paraphrases parenthesis-equals syntax to English."
    tracking_task: 197
```

Deliberately still absent: test list, prose-location list, coverage cells, flake rates. All derived.

## Stage Report: ideation

- DONE: Disentangled spec design.
  Four-layer star schema: L1 spec (stable reqs only), L2 pytest markers in tests, L3 HTML anchors in prose, L4 tooling-derived coverage. Coupling retained and justified: `known_flaky` on L1 as a slow-changing semantic fact distinct from L4 per-run rates.
- DONE: Format decision with tradeoffs.
  L1 = hybrid aggregate markdown with per-requirement YAML block (rejected per-file for directory churn, pure table for loss of rationale). L2 = `@pytest.mark.requirement("FO-R-NNN")` with `--strict-markers` (rejected docstring headers for refactor fragility). L3 = `<!-- FO-R-NNN -->` HTML comments (rejected inline citations for reader-facing leakage, separate xref file for third-source-of-truth drift). Blast radius on rename analyzed; `fo-spec rename` subcommand flagged as tooling requirement.
- DONE: Acceptance criteria + test plan under disentangled design.
  AC-1 through AC-8 rewritten; v0 AC-4 (per-cell flake stamps) removed; v0 AC-5 (static uncovered list) converted to tooling query; AC-7 (tooling) elevated to load-bearing. Test plan: static format check, unit tests per subcommand, one integration drift test, `make test-static` wire-in. No live LLM tests. Bulk prose/test refactor explicitly deferred to follow-ups to contain #202 blast radius.
- DONE: L1 enumeration drafted against real FO contract.
  63 requirements extracted from SKILL.md + shared-core + both runtime adapters + guardrails, organized across a 12-area taxonomy. Enumeration surfaced three schema additions (`runtime`, `anchor_phrase`, `refines`) and validated that RFC 2119's 5 levels suffice. Draft lives in entity body under "L1 draft — requirement enumeration" for captain review before implementation.

### Summary

Redesigned #202 per captain critique: separated stable requirements (L1, spec) from test mapping (L2, pytest markers), prose mapping (L3, HTML comments), and coverage (L4, tooling-derived). Drafted 63 real L1 entries against the current FO contract — the enumeration drove three concrete schema additions (`runtime`, `anchor_phrase`, `refines`) that weren't visible in the abstract. Tooling elevated to load-bearing since coverage is no longer stamped. ACs tightened to ship plumbing + proof-of-life; bulk prose/test adoption deferred to follow-ups. No code written — design lives in entity body.
