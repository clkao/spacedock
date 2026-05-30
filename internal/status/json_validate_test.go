// ABOUTME: AC-1 --validate --json envelope — valid emits valid:"true", invalid
// ABOUTME: emits valid:"false" before the exit-1 return, with stderr preserved.
package status

import (
	"encoding/json"
	"testing"
)

// TestValidateJSONBranches locks both --validate --json branches: a valid
// workflow emits {"command":"validate","valid":"true"} + exit 0, and an invalid
// workflow emits {"command":"validate","valid":"false"} + exit 1 while still
// writing the diagnostic lines to stderr (the --quiet/JSON contract never
// suppresses error diagnostics or alters exit codes). The false path is the
// regression target: before the fix, the invalid path returned 1 after stderr
// errors before ever reaching the if-asJSON block, so it emitted no JSON.
func TestValidateJSONBranches(t *testing.T) {
	cases := []struct {
		name      string
		files     map[string]string
		wantValid string
		wantCode  int
		wantErr   bool
	}{
		{
			name:      "valid",
			files:     map[string]string{"a.md": ent(`"001"`, "backlog"), "b.md": ent(`"002"`, "done")},
			wantValid: "true",
			wantCode:  0,
			wantErr:   false,
		},
		{
			name:      "invalid",
			files:     map[string]string{"a.md": "---\ntitle: T\nstatus: backlog\n---\n# T\n"},
			wantValid: "false",
			wantCode:  1,
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			env := pinnedEnv(t)
			root := validationFixture(t, seqREADME, tc.files)

			out, errOut, code := runNative(t, root, env, "--workflow-dir", root, "--validate", "--json")
			if code != tc.wantCode {
				t.Fatalf("exit=%d want %d (stderr=%q)", code, tc.wantCode, errOut)
			}

			var doc map[string]string
			if err := json.Unmarshal([]byte(out), &doc); err != nil {
				t.Fatalf("parse --validate --json: %v\nstdout=%q", err, out)
			}
			if doc["command"] != "validate" {
				t.Fatalf("command=%q want validate", doc["command"])
			}
			if doc["valid"] != tc.wantValid {
				t.Fatalf("valid=%q want %q", doc["valid"], tc.wantValid)
			}
			// All values are strings — the envelope has exactly command + valid.
			if len(doc) != 2 {
				t.Fatalf("envelope has %d keys, want 2 (command,valid): %v", len(doc), doc)
			}

			// stderr diagnostics are preserved on the invalid path; the JSON
			// envelope is additive on stdout and never silences the error lines.
			if tc.wantErr && errOut == "" {
				t.Fatalf("invalid workflow must still write diagnostics to stderr; stderr was empty")
			}
			if !tc.wantErr && errOut != "" {
				t.Fatalf("valid workflow must not write to stderr; stderr=%q", errOut)
			}
		})
	}
}
