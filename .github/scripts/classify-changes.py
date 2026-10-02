#!/usr/bin/env python3
"""Select engine CI from the complete Git diff; uncertainty never skips tests."""
import argparse
import json
import os
from pathlib import PurePosixPath
import re
import subprocess
import sys

YAML = {'workflow.yaml', 'condition.yaml', 'flow.yaml', 'loop.yaml', 'prompt.yaml'}
ASSET_DIRS = {'test', 'tests', 'testdata', 'fixture', 'fixtures', 'verification',
              'validation', 'scripts', 'features'}


def git(*args):
    return subprocess.check_output(['git', *args], stderr=subprocess.PIPE)


def ordinary(path, mode):
    if mode != '100644':
        return False
    parts = PurePosixPath(path).parts
    if any(part in ASSET_DIRS or part.startswith(('test_', 'test-', 'verify-'))
           or part.endswith(('-verify', '-verification')) for part in parts[:-1]):
        return False
    if path in ('AGENTS.md', 'README.md'):
        return True
    if len(parts) >= 3 and parts[0] == 'skills':
        return path.endswith('.md') or (len(parts) == 3 and parts[2] in YAML)
    return len(parts) >= 2 and parts[0] == 'docs' and path.endswith('.md')


def classify(event, base, head):
    if not re.fullmatch(r'[0-9a-fA-F]{40}|[0-9a-fA-F]{64}', head):
        raise ValueError('head must be a full commit ID')
    git('rev-parse', '--verify', head + '^{commit}')
    local = git('ls-tree', '-r', '--name-only', '-z', head, '--',
                '.scratch', 'docs/research', 'docs/specs')
    if local:
        raise ValueError('local-only materials must not be tracked')
    result = {'head': head, 'base': base, 'kind': 'code', 'reason': 'missing comparison baseline'}
    if not re.fullmatch(r'[0-9a-fA-F]{40}|[0-9a-fA-F]{64}', base) or not base.strip('0'):
        return result
    exists = subprocess.run(['git', 'cat-file', '-e', base + '^{commit}'],
                            stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
    if exists.returncode:
        return result
    if event == 'pull_request':
        base = git('merge-base', base, head).decode().strip()
    raw = git('diff', '--raw', '-z', '--no-abbrev', '--find-renames', base, head, '--')
    entries = iter(raw.split(b'\0')[:-1])
    changes = []
    for header in entries:
        fields = header.decode('ascii').split()
        if len(fields) != 5 or not fields[0].startswith(':'):
            raise ValueError('invalid raw Git diff')
        old_mode, new_mode, _, _, status = fields
        old_mode = old_mode[1:]
        old_path = os.fsdecode(next(entries))
        new_path = os.fsdecode(next(entries)) if status.startswith(('R', 'C')) else old_path
        if status[0] not in 'AMDRCT':
            raise ValueError('unsupported Git change status: ' + status)
        safe = ((old_mode == '000000' or ordinary(old_path, old_mode))
                and (new_mode == '000000' or ordinary(new_path, new_mode)))
        changes.append({'old_path': old_path, 'new_path': new_path, 'status': status,
                        'old_mode': old_mode, 'new_mode': new_mode, 'configuration': safe})
    result.update(base=base, changes=changes,
                  kind='configuration' if changes and all(item['configuration'] for item in changes) else 'code',
                  reason='complete diff')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--event', required=True, choices=('pull_request', 'push'))
    parser.add_argument('--base', required=True)
    parser.add_argument('--head', required=True)
    args = parser.parse_args()
    result = classify(args.event, args.base, args.head)
    print(json.dumps(result, indent=2))
    if os.environ.get('GITHUB_OUTPUT'):
        with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
            output.write('code=' + str(result['kind'] == 'code').lower() + '\n')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError, subprocess.CalledProcessError, StopIteration) as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(1)
