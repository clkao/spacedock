// ABOUTME: spacedock init/doctor command paths — install the per-host plugin via
// ABOUTME: the host plugin mechanism (claude) or emit the documented add prose (codex).
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/clkao/spacedock-v1/internal/contract"
)

// marketplaceSource is the marketplace add source for the Spacedock plugin. The
// default release path resolves the published marketplace repo; a pre-release
// dev branch is pinned via devBranch in the emitted/issued commands.
const marketplaceSource = "clkao/spacedock"

// runInit installs/updates the per-host plugin (claude) or emits the documented
// add command pair (codex), then runs doctor. `--check` runs the report without
// installing. No skill-file copies — install goes through the host plugin
// mechanism, which is what makes Skill()/--agent spacedock:first-officer resolve.
func runInit(ctx context.Context, args []string, ops hostOps, stdout, stderr io.Writer) int {
	host, check, code := parseInitArgs(args, stderr)
	if code != 0 {
		return code
	}

	switch host {
	case "claude":
		if !check {
			out, err := ops.Install(host, marketplaceSource, devBranch)
			if err != nil {
				fmt.Fprintf(stderr, "spacedock init: host install failed: %v\n", err)
				return 1
			}
			if out != "" {
				fmt.Fprintln(stdout, out)
			}
		}
		return runDoctor(ctx, []string{"--host", "claude"}, ops, stdout, stderr)
	case "codex":
		// Codex install is documented prose: the host install verb is `add`
		// (NOT `install`), and the marketplace add accepts the branch via --ref.
		printCodexInstallProse(stdout)
		if !check {
			return 0
		}
		return runDoctor(ctx, []string{"--host", "codex"}, ops, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "spacedock init: unknown host %q (want claude or codex)\n", host)
		return 2
	}
}

// printCodexInstallProse emits the documented Codex install command pair. The
// dev branch, when set, is pinned via --ref on the marketplace add.
func printCodexInstallProse(stdout io.Writer) {
	addCmd := "codex plugin marketplace add " + marketplaceSource
	if devBranch != "" {
		addCmd += " --ref " + devBranch
	}
	fmt.Fprintf(stdout,
		"Codex has no programmatic plugin install from spacedock. Run these in your shell:\n"+
			"  %s\n"+
			"  codex plugin add spacedock@spacedock\n"+
			"Then use the spacedock:first-officer skill in your Codex session.\n", addCmd)
}

// runDoctor is the `spacedock doctor` command path. With `--plugin-manifest PATH`
// it reads that manifest directly (used by fixtures and operators); otherwise it
// resolves the installed manifest via the host-ops seam. A resolved compatible
// manifest exits 0; a mismatch exits 1; no installed plugin is a non-fatal
// report (exit 0).
func runDoctor(ctx context.Context, args []string, ops hostOps, stdout, stderr io.Writer) int {
	manifestPath, host, code := parseDoctorArgs(args, stderr)
	if code != 0 {
		return code
	}

	if manifestPath != "" {
		return contract.RunDoctor(manifestPath, host, devBranch, stdout, stderr)
	}

	resolved, err := ops.ResolveManifest(host)
	if err != nil {
		fmt.Fprintf(stderr, "spacedock doctor: could not resolve the installed %s plugin: %v\n", host, err)
		return 1
	}
	// An empty resolved path is the no-plugin-found report; RunDoctor renders it
	// from a non-existent path as a non-fatal report.
	return contract.RunDoctor(resolved, host, devBranch, stdout, stderr)
}

// parseInitArgs reads `--host claude|codex` (default claude) and `--check`. A
// missing --host value is a usage error (exit 2).
func parseInitArgs(args []string, stderr io.Writer) (host string, check bool, code int) {
	host = "claude"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "spacedock init: --host requires a value (claude or codex)")
				return "", false, 2
			}
			host = args[i+1]
			i++
		case "--check":
			check = true
		default:
			fmt.Fprintf(stderr, "spacedock init: unknown argument %q\n", args[i])
			return "", false, 2
		}
	}
	return host, check, 0
}

// parseDoctorArgs reads `--plugin-manifest PATH` (optional explicit manifest)
// and `--host claude|codex` (default claude).
func parseDoctorArgs(args []string, stderr io.Writer) (manifestPath, host string, code int) {
	host = "claude"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--plugin-manifest":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "spacedock doctor: --plugin-manifest requires a path")
				return "", "", 2
			}
			manifestPath = args[i+1]
			i++
		case "--host":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "spacedock doctor: --host requires a value (claude or codex)")
				return "", "", 2
			}
			host = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "spacedock doctor: unknown argument %q\n", args[i])
			return "", "", 2
		}
	}
	return manifestPath, host, 0
}
