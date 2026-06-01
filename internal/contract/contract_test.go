// ABOUTME: Unit tests for the contract-version axis: CONTRACT_VERSION sanity,
// ABOUTME: half-open range parsing, and the five-verdict compatibility compare.
package contract

import (
	"strconv"
	"strings"
	"testing"
)

// TestContractVersionIsPositiveInteger guards the axis invariant: CONTRACT_VERSION
// is a positive integer (the monotonic compatibility axis, not semver).
func TestContractVersionIsPositiveInteger(t *testing.T) {
	if CONTRACT_VERSION < 1 {
		t.Fatalf("CONTRACT_VERSION = %d, want >= 1", CONTRACT_VERSION)
	}
}

// TestParseRange covers the half-open range grammar ">=N,<M": accepted forms,
// the bracket bounds, and every malformed shape that must be rejected.
func TestParseRange(t *testing.T) {
	cases := []struct {
		raw     string
		wantLo  int
		wantHi  int
		wantErr bool
	}{
		{">=1,<2", 1, 2, false},
		{">=1,<3", 1, 3, false},
		{">=10,<11", 10, 11, false},
		{">= 1 , < 2", 1, 2, false}, // surrounding whitespace tolerated
		{"", 0, 0, true},
		{">=1", 0, 0, true},        // missing upper bound
		{"<2", 0, 0, true},         // missing lower bound
		{">1,<2", 0, 0, true},      // wrong lower operator
		{">=1,<=2", 0, 0, true},    // wrong upper operator
		{">=a,<2", 0, 0, true},     // non-integer lower
		{">=1,<b", 0, 0, true},     // non-integer upper
		{">=2,<2", 0, 0, true},     // empty interval (lo >= hi)
		{">=3,<2", 0, 0, true},     // inverted interval
		{">=1,<2,<3", 0, 0, true},  // extra clause
		{"1,2", 0, 0, true},        // no operators
		{">=1.0,<2.0", 0, 0, true}, // float bounds
	}
	for _, c := range cases {
		lo, hi, err := ParseRange(c.raw)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseRange(%q) = (%d,%d,nil), want error", c.raw, lo, hi)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRange(%q) unexpected error: %v", c.raw, err)
			continue
		}
		if lo != c.wantLo || hi != c.wantHi {
			t.Errorf("ParseRange(%q) = (%d,%d), want (%d,%d)", c.raw, lo, hi, c.wantLo, c.wantHi)
		}
	}
}

// TestCompare drives the five-verdict comparison over a contract C and a raw
// range, asserting the verdict class and that each non-compatible verdict's
// message carries its pinned remedy substring.
func TestCompare(t *testing.T) {
	cases := []struct {
		name        string
		contract    int
		raw         string
		wantVerdict Verdict
		wantPinned  string // substring the message must contain (empty = no check)
	}{
		{"compatible-exact", 1, ">=1,<2", Compatible, ""},
		{"compatible-forward-tolerant", 2, ">=1,<3", Compatible, ""},
		{"compatible-lower-edge", 1, ">=1,<3", Compatible, ""},
		{"too-old-binary", 1, ">=2,<3", TooOldBinary, "too-old-binary"},
		{"too-old-plugin-at-hi", 2, ">=1,<2", TooOldPlugin, "too-old-plugin"},
		{"too-old-plugin-above-hi", 5, ">=1,<2", TooOldPlugin, "too-old-plugin"},
		{"malformed", 1, ">=1", MalformedRange, "malformed contract range"},
		{"predates-contract-empty", 1, "", PluginPredatesContract, "spacedock init --host claude"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := Compare(c.contract, c.raw, "claude", "")
			if res.Verdict != c.wantVerdict {
				t.Fatalf("Compare(%d,%q) verdict = %v, want %v", c.contract, c.raw, res.Verdict, c.wantVerdict)
			}
			if c.wantPinned != "" && !strings.Contains(res.Message, c.wantPinned) {
				t.Fatalf("Compare(%d,%q) message = %q, want substring %q", c.contract, c.raw, res.Message, c.wantPinned)
			}
			// The compatible verdict carries no remedy; non-compatible ones do.
			if c.wantVerdict == Compatible && res.Message == "" {
				t.Fatalf("compatible verdict should still carry an OK message")
			}
		})
	}
}

// TestCompareMessageShape locks the shared mismatch-message header shape: the
// leading "Spacedock contract mismatch" line names contract C and the range, and
// the message ends with the "Run `spacedock doctor`" pointer — for every
// mismatch class except malformed-range (which is a packaging bug, not a
// too-old install) and no-plugin-found.
func TestCompareMessageShape(t *testing.T) {
	for _, c := range []struct {
		contract int
		raw      string
	}{
		{1, ">=2,<3"}, // too-old-binary
		{2, ">=1,<2"}, // too-old-plugin
	} {
		res := Compare(c.contract, c.raw, "claude", "")
		header := "Spacedock contract mismatch: binary is contract " + strconv.Itoa(c.contract)
		if !strings.Contains(res.Message, header) {
			t.Errorf("Compare(%d,%q) message missing header %q: %q", c.contract, c.raw, header, res.Message)
		}
		if !strings.Contains(res.Message, "Run `spacedock doctor`") {
			t.Errorf("Compare(%d,%q) message missing doctor pointer: %q", c.contract, c.raw, res.Message)
		}
	}
}

// TestPluginPredatesContractRemedy locks the new verdict for an absent/empty
// requires-contract: it names the `spacedock init --host <host>` one-liner,
// reflects the dev branch (@next) when set, and OMITS the `plugin update`
// fallback that reusing too-old-plugin would drag in (that fallback no-ops on a
// stale install). A whitespace-only value routes here too; a non-empty
// unparseable value still reads as a packaging bug.
func TestPluginPredatesContractRemedy(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		res := Compare(1, raw, "claude", "next")
		if res.Verdict != PluginPredatesContract {
			t.Fatalf("Compare(1,%q) verdict = %v, want plugin-predates-contract", raw, res.Verdict)
		}
		if !strings.Contains(res.Message, "spacedock init --host claude") {
			t.Errorf("predates-contract remedy missing init one-liner: %q", res.Message)
		}
		if !strings.Contains(res.Message, "@next") {
			t.Errorf("predates-contract remedy missing @next branch: %q", res.Message)
		}
		if strings.Contains(res.Message, "plugin update") {
			t.Errorf("predates-contract remedy must omit the no-op `plugin update` fallback: %q", res.Message)
		}
	}

	// With no dev branch, the remedy is the clean release one-liner (no @suffix).
	plain := Compare(1, "", "claude", "")
	if strings.Contains(plain.Message, "@next") {
		t.Errorf("predates-contract remedy with no branch should omit @next: %q", plain.Message)
	}
	if !strings.Contains(plain.Message, "spacedock init --host claude") {
		t.Errorf("predates-contract remedy with no branch missing init one-liner: %q", plain.Message)
	}

	// A non-empty unparseable value is still a packaging bug, not predates-contract.
	bug := Compare(1, ">=1", "claude", "next")
	if bug.Verdict != MalformedRange {
		t.Fatalf("Compare(1,%q) verdict = %v, want malformed-range", ">=1", bug.Verdict)
	}
	if !strings.Contains(bug.Message, "This is a packaging bug") {
		t.Errorf("non-empty malformed should keep the packaging-bug message: %q", bug.Message)
	}
}

// TestCompareHostSubstitution verifies the host parameter is woven into the
// too-old-plugin remedy (the only place an install/update host appears). The
// remedy must name the live `spacedock install` command, not the removed `init`
// (which now exits 2) — the remedy a user hits at the gate must run.
func TestCompareHostSubstitution(t *testing.T) {
	for _, host := range []string{"claude", "codex"} {
		res := Compare(2, ">=1,<2", host, "")
		want := "spacedock install --host " + host
		if !strings.Contains(res.Message, want) {
			t.Errorf("too-old-plugin remedy for host %q missing %q: %q", host, want, res.Message)
		}
		if strings.Contains(res.Message, "spacedock init") {
			t.Errorf("too-old-plugin remedy for host %q names the removed init command: %q", host, res.Message)
		}
	}
}

// TestCompareBranchSubstitution verifies the pre-release dev branch, when set,
// is woven into the too-old-binary remedy's reinstall hint.
func TestCompareBranchSubstitution(t *testing.T) {
	res := Compare(1, ">=2,<3", "claude", "next")
	if !strings.Contains(res.Message, "@next") {
		t.Errorf("too-old-binary remedy with branch=next missing @next: %q", res.Message)
	}
	// Default release path omits the branch suffix.
	plain := Compare(1, ">=2,<3", "claude", "")
	if strings.Contains(plain.Message, "@next") {
		t.Errorf("too-old-binary remedy with no branch should omit @next: %q", plain.Message)
	}
}
