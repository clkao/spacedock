// ABOUTME: Usage/parse-error parity — bad flags exit 1 (never 2) with the same
// ABOUTME: Error: ... stderr the oracle emits, locking the {0,1} exit domain.
package status

import (
	"path/filepath"
	"testing"
)

// usageCases are argv shapes that must exit 1 with a stderr Error: message in
// both native and oracle — never exit 2.
var usageCases = []struct {
	name string
	args func(root string) []string
}{
	{"bad-where-no-operator", func(r string) []string { return []string{"--workflow-dir", r, "--where", "status"} }},
	{"where-missing-arg", func(r string) []string { return []string{"--workflow-dir", r, "--where"} }},
	{"fields-and-all-fields", func(r string) []string {
		return []string{"--workflow-dir", r, "--fields", "a", "--all-fields"}
	}},
	{"boot-with-next", func(r string) []string { return []string{"--workflow-dir", r, "--boot", "--next"} }},
	{"next-id-with-set", func(r string) []string {
		return []string{"--workflow-dir", r, "--next-id", "--set", "x", "y=z"}
	}},
	{"resolve-missing-arg", func(r string) []string { return []string{"--workflow-dir", r, "--resolve"} }},
	{"workflow-dir-missing-arg", func(r string) []string { return []string{"--workflow-dir"} }},
	{"id-material-without-next-id", func(r string) []string {
		return []string{"--workflow-dir", r, "--id-seed", "x"}
	}},
	{"root-without-discover-or-resolve", func(r string) []string {
		return []string{"--workflow-dir", r, "--root", r}
	}},
}

func TestNativeUsageErrorsExitOneNotTwo(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	for _, tc := range usageCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			args := tc.args(root)
			nOut, nErr, nCode := runNative(t, root, env, args...)
			oOut, oErr, oCode := runOracle(t, root, env, args...)

			if nCode == 2 {
				t.Fatalf("native exit=2 for usage error %q; must be 1 (never 2)", tc.name)
			}
			if nCode != oCode {
				t.Fatalf("exit: native=%d oracle=%d (nErr=%q oErr=%q)", nCode, oCode, nErr, oErr)
			}
			if nCode != 1 {
				t.Fatalf("usage error %q exit=%d, want 1", tc.name, nCode)
			}
			if normalize(nErr, root) != normalize(oErr, root) {
				t.Fatalf("stderr mismatch for %s\n--- native ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(nErr, root), normalize(oErr, root))
			}
			if nOut != "" || oOut != "" {
				t.Fatalf("stdout must be empty on usage error: native=%q oracle=%q", nOut, oOut)
			}
		})
	}
}
