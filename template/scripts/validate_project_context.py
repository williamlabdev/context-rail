#!/usr/bin/env python3

"""Validate the required Project Context Contract files."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import yaml


REQUIRED = (
    "project.yaml",
    "README.md",
    "vision.md",
    "architecture.md",
    "AGENTS.md",
    "docs/engineering/development.md",
    "docs/operations/environments.md",
    "docs/governance/policies.md",
    "docs/ai/context-pack.json",
)


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def validate(root: Path, strict: bool) -> list[str]:
    errors: list[str] = []
    for relative in REQUIRED:
        if not (root / relative).is_file():
            errors.append(f"MISSING {relative}")

    try:
        manifest = yaml.safe_load((root / "project.yaml").read_text(encoding="utf-8"))
        if not isinstance(manifest, dict) or manifest.get("kind") != "Project":
            errors.append("project.yaml kind must be Project")
        metadata = manifest.get("metadata", {}) if isinstance(manifest, dict) else {}
        if not isinstance(metadata, dict) or not str(metadata.get("id", "")):
            errors.append("project.yaml metadata.id is required")
        if not isinstance(manifest, dict) or not isinstance(manifest.get("spec"), dict):
            errors.append("project.yaml spec is required")
        else:
            spec = manifest["spec"]
            environments = spec.get("environments", [])
            if not isinstance(environments, list) or not environments:
                errors.append("project.yaml spec.environments must be a non-empty list")
            else:
                sequences = [environment.get("sequence") for environment in environments if isinstance(environment, dict)]
                if len(sequences) != len(set(sequences)):
                    errors.append("project.yaml environment sequence values must be unique")
                for environment in environments:
                    if not isinstance(environment, dict):
                        errors.append("project.yaml environments must contain objects")
                        continue
                    for field in ("id", "type", "sequence", "targetRef", "requiredEvidence"):
                        if field not in environment:
                            errors.append(f"project.yaml environment is missing {field}")
            declared_documents = spec.get("documents", [])
            if not isinstance(declared_documents, list):
                errors.append("project.yaml spec.documents must be a list")
            else:
                seen_paths: set[str] = set()
                for document in declared_documents:
                    if not isinstance(document, dict):
                        errors.append("project.yaml documents must contain objects")
                        continue
                    path = document.get("path")
                    if not path:
                        errors.append("project.yaml document path is required")
                        continue
                    if path in seen_paths:
                        errors.append(f"project.yaml document path is duplicated: {path}")
                    seen_paths.add(path)
                    if not (root / path).is_file():
                        errors.append(f"DECLARED DOCUMENT MISSING {path}")
                    if document.get("sourceOfTruth") is False and document.get("kind") != "derived-context":
                        errors.append(f"only derived-context documents may set sourceOfTruth=false: {path}")
    except Exception as exc:  # pragma: no cover - exact parser message is environment-specific.
        errors.append(f"project.yaml is not valid YAML: {exc}")

    try:
        context_pack = json.loads((root / "docs/ai/context-pack.json").read_text(encoding="utf-8"))
        if context_pack.get("derived") is not True:
            errors.append("context pack must be marked derived")
        if not str(context_pack.get("status", "")):
            errors.append("context pack must declare a status")
        source_paths = context_pack.get("sources", [])
        if not isinstance(source_paths, list):
            errors.append("context pack sources must be a list")
        else:
            for source in source_paths:
                if not isinstance(source, dict) or not source.get("path"):
                    errors.append("context pack source must declare a path")
                    continue
                source_path = root / source["path"]
                if not source_path.is_file():
                    errors.append(f"CONTEXT SOURCE MISSING {source['path']}")
                    continue
                expected_hash = f"sha256:{sha256(source_path)}"
                if source.get("content_hash") != expected_hash:
                    errors.append(f"CONTEXT SOURCE HASH MISMATCH {source['path']}")
                if not str(source.get("content_hash", "")).startswith("sha256:"):
                    errors.append(f"context source hash missing: {source['path']}")
            if isinstance(manifest, dict) and isinstance(manifest.get("spec"), dict):
                declared_paths = [
                    document.get("path")
                    for document in manifest["spec"].get("documents", [])
                    if isinstance(document, dict) and document.get("sourceOfTruth", True) is not False
                ]
                actual_paths = [
                    source.get("path") for source in source_paths if isinstance(source, dict)
                ]
                if actual_paths != declared_paths:
                    errors.append(
                        "context pack sources must exactly match declared source-of-truth documents"
                    )
                declared_kinds = {
                    document.get("path"): document.get("kind")
                    for document in manifest["spec"].get("documents", [])
                    if isinstance(document, dict)
                }
                for source in source_paths:
                    if isinstance(source, dict) and source.get("path") in declared_kinds:
                        if source.get("kind") != declared_kinds[source["path"]]:
                            errors.append(f"CONTEXT SOURCE KIND MISMATCH {source['path']}")
                snapshot_input = "\n".join(
                    f"{source.get('path')}={source.get('content_hash')}"
                    for source in source_paths
                    if isinstance(source, dict)
                ).encode()
                expected_snapshot = f"sha256:{hashlib.sha256(snapshot_input).hexdigest()}"
                if context_pack.get("source_snapshot_hash") != expected_snapshot:
                    errors.append("context pack source snapshot hash mismatch")
    except Exception as exc:  # pragma: no cover - exact parser message is environment-specific.
        errors.append(f"context-pack.json is not valid JSON: {exc}")

    if strict:
        for relative in REQUIRED:
            path = root / relative
            if path.is_file() and "<REPLACE_ME" in path.read_text(encoding="utf-8"):
                errors.append(f"PLACEHOLDER remains in {relative}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--strict", action="store_true", help="fail when template placeholders remain")
    parser.add_argument("--root", type=Path, help="Project root to validate; defaults to this template root")
    args = parser.parse_args()
    root = args.root.resolve() if args.root else Path(__file__).resolve().parents[1]
    errors = validate(root, args.strict)
    if errors:
        for error in errors:
            print(f"FAIL: {error}")
        return 1
    mode = "STRICT " if args.strict else ""
    print(f"PROJECT CONTEXT {mode}PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
