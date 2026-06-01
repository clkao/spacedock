// ABOUTME: Locks the test-integrity contract that the parity oracle resolves to
// ABOUTME: the in-tree vendored copy, so a missing oracle FAILS rather than skips.
package status

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOracleResolvesInTree pins the AC-2 contract: with no SPACEDOCK_ORACLE
// override, oraclePath resolves the project-vendored oracle (internal/status/
// vendor/status), which is always present in a checkout. This is what makes the
// parity suite hard-fail on a real divergence instead of green-by-skip on CI or
// a fresh clone, where a hardcoded laptop path would be absent.
func TestOracleResolvesInTree(t *testing.T) {
	t.Setenv(oracleEnvVar, "")
	got := oraclePath(t)
	want, err := filepath.Abs(filepath.Join("vendor", "status"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("oraclePath resolved %q, want the in-tree vendored oracle %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("resolved oracle %q does not exist: %v", got, err)
	}
}
