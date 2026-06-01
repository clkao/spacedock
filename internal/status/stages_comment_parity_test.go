// ABOUTME: AC-4 stage-field inline-comment strip parity — a commented stage
// ABOUTME: concurrency drives --next identically for native and oracle (not a default fallback).
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStageInlineCommentDispatchParity locks the #163 stage-field strip end-to-end
// through --next: a stage declaring `concurrency: 1  # cap` must cap dispatch at 1
// (the stripped value), not 2 (the silent atoiOr default the un-stripped
// `1  # cap` would fall back to). Asserted native-vs-oracle so the strip is
// parity-pinned on the command path, and the dispatch count is checked to prove
// the stripped value — not the default — drove the scheduling.
func TestStageInlineCommentDispatchParity(t *testing.T) {
	env := pinnedEnv(t)
	root := t.TempDir()

	readme := `---
entity-type: task
id-style: slug
stages:
  defaults:
    worktree: false  # default
    concurrency: 1  # cap
  states:
    - name: ideation
      initial: true
    - name: build  # the build stage
      concurrency: 1  # only one at a time
    - name: done
      terminal: true
---

# Inline-comment stage fixture
`
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	// Two ideation entities are both eligible to advance into build; build's
	// stripped concurrency is 1, so exactly one may dispatch this round.
	for _, slug := range []string{"alpha", "beta"} {
		body := "---\nid: " + slug + "\ntitle: " + slug + "\nstatus: ideation\nscore: \"0.5\"\nsource: probe\n---\nbody\n"
		if err := os.WriteFile(filepath.Join(root, slug+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitInit(t, root)

	nOut, nErr, nCode := runNative(t, root, env, "--workflow-dir", root, "--next")
	oOut, oErr, oCode := runOracle(t, root, env, "--workflow-dir", root, "--next")
	if nCode != 0 || oCode != 0 {
		t.Fatalf("exit: native=%d (%q) oracle=%d (%q)", nCode, nErr, oCode, oErr)
	}
	if nOut != oOut {
		t.Fatalf("--next native vs oracle mismatch\n--- native ---\n%s\n--- oracle ---\n%s", nOut, oOut)
	}
	// Count dispatch rows (lines naming build as NEXT). The stripped concurrency
	// 1 caps this at one; an un-stripped value would fall back to the default 2
	// and dispatch both, which this asserts against.
	rows := 0
	for _, line := range strings.Split(strings.TrimRight(nOut, "\n"), "\n") {
		if strings.Contains(line, "build") && (strings.Contains(line, "alpha") || strings.Contains(line, "beta")) {
			rows++
		}
	}
	if rows != 1 {
		t.Fatalf("expected exactly 1 dispatch into build (stripped concurrency=1), got %d:\n%s", rows, nOut)
	}
}
