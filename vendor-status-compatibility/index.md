---
id: jepnnnp1jr3zjzes2rz6ng6g
title: Vendor status compatibility
status: ideation
score: "0.90"
source: bootstrap roadmap
worktree:
started: 2026-05-30T04:24:41Z
---

# Vendor Status Compatibility

Make `spacedock status` preserve the current status script behavior through a vendored or delegated compatibility path.

## Acceptance Criteria

- `spacedock status` accepts the current status flags needed by FO workflows.
- Current output remains byte-for-byte compatible for selected fixtures.
- Mutation commands still use current frontmatter semantics.
- The compatibility layer has a narrow interface so it can later be replaced by native Go code.

## Test Gates

- `go test ./...`
- Golden parity for default status output.
- Golden parity for `--next`, `--validate`, `--resolve`, and `--short-id`.
- Temporary-workflow mutation tests for `--set` and `--archive`.

## Notes

This stage should not add split-root behavior. It protects the known contract first.
