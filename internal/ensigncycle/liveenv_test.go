// ABOUTME: Auth + HOME-isolation decision tree for the live cycle test, ported
// ABOUTME: from the proven Python _isolated_claude_env (offline-testable helper).
package ensigncycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// claudeAuthMode names which credential the live cycle authenticates with.
type claudeAuthMode int

const (
	// authNone means no credential is available; the live test must skip.
	authNone claudeAuthMode = iota
	// authOAuthToken means the operator-local OAuth path: a non-empty
	// ~/.claude/benchmark-token authenticates via CLAUDE_CODE_OAUTH_TOKEN and
	// ANTHROPIC_API_KEY is dropped so the token is authoritative.
	authOAuthToken
	// authAPIKey means the CI path: no token file but ANTHROPIC_API_KEY is set,
	// passed through to the child.
	authAPIKey
)

// claudeEnvDecision is the pure result of the auth + HOME-isolation decision
// tree. It is computed from explicit inputs (no live $HOME / no temp dir) so the
// offline unit test can assert on it without spending a model. isolatedClaudeEnv
// turns it into the concrete child env (a fresh HOME tmpdir + the credential) or
// a skip.
type claudeEnvDecision struct {
	mode claudeAuthMode
	// oauthToken is the trimmed benchmark-token contents when mode is
	// authOAuthToken, else empty.
	oauthToken string
}

// decideClaudeEnv is the port of the Python _isolated_claude_env decision tree
// (scripts/test_lib.py:319-375), factored pure for offline testability:
//
//	(a) realHome/.claude/benchmark-token non-empty -> authOAuthToken (inject
//	    CLAUDE_CODE_OAUTH_TOKEN, drop ANTHROPIC_API_KEY) — operator-local.
//	(b) no token but apiKey set -> authAPIKey (pass ANTHROPIC_API_KEY through) — CI.
//	(c) neither -> authNone (caller skips, does NOT fatal).
//
// A locked-down or sandboxed ~/.claude that errors on read is treated as "no
// token" (the original catches OSError there) so offline/static callers do not
// crash. realHome == "" likewise yields no token.
func decideClaudeEnv(realHome, apiKey string) claudeEnvDecision {
	if realHome != "" {
		tokenPath := filepath.Join(realHome, ".claude", "benchmark-token")
		// An unreadable token path (PermissionError on a locked-down or
		// sandboxed HOME) is treated as absent, matching the original's
		// try/except OSError.
		if b, err := os.ReadFile(tokenPath); err == nil {
			if token := strings.TrimSpace(string(b)); token != "" {
				return claudeEnvDecision{mode: authOAuthToken, oauthToken: token}
			}
		}
	}
	if apiKey != "" {
		return claudeEnvDecision{mode: authAPIKey}
	}
	return claudeEnvDecision{mode: authNone}
}

// cleanEnviron returns os.Environ() filtered to drop the keys in drop. It is the
// port of the Python _clean_env (strip CLAUDECODE so the child can launch
// claude); the live path also drops/overrides HOME and the credential keys.
func cleanEnviron(drop ...string) []string {
	dropped := make(map[string]bool, len(drop))
	for _, k := range drop {
		dropped[k] = true
	}
	var env []string
	for _, kv := range os.Environ() {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if dropped[key] {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// isolatedClaudeEnv resolves the child environment for the live `spacedock
// claude` launch: a fresh empty HOME (so parallel invocations never collide in
// ~/.claude) plus the authoritative credential per decideClaudeEnv. When no
// credential is available it t.Skips (never fatals), matching the original's
// path (c). realHome is the home directory to probe for the benchmark-token;
// pass os.Getenv("HOME") for the live path. CLAUDECODE is always dropped so the
// child binary takes the real front-door path rather than a nested-session
// shortcut.
func isolatedClaudeEnv(t *testing.T, realHome string) []string {
	t.Helper()
	decision := decideClaudeEnv(realHome, os.Getenv("ANTHROPIC_API_KEY"))
	if decision.mode == authNone {
		t.Skip("no live auth available: set ~/.claude/benchmark-token " +
			"(operator/OAuth) or ANTHROPIC_API_KEY (CI) to run the live cycle")
	}

	cleanHome := t.TempDir()
	switch decision.mode {
	case authOAuthToken:
		// Operator-local: drop the API key so the OAuth token is authoritative.
		env := cleanEnviron("CLAUDECODE", "HOME", "ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN")
		env = append(env, "HOME="+cleanHome, "CLAUDE_CODE_OAUTH_TOKEN="+decision.oauthToken)
		return env
	case authAPIKey:
		// CI: pass ANTHROPIC_API_KEY through against the fresh HOME.
		env := cleanEnviron("CLAUDECODE", "HOME")
		env = append(env, "HOME="+cleanHome)
		return env
	default:
		t.Fatalf("unreachable auth mode %d", decision.mode)
		return nil
	}
}

// envOr returns the environment value for key, or def when unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// tail returns the last n bytes of s, prefixed with an elision marker when s was
// truncated, so the transcript log stays bounded.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…(truncated)…\n" + s[len(s)-n:]
}
