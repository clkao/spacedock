// ABOUTME: B.2 proof — StateBranch derives spacedock-state/<workflow-dir-basename>
// ABOUTME: by default and honors an explicit README state-branch: override.
package status

import (
	"path/filepath"
	"testing"
)

// TestStateBranchDefaultFromBasename derives the state branch from the workflow
// dir's basename, reproducing the shipped spacedock-state/dev for docs/dev.
func TestStateBranchDefaultFromBasename(t *testing.T) {
	def := t.TempDir()
	wf := filepath.Join(def, "docs", "dev")
	writeFile(t, filepath.Join(wf, "README.md"), "---\nstate: .spacedock-state\n---\n")

	got, err := StateBranch(wf)
	if err != nil {
		t.Fatalf("StateBranch error: %v", err)
	}
	if got != "spacedock-state/dev" {
		t.Fatalf("StateBranch = %q, want spacedock-state/dev", got)
	}
}

// TestStateBranchOverride honors an explicit README state-branch: verbatim.
func TestStateBranchOverride(t *testing.T) {
	wf := t.TempDir()
	writeFile(t, filepath.Join(wf, "README.md"),
		"---\nstate: .spacedock-state\nstate-branch: custom/state-line\n---\n")

	got, err := StateBranch(wf)
	if err != nil {
		t.Fatalf("StateBranch error: %v", err)
	}
	if got != "custom/state-line" {
		t.Fatalf("StateBranch = %q, want custom/state-line (override wins)", got)
	}
}
