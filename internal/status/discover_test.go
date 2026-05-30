// ABOUTME: AC-3 discovery tests — flat/folder forms, conflict warning,
// ABOUTME: reserved-dir set {_archive,_mods}, dot-dirs and _debriefs ignored.
package status

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkfile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverEntityFiles(t *testing.T) {
	root := t.TempDir()
	fm := "---\nid: x\n---\nbody\n"

	mkfile(t, root, "README.md", "---\nid-style: sequential\n---\n")
	mkfile(t, root, "alpha.md", fm)                  // flat
	mkfile(t, root, "beta/index.md", fm)             // folder
	mkfile(t, root, "gamma.md", fm)                  // conflict flat
	mkfile(t, root, "gamma/index.md", fm)            // conflict folder (wins)
	mkfile(t, root, "_archive/old.md", fm)           // reserved dir
	mkfile(t, root, "_mods/hook.md", fm)             // reserved dir
	mkfile(t, root, "_debriefs/note.md", fm)         // un-reserved, no index.md
	mkfile(t, root, ".hidden/index.md", fm)          // dot-dir ignored
	mkfile(t, root, "nofence.md", "no fence here\n") // fence-less, skipped
	mkfile(t, root, "notmd.txt", "x")                // non-.md, skipped

	var stderr bytes.Buffer
	got := discoverEntityFiles(root, &stderr)

	var slugs []string
	pathBySlug := map[string]string{}
	for _, pair := range got {
		slugs = append(slugs, pair[0])
		pathBySlug[pair[0]] = pair[1]
	}

	want := []string{"alpha", "beta", "gamma"}
	if strings.Join(slugs, ",") != strings.Join(want, ",") {
		t.Fatalf("slugs = %v, want %v", slugs, want)
	}

	// Folder form wins for gamma.
	if !strings.HasSuffix(pathBySlug["gamma"], filepath.Join("gamma", "index.md")) {
		t.Fatalf("gamma should resolve to folder form, got %s", pathBySlug["gamma"])
	}
	// alpha is flat.
	if !strings.HasSuffix(pathBySlug["alpha"], "alpha.md") {
		t.Fatalf("alpha should be flat, got %s", pathBySlug["alpha"])
	}

	// Conflict warning emitted for gamma only.
	warn := stderr.String()
	if !strings.Contains(warn, "entity 'gamma' has both") {
		t.Fatalf("expected gamma conflict warning, got %q", warn)
	}
	if strings.Contains(warn, "_debriefs") || strings.Contains(warn, "alpha") {
		t.Fatalf("unexpected warning content: %q", warn)
	}
}

func TestResolveEntityPathFolderWins(t *testing.T) {
	root := t.TempDir()
	fm := "---\nid: x\n---\n"
	mkfile(t, root, "z.md", fm)
	mkfile(t, root, "z/index.md", fm)

	var stderr bytes.Buffer
	got := resolveEntityPath(root, "z", &stderr)
	if !strings.HasSuffix(got, filepath.Join("z", "index.md")) {
		t.Fatalf("resolveEntityPath = %s, want folder form", got)
	}
	if !strings.Contains(stderr.String(), "preferring folder form") {
		t.Fatalf("expected conflict warning, got %q", stderr.String())
	}
}

func TestResolveEntityPathMissing(t *testing.T) {
	var stderr bytes.Buffer
	if got := resolveEntityPath(t.TempDir(), "nope", &stderr); got != "" {
		t.Fatalf("expected empty for missing slug, got %q", got)
	}
}
