// ABOUTME: AC-1 text-presence audit — the team's proven working habits ship in
// ABOUTME: the three instruction files in plain language with zero insider jargon.
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shippedInstructionFiles is the trio of instruction surfaces this task encodes
// the working principles into: the workflow guide a captain reads, the FO
// operating contract, and the worker (ensign) contract. The map value is a
// human label used in failure messages.
func shippedInstructionFiles(t *testing.T) map[string]string {
	t.Helper()
	root := skillsRoot(t)
	repo := repoRoot(t)
	paths := map[string]string{
		"workflow guide (docs/dev/README.md)":        filepath.Join(repo, "docs", "dev", "README.md"),
		"FO contract (first-officer-shared-core.md)": filepath.Join(root, "first-officer", "references", "first-officer-shared-core.md"),
		"ensign contract (ensign-shared-core.md)":    filepath.Join(root, "ensign", "references", "ensign-shared-core.md"),
	}
	out := make(map[string]string, len(paths))
	for label, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read shipped instruction file %s (%s): %v", label, p, err)
		}
		out[label] = string(b)
	}
	return out
}

// TestShippedInstructionsCarryNoInsiderJargon locks the plain-language guarantee
// of AC-1: the three shipped instruction files contain zero insider-jargon
// tokens. "oracle" is the named token — the design proposal that seeded this work
// uses it pervasively for "external check," and that jargon must not leak into the
// instructions a clean-room contributor reads. The check is case-insensitive so
// "Oracle"/"ORACLE" cannot sneak through.
func TestShippedInstructionsCarryNoInsiderJargon(t *testing.T) {
	bannedJargon := []string{"oracle"}
	for label, content := range shippedInstructionFiles(t) {
		lower := strings.ToLower(content)
		for _, token := range bannedJargon {
			if strings.Contains(lower, token) {
				t.Errorf("%s contains insider-jargon token %q — shipped instructions must be plain language", label, token)
			}
		}
	}
}

// TestWorkflowGuideCarriesPrinciples locks the half of AC-1 that the workflow
// guide owns: the four working principles appear in plain language in the
// workflow-specific stage slots a captain and a worker read. These are durable
// plain-language phrases the README prose commits to — not the proposal's jargon.
func TestWorkflowGuideCarriesPrinciples(t *testing.T) {
	readme := shippedInstructionFiles(t)["workflow guide (docs/dev/README.md)"]
	markers := map[string]string{
		"no doc-only deliverable (real checkable change)": "a real, checkable change",
		"prove by exercising, not by re-reading":          "exercises the behavior and observes the outcome",
		"spike the riskiest unknown first":                "riskiest",
		"code gate preferred over prose-only rule":        "a code gate can enforce",
	}
	for principle, marker := range markers {
		if !strings.Contains(readme, marker) {
			t.Errorf("workflow guide missing the %q principle (expected plain-language marker %q)", principle, marker)
		}
	}
}

// TestFOContractCarriesWorkingPrinciplesAndPosture locks the FO-contract half of
// AC-1: a `## Working Principles` section names the cross-workflow gate discipline
// and the FO posture (name the end value, lead with a yes-able recommendation, do
// obvious reversible work without ceremony). These live in sections disjoint from
// the zs contract reorg.
func TestFOContractCarriesWorkingPrinciplesAndPosture(t *testing.T) {
	fo := shippedInstructionFiles(t)["FO contract (first-officer-shared-core.md)"]
	if !strings.Contains(fo, "## Working Principles") {
		t.Errorf("FO contract missing the `## Working Principles` section")
	}
	postureMarkers := map[string]string{
		"name the end value before starting":          "Name the end value",
		"lead with a yes-able recommendation":         "a recommendation the captain can say yes to",
		"do obvious reversible work without ceremony": "reversible work without ceremony",
	}
	for posture, marker := range postureMarkers {
		if !strings.Contains(fo, marker) {
			t.Errorf("FO contract missing the %q posture (expected marker %q)", posture, marker)
		}
	}
	// The spike-first discipline rides the FO's existing ideation-probe section.
	if !strings.Contains(fo, "riskiest") {
		t.Errorf("FO contract missing the spike-first (riskiest unverified path) discipline")
	}
}

// TestEnsignContractCarriesTestFirstRule locks the ensign-contract half of AC-1:
// the worker contract carries the write-the-failing-test-first rule in plain
// language. Per the companion design study, the authoring-order rule belongs in
// the worker's standing practice (the ensign contract), not in the gate-facing
// workflow template.
func TestEnsignContractCarriesTestFirstRule(t *testing.T) {
	ensign := shippedInstructionFiles(t)["ensign contract (ensign-shared-core.md)"]
	for _, marker := range []string{
		"failing test",
		"watch it fail",
	} {
		if !strings.Contains(ensign, marker) {
			t.Errorf("ensign contract missing the write-the-failing-test-first rule (expected marker %q)", marker)
		}
	}
	// The "no hidden machine dependencies" principle is worker-facing discipline.
	if !strings.Contains(ensign, "hidden") {
		t.Errorf("ensign contract missing the no-hidden-machine-dependencies principle")
	}
}
