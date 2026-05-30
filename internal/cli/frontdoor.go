// ABOUTME: Host front doors (spacedock claude/codex) + init/doctor — the three
// ABOUTME: version-gate points wired through an injectable host-ops seam.
package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/clkao/spacedock-v1/internal/contract"
)

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
// CONTRACT_VERSION. It returns the verdict result and a launch-permitted bool.
// An unresolvable manifest (no plugin found, or a host-CLI error) is NOT
// compatible — the front door's fail-fast job — so launch is denied with an
// actionable message. The doctor path treats no-plugin-found as a non-fatal
// report; that policy lives in RunDoctor, not here.
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
	// The manifest resolved, so doctor's no-plugin-found branch cannot fire here;
	// a non-zero exit means a real mismatch and gateStderr holds the pinned remedy.
	var out, gateStderr bytes.Buffer
	code := contract.RunDoctor(manifestPath, host, devBranch, &out, &gateStderr)
	if code != 0 {
		io.WriteString(stderr, gateStderr.String())
		return false
	}
	return true
}

// runClaude is the `spacedock claude` front door: version-gate (fail fast), then
// launch `claude --agent spacedock:first-officer` with the operator's passthrough
// args. `--skip-contract-check` bypasses the gate for first-install bootstrap.
func runClaude(ctx context.Context, args []string, ops hostOps, stdout, stderr io.Writer) int {
	passthrough, skipCheck := splitFrontDoorArgs(args)
	if !skipCheck {
		if !gateHost(ops, "claude", stderr) {
			return 1
		}
	}
	argv := append([]string{"claude", "--agent", "spacedock:first-officer"}, passthrough...)
	if err := ops.Launch(argv); err != nil {
		fmt.Fprintf(stderr, "spacedock claude: launch failed: %v\n", err)
		return 1
	}
	return 0
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
