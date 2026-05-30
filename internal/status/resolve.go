// ABOUTME: Reference resolution — parse_reference, resolve_reference_candidates,
// ABOUTME: and the --resolve / --short-id / mutation-target resolvers.
package status

import (
	"fmt"
	"io"
	"strings"
)

// reference is a parsed resolver reference: scope filter, lookup mode, value.
type reference struct {
	scopeFilter string // "", "active", or "archived"
	mode        string // "bare", "id", "slug", "prefix"
	value       string
}

// parseReference splits a ref into (scope filter, mode, value). Matches
// parse_reference.
func parseReference(ref string) reference {
	r := reference{value: ref, mode: "bare"}
	switch {
	case strings.HasPrefix(ref, "active:"):
		r.scopeFilter = "active"
		r.value = ref[len("active:"):]
	case strings.HasPrefix(ref, "archive:"):
		r.scopeFilter = "archived"
		r.value = ref[len("archive:"):]
	}
	switch {
	case strings.HasPrefix(r.value, "id:"):
		r.mode = "id"
		r.value = r.value[len("id:"):]
	case strings.HasPrefix(r.value, "slug:"):
		r.mode = "slug"
		r.value = r.value[len("slug:"):]
	case strings.HasPrefix(r.value, "prefix:"):
		r.mode = "prefix"
		r.value = r.value[len("prefix:"):]
	}
	return r
}

// formatResolveLine renders the --resolve output line. workflow= is realpath'd;
// path= is not. Matches format_resolve_line. workflowDir here is the entity
// root (matching the oracle, which passes pipeline_dir).
func formatResolveLine(workflowDir string, e *entity) string {
	return fmt.Sprintf("workflow=%s scope=%s slug=%s id=%s path=%s",
		realpathOf(workflowDir), scopeOf(e), e.slug, e.storedID, e.path)
}

// candidateLine renders an ambiguity/diagnostic candidate line. Matches
// candidate_line.
func candidateLine(e *entity) string {
	display := e.displayID
	if display == "" {
		display = e.fields["id"]
	}
	return fmt.Sprintf("scope=%s slug=%s id=%s display=%s path=%s",
		scopeOf(e), e.slug, e.storedID, display, e.path)
}

// resolveResult is the outcome of resolveReferenceCandidates.
type resolveResult struct {
	status  string // "ok", "unknown", "ambiguous", "too_short"
	matches []*entity
	errors  []string
}

// resolveReferenceCandidates resolves a ref within one workflow. Matches
// resolve_reference_candidates. definitionDir/entityDir are the README and
// entity roots (equal in this stage).
func resolveReferenceCandidates(definitionDir, entityDir, ref string, includeArchived bool, idStyle string, stderr io.Writer) resolveResult {
	all := activeAndArchivedEntities(entityDir, stderr)
	applyEffectiveIDs(all, idStyle, all)

	r := parseReference(ref)
	allowed := map[string]bool{"active": true}
	if includeArchived {
		allowed["archived"] = true
	}
	if r.scopeFilter != "" {
		allowed = map[string]bool{r.scopeFilter: true}
	}
	var entities []*entity
	for _, e := range all {
		if allowed[scopeOf(e)] {
			entities = append(entities, e)
		}
	}

	inScope := func(pred func(*entity) bool) []*entity {
		var out []*entity
		for _, e := range entities {
			if pred(e) {
				out = append(out, e)
			}
		}
		return out
	}

	var matches []*entity
	switch r.mode {
	case "slug":
		matches = inScope(func(e *entity) bool { return e.slug == r.value })
	case "id":
		matches = inScope(func(e *entity) bool { return e.storedID == r.value })
	case "prefix":
		if idStyle != "sd-b32" {
			return resolveResult{"unknown", nil, []string{"Error: prefix resolution is only available for id-style: sd-b32"}}
		}
		if len(r.value) < sdB32MinPrefix {
			return resolveResult{"too_short", nil, []string{fmt.Sprintf("Error: sd-b32 address prefix '%s' is too short; minimum prefix length is %d", r.value, sdB32MinPrefix)}}
		}
		matches = inScope(func(e *entity) bool { return strings.HasPrefix(e.storedID, r.value) })
	default:
		slugMatches := inScope(func(e *entity) bool { return e.slug == r.value })
		idMatches := inScope(func(e *entity) bool { return e.storedID == r.value })
		var prefixMatches []*entity
		prefixTooShort := false
		if idStyle == "sd-b32" && allInAlphabet(r.value) {
			if len(r.value) < sdB32MinPrefix {
				prefixTooShort = true
			} else {
				prefixMatches = inScope(func(e *entity) bool { return strings.HasPrefix(e.storedID, r.value) })
			}
		}

		var combined []*entity
		seen := map[string]bool{}
		for _, e := range append(append(append([]*entity{}, slugMatches...), idMatches...), prefixMatches...) {
			key := scopeOf(e) + "\x00" + e.slug + "\x00" + e.path
			if seen[key] {
				continue
			}
			seen[key] = true
			combined = append(combined, e)
		}
		scopes := map[string]bool{}
		for _, e := range combined {
			scopes[scopeOf(e)] = true
		}
		if includeArchived && r.scopeFilter == "" && len(scopes) > 1 {
			errs := []string{"Error: ambiguous reference across active and archived entities"}
			for _, e := range combined {
				errs = append(errs, candidateLine(e))
			}
			return resolveResult{"ambiguous", combined, errs}
		}

		if len(slugMatches) > 0 {
			if len(slugMatches) == 1 {
				return resolveResult{"ok", slugMatches, nil}
			}
			errs := []string{"Error: ambiguous slug reference"}
			for _, e := range slugMatches {
				errs = append(errs, candidateLine(e))
			}
			return resolveResult{"ambiguous", slugMatches, errs}
		}
		if len(idMatches) > 0 {
			matches = idMatches
		} else if len(prefixMatches) > 0 {
			matches = prefixMatches
		} else if prefixTooShort {
			return resolveResult{"too_short", nil, []string{fmt.Sprintf("Error: sd-b32 address prefix '%s' is too short; minimum prefix length is %d", r.value, sdB32MinPrefix)}}
		}
	}

	if len(matches) == 0 {
		return resolveResult{"unknown", nil, []string{fmt.Sprintf("Error: unknown reference: %s", ref)}}
	}
	if len(matches) > 1 {
		errs := []string{fmt.Sprintf("Error: ambiguous reference: %s", ref)}
		for _, e := range matches {
			errs = append(errs, candidateLine(e))
		}
		return resolveResult{"ambiguous", matches, errs}
	}
	return resolveResult{"ok", matches, nil}
}

func allInAlphabet(value string) bool {
	for _, ch := range value {
		if !strings.ContainsRune(sdB32Chars, ch) {
			return false
		}
	}
	return true
}
