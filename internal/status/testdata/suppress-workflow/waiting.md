---
id: waiting
title: Ready but next stage is full
status: ideation
score: "0.80"
source: probe
---

# Ready but next stage is full

At ideation with no worktree: would advance to build, but build (concurrency 1)
is saturated by the in-flight build, so --next suppresses it for concurrency.
