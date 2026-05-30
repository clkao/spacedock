// ABOUTME: Builds the per-command --json envelopes (status/next/resolve/boot/
// ABOUTME: next-id/validate) as ordered string objects over the same data the table renders.
package status

import (
	"sort"
	"strconv"
	"strings"
)

// resolveJSONFields returns the ordered key set for a status-shape JSON object.
// Strict in JSON mode: explicit --fields is exactly the named order; --all-fields
// is defaults followed by sorted non-empty non-underscore non-default keys; no
// flag is the defaults. Mirrors resolveExtraFields' --all-fields scan but yields
// the full key set (defaults + extras), since JSON keys ARE the projection.
func resolveJSONFields(entities []*entity, explicitFields []string, allFields bool, defaultFields []string) []string {
	if explicitFields != nil {
		return append([]string(nil), explicitFields...)
	}
	if allFields {
		defaults := map[string]bool{}
		for _, f := range defaultFields {
			defaults[f] = true
		}
		seen := map[string]bool{}
		for _, e := range entities {
			for key, val := range e.fields {
				if strings.HasPrefix(key, "_") || defaults[key] || val == "" {
					continue
				}
				seen[key] = true
			}
		}
		extras := make([]string, 0, len(seen))
		for k := range seen {
			extras = append(extras, k)
		}
		sort.Strings(extras)
		return append(append([]string(nil), defaultFields...), extras...)
	}
	return append([]string(nil), defaultFields...)
}

// entityJSONObj projects one entity to an ordered object over fields, reading
// each value from e.fields (id is already the display id post-applyEffectiveIDs).
func entityJSONObj(e *entity, fields []string) *jsonObj {
	o := newJSONObj()
	for _, f := range fields {
		o.set(f, e.fields[f])
	}
	return o
}

// statusJSON builds the {"command":"status","entities":[...]} envelope for the
// default / --archived / --where reads. Array order is sortDefault.
func statusJSON(entities []*entity, stages []Stage, fields []string) *jsonObj {
	sorted := sortDefault(entities, stages)
	arr := make(jsonArr, 0, len(sorted))
	for _, e := range sorted {
		arr = append(arr, entityJSONObj(e, fields))
	}
	return newJSONObj().set("command", "status").setValue("entities", arr)
}

// nextFixedFields are the always-present --next keys: id, slug, plus the three
// computed dispatch columns. --fields adds frontmatter keys after these.
var nextFixedFields = []string{"id", "slug", "current", "next", "worktree"}

// nextJSON builds the {"command":"next","dispatchable":[...]} envelope. The
// fixed five are always present; explicit/--all-fields frontmatter keys are
// additive after them (the computed columns are not projectable, per spike).
func nextJSON(entities []*entity, stages []Stage, explicitFields []string, allFields bool) *jsonObj {
	disp := computeDispatchable(entities, stages)
	extras := resolveNextExtras(entities, explicitFields, allFields)
	arr := dispatchableJSONArr(disp)
	for i, d := range disp {
		for _, f := range extras {
			arr[i].set(f, d.e.fields[f])
		}
	}
	return newJSONObj().set("command", "next").setValue("dispatchable", arr)
}

// resolveNextExtras returns the additive frontmatter keys for --next JSON:
// explicit --fields minus any that collide with the fixed five, or the
// --all-fields scan minus the fixed five.
func resolveNextExtras(entities []*entity, explicitFields []string, allFields bool) []string {
	fixed := map[string]bool{}
	for _, f := range nextFixedFields {
		fixed[f] = true
	}
	if explicitFields != nil {
		var out []string
		for _, f := range explicitFields {
			if !fixed[f] {
				out = append(out, f)
			}
		}
		return out
	}
	if allFields {
		seen := map[string]bool{}
		for _, e := range entities {
			for key, val := range e.fields {
				if strings.HasPrefix(key, "_") || fixed[key] || val == "" {
					continue
				}
				seen[key] = true
			}
		}
		out := make([]string, 0, len(seen))
		for k := range seen {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return nil
}

// dispatchableJSONArr builds the dispatchable array shared by --next and --boot:
// the fixed five keys per row, no projected extras (boot takes no --fields).
func dispatchableJSONArr(disp []dispatchable) jsonArr {
	arr := make(jsonArr, 0, len(disp))
	for _, d := range disp {
		o := newJSONObj()
		o.set("id", d.e.fields["id"])
		o.set("slug", d.e.fields["slug"])
		o.set("current", d.e.fields["status"])
		o.set("next", d.next)
		o.set("worktree", d.nextWorktree)
		arr = append(arr, o)
	}
	return arr
}

// bootJSON builds the nested {"command":"boot",...} envelope from gathered boot
// data. mods is an object of point->[mods] (empty -> {}); min_prefix is present
// only for sd-b32; team_state.present is the string "true"/"false".
func bootJSON(d *bootData) *jsonObj {
	out := newJSONObj().set("command", "boot")

	mods := newJSONObj()
	points := make([]string, 0, len(d.hooks))
	for p := range d.hooks {
		points = append(points, p)
	}
	sort.Strings(points)
	for _, p := range points {
		vals := append([]string(nil), d.hooks[p]...)
		sort.Strings(vals)
		mods.setValue(p, jsonStrArr(vals))
	}
	out.setValue("mods", mods)

	out.set("id_style", d.idStyle)
	out.set("next_id", d.nextID)
	if d.idStyle == "sd-b32" {
		out.set("min_prefix", strconv.Itoa(sdB32MinPrefix))
	}

	orphans := make(jsonArr, 0, len(d.orphans))
	for _, o := range d.orphans {
		orphans = append(orphans, newJSONObj().
			set("id", o.id).set("slug", o.slug).set("worktree", o.worktree).
			set("dir_exists", o.dirExists).set("branch_exists", o.branchExists))
	}
	out.setValue("orphans", orphans)

	prState := newJSONObj().set("status", d.prStatus)
	prEntries := make(jsonArr, 0, len(d.prResults))
	for _, r := range d.prResults {
		prEntries = append(prEntries, newJSONObj().
			set("id", r.id).set("slug", r.slug).set("pr", r.pr).set("state", r.state))
	}
	prState.setValue("entries", prEntries)
	out.setValue("pr_state", prState)

	out.setValue("dispatchable", dispatchableJSONArr(d.dispatchable))

	team := newJSONObj()
	if d.teamPresent {
		team.set("present", "true").set("hint", d.teamHint)
	} else {
		team.set("present", "false").set("hint", "run TeamCreate before first team-mode dispatch (claude runtime supports it)")
	}
	out.setValue("team_state", team)

	return out
}

// resolveJSON builds the single {"command":"resolve",...} object. id is the
// display id (uniform with every other command); stored_id is the full stored
// form the text resolve line carries, so cross-command round-trip holds.
func resolveJSON(workflowDir string, e *entity) *jsonObj {
	return newJSONObj().
		set("command", "resolve").
		set("workflow", realpathOf(workflowDir)).
		set("scope", scopeOf(e)).
		set("slug", e.slug).
		set("id", e.fields["id"]).
		set("stored_id", e.storedID).
		set("path", e.path)
}

// singletonJSON builds a {"command":cmd,key:value} object for the single-token
// read outputs (--next-id, --short-id, --validate) when --json is explicit.
func singletonJSON(cmd, key, value string) *jsonObj {
	return newJSONObj().set("command", cmd).set(key, value)
}
