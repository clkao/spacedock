// ABOUTME: realpathOf mirrors Python os.path.realpath — canonicalize and
// ABOUTME: resolve symlinks, returning a best-effort absolute path.
package status

import (
	"os"
	"path/filepath"
)

// realpathOf resolves symlinks the way os.path.realpath does: it returns the
// canonical absolute path, resolving symlinks in the existing prefix and
// appending the remaining (non-existent) components. EvalSymlinks alone errors
// on a missing leaf, so fall back to abs-cleaning when full resolution fails.
func realpathOf(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	// Resolve the longest existing ancestor, then re-attach the rest. This
	// matches os.path.realpath's behavior for paths whose leaf does not exist.
	dir := abs
	var trailing []string
	for {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			parts := append([]string{resolved}, reversed(trailing)...)
			return filepath.Join(parts...)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		trailing = append(trailing, filepath.Base(dir))
		dir = parent
	}
	return filepath.Clean(abs)
}

func reversed(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

// fileExists reports whether path exists (file or dir), following symlinks.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// findGitRoot walks up from startDir to the directory containing a .git entry
// (dir or file), returning startDir when none is found. Matches find_git_root.
func findGitRoot(startDir string) string {
	d, err := filepath.Abs(startDir)
	if err != nil {
		d = startDir
	}
	for {
		gitPath := filepath.Join(d, ".git")
		if st, err := os.Stat(gitPath); err == nil && (st.IsDir() || st.Mode().IsRegular()) {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return startDir
		}
		d = parent
	}
}
