---
id: 8xyvkvsgb93sch31cmz6nf9w
title: "debrief assumes workflow status executable exists"
status: ideation
source: "GitHub issue #175 (filed by Kent Chen / iamcxa, 2026-04-30)"
started: 2026-04-30T19:47:24Z
completed:
verdict:
score: 0.55
worktree:
issue: "#175"
pr:
mod-block:
---

## Problem

`spacedock:debrief` Phase 2e ("What's Next" extraction, `skills/debrief/SKILL.md:152`) instructs agents to run `{dir}/status --next` and `{dir}/status` against the workflow directory. Modern Spacedock ships the `status` viewer with the plugin at `{spacedock_plugin_dir}/skills/commission/bin/status` and invokes it as `... --workflow-dir {dir}`; workflows commissioned with newer versions do not carry a local `{dir}/status` executable. The debrief skill treats the local script as mandatory and fails with `no such file or directory` when absent, breaking Phase 2e silently for any newer-than-0.10.1 workflow.

Reporter (Kent Chen) hit this on a workflow commissioned by `spacedock@0.10.1` debriefed with plugin `spacedock@0.10.2`.

## Selected approach: A — Document the fallback explicitly

Pick **Shape A (documented fallback)** over Shape B (runtime detect-and-route).

Justification:
- Debrief is prose-driven; an agent reading the skill can branch on "executable exists vs not" without us teaching them a preflight ritual. Adding detect-and-route is more text and more conditional logic for the same outcome.
- This task is adjacent to **#5a `commission-readme-portable-status-path`** (also in flight). #5a is moving status-invocation knowledge out of generated READMEs and into the first-officer skill. Once #5a lands, the local `{dir}/status` artifact becomes purely a 0.10.x-and-earlier vestige; documenting "plugin-shipped is the primary path, local is a legacy fallback" is the shape that ages well. We do not coordinate with #5a here — just note the directional alignment.
- Frontmatter-only fallback is already within agent capabilities (Phase 2e enumerates entity files for the same data); we just need to license it in prose as the documented degraded mode rather than a silent improvisation.

### Concrete edits in `skills/debrief/SKILL.md`

Rewrite Phase 2e ("What's next", currently a 6-line block at `skills/debrief/SKILL.md:150-156`) so the primary invocation is the plugin-shipped status viewer, with two documented fallbacks:

1. **Primary**: `{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir} --next` and `... --workflow-dir {dir}` for the dispatchable list and overview.
2. **Legacy fallback**: if the plugin-shipped status is unreachable AND a local `{dir}/status` exists, invoke `{dir}/status --next` / `{dir}/status` (back-compat with workflows commissioned by spacedock<=0.10.x that still ship a local script).
3. **Degraded fallback**: if neither is available, scan entity frontmatter directly to reconstruct the same three views (dispatchable, gate-blocked, in-progress with non-empty `worktree`). Note this in the rendered "What's Next" section so the captain knows the data was derived without the status helper.

Keep the section's downstream consumers (the "What's Next" section in the draft and final debrief at `skills/debrief/SKILL.md:206-207, 332-336`) unchanged — only the extraction mechanism shifts.

## Acceptance criteria

- AC1: Debrief Phase 2e runs to completion against a workflow that has no `{dir}/status` executable.
  - Verified by: behavioral fixture below (a workflow directory containing only README.md + entity files, no `status` file). Phase 2e produces a populated "What's Next" section.

- AC2: `skills/debrief/SKILL.md` Phase 2e prose names plugin-shipped status as the primary invocation and documents both the legacy `{dir}/status` fallback and the frontmatter-scan degraded fallback.
  - Verified by: grep guards on `skills/debrief/SKILL.md` — must contain both `{spacedock_plugin_dir}/skills/commission/bin/status` and a sentence licensing frontmatter-scan when no status helper is reachable; must NOT present `{dir}/status` as the unconditional primary.

- AC3: Frontmatter-scan degraded mode is surfaced to the captain in the rendered "What's Next" section when triggered (so a debrief produced without a status helper is self-describing).
  - Verified by: skill prose at the Phase 2e edit site instructs the agent to add a one-line "_(reconstructed from entity frontmatter — no status helper available)_" annotation in that section when the degraded path was taken.

- AC4: Adjacent #174 (debrief discovery ignoring `.claude/worktrees`) and #5a (commission README portability) remain independent — this task ships alone.
  - Verified by: implementation diff touches only `skills/debrief/SKILL.md` (Phase 2e block) and the regression fixture/test; no edits to `skills/commission/SKILL.md` or to debrief discovery (Phase 1) prose.

## Test plan

**Regression fixture** — add `tests/fixtures/debrief-no-local-status/` (new fixture directory):
- `README.md` with minimal Spacedock workflow frontmatter (`commissioned-by: spacedock@<current>`, one stage, slug id-style for simplicity).
- 2-3 entity `.md` files spanning at least one dispatchable status, one gated status, and one with a populated `worktree` field — enough to exercise all three "What's Next" buckets.
- **No `status` executable** — this is the crux of the fixture. Existing fixtures (e.g. `tests/fixtures/multi-stage-pipeline/status`) all ship a local `status` script; this one deliberately omits it.

**Test entrypoint** — no debrief test host exists today (`tests/` has 40+ files, none named `test_debrief_*`). Two options for the implementation stage to choose from:

- Add `tests/test_debrief_skill.py` as a new test host. Initial test asserts the prose-level guards from AC2/AC3 (grep `skills/debrief/SKILL.md` for the required tokens and the degraded-mode annotation prose). This is a static check — appropriate for a skill that is prose-executed by an agent rather than a binary we can subprocess.
- Optionally add a behavioral check that invokes the plugin-shipped status against the new fixture (`skills/commission/bin/status --workflow-dir tests/fixtures/debrief-no-local-status --next`) and asserts the dispatchable entity is reported. This proves the documented primary path actually works against a fixture matching the failure shape — but does not exercise the debrief skill itself (no agent in the loop). Keep this as a supplementary fixture sanity test, not the regression test for AC1.

The AC1 behavioral verification (debrief runs to completion against the no-local-status fixture) is best validated by a pilot manually invoking `spacedock:debrief` against the fixture during stage-report time and capturing the rendered output in the entity body. Document this expectation in the test plan; a fully automated agent-in-the-loop test is out of scope.

## Out of scope

- Coordinating the prose with #5a (`commission-readme-portable-status-path`). #5a removes status-usage prose from generated READMEs; this task fixes the debrief skill's status invocation. The two land independently and the directional alignment is intentional but not enforced.
- Fixing #174 (debrief discovery ignoring `.claude/worktrees`). Same skill, same reporter, but a different code path (Phase 1 Discovery vs Phase 2e Extraction).
- Removing the local-`{dir}/status` fallback entirely. Workflows commissioned by spacedock<=0.10.x in the wild still carry it; the legacy fallback is cheap to document and avoids a silent regression for those users.
- Adding a `spacedock` CLI wrapper on PATH (the systemic portability fix; cross-referenced from #5a, captain-future).

## Stage Report: ideation

PASS

- Picked Shape A (documented fallback) over Shape B (runtime detect-and-route). Justification: prose-driven skill, ages well alongside #5a, frontmatter-scan already within agent capabilities.
- Test plan names a regression fixture (`tests/fixtures/debrief-no-local-status/`, deliberately omits the local `status` executable) and a test entrypoint (`tests/test_debrief_skill.py` — does not exist today; would be a new test host). Distinguished prose-guard tests (automatable) from behavioral debrief-runs-to-completion verification (pilot-driven, not fully automatable).
- AC items are end-state properties with concrete `Verified by:` clauses: AC1 (fixture passes), AC2 (skill prose contains required tokens), AC3 (degraded-mode self-annotation), AC4 (diff scoped to this skill + fixture only).
- Adjacency to #5a noted in the entity body without coordination dependency.
- No worktree; edited the entity file directly.
