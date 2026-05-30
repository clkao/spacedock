---
id: jsacmxpkcwp3vg2t60yjdc4y
title: Agent-facing output modes — JSON, field projection, quiet (token-frugal)
status: ideation
source: session 1 debrief — rtk-inspired
score: "0.38"
worktree:
started: 2026-05-30T19:17:39Z
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

### Framing and scope discipline

`--json` is a stable projection of the same data the table already renders, not a new data model. Every JSON field traces to a value the existing formatter prints: a frontmatter field (`id`, `slug`, `status`, `title`, `score`, `source`, plus any extra under `--fields`/`--all-fields`) or a computed read-time value the table already shows (`--next`'s `current`/`next`/`worktree`; `--boot`'s section probes). No new computation, no new fields the table cannot show. This keeps `--json` a serialization choice, observable purely as output bytes, and means the round-trip property (AC-1) is the design's correctness anchor: for a given workflow state, `--json` and the table must carry identical field values.

Three orthogonal output knobs, each independently testable:

- `--json`: serialization format (table text → JSON). Applies to read commands.
- `--fields a,b,c`: column/key projection. Already implemented for the table (`resolveExtraFields` → `extras`); `--json` reuses the exact same resolved field set, so projection composes for free in both modes.
- `--quiet`: chrome suppression. On reads it is a near-no-op for `--json` (JSON has no decorative chrome); its real job is on mutations (`--set`/`--archive`) and on the human table (drop header/separator rows).

These compose as a matrix: `--json --fields id,status` is "project then serialize"; `--set … --quiet` is "mutate, emit one machine line."

### `--json` shape per read command

Top-level shape is a small, fixed set of envelopes. Two principles: (1) collections are JSON arrays of flat string-valued objects keyed by the same field names the table uses (lowercased table headers); (2) every value is a JSON string — entity frontmatter is textual and the table never coerces types, so `score: "0.38"` stays a string and byte-stability is trivial to guarantee. No nulls, no numbers, no booleans except where a probe is inherently boolean (`--boot` team_state.present).

Determinism rules (shared by every command):
- Marshal with a fixed key order, not Go's `map` iteration. Emit objects as an explicit ordered key list (the resolved field order: defaults first in their canonical order, then `--fields` order, or sorted `--all-fields`), so byte output is reproducible run-to-run.
- One JSON document, newline-terminated, no indentation (compact). A token proxy cannot "summarize" a single compact line-delimited document the way it collapses padded tables.
- Array element order is the existing sort order (`sortDefault` / `sortNext`), already deterministic and stable.
- HTML escaping off (`SetEscapeHTML(false)`) so `&`, `<`, `>` in titles/sources survive as written, matching the table.

Per command:

- **default table** → `{"command":"status","entities":[{"id":…,"slug":…,"status":…,"title":…,"score":…,"source":…}, …]}`. Keys are the resolved field set (defaults `id,slug,status,title,source,score` order as the table, or projected). Array order = `sortDefault`.
- **`--next`** → `{"command":"next","dispatchable":[{"id":…,"slug":…,"current":…,"next":…,"worktree":…}, …]}`. The three computed columns (`current`=status, `next`, `worktree`=yes/no) come straight from `computeDispatchable`. `--fields` adds projected frontmatter keys after the fixed five, mirroring the table's extras.
- **`--archived`** is a modifier, not a command: it widens the entity set. JSON is identical in shape to whichever base command it modifies (`status` or `next`); the only difference is membership. No `archived` flag per row is needed for parity (the table doesn't show one) — but a per-row `scope` key is a reasonable cheap add IF and only if we also add a `scope` column to `--all-fields`; otherwise omit to preserve strict round-trip. **Decision: omit `scope` from default JSON to keep round-trip exact; revisit only if a consumer needs it.**
- **`--where`** is a filter modifier; same shape as the base command, fewer rows.
- **`--resolve REF`** → `{"command":"resolve","workflow":…,"scope":…,"slug":…,"id":…,"path":…}`. Direct projection of `formatResolveLine`'s five `k=v` pairs into an object. Single object, not an array (resolve yields exactly one match or errors).
- **`--boot`** → one object with a key per section, each section preserving its current structure:
  ```
  {"command":"boot",
   "mods":{"merge":["roborev"], …} | {},
   "id_style":"sd-b32","next_id":"…","min_prefix":2,            // min_prefix omitted unless sd-b32
   "orphans":[{"id":…,"slug":…,"worktree":…,"dir_exists":"yes","branch_exists":"no"}, …],
   "pr_state":{"status":"ok|none|gh not available","entries":[{"id":…,"slug":…,"pr":…,"state":"OPEN"}, …]},
   "dispatchable":[ … same element shape as --next … ],
   "team_state":{"present":true,"hint":"…"}}
  ```
  `mods` empty → `{}` (not the string "none"); `orphans` empty → `[]`; `pr_state.status` carries the "none"/"gh not available"/"ok" sentinel the text form encodes in a header line, with `entries` `[]` when not "ok". `min_prefix` is present only for sd-b32 (matches the table, which only prints `MIN_PREFIX` then). This is the highest-value command for the FO (read on every boot) and the highest-risk for schema churn — it has the most internal structure.

Singleton read outputs that are already a single token stay token-shaped and are NOT wrapped in JSON by default: `--next-id` (one id), `--short-id` (one token), `--validate` (`VALID`). These are already maximally frugal and proxy-safe; wrapping them in JSON would add cost, not remove it. `--json` on these is accepted but emits `{"command":"next-id","id":"…"}` / `{"command":"validate","valid":true}` only when explicitly requested, for callers that want uniform parsing. **Decision: `--json` is opt-in everywhere; default behavior is byte-identical to today.**

### `--fields` composition with `--json`

`--fields a,b,c` already resolves to an ordered `extras` slice via `resolveExtraFields`. In JSON mode the resolved field set IS the object key set. Open design question to settle in spec: does `--fields id,status` mean "only these keys" (strict projection, drops the other defaults) or "defaults plus these"? The table semantics are "defaults plus extras." For JSON the agent-frugal intent (request the minimal set) argues for strict projection — `--fields id,status` → `{"id":…,"status":…}` only. **Decision: in JSON mode, `--fields` is strict projection (only the named keys, in named order); in table mode it stays additive (unchanged), because changing the table is out of scope and would break the captain-facing verbatim-forward contract.** This divergence is deliberate and must be documented and tested. `--all-fields` in JSON emits defaults + every non-empty non-underscore frontmatter key, sorted (same set the table computes).

### `--quiet` composition

- On **reads + `--json`**: no-op (JSON already chrome-free). Accepted, no effect, documented as such. Not an error to combine.
- On **reads + table**: suppress the header row and the `----` separator row, emitting data rows only — useful for `| while read` scripts. (Lower priority; the table is captain-facing. Could be deferred, but cheap.)
- On **`--set --quiet`**: replace the multi-line `field: old -> new` narration with a single machine line, e.g. `set slug=<slug> status=ideation->design` (one line, space-joined `field=old->new` pairs) OR `{"command":"set","slug":…,"changes":[{"field":…,"old":…,"new":…}]}` under `--set --json --quiet`. Exit code unchanged (0 on success, 1 on guard failure). Guard-failure stderr is unchanged — `--quiet` suppresses success narration, never error diagnostics.
- On **`--archive --quiet`**: single machine line `archived slug=<slug>` (or JSON `{"command":"archive","slug":…}`), exit code preserved.
- `--quiet` NEVER changes exit codes and NEVER suppresses stderr error output. It only trims success-path chrome on stdout. This invariant is the testable heart of AC-3.

### Acceptance criteria (rewritten as end-state properties)

Each is an observable-output property with the smallest proof surface (Go unit/golden over fixtures, no live process needed — the runner writes to `io.Writer` and returns an exit int).

**AC-1 — `--json` is a byte-deterministic, glyph-free projection of the same data each read command renders, round-tripping every table field.**
End state: for the default table, `--next`, `--archived`, `--where`, `--resolve`, and `--boot`, `--json` emits one compact newline-terminated JSON document; for every entity/section, each JSON value equals the corresponding table cell's underlying field value; output contains no padding spaces, no box/ellipsis glyphs (`…`), and is identical across repeated runs over the same fixture.
Test-how: golden-file tests per command over a shared fixture workflow (reuse the existing status fixtures). For each command run both modes; assert (a) JSON golden matches byte-for-byte, (b) a parity check that walks the parsed JSON and the parsed table rows and asserts equal field values (this is the round-trip assertion, not a second golden), (c) `bytes.Contains` negative assertions for `"  "` runs and `"…"`. Run the same command twice and assert identical bytes (determinism).

**AC-2 — `--fields a,b,c` projects exactly the requested keys/columns; in JSON mode strictly (only named keys, named order), in table mode additively (unchanged).**
End state: `--json --fields id,status` emits objects with exactly `{"id":…,"status":…}` and no other keys; `--fields id,status` (table) still shows defaults + the named extra columns as today. `--all-fields --json` emits defaults + sorted non-empty extra frontmatter keys.
Test-how: golden tests asserting the JSON object key set equals the requested set (parse + compare sorted keys); a table test asserting the additive behavior is unchanged (regression guard); an `--all-fields --json` test over a fixture with a non-default field.

**AC-3 — `--quiet` trims success-path stdout chrome to a single machine line on mutations without altering exit codes or stderr diagnostics.**
End state: `--set <slug> status=X --quiet` on success emits one stdout line and exit 0; the same `--set` that trips a guard (mod-block, merge-hook, terminal) emits the unchanged stderr error and exit 1 regardless of `--quiet`; `--archive <slug> --quiet` emits one stdout line and exit 0. On reads, `--quiet --json` is a documented no-op (identical bytes to `--json`).
Test-how: mutation tests over a temp-dir fixture: assert (a) success stdout is exactly the one expected machine line + exit 0, (b) a guard-tripping `--set --quiet` produces the same stderr + exit 1 as without `--quiet` (diff against the existing guard test's expected stderr), (c) `--archive --quiet` one line + exit 0. A read test asserting `--quiet --json` bytes == `--json` bytes.

**AC-4 — `--json` output is proxy-survivable and the FO contract prefers `--json`/`--fields` for its own reads.**
End state: a documented note + test that `--json` is a single compact line-delimited document free of decorative formatting a token proxy (rtk) would collapse or summarize (no aligned columns, no separator rule, no ellipsis); and `skills/first-officer/references/first-officer-shared-core.md` is updated so the FO's event-loop and `--boot` reads consume `--json` (parsed, not text-scraped), while the captain-facing verbatim-forward path keeps the human table.
Test-how: the AC-1 negative-glyph/determinism assertions double as the proxy-survivability test (the property "no aligned padding, no glyphs, stable bytes" IS proxy-survivability for rtk's known failure modes). The FO-contract update is a docs change verified by a skill-text test (the repo already has `skills/integration/skill_text_test.go`) asserting the contract mentions `--json` consumption for FO-internal reads; no behavioral test needed for prose, but the test pins the contract so it cannot silently regress.

### FO-contract update (the AC-2/AC-4 doc deliverable, scoped now)

Two distinct FO read paths, treated differently:
1. **FO-internal scheduling reads** (`## Event Loop` `--next`/`--where`, and `--boot` at startup): switch to `--json` (+ `--fields` for the minimal set the FO needs, e.g. `--next --json --fields id,slug,status`). The FO parses JSON instead of scraping padded columns — robust against rtk and against column-width drift.
2. **Captain-facing state display** (`### Captain-Facing State Display`): unchanged. The FO still forwards the human table verbatim inside a fenced block. JSON is for the machine reader, the table is for the human.

The contract edit names which reads use which mode and why, and pins `--boot --json` section keys (`mods`, `id_style`, `next_id`, `orphans`, `pr_state`, `dispatchable`, `team_state`) so FO parsing code/instructions and the JSON schema stay in lockstep.

### Staff review recommendation

**Yes — this ideation warrants an independent staff review before design/implementation.** The load-bearing risk is exactly the one the FLYWHEEL note calls out: JSON schema stability across the six read commands plus the two singleton commands. Once the FO (and any ensign tooling) parses these shapes, the keys become a contract; a later rename (`current`→`status`, restructuring `pr_state`, deciding `--fields` strict-vs-additive) is a breaking change to the heaviest consumer in the system. The specific decisions that merit a second reviewer's sign-off: (a) the `--fields` strict-projection-in-JSON / additive-in-table divergence; (b) the `--boot` nested-object schema (most structure, most churn surface); (c) whether singleton reads stay bare or gain opt-in JSON; (d) the per-row `scope` omission for round-trip exactness. These are cheap to get right now and expensive to change after the FO depends on them. A focused schema-stability review (not a full architectural review) is the right size: review the JSON shapes and the strict/additive decision, not the Go internals (which are a thin serialization layer over data the table already produces).

### Out of scope (YAGNI guardrails)

- No new data the table cannot already show — `--json` is serialization only.
- No JSON Schema file / versioned schema endpoint — the golden tests + pinned FO contract ARE the schema for v0.
- No streaming/NDJSON-per-entity — one document per invocation; entity arrays are small (workflow-sized).
- No change to table-mode defaults or the captain-facing verbatim-forward path.
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

## Feedback Cycles

### Cycle 1 — ideation gate REJECT (staff audit, 2026-05-30)

Three-lens adversarial staff audit returned material-concerns (8 confirmed against code); rejected to ideation. The core thesis holds; this is reject-to-fix. Captain- and FO-resolved directives:

- **Table `--fields` de-dupe (captain).** `--fields` naming an already-default column de-dupes (suppresses the duplicate) — this fixes a today-latent bug (`--fields id,status` currently emits 8 columns, id/status twice). JSON stays strict-projection.
- **JSON value typing (FO).** All values are strings — stringify `min_prefix`, `validate.valid`, and `team_state.present` — for a uniform, byte-stable parser contract (resolves the rule-vs-examples contradiction).
- **`--resolve` id (FO).** Emit both `id` (the display id, uniform with every other command) and `stored_id` (the full stored form), so cross-command round-trip holds.
- **`--boot` determinism (FO).** AC-1 byte-determinism/round-trip is scoped to the env-independent sections; the volatile sections (`next_id` for sd-b32, `team_state`, `pr_state`) are asserted structurally, mirroring `nextid_boot_test.go`. `--boot` is non-deterministic by construction (time.Now vs mtime, timestamp-minted next_id, live gh) and has no single table to round-trip against.

Revision must run a **SPIKE**: run `spacedock status --boot` and `spacedock status --fields id,status` on a fixture, observe the non-determinism and the duplicate columns directly, and revise AC-1's scope + the table fix from observed output. Also: AC-4 needs a real positive assertion that the FO-internal read lines contain `--json` (the current superstring/presence test cannot detect the switch or a revert); the FO-contract edit targets `claude-first-officer-runtime.md` (the `## Event Loop` reads live there, not in shared-core); pin the `--next` field vocabulary (`status` vs `current`, and whether computed columns are projectable); resolve extra-cell truncation vs the round-trip parity rule.
