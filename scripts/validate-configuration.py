#!/usr/bin/env python3
"""Validate a complete local Workflow source and run its public CLI Route Matrix.

Panorama Skills must block headless file-save delivery. --check-panorama-skills
checks that text without a matrix. Skills that mention local_agent must also
state the parent/child handoff: the child must not submit panorama_presented.
--reject-file-save-evidence fails when pasted evidence treats artifacts/panorama-*
as panorama_presented.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys

# Agents still reported panorama_presented after writing artifacts/panorama-*.md.
# Runtime only checks the path receipt (ADR-0119), so the Skill text and this lint
# are what block that claim.
REQUIRED_BLOCKED_SENTENCE = (
    'report blocked if no user-visible channel; never write artifacts/panorama-* as delivery.'
)
# A headless child rendered markdown and submitted panorama_presented while the
# parent that can post to the user chat never sent data.content (ADR-0119 still
# only checks the path receipt).
REQUIRED_HANDOFF_SENTENCES = (
    'The child must not submit panorama_presented.',
    'After non-dry-run fanloop card render --view panorama --format markdown, the child stops and returns data.content and data.snapshot_path to the parent; report blocked if render fails.',
    'The parent sends the exact data.content as the first user-visible message for that Step, then reports panorama_presented with that same snapshot_path, then continues the Step business Prompt.',
    'Never run a panorama-gated Workflow fully headless in a child that cannot post user-visible messages.',
)
DELIVERY_MARKERS = (
    'headless',
    'data.content',
    'fanloop card render',
    '--view panorama',
    '--format markdown',
)
FILE_SAVE_DELIVERY = re.compile(
    r'content saved artifacts/panorama-|saved to artifacts(?:/panorama-|\b)',
    re.IGNORECASE,
)
FILE_SAVE_INSTRUCTION = re.compile(
    r'(?i)(?:\b(?:save|saved|write|store|storing)\b|写入|保存).{0,80}artifacts/panorama'
    r'|artifacts/panorama.{0,80}(?:\b(?:save|saved|write|store)\b|写入|保存)'
)
PROHIBITION = ('never', 'do not', "don't", 'forbidden', 'not delivery', 'must not',
               '不得', '禁止', '不能', '不是', '不要')


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


def line_is_prohibition(line):
    lowered = line.lower()
    return any(token in lowered or token in line for token in PROHIBITION)


def file_save_delivery_errors(text):
    """Reject the FOMO evidence shape: a file save claimed as panorama delivery."""
    errors = []
    for line in text.splitlines():
        if line_is_prohibition(line):
            continue
        claim = FILE_SAVE_DELIVERY.search(line)
        if claim:
            errors.append(
                'panorama file-save is not delivery: '
                f'found {claim.group(0)!r}. {REQUIRED_BLOCKED_SENTENCE}'
            )
    return errors


def panorama_skill_errors(text):
    errors = []
    if REQUIRED_BLOCKED_SENTENCE not in text:
        errors.append(
            'local_agent panorama delivery is missing the blocked-when-headless rule. '
            f'Add this sentence: "{REQUIRED_BLOCKED_SENTENCE}"'
        )
    missing = [marker for marker in DELIVERY_MARKERS if marker not in text]
    if missing:
        errors.append(
            'panorama delivery is missing required markers: ' + ', '.join(missing) + '. '
            f'Add this sentence: "{REQUIRED_BLOCKED_SENTENCE}"'
        )
    for line in text.splitlines():
        if 'artifacts/panorama' not in line or line_is_prohibition(line):
            continue
        claim = FILE_SAVE_DELIVERY.search(line)
        if claim:
            errors.append(
                'panorama file-save is not delivery: '
                f'found {claim.group(0)!r}. {REQUIRED_BLOCKED_SENTENCE}'
            )
            continue
        if FILE_SAVE_INSTRUCTION.search(line):
            errors.append(
                'Skill instructs satisfying panorama by writing a file. '
                f'Remove that instruction. {REQUIRED_BLOCKED_SENTENCE}'
            )
    if 'local_agent' in text:
        missing = [sentence for sentence in REQUIRED_HANDOFF_SENTENCES if sentence not in text]
        if missing:
            quoted = ' '.join(f'"{sentence}"' for sentence in missing)
            errors.append(
                'local_agent panorama delivery is missing the parent/child handoff. '
                f'Add these sentences: {quoted}'
            )
    return errors


def _skill_dirs(root):
    if not root.is_dir() or root.is_symlink():
        return []
    children = []
    for item in sorted(root.iterdir()):
        if item.is_symlink():
            raise ValueError(f'unsupported package entry: {item}')
        if item.is_dir():
            children.append(item)
    return children


def panorama_skill_files(source, workflows, include_examples):
    found = []
    selected = None if workflows is None else set(workflows)
    skills_root = source / 'skills'
    if skills_root.is_dir() and not skills_root.is_symlink():
        for workflow_dir in _skill_dirs(skills_root):
            if selected is not None and workflow_dir.name not in selected:
                continue
            for skill_dir in _skill_dirs(workflow_dir / 'skills'):
                if 'panorama' not in skill_dir.name:
                    continue
                skill = skill_dir / 'SKILL.md'
                if skill.is_symlink():
                    raise ValueError(f'unsupported package entry: {skill}')
                if skill.is_file():
                    found.append(skill)
    examples = source / 'examples' / 'workflows'
    if include_examples and examples.is_dir() and not examples.is_symlink():
        for workflow_dir in _skill_dirs(examples):
            for skill_dir in _skill_dirs(workflow_dir / 'skills'):
                if 'panorama' not in skill_dir.name:
                    continue
                skill = skill_dir / 'SKILL.md'
                if skill.is_symlink():
                    raise ValueError(f'unsupported package entry: {skill}')
                if skill.is_file():
                    found.append(skill)
    return found


def _relative(path, source):
    try:
        return path.resolve().relative_to(source.resolve()).as_posix()
    except ValueError:
        return path.as_posix()


def check_panorama_skills(source, workflows, include_examples):
    errors = []
    for skill in panorama_skill_files(source, workflows, include_examples):
        text = skill.read_text()
        for error in panorama_skill_errors(text):
            errors.append(f'{_relative(skill, source)}: {error}')
    return errors


def reject_file_save_evidence(path):
    text = path.read_text()
    errors = file_save_delivery_errors(text)
    if not errors:
        print('panorama evidence has no file-save delivery claim')
        return 0
    for error in errors:
        print(f'{path}: {error}', file=sys.stderr)
    return 1


def select_workflows(source, requested):
    directories = sorted((source / 'skills').iterdir())
    if any(not item.is_dir() or item.is_symlink() for item in directories):
        raise ValueError('invalid Workflow package root')
    available = {item.name for item in directories}
    selected = sorted(set(requested)) if requested else sorted(available)
    unknown = set(selected) - available
    if unknown:
        raise ValueError('unknown Workflow: ' + ', '.join(sorted(unknown)))
    return selected


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config-source', type=Path)
    parser.add_argument('--binary', type=Path)
    parser.add_argument('--output-root', type=Path)
    parser.add_argument('--workflow', action='append', help='Workflow matrix to run; repeatable, default all')
    parser.add_argument('--check-panorama-skills', action='store_true',
                        help='Check panorama Skill delivery text, including the local_agent parent/child handoff, then exit')
    parser.add_argument('--reject-file-save-evidence', type=Path,
                        help='Exit 1 when text claims panorama delivery by saving artifacts/panorama-*')
    args = parser.parse_args()
    if args.reject_file_save_evidence is not None:
        return reject_file_save_evidence(args.reject_file_save_evidence)
    repo = Path(__file__).resolve().parent.parent
    if args.config_source is None or not args.config_source.is_absolute():
        raise ValueError('configuration source must be absolute')
    source = args.config_source
    workflows = select_workflows(source, args.workflow)
    errors = check_panorama_skills(source, workflows, include_examples=not args.workflow)
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    if args.check_panorama_skills:
        count = len(panorama_skill_files(source, workflows, include_examples=not args.workflow))
        print(f'panorama skills: {count} ok')
        return 0
    if args.binary is None or args.output_root is None:
        raise ValueError('matrix validation requires --binary and --output-root')
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
    for workflow in workflows:
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
