// ABOUTME: AC-4 decoupling tests — the status command path depends only on the
// ABOUTME: status.Runner interface and forwards Args/Dir/Env/Stdin verbatim.
package cli

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/status"
)

// fakeRunner records the request it received and returns a canned result. It
// lets the CLI status path be exercised with no Python, no exec, no real script.
type fakeRunner struct {
	gotReq   status.Request
	gotStdin string
	exitCode int
	err      error
	stdout   string
	stderr   string
}

func (f *fakeRunner) Run(ctx context.Context, req status.Request) (int, error) {
	f.gotReq = req
	if req.Stdin != nil {
		b, _ := io.ReadAll(req.Stdin)
		f.gotStdin = string(b)
	}
	if f.stdout != "" {
		io.WriteString(req.Stdout, f.stdout)
	}
	if f.stderr != "" {
		io.WriteString(req.Stderr, f.stderr)
	}
	return f.exitCode, f.err
}

func TestStatusForwardsRequestVerbatim(t *testing.T) {
	fake := &fakeRunner{exitCode: 0, stdout: "TABLE\n"}
	var stdout, stderr bytes.Buffer

	env := []string{"USER=pinned", "SPACEDOCK_TEST_SD_B32_TIMESTAMP=2026-01-01T00:00:00.000000Z"}
	dir := "/pinned/dir"
	stdin := strings.NewReader("BODY-FROM-STDIN")
	args := []string{"status", "--workflow-dir", "/wf", "--next"}

	code := run(context.Background(), args, env, dir, stdin, &stdout, &stderr, fake)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	wantArgs := []string{"--workflow-dir", "/wf", "--next"}
	if !reflect.DeepEqual(fake.gotReq.Args, wantArgs) {
		t.Fatalf("Args = %v, want %v (the leading \"status\" token must be stripped, rest verbatim)", fake.gotReq.Args, wantArgs)
	}
	if fake.gotReq.Dir != dir {
		t.Fatalf("Dir = %q, want %q", fake.gotReq.Dir, dir)
	}
	if !reflect.DeepEqual(fake.gotReq.Env, env) {
		t.Fatalf("Env = %v, want %v", fake.gotReq.Env, env)
	}
	if fake.gotStdin != "BODY-FROM-STDIN" {
		t.Fatalf("Stdin = %q, want %q", fake.gotStdin, "BODY-FROM-STDIN")
	}
	if stdout.String() != "TABLE\n" {
		t.Fatalf("stdout = %q, want runner stdout passthrough", stdout.String())
	}
}

// AC-1: the CLI returns the runner's exit code unmodified across the {0,1}
// domain and never injects an exit-2 path of its own for status.
func TestStatusReturnsRunnerExitCodeUnmodified(t *testing.T) {
	for _, code := range []int{0, 1} {
		fake := &fakeRunner{exitCode: code}
		var stdout, stderr bytes.Buffer
		got := run(context.Background(), []string{"status", "--workflow-dir", "/wf"}, nil, "", nil, &stdout, &stderr, fake)
		if got != code {
			t.Fatalf("exit code = %d, want %d (CLI must not translate)", got, code)
		}
	}
}

// AC-1: the unknown-top-level-flag case. The CLI forwards the flag rather than
// rejecting it; whatever the runner produces (here: default table, exit 0) is
// what the CLI returns. The CLI does not strip or validate flags.
func TestStatusUnknownTopLevelFlagForwarded(t *testing.T) {
	fake := &fakeRunner{exitCode: 0, stdout: "DEFAULT-TABLE\n"}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"status", "--workflow-dir", "/wf", "--bogus-top-level"}, nil, "", nil, &stdout, &stderr, fake)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	wantArgs := []string{"--workflow-dir", "/wf", "--bogus-top-level"}
	if !reflect.DeepEqual(fake.gotReq.Args, wantArgs) {
		t.Fatalf("Args = %v, want %v (unknown flag forwarded verbatim)", fake.gotReq.Args, wantArgs)
	}
}

// When the runner cannot run at all (interpreter missing), the CLI fails loudly
// with exit 1 and a diagnostic on stderr rather than masquerading as success.
func TestStatusRunnerErrorIsLoud(t *testing.T) {
	fake := &fakeRunner{exitCode: -1, err: errFakeNoPython}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"status", "--workflow-dir", "/wf"}, nil, "", nil, &stdout, &stderr, fake)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "python3 missing") {
		t.Fatalf("stderr = %q, want the runner diagnostic surfaced", stderr.String())
	}
}

var errFakeNoPython = errFake("spacedock status: python3 missing")

type errFake string

func (e errFake) Error() string { return string(e) }
