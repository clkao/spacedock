---
id: 9bt646cz0h4q79g98qz68k9d
title: "Spacedock launcher binary: spacedock claude → safehouse claude --agent spacedock:first-officer (Go port sub-project #2)"
status: ideation
source: "Sub-project #2 of the Go port roadmap at docs/superpowers/specs/2026-05-12-spacedock-go-port-roadmap.md. Captain (CL) request 2026-05-22 to start a tiny Go skeleton in parallel with sub-project #1 (frontmatter spec, slug spacedock-frontmatter-contract-spec, id bxntyscd4sgxxdar9xty4nnt). Pattern: rtk-style brew formula install (rtk lives at /opt/homebrew/bin/rtk → Cellar/rtk/0.40.0/bin/rtk; mirror that install path). The `spacedock` binary's `claude` subcommand translates to `safehouse claude --agent spacedock:first-officer`, verifies the Claude Code plugin is installed, and optionally loads a safehouse config to apply flags like `--enable ssh`."
started: 2026-05-22T23:10:56Z
completed:
verdict:
score:
worktree:
---

# Spacedock launcher binary — sub-project #2 of Go port

A tiny Go module exposing a `spacedock` binary with a `claude` subcommand that launches the Spacedock first officer through safehouse. This is the foundational entry point of the Go port — sub-projects #3 (status port) and #4 (claude-team port) will land as additional subcommands of this binary later.

## Problem

Today the only way to run Spacedock as a Claude Code plugin is to manually invoke `claude` with the plugin installed and dispatch the first officer by hand. There is no single-command entry point, no plugin-presence check, and no bridge to safehouse for the increasingly-common case where the captain wants `--enable ssh` (or similar safehouse flags) without retyping them every session.

A binary launcher solves all three:

```
spacedock claude
  → ensure spacedock Claude Code plugin is installed (error early if not)
  → optionally load safehouse config and forward flags (--enable ssh, etc.)
  → exec safehouse claude --agent spacedock:first-officer [forwarded args]
```

The same binary becomes the natural home for `spacedock status` (sub-project #3) and `spacedock claude-team` (sub-project #4) later — sibling subcommands that re-implement the current shell scripts.

## Proposed approach

### Distribution: brew formula, mirroring rtk

`rtk` is installed at `/opt/homebrew/bin/rtk → Cellar/rtk/0.40.0/bin/rtk`, a standard Homebrew formula install. Mirror this. The Go module ships a `brew install spacedock` path (own tap or future homebrew-core), so on a fresh Mac the captain can:

```
brew tap clkao/spacedock         # or whatever tap name we pick
brew install spacedock
spacedock claude
```

### Subcommand structure

- `spacedock claude [args...]` — primary entry point this entity ships
- `spacedock codex [args...]` — stub that exits non-zero with "codex runtime not yet implemented"; signals the future sibling subcommand exists
- `spacedock --version` — prints the binary version
- (Future: `spacedock status`, `spacedock claude-team` — out of scope here)

### Plugin presence check

Before exec, verify the Spacedock Claude Code plugin is installed in the host config (typically `~/.claude/plugins/` or wherever the Claude Code CLI looks). If missing, print a one-line install hint pointing to the plugin source (the spacedock repo or a published plugin URL) and exit non-zero. Do not attempt to auto-install — the plugin install path is user-controlled.

### Safehouse config bridge

Accept an optional `--safehouse-config <path>` flag or `SPACEDOCK_SAFEHOUSE_CONFIG` env var. If provided, parse the config file (YAML or TOML — decide at ideation) and translate documented fields into safehouse CLI flags. Initial fields:

- `enable: [ssh, ...]` → `--enable ssh ...`
- Future fields documented as added

Unknown fields warn (don't error) so newer safehouse versions don't break the bridge.

### Translation contract

`spacedock claude [args...]` execs:

```
safehouse [safehouse-flags from config] -- claude --agent spacedock:first-officer [args...]
```

Notes from the ideation spike (see Design below):

- **`--agent` is a Claude Code flag, NOT a safehouse flag.** Safehouse forwards every argument after its own positional agent-binary (`claude`) verbatim to that binary. The spacedock first officer is selected by `claude --agent spacedock:first-officer`, which is only available when the Spacedock Claude Code plugin is installed (hence the plugin-presence check).
- **Safehouse flag form:** `--enable=ssh` (equals form, per agent-safehouse.dev/docs/options.html). Safehouse uses `--enable=KEY` for optional capabilities (`ssh`, `docker`, `kubectl`, `keychain`, `1password`, `cloud-credentials`, etc.).
- **Argument separator:** when any safehouse-flags are inserted, the launcher emits an explicit `--` before `claude` to disambiguate. When no safehouse flags are present, the launcher omits `--` (both forms are accepted by safehouse, but the `--` form is more robust).
- All `args...` after `spacedock claude` are forwarded verbatim to the inner `claude` binary, after `--agent spacedock:first-officer`. Safehouse flags from the config are inserted before `--`.

## Design

### Binary layout

```
spacedock/
├── cmd/spacedock/
│   └── main.go              # entry point; subcommand dispatch (claude, codex, --version)
├── internal/
│   ├── claude/
│   │   ├── run.go           # `spacedock claude` implementation: exec safehouse claude --agent ...
│   │   └── run_test.go      # integration tests with safehouse stub on PATH
│   ├── codex/
│   │   └── run.go           # `spacedock codex` placeholder (AC-5)
│   ├── plugin/
│   │   ├── detect.go        # plugin-presence check
│   │   └── detect_test.go
│   └── safehouseconfig/
│       ├── parse.go         # YAML config → []string safehouse flags
│       └── parse_test.go
├── formula/
│   └── spacedock.rb         # Homebrew formula (own-tap: clkao/spacedock)
├── go.mod
├── go.sum
└── README.md
```

Rationale for layout: standard Go project shape with `cmd/` for binaries and `internal/` for packages not intended for external import. The four `internal/` packages map 1:1 to the four ACs that touch behavior (1: claude, 3: safehouseconfig, 2: plugin, 5: codex). AC-4 (brew) and AC-6 (no frontmatter dep) are structural and tested by formula presence + `go list -m all` grep.

### Plugin-presence check

The check probes `~/.claude/plugins/installed_plugins.json` (the canonical Claude Code plugin registry observed on this machine at `/Users/clkao/.claude/plugins/installed_plugins.json`). It looks for a key matching the Spacedock plugin's installed name. On this machine the spacedock plugin marketplace is registered at `~/.claude/plugins/marketplaces/spacedock/`, so the binary probes BOTH:

1. `~/.claude/plugins/installed_plugins.json` contains a key starting with `spacedock@` (e.g. `spacedock@spacedock-marketplace` or similar — exact key shape verified at implementation time against the actual installed plugin entry)
2. OR `~/.claude/plugins/marketplaces/spacedock/` exists as a directory (fallback for dev installs from a local marketplace clone)

If neither is present, print to stderr:

```
spacedock: Claude Code plugin not detected.
Install via: /plugin marketplace add clkao/spacedock-marketplace && /plugin install spacedock@spacedock-marketplace
(or your tap's plugin URL)
```

and exit rc=2. Rationale: file-system probe is more durable than parsing claude CLI output, which has no stable `list-plugins` subcommand today. The check is a soft contract; if Claude Code changes its plugin registry path in the future, the launcher fails fast with a clear error rather than silently misbehaving — matches the spec's stance that the launcher does not auto-install.

### Safehouse-config bridge schema

YAML, with rationale: every other config file the captain interacts with daily (Spacedock entity frontmatter, claude `.claude/settings.json` neighbors, brew formula adjacents) is YAML or YAML-adjacent. TOML adds zero value for this small key set and a new mental model.

`~/.config/spacedock/safehouse.yml` (or whatever path is passed via `--safehouse-config`):

```yaml
# Optional safehouse features enabled by spacedock claude.
# Each item becomes a safehouse --enable=<key> flag.
# Valid keys (from agent-safehouse.dev/docs/options.html):
#   ssh, docker, kubectl, keychain, macos-gui, microphone, 1password,
#   cloud-credentials, cloud-storage, shell-init, process-control,
#   lldb, vscode, xcode, wide-read, etc.
enable:
  - ssh
  - 1password

# Optional read-write directory grants. Each becomes --add-dirs=<path>.
add_dirs:
  - ~/scratch

# Optional read-only directory grants. Each becomes --add-dirs-ro=<path>.
add_dirs_ro:
  - ~/reference-data

# Optional extra sandbox profile files. Each becomes --append-profile=<path>.
append_profile:
  - ~/.config/spacedock/local-overrides.sb
```

Translation order: `--enable=` first, then `--add-dirs=`, then `--add-dirs-ro=`, then `--append-profile=`, then `--`, then `claude --agent spacedock:first-officer`, then forwarded args. Unknown top-level keys produce a stderr warning naming the key and continue (forward-compat with newer safehouse features).

### Brew formula skeleton

Own-tap at `github.com/clkao/homebrew-spacedock` (formula path: `Formula/spacedock.rb`). Recommend own-tap for v1 with rationale: homebrew-core has strict acceptance criteria (notable user count, stability, no rapid version churn) that v0.1 of a personal tool doesn't meet. Mirror rtk's distribution model — rtk uses the `rtk-ai/homebrew-tap` pattern with pre-built release binaries.

```ruby
class Spacedock < Formula
  desc "Spacedock launcher — runs Claude Code with the Spacedock first officer via safehouse"
  homepage "https://github.com/clkao/spacedock"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/clkao/spacedock/releases/download/v#{version}/spacedock-aarch64-apple-darwin.tar.gz"
      sha256 "TBD-at-first-release"
    end
    on_intel do
      url "https://github.com/clkao/spacedock/releases/download/v#{version}/spacedock-x86_64-apple-darwin.tar.gz"
      sha256 "TBD-at-first-release"
    end
  end

  # Note: agent-safehouse is the runtime dependency. It is NOT auto-installed
  # by brew (different tap, different lifecycle) — the launcher errors at
  # exec-time if safehouse is missing. Document the install hint in README.
  # depends_on "eugene1g/safehouse/agent-safehouse"  # optional; consider once stable

  def install
    bin.install "spacedock"
  end

  test do
    assert_match "spacedock #{version}", shell_output("#{bin}/spacedock --version")
  end
end
```

User install path: `brew tap clkao/spacedock && brew install spacedock`. Promote to homebrew-core only if the binary stabilizes and external users ask.

### Versioning

`spacedock --version` returns `spacedock 0.1.0` (literal version string baked at build time via `-ldflags "-X main.Version=$(git describe --tags --always)"`). Matches rtk's `rtk 0.40.0` pattern. The class-level `version "0.1.0"` in the formula and the build-time `-X main.Version` are kept in sync at release time by the release script (out of scope here; documented in README at implementation time).

### Spike evidence

Ideation spike at `docs/plans/_evidence/spacedock-launcher-binary-ideation/`:

- `main.go` — ~50-line Go program that constructs the canonical argv and execs `safehouse` (stubbed)
- `safehouse-stub.sh` — records its argv to a file for assertion

Run output (verified in this stage):

- Case 1 (no `--enable-ssh`): argv = `claude --agent spacedock:first-officer --foo bar` — PASS
- Case 2 (`--enable-ssh`): argv = `--enable=ssh -- claude --agent spacedock:first-officer --foo bar` — PASS

Verdict: PASS for the translation contract. Both argv shapes match the documented safehouse CLI. Note: the spike does not exercise live safehouse — the local safehouse binary at `~/.local/bin/safehouse` is not accessible in this sandbox. The contract is established from official docs (agent-safehouse.dev/docs/options.html and /docs/usage.html) which confirm: (a) safehouse forwards all args after the agent binary verbatim, (b) `--enable=KEY` is the documented flag form, (c) the `--` separator is the documented usage pattern when safehouse flags are present, (d) `--agent` is a Claude Code flag (not safehouse) so it must appear AFTER `claude` in argv.

## Acceptance criteria

End-state properties of the finished entity. Each AC is testable inside the binary's own behavior.

1. **`spacedock claude` invokes safehouse with the canonical argv.** Running `spacedock claude --foo bar` execs `safehouse claude --agent spacedock:first-officer --foo bar` (no config loading, no `--` separator). Captain-provided args after `claude` are forwarded verbatim to the inner claude binary, positioned after `--agent spacedock:first-officer`.
   - **Test:** Go integration test (`exec.Command` with a `safehouse` test stub on PATH that records its argv to a file); assert recorded argv equals the expected canonical form. The ideation spike at `docs/plans/_evidence/spacedock-launcher-binary-ideation/` demonstrates the pattern.

2. **Missing plugin produces a clear error and non-zero exit.** When the Spacedock Claude Code plugin is not installed in the host config, `spacedock claude` prints a one-line install hint naming the plugin source and exits with rc ≠ 0. Does not exec safehouse.
   - **Test:** integration test with a temp HOME containing no plugin config; assert stderr matches the hint pattern and `safehouse` was never invoked.

3. **`--safehouse-config <path>` loads config and forwards safehouse flags.** Given a fixture config with `enable: [ssh]`, `spacedock claude --safehouse-config fixture.yml` execs `safehouse --enable=ssh -- claude --agent spacedock:first-officer`. Safehouse flags appear before the `--` separator; `--agent` and forwarded args appear after.
   - **Test:** Go integration test with the fixture and the safehouse stub; assert `--enable=ssh` and `--` appear in recorded argv before `claude --agent spacedock:first-officer`.

4. **Brew install puts `spacedock` on PATH.** A `brew install spacedock` (via own tap or local-formula install) results in `which spacedock` returning the brew-managed path and `spacedock --version` returning a non-empty version string.
   - **Test:** install-path smoke test documented in the README; CI may skip the actual brew install but the formula file is committed and lint-passes (`brew audit`, etc.).

5. **`spacedock codex` is a placeholder, not a no-op silent.** Running `spacedock codex` exits with rc ≠ 0 and prints `codex runtime not yet implemented (sub-project of Go port)` (or similar) to stderr. Signals the sibling subcommand exists; future port plugs in without breaking the CLI shape.
   - **Test:** Go integration test asserts the stub exit code and stderr substring.

6. **No dependency on sub-project #1 (frontmatter spec).** The launcher does not parse any Spacedock frontmatter or mdschema files. It only handles process exec + config translation. This keeps the two sub-projects shippable independently.
   - **Test:** static check — `go list -m all` shows no dependency on a Spacedock frontmatter parser; ripgrep the source for `frontmatter` / `mdschema` returns no hits.

## Test plan

- **Go integration tests** under `cmd/spacedock/` (or wherever the binary lives) for ACs 1, 2, 3, 5, 6. The pattern is: stub `safehouse` on PATH, run `spacedock claude`, inspect recorded argv.
- **Brew formula** committed at `formula/spacedock.rb` (or in an own-tap repo); `brew audit` runs as part of CI if practical, otherwise documented in the entity.
- **No live-claude E2E required at this stage** — the binary's contract is process exec + flag translation. Live runs of `safehouse claude --agent spacedock:first-officer` are exercised by sub-project #3/#4 and by ordinary user sessions, not by this entity's tests.

## Out of scope

- **`spacedock status`, `spacedock claude-team`.** Sub-projects #3 and #4. This entity ships the binary skeleton + the `claude` subcommand only.
- **Plugin auto-install.** Plugin install is user-controlled. The launcher detects and errors; it does not install.
- **Cross-platform packaging.** Mac-first via brew. Linux/Windows packaging is a follow-up if/when users ask.
- **Sub-project #1 dependency.** The launcher does not parse frontmatter; AC-6 enforces this independence.
- **Larger refactor that consolidates skill + claude-team semantics.** That refactor is acknowledged separately (captain note 2026-05-22 in the rdt entity's implementation cycle).

## Risks

### Risk A — safehouse CLI shape changes

The translation contract assumes safehouse accepts `--agent <id>` and the bridged flags (`--enable ssh`, etc.) in the form documented today. If safehouse changes its flag shape, the launcher needs an update. Mitigation: keep the translation logic in one small function; document the safehouse version this launcher is tested against; warn on unknown safehouse flags.

### Risk B — brew formula maintenance overhead

A custom tap is a small but real maintenance surface (version bumps, formula updates on new releases). Mitigation: start with an own-tap (`clkao/spacedock`); promote to homebrew-core only if the binary stabilizes and there's external user demand.

### Risk C — plugin-detection brittleness

The plugin-presence check needs to know where Claude Code stores plugin metadata. If that location changes across Claude Code versions, the check can false-positive or false-negative. Mitigation: probe via the Claude Code CLI itself if it exposes a "list-plugins" command, or document the path explicitly and version-gate it.

## Scale context

- Spacedock version: 0.12.0+
- Builds on: rtk's brew-install pattern as a distribution model; Claude Code's `--agent` flag (NOT safehouse's — see Design); the existing Spacedock Claude Code plugin
- Composes with: sub-project #1 (frontmatter spec) is independent per AC-6; sub-projects #3 (status port) and #4 (claude-team port) will plug in as additional subcommands of this binary later
- Estimated complexity: small. ~500-700 LOC Go (per the roadmap doc), one brew formula, ~5 integration tests
- Cost estimate: ~$15-25 in agent budget. No live-claude E2E required.

## Stage Report: ideation

- DONE: Inspect rtk's distribution to ground the brew-install pattern concretely.
  rtk uses own-tap `rtk-ai/homebrew-tap` with `Formula/rtk.rb` shipping pre-built release tarballs per platform; verified rtk binary at `/opt/homebrew/bin/rtk -> Cellar/rtk/0.40.0/bin/rtk` and inspected formula via GitHub. Recommend own-tap (`clkao/spacedock`) for v1; promote to homebrew-core only after external user demand and version stability. Design section documents the full formula skeleton.
- DONE: Run a tiny end-to-end spike of the translation contract.
  Spike at `docs/plans/_evidence/spacedock-launcher-binary-ideation/` (~30-line `main.go` + bash stub). Two cases verified: no-config → `claude --agent spacedock:first-officer --foo bar`; with `--enable-ssh` → `--enable=ssh -- claude --agent spacedock:first-officer --foo bar`. Verdict: PASS. Local `~/.local/bin/safehouse` was sandbox-blocked so the live binary was not exercised; the contract is grounded in official safehouse docs (options.html, usage.html) which independently confirm the argv shape.
- DONE: Populate ## Design with binary layout, plugin-check path, config schema, formula skeleton.
  Design section added before Acceptance criteria with five subsections: Binary layout (`cmd/spacedock/`, `internal/{claude,codex,plugin,safehouseconfig}/`, `formula/`), Plugin-presence check (probes `~/.claude/plugins/installed_plugins.json` and `~/.claude/plugins/marketplaces/spacedock/`), Safehouse-config bridge schema (YAML with `enable`/`add_dirs`/`add_dirs_ro`/`append_profile`), Brew formula skeleton (own-tap pattern mirroring rtk), Versioning (`-ldflags -X main.Version`).
- DONE: Tighten ACs to reflect spike findings.
  AC-1 clarified to position `--agent spacedock:first-officer` BEFORE forwarded args (matches safehouse forward-verbatim semantics). AC-3 corrected from `--enable ssh` (space) to `--enable=ssh` (equals) and now requires the `--` separator. Translation contract section rewritten: `--agent` is a Claude Code flag (not safehouse), safehouse uses `safehouse [flags] -- claude [...]` pattern.

### Summary

Spike validated the translation contract end-to-end with a real Go binary and stubbed safehouse, confirming the argv shape is `safehouse [--enable=K]* -- claude --agent spacedock:first-officer [forwarded...]`. The biggest correction relative to the seed spec: `--agent` is Claude Code's flag, not safehouse's, and safehouse uses `--enable=KEY` (equals form) with an optional `--` separator — both verified against the official agent-safehouse.dev docs. Design section now contains binary layout, plugin-check probe paths, full YAML config schema with four sections (enable/add_dirs/add_dirs_ro/append_profile), and a brew formula skeleton mirroring rtk's own-tap pattern. AC-6 (independence from sub-project #1) is preserved; no Spacedock frontmatter is parsed by the launcher.
