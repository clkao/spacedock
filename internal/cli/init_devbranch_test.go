// ABOUTME: AC-3a — `spacedock init --host claude` targets @next when devBranch is
// ABOUTME: pinned to next, and the composed marketplace-add argv carries @next.
package cli

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

// TestInitTargetsNextWhenDevBranchPinned locks AC-3a: with devBranch pinned to
// `next` (the released binary's default, until `next` is the default branch),
// `spacedock init --host claude` drives the install seam with branch=next, so the
// issued `marketplace add` resolves `spacedock-dev/spacedock@next`. The package
// var is saved/restored so the assertion does not leak into sibling tests.
func TestInitTargetsNextWhenDevBranchPinned(t *testing.T) {
	saved := devBranch
	devBranch = "next"
	defer func() { devBranch = saved }()

	fake := &fakeHost{manifest: compatibleManifest(t)}
	var stdout, stderr bytes.Buffer

	code := runInit(context.Background(), []string{"--host", "claude"}, fake, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	// Install records {host, source, branch}; branch carries the @ref pin.
	if len(fake.installCmds) < 3 {
		t.Fatalf("install seam recorded %v, want {host, source, branch}", fake.installCmds)
	}
	if got := fake.installCmds[2]; got != "next" {
		t.Fatalf("install branch = %q, want next (init must target @next)", got)
	}
}

// TestMarketplaceAddArgvCarriesRef locks the argv composition AC-3a asserts: the
// `claude plugin marketplace add` argv pins `source@branch` when a branch is set,
// and is the bare source when it is not. This is the exact 2-command argv shape
// owned today; task 38 changes Install to a 3-command shape (add/uninstall/
// install) and this assertion is updated in lockstep then.
func TestMarketplaceAddArgvCarriesRef(t *testing.T) {
	if got := marketplaceAddArg("spacedock-dev/spacedock", "next"); got != "spacedock-dev/spacedock@next" {
		t.Errorf("marketplaceAddArg with branch = %q, want spacedock-dev/spacedock@next", got)
	}
	if got := marketplaceAddArg("spacedock-dev/spacedock", ""); got != "spacedock-dev/spacedock" {
		t.Errorf("marketplaceAddArg without branch = %q, want bare source", got)
	}
}

// TestInstallArgvSequence locks AC-2: execHost.Install issues the 3-command
// upgrade shape — marketplace add (with the @ref pin), then `plugin uninstall
// spacedock@spacedock`, then `plugin install spacedock@spacedock`, in that order.
// The uninstall step is what moves a stale already-installed plugin off (plain
// install no-ops on it); uninstall is itself a no-op on a fresh box. With an
// empty branch the marketplace arg is the bare source.
func TestInstallArgvSequence(t *testing.T) {
	wantWithBranch := [][]string{
		{"plugin", "marketplace", "add", "spacedock-dev/spacedock@next"},
		{"plugin", "uninstall", "spacedock@spacedock"},
		{"plugin", "install", "spacedock@spacedock"},
	}
	if got := installArgvSequence("spacedock-dev/spacedock", "next"); !reflect.DeepEqual(got, wantWithBranch) {
		t.Errorf("installArgvSequence(branch=next) = %v, want %v", got, wantWithBranch)
	}

	wantBareSource := [][]string{
		{"plugin", "marketplace", "add", "spacedock-dev/spacedock"},
		{"plugin", "uninstall", "spacedock@spacedock"},
		{"plugin", "install", "spacedock@spacedock"},
	}
	if got := installArgvSequence("spacedock-dev/spacedock", ""); !reflect.DeepEqual(got, wantBareSource) {
		t.Errorf("installArgvSequence(no branch) = %v, want %v", got, wantBareSource)
	}
}
