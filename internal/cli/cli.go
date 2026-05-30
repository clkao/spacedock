package cli

import (
	"fmt"
	"io"
)

const Version = "0.1.0-dev"

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
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
		fmt.Fprintln(stderr, "spacedock status: not implemented yet")
		return 2
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `spacedock is the Spacedock v1 launcher.

Usage:
  spacedock status [args...]
  spacedock --version
  spacedock --help

Bootstrap status:
  status is intentionally filed as a development entity before implementation.
`)
}
