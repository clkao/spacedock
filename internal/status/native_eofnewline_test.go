// ABOUTME: EOF-newline identity — the frontmatter writer preserves the file's
// ABOUTME: terminal-newline state exactly, verified native-vs-oracle both ways.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeEOFNewlineIdentity locks the byte trap the design flagged: a --set
// on a file WITHOUT a trailing newline must not add one, and a file WITH one
// must keep it. Verified by diffing the whole mutated file native-vs-oracle for
// both shapes (the oracle's '\n'.join(content.split('\n')) is EOF-identity).
func TestNativeEOFNewlineIdentity(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "no-trailing-newline",
			body: "---\nid: \"060\"\ntitle: No EOF newline\nstatus: backlog\nscore: \"0.5\"\nsource: x\n---\n# No EOF newline\n\nThis file does not end in a newline.",
		},
		{
			name: "with-trailing-newline",
			body: "---\nid: \"061\"\ntitle: EOF newline\nstatus: backlog\nscore: \"0.5\"\nsource: x\n---\n# EOF newline\n\nThis file ends in a newline.\n",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Confirm the input shape is what we intend before mutating.
			endsWithNewline := strings.HasSuffix(tc.body, "\n")

			env := pinnedEnv(t)
			nativeRoot := stageFixtureWith(t, "seq-workflow", map[string]string{"eof-entity.md": tc.body})
			oracleRoot := stageFixtureWith(t, "seq-workflow", map[string]string{"eof-entity.md": tc.body})

			args := []string{"--set", "eof-entity", "status=done"}
			nArgs := append([]string{"--workflow-dir", nativeRoot}, args...)
			oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

			_, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
			_, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)
			if nCode != oCode {
				t.Fatalf("exit: native=%d oracle=%d (nErr=%q oErr=%q)", nCode, oCode, nErr, oErr)
			}

			nFile := readWhole(t, filepath.Join(nativeRoot, "eof-entity.md"))
			oFile := readWhole(t, filepath.Join(oracleRoot, "eof-entity.md"))
			if nFile != oFile {
				t.Fatalf("EOF-newline mismatch\n--- native ---\n%q\n--- oracle ---\n%q", nFile, oFile)
			}
			// The mutated file must keep the input's terminal-newline state.
			if strings.HasSuffix(nFile, "\n") != endsWithNewline {
				t.Fatalf("native changed terminal-newline state: input had newline=%v, output %q", endsWithNewline, nFile)
			}
		})
	}
}
