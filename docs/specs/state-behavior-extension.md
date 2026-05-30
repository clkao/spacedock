# Spacedock State Behavior Extension

Status: draft bootstrap extension.

This document extends the frontmatter and state-machine contract with a storage behavior profile. It does not replace the entity state machine. The existing contract still defines stages, frontmatter fields, guards, archive behavior, and report conventions.

## Purpose

The extension separates workflow definition from workflow runtime state for development workflows that want shared Spacedock issues without noisy state transitions in the code branch.

The v0 profile is `state-branch`:

- the workflow README stays in the main repo;
- mutable workflow entities live in a per-workflow `.spacedock-state` checkout;
- the compatibility phase uses a README symlink inside `.spacedock-state`;
- the native phase reads the main README and state checkout as one composed workflow view.

## V0 Layout

```text
docs/plans/
  README.md
  .spacedock-state/
    README.md -> ../README.md
    add-login.md
    refactor-dispatch/
      index.md
      reports/
      artifacts/
    _archive/
    _debriefs/
```

Active entities live directly under `.spacedock-state`. There is no `entities/` directory in v0.

## README Frontmatter

The simplest extension is one top-level field:

```yaml
state: .spacedock-state
```

The path is resolved relative to the workflow README directory. If `state` is absent, current same-directory behavior applies.

The compatibility phase does not require the current status script to understand this field because `.spacedock-state/README.md` symlinks back to the real README. The native Go implementation must understand it and must not require the symlink.

## Mutation Rules

For `state: .spacedock-state` workflows:

- `status`, `--next`, `--resolve`, `--short-id`, and `--validate` read stages from the main README and entities from `.spacedock-state`;
- `--set` mutates entity frontmatter in `.spacedock-state`;
- `--archive` moves entities under `.spacedock-state/_archive`;
- stage reports and validation evidence live under the folder-form entity when they are part of the workflow record;
- product code, tests, docs, fixtures, examples, and release artifacts live in the main repo when the task asks for them.

Mods and PR merge behavior are explicitly out of scope for the bootstrap.

## External Tracker Integration

The extension creates a cleaner integration point for systems such as kata, Linear, GitHub Issues, or other ticket ledgers.

Spacedock remains the execution workflow. External systems may own backlog intake, discussion, assignment, or reporting, but they should not redefine Spacedock stage semantics inside the entity file.

The v0 bridge should use simple top-level fields that the current frontmatter parser can preserve:

```yaml
issue: ENG-123
source: linear
```

or:

```yaml
issue: kata:task-abc123
source: kata
```

Principles:

- `issue` is the human-facing external reference.
- `source` records where the entity came from when useful.
- Spacedock status remains the execution status unless a future bridge explicitly declares bidirectional ownership.
- Bridge tools should sync through entity creation, state changes, and stage reports rather than by adding tracker-specific stage rules.
- If richer metadata becomes necessary, add flat custom fields before introducing nested YAML because v1 parser semantics are line-oriented.

This keeps the current design open to kata and Linear without making either one a required backend.

## Native Go Status Requirements

The native `spacedock status` command must support two roots:

```text
definition_dir = directory containing README.md
state_dir = definition_dir / state
```

It must read:

- workflow identity and stage declarations from `definition_dir/README.md`;
- active entities from `state_dir`;
- archived entities from `state_dir/_archive`.

It must write:

- frontmatter updates to entities in `state_dir`;
- archive moves to `state_dir/_archive`.

It must preserve current status output and validation behavior before adding new UX.
