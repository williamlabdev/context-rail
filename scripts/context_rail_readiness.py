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


def decision_is_human_accepted(decision: dict[str, Any]) -> bool:
    if decision.get("status") not in {"ACCEPTED", "ACCEPTED_FOR_DEVELOPMENT", "ACCEPTED_FOR_STAGING"}:
        return False
    human = decision.get("human_decision") or decision.get("human_decisions") or {}
    values = json.dumps(human).upper()
    return not any(marker in values for marker in ("PENDING", "UNKNOWN", "BLOCKED"))


def evidence_state(root: Path) -> tuple[str, list[str]]:
    evidence_root = root / "evidence"
    if not evidence_root.is_dir():
        return "NEEDS_INPUT", ["Evidence Bundle is missing"]
    reasons: list[str] = []
    review_files = list(evidence_root.glob("**/code-review.md"))
    if not review_files or any("REVIEW_REQUIRED" in path.read_text(encoding="utf-8") for path in review_files):
        reasons.append("independent code review is not PASS")
    receipts = list(evidence_root.glob("**/*receipt*.json"))
    if not receipts:
        reasons.append("release receipt is missing")
    else:
        for path in receipts:
            receipt = load_json(path) or {}
            if receipt.get("status") != "PASS":
                reasons.append(f"release receipt is {receipt.get('status', 'UNKNOWN')}: {path.relative_to(root)}")
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
    review_files = list(evidence_root.glob("**/code-review.md"))
    if not review_files or any("REVIEW_REQUIRED" in path.read_text(encoding="utf-8") for path in review_files):
        reasons.append("independent code review is not PASS")
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
    human_accepted = any(decision_is_human_accepted(decision) for decision in decisions)
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
    local_development_ready = "READY" if not local_development_reasons else "NEEDS_INPUT"
    cloud_testing_evidence, cloud_testing_reasons = cloud_testing_evidence_state(root)
    cloud_testing_reasons = list(cloud_testing_reasons)
    if local_development_ready != "READY":
        cloud_testing_reasons.insert(0, "local development readiness is not READY")
    cloud_testing_ready = "READY" if cloud_testing_evidence == "READY" and not cloud_testing_reasons else "NEEDS_INPUT"
    staging_evidence, staging_reasons = evidence_state(root)
    staging_reasons = list(staging_reasons)
    if local_development_ready != "READY":
        staging_reasons.insert(0, "local development readiness is not READY")
    if cloud_testing_ready != "READY":
        staging_reasons.insert(0, "cloud testing readiness is not READY")
    staging_ready = "READY" if staging_evidence == "READY" and not staging_reasons else "NEEDS_INPUT"
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
