// ABOUTME: Walk-up workflow discovery and state-checkout detection — find the
// ABOUTME: enclosing commissioned workflow, or name a misdirected state checkout.
package status

import (
	"path/filepath"
	"strings"
)

// discoverWorkflowDir walks up from startDir to the nearest ancestor whose
// README.md frontmatter declares a `commissioned-by: spacedock@` field, the
// same predicate discoverWorkflows uses to recognize a workflow. The first
// match on the way up wins — when workflows are nested, this resolves to the
// innermost enclosing workflow. Returns (dir, true) on a match, ("", false) at
// the filesystem root with no match.
func discoverWorkflowDir(startDir string) (string, bool) {
	d, err := filepath.Abs(startDir)
	if err != nil {
		d = startDir
	}
	for {
		readme := filepath.Join(d, "README.md")
		if isRegularFile(readme) {
			fields := ParseFrontmatter(readme)
			if strings.HasPrefix(fields["commissioned-by"], "spacedock@") {
				return d, true
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", false
		}
		d = parent
	}
}

// stateCheckoutParent reports whether pointedDir is the state checkout of an
// enclosing workflow. It walks up from pointedDir; at each ancestor A it reads
// A/README.md's `state:` field, and if that field's cleaned, non-escaping value
// resolves to the same realpath as pointedDir, A is the definition dir. This
// reuses the same `state:` validation rules as resolveRoots (reject absolute /
// `..`-escaping). Returns (defDir, true) on a match, ("", false) otherwise.
func stateCheckoutParent(pointedDir string) (string, bool) {
	target := realpathOf(pointedDir)
	d, err := filepath.Abs(pointedDir)
	if err != nil {
		d = pointedDir
	}
	for {
		readme := filepath.Join(d, "README.md")
		if isRegularFile(readme) {
			stateValue := strings.TrimSpace(ParseFrontmatter(readme)["state"])
			if stateValue != "" && !filepath.IsAbs(stateValue) {
				cleaned := filepath.Clean(stateValue)
				if cleaned != ".." && !strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
					if realpathOf(filepath.Join(d, cleaned)) == target {
						return d, true
					}
				}
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", false
		}
		d = parent
	}
}
