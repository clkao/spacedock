# Spacedock

Spacedock runs agent work through defined stages so you can delegate in batches and only weigh in on the calls that need your judgment.

You queue up the work, the agents move each item through its stages, and you get pulled in at approval gates with a stage report (findings, verdicts, artifacts, anomalies) ready for a yes or a no. No raw output dumps, no babysitting one chat at a time.

## Is this for me?

**You triage Gmail every morning.** Spacedock can fetch your inbox, sort receipts toward a tax folder, archive newsletters, and surface anything that smells like customer support back to you with a proposed reply. You approve the batch; it executes.

**You are planning two weeks in Japan.** Spacedock can research neighborhoods, draft an itinerary, and stop at the decisions that need you (which hotel in Kyoto, which day trip out of Tokyo). Once bookings are locked, the next stage produces a packing list and a daily run sheet.

**Your inbound code-review queue keeps piling up.** Spacedock can pull each open PR, run an adversarial review, queue the verdict for a thumbs up or down, and post the approved review to GitHub.

## What is Spacedock?

A workflow is a directory of markdown work item files plus a README that defines the stages, the schema, and the gates. There are three roles: Captain (you), First Officer (orchestrator), Ensign (worker). The First Officer reads the workflow README, dispatches Ensigns for items ready to advance, and pauses at gates to ask the Captain to approve or reject. (Rejection can bounce the item back to an earlier stage for revision; details in `docs/USAGE.md`.)

Spacedock is not a chat agent and not a single-skill loop. Gates present structured evidence so the Captain decides on findings, not transcript. Review gates can be adversarial: they push back instead of rubber-stamping. The Captain queues many work items and decides as each surfaces, instead of running one session at a time. When a pattern emerges (a stage that never fires, a gate that keeps bouncing), `/spacedock:refit` adjusts the workflow without losing local mods. Stages that touch shared state run in their own git worktree; lighter stages run inline. When an Ensign hits the context limit, a successor picks up the in-flight state from the markdown files and carries on.

## Quick start

Prerequisites: [Claude Code](https://docs.claude.com/en/docs/claude-code) installed (Anthropic's CLI; runs on macOS, Linux, and Windows via WSL). For the email example: a working `gws-cli` for Gmail (setup notes: https://github.com/googleworkspace/cli/tree/main/skills/gws-gmail; this includes a one-time Google OAuth flow).

Once those are in place, the steps below take about five minutes. With a clean machine and no tools installed, plan on twenty minutes total. Nothing runs against your inbox until the First Officer pauses at a gate and you approve.

1. Install the plugin:

   ```bash
   claude plugin marketplace add clkao/spacedock && claude plugin install spacedock
   ```

2. Commission an email triage workflow:

   ```bash
   claude --agent spacedock:first-officer "/spacedock:commission Email triage: fetch, categorize, and act on Gmail inbox. Entity: a batch of up to 50 emails. Stages: intake (use gws-cli, triage in:inbox and read email body if necessary, categorize, propose action per email, output as table) then approval (Captain reviews proposal) then execute (carry out approved actions, do not mark as read). Use gws-cli (https://github.com/googleworkspace/cli/tree/main/skills/gws-gmail), GOOGLE_WORKSPACE_CLI_CONFIG_DIR=~/.config/gws/<account> for different accounts. Walk me through gws-cli setup if not already done."
   ```

The First Officer commissions the workflow, dispatches an Ensign to gather your inbox, then pauses with a categorized proposal and waits for your approval before touching anything.

### If you are a developer

Same install line. Commission with this mission instead:

```bash
claude --agent spacedock:first-officer "/spacedock:commission Dev task workflow: superpowers-style design then plan then implement then review with ## Design and ## Implementation Plan inlined in the entity body (no separate spec/plan files), implement on isolated worktrees with strict TDD, design and review gated for approval."
```

The First Officer commissions a generic dev workflow with four stages, opens a worktree for implementation, and pauses at design and review for your call. For deeper dev shapes (PR review queue, Linear ticket ship, cross-repo upgrade), see [`docs/EXAMPLES.md`](docs/EXAMPLES.md).

## Codex CLI

The Codex CLI path is supported but experimental. See the Codex section of [`docs/USAGE.md`](docs/USAGE.md) for the setup steps and the plugin manifest layout.

## Where to go next

- [`docs/GETTING_STARTED.md`](docs/GETTING_STARTED.md): a guided first run, end to end.
- [`docs/USAGE.md`](docs/USAGE.md): the mental model, the YAML schema, and stage flags.
- [`docs/EXAMPLES.md`](docs/EXAMPLES.md): eight worked examples across household, knowledge work, and development.
- [`docs/PROMPTS.md`](docs/PROMPTS.md): the fill-in-the-blank Initiating Prompt template and persona variants.

## License

Spacedock is released under the [Apache License 2.0](LICENSE).
