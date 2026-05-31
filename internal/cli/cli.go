// ABOUTME: Command routing, usage text, and exit-code behavior for spacedock.
// ABOUTME: status forwards argv verbatim to the status.Runner seam.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spacedock-dev/spacedock/internal/contract"
	"github.com/spacedock-dev/spacedock/internal/dispatch"
	"github.com/spacedock-dev/spacedock/internal/status"
)

// Version is the single source of truth for the binary version. It is stamped by
// the release pipeline via -ldflags "-X
// github.com/spacedock-dev/spacedock/internal/cli.Version=$(git describe --tags --always)".
// It is a var (not a const) because the linker can only write package-level vars;
// a const is silently ignored by -X. The default is the current release version,
// overwritten by the git-describe tag on a stamped release build.
var Version = "0.19.0"

// Run is the process entry point. status is routed to the native Go runner,
// which composes the definition root (README) and the entity root (the README's
// state: dir) itself; all other commands are handled directly. The vendored
// Python runner stays selectable through the injectable run() core.
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
		fmt.Fprintf(stdout, "spacedock %s (contract %d)\n", Version, contract.CONTRACT_VERSION)
		return 0
	case "status":
		return runStatus(ctx, args[1:], env, dir, stdin, stdout, stderr, runner)
	case "dispatch":
		return dispatch.Run(args[1:], stdin, stdout, stderr)
	case "claude":
		applyDevBranchOverride(env)
		return runClaude(ctx, args[1:], dir, execHost{}, exec.LookPath, stdout, stderr)
	case "codex":
		applyDevBranchOverride(env)
		return runCodex(ctx, args[1:], dir, execHost{}, exec.LookPath, stdout, stderr)
	case "init":
		applyDevBranchOverride(env)
		return runInit(ctx, args[1:], execHost{}, stdout, stderr)
	case "doctor":
		applyDevBranchOverride(env)
		return runDoctor(ctx, args[1:], execHost{}, stdout, stderr)
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

// applyDevBranchOverride lets SPACEDOCK_DEV_BRANCH override the pinned devBranch
// default (and the linker stamp). An UNSET env var leaves the default in place —
// the released binary keeps targeting `@next` — while an explicit value (including
// empty, to force the no-ref release path) wins.
func applyDevBranchOverride(env []string) {
	for _, kv := range env {
		if strings.HasPrefix(kv, "SPACEDOCK_DEV_BRANCH=") {
			devBranch = strings.TrimPrefix(kv, "SPACEDOCK_DEV_BRANCH=")
			return
		}
	}
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
  spacedock claude [args...]                          version-gate then launch claude --agent spacedock:first-officer
  spacedock codex [args...]                           version-gate then launch codex with the spacedock:first-officer skill
  spacedock init [--host claude|codex] [--check]      install the per-host plugin, then run doctor
  spacedock doctor [--host claude|codex] [--plugin-manifest PATH]
  spacedock status [args...]
  spacedock dispatch build --workflow-dir DIR
  spacedock dispatch show-stage-def --workflow-dir DIR --stage STAGE
  spacedock --version
  spacedock --help

claude/codex are the host front doors: they version-gate against the installed
plugin's requires-contract and fail fast on a mismatch before launching. Append a
task with a -- fence (spacedock claude -- "the task"); host flags go before the
fence and forward verbatim. Sandbox knobs: --safehouse forces the safehouse wrap,
--safehouse-enable=KEY[,KEY], --safehouse-add-dirs=PATH, --safehouse-add-dirs-ro=PATH.
A --plugin-dir launch loads the local checkout and relaxes the contract gate.
init installs the per-host plugin via the host plugin mechanism (no skill-file copies).
doctor reports the compatibility verdict against the binary's CONTRACT_VERSION.
status forwards its arguments to the workflow status command.
dispatch assembles ensign dispatch artifacts (build, show-stage-def).
`)
}
