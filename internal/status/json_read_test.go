// ABOUTME: AC-1 --json read parity — glyph-free, byte-deterministic, table-round-
// ABOUTME: tripping JSON for the env-independent reads; --boot structural+normalized.
package status

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// jsonReadCases are the deterministic, full-value reads whose --json output is
// captured to a golden and asserted byte-stable + glyph-free + round-tripping.
var jsonReadCases = []struct {
	name   string
	golden string
	extra  []string
}{
	{name: "default", golden: "seq-default.json", extra: nil},
	{name: "archived", golden: "seq-archived.json", extra: []string{"--archived"}},
	{name: "next", golden: "seq-next.json", extra: []string{"--next"}},
	{name: "where", golden: "seq-where.json", extra: []string{"--where", "status=ideation"}},
	{name: "resolve", golden: "seq-resolve.json", extra: []string{"--resolve", "003-wire-cli"}},
}

// TestJSONReadGolden (AC-1 oracle a + b + c) captures/compares the --json output
// per deterministic read command, asserts it is byte-stable run-to-run, and
// asserts it carries no aligned-column padding run and no ellipsis glyph — the
// properties that survive a token proxy intact.
func TestJSONReadGolden(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	for _, tc := range jsonReadCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--workflow-dir", root, "--json"}, tc.extra...)

			out, errOut, code := runNative(t, root, env, args...)
			if code != 0 {
				t.Fatalf("native exit=%d stderr=%q", code, errOut)
			}
			got := normalize(out, root)

			if *update {
				writeGolden(t, tc.golden, got)
				return
			}
			if want := readGolden(t, tc.golden); got != want {
				t.Fatalf("--json %s golden mismatch\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}

			// (b) determinism: a second run is byte-identical.
			out2, _, _ := runNative(t, root, env, args...)
			if out != out2 {
				t.Fatalf("--json %s not byte-stable across runs\nrun1: %q\nrun2: %q", tc.name, out, out2)
			}

			// (c) glyph-free: no two-space alignment run, no ellipsis.
			if strings.Contains(out, "  ") {
				t.Fatalf("--json %s contains an aligned-padding run (two spaces): %q", tc.name, out)
			}
			if strings.Contains(out, "…") {
				t.Fatalf("--json %s contains an ellipsis glyph: %q", tc.name, out)
			}
			// One compact document: exactly one trailing newline, no interior ones.
			if !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 1 {
				t.Fatalf("--json %s is not a single newline-terminated document: %q", tc.name, out)
			}
		})
	}
}

// statusEnvelope mirrors the {"command","entities":[...]} status/where/archived
// shape for the round-trip parity walk.
type statusEnvelope struct {
	Command  string              `json:"command"`
	Entities []map[string]string `json:"entities"`
}

// defaultColumnWidths are the padRight min-widths printStatusTable uses for the
// no-extras default row, in defaultStatusFields order. The final column (source)
// is unpadded, so it has no width entry — the cursor walk treats it as the row
// tail.
var defaultColumnWidths = []int{6, 30, 20, 30, 8}

// TestJSONStatusRoundTripsTableColumns (AC-1 oracle d) walks the parsed --json
// entities against the default table the same run renders: each JSON value must
// equal the corresponding default (full-value) table column AT ITS OWN COLUMN,
// not merely appear somewhere in the row. The table prints default columns
// untruncated (padRight is a min-width, not a cap), so each value starts its
// column verbatim; a left-to-right cursor walk pins every value to its column
// boundary, so a value cannot pass by coincidentally matching another column's
// text (the weakness of a whole-row Contains).
func TestJSONStatusRoundTripsTableColumns(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "seq-workflow"))
	if err != nil {
		t.Fatal(err)
	}
	env := pinnedEnv(t)

	jsonOut, _, jc := runNative(t, root, env, "--workflow-dir", root, "--json")
	if jc != 0 {
		t.Fatalf("--json exit=%d", jc)
	}
	tableOut, _, tc := runNative(t, root, env, "--workflow-dir", root)
	if tc != 0 {
		t.Fatalf("table exit=%d", tc)
	}

	var env1 statusEnvelope
	if err := json.Unmarshal([]byte(jsonOut), &env1); err != nil {
		t.Fatalf("parse --json: %v\n%s", err, jsonOut)
	}
	if env1.Command != "status" {
		t.Fatalf("command = %q, want status", env1.Command)
	}

	// Map each table data row by its slug (the second column), so we can match the
	// JSON entity to its rendered row and walk that row column-by-column.
	tableRows := map[string]string{}
	for _, line := range strings.Split(tableOut, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] == "ID" || fields[0] == "--" {
			continue
		}
		tableRows[fields[1]] = line // fields[1] is the slug column
	}

	if len(env1.Entities) == 0 {
		t.Fatal("no entities in --json output")
	}
	for _, e := range env1.Entities {
		slug := e["slug"]
		row, ok := tableRows[slug]
		if !ok {
			t.Fatalf("JSON entity slug %q has no table row\ntable:\n%s", slug, tableOut)
		}
		assertRowColumnsMatchJSON(t, row, e, slug)
	}
}

// assertRowColumnsMatchJSON walks row left-to-right and asserts each default
// column begins with the JSON value for that field. The cursor advances by the
// column's rendered width (max of padRight min-width and the value's rune length,
// since padRight does not truncate) plus the single-space separator, so each
// value is checked against ITS column position rather than the whole row. The
// final column (source) is unpadded and consumes the row tail.
func assertRowColumnsMatchJSON(t *testing.T, row string, e map[string]string, slug string) {
	t.Helper()
	runes := []rune(row)
	cursor := 0
	for i, field := range defaultStatusFields {
		val := e[field]
		valRunes := []rune(val)
		if cursor > len(runes) {
			t.Fatalf("row exhausted before column %q for slug %q\nrow: %q", field, slug, row)
		}
		seg := runes[cursor:]
		if !strings.HasPrefix(string(seg), val) {
			t.Fatalf("JSON %s=%q for slug %q not at its column position (col %d)\nrow: %q\ncolumn tail: %q",
				field, val, slug, i, row, string(seg))
		}
		if i == len(defaultColumnWidths) {
			// Last field (source) is unpadded — it is the row tail; nothing follows.
			continue
		}
		width := defaultColumnWidths[i]
		colLen := len(valRunes)
		if colLen < width {
			colLen = width
		}
		cursor += colLen + 1 // advance past the padded column and the space separator
	}
}
