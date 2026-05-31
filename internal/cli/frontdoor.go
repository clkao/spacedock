// ABOUTME: Host front doors (spacedock claude/codex) + init/doctor — the three
// ABOUTME: version-gate points wired through an injectable host-ops seam.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/clkao/spacedock-v1/internal/contract"
	"github.com/clkao/spacedock-v1/internal/safehouse"
)

// bootstrapPrompt is the fixed launch-and-go message appended as the last inner
// argv token so a fresh `spacedock claude` session starts the first officer
// rather than opening an idle agent. It is omitted when `--resume` is forwarded
// (a resume already carries its own session intent).
const bootstrapPrompt = "Begin as the Spacedock first officer: run your startup sequence and work the event loop."

// hostOps is the injectable seam the front-door, init, and doctor paths depend
// on. Production backs it with real `claude`/`codex` plugin commands and exec;
// tests back it with a fake that records interactions. Mirrors the status.Runner
// decoupling: the command logic never touches exec or the host CLI directly.
type hostOps interface {
	// ResolveManifest returns the installed plugin manifest path for host, or ""
	// when no plugin is installed (a distinct, non-error state). A non-nil error
	// means the host CLI itself failed.
	ResolveManifest(host string) (string, error)
	// Launch execs argv, replacing the current process on success (production) or
	// recording it (test). It returns only on failure to launch.
	Launch(argv []string) error
	// Install issues the host plugin commands to install/update the plugin from
	// source (optionally pinned to branch), returning combined output.
	Install(host, source, branch string) (string, error)
}

// devBranch is the pre-release dev branch woven into remedy/install hints. Empty
// is the default release path (no @branch suffix). Production wiring may set it
// from the environment; it stays a package var so tests pin "".
var devBranch = ""

// gateHost resolves the installed manifest for host and compares it against
// CONTRACT_VERSION. It returns whether launch is permitted. Only a Compatible
// verdict permits launch; everything else (a host-CLI error, no installed
// plugin, a resolved-but-missing manifest, a mismatch, or a malformed range) is
// NOT compatible — the front door's fail-fast job — so launch is denied with an
// actionable message. The gate inspects the VERDICT, not a doctor exit code:
// RunDoctor maps no-plugin-found to exit 0 (a non-fatal report), so a non-empty
// installPath to a missing manifest would otherwise slip through as "compatible".
func gateHost(ops hostOps, host string, stderr io.Writer) (ok bool) {
	manifestPath, err := ops.ResolveManifest(host)
	if err != nil {
		fmt.Fprintf(stderr,
			"Spacedock: could not resolve the installed %s plugin (%v). "+
				"Run `spacedock doctor` or `spacedock init --host %s`.\n", host, err, host)
		return false
	}
	if manifestPath == "" {
		fmt.Fprintf(stderr,
			"Spacedock: no installed %s plugin found. "+
				"Run `spacedock init --host %s` (or `spacedock claude --skip-contract-check` to bootstrap).\n", host, host)
		return false
	}
	res := contract.ManifestVerdict(manifestPath, host, devBranch)
	if res.Verdict == contract.Compatible {
		return true
	}
	if res.Verdict == contract.NoPluginFound {
		fmt.Fprintf(stderr,
			"Spacedock: the installed %s plugin reported a manifest path that does not exist (%s). "+
				"Run `spacedock init --host %s` (or `spacedock claude --skip-contract-check` to bootstrap).\n",
			host, manifestPath, host)
		return false
	}
	fmt.Fprintln(stderr, res.Message)
	return false
}

// runClaude is the `spacedock claude` front door: version-gate (fail fast), then
// launch the first officer. When `dir` carries a `.safehouse` profile the launch
// is interposed through `safehouse --trust-workdir-config -- claude
// --dangerously-skip-permissions …`; otherwise it is plain `claude --agent
// spacedock:first-officer …` (no skip-permissions in an unsandboxed launch). A
// fixed bootstrap prompt is appended last unless `--resume` is forwarded.
// `--skip-contract-check` bypasses the gate for first-install bootstrap.
// `lookPath` resolves the safehouse binary (default exec.LookPath; injected so
// tests pin not-found).
func runClaude(ctx context.Context, args []string, dir string, ops hostOps, lookPath func(string) (string, error), stdout, stderr io.Writer) int {
	passthrough, skipCheck := splitFrontDoorArgs(args)
	if !skipCheck {
		if !gateHost(ops, "claude", stderr) {
			return 1
		}
	}

	resume := containsResume(passthrough)
	inner := []string{"claude"}
	if safehouse.Present(dir) {
		inner = append(inner, "--dangerously-skip-permissions")
	}
	inner = append(inner, "--agent", "spacedock:first-officer")
	inner = append(inner, passthrough...)
	if !resume {
		inner = append(inner, bootstrapPrompt)
	}

	argv := inner
	if safehouse.Present(dir) {
		if ok, hint := safehouse.Available(lookPath); !ok {
			fmt.Fprintln(stderr, hint)
			return 1
		}
		argv = safehouse.Wrap(inner, nil)
	}

	if err := ops.Launch(argv); err != nil {
		fmt.Fprintf(stderr, "spacedock claude: launch failed: %v\n", err)
		return 1
	}
	return 0
}

// containsResume reports whether the operator forwarded any of claude's
// session-resume forms (which carry their own session intent, so the bootstrap
// prompt is suppressed): `--resume`, `--resume=<id>`, `-r`, `--continue`, `-c`.
func containsResume(args []string) bool {
	for _, a := range args {
		switch a {
		case "--resume", "-r", "--continue", "-c":
			return true
		}
		if strings.HasPrefix(a, "--resume=") {
			return true
		}
	}
	return false
}

// runCodex is the `spacedock codex` front door: version-gate (fail fast), then
// print the documented prose. Codex has no --agent/named-subagent launch flag
// (spike-confirmed), so this is NOT an agent-launch wrapper — it tells the
// operator to use the spacedock:first-officer skill in Codex.
func runCodex(ctx context.Context, args []string, ops hostOps, stdout, stderr io.Writer) int {
	_, skipCheck := splitFrontDoorArgs(args)
	if !skipCheck {
		if !gateHost(ops, "codex", stderr) {
			return 1
		}
	}
	fmt.Fprint(stdout,
		"Codex has no agent-launch flag, so spacedock cannot start the first officer for you.\n"+
			"In your Codex session, use the spacedock:first-officer skill in this directory to run the workflow.\n")
	return 0
}

// splitFrontDoorArgs separates the `--skip-contract-check` override and an
// optional `--` separator from the host passthrough args. The override flag is
// consumed (never forwarded to the host); everything else passes through.
func splitFrontDoorArgs(args []string) (passthrough []string, skipCheck bool) {
	for _, a := range args {
		switch a {
		case "--skip-contract-check":
			skipCheck = true
		case "--":
			// Argument separator: drop it, keep the rest as passthrough.
		default:
			passthrough = append(passthrough, a)
		}
	}
	return passthrough, skipCheck
}
