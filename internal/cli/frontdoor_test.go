// ABOUTME: AC-3/AC-4 front-door + init seam tests — version-gate fail-fast,
// ABOUTME: launch-seam argv on compatible, install-seam host commands, codex prose.
package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHost records every seam interaction and returns canned results so the
// front door / init paths run with no real host CLI, no exec, no network.
type fakeHost struct {
	// manifest is the path returned by ResolveManifest; "" means no plugin found.
	manifest    string
	resolveErr  error
	launchedArg []string // argv captured by Launch
	launchErr   error
	installCmds []string // host commands captured by Install
	installOut  string
}

func (f *fakeHost) ResolveManifest(host string) (string, error) {
	return f.manifest, f.resolveErr
}

func (f *fakeHost) Launch(argv []string) error {
	f.launchedArg = argv
	return f.launchErr
}

func (f *fakeHost) Install(host, source, branch string) (string, error) {
	f.installCmds = append(f.installCmds, host, source, branch)
	return f.installOut, nil
}

// compatibleManifest returns a fixture path whose requires-contract brackets
// CONTRACT_VERSION (the testdata/compatible.json fixture is >=1,<2).
func compatibleManifest(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "contract", "testdata", "compatible.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func tooOldBinaryManifest(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "contract", "testdata", "too-old-binary.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestClaudeFrontDoorLaunchesOnCompatible: on a compatible contract the front
// door invokes the launch seam with argv beginning `claude --agent
// spacedock:first-officer` and passes through the operator's trailing args.
func TestClaudeFrontDoorLaunchesOnCompatible(t *testing.T) {
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--", "-p", "do the thing"}, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"claude", "--agent", "spacedock:first-officer", "-p", "do the thing", wantBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// TestClaudeFrontDoorFailFastOnMismatch: on a mismatch verdict the launch seam is
// NOT invoked and the process exits non-zero with the pinned remedy on stderr.
func TestClaudeFrontDoorFailFastOnMismatch(t *testing.T) {
	fake := &fakeHost{manifest: tooOldBinaryManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("exit = 0, want non-zero on mismatch")
	}
	if fake.launchedArg != nil {
		t.Fatalf("launch seam invoked on mismatch: %v", fake.launchedArg)
	}
	if !strings.Contains(stderr.String(), "too-old-binary") {
		t.Fatalf("stderr missing pinned remedy: %q", stderr.String())
	}
}

// TestClaudeFrontDoorUnresolvableManifestFailsFast: an unresolvable manifest
// (no installed plugin) is NOT treated as compatible — the front door warns and
// exits non-zero WITHOUT launching.
func TestClaudeFrontDoorUnresolvableManifestFailsFast(t *testing.T) {
	fake := &fakeHost{manifest: ""} // no plugin found
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("exit = 0, want non-zero when manifest unresolvable")
	}
	if fake.launchedArg != nil {
		t.Fatalf("launch seam invoked with unresolvable manifest: %v", fake.launchedArg)
	}
	if !strings.Contains(stderr.String(), "spacedock doctor") && !strings.Contains(stderr.String(), "spacedock init") {
		t.Fatalf("stderr missing actionable remedy pointer: %q", stderr.String())
	}
}

// TestClaudeFrontDoorNonEmptyMissingManifestFailsFast: a host that reports a
// non-empty installPath to a directory LACKING the plugin manifest must NOT
// launch. The resolver returned a path, but the file does not exist — the gate
// must reject the no-plugin-found verdict by inspecting the verdict, not the
// doctor exit code (which is 0 for a non-fatal no-plugin report).
func TestClaudeFrontDoorNonEmptyMissingManifestFailsFast(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir", ".claude-plugin", "plugin.json")
	fake := &fakeHost{manifest: missing} // non-empty path, but the file is absent
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("exit = 0, want non-zero when resolved manifest path is missing")
	}
	if fake.launchedArg != nil {
		t.Fatalf("launch seam invoked with a missing manifest: %v", fake.launchedArg)
	}
	if !strings.Contains(stderr.String(), "spacedock doctor") && !strings.Contains(stderr.String(), "spacedock init") {
		t.Fatalf("stderr missing actionable remedy pointer: %q", stderr.String())
	}
}

// TestClaudeFrontDoorSkipContractCheckBootstrap: the --skip-contract-check
// override launches without resolving the manifest (bootstrap case where the
// plugin is being installed for the first time).
func TestClaudeFrontDoorSkipContractCheckBootstrap(t *testing.T) {
	fake := &fakeHost{manifest: tooOldBinaryManifest(t)} // would mismatch if checked
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--skip-contract-check"}, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 with --skip-contract-check (stderr=%q)", code, stderr.String())
	}
	want := []string{"claude", "--agent", "spacedock:first-officer", wantBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v (skip-check must not pass the flag through)", fake.launchedArg, want)
	}
}

// TestCodexFrontDoorLaunchesOnCompatible: on a compatible contract the codex
// front door invokes the launch seam with argv beginning `codex
// --dangerously-bypass-approvals-and-sandbox` (under .safehouse) and passes
// through the operator's trailing args before the FO-skill prompt.
func TestCodexFrontDoorLaunchesOnCompatible(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runCodex(context.Background(), []string{"--", "-m", "gpt-x"}, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"safehouse", "--trust-workdir-config", "--",
		"codex", "--dangerously-bypass-approvals-and-sandbox", "-m", "gpt-x", wantCodexBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// TestCodexFrontDoorFailFastOnMismatch: codex fails fast on a mismatch verdict
// with the pinned remedy and does NOT launch.
func TestCodexFrontDoorFailFastOnMismatch(t *testing.T) {
	fake := &fakeHost{manifest: tooOldBinaryManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runCodex(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("exit = 0, want non-zero on mismatch")
	}
	if fake.launchedArg != nil {
		t.Fatalf("launch seam invoked on mismatch: %v", fake.launchedArg)
	}
	if !strings.Contains(stderr.String(), "too-old-binary") {
		t.Fatalf("stderr missing pinned remedy: %q", stderr.String())
	}
}

func equalArgv(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
