from __future__ import annotations

"""Thin Spacedock mapping over Pi's built-in session persistence.

Pi session ids / session files are the canonical worker handle. This module does
not try to replace Pi's own session system or emulate Claude-style team state.
It only records the workflow-specific association between an FO-owned worker
label and the Pi session that currently backs it, plus the minimum extra state
Spacedock needs for reuse/shutdown bookkeeping.
"""

import json
from dataclasses import asdict, dataclass
from pathlib import Path


@dataclass
class WorkerSessionRecord:
    worker_label: str
    dispatch_agent_id: str
    worker_key: str
    session_id: str
    session_file: str
    cwd: str
    entity_slug: str
    stage_name: str
    state: str
    completion_epoch: int


class PiSessionRegistry:
    """Persist a minimal worker-label -> Pi-session mapping for Spacedock."""

    def __init__(self, path: Path):
        self.path = Path(path)

    def _load(self) -> dict[str, WorkerSessionRecord]:
        if not self.path.exists():
            return {}
        data = json.loads(self.path.read_text())
        return {
            worker_label: WorkerSessionRecord(**record)
            for worker_label, record in data.items()
        }

    def _save(self, records: dict[str, WorkerSessionRecord]) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            worker_label: asdict(record)
            for worker_label, record in records.items()
        }
        self.path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n")

    def upsert(self, record: WorkerSessionRecord) -> WorkerSessionRecord:
        records = self._load()
        records[record.worker_label] = record
        self._save(records)
        return record

    def get(self, worker_label: str) -> WorkerSessionRecord | None:
        return self._load().get(worker_label)

    def mark_active_again(self, worker_label: str) -> WorkerSessionRecord:
        records = self._load()
        record = records[worker_label]
        record.state = "active"
        record.completion_epoch += 1
        self._save(records)
        return record

    def mark_shutdown(self, worker_label: str) -> WorkerSessionRecord:
        records = self._load()
        record = records[worker_label]
        record.state = "shutdown"
        self._save(records)
        return record

    def routable(self, worker_label: str) -> bool:
        record = self.get(worker_label)
        return bool(record is not None and record.state != "shutdown")
