// ABOUTME: Discovery + --discover native parity — flat/folder conflict warning
// ABOUTME: on the read path, and the --discover workflow walk match the oracle.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeConflictWarningParity locks the flat/folder conflict warning on the
// read path: a workflow where a slug has both forms emits the same stderr
// warning native-vs-oracle and prefers the folder form. The fixture keeps ids
// distinct so validation does not fail first (the warning, not a validation
// error, is what we assert).
func TestNativeConflictWarningParity(t *testing.T) {
	env := pinnedEnv(t)
	body := func(id string) string {
		return "---\nid: \"" + id + "\"\ntitle: T\nstatus: backlog\nscore: \"0.5\"\nsource: x\n---\n# T\n"
	}
	// Note: a flat/folder conflict makes --validate fail, which gates the default
	// table. So assert the warning via --next-id is not possible either (it also
	// validates). Instead drive discovery directly through the resolver path that
	// emits the warning without global validation: --set targets the slug.
	mk := func() string {
		dst := t.TempDir()
		writeAll(t, dst, map[string]string{
			"README.md":         seqREADME,
			"conflict.md":       body("001"),
			"conflict/index.md": body("002"),
			"other.md":          body("003"),
		})
		gitInit(t, dst)
		return dst
	}
	nativeRoot := mk()
	oracleRoot := mk()

	// --set on the conflicting slug resolves via discovery (folder wins) and
	// emits the warning, without running global validation.
	args := []string{"--set", "conflict", "status=done"}
	nArgs := append([]string{"--workflow-dir", nativeRoot}, args...)
	oArgs := append([]string{"--workflow-dir", oracleRoot}, args...)

	nOut, nErr, nCode := runNative(t, nativeRoot, env, nArgs...)
	oOut, oErr, oCode := runOracle(t, oracleRoot, env, oArgs...)

	if nCode != oCode {
		t.Fatalf("exit: native=%d oracle=%d (nErr=%q oErr=%q)", nCode, oCode, nErr, oErr)
	}
	if !strings.Contains(nErr, "preferring folder form") {
		t.Fatalf("native stderr missing conflict warning: %q", nErr)
	}
	if normalize(nErr, nativeRoot) != normalize(oErr, oracleRoot) {
		t.Fatalf("conflict warning mismatch\n--- native ---\n%s\n--- oracle ---\n%s",
			normalize(nErr, nativeRoot), normalize(oErr, oracleRoot))
	}
	if normalize(nOut, nativeRoot) != normalize(oOut, oracleRoot) {
		t.Fatalf("stdout mismatch: native=%q oracle=%q", nOut, oOut)
	}
	// Folder form is the one that got mutated.
	folder := readWhole(t, filepath.Join(nativeRoot, "conflict", "index.md"))
	if !strings.Contains(folder, "status: done") {
		t.Fatalf("folder form should have been mutated (folder wins):\n%s", folder)
	}
}

// TestNativeDiscoverParity locks --discover: the native workflow walk returns
// the same workflow dirs as the oracle for a tree with a commissioned README.
func TestNativeDiscoverParity(t *testing.T) {
	env := pinnedEnv(t)
	root := t.TempDir()
	// A commissioned workflow and a non-workflow README in the tree.
	writeAll(t, root, map[string]string{
		"wf/README.md":         "---\ncommissioned-by: spacedock@1.0\nid-style: sequential\n---\n# WF\n",
		"plain/README.md":      "---\ntitle: not a workflow\n---\n# Plain\n",
		"nested/wf2/README.md": "---\ncommissioned-by: spacedock@\n---\n# WF2\n",
	})
	gitInit(t, root)

	args := []string{"--discover", "--root", root}
	nOut, nErr, nCode := runNative(t, root, env, args...)
	oOut, oErr, oCode := runOracle(t, root, env, args...)

	if nCode != oCode {
		t.Fatalf("exit: native=%d oracle=%d (nErr=%q oErr=%q)", nCode, oCode, nErr, oErr)
	}
	if normalize(nOut, root) != normalize(oOut, root) {
		t.Fatalf("--discover output mismatch\n--- native ---\n%s\n--- oracle ---\n%s",
			normalize(nOut, root), normalize(oOut, root))
	}
}

func writeAll(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
