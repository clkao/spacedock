// ABOUTME: AC-P2 build regression — the bare-mode advisory is gated on the probe:
// ABOUTME: Claude (probe) + no evidence emits it; a nil probe (Codex/bare) emits none.
package dispatch

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// bareBuildFixture builds a single-root bare-mode fixture under an empty HOME (so
// claudeteam.Probe reports no recent team evidence, the advisory-eligible state)
// and returns the root and the bare_mode build stdin. Both probe arms run against
// THIS one fixture so the only varying input is the probe — the dispatch envelope
// must then be byte-identical across arms.
func bareBuildFixture(t *testing.T) (root, stdin string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir()) // empty ~/.claude → no recent team evidence
	root = t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
	entityPath := filepath.Join(root, "thing.md")
	writeFile(t, entityPath, entityFM("Thing", "backlog", ""))
	gitInit(t, root)

	stdin = mergeStdin(map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "backlog",
		"checklist":      []string{"- a", "- b"},
		"bare_mode":      true,
	}, nil)
	return root, stdin
}

// runBareBuild runs the bare_mode build over the given fixture with the given probe.
func runBareBuild(t *testing.T, root, stdin string, probe claudeteam.TeamStateProbe) (stdout, stderr string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	if code := Run(probe, []string{"build", "--workflow-dir", root}, strings.NewReader(stdin), &out, &errBuf); code != 0 {
		t.Fatalf("bare build exit=%d stderr=%q", code, errBuf.String())
	}
	return out.String(), errBuf.String()
}

// advisoryMarker is a stable fragment of the Claude bare-mode advisory text. The
// regression keys on its presence/absence rather than the full sentence so a
// future wording tweak in claudeteam.BareModeAdvisory does not break the gate.
const advisoryMarker = "bare_mode dispatch with no recent TeamCreate evidence"

// TestBuildBareAdvisoryProbeGate is the persistent AC-P2 regression for the build
// surface: the bare-mode advisory's PRESENCE is the seam's only observable effect.
//   - probe supplied (Claude) + no recent evidence → advisory fires on stderr.
//   - probe nil (Codex/bare)                        → NO advisory at all.
//
// The dispatch envelope on stdout is byte-identical across both arms — the seam
// touches only the advisory's presence. The nil-probe-no-advisory arm documents
// the host-neutrality refinement: the advisory names a Claude-only tool
// (TeamCreate), so a non-Claude host correctly emits none. This is a deliberate,
// scoped behavior change, NOT a parity regression: the byte-for-byte AC-P2 claim
// is the Claude (probe-supplied) path, whose advisory behavior is unchanged from
// the pre-seam unconditional read.
func TestBuildBareAdvisoryProbeGate(t *testing.T) {
	root, stdin := bareBuildFixture(t)
	stdoutProbe, stderrProbe := runBareBuild(t, root, stdin, claudeteam.Probe)
	stdoutNil, stderrNil := runBareBuild(t, root, stdin, nil)

	if stdoutProbe != stdoutNil {
		t.Fatalf("dispatch envelope diverged between probe and nil arms:\n--- probe ---\n%s\n--- nil ---\n%s", stdoutProbe, stdoutNil)
	}
	if !strings.Contains(stderrProbe, advisoryMarker) {
		t.Fatalf("Claude probe + no evidence must emit the bare-mode advisory; stderr=%q", stderrProbe)
	}
	if strings.Contains(stderrNil, advisoryMarker) {
		t.Fatalf("nil probe (Codex/bare) must emit NO bare-mode advisory; stderr=%q", stderrNil)
	}
}
