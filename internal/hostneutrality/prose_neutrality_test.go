// ABOUTME: AC-3 prose-side host-neutrality oracle — a markdown-structure parse of
// ABOUTME: first-officer-shared-core.md flagging unqualified Claude-only helper tokens.
package hostneutrality

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sharedCorePath is the generic FO operating-contract core the prose oracle
// polices. Relative to this test's package dir (go test cwd = package source dir).
var sharedCorePath = filepath.Join("..", "..", "skills", "first-officer", "references",
	"first-officer-shared-core.md")

// claudeRuntimePath is the Claude adapter the relocated commands move into. The
// oracle does NOT police it — Claude-only tokens are expected there.
var claudeRuntimePath = filepath.Join("..", "..", "skills", "first-officer", "references",
	"claude-first-officer-runtime.md")

// claudeHelperTokens are the named Claude-runtime helper commands / functions that
// must not appear UNQUALIFIED in the generic core. Each is a Claude-only mechanism
// (a binary subcommand or a ~/.claude-reading helper function), not a cross-runtime
// capability. A bare mention inside a host-qualified span is allowed; an unqualified
// algorithm step that names one fails.
var claudeHelperTokens = []string{
	"claude-team",
	"context-budget",
	"spawn-standing",
	"list-standing",
	"show-standing",
	"member_exists",
	"lookup_model",
}

// hostQualifierMarkers mark a span as explicitly host-qualified: it names a
// non-Claude runtime alternative (the `X on Codex, Y on Claude` shape) so the
// Claude token is a qualified realization, not an unqualified generic step.
var hostQualifierMarkers = []string{"Codex", "codex"}

// TestSharedCoreHasNoUnqualifiedClaudeHelpers parses the generic core's markdown
// structure into spans (numbered/bulleted list items and paragraphs) and fails if
// any Claude-only helper token sits in a span lacking a host qualifier. This is a
// structural parse, NOT a substring count: re-introducing the line-142 unqualified
// `claude-team context-budget` reuse step makes it fail; the line-207
// `send_input on Codex, SendMessage on Claude teams` span passes (host-qualified).
func TestSharedCoreHasNoUnqualifiedClaudeHelpers(t *testing.T) {
	spans := parseSpans(t, sharedCorePath)
	var violations []string
	for _, sp := range spans {
		for _, tok := range claudeHelperTokens {
			if !strings.Contains(sp.text, tok) {
				continue
			}
			if spanHostQualified(sp.text) {
				continue
			}
			violations = append(violations,
				strings.TrimSpace(
					sp.text[:min(len(sp.text), 120)])+
					" [unqualified token: "+tok+", first line "+itoa(sp.startLine)+"]")
		}
	}
	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("unqualified Claude-only helper in generic core: %s", v)
		}
		t.Fatalf("found %d unqualified Claude-runtime helper reference(s) in %s; relocate them to %s",
			len(violations), sharedCorePath, claudeRuntimePath)
	}
}

// TestClaudeAdapterOwnsRelocatedCommands confirms the relocation landed: the
// concrete Claude invocation of each relocated capability lives in the Claude
// adapter. This is the other half of the invariant — the commands did not vanish,
// they moved. Asserts the adapter names the four subcommand surfaces.
func TestClaudeAdapterOwnsRelocatedCommands(t *testing.T) {
	data, err := os.ReadFile(claudeRuntimePath)
	if err != nil {
		t.Fatalf("read %s: %v", claudeRuntimePath, err)
	}
	body := string(data)
	for _, want := range []string{"context-budget", "list-standing", "spawn-standing", "show-standing"} {
		if !strings.Contains(body, want) {
			t.Errorf("Claude adapter %s does not name the relocated command %q", claudeRuntimePath, want)
		}
	}
}

// span is one markdown block: a list item or a paragraph, with its first line
// number for diagnostics. A list item spans its bullet line plus indented
// continuation lines; a paragraph spans consecutive non-blank, non-heading lines.
type span struct {
	text      string
	startLine int
}

// parseSpans reads a markdown file and groups its lines into spans. Headings
// (lines starting with #) are span boundaries and are not themselves spans —
// section ledes that follow a heading are their own paragraph span. Fenced code
// blocks (```) are kept whole as one span so a token inside a code example is
// attributed to the enclosing block, not split.
func parseSpans(t *testing.T, path string) []span {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var spans []span
	var cur []string
	curStart := 0
	inFence := false
	lineNo := 0

	flush := func() {
		if len(cur) > 0 {
			spans = append(spans, span{text: strings.Join(cur, "\n"), startLine: curStart})
			cur = nil
		}
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			// Toggle fence; keep the fence and its body in the current span so a
			// token inside an example stays attributed to its block.
			if len(cur) == 0 {
				curStart = lineNo
			}
			cur = append(cur, line)
			inFence = !inFence
			continue
		}
		if inFence {
			cur = append(cur, line)
			continue
		}

		if trimmed == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			// Heading is a boundary; do not include it in any span.
			flush()
			continue
		}

		isListItem := isListItemStart(line)
		isIndentCont := len(line) > 0 && (line[0] == ' ' || line[0] == '\t')

		if isListItem && len(cur) > 0 && !isIndentCont {
			// A new top-level list item starts a new span unless it is an indented
			// continuation of the prior one.
			flush()
		}
		if len(cur) == 0 {
			curStart = lineNo
		}
		cur = append(cur, line)
	}
	flush()
	if err := sc.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return spans
}

// isListItemStart reports whether line begins a markdown list item: an optional
// indent then `- `, `* `, or `N.` / `N) ` (ordered). Indented items are still
// list-item starts; the caller distinguishes continuation by leading whitespace.
func isListItemStart(line string) bool {
	t := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
		return true
	}
	// Ordered: leading digits then `.` or `)` then space (or end).
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(t) {
		return false
	}
	return (t[i] == '.' || t[i] == ')')
}

// spanHostQualified reports whether a span is explicitly host-qualified — it names
// a non-Claude runtime alternative, the `X on Codex, Y on Claude` shape. Such a
// span may reference a Claude helper because it is presenting the Claude
// realization alongside the alternative, not stating an unqualified generic step.
func spanHostQualified(text string) bool {
	for _, m := range hostQualifierMarkers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// itoa is a tiny int-to-string for diagnostics (avoids importing strconv twice).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
