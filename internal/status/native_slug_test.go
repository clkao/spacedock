// ABOUTME: id-style: slug parity — default table id==slug, --short-id prints
// ABOUTME: the slug, and --next-id is not applicable (exit 1), all vs the oracle.
package status

import (
	"testing"
)

func slugFixture(t *testing.T) (nativeRoot, oracleRoot string) {
	t.Helper()
	readme := "---\nid-style: slug\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# Slug WF\n"
	alpha := "---\ntitle: A\nstatus: backlog\nscore: \"0.5\"\nsource: x\n---\n# A\n"
	beta := "---\ntitle: B\nstatus: done\nscore: \"0.7\"\nsource: y\n---\n# B\n"
	mk := func() string {
		dst := t.TempDir()
		writeAll(t, dst, map[string]string{
			"README.md": readme,
			"alpha.md":  alpha,
			"beta.md":   beta,
		})
		gitInit(t, dst)
		return dst
	}
	return mk(), mk()
}

func TestNativeSlugStyleParity(t *testing.T) {
	env := pinnedEnv(t)
	cases := []struct {
		name  string
		extra []string
	}{
		{"default", nil},
		{"short-id", []string{"--short-id", "alpha"}},
		{"resolve", []string{"--resolve", "alpha"}},
		{"next", []string{"--next"}},
		{"validate", []string{"--validate"}},
		{"next-id-not-applicable", []string{"--next-id"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nativeRoot, oracleRoot := slugFixture(t)
			nArgs := append([]string{"--workflow-dir", nativeRoot}, tc.extra...)
			oArgs := append([]string{"--workflow-dir", oracleRoot}, tc.extra...)

			nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
			oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

			if nCode != oCode {
				t.Fatalf("exit: native=%d oracle=%d (nErr=%q oErr=%q)", nCode, oCode, nErr, oErr)
			}
			if normalize(nOut, nativeRoot) != normalize(oOut, oracleRoot) {
				t.Fatalf("stdout mismatch for %s\n--- native ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(nOut, nativeRoot), normalize(oOut, oracleRoot))
			}
			if normalize(nErr, nativeRoot) != normalize(oErr, oracleRoot) {
				t.Fatalf("stderr mismatch for %s\n--- native ---\n%s\n--- oracle ---\n%s",
					tc.name, normalize(nErr, nativeRoot), normalize(oErr, oracleRoot))
			}
		})
	}
}

// TestNativeSlugDuplicateValidation locks slug-style duplicate-id validation: two
// entities with the same slug is impossible on disk, so the duplicate-effective-
// id path is exercised via an archived sibling sharing the active slug.
func TestNativeSlugDuplicateValidation(t *testing.T) {
	env := pinnedEnv(t)
	readme := "---\nid-style: slug\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# Slug WF\n"
	ent := "---\ntitle: T\nstatus: backlog\nscore: \"0.5\"\nsource: x\n---\n# T\n"
	mk := func() string {
		dst := t.TempDir()
		writeAll(t, dst, map[string]string{
			"README.md":       readme,
			"dup.md":          ent,
			"_archive/dup.md": ent, // archived sibling shares the slug
		})
		gitInit(t, dst)
		return dst
	}
	nativeRoot, oracleRoot := mk(), mk()

	nOut, nErr, nCode := runNative(t, nativeRoot, env, "--workflow-dir", nativeRoot, "--validate")
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, "--workflow-dir", oracleRoot, "--validate")

	if nCode != oCode || nCode != 1 {
		t.Fatalf("exit: native=%d oracle=%d, want 1 (duplicate effective id)", nCode, oCode)
	}
	if normalize(nErr, nativeRoot) != normalize(oErr, oracleRoot) {
		t.Fatalf("stderr mismatch\n--- native ---\n%s\n--- oracle ---\n%s",
			normalize(nErr, nativeRoot), normalize(oErr, oracleRoot))
	}
	if nOut != "" || oOut != "" {
		t.Fatalf("stdout must be empty on validation failure: native=%q oracle=%q", nOut, oOut)
	}
}
