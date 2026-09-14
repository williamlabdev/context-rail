# UI Contract: Project Registry／Documents & AI Context

## Views

### Registry view

- Shows the explicitly configured Projects.
- Shows stable identity and a non-authoritative summary status.
- Selecting a Project is the only primary action.
- No create, edit, archive, upload, approve or deploy control appears in this slice.

### Project context view

- Shows Project identity and source root/reference.
- Shows repositories, services and environments as declared relationships.
- Shows Documents / AI Context with source-versus-derived distinction.
- Shows readiness status and reason.
- Shows `STALE`, `MISSING`, `CONFLICT`, `UNKNOWN` and `UNDECLARED` as non-success states.

## State contract

| State | Required user-visible behavior |
| --- | --- |
| `LOADING` | Show loading indicator; do not show old detail data as current. |
| `READY` | Show the selected Project and its observed statuses. |
| `EMPTY` | Explain that no Projects are configured; no create action. |
| `ERROR` | Explain the unavailable/invalid input; no mutation action. |
| `STALE`/`MISSING`/`CONFLICT`/`UNKNOWN`/`UNDECLARED` | Preserve exact state and show reason/next information needed when available. |

## Interaction invariants

- Switching Project replaces all detail sections atomically from the selected record.
- Refreshing is read-only and does not rebuild Context Pack or create audit events.
- UI labels must not say `approved`, `authorized`, `deployable` or `production ready` solely because a registry/readiness value exists.
- The UI must not display secret or credential fields.
