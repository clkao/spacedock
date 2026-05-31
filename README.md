# Spacedock v1

Spacedock v1 is the Go launcher and compatibility bridge for the next Spacedock command surface.

The first implementation target is conservative:

- provide a `spacedock` binary entry point;
- preserve current `status` behavior through a vendored compatibility path;
- prove per-workflow `.spacedock-state` state checkouts with the README symlink model;
- then replace the symlink dependency with native split-root status handling.

The development workflow for this repo lives in `docs/dev/README.md`. Runtime entities for that workflow live in `docs/dev/.spacedock-state/`, which is intended to be a separate git checkout or nested state repo.

## Install

Two lanes — see [`docs/install-journey.md`](docs/install-journey.md) for the
step-by-step journey with the observable output at each step.

**Released lane (brew)** — available after the first tagged release:

```bash
brew tap spacedock-dev/homebrew-tap
brew install spacedock
spacedock init --host claude
```

The no-tap one-liner `brew install spacedock-dev/homebrew-tap/spacedock` is
equivalent.

**Dev lane (`--plugin-dir`)** — source build from `next`, the primary dev
workflow (`next` has no release artifact — the pipeline triggers on `v*` tags
only, so `@next` is a source build, not brew):

```bash
git clone https://github.com/spacedock-dev/spacedock
cd spacedock
go build -o spacedock ./cmd/spacedock
./spacedock claude --plugin-dir "$PWD" -- "your task"
```

`--plugin-dir` loads the repo's own vendored `spacedock:first-officer` /
`spacedock:ensign` skills and relaxes the contract gate.

[safehouse](https://agent-safehouse.dev) is a separate runtime dependency for
sandboxed launches — not installed by either lane. A `.safehouse` profile in the
working directory (or the `--safehouse` / `--safehouse-<key>=…` flags) wraps the
launch through it.

## Usage

```bash
spacedock claude [host-flags…] [--safehouse…] -- "task"   # launch claude --agent spacedock:first-officer
spacedock codex  [host-flags…] [--safehouse…] -- "task"   # launch codex with the spacedock:first-officer skill
spacedock --version                                       # spacedock <version> (contract 1)
spacedock doctor                                          # contract compatibility verdict
```

Flags before `--` pass through to the host; the bare text after `--` is the
launch task. `--skip-contract-check` bypasses the contract gate (bootstrap
only); a `--plugin-dir` launch relaxes it without the flag.

## Commands

```bash
go test ./...
go run ./cmd/spacedock --help
```
