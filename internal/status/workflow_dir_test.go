// ABOUTME: Vendoring-landmine negative test — without --workflow-dir the launcher
// ABOUTME: surfaces the empty dirname(__file__) scan instead of faking parity.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestOmittedWorkflowDirSurfacesEmptyScan locks the vendoring landmine: the
// vendored script resolves its target as `--workflow-dir or $PIPELINE_DIR or
// dirname(__file__)`. Once vendored (and materialized to a temp file), the
// dirname(__file__) fallback points at an empty directory, not the populated
// fixture. So a no---workflow-dir invocation MUST scan that empty dir and render
// an empty table — even when cwd is a populated workflow — rather than silently
// falling back to cwd or the plugin commission dir and masquerading as parity.
func TestOmittedWorkflowDirSurfacesEmptyScan(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	// cwd is the populated fixture; --workflow-dir is intentionally omitted.
	out, stderr, code := runLauncher(t, root, env)
	if code != 0 {
		t.Fatalf("launcher exit=%d stderr=%q", code, stderr)
	}

	// Header rows render, but no entity rows: the scan target is the empty
	// dirname(__file__), not the cwd fixture.
	if !strings.Contains(out, "ID") || !strings.Contains(out, "SLUG") {
		t.Fatalf("expected a table header, got:\n%s", out)
	}
	for _, slug := range []string{"001-design-seam", "002-vendor-script", "003-wire-cli", "004-no-score"} {
		if strings.Contains(out, slug) {
			t.Fatalf("omitted --workflow-dir leaked cwd fixture entity %q — it must NOT fall back to cwd:\n%s", slug, out)
		}
	}

	// Contrast: WITH --workflow-dir the same launcher run shows the fixture
	// entities, proving the empty result above is the fallback, not a bug.
	withDir, _, withCode := runLauncher(t, root, env, "--workflow-dir", root)
	if withCode != 0 {
		t.Fatalf("launcher --workflow-dir exit=%d", withCode)
	}
	if !strings.Contains(withDir, "001-design-seam") {
		t.Fatalf("with --workflow-dir the fixture entities should appear:\n%s", withDir)
	}
}
