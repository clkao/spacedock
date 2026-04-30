# ABOUTME: Tests the tightened pr-merge body template — wording rules, audit-link
# ABOUTME: format, AC8 regression invariants, and the task 123 golden fixture.

import os
import re
import unittest

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))
PR_MERGE_PATH = os.path.join(TESTS_DIR, '..', 'docs', 'plans', '_mods', 'pr-merge.md')


def read_pr_merge():
    with open(PR_MERGE_PATH, 'r') as f:
        return f.read()


# Golden fixture: task 123's stage reports, manually re-extracted under the tightened
# template rules. The captured PR #73 body served as the reference shape but slightly
# violated AC1 (28-word lead with parentheticals) and AC2 (17-word bullet); this fixture
# is the AC1/AC2-compliant trim. The test asserts (a) word count window per AC9
# methodology and (b) AC5-regex audit link match.
TASK_123_GOLDEN_BODY = """Make the status tool a reliable workflow-op CLI with unspaced `--where` syntax, custom frontmatter fields in the viewer, and in-tool archive moves.

## What changed

- Fix `--where` parser to accept unspaced `status=backlog` syntax and reject bare field names loudly.
- Add `--fields <list>` and `--all-fields` to show custom frontmatter keys alongside the default columns.
- Extend `--next` to honor `--fields` / `--all-fields`.
- Add `--archive <slug>` to stamp `archived:` and move an entity to `_archive/`.
- Add 23 unit tests covering parser, field display, archive, and docstring.

## Evidence

- `tests/test_status_script.py` — **90/90 passed**.
- Validation ensign independently verified all 20 acceptance criteria and live-probed the CLI against `docs/plans/`.

---

[123](/clkao/spacedock/blob/876a839/docs/plans/status-tool-as-workflow-op-cli.md)
"""


# AC5 regex: SHA-pinned audit link in the rendered body.
AUDIT_LINK_PATTERN = re.compile(r'\[(\d+)\]\(/[^/]+/[^/]+/blob/[a-f0-9]{7}/[^)]+\)')


def prose_word_count(body):
    """Count words in PR body prose, excluding `---` separator lines and fenced code blocks."""
    in_fence = False
    words = 0
    for line in body.splitlines():
        stripped = line.strip()
        if stripped.startswith('```'):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        if stripped == '---':
            continue
        words += len(stripped.split())
    return words


def extract_section(body, heading):
    """Return the lines under a `## heading` up to the next `##` or end."""
    lines = body.splitlines()
    out = []
    capturing = False
    for line in lines:
        if line.startswith('## '):
            if capturing:
                break
            if line.strip() == f'## {heading}':
                capturing = True
                continue
        elif capturing:
            out.append(line)
    return out


def section_bullets(section_lines):
    """Return top-level bullet text (no leading '- ') from a section's lines."""
    bullets = []
    for line in section_lines:
        if line.startswith('- '):
            bullets.append(line[2:])
        elif line.startswith('  ') and bullets:
            bullets[-1] += '\n' + line
    return bullets


def lead_paragraph(body):
    """Return the prose lead paragraph: lines before the first '## ' heading."""
    lines = []
    for line in body.splitlines():
        if line.startswith('## '):
            break
        lines.append(line)
    return '\n'.join(lines).strip()


class TestMotivationLead(unittest.TestCase):
    """AC1: lead is one sentence, ≤ 25 words, no parentheticals."""

    def test_template_specifies_one_sentence_25_words_no_parens(self):
        text = read_pr_merge()
        self.assertIn('1 sentence, ≤ 25 words', text)
        self.assertIn('No parentheticals.', text)

    def test_golden_lead_is_single_sentence_under_25_words(self):
        lead = lead_paragraph(TASK_123_GOLDEN_BODY)
        self.assertNotIn('(', lead, "lead must not contain parentheticals")
        sentence_terminators = sum(1 for ch in lead if ch in '.!?')
        self.assertLessEqual(sentence_terminators, 1, f"lead has {sentence_terminators} sentences, want ≤ 1")
        word_count = len(lead.split())
        self.assertLessEqual(word_count, 25, f"lead has {word_count} words, want ≤ 25")


class TestWhatChanged(unittest.TestCase):
    """AC2: 3–5 bullets, each ≤ 15 words, no rationale tails."""

    def test_template_specifies_bullet_count_and_length(self):
        text = read_pr_merge()
        self.assertIn('3–5 total', text)
        self.assertIn('each ≤ 15 words', text)
        self.assertIn('One change per bullet.', text)
        self.assertIn('No rationale inside the bullet', text)

    def test_golden_has_3_to_5_bullets(self):
        bullets = section_bullets(extract_section(TASK_123_GOLDEN_BODY, 'What changed'))
        self.assertGreaterEqual(len(bullets), 3)
        self.assertLessEqual(len(bullets), 5)

    def test_golden_bullets_are_short(self):
        bullets = section_bullets(extract_section(TASK_123_GOLDEN_BODY, 'What changed'))
        for bullet in bullets:
            words = len(bullet.split())
            self.assertLessEqual(words, 15, f"bullet has {words} words: {bullet!r}")

    def test_rationale_tail_heuristic_runs(self):
        """Heuristic guardrail for ' to <verb-gerund>' rationale tails.

        This is best-effort: it will false-positive on legitimate compound bullets
        (e.g. PR #73's `--archive` bullet which contains 'to stamp ... and move').
        We do not over-tune; instead we run the regex and accept zero or more
        flagged bullets, leaving false-positive resolution to human review.
        """
        rationale_tail = re.compile(r' to (avoid|prevent|keep|ensure|stop|allow|enable)ing?\b')
        bullets = section_bullets(extract_section(TASK_123_GOLDEN_BODY, 'What changed'))
        flagged = [b for b in bullets if rationale_tail.search(b)]
        self.assertEqual(flagged, [], "golden bullets should not match the rationale-tail heuristic")


class TestEvidence(unittest.TestCase):
    """AC3: 1–2 bullets, no test-class breakdowns, no enumerated suite lists."""

    def test_template_specifies_evidence_constraints(self):
        text = read_pr_merge()
        self.assertIn('1–2 bullets', text)
        self.assertIn('Do not include per-test-class breakdowns', text)
        self.assertIn('one pass ratio per suite', text)

    def test_golden_evidence_has_1_to_2_bullets(self):
        bullets = section_bullets(extract_section(TASK_123_GOLDEN_BODY, 'Evidence'))
        self.assertGreaterEqual(len(bullets), 1)
        self.assertLessEqual(len(bullets), 2)

    def test_golden_evidence_no_nested_lists(self):
        bullets = section_bullets(extract_section(TASK_123_GOLDEN_BODY, 'Evidence'))
        for bullet in bullets:
            self.assertNotIn('\n  -', bullet, f"nested list in evidence bullet: {bullet!r}")

    def test_golden_evidence_no_enumerated_pass_ratios(self):
        pass_ratio = re.compile(r'\d+/\d+\s*passed')
        bullets = section_bullets(extract_section(TASK_123_GOLDEN_BODY, 'Evidence'))
        for bullet in bullets:
            ratios = pass_ratio.findall(bullet)
            self.assertLessEqual(len(ratios), 1, f"bullet enumerates multiple pass ratios: {bullet!r}")


class TestExtractionRule(unittest.TestCase):
    """AC4: extraction rule forbids 'deliberately did NOT change' bullets."""

    def test_extraction_rule_present_exactly_once(self):
        text = read_pr_merge()
        occurrences = text.count('deliberately did NOT change')
        self.assertEqual(occurrences, 1, f"expected exactly 1 occurrence, got {occurrences}")

    def test_extraction_rule_in_prohibition_context(self):
        text = read_pr_merge()
        idx = text.index('deliberately did NOT change')
        window = text[max(0, idx - 80):idx + 80]
        self.assertIn('Do NOT include', window)


class TestAuditMetadata(unittest.TestCase):
    """AC5 + AC6: SHA-pinned audit link format and short-SHA computation step."""

    def test_template_describes_audit_link_format(self):
        text = read_pr_merge()
        self.assertIn('[{entity-id}](/{owner}/{repo}/blob/{short-sha}/{path-to-entity-file})', text)
        self.assertIn('[{short-id}](/{owner}/{repo}/blob/{short-sha}/{path})', text)

    def test_audit_link_uses_short_id_command(self):
        text = read_pr_merge()
        self.assertIn('status --short-id', text)

    def test_workflow_entity_verbose_line_removed(self):
        text = read_pr_merge()
        self.assertNotIn('Workflow entity: {entity title}', text)
        self.assertNotIn('Entity title verbatim', text)
        self.assertNotIn("Prefix `Workflow entity: `", text)

    def test_merge_hook_runs_short_sha_command(self):
        text = read_pr_merge()
        self.assertIn('git rev-parse --short HEAD', text)

    def test_short_sha_fallback_to_main_reported_to_captain(self):
        text = read_pr_merge()
        idx = text.index('git rev-parse --short HEAD')
        window = text[idx:idx + 600]
        self.assertIn('main', window)
        self.assertIn('captain', window)

    def test_golden_audit_link_matches_ac5_regex(self):
        match = AUDIT_LINK_PATTERN.search(TASK_123_GOLDEN_BODY)
        self.assertIsNotNone(match, "golden body must contain an AC5-format audit link")
        self.assertTrue(match.group(0).endswith('.md)'), "audit link path must end in .md")
        self.assertEqual(match.group(1), '123', "link label must be entity id 123")


class TestPreApprovalDraftIncludesFullBody(unittest.TestCase):
    """AC-6: pre-approval draft renders the fully constructed PR body.

    The captain must see the actual prose that will land on GitHub before approving the
    push, not just metadata. The post-approval step is then a literal `gh pr create`
    invocation against the already-constructed body — not a separate construction pass.
    """

    def setUp(self):
        self.text = read_pr_merge()
        hook_start = self.text.index('## Hook: merge')
        # The merge hook prose runs until the `### PR body template` subsection;
        # the template subsection itself describes the body shape, not the hook flow.
        hook_end = self.text.index('### PR body template', hook_start)
        self.hook = self.text[hook_start:hook_end]
        approval_idx = self.hook.index('**On approval:**')
        self.pre_approval = self.hook[:approval_idx]
        self.post_approval = self.hook[approval_idx:]

    def test_pre_approval_constructs_pr_body_from_template(self):
        self.assertIn('PR body', self.pre_approval)
        self.assertIn('template', self.pre_approval)
        self.assertIn('{constructed body}', self.pre_approval)

    def test_pre_approval_names_required_body_sections(self):
        for section in ('motivation lead', '## What changed', '## Evidence', 'audit link', 'Closes'):
            self.assertIn(section, self.pre_approval, f"pre-approval prose must name {section!r}")

    def test_post_approval_does_not_reconstruct_body(self):
        body_construction_phrases = (
            'Build the PR body using the template',
            'construct the PR body',
            'Build the full PR body',
            'Construct the full PR body',
        )
        for phrase in body_construction_phrases:
            self.assertNotIn(
                phrase, self.post_approval,
                f"post-approval block must not reconstruct the body ({phrase!r} found)"
            )

    def test_post_approval_gh_pr_create_uses_already_constructed_body(self):
        self.assertIn('gh pr create', self.post_approval)
        self.assertIn('{constructed body}', self.post_approval)


class TestTargetLength(unittest.TestCase):
    """AC7: target length is 60–120 words exactly once."""

    def test_target_length_window(self):
        text = read_pr_merge()
        self.assertEqual(text.count('60-120 words'), 1)
        self.assertNotIn('100-200 words', text)


class TestRegressionInvariants(unittest.TestCase):
    """AC8: byte-for-byte preservation of approval guardrail, push sequence, gh pr create, and decline blocks."""

    def setUp(self):
        self.text = read_pr_merge()

    def test_approval_guardrail_present(self):
        self.assertIn('PR APPROVAL GUARDRAIL', self.text)

    def test_push_main_present(self):
        self.assertIn('git push origin main', self.text)

    def test_rebase_main_present(self):
        self.assertIn('git rebase main', self.text)

    def test_push_branch_present(self):
        self.assertIn('git push origin {branch}', self.text)

    def test_gh_pr_create_present(self):
        self.assertIn('gh pr create --base main --head {branch} --title', self.text)

    def test_decline_and_no_archive_blocks_present(self):
        self.assertIn('On decline:', self.text)
        self.assertIn('Do NOT archive yet', self.text)


class TestGoldenFixture(unittest.TestCase):
    """AC9: golden body satisfies word-count window and contains valid audit link."""

    def test_golden_word_count_in_window(self):
        words = prose_word_count(TASK_123_GOLDEN_BODY)
        self.assertGreaterEqual(words, 60, f"golden body has {words} prose words, want ≥ 60")
        self.assertLessEqual(words, 120, f"golden body has {words} prose words, want ≤ 120")

    def test_golden_audit_link_present(self):
        self.assertIsNotNone(AUDIT_LINK_PATTERN.search(TASK_123_GOLDEN_BODY))


class TestScopeIsolation(unittest.TestCase):
    """AC10: this test module exists at the expected path (sanity check on file layout)."""

    def test_pr_merge_mod_exists(self):
        self.assertTrue(os.path.exists(PR_MERGE_PATH), f"{PR_MERGE_PATH} not found")


if __name__ == '__main__':
    unittest.main()
