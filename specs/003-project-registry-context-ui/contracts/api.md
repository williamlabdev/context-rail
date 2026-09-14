# API Contract: Project Registry／Documents & AI Context

## `GET /v1/projects`

Returns the explicitly configured Project registry entries needed for selection.

### Success

```json
{
  "kind": "ProjectRegistrySnapshot",
  "schema_version": "project-registry/v1",
  "projects": [
    {
      "project": {"id": "order-operations-portal", "name": "Order Operations Portal"},
      "readiness": {"status": "READY_FOR_DECISION"},
      "context": {"status": "DERIVED"},
      "read_only": true,
      "observed_at": "<timestamp>"
    }
  ]
}
```

List entries may contain summary fields only, but identity and status must be stable. The service must not scan arbitrary directories or URLs.

## `GET /v1/projects/{project_id}`

Returns the complete normalized ProjectRecord used by the detail view, including repositories, services, environments, documents, decisions, readiness, context status, source root and observation time.

### Read-only headers and semantics

- No `POST`, `PUT`, `PATCH` or `DELETE` route is part of this slice.
- A successful response includes `read_only: true`.
- The response preserves `STALE`, `MISSING`, `CONFLICT`, `UNKNOWN` and `UNDECLARED`.
- `CURRENT` or `READY_FOR_*` is observed/derived status only, not approval or authorization.

## Errors

Errors are JSON and must identify the requested project/input without exposing secrets or local credential values.

```json
{
  "kind": "ProjectRegistryError",
  "code": "PROJECT_NOT_FOUND",
  "message": "The requested Project is not configured for this local registry.",
  "read_only": true
}
```

Required codes for this slice: `PROJECT_NOT_FOUND`, `PROJECT_INVALID`, `PROJECT_UNAVAILABLE` and `REGISTRY_UNAVAILABLE`.
