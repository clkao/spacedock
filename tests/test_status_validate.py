# ABOUTME: Tests that status --validate rejects workflow stage names that violate the
# ABOUTME: dispatch-name character class ^[a-z0-9][a-z0-9-]*[a-z0-9]$.

import os
import tempfile
import textwrap
import unittest

import pytest

from test_status_script import (
    build_status_script,
    make_pipeline,
    run_status,
    README_WITH_STAGES,
)


def _readme_with_stage(stage_name):
    """Return a workflow README whose middle stage has the given name."""
    return textwrap.dedent(f"""\
        ---
        entity-type: task
        entity-label: task
        stages:
          defaults:
            worktree: false
            concurrency: 2
          states:
            - name: backlog
              initial: true
            - name: {stage_name}
            - name: done
              terminal: true
        ---

        # Test Pipeline
        """)


def _readme_no_stages_block():
    return textwrap.dedent("""\
        ---
        entity-type: task
        entity-label: task
        ---

        # Test Pipeline
        """)


@pytest.fixture(scope='module')
def script_path():
    tmpdir = tempfile.mkdtemp()
    path = build_status_script(tmpdir)
    yield path
    os.unlink(path)
    os.rmdir(tmpdir)


@pytest.mark.parametrize('stage_name', [
    'in_progress',
    'InProgress',
    'in progress',
    'in.progress',
    '-leading-hyphen',
    'trailing-hyphen-',
    'x',
])
def test_validate_rejects_invalid_stage_names(script_path, stage_name):
    """status --validate must reject stage names that violate the dispatch-name regex."""
    with tempfile.TemporaryDirectory() as tmpdir:
        make_pipeline(tmpdir, _readme_with_stage(stage_name))
        result = run_status(tmpdir, '--validate', script_path=script_path)
        assert result.returncode != 0, (
            f'expected --validate to fail for stage name {stage_name!r}, '
            f'got returncode 0; stdout={result.stdout!r} stderr={result.stderr!r}'
        )
        assert stage_name in result.stderr, (
            f'error must name the offending stage; stderr={result.stderr!r}'
        )
        assert 'stage name' in result.stderr.lower(), (
            f"error must include 'stage name'; stderr={result.stderr!r}"
        )


def test_validate_underscore_suggests_kebab(script_path):
    """The underscore case must include the kebab-case suggestion in the error."""
    with tempfile.TemporaryDirectory() as tmpdir:
        make_pipeline(tmpdir, _readme_with_stage('in_progress'))
        result = run_status(tmpdir, '--validate', script_path=script_path)
        assert result.returncode != 0
        assert 'in-progress' in result.stderr, (
            f"underscore error must suggest the kebab form 'in-progress'; "
            f"stderr={result.stderr!r}"
        )


def test_validate_accepts_kebab_stage_names(script_path):
    """A workflow whose stage names already match the regex must pass with exit 0 and 'VALID'."""
    with tempfile.TemporaryDirectory() as tmpdir:
        make_pipeline(tmpdir, README_WITH_STAGES)
        result = run_status(tmpdir, '--validate', script_path=script_path)
        assert result.returncode == 0, (
            f'expected --validate to pass on standard fixture; '
            f'stdout={result.stdout!r} stderr={result.stderr!r}'
        )
        assert 'VALID' in result.stdout


def test_validate_silent_when_readme_missing(script_path):
    """validate_workflow must not crash or emit a stage-name error when README is absent."""
    with tempfile.TemporaryDirectory() as tmpdir:
        # No README.md created.
        result = run_status(tmpdir, '--validate', script_path=script_path)
        # validate may exit non-zero for other reasons (no workflow), but it must not
        # raise an exception or emit a 'stage name' error.
        assert 'stage name' not in result.stderr.lower(), (
            f'README-missing must not produce a stage-name error; '
            f'stderr={result.stderr!r}'
        )
        assert 'Traceback' not in result.stderr, (
            f'validate must not crash when README is missing; stderr={result.stderr!r}'
        )


def test_validate_silent_when_no_stages_block(script_path):
    """validate_workflow must not emit a stage-name error when README has no stages block."""
    with tempfile.TemporaryDirectory() as tmpdir:
        make_pipeline(tmpdir, _readme_no_stages_block())
        result = run_status(tmpdir, '--validate', script_path=script_path)
        assert 'stage name' not in result.stderr.lower(), (
            f'no-stages-block must not produce a stage-name error; '
            f'stderr={result.stderr!r}'
        )
        assert 'Traceback' not in result.stderr, (
            f'validate must not crash when README has no stages block; '
            f'stderr={result.stderr!r}'
        )


if __name__ == '__main__':
    unittest.main()
