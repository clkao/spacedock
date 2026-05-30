# Agent Instructions

This repo builds the Go-based `spacedock` launcher and the project-side skill integration needed to migrate Spacedock from plugin-shipped scripts to a stable command surface.

## Priorities

- Keep the first implementation compatibility-first. Preserve current `status` behavior before adding new semantics.
- Do not add PR or mod behavior yet. The bootstrap scope is status, state checkout layout, skill command routing, and tests.
- Prefer small Go packages with clear boundaries. Avoid a single large CLI file.
- Use the standard library unless a dependency removes real complexity.
- Keep command output stable and test it with fixtures.
- Use folder-form workflow entities when reports or artifacts may accumulate beside an entity.

## Expected Commands

Run these before claiming work is complete:

```bash
go test ./...
go test ./... -race
gofmt -w ./cmd ./internal
```

Use `go test ./...` as the baseline gate for every stage. Add focused tests for each stage before implementing that stage.

## Project Shape

- `cmd/spacedock/`: process entry point only.
- `internal/cli/`: command routing, usage text, exit-code behavior.
- `internal/status/`: future status implementation and compatibility runner.
- `docs/specs/`: design contracts, including the state behavior extension.
- `docs/roadmap/`: bootstrap and migration roadmap.
- `docs/dev/README.md`: development workflow definition.
- `docs/dev/.spacedock-state/`: development workflow entities in a separate state checkout.
- `skills/`: future skill-facing command integration notes and fixtures.

## State-Branch Bootstrap Rules

- For v0, each workflow may use a per-workflow `.spacedock-state` checkout.
- The workflow README stays in the main repo.
- During the compatibility phase, `.spacedock-state/README.md` may be a symlink to `../README.md`.
- Active entities live directly under `.spacedock-state/`; do not introduce an `entities/` directory in v0.
- Entity reports and artifacts live under the folder-form entity when they are part of the workflow record.
- Product code, fixtures, docs, and release artifacts belong in the main repo when the task explicitly calls for them.

## Skill Development

- Skills should call `spacedock`, not plugin-private script paths.
- Keep skill instructions declarative. Let the binary own path resolution and mutation guards.
- Add skill smoke tests before changing first-officer or ensign command text.
- Preserve current FO/ensign write-scope rules: the first officer mutates entity state; ensigns write assigned code, reports, and artifacts.
