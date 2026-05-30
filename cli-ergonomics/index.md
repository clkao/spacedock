---
id: xdt3cjnppc89amm5g23s86mm
title: CLI ergonomics — workflow auto-discovery and actionable errors
status: backlog
source: session 1 debrief — ergonomics
score: "0.30"
worktree:
---

Make `spacedock status` forgiving and discoverable. Today `--workflow-dir` is mandatory and unforgiving: omitting it (or running a bare `spacedock status --boot`) falls back to the cwd / `dirname(__file__)` and fails with a misleading "README.md has no stages block"; and pointing at a deprecated state dir post-migration errors with `non-numeric sequential id` instead of naming the real problem.

## Acceptance criteria

**AC-1 - With no `--workflow-dir`, `spacedock status` discovers the enclosing commissioned workflow by walking up from the cwd to the nearest README whose frontmatter is `commissioned-by: spacedock@…` (like git finds `.git`).**
Verified by: a fixture tree where the cwd is several levels below the workflow README; `spacedock status` (no flag) resolves and renders that workflow; a no-workflow cwd yields the actionable error in AC-2 (not the cwd fallback).

**AC-2 - Errors are actionable: they name the fix, not just the symptom.**
Verified by: bare `spacedock status` with no discoverable workflow prints "no Spacedock workflow here — pass --workflow-dir or run inside a workflow"; pointing at a `state:` checkout (definition README absent) prints "this is a state checkout; point --workflow-dir at the definition dir (the one whose README declares `state:`)" rather than a downstream id/stage error. Golden tests on stderr + exit code.

**AC-3 - Discoverable top-level verbs.**
Verified by: `spacedock new <slug>` wraps the `--new` atomic create; `spacedock completion <shell>` emits a completion script; both appear in `spacedock --help`.

## Test gates

- `go test ./...`
- Auto-discovery walk-up fixture test; no-workflow + state-dir actionable-error golden tests; `spacedock new` / `completion` smoke tests.

## Notes

From the session-1 debrief ergonomics list. Auto-discovery is the biggest single ergonomic win; actionable errors are pure UX. `spacedock doctor` lives in spacedock-packaging; `spacedock dispatch` in native-dispatch-helper; `spacedock claude`/`codex` in spacedock-packaging — not duplicated here.
