// ABOUTME: Native-runner parity harness — runNative drives NativeRunner with the
// ABOUTME: same (args,dir,env,stdin) the oracle/launcher harness uses.
package status

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// runNative runs the native runner with the given args/dir/env (and optional
// stdin) and returns stdout, stderr, exit code. It is the native analog of
// runLauncher / runOracle.
func runNative(t *testing.T, dir string, env []string, args ...string) (string, string, int) {
	t.Helper()
	return runNativeStdin(t, dir, env, nil, args...)
}

func runNativeStdin(t *testing.T, dir string, env []string, stdin io.Reader, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	// The Claude team-state probe matches the Python oracle's ~/.claude/teams read,
	// so boot TEAM_STATE parity holds byte-for-byte under the pinned (empty) HOME.
	runner := &NativeRunner{TeamStateProbe: claudeteam.Probe}
	code, err := runner.Run(context.Background(), Request{
		Args:   args,
		Dir:    dir,
		Env:    env,
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("native runner error: %v (stderr=%q)", err, stderr.String())
	}
	return stdout.String(), stderr.String(), code
}

func reader(s string) io.Reader { return strings.NewReader(s) }
