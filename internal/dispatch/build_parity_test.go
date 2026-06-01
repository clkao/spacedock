// ABOUTME: Build golden parity — native dispatch build vs the project-vendored
// ABOUTME: Python oracle across the slug/split/worktree/team cross-product + branches.
package dispatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// README with a worktree implementation stage; split-root toggled by the caller
// adding a state: field. Models are unset (model-precedence has its own fixture).
func readmeWorktree(splitRoot bool) string {
	state := ""
	if splitRoot {
		state = "state: state-checkout\n"
	}
	return "---\n" +
		"entity-type: task\n" +
		"id-style: slug\n" +
		state +
		"stages:\n" +
		"  defaults:\n" +
		"    worktree: false\n" +
		"    concurrency: 1\n" +
		"  states:\n" +
		"    - name: backlog\n" +
		"      initial: true\n" +
		"    - name: implementation\n" +
		"      worktree: true\n" +
		"    - name: validation\n" +
		"      worktree: true\n" +
		"      feedback-to: implementation\n" +
		"    - name: done\n" +
		"      terminal: true\n" +
		"---\n" +
		"# Fixture Workflow\n" +
		"\n" +
		"### backlog\n\nseed.\n\n- **Outputs:** x.\n\n" +
		"### implementation\n\nwork.\n\n- **Outputs:** y.\n\n" +
		"### validation\n\nverify.\n\n- **Outputs:** z.\n\n" +
		"### done\n\nterm.\n"
}

// entityFM builds an entity file with the given title, status, and worktree
// frontmatter value.
func entityFM(title, status, worktree string) string {
	return "---\n" +
		"id: \"001\"\n" +
		"title: " + title + "\n" +
		"status: " + status + "\n" +
		"worktree: " + worktree + "\n" +
		"---\n" +
		"# " + title + "\n\nBody.\n"
}

// buildCase is a positive/branch parity fixture: it builds a fixture tree, runs
// both native and oracle on the same stdin, and asserts three-channel parity
// (with the fetch line rewritten) plus dispatch-body parity.
type buildCase struct {
	name       string
	splitRoot  bool
	folder     bool // folder-form {slug}/index.md vs flat {slug}.md
	worktree   bool // entity has a stamped worktree: value
	stage      string
	stdinExtra map[string]any // merged into the base stdin
}

func TestBuildParityCrossProduct(t *testing.T) {
	cases := []buildCase{
		{name: "split+folder+worktree+team", splitRoot: true, folder: true, worktree: true, stage: "implementation"},
		{name: "single+folder+worktree+team", splitRoot: false, folder: true, worktree: true, stage: "implementation"},
		{name: "split+flat+worktree+team", splitRoot: true, folder: false, worktree: true, stage: "implementation"},
		{name: "single+flat+worktree+team", splitRoot: false, folder: false, worktree: true, stage: "implementation"},
		{name: "single+flat+nonworktree+team", splitRoot: false, folder: false, worktree: false, stage: "backlog"},
		{name: "split+folder+nonworktree+team", splitRoot: true, folder: true, worktree: false, stage: "backlog"},
		{
			name: "single+flat+nonworktree+bare", splitRoot: false, folder: false, worktree: false, stage: "backlog",
			stdinExtra: map[string]any{"bare_mode": true},
		},
		{
			name: "feedback+scope+reflow", splitRoot: false, folder: false, worktree: true, stage: "validation",
			stdinExtra: map[string]any{
				"is_feedback_reflow": true,
				"feedback_context":   "REJECTED: do better.",
				"scope_notes":        "### Scope\nLimit to module X.",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()

			// workflow dir is the state checkout under split root, else the root.
			workflowDir := root
			if tc.splitRoot {
				workflowDir = filepath.Join(root, "state-checkout")
			}
			writeFile(t, filepath.Join(workflowDir, "README.md"), readmeWorktree(tc.splitRoot))

			worktreeRel := ""
			if tc.worktree {
				worktreeRel = ".worktrees/spacedock-ensign-thing"
				if err := os.MkdirAll(filepath.Join(root, worktreeRel), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			var entityPath string
			if tc.folder {
				entityPath = filepath.Join(workflowDir, "thing", "index.md")
			} else {
				entityPath = filepath.Join(workflowDir, "thing.md")
			}
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
			}, tc.stdinExtra)

			// Native and oracle write the dispatch body to the SAME deterministic
			// path, so capture the oracle's body before the native run overwrites it.
			oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", workflowDir)
			oracleBody := readDispatchBody(t, dispatchFilePathFromStdout(t, oracle.stdout))

			native := runNative(stdin, "build", "--workflow-dir", workflowDir)
			nativeBody := readDispatchBody(t, dispatchFilePathFromStdout(t, native.stdout))

			assertParity(t, tc.name, native, oracle)
			wantBody := stripStateCommitGuidance(rewriteOracleFetch(oracleBody))
			gotBody := stripStateCommitGuidance(nativeBody)
			if gotBody != wantBody {
				t.Errorf("%s: dispatch body mismatch\n--- native ---\n%s\n--- oracle(rewritten) ---\n%s",
					tc.name, gotBody, wantBody)
			}
		})
	}
}

// mergeStdin merges extra into base (extra wins) and returns the JSON string.
func mergeStdin(base, extra map[string]any) string {
	for k, v := range extra {
		base[k] = v
	}
	raw, _ := json.Marshal(base)
	return string(raw)
}

// dispatchFilePathFromStdout pulls dispatch_file_path out of a build stdout JSON.
func dispatchFilePathFromStdout(t *testing.T, stdout string) string {
	t.Helper()
	var out struct {
		DispatchFilePath string `json:"dispatch_file_path"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout is not build JSON: %v\n%s", err, stdout)
	}
	if out.DispatchFilePath == "" {
		t.Fatalf("no dispatch_file_path in stdout:\n%s", stdout)
	}
	return out.DispatchFilePath
}
