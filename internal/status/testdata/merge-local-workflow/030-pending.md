---
id: "030"
title: Entity with an in-flight mod-block under merge local
status: implementation
score: "0.30"
source: roadmap
mod-block: merge:local-merge
---
# Entity with an in-flight mod-block under merge local

A `mod-block` is set: the merge ceremony is in flight. `merge: local` relaxes the
pr-requirement of the merge-hook check, but it does NOT relax the mod-block-pending
guard — a terminal `--set` here must still refuse, because clearing the block and
terminalizing are mandatory separate steps regardless of policy.
