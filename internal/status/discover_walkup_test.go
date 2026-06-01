// ABOUTME: Walk-up discovery + state-checkout detection tests — discoverWorkflowDir
// ABOUTME: finds the enclosing workflow; stateCheckoutParent names a misdirected dir.
package status

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverWorkflowDirWalksUp proves discoverWorkflowDir finds the nearest
// ancestor whose README is commissioned, starting from a deep cwd.
func TestDiscoverWorkflowDirWalksUp(t *testing.T) {
	def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md": "---\nstatus: ideation\n---\n",
	})
	deep := filepath.Join(state, "add-login", "reports")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := DiscoverWorkflowDir(deep)
	if !ok {
		t.Fatalf("DiscoverWorkflowDir(%q) = ok false, want true", deep)
	}
	if realpathOf(got) != realpathOf(def) {
		t.Fatalf("DiscoverWorkflowDir(%q) = %q, want %q", deep, got, def)
	}
}

// TestDiscoverWorkflowDirMiss proves a tree with no commissioned README yields
// ok false.
func TestDiscoverWorkflowDirMiss(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := DiscoverWorkflowDir(deep); ok {
		t.Fatalf("DiscoverWorkflowDir(%q) = ok true, want false (no commissioned README)", deep)
	}
}

// TestDiscoverWorkflowDirInnermostWins is the M-1 nested-workflow resolution
// test: when two commissioned workflows are nested, the walk-up resolves to the
// INNER one (first match wins on the way up).
func TestDiscoverWorkflowDirInnermostWins(t *testing.T) {
	outer := t.TempDir()
	writeFile(t, filepath.Join(outer, "README.md"), splitRootReadme)
	inner := filepath.Join(outer, "sub", "inner")
	writeFile(t, filepath.Join(inner, "README.md"), splitRootReadme)
	deep := filepath.Join(inner, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := DiscoverWorkflowDir(deep)
	if !ok {
		t.Fatalf("DiscoverWorkflowDir(%q) = ok false, want true", deep)
	}
	if realpathOf(got) != realpathOf(inner) {
		t.Fatalf("DiscoverWorkflowDir(%q) = %q, want innermost %q (first match wins)", deep, got, inner)
	}
}

// TestStateCheckoutParentDetects proves stateCheckoutParent walks up from a
// state checkout to the definition dir whose README declares state: pointing
// back at it.
func TestStateCheckoutParentDetects(t *testing.T) {
	def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md": "---\nstatus: ideation\n---\n",
	})
	got, ok := stateCheckoutParent(state)
	if !ok {
		t.Fatalf("stateCheckoutParent(%q) = ok false, want true", state)
	}
	if realpathOf(got) != realpathOf(def) {
		t.Fatalf("stateCheckoutParent(%q) = %q, want def dir %q", state, got, def)
	}
}

// TestStateCheckoutParentMiss proves an ordinary dir with no enclosing state:
// README yields ok false.
func TestStateCheckoutParentMiss(t *testing.T) {
	dir := t.TempDir()
	if _, ok := stateCheckoutParent(dir); ok {
		t.Fatalf("stateCheckoutParent(%q) = ok true, want false", dir)
	}
}
