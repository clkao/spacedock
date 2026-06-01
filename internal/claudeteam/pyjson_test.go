// ABOUTME: unit table for EmitPythonJSON / escapeNonASCII — the ensure_ascii
// ABOUTME: parity escaping that makes native JSON byte-match Python json.dumps.
package claudeteam

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// TestEscapeNonASCIIMatchesPythonEnsureASCII drives the escape helper over the
// chars json.dumps escapes that Go's encoder does not — DEL and every non-ASCII
// rune (BMP accents/dashes/arrows + a surrogate-pair astral emoji) — plus the
// chars both leave alone (plain ASCII, the encoder's own \t/\n escapes, and the
// HTML-significant < > & that ensure_ascii does NOT touch). Each input's native
// emission must byte-match the live python3 json.dumps, so the table can't drift
// from json.dumps' actual ensure_ascii=True behavior. Inputs are written with Go
// escapes so this source file stays pure ASCII.
func TestEscapeNonASCIIMatchesPythonEnsureASCII(t *testing.T) {
	inputs := []string{
		"plain",
		"em—dash",          // em-dash U+2014
		"smart“q”",         // curly quotes
		"arrow→there",      // rightwards arrow
		"café",             // accented e
		"rocket\U0001f680", // astral emoji -> surrogate pair
		"del\x7fx",         // DEL: ASCII but json.dumps escapes it, Go's encoder does not
		"nbsp x",           // non-breaking space
		"tab\tnl\n",        // encoder already escapes these; helper leaves them
		"less<more>amp&",   // HTML chars stay raw under ensure_ascii
	}
	for _, in := range inputs {
		got := emitJSONString(t, in)
		want := pythonJSONDumps(t, in)
		if got != want {
			t.Errorf("emit %q\n  native = %s\n  python = %s", in, got, want)
		}
	}
}

// emitJSONString runs a bare string value through EmitPythonJSON and returns the
// emitted JSON with the trailing newline trimmed, so the comparison matches the
// json.dumps form (no indent on a scalar, no trailing newline).
func emitJSONString(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	if EmitPythonJSON(&buf, s) != 0 {
		t.Fatalf("EmitPythonJSON returned non-zero for %q", s)
	}
	return strings.TrimRight(buf.String(), "\n")
}

// pythonJSONDumps shells to python3 to compute json.dumps(s) for the cross-check.
func pythonJSONDumps(t *testing.T, s string) string {
	t.Helper()
	cmd := exec.Command("python3", "-c", "import json,sys; sys.stdout.write(json.dumps(sys.stdin.read()))")
	cmd.Stdin = strings.NewReader(s)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3 json.dumps failed: %v", err)
	}
	return string(out)
}
