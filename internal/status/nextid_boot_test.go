// ABOUTME: AC-1 parity for the non-static read flags — --next-id (format +
// ABOUTME: oracle-equality under pinned env) and --boot (structural + section parity).
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// sdB32Alphabet is the SD-B32 alphabet the --next-id candidate is drawn from.
const sdB32Alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// AC-1: --next-id is in scope for pass-through but asserted at format + oracle-
// equality level (not a static golden), since the candidate is SHA-derived. The
// harness pins all id material (timestamp via env, seed/actor via flags) so the
// launcher and the oracle produce the identical reproducible candidate.
func TestNextIDParity(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "sdb32-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	args := []string{"--workflow-dir", root, "--next-id", "--id-seed", "pinnedseed", "--id-actor", "pinnedactor"}

	launcherOut, launcherErr, launcherCode := runLauncher(t, root, env, args...)
	if launcherCode != 0 {
		t.Fatalf("launcher exit=%d stderr=%q", launcherCode, launcherErr)
	}
	candidate := strings.TrimSpace(launcherOut)

	// Format: 24 chars, all in the SD-B32 alphabet.
	if len(candidate) != 24 {
		t.Fatalf("--next-id candidate %q length=%d, want 24", candidate, len(candidate))
	}
	for _, c := range candidate {
		if !strings.ContainsRune(sdB32Alphabet, c) {
			t.Fatalf("--next-id candidate %q has char %q outside SD-B32 alphabet", candidate, c)
		}
	}

	// Oracle-equality under the identical pinned env: the launcher injects no
	// id-material difference of its own.
	oracleOut, oracleErr, oracleCode := runOracle(t, root, env, args...)
	if oracleCode != 0 {
		t.Fatalf("oracle exit=%d stderr=%q", oracleCode, oracleErr)
	}
	if got, want := strings.TrimSpace(oracleOut), candidate; got != want {
		t.Fatalf("--next-id: launcher=%q oracle=%q (pinned env must reproduce)", want, got)
	}
}

// bootSections are the --boot section headers in their required order. The FO
// parses --boot by section at startup, so order and presence are load-bearing.
var bootSections = []string{
	"MODS:", "ID_STYLE:", "NEXT_ID:", "ORPHANS:", "PR_STATE:", "DISPATCHABLE", "TEAM_STATE",
}

// AC-1: --boot is verified structurally (headers present and in order) and the
// deterministic section bodies (ID_STYLE, NEXT_ID, DISPATCHABLE) are parity-
// checked against the oracle. Volatile material (NEXT_ID value, TEAM_STATE hint)
// is normalized; the fixture has no orphans/PRs so those render their `none`
// forms. The launcher and oracle run under the identical env (same HOME) so the
// TEAM_STATE probe is identical between them.
func TestBootStructuralParity(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "sdb32-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	args := []string{"--workflow-dir", root, "--boot"}

	launcherOut, launcherErr, launcherCode := runLauncher(t, root, env, args...)
	if launcherCode != 0 {
		t.Fatalf("launcher exit=%d stderr=%q", launcherCode, launcherErr)
	}

	// Section headers present and in order.
	lastIdx := -1
	for _, section := range bootSections {
		idx := strings.Index(launcherOut, section)
		if idx < 0 {
			t.Fatalf("--boot output missing section %q\n%s", section, launcherOut)
		}
		if idx < lastIdx {
			t.Fatalf("--boot section %q out of order\n%s", section, launcherOut)
		}
		lastIdx = idx
	}

	// ID_STYLE body parity (deterministic).
	if !strings.Contains(launcherOut, "ID_STYLE: sd-b32") {
		t.Fatalf("--boot ID_STYLE body wrong\n%s", launcherOut)
	}

	// Full structural parity vs oracle after normalizing the sd-b32 NEXT_ID and
	// the root prefix. HOME is pinned identically so TEAM_STATE matches.
	oracleOut, _, oracleCode := runOracle(t, root, env, args...)
	if oracleCode != 0 {
		t.Fatalf("oracle --boot exit=%d", oracleCode)
	}
	normLauncher := sdB32Re.ReplaceAllString(normalize(launcherOut, root), "<ID>")
	normOracle := sdB32Re.ReplaceAllString(normalize(oracleOut, root), "<ID>")
	if normLauncher != normOracle {
		t.Fatalf("--boot structural parity mismatch\n--- launcher ---\n%s\n--- oracle ---\n%s", normLauncher, normOracle)
	}
}
