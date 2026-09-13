#!/usr/bin/env python3

"""Rebuild the derived Context Pack from declared source documents."""

from __future__ import annotations

import argparse
import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path

import yaml


def utc_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def sha256(content: bytes) -> str:
    return hashlib.sha256(content).hexdigest()


def declared_sources(manifest: dict) -> list[tuple[str, str]]:
    documents = manifest.get("spec", {}).get("documents", [])
    return [
        (str(document["path"]), str(document["kind"]))
        for document in documents
        if document.get("sourceOfTruth", True) is not False
    ]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, help="Project root to rebuild; defaults to this template root")
    args = parser.parse_args()
    root = args.root.resolve() if args.root else Path(__file__).resolve().parents[1]
    manifest = yaml.safe_load((root / "project.yaml").read_text(encoding="utf-8"))
    sources = declared_sources(manifest)
    generated_at = utc_now()
    has_placeholders = any(
        "<REPLACE_ME" in (root / relative).read_text(encoding="utf-8")
        for relative, _kind in sources
    )
    documents = []
    for relative, kind in sources:
        content = (root / relative).read_bytes()
        documents.append(
            {
                "path": relative,
                "kind": kind,
                "version": str(manifest["metadata"]["version"]),
                "content_hash": f"sha256:{sha256(content)}",
                "effective_time": generated_at,
                "access_scope": "repository-read",
            }
        )
    snapshot_input = "\n".join(f"{doc['path']}={doc['content_hash']}" for doc in documents).encode()
    limitations = [
        "Generated from declared repository sources; external Git, CI and runtime evidence is not included"
    ]
    if has_placeholders:
        limitations.append(
            "Template placeholders remain; complete the source documents before using this pack for a decision"
        )
    pack = {
        "kind": "ContextPack",
        "derived": True,
        "status": "NEEDS_INPUT" if has_placeholders else "CURRENT",
        "project_id": manifest["metadata"]["id"],
        "schema_version": "context-pack/v1",
        "generated_at": generated_at,
        "source_snapshot_hash": f"sha256:{sha256(snapshot_input)}",
        "sources": documents,
        "decision_refs": [],
        "limitations": limitations,
    }
    output = root / "docs/ai/context-pack.json"
    output.write_text(json.dumps(pack, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"REBUILT {output}")
    print(f"SOURCE SNAPSHOT {pack['source_snapshot_hash']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
