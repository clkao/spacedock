---
name: refit
description: "This skill should be used when the user asks to \"refit a workflow\", \"upgrade a workflow\", \"update workflow scaffolding\", or wants to bring an existing workflow's scaffolding files up to date with the current Spacedock version."
user-invocable: true
---

# Refit a Workflow

You are refitting (upgrading) an existing workflow to match the current Spacedock version. This covers scaffolding files (README, mods) and, when schema changes require it, migrating entity frontmatter data. Agent files and the status viewer are shipped with the Spacedock plugin and do not need local updates.

Follow these five phases in order. Do not skip or combine phases.

---

## Phase 1: Discovery

### Step 1 — Identify the workflow

The user must provide a workflow directory path. If they didn't, ask:

> Which workflow directory should I refit?

Store the confirmed path as `{dir}`. Resolve it to an absolute path. Also derive `{project_root}` (git root or cwd) and `{dir_basename}` (last path component).

### Step 2 — Read current scaffolding and extract version stamps

Read each scaffolding file and extract its version stamp:

1. **README** — Read `{dir}/README.md`. Extract version from YAML frontmatter `commissioned-by: spacedock@X.Y.Z`. Store as `{readme_version}`.
2. **Mod files** — Scan `{dir}/_mods/*.md` for installed mods. For each, read the `version` frontmatter field. Match against canonical mods at `{spacedock_plugin_dir}/mods/{name}.md` by filename. Store the local version and canonical version for each. Also scan `{spacedock_plugin_dir}/mods/*.md` for mods not yet installed.
3. **Legacy status script** — If `{dir}/status` exists, note it for cleanup (status now ships with the plugin).

If a file doesn't exist, note it as missing and skip it.

### Step 3 — Read current Spacedock version

Read `.codex-plugin/plugin.json` from the Spacedock plugin directory (the directory containing the `skills/` folder — resolve from your own plugin context). Treat `.agents/plugins/marketplace.json` as the authoritative local marketplace surface and `.claude-plugin/*` as synchronized legacy mirrors. Extract the `version` field from `.codex-plugin/plugin.json` and store it as `{current_version}`.

### Step 4 — Evaluate

- If all version stamps match `{current_version}`: report "Workflow is already up to date." and stop.
- If no stamps were found on any versioned file (README): enter **Degraded Mode** (see below).
- Otherwise: proceed to Phase 2 with the list of outdated files.

---

## Phase 2: Classify Files by Upgrade Strategy

Each scaffolding file gets a specific upgrade strategy based on how safe it is to auto-replace:

| File | Strategy | Rationale |
|------|----------|-----------|
| `README.md` | **Show diff** | Users customize stages, schema fields, quality criteria. Too risky to auto-replace. Show what the current template would produce and let the captain decide. |
| `_mods/{name}.md` | **Version diff** | Compare `version` frontmatter against canonical. Show diff if changed, ask for confirmation. |
| `status` (legacy) | **Remove** | Status viewer now ships with the plugin. Remove the workflow-local copy if present. |

Present the classification to the captain:

> **Upgrade plan:**
>
> | File | Current State | Strategy |
> |------|--------------|----------|
> | `status` | {status_version or "no stamp"} | Replace |
> | `README.md` | {readme_version or "no stamp"} | Show diff (manual review) |
> | `_mods/{name}.md` (for each) | {local_version vs canonical_version / custom mod} | Version diff |
>
> Proceed with this upgrade plan? (y/n)

For mods, include all files found in `_mods/`.

Wait for the captain to confirm before proceeding.

---

## Phase 3: Execute Upgrades

### Extract workflow-specific values from README

Before generating any files, read `{dir}/README.md` and extract:

1. **Mission** — from the `# {title}` heading (first H1).
2. **Stages** — from the `## Stages` section. Each `### \`{stage_name}\`` subsection is a stage, in order. For each stage, extract:
   - Stage name
   - Inputs, Outputs, Good, Bad descriptions
   - Whether "Human approval: Yes" appears (indicates an approval gate for the transition INTO this stage)
3. **Schema fields** — from the `## Schema` section's YAML block.
4. **Entity description** — from the first paragraph after the H1.

### 3a. Legacy Status Script (Remove)

If `{dir}/status` exists, it's a legacy workflow-local status script. The status viewer now ships with the Spacedock plugin at `skills/commission/bin/status`.

1. Inform the captain: "The status viewer is now plugin-shipped. Removing the workflow-local `{dir}/status` script."
2. `git rm {dir}/status`

### 3b. README (Show Diff)

1. Generate what the current commission template would produce for this workflow, using the extracted values (mission, stages, schema, etc.).
2. Diff it against the user's current README.
3. Present the diff to the captain, noting which differences are likely template changes vs user customizations:

> **README template diff:**
>
> The following differences exist between your README and what the current template would generate. Differences may be template improvements or your intentional customizations.
>
> {diff output}
>
> I have NOT modified your README. Review the diff and apply any changes you want manually, or tell me which specific changes to make.

Do NOT auto-modify the README. The captain decides what to adopt.

### 3c. Mods (Version diff)

For each mod file in `{dir}/_mods/*.md`:

1. **Match against canonical** — Check if `{spacedock_plugin_dir}/mods/{name}.md` exists (matching by filename).

2. **If canonical exists** — Compare the `version` frontmatter field in the local file against the canonical file.
   - If versions match: report "up to date" and skip.
   - If versions differ: show a diff between the local and canonical files. Ask the captain:

> **Mod update: {name}** (local: {local_version} → canonical: {canonical_version})
> {diff output}
>
> Update this mod? (y/n)

3. **If no canonical match** — The mod is custom (user-authored or third-party). Acknowledge neutrally:

> Found custom mod: **{name}** — {description from frontmatter}. No canonical version to compare against. No action needed.

4. **New mods available** — For each canonical mod in `{spacedock_plugin_dir}/mods/*.md` not present in `{dir}/_mods/`, offer to install:

> New mod available: **{name}** — {description from frontmatter}. Install it? (y/n)

   If accepted, copy the canonical file to `{dir}/_mods/{name}.md`. Create `_mods/` if it doesn't exist.

### 3d. Legacy Migration (pr-lieutenant → pr-merge mod)

If `{project_root}/.claude/agents/pr-lieutenant.md` exists and `{dir}/_mods/pr-merge.md` does not, offer migration:

> **Migration: pr-lieutenant → pr-merge mod**
>
> Your workflow uses a pr-lieutenant agent for PR management. Spacedock now uses mods instead. I'll:
> 1. Create `{dir}/_mods/pr-merge.md` with the PR management mod
> 2. Remove `agent: pr-lieutenant` from any README stage entries
> 3. Regenerate the first-officer with mod discovery
>
> The pr-lieutenant agent file at `.claude/agents/pr-lieutenant.md` will be left in place. You can delete it manually if no longer needed.
>
> Proceed with pr-lieutenant migration? (y/n)

If accepted:
1. Copy `{spacedock_plugin_dir}/mods/pr-merge.md` to `{dir}/_mods/pr-merge.md`
2. Edit the README frontmatter: remove `agent: pr-lieutenant` from any stage entries

---

## Phase 4: Migrate Entity Data

After upgrading scaffolding, check whether schema changes require migrating existing entity data.

### ID Style Preservation

README frontmatter may contain `id-style: sequential`, `id-style: generated`, or `id-style: slug`. Refit must preserve the current `id-style`; do not silently change it. Report the current style in the refit summary, recommend `generated` only when collaboration pressure exists (worktree stages, PR/merge mods, multiple creators, branches, or offline work), and require explicit captain approval before changing README frontmatter.

Before any approved style change, run `status --validate` against the workflow and report failures. Existing sequential workflows require no data rewrite. Generated style requires each entity to store a 24-character lowercase Crockford Base32 full ID while status displays shortest unique prefixes. Slug style derives identity from filenames and makes `status --next-id` non-applicable. Migration is manual migration in this release: use validation to make the change safe, and defer rewrite automation or bulk frontmatter rewriting to a separate tracked task.

### Step 1 — Detect schema changes

Compare the old README's `## Schema` and `### Field Reference` sections against the new version. Look for:

- **Changed field types or ranges** (e.g., score changed from integer/25 to float/0.0–1.0)
- **Renamed fields** (e.g., `priority` → `score`)
- **Removed fields** (fields in entities that are no longer in the schema)
- **New required fields** (fields added to the schema that existing entities lack)

If no schema changes affect entity data, skip to Phase 5.

### Step 2 — Scan entities

For each detected schema change, scan all entity files in `{dir}/*.md` (excluding README.md) and identify which entities have values in the affected fields.

Present findings to the captain:

> **Schema migration needed:**
>
> {description of what changed in the schema}
>
> **Affected entities:**
> {list of entities with current values that need migration}
>
> **Proposed migration:**
> {what the migration would do — e.g., "Convert score from /25 to 0.0–1.0 by dividing by 25"}
>
> Apply this migration? (y/n)

### Step 3 — Execute migration

On the captain's approval, update the affected entity frontmatter fields. Use the Edit tool — never rewrite whole entity files. Only touch the specific fields identified in the migration plan.

Show a summary of what was migrated:

> **Migrated {N} entities:**
> {list of entity: old_value → new_value}

---

## Phase 5: Finalize

1. Update version stamp to `{current_version}` in the README YAML frontmatter: `commissioned-by: spacedock@{current_version}`.
2. Show a summary:

> **Refit complete:**
>
> | File | Action |
> |------|--------|
> | `status` | {Removed (legacy) / Not present} |
> | `_mods/{name}.md` (for each) | {Updated / Already current / Installed / Custom (no action)} |
> | `README.md` | {Stamp updated / User-reviewed / No changes} |
>
> Suggest committing:
> ```
> git commit -m "refit: upgrade workflow scaffolding to spacedock@{current_version}"
> ```

---

## Degraded Mode (No Version Stamp)

When no version stamps are found on the README, the original baseline cannot be determined. Inform the captain and offer two options:

> **No version stamps found.** This workflow was commissioned before version stamping was implemented, or the stamps were removed. I can't determine what the original scaffolding looked like.
>
> Two options:
>
> 1. **Stamp only** — Add version stamps to existing files without changing anything else. This establishes a baseline for future refits.
> 2. **Full refit with review** — Generate what the current templates would produce and show a full diff for every scaffolding file. You review and approve each change.
>
> Which option?

### Option 1: Stamp Only

Add version stamps to versioned files without modifying anything else:

- **README.md** — Add YAML frontmatter with `commissioned-by: spacedock@{current_version}` (wrap in `---` delimiters if frontmatter doesn't exist).
- **status** — Insert `# commissioned-by: spacedock@{current_version}` as the second line (after `#!/bin/bash`).

### Option 2: Full Refit with Review

Execute Phase 3, but show a full diff for every file (including status) and require the captain's explicit approval before replacing each one. Never auto-replace files without a version stamp — the risk of overwriting customizations is too high.

---

## Safety Rules

- **Never modify entity file bodies** — only frontmatter, and only during an approved schema migration.
- **Never auto-replace without a version stamp** — always enter degraded mode.
- **Always show diffs** — even for "replace" strategy files, show the diff before replacing.
- **Git is the safety net** — remind the captain they can `git diff` or `git checkout` to recover.
