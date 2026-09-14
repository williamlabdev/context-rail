# Data Model: Project Registry／Documents & AI Context 唯讀工作區

## ProjectRegistrySnapshot

The top-level read result consumed by the UI.

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `kind` | string | yes | Must be `ProjectRegistrySnapshot`. |
| `schema_version` | string | yes | Must be `project-registry/v1` for this slice. |
| `projects` | array of ProjectRecord | yes | May be empty; entries are independently identified. |

## ProjectRecord

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `project.id` | string | yes | Stable Project identity; never derive from display name. |
| `project.name` | string | yes | Human-readable label. |
| `project.status` | string | no | Preserve declared status or `UNDECLARED`. |
| `project.classification` | string | no | Preserve declared classification. |
| `project.owners` | object | no | Display values only; no membership decision. |
| `project.root` | string | yes | Observed source root or safe display reference. |
| `repositories` | array | yes | Declared repository relationships only. |
| `services` | array | yes | Normalized from supported singular/plural source declarations. |
| `environments` | array | yes | Declared environment records only. |
| `documents` | array | yes | Source/derived document status and provenance. |
| `decisions` | array | yes | Observed decision records; no approval inference. |
| `readiness` | object/array | yes | Derived readiness with reason and evidence state. |
| `context.status` | string | yes | Preserve `CURRENT`, `PARTIAL`, `STALE`, `MISSING` or `UNKNOWN`. |
| `read_only` | boolean | yes | Must be `true` for this slice. |
| `observed_at` | timestamp | yes | Observation time for the record. |

## DocumentStatus

Allowed displayed states are `CURRENT`, `DRAFT`, `MISSING`, `STALE`, `CONFLICT` and `REVOKED`. A derived Context Pack must be visually distinguishable from its source documents and must retain source path/version/hash when available.

## UncertaintyStatus

`UNKNOWN` and `UNDECLARED` are valid values for fields where the source or observation does not provide a fact. They are not errors by themselves and must not be mapped to `CURRENT`, `PASS`, approval or authorization.

## WorkspaceState

The UI has four top-level states:

```text
LOADING → READY
       ├→ EMPTY
       └→ ERROR
```

`READY` may contain incomplete or uncertain Project records. It must not imply governance readiness. `ERROR` contains a safe user-facing message and no mutation action.

## Relationships

- One snapshot contains zero or more ProjectRecords.
- One ProjectRecord contains zero or more declared repositories, services, environments, documents and decisions.
- A ProjectDocument may be a source document or derived Context Pack entry, but derived status never upgrades source status.
- A selected ProjectRecord controls the detail view; switching selection replaces the complete detail state.
