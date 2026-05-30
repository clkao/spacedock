---
id: zse4a3ds0x19gpdcjh7anhgs
title: Segregate Claude-specific runtime coupling (context-budget, team/standing) from the generic contract
status: backlog
source: sprint — FO/captain (native-dispatch-helper scope split)
score: "0.32"
worktree:
---

The `claude-team` helper carries a runtime-coupled surface that `native-dispatch-helper` deliberately scopes OUT of the native dispatch path: `context-budget` (reads Claude Code's `~/.claude/.../agent-*.jsonl` transcripts + team `config.json`), and the standing-teammate / team subcommands `spawn-standing`, `list-standing`, `show-standing` (emit Agent() specs, probe team membership, enumerate `_mods/`). These are Claude-Code-specific and not on the initial-dispatch critical path — but the FO still shells to `python3 + claude-team` for **every ensign-reuse decision** (`context-budget`), the feedback-rejection flow, and standing-teammate spawn. So a "Python-free self-hosted" handoff is, today, only Python-free for *initial dispatch*; the steady-state FO loop is not.

This entity owns that surface and the contract reorganization it implies. Two coupled questions:

1. **Subcommand disposition.** For each runtime-coupled `claude-team` subcommand (`context-budget`, `spawn-standing`, `list-standing`, `show-standing`, and `build`'s `_mods`-enumeration branch that decides whether to emit a `show-standing` fetch line), decide: reimplement native (in a Claude-Code-specific package, NOT the generic dispatch/status core), keep a thin shim, or keep Python — and justify against the cost (transcript-format coupling, per-host layout) vs the benefit (a truly Python-free FO loop).
2. **Claude-vs-generic contract split.** The operating contract is split today into a generic core (`first-officer-shared-core.md`) and host runtime adapters (`claude-first-officer-runtime.md`, `codex-first-officer-runtime.md`). Determine whether Claude-Code-specific assumptions (transcript-budget reads, team/standing semantics, on-disk team layout) have leaked into the generic core, and if so, relocate them into the Claude runtime adapter (or a Claude-specific module) so the generic core is host-neutral. The native binary's package boundaries should mirror this: generic dispatch/status logic in host-neutral packages, Claude-Code coupling behind a clear seam.

## Acceptance criteria

These are SEED criteria for ideation to sharpen. **Avoid the doc-only antipattern:** the deliverable is not a decision memo — every AC must be falsifiable by an oracle external to this entity (a test, a command's output/exit/state, or a parametrized invariant over real artifacts), per the workflow's behavioral-proof discipline. A pure "we decided X" with no shipped artifact does not belong here.

**AC-1 — The runtime-coupled-subcommand disposition is implemented, not just decided, with a behavioral oracle per subcommand.**
Verified by: for any subcommand reimplemented native, a parity test drives native-vs-Python over real fixtures (transcript fixtures for `context-budget`; `_mods` fixtures for the standing subcommands) and asserts equivalent results; for any kept-Python/shim, a behavioral test that the FO/ensign path invokes it correctly; for any scoped-out, a behavioral test of the defined fallback (e.g. the FO fresh-dispatches when the budget tool is absent rather than erroring).

**AC-2 — The FO ensign-reuse path has a defined, tested behavior when the native binary cannot read a context budget.**
Verified by: a behavior fixture exercises the reuse decision with the budget source absent/unavailable and asserts the FO's documented fallback (fresh-dispatch, not crash, not silent reuse). This closes the "steady-state Python dependency" question with an observable contract, whatever the disposition.

**AC-3 — The generic operating-contract core carries no Claude-Code-specific runtime coupling; Claude-specific content lives in the Claude runtime adapter / a Claude-specific seam.**
Verified by: a structural invariant over the real contract/source artifacts — the generic core (`first-officer-shared-core.md` and the host-neutral binary packages) resolves with zero Claude-Code-specific runtime references (transcript paths, `~/.claude` layout, team-config assumptions), all such references residing in the Claude adapter / Claude-specific package. Framed as an invariant over real files, not a substring spelling check.

## Test gates

- `go test ./...`
- Parity / behavior tests for whatever subcommand disposition ideation chooses.
- The host-neutrality invariant for the generic contract core.

## Notes

Sequences after `native-dispatch-helper` (`7w8w5nsa5mbc807b3jb88psv`), which defines the native dispatch surface and the package seam this entity extends. native-dispatch-helper's `build` is scoped to NON-`_mods` workflows (the self-hosted `docs/dev` path); this entity owns `build`'s `_mods`/`show-standing` fetch-line branch + the standing subcommands, closing the `_mods` parity gap.

Per the captain's note: if a design choice here rests on an unverified mechanism (e.g. exact transcript-budget parity, or whether a host-neutral seam is achievable without per-host branching), ideation must SPIKE it — run the smallest end-to-end exercise first and record the behavioral evidence — rather than assert it.

Relation to the deliverable-principles proposal (`docs/dev/_proposals/encoding-deliverable-principles.md`): that proposal's contract edits and this entity's contract reorganization both touch the operating-contract reference files; coordinate so they don't collide, and prefer the vendored-copy-first authoring direction documented there.

**Context-budget 1M-detection bug (found 2026-05-30, fold into this entity's scope).** `context_limit_for_model` returns the 200k DEFAULT for `claude-opus-4-8` ensigns: the `EXTENDED` list has `claude-opus-4-7` but NOT `4-8`, and spawned ensigns drop the `[1m]` suffix in their team-config/jsonl `model` field (only the lead keeps `claude-opus-4-8[1m]`). But since opus **4-7**, ensigns run the 1M window REGARDLESS of the dropped suffix (the suffix was only load-bearing back on 4-6). Result: `reuse_ok` false-negatives every opus-4-8 ensign at the wrong 200k denominator (a 159k-resident ensign reads as 80% when it is ~16% of 1M). Fix: a forward FAMILY RULE — parse the opus minor version and return 1M for minor ≥ 7 — so the mapping never goes stale on the next release; do NOT depend on the `[1m]` suffix or a per-release exact-name list. Applies whether context-budget stays Python or is reimplemented native. Behavioral test: a fixture stamping `claude-opus-4-8` (no suffix) asserts `context_limit` == 1_000_000 and `reuse_ok` true at a sub-600k resident count.
