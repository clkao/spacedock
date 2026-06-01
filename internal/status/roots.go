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
	// entityDirSpelling is the as-passed spelling used in output so a relative
	// --workflow-dir renders relative dests, matching the oracle's literal
	// os.path.join(pipeline_dir, ...) output.
	entityDirSpelling string
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
		definitionDir:     abs,
		entityDir:         abs,
		entityDirSpelling: workflowDir,
	}

	stateValue := ParseFrontmatter(filepath.Join(abs, "README.md"))["state"]
	mode, relPath, err := ClassifyState(stateValue)
	if err != nil {
		return roots{}, err
	}
	if mode == StateInline {
		return r, nil
	}

	r.entityDir = filepath.Join(abs, relPath)
	r.entityDirSpelling = PyJoin(spellingOr(workflowDir, abs), relPath)
	return r, nil
}

// mergePolicy is a workflow's declared merge policy, read from the README's
// top-level `merge:` key. It tells the terminal-transition guard whether a PR is
// expected at the merge boundary (mergePR) or the workflow merges locally
// (mergeLocal). Under mergeLocal the guard exempts the pr-requirement of the
// merge-hook invariant; it never relaxes the ceremony structure.
type mergePolicy int

const (
	mergePR mergePolicy = iota
	mergeLocal
)

// resolveMergePolicy reads the README's top-level `merge:` key and returns the
// declared policy. An absent or empty key defaults to mergePR — byte-identical to
// a workflow that never declared the key. An unknown value is rejected loudly
// rather than silently coerced to mergePR, so a typo (`merge: locl`) fails fast
// instead of silently demanding a PR forever. Matches the oracle's
// resolve_merge_policy.
func resolveMergePolicy(definitionDir string) (mergePolicy, error) {
	value := strings.TrimSpace(ParseFrontmatter(filepath.Join(definitionDir, "README.md"))["merge"])
	switch value {
	case "", "pr":
		return mergePR, nil
	case "local":
		return mergeLocal, nil
	default:
		// Single-quote the bad value to match the oracle's {value!r} repr.
		return mergePR, fmt.Errorf("README merge: must be 'local' or 'pr' (or absent for the default 'pr'), not '%s'", value)
	}
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
