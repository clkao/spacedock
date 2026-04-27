---
id: 153
title: "Codex FO: completion notifications must preempt side discussion"
status: implementation
source: "CL direction during 2026-04-14 session after task 148 validation completion handling"
started: 2026-04-15T04:11:21Z
completed:
verdict:
score: 0.69
worktree: .worktrees/spacedock-ensign-codex-completion-notifications-must-preempt-side-discussion
issue:
pr:
blocked-on: https://github.com/openai/codex/issues/15723
blocked-reason: Upstream Codex does not wake the parent turn on subagent completion; task 216 landed the local preemptible-wait mitigation instead.
---

Follow-up to task 140.

Task 140 established the law: in interactive Codex mode, a completed gated or critical-path stage becomes the next required action before unrelated orchestration continues. The remaining gap is narrower: the real async completion-notification surface still allows the first officer to drift into unrelated side discussion before consuming the completion.

That is what reappeared during task 148's validation completion handling. The validator finished in the background, Codex surfaced the completion, and the FO still answered unrelated conversation before reconciling the validation result into the entity's true next action. This task should close that runtime-ordering gap, not restate the law again.

## Problem Statement

Task 138 covered pre-completion wait policy: in interactive Codex mode, workers may run in the background until the next orchestration step is blocked on their result. Task 140 covered post-completion ergonomics and contract wording: once the result exists, gated and critical-path completions should become the next required action.

This task is the remaining implementation gap between those two:

- the worker is already complete
- the completion arrives asynchronously in an interactive Codex session
- the FO receives a real completion event for an active entity
- the FO must consume that event before replying to unrelated side discussion

The relevant surface is the actual Codex async completion signal for a spawned worker, whether it appears as `<subagent_notification>` or the equivalent completion event/tool transcript item. A bounded `wait_agent(...)` path is not enough proof here because the regression is specifically about what happens when completion arrives while the captain is still talking about something else.

Current behavior is effectively "completion exists, but the next captain turn still flows through ordinary conversation routing." Desired behavior is stricter, but still narrow: once Codex surfaces a real completion notification for an active entity on the task-153 surface, that completion becomes a higher-priority pending action than unrelated side discussion until it is reconciled into the entity's next real workflow move.

## Scope Boundary

- In scope: interactive Codex completion ordering after a background worker finishes, but only for completions that already imply an immediate next required action under task 140.
- In scope: how the FO consumes async completion into gate presentation, advancement, reroute, or explicit blocked/terminal reporting.
- In scope: a narrow Codex interactive test surface that can prove ordering against real side discussion.
- In scope: Codex-specific operator/runtime behavior for the task-153 completion surfaces only:
  - active gated-stage completions
  - active non-gated completions on the entity's current critical path
  - active `validation` completions that return `REJECTED` and define `feedback-to`
- Out of scope: changing task 138's background-wait policy before completion.
- Out of scope: rewriting shared first-officer scheduling semantics across Claude and Codex.
- Out of scope: proving the fix only by prompt text, runtime prose, or invocation-prompt coaching.
- Out of scope: widening this into a shared-core rewrite or generic "conversation discipline" project.
- Out of scope: unrelated background notifications, non-active entities, informational completions with no immediate next required action, or a blanket rule for every Codex notification.

## Desired Behavior Change

This task needs the behavior change spelled out, not just the test names.

- Current operator-facing behavior: a worker can finish, Codex can surface the completion, and the next captain turn can still be answered as ordinary side discussion.
- Desired operator-facing behavior: the FO treats that completion as the next piece of business and visibly handles it before answering unrelated conversation.

- Current runtime-facing behavior: the completion notification is effectively passive context.
- Desired runtime-facing behavior: a task-153 completion notification for an active entity sets a higher-priority pending-completion obligation in the interactive Codex loop. While that obligation exists, unrelated captain input is deferred until the completion has been consumed into a concrete workflow action.

Concrete examples:

1. Gated stage example:
   - Current: `153-ideation/Ensign` finishes, Codex shows the completion, the captain asks an unrelated question, and the FO answers that question first.
   - Desired: the FO first surfaces the ideation gate review / waiting-for-approval state for task 153, then returns to the unrelated question.
2. Non-gated critical-path example:
   - Current: implementation finishes, but the FO can answer a side question before advancing to validation or reporting a blocker.
   - Desired: the FO first advances/redispatches the next stage or reports the blocker, then resumes side discussion.
3. Validation rejection example:
   - Current: validation completes with `REJECTED`, but the captain can pull the FO into side discussion before the `feedback-to` bounce happens.
   - Desired: the FO first announces the rejection and routes follow-up to the `feedback-to` target, then resumes side discussion.

This is specifically a Codex interactive ordering rule. It is not a shared-core rewrite and it is not prompt coaching.

## Approach Options

### Recommended: add a narrow Codex interactive completion-ordering test surface

Keep the implementation local to the Codex runtime contract and add one focused interactive Codex harness path that can observe:

1. a worker dispatched into the background,
2. an async completion notification,
3. an immediate unrelated captain message injected after that notification while the completion obligation is still observably pending and unconsumed,
4. the FO choosing completion handling before side discussion.

This is the smallest approach that proves the real bug. It likely means extending the existing PTY test utilities for Codex or adding a sibling Codex interactive helper, then writing focused live E2Es.

### Alternative: rely on `codex exec --json` non-interactive logs

This is cheaper but insufficient. The existing `run_codex_first_officer()` and `CodexLogParser` path can prove bounded completion, `wait_agent(...)`, reuse, and rejection ordering in non-interactive runs. It cannot prove that an async completion preempts a later unrelated captain message because there is no side-discussion turn to interrupt.

### Alternative: push the whole rule into shared core

This broadens the task without evidence that the bug is shared. The observed gap is on the Codex interactive notification surface, so the safer design is to harden that runtime first and leave shared-core generalization to a later task only if another runtime exhibits the same failure.

## Required Behavior

In interactive Codex mode, any completion notification on the task-153 surface becomes an interrupt-worthy dispatch event. The FO must consume that event before continuing unrelated side discussion.

Concrete outcomes:

1. If the completed stage is gated, the FO immediately reads the stage report or completion evidence and presents the gate review as the next required action.
2. If the completed stage is non-gated but on the entity's critical path, the FO advances the entity, redispatches the next stage, reports terminal handling, or reports the concrete blocked state before returning to unrelated discussion.
3. If the completed stage is `validation`, the report recommends `REJECTED`, and the stage defines `feedback-to`, the FO immediately routes the rejection to the `feedback-to` target instead of letting the result sit behind side discussion.
4. Side discussion may resume only after the completion event has been consumed into one of:
   - gate presentation / waiting-for-approval
   - routed `feedback-to` follow-up
   - next-stage advancement or redispatch
   - terminal merge / PR-pending handling
   - explicit blocked-state reporting
5. This ordering rule applies to the real async completion-notification surface, not just to synchronous `wait_agent(...)` paths that already foreground completion by construction.

## Interactive Harness Design

The PTY proof needs an explicit synchronization point. Sleeping for "long enough" after dispatch is race-prone and does not prove that side discussion was injected after a real completion notification while the completion obligation was still pending.

### Deterministic pending-obligation barrier

For each interactive Codex E2E, the harness should use this sequence:

1. Dispatch the background worker and let the session continue normally.
2. Wait until raw PTY output contains an observable completion notification for the target worker/entity/stage.
   - Acceptable markers: a literal `<subagent_notification>` block or a stable Codex transcript/tool event that unambiguously means "this worker finished."
   - The marker must be visible in captured PTY output, not inferred from elapsed time.
3. Start a pending-obligation window at the byte/character offset where that notification finishes.
4. Before injecting side discussion, confirm that no completion-consumption marker for the same entity/stage has appeared after the notification offset.
   - For a gated path, consumption markers are gate review / waiting-for-approval handling.
   - For a non-gated critical-path path, consumption markers are advancement, redispatch, terminal handling, or explicit blocked-state reporting.
   - For a `validation` `REJECTED` + `feedback-to` path, consumption markers are rejection findings plus explicit reroute/follow-up activation.
5. Inject the unrelated side-discussion message at the first observable prompt-ready or otherwise input-accepting point after the notification while that pending-obligation window is still open.
6. Persist ordering evidence showing:
   - notification offset < injection offset
   - no path-specific consumption marker exists between notification offset and injection offset
   - first consumption marker offset > injection offset

The test must not inject side discussion before the completion notification is visible, and it must not rely on fixed sleeps after dispatch. The barrier is not just "notification seen"; it is "notification seen and still unconsumed at injection time."

If the harness cannot find an observable input-accepting point between notification and first consumption marker, that run does not prove preemption against side discussion. It either belongs in the feasibility/fallback path or needs a more observable harness surface.

### Why this barrier matters

The bug is specifically "completion was already surfaced, then side discussion happened first." The test therefore needs to place the unrelated message after a real notification and before reconciliation, not merely after enough time has probably passed or after the FO has already consumed the result.

## Completion Consumption Oracles

The task needs explicit success oracles for when a completion counts as consumed. "The FO seemed to deal with it" is not sufficient.

### 1. Gated completion consumed

Evidence required:

- the transcript shows gate handling for the completed entity/stage: gate review, stage report verdict/evidence, or an explicit waiting-for-approval state
- the unrelated side-discussion answer does not appear before that gate-handling output
- the entity does not silently advance past the gate; it remains at the gated stage or an equivalent waiting-for-approval state

What does not count:

- merely noticing the notification
- merely restating that something completed
- answering the side discussion first and mentioning the gate later

### 2. Non-gated critical-path completion consumed

Evidence required:

- the transcript shows one concrete critical-path outcome before any unrelated reply:
  - next-stage advancement
  - redispatch of the next stage
  - terminal completion handling
  - or an explicit blocked-state report naming the blocker
- that outcome is corroborated by runtime evidence such as updated entity state, next-stage dispatch text, or explicit blocked wording tied to the same entity

What does not count:

- a vague summary like "implementation is done" with no advancement/blocking action
- generic side-discussion text that happens to mention the completed task

### 3. `validation` `REJECTED` + `feedback-to` consumed

Evidence required:

- the transcript shows the rejection verdict/findings
- the transcript then shows explicit routing to the `feedback-to` target stage before any unrelated reply
- the reroute is corroborated by concrete follow-up evidence: target-stage activation, fresh dispatch, or same-handle reuse/send-input against the `feedback-to` target

What does not count:

- stopping at gate presentation for the rejected validation stage
- mentioning that the captain should probably send it back later
- leaving the rejection visible but unrouted while answering the unrelated message

## Feasibility Checkpoint And Fallback

The Codex interactive harness is the highest-risk part of this task. Before committing to full E2E coverage, implementation should prove that the completion-notification surface is stably detectable.

### Phase 0 checkpoint

Add a narrow probe or helper that answers one question: can the PTY harness reliably observe a worker-completion marker and the prompt-ready boundary after that marker in a Codex interactive session?

Success for the checkpoint:

- raw PTY output exposes a repeatable completion marker for the target worker/entity/stage
- the harness can distinguish "notification appeared" from "completion already consumed"
- the harness can identify an input-accepting point while the completion obligation is still pending
- the harness can inject the next captain message at a deterministic point after notification and before first consumption evidence

### Fallback if the PTY surface is not stable enough

Do not pretend the full preemption E2E is proven. Instead:

1. land a smaller Codex-specific observability slice first
2. expose or harvest the notification signal through a stable Codex-side helper/log surface
3. keep the runtime behavior change scoped to Codex interactive handling
4. defer the final side-discussion-ordering E2E to a follow-up once the signal is observable

Acceptable fallback work is harness observability or notification-surface stabilization. Unacceptable fallback is widening into shared-core scheduler changes or replacing the proof with prompt wording.

## Likely Implementation Surfaces

Keep the branch local to Codex interactive completion handling:

- `skills/first-officer/references/codex-first-officer-runtime.md`
  - tighten the wording around async completion notifications so it explicitly governs the interactive completion-event path, not just bounded waits
  - describe the pending-completion priority rule in operator-facing terms
- Codex interactive harness support
  - extend [scripts/test_lib_interactive.py](/Users/clkao/git/spacedock/scripts/test_lib_interactive.py) for Codex or add a narrow Codex sibling helper
  - add helper methods for "wait for completion notification marker", "detect pending-obligation window", and "inject before first consumption marker"
  - current harness coverage is Claude-only for PTY interaction and non-interactive for Codex
- New focused Codex interactive E2E(s)
  - one gated completion preemption test
  - one non-gated critical-path preemption test
  - one rejection-flow preemption test, or a single carefully-scoped test that proves both notification consumption and immediate reroute
- Supporting guardrails
  - [tests/README.md](/Users/clkao/git/spacedock/tests/README.md) if the new Codex interactive harness needs usage guidance
  - [tests/test_agent_content.py](/Users/clkao/git/spacedock/tests/test_agent_content.py) for lightweight contract guardrails only

Avoid widening into `skills/first-officer/references/first-officer-shared-core.md` unless a tiny metadata exposure is unavoidable. The task is not a shared-core scheduler rewrite.

## Acceptance Criteria

1. In interactive Codex mode, a real completion notification for an active gated stage becomes the next required action before unrelated side discussion is answered.
   - Test method: live Codex PTY E2E with a gated fixture.
   - Required synchronization: wait for the observable completion-notification marker, verify no gate-handling consumption marker has yet appeared for that entity/stage, then inject unrelated side discussion at the first input-accepting point while the completion obligation is still pending.
   - Required ordering proof: notification offset < injection offset < first gate-handling consumption offset.
   - Success oracle: transcript shows gate review / waiting-for-approval output for the completed entity before any unrelated answer, and the entity remains at the gated stage or equivalent waiting state.
2. In interactive Codex mode, a real completion notification for a non-gated critical-path stage is consumed into advancement, redispatch, terminal handling, or explicit blocked-state reporting before unrelated side discussion is answered.
   - Test method: live Codex PTY E2E with a non-gated multi-stage fixture.
   - Required synchronization: wait for the observable completion-notification marker, verify no advancement/redispatch/terminal/blocked consumption marker has yet appeared for that entity/stage, then inject unrelated side discussion at the first input-accepting point while the completion obligation is still pending.
   - Required ordering proof: notification offset < injection offset < first advancement/blocked consumption offset.
   - Success oracle: transcript shows the concrete critical-path outcome before any unrelated answer, corroborated by dispatch/state evidence tied to that entity.
3. In interactive Codex mode, when `validation` completes with `REJECTED` and `feedback-to`, the completion notification is consumed only when the rejection is surfaced and routed to the `feedback-to` target before unrelated side discussion resumes.
   - Test method: live Codex PTY E2E using the rejection-flow fixture.
   - Required synchronization: wait for the observable completion-notification marker, verify no rejection+routing consumption marker has yet appeared for that entity/stage, then inject unrelated side discussion at the first input-accepting point while the completion obligation is still pending.
   - Required ordering proof: notification offset < injection offset < first rejection+routing consumption offset.
   - Success oracle: transcript shows rejection findings plus explicit reroute/follow-up activation before any unrelated answer. Gate display alone does not satisfy this AC.
4. The implementation includes a feasibility checkpoint for Codex PTY observability and does not overclaim if the notification surface is not stably detectable.
   - Test method: branch review confirms either:
     - the harness exposes a stable pending-obligation barrier and the interactive E2Es use it, or
     - the task is intentionally split so harness observability lands first and the full side-discussion-ordering proof is deferred explicitly
5. The implementation targets the real Codex interactive completion-notification surface rather than re-proving bounded `wait_agent(...)` behavior or injecting behavioral coaching into the invocation prompt.
   - Test method: branch review plus focused static checks confirm the proof surface is a live interactive Codex path, the invocation prompt stays minimal, and static assertions remain limited to runtime/harness guardrails.
6. The task remains a focused follow-up to 140 and does not reopen 138 or broaden into shared-core scheduling changes.
   - Test method: diff review confirms touched files stay inside Codex interactive runtime guidance, a Codex-specific interactive harness/observability slice, and focused Codex tests.

## Test Plan

- Phase 0 feasibility probe: prove that Codex PTY output exposes a stable completion-notification marker and a prompt-ready boundary after that marker.
  - Cost: low/medium
  - Reason: de-risk the expensive E2E path before building tests on a race-prone assumption
- Required live E2E: gated completion preempts side discussion in an interactive Codex session.
  - Cost: medium/high
  - Reason: this is the direct reproduction of the observed bug and cannot be proven by `codex exec --json`
- Required live E2E: validation `REJECTED` + `feedback-to` completion preempts side discussion and routes the follow-up immediately.
  - Cost: high
  - Reason: this is the highest-risk follow-on path because it combines async completion ordering with existing rejection-routing semantics
- Required live E2E: non-gated critical-path completion preempts side discussion with advancement or blocked-state reporting.
  - Cost: medium/high
  - Reason: this is part of the task-140 law being enforced here and keeps the fix from overfitting gated-only behavior
- Supporting static checks: keep the Codex runtime wording and prompt-discipline guardrails aligned with the live behavior under test.
  - Cost: low
  - Reason: useful regression net, but explicitly not the proof surface
- Existing non-interactive Codex E2Es such as [tests/test_gate_guardrail.py](/Users/clkao/git/spacedock/tests/test_gate_guardrail.py) and [tests/test_rejection_flow.py](/Users/clkao/git/spacedock/tests/test_rejection_flow.py) remain valuable regression coverage for bounded paths, but they do not satisfy this task's primary proof requirement because they do not model side discussion after a visible background completion notification.

E2E need: yes. A real interactive Codex run is required for acceptance because the bug is about async completion ordering against unrelated captain messages.

## Stage Report: ideation

- DONE - Clarified the scope boundary relative to task 138 and task 140.
  - 138 owns pre-completion background wait policy.
  - 140 owns the contract that completed gated/critical-path work becomes the next required action.
  - This task owns the remaining async completion-notification ordering gap in interactive Codex mode.
- DONE - Spelled out the desired behavior change from current state to desired state, including operator-facing and runtime-facing examples.
- DONE - Specified the concrete behavior for gated completions, non-gated critical-path completions, and `validation` + `feedback-to` rejection handling.
- DONE - Tightened the Codex PTY synchronization design into a pending-obligation barrier: inject side discussion only after notification is visible and before any path-specific consumption marker has appeared.
- DONE - Tightened the success oracles so "completion consumed" now means concrete gate handling, advancement/blocked-state handling, or rejection reroute with follow-up evidence.
- DONE - Added a feasibility checkpoint and explicit fallback plan in case the Codex PTY notification surface is not stably detectable.
- DONE - Resolved the AC/Test Plan mismatch by making the non-gated critical-path interactive proof required in both places.
- DONE - Identified the narrow implementation surfaces: Codex first-officer runtime guidance, a Codex interactive PTY harness/observability surface, focused interactive Codex E2Es, and lightweight static guardrails.
- DONE - Reframed acceptance criteria around real completion-notification paths instead of doc-text tautologies or bounded `wait_agent(...)` flows.
- DONE - Produced a proportional test plan with a Phase 0 observability probe plus required live Codex interactive E2Es, and explicitly called out why non-interactive Codex coverage is not sufficient.
- DONE - Tightened scope so the preemption rule is explicitly limited to the task-153 completion surfaces, not every Codex notification.
- DONE - Updated only the entity body on main and appended this ideation report.

### Summary

The task is now scoped as a Codex-interactive follow-up to task 140: harden the real async completion-notification path so task-153 completion surfaces preempt side discussion until they are consumed into the entity's real next action. The doc now defines a pending-obligation PTY barrier with ordering proof, concrete consumption oracles, aligned required E2Es, and a fallback plan if Codex interactive observability is not yet stable enough for the full proof.

### Feedback Cycles

- Cycle 1 (2026-04-15): Validation REJECTED. Static Codex/runtime guardrails passed, but the required live interactive Codex completion-preemption proof did not complete in this environment: the PTY probe timed out before a prompt-ready boundary was observed, so AC1-AC3 remain unverified. Route back to implementation to either obtain true terminal-backed Codex evidence for the live probe or tighten the readiness/observability path so the task no longer stalls at the prompt-ready boundary.
- Cycle 2 (2026-04-15): Validation REJECTED again. The bounded fallback is now cleaner: the live Codex PTY probe skips quickly instead of hanging, so AC4-AC6 remain supported without overclaiming. But the real interactive completion-obligation boundary is still unproven here, so AC1-AC3 remain unverified. Route back to implementation to either obtain true terminal-backed Codex evidence for the live probe or decide whether the task should be explicitly narrowed to the fallback observability slice instead of the full live proof.
- Cycle 3 (2026-04-15): Validation REJECTED again. The branch now consistently fails closed into the bounded fallback (`43 passed, 1 skipped` for the focused contract surface), but AC1-AC3 are still not proven because this environment never exposes the real terminal-backed Codex completion/prompt boundary. Upstream reference: [openai/codex#15723](https://github.com/openai/codex/issues/15723), "Background subprocesses/subagents do not wake the calling agent on completion" (opened 2026-03-25), matches this session's observation that a subagent completion notification did not autonomously wake the parent Codex turn. Feedback cycle limit reached: escalate to the captain instead of dispatching another implementation round.
