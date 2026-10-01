#!/usr/bin/env python3
"""Validate a complete local Workflow source and run its public CLI Route Matrix."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys


def snapshot(root):
    files = []
    for item in sorted(root.rglob('*')):
        mode = item.lstat().st_mode
        if stat.S_ISDIR(mode):
            continue
        if not stat.S_ISREG(mode):
            raise ValueError(f'unsupported package entry: {item}')
        files.append({'path': str(item.relative_to(root)), 'sha256': hashlib.sha256(item.read_bytes()).hexdigest(), 'executable': bool(mode & 0o111)})
    if not files:
        raise ValueError('source contains no Workflow packages')
    return files


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config-source', required=True, type=Path)
    parser.add_argument('--binary', required=True, type=Path)
    parser.add_argument('--output-root', required=True, type=Path)
    args = parser.parse_args()
    repo = Path(__file__).resolve().parent.parent
    if not args.config_source.is_absolute():
        raise ValueError('configuration source must be absolute')
    source = args.config_source
    binary = args.binary.resolve(strict=True)
    output = args.output_root.resolve()
    output.mkdir(parents=True, exist_ok=False)
    files = snapshot(source / 'skills')
    data = output / 'data'
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
    for directory in sorted((source / 'skills').iterdir()):
        if not directory.is_dir() or directory.is_symlink():
            raise ValueError('invalid Workflow package root')
        workflow = directory.name
        env.update(FAKE_LARK_LOG=str(output / f'{workflow}-lark.log'), FAKE_TRACE_CONTENT=str(output / f'{workflow}-trace.md'), FAKE_TRACE_KEY='docx:FanloopConfigurationMatrix')
        argv = [sys.executable, str(repo / 'tests/requirement_e2e/route_matrix.py'), '--source-root', str(repo), '--binary', str(binary), '--workflow', workflow, '--output-root', str(output / workflow)]
        with (output / f'{workflow}.stdout').open('wb') as stdout, (output / f'{workflow}.stderr').open('wb') as stderr:
            result = subprocess.run(argv, env=env, stdout=stdout, stderr=stderr)
        outcomes.append({'workflow': workflow, 'argv': argv, 'exit_code': result.returncode})
        print(f'{workflow}: {result.returncode}', flush=True)
    unchanged = files == snapshot(source / 'skills')
    passed = unchanged and bool(outcomes) and all(item['exit_code'] == 0 for item in outcomes)
    manifest = {'source': str(source), 'files': files}
    (output / 'configuration-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    (output / 'configuration-validation.json').write_text(json.dumps({'configuration': manifest, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(), 'source_unchanged': unchanged, 'outcome': 'passed' if passed else 'failed', 'matrices': outcomes}, indent=2) + '\n')
    return 0 if passed else 1


if __name__ == '__main__':
    raise SystemExit(main())
