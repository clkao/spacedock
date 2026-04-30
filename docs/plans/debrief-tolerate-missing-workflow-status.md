---
id: 8xyvkvsgb93sch31cmz6nf9w
title: "debrief assumes workflow status executable exists"
status: backlog
source: "GitHub issue #175 (filed by Kent Chen / iamcxa, 2026-04-30)"
started:
completed:
verdict:
score: 0.55
worktree:
issue: "#175"
pr:
mod-block:
---

`spacedock:debrief` Phase 2e ("What's Next" extraction) instructs agents to run `{dir}/status --next` and `{dir}/status` against the workflow directory. Modern Spacedock ships the `status` viewer with the plugin (`{spacedock_plugin_dir}/skills/commission/bin/status`); workflows commissioned with newer versions do not carry a local `{dir}/status` executable. The debrief skill treats the local script as mandatory and fails with `no such file or directory` when it's absent.

Reporter (Kent Chen) hit this on a workflow commissioned by `spacedock@0.10.1` with the debrief skill from plugin `spacedock@0.10.2`.

## Suggested fix shapes (for ideation)

Two reasonable directions:

- **Document the fallback explicitly** — debrief skill notes that `{dir}/status` is optional, instructs agents to use the plugin-shipped status instead (`{spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {dir} --next`), and falls back to entity-frontmatter scanning if even that's unavailable.
- **Detect-and-route** — debrief preflight: if `{dir}/status` is missing, swap to plugin-shipped invocation; if the plugin-shipped helper is also unreachable, emit a clear "no status binary found, falling back to manual frontmatter scan" notice and proceed.

The workflow-frontmatter fallback (last resort) is already implicit in the agent's capabilities — debrief just needs to make it a documented degraded mode rather than a silent improvisation.

Adjacent: #174 (debrief discovery ignoring `.claude/worktrees`) — same skill, same Kent Chen report. Both are debrief-skill robustness fixes; could be batched into one commission cycle if convenient.
