// ABOUTME: ORPHANS absolute-worktree parity — an absolute `worktree:` value is
// ABOUTME: used as-is for the DIR_EXISTS probe, matching os.path.join semantics.
package status

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// bootOrphanReadme defines a single worktree-bearing stage so --boot renders an
// ORPHANS table for an entity that carries a worktree field.
const bootOrphanReadme = `---
commissioned-by: spacedock@1
id-style: slug
stages:
  states:
    - name: build
      initial: true
      worktree: true
---

# Orphan Boot Workflow
`

// orphanDirExists returns the DIR_EXISTS cell of the single ORPHANS data row in
// a --boot rendering, or "" if no ORPHANS table is present.
func orphanDirExists(boot string) string {
	lines := strings.Split(boot, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "ORPHANS") && i+2 < len(lines) {
			fields := strings.Fields(lines[i+2])
			if len(fields) >= 4 {
				return fields[3]
			}
		}
	}
	return ""
}

// TestBootAbsoluteWorktreeDirExists is the carried boot-fix parity test: an
// absolute `worktree:` value resolves to that absolute dir (os.path.join drops
// git_root), so DIR_EXISTS is yes when it exists — both directly and vs the
// oracle. The pre-fix native code joined the absolute path onto git_root,
// yielding a non-existent path and DIR_EXISTS=no.
func TestBootAbsoluteWorktreeDirExists(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	// An absolute worktree path that exists but is OUTSIDE the git root, so a
	// filepath.Join(git_root, wt) would point at a non-existent nested path.
	absWorktree := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), bootOrphanReadme)
	writeFile(t, filepath.Join(root, "feature.md"),
		"---\nstatus: build\nworktree: "+absWorktree+"\n---\n")

	env := pinnedEnv(t)
	args := []string{"--workflow-dir", root, "--boot"}

	nativeOut, nativeErr, nativeCode := runNative(t, root, env, args...)
	if nativeCode != 0 {
		t.Fatalf("native --boot exit=%d stderr=%q", nativeCode, nativeErr)
	}
	if got := orphanDirExists(nativeOut); got != "yes" {
		t.Fatalf("DIR_EXISTS for absolute existing worktree = %q, want \"yes\"\n%s", got, nativeOut)
	}

	// Oracle parity. The oracle is resolved in-tree, so this comparison always
	// runs (and hard-fails on a real divergence) on top of the direct DIR_EXISTS
	// assertion above.
	oracleOut, _, oracleCode := runOracle(t, root, env, args...)
	if oracleCode != 0 {
		t.Fatalf("oracle --boot exit=%d", oracleCode)
	}
	if got, want := orphanDirExists(nativeOut), orphanDirExists(oracleOut); got != want {
		t.Fatalf("ORPHANS DIR_EXISTS native=%q oracle=%q\n--- native ---\n%s\n--- oracle ---\n%s",
			got, want, nativeOut, oracleOut)
	}
}
