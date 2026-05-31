// ABOUTME: Release-pipeline CLI: `stamp-version` writes the release version into
// ABOUTME: the plugin.json manifests; `bump-calendar` advances the marketplace key.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spacedock-dev/spacedock/internal/release"
)

// main is the release-tooling entry point invoked by CI (not the user binary).
//
// Usage:
//
//	spacedock-release stamp-version <release-version> <plugin.json> [<plugin.json> ...]
//	spacedock-release bump-calendar <marketplace.json>
//
// stamp-version rewrites each manifest's top-level `version` to the release
// version (AC-4). bump-calendar advances the marketplace plugin entry's calendar
// key to today's `0.0.YYYYMMDDNN` (AC-2d). Both rewrite in place.
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "stamp-version":
		os.Exit(stampVersion(os.Args[2:]))
	case "bump-calendar":
		os.Exit(bumpCalendar(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "spacedock-release: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func stampVersion(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "spacedock-release stamp-version: need <release-version> <manifest> [<manifest> ...]")
		return 2
	}
	version, manifests := args[0], args[1:]
	for _, path := range manifests {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			return 1
		}
		out, err := release.StampVersion(data, version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stamp %s: %v\n", path, err)
			return 1
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			return 1
		}
		fmt.Printf("stamped %s version=%s\n", path, version)
	}
	return 0
}

func bumpCalendar(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "spacedock-release bump-calendar: need exactly one <marketplace.json>")
		return 2
	}
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		return 1
	}
	out, err := release.BumpCalendarVersion(data, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "bump %s: %v\n", path, err)
		return 1
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		return 1
	}
	fmt.Printf("bumped %s\n", path)
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `spacedock-release is the release-pipeline version tool.

Usage:
  spacedock-release stamp-version <release-version> <plugin.json> [<plugin.json> ...]
  spacedock-release bump-calendar <marketplace.json>
`)
}
