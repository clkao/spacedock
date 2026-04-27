#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Unit coverage for thin Pi worker runtime dispatch/reuse/shutdown bookkeeping.

from __future__ import annotations

from pathlib import Path
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from pi_session_registry import PiSessionRegistry  # noqa: E402
from pi_worker_runtime import PiWorkerRuntime  # noqa: E402
from test_lib import TestRunner, create_test_project  # noqa: E402


class _FakePiInvoker:
    def __init__(self, runner: TestRunner, session_dir: Path):
        self.runner = runner
        self.session_dir = Path(session_dir)
        self.calls: list[dict] = []
        self.turn = 0
        self.session_id = "pi-session-123"
        self.session_file = self.session_dir / f"2026-04-27T23-41-25-454Z_{self.session_id}.jsonl"

    def __call__(self, runner: TestRunner, prompt: str, **kwargs) -> int:
        assert runner is self.runner
        self.turn += 1
        self.calls.append({"prompt": prompt, **kwargs})
        log_path = runner.log_dir / kwargs["log_name"]
        self.session_dir.mkdir(parents=True, exist_ok=True)
        self.session_file.write_text((self.session_file.read_text() if self.session_file.exists() else "") + f"turn {self.turn}\n")
        reply = "INITIAL_OK" if self.turn == 1 else "FOLLOWUP_OK"
        log_path.write_text(
            f'{{"type":"session","id":"{self.session_id}"}}\n'
            f'{{"type":"message_end","message":{{"role":"assistant","content":[{{"type":"text","text":"{reply}"}}]}}}}\n'
        )
        return 0


def test_pi_worker_runtime_tracks_dispatch_reuse_and_shutdown(tmp_path):
    runner = TestRunner("pi worker runtime", keep_test_dir=False)
    create_test_project(runner)

    session_dir = tmp_path / "pi-sessions"
    invoker = _FakePiInvoker(runner, session_dir)
    runtime = PiWorkerRuntime(
        PiSessionRegistry(tmp_path / "pi-workers.json"),
        session_dir,
        invoke_pi=invoker,
    )

    initial = runtime.dispatch(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the first task.",
        cwd=runner.test_project_dir,
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        no_context_files=True,
        log_name="dispatch.jsonl",
    )
    assert initial.final_text == "INITIAL_OK"
    assert initial.completion_epoch == 0
    assert runtime.completion_is_current("218-implementation/Ensign", initial) is True

    follow_up = runtime.reuse(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the second task.",
        stage_name="validation",
        no_context_files=True,
        log_name="reuse.jsonl",
    )
    assert follow_up.final_text == "FOLLOWUP_OK"
    assert follow_up.session_id == initial.session_id
    assert follow_up.session_file == initial.session_file
    assert follow_up.completion_epoch == 1
    assert runtime.completion_is_current("218-implementation/Ensign", follow_up) is True
    assert runtime.completion_is_current("218-implementation/Ensign", initial) is False

    assert invoker.calls[0].get("session") is None
    assert invoker.calls[1].get("session") == initial.session_id

    updated = runtime.shutdown("218-implementation/Ensign")
    assert updated.state == "shutdown"
    with pytest.raises(RuntimeError, match="not routable"):
        runtime.reuse(
            runner,
            worker_label="218-implementation/Ensign",
            prompt="This should not run.",
            log_name="after-shutdown.jsonl",
        )


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
