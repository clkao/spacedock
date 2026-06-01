// ABOUTME: AC-P3 regression — this entity adds no native subcommand handlers: the
// ABOUTME: four standing commands stay deferred and build emits no standing fetch line.
package dispatch

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// TestDeferredSubcommandSetUnchanged pins the deferred set to exactly the four
// runtime-coupled standing subcommands. A native handler added here would have to
// remove its name from this set (so Run stops routing it to the deferral
// diagnostic), tripping this assertion — making "no new native handler" observable
// at the routing table, complementing the behavioral TestDeferredSubcommandGuard.
func TestDeferredSubcommandSetUnchanged(t *testing.T) {
	var got []string
	for name := range deferredSubcommands {
		got = append(got, name)
	}
	sort.Strings(got)
	want := []string{"context-budget", "list-standing", "show-standing", "spawn-standing"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("deferred subcommand set changed: got %v, want %v", got, want)
	}
}

// TestBuildEmitsNoStandingFetchLine asserts the build `_mods`/standing branch is
// still deferred to claude-runtime-segregation: even with a `_mods/` dir present,
// the native build emits ONLY the show-stage-def fetch line — never a
// `show-standing` / standing-teammate fetch line. Adding the native `_mods` branch
// here would emit that line and trip this test.
func TestBuildEmitsNoStandingFetchLine(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
	// A _mods dir declaring a standing teammate — the input the deferred branch
	// would consume. The native build must ignore it (the branch is not ported).
	if err := os.MkdirAll(filepath.Join(root, "_mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "_mods", "helper.md"),
		"---\nstanding: true\nname: helper\n---\n## Hook: startup\nx\n## Agent Prompt\ny\n")
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
		Prompt        string   `json:"prompt"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("build output not JSON: %v\n%s", err, out.String())
	}
	if len(env.FetchCommands) != 1 || !strings.Contains(env.FetchCommands[0], "show-stage-def") {
		t.Fatalf("expected exactly one show-stage-def fetch line; got %v", env.FetchCommands)
	}
	for _, fc := range env.FetchCommands {
		if strings.Contains(fc, "standing") {
			t.Fatalf("native build emitted a standing fetch line (the _mods branch must stay deferred): %q", fc)
		}
	}
}
