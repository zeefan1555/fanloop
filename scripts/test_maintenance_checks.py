"""Small public-script checks using real Git repositories and subprocesses."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
SELECTOR = ROOT / '.github/scripts/classify-changes.py'
spec = importlib.util.spec_from_file_location('configuration', ROOT / 'scripts/validate-configuration.py')
configuration = importlib.util.module_from_spec(spec)
spec.loader.exec_module(configuration)


def synthetic_gate_text():
    return '\n'.join(str(value) for name, value in vars(configuration).items()
                     if name.startswith(('REQUIRED_', 'NOTION_EXPERIENCE_')) and isinstance(value, str))


class MaintenanceChecks(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.git('init', '-q')
        self.git('config', 'user.email', 'test@example.invalid')
        self.git('config', 'user.name', 'Test')
        self.git('config', 'core.hooksPath', '/dev/null')
        self.write('README.md')
        self.base = self.commit()

    def git(self, *args):
        return subprocess.check_output(['git', *args], cwd=self.root).decode().strip()

    def write(self, name, content='sample\n'):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
        return path

    def commit(self):
        self.git('add', '-A')
        self.git('commit', '-qm', 'fixture')
        return self.git('rev-parse', 'HEAD')

    def classify(self, expected, event='push', base=None):
        output = subprocess.check_output([sys.executable, str(SELECTOR), '--event', event,
                                          '--base', self.base if base is None else base,
                                          '--head', self.git('rev-parse', 'HEAD')], cwd=self.root)
        result = json.loads(output)
        self.assertEqual(expected, result['kind'], result)
        return result

    def test_configuration_and_unknown_paths(self):
        for name in ('AGENTS.md', 'docs/adr/example.md', 'docs/testing-guide.md',
                     '.agents/skills/edit-example/SKILL.md',
                     '.agents/skills/edit-example/references/example.md'):
            self.write(name)
        self.commit()
        self.classify('configuration')
        self.write('skills/example/skills/write/script.py')
        self.commit()
        self.classify('code')

    def test_mode_symlink_and_verification_assets(self):
        for name, mode in [('skills/x/SKILL.md', 0o755), ('docs/tests/input.md', 0o644),
                           ('skills/x/skills/verify-x/SKILL.md', 0o644),
                           ('.cursor/skills/verify-fanloop/features/example.md', 0o644),
                           ('.agents/skills/x/SKILL.md', 0o755),
                           ('.agents/skills/x/scripts/example.md', 0o644),
                           ('.agents/skills/verify-x/SKILL.md', 0o644),
                           ('.agents/skills/x/fixtures/example.md', 0o644),
                           ('.agents/skills/x/run.py', 0o644),
                           ('.agents/skills/x/workflow.yaml', 0o644),
                           ('.agents/skills/SKILL.md', 0o644),
                           ('.agents/other/x/SKILL.md', 0o644)]:
            with self.subTest(name=name):
                self.write(name).chmod(mode)
                self.commit()
                self.classify('code')
                self.base = self.git('rev-parse', 'HEAD')
        (self.root / 'skills/x/SKILL.md').chmod(0o644)
        self.commit()
        self.classify('code')
        self.base = self.git('rev-parse', 'HEAD')
        (self.root / 'docs/link.md').symlink_to('../README.md')
        self.commit()
        self.classify('code')
        self.base = self.git('rev-parse', 'HEAD')
        (self.root / '.agents/skills/x/link.md').symlink_to('../../../README.md')
        self.commit()
        self.classify('code')

    def test_repository_skill_renames_and_mixed_diff(self):
        self.write('.agents/skills/x/run.py')
        self.base = self.commit()
        self.git('mv', '.agents/skills/x/run.py', '.agents/skills/x/SKILL.md')
        self.commit()
        self.classify('code')
        self.base = self.git('rev-parse', 'HEAD')
        self.git('mv', '.agents/skills/x/SKILL.md', '.agents/skills/x/renamed.md')
        self.commit()
        self.classify('configuration')
        self.write('engine.go')
        self.commit()
        self.classify('code')

    def test_rename_both_paths_and_delete(self):
        self.write('script.py')
        self.base = self.commit()
        (self.root / 'docs').mkdir()
        self.git('mv', 'script.py', 'docs/ordinary.md')
        self.commit()
        self.classify('code')
        self.base = self.git('rev-parse', 'HEAD')
        self.git('mv', 'docs/ordinary.md', 'script.py')
        self.commit()
        self.classify('code')
        self.base = self.git('rev-parse', 'HEAD')
        self.git('rm', 'README.md')
        self.commit()
        self.classify('configuration')

    def test_configuration_rename_and_strange_names(self):
        self.write('docs/old.md')
        self.base = self.commit()
        self.git('mv', 'docs/old.md', 'docs/new\nline space.md')
        self.commit()
        self.classify('configuration')

    def test_missing_base_and_empty_diff_are_code(self):
        self.classify('code')
        self.classify('code', base='0' * 40)
        self.classify('code', base='f' * 40)

    def test_pr_merge_base_excludes_base_branch_change(self):
        self.git('checkout', '-qb', 'feature')
        self.write('docs/change.md')
        feature = self.commit()
        self.git('checkout', '--detach', self.base)
        self.write('engine.go')
        advanced_base = self.commit()
        self.git('checkout', '--detach', feature)
        self.classify('configuration', event='pull_request', base=advanced_base)
        self.classify('code', base=advanced_base)

    def test_local_only_materials_fail_even_without_base(self):
        self.write('docs/specs/forbidden.md')
        self.commit()
        result = subprocess.run([sys.executable, str(SELECTOR), '--event', 'push',
                                 '--base', '0' * 40, '--head', self.git('rev-parse', 'HEAD')],
                                cwd=self.root, capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('local-only', result.stderr)

    def test_matrix_selection_and_snapshot(self):
        for name in ('alpha', 'beta'):
            self.write(f'fanloop-workflow/{name}/SKILL.md')
        self.assertEqual(['alpha', 'beta'], configuration.select_workflows(self.root, None))
        self.assertEqual(['beta'], configuration.select_workflows(self.root, ['beta', 'beta']))
        with self.assertRaisesRegex(ValueError, 'unknown Workflow'):
            configuration.select_workflows(self.root, ['alpha', 'missing'])
        before = configuration.snapshot(self.root / 'fanloop-workflow')
        self.write('fanloop-workflow/alpha/SKILL.md', 'changed\n')
        self.assertNotEqual(before, configuration.snapshot(self.root / 'fanloop-workflow'))
        (self.root / 'fanloop-workflow/escape').symlink_to(self.root / 'fanloop-workflow/alpha')
        with self.assertRaises(ValueError):
            configuration.select_workflows(self.root, None)
        with self.assertRaises(ValueError):
            configuration.snapshot(self.root / 'fanloop-workflow')

    def test_unknown_workflow_has_no_matrix_side_effect(self):
        self.write('fanloop-workflow/alpha/SKILL.md')
        output = self.root / 'output'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--config-source', str(self.root), '--binary', sys.executable,
                                 '--output-root', str(output), '--workflow', 'missing'],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('unknown Workflow: missing', result.stderr)
        self.assertFalse(output.exists())

    def test_fomo_file_save_evidence_is_rejected(self):
        evidence = ROOT / 'scripts/testdata/panorama-file-save-evidence.json'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--reject-file-save-evidence', str(evidence)],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('content saved artifacts/panorama-', result.stderr)
        self.assertIn(configuration.REQUIRED_BLOCKED_SENTENCE, result.stderr)
        self.assertEqual('', result.stdout)

    def test_file_save_skill_instruction_names_the_fix(self):
        source = (ROOT / 'scripts/testdata/panorama-file-save-source').resolve()
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--check-panorama-skills', '--config-source', str(source)],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('fanloop-workflow/fomo/skills/workflow-panorama/SKILL.md', result.stderr)
        self.assertIn(configuration.REQUIRED_BLOCKED_SENTENCE, result.stderr)
        self.assertIn('content saved artifacts/panorama-', result.stderr)
        self.assertIn('writing a file', result.stderr)

    def test_parent_child_handoff_is_required_for_local_agent(self):
        source = (ROOT / 'scripts/testdata/panorama-child-reports-source').resolve()
        failed = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--check-panorama-skills', '--config-source', str(source)],
                                capture_output=True, text=True)
        self.assertNotEqual(0, failed.returncode)
        self.assertIn('fanloop-workflow/fomo/skills/workflow-panorama/SKILL.md', failed.stderr)
        self.assertIn('parent/child handoff', failed.stderr)
        for sentence in configuration.REQUIRED_HANDOFF_SENTENCES:
            self.assertIn(sentence, failed.stderr)
        self.assertNotIn('missing the blocked-when-headless rule', failed.stderr)
        self.assertNotIn('writing a file', failed.stderr)

        text = (source / 'fanloop-workflow/fomo/skills/workflow-panorama/SKILL.md').read_text()
        self.write(
            'with-handoff/fanloop-workflow/fomo/skills/workflow-panorama/SKILL.md',
            text + '\n' + '\n'.join(configuration.REQUIRED_HANDOFF_SENTENCES) + '\n',
        )
        self.write(
            'with-handoff/fanloop-workflow/fomo/condition.yaml',
            'schema_version: 3\nconditions:\n  panorama_presented:\n'
            '    prompt_ref: {file: prompt.yaml, prompt_id: panorama_condition}\n'
            '    output: {key: panorama_snapshot_path, type: path}\n',
        )
        self.write(
            'with-handoff/fanloop-workflow/fomo/prompt.yaml',
            'schema_version: 2\nprompts:\n  panorama_condition:\n'
            '    prompt: Deliver Panorama.\n    skills:\n'
            '      - {id: workflow-panorama, prompt: Deliver Panorama., optional: false}\n',
        )
        passed = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--check-panorama-skills', '--config-source', str(self.root / 'with-handoff')],
                                capture_output=True, text=True)
        self.assertEqual(0, passed.returncode, passed.stderr)
        self.assertIn('panorama skills: 1 ok', passed.stdout)

    def test_handoff_is_not_required_without_local_agent(self):
        text = (
            'report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.\n'
            'headless fanloop card render --view panorama --format markdown data.content\n'
        )
        self.assertEqual([], configuration.panorama_skill_errors(text))
        errors = configuration.panorama_skill_errors(text + 'local_agent\n')
        self.assertTrue(any('parent/child handoff' in error for error in errors))

    def test_unbound_panorama_named_directory_is_not_linted(self):
        self.write(
            'unbound/fanloop-workflow/fomo/condition.yaml',
            'schema_version: 3\nconditions:\n  work_ready:\n'
            '    prompt_ref: {file: prompt.yaml, prompt_id: work_condition}\n'
            '    output: {key: work_path, type: path}\n',
        )
        self.write(
            'unbound/fanloop-workflow/fomo/prompt.yaml',
            'schema_version: 2\nprompts:\n  work_condition:\n'
            '    prompt: Work is ready.\n    skills: []\n',
        )
        self.write(
            'unbound/fanloop-workflow/fomo/skills/unused-panorama/SKILL.md',
            'This intentionally invalid Panorama text is not bound by YAML.\n',
        )
        result = subprocess.run(
            [sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
             '--check-panorama-skills', '--config-source', str(self.root / 'unbound')],
            capture_output=True, text=True,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn('panorama skills: 0 ok', result.stdout)

    def test_bound_panorama_skill_is_linted_without_panorama_in_directory_name(self):
        self.write(
            'bound/fanloop-workflow/fomo/condition.yaml',
            'schema_version: 3\nconditions:\n  panorama_presented:\n'
            '    prompt_ref: {file: prompt.yaml, prompt_id: panorama_condition}\n'
            '    output: {key: panorama_snapshot_path, type: path}\n',
        )
        self.write(
            'bound/fanloop-workflow/fomo/prompt.yaml',
            'schema_version: 2\nprompts:\n  panorama_condition:\n'
            '    prompt: Deliver Panorama.\n    skills:\n'
            '      - {id: delivery, prompt: Deliver Panorama., optional: false}\n',
        )
        self.write(
            'bound/fanloop-workflow/fomo/skills/delivery/SKILL.md',
            'This intentionally lacks the required delivery contract.\n',
        )
        result = subprocess.run(
            [sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
             '--check-panorama-skills', '--config-source', str(self.root / 'bound')],
            capture_output=True, text=True,
        )
        self.assertNotEqual(0, result.returncode)
        self.assertIn('fanloop-workflow/fomo/skills/delivery/SKILL.md', result.stderr)

    def test_flow_style_panorama_skill_binding_is_linted(self):
        self.write(
            'flow-binding/fanloop-workflow/fomo/condition.yaml',
            'schema_version: 3\nconditions:\n  panorama_presented:\n'
            '    prompt_ref: {file: prompt.yaml, prompt_id: panorama_condition}\n'
            '    output: {key: panorama_snapshot_path, type: path}\n',
        )
        self.write(
            'flow-binding/fanloop-workflow/fomo/prompt.yaml',
            'schema_version: 2\nprompts:\n  panorama_condition:\n'
            '    prompt: Deliver Panorama.\n'
            '    skills: [{id: delivery, prompt: Deliver Panorama., optional: false}]\n',
        )
        self.write(
            'flow-binding/fanloop-workflow/fomo/skills/delivery/SKILL.md',
            'This intentionally lacks the required delivery contract.\n',
        )
        result = subprocess.run(
            [sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
             '--check-panorama-skills', '--config-source', str(self.root / 'flow-binding')],
            capture_output=True, text=True,
        )
        self.assertNotEqual(0, result.returncode)
        self.assertIn('fanloop-workflow/fomo/skills/delivery/SKILL.md', result.stderr)

    def test_path_only_receipt_is_not_a_file_save_claim(self):
        path = self.root / 'prepare-result.json'
        path.write_text(
            '{"condition_results":[{"condition_id":"panorama_presented",'
            '"output":{"type":"path","value":".fanloop/card/20261007T085445.957396+0800.json"}}]}\n'
        )
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--reject-file-save-evidence', str(path)],
                                capture_output=True, text=True)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn('no file-save delivery claim', result.stdout)

    def test_prohibition_line_is_not_a_delivery_claim(self):
        text = (
            '不得把 content saved artifacts/panorama-step.md 当作投递。\n'
            + configuration.REQUIRED_BLOCKED_SENTENCE + '\n'
        )
        self.assertEqual([], configuration.file_save_delivery_errors(text))

    def test_retrospect_transcript_without_done_panorama_fails(self):
        evidence = ROOT / 'scripts/testdata/panorama-retrospect-without-done.md'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--reject-missing-done-panorama', str(evidence)],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('missing Done 100% panorama', result.stderr)
        self.assertIn(configuration.REQUIRED_DONE_PANORAMA_SENTENCE, result.stderr)
        self.assertIn('`Done` `100%`', result.stderr)
        delivered = evidence.read_text() + (
            '\n# 制卡 `Done` `100%`\n整体进度：100%\n'
        )
        self.assertEqual([], configuration.missing_done_panorama_errors(delivered))
        in_progress = evidence.read_text().replace('workflow status completed', 'still on retrospect_run')
        self.assertEqual([], configuration.missing_done_panorama_errors(in_progress))

    def test_done_panorama_is_workflow_owned_not_globally_required(self):
        text = (
            'local_agent\n'
            + configuration.REQUIRED_BLOCKED_SENTENCE + '\n'
            + 'headless fanloop card render --view panorama --format markdown data.content\n'
            + '\n'.join(configuration.REQUIRED_HANDOFF_SENTENCES) + '\n'
        )
        self.assertEqual([], configuration.panorama_skill_errors(text))
        self.assertEqual([], configuration.panorama_skill_errors(
            text + configuration.REQUIRED_DONE_PANORAMA_SENTENCE + '\n'))

    def test_bundled_manage_workflow_panorama_skill_passes_delivery_lint(self):
        skill = ROOT / 'examples/workflows/manage-fanloop-workflow/skills/manage-fanloop-panorama/SKILL.md'
        self.assertEqual([], configuration.panorama_skill_errors(skill.read_text()))

    def test_combo_slang_fixture_fails_and_plain_words_pass(self):
        evidence = ROOT / 'scripts/testdata/combo-slang-one-pass.md'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--reject-combo-slang', str(evidence)],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('plain_words', result.stderr)
        self.assertIn('one_pass', result.stderr)
        self.assertIn('一次花光', result.stderr)
        self.assertIn('梭哈', result.stderr)
        plain = (
            'question: 这笔钱要不要一次花光？\n'
            'S: 你把零花钱分开买两样东西。\n'
            'kid_clarity:\n'
            '  - {card_key: batch-add, one_pass: yes, plain_words: yes}\n'
        )
        self.assertEqual([], configuration.combo_slang_errors(plain))

    def test_combo_markdown_id_ref_fixture_names_the_writer(self):
        evidence = ROOT / 'scripts/testdata/combo-create-content-file-id-ref.md'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--reject-combo-markdown-ref', str(evidence)],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('[[DCF]]0', result.stderr)
        self.assertIn('gTpsZTEYvBxB4S1PM', result.stderr)
        self.assertIn('object-set-text', result.stderr)
        self.assertIn('{"remId":"..."}', result.stderr)
        banned = (
            '不得用 create --content-file 写 [[id:gTpsZTEYvBxB4S1PM]]。失败例被写成 [[DCF]]0。\n'
            + configuration.REQUIRED_COMBO_REF_SENTENCE + '\n'
        )
        self.assertEqual([], configuration.combo_markdown_ref_errors(banned))
        self.assertEqual([], configuration.combo_id_teaching_errors(banned))

    def test_cli_help_renders_literal_percent(self):
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'), '--help'],
                                capture_output=True, text=True)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn('75%', result.stdout)
        self.assertNotIn('75%%', result.stdout)

    def test_missing_live_run_sentence_names_the_prompt(self):
        source = self.root / 'live-run-gates'
        relative = 'fanloop-workflow/remnote-flashcards/prompt.yaml'
        path = source / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(synthetic_gate_text().replace(
            configuration.REQUIRED_FACTUAL_REVIEW_SENTENCE, ''))
        contract = 'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md'
        (source / contract).parent.mkdir(parents=True, exist_ok=True)
        (source / contract).write_text(synthetic_gate_text())
        errors = configuration.learning_card_gate_errors(source)
        self.assertTrue(
            any('prompt.yaml' in error and configuration.REQUIRED_FACTUAL_REVIEW_SENTENCE in error
                for error in errors),
            errors,
        )

    def test_missing_notion_sync_sentence_names_the_prompt(self):
        source = self.root / 'notion-source'
        for relative in configuration.NOTION_EXPERIENCE_SENTENCE_FILES:
            text = synthetic_gate_text()
            if relative.endswith('prompt.yaml'):
                text = text.replace(configuration.REQUIRED_NOTION_SYNC_SENTENCE, '')
            path = source / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text)
        (source / 'README.md').write_text(
            'ntn login\n' + configuration.NOTION_EXPERIENCE_TITLE + '\n'
        )
        errors = configuration.notion_experience_sync_errors(source)
        self.assertTrue(
            any(relative.endswith('prompt.yaml') and configuration.REQUIRED_NOTION_SYNC_SENTENCE in error
                for relative in configuration.NOTION_EXPERIENCE_SENTENCE_FILES
                for error in errors if relative.endswith('prompt.yaml') and 'prompt.yaml' in error),
            errors,
        )
        self.assertFalse(any('SKILL.md' in error or 'notion-experience.md' in error for error in errors), errors)

    def test_missing_notion_data_source_id_names_the_reference(self):
        source = self.root / 'notion-ids'
        for relative in configuration.NOTION_EXPERIENCE_SENTENCE_FILES:
            path = source / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(synthetic_gate_text().replace(
                configuration.NOTION_EXPERIENCE_DATA_SOURCE_ID, 'removed-data-source'))
        (source / 'README.md').write_text(
            'ntn login\n' + configuration.NOTION_EXPERIENCE_TITLE + '\n'
        )
        errors = configuration.notion_experience_sync_errors(source)
        self.assertTrue(
            any(configuration.NOTION_EXPERIENCE_DATA_SOURCE_ID in error for error in errors),
            errors,
        )


if __name__ == '__main__':
    unittest.main()
