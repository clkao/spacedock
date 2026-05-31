---
id: rgsk9fyka3s5ah87rb773a2f
title: Homebrew own-tap — spacedock-dev/homebrew-tap formula
status: validation
source: sprint — Ship the Launcher slice B (captain, 2026-05-30)
started: 2026-05-31T01:32:22Z
completed:
verdict:
score: "0.28"
worktree: /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-homebrew-tap
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

## Stage Report: implementation

- DONE: Author `Formula/spacedock.rb` (staged in the v1 repo, e.g. dist/homebrew/) with per-platform url+sha256 at spacedock-dev/spacedock releases, `bin.install "spacedock"`, `test do` asserting stamped `spacedock --version`, `license "Apache-2.0"`, and safehouse documented via `caveats` linking to safehouse's docs.
  `dist/homebrew/Formula/spacedock.rb` (commit bbdab4d, branch spacedock-ensign/homebrew-tap); url path uses the migrated module `github.com/spacedock-dev/spacedock` (per go.mod), `version "0.1.0"` + all-zero sha256 placeholders for jf, `caveats` links https://github.com/spacedock-dev/safehouse#install (link, not a pinned command). `ruby -c` parse: Syntax OK.
- FAILED: `brew audit --strict` on the formula passes (exit 0) — capture the rc.
  Could NOT be run in-sandbox: every write under `/opt/homebrew` returns "Operation not permitted" (env-level block, not perms — dir is clkao:admin 0755, no file flags; sandbox-off flag does not lift it). `brew audit`/`brew style` both abort at bootstrap trying to write the vendored-gem marker `vendor/bundle/ruby/4.0.0/.homebrew_vendor_version` into the read-only prefix, before any linting. AUDIT_RC=1 is the bootstrap error, NOT a formula verdict. Separately, the all-zero placeholder `sha256` is expected to fail a real offline audit as a bad checksum until jf injects real values — so a fully-green audit is gated on jf's first release (real urls + checksums). Deferred to a captain-run step. Exact command: `brew audit --strict /Users/clkao/git/spacedock-research/spacedock-v1/.worktrees/spacedock-ensign-homebrew-tap/dist/homebrew/Formula/spacedock.rb; echo "AUDIT_RC=$?"`.
- DONE: Document the B↔C contract (release `url` shape + sha256 injection point + tarball payload layout) so it agrees with jf's release pipeline.
  `dist/homebrew/README.md` "Release contract (B↔C…)" section: url `…/releases/download/v{version}/spacedock_{version}_darwin_{arch}.tar.gz` (goreleaser default name_template), sha256 injection into `on_arm`/`on_intel` lines per release, top-level bare-binary payload (`wrap_in_directory: false`), plus the version-stamp ldflags target corrected to `github.com/spacedock-dev/spacedock/internal/cli.Version`.

### Summary

Authored the binary-only own-tap formula + tap README, staged under `dist/homebrew/` in the v1 repo (the `spacedock-dev/homebrew-tap` repo does not exist yet; captain/FO creates it). Formula uses the migrated release path `github.com/spacedock-dev/spacedock` (the go.mod module name; the ideation body's `clkao/spacedock-v1` predates the migration), per-platform url+sha256 placeholders for jf, `caveats` + README both carrying the safehouse runtime-dep hint (AC-3), and the full B↔C release contract recorded (AC-4). The pinned in-entity oracle — `brew audit --strict` exit 0 — could NOT run here: this sandbox blocks all writes to the read-only `/opt/homebrew` prefix, so brew aborts at vendored-gem bootstrap before evaluating the formula. I refused to fabricate a pass. AC-1's audit is flagged for a captain-run step (exact command in the FAILED line), and is additionally gated on jf cutting a release with real urls + checksums (the all-zero placeholder sha256 will legitimately fail audit until then). `ruby -c` confirms the formula is syntactically valid.

## Stage Report: implementation (cycle 2)

- DONE: Captain version bump to 0.19.0.
  `version "0.19.0"` (commit 31fd552); single `version` line propagates via `#{version}` to both urls → `.../v0.19.0/spacedock_0.19.0_darwin_{arm64,amd64}.tar.gz`; README contract example bumped to match; `ruby -c` Syntax OK. Url naming kept underscore form per jf's actual goreleaser `name_template` (the loose dash example in the bump request is superseded by jf's config, which is the source of truth).
- DONE: Cross-checked formula against jf's committed `.goreleaser.yaml` (release-pipeline worktree) — B↔C contract verified, not assumed.
  jf's `archives.name_template "{{.ProjectName}}_{{.Version}}_{{.Os}}_{{.Arch}}"` + `wrap_in_directory: false` + ldflags `-X github.com/spacedock-dev/spacedock/internal/cli.Version={{.Version}}` all agree with my formula's urls, `bin.install "spacedock"`, and version-stamp seam. desc string matches jf's `brews.description` verbatim.
- FAILED: AC-3 link survives jf's generated formula.
  jf's `brews.caveats` block GENERATES the released formula's caveats and DROPS the safehouse docs link (it ends at "...before first launch." with no URL). My staged formula + README carry the link as AC-3 requires, but the per-release goreleaser-generated formula would overwrite mine without it. Flagged to jf so `brews.caveats` keeps the safehouse docs link; otherwise released formulas silently lose AC-3's link.

### Summary (cycle 2)

Bumped the formula to the captain's 0.19.0 and verified the B↔C contract against jf's actual `.goreleaser.yaml` rather than the ideation prose — name_template, wrap_in_directory, ldflags target, and desc all agree. One divergence surfaced for jf: goreleaser's `brews:` block regenerates the formula on each release and its `caveats` drops the safehouse docs link, which would violate AC-3 on released formulas; flagged to jf to add the link to `brews.caveats`. The `brew audit --strict` oracle is still captain-gated (sandbox blocks `/opt/homebrew`) and the bump does not affect that version-neutral structural audit.

## Stage Report: implementation (cycle 3)

- DONE: Formula + tap README + B↔C contract committed on the branch.
  `dist/homebrew/Formula/spacedock.rb` (version 0.19.0) + `dist/homebrew/README.md` on branch `spacedock-ensign/homebrew-tap`, commits bbdab4d (initial) + 31fd552 (0.19.0 bump). `ruby -c` Syntax OK.
- DONE: AC-1 (`brew audit --strict`) recorded as a CAPTAIN-RUN post-tap check.
  Verification method changed by the current brew: `brew audit [path]` is disabled — audit is name-only (`brew audit <tap>/<name>`), so a loose-file audit on the staged path cannot run (independent of this sandbox's `/opt/homebrew` write block, which also blocks brew here). Authoritative AC-1 check is run by the captain after the FO creates `spacedock-dev/homebrew-tap` and pushes the formula. Exact command: `brew audit --strict spacedock-dev/homebrew-tap/spacedock; echo "AUDIT_RC=$?"`. The rc is relayed into the validation gate by the FO. Not faked — no in-entity green claimed.

### Summary (cycle 3)

Per the FO's revised plan: the current brew disables path-form `brew audit`, so AC-1 cannot be a loose-file audit — it becomes a captain-run name-based audit (`brew audit --strict spacedock-dev/homebrew-tap/spacedock`) after the FO creates the tap repo and pushes this formula. Formula (0.19.0) + tap README + B↔C contract are committed on `spacedock-ensign/homebrew-tap` (bbdab4d, 31fd552); `ruby -c` is the only in-entity static check I can run (Syntax OK). The AC-1 rc and the AC-2 live `brew install` both flow through the FO/captain post-tap; no fabricated audit pass.

## Stage Report: implementation (cycle 4)

- DONE: AC-1 satisfied — captain ran `brew audit --strict spacedock-dev/homebrew-tap/spacedock` → AUDIT_RC=0.
  Reported by the FO from the captain's run against the tapped formula; the audit oracle is green.
- DONE: safehouse docs URL pinned to the captain's canonical link.
  `https://agent-safehouse.dev` replaces the provisional `github.com/spacedock-dev/safehouse#install` in both the formula `caveats` and the README (commit 1b07d25). String-only, audit-neutral (no re-audit needed). grep confirms the provisional URL is gone and the canonical one appears in both files; `ruby -c` Syntax OK.

### Summary (cycle 4)

AC-1's audit oracle is green: the captain's `brew audit --strict spacedock-dev/homebrew-tap/spacedock` returned 0. Corrected the safehouse docs link to the captain-pinned `https://agent-safehouse.dev` in the formula `caveats` and the README (commit 1b07d25 on `spacedock-ensign/homebrew-tap`) — a string-only, audit-neutral change. The FO re-pushes the corrected formula to the tap and merges. Still pending (external, post-first-release): AC-2 live `brew install`, and jf's `brews.caveats` keeping the safehouse link so released formulas don't drop it.
