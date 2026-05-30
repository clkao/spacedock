// ABOUTME: Defines the narrow Runner seam for spacedock status — the whole
// ABOUTME: contract Stage 5 native Go must satisfy: (args,dir,env,stdin)->(out,err,exit).
package status

import (
	"context"
	"io"
)

// Request carries every observable input the status runner depends on. Args is
// forwarded verbatim (including --workflow-dir and unrecognized tokens). Dir is
// the working directory, load-bearing for --discover without --root. Env is the
// explicit environment; load-bearing variables include USER/USERNAME,
// SPACEDOCK_ID_ACTOR/SPACEDOCK_ID_CONTEXT/SPACEDOCK_TEST_SD_B32_TIMESTAMP, HOME,
// and PATH. Stdin is carried forward for Stage 5's native --new < STDIN even
// though today's vendored read/mutation flags do not consume it.
type Request struct {
	Args   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Runner executes a status request and returns the process exit code. The exit
// domain is {0 success, 1 error}; the runner never injects an exit code of its
// own. A non-nil err signals the runner could not run at all (e.g. interpreter
// missing), distinct from a non-zero exitCode the script itself produced.
type Runner interface {
	Run(ctx context.Context, req Request) (exitCode int, err error)
}
