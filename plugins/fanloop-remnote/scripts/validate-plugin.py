#!/usr/bin/env python3
"""Validate the fanloop-remnote Agent Plugin without network, PyYAML, or RemNote.

Checks plugin.json against the vendored Agent Plugins 1.0.0 schema, parses
each skills/<name>/SKILL.md frontmatter with the standard library, and resolves
relative markdown links. A link that leaves the plugin root fails.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCHEMA_PATH = Path(__file__).resolve().parent / "plugin.schema.json"
WHEN = re.compile(r"(?i)\bwhen\b")
SKILL_NAME = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
FENCE = re.compile(r"```.*?```", re.DOTALL)
LINK = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
SECRET_FILE_NAMES = {"connection.json", ".env", "credentials.json"}
SECRET_TEXT = re.compile(
    r"-----BEGIN [A-Z ]*PRIVATE KEY-----"
    r"|ghp_[A-Za-z0-9]{20,}"
    r"|github_pat_[A-Za-z0-9_]{20,}"
    r"|sk-[A-Za-z0-9]{20,}"
    r"|ntn_[A-Za-z0-9]{20,}"
    r"|secret_[A-Za-z0-9]{20,}"
    r"|xox[baprs]-[A-Za-z0-9-]{10,}"
)


class Failure(Exception):
    pass


def fail(message: str) -> None:
    raise Failure(message)


def load_json(path: Path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        fail(f"{path}: {error}")


def validate_instance(instance, schema, path="$"):
    """Apply the keywords used by plugin.schema.json. Unknown keywords fail closed."""
    allowed = {
        "$schema", "$id", "title", "description", "type", "properties", "required",
        "additionalProperties", "const", "minLength", "maxLength", "pattern", "items",
    }
    unknown = set(schema) - allowed
    if unknown:
        fail(f"schema at {path} uses unsupported keywords: {sorted(unknown)}")
    if "const" in schema and instance != schema["const"]:
        fail(f"{path}: expected constant {schema['const']!r}")
    expected = schema.get("type")
    if expected == "object":
        if not isinstance(instance, dict):
            fail(f"{path}: expected object")
        properties = schema.get("properties", {})
        additional = schema.get("additionalProperties", True)
        for key, value in instance.items():
            if key in properties:
                validate_instance(value, properties[key], f"{path}.{key}")
            elif additional is False:
                fail(f"{path}: unknown field {key}")
            elif isinstance(additional, dict):
                validate_instance(value, additional, f"{path}.{key}")
        for key in schema.get("required", []):
            if key not in instance:
                fail(f"{path}: missing required field {key}")
    elif expected == "string":
        if not isinstance(instance, str):
            fail(f"{path}: expected string")
        if "minLength" in schema and len(instance) < schema["minLength"]:
            fail(f"{path}: shorter than {schema['minLength']}")
        if "maxLength" in schema and len(instance) > schema["maxLength"]:
            fail(f"{path}: longer than {schema['maxLength']}")
        if "pattern" in schema and re.fullmatch(schema["pattern"], instance) is None:
            fail(f"{path}: {instance!r} does not match {schema['pattern']}")
    elif expected == "array":
        if not isinstance(instance, list):
            fail(f"{path}: expected array")
        if "items" in schema:
            for index, item in enumerate(instance):
                validate_instance(item, schema["items"], f"{path}[{index}]")
    elif expected is not None:
        fail(f"{path}: unsupported schema type {expected}")


def parse_scalar(value: str, path: Path) -> str:
    if value in {">", "|", ">-", "|-", ">+", "|+"}:
        fail(f"{path}: keep frontmatter values on one line")
    if len(value) >= 2 and value[0] == value[-1] == '"':
        return value[1:-1].replace('\\"', '"').replace("\\\\", "\\")
    if len(value) >= 2 and value[0] == value[-1] == "'":
        return value[1:-1].replace("''", "'")
    return value


def frontmatter(text: str, path: Path):
    if not text.startswith("---\n"):
        fail(f"{path}: frontmatter must start at the first line")
    end = text.find("\n---\n", 4)
    if end < 0:
        fail(f"{path}: frontmatter is not closed")
    data = {}
    for line in text[4:end].splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if ":" not in line:
            fail(f"{path}: frontmatter line has no key: {line!r}")
        key, value = line.split(":", 1)
        key = key.strip()
        if not key or key in data:
            fail(f"{path}: bad frontmatter key {key!r}")
        data[key] = parse_scalar(value.strip(), path)
    if not data:
        fail(f"{path}: frontmatter must be a mapping")
    return data, text[end + 5 :]


def markdown_links(body: str):
    stripped = FENCE.sub("", body)
    for match in LINK.finditer(stripped):
        dest = match.group(1).strip()
        if dest.startswith("<") and ">" in dest:
            dest = dest[1 : dest.index(">")]
        else:
            if len(dest) >= 2 and dest[0] in "\"'" and dest.rfind(dest[0]) > 0:
                dest = dest[1 : dest.rfind(dest[0])]
            elif " " in dest:
                dest = dest.split()[0]
        if dest.startswith(("#", "mailto:")) or re.match(r"^[a-z][a-z0-9+.-]*://", dest, re.I):
            continue
        yield dest.split("#", 1)[0]


def resolve_markdown_links(path: Path, body: str, plugin_root: Path):
    root = plugin_root.resolve()
    for dest in markdown_links(body):
        if not dest:
            continue
        if dest.startswith(("/", "\\")):
            fail(f"{path}: link escapes plugin root: {dest}")
        target = (path.parent / dest).resolve()
        try:
            target.relative_to(root)
        except ValueError:
            fail(f"{path}: link escapes plugin root: {dest}")
        if not target.is_file():
            fail(f"{path}: unresolved link {dest}")


def check_links(root: Path):
    plugin_root = root.resolve()
    for path in sorted(root.rglob("*.md")):
        if path.is_file():
            resolve_markdown_links(path, path.read_text(encoding="utf-8"), plugin_root)


def validate_skill(directory: Path):
    skill_md = directory / "SKILL.md"
    if not skill_md.is_file():
        fail(f"{directory}: missing SKILL.md")
    data, _body = frontmatter(skill_md.read_text(encoding="utf-8"), skill_md)
    name = data.get("name")
    description = data.get("description")
    if not isinstance(name, str) or not SKILL_NAME.fullmatch(name) or len(name) > 64:
        fail(f"{skill_md}: name must be lowercase kebab-case, 1-64 chars")
    if name != directory.name:
        fail(f"{skill_md}: name {name!r} does not match directory {directory.name!r}")
    if not isinstance(description, str) or not description.strip():
        fail(f"{skill_md}: description is required")
    if len(description) > 1024:
        fail(f"{skill_md}: description is {len(description)} chars; limit is 1024")
    if WHEN.search(description) is None:
        fail(f"{skill_md}: description must say when to use the skill")
    for extra in directory.rglob("SKILL.md"):
        if extra != skill_md:
            fail(f"{extra}: nested SKILL.md is not discovered by Agent Plugins")


def scan_secrets(root: Path):
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if path.name in SECRET_FILE_NAMES:
            fail(f"{path}: secret-bearing filename must not be in the plugin")
        if path.suffix in {".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}:
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        if SECRET_TEXT.search(text):
            fail(f"{path}: possible secret material")


REQUIRED_SNIPPETS = {
    "skills/remnote-flashcards/SKILL.md": (
        "--apply",
        "--idempotency-key",
        "`partial`",
        "`unknown`",
        "最多 7",
        "每次 shell 只跑一个",
        "term_search",
        "atomics_first",
        "short_titles",
        "aliases_after_create",
        "rem_ref_title_ids",
        "star_labels_exact",
        "arrow_scan",
        "theme_literal",
        "tag_path",
        "no_spec_unless_asked",
        "card_cluster",
        "named_methods",
        "redo_reuse",
        "arrow_hits",
        'combo rem-refs use flashcards apply TextSegments {"remId":"..."}',
        "combo plain_words rejects unexplained trading slang",
        "FANLOOP_REMN_EXPERIENCE_DATA_SOURCE_ID overrides the data source id",
        "Auth is ntn login",
        "When findings is non-empty, POST one page per finding",
        "sdk rem object-get-or-create-alias-with-text",
        "SDK 0.0.46 cannot acknowledge abort",
        "reviewer=user",
        "没有字段可以指定已经存在的主题",
        "object-set-type --target-id <新 Rem ID> --args-json '[1]'",
    ),
    "skills/remnote-cli/SKILL.md": (
        "每次 shell 执行只运行一个 `remnote-cli` 命令",
        "`partial`",
        "`unknown`",
        "最多 7 张新卡",
        "CLI port occupied",
        "中文搜索缓存",
        "`validated` 和 `planHash`",
        "get-todays-doc` 是写入",
        "没有字段能指定已有主题",
        "sdk rem object-get-or-create-alias-with-text",
        "SDK 0.0.46 cannot acknowledge abort",
    ),
}


def require_invariants(root: Path):
    if root.resolve() != ROOT.resolve():
        return
    for relative, snippets in REQUIRED_SNIPPETS.items():
        text = (root / relative).read_text(encoding="utf-8")
        for snippet in snippets:
            if snippet not in text:
                fail(f"{relative}: missing required invariant {snippet!r}")


def validate_tree(root: Path):
    manifest_path = root / "plugin.json"
    if not manifest_path.is_file():
        fail(f"{root}: missing plugin.json")
    if (root / "mcp.json").exists():
        fail(f"{root}: mcp.json is not part of this plugin")
    if (root / ".cursor-plugin").exists():
        fail(f"{root}: .cursor-plugin/ is the other plugin format")
    schema = load_json(SCHEMA_PATH)
    manifest = load_json(manifest_path)
    validate_instance(manifest, schema)
    if manifest.get("name") != root.name:
        fail(f"{manifest_path}: name {manifest.get('name')!r} does not match directory {root.name!r}")
    skills = root / "skills"
    if not skills.is_dir():
        fail(f"{root}: missing skills/")
    found = []
    for child in sorted(skills.iterdir()):
        if child.name.startswith("."):
            continue
        if not child.is_dir():
            fail(f"{child}: skills/ entries must be skill directories")
        validate_skill(child)
        found.append(child.name)
    if not found:
        fail(f"{root}: no skills")
    for extra in skills.rglob("SKILL.md"):
        if extra.parent.parent != skills:
            fail(f"{extra}: SKILL.md must be a direct child of skills/<name>/")
    scan_secrets(root)
    check_links(root)
    require_invariants(root)
    return found


def self_test():
    schema = load_json(SCHEMA_PATH)
    bad_name = {
        "$schema": schema["properties"]["$schema"]["const"],
        "name": "Bad_Name",
    }
    try:
        validate_instance(bad_name, schema)
    except Failure:
        pass
    else:
        fail("self-test: uppercase name was accepted")
    extra = {
        "$schema": schema["properties"]["$schema"]["const"],
        "name": "ok-plugin",
        "mcpServers": {},
    }
    try:
        validate_instance(extra, schema)
    except Failure:
        pass
    else:
        fail("self-test: unknown field was accepted")
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp) / "fixture"
        skill = root / "skills" / "demo"
        skill.mkdir(parents=True)
        (root / "plugin.json").write_text(json.dumps({
            "$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
            "name": "fixture",
            "description": "fixture",
            "version": "0.0.0",
            "author": {"name": "fixture"},
        }), encoding="utf-8")
        (skill / "SKILL.md").write_text(
            "---\nname: demo\ndescription: Does a thing.\n---\n\n# Demo\n\nSee [missing](missing.md).\n",
            encoding="utf-8",
        )
        try:
            validate_tree(root)
        except Failure as error:
            if "when to use" not in str(error):
                fail(f"self-test: expected a when-to-use failure, got {error}")
        else:
            fail("self-test: description without Use when was accepted")
        (skill / "SKILL.md").write_text(
            "---\nname: demo\ndescription: Use when testing links.\n---\n\n# Demo\n\nSee [missing](missing.md).\n",
            encoding="utf-8",
        )
        try:
            validate_tree(root)
        except Failure as error:
            if "unresolved link" not in str(error):
                fail(f"self-test: expected an unresolved link, got {error}")
        else:
            fail("self-test: broken link was accepted")
        (skill / "notes.md").write_text("ok\n", encoding="utf-8")
        (skill / "SKILL.md").write_text(
            "---\nname: demo\ndescription: Use when testing links.\n---\n\n# Demo\n\nSee [notes](notes.md).\n",
            encoding="utf-8",
        )
        found = validate_tree(root)
        if found != ["demo"]:
            fail(f"self-test: expected ['demo'], got {found}")
        outside = Path(tmp) / "outside.md"
        outside.write_text("outside\n", encoding="utf-8")
        (skill / "SKILL.md").write_text(
            "---\nname: demo\ndescription: Use when testing escape.\n---\n\n"
            "# Demo\n\nSee [out](../../../outside.md).\n",
            encoding="utf-8",
        )
        try:
            validate_tree(root)
        except Failure as error:
            if "escapes plugin root" not in str(error):
                fail(f"self-test: expected an escaping link, got {error}")
        else:
            fail("self-test: link outside the plugin root was accepted")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--self-test", action="store_true", help="run negative fixtures only")
    args = parser.parse_args()
    try:
        if args.self_test:
            self_test()
            print("self-test passed")
            return
        self_test()
        found = validate_tree(args.root.resolve())
    except Failure as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    print(f"ok: {args.root} ({len(found)} skills)")
    for name in found:
        print(f"  {name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
