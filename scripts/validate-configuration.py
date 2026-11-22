#!/usr/bin/env python3
"""Validate an optional synthetic Workflow source; current production inventory is empty."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def tree_snapshot(root: Path) -> dict[str, str]:
    if not root.exists():
        return {}
    result = {}
    for path in sorted(root.rglob("*")):
        if path.is_symlink():
            raise ValueError(f"configuration source contains symlink: {path}")
        if path.is_file():
            result[path.relative_to(root).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    return result


def packages(source: Path) -> dict[str, Path]:
    root = source / "fanloop-workflow"
    if not root.exists():
        return {}
    if root.is_symlink() or not root.is_dir():
        raise ValueError("fanloop-workflow must be a real directory")
    result = {}
    for path in sorted(root.iterdir()):
        if path.is_symlink() or not path.is_dir():
            raise ValueError(f"invalid Workflow package entry: {path.name}")
        if not (path / "SKILL.md").is_file() or not (path / "workflow.yaml").is_file():
            raise ValueError(f"incomplete Workflow package: {path.name}")
        result[path.name] = path
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config-source", type=Path)
    parser.add_argument("--binary", type=Path)
    parser.add_argument("--output-root", type=Path)
    parser.add_argument("--workflow", action="append", help="synthetic Workflow id; repeatable")
    args = parser.parse_args()
    if args.config_source is None:
        print("production Workflow inventory: empty")
        return 0
    if not args.config_source.is_absolute():
        print("configuration source must be absolute", file=sys.stderr)
        return 2
    try:
        source = args.config_source.resolve(strict=True)
        available = packages(source)
    except (OSError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    selected = sorted(set(args.workflow or available))
    unknown = set(selected) - set(available)
    if unknown:
        print("unknown Workflow: " + ", ".join(sorted(unknown)), file=sys.stderr)
        return 2
    snapshot = tree_snapshot(source / "fanloop-workflow")
    if not selected:
        if args.output_root:
            output = args.output_root.resolve()
            output.mkdir(parents=True, exist_ok=False)
            (output / "configuration-manifest.json").write_text(
                json.dumps({"source": str(source), "files": snapshot}, indent=2) + "\n",
                encoding="utf-8",
            )
        print("production Workflow inventory: empty")
        return 0
    if args.binary is None or args.output_root is None:
        print("synthetic matrices require --binary and --output-root", file=sys.stderr)
        return 2
    binary = args.binary.resolve(strict=True)
    output = args.output_root.resolve()
    output.mkdir(parents=True, exist_ok=False)
    data = output / "data"
    fake = output / "bin"
    fake.mkdir()
    lark = ROOT / "tests/requirement_e2e/testdata/lark-cli"
    if lark.is_file():
        fake_lark = fake / "lark-cli"
        fake_lark.write_bytes(lark.read_bytes())
        fake_lark.chmod(0o755)
    environment = {key: value for key, value in os.environ.items() if key not in ("BOTMUX_CHAT_ID", "BOTMUX_SESSION_ID")}
    environment.update(FANLOOP_DATA_HOME=str(data), FANLOOP_CONFIG_SOURCE=str(source), PATH=str(fake) + os.pathsep + environment["PATH"])
    for kind in ("CODEX", "AGENT", "TRAE", "CLAUDE"):
        skills = output / "skill-roots" / kind.lower()
        skills.mkdir(parents=True)
        (skills / "fanloop-workflow").symlink_to(binary.parent.parent / "entrypoints" / "fanloop-workflow")
        environment[f"FANLOOP_{kind}_SKILLS_ROOT"] = str(skills)
    outcomes = []
    for workflow in selected:
        argv = [sys.executable, str(ROOT / "tests/requirement_e2e/route_matrix.py"), "--source-root", str(ROOT),
                "--binary", str(binary), "--workflow", workflow, "--output-root", str(output / workflow)]
        result = subprocess.run(argv, env=environment)
        outcomes.append({"workflow": workflow, "exit_code": result.returncode, "argv": argv})
        print(f"{workflow}: {result.returncode}")
    unchanged = snapshot == tree_snapshot(source / "fanloop-workflow")
    passed = unchanged and all(item["exit_code"] == 0 for item in outcomes)
    (output / "configuration-manifest.json").write_text(
        json.dumps({"source": str(source), "files": snapshot}, indent=2) + "\n", encoding="utf-8"
    )
    (output / "configuration-validation.json").write_text(
        json.dumps({"outcome": "passed" if passed else "failed", "source_unchanged": unchanged, "matrices": outcomes}, indent=2) + "\n",
        encoding="utf-8",
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
