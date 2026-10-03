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
        self.git('commit', '-qm', 'fixture\n\nCo-Authored-By: Aiden')
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
                     'skills/example/SKILL.md', 'skills/example/prompt.yaml',
                     'skills/example/skills/write/references/example.md',
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
            self.write(f'skills/{name}/SKILL.md')
        self.assertEqual(['alpha', 'beta'], configuration.select_workflows(self.root, None))
        self.assertEqual(['beta'], configuration.select_workflows(self.root, ['beta', 'beta']))
        with self.assertRaisesRegex(ValueError, 'unknown Workflow'):
            configuration.select_workflows(self.root, ['alpha', 'missing'])
        before = configuration.snapshot(self.root / 'skills')
        self.write('skills/alpha/SKILL.md', 'changed\n')
        self.assertNotEqual(before, configuration.snapshot(self.root / 'skills'))
        (self.root / 'skills/escape').symlink_to(self.root / 'skills/alpha')
        with self.assertRaises(ValueError):
            configuration.select_workflows(self.root, None)
        with self.assertRaises(ValueError):
            configuration.snapshot(self.root / 'skills')

    def test_unknown_workflow_has_no_matrix_side_effect(self):
        self.write('skills/alpha/SKILL.md')
        output = self.root / 'output'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts/validate-configuration.py'),
                                 '--config-source', str(self.root), '--binary', sys.executable,
                                 '--output-root', str(output), '--workflow', 'missing'],
                                capture_output=True, text=True)
        self.assertNotEqual(0, result.returncode)
        self.assertIn('unknown Workflow: missing', result.stderr)
        self.assertFalse(output.exists())


if __name__ == '__main__':
    unittest.main()
