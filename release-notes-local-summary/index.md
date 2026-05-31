---
id: 7havdt4r7mett5q13tcxaxdj
title: Local release script — LLM-summarized release notes (reuse old release.sh prompt), CI builds on tag push
status: backlog
source: captain (2026-05-31) — "refine the release script, use the python-based release prompt to simplify the release notes; local so I can tweak, build still triggers on tag push"
started:
completed:
verdict:
score: "0.32"
worktree:
issue:
---

The release notes on the GitHub Release are goreleaser's **default raw changelog** — `.goreleaser.yaml` has no `changelog:` config, so the notes are an unfiltered commit dump including all the workflow-state noise (`dispatch:`/`advance:`/`merge:` entity commits, archived-task frontmatter). Refine the release flow to produce **clean, user-value release notes** via an LLM summary, generated **LOCALLY** (so the captain can review/tweak before tagging), while the **build still triggers on the `v*` tag push** (CI goreleaser stays the builder/publisher).

## Reuse the proven prompt (captain-confirmed)

The OLD `scripts/release.sh` (on `origin/main`, the Python-era plugin) already had the right prompt — pipe `git log` since the last tag through `claude -p`:

> "Summarize these git commits into a release changelog for spacedock v$VERSION. Plain text only — no markdown headers, no bold/italic. Start with one sentence describing the major theme of this release. Then list individual changes as '- ' bullet lines. For each bullet, lead with the user value (what upgrading gives you), then briefly describe what changed at a high level. **Ignore workflow state changes (dispatch/done/backlog/validation commits, archived task frontmatter updates, entity file changes under docs/plans/).** Group related commits into single entries."

(`--model opus --effort low`; **fall back to raw `git log --oneline` if `claude` is unavailable**.) Adapt the ignore-list to this repo's noise (`docs/dev/.spacedock-state/` entity commits, `dispatch:`/`advance:`/`merge:`/`archive:` prefixes, the `release: stamp …` + `next: bump …` CI commits).

## Design seam (the local↔CI question — ideation decides)

Local generates the notes; CI (release.yml on `v*`) still builds. The notes must reach the GitHub Release. Recommended seam to validate:
- **Local script** generates the summary, shows it for the captain to edit, then creates the **annotated tag with the (tweaked) notes as the tag message** and pushes the tag.
- **`release.yml`** extracts the tag annotation (`git tag -l --format='%(contents)' $TAG`) and passes it to goreleaser via **`--release-notes <file>`** (goreleaser otherwise regenerates its raw changelog). No `claude` needed in CI — it just consumes the locally-authored notes.

Alternative seams ideation should weigh: a committed `RELEASE_NOTES.md` passed via `--release-notes`; or goreleaser's tag-body support if it fits. Pick the one that keeps "local-authored, CI-built, captain-tweakable."

## Coordinate with n1's existing release tooling

This is NOT greenfield — n1 shipped `internal/release/release.go` (`StampVersion`, `BumpCalendarVersion`) + `cmd/spacedock-release/main.go` + the `release.yml` (version-stamp) and `next-publish.yml` (calendar-bump) workflows. The local release flow should ORCHESTRATE: version decision (captain stays on 0.19.x until they flip the minor) → notes generation/tweak → annotated tag + push. Decide whether the local entry point extends `cmd/spacedock-release` (Go, consistent with the binary) or a `scripts/release.sh` (bash, like the old one). Lean Go to match the binary + n1's tooling.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — Local notes generation reuses the prompt and filters workflow noise.** End state: a local command produces plain-text, user-value-bulleted notes from `git log` since the last tag, filtering the workflow-state commits per the prompt; falls back to raw `git log` when `claude` is absent. Verified by: a unit test over the filter/format step (fixture git-log → expected filtered input to the prompt) + the no-claude fallback path.
**AC-2 — Captain can review/tweak before the tag is cut.** End state: the script shows the proposed notes and only proceeds to tag on confirmation (editable). Verified by: the interactive-confirmation seam tested via the injected-IO pattern (no live prompt in CI).
**AC-3 — The locally-authored notes land on the GitHub Release, build still on tag push.** End state: the tag carries the notes; `release.yml` passes them to goreleaser (`--release-notes`) so the published Release shows the clean notes, not goreleaser's raw changelog; the build is still triggered by the `v*` tag push (CI unchanged as the builder). Verified by: a release.yml step asserting the tag-annotation→`--release-notes` wiring (fixture/CI test); the seam proven on the next real release.

## Notes
- Should land **before the 0.19.2 ship** (which bundles cli-cobra + 38) so that release gets clean notes — it's release-prep for the coordinated ship.
- Touches `cmd/spacedock-release` / `internal/release` / `.github/workflows/release.yml` / maybe `scripts/` — NO overlap with `internal/cli` (cli-cobra) or dispatch (split-root), so safe to run in parallel.
- ANTHROPIC_API_KEY is already a repo secret, but with LOCAL generation the script uses the captain's own `claude` auth; CI needs no key for this (it only consumes the tag notes).
