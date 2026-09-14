# ContextRail Project Registry Read-only Import

- Date: 2026-09-14 Asia/Taipei
- Commit: `6e523b9`
- Status: `PASS`
- Scope: read-only Project manifest import and readiness projection
- Cloud Run: not used

## Implemented slice

`scripts/context_rail_registry.py` reads one or more Project roots and emits a normalized `ProjectRegistrySnapshot` containing:

- Project identity, purpose, owners and lifecycle declaration;
- repository and service relationships;
- ordered environments and required evidence;
- declared document status and Context Pack status;
- DecisionRecord references and readiness states.

The importer does not copy, modify, rebuild or silently assign any source. Missing or stale information remains visible.

## Verification

```sh
template/.venv/bin/python -m unittest discover -s tests -v
template/.venv/bin/python -m unittest discover -s template/tests -v
template/.venv/bin/python scripts/context_rail_registry.py \
  --root demo/order-operations-portal \
  --root examples/support-insights
```

Observed:

```text
Ran 15 tests ... OK
Ran 8 tests ... OK
order-operations-portal: context DERIVED
support-insights: context STALE
```

The registry correctly preserves the Support Insights `STALE` Context Pack caused by source drift. It does not rebuild that Project as a side effect.

## Boundary

This is a read-only contract/import adapter, not yet the Go HTTP API, Firestore persistence, authentication layer or Project CRUD implementation. The next core slice should expose this normalized snapshot through a Go modular-monolith API while preserving the same read-only and evidence-bound semantics.
