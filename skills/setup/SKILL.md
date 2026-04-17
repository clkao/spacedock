---
name: setup
description: "Set up a workflow in the current project. Discovers installed workflow plugins with templates, lets captain choose one, then invokes spacedock:commission in batch mode. Triggers on '/setup', 'setup workflow', 'install workflow'."
user-invocable: true
---

# Setup — Workflow Bootstrap

You are setting up a workflow in the current project. This skill discovers installed workflow plugins that ship a `workflow-template.yaml`, lets {captain} choose one, then invokes `spacedock:commission` in batch mode with the template's pre-configured stages.

## Step 1: Detect existing workflow

Search for an existing commissioned workflow:

```bash
find "$(git rev-parse --show-toplevel)" -maxdepth 3 -name "README.md" -exec grep -l "commissioned-by: spacedock@" {} \; 2>/dev/null
```

**If found:**

> Workflow already exists at `{workflow_dir}/`.
>
> Run `python3 {spacedock_plugin_dir}/skills/commission/bin/status --workflow-dir {workflow_dir} --next` to see dispatchable entities, or use the workflow's skills to start working.

**STOP. Do not invoke commission.**

## Step 2: Discover workflow templates

Scan installed plugins for `workflow-template.yaml` files. Use the Glob tool:

```
Glob: "**/workflow-template.yaml"
```

Also check the CC plugin cache directories:

```bash
find ~/.claude/plugins/cache -name "workflow-template.yaml" 2>/dev/null
```

For each template found, read it and extract `name` and `description` fields.

**If no templates found:**

> No workflow plugins with templates found. Install a workflow plugin first, or use `/spacedock:commission` to design a workflow from scratch.

**STOP.**

**If one template found:**

> Found workflow template: **{name}** — {description}
>
> From plugin: {plugin}
>
> Stages: {list stage names from template}
>
> Set up this workflow? (yes / no / customize location)

**If multiple templates found:**

> Found {N} workflow templates:
>
> {for each: "{N}. **{name}** ({plugin}) — {description}"}
>
> Which workflow would you like to set up? (number / customize)

Wait for captain's choice.

## Step 3: Invoke commission in batch mode

Read the selected `workflow-template.yaml` fully. Build a batch-mode payload for `spacedock:commission` from the template fields.

Map template fields to commission inputs:

| Template field | Commission input |
|---|---|
| `description` | Mission |
| `entity.type` | Entity type |
| `entity.description` | Entity description |
| `stages[*]` | Stages (with all properties: gate, feedback-to, skill, model, manual, dispatch) |
| `gates` | Approval gates |
| `location_default` | Location (unless captain specified custom) |

Call commission via the Skill tool:

```
Skill("spacedock:commission", args: "<assembled batch payload>")
```

The batch payload must include the instruction: **"Skip interactive confirmation — generate files directly."**

**Critical:** Ensure every stage with a `skill:` field in the template gets that `skill:` field written into the generated workflow README's stage definition. This is how the first-officer knows which plugin skill to dispatch for each stage.

## Step 4: Post-commission verification

After commission completes:

1. Confirm `{dir}/README.md` exists and contains `commissioned-by: spacedock@`
2. For each stage with a `skill:` field in the template, confirm the README's stage definition includes the matching `skill:` value
3. Show the captain:

> Workflow **{name}** created at `{dir}/`.
>
> Available commands from the **{plugin}** plugin:
> {List user-invocable skills from the workflow plugin, if discoverable}
>
> Or run `claude --agent spacedock:first-officer` to orchestrate the pipeline.
