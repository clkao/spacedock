package cli

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/contract"
)

// TestTopLevelHelpGroupedJargonFree pins AC-1: `--help` renders the tagline, the
// three group headers in order, the six grouped command names with terse
// one-liners, and a single footer pointing at per-command help + --version. The
// banned internal jargon (`front door`, `contract-gated`, `META`) is absent, and
// neither `--version` nor `--help` appears as its own command row (they live in
// the footer). The render goes to stdout with exit 0 and empty stderr.
func TestTopLevelHelpGroupedJargonFree(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	out := stdout.String()

	for _, want := range []string{
		"spacedock — agentic workflow launcher",
		"Launch", "Setup", "Workflow",
		"claude", "codex", "install", "doctor", "status", "dispatch",
		`Run "spacedock <command> --help" for details.`,
		"--version prints the version.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("top-level help missing %q:\n%s", want, out)
		}
	}

	for _, banned := range []string{"front door", "contract-gated", "META"} {
		if strings.Contains(out, banned) {
			t.Errorf("top-level help carried banned jargon %q:\n%s", banned, out)
		}
	}

	// The group headers appear in the captain-approved order.
	iLaunch := strings.Index(out, "Launch")
	iSetup := strings.Index(out, "Setup")
	iWorkflow := strings.Index(out, "Workflow")
	if !(iLaunch < iSetup && iSetup < iWorkflow) {
		t.Errorf("group headers out of order: Launch=%d Setup=%d Workflow=%d", iLaunch, iSetup, iWorkflow)
	}
}

// TestTopLevelHelpFormsAreIdentical pins AC-1's invariant that bare `spacedock`,
// `-h`, and the `help` subcommand render byte-identical output to `--help`.
func TestTopLevelHelpFormsAreIdentical(t *testing.T) {
	ref := helpStdout(t, "--help")
	for _, form := range [][]string{nil, {"-h"}, {"help"}} {
		got := helpStdout(t, form...)
		if got != ref {
			t.Errorf("help form %v output differs from --help:\n--- form ---\n%s\n--- --help ---\n%s", form, got, ref)
		}
	}
}

func helpStdout(t *testing.T, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(%v) = %d, want 0 (stderr=%q)", args, code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run(%v) stderr = %q, want empty", args, stderr.String())
	}
	return stdout.String()
}

func TestVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0", code)
	}
	want := "spacedock " + Version + " (contract " + strconv.Itoa(contract.CONTRACT_VERSION) + ")"
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// TestVersionContractToken locks the AC-1 contract token: --version emits a
// `contract <N>` token where <N> equals CONTRACT_VERSION and parses as an integer.
func TestVersionContractToken(t *testing.T) {
	var stdout bytes.Buffer
	Run([]string{"--version"}, &stdout, &bytes.Buffer{})

	got := stdout.String()
	marker := "(contract "
	idx := strings.Index(got, marker)
	if idx < 0 {
		t.Fatalf("version output %q missing %q token", got, marker)
	}
	rest := got[idx+len(marker):]
	end := strings.IndexByte(rest, ')')
	if end < 0 {
		t.Fatalf("version contract token not closed by ')': %q", got)
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest[:end]))
	if err != nil {
		t.Fatalf("contract token %q does not parse as integer: %v", rest[:end], err)
	}
	if n != contract.CONTRACT_VERSION {
		t.Fatalf("contract token = %d, want CONTRACT_VERSION = %d", n, contract.CONTRACT_VERSION)
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"bogus"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("Run returned %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command: bogus") {
		t.Fatalf("stderr missing unknown-command message: %q", stderr.String())
	}
}
