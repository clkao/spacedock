// ABOUTME: Release-pipeline version steps — AC-4 plugin.json stamp + AC-2d
// ABOUTME: marketplace-entry calendar bump, as pure functions a CI step invokes.
package release

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestStampVersionRewritesPluginVersion locks AC-4: StampVersion rewrites the
// top-level plugin.json `version` to the release value, leaving every other
// field (and the file's formatting) intact.
func TestStampVersionRewritesPluginVersion(t *testing.T) {
	src := `{
  "name": "spacedock",
  "version": "0.1.0-dev",
  "skills": "./skills/",
  "requires-contract": ">=1,<2"
}
`
	out, err := StampVersion([]byte(src), "0.19.0")
	if err != nil {
		t.Fatalf("StampVersion: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("stamped manifest does not parse: %v\n%s", err, out)
	}
	if m["version"] != "0.19.0" {
		t.Errorf("version = %v, want 0.19.0", m["version"])
	}
	// Untouched fields survive.
	if m["name"] != "spacedock" {
		t.Errorf("name field lost: %v", m["name"])
	}
	if m["requires-contract"] != ">=1,<2" {
		t.Errorf("requires-contract field lost: %v", m["requires-contract"])
	}
	if m["skills"] != "./skills/" {
		t.Errorf("skills field lost: %v", m["skills"])
	}
}

// TestStampVersionLeavesMarketplaceCalendarUntouched locks AC-4's negative half:
// the stamp step targets the top-level plugin `version` only. Applied to a
// marketplace.json (whose meaningful version lives on the nested plugin ENTRY,
// not at top level), it must NOT rewrite the entry's calendar version. The stamp
// is a plugin.json operation; a marketplace.json has no top-level version to
// stamp, so the entry calendar key is left exactly as-is.
func TestStampVersionLeavesMarketplaceCalendarUntouched(t *testing.T) {
	src := `{
  "name": "spacedock",
  "plugins": [
    {
      "name": "spacedock",
      "version": "0.0.2026053101"
    }
  ]
}
`
	out, err := StampVersion([]byte(src), "0.19.0")
	if err != nil {
		t.Fatalf("StampVersion: %v", err)
	}
	if strings.Contains(string(out), "0.19.0") {
		t.Errorf("stamp wrote the release version into a marketplace.json entry; calendar key must be untouched:\n%s", out)
	}
	if !strings.Contains(string(out), "0.0.2026053101") {
		t.Errorf("marketplace entry calendar version was lost:\n%s", out)
	}
}

// TestBumpCalendarVersionStrictlyIncreases locks AC-2d: invoking the bump
// function twice over the SAME marketplace.json produces a strictly increasing
// entry version (the `plugin update` re-pull key actually moves), not two
// hand-written literals. The second call (same day) must increment the per-day
// sequence so the value is monotonic even within a single publish day.
func TestBumpCalendarVersionStrictlyIncreases(t *testing.T) {
	src := `{
  "name": "spacedock",
  "plugins": [
    {
      "name": "spacedock",
      "source": { "source": "url", "url": "https://example/spacedock.git", "ref": "next" },
      "version": "0.0.2026053101",
      "category": "workflow"
    }
  ]
}
`
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	first, err := BumpCalendarVersion([]byte(src), now)
	if err != nil {
		t.Fatalf("first bump: %v", err)
	}
	second, err := BumpCalendarVersion(first, now)
	if err != nil {
		t.Fatalf("second bump: %v", err)
	}

	v1 := entryVersion(t, first)
	v2 := entryVersion(t, second)
	if !(v2 > v1) {
		t.Errorf("calendar version did not strictly increase: %q then %q", v1, v2)
	}
	// The bump is calendar-keyed: same day -> shared date prefix, incremented seq.
	if !strings.HasPrefix(v1, "0.0.20260601") {
		t.Errorf("first bump = %q, want 0.0.20260601NN prefix", v1)
	}
	if !strings.HasPrefix(v2, "0.0.20260601") {
		t.Errorf("second bump = %q, want 0.0.20260601NN prefix", v2)
	}
}

// TestBumpCalendarVersionNewDayResetsSequence locks the cross-day behavior: a
// bump on a later date produces a strictly greater value than a prior-day bump
// even though the new day's sequence restarts at 01 — the date component
// dominates the ordering.
func TestBumpCalendarVersionNewDayResetsSequence(t *testing.T) {
	src := `{"name":"spacedock","plugins":[{"name":"spacedock","version":"0.0.2026053199"}]}`
	day2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	out, err := BumpCalendarVersion([]byte(src), day2)
	if err != nil {
		t.Fatalf("bump: %v", err)
	}
	v := entryVersion(t, out)
	if v != "0.0.2026060101" {
		t.Errorf("new-day bump = %q, want 0.0.2026060101 (seq resets to 01 on a new date)", v)
	}
	if !(v > "0.0.2026053199") {
		t.Errorf("new-day bump %q did not exceed prior-day max 0.0.2026053199", v)
	}
}

// entryVersion extracts plugins[0].version from a marketplace.json blob.
func entryVersion(t *testing.T, blob []byte) string {
	t.Helper()
	var m struct {
		Plugins []struct {
			Version string `json:"version"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(blob, &m); err != nil {
		t.Fatalf("parse marketplace blob: %v\n%s", err, blob)
	}
	if len(m.Plugins) == 0 {
		t.Fatalf("no plugins entry in blob:\n%s", blob)
	}
	return m.Plugins[0].Version
}
