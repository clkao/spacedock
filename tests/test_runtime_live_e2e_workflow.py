#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Static checks for the runtime live E2E workflow and its operator docs.
# ABOUTME: Verifies PR/manual triggers, CI-E2E approval gating, and the two runtime jobs.

from __future__ import annotations

from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parent.parent
WORKFLOW_PATH = REPO_ROOT / ".github" / "workflows" / "runtime-live-e2e.yml"
README_PATH = REPO_ROOT / "tests" / "README.md"
TOP_README_PATH = REPO_ROOT / "README.md"
MAKEFILE_PATH = REPO_ROOT / "Makefile"
GATE_GUARDRAIL_PATH = REPO_ROOT / "tests" / "test_gate_guardrail.py"
CODEX_PACKAGED_AGENT_PATH = REPO_ROOT / "tests" / "test_codex_packaged_agent_e2e.py"
MERGE_HOOK_GUARDRAIL_PATH = REPO_ROOT / "tests" / "test_merge_hook_guardrail.py"


def read_workflow() -> str:
    return WORKFLOW_PATH.read_text()


def read_readme() -> str:
    return README_PATH.read_text()


def read_makefile() -> str:
    return MAKEFILE_PATH.read_text()


def read_gate_guardrail() -> str:
    return GATE_GUARDRAIL_PATH.read_text()


def read_codex_packaged_agent() -> str:
    return CODEX_PACKAGED_AGENT_PATH.read_text()


def read_merge_hook_guardrail() -> str:
    return MERGE_HOOK_GUARDRAIL_PATH.read_text()


def section(text: str, heading: str) -> str:
    marker = f"{heading}:"
    start = text.index(marker)
    lines = []
    for line in text[start:].splitlines()[1:]:
        if line.startswith("  ") and not line.startswith("    "):
            break
        lines.append(line)
    return "\n".join(lines)


def test_runtime_live_e2e_workflow_supports_pr_and_manual_triggers():
    text = read_workflow()

    # pull_request_target runs with base-branch secrets even for fork PRs.
    # Env-approval gate is the sole protection; see top-of-file security comment.
    assert "pull_request_target:" in text
    assert "types: [opened, synchronize, reopened]" in text
    assert "workflow_dispatch:" in text
    # pr_number dispatch input is dropped — head-SHA checkout makes it redundant.
    assert "pr_number:" not in text
    # Manual-dispatch operator knobs are preserved.
    for knob in ("claude_version:", "test_selector:", "effort_override:", "model_override:"):
        assert knob in text


def test_runtime_live_e2e_workflow_checks_out_pr_head_with_persist_credentials_false():
    """Security-critical: every checkout in this workflow must target the PR
    merge ref (refs/pull/<N>/merge — PR head merged into the target) and MUST
    set persist-credentials: false so GITHUB_TOKEN is not persisted into
    .git/config where post-checkout steps could use it.

    Testing the merge ref (not the bare PR head) catches base-drift: a PR that
    was green against an older main may fail once new commits land. The trust
    boundary (env-approval before the job starts) is unchanged.

    Five checkouts in current shape: 4 live-e2e jobs (claude-live, claude-live-bare,
    claude-live-opus, codex-live) + 1 static-offline gate job. All five must
    satisfy the merge-ref + persist-credentials invariant."""
    text = read_workflow()

    checkout_count = text.count("actions/checkout@v4")
    merge_ref_count = text.count(
        "ref: refs/pull/${{ github.event.pull_request.number }}/merge"
    )
    # Each checkout must target the PR merge ref.
    assert checkout_count == 5
    assert merge_ref_count == checkout_count, (
        f"Every checkout@v4 must pin to the PR merge ref "
        f"(found {checkout_count} checkouts, {merge_ref_count} merge-ref refs)"
    )
    # The bare PR head SHA must NOT appear as a checkout ref — that would skip
    # the merge-with-target step and reintroduce base-drift blind spots.
    assert "ref: ${{ github.event.pull_request.head.sha }}" not in text, (
        "Checkout ref must be the merge ref, not the bare PR head SHA"
    )
    # persist-credentials: false appears on every checkout step AND is documented
    # in the top-of-file security comment — so count is checkout_count + 1.
    persist_false_count = text.count("persist-credentials: false")
    assert persist_false_count == checkout_count + 1, (
        f"Expected persist-credentials: false on all {checkout_count} checkouts "
        f"+ 1 reference in the security comment, got {persist_false_count}"
    )


def test_runtime_live_e2e_workflow_gates_live_jobs_on_static_offline():
    """Live-e2e jobs MUST NOT run if static-offline fails. Captain env-approval
    is expensive; gating on static-offline prevents wasted approvals on runs
    that would fail on static checks anyway."""
    text = read_workflow()

    # The gate job must exist inline (not only in the standalone ci-static.yml).
    assert "  static-offline:\n" in text, (
        "Expected a static-offline job in runtime-live-e2e.yml as the needs: target"
    )
    # All 4 live jobs must declare needs: static-offline.
    needs_count = text.count("needs: static-offline")
    assert needs_count == 4, (
        f"Expected all 4 live-e2e jobs to declare 'needs: static-offline', got {needs_count}"
    )


def test_runtime_live_e2e_workflow_narrows_default_permissions():
    text = read_workflow()

    assert "permissions:\n  contents: read\n  pull-requests: read\n" in text


def test_runtime_live_e2e_workflow_documents_security_model_at_top():
    """Because pull_request_target runs fork PR code with repo secrets, the
    env-approval gate is load-bearing. Maintainers must see this before
    editing the workflow."""
    text = read_workflow()

    assert "SECURITY MODEL" in text
    assert "env-approval gate" in text
    assert "ONLY protection" in text
    assert "review the PR head SHA" in text
    assert "refs/pull/<N>/merge" in text


def test_runtime_live_e2e_workflow_has_expected_runtime_jobs():
    text = read_workflow()
    claude_section = section(text, "  claude-live")
    claude_bare_section = section(text, "  claude-live-bare")
    claude_opus_section = section(text, "  claude-live-opus")
    codex_section = section(text, "  codex-live")

    assert "\n  claude-live:\n" in text
    assert "\n  claude-live-bare:\n" in text
    assert "\n  claude-live-opus:\n" in text
    assert "\n  codex-live:\n" in text
    assert "environment:" in claude_section
    assert "name: CI-E2E" in claude_section
    assert "deployment: false" in claude_section
    assert "environment:" in claude_bare_section
    assert "name: CI-E2E" in claude_bare_section
    assert "deployment: false" in claude_bare_section
    assert "environment:" in claude_opus_section
    assert "name: CI-E2E-OPUS" in claude_opus_section
    assert "deployment: false" in claude_opus_section
    assert "environment:" in codex_section
    assert "name: CI-E2E-CODEX" in codex_section
    assert "deployment: false" in codex_section
    assert "path classifier" not in text.lower()
    assert "shard" not in text.lower()
    assert "matrix:" not in text


def test_runtime_live_e2e_workflow_preserves_and_uploads_live_test_dirs():
    text = read_workflow()
    claude_section = section(text, "  claude-live")
    claude_bare_section = section(text, "  claude-live-bare")
    claude_opus_section = section(text, "  claude-live-opus")
    codex_section = section(text, "  codex-live")

    for job_section, artifact_name in (
        (claude_section, "runtime-live-e2e-claude-live"),
        (claude_bare_section, "runtime-live-e2e-claude-live-bare"),
        (claude_opus_section, "runtime-live-e2e-claude-live-opus"),
        (codex_section, "runtime-live-e2e-codex-live"),
    ):
        assert 'KEEP_TEST_DIR: "1"' in job_section
        assert "SPACEDOCK_TEST_TMP_ROOT:" not in job_section
        assert 'echo "SPACEDOCK_TEST_TMP_ROOT=$RUNNER_TEMP/spacedock-live/$GITHUB_JOB" >> "$GITHUB_ENV"' in job_section
        assert 'mkdir -p "$RUNNER_TEMP/spacedock-live/$GITHUB_JOB"' in job_section
        assert "uses: actions/upload-artifact@v4" in job_section
        assert "if: always()" in job_section
        assert f"name: {artifact_name}" in job_section
        assert "path: ${{ runner.temp }}/spacedock-live/${{ github.job }}" in job_section

    assert "Login Codex with API key" in codex_section
    assert "printenv OPENAI_API_KEY | codex login --with-api-key" in codex_section


def test_runtime_live_e2e_workflow_scopes_secrets_to_the_matching_job():
    text = read_workflow()
    claude_section = section(text, "  claude-live")
    claude_bare_section = section(text, "  claude-live-bare")
    claude_opus_section = section(text, "  claude-live-opus")
    codex_section = section(text, "  codex-live")

    assert "ANTHROPIC_API_KEY" in claude_section
    assert "OPENAI_API_KEY" not in claude_section
    assert "ANTHROPIC_API_KEY" in claude_bare_section
    assert "OPENAI_API_KEY" not in claude_bare_section
    assert "ANTHROPIC_API_KEY" in claude_opus_section
    assert "OPENAI_API_KEY" not in claude_opus_section
    assert "OPENAI_API_KEY" in codex_section
    assert "ANTHROPIC_API_KEY" not in codex_section
    assert "is required for claude-live." in claude_section
    assert "is required for claude-live-bare." in claude_bare_section
    assert "is required for claude-live-opus." in claude_opus_section
    assert "is required for codex-live" in codex_section


def test_runtime_live_e2e_workflow_uses_stable_make_targets_and_provenance_fields():
    text = read_workflow()
    claude_section = section(text, "  claude-live")
    claude_bare_section = section(text, "  claude-live-bare")
    claude_opus_section = section(text, "  claude-live-opus")

    assert "make test-live-claude" in text, "claude-live job should call make test-live-claude"
    assert "make test-live-claude-bare" in claude_bare_section, (
        "claude-live-bare job should call make test-live-claude-bare"
    )
    assert "make test-live-claude-opus" in claude_opus_section, (
        "claude-live-opus job should call make test-live-claude-opus"
    )
    assert "make test-live-codex" in text, "codex-live job should call make test-live-codex"

    # Team-flag matrix: teams-mode jobs set the env var to "1"; the bare job
    # explicitly sets it to "0" so the pytest conftest auto-resolution picks bare.
    assert 'CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS: "1"' in claude_section
    assert 'CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS: "1"' in claude_opus_section
    assert 'CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS: "0"' in claude_bare_section

    # Provenance field labels under the pull_request_target + merge-ref design:
    # - "Base SHA" is the workflow-definition commit on the target branch.
    # - "PR head SHA (diff under review)" is the PR head — the thing a
    #   maintainer inspects before approving the env deployment.
    # - "Checkout ref" names the merge ref the job checks out.
    # - The "Resolved merge SHA" printer step emits the post-merge HEAD so the
    #   audit trail records what actually ran (merge-ref resolution is dynamic).
    for field in (
        "PR number",
        "Base SHA",
        "PR head SHA (diff under review)",
        "Checkout ref",
        "Resolved merge SHA",
        "same-repo",
        "fork",
        "Approval context",
        "Trigger source",
    ):
        assert field in text

    # pull_request_target makes the PR payload directly available — no REST
    # lookup, no dispatch-vs-pr disambiguation needed.
    assert "context.payload.pull_request" in text
    assert "inputs.pr_number" not in text
    assert "TRIGGER_SOURCE" not in text
    assert "DISPATCH_PR_NUMBER" not in text
    assert "continue-on-error" not in text
    assert "|| true" not in text


def test_live_makefile_targets_do_not_require_bash_without_declaring_it():
    text = read_makefile()

    assert "test-live-claude:" in text
    assert "test-live-codex:" in text

    if "set -euo pipefail" in text:
        assert "SHELL := /bin/bash" in text


def test_live_makefile_uses_pytest_markers_not_raw_script_chains():
    """Post #148 migration: the Makefile uses pytest marker filters
    (live_claude + serial / not serial) instead of raw `uv run tests/*.py` chains.
    Any per-test skip lives as a decorator on the test file, not a Makefile comment."""
    text = read_makefile()

    # Raw script invocations are gone — pytest with marker filter replaces them.
    assert "\n\tuv run tests/test_push_main_before_pr.py" not in text
    assert "\n\tuv run tests/test_rebase_branch_before_push.py" not in text
    assert "\n\tuv run tests/test_dispatch_completion_signal.py --runtime claude" not in text
    assert '-m "live_claude and serial"' in text
    assert '-m "live_claude and not serial"' in text

    test_path = Path(__file__).resolve().parent / "test_push_main_before_pr.py"
    if test_path.exists():
        body = test_path.read_text()
        # Once migrated to pytest the skip moves into a decorator on the test function.
        if "import pytest" in body:
            assert "pytest.mark.skip" in body
            assert "#114" in body


def test_codex_makefile_targets_allow_empty_tiers_without_masking_failures():
    text = read_makefile()

    assert text.count("scripts/run_pytest_tier.py --allow-no-tests --") == 4
    assert '-m "live_codex and serial" --runtime codex -x -v' in text
    assert '-m "live_codex and not serial" --runtime codex' in text
    assert '-m "live_codex and serial" --runtime codex --team-mode=bare -x -v' in text
    assert '-m "live_codex and not serial" --runtime codex --team-mode=bare' in text
    assert text.count("scripts/run_pytest_tier.py --allow-no-tests --") == text.count(
        'live_codex and serial" --runtime codex'
    ) + text.count('live_codex and not serial" --runtime codex')


def test_gate_guardrail_is_the_shared_serial_live_preflight():
    text = read_gate_guardrail()

    assert "@pytest.mark.live_claude" in text
    assert "@pytest.mark.live_codex" in text
    assert "@pytest.mark.serial" in text


def test_codex_packaged_agent_stays_in_parallel_live_tier():
    text = read_codex_packaged_agent()

    assert "@pytest.mark.live_codex" in text
    assert "@pytest.mark.serial" not in text


def test_merge_hook_guardrail_is_not_locally_xfailed_or_moved_to_serial_tier():
    text = read_merge_hook_guardrail()

    assert "@pytest.mark.live_codex" in text
    assert "@pytest.mark.serial" not in text
    assert "@pytest.mark.xfail" not in text


def test_tests_readme_documents_codex_preflight_before_parallel_tier():
    text = read_readme()

    assert "test_gate_guardrail.py --runtime codex" in text
    assert "before burning the expensive parallel tier" in text
    assert "shared runtime pilot" in text


def test_tests_readme_documents_runtime_live_e2e_workflow():
    text = read_readme()

    assert "runtime-live-e2e.yml" in text
    assert "workflow_dispatch" in text
    assert "pull_request_target" in text
    assert "CI-E2E" in text
    assert "CI-E2E-OPUS" in text
    assert "claude-live" in text
    assert "claude-live-opus" in text
    assert "codex-live" in text
    assert "ANTHROPIC_API_KEY" in text
    assert "OPENAI_API_KEY" in text
    assert "PR number" in text
    assert "Base SHA" in text
    assert "PR head SHA" in text
    assert "same-repo vs fork" in text
    assert "approval/reviewer context" in text
    assert "Trigger source" in text
    assert "job stays red" in text
    assert "pending environment review" in text
    # The env-approval gate is the security guarantee for fork-PR runs —
    # README must tell maintainers what to review before approving.
    assert "env-approval gate" in text or "environment approval" in text


def _makefile_target_body(text: str, target: str) -> str:
    marker = f"\n{target}:\n"
    start = text.index(marker) + len(marker)
    lines = []
    for line in text[start:].splitlines():
        if line and not (line.startswith("\t") or line.startswith(" ") or line.startswith("#")):
            break
        lines.append(line)
    return "\n".join(lines)


def test_runtime_live_e2e_bare_job_displays_sonnet_as_effective_model():
    """AC-2: the claude-live-bare job's tool-versions step must display sonnet
    (not haiku) as the effective model when MODEL_OVERRIDE is empty. The bare
    job's default routes through `make test-live-claude-bare`, which now
    pins --model sonnet."""
    text = read_workflow()
    bare_section = section(text, "  claude-live-bare")

    assert 'EFFECTIVE_MODEL="(make default — sonnet)"' in bare_section
    assert 'EFFECTIVE_MODEL="(pytest default — haiku)"' not in bare_section


def test_makefile_bare_target_pins_sonnet_and_drops_haiku():
    """AC-3: the test-live-claude-bare target must default to sonnet (via the
    BARE_MODEL variable) and must not invoke --model haiku."""
    text = read_makefile()

    assert "BARE_MODEL ?= sonnet" in text

    bare_body = _makefile_target_body(text, "test-live-claude-bare")
    assert "--model $(BARE_MODEL)" in bare_body
    assert "--model haiku" not in bare_body


def test_top_readme_documents_supported_models_floor():
    """AC-4: the top-level README must declare the sonnet-or-above floor for
    bare-mode FO and cross-reference the archived empirical evidence."""
    text = TOP_README_PATH.read_text()

    assert "## Supported models" in text
    supported_start = text.index("## Supported models")
    supported_end = text.index("\n## ", supported_start + 1)
    supported_section = text[supported_start:supported_end]

    assert "sonnet" in supported_section.lower()
    assert "bare-mode" in supported_section.lower() or "bare mode" in supported_section.lower()
    assert "_archive/haiku-bare-fo-startup-protocol-adherence.md" in supported_section
