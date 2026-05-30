// ABOUTME: Stage parser matching the oracle's parse_stages_block /
// ABOUTME: parse_stages_with_defaults — indentation-driven, stdlib only.
package status

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

// stageNameRe is the dispatch-name regex stage names must match.
var stageNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// stage is a resolved workflow stage with defaults applied. Optional carried
// fields (feedback-to, agent, fresh, model) are kept verbatim when present.
type stage struct {
	name        string
	worktree    bool
	concurrency int
	gate        bool
	terminal    bool
	initial     bool
	optional    map[string]string
}

// frontmatterLines returns the raw lines between the first two `---` fences of
// the file at path (the body the stages block lives in), matching the oracle's
// README frontmatter slice in parse_stages_block.
func frontmatterLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	inFM := false
	for _, line := range splitLines(string(data)) {
		if line == "---" {
			if inFM {
				break
			}
			inFM = true
			continue
		}
		if inFM {
			lines = append(lines, line)
		}
	}
	return lines
}

// indentOf returns the leading-whitespace width of a line.
func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// parseStagesBlock parses the stages: block from README frontmatter, returning
// ordered stages with resolved defaults, or nil when there is no stages: block
// or no states entries. Matches parse_stages_block.
func parseStagesBlock(path string) []stage {
	stages, _ := parseStagesWithDefaults(path)
	return stages
}

// parseStagesWithDefaults returns the ordered stages and the raw stages.defaults
// map. Matches parse_stages_block + parse_stages_with_defaults (the oracle
// re-parses for the defaults map; we collect both in one pass).
func parseStagesWithDefaults(path string) ([]stage, map[string]string) {
	lines := frontmatterLines(path)

	stagesStart := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == "stages:" {
			stagesStart = i
			break
		}
	}
	if stagesStart < 0 {
		return nil, map[string]string{}
	}

	defaults := map[string]string{}
	var states []map[string]string

	i := stagesStart + 1
	stagesIndent := -1
	for i < len(lines) {
		line := lines[i]
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" {
			i++
			continue
		}
		indent := indentOf(line)
		if stagesIndent < 0 {
			stagesIndent = indent
		} else if indent < stagesIndent {
			break
		}

		if indent == stagesIndent {
			switch stripped {
			case "defaults:":
				i++
				for i < len(lines) {
					dline := lines[i]
					dstripped := strings.TrimLeft(dline, " \t")
					if dstripped == "" {
						i++
						continue
					}
					if indentOf(dline) <= stagesIndent {
						break
					}
					if strings.Contains(dstripped, ":") {
						k, v, _ := strings.Cut(dstripped, ":")
						defaults[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
					i++
				}
				continue
			case "states:":
				i++
				var current map[string]string
				for i < len(lines) {
					sline := lines[i]
					sstripped := strings.TrimLeft(sline, " \t")
					if sstripped == "" {
						i++
						continue
					}
					if indentOf(sline) <= stagesIndent {
						break
					}
					if strings.HasPrefix(sstripped, "- name:") {
						name := strings.TrimSpace(strings.TrimPrefix(sstripped, "- name:"))
						current = map[string]string{"name": name}
						states = append(states, current)
					} else if current != nil && strings.Contains(sstripped, ":") && !strings.HasPrefix(sstripped, "- ") {
						k, v, _ := strings.Cut(sstripped, ":")
						current[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
					i++
				}
				continue
			}
		}
		i++
	}

	if len(states) == 0 {
		return nil, defaults
	}

	defaultWorktree := strings.EqualFold(getOr(defaults, "worktree", "false"), "true")
	defaultConcurrency := atoiOr(getOr(defaults, "concurrency", "2"), 2)

	result := make([]stage, 0, len(states))
	for _, st := range states {
		s := stage{
			name:        st["name"],
			worktree:    strings.EqualFold(getOr(st, "worktree", boolStr(defaultWorktree)), "true"),
			concurrency: atoiOr(getOr(st, "concurrency", strconv.Itoa(defaultConcurrency)), defaultConcurrency),
			gate:        strings.EqualFold(getOr(st, "gate", "false"), "true"),
			terminal:    strings.EqualFold(getOr(st, "terminal", "false"), "true"),
			initial:     strings.EqualFold(getOr(st, "initial", "false"), "true"),
			optional:    map[string]string{},
		}
		for _, of := range []string{"feedback-to", "agent", "fresh", "model"} {
			if v, ok := st[of]; ok {
				s.optional[of] = v
			}
		}
		result = append(result, s)
	}
	return result, defaults
}

func getOr(m map[string]string, k, def string) string {
	if v, ok := m[k]; ok {
		return v
	}
	return def
}

func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}

func boolStr(b bool) string {
	if b {
		return "True"
	}
	return "False"
}
