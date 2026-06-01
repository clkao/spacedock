// ABOUTME: AC-4(ii) writer-quoting — a --set value containing a space-then-#
// ABOUTME: is written quoted so it round-trips; values without ` #` are byte-preserved.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateFrontmatterQuotesHashValue locks the writer half of option C: when
// --set writes a value containing a space-then-`#` (` #`), the writer quotes it
// so the reader (which strips an unquoted whitespace-preceded comment) reads the
// full value back — a clean round-trip. A value WITHOUT ` #` is written
// unquoted (byte-preservation).
func TestUpdateFrontmatterQuotesHashValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "e.md")
	const seed = "---\nid: e\nstatus: backlog\nsource: roadmap\n---\nbody\n"
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a value containing a space-then-# .
	if _, err := updateFrontmatter(path, []fieldUpdate{{field: "source", value: "consolidates #223, #217", hasValue: true}}); err != nil {
		t.Fatalf("updateFrontmatter: %v", err)
	}
	b, _ := os.ReadFile(path)
	written := string(b)
	if !strings.Contains(written, `source: "consolidates #223, #217"`) {
		t.Fatalf("expected the #-bearing value to be written quoted, got:\n%s", written)
	}
	// Round-trip: the reader yields the full value (quote-protected), not a
	// truncated comment-stripped fragment.
	if got := ParseFrontmatter(path)["source"]; got != "consolidates #223, #217" {
		t.Fatalf("round-trip read = %q, want %q", got, "consolidates #223, #217")
	}

	// A value WITHOUT ` #` is written unquoted (byte-preservation).
	if _, err := updateFrontmatter(path, []fieldUpdate{{field: "status", value: "implementation", hasValue: true}}); err != nil {
		t.Fatalf("updateFrontmatter: %v", err)
	}
	b, _ = os.ReadFile(path)
	if !strings.Contains(string(b), "status: implementation\n") {
		t.Fatalf("expected a plain value written unquoted, got:\n%s", string(b))
	}
}
