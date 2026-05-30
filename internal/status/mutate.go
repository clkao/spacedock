// ABOUTME: Mutation engine — update_frontmatter (--set) and run_archive
// ABOUTME: (--archive) with atomic temp+rename writes and the oracle's guards.
package status

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// timestampFields auto-fill now() when given bare (no value). Matches
// TIMESTAMP_FIELDS.
var timestampFields = map[string]bool{"started": true, "completed": true}

// fieldUpdate is one --set field operation. hasValue distinguishes an explicit
// value (including the empty clear "") from a bare timestamp field (value=None
// in the oracle, auto-filled with now()).
type fieldUpdate struct {
	field    string
	value    string
	hasValue bool // false => bare timestamp field
}

// nowTimestamp returns the YYYY-MM-DDTHH:MM:SSZ stamp the oracle writes.
func nowTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// updateFrontmatter rewrites matching top-level frontmatter lines in place,
// inserting genuinely-missing fields before the closing ---, and returns the
// resolved field->value map (in the order resolved). Bare timestamp fields skip
// when already set. The write is atomic (temp file + rename) and preserves the
// file's EOF-newline state byte-for-byte via split/join on "\n". Matches
// update_frontmatter.
func updateFrontmatter(path string, updates []fieldUpdate) (*orderedMap, error) {
	if !hasOpeningFence(path) {
		return nil, fmt.Errorf("No frontmatter found in %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Exact analog of Python content.split('\n'): the trailing empty element of
	// a newline-terminated file is preserved so '\n'.join restores it.
	lines := strings.Split(string(data), "\n")

	fmStart, fmEnd := -1, -1
	inFM := false
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == "---" {
			if !inFM {
				inFM = true
				fmStart = i
			} else {
				fmEnd = i
				break
			}
		}
	}
	if fmStart < 0 || fmEnd < 0 {
		return nil, fmt.Errorf("No frontmatter found in %s", path)
	}

	// Current values support skip-if-set for bare timestamp fields.
	current := map[string]string{}
	for i := fmStart + 1; i < fmEnd; i++ {
		line := lines[i]
		if strings.Contains(line, ":") && !(len(line) > 0 && isSpaceByte(line[0])) {
			k, v, _ := strings.Cut(line, ":")
			current[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}

	now := nowTimestamp()
	resolved := newOrderedMap()
	for _, u := range updates {
		if !u.hasValue {
			// Bare timestamp field — skip if already has a value.
			if current[u.field] != "" {
				continue
			}
			resolved.set(u.field, now)
		} else {
			resolved.set(u.field, u.value)
		}
	}

	// Rewrite matching lines.
	written := map[string]bool{}
	for i := fmStart + 1; i < fmEnd; i++ {
		line := lines[i]
		if strings.Contains(line, ":") && !(len(line) > 0 && isSpaceByte(line[0])) {
			k, _, _ := strings.Cut(line, ":")
			key := strings.TrimSpace(k)
			if val, ok := resolved.get(key); ok {
				lines[i] = key + ": " + val
				written[key] = true
			}
		}
	}

	// Insert genuinely-missing fields before the closing ---.
	for _, key := range resolved.keys() {
		if written[key] {
			continue
		}
		val, _ := resolved.get(key)
		ins := key + ": " + val
		lines = append(lines[:fmEnd], append([]string{ins}, lines[fmEnd:]...)...)
		fmEnd++
	}

	if err := atomicWrite(path, []byte(strings.Join(lines, "\n"))); err != nil {
		return nil, err
	}
	return resolved, nil
}

// atomicWrite writes data to a temp file in the same directory and renames it
// into place, so a reader never observes a half-written entity. The principle
// stated in decision B applied to --set, --archive's stamp, and --new.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".status-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// runArchive archives an entity (flat or folder form), stamping archived: before
// the move and printing `archived: {dest}`. Enforces the source-missing,
// already-archived, mod-block, and merge-hook guards. Matches run_archive.
// entityDir is the absolute entity root for I/O; spellingDir is the as-passed
// spelling used for the printed dest (so a relative --workflow-dir renders a
// relative `archived:` path, matching the oracle's literal os.path.join).
func runArchive(entityDir, spellingDir, slug string, force bool, stdout, stderr io.Writer) int {
	flatPath := filepath.Join(entityDir, slug+".md")
	folderRoot := filepath.Join(entityDir, slug)
	folderIndex := filepath.Join(folderRoot, "index.md")
	flatExists := isRegularFile(flatPath)
	folderExists := isRegularFile(folderIndex)

	var isFolder bool
	var sourcePath string
	switch {
	case folderExists:
		if flatExists {
			fmt.Fprintf(stderr,
				"Warning: entity '%s' has both %s and %s; preferring folder form. "+
					"Remove the flat file to silence this warning.\n",
				slug, flatPath, folderIndex)
		}
		isFolder = true
		sourcePath = folderIndex
	case flatExists:
		isFolder = false
		sourcePath = flatPath
	default:
		fmt.Fprintf(stderr, "Error: entity not found: %s\n", slug)
		return 1
	}

	fields := parseFrontmatter(sourcePath)
	modBlock := strings.TrimSpace(fields["mod-block"])
	pr := strings.TrimSpace(fields["pr"])
	if modBlock != "" {
		if !force {
			fmt.Fprintf(stderr, "Error: entity %s has pending mod-block (%s). Use --force to override.\n", slug, modBlock)
			return 1
		}
		fmt.Fprintf(stderr, "Warning: --force overriding mod-block (%s) on entity %s\n", modBlock, slug)
	}

	// Merge-hook invariant: archival is terminal. Refuse unless the hook ran
	// (pr set), is in flight (mod-block set, handled above), or --force.
	if !force && modBlock == "" && pr == "" {
		mergeHooks := scanMods(entityDir)["merge"]
		if len(mergeHooks) > 0 {
			fmt.Fprintf(stderr,
				"Error: entity %s cannot be archived — workflow has merge hook(s) [%s] "+
					"that have not run (pr field is empty and mod-block is empty). "+
					"Invoke the hook first, or use --force to bypass.\n",
				slug, strings.Join(mergeHooks, ", "))
			return 1
		}
	}

	archiveDir := filepath.Join(entityDir, "_archive")
	var destPath, destSpelling string
	if isFolder {
		destPath = filepath.Join(archiveDir, slug)
		destSpelling = pyJoin(spellingDir, "_archive", slug)
		if fileExists(destPath) {
			fmt.Fprintf(stderr, "Error: already archived: %s/\n", slug)
			return 1
		}
	} else {
		destPath = filepath.Join(archiveDir, slug+".md")
		destSpelling = pyJoin(spellingDir, "_archive", slug+".md")
		if fileExists(destPath) {
			fmt.Fprintf(stderr, "Error: already archived: %s.md\n", slug)
			return 1
		}
	}

	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	if _, err := updateFrontmatter(sourcePath, []fieldUpdate{{field: "archived", value: nowTimestamp(), hasValue: true}}); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	var moveErr error
	if isFolder {
		moveErr = os.Rename(folderRoot, destPath)
	} else {
		moveErr = os.Rename(sourcePath, destPath)
	}
	if moveErr != nil {
		fmt.Fprintf(stderr, "Error: %s\n", moveErr)
		return 1
	}
	fmt.Fprintf(stdout, "archived: %s\n", destSpelling)
	return 0
}

// pyJoin concatenates path components the way Python's os.path.join does: it
// joins with the OS separator without cleaning a leading "." (so
// pyJoin(".", "_archive", "x.md") == "./_archive/x.md", unlike filepath.Join
// which would collapse it to "_archive/x.md"). An absolute later component
// resets the accumulated path, matching os.path.join.
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

// scanMods scans entityDir/_mods/*.md for `## Hook:` headings, returning
// hookPoint -> sorted mod names. Matches scan_mods.
func scanMods(entityDir string) map[string][]string {
	modsDir := filepath.Join(entityDir, "_mods")
	info, err := os.Stat(modsDir)
	if err != nil || !info.IsDir() {
		return map[string][]string{}
	}
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return map[string][]string{}
	}
	var files []string
	for _, ent := range entries {
		if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".md") {
			files = append(files, ent.Name())
		}
	}
	// glob returns sorted order.
	sortStrings(files)
	hooks := map[string][]string{}
	for _, name := range files {
		modName := strings.TrimSuffix(name, ".md")
		data, err := os.ReadFile(filepath.Join(modsDir, name))
		if err != nil {
			continue
		}
		for _, line := range splitLines(string(data)) {
			if strings.HasPrefix(line, "## Hook:") {
				point := strings.TrimSpace(strings.TrimPrefix(line, "## Hook:"))
				if point != "" {
					hooks[point] = append(hooks[point], modName)
				}
			}
		}
	}
	for k := range hooks {
		sortStrings(hooks[k])
	}
	return hooks
}
