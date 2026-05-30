---
id: trap-not-an-entity
title: Ideation stage report
status: ideation
score: "0.99"
source: misdetection-trap
---
# Stage Report: ideation

This is a STAGE REPORT, not an entity. It deliberately carries a valid opening
`---` fence and an `id:` field so that a naively-recursive discovery would mistake
it for an entity. Single-level discovery (os.listdir on the entity directory) must
never reach this nested path, so no `ideation`/`reports` slug may appear in status.
