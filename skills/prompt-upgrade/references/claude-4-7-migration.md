# Claude Opus 4.6 → 4.7 Migration & Prompt Optimization Guide

> **Handoff context**: This document is intended as a reference for Claude Code running locally. Use it to audit and update existing prompts, API calls, and agent harnesses when migrating production workloads from `claude-opus-4-6` to `claude-opus-4-7`.
>
> **Release**: Claude Opus 4.7 shipped April 16, 2026. Same pricing as 4.6 ($5/$25 per MTok), but not a zero-effort drop-in.

---

## TL;DR

- **API breaking**: `temperature`, `top_p`, `top_k`, `thinking.budget_tokens`, prefilled last assistant turn, `interleaved-thinking-2025-05-14` header, `output_format` → all gone or deprecated.
- **Behavioral breaking**: more literal instruction following, shorter-by-default responses, fewer emoji / less warmth, fewer tool calls and subagents by default, new house design style.
- **New controls**: `xhigh` effort level, adaptive-only thinking, task budgets (beta), higher-res vision (2576px).
- **Cost**: same per-token price, but new tokenizer can produce 1.0–1.35× more tokens on the same input. Re-baseline with `count_tokens`.
- **Migration shape**: code changes are small and mechanical; prompt changes are where the real work is.

---

## 1. API Breaking Changes

Run through this list for every codepath that hits `claude-opus-4-7`.

| Change | What to do |
|---|---|
| `temperature`, `top_p`, `top_k` silently ignored | Remove from request builders; if migrating from 4.1 or earlier, non-default values return **400** |
| `thinking.budget_tokens` deprecated | Switch to `thinking: {type: "adaptive"}` + `output_config.effort` |
| Prefilled assistant messages on last turn | Returns **400** on Mythos Preview, deprecated on 4.6+. Replace with structured output / system prompt / XML scaffolding |
| `interleaved-thinking-2025-05-14` beta header | Remove — adaptive thinking now enables interleaved thinking automatically |
| `output_format` | Migrate to `output_config.format` |
| New `xhigh` effort level | Available between `high` and `max`; Claude Code defaults to `xhigh` |
| Tool versions (only if coming from 4.1 or earlier) | Update to `text_editor_20250728`, `code_execution_20250825` |
| Stop reasons (only if coming from 4.1 or earlier) | Handle `refusal` and `model_context_window_exceeded` |

### Before → After: thinking config

**Before (4.6 with extended thinking):**
```python
client.messages.create(
    model="claude-opus-4-6",
    max_tokens=64000,
    thinking={"type": "enabled", "budget_tokens": 32000},
    messages=[{"role": "user", "content": "..."}],
)
```

**After (4.7 with adaptive thinking):**
```python
client.messages.create(
    model="claude-opus-4-7",
    max_tokens=64000,
    thinking={"type": "adaptive"},
    output_config={"effort": "high"},
    messages=[{"role": "user", "content": "..."}],
)
```

**Rule of thumb**: when running at `max` or `xhigh`, set `max_tokens` to at least **64k** so the model has room to think, call tools, and coordinate subagents.

---

## 2. Behavioral Changes That Require Prompt Edits

These are not API errors — your prompts still "work" — but output quality or shape will shift. Audit each production prompt for the patterns below.

### 2.1 More literal instruction following

4.7 does **not** silently generalize an instruction from one item to another, and does **not** infer unstated requests. If you relied on "Claude will figure it out," you'll see gaps.

**Fix**: state the scope explicitly.

```
❌ "Format the section heading like this: …"
✅ "Apply this formatting to every section heading, not just the first one."
```

### 2.2 Response length is now adaptive

Simple lookups get shorter answers; open-ended analysis gets longer ones. No fixed verbosity default.

**To decrease verbosity**, add:
```
Provide concise, focused responses. Skip non-essential context, and keep examples minimal.
```

Positive examples (showing the ideal level of concision) outperform negative instructions ("don't be verbose").

### 2.3 Tone shifted: more direct, fewer emoji, less validation

4.7 drops the "Great question!" scaffolding and warm-up language of 4.6. If your product voice needs warmth (customer support, coaching, wellness), specify explicitly:

```
Use a warm, collaborative tone. Acknowledge the user's framing before answering.
```

### 2.4 Fewer tool calls, more reasoning

4.7 prefers to reason through problems before calling tools. Generally a win, but knowledge-work agents that depend on heavy search may under-call tools.

**Lever 1**: raise effort to `high` or `xhigh` (substantially increases tool usage).
**Lever 2**: explicit prompting about when/how to use each tool.

```
When the user asks about current events or any information that may have changed
since training, use the web_search tool before answering. Do not answer from memory.
```

### 2.5 Fewer subagents by default

4.7 spawns fewer subagents. If parallelization is required for your workflow, state it:

```
Do not spawn a subagent for work you can complete directly in a single response
(e.g. refactoring a function you can already see).

Spawn multiple subagents in the same turn when fanning out across items or reading
multiple files.
```

### 2.6 Remove 4.6-era scaffolding

These instructions are now noise (or actively harmful):

- "After every 3 tool calls, summarize progress" → 4.7 does this naturally
- "Double-check your output before returning" → baked in
- "Verify the slide layout / code / answer before responding" → baked in
- Aggressive caps-lock ("CRITICAL: You MUST use this tool when…") → dial back to normal language; 4.7 is more responsive to system prompts and will overtrigger

### 2.7 Progress updates during long traces

4.7 already provides good user-facing updates. Strip scaffolding that forces interim status messages. If you want a specific update format, describe it + give one example.

### 2.8 Frontend design house style

4.7 has a persistent default aesthetic: warm cream/off-white (`~#F4F1EA`), serif display type (Georgia / Fraunces / Playfair), italic accents, terracotta/amber highlights. Great for editorial and hospitality briefs. Wrong for dashboards, fintech, dev tools, healthcare, enterprise apps.

**The default is sticky.** Generic "don't use cream, make it clean and minimal" just swaps to another fixed palette. Two reliable approaches:

**A. Specify a concrete alternative** (colors, fonts, radii, spacing):
```
Use this tonal system: #E9ECEC, #C9D2D4, #8C9A9E, #44545B, #11171B.
Typography: square angular sans-serif, wider letter spacing in headings.
4px corner radius consistently across all surfaces.
```

**B. Have the model propose before building** (replaces temperature for variety):
```
Before building, propose 4 distinct visual directions tailored to this brief
(each as: bg hex / accent hex / typeface — one-line rationale).
Ask the user to pick one, then implement only that direction.
```

### 2.9 Code review harnesses: recall looks lower (usually isn't)

4.7 is **better** at bug finding (+11pp recall on one of Anthropic's hardest internal bug-finding evals). But if your review prompt says "only report high-severity" or "be conservative," 4.7 will follow that more faithfully than 4.6 did — it will investigate, find the bugs, then choose not to report ones it judges below your stated bar.

**Fix**: split "find" from "filter":
```
Report every issue you find, including ones you are uncertain about or consider
low-severity. Do not filter for importance or confidence at this stage — a separate
verification step will do that. Your goal here is coverage: it is better to surface
a finding that later gets filtered out than to silently drop a real bug. For each
finding, include your confidence level and an estimated severity so a downstream
filter can rank them.
```

---

## 3. Effort Level Strategy

Effort matters more on 4.7 than on any previous Opus. Calibrate explicitly.

| Level | When to use |
|---|---|
| `max` | Peak intelligence needs; can overthink and show diminishing returns. Test per task. |
| `xhigh` **(new)** | Best default for coding and agentic work. Claude Code default. |
| `high` | Minimum for most intelligence-sensitive use cases. Balanced. |
| `medium` | Cost-sensitive work; trades some intelligence. |
| `low` | Short scoped tasks, latency-sensitive, non-intelligence work. Strict scoping — may under-think on moderately complex tasks. |

**Hex's empirical observation**: low-effort 4.7 ≈ medium-effort 4.6. If your 4.6 pipeline ran at `medium`, try `low` on 4.7 first.

**If reasoning looks shallow**, raise effort before adding prompt scaffolding.

**If you must keep low for latency**, add targeted guidance:
```
This task involves multi-step reasoning. Think carefully through the problem before
responding.
```

**If the model over-thinks on simple queries** (can happen with large system prompts):
```
Thinking adds latency and should only be used when it will meaningfully improve
answer quality — typically for problems that require multi-step reasoning.
When in doubt, respond directly.
```

---

## 4. Cost & Tokenizer

- **List price unchanged**: $5/MTok input, $25/MTok output.
- **New tokenizer**: same content can consume **1.0–1.35×** more tokens (both input and output).
- **Interactive coding uses more tokens** than autonomous single-turn jobs because 4.7 reasons more after each user turn.
- **High-res images** consume more tokens (new max 2576px / 3.75MP vs 4.6's 1568px / 1.15MP).

**Before migrating production traffic**, run real prompts through the new tokenizer:

```python
import anthropic
client = anthropic.Anthropic()

result = client.messages.count_tokens(
    model="claude-opus-4-7",
    messages=[{"role": "user", "content": your_actual_prompt}],
)
print(result.input_tokens)
```

**Cost control levers**:
- Drop `xhigh` → `high` for tasks that don't need deep reasoning.
- Use task budgets (beta) to cap agentic sessions.
- Add "Be concise. Omit explanations unless asked." to system prompts.
- Route simple jobs (classification, extraction, simple generation) to Sonnet 4.6 at $3/$15.

---

## 5. Migration Checklist (one-sitting job)

Work through this in order.

### Code changes
- [ ] Grep and replace `claude-opus-4-6` → `claude-opus-4-7` (source, configs, DB-stored prompts, feature flags, runtime model selectors)
- [ ] Grep `temperature`, `top_p`, `top_k` → remove for Claude branches
- [ ] Grep `budget_tokens` → replace with `thinking: {type: "adaptive"}` + `output_config.effort`
- [ ] Remove `interleaved-thinking-2025-05-14` beta header
- [ ] Migrate `output_format` → `output_config.format`
- [ ] Find prefilled assistant messages on last turn → replace with structured output / system prompt
- [ ] Set `max_tokens` to >= 64k for workloads running at `xhigh` or `max`
- [ ] If streaming thinking to users, set `"display": "summarized"` so they see progress instead of a dead pause

### Prompt audit (for each production prompt)
- [ ] Remove 4.6-era scaffolding: "double-check," "verify before returning," "after N tool calls summarize"
- [ ] Dial back aggressive emphasis: `CRITICAL: You MUST…` → `Use this when…`
- [ ] Add explicit scope where 4.6 relied on silent generalization ("apply to every X, not just the first")
- [ ] Add verbosity instruction if you need specific output length
- [ ] Add tone instruction if you need warmth, formality, or a specific voice
- [ ] Add tool-use instruction if knowledge-work agent is under-calling tools
- [ ] Add subagent guidance if your workflow depends on parallelization
- [ ] For code-review harnesses: split "find" from "filter," require confidence + severity per finding
- [ ] For frontend prompts: specify concrete design tokens OR use the "propose 4 directions" pattern
- [ ] For long-context prompts: put longform data near the top, query/instructions after

### Cost & validation
- [ ] Run `count_tokens` on representative production prompts → project real cost delta
- [ ] A/B: run same prompt set on 4.6 and 4.7, compare eval scores, latency, token spend
- [ ] For code-review harnesses, measure both precision and recall (recall may look lower; check actual findings)

---

## 6. Rollout Guidance by Product Type

| Product type | Recommendation |
|---|---|
| Agentic coding, tool-heavy agents, Claude Code users | **Upgrade now.** Biggest wins in implicit-need inference, tool selection, fewer retries. |
| Computer use, screenshot agents, document analysis | **Upgrade now.** 3.75 MP vision is the biggest vision jump since vision shipped. |
| Plain chat apps, no tools | **No rush.** Small quality bump, small cost bump. Low-traffic deploy window. |
| Products hardcoding `temperature=0` for "determinism" | **Migrate code first, then ship.** Every request 400s until sampling fields come out. |
| Products streaming thinking to end users | **Include the `"display": "summarized"` fix in the same deploy** — otherwise users see a dead pause. |
| Security research tooling | **Apply to Cyber Verification Program first**, then upgrade. |
| Fine-tuned sampling params for creativity | **Evaluate carefully.** Variety now comes from effort + "propose options" pattern, not temperature. |

---

## 7. Required Reading (Primary Sources)

**Anthropic official — authoritative**
- Migration guide: https://platform.claude.com/docs/en/about-claude/models/migration-guide
- Prompting best practices: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
- What's new in Claude Opus 4.7: https://platform.claude.com/docs/en/about-claude/models/whats-new-claude-4-7
- Effort parameter: https://platform.claude.com/docs/en/build-with-claude/effort
- Adaptive thinking: https://platform.claude.com/docs/en/build-with-claude/adaptive-thinking
- Task budgets (beta): https://platform.claude.com/docs/en/build-with-claude/task-budgets
- Token counting: https://platform.claude.com/docs/en/build-with-claude/token-counting

**Third-party deep dives**
- OpenRouter Claude 4.7 migration summary: https://openrouter.ai/docs/guides/evaluate-and-optimize/model-migrations/claude-4-7
- Caylent deep dive: https://caylent.com/blog/claude-opus-4-7-deep-dive-capabilities-migration-and-the-new-economics-of-long-running-agents
- NxCode developer guide: https://www.nxcode.io/resources/news/claude-opus-4-7-developer-guide-api-claude-code-migration-2026
- KeepMyPrompts: https://www.keepmyprompts.com/en/blog/claude-opus-4-7-prompting-guide-whats-changed
- Rabinarayan Patra: https://www.rabinarayanpatra.com/blogs/claude-opus-4-7-release-and-migration-guide
- Digital Applied: https://www.digitalapplied.com/blog/claude-opus-4-7-complete-guide

---

## 8. Quick-Reference Prompt Snippets

Drop these into system prompts as needed.

**Concise output**
```
Provide concise, focused responses. Skip non-essential context, and keep examples minimal.
```

**Warm tone**
```
Use a warm, collaborative tone. Acknowledge the user's framing before answering.
```

**Proactive action**
```xml
<default_to_action>
By default, implement changes rather than only suggesting them. If the user's intent
is unclear, infer the most useful likely action and proceed, using tools to discover
any missing details instead of guessing.
</default_to_action>
```

**Conservative action**
```xml
<do_not_act_before_instructions>
Do not jump into implementation or change files unless clearly instructed to make
changes. When the user's intent is ambiguous, default to providing information,
research, and recommendations rather than taking action.
</do_not_act_before_instructions>
```

**Parallel tool calls**
```xml
<use_parallel_tool_calls>
If you intend to call multiple tools and there are no dependencies between them,
make all the independent tool calls in parallel. If some tool calls depend on
previous calls to inform parameters, call them sequentially instead.
Never use placeholders or guess missing parameters.
</use_parallel_tool_calls>
```

**Investigate before answering**
```xml
<investigate_before_answering>
Never speculate about code you have not opened. If the user references a specific
file, you MUST read the file before answering. Investigate and read relevant files
BEFORE answering questions about the codebase.
</investigate_before_answering>
```

**Avoid overengineering**
```
Only make changes that are directly requested or clearly necessary. Don't add features,
refactor code, or make improvements beyond what was asked. Don't add error handling
or validation for scenarios that can't happen. Don't create abstractions for one-time
operations. The right amount of complexity is the minimum needed for the current task.
```

**Context-window-aware agent**
```
Your context window will be automatically compacted as it approaches its limit,
allowing you to continue working indefinitely from where you left off. Do not stop
tasks early due to token budget concerns. As you approach your token budget limit,
save your current progress and state to memory before the context window refreshes.
```

**Code review — coverage over filtering**
```
Report every issue you find, including ones you are uncertain about or consider
low-severity. Do not filter for importance or confidence at this stage — a separate
verification step will do that. For each finding, include your confidence level and
an estimated severity so a downstream filter can rank them.
```
