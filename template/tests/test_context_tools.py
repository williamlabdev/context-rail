from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
VALIDATOR = ROOT / "scripts" / "validate_project_context.py"
REBUILDER = ROOT / "scripts" / "rebuild_context_pack.py"


class ContextToolTests(unittest.TestCase):
    def copy_template(self) -> Path:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-template-test-"))
        for source in ROOT.iterdir():
            if source.name in {".venv", "tests"}:
                continue
            destination = temp_dir / source.name
            if source.is_dir():
                shutil.copytree(source, destination)
            else:
                shutil.copy2(source, destination)
        self.addCleanup(shutil.rmtree, temp_dir)
        return temp_dir

    def run_validator(self, root: Path, *arguments: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(VALIDATOR), "--root", str(root), *arguments],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_template_structure_passes(self) -> None:
        result = self.run_validator(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_strict_mode_reports_placeholders(self) -> None:
        result = self.run_validator(ROOT, "--strict")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("PLACEHOLDER remains in project.yaml", result.stdout)

    def test_declared_document_must_exist(self) -> None:
        root = self.copy_template()
        project_yaml = root / "project.yaml"
        project_yaml.write_text(project_yaml.read_text().replace("README.md\n", "missing.md\n"), encoding="utf-8")
        result = self.run_validator(root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("DECLARED DOCUMENT MISSING missing.md", result.stdout)

    def test_environment_sequence_must_be_unique(self) -> None:
        root = self.copy_template()
        project_yaml = root / "project.yaml"
        project_yaml.write_text(project_yaml.read_text().replace("sequence: 2", "sequence: 1"), encoding="utf-8")
        result = self.run_validator(root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("environment sequence values must be unique", result.stdout)

    def test_context_pack_must_remain_derived(self) -> None:
        root = self.copy_template()
        pack_path = root / "docs/ai/context-pack.json"
        pack = json.loads(pack_path.read_text(encoding="utf-8"))
        pack["derived"] = False
        pack_path.write_text(json.dumps(pack), encoding="utf-8")
        result = self.run_validator(root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("context pack must be marked derived", result.stdout)

    def test_rebuild_writes_lineage_hashes(self) -> None:
        root = self.copy_template()
        result = subprocess.run(
            [sys.executable, str(REBUILDER), "--root", str(root)],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        pack = json.loads((root / "docs/ai/context-pack.json").read_text(encoding="utf-8"))
        self.assertTrue(pack["source_snapshot_hash"].startswith("sha256:"))
        self.assertTrue(all(source["content_hash"].startswith("sha256:") for source in pack["sources"]))
        self.assertIn("docs/governance/policies.md", [source["path"] for source in pack["sources"]])
        self.assertIn("governance/role-assignments.json", [source["path"] for source in pack["sources"]])

    def test_validator_rejects_changed_source_hash(self) -> None:
        root = self.copy_template()
        pack_path = root / "docs/ai/context-pack.json"
        pack = json.loads(pack_path.read_text(encoding="utf-8"))
        pack["sources"][0]["content_hash"] = "sha256:" + ("0" * 64)
        pack_path.write_text(json.dumps(pack), encoding="utf-8")
        result = self.run_validator(root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("CONTEXT SOURCE HASH MISMATCH", result.stdout)

    def test_validator_rejects_source_coverage_drift(self) -> None:
        root = self.copy_template()
        pack_path = root / "docs/ai/context-pack.json"
        pack = json.loads(pack_path.read_text(encoding="utf-8"))
        pack["sources"] = pack["sources"][:-1]
        pack_path.write_text(json.dumps(pack), encoding="utf-8")
        result = self.run_validator(root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly match declared source-of-truth", result.stdout)


if __name__ == "__main__":
    unittest.main()
