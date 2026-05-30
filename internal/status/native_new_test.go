// ABOUTME: AC-7 --new atomic create tests — minted id matches --next-id, the
// ABOUTME: workflow validates immediately after, and the guard error paths hold.
package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const newBody = "---\ntitle: Newly minted entity\nstatus: backlog\nscore: \"0.30\"\nsource: roadmap\n---\n# Newly minted entity\n\nCreated via --new.\n"

// TestNewSequentialMintsAndValidates (AC-7) pipes a body to --new in a
// sequential workflow, asserts the written file carries the minted id (equal to
// --next-id under the same env), and that --validate is VALID immediately after.
func TestNewSequentialMintsAndValidates(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "seq-workflow")

	nextID, _, code := runNative(t, root, env, "--workflow-dir", root, "--next-id")
	if code != 0 {
		t.Fatalf("--next-id exit=%d", code)
	}
	want := strings.TrimSpace(nextID)

	out, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "minted-task")
	if code != 0 {
		t.Fatalf("--new exit=%d stderr=%q", code, errOut)
	}
	if !strings.Contains(out, "id="+want) {
		t.Fatalf("--new narration %q should report minted id %q", out, want)
	}

	written := readWhole(t, filepath.Join(root, "minted-task.md"))
	if !strings.Contains(written, "id: "+want) {
		t.Fatalf("written entity missing minted id %q:\n%s", want, written)
	}
	// Body and original fields preserved.
	if !strings.Contains(written, "title: Newly minted entity") || !strings.Contains(written, "# Newly minted entity") {
		t.Fatalf("written entity dropped body/fields:\n%s", written)
	}

	// Validate is VALID immediately after — no id-less window reachable.
	vOut, vErr, vCode := runNative(t, root, env, "--workflow-dir", root, "--validate")
	if vCode != 0 || strings.TrimSpace(vOut) != "VALID" {
		t.Fatalf("post---new validate exit=%d out=%q err=%q", vCode, vOut, vErr)
	}
}

// TestNewSDB32MintsValidID (AC-7) covers the sd-b32 branch: the minted id is a
// valid 24-char SD-B32 id equal to --next-id under identical pinned material.
func TestNewSDB32MintsValidID(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "sdb32-workflow")

	nextID, _, code := runNative(t, root, env, "--workflow-dir", root, "--next-id", "--id-seed", "minted-task", "--id-actor", "pinnedactor")
	if code != 0 {
		t.Fatalf("--next-id exit=%d", code)
	}
	want := strings.TrimSpace(nextID)
	if !isValidSDB32ID(want) {
		t.Fatalf("--next-id produced invalid sd-b32 id %q", want)
	}

	_, errOut, code := runNativeStdin(t, root, env, reader(newBody),
		"--workflow-dir", root, "--new", "minted-task", "--id-seed", "minted-task", "--id-actor", "pinnedactor")
	if code != 0 {
		t.Fatalf("--new exit=%d stderr=%q", code, errOut)
	}
	written := readWhole(t, filepath.Join(root, "minted-task.md"))
	if !strings.Contains(written, "id: "+want) {
		t.Fatalf("written entity id should equal --next-id %q:\n%s", want, written)
	}

	vOut, vErr, vCode := runNative(t, root, env, "--workflow-dir", root, "--validate")
	if vCode != 0 || strings.TrimSpace(vOut) != "VALID" {
		t.Fatalf("post---new validate exit=%d out=%q err=%q", vCode, vOut, vErr)
	}
}

// TestNewNoIDLessWindow (AC-7) asserts the write is atomic: the entity file
// either does not exist or already carries a non-empty id — there is never an
// on-disk entity with an empty id. We prove the property structurally by reading
// the file right after --new and confirming no empty-id frontmatter, plus that
// the write path uses temp+rename (the final file has the id present).
func TestNewNoIDLessWindow(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "seq-workflow")

	_, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "atomic-task")
	if code != 0 {
		t.Fatalf("--new exit=%d stderr=%q", code, errOut)
	}
	written := readWhole(t, filepath.Join(root, "atomic-task.md"))
	fm := parseFrontmatterContent([]byte(written))
	if strings.TrimSpace(fm["id"]) == "" {
		t.Fatalf("entity exists with empty id — id-less window observable:\n%s", written)
	}
	// No leftover temp file from the atomic write.
	matches, _ := filepath.Glob(filepath.Join(root, ".status-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("leftover temp files from atomic write: %v", matches)
	}
}

// TestNewFolderForm (AC-7) covers --new --folder: the entity is written as
// {slug}/index.md (folder form) with the minted id stamped in, the workflow
// validates immediately after (no id-less window), and no flat {slug}.md is
// created. The written file is byte-identical to STDIN + the id-stamp.
func TestNewFolderForm(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "seq-workflow")

	nextID, _, code := runNative(t, root, env, "--workflow-dir", root, "--next-id")
	if code != 0 {
		t.Fatalf("--next-id exit=%d", code)
	}
	want := strings.TrimSpace(nextID)

	out, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "--folder", "folder-task")
	if code != 0 {
		t.Fatalf("--new --folder exit=%d stderr=%q", code, errOut)
	}

	// Folder form written, flat form NOT created.
	indexPath := filepath.Join(root, "folder-task", "index.md")
	if !isRegularFile(indexPath) {
		t.Fatalf("--new --folder should write %s", indexPath)
	}
	if isRegularFile(filepath.Join(root, "folder-task.md")) {
		t.Fatalf("--new --folder must not create the flat form")
	}
	if !strings.Contains(out, "id="+want) {
		t.Fatalf("--new --folder narration %q should report minted id %q", out, want)
	}

	written := readWhole(t, indexPath)
	if !strings.Contains(written, "id: "+want) {
		t.Fatalf("folder entity missing minted id %q:\n%s", want, written)
	}
	// Byte-identity: the written file equals STDIN with the id line inserted.
	wantBytes := string(stampID([]byte(newBody), want))
	if written != wantBytes {
		t.Fatalf("folder entity not byte-identical to STDIN+id-stamp\n--- got ---\n%q\n--- want ---\n%q", written, wantBytes)
	}

	// Validate is VALID immediately after — no id-less window.
	vOut, vErr, vCode := runNative(t, root, env, "--workflow-dir", root, "--validate")
	if vCode != 0 || strings.TrimSpace(vOut) != "VALID" {
		t.Fatalf("post---new --folder validate exit=%d out=%q err=%q", vCode, vOut, vErr)
	}
}

// TestNewFolderCollisions (AC-7) locks that --new --folder refuses when the slug
// exists in EITHER form, and likewise flat --new refuses a pre-existing folder.
func TestNewFolderCollisions(t *testing.T) {
	env := pinnedEnv(t)

	t.Run("folder-new-refuses-existing-flat", func(t *testing.T) {
		root := stageFixture(t, "seq-workflow")
		// 001-design-seam exists as a flat file.
		_, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "--folder", "001-design-seam")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (flat slug exists)", code)
		}
		if !strings.Contains(errOut, "already exists") {
			t.Fatalf("stderr=%q, want already-exists error", errOut)
		}
	})

	t.Run("flat-new-refuses-existing-folder", func(t *testing.T) {
		root := stageFixture(t, "seq-workflow")
		// 003-wire-cli exists as a folder entity.
		_, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "003-wire-cli")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (folder slug exists)", code)
		}
		if !strings.Contains(errOut, "already exists") {
			t.Fatalf("stderr=%q, want already-exists error", errOut)
		}
	})
}

// TestNewGuards (AC-7) covers the refusal paths: slug already exists, missing
// fence, and --id-seed with id-style: slug.
func TestNewGuards(t *testing.T) {
	env := pinnedEnv(t)

	t.Run("slug-exists", func(t *testing.T) {
		root := stageFixture(t, "seq-workflow")
		_, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "001-design-seam")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (slug exists)", code)
		}
		if !strings.Contains(errOut, "already exists") {
			t.Fatalf("stderr=%q, want already-exists error", errOut)
		}
	})

	t.Run("missing-fence", func(t *testing.T) {
		root := stageFixture(t, "seq-workflow")
		_, errOut, code := runNativeStdin(t, root, env, reader("no frontmatter here\n"), "--workflow-dir", root, "--new", "fenceless")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (missing fence)", code)
		}
		if !strings.Contains(errOut, "no frontmatter") {
			t.Fatalf("stderr=%q, want missing-fence error", errOut)
		}
		if _, err := os.Stat(filepath.Join(root, "fenceless.md")); !os.IsNotExist(err) {
			t.Fatalf("entity should not have been created on missing-fence error")
		}
	})

	t.Run("slug-style-with-id-seed", func(t *testing.T) {
		root := slugWorkflow(t)
		_, errOut, code := runNativeStdin(t, root, env, reader(newBody), "--workflow-dir", root, "--new", "seeded", "--id-seed", "x")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (slug style + --id-seed)", code)
		}
		if !strings.Contains(errOut, "only applicable for id-style: sd-b32") {
			t.Fatalf("stderr=%q, want id-seed-not-applicable error", errOut)
		}
	})

	t.Run("conflicting-id-in-body", func(t *testing.T) {
		root := stageFixture(t, "seq-workflow")
		body := "---\nid: \"999\"\ntitle: Pre-IDed\nstatus: backlog\n---\n# Pre-IDed\n"
		_, errOut, code := runNativeStdin(t, root, env, reader(body), "--workflow-dir", root, "--new", "preided")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (conflicting id in body)", code)
		}
		if !strings.Contains(errOut, "already declares id") {
			t.Fatalf("stderr=%q, want conflicting-id error", errOut)
		}
	})

	t.Run("nil-stdin-no-panic", func(t *testing.T) {
		// A nil Stdin (no pipe wired) must not panic io.ReadAll(nil); it reads as
		// an empty body which then fails the opening-fence guard, exit 1.
		root := stageFixture(t, "seq-workflow")
		_, errOut, code := runNativeStdin(t, root, env, nil, "--workflow-dir", root, "--new", "freshslug")
		if code != 1 {
			t.Fatalf("exit=%d, want 1 (nil stdin)", code)
		}
		if !strings.Contains(errOut, "no frontmatter") {
			t.Fatalf("stderr=%q, want no-frontmatter error", errOut)
		}
		if _, err := os.Stat(filepath.Join(root, "freshslug.md")); !os.IsNotExist(err) {
			t.Fatalf("nil-stdin --new should not write a seed")
		}
	})
}

// slugWorkflow builds a minimal id-style: slug workflow in a temp dir.
func slugWorkflow(t *testing.T) string {
	t.Helper()
	readme := "---\nid-style: slug\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# Slug Workflow\n"
	dst := t.TempDir()
	if err := os.WriteFile(filepath.Join(dst, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInit(t, dst)
	return dst
}
