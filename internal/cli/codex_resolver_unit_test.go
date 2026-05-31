// ABOUTME: Unit tests for the codex resolver helpers — version-dir selection
// ABOUTME: (semver order), install-listing parse, CODEX_HOME, cache degradation.
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLatestVersionDirSemverOrder pins the semver-aware selection: with a stale
// cache holding both 0.9.0 and 0.10.0, the resolver must pick 0.10.0 (the newer
// install) rather than the lexically-greater 0.9.0.
func TestLatestVersionDirSemverOrder(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"0.9.0", "0.10.0", "0.12.1"} {
		if err := os.Mkdir(filepath.Join(root, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := latestVersionDir(root)
	if err != nil {
		t.Fatalf("latestVersionDir errored: %v", err)
	}
	if filepath.Base(got) != "0.12.1" {
		t.Fatalf("latestVersionDir = %q, want the 0.12.1 dir", got)
	}
}

// TestLatestVersionDirSemverNotLexical isolates the lexical-vs-semver hazard:
// 0.9.0 sorts lexically AFTER 0.10.0, so a string compare would wrongly pick the
// older version. The semver compare must pick 0.10.0.
func TestLatestVersionDirSemverNotLexical(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"0.9.0", "0.10.0"} {
		if err := os.Mkdir(filepath.Join(root, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := latestVersionDir(root)
	if err != nil {
		t.Fatalf("latestVersionDir errored: %v", err)
	}
	if filepath.Base(got) != "0.10.0" {
		t.Fatalf("latestVersionDir = %q, want the 0.10.0 dir (lexical compare would wrongly pick 0.9.0)", got)
	}
}

// TestLatestVersionDirSingleVersion is the live-cache shape today: a single
// version dir is returned as-is.
func TestLatestVersionDirSingleVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "0.12.1"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := latestVersionDir(root)
	if err != nil {
		t.Fatalf("latestVersionDir errored: %v", err)
	}
	if filepath.Base(got) != "0.12.1" {
		t.Fatalf("latestVersionDir = %q, want the 0.12.1 dir", got)
	}
}

// TestLatestVersionDirAbsentRoot: a missing cache root is not an error — it is
// the no-install degradation state, returning "" with no error.
func TestLatestVersionDirAbsentRoot(t *testing.T) {
	got, err := latestVersionDir(filepath.Join(t.TempDir(), "no-such-cache"))
	if err != nil {
		t.Fatalf("latestVersionDir errored on absent root: %v", err)
	}
	if got != "" {
		t.Fatalf("latestVersionDir = %q, want empty for an absent cache root", got)
	}
}

// TestLatestVersionDirNoSubdirs: a present root with no version subdirectories
// (only files) returns "" with no error.
func TestLatestVersionDirNoSubdirs(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a-file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := latestVersionDir(root)
	if err != nil {
		t.Fatalf("latestVersionDir errored: %v", err)
	}
	if got != "" {
		t.Fatalf("latestVersionDir = %q, want empty when root has no subdirectories", got)
	}
}

// TestCodexEntryInstalled exercises the `codex plugin list` text parse: an
// installed entry carries `<id> (installed`; a not-installed entry, or a
// listing without the id, does not match.
func TestCodexEntryInstalled(t *testing.T) {
	cases := []struct {
		name    string
		listing string
		want    bool
	}{
		{"installed", "  spacedock@spacedock (installed)", true},
		{"installed-enabled", "  spacedock@spacedock (installed, enabled)", true},
		{"not-installed", "  spacedock@spacedock (not installed)", false},
		{"other-plugin", "  other@market (installed)", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexEntryInstalled(tc.listing, "spacedock@spacedock"); got != tc.want {
				t.Fatalf("codexEntryInstalled(%q) = %v, want %v", tc.listing, got, tc.want)
			}
		})
	}
}

// TestCodexHomeFromEnv: CODEX_HOME takes precedence over ~/.codex.
func TestCodexHomeFromEnv(t *testing.T) {
	t.Setenv("CODEX_HOME", "/tmp/custom-codex")
	if got := codexHome(); got != "/tmp/custom-codex" {
		t.Fatalf("codexHome() = %q, want /tmp/custom-codex", got)
	}
}

// TestCodexHomeDefault: with CODEX_HOME unset, codexHome resolves <home>/.codex.
func TestCodexHomeDefault(t *testing.T) {
	t.Setenv("CODEX_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no user home dir: %v", err)
	}
	want := filepath.Join(home, ".codex")
	if got := codexHome(); got != want {
		t.Fatalf("codexHome() = %q, want %q", got, want)
	}
}
