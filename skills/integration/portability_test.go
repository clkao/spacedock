// ABOUTME: Portability oracle — the shipped instruction surface assumes nothing
// ABOUTME: from the running user's machine (no HOME config, interpreter, or internal path).
package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A clean install must run for any user. The shipped instruction surface — the
// text a clean-room user actually reads and follows (skills/**/*.md excluding
// the test-only integration dir, plus mods/) — must therefore name none of three
// non-portable dependencies:
//
//  1. personal-config:    a HOME-rooted ~/.claude / $HOME+.claude /
//     os.UserHomeDir+.claude read (depends on the running user's home layout).
//  2. interpreter-on-PATH: a python / python3 shell-out, or a commission/bin
//     helper invocation, on the dispatch critical path (depends on an
//     interpreter or plugin-private script being installed).
//  3. internal-helper-path: a plugin-private absolute path baked into shipped
//     instructions (skills/commission/bin/status, {spacedock_plugin_dir},
//     .agents/plugins/marketplace.json) — a path that does not exist on a fresh
//     install.
//
// This is the shipped-instruction-surface complement to zs #246's two
// host-neutrality oracles (which police the Go source and the generic FO
// contract prose, one altitude down). The walk helper shippedSkillText is shared
// read-only with skill_surface_test.go; this file adds no new walk.

// homeRootedClaudeRe matches only the HOME-rooted personal-config forms: a
// `~/.claude` tilde path, or `$HOME` / `os.UserHomeDir` joined with `.claude` on
// the same line. It deliberately does NOT match a project-relative `.claude/`
// path (e.g. commission's {project_root}/.claude/agents or debrief's
// .claude/worktrees prune note) — those exist in any checkout and are portable.
// That HOME-rooted-vs-project-relative distinction is the discriminator that
// keeps the personal-config check from false-positiving on legitimate usage.
var homeRootedClaudeRe = regexp.MustCompile(`~/\.claude|\$HOME[^\n]*\.claude|os\.UserHomeDir[^\n]*\.claude`)

// interpreterRe matches an interpreter-on-PATH dependency on the dispatch path:
// a `python`/`python3` shell-out or a `commission/bin/...` helper invocation.
// `\bpython` avoids matching unrelated substrings; commission/bin is the
// pre-#246 plugin-private helper path the FO loop no longer needs.
var interpreterRe = regexp.MustCompile(`\bpython3?\b|commission/bin`)

// internalHelperPaths are the plugin-private absolute paths that do not exist on
// a fresh install. Substring (not regex) — these are literal path fragments.
var internalHelperPaths = []string{
	"skills/commission/bin/status",
	"{spacedock_plugin_dir}",
	".agents/plugins/marketplace.json",
}

// isClaudeAdapter reports whether a shipped file is a Claude host-runtime
// adapter (claude-*-runtime.md). Per zs #246 a `~/.claude/teams` read is the
// legitimate, quarantined Claude coupling and lives exactly in these adapters,
// so the personal-config check — and ONLY that check — excludes them. The
// interpreter and internal-helper-path checks still apply to adapters.
func isClaudeAdapter(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "claude-") && strings.HasSuffix(base, "-runtime.md")
}

// TestShippedSurfaceHasNoHiddenMachineDependency locks AC-1: the shipped
// instruction surface names none of the three non-portable markers. It is
// falsifiable — reintroducing any marker into a shipped file turns it RED naming
// the file — and it is not a tautology: it guards against an empty walk so a
// future scope bug that empties shippedSkillText fails loudly rather than
// passing vacuously.
func TestShippedSurfaceHasNoHiddenMachineDependency(t *testing.T) {
	root := skillsRoot(t)
	repo := repoRoot(t)
	files := shippedSkillText(t, root, repo)
	if len(files) == 0 {
		t.Fatal("shippedSkillText walked zero files — scope bug; the portability oracle would pass vacuously")
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		content := string(data)

		// personal-config: HOME-rooted ~/.claude, excluding the Claude adapters
		// where a ~/.claude/teams read is the legitimate quarantined coupling.
		if !isClaudeAdapter(path) {
			if m := homeRootedClaudeRe.FindString(content); m != "" {
				t.Errorf("%s carries a HOME-rooted personal-config dependency %q — a clean install has no such file", path, m)
			}
		}

		// interpreter-on-PATH and internal-helper-path apply to ALL shipped
		// files, adapters included.
		if m := interpreterRe.FindString(content); m != "" {
			t.Errorf("%s carries an interpreter-on-PATH dependency %q — the dispatch path must not assume an installed interpreter/helper", path, m)
		}
		for _, p := range internalHelperPaths {
			if strings.Contains(content, p) {
				t.Errorf("%s bakes in plugin-private path %q — it does not exist on a fresh install", path, p)
			}
		}
	}
}

// TestPortabilityCheckDiscriminatesHostSpecific locks AC-2: the oracle is a
// discriminator, not a blunt absence test. It proves — against the REAL shipped
// surface — that the legitimately host-specific forms are present yet GREEN:
//
//   - the Claude adapter's legitimate `~/.claude/teams` read is present in the
//     walked surface (so the adapter exclusion is load-bearing, not vacuous),
//     and
//   - the project-relative `.claude/agents` / `.claude/worktrees` paths that
//     commission/refit/debrief use are present in the walked surface yet the
//     HOME-rooted regex does NOT match them (so the personal-config check does
//     not false-positive on portable project-relative paths).
//
// If either positive control disappears from the surface, this test fails —
// catching a refactor that quietly removed the very thing the discriminator
// distinguishes, which would otherwise let TestShippedSurfaceHasNoHiddenMachineDependency
// pass for the wrong reason.
func TestPortabilityCheckDiscriminatesHostSpecific(t *testing.T) {
	root := skillsRoot(t)
	repo := repoRoot(t)
	files := shippedSkillText(t, root, repo)

	var sawAdapterHomeClaude bool
	var sawProjectRelativeClaude bool
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		content := string(data)

		// Positive control 1: a Claude adapter carries a legitimate ~/.claude
		// read that the personal-config check (correctly) excludes.
		if isClaudeAdapter(path) && strings.Contains(content, "~/.claude") {
			sawAdapterHomeClaude = true
		}

		// Positive control 2: a non-adapter shipped file carries a
		// project-relative .claude/ path. It must be present (proving the
		// discriminator has something to discriminate) AND the HOME-rooted regex
		// must not match it (proving no false positive). Scan line by line so a
		// HOME-rooted hit elsewhere in the same file cannot mask a project-
		// relative line.
		if !isClaudeAdapter(path) {
			for _, line := range strings.Split(content, "\n") {
				if !strings.Contains(line, ".claude/") {
					continue
				}
				if strings.Contains(line, "~/.claude") || strings.Contains(line, "$HOME") {
					continue // a HOME-rooted line, not the project-relative form
				}
				sawProjectRelativeClaude = true
				if homeRootedClaudeRe.MatchString(line) {
					t.Errorf("%s: project-relative .claude line wrongly matched the HOME-rooted personal-config regex (false positive): %q", path, strings.TrimSpace(line))
				}
			}
		}
	}

	if !sawAdapterHomeClaude {
		t.Error("positive control missing: no Claude adapter carries a ~/.claude read — the adapter-exclusion in the personal-config check is no longer load-bearing")
	}
	if !sawProjectRelativeClaude {
		t.Error("positive control missing: no shipped file carries a project-relative .claude/ path — the HOME-rooted-vs-project-relative discriminator has nothing to discriminate")
	}
}
