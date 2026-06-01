// ABOUTME: AC-4 stage numeric/bool inline-comment strip — concurrency/worktree
// ABOUTME: with a trailing `# comment` parse to the value, not a silent default fallback.
package status

import "testing"

// TestParseStagesStripsInlineComment locks the #163 stage-field silent-misparse
// fix: a stage `concurrency: 5  # debate` must yield 5 (not the default 2 via the
// atoiOr fallback) and `worktree: true  # iso` must yield true (not false). These
// typed fields never legitimately contain `#`, so the inline-comment strip is
// unambiguous. The defaults block is exercised too so the strip covers both the
// per-stage and defaults paths.
func TestParseStagesStripsInlineComment(t *testing.T) {
	readme := `---
stages:
  defaults:
    worktree: false  # default off
    concurrency: 3  # default
  states:
    - name: backlog
      initial: true  # the seed stage
    - name: build
      concurrency: 5  # debate
      worktree: true  # iso
    - name: done
      terminal: true  # last
---
`
	path := writeTemp(t, readme)
	stages, defaults := ParseStagesWithDefaults(path)
	if len(stages) != 3 {
		t.Fatalf("got %d stages, want 3", len(stages))
	}

	if defaults["concurrency"] != "3" {
		t.Fatalf("defaults concurrency = %q, want \"3\" (inline comment stripped)", defaults["concurrency"])
	}
	if defaults["worktree"] != "false" {
		t.Fatalf("defaults worktree = %q, want \"false\" (inline comment stripped)", defaults["worktree"])
	}

	backlog := stages[0]
	if !backlog.initial {
		t.Fatalf("backlog.initial should be true (inline comment stripped), got %+v", backlog)
	}
	if backlog.concurrency != 3 {
		t.Fatalf("backlog concurrency = %d, want 3 (inherits stripped default, not the built-in 2)", backlog.concurrency)
	}

	build := stages[1]
	if build.concurrency != 5 {
		t.Fatalf("build concurrency = %d, want 5 (inline comment stripped, not default fallback)", build.concurrency)
	}
	if !build.Worktree {
		t.Fatalf("build worktree should be true (inline comment stripped), got %+v", build)
	}

	done := stages[2]
	if !done.terminal {
		t.Fatalf("done.terminal should be true (inline comment stripped), got %+v", done)
	}
}
