---
id: "213"
title: "test_codex_plugin_manifest_matches_approved_contract lags behind version bumps"
status: done
source: "#212 ensign's make test-static run observed `test_codex_plugin_manifest_matches_approved_contract` failing on main — pre-existing from the local `0d2d7a45 release: bump version to spacedock@0.10.0` commit (CL, 2026-04-20 20:11 PDT). Plugin manifests updated to 0.10.0; test assertion at `tests/test_codex_plugin_packaging.py:35` still pins `== \"0.9.6\"`."
started: 2026-04-21T04:08:04Z
completed: 2026-04-21T04:14:04Z
verdict: PASSED
score: 0.45
worktree: 
issue:
pr:
mod-block:
---

# `test_codex_plugin_manifest_matches_approved_contract` lags behind version bumps

## Problem statement

`tests/test_codex_plugin_packaging.py` hardcodes the plugin version in an "approved contract" assertion:

```python
# tests/test_codex_plugin_packaging.py:35
assert manifest["version"] == "0.9.6"
```

When the repo cuts a new release (e.g., the local `0d2d7a45 release: bump version to spacedock@0.10.0`), `.codex-plugin/plugin.json` and `.claude-plugin/plugin.json` get bumped but the test pin doesn't. Result: `test-static` goes red immediately after any version bump, forcing a follow-up test-patch commit per bump.

## Scale

Every release bump (roughly one per 1-2 weeks recently) trips this. The 0.10.0 bump tripped it; the 0.9.6 bump before that presumably didn't trip it *because* the bump commit included the test update. That's fragile — it relies on whoever does the version bump remembering to also touch this test.

## Fix options

### Option A — Derive the pinned version from `.codex-plugin/plugin.json` itself

Make the assertion self-referential. The test no longer "approves" a specific version string; it approves the *schema* (presence of `version`, specific format). This removes the lag class entirely but also removes the "approved contract" gate (the test no longer rejects unintentional version changes).

```python
assert "version" in manifest
assert re.match(r"^\d+\.\d+\.\d+$", manifest["version"])
```

### Option B — Read the version from a single authoritative source

Designate one file (e.g., `pyproject.toml` or a new `VERSION` file) as the authoritative version, and have both plugin manifests + the test read it from there. This keeps the "approved" nature (all three files must agree) but moves the drift point from three places down to one.

### Option C — Add a lint target that fails bumps until the test is updated

Add a `make check-version-sync` target or pre-commit hook that compares the test's pinned version to `.codex-plugin/plugin.json["version"]` and fails if they disagree. Doesn't remove the lag, just forces it to be caught earlier.

## Acceptance criteria

**AC-1 — A version bump no longer requires hand-editing `tests/test_codex_plugin_packaging.py`.**
Verified by: simulate a version bump (change both plugin JSONs to a new string), run `make test-static`, confirm the plugin-packaging test still passes without editing the test file.

**AC-2 — Unintentional version drift between the two plugin manifests is still caught.**
Verified by: `test_legacy_plugin_manifest_is_a_synchronized_mirror` at line 77 stays green (it already asserts claude-plugin mirrors codex-plugin). Additionally, whichever single-source-of-truth we adopt in Option B must be asserted equal to both manifests.

**AC-3 — Current main goes green.**
Verified by: `make test-static` passes at ≥ the current count (511 on current local main including #212's additions).

## Out of scope

- Rewriting the broader approved-contract assertions (description, author, license, keywords, skills). Those are intentional gates against unintentional drift.
- Moving to a release-automation tool. Keep it manual; just stop stepping on the version pin.
- Refactoring `test_codex_marketplace_matches_approved_contract` (line 53). It doesn't pin a version string and isn't affected.

## Test plan

Whichever option ships, the existing ~5 assertions in `test_codex_plugin_manifest_matches_approved_contract` need to either stay (Option B/C) or move to a schema-shape form (Option A). The sibling `test_legacy_plugin_manifest_is_a_synchronized_mirror` assertion stays untouched.

Ideation should pick between Options A/B/C (my lean: B — single-source-of-truth is the smallest change that removes the lag).

## Cross-references

- Observed during #212 ensign's `make test-static` run, 2026-04-21
- Local `0d2d7a45 release: bump version to spacedock@0.10.0` (CL, 2026-04-20 20:11 PDT) is the trigger commit (currently unpushed)
- `tests/test_codex_plugin_packaging.py:35` is the failing assertion site
