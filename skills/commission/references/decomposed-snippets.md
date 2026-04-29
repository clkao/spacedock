# Decomposed snippets

The structural decomposition the captain-facing templates assemble from. This reference is the source-of-truth for the layer-assembly fallback (when no template matches, commission walks this list and includes the snippets the cues fired on).

## Base scaffolding (every workflow)

- Frontmatter: `entity-type`, `entity-label`, `entity-label-plural`, `commissioned-by` stamp.
- At least one stage with `initial: true`.
- At least one stage with `terminal: true`.
- `id-style: sequential` (default; `sd-b32` only on confirmed concurrent cross-branch entity creation; `slug` when slug is the canonical identity).
- Flat entity file (`{id}-{slug}.md` or `{slug}.md`) by default; folder-based (`{slug}/index.md` + siblings) when the entity carries artifacts beyond what reads inline.
- File Naming, Schema, Stages, Workflow State, and Entity Template sections in the README body.

## Repo-mutation layer

Fires when any stage modifies the codebase.

- `worktree: true` flag on the repo-mutating stages.
- `pr-merge` mod installed in `_mods/` and referenced in the workflow README, when the shipping ritual is "open a PR, get review, merge to main." Captains who commit directly to main or who don't ship via PR review skip the mod.
- Phase 1 framing: stages-stay-clean teaching moment; the mod removes the need for a `pr_open` or `awaiting_merge` stage.

## Parked-stages layer

Fires when any stage waits on external response, evidence accumulation, or time passing.

- Parked flag on the waiting stage(s).
- `silence-watcher` (or equivalent) idle-mod offered when the parked stage has timeout or nudge semantics (entity should advance after N days of no external event).
- Phase 1 framing: parked stages are normal — entities can sit indefinitely; the idle-mod handles stalled entities.

## Entity-template snippets

Composable into the entity body, independent of structural layers. The three pre-baked snippets:

- **Hypothesis-result snippet** (used by `experiment` template): hypothesis / methodology / smoke result / run result / holdout result / verdict.
- **Refinement snippet** (used by `refinement` template): draft / review notes / final.
- **Development snippet** (used by `development` template): problem / proposed approach / acceptance criteria / test plan / out-of-scope.

The refinement README documents three additional inline variants — outreach (contact / message / sent-at / response), integration (incoming record / enrichment notes / sync target), content production (artifact draft / review notes / publish target) — without shipping them as separate template files, since they're variants of the refinement snippet structure rather than distinct shapes.

## Captain-facing templates as snippet combinations

The three template files are popular pre-baked combinations of the snippets above:

- `refinement` = base + refinement snippet.
- `development` = base + repo-mutation layer + development snippet.
- `experiment` = base + parked-stages layer + hypothesis-result snippet.

Other combinations (e.g., a workflow with both repo-mutation and parked-stages layers active) are assembled on demand from the snippet list during the layer-assembly fallback path; they don't need their own template file.
