// ABOUTME: dispatch command router — build + show-stage-def, the native surface
// ABOUTME: of the self-hosted dispatch path; everything else is a usage error.
package dispatch

import (
	"fmt"
	"io"
	"os"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// Run routes a `spacedock dispatch <subcommand> [flags]` invocation. build and
// show-stage-def are the host-neutral surface (assembled here); context-budget,
// list-standing, show-standing, and spawn-standing are the Claude-coupled surface
// (their ~/.claude and standing-mod reads live in internal/claudeteam). An unknown
// subcommand fails with exit 2 and a usage diagnostic on stderr. probe is the
// host-supplied team-state probe gating the bare-mode advisory (nil on a non-Claude
// host → no advisory).
func Run(probe claudeteam.TeamStateProbe, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "build":
		workflowDir, code := requireFlag(args[1:], "--workflow-dir", stderr)
		if code != 0 {
			return code
		}
		return runBuild(probe, workflowDir, stdin, stdout, stderr)
	case "show-stage-def":
		workflowDir, stage, code := requireStageFlags(args[1:], stderr)
		if code != 0 {
			return code
		}
		return runShowStageDef(workflowDir, stage, stdout, stderr)
	case "context-budget":
		name, code := requireSubcommandFlag(args[1:], "context-budget", "--name", stderr)
		if code != 0 {
			return code
		}
		return claudeteam.ContextBudget(os.Getenv("HOME"), name, stdout, stderr)
	case "list-standing":
		workflowDir, code := requireSubcommandFlag(args[1:], "list-standing", "--workflow-dir", stderr)
		if code != 0 {
			return code
		}
		return runListStanding(workflowDir, stdout, stderr)
	case "show-standing":
		workflowDir, code := requireSubcommandFlag(args[1:], "show-standing", "--workflow-dir", stderr)
		if code != 0 {
			return code
		}
		return runShowStanding(workflowDir, stdout, stderr)
	case "spawn-standing":
		flags := parseFlags(args[1:], map[string]bool{"--mod": true, "--team": true})
		mod, okMod := flags["--mod"]
		team, okTeam := flags["--team"]
		if !okMod || !okTeam {
			fmt.Fprintln(stderr, "error: dispatch spawn-standing requires --mod and --team")
			return 2
		}
		return runSpawnStanding(os.Getenv("HOME"), mod, team, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown dispatch subcommand: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

// requireFlag returns the value of a single required `name value` flag. A
// missing flag, missing value, or trailing junk is a usage error (exit 2).
func requireFlag(args []string, name string, stderr io.Writer) (string, int) {
	val, ok := parseFlags(args, map[string]bool{name: true})[name]
	if !ok {
		fmt.Fprintf(stderr, "error: dispatch build requires %s\n", name)
		return "", 2
	}
	return val, 0
}

// requireSubcommandFlag returns the value of a single required flag for a
// claude-coupled subcommand, with a usage error (exit 2) naming the subcommand
// and flag when missing. The diagnostic is the native CLI's own ergonomic — the
// command-logic loud-failures (not this argument-parse error) are what the parity
// harness byte-compares.
func requireSubcommandFlag(args []string, subcommand, name string, stderr io.Writer) (string, int) {
	val, ok := parseFlags(args, map[string]bool{name: true})[name]
	if !ok {
		fmt.Fprintf(stderr, "error: dispatch %s requires %s\n", subcommand, name)
		return "", 2
	}
	return val, 0
}

// requireStageFlags returns the --workflow-dir and --stage values show-stage-def
// requires, with a usage error (exit 2) when either is missing.
func requireStageFlags(args []string, stderr io.Writer) (string, string, int) {
	flags := parseFlags(args, map[string]bool{"--workflow-dir": true, "--stage": true})
	wd, okWD := flags["--workflow-dir"]
	stage, okStage := flags["--stage"]
	if !okWD || !okStage {
		fmt.Fprintln(stderr, "error: dispatch show-stage-def requires --workflow-dir and --stage")
		return "", "", 2
	}
	return wd, stage, 0
}

// parseFlags reads `--flag value` pairs from args for the flags in want. Unknown
// flags and bare arguments are ignored — the required-flag checks above surface
// the actionable error, matching the oracle's argparse required-flag behavior.
func parseFlags(args []string, want map[string]bool) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if want[a] && i+1 < len(args) {
			out[a] = args[i+1]
			i++
			continue
		}
		if eq := indexByte(a, '='); eq > 0 && want[a[:eq]] {
			out[a[:eq]] = a[eq+1:]
		}
	}
	return out
}

// indexByte returns the index of the first b in s, or -1.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `spacedock dispatch assembles ensign dispatch artifacts.

Usage:
  spacedock dispatch build --workflow-dir DIR        (stdin JSON -> stdout JSON)
  spacedock dispatch show-stage-def --workflow-dir DIR --stage STAGE
`)
}
