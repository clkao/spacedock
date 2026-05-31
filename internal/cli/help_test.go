// ABOUTME: AC-4 per-command help coverage — claude/codex/install --help carry the
// ABOUTME: sandbox knobs, --plugin-dir, the -- forwarding note, and an Examples block.
package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestFrontDoorHelpCarriesDetail pins AC-4: `spacedock claude --help` and
// `spacedock codex --help` render (exit 0) the sandbox knobs, --skip-contract-check,
// --plugin-dir, the `--` host-flag forwarding note, and an Examples block.
func TestFrontDoorHelpCarriesDetail(t *testing.T) {
	for _, host := range []string{"claude", "codex"} {
		t.Run(host, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run([]string{host, "--help"}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
			}
			out := stdout.String()
			for _, want := range []string{
				"--safehouse",
				"--safehouse-enable",
				"--safehouse-add-dirs",
				"--safehouse-add-dirs-ro",
				"--skip-contract-check",
				"--plugin-dir",
				"forward verbatim",
				"Examples:",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("%s --help missing %q:\n%s", host, want, out)
				}
			}
		})
	}
}

// TestInstallHelpDocumentsHostAndCheck pins AC-4 for the setup verb: `spacedock
// install --help` documents --host and --check.
func TestInstallHelpDocumentsHostAndCheck(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"install", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"--host", "--check", "Examples:"} {
		if !strings.Contains(out, want) {
			t.Errorf("install --help missing %q:\n%s", want, out)
		}
	}
}
