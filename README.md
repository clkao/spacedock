# Spacedock

Hand an agent a multi-step job and it drifts, skips steps, or invents its own path. Hand it a workflow and it follows the workflow.

Spacedock runs agent work through stages you define. Each stage gets a fresh context, an explicit gate, and a structured report. Reviewers are allowed to push back. State lives outside the agent, so work survives the context limit and picks up the next session where it left off.

We built Spacedock because we were there. We lived the pipelines that didn't finish, the reviewers that rubber-stamped, the work that died at the context limit, and the routines spread across too many tools. The tool is the way we wished agent work had been organized from the start.

**You want Spacedock if:**

- **You delegate repeatable work to agents** (the same pipeline run on many inputs: code reviews, content drafts, outreach batches) and you want each run to finish, not stop halfway. Spacedock dispatches a fresh agent per stage, with an approval gate where your judgment actually matters.
- **You don't trust single-agent output** on its own. Spacedock review stages are allowed to push back, with a 3-strikes escalation so you only see what the reviewer couldn't resolve.
- **Your work spans days or weeks.** A blog draft, a launch strategy, a multi-week benchmark. Spacedock holds artifact state outside the agent, so the next session resumes instead of restarts.

## What's Different

- **Approval gates with structured evidence.** Every gate comes with a stage report: findings, verdicts, artifacts, anomalies. You approve, redirect, or bounce back faster than sifting through raw output or a sprawling log.
- **Adversarial review gates.** Review stages can be configured to push back rather than rubber-stamp. They target sycophancy, thin evidence, and work that looks busy without proving its claim. Work clears the gate when it survives the challenge.
- **Plan in batches, decide as work flows back.** Queue multiple work items at once; agents advance each through its stages independently while you handle approvals as they surface.
- **The workflow learns with you.** The first officer helps you adjust it when patterns emerge: a stage that never fires, a gate that keeps bouncing the same issue back, a schema field that always ends up empty.
- **Isolation when needed.** Stages that touch shared state run in their own git worktree; lightweight stages run inline. You declare which is which, and the first officer enforces it.
- **Work doesn't die at the context limit.** When an agent runs out of context, Spacedock swaps in a successor that carries forward what's in flight. Nothing gets lost in the handoff.

## Quick Start

**Prerequisites:** Claude Code or Codex CLI.

### Claude Code

1. Install the plugin:

   ```bash
   claude plugin marketplace add clkao/spacedock && claude plugin install spacedock
   ```

2. Commission a workflow with your own mission prompt:

   ```bash
   claude --agent spacedock:first-officer "/commission <your mission prompt>"
   ```

3. Or start from one of these example workflows — copy and run:

   **Email triage:**
   ```bash
   claude --agent spacedock:first-officer "/commission Email triage: fetch, categorize, and act on Gmail inbox. Entity: a batch of up to 50 emails. Stages: intake (use gws-cli, triage in:inbox and read email body if necessary, categorize, propose action per email, output as table) → approval (Captain reviews proposal) -> execute (carry out approved actions, do not mark as read). Use gws-cli (https://github.com/googleworkspace/cli/tree/main/skills/gws-gmail), GOOGLE_WORKSPACE_CLI_CONFIG_DIR=~/.config/gws/<account> for different accounts. Walk me through gws-cli setup if not already done."
   ```

   **[Superpowers](https://github.com/obra/superpowers)-style dev task workflow:**
   ```bash
   claude --agent spacedock:first-officer "/commission Dev task workflow: superpowers-style design → plan → implement → review with ## Design and ## Implementation Plan inlined in the entity body (no separate spec/plan files), implement on isolated worktrees with strict TDD, design and review gated for approval."
   ```

### Codex CLI

1. Clone Spacedock and start Codex from the repo root:

   ```bash
   git clone https://github.com/clkao/spacedock.git /path/to/spacedock
   cd /path/to/spacedock
   codex --enable multi_agent
   ```

2. Restart Codex if it was already open, then open `/plugins` and install **Spacedock** from the repo-local marketplace entry.

   The authoritative Codex plugin manifest is `.codex-plugin/plugin.json`, and the authoritative local catalog is `.agents/plugins/marketplace.json`. That catalog points to `./plugins/spacedock`, which is a checked-in symlink to the repository root so Codex loads the real plugin package directly.

3. Prompt Codex to use the first-officer skill and commission your workflow:

   ```bash
   Use the spacedock:first-officer skill to run /commission <your mission prompt> in this directory.
   ```

   Legacy compatibility: older Codex setups can still expose `~/.agents/skills/spacedock` directly:

   ```bash
   mkdir -p ~/.agents/skills
   ln -s /path/to/spacedock/skills ~/.agents/skills/spacedock
   ```

   The `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` files remain synchronized legacy mirrors of the Codex-first metadata for migration compatibility.

> Codex multi-agent is experimental. The Claude Code path is the primary supported surface.

## What a Work Item Looks Like

```yaml
---
id: 054
title: Session debrief command
status: done
---

Problem statement, design notes, acceptance criteria, and stage reports
all live in the body of this file as the work moves through stages.
```

See [a completed example](https://github.com/clkao/spacedock/blob/main/docs/plans/_archive/session-debrief.md) from Spacedock's own workflow.

## Concepts

| Concept | What it is |
|---------|------------|
| **Mission** | The purpose of the workflow: what it processes and what it delivers. |
| **Work item** | A single markdown file describing one thing being worked on: an email batch, a dev task, a draft. |
| **Workflow** | A directory of work items plus the README that defines stages, schema, and gates. |
| **Stage** | A named step a work item passes through (e.g. design, implement, review). |
| **Gate** | A pause point at a stage boundary where the captain approves, redirects, or bounces the work back. |

*"I am the master of my fate, I am the captain of my soul."* -- William Ernest Henley, *Invictus*

| Role | Who |
|---------|------------|
| **Captain** | You. You define the mission and make the calls at approval gates. |
| **First Officer** | The orchestrator agent that manages the workflow and reports to you at gates. |
| **Ensign** | The worker agent that moves a single item forward through one stage. |

## How It Works

The first officer reads the workflow README, checks work item statuses, and dispatches ensigns for items ready to advance. Stages that need isolation (typically implementation work with commits) run inside their own git worktree; lightweight stages (design, review, triage) run inline. At approval gates the first officer pauses and presents the ensign's stage report for your review: approve, redo with feedback, or reject. Rejected work automatically bounces back for revision in a fresh round of the earlier stage, with a hard cap so you never get stuck in an infinite loop. When you end a session, `/spacedock:debrief` captures what happened (commits, task state changes, decisions, open issues) into a record the next session picks up automatically (see [an example debrief](https://github.com/clkao/spacedock/blob/main/docs/plans/_debriefs/2026-04-09-01.md) from a real session).

## What Gets Generated

When you run `/spacedock:commission`, the following files are added to your workflow directory:

- **`{dir}/README.md`**: workflow schema, stage definitions, and work item template
- **`{dir}/*.md`**: seed work item files
- **`{dir}/_mods/`**: local modifications carried across refits

**Shipped by the Spacedock plugin:**

- **`spacedock:first-officer`**: the orchestrator agent that reads workflow state and dispatches ensigns
- **`spacedock:ensign`**: the worker agent dispatched to do stage work
- **`skills/commission/bin/status`**: read and advance workflow state without switching to a separate tracking tool

The generated workflow README is the single source of truth. The first officer reads it to know what stages exist, what quality criteria to enforce, and when to pause for your review.

Workflows can extend their own behavior via markdown mod files (`_mods/*.md`) that declare hook handlers for lifecycle events like startup, idle, or merge. For example, the [`pr-merge` mod](docs/plans/_mods/pr-merge.md) opens a pull request automatically when a completed worktree branch is ready to land.

When a new Spacedock release is available, use `/spacedock:refit` to upgrade your workflow scaffolding while keeping local modifications.

## Tips

- **Run Spacedock inside a sandbox.** Recommended: [agent-safehouse](https://github.com/eugene1g/agent-safehouse) (macOS), [packnplay](https://github.com/obra/packnplay), a devcontainer, or a VM.
- **Talk directly to an ensign.** Claude Code supports agent team chat: while a dispatched ensign is running, you can `Shift+Up` / `Shift+Down` to switch panes and give the ensign feedback directly instead of routing everything through the first officer.

## Use Cases

- **Email triage**: classify and route incoming messages with AI agents, escalate to a human at review gates
- **Dev task workflow**: [superpowers](https://github.com/obra/superpowers)-style design -> plan -> implement -> review with approval gates
- **Content publishing**: manage drafts through editing, review, and publication stages
- **Research workflows**: process papers or data through analysis, synthesis, and validation
- **Dogfooding Spacedock's own development.** Spacedock is self-hosted. Its own development runs on a plain text workflow at [`docs/plans/`](docs/plans/). Run `skills/commission/bin/status --workflow-dir docs/plans` to see the current state.

## License

Spacedock is released under the [Apache License 2.0](LICENSE).
