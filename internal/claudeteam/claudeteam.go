// ABOUTME: The Claude runtime seam — owns the ~/.claude/teams reads behind a
// ABOUTME: host-supplied TeamStateProbe plus the boot hint + bare-mode advisory text.
package claudeteam

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// teamStateWindow is the lookback for "recent team-runtime evidence": a
// ~/.claude/teams/*/config.json touched inside this window means a Claude team
// session is live. The boot TEAM_STATE and the build bare-mode advisory share it.
const teamStateWindow = 30 * time.Minute

// PresentFalseHint is the boot TEAM_STATE present:false hint the Claude seam
// supplies. It names a Claude-only tool (TeamCreate), so it lives here, not in
// the generic internal/status renderer — the generic renderer reads it only when
// a probe is wired, and emits a host-neutral line when the probe is nil.
const PresentFalseHint = "run TeamCreate before first team-mode dispatch (claude runtime supports it)"

// TeamStateProbe reports recent local team-runtime evidence over the shared
// ~/.claude/teams read. present drives the boot TEAM_STATE present field; hint is
// the boot present:true hint line; recent drives the build bare-mode advisory
// gate. now is injected so the 30-minute window is testable. internal/status and
// internal/dispatch take this as a value (nil on a non-Claude host) so their
// source carries no ~/.claude read.
type TeamStateProbe func(home string, now time.Time) (present bool, hint string, recent bool)

// Probe is the concrete Claude implementation: it scans ~/.claude/teams/*/config.json
// mtimes under home for the newest one inside the 30-minute window. present and
// recent both report whether such a config exists; hint names the newest team
// directory on the present path. home is the resolved HOME (the caller keeps HOME
// resolution generic; only this ~/.claude read is Claude-specific). The Claude CLI
// front door wires claudeteam.Probe; Codex/bare wire nil.
func Probe(home string, now time.Time) (present bool, hint string, recent bool) {
	teamsDir := filepath.Join(home, ".claude", "teams")
	info, err := os.Stat(teamsDir)
	if err != nil || !info.IsDir() {
		return false, "", false
	}
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		return false, "", false
	}
	cutoff := now.Add(-teamStateWindow)
	var newest string
	var newestMtime time.Time
	for _, ent := range entries {
		cfg := filepath.Join(teamsDir, ent.Name(), "config.json")
		st, err := os.Stat(cfg)
		if err != nil || !st.Mode().IsRegular() {
			continue
		}
		if st.ModTime().After(newestMtime) {
			newestMtime = st.ModTime()
			newest = ent.Name()
		}
	}
	if newest != "" && !newestMtime.Before(cutoff) {
		return true, "recent team directory: " + newest, true
	}
	return false, "", false
}

// BareModeAdvisory writes the bare-mode dispatch warning to w. It names a
// Claude-only bootstrap tool (TeamCreate) and the ~/.claude/teams path, so the
// text lives in the Claude seam, not in the generic internal/dispatch build path.
// The generic build path calls this only when a Claude probe is wired AND reports
// no recent evidence; a nil-probe (Codex/bare) host emits no advisory at all.
func BareModeAdvisory(w io.Writer) {
	fmt.Fprintln(w,
		"WARN: bare_mode dispatch with no recent TeamCreate evidence "+
			"(no ~/.claude/teams/*/config.json modified in the last 30 minutes). "+
			"If you intend teams mode, run ToolSearch select:TeamCreate and TeamCreate first. "+
			"If bare is intentional, this warning can be ignored.")
}
