---
id: "215"
title: Interactive stage type for captain-performed work
status: ideation
source: captain session 2026-04-21
started: 2026-04-22T05:26:49Z
completed:
verdict:
score:
worktree:
---

Allow a workflow stage to be declared `interactive: true` so the FO keeps that stage foreground and hands the work to the captain instead of dispatching a worker. Intended for stages that need heavy back-and-forth (clarification-heavy ideation, design critique, hand-authored content) where a subagent would ping-pong more than it would progress.

The ideation pass should clarify the schema change, the FO dispatch-path behavior, how completion is signalled (captain writes the stage report? FO prompts and transcribes? something else?), gating interactions, reuse implications, and whether this conflicts with `worktree: true` or `feedback-to` stages. Acceptance criteria and test plan follow from those choices.
