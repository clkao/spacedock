// ABOUTME: Offline-testable end-state helpers for the live cycle test: locate the
// ABOUTME: entity (in place OR archived) and scan the git log for a path-scoped commit.
package ensigncycle

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// frontmatterField anchors the terminal `status: done` frontmatter line the
	// FO writes when the entity reaches the terminal stage. Anchored at the line
	// start so a `status:` mention in prose cannot satisfy it.
	frontmatterField = regexp.MustCompile(`(?im)^status:\s*done\s*$`)
	// verdictSet anchors a finalized `verdict:` line carrying a non-empty value.
	// The exact verdict WORD is FO judgment that varies by model (sonnet wrote
	// `verdict: done`, opus wrote `verdict: passed`); both completed the full
	// cycle. The live test gates on MECHANICAL completion, so it only requires the
	// verdict be SET — `\S` after the colon rejects an empty/whitespace-only
	// `verdict:` (the incomplete-cycle shape) while accepting any decided value.
	// `[^\S\n]*` is horizontal whitespace only so it cannot consume the line break
	// and let `\S` reach into the next frontmatter line.
	verdictSet = regexp.MustCompile(`(?im)^verdict:[^\S\n]*\S.*$`)
)

// A real FO driving the fixture entity to the TERMINAL `done` stage ARCHIVES it:
// the flat `make-it-work.md` moves to `_archive/make-it-work.md`. These helpers
// locate the entity wherever the completed cycle left it and inspect the git log
// for the path-scoped state commit, so the live assertions match the REAL
// completed-and-archived end-state rather than the scripted skeleton's single
// in-place append. They live under the DEFAULT build tags (no //go:build live) so
// the live test reuses them AND an offline unit test exercises the locate/scan
// logic against a staged fixture without spending a model.

// locateEntity returns the contents of the fixture entity after the cycle,
// searching the three end-state locations in order: the original flat path, the
// flat archive `_archive/<slug>.md`, and the folder archive `_archive/<slug>/index.md`.
// found reports whether any of them existed. An INCOMPLETE cycle that never
// archived still resolves the original path; a completed cycle resolves the
// archive. When none exist (the entity vanished entirely), found is false and the
// caller fails loudly rather than asserting against empty content.
func locateEntity(root, slug string) (content string, where string, found bool) {
	candidates := []string{
		filepath.Join(root, slug+".md"),
		filepath.Join(root, "_archive", slug+".md"),
		filepath.Join(root, "_archive", slug, "index.md"),
	}
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			return string(b), p, true
		}
	}
	return "", "", false
}

// someCommitNamesOnly reports whether ANY commit in the entity's history named
// ONLY the entity slug — the path-scoped state-commit invariant at the cycle
// level. A full FO-to-done cycle makes MULTIPLE commits (the ensign's path-scoped
// state commit, then the FO's archive/finalize commits), so HEAD is no longer the
// path-scoped one; this scans the whole log for at least one path-scoped commit
// instead of pinning HEAD. The haiku INCOMPLETE cycle committed `[README.md
// make-it-work.md]` (a sibling sweep, not path-scoped) and produced no other
// commit, so this returns false for it — keeping the live test red on an
// incomplete cycle.
func someCommitNamesOnly(t *testing.T, root, slug string) bool {
	t.Helper()
	// One commit per line, name-only files separated by tabs after a leading
	// marker so per-commit boundaries are unambiguous.
	out := git(t, root, "log", "--pretty=format:@@COMMIT@@", "--name-only")
	target := slug + ".md"
	var files []string
	flush := func() bool {
		if len(files) == 1 && filepath.Base(files[0]) == target {
			return true
		}
		files = files[:0]
		return false
	}
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "@@COMMIT@@" {
			if flush() {
				return true
			}
			continue
		}
		if ln != "" {
			files = append(files, ln)
		}
	}
	return flush()
}
