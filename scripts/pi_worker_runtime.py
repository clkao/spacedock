from __future__ import annotations

"""Thin Pi worker adapter for first-slice Spacedock reuse semantics.

This module intentionally stays close to Pi's native session model: one Pi
session per worker, reopened by session id or session file for routed follow-up.
It layers only the minimum workflow bookkeeping needed to map an FO-owned worker
label to that Pi session and to reject stale pre-reuse completion evidence.
"""

import argparse
import json
import sys
from dataclasses import asdict, dataclass
from pathlib import Path
from types import SimpleNamespace
from typing import Callable

from pi_session_registry import PiSessionRegistry, WorkerSessionRecord
from test_lib import PiBackgroundProcess, PiLogParser, PiLogWatcher, launch_pi_ensign_background, run_pi_ensign


@dataclass
class PiWorkerCompletion:
    worker_label: str
    session_id: str
    session_file: str
    completion_epoch: int
    final_text: str


class PiWorkerRuntime:
    """Dispatch and reuse Pi workers via reopened Pi sessions."""

    def __init__(
        self,
        registry: PiSessionRegistry,
        session_dir: Path | str,
        invoke_pi: Callable[..., int] = run_pi_ensign,
        launch_pi: Callable[..., PiBackgroundProcess] = launch_pi_ensign_background,
    ):
        self.registry = registry
        self.session_dir = Path(session_dir)
        self.invoke_pi = invoke_pi
        self.launch_pi = launch_pi
        self._active_processes: dict[str, PiBackgroundProcess] = {}

    def dispatch(
        self,
        runner,
        *,
        worker_label: str,
        prompt: str,
        cwd: Path | str,
        entity_slug: str,
        stage_name: str,
        dispatch_agent_id: str = "spacedock:ensign",
        worker_key: str | None = None,
        log_name: str = "pi-worker-dispatch.jsonl",
        timeout_s: int = 120,
        skill_paths: list[Path | str] | None = None,
        no_context_files: bool = False,
    ) -> PiWorkerCompletion:
        exit_code = self.invoke_pi(
            runner,
            prompt,
            session_dir=self.session_dir,
            log_name=log_name,
            timeout_s=timeout_s,
            cwd=cwd,
            skill_paths=skill_paths,
            no_context_files=no_context_files,
        )
        if exit_code != 0:
            raise RuntimeError(f"Pi worker dispatch failed for {worker_label}: exit={exit_code}")

        completion = self._parse_completion(runner.log_dir / log_name, worker_label, completion_epoch=0)
        self.registry.upsert(
            WorkerSessionRecord(
                worker_label=worker_label,
                dispatch_agent_id=dispatch_agent_id,
                worker_key=worker_key or worker_label,
                session_id=completion.session_id,
                session_file=completion.session_file,
                cwd=str(Path(cwd)),
                entity_slug=entity_slug,
                stage_name=stage_name,
                state="completed",
                completion_epoch=completion.completion_epoch,
            )
        )
        return completion

    def reuse(
        self,
        runner,
        *,
        worker_label: str,
        prompt: str,
        stage_name: str | None = None,
        log_name: str = "pi-worker-reuse.jsonl",
        timeout_s: int = 120,
        skill_paths: list[Path | str] | None = None,
        no_context_files: bool = False,
    ) -> PiWorkerCompletion:
        if not self.registry.routable(worker_label):
            raise RuntimeError(f"Pi worker {worker_label} is not routable")

        record = self.registry.mark_active_again(worker_label)
        exit_code = self.invoke_pi(
            runner,
            prompt,
            session_dir=self.session_dir,
            session=self._reopen_handle(record),
            log_name=log_name,
            timeout_s=timeout_s,
            cwd=record.cwd,
            skill_paths=skill_paths,
            no_context_files=no_context_files,
        )
        if exit_code != 0:
            raise RuntimeError(f"Pi worker reuse failed for {worker_label}: exit={exit_code}")

        completion = self._parse_completion(
            runner.log_dir / log_name,
            worker_label,
            completion_epoch=record.completion_epoch,
        )
        self.registry.upsert(
            WorkerSessionRecord(
                worker_label=record.worker_label,
                dispatch_agent_id=record.dispatch_agent_id,
                worker_key=record.worker_key,
                session_id=completion.session_id,
                session_file=completion.session_file,
                cwd=record.cwd,
                entity_slug=record.entity_slug,
                stage_name=stage_name or record.stage_name,
                state="completed",
                completion_epoch=completion.completion_epoch,
            )
        )
        return completion

    def dispatch_background(
        self,
        runner,
        *,
        worker_label: str,
        prompt: str,
        cwd: Path | str,
        entity_slug: str,
        stage_name: str,
        dispatch_agent_id: str = "spacedock:ensign",
        worker_key: str | None = None,
        log_name: str = "pi-worker-dispatch.jsonl",
        skill_paths: list[Path | str] | None = None,
        no_context_files: bool = False,
    ) -> tuple[WorkerSessionRecord, PiBackgroundProcess]:
        process = self.launch_pi(
            runner,
            prompt,
            session_dir=self.session_dir,
            log_name=log_name,
            cwd=cwd,
            skill_paths=skill_paths,
            no_context_files=no_context_files,
        )
        watcher = PiLogWatcher(process)
        session_id = watcher.wait_for_session_id(timeout_s=10)
        session_file = watcher.wait_for_session_file(timeout_s=10)
        record = WorkerSessionRecord(
            worker_label=worker_label,
            dispatch_agent_id=dispatch_agent_id,
            worker_key=worker_key or worker_label,
            session_id=session_id,
            session_file=str(session_file),
            cwd=str(Path(cwd)),
            entity_slug=entity_slug,
            stage_name=stage_name,
            state="active",
            completion_epoch=0,
        )
        self.registry.upsert(record)
        self._active_processes[worker_label] = process
        return record, process

    def reuse_background(
        self,
        runner,
        *,
        worker_label: str,
        prompt: str,
        log_name: str = "pi-worker-reuse.jsonl",
        skill_paths: list[Path | str] | None = None,
        no_context_files: bool = False,
    ) -> tuple[WorkerSessionRecord, PiBackgroundProcess]:
        if not self.registry.routable(worker_label):
            raise RuntimeError(f"Pi worker {worker_label} is not routable")
        record = self.registry.mark_active_again(worker_label)
        process = self.launch_pi(
            runner,
            prompt,
            session_dir=self.session_dir,
            session=self._reopen_handle(record),
            log_name=log_name,
            cwd=record.cwd,
            skill_paths=skill_paths,
            no_context_files=no_context_files,
        )
        self._active_processes[worker_label] = process
        return record, process

    def collect_background_completion(
        self,
        worker_label: str,
        process: PiBackgroundProcess,
        *,
        timeout_s: float,
        completion_epoch: int,
        stage_name: str | None = None,
    ) -> PiWorkerCompletion:
        watcher = PiLogWatcher(process)
        exit_code = watcher.wait_for_exit(timeout_s=timeout_s)
        if exit_code != 0:
            raise RuntimeError(f"Pi worker background completion failed for {worker_label}: exit={exit_code}")
        completion = self._parse_completion(process.log_path, worker_label, completion_epoch=completion_epoch)
        record = self.registry.get(worker_label)
        if record is None:
            raise RuntimeError(f"Pi worker {worker_label} missing from registry during background completion")
        self.registry.upsert(
            WorkerSessionRecord(
                worker_label=record.worker_label,
                dispatch_agent_id=record.dispatch_agent_id,
                worker_key=record.worker_key,
                session_id=completion.session_id,
                session_file=completion.session_file,
                cwd=record.cwd,
                entity_slug=record.entity_slug,
                stage_name=stage_name or record.stage_name,
                state="completed",
                completion_epoch=completion.completion_epoch,
            )
        )
        self._active_processes.pop(worker_label, None)
        return completion

    def shutdown(self, worker_label: str, *, timeout_s: float = 5) -> WorkerSessionRecord:
        process = self._active_processes.pop(worker_label, None)
        if process is not None and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=timeout_s)
            except Exception:
                process.proc.kill()
                process.wait(timeout=timeout_s)
            finally:
                process.close()
        elif process is not None:
            process.close()
        return self.registry.mark_shutdown(worker_label)

    def completion_is_current(self, worker_label: str, completion: PiWorkerCompletion) -> bool:
        record = self.registry.get(worker_label)
        if record is None:
            return False
        return (
            record.state == "completed"
            and record.session_id == completion.session_id
            and record.completion_epoch == completion.completion_epoch
        )

    def _reopen_handle(self, record: WorkerSessionRecord) -> str:
        return record.session_id or record.session_file

    def _parse_completion(
        self,
        log_path: Path,
        worker_label: str,
        *,
        completion_epoch: int,
    ) -> PiWorkerCompletion:
        parser = PiLogParser(log_path)
        session_id = parser.session_id()
        session_file = parser.session_file(self.session_dir)
        if not session_id or session_file is None:
            raise RuntimeError(f"Pi worker log {log_path} did not expose a session handle")
        return PiWorkerCompletion(
            worker_label=worker_label,
            session_id=session_id,
            session_file=str(session_file),
            completion_epoch=completion_epoch,
            final_text=parser.full_text().strip(),
        )


def _build_runtime(registry: Path, session_dir: Path) -> PiWorkerRuntime:
    return PiWorkerRuntime(PiSessionRegistry(registry), session_dir)


def _load_prompt(prompt_file: Path) -> str:
    return Path(prompt_file).read_text()


def _cli_runner(log_dir: Path, cwd: Path) -> SimpleNamespace:
    repo_root = Path(__file__).resolve().parent.parent
    log_dir.mkdir(parents=True, exist_ok=True)
    return SimpleNamespace(log_dir=log_dir, test_project_dir=cwd, repo_root=repo_root)


def _print_json(payload: dict) -> None:
    print(json.dumps(payload, indent=2, sort_keys=True))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Pi worker runtime helper for Spacedock first-officer flows")
    subparsers = parser.add_subparsers(dest="command", required=True)

    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--registry", type=Path, required=True)
    common.add_argument("--session-dir", type=Path, required=True)
    common.add_argument("--worker-label", required=True)

    dispatch_parser = subparsers.add_parser("dispatch", parents=[common])
    dispatch_parser.add_argument("--prompt-file", type=Path, required=True)
    dispatch_parser.add_argument("--cwd", type=Path, required=True)
    dispatch_parser.add_argument("--entity-slug", required=True)
    dispatch_parser.add_argument("--stage-name", required=True)
    dispatch_parser.add_argument("--dispatch-agent-id", default="spacedock:ensign")
    dispatch_parser.add_argument("--worker-key")
    dispatch_parser.add_argument("--log-name", default="pi-worker-dispatch.jsonl")
    dispatch_parser.add_argument("--timeout-s", type=int, default=120)
    dispatch_parser.add_argument("--skill-path", action="append", default=[])
    dispatch_parser.add_argument("--no-context-files", action="store_true")

    reuse_parser = subparsers.add_parser("reuse", parents=[common])
    reuse_parser.add_argument("--prompt-file", type=Path, required=True)
    reuse_parser.add_argument("--stage-name")
    reuse_parser.add_argument("--log-name", default="pi-worker-reuse.jsonl")
    reuse_parser.add_argument("--timeout-s", type=int, default=120)
    reuse_parser.add_argument("--skill-path", action="append", default=[])
    reuse_parser.add_argument("--no-context-files", action="store_true")

    shutdown_parser = subparsers.add_parser("shutdown", parents=[common])
    shutdown_parser.add_argument("--timeout-s", type=float, default=5)

    args = parser.parse_args(argv)
    runtime = _build_runtime(args.registry, args.session_dir)

    if args.command == "dispatch":
        runner = _cli_runner(args.registry.parent, args.cwd)
        completion = runtime.dispatch(
            runner,
            worker_label=args.worker_label,
            prompt=_load_prompt(args.prompt_file),
            cwd=args.cwd,
            entity_slug=args.entity_slug,
            stage_name=args.stage_name,
            dispatch_agent_id=args.dispatch_agent_id,
            worker_key=args.worker_key,
            log_name=args.log_name,
            timeout_s=args.timeout_s,
            skill_paths=args.skill_path or None,
            no_context_files=args.no_context_files,
        )
        _print_json({"completion": asdict(completion), "record": asdict(runtime.registry.get(args.worker_label))})
        return 0

    if args.command == "reuse":
        record = runtime.registry.get(args.worker_label)
        if record is None:
            raise RuntimeError(f"Pi worker {args.worker_label} missing from registry")
        runner = _cli_runner(args.registry.parent, Path(record.cwd))
        completion = runtime.reuse(
            runner,
            worker_label=args.worker_label,
            prompt=_load_prompt(args.prompt_file),
            stage_name=args.stage_name,
            log_name=args.log_name,
            timeout_s=args.timeout_s,
            skill_paths=args.skill_path or None,
            no_context_files=args.no_context_files,
        )
        _print_json({"completion": asdict(completion), "record": asdict(runtime.registry.get(args.worker_label))})
        return 0

    record = runtime.shutdown(args.worker_label, timeout_s=args.timeout_s)
    _print_json({"record": asdict(record)})
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
