// ABOUTME: AC-2 dev-lane front-door seam — `--plugin-dir <vendored-repo>` reaches
// ABOUTME: the launch seam (claude --agent spacedock:first-officer) with no gate call.
package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// vendoredRepoRoot is the project root carrying the vendored
// .claude-plugin/plugin.json — the dev-lane --plugin-dir target. The internal/cli
// package sits two levels under root.
func vendoredRepoRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(p, ".claude-plugin", "plugin.json")); err != nil {
		t.Fatalf("vendored manifest not at repo root %s: %v", p, err)
	}
	return p
}

// resolveErrHost fails every ResolveManifest call. Wired so the dev-lane test
// proves the contract gate is NOT consulted: if --plugin-dir did not relax the
// gate, gateHost would call ResolveManifest, hit this error, and deny launch.
type resolveErrHost struct {
	fakeHost
}

func (h *resolveErrHost) ResolveManifest(string) (string, error) {
	return "", errors.New("ResolveManifest must not be called on the --plugin-dir dev lane")
}

// TestDevLanePluginDirReachesLaunchSeam locks AC-2(a): `spacedock claude
// --plugin-dir <vendored-repo> -- "task"` reaches the launch seam with the inner
// argv beginning `claude --agent spacedock:first-officer`, the fenced task
// appended, and NO contract-gate rejection — proving the manifest this entity
// vendors flows through the dev lane. The host's ResolveManifest is wired to
// fail; a launch on exit 0 with the FO seam present proves the gate was relaxed
// (ResolveManifest never consulted).
func TestDevLanePluginDirReachesLaunchSeam(t *testing.T) {
	repo := vendoredRepoRoot(t)
	host := &resolveErrHost{}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--plugin-dir", repo, "--", "do the thing"}, t.TempDir(), host, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (--plugin-dir <repo> must relax the gate); stderr=%q", code, stderr.String())
	}
	if host.launchedArg == nil {
		t.Fatalf("launch seam not reached on the --plugin-dir dev lane")
	}
	want := []string{
		"claude", "--agent", "spacedock:first-officer",
		"--plugin-dir", repo,
		wantBootstrapPrompt + " do the thing",
	}
	if !equalArgv(host.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", host.launchedArg, want)
	}
}
