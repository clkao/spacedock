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

// TestNativeFolderEntityReportAppendPreservesTracker (AC-3) verifies the
// folder-form combination: a folder entity (index.md + reports/ subdir) carrying
// issue/source, after a stage-report body append, is discovered as exactly ONE
// entity (the reports/ subdir is not misdetected as a second entity) and its
// issue/source frontmatter survives the append intact.
func TestNativeFolderEntityReportAppendPreservesTracker(t *testing.T) {
	const index = "---\nid: \"070\"\ntitle: Tracker-linked folder entity\nstatus: implementation\nscore: \"0.50\"\nsource: kata\nissue: kata:task-xyz789\n---\n# Tracker-linked folder entity\n\nFolder-form entity carrying external-tracker fields.\n"
	const reportSeed = "ideation notes\n"
	env := pinnedEnv(t)

	// Folder entity with a reports/ subdir, mirrored into native + oracle copies.
	extra := map[string]string{
		"070-tracker/index.md":            index,
		"070-tracker/reports/ideation.md": reportSeed,
	}
	nativeRoot := stageFixtureWith(t, "seq-workflow", extra)
	oracleRoot := stageFixtureWith(t, "seq-workflow", extra)

	// Stage-report append: grow the entity body, the way an ensign appends its
	// "## Stage Report" section to index.md.
	report := "\n## Stage Report: implementation\n\n- DONE: wired the thing\n  ref abc123\n"
	for _, root := range []string{nativeRoot, oracleRoot} {
		idx := filepath.Join(root, "070-tracker", "index.md")
		body := readWhole(t, idx)
		if err := os.WriteFile(idx, []byte(body+report), 0o644); err != nil {
			t.Fatalf("append report to %s: %v", idx, err)
		}
	}

	// Default status read after the append: native must match the oracle on every
	// channel. The oracle is the authoritative entity-count oracle, so parity here
	// already proves native counts the folder entity exactly as the oracle does.
	args := []string{"--workflow-dir", nativeRoot}
	nOut, nErr, nCode := runNative(t, nativeRoot, env, args...)
	oArgs := []string{"--workflow-dir", oracleRoot}
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)
	if nCode != oCode {
		t.Fatalf("exit: native=%d oracle=%d (nativeErr=%q oracleErr=%q)", nCode, oCode, nErr, oErr)
	}
	if normalize(nOut, nativeRoot) != normalize(oOut, oracleRoot) {
		t.Fatalf("status read mismatch after report append\n--- native ---\n%s\n--- oracle ---\n%s",
			normalize(nOut, nativeRoot), normalize(oOut, oracleRoot))
	}

	// Discovered as exactly ONE entity: the folder slug appears in a single data
	// row, never duplicated by the reports/ subdir.
	if got := strings.Count(nOut, "070-tracker"); got != 1 {
		t.Fatalf("folder entity '070-tracker' appears %d times, want exactly 1 (misdetected?):\n%s", got, nOut)
	}

	// issue/source survive the body append byte-for-byte.
	fm := readFrontmatter(t, filepath.Join(nativeRoot, "070-tracker", "index.md"))
	for _, line := range []string{"source: kata", "issue: kata:task-xyz789"} {
		if !strings.Contains(fm, line) {
			t.Fatalf("report append dropped/reformatted %q from frontmatter:\n%s", line, fm)
		}
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
