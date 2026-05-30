// ABOUTME: Static skill-text tests over the vendored FO/ensign skill surface —
// ABOUTME: AC-1 (no plugin status path), AC-2 (launcher flags), AC-6 (no PR/mod), AC-7 (concurrency-safe commits).
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

// TestLauncherStatusInvocations locks AC-2: the FO reference text issues the
// load-bearing status reads and mutations through `spacedock status`, with each
// flag's role preserved for startup discovery, --boot, dispatch --set, and merge
// --archive.
func TestLauncherStatusInvocations(t *testing.T) {
	files := vendoredSkillFiles(t)
	fo := files["first-officer/references/first-officer-shared-core.md"]

	wantSubstrings := []struct {
		role string
		text string
	}{
		{"startup discovery", "spacedock status --discover"},
		{"--boot", "spacedock status --boot"},
		{"dispatch --set", "spacedock status --workflow-dir {workflow_dir} --set {slug} status={next_stage}"},
		{"merge --archive", "spacedock status --workflow-dir {workflow_dir} --archive {slug}"},
		{"overview read", "spacedock status --workflow-dir {workflow_dir}"},
	}
	for _, w := range wantSubstrings {
		if !strings.Contains(fo, w.text) {
			t.Errorf("FO reference missing launcher invocation for %s: %q", w.role, w.text)
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

// TestConcurrencySafeCommitClause locks AC-7 (static half): the vendored
// ensign/FO commit instructions require concurrency-safe (path-scoped or
// tool-owned) state commits and forbid bare `git add -A` / `git commit` in the
// state checkout.
func TestConcurrencySafeCommitClause(t *testing.T) {
	files := vendoredSkillFiles(t)
	for _, name := range []string{
		"ensign/references/ensign-shared-core.md",
		"first-officer/references/first-officer-shared-core.md",
	} {
		content := files[name]
		if !strings.Contains(content, "path-scoped") {
			t.Errorf("%s missing the path-scoped state-commit requirement", name)
		}
		if !strings.Contains(content, "git -C {state_checkout} commit -m") {
			t.Errorf("%s missing the path-scoped commit form", name)
		}
		if !strings.Contains(content, "Never a bare `git add -A`") {
			t.Errorf("%s missing the bare-commit prohibition", name)
		}
		if !strings.Contains(content, "tool-managed atomic state commits") {
			t.Errorf("%s missing the preferred tool-owned commit option", name)
		}
	}
}

// TestSplitRootContractClause locks the amendment-B contract text: a worktree
// stage isolates CODE only and the entity body + stage reports go to the shared
// state checkout, in both the FO and ensign vendored surfaces.
func TestSplitRootContractClause(t *testing.T) {
	files := vendoredSkillFiles(t)
	for _, name := range []string{
		"ensign/references/ensign-shared-core.md",
		"first-officer/references/first-officer-shared-core.md",
	} {
		content := files[name]
		if !strings.Contains(content, "Split-Root Worktree Contract") {
			t.Errorf("%s missing the Split-Root Worktree Contract section", name)
		}
		if !strings.Contains(content, "CODE only") {
			t.Errorf("%s split-root contract does not state CODE-only worktree isolation", name)
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
