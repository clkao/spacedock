// ABOUTME: EmitPythonJSON renders a value byte-identically to Python
// ABOUTME: json.dumps(v, indent=2) + print(), including its ensure_ascii escaping.
package claudeteam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// EmitPythonJSON writes v as two-space-indented JSON with a trailing newline,
// byte-matching Python json.dumps(v, indent=2) followed by print(). Go's encoder
// emits raw UTF-8 and leaves DEL unescaped, where json.dumps defaults to
// ensure_ascii=True; escapeNonASCII closes that gap so non-ASCII content
// (em-dashes, smart quotes, arrows, accents, emoji) is byte-identical to the
// oracle. HTML escaping is off, matching json.dumps' default of not escaping
// <, >, &. Returns 1 on an encode error, 0 on success.
func EmitPythonJSON(w io.Writer, v any) int {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return 1
	}
	if _, err := w.Write(EscapeNonASCII(buf.Bytes())); err != nil {
		return 1
	}
	return 0
}

// EscapeNonASCII rewrites every rune in already-encoded JSON that Python's
// json.dumps escapes but Go's encoder does not — DEL (U+007F) and every
// non-ASCII rune (>= U+0080) — into a \uXXXX escape, using a UTF-16 surrogate
// pair for runes above the BMP, exactly as json.dumps with ensure_ascii=True
// does. Every other byte (the encoder's ASCII output, including its own \uXXXX
// control escapes and \t / \n / \" / \\) passes through untouched: the Go
// encoder never emits a byte >= 0x80 except as part of a raw non-ASCII rune, so
// scanning runes and re-escaping the high ones cannot corrupt an existing escape.
// Exported so callers that hand-format a compact JSON object (the spawn-standing
// already-alive line, which Python emits without indent) can escape a marshaled
// string field through the same single routine EmitPythonJSON uses.
func EscapeNonASCII(in []byte) []byte {
	// Fast path: pure-ASCII output (DEL aside) needs no rewrite, which is the
	// common case — most dispatch content is ASCII.
	if !bytes.ContainsFunc(in, func(r rune) bool { return r == 0x7f || r >= 0x80 }) {
		return in
	}
	var out bytes.Buffer
	out.Grow(len(in))
	for i := 0; i < len(in); {
		r, size := utf8.DecodeRune(in[i:])
		if r == 0x7f || r >= 0x80 {
			if r > 0xffff {
				v := r - 0x10000
				hi := 0xd800 + (v >> 10)
				lo := 0xdc00 + (v & 0x3ff)
				fmt.Fprintf(&out, "\\u%04x\\u%04x", hi, lo)
			} else {
				fmt.Fprintf(&out, "\\u%04x", r)
			}
		} else {
			out.WriteByte(in[i])
		}
		i += size
	}
	return out.Bytes()
}
