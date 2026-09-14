from __future__ import annotations

import json
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TEMPLATE = ROOT / "template"


class TemplateInstantiationTests(unittest.TestCase):
    def fresh_project(self) -> Path:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-rail-template-smoke-"))
        project = temp_dir / "sample-project"
        shutil.copytree(
            TEMPLATE,
            project,
            ignore=shutil.ignore_patterns(".venv", "tests", "__pycache__"),
        )
        placeholder = re.compile(r"<REPLACE_ME[^\r\n>]*>")
        for path in project.rglob("*"):
            if path.is_file():
                text = path.read_text(encoding="utf-8")
                path.write_text(placeholder.sub("fixture-value", text), encoding="utf-8")
        self.addCleanup(shutil.rmtree, temp_dir)
        return project

    def test_fresh_project_rebuilds_and_validates_both_policy_modes(self) -> None:
        project = self.fresh_project()
        rebuild = subprocess.run(
            [sys.executable, str(project / "scripts/rebuild_context_pack.py"), "--root", str(project)],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(rebuild.returncode, 0, rebuild.stdout + rebuild.stderr)
        validate = subprocess.run(
            [sys.executable, str(project / "scripts/validate_project_context.py"), "--root", str(project), "--strict"],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(validate.returncode, 0, validate.stdout + validate.stderr)

        role_path = project / "governance/role-assignments.json"
        roles = json.loads(role_path.read_text(encoding="utf-8"))
        self.assertEqual(roles["operating_mode"], "multi_operator")
        self.assertFalse(roles["staging_policy"]["self_approval"])
        self.assertTrue(roles["production_policy"]["requires_distinct_human"])

        roles["operating_mode"] = "single_operator"
        roles["staging_policy"] = {
            "mode": "self_approval_with_compensating_controls",
            "allowed_risk_levels": ["low"],
            "required_controls": ["ai-review", "tests", "build", "production-block"],
            "self_approval": True,
        }
        role_path.write_text(json.dumps(roles), encoding="utf-8")
        self.assertEqual(roles["operating_mode"], "single_operator")
        self.assertTrue(roles["staging_policy"]["self_approval"])
        self.assertTrue(roles["production_policy"]["requires_distinct_human"])


if __name__ == "__main__":
    unittest.main()
