// ABOUTME: Validation matching validate_workflow — flat/folder conflicts,
// ABOUTME: stage-name regex, and per-id-style id rules with evidence lines.
package status

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// entityEvidence formats the Error: ... workflow= scope= slug= id= [display=]
// path= evidence line. Matches entity_evidence.
func entityEvidence(e *entity, workflowDir, problem, displayID string) string {
	display := displayID
	if display == "" {
		display = e.displayID
	}
	parts := []string{
		fmt.Sprintf("Error: %s:", problem),
		"workflow=" + workflowDir,
		"scope=" + scopeOf(e),
		"slug=" + e.slug,
		"id=" + e.storedID,
	}
	if display != "" {
		parts = append(parts, "display="+display)
	}
	parts = append(parts, "path="+e.path)
	return strings.Join(parts, " ")
}

func scopeOf(e *entity) string {
	if e.scope == "" {
		return "active"
	}
	return e.scope
}

// findEntityFormConflicts returns conflict errors for slugs present as both flat
// and folder entities in directory. Matches find_entity_form_conflicts.
func findEntityFormConflicts(directory, scope string) []string {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	flat := map[string]bool{}
	folder := map[string]bool{}
	for _, ent := range entries {
		name := ent.Name()
		full := filepath.Join(directory, name)
		st, err := os.Stat(full)
		if err != nil {
			continue
		}
		if st.Mode().IsRegular() {
			if strings.HasSuffix(name, ".md") && name != "README.md" && !strings.HasPrefix(name, ".") {
				flat[strings.TrimSuffix(name, ".md")] = true
			}
		} else if st.IsDir() {
			if !reservedSubdirs[name] && !strings.HasPrefix(name, ".") {
				if isRegularFile(filepath.Join(full, "index.md")) {
					folder[name] = true
				}
			}
		}
	}

	var conflicts []string
	for slug := range flat {
		if folder[slug] {
			conflicts = append(conflicts, slug)
		}
	}
	sort.Strings(conflicts)

	workflowField := directory
	if scope == "archived" {
		workflowField = filepath.Dir(directory)
	}
	var errs []string
	for _, slug := range conflicts {
		folderPath := filepath.Join(directory, slug, "index.md")
		flatPath := filepath.Join(directory, slug+".md")
		errs = append(errs, fmt.Sprintf(
			"Error: flat/folder conflict: workflow=%s scope=%s slug=%s flat_path=%s folder_path=%s",
			workflowField, scope, slug, flatPath, folderPath))
	}
	return errs
}

// kebabSuggestion mirrors _kebab_suggestion.
func kebabSuggestion(name string) string {
	lowered := strings.ToLower(name)
	replaced := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(lowered, "-")
	return strings.Trim(replaced, "-")
}

// validateWorkflowStageNames returns errors for README stage names violating the
// dispatch-name regex. Matches validate_workflow_stage_names.
func validateWorkflowStageNames(definitionDir string) []string {
	readme := filepath.Join(definitionDir, "README.md")
	if !fileExists(readme) {
		return nil
	}
	states := parseStagesBlock(readme)
	if len(states) == 0 {
		return nil
	}
	var errs []string
	for _, s := range states {
		name := s.name
		if name == "" || stageNameRe.MatchString(name) {
			continue
		}
		suggestion := kebabSuggestion(name)
		if suggestion == "" {
			suggestion = "a-kebab-name"
		}
		errs = append(errs, fmt.Sprintf(
			"workflow '%s': stage name '%s' must match ^[a-z0-9][a-z0-9-]*[a-z0-9]$; rename to '%s' or similar",
			definitionDir, name, suggestion))
	}
	return errs
}

// validateWorkflow returns validation error lines for active + archived
// entities. Matches validate_workflow. definitionDir is the README root and
// entityDir the entity root (equal in this stage).
func validateWorkflow(definitionDir, entityDir, idStyle string, stderr io.Writer) []string {
	var errs []string
	errs = append(errs, findEntityFormConflicts(entityDir, "active")...)
	errs = append(errs, findEntityFormConflicts(filepath.Join(entityDir, "_archive"), "archived")...)
	errs = append(errs, validateWorkflowStageNames(definitionDir)...)

	entities := activeAndArchivedEntities(entityDir, stderr)
	var sdDisplay map[string]string
	if idStyle == "sd-b32" {
		sdDisplay = computeSDB32DisplayIDs(entities)
	}

	switch idStyle {
	case "sequential":
		errs = append(errs, validateSequential(entities, entityDir)...)
	case "slug":
		errs = append(errs, validateSlug(entities, entityDir)...)
	case "sd-b32":
		errs = append(errs, validateSDB32(entities, entityDir, sdDisplay)...)
	}
	return errs
}

func validateSequential(entities []*entity, workflowDir string) []string {
	var errs []string
	byID := map[string][]*entity{}
	var order []string
	for _, e := range entities {
		id := e.storedID
		if id == "" {
			errs = append(errs, entityEvidence(e, workflowDir, "missing required id", ""))
			continue
		}
		if !isDigits(id) {
			errs = append(errs, entityEvidence(e, workflowDir, "non-numeric sequential id", ""))
			continue
		}
		if _, ok := byID[id]; !ok {
			order = append(order, id)
		}
		byID[id] = append(byID[id], e)
	}
	for _, id := range order {
		group := byID[id]
		if len(group) < 2 || !anyActive(group) {
			continue
		}
		for _, e := range group {
			errs = append(errs, entityEvidence(e, workflowDir, "duplicate id", ""))
		}
	}
	return errs
}

func validateSlug(entities []*entity, workflowDir string) []string {
	var errs []string
	seen := map[string]*entity{}
	for _, e := range entities {
		effective := e.slug
		e.displayID = effective
		if first, ok := seen[effective]; ok {
			errs = append(errs, entityEvidence(first, workflowDir, "duplicate effective id", effective))
			errs = append(errs, entityEvidence(e, workflowDir, "duplicate effective id", effective))
		} else {
			seen[effective] = e
		}
	}
	return errs
}

func validateSDB32(entities []*entity, workflowDir string, sdDisplay map[string]string) []string {
	var errs []string
	byID := map[string][]*entity{}
	var order []string
	for _, e := range entities {
		id := e.storedID
		display := sdDisplay[id]
		if id == "" {
			errs = append(errs, entityEvidence(e, workflowDir, "missing required id", display))
			continue
		}
		if !(isValidSDB32ID(id) || isDigits(id)) {
			errs = append(errs, entityEvidence(e, workflowDir, "invalid sd-b32 stored id", display))
			continue
		}
		if _, ok := byID[id]; !ok {
			order = append(order, id)
		}
		byID[id] = append(byID[id], e)
	}
	for _, id := range order {
		group := byID[id]
		if len(group) < 2 || !anyActive(group) {
			continue
		}
		display := sdDisplay[id]
		for _, e := range group {
			errs = append(errs, entityEvidence(e, workflowDir, "duplicate sd-b32 stored id", display))
		}
	}
	return errs
}

func anyActive(group []*entity) bool {
	for _, e := range group {
		if e.scope == "active" {
			return true
		}
	}
	return false
}
