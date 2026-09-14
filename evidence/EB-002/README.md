# EB-002 — Project Registry import baseline

- Checked at: 2026-09-14 Asia/Taipei
- Current source commit: `cfbf0a8d1fa3859f3085cd53ce827257367f43c6`
- Scope: local read-only VS-001 implementation handoff baseline
- Cloud Run, IAM and production: not used

## Explicit fixture roots

- `demo/order-operations-portal`
- `examples/support-insights`

## Contract oracle and existing reference

- Python oracle: `scripts/context_rail_registry.py`
- Existing validation reference: `docs/validation/CONTEXT_RAIL_PROJECT_REGISTRY_IMPORT_2026-09-14.zh-TW.md`
- Decision boundary: `decisions/DR-002-project-registry-import.json`

## Local tool versions observed

```text
go version go1.26.6 darwin/arm64
Node.js v25.8.1
npm 11.11.0
Python 3.9.6 (template/.venv/bin/python)
```

## Evidence index

- `test-output.txt` — Go contract and mutation tests
- `vet-output.txt` — `go vet ./...`
- `build-output.txt` — `go build ./...`
- `oracle-comparison.txt` — Go/Python normalized snapshot comparison
- `mutation-check.txt` — before/after hashes for both consumer fixtures

Implementation commit: `38c965f09e0b3001ec8b1a699a7e42d854c406a3`.

The Go snapshot matches the Python oracle for both fixtures after removing the runtime-generated `observed_at`. The contract suite also compares complete fixture tree hashes before and after import.
