# Skill Integration Notes

Skill integration starts after the launcher can preserve current `status` behavior.

The target is simple:

- first-officer and ensign instructions call `spacedock status`, not plugin-private script paths;
- skill tests verify the command text and the workflow paths handed to agents;
- no PR merge mod or lifecycle mod behavior is in scope for the bootstrap.

This directory intentionally starts as documentation only. Add fixtures here when the skill integration stage begins.
