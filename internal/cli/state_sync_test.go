// ABOUTME: B.6 real-git 2-writer sync e2e — push / non-ff rejection / pull-rebase /
// ABOUTME: re-push (Spike c) plus the M-3 same-entity rebase-conflict halt path.
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cloneOnStateBranch makes a working clone of the bare origin and checks out the
// shared state branch, configured with a committer identity. Returns the clone
// path. This models a host's state checkout participating in the shared orphan
// state branch (the e2e exercises the state branch directly, the unit the sync
// contract governs).
func cloneOnStateBranch(t *testing.T, bare, name, stateBranch string) string {
	t.Helper()
	clone := filepath.Join(t.TempDir(), name)
	git(t, t.TempDir(), "clone", "-q", "-b", stateBranch, bare, clone)
	git(t, clone, "config", "user.email", name+"@t")
	git(t, clone, "config", "user.name", name)
	return clone
}

// commitEntity writes/overwrites an entity file in the clone and commits it
// path-scoped (the concurrency-safe commit rule), returning nothing. fail the
// test on git error.
func commitEntity(t *testing.T, clone, slug, body, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(clone, slug+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, clone, "add", slug+".md")
	git(t, clone, "commit", "-q", "-m", msg, "--", slug+".md")
}

// seedStateBranch births a state branch on the bare origin with one entity, the
// minimal fixture the 2-writer sync e2e shares. Returns the state branch name.
func seedStateBranch(t *testing.T, bare string) string {
	t.Helper()
	stateBranch := "spacedock-state/dev"
	seed := filepath.Join(t.TempDir(), "seed")
	git(t, t.TempDir(), "clone", "-q", bare, seed)
	git(t, seed, "config", "user.email", "seed@t")
	git(t, seed, "config", "user.name", "seed")
	git(t, seed, "checkout", "--orphan", stateBranch)
	// A fresh bare clone has no commits, so the orphan index is already empty;
	// rm --cached is tolerated as a no-op (and clears any inherited tree when the
	// origin did carry a code branch).
	gitOK(t, seed, "rm", "-rf", "--cached", ".")
	entries, _ := os.ReadDir(seed)
	for _, e := range entries {
		if e.Name() != ".git" {
			os.RemoveAll(filepath.Join(seed, e.Name()))
		}
	}
	if err := os.WriteFile(filepath.Join(seed, "alpha.md"),
		[]byte("---\nstatus: ideation\n---\n# Alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, seed, "add", "-A")
	git(t, seed, "commit", "-q", "-m", "seed alpha")
	git(t, seed, "push", "origin", stateBranch)
	return stateBranch
}

// TestTwoWriterSyncHappyPath replicates Spike (c): host A commits+pushes a
// path-scoped entity; host B's push is REJECTED (non-fast-forward); B
// `pull --rebase` replays B's distinct-file commit atop A's with ZERO conflict →
// both entities present + linear history; B re-pushes; A `pull --rebase` sees B's.
func TestTwoWriterSyncHappyPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bare := filepath.Join(t.TempDir(), "origin.git")
	git(t, t.TempDir(), "init", "-q", "--bare", bare)
	stateBranch := seedStateBranch(t, bare)

	hostA := cloneOnStateBranch(t, bare, "hostA", stateBranch)
	hostB := cloneOnStateBranch(t, bare, "hostB", stateBranch)

	// A commits a distinct entity and pushes first.
	commitEntity(t, hostA, "beta", "---\nstatus: ideation\n---\n# Beta (A)\n", "A: add beta")
	git(t, hostA, "push", "origin", stateBranch)

	// B commits a DIFFERENT distinct entity; its push is rejected (non-ff).
	commitEntity(t, hostB, "gamma", "---\nstatus: ideation\n---\n# Gamma (B)\n", "B: add gamma")
	if out, ok := gitOK(t, hostB, "push", "origin", stateBranch); ok {
		t.Fatalf("B's push should be REJECTED (non-fast-forward); it succeeded:\n%s", out)
	}

	// B pull --rebase: distinct files → zero conflict, B's commit replays atop A's.
	if out, ok := gitOK(t, hostB, "pull", "--rebase", "origin", stateBranch); !ok {
		t.Fatalf("B pull --rebase should replay cleanly (distinct files); it failed:\n%s", out)
	}
	// Both entities present in B's tree.
	for _, slug := range []string{"alpha", "beta", "gamma"} {
		if _, err := os.Stat(filepath.Join(hostB, slug+".md")); err != nil {
			t.Fatalf("B tree missing %s after pull --rebase: %v", slug, err)
		}
	}
	// Linear history: no merge commit (all commits have a single parent).
	if out := git(t, hostB, "log", "--merges", "--oneline"); strings.TrimSpace(out) != "" {
		t.Fatalf("history is not linear after pull --rebase; merge commits:\n%s", out)
	}
	// B re-pushes successfully.
	git(t, hostB, "push", "origin", stateBranch)
	// A pull --rebase sees B's gamma.
	git(t, hostA, "pull", "--rebase", "origin", stateBranch)
	if _, err := os.Stat(filepath.Join(hostA, "gamma.md")); err != nil {
		t.Fatalf("A tree missing gamma after pull --rebase: %v", err)
	}
}

// TestTwoWriterSameEntityConflictHalts pins the M-3 contract behavior: when two
// writers edit the SAME entity's frontmatter concurrently, B's `pull --rebase`
// CONFLICTS; the writer aborts the rebase (clean checkout, no merge in progress)
// and does NOT force-push — the halt. The e2e asserts the conflict is real, the
// abort restores a clean tree, and a plain re-push stays rejected (B did not
// force-push to clobber A).
func TestTwoWriterSameEntityConflictHalts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bare := filepath.Join(t.TempDir(), "origin.git")
	git(t, t.TempDir(), "init", "-q", "--bare", bare)
	stateBranch := seedStateBranch(t, bare)

	hostA := cloneOnStateBranch(t, bare, "hostA", stateBranch)
	hostB := cloneOnStateBranch(t, bare, "hostB", stateBranch)

	// Both edit the SAME entity (alpha) frontmatter concurrently.
	commitEntity(t, hostA, "alpha", "---\nstatus: implementation\n---\n# Alpha (A)\n", "A: alpha -> implementation")
	git(t, hostA, "push", "origin", stateBranch)
	commitEntity(t, hostB, "alpha", "---\nstatus: review\n---\n# Alpha (B)\n", "B: alpha -> review")

	// B's push is rejected (non-ff).
	if _, ok := gitOK(t, hostB, "push", "origin", stateBranch); ok {
		t.Fatalf("B's push should be rejected (non-fast-forward)")
	}
	// B pull --rebase CONFLICTS (same file, divergent frontmatter).
	out, ok := gitOK(t, hostB, "pull", "--rebase", "origin", stateBranch)
	if ok {
		t.Fatalf("same-entity pull --rebase should CONFLICT; it succeeded:\n%s", out)
	}
	if !strings.Contains(out, "CONFLICT") && !strings.Contains(out, "conflict") {
		t.Fatalf("pull --rebase output should report a conflict; got:\n%s", out)
	}
	// A rebase is in progress (the conflict state the contract halts on).
	if _, err := os.Stat(filepath.Join(hostB, ".git", "rebase-merge")); err != nil {
		if _, err2 := os.Stat(filepath.Join(hostB, ".git", "rebase-apply")); err2 != nil {
			t.Fatalf("expected a rebase in progress after the conflict; none found")
		}
	}

	// The M-3 halt: abort the rebase (do NOT auto-resolve, do NOT force-push).
	git(t, hostB, "rebase", "--abort")
	// The abort leaves a clean checkout.
	if out := git(t, hostB, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Fatalf("rebase --abort should leave a clean checkout; porcelain:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(hostB, ".git", "rebase-merge")); err == nil {
		t.Fatalf("rebase still in progress after --abort")
	}
	// B did NOT force-push, so a plain push stays rejected — A's edit survives on
	// origin (the contract: no silent clobber of a peer's frontmatter edit).
	if _, ok := gitOK(t, hostB, "push", "origin", stateBranch); ok {
		t.Fatalf("a plain (non-force) push after abort must stay rejected; A's edit would be clobbered otherwise")
	}
	// A's edit is what's on origin (verify via a fresh read of the bare ref).
	originAlpha := showOriginFile(t, bare, stateBranch, "alpha.md")
	if !strings.Contains(originAlpha, "status: implementation") {
		t.Fatalf("origin alpha should carry A's edit (status: implementation); got:\n%s", originAlpha)
	}
}

// showOriginFile reads a file's contents at the tip of branch in the bare repo.
func showOriginFile(t *testing.T, bare, branch, file string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", bare, "show", branch+":"+file)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git show %s:%s in %s: %v\n%s", branch, file, bare, err, out)
	}
	return string(out)
}
