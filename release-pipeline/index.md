---
id: jfk8g2h0h1wsbr3qtncsqsbt
title: Release pipeline — version-stamped darwin binaries + GitHub Release (closes F9)
status: backlog
source: sprint — Ship the Launcher slice C (captain, 2026-05-30)
started:
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
