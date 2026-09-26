#!/usr/bin/env python3
"""Run the public CLI route matrix against an explicit frozen configuration publication."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config-source', required=True, type=Path)
    parser.add_argument('--publication-manifest', required=True, type=Path)
    parser.add_argument('--binary', required=True, type=Path)
    parser.add_argument('--output-root', required=True, type=Path)
    args = parser.parse_args()
    repo = Path(__file__).resolve().parent.parent
    source = args.config_source.resolve(strict=True)
    binary = args.binary.resolve(strict=True)
    output = args.output_root.resolve()
    output.mkdir(parents=True, exist_ok=False)
    publication = json.loads(args.publication_manifest.read_text())
    files = {}
    for item in sorted((source / 'workflows').rglob('*')):
        if item.is_symlink():
            raise ValueError(f'symlink in frozen configuration: {item}')
        if item.is_file():
            files[str(item.relative_to(source / 'workflows'))] = hashlib.sha256(item.read_bytes()).hexdigest()
    expected = {item['path']: item['sha256'] for item in publication['files']}
    if files != expected:
        raise ValueError('configuration bytes differ from publication manifest')
    data = output / 'data'
    shutil.copytree(source / 'workflows', data / 'current/workflows')
    fake = output / 'bin'
    fake.mkdir()
    shutil.copy2(repo / 'tests/requirement_e2e/testdata/lark-cli', fake / 'lark-cli')
    env = {k: v for k, v in os.environ.items() if k not in ('BOTMUX_CHAT_ID', 'BOTMUX_SESSION_ID')}
    env.update(FANLOOP_DATA_HOME=str(data), FANLOOP_CONFIG_SOURCE=str(source), PATH=str(fake) + os.pathsep + env['PATH'])
    for kind in ('CODEX', 'AGENT', 'TRAE', 'CLAUDE'):
        skill_root = output / 'skill-roots' / kind.lower()
        skill_root.mkdir(parents=True)
        for skill in ('fanloop-workflow', 'create-fanloop-workflow'):
            (skill_root / skill).symlink_to(binary.parent.parent / 'entrypoints' / skill)
        env[f'FANLOOP_{kind}_SKILLS_ROOT'] = str(skill_root)
    outcomes = []
    for directory in sorted((source / 'workflows').iterdir()):
        if not directory.is_dir():
            raise ValueError('invalid Workflow root')
        workflow = directory.name
        env.update(FAKE_LARK_LOG=str(output / f'{workflow}-lark.log'), FAKE_TRACE_CONTENT=str(output / f'{workflow}-trace.md'), FAKE_TRACE_KEY='docx:FanloopConfigurationMatrix')
        argv = [sys.executable, str(repo / 'tests/requirement_e2e/route_matrix.py'), '--source-root', str(repo), '--binary', str(binary), '--workflow', workflow, '--output-root', str(output / workflow)]
        with (output / f'{workflow}.stdout').open('wb') as stdout, (output / f'{workflow}.stderr').open('wb') as stderr:
            result = subprocess.run(argv, env=env, stdout=stdout, stderr=stderr)
        outcomes.append({'workflow': workflow, 'argv': argv, 'exit_code': result.returncode})
        print(f'{workflow}: {result.returncode}', flush=True)
    passed = bool(outcomes) and all(item['exit_code'] == 0 for item in outcomes)
    (output / 'publication-validation.json').write_text(json.dumps({'publication': publication, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(), 'configuration_files': files, 'outcome': 'passed' if passed else 'failed', 'matrices': outcomes}, indent=2))
    return 0 if passed else 1


if __name__ == '__main__':
    raise SystemExit(main())
