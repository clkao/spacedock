// ABOUTME: AC-1 (#230) next-suppressed-by visibility surface — the computed
// ABOUTME: suppression reason is observable via --fields/--where, gated out of --all-fields, native vs oracle.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fieldCell returns the value of the named extra column for slug in a status
// table rendered with that single extra field. It locates the data row by slug
// and returns the trailing cell text (the extra column is rendered last).
func extraCellFor(t *testing.T, out, slug string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == slug {
			// The extra column is the final cell; with a single --fields column
			// it is the last whitespace-trimmed token, or empty if absent.
			last := fields[len(fields)-1]
			// The default columns are id slug status title score source; a
			// non-empty extra adds one more token after source.
			if len(fields) >= 7 {
				return last
			}
			return ""
		}
	}
	t.Fatalf("slug %q not found in table:\n%s", slug, out)
	return ""
}

// TestNextSuppressedByObservable locks the #230 visibility surface: the computed
// next-suppressed-by reason mirrors exactly why computeDispatchable skipped an
// entity, is observable via --fields and filterable via --where, and is GATED
// out of --all-fields (a parity-pinned frontmatter-keys surface). Native vs
// oracle throughout.
func TestNextSuppressedByObservable(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "suppress-workflow")

	t.Run("fields-shows-reason", func(t *testing.T) {
		nOut, nErr, nCode := runNative(t, root, env, "--workflow-dir", root, "--fields", "next-suppressed-by")
		oOut, oErr, oCode := runOracle(t, root, env, "--workflow-dir", root, "--fields", "next-suppressed-by")
		if nCode != 0 || oCode != 0 {
			t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
		}
		if nOut != oOut {
			t.Fatalf("--fields native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
		}
		want := map[string]string{
			"building": "worktree-set",
			"waiting":  "concurrency-full",
			"gated":    "gate",
		}
		for slug, reason := range want {
			if got := extraCellFor(t, nOut, slug); got != reason {
				t.Fatalf("next-suppressed-by for %q = %q, want %q\n%s", slug, got, reason, nOut)
			}
		}
	})

	t.Run("where-filters-by-reason", func(t *testing.T) {
		nOut, _, nCode := runNative(t, root, env, "--workflow-dir", root, "--where", "next-suppressed-by = concurrency-full")
		oOut, _, oCode := runOracle(t, root, env, "--workflow-dir", root, "--where", "next-suppressed-by = concurrency-full")
		if nCode != 0 || oCode != 0 {
			t.Fatalf("exit: native=%d oracle=%d", nCode, oCode)
		}
		if nOut != oOut {
			t.Fatalf("--where native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
		}
		if !strings.Contains(nOut, "waiting") {
			t.Fatalf("--where concurrency-full should surface waiting:\n%s", nOut)
		}
		for _, other := range []string{"building", "gated"} {
			if strings.Contains(nOut, other) {
				t.Fatalf("--where concurrency-full must not surface %q:\n%s", other, nOut)
			}
		}
	})

	t.Run("all-fields-excludes-computed", func(t *testing.T) {
		nOut, _, nCode := runNative(t, root, env, "--workflow-dir", root, "--all-fields")
		oOut, _, oCode := runOracle(t, root, env, "--workflow-dir", root, "--all-fields")
		if nCode != 0 || oCode != 0 {
			t.Fatalf("exit: native=%d oracle=%d", nCode, oCode)
		}
		if nOut != oOut {
			t.Fatalf("--all-fields native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
		}
		if strings.Contains(strings.ToUpper(nOut), "NEXT-SUPPRESSED-BY") {
			t.Fatalf("--all-fields must NOT surface the computed next-suppressed-by column:\n%s", nOut)
		}
	})
}

// TestNextSuppressedByEmptyForDispatchable locks the "" value: an entity that is
// actually dispatchable (not suppressed) has an empty next-suppressed-by, native
// vs oracle. Uses a fixture where the next stage has room.
func TestNextSuppressedByEmptyForDispatchable(t *testing.T) {
	env := pinnedEnv(t)
	root := t.TempDir()
	readme := `---
entity-type: task
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: ideation
      initial: true
    - name: build
    - name: done
      terminal: true
---

# dispatchable fixture
`
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: ready\ntitle: ready\nstatus: ideation\nscore: \"0.5\"\nsource: probe\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(root, "ready.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInit(t, root)

	nOut, nErr, nCode := runNative(t, root, env, "--workflow-dir", root, "--fields", "next-suppressed-by")
	oOut, oErr, oCode := runOracle(t, root, env, "--workflow-dir", root, "--fields", "next-suppressed-by")
	if nCode != 0 || oCode != 0 {
		t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
	}
	if nOut != oOut {
		t.Fatalf("--fields native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
	}
	if got := extraCellFor(t, nOut, "ready"); got != "" {
		t.Fatalf("next-suppressed-by for dispatchable 'ready' = %q, want empty\n%s", got, nOut)
	}
}
