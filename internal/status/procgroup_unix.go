// ABOUTME: Unix process-group handling so context cancel kills the python
// ABOUTME: child and its git/gh grandchildren together, not just the child.
//go:build unix

package status

import (
	"context"
	"os/exec"
	"syscall"
)

// configureProcessGroup makes the child the leader of a new process group so the
// whole tree can be signaled at once.
func configureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// waitWithGroupKill waits for cmd, but if ctx is cancelled first it kills the
// child's whole process group (negative pid) so git/gh grandchildren die too.
func waitWithGroupKill(ctx context.Context, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			// Negative pid targets the process group led by the child.
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done
		return ctx.Err()
	case err := <-done:
		return err
	}
}
