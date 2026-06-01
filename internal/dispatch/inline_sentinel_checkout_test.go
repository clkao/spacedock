// ABOUTME: M-2 site-level negative — splitRootStateCheckout returns "" for the
// ABOUTME: $inline sentinel, never a literal `…/$inline` checkout path.
package dispatch

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSplitRootStateCheckoutInlineSentinel pins the M-2 negative at the
// splitRootStateCheckout site: a `state: $inline` README yields "" (inline, no
// state checkout), identical to an absent state: field — never a literal
// `…/$inline` path.
func TestSplitRootStateCheckoutInlineSentinel(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state string
	}{
		{"explicit inline sentinel", "$inline"},
		{"absent state", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wf := t.TempDir()
			fm := "---\ncommissioned-by: spacedock@1\nid-style: slug\n"
			if tc.state != "" {
				fm += "state: " + tc.state + "\n"
			}
			fm += "---\n\n# WF\n"
			writeFile(t, filepath.Join(wf, "README.md"), fm)

			got := splitRootStateCheckout(wf)
			if got != "" {
				t.Fatalf("splitRootStateCheckout = %q, want \"\" (inline has no state checkout)", got)
			}
			if strings.Contains(got, "$inline") {
				t.Fatalf("splitRootStateCheckout joined the literal sentinel: %q", got)
			}
		})
	}
}
