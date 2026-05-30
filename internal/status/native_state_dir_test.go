// ABOUTME: Split-root (`state:`) proof — resolveRoots diverges entityDir from
// ABOUTME: definitionDir, and status/--set/--archive/discovery honor the split.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes content to path, creating parent dirs, failing the test on
// error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestResolveRootsStateField is the AC-1 resolver unit test: `state:` set/absent/
// empty diverge entityDir from definitionDir, and an absolute or escaping value
// is rejected.
func TestResolveRootsStateField(t *testing.T) {
	cases := []struct {
		name      string
		readme    string
		wantState string // expected entityDir suffix relative to definitionDir, "" => same dir
	}{
		{"state set", "---\nstate: .spacedock-state\n---\n", ".spacedock-state"},
		{"state absent", "---\nid-style: slug\n---\n", ""},
		{"state empty", "---\nstate:\n---\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			def := t.TempDir()
			writeFile(t, filepath.Join(def, "README.md"), tc.readme)
			r, err := resolveRoots(def, "")
			if err != nil {
				t.Fatalf("resolveRoots returned error: %v", err)
			}
			if r.definitionDir != def {
				t.Fatalf("definitionDir = %q, want %q", r.definitionDir, def)
			}
			wantEntity := def
			if tc.wantState != "" {
				wantEntity = filepath.Join(def, tc.wantState)
			}
			if r.entityDir != wantEntity {
				t.Fatalf("entityDir = %q, want %q", r.entityDir, wantEntity)
			}
		})
	}
}

// TestResolveRootsStateRejected is the AC-1 malformed case: an absolute or
// `..`-escaping state value is rejected with an error rather than followed.
func TestResolveRootsStateRejected(t *testing.T) {
	cases := []struct {
		name   string
		readme string
	}{
		{"absolute", "---\nstate: /abs\n---\n"},
		{"escape", "---\nstate: ../escape\n---\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			def := t.TempDir()
			writeFile(t, filepath.Join(def, "README.md"), tc.readme)
			if _, err := resolveRoots(def, ""); err == nil {
				t.Fatalf("expected resolveRoots to reject %q, got nil error", tc.readme)
			}
		})
	}
}

// buildSplitRoot materializes a native split-root layout with NO
// .spacedock-state/README.md symlink:
//
//	<def>/README.md            (defines stages + id-style)
//	<def>/.spacedock-state/    (active entities + _archive live here)
//
// Returns the definition dir (what --workflow-dir points at) and the state dir.
func buildSplitRoot(t *testing.T, readme string, entities map[string]string) (string, string) {
	t.Helper()
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), readme)
	state := filepath.Join(def, ".spacedock-state")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range entities {
		writeFile(t, filepath.Join(state, name), content)
	}
	return def, state
}

// splitRootReadme is a slug-style README defining three stages in dispatch order,
// used by the split-root integration fixtures.
const splitRootReadme = `---
commissioned-by: spacedock@1
id-style: slug
state: .spacedock-state
stages:
  states:
    - name: ideation
      initial: true
    - name: implementation
    - name: review
      terminal: true
---

# Split-Root Workflow
`

// TestSplitRootStatusNoSymlink is AC-2: status lists entities from the state
// checkout and renders stage columns from the main README, with no
// .spacedock-state/README.md present.
func TestSplitRootStatusNoSymlink(t *testing.T) {
	def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md":         "---\nstatus: implementation\n---\n",
		"refactor-dispatch.md": "---\nstatus: ideation\n---\n",
	})
	env := pinnedEnv(t)

	// Guard: no README symlink/file exists in the state checkout during the run.
	if _, err := os.Lstat(filepath.Join(state, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("state checkout must have no README.md, lstat err=%v", err)
	}

	out, stderr, code := runNative(t, def, env, "--workflow-dir", def)
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%q", code, stderr)
	}
	slugs := tableSlugs(t, out)
	if got := sortedCopy(slugs); !equalStrings(got, []string{"add-login", "refactor-dispatch"}) {
		t.Fatalf("active slugs = %v, want [add-login refactor-dispatch]\n%s", got, out)
	}
	// Stage ordering proves stages came from the main README: ideation < impl.
	if !equalStrings(slugs, []string{"refactor-dispatch", "add-login"}) {
		t.Fatalf("stage ordering = %v, want [refactor-dispatch add-login]\n%s", slugs, out)
	}
}

// TestSplitRootStagesAndIdentity is AC-3: stages, id-style, and a duplicate-ID
// validation all derive from the main README + state-checkout entities.
func TestSplitRootStagesAndIdentity(t *testing.T) {
	readme := `---
commissioned-by: spacedock@1
id-style: sequential
state: .spacedock-state
stages:
  states:
    - name: ideation
      initial: true
    - name: done
      terminal: true
---

# Dup-ID Workflow
`
	def, state := buildSplitRoot(t, readme, map[string]string{
		"alpha.md": "---\nid: 1\nstatus: ideation\n---\n",
		"beta.md":  "---\nid: 1\nstatus: ideation\n---\n",
	})
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, def, env, "--workflow-dir", def, "--validate")
	if code == 0 {
		t.Fatalf("--validate should fail on duplicate id sourced from state checkout\nstdout=%q stderr=%q", out, stderr)
	}
	// The duplicate-id diagnostic must name both colliding entities by their
	// state-checkout paths, proving identity allocation reads entities from the
	// state dir while stages/id-style come from the definition-dir README.
	if !strings.Contains(stderr, "duplicate id") {
		t.Fatalf("validation error should report a duplicate id; stderr=%q", stderr)
	}
	for _, slug := range []string{"alpha", "beta"} {
		wantPath := filepath.Join(state, slug+".md")
		if !strings.Contains(stderr, wantPath) {
			t.Fatalf("duplicate-id diagnostic should reference state path %q; stderr=%q", wantPath, stderr)
		}
	}
}

// TestSplitRootSetMutatesOnlyState is AC-4: --set rewrites the entity under the
// state checkout and changes nothing under the definition dir (README et al.).
func TestSplitRootSetMutatesOnlyState(t *testing.T) {
	def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md": "---\nstatus: ideation\n---\n",
	})
	env := pinnedEnv(t)

	readmeBefore := readBytes(t, filepath.Join(def, "README.md"))
	defSnap := snapshotDir(t, def, state)

	out, stderr, code := runNative(t, def, env, "--workflow-dir", def, "--set", "add-login", "status=implementation")
	if code != 0 {
		t.Fatalf("--set exit=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(out, "status: ideation -> implementation") {
		t.Fatalf("--set narration = %q, want status transition", out)
	}

	// The entity under the state checkout changed.
	entity := readBytes(t, filepath.Join(state, "add-login.md"))
	if !strings.Contains(entity, "status: implementation") {
		t.Fatalf("state entity not updated:\n%s", entity)
	}
	// README untouched.
	if got := readBytes(t, filepath.Join(def, "README.md")); got != readmeBefore {
		t.Fatalf("README.md was modified by --set:\n%s", got)
	}
	// No definition-dir file outside the state checkout changed.
	if diff := defSnap.diff(t, def, state); diff != "" {
		t.Fatalf("definition dir churned outside state checkout:\n%s", diff)
	}
}

// TestSplitRootArchiveMovesOnlyState is AC-5: --archive moves flat and folder
// entities under <state>/_archive and touches no definition-dir file.
func TestSplitRootArchiveMovesOnlyState(t *testing.T) {
	def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
		"add-login.md":                   "---\nstatus: ideation\n---\n",
		"refactor-dispatch/index.md":     "---\nstatus: ideation\n---\n",
		"refactor-dispatch/reports/x.md": "ideation notes\n",
	})
	env := pinnedEnv(t)
	defSnap := snapshotDir(t, def, state)

	for _, slug := range []string{"add-login", "refactor-dispatch"} {
		_, stderr, code := runNative(t, def, env, "--workflow-dir", def, "--archive", slug)
		if code != 0 {
			t.Fatalf("--archive %s exit=%d stderr=%q", slug, code, stderr)
		}
	}

	// Sources gone from the active state root.
	if _, err := os.Stat(filepath.Join(state, "add-login.md")); !os.IsNotExist(err) {
		t.Fatalf("flat source should be gone, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(state, "refactor-dispatch")); !os.IsNotExist(err) {
		t.Fatalf("folder source should be gone, err=%v", err)
	}
	// Destinations exist under the state checkout's _archive (never the def dir).
	if !isRegularFile(filepath.Join(state, "_archive", "add-login.md")) {
		t.Fatalf("flat entity not archived under state/_archive")
	}
	if !isRegularFile(filepath.Join(state, "_archive", "refactor-dispatch", "index.md")) {
		t.Fatalf("folder entity not archived under state/_archive")
	}
	if !isRegularFile(filepath.Join(state, "_archive", "refactor-dispatch", "reports", "x.md")) {
		t.Fatalf("folder report subtree not carried into archive")
	}
	// No _archive under the definition dir.
	if _, err := os.Stat(filepath.Join(def, "_archive")); !os.IsNotExist(err) {
		t.Fatalf("definition dir must not get an _archive, err=%v", err)
	}
	// Definition dir outside the state checkout untouched.
	if diff := defSnap.diff(t, def, state); diff != "" {
		t.Fatalf("definition dir churned outside state checkout:\n%s", diff)
	}
}

// TestSplitRootDiscoverySingleCount is AC-6: discovery returns the definition
// dir exactly once, with and without a stray symlinked state README.
func TestSplitRootDiscoverySingleCount(t *testing.T) {
	t.Run("no state README", func(t *testing.T) {
		def, _ := buildSplitRoot(t, splitRootReadme, map[string]string{
			"add-login.md": "---\nstatus: ideation\n---\n",
		})
		got := discoverWorkflows(def)
		if len(got) != 1 || realpathOf(got[0]) != realpathOf(def) {
			t.Fatalf("discovery = %v, want exactly [%s]", got, def)
		}
	})

	t.Run("stray state README symlink", func(t *testing.T) {
		def, state := buildSplitRoot(t, splitRootReadme, map[string]string{
			"add-login.md": "---\nstatus: ideation\n---\n",
		})
		if err := os.Symlink("../README.md", filepath.Join(state, "README.md")); err != nil {
			t.Fatalf("create stray symlink: %v", err)
		}
		got := discoverWorkflows(def)
		if len(got) != 1 || realpathOf(got[0]) != realpathOf(def) {
			t.Fatalf("discovery = %v, want exactly [%s] (state checkout must be pruned)", got, def)
		}
	})
}

// readBytes reads a file as a string, failing the test on error.
func readBytes(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// dirSnapshot records file paths + contents under a root, excluding a subtree.
type dirSnapshot struct {
	files map[string]string
}

// snapshotDir snapshots every regular file under root except those under
// exclude, capturing relative path -> content.
func snapshotDir(t *testing.T, root, exclude string) dirSnapshot {
	t.Helper()
	snap := dirSnapshot{files: map[string]string{}}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if exclude != "" && strings.HasPrefix(path, exclude+string(filepath.Separator)) {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		snap.files[rel] = readBytes(t, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// diff returns a human-readable description of changes between the snapshot and
// the current tree under root (excluding the exclude subtree), or "" if none.
func (s dirSnapshot) diff(t *testing.T, root, exclude string) string {
	t.Helper()
	now := snapshotDir(t, root, exclude)
	var b strings.Builder
	for rel, before := range s.files {
		after, ok := now.files[rel]
		if !ok {
			b.WriteString("removed: " + rel + "\n")
		} else if after != before {
			b.WriteString("changed: " + rel + "\n")
		}
	}
	for rel := range now.files {
		if _, ok := s.files[rel]; !ok {
			b.WriteString("added: " + rel + "\n")
		}
	}
	return b.String()
}
