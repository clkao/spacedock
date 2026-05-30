// ABOUTME: Command routing, usage text, and exit-code behavior for spacedock.
// ABOUTME: status forwards argv verbatim to the status.Runner seam.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/clkao/spacedock-v1/internal/status"
)

const Version = "0.1.0-dev"

// Run is the process entry point. status is routed to the native Go runner;
// all other commands are handled directly.
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	return run(context.Background(), args, os.Environ(), cwd(), os.Stdin, stdout, stderr, &status.NativeRunner{})
}

// run is the injectable core. It depends only on the status.Runner interface,
// never on the vendored script or any exec detail, so AC-4's fake-runner test
// can drive the status path with pinned env/cwd.
func run(ctx context.Context, args []string, env []string, dir string, stdin io.Reader, stdout io.Writer, stderr io.Writer, runner status.Runner) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0
	case "--version", "version":
		fmt.Fprintf(stdout, "spacedock %s\n", Version)
		return 0
	case "status":
		return runStatus(ctx, args[1:], env, dir, stdin, stdout, stderr, runner)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

// runStatus forwards the post-"status" argv verbatim to the runner and returns
// its exit code unmodified. The CLI adds nothing to and removes nothing from the
// runner's contract: it does not parse, reformat, interpret, or strip flags. If
// the runner itself cannot run (interpreter missing), surface a diagnostic and
// fail loudly with exit 1 — matching the script's own error exit code.
func runStatus(ctx context.Context, args []string, env []string, dir string, stdin io.Reader, stdout io.Writer, stderr io.Writer, runner status.Runner) int {
	code, err := runner.Run(ctx, status.Request{
		Args:   args,
		Dir:    dir,
		Env:    env,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return code
}

// cwd returns the working directory, falling back to "" so a getwd failure does
// not abort the command — the runner derives a scan root from --workflow-dir.
func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `spacedock is the Spacedock v1 launcher.

Usage:
  spacedock status [args...]
  spacedock --version
  spacedock --help

status forwards its arguments to the workflow status command.
`)
}
