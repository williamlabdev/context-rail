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

This baseline records the accepted development boundary. Contract tests, implementation output, build output, oracle comparison and zero-mutation evidence are added by the remaining tasks.
