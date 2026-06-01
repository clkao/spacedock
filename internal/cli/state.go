// ABOUTME: `spacedock state init` — resume a cloned split-root workflow by fetching
// ABOUTME: the orphan state branch and adding it as a linked worktree at state:.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spacedock-dev/spacedock/internal/status"
)

// runStateInit implements `spacedock state init`. It reads the workflow's
// `state:` and `state-branch:` from the README, and for a split-root workflow
// whose state checkout is ABSENT, fetches the orphan state branch from origin and
// checks it out as a linked worktree at the state path. The path-exists guard
// makes a second run a no-op (a raw 2nd `git worktree add` fatals "already
// exists"). An inline/empty workflow has nothing to init.
func runStateInit(ctx context.Context, args []string, env []string, dir string, stdout, stderr io.Writer) int {
	workflowDir, code := parseStateInitArgs(args, dir, stderr)
	if code != 0 {
		return code
	}

	readme := filepath.Join(workflowDir, "README.md")
	if !fileExists(readme) {
		fmt.Fprintf(stderr, "spacedock state init: no README.md at %s\n", workflowDir)
		return 1
	}
	fm := status.ParseFrontmatter(readme)
	mode, relPath, err := status.ClassifyState(fm["state"])
	if err != nil {
		fmt.Fprintf(stderr, "spacedock state init: %v\n", err)
		return 1
	}
	if mode == status.StateInline {
		fmt.Fprintln(stdout, "Inline workflow — entities live beside the README; nothing to init.")
		return 0
	}

	branch, err := status.StateBranch(workflowDir)
	if err != nil {
		fmt.Fprintf(stderr, "spacedock state init: %v\n", err)
		return 1
	}
	statePath := filepath.Join(workflowDir, relPath)

	// Path-exists guard: a present state checkout is a no-op. Refresh it from
	// origin so a resume sees peers' commits, then report. Never re-`worktree add`
	// (the spike showed a 2nd add fatals "already exists").
	if dirExists(statePath) {
		if fetchOK, _ := runGit(statePath, "fetch", "origin", branch); fetchOK {
			runGit(statePath, "pull", "--rebase", "origin", branch)
		}
		fmt.Fprintf(stdout, "State checkout already initialized at %s (branch %s).\n", statePath, branch)
		return 0
	}

	// Fresh resume: fetch the orphan branch, then add it as a linked worktree.
	if ok, out := runGit(workflowDir, "fetch", "origin", branch); !ok {
		fmt.Fprintf(stderr, "spacedock state init: git fetch origin %s failed:\n%s\n"+
			"Manual fallback: git fetch origin %s && git worktree add %s %s\n",
			branch, out, branch, statePath, branch)
		return 1
	}
	if ok, out := runGit(workflowDir, "worktree", "add", statePath, branch); !ok {
		fmt.Fprintf(stderr, "spacedock state init: git worktree add %s %s failed:\n%s\n",
			statePath, branch, out)
		return 1
	}
	fmt.Fprintf(stdout, "Initialized state checkout at %s (branch %s).\n", statePath, branch)
	return 0
}

// parseStateInitArgs reads `--workflow-dir DIR`, resolving a relative path
// against dir. With no flag it discovers the enclosing workflow from dir.
func parseStateInitArgs(args []string, dir string, stderr io.Writer) (workflowDir string, code int) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--workflow-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "spacedock state init: --workflow-dir requires a path")
				return "", 2
			}
			workflowDir = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "spacedock state init: unknown argument %q\n", args[i])
			return "", 2
		}
	}
	if workflowDir == "" {
		discovered, ok := status.DiscoverWorkflowDir(dir)
		if !ok {
			fmt.Fprintln(stderr, "spacedock state init: no workflow here — pass --workflow-dir or run inside a workflow")
			return "", 1
		}
		workflowDir = discovered
	} else if !filepath.IsAbs(workflowDir) {
		workflowDir = filepath.Join(dir, workflowDir)
	}
	return workflowDir, 0
}

// runGit runs a git command in dir, returning success and combined output. On
// failure it does NOT print — the caller decides whether the failure is fatal (a
// fresh fetch) or tolerable (a refresh pull on an already-initialized checkout).
func runGit(dir string, gitArgs ...string) (bool, string) {
	cmd := exec.Command("git", append([]string{"-C", dir}, gitArgs...)...)
	out, err := cmd.CombinedOutput()
	return err == nil, string(out)
}

// fileExists reports whether path is an existing regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// dirExists reports whether path is an existing directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
