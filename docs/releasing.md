# Releasing Spacedock

Releases are cut from `next` (the source of truth). `origin/main` is vestigial —
do not push or merge to it. A release is a `vX.Y.Z` annotated-tag push, which
triggers `.github/workflows/release.yml`.

## What the tag push does

`release.yml` runs one goreleaser job on `macos-latest` that:

- cross-builds the darwin arm64 + amd64 tarballs and `checksums.txt`, stamping
  `git describe --tags` into `internal/cli.Version` (so the binary reports the
  release version);
- publishes the GitHub Release with those assets;
- bumps the `spacedock-dev/homebrew-tap` cask (via the `HOMEBREW_TAP_TOKEN`
  secret — a PAT, since the default `GITHUB_TOKEN` can't write cross-repo);
- stamps the plugin manifests' `version` on `next`
  (`spacedock-release stamp-version`, idempotent).

Pushing the tag is therefore by itself sufficient to stamp `next` — the manual
bump in the steps below just pre-stamps (release.yml then finds it already done
and no-ops). Its real value is producing a reviewable annotated-tag changelog
before publishing.

## Cutting a release

1. Ensure all release content is merged to `next`. Choose the version `X.Y.Z`.

2. Create a release worktree off `next`:

   ```bash
   git worktree add .worktrees/release-X.Y.Z -b release/X.Y.Z origin/next
   ```

3. Bump the two plugin manifests with the release tool, then commit:

   ```bash
   go run ./cmd/spacedock-release stamp-version X.Y.Z .claude-plugin/plugin.json .codex-plugin/plugin.json
   git commit -m "release: bump version to spacedock@X.Y.Z" -- .claude-plugin/plugin.json .codex-plugin/plugin.json
   ```

4. Write a changelog. Summarize the commits since the last tag into plain text:

   ```bash
   git log $(git describe --tags --abbrev=0 origin/next)..HEAD --oneline
   ```

   One sentence naming the release theme, then user-value-led `- ` bullets (lead
   with what upgrading gives you). Ignore workflow-state churn
   (dispatch/advance/archive/mod-block/pr/report commits).

5. Create the annotated tag (local, nothing pushed):

   ```bash
   git tag -a vX.Y.Z -F <changelog-file>
   ```

6. Review:

   ```bash
   git show vX.Y.Z
   git diff origin/next..release/X.Y.Z
   ```

   To amend the changelog: `git tag -d vX.Y.Z` then re-tag.

7. Publish (after confirmation):

   ```bash
   git push origin release/X.Y.Z:next   # fast-forwards next with the stamp
   git push origin vX.Y.Z               # triggers release.yml
   ```

   NEVER push `origin/main`.

8. Clean up:

   ```bash
   git worktree remove .worktrees/release-X.Y.Z
   git branch -d release/X.Y.Z
   ```

## Notes

- Don't stamp the version via a pull request; the release branch + annotated tag
  IS the mechanism.
- macOS binaries are adhoc-signed, not yet notarized; the Homebrew cask's
  postflight strips the `com.apple.quarantine` xattr as the interim Gatekeeper
  fix until Developer-ID notarization lands.
- This flow is the v1 adaptation of the original `scripts/release.sh` from the
  upstream spacedock plugin repo.
