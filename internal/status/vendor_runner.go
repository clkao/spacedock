// ABOUTME: VendorRunner backs the Runner seam by exec'ing the embedded Python
// ABOUTME: status oracle under python3, forwarding argv/dir/env/stdin verbatim.
package status

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

//go:embed vendor/status
var vendorScript []byte

// VendorRunner runs the vendored Python status script under python3. It writes
// the embedded script to a temp file per call and execs `python3 <script> <args>`,
// streaming stdout/stderr and returning the script's exit code unmodified.
type VendorRunner struct {
	// Python is the interpreter to exec. Empty means "python3" on PATH.
	Python string
}

var _ Runner = (*VendorRunner)(nil)

// Run materializes the embedded script and execs it, forwarding req verbatim.
// The returned err is non-nil only when the script could not be run at all
// (interpreter missing, materialization failure); a non-zero exitCode from the
// script itself is returned with a nil err.
func (r *VendorRunner) Run(ctx context.Context, req Request) (int, error) {
	scriptPath, cleanup, err := materializeScript()
	if err != nil {
		return -1, err
	}
	defer cleanup()

	python := r.Python
	if python == "" {
		python = "python3"
	}

	args := append([]string{scriptPath}, req.Args...)
	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Dir = req.Dir
	cmd.Env = req.Env
	cmd.Stdin = req.Stdin
	cmd.Stdout = req.Stdout
	cmd.Stderr = req.Stderr

	// Start the interpreter in its own process group so that on context cancel
	// the whole tree dies — the script spawns git/gh grandchildren that a naive
	// cancel of the python child alone would orphan.
	configureProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("spacedock status: cannot start %s: %w", python, err)
	}

	// On cancel, signal the whole group rather than just the child.
	waitErr := waitWithGroupKill(ctx, cmd)

	if waitErr == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return -1, fmt.Errorf("spacedock status: %s failed to run: %w", python, waitErr)
}

// materializeScript writes the embedded script to a temp file and returns its
// path plus a cleanup func.
func materializeScript() (string, func(), error) {
	f, err := os.CreateTemp("", "spacedock-status-*.py")
	if err != nil {
		return "", func() {}, fmt.Errorf("spacedock status: cannot create temp script: %w", err)
	}
	path := f.Name()
	cleanup := func() { os.Remove(path) }
	if _, err := f.Write(vendorScript); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("spacedock status: cannot write temp script: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("spacedock status: cannot close temp script: %w", err)
	}
	return path, cleanup, nil
}
