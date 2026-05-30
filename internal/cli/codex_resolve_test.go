// ABOUTME: RUN-verified codex resolver test — execHost.ResolveManifest("codex")
// ABOUTME: against the installed codex CLI resolves a real .codex-plugin manifest.
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodexResolveManifestAgainstInstalledHost drives the production codex
// resolver against the real `codex` CLI. codex 0.132.0 rejects `--json` (exit 2),
// so the resolver must use the supported `codex plugin list` text output. When
// spacedock@spacedock is installed, ResolveManifest must return a non-empty path
// to an existing .codex-plugin/plugin.json; when it is NOT installed, it must
// return "" with no error (the no-plugin-found state). Skips when codex is absent.
func TestCodexResolveManifestAgainstInstalledHost(t *testing.T) {
	codexBin, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("codex not on PATH; codex resolver test requires the host CLI")
	}

	listOut, err := exec.Command(codexBin, "plugin", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("codex plugin list failed (exit/output): %v\n%s", err, listOut)
	}
	installed := strings.Contains(string(listOut), "spacedock@spacedock (installed")

	path, err := execHost{}.ResolveManifest("codex")
	if err != nil {
		t.Fatalf("ResolveManifest(codex) errored: %v", err)
	}

	if !installed {
		if path != "" {
			t.Fatalf("spacedock@spacedock not installed in codex, but resolver returned %q", path)
		}
		t.Skip("spacedock@spacedock not installed in codex; resolved empty as expected")
	}

	if path == "" {
		t.Fatalf("spacedock@spacedock installed in codex, but resolver returned empty path")
	}
	if filepath.Base(path) != "plugin.json" || !strings.Contains(path, ".codex-plugin") {
		t.Fatalf("resolved codex manifest path is not a .codex-plugin/plugin.json: %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resolved codex manifest does not exist: %q (%v)", path, err)
	}
}
