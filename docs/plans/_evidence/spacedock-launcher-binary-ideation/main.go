package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Minimal spike: validate that we can build the argv shape
//   safehouse [--enable=ssh ...] -- claude --agent spacedock:first-officer [forwarded...]
// and exec it, capturing the actual argv via a safehouse stub on PATH.

func main() {
	args := []string{}
	forwarded := []string{}
	enable := []string{}
	useSeparator := false

	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch a {
		case "--enable-ssh":
			enable = append(enable, "ssh")
			useSeparator = true
		default:
			forwarded = append(forwarded, a)
		}
	}

	for _, e := range enable {
		args = append(args, "--enable="+e)
	}
	if useSeparator {
		args = append(args, "--")
	}
	args = append(args, "claude", "--agent", "spacedock:first-officer")
	args = append(args, forwarded...)

	cmd := exec.Command("safehouse", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "exec failed:", err)
		os.Exit(1)
	}
}
