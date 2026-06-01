// ABOUTME: Three-channel parity harness — drives native dispatch in-process and
// ABOUTME: the project-vendored Python oracle by exec, with split stdout/stderr.
package dispatch

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// oracleFetchPrefix and nativeFetchPrefix are the one intentional divergence:
// the oracle emits a claude-team show-stage-def fetch line; the native emitter
// rewrites it to spacedock dispatch show-stage-def. Parity assertions rewrite
// the oracle bytes with this substitution before byte-comparing.
const (
	oracleFetchPrefix = "claude-team show-stage-def"
	nativeFetchPrefix = "spacedock dispatch show-stage-def"
)

// vendoredOracle returns the project-vendored claude-team path (not the plugin
// copy — it carries the Stage-4 slug-not-stem + split-root amendments).
func vendoredOracle(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "skills", "commission", "bin", "claude-team"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("vendored claude-team not found at %s: %v", p, err)
	}
	return p
}

// runResult is one process run's three channels.
type runResult struct {
	stdout string
	stderr string
	exit   int
}

// runOracle drives the vendored Python oracle with the given subcommand args,
// stdin, and pinned hermetic HOME, capturing stdout and stderr into separate
// buffers (CombinedOutput cannot byte-compare both). dir is the process working
// directory (the git-init'd fixture root, so find_git_root resolves there).
func runOracle(t *testing.T, dir, home, stdin string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command("python3", append([]string{vendoredOracle(t)}, args...)...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), "HOME="+home)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			t.Fatalf("oracle exec error: %v", err)
		}
	}
	return runResult{stdout.String(), stderr.String(), exit}
}

// runNative drives the native dispatch surface in-process with the given args
// and stdin, capturing stdout and stderr into separate buffers. HOME is pinned
// via the process env (set by the caller through t.Setenv) so the bare-mode
// team-evidence probe is hermetic, matching the oracle's pinned HOME.
func runNative(stdin string, args ...string) runResult {
	var stdout, stderr bytes.Buffer
	exit := Run(args, strings.NewReader(stdin), &stdout, &stderr)
	return runResult{stdout.String(), stderr.String(), exit}
}

// rewriteOracleFetch substitutes the oracle's claude-team show-stage-def fetch
// prefix with the native spacedock dispatch show-stage-def prefix, so the
// non-fetch bytes can be byte-compared after carving out the one rewritten line.
func rewriteOracleFetch(s string) string {
	return strings.ReplaceAll(s, oracleFetchPrefix, nativeFetchPrefix)
}

// stateCommitGuidanceLine matches the split-root state-commit guidance sentence
// (a single line ending in "after a short wait.\n"). The native emitter and the
// Python oracle diverge here intentionally — the same documented divergence as
// the fetch line: the oracle ships the gated literal-brace block (and emits
// nothing at all for non-worktree split-root stages), while the native emitter
// substitutes resolved absolute paths on both branches. The body-parity compare
// strips this block from BOTH sides so the unchanged bytes still byte-match; the
// diverged content is covered by build_statecommit_test.go instead.
var stateCommitGuidanceLine = regexp.MustCompile(`This workflow is split-root: [^\n]*? after a short wait\.\n`)

// stripStateCommitGuidance removes the diverged state-commit guidance line and
// collapses the blank-line artifact left where it was removed, so a body with
// the block and a body without it normalize to the same bytes.
func stripStateCommitGuidance(s string) string {
	s = stateCommitGuidanceLine.ReplaceAllString(s, "")
	return regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
}

// assertParity compares the native run against the oracle run on all three
// channels, rewriting the oracle's fetch prefix first. stdout and stderr must be
// byte-identical after the rewrite; exit codes must match.
func assertParity(t *testing.T, label string, native, oracle runResult) {
	t.Helper()
	wantStdout := rewriteOracleFetch(oracle.stdout)
	if native.stdout != wantStdout {
		t.Errorf("%s: stdout mismatch\n--- native ---\n%q\n--- oracle(rewritten) ---\n%q",
			label, native.stdout, wantStdout)
	}
	wantStderr := rewriteOracleFetch(oracle.stderr)
	if native.stderr != wantStderr {
		t.Errorf("%s: stderr mismatch\n--- native ---\n%q\n--- oracle(rewritten) ---\n%q",
			label, native.stderr, wantStderr)
	}
	if native.exit != oracle.exit {
		t.Errorf("%s: exit mismatch native=%d oracle=%d", label, native.exit, oracle.exit)
	}
}

// readDispatchBody reads the dispatch body file the run wrote (the path is
// deterministic from the derived name). Returns "" when the run wrote none.
func readDispatchBody(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read dispatch body %s: %v", path, err)
	}
	return string(b)
}

// gitInit initializes a git repo at dir so find_git_root resolves there.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "add", "-A"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
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
