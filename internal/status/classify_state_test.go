// ABOUTME: B.1 classifier proof — ClassifyState maps $inline/empty to inline and
// ABOUTME: a relative path to split-root, the single shared state: interpreter.
package status

import (
	"path/filepath"
	"testing"
)

// TestClassifyState covers the three modes and the absolute/escape rejection the
// classifier owns for all three read sites. $inline and empty both classify as
// inline with no path to join; a relative path classifies as split-root with the
// cleaned path; an absolute or `..`-escaping value is rejected.
func TestClassifyState(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		wantMode StateMode
		wantPath string
		wantErr  bool
	}{
		{"empty", "", StateInline, "", false},
		{"whitespace", "   ", StateInline, "", false},
		{"explicit inline sentinel", "$inline", StateInline, "", false},
		{"inline sentinel with surrounding space", "  $inline  ", StateInline, "", false},
		{"relative path", ".spacedock-state", StateSplitRoot, ".spacedock-state", false},
		{"nested relative path", "state/dev", StateSplitRoot, filepath.Join("state", "dev"), false},
		{"dot-prefixed relative", "./.spacedock-state", StateSplitRoot, ".spacedock-state", false},
		{"absolute rejected", "/abs/state", StateInline, "", true},
		{"dotdot escape rejected", "../escape", StateInline, "", true},
		{"dotdot bare rejected", "..", StateInline, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mode, relPath, err := ClassifyState(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ClassifyState(%q) = (%v,%q,nil), want error", tc.value, mode, relPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("ClassifyState(%q) returned error: %v", tc.value, err)
			}
			if mode != tc.wantMode {
				t.Fatalf("ClassifyState(%q) mode = %v, want %v", tc.value, mode, tc.wantMode)
			}
			if relPath != tc.wantPath {
				t.Fatalf("ClassifyState(%q) relPath = %q, want %q", tc.value, relPath, tc.wantPath)
			}
		})
	}
}

// TestClassifyStateInlineNeverJoinsLiteral is the M-2 negative: the $inline
// sentinel must never become a path to join. A split-root mode is the only mode
// that yields a non-empty relPath; inline (from $inline or empty) yields "".
func TestClassifyStateInlineNeverJoinsLiteral(t *testing.T) {
	for _, v := range []string{"", "$inline", "  $inline  "} {
		mode, relPath, err := ClassifyState(v)
		if err != nil {
			t.Fatalf("ClassifyState(%q) error: %v", v, err)
		}
		if mode != StateInline {
			t.Fatalf("ClassifyState(%q) mode = %v, want inline", v, mode)
		}
		if relPath != "" {
			t.Fatalf("ClassifyState(%q) relPath = %q, want empty (never a literal $inline join)", v, relPath)
		}
	}
}
