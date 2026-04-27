---
id: 216
title: "Codex FO runtime: add preemptible wait mode for subagent completions"
status: ideation
source: "GitHub issue #148, 2026-04-27 - local mitigation for openai/codex#15723 completion wakeup limitation"
started: 2026-04-27T18:27:05Z
completed:
verdict:
score: 0.72
worktree:
issue: "#148"
pr:
---

Codex currently does not wake the calling agent when background subprocesses or subagents complete (`openai/codex#15723`). Spacedock has tried to preserve keep-waiting behavior, but in interactive sessions the first officer can lose the intended wait after a user interruption or rely on a completion notification that will not schedule a new turn.

This task should make `wait_agent` an explicit preemptible first-officer mode. When the first officer is actively expecting worker results and the next orchestration step is blocked, it should record the live wait intent, tell the captain it is waiting and interruptions are allowed, enter `wait_agent` on unresolved handles, treat user input during the wait as `preempted_by_user_input`, then resume waiting unless the captain explicitly pauses/stops or a clarification is required.

Completion notifications should be treated as opportunistic evidence. Under the current Codex constraint, `wait_agent` remains the authoritative collection path.

## Acceptance criteria

**AC-1 - Codex first-officer runtime docs define the preemptible wait contract.**
Verified by: static content checks against `skills/first-officer/references/codex-first-officer-runtime.md`.

**AC-2 - User-facing wait status includes worker labels and handles, and explicitly says interruptions are allowed.**
Verified by: static or transcript fixture checks for the wait-status wording.

**AC-3 - Runtime guidance resumes waiting after non-stopping user interruptions.**
Verified by: transcript fixture or focused runtime test where a wait is interrupted by a user question, answered, and then resumed on the same unresolved handle.

**AC-4 - Runtime guidance distinguishes interrupted/preempted waits from completed, timed-out, or failed waits.**
Verified by: static contract checks plus parser/fixture assertions for wait outcome labels.

**AC-5 - Tests or transcript fixtures cover an interrupted wait resuming on the same unresolved worker handle.**
Verified by: a focused test or fixture that observes the same handle in the initial `wait_agent` call and the resumed `wait_agent` call.

## Related

- GitHub issue: https://github.com/clkao/spacedock/issues/148
- Upstream Codex issue: https://github.com/openai/codex/issues/15723
- Related but not equivalent local task: `153` (`codex-completion-notifications-must-preempt-side-discussion`) depends on completion notification wakeup behavior that is currently upstream-blocked.
