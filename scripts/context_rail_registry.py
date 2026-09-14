#!/usr/bin/env python3

"""Read-only Project Registry import for ContextRail contract fixtures."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

import yaml

from context_rail_readiness import inspect_project


def load_manifest(root: Path) -> dict[str, Any]:
    path = root / "project.yaml"
    try:
        value = yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError) as exc:
        raise ValueError(f"cannot read project manifest: {exc}") from exc
    if not isinstance(value, dict) or value.get("kind") != "Project":
        raise ValueError("project.yaml kind must be Project")
    return value


def repositories(spec: dict[str, Any]) -> list[dict[str, Any]]:
    value = spec.get("repository")
    if not isinstance(value, dict):
        return []
    return [
        {
            "provider": value.get("provider"),
            "visibility": value.get("visibility"),
            "url": value.get("url"),
            "default_branch": value.get("defaultBranch"),
            "status": value.get("status", "UNDECLARED"),
        }
    ]


def services(spec: dict[str, Any]) -> list[dict[str, Any]]:
    values = spec.get("services")
    if not isinstance(values, list):
        value = spec.get("service")
        values = [value] if isinstance(value, dict) else []
    return [
        {
            "id": value.get("id"),
            "repository": value.get("repository"),
            "path": value.get("path", value.get("sourcePath")),
            "runtime": value.get("runtime", "UNDECLARED"),
            "role": value.get("role", "primary"),
        }
        for value in values
        if isinstance(value, dict)
    ]


def environments(spec: dict[str, Any]) -> list[dict[str, Any]]:
    values = spec.get("environments", [])
    if not isinstance(values, list):
        return []
    return [
        {
            "id": value.get("id"),
            "type": value.get("type"),
            "sequence": value.get("sequence"),
            "target_ref": value.get("targetRef"),
            "required_evidence": value.get("requiredEvidence", []),
            "action": value.get("action"),
        }
        for value in values
        if isinstance(value, dict)
    ]


def import_project(root: Path) -> dict[str, Any]:
    root = root.resolve()
    manifest = load_manifest(root)
    metadata = manifest.get("metadata", {})
    spec = manifest.get("spec", {})
    if not isinstance(metadata, dict) or not isinstance(spec, dict):
        raise ValueError("project.yaml metadata and spec must be objects")
    readiness = inspect_project(root)
    return {
        "project": {
            "id": metadata.get("id"),
            "name": metadata.get("name"),
            "version": metadata.get("version"),
            "classification": metadata.get("classification"),
            "status": metadata.get("status", "UNDECLARED"),
            "purpose": spec.get("purpose"),
            "owners": spec.get("owners", {}),
            "root": str(root),
        },
        "repositories": repositories(spec),
        "services": services(spec),
        "environments": environments(spec),
        "documents": readiness["documents"],
        "decisions": readiness["decisions"],
        "readiness": readiness["readiness"],
        "context": {
            "status": next(
                (
                    document["status"]
                    for document in readiness["documents"]
                    if document["path"] == "docs/ai/context-pack.json"
                ),
                "MISSING",
            ),
        },
        "read_only": True,
        "observed_at": readiness["observed_at"],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", action="append", required=True, type=Path, help="Project root; repeat for multiple Projects")
    args = parser.parse_args()
    try:
        projects = [import_project(root) for root in args.root]
    except ValueError as exc:
        print(f"PROJECT REGISTRY IMPORT BLOCKED: {exc}", file=sys.stderr)
        return 2
    print(
        json.dumps(
            {"kind": "ProjectRegistrySnapshot", "schema_version": "project-registry/v1", "projects": projects},
            indent=2,
            ensure_ascii=False,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
