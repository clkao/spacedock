// ABOUTME: show-stage-def parity vs the oracle across well-formed, decorated,
// ABOUTME: malformed, and missing headings; plus the deferred-subcommand guard.
package dispatch

import (
	"os"
	"path/filepath"
	"testing"
)

// readmeHeadings is a README whose ### subsections exercise the extractor: a
// plain heading, a decorated heading (backticks + parenthetical), and a heading
// whose stage name is NOT the first token (malformed). The frontmatter stages
// block is irrelevant to show-stage-def (it reads the body sections directly).
const readmeHeadings = `---
entity-type: task
id-style: slug
---
# Headings Fixture

### ideation

Ideation body line one.

- **Inputs:** the seed.
- **Outputs:** the design.

### ` + "`implementation`" + ` *(captain-interactive)*

Implementation body.

- **Outputs:** the deliverable.

### review of the validation stage

This heading mentions validation as a non-first token.

### done

Terminal.
`

// TestShowStageDefParity diffs native show-stage-def against the oracle across
// the well-formed, decorated-heading, malformed-heading, and missing-stage
// cases. The malformed-heading diagnostic is a hand-built f-string (not a
// Python str(e)), so it is byte-reproducible and asserted byte-for-byte.
func TestShowStageDefParity(t *testing.T) {
	cases := []struct {
		name  string
		stage string
	}{
		{"well-formed", "ideation"},
		{"decorated-heading", "implementation"},
		{"malformed-heading", "validation"},
		{"missing-stage", "nonesuch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "README.md"), readmeHeadings)

			oracle := runOracle(t, root, home, "", "show-stage-def", "--workflow-dir", root, "--stage", tc.stage)
			native := runNative("", "show-stage-def", "--workflow-dir", root, "--stage", tc.stage)

			// show-stage-def has no fetch-line rewrite; all three channels are
			// byte-identical to the oracle.
			if native.stdout != oracle.stdout {
				t.Errorf("%s: stdout mismatch\n--- native ---\n%q\n--- oracle ---\n%q", tc.name, native.stdout, oracle.stdout)
			}
			if native.stderr != oracle.stderr {
				t.Errorf("%s: stderr mismatch\n--- native ---\n%q\n--- oracle ---\n%q", tc.name, native.stderr, oracle.stderr)
			}
			if native.exit != oracle.exit {
				t.Errorf("%s: exit mismatch native=%d oracle=%d", tc.name, native.exit, oracle.exit)
			}
		})
	}
}

// TestShowStageDefMissingReadme locks the workflow-dir / README-not-found
// diagnostics against the oracle.
func TestShowStageDefMissingReadme(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	// A workflow dir that does not exist.
	missing := filepath.Join(root, "nope")
	oracle := runOracle(t, root, home, "", "show-stage-def", "--workflow-dir", missing, "--stage", "ideation")
	native := runNative("", "show-stage-def", "--workflow-dir", missing, "--stage", "ideation")
	if native.stderr != oracle.stderr || native.exit != oracle.exit {
		t.Errorf("missing-dir mismatch\nnative=(%q,%d)\noracle=(%q,%d)", native.stderr, native.exit, oracle.stderr, oracle.exit)
	}

	// A dir that exists but has no README.
	noReadme := filepath.Join(root, "empty")
	if err := os.MkdirAll(noReadme, 0o755); err != nil {
		t.Fatal(err)
	}
	oracle2 := runOracle(t, root, home, "", "show-stage-def", "--workflow-dir", noReadme, "--stage", "ideation")
	native2 := runNative("", "show-stage-def", "--workflow-dir", noReadme, "--stage", "ideation")
	if native2.stderr != oracle2.stderr || native2.exit != oracle2.exit {
		t.Errorf("no-readme mismatch\nnative=(%q,%d)\noracle=(%q,%d)", native2.stderr, native2.exit, oracle2.stderr, oracle2.exit)
	}
}
