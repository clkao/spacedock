// ABOUTME: dispatch command router — build + show-stage-def, the native surface
// ABOUTME: of the self-hosted dispatch path; everything else is a usage error.
package dispatch

import (
	"fmt"
	"io"
)

// deferredSubcommands are the claude-team subcommands moved to the sibling
// claude-runtime-segregation entity. Naming them in the usage diagnostic makes
// the deferral observable rather than a silent unknown-command rejection.
var deferredSubcommands = map[string]bool{
	"context-budget": true,
	"list-standing":  true,
	"show-standing":  true,
	"spawn-standing": true,
}

// Run routes a `spacedock dispatch <subcommand> [flags]` invocation. build reads
// stdin JSON and writes the dispatch envelope to stdout; show-stage-def writes a
// README subsection to stdout. A deferred or unknown subcommand fails with exit
// 2 and a usage diagnostic on stderr.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
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
		return runBuild(workflowDir, stdin, stdout, stderr)
	case "show-stage-def":
		workflowDir, stage, code := requireStageFlags(args[1:], stderr)
		if code != 0 {
			return code
		}
		return runShowStageDef(workflowDir, stage, stdout, stderr)
	default:
		if deferredSubcommands[args[0]] {
			fmt.Fprintf(stderr,
				"error: 'dispatch %s' is not implemented by this binary; it is deferred "+
					"to the claude-runtime-segregation surface. Native dispatch implements "+
					"only: build, show-stage-def.\n", args[0])
			return 2
		}
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
