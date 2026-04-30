---
commissioned-by: spacedock@0.11.0
mission: Regression fixture — workflow with no local status executable
entity-label: task
entity-label-plural: tasks
id-style: slug
stages:
  defaults:
    worktree: false
    fresh: false
    gate: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: review
      gate: true
    - name: build
      worktree: true
    - name: done
      terminal: true
---

# Debrief No-Local-Status Fixture

Regression fixture for #175. Modern Spacedock workflows do not ship a local
`status` executable in the workflow directory — the viewer is plugin-shipped
under `{spacedock_plugin_dir}/skills/commission/bin/status`. This fixture
deliberately omits the local script so the debrief skill's Phase 2e fallback
chain can be exercised against the failure shape.

The three entity files span the buckets the "What's Next" section enumerates:
dispatchable, gate-blocked, in-progress with worktree.
