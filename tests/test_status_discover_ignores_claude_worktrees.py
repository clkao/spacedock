# ABOUTME: Regression tests for `status --discover` honoring .gitignore directory entries.
# ABOUTME: Verifies that gitignore-augmented DISCOVER_IGNORE_DIRS suppresses .claude/worktrees/ duplicates.

import os
import tempfile
import unittest

from test_status_script import build_status_script, make_workflow_readme, run_status


def _write_readme(path, commissioned_by='spacedock@1.0'):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w') as f:
        f.write(make_workflow_readme(commissioned_by=commissioned_by))


def _write_gitignore(tmpdir, entries):
    with open(os.path.join(tmpdir, '.gitignore'), 'w') as f:
        for entry in entries:
            f.write(entry + '\n')


class TestDiscoverHonorsGitignore(unittest.TestCase):
    """`status --discover` augments DISCOVER_IGNORE_DIRS with gitignore directory entries."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _build_fixture(self, tmpdir):
        _write_gitignore(tmpdir, ['.claude/worktrees/'])

        primary = os.path.join(tmpdir, 'workflows', 'planning', 'README.md')
        _write_readme(primary)

        for branch in ('ensign-foo', 'ensign-bar'):
            dup = os.path.join(
                tmpdir, '.claude', 'worktrees', branch,
                'workflows', 'planning', 'README.md',
            )
            _write_readme(dup)

        legacy = os.path.join(
            tmpdir, '.worktrees', 'legacy-slug',
            'workflows', 'planning', 'README.md',
        )
        _write_readme(legacy)

        return primary

    def test_discover_drops_claude_worktrees_duplicates(self):
        """A `.gitignore` entry of `.claude/worktrees/` suppresses every duplicate copy."""
        with tempfile.TemporaryDirectory() as tmpdir:
            primary = self._build_fixture(tmpdir)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = [ln for ln in result.stdout.strip().split('\n') if ln]

            self.assertIn(os.path.realpath(os.path.dirname(primary)), lines)
            for line in lines:
                self.assertNotIn('/.claude/worktrees/', line,
                                 f'discovery returned a `.claude/worktrees/` path: {line}')

    def test_discover_preserves_existing_dot_worktrees_exclusion(self):
        """The hardcoded `.worktrees/` baseline still suppresses duplicates."""
        with tempfile.TemporaryDirectory() as tmpdir:
            self._build_fixture(tmpdir)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = [ln for ln in result.stdout.strip().split('\n') if ln]

            for line in lines:
                self.assertNotIn(
                    os.sep + '.worktrees' + os.sep, line,
                    f'discovery returned a `.worktrees/` path: {line}',
                )

    def test_discover_without_gitignore_entry_returns_claude_worktrees(self):
        """Without the gitignore entry, duplicates leak through — confirms gitignore is the mechanism."""
        with tempfile.TemporaryDirectory() as tmpdir:
            _write_gitignore(tmpdir, [])

            primary = os.path.join(tmpdir, 'workflows', 'planning', 'README.md')
            _write_readme(primary)

            dup = os.path.join(
                tmpdir, '.claude', 'worktrees', 'ensign-foo',
                'workflows', 'planning', 'README.md',
            )
            _write_readme(dup)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = [ln for ln in result.stdout.strip().split('\n') if ln]

            self.assertTrue(
                any('/.claude/worktrees/' in ln for ln in lines),
                f'expected `.claude/worktrees/` duplicate to leak through without gitignore entry, got: {lines}',
            )


if __name__ == '__main__':
    unittest.main()
