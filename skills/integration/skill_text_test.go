// ABOUTME: Absence-invariant tests over the vendored FO/ensign skill surface —
// ABOUTME: AC-1 (no plugin status path) and AC-6 (no new PR-merge / `## Hook:` mod), oracle = the structural scope-fence.
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// skillsRoot is the vendored skill tree under test (the project skills/ dir
// this test package lives inside).
func skillsRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// vendoredSkillFiles returns the vendored skill instruction surface: the FO and
// ensign reference markdown plus the vendored claude-team helper. The vendored
// status library is excluded — it is the status oracle, not skill instruction
// text, and legitimately carries the literal status filename internally.
func vendoredSkillFiles(t *testing.T) map[string]string {
	t.Helper()
	root := skillsRoot(t)
	rel := []string{
		"first-officer/references/first-officer-shared-core.md",
		"first-officer/references/claude-first-officer-runtime.md",
		"ensign/references/ensign-shared-core.md",
		"commission/bin/claude-team",
	}
	out := make(map[string]string, len(rel))
	for _, r := range rel {
		p := filepath.Join(root, r)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read vendored skill file %s: %v", p, err)
		}
		out[r] = string(b)
	}
	return out
}

// TestNoPluginStatusPathInVendoredSkills locks AC-1: no file in the vendored
// skill instruction surface references the plugin-private status path.
func TestNoPluginStatusPathInVendoredSkills(t *testing.T) {
	for name, content := range vendoredSkillFiles(t) {
		if strings.Contains(content, "skills/commission/bin/status") {
			t.Errorf("%s references plugin-private status path 'skills/commission/bin/status'", name)
		}
		if strings.Contains(content, "spacedock_plugin_dir") {
			t.Errorf("%s still references {spacedock_plugin_dir} plugin root", name)
		}
	}
}

// TestNoPRMergeOrModBehaviorIntroduced locks AC-6: the vendored skill surface
// introduces no new `## Hook:` mod heading and no PR-merge flow beyond the
// existing mod-block convention the surface already documents. The vendored
// files are copies of the plugin skill text plus the three amendments; the
// amendments add no new lifecycle hook or PR-merge command. This asserts the
// amendment regions do not introduce a `## Hook:` heading and that no PR-merge
// invocation (gh pr merge / git merge --no-ff into main) was added.
//
// Oracle: the amendment-region scope-fence — an absence invariant over the
// vendored on-disk skill surface (the ensign file and the FO Split-Root
// amendment region, scoped via sectionAfter). No positive behavioral seam can
// prove an absence of behavior: a re-introduced `## Hook:` lifecycle mod or a
// new PR-merge command would silently change the dispatch lifecycle, and only
// this structural scope-fence over the amendment regions catches it. This is
// NOT bare prose-grep — it asserts a structural negative over the amendments,
// not the presence of an instruction clause.
func TestNoPRMergeOrModBehaviorIntroduced(t *testing.T) {
	files := vendoredSkillFiles(t)

	// The only `## Hook:` text legitimately present is the pre-existing Mod Hook
	// Convention documentation in the FO shared core (describing startup/idle/
	// merge points). The amendments must not add a NEW `## Hook: {point}` mod
	// declaration. Assert the ensign file (which the split-root amendment B
	// touched) introduces no `## Hook:` heading at all.
	ensign := files["ensign/references/ensign-shared-core.md"]
	if strings.Contains(ensign, "## Hook:") {
		t.Errorf("ensign reference unexpectedly introduces a `## Hook:` heading")
	}

	// The amendment-introduced region in the FO file is the Split-Root Worktree
	// Contract subsection. Assert that region introduces no `## Hook:` heading —
	// the pre-existing Mod Hook Convention text lives in a different section.
	fo := files["first-officer/references/first-officer-shared-core.md"]
	if region := sectionAfter(fo, "### Split-Root Worktree Contract"); strings.Contains(region, "## Hook:") {
		t.Errorf("FO split-root amendment region introduces a `## Hook:` heading")
	}

	// No PR-merge invocation may be introduced anywhere in the vendored surface.
	prMergeMarkers := []string{"gh pr merge", "git merge --no-ff", "git merge --ff-only main"}
	for name, content := range files {
		for _, m := range prMergeMarkers {
			if strings.Contains(content, m) {
				t.Errorf("%s introduces a PR-merge invocation %q (out of scope per AC-6)", name, m)
			}
		}
	}
}

// TestCommissionStateBackendDecisionRule locks AC-2: the commission SKILL.md
// carries the split-root-vs-single-root state-backend decision rule, so a newly
// commissioned workflow no longer defaults to single-root with no guidance. The
// three load-bearing fragments are the split-root frontmatter spelling, the
// split-root trigger phrase, and the single-root "omit" guidance.
func TestCommissionStateBackendDecisionRule(t *testing.T) {
	p := filepath.Join(skillsRoot(t), "commission", "SKILL.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read commission SKILL.md: %v", err)
	}
	skill := string(b)

	for _, frag := range []string{
		"state: .spacedock-state",
		"embedded in a code repo whose PRs you care about",
		"omit `state:`",
	} {
		if !strings.Contains(skill, frag) {
			t.Errorf("commission SKILL.md missing state-backend decision-rule fragment %q", frag)
		}
	}
}

// sectionAfter returns the body of the markdown section beginning at the line
// equal to heading, up to (but excluding) the next top-level `## ` heading, or
// "" when the heading is absent. Used to scope an assertion to one section.
func sectionAfter(text, heading string) string {
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
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}
