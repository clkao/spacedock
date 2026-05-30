// ABOUTME: Table-driven doctor tests over manifest fixtures — exit code +
// ABOUTME: pinned verdict/remedy substring for all five compatibility verdicts.
package contract

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestDoctorVerdicts drives `spacedock doctor --plugin-manifest <fixture>`
// against each verdict fixture and observes exit code + the pinned
// verdict/remedy substring. The five fixtures are the oracle.
func TestDoctorVerdicts(t *testing.T) {
	cases := []struct {
		name       string
		manifest   string // fixture filename under testdata/, "" for absent
		host       string
		wantExit   int
		wantStdout string // substring on stdout (compatible / no-plugin report)
		wantStderr string // substring on stderr (mismatch remedy)
	}{
		{
			name:       "compatible",
			manifest:   "compatible.json",
			host:       "claude",
			wantExit:   0,
			wantStdout: "OK",
		},
		{
			name:       "too-old-binary",
			manifest:   "too-old-binary.json",
			host:       "claude",
			wantExit:   1,
			wantStderr: "too-old-binary: your spacedock binary",
		},
		{
			name:       "too-old-plugin",
			manifest:   "too-old-plugin.json",
			host:       "claude",
			wantExit:   1,
			wantStderr: "too-old-plugin: your installed plugin",
		},
		{
			name:       "malformed-range",
			manifest:   "malformed-range.json",
			host:       "claude",
			wantExit:   1,
			wantStderr: "malformed contract range",
		},
		{
			name:       "no-plugin-found",
			manifest:   "does-not-exist.json",
			host:       "claude",
			wantExit:   0, // a report, non-fatal-by-default
			wantStdout: "no installed Spacedock plugin found",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			manifestPath := filepath.Join("testdata", c.manifest)
			code := RunDoctor(manifestPath, c.host, "", &stdout, &stderr)
			if code != c.wantExit {
				t.Fatalf("exit = %d, want %d (stdout=%q stderr=%q)", code, c.wantExit, stdout.String(), stderr.String())
			}
			if c.wantStdout != "" && !strings.Contains(stdout.String(), c.wantStdout) {
				t.Fatalf("stdout = %q, want substring %q", stdout.String(), c.wantStdout)
			}
			if c.wantStderr != "" && !strings.Contains(stderr.String(), c.wantStderr) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), c.wantStderr)
			}
		})
	}
}

// TestDoctorMalformedNamesManifest locks that the malformed-range message names
// the offending manifest path (the packaging-bug verdict points at the file).
func TestDoctorMalformedNamesManifest(t *testing.T) {
	var stdout, stderr bytes.Buffer
	manifestPath := filepath.Join("testdata", "malformed-range.json")
	code := RunDoctor(manifestPath, "claude", "", &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), manifestPath) {
		t.Fatalf("malformed-range message must name the manifest path %q: %q", manifestPath, stderr.String())
	}
	// A malformed range carries NO upgrade remedy (neither side is too old).
	if strings.Contains(stderr.String(), "too-old") {
		t.Fatalf("malformed-range must not offer a too-old remedy: %q", stderr.String())
	}
}

// TestVendoredFixtureBracketsContractVersion is the AC-5a Go half: the vendored
// manifest fixture (a checked-in copy of the authoritative .codex-plugin
// manifest) declares a well-formed requires-contract that brackets
// CONTRACT_VERSION. Closes fixture-vs-binary drift with a single go test.
func TestVendoredFixtureBracketsContractVersion(t *testing.T) {
	raw, err := readRequiresContract(filepath.Join("testdata", "plugin.json"))
	if err != nil {
		t.Fatalf("read vendored fixture requires-contract: %v", err)
	}
	lo, hi, err := ParseRange(raw)
	if err != nil {
		t.Fatalf("vendored fixture requires-contract %q does not parse: %v", raw, err)
	}
	if !(lo <= CONTRACT_VERSION && CONTRACT_VERSION < hi) {
		t.Fatalf("vendored fixture range %s does not bracket CONTRACT_VERSION=%d", raw, CONTRACT_VERSION)
	}
}
