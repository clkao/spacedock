// ABOUTME: AC-5 launcher smoke — a pilot split-root (symlink-profile) workflow
// ABOUTME: lists, --sets, and --archives a folder-form entity through `spacedock status`.
package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	launcherOnce sync.Once
	launcherBin  string
	launcherErr  error
)

// spacedockBinary builds the real spacedock launcher once and returns its path,
// so the smoke test exercises the actual command surface (not the in-process
// runner seam). The build output goes in a per-process temp dir.
func spacedockBinary(t *testing.T) string {
	t.Helper()
	launcherOnce.Do(func() {
		dir, err := os.MkdirTemp("", "spacedock-bin-*")
		if err != nil {
			launcherErr = err
			return
		}
		bin := filepath.Join(dir, "spacedock")
		cmd := exec.Command("go", "build", "-o", bin, "github.com/clkao/spacedock-v1/cmd/spacedock")
		if out, err := cmd.CombinedOutput(); err != nil {
			launcherErr = err
			t.Logf("go build spacedock failed:\n%s", out)
			return
		}
		launcherBin = bin
	})
	if launcherErr != nil {
		t.Fatalf("build spacedock launcher: %v", launcherErr)
	}
	return launcherBin
}

// pilotReadme is a folder-form-entity workflow README with the symlink-profile
// state field. The single non-initial worktree-free stage keeps --set simple.
const pilotReadme = `---
entity-type: task
entity-label: task
entity-label-plural: tasks
id-style: slug
state: .spacedock-state
stages:
  defaults:
    worktree: false
    concurrency: 1
  states:
    - name: backlog
      initial: true
    - name: ideation
    - name: done
      terminal: true
---

# Pilot Split-Root Workflow

### backlog

Start.

- **Outputs:** seed.

### ideation

Think.

- **Outputs:** approach.

### done

Terminal.
`

// stagePilotWorkflow builds a native split-root workflow in a fresh git repo:
// README in the main repo carrying state: .spacedock-state, and a folder-form
// entity in the state checkout. The launcher is pointed at the definition dir
// (the native operator model the production NativeRunner enables); no
// .spacedock-state/README.md symlink is required. Returns the definition dir,
// the state-checkout dir, and the entity slug.
func stagePilotWorkflow(t *testing.T) (defDir, stateDir, slug string) {
	t.Helper()
	defDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(defDir, "README.md"), []byte(pilotReadme), 0o644); err != nil {
		t.Fatal(err)
	}
	stateDir = filepath.Join(defDir, ".spacedock-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	slug = "pilot-entity"
	entityPath := filepath.Join(stateDir, slug, "index.md")
	if err := os.MkdirAll(filepath.Dir(entityPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `---
id: "001"
title: Pilot entity
status: backlog
score: "0.50"
source: smoke
---
# Pilot entity

A folder-form entity driven through the launcher.
`
	if err := os.WriteFile(entityPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, defDir)
	return defDir, stateDir, slug
}

// runStatus runs `spacedock status --workflow-dir {defDir} {args...}` and
// returns combined output and exit code. defDir is the definition dir (holds
// README.md); the native runner resolves entities under its state: checkout.
func runStatus(t *testing.T, defDir string, args ...string) (string, int) {
	t.Helper()
	full := append([]string{"status", "--workflow-dir", defDir}, args...)
	cmd := exec.Command(spacedockBinary(t), full...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("run spacedock status %v: %v\n%s", args, err, out)
		}
	}
	return string(out), code
}

// TestLauncherListSetArchive locks AC-5: list renders the entity row, --set
// narrates field: old -> new, and --archive moves the entity under _archive in
// the state checkout — all through the real launcher binary.
func TestLauncherListSetArchive(t *testing.T) {
	defDir, stateDir, slug := stagePilotWorkflow(t)

	// List: the entity row must render.
	list, code := runStatus(t, defDir)
	if code != 0 {
		t.Fatalf("list exit %d:\n%s", code, list)
	}
	if !strings.Contains(list, slug) || !strings.Contains(list, "Pilot entity") {
		t.Fatalf("list output missing entity row for %q:\n%s", slug, list)
	}

	// --set: status backlog -> ideation, narrated on stdout.
	setOut, code := runStatus(t, defDir, "--set", slug, "status=ideation")
	if code != 0 {
		t.Fatalf("--set exit %d:\n%s", code, setOut)
	}
	if !strings.Contains(setOut, "status: backlog -> ideation") {
		t.Fatalf("--set narration missing 'status: backlog -> ideation':\n%s", setOut)
	}

	// --archive: the entity moves under _archive in the state checkout.
	archiveOut, code := runStatus(t, defDir, "--archive", slug)
	if code != 0 {
		t.Fatalf("--archive exit %d:\n%s", code, archiveOut)
	}
	archivedEntity := filepath.Join(stateDir, "_archive", slug, "index.md")
	if _, err := os.Stat(archivedEntity); err != nil {
		t.Fatalf("entity not archived into state checkout at %s: %v\n%s", archivedEntity, err, archiveOut)
	}
	// And it must no longer be in the active location.
	activeEntity := filepath.Join(stateDir, slug, "index.md")
	if _, err := os.Stat(activeEntity); err == nil {
		t.Fatalf("entity still present at active location %s after archive", activeEntity)
	}
}
