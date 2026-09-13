#!/usr/bin/env python3

"""Validate the demo's pre-deploy staging evidence without changing state."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path


def field(text: str, label: str) -> str:
    match = re.search(rf"(?m)^\s*{re.escape(label)}:\s*`?([^`\n]+?)`?\s*$", text)
    return match.group(1).strip() if match else ""


def evidence_controls_pass(root: Path, required_controls: list[str]) -> bool:
    path = root / "evidence/EB-001/single-operator-controls.md"
    if not path.is_file():
        return False
    text = path.read_text(encoding="utf-8")
    if field(text, "Status").upper() != "PASS":
        return False
    return all(field(text, f"Control {control}").upper() == "PASS" for control in required_controls)


def action_value(human_decisions: dict, action: str) -> str:
    value = human_decisions.get(action, "")
    if isinstance(value, dict):
        value = value.get("decision", "")
    return str(value).upper()


def fail(message: str) -> int:
    print(f"STAGING GATE BLOCKED: {message}", file=sys.stderr)
    return 2


def main() -> int:
    if len(sys.argv) != 3:
        return fail("usage: check-staging-gate.py PROJECT_ROOT APPROVED_COMMIT")
    root = Path(sys.argv[1]).resolve()
    approved_commit = sys.argv[2].strip()
    if not approved_commit:
        return fail("APPROVED_COMMIT is required")

    try:
        decision = json.loads(
            (root / "decisions/DR-001-manual-order-review.json").read_text(encoding="utf-8")
        )
        context_pack = json.loads((root / "docs/ai/context-pack.json").read_text(encoding="utf-8"))
        review_text = (root / "evidence/EB-001/code-review.md").read_text(encoding="utf-8")
        role_assignments = json.loads((root / "governance/role-assignments.json").read_text(encoding="utf-8"))
        work_order = json.loads(
            (root / "work-orders/AWO-001-manual-order-review.json").read_text(encoding="utf-8")
        )
    except (OSError, json.JSONDecodeError) as exc:
        return fail(f"required governance artifact cannot be read: {exc}")

    if decision.get("status") != "ACCEPTED_FOR_STAGING":
        return fail("DecisionRecord status is not ACCEPTED_FOR_STAGING")
    human_decisions = decision.get("human_decisions", {})
    if action_value(human_decisions, "allow_staging") not in {
        "ACCEPTED",
        "APPROVED",
        "TRUE",
    }:
        return fail("human staging approval is not recorded")
    if context_pack.get("derived") is not True or context_pack.get("status") != "CURRENT":
        return fail("Context Pack is not CURRENT")
    if context_pack.get("source_snapshot_hash") != decision.get("source_snapshot_hash"):
        return fail("DecisionRecord and Context Pack snapshots do not match")
    operating_mode = str(role_assignments.get("operating_mode", "multi_operator")).lower()
    if operating_mode not in {"single_operator", "multi_operator"}:
        return fail("role assignment operating_mode is invalid")
    staging_policy = role_assignments.get("staging_policy", {})
    risk_level = str(decision.get("risk_level", "")).lower()
    allowed_risk_levels = {str(value).lower() for value in staging_policy.get("allowed_risk_levels", [])}
    if risk_level not in allowed_risk_levels:
        return fail("DecisionRecord risk level is not allowed by the staging policy")
    if field(review_text, "Status").upper() != "PASS":
        return fail("required code review evidence is not PASS")
    review_type = field(review_text, "Review type").upper()
    reviewer = field(review_text, "Reviewer").lower()
    reviewer_actor_id = field(review_text, "Reviewer actor_id")
    if not reviewer or reviewer in {"unassigned", "unknown", "pending"}:
        return fail("independent reviewer identity is missing")
    if not reviewer_actor_id:
        return fail("independent reviewer actor_id is missing")
    if field(review_text, "Reviewed commit") != approved_commit:
        return fail("reviewed commit does not match APPROVED_COMMIT")
    actors = {
        actor.get("actor_id"): actor
        for actor in role_assignments.get("actors", [])
        if isinstance(actor, dict) and actor.get("actor_id")
    }
    if reviewer_actor_id not in actors:
        return fail("reviewer actor_id is not declared in role assignments")
    reviewer_actor = actors[reviewer_actor_id]
    if operating_mode == "single_operator":
        if review_type != "AI_REVIEW":
            return fail("single_operator staging requires an explicit AI_REVIEW")
        if reviewer_actor.get("identity_type") != "agent" or "ai_reviewer" not in reviewer_actor.get("roles", []):
            return fail("AI reviewer actor must be a declared ai_reviewer agent")
        required_controls = [str(value) for value in staging_policy.get("required_controls", [])]
        if not evidence_controls_pass(root, required_controls):
            return fail("single_operator compensating-control evidence is not PASS")
    else:
        if review_type not in {"", "HUMAN_REVIEW"}:
            return fail("multi_operator staging requires human review evidence")
        if reviewer_actor.get("identity_type") != "human" or "independent_reviewer" not in reviewer_actor.get("roles", []):
            return fail("multi_operator reviewer must be a declared independent human reviewer")
        if reviewer_actor.get("status") == "UNASSIGNED":
            return fail("multi_operator reviewer is not assigned")
    if work_order.get("status") in {"READY", "ISSUED", "ACCEPTED_FOR_EXECUTION"}:
        issuer = work_order.get("issuer", {})
        issuer_actor_id = issuer.get("actor_id")
        issuer_role = issuer.get("actor_role")
        if not issuer_actor_id or issuer.get("status") == "NOT_ISSUED":
            return fail("active Work Order has no issued actor identity")
        if issuer_actor_id not in actors:
            return fail("Work Order issuer actor_id is not declared in role assignments")
        if issuer_role not in actors[issuer_actor_id].get("roles", []):
            return fail("Work Order issuer role is not assigned to that actor")
        if issuer_actor_id == reviewer_actor_id and operating_mode == "multi_operator":
            return fail("Work Order issuer and independent reviewer must be different actors")
        staging_decision = human_decisions.get("allow_staging", {})
        staging_actor_id = staging_decision.get("actor_id") if isinstance(staging_decision, dict) else None
        if not staging_actor_id:
            return fail("human staging approval actor_id is missing")
        if staging_actor_id not in actors:
            return fail("staging approver actor_id is not declared in role assignments")
        if actors[staging_actor_id].get("identity_type") != "human":
            return fail("staging approver must be a human actor")
        if staging_actor_id == reviewer_actor_id:
            return fail("staging approver must be different from reviewer")
        if staging_actor_id == issuer_actor_id and not staging_policy.get("self_approval", False):
            return fail("staging self-approval is not allowed by the selected policy")
        if operating_mode == "multi_operator" and staging_actor_id == issuer_actor_id:
            return fail("staging approver must be different from issuer and reviewer")
    print("STAGING GATE PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
