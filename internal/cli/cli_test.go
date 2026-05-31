package cli

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/contract"
)

func TestHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run returned %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help output missing Usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
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
