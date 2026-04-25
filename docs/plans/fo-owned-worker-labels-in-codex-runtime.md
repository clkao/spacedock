---
id: 137
title: FO-owned worker labels in Codex runtime
status: ideation
source: FO observation during task 136 dispatch on 2026-04-12
score: 0.58
started: 2026-04-15T03:21:24Z
completed:
verdict:
worktree:
issue:
pr:
---

## Problem Statement

Codex collaborator mode can return incidental worker nicknames such as `Leibniz` from `spawn_agent`. Those names are useful only as platform metadata, but they can leak into first-officer narration as if they were the workflow identity of the worker. That makes the captain correlate at least four separate identifiers by hand: the logical packaged worker id (`spacedock:ensign`), the filesystem-safe worker key (`spacedock-ensign`), the Codex runtime handle, and the incidental nickname Codex happened to choose.

The Codex first-officer runtime already points toward human-readable worker labels, but the contract and tests need to make the ownership boundary unambiguous. Operator-facing dispatch, reuse, wait, and shutdown messages should lead with a deterministic FO-owned worker label. Runtime handles and logical ids remain visible as secondary metadata; incidental Codex nicknames must never be the leading identity a captain uses to follow the workflow.

## Scope Boundary

In scope:

- `skills/first-officer/references/codex-first-officer-runtime.md` contract wording for Codex worker labels.
- Static assertions in `tests/test_agent_content.py` or a similarly small static test surface.
- Live Codex log assertions in `tests/test_codex_packaged_agent_e2e.py` or the narrowest existing Codex E2E that exercises fresh dispatch, reuse through `send_input`, `wait_agent`, and explicit shutdown.
- Test helper parsing only when needed to inspect operator-facing Codex log lines and correlate labels with runtime handles.

Out of scope:

- Claude Code team naming, `claude-team` member names, and `SendMessage` addressing.
- Changing `dispatch_agent_id`, `worker_key`, worktree path, or branch naming semantics.
- Depending on a specific Codex nickname or trying to suppress Codex's own native UI labels.
- Redesigning reuse, feedback routing, or shutdown behavior beyond the label shown on those surfaces.

## Proposed Approach

Define the identity model explicitly in the Codex runtime contract:

- `dispatch_agent_id`: the logical packaged worker id, for example `spacedock:ensign`; preserved in worker assignments and summaries.
- `worker_key`: the filesystem-safe key, for example `spacedock-ensign`; used for worktree paths, branch names, and other machine names.
- `runtime_handle`: the Codex thread or item handle returned by `spawn_agent`; used for `send_input`, `wait_agent`, and `close_agent`.
- `codex_nickname`: any display nickname Codex returns, for example `Leibniz`; treated as optional platform metadata only.
- `worker_label`: the FO-owned operator label, formed before `spawn_agent` as `{entity_id}-{stage_key}/{display_name}`.

The label must be deterministic from workflow-owned inputs. `entity_id` comes from entity frontmatter, `stage_key` is the normalized stage name, and `display_name` comes from an FO-controlled source such as the role display name (`Ensign`) or a deterministic roster keyed by `{entity_id}:{stage_key}:{worker_key}`. If a roster is used, its choice must happen before dispatch and must not read Codex's returned nickname. For this entity, a valid label shape would be `137-ideation/Ensign` or `137-ideation/Herschel`; `Leibniz` is not valid unless the FO-owned deterministic source selected it before Codex returned it.

Tighten the Codex runtime wording from the current loose pattern:

```text
Report that label alongside the logical id or thread handle; do not rely on opaque agent ids or incidental nicknames alone.
```

to target wording with an explicit leading-position rule:

```text
Before calling spawn_agent, assign worker_label = {entity_id}-{stage_key}/{display_name} from workflow-owned inputs. Every operator-facing Codex update about dispatching, reusing, waiting on, or shutting down that worker MUST start with worker_label. Put logical ids and runtime handles after the label, for example `137-ideation/Ensign` (spacedock:ensign, handle: item_23). If Codex returns a nickname, keep it as optional metadata only; never use it as the leading worker identity and never derive worker_label from it.
```

Then update the examples so every Codex operator-facing surface leads with the label:

- Fresh dispatch before or around `spawn_agent`: ``137-ideation/Ensign` dispatching (spacedock:ensign) on main.`
- Handle binding after spawn: ``137-ideation/Ensign` is active on handle item_23.`
- Reuse/routing through `send_input`: ``137-implementation/Ensign` routing follow-up on existing handle item_23.`
- Active-again status after reuse: ``137-implementation/Ensign` is active again on handle item_23; the routed follow-up is this entity's critical path.`
- Critical-path wait: ``137-implementation/Ensign` waiting on handle item_23 for the routed follow-up completion.`
- Shutdown: ``137-validation/Ensign` shutting down handle item_32; no later routing remains.`
- Bounded final or gate summaries that mention a worker: ``137-validation/Ensign` reported PASSED; gate review follows.`

The implementation should avoid tests that simply ban one known nickname. Instead, tests should assert the label grammar, its leading position, and its stability across handle-bound lifecycle messages. When a Codex log exposes returned nicknames, the test should derive those nicknames from the log and assert they are not used as the leading identity. When the log does not expose nickname metadata, a static or parser-level fixture should use arbitrary sample nicknames to prove the matcher is not tied to a specific name.

## Acceptance criteria

**AC-1 - The Codex runtime contract separates all worker identity concepts.** The finished runtime contract defines `dispatch_agent_id`, `worker_key`, `runtime_handle`, `codex_nickname`, and `worker_label` as distinct concepts, and states that only `worker_label` leads operator-facing lifecycle narration.

Test: a static content check in `tests/test_agent_content.py` or equivalent reads `skills/first-officer/references/codex-first-officer-runtime.md` or the assembled Codex first-officer contract and asserts all five concepts are present near the label convention, including text that says nicknames are metadata only.

**AC-2 - The FO-owned label convention is deterministic and independent of Codex-returned nicknames.** The finished contract specifies `{entity_id}-{stage_key}/{display_name}`, explains where each component comes from, and says the label is assigned before `spawn_agent`.

Test: a static content check asserts the target convention, the pre-`spawn_agent` assignment requirement, and a prohibition on deriving the label from Codex's returned nickname. The check must not assert one fixed display name such as `Herschel`.

**AC-3 - Every Codex operator-facing dispatch/reuse/wait/shutdown surface leads with the label.** Fresh dispatch, post-spawn handle binding, reuse routing, active-again reuse status, critical-path wait status, and explicit shutdown status all start with `worker_label` when they appear in operator-facing text.

Test: extend the relevant Codex E2E log assertion to filter `agent_message_texts()` for lifecycle lines containing dispatch, active, routing, reuse, wait, or shutdown language. Each matched line must satisfy a leading label regex such as ``^`?\d+[-_][a-z0-9._-]+/[A-Za-z0-9._-]+`?`` and must include the logical id or runtime handle after the label where that metadata is known.

**AC-4 - The same runtime handle is narrated with a stable FO-owned label across its lifecycle.** A worker handle introduced at dispatch keeps the same `worker_label` in later reuse, wait, and shutdown messages for that stage assignment.

Test: in the live Codex packaged-agent E2E, map each `spawn_agent`, `send_input`, `wait_agent`, and `close_agent` receiver handle to the label observed in surrounding operator-facing messages. The test passes only when later messages for the same handle reuse the same label; it fails if the leading identity switches to a Codex nickname or a different label for the same stage assignment.

**AC-5 - Nickname leakage regressions are caught without overfitting to incidental Codex names.** Tests fail when any returned Codex nickname is used as the leading identity, but they do not encode `Leibniz` or any other one-off nickname as the only forbidden value.

Test: add either a parser-level unit test with synthetic Codex log entries that include two arbitrary nickname values, or a live-log assertion that derives nickname candidates from the actual log metadata when available. The assertion checks "returned nickname is not the leading lifecycle identity" rather than "the string `Leibniz` is absent."

**AC-6 - Existing packaged-worker machine identity remains intact.** Worktree paths and branch names continue to use `worker_key`, worker prompts continue to preserve `dispatch_agent_id: spacedock:ensign`, and neither the FO-owned label nor Codex nickname leaks into those machine names.

Test: keep or extend the existing packaged-agent E2E assertions that `spacedock-ensign` appears in worktree paths and branch names while `spacedock:ensign` does not. Add a static or live assertion that the worker prompt still contains `dispatch_agent_id: spacedock:ensign` separately from the label convention.

## Test Plan

Static tests are low cost and should run in the normal offline suite. They should pin the contract wording in the Codex runtime adapter and assembled first-officer content so future prompt edits cannot collapse `worker_label`, `worker_key`, handle, and nickname back together.

Parser-level unit tests are low to moderate cost and are useful if the live Codex log shape is awkward. A synthetic JSONL fixture can prove the lifecycle-line matcher rejects leading nicknames without depending on whichever nickname Codex chooses in a live run.

Live Codex E2E coverage is required because the bug is operator-facing runtime behavior. The existing packaged-agent E2E is the best fit because it already exercises fresh dispatch, routed reuse through `send_input`, `wait_agent`, and explicit shutdown on real Codex collaborator handles. Expected cost is high compared with static tests, so the live check should be narrow: label-leading/stability assertions only, not a broader rewrite of the reuse test.

No new Claude E2E is needed. Claude team member names are a different runtime surface, and this task is specifically about the Codex `spawn_agent` nickname/handle path.

## Stage Report: ideation

- DONE: A deterministic FO-owned label convention is specified separately from logical worker id, runtime handle, and incidental nickname.
  The body defines `worker_label = {entity_id}-{stage_key}/{display_name}` and separates it from `dispatch_agent_id`, `worker_key`, `runtime_handle`, and `codex_nickname`.
- DONE: The proposed approach identifies every Codex operator-facing dispatch/reuse/wait/shutdown surface that must lead with the label.
  The surface inventory covers fresh dispatch, handle binding, reuse routing, active-again status, critical-path wait, shutdown, and worker-bearing bounded/gate summaries.
- DONE: Acceptance criteria and test plan are concrete enough to catch nickname leakage regressions without overfitting to incidental Codex names.
  AC-3 through AC-5 require leading-label, stable-handle, and nickname-derived-not-hardcoded checks; the test plan splits static, parser-level, and live Codex coverage.

### Summary

This ideation pass turns the seed into a bounded Codex-runtime labeling task. The proposed end state is a deterministic FO-owned label that leads every operator-facing worker lifecycle message, while logical ids, worker keys, runtime handles, and Codex nicknames remain distinct secondary concepts.
