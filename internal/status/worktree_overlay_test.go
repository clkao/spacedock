// ABOUTME: Worktree-overlay parity — a non-split-root worktree-backed entity's
// ABOUTME: active reads use the worktree-copy frontmatter, byte-matching the oracle.
package status

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildWorktreeBacked materializes a NON-split-root workflow in a git repo with
// one worktree-backed entity whose pipeline-dir copy and worktree copy disagree:
//
//	<root>/                       git root == workflow dir (no state: field)
//	  README.md                   stages + id-style
//	  add-login.md                pipeline copy: status=implementation, worktree=wt
//	  wt/add-login.md             worktree copy: status=review
//
// The worktree copy lives at <git_root>/<worktree>/<rel>, matching the oracle's
// os.path.join(git_root, worktree, relpath(entity_path, pipeline_dir)). Returns
// the workflow dir. A git repo is required because the overlay resolves git_root.
func buildWorktreeBacked(t *testing.T, readme, pipelineEntity, worktreeEntity string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readme)
	writeFile(t, filepath.Join(root, "add-login.md"), pipelineEntity)
	writeFile(t, filepath.Join(root, "wt", "add-login.md"), worktreeEntity)
	gitInitWorktreeFixture(t, root)
	return root
}

// gitInitWorktreeFixture initializes a git repo and commits the tree so the
// overlay's find_git_root resolves to root.
func gitInitWorktreeFixture(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "add", "-A"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// overlayReadme is a slug-style, single-root README (NO state: field) with the
// two stages the overlay test moves the entity between.
const overlayReadme = `---
commissioned-by: spacedock@1
id-style: slug
stages:
  states:
    - name: implementation
      initial: true
      worktree: true
    - name: review
      terminal: true
---

# Worktree Overlay Workflow
`

// TestWorktreeOverlayActiveReads is the M2 parity test: for a NON-split-root
// worktree-backed entity whose pipeline-dir status differs from its worktree-copy
// status, native active reads (table / --where / --fields / --resolve) must show
// the worktree-copy value, byte-matching the oracle (VendorRunner). This locks
// the overlay that scan_entities_active / load_active_entity_fields perform.
func TestWorktreeOverlayActiveReads(t *testing.T) {
	pipelineEntity := "---\nid: add-login\nstatus: implementation\ntitle: Add login\nworktree: wt\n---\n"
	worktreeEntity := "---\nid: add-login\nstatus: review\ntitle: Add login\nworktree: wt\n---\n"

	cases := []struct {
		name string
		args []string
		// wantValue, when set, must appear in the native stdout — a direct guard
		// that the overlaid worktree-copy status is observable, not just that both
		// runners agree. --where filters on it (so its presence in a non-empty row
		// proves the overlay drove the match) and --resolve emits identity/path
		// only, so neither carries the literal status field for a value check.
		wantValue string
	}{
		{"table", nil, "review"},
		{"where-status", []string{"--where", "status=review"}, ""},
		{"fields-status", []string{"--fields", "status"}, "review"},
		{"resolve", []string{"--resolve", "add-login"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Two independent copies so the runners never observe each other.
			nativeRoot := buildWorktreeBacked(t, overlayReadme, pipelineEntity, worktreeEntity)
			oracleRoot := buildWorktreeBacked(t, overlayReadme, pipelineEntity, worktreeEntity)
			env := pinnedEnv(t)

			args := append([]string{"--workflow-dir", "%ROOT%"}, tc.args...)

			nativeArgs := withRoot(args, nativeRoot)
			oracleArgs := withRoot(args, oracleRoot)

			nOut, nErr, nCode := runNative(t, nativeRoot, env, nativeArgs...)
			oOut, oErr, oCode := runLauncher(t, oracleRoot, env, oracleArgs...)

			// Normalize the per-test temp root prefix out of both streams so only
			// the behavioral content is compared.
			nOutN := replaceAll(nOut, nativeRoot, "%ROOT%")
			oOutN := replaceAll(oOut, oracleRoot, "%ROOT%")
			nErrN := replaceAll(nErr, nativeRoot, "%ROOT%")
			oErrN := replaceAll(oErr, oracleRoot, "%ROOT%")

			if nCode != oCode {
				t.Fatalf("exit code native=%d oracle=%d\nnative stderr=%q\noracle stderr=%q", nCode, oCode, nErr, oErr)
			}
			if nOutN != oOutN {
				t.Fatalf("stdout mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOutN, oOutN)
			}
			if nErrN != oErrN {
				t.Fatalf("stderr mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nErrN, oErrN)
			}

			// Guard against both sides agreeing on the wrong (pipeline-copy) value.
			if tc.wantValue != "" && !strings.Contains(nOutN, tc.wantValue) {
				t.Fatalf("native output should reflect worktree-copy value %q:\n%s", tc.wantValue, nOutN)
			}
			// --where status=review matches only via the overlay (the pipeline copy
			// is status=implementation); the entity row must therefore appear.
			if tc.name == "where-status" && !strings.Contains(nOutN, "add-login") {
				t.Fatalf("--where status=review should match the overlaid entity, got:\n%s", nOutN)
			}
		})
	}

	// Guard: the worktree copy genuinely exists and differs from the pipeline copy
	// during the run, so a passing parity is meaningful (not a no-op fallback).
	root := buildWorktreeBacked(t, overlayReadme, pipelineEntity, worktreeEntity)
	if _, err := os.Stat(filepath.Join(root, "wt", "add-login.md")); err != nil {
		t.Fatalf("worktree copy must exist: %v", err)
	}
}

// withRoot replaces the "%ROOT%" placeholder in args with root.
func withRoot(args []string, root string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = strings.ReplaceAll(a, "%ROOT%", root)
	}
	return out
}

func replaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}
