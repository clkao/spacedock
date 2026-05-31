// ABOUTME: stdin-JSON typed accessors, path joins, and the team/split-root
// ABOUTME: probes build needs, matching the oracle's helper semantics.
package dispatch

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spacedock-dev/spacedock/internal/status"
)

// teamEvidenceWindow is the lookback for the bare-mode TeamCreate-evidence probe.
const teamEvidenceWindow = 30 * time.Minute

// isJSONNull reports whether a raw JSON value is the literal null.
func isJSONNull(v json.RawMessage) bool {
	return string(bytes.TrimSpace(v)) == "null"
}

// isSchemaVersion reports whether the raw value equals schema version 2 the way
// the oracle's `sv != SCHEMA_VERSION` does after json.loads: a numeric
// comparison. JSON 2 and 2.0 both accept (Python `2.0 != 2` is False); a string
// "2", a bool, or null reject (Python `"2" != 2` is True). Only a JSON number
// whose value is exactly 2 is accepted.
func isSchemaVersion(v json.RawMessage) bool {
	var val interface{}
	dec := json.NewDecoder(bytes.NewReader(v))
	dec.UseNumber()
	if err := dec.Decode(&val); err != nil {
		return false
	}
	n, ok := val.(json.Number)
	if !ok {
		return false
	}
	f, err := n.Float64()
	return err == nil && f == 2
}

// renderSchemaVersion renders the parsed schema_version the way Python's f-string
// renders it in the unsupported-version error: a JSON number prints bare (2.0
// stays 2.0), a string prints its contents, a bool/null print Python-style.
func renderSchemaVersion(v json.RawMessage) string {
	var val interface{}
	dec := json.NewDecoder(bytes.NewReader(v))
	dec.UseNumber()
	if err := dec.Decode(&val); err != nil {
		return strings.TrimSpace(string(v))
	}
	switch t := val.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	default:
		return strings.TrimSpace(string(v))
	}
}

// jsonString decodes a raw JSON string value, returning "" for a non-string.
func jsonString(v json.RawMessage) string {
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	return ""
}

// optString returns the string value of an optional field, "" when absent, null,
// or non-string — matching inp.get(field) used as a truthy string in the oracle.
func optString(fields map[string]json.RawMessage, key string) string {
	v, ok := fields[key]
	if !ok || isJSONNull(v) {
		return ""
	}
	return jsonString(v)
}

// optBool returns the bool value of an optional field, false when absent, null,
// or non-bool — matching inp.get(field, False).
func optBool(fields map[string]json.RawMessage, key string) bool {
	v, ok := fields[key]
	if !ok || isJSONNull(v) {
		return false
	}
	var b bool
	if json.Unmarshal(v, &b) == nil {
		return b
	}
	return false
}

// jsonStringList decodes a raw JSON array of strings. ok is false when the value
// is not a JSON array (so the caller collapses both empty and non-list to the
// "checklist must not be empty" message, as the oracle does).
func jsonStringList(v json.RawMessage) ([]string, bool) {
	var list []string
	if json.Unmarshal(v, &list) != nil {
		return nil, false
	}
	return list, true
}

// isFile reports whether path is an existing regular file (os.path.isfile).
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// workflowIsSplitRoot reports whether the workflow README declares a state:
// checkout. Matches workflow_is_split_root: false when the README is unreadable
// or carries no non-empty state: field.
func workflowIsSplitRoot(workflowDir string) bool {
	readmePath := filepath.Join(workflowDir, "README.md")
	fm := status.ParseFrontmatter(readmePath)
	return strings.TrimSpace(fm["state"]) != ""
}

// recentTeamEvidence reports whether any ~/.claude/teams/*/config.json was
// modified within teamEvidenceWindow. Matches _recent_team_evidence: the
// cheapest plausible proxy for "this session ran TeamCreate".
func recentTeamEvidence() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	teamsDir := filepath.Join(home, ".claude", "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		return false
	}
	cutoff := time.Now().Add(-teamEvidenceWindow)
	for _, ent := range entries {
		cfg := filepath.Join(teamsDir, ent.Name(), "config.json")
		info, err := os.Stat(cfg)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if !info.ModTime().Before(cutoff) {
			return true
		}
	}
	return false
}

// pyRelpath returns path relative to base the way os.path.relpath does for the
// absolute clean paths build passes (entity_path under git_root). filepath.Rel
// computes the same relative path for these inputs.
func pyRelpath(path, base string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

// pyJoin concatenates path components the way os.path.join does: an absolute
// later component resets the path; otherwise components join with the OS
// separator without cleaning. Matches the status pyJoin.
func pyJoin(parts ...string) string {
	sep := string(filepath.Separator)
	result := ""
	for _, p := range parts {
		switch {
		case result == "":
			result = p
		case filepath.IsAbs(p):
			result = p
		case strings.HasSuffix(result, sep):
			result += p
		default:
			result += sep + p
		}
	}
	return result
}
