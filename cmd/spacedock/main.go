package main

import (
	"os"

	"github.com/clkao/spacedock-v1/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
