// ABOUTME: AC-2 manifest tests — root marketplace.json self-referential url+ref
// ABOUTME: entry, and .codex-plugin/plugin.json requires-contract brackets binary.
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/contract"
)

// calendarVersionRe matches the marketplace entry's calendar key, `0.0.YYYYMMDDNN`
// — the `claude plugin update` re-pull key (AC-2d). It is DISTINCT from the
// plugin.json semver-ish `version` (the release-stamped display version, AC-4).
var calendarVersionRe = regexp.MustCompile(`^0\.0\.\d{10}$`)

// TestRootMarketplaceSelfReferentialEntry locks AC-2a/2c: the root
// .claude-plugin/marketplace.json names the marketplace `spacedock` with one
// plugin entry also named `spacedock` (so the install id is `spacedock@spacedock`,
// the id the binary hardcodes), sourced url+ref:next (the no-restructure path),
// carrying a calendar version key.
func TestRootMarketplaceSelfReferentialEntry(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read root marketplace %s: %v", path, err)
	}
	var mp struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name    string          `json:"name"`
			Source  json.RawMessage `json:"source"`
			Version string          `json:"version"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &mp); err != nil {
		t.Fatalf("parse marketplace: %v", err)
	}
	if mp.Name != "spacedock" {
		t.Errorf("marketplace name = %q, want spacedock (install id is {entry}@{marketplace})", mp.Name)
	}
	if len(mp.Plugins) != 1 {
		t.Fatalf("marketplace has %d plugin entries, want exactly 1", len(mp.Plugins))
	}
	entry := mp.Plugins[0]
	if entry.Name != "spacedock" {
		t.Errorf("plugin entry name = %q, want spacedock (the binary hardcodes spacedock@spacedock)", entry.Name)
	}
	if !calendarVersionRe.MatchString(entry.Version) {
		t.Errorf("entry version = %q, want a 0.0.YYYYMMDDNN calendar key", entry.Version)
	}

	var src struct {
		Source string `json:"source"`
		URL    string `json:"url"`
		Ref    string `json:"ref"`
	}
	if err := json.Unmarshal(entry.Source, &src); err != nil {
		t.Fatalf("entry source is not the {source,url,ref} object url+ref form: %v", err)
	}
	if src.Source != "url" {
		t.Errorf("entry source.source = %q, want url (source:\".\" is rejected by the host)", src.Source)
	}
	if src.Ref != "next" {
		t.Errorf("entry source.ref = %q, want next (the moving branch the manifest lives on)", src.Ref)
	}
	if src.URL == "" {
		t.Errorf("entry source.url is empty")
	}
}

// TestCodexManifestBracketsContractVersion locks AC-2's Codex half: the new
// .codex-plugin/plugin.json carries a requires-contract that parses via the real
// ParseRange and brackets the binary's CONTRACT_VERSION (so `spacedock doctor
// --host codex` resolves a compatible manifest), names the plugin `spacedock`,
// and points skills at ./skills/.
func TestCodexManifestBracketsContractVersion(t *testing.T) {
	path := filepath.Join(repoRoot(t), ".codex-plugin", "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read codex manifest %s: %v", path, err)
	}
	var m struct {
		Name             string `json:"name"`
		RequiresContract string `json:"requires-contract"`
		Skills           string `json:"skills"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse codex manifest: %v", err)
	}
	if m.Name != "spacedock" {
		t.Errorf("codex manifest name = %q, want spacedock", m.Name)
	}
	if m.Skills != "./skills/" {
		t.Errorf("codex manifest skills = %q, want ./skills/", m.Skills)
	}
	if m.RequiresContract == "" {
		t.Fatalf("codex manifest has no requires-contract (AC-2 requires it for doctor --host codex)")
	}
	lo, hi, err := contract.ParseRange(m.RequiresContract)
	if err != nil {
		t.Fatalf("codex requires-contract %q does not parse: %v", m.RequiresContract, err)
	}
	if !(lo <= contract.CONTRACT_VERSION && contract.CONTRACT_VERSION < hi) {
		t.Fatalf("codex requires-contract %s does not bracket CONTRACT_VERSION=%d", m.RequiresContract, contract.CONTRACT_VERSION)
	}
}
