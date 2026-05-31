---
id: jfk8g2h0h1wsbr3qtncsqsbt
title: Release pipeline — version-stamped darwin binaries + GitHub Release (closes F9)
status: validation
source: sprint — Ship the Launcher slice C (captain, 2026-05-30)
started: 2026-05-31T01:32:22Z
completed:
verdict:
score: "0.30"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-release-pipeline
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
   var's fully-qualified package path, the stamp value flows through and the built
   binary reports it (probe used a seeded tag `v0.2.3`).
5. **No git tags exist in this repo** (`git tag` count = 0). `git describe --tags`
   exits 128 (`fatal: No names found, cannot describe anything`) — so a build that
   feeds raw `git describe --tags` into ldflags would ERROR, not stamp. `git describe
   --tags --always` falls back to a commit-ish (`cab8b48`) and does not error.

**Consequences the build phase MUST honor:**
- `internal/cli.Version` must become a `var` (not `const`) so the linker can write it.
  This is a one-line SOURCE edit (`const Version = "0.1.0-dev"` →
  `var Version = "0.1.0-dev"` at `internal/cli/cli.go:17`) that the pipeline depends
  on — it lands in `internal/cli`, not in the pipeline config. Sequencing/ownership:
  this source change ships WITH the pipeline work (same entity), on `next`; it is the
  smallest change that unblocks stamping and does not alter the default-build output.
- The ldflags `-X` key is `github.com/clkao/spacedock-v1/internal/cli.Version`,
  **NOT** `main.Version` (the entity's provisional `-X main.Version=...` wording is the
  F9 trap and would silently no-op — there is no `Version` symbol in package `main`).
  The committed pipeline must use the cli-package path.
- The stamp value must use `git describe --tags --always` (or equivalent safe
  fallback), NOT bare `git describe --tags`, so a tagless build stamps a commit-ish
  instead of erroring. Separately, the **first release must seed an initial tag**
  (e.g. `v0.1.0`): goreleaser requires a tag to cut a release, so tag-seeding is a
  prerequisite of the first release path, and only a tagged build stamps a real semver.
- AC-1 oracle = run the built binary and assert `spacedock --version` contains the
  `git describe --tags --always` value byte-for-byte (a real `git describe` semver on a
  tagged build) — not "non-empty".

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

## Stage Report: ideation (cycle 2)

Folded three factual corrections (team-lead, verified against code) into the version-stamp design.

- DONE: Pipeline design is concrete and a choice is made (goreleaser config OR script+GH-Actions).
  Unchanged: goreleaser via tag-triggered `.github/workflows/release.yml`; goreleaser-NOT-installed dev-dependency pinned.
- DONE: Version-stamping is a behavioral oracle (closes F9), observed by RUNNING the built binary.
  Corrected: (1) ldflags target is `github.com/clkao/spacedock-v1/internal/cli.Version`, not `main.Version` (no Version symbol in package main); (2) `Version` is a `const` and `-X` cannot stamp a const — pipeline depends on a one-line `const`→`var` source edit at `internal/cli/cli.go:17`, shipping with the pipeline on `next`; (3) `git tag` count = 0 so `git describe --tags` exits 128 — use `git describe --tags --always` fallback and seed an initial tag (e.g. `v0.1.0`) for the first release. AC-1 oracle now reflects the real var + const→var change + tag-seeding.
- DONE: The release origin + formula-bump seam are named.
  Unchanged: origin `clkao/spacedock@next` (captain push); formula bump = tap `url`+`sha256` from `checksums.txt`, agreeing with the homebrew-tap entity.

### Summary

Three corrections folded in: the ldflags `-X` key is `internal/cli.Version` (package `main` has no Version symbol); `Version` must change from `const` to `var` (a const is linker-silent) — a one-line source edit at `internal/cli/cli.go:17` that ships with the pipeline on `next`; and because the repo has zero tags, `git describe --tags` errors (exit 128), so the stamp must use `git describe --tags --always` and the first release must seed an initial tag (goreleaser requires a tag to cut a release). All three verified against the code. Pipeline choice, origin, and formula-bump seam are unchanged from cycle 1.

## Stage Report: implementation

- DONE: Release config committed: a goreleaser config (.goreleaser.yaml) + a tag-triggered `.github/workflows/release.yml` that build darwin arm64+amd64 with `-ldflags "-X github.com/spacedock-dev/spacedock/internal/cli.Version=$(git describe --tags --always)"`, emit tar.gz + a checksums file, and publish a GitHub Release; AND the required `const Version` → `var Version` edit at internal/cli/cli.go.
  Commit `e745d4d` on `spacedock-ensign/release-pipeline`: `.goreleaser.yaml` (darwin arm64+amd64, ldflags `-X .../internal/cli.Version={{.Version}}`, tar.gz, `checksums.txt`, GitHub Release, `brews:` tap bump), `.github/workflows/release.yml` (v* tag → goreleaser-action on macos-latest, fetch-depth:0 for git-describe), and `const`→`var Version` at internal/cli/cli.go:17.
- DONE: Version-stamp behavioral proof (closes F9): run a dry-run producing a darwin binary whose `spacedock --version` shows the git-describe-derived version — OBSERVED by RUNNING the built binary, not asserted. No tags exist yet → `--always` fallback; document the first-release tag-seeding.
  Real F9 proof via PLAIN `go build` (no goreleaser needed): `go build -ldflags "-X github.com/spacedock-dev/spacedock/internal/cli.Version=$(git describe --tags --always)" -o /tmp/sd-f9 ./cmd/spacedock` then RAN `/tmp/sd-f9 --version` → `spacedock e745d4d (contract 1)`, which equals `spacedock $(git describe --tags --always) (contract 1)` by strict full-string equality (HEAD `e745d4d`, no tags → commit-ish fallback, no error). Unstamped control (`go build` w/o `-X`) still reports `spacedock 0.1.0-dev (contract 1)`. A scripted equivalent of the goreleaser archive/checksum stages also produced `spacedock_{ver}_darwin_{arm64,amd64}.tar.gz` (bare-binary payload, no wrap dir) + a sha256 `checksums.txt`. First release must seed a tag (e.g. `v0.1.0`) for a real semver stamp — config uses `--always` so a tagless build stamps the commit-ish instead of erroring.
- DONE: Gates green with REAL captured exit codes (the const→var must keep `TestVersion` + the contract-token test green); name the formula-bump seam (the tap's url + sha256 sourced from checksums.txt) so rg agrees.
  `go test ./...` exit 0 (470 passed / 8 pkgs), `go vet ./...` exit 0, `go build ./...` exit 0; `TestVersion` + `TestVersionContractToken` exit 0 both PASS (they read the `Version` symbol, so const→var is transparent). Formula-bump seam for rg (homebrew-tap): goreleaser `brews:` pushes `Formula/spacedock.rb` to `spacedock-dev/homebrew-tap` with per-arch `url` = `spacedock_{version}_darwin_{arm64|amd64}.tar.gz` (asset name template, bare-binary payload, no wrap dir) and `sha256` sourced from `checksums.txt` — matching the tap entity's three-part url/sha256/payload contract.

### Summary

Committed the version-stamped darwin release pipeline (commit `e745d4d` on `spacedock-ensign/release-pipeline`): a goreleaser v2 config + a tag-triggered macOS workflow that build arm64+amd64, stamp `internal/cli.Version` via ldflags `-X`, emit tar.gz + checksums.txt, publish a GitHub Release, and bump the homebrew-tap formula. The F9 fix is the `const`→`var Version` edit at internal/cli/cli.go:17 — proven by RUNNING a built binary that reports the git-describe value (`e745d4d` at HEAD) by strict full-string equality, while the unstamped control still reports `0.1.0-dev`. MIGRATED note honored over the stale ideation body: module/origin are `spacedock-dev/spacedock` (not `clkao/spacedock`), so the ldflags `-X` key and brews/release owners all target `spacedock-dev`. Two caveats for the gate: (1) goreleaser itself could NOT be RUN in the sandbox (disk at 100%/1.7Gi free, brew cache permission-blocked, `go install` ENOSPC) — the F9 stamp proof uses a plain `go build` and needs no goreleaser, but the goreleaser-specific YAML schema is UNVALIDATED by the tool and must be captain-verified (commands below); (2) the first real release must seed an initial tag (e.g. `v0.1.0`) and CI needs a `HOMEBREW_TAP_TOKEN` secret for the cross-repo formula push.

### Captain-run verification (could NOT run in sandbox — not claimed to pass)

These validate the goreleaser-specific config shape, which the in-sandbox plain-`go-build` F9 proof does not cover. Run outside the sandbox, from the worktree root (`.worktrees/spacedock-ensign-release-pipeline`):

1. Config lint: `goreleaser check` — expect exit 0 (validates the v2 schema, `builds`/`archives`/`checksum`/`brews` keys).
2. Full dry-run + stamp re-proof through goreleaser: `goreleaser release --snapshot --clean` then run the emitted arm64 binary — `./dist/spacedock_darwin_arm64*/spacedock --version` should report the snapshot version goreleaser stamped (not `0.1.0-dev`), and `dist/checksums.txt` + `dist/spacedock_*_darwin_{arm64,amd64}.tar.gz` should exist with the bare-`spacedock` payload.
3. Live release (real GitHub ops, captain/CI only): seed `git tag v0.19.0 && git push origin v0.19.0` on `spacedock-dev/spacedock@next`, set the `HOMEBREW_TAP_TOKEN` repo secret, and let `.github/workflows/release.yml` cut the release + bump the tap formula.

## Stage Report: implementation (cycle 2)

Captain version bump to 0.19.0, folded into the single source of truth.

- DONE: bump default `Version` to 0.19.0 at internal/cli/cli.go, keep it the single source of truth.
  Commit `cd9a76c` on `spacedock-ensign/release-pipeline`: `var Version = "0.19.0"`. Verified the unstamped build now reports `spacedock 0.19.0 (contract 1)` by RUNNING it. No other version literals exist (grep: 0 hits in `.go`/`.yaml`/`.yml` outside the symbol) — the goreleaser config stamps `{{.Version}}` (tag-derived) and the cli tests read the `Version` symbol, so nothing else needed editing. The tap formula version follows this (rg bumping to 0.19.0 in parallel).
- DONE: F9 stamp proof still holds after the bump.
  `go build -ldflags "-X .../internal/cli.Version=$(git describe --tags --always)"` then RAN it → `spacedock e745d4d (contract 1)`, strict full-string equality vs the recomputed git-describe value; the `-X` stamp correctly overrides the 0.19.0 default. First-release tag corrected to `v0.19.0` (was `v0.1.0`) — see Captain-run verification step 3.
- DONE: gates green after the bump.
  `go test ./...` exit 0, `go vet ./...` exit 0, `go build ./...` exit 0, `TestVersion`+`TestVersionContractToken` exit 0.

### Summary

Folded the captain's 0.19.0 binary-version bump into the single source of truth at internal/cli/cli.go (commit `cd9a76c`): the unstamped build now reports `spacedock 0.19.0 (contract 1)` (observed by running it), and the F9 git-describe `-X` stamp still overrides it. The first-release tag is corrected to `v0.19.0`. No other version literals exist — the goreleaser config stamps `{{.Version}}` and the tests read the symbol — so this is the only edit, and all gates plus the F9 proof remain green.

## Stage Report: implementation (cycle 3)

Fixed an AC-3 leak in the generated tap formula (caught by rg, via team-lead).

- DONE: `brews.caveats` carries the safehouse docs link so the generated formula honors homebrew-tap AC-3.
  The `brews:` block regenerates `Formula/spacedock.rb` on every release, so its caveats must match the tap's AC-3 (safehouse documented via a link). The block previously ended at "before first launch." and dropped the link — a silent AC-3 violation on every released formula. Commit `60989ed` on `spacedock-ensign/release-pipeline`: aligned the wording to rg's hand-authored formula (`runtime dependency` lowercase; "install it through the host tooling"; "See the safehouse install docs:") and added the captain-pinned link `https://agent-safehouse.dev`, so the generated caveats equals rg's `dist/homebrew/Formula/spacedock.rb` caveats byte-for-byte.
- DONE: gates stay green (config-only change).
  `go test ./...` exit 0.

### Summary

rg caught that goreleaser's `brews:` block overwrites the tap formula on every release and my `brews.caveats` dropped the safehouse docs link, silently breaking homebrew-tap AC-3 on each released formula. Fixed in commit `60989ed` by aligning the caveats wording to rg's hand-authored formula and adding the captain-pinned `https://agent-safehouse.dev` link — the generated formula's caveats now match rg's byte-for-byte. Config-only change; `go test ./...` stays green (exit 0). Ready for validation.

## Stage Report: validation

- DONE: Independently reproduce the F9 version-stamp proof (plain `go build -ldflags ...internal/cli.Version=$(git describe --tags --always)` → RUN binary → strict byte-for-byte equality; unstamped default prints `spacedock 0.19.0 (contract 1)`; `var Version = "0.19.0"` + TestVersion + TestVersionContractToken pass).
  Reproduced in the worktree. `git describe --tags --always` = `60989ed` (0 tags). Stamped build RAN → `spacedock 60989ed (contract 1)` == `spacedock $(git describe --tags --always) (contract 1)` (strict `=` PASS). Unstamped control RAN → `spacedock 0.19.0 (contract 1)` (default match). `var Version = "0.19.0"` at internal/cli/cli.go:23; `go test ./internal/cli -run 'TestVersion$|TestVersionContractToken' -v` → both `--- PASS`. Negative proof: `-X main.Version=v9.9.9-trap` RAN → still `0.19.0` (the F9 trap silently no-ops, confirming the cli-package path is load-bearing).
- DONE: Verify `.goreleaser.yaml` + `release.yml` structurally AND against the B↔C contract (darwin arm64+amd64; ldflags `-X .../internal/cli.Version`; archives name_template produces the SAME url filenames rg's merged formula expects; checksums.txt; brews→spacedock-dev/homebrew-tap; brews.caveats carries the agent-safehouse.dev link; tag-triggered).
  Structural: builds darwin arm64+amd64, CGO_ENABLED=0, ldflags `-X github.com/spacedock-dev/spacedock/internal/cli.Version={{.Version}}`; archives `{{.ProjectName}}_{{.Version}}_{{.Os}}_{{.Arch}}` tar.gz `wrap_in_directory: false`; `checksum: checksums.txt` sha256; `release.github` spacedock-dev/spacedock; `brews` → spacedock-dev/homebrew-tap via HOMEBREW_TAP_TOKEN; release.yml `on: push: tags: v*`, macos-latest, fetch-depth:0, goreleaser-action@v6 `~> v2`. B↔C against merged rg formula (`main:dist/homebrew/Formula/spacedock.rb`, version 0.19.0): template on tag v0.19.0 → `spacedock_0.19.0_darwin_{arm64,amd64}.tar.gz`, byte-for-byte the formula's two url filenames. brews.caveats vs formula caveats: `diff` empty (IDENTICAL), both carry `https://agent-safehouse.dev` — AC-3-on-generated-formula fix confirmed at source.
- DONE: Gates green with REAL captured exit codes; confirm jf flagged goreleaser check / release --snapshot + first-release v0.19.0 tag + HOMEBREW_TAP_TOKEN as CAPTAIN-RUN honestly — verify NO goreleaser pass was fabricated.
  `go test ./...` exit 0 (470 passed/8 pkgs); `go test -race ./...` exit 0 (7 ok pkgs, no DATA RACE); `gofmt -l .` exit 0 (empty); `go vet ./...` exit 0. Deferral is HONEST — the implementation report's "could NOT run in sandbox (disk full)" is accurate and NO goreleaser pass was fabricated (the F9 claim rests on plain `go build`, not goreleaser). goreleaser turned out to be installed in THIS validation env (v2.16.0), so I ran the deferred captain steps for real (see below) rather than only trusting the deferral.

### Validation findings (ran the deferred captain-run goreleaser steps)

goreleaser v2.16.0 IS present here (the implementer's sandbox could not install it — disk full). I executed the deferred steps:
- `goreleaser check` → **exit 2**: "configuration is valid, but uses deprecated properties" — the `brews` key is soft-deprecated (deprecated since v2.10, removal slated for a future major). The implementation report's "expect exit 0" for `goreleaser check` is WRONG; check fails on the deprecation today.
- `goreleaser release --snapshot --clean` → **exit 0, "release succeeded"**: `brews` still FUNCTIONS in v2.16.0 (warns, does not block). It built both darwin targets, wrote `dist/spacedock_0.0.1-snapshot-60989ed_darwin_{arm64,amd64}.tar.gz`, `dist/checksums.txt` (sha256 of both, verified vs `shasum -a 256`), and generated `dist/homebrew/spacedock.rb`.
- RAN the goreleaser-built arm64 binary → `spacedock 0.0.1-snapshot-60989ed (contract 1)` (the goreleaser-stamped snapshot version, NOT the 0.19.0 default) — the ldflags stamp flows through goreleaser end-to-end, F9 proven through the real release tool.
- Generated formula carries the correct per-arch `spacedock_{version}_darwin_{arch}.tar.gz` url filenames + `https://agent-safehouse.dev` caveats. Structural note: goreleaser emits `if Hardware::CPU.intel?/.arm?` blocks vs rg's hand-authored `on_macos/on_arm/on_intel` — functionally equivalent per-arch url+sha256 selection; the `brews:` push overwrites rg's formula on each release with this shape.
- Minor report inaccuracies (non-blocking): archive payload is `README.md` + `spacedock` (goreleaser's default README inclusion), not the "bare-binary payload" the report claims — `spacedock` is at the archive root so `bin.install "spacedock"` still resolves. `dist/` is NOT in `.gitignore`; harmless on a fresh CI `--clean` checkout but leaves an untracked `dist/` on local dry-runs.

### Summary

RECOMMENDATION: **PASSED**. Both ACs hold by behavioral proof. AC-1 (F9): the version stamp flows through both plain `go build` (`spacedock 60989ed`, strict equality vs git-describe) and the real goreleaser snapshot (`spacedock 0.0.1-snapshot-60989ed`), the unstamped default is `spacedock 0.19.0 (contract 1)`, the `-X main.Version` trap is a confirmed silent no-op, and TestVersion + TestVersionContractToken pass. AC-2: goreleaser produced per-arch `spacedock_{version}_darwin_{arm64,amd64}.tar.gz` + checksums.txt at exactly the url shape rg's merged formula consumes, and brews.caveats matches the formula's safehouse link byte-for-byte. All gates green with real exit codes (test/race/gofmt/vet all 0) and the goreleaser deferral was honest (no fabricated pass). Two follow-ups for the captain, neither blocking the merge: (1) `goreleaser check` exits 2 because `brews` is deprecated in v2.16.0 — `brews` still works (snapshot succeeded) but should migrate to `homebrew_casks` before goreleaser's next major removes it; the implementation report's "expect exit 0 from goreleaser check" is incorrect. (2) add `dist/` to `.gitignore`. The first-release `v0.19.0` tag + `HOMEBREW_TAP_TOKEN` secret remain genuine captain/CI-only steps.
