---
id: 180
title: Decouple workflow timeline from the code repository
status: ideation
source: "Karen directive during 2026-04-29 session — every entity state change (dispatch / advance / mod-block / complete) commits directly to main without a PR, violating the hygiene rule that every commit on main should be reviewed via a PR. The workflow timeline lives in the same repo as the code it tracks, so timeline writes pollute the protected branch."
started: 2026-04-29T04:07:14Z
completed:
verdict:
score:
worktree:
issue:
pr:
mod-block:
---

## Problem

The first officer commits to the code repo's main on every entity state transition — `dispatch:`, `advance:`, `mod-block:`, `complete:` — and those commits never go through a PR. This violates the hygiene rule that everything landing on main should be reviewed via a PR, and it makes the main branch noisy with non-code state churn.

## Why this matters

- **Branch protection conflict.** Repos that enforce "PR required for main" cannot host a spacedock workflow without exempting the FO, weakening the protection.
- **Audit and history noise.** Code-review tooling, blame, and CHANGELOG generation all see workflow state commits mixed with real code changes.
- **Coupling to git.** The current state log is git-shaped (frontmatter + commits in the code repo). That blocks integrations with surfaces where the captain might prefer to review state — GitHub Projects, Notion databases, Linear projects.

## Direction (re-locked 2026-04-29 with Karen — state-only decoupling)

Two clean concepts:

- **The entity** = the markdown file with id, title, source, design content, stage reports, archive history. **Stays in the code repo** at `docs/plans/{slug}.md` (or wherever the workflow defines it). Browsed and edited like today.
- **The workflow timeline** = the per-entity *state log* — every `status` change, every `dispatch / advance / mod-block / complete` write, every timestamp. **Moves to a separate store.** The code repo never sees these writes again.

This is narrower than the prior framing, which proposed relocating the entire entity file. The entity file stays put. Only the time-varying state moves.

### What's state vs. what stays

| Field | Today | After |
|---|---|---|
| `id`, `title`, `source`, `score`, `issue` | entity file frontmatter | entity file frontmatter (unchanged) |
| `status`, `started`, `completed`, `verdict`, `worktree`, `pr`, `mod-block` | entity file frontmatter | external state store |
| Body content (Problem, Design, `## Stage Report`, `### Feedback Cycles`) | entity file body | entity file body (unchanged) |
| Audit log of state transitions | git log of FO commits on main | external state store |

### What FO commits look like after

| Today's commit pattern | Frequency | After |
|---|---|---|
| `dispatch:` / `advance:` / `mod-block:` / `complete:` | many per entity | Writes to state store. Code repo main never sees these. |
| `task: {slug} ({stage})` (initial entity creation) | rare (captain-driven) | Stays in code repo as a normal file write. Acceptable low-frequency noise; ideation evaluates whether to PR these too. |
| Captain-directed body edits | rare | Same. |
| `_archive/` directory move at terminal stage | once per entity | **Eliminated.** "Archived" becomes a state-store flag, not a file move. |

Net result: code repo main becomes essentially PR-only. Initial entity creation is the only remaining FO-direct write, and it's rare enough to live with — or to PR if the captain wants strict purity.

### What the state store could be (open for ideation)

- A sidecar git repo (small, single-purpose, captain-clonable).
- An orphan branch in the same code repo (no second checkout, but FO juggles the branch).
- A sidecar file per entity (e.g., `.state/{slug}.yml`) synced to an external store.
- A single append-only journal file (`timeline.jsonl`) on a separate branch or repo.
- SQLite DB.
- An external system as canonical (Notion / Linear / GitHub Project) — rejected as canonical store in the prior round; can re-evaluate now that scope is smaller.

Ideation picks one for the first delivery and lays out the data-shape contract so other backends can layer on later as adapters.

## Data-shape contract (state log)

The timeline is a sequence of state events plus the current-state projection of those events. Each event:

| Field | Type | Description |
|---|---|---|
| `workflow_id` | string | Stable identifier for the workflow this event belongs to. Defaults to the workflow directory path (e.g., `docs/plans`); overridable via `workflow-id:` in the workflow `README.md` frontmatter. |
| `entity_id` | string | The entity this event applies to |
| `entity_slug` | string | Convenience denormalization |
| `field` | string | Which state field changed (`status`, `worktree`, `pr`, `mod-block`, `started`, `completed`, `verdict`, `archived`) |
| `old` | string \| null | Previous value |
| `new` | string \| null | New value |
| `timestamp` | ISO 8601 | When the event happened |
| `actor` | string \| null | FO, ensign name, captain, or hook name |
| `commit_msg` | string \| null | Message that would have gone on the FO commit (e.g., `dispatch: 180 entering ideation`) |

Events are uniquely keyed by `(workflow_id, entity_id)`. Current state per entity is the most-recent-by-timestamp projection of these events filtered by `workflow_id`. This is what `status --boot` and `status --next` consume.

For the first delivery, each workflow gets one sidecar git repo, so `workflow_id` is implicit in the repo (every event in `{repo}-timeline` belongs to that workflow) but is still **stamped on every event** so future cross-workflow backends can disambiguate. Future backends — Notion, Linear, GitHub Projects, a shared SQLite db across workflows — need `workflow_id` for disambiguation when one store hosts multiple workflows.

This contract is what later adapters (sidecar git repo, orphan branch, journal file, GitHub Projects, Notion, Linear) read and write against.

## Scope for this delivery

This task changes the spacedock **plugin** so any workflow can store its state log outside the code repo. It does not migrate any specific workflow.

- Move the state log out of code-repo main commits.
- Build the first state-store backend (ideation picks one).
- Define and document the data-shape contract above.
- Eliminate `_archive/` directory moves; "archived" becomes a state flag.
- Document a one-page generic cutover guide so any captain can convert an existing workflow.

**Explicitly out of scope:** moving the entity files themselves, building external-UI adapters (Notion / Linear / GitHub Projects), and migrating `docs/plans/` itself. Those are separate operational tasks the captain runs against the finished plugin.

This task **subsumes #165** (`entity-state-sidecar-file`) — same problem, more general formulation. Implementation closes #165. **#167** (`relocate-status-and-claude-team`) is independent and can run in either order.

## Open questions for ideation

- **State-store backend choice.** Pick one for the first delivery. Default candidate: sidecar git repo (preserves history, captain-inspectable, no new infra). Ideation justifies the choice and notes which alternatives are easy to layer on later.
- **State-store discovery.** How does `status` find the state store from the code repo? Config file (`.spacedock`)? Env var? Sibling-directory convention? Default-path inside the code repo on a separate branch?
- **Frontmatter migration.** Existing entity files have state fields in frontmatter. After cutover, those fields move out. How are stale frontmatter values handled — migration script that strips them, or `status` ignores them and treats the state store as authoritative?
- **`_archive/` elimination.** Today, archived entities live under `docs/plans/_archive/`. After: the file stays in `docs/plans/`, just marked archived in the state store. How does `status --archived` and the existing pr-merge startup-hook scan (which excludes `_archive/`) work?
- **Race conditions.** Two captain sessions or two FOs hitting the same state store simultaneously. Single-writer assumption, optimistic concurrency, or rebase-and-retry?
- **Observability.** Captain wants to see "what happened to entity X over time." How does she view the state log? `status --history {slug}`? A separate viewer? Just `git log` against the sidecar store?

## Prior ideation (discarded direction — entity-relocation framing)

The first ideation pass solved the wrong problem: it proposed relocating the entire entity file (frontmatter + body + reports) into a separate workflow repo. Karen clarified afterward that entity files should stay in the code repo and only the state log needs to move.

Discarded design, staff review, and proposed approach are preserved in git history at commit `01b2944` and earlier. Not reproduced here to keep the captain's view clean.

The new ideation pass starts from the state-only framing in `## Direction` above.

## State-store backend choice (first delivery)

**Pick: sidecar git repo.** A small, single-purpose repo (e.g., `{code-repo-name}-timeline`) cloned next to the code repo. Each entity has its state stored as a flat YAML file plus a per-entity append-only event log file; the FO writes there instead of committing to the code repo's main.

Justification:

- **Zero new infrastructure.** Captain already has git, gh, and a clone workflow. No DB to host, no API key to rotate, no second tool surface to learn. Implementation reuses the same `git -C {sidecar}` invocation pattern the FO already uses for worktrees.
- **Preserves the audit history we already produce.** Today's `dispatch:` / `advance:` / `mod-block:` / `complete:` commit messages map 1:1 onto commits in the sidecar — the captain's mental model for "look at the timeline" stays "git log" — just against a different repo.
- **Captain-inspectable.** Karen can `cd ~/code-repo-timeline && git log` or open the sidecar in any git UI. No new viewer to build for the first delivery.
- **Single-writer is realistic.** The FO is the sole writer. Two captain sessions running two FOs against the same workflow is rare enough that "rebase-and-retry on push" handles it (see Race conditions below).
- **Survives the captain's clone.** Sidecar has no app-server dependency. A fresh checkout on a new laptop = `git clone {code} && git clone {timeline}`.

Easy alternatives to layer on later via the data-shape contract:

| Backend | Adapter difficulty | When to add |
|---|---|---|
| Orphan branch in code repo | Easy — same git plumbing, different ref | When captain wants one-clone setup |
| Single `timeline.jsonl` journal file | Easy — write-line vs. write-files, same event shape | When per-entity files become noisy at scale |
| SQLite | Medium — adapter writes events as rows | When the workflow grows past a few hundred entities |
| GitHub Projects / Notion / Linear | Hard — external API, rate limits, schema drift | When captain wants UI review of state, not git |

The data-shape contract above is what each adapter reads and writes; switching backends is an adapter swap, not a redesign.

## Open questions — answered

### Sidecar bootstrap on first run

When the configured sidecar dir is absent, the FO bootstraps it without captain intervention. Mechanics:

- **No sidecar dir present:** the FO `git init`s it locally at the configured `timeline.path`. First event is the initial commit.
- **Remote configured:** the workflow `README.md` `timeline:` block accepts an optional `remote:` key. When set and the sidecar dir is absent, the FO `git clone`s from `remote:` instead of `git init`. After each event commit, the FO `git push`es to that remote.
- **No remote configured:** push is skipped silently. The captain is told ONCE per session (one captain-visible line at FO startup) that the sidecar is local-only, so a fresh laptop will start with an empty timeline. No per-event warning spam.

This makes "first run on a new laptop" a no-op for captain configuration: clone the code repo, run the FO, the sidecar appears next to it.

### State-store discovery

Three layers, evaluated in order, first hit wins:

1. **Per-workflow config in the workflow `README.md` frontmatter.** Add `timeline:` block with `backend: sidecar-git` and `path: ../{repo}-timeline` (relative to code-repo root). This is the canonical declaration — versioned with the workflow.
2. **Environment override.** `SPACEDOCK_TIMELINE_PATH={absolute-path}` for ad-hoc test runs and CI fixtures.
3. **Default fallback.** When neither is set, the FO falls back to current behavior (commit on code-repo main) and emits a one-line captain-visible warning naming the workflow. This keeps un-migrated workflows working unchanged — opt-in, not flag-day.

Discovery is performed once at FO startup as part of `status --boot` and cached for the session.

### Frontmatter migration (stripped vs. tolerated)

**Tolerated, then stripped.** Two-phase:

- Phase 1 (this delivery): `status --boot` and `status --next` treat the state store as authoritative when configured. **State-store wins on read for the moved fields** (`status`, `started`, `completed`, `verdict`, `worktree`, `mod-block`); the FO never reads those fields from frontmatter once the workflow has a configured timeline. Stale frontmatter values are dead data on read. They're not stripped automatically — captain-edited body content sits next to them, and a flag-day rewrite of every entity file would be its own commit storm.
- **Warning policy:** the FO emits ONE captain-visible warning per session naming the entities that carry stale frontmatter for the moved fields, with a one-line pointer to the Phase-2 strip command. Silent divergence is worse than an audible one. Not per-event spam — exactly one consolidated line at FO startup listing the affected slugs.
- Phase 2 (out of scope, separate task): a `status --strip-stale-frontmatter` one-shot command the captain runs when she wants the entity files cleaned up. This is captain-initiated, single-commit-per-file, PR-able.

Net for first delivery: existing entity files keep working; new entity creation writes minimal frontmatter (only `id`, `title`, `source`, `score`, `issue`).

### `_archive/` elimination mechanics

**`_archive/` directory move is replaced by an `archived` flag in the state store.** Mechanics:

- `status --archive {slug}` stops moving the file. Instead, it writes an `archived: <ISO-8601 UTC>` event to the state store. The file stays in `{workflow_dir}/`.
- `status --archived` (default scan) reads `archived` from the state store, not from `_archive/` directory presence.
- The `pr-merge.md` startup-hook scan that "excludes `_archive/`" becomes "excludes entities with `archived` flag set in state store" — same intent, different lookup.
- Existing entities already in `_archive/` keep working: `status` continues to scan `{workflow_dir}/_archive/` as a back-compat path. **Dual-read collision rule:** filesystem location under `_archive/` is treated as a synonym for `archived` ONLY when the state store has no `archived` event for that slug. State-store events take precedence over filesystem location for archived/unarchived determination — a state-store `unarchived` event after a stale `_archive/` placement wins; a state-store `archived` event when the file still sits in the main workflow dir wins. New archives no longer move.
- The captain can run a follow-up `status --consolidate-archive` (separate task, out of scope) to physically move the legacy `_archive/` contents back if she wants flat layout. Not required for this delivery.

### Push ownership

**Per-event commit, batched push at FO turn-end.** Two-tier write path:

- Each state event (`dispatch:` / `advance:` / `mod-block:` / `complete:` / `archived`) commits to the sidecar locally and immediately. No remote round trip on the hot path. Captains who want to read the timeline mid-turn run `git -C {sidecar} log` once they pull — fast feedback, no latency cost on the FO.
- The `git push` happens once at the end of the FO turn, after every event for that turn has been committed. A slow remote does not gate per-event latency; one round trip per turn instead of per event.
- If the turn-end push fails (rebase rejection, network), the rebase-and-retry rule from `### Race conditions` applies. The local commits are durable regardless — captain re-running the FO replays the unpushed range on the next turn-end push.

Implementation owner is the sidecar adapter, not the FO directly: the adapter exposes `append_event` (commits locally) and `flush` (pushes if remote is configured). The FO calls `flush` once at turn end.

### Race conditions

**Single-writer assumption with rebase-and-retry on conflict.** The FO is the only writer. The conflict surface is two FOs in two captain sessions writing to the same sidecar.

Strategy:

- Sidecar adapter does `git pull --rebase` before each event-append, then `git push`.
- On push rejection (non-fast-forward), retry once: pull-rebase, append again, push.
- On second failure, fail loudly to the captain with the conflicting entity slug. No silent overwrites.
- Document "do not run two FOs against the same workflow concurrently" as the supported model. The retry handles transient races; it doesn't solve sustained concurrent operation.

This matches what Karen already does manually (one FO per workflow per session) and is testable with a deterministic two-writer fixture.

### Worktree sidecar resolution

**Timeline path resolution uses `git rev-parse --git-common-dir`, not `git rev-parse --show-toplevel`.** Inside a worktree, `--show-toplevel` returns the *worktree* root, which would resolve `timeline.path: ../{repo}-timeline` to a sibling of the worktree — wrong directory, possibly nonexistent, and definitely not the same sidecar the main FO writes to. `--git-common-dir` returns the shared `.git` of the main checkout, so `dirname(--git-common-dir)/..` is the main code-repo root regardless of which worktree the caller sits in.

Both worktree workers and the main FO must therefore resolve to the *same* sidecar path. Verified once at FO startup, cached for the session.

**Writer-vs-reader rule, restated:** only the FO writes to the timeline. Worker-side `status --set` invocations from inside a worktree (e.g., a stage worker marking sub-stage progress) update **worktree-local frontmatter only** and remain invisible to the timeline. The state store is an FO-only audit channel; worker frontmatter scribbles inside a worktree are ephemeral until the FO observes them and writes the corresponding event.

### Observability for the captain

**Three views, all captain-runnable from the code repo:**

- **Per-entity history:** `status --history {slug}` prints the event log for one entity in chronological order — one event per line, formatted as `{timestamp} {field}: {old} -> {new} ({actor}, "{commit_msg}")`. Equivalent to today's "git log against entity file."
- **Workflow timeline:** `status --timeline` prints the most recent N events across all entities (default 50). Equivalent to today's `git log main` filtered to FO commits.
- **Raw access:** `git -C {sidecar} log` works as today. The sidecar is a normal git repo.

`status --history` is the captain-facing primary. `--timeline` is for "what changed today across the whole workflow." Both exist behind the data-shape contract, so swapping to a different backend later doesn't change the command surface.

## Acceptance criteria

Each criterion below names how it is tested and what evidence Karen (non-engineer captain) can verify directly.

| # | Criterion | Test | Captain-visible evidence |
|---|---|---|---|
| 1 | After cutover, a 3-entity dry-run produces zero `dispatch:` / `advance:` / `mod-block:` / `complete:` commits on the code repo's `main`. | E2E live test (cheap haiku run) drives 3 entities through 2 stages with timeline backend configured; assertion is `git log main --grep '^(dispatch\|advance\|mod-block\|complete):' --since={test-start}` returns zero lines. | Karen runs `git log main --oneline -20` after the test and sees only her own commits — no `dispatch:` / `advance:` lines. |
| 2 | The same 3-entity dry-run produces a populated state log in the sidecar with one event per state transition. | Same E2E test asserts the sidecar repo has commits matching the entity slugs and the event count matches the state transitions counted in test logs. | Karen runs `cd ~/{repo}-timeline && git log --oneline` after the test and sees commit messages mirroring today's `dispatch:` / `advance:` lines. |
| 3 | `status --boot` reads from the state store when configured, and falls back to frontmatter with a captain-visible warning when not configured. | Static unit test on `status` script: configured-store fixture returns store-derived state; unconfigured fixture emits the warning string and returns frontmatter state. | Karen runs `status --boot` against an un-migrated workflow and sees the warning line; runs against a migrated workflow and sees the migrated state with no warning. |
| 4 | `_archive/` directory moves are eliminated for new archivals. Entities advancing to terminal stay in `{workflow_dir}/` with `archived` set in the state store. | E2E live test that drives one entity to terminal and asserts (a) no file under `{workflow_dir}/_archive/`, (b) state store has `archived: {timestamp}`, (c) `status --archived` lists the entity. | Karen archives an entity, looks at `docs/plans/`, and sees the file still in the main directory (not in `_archive/`); `status --archived` still lists it. |
| 5 | Concurrent writers (two FO processes against same sidecar) do not silently lose events. | Static integration test: two parallel processes append events; after both finish, the sidecar log contains exactly the union of events with no duplicates and no drops. Failure case (sustained collision after retry) surfaces a loud error. | Engineering invariant — no captain evidence. (The closest captain-runnable equivalent would be running two FO sessions in parallel and inspecting `status --history {slug}` for the union, but that's not part of the routine acceptance run; the test is the only intended audit point.) |
| 6 | `status --history {slug}` prints the entity's full state-transition history sourced from the state store. | Static unit test: pre-seeded sidecar fixture; assert `--history` output matches expected timestamps and field deltas line-for-line. | Karen runs `status --history {slug}` on any active entity and sees a chronological list of state changes, one per line, in human-readable form. |
| 7 | An existing entity with stale frontmatter state fields keeps working — `status --boot` reads from the state store, ignores the stale frontmatter on read, and emits ONE consolidated captain-visible warning per session naming the affected slugs and pointing at the Phase-2 strip command. | Static unit test with a fixture entity carrying both old frontmatter values and state-store events; assert state-store wins on read, exactly one warning line is emitted naming the slug, and a second `status --boot` in the same session does NOT re-emit the warning. | Karen opens an existing entity file with stale `status:` / `started:` in frontmatter, runs `status --boot`, and sees (a) the entity reported with its current (state-store) status — not the stale one — and (b) one warning line listing the slug as eligible for the Phase-2 strip. |

Cap of 7. Each is concretely testable and produces a captain-readable artifact.

## Test plan

Keyed to `tests/README.md` harness selection. Spot-check rule: run the cheap dry-run before any expensive live E2E.

| Layer | Test file (proposed) | Harness | Cost | When to run |
|---|---|---|---|---|
| **Static unit — state-store adapter** | `tests/test_timeline_sidecar_adapter.py` | offline (no Claude/Codex) | ~free, ~5s | Every commit; `make test-static` |
| **Static unit — `status --history` / `status --timeline`** | `tests/test_status_history.py`, `tests/test_status_timeline.py` | offline | ~free, ~5s | Every commit |
| **Static unit — frontmatter tolerated read** | `tests/test_status_frontmatter_tolerated.py` | offline | ~free, ~3s | Every commit |
| **Static unit — `_archive/` elimination + back-compat scan** | `tests/test_archive_flag_in_state_store.py` | offline | ~free, ~3s | Every commit |
| **Static integration — concurrent writers** | `tests/test_timeline_concurrent_writers.py` | offline (subprocess fork) | ~free, ~10s | Every commit |
| **Static perf — per-event commit latency** | `tests/test_timeline_event_commit_perf.py` | offline (local sidecar, no remote) | ~free, ~3s | Every commit; asserts `dispatch:` event commit completes in under 200ms locally on a freshly-init'd sidecar (push is excluded — push is batched at turn-end per `### Push ownership`) |
| **Cheap dry-run live E2E** | extend `tests/test_gate_guardrail.py` with timeline fixture variant | `test_lib.py` + `run_first_officer`, haiku, `live_claude` + `serial` | ~$0.02, ~60s | Before any expensive live run; matches existing shared-runtime pilot pattern |
| **Full live E2E — main-branch-clean assertion** | `tests/test_timeline_no_main_state_commits.py` | `test_lib.py` + `run_first_officer`, haiku, `live_claude` | ~$0.05, ~90s | PR live-tier run (`make test-live-claude`) |
| **Full live E2E — Codex parity** | same test, `--runtime codex` | `run_codex_first_officer`, `live_codex` | ~$0.05, ~90s | `make test-live-codex` |

Cheap dry-run gating: the extension to `test_gate_guardrail.py` is the cheapest live signal. If it fails, the expensive parallel-tier tests do not need to run. Matches the existing "shared runtime pilot" discipline in `tests/README.md`.

E2E is required for criteria 1, 2, and 4 — they're behavioral guarantees about commit shape and filesystem state under a real FO run. Criteria 3, 5, 6, 7 are static-test territory.

Risk-proportional: this is a structural change to scaffolding (status binary, FO contract, mod files), so it earns full E2E. The dry-run-first rule keeps the cost bounded.

## Proposed approach

Concrete artifacts at design level only — no code in this section.

**#167 ordering note:** if #167 (`relocate-status-and-claude-team-to-first-officer-skill`) lands first, write the new `timeline` and `timeline_sidecar.py` modules under the relocated path (`skills/first-officer/bin/`) instead of `skills/commission/bin/`. The implementation worker checks the on-disk location of `status` at dispatch time and follows it.

### New files

- `skills/commission/bin/timeline` — Python 3 stdlib module/library imported by `status`. Exposes `read_state(slug)`, `append_event(slug, field, old, new, actor, commit_msg)`, `iter_history(slug)`, `iter_timeline(limit)`. Backend-dispatching: reads `timeline.backend` from workflow `README.md` frontmatter, instantiates the right adapter.
- `skills/commission/bin/timeline_sidecar.py` — sidecar-git-repo adapter implementing the timeline interface. Per-entity event-log files; `git -C {sidecar} pull --rebase` / `commit` / `push` plumbing; rebase-and-retry once on conflict.
- `tests/test_timeline_sidecar_adapter.py`, `tests/test_timeline_concurrent_writers.py`, `tests/test_status_history.py`, `tests/test_status_timeline.py`, `tests/test_status_frontmatter_tolerated.py`, `tests/test_archive_flag_in_state_store.py`, `tests/test_timeline_no_main_state_commits.py` — see test plan above.
- `tests/fixtures/timeline-sidecar/` — minimal workflow fixture configured with a sidecar timeline, one pre-seeded entity, used by the live E2E tests.
- `docs/cutover-guide.md` — one-page captain-facing migration guide: how to clone a sidecar, how to set the `timeline:` block in `README.md`, how to run the optional Phase-2 frontmatter-strip later.

### Modified files (scaffolding-protected — must dispatch a worker)

- `skills/commission/bin/status` — add `--history`, `--timeline`, plumb `--set` and `--archive` through the timeline adapter when configured; preserve current behavior when not configured.
- `skills/first-officer/references/first-officer-shared-core.md` — replace "Commit the state transition on main with `dispatch:` / `advance:` / `mod-block:`" with "Append the state transition to the timeline." Update FO Write Scope to remove state-transition commits from the main-write list. **Specific line pins** (sourced from staff review, must be re-confirmed by the implementation worker before edit):
  - Line 68 — Dispatch step 6: `Commit the state transition on main with 'dispatch: {slug} entering {next_stage}'.` → "Append the state transition to the timeline (`dispatch: {slug} entering {next_stage}`)."
  - Line 113 — Reuse path: `Update frontmatter on main (..., commit: 'advance: {slug} entering {next_stage}').` → re-route to timeline append.
  - Lines 150 & 157 — Mod-block lifecycle: `Commit: 'mod-block: ...'.` → timeline append.
  - Line 168 — State Management: `Commit state changes at dispatch and merge boundaries.` → "Append state events to the timeline at dispatch and merge boundaries."
  - Line 184 — FO Write Scope: `**State-transition commits** — dispatch, advance, merge boundary commits` — pull this bullet OUT of the allowed-on-main list and replace with "**State-transition events** — dispatch, advance, merge-boundary, mod-block, archive events go to the configured timeline; the FO never commits these to main."
- `skills/first-officer/references/claude-first-officer-runtime.md`, `skills/first-officer/references/codex-first-officer-runtime.md` — same prose update for runtime-specific dispatch sections. **Specific line pin:** `skills/first-officer/references/codex-first-officer-runtime.md:177` — the runtime "perform[s] local merge, archive, terminal commit, and worktree cleanup." The `terminal commit` term must be re-mapped to a timeline event (`complete:` event) plus, if any code-side terminal write is still required, a normal main-eligible commit; the archive piece becomes a timeline `archived` event, not an `_archive/` move.
- `mods/pr-merge.md` AND `docs/plans/_mods/pr-merge.md` — both files identical, both must be updated. **Specific line pin:** line 38 in each — `First, push main to ensure the remote is up to date with local state commits: 'git push origin main'. Then rebase the worktree branch onto main`. This step goes away cleanly once state commits leave main; the rebase plumbing simplifies. The live workflow copy at `docs/plans/_mods/pr-merge.md` and the plugin template at `mods/pr-merge.md` both need the edit (refit propagates `mods/` → `_mods/` for new workflows; existing live copies must be touched explicitly). Also change `## Hook: startup` and `## Hook: idle` to read the `archived` flag from the state store instead of excluding `_archive/`.
- `agents/first-officer.md`, `agents/ensign.md` — minor: drop language about "commit state transition on main."
- `plugin.json`, workflow `README.md` template — add the `timeline:` frontmatter block with documented schema.

No edits to entity files in `docs/plans/`. No move of `docs/plans/` itself. The `_archive/` directory stays as a back-compat read path.

## Scaffolding-protected surfaces touched

This work touches scaffolding. Per `code-project-guardrails.md`, the FO must dispatch a worker for these — direct FO edits on main are off-limits.

- `skills/commission/bin/status` — status binary
- `skills/commission/bin/timeline*` — new (still under `skills/`, scaffolding-protected)
- `skills/first-officer/references/first-officer-shared-core.md`
- `skills/first-officer/references/claude-first-officer-runtime.md`
- `skills/first-officer/references/codex-first-officer-runtime.md`
- `mods/pr-merge.md`
- `agents/first-officer.md`, `agents/ensign.md`
- `plugin.json`
- workflow `README.md` template (the commissioned-by template, not arbitrary entity files)
- new test files under `tests/`

Implementation stage(s) for this entity must be worktree-backed and dispatched, not direct on main.

## Stage Report: ideation

- DONE: Read the entity body in full. Direction stays state-only; no entity-file relocation reintroduced.
  Confirmed by reading the file end-to-end before drafting; only the state-only framing is referenced below.
- DONE: Pick a state-store backend for the first delivery and justify.
  Sidecar git repo picked; alternatives layered via the data-shape contract documented in `## State-store backend choice`.
- DONE: Answer each `## Open questions for ideation` item.
  Five answers in `## Open questions — answered`: discovery, frontmatter migration, `_archive/` mechanics, race conditions, observability.
- DONE: Produce `## Acceptance criteria` with how-tested + captain-visible evidence per criterion.
  7 criteria; each has a test column and a captain-visible-evidence column; criterion 1 is the captain-runnable `git log main` check the FO instructions called out.
- DONE: Produce `## Test plan` keyed to `tests/README.md` harness selection with cost estimates and a cheap dry-run named.
  Static layer + dry-run-first live extension on `test_gate_guardrail.py` (~$0.02), then full live E2E (~$0.05 each per runtime).
- DONE: Produce `## Proposed approach` listing concrete artifacts at design level only.
  New files (`timeline`, `timeline_sidecar.py`, six test files, fixture, cutover guide); modified scaffolding files (status binary, FO references, runtime adapters, pr-merge mod, agents, plugin.json, workflow README template).
- DONE: List scaffolding-protected surfaces; note implementation must dispatch.
  `## Scaffolding-protected surfaces touched` section; explicit "implementation stage(s) must be worktree-backed and dispatched."
- DONE: Append `## Stage Report` summarizing checklist with DONE/SKIPPED/FAILED.
  This section.

### Summary

Ideation re-anchored on state-only decoupling: the entity file stays in the code repo; only the state log moves out. First delivery uses a sidecar git repo (justified for zero new infrastructure, captain-inspectable via `git log`), with a documented data-shape contract that makes orphan-branch / journal-file / SQLite / external-UI adapters drop-in swaps. `_archive/` becomes a state flag, not a file move. The acceptance criteria lead with the captain-runnable `git log main` zero-state-commits check; the test plan front-loads cheap static and dry-run live signals before any expensive E2E. All implementation surfaces are scaffolding-protected and must dispatch through workers.

## Staff Review (redo)

### Reviewer summary

APPROVE WITH NOTES. The state-only re-framing is correct and the sidecar-git-repo pick is defensible for a first delivery. But several mechanical questions are under-specified to a degree that the implementation worker will hit them on day one and have to make policy decisions on the fly: bootstrap of the sidecar (does it auto-clone/auto-init, what if there's no remote?), the precedence rule when frontmatter and state-store disagree, and the dual-source `_archive/` resolution. Acceptance criterion 5 also has a clearly engineer-facing "evidence" line that needs to be re-cast or dropped to captain. Fix those gaps before dispatching implementation.

### Design soundness

- **Sidecar bootstrap is a missing leg.** The doc describes "pull --rebase / commit / push" but never the first run. What does the FO do when `../{repo}-timeline` doesn't exist on the captain's machine? Auto-`git init`? Auto-`git clone {remote}`? Where would it derive that remote? And when there is no remote at all (single-laptop captain), the push step has to be skipped — the doc treats push as unconditional in the race-conditions section. Spell out: bootstrap = "init local if absent; remote optional; push only when `origin` is configured." Without that, criterion 1's E2E will trip on the first laptop without a sibling repo.
- **Frontmatter-vs-state-store precedence has a real race.** Section says "store wins on read." Fine for steady state. But during the transition window, an entity created today carries `status: ideation` in frontmatter; tomorrow the FO advances it and writes only the state-store event. The frontmatter still says `ideation`. If a captain edits the entity body in-between (a normal operation), git diff shows the file has changed but `status:` is stale — not the FO's bug, but it will look like one. Worth a one-line rule: "the FO never reads a moved field from frontmatter once the workflow has a configured timeline; the stale value is dead data." Also: criterion 7 says "no warning, no error" on stale frontmatter — but at minimum the captain should see *one* warning per session that the entity has stale frontmatter eligible for the Phase-2 strip. Silent divergence is worse than an audible one.
- **`_archive/` dual-read collision rule is missing.** The doc says back-compat scan continues to read `_archive/{slug}.md` AND that `archived` becomes a state-store flag. What happens when both are true (entity file under `_archive/` *and* an `archived` event in the store)? What happens when only the file is in `_archive/` but the store has no event — does the back-compat path synthesize an `archived` event for projection, or does `status --archived` just OR the two sources? Pick a rule and write it down. Recommended: "filesystem location is treated as a synonym only when the state store has no `archived` event for that slug." Otherwise live state-store data gets shadowed by a stale `_archive/` move.
- **Discovery via README frontmatter is mechanically feasible** — verified `skills/commission/bin/status:165-329` already has `parse_stages_block` and `parse_stages_with_defaults` walking the README YAML, so adding a `timeline:` block is a localized parser extension, not new scaffolding. Good.
- **Push ownership is implicit.** The proposal says the FO "writes events to the store" but never names the actor for the `git push`. Adapter? FO? Per-event or batch-at-end-of-turn? Per-event push is simple but slow (a remote round trip per `dispatch:` / `advance:`); batched push at FO turn end is faster but loses the "captain `git log` mirrors the timeline" property mid-turn. Pick one — the test plan asserts captain visibility "after the test," which doesn't disambiguate.

### Test plan sufficiency

- **No test for sidecar absent on first run.** Criterion 1 assumes the sidecar already exists. Add: "FO with `timeline:` configured but sidecar dir missing — does it auto-create or fail loudly?" This is the captain's first-laptop experience.
- **No test for missing/configured-but-no-remote sidecar.** Push will fail on a freshly-init'd local-only sidecar. The retry loop in race-conditions presumes a remote.
- **No test for state-store-vs-frontmatter disagreement.** Criterion 7 covers stale frontmatter being ignored when store is authoritative. Add the inverse: "store is empty for this entity, frontmatter has values" — does the FO treat the entity as un-migrated and fall back, or does it write fresh events? Migration-window correctness depends on this.
- **`timeline:` block malformed/missing.** Criterion 3 covers "not configured → warning + frontmatter fallback." Add: "block present but malformed (e.g., `backend: typo-name`) → loud error, not silent fallback."
- **Concurrent-writers test is realistic but the perf cost is not addressed.** `git pull --rebase` per event is a hidden performance regression that's not in the test plan. Today, `dispatch:` is a local commit (~50ms). After: pull-rebase + commit + push could be 1-3s per event over a slow network. A 10-entity dispatch loop becomes 30s of git plumbing. Add a perf assertion or at least a measured baseline so the captain knows what she's signing up for.
- **Worktree-stage worker writing state from inside a worktree.** Criterion 4's E2E drives "one entity to terminal" but doesn't flag whether the terminal write originated from a worktree or main. The worktree case is the riskier one — see worktree compatibility below.

### Gaps

- **`mods/pr-merge.md` `git push origin main` step exists and dropping it is safe.** Verified at `mods/pr-merge.md:38` and `docs/plans/_mods/pr-merge.md:38` — both files identical, both contain `First, push main to ensure the remote is up to date with local state commits: 'git push origin main'. Then rebase the worktree branch onto main`. The rationale ("up to date with local state commits") goes away cleanly once state commits leave main, so dropping the step is sound. Note both copies need updating: the proposal cites `mods/pr-merge.md` (singular template), but the live workflow copy is at `docs/plans/_mods/pr-merge.md`. Confirm refit propagates the template change to existing live copies, or call out that the live copy is updated separately.
- **The "commit state transition on main" prose is at specific lines and needs concrete naming.** Found three load-bearing call sites in `skills/first-officer/references/first-officer-shared-core.md`:
  - Line 68 — Dispatch step 6: `Commit the state transition on main with 'dispatch: {slug} entering {next_stage}'.`
  - Line 113 — Reuse path: `Update frontmatter on main (..., commit: 'advance: {slug} entering {next_stage}').`
  - Lines 150 & 157 — Mod-block lifecycle: `Commit: 'mod-block: ...'.`
  - Line 168 — State Management: `Commit state changes at dispatch and merge boundaries.`
  - Line 184 — FO Write Scope: `**State-transition commits** — dispatch, advance, merge boundary commits`. This bullet has to come *out* of the allowed-on-main list and be replaced with "Append state-transition events to the configured timeline." The proposal should call these line numbers explicitly so the implementation worker doesn't have to re-discover them.
- **Codex runtime adapter's terminal-commit reference.** `skills/first-officer/references/codex-first-officer-runtime.md:177` says the runtime "perform[s] local merge, archive, terminal commit, and worktree cleanup." That `terminal commit` term needs to be re-mapped to a timeline event too — proposal mentions Codex parity but doesn't pin the line.
- **#167 is tangentially in conflict.** #167 (`relocate-status-and-claude-team-to-first-officer-skill.md`) plans to move `skills/commission/bin/status` → `skills/first-officer/bin/status`. This work adds `skills/commission/bin/timeline` and `skills/commission/bin/timeline_sidecar.py` next to the existing `status`. If #167 lands first, the new files have to be authored at the new location. If this lands first, #167 has to relocate two more files. The proposal says "#167 is independent and can run in either order" — true, but the implementation plan should add a "if #167 has already moved status, write the new modules under the new dir" line. One-line note, prevents a worker writing to the wrong path.
- **#165 is correctly subsumed.** Verified — #165's "(a) Sidecar state file" recommendation maps cleanly onto this delivery's broader framing. No conflict.
- **Per-entity event-log file shape is unspecified.** "Flat YAML file plus per-entity append-only event log" is named but not schema'd. The data-shape contract table specifies the *event* fields; it does not specify the *file* layout (one YAML doc per file? JSONL? what's the filename convention — `{slug}.events.yaml`?). The implementation worker has to pick. Pin one.

### Captain-visible evidence quality

- **Criterion 5's evidence is engineer-disguised.** The body explicitly admits this: `"Captain can read the test output asserting 'expected 6 events, found 6' — the engineer-facing test is the evidence."` Reading test output of a concurrent-writer integration test is not captain evidence. Either re-cast as something captain-runnable (e.g., "captain runs `status --history {slug}` after a manual two-session collision and sees the union of events") or accept that this criterion is engineer-internal and remove the captain-visible column. The honest move is the latter — flag it as an engineering invariant rather than pretending it's captain-checkable.
- **Criteria 1, 2, 4 are genuinely captain-visible.** `git log main --oneline -20`, `git log` against the sidecar, and "look in `docs/plans/`" are all things Karen does today.
- **Criteria 3, 6, 7 are borderline.** They say "Karen runs `status --boot`/`--history` and sees X." That's captain-visible only if Karen knows what to look for. Add the *expected line* in plain English to each row, not just the operation.

### Worktree compatibility

- **Sidecar resolution from inside a worktree is not specified.** `skills/commission/bin/status:349-378` (`resolve_active_entity_path`, `load_active_entity_fields`) already handles worktree-vs-main reads of entity fields — but those read frontmatter from a path inside the worktree. The new timeline lookup has to find the sidecar from a worktree CWD. If the `timeline:` config is read from the workflow `README.md` and the path is "relative to code-repo root," does the worker resolve "code-repo root" via `git rev-parse --show-toplevel` (which inside a worktree returns the *worktree* root, not the main checkout)? Spell out: timeline path resolution must use `git rev-parse --git-common-dir` or equivalent so worktree workers point at the same sidecar as the main FO. Otherwise two workers from two worktrees will resolve to different (or nonexistent) sidecar paths.
- **Worker-side `status --set` calls during worktree stages should NOT write timeline events.** Today, worktree-stage workers write frontmatter into the worktree copy (e.g., to mark stage progress). After cutover, those writes have to either (a) bypass the timeline and stay frontmatter-local, or (b) all route to the central timeline. The doc doesn't pick. Recommended: only the FO writes timeline events; worker-side `status --set` inside a worktree updates the worktree-local frontmatter and remains invisible to the timeline. State-store events are an FO-only audit channel.

### Recommendation

APPROVE WITH NOTES — the direction is correct and the sidecar pick is sound, but the bootstrap-on-first-run rule, the dual-read precedence rules (frontmatter/store and `_archive/`/store), criterion 5's captain-evidence honesty, the worktree-resolution rule, and the explicit line-number pins for the FO-reference edits all need to be added before an implementation worker can be dispatched without making policy on the fly.

## Stage Report: ideation (tightening pass)

This pass is a body-tightening within ideation — no design redirection. The locked direction (state-only decoupling, sidecar git first delivery, `workflow_id` in event schema, entity files stay in code repo) is fixed. The 6 staff-review gaps + Karen's `workflow_id` catch are now resolved in prose so the implementation worker has zero policy decisions left.

1. **DONE** — Read the entity body in full. Locked direction (state-only decoupling, sidecar git first delivery) and the existing `## Open questions — answered`, `## Acceptance criteria`, `## Test plan`, `## Proposed approach`, `## Scaffolding-protected surfaces touched`, `## Stage Report`, and `## Staff Review (redo)` sections all stay; tightening happened WITHIN them; no relitigation.
2. **DONE** — Tightening gap 1 (sidecar bootstrap on first run). Added subsection `### Sidecar bootstrap on first run` to `## Open questions — answered` with the three rules: (a) absent dir → `git init` locally; (b) optional `remote:` in `timeline:` block triggers `git clone` when absent and `git push` after each event; (c) no remote → push skipped silently with one captain-visible startup line per session.
3. **DONE** — Tightening gap 2 (frontmatter-vs-store precedence and warning policy). Updated `### Frontmatter migration (stripped vs. tolerated)` to make state-store-wins-on-read explicit for the moved fields and to specify ONE consolidated warning per session naming affected slugs. Updated Acceptance criterion 7 to assert the warning IS emitted (not silent) and that a second `--boot` in the same session does not re-emit.
4. **DONE** — Tightening gap 3 (`_archive/` dual-read collision rule). Updated `### `_archive/` elimination mechanics` so filesystem location is treated as a synonym for `archived` ONLY when the state store has no `archived` event for that slug; state-store events take precedence over filesystem location for archived/unarchived determination.
5. **DONE** — Tightening gap 4 (push ownership). Added `### Push ownership` subsection to `## Open questions — answered`: per-event commit, batched push at FO turn-end. Added a perf-assertion row to the test plan for `dispatch:` event commit completing in under 200ms locally on a freshly-init'd sidecar (push excluded — push is batched at turn-end).
6. **DONE** — Tightening gap 5 (worktree sidecar resolution). Added `### Worktree sidecar resolution` subsection: timeline path resolution uses `git rev-parse --git-common-dir` (NOT `--show-toplevel`); only the FO writes to the timeline; worker-side `status --set` inside a worktree updates worktree-local frontmatter only and remains invisible to the timeline.
7. **DONE** — Tightening gap 6 (`workflow_id` in event schema, Karen's catch). Updated `## Data-shape contract (state log)` table to add `workflow_id` (string, defaults to workflow directory path, overridable via `workflow-id:` in workflow `README.md` frontmatter). Updated narrative: events keyed by `(workflow_id, entity_id)`, first delivery uses one sidecar per workflow so `workflow_id` is implicit-but-stamped, future cross-workflow backends need it for disambiguation.
8. **DONE** — Honesty fix on Acceptance criterion 5. Re-cast captain-visible-evidence as "Engineering invariant — no captain evidence," with a parenthetical noting the closest captain-runnable equivalent (running two FO sessions in parallel + `status --history`) is not part of the routine acceptance run.
9. **DONE** — Added explicit line-number pins to `## Proposed approach` > `### Modified files (scaffolding-protected — must dispatch a worker)`:
   - `skills/first-officer/references/first-officer-shared-core.md` lines 68, 113, 150, 157, 168, 184 with the specific old-text → new-text mapping.
   - `skills/first-officer/references/codex-first-officer-runtime.md` line 177 (the `terminal commit` term).
   - `mods/pr-merge.md` and `docs/plans/_mods/pr-merge.md` both line 38 (push-main-before-rebase step). Noted both files are identical and both must be updated; live workflow copy is separate from plugin template.
10. **DONE** — Added a one-line `#167` ordering note to `## Proposed approach`: if `#167` lands first, write the new `timeline` and `timeline_sidecar.py` modules under `skills/first-officer/bin/` instead of `skills/commission/bin/`. Implementation worker checks `status`'s on-disk location at dispatch time.

### Summary

Body-tightening pass complete: every gap surfaced by the staff reviewer (sidecar bootstrap, frontmatter precedence + warning policy, `_archive/` dual-read collision, push ownership, worktree resolution, line-number pins for the FO/runtime/pr-merge edits, criterion-5 honesty fix, `#167` ordering note) plus Karen's `workflow_id` catch (added to data-shape contract with disambiguation rationale) is now spelled out in the ideation body. The locked direction and the prior Stage Report / Staff Review (redo) sections are preserved untouched as audit history. Implementation worker has zero remaining policy decisions; FO can re-present at the ideation gate for Karen + CL approval.
