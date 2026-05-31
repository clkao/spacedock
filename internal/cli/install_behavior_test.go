// ABOUTME: AC-4 behavioral install — a real isolated-CLAUDE_CONFIG_DIR install of
// ABOUTME: a local-path marketplace, observing installPath and untouched skills.
package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestClaudePluginInstallIsHostNative runs the real `claude plugin marketplace
// add`/`install` pair against an isolated CLAUDE_CONFIG_DIR + plugin cache and
// observes that (a) `claude plugin list --json` reports spacedock@spacedock with
// an installPath whose manifest carries the fixture requires-contract, and (b)
// no path under the isolated config's skills/ tree was written — the install is
// the host plugin mechanism, not a skill-file copy. Skips when `claude` is not
// on PATH; this is a real install kept hermetic by env isolation, not a mock.
func TestClaudePluginInstallIsHostNative(t *testing.T) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("claude not on PATH; behavioral install test requires the host CLI")
	}

	tmp := t.TempDir()
	marketplace := buildLocalMarketplace(t, tmp)
	configDir := filepath.Join(tmp, "config")
	cacheDir := filepath.Join(tmp, "cache")
	mustMkdir(t, configDir)
	mustMkdir(t, cacheDir)

	env := append(os.Environ(),
		"CLAUDE_CONFIG_DIR="+configDir,
		"CLAUDE_CODE_PLUGIN_CACHE_DIR="+cacheDir,
	)

	runHost(t, claudeBin, env, "plugin", "marketplace", "add", marketplace)
	runHost(t, claudeBin, env, "plugin", "install", "spacedock@spacedock")

	listOut := runHost(t, claudeBin, env, "plugin", "list", "--json")
	var entries []struct {
		ID          string `json:"id"`
		InstallPath string `json:"installPath"`
	}
	if err := json.Unmarshal([]byte(listOut), &entries); err != nil {
		t.Fatalf("parse plugin list --json: %v\n%s", err, listOut)
	}
	var installPath string
	for _, e := range entries {
		if e.ID == "spacedock@spacedock" {
			installPath = e.InstallPath
		}
	}
	if installPath == "" {
		t.Fatalf("plugin list --json did not report an installPath for spacedock@spacedock:\n%s", listOut)
	}

	// (a) The installed manifest carries the fixture requires-contract intact.
	manifestPath := filepath.Join(installPath, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read installed manifest %s: %v", manifestPath, err)
	}
	if !strings.Contains(string(data), `"requires-contract"`) {
		t.Fatalf("installed manifest missing requires-contract:\n%s", data)
	}

	// (b) The host install wrote nothing under the isolated config's skills/ tree
	// outside the plugin install root — init does not copy skill files into
	// ~/.claude/skills.
	skillsDir := filepath.Join(configDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil && len(entries) > 0 {
		t.Fatalf("isolated config skills/ tree was written by the install (want untouched): %v", entries)
	}
}

// buildLocalMarketplace writes a minimal valid local-path marketplace under root
// and returns the marketplace directory. The plugin manifest carries a
// requires-contract bracketing CONTRACT_VERSION (>=1,<2).
func buildLocalMarketplace(t *testing.T, root string) string {
	t.Helper()
	marketplace := filepath.Join(root, "marketplace")
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
	mustWrite(t, filepath.Join(plugin, ".claude-plugin", "plugin.json"), `{ "name": "spacedock", "version": "0.0.0", "requires-contract": ">=1,<2", "skills": "./skills/" }
`)
	mustWrite(t, filepath.Join(plugin, "skills", "demo", "SKILL.md"), "---\nname: demo\ndescription: demo skill\n---\ndemo\n")
	return marketplace
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runHost runs the host CLI with the given env and returns its combined output,
// failing the test on a non-zero exit.
func runHost(t *testing.T, bin string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", bin, strings.Join(args, " "), err, out)
	}
	return string(out)
}
