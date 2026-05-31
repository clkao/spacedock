// ABOUTME: AC-4 init seam tests — claude install issues host plugin commands
// ABOUTME: (not a file copy); codex emits the documented add command pair as prose.
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestInitClaudeIssuesHostPluginCommands: `spacedock init --host claude` drives
// the install seam (host plugin marketplace add + install) rather than a
// filesystem copy of skill files.
func TestInitClaudeIssuesHostPluginCommands(t *testing.T) {
	fake := &fakeHost{
		manifest:   compatibleManifest(t), // doctor after install sees a compatible plugin
		installOut: "installed spacedock@spacedock",
	}
	var stdout, stderr bytes.Buffer

	code := runInit(context.Background(), []string{"--host", "claude"}, fake, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if len(fake.installCmds) == 0 {
		t.Fatalf("install seam not invoked — init must use the host plugin mechanism, not a file copy")
	}
	// The seam was called with host=claude.
	if fake.installCmds[0] != "claude" {
		t.Fatalf("install seam host = %q, want claude", fake.installCmds[0])
	}
	// After install, init runs doctor — a compatible report on stdout.
	if !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("init should run doctor after install; stdout = %q", stdout.String())
	}
}

// TestInitCheckRunsDoctorWithoutInstalling: `--check` runs the compatibility
// report without invoking the install seam.
func TestInitCheckRunsDoctorWithoutInstalling(t *testing.T) {
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runInit(context.Background(), []string{"--host", "claude", "--check"}, fake, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if len(fake.installCmds) != 0 {
		t.Fatalf("--check must not install: %v", fake.installCmds)
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("--check should print the doctor report; stdout = %q", stdout.String())
	}
}

// TestInitCodexEmitsAddCommandProse: `spacedock init --host codex` emits the
// documented `codex plugin marketplace add` + `codex plugin add` pair as prose
// (install verb is `add`, not `install`) and does NOT shell the claude install
// seam.
func TestInitCodexEmitsAddCommandProse(t *testing.T) {
	fake := &fakeHost{}
	var stdout, stderr bytes.Buffer

	code := runInit(context.Background(), []string{"--host", "codex"}, fake, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"codex plugin marketplace add",
		"codex plugin add spacedock@spacedock",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("codex init prose missing %q:\n%s", want, out)
		}
	}
	// Codex install verb is `add`, never `install`.
	if strings.Contains(out, "codex plugin install") {
		t.Errorf("codex init prose must not use 'codex plugin install' (verb is add):\n%s", out)
	}
}
