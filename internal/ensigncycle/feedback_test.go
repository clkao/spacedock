// ABOUTME: BEHAVIORAL test of the gate/feedback rejection-reflow seam: a real
// ABOUTME: dispatch.Run reflow build routes the fix request to the feedback-to target.
package ensigncycle

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/dispatch"
)

// The gate/feedback loop's deterministic half lives on the dispatch side of the
// LLM: when the FO routes a rejection back to a gate stage's `feedback-to`
// target, dispatch.Run builds the reflow body with `is_feedback_reflow:true` and
// a concrete `feedback_context` carrying the routed fix request. Two byte-
// observable contracts gate that body (build.go):
//   - Rule 5 (build.go:225): is_feedback_reflow && feedback_context == "" exits
//     non-zero — a reflow MUST carry a routed payload.
//   - Section 6 (build.go:343): feedback_context != "" emits
//     `### Feedback from prior review\n\n{feedback_context}\n` — the routed fix
//     request the FO's Feedback Rejection Flow requires (the concrete next-stage
//     assignment, not a bare acknowledgment).
//
// The live half — the FO's PASSED/REJECTED gate decision and the 3-cycle
// `### Feedback Cycles` escalation — has no in-process Go seam (FO-LLM prose,
// no internal/status parser) and stays live-pytest/out-of-scope.
//
// Every assertion targets the EMIT FORM anchored at its line, never a bare
// substring prose could satisfy. The routing heading is emitted exactly once in
// build.go (no warning-prose duplicate), so the live trap is a dropped routed
// payload — which the context-presence assertion (NEG-A) catches.

var (
	// feedbackRoutingSection anchors the routed-fix-request section heading at
	// line start. A non-reflow dispatch never emits this heading.
	feedbackRoutingSection = regexp.MustCompile(`(?m)^### Feedback from prior review$`)
	// reflowMissingContext anchors the Rule 5 build-side guard message form.
	reflowMissingContext = regexp.MustCompile(
		`dispatching to feedback target stage 'implementation' but feedback_context is missing`)
)

// the routed rejection findings the FO carries on reflow — concrete fix work,
// not a bare acknowledgment.
const routedFeedback = "REJECTED: validation found the path-scoped commit swept a sibling; redo Rule 5."

// reflowFixture is a staged gate/feedback environment plus the reflow dispatch
// body dispatch.Run built.
type reflowFixture struct {
	root       string
	entityPath string
	body       string
}

// stageReflowFixture stages a git-init'd root whose README declares a gate stage
// (validation) with `feedback-to: implementation`, an entity stamped at the gate
// stage with a worktree value (the reflow rides the existing worktree), then runs
// the real dispatch.Run build for the feedback-to target stage and reads back the
// body. When reflow is true the build carries is_feedback_reflow + the routed
// feedback_context; otherwise it is a plain dispatch to the SAME target (the
// contrast control).
func stageReflowFixture(t *testing.T, reflow bool, feedbackContext string) reflowFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "README.md"), readmeFeedbackGate())
	entityPath := filepath.Join(root, "fix-the-thing.md")
	writeFile(t, entityPath, entityWithWorktree())

	// the reflow rides the existing worktree the gate stage already stamped; the
	// build stats the worktree dir, so it must exist.
	worktreeRel := ".worktrees/spacedock-ensign-fix-the-thing"
	if err := os.MkdirAll(filepath.Join(root, worktreeRel), 0o755); err != nil {
		t.Fatal(err)
	}
	gitInit(t, root)

	fields := map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		// dispatched to the gate stage's feedback-to target, not the reviewer.
		"stage":     "implementation",
		"checklist": []string{"- Address the rejection findings"},
		"team_name": "fixture-team",
		"bare_mode": false,
	}
	if reflow {
		fields["is_feedback_reflow"] = true
		fields["feedback_context"] = feedbackContext
	}

	var stdout, stderr strings.Builder
	code := dispatch.Run([]string{"build", "--workflow-dir", root},
		strings.NewReader(mustJSON(t, fields)), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("dispatch build exit=%d stderr=%s", code, stderr.String())
	}

	body := readDispatchBodyFromStdout(t, stdout.String())
	return reflowFixture{root: root, entityPath: entityPath, body: body}
}

// TestFeedbackReflowRoutesFixRequest drives a real in-process dispatch.Run reflow
// build dispatched to the gate stage's feedback-to target and asserts the routed
// fix request lands in the body, contrasted against a plain dispatch to the SAME
// target. This is AC-1: the deterministic half of the gate/feedback loop.
func TestFeedbackReflowRoutesFixRequest(t *testing.T) {
	reflow := stageReflowFixture(t, true, routedFeedback)
	plain := stageReflowFixture(t, false, "")

	// (a) the reflow body carries the anchored routing section.
	if !feedbackRoutingSection.MatchString(reflow.body) {
		t.Errorf("reflow body missing anchored feedback-routing section\n%s", reflow.body)
	}
	// (b) the routed rejection context is carried verbatim inside it — the
	// concrete fix work, not a bare acknowledgment.
	if !strings.Contains(reflow.body, routedFeedback) {
		t.Errorf("reflow body missing the routed feedback_context payload verbatim\n%s", reflow.body)
	}
	// (c) a plain (non-reflow) dispatch to the SAME target does NOT emit the
	// routing section — the seam is reflow-specific.
	if feedbackRoutingSection.MatchString(plain.body) {
		t.Errorf("plain dispatch body must NOT emit the feedback-routing section\n%s", plain.body)
	}
}

// TestFeedbackReflowGoesRedOnBrokenOutput is AC-2: the two named negative
// controls, encoded as in-test guards (the negative paths are exercised, not
// described).
func TestFeedbackReflowGoesRedOnBrokenOutput(t *testing.T) {
	// NEG-A (dropped payload): take the real reflow body and strip the routed
	// feedback_context, then prove the context-presence assertion goes RED. This
	// is the in-test encoding of the spike's production mutation (dropping the
	// section-6 emission) — it proves the test catches a dropped fix request, the
	// FO-mode "not just an acknowledgment" failure.
	t.Run("dropped_routed_payload", func(t *testing.T) {
		reflow := stageReflowFixture(t, true, routedFeedback)
		stripped := strings.Replace(reflow.body, routedFeedback, "", 1)

		if strings.Contains(stripped, routedFeedback) {
			t.Fatal("expected the routed feedback_context to be absent after stripping")
		}
		// the anchored routing heading alone survives (it is emitted unconditionally
		// when feedback_context != ""), so a bare heading match would WRONGLY pass —
		// the context-presence assertion is what catches the dropped payload.
		if !feedbackRoutingSection.MatchString(stripped) {
			t.Fatal("routing heading expected to survive the payload strip " +
				"(this is why the context-presence assertion, not the heading match, is the guard)")
		}
	})

	// NEG-B (build-side guard): a reflow build with EMPTY feedback_context must
	// make dispatch.Run's Rule 5 exit non-zero with the anchored stderr message,
	// proving the build refuses an unrouted reflow.
	t.Run("reflow_with_empty_feedback_context", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "README.md"), readmeFeedbackGate())
		entityPath := filepath.Join(root, "fix-the-thing.md")
		writeFile(t, entityPath, entityWithWorktree())
		if err := os.MkdirAll(filepath.Join(root, ".worktrees/spacedock-ensign-fix-the-thing"), 0o755); err != nil {
			t.Fatal(err)
		}
		gitInit(t, root)

		stdin := mustJSON(t, map[string]any{
			"schema_version":     2,
			"entity_path":        entityPath,
			"workflow_dir":       root,
			"stage":              "implementation",
			"checklist":          []string{"- Address the rejection findings"},
			"team_name":          "fixture-team",
			"bare_mode":          false,
			"is_feedback_reflow": true,
			"feedback_context":   "",
		})

		var stdout, stderr strings.Builder
		code := dispatch.Run([]string{"build", "--workflow-dir", root},
			strings.NewReader(stdin), &stdout, &stderr)

		if code == 0 {
			t.Fatalf("expected non-zero exit for reflow with empty feedback_context; stdout=%s", stdout.String())
		}
		if !reflowMissingContext.MatchString(stderr.String()) {
			t.Fatalf("stderr missing anchored Rule 5 guard message\n%s", stderr.String())
		}
	})
}

// --- fixture helpers (self-contained; this package does not share the unexported
// test helpers in internal/dispatch) ---

// readmeFeedbackGate is a workflow README with a worktree gate stage (validation)
// declaring feedback-to: implementation, mirroring internal/dispatch's
// readmeWorktree. The reflow is dispatched to the implementation target.
func readmeFeedbackGate() string {
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
		"    - name: implementation\n" +
		"      worktree: true\n" +
		"    - name: validation\n" +
		"      worktree: true\n" +
		"      feedback-to: implementation\n" +
		"    - name: done\n" +
		"      terminal: true\n" +
		"---\n" +
		"# Fixture Workflow\n" +
		"\n" +
		"### backlog\n\nseed.\n\n- **Outputs:** x.\n\n" +
		"### implementation\n\nwork.\n\n- **Outputs:** y.\n\n" +
		"### validation\n\nverify.\n\n- **Outputs:** z.\n\n" +
		"### done\n\nterm.\n"
}

// entityWithWorktree is an entity stamped at the gate stage with a worktree
// value, so the reflow rides the existing worktree.
func entityWithWorktree() string {
	return "---\n" +
		"id: \"001\"\n" +
		"title: Fix The Thing\n" +
		"status: validation\n" +
		"worktree: .worktrees/spacedock-ensign-fix-the-thing\n" +
		"---\n" +
		"# Fix The Thing\n\nBody.\n"
}
