# spacedock-dev/homebrew-tap

Homebrew own-tap for `spacedock`. Installs a pre-built per-platform binary
produced by the release pipeline (no Go toolchain required on the user's
machine).

## Install

```bash
brew tap spacedock-dev/homebrew-tap
brew install spacedock
spacedock --version   # prints the stamped release version
```

## safehouse runtime dependency

`spacedock` launches host agent tooling (Claude Code / Codex) inside a sandbox
called **safehouse**. safehouse is a *runtime* dependency and is **NOT**
installed by brew — it ships with the host tooling, not as a brew formula.
Install it through the host tooling before first launch. See the safehouse
install docs:

<https://github.com/spacedock-dev/safehouse#install>

The formula surfaces the same hint via `brew install`'s caveats output.

## Layout

```
homebrew-tap/
├── Formula/
│   └── spacedock.rb
└── README.md
```

This directory (`dist/homebrew/`) stages the tap contents inside the spacedock
v1 repo until the `spacedock-dev/homebrew-tap` repo is created. On tap creation,
copy `Formula/spacedock.rb` and this `README.md` into the repo root.

## Release contract (B↔C, with the `jf` release pipeline)

The formula and the release pipeline must agree on three things:

1. **Tarball URL shape.** GitHub Release asset at
   `https://github.com/spacedock-dev/spacedock/releases/download/v{version}/spacedock_{version}_darwin_{arch}.tar.gz`
   with `{arch}` ∈ {`arm64`, `amd64`}. goreleaser's default name template
   `{{.ProjectName}}_{{.Version}}_{{.Os}}_{{.Arch}}` produces exactly this
   (`spacedock_0.19.0_darwin_arm64`); pin `name_template` to that shape.
2. **Checksum injection point.** Per-asset sha256 written into the formula's
   `on_arm`/`on_intel` `sha256` lines on each release. The committed skeleton
   ships an all-zero placeholder so an unfilled formula is detectably wrong, not
   silently valid. The release job (or goreleaser's `brews:` block) bumps both
   the `version` line and the two `sha256` lines.
3. **Tarball payload layout.** Top-level bare `spacedock` binary (no nested
   directory), so `bin.install "spacedock"` resolves. Keep goreleaser's archive
   default (`wrap_in_directory: false`).

### Version-stamp seam

`spacedock --version` emits `spacedock {Version} (contract {N})`, where
`Version` is `github.com/spacedock-dev/spacedock/internal/cli.Version`
(`internal/cli/cli.go:17`). The released binary must be built with:

```
-ldflags "-X github.com/spacedock-dev/spacedock/internal/cli.Version=$(git describe --tags)"
```

so `spacedock --version` reports the real tag and the formula's
`test do { assert_match version.to_s, ... }` holds. The formula `version` line,
the git tag, and the ldflags-stamped binary must all agree. There are no git
tags yet, so the first release must create the seed tag.
