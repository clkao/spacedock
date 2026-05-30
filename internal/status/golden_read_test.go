// ABOUTME: AC-1/AC-2 golden read parity — launcher stdout for the five read
// ABOUTME: subcommands matches goldens captured from the oracle, after normalization.
package status

import (
	"path/filepath"
	"testing"
)

// readCases are the five FO-load-bearing read subcommands compared byte-for-byte
// (post-normalization) against goldens captured from the oracle.
var readCases = []struct {
	name   string
	golden string
	extra  []string // args after --workflow-dir <root>
}{
	{name: "default", golden: "seq-default.txt", extra: nil},
	{name: "next", golden: "seq-next.txt", extra: []string{"--next"}},
	{name: "validate", golden: "seq-validate.txt", extra: []string{"--validate"}},
	{name: "resolve", golden: "seq-resolve.txt", extra: []string{"--resolve", "003-wire-cli"}},
	{name: "short-id", golden: "seq-short-id.txt", extra: []string{"--short-id", "003-wire-cli"}},
}

func TestGoldenRead(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	for _, tc := range readCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--workflow-dir", root}, tc.extra...)

			if *update {
				out, stderr, code := runOracle(t, root, env, args...)
				if code != 0 {
					t.Fatalf("oracle exit=%d stderr=%q while capturing golden", code, stderr)
				}
				writeGolden(t, tc.golden, normalize(out, root))
				return
			}

			out, stderr, code := runLauncher(t, root, env, args...)
			if code != 0 {
				t.Fatalf("launcher exit=%d stderr=%q", code, stderr)
			}
			got := normalize(out, root)
			want := readGolden(t, tc.golden)
			if got != want {
				t.Fatalf("read parity mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}

// AC-1/AC-2: the launcher introduces no formatting difference vs the live oracle
// for the same (args, dir, env). This runs both on this machine and compares
// normalized output directly, so it catches launcher-injected drift even if the
// goldens were captured on a different machine.
func TestLauncherMatchesOracleRead(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	for _, tc := range readCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--workflow-dir", root}, tc.extra...)

			oracleOut, oracleErr, oracleCode := runOracle(t, root, env, args...)
			launcherOut, launcherErr, launcherCode := runLauncher(t, root, env, args...)

			if launcherCode != oracleCode {
				t.Fatalf("exit code: launcher=%d oracle=%d", launcherCode, oracleCode)
			}
			if normalize(launcherOut, root) != normalize(oracleOut, root) {
				t.Fatalf("stdout differs for %s\n--- launcher ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(launcherOut, root), normalize(oracleOut, root))
			}
			if normalize(launcherErr, root) != normalize(oracleErr, root) {
				t.Fatalf("stderr differs for %s\n--- launcher ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(launcherErr, root), normalize(oracleErr, root))
			}
		})
	}
}
