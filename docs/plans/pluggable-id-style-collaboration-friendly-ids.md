---
id: 217
title: "Pluggable id-style with collaboration-friendly generated IDs"
status: ideation
source: "GitHub issue #150 (filed by CL, 2026-04-27)"
started: 2026-04-27T21:32:17Z
completed:
verdict:
score: 0.66
worktree:
issue: "#150"
pr:
mod-block:
---

Spacedock should support pluggable `id-style` strategies and add a generated ID style that is better for collaborative workflows than centrally allocated sequential numbers.

Sequential IDs are readable, but they create coordination pressure when multiple people or agents create entities concurrently, especially across branches, worktrees, offline edits, or multi-workflow projects. The new style should preserve enough human readability for operators and agents while avoiding "who owns the next number" conflicts.

Ideation should research prior art before proposing a design. Relevant directions include Git-style abbreviated object IDs, shortest-unique-prefix schemes, UUID/ULID/KSUID short-prefix variants, NanoID-style collision-budgeted IDs, and any distributed-ID techniques where displayed length can grow with the number of entities or collision risk.

The ideation output should compare at least these design choices:

- Store a full stable generated ID and display/accept the shortest unique prefix.
- Store only a short generated ID and resolve collisions at creation time.
- Treat `id-style` as a pluggable strategy that can also cover the existing proposed `slug` style from GitHub issue #98.

It should also cover how entity reference resolution, `status --next-id`, archived entities, folder-form entities, cross-workflow ambiguity, migrations, and backward compatibility behave.
