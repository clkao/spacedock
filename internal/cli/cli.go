// ABOUTME: cobra command tree, grouped help, and exit-code behavior for spacedock.
// ABOUTME: status/dispatch forward argv verbatim; claude/codex use the Option-2 grammar.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

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

// tagline is the one-line product description rendered as the first help line.
const tagline = "spacedock — agentic workflow launcher"

// Run is the process entry point. status is routed to the native Go runner,
// which composes the definition root (README) and the entity root (the README's
// state: dir) itself; all other commands are handled directly. The vendored
// Python runner stays selectable through the injectable run() core.
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	return run(context.Background(), args, os.Environ(), cwd(), os.Stdin, stdout, stderr, &status.NativeRunner{})
}

// run is the injectable core. It depends only on the status.Runner interface,
// never on the vendored script or any exec detail, so the fake-runner tests can
// drive the status path with pinned env/cwd. cobra is wired INSIDE run so the
// package's public surface (Run) and the exit-code contract are unchanged: the
// command tree captures env/dir/stdin/stdout/stderr/runner in its RunE closures.
func run(ctx context.Context, args []string, env []string, dir string, stdin io.Reader, stdout io.Writer, stderr io.Writer, runner status.Runner) int {
	root := newRootCommand(ctx, env, dir, stdin, stdout, stderr, runner)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return exitCodeFor(err)
	}
	return 0
}

// exitCodeError carries an explicit process exit code out of a RunE so the
// command tree can preserve the hand-rolled router's exit-code contract (status
// exit-1 surfacing, the front-door fail-fast exit 1) through cobra's single error
// return. cobra's own command-resolution errors (unknown command, unknown flag)
// carry no code and map to exit 2.
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return fmt.Sprintf("exit %d", e.code) }

// exitCodeFor maps an Execute error to a process exit code. An explicit
// exitCodeError carries its own code (RunE already wrote any diagnostic); every
// other error is a cobra command/flag-resolution failure, which exits 2 to match
// the hand-rolled router's unknown-command contract (TestUnknownCommand).
func exitCodeFor(err error) int {
	var ec exitCodeError
	if errors.As(err, &ec) {
		return ec.code
	}
	return 2
}

// newRootCommand assembles the cobra tree. The root owns the grouped jargon-free
// help (AC-1) and the explicit `--version` handler with the `(contract N)` token
// (AC-5). SilenceErrors/SilenceUsage hand all output and exit-code control to this
// package: cobra never prints its own error or usage, so the unknown-command path
// emits the pinned message and exits 2 (the root RunE below), and the help is
// rendered solely by printHelp.
func newRootCommand(ctx context.Context, env []string, dir string, stdin io.Reader, stdout, stderr io.Writer, runner status.Runner) *cobra.Command {
	versionFlag := false

	root := &cobra.Command{
		Use:               "spacedock",
		SilenceErrors:     true,
		SilenceUsage:      true,
		Args:              cobra.ArbitraryArgs,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(cmd *cobra.Command, args []string) error {
			if versionFlag {
				printVersion(stdout)
				return nil
			}
			// No subcommand and no recognized flag: an unknown command token
			// (e.g. `spacedock bogus`) exits 2 with the pinned message; bare
			// `spacedock` renders the grouped help.
			if len(args) > 0 {
				return unknownCommand(args[0], stderr)
			}
			printHelp(stdout)
			return nil
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.Flags().BoolVar(&versionFlag, "version", false, "Print the spacedock version and contract level")
	root.SetHelpFunc(func(*cobra.Command, []string) { printHelp(stdout) })

	root.AddGroup(
		&cobra.Group{ID: "launch", Title: "Launch"},
		&cobra.Group{ID: "setup", Title: "Setup"},
		&cobra.Group{ID: "workflow", Title: "Workflow"},
	)

	root.AddCommand(
		newClaudeCommand(ctx, env, dir, stdout, stderr),
		newCodexCommand(ctx, env, dir, stdout, stderr),
		newInstallCommand(ctx, env, stdout, stderr),
		newDoctorCommand(ctx, env, stdout, stderr),
		newStatusCommand(ctx, env, dir, stdin, stdout, stderr, runner),
		newDispatchCommand(stdin, stdout, stderr),
	)
	return root
}

// newClaudeCommand wires `spacedock claude`. Flag parsing is disabled at the cobra
// layer so the post-subcommand argv reaches runClaude verbatim — runClaude owns the
// Option-2 grammar via parseFrontDoorArgs (ArgsLenAtDash). The flags are declared
// only so `--help` renders them (AC-4); `-h`/`--help` is intercepted here because
// DisableFlagParsing routes it to RunE rather than cobra's own help.
func newClaudeCommand(ctx context.Context, env []string, dir string, stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "claude [task] [-- claude-flags]",
		Short:              "Start Claude Code as your Spacedock first officer",
		GroupID:            "launch",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			applyDevBranchOverride(env)
			if code := runClaude(ctx, args, dir, execHost{}, exec.LookPath, stdout, stderr); code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}
	setFrontDoorHelp(cmd, "claude", stdout)
	return cmd
}

// newCodexCommand mirrors newClaudeCommand for `spacedock codex`.
func newCodexCommand(ctx context.Context, env []string, dir string, stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "codex [task] [-- codex-flags]",
		Short:              "Start Codex as your Spacedock first officer",
		GroupID:            "launch",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			applyDevBranchOverride(env)
			if code := runCodex(ctx, args, dir, execHost{}, exec.LookPath, stdout, stderr); code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}
	setFrontDoorHelp(cmd, "codex", stdout)
	return cmd
}

// newInstallCommand wires `spacedock install` (the renamed `init`). Behavior is
// unchanged from init: install the per-host plugin then run doctor (claude), or
// emit the documented codex add prose. DisableFlagParsing keeps the post-subcommand
// argv verbatim for the existing hand-parsed runInit (so `--host`/`--check` parse
// exactly as before); `-h`/`--help` is intercepted here.
func newInstallCommand(ctx context.Context, env []string, stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "install [--host claude|codex] [--check]",
		Short:              "Install the Spacedock plugin for a host, then check it",
		GroupID:            "setup",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			applyDevBranchOverride(env)
			if code := runInit(ctx, args, execHost{}, stdout, stderr); code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}
	cmd.Flags().String("host", "claude", "Host to install the plugin for (claude or codex)")
	cmd.Flags().Bool("check", false, "Run the compatibility report without installing")
	setSetupHelp(cmd, stdout, `
Examples:
  spacedock install
  spacedock install --host codex
  spacedock install --check
`)
	return cmd
}

// newDoctorCommand wires `spacedock doctor` with its existing hand-parsed
// `--host`/`--plugin-manifest` handling preserved verbatim.
func newDoctorCommand(ctx context.Context, env []string, stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "doctor [--host claude|codex]",
		Short:              "Check the installed plugin and this binary are compatible",
		GroupID:            "setup",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			applyDevBranchOverride(env)
			if code := runDoctor(ctx, args, execHost{}, stdout, stderr); code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}
	cmd.Flags().String("host", "claude", "Host to check (claude or codex)")
	cmd.Flags().String("plugin-manifest", "", "Read this manifest directly instead of resolving the installed plugin")
	setSetupHelp(cmd, stdout, `
Examples:
  spacedock doctor
  spacedock doctor --host codex
`)
	return cmd
}

// newStatusCommand reparents `spacedock status` under cobra with flag parsing
// disabled, so its post-subcommand argv forwards VERBATIM to runStatus exactly as
// the hand-rolled router did — cobra never consumes, reorders, or validates a
// status flag (AC-5).
func newStatusCommand(ctx context.Context, env []string, dir string, stdin io.Reader, stdout, stderr io.Writer, runner status.Runner) *cobra.Command {
	return &cobra.Command{
		Use:                "status [args]",
		Short:              "Show or update workflow state",
		GroupID:            "workflow",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := runStatus(ctx, args, env, dir, stdin, stdout, stderr, runner); code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}
}

// newDispatchCommand reparents `spacedock dispatch` under cobra with flag parsing
// disabled, forwarding its post-subcommand argv verbatim to dispatch.Run (AC-5).
func newDispatchCommand(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:                "dispatch build | show-stage-def",
		Short:              "Build worker dispatch artifacts",
		GroupID:            "workflow",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if code := dispatch.Run(args, stdin, stdout, stderr); code != 0 {
				return exitCodeError{code}
			}
			return nil
		},
	}
}

// wantsHelp reports whether the operator asked for command help. Commands with
// DisableFlagParsing receive `-h`/`--help` as ordinary args, so each RunE checks
// for it before doing work. Only a leading help token counts: a `--help` after
// `--` is host passthrough, not a request for spacedock's help.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// unknownCommand writes the pinned unknown-command diagnostic plus the grouped
// help to stderr and returns the exit-2 carrier.
func unknownCommand(name string, stderr io.Writer) error {
	fmt.Fprintf(stderr, "unknown command: %s\n", name)
	printHelp(stderr)
	return exitCodeError{2}
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

// printVersion emits the version line with the contract token. The token is
// load-bearing: the FO/ensign skills read `(contract N)` from `spacedock
// --version`, so cobra's auto version-flag (a bare version string, plus a command
// row in help) is deliberately NOT used.
func printVersion(w io.Writer) {
	fmt.Fprintf(w, "spacedock %s (contract %d)\n", Version, contract.CONTRACT_VERSION)
}
