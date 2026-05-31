// ABOUTME: AC-5a absence invariant — the vendored FO/ensign contracts carry
// ABOUTME: zero plugin-private status-path refs (no real seam can prove an absence).
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pluginPrivateStatusRefs are the plugin-private status invocation forms the
// vendored skill surface must never reference once it calls `spacedock status`.
var pluginPrivateStatusRefs = []string{
	"skills/commission/bin/status",
	"{spacedock_plugin_dir}",
	"commission/bin/status",
}

// TestNoPluginPrivateStatusPathInContracts locks the AC-5a absence invariant:
// NEITHER the FO nor the ensign contract references any plugin-private status
// path. The positive "the contract calls `spacedock status`" claim is owned
// behaviorally by the launcher smoke seam (launcher_smoke_test.go drives the
// real status binary for list/set/archive) and internal/status/*; that
// behavioral coverage makes a bare prose-grep over the FO text redundant.
//
// Oracle: the absence invariant over the vendored on-disk skill surface — the
// vendored skill tree must NOT re-introduce a retired plugin-private status
// path (skills/commission/bin/status, {spacedock_plugin_dir}, commission/bin/
// status). No positive behavioral seam can prove an absence: a re-introduced
// plugin path would silently break the `spacedock status` launcher contract,
// and only this structural scan over the contract bytes catches it. This is NOT
// bare prose-grep — it asserts a structural negative the system depends on.
func TestNoPluginPrivateStatusPathInContracts(t *testing.T) {
	root := skillsRoot(t)
	fo := readSkill(t, root, "first-officer/references/first-officer-shared-core.md")
	ensign := readSkill(t, root, "ensign/references/ensign-shared-core.md")

	for name, content := range map[string]string{
		"first-officer-shared-core.md": fo,
		"ensign-shared-core.md":        ensign,
	} {
		for _, ref := range pluginPrivateStatusRefs {
			if strings.Contains(content, ref) {
				t.Errorf("%s references plugin-private status path %q", name, ref)
			}
		}
	}
}

func readSkill(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read vendored skill %s: %v", rel, err)
	}
	return string(b)
}
