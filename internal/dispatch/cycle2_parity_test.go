// ABOUTME: Cycle-2 parity fixtures — abs-worktree join, str.splitlines separator
// ABOUTME: set, numeric schema_version, and non-object stdin, each vs the oracle.
package dispatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildAbsoluteWorktreeParity locks fix 1: an ABSOLUTE worktree: frontmatter
// value must resolve via os.path.join semantics (absolute component wins), not
// filepath.Join (which doubles the path under gitRoot). The FO stamps absolute
// worktree: values on live entities, so this is the happy path. Before the fix
// native rejected with a doubled-path "does not exist"; the oracle accepts.
func TestBuildAbsoluteWorktreeParity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))

	// An ABSOLUTE worktree path (the shape the FO stamps), existing on disk.
	absWorktree := filepath.Join(root, ".worktrees", "spacedock-ensign-thing")
	if err := os.MkdirAll(absWorktree, 0o755); err != nil {
		t.Fatal(err)
	}
	entityPath := filepath.Join(root, "thing.md")
	writeFile(t, entityPath, entityFM("Thing", "implementation", absWorktree))
	gitInit(t, root)

	stdin := mergeStdin(map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "implementation",
		"checklist":      []string{"- a"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	}, nil)

	oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", root)
	oracleBody := readDispatchBody(t, dispatchFilePathFromStdout(t, oracle.stdout))
	native := runNative(stdin, "build", "--workflow-dir", root)
	nativeBody := readDispatchBody(t, dispatchFilePathFromStdout(t, native.stdout))

	assertParity(t, "abs-worktree", native, oracle)
	if nativeBody != rewriteOracleFetch(oracleBody) {
		t.Errorf("abs-worktree body mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nativeBody, rewriteOracleFetch(oracleBody))
	}
	// Explicit: native must exit 0 (full dispatch), not the doubled-path error.
	if native.exit != 0 {
		t.Errorf("abs-worktree native exit=%d, want 0; stderr=%q", native.exit, native.stderr)
	}
	if strings.Contains(native.stderr, "does not exist") {
		t.Errorf("abs-worktree native still doubles the path: %q", native.stderr)
	}
}

// TestShowStageDefSeparatorParity locks fix 2: show-stage-def's line splitting
// matches Python str.splitlines() across the full separator set, not just
// \r\n/\r/\n. Each case embeds one separator inside a ### subsection body; the
// oracle breaks the line there and the native must produce byte-identical
// stdout.
func TestShowStageDefSeparatorParity(t *testing.T) {
	// name -> the separator rune to embed between two body words.
	separators := []struct {
		name string
		sep  string
	}{
		{"VT", "\v"},
		{"FF", "\f"},
		{"FS", "\x1c"},
		{"GS", "\x1d"},
		{"RS", "\x1e"},
		{"NEL", "\u0085"},
		{"LS", "\u2028"},
		{"PS", "\u2029"},
	}
	for _, sc := range separators {
		t.Run(sc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			// A README whose ideation body has two words split by the separator.
			readme := "---\nentity-type: task\nid-style: slug\n---\n# Sep\n\n### ideation\n\nalpha" + sc.sep + "beta gamma.\n\n### done\n\nterm.\n"
			writeFile(t, filepath.Join(root, "README.md"), readme)

			oracle := runOracle(t, root, home, "", "show-stage-def", "--workflow-dir", root, "--stage", "ideation")
			native := runNative("", "show-stage-def", "--workflow-dir", root, "--stage", "ideation")

			if native.stdout != oracle.stdout {
				t.Errorf("%s: stdout mismatch\n--- native ---\n%q\n--- oracle ---\n%q", sc.name, native.stdout, oracle.stdout)
			}
			if native.stderr != oracle.stderr || native.exit != oracle.exit {
				t.Errorf("%s: stderr/exit mismatch native=(%q,%d) oracle=(%q,%d)", sc.name, native.stderr, native.exit, oracle.stderr, oracle.exit)
			}
		})
	}
}

// TestBuildSchemaVersionNumericParity locks fix 3: schema_version is compared
// numerically. 2 and 2.0 accept; "2" (string), true, and a wrong number reject —
// matching the oracle's `sv != 2` in both directions.
func TestBuildSchemaVersionNumericParity(t *testing.T) {
	cases := []struct {
		name string
		sv   string // raw JSON literal for schema_version
	}{
		{"int-2", "2"},
		{"float-2.0", "2.0"},
		{"string-2", `"2"`},
		{"int-1", "1"},
		{"float-2.5", "2.5"},
		{"bool-true", "true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
			entityPath := filepath.Join(root, "thing.md")
			writeFile(t, entityPath, entityFM("Thing", "backlog", ""))
			gitInit(t, root)

			stdin := fmt.Sprintf(
				`{"schema_version":%s,"entity_path":%q,"workflow_dir":%q,"stage":"backlog","checklist":["- a"],"team_name":"t"}`,
				tc.sv, entityPath, root)

			oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", root)
			native := runNative(stdin, "build", "--workflow-dir", root)
			// Accepted cases write a dispatch body; capture order does not matter
			// here since we only byte-compare the channels.
			assertParity(t, tc.name, native, oracle)
		})
	}
}

// TestBuildNonObjectStdinParity locks fix 4: a valid non-object top-level (null,
// array, scalar) yields "stdin must be a JSON object" with exit 1, matching the
// oracle's isinstance(inp, dict) check after json.loads — not a downstream
// missing-field error.
func TestBuildNonObjectStdinParity(t *testing.T) {
	cases := []struct {
		name  string
		stdin string
	}{
		{"null", "null"},
		{"array", "[1,2,3]"},
		{"number", "42"},
		{"string", `"hello"`},
		{"bool", "true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
			gitInit(t, root)

			oracle := runOracle(t, root, home, tc.stdin, "build", "--workflow-dir", root)
			native := runNative(tc.stdin, "build", "--workflow-dir", root)

			if native.stderr != oracle.stderr {
				t.Errorf("%s: stderr mismatch\nnative=%q\noracle=%q", tc.name, native.stderr, oracle.stderr)
			}
			if native.exit != oracle.exit {
				t.Errorf("%s: exit mismatch native=%d oracle=%d", tc.name, native.exit, oracle.exit)
			}
		})
	}
}
