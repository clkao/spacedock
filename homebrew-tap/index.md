---
id: rgsk9fyka3s5ah87rb773a2f
title: Homebrew own-tap — spacedock-dev/homebrew-tap formula
status: ideation
source: sprint — Ship the Launcher slice B (captain, 2026-05-30)
started: 2026-05-31T01:32:22Z
completed:
verdict:
score: "0.28"
worktree:
issue:
---

Ship a Homebrew own-tap so `brew install` puts `spacedock` on PATH on a fresh Mac.

## Target
- New repo `spacedock-dev/homebrew-tap` (org already exists), `Formula/spacedock.rb`.
- Consumes pre-built per-platform release tarballs (darwin arm64/amd64) via `url` + `sha256` produced by the release pipeline (`jfk8g2h0h1wsbr3qtncsqsbt`).
- `bin.install "spacedock"`; formula `test do` asserts `spacedock --version` is stamped.
- `brew audit --strict` green.
- safehouse is a RUNTIME dependency, NOT auto-installed — the README install hint covers it.

## Dependencies
- Formula `url`/`sha256` are filled by the release pipeline (C). B can author the formula skeleton in parallel; the live `brew install` smoke waits on C's first release.

## Problem statement

A fresh Mac has no `spacedock` on PATH. We want `brew install` to drop a working,
version-stamped binary so the fresh-install journey starts with a one-liner. v1
mirrors rtk's own-tap pattern (a `spacedock-dev/homebrew-tap` repo with a single
`Formula/spacedock.rb`), with one deliberate divergence: rtk builds from source
(`cargo install` off a source tarball), whereas spacedock installs a **pre-built
per-platform binary** produced by the release pipeline (entity `jf`). No Go
toolchain is required on the user's machine, and the formula has no build step.

## Proposed approach

A binary-only formula. The formula carries a per-platform `url` + `sha256` pointing
at the release tarball, unpacks it, and `bin.install "spacedock"`. There is no
`depends_on "go" => :build` and no `def install` build invocation — the divergence
from rtk's source build is intentional and load-bearing for the contract with `jf`.

`safehouse` is a **runtime** dependency that brew does **not** install: it ships
with Claude Code / Codex tooling, not as a brew formula. The formula does not
declare it; the tap README documents the install hint so a user who lacks it gets
a pointer rather than a silent failure.

### Formula skeleton (the design artifact — placeholders filled by `jf`)

```ruby
class Spacedock < Formula
  desc "Workflow launcher and first-officer dispatch for agentic dev"
  homepage "https://github.com/clkao/spacedock"
  version "0.1.0"   # bumped by the release pipeline to the git-describe tag
  license "Apache-2.0"  # confirm with captain; rtk uses Apache-2.0

  on_macos do
    on_arm do
      url "https://github.com/clkao/spacedock/releases/download/v#{version}/spacedock_#{version}_darwin_arm64.tar.gz"
      sha256 "0" * 64  # PLACEHOLDER — release pipeline writes the real arm64 checksum
    end
    on_intel do
      url "https://github.com/clkao/spacedock/releases/download/v#{version}/spacedock_#{version}_darwin_amd64.tar.gz"
      sha256 "0" * 64  # PLACEHOLDER — release pipeline writes the real amd64 checksum
    end
  end

  def install
    bin.install "spacedock"
  end

  def caveats
    <<~EOS
      spacedock launches host agent tooling (Claude Code / Codex) inside a
      sandbox called "safehouse". safehouse is a RUNTIME dependency and is NOT
      installed by brew. Install it via the host tooling before first launch:
        <safehouse install hint — confirm exact command with captain>
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/spacedock --version")
  end
end
```

Notes on the skeleton:
- `bin.install "spacedock"` assumes the tarball's top-level entry is the bare
  `spacedock` binary (matching `go build ./cmd/spacedock` output). That layout is
  part of the `jf` contract below.
- The `test do` block asserts the brew-tracked `version` string appears in
  `spacedock --version`. Today `--version` emits `spacedock 0.1.0-dev (contract 1)`;
  the released binary must emit the real tag (e.g. `spacedock 0.1.0 ...`) so
  `assert_match version.to_s` holds. This couples the formula `version` line, the
  tag, and the ldflags-stamped binary — see the version-stamp seam below.
- `caveats` (not `depends_on`) is how the safehouse runtime dep surfaces at install
  time, complementing the README hint. brew prints caveats after install.

## Dependency seam on the release pipeline (`jf`)

The formula and the release pipeline must agree on three things. This is the
B↔C contract:

1. **Tarball URL shape.** GitHub Release asset at
   `https://github.com/clkao/spacedock/releases/download/v{version}/spacedock_{version}_darwin_{arch}.tar.gz`
   with `{arch}` ∈ {`arm64`, `amd64`}. If `jf` uses goreleaser defaults, confirm the
   asset name template matches this exactly (goreleaser's default is
   `{{.ProjectName}}_{{.Version}}_{{.Os}}_{{.Arch}}` → `spacedock_0.1.0_darwin_arm64`,
   which lines up). Whoever owns `jf` should pin the goreleaser `name_template` to
   this shape or update this AC if they diverge.
2. **Checksums.** Per-asset sha256, written into the formula's `on_arm`/`on_intel`
   `sha256` lines on each release (the release job bumps the formula). The placeholder
   `"0" * 64` makes the unfilled skeleton lint-detectably wrong, not silently valid.
3. **Tarball payload.** Top-level `spacedock` binary (no nested dir), so
   `bin.install "spacedock"` resolves. goreleaser's default archive wraps binaries
   without a leading directory when `wrap_in_directory` is false — confirm `jf` keeps
   that default.

### Version-stamp seam (correction for `jf`)

`jf`'s ideation says build with `-ldflags "-X main.Version=$(git describe --tags)"`.
That target is wrong for this codebase: the version constant is
`github.com/clkao/spacedock-v1/internal/cli.Version` (`internal/cli/cli.go:17`,
currently `const Version = "0.1.0-dev"`), and `main` has no `Version` var
(`cmd/spacedock/main.go` just calls `cli.Run`). The correct ldflags target is:

```
-ldflags "-X github.com/clkao/spacedock-v1/internal/cli.Version=$(git describe --tags)"
```

(Module path is `github.com/clkao/spacedock-v1` per `go.mod`; release origin is
`clkao/spacedock@next` per `jf`, so the module path may be renamed to
`github.com/clkao/spacedock` on `next` — `jf` must point ldflags at whatever the
package path actually is on the release branch.) There are no git tags yet
(`git describe` currently fails), so the first release must create the seed tag.
This correction is surfaced here for `jf`; the homebrew entity does not own the
ldflags change.

## Tap repo layout + install UX

`spacedock-dev/homebrew-tap` (the `spacedock-dev` org exists; **creating the repo is
a captain action**):

```
spacedock-dev/homebrew-tap/
├── Formula/
│   └── spacedock.rb
└── README.md      # tap usage + safehouse runtime-dep install hint
```

Install flow (the documented UX):

```
brew tap spacedock-dev/homebrew-tap
brew install spacedock
# spacedock --version  → stamped release version
# (safehouse: install separately via host tooling — see README/caveats)
```

## Acceptance criteria

**AC-1 — formula committed + lints (in-entity bar).** `Formula/spacedock.rb` exists
in the tap repo with the binary-only structure above (per-platform `url` + `sha256`,
`bin.install`, `caveats` documenting safehouse, `test do` asserting `version`).
`brew audit --strict --new Formula/spacedock.rb` exits 0 against the formula with
real (or fixture) checksums.
- **Test:** run `brew audit --strict` on the committed formula and observe exit 0.
  This is the pinned in-entity behavioral oracle — it is a static-shape audit that
  needs no live download, so it runs in-entity without waiting on `jf`'s release.

**AC-2 — `brew install` puts spacedock on PATH with a stamped version (live bar).**
`brew tap spacedock-dev/homebrew-tap && brew install spacedock` → `which spacedock`
resolves under the brew prefix and `spacedock --version` returns the non-empty
release version (matching the formula's `version` line).
- **Test:** live `brew install` then `which spacedock` + `spacedock --version`,
  asserting the output matches the formula `version`. **Gated on `jf`'s first
  release** (real url/sha256/binary). May be captain- or CI-run; the in-entity bar
  is AC-1's audit.

**AC-3 — safehouse documented as a non-auto-installed runtime dep.** The tap README
and the formula `caveats` both state that safehouse is a runtime dependency brew does
not install, with an install hint.
- **Test:** static check — README and `caveats` block both contain the safehouse
  hint. Verified by inspection / a grep assertion in the implement stage.

**AC-4 — B↔C contract recorded.** The url shape, checksum injection point, and
tarball payload layout (the three items above) are written into the entity so `jf`
and this formula cannot silently diverge.
- **Test:** this section exists and `jf` references it (cross-entity coordination,
  verified by FO at the gate).

## Test plan

- **AC-1 (in-entity, cheap):** `brew audit --strict` on the committed formula skeleton.
  Cost: seconds, local, no network beyond brew's own taps. This is the riskiest path
  to validate first (formula shape correctness) and it does not depend on `jf`.
- **AC-2 (live, gated on `jf`):** real `brew tap`/`brew install` + `--version` check.
  Cost: one release cycle + an install; captain/CI-run. Deferred until `jf` ships its
  first release with real artifacts.
- **AC-3/AC-4 (static):** inspection / grep of README, `caveats`, and this contract
  section. Cost: trivial.
- No Go unit tests apply — there is no spacedock-side code change in this entity; the
  binary-only formula has no build logic to unit test. The artifact is Ruby + docs,
  so the audit + live install are the right-altitude proofs.

## Open questions for the gate

- **License:** rtk uses Apache-2.0; confirm spacedock's license string for the formula.
- **safehouse install hint:** the exact command is a placeholder — needs the real
  host-tooling install command from the captain.
- **Module path on `next`:** if the release origin renames the module to
  `github.com/clkao/spacedock`, the ldflags target (surfaced to `jf`) follows the
  package path on that branch.

## Stage Report: ideation

- DONE: The formula is a concrete artifact: a `Formula/spacedock.rb` skeleton with `bin.install`, on_macos arm/intel `url`+`sha256` placeholders (filled by the release pipeline), and a `test do` asserting a stamped `spacedock --version` — with the `brew audit --strict` pass pinned as the in-entity behavioral oracle.
  Skeleton in entity body under "Formula skeleton"; binary-only (no build step, divergence from rtk's `cargo install` documented); `test do` asserts `version.to_s` in `--version`; AC-1 pins `brew audit --strict` as the in-entity oracle.
- DONE: Tap + install UX pinned: `spacedock-dev/homebrew-tap` repo layout; the `brew tap spacedock-dev/homebrew-tap && brew install spacedock` flow; safehouse documented as a NON-auto-installed runtime dep + the install hint.
  "Tap repo layout + install UX" section gives the repo tree + flow; safehouse is documented via formula `caveats` (not `depends_on`) and README, AC-3 + open question track the exact hint as a captain input.
- DONE: The dependency seam on the release pipeline (jf) is named: the exact release-artifact url shape + checksums the formula consumes, so B and C agree on the contract.
  "Dependency seam on the release pipeline (`jf`)" section pins the 3-part contract (url shape, sha256 injection point, tarball payload layout) as AC-4; also surfaced a correction to jf's ldflags target (version const is `internal/cli.Version`, not `main.Version`; no tags exist yet).

### Summary

Hardened ideation for the binary-only Homebrew own-tap. The key design decision: spacedock's formula installs a pre-built per-platform binary (per-arch url+sha256, `bin.install`, no Go toolchain) rather than building from source as rtk does — this is the load-bearing divergence and the seam to the `jf` release pipeline. ACs were rewritten so the cheap, jf-independent `brew audit --strict` is the in-entity oracle and the live `brew install` is the gated-on-jf bar. Two corrections were surfaced for the captain/jf: the ldflags target must be `internal/cli.Version` (not `main.Version`), and no git tags exist yet so the first release must seed the tag. Open questions left for the gate: license string and the exact safehouse install-hint command.
