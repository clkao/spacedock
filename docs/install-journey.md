# Fresh-install journey

This walks a fresh install of Spacedock end to end on a clean Mac, naming the
observable command at each step and the output you should see. There are two
install lanes:

- **Released lane (brew)** — the published, tagged release. Available only after
  the first tagged release is cut.
- **Dev lane (`--plugin-dir`)** — a source build from `next`. The captain's
  primary development workflow today.

Each documented command is one you can run and observe the stated output.

## What gets installed

Spacedock is two pieces that install separately:

1. **The `spacedock` binary** — the launcher and contract gate.
2. **The host plugin** (`spacedock:first-officer` / `spacedock:ensign` skills and
   named agents) — loaded by Claude Code / Codex. The released lane installs it
   through the host's marketplace; the dev lane loads it from your checkout with
   `--plugin-dir`.

[safehouse](https://agent-safehouse.dev) is a separate runtime dependency used
to sandbox launches. It is NOT installed by either lane — install it yourself
when you want sandboxed runs.

## Released lane (brew)

> The brew lane is available only after the first tagged release. Until then the
> formula ships a placeholder url + sha256 and `brew install` will not fetch a
> real binary — use the dev lane below.

1. **Install the binary.**

   ```bash
   brew tap spacedock-dev/homebrew-tap
   brew install spacedock
   ```

   The no-tap one-liner is equivalent (the bare formula name is safe — no
   homebrew-core `spacedock` exists to disambiguate):

   ```bash
   brew install spacedock-dev/homebrew-tap/spacedock
   ```

2. **Confirm the binary.**

   ```bash
   spacedock --version
   ```

   Prints `spacedock <version> (contract 1)`. The `(contract 1)` token is the
   binary's compiled-in contract version; it is always correct regardless of how
   the binary was built.

3. **Install the host plugin.**

   ```bash
   spacedock init --host claude
   ```

   Adds the `spacedock-dev/spacedock` marketplace plugin for Claude Code, then
   runs the contract doctor. The published plugin carries
   `requires-contract: ">=1,<2"` so the doctor reports
   `OK: binary contract 1 satisfies plugin range >=1,<2.`

4. **Launch.**

   ```bash
   spacedock claude -- "your task"
   ```

   Version-gates against the installed plugin's `requires-contract` (GREEN, no
   `--skip-contract-check` needed) and launches `claude --agent
   spacedock:first-officer`. When a `.safehouse` profile is present in the
   working directory the launch is wrapped through safehouse.

## Dev lane (`--plugin-dir`)

This lane builds from source and loads the repo's own vendored plugin straight
from disk — no marketplace install, and `--plugin-dir` relaxes the contract gate
because the local checkout supersedes any installed plugin.

`next` has no release artifact — the release pipeline triggers on `v*` tags
only, so there is no `brew install …@next`. The `@next` lane is a source build.
Three source routes, each with different version-stamp behavior:

| Route | Command | `spacedock --version` reports |
|---|---|---|
| Local checkout (recommended) | `git clone … && go build -o spacedock ./cmd/spacedock` | the default `Version` (unstamped), `(contract 1)` correct |
| Toolchain fallback | `go install github.com/spacedock-dev/spacedock/cmd/spacedock@next` | the default `Version` (unstamped — `go install` does not pass release ldflags), NOT a git-describe pre-release identifier |
| Dev snapshot | `goreleaser release --snapshot --clean` | a snapshot-stamped tarball, not published |

Only a `v*` tag produces a git-describe-stamped binary. Do not expect `@next` to
yield a stamped version.

1. **Clone and build.**

   ```bash
   git clone https://github.com/spacedock-dev/spacedock
   cd spacedock
   go build -o spacedock ./cmd/spacedock
   ```

2. **Confirm the binary.**

   ```bash
   ./spacedock --version
   ```

   Prints `spacedock <version> (contract 1)` — the unstamped default `Version`
   on a local build, with the correct `(contract 1)` token.

3. **Confirm the repo loads as a plugin** (optional, observe-don't-grep). In an
   isolated config so no installed plugin masks the result:

   ```bash
   CLAUDE_CONFIG_DIR=$(mktemp -d) claude --plugin-dir "$PWD" plugin details spacedock
   ```

   Reports `Source: spacedock@inline` and an inventory naming `first-officer`
   and `ensign` under Skills and Agents, exit 0.

4. **Confirm the contract gate** (optional). The binary reads the vendored
   manifest's `requires-contract` directly:

   ```bash
   ./spacedock doctor --plugin-manifest .claude-plugin/plugin.json
   ```

   Prints `OK: binary contract 1 satisfies plugin range >=1,<2.` exit 0.

5. **Launch.**

   ```bash
   ./spacedock claude --plugin-dir "$PWD" -- "your task"
   ```

   Loads the repo's own vendored `spacedock:first-officer` / `spacedock:ensign`
   skills, relaxes the contract gate (the `--plugin-dir` checkout supersedes the
   installed plugin), and launches `claude --agent spacedock:first-officer` with
   your task appended.

## Command grammar

The front door is `spacedock claude [host-flags…] [--safehouse…] -- "task"`
(and the same shape for `spacedock codex`):

- Flags before `--` are passed through to the host (`claude` / `codex`),
  including `--plugin-dir`, `--resume`, `--model`, etc.
- The bare text after the `--` fence is the launch task, appended to the
  first-officer bootstrap prompt.
- `--safehouse` (or any `--safehouse-<key>=…` knob) forces the launch through
  safehouse; a `.safehouse` profile in the working directory does the same
  automatically.
- `--skip-contract-check` bypasses the contract gate (bootstrap only); a
  `--plugin-dir` launch relaxes the gate without it.
