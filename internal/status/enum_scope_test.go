// ABOUTME: AC-2 (#207) enumeration-scope rule — placement, not the status value,
// ABOUTME: decides active vs archived scope; asserted both placements, native vs oracle.
package status

import (
	"strings"
	"testing"
)

// TestEnumerationScopeByPlacement locks the #207 rule: enumeration scope is
// decided by PLACEMENT, not by the `status` frontmatter value. The fixture has
// two entities with the SAME status, one top-level (top-placed) and one under
// _archive/ (arch-placed). The rule, asserted identically for both placements:
//   - the default (active) read surfaces the top-level entity and NOT the
//     archived one, even though both share status: backlog;
//   - the --archived read surfaces the archived entity (and still the active
//     one, since --archived appends archived to active).
// Each read is compared native-vs-oracle so the rule is parity-pinned, not just
// asserted on the Go side.
func TestEnumerationScopeByPlacement(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "enum-scope-workflow")

	t.Run("active-scope", func(t *testing.T) {
		nOut, nErr, nCode := runNative(t, root, env, "--workflow-dir", root)
		oOut, oErr, oCode := runOracle(t, root, env, "--workflow-dir", root)
		if nCode != 0 || oCode != 0 {
			t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
		}
		if nOut != oOut {
			t.Fatalf("active-read native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
		}
		// The rule: top-level placement => active scope; _archive/ => not active,
		// regardless of the shared status value.
		if !strings.Contains(nOut, "top-placed") {
			t.Fatalf("active read must surface the top-level entity:\n%s", nOut)
		}
		if strings.Contains(nOut, "arch-placed") {
			t.Fatalf("active read must NOT surface the _archive/-placed entity:\n%s", nOut)
		}
	})

	t.Run("archived-scope", func(t *testing.T) {
		nOut, nErr, nCode := runNative(t, root, env, "--workflow-dir", root, "--archived")
		oOut, oErr, oCode := runOracle(t, root, env, "--workflow-dir", root, "--archived")
		if nCode != 0 || oCode != 0 {
			t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
		}
		if nOut != oOut {
			t.Fatalf("archived-read native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
		}
		// The rule: _archive/ placement => archived scope is enumerated under
		// --archived; the active top-level entity is still present (--archived
		// appends archived to active).
		if !strings.Contains(nOut, "arch-placed") {
			t.Fatalf("--archived read must surface the _archive/-placed entity:\n%s", nOut)
		}
		if !strings.Contains(nOut, "top-placed") {
			t.Fatalf("--archived read must still surface the active top-level entity:\n%s", nOut)
		}
	})
}
