// ABOUTME: AC-2 stage parser tests — defaults, gates, terminal, worktree,
// ABOUTME: feedback-to, and the no-stages -> nil case match parse_stages_block.
package status

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseStagesBlock(t *testing.T) {
	readme := `---
id-style: sequential
stages:
  defaults:
    worktree: false
    concurrency: 1
  states:
    - name: backlog
      initial: true
      gate: true
    - name: ideation
      gate: true
      feedback-to: backlog
    - name: implementation
      worktree: true
    - name: done
      terminal: true
---

# Workflow
`
	path := writeTemp(t, readme)
	stages, defaults := ParseStagesWithDefaults(path)
	if len(stages) != 4 {
		t.Fatalf("got %d stages, want 4", len(stages))
	}

	if defaults["worktree"] != "false" || defaults["concurrency"] != "1" {
		t.Fatalf("defaults = %v", defaults)
	}

	backlog := stages[0]
	if backlog.Name != "backlog" || !backlog.initial || !backlog.gate {
		t.Fatalf("backlog = %+v", backlog)
	}
	if backlog.concurrency != 1 {
		t.Fatalf("backlog concurrency = %d, want 1 (from defaults)", backlog.concurrency)
	}
	if backlog.Worktree {
		t.Fatalf("backlog worktree should default false")
	}

	ideation := stages[1]
	if !ideation.gate || ideation.optional["feedback-to"] != "backlog" {
		t.Fatalf("ideation = %+v", ideation)
	}

	impl := stages[2]
	if !impl.Worktree {
		t.Fatalf("implementation worktree should be true")
	}

	done := stages[3]
	if !done.terminal {
		t.Fatalf("done should be terminal")
	}
}

func TestParseStagesBlockDefaultConcurrencyAndWorktree(t *testing.T) {
	// No defaults block: concurrency defaults to 2, worktree to false.
	readme := `---
stages:
  states:
    - name: alpha
    - name: beta
      worktree: true
---
`
	path := writeTemp(t, readme)
	stages := parseStagesBlock(path)
	if len(stages) != 2 {
		t.Fatalf("got %d stages, want 2", len(stages))
	}
	if stages[0].concurrency != 2 {
		t.Fatalf("alpha concurrency = %d, want 2 (built-in default)", stages[0].concurrency)
	}
	if stages[0].Worktree {
		t.Fatalf("alpha worktree should default false")
	}
	if !stages[1].Worktree {
		t.Fatalf("beta worktree should be true")
	}
}

func TestParseStagesBlockNoStagesReturnsNil(t *testing.T) {
	readme := `---
id-style: sequential
title: no stages here
---

# Body
`
	path := writeTemp(t, readme)
	if stages := parseStagesBlock(path); stages != nil {
		t.Fatalf("expected nil for missing stages block, got %+v", stages)
	}
}

func TestParseStagesBlockEmptyStatesReturnsNil(t *testing.T) {
	readme := `---
stages:
  defaults:
    concurrency: 3
---
`
	path := writeTemp(t, readme)
	if stages := parseStagesBlock(path); stages != nil {
		t.Fatalf("expected nil for empty states, got %+v", stages)
	}
}
