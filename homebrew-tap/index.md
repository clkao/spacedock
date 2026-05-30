---
id: rgsk9fyka3s5ah87rb773a2f
title: Homebrew own-tap — spacedock-dev/homebrew-tap formula
status: backlog
source: sprint — Ship the Launcher slice B (captain, 2026-05-30)
started:
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

## Acceptance criteria (provisional — ideation hardens each)

**AC-1 — formula committed + lints.** `Formula/spacedock.rb` exists in the tap repo; `brew audit --strict` on the formula exits 0.

**AC-2 — install puts spacedock on PATH, --version stamped.** `brew tap spacedock-dev/homebrew-tap && brew install spacedock` → `which spacedock` is brew-managed and `spacedock --version` returns a non-empty stamped version. (Live install may be captain/CI-run; the committed formula + `brew audit --strict` are the in-entity bar.)
