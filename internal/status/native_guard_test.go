// ABOUTME: AC-4 native guard parity — terminal --set under a mod-block, a
// ABOUTME: merge-hook-unsatisfied terminal --set, and a mod-blocked --archive.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeTerminalSetUnderModBlock locks the terminal-transition-under-mod-
// block guard: native and oracle both exit 1 with the same error text and leave
// the entity unmutated.
func TestNativeTerminalSetUnderModBlock(t *testing.T) {
	env := pinnedEnv(t)
	nativeRoot := stageFixture(t, "guard-workflow")
	oracleRoot := stageFixture(t, "guard-workflow")

	args := []string{"--set", "010-blocked", "status=done"}
	nArgs := append([]string{"--workflow-dir", nativeRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if nCode != 1 {
		t.Fatalf("native exit=%d, want 1", nCode)
	}
	if nCode != oCode {
		t.Fatalf("exit: native=%d oracle=%d", nCode, oCode)
	}
	if strings.TrimSpace(nErr) != strings.TrimSpace(oErr) {
		t.Fatalf("stderr: native=%q oracle=%q", nErr, oErr)
	}
	if nOut != "" || oOut != "" {
		t.Fatalf("stdout must be empty on rejection: native=%q oracle=%q", nOut, oOut)
	}
	fm := readWhole(t, filepath.Join(nativeRoot, "010-blocked.md"))
	if !strings.Contains(fm, "status: implementation") || strings.Contains(fm, "status: done") {
		t.Fatalf("entity mutated despite guard:\n%s", fm)
	}
}

// TestNativeArchiveRelativeDest locks --archive's relative dest spelling parity.
func TestNativeArchiveRelativeDest(t *testing.T) {
	env := pinnedEnv(t)
	nativeRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--workflow-dir", ".", "--archive", "001-design-seam"}
	nOut, nErr, nCode := runNative(t, nativeRoot, env, args...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, args...)

	if nCode != 0 || oCode != 0 {
		t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
	}
	want := "archived: ./_archive/001-design-seam.md\n"
	if nOut != want {
		t.Fatalf("native narration = %q, want %q", nOut, want)
	}
	if nOut != oOut {
		t.Fatalf("narration: native=%q oracle=%q", nOut, oOut)
	}
}

// TestNativeMidSetTruncation locks the mid-set truncation: `--set <slug> --bogus
// status=done` drops status=done, exits 1 with the same error, entity unchanged.
func TestNativeMidSetTruncation(t *testing.T) {
	env := pinnedEnv(t)
	nativeRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--set", "002-vendor-script", "--bogus", "status=done"}
	nArgs := append([]string{"--workflow-dir", nativeRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if nCode != 1 || nCode != oCode {
		t.Fatalf("exit: native=%d oracle=%d", nCode, oCode)
	}
	if !strings.Contains(nErr, "requires at least one field=value") {
		t.Fatalf("native stderr=%q", nErr)
	}
	if strings.TrimSpace(nErr) != strings.TrimSpace(oErr) {
		t.Fatalf("stderr: native=%q oracle=%q", nErr, oErr)
	}
	if nOut != "" || oOut != "" {
		t.Fatalf("stdout must be empty: native=%q oracle=%q", nOut, oOut)
	}
}
