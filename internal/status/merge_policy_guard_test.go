// ABOUTME: merge: local guard-relaxation parity — the registered-merge-hook
// ABOUTME: fixture exercises the terminal-guard branch the policy relaxes.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// assertParity runs the same args through native and oracle and fails unless
// their exit code and stderr match. It returns the native exit code so the
// caller can additionally assert the expected pass/refuse outcome.
func assertParity(t *testing.T, fixture, slug string, args ...string) (int, string, string) {
	t.Helper()
	env := pinnedEnv(t)
	nativeRoot := stageFixture(t, fixture)
	oracleRoot := stageFixture(t, fixture)

	nArgs := append([]string{"--workflow-dir", nativeRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)
	nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if nCode != oCode {
		t.Fatalf("exit: native=%d oracle=%d (nErr=%q oErr=%q)", nCode, oCode, nErr, oErr)
	}
	if strings.TrimSpace(nErr) != strings.TrimSpace(oErr) {
		t.Fatalf("stderr: native=%q oracle=%q", nErr, oErr)
	}
	// Normalize each run's own temp root (and the realpath'd archive dest) to a
	// placeholder so the two distinct fixture roots compare structurally — the
	// --archive dest carries an absolute path that differs between the two runs.
	if normalize(nOut, nativeRoot) != normalize(oOut, oracleRoot) {
		t.Fatalf("stdout: native=%q oracle=%q", normalize(nOut, nativeRoot), normalize(oOut, oracleRoot))
	}
	return nCode, nOut, nErr
}

// TestMergeLocalNoSentinelTerminalSetSucceeds (AC-3): under merge: local, a
// terminal --set with empty pr and empty mod-block succeeds without --force even
// though a merge hook is registered — the policy exempts the pr-requirement.
func TestMergeLocalNoSentinelTerminalSetSucceeds(t *testing.T) {
	code, _, errOut := assertParity(t, "merge-local-workflow", "020-no-sentinel",
		"--set", "020-no-sentinel", "status=done")
	if code != 0 {
		t.Fatalf("merge: local terminal --set should succeed (exit 0), got %d (stderr=%q)", code, errOut)
	}
}

// TestMergeLocalNoSentinelArchiveSucceeds (AC-3): under merge: local, --archive
// with empty pr and empty mod-block succeeds without --force.
func TestMergeLocalNoSentinelArchiveSucceeds(t *testing.T) {
	code, _, errOut := assertParity(t, "merge-local-workflow", "020-no-sentinel",
		"--archive", "020-no-sentinel")
	if code != 0 {
		t.Fatalf("merge: local --archive should succeed (exit 0), got %d (stderr=%q)", code, errOut)
	}
}

// TestSentinelSatisfiesGuardTerminalSet (AC-1): the post-merge pr=local-merge:{sha}
// sentinel satisfies the merge-hook guard with NO --force, so a terminal --set
// succeeds. This holds under merge: local too — the sentinel records the landed
// merge regardless of policy.
func TestSentinelSatisfiesGuardTerminalSet(t *testing.T) {
	code, _, errOut := assertParity(t, "merge-local-workflow", "010-sentinel",
		"--set", "010-sentinel", "status=done")
	if code != 0 {
		t.Fatalf("sentinel terminal --set should succeed (exit 0), got %d (stderr=%q)", code, errOut)
	}
}

// TestSentinelSatisfiesGuardArchive (AC-1): the sentinel also satisfies the
// --archive merge-hook guard with NO --force.
func TestSentinelSatisfiesGuardArchive(t *testing.T) {
	code, _, errOut := assertParity(t, "merge-local-workflow", "010-sentinel",
		"--archive", "010-sentinel")
	if code != 0 {
		t.Fatalf("sentinel --archive should succeed (exit 0), got %d (stderr=%q)", code, errOut)
	}
}

// TestMergeLocalModBlockPendingStillBlocks (AC-3, the wrongful-terminalization
// mechanical survivor): merge: local relaxes the pr-requirement of the merge-hook
// check, but NOT the mod-block-pending guard. An in-flight mod-block must still
// refuse a terminal --set under merge: local — the set->clear->terminalize
// ceremony separation stays mechanically enforced. This is the named scenario
// that the safety invariant turns on: an FO must not collapse the ceremony, and
// the mechanism still catches a terminal transition that combines a live block.
func TestMergeLocalModBlockPendingStillBlocks(t *testing.T) {
	code, out, errOut := assertParity(t, "merge-local-workflow", "030-pending",
		"--set", "030-pending", "status=done")
	if code != 1 {
		t.Fatalf("mod-block-pending terminal --set must refuse (exit 1) under merge: local, got %d", code)
	}
	if !strings.Contains(errOut, "pending mod-block (merge:local-merge)") {
		t.Fatalf("stderr should name the pending mod-block, got %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout must be empty on rejection, got %q", out)
	}
}

// TestMergeLocalCombinedClearAndTerminalizeRefused (AC-3, the wrongful-
// terminalization mechanical survivor): merge: local does NOT permit collapsing
// the mandatory clear-then-terminalize ceremony into one call. Clearing mod-block
// in the SAME --set that terminalizes is refused regardless of policy, so the
// audit history must show the block resolving separately from terminalization.
func TestMergeLocalCombinedClearAndTerminalizeRefused(t *testing.T) {
	code, out, errOut := assertParity(t, "merge-local-workflow", "030-pending",
		"--set", "030-pending", "mod-block=", "status=done")
	if code != 1 {
		t.Fatalf("combined clear+terminalize must refuse (exit 1) under merge: local, got %d", code)
	}
	if !strings.Contains(errOut, "combined mod-block clear with terminal transition") {
		t.Fatalf("stderr should name the combined-clear refusal, got %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout must be empty on rejection, got %q", out)
	}
}

// TestMergePrDefaultNoSentinelStillRefuses (AC-1 companion / AC-4): the SAME
// no-sentinel/no-mod-block entity, under the DEFAULT policy (merge: key absent),
// still refuses a terminal --set — the merge-hook catch is preserved. This is the
// byte-identical-to-today guarantee for un-declared workflows.
func TestMergePrDefaultNoSentinelStillRefuses(t *testing.T) {
	code, out, errOut := assertParity(t, "merge-pr-workflow", "020-no-sentinel",
		"--set", "020-no-sentinel", "status=done")
	if code != 1 {
		t.Fatalf("default-policy terminal --set must still refuse (exit 1), got %d", code)
	}
	if !strings.Contains(errOut, "cannot advance to terminal") {
		t.Fatalf("stderr should name the merge-hook refusal, got %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout must be empty on rejection, got %q", out)
	}
}

// TestMergePrDefaultNoSentinelArchiveStillRefuses (AC-4): the default-policy
// --archive of the same no-sentinel entity also still refuses.
func TestMergePrDefaultNoSentinelArchiveStillRefuses(t *testing.T) {
	code, out, errOut := assertParity(t, "merge-pr-workflow", "020-no-sentinel",
		"--archive", "020-no-sentinel")
	if code != 1 {
		t.Fatalf("default-policy --archive must still refuse (exit 1), got %d", code)
	}
	if !strings.Contains(errOut, "cannot be archived") {
		t.Fatalf("stderr should name the merge-hook archive refusal, got %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout must be empty on rejection, got %q", out)
	}
}

// TestSentinelDisplaysAsLocal (AC-2): a pr field of local-merge:{short-sha}
// renders in the status table as `{short-sha} (local)`, distinguishable from a
// real PR reference. Native and oracle must agree.
func TestSentinelDisplaysAsLocal(t *testing.T) {
	env := pinnedEnv(t)
	nativeRoot := stageFixture(t, "merge-local-workflow")
	oracleRoot := stageFixture(t, "merge-local-workflow")

	args := []string{"--fields", "pr"}
	nArgs := append([]string{"--workflow-dir", nativeRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)
	nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if nCode != 0 || oCode != 0 {
		t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
	}
	if nOut != oOut {
		t.Fatalf("display parity: native=%q oracle=%q", nOut, oOut)
	}
	if !strings.Contains(nOut, "abc1234 (local)") {
		t.Fatalf("sentinel should render as 'abc1234 (local)', got:\n%s", nOut)
	}
	if strings.Contains(nOut, "local-merge:abc1234") {
		t.Fatalf("raw sentinel value should not appear verbatim in the table:\n%s", nOut)
	}
}

// TestMergeLocalEntityActuallyAdvances (AC-1): confirms the merge: local terminal
// --set truly mutated the entity to done (not a vacuous exit-0).
func TestMergeLocalEntityActuallyAdvances(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "merge-local-workflow")
	_, errOut, code := runNative(t, root, env,
		"--workflow-dir", root, "--set", "020-no-sentinel", "status=done")
	if code != 0 {
		t.Fatalf("terminal --set should succeed, got %d (stderr=%q)", code, errOut)
	}
	fm := readWhole(t, filepath.Join(root, "020-no-sentinel.md"))
	if !strings.Contains(fm, "status: done") {
		t.Fatalf("entity should have advanced to done:\n%s", fm)
	}
}
