// ABOUTME: The single shared `state:` interpreter — classifyState maps the README
// ABOUTME: state: field to a backend mode, and stateBranch derives the state branch.
package status

import (
	"fmt"
	"path/filepath"
	"strings"
)

// inlineSentinel is the explicit-inline token for the README `state:` field. The
// `$` sigil reads as "not a path" at a glance and cannot collide with a v0 child
// checkout name (the validator rejects absolute and `..`-escaping values, and a
// real checkout name never leads with `$`). An empty `state:` means the same
// thing — inline — as the backward-compatible default.
const inlineSentinel = "$inline"

// StateMode names how a workflow's mutable entity state is laid out, derived from
// the README `state:` field.
type StateMode int

const (
	// StateInline keeps entities beside the README on the same branch (single-root).
	// Both an empty `state:` and the explicit $inline sentinel resolve here.
	StateInline StateMode = iota
	// StateSplitRoot keeps entities in a separate state checkout at the `state:`
	// relative path (split-root).
	StateSplitRoot
)

// ClassifyState is the SINGLE interpreter of the README `state:` field, consumed
// by every site that reads it (resolveRoots, splitRootStateCheckout, the
// discoverWorkflows prune, stateCheckoutParent). It runs after TrimSpace:
//
//   - empty / whitespace → inline, no path to join (backward-compatible default).
//   - the $inline sentinel → inline, no path to join (explicit inline).
//   - any other value → split-root, after rejecting an absolute or `..`-escaping
//     path. The returned relPath is filepath.Clean'd, ready to join under the
//     definition dir.
//
// Centralizing the absolute/`..` rejection here means the inline sentinel is never
// treated as an ordinary relative path and joined into a literal `…/$inline` dir
// at any read site.
func ClassifyState(stateValue string) (StateMode, string, error) {
	value := strings.TrimSpace(stateValue)
	if value == "" || value == inlineSentinel {
		return StateInline, "", nil
	}
	if filepath.IsAbs(value) {
		return StateInline, "", fmt.Errorf("state: must be a path relative to the workflow README directory, not absolute: %s", value)
	}
	cleaned := filepath.Clean(value)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return StateInline, "", fmt.Errorf("state: must not escape the workflow README directory: %s", value)
	}
	return StateSplitRoot, cleaned, nil
}

// StateBranch returns the orphan state branch for a split-root workflow. By
// default it derives from the workflow dir's basename
// (spacedock-state/<basename>), reproducing the shipped spacedock-state/dev for
// docs/dev. An explicit README `state-branch:` field always wins verbatim.
func StateBranch(workflowDir string) (string, error) {
	fm := ParseFrontmatter(filepath.Join(workflowDir, "README.md"))
	if override := strings.TrimSpace(fm["state-branch"]); override != "" {
		return override, nil
	}
	base := filepath.Base(filepath.Clean(workflowDir))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("cannot derive state-branch from workflow dir: %s", workflowDir)
	}
	return "spacedock-state/" + base, nil
}
