// ABOUTME: build assembles structured dispatch JSON from stdin + workflow README
// ABOUTME: + entity file, matching the vendored claude-team build oracle (non-_mods).
package dispatch

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
	"github.com/spacedock-dev/spacedock/internal/status"
)

const (
	schemaVersion   = 2
	nameMaxLen      = 200
	modelEnumList   = "must be one of: sonnet, opus, haiku"
	dispatchFileDir = "/tmp/spacedock-dispatch"
)

// buildRequiredFields are the stdin keys that must be present and non-null.
var buildRequiredFields = []string{"schema_version", "entity_path", "workflow_dir", "stage", "checklist"}

// namePattern is the dispatch-name regex derived worker names must match.
var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// modelEnum is the Agent-schema model enum declared values are validated against.
var modelEnum = map[string]bool{"sonnet": true, "opus": true, "haiku": true}

// buildOutput is the stdout JSON envelope. Field order is the emission order
// (insertion order in the oracle): schema_version, subagent_type, description,
// fetch_commands, dispatch_file_path, prompt, model, then name / team_name in
// team mode only. Model is a *string so an unresolved model serializes as the
// JSON literal null; Name / TeamName are *string with omitempty so bare-mode
// dispatches omit the keys entirely (absent, not null).
type buildOutput struct {
	SchemaVersion int      `json:"schema_version"`
	SubagentType  string   `json:"subagent_type"`
	Description   string   `json:"description"`
	FetchCommands []string `json:"fetch_commands"`
	DispatchFile  string   `json:"dispatch_file_path"`
	Prompt        string   `json:"prompt"`
	Model         *string  `json:"model"`
	Name          *string  `json:"name,omitempty"`
	TeamName      *string  `json:"team_name,omitempty"`
}

// buildError prints `error: {msg}` to stderr and returns code (1 by default).
func buildError(stderr io.Writer, code int, format string, a ...any) int {
	fmt.Fprintf(stderr, "error: "+format+"\n", a...)
	return code
}

// runBuild reads a dispatch request as JSON on stdin and assembles the dispatch
// envelope on stdout plus the dispatch-file body written to a deterministic
// path. Scoped to non-_mods workflows: it emits the single show-stage-def fetch
// line and never the standing-teammate fetch line. Matches cmd_build minus the
// _mods/standing branch (deferred to the sibling claude-runtime-segregation).
func runBuild(probe claudeteam.TeamStateProbe, workflowDir string, stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return buildError(stderr, 1, "failed to read stdin: %s", err)
	}

	// Classify the top-level value the way the oracle does (json.loads then
	// isinstance(inp, dict)): invalid JSON → "invalid JSON on stdin"; a valid
	// non-object top-level (null, array, scalar) → "stdin must be a JSON object".
	// A bare-map decode cannot tell these apart — decoding JSON null into a map
	// succeeds with a nil map, masking the non-object case as a missing field.
	var top interface{}
	if err := json.Unmarshal(raw, &top); err != nil {
		return buildError(stderr, 1, "invalid JSON on stdin: %s", err)
	}
	if _, ok := top.(map[string]interface{}); !ok {
		return buildError(stderr, 1, "stdin must be a JSON object")
	}

	// Distinguish present-but-null from absent (the required-field rule fires for
	// both), so decode into a raw-message map for typed field access.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return buildError(stderr, 1, "invalid JSON on stdin: %s", err)
	}

	// Rule 1: Required fields present and non-null.
	for _, field := range buildRequiredFields {
		v, ok := fields[field]
		if !ok || isJSONNull(v) {
			return buildError(stderr, 1, "missing required field '%s'", field)
		}
	}

	// Rule 2: Schema version supported (CLEAN-BREAK: v1 rejected). The oracle
	// compares the parsed value against 2; a non-integer or wrong version is
	// rejected with exit 2 and the value rendered as the oracle renders it.
	if !isSchemaVersion(fields["schema_version"]) {
		return buildError(stderr, 2,
			"unsupported input schema_version %s, schema_version: %d required",
			renderSchemaVersion(fields["schema_version"]), schemaVersion)
	}

	entityPath := jsonString(fields["entity_path"])
	stage := jsonString(fields["stage"])
	teamName := optString(fields, "team_name")
	feedbackContext := optString(fields, "feedback_context")
	scopeNotes := optString(fields, "scope_notes")
	bareMode := optBool(fields, "bare_mode")
	isFeedbackReflow := optBool(fields, "is_feedback_reflow")

	// FO bootstrap discipline: a bare_mode dispatch with no recent TeamCreate
	// evidence on disk gets an advisory stderr warning (exit stays 0). The evidence
	// read and the warning text both live in the Claude seam; HOME resolution stays
	// generic here. A nil probe (a non-Claude host) emits no advisory — the warning
	// is Claude-specific bootstrap advice, host-neutral by absence.
	if bareMode && probe != nil {
		// HOME resolution stays generic (plain env, no ~/.claude read); the probe
		// owns the team-state read. An unset HOME yields "", on which the probe
		// reports no recent evidence and the advisory fires — the pre-seam behavior.
		if _, _, recent := probe(os.Getenv("HOME"), time.Now()); !recent {
			claudeteam.BareModeAdvisory(stderr)
		}
	}

	// Rule 12: entity_path must be project-root, not worktree-absolute.
	if strings.Contains(entityPath, "/.worktrees/") || strings.HasPrefix(entityPath, ".worktrees/") {
		return buildError(stderr, 1,
			"entity_path must be a project-root absolute path; got worktree path '%s'. "+
				"Pass the project-root location (e.g. '/repo/docs/plans/{slug}.md'), not the worktree copy. "+
				"The helper derives the worktree read target internally.", entityPath)
	}

	// Rule 9: Checklist non-empty (a non-list collapses to the same message).
	checklist, ok := jsonStringList(fields["checklist"])
	if !ok || len(checklist) == 0 {
		return buildError(stderr, 1, "checklist must not be empty")
	}

	// Rule 10: Entity file readable.
	if !isFile(entityPath) {
		return buildError(stderr, 1, "entity file not readable at '%s'", entityPath)
	}

	// Absolutize workflowDir against the process cwd once, so every downstream
	// join — README path, splitRootStateCheckout, the fetch line's --workflow-dir,
	// and the state-commit guidance — inherits an absolute, cwd-independent base.
	// A worktree worker runs with its cwd inside .worktrees/…, where a relative
	// emitted `git -C docs/dev/.spacedock-state` resolves nowhere; absolutizing
	// here makes both halves of the emitted state-commit command absolute.
	if abs, err := filepath.Abs(workflowDir); err == nil {
		workflowDir = abs
	}

	// Rule 11: Workflow README readable.
	readmePath := filepath.Join(workflowDir, "README.md")
	if !isFile(readmePath) {
		return buildError(stderr, 1, "workflow README not found at '%s'", readmePath)
	}

	// Parse workflow stages + defaults.
	stages, stageDefaults := status.ParseStagesWithDefaults(readmePath)
	if stages == nil {
		return buildError(stderr, 1, "no stages block found in %s", readmePath)
	}

	stageIdx := -1
	for i, s := range stages {
		if s.Name == stage {
			stageIdx = i
			break
		}
	}
	// Rule 3: Stage exists in workflow.
	if stageIdx < 0 {
		return buildError(stderr, 1, "stage '%s' not found in %s", stage, readmePath)
	}
	stageMeta := stages[stageIdx]

	// Resolve effective_model with precedence stage > defaults > null. Validate
	// any declared value against the enum loudly, stage before defaults.
	stageModel, stageModelSet := stageMeta.Model()
	defaultsModel, defaultsModelSet := stageDefaults["model"]
	if stageModelSet && !modelEnum[stageModel] {
		return buildError(stderr, 1,
			"invalid model for stages.states[%d].model: '%s' — %s",
			stageIdx, stageModel, modelEnumList)
	}
	if defaultsModelSet && !modelEnum[defaultsModel] {
		return buildError(stderr, 1,
			"invalid model for stages.defaults.model: '%s' — %s",
			defaultsModel, modelEnumList)
	}

	var effectiveModel *string
	modelSource := "null"
	if stageModelSet {
		m := stageModel
		effectiveModel = &m
		modelSource = "stage"
	} else if defaultsModelSet {
		m := defaultsModel
		effectiveModel = &m
		modelSource = "defaults"
	}
	if effectiveModel != nil {
		fmt.Fprintf(stderr,
			"[build] effective_model=%s (from %s) → Agent model=%s\n",
			*effectiveModel, modelSource, *effectiveModel)
	}

	// Rule 4: Stickiness — route on the entity's stamped worktree: field, not the
	// next stage's declared mode.
	entityFields := status.ParseFrontmatter(entityPath)
	entityTitle := entityFields["title"]
	entityWorktree := strings.TrimSpace(entityFields["worktree"])

	var worktreePath, gitRoot string
	if entityWorktree != "" {
		gitRoot = status.FindGitRoot(workflowDir)
		// os.path.join (pyJoin) lets an absolute worktree value win, matching the
		// oracle (claude-team:329). filepath.Join would graft an absolute value
		// under gitRoot and double the path, missing the existing worktree dir —
		// the FO stamps absolute worktree: values on live entities.
		worktreePath = pyJoin(gitRoot, entityWorktree)
		if info, err := os.Stat(worktreePath); err != nil || !info.IsDir() {
			return buildError(stderr, 1, "worktree path '%s' does not exist", worktreePath)
		}
	} else if stageMeta.Worktree {
		return buildError(stderr, 1, "worktree stage '%s' but entity has no worktree path", stage)
	}

	// Split-root: the README declares a state: checkout, so a worktree stage
	// isolates CODE only — the entity body stays at the FO-passed entity_path.
	// stateCheckout is the resolved absolute state-checkout dir (workflowDir/<state>),
	// the git repo where the entity body lives; "" when the workflow is single-root.
	stateCheckout := splitRootStateCheckout(workflowDir)
	splitRoot := stateCheckout != ""

	// Rule 5: Feedback context required for feedback reflow.
	if isFeedbackReflow && feedbackContext == "" {
		return buildError(stderr, 1,
			"dispatching to feedback target stage '%s' but feedback_context is missing", stage)
	}

	// Rule 8: Team name non-empty in team mode.
	if !bareMode && teamName == "" {
		return buildError(stderr, 1, "team mode requires team_name")
	}

	// Rule 6: subagent_type from the stage agent field.
	subagentType := "spacedock:ensign"
	if agent, ok := stageMeta.Agent(); ok {
		subagentType = agent
	}

	// Derive worker_key, slug, and name (slug-not-stem via EntitySlug).
	workerKey := strings.ReplaceAll(subagentType, ":", "-")
	slug := status.EntitySlug(entityPath)
	derivedName := fmt.Sprintf("%s-%s-%s", workerKey, slug, stage)

	// Rule 7: Name length and safety.
	if len(derivedName) > nameMaxLen {
		return buildError(stderr, 1, "derived name '%s' exceeds %d characters", derivedName, nameMaxLen)
	}
	if !namePattern.MatchString(derivedName) {
		return buildError(stderr, 1,
			"derived name '%s' contains invalid characters: "+
				"stage name '%s' must match %s (kebab-case "+
				"lowercase letters, digits, and hyphens only). "+
				"Run `status --validate` against the workflow to surface the same "+
				"stage-name error upstream of dispatch.",
			derivedName, stage, namePattern.String())
	}

	// Extract the stage subsection (validates the README heading parses).
	stageSubsection, err := extractStageSubsection(readmePath, stage)
	if err != nil {
		if she, ok := err.(*stageHeadingError); ok {
			return buildError(stderr, 1, "%s", she.msg)
		}
		return buildError(stderr, 1, "%s", err)
	}
	if stageSubsection == "" {
		return buildError(stderr, 1, "stage '%s' heading not found in %s", stage, readmePath)
	}

	// --- Prompt assembly ---
	var parts []string

	// 0. Operating-contract first-action directive.
	parts = append(parts,
		"## First action\n"+
			"\n"+
			"Before anything else, invoke your operating contract:\n"+
			"\n"+
			"    Skill(skill=\"spacedock:ensign\")\n"+
			"\n"+
			"This loads the shared ensign discipline (stage-report format, BashOutput "+
			"polling, worktree ownership, completion signal protocol). The call is safe "+
			"to call more than once; if the agent-definition preload ever starts "+
			"working, calling it again is a no-op (the skill content is re-loaded but "+
			"has no behavioral effect). Do not paraphrase; call the tool.\n")

	// 1. Header.
	parts = append(parts, fmt.Sprintf("You are working on: %s\n\nStage: %s\n", entityTitle, stage))

	// 2. Stage definition — replaced by the show-stage-def fetch line. The native
	// fetch line targets `spacedock dispatch show-stage-def` so the dispatch path
	// stays Python-free (the one intentional divergence from the oracle).
	fetchCommands := []string{
		fmt.Sprintf("spacedock dispatch show-stage-def --workflow-dir %s --stage %s",
			shlexQuote(workflowDir), shlexQuote(stage)),
	}

	// 3. Worktree instructions (conditional). Under split root the state-commit
	// guidance applies to every stage, worktree or not: the worktree branch
	// prepends CODE-directory/branch lines, a non-worktree stage emits the
	// standalone guidance. The guidance carries the resolved absolute state
	// checkout (workflowDir/<state>, the git repo holding the entity body) and
	// entity path — never literal {state_checkout}/{entity_path} brace tokens.
	var worktreeEntityPath string
	if worktreePath != "" {
		branch := fmt.Sprintf("%s/%s", workerKey, slug)
		if splitRoot {
			parts = append(parts, fmt.Sprintf(
				"Your working directory for CODE is %s\n"+
					"All CODE reads and writes MUST use paths under %s.\n"+
					"Your git branch for CODE is %s. All CODE commits MUST be on "+
					"this branch in the worktree. Do NOT switch branches or commit "+
					"code to main.\n"+
					"%s",
				worktreePath, worktreePath, branch,
				stateCommitGuidance(stateCheckout, entityPath)))
		} else {
			parts = append(parts, fmt.Sprintf(
				"Your working directory is %s\n"+
					"All file reads and writes MUST use paths under %s.\n"+
					"Your git branch is %s. All commits MUST be on this branch. "+
					"Do NOT switch branches or commit to main.\n",
				worktreePath, worktreePath, branch))
		}
	} else if splitRoot {
		parts = append(parts, stateCommitGuidance(stateCheckout, entityPath))
	}

	// 4. Entity-read instruction. Under split root the entity lives in the state
	// checkout; a non-split worktree stage rewrites the path into the worktree.
	if worktreePath != "" && !splitRoot {
		entityRel := pyRelpath(entityPath, gitRoot)
		worktreeEntityPath = pyJoin(worktreePath, entityRel)
		parts = append(parts, fmt.Sprintf(
			"Read the entity file at %s for the full spec. It contains:\n", worktreeEntityPath))
	} else {
		parts = append(parts, fmt.Sprintf(
			"Read the entity file at %s for the current spec.\n", entityPath))
	}

	// 6. Feedback context (conditional).
	if feedbackContext != "" {
		parts = append(parts, fmt.Sprintf("### Feedback from prior review\n\n%s\n", feedbackContext))
	}

	// 7. Scope notes (conditional).
	if scopeNotes != "" {
		parts = append(parts, scopeNotes+"\n")
	}

	// 8. Completion checklist + Summary slot.
	checklistText := strings.Join(checklist, "\n")
	parts = append(parts, fmt.Sprintf(
		"### Completion checklist\n\n%s\n\n### Summary\n{brief description of what was accomplished}\n",
		checklistText))

	// 9. Standing-teammate fetch line — appended only when at least one declared
	// standing teammate exists for this workflow. Bare-mode (empty team_name) and
	// zero-standing dispatches omit the line, so show-standing never runs where it
	// would render nothing. The enumeration is runtime-neutral; the native fetch
	// line targets `spacedock dispatch show-standing` (the same documented
	// claude-team→spacedock dispatch rewrite as the show-stage-def line).
	if len(EnumerateDeclaredStandingTeammates(workflowDir, teamName)) > 0 {
		fetchCommands = append(fetchCommands,
			fmt.Sprintf("spacedock dispatch show-standing --workflow-dir %s", shlexQuote(workflowDir)))
	}

	// Emit the `### Fetch commands` block.
	fetchLines := []string{"### Fetch commands", ""}
	for _, cmd := range fetchCommands {
		fetchLines = append(fetchLines, "    "+cmd)
	}
	parts = append(parts, strings.Join(fetchLines, "\n"))

	// 10. Completion signal (team mode only).
	if !bareMode && teamName != "" {
		entityFileRef := entityPath
		if worktreePath != "" && !splitRoot {
			entityFileRef = worktreeEntityPath
		}
		parts = append(parts, fmt.Sprintf(
			"\n\n### Completion Signal\n\n"+
				"When you finish (after all commits and stage report writes are done), "+
				"your last action MUST be:\n\n"+
				"    SendMessage(to=\"team-lead\", message=\"Done: %s completed "+
				"%s. Report written to %s.\")\n\n"+
				"**If you are the first officer forwarding this prompt to Agent():** copy "+
				"the entire block above into `Agent(prompt=...)` character-for-character. "+
				"Do NOT paraphrase `SendMessage(to=\"team-lead\", ...)` as \"SendMessage with "+
				"to='team-lead'\" or any other English rewrite — the parenthesis-equals "+
				"syntax is the literal call the ensign must emit, not a description of one.",
			entityTitle, stage, entityFileRef))
	}

	dispatchBody := strings.Join(parts, "\n")

	// v2 file-pointer: write the body to a deterministic path; emit a tiny prompt
	// the ensign Reads on first action.
	dispatchFilePath := filepath.Join(dispatchFileDir, derivedName+".md")
	if err := os.MkdirAll(dispatchFileDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "dispatch_file_write_failed: %s: %s\n", dispatchFilePath, err)
		return 1
	}
	if err := os.WriteFile(dispatchFilePath, []byte(dispatchBody), 0o644); err != nil {
		fmt.Fprintf(stderr, "dispatch_file_write_failed: %s: %s\n", dispatchFilePath, err)
		return 1
	}

	prompt := fmt.Sprintf(
		"Skill(skill=\"spacedock:ensign\"); then Read %s and treat its content as your assignment.",
		dispatchFilePath)

	out := buildOutput{
		SchemaVersion: schemaVersion,
		SubagentType:  subagentType,
		Description:   fmt.Sprintf("%s: %s", entityTitle, stage),
		FetchCommands: fetchCommands,
		DispatchFile:  dispatchFilePath,
		Prompt:        prompt,
		Model:         effectiveModel,
	}
	if !bareMode {
		out.Name = &derivedName
		if teamName != "" {
			out.TeamName = &teamName
		}
	}

	return emitBuildJSON(stdout, out)
}

// stateCommitGuidance is the split-root state-commit instruction, shared by the
// worktree and non-worktree branches so the wording lives in one place. It
// substitutes the resolved absolute state checkout and entity paths into the
// path-scoped commit command — never literal {state_checkout}/{entity_path}
// brace tokens — and carries the concurrency-safe "never a bare git add -A"
// rule that governs every split-root stage.
func stateCommitGuidance(stateCheckout, entityPath string) string {
	return fmt.Sprintf(
		"This workflow is split-root: the entity body and your stage report "+
			"live in the shared state checkout, not alongside the code. Write and "+
			"commit them to the state checkout at the entity path below. Commit "+
			"state path-scoped — "+
			"`git -C %s add %s && "+
			"git -C %s commit -m \"...\" -- %s` — "+
			"never a bare `git add -A` or bare `git commit` in the state "+
			"checkout (a bare commit cross-attributes a concurrent writer's "+
			"staged entity). Retry on index.lock contention after a short wait.\n",
		stateCheckout, entityPath, stateCheckout, entityPath)
}

// emitBuildJSON writes out as two-space-indented JSON with HTML escaping off and
// a trailing newline, matching Python json.dumps(indent=2) followed by print().
func emitBuildJSON(stdout io.Writer, out buildOutput) int {
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	// Encoder.Encode appends the trailing newline print() also emits.
	if err := enc.Encode(out); err != nil {
		return 1
	}
	return 0
}
