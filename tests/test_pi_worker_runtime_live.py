#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Live proof for thin Pi worker runtime reuse, stale-completion rejection, and shutdown gating.

from __future__ import annotations

from pathlib import Path
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from pi_session_registry import PiSessionRegistry  # noqa: E402
from pi_worker_runtime import PiWorkerRuntime  # noqa: E402


@pytest.mark.live_pi
@pytest.mark.serial
def test_pi_worker_runtime_reuse_rejects_stale_completion_and_blocks_post_shutdown(test_project):
    t = test_project
    session_dir = t.test_dir / "pi-worker-runtime-sessions"
    runtime = PiWorkerRuntime(
        PiSessionRegistry(t.test_dir / "pi-worker-runtime.json"),
        session_dir,
    )

    initial = runtime.dispatch(
        t,
        worker_label="218-implementation/Ensign",
        prompt=(
            "Remember the token `PI_WORKER_RUNTIME_218`. "
            "Reply with exactly `INITIAL_WORKER_OK` and nothing else."
        ),
        cwd=t.test_project_dir,
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        no_context_files=True,
        log_name="pi-worker-dispatch.jsonl",
    )
    t.check("initial worker dispatch returned expected text", initial.final_text == "INITIAL_WORKER_OK")
    t.check("initial worker completion is current", runtime.completion_is_current(initial.worker_label, initial))

    follow_up = runtime.reuse(
        t,
        worker_label=initial.worker_label,
        prompt="Reply with only the remembered token.",
        stage_name="validation",
        no_context_files=True,
        log_name="pi-worker-reuse.jsonl",
    )
    t.check("follow-up reused the same Pi session id", follow_up.session_id == initial.session_id)
    t.check("follow-up reused the same Pi session file", follow_up.session_file == initial.session_file)
    t.check("follow-up completion epoch advanced", follow_up.completion_epoch == initial.completion_epoch + 1)
    t.check("follow-up returned new completion content", follow_up.final_text == "PI_WORKER_RUNTIME_218")
    t.check("follow-up completion is current", runtime.completion_is_current(initial.worker_label, follow_up))
    t.check("initial completion is stale after reuse", not runtime.completion_is_current(initial.worker_label, initial))

    shutdown_record = runtime.shutdown(initial.worker_label)
    t.check("shutdown marks the worker unroutable", shutdown_record.state == "shutdown")
    with pytest.raises(RuntimeError, match="not routable"):
        runtime.reuse(
            t,
            worker_label=initial.worker_label,
            prompt="Reply with only the remembered token.",
            no_context_files=True,
            log_name="pi-worker-after-shutdown.jsonl",
        )
    t.pass_("post-shutdown reuse is blocked")

    t.finish()
