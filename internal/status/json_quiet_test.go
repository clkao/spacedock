// ABOUTME: AC-3 --quiet mutation behavior — single machine line on success, exit
// ABOUTME: codes and stderr diagnostics unchanged; AC-3(d) --quiet --json read no-op.
package status

import (
	"path/filepath"
	"testing"
)

// TestQuietSetSuccessLine (AC-3 oracle a) locks the --set --quiet success line:
// one machine line `set slug=<slug> <field=old->new>...` and exit 0.
func TestQuietSetSuccessLine(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "seq-workflow")

	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--set", "002-vendor-script", "status=implementation", "--quiet")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, errOut)
	}
	want := "set slug=002-vendor-script status=ideation->implementation\n"
	if out != want {
		t.Fatalf("--set --quiet stdout = %q, want %q", out, want)
	}
}

// TestQuietSetGuardUnchanged (AC-3 oracle b) locks the invariant heart of AC-3:
// a guard-tripping --set with --quiet produces stderr + exit byte-identical to
// the same call WITHOUT --quiet. --quiet trims success chrome only; it never
// touches exit codes or error diagnostics.
func TestQuietSetGuardUnchanged(t *testing.T) {
	env := pinnedEnv(t)
	plainRoot := stageFixture(t, "guard-workflow")
	quietRoot := stageFixture(t, "guard-workflow")

	pOut, pErr, pCode := runNative(t, plainRoot, env, "--workflow-dir", plainRoot, "--set", "010-blocked", "status=done")
	qOut, qErr, qCode := runNative(t, quietRoot, env, "--workflow-dir", quietRoot, "--set", "010-blocked", "status=done", "--quiet")

	if pCode != 1 || qCode != 1 {
		t.Fatalf("guard exit codes: plain=%d quiet=%d, want both 1", pCode, qCode)
	}
	if normalize(qErr, quietRoot) != normalize(pErr, plainRoot) {
		t.Fatalf("--quiet altered guard stderr\nplain: %q\nquiet: %q", pErr, qErr)
	}
	if pOut != "" || qOut != "" {
		t.Fatalf("guard rejection emits no stdout: plain=%q quiet=%q", pOut, qOut)
	}
}

// TestQuietArchiveLine (AC-3 oracle c) locks --archive --quiet: one machine line
// `archived slug=<slug>` and exit 0.
func TestQuietArchiveLine(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "seq-workflow")

	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--archive", "001-design-seam", "--quiet")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, errOut)
	}
	want := "archived slug=001-design-seam\n"
	if out != want {
		t.Fatalf("--archive --quiet stdout = %q, want %q", out, want)
	}
}

// TestQuietJSONReadNoOp (AC-3 oracle d) locks that --quiet --json on a read is a
// no-op: the bytes equal a plain --json read (JSON has no decorative chrome to
// trim).
func TestQuietJSONReadNoOp(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	for _, extra := range [][]string{nil, {"--next"}} {
		plainArgs := append([]string{"--workflow-dir", root, "--json"}, extra...)
		quietArgs := append([]string{"--workflow-dir", root, "--json", "--quiet"}, extra...)

		plain, _, pc := runNative(t, root, env, plainArgs...)
		quiet, _, qc := runNative(t, root, env, quietArgs...)
		if pc != 0 || qc != 0 {
			t.Fatalf("exit: plain=%d quiet=%d for extra=%v", pc, qc, extra)
		}
		if plain != quiet {
			t.Fatalf("--quiet --json read not a no-op for extra=%v\nplain: %q\nquiet: %q", extra, plain, quiet)
		}
	}
}

// TestSetJSONEnvelope locks the --set --json mutation envelope: a single object
// {"command":"set","slug":...,"changes":[{field,old,new}]} on success, exit 0.
func TestSetJSONEnvelope(t *testing.T) {
	env := pinnedEnv(t)
	root := stageFixture(t, "seq-workflow")

	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--set", "002-vendor-script", "status=design", "--json")
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, errOut)
	}
	want := `{"command":"set","slug":"002-vendor-script","changes":[{"field":"status","old":"ideation","new":"design"}]}` + "\n"
	if out != want {
		t.Fatalf("--set --json stdout = %q, want %q", out, want)
	}
}
