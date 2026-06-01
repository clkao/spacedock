// ABOUTME: AC-P2 boot regression — the seam's observable effect is confined to
// ABOUTME: TEAM_STATE present/hint; a nil probe yields a host-neutral present:false.
package status

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// bootWithProbe runs `status --boot` over the sdb32 fixture with the given probe
// wired and returns stdout. The pinned env's empty HOME makes claudeteam.Probe
// deterministic (present:false) without touching the developer's ~/.claude.
func bootWithProbe(t *testing.T, probe claudeteam.TeamStateProbe) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "sdb32-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	runner := &NativeRunner{TeamStateProbe: probe}
	code, err := runner.Run(context.Background(), Request{
		Args:   []string{"--workflow-dir", root, "--boot"},
		Dir:    root,
		Env:    pinnedEnv(t),
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil || code != 0 {
		t.Fatalf("boot exit=%d err=%v stderr=%q", code, err, stderr.String())
	}
	return stdout.String()
}

// teamStateBlock returns the TEAM_STATE section (the present + hint lines) and the
// rest of the boot output with that section excised, so the regression can assert
// the seam's effect is CONFINED to TEAM_STATE: everything outside it is byte-equal
// across probe configurations, and only the present/hint lines move.
func teamStateBlock(boot string) (block, rest string) {
	lines := strings.Split(boot, "\n")
	var blockLines, restLines []string
	inBlock := false
	for _, ln := range lines {
		switch {
		case ln == "TEAM_STATE":
			inBlock = true
			blockLines = append(blockLines, ln)
		case inBlock && (strings.HasPrefix(ln, "present:") || strings.HasPrefix(ln, "hint:")):
			blockLines = append(blockLines, ln)
		default:
			inBlock = false
			restLines = append(restLines, ln)
		}
	}
	return strings.Join(blockLines, "\n"), strings.Join(restLines, "\n")
}

// presentProbe is a stub Claude probe reporting a live team (present:true) with a
// fixed hint, so the present:true rendering is exercised without an on-disk tree.
func presentProbe(string, time.Time) (bool, string, bool) {
	return true, "recent team directory: alpha", true
}

// TestBootTeamStateProbeConfinement is the persistent AC-P2 regression pinning the
// ideation SPIKE: the injected-probe seam's observable boot effect is confined to
// the TEAM_STATE present/hint lines. Every other boot byte is identical across a
// present-true probe, a present-false probe (claudeteam.Probe over empty HOME),
// and a nil probe. TEAM_STATE itself moves exactly as designed:
//   - present-true  → present: true  + the probe's hint
//   - present-false → present: false + claudeteam.PresentFalseHint (the Claude string)
//   - nil probe     → present: false + the host-neutral hint, NO Claude string
//
// The nil-probe arm documents the host-neutrality refinement: a non-Claude host
// emits no Claude-specific advice. The byte-identical claim is scoped to the
// non-TEAM_STATE bytes; the Claude path's TEAM_STATE matches today's oracle output
// (locked separately by TestNativeBootMatchesOracle).
func TestBootTeamStateProbeConfinement(t *testing.T) {
	bootPresent := bootWithProbe(t, presentProbe)
	bootAbsent := bootWithProbe(t, claudeteam.Probe) // empty HOME → present:false
	bootNil := bootWithProbe(t, nil)

	// Confinement: the non-TEAM_STATE bytes are identical across all three.
	_, restPresent := teamStateBlock(bootPresent)
	_, restAbsent := teamStateBlock(bootAbsent)
	_, restNil := teamStateBlock(bootNil)
	if restPresent != restAbsent {
		t.Fatalf("seam leaked outside TEAM_STATE (present-true vs present-false):\n--- present ---\n%s\n--- absent ---\n%s", restPresent, restAbsent)
	}
	if restAbsent != restNil {
		t.Fatalf("seam leaked outside TEAM_STATE (probe vs nil):\n--- probe ---\n%s\n--- nil ---\n%s", restAbsent, restNil)
	}

	// TEAM_STATE moves exactly as designed.
	blockPresent, _ := teamStateBlock(bootPresent)
	wantPresent := "TEAM_STATE\npresent: true\nhint: recent team directory: alpha"
	if blockPresent != wantPresent {
		t.Fatalf("present-true TEAM_STATE:\n got %q\nwant %q", blockPresent, wantPresent)
	}

	blockAbsent, _ := teamStateBlock(bootAbsent)
	wantAbsent := "TEAM_STATE\npresent: false\nhint: " + claudeteam.PresentFalseHint
	if blockAbsent != wantAbsent {
		t.Fatalf("present-false (Claude) TEAM_STATE:\n got %q\nwant %q", blockAbsent, wantAbsent)
	}

	blockNil, _ := teamStateBlock(bootNil)
	wantNil := "TEAM_STATE\npresent: false\nhint: " + teamStateNeutralHint
	if blockNil != wantNil {
		t.Fatalf("nil-probe TEAM_STATE:\n got %q\nwant %q", blockNil, wantNil)
	}

	// The host-neutral hint must NOT carry the Claude-only advice (AC-3 intent).
	if strings.Contains(blockNil, "claude runtime") || strings.Contains(blockNil, "TeamCreate") {
		t.Fatalf("nil-probe TEAM_STATE leaked Claude-specific advice: %q", blockNil)
	}
}
