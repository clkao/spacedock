// ABOUTME: --set mutation flow, the read (table/next/boot/validate) flow, and
// ABOUTME: workflow discovery / git helpers backing the native runner.
package status

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// runSet handles the --set mutation flow with mod-block / merge-hook /
// terminal-transition guards and the field: old -> new narration. Matches the
// --set branch of main().
func runSet(roots roots, set *setUpdate, args []string, whereFilters []whereFilter,
	includeArchive, showNext, showBoot, showNextID, showValidate, hasFieldsFlag bool,
	stdout, stderr io.Writer) int {

	var incompatible []string
	if showNext {
		incompatible = append(incompatible, "--next")
	}
	if includeArchive {
		incompatible = append(incompatible, "--archived")
	}
	if showBoot {
		incompatible = append(incompatible, "--boot")
	}
	if showNextID {
		incompatible = append(incompatible, "--next-id")
	}
	if len(whereFilters) > 0 {
		incompatible = append(incompatible, "--where")
	}
	if hasFieldsFlag {
		incompatible = append(incompatible, "--fields/--all-fields")
	}
	if showValidate {
		incompatible = append(incompatible, "--validate")
	}
	if len(incompatible) > 0 {
		return errExit(stderr, "--set cannot be combined with "+strings.Join(incompatible, ", "))
	}

	force := contains(args, "--force")
	resolved, rc := resolveMutationEntity(roots, set.slug, stderr)
	if rc != 0 {
		return rc
	}
	slug := resolved.slug
	mainEntityPath := resolveEntityPath(roots.entityDir, slug, stderr)
	if mainEntityPath == "" {
		return errExit(stderr, "entity not found: "+slug)
	}
	// Single-root stage: no worktree overlay, entity path is the main path.
	entityPath := mainEntityPath

	currentFields := parseFrontmatter(entityPath)
	modBlock := strings.TrimSpace(currentFields["mod-block"])
	currentPR := strings.TrimSpace(currentFields["pr"])
	currentVerdict := strings.TrimSpace(currentFields["verdict"])
	clearingModBlock := false
	for _, u := range set.updates {
		if u.field == "mod-block" && u.hasValue && u.value == "" {
			clearingModBlock = true
		}
	}

	readme := filepath.Join(roots.definitionDir, "README.md")
	var stages []stage
	if fileExists(readme) {
		stages = parseStagesBlock(readme)
	}
	terminalNames := map[string]bool{}
	for _, s := range stages {
		if s.terminal {
			terminalNames[s.name] = true
		}
	}

	isTerminalUpdate := func() bool {
		for _, u := range set.updates {
			if u.field == "status" && u.hasValue && terminalNames[u.value] {
				return true
			}
			if u.field == "completed" || u.field == "verdict" {
				return true
			}
			if u.field == "worktree" && u.hasValue && u.value == "" {
				return true
			}
		}
		return false
	}

	postUpdatePR := currentPR
	for _, u := range set.updates {
		if u.field == "pr" {
			postUpdatePR = strings.TrimSpace(u.value)
		}
	}
	postUpdateVerdict := currentVerdict
	for _, u := range set.updates {
		if u.field == "verdict" {
			postUpdateVerdict = strings.TrimSpace(u.value)
		}
	}

	if (modBlock != "" || clearingModBlock) && !force {
		if isTerminalUpdate() {
			var reason string
			if modBlock != "" && !clearingModBlock {
				reason = fmt.Sprintf("pending mod-block (%s)", modBlock)
			} else if clearingModBlock {
				reason = fmt.Sprintf("combined mod-block clear with terminal transition (current mod-block='%s')", modBlock)
			} else {
				reason = "mod-block transition"
			}
			return errExit(stderr, fmt.Sprintf(
				"entity %s has %s. Clear mod-block in a separate --set call, or use --force.",
				slug, reason))
		}
	}

	if !force && isTerminalUpdate() && modBlock == "" && postUpdatePR == "" && postUpdateVerdict != "rejected" {
		mergeHooks := scanMods(roots.entityDir)["merge"]
		if len(mergeHooks) > 0 {
			return errExit(stderr, fmt.Sprintf(
				"entity %s cannot advance to terminal — workflow has merge hook(s) [%s] that have not run "+
					"(pr field is empty and mod-block is empty). Set mod-block=merge:%s and invoke the hook, or use --force to bypass.",
				slug, strings.Join(mergeHooks, ", "), mergeHooks[0]))
		}
	}

	if force && modBlock != "" {
		fmt.Fprintf(stderr, "Warning: --force overriding mod-block (%s) on entity %s\n", modBlock, slug)
	}

	resolvedFields, err := updateFrontmatter(entityPath, set.updates)
	if err != nil {
		return errExit(stderr, err.Error())
	}

	for _, field := range resolvedFields.keys() {
		val, _ := resolvedFields.get(field)
		oldValue := currentFields[field]
		fmt.Fprintf(stdout, "%s: %s -> %s\n", field, oldValue, val)
	}
	return 0
}

// runRead handles the table / --next / --boot / --validate read flows. Matches
// the tail of main() after the mutation branches.
func runRead(roots roots, args []string, e env, whereFilters []whereFilter,
	includeArchive, showNext, showBoot, showNextID, showValidate bool,
	explicitFields []string, allFieldsFlag bool,
	hasArchiveSlug, hasSet, hasResolve bool,
	stdout, stderr io.Writer) int {

	readme := filepath.Join(roots.definitionDir, "README.md")
	var stages []stage
	if fileExists(readme) {
		stages = parseStagesBlock(readme)
	}

	if showNext && stages == nil {
		return errExit(stderr, "README.md has no stages block. --next requires stage metadata.")
	}
	if showBoot && stages == nil {
		return errExit(stderr, "README.md has no stages block. --boot requires stage metadata.")
	}

	gitRoot := findGitRoot(roots.entityDir)
	idStyle, err := workflowIDStyle(roots.definitionDir)
	if err != nil {
		return errExit(stderr, err.Error())
	}

	if showValidate {
		var incompatible []string
		if showNext {
			incompatible = append(incompatible, "--next")
		}
		if includeArchive {
			incompatible = append(incompatible, "--archived")
		}
		if showBoot {
			incompatible = append(incompatible, "--boot")
		}
		if showNextID {
			incompatible = append(incompatible, "--next-id")
		}
		if len(whereFilters) > 0 {
			incompatible = append(incompatible, "--where")
		}
		if explicitFields != nil || allFieldsFlag {
			incompatible = append(incompatible, "--fields/--all-fields")
		}
		if hasArchiveSlug {
			incompatible = append(incompatible, "--archive")
		}
		if hasSet {
			incompatible = append(incompatible, "--set")
		}
		if len(incompatible) > 0 {
			return errExit(stderr, "--validate cannot be combined with "+strings.Join(incompatible, ", "))
		}
		errs := validateWorkflow(roots.definitionDir, roots.entityDir, idStyle, stderr)
		if len(errs) > 0 {
			for _, er := range errs {
				fmt.Fprintln(stderr, er)
			}
			return 1
		}
		fmt.Fprintln(stdout, "VALID")
		return 0
	}

	if rc := failOnValidationErrors(roots, idStyle, stderr); rc != 0 {
		return rc
	}

	allEntities := activeAndArchivedEntities(roots.entityDir, stderr)
	entities := scanEntities(roots.entityDir, stderr)
	if includeArchive {
		entities = append(entities, archiveEntities(roots.entityDir, stderr)...)
	}

	applyEffectiveIDs(allEntities, idStyle, allEntities)
	applyEffectiveIDs(entities, idStyle, allEntities)
	entities = applyFilters(entities, whereFilters)

	switch {
	case showBoot:
		if err := printBoot(stdout, entities, stages, roots.definitionDir, roots.entityDir, gitRoot, idStyle, e, stderr); err != nil {
			return 1
		}
	case showNext:
		extras := resolveExtraFields(entities, explicitFields, allFieldsFlag, defaultNextFields)
		printNextTable(stdout, entities, stages, extras)
	default:
		extras := resolveExtraFields(entities, explicitFields, allFieldsFlag, defaultStatusFields)
		printStatusTable(stdout, entities, stages, extras)
	}
	return 0
}

// discoverIgnoreDirs is the baseline prune set for --discover. Matches
// DISCOVER_IGNORE_DIRS.
var discoverIgnoreDirs = map[string]bool{
	".git": true, ".worktrees": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, "__pycache__": true, "tests": true,
}

// readGitignoreDirBasenames returns basenames of directory-pattern entries in
// {root}/.gitignore. Matches _read_gitignore_dir_basenames.
func readGitignoreDirBasenames(root string) map[string]bool {
	names := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return names
	}
	for _, line := range strings.Split(string(data), "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" || strings.HasPrefix(entry, "#") || strings.HasPrefix(entry, "!") {
			continue
		}
		if !strings.HasSuffix(entry, "/") {
			continue
		}
		base := filepath.Base(strings.TrimRight(entry, "/"))
		if base != "" {
			names[base] = true
		}
	}
	return names
}

// discoverWorkflows walks root and returns workflow dirs (README with a
// commissioned-by: spacedock@ frontmatter), realpath'd and sorted. Matches
// discover_workflows.
func discoverWorkflows(root string) []string {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	ignore := map[string]bool{}
	for k := range discoverIgnoreDirs {
		ignore[k] = true
	}
	for k := range readGitignoreDirBasenames(absRoot) {
		ignore[k] = true
	}

	seenReal := map[string]bool{}
	var results []string
	resultSet := map[string]bool{}

	var walk func(dir string)
	walk = func(dir string) {
		realDir := realpathOf(dir)
		if seenReal[realDir] {
			return
		}
		seenReal[realDir] = true

		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		readmePath := filepath.Join(dir, "README.md")
		if isRegularFile(readmePath) {
			fields := parseFrontmatter(readmePath)
			if strings.HasPrefix(fields["commissioned-by"], "spacedock@") {
				resolved := realpathOf(dir)
				if !resultSet[resolved] {
					resultSet[resolved] = true
					results = append(results, resolved)
				}
			}
		}
		for _, ent := range entries {
			// os.Stat follows symlinks, matching os.walk(followlinks=True).
			st, err := os.Stat(filepath.Join(dir, ent.Name()))
			if err != nil || !st.IsDir() {
				continue
			}
			if ignore[ent.Name()] {
				continue
			}
			walk(filepath.Join(dir, ent.Name()))
		}
	}
	walk(absRoot)

	sort.Strings(results)
	return results
}

// runGitCmd runs git in dir and returns stdout, or an error on failure.
func runGitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
