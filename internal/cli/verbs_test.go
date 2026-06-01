// ABOUTME: AC-3 verb tests — `new` aliases status --new through discovery, and
// ABOUTME: `completion <shell>` emits a script (exit 0) or names the usage error.
package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/status"
)

const newEntityBody = "---\ntitle: Minted via new\nstatus: ideation\nscore: \"0.30\"\nsource: roadmap\n---\n# Minted via new\n"

// TestNewVerbMintsInDiscoveredWorkflow (AC-3) runs `spacedock new <slug>` with a
// body on stdin from a cwd inside a commissioned workflow (NO --workflow-dir),
// and asserts the alias reaches runNew: minted-id narration on stdout, the
// entity file exists, and --validate is clean.
func TestNewVerbMintsInDiscoveredWorkflow(t *testing.T) {
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), "---\ncommissioned-by: spacedock@1\nid-style: slug\nstages:\n  states:\n    - name: ideation\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n")

	env := []string{"USER=pinned", "PATH=" + os.Getenv("PATH")}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"new", "minted-task"}, env, def, strings.NewReader(newEntityBody), &stdout, &stderr, &status.NativeRunner{}, nil)
	if code != 0 {
		t.Fatalf("new exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "id=") {
		t.Fatalf("new stdout = %q, want minted-id narration", stdout.String())
	}
	if !isFile(filepath.Join(def, "minted-task.md")) {
		t.Fatalf("new did not create the entity file")
	}

	// --validate clean immediately after, proving a complete entity was minted.
	var vout, verr bytes.Buffer
	vcode := run(context.Background(), []string{"status", "--workflow-dir", def, "--validate"}, env, def, nil, &vout, &verr, &status.NativeRunner{}, nil)
	if vcode != 0 || strings.TrimSpace(vout.String()) != "VALID" {
		t.Fatalf("post-new validate exit=%d out=%q err=%q", vcode, vout.String(), verr.String())
	}
}

// TestNewVerbFolderForm proves `new --folder <slug>` threads the --folder flag
// to the runner's folder-form create path.
func TestNewVerbFolderForm(t *testing.T) {
	def := t.TempDir()
	writeFile(t, filepath.Join(def, "README.md"), "---\ncommissioned-by: spacedock@1\nid-style: slug\nstages:\n  states:\n    - name: ideation\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n")

	env := []string{"USER=pinned", "PATH=" + os.Getenv("PATH")}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"new", "--folder", "foldered"}, env, def, strings.NewReader(newEntityBody), &stdout, &stderr, &status.NativeRunner{}, nil)
	if code != 0 {
		t.Fatalf("new --folder exit=%d stderr=%q", code, stderr.String())
	}
	if !isFile(filepath.Join(def, "foldered", "index.md")) {
		t.Fatalf("new --folder did not create folder-form entity")
	}
}

// TestCompletionShells (AC-3) covers the completion verb's exit-code contract.
func TestCompletionShells(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"completion", shell}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("completion %s exit=%d stderr=%q", shell, code, stderr.String())
		}
		for _, verb := range []string{"status", "new", "completion"} {
			if !strings.Contains(stdout.String(), verb) {
				t.Fatalf("completion %s script missing verb %q:\n%s", shell, verb, stdout.String())
			}
		}
	}

	// Missing shell and unknown shell both exit 2 with the named usage error.
	for _, args := range [][]string{{"completion"}, {"completion", "fish"}} {
		var stdout, stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("%v exit=%d, want 2 (stderr=%q)", args, code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "completion requires a shell: bash or zsh") {
			t.Fatalf("%v stderr = %q, want named usage error", args, stderr.String())
		}
	}
}

// TestHelpListsNewVerbs (AC-3) asserts the grouped --help lists both new verbs
// under the Workflow group with their one-liners, in the cobra help structure.
func TestHelpListsNewVerbs(t *testing.T) {
	var stdout bytes.Buffer
	Run([]string{"--help"}, &stdout, &bytes.Buffer{})
	out := stdout.String()
	for _, want := range []string{
		"new",
		"Create an entity from a stdin body",
		"completion",
		"Print a bash or zsh completion script",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("--help missing %q:\n%s", want, out)
		}
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
