from __future__ import annotations

import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
GATE = ROOT / "demo/order-operations-portal/scripts/check-staging-gate.py"


class StagingGateTests(unittest.TestCase):
    def copy_demo(self) -> Path:
        temp_dir = Path(tempfile.mkdtemp(prefix="context-rail-staging-gate-"))
        project = temp_dir / "order-operations-portal"
        shutil.copytree(ROOT / "demo/order-operations-portal", project)
        self.addCleanup(shutil.rmtree, temp_dir)
        return project

    def run_gate(self, project: Path, commit: str = "10876c68ad460d5dbf0edd584c1db7de2e2cdb17") -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["python3", str(GATE), str(project), commit],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_pending_staging_decision_blocks_before_cloud_access(self) -> None:
        result = self.run_gate(self.copy_demo())
        self.assertEqual(result.returncode, 2)
        self.assertIn("DecisionRecord status is not ACCEPTED_FOR_STAGING", result.stderr)

    def test_pass_gate_requires_reviewed_commit_identity(self) -> None:
        project = self.copy_demo()
        decision_path = project / "decisions/DR-001-manual-order-review.json"
        decision = json.loads(decision_path.read_text(encoding="utf-8"))
        decision["status"] = "ACCEPTED_FOR_STAGING"
        decision["human_decisions"]["allow_staging"] = "ACCEPTED"
        decision_path.write_text(json.dumps(decision), encoding="utf-8")
        review_path = project / "evidence/EB-001/code-review.md"
        review_path.write_text(
            "Status: `PASS`\nReviewer: `independent-reviewer`\nReviewed commit: `different-commit`\n",
            encoding="utf-8",
        )
        result = self.run_gate(project)
        self.assertEqual(result.returncode, 2)
        self.assertIn("reviewed commit does not match APPROVED_COMMIT", result.stderr)


if __name__ == "__main__":
    unittest.main()
