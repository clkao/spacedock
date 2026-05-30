// ABOUTME: AC-1 native read parity — NativeRunner stdout/stderr/exit equals the
// ABOUTME: oracle for every read subcommand, after the shared normalization.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// nativeReadCases are the read subcommands compared native-vs-oracle on the
// seq-workflow fixture (sequential id-style, flat+folder, empty-score, etc.).
var nativeReadCases = []struct {
	name  string
	extra []string
}{
	{"default", nil},
	{"archived", []string{"--archived"}},
	{"next", []string{"--next"}},
	{"validate", []string{"--validate"}},
	{"resolve", []string{"--resolve", "003-wire-cli"}},
	{"short-id", []string{"--short-id", "003-wire-cli"}},
	{"where-status", []string{"--where", "status=ideation"}},
	// `worktree` is a non-default frontmatter key, so it appends as a single
	// extra in both runners. A default-named --fields (e.g. `source`) is NOT a
	// parity case: native de-dupes the duplicate column (captain-approved bug
	// fix) while the oracle still renders it twice — that deliberate divergence
	// is locked by TestFieldsDedupeNoDuplicateDefaultColumns instead.
	{"fields", []string{"--fields", "worktree"}},
	{"all-fields", []string{"--all-fields"}},
}

func TestNativeReadMatchesOracle(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	for _, tc := range nativeReadCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--workflow-dir", root}, tc.extra...)

			oracleOut, oracleErr, oracleCode := runOracle(t, root, env, args...)
			nativeOut, nativeErr, nativeCode := runNative(t, root, env, args...)

			if nativeCode != oracleCode {
				t.Fatalf("exit: native=%d oracle=%d (nativeErr=%q oracleErr=%q)", nativeCode, oracleCode, nativeErr, oracleErr)
			}
			if normalize(nativeOut, root) != normalize(oracleOut, root) {
				t.Fatalf("stdout differs for %s\n--- native ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(nativeOut, root), normalize(oracleOut, root))
			}
			if normalize(nativeErr, root) != normalize(oracleErr, root) {
				t.Fatalf("stderr differs for %s\n--- native ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(nativeErr, root), normalize(oracleErr, root))
			}
		})
	}
}

// TestNativeNextIDMatchesOracle locks the sd-b32 minting path: the native
// --next-id candidate equals the oracle's under identical pinned id material,
// and has the right format. This is the riskiest mechanism (SHA-256 digest +
// 5-bit big-endian extraction), validated against the oracle directly.
func TestNativeNextIDMatchesOracle(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "sdb32-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	args := []string{"--workflow-dir", root, "--next-id", "--id-seed", "pinnedseed", "--id-actor", "pinnedactor"}

	nativeOut, nativeErr, nativeCode := runNative(t, root, env, args...)
	if nativeCode != 0 {
		t.Fatalf("native exit=%d stderr=%q", nativeCode, nativeErr)
	}
	candidate := strings.TrimSpace(nativeOut)
	if len(candidate) != 24 {
		t.Fatalf("candidate %q length=%d, want 24", candidate, len(candidate))
	}
	for _, c := range candidate {
		if !strings.ContainsRune(sdB32Alphabet, c) {
			t.Fatalf("candidate %q has char %q outside SD-B32 alphabet", candidate, c)
		}
	}

	oracleOut, oracleErr, oracleCode := runOracle(t, root, env, args...)
	if oracleCode != 0 {
		t.Fatalf("oracle exit=%d stderr=%q", oracleCode, oracleErr)
	}
	if got := strings.TrimSpace(oracleOut); got != candidate {
		t.Fatalf("--next-id: native=%q oracle=%q (pinned env must reproduce)", candidate, got)
	}
}

// TestNativeBootMatchesOracle locks --boot structural + section parity for the
// sd-b32 fixture, normalizing the volatile sd-b32 NEXT_ID and root prefix.
func TestNativeBootMatchesOracle(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "sdb32-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	args := []string{"--workflow-dir", root, "--boot"}

	nativeOut, nativeErr, nativeCode := runNative(t, root, env, args...)
	if nativeCode != 0 {
		t.Fatalf("native --boot exit=%d stderr=%q", nativeCode, nativeErr)
	}
	oracleOut, _, oracleCode := runOracle(t, root, env, args...)
	if oracleCode != 0 {
		t.Fatalf("oracle --boot exit=%d", oracleCode)
	}
	normNative := sdB32Re.ReplaceAllString(normalize(nativeOut, root), "<ID>")
	normOracle := sdB32Re.ReplaceAllString(normalize(oracleOut, root), "<ID>")
	if normNative != normOracle {
		t.Fatalf("--boot parity mismatch\n--- native ---\n%s\n--- oracle ---\n%s", normNative, normOracle)
	}
}
