// ABOUTME: pyRepr renders a string the way Python's repr() does, for the
// ABOUTME: malformed-heading diagnostic which embeds the raw heading via {raw!r}.
package dispatch

import "strings"

// pyRepr renders s the way Python str.__repr__ does for the printable heading
// text this is used on: prefer single-quote delimiters; switch to double quotes
// when s contains a single quote but no double quote; otherwise use single
// quotes and backslash-escape any single quotes. Backslashes are escaped in all
// cases. Headings are printable Markdown text, so the non-printable \xNN / tab /
// newline escapes Python applies to control characters are out of scope here —
// a malformed stage heading is a one-line `### ` heading, never a control char.
func pyRepr(s string) string {
	quote := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, "\"") {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\':
			b.WriteString(`\\`)
		case c == quote:
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte(quote)
	return b.String()
}
