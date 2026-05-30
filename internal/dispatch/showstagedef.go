// ABOUTME: show-stage-def extracts a workflow README's ### {stage} subsection,
// ABOUTME: matching the oracle's extract_stage_subsection + cmd_show_stage_def.
package dispatch

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// headingDecorationChars are the inline-markdown decoration characters stripped
// from a heading before tokenizing, matching _HEADING_DECORATION_CHARS.
const headingDecorationChars = "`*_~"

// headingTokens returns the content tokens of a `### `-prefixed heading, or nil
// when the line is not such a heading. Strips inline decoration (“ ` “, `*`,
// `_`, `~`) and treats `(` and `[` as token terminators so trailing annotations
// like `*(captain-interactive)*` or `[terminal]` do not merge with the name.
// Matches _heading_tokens.
func headingTokens(line string) []string {
	stripped := strings.TrimSpace(line)
	if !strings.HasPrefix(stripped, "### ") {
		return nil
	}
	rest := strings.TrimSpace(stripped[4:])
	rest = stripDecoration(rest)
	rest = strings.ReplaceAll(rest, "(", " ")
	rest = strings.ReplaceAll(rest, "[", " ")
	return strings.Fields(rest)
}

// stripDecoration removes every decoration character from s (str.translate with
// a delete table).
func stripDecoration(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(headingDecorationChars, r) {
			return -1
		}
		return r
	}, s)
}

// headingFirstToken returns the first content token of a `### ` heading, or ""
// when the line is not a heading or carries no tokens. Matches
// _heading_first_token (which returns None; "" is the no-match sentinel here).
func headingFirstToken(line string) string {
	tokens := headingTokens(line)
	if len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}

// stageHeadingError is the parser diagnostic raised when a `###` heading
// mentions the stage name as a token but the stage name is not the first token.
// It carries the oracle's ValueError text so cmd-level wrappers can prefix it.
type stageHeadingError struct{ msg string }

func (e *stageHeadingError) Error() string { return e.msg }

// extractStageSubsection returns the full ### {stage} subsection from a workflow
// README. Heading match is permissive: any `###` line whose first content token
// (after stripping “ ` “, `*`, `_`, `~` and treating `(` / `[` as token
// terminators) equals stage is a match. When no permissive match is found but
// some `###` line mentions the stage name as a token, a *stageHeadingError is
// returned surfacing the malformed heading. Returns ("", nil) only when no
// `###` line mentions the stage name at all (genuinely missing stage). Matches
// extract_stage_subsection.
func extractStageSubsection(readmePath, stage string) (string, error) {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return "", err
	}
	lines := splitTextLines(string(data))

	start := -1
	for i, line := range lines {
		if headingFirstToken(line) == stage {
			start = i
			break
		}
	}
	if start < 0 {
		for i, line := range lines {
			tokens := headingTokens(line)
			if len(tokens) > 0 && containsToken(tokens, stage) {
				stripped := strings.TrimSpace(line)
				return "", &stageHeadingError{msg: fmt.Sprintf(
					"stage heading at line %d mentions '%s' "+
						"but does not parse as a stage heading: %s. "+
						"The stage name must be the first content token of the "+
						"heading after stripping Markdown decoration "+
						"(backticks, *, _, ~) and treating '(' and '[' as "+
						"token terminators.",
					i+1, stage, pyRepr(stripped))}
			}
		}
		return "", nil
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		stripped := strings.TrimSpace(lines[i])
		if strings.HasPrefix(stripped, "### ") || strings.HasPrefix(stripped, "## ") {
			end = i
			break
		}
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n"), nil
}

// containsToken reports whether token is in tokens, matching Python's
// `stage_name in tokens` membership test.
func containsToken(tokens []string, token string) bool {
	for _, t := range tokens {
		if t == token {
			return true
		}
	}
	return false
}

// lineBoundary reports whether r is one of the boundaries Python str.splitlines()
// breaks on: LF, CR (and CRLF, handled by the scanner), VT, FF, FS, GS, RS, NEL
// (U+0085), LS (U+2028), PS (U+2029). The oracle reads the README in universal-
// newline text mode then calls splitlines(); the CR-family translation is
// subsumed here since splitlines() itself breaks on CR/CRLF, so a direct
// splitlines() gives the identical result.
func lineBoundary(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', '\x1c', '\x1d', '\x1e', '\u0085', '\u2028', '\u2029':
		return true
	}
	return false
}

// splitTextLines splits text into lines exactly as Python's str.splitlines()
// does: a boundary terminates the current line (CRLF counts as one boundary),
// and the trailing line is dropped when the text ends in a boundary (no empty
// final element). An empty input yields no lines.
func splitTextLines(text string) []string {
	if text == "" {
		return nil
	}
	var lines []string
	var cur []rune
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if !lineBoundary(r) {
			cur = append(cur, r)
			continue
		}
		lines = append(lines, string(cur))
		cur = nil
		// CRLF is a single boundary: consume the following LF after a CR.
		if r == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			i++
		}
	}
	if len(cur) > 0 {
		lines = append(lines, string(cur))
	}
	return lines
}

// runShowStageDef emits the README's ### {stage} subsection on stdout. Exit 0
// with stdout on success; exit 1 with a parser diagnostic on a malformed or
// missing heading or an unresolvable workflow dir / README. Matches
// cmd_show_stage_def.
func runShowStageDef(workflowDir, stage string, stdout, stderr io.Writer) int {
	if info, err := os.Stat(workflowDir); err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "error: workflow directory not found: %s\n", workflowDir)
		return 1
	}
	readmePath := filepath.Join(workflowDir, "README.md")
	if !isFile(readmePath) {
		fmt.Fprintf(stderr, "error: workflow README not found at '%s'\n", readmePath)
		return 1
	}

	subsection, err := extractStageSubsection(readmePath, stage)
	if err != nil {
		if she, ok := err.(*stageHeadingError); ok {
			fmt.Fprintf(stderr, "error: %s\n", she.msg)
			return 1
		}
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}
	if subsection == "" {
		fmt.Fprintf(stderr, "error: stage '%s' heading not found in %s\n", stage, readmePath)
		return 1
	}
	fmt.Fprintln(stdout, subsection)
	return 0
}
