// ABOUTME: AC-3 behavioral guard — deferred and unknown dispatch subcommands
// ABOUTME: exit non-zero with a usage diagnostic, making the deferral observable.
package dispatch

import (
	"strings"
	"testing"
)

// TestDeferredSubcommandGuard asserts each subcommand moved to the sibling
// claude-runtime-segregation surface exits non-zero with a diagnostic naming
// the deferral, rather than silently no-op'ing. This converts the prose
// deferral into observed behavior.
func TestDeferredSubcommandGuard(t *testing.T) {
	for _, sub := range []string{"context-budget", "list-standing", "show-standing", "spawn-standing"} {
		t.Run(sub, func(t *testing.T) {
			res := runNative("", sub, "--workflow-dir", "/tmp")
			if res.exit == 0 {
				t.Errorf("deferred subcommand %q exited 0 (must be non-zero)", sub)
			}
			if !strings.Contains(res.stderr, "claude-runtime-segregation") {
				t.Errorf("deferred subcommand %q diagnostic does not name the sibling surface:\n%q", sub, res.stderr)
			}
			if res.stdout != "" {
				t.Errorf("deferred subcommand %q wrote stdout (must be silent on stdout):\n%q", sub, res.stdout)
			}
		})
	}
}

// TestUnknownSubcommandGuard asserts an unknown subcommand and a bare dispatch
// invocation both exit non-zero with usage on stderr.
func TestUnknownSubcommandGuard(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		res := runNative("", "frobnicate")
		if res.exit == 0 {
			t.Errorf("unknown subcommand exited 0")
		}
		if !strings.Contains(res.stderr, "unknown dispatch subcommand") {
			t.Errorf("unknown subcommand lacks diagnostic:\n%q", res.stderr)
		}
	})
	t.Run("no-subcommand", func(t *testing.T) {
		res := runNative("")
		if res.exit == 0 {
			t.Errorf("bare dispatch exited 0")
		}
		if !strings.Contains(res.stderr, "Usage:") {
			t.Errorf("bare dispatch lacks usage:\n%q", res.stderr)
		}
	})
}

// TestRequiredFlagGuard asserts build and show-stage-def reject a missing
// required flag with a non-zero exit and a diagnostic.
func TestRequiredFlagGuard(t *testing.T) {
	t.Run("build-missing-workflow-dir", func(t *testing.T) {
		res := runNative(`{}`, "build")
		if res.exit == 0 {
			t.Errorf("build without --workflow-dir exited 0")
		}
		if !strings.Contains(res.stderr, "--workflow-dir") {
			t.Errorf("build missing-flag diagnostic lacks --workflow-dir:\n%q", res.stderr)
		}
	})
	t.Run("show-stage-def-missing-stage", func(t *testing.T) {
		res := runNative("", "show-stage-def", "--workflow-dir", "/tmp")
		if res.exit == 0 {
			t.Errorf("show-stage-def without --stage exited 0")
		}
		if !strings.Contains(res.stderr, "--stage") {
			t.Errorf("show-stage-def missing-flag diagnostic lacks --stage:\n%q", res.stderr)
		}
	})
}
