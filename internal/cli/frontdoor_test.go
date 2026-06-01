// ABOUTME: AC-3/AC-4 front-door + init seam tests — version-gate fail-fast,
// ABOUTME: launch-seam argv on compatible, install-seam host commands, codex prose.
package cli

import (
	"bytes"
	"context"
	"errors"
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
	if !strings.Contains(stderr.String(), "spacedock doctor") && !strings.Contains(stderr.String(), "spacedock install") {
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
	if !strings.Contains(stderr.String(), "spacedock doctor") && !strings.Contains(stderr.String(), "spacedock install") {
		t.Fatalf("stderr missing actionable remedy pointer: %q", stderr.String())
	}
}

// TestGateRemedyNamesLiveInstallCommand: every gateHost remedy must point at a
// command the binary actually recognizes. After the init->install rename a user
// who hits the gate and is told to "run spacedock init --host …" runs a command
// that now exits 2 (unknown command). Drive each remedy branch (resolve error,
// no plugin, missing manifest) and assert the printed remedy names `spacedock
// install` and never `spacedock init`; then prove the named command resolves by
// feeding it through cli.Run and asserting it is not the unknown-command exit 2.
func TestGateRemedyNamesLiveInstallCommand(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir", ".claude-plugin", "plugin.json")
	cases := []struct {
		name string
		fake *fakeHost
	}{
		{"resolve error", &fakeHost{resolveErr: errors.New("host CLI failed")}},
		{"no plugin", &fakeHost{manifest: ""}},
		{"missing manifest", &fakeHost{manifest: missing}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if ok := gateHost(tc.fake, "claude", &stderr); ok {
				t.Fatalf("gateHost = ok, want denied for %s", tc.name)
			}
			remedy := stderr.String()
			if !strings.Contains(remedy, "spacedock install") {
				t.Fatalf("remedy does not name the live install command: %q", remedy)
			}
			if strings.Contains(remedy, "spacedock init") {
				t.Fatalf("remedy names the removed init command (exits 2): %q", remedy)
			}
		})
	}

	// The command the remedy names must resolve in the live command tree. cobra's
	// Find returns the matched command for a registered name and falls back to the
	// root for an unknown one — so `install` must resolve to a non-root command
	// while the removed `init` must fall back to root (the unknown-command path
	// that exits 2). Resolution is deterministic and touches no host CLI.
	var stdout, stderr bytes.Buffer
	root := newRootCommand(context.Background(), nil, t.TempDir(), nil, &stdout, &stderr, &fakeRunner{})
	if cmd, _, err := root.Find([]string{"install"}); err != nil || cmd == root {
		t.Fatalf("`install` did not resolve to a registered command (cmd=%v, err=%v)", cmd.Name(), err)
	}
	if cmd, _, _ := root.Find([]string{"init"}); cmd != root {
		t.Fatalf("`init` resolved to a command (%v) — the removed verb must fall back to the unknown-command path", cmd.Name())
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
