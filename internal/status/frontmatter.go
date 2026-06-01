// ABOUTME: Line-oriented frontmatter parser matching the oracle's
// ABOUTME: _has_opening_fence + parse_frontmatter — no YAML dependency.
package status

import (
	"os"
	"strings"
)

// utf8BOM is the leading byte-order mark stripped from a file's first line
// before the opening-fence check, matching the oracle's BOM handling.
const utf8BOM = "\uFEFF"

// hasOpeningFence reports whether the file's first non-empty, non-BOM line is
// exactly `---`. Leading truly-empty lines (`\n` only) are skipped; a
// whitespace-only first content line disqualifies the file. A leading UTF-8 BOM
// on the first line is stripped before the check. Matches _has_opening_fence.
func hasOpeningFence(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return contentHasOpeningFence(data)
}

// contentHasOpeningFence is hasOpeningFence over in-memory bytes (for --new,
// which validates STDIN before any file exists).
func contentHasOpeningFence(data []byte) bool {
	first := true
	for _, raw := range splitLines(string(data)) {
		line := raw
		if first {
			line = strings.TrimPrefix(line, utf8BOM)
			first = false
		}
		if line == "" {
			continue
		}
		return line == "---"
	}
	return false
}

// normalizeNewlines translates CRLF and lone CR into LF, matching the oracle's
// Python text-mode universal-newlines read (open(path, 'r')). Applied at every
// content-read boundary so a `---\r` fence compares equal to `---` and a CRLF
// README's stages block parses. `\r\n` is collapsed first so a CRLF pair never
// yields two LFs.
func normalizeNewlines(s string) string {
	if !strings.ContainsRune(s, '\r') {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// splitLines splits on '\n' the way Python's `for line in f` iteration does
// after rstrip('\n'): the trailing newline is removed from each line. A file
// ending in '\n' yields no extra empty trailing element here (file iteration
// stops at EOF), unlike strings.Split which would. We replicate file iteration:
// split on '\n' and drop a single trailing empty element only when the input
// ended with '\n'. Newlines are normalized first (universal newlines).
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(normalizeNewlines(s), "\n")
	// Python file iteration over a trailing-newline file does not yield a final
	// empty line; strings.Split does. Drop that final empty element so the parse
	// loops see the same lines the oracle does.
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// ParseFrontmatter extracts top-level key/value pairs between the first and
// second `---` fences. Matches parse_frontmatter: split on the first `:`, strip
// key and value, strip matched surrounding quotes (len>=2, same char at both
// ends, `"` or `'`), ignore nested/indented lines, last top-level key wins,
// empty values yield "". Returns an empty map when there is no opening fence.
func ParseFrontmatter(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	return parseFrontmatterContent(data)
}

// parseFrontmatterContent is ParseFrontmatter over in-memory bytes.
func parseFrontmatterContent(data []byte) map[string]string {
	fields := map[string]string{}
	if !contentHasOpeningFence(data) {
		return fields
	}
	inFM := false
	first := true
	for _, raw := range splitLines(string(data)) {
		line := raw
		if first {
			line = strings.TrimPrefix(line, utf8BOM)
			first = false
		}
		if line == "---" {
			if inFM {
				break
			}
			inFM = true
			continue
		}
		if !inFM {
			continue
		}
		if !strings.Contains(line, ":") {
			continue
		}
		// Indented lines (first char is whitespace) are ignored — only
		// top-level key: value pairs become fields.
		if len(line) > 0 && isSpaceByte(line[0]) {
			continue
		}
		key, val, _ := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		fields[key] = parseValue(val)
	}
	return fields
}

// parseValue resolves a frontmatter value: it trims surrounding whitespace,
// then either unwraps a quoted scalar (dropping any trailing inline comment
// after the close quote) or strips an inline comment from an unquoted scalar.
// A quoted scalar protects an interior `#` (it is literal, not a comment); an
// unquoted scalar drops a whitespace-preceded `#…` per the YAML rule (an
// unspaced `v1.0#163`-style token is kept). Matches the oracle's parse_value.
func parseValue(raw string) string {
	val := strings.TrimSpace(raw)
	if len(val) > 0 && (val[0] == '"' || val[0] == '\'') {
		q := val[0]
		if j := strings.IndexByte(val[1:], q); j >= 0 {
			closeAt := j + 1
			rest := strings.TrimLeft(val[closeAt+1:], " \t")
			if rest == "" || rest[0] == '#' {
				return val[1:closeAt]
			}
		}
		// Unterminated or trailing non-comment content: preserve the historical
		// matched-surrounding-quotes behavior, no comment strip.
		return stripMatchedQuotes(val)
	}
	return stripInlineComment(val)
}

// stripMatchedQuotes removes a single pair of matched surrounding quotes from a
// value that is >= 2 chars and identically quoted at both ends with `"` or `'`.
func stripMatchedQuotes(val string) string {
	if len(val) >= 2 && val[0] == val[len(val)-1] && (val[0] == '"' || val[0] == '\'') {
		return val[1 : len(val)-1]
	}
	return val
}

// isSpaceByte reports whether b is one of the ASCII whitespace bytes Python's
// str.isspace treats as space for the leading-char check used here.
func isSpaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}
