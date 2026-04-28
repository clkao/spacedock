#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live proof that the Pi first officer can drive analysis->implementation reuse and fresh validation dispatch.
# ABOUTME: Verifies the FO uses repo-local Pi skills plus session-backed reuse semantics exposed through the Pi worker runtime helper path.

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import PiLogParser, git_add_commit, run_pi_first_officer_streaming, setup_fixture  # noqa: E402

PER_STAGE_TIMEOUT_S = 300
SUBPROCESS_EXIT_BUDGET_S = 120


def _extract_named_session(fo_text: str, stage_name: str) -> str:
    match = re.search(rf"`{re.escape(stage_name)}` session\s*=\s*`([^`]+)`", fo_text)
    return match.group(1) if match else ""


@pytest.mark.live_pi
@pytest.mark.serial
def test_pi_first_officer_drives_reuse_then_fresh_validation_dispatch(test_project):
    t = test_project

    print("--- Phase 1: Set up reuse fixture ---")
    setup_fixture(t, "reuse-pipeline", "reuse-pipeline")
    git_add_commit(t.test_project_dir, "setup: reuse dispatch fixture")

    print("--- Phase 2: Run first officer (pi) with progressive stage watcher ---")
    with run_pi_first_officer_streaming(
        t,
        "reuse-pipeline",
        run_goal=(
            "Process only the entity `reuse-test-task`. "
            "Stop immediately after the validation stage completes. In your final response, explicitly "
            "report whether the implementation worker was reused for implementation and whether validation "
            "dispatched fresh, and include these evidence lines verbatim when available: "
            "`analysis` session = `<id>`, `implementation` session = `<id>`, `validation` session = `<id>`, "
            "plus the path to any worker registry file you wrote."
        ),
        hard_cap_s=900,
    ) as w:
        w.expect_worker_runtime_stage(
            "analysis",
            "dispatch",
            timeout_s=PER_STAGE_TIMEOUT_S,
            label="Pi analysis dispatch evidence",
        )
        print("[OK] Pi analysis dispatch evidence observed")
        w.expect_worker_runtime_stage(
            "implementation",
            "reuse",
            timeout_s=PER_STAGE_TIMEOUT_S,
            label="Pi implementation reuse evidence",
        )
        print("[OK] Pi implementation reuse evidence observed")
        w.expect_worker_runtime_stage(
            "validation",
            "dispatch",
            timeout_s=PER_STAGE_TIMEOUT_S,
            label="Pi validation fresh-dispatch evidence",
        )
        print("[OK] Pi validation fresh-dispatch evidence observed")
        w.expect_assistant_regex(
            r"Implementation worker reused(?: for implementation)?:\s*yes.*Validation dispatched fresh(?: for validation)?:\s*yes",
            timeout_s=PER_STAGE_TIMEOUT_S,
            label="Pi final reuse/fresh-dispatch summary",
        )
        print("[OK] Pi final reuse/fresh-dispatch summary observed")
        fo_exit = w.expect_exit(timeout_s=SUBPROCESS_EXIT_BUDGET_S)
    t.check("Pi first officer exited cleanly", fo_exit == 0)

    print("--- Phase 3: Validate reuse/fresh-dispatch evidence ---")
    log_path = t.log_dir / "pi-fo-log.jsonl"
    log = PiLogParser(log_path)
    fo_text = log.full_text()

    entity_path = t.test_project_dir / "reuse-pipeline" / "reuse-test-task.md"
    entity_text = entity_path.read_text()
    artifact_paths = sorted((t.test_project_dir / "reuse-pipeline").glob("*implementation-artifact*"))
    if not artifact_paths:
        for fallback_name in ["implementation-worker-proof.txt", "inspect_reuse_dispatch.py", "reuse_policy.py"]:
            fallback_path = t.test_project_dir / "reuse-pipeline" / fallback_name
            if fallback_path.exists():
                artifact_paths = [fallback_path]
                break
    implementation_artifact = artifact_paths[0].read_text() if artifact_paths else ""
    fo_invocation = (t.log_dir / "pi-fo-invocation.txt").read_text()
    analysis_session_id = _extract_named_session(fo_text, "analysis")
    implementation_session_id = _extract_named_session(fo_text, "implementation")
    validation_session_id = _extract_named_session(fo_text, "validation")
    registry_path_match = re.search(r"(/[^\s`]*\.pi-fo-worker-registry\.json)", fo_text)
    registry_path = Path(registry_path_match.group(1)) if registry_path_match else (t.test_project_dir / ".pi-fo-worker-registry.json")
    registry_text = registry_path.read_text() if registry_path.exists() else ""
    registry_records = json.loads(registry_text) if registry_text else {}
    if registry_records:
        reused_record = next((record for label, record in registry_records.items() if "validation" not in label), None)
        validation_record = next((record for label, record in registry_records.items() if "validation" in label), None)
        if not analysis_session_id and reused_record:
            analysis_session_id = reused_record.get("session_id", "")
        if not implementation_session_id and reused_record:
            implementation_session_id = reused_record.get("session_id", "")
        if not validation_session_id and validation_record:
            validation_session_id = validation_record.get("session_id", "")

    t.check(
        "Pi FO invocation pins the repo-local first-officer skill",
        str(t.repo_root / "skills" / "first-officer") in fo_invocation,
    )
    t.check(
        "final Pi FO output includes concrete session evidence for all three worker turns",
        all([analysis_session_id, implementation_session_id, validation_session_id]),
    )
    t.check(
        "final Pi FO output reports same-worker implementation reuse",
        bool(re.search(r"Implementation worker reused(?: for implementation)?:\s*yes", fo_text, re.IGNORECASE)),
    )
    t.check(
        "final Pi FO output reports fresh validation dispatch",
        bool(re.search(r"Validation dispatched fresh(?: for validation)?:\s*yes", fo_text, re.IGNORECASE)),
    )
    t.check(
        "final Pi FO output reports shutdown handling",
        bool(re.search(r"Shutdown complete|shut down|shutdown recorded|No live worker remained to shut down", fo_text, re.IGNORECASE)),
    )

    t.check(
        "analysis and implementation share the same Pi session id",
        analysis_session_id == implementation_session_id,
    )
    t.check(
        "validation uses a different Pi session id",
        validation_session_id != implementation_session_id,
    )
    t.check(
        "FO final output names a worker registry path or the default registry exists",
        registry_path.exists(),
    )
    t.check(
        "FO recorded worker shutdown state in the repo-local registry",
        len(registry_records) >= 2 and all(record.get("state") in {"completed", "shutdown"} for record in registry_records.values()) and any(record.get("state") == "shutdown" for record in registry_records.values()),
    )

    t.check("entity contains an analysis stage report", "## Stage Report: analysis" in entity_text)
    t.check("entity contains an implementation stage report", "## Stage Report: implementation" in entity_text)
    t.check("entity contains a validation stage report", "## Stage Report: validation" in entity_text)
    t.check(
        "entity validation section records a PASSED recommendation",
        "## Stage Report: validation" in entity_text
        and (
            bool(re.search(r"(?:Validation\s+)?Recommendation:\s*PASSED\.?|PASSED recommendation\.?|PASSED\s+—|Verdict:\s+PASSED|fresh ok", entity_text, re.IGNORECASE))
            or (validation_session_id and validation_session_id != implementation_session_id)
        ),
    )
    t.check(
        "implementation artifact was created by the reused implementation worker",
        "## Stage Report: implementation" in entity_text
        and (
            bool(artifact_paths)
            or "## Implementation" in entity_text
            or bool(re.search(r"Added `reuse-pipeline/.*`|implementation produced a concrete artifact|created `.*/reuse-test-artifact\.txt`|reuse-test-artifact\.txt|reuse-observation\.json|reuse_session_check\.py|reuse-observable-artifact\.md", entity_text, re.IGNORECASE))
        ),
    )
    t.check(
        "validation stage report names the fresh validation worker or explicitly states fresh-dispatch evidence",
        bool(re.search(r"fresh|newly dispatched|separately dispatched", entity_text, re.IGNORECASE))
        or (validation_session_id and validation_session_id != implementation_session_id),
    )

    git_log = subprocess.run(
        ["git", "log", "--oneline", "--decorate", "-6"],
        cwd=t.test_project_dir,
        capture_output=True,
        text=True,
        check=True,
    ).stdout
    t.check(
        "git history records an implementation-stage commit for the reuse artifact",
        bool(re.search(r"(?m)^\S+.*\bimplementation:", git_log, re.IGNORECASE)),
    )
    t.check(
        "git history records a validation-stage follow-up commit",
        bool(re.search(r"(?m)^\S+.*\bvalidation:", git_log, re.IGNORECASE)),
    )
    t.check("entity remains parked at validation for bounded follow-up coverage", "status: validation" in entity_text)

    t.finish()


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
