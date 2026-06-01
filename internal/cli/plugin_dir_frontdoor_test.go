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

// TestDevLanePluginDirReachesLaunchSeam locks AC-2(a) under the Option-2 grammar:
// `spacedock claude "task" -- --plugin-dir <vendored-repo>` reaches the launch
// seam with the inner argv beginning `claude --agent spacedock:first-officer`, the
// task-bearing prompt appended LAST, and NO contract-gate rejection — proving the
// manifest this entity vendors flows through the dev lane. The host's
// ResolveManifest is wired to fail; a launch on exit 0 with the FO seam present
// proves the gate was relaxed (ResolveManifest never consulted). The prompt is
// always the last spacedock-built token and --plugin-dir rides in the post-`--`
// passthrough with its value bound, so the host never binds the prompt here
// (AC-3). The narrow limit — a dangling value-taking flag as the FINAL passthrough
// token — is covered by TestDanglingValueTakingHostFlagStillSwallows.
func TestDevLanePluginDirReachesLaunchSeam(t *testing.T) {
	repo := vendoredRepoRoot(t)
	host := &resolveErrHost{}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"do the thing", "--", "--plugin-dir", repo}, t.TempDir(), host, lookFound, &stdout, &stderr)

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
	// The last token is the spacedock-built prompt, and --plugin-dir <repo> rides
	// before it in passthrough with its value bound — the prompt is never adjacent
	// to a value-starved host flag.
	if last := host.launchedArg[len(host.launchedArg)-1]; last != wantBootstrapPrompt+" do the thing" {
		t.Fatalf("last token = %q, want the spacedock prompt", last)
	}
}

// TestDanglingValueTakingHostFlagStillSwallows pins the ACCURATE AC-3 property and
// its honest limitation. The invariant the Option-2 grammar guarantees is narrow:
// the spacedock prompt is ALWAYS the last assembled host-argv token and ALWAYS
// spacedock-constructed (never sourced from a host token). It is NOT a structural
// guarantee that the host can never bind the prompt: a user who dangles a
// value-taking host flag as the FINAL post-`--` passthrough token (`-- --plugin-dir`
// with no value) places that flag immediately before the prompt, so the host's
// own parser binds the prompt as the flag's value. This is byte-identical to
// origin/next and is a user error (the user gave a value-taking flag no value),
// not a regression introduced by this change. The test proves both halves: the
// prompt is still the last spacedock-built token (our invariant holds), and the
// dangling flag is the one immediately before it (the host-side consequence).
func TestDanglingValueTakingHostFlagStillSwallows(t *testing.T) {
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"do the thing", "--", "--plugin-dir"}, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	argv := fake.launchedArg
	wantPrompt := wantBootstrapPrompt + " do the thing"
	// Our invariant: the prompt is the last token and spacedock-constructed.
	if last := argv[len(argv)-1]; last != wantPrompt {
		t.Fatalf("last token = %q, want the spacedock prompt (the always-last invariant must hold)", last)
	}
	// The honest limitation: a dangling value-taking flag sits right before the
	// prompt, so the host parser will bind the prompt as that flag's value. This is
	// a user error identical to origin/next, NOT a structural impossibility.
	if before := argv[len(argv)-2]; before != "--plugin-dir" {
		t.Fatalf("token before prompt = %q, want the dangling --plugin-dir (the host would bind the prompt as its value)", before)
	}
}
