// ABOUTME: Table test for splitFrontDoorArgs — the front-door grammar that pulls
// ABOUTME: --skip-contract-check/--safehouse(-*) and the post-fence task off args.
package cli

import "testing"

// TestSplitFrontDoorArgs pins the front-door grammar (LP-1 fence convention +
// cycle-2 safehouse knobs). Host value-taking flags ride before the `--` fence
// and forward verbatim; the task is the bare text AFTER the fence; the
// front-door flags (--skip-contract-check, bare --safehouse, --safehouse-<key>=)
// are consumed wherever they appear and never forwarded.
func TestSplitFrontDoorArgs(t *testing.T) {
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
			name:    "fenced-task",
			args:    []string{"--", "do the thing"},
			task:    "do the thing",
			hasTask: true,
		},
		{
			name:        "host-flag-then-fenced-task",
			args:        []string{"--plugin-dir", "/p", "--", "do the thing"},
			passthrough: []string{"--plugin-dir", "/p"},
			task:        "do the thing",
			hasTask:     true,
		},
		{
			name:        "no-fence-all-passthrough",
			args:        []string{"--model", "gpt-x"},
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
			name:           "safehouse-knob-deprefixed",
			args:           []string{"--safehouse-enable=docker"},
			safehouseFlags: []string{"enable=docker"},
		},
		{
			name:           "knob-after-fence-still-consumed",
			args:           []string{"--", "--safehouse-enable=ssh", "task text"},
			safehouseFlags: []string{"enable=ssh"},
			task:           "task text",
			hasTask:        true,
		},
		{
			name:    "fenced-empty-task-string-counts",
			args:    []string{"--", ""},
			task:    "",
			hasTask: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fd, err := splitFrontDoorArgs(tc.args)
			if err != nil {
				t.Fatalf("splitFrontDoorArgs(%v) err = %v, want nil", tc.args, err)
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
