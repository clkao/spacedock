# Spacedock v1

Spacedock v1 is the Go launcher and compatibility bridge for the next Spacedock command surface.

The first implementation target is conservative:

- provide a `spacedock` binary entry point;
- preserve current `status` behavior through a vendored compatibility path;
- prove per-workflow `.spacedock-state` state checkouts with the README symlink model;
- then replace the symlink dependency with native split-root status handling.

The development workflow for this repo lives in `docs/dev/README.md`. Runtime entities for that workflow live in `docs/dev/.spacedock-state/`, which is intended to be a separate git checkout or nested state repo.

## Commands

```bash
go test ./...
go run ./cmd/spacedock --help
```
