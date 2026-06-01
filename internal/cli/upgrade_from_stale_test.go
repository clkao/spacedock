// ABOUTME: AC-3 live upgrade-from-stale smoke — a real isolated-CLAUDE_CONFIG_DIR
// ABOUTME: install of a stale (no requires-contract) plugin, then the 3-command upgrade to green.
package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/contract"
)

// TestUpgradeFromStaleMovesToGreen is the load-bearing AC-3 proof: a plugin
// installed from a stale marketplace (no requires-contract, the 0.12.1 shape)
// resolves to the plugin-predates-contract verdict (exit 1, the dead-end the
// captain hit); running the 3-command installArgvSequence against an upgraded
// marketplace (requires-contract >=1,<2) then leaves doctor reporting compatible
// (exit 0). This proves plain `plugin install` no-ops on an already-installed
// plugin and the inserted `plugin uninstall` is what moves the stale install off.
// Skips when `claude` is not on PATH; a real install kept hermetic by env
// isolation, not a mock — mirrors TestClaudePluginInstallIsHostNative.
func TestUpgradeFromStaleMovesToGreen(t *testing.T) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("claude not on PATH; upgrade-from-stale smoke requires the host CLI")
	}

	tmp := t.TempDir()
	staleMarketplace := buildStaleMarketplace(t, tmp)
	upgradedMarketplace := buildLocalMarketplace(t, filepath.Join(tmp, "upgraded"))
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	mustMkdir(t, configDir)
	mustMkdir(t, cacheDir)

	env := append(os.Environ(),
		"CLAUDE_CONFIG_DIR="+configDir,
		"CLAUDE_CODE_PLUGIN_CACHE_DIR="+cacheDir,
	)

	// Seed the stale install: marketplace add + install, the 0.12.1 shape.
	runHost(t, claudeBin, env, "plugin", "marketplace", "add", staleMarketplace)
	runHost(t, claudeBin, env, "plugin", "install", "spacedock@spacedock")

	// The stale install resolves to the predates-contract verdict (exit 1) — the
	// dead-end this entity fixes.
	staleManifest := resolveClaudeManifestEnv(t, claudeBin, env)
	staleVerdict := contract.ManifestVerdict(staleManifest, "claude", "next")
	if staleVerdict.Verdict != contract.PluginPredatesContract {
		t.Fatalf("stale install verdict = %v, want plugin-predates-contract (message=%q)", staleVerdict.Verdict, staleVerdict.Message)
	}

	// Upgrade via the committed 3-command shape. Plain `plugin install` would
	// no-op here (the plugin is already installed); the inserted uninstall is what
	// moves it. Run the exact argv installArgvSequence emits.
	for _, args := range installArgvSequence(upgradedMarketplace, "") {
		runHost(t, claudeBin, env, args...)
	}

	// Doctor now reports compatible (exit 0) — the install moved off the stale plugin.
	upgradedManifest := resolveClaudeManifestEnv(t, claudeBin, env)
	upgradedVerdict := contract.ManifestVerdict(upgradedManifest, "claude", "next")
	if upgradedVerdict.Verdict != contract.Compatible {
		t.Fatalf("after upgrade, verdict = %v, want compatible (message=%q)", upgradedVerdict.Verdict, upgradedVerdict.Message)
	}
}

// buildStaleMarketplace writes a local-path marketplace whose plugin manifest has
// NO requires-contract field — the 0.12.1 shape that predates the contract
// mechanism. Returns the marketplace directory.
func buildStaleMarketplace(t *testing.T, root string) string {
	t.Helper()
	marketplace := filepath.Join(root, "stale")
	plugin := filepath.Join(marketplace, "spacedock")
	mustMkdir(t, filepath.Join(marketplace, ".claude-plugin"))
	mustMkdir(t, filepath.Join(plugin, ".claude-plugin"))
	mustMkdir(t, filepath.Join(plugin, "skills", "demo"))

	mustWrite(t, filepath.Join(marketplace, ".claude-plugin", "marketplace.json"), `{
  "name": "spacedock",
  "owner": { "name": "CL Kao" },
  "plugins": [
    { "name": "spacedock", "source": "./spacedock", "description": "test", "category": "workflow" }
  ]
}
`)
	mustWrite(t, filepath.Join(plugin, ".claude-plugin", "plugin.json"), `{ "name": "spacedock", "version": "0.12.1", "skills": "./skills/" }
`)
	mustWrite(t, filepath.Join(plugin, "skills", "demo", "SKILL.md"), "---\nname: demo\ndescription: demo skill\n---\ndemo\n")
	return marketplace
}

// resolveClaudeManifestEnv resolves the installed spacedock@spacedock manifest
// path under an isolated env, mirroring execHost.resolveClaudeManifest but with
// the test's env applied.
func resolveClaudeManifestEnv(t *testing.T, claudeBin string, env []string) string {
	t.Helper()
	cmd := exec.Command(claudeBin, "plugin", "list", "--json")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("claude plugin list --json: %v", err)
	}
	var entries []pluginListEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		t.Fatalf("parse plugin list --json: %v\n%s", err, out)
	}
	for _, e := range entries {
		if e.ID == "spacedock@spacedock" && e.InstallPath != "" {
			return filepath.Join(e.InstallPath, manifestSubpath("claude"))
		}
	}
	t.Fatalf("spacedock@spacedock not resolved in:\n%s", out)
	return ""
}
