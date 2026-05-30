// ABOUTME: Negative parity — the 18 byte-identical error fixtures (exit + stderr)
// ABOUTME: and the 2 str(e) prefix-only paths (exit + stderr prefix), AC-1b.
package dispatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errFixture sets up a fixture tree and returns (workflowDir, stdin) for an
// error-path parity run. Each closure crafts exactly the condition the error
// guards, leaving the rest of the request well-formed so the guard under test
// is the one that fires.
type errFixture func(t *testing.T, root string) (workflowDir, stdin string)

// goodReadme is a minimal well-formed README with a non-worktree backlog stage
// and a worktree implementation stage, reused by most error fixtures.
const goodReadme = `---
entity-type: task
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
    - name: validation
      worktree: true
      feedback-to: implementation
    - name: done
      terminal: true
---
# Good

### backlog

seed.

### implementation

work.

### validation

verify.

### done

term.
`

// TestBuildByteIdenticalErrors asserts the 18 deterministic error byte-strings
// are byte-identical between native and oracle (stderr) and exit codes match.
func TestBuildByteIdenticalErrors(t *testing.T) {
	cases := []struct {
		name string
		fx   errFixture
	}{
		{"stdin-not-object", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			return wd, `[1,2,3]`
		}},
		{"missing-required-field", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			// schema_version present, entity_path omitted.
			return wd, `{"schema_version":2,"workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`
		}},
		{"present-but-null-field", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			ep := writeFlatEntity(t, wd, "backlog", "")
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":null,"checklist":["- a"],"team_name":"t"}`
		}},
		{"unsupported-schema-version", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			ep := writeFlatEntity(t, wd, "backlog", "")
			return wd, `{"schema_version":1,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`
		}},
		{"worktree-absolute-entity-path", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			bad := filepath.Join(wd, ".worktrees", "x", "thing.md")
			return wd, `{"schema_version":2,"entity_path":"` + bad + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`
		}},
		{"checklist-empty", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			ep := writeFlatEntity(t, wd, "backlog", "")
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":[],"team_name":"t"}`
		}},
		{"checklist-not-a-list", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			ep := writeFlatEntity(t, wd, "backlog", "")
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":"not a list","team_name":"t"}`
		}},
		{"entity-not-readable", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			missing := filepath.Join(wd, "nope.md")
			return wd, `{"schema_version":2,"entity_path":"` + missing + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`
		}},
		{"readme-not-found", func(t *testing.T, root string) (string, string) {
			// workflow_dir has no README; entity lives in root (readable).
			wd := filepath.Join(root, "noreadme")
			if err := os.MkdirAll(wd, 0o755); err != nil {
				t.Fatal(err)
			}
			ep := filepath.Join(root, "thing.md")
			writeFile(t, ep, entityFM("Thing", "backlog", ""))
			gitInit(t, root)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`
		}},
		{"no-stages-block", func(t *testing.T, root string) (string, string) {
			wd := root
			writeFile(t, filepath.Join(wd, "README.md"), "---\nentity-type: task\nid-style: slug\n---\n# No stages\n")
			ep := filepath.Join(wd, "thing.md")
			writeFile(t, ep, entityFM("Thing", "backlog", ""))
			gitInit(t, root)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`
		}},
		{"stage-not-in-workflow", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			ep := writeFlatEntity(t, wd, "nonesuch", "")
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"nonesuch","checklist":["- a"],"team_name":"t"}`
		}},
		{"invalid-stage-model", func(t *testing.T, root string) (string, string) {
			wd := root
			writeFile(t, filepath.Join(wd, "README.md"), readmeBadModel("badmodel", "frobnicate", ""))
			ep := filepath.Join(wd, "thing.md")
			writeFile(t, ep, entityFM("Thing", "badmodel", ""))
			gitInit(t, root)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"badmodel","checklist":["- a"],"team_name":"t"}`
		}},
		{"invalid-defaults-model", func(t *testing.T, root string) (string, string) {
			wd := root
			writeFile(t, filepath.Join(wd, "README.md"), readmeBadModel("ok", "", "frobnicate"))
			ep := filepath.Join(wd, "thing.md")
			writeFile(t, ep, entityFM("Thing", "ok", ""))
			gitInit(t, root)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"ok","checklist":["- a"],"team_name":"t"}`
		}},
		{"worktree-path-missing", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			// Entity stamps a worktree path that does not exist on disk.
			ep := writeFlatEntity(t, wd, "implementation", ".worktrees/does-not-exist")
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"implementation","checklist":["- a"],"team_name":"t"}`
		}},
		{"worktree-stage-no-worktree", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			// implementation is a worktree stage but the entity has no worktree.
			ep := writeFlatEntity(t, wd, "implementation", "")
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"implementation","checklist":["- a"],"team_name":"t"}`
		}},
		{"feedback-context-missing", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			wtRel := ".worktrees/spacedock-ensign-thing"
			if err := os.MkdirAll(filepath.Join(root, wtRel), 0o755); err != nil {
				t.Fatal(err)
			}
			ep := writeFlatEntity(t, wd, "validation", wtRel)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"validation","checklist":["- a"],"team_name":"t","is_feedback_reflow":true}`
		}},
		{"team-mode-requires-team-name", func(t *testing.T, root string) (string, string) {
			wd := writeGood(t, root)
			ep := writeFlatEntity(t, wd, "backlog", "")
			// bare_mode false (default) and no team_name.
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"]}`
		}},
		{"invalid-stage-name", func(t *testing.T, root string) (string, string) {
			// A stage name with an uppercase letter fails the kebab-case regex
			// at name-derivation time.
			wd := root
			writeFile(t, filepath.Join(wd, "README.md"), readmeStageName("BadStage"))
			ep := filepath.Join(wd, "thing.md")
			writeFile(t, ep, entityFM("Thing", "BadStage", ""))
			gitInit(t, root)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"BadStage","checklist":["- a"],"team_name":"t"}`
		}},
		{"malformed-heading", func(t *testing.T, root string) (string, string) {
			wd := root
			// The stage name appears as a non-first token in a ### heading.
			writeFile(t, filepath.Join(wd, "README.md"), readmeMalformedHeading())
			ep := filepath.Join(wd, "thing.md")
			writeFile(t, ep, entityFM("Thing", "ideation", ""))
			gitInit(t, root)
			return wd, `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"ideation","checklist":["- a"],"team_name":"t"}`
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			wd, stdin := tc.fx(t, root)

			oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", wd)
			native := runNative(stdin, "build", "--workflow-dir", wd)

			if native.stderr != oracle.stderr {
				t.Errorf("%s: stderr mismatch\n--- native ---\n%q\n--- oracle ---\n%q", tc.name, native.stderr, oracle.stderr)
			}
			if native.exit != oracle.exit {
				t.Errorf("%s: exit mismatch native=%d oracle=%d (native stderr=%q)", tc.name, native.exit, oracle.exit, native.stderr)
			}
			if native.stdout != oracle.stdout {
				t.Errorf("%s: stdout mismatch native=%q oracle=%q", tc.name, native.stdout, oracle.stdout)
			}
		})
	}
}

// TestBuildStrEErrors covers the two str(e)-bearing paths (AC-1b): they match on
// exit code and a byte-identical stderr PREFIX, not the interpreter-version tail.
func TestBuildStrEErrors(t *testing.T) {
	t.Run("invalid-json-on-stdin", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		root := t.TempDir()
		wd := writeGood(t, root)
		stdin := `{"a": ` // truncated -> JSON syntax error

		oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", wd)
		native := runNative(stdin, "build", "--workflow-dir", wd)

		const prefix = "error: invalid JSON on stdin: "
		if native.exit != 1 || oracle.exit != 1 {
			t.Errorf("exit native=%d oracle=%d, want 1/1", native.exit, oracle.exit)
		}
		if !strings.HasPrefix(native.stderr, prefix) {
			t.Errorf("native stderr lacks prefix %q:\n%q", prefix, native.stderr)
		}
		if !strings.HasPrefix(oracle.stderr, prefix) {
			t.Errorf("oracle stderr lacks prefix %q:\n%q", prefix, oracle.stderr)
		}
	})

	t.Run("dispatch-file-write-failed", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		root := t.TempDir()
		wd := writeGood(t, root)
		ep := writeFlatEntity(t, wd, "backlog", "")
		stdin := `{"schema_version":2,"entity_path":"` + ep + `","workflow_dir":"` + wd + `","stage":"backlog","checklist":["- a"],"team_name":"t"}`

		// Force the write to fail: make the deterministic dispatch-file path a
		// directory so open()/WriteFile cannot write a regular file there. Clear
		// any leftover regular file other fixtures wrote at this same path first.
		derived := "spacedock-ensign-thing-backlog"
		target := filepath.Join(dispatchFileDir, derived+".md")
		if err := os.RemoveAll(target); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(target)

		oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", wd)
		native := runNative(stdin, "build", "--workflow-dir", wd)

		prefix := "dispatch_file_write_failed: " + target + ": "
		if native.exit != 1 || oracle.exit != 1 {
			t.Errorf("exit native=%d oracle=%d, want 1/1", native.exit, oracle.exit)
		}
		if !strings.HasPrefix(native.stderr, prefix) {
			t.Errorf("native stderr lacks prefix %q:\n%q", prefix, native.stderr)
		}
		if !strings.HasPrefix(oracle.stderr, prefix) {
			t.Errorf("oracle stderr lacks prefix %q:\n%q", prefix, oracle.stderr)
		}
	})
}

// writeGood writes the good README at root and git-inits root, returning the
// workflow dir (root). Fixtures that need an entity add it after.
func writeGood(t *testing.T, root string) string {
	t.Helper()
	writeFile(t, filepath.Join(root, "README.md"), goodReadme)
	gitInit(t, root)
	return root
}

// writeFlatEntity writes a flat thing.md with the given status + worktree and
// returns its path. (root is already git-init'd by writeGood.)
func writeFlatEntity(t *testing.T, wd, status, worktree string) string {
	t.Helper()
	ep := filepath.Join(wd, "thing.md")
	writeFile(t, ep, entityFM("Thing", status, worktree))
	return ep
}

// readmeBadModel builds a README whose single stage carries stageModel (when
// non-empty) and whose defaults carry defaultsModel (when non-empty), for the
// model-enum error fixtures.
func readmeBadModel(stage, stageModel, defaultsModel string) string {
	defModel := ""
	if defaultsModel != "" {
		defModel = "    model: " + defaultsModel + "\n"
	}
	stModel := ""
	if stageModel != "" {
		stModel = "      model: " + stageModel + "\n"
	}
	return "---\nentity-type: task\nid-style: slug\nstages:\n  defaults:\n    worktree: false\n    concurrency: 1\n" +
		defModel +
		"  states:\n    - name: " + stage + "\n      initial: true\n" + stModel +
		"    - name: done\n      terminal: true\n---\n# Bad Model\n\n### " + stage + "\n\nx.\n\n### done\n\nterm.\n"
}

// readmeStageName builds a README whose initial stage has the given (possibly
// regex-violating) name, for the invalid-stage-name fixture.
func readmeStageName(stage string) string {
	return "---\nentity-type: task\nid-style: slug\nstages:\n  defaults:\n    worktree: false\n    concurrency: 1\n  states:\n    - name: " +
		stage + "\n      initial: true\n    - name: done\n      terminal: true\n---\n# StageName\n\n### " + stage + "\n\nx.\n\n### done\n\nterm.\n"
}

// readmeMalformedHeading builds a README with an ideation stage in the stages
// block but whose only ### heading mentioning "ideation" puts it as a non-first
// token, so extract_stage_subsection raises the malformed-heading diagnostic.
func readmeMalformedHeading() string {
	return "---\nentity-type: task\nid-style: slug\nstages:\n  defaults:\n    worktree: false\n    concurrency: 1\n  states:\n    - name: ideation\n      initial: true\n    - name: done\n      terminal: true\n---\n# Malformed\n\n### the ideation stage\n\nx.\n\n### done\n\nterm.\n"
}
