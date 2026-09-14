from __future__ import annotations

import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REGISTRY = ROOT / "scripts/context_rail_registry.py"
PYTHON = ROOT / "template/.venv/bin/python"


class ProjectRegistryImportTests(unittest.TestCase):
    def import_projects(self, *roots: Path) -> subprocess.CompletedProcess[str]:
        arguments = sum((["--root", str(root)] for root in roots), [])
        return subprocess.run(
            [str(PYTHON), str(REGISTRY), *arguments],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_imports_multiple_project_shapes_read_only(self) -> None:
        roots = (ROOT / "demo/order-operations-portal", ROOT / "examples/support-insights")
        before = {
            root: (root / "project.yaml").read_bytes()
            for root in roots
        }
        result = self.import_projects(*roots)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        snapshot = json.loads(result.stdout)
        self.assertEqual(snapshot["kind"], "ProjectRegistrySnapshot")
        self.assertTrue(all(project["read_only"] for project in snapshot["projects"]))
        by_id = {project["project"]["id"]: project for project in snapshot["projects"]}
        self.assertEqual(set(by_id), {"order-operations-portal", "support-insights"})
        self.assertEqual(by_id["order-operations-portal"]["services"][0]["runtime"], "go")
        self.assertEqual(by_id["support-insights"]["services"][0]["runtime"], "UNDECLARED")
        self.assertEqual(
            [environment["id"] for environment in by_id["order-operations-portal"]["environments"]],
            ["development", "testing", "staging", "production"],
        )
        self.assertEqual(by_id["order-operations-portal"]["context"]["status"], "DERIVED")
        self.assertEqual(by_id["support-insights"]["context"]["status"], "STALE")
        support_context = next(
            document
            for document in by_id["support-insights"]["documents"]
            if document["path"] == "docs/ai/context-pack.json"
        )
        self.assertIn("README.md", support_context["stale_sources"])
        for root, content in before.items():
            self.assertEqual((root / "project.yaml").read_bytes(), content)

    def test_invalid_project_is_blocked_without_partial_registry_result(self) -> None:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-rail-registry-"))
        self.addCleanup(shutil.rmtree, temp_dir)
        invalid = temp_dir / "invalid"
        invalid.mkdir()
        (invalid / "project.yaml").write_text("kind: NotAProject\n", encoding="utf-8")
        result = self.import_projects(invalid)
        self.assertEqual(result.returncode, 2)
        self.assertIn("PROJECT REGISTRY IMPORT BLOCKED", result.stderr)
        self.assertEqual(result.stdout, "")


if __name__ == "__main__":
    unittest.main()
