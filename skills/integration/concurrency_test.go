// ABOUTME: AC-7 concurrency regression — the prescribed path-scoped state commit
// ABOUTME: for entity B must not sweep in a concurrent writer's already-staged entity A.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// git runs a git subcommand in dir and fails the test on error.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", dir,
		"-c", "user.email=t@t", "-c", "user.name=t"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// TestPathScopedCommitDoesNotSweepSibling reproduces the cross-attribution
// hazard the split-root contract forbids: in one shared, non-branched state
// checkout, writer A stages its entity file while writer B commits its own. The
// prescribed path-scoped commit (`git commit -- {B}`) must commit ONLY B and
// leave A's staged change intact — a bare `git commit` would sweep A in.
func TestPathScopedCommitDoesNotSweepSibling(t *testing.T) {
	checkout := t.TempDir()
	git(t, checkout, "init", "-q")

	entityA := filepath.Join(checkout, "entity-a", "index.md")
	entityB := filepath.Join(checkout, "entity-b", "index.md")
	for _, p := range []string{entityA, entityB} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("seed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, checkout, "add", "-A")
	git(t, checkout, "commit", "-q", "-m", "seed both entities")

	// Writer A stages a change to entity A (mid-flight, not yet committed).
	if err := os.WriteFile(entityA, []byte("seed\nA's in-flight change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, checkout, "add", entityA)

	// Writer B makes and commits ITS OWN change with the prescribed path-scoped
	// commit, naming only entity B's path.
	if err := os.WriteFile(entityB, []byte("seed\nB's change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, checkout, "add", entityB)
	git(t, checkout, "commit", "-m", "state: entity-b update", "--", entityB)

	// B's commit must contain ONLY entity B, never entity A.
	committed := git(t, checkout, "show", "--name-only", "--format=", "HEAD")
	if strings.Contains(committed, "entity-a/index.md") {
		t.Fatalf("path-scoped commit for B swept in entity A:\n%s", committed)
	}
	if !strings.Contains(committed, "entity-b/index.md") {
		t.Fatalf("path-scoped commit for B did not include entity B:\n%s", committed)
	}

	// A's change must still be staged afterward — uncommitted, not clobbered.
	staged := git(t, checkout, "diff", "--cached", "--name-only")
	if !strings.Contains(staged, "entity-a/index.md") {
		t.Fatalf("entity A's staged change was swept away by B's commit; staged now:\n%q", staged)
	}
}
