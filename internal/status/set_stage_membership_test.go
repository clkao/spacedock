// ABOUTME: AC-3 --set status= membership parity — a non-member stage value is
// ABOUTME: rejected (exit 1, unchanged frontmatter), a member is accepted, launcher vs oracle.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSetStatusNonMemberRejected locks the #189 membership check: a `--set
// status=zzz` where zzz is not a declared stage in the workflow's
// stages.states[].name exits non-zero with an actionable error listing the known
// stages, and leaves the entity frontmatter UNCHANGED. Asserted launcher vs
// oracle so the parity suite covers the new guard.
func TestSetStatusNonMemberRejected(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--set", "002-vendor-script", "status=zzz"}
	lArgs := append([]string{"--workflow-dir", launcherRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if lCode != 1 {
		t.Fatalf("launcher exit=%d, want 1 (non-member status must reject)", lCode)
	}
	if lCode != oCode {
		t.Fatalf("exit: launcher=%d oracle=%d (launcherErr=%q oracleErr=%q)", lCode, oCode, lErr, oErr)
	}
	// The error names the unknown value and lists the known stages.
	for _, want := range []string{"zzz", "backlog", "ideation", "implementation", "done"} {
		if !strings.Contains(lErr, want) {
			t.Fatalf("launcher stderr = %q, want it to mention %q", lErr, want)
		}
	}
	// The error embeds the workflow dir, which differs per run; normalize each
	// run's own root to compare launcher vs oracle structurally.
	if normalize(lErr, launcherRoot) != normalize(oErr, oracleRoot) {
		t.Fatalf("stderr: launcher=%q oracle=%q",
			normalize(lErr, launcherRoot), normalize(oErr, oracleRoot))
	}
	if lOut != "" || oOut != "" {
		t.Fatalf("stdout must be empty on rejection: launcher=%q oracle=%q", lOut, oOut)
	}
	// Frontmatter unchanged: status still ideation, not zzz.
	fm := readFrontmatter(t, filepath.Join(launcherRoot, "002-vendor-script.md"))
	if !strings.Contains(fm, "status: ideation") {
		t.Fatalf("entity status was mutated despite rejection:\n%s", fm)
	}
	if strings.Contains(fm, "status: zzz") {
		t.Fatalf("entity advanced to zzz despite rejection:\n%s", fm)
	}
}

// TestSetStatusMemberAccepted locks the complement: a `--set status=implementation`
// (a declared stage) exits zero and mutates, launcher vs oracle, so the membership
// guard does not reject legitimate transitions.
func TestSetStatusMemberAccepted(t *testing.T) {
	env := pinnedEnv(t)
	launcherRoot := stageFixture(t, "seq-workflow")
	oracleRoot := stageFixture(t, "seq-workflow")

	args := []string{"--set", "002-vendor-script", "status=implementation"}
	lArgs := append([]string{"--workflow-dir", launcherRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if lCode != 0 || oCode != 0 {
		t.Fatalf("exit: launcher=%d (%q) oracle=%d (%q)", lCode, lErr, oCode, oErr)
	}
	if normalize(lOut, "") != normalize(oOut, "") {
		t.Fatalf("narration: launcher=%q oracle=%q", lOut, oOut)
	}
	lFM := normalize(readFrontmatter(t, filepath.Join(launcherRoot, "002-vendor-script.md")), "")
	oFM := normalize(readFrontmatter(t, filepath.Join(oracleRoot, "002-vendor-script.md")), "")
	if lFM != oFM {
		t.Fatalf("frontmatter mismatch\n--- launcher ---\n%s\n--- oracle ---\n%s", lFM, oFM)
	}
	if !strings.Contains(lFM, "status: implementation") {
		t.Fatalf("member status was not applied:\n%s", lFM)
	}
}
