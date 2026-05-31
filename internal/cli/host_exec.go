// ABOUTME: Production hostOps — resolves the installed plugin manifest via
// ABOUTME: `claude/codex plugin list --json`, execs the host, and shells installs.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// ResolveManifest returns the installed spacedock@spacedock plugin manifest path
// for host, or "" (no error) when no plugin is installed. The two hosts resolve
// differently: Claude reports an installPath in `claude plugin list --json`;
// Codex 0.132.0 has no --json (it rejects the flag, exit 2) and its text listing
// carries no install path, so the Codex path confirms the install via the text
// listing and resolves the manifest under the deterministic Codex plugin cache.
func (e execHost) ResolveManifest(host string) (string, error) {
	if host == "codex" {
		return e.resolveCodexManifest()
	}
	return e.resolveClaudeManifest(host)
}

// resolveClaudeManifest shells `claude plugin list --json`, finds the spacedock@
// spacedock entry, and returns <installPath>/.claude-plugin/plugin.json. Returns
// "" (no error) when the host reports no matching install or no installPath.
func (execHost) resolveClaudeManifest(host string) (string, error) {
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

// resolveCodexManifest confirms spacedock@spacedock is installed via the text
// `codex plugin list` (no --json — 0.132.0 rejects it) and resolves the manifest
// under the Codex plugin cache. Codex installs land at
// <CODEX_HOME>/plugins/cache/<marketplace>/<plugin>/<version>/.codex-plugin/plugin.json;
// the listing carries no install path, so the cache layout is the resolver.
// Returns "" (no error) when the plugin is not installed or no cached manifest
// exists for it yet.
func (execHost) resolveCodexManifest() (string, error) {
	out, err := exec.Command("codex", "plugin", "list").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codex plugin list: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	if !codexEntryInstalled(string(out), "spacedock@spacedock") {
		return "", nil
	}
	cacheRoot := filepath.Join(codexHome(), "plugins", "cache", "spacedock", "spacedock")
	versionDir, err := latestVersionDir(cacheRoot)
	if err != nil || versionDir == "" {
		return "", nil
	}
	manifest := filepath.Join(versionDir, manifestSubpath("codex"))
	if _, statErr := os.Stat(manifest); statErr != nil {
		return "", nil
	}
	return manifest, nil
}

// codexEntryInstalled reports whether the `codex plugin list` text output marks
// the given plugin id as installed. The listing renders one indented line per
// plugin as `<id> (installed[, enabled]) | (not installed)`; an installed entry
// carries the literal `<id> (installed`.
func codexEntryInstalled(listing, id string) bool {
	for _, line := range strings.Split(listing, "\n") {
		if strings.Contains(line, id+" (installed") {
			return true
		}
	}
	return false
}

// codexHome returns the Codex config/cache root: $CODEX_HOME when set, else
// ~/.codex (matching the Codex CLI's own resolution).
func codexHome() string {
	if h := os.Getenv("CODEX_HOME"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex"
	}
	return filepath.Join(home, ".codex")
}

// latestVersionDir returns the semver-greatest immediate subdirectory of root
// (the installed plugin's version dir). Returns "" (no error) when root is absent
// or has no subdirectories. Codex installs a single version, but a stale cache
// may hold several; a semver compare picks the most recent install — a lexical
// compare would wrongly order `0.10.0` before `0.9.0`.
func latestVersionDir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	latest := ""
	for _, e := range entries {
		if e.IsDir() && (latest == "" || compareVersion(e.Name(), latest) > 0) {
			latest = e.Name()
		}
	}
	if latest == "" {
		return "", nil
	}
	return filepath.Join(root, latest), nil
}

// compareVersion orders dotted plugin version names (e.g. `0.12.1`) numerically
// per component, so `0.10.0` sorts after `0.9.0`. It returns -1, 0, or 1. A
// component that does not parse as an integer falls back to a lexical compare of
// that component, so non-numeric names still order deterministically.
func compareVersion(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv string
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		an, aerr := strconv.Atoi(av)
		bn, berr := strconv.Atoi(bv)
		if aerr == nil && berr == nil {
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
			continue
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

// manifestSubpath returns the per-host manifest location under an install root.
func manifestSubpath(host string) string {
	if host == "codex" {
		return filepath.Join(".codex-plugin", "plugin.json")
	}
	return filepath.Join(".claude-plugin", "plugin.json")
}

// marketplaceAddArg composes the `claude plugin marketplace add` target: the
// bare source, or `source@branch` when a branch is pinned (the @ref shorthand the
// host resolves against the repo-root marketplace.json on that ref).
func marketplaceAddArg(source, branch string) string {
	if branch != "" {
		return source + "@" + branch
	}
	return source
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
	marketplaceArg := marketplaceAddArg(source, branch)
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
