// ABOUTME: POSIX shell quoting matching Python shlex.quote, for the emitted
// ABOUTME: show-stage-def fetch line whose --workflow-dir / --stage args may have spaces.
package dispatch

import "strings"

// shlexQuote returns s quoted for safe use as a single POSIX shell token,
// matching Python's shlex.quote: an empty string becomes ”; a string composed
// only of the shell-safe set [A-Za-z0-9_@%+=:,./-] is returned unchanged; any
// other string is wrapped in single quotes with embedded single quotes escaped
// as the '"'"' idiom.
func shlexQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !containsUnsafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// containsUnsafe reports whether s has any byte outside shlex.quote's safe set
// (Python's _find_unsafe = re.compile(r'[^\w@%+=:,./-]', re.ASCII); \w in ASCII
// mode is [A-Za-z0-9_]).
func containsUnsafe(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case '_', '@', '%', '+', '=', ':', ',', '.', '/', '-':
			continue
		}
		return true
	}
	return false
}
