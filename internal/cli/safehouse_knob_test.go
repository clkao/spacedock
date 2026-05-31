// ABOUTME: AC-6 sandbox-knob parsing — space/equals/repeat forms produce the same
// ABOUTME: safehouse extra argv, the reported space-form bug leaks nothing, bad value errors.
package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spacedock-dev/spacedock/internal/safehouse"
)

// TestSafehouseKnobFormsEquivalent pins AC-6: each value-taking knob accepts the
// space form, the equals form, and repeats — all producing the same safehouse
// `extra` argv. The equivalence is asserted through the full chain
// parseFrontDoorArgs → safehouse.TranslateFlags, which is what the launcher feeds.
func TestSafehouseKnobFormsEquivalent(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantExtra []string
	}{
		{"enable-equals", []string{"--safehouse-enable=docker"}, []string{"--enable=docker"}},
		{"enable-space", []string{"--safehouse-enable", "docker"}, []string{"--enable=docker"}},
		{"enable-comma-split", []string{"--safehouse-enable=ssh,docker"}, []string{"--enable=ssh", "--enable=docker"}},
		{"enable-repeat-accumulates", []string{"--safehouse-enable", "ssh", "--safehouse-enable", "docker"}, []string{"--enable=ssh", "--enable=docker"}},
		{"add-dirs-equals", []string{"--safehouse-add-dirs=/a"}, []string{"--add-dirs=/a"}},
		{"add-dirs-space", []string{"--safehouse-add-dirs", "/a"}, []string{"--add-dirs=/a"}},
		{"add-dirs-repeat", []string{"--safehouse-add-dirs", "/a", "--safehouse-add-dirs", "/b"}, []string{"--add-dirs=/a", "--add-dirs=/b"}},
		{"add-dirs-ro-space", []string{"--safehouse-add-dirs-ro", "/c"}, []string{"--add-dirs-ro=/c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fd, err := parseFrontDoorArgs(tc.args)
			if err != nil {
				t.Fatalf("parseFrontDoorArgs(%v) err = %v", tc.args, err)
			}
			extra, err := safehouse.TranslateFlags(fd.safehouseFlags)
			if err != nil {
				t.Fatalf("TranslateFlags(%v) err = %v", fd.safehouseFlags, err)
			}
			if !equalArgv(extra, tc.wantExtra) {
				t.Fatalf("extra = %v, want %v", extra, tc.wantExtra)
			}
		})
	}
}

// TestSafehouseAddDirsSpaceFormNoLeak is the regression for the captain-reported
// bug: `--safehouse-add-dirs ~/a --safehouse-add-dirs ~/b` (space form) used to
// fail with `malformed flag` and leak the paths into host passthrough. In the end
// state cobra captures each value into two `--add-dirs=` entries in the safehouse
// extra slot, and the paths appear NOWHERE in the host argv (they are not host
// passthrough). Driven through runClaude to the recorded launch.
func TestSafehouseAddDirsSpaceFormNoLeak(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(),
		[]string{"--safehouse-add-dirs", "/home/a", "--safehouse-add-dirs", "/home/b"},
		dir, fake, lookFound, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	want := []string{"safehouse", "--trust-workdir-config", "--add-dirs=/home/a", "--add-dirs=/home/b", "--",
		"claude", "--dangerously-skip-permissions", "--agent", "spacedock:first-officer", wantBootstrapPrompt}
	if !equalArgv(fake.launchedArg, want) {
		t.Fatalf("launch argv = %v, want %v", fake.launchedArg, want)
	}
	// The paths ride in the safehouse extra slot (before the inner `--`), never as
	// host passthrough adjacent to the inner claude argv.
	dash := -1
	for i, tok := range fake.launchedArg {
		if tok == "--" {
			dash = i
			break
		}
	}
	if dash < 0 {
		t.Fatalf("no safehouse `--` separator in argv: %v", fake.launchedArg)
	}
	for _, tok := range fake.launchedArg[dash:] {
		if tok == "/home/a" || tok == "/home/b" || strings.HasPrefix(tok, "--safehouse-add-dirs") {
			t.Fatalf("a knob path/token leaked past the safehouse `--` into host passthrough: %v", fake.launchedArg)
		}
	}
}

// TestSafehouseBadValueNamesKnob pins AC-6's clear-error end state: a genuinely
// bad knob value surfaces a knob-named error, not the internal `malformed flag`
// text. An unknown safehouse key is the representative bad value; the error names
// the knob (`--safehouse-bogus`) and Launch is never reached.
func TestSafehouseBadValueNamesKnob(t *testing.T) {
	dir := safehouseFixtureDir(t)
	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runClaude(context.Background(), []string{"--safehouse-bogus=x"}, dir, fake, lookFound, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("exit = 0, want non-zero for a bad knob value")
	}
	if fake.launchedArg != nil {
		t.Fatalf("Launch invoked on a bad knob value: %v", fake.launchedArg)
	}
	if !strings.Contains(stderr.String(), "--safehouse-bogus") {
		t.Fatalf("error does not name the knob --safehouse-bogus: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "malformed flag") {
		t.Fatalf("error leaked the internal malformed-flag text: %q", stderr.String())
	}
}
