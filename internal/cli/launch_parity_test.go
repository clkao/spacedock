// ABOUTME: Recorded-Launch oracles for the sandbox knobs (AC-1..AC-8) and the
// ABOUTME: launch-parity gaps (LP-AC-1..3): fence task, codex resume, plugin-dir.
package cli

import (
	"bytes"
	"context"
	"testing"
)

// AC-1: --safehouse-enable=ssh,docker comma-splits into repeated --enable=KEY in
// the pre-`--` extra slot, after --trust-workdir-config.
func TestSafehouseEnableForwardsRepeatedFlags(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--safehouse-enable=ssh,docker"}, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"safehouse", "--trust-workdir-config", "--enable=ssh", "--enable=docker", "--",
		"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", wantBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// AC-2: --safehouse-add-dirs / --safehouse-add-dirs-ro forward path grants into
// the pre-`--` extra slot, in operator order.
func TestSafehouseAddDirsForwardsPathGrants(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--safehouse-add-dirs=/a", "--safehouse-add-dirs-ro=/b"}, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"safehouse", "--trust-workdir-config", "--add-dirs=/a", "--add-dirs-ro=/b", "--",
		"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", wantBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// AC-3: the --safehouse- prefix is stripped by the front-door dispatcher before
// reaching the translator — no --safehouse-* token survives into the safehouse
// extra slot or the inner argv.
func TestSafehousePrefixStrippedByDispatcher(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--safehouse-enable=docker"}, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	for _, tok := range fake.launchedArg {
		if tok == "--safehouse-enable=docker" || tok == "--safehouse" {
			t.Fatalf("a --safehouse* token survived into the argv: %v", fake.launchedArg)
		}
	}
}

// AC-4: explicit --safehouse forces the wrap in a no-profile dir (claude); the
// bare token is consumed and never reaches claude.
func TestClaudeForceSafehouseWrapsNoProfile(t *testing.T) {
	dir := t.TempDir() // no .safehouse
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--safehouse"}, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"safehouse", "--trust-workdir-config", "--",
		"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", wantBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// AC-5: explicit --safehouse forces the wrap in a no-profile dir (codex); the
// bypass flag appears only inside the forced wrap; the bare token never reaches
// codex.
func TestCodexForceSafehouseWrapsNoProfile(t *testing.T) {
	dir := t.TempDir() // no .safehouse
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runCodex(context.Background(), []string{"--safehouse"}, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"safehouse", "--trust-workdir-config", "--",
		"codex", "--dangerously-bypass-approvals-and-sandbox", wantCodexBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// AC-6: a single --safehouse-* knob implies sandbox-on in a no-profile dir for
// BOTH claude and codex (the reversal of the old fail-fast: a knob never lands
// on the plain path).
func TestKnobImpliesSandboxOnNoProfile(t *testing.T) {
	t.Run("claude", func(t *testing.T) {
		dir := t.TempDir()
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runClaude(context.Background(), []string{"--safehouse-enable=docker"}, dir, fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		want := []string{"safehouse", "--trust-workdir-config", "--enable=docker", "--",
			"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", wantBootstrapPrompt}
		if !equalArgv(fake.launchedArg, want) {
			t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
		}
	})
	t.Run("codex", func(t *testing.T) {
		dir := t.TempDir()
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runCodex(context.Background(), []string{"--safehouse-enable=docker"}, dir, fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		want := []string{"safehouse", "--trust-workdir-config", "--enable=docker", "--",
			"codex", "--dangerously-bypass-approvals-and-sandbox", wantCodexBootstrapPrompt}
		if !equalArgv(fake.launchedArg, want) {
			t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
		}
	})
}

// AC-7 codex analog: the plain (unwrapped) launch happens only when none of
// {profile, --safehouse, knob} is present — plain codex with no bypass flag.
// (The claude analog is TestClaudeNoSafehouseLaunchesPlain, unchanged.)
func TestCodexPlainWhenNoTrigger(t *testing.T) {
	dir := t.TempDir() // no .safehouse, no --safehouse, no knob
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runCodex(context.Background(), nil, dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"codex", wantCodexBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
}

// AC-8: an unknown --safehouse-<key> is a hard error (rc≠0, no Launch).
func TestUnknownSafehouseKeyErrors(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--safehouse-bogus=x"}, dir, fake, lookFound, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("exit = 0, want non-zero for an unknown --safehouse-* key")
	}
	if fake.launchedArg != nil {
		t.Fatalf("Launch invoked on an unknown --safehouse-* key: %v", fake.launchedArg)
	}
}

// LP-AC-1 (Option-2 grammar): a task positional (BEFORE any `--`) becomes
// base + " " + task as the LAST inner token; bare → base exactly; a host flag
// AFTER the `--` still forwards verbatim, with the task riding before the fence.
func TestFenceTaskPromptOverride(t *testing.T) {
	t.Run("claude-task-positional", func(t *testing.T) {
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runClaude(context.Background(), []string{"do the thing"}, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		last := fake.launchedArg[len(fake.launchedArg)-1]
		if last != wantBootstrapPrompt+" do the thing" {
			t.Fatalf("last token = %q, want base+space+task", last)
		}
	})
	t.Run("claude-bare-base-only", func(t *testing.T) {
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runClaude(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		last := fake.launchedArg[len(fake.launchedArg)-1]
		if last != wantBootstrapPrompt {
			t.Fatalf("last token = %q, want bare base prompt (no trailing space)", last)
		}
	})
	t.Run("claude-task-before-fenced-host-flag", func(t *testing.T) {
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runClaude(context.Background(), []string{"do the thing", "--", "--model", "gpt-x"}, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		want := []string{"claude", "--agent", "spacedock:first-officer", "--model", "gpt-x", wantBootstrapPrompt + " do the thing"}
		if !equalArgv(fake.launchedArg, want) {
			t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
		}
	})
	t.Run("codex-task-positional", func(t *testing.T) {
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runCodex(context.Background(), []string{"do the thing"}, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		last := fake.launchedArg[len(fake.launchedArg)-1]
		if last != wantCodexBootstrapPrompt+" do the thing" {
			t.Fatalf("last token = %q, want codexBase+space+task", last)
		}
	})
}

// LP-AC-2: codex resume subcommand suppresses the prompt; bare codex gets base.
func TestCodexResumeSubcommandSuppressesPrompt(t *testing.T) {
	t.Run("resume-no-prompt", func(t *testing.T) {
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runCodex(context.Background(), []string{"--", "resume", "abc-123"}, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		want := []string{"codex", "resume", "abc-123"}
		if !equalArgv(fake.launchedArg, want) {
			t.Fatalf("launch argv = %v, want %v (resume forwards verbatim, no prompt)", fake.launchedArg, want)
		}
		for _, tok := range fake.launchedArg {
			if tok == wantCodexBootstrapPrompt {
				t.Fatalf("codex resume carried the bootstrap prompt: %v", fake.launchedArg)
			}
		}
	})
	t.Run("bare-codex-base-prompt", func(t *testing.T) {
		fake := &fakeHost{manifest: compatibleManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runCodex(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		last := fake.launchedArg[len(fake.launchedArg)-1]
		if last != wantCodexBootstrapPrompt {
			t.Fatalf("bare codex last token = %q, want codex base prompt", last)
		}
	})
}

// LP-AC-3: --plugin-dir passes through (multiplicity, order) AND relaxes the
// gate (launches even on a failing manifest); without it the gate still fails.
func TestPluginDirRelaxesGate(t *testing.T) {
	t.Run("claude-relaxes-on-failing-manifest", func(t *testing.T) {
		fake := &fakeHost{manifest: tooOldBinaryManifest(t)} // gate would FAIL
		var stdout, stderr bytes.Buffer
		code := runClaude(context.Background(), []string{"--", "--plugin-dir", "/a", "--plugin-dir", "/b"}, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (--plugin-dir relaxes the gate); stderr=%q", code, stderr.String())
		}
		want := []string{"claude", "--agent", "spacedock:first-officer", "--plugin-dir", "/a", "--plugin-dir", "/b", wantBootstrapPrompt}
		if !equalArgv(fake.launchedArg, want) {
			t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
		}
	})
	t.Run("codex-relaxes-on-failing-manifest", func(t *testing.T) {
		fake := &fakeHost{manifest: tooOldBinaryManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runCodex(context.Background(), []string{"--", "--plugin-dir", "/a"}, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (--plugin-dir relaxes the gate); stderr=%q", code, stderr.String())
		}
	})
	t.Run("no-plugin-dir-still-fails-fast", func(t *testing.T) {
		fake := &fakeHost{manifest: tooOldBinaryManifest(t)}
		var stdout, stderr bytes.Buffer
		code := runClaude(context.Background(), nil, t.TempDir(), fake, lookFound, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("exit = 0, want non-zero with a failing manifest and no --plugin-dir")
		}
		if fake.launchedArg != nil {
			t.Fatalf("Launch invoked despite failing gate and no --plugin-dir: %v", fake.launchedArg)
		}
	})
}
