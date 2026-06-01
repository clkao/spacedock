# ABOUTME: Tests for --set status= validation against the workflow's stages.states[].name list.
# ABOUTME: Covers unknown-stage rejection, every-declared-stage acceptance, --force override, and per-invocation README re-read.

import os
import tempfile
import textwrap
import unittest

from test_status_script import (
    build_status_script,
    make_pipeline,
    run_status,
    README_WITH_STAGES,
)


def minimal_entity(id, title, status):
    return textwrap.dedent(f"""\
        ---
        id: {id}
        title: {title}
        status: {status}
        ---

        Description.
        """)


README_TWO_STAGES = textwrap.dedent("""\
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
        - name: done
          terminal: true
    ---

    # Test Pipeline
    """)


README_TWO_STAGES_PLUS_IDEATION = textwrap.dedent("""\
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
        - name: done
          terminal: true
    ---

    # Test Pipeline
    """)


class TestStatusValidation(unittest.TestCase):
    """Test --set status= validation against the workflow's declared stages."""

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

    def test_unknown_stage_rejected(self):
        """--set status=<unknown> exits non-zero, names the value, lists known stages, leaves file untouched."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': minimal_entity('001', 'Task A', 'backlog'),
            })
            entity_path = os.path.join(tmpdir, 'task-a.md')
            with open(entity_path, 'rb') as f:
                pre_bytes = f.read()

            result = run_status(tmpdir, '--set', 'task-a', 'status=review',
                                script_path=self.script_path)

            self.assertNotEqual(result.returncode, 0,
                                f'expected non-zero exit; stdout={result.stdout!r} stderr={result.stderr!r}')
            self.assertIn("'review' is not a defined stage", result.stderr)
            for stage in ('backlog', 'ideation', 'implementation', 'validation', 'done'):
                self.assertIn(stage, result.stderr)
            # Known stages must appear in declaration order.
            stderr = result.stderr
            order = ['backlog', 'ideation', 'implementation', 'validation', 'done']
            indices = [stderr.index(s) for s in order]
            self.assertEqual(indices, sorted(indices),
                             f'known stages must appear in declaration order; got stderr={stderr!r}')
            self.assertNotIn('->', result.stdout)

            with open(entity_path, 'rb') as f:
                post_bytes = f.read()
            self.assertEqual(pre_bytes, post_bytes,
                             'entity file must be byte-identical after rejected --set')

    def test_every_declared_stage_accepted(self):
        """Every value in stages.states[].name is accepted by the validator."""
        stages = ['backlog', 'ideation', 'implementation', 'validation', 'done']
        for stage in stages:
            with tempfile.TemporaryDirectory() as tmpdir:
                make_pipeline(tmpdir, README_WITH_STAGES, {
                    'task-a.md': minimal_entity('001', 'Task A', 'backlog'),
                })
                # Use --force so unrelated mod-block / merge-hook / pr guards
                # cannot mask the validator behavior under test. The validator
                # itself must not reject any defined stage; --force only
                # bypasses *other* guards (mod-block, merge-hook), and is the
                # same pattern test_status_set_missing_field uses to neutralize
                # unrelated guards.
                result = run_status(tmpdir, '--set', 'task-a',
                                    f'status={stage}', '--force',
                                    script_path=self.script_path)
                self.assertEqual(result.returncode, 0,
                                 f'stage {stage!r} rejected; stderr={result.stderr!r}')
                fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
                self.assertEqual(fields['status'], stage,
                                 f'frontmatter not updated to {stage!r}; got {fields!r}')

    def test_force_overrides_validation(self):
        """--set status=<unknown> --force succeeds with a warning; frontmatter is updated."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task-a.md': minimal_entity('001', 'Task A', 'backlog'),
            })
            result = run_status(tmpdir, '--set', 'task-a',
                                'status=review', '--force',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0,
                             f'expected success with --force; stderr={result.stderr!r}')
            self.assertIn('Warning', result.stderr)
            self.assertIn("'review'", result.stderr)
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'review')

    def test_readme_reread_per_invocation(self):
        """README is re-read on every --set; renaming a stage between two calls flips the verdict."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_TWO_STAGES, {
                'task-a.md': minimal_entity('001', 'Task A', 'backlog'),
            })
            # Call 1: ideation is not declared yet.
            result1 = run_status(tmpdir, '--set', 'task-a', 'status=ideation',
                                 script_path=self.script_path)
            self.assertNotEqual(result1.returncode, 0,
                                f'expected rejection on call 1; stderr={result1.stderr!r}')

            # Mutate README to add ideation.
            with open(os.path.join(tmpdir, 'README.md'), 'w') as f:
                f.write(README_TWO_STAGES_PLUS_IDEATION)

            # Call 2: now ideation is declared and the same input is accepted.
            result2 = run_status(tmpdir, '--set', 'task-a', 'status=ideation',
                                 script_path=self.script_path)
            self.assertEqual(result2.returncode, 0,
                             f'expected success on call 2; stderr={result2.stderr!r}')
            fields = self._read_frontmatter(os.path.join(tmpdir, 'task-a.md'))
            self.assertEqual(fields['status'], 'ideation')


if __name__ == '__main__':
    unittest.main()
