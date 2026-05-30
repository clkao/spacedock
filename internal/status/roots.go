// ABOUTME: Root-resolution seam — resolveRoots(workflowDir) splits the README
// ABOUTME: (definition) root from the entity root; both equal workflowDir for now.
package status

import "path/filepath"

// roots carries the two directory roles the status units consume. In this stage
// definitionDir == entityDir == workflowDir (single-root), matching the oracle's
// os.path.join(workflow_dir, 'README.md') and same-directory entity scan. The
// split is threaded now so native-state-dir (Stage 6) becomes a one-function
// change to resolveRoots rather than a call-site-wide retrofit.
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
// literal spelling for output. Both roles equal workflowDir in this stage;
// Stage 6 makes definitionDir the README dir and entityDir the state: path with
// no other call-site change.
func resolveRoots(workflowDir, baseDir string) roots {
	abs := workflowDir
	if workflowDir == "" {
		abs = baseDir
	} else if !filepath.IsAbs(workflowDir) && baseDir != "" {
		abs = filepath.Join(baseDir, workflowDir)
	}
	return roots{
		definitionDir:         abs,
		entityDir:             abs,
		definitionDirSpelling: workflowDir,
		entityDirSpelling:     workflowDir,
	}
}
