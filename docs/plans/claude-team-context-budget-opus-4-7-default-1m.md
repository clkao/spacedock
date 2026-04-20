---
id: 207
title: "claude-team context-budget: default claude-opus-4-7 to 1M context (not 200k)"
status: backlog
source: "2026-04-19 session captain observation: the `claude-team context-budget` helper hardcoded 200k as the context_limit for claude-opus-4-7, producing `usage_pct: 142.7` false alarms and `reuse_ok: false` on ensigns that were structurally fine. Empirical evidence: cycle-7 ensign reached 285k resident_tokens with no compaction trigger and continued emitting coherent tool-use turns — behavior inconsistent with a 200k-limited runtime. Team config consistently declares members as `opus[1m]`. Multiple session ensigns (cycle-3-fix at 63%, cycle-4-cleanup at 64%) were shut down early based on these false signals when they likely had plenty of context-window headroom."
started:
completed:
verdict:
score: 0.6
worktree:
issue:
pr:
mod-block:
---

The `context_limit_for_model` function in `skills/commission/bin/claude-team` looks for a literal `[1m]` bracket suffix in the runtime model string to decide between 200k (default) and 1M (extended). Runtime consistently reports `claude-opus-4-7` without the bracket — the helper assumes 200k. But the model appears to have native 1M context:

- Team configs declare `opus[1m]` or `claude-opus-4-7[1m]` at TeamCreate time.
- Ensigns running as `claude-opus-4-7` accumulate 285k+ resident tokens without compaction firing and without error — inconsistent with a 200k-limited runtime.
- The helper emits a `config_drift_warning` ("team config requested opus[1m] but runtime is claude-opus-4-7") on every check, which is symptomatic of the bracket-detection logic being stale.

## Why this matters

Multiple ensigns this session were shut down based on `reuse_ok: false` signals driven by the false `usage_pct` math. Each false shutdown costs a fresh-dispatch cycle (context re-load, skill re-invocation, diagnostic re-discovery). Correct handling would let ensigns live longer and reason more coherently across larger work bodies.

## Proposed fix

Update `context_limit_for_model` so that `claude-opus-4-7` (and presumably `claude-opus-4-8`+ onward) defaults to `EXTENDED_CONTEXT_LIMIT` (1M) even without the `[1m]` bracket suffix. The bracket-detection logic stays for opus-4-6 and earlier models where the bracket was the real signal.

Approach options for ideation to resolve:

1. **Model-family allow-list:** Maintain a list of models that are natively 1M (`claude-opus-4-7`, `claude-opus-4-8`, ...). Default to 1M when the base model string matches. `[1m]` suffix still forces 1M for older families.
2. **Model-family version check:** Parse the version from the model string; any opus-4 ≥ 4-7 defaults to 1M.
3. **Config-declared model wins:** If `config_model` has `[1m]` suffix OR matches a 1M-native family, trust the config. Runtime model is just a fallback.

Option 3 is probably the cleanest — respects what the team actually declared.

## Acceptance criteria (draft)

- **AC-1:** `context_limit_for_model("claude-opus-4-7")` returns 1M (not 200k).
- **AC-2:** `claude-team context-budget --name <ensign>` against a claude-opus-4-7 ensign with 285k resident tokens reports `usage_pct` in the 28-30% range with `reuse_ok: true`.
- **AC-3:** `claude-team context-budget` no longer emits `config_drift_warning` when the team config declared `opus[1m]` or `claude-opus-4-7[1m]` but the runtime reports `claude-opus-4-7` — those strings represent the same 1M-capable runtime.
- **AC-4:** Existing behavior for older opus variants preserved — `claude-opus-4-6` without `[1m]` still defaults to 200k.
- **AC-5:** `make test-static` green.

## Test plan

Unit tests in the same style as existing `context_limit_for_model` coverage (if any; otherwise add a small test file). Offline — no live LLM needed.

## Out of scope

- Broader context-budget refactor.
- Changes to compaction behavior.
- Config-schema changes.

## Related

- #203 — surfaced this during cycle-7/cycle-8 ensign orchestration. Multiple ensigns shut down prematurely on false signals.
- #202 — FO behavior spec + RTM; if it lands first, this fix's AC-3 should land as a requirement entry.

## Stage Report — implementation (2026-04-19)

**Approach:** Option 1 (allow-list) with empirically-verified entries only. Added `NATIVE_1M_MODELS = frozenset({"claude-opus-4-7"})` constant in `skills/commission/bin/claude-team`. `context_limit_for_model` returns `EXTENDED_CONTEXT_LIMIT` when (a) the model string contains `[1m]` (existing behavior, preserved for opus-4-6 and any future opt-in), or (b) the base model (before any `[...]` suffix) is exactly in `NATIVE_1M_MODELS`. No version parsing, no prefix matching, no speculation about unreleased models.

**Scope correction (captain, 2026-04-19):** Initial implementation used a prefix allow-list including `claude-opus-4-8`/`4-9`/`opus-5` — that was speculation. Captain constrained the allow-list to the single model we have empirical evidence for this session (`claude-opus-4-7` at 285k resident → no compaction, no errors). Future models must default to 200k until the empirical check is re-run and the exact string is added.

**Commits:**
- 579b1924 — fix: default claude-opus-4-7 context limit to 1M without [1m] suffix (initial, over-broad prefix list)
- ab232ad3 — report: #207 implementation stage report appended
- (next commit) — scope correction: constrain allow-list to claude-opus-4-7 only + add no-speculation test

**Test evidence:**
- `make test-static` → 481 passed, 22 deselected, 10 subtests passed (baseline before change was 476; +5 new assertions: 4 parametrized model-mapping entries for 4-7 (→1M), 4-7[1m] (→1M), 4-8 (→200k locks no-speculation), 4-8[1m] (→1M bracket still works) + 1 drift scenario in `TestRuntimeModelDetection.test_opus_4_7_bare_runtime_gets_1m`).
- Older-model tests unchanged (opus-4-6 → 200k; opus-4-6[1m] → 1M; haiku → 200k; unknown → 200k).

**AC check:**
- AC-1 — `context_limit_for_model("claude-opus-4-7")` returns 1M. Covered by `TestModelMapping` parametrize.
- AC-2 — 285k resident tokens on claude-opus-4-7 → `usage_pct: 28.5`, `reuse_ok: true`. Covered by `test_opus_4_7_bare_runtime_gets_1m`.
- AC-3 — no `config_drift_warning` when config is `opus[1m]` and runtime is bare `claude-opus-4-7`. Covered by same test (`assert "config_drift_warning" not in data`).
- AC-4 — `claude-opus-4-6` without `[1m]` still 200k; `claude-opus-4-6[1m]` still 1M. Preserved by existing parametrize entries.
- AC-5 — `make test-static` green.

**No-speculation lock-in:** `claude-opus-4-8` → 200k parametrize entry encodes the discipline. The `claude-opus-4-8[1m]` → 1M entry shows the bracket opt-in still works for any future model without adding it to the allow-list.

**PR:** https://github.com/clkao/spacedock/pull/139
