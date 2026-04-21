# ABOUTME: Tests for the Python status script's parsing and dispatch logic.
# ABOUTME: Covers frontmatter parsing, stage ordering, --next eligibility rules, and output format.

import os
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
TEMPLATE_PATH = os.path.join(SCRIPT_DIR, '..', 'skills', 'commission', 'bin', 'status')


def build_status_script(tmpdir):
    """Substitute template variables and return path to a runnable status script."""
    script_path = os.path.join(tmpdir, 'status')
    with open(TEMPLATE_PATH, 'r') as f:
        content = f.read()
    content = content.replace('{spacedock_version}', '0.0.0-test')
    content = content.replace('{entity_label}', 'task')
    content = content.replace('{stage1}, {stage2}, ..., {last_stage}', 'backlog, ideation, implementation, validation, done')
    with open(script_path, 'w') as f:
        f.write(content)
    os.chmod(script_path, 0o755)
    return script_path


def run_status(pipeline_dir, *args, script_path=None, extra_env=None):
    """Run the status script against a pipeline directory."""
    env = {**os.environ, 'PIPELINE_DIR': pipeline_dir}
    if extra_env:
        env.update(extra_env)
    result = subprocess.run(
        ['python3', script_path] + list(args),
        capture_output=True, text=True,
        env=env,
    )
    return result


def make_pipeline(tmpdir, readme_content, entities=None, archived=None):
    """Create a pipeline directory with README and entity files."""
    with open(os.path.join(tmpdir, 'README.md'), 'w') as f:
        f.write(readme_content)
    for name, content in (entities or {}).items():
        with open(os.path.join(tmpdir, name), 'w') as f:
            f.write(content)
    if archived:
        archive_dir = os.path.join(tmpdir, '_archive')
        os.makedirs(archive_dir, exist_ok=True)
        for name, content in archived.items():
            with open(os.path.join(archive_dir, name), 'w') as f:
                f.write(content)


README_WITH_STAGES = textwrap.dedent("""\
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
        - name: ideation
          gate: true
        - name: implementation
          worktree: true
        - name: validation
          worktree: true
          fresh: true
          gate: true
        - name: done
          terminal: true
    ---

    # Test Pipeline
    """)

README_NO_STAGES = textwrap.dedent("""\
    ---
    entity-type: task
    entity-label: task
    ---

    # Test Pipeline
    """)


def entity(id, title, status, score='', source='', worktree='', pr=''):
    """Generate entity frontmatter."""
    return textwrap.dedent(f"""\
        ---
        id: {id}
        title: {title}
        status: {status}
        source: {source}
        started:
        completed:
        verdict:
        score: {score}
        worktree: {worktree}
        pr: {pr}
        ---

        Description.
        """)


class TestDefaultStatus(unittest.TestCase):
    """Test the default status table output."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_basic_output(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'feature-a.md': entity('001', 'Feature A', 'backlog', '0.80', 'user'),
                'feature-b.md': entity('002', 'Feature B', 'ideation', '0.90', 'CL'),
            })
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 4)  # header + separator + 2 data rows
            # Check header
            self.assertIn('ID', lines[0])
            self.assertIn('SLUG', lines[0])
            self.assertIn('STATUS', lines[0])
            self.assertIn('TITLE', lines[0])
            self.assertIn('SCORE', lines[0])
            self.assertIn('SOURCE', lines[0])

    def test_plugin_mode_uses_explicit_workflow_dir(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'feature-a.md': entity('001', 'Feature A', 'backlog', '0.80', 'user'),
            })
            result = subprocess.run(
                ['python3', self.script_path, '--workflow-dir', tmpdir],
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('feature-a', result.stdout)

    def test_sort_order_stage_then_score(self):
        """Entities sorted by stage order ascending, then score descending."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'low-score.md': entity('001', 'Low Score', 'backlog', '0.30'),
                'high-score.md': entity('002', 'High Score', 'backlog', '0.90'),
                'ideation-task.md': entity('003', 'Ideation Task', 'ideation', '0.50'),
            })
            result = run_status(tmpdir, script_path=self.script_path)
            lines = result.stdout.strip().split('\n')[2:]  # skip header+separator
            # backlog (order 1) should come before ideation (order 2)
            # within backlog, high score first
            self.assertIn('high-score', lines[0])
            self.assertIn('low-score', lines[1])
            self.assertIn('ideation-task', lines[2])

    def test_empty_score_sorts_last(self):
        """Entities with empty scores sort after scored ones in same stage."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'scored.md': entity('001', 'Scored', 'backlog', '0.50'),
                'unscored.md': entity('002', 'Unscored', 'backlog', ''),
            })
            result = run_status(tmpdir, script_path=self.script_path)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertIn('scored', lines[0])
            self.assertIn('unscored', lines[1])

    def test_excludes_archive_by_default(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES,
                entities={'active.md': entity('001', 'Active', 'backlog', '0.50')},
                archived={'old.md': entity('002', 'Old', 'done', '0.80')},
            )
            result = run_status(tmpdir, script_path=self.script_path)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('active', lines[0])

    def test_archived_flag_includes_archive(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES,
                entities={'active.md': entity('001', 'Active', 'backlog', '0.50')},
                archived={'old.md': entity('002', 'Old', 'done', '0.80')},
            )
            result = run_status(tmpdir, '--archived', script_path=self.script_path)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 2)

    def test_empty_fields_show_blank(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'blank.md': entity('001', 'Blank Fields', 'backlog'),
            })
            result = run_status(tmpdir, script_path=self.script_path)
            # Should not contain '-' or '0' for empty fields
            self.assertEqual(result.returncode, 0)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)


class TestNextOption(unittest.TestCase):
    """Test --next dispatch eligibility detection."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_basic_dispatchable(self):
        """Entity in backlog (non-terminal, no gate, no worktree) is dispatchable."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'ready.md': entity('001', 'Ready Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertIn('ID', lines[0])
            self.assertIn('SLUG', lines[0])
            self.assertIn('CURRENT', lines[0])
            self.assertIn('NEXT', lines[0])
            self.assertIn('WORKTREE', lines[0])
            data_lines = lines[2:]  # skip header+separator
            self.assertEqual(len(data_lines), 1)
            self.assertIn('ready', data_lines[0])
            self.assertIn('backlog', data_lines[0])
            self.assertIn('ideation', data_lines[0])

    def test_terminal_excluded(self):
        """Entity in terminal stage (done) is not dispatchable."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'finished.md': entity('001', 'Finished', 'done', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 0)

    def test_gate_blocked_excluded(self):
        """Entity in a gated stage (ideation has gate: true) is not dispatchable."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'gated.md': entity('001', 'Gated Task', 'ideation', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 0)

    def test_active_worktree_excluded(self):
        """Entity with non-empty worktree field is not dispatchable."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'working.md': entity('001', 'Working', 'implementation', '0.80', worktree='.worktrees/ensign-working'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 0)

    def test_concurrency_limit(self):
        """Entity not dispatchable when next stage has active ensigns at capacity."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                # Two actively-worked entities in ideation (concurrency default is 2)
                'in-ideation-1.md': entity('001', 'Ideation 1', 'ideation', '0.90', worktree='.worktrees/ensign-in-ideation-1'),
                'in-ideation-2.md': entity('002', 'Ideation 2', 'ideation', '0.85', worktree='.worktrees/ensign-in-ideation-2'),
                # This backlog entity wants to move to ideation but it's full
                'waiting.md': entity('003', 'Waiting', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 0)

    def test_concurrency_parked_not_counted(self):
        """Parked entities (no worktree) in a stage don't consume concurrency slots."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                # Two entities in ideation but NOT actively worked (no worktree)
                'in-ideation-1.md': entity('001', 'Ideation 1', 'ideation', '0.90'),
                'in-ideation-2.md': entity('002', 'Ideation 2', 'ideation', '0.85'),
                # This backlog entity should be dispatchable — parked entities don't block
                'waiting.md': entity('003', 'Waiting', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 1)
            self.assertIn('waiting', data_lines[0])

    def test_concurrency_available(self):
        """Entity dispatchable when next stage has room below concurrency limit."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                # One actively-worked entity in ideation (concurrency=2, room for one more)
                'in-ideation.md': entity('001', 'Ideation 1', 'ideation', '0.90', worktree='.worktrees/ensign-in-ideation'),
                'waiting.md': entity('002', 'Waiting', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 1)
            self.assertIn('waiting', data_lines[0])

    def test_next_sorted_by_score_desc(self):
        """--next output sorted by score descending."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Use a README with concurrency 2 and one active slot taken
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'low.md': entity('001', 'Low Priority', 'backlog', '0.30'),
                'high.md': entity('002', 'High Priority', 'backlog', '0.90'),
                'mid.md': entity('003', 'Mid Priority', 'backlog', '0.60'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            # No active ensigns in ideation, so concurrency allows 2 dispatches
            self.assertEqual(len(data_lines), 2)
            self.assertIn('high', data_lines[0])
            self.assertIn('mid', data_lines[1])

    def test_next_worktree_column(self):
        """WORKTREE column reflects the next stage's worktree property."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                # backlog -> ideation (worktree: false by default)
                'to-ideation.md': entity('001', 'To Ideation', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertIn('no', data_lines[0])

    def test_next_worktree_yes(self):
        """WORKTREE column shows 'yes' when next stage has worktree: true."""
        with tempfile.TemporaryDirectory() as tmpdir:
            readme = textwrap.dedent("""\
                ---
                entity-type: task
                stages:
                  defaults:
                    worktree: false
                    concurrency: 2
                  states:
                    - name: backlog
                      initial: true
                    - name: ideation
                    - name: implementation
                      worktree: true
                    - name: done
                      terminal: true
                ---

                # Test
                """)
            make_pipeline(tmpdir, readme, {
                'task.md': entity('001', 'Task', 'ideation', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 1)
            self.assertIn('yes', data_lines[0])
            self.assertIn('implementation', data_lines[0])

    def test_no_stages_block_error(self):
        """--next prints error and exits non-zero if README lacks stages block."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_NO_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('stages', result.stderr.lower())

    def test_unknown_status_skipped(self):
        """Entity with unknown status is skipped in --next."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'unknown.md': entity('001', 'Unknown', 'mystery', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 0)

    def test_empty_pipeline(self):
        """--next with no entities outputs header only."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            self.assertEqual(result.returncode, 0)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 2)  # header + separator

    def test_id_column_present(self):
        """--next output includes ID column."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('042', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertIn('042', data_lines[0])


class TestNextIdOption(unittest.TestCase):
    """Test the narrow --next-id output path."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_next_id_prints_only_value(self):
        """--next-id prints just the next sequential ID, with archived ids included."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                README_WITH_STAGES,
                entities={'active.md': entity('001', 'Active', 'backlog', '0.50')},
                archived={'archived.md': entity('009', 'Archived', 'done', '0.80')},
            )
            result = run_status(tmpdir, '--next-id', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, '010\n')


class TestFrontmatterParsing(unittest.TestCase):
    """Test edge cases in YAML frontmatter parsing."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_multiword_title(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'test.md': entity('001', 'A Multi Word Title', 'backlog', '0.50'),
            })
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertIn('A Multi Word Title', result.stdout)

    def test_empty_worktree_not_blocked(self):
        """Entity with empty worktree: field is NOT considered actively worked."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80', worktree=''),
            })
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            data_lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(data_lines), 1)


class TestWhereFilter(unittest.TestCase):
    """Test --where filtering."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_exact_match(self):
        """--where 'status = backlog' returns only backlog entities."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'in-backlog.md': entity('001', 'Backlog Task', 'backlog', '0.80'),
                'in-ideation.md': entity('002', 'Ideation Task', 'ideation', '0.90'),
                'also-backlog.md': entity('003', 'Also Backlog', 'backlog', '0.70'),
            })
            result = run_status(tmpdir, '--where', 'status = backlog', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 2)
            for line in lines:
                self.assertIn('backlog', line)
            # ideation entity must not appear
            self.assertNotIn('in-ideation', result.stdout)

    def test_not_equal_with_value(self):
        """--where 'status != done' excludes entities with status done."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'active.md': entity('001', 'Active', 'backlog', '0.80'),
                'finished.md': entity('002', 'Finished', 'done', '0.90'),
                'working.md': entity('003', 'Working', 'implementation', '0.70'),
            })
            result = run_status(tmpdir, '--where', 'status != done', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 2)
            for line in lines:
                self.assertNotIn('done', line.split()[2] if len(line.split()) > 2 else '')
            self.assertNotIn('finished', result.stdout)

    def test_non_empty_filter(self):
        """--where 'worktree !=' returns entities with non-empty worktree."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'has-wt.md': entity('001', 'Has Worktree', 'implementation', '0.80', worktree='.worktrees/test'),
                'no-wt.md': entity('002', 'No Worktree', 'backlog', '0.90'),
            })
            result = run_status(tmpdir, '--where', 'worktree !=', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('has-wt', lines[0])

    def test_empty_filter(self):
        """--where 'worktree =' returns entities with empty worktree."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'has-wt.md': entity('001', 'Has Worktree', 'implementation', '0.80', worktree='.worktrees/test'),
                'no-wt.md': entity('002', 'No Worktree', 'backlog', '0.90'),
            })
            result = run_status(tmpdir, '--where', 'worktree =', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('no-wt', lines[0])

    def test_non_empty_pr_field(self):
        """--where 'pr !=' returns entities with non-empty pr field."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'has-pr.md': entity('001', 'Has PR', 'validation', '0.80', pr='#19'),
                'no-pr.md': entity('002', 'No PR', 'backlog', '0.90'),
            })
            result = run_status(tmpdir, '--where', 'pr !=', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('has-pr', lines[0])

    def test_multiple_where_and_together(self):
        """Multiple --where clauses AND together."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'match.md': entity('001', 'Match', 'backlog', '0.80', source='user'),
                'wrong-status.md': entity('002', 'Wrong Status', 'ideation', '0.90', source='user'),
                'wrong-source.md': entity('003', 'Wrong Source', 'backlog', '0.70', source='CL'),
            })
            result = run_status(tmpdir,
                '--where', 'status = backlog',
                '--where', 'source = user',
                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('match', lines[0])

    def test_where_composes_with_next(self):
        """--where composes with --next."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'high.md': entity('001', 'High Priority', 'backlog', '0.90', source='user'),
                'low.md': entity('002', 'Low Priority', 'backlog', '0.50', source='CL'),
            })
            result = run_status(tmpdir,
                '--next', '--where', 'source = user',
                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('high', lines[0])
            self.assertNotIn('low', result.stdout.split('\n', 3)[-1])

    def test_where_composes_with_archived(self):
        """--where composes with --archived."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES,
                entities={'active.md': entity('001', 'Active', 'backlog', '0.80')},
                archived={
                    'old-done.md': entity('002', 'Old Done', 'done', '0.90'),
                    'old-backlog.md': entity('003', 'Old Backlog', 'backlog', '0.70'),
                },
            )
            result = run_status(tmpdir,
                '--archived', '--where', 'status = backlog',
                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 2)
            for line in lines:
                self.assertIn('backlog', line)

    def test_no_matching_entities_header_only(self):
        """--where with no matches returns header only."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--where', 'status = done', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 2)  # header + separator only

    def test_where_on_nonexistent_field(self):
        """--where on a field not in frontmatter treats it as empty."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            # Filter for non-empty 'nonexistent' — should match nothing
            result = run_status(tmpdir, '--where', 'nonexistent !=', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 2)  # header + separator only

            # Filter for empty 'nonexistent' — should match everything
            result2 = run_status(tmpdir, '--where', 'nonexistent =', script_path=self.script_path)
            self.assertEqual(result2.returncode, 0, result2.stderr)
            lines2 = result2.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines2), 1)

    # AC1: --where accepts unspaced equality
    def test_equality_no_spaces(self):
        """--where 'status=backlog' matches the same rows as 'status = backlog'."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'in-backlog.md': entity('001', 'Backlog Task', 'backlog', '0.80'),
                'in-ideation.md': entity('002', 'Ideation Task', 'ideation', '0.90'),
            })
            spaced = run_status(tmpdir, '--where', 'status = backlog', script_path=self.script_path)
            unspaced = run_status(tmpdir, '--where', 'status=backlog', script_path=self.script_path)
            self.assertEqual(spaced.returncode, 0, spaced.stderr)
            self.assertEqual(unspaced.returncode, 0, unspaced.stderr)
            self.assertEqual(spaced.stdout, unspaced.stdout)
            self.assertIn('in-backlog', unspaced.stdout)
            self.assertNotIn('in-ideation', unspaced.stdout)

    # AC1: same on a custom field
    def test_equality_no_spaces_custom_field(self):
        """Unspaced equality works on custom frontmatter fields (e.g. last-outbound-at)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            custom_a = textwrap.dedent("""\
                ---
                id: 001
                title: Outreach A
                status: backlog
                source:
                started:
                completed:
                verdict:
                score: 0.80
                worktree:
                pr:
                last-outbound-at: 2026-04-01
                ---

                Description.
                """)
            custom_b = textwrap.dedent("""\
                ---
                id: 002
                title: Outreach B
                status: backlog
                source:
                started:
                completed:
                verdict:
                score: 0.70
                worktree:
                pr:
                last-outbound-at: 2026-04-02
                ---

                Description.
                """)
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'outreach-a.md': custom_a,
                'outreach-b.md': custom_b,
            })
            spaced = run_status(tmpdir, '--where', 'last-outbound-at = 2026-04-01',
                                script_path=self.script_path)
            unspaced = run_status(tmpdir, '--where', 'last-outbound-at=2026-04-01',
                                  script_path=self.script_path)
            self.assertEqual(spaced.returncode, 0, spaced.stderr)
            self.assertEqual(unspaced.returncode, 0, unspaced.stderr)
            self.assertEqual(spaced.stdout, unspaced.stdout)
            self.assertIn('outreach-a', unspaced.stdout)
            self.assertNotIn('outreach-b', unspaced.stdout)

    # AC2: --where accepts unspaced negation
    def test_negation_no_spaces(self):
        """--where 'status!=done' matches the same rows as 'status != done'."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'active.md': entity('001', 'Active', 'backlog', '0.80'),
                'finished.md': entity('002', 'Finished', 'done', '0.90'),
            })
            spaced = run_status(tmpdir, '--where', 'status != done', script_path=self.script_path)
            unspaced = run_status(tmpdir, '--where', 'status!=done', script_path=self.script_path)
            self.assertEqual(spaced.returncode, 0, spaced.stderr)
            self.assertEqual(unspaced.returncode, 0, unspaced.stderr)
            self.assertEqual(spaced.stdout, unspaced.stdout)
            self.assertIn('active', unspaced.stdout)
            self.assertNotIn('finished', unspaced.stdout)

    # AC4: bare field name is rejected with a clear error listing the four valid forms
    def test_bare_field_errors(self):
        """--where 'completed' exits non-zero and error names the four valid forms."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--where', 'completed', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('field = value', result.stderr)
            self.assertIn('field != value', result.stderr)
            self.assertIn("field !=", result.stderr)
            self.assertIn("field =", result.stderr)

    # AC5: unknown operators are rejected
    def test_unknown_operator_errors(self):
        """--where 'status ~ watching' exits non-zero with an error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'watching', '0.80'),
            })
            result = run_status(tmpdir, '--where', 'status ~ watching', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('--where', result.stderr)

    # Edge case 1: values with spaces parse correctly
    def test_value_with_spaces(self):
        """--where 'title = My Task' matches a title containing spaces."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'My Task', 'backlog', '0.80'),
                'b.md': entity('002', 'Other', 'backlog', '0.70'),
            })
            result = run_status(tmpdir, '--where', 'title = My Task', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('My Task', lines[0])
            self.assertNotIn('Other', result.stdout.split('\n', 2)[-1])

    # Edge case 3: field names are case-sensitive
    def test_field_name_case_sensitive(self):
        """--where 'Status = backlog' does NOT match 'status: backlog'."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            # Uppercase field name should not match a lowercase frontmatter key.
            # Field lookup returns empty, so the equality filter yields zero rows.
            result = run_status(tmpdir, '--where', 'Status = backlog', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 2)  # header + separator only


class TestBootOption(unittest.TestCase):
    """Test --boot startup data output."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)


    def _init_git_repo(self, tmpdir):
        """Initialize a real git repo for worktree integration tests."""
        subprocess.run(['git', 'init', '-b', 'main'], cwd=tmpdir, check=True,
                       capture_output=True, text=True)
        subprocess.run(['git', 'config', 'user.name', 'Test User'], cwd=tmpdir, check=True,
                       capture_output=True, text=True)
        subprocess.run(['git', 'config', 'user.email', 'test@example.com'], cwd=tmpdir, check=True,
                       capture_output=True, text=True)
        subprocess.run(['git', 'add', 'README.md'], cwd=tmpdir, check=True,
                       capture_output=True, text=True)
        for name in os.listdir(tmpdir):
            if name.endswith('.md') and name != 'README.md':
                subprocess.run(['git', 'add', name], cwd=tmpdir, check=True,
                               capture_output=True, text=True)
        subprocess.run(['git', 'commit', '-m', 'init'], cwd=tmpdir, check=True,
                       capture_output=True, text=True)

    def _add_real_worktree(self, tmpdir, worktree_name, branch_name):
        """Create a real worktree and branch under the test pipeline repo."""
        worktree_dir = os.path.join(tmpdir, '.worktrees', worktree_name)
        os.makedirs(os.path.dirname(worktree_dir), exist_ok=True)
        subprocess.run(
            ['git', 'worktree', 'add', '-b', branch_name, worktree_dir],
            cwd=tmpdir,
            check=True,
            capture_output=True,
            text=True,
        )
        return worktree_dir

    def _make_fake_gh(self, tmpdir, pr_states=None):
        """Create a fake gh script that returns canned PR states.

        pr_states: dict mapping PR number (str) to state (e.g., {'19': 'MERGED'})
        """
        fake_bin = os.path.join(tmpdir, '_fake_bin')
        os.makedirs(fake_bin, exist_ok=True)
        gh_script = os.path.join(fake_bin, 'gh')
        # Build case branches
        cases = ''
        for pr_num, state in (pr_states or {}).items():
            cases += '  %s) echo "%s"; exit 0;;\n' % (pr_num, state)
        with open(gh_script, 'w') as f:
            f.write('#!/bin/sh\n')
            f.write('# Fake gh for testing PR state checks\n')
            f.write('if [ "$1" = "pr" ] && [ "$2" = "view" ]; then\n')
            f.write('  PR_NUM="$3"\n')
            f.write('  case "$PR_NUM" in\n')
            f.write(cases)
            f.write('  *) echo "not found" >&2; exit 1;;\n')
            f.write('  esac\n')
            f.write('fi\n')
            f.write('exit 1\n')
        os.chmod(gh_script, 0o755)
        return fake_bin

    def _path_without_gh(self):
        """Return a PATH string that excludes directories containing a real gh binary."""
        dirs = os.environ.get('PATH', '').split(os.pathsep)
        filtered = [d for d in dirs if not os.path.isfile(os.path.join(d, 'gh'))]
        return os.pathsep.join(filtered)

    # AC1: MODS section with hooks
    def test_mods_with_hooks(self):
        """MODS section lists hooks grouped by lifecycle point."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            mods_dir = os.path.join(tmpdir, '_mods')
            os.makedirs(mods_dir)
            with open(os.path.join(mods_dir, 'pr-merge.md'), 'w') as f:
                f.write('---\nname: pr-merge\n---\n\n## Hook: startup\n\nDo stuff.\n\n## Hook: idle\n\nMore stuff.\n')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            self.assertIn('MODS', lines)
            # Check hook points are listed
            self.assertTrue(any('startup: pr-merge' in line for line in lines), lines)
            self.assertTrue(any('idle: pr-merge' in line for line in lines), lines)

    # AC2: MODS: none when no mods exist
    def test_mods_none(self):
        """MODS shows 'MODS: none' when no mods directory exists."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('MODS: none', result.stdout)

    # AC3: NEXT_ID across active + archive
    def test_next_id_across_active_and_archive(self):
        """NEXT_ID is computed from highest ID across active and archived entities."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES,
                entities={
                    'task-a.md': entity('001', 'Task A', 'backlog'),
                    'task-c.md': entity('003', 'Task C', 'backlog'),
                },
                archived={
                    'task-b.md': entity('"004"', 'Task B', 'done'),
                },
            )
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('NEXT_ID: 005', result.stdout)

    # AC4: Quoted values normalize consistently
    def test_quoted_frontmatter_values_are_normalized(self):
        """Quoted YAML frontmatter values are parsed without literal quote characters."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'quoted.md': entity('"084"', 'Quoted Task', 'backlog', pr='"#28"'),
            })
            result = run_status(tmpdir, '--where', 'id = 084', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines), 1)
            self.assertIn('084', lines[0])
            self.assertNotIn('"084"', result.stdout)

            result2 = run_status(tmpdir, '--where', 'pr = #28', script_path=self.script_path)
            self.assertEqual(result2.returncode, 0, result2.stderr)
            lines2 = result2.stdout.strip().split('\n')[2:]
            self.assertEqual(len(lines2), 1)
            self.assertIn('quoted', lines2[0])
            self.assertNotIn('"#28"', result2.stdout)

    # AC4: ORPHANS with DIR_EXISTS and BRANCH_EXISTS
    def test_orphans_with_existence_checks(self):
        """ORPHANS section shows entities with worktree field and existence columns."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'implementation', worktree='.worktrees/ensign-task-a'),
                'task-b.md': entity('002', 'Task B', 'implementation', worktree='.worktrees/ensign-task-b'),
            })
            self._init_git_repo(tmpdir)
            self._add_real_worktree(tmpdir, 'ensign-task-a', 'ensign-task-a')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': os.environ.get('PATH', '')})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            # Find ORPHANS section
            orphan_idx = lines.index('ORPHANS')
            self.assertGreater(orphan_idx, -1)
            # Header row
            self.assertIn('DIR_EXISTS', lines[orphan_idx + 1])
            self.assertIn('BRANCH_EXISTS', lines[orphan_idx + 1])
            # task-a: dir exists, branch exists
            task_a_line = [l for l in lines if 'task-a' in l and '001' in l][0]
            self.assertIn('yes', task_a_line.split('ensign-task-a')[1][:30])
            # task-b: dir does not exist, branch does not exist
            task_b_line = [l for l in lines if 'task-b' in l and '002' in l][0]
            parts = task_b_line.split()
            # Last two columns should be 'no' 'no'
            self.assertEqual(parts[-2], 'no')
            self.assertEqual(parts[-1], 'no')

    def test_orphans_namespaced_branch_detected(self):
        """Branch with / in name is correctly detected via worktree path lookup."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'feature-name.md': entity('001', 'Feature', 'implementation',
                                          worktree='.worktrees/ensign-feature-name'),
            })
            self._init_git_repo(tmpdir)
            self._add_real_worktree(tmpdir, 'ensign-feature-name', 'ensign/feature-name')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': os.environ.get('PATH', '')})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            feature_line = [l for l in lines if 'feature-name' in l and '001' in l][0]
            parts = feature_line.split()
            # dir exists, branch exists
            self.assertEqual(parts[-2], 'yes', 'DIR_EXISTS should be yes')
            self.assertEqual(parts[-1], 'yes', 'BRANCH_EXISTS should be yes')

    def test_orphans_missing_worktree_detected(self):
        """Entity whose worktree path is not in git worktree list reports BRANCH_EXISTS: no."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'ghost.md': entity('002', 'Ghost', 'implementation',
                                   worktree='.worktrees/ensign-ghost'),
            })
            self._init_git_repo(tmpdir)
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': os.environ.get('PATH', '')})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            ghost_line = [l for l in lines if 'ghost' in l and '002' in l][0]
            parts = ghost_line.split()
            self.assertEqual(parts[-2], 'no', 'DIR_EXISTS should be no')
            self.assertEqual(parts[-1], 'no', 'BRANCH_EXISTS should be no')

    def test_orphans_simple_branch_still_works(self):
        """Branch without / in name is still correctly detected (no regression)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'codex.md': entity('003', 'Codex', 'implementation',
                                   worktree='.worktrees/remove-codex-dispatcher'),
            })
            self._init_git_repo(tmpdir)
            self._add_real_worktree(tmpdir, 'remove-codex-dispatcher', 'remove-codex-dispatcher')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': os.environ.get('PATH', '')})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            codex_line = [l for l in lines if 'codex' in l and '003' in l][0]
            parts = codex_line.split()
            self.assertEqual(parts[-2], 'yes', 'DIR_EXISTS should be yes')
            self.assertEqual(parts[-1], 'yes', 'BRANCH_EXISTS should be yes')

    # AC5: ORPHANS: none
    def test_orphans_none(self):
        """ORPHANS shows 'ORPHANS: none' when no entities have worktree fields."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('ORPHANS: none', result.stdout)

    # AC6: PR_STATE with PR number and state
    def test_pr_state_with_pr(self):
        """PR_STATE shows PR number and state for PR-pending entities."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'validation', pr='#19'),
            })
            fake_bin = self._make_fake_gh(tmpdir, {'19': 'MERGED'})
            path = fake_bin + os.pathsep + os.environ.get('PATH', '')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': path})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            self.assertIn('PR_STATE', lines)
            pr_line = [l for l in lines if '#19' in l][0]
            self.assertIn('MERGED', pr_line)
            self.assertIn('001', pr_line)

    # AC7: PR_STATE when gh unavailable
    def test_pr_state_gh_unavailable(self):
        """PR_STATE shows 'gh not available' when gh is not on PATH."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'validation', pr='#19'),
            })
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('PR_STATE: gh not available', result.stdout)

    # AC8: PR_STATE skips terminal entities
    def test_pr_state_skips_terminal(self):
        """PR_STATE skips entities in terminal status."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'done-task.md': entity('001', 'Done Task', 'done', pr='#19'),
            })
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('PR_STATE: none', result.stdout)

    # AC9: DISPATCHABLE section matches --next
    def test_dispatchable_matches_next(self):
        """DISPATCHABLE section contains the same data as --next output."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'ready.md': entity('001', 'Ready Task', 'backlog', '0.80'),
            })
            # Get --next output for comparison
            next_result = run_status(tmpdir, '--next', script_path=self.script_path)
            # Get --boot output
            boot_result = run_status(tmpdir, '--boot', script_path=self.script_path,
                                    extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(boot_result.returncode, 0, boot_result.stderr)
            lines = boot_result.stdout.split('\n')
            disp_idx = lines.index('DISPATCHABLE')
            # The next table content should follow
            next_lines = next_result.stdout.strip().split('\n')
            boot_disp_lines = lines[disp_idx + 1:]
            # Compare header and data (strip trailing empty lines)
            boot_disp_lines = [l for l in boot_disp_lines if l]
            for i, expected in enumerate(next_lines):
                self.assertEqual(boot_disp_lines[i].rstrip(), expected.rstrip())

    # AC12: --boot errors without stages block
    def test_boot_requires_stages(self):
        """--boot prints error and exits non-zero if README lacks stages block."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_NO_STAGES, {
                'task.md': entity('001', 'Task', 'backlog'),
            })
            result = run_status(tmpdir, '--boot', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('stages', result.stderr.lower())

    # AC13: --boot incompatible with --next
    def test_boot_incompatible_with_next(self):
        """--boot combined with --next produces an error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog'),
            })
            result = run_status(tmpdir, '--boot', '--next', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('incompatible', result.stderr.lower())

    # AC13: --boot incompatible with --archived
    def test_boot_incompatible_with_archived(self):
        """--boot combined with --archived produces an error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog'),
            })
            result = run_status(tmpdir, '--boot', '--archived', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('incompatible', result.stderr.lower())

    # AC13: --boot incompatible with --where
    def test_boot_incompatible_with_where(self):
        """--boot combined with --where produces an error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog'),
            })
            result = run_status(tmpdir, '--boot', '--where', 'status = backlog',
                               script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('incompatible', result.stderr.lower())

    # AC14: section order
    def test_section_order(self):
        """All sections appear in deterministic order: MODS, NEXT_ID, ORPHANS, PR_STATE, DISPATCHABLE."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            output = result.stdout
            # Find positions of section markers
            mods_pos = output.index('MODS')
            next_id_pos = output.index('NEXT_ID')
            orphans_pos = output.index('ORPHANS')
            pr_state_pos = output.index('PR_STATE')
            disp_pos = output.index('DISPATCHABLE')
            self.assertLess(mods_pos, next_id_pos)
            self.assertLess(next_id_pos, orphans_pos)
            self.assertLess(orphans_pos, pr_state_pos)
            self.assertLess(pr_state_pos, disp_pos)

    # Reviewer addition: mod file with no ## Hook: headings
    def test_mods_file_without_hooks(self):
        """Mod file with no ## Hook: headings is silently skipped."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog'),
            })
            mods_dir = os.path.join(tmpdir, '_mods')
            os.makedirs(mods_dir)
            with open(os.path.join(mods_dir, 'empty-mod.md'), 'w') as f:
                f.write('---\nname: empty-mod\n---\n\n# No hooks here\n\nJust text.\n')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('MODS: none', result.stdout)

    # Reviewer addition: multiple mods registering for the same hook point
    def test_multiple_mods_same_hook(self):
        """Multiple mods registering for the same hook point are listed comma-separated."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': entity('001', 'Task', 'backlog'),
            })
            mods_dir = os.path.join(tmpdir, '_mods')
            os.makedirs(mods_dir)
            with open(os.path.join(mods_dir, 'auto-label.md'), 'w') as f:
                f.write('---\nname: auto-label\n---\n\n## Hook: startup\n\nLabel stuff.\n')
            with open(os.path.join(mods_dir, 'pr-merge.md'), 'w') as f:
                f.write('---\nname: pr-merge\n---\n\n## Hook: startup\n\nMerge stuff.\n')
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': self._path_without_gh()})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            startup_line = [l for l in lines if l.startswith('startup:')][0]
            self.assertIn('auto-label', startup_line)
            self.assertIn('pr-merge', startup_line)
            # Alphabetical order
            self.assertIn('startup: auto-label, pr-merge', startup_line)

    # Reviewer addition: PR_STATE ERROR for specific PR failure
    def test_pr_state_per_pr_error(self):
        """PR_STATE shows ERROR when gh pr view fails for a specific PR."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'good-pr.md': entity('001', 'Good PR', 'validation', pr='#19'),
                'bad-pr.md': entity('002', 'Bad PR', 'validation', pr='#99'),
            })
            # Only PR 19 returns a state; PR 99 will get an error
            fake_bin = self._make_fake_gh(tmpdir, {'19': 'MERGED'})
            path = fake_bin + os.pathsep + self._path_without_gh()
            result = run_status(tmpdir, '--boot', script_path=self.script_path,
                               extra_env={'PATH': path})
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.split('\n')
            good_line = [l for l in lines if '#19' in l][0]
            bad_line = [l for l in lines if '#99' in l][0]
            self.assertIn('MERGED', good_line)
            self.assertIn('ERROR', bad_line)


class TestSetOption(unittest.TestCase):
    """Test --set field update functionality."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _read_frontmatter(self, filepath):
        """Read frontmatter from a file, returning dict of fields."""
        fields = {}
        in_fm = False
        with open(filepath, 'r') as f:
            for line in f:
                line = line.rstrip('\n')
                if line == '---':
                    if in_fm:
                        break
                    in_fm = True
                    continue
                if in_fm and ':' in line:
                    key, _, val = line.partition(':')
                    fields[key.strip()] = val.strip()
        return fields

    def _read_body(self, filepath):
        """Read file content after frontmatter."""
        lines = []
        in_fm = False
        past_fm = False
        with open(filepath, 'r') as f:
            for line in f:
                if past_fm:
                    lines.append(line)
                    continue
                if line.rstrip('\n') == '---':
                    if in_fm:
                        past_fm = True
                    else:
                        in_fm = True
        return ''.join(lines)

    def test_set_single_field(self):
        """AC1: --set {slug} {field}={value} updates the specified field."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=ideation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'ideation')

    def test_set_multiple_fields(self):
        """AC2: Multiple field=value pairs in one call update all fields."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a',
                                'status=ideation', 'worktree=.worktrees/ensign-foo',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'ideation')
            self.assertEqual(fields['worktree'], '.worktrees/ensign-foo')

    def test_set_clear_field(self):
        """AC3: field= (empty value) clears the field."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'implementation', '0.80',
                                    worktree='.worktrees/ensign-foo'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'worktree=',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['worktree'], '')

    def test_set_timestamp_auto_fill(self):
        """AC4: Bare timestamp field auto-fills with current UTC ISO 8601 time."""
        import re
        from datetime import datetime, timezone
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            before = datetime.now(timezone.utc)
            result = run_status(tmpdir, '--set', 'task-a', 'started',
                                script_path=self.script_path)
            after = datetime.now(timezone.utc)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            ts = fields['started']
            # Verify ISO 8601 format
            self.assertRegex(ts, r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$')
            # Verify time is within tolerance
            parsed = datetime.strptime(ts, '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=timezone.utc)
            self.assertGreaterEqual(parsed, before.replace(microsecond=0))
            self.assertLessEqual(parsed, after.replace(microsecond=0) + __import__('datetime').timedelta(seconds=1))

    def test_set_bare_non_timestamp_error(self):
        """AC5: Bare non-timestamp field name is rejected with an error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('status', result.stderr.lower())

    def test_set_nonexistent_entity_error(self):
        """AC6: Non-zero exit and error if entity file does not exist."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            result = run_status(tmpdir, '--set', 'nonexistent', 'status=done',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('nonexistent', result.stderr.lower())

    def test_set_preserves_unmodified_fields(self):
        """AC7: Fields not specified in the command are preserved unchanged."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80', source='user'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['title'], 'Task A')
            self.assertEqual(fields['score'], '0.80')
            self.assertEqual(fields['source'], 'user')
            self.assertEqual(fields['id'], '001')

    def test_set_preserves_body(self):
        """AC8: File body content below frontmatter is preserved unchanged."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            body_before = self._read_body(os.path.join(tmpdir, 'task-a.md'))
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            body_after = self._read_body(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(body_before, body_after)

    def test_set_prints_updated_fields(self):
        """AC9: Updated fields are printed to stdout after write in `old -> new` shape."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('status: backlog -> done', result.stdout)

    def test_set_stdout_shape_non_empty_transition(self):
        """AC-3 (#159): stdout for `field=new` on existing non-empty field renders `field: old -> new`."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=ideation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('status: backlog -> ideation', result.stdout)

    def test_set_stdout_shape_clear_to_empty(self):
        """AC-3 (#159): clearing a populated field renders `field: old -> ` (empty right side, no brackets)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'implementation', '0.80',
                                    worktree='.worktrees/ensign-foo'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'worktree=',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            # Raw stdout must contain `worktree: .worktrees/ensign-foo -> \n`
            # (empty right side, no brackets, trailing space preserved).
            self.assertIn('worktree: .worktrees/ensign-foo -> \n', result.stdout)
            # No bracket syntax around the empty new-value
            self.assertNotIn('<', result.stdout)
            self.assertNotIn('empty', result.stdout.lower())

    def test_set_stdout_shape_add_missing_field(self):
        """AC-3 (#159): inserting a field absent from frontmatter renders `field:  -> new` (empty left side)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            content = textwrap.dedent("""\
                ---
                id: 001
                title: Task A
                status: backlog
                ---

                Description.
                """)
            make_pipeline(tmpdir, README_WITH_STAGES, {'task-a.md': content})
            result = run_status(tmpdir, '--set', 'task-a', 'pr=#42',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('pr:  -> #42', result.stdout)

    def test_set_stdout_shape_bare_timestamp_autofill(self):
        """AC-3 (#159): bare-timestamp auto-fill on empty field renders `field:  -> {iso-ts}`."""
        import re
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'started',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertRegex(
                result.stdout,
                r'started:\s+->\s+\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z',
            )

    def test_set_incompatible_flags(self):
        """AC10: --set is incompatible with --next, --archived, --boot, --where."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            for flag in ['--next', '--archived', '--boot', '--where']:
                args = ['--set', 'task-a', 'status=done']
                if flag == '--where':
                    args.extend([flag, 'status = backlog'])
                elif flag == '--boot':
                    args.extend([flag, 'task-a'])
                else:
                    args.append(flag)
                result = run_status(tmpdir, *args, script_path=self.script_path)
                self.assertNotEqual(result.returncode, 0,
                    f'Expected non-zero exit for --set with {flag}')
                self.assertTrue(
                    'incompatible' in result.stderr.lower() or 'cannot' in result.stderr.lower(),
                    f'Expected incompatibility error for --set with {flag}, got: {result.stderr}')

    def test_set_timestamp_skip_if_already_set(self):
        """AC11: Bare timestamp auto-fill skips if field already has a value."""
        with tempfile.TemporaryDirectory() as tmpdir:
            content = textwrap.dedent("""\
                ---
                id: 001
                title: Task A
                status: implementation
                source:
                started: 2026-01-01T00:00:00Z
                completed:
                verdict:
                score: 0.80
                worktree:
                pr:
                ---

                Description.
                """)
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': content,
            })
            result = run_status(tmpdir, '--set', 'task-a', 'started',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['started'], '2026-01-01T00:00:00Z')

    def test_set_uses_workflow_dir(self):
        """AC12: --set uses --workflow-dir to locate entity files."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            result = subprocess.run(
                ['python3', self.script_path, '--workflow-dir', tmpdir,
                 '--set', 'task-a', 'status=done'],
                capture_output=True, text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'done')

    def test_set_updates_active_worktree_copy_not_main(self):
        """Active worktree entities keep ordinary stage transitions off main."""
        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = os.path.join(tmpdir, 'repo')
            pipeline_dir = os.path.join(repo_root, 'docs', 'plans')
            os.makedirs(pipeline_dir, exist_ok=True)
            main_entity_path = os.path.join(pipeline_dir, 'task-a.md')
            worktree_dir = os.path.join(repo_root, '.worktrees', 'ensign-task-a', 'docs', 'plans')
            os.makedirs(worktree_dir, exist_ok=True)
            worktree_entity_path = os.path.join(worktree_dir, 'task-a.md')

            Path(os.path.join(repo_root, '.git')).write_text('gitdir: /tmp/fake-gitdir\n')
            make_pipeline(pipeline_dir, README_WITH_STAGES, {
                'task-a.md': entity(
                    '001', 'Task A', 'implementation', '0.80',
                    worktree='.worktrees/ensign-task-a/docs/plans'
                ),
            })
            Path(worktree_entity_path).write_text(entity(
                '001', 'Task A', 'validation', '0.80',
                worktree='.worktrees/ensign-task-a/docs/plans'
            ))

            result = subprocess.run(
                ['python3', self.script_path, '--workflow-dir', pipeline_dir,
                 '--set', 'task-a', 'status=validation'],
                capture_output=True, text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            main_fields = self._read_frontmatter(main_entity_path)
            worktree_fields = self._read_frontmatter(worktree_entity_path)
            self.assertEqual(main_fields['status'], 'implementation')
            self.assertEqual(worktree_fields['status'], 'validation')

    def test_set_mirrors_pr_to_main_and_worktree_copy(self):
        """Worktree-backed PR metadata is mirrored on main and kept in the worktree copy."""
        with tempfile.TemporaryDirectory() as tmpdir:
            repo_root = os.path.join(tmpdir, 'repo')
            pipeline_dir = os.path.join(repo_root, 'docs', 'plans')
            os.makedirs(pipeline_dir, exist_ok=True)
            main_entity_path = os.path.join(pipeline_dir, 'task-a.md')
            worktree_dir = os.path.join(repo_root, '.worktrees', 'ensign-task-a', 'docs', 'plans')
            os.makedirs(worktree_dir, exist_ok=True)
            worktree_entity_path = os.path.join(worktree_dir, 'task-a.md')

            Path(os.path.join(repo_root, '.git')).write_text('gitdir: /tmp/fake-gitdir\n')
            make_pipeline(pipeline_dir, README_WITH_STAGES, {
                'task-a.md': entity(
                    '001', 'Task A', 'implementation', '0.80',
                    worktree='.worktrees/ensign-task-a/docs/plans'
                ),
            })
            Path(worktree_entity_path).write_text(entity(
                '001', 'Task A', 'validation', '0.80',
                worktree='.worktrees/ensign-task-a/docs/plans'
            ))

            result = subprocess.run(
                ['python3', self.script_path, '--workflow-dir', pipeline_dir,
                 '--set', 'task-a', 'pr=#42'],
                capture_output=True, text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            main_fields = self._read_frontmatter(main_entity_path)
            worktree_fields = self._read_frontmatter(worktree_entity_path)
            self.assertEqual(main_fields['pr'], '#42')
            self.assertEqual(worktree_fields['pr'], '#42')
            self.assertEqual(main_fields['status'], 'implementation')
            self.assertEqual(worktree_fields['status'], 'validation')


class TestStatusScriptExecutable(unittest.TestCase):
    """Regression: the status script file must have the executable bit set."""

    def test_status_script_is_executable(self):
        self.assertTrue(os.path.exists(TEMPLATE_PATH), f"{TEMPLATE_PATH} not found")
        self.assertTrue(
            os.access(TEMPLATE_PATH, os.X_OK),
            f"{TEMPLATE_PATH} is not executable",
        )


def entity_with_custom(id, title, status, score='', **custom):
    """Generate entity frontmatter with arbitrary extra fields."""
    lines = [
        '---',
        f'id: {id}',
        f'title: {title}',
        f'status: {status}',
        'source:',
        'started:',
        'completed:',
        'verdict:',
        f'score: {score}',
        'worktree:',
        'pr:',
    ]
    for key, val in custom.items():
        lines.append(f'{key}: {val}')
    lines.append('---')
    lines.append('')
    lines.append('Description.')
    lines.append('')
    return '\n'.join(lines)


class TestFieldsOption(unittest.TestCase):
    """Test --fields and --all-fields for appending extra frontmatter columns."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _header(self, stdout):
        return stdout.strip().split('\n')[0]

    # AC6: --fields appends requested fields in user-specified order
    def test_fields_appends_in_order(self):
        """--fields pr,worktree adds PR and WORKTREE after SOURCE, in that order."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'A', 'backlog', '0.80', pr='#10', worktree='.worktrees/a'),
                'b.md': entity('002', 'B', 'backlog', '0.70'),
            })
            result = run_status(tmpdir, '--fields', 'pr,worktree', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            header = self._header(result.stdout)
            # Default columns must still be present in order
            self.assertIn('ID', header)
            self.assertIn('SLUG', header)
            self.assertIn('STATUS', header)
            self.assertIn('TITLE', header)
            self.assertIn('SCORE', header)
            self.assertIn('SOURCE', header)
            # Extras are after SOURCE in the requested order
            source_idx = header.index('SOURCE')
            pr_idx = header.index('PR', source_idx)
            wt_idx = header.index('WORKTREE', source_idx)
            self.assertGreater(pr_idx, source_idx)
            self.assertGreater(wt_idx, pr_idx)
            # Values appear in the data rows
            self.assertIn('#10', result.stdout)
            self.assertIn('.worktrees/a', result.stdout)

    # AC6: missing fields render as empty in extra columns
    def test_fields_missing_renders_empty(self):
        """--fields pr on an entity without a PR still renders a column, empty for that row."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'has.md': entity('001', 'Has PR', 'backlog', '0.80', pr='#10'),
                'none.md': entity('002', 'No PR', 'backlog', '0.70'),
            })
            result = run_status(tmpdir, '--fields', 'pr', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')[2:]
            has_line = [l for l in lines if 'has' in l][0]
            none_line = [l for l in lines if 'none' in l][0]
            self.assertIn('#10', has_line)
            # The 'none' row must not contain '#10' and must not inherit another
            # row's PR value.
            self.assertNotIn('#10', none_line)

    # AC7: --fields works on custom frontmatter fields
    def test_fields_custom_field_populated(self):
        """--fields last-outbound-at includes the custom field as a populated column."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity_with_custom('001', 'A', 'backlog', '0.80',
                                           **{'last-outbound-at': '2026-04-01'}),
                'b.md': entity_with_custom('002', 'B', 'backlog', '0.70',
                                           **{'last-outbound-at': '2026-04-02'}),
            })
            result = run_status(tmpdir, '--fields', 'last-outbound-at',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            header = self._header(result.stdout)
            self.assertIn('LAST-OUTBOUND-AT', header)
            self.assertIn('2026-04-01', result.stdout)
            self.assertIn('2026-04-02', result.stdout)

    # AC8: --fields on a nonexistent field does NOT error
    def test_fields_nonexistent_no_error(self):
        """--fields made-up-field adds an empty column; exits 0."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--fields', 'made-up-field',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            header = self._header(result.stdout)
            self.assertIn('MADE-UP-FIELD', header)

    # AC9: --all-fields appends every non-empty custom field, sorted, deduped vs defaults
    def test_all_fields_sorted_dedup(self):
        """--all-fields collects custom fields across all entities in sorted order."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity_with_custom('001', 'A', 'backlog', '0.80',
                                           **{'last-outbound-at': '2026-04-01'}),
                'b.md': entity_with_custom('002', 'B', 'backlog', '0.70',
                                           **{'nudge-count': '2'}),
            })
            result = run_status(tmpdir, '--all-fields', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            header = self._header(result.stdout)
            # Default columns present
            for col in ('ID', 'SLUG', 'STATUS', 'TITLE', 'SCORE', 'SOURCE'):
                self.assertIn(col, header)
            # Custom fields present and in sorted order (LAST-OUTBOUND-AT < NUDGE-COUNT)
            source_idx = header.index('SOURCE')
            last_idx = header.index('LAST-OUTBOUND-AT', source_idx)
            nudge_idx = header.index('NUDGE-COUNT', source_idx)
            self.assertGreater(last_idx, source_idx)
            self.assertLess(last_idx, nudge_idx)
            # Default columns must not be duplicated after the default block
            # (check by counting occurrences of 'STATUS')
            self.assertEqual(header.count('STATUS'), 1)
            self.assertEqual(header.count('TITLE'), 1)

    # AC10: --fields and --all-fields are mutually exclusive
    def test_fields_and_all_fields_conflict(self):
        """Passing both --fields and --all-fields exits non-zero with an error."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--fields', 'pr', '--all-fields',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mutually exclusive', result.stderr)

    # AC11: --fields composes with --next
    def test_fields_composes_with_next(self):
        """--next --fields pr appends the pr column to the next-table output."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'ready.md': entity('001', 'Ready', 'backlog', '0.80', pr='#42'),
            })
            result = run_status(tmpdir, '--next', '--fields', 'pr',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            header = self._header(result.stdout)
            # Default --next columns still present
            self.assertIn('CURRENT', header)
            self.assertIn('NEXT', header)
            self.assertIn('WORKTREE', header)
            # Extra PR column appended
            self.assertIn('PR', header)
            # Row contains the pr value
            self.assertIn('#42', result.stdout)

    # AC12: --fields with --boot errors
    def test_fields_incompatible_with_boot(self):
        """--boot --fields exits non-zero."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--boot', '--fields', 'pr',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('--boot', result.stderr)

    # AC12: --all-fields with --boot errors
    def test_all_fields_incompatible_with_boot(self):
        """--boot --all-fields exits non-zero."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--boot', '--all-fields',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('--boot', result.stderr)

    # AC13: --fields with --set errors
    def test_fields_incompatible_with_set(self):
        """--set ... --fields exits non-zero."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'a.md': entity('001', 'A', 'backlog', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'a', 'status=ideation',
                                '--fields', 'pr', script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('--set', result.stderr)


class TestArchiveOption(unittest.TestCase):
    """Test --archive subcommand."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _read_frontmatter(self, filepath):
        fields = {}
        in_fm = False
        with open(filepath, 'r') as f:
            for line in f:
                line = line.rstrip('\n')
                if line == '---':
                    if in_fm:
                        break
                    in_fm = True
                    continue
                if in_fm and ':' in line:
                    key, _, val = line.partition(':')
                    fields[key.strip()] = val.strip()
        return fields

    # AC14: --archive moves the file and stamps archived:
    def test_archive_moves_and_stamps(self):
        """--archive moves {slug}.md to _archive/ and inserts an ISO-8601 archived: stamp."""
        import re
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80'),
            })
            source = os.path.join(tmpdir, 'task-a.md')
            dest = os.path.join(tmpdir, '_archive', 'task-a.md')
            self.assertTrue(os.path.exists(source))
            self.assertFalse(os.path.exists(dest))
            result = run_status(tmpdir, '--archive', 'task-a', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse(os.path.exists(source))
            self.assertTrue(os.path.exists(dest))
            self.assertTrue(os.path.isdir(os.path.join(tmpdir, '_archive')))
            fields = self._read_frontmatter(dest)
            self.assertIn('archived', fields)
            self.assertRegex(fields['archived'], r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$')
            self.assertIn('task-a', result.stdout)

    # AC15: --archive errors on missing source
    def test_archive_missing_source_errors(self):
        """--archive on a nonexistent slug exits non-zero with 'entity not found'."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            result = run_status(tmpdir, '--archive', 'no-such-slug',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('entity not found', result.stderr)

    # AC16: --archive errors if the destination already exists
    def test_archive_existing_destination_errors(self):
        """--archive refuses to clobber an existing file in _archive/."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                README_WITH_STAGES,
                entities={'task-a.md': entity('001', 'Task A', 'done', '0.80')},
                archived={'task-a.md': entity('001', 'Old Task A', 'done', '0.90')},
            )
            source = os.path.join(tmpdir, 'task-a.md')
            dest = os.path.join(tmpdir, '_archive', 'task-a.md')
            with open(source) as f:
                source_before = f.read()
            with open(dest) as f:
                dest_before = f.read()
            result = run_status(tmpdir, '--archive', 'task-a',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('already archived', result.stderr)
            # Both files still present, unchanged
            self.assertTrue(os.path.exists(source))
            self.assertTrue(os.path.exists(dest))
            with open(source) as f:
                self.assertEqual(f.read(), source_before)
            with open(dest) as f:
                self.assertEqual(f.read(), dest_before)

    # AC17: --archive does not touch `completed`
    def test_archive_preserves_completed(self):
        """Entity with a completed value keeps it, and one without does not gain one."""
        with tempfile.TemporaryDirectory() as tmpdir:
            prestamped = textwrap.dedent("""\
                ---
                id: 001
                title: Task A
                status: done
                source:
                started:
                completed: 2026-01-01T00:00:00Z
                verdict:
                score: 0.80
                worktree:
                pr:
                ---

                Body.
                """)
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': prestamped,
                'task-b.md': entity('002', 'Task B', 'backlog', '0.70'),
            })
            r1 = run_status(tmpdir, '--archive', 'task-a', script_path=self.script_path)
            self.assertEqual(r1.returncode, 0, r1.stderr)
            fields_a = self._read_frontmatter(os.path.join(tmpdir, '_archive', 'task-a.md'))
            self.assertEqual(fields_a['completed'], '2026-01-01T00:00:00Z')

            r2 = run_status(tmpdir, '--archive', 'task-b', script_path=self.script_path)
            self.assertEqual(r2.returncode, 0, r2.stderr)
            fields_b = self._read_frontmatter(os.path.join(tmpdir, '_archive', 'task-b.md'))
            # `completed` is an existing (but empty) frontmatter field in the
            # entity() helper. The tool must not fill it in.
            self.assertEqual(fields_b.get('completed', ''), '')

    # AC18: --archive does not run git — changes are left pending
    def test_archive_does_not_commit(self):
        """git status reports the move as uncommitted after --archive."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80'),
            })
            # Initialize a local git repo in the tmpdir
            subprocess.run(['git', 'init', '-q'], cwd=tmpdir, check=True)
            subprocess.run(['git', 'config', 'user.email', 'test@example.com'], cwd=tmpdir, check=True)
            subprocess.run(['git', 'config', 'user.name', 'Test'], cwd=tmpdir, check=True)
            subprocess.run(['git', 'add', '.'], cwd=tmpdir, check=True)
            subprocess.run(['git', 'commit', '-q', '-m', 'initial'], cwd=tmpdir, check=True)

            result = run_status(tmpdir, '--archive', 'task-a', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

            porcelain = subprocess.run(
                ['git', 'status', '--porcelain'], cwd=tmpdir,
                capture_output=True, text=True, check=True,
            )
            # Expect pending changes — the tool must not have committed for us.
            self.assertNotEqual(porcelain.stdout.strip(), '',
                                f'expected pending changes, got: {porcelain.stdout!r}')
            # Confirm we did not run into a fresh commit
            log = subprocess.run(
                ['git', 'log', '--oneline'], cwd=tmpdir,
                capture_output=True, text=True, check=True,
            )
            self.assertEqual(len(log.stdout.strip().split('\n')), 1,
                             f'expected single initial commit, got: {log.stdout!r}')


class TestStatusDocstring(unittest.TestCase):
    """Static check: the status script header must document the new flags."""

    def test_docstring_mentions_new_flags(self):
        """Script header names --where (with or without spaces), --fields, --all-fields, --archive."""
        with open(TEMPLATE_PATH, 'r') as f:
            # Read header only — stop at the first non-comment, non-shebang line.
            header_lines = []
            for line in f:
                if line.startswith('#') or line.strip() == '':
                    header_lines.append(line)
                else:
                    break
            header = ''.join(header_lines)
        self.assertIn('--where', header)
        self.assertIn('--fields', header)
        self.assertIn('--all-fields', header)
        self.assertIn('--archive', header)
        self.assertIn('with or without spaces', header)


SHARED_CORE_PATH = os.path.join(
    SCRIPT_DIR, '..', 'skills', 'first-officer', 'references', 'first-officer-shared-core.md',
)
CODEX_RUNTIME_PATH = os.path.join(
    SCRIPT_DIR, '..', 'skills', 'first-officer', 'references', 'codex-first-officer-runtime.md',
)


def make_workflow_readme(commissioned_by='spacedock@1.0'):
    """Return a minimal README.md with commissioned-by frontmatter."""
    return textwrap.dedent(f"""\
        ---
        commissioned-by: {commissioned_by}
        entity-type: task
        ---

        # Workflow
        """)


class TestDiscover(unittest.TestCase):
    """Tests for --discover workflow directory discovery."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_discover_single_workflow(self):
        """--discover with one workflow dir outputs that path and exits 0."""
        with tempfile.TemporaryDirectory() as tmpdir:
            wf_dir = os.path.join(tmpdir, 'docs', 'plans')
            os.makedirs(wf_dir)
            with open(os.path.join(wf_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme())

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 1)
            self.assertEqual(lines[0], os.path.realpath(wf_dir))

    def test_discover_no_workflows(self):
        """--discover with no workflows outputs nothing and exits 0."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), '')

    def test_discover_multiple_workflows(self):
        """--discover with multiple workflows outputs paths alphabetically."""
        with tempfile.TemporaryDirectory() as tmpdir:
            for name in ('beta-workflow', 'alpha-workflow'):
                wf_dir = os.path.join(tmpdir, name)
                os.makedirs(wf_dir)
                with open(os.path.join(wf_dir, 'README.md'), 'w') as f:
                    f.write(make_workflow_readme())

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 2)
            self.assertIn('alpha-workflow', lines[0])
            self.assertIn('beta-workflow', lines[1])

    def test_discover_ignores_excluded_dirs(self):
        """--discover skips directories in the ignore list."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Valid workflow
            valid_dir = os.path.join(tmpdir, 'valid')
            os.makedirs(valid_dir)
            with open(os.path.join(valid_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme())

            # Workflow inside an ignored directory
            for ignored in ('tests', 'node_modules', '.worktrees', 'vendor',
                            'dist', 'build', '__pycache__'):
                ignored_dir = os.path.join(tmpdir, ignored, 'fixtures')
                os.makedirs(ignored_dir, exist_ok=True)
                with open(os.path.join(ignored_dir, 'README.md'), 'w') as f:
                    f.write(make_workflow_readme())

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 1)
            self.assertIn('valid', lines[0])

    def test_discover_skips_non_spacedock_readme(self):
        """--discover skips READMEs not commissioned by spacedock@."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Non-spacedock workflow
            other_dir = os.path.join(tmpdir, 'other')
            os.makedirs(other_dir)
            with open(os.path.join(other_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme(commissioned_by='other@1.0'))

            # No commissioned-by at all
            bare_dir = os.path.join(tmpdir, 'bare')
            os.makedirs(bare_dir)
            with open(os.path.join(bare_dir, 'README.md'), 'w') as f:
                f.write(textwrap.dedent("""\
                    ---
                    entity-type: task
                    ---

                    # Not a workflow
                    """))

            # Bare spacedock@ (no version suffix) — SHOULD match per staff-review #1
            bare_version_dir = os.path.join(tmpdir, 'bare-version')
            os.makedirs(bare_version_dir)
            with open(os.path.join(bare_version_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme(commissioned_by='spacedock@'))

            # Valid spacedock workflow for positive control
            valid_dir = os.path.join(tmpdir, 'valid')
            os.makedirs(valid_dir)
            with open(os.path.join(valid_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme(commissioned_by='spacedock@2.0'))

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            # bare-version (spacedock@) and valid (spacedock@2.0) should match
            self.assertEqual(len(lines), 2, f'expected 2 matches, got: {lines}')
            self.assertIn('bare-version', lines[0])
            self.assertIn('valid', lines[1])

    def test_discover_incompatible_flags(self):
        """--discover errors when combined with any incompatible flag."""
        incompatible = [
            ['--boot'],
            ['--next'],
            ['--next-id'],
            ['--archived'],
            ['--where', 'status=backlog'],
            ['--set', 'slug', 'field=val'],
            ['--archive', 'slug'],
            ['--fields', 'id,title'],
            ['--all-fields'],
            ['--workflow-dir', '/tmp'],
        ]
        with tempfile.TemporaryDirectory() as tmpdir:
            for flags in incompatible:
                with self.subTest(flags=flags):
                    result = run_status(tmpdir, '--discover', *flags,
                                        script_path=self.script_path)
                    self.assertEqual(result.returncode, 1,
                                     f'expected rc=1 for --discover + {flags}, '
                                     f'got rc={result.returncode}, stderr={result.stderr!r}')
                    self.assertIn('Error', result.stderr,
                                  f'expected Error in stderr for {flags}')

    def test_discover_bad_root(self):
        """--discover --root /nonexistent errors with exit 1."""
        with tempfile.TemporaryDirectory() as tmpdir:
            result = run_status(tmpdir, '--discover', '--root', '/nonexistent/path',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 1)
            self.assertIn('Error', result.stderr)

    def test_discover_default_root(self):
        """--discover without --root defaults to git toplevel."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Initialize a git repo
            subprocess.run(['git', 'init'], cwd=tmpdir,
                           capture_output=True, check=True)
            subprocess.run(['git', 'config', 'user.email', 'test@test.com'], cwd=tmpdir,
                           capture_output=True, check=True)
            subprocess.run(['git', 'config', 'user.name', 'Test'], cwd=tmpdir,
                           capture_output=True, check=True)

            # Create a workflow subdir
            wf_dir = os.path.join(tmpdir, 'docs', 'plans')
            os.makedirs(wf_dir)
            with open(os.path.join(wf_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme())

            # Create an initial commit so git rev-parse works
            subprocess.run(['git', 'add', '.'], cwd=tmpdir,
                           capture_output=True, check=True)
            subprocess.run(['git', 'commit', '-m', 'init'], cwd=tmpdir,
                           capture_output=True, check=True)

            # Run from a subdirectory within the repo
            sub = os.path.join(tmpdir, 'docs')
            result = subprocess.run(
                ['python3', self.script_path, '--discover'],
                capture_output=True, text=True, cwd=sub,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 1)
            self.assertEqual(lines[0], os.path.realpath(wf_dir))

    def test_discover_deduplicates_symlinks(self):
        """Symlinked directories are discovered and deduplicated; canonical path returned."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Real workflow directory
            real_dir = os.path.join(tmpdir, 'real-workflow')
            os.makedirs(real_dir)
            with open(os.path.join(real_dir, 'README.md'), 'w') as f:
                f.write(make_workflow_readme())

            # Symlink pointing to the real directory
            link_path = os.path.join(tmpdir, 'link-workflow')
            os.symlink(real_dir, link_path)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            self.assertEqual(len(lines), 1, f'expected 1 path after dedup, got: {lines}')
            # Must be the canonical (realpath) path, not the symlink
            self.assertEqual(lines[0], os.path.realpath(real_dir))

    def test_discover_prose_shared_core(self):
        """Shared core step 2 references status --discover and omits old grep prose."""
        with open(SHARED_CORE_PATH, 'r') as f:
            content = f.read()
        self.assertIn('status --discover', content)
        self.assertNotIn('search for `README.md` files', content)

    def test_discover_prose_codex_runtime(self):
        """Codex runtime Workflow Target references status --discover and omits old prose."""
        with open(CODEX_RUNTIME_PATH, 'r') as f:
            content = f.read()
        self.assertIn('status --discover', content)
        self.assertNotIn('discover candidate workflows from the current repository', content)


class TestModBlockGuard(unittest.TestCase):
    """Test mod-block enforcement in --set and --archive."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _read_frontmatter(self, filepath):
        fields = {}
        in_fm = False
        with open(filepath, 'r') as f:
            for line in f:
                line = line.rstrip('\n')
                if line == '---':
                    if in_fm:
                        break
                    in_fm = True
                    continue
                if in_fm and ':' in line:
                    key, _, val = line.partition(':')
                    fields[key.strip()] = val.strip()
        return fields

    @staticmethod
    def _blocked_entity(id, title, status, mod_block='merge:pr-merge',
                        score='', worktree='', pr=''):
        return textwrap.dedent(f"""\
            ---
            id: {id}
            title: {title}
            status: {status}
            source:
            started:
            completed:
            verdict:
            score: {score}
            worktree: {worktree}
            pr: {pr}
            mod-block: {mod_block}
            ---

            Description.
            """)

    def test_modblock_guard_refuses_terminal_status(self):
        """--set slug status=done on entity with mod-block exits 1."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)
            self.assertIn('merge:pr-merge', result.stderr)

    def test_modblock_guard_refuses_completed(self):
        """--set slug completed on mod-blocked entity exits 1."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'completed',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)

    def test_modblock_guard_refuses_verdict(self):
        """--set slug verdict=PASSED on mod-blocked entity exits 1."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'verdict=PASSED',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)

    def test_modblock_guard_refuses_worktree_clear(self):
        """--set slug worktree= on mod-blocked entity exits 1."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation',
                                                   worktree='.worktrees/ensign-task-a'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'worktree=',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)

    def test_modblock_guard_allows_pr_update(self):
        """--set slug pr=#57 on mod-blocked entity succeeds."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'pr=#57',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['pr'], '#57')

    def test_modblock_guard_allows_nonterminal_status(self):
        """--set slug status=implementation on mod-blocked entity succeeds."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'ideation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=implementation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'implementation')

    def test_modblock_force_overrides_guard(self):
        """--set slug status=done --force on mod-blocked entity succeeds with warning."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done', '--force',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('Warning', result.stderr)
            self.assertIn('mod-block', result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'done')

    def test_modblock_archive_guard(self):
        """--archive slug on mod-blocked entity exits 1."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'done'),
            })
            result = run_status(tmpdir, '--archive', 'task-a',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)

    def test_modblock_archive_force_overrides(self):
        """--archive slug --force on mod-blocked entity succeeds with warning."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'done'),
            })
            result = run_status(tmpdir, '--archive', 'task-a', '--force',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('Warning', result.stderr)
            dest = os.path.join(tmpdir, '_archive', 'task-a.md')
            self.assertTrue(os.path.exists(dest))

    def test_modblock_set_and_clear(self):
        """mod-block field is settable and clearable via --set."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            # Set mod-block
            result = run_status(tmpdir, '--set', 'task-a', 'mod-block=merge:pr-merge',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['mod-block'], 'merge:pr-merge')

            # Clear mod-block
            result = run_status(tmpdir, '--set', 'task-a', 'mod-block=',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['mod-block'], '')

    def test_absent_modblock_treated_as_no_block(self):
        """Entity without mod-block field allows terminal transitions freely."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'done')

    def test_empty_modblock_treated_as_no_block(self):
        """Entity with empty mod-block field allows terminal transitions."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation',
                                                   mod_block=''),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_modblock_guard_refuses_combined_clear_with_terminal(self):
        """Clearing mod-block and setting terminal fields in one --set refuses."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a',
                                'mod-block=', 'verdict=PASSED', 'worktree=',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)
            self.assertIn('separate', result.stderr)
            # Nothing was written — the frontmatter must still show the block.
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['mod-block'], 'merge:pr-merge')
            self.assertEqual(fields.get('verdict', ''), '')

    def test_modblock_guard_refuses_combined_clear_with_status_done(self):
        """Clearing mod-block and advancing to terminal status in one --set refuses."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a',
                                'mod-block=', 'status=done',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('mod-block', result.stderr)

    def test_modblock_guard_allows_standalone_clear(self):
        """Clearing mod-block on its own is allowed."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'mod-block=',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['mod-block'], '')

    def test_modblock_force_overrides_combined_clear_guard(self):
        """--force permits clearing mod-block together with terminal fields."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': self._blocked_entity('001', 'Task A', 'validation'),
            })
            result = run_status(tmpdir, '--set', 'task-a',
                                'mod-block=', 'verdict=PASSED', 'worktree=',
                                '--force', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['mod-block'], '')
            self.assertEqual(fields['verdict'], 'PASSED')


PR_MERGE_HOOK = textwrap.dedent("""\
    ---
    name: pr-merge
    description: test merge hook
    ---

    # PR Merge

    ## Hook: merge

    Push branch and create PR.
    """)


def _add_merge_hook(pipeline_dir, mod_name='pr-merge', content=PR_MERGE_HOOK):
    """Drop a merge-hook mod into {pipeline_dir}/_mods/."""
    mods_dir = os.path.join(pipeline_dir, '_mods')
    os.makedirs(mods_dir, exist_ok=True)
    with open(os.path.join(mods_dir, f'{mod_name}.md'), 'w') as f:
        f.write(content)


class TestMergeHookTerminalGuard(unittest.TestCase):
    """Mechanism-level enforcement: terminal transitions require merge-hook completion."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _read_frontmatter(self, filepath):
        fields = {}
        in_fm = False
        with open(filepath, 'r') as f:
            for line in f:
                line = line.rstrip('\n')
                if line == '---':
                    if in_fm:
                        break
                    in_fm = True
                    continue
                if in_fm and ':' in line:
                    key, _, val = line.partition(':')
                    fields[key.strip()] = val.strip()
        return fields

    def test_set_status_done_refused_when_merge_hook_and_no_pr(self):
        """Terminal status transition refused when merge hook exists and pr/mod-block empty."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('merge hook', result.stderr)
            self.assertIn('pr-merge', result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'validation')

    def test_set_completed_refused_when_merge_hook_and_no_pr(self):
        """Setting completed refused when merge hook exists and pr/mod-block empty."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'completed',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('merge hook', result.stderr)

    def test_set_verdict_refused_when_merge_hook_and_no_pr(self):
        """Setting verdict refused when merge hook exists and pr/mod-block empty."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'verdict=PASSED',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('merge hook', result.stderr)

    def test_set_worktree_clear_refused_when_merge_hook_and_no_pr(self):
        """Clearing worktree refused when merge hook exists and pr/mod-block empty."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80',
                                    worktree='.worktrees/ensign-task-a'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'worktree=',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('merge hook', result.stderr)

    def test_set_terminal_allowed_when_pr_set(self):
        """Terminal transition permitted when pr field is non-empty (hook ran)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80', pr='#57'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'done')

    def test_set_terminal_allowed_when_pr_set_in_same_call(self):
        """Terminal transition permitted when pr is being set in the same --set."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a',
                                'pr=#57', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'done')
            self.assertEqual(fields['pr'], '#57')

    def test_set_force_bypasses_merge_hook_guard(self):
        """--force bypasses merge-hook guard even with empty pr/mod-block."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'status=done', '--force',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_set_terminal_allowed_without_merge_hook(self):
        """Workflow without merge hooks — terminal transition permitted freely."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_set_nonterminal_allowed_with_merge_hook(self):
        """Non-terminal transitions unaffected by merge hook guard."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'backlog', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'status=implementation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_set_pr_only_allowed_with_merge_hook(self):
        """Setting pr alone unaffected by merge hook guard (not a terminal field)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--set', 'task-a', 'pr=#57',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_archive_refused_when_merge_hook_and_no_pr(self):
        """--archive refused when merge hook exists and pr/mod-block empty."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--archive', 'task-a',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('merge hook', result.stderr)
            self.assertTrue(os.path.exists(os.path.join(tmpdir, 'task-a.md')))

    def test_archive_allowed_when_pr_set(self):
        """--archive permitted when pr is set (hook ran)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80', pr='#57'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--archive', 'task-a',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(
                os.path.exists(os.path.join(tmpdir, '_archive', 'task-a.md'))
            )

    def test_archive_force_bypasses_guard(self):
        """--archive --force bypasses the merge-hook invariant."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'done', '0.80'),
            })
            _add_merge_hook(tmpdir)
            result = run_status(tmpdir, '--archive', 'task-a', '--force',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_non_merge_hook_does_not_trigger_guard(self):
        """Only merge hooks trigger the invariant — startup/idle hooks are ignored."""
        startup_only = textwrap.dedent("""\
            ---
            name: pr-scanner
            ---

            # PR Scanner

            ## Hook: startup

            Scan PRs.
            """)
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': entity('001', 'Task A', 'validation', '0.80'),
            })
            _add_merge_hook(tmpdir, mod_name='pr-scanner', content=startup_only)
            result = run_status(tmpdir, '--set', 'task-a', 'status=done',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)


def make_folder_entity(tmpdir, slug, content):
    """Create a folder-form entity: {tmpdir}/{slug}/index.md with the given content."""
    folder = os.path.join(tmpdir, slug)
    os.makedirs(folder, exist_ok=True)
    with open(os.path.join(folder, 'index.md'), 'w') as f:
        f.write(content)
    return folder


def _read_frontmatter(filepath):
    """Read frontmatter from a file, returning a dict of fields."""
    fields = {}
    in_fm = False
    with open(filepath, 'r') as f:
        for line in f:
            line = line.rstrip('\n')
            if line == '---':
                if in_fm:
                    break
                in_fm = True
                continue
            if in_fm and ':' in line:
                key, _, val = line.partition(':')
                fields[key.strip()] = val.strip()
    return fields


class TestEntityAsFolder(unittest.TestCase):
    """Test first-class entity-as-folder support (issue #99).

    An entity may now live either as a flat `{slug}.md` file or as a folder
    `{slug}/` containing `index.md`. Reserved subdirectories (`_archive`,
    `_mods`) are never treated as entity folders.
    """

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    # ---------- Discovery: mixed flat + folder ----------

    def test_default_overview_lists_folder_entity(self):
        """Folder entity (`{slug}/index.md`) appears in the default overview."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'backlog', '0.80'))
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('folder-entity', result.stdout)
            self.assertIn('Folder Entity', result.stdout)

    def test_mixed_workflow_default_overview_shows_both(self):
        """Flat and folder entities coexist in the default overview."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'flat-entity.md': entity('001', 'Flat Entity', 'backlog', '0.90'),
            })
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('002', 'Folder Entity', 'backlog', '0.80'))
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('flat-entity', result.stdout)
            self.assertIn('folder-entity', result.stdout)

    def test_folder_only_workflow_default_overview(self):
        """Folder-only workflow: all entities still appear."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'post-one',
                               entity('001', 'Post One', 'backlog', '0.70'))
            make_folder_entity(tmpdir, 'post-two',
                               entity('002', 'Post Two', 'ideation', '0.90'))
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = result.stdout.strip().split('\n')
            # header + separator + 2 rows
            self.assertEqual(len(lines), 4, result.stdout)
            self.assertIn('post-one', result.stdout)
            self.assertIn('post-two', result.stdout)

    def test_next_includes_folder_entity(self):
        """--next includes folder entities that are dispatchable."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'backlog', '0.80'))
            result = run_status(tmpdir, '--next', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('folder-entity', result.stdout)

    def test_next_id_counts_folder_entities(self):
        """--next-id considers folder entities' ids."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'flat-a.md': entity('003', 'Flat A', 'backlog', '0.40'),
            })
            make_folder_entity(tmpdir, 'folder-b',
                               entity('007', 'Folder B', 'backlog', '0.70'))
            result = run_status(tmpdir, '--next-id', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), '008')

    def test_boot_reports_folder_entities(self):
        """--boot renders folder entities in the DISPATCHABLE section."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'backlog', '0.80'))
            result = run_status(tmpdir, '--boot', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            # NEXT_ID accounts for the folder entity id=001 -> next 002.
            self.assertIn('NEXT_ID: 002', result.stdout)
            # Folder entity dispatchable row is present.
            self.assertIn('folder-entity', result.stdout)

    def test_where_filter_applies_to_folder_entities(self):
        """--where matches on folder-entity fields."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'backlog', '0.80'))
            make_folder_entity(tmpdir, 'folder-other',
                               entity('002', 'Folder Other', 'ideation', '0.80'))
            result = run_status(tmpdir, '--where', 'status=backlog',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('folder-entity', result.stdout)
            self.assertNotIn('folder-other', result.stdout)

    # ---------- Reserved subdirectories ----------

    def test_reserved_archive_subdir_not_treated_as_entity(self):
        """A `_archive/` directory must never surface as an entity."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                README_WITH_STAGES,
                entities={'flat-a.md': entity('001', 'Flat A', 'backlog', '0.50')},
                archived={'old.md': entity('002', 'Old', 'done', '0.80')},
            )
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            # `_archive` should never appear as a slug in the overview.
            self.assertNotIn('_archive', result.stdout)
            self.assertIn('flat-a', result.stdout)

    def test_reserved_mods_subdir_not_treated_as_entity(self):
        """A `_mods/` directory must never surface as an entity."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'flat-a.md': entity('001', 'Flat A', 'backlog', '0.50'),
            })
            # Create a `_mods/` with an index.md — it must still be treated as
            # reserved, not as a `_mods` entity.
            mods_dir = os.path.join(tmpdir, '_mods')
            os.makedirs(mods_dir)
            with open(os.path.join(mods_dir, 'index.md'), 'w') as f:
                f.write('---\nname: not-an-entity\n---\n')
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            # Only the flat entity shows in the SLUG column.
            self.assertIn('flat-a', result.stdout)
            # `_mods` must not appear as a slug (rely on the row format —
            # the substring must not match an ID or slug cell).
            for line in result.stdout.strip().split('\n')[2:]:
                cells = line.split()
                self.assertNotIn('_mods', cells)

    # ---------- Conflict: both flat and folder present ----------

    def test_conflict_prefers_folder_and_warns(self):
        """When both flat and folder exist, folder wins and stderr warns."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'dup.md': entity('001', 'Flat Copy', 'backlog', '0.10'),
            })
            make_folder_entity(tmpdir, 'dup',
                               entity('001', 'Folder Copy', 'backlog', '0.90'))
            result = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            # The folder copy's title wins.
            self.assertIn('Folder Copy', result.stdout)
            self.assertNotIn('Flat Copy', result.stdout)
            # Warning on stderr, not stdout.
            self.assertIn("entity 'dup'", result.stderr)
            self.assertIn('preferring folder', result.stderr)
            # Exactly one row in the overview (dedup by slug).
            data_rows = [
                line for line in result.stdout.strip().split('\n')[2:]
                if 'dup' in line
            ]
            self.assertEqual(len(data_rows), 1, result.stdout)

    # ---------- --set on a folder entity ----------

    def test_set_on_folder_entity_writes_to_index(self):
        """--set writes to `{slug}/index.md`, not a sibling file."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'backlog', '0.80'))
            result = run_status(tmpdir, '--set', 'folder-entity', 'status=ideation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            # index.md updated
            fields = _read_frontmatter(
                os.path.join(tmpdir, 'folder-entity', 'index.md'))
            self.assertEqual(fields['status'], 'ideation')
            # No sibling `folder-entity.md` created
            self.assertFalse(
                os.path.exists(os.path.join(tmpdir, 'folder-entity.md')))

    def test_set_on_missing_folder_entity_errors(self):
        """--set on a slug with neither flat nor folder form errors."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            result = run_status(tmpdir, '--set', 'no-such', 'status=ideation',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('entity not found', result.stderr)

    def test_set_prefers_folder_when_both_forms_exist(self):
        """--set on a conflicting slug writes to the folder copy and warns."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'dup.md': entity('001', 'Flat Copy', 'backlog', '0.10'),
            })
            make_folder_entity(tmpdir, 'dup',
                               entity('001', 'Folder Copy', 'backlog', '0.90'))
            result = run_status(tmpdir, '--set', 'dup', 'status=ideation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            folder_fields = _read_frontmatter(
                os.path.join(tmpdir, 'dup', 'index.md'))
            flat_fields = _read_frontmatter(os.path.join(tmpdir, 'dup.md'))
            self.assertEqual(folder_fields['status'], 'ideation')
            self.assertEqual(flat_fields['status'], 'backlog')
            self.assertIn('preferring folder', result.stderr)

    # ---------- --archive on a folder entity ----------

    def test_archive_folder_entity_moves_whole_directory(self):
        """--archive on a folder entity moves the whole folder into _archive/."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            folder = make_folder_entity(
                tmpdir, 'folder-entity',
                entity('001', 'Folder Entity', 'done', '0.80'))
            # Add an artifact next to index.md to verify the whole folder moves.
            with open(os.path.join(folder, 'draft-v1.md'), 'w') as f:
                f.write('# draft v1\n')

            result = run_status(tmpdir, '--archive', 'folder-entity',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)

            # Original folder gone
            self.assertFalse(os.path.exists(folder))
            # Archived folder present with both files
            archived_folder = os.path.join(tmpdir, '_archive', 'folder-entity')
            self.assertTrue(os.path.isdir(archived_folder))
            self.assertTrue(os.path.isfile(os.path.join(archived_folder, 'index.md')))
            self.assertTrue(os.path.isfile(os.path.join(archived_folder, 'draft-v1.md')))
            # No stray sibling .md file created in _archive
            self.assertFalse(
                os.path.exists(os.path.join(tmpdir, '_archive', 'folder-entity.md')))

    def test_archive_folder_entity_stamps_archived_in_index(self):
        """--archive stamps `archived:` into the inner index.md before moving."""
        import re
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'done', '0.80'))
            result = run_status(tmpdir, '--archive', 'folder-entity',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            archived_index = os.path.join(
                tmpdir, '_archive', 'folder-entity', 'index.md')
            fields = _read_frontmatter(archived_index)
            self.assertIn('archived', fields)
            self.assertRegex(
                fields['archived'],
                r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$',
            )

    def test_archive_folder_entity_errors_when_destination_exists(self):
        """--archive refuses to clobber an existing folder under _archive/."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES)
            make_folder_entity(tmpdir, 'folder-entity',
                               entity('001', 'Folder Entity', 'done', '0.80'))
            # Pre-create the destination folder.
            archive_dest = os.path.join(tmpdir, '_archive', 'folder-entity')
            os.makedirs(archive_dest)
            with open(os.path.join(archive_dest, 'index.md'), 'w') as f:
                f.write('---\nid: 001\n---\nold\n')

            result = run_status(tmpdir, '--archive', 'folder-entity',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('already archived', result.stderr)
            # Source still present (not clobbered).
            self.assertTrue(os.path.isfile(
                os.path.join(tmpdir, 'folder-entity', 'index.md')))

    def test_archived_flag_lists_folder_archive(self):
        """--archived picks up folder-archived entities via `_archive/{slug}/index.md`."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                README_WITH_STAGES,
                entities={'active.md': entity('001', 'Active', 'backlog', '0.50')},
            )
            # Pre-create an archived folder entity.
            archived_dir = os.path.join(tmpdir, '_archive', 'old-folder')
            os.makedirs(archived_dir)
            with open(os.path.join(archived_dir, 'index.md'), 'w') as f:
                f.write(entity('002', 'Old Folder', 'done', '0.60'))

            result = run_status(tmpdir, '--archived',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('old-folder', result.stdout)
            self.assertIn('active', result.stdout)


class TestStatusDocstringEntityFolder(unittest.TestCase):
    """Static check: the status script header must document entity-as-folder."""

    def test_docstring_mentions_folder_form(self):
        with open(TEMPLATE_PATH, 'r') as f:
            content = f.read()
        # The script header must describe the folder-form discovery rule.
        self.assertIn('index.md', content)
        self.assertIn('folder-per-entity', content)


if __name__ == '__main__':
    unittest.main()
