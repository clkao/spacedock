---
id: jsacmxpkcwp3vg2t60yjdc4y
title: Agent-facing output modes — JSON, field projection, quiet (token-frugal)
status: done
source: session 1 debrief — rtk-inspired
score: "0.38"
worktree: 
started: 2026-05-30T19:17:39Z
completed: 2026-05-30T21:11:18Z
verdict: PASSED
archived: 2026-05-30T21:11:18Z
---

Give `spacedock status` agent-facing output modes so first officers and ensigns parse state cheaply and robustly instead of scraping a padded table. Inspired by rtk (Rust Token Killer): the agent surface should be token-frugal and structured. FLYWHEEL: the FO reads status on every boot / dispatchable check / gate across a long autonomous sprint, so per-read cost compounds.

## Acceptance criteria

**AC-1 - `spacedock status --json` emits valid, stable JSON for the read commands (default table, --next, --archived, --where, --resolve, --boot sections) that an agent can parse without table glyphs/padding.**
Verified by: Go tests assert the JSON shape per command and that it round-trips the same fields as the table; documented schema.

**AC-2 - `--fields a,b,c` projects only the requested columns (table and JSON), so an agent can request the minimal set.**
Verified by: tests that `--fields id,status` emits only those keys/columns in both modes.

**AC-3 - `--quiet` suppresses non-essential chrome for hook/script use (e.g. mutation narration reduced to a machine line), without losing exit-code semantics.**
Verified by: tests on `--set`/`--archive --quiet` output + exit codes.

**AC-4 - Output is proxy-survivable / structured.**
Verified by: a note + test that `--json` output is byte-stable and free of decorative formatting that a token-optimizing proxy (e.g. rtk) would mangle; the FO contract is updated to prefer `--json`/`--fields` for its own reads.

## Test gates

- `go test ./...`
- `--json` schema tests per read command; `--fields` projection tests; `--quiet` mutation tests; FO-contract update to consume `--json`.

## Notes

rtk this session mangled `git log` output and summarized `go test` — a tool meant to be agent-driven should emit structured output that survives proxying, or ship an rtk profile. Highest-leverage of the ergonomic items because the FO itself is the heaviest reader; worth pulling early in the sprint.

## Ideation

### Spike findings (behavioral grounding for this revision)

Before revising, I built the binary (`go build -o /tmp/sd-aom ./cmd/spacedock`) and exercised the live commands. Observed facts that this revision is built on:

- **`--fields id,status` emits DUPLICATE columns** — confirmed: the default table shows `ID SLUG STATUS TITLE SCORE SOURCE` then appends `ID STATUS` again as extras (id/status rendered twice). This is a today-latent bug `resolveExtraFields`+`printStatusTable` carries: extras are not de-duped against defaults.
- **`--boot` is non-deterministic by construction** — confirmed: two back-to-back runs gave `NEXT_ID: tw35xdtze768r1mf4q47r47n` then `NEXT_ID: a2an2kt9rnzwjxd2pdredy8p` (sd-b32 candidates are SHA-derived over `time.Now()`); `TEAM_STATE` reported `present: true / hint: recent team directory: …` (mtime+HOME dependent); `PR_STATE` shells to live `gh`. There is no single static table to byte-compare `--boot` against — the established `internal/status/nextid_boot_test.go` already handles this with structural + section parity and `<ID>`-normalization of the volatile material.
- **Computed `--next` columns are NOT projectable via `--fields`** — confirmed: `--next --fields current,next` renders the appended CURRENT/NEXT extra columns BLANK (`formatExtraCell(e.fields["current"])` reads a nonexistent frontmatter key), while `--next --fields status` renders the frontmatter `status` value (`implementation`). So `--fields` projects frontmatter keys only; the computed `current`/`next`/`worktree` exist solely as `--next` table headers.
- **Extra columns truncate; default columns do not** — confirmed: TITLE as a default column prints the full 61-char title un-truncated (just unpadded past width 30); `--fields title` (TITLE as an EXTRA) truncates to 20 runes with `…` (`CLI ergonomics — wo…`). The default-column "width" is a minimum (`padRight`), not a cap; only `formatExtraCell` is lossy.

### Framing and scope discipline

`--json` is a stable projection of the same data the table already renders, not a new data model. Every JSON field traces to a value the existing formatter prints: a frontmatter field (`id`, `slug`, `status`, `title`, `score`, `source`, plus any extra under `--fields`/`--all-fields`) or a computed read-time value the table already shows (`--next`'s `current`/`next`/`worktree`; `--boot`'s section probes). No new computation, no new fields the table cannot show. This keeps `--json` a serialization choice, observable purely as output bytes.

**Behavioral-proof discipline (load-bearing).** Every AC below is provable by exercise-and-observe: run the command, assert output bytes / exit code / resulting on-disk state. No AC rests on a grep over prose or code as its sole oracle — a prose-grep proves wording, not behavior. Each AC names its behavioral oracle explicitly. The one docs deliverable (the FO-contract edit) is the single exception, and even it gets a *positive* assertion strong enough to detect both the switch and a later revert (see AC-4), not a presence-of-substring check that a superstring would falsely satisfy.

The round-trip property is the correctness anchor for the **deterministic, full-value** read commands (default table, `--next`, `--archived`, `--where`), with one carve-out: round-trip parity compares JSON against the table's **default (full-value) columns only**, never against truncated extra cells (the spike shows extras are lossy 20-rune ellipses while JSON is lossless). `--boot` and `--resolve` are handled per their own rules below — `--boot` is not byte-deterministic and has no single table to round-trip against.

Three orthogonal output knobs, each independently testable:

- `--json`: serialization format (table text → JSON). Applies to read commands.
- `--fields a,b,c`: column/key projection. Already implemented for the table (`resolveExtraFields` → `extras`); `--json` reuses the exact same resolved field set, so projection composes for free in both modes.
- `--quiet`: chrome suppression. On reads it is a near-no-op for `--json` (JSON has no decorative chrome); its real job is on mutations (`--set`/`--archive`) and on the human table (drop header/separator rows).

These compose as a matrix: `--json --fields id,status` is "project then serialize"; `--set … --quiet` is "mutate, emit one machine line."

### `--json` shape per read command

Top-level shape is a small, fixed set of envelopes. Two principles: (1) collections are JSON arrays of flat string-valued objects keyed by the same field names the table uses (lowercased table headers); (2) **every value is a JSON string — no exceptions.** Entity frontmatter is textual and the table never coerces types, so `score` stays `"0.38"`. The earlier draft contradicted itself by allowing a boolean `team_state.present` and a numeric `min_prefix`; this revision removes that exception. `min_prefix` is `"2"`, `team_state.present` is `"true"`/`"false"`, and `--json --validate`'s `valid` is `"true"`/`"false"` (all strings). A uniform "every value is a string" contract is byte-stable, lets the FO parse with one rule, and removes the type-vs-example contradiction the audit flagged. No nulls, no numbers, no booleans anywhere in the JSON.

Determinism rules (shared by every command):
- Marshal with a fixed key order, not Go's `map` iteration. Emit objects as an explicit ordered key list (the resolved field order: defaults first in their canonical order, then `--fields` order, or sorted `--all-fields`), so byte output is reproducible run-to-run.
- One JSON document, newline-terminated, no indentation (compact). A token proxy cannot "summarize" a single compact line-delimited document the way it collapses padded tables.
- Array element order is the existing sort order (`sortDefault` / `sortNext`), already deterministic and stable.
- HTML escaping off (`SetEscapeHTML(false)`) so `&`, `<`, `>` in titles/sources survive as written, matching the table.

Per command:

- **default table** → `{"command":"status","entities":[{"id":…,"slug":…,"status":…,"title":…,"source":…,"score":…}, …]}`. Keys are the resolved field set (defaults in table order `id,slug,status,title,source,score`, or projected — see `--fields`). Array order = `sortDefault`. All values strings.
- **`--next`** → `{"command":"next","dispatchable":[{"id":…,"slug":…,"current":…,"next":…,"worktree":…}, …]}`. The three computed values come straight from `computeDispatchable`: `current` = the entity's `status`, `next` = the computed next stage, `worktree` = `"yes"`/`"no"`. **`--next` field vocabulary (resolved from spike):** the fixed JSON keys are `id,slug,current,next,worktree`. Keep `current` (it is the table's CURRENT header and the computed dispatch-current); do NOT alias it to `status`, because `status` is a real projectable frontmatter key and `current` is the computed value — collapsing them would be ambiguous. The spike confirmed the computed columns (`current`/`next`/`worktree`) are NOT projectable via `--fields` (frontmatter projection renders them blank); so in JSON `--fields` on `--next` adds projected *frontmatter* keys after the fixed five, exactly as in the table. The fixed five are always present; `--fields` is additive over them on `--next` (it cannot remove a computed dispatch column without breaking the command's meaning).
- **`--archived`** is a modifier, not a command: it widens the entity set. JSON is identical in shape to whichever base command it modifies (`status` or `next`); the only difference is membership. No per-row `scope` key in default JSON — the table shows no scope column, so omitting it keeps strict round-trip; revisit only if a consumer needs it (it would arrive via `--fields scope` / `--all-fields`, not by default).
- **`--where`** is a filter modifier; same shape as the base command, fewer rows.
- **`--resolve REF`** → `{"command":"resolve","workflow":…,"scope":…,"slug":…,"id":…,"stored_id":…,"path":…}`. `formatResolveLine` today prints `id=<stored_id>` (the resolve line uses the stored form). JSON emits BOTH: `id` = the **display id** (uniform with every other command, which all emit the display id via `applyEffectiveIDs`) and `stored_id` = the full stored form (`e.storedID`, what the text resolve line carries). This makes cross-command round-trip hold (an `id` from `--next`/`status` matches `--resolve`'s `id`) while preserving the full stored value the resolve consumer needs. Single object, not an array (resolve yields exactly one match or errors).
- **`--boot`** → one object with a key per section. **`--boot` is non-deterministic by construction (spike-confirmed): `next_id` is timestamp-minted for sd-b32, `team_state` is mtime+HOME dependent, `pr_state` shells to live `gh`.** It is NOT in scope for byte-determinism/round-trip; its deterministic sections are parity-checked and its volatile sections asserted structurally (see AC-1). Shape:
  ```
  {"command":"boot",
   "mods":{"merge":["roborev"], …} | {},
   "id_style":"sd-b32","next_id":"…","min_prefix":"2",          // min_prefix key present only for sd-b32
   "orphans":[{"id":…,"slug":…,"worktree":…,"dir_exists":"yes","branch_exists":"no"}, …],
   "pr_state":{"status":"ok|none|gh not available","entries":[{"id":…,"slug":…,"pr":…,"state":"OPEN"}, …]},
   "dispatchable":[ … same element shape as --next … ],
   "team_state":{"present":"true","hint":"…"}}
  ```
  `mods` empty → `{}` (not the string "none"); `orphans` empty → `[]`; `pr_state.status` carries the "none"/"gh not available"/"ok" sentinel the text form encodes in a header line, with `entries` `[]` when not "ok". `min_prefix` is `"2"` (string) and present only for sd-b32 (matches the table, which only prints `MIN_PREFIX` then). `team_state.present` is the string `"true"`/`"false"`. This is the highest-value command for the FO (read on every boot) and the highest-risk for schema churn — it has the most internal structure.

Singleton read outputs that are already a single token stay token-shaped and are NOT wrapped in JSON by default: `--next-id` (one id), `--short-id` (one token), `--validate` (`VALID`). These are already maximally frugal and proxy-safe; wrapping them in JSON would add cost, not remove it. `--json` on these is accepted but emits `{"command":"next-id","id":"…"}` / `{"command":"validate","valid":"true"}` (string-valued, per the all-strings rule) only when explicitly requested. **Decision: `--json` is opt-in everywhere; default behavior is byte-identical to today.**

### `--fields` composition with `--json`

`--fields a,b,c` already resolves to an ordered `extras` slice via `resolveExtraFields`. In JSON mode the resolved field set IS the object key set.

**JSON mode: strict projection.** `--fields id,status` → `{"id":…,"status":…}` only, in named order. The agent-frugal intent is "request the minimal set"; strict projection delivers exactly that. `--all-fields --json` emits defaults + every non-empty non-underscore frontmatter key, sorted (the same set the table computes).

**Table mode: de-dupe, then additive (captain directive — fixes a today-latent bug).** The spike confirmed `--fields id,status` on the table emits id and status TWICE (the six defaults, then `id`/`status` re-appended as extras). The fix: when an explicit `--fields` name is already a default column, the table **de-dupes** it — the name is suppressed from the appended extras rather than rendered a second time. `--fields id,status` then shows the six default columns once (no duplicate ID/STATUS extras); `--fields title,foo` (where `foo` is a non-default frontmatter key) appends only `FOO` as an extra. This is the minimal change that removes the duplicate-column bug without changing the table's "defaults are always shown" contract or the captain-facing verbatim-forward path. The de-dupe lives in `resolveExtraFields` and filters appended extras against the columns the command **actually displays**, not the abstract `defaultFields` argument: the default table de-dupes against its rendered columns `{id, slug, status, title, score, source}`, while `--next` de-dupes against its rendered columns `{id, slug, current, next, worktree}` — NOT `defaultNextFields` `{id, slug, status}`. So `--fields id,status` collapses the default table's duplicate ID/STATUS, `--next --fields id` suppresses the duplicate ID extra (id is a displayed `--next` column), and `--next --fields status` STILL appends STATUS — `status` is a frontmatter key but not a displayed `--next` column (the value shows under the `current` header), so suppressing it would regress a column that appears today. The displayed-columns scope is what keeps the directive's "fix the duplicate-displayed-column bug" and "does not change which columns appear" both true.

The result is a clean divergence, both halves now well-defined: **table** = defaults always shown + de-duped extras appended; **JSON** = strict projection of exactly the named keys. The divergence is deliberate (the table is captain-facing and must keep its default columns; JSON is machine-facing and wants minimality) and is tested on both sides (AC-2).

### `--quiet` composition

- On **reads + `--json`**: no-op (JSON already chrome-free). Accepted, no effect, documented as such. Not an error to combine.
- On **reads + table**: suppress the header row and the `----` separator row, emitting data rows only — useful for `| while read` scripts. (Lower priority; the table is captain-facing. Could be deferred, but cheap.)
- On **`--set --quiet`**: replace the multi-line `field: old -> new` narration with a single machine line, e.g. `set slug=<slug> status=ideation->design` (one line, space-joined `field=old->new` pairs) OR `{"command":"set","slug":…,"changes":[{"field":…,"old":…,"new":…}]}` under `--set --json --quiet`. Exit code unchanged (0 on success, 1 on guard failure). Guard-failure stderr is unchanged — `--quiet` suppresses success narration, never error diagnostics.
- On **`--archive --quiet`**: single machine line `archived slug=<slug>` (or JSON `{"command":"archive","slug":…}`), exit code preserved.
- `--quiet` NEVER changes exit codes and NEVER suppresses stderr error output. It only trims success-path chrome on stdout. This invariant is the testable heart of AC-3.

### Acceptance criteria (rewritten as end-state properties)

Each is an observable-output property proven by exercise-and-observe over Go fixtures: the runner writes to `io.Writer` and returns an exit int, so a test runs the command and asserts bytes / exit code / on-disk state. Each AC names its behavioral oracle.

**AC-1 — `--json` is a glyph-free projection of the same data each read command renders; byte-deterministic and table-round-tripping for the env-independent read commands, structurally parity-checked for `--boot`.**
End state, two tiers (the spike forced this split):
- *Deterministic, full-value reads* — default table, `--next`, `--archived`, `--where`: `--json` emits one compact newline-terminated JSON document containing no padding-space runs and no box/ellipsis glyph (`…`); it is byte-identical across repeated runs over the same fixture; and every JSON value equals the corresponding **default (full-value) table column's** underlying field value. The round-trip parity compares JSON only against the table's default columns (which `padRight` leaves un-truncated); **truncated extra cells are explicitly carved out** of the parity assertion because `formatExtraCell` is lossy (20-rune `…`) while JSON is lossless — JSON emits the full value and parity for an extra field is checked against the frontmatter source value, not the rendered cell.
- *`--boot`* — non-deterministic by construction (spike: `next_id` minted from `time.Now()`, `team_state` from mtime/HOME, `pr_state` from live `gh`). NOT byte-compared. Asserted exactly as `internal/status/nextid_boot_test.go` already does for the text form: section keys present and in order; the deterministic section bodies (`id_style`, `dispatchable`, the static parts) parity-checked against a fixture; the volatile material (`next_id` value, `team_state`, `pr_state` entries) normalized/regex-asserted (24-char sd-b32 alphabet for `next_id`; `team_state.present` ∈ {`"true"`,`"false"`}; `pr_state.status` ∈ {`"ok"`,`"none"`,`"gh not available"`}). `--resolve` is deterministic and gets a byte-stable golden of its single object.
Behavioral oracle: `go test` running each command's `--json` against a `testdata` fixture — (a) byte-equal golden per deterministic command, (b) run-twice-and-diff for determinism, (c) `bytes.Contains` negative assertions for a two-space run and for `"…"`, (d) parse-JSON-vs-parse-default-table-columns parity walk, (e) `--boot` structural+normalized parity mirroring `nextid_boot_test.go`.

**AC-2 — `--fields` projects exactly the requested set: strict in JSON, de-duped-additive in the table (the de-dupe fixes the observed duplicate-column bug).**
End state: `--json --fields id,status` emits objects whose key set is exactly `{id,status}` (named order), nothing else; `--all-fields --json` emits defaults + sorted non-empty non-underscore frontmatter keys. In the table, `--fields id,status` emits the six default columns ONCE — no duplicate ID/STATUS extras (regression-fixing the spike-observed bug where id/status appear twice); `--fields <non-default-key>` appends that one extra column.
Behavioral oracle: `go test` — (a) `--json --fields id,status` parsed, assert `sorted(keys) == [id,status]`; (b) a table test that runs `--fields id,status` and asserts the header line contains exactly one `ID` and one `STATUS` token (this fails today and passes after the de-dupe — it is the bug's reproduction-and-fix oracle); (c) `--all-fields --json` over a fixture carrying a non-default field, assert the extra key is present.

**AC-3 — `--quiet` trims success-path stdout to a single machine line on mutations, never altering exit codes or stderr diagnostics.**
End state: `--set <slug> status=X --quiet` on success emits one stdout line + exit 0; the same `--set` that trips a guard (mod-block / merge-hook / terminal) emits the unchanged stderr error + exit 1 regardless of `--quiet`; `--archive <slug> --quiet` emits one stdout line + exit 0. `--quiet --json` on reads is a no-op (bytes identical to `--json`).
Behavioral oracle: `go test` over a temp-dir fixture (the existing `mutation_test.go` pattern) — (a) success `--set --quiet` stdout is exactly the one expected machine line and exit==0; (b) a guard-tripping `--set --quiet` produces stderr and exit==1 byte-identical to the same call without `--quiet` (diff against the existing guard test's captured stderr); (c) `--archive --quiet` one line + exit==0; (d) `--quiet --json` read bytes == `--json` read bytes.

**AC-4 — `--json` is proxy-survivable, and the FO runtime's own scheduling reads are switched to `--json` (provably, with a positive contract assertion that detects both the switch and a revert).**
End state: `--json` output is a single compact line-delimited document with no aligned-column padding, no separator rule, no ellipsis — the structural properties that survive a token proxy (rtk) intact. And `skills/first-officer/references/claude-first-officer-runtime.md` (NOT shared-core — the spike/grep confirms the `## Event Loop` reads live in the runtime file: `status --where "pr !="`, `status --where "mod-block !="`, `status --next`, plus the startup `status --boot`) is edited so those FO-internal scheduling reads use `--json`; the captain-facing verbatim-forward table path stays unchanged.
Behavioral oracle (two parts, both real):
- proxy-survivability is the SAME observable property as AC-1's negative-glyph + determinism assertions (no aligned padding, no glyph, stable bytes IS what survives rtk's collapse/summarize failure modes) — no separate behavioral surface needed, it reuses AC-1's oracle.
- the contract switch gets a **positive** skill-text assertion that replaces the weak presence check: in `skills/integration/skill_text_test.go`, scope to the `## Event Loop` section of `claude-first-officer-runtime.md` via the existing `sectionAfter` helper, and assert each scheduling-read line in that section contains `--json` (e.g. the line that issues `status --next` also contains `--json`; same for the two `--where` reads). Because the assertion is "the read line CONTAINS `--json`," it fails both if the switch was never made AND if a later edit reverts the line to bare `status --next` — which the old superstring/`Contains("--json")`-anywhere check could not detect. (`claude-first-officer-runtime.md` is already in the test's `vendoredSkillFiles` list.)

### FO-contract update (the AC-4 doc deliverable, scoped now)

Two distinct FO read paths, treated differently. The edit targets `skills/first-officer/references/claude-first-officer-runtime.md` for the scheduling reads (where `## Event Loop` lives) — the audit corrected my earlier mis-targeting of shared-core.

1. **FO-internal scheduling reads** — the `## Event Loop` lines in the runtime file (`status --where "pr !="`, `status --where "mod-block !="`, `status --next`) and the startup `status --boot` read: switch to `--json` (+ `--fields` for the minimal set the FO needs, e.g. `status --next --json --fields id,slug`). The FO parses JSON instead of scraping padded columns — robust against rtk and against column-width drift. Note `--fields` on `--next` is additive over the fixed five computed columns (spike: computed columns aren't projectable), so the FO gets `id,slug,current,next,worktree` plus any named frontmatter keys.
2. **Captain-facing state display** (`### Captain-Facing State Display`, in shared-core): unchanged. The FO still forwards the human table verbatim inside a fenced block. JSON is for the machine reader, the table is for the human.

The contract edit names which reads use which mode and why, and pins the `--boot --json` section keys (`mods`, `id_style`, `next_id`, `min_prefix`, `orphans`, `pr_state`, `dispatchable`, `team_state`) so the FO's parsing instructions and the JSON schema stay in lockstep. The positive section-scoped skill-text assertion (AC-4) keeps the edit from silently regressing.

### Staff review recommendation

**Yes — this ideation warrants an independent staff review before design/implementation, and one such adversarial audit (Cycle 1) has now run and resolved the open forks.** The load-bearing risk is exactly the FLYWHEEL one: JSON schema stability across the read commands. Once the FO (and any ensign tooling) parses these shapes, the keys become a contract; a later rename or restructuring is a breaking change to the heaviest consumer in the system.

The Cycle-1 audit settled the decisions that earlier sat open: all-strings typing (no booleans/numbers), `--resolve` emits both `id` and `stored_id`, `--next` keeps `current` (not aliased to `status`), `--boot` is structurally (not byte-) asserted, the `--fields` table de-dupe, and the FO-contract target file. With those resolved and grounded in the spike, the residual review surface is small: the `--boot` nested-object schema (still the most structure / most future churn) and the strict-vs-de-duped-additive `--fields` divergence (now well-defined, but it is a deliberate behavioral split worth a second reviewer confirming). A focused schema-stability review at design-gate is the right size — review the JSON shapes and the divergence, not the Go internals (a thin serialization layer over data the table already produces).

### Out of scope (YAGNI guardrails)

- No new data the table cannot already show — `--json` is serialization only.
- No JSON Schema file / versioned schema endpoint — the golden tests + pinned FO contract ARE the schema for v0.
- No streaming/NDJSON-per-entity — one document per invocation; entity arrays are small (workflow-sized).
- No change to table-mode DEFAULT columns or the captain-facing verbatim-forward path. (The `--fields` table de-dupe is in scope — it is a bug fix removing duplicate columns when an explicit field names a default; it does not change which default columns appear.)
- No rtk profile shipped — the design goal is "survives a dumb proxy," not "configures a specific one." A separate entity can add an rtk profile later if the structured output proves insufficient.

## Stage Report: ideation

- DONE: Decide the --json schema/shape per read command (default table, --next, --archived, --where, --resolve, --boot) — stable, byte-deterministic, glyph/padding-free — and how --fields projection and --quiet compose with it (incl. --quiet on --set/--archive mutations: machine line vs suppressed chrome, exit codes preserved).
  `## Ideation` → "`--json` shape per read command", "`--fields` composition", "`--quiet` composition"; per-command envelopes pinned, determinism rules (fixed key order, compact, SetEscapeHTML(false)), --quiet invariant: never alters exit code/stderr.
- DONE: Rewrite AC-1..AC-4 as end-state properties each with a reproducible test-how (Go unit/golden over fixtures, smallest proof surface), and include the FO-contract update to prefer --json/--fields for its own reads.
  `## Ideation` → "Acceptance criteria (rewritten as end-state properties)" (AC-1 round-trip+determinism, AC-2 projection, AC-3 quiet-mutation, AC-4 proxy-survivability) and "FO-contract update" splitting FO-internal reads (→ --json) from captain-facing display (→ unchanged table); pins skills/first-officer/references/first-officer-shared-core.md as the doc target, verified via skills/integration/skill_text_test.go.
- DONE: State whether this ideation warrants an independent staff review and why (JSON schema stability across the read commands is the main design risk).
  `## Ideation` → "Staff review recommendation": YES, focused schema-stability review; load-bearing decisions (a) --fields strict-in-JSON vs additive-in-table, (b) --boot nested schema, (c) singleton reads bare vs opt-in JSON, (d) per-row scope omission.

### Summary

Designed agent-facing output modes as three orthogonal, independently-testable knobs — `--json` (serialization), `--fields` (projection), `--quiet` (chrome suppression) — holding the line that `--json` is a byte-deterministic projection of the SAME data the table renders, not a new data model (round-trip is the correctness anchor). Pinned a per-command JSON envelope set (string-valued, fixed key order, compact, no glyphs), resolved the one real semantic fork (`--fields` strict in JSON vs additive in table) explicitly, and rewrote AC-1..AC-4 as observable-output properties testable via Go goldens over existing status fixtures with no live process. Recommended an independent focused staff review because JSON schema keys become a hard contract with the FO (the heaviest reader) the moment it parses them, and the `--boot` nested schema + the strict/additive fork are cheap to settle now and breaking to change later.

Note: the `### Fetch commands` stage-def fetch (`claude-team show-stage-def …`) could not run — `claude-team` is not on PATH in this environment (the vendored copy at `skills/commission/bin/claude-team` is not exported). The inlined dispatch checklist was self-contained, so ideation proceeded against it; flagging the missing binary as a dispatch-environment note, not a blocker.

## Stage Report: ideation (cycle 2)

Revision after Cycle-1 gate REJECT (8 confirmed findings). SPIKE run first: built `/tmp/sd-aom` and exercised `--boot` (×2) and `--fields id,status` on `docs/dev` + `internal/status/testdata/seq-workflow`; findings drive the revision and are recorded in `## Ideation` → "Spike findings". All 8 directives addressed.

- DONE: SPIKE — observe `--boot` non-determinism and `--fields` duplicate columns directly, revise AC-1 scope + table fix from observed output.
  Spike confirmed: `--boot` NEXT_ID differs run-to-run (`tw35…` vs `a2an…`), TEAM_STATE mtime/HOME-dependent, PR_STATE shells to gh; `--fields id,status` emits ID/STATUS twice. "Spike findings" subsection + AC-1 two-tier rewrite + AC-2 de-dupe.
- DONE: Table `--fields` de-dupe (captain) — suppress already-default names; JSON stays strict-projection.
  `## Ideation` → "`--fields` composition": de-dupe in `resolveExtraFields`; AC-2 oracle (b) reproduces-and-fixes the duplicate-column bug. Out-of-scope guardrail reconciled (de-dupe is a bug fix, not a defaults change).
- DONE: JSON value typing (FO) — all strings; stringify `min_prefix`/`validate.valid`/`team_state.present`.
  `## Ideation` → "`--json` shape": all-strings rule, no booleans/numbers; per-command shapes updated (`min_prefix":"2"`, `team_state.present":"true"`, `valid":"true"`).
- DONE: `--resolve` emits both `id` (display) and `stored_id` (full stored).
  `## Ideation` → `--resolve` shape: `{…,"id":<display>,"stored_id":<storedID>,…}` for cross-command round-trip.
- DONE: `--boot` determinism scoped to env-independent sections; volatile sections asserted structurally, mirroring `nextid_boot_test.go`.
  AC-1 split into two tiers; `--boot` tier mirrors the existing structural+`<ID>`-normalized parity pattern (read at `internal/status/nextid_boot_test.go`).
- DONE: AC-4 real positive assertion that FO-internal read lines CONTAIN `--json`; target `claude-first-officer-runtime.md`.
  AC-4 + "FO-contract update": section-scoped (`sectionAfter("## Event Loop")`) assertion on each scheduling-read line in `claude-first-officer-runtime.md` (file already in `vendoredSkillFiles`); detects both switch and revert.
- DONE: Pin `--next` field vocabulary (`status` vs `current`; computed-column projectability).
  Spike: computed `current`/`next`/`worktree` are NOT projectable (render blank); `status` is a real frontmatter key. Keep `current` (not aliased to `status`); `--fields` on `--next` is additive over the fixed five.
- DONE: Resolve extra-cell truncation vs round-trip parity.
  Spike: default columns full-value (`padRight` min-width), extras lossy (20-rune `…`). Decision: JSON emits full value; round-trip parity compares JSON to default columns only; truncated extras carved out (parity for an extra field checks the frontmatter source, not the rendered cell). Recorded in framing + AC-1.

### Summary

Re-grounded the design in observed behavior via the mandated spike, then resolved all 8 audit findings: every JSON value is a string (no booleans/numbers), `--resolve` carries both display `id` and `stored_id`, `--boot` is two-tier (structural+normalized, not byte-deterministic — confirmed non-deterministic by running it twice), the `--next` computed columns stay as the fixed five (`current` not aliased; computed columns proven non-projectable), and the table `--fields` de-dupe fixes the spike-observed duplicate-column bug. Every AC now names a behavioral exercise-and-observe oracle; the lone docs deliverable (FO-contract edit, correctly targeting `claude-first-officer-runtime.md`) gets a section-scoped positive assertion that catches both the switch and a revert. Round-trip parity is scoped to full-value default columns with truncated extras explicitly carved out.

## Feedback Cycles

### Cycle 1 — ideation gate REJECT (staff audit, 2026-05-30)

Three-lens adversarial staff audit returned material-concerns (8 confirmed against code); rejected to ideation. The core thesis holds; this is reject-to-fix. Captain- and FO-resolved directives:

- **Table `--fields` de-dupe (captain).** `--fields` naming an already-default column de-dupes (suppresses the duplicate) — this fixes a today-latent bug (`--fields id,status` currently emits 8 columns, id/status twice). JSON stays strict-projection.
- **JSON value typing (FO).** All values are strings — stringify `min_prefix`, `validate.valid`, and `team_state.present` — for a uniform, byte-stable parser contract (resolves the rule-vs-examples contradiction).
- **`--resolve` id (FO).** Emit both `id` (the display id, uniform with every other command) and `stored_id` (the full stored form), so cross-command round-trip holds.
- **`--boot` determinism (FO).** AC-1 byte-determinism/round-trip is scoped to the env-independent sections; the volatile sections (`next_id` for sd-b32, `team_state`, `pr_state`) are asserted structurally, mirroring `nextid_boot_test.go`. `--boot` is non-deterministic by construction (time.Now vs mtime, timestamp-minted next_id, live gh) and has no single table to round-trip against.

Revision must run a **SPIKE**: run `spacedock status --boot` and `spacedock status --fields id,status` on a fixture, observe the non-determinism and the duplicate columns directly, and revise AC-1's scope + the table fix from observed output. Also: AC-4 needs a real positive assertion that the FO-internal read lines contain `--json` (the current superstring/presence test cannot detect the switch or a revert); the FO-contract edit targets `claude-first-officer-runtime.md` (the `## Event Loop` reads live there, not in shared-core); pin the `--next` field vocabulary (`status` vs `current`, and whether computed columns are projectable); resolve extra-cell truncation vs the round-trip parity rule.

### Cycle 2 — validation gate REJECT (validator + parallel staff audit, 2026-05-30)

Both the validator and an independent staff audit (detached read-only checkout) reject — NARROW. All four ACs' behavior is correct and reproduced; all gates green (267 pass); default output byte-identical to the oracle. Two MATERIAL defects, both test/schema quality, NOT behavior:

- **AC-4 test under-asserts (both reviewers; reproduced by running).** `TestEventLoopReadsUseJSON` uses `strings.Contains` over the whole `## Event Loop` section, but `status --next --json` appears TWICE (step-3 dispatch + step-4 idle re-run). Reverting ONLY one to bare `status --next` still passes — breaking AC-4's own revert-detection promise. Fix: assert per-read-line or count occurrences (each scheduling read contains `--json`; zero bare reads).
- **AC-1 `--validate --json` false-path missing + untested (audit only — validator missed it).** On an INVALID workflow, `--validate --json` never emits the design-promised `{"command":"validate","valid":"false"}` — `handlers.go:211-219` returns 1 after stderr errors BEFORE the `if-asJSON` block; no test covers `--validate`+`--json`. Mitigant: `--validate` is a singleton OUTSIDE the FO hot path (the FO uses text `status --validate`), so it is a schema-vs-impl gap on an untested branch, not a default-behavior regression. Fix: emit the false document under `if-asJSON` before the early return + test both `--validate --json` branches.

Polish (fold in cheaply): `seq-archived.json` golden has no real archived row — add one archived entity to the seq-workflow fixture; the round-trip parity walk matches JSON values against the whole table row — column-scoped matching is stronger.

Routed to implementation (fresh dispatch — the prior impl ensign is over context budget). No behavior change required.

## Stage Report: implementation

- DONE: Implement --json (per-command envelopes per the ## Ideation design: status/next/archived/where/resolve/boot; ALL values JSON strings; fixed key order; compact + newline-terminated; SetEscapeHTML(false); --json opt-in, default output byte-identical to today), --fields (strict projection in JSON; TABLE DE-DUPE bug fix in resolveExtraFields so an explicit field naming a default no longer duplicates the column), and --quiet (single machine line on --set/--archive success; NEVER alters exit code or stderr).
  internal/status/json.go (ordered string-only serializer, SetEscapeHTML(false)), json_commands.go (status/next/archived/where/resolve/boot/next-id/short-id/validate envelopes + resolveJSONFields strict), boot.go (gatherBoot extracted so text+JSON share one source), native_runner.go/handlers.go/mutate.go (--json/--quiet wiring); de-dupe scoped to DISPLAYED columns (commit 9b23f30) — captain ruling B (held for confirmation, implemented per the directive's own "removes duplicate columns" wording, which reading A contradicts). Verified live: default --json byte-identical default path; sdb32 --resolve --json id="je" + stored_id full; --boot --json mods:{}/min_prefix:"2"/team_state.present:"false".
- DONE: Write the behavioral oracle PER AC exactly as the design's 'Behavioral oracle' lines specify: AC-1 two-tier (byte-golden + round-trip-vs-default-columns + run-twice determinism + negative-glyph for the deterministic reads; --boot structural+normalized mirroring nextid_boot_test.go), AC-2 (strict-keys + the de-dupe reproduction-and-fix table test that fails today and passes after), AC-3 (--quiet success-line + guard-stderr-byte-identical-with/without-quiet + --quiet --json read no-op), AC-4 (proxy-survivability reuses AC-1 assertions + the section-scoped POSITIVE skill_text assertion on claude-first-officer-runtime.md's ## Event Loop reads). Edit the VENDORED skills/first-officer/references/claude-first-officer-runtime.md to switch the ## Event Loop scheduling reads + startup --boot to --json (+ --fields minimal set); keep the captain-facing table path verbatim.
  json_read_test.go (golden+determinism+glyph-free+round-trip-vs-table-columns), json_boot_test.go (key-order+range-checked volatile+dispatchable mirrors --next), json_quiet_test.go (success line + guard byte-identical + --quiet --json no-op + --set --json envelope), fields_dedupe_test.go (reproduces ID/STATUS-twice, passes after; --next --fields status unchanged; non-default still appends), skill_text_test.go TestEventLoopReadsUseJSON (section-scoped positive, fails on switch-absent AND on revert). FO runtime ## Event Loop + startup --boot switched to --json --fields; mod-block clear --set left as text (guarded against over-broad sweep). JSON tests use NativeRunner directly (the Python oracle has no --json to compare against).
- DONE: go test ./... and go test ./... -race, gofmt -l, go vet — all clean with REAL captured exit codes (capture $?; never pipe to tail).
  With SPACEDOCK_ORACLE set: `go test ./...` exit=0 (266 passed, 0 skipped); `go test ./... -race` exit=0 (266 passed); `gofmt -l .` exit=0 (empty); `go vet ./...` exit=0. Exit codes captured via `> log 2>&1; echo $?`, not piped.

### Summary

Landed three orthogonal output knobs as additive code within internal/status (new json.go/json_commands.go + format/handlers/mutate/native_runner/boot edits) plus the vendored FO-runtime contract switch — no shared-parser restructuring, so the later native-dispatch-helper merge stays clean. `--json` is a string-only, fixed-key-order, compact, HTML-unescaped projection of the same data the table renders; default (non-`--json`) output is byte-identical to today, confirmed by the unchanged oracle-parity suite. The `--fields` table de-dupe is scoped to the columns the table actually DISPLAYS (so `--next --fields status` is unchanged and the default-table ID/STATUS-twice bug is fixed); this makes native intentionally diverge from the buggy Python oracle for default-named `--fields`, so the two affected oracle-parity cases were retargeted to non-default fields with the de-dupe behavior locked by dedicated native tests. Two commits on spacedock-ensign/agent-output-modes: 4f55c7d (--json/--quiet/AC-4), 9b23f30 (de-dupe). One open item flagged to team-lead: the AC-2 de-dupe scope was an A-vs-B fork I escalated; I implemented reading B (de-dupe only against DISPLAYED columns) because reading A contradicts the captain directive's own "removes duplicate columns" wording and would change which columns `--next` shows — confirm B or override.

## Stage Report: validation

RECOMMENDATION: REJECTED (narrow — one AC-4 test-strength gap; all behavior correct).

- DONE: Independently reproduce ALL FOUR ACs' behavioral oracles (built own fixtures + live binary `/tmp/sd-aom-val`, default NativeRunner confirmed at internal/cli/cli.go:21).
  AC-1: reproduced — `--json` glyph-free (0 double-spaces, 0 `…` via Python `.count()` + `od -c`), byte-deterministic (run-twice-diff identical for default/next/archived/where/resolve), all-string leaves on every command (Python type-walk), `--resolve` emits id="ab" + stored_id="abc…23" on sd-b32 (display vs stored differ), `--boot` key order command,mods,id_style,next_id,min_prefix,orphans,pr_state,dispatchable,team_state with min_prefix="2"/mods:{}/team_state.present="true", HTML `< > &` survive literally (SetEscapeHTML(false) holds), round-trip-vs-default-columns parity holds. AC-2: reproduced — `--fields id,status` table shows ID/STATUS once, `--next --fields status` STILL appends STATUS, `--next --fields id` suppressed, JSON `--fields id,status` keys exactly {id,status}. AC-3: reproduced — `--set --quiet` one line+exit0, guard-tripping `--set` stderr+exit BYTE-IDENTICAL with/without `--quiet` (diff empty, both exit 1), `--archive --quiet` one line+exit0, `--quiet --json` read bytes==`--json`. AC-4: switch present on all four reads; over-broad-sweep guard holds (mod-block clear stays bare); proxy-survivability == AC-1 oracle (single newline-terminated line).
- DONE: Run go test ./... and go test ./... -race, gofmt -l, go vet with REAL captured exit codes; confirm DEFAULT output byte-identical to pre-change.
  `go test ./...` exit=0 (267 passed, 0 SKIP — oracle present at default path so parity tests ran), `-race` exit=0 (267 passed), `gofmt -l .` exit=0 (empty), `go vet ./...` exit=0 (no issues). Default table byte-identical to oracle: plain `status`, `--next`, `--archived`, `--all-fields` all BYTE-IDENTICAL; `--fields issue` (non-default) byte-identical; the only oracle divergence is the `--fields id,status` duplicate-column removal (diff confirms ONLY the trailing duplicate ID/STATUS extras dropped).
- FAILED: Verify the AC-4 contract test detects a revert of the FO-internal read line to bare `status --next`.
  AC-4's "Verified by" oracle promises the assertion "fails ... if a later edit reverts the line to bare `status --next`." Reproduced 4 revert scenarios on a scratch copy of claude-first-officer-runtime.md: revert `--where "pr !="` → test FAILS (caught); revert `--where "mod-block !="` → FAILS (caught); revert BOTH `--next` reads → FAILS (caught); revert ONLY step-3 `--next` (step-4 re-run still `--json`) → test PASSES (NOT caught). Root cause: `status --next --json` appears TWICE in the `## Event Loop` (step 3 dispatch + step 4 idle-hook re-run, both legitimate) and TestEventLoopReadsUseJSON uses `strings.Contains` not a count, so a partial single-line revert satisfies the substring via the other occurrence. The AC-2 de-dupe test already uses a count-based `countToken` assertion — the same pattern would close this gap (assert two `status --next --json` reads, or assert zero bare `status --next` tokens in the section).

### Summary

All four ACs' BEHAVIOR is independently reproduced and correct, all four gates are green with captured exit codes (267 passed, 0 skipped), and default (non-`--json`/`--fields`/`--quiet`) output is byte-identical to the oracle — the opt-in claim holds and the de-dupe divergence is exactly the duplicate-column removal. The JSON contract the FO now reads is sound: per-command envelopes match the design, all values are strings, `--resolve` carries both display id and stored_id, and `--boot` keys are pinned. The single defect is in AC-4's verification, not its deliverable: the FO runtime genuinely switched all four scheduling reads to `--json`, but the cited revert-detecting test misses a partial revert of one of the two `status --next` reads (Contains vs count). Rejecting narrowly so the test is tightened to honor the AC-4 oracle's stated promise — a one-line fix mirroring the AC-2 count-based assertion. No behavior change is required.

## Stage Report: implementation (cycle 2)

Narrow re-dispatch after validation REJECT — two material test/schema defects + two polish items, NO behavior change. Each fix was proven by reproducing the defect first, then fixing (TDD on the regressions). Code commit: 30d4562 on spacedock-ensign/agent-output-modes (tip eb1e005).

- DONE: Fix AC-4's test (skills/integration/skill_text_test.go TestEventLoopReadsUseJSON): replace the whole-section strings.Contains with a per-read-line or count-based assertion so reverting ANY single Event-Loop scheduling read to its bare form fails the test. Confirm by reverting ONLY the step-3 --next dispatch read in a scratch check — the test must now FAIL (it currently passes because step-4's duplicate --next satisfies Contains).
  Rewrote as a per-line walk over the `## Event Loop` section: any line issuing a scheduling-read base (`status --next`, `status --where`) must also carry `--json`; plus a guard that `status --next` appears >=2 times (step 3 + step 4). Behavioral proof: reverted step-3 --next only → FAIL (was PASS before); reverted each --where read independently → FAIL; correct file → PASS. Defect reproduced before fix (step-3-only revert passed the old test).
- DONE: Fix AC-1's --validate --json false-path (internal/status/handlers.go ~211-219): emit {"command":"validate","valid":"false"} under the if-asJSON branch BEFORE the early return on an invalid workflow (today it returns 1 after stderr errors, never producing the JSON). Add a test exercising BOTH --validate --json branches (valid -> valid:"true", invalid -> valid:"false"); today neither is covered.
  3-line fix in handlers.go runRead: emit the false envelope under `if asJSON` before `return 1`. New json_validate_test.go drives both branches via runNative (oracle has no --json): valid→`valid:"true"`+exit0+empty stderr; invalid→`valid:"false"`+exit1+stderr preserved (envelope additive, never silences diagnostics or alters exit). Defect reproduced before fix (invalid path emitted empty stdout — JSON parse failed). Live-confirmed shape.
- DONE: Fold in the two polish items: add one archived entity to the seq-workflow fixture so seq-archived.json exercises a real archived row; make the round-trip parity walk match each JSON value against its specific column, not the whole row. Then run go test ./... and -race, gofmt -l, go vet with REAL captured exit codes.
  Added _archive/005-archived-cleanup.md (status done, id 005) — fixture still VALID; seq-archived.json regenerated via -update and now genuinely differs from seq-default.json (005 present only under --archived). Round-trip walk replaced with a column-scoped left-to-right cursor (assertRowColumnsMatchJSON) pinning each value to its column boundary; scratch-proved it rejects a value placed in the wrong column (status="roadmap") that whole-row Contains would have passed. Gates: `go test ./...` exit=0 (270 passed, 0 skipped), `-race` exit=0 (270), `gofmt -l .` exit=0 (empty), `go vet ./...` exit=0.

### Summary

Three surgical validation-cycle fixes with zero behavior change, each proven by reproducing the defect before fixing. AC-4's revert-detection promise is now honored: the Event-Loop test walks per-line and requires `--json` on every scheduling read (the two duplicate `status --next` reads no longer let a single-line revert escape via Contains). AC-1's `--validate --json` now emits the `valid:"false"` envelope on the invalid path under the if-asJSON branch before the exit-1 return — both branches tested, stderr/exit semantics unchanged. The two polish items are folded in: the seq-workflow fixture has a real archived entity so `seq-archived.json` no longer mirrors the default golden, and the round-trip parity walk is column-scoped (each value pinned to its column, not matched anywhere in the row). The working `--json`/`--fields`/`--quiet` code was untouched; all four gates are green with captured exit codes (270 passed, 0 skipped; race/gofmt/vet clean).

## Stage Report: validation (cycle 2)

RECOMMENDATION: PASSED. Both cycle-2 fixes (commit 30d4562, branch tip) are REAL and load-bearing — each defect reproduced-before-fix and the fix proven to close it. All four ACs hold, default output byte-identical to the oracle, both polish items confirmed, all four gates green with captured exit codes (270 pass). Fresh independent re-validation; built my own binary and scratch fixtures, reverted each fix in isolation to prove the regression.

- DONE: Confirm the two cycle-2 fixes (commit 30d4562) are REAL and load-bearing, reproduce each: (AC-4) revert ONLY one Event-Loop scheduling read to its bare form in a scratch copy and confirm TestEventLoopReadsUseJSON now FAILS, then restore; (AC-1) run `status --validate --json` on an INVALID workflow and confirm it emits {"command":"validate","valid":"false"} + exit 1 + stderr preserved, and on a VALID workflow emits valid:"true" + exit 0.
  AC-4: reverted ONLY step-3 `status --next --json` (line 233 of claude-first-officer-runtime.md) to bare `status --next`, step-4 left as `--json` → NEW per-line-walk test FAILS (`skill_text_test.go:206: ...missing --json on line: "3. **Run status --ne..."`, exit 1); reconstructed the pre-fix whole-section `strings.Contains` assertion against the same scratch file → PASSES (the exact partial-revert-escapes defect cycle-2 named). Restored via git checkout, test green again. AC-1: live INVALID workflow (entity missing required id, seq style) → stdout `{"command":"validate","valid":"false"}\n` (39 bytes) + exit 1 + stderr `Error: missing required id...` (139 bytes preserved); live VALID workflow → `{"command":"validate","valid":"true"}\n` (38 bytes) + exit 0 + empty stderr. Reverted the 3-line handlers.go fix (emit-false-under-if-asJSON before return 1) and rebuilt → invalid path emits 0 bytes stdout, TestValidateJSONBranches/invalid FAILS (`unexpected end of JSON input, stdout=""`); restored, test green.
- DONE: Re-confirm the four ACs still hold and DEFAULT output (no --json/--fields/--quiet) is byte-identical to the oracle; spot-check the two polish items.
  Four ACs green: 11 AC-covering top-level tests pass (TestJSONReadGolden, TestJSONStatusRoundTripsTableColumns, TestValidateJSONBranches, TestGoldenRead, TestFieldsDedupe*, TestQuiet*, TestEventLoopReadsUseJSON), 0 fail/skip. Default byte-identity: live native vs oracle diff EMPTY for `status`/`--next`/`--archived`/`--all-fields` with matching exit codes; 92 parity/golden tests ran (oracle present at default path, 0 skipped). Polish 1: `_archive/005-archived-cleanup.md` exists (id 005, done); fixture VALID; `seq-archived.json` = `seq-default.json` + exactly one extra archived row (005); live `--json --archived`/`--json` match their goldens. Polish 2: reproduced the column-scoped walk via a standalone harness over a real fixture row — `status="roadmap"` (a value belonging to the source column) is REJECTED by the cursor walk (`not at its column position (col 2)`) while the old whole-row Contains ACCEPTS it.
- DONE: Run go test ./... and -race, gofmt -l, go vet with REAL captured exit codes (expect 270 pass).
  `go test ./...` exit=0 (test-level 270 pass / 0 fail / 0 skip via -json action count; the lone package-level skip is cmd/spacedock [no test files], not a test); `go test ./... -race` exit=0 (270 pass, 0 DATA RACE); `gofmt -l .` exit=0 (empty); `go vet ./...` exit=0 (no issues). Exit codes captured directly (`> log 2>&1; echo $?`), not piped to a summarizer.

### Summary

Fresh independent re-validation of the NARROW cycle-2 fixes: both are REAL and load-bearing, each proven by reverting the fix in isolation and watching the regression reappear. AC-4 — reverting only one of the two duplicate `status --next --json` reads makes the new per-line walk FAIL while the old whole-section Contains would have PASSED (the precise gap cycle-2 closed). AC-1 — the invalid-workflow `--validate --json` path now emits `valid:"false"` + exit 1 + preserved stderr; reverting the 3-line handlers.go change drops stdout to zero bytes and the regression test catches it. All four ACs hold, default (opt-out) output is byte-identical to the Python oracle across the read commands, and both polish items check out (a real archived row distinguishes `seq-archived.json` from `seq-default.json`; the column-scoped parity walk rejects a wrong-column value the old whole-row Contains accepted). Four gates green with captured exit codes — 270 test-level passes, 0 fail, 0 skip; race/gofmt/vet clean. Recommend PASSED.
