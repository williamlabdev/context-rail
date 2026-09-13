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
            "Status: `PASS`\nReview type: `AI_REVIEW`\nReviewer: `ContextRail AI review agent`\nReviewer actor_id: `context-rail-ai-reviewer-001`\nReviewed commit: `different-commit`\n",
            encoding="utf-8",
        )
        result = self.run_gate(project)
        self.assertEqual(result.returncode, 2)
        self.assertIn("reviewed commit does not match APPROVED_COMMIT", result.stderr)

    def test_issued_work_order_cannot_share_reviewer_actor(self) -> None:
        project = self.copy_demo()
        decision_path = project / "decisions/DR-001-manual-order-review.json"
        decision = json.loads(decision_path.read_text(encoding="utf-8"))
        decision["status"] = "ACCEPTED_FOR_STAGING"
        decision["human_decisions"]["allow_staging"] = {"decision": "ACCEPTED", "actor_id": "founder-001"}
        decision_path.write_text(json.dumps(decision), encoding="utf-8")
        role_path = project / "governance/role-assignments.json"
        roles = json.loads(role_path.read_text(encoding="utf-8"))
        roles["operating_mode"] = "multi_operator"
        founder = next(actor for actor in roles["actors"] if actor["actor_id"] == "founder-001")
        founder["roles"].append("independent_reviewer")
        role_path.write_text(json.dumps(roles), encoding="utf-8")
        review_path = project / "evidence/EB-001/code-review.md"
        review_path.write_text(
            "Status: `PASS`\nReview type: `HUMAN_REVIEW`\nReviewer: `Founder`\nReviewer actor_id: `founder-001`\nReviewed commit: `10876c68ad460d5dbf0edd584c1db7de2e2cdb17`\n",
            encoding="utf-8",
        )
        work_order_path = project / "work-orders/AWO-001-manual-order-review.json"
        work_order = json.loads(work_order_path.read_text(encoding="utf-8"))
        work_order["status"] = "ISSUED"
        work_order["issuer"]["status"] = "ISSUED"
        work_order_path.write_text(json.dumps(work_order), encoding="utf-8")
        result = self.run_gate(project)
        self.assertEqual(result.returncode, 2)
        self.assertIn("issuer and independent reviewer must be different", result.stderr)

    def test_single_operator_allows_low_risk_staging_with_compensating_controls(self) -> None:
        project = self.copy_demo()
        decision_path = project / "decisions/DR-001-manual-order-review.json"
        decision = json.loads(decision_path.read_text(encoding="utf-8"))
        decision["status"] = "ACCEPTED_FOR_STAGING"
        decision["human_decisions"] = {
            "accept_request": {"decision": "ACCEPTED", "actor_id": "founder-001"},
            "allow_staging": {"decision": "ACCEPTED", "actor_id": "founder-001"},
            "allow_production": {"decision": "BLOCKED_IN_DEMO", "actor_id": None},
        }
        decision_path.write_text(json.dumps(decision), encoding="utf-8")
        review_path = project / "evidence/EB-001/code-review.md"
        review_path.write_text(
            "Status: `PASS`\nReview type: `AI_REVIEW`\nReviewer: `ContextRail AI review agent`\nReviewer actor_id: `context-rail-ai-reviewer-001`\nReviewed commit: `10876c68ad460d5dbf0edd584c1db7de2e2cdb17`\n",
            encoding="utf-8",
        )
        controls_path = project / "evidence/EB-001/single-operator-controls.md"
        controls_path.write_text(
            "Status: `PASS`\n"
            "Control ai-review: `PASS`\n"
            "Control tests: `PASS`\n"
            "Control build: `PASS`\n"
            "Control production-block: `PASS`\n",
            encoding="utf-8",
        )
        work_order_path = project / "work-orders/AWO-001-manual-order-review.json"
        work_order = json.loads(work_order_path.read_text(encoding="utf-8"))
        work_order["status"] = "ISSUED"
        work_order["issuer"]["status"] = "ISSUED"
        work_order_path.write_text(json.dumps(work_order), encoding="utf-8")
        result = self.run_gate(project)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("STAGING GATE PASS", result.stdout)

    def test_single_operator_without_controls_remains_blocked(self) -> None:
        project = self.copy_demo()
        decision_path = project / "decisions/DR-001-manual-order-review.json"
        decision = json.loads(decision_path.read_text(encoding="utf-8"))
        decision["status"] = "ACCEPTED_FOR_STAGING"
        decision["human_decisions"]["allow_staging"] = {"decision": "ACCEPTED", "actor_id": "founder-001"}
        decision_path.write_text(json.dumps(decision), encoding="utf-8")
        review_path = project / "evidence/EB-001/code-review.md"
        review_path.write_text(
            "Status: `PASS`\nReview type: `AI_REVIEW`\nReviewer: `ContextRail AI review agent`\nReviewer actor_id: `context-rail-ai-reviewer-001`\nReviewed commit: `10876c68ad460d5dbf0edd584c1db7de2e2cdb17`\n",
            encoding="utf-8",
        )
        work_order_path = project / "work-orders/AWO-001-manual-order-review.json"
        work_order = json.loads(work_order_path.read_text(encoding="utf-8"))
        work_order["status"] = "ISSUED"
        work_order["issuer"]["status"] = "ISSUED"
        work_order_path.write_text(json.dumps(work_order), encoding="utf-8")
        result = self.run_gate(project)
        self.assertEqual(result.returncode, 2)
        self.assertIn("compensating-control evidence is not PASS", result.stderr)


if __name__ == "__main__":
    unittest.main()
