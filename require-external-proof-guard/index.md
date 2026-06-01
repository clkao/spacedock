---
id: 2adcrvj56b5camy1v70c4ncc
title: Refuse to close a task whose only "proof" is re-reading its own write-up
status: backlog
source: hx decomposition (A of 3) — captain 2026-06-01; staff review
score: "0.32"
started:
completed:
verdict:
worktree:
issue:
---

**What this is for (plain).** Today nothing stops a task from being marked finished when its "proof"
is just pointing back at its own description — no real test, no command, no actual check behind it.
This adds a guard in the `spacedock` tool that refuses to finalize such a task (with an override flag
for the rare deliberate exception), so every finished task carries evidence that can actually fail.

**Value to the user / FO.** A mechanical backstop under the human gate: the FO no longer has to catch
every empty or circular "proof" by eye — the tool blocks the close and the run-time validate flags it.
Finished work always has real evidence, so the workflow can't quietly rubber-stamp a task that proves
nothing.

This is part A of three from the now-superseded parent `deliverable-contract-hardening`
(id `hxs93wd0bjwhc3vsjwx1seew`) — the parent's full design, the 172-AC classifier spike, and the
staff review remain the reference. This child is the CODE half: the guard in `internal/status`.

## Scope (ideation hardens; carries the staff-review corrections)

- A check in `spacedock status --validate` that flags a task whose proof clause cites nothing external
  to itself, plus a guard on the terminal "mark done / passed" set that refuses the transition (leaves
  the task unchanged) unless `--force` is passed.
- The classifier mechanism is already spike-proven (scope to the proof clause; ignore quoted examples;
  require an external-proof word like a test/command/file to be present) — ship the spike corpus + the
  adversarial cases as the guard's test table. **Refresh the stale spike counts first** (the corpus
  grew from 79 to ~172 tasks; the old headline example no longer reproduces).

## Acceptance criteria (provisional)

**AC-1 — Marking a self-referential-proof task done is refused; a real-proof task closes cleanly.**
Verified by: native behavioral tests (NOT launcher-vs-oracle parity — the Python comparison tool has
no such check, so a parity test would falsely fail; use a native-only assertion in the STATE_BACKEND
divergence style). Fixtures: (a) self-reference-only task → terminal set exits 1, frontmatter
unchanged; (b) external-proof task → reaches done; (c) `--force` → bypass with the override warning;
(d) `--validate` emits the standard evidence line for the flagged task and `VALID` for a clean one.
Flip-test: change a fixture's proof from self-reference to external and confirm the guard stops firing.

**AC-2 — Everyday reads are not gated by this check.** Verified by: plain `status` / `--next` /
`--boot` still succeed on a workflow that contains a self-referential task (the check is invoked only
by `--validate` and the terminal set, NOT on the read path — otherwise one bad task breaks every
read, with no override). The existing `--validate` parity tests are reconciled (the new native
evidence line is handled, not left to red the suite).

**AC-3 — One shared classifier serves both the `--validate` flag and the terminal-set guard.**
Verified by: a single body-parser/classifier helper (new — neither path parses the task body today)
is the only place the rule lives, so the two surfaces can't drift; a test asserts both call it.

## Out of scope
- The prose/principle edits (part B) and the portability test (part C).
- Any soft warning lints beyond this one hard-backed check.
