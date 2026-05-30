// ABOUTME: AC-4/AC-5 native mutation parity — --set and --archive produce the
// ABOUTME: same frontmatter, narration, guards, and unknown-field preservation.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nativeMutationCase runs a mutation natively (one temp copy) and through the
// oracle (a second temp copy), diffing the mutated file + narration.
type nativeMutationCase struct {
	name       string
	fixture    string
	mutateArgs func(root string) []string
	checkFile  string
}

func TestNativeMutationParity(t *testing.T) {
	cases := []nativeMutationCase{
		{
			name:    "set-field",
			fixture: "seq-workflow",
			mutateArgs: func(root string) []string {
				return []string{"--set", "002-vendor-script", "status=implementation"}
			},
			checkFile: "002-vendor-script.md",
		},
		{
			name:    "set-clear",
			fixture: "seq-workflow",
			mutateArgs: func(root string) []string {
				return []string{"--set", "002-vendor-script", "source="}
			},
			checkFile: "002-vendor-script.md",
		},
		{
			name:    "set-bare-timestamp-fill",
			fixture: "seq-workflow",
			mutateArgs: func(root string) []string {
				return []string{"--set", "002-vendor-script", "started"}
			},
			checkFile: "002-vendor-script.md",
		},
		{
			name:    "set-insert-missing-field",
			fixture: "seq-workflow",
			mutateArgs: func(root string) []string {
				return []string{"--set", "002-vendor-script", "pr=#42"}
			},
			checkFile: "002-vendor-script.md",
		},
		{
			name:    "archive-flat",
			fixture: "seq-workflow",
			mutateArgs: func(root string) []string {
				return []string{"--archive", "001-design-seam"}
			},
			checkFile: "_archive/001-design-seam.md",
		},
		{
			name:    "archive-folder",
			fixture: "seq-workflow",
			mutateArgs: func(root string) []string {
				return []string{"--archive", "003-wire-cli"}
			},
			checkFile: "_archive/003-wire-cli/index.md",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			env := pinnedEnv(t)
			nativeRoot := stageFixture(t, tc.fixture)
			oracleRoot := stageFixture(t, tc.fixture)

			nArgs := append([]string{"--workflow-dir", nativeRoot}, tc.mutateArgs(nativeRoot)...)
			oArgs := append([]string{"--workflow-dir", oracleRoot}, tc.mutateArgs(oracleRoot)...)

			nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
			oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

			if nCode != oCode {
				t.Fatalf("exit: native=%d oracle=%d (nativeErr=%q oracleErr=%q)", nCode, oCode, nErr, oErr)
			}
			// Narration parity: normalize timestamps and each run's own root.
			if normalize(nOut, nativeRoot) != normalize(oOut, oracleRoot) {
				t.Fatalf("narration mismatch\n--- native ---\n%q\n--- oracle ---\n%q", nOut, oOut)
			}
			// Full mutated-file parity (whole file, not just frontmatter) so the
			// EOF-newline identity and body are byte-checked.
			nFile := readWhole(t, filepath.Join(nativeRoot, tc.checkFile))
			oFile := readWhole(t, filepath.Join(oracleRoot, tc.checkFile))
			if normalize(nFile, nativeRoot) != normalize(oFile, oracleRoot) {
				t.Fatalf("file mismatch for %s\n--- native ---\n%q\n--- oracle ---\n%q", tc.checkFile, nFile, oFile)
			}
		})
	}
}

func readWhole(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestNativeUnknownFieldPreservation (AC-5) verifies issue/source/arbitrary
// unknown top-level fields survive --set (on a different field) and --archive
// byte-for-byte, diffed against the oracle.
func TestNativeUnknownFieldPreservation(t *testing.T) {
	const body = "---\nid: \"050\"\ntitle: Tracker-linked entity\nstatus: ideation\nscore: \"0.40\"\nsource: linear\nissue: ENG-123\ntracker-url: https://linear.app/x/ENG-123\n---\n# Tracker-linked entity\n\nCarries external-tracker fields that must survive mutation.\n"

	steps := []struct {
		name string
		args func(root string) []string
		file string
	}{
		{
			name: "set-unrelated-field",
			args: func(root string) []string { return []string{"--set", "050-tracker", "status=implementation"} },
			file: "050-tracker.md",
		},
		{
			name: "archive",
			args: func(root string) []string { return []string{"--archive", "050-tracker"} },
			file: "_archive/050-tracker.md",
		},
	}

	for _, step := range steps {
		step := step
		t.Run(step.name, func(t *testing.T) {
			env := pinnedEnv(t)
			nativeRoot := stageFixtureWith(t, "seq-workflow", map[string]string{"050-tracker.md": body})
			oracleRoot := stageFixtureWith(t, "seq-workflow", map[string]string{"050-tracker.md": body})

			nArgs := append([]string{"--workflow-dir", nativeRoot}, step.args(nativeRoot)...)
			oArgs := append([]string{"--workflow-dir", oracleRoot}, step.args(oracleRoot)...)

			nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
			oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)
			if nCode != oCode {
				t.Fatalf("exit: native=%d oracle=%d (nativeErr=%q oracleErr=%q)", nCode, oCode, nErr, oErr)
			}
			if normalize(nOut, nativeRoot) != normalize(oOut, oracleRoot) {
				t.Fatalf("narration mismatch\n--- native ---\n%q\n--- oracle ---\n%q", nOut, oOut)
			}

			nFile := readWhole(t, filepath.Join(nativeRoot, step.file))
			oFile := readWhole(t, filepath.Join(oracleRoot, step.file))
			if normalize(nFile, nativeRoot) != normalize(oFile, oracleRoot) {
				t.Fatalf("file mismatch\n--- native ---\n%q\n--- oracle ---\n%q", nFile, oFile)
			}
			// Explicit byte-for-byte survival of the unknown/tracker fields.
			for _, line := range []string{"source: linear", "issue: ENG-123", "tracker-url: https://linear.app/x/ENG-123"} {
				if !strings.Contains(nFile, line) {
					t.Fatalf("native dropped/reformatted %q:\n%s", line, nFile)
				}
			}
		})
	}
}

// stageFixtureWith copies a fixture into a git temp dir and adds/overwrites the
// given relative files before committing, so extra fixture entities can be
// injected per-test.
func stageFixtureWith(t *testing.T, fixture string, extra map[string]string) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	cpTree(t, src, dst)
	for rel, content := range extra {
		p := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitInit(t, dst)
	return dst
}
