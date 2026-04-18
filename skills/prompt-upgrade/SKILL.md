---
name: prompt-upgrade
description: "Audit and upgrade Claude prompts, skills, and API configurations from Claude 4.6 to 4.7. Use when migrating prompts to 4.7, auditing for deprecated API patterns, or optimizing prompt instructions for 4.7's behavioral changes. Triggers on 'upgrade prompt', 'migrate to 4.7', 'prompt upgrade', '4.6 to 4.7', 'audit for 4.7'."
user-invocable: true
---

# Prompt Upgrade: Claude 4.6 → 4.7

You are auditing and upgrading Claude prompts, skills, and API configurations from Claude Opus 4.6 to 4.7. This skill scans target files for deprecated API patterns, outdated prompt scaffolding, and missing 4.7 optimizations, then produces a structured report with optional auto-fix.

The authoritative migration reference is at `skills/prompt-upgrade/references/claude-4-7-migration.md` (relative to the Spacedock plugin directory). Read it at the start of every invocation — it contains the full pattern catalog, before/after examples, and prompt snippets.

Follow these five phases in order. Do not skip phases, but skip individual checks within a phase if `--scope` excludes them.

---

## Phase 0: Parse Args and Load Reference

### Step 1 — Extract arguments

Parse the user's invocation for:

- **`{target}`** — a file path, directory path, or glob pattern. If absent, default to scanning `skills/` in the current project root (resolved via git root or cwd).
- **`{apply}`** — boolean. Present if the user passes `--apply`. Default: `false` (report-only mode).
- **`{scope}`** — one of `api`, `prompt`, or `all`. Present if the user passes `--scope <value>`. Default: `all`.

Store these as working variables for the rest of the skill.

### Step 2 — Load the migration reference

Read `skills/prompt-upgrade/references/claude-4-7-migration.md` from the Spacedock plugin directory. This is your pattern catalog — every finding you report must cite a section from this document.

If the file is missing, stop and report:

> Cannot find migration reference at `skills/prompt-upgrade/references/claude-4-7-migration.md`. Is the prompt-upgrade skill installed?

---

## Phase 1: Discovery

### Step 1 — Resolve target files

Use Glob and Read to build the file list:

- If `{target}` is a single file: use that file.
- If `{target}` is a directory: scan for `**/*.md`, `**/*.py`, `**/*.ts`, `**/*.js`, `**/*.json`, `**/*.yaml`, `**/*.yml` within it. Exclude `node_modules/`, `.git/`, `dist/`, `build/`, `__pycache__/`.
- If `{target}` is a glob: expand it directly.

### Step 2 — Classify each file

Assign each file one of these types:

| Type | Pattern | Audit phases |
|------|---------|-------------|
| `skill-md` | `**/SKILL.md`, `**/skills/**/*.md` | Prompt audit (Phase 3) |
| `agent-md` | `**/agents/**/*.md`, `**/*.agent.md` | Prompt audit (Phase 3) |
| `prompt-md` | Other `.md` files containing system prompt content, CLAUDE.md files | Prompt audit (Phase 3) |
| `code` | `.py`, `.ts`, `.js` files | API audit (Phase 2) + inline prompt audit (Phase 3) |
| `config` | `.json`, `.yaml`, `.yml` files | API audit (Phase 2) |
| `skip` | Everything else | None |

To detect `prompt-md` vs plain documentation: use Grep to check if the file contains patterns like `system prompt`, `<system>`, `You are`, `role.*system`, or CLAUDE.md in the filename. Files without prompt-like content are classified as `skip`.

### Step 3 — Report discovery

Print a summary for {captain}:

```
## Discovery

Target: {target}
Files found: {total}
  - skill-md: {n}
  - agent-md: {n}
  - prompt-md: {n}
  - code: {n}
  - config: {n}
  - skipped: {n}

Proceeding with {scope} audit...
```

If zero auditable files are found, stop and report: "No auditable files found in `{target}`."

---

## Phase 2: API/Code Audit

**Skip this phase if `{scope}` is `prompt`.**

For every file classified as `code` or `config`, run these checks using Grep. Collect all findings into `{api_findings}` — a list of objects with: `file`, `line`, `pattern`, `severity`, `description`, `before`, `after`, `ref_section`, `auto_fixable`.

### Check 2.1 — Sampling parameters

Grep for `temperature`, `top_p`, `top_k` in API call contexts (not in comments or documentation strings).

- **Severity**: `breaking` (returns 400 if non-default values on pre-4.6 migration) or `warning` (silently ignored on 4.6+)
- **Fix**: Remove the parameter
- **Ref**: Section 1, row 1
- **Auto-fixable**: Yes (line removal)

### Check 2.2 — budget_tokens

Grep for `budget_tokens`.

- **Severity**: `deprecated`
- **Fix**: Replace `thinking: {"type": "enabled", "budget_tokens": N}` with `thinking: {"type": "adaptive"}` + `output_config: {"effort": "high"}` (or appropriate level)
- **Ref**: Section 1, row 2
- **Auto-fixable**: No (requires choosing an effort level)

### Check 2.3 — Interleaved thinking header

Grep for `interleaved-thinking-2025-05-14` or `interleaved.thinking`.

- **Severity**: `deprecated`
- **Fix**: Remove the header entirely
- **Ref**: Section 1, row 4
- **Auto-fixable**: Yes (line removal)

### Check 2.4 — output_format

Grep for `output_format` (not `output_config`) in API call contexts.

- **Severity**: `deprecated`
- **Fix**: Replace with `output_config.format` or `output_config: {"format": ...}`
- **Ref**: Section 1, row 5
- **Auto-fixable**: No (structural change)

### Check 2.5 — Prefilled assistant messages

Grep for patterns indicating prefilled assistant content on the last message turn: `"role":\s*"assistant"` as the final message in a messages array, or `prefill` in variable/comment names near message construction.

- **Severity**: `breaking`
- **Fix**: Replace with structured output, system prompt instruction, or XML scaffolding
- **Ref**: Section 1, row 3
- **Auto-fixable**: No (requires redesign)

### Check 2.6 — Model string

Grep for `claude-opus-4-6`, `claude-sonnet-4-6`, or other 4.6-era model strings.

- **Severity**: `info`
- **Fix**: Replace with `claude-opus-4-7` (or keep 4.6 if intentional)
- **Ref**: Section 5, Code changes checklist item 1
- **Auto-fixable**: Yes (string replacement) — but confirm with {captain} first since model pinning may be intentional

### Check 2.7 — max_tokens for high-effort workloads

If a file contains `xhigh` or `max` effort AND `max_tokens` is set below 64000:

- **Severity**: `warning`
- **Fix**: Set `max_tokens` to at least 64000
- **Ref**: Section 1, thinking config "Rule of thumb"
- **Auto-fixable**: No (value judgment)

---

## Phase 3: Prompt/Skill Audit

**Skip this phase if `{scope}` is `api`.**

For every file classified as `skill-md`, `agent-md`, `prompt-md`, or `code` (for inline prompt strings), run these checks. Collect findings into `{prompt_findings}`.

### Check 3.1 — 4.6-era scaffolding (removal candidates)

Use Grep (case-insensitive) to search for these patterns:

| Pattern | What to look for |
|---------|-----------------|
| Progress scaffolding | "after every N tool calls", "summarize progress", "provide a status update every" |
| Self-verification | "double-check", "verify before returning", "verify the .* before responding", "check your .* before" |
| Aggressive emphasis | Lines with 3+ consecutive caps words (`[A-Z]{2,}(\s+[A-Z]{2,}){2,}`), "CRITICAL:", "IMPORTANT:", "YOU MUST", "ALWAYS:", "NEVER:" when used as section-level directives |

For each match:
- **Severity**: `scaffolding`
- **Description**: Explain why this is no longer needed (4.7 does this naturally)
- **Ref**: Section 2.6 or 2.7
- **Auto-fixable**: No (removal is safe but {captain} should review what replaces it)

### Check 3.2 — Missing scope explicitness

Read each prompt file. Look for instructions that apply a rule to a single example without stating it applies broadly. This is a judgment call — flag patterns like:

- "Format the heading like this: ..." (without "apply to every heading")
- "Use this style for the first section" (without "and all subsequent sections")
- Instructions that demonstrate on one item but should apply to all

- **Severity**: `opportunity`
- **Ref**: Section 2.1
- **Auto-fixable**: No

### Check 3.3 — Missing verbosity/tone guidance

Check if the prompt has any explicit verbosity or tone instructions. If the prompt is longer than 500 words and contains no verbosity/tone guidance:

- **Severity**: `opportunity`
- **Description**: "4.7 defaults shorter and more direct. If this prompt needs a specific output length or tone, add explicit guidance."
- **Ref**: Section 2.2, 2.3
- **Auto-fixable**: No
- **Suggestion**: Include relevant snippet from Section 8 of the migration guide

### Check 3.4 — Missing tool-use guidance

For agent/skill prompts that reference tools: check if there are explicit instructions about when to use each tool. If tools are mentioned but no usage guidance exists:

- **Severity**: `opportunity`
- **Description**: "4.7 calls fewer tools by default, preferring to reason first. If this agent needs proactive tool use, add explicit instructions."
- **Ref**: Section 2.4
- **Auto-fixable**: No

### Check 3.5 — Missing subagent guidance

For orchestrator-style prompts (ones that mention "subagent", "dispatch", "spawn", "fan out", or "parallel"): check if there is explicit guidance on when to use subagents vs. work inline.

- **Severity**: `opportunity`
- **Description**: "4.7 spawns fewer subagents by default. If this workflow depends on parallelization, state it explicitly."
- **Ref**: Section 2.5
- **Auto-fixable**: No

### Check 3.6 — Frontend design prompts

For prompts that deal with UI generation, CSS, frontend design, or component building: check if they specify concrete design tokens (colors, fonts, radii, spacing). If not:

- **Severity**: `warning`
- **Description**: "4.7 has a sticky default aesthetic (cream/off-white, serif display type, terracotta highlights). Frontend prompts without concrete design tokens will get this default. Specify colors/fonts/radii or use the 'propose 4 directions' pattern."
- **Ref**: Section 2.8
- **Auto-fixable**: No

### Check 3.7 — Code review find/filter conflation

For prompts that contain code review instructions: check if "find" and "filter" are conflated (e.g., "only report high-severity issues", "be conservative in what you report", "focus on important bugs").

- **Severity**: `warning`
- **Description**: "4.7 follows filtering instructions more literally. Split 'find' from 'filter' — have the model report everything, then filter in a separate step."
- **Ref**: Section 2.9
- **Auto-fixable**: No

---

## Phase 4: Report

Generate the structured report. Print it directly to {captain}.

```
## Prompt Upgrade Report: 4.6 → 4.7

### Summary
- Files scanned: {total_files}
- API issues: {api_count} ({auto_fixable_count} auto-fixable)
- Prompt issues: {prompt_count}
- Opportunities: {opportunity_count}

### API Breaking Changes
{for each api_finding with severity "breaking":}
**{pattern}** — `{file}:{line}`
> {description}
> **Before:** `{before}`
> **After:** `{after}`
> **Ref:** Migration guide {ref_section}
{end}

### API Deprecations
{for each api_finding with severity "deprecated":}
**{pattern}** — `{file}:{line}`
> {description}
> **Ref:** Migration guide {ref_section}
{end}

### Prompt Scaffolding to Remove
{for each prompt_finding with severity "scaffolding":}
**{pattern}** — `{file}:{line}`
> {description}
> **Ref:** Migration guide {ref_section}
{end}

### Prompt Warnings
{for each prompt_finding with severity "warning":}
**{pattern}** — `{file}:{line}`
> {description}
> **Ref:** Migration guide {ref_section}
{end}

### Opportunities
{for each finding with severity "opportunity" or "info":}
**{pattern}** — `{file}:{line}`
> {description}
> **Ref:** Migration guide {ref_section}
{end}

### Safe Auto-Fixes Available
{for each finding where auto_fixable is true:}
- `{file}:{line}` — {description} ({before} → {after})
{end}
{if no auto-fixable findings: "No auto-fixable changes found."}
```

If any section is empty, print "(none)" under it. Do not omit sections.

---

## Phase 5: Apply

### If `{apply}` is false (default)

After printing the report, ask {captain}:

> Would you like me to apply the {auto_fixable_count} safe auto-fixes? (y/n)
>
> Safe auto-fixes are limited to:
> - Removing deprecated sampling parameters (temperature, top_p, top_k)
> - Removing deprecated beta headers
> - Model string replacement (after confirmation)
>
> Prompt rewording, effort level selection, and structural API changes are always manual.

If {captain} declines, stop. If {captain} confirms, proceed to the apply steps below.

### If `{apply}` is true (or {captain} confirmed)

For each auto-fixable finding:

1. Read the file with Read
2. Show the specific change as a diff:
   ```
   Applying: {description}
   File: {file}
   Line: {line}
   - {before}
   + {after}
   ```
3. Apply the change with Edit
4. Track the change in `{applied_changes}`

After all changes are applied, print:

```
## Applied Changes

{n} changes applied:
{for each applied change:}
- {file}:{line} — {description}
{end}

{m} findings remain (manual review required):
{for each non-auto-fixable finding:}
- {file}:{line} — {description}
{end}
```

### Model string replacement safety

When replacing model strings (`claude-opus-4-6` → `claude-opus-4-7`), always confirm with {captain} first even with `--apply`, because:
- The project may intentionally pin to 4.6 for specific workloads
- Model routing logic may have conditional branches that should stay on 4.6
- Config files may contain model strings for multiple providers

Ask:

> Found {n} model string(s) referencing claude-opus-4-6. Replace all with claude-opus-4-7?
>
> {list each file:line}
>
> (y/n/pick — 'pick' to choose individually)

---

## Design Principles

These are invariants — do not violate them regardless of args or context.

1. **Report-first**: Always show findings before changing anything. Even with `--apply`, print the report first.
2. **Conservative auto-fix**: Only auto-apply mechanical changes (parameter removal, header removal, confirmed model string replacement). Never auto-apply prompt rewording, tone changes, or structural API changes.
3. **Reference-backed**: Every finding must cite a section from `references/claude-4-7-migration.md`. If you find something that doesn't map to the reference, report it under "Unlisted" with a note that it may be a false positive.
4. **Idempotent**: Running twice on the same files produces the same report (minus already-applied fixes). Do not create state files or modify non-target files.
5. **Non-destructive**: Never delete files. All changes are line-level edits within existing files.
