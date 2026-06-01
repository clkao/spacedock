// ABOUTME: native context-budget — the per-reuse hot path that reads a team
// ABOUTME: member's ~/.claude transcript and team config to decide reuse_ok.
package claudeteam

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultContextLimit  = 200000
	extendedContextLimit = 1000000
	thresholdPct         = 60
	// opus1MMinMinor is the opus minor version at and above which the 1M context
	// window is the default regardless of the [1m] suffix. Since opus 4-7 the
	// extended window runs by default and ensigns drop the suffix in their
	// team-config/jsonl model field, so a forward family rule never goes stale on
	// the next release; 4-6 stays 200k (it still needs the explicit suffix).
	opus1MMinMinor = 7
)

// opusFamily matches claude-opus-4-{minor}, capturing the minor version. The
// family rule reads the minor to decide the default context window.
var opusFamily = regexp.MustCompile(`^claude-opus-4-(\d+)`)

// contextLimitForModel infers the context window size from the model name. The
// [1m] suffix always means 1M; otherwise an opus 4-{minor} with minor >=
// opus1MMinMinor runs the 1M window by default (spawned ensigns drop the suffix).
// Everything else — opus 4-6 without the suffix, sonnet, haiku, unknown models —
// defaults to 200k. Mirrors the vendored claude-team context_limit_for_model.
func contextLimitForModel(model string) int {
	if strings.Contains(model, "[1m]") {
		return extendedContextLimit
	}
	base := model
	if i := strings.Index(base, "["); i >= 0 {
		base = base[:i]
	}
	if m := opusFamily.FindStringSubmatch(base); m != nil {
		if minor, err := strconv.Atoi(m[1]); err == nil && minor >= opus1MMinMinor {
			return extendedContextLimit
		}
	}
	return defaultContextLimit
}

// pyFloat renders a float the way Python's json.dumps does (via float repr): a
// whole-valued float keeps a trailing .0 (20.0, not 20), so the usage_pct field
// byte-matches the oracle. Non-whole values render shortest, like 8.5 or 66.7.
type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return []byte(s), nil
}

// contextBudgetResult is the context-budget JSON envelope. Field order is the
// oracle's insertion order: name, resident_tokens, model, context_limit,
// usage_pct, threshold_pct, reuse_ok, then config_declared_model + the warnings
// (only on drift / fallback). usagePct renders via pyFloat to match Python repr.
type contextBudgetResult struct {
	Name               string  `json:"name"`
	ResidentTokens     int     `json:"resident_tokens"`
	Model              string  `json:"model"`
	ContextLimit       int     `json:"context_limit"`
	UsagePct           pyFloat `json:"usage_pct"`
	ThresholdPct       int     `json:"threshold_pct"`
	ReuseOK            bool    `json:"reuse_ok"`
	ConfigDeclaredMdl  string  `json:"config_declared_model,omitempty"`
	MixedModelsWarning string  `json:"mixed_models_warning,omitempty"`
	ConfigFallbackWarn string  `json:"config_fallback_warning,omitempty"`
	ConfigDriftWarning string  `json:"config_drift_warning,omitempty"`
}

// ContextBudget reads the named team member's most recent subagent transcript and
// team-config model, computes resident-token usage against the model's context
// window, and writes the budget JSON to stdout. home is the resolved HOME (the
// only ~/.claude read seam). The three loud-failure paths each exit 1 with the
// oracle's exact stderr and emit no reuse_ok: missing jsonl, usage-free jsonl,
// agent-not-in-team-config. Mirrors cmd_context_budget.
func ContextBudget(home, name string, stdout, stderr io.Writer) int {
	jsonlPath := findSubagentJSONL(home, name, stderr)
	if jsonlPath == "" {
		fmt.Fprintf(stderr, "Error: no subagent jsonl found for '%s'\n", name)
		return 1
	}

	residentTokens, ok := extractResidentTokens(jsonlPath)
	if !ok {
		fmt.Fprintf(stderr, "Error: no assistant entries with usage in %s\n", jsonlPath)
		return 1
	}

	configModel, ok := lookupModel(home, name)
	if !ok {
		fmt.Fprintf(stderr, "Error: no team config found for member '%s'\n", name)
		return 1
	}

	runtimeModels := extractRuntimeModels(jsonlPath)

	var model string
	var mixedWarning, fallbackWarning string
	switch len(runtimeModels) {
	case 0:
		model = configModel
		fallbackWarning = "no model field in jsonl assistant entries — using config-declared model (provisional)"
	case 1:
		model = runtimeModels[0]
	default:
		// Multiple runtime models: use the smallest context window. min by
		// context_limit, matching Python min(..., key=context_limit_for_model);
		// ties resolve to the first in sorted order (Python min keeps the first).
		sorted := append([]string(nil), runtimeModels...)
		sort.Strings(sorted)
		model = sorted[0]
		minLimit := contextLimitForModel(model)
		for _, m := range sorted[1:] {
			if contextLimitForModel(m) < minLimit {
				model = m
				minLimit = contextLimitForModel(m)
			}
		}
		mixedWarning = fmt.Sprintf(
			"multiple models seen in jsonl: %s — using smallest context window",
			pyStrList(sorted))
	}

	contextLimit := contextLimitForModel(model)
	configLimit := contextLimitForModel(configModel)

	var driftWarning string
	driftSeen := len(runtimeModels) > 0 && configLimit != contextLimit
	if driftSeen {
		driftWarning = fmt.Sprintf("team config requested %s but runtime is %s", configModel, model)
	}

	usagePct := pyRound1(float64(residentTokens) / float64(contextLimit) * 100)
	reuseOK := usagePct <= thresholdPct

	result := contextBudgetResult{
		Name:           name,
		ResidentTokens: residentTokens,
		Model:          model,
		ContextLimit:   contextLimit,
		UsagePct:       pyFloat(usagePct),
		ThresholdPct:   thresholdPct,
		ReuseOK:        reuseOK,
	}
	if driftSeen {
		result.ConfigDeclaredMdl = configModel
	}
	result.MixedModelsWarning = mixedWarning
	result.ConfigFallbackWarn = fallbackWarning
	result.ConfigDriftWarning = driftWarning

	return emitJSON(stdout, result)
}

// findSubagentJSONL finds the most recently modified subagent jsonl whose
// agentType matches name. It scopes the scan to the team session that declares
// name as a member (narrowing from every project's subagents to one session's),
// falling back to the broad ~/.claude/projects scan with a stderr warning when
// the narrowed scan finds nothing. Returns "" when no match exists. Mirrors
// find_subagent_jsonl.
func findSubagentJSONL(home, name string, stderr io.Writer) string {
	broadPatterns := []string{
		filepath.Join(home, ".claude", "projects", "*", "subagents", "agent-*.meta.json"),
		filepath.Join(home, ".claude", "projects", "*", "*", "subagents", "agent-*.meta.json"),
	}

	if narrowed := narrowedSubagentPatterns(home, name); narrowed != nil {
		matches := scanSubagentMeta(narrowed, name)
		if len(matches) == 0 {
			fmt.Fprintf(stderr,
				"warning: narrowed subagent scan for '%s' found nothing; "+
					"falling back to broad ~/.claude/projects scan\n", name)
			matches = scanSubagentMeta(broadPatterns, name)
		}
		return newestMatch(matches)
	}
	return newestMatch(scanSubagentMeta(broadPatterns, name))
}

// narrowedSubagentPatterns returns the glob patterns scoped to the one team
// session whose config.json lists name as a member and declares a leadSessionId.
// Returns nil when no such config exists. Mirrors _narrowed_subagent_patterns.
func narrowedSubagentPatterns(home, name string) []string {
	teamsPattern := filepath.Join(home, ".claude", "teams", "*", "config.json")
	configs, _ := filepath.Glob(teamsPattern)
	for _, configPath := range configs {
		cfg, ok := readTeamConfig(configPath)
		if !ok {
			continue
		}
		if !cfg.hasMember(name) {
			continue
		}
		if cfg.LeadSessionID == "" {
			continue
		}
		return []string{
			filepath.Join(home, ".claude", "projects", "*", cfg.LeadSessionID,
				"subagents", "agent-*.meta.json"),
		}
	}
	return nil
}

// subagentMatch pairs a jsonl path with its meta-file mtime for newest-wins.
type subagentMatch struct {
	mtime int64
	jsonl string
}

// scanSubagentMeta globs the patterns for agent meta files, decoding each and
// collecting the sibling jsonl path when agentType matches name and the jsonl
// exists. Mirrors _scan_subagent_meta.
func scanSubagentMeta(patterns []string, name string) []subagentMatch {
	var matches []subagentMatch
	for _, pattern := range patterns {
		paths, _ := filepath.Glob(pattern)
		for _, metaPath := range paths {
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var meta struct {
				AgentType string `json:"agentType"`
			}
			if json.Unmarshal(data, &meta) != nil {
				continue
			}
			if meta.AgentType != name {
				continue
			}
			info, err := os.Stat(metaPath)
			if err != nil {
				continue
			}
			jsonlPath := strings.Replace(metaPath, ".meta.json", ".jsonl", 1)
			if fi, err := os.Stat(jsonlPath); err == nil && fi.Mode().IsRegular() {
				matches = append(matches, subagentMatch{info.ModTime().UnixNano(), jsonlPath})
			}
		}
	}
	return matches
}

// newestMatch returns the jsonl path of the newest-mtime match, or "".
func newestMatch(matches []subagentMatch) string {
	if len(matches) == 0 {
		return ""
	}
	newest := matches[0]
	for _, m := range matches[1:] {
		if m.mtime > newest.mtime {
			newest = m
		}
	}
	return newest.jsonl
}

// extractResidentTokens scans the jsonl backward for the first assistant entry
// with a non-zero usage sum (input + cache_creation + cache_read), which is the
// true peak resident for both live and overflow-dead ensigns. ok is false when
// no such entry exists. Mirrors extract_resident_tokens.
func extractResidentTokens(jsonlPath string) (int, bool) {
	lines := readLines(jsonlPath)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		usage, ok := assistantUsage(line)
		if !ok {
			continue
		}
		resident := usage.Input + usage.CacheCreation + usage.CacheRead
		if resident > 0 {
			return resident, true
		}
	}
	return 0, false
}

// extractRuntimeModels returns the distinct model strings across assistant
// entries in the jsonl, sorted for deterministic warning rendering. Mirrors
// extract_runtime_models (set), with sorting applied where the oracle sorts.
func extractRuntimeModels(jsonlPath string) []string {
	seen := map[string]bool{}
	for _, line := range readLines(jsonlPath) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry struct {
			Type    string `json:"type"`
			Message *struct {
				Model string `json:"model"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		if entry.Type != "assistant" || entry.Message == nil || entry.Message.Model == "" {
			continue
		}
		seen[entry.Message.Model] = true
	}
	out := make([]string, 0, len(seen))
	for m := range seen {
		out = append(out, m)
	}
	return out
}

// usageFields is the assistant-entry usage shape resident tokens sum over.
type usageFields struct {
	Input         int
	CacheCreation int
	CacheRead     int
}

// assistantUsage decodes one jsonl line and returns its usage when it is an
// assistant entry carrying a usage object. ok is false otherwise (non-assistant,
// missing message/usage, or undecodable), matching the oracle's skip conditions.
func assistantUsage(line string) (usageFields, bool) {
	var entry struct {
		Type    string `json:"type"`
		Message *struct {
			Usage *struct {
				Input         int `json:"input_tokens"`
				CacheCreation int `json:"cache_creation_input_tokens"`
				CacheRead     int `json:"cache_read_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal([]byte(line), &entry) != nil {
		return usageFields{}, false
	}
	if entry.Type != "assistant" || entry.Message == nil || entry.Message.Usage == nil {
		return usageFields{}, false
	}
	u := entry.Message.Usage
	return usageFields{u.Input, u.CacheCreation, u.CacheRead}, true
}

// teamConfig is the subset of a team config.json the context-budget reads.
type teamConfig struct {
	LeadSessionID string `json:"leadSessionId"`
	Members       []struct {
		Name  string `json:"name"`
		Model string `json:"model"`
	} `json:"members"`
}

// hasMember reports whether the config lists a member with the given name.
func (c teamConfig) hasMember(name string) bool {
	for _, m := range c.Members {
		if m.Name == name {
			return true
		}
	}
	return false
}

// readTeamConfig decodes a team config.json. ok is false on read/parse error,
// matching the oracle's try/except skip.
func readTeamConfig(path string) (teamConfig, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return teamConfig{}, false
	}
	var cfg teamConfig
	if json.Unmarshal(data, &cfg) != nil {
		return teamConfig{}, false
	}
	return cfg, true
}

// lookupModel scans every ~/.claude/teams/*/config.json for a member named name
// and returns its model. ok is false when no team config lists the member.
// Mirrors lookup_model. A member with an empty model string still resolves (ok
// true, model ""), matching the oracle returning member.get("model").
func lookupModel(home, name string) (string, bool) {
	pattern := filepath.Join(home, ".claude", "teams", "*", "config.json")
	configs, _ := filepath.Glob(pattern)
	for _, configPath := range configs {
		cfg, ok := readTeamConfig(configPath)
		if !ok {
			continue
		}
		for _, m := range cfg.Members {
			if m.Name == name {
				return m.Model, true
			}
		}
	}
	return "", false
}

// readLines reads a file into its lines (no trailing-newline split artifact
// beyond what Python readlines yields). On read error returns nil — the callers
// treat a missing/unreadable jsonl as "no usable entries", which the upstream
// find_subagent_jsonl existence check already guards.
func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	s := string(data)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// pyRound1 rounds to one decimal place using banker's rounding, matching Python
// round(x, 1) (round-half-to-even). The reuse threshold comparison and the
// emitted usage_pct both depend on this matching the oracle byte-for-byte.
func pyRound1(x float64) float64 {
	scaled := x * 10
	r := math.RoundToEven(scaled)
	return r / 10
}

// pyStrList renders a string slice the way Python renders sorted(set) in the
// mixed-models warning: ['a', 'b'] with single quotes and ", " separators.
func pyStrList(items []string) string {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = "'" + s + "'"
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// emitJSON writes v as two-space-indented JSON with a trailing newline, matching
// Python json.dumps(indent=2) + print() byte-for-byte including its ensure_ascii
// escaping of non-ASCII content (e.g. an em-dash in the mixed-models warning).
func emitJSON(w io.Writer, v any) int {
	return EmitPythonJSON(w, v)
}
