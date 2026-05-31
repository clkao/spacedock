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
// Where the skeleton stubs the LLM with a scripted Go ensign, this test shells
// the v1 binary's REAL front door so an actual model drives the
// dispatch->ensign->stage protocol end to end, then asserts the SAME anchored
// on-disk mechanical outputs the skeleton pins (stageReportHeading, doneMarker,
// `### Summary`, NOT checkboxBullet, commitNameOnly == [the entity]). Only the
// producer changes (real runtime vs scripted ensign); the assertion vocabulary
// is reused verbatim from the skeleton's package-level regexes/helpers.
//
// The `//go:build live` tag keeps this out of the default `go test ./...` suite
// (the secret-free offline job). It compiles and runs ONLY under
// `go test -tags live`, the gated job's invocation that spends ANTHROPIC_API_KEY
// behind the CI-E2E approval gate.

// liveTimeout caps the single live dispatch->cycle run. A haiku/low run of one
// flat-entity backlog stage is well inside this; the cap turns a hung model run
// into a red FAIL with output rather than a silent CI hang.
const liveTimeout = 12 * time.Minute

// TestLiveEnsignCycle stages the skeleton's flat-entity backlog fixture, shells
// the real `spacedock claude` front door headless to drive the entity to done
// through a live model, then reads back the entity + git log and asserts the
// anchored mechanical contract. It is the smallest meaningful live mechanism
// proof: real binary front door + real plugin load + real model + real
// dispatch->ensign->stage cycle + a real path-scoped state commit.
func TestLiveEnsignCycle(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Fatal("ANTHROPIC_API_KEY must be set for the live cycle test")
	}
	binary := spacedockBinary(t)
	repoRoot := repoRoot(t)
	model := envOr("SPACEDOCK_LIVE_MODEL", "haiku")

	// Stage the SAME flat-entity backlog fixture the skeleton builds: a
	// git-init'd root with a non-worktree workflow README and a flat entity in
	// the initial (backlog) stage. The produced stage-report heading is
	// `## Stage Report: backlog`, matching the package-level stageReportHeading
	// regex exactly.
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
	// launch-prompt override. CLAUDECODE is unset so the binary takes the real
	// front-door path rather than a nested-session shortcut.
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
	cmd.Env = liveEnv()

	out, err := cmd.CombinedOutput()
	t.Logf("spacedock claude exit: %v\n--- transcript (tail) ---\n%s", err, tail(string(out), 8000))
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("live cycle timed out after %s", liveTimeout)
	}
	if err != nil {
		t.Fatalf("spacedock claude failed: %v", err)
	}

	// Read back the fixture entity + git log and run the SAME anchored assertions
	// the skeleton uses. The real cycle must have produced the protocol-shaped
	// stage report and a path-scoped commit naming only the entity.
	entity := readFile(t, entityPath)

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

	// (b) a commit landed and named ONLY the entity (path-scoped, no sibling
	// sweep) — the concurrency-safe state-commit invariant at the cycle level.
	named := commitNameOnly(t, root)
	if len(named) != 1 || filepath.Base(named[0]) != "make-it-work.md" {
		t.Errorf("HEAD commit must name only the entity; named=%v", named)
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

// liveEnv returns the child environment for the front-door launch: the parent
// env minus CLAUDECODE (so the binary takes the real front-door path, not a
// nested-session shortcut). ANTHROPIC_API_KEY is inherited from the parent (the
// CI job env / the operator shell).
func liveEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "CLAUDECODE=") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// envOr returns the environment value for key, or def when unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// tail returns the last n bytes of s, prefixed with an elision marker when s was
// truncated, so the transcript log stays bounded.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…(truncated)…\n" + s[len(s)-n:]
}
