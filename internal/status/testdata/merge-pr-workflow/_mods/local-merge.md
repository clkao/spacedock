---
name: local-merge
description: Registers a merge hook so the terminal-guard merge-hook branch is reachable.
---

# Local Merge

A minimal stub mod whose only purpose is to register a `merge` lifecycle hook.
The terminal-transition guard reads only the hook registration (the `## Hook:`
heading), not the hook body, so the body is left intentionally empty.

## Hook: merge

(stub — registration only)
