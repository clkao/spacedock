//go:build live

// ABOUTME: Live BEHAVIORAL test of the dispatch->ensign->stage cycle driven by a
// ABOUTME: REAL model through the spacedock claude front door (gated, -tags live only).
package ensigncycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This is the live analog of TestEnsignCycleMechanicalOutputs (cycle_test.go).
// Where the skeleton stubs the LLM with a scripted Go ensign that works ONE stage
// in place, this test shells the v1 binary's REAL front door so an actual model
// drives the whole dispatch->ensign->stage protocol all the way to the terminal
// `done` stage. A real full-cycle FO makes MULTIPLE commits and ARCHIVES the
// terminal entity to `_archive/`, so the assertions target the REAL
// completed-and-archived end-state: the entity (located in place OR under
// `_archive/`) carries the anchored stage-report shape
// (liveStageReportHeading, doneMarker, `### Summary`, NOT checkboxBullet) AND the
// FO's terminal frontmatter (`status: done`, `verdict: passed`), and SOME commit
// in the history is path-scoped to the entity. The heading regex is stage-agnostic
// (the real cycle finishes at the TERMINAL stage, so the ensign writes
// `## Stage Report: done`); the remaining regexes are reused verbatim from the
// skeleton and the producer is the real runtime.
//
// The `//go:build live` tag keeps this out of the default `go test ./...` suite
// (the secret-free offline job). It compiles and runs ONLY under
// `go test -tags live`, the gated job's invocation that spends the live
// credential behind the CI-E2E approval gate.

// liveTimeout caps the single live dispatch->cycle run. A sonnet/opus run driving
// the flat entity to the terminal stage is well inside this (the FO live runs
// landed at ~350s); the cap turns a hung model run into a red FAIL with output
// rather than a silent CI hang.
const liveTimeout = 12 * time.Minute

// TestLiveEnsignCycle stages the skeleton's flat-entity backlog fixture, shells
// the real `spacedock claude` front door headless to drive the entity to done
// through a live model, then reads back the entity + git log and asserts the
// anchored mechanical contract. It is the smallest meaningful live mechanism
// proof: real binary front door + real plugin load + real model + real
// dispatch->ensign->stage cycle + a real path-scoped state commit.
//
// Auth + HOME isolation are resolved by isolatedClaudeEnv: an operator machine
// authenticates via the OAuth benchmark-token (~/.claude/benchmark-token), the
// CI runner via ANTHROPIC_API_KEY, and a machine with neither SKIPS (never
// fatals). The chosen credential runs against a fresh empty HOME so parallel
// `spacedock claude` invocations never collide in ~/.claude.
func TestLiveEnsignCycle(t *testing.T) {
	binary := spacedockBinary(t)
	repoRoot := repoRoot(t)
	model := envOr("SPACEDOCK_LIVE_MODEL", "sonnet")

	// Resolve the isolated child env (clean HOME + the authoritative credential)
	// or skip when no auth mechanism is available. The empty home argument means
	// "read the live $HOME"; the offline unit test drives isolatedClaudeEnv
	// directly with a fake home so it never touches the real ~/.claude.
	childEnv := isolatedClaudeEnv(t, os.Getenv("HOME"))

	// Stage the SAME flat-entity backlog fixture the skeleton builds: a
	// git-init'd root with a non-worktree workflow README and a flat entity in
	// the initial (backlog) stage. The real FO drives it to the TERMINAL stage,
	// so the ensign that finishes the cycle writes `## Stage Report: done`, which
	// the stage-agnostic liveStageReportHeading regex matches.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeNonWorktree())
	entityPath := filepath.Join(root, "make-it-work.md")
	writeFile(t, entityPath, entityFixture())
	gitInit(t, root)

	task := "Discover the workflow in this directory and drive the single backlog " +
		"entity make-it-work.md all the way to the done stage by dispatching an " +
		"ensign for it. Do not stop until the entity reaches the terminal stage."

	ctx, cancel := context.WithTimeout(context.Background(), liveTimeout)
	defer cancel()

	// The real front door: `spacedock claude --plugin-dir <repo> --skip-contract-check
	// -p <bootstrap> ... -- <task>`. --plugin-dir loads the local v1 plugin checkout
	// (and relaxes the contract gate); --skip-contract-check is belt-and-braces.
	// stream-json + bypassPermissions + the model pin mirror the headless launch
	// the Python net uses. The task is fenced after `--` so it rides as the
	// launch-prompt override. CLAUDECODE is dropped by isolatedClaudeEnv so the
	// binary takes the real front-door path rather than a nested-session shortcut.
	cmd := exec.CommandContext(ctx, binary, "claude",
		"--plugin-dir", repoRoot,
		"--skip-contract-check",
		"-p", "Drive the workflow.",
		"--permission-mode", "bypassPermissions",
		"--output-format", "stream-json",
		"--verbose",
		"--model", model,
		"--", task,
	)
	cmd.Dir = root
	cmd.Env = childEnv

	out, err := cmd.CombinedOutput()
	t.Logf("spacedock claude exit: %v\n--- transcript (tail) ---\n%s", err, tail(string(out), 8000))
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("live cycle timed out after %s", liveTimeout)
	}
	if err != nil {
		t.Fatalf("spacedock claude failed: %v", err)
	}

	// Locate the entity at the REAL completed-cycle end-state. A full FO-to-done
	// cycle ARCHIVES the terminal entity: the flat `make-it-work.md` moves to
	// `_archive/make-it-work.md`. locateEntity searches the original path AND both
	// archive spellings; a missing entity everywhere is a hard FAIL (the cycle
	// neither completed nor left the entity in place).
	entity, where, found := locateEntity(root, "make-it-work")
	if !found {
		t.Fatalf("entity make-it-work not found in place or under _archive/ after the cycle")
	}
	t.Logf("located entity at %s", where)

	// (a) the appended stage-report section has the protocol shape: heading, a
	// DONE accounting marker, a Summary, and NO checkbox-bullet form. An
	// INCOMPLETE cycle (the haiku run) appends no stage report, so these go red.
	if !liveStageReportHeading.MatchString(entity) {
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

	// (b) the FO finalized the cycle: the entity carries the terminal frontmatter
	// `status: done` and a SET (non-empty) `verdict:`. The exact verdict word is FO
	// judgment that varies by model (sonnet wrote `verdict: done`, opus wrote
	// `verdict: passed`) — both completed the full cycle — so the live test gates
	// on the verdict being decided, not on a specific word. An incomplete cycle
	// never reaches the terminal stage and leaves the verdict empty, so these go red.
	if !frontmatterField.MatchString(entity) {
		t.Errorf("entity missing terminal `status: done`\n%s", entity)
	}
	if !verdictSet.MatchString(entity) {
		t.Errorf("entity missing a finalized (non-empty) `verdict:`\n%s", entity)
	}

	// (c) SOME commit in the history is path-scoped to the entity (names only the
	// entity), the concurrency-safe state-commit invariant at the cycle level.
	// HEAD itself is the FO's archive/finalize commit on a full cycle, so this
	// scans the whole log rather than pinning HEAD (the strict single-file HEAD
	// invariant is pinned deterministically by the skeleton's
	// TestEnsignCycleMechanicalOutputs). The haiku incomplete cycle's only
	// entity-touching commit swept a sibling, so this goes red on it.
	if !someCommitNamesOnly(t, root, "make-it-work") {
		t.Errorf("no path-scoped commit named only the entity in the cycle history")
	}
}

// spacedockBinary resolves the built v1 binary the test shells. SPACEDOCK_BIN
// (set by the CI job after `go build -o ./spacedock`) takes precedence; locally
// it falls back to a `spacedock` on PATH. The test fails loudly when neither
// resolves rather than silently shelling a stale or absent binary.
func spacedockBinary(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("SPACEDOCK_BIN"); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			t.Fatalf("SPACEDOCK_BIN=%q is not resolvable: %v", p, err)
		}
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("SPACEDOCK_BIN=%q does not exist: %v", abs, err)
		}
		return abs
	}
	p, err := exec.LookPath("spacedock")
	if err != nil {
		t.Fatal("no spacedock binary: set SPACEDOCK_BIN to the built binary or put spacedock on PATH")
	}
	return p
}

// repoRoot resolves the plugin-checkout root passed to --plugin-dir. The
// ensigncycle package lives at internal/ensigncycle, so the repo root is two
// directories up from the test's working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("SPACEDOCK_REPO_ROOT"); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			t.Fatalf("SPACEDOCK_REPO_ROOT=%q is not resolvable: %v", p, err)
		}
		return abs
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
