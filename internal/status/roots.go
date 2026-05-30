// ABOUTME: Root-resolution seam — resolveRoots(workflowDir) splits the README
// ABOUTME: (definition) root from the entity root via the README's `state:` field.
package status

import (
	"fmt"
	"path/filepath"
	"strings"
)

// roots carries the two directory roles the status units consume. In single-root
// mode definitionDir == entityDir == workflowDir, matching the oracle's
// os.path.join(workflow_dir, 'README.md') and same-directory entity scan. A
// `state:` field in the README diverges entityDir to definitionDir/<state>.
//
// The oracle relies on the process cwd to resolve a relative --workflow-dir; an
// in-process runner cannot chdir safely, so each role carries two spellings: the
// absolute path used for filesystem I/O (joined with the request's working dir)
// and the literal spelling the operator passed (preserved in output, e.g. the
// `archived: ./_archive/...` dest and the validation `workflow=` evidence).
type roots struct {
	// definitionDir is the absolute README root for I/O.
	definitionDir string
	// entityDir is the absolute entity root for I/O.
	entityDir string
	// definitionDirSpelling / entityDirSpelling are the as-passed spellings used
	// in output so a relative --workflow-dir renders relative dests, matching the
	// oracle's literal os.path.join(pipeline_dir, ...) output.
	definitionDirSpelling string
	entityDirSpelling     string
}

// resolveRoots returns the definition and entity roots for a workflow dir,
// resolving relative spellings against baseDir for I/O while preserving the
// literal spelling for output. The definition dir always holds the README; the
// entity dir is definitionDir/<state> when the README frontmatter carries a
// non-empty `state:` field, else the definition dir itself. An absolute `state:`
// value, or one that escapes the definition dir via `..`, is rejected with an
// error rather than silently followed — the v0 contract is a child checkout.
func resolveRoots(workflowDir, baseDir string) (roots, error) {
	abs := workflowDir
	if workflowDir == "" {
		abs = baseDir
	} else if !filepath.IsAbs(workflowDir) && baseDir != "" {
		abs = filepath.Join(baseDir, workflowDir)
	}

	r := roots{
		definitionDir:         abs,
		entityDir:             abs,
		definitionDirSpelling: workflowDir,
		entityDirSpelling:     workflowDir,
	}

	stateValue := strings.TrimSpace(ParseFrontmatter(filepath.Join(abs, "README.md"))["state"])
	if stateValue == "" {
		return r, nil
	}
	if filepath.IsAbs(stateValue) {
		return roots{}, fmt.Errorf("state: must be a path relative to the workflow README directory, not absolute: %s", stateValue)
	}
	cleaned := filepath.Clean(stateValue)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return roots{}, fmt.Errorf("state: must not escape the workflow README directory: %s", stateValue)
	}

	r.entityDir = filepath.Join(abs, cleaned)
	r.entityDirSpelling = pyJoin(spellingOr(workflowDir, abs), stateValue)
	return r, nil
}

// spellingOr returns the as-passed spelling when non-empty, else the absolute
// fallback, so a state-checkout dest still renders coherently when the workflow
// dir was derived from baseDir rather than passed explicitly.
func spellingOr(spelling, fallback string) string {
	if spelling != "" {
		return spelling
	}
	return fallback
}
