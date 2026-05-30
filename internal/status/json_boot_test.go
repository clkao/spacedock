// ABOUTME: AC-1 --boot --json structural+normalized parity — section keys present
// ABOUTME: and ordered, deterministic bodies pinned, volatile material range-checked.
package status

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// bootJSONKeys are the --boot --json top-level keys in their required order. The
// FO parses --boot by key at startup, so order and presence are load-bearing.
var bootJSONKeys = []string{
	"command", "mods", "id_style", "next_id", "min_prefix",
	"orphans", "pr_state", "dispatchable", "team_state",
}

// TestBootJSONStructure (AC-1 oracle e) mirrors nextid_boot_test.go for the JSON
// form: it asserts the key order, the deterministic section bodies, and the
// range of the volatile material (next_id alphabet, team_state.present,
// pr_state.status). --boot is non-deterministic by construction so it is NOT
// byte-compared.
func TestBootJSONStructure(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "sdb32-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)
	out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--boot", "--json")
	if code != 0 {
		t.Fatalf("native --boot --json exit=%d stderr=%q", code, errOut)
	}

	// Key order: each key appears after the previous one in the raw bytes.
	last := -1
	for _, key := range bootJSONKeys {
		idx := strings.Index(out, `"`+key+`"`)
		if idx < 0 {
			t.Fatalf("--boot --json missing key %q\n%s", key, out)
		}
		if idx < last {
			t.Fatalf("--boot --json key %q out of order\n%s", key, out)
		}
		last = idx
	}

	// Glyph-free single document.
	if strings.Contains(out, "  ") || strings.Contains(out, "…") {
		t.Fatalf("--boot --json carries padding/glyph: %q", out)
	}
	if !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 1 {
		t.Fatalf("--boot --json not a single newline-terminated document: %q", out)
	}

	var boot struct {
		Command   string              `json:"command"`
		Mods      map[string][]string `json:"mods"`
		IDStyle   string              `json:"id_style"`
		NextID    string              `json:"next_id"`
		MinPrefix string              `json:"min_prefix"`
		Orphans   []map[string]string `json:"orphans"`
		PRState   struct {
			Status  string              `json:"status"`
			Entries []map[string]string `json:"entries"`
		} `json:"pr_state"`
		Dispatchable []map[string]string `json:"dispatchable"`
		TeamState    struct {
			Present string `json:"present"`
			Hint    string `json:"hint"`
		} `json:"team_state"`
	}
	if err := json.Unmarshal([]byte(out), &boot); err != nil {
		t.Fatalf("parse --boot --json: %v\n%s", err, out)
	}

	// Deterministic sections (env-independent on this fixture).
	if boot.Command != "boot" {
		t.Fatalf("command = %q, want boot", boot.Command)
	}
	if boot.IDStyle != "sd-b32" {
		t.Fatalf("id_style = %q, want sd-b32", boot.IDStyle)
	}
	if boot.MinPrefix != "2" {
		t.Fatalf("min_prefix = %q, want \"2\" (string)", boot.MinPrefix)
	}
	if len(boot.Orphans) != 0 {
		t.Fatalf("orphans = %v, want empty []", boot.Orphans)
	}
	if boot.PRState.Status != "none" {
		t.Fatalf("pr_state.status = %q, want none (fixture has no PRs)", boot.PRState.Status)
	}

	// Volatile material: range-checked, not byte-pinned.
	if len(boot.NextID) != 24 {
		t.Fatalf("next_id %q length=%d, want 24", boot.NextID, len(boot.NextID))
	}
	for _, c := range boot.NextID {
		if !strings.ContainsRune(sdB32Alphabet, c) {
			t.Fatalf("next_id %q has char %q outside SD-B32 alphabet", boot.NextID, c)
		}
	}
	if boot.TeamState.Present != "true" && boot.TeamState.Present != "false" {
		t.Fatalf("team_state.present = %q, want \"true\" or \"false\" (string)", boot.TeamState.Present)
	}
	switch boot.PRState.Status {
	case "ok", "none", "gh not available":
	default:
		t.Fatalf("pr_state.status = %q, want one of ok|none|gh not available", boot.PRState.Status)
	}
}

// TestBootJSONDispatchableMirrorsNext (AC-1) locks the design's claim that
// --boot --json's dispatchable rows share the --next element shape (the fixed
// five keys), so a consumer parses both with one struct.
func TestBootJSONDispatchableMirrorsNext(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	bootOut, _, bc := runNative(t, root, env, "--workflow-dir", root, "--boot", "--json")
	if bc != 0 {
		t.Fatalf("--boot --json exit=%d", bc)
	}
	nextOut, _, nc := runNative(t, root, env, "--workflow-dir", root, "--next", "--json")
	if nc != 0 {
		t.Fatalf("--next --json exit=%d", nc)
	}

	var boot struct {
		Dispatchable []map[string]string `json:"dispatchable"`
	}
	var next struct {
		Dispatchable []map[string]string `json:"dispatchable"`
	}
	if err := json.Unmarshal([]byte(bootOut), &boot); err != nil {
		t.Fatalf("parse boot: %v", err)
	}
	if err := json.Unmarshal([]byte(nextOut), &next); err != nil {
		t.Fatalf("parse next: %v", err)
	}
	if len(boot.Dispatchable) != len(next.Dispatchable) {
		t.Fatalf("dispatchable lengths differ: boot=%d next=%d", len(boot.Dispatchable), len(next.Dispatchable))
	}
	for i := range boot.Dispatchable {
		for _, k := range nextFixedFields {
			if boot.Dispatchable[i][k] != next.Dispatchable[i][k] {
				t.Fatalf("dispatchable[%d].%s differs: boot=%q next=%q", i, k, boot.Dispatchable[i][k], next.Dispatchable[i][k])
			}
		}
	}
}
