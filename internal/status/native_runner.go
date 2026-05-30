// ABOUTME: NativeRunner backs the Runner seam with a native Go implementation
// ABOUTME: of the status tool, reproducing the oracle's observable contract.
package status

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// NativeRunner is the native-Go status runner. It satisfies the same Runner
// interface as VendorRunner; callers select it without any contract change.
type NativeRunner struct{}

var _ Runner = (*NativeRunner)(nil)

// Run parses req.Args and dispatches to the matching subcommand, writing to
// req.Stdout/Stderr and returning the exit code. The exit domain is {0,1}; a
// usage error is exit 1 (never 2). The returned err is always nil — the native
// runner can always run; failures are reported as exit code 1.
func (r *NativeRunner) Run(ctx context.Context, req Request) (int, error) {
	e := envFromSlice(req.Env)
	code := dispatch(req.Args, req.Dir, e, req.Stdin, req.Stdout, req.Stderr)
	return code, nil
}

// errExit prints "Error: <msg>" to stderr and returns 1, the oracle's
// usage/parse-error shape.
func errExit(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "Error: %s\n", msg)
	return 1
}

// dispatch is the argv-driven entry point mirroring the oracle's main().
func dispatch(args []string, dir string, e env, stdin io.Reader, stdout, stderr io.Writer) int {
	// --discover (incompatible with all other flags).
	if contains(args, "--discover") {
		return runDiscover(args, dir, stderr, stdout)
	}

	workflowDir, args, err := parseWorkflowDir(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}

	rootPath, err := parseRootArg(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}

	// The native runner requires --workflow-dir (there is no embedded-script
	// dirname(__file__) fallback). PIPELINE_DIR remains an accepted source.
	pipelineDir := workflowDir
	if pipelineDir == "" {
		pipelineDir = e.get("PIPELINE_DIR")
	}

	setResult, err := parseSetArgs(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	archiveSlug, err := parseArchiveArg(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	resolveRef, err := parseSingleArg(args, "--resolve", "reference")
	if err != nil {
		return errExit(stderr, err.Error())
	}
	shortIDRef, err := parseSingleArg(args, "--short-id", "reference")
	if err != nil {
		return errExit(stderr, err.Error())
	}
	newSlug, err := parseNewArg(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	explicitFields, allFieldsFlag, err := parseFieldsOptions(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	idSeed, idActor, idMaterialFlags, err := parseIDMaterialOptions(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}

	includeArchive := contains(args, "--archived")
	showNext := contains(args, "--next")
	showNextID := contains(args, "--next-id")
	showBoot := contains(args, "--boot")
	showValidate := contains(args, "--validate")
	hasFieldsFlag := explicitFields != nil || allFieldsFlag

	// --new accepts --id-seed/--id-actor too, so only gate them when neither
	// --next-id nor --new is present.
	if len(idMaterialFlags) > 0 && !showNextID && newSlug == "" {
		return errExit(stderr, "--id-seed and --id-actor can only be used with --next-id")
	}

	if showBoot {
		if showNext {
			return errExit(stderr, "--boot is incompatible with --next")
		}
		if showNextID {
			return errExit(stderr, "--boot is incompatible with --next-id")
		}
		if includeArchive {
			return errExit(stderr, "--boot is incompatible with --archived")
		}
		if hasFieldsFlag {
			return errExit(stderr, "--boot is incompatible with --fields / --all-fields")
		}
	}

	whereFilters, err := parseWhereFilters(args)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	if showBoot && len(whereFilters) > 0 {
		return errExit(stderr, "--boot is incompatible with --where")
	}

	roots := resolveRoots(pipelineDir)

	// --new atomic create.
	if newSlug != "" {
		return runNew(roots, newSlug, idSeed, idActor, idMaterialFlags, stdin, stdout, stderr, e)
	}

	if showNextID {
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
		if len(whereFilters) > 0 {
			incompatible = append(incompatible, "--where")
		}
		if hasFieldsFlag {
			incompatible = append(incompatible, "--fields/--all-fields")
		}
		if archiveSlug != "" {
			incompatible = append(incompatible, "--archive")
		}
		if setResult != nil {
			incompatible = append(incompatible, "--set")
		}
		if resolveRef != "" {
			incompatible = append(incompatible, "--resolve")
		}
		if showValidate {
			incompatible = append(incompatible, "--validate")
		}
		if len(incompatible) > 0 {
			return errExit(stderr, "--next-id cannot be combined with "+strings.Join(incompatible, ", "))
		}

		idStyle, err := workflowIDStyle(roots.definitionDir)
		if err != nil {
			return errExit(stderr, err.Error())
		}
		if rc := failOnValidationErrors(roots, idStyle, stderr); rc != 0 {
			return rc
		}
		id, err := computeNextID(roots.definitionDir, roots.entityDir, idStyle, idSeed, idActor, e, stderr)
		if err != nil {
			return errExit(stderr, err.Error())
		}
		fmt.Fprintln(stdout, id)
		return 0
	}

	if resolveRef != "" {
		var incompatible []string
		if showNext {
			incompatible = append(incompatible, "--next")
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
		if archiveSlug != "" {
			incompatible = append(incompatible, "--archive")
		}
		if setResult != nil {
			incompatible = append(incompatible, "--set")
		}
		if showValidate {
			incompatible = append(incompatible, "--validate")
		}
		if len(incompatible) > 0 {
			return errExit(stderr, "--resolve cannot be combined with "+strings.Join(incompatible, ", "))
		}
		if rootPath != "" {
			return resolveFromRootOrExit(rootPath, resolveRef, includeArchive, stdout, stderr)
		}
		return resolveReferenceOrExit(roots, resolveRef, includeArchive, stdout, stderr)
	}

	if shortIDRef != "" {
		var incompatible []string
		if showNext {
			incompatible = append(incompatible, "--next")
		}
		if showBoot {
			incompatible = append(incompatible, "--boot")
		}
		if showNextID {
			incompatible = append(incompatible, "--next-id")
		}
		if includeArchive {
			incompatible = append(incompatible, "--archived")
		}
		if len(whereFilters) > 0 {
			incompatible = append(incompatible, "--where")
		}
		if hasFieldsFlag {
			incompatible = append(incompatible, "--fields/--all-fields")
		}
		if archiveSlug != "" {
			incompatible = append(incompatible, "--archive")
		}
		if setResult != nil {
			incompatible = append(incompatible, "--set")
		}
		if showValidate {
			incompatible = append(incompatible, "--validate")
		}
		if rootPath != "" {
			incompatible = append(incompatible, "--root")
		}
		if len(incompatible) > 0 {
			return errExit(stderr, "--short-id cannot be combined with "+strings.Join(incompatible, ", "))
		}
		return printShortIDOrExit(roots, shortIDRef, stdout, stderr)
	}

	if rootPath != "" {
		return errExit(stderr, "--root is only supported with --discover or --resolve")
	}

	if archiveSlug != "" {
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
		if setResult != nil {
			incompatible = append(incompatible, "--set")
		}
		if showValidate {
			incompatible = append(incompatible, "--validate")
		}
		if len(incompatible) > 0 {
			return errExit(stderr, "--archive cannot be combined with "+strings.Join(incompatible, ", "))
		}

		resolved, rc := resolveMutationEntity(roots, archiveSlug, stderr)
		if rc != 0 {
			return rc
		}
		return runArchive(roots.entityDir, resolved.slug, contains(args, "--force"), stdout, stderr)
	}

	if setResult != nil {
		return runSet(roots, setResult, args, whereFilters, includeArchive, showNext, showBoot, showNextID, showValidate, hasFieldsFlag, stdout, stderr)
	}

	// Read paths (table / next / boot / validate).
	return runRead(roots, args, e, whereFilters, includeArchive, showNext, showBoot, showNextID, showValidate, explicitFields, allFieldsFlag, archiveSlug != "", setResult != nil, resolveRef != "", stdout, stderr)
}

// failOnValidationErrors prints validation errors to stderr and returns 1 when
// any exist, else 0. Matches fail_on_validation_errors.
func failOnValidationErrors(roots roots, idStyle string, stderr io.Writer) int {
	errs := validateWorkflow(roots.definitionDir, roots.entityDir, idStyle, stderr)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	return 0
}

// contains reports whether s contains v.
func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// resolveReferenceOrExit resolves a ref in one workflow and prints the resolve
// line, or fails. Matches resolve_reference_or_exit.
func resolveReferenceOrExit(roots roots, ref string, includeArchived bool, stdout, stderr io.Writer) int {
	idStyle, err := workflowIDStyle(roots.definitionDir)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	if rc := failOnValidationErrors(roots, idStyle, stderr); rc != 0 {
		return rc
	}
	res := resolveReferenceCandidates(roots.definitionDir, roots.entityDir, ref, includeArchived, idStyle, stderr)
	if res.status != "ok" {
		for _, e := range res.errors {
			fmt.Fprintln(stderr, e)
		}
		return 1
	}
	fmt.Fprintln(stdout, formatResolveLine(roots.entityDir, res.matches[0]))
	return 0
}

// printShortIDOrExit resolves REF across active+archived and prints its short
// display id. Matches print_short_id_or_exit.
func printShortIDOrExit(roots roots, ref string, stdout, stderr io.Writer) int {
	idStyle, err := workflowIDStyle(roots.definitionDir)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	if rc := failOnValidationErrors(roots, idStyle, stderr); rc != 0 {
		return rc
	}
	res := resolveReferenceCandidates(roots.definitionDir, roots.entityDir, ref, true, idStyle, stderr)
	if res.status != "ok" {
		for _, e := range res.errors {
			fmt.Fprintln(stderr, e)
		}
		return 1
	}
	entity := res.matches[0]
	switch idStyle {
	case "slug":
		fmt.Fprintln(stdout, entity.slug)
	case "sd-b32":
		all := activeAndArchivedEntities(roots.entityDir, stderr)
		display := computeSDB32DisplayIDs(all)
		if d, ok := display[entity.storedID]; ok {
			fmt.Fprintln(stdout, d)
		} else {
			fmt.Fprintln(stdout, entity.storedID)
		}
	default:
		fmt.Fprintln(stdout, entity.storedID)
	}
	return 0
}

// resolveMutationEntity resolves a mutation target in active scope, with
// archived read-only diagnostics. Matches resolve_mutation_entity. Returns
// (entity, 0) on success or (nil, 1) after printing the error.
func resolveMutationEntity(roots roots, ref string, stderr io.Writer) (*entity, int) {
	idStyle, err := workflowIDStyle(roots.definitionDir)
	if err != nil {
		errExit(stderr, err.Error())
		return nil, 1
	}
	res := resolveReferenceCandidates(roots.definitionDir, roots.entityDir, ref, false, idStyle, stderr)
	if res.status == "ok" {
		return res.matches[0], 0
	}

	archivedRes := resolveReferenceCandidates(roots.definitionDir, roots.entityDir, ref, true, idStyle, stderr)
	var archivedOnly []*entity
	for _, e := range archivedRes.matches {
		if e.scope == "archived" {
			archivedOnly = append(archivedOnly, e)
		}
	}
	if len(archivedOnly) > 0 && len(res.matches) == 0 {
		fmt.Fprintf(stderr, "Error: archived entity is read-only: %s\n", ref)
		for _, e := range archivedOnly {
			fmt.Fprintln(stderr, candidateLine(e))
		}
		return nil, 1
	}
	if res.status == "unknown" {
		fmt.Fprintf(stderr, "Error: entity not found: %s\n", ref)
		return nil, 1
	}
	for _, e := range res.errors {
		fmt.Fprintln(stderr, e)
	}
	return nil, 1
}

// runDiscover handles --discover. Matches the --discover branch of main().
func runDiscover(args []string, dir string, stderr, stdout io.Writer) int {
	incompatibleFlags := map[string]bool{
		"--boot": true, "--next": true, "--next-id": true, "--archived": true, "--where": true,
		"--set": true, "--archive": true, "--fields": true, "--all-fields": true, "--workflow-dir": true,
		"--validate": true, "--resolve": true, "--short-id": true, "--id-seed": true, "--id-actor": true,
	}
	var found map[string]bool = map[string]bool{}
	for _, a := range args {
		if incompatibleFlags[a] {
			found[a] = true
		}
	}
	if len(found) > 0 {
		names := make([]string, 0, len(found))
		for k := range found {
			names = append(names, k)
		}
		sortStrings(names)
		return errExit(stderr, "--discover is incompatible with "+strings.Join(names, ", "))
	}

	rootPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--root" {
			if i+1 >= len(args) {
				return errExit(stderr, "--root requires an argument")
			}
			rootPath = args[i+1]
			i++
		}
	}
	if rootPath == "" {
		out, err := runGit(dir, "rev-parse", "--show-toplevel")
		if err == nil {
			rootPath = strings.TrimSpace(out)
		} else {
			rootPath = dir
		}
	}
	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		return errExit(stderr, "--root path does not exist: "+rootPath)
	}
	for _, p := range discoverWorkflows(rootPath) {
		fmt.Fprintln(stdout, p)
	}
	return 0
}

// runGit runs git in dir and returns stdout, or an error on non-zero exit.
func runGit(dir string, args ...string) (string, error) {
	return runGitCmd(dir, args...)
}

// resolveFromRootOrExit is the multi-workflow --root resolver. Matches
// resolve_from_root_or_exit.
func resolveFromRootOrExit(rootPath, ref string, includeArchived bool, stdout, stderr io.Writer) int {
	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		return errExit(stderr, "--root path does not exist: "+rootPath)
	}

	workflowRef := ""
	innerRef := ref
	if idx := strings.Index(ref, "::"); idx >= 0 {
		workflowRef = ref[:idx]
		innerRef = ref[idx+2:]
	}

	workflows := discoverWorkflows(rootPath)
	if workflowRef != "" {
		var candidates []string
		if filepath.IsAbs(workflowRef) {
			if st, err := os.Stat(workflowRef); err == nil && st.IsDir() {
				candidates = []string{realpathOf(workflowRef)}
			}
		} else {
			for _, wf := range workflows {
				if filepath.Base(wf) == workflowRef {
					candidates = append(candidates, wf)
				}
			}
		}
		if len(candidates) != 1 {
			fmt.Fprintf(stderr, "Error: workflow qualifier is ambiguous or unknown: %s\n", workflowRef)
			for _, wf := range candidates {
				fmt.Fprintln(stderr, wf)
			}
			return 1
		}
		return resolveReferenceOrExit(resolveRoots(candidates[0]), innerRef, includeArchived, stdout, stderr)
	}

	type match struct {
		workflow string
		entity   *entity
	}
	var matches []match
	var hardErrors []string
	for _, workflow := range workflows {
		idStyle, err := workflowIDStyle(workflow)
		if err != nil {
			hardErrors = append(hardErrors, fmt.Sprintf("Error: %s: %s", workflow, err))
			continue
		}
		res := resolveReferenceCandidates(workflow, workflow, ref, includeArchived, idStyle, stderr)
		if res.status == "ok" {
			matches = append(matches, match{workflow, res.matches[0]})
		} else if res.status != "unknown" {
			for _, e := range res.errors {
				hardErrors = append(hardErrors, fmt.Sprintf("%s: %s", workflow, e))
			}
		}
	}

	if len(matches) == 1 && len(hardErrors) == 0 {
		fmt.Fprintln(stdout, formatResolveLine(matches[0].workflow, matches[0].entity))
		return 0
	}
	if len(matches) > 1 {
		fmt.Fprintf(stderr, "Error: multiple workflows match reference: %s\n", ref)
		for _, m := range matches {
			fmt.Fprintf(stderr, "%s %s\n", m.workflow, candidateLine(m.entity))
		}
		return 1
	}
	if len(hardErrors) > 0 {
		for _, e := range hardErrors {
			fmt.Fprintln(stderr, e)
		}
		return 1
	}
	fmt.Fprintf(stderr, "Error: unknown reference: %s\n", ref)
	return 1
}
