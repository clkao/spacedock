<!-- ABOUTME: Recovery procedures for the Claude Code first officer — TeamCreate failure ladder, -->
<!-- ABOUTME: Degraded Mode, Break-Glass manual dispatch, and cooperative shutdown sweep. -->

# Claude Code First Officer Runtime — Recovery

This file contains fault-recovery procedures for the Claude Code first officer runtime. It is loaded on demand when a fault signal occurs — see the core runtime file for the trigger list.

## TeamCreate Failure Recovery (priority-ordered ladder)

If TeamCreate or any subsequent `Agent()` dispatch surfaces "Team does not exist" or any equivalent registry-desync signal mid-session, follow this ladder in order — do NOT retry within the same tier:

1. **Fresh-suffixed TeamCreate.** Attempt one new `TeamCreate` with a fresh name `{project_name}-{dir_basename}-{YYYYMMDD-HHMM}-{shortuuid}` computed at call time (new timestamp, new shortuuid, distinct from any name used earlier this session). Retry to the same team name is banned. Do NOT call `TeamDelete` on the failed team — the registry is already desynced and another `TeamDelete → TeamCreate` cycle will re-contaminate the same slot per anthropics/claude-code#36806. Store the returned `team_name`. All prior agent names are presumed zombified — do not SendMessage them; re-dispatch from entity frontmatter.
2. **Fall back to Degraded Mode per the Degraded Mode section below.** A second dispatch failure (including failure of the tier-1 fresh-suffixed TeamCreate, or a second "Team does not exist" at any point in the session) trips Degraded Mode immediately.
3. **Surface to captain** with an explicit recovery prompt if tiers 1 and 2 both fail (e.g., TeamCreate errors with quota or internal failure on the fresh name, AND Degraded Mode cannot be entered because `Agent` itself is unavailable). Do not silently retry. Do not block indefinitely — report the failure, name the tiers attempted, and wait for captain direction.

## Break-Glass Manual Dispatch

**Fallback ONLY when `claude-team build` exits non-zero or is unavailable.** Do NOT use this template while the helper is working. Report the helper failure to the captain before proceeding. Use this minimal template as a degraded fallback:
```
Agent(
    subagent_type="{dispatch_agent_id}",
    name="{worker_key}-{slug}-{stage}",
    team_name="{team_name}",
    model="{effective_model}",
    prompt="You are working on: {entity title}\n\nStage: {stage}\n\n### Stage definition:\n\n{copy stage subsection from README verbatim}\n\nRead the entity file at {entity_file_path}.\n\n### Completion checklist\n\n{numbered checklist}\n\n### Completion Signal\n\nSendMessage(to=\"team-lead\", message=\"Done: {entity title} completed {stage}. Report written to {entity_file_path}.\")"
)
```
The break-glass template omits worktree instructions, feedback context, and scope notes. The `model=` slot is conditional — include it only when the stage (or `stages.defaults`) declares a model from `sonnet | opus | haiku`; omit the entire `model=` argument otherwise. Use only when the helper is unavailable.

## Degraded Mode

Degraded Mode is an explicit, session-wide mid-session transition. Once entered, it persists until the session ends — there is no recovery back to teams mode in the same session.

### Triggers

Any one of the following trips Degraded Mode:

- First "Team does not exist" error (or equivalent registry-desync signal) surfaced by `Agent()` or any team-registry tool.
- Any SECOND dispatch failure within the session — no time window, no durable counter. The counter-free rule is deliberate: the FO cannot reliably track failure timestamps across context pressure and idle notifications, so "second failure anywhere in the session" is the fail-early trigger.
- Captain command `/spacedock bare` (explicit operator-initiated degrade).

### Effects

Once Degraded Mode is active, the following invariants hold for the remainder of the session:

- No `team_name` parameter on any subsequent `Agent()` dispatch. The input JSON sets `team_name: null` and `bare_mode: true`; `claude-team build` emits a bare-mode Agent call with `name` and `team_name` absent.
- Every stage dispatches fresh and blocks until completion. No concurrent dispatch; one entity through one stage at a time.
- No SendMessage reuse of prior agent names. Stage advancement is always a fresh `Agent()` dispatch seeded from entity frontmatter. `SendMessage(to="{ensign_name}")` against any pre-degrade name is forbidden.

### Captain Report Template

On Degraded Mode entry, the FO emits the following sentence verbatim to the captain (direct text output, not SendMessage):

> Falling back to bare mode for the remainder of this session due to team-infrastructure failure. Prior team agents are presumed-zombified; I will not route work to them or through the team registry. If you want to escalate: restart the session to retry team mode with a fresh name, or let me continue — every stage will still complete, just without concurrent dispatch.

### Cooperative Shutdown Sweep

On Degraded Mode entry, perform a single-pass cooperative shutdown sweep of every known agent name from session memory: one `SendMessage(to="{ensign_name}", message="shutdown_request")` per name. Ignore failures — best-effort, not transactional. Do not retry, track responses, or block on the outcome; proceed immediately to the first fresh bare-mode dispatch.

Exempt any agent whose entity is in an active feedback-cycle state (tracked via a `### Feedback Cycles` subsection in the entity body). Those reviewers may hold load-bearing context from the prior cycle that re-dispatch cannot reconstruct. Sweep feedback-cycle reviewers only on explicit captain confirmation.
