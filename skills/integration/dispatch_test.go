// ABOUTME: Dispatch-derivation fixture tests over the native in-process dispatch.Run build
// ABOUTME: under split-root + folder-form + worktree — AC-3 (state-path handoff) and AC-4 (slug-not-stem).
package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/dispatch"
)

// buildResult is the subset of `dispatch build` JSON output the dispatch
// derivation tests assert on, plus the dispatch body read from disk.
type buildResult struct {
	Name             string `json:"name"`
	DispatchFilePath string `json:"dispatch_file_path"`
	body             string
}

// runBuild drives the native in-process `dispatch.Run build` with the given
// input JSON, returning the parsed output and the dispatch body written to
// dispatch_file_path. The build derives the worktree/state paths from the
// git root of the workflow dir (FindGitRoot), so the git-initialized fixture
// root is reachable through the absolute entity_path/workflow_dir in the input,
// not the process working directory. It fails the test on a non-zero exit so
// callers can assert on a known-good build.
func runBuild(t *testing.T, root, workflowDir string, input map[string]any) buildResult {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	// A unique HOME keeps the build's bare-mode team-evidence probe from reading
	// the developer's real ~/.claude/teams; the build path does not depend on it
	// in team mode (team_name supplied), but pin it for hermeticity.
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	if exit := dispatch.Run([]string{"build", "--workflow-dir", workflowDir}, strings.NewReader(string(raw)), &stdout, &stderr); exit != 0 {
		t.Fatalf("dispatch build exited %d\nstdout: %s\nstderr: %s", exit, stdout.String(), stderr.String())
	}
	var res buildResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("build output is not JSON: %v\n%s", err, stdout.String())
	}
	if res.DispatchFilePath == "" {
		t.Fatalf("build output has no dispatch_file_path\n%s", stdout.String())
	}
	bodyBytes, err := os.ReadFile(res.DispatchFilePath)
	if err != nil {
		t.Fatalf("read dispatch body %s: %v", res.DispatchFilePath, err)
	}
	res.body = string(bodyBytes)
	return res
}

// splitRootReadme is a workflow README with a `state:` field (marking the
// workflow split-root) and a stages block whose implementation stage is a
// worktree stage. The helper parses both the state field and the stages.
const splitRootReadme = `---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: slug
state: state-checkout
stages:
  defaults:
    worktree: false
    concurrency: 1
  states:
    - name: backlog
      initial: true
    - name: implementation
      worktree: true
    - name: done
      terminal: true
---

# Split-Root Fixture Workflow

### backlog

A task starts in backlog.

- **Outputs:** the seed.

### implementation

Do the work in a worktree.

- **Outputs:** the deliverable.

### done

Terminal.
`

// flatRootReadme mirrors splitRootReadme but without a `state:` field, so the
// workflow is non-split-root and flat-entity slug derivation can be exercised.
const flatRootReadme = `---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: slug
stages:
  defaults:
    worktree: false
    concurrency: 1
  states:
    - name: backlog
      initial: true
    - name: implementation
      worktree: true
    - name: done
      terminal: true
---

# Flat Fixture Workflow

### backlog

A task starts in backlog.

- **Outputs:** the seed.

### implementation

Do the work in a worktree.

- **Outputs:** the deliverable.

### done

Terminal.
`

// gitInitFixture initializes a git repo at dir so the helper's find_git_root
// resolves to it.
func gitInitFixture(t *testing.T, dir string) {
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

// writeFile writes content to path, creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSplitRootFolderWorktreeDispatch is the riskiest-mechanism test: a
// split-root, folder-form entity dispatched into a worktree stage. It locks
// AC-3 (state-checkout entity-path handoff, no .worktrees/ segment in the
// entity-read line or completion signal, worktree CODE instructions still
// emitted) and AC-4 (worker name and dispatch file use the folder slug, not
// `index`).
func TestSplitRootFolderWorktreeDispatch(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "state-checkout")
	worktreeRel := ".worktrees/spacedock-ensign-skill-launcher"
	worktreePath := filepath.Join(root, worktreeRel)

	writeFile(t, filepath.Join(stateDir, "README.md"), splitRootReadme)
	entityPath := filepath.Join(stateDir, "skill-launcher", "index.md")
	writeFile(t, entityPath, `---
id: "001"
title: Skill launcher integration
status: implementation
worktree: `+worktreeRel+`
---
# Skill launcher integration

Body.
`)
	// The worktree directory must exist on disk; the helper validates it.
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, root)

	res := runBuild(t, root, stateDir, map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   stateDir,
		"stage":          "implementation",
		"checklist":      []string{"- do the work"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	})

	// AC-4: name and dispatch file use the folder slug, never `index`.
	wantName := "spacedock-ensign-skill-launcher-implementation"
	if res.Name != wantName {
		t.Errorf("name = %q, want %q (folder slug, not index)", res.Name, wantName)
	}
	if !strings.HasSuffix(res.DispatchFilePath, wantName+".md") {
		t.Errorf("dispatch_file_path = %q, want suffix %q.md", res.DispatchFilePath, wantName)
	}
	if strings.Contains(res.Name, "index") || strings.Contains(res.DispatchFilePath, "/index") {
		t.Errorf("name/dispatch path leaked the index stem: name=%q path=%q", res.Name, res.DispatchFilePath)
	}

	// AC-3: the branch line uses the folder slug (AC-4) and the worktree CODE
	// working-dir/branch instructions are still emitted (AC-3).
	wantBranch := "spacedock-ensign/skill-launcher"
	if !strings.Contains(res.body, wantBranch) {
		t.Errorf("dispatch body missing folder-slug branch %q\n%s", wantBranch, res.body)
	}
	if !strings.Contains(res.body, worktreePath) {
		t.Errorf("dispatch body missing worktree working-dir %q (CODE isolation must still be emitted)", worktreePath)
	}

	// AC-3: the entity-read line and the completion-signal ref point at the
	// state-checkout entity path and contain no .worktrees/ segment.
	entityReadLine := lineContaining(res.body, "Read the entity file at")
	if entityReadLine == "" {
		t.Fatalf("no entity-read line in dispatch body\n%s", res.body)
	}
	if !strings.Contains(entityReadLine, entityPath) {
		t.Errorf("entity-read line does not point at state-checkout path\n got: %s\nwant contains: %s", entityReadLine, entityPath)
	}
	if strings.Contains(entityReadLine, ".worktrees/") {
		t.Errorf("entity-read line contains a .worktrees/ segment (must be the state path)\n%s", entityReadLine)
	}

	completionLine := lineContaining(res.body, "Report written to")
	if completionLine == "" {
		t.Fatalf("no completion-signal line in dispatch body\n%s", res.body)
	}
	if !strings.Contains(completionLine, entityPath) {
		t.Errorf("completion signal does not reference state-checkout path\n got: %s\nwant contains: %s", completionLine, entityPath)
	}
	if strings.Contains(completionLine, ".worktrees/") {
		t.Errorf("completion signal contains a .worktrees/ segment (must be the state path)\n%s", completionLine)
	}
}

// TestFlatEntitySlugUnchanged locks the flat-entity case (AC-4): a flat
// `{slug}.md` entity continues to derive its slug from the filename stem.
func TestFlatEntitySlugUnchanged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), flatRootReadme)
	entityPath := filepath.Join(root, "vendor-script.md")
	writeFile(t, entityPath, `---
id: "002"
title: Vendor the script
status: implementation
worktree: ""
---
# Vendor the script

Body.
`)
	gitInitFixture(t, root)

	res := runBuild(t, root, root, map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "backlog",
		"checklist":      []string{"- do the work"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	})

	wantName := "spacedock-ensign-vendor-script-backlog"
	if res.Name != wantName {
		t.Errorf("flat-entity name = %q, want %q (stem slug)", res.Name, wantName)
	}
}

// lineContaining returns the first line of body that contains substr, or "".
func lineContaining(body, substr string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}
