// ABOUTME: Root-resolution seam — resolveRoots(workflowDir) splits the README
// ABOUTME: (definition) root from the entity root; both equal workflowDir for now.
package status

// roots carries the two directory roles the status units consume. In this stage
// definitionDir == entityDir == workflowDir (single-root), matching the oracle's
// os.path.join(workflow_dir, 'README.md') and same-directory entity scan. The
// split is threaded now so native-state-dir (Stage 6) becomes a one-function
// change to resolveRoots rather than a call-site-wide retrofit.
type roots struct {
	// definitionDir feeds the stage parser (README stages block) and the
	// identity engine (sd-b32 workflow realpath / id-style from the README).
	definitionDir string
	// entityDir feeds discovery, mutation, archive (entityDir/_archive), and
	// the entity side of validation.
	entityDir string
}

// resolveRoots returns the definition and entity roots for a workflow dir. Both
// equal workflowDir in this stage; Stage 6 makes definitionDir the README dir
// and entityDir the state: path with no other call-site change.
func resolveRoots(workflowDir string) roots {
	return roots{definitionDir: workflowDir, entityDir: workflowDir}
}
