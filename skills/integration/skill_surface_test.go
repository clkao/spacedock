// ABOUTME: AC-1 skill-surface audit — the five user skills ship with valid
// ABOUTME: SKILL.md + resolvable reference closure, integration is test-only.
package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// userSkills is the published user skill surface: the five skills the host
// discovers (each owns a SKILL.md). `integration` is deliberately absent — it
// holds only *_test.go and must not publish.
var userSkills = []string{"commission", "debrief", "refit", "ensign", "first-officer"}

// TestUserSkillsPresentWithFrontmatter locks AC-1: each of the five user skills
// ships a SKILL.md whose YAML frontmatter declares a `name` and a `description`.
func TestUserSkillsPresentWithFrontmatter(t *testing.T) {
	root := skillsRoot(t)
	for _, skill := range userSkills {
		path := filepath.Join(root, skill, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("user skill %q has no SKILL.md at %s: %v", skill, path, err)
			continue
		}
		fm, ok := frontmatter(string(data))
		if !ok {
			t.Errorf("%s/SKILL.md has no YAML frontmatter block", skill)
			continue
		}
		if !strings.Contains(fm, "name:") {
			t.Errorf("%s/SKILL.md frontmatter missing a name field", skill)
		}
		if !strings.Contains(fm, "description:") {
			t.Errorf("%s/SKILL.md frontmatter missing a description field", skill)
		}
	}
}

// TestIntegrationIsTestOnlyAndExcluded locks AC-1's exclusion mechanism: the
// `skills/integration/` directory carries Go test files and NO SKILL.md, so the
// host's SKILL.md-keyed discovery omits it from the published skill set — no
// allow/deny list needed. The audit verifies the property rather than presuming
// it: integration has *_test.go and zero SKILL.md.
func TestIntegrationIsTestOnlyAndExcluded(t *testing.T) {
	root := skillsRoot(t)
	dir := filepath.Join(root, "integration")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read integration dir: %v", err)
	}
	sawTest := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == "SKILL.md" {
			t.Errorf("skills/integration ships a SKILL.md — it would publish as a user skill")
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			sawTest = true
		}
	}
	if !sawTest {
		t.Errorf("skills/integration has no *_test.go — expected the test-only surface")
	}
}

// referenceRe matches the two reference-include forms a SKILL.md uses: an
// `@references/foo.md` directive and a bare `references/foo.md` read path. The
// path captured in group 1 is resolved relative to the skill directory.
var referenceRe = regexp.MustCompile(`@?(references/[A-Za-z0-9_./-]+\.md)`)

// TestUserSkillReferenceClosureResolves locks AC-1's closure half: every
// `@references/...md` / `references/...md` path mentioned in a user SKILL.md
// resolves to a real file under that skill's directory. A dangling reference
// (a ported skill pointing at a path that does not exist on `next`) fails here.
// Brace-placeholder template paths (e.g. references/templates/{name}.md) are
// resolved against their concrete siblings rather than the literal `{name}`.
func TestUserSkillReferenceClosureResolves(t *testing.T) {
	root := skillsRoot(t)
	for _, skill := range userSkills {
		skillDir := filepath.Join(root, skill)
		data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
		if err != nil {
			t.Errorf("%s: %v", skill, err)
			continue
		}
		for _, m := range referenceRe.FindAllStringSubmatch(string(data), -1) {
			rel := m[1]
			if strings.Contains(rel, "{") {
				// A brace placeholder (references/templates/{name}.md): assert the
				// parent directory exists and holds at least one concrete .md.
				parent := filepath.Join(skillDir, filepath.Dir(rel))
				glob, _ := filepath.Glob(filepath.Join(parent, "*.md"))
				if len(glob) == 0 {
					t.Errorf("%s: templated reference %q has no concrete .md under %s", skill, rel, parent)
				}
				continue
			}
			if _, err := os.Stat(filepath.Join(skillDir, rel)); err != nil {
				t.Errorf("%s: dangling reference %q (resolved %s): %v", skill, rel, filepath.Join(skillDir, rel), err)
			}
		}
	}
}

// TestNoPluginPrivateStatusPathInUserSkills locks AC-1/AC-2 reconciliation: no
// user skill (SKILL.md or its references) or the canonical pr-merge mod
// references the plugin-private status path or the {spacedock_plugin_dir} token.
// The reconciled surface calls `spacedock status`; a blind-copied python-era
// path fails here.
func TestNoPluginPrivateStatusPathInUserSkills(t *testing.T) {
	root := skillsRoot(t)
	repo := repoRoot(t)
	banned := []string{
		"skills/commission/bin/status",
		"commission/bin/status",
		"{spacedock_plugin_dir}",
		".agents/plugins/marketplace.json",
	}
	for _, path := range shippedSkillText(t, root, repo) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		content := string(data)
		for _, b := range banned {
			if strings.Contains(content, b) {
				t.Errorf("%s references banned plugin-private path %q", path, b)
			}
		}
	}
}

// shippedSkillText returns every markdown file under skills/ (excluding the
// test-only integration dir) plus the canonical mods/, the full shipped
// instruction surface the banned-path audit walks.
func shippedSkillText(t *testing.T, skillsRootDir, repoRootDir string) []string {
	t.Helper()
	var out []string
	walk := func(base string) {
		filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "integration" {
				return filepath.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(p, ".md") {
				out = append(out, p)
			}
			return nil
		})
	}
	walk(skillsRootDir)
	walk(filepath.Join(repoRootDir, "mods"))
	return out
}

// frontmatter returns the YAML frontmatter block (between the leading `---` and
// the next `---`) and whether the document opened with one.
func frontmatter(doc string) (string, bool) {
	if !strings.HasPrefix(doc, "---\n") {
		return "", false
	}
	rest := doc[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}
