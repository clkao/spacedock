// ABOUTME: Behavioral coverage for split-root state-commit guidance — both the
// ABOUTME: worktree and non-worktree branches must emit resolved, brace-free paths.
package dispatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStateCommitGuidanceResolvesPaths pins both defects on the split-root
// state-commit-guidance surface: a worktree stage AND a non-worktree stage must
// each emit the path-scoped state-commit instruction with the resolved absolute
// state-checkout and entity paths substituted in — and never a literal
// {state_checkout} or {entity_path} brace token.
func TestStateCommitGuidanceResolvesPaths(t *testing.T) {
	cases := []struct {
		name     string
		worktree bool
		stage    string
	}{
		{name: "worktree-stage", worktree: true, stage: "implementation"},
		{name: "nonworktree-stage", worktree: false, stage: "backlog"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()

			// Split root: the workflow dir is the README/definition root; the
			// README's `state: state-checkout` field diverges the entity/state
			// checkout to workflowDir/state-checkout. CODE (if any) lives in the
			// worktree under root.
			workflowDir := root
			stateCheckout := filepath.Join(workflowDir, "state-checkout")
			writeFile(t, filepath.Join(workflowDir, "README.md"), readmeWorktree(true))

			worktreeRel := ""
			if tc.worktree {
				worktreeRel = ".worktrees/spacedock-ensign-thing"
				if err := os.MkdirAll(filepath.Join(root, worktreeRel), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			entityPath := filepath.Join(stateCheckout, "thing", "index.md")
			writeFile(t, entityPath, entityFM("Thing", tc.stage, worktreeRel))

			gitInit(t, root)

			stdin := mergeStdin(map[string]any{
				"schema_version": 2,
				"entity_path":    entityPath,
				"workflow_dir":   workflowDir,
				"stage":          tc.stage,
				"checklist":      []string{"- a", "- b"},
				"team_name":      "fixture-team",
				"bare_mode":      false,
			}, nil)

			native := runNative(stdin, "build", "--workflow-dir", workflowDir)
			body := readDispatchBody(t, dispatchFilePathFromStdout(t, native.stdout))

			// POSITIVE: the resolved path-scoped state-commit command targets the
			// resolved state checkout (workflowDir/state-checkout), not the bare
			// workflow/definition dir, both halves.
			wantAdd := "git -C " + stateCheckout + " add " + entityPath
			wantCommit := "git -C " + stateCheckout + " commit -m \"...\" -- " + entityPath
			if !strings.Contains(body, wantAdd) {
				t.Errorf("%s: body missing resolved add command %q\n--- body ---\n%s",
					tc.name, wantAdd, body)
			}
			if !strings.Contains(body, wantCommit) {
				t.Errorf("%s: body missing resolved commit command %q\n--- body ---\n%s",
					tc.name, wantCommit, body)
			}

			// NEGATIVE (the encoded failure): no literal brace tokens survive.
			if strings.Contains(body, "{state_checkout}") {
				t.Errorf("%s: body still carries literal {state_checkout}\n--- body ---\n%s",
					tc.name, body)
			}
			if strings.Contains(body, "{entity_path}") {
				t.Errorf("%s: body still carries literal {entity_path}\n--- body ---\n%s",
					tc.name, body)
			}
		})
	}
}
