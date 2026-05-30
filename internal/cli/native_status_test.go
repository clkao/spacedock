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

	"github.com/clkao/spacedock-v1/internal/status"
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
	code := run(context.Background(), []string{"status", "--workflow-dir", root}, env, root, nil, &stdout, &stderr, &status.NativeRunner{})

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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
