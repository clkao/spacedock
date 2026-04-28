#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live proof that Pi workers can launch in the background with dedicated log sinks and session handles.

from __future__ import annotations

from pathlib import Path
import subprocess
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from pi_session_registry import PiSessionRegistry  # noqa: E402
from pi_worker_runtime import PiWorkerRuntime  # noqa: E402
from test_lib import build_pi_ensign_invocation_prompt  # noqa: E402


@pytest.mark.live_pi
@pytest.mark.serial
def test_pi_background_worker_launch_keeps_output_in_log_and_returns_handle(test_project):
    t = test_project
    entity_path = t.test_project_dir / "task-218-bg.md"
    entity_path.write_text(
        "---\n"
        "id: 218\n"
        "title: Pi background worker launch\n"
        "status: implementation\n"
        "---\n\n"
        "Initial body.\n"
    )
    subprocess.run(["git", "add", str(entity_path.name)], cwd=t.test_project_dir, check=True)
    subprocess.run(["git", "commit", "-m", "setup: add background entity"], cwd=t.test_project_dir, check=True)

    runtime = PiWorkerRuntime(
        PiSessionRegistry(t.test_dir / "pi-bg-workers.json"),
        t.test_dir / "pi-bg-sessions",
    )

    prompt = build_pi_ensign_invocation_prompt(
        t.test_project_dir,
        entity_path,
        "implementation",
        "Use bash to run `python3 -c \"import time; time.sleep(2)\"` before editing the entity. Then append the line `Background launch note: log-sink path confirmed.` to the entity body and append the implementation stage report using the checklist below.",
        None,
        [
            "Append the background-launch note to the entity body",
            "Append the implementation stage report",
            "Commit the work",
        ],
        local_skill_path=t.repo_root / "skills" / "ensign",
        local_plugin_root=t.repo_root,
        worker_label="218-implementation/Ensign",
    )

    record, process = runtime.dispatch_background(
        t,
        worker_label="218-implementation/Ensign",
        prompt=prompt,
        cwd=t.test_project_dir,
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        no_context_files=True,
        log_name="pi-bg-dispatch.jsonl",
    )
    t.check("background dispatch returned an active worker record", record.state == "active")
    t.check("background dispatch returned a stable session id immediately", bool(record.session_id))
    t.check("background dispatch wrote to a dedicated log file", process.log_path.is_file())

    completion = runtime.collect_background_completion(
        "218-implementation/Ensign",
        process,
        timeout_s=180,
        completion_epoch=record.completion_epoch,
    )
    t.check("background completion is current", runtime.completion_is_current(record.worker_label, completion))
    updated_text = entity_path.read_text()
    t.check(
        "background worker applied the entity change",
        "Background launch note: log-sink path confirmed." in updated_text,
    )
    t.check(
        "background worker appended the stage report",
        "## Stage Report: implementation" in updated_text,
    )

    t.finish()
