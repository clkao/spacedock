// ABOUTME: M-2 site-level negative — the $inline sentinel is never joined as a
// ABOUTME: literal path at resolveRoots or the discoverWorkflows prune.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inlineSentinelReadme is a commissioned slug-style workflow declaring the
// explicit-inline sentinel, with an entity beside the README (inline layout).
const inlineSentinelReadme = `---
commissioned-by: spacedock@1
id-style: slug
state: $inline
stages:
  states:
    - name: ideation
      initial: true
    - name: done
      terminal: true
---

# Inline Sentinel Workflow
`

// TestResolveRootsInlineSentinelNoLiteralJoin pins the M-2 negative at the
// resolveRoots site: a `state: $inline` workflow resolves entityDir ==
// definitionDir (no `/$inline` suffix), exactly like an absent state: field.
func TestResolveRootsInlineSentinelNoLiteralJoin(t *testing.T) {
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), inlineSentinelReadme)

	r, err := resolveRoots(def, "")
	if err != nil {
		t.Fatalf("resolveRoots returned error: %v", err)
	}
	if r.entityDir != def {
		t.Fatalf("entityDir = %q, want %q (inline sentinel must not join /$inline)", r.entityDir, def)
	}
	if strings.Contains(r.entityDir, inlineSentinel) || strings.Contains(r.entityDirSpelling, inlineSentinel) {
		t.Fatalf("entityDir/spelling references the literal sentinel: dir=%q spelling=%q", r.entityDir, r.entityDirSpelling)
	}
}

// TestDiscoverWorkflowsInlineSentinelPrunesNothing pins the M-2 negative at the
// discoverWorkflows prune site: a `state: $inline` workflow is discovered once,
// and the prune scan creates/references no `…/$inline` path on disk.
func TestDiscoverWorkflowsInlineSentinelPrunesNothing(t *testing.T) {
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), inlineSentinelReadme)
	writeFile(t, filepath.Join(def, "thing.md"), "---\nstatus: ideation\n---\n# Thing\n")

	got := discoverWorkflows(def)
	if len(got) != 1 || realpathOf(got[0]) != realpathOf(def) {
		t.Fatalf("discovery = %v, want exactly [%s]", got, def)
	}
	if _, err := os.Stat(filepath.Join(def, inlineSentinel)); !os.IsNotExist(err) {
		t.Fatalf("a literal %q path exists under the definition dir (stat err=%v)", inlineSentinel, err)
	}
}

// TestStateCheckoutParentInlineSentinelNeverMatches pins the M-2 negative at the
// stateCheckoutParent site: a `state: $inline` README never claims any pointed
// dir as its state checkout (the sentinel is not a path).
func TestStateCheckoutParentInlineSentinelNeverMatches(t *testing.T) {
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), inlineSentinelReadme)
	// A child dir under the def that a buggy literal join could resolve to.
	child := filepath.Join(def, inlineSentinel)
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := stateCheckoutParent(child); ok {
		t.Fatalf("stateCheckoutParent matched the literal %q dir; the sentinel must not be treated as a path", inlineSentinel)
	}
}
