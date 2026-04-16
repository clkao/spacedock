# ABOUTME: Shared test library for all Spacedock test scripts.
# ABOUTME: Provides test framework, project setup, claude wrappers, log parsing, and stats extraction.

from __future__ import annotations

import argparse
import atexit
import json
import os
import re
import select
import shlex
import shutil
import subprocess
import sys
import tempfile
import textwrap
import time
from collections.abc import Callable
from datetime import datetime
from pathlib import Path


def _agent_targets_stage(agent_input: dict, stage: str) -> bool:
    """Check whether an Agent() call targets a given stage.

    Checks both the name field (which should contain the stage when the FO
    follows the naming convention) and the prompt field, which contains a
    'Stage: {stage}' header. The header appears in two formats depending on
    whether the FO uses the claude-team build helper or hand-assembles the
    prompt:

    - plain (helper): ``Stage: implementation``
    - markdown-bold (hand-assembled by haiku): ``**Stage:** implementation``

    Both forms are accepted.
    """
    name_lower = agent_input.get("name", "").lower()
    if stage in name_lower:
        return True
    prompt = agent_input.get("prompt", "")
    pattern = rf"(?m)^\*{{0,2}}Stage:?\*{{0,2}}\s+\*{{0,2}}{re.escape(stage)}\*{{0,2}}\s*$"
    if re.search(pattern, prompt):
        return True
    return False


def _codex_skill_namespace_root(home_dir: Path) -> Path:
    return home_dir / ".agents" / "skills" / "spacedock"


def prepare_codex_skill_home(test_root: Path, repo_root: Path) -> Path:
    """Create an isolated HOME that exposes the current repo as the spacedock Codex skill namespace."""
    home_dir = test_root / "codex-home"
    namespace_root = _codex_skill_namespace_root(home_dir)
    namespace_root.mkdir(parents=True, exist_ok=True)

    skills_root = repo_root / "skills"
    for skill_dir in sorted(skills_root.iterdir()):
        if not skill_dir.is_dir() or not (skill_dir / "SKILL.md").is_file():
            continue
        link_path = namespace_root / skill_dir.name
        if link_path.exists() or link_path.is_symlink():
            link_path.unlink()
        link_path.symlink_to(skill_dir, target_is_directory=True)

    for name, target in {
        "skills": skills_root,
        "scripts": repo_root / "scripts",
        "references": repo_root / "references",
    }.items():
        link_path = namespace_root / name
        if link_path.exists() or link_path.is_symlink():
            link_path.unlink()
        link_path.symlink_to(target, target_is_directory=True)

    real_codex_home = Path(os.path.expanduser("~")) / ".codex"
    codex_home_link = home_dir / ".codex"
    if codex_home_link.exists() or codex_home_link.is_symlink():
        codex_home_link.unlink()
    if real_codex_home.exists():
        codex_home_link.symlink_to(real_codex_home, target_is_directory=True)
    else:
        codex_home_link.mkdir(parents=True, exist_ok=True)

    return home_dir


def resolve_codex_worker(agent_id: str, repo_root: Path | None = None) -> dict[str, object]:
    """Resolve a logical Codex worker id to a packaged asset and safe worker key."""
    if repo_root is None:
        repo_root = Path(__file__).resolve().parent.parent

    if not agent_id:
        raise ValueError("Codex worker id must not be empty")

    if agent_id.startswith("spacedock:"):
        skill_name = agent_id.split(":", 1)[1]
        asset_path = repo_root / "skills" / skill_name / "SKILL.md"
        if not asset_path.is_file():
            raise ValueError(f"Unknown Codex worker id: {agent_id}")
        return {
            "dispatch_agent_id": agent_id,
            "worker_key": re.sub(r"[^A-Za-z0-9._-]", "-", agent_id),
            "asset_kind": "skill",
            "asset_path": asset_path,
            "asset_name": skill_name,
        }

    worker_key = re.sub(r"[^A-Za-z0-9._-]", "-", agent_id)
    return {
        "dispatch_agent_id": agent_id,
        "worker_key": worker_key,
        "asset_kind": "prompt",
        "asset_name": "generic-worker",
    }


def _extract_skill_includes(skill_text: str) -> list[str]:
    includes: list[str] = []
    for line in skill_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("@") and len(stripped) > 1:
            includes.append(stripped[1:])
    return includes


def resolve_skill_include(
    skill_path: Path,
    include: str,
    repo_root: Path,
) -> tuple[Path, str]:
    """Resolve one skill include relative to the active SKILL.md directory.

    The direct hit is the directory containing the active SKILL.md. If that
    lookup fails, fall back to the repo-level references directory only.
    """
    skill_path = Path(skill_path)
    include_path = Path(include)
    skill_dir = skill_path.parent

    direct_candidate = (skill_dir / include_path).resolve()
    if direct_candidate.is_file():
        return direct_candidate, "skill-relative"

    fallback_candidate = (repo_root / "references" / include_path.name).resolve()
    if fallback_candidate.is_file():
        return fallback_candidate, "bounded-fallback"

    raise FileNotFoundError(
        f"Missing skill include {include!r} requested by {skill_path}"
    )


def _assemble_skill_contract(
    skill_path: Path,
    repo_root: Path,
    seen: set[Path] | None = None,
) -> tuple[list[str], list[str]]:
    skill_path = Path(skill_path).resolve()
    if seen is None:
        seen = set()
    if skill_path in seen:
        return [], []
    seen.add(skill_path)

    text = skill_path.read_text()
    parts = [text]
    trace: list[str] = []
    for include in _extract_skill_includes(text):
        resolved_path, resolution_kind = resolve_skill_include(skill_path, include, repo_root)
        trace.append(f"{include} -> {resolved_path} ({resolution_kind})")
        child_parts, child_trace = _assemble_skill_contract(resolved_path, repo_root, seen)
        parts.extend(child_parts)
        trace.extend(child_trace)
    return parts, trace


def build_codex_first_officer_invocation_prompt(
    workflow_dir: str | Path,
    agent_id: str = "spacedock:first-officer",
    run_goal: str | None = None,
) -> str:
    workflow_dir = Path(workflow_dir)
    prompt = f"Use the `{agent_id}` skill to manage the Codex workflow at `{workflow_dir}`."
    if run_goal:
        prompt = f"{prompt}\n\n{run_goal.strip()}"
    return prompt


def build_codex_worker_bootstrap_prompt(
    resolved_worker: dict[str, object],
    workflow_dir: Path,
    entity_path: Path,
    stage_name: str,
    stage_definition_text: str,
    worktree_path: Path | None,
    checklist: list[str],
) -> str:
    if resolved_worker["asset_kind"] == "skill":
        lines = [
            f"You are the packaged worker `{resolved_worker['dispatch_agent_id']}`.",
            "Resolve your role definition before doing anything else.",
            f"If your operating contract was not already loaded via skill preloading, invoke the `{resolved_worker['dispatch_agent_id']}` skill now to load it.",
            "After the skill is loaded, continue with the assignment below.",
            "",
            "Assignment:",
            f"dispatch_agent_id: {resolved_worker['dispatch_agent_id']}",
            f"worker_key: {resolved_worker['worker_key']}",
            f"role_asset_kind: {resolved_worker['asset_kind']}",
            f"role_asset_name: {resolved_worker['asset_name']}",
            f"role_asset_path: {resolved_worker['asset_path']}",
            f"workflow_dir: {workflow_dir}",
            f"entity_path: {entity_path}",
            f"stage_name: {stage_name}",
            "stage_definition_text:",
            stage_definition_text,
        ]
    else:
        lines = [
            "You are a generic worker handling one entity for one stage.",
            "Operate directly from the assignment below.",
            "Do not modify YAML frontmatter in entity files.",
            "Do not take over first-officer responsibilities.",
            "",
            "Assignment:",
            f"dispatch_agent_id: {resolved_worker['dispatch_agent_id']}",
            f"worker_key: {resolved_worker['worker_key']}",
            f"role_asset_kind: {resolved_worker['asset_kind']}",
            f"role_asset_name: {resolved_worker['asset_name']}",
            f"workflow_dir: {workflow_dir}",
            f"entity_path: {entity_path}",
            f"stage_name: {stage_name}",
            "stage_definition_text:",
            stage_definition_text,
        ]
    if worktree_path is not None:
        lines.append(f"worktree_path: {worktree_path}")
    if checklist:
        lines.extend(["checklist:"] + [f"- {item}" for item in checklist])
    lines.extend(
        [
            "",
            "Completion rule:",
            "After you finish the assignment, write the stage report, commit your work, return one concise final response, and stop immediately.",
            "Do not continue exploring the repo, do not wait for follow-up instructions, and do not start another task after that final response.",
        ]
    )
    return "\n".join(lines)


def _clean_env() -> dict[str, str]:
    """Return a copy of os.environ without CLAUDECODE so subprocess can launch claude."""
    return {k: v for k, v in os.environ.items() if k != "CLAUDECODE"}


def _isolated_claude_env() -> dict[str, str] | None:
    """Return an env dict with an isolated HOME whenever any auth mechanism is available.

    Isolation is unconditional as long as the process can authenticate to the
    Claude API. Parallel `claude -p` subprocesses spawned by pytest-xdist all
    write to `$HOME/.claude/` (caches, state, telemetry); sharing that
    directory across workers causes concurrent-write collisions that surface
    as flaky, correlated inner-check failures. Giving each invocation a fresh
    empty HOME eliminates that coupling.

    Decision tree:

    (a) `~/.claude/benchmark-token` exists and is non-empty → create a fresh
        HOME tmpdir, inject `CLAUDE_CODE_OAUTH_TOKEN`, and drop
        `ANTHROPIC_API_KEY` so the token is the authoritative credential.
        This is the operator-local path (`claude setup-token`).

    (b) No token file, but `ANTHROPIC_API_KEY` is present in the environment
        → create a fresh HOME tmpdir and pass `ANTHROPIC_API_KEY` through.
        This is the CI path: GitHub Actions runners authenticate via that
        env var and never have a benchmark-token on disk, but they still
        need HOME isolation for the same concurrency reason.

    (c) Neither credential is available → return None so the caller can
        fall back to `_clean_env()` without claiming isolation.

    Returns the env dict (caller is responsible for cleaning up the temp dir
    if it tracks one) or None when no auth mechanism is available.
    """
    real_home = os.environ.get("HOME")
    if not real_home:
        return None
    token_path = Path(real_home) / ".claude" / "benchmark-token"
    token = ""
    if token_path.is_file():
        token = token_path.read_text().strip()
    if token:
        clean_home = tempfile.mkdtemp(prefix="spacedock-clean-home-")
        env = _clean_env()
        env["HOME"] = clean_home
        env["CLAUDE_CODE_OAUTH_TOKEN"] = token
        env.pop("ANTHROPIC_API_KEY", None)
        return env
    if os.environ.get("ANTHROPIC_API_KEY"):
        clean_home = tempfile.mkdtemp(prefix="spacedock-clean-home-")
        env = _clean_env()
        env["HOME"] = clean_home
        return env
    return None


def emit_skip_result(reason: str) -> None:
    """Print a standardized SKIP result and exit 0 for standalone uv-run scripts."""
    print(f"  SKIP: {reason}")
    print()
    print("=== Results ===")
    print("  0 passed, 0 failed, 1 skipped")
    print()
    print("RESULT: SKIP")
    raise SystemExit(0)


def probe_claude_runtime(model: str, timeout_s: int = 30) -> tuple[bool, str]:
    """Return whether the local Claude runtime is responsive enough for live E2E."""
    cmd = [
        "claude",
        "-p",
        "Reply with OK and nothing else.",
        "--output-format",
        "stream-json",
        "--verbose",
        "--model",
        model,
        "--max-budget-usd",
        "0.20",
    ]
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            env=_isolated_claude_env() or _clean_env(),
            timeout=timeout_s,
        )
    except FileNotFoundError:
        return False, "claude CLI not found in PATH"
    except subprocess.TimeoutExpired:
        return False, f"claude preflight for model {model!r} produced no result within {timeout_s}s"

    if result.returncode != 0:
        return False, f"claude preflight for model {model!r} exited {result.returncode}"

    if '"type":"result"' not in result.stdout and '"type": "result"' not in result.stdout:
        return False, f"claude preflight for model {model!r} returned no stream-json result record"

    return True, ""


_READ_ONLY_SHELL_COMMANDS = frozenset({
    "cat",
    "file",
    "find",
    "grep",
    "head",
    "ls",
    "rg",
    "stat",
    "tail",
    "wc",
})
_ENV_ASSIGNMENT_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=.*$")
_OPTION_TOKEN_RE = re.compile(r"^-")


def _strip_harmless_redirections(command: str) -> str:
    return re.sub(r"\b\d*>\s*/dev/null\b", "", command)


def _shell_words(command: str) -> list[str]:
    try:
        return shlex.split(command, posix=True)
    except ValueError:
        return command.split()


def _matches_any_target(path: str, target_patterns: tuple[str, ...] | list[str]) -> bool:
    cleaned = path.strip().strip("\"'")
    return any(pattern in cleaned for pattern in target_patterns)


def _segment_is_read_only_probe(segment: str) -> bool:
    words = _shell_words(segment)
    while words and _ENV_ASSIGNMENT_RE.match(words[0]):
        words.pop(0)
    if not words:
        return False
    return words[0] in _READ_ONLY_SHELL_COMMANDS


def bash_command_targets_write(command: str, target_patterns: tuple[str, ...] | list[str]) -> bool:
    """Heuristic for whether a Bash command writes to one of the guarded target paths."""
    stripped = _strip_harmless_redirections(command)

    segments = [
        segment.strip()
        for segment in re.split(r"&&|\|\||;|\|", stripped)
        if segment.strip()
    ]
    if segments and all(_segment_is_read_only_probe(segment) for segment in segments):
        return False

    redirection_targets = re.findall(r">>?\s*([^&;\s|]+)", stripped)
    if any(_matches_any_target(path, target_patterns) for path in redirection_targets):
        return True

    for segment in segments:
        words = _shell_words(segment)
        while words and _ENV_ASSIGNMENT_RE.match(words[0]):
            words.pop(0)
        if not words:
            continue

        cmd = words[0]
        args = words[1:]

        if cmd == "tee":
            if any(not _OPTION_TOKEN_RE.match(arg) and _matches_any_target(arg, target_patterns) for arg in args):
                return True
            continue

        if cmd == "sed" and any(arg.startswith("-i") for arg in args):
            if args and _matches_any_target(args[-1], target_patterns):
                return True
            continue

        if cmd == "perl" and any(arg.startswith("-") and "i" in arg for arg in args):
            if args and _matches_any_target(args[-1], target_patterns):
                return True
            continue

        if cmd in {"cp", "mv", "install", "ln"}:
            path_args = [arg for arg in args if not _OPTION_TOKEN_RE.match(arg)]
            if path_args and _matches_any_target(path_args[-1], target_patterns):
                return True
            continue

        if cmd in {"touch", "mkdir", "chmod", "chown", "rm"}:
            if any(not _OPTION_TOKEN_RE.match(arg) and _matches_any_target(arg, target_patterns) for arg in args):
                return True

    return False


class TestRunner:
    """Test framework with pass/fail counters, check helpers, and results summary."""
    __test__ = False

    __test__ = False

    def __init__(self, test_name: str, keep_test_dir: bool = False):
        self.test_name = test_name
        self.passes = 0
        self.failures = 0
        self.repo_root = Path(__file__).resolve().parent.parent
        temp_root = os.environ.get("SPACEDOCK_TEST_TMP_ROOT")
        if temp_root:
            temp_root_path = Path(temp_root)
            temp_root_path.mkdir(parents=True, exist_ok=True)
            self.test_dir = Path(
                tempfile.mkdtemp(prefix="spacedock-test-", dir=str(temp_root_path))
            )
        else:
            self.test_dir = Path(tempfile.mkdtemp())
        self.log_dir = self.test_dir
        self.keep_test_dir = keep_test_dir or bool(os.environ.get("KEEP_TEST_DIR"))
        self.test_project_dir: Path | None = None

        atexit.register(self._cleanup)

        print(f"=== {test_name} ===")
        print(f"Repo root:  {self.repo_root}")
        print(f"Test dir:   {self.test_dir}")
        print()

    def _cleanup(self):
        if self.keep_test_dir:
            print(f"Test dir preserved at: {self.test_dir}")
        elif self.test_dir.exists():
            shutil.rmtree(self.test_dir)

    def pass_(self, label: str):
        self.passes += 1
        print(f"  PASS: {label}")

    def fail(self, label: str):
        self.failures += 1
        print(f"  FAIL: {label}")

    def check(self, label: str, condition: bool):
        if condition:
            self.pass_(label)
        else:
            self.fail(label)

    def check_cmd(self, label: str, cmd: list[str], **kwargs) -> bool:
        """Run a command; pass if exit code 0, fail otherwise."""
        try:
            subprocess.run(cmd, capture_output=True, check=True, **kwargs)
            self.pass_(label)
            return True
        except (subprocess.CalledProcessError, FileNotFoundError):
            self.fail(label)
            return False

    def results(self):
        """Print summary and exit with appropriate code. Legacy entrypoint for uv-run scripts."""
        self._print_summary()
        sys.exit(1 if self.failures > 0 else 0)

    def finish(self):
        """Print summary and raise AssertionError if any checks failed. Pytest entrypoint."""
        self._print_summary()
        if self.failures > 0:
            raise AssertionError(
                f"{self.failures} of {self.passes + self.failures} checks failed in {self.test_name}"
            )

    def _print_summary(self):
        print()
        print("=== Results ===")
        total = self.passes + self.failures
        print(f"  {self.passes} passed, {self.failures} failed (out of {total} checks)")
        print()

        if self.failures > 0:
            print("RESULT: FAIL")
            print()
            print("Debug info:")
            print(f"  Test dir:   {self.test_dir}")
            for f in sorted(self.log_dir.glob("*.jsonl")) + sorted(self.log_dir.glob("*.txt")):
                print(f"  Log:        {f}")
            # Preserve test dir on failure
            self.keep_test_dir = True
        else:
            print("RESULT: PASS")


def create_test_project(runner: TestRunner, name: str = "test-project") -> Path:
    """Create a temp git project with an empty initial commit."""
    project_dir = runner.test_dir / name
    subprocess.run(["git", "init", str(project_dir)], capture_output=True, check=True)
    subprocess.run(
        ["git", "commit", "--allow-empty", "-m", "init"],
        capture_output=True, check=True, cwd=project_dir,
    )
    runner.test_project_dir = project_dir
    return project_dir


def setup_fixture(runner: TestRunner, fixture_name: str, pipeline_dir: str) -> Path:
    """Copy a fixture from tests/fixtures/ into the test project pipeline directory.

    When the fixture registers a merge hook (`_mods/*.md` with `## Hook: merge`),
    the fixture's stub `status` script is overwritten with the commissioned
    Python status script from `skills/commission/bin/status`. The commissioned
    script enforces the mod-block + merge-hook invariants the stub scripts
    cannot implement in shell, and without it live tests would bypass those
    mechanism-level guards by editing entity frontmatter directly.
    """
    fixture_path = runner.repo_root / "tests" / "fixtures" / fixture_name
    dest = runner.test_project_dir / pipeline_dir
    dest.mkdir(parents=True, exist_ok=True)

    # Copy all files and subdirectories from fixture
    for item in fixture_path.iterdir():
        target = dest / item.name
        if item.is_dir():
            shutil.copytree(item, target, dirs_exist_ok=True)
        else:
            shutil.copy2(item, target)

    if _fixture_has_merge_hook(dest):
        _install_commissioned_status_script(runner.repo_root, dest)

    # Make status script executable if present
    status = dest / "status"
    if status.exists():
        status.chmod(status.stat().st_mode | 0o111)

    return dest


def _fixture_has_merge_hook(pipeline_dir: Path) -> bool:
    """True when the fixture's _mods/ contains at least one `## Hook: merge` entry."""
    mods_dir = pipeline_dir / "_mods"
    if not mods_dir.is_dir():
        return False
    for mod_file in mods_dir.glob("*.md"):
        for line in mod_file.read_text().splitlines():
            if line.strip() == "## Hook: merge":
                return True
    return False


def _install_commissioned_status_script(repo_root: Path, pipeline_dir: Path) -> None:
    """Template and install the real skills/commission/bin/status into pipeline_dir."""
    template = (repo_root / "skills" / "commission" / "bin" / "status").read_text()
    content = template.replace("{spacedock_version}", "0.0.0-live-fixture")
    content = content.replace("{entity_label}", "task")
    content = content.replace(
        "{stage1}, {stage2}, ..., {last_stage}",
        "backlog, ideation, implementation, validation, done",
    )
    (pipeline_dir / "status").write_text(content)


def install_agents(runner: TestRunner, include_ensign: bool = False) -> Path:
    """Copy thin agent wrappers into the test project's .claude/agents/ directory."""
    agents_dir = runner.test_project_dir / ".claude" / "agents"
    agents_dir.mkdir(parents=True, exist_ok=True)

    fo_path = agents_dir / "first-officer.md"
    shutil.copy2(runner.repo_root / "agents" / "first-officer.md", fo_path)

    if include_ensign:
        shutil.copy2(runner.repo_root / "agents" / "ensign.md", agents_dir / "ensign.md")

    return fo_path


def assembled_agent_content(runner: TestRunner, agent_name: str, runtime: str = "claude") -> str:
    """Read a thin agent wrapper and all its referenced files, returning combined content.

    This concatenates the agent entry point with the reference files it
    instructs the agent to read, so tests can check the full behavioral
    contract without running the agent.
    """
    if runtime not in {"claude", "codex"}:
        raise ValueError(f"Unknown runtime: {runtime}")
    skill_root = runner.repo_root / "skills"
    if agent_name == "first-officer":
        skill_path = skill_root / "first-officer" / "SKILL.md"
        core_path = skill_root / "first-officer" / "references" / f"{runtime}-first-officer-runtime-core.md"
        legacy_path = skill_root / "first-officer" / "references" / f"{runtime}-first-officer-runtime.md"
        runtime_path = core_path if core_path.exists() else legacy_path
    elif agent_name == "ensign":
        skill_path = skill_root / "ensign" / "SKILL.md"
        runtime_path = skill_root / "ensign" / "references" / f"{runtime}-ensign-runtime.md"
    else:
        skill_path = None
        runtime_path = None

    parts = [(runner.repo_root / "agents" / f"{agent_name}.md").read_text()]
    if skill_path is not None and skill_path.exists():
        skill_parts, resolution_trace = _assemble_skill_contract(skill_path, runner.repo_root)
        parts.extend(skill_parts)
        if runtime_path is not None and runtime_path.exists():
            parts.append(runtime_path.read_text())
        if resolution_trace:
            trace_lines = ["<!-- skill include resolution -->"]
            trace_lines.extend(f"<!-- {line} -->" for line in resolution_trace)
            parts.append("\n".join(trace_lines))
    return "\n\n".join(parts)


def run_commission(
    runner: TestRunner,
    prompt: str,
    extra_args: list[str] | None = None,
    log_name: str = "commission-log.jsonl",
) -> int:
    """Run claude -p with commission flags. Returns exit code."""
    log_path = runner.log_dir / log_name
    cmd = [
        "claude", "-p", prompt,
        "--plugin-dir", str(runner.repo_root),
        "--permission-mode", "bypassPermissions",
        "--verbose",
        "--output-format", "stream-json",
    ]
    if extra_args:
        cmd.extend(extra_args)

    with open(log_path, "w") as log_file:
        try:
            result = subprocess.run(
                cmd, stdout=log_file, stderr=subprocess.STDOUT,
                cwd=runner.test_project_dir, env=_clean_env(), timeout=600,
            )
        except subprocess.TimeoutExpired:
            print("\n  TIMEOUT: commission exceeded 600s limit")
            return 124

    print()
    if result.returncode != 0:
        print(f"WARNING: claude exited with code {result.returncode}")

    # Auto-extract stats
    extract_stats(log_path, "commission", runner.log_dir)

    return result.returncode


def run_first_officer(
    runner: TestRunner,
    prompt: str,
    agent_id: str = "spacedock:first-officer",
    extra_args: list[str] | None = None,
    log_name: str = "fo-log.jsonl",
) -> int:
    """Run claude -p --plugin-dir ... --agent <agent_id>. Returns exit code."""
    log_path = runner.log_dir / log_name
    cmd = [
        "claude", "-p", prompt,
        "--plugin-dir", str(runner.repo_root),
        "--agent", agent_id,
        "--permission-mode", "bypassPermissions",
        "--verbose",
        "--output-format", "stream-json",
    ]
    if extra_args:
        cmd.extend(extra_args)

    env = _isolated_claude_env() or _clean_env()
    with open(log_path, "w") as log_file:
        try:
            result = subprocess.run(
                cmd, stdout=log_file, stderr=subprocess.STDOUT,
                cwd=runner.test_project_dir, env=env, timeout=600,
            )
        except subprocess.TimeoutExpired:
            print("\n  TIMEOUT: first officer exceeded 600s limit")
            return 124

    print()
    if result.returncode != 0:
        print(f"WARNING: first officer exited with code {result.returncode}")

    # Auto-extract stats
    extract_stats(log_path, "fo", runner.log_dir)

    return result.returncode


def run_codex_first_officer(
    runner: TestRunner,
    workflow_dir: str,
    agent_id: str = "spacedock:first-officer",
    run_goal: str | None = None,
    extra_args: list[str] | None = None,
    log_name: str = "codex-fo-log.txt",
    timeout_s: int = 120,
    stop_checker: Callable[[Path], bool] | None = None,
) -> int:
    """Run the Codex first-officer skill via codex exec. Returns exit code."""
    log_path = runner.log_dir / log_name
    workflow_path = (runner.test_project_dir / workflow_dir).resolve()
    prompt = build_codex_first_officer_invocation_prompt(workflow_path, agent_id=agent_id, run_goal=run_goal)
    (runner.log_dir / "codex-fo-invocation.txt").write_text(prompt + "\n")

    skill_home = prepare_codex_skill_home(runner.test_dir, runner.repo_root)
    env = os.environ.copy()
    env["HOME"] = str(skill_home)
    env["CODEX_HOME"] = str(skill_home / ".codex")

    cmd = [
        "codex",
        "exec",
        "--json",
        "--ephemeral",
        "--skip-git-repo-check",
        "--enable",
        "multi_agent",
        "--dangerously-bypass-approvals-and-sandbox",
        "-C",
        str(runner.test_project_dir),
    ]
    if extra_args:
        cmd.extend(extra_args)
    cmd.append("-")

    with open(log_path, "w") as log_file:
        if stop_checker is None:
            try:
                result = subprocess.run(
                    cmd,
                    input=prompt,
                    text=True,
                    stdout=log_file,
                    stderr=subprocess.STDOUT,
                    cwd=runner.test_project_dir,
                    timeout=timeout_s,
                    env=env,
                )
            except subprocess.TimeoutExpired:
                print(f"\n  TIMEOUT: codex first officer exceeded {timeout_s}s limit")
                return 124
        else:
            idle_after_agent_message_s = 5.0
            process_start = time.monotonic()
            active_item_ids: set[str] = set()
            last_output_at = process_start
            last_completed_agent_message_at: float | None = None
            saw_workflow_activity = False

            proc = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                bufsize=1,
                cwd=runner.test_project_dir,
                env=env,
            )
            assert proc.stdin is not None
            assert proc.stdout is not None
            proc.stdin.write(prompt)
            proc.stdin.close()

            while True:
                if time.monotonic() - process_start > timeout_s:
                    proc.terminate()
                    try:
                        proc.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        proc.kill()
                        proc.wait()
                    print(f"\n  TIMEOUT: codex first officer exceeded {timeout_s}s limit")
                    return 124

                ready, _, _ = select.select([proc.stdout], [], [], 0.5)
                if ready:
                    line = proc.stdout.readline()
                    if line:
                        log_file.write(line)
                        log_file.flush()
                        last_output_at = time.monotonic()
                        try:
                            entry = json.loads(line)
                        except json.JSONDecodeError:
                            entry = None
                        if isinstance(entry, dict):
                            entry_type = entry.get("type")
                            item = entry.get("item", {})
                            if isinstance(item, dict):
                                item_id = item.get("id")
                                trackable_item = item.get("type") in {
                                    "collab_tool_call",
                                    "command_execution",
                                    "file_change",
                                }
                                if entry_type == "item.started" and item_id and trackable_item:
                                    active_item_ids.add(str(item_id))
                                elif entry_type == "item.completed":
                                    if item_id:
                                        active_item_ids.discard(str(item_id))
                                    if item.get("type") == "collab_tool_call":
                                        saw_workflow_activity = True
                                    if item.get("type") == "agent_message":
                                        last_completed_agent_message_at = time.monotonic()
                    elif proc.poll() is not None:
                        break
                elif proc.poll() is not None:
                    break

                idle_long_enough = time.monotonic() - last_output_at >= idle_after_agent_message_s
                if (
                    last_completed_agent_message_at is not None
                    and idle_long_enough
                    and not active_item_ids
                    and saw_workflow_activity
                    and proc.poll() is None
                    and stop_checker(log_path)
                ):
                    proc.terminate()
                    try:
                        proc.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        proc.kill()
                        proc.wait()
                    return 0

            result = subprocess.CompletedProcess(cmd, proc.returncode or 0)

    print()
    if result.returncode != 0:
        print(f"WARNING: codex first officer exited with code {result.returncode}")

    return result.returncode


class LogParser:
    """Parses a stream-json JSONL log file from claude."""

    def __init__(self, log_path: Path | str):
        self.log_path = Path(log_path)
        self._entries: list[dict] | None = None

    @property
    def entries(self) -> list[dict]:
        if self._entries is None:
            self._entries = []
            with open(self.log_path) as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        self._entries.append(json.loads(line))
                    except json.JSONDecodeError:
                        pass
        return self._entries

    def assistant_messages(self) -> list[dict]:
        """Return all assistant message objects."""
        return [
            e for e in self.entries
            if e.get("type") == "assistant" and "message" in e
        ]

    def agent_calls(self) -> list[dict]:
        """Extract Agent() tool calls with subagent_type, name, team_name, and prompt."""
        calls = []
        for msg in self.assistant_messages():
            for block in msg["message"].get("content", []):
                if block.get("type") == "tool_use" and block.get("name") == "Agent":
                    inp = block.get("input", {})
                    calls.append({
                        "subagent_type": inp.get("subagent_type", ""),
                        "name": inp.get("name", ""),
                        "team_name": inp.get("team_name", ""),
                        "prompt": inp.get("prompt", ""),
                    })
        return calls

    def fo_texts(self) -> list[str]:
        """Extract text blocks from assistant messages."""
        texts = []
        for msg in self.assistant_messages():
            for block in msg["message"].get("content", []):
                if block.get("type") == "text":
                    texts.append(block["text"])
        return texts

    def tool_calls(self) -> list[dict]:
        """Extract all tool_use blocks from assistant messages."""
        calls = []
        for msg in self.assistant_messages():
            for block in msg["message"].get("content", []):
                if block.get("type") == "tool_use":
                    calls.append({
                        "name": block.get("name", ""),
                        "input": block.get("input", {}),
                    })
        return calls

    def agent_prompt(self) -> str:
        """Extract the prompt from the last Agent() call (for tests that check dispatch content)."""
        prompt = ""
        for call in self.agent_calls():
            prompt = call["prompt"]
        return prompt

    def write_agent_calls(self, output_path: Path | str):
        """Write agent calls summary to a text file."""
        with open(output_path, "w") as f:
            for call in self.agent_calls():
                f.write(f"subagent_type={call['subagent_type']} name={call['name']}\n")
                f.write(f"prompt_preview={call['prompt'][:200]}\n")
                f.write("---\n")

    def write_fo_texts(self, output_path: Path | str):
        """Write concatenated FO texts to a file."""
        with open(output_path, "w") as f:
            f.write("\n".join(self.fo_texts()))

    def write_tool_calls(self, output_path: Path | str):
        """Write tool calls as JSON to a file."""
        with open(output_path, "w") as f:
            json.dump(self.tool_calls(), f, indent=2)

    def write_agent_prompt(self, output_path: Path | str):
        """Write the agent prompt to a file."""
        with open(output_path, "w") as f:
            f.write(self.agent_prompt())


class CodexLogParser:
    """Parses a mixed JSONL/plain-text log file from `codex exec --json`."""

    def __init__(self, log_path: Path | str):
        self.log_path = Path(log_path)
        self._raw_lines: list[str] | None = None
        self._json_entries: list[dict] | None = None

    @property
    def raw_lines(self) -> list[str]:
        if self._raw_lines is None:
            self._raw_lines = self.log_path.read_text().splitlines() if self.log_path.exists() else []
        return self._raw_lines

    @property
    def json_entries(self) -> list[dict]:
        if self._json_entries is None:
            entries: list[dict] = []
            for line in self.raw_lines:
                try:
                    entries.append(json.loads(line))
                except json.JSONDecodeError:
                    continue
            self._json_entries = entries
        return self._json_entries

    def full_text(self) -> str:
        texts = list(self.raw_lines)
        for entry in self.json_entries:
            item = entry.get("item", {})
            if isinstance(item, dict):
                text = item.get("text")
                if text:
                    texts.append(text)
        return "\n".join(texts)

    def collab_tool_calls(self, tool: str | None = None) -> list[dict]:
        calls: list[dict] = []
        for entry in self.json_entries:
            item = entry.get("item", {})
            if not isinstance(item, dict):
                continue
            if item.get("type") != "collab_tool_call":
                continue
            if tool is not None and item.get("tool") != tool:
                continue
            calls.append(item)
        return calls

    def agent_message_texts(self) -> list[str]:
        texts: list[str] = []
        for entry in self.json_entries:
            item = entry.get("item", {})
            if not isinstance(item, dict):
                continue
            if item.get("type") != "agent_message":
                continue
            text = item.get("text")
            if text:
                texts.append(str(text))
        return texts

    def spawn_count(self) -> int:
        count = 0
        for item in self.collab_tool_calls():
            if item.get("tool") in {"spawn", "spawn_agent"}:
                count += 1
        return count

    def completed_agent_messages(self) -> list[str]:
        messages: list[str] = []
        for item in self.collab_tool_calls("wait"):
            states = item.get("agents_states", {})
            if not isinstance(states, dict):
                continue
            for state in states.values():
                if not isinstance(state, dict):
                    continue
                if state.get("status") == "completed" and state.get("message"):
                    messages.append(str(state["message"]))
        return messages

    def write_text(self, output_path: Path | str):
        with open(output_path, "w") as f:
            f.write(self.full_text())


def extract_stats(log_path: Path | str, phase_name: str, output_dir: Path | str) -> dict:
    """Extract stats from a stream-json log. Prints and writes to file. Returns stats dict."""
    log_path = Path(log_path)
    output_dir = Path(output_dir)

    if not log_path.exists():
        return {}

    first_ts = None
    last_ts = None
    assistant_count = 0
    tool_result_count = 0
    model_counts: dict[str, int] = {}
    input_tokens = 0
    output_tokens = 0
    cache_read = 0
    cache_write = 0

    with open(log_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            ts = obj.get("timestamp")
            if ts:
                if first_ts is None:
                    first_ts = ts
                last_ts = ts

            msg_type = obj.get("type", "")

            if msg_type == "assistant" and "message" in obj:
                assistant_count += 1
                model = obj["message"].get("model", obj.get("model", "unknown"))
                model_counts[model] = model_counts.get(model, 0) + 1
                usage = obj["message"].get("usage", {})
                input_tokens += usage.get("input_tokens", 0)
                output_tokens += usage.get("output_tokens", 0)
                cache_read += usage.get("cache_read_input_tokens", 0)
                cache_write += usage.get("cache_creation_input_tokens", 0)
            elif msg_type == "tool_result":
                tool_result_count += 1

    # Wallclock calculation
    wallclock_s = None
    if first_ts and last_ts:
        try:
            d1 = datetime.fromisoformat(first_ts[:19])
            d2 = datetime.fromisoformat(last_ts[:19])
            wallclock_s = int((d2 - d1).total_seconds())
        except (ValueError, TypeError):
            pass

    wallclock_str = f"{wallclock_s}s" if wallclock_s is not None else "?"
    model_str = ", ".join(f"{m}: {c}" for m, c in sorted(model_counts.items()))

    lines = [
        f"=== Stats: {phase_name} ===",
        f"  Wallclock:        {wallclock_str}",
        f"  Messages:         {assistant_count} assistant, {tool_result_count} tool_result",
        f"  Model delegation: {model_str}",
        f"  Input tokens:     {input_tokens:,}",
        f"  Output tokens:    {output_tokens:,}",
        f"  Cache read:       {cache_read:,}",
        f"  Cache write:      {cache_write:,}",
    ]

    output = "\n".join(lines)
    print(output)

    output_file = output_dir / f"stats-{phase_name}.txt"
    output_file.write_text(output + "\n")

    return {
        "wallclock_s": wallclock_s,
        "assistant_count": assistant_count,
        "tool_result_count": tool_result_count,
        "model_counts": model_counts,
        "input_tokens": input_tokens,
        "output_tokens": output_tokens,
        "cache_read": cache_read,
        "cache_write": cache_write,
    }


def git_add_commit(project_dir: Path, message: str):
    """Stage all and commit in the given project directory."""
    subprocess.run(["git", "add", "-A"], capture_output=True, check=True, cwd=project_dir)
    subprocess.run(
        ["git", "commit", "-m", message],
        capture_output=True, check=True, cwd=project_dir,
    )


def read_entity_frontmatter(entity_path: Path) -> dict[str, str]:
    """Read YAML frontmatter fields from an entity file. Returns dict of field: value."""
    fields = {}
    in_frontmatter = False
    text = entity_path.read_text()
    for line in text.splitlines():
        if line.strip() == "---":
            if in_frontmatter:
                break
            in_frontmatter = True
            continue
        if in_frontmatter and ":" in line:
            key, _, value = line.partition(":")
            fields[key.strip()] = value.strip()
    return fields


def iter_worktree_entity_paths(worktrees_dir: Path, workflow_dir: str, entity_slug: str) -> list[Path]:
    """Return matching entity paths under per-worker worktrees for one workflow entity."""
    if not worktrees_dir.is_dir():
        return []
    return [
        wt / workflow_dir / f"{entity_slug}.md"
        for wt in worktrees_dir.iterdir()
        if (wt / workflow_dir / f"{entity_slug}.md").is_file()
    ]


def check_gate_hold_behavior(runner: TestRunner, workflow_dir: str, entity_slug: str, fo_text_output: str) -> None:
    """Assert that a gated entity remains active and unarchived."""
    entity_file = runner.test_project_dir / workflow_dir / f"{entity_slug}.md"
    archive_file = runner.test_project_dir / workflow_dir / "_archive" / f"{entity_slug}.md"

    if entity_file.is_file():
        fm = read_entity_frontmatter(entity_file)
        status_val = fm.get("status", "")
        if status_val == "done":
            runner.fail("entity did NOT advance past gate (found status: done)")
        else:
            runner.pass_(f"entity did NOT advance past gate (status: {status_val})")
    else:
        runner.fail("entity file exists for status check")

    if archive_file.is_file():
        runner.fail("entity was NOT archived (found in _archive)")
    else:
        runner.pass_("entity was NOT archived (gate held)")

    runner.check(
        "first officer output mentions gate or approval handling",
        bool(re.search(r"gate|approval|approve|reject|waiting", fo_text_output, re.IGNORECASE)),
    )


def rejection_signal_present(
    workflow_dir: str,
    entity_slug: str,
    main_entity_path: Path,
    worktrees_dir: Path,
    *texts: str,
) -> bool:
    """Return True when rejection evidence appears in main/worktree entities or runtime output."""
    patterns = r"REJECTED|recommend reject|failing test|Expected 5, got -1"
    if any(re.search(patterns, text, re.IGNORECASE) for text in texts if text):
        return True
    if main_entity_path.is_file() and re.search(r"REJECTED", main_entity_path.read_text(), re.IGNORECASE):
        return True
    return any(
        re.search(r"REJECTED", path.read_text(), re.IGNORECASE)
        for path in iter_worktree_entity_paths(worktrees_dir, workflow_dir, entity_slug)
    )


def rejection_follow_up_observed(
    workflow_dir: str,
    entity_slug: str,
    worktrees_dir: Path,
    *texts: str,
) -> bool:
    """Return True when logs or entity artifacts show post-rejection follow-up activity."""
    pattern = r"feedback-to|follow-up|fix|rework|implementation"
    if any(re.search(pattern, text, re.IGNORECASE) for text in texts if text):
        return True
    for path in iter_worktree_entity_paths(worktrees_dir, workflow_dir, entity_slug):
        text = path.read_text()
        if re.search(r"Feedback Cycles|Stage Report: validation|Stage Report: implementation", text, re.IGNORECASE):
            return True
    return False


def check_merge_outcome(
    runner: TestRunner,
    project_dir: Path,
    workflow_dir: str,
    entity_slug: str,
    branch_name: str,
    hook_expected: bool,
    archive_required: bool,
) -> None:
    """Assert merge-hook/local-merge outcomes for one terminal entity."""
    workflow_path = project_dir / workflow_dir
    hook_file = workflow_path / "_merge-hook-fired.txt"
    archive_file = workflow_path / "_archive" / f"{entity_slug}.md"
    entity_file = workflow_path / f"{entity_slug}.md"
    worktree_dir = project_dir / ".spacedock" / "worktrees" / branch_name

    if hook_expected:
        runner.check("merge hook fired marker exists", hook_file.is_file())
        if hook_file.is_file():
            runner.check("merge hook fired marker contains entity slug", entity_slug in hook_file.read_text())
    else:
        runner.check("no merge hook marker exists in no-mods run", not hook_file.exists())

    if archive_required:
        if hook_expected:
            runner.check("entity archived after merge hook run", archive_file.is_file())
        else:
            runner.check("entity archived via no-mods fallback", archive_file.is_file())
    elif archive_file.is_file():
        if hook_expected:
            runner.pass_("entity was archived (merge completed after hook)")
        else:
            runner.pass_("entity was archived via local merge (no-mods fallback works)")
    elif entity_file.is_file():
        fm = read_entity_frontmatter(entity_file)
        status_val = fm.get("status", "?")
        if hook_expected:
            print(f"  SKIP: entity not archived (status: {status_val}) — FO may not have completed the full cycle within budget")
        else:
            print(f"  SKIP: entity not archived (status: {status_val}) — FO may not have completed the full cycle within budget")
    else:
        if hook_expected:
            runner.fail("entity was archived (entity file not found in either location)")
        else:
            runner.fail("entity was archived via local merge (entity file not found)")

    if hook_expected:
        runner.check("worktree cleaned up after merge hook run", not worktree_dir.exists())
    else:
        runner.check("worktree cleaned up after no-mods fallback", not worktree_dir.exists())

    branches = subprocess.run(
        ["git", "branch", "--list", branch_name],
        capture_output=True,
        text=True,
        cwd=project_dir,
        check=True,
    ).stdout.strip()
    if hook_expected:
        runner.check("temporary branch cleaned up after merge hook run", branches == "")
    else:
        runner.check("temporary branch cleaned up after no-mods fallback", branches == "")


def file_contains(path: Path | str, pattern: str, case_insensitive: bool = False) -> bool:
    """Check if a file contains a regex pattern."""
    text = Path(path).read_text()
    flags = re.IGNORECASE if case_insensitive else 0
    return bool(re.search(pattern, text, flags))


def file_grep(path: Path | str, pattern: str, case_insensitive: bool = False) -> list[str]:
    """Return all lines matching a regex pattern in a file."""
    text = Path(path).read_text()
    flags = re.IGNORECASE if case_insensitive else 0
    return [line for line in text.splitlines() if re.search(pattern, line, flags)]
