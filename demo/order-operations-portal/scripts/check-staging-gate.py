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
    except (OSError, json.JSONDecodeError) as exc:
        return fail(f"required governance artifact cannot be read: {exc}")

    if decision.get("status") != "ACCEPTED_FOR_STAGING":
        return fail("DecisionRecord status is not ACCEPTED_FOR_STAGING")
    if str(decision.get("human_decisions", {}).get("allow_staging", "")).upper() not in {
        "ACCEPTED",
        "APPROVED",
        "TRUE",
    }:
        return fail("human staging approval is not recorded")
    if context_pack.get("derived") is not True or context_pack.get("status") != "CURRENT":
        return fail("Context Pack is not CURRENT")
    if context_pack.get("source_snapshot_hash") != decision.get("source_snapshot_hash"):
        return fail("DecisionRecord and Context Pack snapshots do not match")
    if field(review_text, "Status").upper() != "PASS":
        return fail("independent code review is not PASS")
    reviewer = field(review_text, "Reviewer").lower()
    if not reviewer or reviewer in {"unassigned", "unknown", "pending"}:
        return fail("independent reviewer identity is missing")
    if field(review_text, "Reviewed commit") != approved_commit:
        return fail("reviewed commit does not match APPROVED_COMMIT")
    print("STAGING GATE PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
