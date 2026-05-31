---
id: jfk8g2h0h1wsbr3qtncsqsbt
title: Release pipeline — version-stamped darwin binaries + GitHub Release (closes F9)
status: ideation
source: sprint — Ship the Launcher slice C (captain, 2026-05-30)
started: 2026-05-31T01:32:22Z
completed:
verdict:
score: "0.30"
worktree:
issue:
---

Build and publish version-stamped `spacedock` release artifacts so the brew formula has real `url` + `sha256`.

## Target
- Release origin = `clkao/spacedock` @ `next` branch (spacedock-v1 gains that origin; work lands on next, not main — see `fresh-install-journey`).
- goreleaser (or a script + GitHub Actions): build darwin arm64 + amd64 with `-ldflags "-X main.Version=$(git describe --tags)"`, produce `tar.gz` + checksums, publish a GitHub Release.
- Bump the `spacedock-dev/homebrew-tap` formula `url` + `sha256` on release.
- `spacedock --version` on the released binary returns the stamped `git describe` version.

## Dependencies
- Produces the artifacts the homebrew-tap (B) formula points at.
- Releases the launcher-complete binary → land after A (and ideally A′) so the released binary carries the launcher.

## Acceptance criteria (provisional — ideation hardens each)

**AC-1 (closes F9) — pipeline committed + produces a stamped binary.** the release config (`goreleaser.yaml` / `.github/workflows/release.yml`) is committed; a dry-run or tag build produces a darwin binary whose `spacedock --version` equals the `git describe` tag — observed running the binary, not asserted.

**AC-2 — artifacts + checksums published.** a release produces per-platform `tar.gz` + a checksums file at the `url` shape the formula expects.

## Ideation design (hardened)

### Pipeline choice: goreleaser (config) driven by a thin GitHub Actions workflow

Chosen: **goreleaser**, invoked from `.github/workflows/release.yml` on tag push.

Rationale (vs. hand-rolled `go build` matrix + `gh release create` script):
- goreleaser owns the cross-build matrix (darwin arm64 + amd64), tarball naming,
  `checksums.txt` (sha256), and GitHub Release creation in one declarative config —
  the exact four things AC-1/AC-2 ask for. A hand-rolled script re-implements all of
  this and gets the checksums format / archive layout subtly wrong (the precise risk
  the homebrew-tap consumes).
- goreleaser's `builds.ldflags` is the canonical place to wire the version stamp, and
  its `--snapshot --clean` dry-run produces real local artifacts without a tag or a
  publish — that is the cheap behavioral oracle this stage needs (see below).
- It can emit the formula bump directly (`brews:`), which de-risks the tap seam.

Dev-dependency note: **goreleaser is NOT installed** (`which goreleaser` → not found).
Picking it adds a dev/CI dependency. In CI it is `goreleaser/goreleaser-action`; for
local dry-run a dev installs `brew install goreleaser`. The build phase must add this
to the dev-setup docs.

### Version stamp — F9 root cause, observed by running binaries (closes F9)

The launcher plan's gap was an AC that only required a non-empty `--version`. That
hides a real defect, confirmed empirically here by RUNNING built binaries:

1. The version source today is `const Version = "0.1.0-dev"` in
   `internal/cli/cli.go` — NOT a `var`, and NOT in package `main`.
2. `go build -ldflags "-X main.Version=v9.9.9-test"` builds cleanly and the binary
   STILL reports `spacedock 0.1.0-dev` — the stamp is silently dropped (wrong package
   path AND a const cannot be stamped).
3. `go build -ldflags "-X github.com/clkao/spacedock-v1/internal/cli.Version=v1.2.3-stamped"`
   against the existing `const` is ALSO a silent no-op → `0.1.0-dev`. A const is not
   linker-writable; the build does not error.
4. In a probe module where `Version` is a `var` (not const) and the `-X` path is the
   var's fully-qualified package path, `git describe --tags` (`v0.2.3`) stamps through:
   the built binary reported `spacedock v0.2.3`.

**Consequences the build phase MUST honor:**
- `internal/cli.Version` must become a `var` (not `const`) so the linker can write it.
- The ldflags `-X` key is `github.com/clkao/spacedock-v1/internal/cli.Version`,
  **NOT** `main.Version` (the entity's provisional `-X main.Version=...` wording is the
  F9 trap and would silently no-op). The committed pipeline must use the cli-package
  path. (Alternative: move the var into package `main` and have cli read it — extra
  wiring for no benefit; keep it in `internal/cli` where `--version` already prints it.)
- AC-1 oracle = run the built binary and assert `spacedock --version` contains the
  `git describe --tags` value, byte-for-byte on the tag — not "non-empty".

### Release origin + formula-bump seam (named)

- **Origin:** releases cut from `clkao/spacedock` @ `next` branch. spacedock-v1 gains
  that origin; the pipeline config lands on `next`, not `main`. The actual origin push
  (`next` → `clkao/spacedock`) is a CAPTAIN action (per `fresh-install-journey`), not
  the pipeline's.
- **Trigger:** tag push (`v*`) on the origin runs `release.yml` → goreleaser →
  per-platform `tar.gz` + `checksums.txt` attached to a GitHub Release.
- **Formula bump:** the release updates `spacedock-dev/homebrew-tap`
  `Formula/spacedock.rb` `url` + `sha256`. goreleaser's `brews:` block can push this
  automatically (needs a tap-repo token); the fallback is a manual/scripted bump of the
  two fields. Either way the contract with the tap entity (`rgsk9fyka3s5ah87rb773a2f`)
  is: **per-platform darwin `tar.gz` `url` + matching `sha256` from `checksums.txt`**,
  which is exactly the `url`/`sha256` shape the tap's AC-1/AC-2 consume.

## Stage Report: ideation

- DONE: Pipeline design is concrete and a choice is made (goreleaser config OR script+GH-Actions).
  Chose goreleaser driven by a thin `.github/workflows/release.yml` on tag push; rationale + the goreleaser-NOT-installed dev-dependency pinned in "Pipeline choice".
- DONE: Version-stamping is a behavioral oracle (closes F9), observed by RUNNING the built binary.
  Ran three real builds: `-X main.Version=...` → silent no-op (`0.1.0-dev`); `-X .../internal/cli.Version=...` against the const → silent no-op; probe with a `var` + fully-qualified path → binary reported the `git describe` tag `v0.2.3`.
- DONE: The release origin + formula-bump seam are named.
  Origin = `clkao/spacedock@next` (config lands on next, push is a captain action); formula bump = goreleaser `brews:` (or scripted) updating tap `url`+`sha256` from `checksums.txt`, agreeing with the tap entity's url/sha256 consumption contract.

### Summary

Picked goreleaser (config + a thin tag-triggered GH-Actions workflow) over a hand-rolled build script because it owns the darwin arm64+amd64 matrix, `checksums.txt`, the GitHub Release, AND `--snapshot --clean` as a cheap dry-run oracle — at the cost of a new dev/CI dependency (goreleaser is not installed). The F9 root cause was found by running binaries: the entity's provisional `-X main.Version=...` is a silent no-op because the version source is a `const` (not a `var`) in `internal/cli` (not `main`); the build phase must make it a `var` and stamp `github.com/clkao/spacedock-v1/internal/cli.Version`, with AC-1 asserting `spacedock --version` equals `git describe --tags` byte-for-byte. The release origin (`clkao/spacedock@next`, captain-pushed) and the formula-bump seam (tap `url`+`sha256` from `checksums.txt`) are named and agree with the homebrew-tap entity's consumption contract.
