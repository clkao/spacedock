# ABOUTME: Tests for strict opening-fence detection in parse_frontmatter, update_frontmatter, and discover_entity_files.
# ABOUTME: Guards issue #186 where a body horizontal rule was treated as YAML frontmatter and corrupted user prose.

import hashlib
import importlib.machinery
import importlib.util
import os
import tempfile
import textwrap
import unittest
from pathlib import Path

import pytest

from test_status_script import (
    build_status_script,
    make_pipeline,
    run_status,
    README_WITH_STAGES,
)

REPO_ROOT = Path(__file__).resolve().parent.parent
STATUS_SCRIPT = REPO_ROOT / "skills" / "commission" / "bin" / "status"


def _load_status_module():
    loader = importlib.machinery.SourceFileLoader("_status_lib_strict_fence", str(STATUS_SCRIPT))
    spec = importlib.util.spec_from_file_location(
        "_status_lib_strict_fence", str(STATUS_SCRIPT), loader=loader
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


status = _load_status_module()
parse_frontmatter = status.parse_frontmatter
update_frontmatter = status.update_frontmatter


class TestParseFrontmatterStrictFence:
    """AC-1: parse_frontmatter requires the first non-empty, non-BOM line to be `---`."""

    def test_prose_before_first_fence_returns_empty(self, tmp_path):
        """T-PARSE-1: prose first then `---` -> {}"""
        f = tmp_path / "notes.md"
        f.write_text(
            "# My research notes\n"
            "\n"
            "Some prose here.\n"
            "\n"
            "---\n"
            "\n"
            "More prose after a horizontal rule.\n"
        )
        assert parse_frontmatter(str(f)) == {}

    def test_leading_blank_lines_then_fence_parses(self, tmp_path):
        """T-PARSE-2: leading blank lines (truly empty) before fence are allowed."""
        f = tmp_path / "task.md"
        f.write_text("\n\n---\nid: 001\n---\n\nbody\n")
        assert parse_frontmatter(str(f)) == {"id": "001"}

    def test_leading_bom_then_fence_parses(self, tmp_path):
        """T-PARSE-3: BOM then `---` parses as if BOM were absent."""
        f = tmp_path / "task.md"
        f.write_text("﻿---\nid: 001\n---\n\nbody\n")
        assert parse_frontmatter(str(f)) == {"id": "001"}

    def test_whitespace_only_first_line_is_content(self, tmp_path):
        """T-PARSE-4: a whitespace-only line is content, not blank — fence cannot follow."""
        f = tmp_path / "task.md"
        f.write_text("   \n---\nid: 001\n---\n\nbody\n")
        assert parse_frontmatter(str(f)) == {}


class TestUpdateFrontmatterWriteGuard:
    """AC-2: update_frontmatter refuses to write when file lacks an opening fence."""

    def test_body_separator_only_raises_and_preserves_bytes(self, tmp_path):
        """T-WRITE-1: file whose first `---` is a body separator raises and is unchanged."""
        f = tmp_path / "notes.md"
        original = (
            "# My research notes\n"
            "\n"
            "Some prose here.\n"
            "\n"
            "---\n"
            "\n"
            "More prose after a horizontal rule.\n"
        )
        f.write_text(original)
        before = f.read_bytes()
        with pytest.raises(ValueError, match=r"No frontmatter found in"):
            update_frontmatter(str(f), [("id", "001")])
        after = f.read_bytes()
        assert before == after

    def test_pure_prose_no_fence_raises_and_preserves_bytes(self, tmp_path):
        """T-WRITE-2: file with no `---` at all raises and is unchanged."""
        f = tmp_path / "prose.md"
        original = "# Heading\n\nJust prose, no fences anywhere.\n"
        f.write_text(original)
        before = f.read_bytes()
        with pytest.raises(ValueError, match=r"No frontmatter found in"):
            update_frontmatter(str(f), [("id", "001")])
        after = f.read_bytes()
        assert before == after


# Reproduction prose used in T-DISCOVER-* and T-E2E-186.
NOTES_BODY_RULE = (
    "# My research notes\n"
    "\n"
    "Some prose here.\n"
    "\n"
    "---\n"
    "\n"
    "More prose after a horizontal rule.\n"
)


def valid_entity(slug, id_):
    return textwrap.dedent(f"""\
        ---
        id: {id_}
        title: {slug}
        status: backlog
        ---

        Description.
        """)


class TestDiscoveryStrictFence(unittest.TestCase):
    """AC-3 / AC-4: discover_entity_files skips files that lack an opening fence."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_active_scope_skips_body_rule_file(self):
        """T-DISCOVER-1: validate exits 0; default table lists valid only."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'valid.md': valid_entity('valid', '001'),
                'notes.md': NOTES_BODY_RULE,
            })

            validate = run_status(tmpdir, '--validate', script_path=self.script_path)
            self.assertEqual(validate.returncode, 0, validate.stderr)

            default = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(default.returncode, 0, default.stderr)
            self.assertIn('valid', default.stdout)
            self.assertNotIn('notes', default.stdout)

    def test_folder_form_index_without_fence_skipped(self):
        """T-DISCOVER-2: folder-form `slug/index.md` with body-rule first `---` is skipped."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'valid.md': valid_entity('valid', '001'),
            })
            slug_dir = os.path.join(tmpdir, 'orphan')
            os.makedirs(slug_dir)
            with open(os.path.join(slug_dir, 'index.md'), 'w') as f:
                f.write(NOTES_BODY_RULE)

            default = run_status(tmpdir, script_path=self.script_path)
            self.assertEqual(default.returncode, 0, default.stderr)
            self.assertIn('valid', default.stdout)
            self.assertNotIn('orphan', default.stdout)

    def test_archived_scope_skips_body_rule_file(self):
        """T-DISCOVER-3: --archived skips body-rule files but lists valid archived entities."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(
                tmpdir,
                README_WITH_STAGES,
                entities={},
                archived={
                    'orphan.md': NOTES_BODY_RULE,
                    'valid.md': valid_entity('valid', '001'),
                },
            )
            archived = run_status(tmpdir, '--archived', script_path=self.script_path)
            self.assertEqual(archived.returncode, 0, archived.stderr)
            self.assertIn('valid', archived.stdout)
            self.assertNotIn('orphan', archived.stdout)


class TestEndToEndIssue186(unittest.TestCase):
    """AC-5/AC-6: end-to-end --set against issue reproduction and a valid entity."""

    def setUp(self):
        self._script_dir = tempfile.mkdtemp()
        self.script_path = build_status_script(self._script_dir)

    def tearDown(self):
        os.unlink(self.script_path)
        os.rmdir(self._script_dir)

    def test_set_against_body_rule_file_fails_safely(self):
        """T-E2E-186: --set notes against body-rule notes.md fails non-zero, file unchanged."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'notes.md': NOTES_BODY_RULE,
            })
            notes_path = os.path.join(tmpdir, 'notes.md')
            sha_before = hashlib.sha256(open(notes_path, 'rb').read()).hexdigest()

            result = run_status(tmpdir, '--set', 'notes', 'id=001',
                                script_path=self.script_path)
            self.assertNotEqual(result.returncode, 0)
            # Slug pre-resolution rejects the unknown ref; the exact error path
            # is `resolve_mutation_entity` -> "Error: entity not found: <ref>".
            # The AC's intent is satisfied: non-zero exit + slug-not-resolved
            # diagnostic on stderr + zero file mutation (asserted via SHA256).
            self.assertIn('entity not found: notes', result.stderr)

            sha_after = hashlib.sha256(open(notes_path, 'rb').read()).hexdigest()
            self.assertEqual(sha_before, sha_after,
                             'notes.md must be byte-identical after rejected --set')

    def test_set_against_valid_entity_still_works(self):
        """T-E2E-VALID: --set against a properly-fenced entity rewrites status field."""
        with tempfile.TemporaryDirectory() as tmpdir:
            make_pipeline(tmpdir, README_WITH_STAGES, {
                'task.md': valid_entity('task', '001'),
            })
            result = run_status(tmpdir, '--set', 'task', 'status=ideation',
                                script_path=self.script_path)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('status:', result.stdout)
            self.assertIn('-> ideation', result.stdout)

            with open(os.path.join(tmpdir, 'task.md'), 'r') as f:
                content = f.read()
            self.assertIn('status: ideation', content)
            self.assertIn('Description.', content)


if __name__ == '__main__':
    import sys
    sys.exit(pytest.main([__file__, '-v']))
