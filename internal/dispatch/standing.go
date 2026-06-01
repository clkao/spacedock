// ABOUTME: standing-subcommand orchestration — list/show/spawn-standing wire the
// ABOUTME: runtime-neutral _mods parsing to the Claude render + member-exists probe.
package dispatch

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// spawnModelEnum is the Agent-schema model enum spawn-standing validates against.
var spawnModelEnum = map[string]bool{"sonnet": true, "opus": true, "haiku": true}

const spawnModelEnumList = "must be one of: sonnet, opus, haiku"

// runListStanding writes the absolute paths of standing: true mods under
// {workflowDir}/_mods/*.md, one per line, sorted by filename. Exit 0 on success
// including zero matches (empty stdout); exit 1 only when the workflow dir is
// unresolvable or a mod is unreadable. The _mods parsing is runtime-neutral, so
// this lives in the generic dispatch package. Mirrors cmd_list_standing.
func runListStanding(workflowDir string, stdout, stderr io.Writer) int {
	if !isDir(workflowDir) {
		fmt.Fprintf(stderr, "error: workflow directory not found: %s\n", workflowDir)
		return 1
	}
	modsDir := filepath.Join(workflowDir, "_mods")
	if !isDir(modsDir) {
		return 0
	}

	var standingPaths []string
	for _, modPath := range sortedModPaths(modsDir) {
		meta, err := ParseModMetadata(modPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: unreadable mod %s: %s\n", modPath, err)
			return 1
		}
		if meta.Standing {
			abs, e := filepath.Abs(modPath)
			if e != nil {
				abs = modPath
			}
			standingPaths = append(standingPaths, abs)
		}
	}
	for _, p := range standingPaths {
		fmt.Fprintln(stdout, p)
	}
	return 0
}

// runShowStanding writes the rendered `### Standing teammates available in your
// team` markdown for the workflow's declared standing teammates. The enumeration
// is runtime-neutral (here); the rendered body is Claude SendMessage routing prose
// (claudeteam.RenderStandingTeammatesSection). Exit 0 on success including the
// degenerate empty case. Mirrors cmd_show_standing.
func runShowStanding(workflowDir string, stdout, stderr io.Writer) int {
	if !isDir(workflowDir) {
		fmt.Fprintf(stderr, "error: workflow directory not found: %s\n", workflowDir)
		return 1
	}
	// A truthy sentinel team name: the filesystem scan is team-agnostic, and the
	// bare-mode short-circuit lives in build upstream of this call.
	teammates := EnumerateDeclaredStandingTeammates(workflowDir, "_show_standing_")
	rendered := claudeteam.RenderStandingTeammatesSection(toClaudeTeammates(teammates))
	if rendered != "" {
		fmt.Fprintln(stdout, rendered)
	}
	return 0
}

// runSpawnStanding emits an Agent() spec JSON for a standing-teammate mod, or
// reports already-alive when the target team config already lists the declared
// member. It fails loudly (exit 1, stderr) on: a placeholder team name, missing
// mod file, missing standing: true, missing ## Agent Prompt, missing subagent_type
// / name / model, or a model outside the enum. home is the resolved HOME (the
// member-exists probe's only ~/.claude read, owned by claudeteam). Mirrors
// cmd_spawn_standing.
func runSpawnStanding(home, modPath, teamName string, stdout, stderr io.Writer) int {
	if teamName == "" || teamName == "none" || teamName == "None" {
		fmt.Fprintf(stderr,
			"error: spawn-standing requires a real team name; got '%s'. "+
				"Call TeamCreate first and pass the returned team_name via --team.\n", teamName)
		return 1
	}
	if !isFile(modPath) {
		fmt.Fprintf(stderr, "error: mod file not found: %s\n", modPath)
		return 1
	}

	meta, err := ParseModMetadata(modPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}
	if !meta.Standing {
		fmt.Fprintf(stderr,
			"error: mod %s is missing 'standing: true' in frontmatter — "+
				"not a standing-teammate mod\n", modPath)
		return 1
	}
	if meta.AgentPrompt == nil {
		fmt.Fprintf(stderr,
			"error: mod %s has no '## Agent Prompt' section — "+
				"required for standing-teammate spawn\n", modPath)
		return 1
	}

	hookConfig := ParseHookStartupSpawnConfig(modPath)
	subagentType := hookConfig["subagent_type"]
	if subagentType == "" {
		subagentType = meta.Frontmatter["subagent_type"]
	}
	declaredName := hookConfig["name"]
	if declaredName == "" {
		declaredName = meta.Name
	}
	model := hookConfig["model"]

	if subagentType == "" {
		fmt.Fprintf(stderr,
			"error: mod %s has no 'subagent_type' in '## Hook: startup' or frontmatter\n", modPath)
		return 1
	}
	if declaredName == "" {
		fmt.Fprintf(stderr,
			"error: mod %s has no 'name' in '## Hook: startup' or frontmatter\n", modPath)
		return 1
	}
	if model == "" {
		fmt.Fprintf(stderr,
			"error: mod %s has no 'model' in '## Hook: startup' — %s\n", modPath, spawnModelEnumList)
		return 1
	}
	if !spawnModelEnum[model] {
		fmt.Fprintf(stderr,
			"error: invalid model '%s' in '## Hook: startup' of %s — %s\n", model, modPath, spawnModelEnumList)
		return 1
	}

	if claudeteam.MemberExists(home, teamName, declaredName) {
		// already-alive: a one-line JSON object matching the oracle's json.dumps
		// without indent — Python's default ", " / ": " separators, not Go's
		// space-free Marshal. The two keys are fixed, so format directly to match.
		// Escape the name through the same ensure_ascii routine the emitters use,
		// so a non-ASCII declared name stays byte-identical to json.dumps.
		nameJSON, _ := json.Marshal(declaredName)
		nameJSON = claudeteam.EscapeNonASCII(nameJSON)
		fmt.Fprintf(stdout, "{\"status\": \"already-alive\", \"name\": %s}\n", nameJSON)
		return 0
	}

	spec := spawnSpec{
		SubagentType: subagentType,
		Name:         declaredName,
		TeamName:     teamName,
		Model:        model,
		Prompt:       *meta.AgentPrompt,
	}
	return emitSpawnJSON(stdout, spec)
}

// spawnSpec is the Agent() call spec spawn-standing emits when the member is not
// yet alive. Field order matches the oracle's dict: subagent_type, name,
// team_name, model, prompt.
type spawnSpec struct {
	SubagentType string `json:"subagent_type"`
	Name         string `json:"name"`
	TeamName     string `json:"team_name"`
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
}

// emitSpawnJSON writes the spec as two-space-indented JSON with a trailing
// newline, matching Python json.dumps(spec, indent=2) + print() byte-for-byte,
// including its ensure_ascii escaping of a non-ASCII Agent Prompt.
func emitSpawnJSON(stdout io.Writer, spec spawnSpec) int {
	return claudeteam.EmitPythonJSON(stdout, spec)
}

// toClaudeTeammates maps the runtime-neutral StandingTeammate to the Claude
// render's input type (same fields), so the render package needs no dispatch import.
func toClaudeTeammates(in []StandingTeammate) []claudeteam.StandingTeammate {
	out := make([]claudeteam.StandingTeammate, len(in))
	for i, tm := range in {
		out[i] = claudeteam.StandingTeammate{
			Name:             tm.Name,
			Description:      tm.Description,
			RoutingUsageBody: tm.RoutingUsageBody,
		}
	}
	return out
}
