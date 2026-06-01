// ABOUTME: Claude standing-teammate surface — the SendMessage routing-prose render
// ABOUTME: and the ~/.claude/teams member-exists probe spawn-standing gates on.
package claudeteam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StandingTeammate is the data the routing-prose render consumes: the spawn name,
// the frontmatter description, and the pre-extracted ## Routing Usage body ("" when
// the mod declares none). The runtime-neutral _mods parsing (in internal/dispatch)
// produces this; the render here turns it into Claude SendMessage routing prose.
type StandingTeammate struct {
	Name             string
	Description      string
	RoutingUsageBody string
}

// RenderStandingTeammatesSection assembles the `### Standing teammates available
// in your team` markdown from the declared teammates. Returns "" for an empty
// input (no declared standing teammates). The body uses Claude SendMessage routing
// language, which is why it lives in the Claude seam. Mirrors
// render_standing_teammates_section.
func RenderStandingTeammatesSection(teammates []StandingTeammate) string {
	if len(teammates) == 0 {
		return ""
	}
	lines := []string{
		"### Standing teammates available in your team",
		"",
		"These standing teammates are available in your team; you MAY route to them via " +
			"SendMessage. Best-effort, non-blocking, 2-minute timeout; proceed with " +
			"un-polished/un-reviewed content if no reply.",
		"",
	}
	for _, tm := range teammates {
		desc := tm.Description
		if desc == "" {
			desc = "standing teammate"
		}
		if tm.RoutingUsageBody != "" {
			lines = append(lines, fmt.Sprintf("- **%s** (%s)", tm.Name, desc))
			for _, bodyLine := range strings.Split(tm.RoutingUsageBody, "\n") {
				if bodyLine != "" {
					lines = append(lines, "  "+bodyLine)
				} else {
					lines = append(lines, "")
				}
			}
		} else {
			lines = append(lines, fmt.Sprintf(
				"- **%s** (%s): SendMessage with the relevant input shape; reply format per the mod.",
				tm.Name, desc))
		}
	}
	lines = append(lines, "")
	lines = append(lines,
		"Full routing contract: see "+
			"`skills/first-officer/references/first-officer-shared-core.md` "+
			"`## Standing Teammates`.")
	return strings.Join(lines, "\n")
}

// MemberExists reports whether member name is in team team's config.json members
// list. Team-scoped (one file), distinct from lookupModel's all-teams scan. The
// ~/.claude/teams read lives here in the Claude seam. home is the resolved HOME.
// Mirrors member_exists.
func MemberExists(home, team, name string) bool {
	configPath := filepath.Join(home, ".claude", "teams", team, "config.json")
	if !isFile(configPath) {
		return false
	}
	cfg, ok := readTeamConfig(configPath)
	if !ok {
		return false
	}
	return cfg.hasMember(name)
}

// isFile reports whether path is an existing regular file (os.path.isfile).
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
