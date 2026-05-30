// ABOUTME: Build byte-hazard parity — model precedence stderr/null literal,
// ABOUTME: no-HTML-escape, and shlex space-quoting locked against the oracle.
package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readmeModels declares model at three precedence sites so a fixture can pick a
// stage to toggle which one wins: stage-set (opus), defaults-set only (haiku),
// or neither (null). The states are non-worktree backlog stages.
const readmeModels = `---
entity-type: task
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 1
    model: haiku
  states:
    - name: stagemodel
      initial: true
      model: opus
    - name: defaultsmodel
    - name: done
      terminal: true
---
# Models Fixture

### stagemodel

stage wins.

- **Outputs:** x.

### defaultsmodel

defaults win.

- **Outputs:** y.

### done

term.
`

// readmeModelsNull mirrors readmeModels but with no model anywhere, so the
// effective model resolves to null (empty stderr, "model": null in stdout).
const readmeModelsNull = `---
entity-type: task
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 1
  states:
    - name: nomodel
      initial: true
    - name: done
      terminal: true
---
# No Models Fixture

### nomodel

no model anywhere.

- **Outputs:** x.

### done

term.
`

// TestBuildModelPrecedence locks the effective_model precedence (stage >
// defaults > null): the [build] effective_model stderr line (with the U+2192
// arrow) and the "model" JSON value across all three sources.
func TestBuildModelPrecedence(t *testing.T) {
	type modelCase struct {
		name      string
		readme    string
		stage     string
		wantModel string // expected JSON model value, "" => null
	}
	cases := []modelCase{
		{name: "stage-wins-opus", readme: readmeModels, stage: "stagemodel", wantModel: "opus"},
		{name: "defaults-haiku", readme: readmeModels, stage: "defaultsmodel", wantModel: "haiku"},
		{name: "null", readme: readmeModelsNull, stage: "nomodel", wantModel: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "README.md"), tc.readme)
			entityPath := filepath.Join(root, "thing.md")
			writeFile(t, entityPath, entityFM("Thing", tc.stage, ""))
			gitInit(t, root)

			stdin := mergeStdin(map[string]any{
				"schema_version": 2,
				"entity_path":    entityPath,
				"workflow_dir":   root,
				"stage":          tc.stage,
				"checklist":      []string{"- a"},
				"team_name":      "fixture-team",
				"bare_mode":      false,
			}, nil)

			oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", root)
			native := runNative(stdin, "build", "--workflow-dir", root)
			assertParity(t, tc.name, native, oracle)

			// Lock the "model" JSON value explicitly (parity already covers it, but
			// this names the contract: null literal vs string).
			var out map[string]json.RawMessage
			if err := json.Unmarshal([]byte(native.stdout), &out); err != nil {
				t.Fatalf("native stdout not JSON: %v", err)
			}
			gotModel := string(out["model"])
			wantModel := "null"
			if tc.wantModel != "" {
				wantModel = `"` + tc.wantModel + `"`
			}
			if gotModel != wantModel {
				t.Errorf("model = %s, want %s", gotModel, wantModel)
			}
		})
	}
}

// TestBuildNoHTMLEscape locks that <, >, & survive verbatim in the emitted JSON
// (Go's default json.Marshal escapes them to < / > / &). The
// entity title carries all three, so they flow into the stdout JSON description
// field — the only build-output JSON field that can carry caller prose. The
// feedback_context field carries them too, exercising the dispatch body (plain
// markdown, no escaping). Native and oracle stdout + body must be byte-identical.
func TestBuildNoHTMLEscape(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
	entityPath := filepath.Join(root, "thing.md")
	htmlTitle := "Compare a < b && c > d in <Tag>"
	writeFile(t, entityPath, entityFM(htmlTitle, "validation", ".worktrees/spacedock-ensign-thing"))
	if err := os.MkdirAll(filepath.Join(root, ".worktrees/spacedock-ensign-thing"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitInit(t, root)

	stdin := mergeStdin(map[string]any{
		"schema_version":     2,
		"entity_path":        entityPath,
		"workflow_dir":       root,
		"stage":              "validation",
		"checklist":          []string{"- a < b", "- c > d & e"},
		"team_name":          "fixture-team",
		"bare_mode":          false,
		"is_feedback_reflow": true,
		"feedback_context":   "compare a < b && c > d in <Tag> & raw &amp; ampersand",
	}, nil)

	oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", root)
	oracleBody := readDispatchBody(t, dispatchFilePathFromStdout(t, oracle.stdout))
	native := runNative(stdin, "build", "--workflow-dir", root)
	nativeBody := readDispatchBody(t, dispatchFilePathFromStdout(t, native.stdout))

	assertParity(t, "html-escape", native, oracle)
	if nativeBody != rewriteOracleFetch(oracleBody) {
		t.Errorf("html-escape body mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nativeBody, rewriteOracleFetch(oracleBody))
	}

	// Explicit: the raw <, >, & bytes survive in the native stdout description,
	// and no \uXXXX-escaped sequences appear (Go's default HTML escaping is off).
	if !strings.Contains(native.stdout, htmlTitle) {
		t.Errorf("native stdout HTML-escaped the description title away:\n%s", native.stdout)
	}
	for _, ch := range []rune{'<', '>', '&'} {
		esc := fmt.Sprintf("\\u%04x", ch) // the \uXXXX sequence Go emits when escaping is ON
		if strings.Contains(native.stdout, esc) {
			t.Errorf("native stdout contains %s (SetEscapeHTML not off):\n%s", esc, native.stdout)
		}
	}
}

// TestBuildSpaceBearingPath locks the shlex quoting rule: a workflow dir with a
// space gets single-quoted in the emitted fetch line. Native and oracle (with
// the fetch prefix rewritten) must match byte-for-byte.
func TestBuildSpaceBearingPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	// A workflow dir whose name contains a space.
	workflowDir := filepath.Join(root, "work dir")
	writeFile(t, filepath.Join(workflowDir, "README.md"), readmeWorktree(false))
	entityPath := filepath.Join(workflowDir, "thing.md")
	writeFile(t, entityPath, entityFM("Thing", "backlog", ""))
	gitInit(t, root)

	stdin := mergeStdin(map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   workflowDir,
		"stage":          "backlog",
		"checklist":      []string{"- a"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	}, nil)

	oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", workflowDir)
	oracleBody := readDispatchBody(t, dispatchFilePathFromStdout(t, oracle.stdout))
	native := runNative(stdin, "build", "--workflow-dir", workflowDir)
	nativeBody := readDispatchBody(t, dispatchFilePathFromStdout(t, native.stdout))

	assertParity(t, "space-path", native, oracle)
	if nativeBody != rewriteOracleFetch(oracleBody) {
		t.Errorf("space-path body mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nativeBody, rewriteOracleFetch(oracleBody))
	}
	// Explicit: the space-bearing dir is single-quoted in the native fetch line.
	wantQuoted := "spacedock dispatch show-stage-def --workflow-dir '" + workflowDir + "' --stage backlog"
	if !strings.Contains(native.stdout, wantQuoted) {
		t.Errorf("native fetch line not shlex-quoted for space path:\nwant contains: %s\ngot:\n%s", wantQuoted, native.stdout)
	}
}
