// ABOUTME: AC-3 --archive dest-spelling parity (relative vs absolute --workflow-dir)
// ABOUTME: and the terminal-transition-under-mod-block guard (exit 1, current text).
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestArchiveRelativeDest locks that --archive's dest tracks the --workflow-dir
// spelling: a relative `--workflow-dir .` (run with cwd=root) yields
// `archived: ./_archive/{slug}.md`. Compared launcher-vs-oracle, both run from
// the same relative spelling, and the moved file lands under _archive.
func TestArchiveRelativeDest(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--workflow-dir", ".", "--archive", "001-design-seam"}
	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, args...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, args...)

	if lCode != 0 || oCode != 0 {
		t.Fatalf("exit: launcher=%d (%q) oracle=%d (%q)", lCode, lErr, oCode, oErr)
	}
	want := "archived: ./_archive/001-design-seam.md\n"
	if lOut != want {
		t.Fatalf("launcher narration = %q, want %q (relative dest spelling)", lOut, want)
	}
	if lOut != oOut {
		t.Fatalf("narration: launcher=%q oracle=%q", lOut, oOut)
	}
	// The entity actually moved under _archive and left the active dir.
	if _, err := os.Stat(filepath.Join(launcherRoot, "_archive", "001-design-seam.md")); err != nil {
		t.Fatalf("archived file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(launcherRoot, "001-design-seam.md")); !os.IsNotExist(err) {
		t.Fatalf("source file should be gone after archive, stat err=%v", err)
	}
}

// TestArchiveAbsoluteDest locks the absolute-spelling case: an absolute
// --workflow-dir yields an absolute archived: dest. Compared launcher-vs-oracle
// with the root prefix normalized so no machine path is asserted literally.
func TestArchiveAbsoluteDest(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	lArgs := []string{"--workflow-dir", launcherRoot, "--archive", "001-design-seam"}
	oArgs := []string{"--workflow-dir", oracleRoot, "--archive", "001-design-seam"}
	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if lCode != 0 || oCode != 0 {
		t.Fatalf("exit: launcher=%d (%q) oracle=%d (%q)", lCode, lErr, oCode, oErr)
	}
	// The launcher emits an absolute dest (not realpath'd) under its own root.
	wantLauncher := "archived: " + filepath.Join(launcherRoot, "_archive", "001-design-seam.md") + "\n"
	if lOut != wantLauncher {
		t.Fatalf("launcher narration = %q, want %q (absolute dest spelling)", lOut, wantLauncher)
	}
	// Normalize each run's own root to compare launcher vs oracle structurally.
	if normalize(lOut, launcherRoot) != normalize(oOut, oracleRoot) {
		t.Fatalf("narration: launcher=%q oracle=%q", normalize(lOut, launcherRoot), normalize(oOut, oracleRoot))
	}
}

// TestTerminalSetUnderModBlockRejected locks the guard: a terminal --set
// (status -> terminal stage) on an entity with an active mod-block exits 1 with
// the current error text, and the entity is NOT mutated. Compared launcher vs
// oracle for exit code, stderr text, and unchanged frontmatter.
func TestTerminalSetUnderModBlockRejected(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "guard-workflow")
	oracleRoot := stageFixture(t, "guard-workflow")

	args := []string{"--set", "010-blocked", "status=done"}
	lArgs := append([]string{"--workflow-dir", launcherRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if lCode != 1 {
		t.Fatalf("launcher exit=%d, want 1 (guard must reject)", lCode)
	}
	if lCode != oCode {
		t.Fatalf("exit: launcher=%d oracle=%d", lCode, oCode)
	}
	wantErr := "Error: entity 010-blocked has pending mod-block (merge:pr-merge). Clear mod-block in a separate --set call, or use --force."
	if !strings.Contains(lErr, wantErr) {
		t.Fatalf("launcher stderr = %q, want it to contain %q", lErr, wantErr)
	}
	if strings.TrimSpace(lErr) != strings.TrimSpace(oErr) {
		t.Fatalf("stderr: launcher=%q oracle=%q", lErr, oErr)
	}
	if lOut != "" || oOut != "" {
		t.Fatalf("stdout must be empty on rejection: launcher=%q oracle=%q", lOut, oOut)
	}
	// The entity status must be unchanged (still implementation, not done).
	fm := readFrontmatter(t, filepath.Join(launcherRoot, "010-blocked.md"))
	if !strings.Contains(fm, "status: implementation") {
		t.Fatalf("entity was mutated despite guard rejection:\n%s", fm)
	}
	if strings.Contains(fm, "status: done") {
		t.Fatalf("entity advanced to done despite guard rejection:\n%s", fm)
	}
}
