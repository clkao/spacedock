// ABOUTME: AC-2 oracle (3) behavior fixture — a real spacedock stub whose
// ABOUTME: --version prints an out-of-range contract drives the startup gate.
package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestStartupGateAbortsBeforeDiscover builds a real `spacedock` stub whose
// `--version` prints a chosen `contract N`, runs the stub for its version
// output, drives the startup gate against an embedded range, and observes: an
// out-of-range contract aborts with the pinned message and the stub's
// `status --discover` / `--boot` subcommands are NEVER invoked; an in-range
// contract proceeds and the discover call fires exactly once.
func TestStartupGateAbortsBeforeDiscover(t *testing.T) {
	stub, marker := buildVersionStub(t)

	cases := []struct {
		name          string
		stubContract  string // value the stub prints in its contract token
		embeddedRange string
		wantProceed   bool
		wantPinned    string // abort-message substring (empty when proceeding)
	}{
		{"too-old-binary-aborts", "0", ">=1,<2", false, "too-old-binary"},
		{"too-old-plugin-aborts", "5", ">=1,<2", false, "too-old-plugin"},
		{"compatible-proceeds", "1", ">=1,<2", true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			discoverCalls := 0
			stubEnv := append(os.Environ(),
				"SD_STUB_CONTRACT="+c.stubContract,
				"SD_STUB_MARKER="+marker,
			)

			// runVersion executes the real stub's --version.
			runVersion := func() (string, error) {
				cmd := exec.Command(stub, "--version")
				cmd.Env = stubEnv
				out, err := cmd.Output()
				return string(out), err
			}
			// runDiscover executes the stub's status --discover and counts the call.
			runDiscover := func() error {
				discoverCalls++
				cmd := exec.Command(stub, "status", "--discover")
				cmd.Env = stubEnv
				return cmd.Run()
			}

			proceed, msg := gateAndMaybeDiscover(runVersion, c.embeddedRange, "claude", "", runDiscover)

			if proceed != c.wantProceed {
				t.Fatalf("proceed = %v, want %v (msg=%q)", proceed, c.wantProceed, msg)
			}
			if c.wantProceed {
				if discoverCalls != 1 {
					t.Fatalf("compatible gate: discover called %d times, want 1", discoverCalls)
				}
			} else {
				if discoverCalls != 0 {
					t.Fatalf("aborting gate invoked discover %d times, want 0", discoverCalls)
				}
				if !strings.Contains(msg, c.wantPinned) {
					t.Fatalf("abort message %q missing pinned remedy %q", msg, c.wantPinned)
				}
			}

			// Sanity: the stub records which subcommands it actually ran. On an
			// aborting gate the marker file must show only the version call.
			if !c.wantProceed {
				record := readMarker(t, marker)
				if strings.Contains(record, "discover") {
					t.Fatalf("stub was invoked with discover on an aborting gate; marker:\n%s", record)
				}
			}
			// Reset the marker between cases.
			os.Remove(marker)
		})
	}
}

// gateAndMaybeDiscover realizes the FO Startup step-0 gate as a callable
// mechanism: run the version probe, parse the `contract N` token, compare
// against the embedded range, and only call discover when compatible. This is
// the Go realization of the prose the FO follows — driven here by a real stub
// process, not a mock.
func gateAndMaybeDiscover(runVersion func() (string, error), embeddedRange, host, branch string, runDiscover func() error) (proceed bool, message string) {
	out, err := runVersion()
	if err != nil {
		return false, "spacedock --version unavailable: " + err.Error()
	}
	c, ok := parseContractToken(out)
	if !ok {
		return false, "could not parse contract token from `spacedock --version`: " + strings.TrimSpace(out)
	}
	res := Compare(c, embeddedRange, host, branch)
	if res.Verdict != Compatible {
		return false, res.Message
	}
	_ = runDiscover()
	return true, res.Message
}

// buildVersionStub compiles a tiny stub binary that prints a contract token from
// the SD_STUB_CONTRACT env var on `--version` and records every subcommand it is
// invoked with into the file named by SD_STUB_MARKER. Returns the binary path
// and the marker path.
func buildVersionStub(t *testing.T) (binPath, markerPath string) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	markerPath = filepath.Join(dir, "marker.txt")
	stubSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	if f, err := os.OpenFile(os.Getenv("SD_STUB_MARKER"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		for _, a := range os.Args[1:] {
			fmt.Fprintln(f, a)
		}
		f.Close()
	}
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("spacedock 0.0.0-stub (contract %s)\n", os.Getenv("SD_STUB_CONTRACT"))
	}
}
`
	if err := os.WriteFile(src, []byte(stubSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	binPath = filepath.Join(dir, "spacedock-stub")
	cmd := exec.Command("go", "build", "-o", binPath, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build version stub: %v\n%s", err, out)
	}
	return binPath, markerPath
}

func readMarker(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// parseContractToken extracts the integer N from a `contract N` token in a
// `spacedock --version` line, the same parse the FO Startup step-0 prose does.
func parseContractToken(versionOut string) (int, bool) {
	const marker = "contract "
	idx := strings.Index(versionOut, marker)
	if idx < 0 {
		return 0, false
	}
	rest := versionOut[idx+len(marker):]
	end := strings.IndexAny(rest, ")\n ")
	if end < 0 {
		end = len(rest)
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest[:end]))
	if err != nil {
		return 0, false
	}
	return n, true
}
