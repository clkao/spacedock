// ABOUTME: AC-2 table --fields de-dupe — an explicit field naming a default no
// ABOUTME: longer duplicates the column; non-default names still append as extras.
package status

import (
	"path/filepath"
	"strings"
	"testing"
)

// headerOf returns the first line of out (the table header row).
func headerOf(out string) string {
	return strings.SplitN(out, "\n", 2)[0]
}

// countToken returns how many whitespace-separated tokens of line equal tok.
func countToken(line, tok string) int {
	n := 0
	for _, t := range strings.Fields(line) {
		if t == tok {
			n++
		}
	}
	return n
}

// TestFieldsDedupeNoDuplicateDefaultColumns is the bug's reproduction-and-fix
// oracle (AC-2): `--fields id,status` on the default table must emit the six
// default columns ONCE — no duplicate ID/STATUS extras. Before the de-dupe the
// header carried ID and STATUS twice; this asserts exactly one of each.
func TestFieldsDedupeNoDuplicateDefaultColumns(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--fields", "id,status")
	if code != 0 {
		t.Fatalf("native exit=%d stderr=%q", code, errOut)
	}
	header := headerOf(out)
	if c := countToken(header, "ID"); c != 1 {
		t.Fatalf("header has %d ID columns, want exactly 1 (duplicate-column bug)\nheader: %q", c, header)
	}
	if c := countToken(header, "STATUS"); c != 1 {
		t.Fatalf("header has %d STATUS columns, want exactly 1 (duplicate-column bug)\nheader: %q", c, header)
	}
}

// TestFieldsNonDefaultStillAppends locks the additive half of the divergence: a
// --fields name that is not a default table column still appends as an extra.
// `worktree` is a non-default frontmatter key, so WORKTREE must appear once
// while the named default `id` is not duplicated.
func TestFieldsNonDefaultStillAppends(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--fields", "id,worktree")
	if code != 0 {
		t.Fatalf("native exit=%d stderr=%q", code, errOut)
	}
	header := headerOf(out)
	if c := countToken(header, "WORKTREE"); c != 1 {
		t.Fatalf("non-default field WORKTREE appears %d times, want 1: %q", c, header)
	}
	if c := countToken(header, "ID"); c != 1 {
		t.Fatalf("ID column duplicated alongside non-default extra: %q", header)
	}
}

// TestNextFieldsStatusUnchanged locks that the de-dupe is scoped to the table's
// DISPLAYED columns: on --next, `status` is a frontmatter key but NOT a
// displayed column (--next shows ID SLUG CURRENT NEXT WORKTREE), so
// `--next --fields status` still appends STATUS as an extra. The de-dupe must
// not suppress it — only true duplicates of a displayed column are removed.
func TestNextFieldsStatusUnchanged(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--next", "--fields", "status")
	if code != 0 {
		t.Fatalf("native exit=%d stderr=%q", code, errOut)
	}
	header := headerOf(out)
	if c := countToken(header, "STATUS"); c != 1 {
		t.Fatalf("--next --fields status should still append STATUS once, got %d: %q", c, header)
	}
}

// TestNextFieldsIDSuppressed locks the de-dupe on the --next path for a name
// that IS a displayed --next column: `id` is rendered as the first --next
// column, so `--next --fields id` must NOT append a duplicate ID extra.
func TestNextFieldsIDSuppressed(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--next", "--fields", "id")
	if code != 0 {
		t.Fatalf("native exit=%d stderr=%q", code, errOut)
	}
	header := headerOf(out)
	if c := countToken(header, "ID"); c != 1 {
		t.Fatalf("--next --fields id should not duplicate the displayed ID column, got %d: %q", c, header)
	}
}
