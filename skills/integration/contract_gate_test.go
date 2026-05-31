// ABOUTME: AC-2 bracketing test over the vendored FO Startup contract — the
// ABOUTME: embedded contract-range literal must bracket the binary's CONTRACT_VERSION.
package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/contract"
)

// foSharedCore reads the vendored first-officer shared core contract text.
func foSharedCore(t *testing.T) string {
	t.Helper()
	p := filepath.Join(skillsRoot(t), "first-officer", "references", "first-officer-shared-core.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read FO shared core: %v", err)
	}
	return string(b)
}

// embeddedRangeRe matches the half-open contract range literal embedded in the
// Startup step-0 prose (e.g. `>=1,<2`).
var embeddedRangeRe = regexp.MustCompile(`>=\s*\d+\s*,\s*<\s*\d+`)

// TestStartupEmbeddedRangeBracketsContractVersion locks the embedded-range
// bracketing invariant: the range literal embedded in the FO Startup prose
// brackets CONTRACT_VERSION. Both surfaces live in spacedock-v1, so a single go
// test closes the 4th-source-of-truth drift (the FO contract embeds its own
// expected range as a literal).
//
// Oracle: contract.ParseRange (the half-open-range parser) + the compiled
// contract.CONTRACT_VERSION constant. This is NOT bare prose-grep — it does not
// assert "the prose says X"; it parses the embedded literal and checks it
// brackets the binary's real contract version, catching FO/binary range drift.
// The contract-gate ORDERING behavior (gate runs before discover/boot) is owned
// behaviorally by internal/contract/gate_test.go, which drives a real spacedock
// stub --version and observes discover invoked 0×/1×.
func TestStartupEmbeddedRangeBracketsContractVersion(t *testing.T) {
	startup := sectionAfter(foSharedCore(t), "## Startup")
	raw := embeddedRangeRe.FindString(startup)
	if raw == "" {
		t.Fatalf("Startup section has no embedded contract range literal (>=N,<M)")
	}
	lo, hi, err := contract.ParseRange(raw)
	if err != nil {
		t.Fatalf("embedded range %q does not parse: %v", raw, err)
	}
	if !(lo <= contract.CONTRACT_VERSION && contract.CONTRACT_VERSION < hi) {
		t.Fatalf("embedded Startup range %s does not bracket CONTRACT_VERSION=%d", raw, contract.CONTRACT_VERSION)
	}
	// Guard against a stray literal: the embedded range must be a single
	// occurrence in the Startup section (one source of truth, not several).
	if got := len(embeddedRangeRe.FindAllString(startup, -1)); got != 1 {
		t.Fatalf("Startup section has %d embedded range literals, want exactly 1", got)
	}
}
