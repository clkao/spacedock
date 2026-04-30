# ABOUTME: pytest wiring for live runtime flags, shared fixtures, and live-marker advisory check.
# ABOUTME: Single conftest at tests/ root — no per-subdir conftests.

from __future__ import annotations

import os
import sys
import warnings
from pathlib import Path

import pytest

_REPO_ROOT = Path(__file__).resolve().parent.parent
_SCRIPTS_DIR = _REPO_ROOT / "scripts"
if str(_SCRIPTS_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS_DIR))

from test_lib import TestRunner, create_test_project, install_agents, run_first_officer, run_codex_first_officer, run_pi_first_officer  # noqa: E402


def pytest_addoption(parser):
    parser.addoption("--runtime", action="store", default="claude",
                     choices=["claude", "codex", "pi"],
                     help="Runtime under test for live E2E (claude, codex, or pi).")
    parser.addoption("--model", action="store", default="haiku",
                     help="Model identifier for live runs (default: haiku).")
    parser.addoption("--effort", action="store", default="low",
                     help="Effort level for live runs (default: low).")
    parser.addoption("--budget", action="store", type=float, default=None,
                     help="Max budget in USD for a live run (optional).")
    parser.addoption("--team-mode", action="store", default="auto",
                     choices=["auto", "teams", "bare"],
                     help="Filter live tests by teams_mode / bare_mode markers. "
                          "'auto' resolves from CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS env "
                          "('1' or 'true' → teams, else bare).")


_LIVE_IMPORT_MARKERS = {"run_first_officer", "run_codex_first_officer", "run_pi_first_officer"}


def _resolve_team_mode(config) -> str:
    """Resolve the effective team mode: 'teams' or 'bare'."""
    value = config.getoption("--team-mode")
    if value in ("teams", "bare"):
        return value
    env = os.environ.get("CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS", "").strip().lower()
    return "teams" if env in ("1", "true") else "bare"


def pytest_collection_modifyitems(config, items):
    """Apply the team-mode filter and run the live-marker advisory check.

    - Items carrying BOTH teams_mode and bare_mode fail collection loudly.
    - Items whose team-mode marker disagrees with the resolved mode get a skip marker.
    - Advisory: warn if a module imports a live-runtime launcher
      but none of its tests carry live_claude, live_codex, or live_pi markers.
    """
    resolved_mode = _resolve_team_mode(config)

    for item in items:
        has_teams = item.get_closest_marker("teams_mode") is not None
        has_bare = item.get_closest_marker("bare_mode") is not None
        if has_teams and has_bare:
            raise pytest.UsageError(
                f"{item.nodeid}: carries both @pytest.mark.teams_mode and "
                "@pytest.mark.bare_mode — pick one. A test is pinned to one mode "
                "or left mode-agnostic (no marker)."
            )
        if has_teams and resolved_mode != "teams":
            item.add_marker(pytest.mark.skip(
                reason=f"requires teams mode; --team-mode={resolved_mode}"
            ))
        elif has_bare and resolved_mode != "bare":
            item.add_marker(pytest.mark.skip(
                reason=f"requires bare mode; --team-mode={resolved_mode}"
            ))

    by_module: dict[str, list] = {}
    for item in items:
        by_module.setdefault(item.module.__name__, []).append(item)

    for module_name, module_items in by_module.items():
        module = module_items[0].module
        source_names = set(getattr(module, "__dict__", {}).keys())
        imports_live = bool(_LIVE_IMPORT_MARKERS & source_names)
        if not imports_live:
            continue
        has_live_marker = any(
            item.get_closest_marker("live_claude") or item.get_closest_marker("live_codex") or item.get_closest_marker("live_pi")
            for item in module_items
        )
        if not has_live_marker:
            warnings.warn(
                f"{module_name}: imports a live runtime launcher "
                f"but no test carries live_claude, live_codex, or live_pi markers",
                stacklevel=1,
            )


@pytest.fixture
def runtime(request):
    return request.config.getoption("--runtime")


@pytest.fixture
def model(request):
    return request.config.getoption("--model")


@pytest.fixture
def effort(request):
    return request.config.getoption("--effort")


@pytest.fixture
def budget(request):
    return request.config.getoption("--budget")


@pytest.fixture
def test_project(request):
    """Yield a TestRunner with tmpdir + git init; cleanup happens via TestRunner's atexit."""
    t = TestRunner(request.node.name)
    create_test_project(t)
    yield t
    # Do not call t.results() here; tests use standard assertions.


@pytest.fixture
def fo_run(test_project, runtime, model, effort):
    """Factory fixture returning a callable that runs the first officer and returns the exit code.

    Tests may still call run_first_officer / run_codex_first_officer directly; this is opt-in.
    """
    def _run(prompt, *, agent_id="spacedock:first-officer", extra_args=None, workflow_dir=None, run_goal=None):
        if runtime == "claude":
            return run_first_officer(
                test_project, prompt,
                agent_id=agent_id,
                extra_args=list(extra_args or []),
            )
        if runtime == "codex":
            return run_codex_first_officer(
                test_project,
                workflow_dir,
                agent_id=agent_id,
                run_goal=run_goal or prompt,
                extra_args=list(extra_args or []),
            )
        return run_pi_first_officer(
            test_project,
            workflow_dir,
            agent_id=agent_id,
            run_goal=run_goal or prompt,
            extra_args=list(extra_args or []),
        )
    return _run
