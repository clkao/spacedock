---
id: rdtbc72j3wszp1skfjxvp3m5
title: "Dispatch prompt as file-pointer: shrink FO per-dispatch context from ~14000 to ~380 chars"
status: done
source: "Captain question in 2026-05-21 session about Agent-spawn-phase expansion. Research doc at docs/research/2026-05-21-agent-spawn-prompt-expansion.md walks the three spawn-phase expansion mechanisms (skills frontmatter, @-include in skill body, Skill args) and empirically confirms @-include does NOT resolve in Agent() prompt args. The proposed mechanism for FO context compression — `DISPATCH_FILE: {path}` convention with ensign-side Read on first action — composes with PR #231's existing fetch-on-demand pattern at one layer up. Predicted ~97% FO per-dispatch context reduction."
started: 2026-05-22T18:08:01Z
completed: 2026-05-23T04:04:54Z
verdict: PASSED
score:
worktree: 
issue:
pr: #234
mod-block: 
archived: 2026-05-23T04:05:16Z
---

## Problem

After PR #231 (0x9, fetch-on-demand dispatch spec), the FO's per-dispatch context cost is still **~6,000 chars** even though the ensign-side prompt was shrunk significantly (F2 from staff review: measured baseline is ~6,000 chars per dispatch, not the originally-claimed ~14,000):

- ~3,000 chars from `claude-team build`'s stdout JSON (the FO reads this to dispatch)
- ~3,000 chars from `Agent(prompt=...)` (the FO emits this verbatim to spawn the ensign)

The prompt enters the FO's context twice per dispatch. PR #231 only fixed the ensign side (`### Fetch commands` block makes the ensign load stage def + standing teammates JIT). The FO still has to carry the full prompt through its own reasoning surface.

A captain-session of 20 dispatches accumulates ~120,000 chars of FO-side prompt-passing. That's a significant fraction of the FO's context budget that the FO never reasons about — it's purely pass-through cost.

## Why this matters

- **Long-session FO context budget.** The FO's context budget compounds across all dispatches. Reducing per-dispatch cost is the most direct win for session-length viability.
- **Codex parity.** The Codex runtime adapter will face the same issue if it follows the helper output → forward-verbatim shape. Solving this at the helper-protocol layer benefits both adapters.
- **Composes cleanly with 0x9.** PR #231's `### Fetch commands` block is the *ensign-side* fetch-on-demand pattern. This entity applies the *same pattern* at the FO-side: the FO emits a tiny pointer, the ensign Reads the per-dispatch content from a file on first action. Different beneficiaries, same architectural idea.

## Proposed approach

The FO's `Agent(prompt=...)` arg becomes a tiny `DISPATCH_FILE:` pointer (~80 chars). The helper writes the per-dispatch content to a stable path. The ensign Reads the file on first action and treats its content as the assignment.

### Address the four design dimensions

#### 1. What the FO actually assembles (the irreducible per-dispatch context)

The FO MUST author and pass through these per-dispatch elements; they cannot move out of the FO's context:

| Element | Source | Approx. size | Why FO-authored |
|---|---|---|---|
| `checklist` | FO judgment per stage | 800-1500 chars | FO decides what this dispatch must accomplish |
| `scope_notes` | FO judgment per dispatch | 0-2000 chars | Conditional context (corrections, prior-cycle issues, captain feedback to relay) |
| `feedback_context` | Reviewer findings (when re-dispatching) | 0-3000 chars | Per-dispatch input from validation/staff-review |
| `bare_mode` | Live team state | 1 byte | FO observes runtime state |
| `is_feedback_reflow` | Stage transition logic | 1 byte | FO knows when routing rejection back |
| Entity identity | Entity slug + stage + worktree path | ~200 chars | FO decides which entity advances when |
| Team context | `team_name` (when applicable) | ~60 chars | FO's team-create or active-team awareness |

**Estimate: ~1500-7000 chars of genuinely FO-authored per-dispatch input.** This is what the FO needs in its own context to assemble. It already does today via the input JSON it pipes to `claude-team build`. Unchanged by this entity.

What is NOT FO-authored and currently still passes through FO context: stage definition prose, standing-teammates routing instructions, the boot directive, the completion-signal prose, the FO-forwarding warning. **These ~5000+ chars are the target — they should not appear in the FO's prompt context.**

#### 2. How (1) flows into the `claude-team build` helper

Unchanged from today's contract. The FO assembles its inputs as a JSON object and pipes to the helper:

```json
{
  "schema_version": 2,
  "entity_path": "/abs/path/to/entity.md",
  "workflow_dir": "/abs/path/to/workflow",
  "stage": "implementation",
  "checklist": ["1. ...", "2. ..."],
  "scope_notes": "...",
  "feedback_context": null,
  "team_name": "...",
  "bare_mode": false,
  "is_feedback_reflow": false
}
```

The interface is the same JSON schema. **Schema version bumps to 2** to signal the new helper output contract. The helper accepts both v1 (today's inline-prompt shape) and v2 (file-pointer shape) input during a transition window, OR the entity treats this as a breaking change and bumps in one cut. Decision tracked in design below.

#### 3. The mechanism turning (2) into the Agent() tool call

The helper's behavior changes to produce a file-pointer output:

**Helper internal flow:**
1. Read the input JSON from stdin.
2. Assemble the full dispatch prompt (same content as today's PR #231 output) into a Python string.
3. Compute a stable path: `/tmp/spacedock-dispatch/{name}.md` (where `{name}` is the deterministic agent name the helper already computes).
4. Create `/tmp/spacedock-dispatch/` if missing (parent dir).
5. Write the full prompt content to the path.
6. Emit a NEW stdout JSON:

```json
{
  "schema_version": 2,
  "subagent_type": "spacedock:ensign",
  "name": "spacedock-ensign-{slug}-{stage}",
  "team_name": "...",
  "model": null,
  "description": "{entity title}: {stage}",
  "dispatch_file_path": "/tmp/spacedock-dispatch/spacedock-ensign-{slug}-{stage}.md",
  "prompt": "DISPATCH_FILE: /tmp/spacedock-dispatch/spacedock-ensign-{slug}-{stage}.md"
}
```

The `prompt` field is now ~80 chars. The `dispatch_file_path` field is the same path, exposed for debugging / tooling (the FO doesn't need to read it; it only needs `prompt`).

**FO consumption flow:**

Unchanged in shape from today:

```python
spec = json.loads(helper_stdout)
Agent(
    subagent_type=spec["subagent_type"],
    name=spec["name"],
    team_name=spec["team_name"],
    model=spec["model"],
    prompt=spec["prompt"]  # ~80 chars now, was ~7000
)
```

The FO doesn't have to know about the file. It just forwards the helper's `prompt` verbatim as before. The helper has done the work of writing the file and emitting a tiny pointer.

**Failure modes for step 3:**

- **Helper can't write to `/tmp/spacedock-dispatch/`** (permission, disk full): helper exits non-zero with a clear error. FO falls back to break-glass per existing contract.
- **Two concurrent dispatches collide on the same `{name}.md`**: shouldn't happen because the name is deterministic per (entity, stage, cycle). If it does, the second write overwrites the first, both ensigns Read the same content — same prompt for both. Idempotent.
- **File deleted between helper-write and ensign-Read**: rare (no cleanup process between dispatch and first-action). Ensign Read fails, surfaces to FO via `SendMessage` with the missing-file error.

#### 4. What the spawned ensign's initial prompt is, and how reliable

The ensign sees:

```
DISPATCH_FILE: /tmp/spacedock-dispatch/spacedock-ensign-{slug}-{stage}.md
```

That's the entire `Agent(prompt=...)` arg. ~80 chars.

The ensign's behavior depends on its agent-definition + skill body. Today's `agents/ensign.md` says:

```
You are an ensign executing stage work for a workflow.

## Boot Sequence

If your operating contract was not already loaded via skill preloading, invoke the `spacedock:ensign` skill now to load it.

Then read your assignment and begin work.
```

This entity adds a clause to `skills/ensign/references/ensign-shared-core.md` `## First action` section:

> If your initial prompt matches the pattern `DISPATCH_FILE: {path}` (one line, no other content), your first action is `Read({path})` and you treat the file's content as if it had been your inline prompt. Then proceed with the operating contract.
>
> If the file does not exist or Read fails, send `SendMessage(to="team-lead", message="DISPATCH_FILE_MISSING: {path} - {error}")` and stop. Do NOT proceed with empty context.

**Reliability question:** does the model reliably interpret a 1-line `DISPATCH_FILE:` prompt as "Read this file, then proceed"?

The empirical risk surface is twofold:

(a) **Skill load happens before the prompt is read**, per the spawn-phase Mechanism 1 / 2 (verified in `_archive/agent-boot-skill-preload.md` #085 and the production `@`-include resolution in `skills/ensign/SKILL.md:8-9`). So when the ensign sees the `DISPATCH_FILE:` prompt, the ensign-shared-core (including the new clause above) is already in its context. The model has the discipline-prose before it has the prompt.

(b) **However, the skill-preload path is broken for plugin agents per `claude-code#25834`.** The body-fallback `Skill(skill="spacedock:ensign")` invocation is the workaround. But the body-fallback would be in the *file content* (because the helper still emits the boot directive into the file at /tmp/spacedock-dispatch/). The 80-char Agent() prompt arg has no boot directive. So the question becomes: will the model invoke the skill before Read'ing the file?

**Design fork:** the 80-char prompt could include the boot directive inline:

```
Skill(skill="spacedock:ensign"); then Read /tmp/spacedock-dispatch/{name}.md and treat its content as your assignment.
```

That's ~150 chars instead of 80. Still ~98% smaller than today's ~7000. Adds the boot-fallback explicitly. **This is the design's recommended shape** — slightly larger prompt arg, but it incorporates the skill-load-failure failsafe that 0x9 introduced.

**Empirical risk: ensign freelance behavior.** Sonnet-bare on `test_gate_guardrail` works reliably today (verified by `2y` spike + CI). But the 0x9 cycle-4 spike showed that haiku-bare freelances on minimal prompts. If we're targeting sonnet-or-above for bare-mode FO (per `2y`'s declaration), the same model floor probably applies to ensign reliability with minimal prompts. **The spike for this entity must verify sonnet (the supported floor) reliably interprets the `Skill(...); Read X; ...` shape.**

### Spike that gates the design

Before locking any helper changes, run a 5-run spike that empirically verifies the ensign reliably interprets the minimal `DISPATCH_FILE` prompt shape.

**Spike harness:**

1. Build a tiny test fixture: single entity with checklist "Create `/tmp/dispatch-file-pointer-spike-output.txt` containing `OK`."
2. For each of the 5 runs:
   - Write the full dispatch content (today's PR-#231 fetch-on-demand shape) to `/tmp/spacedock-dispatch/spike-N.md`
   - Call `Agent(subagent_type="spacedock:ensign", prompt='Skill(skill="spacedock:ensign"); then Read /tmp/spacedock-dispatch/spike-N.md and treat its content as your assignment.')`
   - Capture fo-log.jsonl + check the output file exists with the expected content
3. **Verdict:** PASS if ≥4/5 runs successfully Read the file AND complete the checklist work. FAIL if any run freelances (does not Read), errors out without clean SendMessage, or fails to complete the trivial task.
4. **Model:** sonnet (the supported floor per `2y`). The spike is NOT required to work on haiku per the supported-models declaration.

Cost: ~$3-5, ~10 min wall-clock.

If PASS → land helper change + ensign-shared-core clause + tests.
If FAIL → record the failure mode, consider whether a slightly-larger prompt (e.g., ~200 chars with explicit ordering + error-handling instruction) closes the gap. If even that fails on sonnet, the proposal is structurally unworkable; ship a no-code-change Stage Report recommending re-evaluation.

## Empirical findings

Spike executed 2026-05-22 at `/tmp/rdt-spike/` with sonnet (the supported floor). Five sequential runs invoked `claude -p --agent spacedock:ensign --model sonnet --plugin-dir /Users/clkao/git/spacedock` with the recommended ~150-char prompt shape; each run's per-dispatch content was written to `/tmp/spacedock-dispatch/spike-N.md` (3,632 bytes — representative of a real PR #231 dispatch body, includes operating-contract directive, header, entity-read instruction, scope notes, checklist, fetch-on-demand block, completion-signal block).

The tiny `Agent(prompt=...)` arg under test was 119 characters:

```
Skill(skill="spacedock:ensign"); then Read /tmp/spacedock-dispatch/spike-N.md and treat its content as your assignment.
```

| Run | rc | Read DISPATCH_FILE | Output file `OK` | Done: signal emitted | Verdict |
|---|---|---|---|---|---|
| 1 | 0 | yes | yes | yes | PASS |
| 2 | 0 | yes | yes | yes | PASS |
| 3 | 0 | yes | yes | yes | PASS |
| 4 | 0 | yes | yes | yes | PASS |
| 5 | 0 | yes | yes | yes | PASS |

**Verdict: PASS (5/5, exceeds ≥4/5 captain threshold).**

Observed tool sequence (consistent across all runs): `Skill(spacedock:ensign)` → `Read(ensign-shared-core.md)` → `Read(claude-ensign-runtime.md)` → `Read(/tmp/spacedock-dispatch/spike-N.md)` → fetch-stub `Bash(echo ...)` → `Write(/tmp/dispatch-file-pointer-spike-output.txt)` → `Bash(cat verify)` → text "Done: ...". The ensign treated the file's content as its assignment exactly as the design intends.

Key reliability evidence:

- **The inline `Skill(...)` failsafe works.** The model parsed `Skill(skill="spacedock:ensign")` out of the 119-char prompt and invoked the tool before Reading the file. This means the design does not depend on the broken skill-frontmatter-preload path (`claude-code#25834`); the boot-fallback rides in the prompt arg itself. Risk C (Option D fallback — bake DISPATCH_FILE clause into `agents/ensign.md`) is **not triggered**.
- **No freelance behavior observed on sonnet.** Zero runs exhibited the haiku-bare freelance pattern that gated the 0x9 cycle-4 spike. The 2y declared sonnet-or-above model floor holds for ensign reliability as well.
- **The trailing imperative ("treat its content as your assignment") is load-bearing.** Without it, the model could plausibly Read-and-then-stop, treating the prompt as an information-gathering request. The phrasing converts the Read result into operative instructions.

Risk B residue: each run wrote a fresh `/tmp/spacedock-dispatch/spike-N.md`; idempotent overwrites confirmed (no collision concerns at 5x). No cleanup performed between runs; files persist until tmpfs reboot, consistent with the design recommendation.

## Design

The spike validates the **proposed approach** as written. This section locks the four design dimensions and the helper contract.

### 1. FO-authored per-dispatch elements

Unchanged from `## Proposed approach` table. The FO must continue assembling these in its own context:

| Element | Source | FO-authored? |
|---|---|---|
| `entity_path`, `workflow_dir`, `stage` | FO routing | yes |
| `checklist` (list of imperative lines) | FO judgment per stage | yes |
| `scope_notes` (conditional context) | FO judgment | yes |
| `feedback_context` (conditional) | Prior reviewer findings | yes |
| `bare_mode`, `is_feedback_reflow` | Live team state | yes |
| `team_name` (teams mode) | TeamCreate output | yes |
| Stage definition prose | Workflow `README.md` | **no — moved to file** |
| Standing teammates routing | Workflow standing config | **no — moved to file** (already moved by PR #231 fetch-on-demand) |
| Operating-contract directive | Boilerplate | **no — moved to file** |
| Completion-signal prose | Boilerplate | **no — moved to file** |
| FO-forwarding warning | Boilerplate | **no — moved to file** |

The FO continues to assemble the JSON input it already builds today. **No change to the FO's input shape.** All the size win is on the output side (the helper writes a file and emits a tiny pointer).

### 2. JSON input schema — bump to v2

Input schema bumps to `schema_version: 2` to signal opt-in to the new output contract. Schema fields are unchanged from today (entity_path, workflow_dir, stage, checklist, scope_notes, feedback_context, team_name, bare_mode, is_feedback_reflow). Rationale for the bump:

- Output shape changes meaningfully: `prompt` shrinks ~98%, a new `dispatch_file_path` field appears.
- Downstream tooling that consumes helper output (tests, future Codex adapter port) needs an explicit signal so it can route on shape.
- Back-compat path: helper accepts v1 input and emits today's v1-shape output unchanged. v2 input triggers the new file-pointer behavior. Cutover is opt-in per dispatch via the input version; no Big Bang.

### 3. Helper flow → file write + tiny stdout JSON → FO Agent() forwarding contract

**Helper internal flow (v2 input):**

1. Parse stdin JSON; validate `schema_version == 2`.
2. Run all existing validation (entity exists, workflow README parses, stage resolves, model enum, name length, etc.) — identical to today.
3. Assemble the **full dispatch prompt** (the same string today's helper would emit, identical content) into a Python string. Reuse the existing assembly code; no behavioral change to the content.
4. Compute `dispatch_file_path = f"/tmp/spacedock-dispatch/{derived_name}.md"`. `derived_name` is the deterministic name the helper already computes (`{worker_key}-{slug}-{stage}`).
5. `os.makedirs("/tmp/spacedock-dispatch", exist_ok=True)`. Tmpfs-friendly; no cleanup process needed.
6. Write the assembled prompt to `dispatch_file_path` (UTF-8, overwrite-on-collision — idempotent for same-name re-dispatches).
7. Build the tiny `prompt` field (~119 chars for this entity's path lengths):

```
Skill(skill="spacedock:ensign"); then Read {dispatch_file_path} and treat its content as your assignment.
```

8. Emit stdout JSON:

```json
{
  "schema_version": 2,
  "subagent_type": "spacedock:ensign",
  "description": "{entity title}: {stage}",
  "name": "{derived_name}",
  "team_name": "{team_name or omitted}",
  "model": "{effective_model}",
  "fetch_commands": ["..."],
  "dispatch_file_path": "/tmp/spacedock-dispatch/{derived_name}.md",
  "prompt": "Skill(skill=\"spacedock:ensign\"); then Read /tmp/spacedock-dispatch/{derived_name}.md and treat its content as your assignment."
}
```

Note `fetch_commands` is still mirrored at the top level for tooling parity with v1 (it also appears inside the file body). Helpers that today read `fetch_commands` directly out of the helper stdout (e.g., debug tooling) continue to work without changes.

**FO consumption flow (unchanged in shape):**

```python
spec = json.loads(helper_stdout)
Agent(
    subagent_type=spec["subagent_type"],
    name=spec["name"],
    team_name=spec.get("team_name"),
    model=spec["model"],
    prompt=spec["prompt"],  # 119 chars vs ~7000 in v1
)
```

The FO does not have to know about the file. It forwards `spec["prompt"]` verbatim. The helper's output JSON is also ~98% smaller because `prompt` is the dominant size term — the FO reads ~250 chars of helper stdout instead of ~7000.

**Failure modes:**

- **Helper can't write `/tmp/spacedock-dispatch/`** (permission, disk full): exit non-zero with stderr `dispatch_file_write_failed: {path}: {errno}`. FO falls back to break-glass per existing helper-failure contract. Tested by AC-2.
- **Concurrent same-name dispatches**: name is deterministic per (entity, stage); same content, idempotent overwrite. Tested implicitly by AC-1 (file content matches).
- **File deleted between helper-write and ensign-Read**: rare in practice (no cleanup process touches `/tmp/spacedock-dispatch/` between dispatch and first action). Ensign Read fails, shared-core clause routes to `SendMessage(to="team-lead", message="DISPATCH_FILE_MISSING: {path} - {error}")`. Tested by AC-3.

### 4. Ensign initial prompt shape + reliability discussion

**Locked shape (per spike PASS):**

```
Skill(skill="spacedock:ensign"); then Read {dispatch_file_path} and treat its content as your assignment.
```

Three required components, in this order:

1. `Skill(skill="spacedock:ensign")` — explicit boot-fallback. Defends against the broken skill-frontmatter-preload path (`claude-code#25834`). The spike confirmed the model invokes this even when its own preload would have succeeded — calling twice is idempotent per the skill's own discipline.
2. `Read {dispatch_file_path}` — instructs the model to load the file body as context. Path is absolute, predictable, and deterministic per dispatch.
3. `and treat its content as your assignment` — converts the Read result into operative instructions. Without this trailing clause, a model could plausibly Read-and-then-stop (treating the prompt as a query rather than a dispatch).

**Length:** ~119 chars for typical `derived_name` lengths; up to ~200 chars for the longest possible name (defends AC-1's ≤200 char ceiling).

**Reliability ceiling:** the spike establishes 5/5 PASS on sonnet under realistic dispatch-body conditions. This is the supported model floor. No claim is made for haiku.

**Why the shared-core clause is still needed (AC-3):** the prompt-arg failsafe handles the boot directive, but it does not handle:

- The error case (file missing or unreadable) — needs `DISPATCH_FILE_MISSING:` SendMessage prose.
- The model-discipline case (what does "treat its content as your assignment" actually mean operationally) — needs the existing `Fetch-on-Demand Bootstrap` discipline already in shared-core to apply uniformly. The new clause makes this explicit so the discipline is documented, not just empirical.

The shared-core clause is therefore additive, not redundant — it covers the failure mode the prompt arg cannot cover.

## Acceptance criteria

End-state properties of the finished entity:

1. **`claude-team build` (with `schema_version: 2` input) emits a spec where `prompt` is ≤200 chars** and contains the substrings `Skill(skill="spacedock:ensign")`, `Read`, `/tmp/spacedock-dispatch/` (the predictable path pattern), and `treat its content as your assignment` (the load-bearing trailing imperative confirmed by the spike). The full per-dispatch content (today's PR #231 shape) is written to a file at `/tmp/spacedock-dispatch/{name}.md`.
   - **Test:** parser-level test in `tests/test_claude_team.py` runs the helper against a canonical v2 input and asserts (a) the emitted `prompt` length is ≤200, (b) the four substrings appear, (c) `dispatch_file_path` field is set and the file exists at that path with the expected content.

2. **CLEAN-BREAK: helper rejects v1 input with `schema_version: 2 required`.** Captain decision 2026-05-22 (F4 from staff review): v1 input is rejected, not preserved. Skill + claude-team ship together in this entity; a larger refactor is planned, so the v1 back-compat surface buys nothing.
   - **Test:** parser-level test pipes a v1 input to the helper and asserts exit code 2 with stderr containing `schema_version: 2 required` (`tests/test_claude_team.py::TestBuildSchemaVersion::test_build_schema_version_v1_rejection`).

3. **`skills/ensign/references/ensign-shared-core.md` `## First action` section contains the `DISPATCH_FILE:` clause** with the two properties: (a) instructs Read on a matching prompt pattern; (b) instructs SendMessage-on-failure with `DISPATCH_FILE_MISSING:` prefix.
   - **Test:** static-content test asserts both substrings appear in the file.

4. **Spike result is documented** in the entity body's `## Empirical findings` section with verdict PASS/INCONCLUSIVE/FAIL and per-run table. Captain-set threshold: ≥4/5 PASS. The ideation-cycle spike landed 5/5 PASS on sonnet; this AC is satisfied at the ideation gate and a future cycle does not re-run it.
   - **Test:** entity body content check — section exists, contains verdict, contains per-run table.

5. **v2 helper `prompt` field is ≥85% smaller than the v1-equivalent assembly** on a canonical fixture (F1 from staff review: reframed in percentage terms because the real PR-#231-post v1 prompt is ~2,500-3,200 chars, not the originally-assumed ~7,000; an absolute ≥10,000-char floor is structurally unsupportable. The v1-equivalent is the v2 dispatch-file body — by design, the body equals what v1 would have emitted as `prompt` because the assembly logic is unchanged; only the delivery shape moved).
   - **Test:** parser-level test runs the helper against a canonical v2 input and asserts `1 - len(prompt) / len(dispatch_body) >= 0.85`. Measured on a representative fixture: v1-equivalent = 2,577 chars; v2 prompt = 175 chars; reduction = 93.2% (recorded in implementation Stage Report).

6. **`make test-static` passes with the changes applied** — no regressions from the new behavior.
   - **Test:** `make test-static` exit code 0.

7. **`make test-live-claude` from worktree** validates a real dispatch via the v2 path completes end-to-end without behavioral regression vs. PR #231 baseline. This is the captain's cross-cycle live-test requirement.
   - **Test:** `tests/test_dispatch_completion_signal.py::test_dispatch_completion_signal` (F5 from staff review: verified present, `@pytest.mark.live_claude`, exercises a fresh `Agent(subagent_type="spacedock:ensign")` dispatch via `_first_agent_dispatch_prompt`; not a SendMessage reuse-path test). Pass requires exit code 0 and the v2 file-pointer prompt observed in fo-log.jsonl.

## Test plan

- **Spike** (cost-controlled, gates design): 5 sonnet runs of the trivial `DISPATCH_FILE:` fixture. ~$3-5, ~10 min. Artifacts at `/tmp/rdt-spike/`.
- **`tests/test_claude_team.py`** (extend): 4 new tests covering AC-1, AC-2, AC-5 (helper output shape + size + back-compat + savings floor).
- **`tests/test_agent_content.py`** (extend): 1 new test for AC-3 (ensign-shared-core `DISPATCH_FILE:` clause).
- **Golden file** at `tests/fixtures/dispatch_prompt_v2.txt` for the v2 helper output, checked in with the implementation commit.
- **No new modules.** ~50 lines of Python in `claude-team build` (file-write logic + schema_version handling), ~10 lines of ensign-shared-core prose, ~50 lines of test additions.

## Out of scope

- **Reuse-path dispatch (SendMessage to a kept-alive ensign).** Today's reuse-advance flow doesn't go through `claude-team build`; it's a direct SendMessage with a few hundred chars. Adding `DISPATCH_FILE:` semantics to reuse would compress reuse-advance content too, but the per-dispatch reuse cost is already small (~1500 chars vs ~14000 for a fresh dispatch). The compression benefit/complexity ratio favors deferring reuse-path changes to a future entity. **In scope here: initial Agent() dispatch only.**
- **Breakglass dispatch.** The break-glass manual-assembly path (used when `claude-team build` itself fails) is reserved for catastrophic failure modes; it stays a fully-inlined string per the current FO runtime adapter. Out of scope.
- **Codex runtime adapter.** The proposal is platform-neutral in concept but this entity only ships the Claude path. Codex parity is a follow-up.
- **Workflow-discovery file location.** `/tmp/spacedock-dispatch/` is chosen for simplicity (tmpfs, no cleanup needed across sessions). An alternative under workflow_dir is possible but adds discovery complexity for the ensign. Out of scope unless the spike surfaces a reason.

## Risks

### Risk A — Schema-version bump breaks downstream consumers

Any tool that reads `claude-team build` output today expects v1 shape. The schema_version=2 output has a different prompt size (much smaller) and a new `dispatch_file_path` field. Mitigation: the helper continues to accept v1 input (back-compat AC-2); the output format change is documented; downstream tests (e.g., `tests/test_claude_team_spawn_standing.py`) get updated alongside the helper change.

### Risk B — File-write-during-dispatch race or cleanup gap

The `/tmp/spacedock-dispatch/` directory accumulates files over a long captain session. Each file is ~7000 chars; 100 dispatches = ~700KB. Not a real disk concern, but worth deciding: does the FO clean up the file after Done: signal? Or does the helper overwrite on every dispatch (idempotent name-based addressing)?

**Recommended:** helper overwrites; no cleanup. Same-named dispatches (re-runs of cycle-N implementation) get a fresh file. Stale files from prior sessions are tmpfs-cleaned on reboot. Implementation can verify with a static test that the file path is deterministic per (entity, stage).

### Risk C — Spike-FAIL fallback

If the empirical spike shows the ensign doesn't reliably interpret the `DISPATCH_FILE:` shape, the fallback is option D below (instead of the proposed primary):

**Option D — bake DISPATCH_FILE handling into the agent definition itself.** Edit `agents/ensign.md` to include the DISPATCH_FILE clause directly in the agent's body (which is in spawn-phase context per Mechanism 1). This makes the discipline part of the agent's identity rather than something it has to read from the skill body. Slightly less idiomatic but more reliable.

If spike on the proposed shape FAILS, run a follow-up spike with Option D before recommending REJECTED.

## Scale context

- Spacedock version: 0.11.2+ (post PR #231 + #233)
- Builds on: PR #231 (0x9, fetch-on-demand dispatch spec — established the ensign-side fetch-on-demand pattern this entity extends to FO-side)
- Composes with: `mtc76hh8` (runtime-boundary split — would split `claude-team build` into runtime-neutral core + Claude adapter; the file-write logic in this entity lands in the runtime-neutral core), `y15kpj10` (FO retry-path bypass — orthogonal but related to FO dispatch discipline)
- Measured impact: ~93% reduction in the helper's `prompt` field (v1-equivalent 2,577 → v2 175 chars on a canonical fixture) and ~80% reduction in FO per-dispatch context (~5,154 → 1,032 chars when summing helper stdout + Agent prompt arg). For a 20-dispatch captain session, ~82,000 chars of FO budget freed (F2 from staff review corrected the original ~97%/~260,000 claim against today's PR-#231-post baseline).
- Estimated complexity: small. Helper change ~50 lines Python, ensign-shared-core prose ~10 lines, tests ~50 lines. Spike ~$3-5. Total cost ~$15-20.
- Empirical evidence required at ideation gate: 5-run sonnet spike with verdict PASS/INCONCLUSIVE/FAIL per AC-4.

## Stage Report: ideation

- DONE: Run the empirical spike: 5 sonnet runs of a trivial DISPATCH_FILE fixture
  5/5 PASS on sonnet at `/tmp/rdt-spike/primary/` (summary.tsv); recommended 119-char prompt shape; representative ~3.6 KB fixture body; verdict PASS exceeds ≥4/5 threshold.
- DONE: Populate ## Design with the four design dimensions per the entity body's 'Proposed approach'
  Added ## Design section with (a) FO-authored elements table, (b) v2 schema bump rationale, (c) helper flow + Agent() forwarding contract + failure modes, (d) locked ensign prompt shape with three required components and reliability discussion.
- SKIPPED: Address Risk C: spike Option D in 2 runs as a fallback check during ideation if cycle-1 primary FAILs
  Not triggered. Primary spike was 5/5 PASS, so Option D fallback was unnecessary per the assignment instruction ("only fire if the primary shape disappoints").
- DONE: Tighten ACs to match what the spike empirically supports
  AC-1 updated to require the trailing imperative substring `treat its content as your assignment`; AC-4 amended to record 5/5 PASS satisfies the ideation gate; AC-5 left intact (savings claim verified at implementation time against real dispatch bodies).
- DONE: Document spike result in `## Empirical findings`
  Added section with per-run table, observed tool sequence, key reliability evidence (Skill failsafe works, no sonnet freelancing, trailing imperative is load-bearing), and Risk B residue note.

### Summary

Empirical spike PASSed 5/5 on sonnet, validating the proposed ~150-char prompt shape (`Skill(...); then Read {file} and treat its content as your assignment.`) reliably triggers the ensign to load the operating contract, Read the dispatch file, and execute the embedded checklist. Risk C (Option D fallback baking DISPATCH_FILE into `agents/ensign.md`) is not triggered. The four design dimensions are locked: helper writes per-dispatch content to `/tmp/spacedock-dispatch/{derived_name}.md`, emits a ~119-char `prompt` field plus a new `dispatch_file_path` field under `schema_version: 2`, FO forwards verbatim. Implementation is ready to proceed with helper-change + ensign-shared-core clause + tests as specified in ACs 1-7.

## Staff Review: ideation

Independent ideation-gate review by staff reviewer, 2026-05-22. Reviewer ran `claude-team build` against a real ideation- and implementation-stage input to measure today's dispatch sizes; verified the locked prompt shape against AC-1's substring contract; and cross-checked the spike summary against the entity body's claims.

### Material findings

- **F1. AC-5's "≥10,000 chars saved per dispatch" threshold is not supportable by current empirical evidence.** Measured today's `claude-team build` stdout against a realistic input (`entity=terminology-experiment`, `stage=implementation`, `team_name=spacedock-engineering`, scope_notes + 5-item checklist): `prompt` field is **2,451 bytes**, total stdout is **3,065 bytes**. A more loaded shape (longer scope_notes + `feedback_context` + `is_feedback_reflow=true`) yields `prompt`=**3,141 bytes**. The entity body's `## Problem` section asserts the v1 prompt is `~7,000 chars`; the measured number is ~2,500-3,200 for representative real inputs. With v2 emitting ~250 chars total stdout (per design §3), savings per dispatch are ~2,200-2,900 chars — **roughly a 4× shortfall against AC-5's 10,000 threshold**. The percentage win (~90-92%) is real and meaningful, but the absolute-character claim does not survive contact with the real helper output post-PR #231. AC-5 should be rewritten in percentage terms (e.g., ">=85% reduction in helper-stdout `prompt` field size against a canonical v1 fixture") or restated with a lower absolute floor (e.g., ">=2,000 chars saved on a feedback-reflow fixture"). Without that, the implementation cycle will either fail AC-5 or game it with an artificial fixture.

- **F2. The entity's headline "FO per-dispatch context cost ~14,000 chars" and "~7,000 chars from `claude-team build`'s stdout JSON" claims (lines 17-20) appear to overstate today's baseline.** Today's measured stdout is ~3,000 bytes (one read by the FO when parsing helper output, one when forwarded as `Agent(prompt=...)` arg). That's ~6,000 chars per dispatch passing through FO context, not ~14,000. The "~97% FO per-dispatch context reduction" predicted impact (line 399) — if recomputed as `(6000 - 500) / 6000 ≈ 92%` — is closer to ~92% than 97%. The benefit story is still substantial, but the captain should see the corrected number before approving. Either the entity body needs to be updated to reflect today's PR-#231-post-baseline, or the reviewer is wrong about what enters FO context per dispatch and the entity needs to call out the missing 8,000 chars.

- **F3. Spike fixture realism is plausible but unverified.** The entity body says the spike fixture was 3,632 bytes and "representative of a real PR #231 dispatch body". Measured real implementation dispatch is 2,451 bytes (prompt) / 3,065 bytes (stdout). The 3,632-byte fixture is in the right order of magnitude — slightly larger than a real dispatch, which is actually conservative for a reliability spike (longer body = more text to skip over before finding the checklist). The spike's PASS verdict probably transfers to real dispatches. **However:** the fixture files themselves were on tmpfs and are gone (`/tmp/spacedock-dispatch/spike-*.md` not present at review time), so the reviewer cannot independently verify the fixture content matched the structure of a real `claude-team build` output (Operating contract directive, header, entity-read instruction, scope notes, checklist, fetch-on-demand block, completion-signal block). The entity body asserts this content shape but doesn't preserve a copy. **Recommend:** implementation cycle re-runs at least 1 spike run against a real `claude-team build` v1 output (not a hand-crafted fixture) as part of AC-7's live-test or as a dedicated cycle-1 smoke check, so the v2 path is validated against actual helper-assembly content.

- **F4. AC-2's "decision deferred to implementation" is not appropriate for an ideation gate.** The workflow README at `docs/plans/README.md:73-79` says ideation outputs include "a fleshed-out task body with problem statement, proposed approach, acceptance criteria, and a test plan", and ACs must describe end-state properties. "OR cleanly errors out" with the decision punted to implementation is a stage-imperative ("implementer must decide") wearing an end-state mask. Either pick back-compat (v1 input keeps working, v2 input opts in) or pick clean-break (helper rejects v1 with `schema_version: 2 required`), and write the AC to that choice. The accompanying test then verifies the one chosen behavior. The current AC tests both branches, which means the test is "did the helper do something" — not a contract. **Recommend:** lock the decision now. Design §2 already states the back-compat path ("helper accepts v1 input and emits today's v1-shape output unchanged") as the chosen path. AC-2 should reflect that, not list both.

- **F5. AC-7's test pairing is mislabeled.** AC-7 names `test_dispatch_completion_signal` as "the natural test exercising single-dispatch shape". Reviewer did not verify this test name exists or that it exercises a fresh-Agent() dispatch (vs. a SendMessage reuse-path). If the named live test doesn't exist or doesn't dispatch via `Agent()`, this AC is untestable as specified. **Recommend:** implementation cycle either (a) verify `make test-live-claude TEST=test_dispatch_completion_signal` runs and dispatches a fresh ensign through the v2 path, or (b) name the actually-correct live test for AC-7 before locking the gate. The captain's cross-cycle live-test requirement deserves a verified test ID.

### Polish findings

- **P1. AC-1 substring assertion is correctly aligned with the locked shape.** Reviewer verified by string match: the design §4 locked shape `Skill(skill="spacedock:ensign"); then Read /tmp/spacedock-dispatch/{derived_name}.md and treat its content as your assignment.` contains all four required substrings (`Skill(skill="spacedock:ensign")`, `Read`, `/tmp/spacedock-dispatch/`, `treat its content as your assignment`). Length is 126 chars for typical names. No action needed.

- **P2. Risk B accept "no cleanup process" with 700KB at 100 dispatches is fine for v1, but worth a one-line note in the design that the implementation cycle will not add cleanup.** Captain sessions don't typically hit 100 dispatches; tmpfs handles long-term cleanup. The current design wording is "no cleanup process needed" — clear enough.

- **P3. Reuse-path scope cut is correct.** SendMessage reuse-advance is ~1500 chars per dispatch (entity body §`Out of scope`). At ~1500 chars × 100 reuse dispatches = 150KB session-total — small enough that compressing it is below the worth-it-now threshold. However, reviewer notes that **the reuse path doesn't compose with this entity's file-write pattern automatically** — the FO emits the SendMessage body directly, no helper involved. So when v2 lands, reuse-path costs are *unchanged* (no regression risk), but they don't get the benefit either. The "v2 reuse path" follow-up entity is correctly out of scope here. No action needed.

- **P4. Wording: "FO context savings are measured and ≥10,000 chars per dispatch" in AC-5** reads as an end-state property but is implicitly a stage-imperative (someone must do the measurement). The phrasing is borderline acceptable per the README's AC-vs-imperative distinction — the end-state is "the savings are measured at ≥10,000 chars". After F1's threshold correction, consider rephrasing to "Helper stdout `prompt` field for a canonical v2 input is ≤200 chars AND ≥85% smaller than the same input under v1." That removes both ambiguities (testable, end-state, no reference to "measuring").

- **P5. Risk C ("Spike-FAIL fallback") section can be deleted or shortened to one sentence.** The spike PASSed 5/5; Option D is unused. Keeping the risk section as-is is fine for traceability, but a one-line "Risk C not triggered; primary shape held" plus a pointer to the spike result would clean up the body.

### Out of review scope

- Did not run `make test-static` or `make test-live-claude`. Those are implementation-cycle artifacts, not ideation-gate evidence.
- Did not re-run the 5-sonnet spike. The summary.tsv at `/tmp/rdt-spike/primary/` shows 5/5 PASS and the per-run jsonl logs corroborate the Skill→Read→Write tool sequence. Reviewer trusts the artifact.
- Did not assess complexity estimate or cost. The `~50 lines Python + ~10 lines prose + ~50 lines tests` claim is plausible for the described work but reviewer did not size the helper file.

## Stage Report: implementation

- DONE: Implement the v2 path in `skills/commission/bin/claude-team build`: accept `schema_version: 2` input, write the full per-dispatch prompt to `/tmp/spacedock-dispatch/{derived_name}.md`, emit the ~119-char file-pointer prompt plus a new `dispatch_file_path` field
  Bumped `SCHEMA_VERSION = 2` + added file-write logic in commit 7c9467b0 (skills/commission/bin/claude-team:36-44, 449-485). CLEAN-BREAK on v1 input enforced; v1 stdin yields exit 2 with `schema_version: 2 required` per AC-2 + F4. Updated FO runtime adapter (claude-first-officer-runtime.md) and 4 input fixtures (`tests/fixtures/dispatch_inputs_*.json`) atomically.
- DONE: Add the DISPATCH_FILE clause to `skills/ensign/references/ensign-shared-core.md`
  Added `## DISPATCH_FILE Bootstrap` section (above `## Fetch-on-Demand Bootstrap`) with the two required properties: (a) Read instruction on the file-pointer pattern and (b) `SendMessage(to="team-lead", message="DISPATCH_FILE_MISSING: {path} - {error}")` failure handler. Verified by new static test `test_ensign_shared_core_carries_dispatch_file_bootstrap_clause` in `tests/test_agent_content.py`.
- DONE: Run `make test-live-claude` from the worktree (AC-7)
  See Stage Report below for the live-test outcome. F5 verified the test name: `tests/test_dispatch_completion_signal.py::test_dispatch_completion_signal` exists, is `@pytest.mark.live_claude`, and exercises a fresh `Agent(subagent_type="spacedock:ensign")` dispatch (not a SendMessage reuse path) via `_first_agent_dispatch_prompt`. AC-7 entity-body wording updated to name this verified test ID.

### Cross-cutting work

- F1 honored: AC-5 reframed in percentage terms (≥85% reduction in helper `prompt` field). The ≥10,000-char absolute floor is structurally unsupportable against today's ~2,500-3,200 byte v1 prompt; the percentage framing survives any reasonable input-size variation. New test `TestBuildV2SavingsFloor::test_v2_prompt_at_least_85pct_smaller_than_v1_equivalent` enforces the floor.
- F2 honored: entity-body `## Problem` and `## Scale context` corrected to reflect today's PR-#231-post baseline (~6,000 chars per dispatch, ~92% reduction), not the original ~14,000-char / ~97% claims.
- F4 honored: AC-2 locked CLEAN-BREAK per captain (2026-05-22). Helper rejects v1 input; no back-compat shim.
- F5 honored: AC-7 test name `test_dispatch_completion_signal` verified present + correct dispatch shape.

### AC-5 measurement on canonical fixture

| Field | Value |
|---|---|
| Input fixture | Team-mode dispatch, 3-item checklist, feedback_context + scope_notes |
| v1-equivalent dispatch body | 2,577 chars |
| v2 helper `prompt` field | 175 chars |
| Savings | 2,402 chars |
| Reduction | 93.2% (>85% threshold) |
| v2 helper stdout | 857 chars |
| FO per-dispatch (v1, prompt × 2) | ~5,154 chars |
| FO per-dispatch (v2, stdout + prompt arg) | 1,032 chars |
| FO per-dispatch reduction | ~80.0% |

### Test impact

- `make test-static`: 635 passed, 27 deselected, 15 subtests passed.
- 18 pre-existing tests migrated from `out["prompt"]` substring assertions to `dispatch_body(out)` (reads the file content the ensign would see). The helper introduces a small `dispatch_body(out)` test helper in `tests/test_claude_team.py:651-661`.
- Golden file `tests/fixtures/dispatch_prompt_fetchspec.txt` continues to byte-equal the v2 dispatch body — assembly logic was not changed, only the delivery shape moved (file vs. inline).
- New tests: `TestBuildDispatchFilePointer` (3 tests for AC-1 substring + length + file-existence + body-content), `TestBuildV2SavingsFloor::test_v2_prompt_at_least_85pct_smaller_than_v1_equivalent` (AC-5 floor), `test_ensign_shared_core_carries_dispatch_file_bootstrap_clause` (AC-3).

### Summary

CLEAN-BREAK v2 helper landed in commit 7c9467b0. `claude-team build` writes the per-dispatch body to `/tmp/spacedock-dispatch/{name}.md` and emits a 175-char file-pointer prompt; v1 input is rejected. Ensign-shared-core gained the `## DISPATCH_FILE Bootstrap` clause covering both the Read path and the `DISPATCH_FILE_MISSING:` failure mode. AC-5 measured 93.2% reduction on a canonical fixture (above the 85% floor); FO per-dispatch context drops ~80% (5,154 → 1,032 chars). All 635 static tests pass; staff-review findings F1, F2, F4, F5 honored. AC-7 live test outcome captured below.

### AC-7 live test outcome

`make test-live-claude TEST=test_dispatch_completion_signal` (run with `--model sonnet --effort low`): **PASSED** end-to-end in 167.97s. All 4 internal checks green: TeamCreate emitted, work dispatch closed in 31.9s, entity advanced and was archived, team-mode ensign dispatch carries SendMessage completion-signal instruction (verified against the dispatch body via `_first_agent_dispatch_body`).

Field-fix discovered during live testing: the Agent tool requires a `description` argument, which the FO runtime adapter did not name in its forwarding block (only `subagent_type, name, team_name, model, prompt`). The first live dispatch hit `InputValidationError: required parameter description is missing`; the model retried with `description` included and the second call succeeded. This was a latent bug in the FO runtime adapter that v1 happened to gloss over because the model often inferred `description` from the prompt content (less likely under v2's tiny prompt). Fixed in this entity by adding `description=output.description` to the runtime adapter's Agent() forwarding block, plus an explicit `REQUIRED — Agent tool rejects missing description` annotation.

Test harness update: `_first_agent_dispatch_body` in `tests/test_dispatch_completion_signal.py` extracts the dispatch file path from the v2 prompt and Reads the body so substring assertions about completion-signal content target the right surface.

## Stage Report: validation

- DONE: Re-run `make test-live-claude` from the worktree (AC-7) to confirm reproducibility
  Invoked as `uv run pytest tests/test_dispatch_completion_signal.py --runtime claude --model sonnet --effort low -v` (matches the implementation's recorded `--model sonnet --effort low` overrides; the plain `make test-live-claude` target defaults to haiku per `tests/conftest.py:25`, and the test xfails by design on haiku-4-5 per `anthropics/claude-code#26426`). Result: **PASSED in 175.26s** (+4.3% drift vs implementation's 167.97s baseline, well under the captain's 10% threshold).
- DONE: Static AC cross-check against final entity body and committed implementation
  AC-1 verified: `TestBuildDispatchFilePointer` tests at `tests/test_claude_team.py:1304` cover ≤200-char prompt + 4 substrings + `dispatch_file_path` field + on-disk content. AC-2 verified: `TestBuildSchemaVersion::test_build_schema_version_v1_rejection` at `tests/test_claude_team.py:1289` asserts exit 2 with `schema_version: 2 required` stderr. AC-3 verified: `skills/ensign/references/ensign-shared-core.md:83-99` carries the DISPATCH_FILE clause with both the Read instruction and the `DISPATCH_FILE_MISSING:` SendMessage failure handler. AC-4 verified: `## Empirical findings` section (entity body lines 192-220) preserves the 5/5 PASS per-run table verbatim. AC-5 verified: `TestBuildV2SavingsFloor::test_v2_prompt_at_least_85pct_smaller_than_v1_equivalent` at `tests/test_claude_team.py:1376` enforces the floor; canonical-fixture measurement 93.2% recorded in implementation Stage Report. Description-forwarding fix verified at `skills/first-officer/references/claude-first-officer-runtime.md:103` (`description=output.description, // REQUIRED — Agent tool rejects missing description`).
- DONE: `make test-static` regression check (AC-6)
  Re-ran in worktree: **635 passed, 27 deselected, 15 subtests passed in 30.33s** — byte-exact match to implementation's reported count. No regressions.

### Summary

All seven ACs hold against the committed implementation (commits 7c9467b0 + bfce3da2). AC-7 live test reproduces PASSED in 175.26s (+4.3% drift, within the captain's 10% threshold). `make test-static` reproduces 635 passed exactly. The Agent-`description` fix is in place at the FO runtime adapter line 103. Verdict: **PASSED**.

## Stage Report: implementation (cycle 2)

- DONE: Migrate `tests/test_checklist_e2e.py` live-claude `agent_prompt` validation to handle the v2 file-pointer shape
  After `agent_prompt = log.agent_prompt()` (claude branch only, codex branch untouched), detect the v2 shape via `re.search(r"Read (/tmp/spacedock-dispatch/[^\s]+\.md)", agent_prompt)` — when matched and the file is readable, reassign `agent_prompt` to the file body so the subsequent substring assertions (`Completion checklist` at line 184, checklist item extraction at line 202) land on the materialized dispatch body. Break-glass fallback (fully-inlined prompt, no `Read /tmp/spacedock-dispatch/...` substring) falls through unchanged.
- DONE: `make test-static` regression check
  635 passed, 27 deselected, 15 subtests passed in 29.25s — byte-exact match to validation-cycle baseline. No new failures.
- SKIPPED: Re-run `make test-live-claude` locally
  Per assignment instruction; CI on PR #234 re-runs the live suite after push.

### Summary

CYCLE-2 implementation remediation initiated by CI feedback on PR #234. Migrated the single missed test file (`tests/test_checklist_e2e.py`) to the v2 file-pointer shape using the same regex/Read pattern as `dispatch_body(out)` in `tests/test_claude_team.py:651-661`, but operating on a string (`agent_prompt` from `LogParser.agent_prompt()`) rather than a parser output dict. Production code untouched; helper architecture unchanged. Static suite stays at 635 passed; CI on PR #234 will re-verify the live-claude bare and team paths.

## Stage Report: implementation (cycle 3)

- DONE: Migrate `tests/test_fetch_on_demand_dispatch.py` to handle the v2 file-pointer shape
  Inserted the cycle-2 detection block immediately after `agent_prompt = log.agent_prompt()` (around line 88): `re.search(r"Read (/tmp/spacedock-dispatch/[^\s]+\.md)", agent_prompt)`; when matched and the file is readable, reassign `agent_prompt` to the file body so the `### Fetch commands` and `claude-team show-stage-def` substring assertions land on the materialized dispatch body. Break-glass fallback falls through unchanged.
- DONE: Migrate `tests/test_standing_teammate_spawn.py` to handle the v2 file-pointer shape
  Added `import re` and inserted the same detection block immediately after `ensign_prompt = _agent_input(ensign_dispatch).get("prompt", "")` (around line 147); when matched and the file is readable, reassign `ensign_prompt` to the file body so the `claude-team show-standing` / `### Standing teammates available in your team` substring assertions land on the materialized dispatch body. `Path` was already imported.
- DONE: Audit `tests/` for remaining `agent_prompt`/`ensign_prompt` consumers
  `grep -lE 'log\.agent_prompt\(\)|LogParser.*agent_prompt|ensign_prompt ='` against `tests/` returns exactly three files: `test_checklist_e2e.py` (migrated cycle-2), `test_fetch_on_demand_dispatch.py` + `test_standing_teammate_spawn.py` (migrated this cycle). No fourth file surfaced.
- DONE: `make test-static` regression check
  635 passed, 27 deselected, 15 subtests passed in 28.95s — byte-exact match to cycle-2 baseline. No new failures.
- SKIPPED: Re-run `make test-live-claude` locally
  Per assignment instruction; CI on PR #234 re-runs the live suite after push.

### Summary

CYCLE-3 fix-forward: migrated the two remaining live tests that prior CI on PR #234 flagged as still asserting against the v1-shaped `agent_prompt`/`ensign_prompt`. Pattern is byte-similar to cycle-2's `test_checklist_e2e.py` migration (the 10-line `re.search(r"Read (/tmp/spacedock-dispatch/[^\s]+\.md)", ...)` block). No production code changes; helper architecture unchanged. The three migrated tests now share the same inline detection pattern; a future refactor can DRY this into a `dispatch_body_from_prompt(s)` helper if appropriate. Static suite stays at 635 passed; CI on PR #234 will re-verify the live-claude bare and team paths once the captain approves the next run.
