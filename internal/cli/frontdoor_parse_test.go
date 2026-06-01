// ABOUTME: Table test for parseFrontDoorArgs — the Option-2 grammar that takes the
// ABOUTME: task before --, host flags after --, and the safehouse/skip knobs anywhere.
package cli

import "testing"

// TestParseFrontDoorArgs pins the Option-2 front-door grammar (AC-3 + AC-6). The
// task is the joined non-flag positionals BEFORE `--`; host value-taking flags
// ride AFTER `--` and forward verbatim as passthrough; the spacedock-owned flags
// (--skip-contract-check, --safehouse, the three repeatable --safehouse-* knobs)
// are consumed wherever they appear before `--` in BOTH space and equals form and
// are never forwarded.
func TestParseFrontDoorArgs(t *testing.T) {
	cases := []struct {
		name           string
		args           []string
		passthrough    []string
		task           string
		hasTask        bool
		forceSafehouse bool
		safehouseFlags []string
		skipCheck      bool
	}{
		{name: "bare"},
		{
			name:    "task-positional",
			args:    []string{"do the thing"},
			task:    "do the thing",
			hasTask: true,
		},
		{
			name:    "multi-word-task-joins",
			args:    []string{"do", "the", "thing"},
			task:    "do the thing",
			hasTask: true,
		},
		{
			name:        "task-then-fenced-host-flag",
			args:        []string{"do the thing", "--", "--plugin-dir", "/p"},
			passthrough: []string{"--plugin-dir", "/p"},
			task:        "do the thing",
			hasTask:     true,
		},
		{
			name:        "host-flags-after-fence-no-task",
			args:        []string{"--", "--model", "gpt-x"},
			passthrough: []string{"--model", "gpt-x"},
		},
		{
			name:      "skip-contract-check-consumed",
			args:      []string{"--skip-contract-check"},
			skipCheck: true,
		},
		{
			name:           "bare-safehouse-consumed",
			args:           []string{"--safehouse"},
			forceSafehouse: true,
		},
		{
			name:           "safehouse-knob-equals-form",
			args:           []string{"--safehouse-enable=docker"},
			safehouseFlags: []string{"enable=docker"},
		},
		{
			name:           "safehouse-knob-space-form",
			args:           []string{"--safehouse-enable", "docker"},
			safehouseFlags: []string{"enable=docker"},
		},
		{
			name:           "knob-and-task-and-fenced-host-flags",
			args:           []string{"--safehouse-enable=ssh", "task text", "--", "--model", "x"},
			safehouseFlags: []string{"enable=ssh"},
			passthrough:    []string{"--model", "x"},
			task:           "task text",
			hasTask:        true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fd, err := parseFrontDoorArgs(tc.args)
			if err != nil {
				t.Fatalf("parseFrontDoorArgs(%v) err = %v, want nil", tc.args, err)
			}
			if !equalArgv(fd.passthrough, tc.passthrough) {
				t.Errorf("passthrough = %v, want %v", fd.passthrough, tc.passthrough)
			}
			if fd.task != tc.task || fd.hasTask != tc.hasTask {
				t.Errorf("task = (%q,%v), want (%q,%v)", fd.task, fd.hasTask, tc.task, tc.hasTask)
			}
			if fd.forceSafehouse != tc.forceSafehouse {
				t.Errorf("forceSafehouse = %v, want %v", fd.forceSafehouse, tc.forceSafehouse)
			}
			if !equalArgv(fd.safehouseFlags, tc.safehouseFlags) {
				t.Errorf("safehouseFlags = %v, want %v", fd.safehouseFlags, tc.safehouseFlags)
			}
			if fd.skipCheck != tc.skipCheck {
				t.Errorf("skipCheck = %v, want %v", fd.skipCheck, tc.skipCheck)
			}
		})
	}
}
