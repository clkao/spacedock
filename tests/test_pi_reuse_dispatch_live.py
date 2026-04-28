#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live proof that the Pi first officer can drive analysis->implementation reuse and fresh validation dispatch.
# ABOUTME: Verifies the FO uses repo-local Pi skills plus session-backed reuse semantics exposed through the Pi worker runtime helper path.

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from test_lib import PiLogParser, git_add_commit, run_pi_first_officer, setup_fixture  # noqa: E402


def _extract_named_session(fo_text: str, stage_name: str) -> str:
    match = re.search(rf"`{re.escape(stage_name)}` session\s*=\s*`([^`]+)`", fo_text)
    if not match:
        raise AssertionError(f"Pi FO output did not contain a `{stage_name}` session line")
    return match.group(1)


@pytest.mark.live_pi
@pytest.mark.serial
def test_pi_first_officer_drives_reuse_then_fresh_validation_dispatch(test_project):
    t = test_project

    print("--- Phase 1: Set up reuse fixture ---")
    setup_fixture(t, "reuse-pipeline", "reuse-pipeline")
    git_add_commit(t.test_project_dir, "setup: reuse dispatch fixture")

    print("--- Phase 2: Run first officer (pi) ---")
    fo_exit = run_pi_first_officer(
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
        timeout_s=600,
    )
    t.check("Pi first officer exited cleanly", fo_exit == 0)

    print("--- Phase 3: Validate reuse/fresh-dispatch evidence ---")
    log_path = t.log_dir / "pi-fo-log.jsonl"
    log = PiLogParser(log_path)
    fo_text = log.full_text()

    entity_path = t.test_project_dir / "reuse-pipeline" / "reuse-test-task.md"
    entity_text = entity_path.read_text()
    artifact_paths = sorted((t.test_project_dir / "reuse-pipeline").glob("*implementation-artifact*"))
    if not artifact_paths:
        proof_path = t.test_project_dir / "reuse-pipeline" / "implementation-worker-proof.txt"
        if proof_path.exists():
            artifact_paths = [proof_path]
    implementation_artifact = artifact_paths[0].read_text() if artifact_paths else ""
    fo_invocation = (t.log_dir / "pi-fo-invocation.txt").read_text()
    analysis_session_id = _extract_named_session(fo_text, "analysis")
    implementation_session_id = _extract_named_session(fo_text, "implementation")
    validation_session_id = _extract_named_session(fo_text, "validation")
    registry_path_match = re.search(r"(/[^\s`]*\.pi-fo-worker-registry\.json)", fo_text)
    registry_path = Path(registry_path_match.group(1)) if registry_path_match else (t.test_project_dir / ".pi-fo-worker-registry.json")
    registry_text = registry_path.read_text() if registry_path.exists() else ""

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
        bool(re.search(r'"001-mainline/Ensign".*"state": "shutdown"', registry_text, re.DOTALL))
        and bool(re.search(r'"001-validation/Ensign".*"state": "shutdown"', registry_text, re.DOTALL)),
    )

    t.check("entity contains an analysis stage report", "## Stage Report: analysis" in entity_text)
    t.check("entity contains an implementation stage report", "## Stage Report: implementation" in entity_text)
    t.check("entity contains a validation stage report", "## Stage Report: validation" in entity_text)
    t.check(
        "entity validation section records a PASSED recommendation",
        bool(re.search(r"(?:Validation\s+)?Recommendation:\s*PASSED\.?", entity_text, re.IGNORECASE)),
    )
    t.check(
        "implementation artifact file was created by the reused implementation worker",
        bool(artifact_paths) and "001-mainline/Ensign" in implementation_artifact,
    )
    t.check(
        "validation stage report names the fresh validation worker label",
        "001-validation/Ensign" in entity_text,
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
        bool(re.search(r"implementation: .*reuse.*artifact|add reuse proof artifact|implementation:\s+add worker proof artifact", git_log, re.IGNORECASE)),
    )
    t.check(
        "git history records a validation-stage follow-up commit",
        bool(re.search(r"validation: .*PASSED|validate reuse test task|validation:\s+verify reuse evidence", git_log, re.IGNORECASE)),
    )
    t.check("entity remains parked at validation for bounded follow-up coverage", "status: validation" in entity_text)

    t.finish()


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
