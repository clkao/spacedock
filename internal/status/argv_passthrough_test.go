// ABOUTME: AC-1 argv-passthrough parity — the launcher forwards argv verbatim so
// ABOUTME: the vendored script applies its own semantics (mid-set truncation, unknowns).
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestMidSetUnknownTokenTruncates locks the mid-set truncation case: a token
// starting with -- terminates the --set field-list parse, so
// `--set <slug> --bogus status=done` drops status=done, exits 1 with the
// "requires at least one field=value" error, and leaves the entity unchanged.
// The launcher forwards argv verbatim; this is the script's own semantics.
func TestMidSetUnknownTokenTruncates(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--set", "002-vendor-script", "--bogus", "status=done"}
	lArgs := append([]string{"--workflow-dir", launcherRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if lCode != 1 {
		t.Fatalf("launcher exit=%d, want 1 (truncated --set has no field=value)", lCode)
	}
	if lCode != oCode {
		t.Fatalf("exit: launcher=%d oracle=%d", lCode, oCode)
	}
	if !strings.Contains(lErr, "requires at least one field=value") {
		t.Fatalf("launcher stderr=%q, want the truncation error", lErr)
	}
	if strings.TrimSpace(lErr) != strings.TrimSpace(oErr) {
		t.Fatalf("stderr: launcher=%q oracle=%q", lErr, oErr)
	}
	if lOut != "" || oOut != "" {
		t.Fatalf("stdout must be empty: launcher=%q oracle=%q", lOut, oOut)
	}
	// status=done was dropped: the entity remains in ideation.
	fm := readFrontmatter(t, filepath.Join(launcherRoot, "002-vendor-script.md"))
	if !strings.Contains(fm, "status: ideation") {
		t.Fatalf("status=done should have been dropped; frontmatter:\n%s", fm)
	}
}

// TestUnknownTopLevelFlagFallsThrough locks the other passthrough case: an
// unrecognized top-level flag is not rejected by the launcher; the vendored
// script ignores it and renders the default table at exit 0. Compared launcher
// vs oracle (root normalized).
func TestUnknownTopLevelFlagFallsThrough(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	args := []string{"--workflow-dir", root, "--bogus-top-level"}

	lOut, lErr, lCode := runLauncher(t, root, env, args...)
	oOut, _, oCode := runOracle(t, root, env, args...)

	if lCode != 0 {
		t.Fatalf("launcher exit=%d stderr=%q, want 0 (unknown top-level flag falls through)", lCode, lErr)
	}
	if lCode != oCode {
		t.Fatalf("exit: launcher=%d oracle=%d", lCode, oCode)
	}
	// Default table renders (header present, entities present).
	if !strings.Contains(lOut, "001-design-seam") {
		t.Fatalf("expected default table with entities, got:\n%s", lOut)
	}
	if normalize(lOut, root) != normalize(oOut, root) {
		t.Fatalf("default-table passthrough mismatch\n--- launcher ---\n%s\n--- oracle ---\n%s",
			normalize(lOut, root), normalize(oOut, root))
	}
}
