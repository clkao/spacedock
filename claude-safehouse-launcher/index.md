---
id: e72ambzmkkt3hp1whpz2tczr
title: spacedock claude — safehouse launcher (replace the manual invocation)
status: backlog
source: sprint — Ship the Launcher slice A (captain, 2026-05-30); harvested from superseded ~/git/spacedock launcher plan 9bt646cz0h4q79g98qz68k9d
started:
completed:
verdict:
score: "0.40"
worktree:
issue:
---

Make `spacedock claude` a drop-in replacement for the captain's manual Claude Code invocation, launching the first officer through safehouse when a workdir safehouse profile is present. This builds the shared safehouse-detection + interposition helper that the codex launcher reuses.

## Target behavior (captain, 2026-05-30 — ideation hardens these into behavioral oracles)

- **When `.safehouse` exists in the working directory:**
  `safehouse --trust-workdir-config [extra-args] -- claude --dangerously-skip-permissions --agent spacedock:first-officer [initial-prompt]`
  — the initial prompt is appended UNLESS `--resume` is among the forwarded args.
- **When `.safehouse` does NOT exist:** launch plain `claude --agent spacedock:first-officer [forwarded args]` (no safehouse).
- The front door runs the plugin-presence / contract gate already built in `tq` (`internal/cli/frontdoor.go`) before any launch.

## Provenance / salvage

- Supersedes `~/git/spacedock` launcher plan `9bt646cz0h4q79g98qz68k9d` (status=implementation, dispatch-held). Harvest from its worktree (`~/git/spacedock/.worktrees/spacedock-ensign-spacedock-launcher-binary`):
  - `internal/claude/run.go` — `syscall.Exec` process-replace plumbing, `safehouse` LookPath → exit 127, plugin-detect gate.
  - `docs/plans/_evidence/spacedock-launcher-binary-ideation/safehouse-stub.sh` — argv-recording stub pattern for the canonical-argv test.
- NOT salvageable: its `buildSafehouseArgv` is pre-F11 / pre-this-model (no `.safehouse` detection, no `--trust-workdir-config`, no `--dangerously-skip-permissions`, no prompt/`--resume`).
- Lands on the same `internal/cli/frontdoor.go` as `tq`'s front-door fix → MUST sequence after tq merges (no parallel-merge).

## Acceptance criteria (provisional — ideation hardens each into an exercise-and-observe oracle)

**AC-1 — `.safehouse`-present path emits the canonical safehouse argv.**
Verified by: a Go test with a `safehouse` stub on PATH (recording argv) + a fixture `.safehouse` in the workdir observes `spacedock claude --foo` execs `safehouse --trust-workdir-config -- claude --dangerously-skip-permissions --agent spacedock:first-officer --foo`.

**AC-2 — No-`.safehouse` path launches plain claude (no safehouse).**
Verified by: same harness with no `.safehouse` present observes `claude --agent spacedock:first-officer --foo` and that the safehouse stub was never invoked.

**AC-3 — Missing plugin → clear error, rc≠0, no launch.**
Verified by: temp HOME with no plugin; assert install-hint on stderr and neither safehouse nor claude invoked. (Exercises tq's front-door gate through the launcher path.)

**AC-4 — Missing safehouse (when `.safehouse` present) → clear error + install hint, rc≠0.**
Verified by: a Go test with `.safehouse` present and `safehouse` absent from PATH observes a pinned install-hint on stderr, rc≠0, and no claude launch.

**AC-5 — `--resume` suppresses the injected initial prompt.**
Verified by: the stub harness observes that forwarding `--resume` produces argv WITHOUT the initial-prompt token.

**AC-6 (captain-run, closes F3) — live safehouse smoke.**
`safehouse --trust-workdir-config -- claude --agent spacedock:first-officer --help` yields claude's help (not a safehouse flag error) in a real unsandboxed shell. Run by the captain outside the sandbox; recorded as evidence. This is the riskiest unknown (Risk A) and gates implementation lock.

## Notes
- We run inside safehouse now; nested safehouse won't run here, so AC-6 is captain-run. We design, implement, and stub-test up to that line.
- The injected initial-prompt content (what FO-bootstrap prompt, if any) is an ideation decision.
