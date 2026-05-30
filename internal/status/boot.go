// ABOUTME: --boot section printer — MODS, ID_STYLE/NEXT_ID, ORPHANS, PR_STATE,
// ABOUTME: DISPATCHABLE, TEAM_STATE, matching print_boot and its probes.
package status

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const teamStateMtimeWindowSecs = 30 * 60

// orphan describes a worktree-backed entity for the ORPHANS section.
type orphan struct {
	id, slug, worktree, dirExists, branchExists string
}

// scanOrphans returns entities with a non-empty worktree field plus dir/branch
// existence info from `git worktree list --porcelain`. Matches scan_orphans.
func scanOrphans(entities []*entity, gitRoot string) []orphan {
	worktreePaths := map[string]bool{}
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = gitRoot
	out, err := cmd.Output()
	if err == nil {
		var current string
		for _, line := range strings.Split(string(out), "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				current = strings.TrimSpace(line[len("worktree "):])
			case strings.HasPrefix(line, "branch ") && current != "":
				worktreePaths[realpathOf(current)] = true
			case line == "":
				current = ""
			}
		}
	}

	var orphans []orphan
	for _, e := range entities {
		wt := e.fields["worktree"]
		if wt == "" {
			continue
		}
		// os.path.join(git_root, wt): an absolute wt discards git_root, so an
		// absolute worktree path is probed as-is. filepath.Join would instead
		// graft it under git_root and miss the existing dir. pyJoin matches the
		// oracle's os.path.join absolute-component-reset semantics.
		dirPath := pyJoin(gitRoot, wt)
		dirExists := "no"
		if st, err := os.Stat(dirPath); err == nil && st.IsDir() {
			dirExists = "yes"
		}
		branchExists := "no"
		if worktreePaths[realpathOf(dirPath)] {
			branchExists = "yes"
		}
		orphans = append(orphans, orphan{
			id: e.fields["id"], slug: e.fields["slug"], worktree: wt,
			dirExists: dirExists, branchExists: branchExists,
		})
	}
	return orphans
}

// prResult describes a PR-bearing entity's resolved state.
type prResult struct {
	id, slug, pr, state string
}

// checkPRStates returns (status, results) for entities with a non-empty pr and
// non-terminal status. Matches check_pr_states. status is "none",
// "gh not available", or "ok".
func checkPRStates(entities []*entity, stages []stage, e env) (string, []prResult) {
	stageByName := map[string]stage{}
	for _, s := range stages {
		stageByName[s.name] = s
	}
	var prEntities []*entity
	for _, ent := range entities {
		if ent.fields["pr"] == "" {
			continue
		}
		if st, ok := stageByName[ent.fields["status"]]; ok && st.terminal {
			continue
		}
		prEntities = append(prEntities, ent)
	}
	if len(prEntities) == 0 {
		return "none", nil
	}

	ghPath := lookupExecutable("gh", e.get("PATH"))
	if ghPath == "" {
		return "gh not available", nil
	}

	var results []prResult
	for _, ent := range prEntities {
		pr := strings.TrimPrefix(ent.fields["pr"], "#")
		state := "ERROR"
		out, err := exec.Command("gh", "pr", "view", pr, "--json", "state", "--jq", ".state").Output()
		if err == nil {
			state = strings.ToUpper(strings.TrimSpace(string(out)))
		}
		results = append(results, prResult{id: ent.fields["id"], slug: ent.fields["slug"], pr: ent.fields["pr"], state: state})
	}
	return "ok", results
}

// lookupExecutable returns the first executable named `name` on pathStr, or "".
// Matches the inline PATH scan in check_pr_states.
func lookupExecutable(name, pathStr string) string {
	for _, d := range filepath.SplitList(pathStr) {
		candidate := filepath.Join(d, name)
		if st, err := os.Stat(candidate); err == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return ""
}

// probeTeamState reports whether any ~/.claude/teams/*/config.json was modified
// within the last 30 minutes. Matches probe_team_state. Uses env HOME (the
// oracle expands ~, which resolves HOME).
func probeTeamState(e env) (bool, string) {
	home := e.get("HOME")
	if home == "" {
		home = os.Getenv("HOME")
	}
	teamsDir := filepath.Join(home, ".claude", "teams")
	info, err := os.Stat(teamsDir)
	if err != nil || !info.IsDir() {
		return false, ""
	}
	cutoff := time.Now().Add(-time.Duration(teamStateMtimeWindowSecs) * time.Second)
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		return false, ""
	}
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
		return true, "recent team directory: " + newest
	}
	return false, ""
}

// bootData holds the gathered boot-section material the text and JSON renderers
// both consume, so the two output forms read from one source of truth.
type bootData struct {
	hooks        map[string][]string
	idStyle      string
	nextID       string
	orphans      []orphan
	prStatus     string
	prResults    []prResult
	dispatchable []dispatchable
	teamPresent  bool
	teamHint     string
}

// gatherBoot runs every boot probe once and returns the result. NEXT_ID is
// minted here (timestamp-dependent for sd-b32); on a minting error it returns
// the error after the caller has emitted the stderr diagnostic.
func gatherBoot(entities []*entity, stages []stage, definitionDir, entityDir, gitRoot, idStyle string, e env, stderr io.Writer) (*bootData, error) {
	d := &bootData{idStyle: idStyle, hooks: scanMods(entityDir)}

	if idStyle == "slug" {
		d.nextID = "n/a (id-style: slug)"
	} else {
		id, err := computeNextID(definitionDir, entityDir, idStyle, "", "", e, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %s\n", err)
			return nil, err
		}
		d.nextID = id
	}

	d.orphans = scanOrphans(entities, gitRoot)
	d.prStatus, d.prResults = checkPRStates(entities, stages, e)
	d.dispatchable = computeDispatchable(entities, stages)
	d.teamPresent, d.teamHint = probeTeamState(e)
	return d, nil
}

// printBoot writes all boot sections in order. Matches print_boot.
func printBoot(w io.Writer, entities []*entity, stages []stage, definitionDir, entityDir, gitRoot, idStyle string, e env, stderr io.Writer) error {
	d, err := gatherBoot(entities, stages, definitionDir, entityDir, gitRoot, idStyle, e, stderr)
	if err != nil {
		return err
	}

	// MODS
	if len(d.hooks) == 0 {
		fmt.Fprintln(w, "MODS: none")
	} else {
		fmt.Fprintln(w, "MODS")
		points := make([]string, 0, len(d.hooks))
		for p := range d.hooks {
			points = append(points, p)
		}
		sort.Strings(points)
		for _, point := range points {
			mods := append([]string(nil), d.hooks[point]...)
			sort.Strings(mods)
			fmt.Fprintf(w, "%s: %s\n", point, strings.Join(mods, ", "))
		}
	}

	// ID_STYLE / NEXT_ID
	fmt.Fprintf(w, "ID_STYLE: %s\n", d.idStyle)
	fmt.Fprintf(w, "NEXT_ID: %s\n", d.nextID)
	if d.idStyle == "sd-b32" {
		fmt.Fprintf(w, "MIN_PREFIX: %d\n", sdB32MinPrefix)
	}

	// ORPHANS
	if len(d.orphans) == 0 {
		fmt.Fprintln(w, "ORPHANS: none")
	} else {
		fmt.Fprintln(w, "ORPHANS")
		row := func(a, b, c, d, e string) string {
			return padRight(a, 6) + " " + padRight(b, 30) + " " + padRight(c, 43) + " " + padRight(d, 11) + " " + e
		}
		fmt.Fprintln(w, row("ID", "SLUG", "WORKTREE", "DIR_EXISTS", "BRANCH_EXISTS"))
		for _, o := range d.orphans {
			fmt.Fprintln(w, row(o.id, o.slug, o.worktree, o.dirExists, o.branchExists))
		}
	}

	// PR_STATE
	switch d.prStatus {
	case "none":
		fmt.Fprintln(w, "PR_STATE: none")
	case "gh not available":
		fmt.Fprintln(w, "PR_STATE: gh not available")
	default:
		fmt.Fprintln(w, "PR_STATE")
		row := func(a, b, c, d string) string {
			return padRight(a, 6) + " " + padRight(b, 30) + " " + padRight(c, 8) + " " + d
		}
		fmt.Fprintln(w, row("ID", "SLUG", "PR", "STATE"))
		for _, r := range d.prResults {
			fmt.Fprintln(w, row(r.id, r.slug, r.pr, r.state))
		}
	}

	// DISPATCHABLE
	fmt.Fprintln(w, "DISPATCHABLE")
	printNextTable(w, entities, stages, nil, false)

	// TEAM_STATE
	fmt.Fprintln(w, "TEAM_STATE")
	if d.teamPresent {
		fmt.Fprintln(w, "present: true")
		fmt.Fprintf(w, "hint: %s\n", d.teamHint)
	} else {
		fmt.Fprintln(w, "present: false")
		fmt.Fprintln(w, "hint: run TeamCreate before first team-mode dispatch (claude runtime supports it)")
	}
	return nil
}
