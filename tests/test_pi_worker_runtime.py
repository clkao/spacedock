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
from pi_session_registry import PiSessionRegistry, WorkerSessionRecord  # noqa: E402
from pi_worker_runtime import PiWorkerRuntime  # noqa: E402
from test_lib import PiBackgroundProcess, TestRunner, create_test_project  # noqa: E402


class _FakeBackgroundLauncher:
    def __init__(self, runner: TestRunner, session_dir: Path):
        self.runner = runner
        self.session_dir = Path(session_dir)
        self.calls: list[dict] = []
        self.turn = 0
        self.session_id = "pi-session-123"
        self.session_file = self.session_dir / f"2026-04-27T23-41-25-454Z_{self.session_id}.jsonl"

    def __call__(self, runner: TestRunner, prompt: str, **kwargs):
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

        class _Proc:
            def __init__(self):
                self.returncode = 0
                self.kill_called = False

            def poll(self):
                return self.returncode

            def wait(self, timeout=None):
                return self.returncode

            def terminate(self):
                self.returncode = -15

            def kill(self):
                self.kill_called = True
                self.returncode = -9

        class _LogFile:
            closed = False
            def close(self):
                self.closed = True

        return PiBackgroundProcess(proc=_Proc(), log_path=log_path, session_dir=self.session_dir, log_file=_LogFile())


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


def test_pi_worker_runtime_background_launch_returns_handle_and_collects_completion(tmp_path):
    runner = TestRunner("pi worker runtime background", keep_test_dir=False)
    create_test_project(runner)

    session_dir = tmp_path / "pi-sessions"
    launcher = _FakeBackgroundLauncher(runner, session_dir)
    runtime = PiWorkerRuntime(
        PiSessionRegistry(tmp_path / "pi-workers.json"),
        session_dir,
        launch_pi=launcher,
    )

    record, process = runtime.dispatch_background(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the first task.",
        cwd=runner.test_project_dir,
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        no_context_files=True,
        log_name="dispatch-bg.jsonl",
    )
    assert record.state == "active"
    assert record.session_id == "pi-session-123"
    completion = runtime.collect_background_completion(
        "218-implementation/Ensign",
        process,
        timeout_s=1,
        completion_epoch=record.completion_epoch,
    )
    assert completion.final_text == "INITIAL_OK"
    assert runtime.completion_is_current("218-implementation/Ensign", completion) is True

    reuse_record, reuse_process = runtime.reuse_background(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the second task.",
        log_name="reuse-bg.jsonl",
        no_context_files=True,
    )
    assert reuse_record.completion_epoch == 1
    follow_up = runtime.collect_background_completion(
        "218-implementation/Ensign",
        reuse_process,
        timeout_s=1,
        completion_epoch=reuse_record.completion_epoch,
        stage_name="validation",
    )
    assert follow_up.final_text == "FOLLOWUP_OK"
    assert runtime.completion_is_current("218-implementation/Ensign", follow_up) is True



def test_pi_worker_runtime_prevents_background_reuse_while_worker_is_still_active(tmp_path):
    runner = TestRunner("pi worker runtime active reuse guard", keep_test_dir=False)
    create_test_project(runner)

    session_dir = tmp_path / "pi-sessions"
    launcher = _FakeBackgroundLauncher(runner, session_dir)
    runtime = PiWorkerRuntime(
        PiSessionRegistry(tmp_path / "pi-workers.json"),
        session_dir,
        launch_pi=launcher,
    )

    runtime.dispatch_background(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the first task.",
        cwd=runner.test_project_dir,
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        no_context_files=True,
        log_name="dispatch-bg.jsonl",
    )

    with pytest.raises(RuntimeError, match="not routable"):
        runtime.reuse_background(
            runner,
            worker_label="218-implementation/Ensign",
            prompt="This must not overlap.",
            log_name="reuse-bg.jsonl",
            no_context_files=True,
        )



def test_pi_worker_runtime_reuse_falls_back_to_session_file_when_session_id_is_missing(tmp_path):
    runner = TestRunner("pi worker runtime session file fallback", keep_test_dir=False)
    create_test_project(runner)

    session_dir = tmp_path / "pi-sessions"
    invoker = _FakePiInvoker(runner, session_dir)
    runtime = PiWorkerRuntime(
        PiSessionRegistry(tmp_path / "pi-workers.json"),
        session_dir,
        invoke_pi=invoker,
    )
    runtime.registry.upsert(
        WorkerSessionRecord(
            worker_label="218-implementation/Ensign",
            dispatch_agent_id="spacedock:ensign",
            worker_key="spacedock-ensign",
            session_id="",
            session_file=str(invoker.session_file),
            cwd=str(runner.test_project_dir),
            entity_slug="pi-runtime-compatibility-baseline",
            stage_name="implementation",
            state="completed",
            completion_epoch=0,
        )
    )

    runtime.reuse(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the second task.",
        stage_name="validation",
        no_context_files=True,
        log_name="reuse.jsonl",
    )
    assert invoker.calls[-1].get("session") == str(invoker.session_file)



def test_pi_worker_runtime_shutdown_terminates_tracked_background_process(tmp_path):
    runner = TestRunner("pi worker runtime shutdown", keep_test_dir=False)
    create_test_project(runner)

    session_dir = tmp_path / "pi-sessions"
    launcher = _FakeBackgroundLauncher(runner, session_dir)
    runtime = PiWorkerRuntime(
        PiSessionRegistry(tmp_path / "pi-workers.json"),
        session_dir,
        launch_pi=launcher,
    )

    record, process = runtime.dispatch_background(
        runner,
        worker_label="218-implementation/Ensign",
        prompt="Do the first task.",
        cwd=runner.test_project_dir,
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        no_context_files=True,
        log_name="dispatch-bg.jsonl",
    )
    process.proc.returncode = None

    updated = runtime.shutdown(record.worker_label)
    assert updated.state == "shutdown"
    assert process.proc.returncode == -15
    assert process.log_file.closed is True



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
