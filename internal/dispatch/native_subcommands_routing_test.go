// ABOUTME: routing regression — the four runtime-coupled subcommands now resolve
// ABOUTME: native (no longer exit-2 deferred) and build emits the standing fetch line.
package dispatch

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// TestRuntimeCoupledSubcommandsRouteNative asserts the four runtime-coupled
// subcommands are no longer rejected as deferred (the prerequisite seam returned
// exit 2 for them). Each now routes to a native handler: a recognized subcommand
// invoked with its required flag missing returns the native usage error (exit 2
// naming the subcommand), NOT the old "deferred to the claude-runtime-segregation
// surface" diagnostic. Re-adding a deferral would change this exit/message shape.
func TestRuntimeCoupledSubcommandsRouteNative(t *testing.T) {
	for _, sc := range []string{"context-budget", "list-standing", "show-standing", "spawn-standing"} {
		var out, errBuf bytes.Buffer
		code := Run(claudeteam.Probe, []string{sc}, strings.NewReader(""), &out, &errBuf)
		if code != 2 {
			t.Errorf("%s with no flags: exit=%d, want 2 (native usage error)", sc, code)
		}
		if strings.Contains(errBuf.String(), "deferred") {
			t.Errorf("%s still emits the deferral diagnostic: %q", sc, errBuf.String())
		}
		if !strings.Contains(errBuf.String(), sc) {
			t.Errorf("%s usage error does not name the subcommand: %q", sc, errBuf.String())
		}
	}
}

// TestBuildEmitsStandingFetchLineUnderMods asserts the build _mods/show-standing
// branch is now native: with a _mods dir declaring a standing teammate and a team
// name, build emits BOTH the show-stage-def fetch line AND a
// `spacedock dispatch show-standing` fetch line. The prerequisite deferred this
// branch (emitting only show-stage-def); landing it here trips this assertion if
// the branch is later removed.
func TestBuildEmitsStandingFetchLineUnderMods(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
	if err := os.MkdirAll(filepath.Join(root, "_mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "_mods", "helper.md"),
		"---\nstanding: true\nname: helper\n---\n## Hook: startup\n- name: helper\n## Agent Prompt\ny\n")
	entityPath := filepath.Join(root, "thing.md")
	writeFile(t, entityPath, entityFM("Thing", "backlog", ""))
	gitInit(t, root)

	stdin := mergeStdin(map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "backlog",
		"checklist":      []string{"- a", "- b"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	}, nil)

	var out, errBuf bytes.Buffer
	if code := Run(claudeteam.Probe, []string{"build", "--workflow-dir", root}, strings.NewReader(stdin), &out, &errBuf); code != 0 {
		t.Fatalf("build exit=%d stderr=%q", code, errBuf.String())
	}

	var env struct {
		FetchCommands []string `json:"fetch_commands"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("build output not JSON: %v\n%s", err, out.String())
	}
	if len(env.FetchCommands) != 2 {
		t.Fatalf("expected two fetch lines (show-stage-def + show-standing); got %v", env.FetchCommands)
	}
	if !strings.Contains(env.FetchCommands[0], "show-stage-def") {
		t.Errorf("first fetch line is not show-stage-def: %q", env.FetchCommands[0])
	}
	if !strings.Contains(env.FetchCommands[1], "spacedock dispatch show-standing") {
		t.Errorf("second fetch line is not the native show-standing line: %q", env.FetchCommands[1])
	}
}

// TestBuildOmitsStandingFetchLineWithoutMods asserts the standing fetch line is
// omitted when no standing mod exists — the branch is conditional, not always-on.
func TestBuildOmitsStandingFetchLineWithoutMods(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
	entityPath := filepath.Join(root, "thing.md")
	writeFile(t, entityPath, entityFM("Thing", "backlog", ""))
	gitInit(t, root)

	stdin := mergeStdin(map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "backlog",
		"checklist":      []string{"- a", "- b"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	}, nil)

	var out, errBuf bytes.Buffer
	if code := Run(claudeteam.Probe, []string{"build", "--workflow-dir", root}, strings.NewReader(stdin), &out, &errBuf); code != 0 {
		t.Fatalf("build exit=%d stderr=%q", code, errBuf.String())
	}

	var env struct {
		FetchCommands []string `json:"fetch_commands"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("build output not JSON: %v\n%s", err, out.String())
	}
	if len(env.FetchCommands) != 1 || !strings.Contains(env.FetchCommands[0], "show-stage-def") {
		t.Fatalf("expected exactly one show-stage-def fetch line; got %v", env.FetchCommands)
	}
}
