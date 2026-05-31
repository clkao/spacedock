// ABOUTME: Unit tests for the shared safehouse seam: Present detection,
// ABOUTME: Available lookPath gate + pinned hint, and inner-argv-agnostic Wrap.
package safehouse

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestPresentDetectsFile: a `.safehouse` regular file in workdir is Present.
func TestPresentDetectsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".safehouse"), []byte("profile"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !Present(dir) {
		t.Fatalf("Present(%q) = false, want true for a .safehouse file", dir)
	}
}

// TestPresentDetectsDir: a `.safehouse` directory counts identically to a file
// (os.Stat truthiness), pinning the file-vs-dir case from AC-1.
func TestPresentDetectsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".safehouse"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !Present(dir) {
		t.Fatalf("Present(%q) = false, want true for a .safehouse directory", dir)
	}
}

// TestPresentAbsent: no `.safehouse` entry → not Present.
func TestPresentAbsent(t *testing.T) {
	dir := t.TempDir()
	if Present(dir) {
		t.Fatalf("Present(%q) = true, want false with no .safehouse", dir)
	}
}

// TestAvailableResolvable: a lookPath that resolves the binary → ok, no hint.
func TestAvailableResolvable(t *testing.T) {
	look := func(string) (string, error) { return "/usr/bin/safehouse", nil }
	ok, hint := Available(look)
	if !ok {
		t.Fatalf("Available = (false, %q), want ok=true when lookPath resolves", hint)
	}
	if hint != "" {
		t.Fatalf("Available hint = %q, want empty when resolvable", hint)
	}
}

// TestAvailableNotFound: a lookPath that fails → not ok, with a pinned install
// hint naming the safehouse binary for stderr.
func TestAvailableNotFound(t *testing.T) {
	look := func(string) (string, error) { return "", errors.New("not found") }
	ok, hint := Available(look)
	if ok {
		t.Fatalf("Available ok=true, want false when lookPath fails")
	}
	if hint == "" {
		t.Fatalf("Available hint empty, want a pinned install hint")
	}
}

// TestWrapComposesPrefix: Wrap prepends the safehouse prefix and the `--`
// separator, then the inner argv, with no extra args.
func TestWrapComposesPrefix(t *testing.T) {
	inner := []string{"claude", "--agent", "spacedock:first-officer", "--foo"}
	got := Wrap(inner, nil)
	want := []string{"safehouse", "--trust-workdir-config", "--",
		"claude", "--agent", "spacedock:first-officer", "--foo"}
	if !equalArgv(got, want) {
		t.Fatalf("Wrap = %v, want %v", got, want)
	}
}

// TestWrapWithExtra: extra safehouse args land between --trust-workdir-config
// and the `--` separator, never mixed into the inner command.
func TestWrapWithExtra(t *testing.T) {
	got := Wrap([]string{"claude"}, []string{"--profile", "x"})
	want := []string{"safehouse", "--trust-workdir-config", "--profile", "x", "--", "claude"}
	if !equalArgv(got, want) {
		t.Fatalf("Wrap with extra = %v, want %v", got, want)
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
