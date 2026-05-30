// ABOUTME: VendorRunner-level tests — compile-time seam satisfaction, loud
// ABOUTME: failure on a missing interpreter, and process-group cancel cleanup.
package status

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Compile-time assertion that the vendored-exec runner satisfies the seam Stage
// 5's native runner must also satisfy.
var _ Runner = (*VendorRunner)(nil)

// TestVendorRunnerSatisfiesSeam is the runtime companion to the compile-time
// assertion: a VendorRunner is usable wherever a Runner is expected.
func TestVendorRunnerSatisfiesSeam(t *testing.T) {
	var r Runner = &VendorRunner{}
	if r == nil {
		t.Fatal("VendorRunner does not satisfy Runner")
	}
}

// TestMissingInterpreterFailsLoudly locks the risk-area contract: when the
// configured interpreter is absent, the runner fails loudly — non-nil err, a
// diagnostic naming the interpreter — rather than silently mis-reporting status.
func TestMissingInterpreterFailsLoudly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	runner := &VendorRunner{Python: "python3-does-not-exist-spacedock"}
	code, runErr := runner.Run(context.Background(), Request{
		Args:   []string{"--workflow-dir", root},
		Dir:    root,
		Env:    pinnedEnv(t),
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if runErr == nil {
		t.Fatalf("expected a run error for a missing interpreter, got nil (code=%d, stdout=%q)", code, stdout.String())
	}
	if !strings.Contains(runErr.Error(), "python3-does-not-exist-spacedock") {
		t.Fatalf("error %q should name the missing interpreter", runErr.Error())
	}
}

// TestCancelKillsGrandchildren locks the cancellation contract: cancelling mid
// run kills the whole process group so a spawned grandchild does not outlive
// the cancel. The script is replaced by a tiny python program that spawns a
// long-lived grandchild (sleep) writing its pid, then itself sleeps; on cancel
// the runner must reap the group so the grandchild is gone.
func TestCancelKillsGrandchildren(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	// A program that forks a grandchild (a detached `sleep`) recording its pid,
	// then blocks. exec.CommandContext cancel kills only the python child; the
	// process-group signal must take the grandchild too.
	prog := `
import os, subprocess, sys, time
p = subprocess.Popen(["sleep", "60"])
open(sys.argv[1], "w").write(str(p.pid))
time.sleep(60)
`
	dir := t.TempDir()
	progPath := filepath.Join(dir, "spawn.py")
	if err := os.WriteFile(progPath, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "python3", progPath, pidFile)
	configureProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Wait for the grandchild pid to be recorded.
	var gpid int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(pidFile); err == nil && len(b) > 0 {
			if _, err := fmt.Sscan(strings.TrimSpace(string(b)), &gpid); err == nil && gpid > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if gpid == 0 {
		cancel()
		cmd.Wait()
		t.Fatal("grandchild pid was never recorded")
	}

	cancel()
	_ = waitWithGroupKill(ctx, cmd)

	// After the group kill, the grandchild must be gone. Poll briefly since
	// reaping is asynchronous.
	gone := false
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(gpid) {
			gone = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !gone {
		// Best-effort cleanup so we don't leak a sleep.
		_ = killPid(gpid)
		t.Fatalf("grandchild pid %d survived context cancel — process group was not signaled", gpid)
	}
}
