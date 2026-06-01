// ABOUTME: B5/B6 contract skill-text tests — assert the FO halt-gate + push/pull
// ABOUTME: sync + rebase-conflict halt prose is present at the FO and ensign homes.
package hostneutrality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// foCorePath is the FO shared-core contract.
var foCorePath = filepath.Join("..", "..", "skills", "first-officer", "references",
	"first-officer-shared-core.md")

// ensignCorePath is the ensign shared-core contract.
var ensignCorePath = filepath.Join("..", "..", "skills", "ensign", "references",
	"ensign-shared-core.md")

// commissionSkillPath is the commission SKILL.md.
var commissionSkillPath = filepath.Join("..", "..", "skills", "commission", "SKILL.md")

// readSkill reads a skill file, failing the test on error.
func readSkill(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// assertAll fails with a per-token diagnostic for each token missing from text.
func assertAll(t *testing.T, name, text string, tokens []string) {
	t.Helper()
	for _, tok := range tokens {
		if !strings.Contains(text, tok) {
			t.Errorf("%s missing required contract token: %q", name, tok)
		}
	}
}

// TestFOHaltGateProse pins B5: the FO core carries the boot halt-gate keyed on
// the Phase-A boot fields (split-root && entity_dir_present false → halt dispatch,
// point at `spacedock state init`).
func TestFOHaltGateProse(t *testing.T) {
	text := readSkill(t, foCorePath)
	assertAll(t, "FO core (B5 halt-gate)", text, []string{
		"state_backend",
		"entity_dir_present",
		"split-root",
		"spacedock state init",
		"HALT",
	})
}

// TestFOSyncProse pins the FO half of B6: pull --rebase on boot, push after a
// state commit, and the M-3 rebase-conflict halt (abort + surface + no
// force-push, no auto-resolve).
func TestFOSyncProse(t *testing.T) {
	text := readSkill(t, foCorePath)
	assertAll(t, "FO core (B6 sync)", text, []string{
		"pull --rebase",
		"push origin",
		"rebase --abort",
		"--force",
		"auto-resolve",
		"must NOT",
	})
}

// TestEnsignSyncProse pins the ensign half of B6: push after committing, pull
// --rebase on a push rejection, and the M-3 rebase-conflict halt (abort +
// surface + no force-push, no auto-resolve), alongside the path-scoped rule.
func TestEnsignSyncProse(t *testing.T) {
	text := readSkill(t, ensignCorePath)
	assertAll(t, "ensign core (B6 sync)", text, []string{
		"push origin",
		"pull --rebase",
		"rebase --abort",
		"--force",
		"auto-resolve",
	})
}

// TestCommissionJourneyProse pins B3: the commission SKILL.md carries the
// journey-1 orphan-branch mechanics (clear inherited tree, linked worktree,
// state init pointer) and the journey-2 $inline prose.
func TestCommissionJourneyProse(t *testing.T) {
	text := readSkill(t, commissionSkillPath)
	assertAll(t, "commission SKILL.md (journeys)", text, []string{
		"checkout --orphan",
		"clear",
		"linked worktree",
		"spacedock state init",
		"$inline",
	})
}
