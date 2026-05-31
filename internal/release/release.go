// ABOUTME: Release-pipeline version steps — stamp plugin.json `version` to the
// ABOUTME: release (AC-4) and bump the marketplace entry calendar key (AC-2d).
package release

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// topLevelVersionRe matches the first `"version": "..."` member of a JSON object.
// In a plugin.json the top-level version is the first such member; in a
// marketplace.json the version lives only on the nested plugin entry, so a blob
// with no top-level `version` key is left untouched by StampVersion.
var topLevelVersionRe = regexp.MustCompile(`("version"\s*:\s*")[^"]*(")`)

// StampVersion rewrites the top-level `version` field of a plugin manifest
// (plugin.json / .codex-plugin/plugin.json) to version, preserving the rest of
// the file's formatting. When the manifest has no top-level `version` key (e.g.
// a marketplace.json, whose version lives on the nested plugin entry), the input
// is returned unchanged — the stamp is a plugin.json operation and must not move
// the marketplace entry's calendar key.
func StampVersion(manifest []byte, version string) ([]byte, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(manifest, &top); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if _, ok := top["version"]; !ok {
		// No top-level version (marketplace.json shape): nothing to stamp.
		return manifest, nil
	}
	out := topLevelVersionRe.ReplaceAll(manifest, []byte("${1}"+version+"${2}"))
	return out, nil
}

// entryVersionRe matches the `"version": "..."` member of the nested plugin
// entry in a marketplace.json. A marketplace.json carries no top-level version,
// so the first match is the entry's calendar key.
var entryVersionRe = regexp.MustCompile(`("version"\s*:\s*")[^"]*(")`)

// BumpCalendarVersion advances the marketplace plugin entry's calendar version
// to `0.0.YYYYMMDDNN` for the date in now. NN is a per-day sequence: when the
// existing entry version already carries today's date, NN increments; otherwise
// it restarts at 01. The date component dominates the ordering, so the value is
// strictly monotonic across both same-day re-bumps and day boundaries — exactly
// the `claude plugin update` re-pull key the moving `next` branch needs.
func BumpCalendarVersion(marketplace []byte, now time.Time) ([]byte, error) {
	date := now.UTC().Format("20060102")
	prefix := "0.0." + date

	var doc struct {
		Plugins []struct {
			Version string `json:"version"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(marketplace, &doc); err != nil {
		return nil, fmt.Errorf("parse marketplace: %w", err)
	}
	if len(doc.Plugins) == 0 {
		return nil, fmt.Errorf("marketplace has no plugin entry to bump")
	}

	seq := 1
	if cur := doc.Plugins[0].Version; len(cur) > len(prefix) && cur[:len(prefix)] == prefix {
		if n, err := strconv.Atoi(cur[len(prefix):]); err == nil {
			seq = n + 1
		}
	}
	next := fmt.Sprintf("%s%02d", prefix, seq)

	out := entryVersionRe.ReplaceAll(marketplace, []byte("${1}"+next+"${2}"))
	return out, nil
}
