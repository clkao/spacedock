---
name: sync
description: "Use when a project's workflow README may be outdated relative to its source plugin template. Compares project README stages against the plugin's workflow-template.yaml, reports drift, and applies captain-approved updates. Triggers on '/sync', 'sync workflow', 'update workflow', or after a plugin upgrade."
user-invocable: true
argument-hint: "[workflow-dir]"
---

# Sync — Update Workflow from Plugin Template

Compare this project's workflow README against the plugin template it was adopted from. Report drift and apply captain-approved changes.

Unlike `refit` (which handles spacedock engine version upgrades), `sync` handles **workflow template evolution** — new stages, changed skill bindings, updated properties.

## Args

| Arg | Required | Description |
|-----|----------|-------------|
| `workflow-dir` | no | Path to the workflow directory. Auto-detected if omitted. |

## Step 1: Find the workflow

If `workflow-dir` not provided, search for it:

```bash
find "$(git rev-parse --show-toplevel)" -maxdepth 3 -name "README.md" \
  -not -path "*/node_modules/*" -not -path "*/.git/*" -not -path "*/vendor/*" \
  -exec grep -l "commissioned-by: spacedock@" {} \; 2>/dev/null
```

**Not found → STOP:**

> No workflow found. Use `/spacedock:adopt` to install one first.

**Multiple found:** list them, ask captain to pick.

## Step 2: Identify source template

Read `{workflow_dir}/README.md` frontmatter. Extract:
- `commissioned-by:` — engine version at commission time
- Stage list with `skill:` fields — identifies the source plugin

From the `skill:` namespace (e.g., `spacedock-workflow:build-explore`), derive the plugin name (`spacedock-workflow`).

Find the matching `workflow-template.yaml`:

```bash
find ~/.claude/plugins/cache -path "*/{plugin_name}/*/workflow-template.yaml" 2>/dev/null
```

**Template not found → STOP:**

> Plugin **{plugin_name}** is not installed or has no template. Install it first.

## Step 3: Diff stages

Compare README stages vs template stages. Detect:

| Change type | Detection |
|---|---|
| **New stage** | In template but not in README |
| **Removed stage** | In README but not in template |
| **Skill binding changed** | Same stage name, different `skill:` value |
| **Property changed** | Same stage, different `model:`, `gate:`, `feedback-to:`, `dispatch:` |
| **Stage reordered** | Same stages, different sequence |
| **Description updated** | Same stage, different description text |

Also diff:
- `entity.type`, `entity.label` changes
- `defaults` (worktree, concurrency) changes
- `gates` list changes

## Step 4: Report drift

**No drift:**

> Workflow at `{workflow_dir}/` is up to date with **{template_name}** template.

**STOP.**

**Drift found — present changes one category at a time:**

> Workflow drift detected — **{N} changes** between project README and **{template_name}** template:
>
> **New stages:**
> {list with description}
>
> **Skill binding changes:**
> | Stage | Current | Template |
> |-------|---------|----------|
> | ... | ... | ... |
>
> **Property changes:**
> | Stage | Property | Current | Template |
> |-------|----------|---------|----------|
> | ... | ... | ... | ... |
>
> **Removed stages (in README but not in template):**
> {list — these are project customizations, NOT auto-removed}
>
> Apply these changes? (all / pick / skip)

## Step 5: Apply changes

Based on captain's choice:

- **all** — apply every change from template
- **pick** — let captain accept/reject each change individually
- **skip** — do nothing

For each accepted change, edit the README's YAML frontmatter and stage sections.

**Rules:**
- **New stages:** Insert at the position defined in the template. Generate stage section content from template description.
- **Skill binding changes:** Update `skill:` field in frontmatter.
- **Property changes:** Update the property in frontmatter.
- **Removed stages:** NEVER auto-remove. These may be project customizations. Only remove if captain explicitly confirms.
- **Stage descriptions:** Update the prose section under `### \`{stage_name}\`` if captain accepts.

After applying, commit:

```bash
git add {workflow_dir}/README.md
git commit -m "sync: update workflow from {template_name} template"
```

## Step 6: Verify

Re-read the updated README and confirm all accepted changes are present. Report:

> Workflow synced. Applied {N} of {M} changes.
> {list applied changes}
> {list skipped changes, if any}
