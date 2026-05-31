// ABOUTME: Skeleton BEHAVIORAL test of the dispatch->ensign->stage mechanical
// ABOUTME: cycle: real dispatch.Run build + scripted-ensign stage report + real git commit.
package ensigncycle

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/dispatch"
)

// The cycle exercises the deterministic seams on both sides of the LLM: the
// in-process dispatch.Run build (which dictates the mechanical contract — stage
// report shape, path-scoped commit form, completion-signal emit line) and the
// observable ensign->state outputs (the appended stage-report section + the
// path-scoped git commit). The LLM is stubbed by a Go scripted ensign that
// performs exactly the mechanical protocol the dispatch body prescribes.
//
// Every assertion targets the EMIT FORM anchored at its indentation, never a
// bare substring prose can satisfy. The spike proved a bare strings.Contains for
// the completion signal is fooled by the body's own "Do NOT paraphrase
// SendMessage(...)" warning prose — the prose-grep trap this entity exists to
// replace, reappearing inside the skeleton. The two negative guards below pin
// that the test goes red when the mechanical output breaks.

var (
	// completionEmit anchors the completion-signal emit line at its four-space
	// indentation. The bare prose warning ("Do NOT paraphrase SendMessage(...)")
	// is NOT four-space-indented and cannot satisfy this regex.
	completionEmit = regexp.MustCompile(`(?m)^    SendMessage\(to="team-lead", message="Done: `)
	// stageReportHeading anchors the appended stage-report section heading.
	stageReportHeading = regexp.MustCompile(`(?m)^## Stage Report: backlog$`)
	// doneMarker anchors a DONE checklist-accounting marker.
	doneMarker = regexp.MustCompile(`(?m)^- DONE:`)
	// checkboxBullet matches the forbidden markdown checkbox-bullet form the
	// stage-report protocol bans.
	checkboxBullet = regexp.MustCompile(`(?m)^- \[[ xX]\]`)
	// skillFirstAction anchors the operating-contract first-action emit line: the
	// indented `Skill(skill="spacedock:ensign")` call. This is the mechanical link
	// that loads the stage-report protocol (it lives in ensign-shared-core.md, not
	// the dispatch body), so asserting the emit form proves the body wires the
	// protocol — without grepping the protocol prose itself.
	skillFirstAction = regexp.MustCompile(`(?m)^    Skill\(skill="spacedock:ensign"\)$`)
	// fetchStageDef anchors the fetch-on-demand emit line that resolves the stage
	// definition (the other half of the protocol-loading wiring).
	fetchStageDef = regexp.MustCompile(`(?m)^    spacedock dispatch show-stage-def --workflow-dir `)
)

// cycleFixture is a staged dispatch->ensign->stage environment.
type cycleFixture struct {
	root       string
	entityPath string
	body       string // the dispatch body dispatch.Run wrote
}

// stageFixture stages a git-init'd root with a non-worktree workflow README and
// a flat {slug}.md entity in the initial (backlog) stage, then runs the real
// dispatch.Run build and reads back the dispatch body.
func stageFixture(t *testing.T) cycleFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "README.md"), readmeNonWorktree())
	entityPath := filepath.Join(root, "make-it-work.md")
	writeFile(t, entityPath, entityFixture())
	gitInit(t, root)

	stdin := mustJSON(t, map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "backlog",
		"checklist":      []string{"- Wire the seam", "- Prove it observably"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	})

	var stdout, stderr strings.Builder
	if code := dispatch.Run([]string{"build", "--workflow-dir", root},
		strings.NewReader(stdin), &stdout, &stderr); code != 0 {
		t.Fatalf("dispatch build exit=%d stderr=%s", code, stderr.String())
	}

	var out struct {
		DispatchFilePath string `json:"dispatch_file_path"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &out); err != nil {
		t.Fatalf("build stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if out.DispatchFilePath == "" {
		t.Fatalf("no dispatch_file_path in build stdout:\n%s", stdout.String())
	}
	bodyBytes, err := os.ReadFile(out.DispatchFilePath)
	if err != nil {
		t.Fatalf("read dispatch body %s: %v", out.DispatchFilePath, err)
	}

	return cycleFixture{root: root, entityPath: entityPath, body: string(bodyBytes)}
}

// scriptedEnsign is the LLM stand-in. It parses the checklist items out of the
// dispatch body's `### Completion checklist` block and appends a protocol-shaped
// `## Stage Report: backlog` section to the entity (one `- DONE:` per item plus a
// `### Summary`), then makes a path-scoped commit naming only the entity. markFn
// renders one checklist item into its stage-report bullet line; the positive run
// uses doneBullet, the negative-1 guard swaps in checkboxBullet to prove the test
// catches a protocol-illegal report.
func scriptedEnsign(t *testing.T, f cycleFixture, markFn func(item string) string) {
	t.Helper()
	items := parseChecklist(t, f.body)

	var report strings.Builder
	report.WriteString("\n## Stage Report: backlog\n\n")
	for _, item := range items {
		report.WriteString(markFn(item) + "\n")
		report.WriteString("  observable seam exercised\n")
	}
	report.WriteString("\n### Summary\n\nScripted ensign appended a protocol-shaped report.\n")

	appendFile(t, f.entityPath, report.String())

	rel, err := filepath.Rel(f.root, f.entityPath)
	if err != nil {
		t.Fatal(err)
	}
	gitCommitPathScoped(t, f.root, rel, "stage: backlog report")
}

// doneBullet renders the protocol-correct DONE marker for a checklist item.
func doneBullet(item string) string {
	return "- DONE: " + strings.TrimPrefix(item, "- ")
}

// TestEnsignCycleMechanicalOutputs is the skeleton: a real build seam + scripted
// ensign + real git commit + real entity append, asserting the mechanical
// outputs the dispatch body prescribes. It is the scaffold others extend — each
// ported Python behavior adds a fixture + a scripted-ensign step + an anchored
// mechanical assertion.
func TestEnsignCycleMechanicalOutputs(t *testing.T) {
	f := stageFixture(t)
	scriptedEnsign(t, f, doneBullet)

	entity := readFile(t, f.entityPath)

	// (a) the appended stage-report section has the protocol shape: heading, a
	// DONE accounting marker, a Summary, and NO checkbox-bullet form.
	if !stageReportHeading.MatchString(entity) {
		t.Errorf("entity missing anchored stage-report heading\n%s", entity)
	}
	if !doneMarker.MatchString(entity) {
		t.Errorf("entity missing anchored - DONE: marker\n%s", entity)
	}
	if !strings.Contains(entity, "### Summary") {
		t.Errorf("entity missing ### Summary\n%s", entity)
	}
	if checkboxBullet.MatchString(entity) {
		t.Errorf("entity contains forbidden checkbox-bullet stage-report markers\n%s", entity)
	}

	// (b) the commit landed and named ONLY the entity (path-scoped, no sibling
	// sweep) — mirrors concurrency_test.go's invariant at the cycle level.
	named := commitNameOnly(t, f.root)
	if len(named) != 1 || filepath.Base(named[0]) != "make-it-work.md" {
		t.Errorf("HEAD commit must name only the entity; named=%v", named)
	}

	// (c) the dispatch body carries the anchored completion-signal EMIT line —
	// NOT a bare Contains (per the spike lesson: the prose warning fools Contains).
	if !completionEmit.MatchString(f.body) {
		t.Errorf("dispatch body missing anchored completion-signal emit line\n%s", f.body)
	}

	// (d) the dispatch body wires the stage-report protocol via two anchored emit
	// lines — NOT a prose grep of the protocol text. The protocol shape itself
	// (`## Stage Report: {stage_name}`, DONE/SKIPPED/FAILED markers, the
	// path-scoped commit form) lives in ensign-shared-core.md, loaded by the
	// first-action Skill call; the fetch line resolves the stage definition. The
	// scripted ensign above consumed exactly that protocol when it appended the
	// report, so asserting the body emits the loading mechanism is the behavioral
	// link, not prose-grep.
	if !skillFirstAction.MatchString(f.body) {
		t.Errorf("dispatch body missing anchored Skill first-action emit line\n%s", f.body)
	}
	if !fetchStageDef.MatchString(f.body) {
		t.Errorf("dispatch body missing anchored show-stage-def fetch emit line\n%s", f.body)
	}
}

// TestEnsignCycleGoesRedOnBrokenOutput is the AC-2 verification: the two negative
// controls the spike used. Each turns the skeleton's mechanical assertions red.
// They run the real cycle through the SAME assertion helpers (via a faulty
// scripted ensign / a regressed body) and require the assertions to fail.
func TestEnsignCycleGoesRedOnBrokenOutput(t *testing.T) {
	// Negative-1: a scripted ensign that writes `- [x]` checkbox bullets instead
	// of `- DONE:` markers. The stage-report shape assertions must reject it: the
	// DONE marker is absent AND the forbidden checkbox form is present.
	t.Run("checkbox_bullets_instead_of_DONE", func(t *testing.T) {
		f := stageFixture(t)
		scriptedEnsign(t, f, func(item string) string {
			return "- [x] " + strings.TrimPrefix(item, "- ")
		})
		entity := readFile(t, f.entityPath)

		if doneMarker.MatchString(entity) {
			t.Fatal("expected NO - DONE: marker when ensign writes checkbox bullets")
		}
		if !checkboxBullet.MatchString(entity) {
			t.Fatal("expected the forbidden checkbox bullet to be detectable")
		}
	})

	// Negative-2: regress the dispatch body's emit line by renaming the
	// SendMessage(...) call to Notify(...). A bare strings.Contains would WRONGLY
	// still match (the body's "Do NOT paraphrase SendMessage(...)" prose warning
	// line survives the rename). The anchored emit regex must go red; the bare
	// Contains is shown to be the prose-grep trap.
	t.Run("regressed_emit_line_renamed_to_Notify", func(t *testing.T) {
		f := stageFixture(t)
		regressed := strings.Replace(f.body,
			`    SendMessage(to="team-lead", message="Done: `,
			`    Notify(to="team-lead", message="Done: `, 1)

		if completionEmit.MatchString(regressed) {
			t.Fatal("anchored emit regex must NOT match the regressed body")
		}
		// Demonstrate the trap: a bare Contains is fooled by the prose warning.
		if !strings.Contains(regressed, `SendMessage(to="team-lead"`) {
			t.Fatal("bare Contains expected to be fooled by the prose warning line " +
				"(this is the prose-grep trap the anchored regex avoids)")
		}
	})
}

// --- fixture + protocol helpers (self-contained; this package does not share
// the unexported test helpers in internal/dispatch) ---

// readmeNonWorktree is a minimal workflow README with a non-worktree initial
// stage, so the cycle exercises the flat-entity / no-worktree path.
func readmeNonWorktree() string {
	return "---\n" +
		"entity-type: task\n" +
		"id-style: slug\n" +
		"stages:\n" +
		"  defaults:\n" +
		"    worktree: false\n" +
		"    concurrency: 1\n" +
		"  states:\n" +
		"    - name: backlog\n" +
		"      initial: true\n" +
		"    - name: done\n" +
		"      terminal: true\n" +
		"---\n" +
		"# Fixture Workflow\n" +
		"\n" +
		"### backlog\n\nseed.\n\n- **Outputs:** x.\n\n" +
		"### done\n\nterm.\n"
}

// entityFixture is a flat entity in the initial stage with no worktree value.
func entityFixture() string {
	return "---\n" +
		"id: \"001\"\n" +
		"title: Make It Work\n" +
		"status: backlog\n" +
		"worktree:\n" +
		"---\n" +
		"# Make It Work\n\nBody.\n"
}

// parseChecklist extracts the checklist item lines from the dispatch body's
// `### Completion checklist` block (the lines between that heading and the next
// `### ` heading). It fails the test when the block is absent or empty — the
// scripted ensign cannot account for a checklist that did not arrive.
func parseChecklist(t *testing.T, body string) []string {
	t.Helper()
	lines := strings.Split(body, "\n")
	start := -1
	for i, ln := range lines {
		if ln == "### Completion checklist" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatalf("dispatch body has no '### Completion checklist' block\n%s", body)
	}
	var items []string
	for _, ln := range lines[start:] {
		if strings.HasPrefix(ln, "### ") {
			break
		}
		if strings.HasPrefix(ln, "- ") {
			items = append(items, ln)
		}
	}
	if len(items) == 0 {
		t.Fatalf("dispatch body checklist block is empty\n%s", body)
	}
	return items
}

// commitNameOnly returns the file paths HEAD's commit touched.
func commitNameOnly(t *testing.T, root string) []string {
	t.Helper()
	out := git(t, root, "show", "--name-only", "--pretty=format:", "HEAD")
	var paths []string
	for _, ln := range strings.Split(strings.TrimSpace(out), "\n") {
		if ln != "" {
			paths = append(paths, ln)
		}
	}
	return paths
}

// gitCommitPathScoped stages and commits ONLY rel, path-scoped — never a bare
// `git add -A` / bare `git commit` (which would sweep a concurrent writer's
// staged sibling). Mirrors the dispatch body's prescribed commit form.
func gitCommitPathScoped(t *testing.T, root, rel, msg string) {
	t.Helper()
	git(t, root, "add", "--", rel)
	git(t, root, "commit", "-q", "-m", msg, "--", rel)
}

// gitInit initializes a git repo at dir and lands an initial commit, so
// find_git_root resolves there and HEAD exists for the path-scoped commit.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "init", "-q")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "init")
}

// git runs a git subcommand at dir with pinned identity and returns stdout.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// writeFile writes content to path, creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// appendFile appends content to an existing file.
func appendFile(t *testing.T, path, content string) {
	t.Helper()
	fh, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if _, err := fh.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

// readFile reads a file or fails the test.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// mustJSON marshals v to a JSON string or fails the test.
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
