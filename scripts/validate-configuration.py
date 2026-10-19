#!/usr/bin/env python3
"""Validate a complete local Workflow source and run its public CLI Route Matrix.

Panorama Skills must block headless file-save delivery. --check-panorama-skills
checks that text without a matrix. Skills that mention local_agent must also
state the parent/child handoff and the post-complete Done panorama delivery.
--reject-file-save-evidence fails when pasted evidence treats artifacts/panorama-*
as panorama_presented. --reject-missing-done-panorama fails a transcript that
stops after the retrospect 75% panorama once the workflow has completed.
--reject-combo-slang and --reject-combo-markdown-ref fail the 2026-10-07
remnote-flashcards card mistakes. --check-learning-card-gates checks the
remnote-flashcards package for those two gates.
--check-notion-experience-sync fails when retrospect_run no longer tells the
agent to POST findings with ntn into 「remnote workflow 经验表」.
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
# 2026-10-07 remnote-flashcards: the host posted the retrospect-step panorama
# (`In Progress` `75%`) and stopped after the terminal route. The completed
# renderer content is `Done` `100%`. ADR-0104's "do not render again in the
# ordinary reply" does not cover that later delivery. Runtime still only checks
# the path receipt (ADR-0119), so the Skill sentence and the transcript lint do.
REQUIRED_DONE_PANORAMA_SENTENCE = (
    'after a successful terminal Route, render panorama once more and deliver exact data.content; '
    'the in-step retrospect panorama is not the Done 100% delivery.'
)
FORBIDDEN_SKIP_DONE_PANORAMA = '最终回复不得重复 Panorama'
# 2026-10-07 combo review: one_pass/plain_words failed on unexplained trading slang.
COMBO_SLANG = ('梭哈', '全仓', '一把梭', '优秀个股')
REQUIRED_COMBO_SLANG_SENTENCE = (
    'combo plain_words rejects unexplained trading slang: 梭哈, 全仓, 一把梭, 优秀个股; '
    'rewrite with plain kid words such as 一次花光 or 分开买.'
)
# 2026-10-07 STAR combo: create --content-file with [[id:gTpsZTEYvBxB4S1PM]]
# (分批加仓) mangled the T body to [[DCF]]0. The write types that keep the rem
# ref are TextSegments remId and RichText {i:q,_id}.
REQUIRED_COMBO_REF_SENTENCE = (
    'combo rem-refs use flashcards apply TextSegments {"remId":"..."} or '
    'object-set-text RichText {"i":"q","_id":"..."}; do not use create --content-file with [[id:]].'
)
# 2026-10-09 live runs. Runtime only checks condition types and routes, so these
# sentences keep the new receipts from drifting out of the prompts and contract.
REQUIRED_CONTROL_SEARCH_SENTENCE = (
    '已知术语对照搜索返回空时，先点桌面按钮「重新加载搜索缓存」，再重搜。'
)
REQUIRED_ENGLISH_SEARCH_SENTENCE = 'Also search the English form of the term when it has one.'
REQUIRED_FACTUAL_REVIEW_SENTENCE = (
    'example numbers state their condition, such as when idle; '
    'do not use a platform term as an everyday word, such as 小程序 on a WeChat card; '
    "a binary question's answer line states the choice."
)
REQUIRED_ALIAS_COMMAND = (
    "sdk rem object-get-or-create-alias-with-text --target-id <ID> --args-json '[[\"<alias>\"]]'"
)
REQUIRED_ALIAS_ABORT = 'SDK 0.0.46 cannot acknowledge abort'
REQUIRED_MOVE_NOTE = (
    'move-note <COMBO_ID> --new-parent-rem-id <THEME_ID> --position last '
    '--expected-old-parent-rem-id <DAILY_ID>'
)
REQUIRED_REMOVE_TAG = (
    "object-remove-tag --target-id <COMBO_ID> --args-json '[\"<TAG_ID>\", false]' --allow-destructive"
)
REQUIRED_TAG_CREATE = 'create --title "<Concept 名称>" --parent-id'
REQUIRED_THEME_PARA = (
    'theme_para maps 00-NOTE to note, 01-PROJECTS to project, 02-AREAS to area, '
    '03-RESOURCES to resource, and 04-ARCHIVE to archive.'
)
REQUIRED_PLUGIN_COMMIT = 'plugin_commit'
REQUIRED_REVIEWER_USER = 'reviewer=user'
LIVE_RUN_ROUTE_CONDITIONS = (
    'tag_approved',
    'control_search_passed',
    'combo_placed',
    'aliases_written',
)
MANGLED_COMBO_REF = '[[DCF]]0'
DONE_PANORAMA_HEADER = re.compile(r'`Done`\s*`100%`')
RETROSPECT_PROGRESS = re.compile(r'`75%`|整体进度：75%')
WORKFLOW_COMPLETED = re.compile(
    r'workflow status completed|status completed|"status"\s*:\s*"completed"|terminal route accepted',
    re.IGNORECASE,
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
        if REQUIRED_DONE_PANORAMA_SENTENCE not in text:
            errors.append(
                'local_agent panorama delivery is missing the post-complete Done panorama. '
                f'Add this sentence: "{REQUIRED_DONE_PANORAMA_SENTENCE}"'
            )
        if FORBIDDEN_SKIP_DONE_PANORAMA in text:
            errors.append(
                'Skill tells the host to stop after the in-step panorama. '
                f'Remove "{FORBIDDEN_SKIP_DONE_PANORAMA}". '
                f'Add this sentence: "{REQUIRED_DONE_PANORAMA_SENTENCE}"'
            )
    return errors


def missing_done_panorama_errors(text):
    """Fail the 2026-10-07 transcript: retrospect 75% panorama, then completed, no Done 100%."""
    retrospect_at = None
    for match in re.finditer(r'轻量复盘', text):
        window = text[max(0, match.start() - 500):min(len(text), match.end() + 500)]
        if RETROSPECT_PROGRESS.search(window):
            retrospect_at = match.start()
            break
    if retrospect_at is None:
        return []
    after = text[retrospect_at:]
    if not WORKFLOW_COMPLETED.search(after):
        return []
    if DONE_PANORAMA_HEADER.search(after):
        return []
    return [
        'missing Done 100% panorama after retrospect. '
        'The user-visible message must be a new render whose data.content includes `Done` `100%`. '
        + REQUIRED_DONE_PANORAMA_SENTENCE
    ]


def line_allows_slang(line):
    if line_is_prohibition(line) or REQUIRED_COMBO_SLANG_SENTENCE in line:
        return True
    lowered = line.lower()
    return '失败' in line or 'fail' in lowered


def combo_slang_errors(text):
    errors = []
    for line in text.splitlines():
        if line_allows_slang(line):
            continue
        found = [token for token in COMBO_SLANG if token in line]
        if not found:
            continue
        errors.append(
            'combo kid_clarity plain_words and one_pass are no for unexplained trading slang: '
            + ', '.join(found)
            + '. Rewrite the question and S/T/A/R. ' + REQUIRED_COMBO_SLANG_SENTENCE
        )
    return errors


def combo_markdown_ref_errors(text):
    kept = [line for line in text.splitlines() if not line_is_prohibition(line)]
    body = '\n'.join(kept)
    teaches = '--content-file' in body and '[[id:' in body
    mangled = MANGLED_COMBO_REF in body
    if not teaches and not mangled:
        return []
    return [
        'combo rem-ref was written with create --content-file and markdown [[id:]]. '
        '2026-10-07 [[id:gTpsZTEYvBxB4S1PM]] (分批加仓) was mangled to [[DCF]]0. '
        + REQUIRED_COMBO_REF_SENTENCE
    ]


COMBO_REF_TEACHING_FILES = (
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/remnote-import-format.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/SKILL.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-cli/SKILL.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-verify/SKILL.md',
    'fanloop-workflow/remnote-flashcards/prompt.yaml',
)


def combo_id_teaching_errors(text):
    """Reject markdown [[id:]] write instructions. Wiki [[id:]] lives in another file."""
    errors = []
    for line in text.splitlines():
        if '[[id:' not in line or REQUIRED_COMBO_REF_SENTENCE in line:
            continue
        if '不得' in line and '失败' in line:
            continue
        errors.append(
            'this line teaches markdown [[id:]] as a rem-ref write. '
            + REQUIRED_COMBO_REF_SENTENCE
        )
    return errors


def remnote_done_prompt_errors(source, workflows):
    if workflows is not None and 'remnote-flashcards' not in workflows:
        return []
    prompt = source / 'fanloop-workflow' / 'remnote-flashcards' / 'prompt.yaml'
    if not prompt.is_file():
        return []
    if REQUIRED_DONE_PANORAMA_SENTENCE in prompt.read_text():
        return []
    return [
        f'{_relative(prompt, source)}: remnote-flashcards terminal delivery is missing the Done panorama sentence. '
        f'Add this sentence to retrospect_run_flow: "{REQUIRED_DONE_PANORAMA_SENTENCE}"'
    ]


def _require_sentence(source, relative, sentence, why):
    path = source / relative
    if not path.is_file():
        return [f'{relative}: missing. {why} Add this sentence: "{sentence}"']
    if sentence not in path.read_text():
        return [f'{relative}: {why} Add this sentence: "{sentence}"']
    return []


REQUIRED_NOTION_SYNC_SENTENCE = (
    'When findings is non-empty, POST one page per finding with ntn api v1/pages to data source '
    '「remnote workflow 经验表」; zero findings skip the write; a failed ntn call records notion_sync '
    'and still allows retrospective_recorded.'
)
REQUIRED_NOTION_MAP_SENTENCE = (
    'Map 经验 to the one-line lesson, 日期 to the ISO run date, 主题 to the RemNote theme title, '
    '类别 to 文案清晰度|工具链路|流程门禁|其他, 详情 to evidence + mechanism + suggested change + verification, '
    'and 编码状态 to 待编码 unless the finding already points at a merged fix (已编码) or is a non-code lesson (不适用).'
)
REQUIRED_NOTION_AUTH_SENTENCE = (
    'Auth is ntn login; headless hosts set NOTION_KEYRING=0. Do not require an integration API key.'
)
REQUIRED_NOTION_CONFIG_SENTENCE = (
    'FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID overrides the data source id; '
    'FANLOOP_REMN_EXPERIENCE_DATABASE_ID overrides the database id. '
    'Defaults live in references/notion-experience.md.'
)
NOTION_EXPERIENCE_SENTENCES = (
    REQUIRED_NOTION_SYNC_SENTENCE,
    REQUIRED_NOTION_MAP_SENTENCE,
    REQUIRED_NOTION_AUTH_SENTENCE,
    REQUIRED_NOTION_CONFIG_SENTENCE,
)
NOTION_EXPERIENCE_SENTENCE_FILES = (
    'fanloop-workflow/remnote-flashcards/prompt.yaml',
    'fanloop-workflow/remnote-flashcards/SKILL.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/SKILL.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md',
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/notion-experience.md',
)
NOTION_EXPERIENCE_REFERENCE = (
    'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/notion-experience.md'
)
NOTION_EXPERIENCE_DATABASE_ID = '00000000-0000-4000-8000-000000000001'
NOTION_EXPERIENCE_DATA_SOURCE_ID = '00000000-0000-4000-8000-000000000002'
NOTION_EXPERIENCE_TITLE = 'remnote workflow 经验表'


def notion_experience_sync_errors(source):
    errors = []
    why = 'retrospect_run must POST durable findings with ntn and keep delivery success when that call fails.'
    for relative in NOTION_EXPERIENCE_SENTENCE_FILES:
        for sentence in NOTION_EXPERIENCE_SENTENCES:
            errors.extend(_require_sentence(source, relative, sentence, why))
    reference = source / NOTION_EXPERIENCE_REFERENCE
    if reference.is_file():
        text = reference.read_text()
        for token, label in (
            (NOTION_EXPERIENCE_TITLE, 'database title'),
            (NOTION_EXPERIENCE_DATABASE_ID, 'database id'),
            (NOTION_EXPERIENCE_DATA_SOURCE_ID, 'data source id'),
        ):
            if token not in text:
                errors.append(
                    f'{NOTION_EXPERIENCE_REFERENCE}: missing {label} {token}. '
                    'Keep the documented default for 「remnote workflow 经验表」.'
                )
    readme = source / 'README.md'
    if not readme.is_file():
        errors.append('README.md: missing. Name 「remnote workflow 经验表」 and ntn login.')
    else:
        text = readme.read_text()
        if NOTION_EXPERIENCE_TITLE not in text or 'ntn login' not in text:
            errors.append(
                'README.md: name the database 「remnote workflow 经验表」 and auth command `ntn login`.'
            )
    return errors


def learning_card_gate_errors(source):
    errors = []
    slang_why = 'combo kid_clarity plain_words/one_pass must reject the 2026-10-07 trading slang.'
    ref_why = 'combo rem-refs must name flashcards apply TextSegments and object-set-text, not create --content-file.'
    for relative in (
        'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md',
        'fanloop-workflow/remnote-flashcards/skills/remnote-verify/SKILL.md',
        'fanloop-workflow/remnote-flashcards/prompt.yaml',
    ):
        errors.extend(_require_sentence(source, relative, REQUIRED_COMBO_SLANG_SENTENCE, slang_why))
    for relative in (
        'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md',
        'fanloop-workflow/remnote-flashcards/skills/remnote-cli/SKILL.md',
        'fanloop-workflow/remnote-flashcards/prompt.yaml',
    ):
        errors.extend(_require_sentence(source, relative, REQUIRED_COMBO_REF_SENTENCE, ref_why))
    live_why = '2026-10-09 live-run gates stay in the prompt, contract, and CLI skill.'
    live_sentences = (
        REQUIRED_CONTROL_SEARCH_SENTENCE,
        REQUIRED_ENGLISH_SEARCH_SENTENCE,
        REQUIRED_FACTUAL_REVIEW_SENTENCE,
        REQUIRED_ALIAS_COMMAND,
        REQUIRED_ALIAS_ABORT,
        REQUIRED_MOVE_NOTE,
        REQUIRED_REMOVE_TAG,
        REQUIRED_TAG_CREATE,
        REQUIRED_THEME_PARA,
        REQUIRED_PLUGIN_COMMIT,
        REQUIRED_REVIEWER_USER,
    )
    for relative in (
        'fanloop-workflow/remnote-flashcards/prompt.yaml',
        'fanloop-workflow/remnote-flashcards/skills/remnote-flashcards/references/learning-card-contract.md',
    ):
        for sentence in live_sentences:
            errors.extend(_require_sentence(source, relative, sentence, live_why))
    for sentence in (
        REQUIRED_CONTROL_SEARCH_SENTENCE,
        REQUIRED_ENGLISH_SEARCH_SENTENCE,
        REQUIRED_ALIAS_COMMAND,
        REQUIRED_ALIAS_ABORT,
        REQUIRED_MOVE_NOTE,
        REQUIRED_REMOVE_TAG,
        'CLI port occupied',
        '`validated` 和 `planHash`',
        'get-todays-doc` 是写入',
        'npm run setup && npm run build',
        'background processes do not survive reboot',
        'Node >= 22.13',
    ):
        errors.extend(_require_sentence(
            source,
            'fanloop-workflow/remnote-flashcards/skills/remnote-cli/SKILL.md',
            sentence,
            live_why,
        ))
    for sentence in (
        REQUIRED_FACTUAL_REVIEW_SENTENCE,
        REQUIRED_PLUGIN_COMMIT,
        REQUIRED_REVIEWER_USER,
    ):
        errors.extend(_require_sentence(
            source,
            'fanloop-workflow/remnote-flashcards/skills/remnote-verify/SKILL.md',
            sentence,
            live_why,
        ))
    flow = source / 'fanloop-workflow/remnote-flashcards/flow.yaml'
    if flow.is_file():
        flow_text = flow.read_text()
        for condition_id in LIVE_RUN_ROUTE_CONDITIONS:
            if condition_id not in flow_text:
                errors.append(
                    f'fanloop-workflow/remnote-flashcards/flow.yaml: missing route condition {condition_id}. '
                    'The engine only advances when the forward route lists it.'
                )
        if 'panorama_presented' not in flow_text:
            errors.append('fanloop-workflow/remnote-flashcards/flow.yaml: panorama_presented must stay on every forward route.')
    package = source / 'fanloop-workflow' / 'remnote-flashcards'
    if not package.is_dir():
        return errors
    for path in sorted(package.rglob('*')):
        if path.is_symlink() or not path.is_file() or path.suffix not in {'.md', '.yaml', '.yml'}:
            continue
        text = path.read_text()
        relative = _relative(path, source)
        for error in combo_slang_errors(text):
            errors.append(f'{relative}: {error}')
        for error in combo_markdown_ref_errors(text):
            errors.append(f'{relative}: {error}')
        if relative in COMBO_REF_TEACHING_FILES:
            for error in combo_id_teaching_errors(text):
                errors.append(f'{relative}: {error}')
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
    skills_root = source / 'fanloop-workflow'
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
    errors.extend(remnote_done_prompt_errors(source, workflows))
    return errors


def reject_text(path, checker, ok_message):
    errors = checker(path.read_text())
    if not errors:
        print(ok_message)
        return 0
    for error in errors:
        print(f'{path}: {error}', file=sys.stderr)
    return 1


def reject_file_save_evidence(path):
    return reject_text(path, file_save_delivery_errors, 'panorama evidence has no file-save delivery claim')


def select_workflows(source, requested):
    directories = sorted((source / 'fanloop-workflow').iterdir())
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
                        help='Check panorama Skill delivery text, including the local_agent parent/child handoff and the post-complete Done panorama, then exit')
    parser.add_argument('--check-learning-card-gates', action='store_true',
                        help='Check remnote-flashcards slang and combo rem-ref instructions, then exit')
    parser.add_argument('--check-notion-experience-sync', action='store_true',
                        help='Check retrospect_run still posts findings with ntn into remnote workflow 经验表, then exit')
    parser.add_argument('--reject-file-save-evidence', type=Path,
                        help='Exit 1 when text claims panorama delivery by saving artifacts/panorama-*')
    parser.add_argument('--reject-missing-done-panorama', type=Path,
                        help='Exit 1 when a completed transcript stops after the retrospect 75%% panorama')
    parser.add_argument('--reject-combo-slang', type=Path,
                        help='Exit 1 when combo text uses unexplained trading slang from the 2026-10-07 run')
    parser.add_argument('--reject-combo-markdown-ref', type=Path,
                        help='Exit 1 when combo rem-refs are taught or written as create --content-file plus [[id:]]')
    args = parser.parse_args()
    if args.reject_file_save_evidence is not None:
        return reject_file_save_evidence(args.reject_file_save_evidence)
    if args.reject_missing_done_panorama is not None:
        return reject_text(
            args.reject_missing_done_panorama,
            missing_done_panorama_errors,
            'panorama transcript includes the Done 100% delivery',
        )
    if args.reject_combo_slang is not None:
        return reject_text(
            args.reject_combo_slang,
            combo_slang_errors,
            'combo text has no unexplained trading slang',
        )
    if args.reject_combo_markdown_ref is not None:
        return reject_text(
            args.reject_combo_markdown_ref,
            combo_markdown_ref_errors,
            'combo rem-refs do not use create --content-file with [[id:]]',
        )
    repo = Path(__file__).resolve().parent.parent
    if args.config_source is None or not args.config_source.is_absolute():
        raise ValueError('configuration source must be absolute')
    source = args.config_source
    workflows = select_workflows(source, args.workflow)
    errors = check_panorama_skills(source, workflows, include_examples=not args.workflow)
    if args.check_learning_card_gates:
        errors.extend(learning_card_gate_errors(source))
    if args.check_notion_experience_sync:
        errors.extend(notion_experience_sync_errors(source))
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    if args.check_panorama_skills or args.check_learning_card_gates or args.check_notion_experience_sync:
        if args.check_panorama_skills:
            count = len(panorama_skill_files(source, workflows, include_examples=not args.workflow))
            print(f'panorama skills: {count} ok')
        if args.check_learning_card_gates:
            print('learning card gates: ok')
        if args.check_notion_experience_sync:
            print('notion experience sync: ok')
        return 0
    if args.binary is None or args.output_root is None:
        raise ValueError('matrix validation requires --binary and --output-root')
    binary = args.binary.resolve(strict=True)
    output = args.output_root.resolve()
    output.mkdir(parents=True, exist_ok=False)
    files = snapshot(source / 'fanloop-workflow')
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
    unchanged = files == snapshot(source / 'fanloop-workflow')
    passed = unchanged and bool(outcomes) and all(item['exit_code'] == 0 for item in outcomes)
    manifest = {'source': str(source), 'files': files}
    (output / 'configuration-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    (output / 'configuration-validation.json').write_text(json.dumps({'configuration': manifest, 'binary_sha256': hashlib.sha256(binary.read_bytes()).hexdigest(), 'source_unchanged': unchanged, 'outcome': 'passed' if passed else 'failed', 'matrices': outcomes}, indent=2) + '\n')
    return 0 if passed else 1


if __name__ == '__main__':
    raise SystemExit(main())
