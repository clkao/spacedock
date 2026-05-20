# Initiating prompts

Paste one of the prompts in this doc into Claude Code where Spacedock is checked out. Claude reads the project, asks you about your recurring work, and proposes two or three Spacedock workflows tailored to you. For the mental model, see `USAGE.md`. For copy-paste workflow examples, see `EXAMPLES.md`.

## The template (fill in the blanks)

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` so you understand what it is and how it works.
(If you are running in an environment without local filesystem access, ask me
to paste the contents of those two files into the chat instead.)

Here is the recurring work I want help with. List three to six items, each one
or two sentences with enough detail to be useful (volumes, tools, sensitivities):

<RECURRING WORK>
- <ITEM 1: what you do, how often, what makes it tedious>
- <ITEM 2: ...>
- <ITEM 3: ...>
</RECURRING WORK>

<OPTIONAL HISTORY DIRS>
You may mine my local Claude Code session history for patterns. Claude stores
per-project session histories under `~/.claude/projects/` by default. Look at
directories that match my active work:
- <PATH OR GLOB 1>
- <PATH OR GLOB 2>

Limit the scan to the last three months. Skip this paragraph entirely if I left
it blank.
</OPTIONAL HISTORY DIRS>

Based on what you read and what I told you, please:

1. Tell me which of my recurring items would actually pay off as a Spacedock
   workflow, and which would not.
2. Propose two or three example commissions as full copy-paste mission strings
   I can hand to `/spacedock:commission`.
3. Discuss which one to start with and why.
4. Call out anything that should NOT be a Spacedock workflow (one-shot work,
   single skill calls, things that do not have natural pause points).
5. End with one concrete next step I can do in the next ten minutes.
```

## Notes on making this work

1. Be specific about your recurring work. "Email" is not enough. "Triage 60 to 100 work emails every morning, route receipts to a tax folder, escalate customer-support smell to myself with a proposed reply" is.
2. Name constraints. Time budget per session, sensitivity rules (do not actually book things, do not actually file taxes), tools you already have set up (`gws-cli`, `gh`, Linear MCP, Notion MCP).
3. If you have local history, point Claude at it. Otherwise skip that paragraph; the prompt still works without it.
4. Ask Claude to start small. One workflow run for two weeks beats four workflows on day one.
5. Read `EXAMPLES.md` after Claude proposes a mission. Compare its mission string against the example closest to your persona to sanity-check stage names and flags.

## Variant: Developer

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` first.

My recurring developer work:
- Feature development on two or three active repos.
- Code refactoring passes when a module gets unwieldy.
- Code quality improvements (typing, tests, lint debt).
- Pull request reviews, both mine and others'.

Mine my local Claude history for patterns. Claude Code stores per-project
session histories under `~/.claude/projects/`. Look at directories matching my
active repos. The directory names encode the absolute path with slashes
replaced by dashes; keep the prefix that matches my layout:

- `~/.claude/projects/-Users-<USER>-repos-<ORG>-<REPO 1>`
- `~/.claude/projects/-Users-<USER>-repos-<ORG>-<REPO 2>`
- `~/.claude/projects/-Users-<USER>-repos-<ORG>-<REPO 3>`

Or, if my active repos are: <ACTIVE REPOS>, find the matching directories
yourself.

Limit the scan to the last three months. I care about what kinds of work I
actually do repeatedly, not one-off sessions.

Then please:

1. Give me directions on how to use Spacedock effectively for this kind of work.
2. Propose two or three example commissions as full copy-paste mission strings.
3. Specifically propose at least one workflow that uses worktrees for isolation
   on a code-bearing stage, and at least one that uses an adversarial review
   gate (`fresh: true` + `gate: true` + `feedback-to: <prior-stage>`) so review
   does not run in the same context as implementation.
4. Suggest which workflow to start with and why.
5. Call out anything I do that should stay a one-shot skill call, not a workflow.
6. End with one concrete next step.
```

## Variant: Email triager

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` first.

My recurring work:
- Triage Gmail every morning. Roughly <N> messages, mix of work, customer
  support, vendors, and newsletters.
- Reply to a long tail of customer-support emails that need a real answer.
- Archive newsletters I do not act on.
- Sort receipts into a folder for monthly bookkeeping.

Tools I have set up: `gws-cli` for Gmail.

Constraints:
- Do not auto-reply. Drafts only.
- Surface a proposal for the Captain (me) to approve before anything is sent or
  archived in bulk.
- Sensitive senders (named contacts, my manager, anything tagged Important) go
  to a manual queue, not the auto-archive.

Please:

1. Propose one commission as a full copy-paste mission string to start with.
2. Propose one optional second commission for later if the first works.
3. Tell me which stages should have gates and why.
4. End with one concrete next step.
```

## Variant: Trip planner

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` first.

My recurring work:
- Plan multi-week trips two or three times a year.
- Research destinations (neighborhoods, transit, food, day trips).
- Draft itineraries that survive contact with reality.
- Identify the booking decisions that need to happen and in what order.
- Build packing lists tailored to the trip.

Constraints:
- Do not actually book anything. Ever.
- Collect options with prices and tradeoffs, then surface a decision pass for
  the Captain (me) to make the call.
- Keep one work-item file per trip so I can pick the same trip up next weekend.

Please:

1. Propose one commission as a full copy-paste mission string for my next
   planned trip: <TRIP DESCRIPTION, DATES, ROUGH SHAPE>.
2. Propose a small variation of the same workflow for shorter trips (a long
   weekend), since the booking-decision stage probably collapses.
3. Tell me which stages should be gates.
4. End with one concrete next step.
```

## Variant: Household and finance

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` first.

My recurring work:
- Track recurring bills and subscription renewals so nothing surprises me.
- Categorize transactions for year-end tax prep.
- Manage kids' school paperwork, forms, and appointments.
- Review household admin once a week so the backlog does not grow.

Constraints:
- Do not pay anything. Do not file anything.
- Produce a clear summary I review before I act.
- Anything financial goes through an explicit Captain-approval gate.

Please:

1. Propose one commission as a full copy-paste mission string focused on the
   next concrete pain: <NEXT CONCRETE PAIN>.
2. Tell me which stages should be gates and which should be auto-advance.
3. Call out anything in my list that should stay a one-shot, not a workflow.
4. End with one concrete next step I can do in ten minutes.
```

## Variant: Content creator

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` first.

My recurring work:
- Capture ideas as they come in (notes, voice memos, links).
- Draft pieces from captured ideas.
- Edit drafts (structure pass, line edit, tighten).
- Fact-check claims and links before publishing.
- Publish to <PRIMARY VENUE>.
- Post to social after publishing.

My publishing cadence: <CADENCE, e.g. weekly newsletter, monthly essay, two
shorts per week>.

Constraints:
- Do not publish without an approval gate. The Captain (me) signs off on the
  final draft before anything goes live.
- Fact-check stage runs against a clean Ensign so it cannot rubber-stamp the
  drafting Ensign's claims.

Please:

1. Propose one commission as a full copy-paste mission string that fits my
   actual cadence.
2. Tell me which stages need gates and which need a fresh Ensign.
3. End with one concrete next step.
```

## Variant: Researcher

```markdown
I have Spacedock checked out locally at `<PATH TO SPACEDOCK>`. Please read its
`README.md` and `docs/USAGE.md` first.

My recurring work:
- Ingest papers and transcripts on an active research thread.
- Summarize per source: claims, methods, evidence quality, where it fits.
- Cross-reference across sources to find agreements, conflicts, and gaps.
- Draft write-ups for handoff (to coauthors, to a blog post, to a memo).

My current research thread: <RESEARCH THREAD>.

Constraints:
- The synthesis pass must be an explicit Captain-approval gate. Do not let the
  workflow ship a write-up without me reviewing the cross-reference output.
- Per-source summaries can auto-advance; synthesis cannot.

Please:

1. Propose one commission as a full copy-paste mission string for my current
   thread.
2. Tell me which stages need gates and which benefit from a fresh Ensign.
3. End with one concrete next step.
```

## After Claude responds

1. Read `EXAMPLES.md` for the closest worked example. Compare Claude's proposed mission against it side by side and adjust stage names or flags that drift. The Household and finance variant straddles two examples; compare against both example 3 (Tax and finance prep) and example 6 (Household admin).
2. Commission the one workflow Claude recommends. Run it for two weeks before adding a second.
3. Edit the generated `{workflow-dir}/README.md` directly if a flag is wrong. The First Officer reads it on every loop, so a hand edit takes effect on the next run with no restart.
