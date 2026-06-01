// ABOUTME: Parity-harness shared helpers — pinned env, oracle/launcher runners,
// ABOUTME: and the normalization (timestamps, root prefix, realpath) for compares.
package status

import (
	"bytes"
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// update regenerates checked-in golden files from the oracle instead of
// comparing against them. Run: go test ./internal/status -run TestGoldenRead -update
var update = flag.Bool("update", false, "regenerate golden files from the oracle")

// oracleEnvVar names an env var that overrides the path to the live status
// oracle script. When unset, the harness falls back to the in-tree vendored
// oracle, which is always present in a checkout.
const oracleEnvVar = "SPACEDOCK_ORACLE"

// pinnedEnv returns the locale/id/timestamp-pinned environment both the oracle
// and the launcher must run under so env-dependent output is reproducible. The
// values mirror the test plan: PYTHONUTF8/LANG for locale, USER/actor/seed and
// SPACEDOCK_TEST_SD_B32_TIMESTAMP for sd-b32, HOME for the team probe, PATH for
// locating python3/git/gh.
func pinnedEnv(t *testing.T) []string {
	t.Helper()
	return []string{
		"PYTHONUTF8=1",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"USER=pinned-actor",
		"SPACEDOCK_TEST_SD_B32_TIMESTAMP=2026-01-01T00:00:00.000000Z",
		"HOME=" + t.TempDir(), // empty team dir -> deterministic TEAM_STATE
		"PATH=" + os.Getenv("PATH"),
	}
}

// oraclePath returns the oracle script the parity tests run against. A
// SPACEDOCK_ORACLE override, when set, must point at an existing file — a
// misconfigured override is a hard failure, not a skip. With no override the
// in-tree vendored oracle is used; a missing vendored copy is a hard failure
// too. The oracle is therefore always resolvable in a checkout, so the parity
// assertions hard-fail on a real divergence instead of green-by-skip on CI or a
// fresh clone (the test-integrity contract this enforces).
func oraclePath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv(oracleEnvVar); p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s=%s does not exist: %v", oracleEnvVar, p, err)
		}
		return p
	}
	p, err := filepath.Abs(filepath.Join("vendor", "status"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("vendored oracle not found at %s: %v", p, err)
	}
	return p
}

// runLauncher runs the vendored-exec runner (the launcher path) with the given
// args/dir/env and returns stdout, stderr, exit code.
func runLauncher(t *testing.T, dir string, env []string, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	runner := &VendorRunner{}
	code, err := runner.Run(context.Background(), Request{
		Args:   args,
		Dir:    dir,
		Env:    env,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("launcher runner error: %v (stderr=%q)", err, stderr.String())
	}
	return stdout.String(), stderr.String(), code
}

// runOracle runs the oracle script directly under python3 with the given
// args/dir/env and returns stdout, stderr, exit code. A missing oracle is a hard
// failure (see oraclePath), never a skip.
func runOracle(t *testing.T, dir string, env []string, args ...string) (string, string, int) {
	t.Helper()
	oracle := oraclePath(t)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("python3", append([]string{oracle}, args...)...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("oracle run error: %v (stderr=%q)", err, stderr.String())
		}
	}
	return stdout.String(), stderr.String(), code
}

func asExitError(err error, target **exec.ExitError) bool {
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	return false
}

// tsRe matches ISO-8601 UTC timestamps in BOTH the second-precision mutation
// shape and the microsecond sd-b32 shape. The trailing (\.\d+)? is required or
// the microsecond timestamp slips through un-normalized.
var tsRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z`)

// stateBackendLineRe matches the native-only STATE_BACKEND boot banner. The text
// --boot adds it to surface the state backend for a human reading the boot, but
// the Python oracle has no such line — the same kind of intentional native/oracle
// divergence as the dispatch fetch-line and state-commit guidance. The boot-text
// parity normalizers strip it from BOTH sides (the oracle never emits it) so the
// shared sections still byte-match; the JSON form is covered by json_boot_test.go.
var stateBackendLineRe = regexp.MustCompile(`STATE_BACKEND: [^\n]*\n`)

// stripStateBackend removes the native-only STATE_BACKEND boot banner so a body
// with it and the oracle body without it normalize to the same bytes.
func stripStateBackend(s string) string {
	return stateBackendLineRe.ReplaceAllString(s, "")
}

// sdB32Re matches a 24-char SD-B32 id token (the --next-id / NEXT_ID material),
// used to normalize the non-deterministic id away for structural comparison.
var sdB32Re = regexp.MustCompile(`\b[0-9a-hjkmnp-tv-z]{24}\b`)

// normalize applies the test-plan normalization to output before comparison:
// the timestamp placeholder, and root-prefix placeholders for the workflow root
// (both as-spelled and realpath'd, since --resolve workflow= is realpath'd on
// macOS /var->/private/var while path=/archived: are not).
func normalize(s, root string) string {
	s = tsRe.ReplaceAllString(s, "<TS>")
	if root != "" {
		real := realpath(root)
		// Replace the realpath'd spelling first (longer/more specific on macOS),
		// then the as-spelled root, so both map to the same placeholder.
		if real != root {
			s = strings.ReplaceAll(s, real, "<ROOT>")
		}
		s = strings.ReplaceAll(s, root, "<ROOT>")
	}
	return s
}

// realpath resolves symlinks the way the oracle's os.path.realpath does, so the
// expected workflow= prefix accounts for the macOS /var->/private/var rewrite.
func realpath(p string) string {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return resolved
}

// writeGolden / readGolden manage checked-in golden files under testdata/golden.
func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name)
}

func writeGolden(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(goldenPath(name), []byte(content), 0o644); err != nil {
		t.Fatalf("write golden %s: %v", name, err)
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(goldenPath(name))
	if err != nil {
		t.Fatalf("read golden %s (regenerate with -update): %v", name, err)
	}
	return string(b)
}
