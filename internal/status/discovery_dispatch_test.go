// ABOUTME: AC-1/AC-2 dispatch tests — no-flag discovery renders the enclosing
// ABOUTME: workflow; no-workflow / state-checkout misdirection yield named errors.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	noWorkflowErr        = "Error: no Spacedock workflow here — pass --workflow-dir or run inside a workflow\n"
	stateCheckoutErrHead = "Error: this is a state checkout; point --workflow-dir at the definition dir (the one whose README declares state:): "
)

// TestDiscoveryRendersEnclosingWorkflow (AC-1) runs with NO --workflow-dir from
// a deep cwd inside a commissioned workflow and asserts the workflow renders,
// stage-ordered, exit 0.
func TestDiscoveryRendersEnclosingWorkflow(t *testing.T) {
	def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md":         "---\nstatus: implementation\n---\n",
		"refactor-dispatch.md": "---\nstatus: ideation\n---\n",
	})
	deep := filepath.Join(state, "add-login", "reports")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, deep, env)
	if code != 0 {
		t.Fatalf("no-flag discovery exit=%d stderr=%q", code, stderr)
	}
	_ = def
	slugs := tableSlugs(t, out)
	if !equalStrings(slugs, []string{"refactor-dispatch", "add-login"}) {
		t.Fatalf("slugs = %v, want stage-ordered [refactor-dispatch add-login]\n%s", slugs, out)
	}
}

// TestNoWorkflowHereError (AC-1/AC-2) runs with no --workflow-dir from an
// empty, non-enclosed tempdir and asserts the exact no-workflow stderr, exit 1.
func TestNoWorkflowHereError(t *testing.T) {
	empty := t.TempDir()
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, empty, env)
	if code != 1 {
		t.Fatalf("no-workflow exit=%d, want 1 (stdout=%q stderr=%q)", code, out, stderr)
	}
	if stderr != noWorkflowErr {
		t.Fatalf("stderr = %q, want %q", stderr, noWorkflowErr)
	}
}

// TestStateCheckoutPointedAtError (AC-2, M-9 + M-4) materializes a split-root
// with sd-b32 entities and NO state README, points --workflow-dir AT the state
// checkout under --validate, and asserts the named state-checkout error (ending
// in the def dir), exit 1, with the pre-change `non-numeric sequential id`
// symptom GONE.
func TestStateCheckoutPointedAtError(t *testing.T) {
	readme := `---
commissioned-by: spacedock@1
id-style: sd-b32
state: .spacedock-state
stages:
  states:
    - name: ideation
      initial: true
    - name: done
      terminal: true
---

# SD-B32 Split-Root
`
	def, state := buildSplitRoot(t, readme, map[string]string{
		"alpha.md": "---\nid: c0p9v8u7t6s5r4q3p2n1m0kj\nstatus: ideation\n---\n",
		"beta.md":  "---\nid: 9z8y7x6w5v4u3t2s1r0qp0nm\nstatus: ideation\n---\n",
	})
	env := pinnedEnv(t)

	// Guard: the state checkout carries no README of its own (M-9 — only then
	// does sd-b32 default to sequential and surface the misdiagnosis symptom).
	if _, err := os.Lstat(filepath.Join(state, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("state checkout must have no README.md, lstat err=%v", err)
	}

	// M-4: the regression assertion only fires under --validate, so pin that
	// command context.
	out, stderr, code := runNative(t, def, env, "--workflow-dir", state, "--validate")
	if code != 1 {
		t.Fatalf("state-checkout --validate exit=%d, want 1 (stdout=%q stderr=%q)", code, out, stderr)
	}
	if !strings.HasPrefix(stderr, stateCheckoutErrHead) {
		t.Fatalf("stderr = %q, want prefix %q", stderr, stateCheckoutErrHead)
	}
	if !strings.HasSuffix(strings.TrimRight(stderr, "\n"), def) {
		t.Fatalf("stderr = %q, want it to end in the def dir %q", stderr, def)
	}
	// Regression guard: the pre-change symptom must be gone.
	if strings.Contains(stderr, "non-numeric sequential id") {
		t.Fatalf("stderr still contains the pre-change symptom:\n%s", stderr)
	}
}

// TestExplicitWorkflowDirSkipsDiscovery proves an explicit --workflow-dir at a
// commissioned workflow is used verbatim and discovery is not consulted: the
// run renders the explicit workflow even though cwd is an unrelated empty dir.
func TestExplicitWorkflowDirSkipsDiscovery(t *testing.T) {
	def, _ := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md": "---\nstatus: ideation\n---\n",
	})
	unrelated := t.TempDir()
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, unrelated, env, "--workflow-dir", def)
	if code != 0 {
		t.Fatalf("explicit --workflow-dir exit=%d stderr=%q", code, stderr)
	}
	if got := sortedCopy(tableSlugs(t, out)); !equalStrings(got, []string{"add-login"}) {
		t.Fatalf("slugs = %v, want [add-login]\n%s", got, out)
	}
}

// TestExplicitEmptyDirUnchanged proves the design's deliberate choice: an
// explicit --workflow-dir at an empty non-workflow dir is NOT reclassified as a
// no-workflow error — it falls through to today's empty-table behavior (exit 0).
func TestExplicitEmptyDirUnchanged(t *testing.T) {
	empty := t.TempDir()
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, empty, env, "--workflow-dir", empty)
	if code != 0 {
		t.Fatalf("explicit empty-dir exit=%d, want 0 (unchanged) stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("explicit empty-dir stderr = %q, want empty (no reclassification)", stderr)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "SLUG") {
		t.Fatalf("explicit empty-dir should render an empty table header:\n%s", out)
	}
}
