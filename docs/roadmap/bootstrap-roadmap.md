# Spacedock v1 Bootstrap Roadmap

This roadmap converts the current design into a compatibility-first Go launcher. Each stage has an entity in `docs/dev/.spacedock-state/`.

## Scope

In scope:

- fresh Go launcher repo;
- vendored or delegated current status behavior;
- per-workflow `.spacedock-state` profile;
- README symlink compatibility phase;
- native Go `status` with `state:` split-root support;
- first-officer and ensign skill command routing to the new binary.

Out of scope:

- PR merge flow;
- mods and lifecycle hooks;
- external tracker writeback beyond simple entity link fields;
- replacing the whole first-officer runtime.

## Stages And Required Tests

### 1. Bootstrap Go Launcher

Required tests:

- `go test ./...`
- `go run ./cmd/spacedock --help`
- `go run ./cmd/spacedock --version`
- CLI unit tests for help, version, and unknown commands.

### 2. Vendor Status Compatibility

Required tests:

- `go test ./...`
- fixture tests proving `spacedock status` delegates to the vendored status implementation;
- golden output parity for default status, `--next`, `--validate`, `--resolve`, and `--short-id`;
- mutation smoke tests for `--set` and `--archive` in a temporary workflow.

### 3. Symlink State Profile

Required tests:

- integration test creates `docs/dev/README.md` and `docs/dev/.spacedock-state/README.md -> ../README.md`;
- `spacedock status --workflow-dir docs/dev/.spacedock-state` renders entities from the state checkout;
- folder-form entities with `reports/` and `artifacts/` are not misdetected as separate entities;
- archived entities move under `.spacedock-state/_archive`;
- no mod or PR behavior is required.

### 4. Skill Integration With Vendor Branching

Required tests:

- static skill tests prove first-officer instructions call `spacedock status`;
- ensign dispatch instructions receive the entity path under `.spacedock-state`;
- pilot workflow smoke test can list, mutate, and archive an entity through the launcher;
- no skill path references plugin-private `skills/commission/bin/status`.

### 5. Native Go Status Parity

Required tests:

- Go-native parser tests match current frontmatter parser behavior;
- Go-native stage parser tests match current README stage behavior;
- golden output parity for default status, `--archived`, `--next`, `--where`, `--fields`, `--all-fields`, `--next-id`, `--resolve`, and `--short-id`;
- mutation tests for `--set`, timestamp fill, PR field preservation as a normal field for now, and `--archive`;
- validation tests for duplicate IDs, bad IDs, flat/folder conflicts, unknown stages, and terminal/archive guards.

### 6. Native `state:` Split Root

Required tests:

- `state: .spacedock-state` resolves relative to the workflow README directory;
- no README symlink is required inside `.spacedock-state`;
- status reads stages from the main README and entities from the state checkout;
- `--set` mutates only state checkout files;
- `--archive` moves only state checkout files;
- discovery finds the main workflow README and ignores `.spacedock-state`.

### 7. Retest Without Symlink

Required tests:

- remove `.spacedock-state/README.md`;
- rerun full Go test suite;
- rerun pilot workflow commands against split-root mode;
- verify `git status` in the main repo does not show runtime state churn;
- verify `git status` in `.spacedock-state` shows only workflow state changes.

## External Tracker Checkpoint

After native split-root status works, add a design checkpoint for kata and Linear-style integrations:

- import an external ticket into a Spacedock entity using `issue` and `source`;
- preserve the external reference through status, mutation, reports, and archive;
- decide whether any bidirectional sync belongs in v1 or should remain an adapter.
