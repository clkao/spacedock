// ABOUTME: AC-4(ii) round-trip parity — a #-bearing --set value is written
// ABOUTME: quoted and reads back whole, byte-identically on launcher and oracle.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestCommentValueRoundTripParity locks option C end-to-end across both
// implementations: a --set value containing a space-then-`#` is written quoted
// (so the reader's comment-strip does not truncate it) and reads back whole. The
// on-disk frontmatter and the round-trip read are compared launcher-vs-oracle,
// so the reader-strip AND the writer-quote are parity-pinned together.
func TestCommentValueRoundTripParity(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--set", "002-vendor-script", "source=consolidates #223, #217"}
	lArgs := append([]string{"--workflow-dir", launcherRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)
	if lCode != 0 || oCode != 0 {
		t.Fatalf("exit: launcher=%d (%q) oracle=%d (%q)", lCode, lErr, oCode, oErr)
	}
	// Narration uses the raw (unquoted) value on both sides.
	if normalize(lOut, "") != normalize(oOut, "") {
		t.Fatalf("narration: launcher=%q oracle=%q", lOut, oOut)
	}

	// On-disk frontmatter is byte-identical and carries the QUOTED value.
	lFM := readFrontmatter(t, filepath.Join(launcherRoot, "002-vendor-script.md"))
	oFM := readFrontmatter(t, filepath.Join(oracleRoot, "002-vendor-script.md"))
	if lFM != oFM {
		t.Fatalf("on-disk frontmatter mismatch\n--- launcher ---\n%s\n--- oracle ---\n%s", lFM, oFM)
	}
	if !strings.Contains(lFM, `source: "consolidates #223, #217"`) {
		t.Fatalf("expected quoted #-bearing value on disk, got:\n%s", lFM)
	}

	// Round-trip read yields the whole value on both sides.
	lGot := ParseFrontmatter(filepath.Join(launcherRoot, "002-vendor-script.md"))["source"]
	if lGot != "consolidates #223, #217" {
		t.Fatalf("launcher round-trip read = %q, want whole value", lGot)
	}
	// The oracle reads it back whole too (run a --fields source read and compare).
	lRead, _, _ := runLauncher(t, launcherRoot, env, "--workflow-dir", launcherRoot, "--fields", "source")
	oRead, _, _ := runOracle(t, oracleRoot, env, "--workflow-dir", oracleRoot, "--fields", "source")
	if normalize(lRead, launcherRoot) != normalize(oRead, oracleRoot) {
		t.Fatalf("--fields source read mismatch\n--- launcher ---\n%s\n--- oracle ---\n%s",
			normalize(lRead, launcherRoot), normalize(oRead, oracleRoot))
	}
	if !strings.Contains(oRead, "#223") {
		t.Fatalf("oracle did not read the #-bearing value back whole:\n%s", oRead)
	}
}
