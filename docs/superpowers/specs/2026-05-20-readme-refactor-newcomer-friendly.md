---
title: README refactor for newcomers (developer and non-developer)
date: 2026-05-20
status: approved
---

# Goal

Replace the current developer-leaning `README.md` with a doc set that lets a first-time reader (developer or not) answer two questions fast:

1. What is Spacedock?
2. How do I use it?

The current README assumes the reader is a working developer already comfortable with Claude Code, plugins, and the Star Trek metaphor. New users repeatedly trip on "what is this actually for" before they ever reach the install step. Spacedock is general purpose. Email triage, trip planning, tax and finance prep, content publishing, research synthesis, household admin, and job search all fit. The docs should show that range up front.

# Audience

Lead with non-developer framing; cover developers second. Both audiences get worked examples. The Star Trek terms (Captain, First Officer, Ensign) stay; they get explained once in the README and then used freely.

# Doc set

Five files. Cross-linked. Each one has a single clear job.

## `README.md`

Target length: roughly 120 lines.

Sections (in order):

1. One-sentence positioning that does not assume the reader is a developer.
2. "Is this for me?" with three short scenarios: a household example, a knowledge-worker example, a developer example. The point is to broadcast range.
3. "What is Spacedock?" Two paragraphs. Introduces the Captain / First Officer / Ensign metaphor exactly once with the plain-English equivalent in parentheses. Names what is different about it (approval gates with evidence, adversarial review, batching, learning workflow, isolation, work surviving context limits).
4. Five-minute quick start. Install Claude Code plugin, run one commission example, see output. The default example is the universal email triage case (works for almost anyone). Below that, a developer quick start for users who want to skip ahead.
5. Where to go next: GETTING_STARTED, USAGE, EXAMPLES, PROMPTS.
6. Licence.

What the README does NOT do: explain stage flags, explain the YAML schema, list every concept, walk through Codex setup, document mods. Those live in USAGE.

## `docs/GETTING_STARTED.md`

Target length: roughly 180 lines.

A walkthrough for the very first run. Picks one universal example (email triage) and one developer example (PR review) and shows them work end to end, including:

- Install
- Commission with the example mission string (copy-paste)
- What you should see in the terminal as the First Officer starts
- The first gate (Captain decision point) with sample output
- What happens after approval, after rejection
- How to end the session and resume it tomorrow (`/spacedock:debrief`)
- Common first-run gotchas

The whole point is: someone runs through this once, they have done it, they have value. No mental model required first.

## `docs/USAGE.md`

Target length: roughly 250 lines.

The mental model and reference. Sections:

1. When Spacedock helps and when it does not (paraphrased from the Notion design guide).
2. Vocabulary: Mission, Work item, Workflow, Stage, Gate. Captain, First Officer, Ensign. Plain-English first, jargon second.
3. The work-item file. What goes in the frontmatter, what goes in the body, how it evolves through stages.
4. Stages and the YAML schema. Each stage flag explained with one concrete sentence: `gate`, `worktree`, `fresh`, `feedback-to`, `parked`, `terminal`, `initial`, `concurrency`. Real example block at the end.
5. Approval gates and the adversarial review pattern. How rejection feedback flows back to the previous stage.
6. Refit and iteration. Workflows are not write-once. After two weeks of use, edit the YAML by hand or run `/spacedock:refit`.
7. Sessions, debrief, and context limits. Why work does not die at the context limit.
8. Mods at a glance. Pointer to the pr-merge mod as the canonical example. Note that mods are author-by-hand only; commission does not generate them.
9. Codex CLI path (short).

## `docs/EXAMPLES.md`

Target length: roughly 400 lines.

The cookbook. Eight worked examples. Each example has the same shape:

- Who this is for (one sentence)
- The recurring pain it removes
- The mission string (copy-paste, fenced)
- The stages and what each gate decides
- What success looks like after two weeks of use

Examples in order:

1. Email triage (Gmail via gws-cli, escalate-to-human gate)
2. Trip planning (research, itinerary draft, booking checkpoint, packing list)
3. Tax and finance prep (document intake, categorize, deductions review, summary for accountant)
4. Content publishing (idea capture, draft, edit, fact-check gate, publish)
5. Research synthesis (paper or source ingest, summarize, cross-reference, write-up)
6. Household admin (recurring bills, renewals, appointments, parental-school paperwork)
7. Job search (role intake, tailor materials, apply, follow-up cadence)
8. Developer track: PR review queue, Linear ticket ship workflow, cross-repo upgrade coordination (each presented compactly; deep-linked to the Notion design guide content)

## `docs/PROMPTS.md`

Target length: roughly 200 lines.

Three parts:

1. The fill-in-the-blank Initiating Prompt template. Designed to be pasted into Claude Code (or Codex). Asks Claude to read the local Spacedock repo, ask discovery questions about the user's recurring work, and recommend two or three workflows tailored to that work.
2. Notes on how to make it produce good answers (be specific about the recurring work, give Claude permission to look at recent history if the user keeps logs, name constraints like time budget).
3. Six worked variants. Each variant is a complete copy-paste prompt that personalises the template for: developer (the original Notion variant, sanitised), email triager, trip planner, household and finance, content creator, researcher.

# Style guardrails

These apply to every doc. Reviewer #2 enforces them.

- Zero em-dashes (the `—` character). Use a period, a comma, parentheses, or a colon.
- No emoji in body copy.
- ASCII quotes (`'` and `"`), not curly quotes.
- No `->` arrow where the word `to` works.
- Sentence case headings. Not Title Case Everywhere.
- Banned filler words: `robust`, `leverage`, `utilize`, `delve`, `in essence`, `comprehensive`, `seamless`, `powerful`, `cutting-edge`, `unlock`, `empower`, `streamline`, `harness`, `realm`, `landscape`, `journey`, `navigate the`, `dive deep`, `at the end of the day`.
- No reflexive `However` / `Moreover` / `Furthermore` paragraph openers.
- No closing summary paragraph that restates the section.
- Show, do not claim. If something is "easy" or "powerful," demonstrate it instead of saying so.
- Star Trek terms (Captain, First Officer, Ensign) introduced once in `README.md` with their plain-English equivalents in parentheses, then used freely in every doc afterwards.

# Branching and PR

- Branch: `docs/readme-refactor-newcomer-friendly` (already created from `main`).
- PR targets the `gcko/spacedock` fork, not the `clkao/spacedock` upstream.

# Process

1. Five writer agents in parallel, each handed: this spec, the relevant Notion excerpt, the existing README, and the style guardrails. Each writes one of the five docs.
2. Three review agents in parallel, each handed: all five drafts.
   - Clarity reviewer (does a first-time reader get it? is TTV under five minutes?)
   - AI-tell hygiene reviewer (em-dash sweep, banned word sweep, structural tells)
   - Accuracy reviewer (technical claims cross-checked against the repo)
3. Integrate review fixes. Manual `grep -n "—"` final sweep across all five files to verify zero em-dashes. Commit.

# Out of scope

- Rewriting the existing in-repo docs under `docs/superpowers/`, `docs/plans/`, `docs/research/`. They are working artefacts, not user-facing.
- Updating CONTRIBUTING.md or AGENTS.md (none of those changes are needed for the new-user path).
- Building a website. Just markdown in the repo.
