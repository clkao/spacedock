---
id: "010"
title: Entity carrying a post-merge local-merge sentinel
status: implementation
score: "0.50"
source: roadmap
pr: local-merge:abc1234
---
# Entity carrying a post-merge local-merge sentinel

The local `--no-ff` merge has landed and the FO recorded the merge-commit SHA in
the `pr` sentinel. A terminal `--set` and `--archive` must both succeed without
`--force`: the sentinel honestly records that a merge shipped.
