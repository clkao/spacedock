# Examples

This cookbook has eight worked examples. Each one names the audience, the recurring pain it removes, the mission string to paste, the stages, the gates, and what success looks like after two weeks of use.

Pick the closest example, adapt the mission text to your situation, and paste it into the First Officer.

## 1. Email triage

**Who this is for**: anyone with a Gmail inbox that needs daily attention.

**Recurring pain it removes**: opening Gmail every morning to triage by hand, missing important messages, and replying to the same kinds of emails over and over.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Email triage: fetch, categorize, and act on Gmail inbox. Entity: a batch of up to 50 emails. Stages: intake (use gws-cli, triage in:inbox and read email body if necessary, categorize, propose action per email, output as table) then approval (Captain reviews proposal) then execute (carry out approved actions, do not mark as read). Use gws-cli (https://github.com/googleworkspace/cli/tree/main/skills/gws-gmail), GOOGLE_WORKSPACE_CLI_CONFIG_DIR=~/.config/gws/<account> for different accounts. Walk me through gws-cli setup if not already done."
```

### Stages

| Stage | What the Ensign does | What the gate decides |
| --- | --- | --- |
| `intake` | Pulls `in:inbox`, reads bodies where the subject is ambiguous, categorizes, and writes a proposed action per email into a table. | None. Stage exit is automatic. |
| `approval` | Renders the proposal table. | `gate: true`, `feedback-to: intake`. Captain approves; to send rows back, edit the work-item file and reject. |
| `execute` | Carries out the approved actions (file, archive, draft reply). Does not mark as read. | Terminal. |

### What success looks like

Morning triage drops to one approval pass per batch. Receipts get routed to tax folders without manual sorting, newsletters get archived, and only the messages that need a real response remain in the inbox. If you keep correcting categorizations on rejection, the workflow's prompt evolves with your edits and the approval pass shortens.

## 2. Trip planning

**Who this is for**: someone planning a multi-week or complex trip.

**Recurring pain it removes**: research scattered across browser tabs, the itinerary buried in a doc that never gets reviewed, and bookings done in a rush at the last minute.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Trip planning: shape one trip per entity (destination plus dates). Stages: research (collect neighborhoods, sights, transit notes, weather windows into the entity body) then itinerary (draft a day-by-day plan with decision points called out) then decisions (gate: Captain picks lodging, day trips, and dining priorities) then booking (parked: the Captain executes bookings off-platform; mark which bookings to make, do not actually book) then packing (generate a packing list from the locked itinerary). Use the entity body as the working document; do not create side files."
```

### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `research` | Gathers neighborhoods, sights, transit, and weather notes. Writes them into the entity body. | None. |
| `itinerary` | Drafts a day-by-day plan with decision points highlighted. | None. |
| `decisions` | Surfaces the lodging and day-trip choices. | `gate: true`. Captain picks. |
| `booking` | Lists what to book (links, times, confirmation numbers field empty). | `parked: true`. Waits for the Captain to actually book and paste confirmations back. |
| `packing` | Generates a packing list keyed off climate windows and the locked itinerary. | Terminal. |

### What success looks like

The itinerary is finalized in two short sessions instead of three weeks of tab-juggling. Decisions are made on evidence (neighborhood notes, transit times) instead of guesswork. The packing list is automatic and aware of weather, dress codes, and travel days.

## 3. Tax and finance prep

**Who this is for**: a freelancer or household preparing tax filings or a quarterly finance review.

**Recurring pain it removes**: receipts and statements scattered across email and folders, categorizing transactions is mind-numbing, and deductions get missed.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Tax and finance prep: one entity per tax year or quarter. Stages: intake (collect documents from a designated folder, list what is present and flag what is missing; stay parked while missing documents trickle in) then categorize (Ensign categorizes line items into expense buckets; Captain corrects edge cases inline) then deductions-review (gate: Captain reviews the proposed deductions list with rationale per item; rejection bounces back to categorize) then summary (produce a clean export bundle for the accountant). Inputs live in ~/Documents/tax/<year>; outputs go into the entity body."
```

### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `intake` | Lists every document found in the year folder, names what is missing (W-2, 1099-NEC, brokerage statements, charitable receipts). | `parked: true` while documents trickle in. |
| `categorize` | Bins line items into expense categories with confidence notes on edge cases. | None. |
| `deductions-review` | Proposes deductions with one-line rationale per item. | `gate: true`, `feedback-to: categorize`. Rejection bounces to `categorize`. |
| `summary` | Builds a clean accountant-ready export (CSV plus a one-pager). | Terminal. |

### What success looks like

Filing season collapses from a marathon weekend into three approval passes spread across two weeks. Nothing falls through the cracks because the workflow knows exactly what is missing and parks itself until the document shows up. The accountant gets a tidy bundle instead of a shoebox.

## 4. Content publishing

**Who this is for**: anyone who publishes regularly (a newsletter, a blog, an internal update).

**Recurring pain it removes**: drafts stall mid-edit, fact-checking gets skipped under deadline, and the publishing checklist lives in someone's head.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Content publishing: one entity per piece (essay or newsletter issue). Stages: idea (capture the angle and source notes in the entity body) then draft (Ensign produces a first draft from the notes) then edit (Captain edits in the entity body) then fact-check (gate: Ensign verifies claims and flags anything unsourced; rejection bounces back to edit) then publish (Captain hits publish; Ensign prepares social posts and updates the entity to terminal). Working text lives in the entity body."
```

### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `idea` | Captures the angle, the audience, and source notes. | None. |
| `draft` | Produces a first draft from the idea notes. | None. |
| `edit` | Captain rewrites in the entity body. Ensign is idle. | None. |
| `fact-check` | Verifies claims, flags unsourced statements. | `gate: true`, `feedback-to: edit`. Rejection routes back to `edit`. |
| `publish` | Captain publishes. Ensign drafts social posts. | Terminal. |

### What success looks like

The regular cadence sticks because nothing is in the Captain's head. Fact errors are caught before publish instead of after. The backlog of half-finished drafts shrinks because every piece is in a known stage with a known next move.

## 5. Research synthesis

**Who this is for**: a researcher or analyst ingesting papers, transcripts, or interview notes.

**Recurring pain it removes**: source material piles up, synthesis happens once at the end and badly, and cross-references between sources are missed.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Research synthesis: one entity per research thread (a question plus its sources). Stages: intake (list sources, attach abstracts and provenance) then summarize (Ensign produces a summary per source with quoted evidence) then cross-reference (Ensign identifies overlaps, agreements, and contradictions across sources) then synthesis (gate: Captain reviews the synthesis; rejection routes back to cross-reference with one-line feedback) then write-up (Ensign drafts a handoff-ready write-up). Working notes live in the entity body."
```

### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `intake` | Lists every source, attaches abstracts and citation info. | None. |
| `summarize` | One summary per source with quoted evidence so claims are traceable. | None. |
| `cross-reference` | Surfaces overlaps and contradictions between sources. | None. |
| `synthesis` | Pulls the cross-reference notes into a single argument. | `gate: true`, `feedback-to: cross-reference`. Rejection routes back to `cross-reference`. |
| `write-up` | Drafts a handoff-ready write-up the Captain can pass on. | Terminal. |

### What success looks like

A question that used to take a quiet weekend gets shaped over a week of approval passes. Contradictions surface before the write-up, not after a reviewer points them out. The final write-up traces every claim back to a source.

## 6. Household admin

**Who this is for**: someone running a household: bills, renewals, kids' school paperwork, appointments.

**Recurring pain it removes**: things slip until they become urgent, the same items recur every year but nothing tracks them, and appointments stack on the same Tuesday.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Household admin: one entity per admin item (a renewal, an appointment, a form). Stages: intake (Captain or an inbox mod creates items) then triage (Ensign proposes priority and deadline) then action (gate: Captain approves the proposed action) then follow-up (parked until a date or a reply) then closed (terminal). Keep the entity body short; this is a tracker, not a doc."
```

### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `intake` | Captain or a mod creates new items. | None. |
| `triage` | Proposes priority and a deadline based on the item type. | None. |
| `action` | Lists the proposed action (call, file, schedule). | `gate: true`. Captain approves. |
| `follow-up` | Waits for a reply or a date. | `parked: true`. |
| `closed` | Item resolved. | Terminal. |

### What success looks like

The household runs on the workflow instead of on memory. Renewals get handled a week early because the workflow surfaces them on its own clock. Appointments stop colliding because triage proposes a deadline up front instead of letting items pile onto a single day.

## 7. Job search

**Who this is for**: someone running an active job search across one or several open roles.

**Recurring pain it removes**: resumes and cover letters get written in a panic, follow-ups are forgotten, and interviewing momentum is lost between weeks.

### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Job search: one entity per role (a company plus a job posting). Stages: intake (capture posting text, contact name, deadline) then tailor (Ensign drafts a resume and cover letter tuned to the posting; Captain edits in the entity body) then apply (gate: Captain confirms send) then follow-up (parked until response or a follow-up date) then interview (Captain logs notes per round into the entity body) then outcome (terminal: offer, rejection, or withdrawn). Working text lives in the entity body."
```

### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `intake` | Captures posting, contact, deadline. | None. |
| `tailor` | Drafts a resume and cover letter tuned to the posting. Captain edits. | None. |
| `apply` | Final review of the materials. | `gate: true`. Captain confirms send. |
| `follow-up` | Waits for a reply or a follow-up date. | `parked: true`. |
| `interview` | Captain logs notes round by round into the entity body. | None. |
| `outcome` | Offer, rejection, or withdrawn. | Terminal. |

### What success looks like

The search runs in parallel across many roles without dropping any. Per-role materials accumulate as a library you can reuse on the next round. Follow-ups happen on time because the workflow surfaces parked items on their follow-up date.

## 8. Software development

Three developer workflows. They share a shape: the entity is one unit of work, the implementation stages run on isolated worktrees, and review is a fresh adversarial pass instead of self-review.

### PR review queue

**Who this is for**: a developer who is regularly added as a requested reviewer on GitHub PRs.

**Recurring pain it removes**: the queue piles up silently, reviews end up rubber-stamped under time pressure, and rejected PRs do not get a real second pass.

#### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission PR review queue for PRs where I am set as a requested reviewer. Entity: a single GitHub PR awaiting my review. Auto-intake is provided by a hand-authored mod at _mods/pr-review-intake.md. The mod runs on the First Officer's idle hook with a self-imposed 20-minute minimum between GitHub polls, creates entities for new PRs, and auto-archives entities whose PRs are merged, closed, converted to draft, or whose review request was removed. Stages: intake (auto-populated by the mod; multiple entities can sit here simultaneously while waiting their turn) then review (concurrency: 1, only one PR is reviewed at a time; run an antagonistic review skill; assume the worst, look for hidden brittleness, verify test coverage; output severity-tagged findings into the entity body) then verdict (gate: Captain approves the verdict APPROVE or REQUEST_CHANGES or NEEDS_DEEPER_REVIEW; on rejection bounce back to review with one-line feedback for a fresh adversarial pass) then posted (terminal: an Ensign here posts the approved review to GitHub via gh pr review). Use worktree on review for branch inspection. Set id-style to slug so entity filenames can be {owner}-{repo}-pr-{number}. Decline the pr-merge mod when offered; this workflow does not create PRs."
```

> Heads up: commission cannot scaffold new mods. It only copies pre-shipped ones. The `pr-review-intake.md` mod referenced above has to be authored by hand and dropped into `{workflow-dir}/_mods/` after commission finishes. Order does not matter; the First Officer re-scans `_mods/` on every loop.

#### Stages

| Stage | What the Ensign does | Flags and gate |
| --- | --- | --- |
| `intake` | Mod-populated. Many PRs can sit here. | None. |
| `review` | Runs an antagonistic review skill, writes severity-tagged findings into the entity body. | `worktree: true`, `concurrency: 1`. |
| `verdict` | Surfaces the proposed verdict for Captain approval. | `gate: true`, `feedback-to: review`. Rejection bounces to `review` with feedback for a fresh pass. |
| `posted` | An Ensign posts the approved review to GitHub via `gh pr review --approve` or `--request-changes`. | Terminal. |

#### What success looks like

The review queue clears on a daily pass. Antagonistic re-runs happen automatically on rejection instead of by hand. Nothing sits in your queue silently because the mod auto-archives PRs that no longer need you.

### Linear ticket ship workflow

**Who this is for**: a developer shipping Linear tickets end to end.

**Recurring pain it removes**: tickets stretch across multiple sessions, the Linear ticket stays in Todo while you ship code locally, and review feels stale because it ran in the same context as implementation.

#### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Linear ticket ship workflow: one entity per Linear ticket assigned to me. Auto-intake is provided by a hand-authored mod at _mods/linear-intake.md. Stages: intake (mod-populated, captain-curated, gate, concurrency: 100; the mod creates the entity but never auto-promotes) then triage (gate: classify the ticket, pick the affected repo, escalate if cross-repo) then design (gate: write Design and Acceptance Criteria into the entity body) then implement (worktree, concurrency: 1, TDD; mod transitions Linear to In Progress on stage entry) then review (worktree, fresh, gate, feedback-to: implement; dispatch a separate Ensign for an antagonistic review) then ship (parked: open the PR; mod transitions Linear to In Review when the PR field is set) then merged (terminal; pr-merge mod advances when the PR lands on main; mod transitions Linear to Done). Accept the pr-merge mod when offered."
```

> Heads up: the `linear-intake.md` mod is hand-authored, like the PR review intake mod above. Commission only copies pre-shipped mods (today that means `pr-merge` only). Drop your `linear-intake.md` into `{workflow-dir}/_mods/` after commission finishes.

#### Stages

| Stage | Role | Flags and gate |
| --- | --- | --- |
| `intake` | Mod creates entities from Linear. | `gate: true`, captain-curated. (Concurrency is unbounded in practice: intake entities have no worktree, so the `concurrency` flag is a documentation marker, not enforced.) |
| `triage` | Classify, pick the affected repo. | `gate: true`. |
| `design` | Write Design and Acceptance Criteria into the entity body. | `gate: true`. |
| `implement` | TDD on an isolated branch. | `worktree: true`, `concurrency: 1`. Mod sets Linear to In Progress. |
| `review` | A fresh Ensign runs an antagonistic review. | `worktree: true`, `fresh: true`, `gate: true`, `feedback-to: implement`. |
| `ship` | Open the PR. | `parked: true`. Mod sets Linear to In Review. |
| `merged` | PR merged. | Terminal. Mod sets Linear to Done. |

#### What success looks like

Tickets ship and Linear stays in sync because the mod updates the ticket state on every stage entry. Review reads the diff cold on a fresh worktree, so it catches problems an in-session reviewer would skim past. Multiple PRs can be in flight and the workflow holds the state for each.

### Cross-repo upgrade coordination

**Who this is for**: a developer coordinating a dependency or framework upgrade that touches an upstream package and one or more downstream consumers.

**Recurring pain it removes**: pairing notes get lost across sessions, the consumer breaks because the upstream PR was not actually published, and verification ends up running in the same context as implementation.

#### Mission

```bash
claude --agent spacedock:first-officer "/spacedock:commission Cross-repo upgrade coordination: one entity per upgrade initiative (for example MUI v7 to v9, axios to fetch, Jest to Vitest). Stages: scope (gate: list every consumer call site, propose a phased plan) then upstream (worktree in the OSS package repo; implement, ship a PR, must merge and publish before consumer work begins) then downstream (worktree in the consumer repo; pull the new version, fix breakages, ship a paired PR) then verify (gate, fresh: run full test suites in both repos; rejection routes back to downstream) then done (terminal). Park between upstream merge and downstream start to wait on publish. Accept the pr-merge mod when offered for both implementation stages."
```

#### Stages

| Stage | What happens | Flags and gate |
| --- | --- | --- |
| `scope` | List call sites, propose a phased plan. | `gate: true`. |
| `upstream` | Implement and ship in the OSS package repo. Must merge and publish. | `worktree: true`. pr-merge mod advances the stage. |
| (parked between) | Wait on publish before downstream starts. | `parked: true` on entry to `downstream` until the version is live. |
| `downstream` | Pull the new version, fix breakages, ship a paired PR. | `worktree: true`. pr-merge mod advances. |
| `verify` | Run full test suites in both repos. | `gate: true`, `fresh: true`, `feedback-to: downstream`. Rejection routes to `downstream`. |
| `done` | Both PRs merged, both suites green. | Terminal. |

#### What success looks like

Pairing notes live in the entity body, so they survive sessions and context limits. Consumer work waits for the upstream package to publish because the downstream stage is parked. Verification reads both repos cold because it dispatches a fresh Ensign.
