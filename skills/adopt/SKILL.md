---
name: adopt
description: "Use when a project has no workflow yet and an installed workflow plugin provides a template. Discovers workflow-template.yaml from installed plugins, presents options, invokes spacedock:commission in batch mode. Triggers on '/adopt', 'adopt workflow', 'install workflow', or when /build fails because no workflow exists."
user-invocable: true
argument-hint: "[template-name] [--dir path/to/workflow]"
---

# Adopt — Install a Workflow from Plugin Template

Adopt a workflow template from an installed plugin into the current project. This is the one-time bootstrap that creates the workflow directory and README via `spacedock:commission`.

## Args

| Arg | Required | Description |
|-----|----------|-------------|
| `template-name` | no | Name field from a `workflow-template.yaml`. Skips discovery if provided. |
| `--dir path` | no | Override the template's `location_default` for where to create the workflow. |

No args = interactive discovery (scan → choose → confirm).

## Step 1: Check for existing workflow

```bash
find "$(git rev-parse --show-toplevel)" -maxdepth 3 -name "README.md" \
  -exec grep -l "commissioned-by: spacedock@" {} \; 2>/dev/null
```

**If found → STOP:**

> Workflow already exists at `{workflow_dir}/`.
> Use `/spacedock:sync` to check for template updates, or the workflow's skills to start working.

## Step 2: Discover templates

If `template-name` arg was provided, search for a matching template and skip the selection prompt.

Otherwise, scan for all `workflow-template.yaml` files:

```bash
find ~/.claude/plugins/cache -name "workflow-template.yaml" 2>/dev/null
```

Read each template, extract `name`, `description`, `plugin` fields.

**Zero templates → STOP:**

> No workflow templates found. Install a workflow plugin first, or use `/spacedock:commission` to design a custom workflow.

**One template:**

> Found: **{name}** — {description}
> Plugin: {plugin}
> Stages: {stage names, comma-separated}
> Location: {location_default} {or --dir override}
>
> Adopt this workflow? (yes / no / --dir to change location)

**Multiple templates:**

> Available workflow templates:
>
> {N}. **{name}** ({plugin}) — {description}
>
> Which one? (number)

## Step 3: Commission in batch mode

Read the selected `workflow-template.yaml` fully. Build a batch payload mapping:

| Template field | Commission input |
|---|---|
| `description` | Mission |
| `entity.type`, `entity.label`, `entity.description` | Entity type + label |
| `stages[*]` | Full stage list with all properties |
| `gates` | Approval gates |
| `--dir` or `location_default` | Workflow directory |

Invoke:

```
Skill("spacedock:commission", args: "<batch payload>\n\nSkip interactive confirmation — generate files directly.")
```

**Critical:** Every stage with a `skill:` field in the template MUST get that field written into the README stage definition. This is how first-officer resolves plugin skills at dispatch time.

## Step 4: Verify + report

1. Confirm `{dir}/README.md` exists with `commissioned-by: spacedock@`
2. Confirm stage `skill:` fields match the template
3. Report:

> Workflow **{name}** adopted at `{dir}/`.
>
> Next steps:
> - Use the workflow plugin's skills to start working
> - `claude --agent spacedock:first-officer` to orchestrate
> - `/spacedock:sync` later to check for template updates
