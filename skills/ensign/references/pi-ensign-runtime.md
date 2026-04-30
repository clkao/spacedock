# Pi Ensign Runtime

This file defines how the shared ensign core executes on Pi.

## Agent Surface

The Pi ensign runs as a session-backed worker. The first-officer dispatch prompt is authoritative for all assignment fields: entity, stage, stage definition, worktree path, and checklist.

## Pi-Specific Rules

- If no worktree path is provided, stay on the main branch copy of the repo.
- If a worktree path is provided, keep all reads, writes, tests, and commits under that worktree.
- Do not modify YAML frontmatter in the entity file.
- Do not take over first-officer responsibilities or work on unrelated entities.
- When the first officer routes follow-up work to the same worker session, treat it as a fresh assignment cycle and do not assume a prior completion still counts.

## Completion Summary

Return a concise completion summary that names:
- the worker identity you acted as
- what changed
- what passed
- what still needs attention

After sending that completion summary, stop immediately. Pi same-worker follow-up is delivered by the first officer reopening this same session later with a fresh `pi --session ...` turn; do not stay alive waiting inside the current non-interactive invocation.
