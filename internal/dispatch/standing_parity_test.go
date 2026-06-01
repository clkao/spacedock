// ABOUTME: standing-subcommand three-channel parity — list/show/spawn-standing and
// ABOUTME: the build _mods fetch-line branch, native vs vendored Python over fixtures.
package dispatch

import (
	"path/filepath"
	"strings"
	"testing"
)

// standingMod returns a standing-teammate mod body with the given frontmatter and
// optional ## Routing Usage / ## Agent Prompt sections. A mod's spawn name comes
// from its ## Hook: startup section.
func standingMod(name, model, description, routingUsage string) string {
	fm := "---\nstanding: true\nname: " + name + "\n"
	if description != "" {
		fm += "description: " + description + "\n"
	}
	fm += "---\n"
	body := "## Hook: startup\n" +
		"- subagent_type: general-purpose\n" +
		"- name: " + name + "\n" +
		"- model: " + model + "\n"
	if routingUsage != "" {
		body += "## Routing Usage\n" + routingUsage + "\n"
	}
	body += "## Agent Prompt\nYou are " + name + ".\n"
	return fm + body
}

// writeMods materializes a _mods dir under workflowDir with the given mod files
// (filename -> content) and returns the mods dir path.
func writeMods(t *testing.T, workflowDir string, mods map[string]string) string {
	t.Helper()
	modsDir := filepath.Join(workflowDir, "_mods")
	for name, content := range mods {
		writeFile(t, filepath.Join(modsDir, name), content)
	}
	return modsDir
}

// TestListStandingParity drives list-standing over a _mods fixture mixing standing
// and non-standing mods: the native and oracle emit the same sorted absolute paths.
func TestListStandingParity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	writeMods(t, wd, map[string]string{
		"comm-officer.md":    standingMod("comm-officer", "sonnet", "prose polisher", ""),
		"science-officer.md": standingMod("science-officer", "opus", "researcher", ""),
		"not-standing.md":    "---\nstanding: false\nname: nope\n---\nbody\n",
	})

	oracle := runOracle(t, wd, home, "", "list-standing", "--workflow-dir", wd)
	native := runNative("", "list-standing", "--workflow-dir", wd)
	assertParity(t, "list-standing", native, oracle)
}

// TestListStandingParityNoMods drives the degenerate path: a workflow with no
// _mods dir emits empty stdout, exit 0, on both sides.
func TestListStandingParityNoMods(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	oracle := runOracle(t, wd, home, "", "list-standing", "--workflow-dir", wd)
	native := runNative("", "list-standing", "--workflow-dir", wd)
	assertParity(t, "list-standing-no-mods", native, oracle)
}

// TestShowStandingParity drives show-standing over a _mods fixture: one mod with a
// ## Routing Usage section (extracted body) and one without (fallback one-liner),
// so both render branches are byte-compared native vs oracle.
func TestShowStandingParity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	writeMods(t, wd, map[string]string{
		"comm-officer.md": standingMod("comm-officer", "sonnet", "prose polisher",
			"Send a draft; reply is the polished text.\n\nKeep it tight."),
		"science-officer.md": standingMod("science-officer", "opus", "researcher", ""),
	})

	oracle := runOracle(t, wd, home, "", "show-standing", "--workflow-dir", wd)
	native := runNative("", "show-standing", "--workflow-dir", wd)
	assertParity(t, "show-standing", native, oracle)
}

// TestShowStandingParityEmpty drives the degenerate empty case (no standing mods):
// empty stdout, exit 0, on both sides.
func TestShowStandingParityEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	writeMods(t, wd, map[string]string{
		"not-standing.md": "---\nstanding: false\nname: nope\n---\nbody\n",
	})
	oracle := runOracle(t, wd, home, "", "show-standing", "--workflow-dir", wd)
	native := runNative("", "show-standing", "--workflow-dir", wd)
	assertParity(t, "show-standing-empty", native, oracle)
}

// TestSpawnStandingParitySpecEmit drives the spec-emit path: the named member is
// NOT in the (empty) team config, so both sides emit the Agent() spec JSON.
func TestSpawnStandingParitySpecEmit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	modPath := filepath.Join(wd, "_mods", "comm-officer.md")
	writeFile(t, modPath, standingMod("comm-officer", "sonnet", "prose polisher", ""))
	// A team config that does NOT list comm-officer, so member_exists is false.
	claudeFixture{
		team:    "fixture-team",
		session: "s",
		members: []fixtureMember{{name: "team-lead", model: "opus"}},
		jsonls:  map[string]string{},
	}.write(t, home)

	oracle := runOracle(t, wd, home, "", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
	native := runNative("", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
	assertParity(t, "spawn-standing-spec", native, oracle)
}

// TestSpawnStandingParitySpecNonASCIIPrompt is the A-2 non-ASCII parity case for
// the spawn-standing spec-emit path. The FO forwards spec.prompt VERBATIM, so a
// non-ASCII Agent Prompt (em-dash U+2014 here) must serialize byte-identically
// to the oracle: Python json.dumps escapes it to \u2014, where Go's encoder
// emitted it raw before the EmitPythonJSON ensure_ascii fix. Driving the
// spec-emit path with such a prompt asserts native == Python bytes including the
// escaped prompt, and the escape guard makes the parity assertion load-bearing.
func TestSpawnStandingParitySpecNonASCIIPrompt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	modPath := filepath.Join(wd, "_mods", "comm-officer.md")
	// An Agent Prompt with an em-dash, so spec.prompt carries a non-ASCII rune.
	mod := "---\nstanding: true\nname: comm-officer\n---\n" +
		"## Hook: startup\n" +
		"- subagent_type: general-purpose\n" +
		"- name: comm-officer\n" +
		"- model: sonnet\n" +
		"## Agent Prompt\nYou are comm-officer — the prose polisher.\n"
	writeFile(t, modPath, mod)
	// A team config that does NOT list comm-officer, so member_exists is false and
	// both sides take the spec-emit branch.
	claudeFixture{
		team:    "fixture-team",
		session: "s",
		members: []fixtureMember{{name: "team-lead", model: "opus"}},
		jsonls:  map[string]string{},
	}.write(t, home)

	oracle := runOracle(t, wd, home, "", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
	native := runNative("", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
	assertParity(t, "spawn-standing-spec-nonascii", native, oracle)
	assertEmDashEscaped(t, native.stdout)
}

// assertEmDashEscaped asserts stdout carries the literal \u2014 escape and no raw
// UTF-8 em-dash, so the ensure_ascii fix is demonstrably what makes the byte
// parity hold (a guard against the harness comparing raw == raw if reverted).
func assertEmDashEscaped(t *testing.T, stdout string) {
	t.Helper()
	if !strings.Contains(stdout, "\\u2014") {
		t.Errorf("stdout missing the \\u2014 ensure_ascii escape (em-dash emitted raw?):\n%s", stdout)
	}
	if strings.ContainsRune(stdout, '—') {
		t.Errorf("stdout contains a raw em-dash (ensure_ascii escaping not applied):\n%s", stdout)
	}
}

// TestSpawnStandingParityAlreadyAlive drives the already-alive path: the team
// config already lists the declared member, so both sides emit the compact
// already-alive JSON (exit 0).
func TestSpawnStandingParityAlreadyAlive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	wd := t.TempDir()
	modPath := filepath.Join(wd, "_mods", "comm-officer.md")
	writeFile(t, modPath, standingMod("comm-officer", "sonnet", "prose polisher", ""))
	claudeFixture{
		team:    "fixture-team",
		session: "s",
		members: []fixtureMember{{name: "comm-officer", model: "sonnet"}},
		jsonls:  map[string]string{},
	}.write(t, home)

	oracle := runOracle(t, wd, home, "", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
	native := runNative("", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
	assertParity(t, "spawn-standing-alive", native, oracle)
}

// TestSpawnStandingParityLoudFailures drives the loud-failure paths byte-compared
// native vs oracle: missing mod, missing standing:true, missing ## Agent Prompt,
// missing model, and a bad model enum value. Each must exit 1 with the same
// stderr.
func TestSpawnStandingParityLoudFailures(t *testing.T) {
	cases := []struct {
		name    string
		mod     string // mod body; "" means do not create the file
		modName string
	}{
		{
			name:    "missing-mod",
			mod:     "",
			modName: "ghost.md",
		},
		{
			name:    "not-standing",
			mod:     "---\nstanding: false\nname: nope\n---\n## Hook: startup\n- model: opus\n## Agent Prompt\nx\n",
			modName: "nope.md",
		},
		{
			name:    "missing-agent-prompt",
			mod:     "---\nstanding: true\nname: noprompt\n---\n## Hook: startup\n- subagent_type: general-purpose\n- name: noprompt\n- model: opus\n",
			modName: "noprompt.md",
		},
		{
			name:    "missing-model",
			mod:     "---\nstanding: true\nname: nomodel\n---\n## Hook: startup\n- subagent_type: general-purpose\n- name: nomodel\n## Agent Prompt\nx\n",
			modName: "nomodel.md",
		},
		{
			name:    "bad-model-enum",
			mod:     "---\nstanding: true\nname: badmodel\n---\n## Hook: startup\n- subagent_type: general-purpose\n- name: badmodel\n- model: gpt-4\n## Agent Prompt\nx\n",
			modName: "badmodel.md",
		},
		{
			name:    "trailing-heading",
			mod:     "---\nstanding: true\nname: trailer\n---\n## Hook: startup\n- subagent_type: general-purpose\n- name: trailer\n- model: opus\n## Agent Prompt\nbody\n## Trailing Section\noops\n",
			modName: "trailer.md",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			wd := t.TempDir()
			modPath := filepath.Join(wd, "_mods", tc.modName)
			if tc.mod != "" {
				writeFile(t, modPath, tc.mod)
			}
			oracle := runOracle(t, wd, home, "", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
			native := runNative("", "spawn-standing", "--mod", modPath, "--team", "fixture-team")
			assertParity(t, "spawn-standing-"+tc.name, native, oracle)
			if oracle.exit == 0 {
				t.Fatalf("loud-failure case %q exited 0 on oracle (fixture is wrong)", tc.name)
			}
		})
	}
}

// TestBuildModsParity extends the build parity to the _mods/show-standing branch:
// a workflow declaring a standing teammate emits the show-standing fetch line iff
// the oracle does, byte-identical after the claude-team→spacedock dispatch rewrite.
func TestBuildModsParity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), readmeWorktree(false))
	writeMods(t, root, map[string]string{
		"comm-officer.md": standingMod("comm-officer", "sonnet", "prose polisher", ""),
	})
	entityPath := filepath.Join(root, "thing.md")
	writeFile(t, entityPath, entityFM("Thing", "backlog", ""))
	gitInit(t, root)

	stdin := mergeStdin(map[string]any{
		"schema_version": 2,
		"entity_path":    entityPath,
		"workflow_dir":   root,
		"stage":          "backlog",
		"checklist":      []string{"- a", "- b"},
		"team_name":      "fixture-team",
		"bare_mode":      false,
	}, nil)

	oracle := runOracle(t, root, home, stdin, "build", "--workflow-dir", root)
	oracleBody := readDispatchBody(t, dispatchFilePathFromStdout(t, oracle.stdout))
	native := runNative(stdin, "build", "--workflow-dir", root)
	nativeBody := readDispatchBody(t, dispatchFilePathFromStdout(t, native.stdout))

	assertParity(t, "build-mods", native, oracle)
	wantBody := stripStateCommitGuidance(rewriteOracleFetch(oracleBody))
	gotBody := stripStateCommitGuidance(nativeBody)
	if gotBody != wantBody {
		t.Errorf("build-mods: dispatch body mismatch\n--- native ---\n%s\n--- oracle(rewritten) ---\n%s",
			gotBody, wantBody)
	}
}
