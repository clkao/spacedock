# Pi First Officer Runtime

This file defines how the shared first-officer core executes on Pi.

## Runtime Shape

Pi is a first-class runtime target. For the first slice, treat Pi's built-in session identity as the canonical worker handle. Spacedock should layer only a thin worker-label -> Pi-session mapping on top, rather than assuming Claude-style teams or Codex-style native collaborator handles.

The Pi path must support:
- fresh worker dispatch
- wait for completion
- same-worker routed follow-up / reuse
- explicit shutdown
- interactive and non-interactive execution
- worktree-backed stages

Standing teammates are out of scope for the first Pi slice.

## Worker Model

Use one Pi session per worker assignment. Pi session id or session path is the canonical worker handle. Persist only the minimum extra metadata needed to reopen the same worker for routed reuse:
- FO-owned worker label
- logical dispatch id
- worker key
- Pi session id or session path
- cwd / worktree path
- entity slug and stage
- active / completed / shutdown state
- a completion epoch or equivalent marker so reused follow-up completions are distinguishable from stale prior completions

Do not build a second session system on top of Pi. The sidecar exists only to answer workflow questions such as "which Pi session currently backs `218-implementation/Ensign`?".

## Dispatch Adapter

The first-officer dispatch prompt is authoritative for Pi workers, but the FO must not improvise its own Pi worker lifecycle logic in-model. The authoritative Pi dispatch/reuse/shutdown mechanism is the repo-local helper pair:

- `{spacedock_plugin_dir}/scripts/pi_worker_runtime.py`
- `{spacedock_plugin_dir}/scripts/pi_session_registry.py`

Use one repo-local registry file for the FO session, conventionally `{repo_root}/.pi-fo-worker-registry.json`, plus a repo-local Pi session dir such as `{repo_root}/.pi-worker-sessions/`.

Fresh dispatch must go through the runtime helper's blocking `dispatch` path:

```bash
python3 {spacedock_plugin_dir}/scripts/pi_worker_runtime.py dispatch \
  --registry {repo_root}/.pi-fo-worker-registry.json \
  --session-dir {repo_root}/.pi-worker-sessions \
  --worker-label {worker_label} \
  --prompt-file {prompt_file} \
  --cwd {worker_cwd} \
  --entity-slug {entity_slug} \
  --stage-name {stage_name} \
  --skill-path {spacedock_plugin_dir}/skills/ensign \
  --no-context-files
```

Routed reuse must go through the helper's blocking `reuse` path against the existing registry record:

```bash
python3 {spacedock_plugin_dir}/scripts/pi_worker_runtime.py reuse \
  --registry {repo_root}/.pi-fo-worker-registry.json \
  --session-dir {repo_root}/.pi-worker-sessions \
  --worker-label {worker_label} \
  --prompt-file {prompt_file} \
  --stage-name {next_stage_name} \
  --skill-path {spacedock_plugin_dir}/skills/ensign \
  --no-context-files
```

Shutdown must go through the same helper so any live background Pi subprocess tracked by the runtime is actually terminated before the worker is marked unroutable:

```bash
python3 {spacedock_plugin_dir}/scripts/pi_worker_runtime.py shutdown \
  --registry {repo_root}/.pi-fo-worker-registry.json \
  --session-dir {repo_root}/.pi-worker-sessions \
  --worker-label {worker_label}
```

Behavioral requirements of this helper path:
1. fresh dispatch creates or opens a Pi worker session, sends a fully self-contained assignment, and waits for completion before entity advancement
2. routed reuse reopens the same Pi worker session, marks the worker active again, waits for the new completion, and rejects stale pre-reuse completion evidence
3. active workers are not routable for a second overlapping reuse turn
4. reopen uses the recorded Pi session id or, when needed, the stored session file path
5. shutdown terminates any live/background Pi subprocess still owned by the runtime before the record becomes `shutdown`

For the first Pi slice, reopened same-session follow-up is the accepted reuse model. Do not require a continuously live background worker just to satisfy reuse semantics, but do not claim a worker is shut down while a runtime-owned background Pi process is still running.

Do not treat the existence of an older completed Pi session turn as proof that the follow-up routed work has completed.

## Working Directory

The FO stays anchored at the repo root.

When a stage is worktree-backed, the Pi worker runs in the assigned worktree and uses the worktree copy of the entity file. When a stage is not worktree-backed, the Pi worker stays on the main branch copy.

## Output Discipline

For the first Pi slice, report operator-facing progress in the same high-level lifecycle as other runtimes:
- dispatching worker label
- waiting on worker label
- routed follow-up on existing worker label
- shutting down worker label when no later routing remains

The stage report in the entity file remains the source of truth for completion and gate review.

## Optimization Path

Once reopened-session reuse is proven behaviorally, a preferred optimization path is to move reusable Pi workers to SDK-managed keep-alive sessions inside the FO runtime. That should reduce repeated process/session startup overhead without changing the FO-visible contract.
