// ABOUTME: AC-1 bracketing test over the vendored repo plugin manifest —
// ABOUTME: requires-contract parses via contract.ParseRange and brackets CONTRACT_VERSION.
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/contract"
)

// repoRoot is the project root: the parent of the skills/ dir this test package
// lives inside. The vendored plugin manifest lives at repoRoot/.claude-plugin/.
func repoRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestVendoredManifestBracketsContractVersion locks AC-1's manifest<->binary
// drift check: the vendored .claude-plugin/plugin.json declares a
// requires-contract that parses via the real contract.ParseRange and brackets
// the binary's CONTRACT_VERSION. The manifest fed to --plugin-dir and the binary
// gate must agree in one go test — a future range edit that excludes the binary
// fails here.
func TestVendoredManifestBracketsContractVersion(t *testing.T) {
	manifestPath := filepath.Join(repoRoot(t), ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read vendored manifest %s: %v", manifestPath, err)
	}
	var m struct {
		Name             string `json:"name"`
		RequiresContract string `json:"requires-contract"`
		Skills           string `json:"skills"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse vendored manifest %s: %v", manifestPath, err)
	}
	if m.Name != "spacedock" {
		t.Errorf("manifest name = %q, want spacedock", m.Name)
	}
	if m.Skills != "./skills/" {
		t.Errorf("manifest skills = %q, want ./skills/", m.Skills)
	}
	if m.RequiresContract == "" {
		t.Fatalf("manifest has no requires-contract field (the bootstrap-cliff before-state)")
	}
	lo, hi, err := contract.ParseRange(m.RequiresContract)
	if err != nil {
		t.Fatalf("requires-contract %q does not parse: %v", m.RequiresContract, err)
	}
	if !(lo <= contract.CONTRACT_VERSION && contract.CONTRACT_VERSION < hi) {
		t.Fatalf("requires-contract %s does not bracket CONTRACT_VERSION=%d", m.RequiresContract, contract.CONTRACT_VERSION)
	}
}
