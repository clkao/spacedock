// ABOUTME: merge-policy README parsing — local/pr/absent resolve correctly and
// ABOUTME: an unknown value is rejected loudly (native + oracle parity).
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeReadme writes a minimal README carrying the given top-level merge: line
// (or none when mergeLine is empty) into a fresh temp dir and returns the dir.
func writeReadme(t *testing.T, mergeLine string) string {
	t.Helper()
	dir := t.TempDir()
	body := "---\nid-style: sequential\n"
	if mergeLine != "" {
		body += mergeLine + "\n"
	}
	body += "---\n\n# Fixture\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestResolveMergePolicyValues locks the parse: absent and `pr` resolve to
// mergePR, `local` resolves to mergeLocal, and every case is error-free.
func TestResolveMergePolicyValues(t *testing.T) {
	cases := []struct {
		name      string
		mergeLine string
		want      mergePolicy
	}{
		{"absent", "", mergePR},
		{"explicit-pr", "merge: pr", mergePR},
		{"local", "merge: local", mergeLocal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeReadme(t, tc.mergeLine)
			got, err := resolveMergePolicy(dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("policy = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestResolveMergePolicyRejectsUnknown locks that an unknown value is rejected
// loudly (error, not a silent default to mergePR).
func TestResolveMergePolicyRejectsUnknown(t *testing.T) {
	dir := writeReadme(t, "merge: locl")
	_, err := resolveMergePolicy(dir)
	if err == nil {
		t.Fatal("expected an error for an unknown merge: value, got nil")
	}
	if !strings.Contains(err.Error(), "locl") {
		t.Fatalf("error should name the bad value, got %q", err.Error())
	}
}

// TestMergeBogusPolicyRejectedAtSet (parity): a --set on a workflow declaring an
// invalid merge: value exits 1 with the same error in native and oracle, leaving
// the entity unchanged. This is the loud-rejection guarantee end to end.
func TestMergeBogusPolicyRejectedAtSet(t *testing.T) {
	code, out, errOut := assertParity(t, "merge-bogus-workflow", "020-no-sentinel",
		"--set", "020-no-sentinel", "status=done")
	if code != 1 {
		t.Fatalf("invalid merge: policy must reject the --set (exit 1), got %d", code)
	}
	if !strings.Contains(errOut, "merge:") || !strings.Contains(errOut, "bogus") {
		t.Fatalf("stderr should name the invalid policy value, got %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout must be empty on rejection, got %q", out)
	}
}
