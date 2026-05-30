// ABOUTME: --new atomic create — mint an id and write a valid entity in one fs
// ABOUTME: operation (temp+rename) so a seed never exists id-less. Decision B.
package status

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runNew implements --new [--folder] <slug>: read the entity body from stdin,
// mint the id-style-appropriate id, stamp it into the STDIN frontmatter, and
// write the entity in one filesystem operation (temp file + rename) so no
// id-less window is observable. folderForm selects {slug}/index.md; default
// flat {slug}.md matches current creation. Guards: slug already exists in EITHER
// form, STDIN lacks an opening fence, conflicting id in STDIN, and
// --id-seed/--id-actor with id-style: slug.
func runNew(roots roots, slug string, folderForm bool, idSeed, idActor string, idMaterialFlags []string,
	stdin io.Reader, stdout, stderr io.Writer, e env) int {

	// Slug must not already exist (flat or folder form).
	flatPath := filepath.Join(roots.entityDir, slug+".md")
	folderIndex := filepath.Join(roots.entityDir, slug, "index.md")
	if isRegularFile(flatPath) || isRegularFile(folderIndex) {
		return errExit(stderr, "entity already exists: "+slug)
	}

	// A nil Stdin (no pipe wired) reads as an empty body rather than panicking
	// io.ReadAll(nil); the empty body then fails the opening-fence guard below.
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		return errExit(stderr, "cannot read entity body from stdin: "+err.Error())
	}
	if !contentHasOpeningFence(body) {
		return errExit(stderr, "no frontmatter found in --new entity body (stdin must begin with ---)")
	}

	idStyle, err := workflowIDStyle(roots.definitionDir)
	if err != nil {
		return errExit(stderr, err.Error())
	}
	if idStyle == "slug" && len(idMaterialFlags) > 0 {
		return errExit(stderr, "--id-seed and --id-actor are only applicable for id-style: sd-b32")
	}

	// Reject a conflicting non-empty id already in the STDIN frontmatter.
	existingID := strings.TrimSpace(parseFrontmatterContent(body)["id"])

	var mintedID string
	if idStyle == "slug" {
		mintedID = slug
	} else {
		mintedID, err = computeNextID(roots.definitionDir, roots.entityDir, idStyle, idSeed, idActor, e, stderr)
		if err != nil {
			return errExit(stderr, err.Error())
		}
	}

	if existingID != "" && existingID != mintedID {
		return errExit(stderr, fmt.Sprintf("--new entity body already declares id '%s'; remove it so --new can mint the id", existingID))
	}

	// For id-style: slug the identity IS the slug; a stored `id:` field is
	// redundant and would make --resolve/--short-id emit id=<slug> where
	// hand-authored slug entities emit id= (empty). Leave the seed id-less so it
	// matches a hand-authored slug entity. The body is still newline-normalized
	// (universal newlines) so a later --set sees the same bytes as for any style.
	var stamped []byte
	if idStyle == "slug" {
		stamped = []byte(normalizeNewlines(string(body)))
	} else {
		stamped = stampID(body, mintedID)
	}

	var targetPath string
	if folderForm {
		targetPath = folderIndex
	} else {
		targetPath = flatPath
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return errExit(stderr, err.Error())
	}
	if err := atomicWrite(targetPath, stamped); err != nil {
		return errExit(stderr, err.Error())
	}
	fmt.Fprintf(stdout, "created: %s id=%s\n", targetPath, mintedID)
	return 0
}

// stampID inserts or overwrites the top-level id: line in the frontmatter of
// body, returning the new bytes. The id line is rewritten in place when present
// (matching update_frontmatter's in-place rewrite) or inserted before the
// closing --- when absent. Newlines are normalized to LF first (universal
// newlines), matching the read path so a later --set sees the same bytes; the
// split/join on "\n" then preserves the body's EOF-newline state byte-for-byte.
func stampID(body []byte, id string) []byte {
	lines := strings.Split(normalizeNewlines(string(body)), "\n")
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
		return body
	}
	for i := fmStart + 1; i < fmEnd; i++ {
		line := lines[i]
		if strings.Contains(line, ":") && !(len(line) > 0 && isSpaceByte(line[0])) {
			k, _, _ := strings.Cut(line, ":")
			if strings.TrimSpace(k) == "id" {
				lines[i] = "id: " + id
				return []byte(strings.Join(lines, "\n"))
			}
		}
	}
	// Not present: insert before the closing ---.
	inserted := append(lines[:fmEnd:fmEnd], append([]string{"id: " + id}, lines[fmEnd:]...)...)
	return []byte(strings.Join(inserted, "\n"))
}
