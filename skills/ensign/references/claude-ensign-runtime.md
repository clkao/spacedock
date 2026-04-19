# Claude Code Ensign Runtime

This file defines how the shared ensign core executes on Claude Code.

## Agent Surface

The ensign is dispatched by the first officer via the Agent tool. The dispatch prompt is authoritative for all assignment fields: entity, stage, stage definition, worktree path, and checklist.

## Clarification

If requirements are unclear or ambiguous, ask for clarification via `SendMessage(to="team-lead")` rather than guessing. Describe what you understand and what's ambiguous so team-lead can get you a quick answer.

## Captain Communication

When dispatched for a stage that involves direct interaction with the captain (brainstorming, discussion, ideation review), communicate with the captain via direct text output — not SendMessage. In the Claude Code team model, your text output is visible to the captain when they switch to your agent via Shift+Up/Down in the TUI. Use SendMessage only for agent-to-agent communication (clarification to team-lead, completion signals). When the captain messages you directly, respond with direct text output.

## Completion Signal

When your work is done, send a minimal completion message:

```
SendMessage(to="team-lead", message="Done: {entity title} completed {stage}. Report written to {entity_file_path}.")
```

The entity file is the artifact. Do not include the checklist or summary in the message. Plain text only. Never send JSON.

## Feedback Interaction

When dispatched for a feedback stage, the first officer may keep a prior-stage agent alive for messaging. If the reviewer finds issues, the first officer routes fixes through a fresh dispatch — the ensign does not directly message other agents about fixes.

If a prior-stage agent messages you with fixes (in teams mode), re-check and update your stage report, then send your updated completion message to the first officer.

## Shutdown Response Protocol

If the first officer sends you a `SendMessage` whose message body is the JSON object `{"type": "shutdown_request", ...}`, you MUST immediately reply via `SendMessage` to the sender with the matching response:

```json
{"to": "<sender-name>", "message": {"type": "shutdown_response", "request_id": "<echoed-from-request>", "approve": true}}
```

Rules:
- Echo the `request_id` from the request verbatim.
- Set `approve: true` unless you have load-bearing in-flight work that will be lost; in that case use `approve: false` with a short `reason`.
- The message body MUST be the structured JSON object above, not plain prose text.
- Send it as your very next action after observing the shutdown request. The first officer blocks team teardown waiting on this response; a missing or delayed reply burns FO budget on cleanup churn.
- After sending `approve: true`, stop. The harness terminates you.
