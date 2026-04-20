# Claude Code First Officer Runtime

This file defines how the shared first-officer core executes on Claude Code.

## Team Creation

At startup (after reading the README, before dispatch):

1. **Probe for TeamCreate and run it first.** `TeamCreate` MUST be the first team-mode tool call in every session, before ANY `spawn-standing`, `Agent`, or `SendMessage` invocation. Run `ToolSearch(query="select:TeamCreate", max_results=1)`. If the result contains a TeamCreate definition, derive `{project_name}` from `basename $(git rev-parse --show-toplevel)` and `{dir_basename}` from the workflow directory path, then run `TeamCreate(team_name="{project_name}-{dir_basename}-{YYYYMMDD-HHMM}-{shortuuid}")`. The timestamp token must be lowercase and hyphen-separated — no uppercase, no colons — to stay compatible with Claude Code's NAME_PATTERN and the `claude-team` helper. `{shortuuid}` is eight lowercase alphanumeric characters.
   - **IMPORTANT:** TeamCreate may return a different `team_name` than requested. Always store the returned value and use it for all subsequent calls.
   - **NEVER delete existing team directories** (`rm -rf ~/.claude/teams/...`) — stale directories belong to other sessions.
2. If ToolSearch returns no match, enter **bare mode**: dispatch is sequential (one subagent at a time), completions return inline, feedback cycles are sequential re-dispatches. Report the mode to the captain. All workflow functionality is preserved.

**Diagnostic-only startup probe:** At startup the FO MAY inspect `~/.claude/teams/` with `ls` or `test -f ~/.claude/teams/{project_name}-{dir_basename}*/config.json` to REPORT stale on-disk team directories from prior sessions in the boot summary. This probe is DIAGNOSTIC-ONLY. Its result does NOT gate or skip `TeamCreate` — `TeamCreate` always runs with the fresh-suffixed name regardless of what the probe reports. On-disk state is not evidence of team health (Claude Code bug anthropics/claude-code#36806 leaves config files on disk after the in-memory registry desyncs). Deletion remains forbidden per the NEVER-delete constraint above — the probe surfaces stale directories; it does not authorize removal. No such probe belongs in the `## Dispatch Adapter` pre-dispatch path.

**TeamCreate recovery procedure:** Call TeamDelete in its own message (no other tool calls). Wait for the result. Then call TeamCreate in a subsequent message. Store the returned `team_name`. Do NOT combine TeamDelete, TeamCreate, or Agent dispatch in the same message — Claude Code executes tool calls in a message in parallel, and dependent calls will race. This procedure applies ONLY to the narrow "Already leading team" case at startup (where Claude Code's in-memory slot holds a team the FO wants to replace cleanly). It is NOT a mid-session failure recovery and MUST NOT be invoked in response to "Team does not exist" or any other registry-desync signal — see Degraded Mode below.

**TeamCreate failure recovery (priority-ordered ladder):** If TeamCreate or any subsequent `Agent()` dispatch surfaces "Team does not exist" or any equivalent registry-desync signal mid-session, follow this ladder in order — do NOT retry within the same tier:

1. **Fresh-suffixed TeamCreate.** Attempt one new `TeamCreate` with a fresh name `{project_name}-{dir_basename}-{YYYYMMDD-HHMM}-{shortuuid}` computed at call time (new timestamp, new shortuuid, distinct from any name used earlier this session). Retry to the same team name is banned. Do NOT call `TeamDelete` on the failed team — the registry is already desynced and another `TeamDelete → TeamCreate` cycle will re-contaminate the same slot per anthropics/claude-code#36806. Store the returned `team_name`. All prior agent names are presumed zombified — do not SendMessage them; re-dispatch from entity frontmatter.
2. **Fall back to Degraded Mode per the Degraded Mode section below.** A second dispatch failure (including failure of the tier-1 fresh-suffixed TeamCreate, or a second "Team does not exist" at any point in the session) trips Degraded Mode immediately.
3. **Surface to captain** with an explicit recovery prompt if tiers 1 and 2 both fail (e.g., TeamCreate errors with quota or internal failure on the fresh name, AND Degraded Mode cannot be entered because `Agent` itself is unavailable). Do not silently retry. Do not block indefinitely — report the failure, name the tiers attempted, and wait for captain direction.

**Block all Agent dispatch** until team setup resolves (tier-1 fresh-suffixed TeamCreate succeeds or Degraded Mode is entered). Never dispatch while team state is uncertain.

In single-entity mode, skip team creation. Use bare-mode dispatch for all agent spawning — the Agent tool without `team_name` blocks until the subagent completes, which prevents premature session termination in `-p` mode.

When filing a new task, use `status --next-id` to fetch only the next sequential ID. Reserve `status --boot` for startup diagnostics and broader workflow inventory.

### Standing teammate discovery pass

After team creation succeeds (the ladder has resolved and the returned `team_name` is known) and BEFORE entering the normal dispatch event loop, run the standing-teammate discovery pass:

1. Run `claude-team list-standing --workflow-dir {wd}` and consume its newline-delimited output (one absolute mod path per line, sorted alphabetically, empty stdout on zero matches). Do NOT grep mod frontmatter yourself; authoritative parsing is deferred to the helper.
2. Record the returned mod paths in session memory. **No spawn calls at boot.** Spawn is deferred to the first team-mode dispatch (see lazy-spawn below).

In single-entity (bare) mode and in Degraded Mode, discovery still runs (it is cheap — just `list-standing`), but lazy-spawn is skipped in those modes (no team to spawn into). Standing teammates are a team-scope concept; without a live team they have no lifecycle anchor.

### Standing teammate lazy-spawn

Before the first `Agent()` call that uses a `team_name` (i.e., the first non-bare dispatch), spawn all declared standing teammates:

1. For each declared standing-teammate mod path recorded during the discovery pass:
   a. Run `claude-team spawn-standing --mod {abs_path_to_mod} --team {team_name}`.
   b. If the helper emits JSON with top-level `status: "already-alive"`, log the reported `name` and skip to the next mod. Standing teammates are first-boot-wins across the captain session; subsequent workflows sharing the team pick up the live member.
   c. Otherwise the helper emits an Agent() call spec JSON with keys `subagent_type`, `name`, `team_name`, `model`, `prompt`. **Forward that spec verbatim** to the Agent tool — copy each field into the corresponding Agent() argument without paraphrasing the prompt, rewriting the name, or substituting the team. Same "forward verbatim" discipline as `claude-team build` output.
   d. The spawn is fire-and-forget. Do NOT block on the teammate's first idle notification before continuing to dispatch.
   e. If the helper exits non-zero on any mod (missing Agent Prompt section, invalid model enum, convention-violating trailing heading), surface the error to the captain and continue with the remaining mods. A broken mod does not block the workflow.
2. After all standing teammates are spawned (or skipped), proceed with the ensign `Agent()` dispatch.

This is a one-time cost at first dispatch. Subsequent dispatches skip the spawn pass entirely — the FO tracks "standing teammates spawned for this team" in session memory. In single-entity (bare) mode and in Degraded Mode, skip lazy-spawn (same as the discovery-pass skip note above). Prose-polish round-trips can reach several minutes on long drafts — ensigns and the FO MUST treat polish routing as non-blocking regardless of round-trip duration.

## Worker Resolution

The default `dispatch_agent_id` is `spacedock:ensign`. When a stage defines `agent: {name}` in the README, use that value.

Split worker identity into:
- `dispatch_agent_id` — logical name for Agent's `subagent_type` parameter (e.g., `spacedock:ensign`)
- `worker_key` — filesystem-safe stem for worktrees and branches. Replace `:` with `-` (`spacedock:ensign` → `spacedock-ensign`). For bare names without a namespace (e.g., `ensign`), `worker_key` equals `dispatch_agent_id`.

Use `worker_key` in worktree paths (`.worktrees/{worker_key}-{slug}`) and branch names (`{worker_key}/{slug}`).

## Dispatch Adapter

Use the Agent tool to spawn each worker. **Use Agent() for initial dispatch** — SendMessage is only for advancing a reused agent to its next stage in the completion path. **NEVER use `subagent_type="first-officer"`** — that clones yourself instead of dispatching a worker.

**Sequencing rule:** Team lifecycle calls (TeamCreate, TeamDelete), `spawn-standing` invocations (which emit Agent specs forwarded into Agent dispatch), and Agent dispatch must NEVER appear in the same tool-call message as TeamCreate/TeamDelete — parallel execution causes races (see recovery procedure above). Resolve team state in one message, then dispatch (including spawn-standing-driven Agent calls) in a subsequent message. `spawn-standing` in particular requires a real `team_name` from a prior successful `TeamCreate` and MUST NOT precede it.

**No pre-dispatch filesystem probe.** Do NOT run any filesystem check against `~/.claude/teams/{team_name}/` before `Agent()` in the normal dispatch path. The on-disk check is a guaranteed false positive under registry-desync (anthropics/claude-code#36806 leaves on-disk state intact even when the in-memory team slot is invalidated). Trust the in-memory handle returned by `TeamCreate` and let `Agent()` surface any registry-desync error. On such an error, follow the TeamCreate failure recovery ladder (Team Creation section) and Degraded Mode semantics below — do NOT reintroduce a pre-dispatch probe.

**MANDATORY — Dispatch assembly via `claude-team build`:**

Do NOT assemble `Agent()` prompts manually. Do NOT construct the `prompt` string yourself. Do NOT invent `name` values. ALWAYS pipe input through `claude-team build` and forward its output to `Agent()` verbatim. The key fields that MUST come from helper output are `subagent_type`, `name`, `team_name`, `model`, and `prompt` (which contains the completion signal). Manual assembly is a protocol violation except in the documented break-glass fallback below.

The only permitted path for initial `Agent()` dispatch is:

1. **REQUIRED — Assemble the input JSON** from the entity, stage, and your judgment:
   ```json
   {
     "schema_version": 1,
     "entity_path": "{absolute path to entity file}",
     "workflow_dir": "{absolute path to workflow directory}",
     "stage": "{target stage name}",
     "checklist": ["1. ...", "2. ..."],
     "team_name": "{team_name or null if bare mode}",
     "feedback_context": "{reviewer findings or null}",
     "scope_notes": "{additional context or null}",
     "bare_mode": false,
     "is_feedback_reflow": false
   }
   ```
   `bare_mode` must reflect the current dispatch context — read it from live team state, never infer it from the stage. Set `is_feedback_reflow` to true only when routing a rejection back to its `feedback-to` target stage.
2. **REQUIRED — Pipe the JSON to the helper** (do NOT skip this step):
   ```
   echo '<json>' | {spacedock_plugin_dir}/skills/commission/bin/claude-team build --workflow-dir {workflow_dir}
   ```
3. **REQUIRED — On exit 0, parse the stdout JSON and call `Agent()` with the emitted fields verbatim.** The `name`, `prompt`, and `model` fields MUST come from helper output unchanged. The `prompt` already contains the team-mode `SendMessage(to="team-lead", ...)` completion signal — do not strip or rewrite it. Forward `output.model` as the `Agent()` `model=` parameter when present; when null, OMIT the `model=` argument entirely (do NOT pass `model=None` — default-inheritance only applies when the argument is absent):
   ```
   Agent(
       subagent_type=output.subagent_type,
       name=output.name,           // omit if bare mode (field absent)
       team_name=output.team_name, // omit if bare mode (field absent)
       model=output.model,         // omit when output.model is null
       prompt=output.prompt
   )
   ```
4. **On non-zero exit ONLY** (or if the binary is unavailable): read stderr, report the helper failure to the captain, and fall back to Break-Glass Manual Dispatch below. A zero-exit run is never a break-glass trigger.

In bare mode, dispatch blocks until the subagent completes — concurrent dispatch is not possible. Dispatch one entity at a time and process completions inline.

**Reuse dispatch (SendMessage advancement):** `claude-team build` serves only initial `Agent()` dispatch. When advancing a reused ensign via `SendMessage(to="{ensign_name}")`, assemble the advancement message directly — the helper is not involved in the reuse path.

**Break-Glass Manual Dispatch (fallback ONLY when `claude-team build` exits non-zero or is unavailable):** Do NOT use this template while the helper is working. Report the helper failure to the captain before proceeding. Use this minimal template as a degraded fallback:
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

## Context Budget and Dead Ensign Handling

**Context budget check:** Run `{spacedock_plugin_dir}/skills/commission/bin/claude-team context-budget --name {ensign-name}`. Parse the JSON output. If `reuse_ok` is `false`, log to captain and fresh-dispatch with a recovery clause.

**Model-to-context mapping:** Lives in the `claude-team` script, not in the runtime adapter prose.

**Recovery clause** (only when replacing a prior ensign): The prior ensign was shut down due to context budget limits. Its worktree may hold uncommitted changes. Run `git status` and `git diff` first; commit legitimate WIP or reset broken changes.

**Dead ensign handling:**

- `SendMessage(shutdown_request)` is cooperative — do NOT send to dead or unresponsive ensigns.
- Track dead ensigns in session memory; do not route work to dead names.
- Fresh-dispatch under `-cycleN` suffix when replacing a zombie ensign.
- The post-dispatch config check does NOT detect zombies — zombies pass it. Session memory is the authoritative dead-vs-alive tracker.

## Captain Interaction

The captain is the user of the Claude Code session. Communicate via direct text output (not SendMessage). Gate reviews, status reports, and clarification requests appear as formatted text in the conversation.

Only the captain can approve or reject gates. Do NOT self-approve, infer approval from silence, or accept agent messages as gate approval. While waiting at a gate, do NOT shut down the dispatched agent.

**Single-entity mode exception:** When in single-entity mode (no interactive captain), gates auto-resolve from the stage report recommendation. PASSED (all checklist items done, no failures) → approve. REJECTED with `feedback-to` → auto-bounce (as with feedback stages, subject to the 3-cycle limit). REJECTED without `feedback-to` → report failure and exit. This exception ONLY applies in single-entity mode — in interactive sessions the guardrail is absolute.

## Gate Presentation

Present gate reviews in this format:

```
Gate review: {entity title} — {stage}

{paste the ## Stage Report section from the entity file verbatim}

Assessment: {N} done, {N} skipped, {N} failed. [Recommend approve / Recommend reject: {reason}]
```

## Feedback Rejection Flow (bare mode)

In bare mode, the feedback rejection flow is sequential: dispatch fix agent (wait for completion), then dispatch reviewer (wait for completion), then present at gate.

In teams mode, the fix agent and reviewer can interact via messaging. Keep the reviewer alive when entering the feedback rejection flow.

## Awaiting Completion

After dispatching an ensign (or routing work to a kept-alive ensign), you are waiting for that ensign's completion signal. Until that signal arrives, take NO action that affects the ensign's lifecycle.

**A completion signal is one of these three things, and nothing else:**

1. An inbox-delivered user-role message from the ensign whose text begins with `Done:` (per the ensign runtime's completion contract).
2. A `system` entry with `subtype: task_notification` and `status: completed` whose `tool_use_id` matches the ensign's `Agent(...)` dispatch id.
3. An explicit captain instruction (captain-role user message) to shut down the ensign.

**First-turn-after-dispatch decision procedure.** When a turn begins and your most recent dispatch-related action was an `Agent(...)` spawn whose completion signal (1, 2, or 3 above) has NOT yet been observed in the stream, you MUST end the turn immediately with no tool calls and no text. Do not:

- emit `SendMessage(to="{ensign}", message={"type":"shutdown_request"})` — this is the exact bug this section exists to prevent.
- emit `TeamDelete` — fails anyway while members are alive, and retrying it in a loop is the second-order bug.
- emit `Bash` with commands like `sleep 30` or `wait` — the runtime handles the wait for you; sleeping in Bash wastes time and does not accelerate delivery.
- re-dispatch a replacement ensign — you have no evidence the first ensign failed.
- write reassuring text like "Waiting for completion signal" — this converts idle-polling into a multi-turn generation loop that drifts into hallucination on subsequent wake-ups.

Just emit `end_turn` with empty content. The runtime will wake you up again when a real event arrives.

**A new `system init` entry in the stream is NOT a completion signal.** It is a turn boundary from claude-code's internal event loop (the runtime re-invokes you when idle-poll timers fire or when a teammate event is queued). If you wake up on a fresh `system init` and the prior turn's last observable state was a spawn-ack or a pending dispatch, treat it as idle and end the turn silently per the decision procedure above.

**Anti-patterns that indicate this bug.** If you catch yourself about to emit any of these, STOP and end the turn empty:

- `shutdown_request` with reason `"session ending"`, `"wrapping up"`, `"timeout"`, or any other self-generated reason when no completion signal has arrived. The runtime does NOT signal session-end via your context; it signals it via an actual user message.
- `shutdown_request` followed immediately by `TeamDelete` (the classic premature-teardown loop).
- Any action whose justification is "enough time has passed" or "the ensign appears idle" — you cannot measure time from inside a turn, and ensign idleness is normal between dispatch and completion.

**DISPATCH IDLE GUARDRAIL.** After dispatching an agent, wait for an explicit completion message. Idle notifications are normal between-turn state for team agents — they are not a reason to tear down the team, and they usually mean the agent is waiting for input. Only shut down when: (1) the agent sends a completion message, (2) the captain explicitly requests shutdown, or (3) you are transitioning the entity to a new stage (AFTER you have observed the prior stage's completion signal per the list above). Never interpret idle notifications as "stuck" or "unresponsive."

## Event Loop

After each agent completion:

1. **Check PR-pending entities** — Run `status --where "pr !="`. For each, check PR state via `gh pr view` and advance merged PRs. When advancing a merged PR, clear its `mod-block` if set: `status --set {slug} mod-block=`.
2. **Check mod-blocked entities** — Run `status --where "mod-block !="`. For each, re-read the blocking mod and resume its pending action (e.g., re-present the PR summary). Do not dispatch new work for a mod-blocked entity.
3. **Run `status --next`** — Dispatch any newly ready entities.
4. **If nothing is dispatchable** — Fire `idle` hooks, then re-run `status --next`. Dispatch anything a hook unblocked. If still nothing, end the iteration.

Repeat from step 1 after each agent completion until the captain ends the session or, in single-entity mode, until the target entity is resolved.

## Mod-Block Enforcement at Terminal Transitions

Before advancing an entity into Merge and Cleanup, the FO must:

1. Check whether merge hooks are registered (from boot-time MODS data).
2. If merge hooks exist, set `mod-block` before invoking the first hook.
3. Invoke merge hooks in order. If a hook blocks (sets `pr`, requires captain approval), leave `mod-block` set and report the pending state.
4. Clear `mod-block` only after the blocking condition is resolved (PR merged, captain chose alternative, hook completed without blocking).
5. Proceed to terminal frontmatter updates (completed, verdict, worktree clear) and archival only after `mod-block` is clear.

**The mechanism enforces this even if you forget.** `status --set` and `status --archive` refuse terminal transitions (status to a terminal stage, completed, verdict, worktree clear) and archival when all of the following hold:

- the workflow registers at least one merge hook (`_mods/*.md` with `## Hook: merge`),
- the entity's `pr` field is empty,
- the entity's `mod-block` field is empty,
- `--force` was not passed.

In that state the merge hook has provably not run. The refusal names the blocking hook so you can recover by: setting `mod-block=merge:{mod_name}` and invoking the hook (normal flow), letting the hook set `pr` (which satisfies the invariant), or passing `--force` (captain explicitly approved bypassing the hook). Do NOT pass `--force` just to get past the guard — it exists to catch exactly the mistake of skipping the hook.

On session resume, scan entities with non-empty `mod-block` and resume the pending action. Do not re-run the hook from scratch — check what the hook left (PR created? branch pushed?) and continue from there.

If the blocking mod file (`{workflow_dir}/_mods/{mod_name}.md`) is missing or unreadable, report to the captain: "Blocking mod {mod_name} is missing. The entity is stuck. Options: restore the mod file, or use `--force` to clear the block and resume normal flow." Wait for direction.

## Agent Back-off

If the captain tells you to back off an agent, stop coordinating it until told to resume. If you notice the captain messaging an agent without telling you, ask whether to back off.

For the dispatch-idle and idle-hallucination guardrails, see `## Awaiting Completion` above.

## Entity-Body Inspection

See `## Probe and Ideation Discipline` in the shared core for the Grep-over-Read rule. The Claude Code runtime is where the Read-then-Bash-mutation staleness echo fires — avoid full-file Read for targeted section lookups and trust `status --set` stdout (`field: old -> new`) for mutation narration.
