---
name: first-officer
description: Use when running or resuming a Spacedock workflow, especially to discover a workflow, dispatch packaged workers, manage approval gates, and advance entity state.
user-invocable: true
---

If this skill is invoked directly in a non-interactive run and the prompt names a specific entity to process, enter single-entity mode immediately:
- scope work to that entity only
- follow the shared single-entity rules from the operating contract and any runtime-specific bounded-stop rules
- keep running until the shared/runtime-specific stop condition for the requested bounded outcome is satisfied
- do not treat an initial rejection as terminal when the workflow's feedback flow expects a routed follow-up
- if the prompt only names the entity and does not explicitly request terminal completion, treat the runtime's bounded routed-reuse stop rule as sufficient
- before the final response, explicitly shut down any worker that is no longer needed for later routing or gate handling
- once the bounded stop condition is satisfied, send one concise final response and exit immediately

## Operating contract

@references/first-officer-shared-core.md
@references/code-project-guardrails.md

## Runtime adapter

Load the runtime adapter for your platform:
- Claude Code (`CLAUDECODE` env var is set): read `references/claude-first-officer-runtime-core.md`
- Codex (`CODEX_HOME` env var is set): read `references/codex-first-officer-runtime.md`

Then begin the Startup procedure from the shared core.
