---
id: 2x6ra77h668vwr3gxc7zh80f
title: "FO break-glass manual dispatch template adds Skill(skill=\"spacedock:ensign\") first-action directive"
status: done
source: "carved out of 4q9 (claude-team-build-dispatch-prompt-shrink) cycle-2 ideation — the breakglass-template inconsistency surfaced there is orthogonal to the bigger fetch-on-demand restructure (0x9). Filed standalone so it can land independently; if 0x9 lands first and supersedes this, it can be closed without code change."
started:
completed: 2026-05-21T05:02:00Z
verdict: REJECTED
score:
worktree:
issue:
pr:
archived: 2026-05-21T05:02:21Z
---

## Status: superseded by 0x9 (PR #231 merged 2026-05-21)

Entity `0x9` (`claude-team-build-fetch-on-demand-dispatch-spec`, PR #231, merged at commit `d4c1ba3e`) absorbed the breakglass-template Skill-invoke edit in the same PR. See the archived 0x9 entity's `## Design` → "Breakglass FO template" section + `### Relationship to entity 2x6` for the absorption decision. The breakglass template at `skills/first-officer/references/claude-first-officer-runtime.md:114-123` now opens with the `Skill(skill="spacedock:ensign")` first-action directive as 0x9's PR landed. This entity ships REJECTED-superseded with no code changes of its own — the work was completed under 0x9.

---

## Original problem statement (preserved for audit trail)

## Problem

The FO's break-glass manual dispatch template at `skills/first-officer/references/claude-first-officer-runtime.md:114-123` is the fallback used when `claude-team build` exits non-zero. Its current prompt body does NOT carry the `## First action — Skill(skill="spacedock:ensign")` directive that `claude-team build` emits as the first prompt-body content for the normal path.

This is the only documented path where an ensign is dispatched without the boot directive. The asymmetry is silent — the breakglass template otherwise looks similar enough to the helper output that the gap is easy to miss.

## Why this matters

The Skill-invoke directive is what loads the operating contract (`ensign-shared-core.md`, `claude-ensign-runtime.md`) into the ensign's context. Without it, the ensign is relying on the agent-definition skill-preload (`skills:` frontmatter on the agent file), which is blocked for plugin agents by upstream `claude-code#25834` and observed as failing for haiku-class models. See `_archive/agent-boot-skill-preload.md` for the detailed failure mode and the body-fallback directive that fixes it for the helper-emitted path.

The breakglass path bypassing this fallback means: any dispatch routed via breakglass on haiku is a coin flip on whether the operating contract loads. The breakglass path is rare (only fires when `claude-team build` exits non-zero), but it exists specifically to be a reliable degraded fallback. Adding the directive removes one footgun from the fallback path.

## Proposed approach

Add the `Skill(skill="spacedock:ensign")` first-action directive at the head of the breakglass template's `prompt=` slot in `claude-first-officer-runtime.md:114-123`. Approximately one extra paragraph at the start of the template string.

Optionally update the explanatory paragraph at `claude-first-officer-runtime.md:123` to note that the breakglass also relies on skill-load via the directive (not just on the helper output's skill-load discipline).

## Out of scope

- Any changes to `claude-team build`'s helper-emitted prompt shape (that's `0x9`'s scope).
- Any changes to the ensign skill body itself.
- Any changes to `claude-first-officer-runtime.md` outside the breakglass section.

## Relationship to other entities

- **Supersession risk:** if `0x93enxe1hpmk95a25476zyn` (claude-team-build-fetch-on-demand-dispatch-spec) lands first and restructures the dispatch contract significantly, the breakglass template will be restructured in the same pass and this entity becomes unnecessary. If that happens, close this with verdict REJECTED and source-note pointing at `0x9`.
- **Independent value otherwise:** if `0x9` lingers or scope-creeps, this small fix can land in isolation as a haiku/low robustness improvement.

## Scale context

- Spacedock version: 0.11.2
- Surfaced in: `_archive/claude-team-build-dispatch-prompt-shrink.md` cycle-2 design (rejected entity), the "Breakglass FO template consistency" subsection
- Estimated complexity: ~10 lines of edits to one reference file; one static lint test
