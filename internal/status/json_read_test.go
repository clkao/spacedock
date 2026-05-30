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

// TestJSONStatusRoundTripsTableColumns (AC-1 oracle d) walks the parsed --json
// entities against the default table the same run renders: each JSON value must
// equal the corresponding default (full-value) table column. The table prints
// default columns untruncated (padRight is a min-width, not a cap), so the JSON
// value appears verbatim in that entity's table row.
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

	// Map each table data row by its slug (column 2), so we can match the JSON
	// entity to its rendered row and assert each value appears in that row.
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
		for _, field := range defaultStatusFields {
			val := e[field]
			if val == "" {
				continue // empty score etc. — nothing to find in the row
			}
			if !strings.Contains(row, val) {
				t.Fatalf("JSON %s=%q for slug %q not present in default table row\nrow: %q", field, val, slug, row)
			}
		}
	}
}
