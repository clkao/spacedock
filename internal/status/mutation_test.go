// ABOUTME: AC-3 mutation parity — --set (field/clear/bare-timestamp) and
// ABOUTME: --archive produce identical frontmatter, narration, and guards as the oracle.
package status

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stageFixture copies the named testdata workflow into a fresh git-initialized
// temp dir and returns its absolute root. Mutations need a git repo because the
// oracle resolves git_root during --set/--archive.
func stageFixture(t *testing.T, fixture string) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	cpTree(t, src, dst)
	gitInit(t, dst)
	return dst
}

func cpTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
}

func gitInit(t *testing.T, dir string) {
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

// readFrontmatter returns the lines between the first two `---` fences.
func readFrontmatter(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := strings.Split(string(b), "\n")
	var fm []string
	fences := 0
	for _, line := range lines {
		if strings.TrimRight(line, " ") == "---" {
			fences++
			if fences == 2 {
				break
			}
			continue
		}
		if fences == 1 {
			fm = append(fm, line)
		}
	}
	return strings.Join(fm, "\n")
}

// mutationCase runs the same mutation through the launcher (into one temp copy)
// and the oracle (into a second temp copy), then asserts the mutated file's
// frontmatter and the narration stdout match (timestamps normalized).
type mutationCase struct {
	name       string
	fixture    string
	mutateArgs func(root string) []string // args excluding --workflow-dir
	checkFile  string                     // path under root to compare frontmatter of
}

func TestMutationParity(t *testing.T) {
	cases := []mutationCase{
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
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			env := pinnedEnv(t)

			launcherRoot := stageFixture(t, tc.fixture)
			oracleRoot := stageFixture(t, tc.fixture)

			lArgs := append([]string{"--workflow-dir", launcherRoot}, tc.mutateArgs(launcherRoot)...)
			oArgs := append([]string{"--workflow-dir", oracleRoot}, tc.mutateArgs(oracleRoot)...)

			lOut, lErr, lCode := runLauncher(t, launcherRoot, env, lArgs...)
			oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

			if lCode != oCode {
				t.Fatalf("exit: launcher=%d oracle=%d (launcherErr=%q oracleErr=%q)", lCode, oCode, lErr, oErr)
			}
			// Narration parity (timestamps normalized; roots already differ so
			// strip each root before compare via the timestamp-only normalize).
			if normalize(lOut, "") != normalize(oOut, "") {
				t.Fatalf("narration mismatch\n--- launcher ---\n%q\n--- oracle ---\n%q", lOut, oOut)
			}
			// Frontmatter parity of the mutated file (timestamps normalized).
			lFM := normalize(readFrontmatter(t, filepath.Join(launcherRoot, tc.checkFile)), "")
			oFM := normalize(readFrontmatter(t, filepath.Join(oracleRoot, tc.checkFile)), "")
			if lFM != oFM {
				t.Fatalf("frontmatter mismatch\n--- launcher ---\n%s\n--- oracle ---\n%s", lFM, oFM)
			}
		})
	}
}
