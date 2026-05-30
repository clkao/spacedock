// ABOUTME: Production hostOps — resolves the installed plugin manifest via
// ABOUTME: `claude/codex plugin list --json`, execs the host, and shells installs.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// execHost backs hostOps with the real host CLIs and process exec.
type execHost struct{}

var _ hostOps = execHost{}

// pluginListEntry is the subset of `<host> plugin list --json` this binary
// reads: the `plugin@marketplace` id and the resolved install path. (Observed
// schema: the entry carries `id`, not separate name/marketplace fields.)
type pluginListEntry struct {
	ID          string `json:"id"`
	InstallPath string `json:"installPath"`
}

// ResolveManifest shells `<host> plugin list --json`, finds the spacedock@
// spacedock entry, and returns its manifest path. The Claude manifest lives at
// <installPath>/.claude-plugin/plugin.json; the Codex one at
// <installPath>/.codex-plugin/plugin.json. Returns "" (no error) when the host
// reports no matching install or no installPath.
func (execHost) ResolveManifest(host string) (string, error) {
	out, err := exec.Command(host, "plugin", "list", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("%s plugin list --json: %w", host, err)
	}
	var entries []pluginListEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return "", fmt.Errorf("parse %s plugin list --json: %w", host, err)
	}
	for _, e := range entries {
		if e.ID == "spacedock@spacedock" {
			if e.InstallPath == "" {
				return "", nil
			}
			return filepath.Join(e.InstallPath, manifestSubpath(host)), nil
		}
	}
	return "", nil
}

// manifestSubpath returns the per-host manifest location under an install root.
func manifestSubpath(host string) string {
	if host == "codex" {
		return filepath.Join(".codex-plugin", "plugin.json")
	}
	return filepath.Join(".claude-plugin", "plugin.json")
}

// Launch replaces the current process with argv via execve, so the host CLI
// owns the terminal (interactive `claude --agent …`). It returns only when exec
// itself fails.
func (execHost) Launch(argv []string) error {
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return syscall.Exec(bin, argv, os.Environ())
}

// Install shells the host plugin marketplace add + install pair. The marketplace
// source is pinned to branch via @ref (Claude) when set. Codex installs are not
// shelled here — runInit emits the documented prose for that host.
func (execHost) Install(host, source, branch string) (string, error) {
	if host != "claude" {
		return "", fmt.Errorf("programmatic install is only supported for claude; codex install is documented prose")
	}
	marketplaceArg := source
	if branch != "" {
		marketplaceArg = source + "@" + branch
	}
	var sb strings.Builder
	for _, args := range [][]string{
		{"plugin", "marketplace", "add", marketplaceArg},
		{"plugin", "install", "spacedock@spacedock"},
	} {
		cmd := exec.Command(host, args...)
		out, err := cmd.CombinedOutput()
		sb.Write(out)
		if err != nil {
			return sb.String(), fmt.Errorf("%s %s: %w", host, strings.Join(args, " "), err)
		}
	}
	return sb.String(), nil
}
