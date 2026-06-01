// ABOUTME: runtime-neutral _mods parsing — mod metadata, standing-teammate
// ABOUTME: enumeration, and ## Hook: startup / ## Routing Usage section extraction.
package dispatch

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spacedock-dev/spacedock/internal/status"
)

// ModMetadata is a parsed standing-teammate mod: its frontmatter name, whether
// it declares standing: true, the full frontmatter, and the verbatim ## Agent
// Prompt body (nil when absent). Mirrors the Python parse_mod_metadata return.
type ModMetadata struct {
	Name        string
	Standing    bool
	Frontmatter map[string]string
	AgentPrompt *string
}

// ParseModMetadata parses a mod file's frontmatter and ## Agent Prompt body.
// AgentPrompt is the verbatim body from the line after the heading to EOF, or nil
// when the section is absent. err is non-nil when a ## top-level heading appears
// strictly after ## Agent Prompt (a convention violation) — the message names the
// offending heading. Headings inside ``` fences are not treated as violations.
// Mirrors parse_mod_metadata. Runtime-neutral: reads only the mod file, never
// ~/.claude, so it stays clear of the AC-3 code-side oracle.
func ParseModMetadata(path string) (ModMetadata, error) {
	fm := status.ParseFrontmatter(path)
	standing := strings.EqualFold(strings.TrimSpace(fm["standing"]), "true")

	content := readModFile(path)
	lines := strings.Split(content, "\n")

	agentPromptLine := -1
	for i, line := range lines {
		if line == "## Agent Prompt" {
			agentPromptLine = i
			break
		}
	}

	var agentPrompt *string
	if agentPromptLine >= 0 {
		bodyLines := lines[agentPromptLine+1:]
		inFence := false
		for _, line := range bodyLines {
			stripped := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(stripped, "```") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			if strings.HasPrefix(line, "## ") {
				return ModMetadata{}, &modHeadingError{path: path, heading: line}
			}
		}
		body := strings.Join(bodyLines, "\n")
		agentPrompt = &body
	}

	return ModMetadata{
		Name:        fm["name"],
		Standing:    standing,
		Frontmatter: fm,
		AgentPrompt: agentPrompt,
	}, nil
}

// modHeadingError reports a ## top-level heading after ## Agent Prompt. Its
// message names the offending heading text the way the Python ValueError does.
type modHeadingError struct {
	path    string
	heading string
}

func (e *modHeadingError) Error() string {
	return "mod file " + e.path + ": found trailing top-level heading '" + e.heading +
		"' after '## Agent Prompt' — move the section above '## Agent Prompt' or remove it. " +
		"Everything after '## Agent Prompt' is treated as the prompt body."
}

// StandingTeammate is one declared standing teammate: the spawn name (from the
// mod's ## Hook: startup section, falling back to frontmatter name), the
// frontmatter description, and the routing-usage body (extracted from the mod's
// ## Routing Usage section, "" when absent). The body is carried as data so the
// Claude-specific render lives in internal/claudeteam without re-reading the mod.
type StandingTeammate struct {
	Name             string
	Description      string
	RoutingUsageBody string
}

// EnumerateDeclaredStandingTeammates returns the declared standing teammates for
// a workflow: every {workflow_dir}/_mods/*.md with standing: true in frontmatter,
// in sorted path order. The name comes from ## Hook: startup (authoritative,
// matching spawn-standing) falling back to frontmatter name; a mod with no
// resolvable name is skipped. Returns an empty slice for bare mode (empty
// teamName), no _mods dir, or no standing mods. Mirrors the Python helper.
func EnumerateDeclaredStandingTeammates(workflowDir, teamName string) []StandingTeammate {
	if teamName == "" {
		return nil
	}
	modsDir := filepath.Join(workflowDir, "_mods")
	if !isDir(modsDir) {
		return nil
	}

	var declared []StandingTeammate
	for _, modPath := range sortedModPaths(modsDir) {
		meta, err := ParseModMetadata(modPath)
		if err != nil {
			continue
		}
		if !meta.Standing {
			continue
		}
		hookConfig := ParseHookStartupSpawnConfig(modPath)
		name := hookConfig["name"]
		if name == "" {
			name = meta.Name
		}
		if name == "" {
			continue
		}
		body, _ := ParseRoutingUsageBody(modPath)
		declared = append(declared, StandingTeammate{
			Name:             name,
			Description:      meta.Frontmatter["description"],
			RoutingUsageBody: body,
		})
	}
	return declared
}

// ParseHookStartupSpawnConfig extracts subagent_type / name / model from a mod's
// ## Hook: startup section, declared as `- key: value` bullets. Backtick-wrapped
// inline-code bullets and trailing backtick-wrapped values are unwrapped. Missing
// keys are absent from the map. Mirrors _parse_hook_startup_spawn_config.
func ParseHookStartupSpawnConfig(path string) map[string]string {
	lines := strings.Split(readModFile(path), "\n")

	hookStart := -1
	for i, line := range lines {
		if line == "## Hook: startup" {
			hookStart = i
			break
		}
	}
	if hookStart < 0 {
		return map[string]string{}
	}
	hookEnd := len(lines)
	for i := hookStart + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			hookEnd = i
			break
		}
	}

	config := map[string]string{}
	for _, line := range lines[hookStart+1 : hookEnd] {
		stripped := strings.TrimSpace(line)
		if !strings.HasPrefix(stripped, "- ") {
			continue
		}
		item := strings.TrimSpace(stripped[2:])
		item = unwrapBackticks(item)
		idx := strings.Index(item, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(item[:idx])
		value := strings.TrimSpace(item[idx+1:])
		value = unwrapBackticks(value)
		if key == "subagent_type" || key == "name" || key == "model" {
			config[key] = value
		}
	}
	return config
}

// ParseRoutingUsageBody extracts the body of a mod's ## Routing Usage section:
// the text between the heading and the next ## heading (or EOF), with surrounding
// blank lines stripped. ok is false when the section is missing or blank — callers
// render a fallback. Mirrors _parse_routing_usage_body.
func ParseRoutingUsageBody(path string) (string, bool) {
	lines := strings.Split(readModFile(path), "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Routing Usage" {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	body := lines[start+1 : end]
	for len(body) > 0 && strings.TrimSpace(body[0]) == "" {
		body = body[1:]
	}
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}
	if len(body) == 0 {
		return "", false
	}
	return strings.Join(body, "\n"), true
}

// unwrapBackticks strips a single pair of surrounding backticks from s when the
// whole string is backtick-wrapped, matching the oracle's inline-code unwrap.
func unwrapBackticks(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
		return s[1 : len(s)-1]
	}
	return s
}

// sortedModPaths returns the *.md paths under modsDir in sorted order
// (filepath.Glob already sorts; kept explicit for the contract).
func sortedModPaths(modsDir string) []string {
	paths, _ := filepath.Glob(filepath.Join(modsDir, "*.md"))
	sort.Strings(paths)
	return paths
}

// readModFile reads a file into a string, returning "" on error (the mod parsers
// treat an unreadable mod as having no sections, matching the oracle's
// open()-then-process flow where unreadable mods are skipped upstream).
func readModFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// isDir reports whether path is an existing directory (os.path.isdir).
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
