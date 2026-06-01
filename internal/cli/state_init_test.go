// ABOUTME: B.2/B.3 real-git e2e — commission orphan-branch scaffolding (Spike a)
// ABOUTME: + `spacedock state init` fresh-clone resume (Spike b), no mocks.
package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/status"
)

// splitWorkflowReadme is a commissioned split-root README declaring the state
// checkout path. The state-branch defaults to spacedock-state/<basename> (B.2).
const splitWorkflowReadme = `---
commissioned-by: spacedock@1
id-style: slug
state: .spacedock-state
stages:
  states:
    - name: ideation
      initial: true
    - name: done
      terminal: true
---

# Split Workflow
`

// git runs a git command in dir, failing the test on a non-zero exit.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return string(out)
}

// gitOK runs a git command and reports whether it succeeded, returning combined
// output for the caller to inspect. Used for the path-exists-guard assertion
// where a 2nd `git worktree add` is expected to FATAL.
func gitOK(t *testing.T, dir string, args ...string) (string, bool) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

// commissionSplitWorkflow replicates the spiked Spike (a) commission mechanics
// inside hostClone: write README + gitignore the state path on the code branch,
// commit code, then birth the orphan state branch in a temp detached worktree
// (clear the inherited tree, seed an entity), check it out as a linked worktree
// at the gitignored path, and push the orphan branch to origin. Returns the
// workflow dir and the seeded entity slug.
func commissionSplitWorkflow(t *testing.T, hostClone string) (workflowDir, stateBranch, seedSlug string) {
	t.Helper()
	workflowRel := filepath.Join("docs", "dev")
	workflowDir = filepath.Join(hostClone, workflowRel)
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(workflowDir, ".spacedock-state")
	seedSlug = "first-task"
	stateBranch = "spacedock-state/dev"

	// Code-branch files: README + a tracked .gitignore for the state path (M-1
	// step 2 made durable — collaborators/fresh clones must ignore it).
	if err := os.WriteFile(filepath.Join(workflowDir, "README.md"), []byte(splitWorkflowReadme), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostClone, ".gitignore"), []byte("docs/dev/.spacedock-state/\n.worktrees/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, hostClone, "add", "docs/dev/README.md", ".gitignore")
	git(t, hostClone, "commit", "-q", "-m", "commission split workflow")

	// Birth the orphan state branch in a temp detached worktree, clearing the
	// inherited index/tree (the spike-surfaced mechanic), seeding one entity.
	tmpWT := filepath.Join(t.TempDir(), "orphan-birth")
	git(t, hostClone, "worktree", "add", "--detach", tmpWT)
	git(t, tmpWT, "checkout", "--orphan", stateBranch)
	git(t, tmpWT, "rm", "-rf", "--cached", ".")
	// Remove the inherited working-tree files so the orphan tree starts empty.
	entries, err := os.ReadDir(tmpWT)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(tmpWT, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpWT, seedSlug+".md"),
		[]byte("---\nstatus: ideation\n---\n# First Task\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, tmpWT, "add", "-A")
	git(t, tmpWT, "commit", "-q", "-m", "seed state")
	git(t, tmpWT, "push", "origin", stateBranch)
	git(t, hostClone, "worktree", "remove", "--force", tmpWT)

	// Check the orphan branch out at the gitignored state path as a linked worktree.
	git(t, hostClone, "worktree", "add", statePath, stateBranch)
	return workflowDir, stateBranch, seedSlug
}

// TestCommissionOrphanBranchScaffolding pins B.3/AC-3 + the M-4 spike-replication
// asserts for Spike (a): the orphan branch exists on origin, the state path is a
// linked worktree, the orphan's tree carries NO code-branch files (clear-inherited
// mechanic), the code branch porcelain is empty (R1), and status renders.
func TestCommissionOrphanBranchScaffolding(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bare := filepath.Join(t.TempDir(), "origin.git")
	git(t, t.TempDir(), "init", "-q", "--bare", bare)

	hostClone := filepath.Join(t.TempDir(), "host")
	git(t, t.TempDir(), "clone", "-q", bare, hostClone)
	git(t, hostClone, "config", "user.email", "t@t")
	git(t, hostClone, "config", "user.name", "t")

	workflowDir, stateBranch, seedSlug := commissionSplitWorkflow(t, hostClone)
	statePath := filepath.Join(workflowDir, ".spacedock-state")

	// Orphan branch exists on origin.
	if out := git(t, bare, "branch", "--list", stateBranch); !strings.Contains(out, stateBranch) {
		t.Fatalf("orphan branch %q not on origin; branches=%q", stateBranch, out)
	}
	// State path is a linked worktree of the host clone.
	if out := git(t, hostClone, "worktree", "list"); !strings.Contains(out, statePath) {
		t.Fatalf("state path is not a linked worktree; worktree list=%q", out)
	}
	// Clear-inherited mechanic: the orphan's tree has NO code-branch files (no
	// README.md, no .gitignore) — only the seeded entity.
	orphanFiles := git(t, statePath, "ls-tree", "--name-only", stateBranch)
	if strings.Contains(orphanFiles, "README.md") || strings.Contains(orphanFiles, ".gitignore") || strings.Contains(orphanFiles, "docs") {
		t.Fatalf("orphan tree carries inherited code-branch files: %q", orphanFiles)
	}
	if !strings.Contains(orphanFiles, seedSlug+".md") {
		t.Fatalf("orphan tree missing the seeded entity %q; tree=%q", seedSlug+".md", orphanFiles)
	}
	// R1: the code branch working tree is clean (state path is gitignored).
	if out := git(t, hostClone, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Fatalf("code branch porcelain not empty (R1 violated): %q", out)
	}
	// status renders the seeded entity from the state checkout.
	assertStatusRenders(t, workflowDir, seedSlug)
}

// TestStateInitResumesFreshClone pins B.2/AC-4 + the M-4 spike-replication asserts
// for Spike (b): a fresh clone of origin has the state path absent
// (entity_dir_present:false); `state init` fetches the orphan branch and adds the
// linked worktree so status renders; a 2nd `state init` is a no-op (path-exists
// guard), NOT a fatal (the spike showed a raw 2nd `worktree add` fatals).
func TestStateInitResumesFreshClone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bare := filepath.Join(t.TempDir(), "origin.git")
	git(t, t.TempDir(), "init", "-q", "--bare", bare)

	hostClone := filepath.Join(t.TempDir(), "host")
	git(t, t.TempDir(), "clone", "-q", bare, hostClone)
	git(t, hostClone, "config", "user.email", "t@t")
	git(t, hostClone, "config", "user.name", "t")
	commissionSplitWorkflow(t, hostClone)
	// Push the code branch (the orphan branch was already pushed in commission).
	git(t, hostClone, "push", "-q", "origin", "HEAD")

	// Fresh clone: code branch only, state path absent.
	fresh := filepath.Join(t.TempDir(), "fresh")
	git(t, t.TempDir(), "clone", "-q", bare, fresh)
	git(t, fresh, "config", "user.email", "t@t")
	git(t, fresh, "config", "user.name", "t")
	freshWorkflow := filepath.Join(fresh, "docs", "dev")
	freshState := filepath.Join(freshWorkflow, ".spacedock-state")

	if _, err := os.Stat(freshState); !os.IsNotExist(err) {
		t.Fatalf("fresh clone should NOT have the state path yet (err=%v)", err)
	}

	// Pre-init boot shows entity_dir_present:false.
	bootOut := runStatusBoot(t, freshWorkflow)
	if !strings.Contains(bootOut, "STATE_BACKEND: split-root") {
		t.Fatalf("pre-init boot should show split-root; got\n%s", bootOut)
	}
	if !strings.Contains(bootOut, "present: false") {
		t.Fatalf("pre-init boot should show present: false; got\n%s", bootOut)
	}

	// state init: fetch + worktree add.
	var out, errBuf strings.Builder
	code := run(context.Background(), []string{"state", "init", "--workflow-dir", freshWorkflow},
		os.Environ(), fresh, nil, &out, &errBuf, &status.NativeRunner{}, nil)
	if code != 0 {
		t.Fatalf("state init exit=%d stdout=%q stderr=%q", code, out.String(), errBuf.String())
	}
	if _, err := os.Stat(freshState); err != nil {
		t.Fatalf("state init did not create the state worktree: %v", err)
	}
	assertStatusRenders(t, freshWorkflow, "first-task")

	// 2nd state init is an idempotent no-op (path-exists guard) — NOT a fatal.
	var out2, errBuf2 strings.Builder
	code2 := run(context.Background(), []string{"state", "init", "--workflow-dir", freshWorkflow},
		os.Environ(), fresh, nil, &out2, &errBuf2, &status.NativeRunner{}, nil)
	if code2 != 0 {
		t.Fatalf("2nd state init must be a no-op (exit 0), got exit=%d stderr=%q", code2, errBuf2.String())
	}
	if strings.Contains(errBuf2.String(), "already exists") {
		t.Fatalf("2nd state init leaked git's 'already exists' fatal: %q", errBuf2.String())
	}
	// State still renders after the no-op re-run.
	assertStatusRenders(t, freshWorkflow, "first-task")
}

// TestStateInitInlineNoOp pins the inline/empty no-op: a non-split workflow has
// nothing to init; `state init` exits 0 with a one-liner and creates no worktree.
func TestStateInitInlineNoOp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	wf := filepath.Join(root, "wf")
	if err := os.MkdirAll(wf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wf, "README.md"),
		[]byte("---\ncommissioned-by: spacedock@1\nid-style: slug\nstate: $inline\nstages:\n  states:\n    - name: ideation\n      initial: true\n    - name: done\n      terminal: true\n---\n\n# Inline WF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "init")

	var out, errBuf strings.Builder
	code := run(context.Background(), []string{"state", "init", "--workflow-dir", wf},
		os.Environ(), root, nil, &out, &errBuf, &status.NativeRunner{}, nil)
	if code != 0 {
		t.Fatalf("inline state init exit=%d stderr=%q", code, errBuf.String())
	}
	if out.Len() == 0 {
		t.Fatalf("inline state init should print a one-liner; stdout empty")
	}
}

// assertStatusRenders runs `spacedock status` against workflowDir and asserts the
// seeded entity slug appears in the rendered table.
func assertStatusRenders(t *testing.T, workflowDir, slug string) {
	t.Helper()
	var out, errBuf strings.Builder
	code := run(context.Background(), []string{"status", "--workflow-dir", workflowDir},
		os.Environ(), workflowDir, nil, &out, &errBuf, &status.NativeRunner{}, nil)
	if code != 0 {
		t.Fatalf("status exit=%d stderr=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), slug) {
		t.Fatalf("status did not render entity %q; got\n%s", slug, out.String())
	}
}

// runStatusBoot runs `spacedock status --boot` against workflowDir and returns
// stdout.
func runStatusBoot(t *testing.T, workflowDir string) string {
	t.Helper()
	var out, errBuf strings.Builder
	code := run(context.Background(), []string{"status", "--workflow-dir", workflowDir, "--boot"},
		os.Environ(), workflowDir, nil, &out, &errBuf, &status.NativeRunner{}, nil)
	if code != 0 {
		t.Fatalf("status --boot exit=%d stderr=%q", code, errBuf.String())
	}
	return out.String()
}
