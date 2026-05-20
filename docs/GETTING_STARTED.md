# Getting started

This guide has two complete walkthroughs (email triage and pull request review). Pick whichever fits your work. Each walkthrough takes about five minutes end to end. The mental model lives in [`USAGE.md`](USAGE.md); the cookbook of more examples lives in [`EXAMPLES.md`](EXAMPLES.md).

## Before you start

- Claude Code installed. See https://docs.claude.com/en/docs/claude-code.
- A sandbox is recommended. On macOS try [agent-safehouse](https://github.com/clkao/agent-safehouse); elsewhere a devcontainer or a VM works.
- For the email walkthrough: a configured `gws-cli` for the Gmail account you want triaged. Setup notes at https://github.com/googleworkspace/cli/tree/main/skills/gws-gmail.
- For the developer walkthrough: a git repository to point Spacedock at.

## Walkthrough 1: Email triage

### Step 1: Install the plugin

```bash
claude plugin marketplace add clkao/spacedock && claude plugin install spacedock
```

### Step 2: Commission the workflow

```bash
claude --agent spacedock:first-officer "/spacedock:commission Email triage: fetch, categorize, and act on Gmail inbox. Entity: a batch of up to 50 emails. Stages: intake (use gws-cli, triage in:inbox and read email body if necessary, categorize, propose action per email, output as table) then approval (Captain reviews proposal) then execute (carry out approved actions, do not mark as read). Use gws-cli (https://github.com/googleworkspace/cli/tree/main/skills/gws-gmail), GOOGLE_WORKSPACE_CLI_CONFIG_DIR=~/.config/gws/<account> for different accounts. Walk me through gws-cli setup if not already done."
```

The mission describes the entity (a batch of up to 50 emails), the stages (intake, approval, execute), the tool to use (`gws-cli`), and the constraint that execute must not mark messages as read. Commission turns that prose into a workflow directory plus a README that the First Officer will read on every loop. Nothing executes against your inbox at this point. The workflow files appear on disk, and that is it until the First Officer dispatches the first Ensign.

### Step 3: Watch the First Officer start up

The First Officer reads the new workflow README, prints the stage list it found, scaffolds the work-item file for the first batch, and dispatches an Ensign to the intake stage. You will see output that names the workflow, lists the stages it found, and announces the dispatch. The exact wording varies by Claude Code version; an illustrative shape:

```
First Officer (illustrative)
  workflow: email-triage
  stages: intake then approval then execute
  dispatching ensign to intake (entity: batch-001)
```

The Ensign then runs `gws-cli`, walks your inbox, and writes its findings into the batch markdown file.

### Step 4: Your first gate

When intake finishes, the First Officer pauses at the approval gate and shows you the proposal the Ensign produced. A trimmed example:

```
batch-001 intake report

| from              | subject                  | category | proposed action       |
|-------------------|--------------------------|----------|-----------------------|
| stripe@stripe.com | Receipt #4421            | archive  | move to Receipts 2026 |
| pat@acme.co       | RE: contract redlines    | reply    | draft a 3-line reply  |
| security@aws.com  | Unusual sign-in detected | escalate | surface to Captain    |

approve / redirect / reject?
```

You answer with `approve`, `redirect`, or `reject`. Redirect lets you edit the table inline (recategorize a row, change a proposed action) and re-submit. Reject sends the batch back to intake.

### Step 5: Approve and execute

On approval, the First Officer dispatches an Ensign to the execute stage. That Ensign runs each approved action through `gws-cli`: archive the receipt, draft the reply in your Drafts folder, leave the escalation untouched. The batch file gets a closing report (what ran, what skipped, any failures), and the entity moves to the terminal stage.

On rejection, the batch bounces back to intake with your feedback recorded in the work-item file. The next intake Ensign reads that feedback and produces a revised proposal instead of starting fresh.

### Step 6: End the session

When you are done for the day:

```
/spacedock:debrief
```

Debrief captures the commits, state transitions, decisions, and any open issues into a structured record. Tomorrow, opening the same workflow picks up exactly where you left off because the state lives in the markdown files, not in chat history.

## Walkthrough 2: Pull request review

### Step 1: Install the plugin

```bash
claude plugin marketplace add clkao/spacedock && claude plugin install spacedock
```

### Step 2: Commission the workflow

```bash
claude --agent spacedock:first-officer "/spacedock:commission Dev task workflow: superpowers-style design then plan then implement then review with ## Design and ## Implementation Plan inlined in the entity body (no separate spec/plan files), implement on isolated worktrees with strict TDD, design and review gated for approval."
```

The mission asks for four stages (design, plan, implement, review), inlined design and plan sections in each entity, isolated worktrees for implementation, and approval gates around design and review.

### Step 3: Watch the First Officer start up

The First Officer scaffolds the workflow directory, prints the stage list, and waits for you to seed work-item files (one per ticket or PR you want shipped). You can drop a markdown file into the entities directory by hand, or ask the First Officer to create one from a Linear ticket or PR URL. Once an entity exists, the First Officer dispatches an Ensign to the design stage.

```
First Officer (illustrative)
  workflow: dev-task
  stages: design then plan then implement then review
  no entities yet; seed one to begin
```

### Step 4: Your first gate

When the design Ensign finishes, the First Officer pauses at the design gate and shows you the proposed `## Design` section inlined in the entity body: the problem statement, the chosen approach, the tradeoffs considered, and the test contracts the implement stage will be held to. You answer `approve`, `redirect`, or `reject`. Redirect lets you push back on a specific tradeoff; reject sends design back with your notes.

### Step 5: Approve and execute

On approval, the entity advances through plan and then into implement. The implement stage runs inside an isolated git worktree so the working tree of your main checkout is untouched. The Ensign writes failing tests first, makes them pass, and commits in small increments. When implement finishes, the review gate fires: an adversarial review Ensign reads the diff and either signs off or files specific objections you decide on.

If you reject review, the entity bounces back to implement with the objection text baked in, and the next Ensign starts from there.

### Step 6: End the session

```
/spacedock:debrief
```

Same flow as the email walkthrough: the next session reads the markdown and resumes whichever entities were mid-flight.

## Common first-run gotchas

- The first commission does not actually execute work; it scaffolds the workflow directory and README. Work starts on the next loop or when you re-invoke the First Officer.
- If a stage that should run in a worktree complains about uncommitted changes, commit or stash them first; Spacedock will not silently overwrite local edits.
- Approval gates pause the First Officer; the workflow does not advance until the Captain answers. This is by design.
- If you want to bounce a stage back with feedback, reject (do not approve), and Spacedock will re-dispatch the previous stage with your feedback baked in.
- Commission cannot scaffold custom mods. It can only copy pre-shipped ones (currently just `pr-merge`). Custom mods are authored by hand in `_mods/`.
- The plugin is the source of truth for stage flags; the generated `{workflow-dir}/README.md` controls per-workflow behavior. If commission gets the YAML flags wrong, edit the YAML; the First Officer reads it on every loop.

## Where to go next

- [`USAGE.md`](USAGE.md) for the mental model and the YAML schema.
- [`EXAMPLES.md`](EXAMPLES.md) for the remaining workflows: six non-developer examples (trip planning, taxes, content publishing, research synthesis, household admin, job search) plus the developer cluster (PR review queue, Linear ticket ship, cross-repo upgrade coordination).
- [`PROMPTS.md`](PROMPTS.md) for an Initiating Prompt template that asks Claude to look at your recurring work and propose tailored workflows.
