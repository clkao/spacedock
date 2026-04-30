# ABOUTME: Regression tests for `status --discover` ignoring `.claude/worktrees/` workflow copies.
# ABOUTME: Verifies path-anchored prune of agent worktree duplicates while preserving sibling `worktrees/` dirs.

import os
import tempfile
import textwrap
import unittest

from test_status_script import build_status_script, make_workflow_readme, run_status


def _write_readme(path, commissioned_by='spacedock@1.0'):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w') as f:
        f.write(make_workflow_readme(commissioned_by=commissioned_by))


class TestDiscoverIgnoresClaudeWorktrees(unittest.TestCase):
    """`status --discover` must drop `.claude/worktrees/` copies of workflow READMEs."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def _build_fixture(self, tmpdir):
        # Primary workflow in the main checkout.
        primary = os.path.join(tmpdir, 'workflows', 'planning', 'README.md')
        _write_readme(primary)

        # Two duplicate copies under .claude/worktrees/<branch>/...
        for branch in ('ensign-foo', 'ensign-bar'):
            dup = os.path.join(
                tmpdir, '.claude', 'worktrees', branch,
                'workflows', 'planning', 'README.md',
            )
            _write_readme(dup)

        # Existing top-level .worktrees/ duplicate must remain excluded.
        legacy = os.path.join(
            tmpdir, '.worktrees', 'legacy-slug',
            'workflows', 'planning', 'README.md',
        )
        _write_readme(legacy)

        # User-committed sibling `worktrees/` (no leading dot, not under .claude/)
        # MUST NOT be excluded — the prune is path-anchored to `.claude/worktrees/`.
        sibling = os.path.join(tmpdir, 'worktrees', 'docs', 'README.md')
        _write_readme(sibling)

        return primary, sibling

    def test_discover_drops_claude_worktrees_duplicates(self):
        """Discovery returns the primary workflow and skips every `.claude/worktrees/` copy."""
        with tempfile.TemporaryDirectory() as tmpdir:
            primary, _sibling = self._build_fixture(tmpdir)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = [ln for ln in result.stdout.strip().split('\n') if ln]

            self.assertIn(os.path.realpath(os.path.dirname(primary)), lines)
            for line in lines:
                self.assertNotIn('/.claude/worktrees/', line,
                                 f'discovery returned a `.claude/worktrees/` path: {line}')

    def test_discover_preserves_existing_dot_worktrees_exclusion(self):
        """The existing top-level `.worktrees/` exclusion still suppresses duplicates."""
        with tempfile.TemporaryDirectory() as tmpdir:
            self._build_fixture(tmpdir)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = [ln for ln in result.stdout.strip().split('\n') if ln]

            for line in lines:
                self.assertFalse(
                    line.startswith(os.path.realpath(tmpdir) + os.sep + '.worktrees' + os.sep)
                    or os.sep + '.worktrees' + os.sep in line,
                    f'discovery returned a `.worktrees/` path: {line}',
                )

    def test_discover_keeps_sibling_worktrees_directory(self):
        """A user-committed `worktrees/` directory (no leading dot) is NOT pruned."""
        with tempfile.TemporaryDirectory() as tmpdir:
            _primary, sibling = self._build_fixture(tmpdir)

            result = run_status(tmpdir, '--discover', '--root', tmpdir,
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            lines = [ln for ln in result.stdout.strip().split('\n') if ln]

            self.assertIn(
                os.path.realpath(os.path.dirname(sibling)),
                lines,
                f'expected sibling `worktrees/docs` to be discovered, got: {lines}',
            )


if __name__ == '__main__':
    unittest.main()
