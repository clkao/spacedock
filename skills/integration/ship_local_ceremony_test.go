// ABOUTME: AC-5 ship-local ceremony prose check + #217 false-claim removal —
// ABOUTME: the FO shared core names the ceremony and the pr-merge mod prose is honest.
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// subsectionAfter returns the body of the markdown subsection beginning at the
// line equal to heading, up to (but excluding) the next heading at the same or a
// higher level (`### ` or `## `). It is `sectionAfter`'s `###`-aware sibling,
// needed because a `### ` subsection is otherwise swept past `### ` siblings.
func subsectionAfter(text, heading string) string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || strings.HasPrefix(lines[i], "### ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

// TestShipLocalCeremonyBlockExists locks AC-5: the FO shared core carries a
// single named ship-local ceremony block that references the merge-policy branch
// and documents no affirmative `--force` use in its happy-path steps. The check
// scopes to the ceremony subsection so the worktree-removal `--force` audit
// escape (a different subsection) is irrelevant.
func TestShipLocalCeremonyBlockExists(t *testing.T) {
	fo := vendoredSkillFiles(t)["first-officer/references/first-officer-shared-core.md"]
	region := subsectionAfter(fo, "### Ship-Local Ceremony")
	if region == "" {
		t.Fatal("FO shared core missing the `### Ship-Local Ceremony` block (AC-5)")
	}
	if !strings.Contains(region, "merge: local") {
		t.Error("ship-local ceremony must reference the `merge: local` policy branch")
	}
	if !strings.Contains(region, "local-merge:") {
		t.Error("ship-local ceremony must name the local-merge sentinel")
	}
	// The ceremony must state the no-force guarantee, and must contain no
	// affirmative instruction to use --force — every --force token in this block
	// is a negation ("NO --force", "without --force", "never part of...").
	if !strings.Contains(region, "NO `--force`") && !strings.Contains(region, "no `--force`") {
		t.Error("ship-local ceremony must state the no-force guarantee for the happy path")
	}
	for _, affirmative := range []string{"use --force", "pass --force", "use `--force`", "pass `--force`", "with --force", "--force to bypass"} {
		if strings.Contains(region, affirmative) {
			t.Errorf("ship-local ceremony happy path instructs an affirmative --force use: %q", affirmative)
		}
	}
}

// TestPRMergeFallbackProseIsHonest locks the #217 fix: the relocated pr-merge mod
// fallback no longer claims clearing mod-block "keeps that guard satisfied" (the
// guard checks post-update state, not merge order), and the corrected prose names
// both honest no-force paths — the merge: local policy and the post-merge sentinel.
func TestPRMergeFallbackProseIsHonest(t *testing.T) {
	root := skillsRoot(t)
	// The mod was relocated to the workflow definition dir (docs/dev/_mods/),
	// resolved relative to the repo root (the skills/ parent's parent).
	repoRoot := filepath.Dir(root)
	p := filepath.Join(repoRoot, "docs", "dev", "_mods", "pr-merge.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read pr-merge mod: %v", err)
	}
	mod := string(b)

	if strings.Contains(mod, "keeps that guard satisfied") {
		t.Error("pr-merge fallback still carries the false 'keeps that guard satisfied' claim (#217)")
	}
	if !strings.Contains(mod, "pr=local-merge:") {
		t.Error("pr-merge fallback must document the post-merge local-merge sentinel")
	}
	if !strings.Contains(mod, "merge: local") {
		t.Error("pr-merge fallback must document the merge: local policy exemption")
	}
}
