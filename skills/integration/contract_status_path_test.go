// ABOUTME: AC-5a static half — the vendored FO/ensign contracts call
// ABOUTME: `spacedock status` and carry zero plugin-private status-path refs.
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

// TestVendoredSkillsCallSpacedockStatus locks AC-5a (static half): the FO
// contract issues its load-bearing workflow reads/mutations through
// `spacedock status`, and NEITHER the FO nor the ensign contract references any
// plugin-private status path. Co-located with the vendored-fixture
// requires-contract bracketing test (internal/contract) which closes the
// fixture-vs-binary half of AC-5a.
func TestVendoredSkillsCallSpacedockStatus(t *testing.T) {
	root := skillsRoot(t)
	fo := readSkill(t, root, "first-officer/references/first-officer-shared-core.md")
	ensign := readSkill(t, root, "ensign/references/ensign-shared-core.md")

	// (1) The FO contract calls `spacedock status` for its workflow reads and
	// mutations (the launcher front-end the plugin agents depend on).
	if !strings.Contains(fo, "spacedock status") {
		t.Errorf("FO contract does not call `spacedock status`")
	}

	// (2) Neither contract references a plugin-private status path.
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
