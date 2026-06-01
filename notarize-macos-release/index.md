---
id: 5wcpy828kvytdgvwgc8rfasp
title: macOS Gatekeeper blocks the cask-installed binary — notarize the release (or document the xattr workaround)
status: backlog
source: captain (2026-06-01) — confirmed root cause + fix; fresh `brew install --cask` dies on first launch
started:
completed:
verdict:
score: 0.25
worktree:
issue:
---

A fresh Homebrew-cask install of the released binary is **dead on first launch** on macOS: the
binary is adhoc/linker-signed only (`codesign` identifier `a.out`), the cask download sets the
`com.apple.quarantine` xattr, and Gatekeeper refuses to run an un-notarized quarantined binary. This
breaks the "fresh install works off `next`" goal — a new user's first launch fails.

**Confirmed workaround (captain):** `xattr -d com.apple.quarantine $(brew --prefix)/bin/spacedock`
→ runs fine.

**FO-confirmed gaps (on `next`):** `.goreleaser.yaml` has NO `signs:`/`notarize:` block; `release.yml`
carries NO Developer-ID / notarytool secrets; the `homebrew_casks` `caveats` do NOT mention the
xattr workaround. So the binary ships unsigned-for-distribution and undocumented.

## Two fixes (ship the workaround now; notarize properly when the cert is available)

1. **Immediate, zero-cost — document the workaround in the cask caveats.** Add the
   `xattr -d com.apple.quarantine ...` line to the `homebrew_casks` `caveats` in `.goreleaser.yaml`
   so a fresh installer is told how to unblock first launch. Unblocks install today.
2. **Proper — notarize in the GoReleaser release.** Developer ID Application sign + `notarytool`
   notarize + `staple`, wired into `.goreleaser.yaml` (a `signs:`/post-build notarize step) and
   `release.yml` (the Developer-ID `.p12` + either `APPLE_ID`/`TEAM_ID`/app-specific-password or an
   App Store Connect API key as repo secrets). Removes the Gatekeeper block entirely — no xattr needed.

**Captain dependency:** the proper fix needs an **Apple Developer ID Application certificate** +
notarytool credentials (an Apple Developer account). That is a captain-provided secret/cert — the FO
can wire the GoReleaser/CI plumbing but cannot obtain the cert.

## Acceptance criteria (provisional — ideation hardens)

**AC-1 — A fresh cask install launches without a Gatekeeper block and without manual xattr.**
Verified by: `brew install --cask` of the released binary on a clean macOS user, then launch — no
"cannot be opened because the developer cannot be verified," no manual `xattr` needed. (The proper-fix
end state.)

**AC-2 — The released binary is Developer-ID-signed + notarized + stapled.**
Verified by: `spctl -a -vvv $(brew --prefix)/bin/spacedock` reports accepted/Developer ID, and
`codesign -dv --verbose` / `stapler validate` show the notarization ticket.

**AC-fallback — Until notarization lands, the cask caveats document the xattr workaround**, and an
install followed by the documented command launches cleanly.

## Notes
Install-blocking; undermines the recalibrated sprint goal #2 (fresh install off `spacedock-dev/spacedock@next`).
Touches `.goreleaser.yaml` (caveats + signs/notarize), `.github/workflows/release.yml` (secrets), and
coordinates with the homebrew-tap (the cask consumer). Recommend shipping AC-fallback (the caveat)
immediately and sequencing AC-2 (notarize) once the captain provides the Developer ID cert.
