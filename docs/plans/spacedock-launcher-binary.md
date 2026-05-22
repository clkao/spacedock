---
id: 9bt646cz0h4q79g98qz68k9d
title: "Spacedock launcher binary: spacedock claude → safehouse claude --agent spacedock:first-officer (Go port sub-project #2)"
status: backlog
source: "Sub-project #2 of the Go port roadmap at docs/superpowers/specs/2026-05-12-spacedock-go-port-roadmap.md. Captain (CL) request 2026-05-22 to start a tiny Go skeleton in parallel with sub-project #1 (frontmatter spec, slug spacedock-frontmatter-contract-spec, id bxntyscd4sgxxdar9xty4nnt). Pattern: rtk-style brew formula install (rtk lives at /opt/homebrew/bin/rtk → Cellar/rtk/0.40.0/bin/rtk; mirror that install path). The `spacedock` binary's `claude` subcommand translates to `safehouse claude --agent spacedock:first-officer`, verifies the Claude Code plugin is installed, and optionally loads a safehouse config to apply flags like `--enable ssh`."
started:
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
safehouse claude [optional safehouse flags from config] --agent spacedock:first-officer [args...]
```

All `args...` after `claude` are forwarded verbatim to safehouse. Safehouse flags from the config are inserted before `--agent`.

## Acceptance criteria

End-state properties of the finished entity. Each AC is testable inside the binary's own behavior.

1. **`spacedock claude` invokes safehouse with the canonical argv.** Running `spacedock claude --foo bar` execs `safehouse claude --agent spacedock:first-officer --foo bar` (no config loading). Captain-provided args after `claude` are forwarded verbatim.
   - **Test:** Go integration test (`exec.Command` with a `safehouse` test stub on PATH that records its argv to a file); assert recorded argv equals the expected canonical form.

2. **Missing plugin produces a clear error and non-zero exit.** When the Spacedock Claude Code plugin is not installed in the host config, `spacedock claude` prints a one-line install hint naming the plugin source and exits with rc ≠ 0. Does not exec safehouse.
   - **Test:** integration test with a temp HOME containing no plugin config; assert stderr matches the hint pattern and `safehouse` was never invoked.

3. **`--safehouse-config <path>` loads config and forwards safehouse flags.** Given a fixture config with `enable: [ssh]`, `spacedock claude --safehouse-config fixture.yml` execs `safehouse claude --enable ssh --agent spacedock:first-officer`.
   - **Test:** Go integration test with the fixture and the safehouse stub; assert `--enable ssh` is in recorded argv before `--agent`.

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
- Builds on: rtk's brew-install pattern as a distribution model; safehouse's `--agent` flag; the existing Spacedock Claude Code plugin
- Composes with: sub-project #1 (frontmatter spec) is independent per AC-6; sub-projects #3 (status port) and #4 (claude-team port) will plug in as additional subcommands of this binary later
- Estimated complexity: small. ~500-700 LOC Go (per the roadmap doc), one brew formula, ~5 integration tests
- Cost estimate: ~$15-25 in agent budget. No live-claude E2E required.
