// ABOUTME: Native-runner selectability — NativeRunner satisfies status.Runner
// ABOUTME: and drives the same cli.run status path with no caller change.
package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/status"
)

// TestNativeRunnerSelectableThroughCLI proves the native runner backs the same
// Runner seam: cli.run drives it unchanged, forwarding argv/dir/env/stdin, and
// the native default table renders from a real workflow dir.
func TestNativeRunnerSelectableThroughCLI(t *testing.T) {
	// A compile-time check that NativeRunner is a status.Runner.
	var _ status.Runner = (*status.NativeRunner)(nil)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "---\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n")
	writeFile(t, filepath.Join(root, "001-task.md"), "---\nid: \"001\"\ntitle: A task\nstatus: backlog\nscore: \"0.5\"\nsource: roadmap\n---\n# A task\n")

	var stdout, stderr bytes.Buffer
	env := []string{"USER=pinned", "PATH=" + os.Getenv("PATH")}
	code := run(context.Background(), []string{"status", "--workflow-dir", root}, env, root, nil, &stdout, &stderr, &status.NativeRunner{}, nil)

	if code != 0 {
		t.Fatalf("native status exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "001-task") {
		t.Fatalf("native default table missing entity:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "ID") || !strings.Contains(stdout.String(), "SLUG") {
		t.Fatalf("native default table missing header:\n%s", stdout.String())
	}
}

// TestRunDefaultsToNativeRunner pins cli.Run's public default to the native
// runner with NO runner injection. The proof uses split-root behavior only the
// native runner produces: a workflow whose README carries `state:` keeps its
// entities in a state subdir with NO README of its own. The single-root vendor
// oracle, pointed at the definition dir, renders an empty table (it never
// composes the state subdir); the native runner reads stages from the
// definition README and entities from the state dir, so the entity appears.
// Seeing the state-dir entity through Run is therefore native-only behavior and
// locks the flip at the public entrypoint, not just the injectable run() core.
func TestRunDefaultsToNativeRunner(t *testing.T) {
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), "---\nid-style: slug\nstate: .spacedock-state\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n")
	state := filepath.Join(def, ".spacedock-state")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(state, "add-login.md"), "---\nstatus: backlog\n---\n")

	// Guard: the state subdir has no README of its own — the native runner must
	// compose the two roots itself, not read a symlinked twin.
	if _, err := os.Lstat(filepath.Join(state, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("state subdir must have no README.md, lstat err=%v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"status", "--workflow-dir", def}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run status exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "add-login") {
		t.Fatalf("Run default did not render the state-dir entity — VendorRunner cannot compose split-root, so the default is not native:\n%s", stdout.String())
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
