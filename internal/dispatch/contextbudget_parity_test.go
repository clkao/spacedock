// ABOUTME: context-budget three-channel parity — native vs vendored Python over a
// ABOUTME: fixture ~/.claude tree, success path + AC-4 1M rule + the three exit-1 paths.
package dispatch

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// claudeFixture builds a fixture ~/.claude tree under home for one team session:
// teams/{team}/config.json (members with their declared models + a leadSessionId)
// and projects/p/{session}/subagents/agent-{member}.meta.json + .jsonl for each
// member that has a jsonl spec. It mirrors the real on-disk shape the
// context-budget reads. jsonls maps member name -> raw jsonl content; a member in
// members but absent from jsonls gets a config entry with no transcript.
type claudeFixture struct {
	team      string
	session   string
	members   []fixtureMember
	jsonls    map[string]string
	noSession bool // omit leadSessionId so the narrowed scan falls back to broad
}

type fixtureMember struct {
	name  string
	model string
}

// write materializes the fixture under home and returns the project session dir.
func (f claudeFixture) write(t *testing.T, home string) {
	t.Helper()
	// teams/{team}/config.json
	cfg := map[string]any{}
	if !f.noSession {
		cfg["leadSessionId"] = f.session
	}
	var members []map[string]string
	for _, m := range f.members {
		members = append(members, map[string]string{"name": m.name, "model": m.model})
	}
	cfg["members"] = members
	cfgBytes, _ := json.MarshalIndent(cfg, "", "  ")
	writeFile(t, filepath.Join(home, ".claude", "teams", f.team, "config.json"), string(cfgBytes))

	// projects/p/{session}/subagents/agent-{name}.{meta.json,jsonl}
	subagents := filepath.Join(home, ".claude", "projects", "proj-fixture", f.session, "subagents")
	for name, jsonl := range f.jsonls {
		meta := fmt.Sprintf(`{"agentType": %q}`, name)
		writeFile(t, filepath.Join(subagents, "agent-"+name+".meta.json"), meta)
		writeFile(t, filepath.Join(subagents, "agent-"+name+".jsonl"), jsonl)
	}
}

// assistantLine builds one assistant jsonl entry with the given model and usage
// sum split across the three usage fields (the resident extractor sums them).
func assistantLine(model string, input, cacheCreation, cacheRead int) string {
	entry := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": model,
			"usage": map[string]any{
				"input_tokens":                input,
				"cache_creation_input_tokens": cacheCreation,
				"cache_read_input_tokens":     cacheRead,
			},
		},
	}
	b, _ := json.Marshal(entry)
	return string(b)
}

// runBudget runs both native and oracle context-budget for name over the fixture
// home and asserts three-channel parity (no fetch rewrite — context-budget emits
// no fetch lines). It returns the native result for follow-on field assertions.
func runBudget(t *testing.T, home, name string) runResult {
	t.Helper()
	oracle := runOracle(t, home, home, "", "context-budget", "--name", name)
	native := runNative("", "context-budget", "--name", name)
	assertParity(t, "context-budget "+name, native, oracle)
	return native
}

// TestContextBudgetParitySuccess drives the success path: a sub-threshold ensign
// reads reuse_ok true at parity, on both an opus-4-8 (1M family rule) and an
// opus-4-6 (200k) member, so the AC-4 boundary is exercised through the real
// envelope, not just the unit table.
func TestContextBudgetParitySuccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fix := claudeFixture{
		team:    "fixture-team",
		session: "sess-abc",
		members: []fixtureMember{
			{name: "ensign-a", model: "claude-opus-4-8"},
			{name: "ensign-b", model: "claude-opus-4-6"},
		},
		jsonls: map[string]string{
			// 159k resident on opus-4-8 → 15.9% of 1M (family rule) → reuse_ok.
			"ensign-a": assistantLine("claude-opus-4-8", 100000, 30000, 29000) + "\n",
			// 80k resident on opus-4-6 → 40% of 200k → reuse_ok (just under 60).
			"ensign-b": assistantLine("claude-opus-4-6", 50000, 20000, 10000) + "\n",
		},
	}
	fix.write(t, home)

	a := runBudget(t, home, "ensign-a")
	assertBudgetField(t, a.stdout, "context_limit", float64(1000000))
	assertBudgetField(t, a.stdout, "reuse_ok", true)

	b := runBudget(t, home, "ensign-b")
	assertBudgetField(t, b.stdout, "context_limit", float64(200000))
	assertBudgetField(t, b.stdout, "reuse_ok", true)
}

// TestContextBudget1MFalseNegativeGone is the AC-4 behavioral oracle: an
// opus-4-8 ensign (no [1m] suffix, as the real team config stamps it) at a
// sub-600k resident count reads context_limit 1_000_000 and reuse_ok true — the
// exact false-negative the pre-fix exact-name list produced (200k denominator).
func TestContextBudget1MFalseNegativeGone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fix := claudeFixture{
		team:    "fixture-team",
		session: "sess-1m",
		members: []fixtureMember{{name: "ensign-1m", model: "claude-opus-4-8"}},
		jsonls: map[string]string{
			// 159k resident: 79.5% of 200k (the bug) but 15.9% of 1M (the fix).
			"ensign-1m": assistantLine("claude-opus-4-8", 100000, 30000, 29000) + "\n",
		},
	}
	fix.write(t, home)

	res := runBudget(t, home, "ensign-1m")
	assertBudgetField(t, res.stdout, "context_limit", float64(1000000))
	assertBudgetField(t, res.stdout, "usage_pct", 15.9)
	assertBudgetField(t, res.stdout, "reuse_ok", true)
}

// TestContextBudgetLoudFailures drives the three exit-1 paths the FO treats as
// fail-safe (fresh-dispatch): missing jsonl, usage-free jsonl, and
// agent-not-in-team-config. Each is byte-compared native vs oracle and asserted
// to exit non-zero with no reuse_ok in stdout (AC-2's observable contract).
func TestContextBudgetLoudFailures(t *testing.T) {
	t.Run("missing-jsonl", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// Member in config but no subagent jsonl for it.
		fix := claudeFixture{
			team:    "fixture-team",
			session: "sess-x",
			members: []fixtureMember{{name: "ensign-x", model: "claude-opus-4-8"}},
			jsonls:  map[string]string{},
		}
		fix.write(t, home)
		res := runBudget(t, home, "ensign-x")
		assertFailSafe(t, res, "no subagent jsonl found")
	})

	t.Run("usage-free-jsonl", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// jsonl exists but every assistant entry has all-zero usage (overflow-dead),
		// so extract_resident_tokens yields nothing.
		fix := claudeFixture{
			team:    "fixture-team",
			session: "sess-z",
			members: []fixtureMember{{name: "ensign-z", model: "claude-opus-4-8"}},
			jsonls: map[string]string{
				"ensign-z": assistantLine("claude-opus-4-8", 0, 0, 0) + "\n",
			},
		}
		fix.write(t, home)
		res := runBudget(t, home, "ensign-z")
		assertFailSafe(t, res, "no assistant entries with usage")
	})

	t.Run("agent-not-in-team-config", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// A jsonl exists for the named agent (so resident extraction succeeds) but
		// no team config lists it, so lookup_model returns nothing. The jsonl is
		// reachable via the broad scan because no team config declares the member.
		subagents := filepath.Join(home, ".claude", "projects", "proj-fixture", "subagents")
		writeFile(t, filepath.Join(subagents, "agent-orphan.meta.json"), `{"agentType": "orphan"}`)
		writeFile(t, filepath.Join(subagents, "agent-orphan.jsonl"),
			assistantLine("claude-opus-4-8", 1000, 0, 0)+"\n")
		res := runBudget(t, home, "orphan")
		assertFailSafe(t, res, "no team config found for member")
	})
}

// assertBudgetField decodes the context-budget stdout and asserts one field.
func assertBudgetField(t *testing.T, stdout, key string, want any) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(stdout), &m); err != nil {
		t.Fatalf("context-budget stdout not JSON: %v\n%s", err, stdout)
	}
	got, ok := m[key]
	if !ok {
		t.Fatalf("context-budget stdout missing %q:\n%s", key, stdout)
	}
	if got != want {
		t.Errorf("context-budget %q = %v (%T), want %v (%T)", key, got, got, want, want)
	}
}

// assertFailSafe asserts an exit-1 budget run is fail-safe: non-zero exit, the
// oracle's error fragment on stderr, and NO reuse_ok in stdout (so the FO can
// never silent-reuse). The native/oracle byte parity is already asserted by the
// caller's runBudget.
func assertFailSafe(t *testing.T, res runResult, wantStderrFragment string) {
	t.Helper()
	if res.exit == 0 {
		t.Errorf("budget-unavailable path exited 0 (must be non-zero so the FO fresh-dispatches)")
	}
	if !strings.Contains(res.stderr, wantStderrFragment) {
		t.Errorf("stderr does not name the failure %q:\n%q", wantStderrFragment, res.stderr)
	}
	if strings.Contains(res.stdout, "reuse_ok") {
		t.Errorf("budget-unavailable path emitted reuse_ok on stdout (must be silent so no silent-reuse):\n%q", res.stdout)
	}
}
