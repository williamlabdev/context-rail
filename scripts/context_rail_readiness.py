#!/usr/bin/env python3

"""Read-only Project import and readiness inspection for ContextRail."""

from __future__ import annotations

import argparse
import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import yaml


READINESS = (
    "ready_for_decision",
    "ready_for_local_development",
    "ready_for_cloud_testing",
    "ready_for_staging",
    "staging_verified",
    "production",
)


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def load_json(path: Path) -> dict[str, Any] | None:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
    return value if isinstance(value, dict) else None


def document_status(root: Path, document: dict[str, Any]) -> dict[str, Any]:
    relative = str(document.get("path", ""))
    path = root / relative
    result = {
        "path": relative,
        "kind": document.get("kind"),
        "source_of_truth": document.get("sourceOfTruth", True),
        "status": "CURRENT" if path.is_file() else "MISSING",
    }
    if not path.is_file():
        return result
    if document.get("sourceOfTruth", True) is not False:
        return result
    pack = load_json(path)
    result["status"] = "DERIVED" if pack and pack.get("derived") is True else "NEEDS_INPUT"
    stale_sources: list[str] = []
    for source in (pack or {}).get("sources", []):
        source_path = root / str(source.get("path", ""))
        expected = str(source.get("content_hash", ""))
        if source_path.is_file() and expected != f"sha256:{sha256(source_path)}":
            stale_sources.append(str(source.get("path")))
    if stale_sources:
        result["status"] = "STALE"
        result["stale_sources"] = stale_sources
    return result


def find_decisions(root: Path) -> list[dict[str, Any]]:
    decisions: list[dict[str, Any]] = []
    directory = root / "decisions"
    if not directory.is_dir():
        return decisions
    for path in sorted(directory.glob("*.json")):
        value = load_json(path)
        if value:
            decisions.append(value)
    return decisions


def decision_is_human_accepted(decision: dict[str, Any], action: str = "accept_request") -> bool:
    required_statuses = {
        "accept_request": {"ACCEPTED", "ACCEPTED_FOR_DEVELOPMENT", "ACCEPTED_FOR_STAGING"},
        "allow_staging": {"ACCEPTED_FOR_STAGING"},
    }
    if decision.get("status") not in required_statuses.get(action, set()):
        return False
    human = decision.get("human_decision") or decision.get("human_decisions") or {}
    action_value = human.get(action, "")
    if isinstance(action_value, dict):
        action_value = action_value.get("decision", "")
    return str(action_value).upper() in {"ACCEPTED", "APPROVED", "TRUE"}


def markdown_field(path: Path, label: str) -> str:
    prefix = f"{label}:"
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.strip().startswith(prefix):
            return line.split(":", 1)[1].strip().strip("`")
    return ""


def review_evidence_state(root: Path) -> tuple[str, list[str]]:
    review_files = list((root / "evidence").glob("**/code-review.md"))
    if not review_files:
        return "NEEDS_INPUT", ["independent code review is missing"]
    reasons: list[str] = []
    for path in review_files:
        status = markdown_field(path, "Status").upper()
        reviewer = markdown_field(path, "Reviewer").lower()
        reviewer_actor_id = markdown_field(path, "Reviewer actor_id")
        reviewed_commit = markdown_field(path, "Reviewed commit")
        if status != "PASS":
            reasons.append(f"independent code review is {status or 'UNKNOWN'}: {path.relative_to(root)}")
        if not reviewer or reviewer in {"unassigned", "unknown", "pending"}:
            reasons.append(f"code review has no independent reviewer identity: {path.relative_to(root)}")
        if status == "PASS" and not reviewer_actor_id:
            reasons.append(f"code review has no reviewer actor_id: {path.relative_to(root)}")
        if status == "PASS" and not reviewed_commit:
            reasons.append(f"code review has no reviewed commit: {path.relative_to(root)}")
    return ("READY" if not reasons else "NEEDS_INPUT"), reasons


def decision_matches_context_snapshot(
    decision: dict[str, Any], document_by_path: dict[str, dict[str, Any]], root: Path
) -> bool:
    expected = str(decision.get("source_snapshot_hash", ""))
    context_document = document_by_path.get("docs/ai/context-pack.json")
    if not expected or not context_document or context_document.get("status") != "DERIVED":
        return False
    pack = load_json(root / "docs/ai/context-pack.json") or {}
    return pack.get("source_snapshot_hash") == expected


def evidence_state(root: Path) -> tuple[str, list[str]]:
    evidence_root = root / "evidence"
    if not evidence_root.is_dir():
        return "NEEDS_INPUT", ["Evidence Bundle is missing"]
    reasons: list[str] = []
    review_status, review_reasons = review_evidence_state(root)
    if review_status != "READY":
        reasons.extend(review_reasons)
    receipts = list(evidence_root.glob("**/*receipt*.json"))
    if not receipts:
        reasons.append("release receipt is missing")
    else:
        manifest = yaml.safe_load((root / "project.yaml").read_text(encoding="utf-8"))
        staging = next(
            (
                environment
                for environment in manifest.get("spec", {}).get("environments", [])
                if environment.get("id") == "staging"
            ),
            {},
        )
        expected_target = staging.get("targetRef")
        matching_pass = any(
            (receipt := (load_json(path) or {})).get("status") == "PASS"
            and receipt.get("environment") == "staging"
            and receipt.get("target") == expected_target
            for path in receipts
        )
        if not matching_pass:
            reasons.append("no PASS staging receipt matches the manifest target")
    return ("READY" if not reasons else "NEEDS_INPUT"), reasons


def cloud_testing_evidence_state(root: Path) -> tuple[str, list[str]]:
    """Check testing/CI evidence without requiring a Cloud Run deployment."""
    evidence_root = root / "evidence"
    if not evidence_root.is_dir():
        return "NEEDS_INPUT", ["Evidence Bundle is missing"]
    reasons: list[str] = []
    test_files = list(evidence_root.glob("**/test-output.txt"))
    if not test_files or any("Result: PASS" not in path.read_text(encoding="utf-8") for path in test_files):
        reasons.append("test evidence is not PASS")
    build_files = list(evidence_root.glob("**/build-output.txt"))
    if not build_files or any("Result: PASS" not in path.read_text(encoding="utf-8") for path in build_files):
        reasons.append("build evidence is not PASS")
    review_status, review_reasons = review_evidence_state(root)
    if review_status != "READY":
        reasons.extend(review_reasons)
    return ("READY" if not reasons else "NEEDS_INPUT"), reasons


def inspect_project(root: Path) -> dict[str, Any]:
    root = root.resolve()
    manifest_path = root / "project.yaml"
    manifest = yaml.safe_load(manifest_path.read_text(encoding="utf-8"))
    metadata = manifest.get("metadata", {})
    spec = manifest.get("spec", {})
    documents = [document_status(root, item) for item in spec.get("documents", [])]
    document_by_path = {item["path"]: item for item in documents}
    decision_docs = [item for item in spec.get("documents", []) if "decision" in item.get("requiredFor", [])]
    decision_blockers = [document_by_path[item["path"]]["status"] for item in decision_docs]
    decisions = find_decisions(root)
    accepted_decisions = [
        decision for decision in decisions if decision_is_human_accepted(decision, "accept_request")
    ]
    human_accepted = bool(accepted_decisions)
    staging_accepted = any(decision_is_human_accepted(decision, "allow_staging") for decision in decisions)
    has_accepted_status = any(
        decision.get("status") in {"ACCEPTED", "ACCEPTED_FOR_DEVELOPMENT", "ACCEPTED_FOR_STAGING"}
        for decision in decisions
    )
    decision_reasons: list[str] = []
    if any(status in {"MISSING", "STALE", "CONFLICT", "NEEDS_INPUT"} for status in decision_blockers):
        decision_reasons.append("required decision documents are not all CURRENT")
    decision_ready = "READY" if not decision_reasons else "NEEDS_INPUT"
    local_development_reasons: list[str] = []
    if not human_accepted:
        local_development_reasons.append("no human-accepted DecisionRecord found")
    if has_accepted_status and not human_accepted:
        local_development_reasons.append("DecisionRecord status conflicts with pending human decision")
    development_docs = [item for item in documents if "development" in item.get("requiredFor", [])]
    if any(item["status"] != "CURRENT" for item in development_docs):
        local_development_reasons.append("required development documents are not all CURRENT")
    if accepted_decisions and not any(
        decision_matches_context_snapshot(decision, document_by_path, root) for decision in accepted_decisions
    ):
        local_development_reasons.append("accepted DecisionRecord does not match the current Context Pack snapshot")
    local_development_ready = "READY" if not local_development_reasons else "NEEDS_INPUT"
    cloud_testing_evidence, cloud_testing_reasons = cloud_testing_evidence_state(root)
    cloud_testing_reasons = list(cloud_testing_reasons)
    if local_development_ready != "READY":
        cloud_testing_reasons.insert(0, "local development readiness is not READY")
    cloud_testing_ready = "READY" if cloud_testing_evidence == "READY" and not cloud_testing_reasons else "NEEDS_INPUT"
    staging_evidence, staging_evidence_reasons = evidence_state(root)
    staging_reasons: list[str] = []
    staging_environment = next(
        (environment for environment in spec.get("environments", []) if environment.get("id") == "staging"),
        {},
    )
    if not staging_environment.get("targetRef") or "<REPLACE_ME" in str(staging_environment.get("targetRef")):
        staging_reasons.append("staging target is not configured")
    if cloud_testing_ready != "READY":
        staging_reasons.append("cloud testing readiness is not READY")
    if not staging_accepted:
        staging_reasons.append("no human-approved staging DecisionRecord found")
    staging_ready = "READY" if not staging_reasons else "NEEDS_INPUT"
    staging_verified_reasons = list(staging_evidence_reasons)
    if staging_ready != "READY":
        staging_verified_reasons.insert(0, "staging deployment readiness is not READY")
    staging_verified = "READY" if staging_evidence == "READY" and not staging_verified_reasons else "NEEDS_INPUT"
    production = "BLOCKED"
    return {
        "read_only": True,
        "observed_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
        "project": {
            "id": metadata.get("id"),
            "name": metadata.get("name"),
            "version": metadata.get("version"),
            "root": str(root),
        },
        "documents": documents,
        "decisions": [
            {"decision_id": item.get("decision_id"), "request_id": item.get("request_id"), "status": item.get("status")}
            for item in decisions
        ],
        "readiness": {
            "ready_for_decision": {"status": decision_ready, "reasons": decision_reasons},
            "ready_for_local_development": {
                "status": local_development_ready,
                "reasons": local_development_reasons,
            },
            "ready_for_cloud_testing": {"status": cloud_testing_ready, "reasons": cloud_testing_reasons},
            "ready_for_staging": {"status": staging_ready, "reasons": staging_reasons},
            "staging_verified": {"status": staging_verified, "reasons": staging_verified_reasons},
            "production": {"status": production, "reasons": ["production is human-gated and read-only in P0"]},
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", action="append", required=True, type=Path, help="Project root; repeat for multiple Projects")
    parser.add_argument("--output", choices=("json", "summary"), default="summary")
    args = parser.parse_args()
    reports = [inspect_project(root) for root in args.root]
    if args.output == "json":
        print(json.dumps(reports, indent=2, ensure_ascii=False))
        return 0
    for report in reports:
        project = report["project"]
        print(f"{project['id']} — {project['name']}")
        for name, value in report["readiness"].items():
            print(f"  {name}: {value['status']}")
            for reason in value["reasons"]:
                print(f"    - {reason}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
