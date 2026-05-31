// ABOUTME: AC-2 static + bracketing tests over the vendored FO Startup contract —
// ABOUTME: step-0 ordering, the contract token parse, and embedded-range bracketing.
package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// TestStartupStepZeroIsContractGate locks AC-2 oracle (1): the Startup section's
// FIRST numbered step runs `spacedock --version`, parses the `contract` token,
// and aborts before the `--discover` / `--boot` reads. The ordering is the
// load-bearing claim — the gate must precede discovery/boot.
func TestStartupStepZeroIsContractGate(t *testing.T) {
	startup := sectionAfter(foSharedCore(t), "## Startup")
	if startup == "" {
		t.Fatalf("FO shared core has no `## Startup` section")
	}

	// The first numbered step (`1.`) must be the contract gate.
	firstStep := firstNumberedStep(startup)
	if firstStep == "" {
		t.Fatalf("Startup section has no numbered steps")
	}
	if !strings.Contains(firstStep, "spacedock --version") {
		t.Errorf("Startup step 1 does not run `spacedock --version`:\n%s", firstStep)
	}
	if !strings.Contains(firstStep, "contract") {
		t.Errorf("Startup step 1 does not parse the `contract` token:\n%s", firstStep)
	}
	if !strings.Contains(strings.ToUpper(firstStep), "ABORT") {
		t.Errorf("Startup step 1 does not abort on mismatch:\n%s", firstStep)
	}

	// The gate must precede the discovery and boot reads in the section.
	versionIdx := strings.Index(startup, "spacedock --version")
	discoverIdx := strings.Index(startup, "spacedock status --discover")
	bootIdx := strings.Index(startup, "spacedock status --boot")
	if discoverIdx >= 0 && versionIdx > discoverIdx {
		t.Errorf("contract gate (idx %d) does not precede --discover (idx %d)", versionIdx, discoverIdx)
	}
	if bootIdx >= 0 && versionIdx > bootIdx {
		t.Errorf("contract gate (idx %d) does not precede --boot (idx %d)", versionIdx, bootIdx)
	}
}

// TestStartupEmbeddedRangeBracketsContractVersion locks AC-2 oracle (2): the
// range literal embedded in the FO Startup prose brackets CONTRACT_VERSION. Both
// surfaces live in spacedock-v1, so a single go test closes the 4th-source-of-
// truth drift (the FO contract embeds its own expected range as a literal).
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

// firstNumberedStep returns the body of the first `1.`-prefixed list item in a
// section, up to the next `2.` numbered item (or section end). Used to scope the
// step-1-is-the-gate assertion.
func firstNumberedStep(section string) string {
	lines := strings.Split(section, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "1. ") {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "2. ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}
