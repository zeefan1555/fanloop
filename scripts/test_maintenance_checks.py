#!/usr/bin/env python3
"""Offline checks for the retired product Workflow inventory."""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts/validate-configuration.py"


class MaintenanceTests(unittest.TestCase):
    def test_help(self) -> None:
        result = subprocess.run([sys.executable, str(SCRIPT), "--help"], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_empty_production_inventory(self) -> None:
        result = subprocess.run([sys.executable, str(SCRIPT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("empty", result.stdout)

    def test_empty_source_is_a_valid_zero_package_set(self) -> None:
        with tempfile.TemporaryDirectory(prefix="fanloop-empty-config-") as temporary:
            root = Path(temporary)
            output = root / "evidence"
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--config-source", str(root), "--output-root", str(output)],
                capture_output=True, text=True,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            manifest = json.loads((output / "configuration-manifest.json").read_text())
            self.assertEqual(manifest["files"], {})

    def test_source_symlink_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory(prefix="fanloop-symlink-config-") as temporary:
            root = Path(temporary)
            (root / "fanloop-workflow").symlink_to(root, target_is_directory=True)
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--config-source", str(root)],
                capture_output=True, text=True,
            )
            self.assertEqual(result.returncode, 1)

    def test_unknown_requested_workflow_fails(self) -> None:
        with tempfile.TemporaryDirectory(prefix="fanloop-empty-config-") as temporary:
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--config-source", temporary, "--workflow", "missing"],
                capture_output=True, text=True,
            )
            self.assertEqual(result.returncode, 2)
            self.assertIn("unknown Workflow", result.stderr)


if __name__ == "__main__":
    unittest.main()
