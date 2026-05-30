// ABOUTME: Entity discovery matching discover_entity_files / resolve_entity_path
// ABOUTME: — flat {slug}.md or folder {slug}/index.md, reserved-dir + conflict rules.
package status

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// reservedSubdirs are never treated as entity folders. Dot-prefixed dirs are
// also skipped (checked separately). Matches RESERVED_SUBDIRS exactly.
var reservedSubdirs = map[string]bool{"_archive": true, "_mods": true}

// entity carries an entity's frontmatter fields plus discovery/identity
// metadata. fields holds the raw frontmatter (default keys backfilled to "");
// the underscore-prefixed metadata mirrors the oracle's _stored_id/_path/_scope/
// _display_id dict keys.
type entity struct {
	fields    map[string]string
	slug      string
	storedID  string
	path      string
	scope     string // "active" or "archived"
	displayID string
}

// get returns a frontmatter field value, "" when absent — matching dict.get.
func (e *entity) get(key string) string {
	return e.fields[key]
}

// discoverEntityFiles returns (slug, path) pairs for entities in directory,
// sorted by slug. Flat {slug}.md and folder {slug}/index.md are both entities;
// README.md, non-.md files, dotfiles, fence-less files, reserved dirs, and
// dot-dirs are skipped. On a flat/folder conflict the folder wins and a warning
// is written to stderr. Matches discover_entity_files.
func discoverEntityFiles(directory string, stderr io.Writer) [][2]string {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}

	flatPaths := map[string]string{}
	folderPaths := map[string]string{}
	for _, ent := range entries {
		name := ent.Name()
		full := filepath.Join(directory, name)
		// Classify by os.Stat (follows symlinks) to match the oracle's
		// os.path.isfile / os.path.isdir checks.
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() {
			if name == "README.md" || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
				continue
			}
			if !hasOpeningFence(full) {
				continue
			}
			flatPaths[strings.TrimSuffix(name, ".md")] = full
		} else if info.IsDir() {
			if reservedSubdirs[name] || strings.HasPrefix(name, ".") {
				continue
			}
			indexPath := filepath.Join(full, "index.md")
			if isRegularFile(indexPath) && hasOpeningFence(indexPath) {
				folderPaths[name] = indexPath
			}
		}
	}

	slugSet := map[string]bool{}
	for s := range flatPaths {
		slugSet[s] = true
	}
	for s := range folderPaths {
		slugSet[s] = true
	}
	slugs := make([]string, 0, len(slugSet))
	for s := range slugSet {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)

	var out [][2]string
	for _, slug := range slugs {
		if fp, ok := folderPaths[slug]; ok {
			if flat, both := flatPaths[slug]; both {
				fmt.Fprintf(stderr,
					"Warning: entity '%s' has both %s and %s; preferring folder form. "+
						"Remove the flat file to silence this warning.\n",
					slug, flat, fp)
			}
			out = append(out, [2]string{slug, fp})
		} else {
			out = append(out, [2]string{slug, flatPaths[slug]})
		}
	}
	return out
}

// resolveEntityPath returns the entity file path for a slug, or "" if absent.
// Folder form wins over flat form, with the same conflict warning. Matches
// resolve_entity_path.
func resolveEntityPath(directory, slug string, stderr io.Writer) string {
	flatPath := filepath.Join(directory, slug+".md")
	folderPath := filepath.Join(directory, slug, "index.md")
	flatExists := isRegularFile(flatPath)
	folderExists := isRegularFile(folderPath)
	if folderExists {
		if flatExists {
			fmt.Fprintf(stderr,
				"Warning: entity '%s' has both %s and %s; preferring folder form. "+
					"Remove the flat file to silence this warning.\n",
				slug, flatPath, folderPath)
		}
		return folderPath
	}
	if flatExists {
		return flatPath
	}
	return ""
}

// defaultEntityKeys are backfilled to "" on every scanned entity, matching the
// oracle's setdefault loop.
var defaultEntityKeys = []string{"id", "status", "title", "score", "source", "worktree"}

// scanEntities scans a directory for entities (active scope). Matches
// scan_entities.
func scanEntities(directory string, stderr io.Writer) []*entity {
	var out []*entity
	for _, pair := range discoverEntityFiles(directory, stderr) {
		slug, path := pair[0], pair[1]
		fields := parseFrontmatter(path)
		out = append(out, newEntity(fields, slug, path, "active"))
	}
	return out
}

// newEntity backfills default keys and captures the stored id, path, and scope,
// mirroring the oracle's per-entity dict construction. The slug is written into
// fields (the oracle's entity['slug'] = slug) so formatters/filters can read it.
func newEntity(fields map[string]string, slug, path, scope string) *entity {
	fields["slug"] = slug
	for _, k := range defaultEntityKeys {
		if _, ok := fields[k]; !ok {
			fields[k] = ""
		}
	}
	e := &entity{
		fields:   fields,
		slug:     slug,
		storedID: fields["id"],
		path:     path,
		scope:    scope,
	}
	return e
}

// archiveEntities scans entityDir/_archive in archived scope. Matches
// archive_entities.
func archiveEntities(entityDir string, stderr io.Writer) []*entity {
	archiveDir := filepath.Join(entityDir, "_archive")
	info, err := os.Stat(archiveDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	ents := scanEntities(archiveDir, stderr)
	for _, e := range ents {
		e.scope = "archived"
	}
	return ents
}

// activeAndArchivedEntities returns active + archived entities. Matches
// active_and_archived_entities (single-root; no worktree overlay in this stage).
func activeAndArchivedEntities(entityDir string, stderr io.Writer) []*entity {
	return append(scanEntities(entityDir, stderr), archiveEntities(entityDir, stderr)...)
}

// isRegularFile reports whether path is an existing regular file.
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
