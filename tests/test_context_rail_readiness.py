from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
INSPECTOR = ROOT / "scripts" / "context_rail_readiness.py"
PYTHON = ROOT / "template" / ".venv" / "bin" / "python"


class ReadinessInspectionTests(unittest.TestCase):
    def inspect(self, *roots: Path) -> list[dict]:
        result = subprocess.run(
            [str(PYTHON), str(INSPECTOR), "--output", "json", *sum((["--root", str(root)] for root in roots), [])],
            check=True,
            capture_output=True,
            text=True,
        )
        return json.loads(result.stdout)

    def test_two_consumers_are_importable_read_only(self) -> None:
        reports = self.inspect(ROOT / "demo/order-operations-portal", ROOT / "examples/support-insights")
        self.assertEqual([report["project"]["id"] for report in reports], ["order-operations-portal", "support-insights"])
        self.assertEqual(reports[0]["readiness"]["ready_for_decision"]["status"], "READY")
        self.assertEqual(reports[1]["readiness"]["ready_for_decision"]["status"], "READY")
        self.assertEqual(
            set(reports[0]["readiness"]),
            {
                "ready_for_decision",
                "ready_for_local_development",
                "ready_for_cloud_testing",
                "ready_for_staging",
                "production",
            },
        )
        self.assertEqual(reports[0]["readiness"]["production"]["status"], "BLOCKED")

    def test_pending_human_decision_is_not_promoted_to_ready(self) -> None:
        report = self.inspect(ROOT / "demo/order-operations-portal")[0]
        development = report["readiness"]["ready_for_local_development"]
        self.assertEqual(development["status"], "NEEDS_INPUT")
        self.assertTrue(any("no human-accepted" in reason for reason in development["reasons"]))
        self.assertFalse(any("conflicts" in reason for reason in development["reasons"]))

    def test_request_acceptance_is_scoped_from_staging_and_production_gates(self) -> None:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-rail-readiness-"))
        self.addCleanup(shutil.rmtree, temp_dir)
        project = temp_dir / "order-operations-portal"
        shutil.copytree(ROOT / "demo/order-operations-portal", project)
        decision_path = project / "decisions/DR-001-manual-order-review.json"
        decision = json.loads(decision_path.read_text(encoding="utf-8"))
        decision["status"] = "ACCEPTED_FOR_DEVELOPMENT"
        decision["human_decisions"] = {
            "accept_request": "ACCEPTED",
            "allow_staging": "PENDING_STAGING_EVIDENCE",
            "allow_production": "BLOCKED_IN_DEMO",
        }
        decision_path.write_text(json.dumps(decision), encoding="utf-8")
        report = self.inspect(project)[0]
        self.assertEqual(report["readiness"]["ready_for_local_development"]["status"], "READY")
        self.assertEqual(report["readiness"]["ready_for_staging"]["status"], "NEEDS_INPUT")

    def test_rejected_request_is_not_human_accepted(self) -> None:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-rail-readiness-"))
        self.addCleanup(shutil.rmtree, temp_dir)
        project = temp_dir / "order-operations-portal"
        shutil.copytree(ROOT / "demo/order-operations-portal", project)
        decision_path = project / "decisions/DR-001-manual-order-review.json"
        decision = json.loads(decision_path.read_text(encoding="utf-8"))
        decision["status"] = "ACCEPTED"
        decision["human_decisions"] = {
            "accept_request": "REJECTED",
            "allow_staging": "ACCEPTED",
            "allow_production": "ACCEPTED",
        }
        decision_path.write_text(json.dumps(decision), encoding="utf-8")
        report = self.inspect(project)[0]
        development = report["readiness"]["ready_for_local_development"]
        self.assertEqual(development["status"], "NEEDS_INPUT")
        self.assertTrue(any("no human-accepted" in reason for reason in development["reasons"]))

    def test_local_development_does_not_require_cloud_run(self) -> None:
        report = self.inspect(ROOT / "demo/order-operations-portal")[0]
        local = report["readiness"]["ready_for_local_development"]
        staging = report["readiness"]["ready_for_staging"]
        self.assertEqual(local["status"], "NEEDS_INPUT")
        self.assertFalse(any("Cloud Run" in reason for reason in local["reasons"]))
        self.assertEqual(staging["status"], "NEEDS_INPUT")
        self.assertTrue(any("receipt" in reason for reason in staging["reasons"]))

    def test_changed_source_is_stale(self) -> None:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-rail-readiness-"))
        self.addCleanup(shutil.rmtree, temp_dir)
        source = ROOT / "examples/support-insights"
        shutil.copytree(source, temp_dir / "support-insights")
        project = temp_dir / "support-insights"
        (project / "vision.md").write_text((project / "vision.md").read_text() + "\nChanged during test.\n", encoding="utf-8")
        report = self.inspect(project)[0]
        context_pack = next(item for item in report["documents"] if item["path"] == "docs/ai/context-pack.json")
        self.assertEqual(context_pack["status"], "STALE")


if __name__ == "__main__":
    unittest.main()
