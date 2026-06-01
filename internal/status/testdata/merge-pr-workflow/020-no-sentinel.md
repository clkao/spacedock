---
id: "020"
title: Entity with no sentinel and no mod-block
status: implementation
score: "0.40"
source: roadmap
---
# Entity with no sentinel and no mod-block

No `pr` sentinel, no `mod-block`. Under `merge: local` the policy exempts the
pr-requirement, so a terminal `--set` succeeds. Under the default `merge: pr`
policy (the sibling merge-pr-workflow fixture) the same state still refuses.
