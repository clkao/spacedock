// ABOUTME: Host front doors (spacedock claude/codex) + init/doctor — the three
// ABOUTME: version-gate points wired through an injectable host-ops seam.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/spacedock-dev/spacedock/internal/contract"
	"github.com/spacedock-dev/spacedock/internal/safehouse"
)

// bootstrapPrompt is the fixed launch-and-go message appended as the last inner
// argv token so a fresh `spacedock claude` session starts the first officer
// rather than opening an idle agent. It is omitted when `--resume` is forwarded
// (a resume already carries its own session intent).
const bootstrapPrompt = "You totally got this. Take your time. I love you. And tell all subagents and team members you love them too. Engage."

// hostOps is the injectable seam the front-door, init, and doctor paths depend
// on. Production backs it with real `claude`/`codex` plugin commands and exec;
// tests back it with a fake that records interactions. Mirrors the status.Runner
// decoupling: the command logic never touches exec or the host CLI directly.
type hostOps interface {
	// ResolveManifest returns the installed plugin manifest path for host, or ""
	// when no plugin is installed (a distinct, non-error state). A non-nil error
	// means the host CLI itself failed.
	ResolveManifest(host string) (string, error)
	// Launch execs argv, replacing the current process on success (production) or
	// recording it (test). It returns only on failure to launch.
	Launch(argv []string) error
	// Install issues the host plugin commands to install/update the plugin from
	// source (optionally pinned to branch), returning combined output.
	Install(host, source, branch string) (string, error)
}

// devBranch is the pre-release branch woven into the install/remedy commands as
// the marketplace `@ref` (and Codex `--ref`). The default is `next`: until `next`
// is the repository's default branch, the released binary installs the plugin
// from `spacedock-dev/spacedock@next`, where the root marketplace.json lives. It
// is a var (not a const) so the linker can stamp it, mirroring Version, and so
// `SPACEDOCK_DEV_BRANCH` can override it; tests save/restore it.
var devBranch = "next"

// gateHost resolves the installed manifest for host and compares it against
// CONTRACT_VERSION. It returns whether launch is permitted. Only a Compatible
// verdict permits launch; everything else (a host-CLI error, no installed
// plugin, a resolved-but-missing manifest, a mismatch, or a malformed range) is
// NOT compatible — the front door's fail-fast job — so launch is denied with an
// actionable message. The gate inspects the VERDICT, not a doctor exit code:
// RunDoctor maps no-plugin-found to exit 0 (a non-fatal report), so a non-empty
// installPath to a missing manifest would otherwise slip through as "compatible".
func gateHost(ops hostOps, host string, stderr io.Writer) (ok bool) {
	manifestPath, err := ops.ResolveManifest(host)
	if err != nil {
		fmt.Fprintf(stderr,
			"Spacedock: could not resolve the installed %s plugin (%v). "+
				"Run `spacedock doctor` or `spacedock init --host %s`.\n", host, err, host)
		return false
	}
	if manifestPath == "" {
		fmt.Fprintf(stderr,
			"Spacedock: no installed %s plugin found. "+
				"Run `spacedock init --host %s` (or `spacedock claude --skip-contract-check` to bootstrap).\n", host, host)
		return false
	}
	res := contract.ManifestVerdict(manifestPath, host, devBranch)
	if res.Verdict == contract.Compatible {
		return true
	}
	if res.Verdict == contract.NoPluginFound {
		fmt.Fprintf(stderr,
			"Spacedock: the installed %s plugin reported a manifest path that does not exist (%s). "+
				"Run `spacedock init --host %s` (or `spacedock claude --skip-contract-check` to bootstrap).\n",
			host, manifestPath, host)
		return false
	}
	fmt.Fprintln(stderr, res.Message)
	return false
}

// runClaude is the `spacedock claude` front door: version-gate (fail fast), then
// launch the first officer. The launch is interposed through
// `safehouse --trust-workdir-config [extra] -- claude --dangerously-skip-permissions …`
// when ANY of {a `.safehouse` profile in dir, the bare `--safehouse` flag, a
// `--safehouse-*` knob} is given; otherwise it is plain `claude --agent
// spacedock:first-officer …` (no skip-permissions in an unsandboxed launch). The
// `--safehouse-*` knobs translate into the safehouse `extra` slot. The bootstrap
// prompt is appended last (base, or base + " " + task when a task is fenced after
// `--`) unless a resume is forwarded. The gate is bypassed by an explicit
// `--skip-contract-check` or by any `--plugin-dir` (the local checkout supersedes
// the installed plugin). `lookPath` resolves the safehouse binary (default
// exec.LookPath; injected so tests pin not-found).
func runClaude(ctx context.Context, args []string, dir string, ops hostOps, lookPath func(string) (string, error), stdout, stderr io.Writer) int {
	fd, err := parseFrontDoorArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "spacedock claude: %v\n", err)
		return 1
	}
	extra, err := safehouse.TranslateFlags(fd.safehouseFlags)
	if err != nil {
		fmt.Fprintf(stderr, "spacedock claude: %v\n", err)
		return 1
	}
	// A `--plugin-dir` launch loads the LOCAL plugin checkout, so the installed
	// plugin's contract verdict is irrelevant — it relaxes the gate exactly like
	// an explicit `--skip-contract-check`.
	if !fd.skipCheck && !hasPluginDir(fd.passthrough) {
		if !gateHost(ops, "claude", stderr) {
			return 1
		}
	}

	wrap := safehouse.Present(dir) || fd.forceSafehouse || len(fd.safehouseFlags) > 0
	resume := containsResume(fd.passthrough)
	inner := []string{"claude"}
	if wrap {
		inner = append(inner, "--dangerously-skip-permissions")
	}
	inner = append(inner, "--agent", "spacedock:first-officer")
	inner = append(inner, fd.passthrough...)
	if !resume {
		inner = append(inner, launchPrompt(bootstrapPrompt, fd))
	}

	argv := inner
	if wrap {
		if ok, hint := safehouse.Available(lookPath); !ok {
			fmt.Fprintln(stderr, hint)
			return 1
		}
		argv = safehouse.Wrap(inner, extra)
	}

	if err := ops.Launch(argv); err != nil {
		fmt.Fprintf(stderr, "spacedock claude: launch failed: %v\n", err)
		return 1
	}
	return 0
}

// launchPrompt returns the inner-argv launch prompt: `base + " " + task` when the
// operator fenced a task after `--`, otherwise the bare base prompt. Callers
// suppress it entirely on a resume (which carries its own session intent).
func launchPrompt(base string, fd frontDoorArgs) string {
	if fd.hasTask {
		return base + " " + fd.task
	}
	return base
}

// hasPluginDir reports whether the host passthrough carries a `--plugin-dir`
// flag (either `--plugin-dir P` or `--plugin-dir=P`). Its presence relaxes the
// contract gate (the local checkout supersedes the installed plugin).
func hasPluginDir(passthrough []string) bool {
	for _, a := range passthrough {
		if a == "--plugin-dir" || strings.HasPrefix(a, "--plugin-dir=") {
			return true
		}
	}
	return false
}

// containsResume reports whether the operator forwarded any of claude's
// session-resume forms (which carry their own session intent, so the bootstrap
// prompt is suppressed): `--resume`, `--resume=<id>`, `-r`, `--continue`, `-c`.
func containsResume(args []string) bool {
	for _, a := range args {
		switch a {
		case "--resume", "-r", "--continue", "-c":
			return true
		}
		if strings.HasPrefix(a, "--resume=") {
			return true
		}
	}
	return false
}

// codexBootstrapPrompt is the fixed launch-and-go message appended as the last
// inner argv token so a fresh `spacedock codex` session starts the first
// officer. Codex has no `--agent` analog (spike-confirmed: no agent/skill-select
// flag on the top-level, `exec`, or `plugin` surfaces), so the only FO-selection
// injection point is the positional prompt — this prompt names the
// `spacedock:first-officer` skill explicitly.
const codexBootstrapPrompt = "You totally got this. Take your time. I love you. And tell all subagents and team members you love them too. Engage. Assume $spacedock:first-officer for the entire session."

// runCodex is the `spacedock codex` front door: version-gate (fail fast), then
// launch the first officer. The launch is interposed through
// `safehouse --trust-workdir-config [extra] -- codex --dangerously-bypass-approvals-and-sandbox …`
// when ANY of {a `.safehouse` profile in dir, the bare `--safehouse` flag, a
// `--safehouse-*` knob} is given — safehouse is the sandbox, so codex's own
// sandbox is bypassed. Otherwise the launch is plain `codex …` keeping codex's own
// sandbox (the bypass flag is omitted: it is safe only when safehouse provides the
// sandbox). The FO-skill bootstrap prompt is appended last (base, or base + " " +
// task when a task is fenced after `--`) unless the passthrough begins with the
// `resume` subcommand. The gate is bypassed by `--skip-contract-check` or by any
// `--plugin-dir`. `lookPath` resolves the safehouse binary (default exec.LookPath;
// injected so tests pin not-found).
func runCodex(ctx context.Context, args []string, dir string, ops hostOps, lookPath func(string) (string, error), stdout, stderr io.Writer) int {
	fd, err := parseFrontDoorArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "spacedock codex: %v\n", err)
		return 1
	}
	extra, err := safehouse.TranslateFlags(fd.safehouseFlags)
	if err != nil {
		fmt.Fprintf(stderr, "spacedock codex: %v\n", err)
		return 1
	}
	if !fd.skipCheck && !hasPluginDir(fd.passthrough) {
		if !gateHost(ops, "codex", stderr) {
			return 1
		}
	}

	wrap := safehouse.Present(dir) || fd.forceSafehouse || len(fd.safehouseFlags) > 0
	resume := codexResume(fd.passthrough)
	inner := []string{"codex"}
	if wrap {
		inner = append(inner, "--dangerously-bypass-approvals-and-sandbox")
	}
	inner = append(inner, fd.passthrough...)
	if !resume {
		inner = append(inner, launchPrompt(codexBootstrapPrompt, fd))
	}

	argv := inner
	if wrap {
		if ok, hint := safehouse.Available(lookPath); !ok {
			fmt.Fprintln(stderr, hint)
			return 1
		}
		argv = safehouse.Wrap(inner, extra)
	}

	if err := ops.Launch(argv); err != nil {
		fmt.Fprintf(stderr, "spacedock codex: launch failed: %v\n", err)
		return 1
	}
	return 0
}

// codexResume reports whether the codex passthrough begins with the `resume`
// subcommand (codex's resume is a leading subcommand, not a flag like claude's
// `--resume`). A resume carries its own session intent, so the bootstrap prompt
// is suppressed.
func codexResume(passthrough []string) bool {
	return len(passthrough) > 0 && passthrough[0] == "resume"
}

// frontDoorArgs is the parsed front-door grammar. The launchers read it to
// assemble the inner host argv and decide the safehouse wrap.
type frontDoorArgs struct {
	// passthrough is the host-only argv (claude/codex flags), in operator order.
	passthrough []string
	// task is the launch-prompt override (the bare text after the `--` fence);
	// hasTask distinguishes an explicit empty task from "no fence given".
	task    string
	hasTask bool
	// forceSafehouse is set by the bare `--safehouse` front-door flag.
	forceSafehouse bool
	// safehouseFlags are the de-prefixed `--safehouse-<key>=…` knob tokens, fed to
	// safehouse.TranslateFlags. Their presence also implies sandbox-on.
	safehouseFlags []string
	// skipCheck is set by `--skip-contract-check` (bypasses the contract gate).
	skipCheck bool
}

// frontDoorFlags binds the spacedock-owned front-door flags onto a pflag.FlagSet
// so cobra owns their vocabulary natively: the three value-taking safehouse knobs
// are StringArray (accept both `--flag value` and `--flag=value`, accumulate on
// repeat), and the bare `--safehouse`/`--skip-contract-check` are Bool. The
// returned pointers are read back by parseFrontDoorArgs after Parse. The same
// binding feeds the per-command cobra help (AC-4), so the help and the parser
// never drift.
type frontDoorFlags struct {
	safehouse *bool
	skipCheck *bool
	enable    *[]string
	addDirs   *[]string
	addDirsRO *[]string
}

func bindFrontDoorFlags(fs *pflag.FlagSet) frontDoorFlags {
	return frontDoorFlags{
		safehouse: fs.Bool("safehouse", false,
			"Force the safehouse sandbox wrap even without a .safehouse profile in the directory"),
		skipCheck: fs.Bool("skip-contract-check", false,
			"Bypass the contract gate and launch without resolving the installed plugin (bootstrap)"),
		enable: fs.StringArray("safehouse-enable", nil,
			"Enable a safehouse capability (KEY[,KEY]); repeatable; e.g. --safehouse-enable ssh,docker"),
		addDirs: fs.StringArray("safehouse-add-dirs", nil,
			"Grant safehouse read-write access to a directory; repeatable"),
		addDirsRO: fs.StringArray("safehouse-add-dirs-ro", nil,
			"Grant safehouse read-only access to a directory; repeatable"),
	}
}

// parseFrontDoorArgs parses the Option-2 front-door grammar in one pass via a
// pflag.FlagSet. cobra owns the spacedock flags wherever they appear before `--`;
// the non-flag positionals before `--` join (single space) into the launch task;
// everything after `--` forwards verbatim to the host as passthrough. This is the
// grammar inversion: host flags now ride AFTER `--`, the task before — the prompt
// is always spacedock-constructed and never adjacent to a value-taking host flag,
// so no dangling host flag can swallow it. The collected safehouse-knob values are
// re-prefixed to the `key=value` token form safehouse.TranslateFlags owns, so the
// safehouse vocabulary (the comma-split on enable, the unknown-key error) is
// unchanged — the space-form bug dies because cobra consumes the value as the
// flag's argument instead of leaking it to passthrough.
func parseFrontDoorArgs(args []string) (fd frontDoorArgs, err error) {
	fs := pflag.NewFlagSet("spacedock-front-door", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flags := bindFrontDoorFlags(fs)
	if err := fs.Parse(args); err != nil {
		return frontDoorArgs{}, err
	}

	fd.forceSafehouse = *flags.safehouse
	fd.skipCheck = *flags.skipCheck
	for _, v := range *flags.enable {
		fd.safehouseFlags = append(fd.safehouseFlags, "enable="+v)
	}
	for _, v := range *flags.addDirs {
		fd.safehouseFlags = append(fd.safehouseFlags, "add-dirs="+v)
	}
	for _, v := range *flags.addDirsRO {
		fd.safehouseFlags = append(fd.safehouseFlags, "add-dirs-ro="+v)
	}

	// ArgsLenAtDash is the count of positionals seen before `--` (or -1 when no
	// `--` was given). Without a `--`, every positional is the task and nothing
	// forwards. With a `--`, the pre-dash positionals join into the task and the
	// post-dash positionals forward verbatim as host passthrough.
	positionals := fs.Args()
	dash := fs.ArgsLenAtDash()
	var taskTokens []string
	if dash < 0 {
		taskTokens = positionals
	} else {
		taskTokens = positionals[:dash]
		fd.passthrough = positionals[dash:]
	}
	if len(taskTokens) > 0 {
		fd.task = strings.Join(taskTokens, " ")
		fd.hasTask = true
	}
	return fd, nil
}
