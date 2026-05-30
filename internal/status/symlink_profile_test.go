// ABOUTME: Symlink-profile proof — a .spacedock-state/README.md -> ../README.md
// ABOUTME: layout renders, does not misdetect folder reports/artifacts, and archives whole folders.
package status

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// buildSymlinkProfile materializes the compatibility layout in a temp tree:
//
//	<tmp>/docs/dev/
//	  README.md                       (the symlink target — the real workflow README)
//	  .spacedock-state/
//	    README.md -> ../README.md     (the compatibility symlink, created at runtime)
//	    add-login.md                  (flat-form active entity)
//	    refactor-dispatch/            (folder-form; reports/ideation.md carries frontmatter)
//	    seed-archive/                 (folder-form, archived during the test)
//
// The fixture files are checked in under testdata/symlink-profile/ as plain
// files; the symlink is created here because a checked-in symlink does not
// survive a go test checkout reliably across platforms. Returns the absolute
// state-checkout dir (what --workflow-dir points at).
func buildSymlinkProfile(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", "symlink-profile"))
	if err != nil {
		t.Fatal(err)
	}
	devDir := filepath.Join(t.TempDir(), "docs", "dev")
	stateDir := filepath.Join(devDir, ".spacedock-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// README.md is the symlink target; it lives one level up from the state
	// checkout. Every other fixture file is an entity inside the state checkout.
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		from := filepath.Join(src, e.Name())
		if e.Name() == "README.md" {
			cpTree(t, from, filepath.Join(devDir, "README.md"))
			continue
		}
		cpTree(t, from, filepath.Join(stateDir, e.Name()))
	}

	if err := os.Symlink("../README.md", filepath.Join(stateDir, "README.md")); err != nil {
		t.Fatalf("create compatibility symlink: %v", err)
	}
	return stateDir
}

// tableSlugs returns the SLUG column of each data row in a status table. The
// header and the dash separator are the first two lines; every later non-empty
// line is a data row whose second whitespace-delimited field is the slug (ID and
// SLUG never contain spaces, so Fields[1] is the slug regardless of how wide the
// slug-form ID column runs).
func tableSlugs(t *testing.T, table string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(table, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("table has no header/separator rows:\n%s", table)
	}
	var slugs []string
	for _, line := range lines[2:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			t.Fatalf("malformed data row %q in table:\n%s", line, table)
		}
		slugs = append(slugs, fields[1])
	}
	return slugs
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// TestSymlinkStateProfile proves the symlink compatibility bridge end-to-end
// through the `spacedock status` argv surface (AC-4): every assertion is produced
// by invoking the runner with real argv against the constructed temp layout, not
// by calling discovery internals.
func TestSymlinkStateProfile(t *testing.T) {
	state := buildSymlinkProfile(t)
	env := pinnedEnv(t)

	wantActive := []string{"add-login", "refactor-dispatch", "seed-archive"}

	// AC-1 + AC-2: the default table renders the three active entities through the
	// symlinked README, ordered by the stages block, and never misdetects the
	// folder entities' frontmatter-bearing reports/ or artifacts/ subdirectories.
	out, stderr, code := runLauncher(t, state, env, "--workflow-dir", state)
	if code != 0 {
		t.Fatalf("default table exit=%d stderr=%q", code, stderr)
	}
	slugs := tableSlugs(t, out)

	// AC-2: the active slug set is EXACTLY the three entities — no reports,
	// artifacts, ideation, or any nested path leaks in as a row.
	if got := sortedCopy(slugs); !equalStrings(got, wantActive) {
		t.Fatalf("active slug set = %v, want %v\n--- table ---\n%s", got, wantActive, out)
	}
	if len(slugs) != 3 {
		t.Fatalf("default table data-row count = %d, want 3\n%s", len(slugs), out)
	}
	for _, bad := range []string{"reports", "artifacts", "ideation"} {
		for _, got := range slugs {
			if got == bad {
				t.Fatalf("nested path %q misdetected as an entity\n%s", bad, out)
			}
		}
	}

	// AC-1: ordering follows the stages block read through the symlink:
	// refactor-dispatch (ideation) < add-login (implementation) < seed-archive (review).
	wantOrder := []string{"refactor-dispatch", "add-login", "seed-archive"}
	if !equalStrings(slugs, wantOrder) {
		t.Fatalf("stage ordering = %v, want %v (proves stages block resolved through symlink)\n%s",
			slugs, wantOrder, out)
	}

	// AC-1: --next reads dispatchable entities, which requires the stages block to
	// resolve through the symlinked README. A broken symlink would instead fail
	// with "README.md has no stages block".
	nextOut, nextErr, nextCode := runLauncher(t, state, env, "--workflow-dir", state, "--next")
	if nextCode != 0 {
		t.Fatalf("--next exit=%d stderr=%q (stages block did not resolve through symlink)", nextCode, nextErr)
	}
	if strings.Contains(nextErr, "no stages block") {
		t.Fatalf("--next reported a missing stages block — symlink did not resolve:\n%s", nextErr)
	}
	if len(tableSlugs(t, nextOut)) == 0 {
		t.Fatalf("--next produced no dispatchable rows; stages block did not resolve:\n%s", nextOut)
	}

	// AC-3: archiving the folder entity moves the whole folder (with its reports/
	// subtree) under _archive and removes it from the active view.
	_, archErr, archCode := runLauncher(t, state, env, "--workflow-dir", state, "--archive", "seed-archive")
	if archCode != 0 {
		t.Fatalf("--archive exit=%d stderr=%q", archCode, archErr)
	}

	// (a) the source folder is gone from the active root.
	if _, err := os.Stat(filepath.Join(state, "seed-archive")); !os.IsNotExist(err) {
		t.Fatalf("active seed-archive/ should be gone after archive, stat err=%v", err)
	}
	// (b) the folder is present under _archive with an archived: stamp.
	archivedIndex := filepath.Join(state, "_archive", "seed-archive", "index.md")
	body, err := os.ReadFile(archivedIndex)
	if err != nil {
		t.Fatalf("archived index.md missing: %v", err)
	}
	if !tsRe.MatchString(stampLine(string(body))) {
		t.Fatalf("archived index.md has no well-formed archived: timestamp\n%s", body)
	}
	// (c) the reports/ subtree traveled with the folder.
	if _, err := os.Stat(filepath.Join(state, "_archive", "seed-archive", "reports", "ideation.md")); err != nil {
		t.Fatalf("archived reports/ subtree not carried: %v", err)
	}

	// (d) the default table no longer lists seed-archive.
	afterOut, afterErr, afterCode := runLauncher(t, state, env, "--workflow-dir", state)
	if afterCode != 0 {
		t.Fatalf("post-archive default table exit=%d stderr=%q", afterCode, afterErr)
	}
	if containsSlug(tableSlugs(t, afterOut), "seed-archive") {
		t.Fatalf("seed-archive still in active table after archive:\n%s", afterOut)
	}

	// (e) --archived lists seed-archive.
	archivedOut, archivedErr, archivedCode := runLauncher(t, state, env, "--workflow-dir", state, "--archived")
	if archivedCode != 0 {
		t.Fatalf("--archived exit=%d stderr=%q", archivedCode, archivedErr)
	}
	if !containsSlug(tableSlugs(t, archivedOut), "seed-archive") {
		t.Fatalf("--archived omits the archived seed-archive:\n%s", archivedOut)
	}
}

// stampLine returns the `archived:` frontmatter line, or "" when absent, so the
// timestamp can be asserted by shape rather than exact instant.
func stampLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "archived:") {
			return line
		}
	}
	return ""
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsSlug(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
