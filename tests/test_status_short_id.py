# ABOUTME: Tests for `status --short-id REF` covering AC-1 (3 id-styles) and AC-2 (sd-b32 collision).
# ABOUTME: AC-1: sd-b32 returns shortest unique prefix; sequential/slug return stored ID/slug literal.

import os
import tempfile
import unittest

from test_status_script import (
    build_status_script,
    entity,
    make_pipeline,
    readme_with_id_style,
    run_status,
    sd_b32_id,
    shortest_unique_prefix,
)


class TestShortId(unittest.TestCase):
    """Tests for `status --short-id REF`."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_short_id_sd_b32_returns_shortest_unique_prefix(self):
        """AC-1 sd-b32: --short-id returns the same prefix as the ID column displays."""
        active_id = sd_b32_id('ab')
        other_id = sd_b32_id('cd')
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                readme_with_id_style('sd-b32'),
                entities={
                    'one.md': entity(active_id, 'One', 'backlog'),
                    'two.md': entity(other_id, 'Two', 'backlog'),
                },
            )
            result = run_status(tmpdir, '--short-id', active_id, script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            expected = shortest_unique_prefix(active_id, [active_id, other_id])
            self.assertEqual(result.stdout.strip(), expected)
            self.assertEqual(expected, 'ab')

    def test_short_id_sequential_returns_literal_id(self):
        """AC-1 sequential: --short-id returns the literal numeric stored ID."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                readme_with_id_style('sequential'),
                entities={
                    'one.md': entity('001', 'One', 'backlog'),
                    'two.md': entity('002', 'Two', 'backlog'),
                },
            )
            result = run_status(tmpdir, '--short-id', '001', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), '001')

    def test_short_id_slug_returns_literal_slug(self):
        """AC-1 slug: --short-id returns the literal slug."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                readme_with_id_style('slug'),
                entities={
                    'my-task.md': entity('my-task', 'My Task', 'backlog'),
                    'other-task.md': entity('other-task', 'Other Task', 'backlog'),
                },
            )
            result = run_status(tmpdir, '--short-id', 'my-task', script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout.strip(), 'my-task')

    def test_short_id_sd_b32_lengthens_prefix_when_active_archived_collide(self):
        """AC-2: sd-b32 active+archived collision forces prefix lengthening past MIN_PREFIX."""
        active_id = sd_b32_id('ab0')
        archived_id = sd_b32_id('ab1')
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                readme_with_id_style('sd-b32'),
                entities={'active.md': entity(active_id, 'Active', 'backlog')},
                archived={'archived.md': entity(archived_id, 'Archived', 'done')},
            )

            # Active entity must lengthen past 2 chars due to archived collision.
            active_result = run_status(
                tmpdir, '--short-id', active_id, script_path=self.script_path
            )
            self.assertEqual(active_result.returncode, 0, active_result.stderr)
            self.assertEqual(active_result.stdout.strip(), 'ab0')

            # Archived entity also needs the disambiguating prefix.
            archived_result = run_status(
                tmpdir, '--short-id', archived_id, script_path=self.script_path
            )
            self.assertEqual(archived_result.returncode, 0, archived_result.stderr)
            self.assertEqual(archived_result.stdout.strip(), 'ab1')


if __name__ == '__main__':
    unittest.main()
