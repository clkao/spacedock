// ABOUTME: A-1 regression — discovery must not gate the --root --resolve and
// ABOUTME: plain --resolve paths, which resolve without an enclosing cwd workflow.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// commissionedSeqREADME is a commissioned slug-style README so discoverWorkflows
// recognizes the workflow for the --root cross-workflow resolve path.
const commissionedSeqREADME = `---
commissioned-by: spacedock@1
id-style: slug
stages:
  states:
    - name: backlog
      initial: true
    - name: done
      terminal: true
---

# WFA
`

// TestRootResolveSkipsDiscovery is the A-1 BLOCKER regression: `status --root
// <root> --resolve 'wfa::shared-task'` from a NON-enclosed cwd must exit 0 and
// resolve the entity. The --root path never consumes the cwd workflow, so the
// discovery gate must not hard-error it. Pre-fix this regressed to exit 1 with
// the no-workflow error.
func TestRootResolveSkipsDiscovery(t *testing.T) {
	root := t.TempDir()
	writeAll(t, root, map[string]string{
		"wfa/README.md":      commissionedSeqREADME,
		"wfa/shared-task.md": "---\nstatus: backlog\n---\n# Shared\n",
	})
	gitInit(t, root)

	// cwd is an unrelated, non-enclosed empty tempdir — no workflow above it.
	nonEnclosed := t.TempDir()
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, nonEnclosed, env, "--root", root, "--resolve", "wfa::shared-task")
	if code != 0 {
		t.Fatalf("--root --resolve exit=%d, want 0 (stdout=%q stderr=%q)", code, out, stderr)
	}
	if strings.Contains(stderr, "no Spacedock workflow here") {
		t.Fatalf("--root --resolve must not hit the no-workflow gate; stderr=%q", stderr)
	}
	// path= is realpath-derived (discoverWorkflows canonicalizes the root), so
	// assert on the slug and the entity's file location rather than a raw prefix.
	if !strings.Contains(out, "slug=shared-task") ||
		!strings.Contains(out, filepath.Join("wfa", "shared-task.md")) {
		t.Fatalf("--root --resolve output = %q, want it to resolve shared-task under wfa/", out)
	}
}

// TestRootResolveUnqualifiedSkipsDiscovery covers the unqualified --root
// --resolve form (no wf:: prefix) from a non-enclosed cwd: it scans all
// workflows under root and must likewise skip the discovery gate.
func TestRootResolveUnqualifiedSkipsDiscovery(t *testing.T) {
	root := t.TempDir()
	writeAll(t, root, map[string]string{
		"wfa/README.md":      commissionedSeqREADME,
		"wfa/shared-task.md": "---\nstatus: backlog\n---\n# Shared\n",
	})
	gitInit(t, root)
	nonEnclosed := t.TempDir()
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, nonEnclosed, env, "--root", root, "--resolve", "shared-task")
	if code != 0 {
		t.Fatalf("unqualified --root --resolve exit=%d, want 0 (stderr=%q)", code, stderr)
	}
	if !strings.Contains(out, "slug=shared-task") {
		t.Fatalf("unqualified --root --resolve output = %q, want shared-task resolved", out)
	}
}

// TestPlainResolveFromNonWorkflowEmitsNoWorkflow pins the polish-item decision:
// a plain `--resolve <ref>` (no --root, no --workflow-dir) from a non-enclosed
// cwd resolves via discovery, so a non-workflow cwd yields the named
// no-workflow error (exit 1) — the more actionable message — rather than the
// prior "unknown reference". This path DOES require a workflow, so the change is
// intended and kept.
func TestPlainResolveFromNonWorkflowEmitsNoWorkflow(t *testing.T) {
	nonEnclosed := t.TempDir()
	env := pinnedEnv(t)

	out, stderr, code := runNative(t, nonEnclosed, env, "--resolve", "anything")
	if code != 1 {
		t.Fatalf("plain --resolve from non-workflow exit=%d, want 1 (stdout=%q stderr=%q)", code, out, stderr)
	}
	if stderr != noWorkflowErr {
		t.Fatalf("plain --resolve stderr = %q, want the no-workflow error %q", stderr, noWorkflowErr)
	}
}
