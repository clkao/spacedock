# ABOUTME: Offline test for entity worktree:-field stickiness across non-terminal advancements.
# ABOUTME: AC-2 — frontmatter `worktree:` persists once stamped, including across `fresh: true` stages.

from __future__ import annotations

import os
import tempfile
import textwrap
import unittest

from test_status_script import (
    build_status_script,
    make_pipeline,
    run_status,
)


def _entity(id_, title, status, *, worktree=""):
    """Generate entity frontmatter with optional worktree field."""
    if worktree:
        worktree_line = f"worktree: {worktree}\n"
    else:
        worktree_line = "worktree:\n"
    return textwrap.dedent(f"""\
        ---
        id: {id_}
        title: {title}
        status: {status}
        {worktree_line.rstrip()}
        ---

        Description.
        """)


def _read_frontmatter(filepath):
    fields = {}
    in_fm = False
    with open(filepath, "r") as f:
        for line in f:
            line = line.rstrip("\n")
            if line == "---":
                if in_fm:
                    break
                in_fm = True
                continue
            if in_fm and ":" in line:
                key, _, val = line.partition(":")
                fields[key.strip()] = val.strip()
    return fields


_README_INHERITED_DEFAULT_DOWNSTREAM = textwrap.dedent("""\
    ---
    entity-type: task
    entity-label: task
    stages:
      defaults:
        worktree: false
        concurrency: 2
      states:
        - name: initial
          initial: true
        - name: middle
          worktree: true
        - name: final
          terminal: true
    ---

    # Inherited-default-downstream Pipeline
    """)


_README_DEVELOPMENT_TEMPLATE = textwrap.dedent("""\
    ---
    entity-type: task
    entity-label: task
    stages:
      defaults:
        worktree: false
        concurrency: 2
      states:
        - name: initial
          initial: true
        - name: implementation
          worktree: true
        - name: validation
          worktree: true
          fresh: true
        - name: done
          terminal: true
    ---

    # Development-template Pipeline
    """)


class TestWorktreeStickiness(unittest.TestCase):
    """AC-2: `worktree:` frontmatter persists across non-terminal advancements once stamped."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_inherited_default_downstream_preserves_worktree(self):
        """`initial -> middle (worktree: true) -> final` keeps the worktree across the trailing default-false stage advance."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, _README_INHERITED_DEFAULT_DOWNSTREAM, {
                "task-a.md": _entity("001", "Task A", "initial"),
            })
            entity_path = os.path.join(tmpdir, "task-a.md")

            # Dispatch to middle: stamp worktree.
            r = run_status(
                tmpdir, "--set", "task-a",
                "status=middle", "worktree=.worktrees/test-task-a",
                script_path=self.script_path,
            )
            self.assertEqual(r.returncode, 0, r.stderr)
            fm = _read_frontmatter(entity_path)
            self.assertEqual(fm["status"], "middle")
            self.assertEqual(fm["worktree"], ".worktrees/test-task-a")

            # Advance to final WITHOUT passing worktree=. The field must persist
            # through the call (this is the central stickiness invariant — `--set`
            # already preserves untouched fields, and the FO contract no longer
            # rewrites `worktree=` to empty on default-false stage advance).
            # Note: we deliberately do not call --archive (which clears worktree
            # via the terminal flow) to isolate the persistence property.
            r = run_status(
                tmpdir, "--set", "task-a", "status=final",
                script_path=self.script_path,
            )
            self.assertEqual(r.returncode, 0, r.stderr)
            fm = _read_frontmatter(entity_path)
            self.assertEqual(fm["status"], "final",
                             "status must advance to terminal stage")
            self.assertEqual(
                fm["worktree"], ".worktrees/test-task-a",
                "worktree must persist through advancement when not explicitly cleared",
            )

    def test_development_template_fresh_true_preserves_worktree(self):
        """`implementation (worktree: true) -> validation (worktree: true, fresh: true)` keeps the same worktree."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, _README_DEVELOPMENT_TEMPLATE, {
                "task-b.md": _entity("002", "Task B", "initial"),
            })
            entity_path = os.path.join(tmpdir, "task-b.md")

            # Dispatch to implementation: stamp worktree.
            r = run_status(
                tmpdir, "--set", "task-b",
                "status=implementation", "worktree=.worktrees/test-task-b",
                script_path=self.script_path,
            )
            self.assertEqual(r.returncode, 0, r.stderr)
            fm_before = _read_frontmatter(entity_path)
            self.assertEqual(fm_before["worktree"], ".worktrees/test-task-b")

            # Advance to validation (fresh: true) without passing worktree=.
            # `fresh: true` is agent-lifecycle, NOT worktree-lifecycle: under
            # stickiness the existing worktree must NOT be cleared and a new
            # worktree must NOT be stamped. The field's value is invariant.
            r = run_status(
                tmpdir, "--set", "task-b", "status=validation",
                script_path=self.script_path,
            )
            self.assertEqual(r.returncode, 0, r.stderr)
            fm_after = _read_frontmatter(entity_path)
            self.assertEqual(fm_after["status"], "validation")
            self.assertEqual(
                fm_after["worktree"], fm_before["worktree"],
                "fresh: true must NOT cause a new worktree to be stamped or "
                "the existing one to be cleared",
            )
            self.assertEqual(fm_after["worktree"], ".worktrees/test-task-b")

    def test_terminal_archive_clears_worktree(self):
        """Archival (terminal flow) clears the `worktree:` field."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, _README_DEVELOPMENT_TEMPLATE, {
                "task-c.md": _entity(
                    "003", "Task C", "validation",
                    worktree=".worktrees/test-task-c",
                ),
            })
            entity_path = os.path.join(tmpdir, "task-c.md")
            fm = _read_frontmatter(entity_path)
            self.assertEqual(fm["worktree"], ".worktrees/test-task-c")

            # Terminal-and-clear in one --set with worktree= empty.
            r = run_status(
                tmpdir, "--set", "task-c",
                "status=done", "worktree=",
                script_path=self.script_path,
            )
            self.assertEqual(r.returncode, 0, r.stderr)
            fm = _read_frontmatter(entity_path)
            self.assertEqual(fm["status"], "done")
            self.assertEqual(
                fm["worktree"], "",
                "terminal flow must clear the worktree:` field",
            )


if __name__ == "__main__":
    unittest.main()
