# How Spacedock works

Spacedock has three roles. The Captain is you. The First Officer is the orchestrator agent that reads the workflow and decides what to do next. The Ensign is a worker agent dispatched to move a single work item through one stage. The basic loop: the First Officer reads the workflow, dispatches Ensigns to advance work items, and pauses at gates to ask the Captain for a call.

## When Spacedock helps and when it does not

Spacedock is a batch and approval layer that sits on top of skills. It does not replace skills. It pays off when work has natural pause points where you would want to glance at output before letting an agent move on, when work spans sessions so you come back tomorrow to the same item, or when you would otherwise re-run the same skill manually several times against your own output (the adversarial re-review pattern).

For one-shots, keep using ordinary skills. Looking up a Slack thread, creating a worktree, managing plugins, running `/clear` between thoughts: none of these need a workflow. Reach for Spacedock when there is a stream of similar work items moving through the same shape, or when a single item has enough phases that you want a record of what happened at each one.

## Vocabulary

| Concept | Plain English |
|---------|---------------|
| Mission | The purpose of the workflow: what it processes and what it delivers. |
| Work item | A single markdown file representing one thing being worked on (an email batch, a trip, a ticket, a draft). |
| Workflow | A directory of work items plus a README that defines stages, schema, and gates. |
| Stage | A named step a work item passes through (intake, design, review, etc.). |
| Gate | A pause point at a stage boundary where the Captain approves or rejects. |

| Role | Who |
|------|-----|
| Captain | You. Defines the mission and makes the calls at approval gates. |
| First Officer | Orchestrator agent. Reads the workflow, dispatches Ensigns, surfaces gates. |
| Ensign | Worker agent. Moves a single work item through one stage. |

## The work-item file

A work-item file is markdown with YAML frontmatter. Here is a concrete example:

```yaml
---
id: 054
title: Session debrief command
status: design
worktree:
pr:
verdict:
---

## Context
Background, links, what brought this in.

## Design
Sketched approach, alternatives considered, the choice.

## Acceptance criteria
- What "done" looks like.
- What must NOT regress.

## Captain feedback
(filled in when a gate rejects and bounces back)

## Stage reports
(Ensigns append their work summaries here as the item moves through stages)
```

The `status` field drives the stage. The First Officer reads it to know what to dispatch next and which gate, if any, applies.

The body grows as the item moves through stages. Each Ensign appends to it. Nothing is lost across sessions because the file holds the state, not the agent's memory.

## Stages and the YAML schema

Each workflow's `README.md` has a YAML block defining stages. Here is a representative block:

```yaml
stages:
  defaults:
    worktree: false
    concurrency: 2
  states:
    - name: backlog
      initial: true
    - name: design
      gate: true
    - name: implementation
      worktree: true
      concurrency: 1
    - name: validation
      worktree: true
      fresh: true
      feedback-to: implementation
    - name: ship
      parked: true
    - name: done
      terminal: true
```

| Flag | What it does | Default |
| --- | --- | --- |
| `initial: true` | Where new work items land when created. | false |
| `gate: true` | First Officer pauses and asks the Captain to approve or reject before advancing. | false |
| `worktree: true` | Stage runs inside an isolated git worktree. | false |
| `concurrency: N` | Maximum simultaneously-active worktree dispatches into this stage. Has no effect on stages without `worktree: true`. | 2 |
| `fresh: true` | Dispatches a brand-new Ensign with no prior session context (the manual `/clear` between phases). | false |
| `feedback-to: <stage>` | On rejection at a gate, status routes back to the named stage with the Captain's feedback included in the next Ensign's prompt. | absent |
| `parked: true` | Captain-facing convention marking a stage that is expected to wait on an external signal (PR merge, reply, an out-of-band action). The runtime does not enforce parking; a parked stage advances when the Captain or a mod transitions the entity out of it. | false |
| `terminal: true` | End of the workflow. | false |
| `agent: <name>` | Override the default agent (`spacedock:ensign`) for this stage. Useful for routing a stage to a specialized agent like `spacedock:first-officer` for orchestration work. | `spacedock:ensign` |
| `model: <id>` | Force a specific Claude model for the Ensign at this stage (e.g. `haiku`, `sonnet`, `opus`). Inherits from `stages.defaults.model` if set, otherwise uses the Ensign's default. | inherits |

The YAML is the artifact. The commission mission string is the spec. Running `/spacedock:commission` writes the YAML from the mission. If commission gets a flag wrong, edit the YAML by hand. The First Officer reads the workflow README at startup; a running session uses its in-memory copy of the workflow, so hand edits take effect on the next First Officer boot (close the session and reopen). The status binary always re-reads from disk, so `status --boot` and friends pick up the edit immediately.

Set `feedback-to:` on any gate that should bounce work back to an earlier stage on rejection. Without `feedback-to:`, a rejection has no defined bounce target. On rejection, the Captain gives a one-line reason at the gate prompt; longer feedback goes in the entity body under `## Captain feedback` before rejecting. The next Ensign reads both. The runtime caps feedback cycles at three rounds per stage; after that the entity escalates rather than looping forever.

The workflow README also carries an `id-style:` frontmatter field, set by commission, that chooses how new work items get their IDs: `sequential` (zero-padded numbers, the default), `sd-b32` (short collision-resistant IDs for collaborative or worktree-heavy workflows), or `slug` (kebab-case derived from titles or external identifiers like a Linear ticket or GitHub PR number). Stage names must match `[a-z0-9][a-z0-9-]*[a-z0-9]` (lowercase, kebab-case, no underscores); `status --validate` enforces this.

The workflow directory itself is wherever you ran `/spacedock:commission` from. It is a normal directory inside your project; you can move it, copy it, commit it, or delete it. Worktrees live at `.worktrees/<worker-key>-<slug>` under the repo root and clean up on terminal merge.

## Approval gates and adversarial review

When a stage has `gate: true`, the First Officer pauses, presents the Ensign's stage report (findings, verdicts, artifacts, anomalies), and asks the Captain to approve or reject. You have three responses:

1. Approve as-is. The next stage runs.
2. Edit the entity body, then approve. Your edits carry forward and the next stage uses them.
3. Reject. If `feedback-to: <prior-stage>` is set on this gate, the item routes back to that prior stage with your one-line gate-prompt reason and any `## Captain feedback` you added to the entity body. Without `feedback-to:`, the rejection has no defined bounce target.

Adversarial review is a stage configured to push back instead of confirm. Combine `gate: true`, `fresh: true`, and `feedback-to:` on a review stage. A clean Ensign reads the work cold, the Captain can challenge thin evidence, and rejection re-dispatches with a stronger frame. The intent is to replace the manual loop of rerunning a review skill with progressively stronger language: one stage, three flags, repeatable.

## Refit and iteration

Workflows are not write-once. Run a workflow for two weeks. Note which stages never fire and which gates keep bouncing the same kind of issue back. Then either edit the README YAML by hand or run `/spacedock:refit` for a guided pass.

A few tips:

- Use `gate: true` sparingly. Only at decision points where the agent has actually been wrong (verdicts, classifications, scope), not for things you would rubber-stamp.
- Keep stage names as buckets, not verbs. Good: `review`, `validation`, `merged`. Not good: `reviewing_now`, `awaiting_validation`.
- Four to six stages per workflow is the sweet spot. TDD does not need to be split into red, green, refactor stages. A single `implement` stage is fine.

## Sessions, debrief, and context limits

State lives in the work-item markdown files, not in the Ensign's session. When an Ensign runs out of context, Spacedock dispatches a successor that picks up from the file.

At the end of a working session, run `/spacedock:debrief` to record what happened: commits, status changes, decisions, open issues. The next session reads the debrief and continues from there.

The work item, not the session, is the unit of state. You can come back next week and the workflow still knows what is in flight.

## Mods at a glance

Mods are markdown files in `{workflow-dir}/_mods/` that declare hook handlers for lifecycle events like startup, idle, or merge. The canonical example is `mods/pr-merge.md`, which opens a pull request automatically when a completed worktree branch is ready to land. Mods extend a workflow without changing the core. Heads up: `/spacedock:commission` cannot scaffold new mods. It only copies pre-shipped mods from the plugin into `{workflow-dir}/_mods/`. Custom mods (Linear sync, GitHub PR intake, and so on) are authored by hand. See the PR review queue and Linear ticket ship examples in `EXAMPLES.md` for the patterns.

## Codex CLI

Spacedock works in Codex CLI through the multi-agent path, which is currently experimental. The Claude Code path is the primary supported surface.

```bash
git clone https://github.com/clkao/spacedock.git /path/to/spacedock
cd /path/to/spacedock
```

Then start Codex with multi-agent support enabled, and install Spacedock from the repo-local marketplace entry. The catalog lives at `.agents/plugins/marketplace.json` and points at `./plugins/spacedock`, which is a checked-in symlink to the repository root so Codex loads the real plugin package directly. The authoritative plugin manifest is `.codex-plugin/plugin.json`. The exact Codex install command varies by version; see your Codex docs for the current plugin install path.

Legacy fallback: older Codex setups that predate the repo-local marketplace can still expose Spacedock by manually symlinking the skills directory:

```bash
mkdir -p ~/.agents/skills
ln -s /path/to/spacedock/skills ~/.agents/skills/spacedock
```

The `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` files remain as synchronized legacy mirrors of the Codex-first metadata for migration compatibility.

Once installed, prompt Codex to use the first-officer skill:

```
Use the spacedock:first-officer skill to run /spacedock:commission <your mission prompt> in this directory.
```

## Running Spacedock safely

- Run Spacedock inside a sandbox. Recommended options: `agent-safehouse` (macOS), `packnplay`, a devcontainer, or a VM.
- Approve at gates with care. Approval is irreversible: the next stage executes as soon as you say yes. If you are not sure, reject; the bounce flow is the recovery mechanism.
- Run `git status` before approving a stage that ran in a worktree if you suspect uncommitted local changes.
- Stop a running workflow by closing the Claude Code session (Ctrl-D or `/quit`). The First Officer halts; work-item files keep the in-flight state for next time.

## Cost, data, and recovery

- **Cost.** Each Ensign dispatch is a Claude session. A workflow that runs many Ensigns (intake every few minutes, fresh adversarial review, successors on context limit) will spend tokens at the rate of those sessions. Use `model: haiku` on lightweight stages to cap cost; reserve Sonnet or Opus for stages that need it.
- **Data.** Inbox-, calendar-, and document-touching workflows send the data they read to Claude as part of the Ensign's session. Treat anything in a work-item file or in a stage report as something that has been read by Claude. If your organization restricts what may be sent to third-party LLMs, scope your workflows accordingly.
- **OAuth scopes.** Tools like `gws-cli` ask for Google OAuth scopes during their own setup, not through Spacedock. Read the tool's setup notes before authorising. Revocation is in your Google account's third-party access settings, not in Spacedock.
- **Mistakes.** Ensigns can be wrong. Build in protection at the workflow level: keep the destructive stage (send, file, book, publish) gated; let the Ensign propose the action and only execute on the Captain's approval. Email triage in `EXAMPLES.md` is shaped this way: `intake` writes a proposal, `approval` gates, only `execute` touches Gmail. If an action you approved turns out to be wrong, recover with the touched system's own tools (Gmail trash, `git revert`, the upstream service's undo).
- **Multiple workflows.** Spacedock does not orchestrate across workflows. Each is its own directory with its own First Officer session. Open one at a time.

## Where to go next

- `EXAMPLES.md` for eight worked examples (household, knowledge work, and three developer workflows).
- `PROMPTS.md` for an Initiating Prompt template that asks Claude to look at your recurring work and propose workflows shaped to it.
