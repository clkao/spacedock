// ABOUTME: Offline unit test for the live cycle's entity-locate + path-scoped
// ABOUTME: commit-scan helpers, staged against a fixture repo (no live model).
package ensigncycle

import (
	"path/filepath"
	"testing"
)

// These run under the DEFAULT build tags (no //go:build live) so `go test ./...`
// covers them; they spend NO model — only the locate/scan logic is exercised
// against a staged git fixture. They pin the two behaviors the live assertions
// rely on: (1) the entity is found whether it stayed in place OR was archived
// (flat or folder), and (2) the commit scan accepts a path-scoped commit anywhere
// in the log but REJECTS a sibling-sweep-only history (the haiku incomplete
// cycle's `[README.md make-it-work.md]` shape).

// TestLocateEntity covers the four end-state locations the live test searches.
func TestLocateEntity(t *testing.T) {
	t.Run("in_place_original_path", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "make-it-work.md"), "in place")

		content, where, found := locateEntity(root, "make-it-work")
		if !found || content != "in place" {
			t.Fatalf("found=%v content=%q, want the in-place body", found, content)
		}
		if filepath.Base(where) != "make-it-work.md" {
			t.Errorf("where = %q, want the original flat path", where)
		}
	})

	t.Run("archived_flat", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "_archive", "make-it-work.md"), "archived flat")

		content, where, found := locateEntity(root, "make-it-work")
		if !found || content != "archived flat" {
			t.Fatalf("found=%v content=%q, want the archived-flat body", found, content)
		}
		if filepath.Dir(where) != filepath.Join(root, "_archive") {
			t.Errorf("where = %q, want it under _archive/", where)
		}
	})

	t.Run("archived_folder_index", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "_archive", "make-it-work", "index.md"), "archived folder")

		content, _, found := locateEntity(root, "make-it-work")
		if !found || content != "archived folder" {
			t.Fatalf("found=%v content=%q, want the archived-folder body", found, content)
		}
	})

	t.Run("missing_everywhere_is_not_found", func(t *testing.T) {
		root := t.TempDir()
		if _, _, found := locateEntity(root, "make-it-work"); found {
			t.Fatal("found must be false when the entity exists nowhere")
		}
	})
}

// TestSomeCommitNamesOnly covers the path-scoped commit scan over a real git log.
func TestSomeCommitNamesOnly(t *testing.T) {
	// A path-scoped commit somewhere in the log is accepted even when HEAD is a
	// later multi-file (archive/finalize) commit — the full-cycle shape.
	t.Run("path_scoped_commit_present_below_head", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "README.md"), "readme")
		writeFile(t, filepath.Join(root, "make-it-work.md"), "entity")
		gitInit(t, root) // initial commit touches both files

		// The ensign's path-scoped state commit (names ONLY the entity).
		appendFile(t, filepath.Join(root, "make-it-work.md"), "\nstage report\n")
		gitCommitPathScoped(t, root, "make-it-work.md", "stage: backlog report")

		// A later FO archive/finalize commit that sweeps multiple files — this is
		// HEAD now, so a HEAD-only check would miss the path-scoped commit above.
		writeFile(t, filepath.Join(root, "board.md"), "board")
		git(t, root, "add", "-A")
		git(t, root, "commit", "-q", "-m", "finalize: archive + board")

		if !someCommitNamesOnly(t, root, "make-it-work") {
			t.Fatal("expected a path-scoped entity commit to be found below HEAD")
		}
	})

	// The haiku incomplete-cycle shape: the only entity-touching commit swept a
	// sibling (`[README.md make-it-work.md]`), so NO commit is path-scoped.
	t.Run("sibling_sweep_only_is_rejected", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "README.md"), "readme")
		writeFile(t, filepath.Join(root, "make-it-work.md"), "entity")
		git(t, root, "init", "-q")

		// One commit that adds README.md AND make-it-work.md together (sibling
		// sweep) — the incomplete cycle's non-path-scoped commit.
		git(t, root, "add", "-A")
		git(t, root, "commit", "-q", "-m", "wip: README.md make-it-work.md")

		if someCommitNamesOnly(t, root, "make-it-work") {
			t.Fatal("a sibling-sweep-only history must NOT count as path-scoped")
		}
	})
}

// TestLiveStageReportHeading pins the stage-agnostic heading anchor the live test
// uses. The real full FO cycle finishes at the TERMINAL stage, so the ensign that
// completes it writes `## Stage Report: done` — the backlog-specific skeleton regex
// would never match. This proves the live regex matches ANY named stage heading and
// still goes RED when there is no `## Stage Report:` section at all (the
// incomplete-cycle shape).
func TestLiveStageReportHeading(t *testing.T) {
	if !liveStageReportHeading.MatchString("## Stage Report: done\n- DONE: x") {
		t.Error("must match the terminal `## Stage Report: done` heading")
	}
	if !liveStageReportHeading.MatchString("## Stage Report: backlog\n- DONE: x") {
		t.Error("must still match the backlog heading — any named stage counts")
	}
	if !liveStageReportHeading.MatchString("body\n\n## Stage Report: implementation\n") {
		t.Error("must match a stage-report heading appended mid-body")
	}
	if liveStageReportHeading.MatchString("---\nstatus: done\n---\nbody with no stage report") {
		t.Error("must NOT match an entity with no `## Stage Report:` section")
	}
	if liveStageReportHeading.MatchString("## Stage Report: \n") {
		t.Error("must NOT match a heading with no named stage after the colon")
	}
}

// TestTerminalFrontmatterAnchors pins the go-red discipline of the terminal
// frontmatter assertions: they match a finalized entity whose verdict is SET to
// ANY non-empty value (the exact word is FO judgment that varies by model —
// `passed`, `PASSED`, `done` all mean the cycle finished) and REJECT an entity
// that never reached the terminal stage (empty/unset verdict). This is the
// offline proof that the live assertions go red on an incomplete cycle, no model.
func TestTerminalFrontmatterAnchors(t *testing.T) {
	completedPassed := "---\nstatus: done\nverdict: PASSED\n---\nbody"
	completedLower := "---\nstatus: done\nverdict: passed\n---\nbody"
	completedDone := "---\nstatus: done\nverdict: done\n---\nbody"
	emptyVerdict := "---\nstatus: done\nverdict:\n---\nbody"
	incomplete := "---\nstatus: backlog\nverdict:\n---\nbody"

	if !frontmatterField.MatchString(completedPassed) {
		t.Error("status: done must match a finalized entity")
	}
	if !verdictSet.MatchString(completedPassed) {
		t.Error("verdict: PASSED must match — any non-empty value counts")
	}
	if !verdictSet.MatchString(completedLower) {
		t.Error("verdict: passed must match")
	}
	if !verdictSet.MatchString(completedDone) {
		t.Error("verdict: done must match — the verdict word is FO judgment, not pinned")
	}
	if frontmatterField.MatchString(incomplete) {
		t.Error("status: done must NOT match a backlog (incomplete) entity")
	}
	if verdictSet.MatchString(emptyVerdict) {
		t.Error("verdict must NOT match an empty value (the incomplete-cycle shape)")
	}
}
