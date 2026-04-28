#!/usr/bin/env -S uv run --with pytest python
# /// script
# requires-python = ">=3.10"
# ///
# ABOUTME: Unit coverage for Pi session-backed worker registry bookkeeping.

from __future__ import annotations

from pathlib import Path
import sys

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
from pi_session_registry import PiSessionRegistry, WorkerSessionRecord  # noqa: E402


def test_registry_round_trips_worker_metadata(tmp_path):
    registry = PiSessionRegistry(tmp_path / "pi-workers.json")
    record = WorkerSessionRecord(
        worker_label="218-implementation/Ensign",
        dispatch_agent_id="spacedock:ensign",
        worker_key="spacedock-ensign",
        session_id="pi-session-123",
        session_file=str(tmp_path / "sessions" / "pi-session-123.jsonl"),
        cwd="/tmp/worktree",
        entity_slug="pi-runtime-compatibility-baseline",
        stage_name="implementation",
        state="active",
        completion_epoch=0,
    )

    registry.upsert(record)
    loaded = registry.get("218-implementation/Ensign")

    assert loaded is not None
    assert loaded.session_id == "pi-session-123"
    assert loaded.cwd == "/tmp/worktree"
    assert loaded.state == "active"
    assert loaded.completion_epoch == 0


def test_mark_active_again_bumps_completion_epoch_for_reuse(tmp_path):
    registry = PiSessionRegistry(tmp_path / "pi-workers.json")
    registry.upsert(
        WorkerSessionRecord(
            worker_label="218-implementation/Ensign",
            dispatch_agent_id="spacedock:ensign",
            worker_key="spacedock-ensign",
            session_id="pi-session-123",
            session_file=str(tmp_path / "sessions" / "pi-session-123.jsonl"),
            cwd="/tmp/worktree",
            entity_slug="pi-runtime-compatibility-baseline",
            stage_name="implementation",
            state="completed",
            completion_epoch=1,
        )
    )

    updated = registry.mark_active_again("218-implementation/Ensign")

    assert updated.state == "active"
    assert updated.completion_epoch == 2


def test_active_worker_is_not_routable_and_cannot_be_marked_active_again(tmp_path):
    registry = PiSessionRegistry(tmp_path / "pi-workers.json")
    registry.upsert(
        WorkerSessionRecord(
            worker_label="218-implementation/Ensign",
            dispatch_agent_id="spacedock:ensign",
            worker_key="spacedock-ensign",
            session_id="pi-session-123",
            session_file=str(tmp_path / "sessions" / "pi-session-123.jsonl"),
            cwd="/tmp/worktree",
            entity_slug="pi-runtime-compatibility-baseline",
            stage_name="implementation",
            state="active",
            completion_epoch=1,
        )
    )

    assert registry.routable("218-implementation/Ensign") is False
    with pytest.raises(RuntimeError, match="already active"):
        registry.mark_active_again("218-implementation/Ensign")



def test_mark_shutdown_makes_worker_unroutable(tmp_path):
    registry = PiSessionRegistry(tmp_path / "pi-workers.json")
    registry.upsert(
        WorkerSessionRecord(
            worker_label="218-implementation/Ensign",
            dispatch_agent_id="spacedock:ensign",
            worker_key="spacedock-ensign",
            session_id="pi-session-123",
            session_file=str(tmp_path / "sessions" / "pi-session-123.jsonl"),
            cwd="/tmp/worktree",
            entity_slug="pi-runtime-compatibility-baseline",
            stage_name="implementation",
            state="completed",
            completion_epoch=1,
        )
    )

    updated = registry.mark_shutdown("218-implementation/Ensign")

    assert updated.state == "shutdown"
    assert registry.routable("218-implementation/Ensign") is False



def test_registry_save_is_atomic_and_load_reports_corruption(tmp_path):
    registry = PiSessionRegistry(tmp_path / "pi-workers.json")
    registry.upsert(
        WorkerSessionRecord(
            worker_label="218-implementation/Ensign",
            dispatch_agent_id="spacedock:ensign",
            worker_key="spacedock-ensign",
            session_id="pi-session-123",
            session_file=str(tmp_path / "sessions" / "pi-session-123.jsonl"),
            cwd="/tmp/worktree",
            entity_slug="pi-runtime-compatibility-baseline",
            stage_name="implementation",
            state="completed",
            completion_epoch=1,
        )
    )
    assert list(tmp_path.glob("*.tmp-*")) == []

    registry.path.write_text("{broken")
    with pytest.raises(RuntimeError, match="corrupt"):
        registry.get("218-implementation/Ensign")


if __name__ == "__main__":
    raise SystemExit(pytest.main([__file__]))
