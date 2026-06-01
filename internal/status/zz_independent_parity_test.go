// ABOUTME: Independent validation-stage parity harness — builds fresh fixtures
// ABOUTME: and diffs NativeRunner directly against the live Python oracle.
package status

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/claudeteam"
)

// This file is an INDEPENDENT validator authored at the validation stage. It
// does not reuse the native_*_test.go assertions or the in-tree testdata; it
// builds its own fixtures, drives the production NativeRunner.Run, shells out to
// the live oracle, and diffs the four observable channels after the documented
// normalization. Failure here means a real parity gap, not a stale golden.

const indOracle = "/Users/clkao/git/spacedock/skills/commission/bin/status"

func indOraclePath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("SPACEDOCK_ORACLE"); p != "" {
		return p
	}
	if _, err := os.Stat(indOracle); err == nil {
		return indOracle
	}
	// Oracle-dependent subtests skip when the oracle is absent (matching the
	// in-tree harness). The --new / guard subtests do not touch the oracle and
	// run unconditionally.
	t.Skipf("oracle not found at %s (set SPACEDOCK_ORACLE)", indOracle)
	return ""
}

func indEnv(t *testing.T) []string {
	t.Helper()
	return []string{
		"PYTHONUTF8=1",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"USER=ind-actor",
		"SPACEDOCK_TEST_SD_B32_TIMESTAMP=2026-03-03T03:03:03.030303Z",
		"SPACEDOCK_ID_CONTEXT=ind-ctx",
		"HOME=" + t.TempDir(),
		"PATH=" + os.Getenv("PATH"),
	}
}

func indRunNative(t *testing.T, dir string, env []string, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	var in io.Reader // genuine nil interface when no stdin
	if stdin != "" {
		in = strings.NewReader(stdin)
	}
	// Production wires the Claude team-state probe (the native binary is the Claude
	// runtime's companion); reproduce it so boot TEAM_STATE matches the oracle.
	r := &NativeRunner{TeamStateProbe: claudeteam.Probe}
	code, err := r.Run(context.Background(), Request{
		Args: args, Dir: dir, Env: env,
		Stdin:  in,
		Stdout: &stdout, Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("native error: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

func indRunOracle(t *testing.T, dir string, env []string, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command("python3", append([]string{indOraclePath(t)}, args...)...)
	cmd.Dir = dir
	cmd.Env = env
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("oracle exec error: %v (stderr=%q)", err, stderr.String())
		}
	}
	return stdout.String(), stderr.String(), code
}

var indTSRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z`)

func indNormalize(s, root string) string {
	s = indTSRe.ReplaceAllString(s, "<TS>")
	// Strip the native-only STATE_BACKEND boot banner (the oracle has no such
	// line) — the same documented native/oracle divergence the in-tree harness
	// strips; the backend keys are byte-pinned in json_boot_test.go instead.
	s = stripStateBackend(s)
	if root != "" {
		if real, err := filepath.EvalSymlinks(root); err == nil && real != root {
			s = strings.ReplaceAll(s, real, "<ROOT>")
		}
		s = strings.ReplaceAll(s, root, "<ROOT>")
	}
	return s
}

// indDiff asserts native==oracle for (stdout, stderr, exit) after normalization.
func indDiff(t *testing.T, name, root string, nOut, nErr string, nCode int, oOut, oErr string, oCode int) {
	t.Helper()
	if nCode != oCode {
		t.Errorf("[%s] EXIT mismatch: native=%d oracle=%d\nnative-stderr=%q\noracle-stderr=%q", name, nCode, oCode, nErr, oErr)
	}
	no, oo := indNormalize(nOut, root), indNormalize(oOut, root)
	if no != oo {
		t.Errorf("[%s] STDOUT mismatch:\n--- native ---\n%s\n--- oracle ---\n%s", name, no, oo)
	}
	ne, oe := indNormalize(nErr, root), indNormalize(oErr, root)
	if ne != oe {
		t.Errorf("[%s] STDERR mismatch:\n--- native ---\n%s\n--- oracle ---\n%s", name, ne, oe)
	}
	if nCode != 0 && nCode != 1 {
		t.Errorf("[%s] EXIT %d outside domain {0,1}", name, nCode)
	}
	if oCode != 0 && oCode != 1 {
		t.Errorf("[%s] ORACLE EXIT %d outside domain {0,1}", name, oCode)
	}
}

func indWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// indSeqWorkflow builds a fresh sequential-id workflow with: distinct
// stages/scores, one empty-score entity (locks empty-last), one unknown-status
// entity (locks order 99), one folder-form entity, an archived entity, and an
// unknown-field-carrying entity.
func indSeqWorkflow(t *testing.T) string {
	root := t.TempDir()
	indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: task\nid-style: sequential\nstages:\n  defaults:\n    worktree: false\n    concurrency: 2\n  states:\n    - name: backlog\n      initial: true\n      gate: true\n    - name: ideation\n      gate: true\n    - name: implementation\n      worktree: true\n    - name: done\n      terminal: true\n---\n# Seq WF\n")
	indWrite(t, filepath.Join(root, "001-alpha.md"), "---\nid: \"001\"\ntitle: Alpha task\nstatus: backlog\nscore: \"0.80\"\nsource: roadmap\nissue: ENG-101\ntracker-url: https://x/1\n---\n# Alpha\nbody\n")
	indWrite(t, filepath.Join(root, "002-beta.md"), "---\nid: \"002\"\ntitle: Beta task\nstatus: ideation\nscore: \"0.50\"\nsource: grooming\n---\n# Beta\n")
	// folder form
	indWrite(t, filepath.Join(root, "003-gamma", "index.md"), "---\nid: \"003\"\ntitle: Gamma folder task\nstatus: implementation\nscore: \"0.90\"\nsource: roadmap\n---\n# Gamma\n")
	// empty score -> sorts last within its stage group
	indWrite(t, filepath.Join(root, "004-delta.md"), "---\nid: \"004\"\ntitle: Delta no score\nstatus: ideation\nscore: \"\"\nsource: backlog\n---\n# Delta\n")
	// unknown status -> order 99
	indWrite(t, filepath.Join(root, "005-epsilon.md"), "---\nid: \"005\"\ntitle: Epsilon weird status\nstatus: parked\nscore: \"0.30\"\nsource: roadmap\n---\n# Epsilon\n")
	// archived
	indWrite(t, filepath.Join(root, "_archive", "000-zeta.md"), "---\nid: \"000\"\ntitle: Zeta archived\nstatus: done\nscore: \"0.10\"\nsource: roadmap\narchived: 2026-01-01T00:00:00Z\n---\n# Zeta\n")
	return root
}

func indSDB32Workflow(t *testing.T) string {
	root := t.TempDir()
	indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: task\nid-style: sd-b32\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# SDB32 WF\n")
	indWrite(t, filepath.Join(root, "abcdefghjkmnpqrstvwxyz23-one.md"), "---\nid: abcdefghjkmnpqrstvwxyz23\ntitle: One\nstatus: backlog\nscore: \"0.5\"\nsource: roadmap\n---\n# One\n")
	return root
}

func indSlugWorkflow(t *testing.T) string {
	root := t.TempDir()
	indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: doc\nid-style: slug\nstages:\n  states:\n    - name: draft\n      initial: true\n    - name: published\n      terminal: true\n---\n# Slug WF\n")
	indWrite(t, filepath.Join(root, "intro.md"), "---\ntitle: Intro\nstatus: draft\nscore: \"0.7\"\nsource: roadmap\n---\n# Intro\n")
	indWrite(t, filepath.Join(root, "guide.md"), "---\ntitle: Guide\nstatus: published\nscore: \"0.4\"\nsource: roadmap\n---\n# Guide\n")
	return root
}

// TestIndReadFlagsSeq diffs every read flag against the oracle on a fresh
// sequential workflow (folder entity, empty score, unknown status, archived).
func TestIndReadFlagsSeq(t *testing.T) {
	root := indSeqWorkflow(t)
	env := indEnv(t)
	cases := []struct {
		name string
		args []string
	}{
		{"default", []string{"--workflow-dir", root}},
		{"archived", []string{"--workflow-dir", root, "--archived"}},
		{"next", []string{"--workflow-dir", root, "--next"}},
		{"where-status", []string{"--workflow-dir", root, "--where", "status=ideation"}},
		// `issue` is a non-default frontmatter key, so it appends as a single extra
		// in both runners. A default-named --fields is intentionally NOT a parity
		// case: native de-dupes the duplicate column (captain-approved bug fix)
		// while the oracle renders it twice; that divergence is locked by
		// TestFieldsDedupeNoDuplicateDefaultColumns.
		{"fields", []string{"--workflow-dir", root, "--fields", "issue"}},
		{"all-fields", []string{"--workflow-dir", root, "--all-fields"}},
		{"next-id", []string{"--workflow-dir", root, "--next-id"}},
		{"resolve-id", []string{"--workflow-dir", root, "--resolve", "001"}},
		{"resolve-slug", []string{"--workflow-dir", root, "--resolve", "002-beta"}},
		{"short-id", []string{"--workflow-dir", root, "--short-id", "003"}},
		{"validate", []string{"--workflow-dir", root, "--validate"}},
		{"boot", []string{"--workflow-dir", root, "--boot"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nOut, nErr, nCode := indRunNative(t, root, env, "", c.args...)
			oOut, oErr, oCode := indRunOracle(t, root, env, "", c.args...)
			indDiff(t, c.name, root, nOut, nErr, nCode, oOut, oErr, oCode)
		})
	}
}

// TestIndReadFlagsSDB32 diffs read flags on sd-b32, with --next-id material
// pinned so the candidate is reproducible across native and oracle.
func TestIndReadFlagsSDB32(t *testing.T) {
	root := indSDB32Workflow(t)
	env := indEnv(t)
	cases := []struct {
		name string
		args []string
	}{
		{"default", []string{"--workflow-dir", root}},
		{"next", []string{"--workflow-dir", root, "--next"}},
		{"short-id", []string{"--workflow-dir", root, "--short-id", "abcdefghjkmnpqrstvwxyz23"}},
		{"resolve", []string{"--workflow-dir", root, "--resolve", "abcdefghjkmnpqrstvwxyz23"}},
		{"next-id-seeded", []string{"--workflow-dir", root, "--next-id", "--id-seed", "new-task", "--id-actor", "ind-actor"}},
		{"validate", []string{"--workflow-dir", root, "--validate"}},
		{"boot", []string{"--workflow-dir", root, "--boot"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nOut, nErr, nCode := indRunNative(t, root, env, "", c.args...)
			oOut, oErr, oCode := indRunOracle(t, root, env, "", c.args...)
			indDiff(t, c.name, root, nOut, nErr, nCode, oOut, oErr, oCode)
		})
	}
}

// TestIndReadFlagsSlug diffs read flags on id-style slug, including the
// --next-id-not-applicable error.
func TestIndReadFlagsSlug(t *testing.T) {
	root := indSlugWorkflow(t)
	env := indEnv(t)
	cases := []struct {
		name string
		args []string
	}{
		{"default", []string{"--workflow-dir", root}},
		{"next", []string{"--workflow-dir", root, "--next"}},
		{"short-id", []string{"--workflow-dir", root, "--short-id", "intro"}},
		{"resolve", []string{"--workflow-dir", root, "--resolve", "guide"}},
		{"validate", []string{"--workflow-dir", root, "--validate"}},
		{"next-id-not-applicable", []string{"--workflow-dir", root, "--next-id"}},
		{"boot", []string{"--workflow-dir", root, "--boot"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nOut, nErr, nCode := indRunNative(t, root, env, "", c.args...)
			oOut, oErr, oCode := indRunOracle(t, root, env, "", c.args...)
			indDiff(t, c.name, root, nOut, nErr, nCode, oOut, oErr, oCode)
		})
	}
}

// indCopyTree recursively copies src into a fresh temp dir and returns it, so
// native and oracle each mutate an independent copy of the same fixture.
func indCopyTree(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode())
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// indReadAll returns the relative-path->contents map of every .md file under
// root, so native and oracle filesystem mutations can be diffed.
func indReadAll(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Base(p) == ".git" {
			return nil
		}
		if strings.Contains(p, string(os.PathSeparator)+".git"+string(os.PathSeparator)) {
			return nil
		}
		if !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// indMutationCase runs the same mutation args natively and via the oracle into
// two independent copies of the source workflow, then diffs stdout/stderr/exit
// AND the resulting on-disk .md files. git init so absolute archive dests work.
func indMutationCase(t *testing.T, name, src string, env []string, args ...string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		nDir := indCopyTree(t, src)
		oDir := indCopyTree(t, src)
		gitInit(t, nDir)
		gitInit(t, oDir)
		nArgs := withWorkflowDir(args, nDir)
		oArgs := withWorkflowDir(args, oDir)
		nOut, nErr, nCode := indRunNative(t, nDir, env, "", nArgs...)
		oOut, oErr, oCode := indRunOracle(t, oDir, env, "", oArgs...)
		// Each side ran in its own temp root, so normalize each against its own
		// root before comparing the placeholder forms.
		if nCode != oCode {
			t.Errorf("[%s] EXIT mismatch: native=%d oracle=%d\nnative-stderr=%q oracle-stderr=%q", name, nCode, oCode, nErr, oErr)
		}
		if nCode != 0 && nCode != 1 {
			t.Errorf("[%s] EXIT %d outside domain {0,1}", name, nCode)
		}
		if indNormalize(nOut, nDir) != indNormalize(oOut, oDir) {
			t.Errorf("[%s] STDOUT differs:\nnative=%q\noracle=%q", name, indNormalize(nOut, nDir), indNormalize(oOut, oDir))
		}
		if indNormalize(nErr, nDir) != indNormalize(oErr, oDir) {
			t.Errorf("[%s] STDERR differs:\nnative=%q\noracle=%q", name, indNormalize(nErr, nDir), indNormalize(oErr, oDir))
		}
		nFiles := indReadAll(t, nDir)
		oFiles := indReadAll(t, oDir)
		if len(nFiles) != len(oFiles) {
			t.Errorf("[%s] file-set size differs: native=%d oracle=%d\nnative=%v\noracle=%v", name, len(nFiles), len(oFiles), keysOf(nFiles), keysOf(oFiles))
		}
		for rel, oContent := range oFiles {
			nContent, ok := nFiles[rel]
			if !ok {
				t.Errorf("[%s] file %s present in oracle, missing in native", name, rel)
				continue
			}
			// Normalize before diffing: a bare-timestamp --set auto-fills now() on
			// each side independently, so the written `started:`/`completed:` stamps
			// straddle a wall-clock second boundary and diverge intermittently. The
			// timestamp normalization (test-only, never in product) makes the diff
			// stable; each side is normalized against its own temp root.
			if indNormalize(nContent, nDir) != indNormalize(oContent, oDir) {
				t.Errorf("[%s] file %s differs:\n--- native ---\n%q\n--- oracle ---\n%q", name, rel, indNormalize(nContent, nDir), indNormalize(oContent, oDir))
			}
		}
		for rel := range nFiles {
			if _, ok := oFiles[rel]; !ok {
				t.Errorf("[%s] file %s present in native, missing in oracle", name, rel)
			}
		}
	})
}

func keysOf(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func withWorkflowDir(args []string, dir string) []string {
	out := make([]string, len(args))
	copy(out, args)
	for i, a := range out {
		if a == "@WD@" {
			out[i] = dir
		}
	}
	return out
}

// TestIndMutationSeq diffs --set (field/clear/bare-fill/insert) and --archive
// (flat + folder) byte-for-byte against the oracle, including unknown-field
// preservation (issue/source/tracker-url on 001-alpha).
func TestIndMutationSeq(t *testing.T) {
	src := indSeqWorkflow(t)
	env := indEnv(t)
	indMutationCase(t, "set-field", src, env, "--workflow-dir", "@WD@", "--set", "001", "score=0.95")
	indMutationCase(t, "set-clear", src, env, "--workflow-dir", "@WD@", "--set", "002", "source=")
	indMutationCase(t, "set-bare-timestamp-fill", src, env, "--workflow-dir", "@WD@", "--set", "002", "started")
	indMutationCase(t, "set-insert-missing", src, env, "--workflow-dir", "@WD@", "--set", "002", "owner=alice")
	indMutationCase(t, "set-unrelated-keeps-unknown", src, env, "--workflow-dir", "@WD@", "--set", "001", "status=ideation")
	indMutationCase(t, "archive-flat", src, env, "--workflow-dir", "@WD@", "--archive", "001")
	indMutationCase(t, "archive-folder", src, env, "--workflow-dir", "@WD@", "--archive", "003")
}

// indReadCase runs a read-only args list natively + oracle in the same dir and
// diffs all channels.
func indReadCase(t *testing.T, name, root string, env []string, args ...string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		nOut, nErr, nCode := indRunNative(t, root, env, "", args...)
		oOut, oErr, oCode := indRunOracle(t, root, env, "", args...)
		indDiff(t, name, root, nOut, nErr, nCode, oOut, oErr, oCode)
	})
}

// TestIndValidationDefects diffs each validation defect class against the oracle.
func TestIndValidationDefects(t *testing.T) {
	env := indEnv(t)
	mk := func(entities map[string]string, idStyle, stagesExtra string) string {
		root := t.TempDir()
		readme := "---\nentity-label: task\nid-style: " + idStyle + "\nstages:\n  states:\n    - name: backlog\n      initial: true\n" + stagesExtra + "    - name: done\n      terminal: true\n---\n# WF\n"
		indWrite(t, filepath.Join(root, "README.md"), readme)
		for name, body := range entities {
			indWrite(t, filepath.Join(root, name), body)
		}
		return root
	}

	r1 := mk(map[string]string{
		"001-a.md": "---\nid: \"001\"\ntitle: A\nstatus: backlog\n---\n# A\n",
		"002-b.md": "---\ntitle: B no id\nstatus: backlog\n---\n# B\n",
	}, "sequential", "")
	indReadCase(t, "missing-id-validate", r1, env, "--workflow-dir", r1, "--validate")
	indReadCase(t, "missing-id-default-gated", r1, env, "--workflow-dir", r1)

	r2 := mk(map[string]string{
		"001-a.md": "---\nid: abc\ntitle: A\nstatus: backlog\n---\n# A\n",
	}, "sequential", "")
	indReadCase(t, "non-numeric-id", r2, env, "--workflow-dir", r2, "--validate")

	r3 := mk(map[string]string{
		"001-a.md": "---\nid: \"001\"\ntitle: A\nstatus: backlog\n---\n# A\n",
		"002-b.md": "---\nid: \"001\"\ntitle: B\nstatus: backlog\n---\n# B\n",
	}, "sequential", "")
	indReadCase(t, "duplicate-id", r3, env, "--workflow-dir", r3, "--validate")

	r4 := mk(map[string]string{
		"001-a.md":       "---\nid: \"001\"\ntitle: A\nstatus: backlog\n---\n# A\n",
		"001-a/index.md": "---\nid: \"001\"\ntitle: A folder\nstatus: backlog\n---\n# A\n",
	}, "sequential", "")
	indReadCase(t, "flat-folder-conflict", r4, env, "--workflow-dir", r4, "--validate")

	r5 := mk(map[string]string{
		"001-a.md": "---\nid: \"001\"\ntitle: A\nstatus: backlog\n---\n# A\n",
	}, "sequential", "    - name: Bad_Stage\n")
	indReadCase(t, "bad-stage-name", r5, env, "--workflow-dir", r5, "--validate")

	r6 := mk(map[string]string{
		"abcdefghjkmnpqrstvwxyz23-a.md": "---\nid: NOTVALIDB32!!\ntitle: A\nstatus: backlog\n---\n# A\n",
	}, "sd-b32", "")
	indReadCase(t, "sdb32-invalid-id", r6, env, "--workflow-dir", r6, "--validate")

	r7 := mk(map[string]string{
		"intro.md": "---\ntitle: Intro\nstatus: backlog\n---\n# Intro\n",
	}, "slug", "")
	indReadCase(t, "slug-valid", r7, env, "--workflow-dir", r7, "--validate")

	r8 := mk(map[string]string{
		"001-a.md": "---\nid: \"001\"\ntitle: A\nstatus: backlog\n---\n# A\n",
	}, "sequential", "")
	indReadCase(t, "valid-positive", r8, env, "--workflow-dir", r8, "--validate")
}

// TestIndUsageErrorsExitDomain diffs usage/parse errors: each must exit 1 (never
// 2) with byte-identical stderr.
func TestIndUsageErrorsExitDomain(t *testing.T) {
	root := indSeqWorkflow(t)
	env := indEnv(t)
	cases := []struct {
		name string
		args []string
	}{
		{"bad-where-no-op", []string{"--workflow-dir", root, "--where", "statusideation"}},
		{"where-missing-arg", []string{"--workflow-dir", root, "--where"}},
		{"fields-and-all-fields", []string{"--workflow-dir", root, "--fields", "x", "--all-fields"}},
		{"boot-with-next", []string{"--workflow-dir", root, "--boot", "--next"}},
		{"next-id-with-set", []string{"--workflow-dir", root, "--next-id", "--set", "001", "x=y"}},
		{"resolve-missing-arg", []string{"--workflow-dir", root, "--resolve"}},
		{"workflow-dir-missing-arg", []string{"--workflow-dir"}},
		{"id-material-without-next-id", []string{"--workflow-dir", root, "--id-seed", "x"}},
		{"root-without-discover", []string{"--workflow-dir", root, "--root", root}},
	}
	for _, c := range cases {
		indReadCase(t, c.name, root, env, c.args...)
	}
}

// TestIndNewAtomicCreate verifies --new's contract. --new is Decision B's NEW
// surface — the oracle (current Python script) has ZERO --new support (verified:
// `grep -c -- --new` on the oracle is 0). So there is no oracle to diff against;
// AC-7 instead pins --new to two independent checks: (1) the minted id equals
// `--next-id` under IDENTICAL pinned env, and (2) the workflow passes --validate
// immediately after with no id-less window. Both are checked here.
func TestIndNewAtomicCreate(t *testing.T) {
	env := indEnv(t)
	body := "---\ntitle: New task\nstatus: backlog\nscore: \"0.42\"\nsource: roadmap\n---\n# New task\nseed body\n"

	run := func(name, idStyle string, folder bool, idSeed, idActor string) {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			readme := "---\nentity-label: task\nid-style: " + idStyle + "\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n"
			indWrite(t, filepath.Join(root, "README.md"), readme)

			// Independent expected id: ask --next-id under the same pinned env. For
			// slug style there is no --next-id; the minted id is the slug itself.
			expectedID := ""
			if idStyle != "slug" {
				niArgs := []string{"--workflow-dir", root, "--next-id"}
				if idSeed != "" {
					niArgs = append(niArgs, "--id-seed", idSeed)
				}
				if idActor != "" {
					niArgs = append(niArgs, "--id-actor", idActor)
				}
				out, _, code := indRunNative(t, root, env, "", niArgs...)
				if code != 0 {
					t.Fatalf("[%s] --next-id failed code=%d out=%q", name, code, out)
				}
				expectedID = strings.TrimSpace(out)
			} else {
				expectedID = name // slug style: id is the slug; we use the new slug below
			}

			newSlug := "new-task"
			newArgs := []string{"--workflow-dir", root, "--new", newSlug}
			if folder {
				newArgs = []string{"--workflow-dir", root, "--new", "--folder", newSlug}
			}
			if idSeed != "" {
				newArgs = append(newArgs, "--id-seed", idSeed)
			}
			if idActor != "" {
				newArgs = append(newArgs, "--id-actor", idActor)
			}
			_, nErr, nCode := indRunNative(t, root, env, body, newArgs...)
			if nCode != 0 {
				t.Fatalf("[%s] --new exit=%d stderr=%q", name, nCode, nErr)
			}

			// The seed file must exist with the minted id stamped into frontmatter.
			seedPath := filepath.Join(root, newSlug+".md")
			if folder {
				seedPath = filepath.Join(root, newSlug, "index.md")
			}
			seed, err := os.ReadFile(seedPath)
			if err != nil {
				t.Fatalf("[%s] seed not written at %s: %v", name, seedPath, err)
			}
			// --new stamps the minted id for sequential/sd-b32, where the id is a
			// real stored field. For id-style: slug the identity IS the slug; a
			// stored id line is redundant and would make --resolve emit id=<slug>
			// where hand-authored slug entities emit id= (empty), so the slug seed
			// carries NO id line and equals STDIN verbatim (byte-identity).
			if idStyle == "slug" {
				if strings.Contains(string(seed), "\nid:") || strings.HasPrefix(string(seed), "id:") {
					t.Errorf("[%s] slug seed should carry no id line:\n%s", name, string(seed))
				}
				if string(seed) != body {
					t.Errorf("[%s] slug seed not byte-identical to STDIN:\ngot=%q\nwant=%q", name, string(seed), body)
				}
			} else {
				wantLine := "id: " + expectedID
				if !strings.Contains(string(seed), wantLine) {
					t.Errorf("[%s] seed missing minted id line %q:\n%s", name, wantLine, string(seed))
				}
				// Seed must equal STDIN with exactly the id-stamp added (byte-identity
				// of every other line — AC-5 preservation), verified by reconstructing.
				wantSeed := string(stampID([]byte(body), expectedID))
				if string(seed) != wantSeed {
					t.Errorf("[%s] seed not STDIN+id-stamp:\ngot=%q\nwant=%q", name, string(seed), wantSeed)
				}
			}

			// No id-less window: --validate is VALID immediately after.
			vOut, vErr, vCode := indRunNative(t, root, env, "", "--workflow-dir", root, "--validate")
			if vCode != 0 || !strings.Contains(vOut, "VALID") {
				t.Errorf("[%s] post-create --validate code=%d out=%q stderr=%q", name, vCode, vOut, vErr)
			}
		})
	}

	run("seq-flat", "sequential", false, "", "")
	run("seq-folder", "sequential", true, "", "")
	run("sdb32-seeded", "sd-b32", false, "new-task", "ind-actor")
	run("slug-flat", "slug", false, "", "")
}

// TestIndNewGuards verifies the --new guard paths. The oracle has no --new, so
// these assert the native behavior directly: each guard exits 1 with an Error:
// line and writes no seed.
func TestIndNewGuards(t *testing.T) {
	env := indEnv(t)

	mkSeqWF := func(extra map[string]string) string {
		root := t.TempDir()
		indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: task\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n---\n# WF\n")
		for n, b := range extra {
			indWrite(t, filepath.Join(root, n), b)
		}
		return root
	}

	t.Run("slug-exists", func(t *testing.T) {
		root := mkSeqWF(map[string]string{"dup.md": "---\nid: \"001\"\ntitle: Existing\nstatus: backlog\n---\n# Existing\n"})
		before, _ := os.ReadFile(filepath.Join(root, "dup.md"))
		_, nErr, nCode := indRunNative(t, root, env, "---\ntitle: New\nstatus: backlog\n---\n# New\n", "--workflow-dir", root, "--new", "dup")
		if nCode != 1 || !strings.HasPrefix(nErr, "Error:") {
			t.Errorf("slug-exists want exit1+Error got code=%d stderr=%q", nCode, nErr)
		}
		after, _ := os.ReadFile(filepath.Join(root, "dup.md"))
		if string(before) != string(after) {
			t.Errorf("slug-exists clobbered existing file")
		}
	})

	t.Run("missing-fence", func(t *testing.T) {
		root := mkSeqWF(nil)
		_, nErr, nCode := indRunNative(t, root, env, "no fence here\n", "--workflow-dir", root, "--new", "nofence")
		if nCode != 1 || !strings.HasPrefix(nErr, "Error:") {
			t.Errorf("missing-fence want exit1+Error got code=%d stderr=%q", nCode, nErr)
		}
		if _, err := os.Stat(filepath.Join(root, "nofence.md")); !os.IsNotExist(err) {
			t.Errorf("missing-fence wrote a seed despite the guard")
		}
	})

	t.Run("slug-style-with-id-seed", func(t *testing.T) {
		root := t.TempDir()
		indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: doc\nid-style: slug\nstages:\n  states:\n    - name: draft\n      initial: true\n---\n# WF\n")
		_, nErr, nCode := indRunNative(t, root, env, "---\ntitle: New\nstatus: draft\n---\n# New\n", "--workflow-dir", root, "--new", "newdoc", "--id-seed", "x")
		if nCode != 1 || !strings.HasPrefix(nErr, "Error:") {
			t.Errorf("slug-style-with-id-seed want exit1+Error got code=%d stderr=%q", nCode, nErr)
		}
	})
}

// TestIndEOFNewlineIdentity is the byte-parity trap: a no-trailing-newline file
// must NOT gain one on mutation, and a trailing-newline file must keep one. Both
// native and oracle are mutated on identical copies and the bytes diffed.
func TestIndEOFNewlineIdentity(t *testing.T) {
	env := indEnv(t)
	mk := func(trailing bool) (string, string) {
		nDir := t.TempDir()
		oDir := t.TempDir()
		readme := "---\nentity-label: task\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n---\n# WF\n"
		entity := "---\nid: \"001\"\ntitle: A\nstatus: backlog\nscore: \"0.5\"\n---\n# A\nbody"
		if trailing {
			entity += "\n"
		}
		for _, d := range []string{nDir, oDir} {
			indWrite(t, filepath.Join(d, "README.md"), readme)
			indWrite(t, filepath.Join(d, "001-a.md"), entity)
			gitInit(t, d)
		}
		return nDir, oDir
	}
	for _, tc := range []struct {
		name     string
		trailing bool
	}{
		{"no-trailing-newline", false},
		{"with-trailing-newline", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nDir, oDir := mk(tc.trailing)
			indRunNative(t, nDir, env, "", "--workflow-dir", nDir, "--set", "001", "score=0.9")
			indRunOracle(t, oDir, env, "", "--workflow-dir", oDir, "--set", "001", "score=0.9")
			nb, _ := os.ReadFile(filepath.Join(nDir, "001-a.md"))
			ob, _ := os.ReadFile(filepath.Join(oDir, "001-a.md"))
			if string(nb) != string(ob) {
				t.Errorf("[%s] mutated file differs:\nnative=%q\noracle=%q", tc.name, string(nb), string(ob))
			}
			nEndsNL := strings.HasSuffix(string(nb), "\n")
			if nEndsNL != tc.trailing {
				t.Errorf("[%s] native EOF-newline=%v want %v (identity violated)", tc.name, nEndsNL, tc.trailing)
			}
		})
	}
}

// TestIndArchiveDestSpelling locks the trap that --archive's printed dest tracks
// the --workflow-dir spelling: a relative "." spelling yields a relative
// ./_archive/... dest, an absolute spelling yields an absolute dest. Diffed
// against the oracle (which runs with cwd=workflow-dir for the relative case).
func TestIndArchiveDestSpelling(t *testing.T) {
	env := indEnv(t)
	mk := func() (string, string) {
		nDir := t.TempDir()
		oDir := t.TempDir()
		readme := "---\nentity-label: task\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n---\n# WF\n"
		for _, d := range []string{nDir, oDir} {
			indWrite(t, filepath.Join(d, "README.md"), readme)
			indWrite(t, filepath.Join(d, "001-a.md"), "---\nid: \"001\"\ntitle: A\nstatus: backlog\nscore: \"0.5\"\n---\n# A\n")
			indWrite(t, filepath.Join(d, "002-b.md"), "---\nid: \"002\"\ntitle: B\nstatus: backlog\nscore: \"0.5\"\n---\n# B\n")
			gitInit(t, d)
		}
		return nDir, oDir
	}

	t.Run("relative-dot", func(t *testing.T) {
		nDir, oDir := mk()
		// Native: Dir is the workflow dir, --workflow-dir is ".".
		nOut, _, nCode := indRunNative(t, nDir, env, "", "--workflow-dir", ".", "--archive", "001")
		// Oracle: cwd is the workflow dir, --workflow-dir is ".".
		oOut, _, oCode := indRunOracle(t, oDir, env, "", "--workflow-dir", ".", "--archive", "001")
		if nCode != oCode {
			t.Errorf("relative-dot EXIT native=%d oracle=%d", nCode, oCode)
		}
		if nOut != oOut {
			t.Errorf("relative-dot dest differs:\nnative=%q\noracle=%q", nOut, oOut)
		}
		if !strings.Contains(nOut, "archived: ./_archive/001-a.md") {
			t.Errorf("relative-dot native dest not relative: %q", nOut)
		}
		// The file actually moved under the workflow dir.
		if _, err := os.Stat(filepath.Join(nDir, "_archive", "001-a.md")); err != nil {
			t.Errorf("relative-dot did not move file: %v", err)
		}
	})

	t.Run("absolute", func(t *testing.T) {
		nDir, oDir := mk()
		nOut, _, nCode := indRunNative(t, nDir, env, "", "--workflow-dir", nDir, "--archive", "002")
		oOut, _, oCode := indRunOracle(t, oDir, env, "", "--workflow-dir", oDir, "--archive", "002")
		if nCode != oCode {
			t.Errorf("absolute EXIT native=%d oracle=%d", nCode, oCode)
		}
		if indNormalize(nOut, nDir) != indNormalize(oOut, oDir) {
			t.Errorf("absolute dest differs:\nnative=%q\noracle=%q", indNormalize(nOut, nDir), indNormalize(oOut, oDir))
		}
		if !strings.Contains(nOut, nDir+"/_archive/002-b.md") {
			t.Errorf("absolute native dest not absolute under root: %q", nOut)
		}
	})
}

// TestIndResolveRealpathAsymmetry locks the trap that --resolve realpath's the
// workflow= field (macOS /tmp->/private/tmp, /var->/private/var) but NOT the
// path= field. Diffed against the oracle on the same temp root. t.TempDir on
// macOS lives under /var (symlinked to /private/var), so the asymmetry is real.
func TestIndResolveRealpathAsymmetry(t *testing.T) {
	env := indEnv(t)
	mk := func() string {
		root := t.TempDir()
		indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: task\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n---\n# WF\n")
		indWrite(t, filepath.Join(root, "001-a.md"), "---\nid: \"001\"\ntitle: A\nstatus: backlog\nscore: \"0.5\"\n---\n# A\n")
		return root
	}
	root := mk()
	nOut, _, nCode := indRunNative(t, root, env, "", "--workflow-dir", root, "--resolve", "001")
	oOut, _, oCode := indRunOracle(t, root, env, "", "--workflow-dir", root, "--resolve", "001")
	if nCode != oCode {
		t.Fatalf("EXIT native=%d oracle=%d", nCode, oCode)
	}
	// Raw (un-normalized) comparison so the realpath asymmetry itself is asserted.
	if nOut != oOut {
		t.Errorf("resolve raw output differs (realpath asymmetry not matched):\nnative=%q\noracle=%q", nOut, oOut)
	}
	real, _ := filepath.EvalSymlinks(root)
	if real != root {
		// On macOS the temp root is symlinked: confirm workflow= used the realpath
		// while path= used the as-passed (un-realpath'd) spelling.
		if !strings.Contains(nOut, "workflow="+real+" ") {
			t.Errorf("workflow= not realpath'd: want %q in %q", "workflow="+real, nOut)
		}
		if !strings.Contains(nOut, "path="+root+"/001-a.md") {
			t.Errorf("path= unexpectedly realpath'd: want %q in %q", "path="+root, nOut)
		}
	}
}

// toCRLF rewrites every LF in s as CRLF, producing a Windows-line-ending fixture.
func toCRLF(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// TestIndCRLFParity locks the M1 trap: the oracle reads with Python text-mode
// universal newlines (\r\n and \r both become \n), so a CRLF entity is listed
// and a CRLF README's stages block is read; native must match. A CRLF entity
// that native drops is removed from discovery, validation, AND the sequential-id
// max (colliding-id risk); a CRLF README makes --next exit 1 ("no stages block")
// where the oracle exits 0. Both an entity and the README are CRLF here, diffed
// across read flags plus a --set round-trip (the oracle's universal-newline read
// means the mutated file comes out LF; native must too).
func TestIndCRLFParity(t *testing.T) {
	env := indEnv(t)
	mk := func(t *testing.T) string {
		root := t.TempDir()
		indWrite(t, filepath.Join(root, "README.md"), toCRLF("---\nentity-label: task\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n"))
		indWrite(t, filepath.Join(root, "001-crlf.md"), toCRLF("---\nid: \"001\"\ntitle: CRLF entity\nstatus: backlog\nscore: \"0.5\"\nsource: roadmap\n---\n# CRLF\nbody\n"))
		return root
	}

	t.Run("read-flags", func(t *testing.T) {
		root := mk(t)
		cases := []struct {
			name string
			args []string
		}{
			{"default", []string{"--workflow-dir", root}},
			{"next", []string{"--workflow-dir", root, "--next"}},
			{"validate", []string{"--workflow-dir", root, "--validate"}},
			{"next-id", []string{"--workflow-dir", root, "--next-id"}},
			{"resolve", []string{"--workflow-dir", root, "--resolve", "001"}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				nOut, nErr, nCode := indRunNative(t, root, env, "", c.args...)
				oOut, oErr, oCode := indRunOracle(t, root, env, "", c.args...)
				indDiff(t, c.name, root, nOut, nErr, nCode, oOut, oErr, oCode)
				// Lock the concrete symptoms the feedback named.
				if c.name == "default" && !strings.Contains(nOut, "001-crlf") {
					t.Errorf("native dropped the CRLF entity from the default table:\n%s", nOut)
				}
				if c.name == "next" && nCode != 0 {
					t.Errorf("native --next exit=%d on CRLF README (want 0)\nstderr=%q", nCode, nErr)
				}
			})
		}
	})

	t.Run("set-roundtrip-normalizes-to-lf", func(t *testing.T) {
		src := mk(t)
		// The oracle's universal-newline read normalizes the file to LF on --set;
		// native must produce byte-identical output, so the mutated file has no CRLF.
		indMutationCase(t, "crlf-set", src, env, "--workflow-dir", "@WD@", "--set", "001", "score=0.9")
	})
}

// TestIndExoticScoreSort locks the M2 trap: the sort key parses score with Go's
// strconv.ParseFloat, which accepts hex-floats (`0x1p4` -> 16) that Python
// float() rejects (-> ValueError -> score_val 0). The oracle therefore sorts a
// hex-float score as non-numeric (after every real number in the same stage),
// where native would mis-rank it as 16. Fixtures pin one hex-float score plus
// normal high/low scores in the same stage; both default and --next ordering are
// diffed raw against the oracle.
func TestIndExoticScoreSort(t *testing.T) {
	env := indEnv(t)
	root := t.TempDir()
	indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: task\nid-style: sequential\nstages:\n  states:\n    - name: backlog\n      initial: true\n    - name: done\n      terminal: true\n---\n# WF\n")
	// 0x1p4: Go ParseFloat -> 16 (would sort first); Python float() -> ValueError
	// -> score_val 0 (sorts after the real numbers below).
	indWrite(t, filepath.Join(root, "001-hex.md"), "---\nid: \"001\"\ntitle: HexScore\nstatus: backlog\nscore: \"0x1p4\"\nsource: r\n---\n# A\n")
	indWrite(t, filepath.Join(root, "002-high.md"), "---\nid: \"002\"\ntitle: NormalHigh\nstatus: backlog\nscore: \"0.9\"\nsource: r\n---\n# B\n")
	indWrite(t, filepath.Join(root, "003-low.md"), "---\nid: \"003\"\ntitle: NormalLow\nstatus: backlog\nscore: \"0.1\"\nsource: r\n---\n# C\n")
	// A bare non-numeric score (also Python-float-rejected -> 0); locks that
	// genuinely-non-numeric scores stay parity-equal alongside the hex case.
	indWrite(t, filepath.Join(root, "004-na.md"), "---\nid: \"004\"\ntitle: NotANumber\nstatus: backlog\nscore: \"high\"\nsource: r\n---\n# D\n")

	for _, c := range []struct {
		name string
		args []string
	}{
		{"default", []string{"--workflow-dir", root}},
		{"next", []string{"--workflow-dir", root, "--next"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			nOut, nErr, nCode := indRunNative(t, root, env, "", c.args...)
			oOut, oErr, oCode := indRunOracle(t, root, env, "", c.args...)
			indDiff(t, c.name, root, nOut, nErr, nCode, oOut, oErr, oCode)
		})
	}
}

// TestIndNewSlugNoIDStamp locks that for id-style: slug, --new does NOT stamp a
// redundant `id: <slug>` line into the seed. The oracle has no --new, so the
// parity reference is a hand-authored slug entity, which carries no id field and
// resolves to `id=` (empty). A --new slug seed must resolve identically; a stray
// `id: <slug>` would make --resolve/--short-id emit `id=<slug>`/`<slug>` where
// the oracle emits `id=`/the slug. We build two slug entities — one hand-authored,
// one via --new — and diff each entity's native --resolve against the oracle's.
func TestIndNewSlugNoIDStamp(t *testing.T) {
	env := indEnv(t)
	root := t.TempDir()
	indWrite(t, filepath.Join(root, "README.md"), "---\nentity-label: doc\nid-style: slug\nstages:\n  states:\n    - name: draft\n      initial: true\n    - name: published\n      terminal: true\n---\n# Slug WF\n")
	// Hand-authored slug entity (no id field) — the parity reference.
	indWrite(t, filepath.Join(root, "manual.md"), "---\ntitle: Manual\nstatus: draft\nscore: \"0.7\"\nsource: roadmap\n---\n# Manual\n")

	// Create a second slug entity via --new.
	newBody := "---\ntitle: Minted\nstatus: draft\nscore: \"0.4\"\nsource: roadmap\n---\n# Minted\nseed\n"
	_, nErr, nCode := indRunNative(t, root, env, newBody, "--workflow-dir", root, "--new", "minted")
	if nCode != 0 {
		t.Fatalf("--new slug exit=%d stderr=%q", nCode, nErr)
	}

	// The seed must not carry an id line at all (matching hand-authored slugs).
	seed, err := os.ReadFile(filepath.Join(root, "minted.md"))
	if err != nil {
		t.Fatalf("seed not written: %v", err)
	}
	for _, line := range strings.Split(string(seed), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "id:") {
			t.Errorf("slug --new stamped a redundant id line %q into the seed:\n%s", line, string(seed))
		}
	}

	// Both entities resolve identically native-vs-oracle, with id= empty.
	for _, slug := range []string{"manual", "minted"} {
		nOut, nE, nC := indRunNative(t, root, env, "", "--workflow-dir", root, "--resolve", slug)
		oOut, oE, oC := indRunOracle(t, root, env, "", "--workflow-dir", root, "--resolve", slug)
		indDiff(t, "resolve-"+slug, root, nOut, nE, nC, oOut, oE, oC)
		if !strings.Contains(nOut, "id= ") {
			t.Errorf("resolve %s: native id not empty: %q", slug, nOut)
		}
	}
}
