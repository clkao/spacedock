// ABOUTME: claudeteam unit tests — the AC-4 1M family-rule boundary table and the
// ABOUTME: context-budget envelope rendering, exercised directly in-package.
package claudeteam

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestContextLimitForModelBoundary is the AC-4 boundary table: the forward opus
// family rule (minor >= 7 -> 1M) plus the [1m] suffix override, exercised at the
// version boundary so a future opus release stays correct without a code change.
func TestContextLimitForModelBoundary(t *testing.T) {
	cases := []struct {
		model string
		want  int
	}{
		{"claude-opus-4-8", extendedContextLimit},     // the live false-negative, now 1M
		{"claude-opus-4-8[1m]", extendedContextLimit}, // explicit suffix
		{"claude-opus-4-7", extendedContextLimit},     // first 1M-default minor
		{"claude-opus-4-6", defaultContextLimit},      // pre-default minor stays 200k
		{"claude-opus-4-6[1m]", extendedContextLimit}, // 4-6 with the suffix opts in
		{"claude-opus-4-10", extendedContextLimit},    // forward-safe: never goes stale
		{"claude-opus-4-100", extendedContextLimit},   // multi-digit minor
		{"claude-sonnet-4-6", defaultContextLimit},    // non-opus
		{"claude-haiku-4-5", defaultContextLimit},     // non-opus
		{"some-unknown-model", defaultContextLimit},   // safe fallback
		{"claude-opus-4", defaultContextLimit},        // no minor token -> no match
	}
	for _, tc := range cases {
		if got := contextLimitForModel(tc.model); got != tc.want {
			t.Errorf("contextLimitForModel(%q) = %d, want %d", tc.model, got, tc.want)
		}
	}
}

// TestContextBudgetEnvelopeWholePercent asserts a whole-number usage_pct renders
// with the trailing .0 Python's json.dumps emits (20.0, not 20) — the pyFloat
// rendering the parity harness depends on, exercised directly without a fixture.
func TestContextBudgetEnvelopeWholePercent(t *testing.T) {
	home := t.TempDir()
	// 40k resident on a 200k opus-4-6 member -> exactly 20.0%.
	writeBudgetFixture(t, home, "ensign-w", "claude-opus-4-6", 40000)

	var stdout, stderr bytes.Buffer
	if code := ContextBudget(home, "ensign-w", &stdout, &stderr); code != 0 {
		t.Fatalf("ContextBudget exit=%d stderr=%q", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"usage_pct": 20.0`)) {
		t.Errorf("usage_pct did not render as 20.0 (Python float repr):\n%s", stdout.String())
	}
}

// TestMemberExistsTeamScoped asserts MemberExists is team-scoped: a member in one
// team's config is found there and not via a sibling team's config.
func TestMemberExistsTeamScoped(t *testing.T) {
	home := t.TempDir()
	writeTeamConfig(t, home, "team-a", map[string]string{"comm-officer": "sonnet"})
	writeTeamConfig(t, home, "team-b", map[string]string{"team-lead": "opus"})

	if !MemberExists(home, "team-a", "comm-officer") {
		t.Errorf("MemberExists should find comm-officer in team-a")
	}
	if MemberExists(home, "team-b", "comm-officer") {
		t.Errorf("MemberExists must not find comm-officer in team-b (team-scoped)")
	}
	if MemberExists(home, "team-missing", "anyone") {
		t.Errorf("MemberExists must return false for a missing team config")
	}
}

// writeBudgetFixture writes a minimal ~/.claude tree: a team config listing the
// member with model, and a one-line transcript whose resident equals tokens.
func writeBudgetFixture(t *testing.T, home, name, model string, tokens int) {
	t.Helper()
	writeTeamConfigWithSession(t, home, "fixture-team", "sess", map[string]string{name: model})
	entry := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"model": model,
			"usage": map[string]any{
				"input_tokens":                tokens,
				"cache_creation_input_tokens": 0,
				"cache_read_input_tokens":     0,
			},
		},
	}
	line, _ := json.Marshal(entry)
	subagents := filepath.Join(home, ".claude", "projects", "p", "sess", "subagents")
	writeTestFile(t, filepath.Join(subagents, "agent-"+name+".meta.json"),
		`{"agentType": "`+name+`"}`)
	writeTestFile(t, filepath.Join(subagents, "agent-"+name+".jsonl"), string(line)+"\n")
}

// writeTeamConfig writes a team config.json (no leadSessionId) listing the given
// name->model members.
func writeTeamConfig(t *testing.T, home, team string, members map[string]string) {
	t.Helper()
	writeTeamConfigWithSession(t, home, team, "", members)
}

// writeTeamConfigWithSession writes a team config.json with an optional
// leadSessionId and the given name->model members.
func writeTeamConfigWithSession(t *testing.T, home, team, session string, members map[string]string) {
	t.Helper()
	cfg := map[string]any{}
	if session != "" {
		cfg["leadSessionId"] = session
	}
	var ms []map[string]string
	for name, model := range members {
		ms = append(ms, map[string]string{"name": name, "model": model})
	}
	cfg["members"] = ms
	b, _ := json.MarshalIndent(cfg, "", "  ")
	writeTestFile(t, filepath.Join(home, ".claude", "teams", team, "config.json"), string(b))
}

// writeTestFile writes content to path, creating parent dirs.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
